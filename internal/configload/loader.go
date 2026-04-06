package configload

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/alexmaze/clink/internal/domain"
	"github.com/alexmaze/clink/lib/fileutil"
	"gopkg.in/yaml.v3"
)

var patternVar = regexp.MustCompile(`\${[0-9A-Za-z_-]+}`)

func Load(path string, filters []string) (*domain.Config, error) {
	return load(path, filters, true)
}

func LoadForRestore(path string, filters []string) (*domain.Config, error) {
	return load(path, filters, false)
}

func load(path string, filters []string, requireSources bool) (*domain.Config, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve config path: %w", err)
	}

	raw, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg domain.Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	cfg.ConfigPath = absPath
	cfg.WorkDir = filepath.Dir(absPath)
	cfg.SelectedRaw = slices.Clone(filters)
	if cfg.SSHServers == nil {
		cfg.SSHServers = map[string]domain.SSHServer{}
	}
	if cfg.Vars == nil {
		cfg.Vars = map[string]string{}
	}

	if err := normalize(&cfg); err != nil {
		return nil, err
	}
	if len(filters) > 0 {
		filtered, err := filterRules(cfg.Rules, filters)
		if err != nil {
			return nil, err
		}
		cfg.Rules = filtered
	}

	return &cfg, validate(&cfg, requireSources)
}

