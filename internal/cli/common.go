package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alexmaze/clink/internal/domain"
	"github.com/alexmaze/clink/internal/report"
	"github.com/alexmaze/clink/internal/ui"
	"github.com/alexmaze/clink/lib/fileutil"
)

type commonFlags struct {
	ConfigPath     string
	Rules          multiValue
	DryRun         bool
	Yes            bool
	NonInteractive bool
	Output         string
}

type multiValue []string

func (m *multiValue) String() string {
	return strings.Join(*m, ",")
}

func (m *multiValue) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func registerCommon(fs *flag.FlagSet, flags *commonFlags) {
	fs.StringVar(&flags.ConfigPath, "config", "", "path to config.yaml")
	fs.StringVar(&flags.ConfigPath, "c", "", "path to config.yaml")
	fs.Var(&flags.Rules, "rule", "rule name or 1-based index, repeatable")
	fs.Var(&flags.Rules, "r", "rule name or 1-based index, repeatable")
	fs.BoolVar(&flags.DryRun, "dry-run", false, "preview actions without changes")
	fs.BoolVar(&flags.DryRun, "d", false, "preview actions without changes")
	fs.BoolVar(&flags.Yes, "yes", false, "skip confirmations")
	fs.BoolVar(&flags.Yes, "y", false, "skip confirmations")
	fs.BoolVar(&flags.NonInteractive, "non-interactive", false, "disable prompts")
	fs.StringVar(&flags.Output, "output", "text", "output format: text or json")
}

func defaultConfig(path string) string {
	if path != "" {
		return path
	}
	return "./config.yaml"
}

func outputFormat(raw string) report.Format {
	switch strings.ToLower(raw) {
	case "json":
		return report.FormatJSON
	default:
		return report.FormatText
	}
}

func chooseUI(nonInteractive bool) ui.UI {
	if nonInteractive {
		return ui.NonInteractive{}
	}
	return ui.TUI{}
}

func promptSecrets(uiImpl ui.UI, cfg *domain.Config) error {
	requested := map[string]struct{}{}
	for _, rule := range cfg.Rules {
		if rule.Mode != domain.ModeSSH {
			continue
		}
		server := cfg.SSHServers[rule.SSH]
		if server.Key == "" && server.Password == "" {
			if _, ok := requested[rule.SSH]; ok {
				continue
			}
			requested[rule.SSH] = struct{}{}
			secret, err := uiImpl.Secret(fmt.Sprintf("Password for %s@%s", server.User, server.Host))
			if err != nil {
				return err
			}
			server.Password = secret
			cfg.SSHServers[rule.SSH] = server
		}
	}
	return nil
}

func persistSnapshot(cfg *domain.Config, backupDir string) error {
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}
	body, err := os.ReadFile(cfg.ConfigPath)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(backupDir, "config.snapshot.yaml"), body, 0644)
}

func findBackups(base string) ([]string, error) {
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	backups := []string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(base, entry.Name())
		if _, err := os.Stat(filepath.Join(path, "manifest.json")); err == nil {
			backups = append(backups, path)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(backups)))
	return backups, nil
}

func resolveLocalPath(cwd, configDir, value string) (string, error) {
	if filepath.IsAbs(value) || strings.HasPrefix(value, "~/") {
		return fileutil.ParsePath(configDir, value)
	}
	return fileutil.ParsePath(cwd, value)
}

func slug(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	builder := strings.Builder{}
	lastDash := false
	for _, r := range input {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				builder.WriteRune('-')
				lastDash = true
			}
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "rule"
	}
	return result
}

func (a app) printUsage() {
	fmt.Fprintln(a.stdout, "clink")
	fmt.Fprintln(a.stdout)
	fmt.Fprintln(a.stdout, "Usage:")
	fmt.Fprintln(a.stdout, "  clink apply [flags]")
	fmt.Fprintln(a.stdout, "  clink check [flags]")
	fmt.Fprintln(a.stdout, "  clink restore [flags]")
	fmt.Fprintln(a.stdout, "  clink add [flags] <source>")
	fmt.Fprintln(a.stdout, "  clink version")
}