func Save(cfg *domain.Config) error {
	out := struct {
		Mode       domain.Mode                 `yaml:"mode,omitempty"`
		Hooks      *domain.Hooks               `yaml:"hooks,omitempty"`
		SSHServers map[string]domain.SSHServer `yaml:"ssh_servers,omitempty"`
		Vars       map[string]string           `yaml:"vars,omitempty"`
		Rules      []domain.Rule               `yaml:"rules"`
	}{
		Mode:       cfg.Mode,
		Hooks:      cfg.Hooks,
		SSHServers: make(map[string]domain.SSHServer, len(cfg.SSHServers)),
		Vars:       cfg.Vars,
		Rules:      make([]domain.Rule, 0, len(cfg.Rules)),
	}

	for name, server := range cfg.SSHServers {
		if rel, ok := tryRelative(cfg.WorkDir, server.Key); ok {
			server.Key = rel
		}
		out.SSHServers[name] = server
	}
	for _, rule := range cfg.Rules {
		cloned := rule
		cloned.Items = make([]domain.RuleItem, 0, len(rule.Items))
		for _, item := range rule.Items {
			if rel, ok := tryRelative(cfg.WorkDir, item.Source); ok {
				item.Source = rel
			}
			cloned.Items = append(cloned.Items, item)
		}
		out.Rules = append(out.Rules, cloned)
	}

	body, err := yaml.Marshal(out)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	tmpPath := cfg.ConfigPath + ".tmp"
	if err := os.WriteFile(tmpPath, body, 0644); err != nil {
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := os.Rename(tmpPath, cfg.ConfigPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}

func normalize(cfg *domain.Config) error {
	defaultMode := cfg.Mode
	if defaultMode == "" {
		defaultMode = domain.ModeSymlink
	}

	for name, server := range cfg.SSHServers {
		if server.Port == 0 {
			server.Port = 22
		}
		if server.Key != "" {
			keyPath, err := fileutil.ParsePath(cfg.WorkDir, server.Key)
			if err != nil {
				return fmt.Errorf("resolve ssh key for %s: %w", name, err)
			}
			server.Key = keyPath
		}
		cfg.SSHServers[name] = server
	}

	for i := range cfg.Rules {
		rule := &cfg.Rules[i]
		if rule.Mode == "" {
			rule.Mode = defaultMode
		}
		for j := range rule.Items {
			item := &rule.Items[j]
			renderedSource, err := renderVars(cfg.Vars, item.Source)
			if err != nil {
				return fmt.Errorf("render source for rule %s: %w", rule.Name, err)
			}
			renderedDestination, err := renderVars(cfg.Vars, item.Destination)
			if err != nil {
				return fmt.Errorf("render destination for rule %s: %w", rule.Name, err)
			}
			item.Source = renderedSource
			item.Destination = renderedDestination

			src, err := fileutil.ParsePath(cfg.WorkDir, item.Source)
			if err != nil {
				return fmt.Errorf("resolve source path for rule %s: %w", rule.Name, err)
			}
			item.Source = src

			if rule.Mode != domain.ModeSSH {
				dest, err := fileutil.ParsePath(cfg.WorkDir, item.Destination)
				if err != nil {
					return fmt.Errorf("resolve destination path for rule %s: %w", rule.Name, err)
				}
				item.Destination = dest
			}
		}
	}
	return nil
}

func validate(cfg *domain.Config, requireSources bool) error {
	ruleNames := map[string]struct{}{}
	destinations := map[string]struct{}{}

	for _, rule := range cfg.Rules {
		if strings.TrimSpace(rule.Name) == "" {
			return fmt.Errorf("rule name cannot be empty")
		}
		key := strings.ToLower(rule.Name)
		if _, ok := ruleNames[key]; ok {
			return fmt.Errorf("duplicate rule name: %s", rule.Name)
		}
		ruleNames[key] = struct{}{}

		switch rule.Mode {
		case domain.ModeSymlink, domain.ModeCopy, domain.ModeSSH:
		default:
			return fmt.Errorf("rule %s uses unsupported mode %q", rule.Name, rule.Mode)
		}

		if rule.Mode == domain.ModeSSH {
			if rule.SSH == "" {
				return fmt.Errorf("rule %s must declare ssh server", rule.Name)
			}
			if _, ok := cfg.SSHServers[rule.SSH]; !ok {
				return fmt.Errorf("rule %s references unknown ssh server %q", rule.Name, rule.SSH)
			}
		}

		if len(rule.Items) == 0 {
			return fmt.Errorf("rule %s must define at least one item", rule.Name)
		}

		for _, item := range rule.Items {
			if item.Source == "" || item.Destination == "" {
				return fmt.Errorf("rule %s contains empty src/dest", rule.Name)
			}
			if requireSources {
				exists, err := fileutil.IsFileExists(item.Source)
				if err != nil {
					return fmt.Errorf("inspect source for rule %s: %w", rule.Name, err)
				}
				if !exists {
					return fmt.Errorf("source does not exist for rule %s: %s", rule.Name, item.Source)
				}
			}
			itemKey := item.Destination
			if _, ok := destinations[itemKey]; ok {
				return fmt.Errorf("duplicate destination path: %s", item.Destination)
			}
			destinations[itemKey] = struct{}{}
		}
	}
	return nil
}

func tryRelative(baseDir, path string) (string, bool) {
	if path == "" || !filepath.IsAbs(path) {
		return "", false
	}
	rel, err := filepath.Rel(baseDir, path)
	if err != nil {
		return "", false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	if strings.HasPrefix(rel, "./") {
		return rel, true
	}
	return "./" + rel, true
}

func renderVars(vars map[string]string, input string) (string, error) {
	matches := patternVar.FindAllString(input, -1)
	if len(matches) == 0 {
		return input, nil
	}

	result := input
	for _, token := range matches {
		key := strings.ToLower(strings.TrimSuffix(strings.TrimPrefix(token, "${"), "}"))
		val, ok := vars[key]
		if !ok {
			return "", fmt.Errorf("missing variable %s", key)
		}
		result = strings.ReplaceAll(result, token, val)
	}
	return result, nil
}

func filterRules(rules []domain.Rule, filters []string) ([]domain.Rule, error) {
	selected := map[int]struct{}{}
	for _, token := range filters {
		matched := false
		if n, err := strconv.Atoi(token); err == nil {
			if n < 1 || n > len(rules) {
				return nil, fmt.Errorf("rule index out of range: %s", token)
			}
			selected[n-1] = struct{}{}
			matched = true
		} else {
			for idx, rule := range rules {
				if strings.EqualFold(rule.Name, token) {
					selected[idx] = struct{}{}
					matched = true
				}
			}
		}
		if !matched {
			return nil, fmt.Errorf("rule not found: %s", token)
		}
	}

	filtered := make([]domain.Rule, 0, len(selected))
	for idx, rule := range rules {
		if _, ok := selected[idx]; ok {
			filtered = append(filtered, rule)
		}
	}
	return filtered, nil
}
