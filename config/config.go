package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/alexmaze/clink/lib/color"
	"github.com/alexmaze/clink/lib/fileutil"
	"github.com/alexmaze/clink/lib/icon"
	"github.com/alexmaze/clink/lib/spinner"
	"github.com/alexmaze/clink/lib/tui"
	"github.com/spf13/viper"
)

// 变量查找正则
var patternVar = regexp.MustCompile(`\${[0-9a-zA-Z_-]+}`)

// ConfigFile struct to `config.yaml`
type ConfigFile struct {
	Mode       Mode                  `mapstructure:"mode"`
	Hooks      *Hooks                `mapstructure:"hooks"`
	SSHServers map[string]*SSHServer `mapstructure:"ssh_servers"`
	Vars       map[string]string     `mapstructure:"vars"`
	Rules      []*Rule               `mapstructure:"rules"`
}

// Config clink configs
type Config struct {
	WorkDIR    string // path to `config.yaml`
	DryRun     bool
	BackupPath string
	ConfigPath string // absolute path to the config.yaml used for this run

	*ConfigFile
}

// ReadConfig initialize global configs
func ReadConfig(dryRun bool, configPath string, ruleFilter []string) *Config {
	sp := spinner.New()

	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		sp.Failedf("Failed to resolve config path %v: %v", configPath, err)
		os.Exit(1)
	}

	// 1. 显示配置文件路径
	sp.CheckPoint(icon.IconInfo, color.ColorCyan, "Using: "+absConfigPath, color.ColorReset)

	// 2. 读取并解析配置（带 spinner 动画）
	sp.Start("Parsing config...")
	configFile := readConfigFile(sp, absConfigPath)
	sp.Stop()
	sp.CheckPoint(icon.IconCheck, color.ColorGreen, "Config loaded", color.ColorReset)

	// 2.5. 按 -r 参数过滤 rules（若指定）
	if len(ruleFilter) > 0 {
		configFile.Rules = FilterRules(sp, configFile.Rules, ruleFilter)
	}

	// 3. 友好展示配置预览（此时用户可以看清楚要操作的内容）
	printConfigPreview(sp, configFile)

	// 4. 询问备份路径（用户已知道要备份什么）
	backupPath := confirmBackupPath(sp)

	// 5. 对 SSH 模式中未配置密钥/密码的服务器统一 prompt 输入密码
	PromptSSHPasswords(sp, configFile)

	// 6. 再次确认执行（带上下文边框）
	bullets := buildConfirmBullets(configFile, backupPath)
	if !tui.RunConfirm(tui.ConfirmOpts{
		Title:   "即将执行以下操作",
		Bullets: bullets,
	}) {
		sp.Failed("Canceled")
		os.Exit(0)
	}

	workDIR := filepath.Dir(absConfigPath)
	cfg := &Config{
		DryRun:     dryRun,
		WorkDIR:    workDIR,
		BackupPath: backupPath,
		ConfigPath: absConfigPath,
		ConfigFile: configFile,
	}

	sp.Success("Ready!")
	return cfg
}

// PromptSSHPasswords prompts for passwords for SSH servers that have no key and no password set.
func PromptSSHPasswords(sp spinner.Spinner, configFile *ConfigFile) {
	seen := map[string]bool{}
	needed := []string{}
	for _, rule := range configFile.Rules {
		if rule.Mode != ModeSSH {
			continue
		}
		srv, ok := configFile.SSHServers[rule.SSH]
		if !ok {
			continue
		}
		if srv.Key == "" && srv.Password == "" && !seen[rule.SSH] {
			seen[rule.SSH] = true
			needed = append(needed, rule.SSH)
		}
	}

	for _, name := range needed {
		srv := configFile.SSHServers[name]
		label := fmt.Sprintf("Password for %s@%s (server: %s)", srv.User, srv.Host, name)
		pwd, err := tui.RunMaskedInput(label)
		if err != nil {
			sp.Failedf("failed to read password for server %s: %v", name, err)
			os.Exit(1)
		}
		srv.Password = pwd
	}
}

// printConfigPreview 以人类可读格式展示配置预览
func printConfigPreview(sp spinner.Spinner, configFile *ConfigFile) {
	totalItems := 0
	for _, rule := range configFile.Rules {
		totalItems += len(rule.Items)
	}

	sp.CheckPoint(icon.IconInfo, color.ColorYellow,
		fmt.Sprintf("Config preview: (%d rules, %d items total)",
			len(configFile.Rules), totalItems),
		color.ColorReset)
	fmt.Println()

	// 显示顶层 hooks
	if configFile.Hooks != nil {
		if configFile.Hooks.Pre != "" {
			fmt.Printf("  %spre-all hook:%s  %s\n",
				color.ColorYellow, color.ColorReset, configFile.Hooks.Pre)
		}
		if configFile.Hooks.Post != "" {
			fmt.Printf("  %spost-all hook:%s %s\n",
				color.ColorYellow, color.ColorReset, configFile.Hooks.Post)
		}
		fmt.Println()
	}

	fmt.Print(RenderRulesTable(configFile))
}

// BuildModeLabel constructs the display label like "symlink", "copy", or "ssh → user@host"
func BuildModeLabel(configFile *ConfigFile, rule *Rule) string {
	switch rule.Mode {
	case ModeSSH:
		if srv, ok := configFile.SSHServers[rule.SSH]; ok {
			return fmt.Sprintf("ssh → %s@%s", srv.User, srv.Host)
		}
		return "ssh"
	case ModeCopy:
		return "copy"
	default:
		return "symlink"
	}
}

// ParseConfigFileOnly parses a config.yaml file and resolves mode inheritance,
// SSH server references, and variable substitution — but skips source-file
// existence checks.  It returns an error instead of calling os.Exit, making it
// suitable for restoring from a config snapshot where the original sources may
// no longer be present.
func ParseConfigFileOnly(absPath string) (*ConfigFile, error) {
	v := viper.New()
	v.SetConfigType("yaml")

	f, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %v: %w", absPath, err)
	}
	defer f.Close()

	if err = v.ReadConfig(f); err != nil {
		return nil, fmt.Errorf("failed to read config file %v: %w", absPath, err)
	}

	var configFile ConfigFile
	if err = v.Unmarshal(&configFile); err != nil {
		return nil, fmt.Errorf("wrong config file format %v: %w", absPath, err)
	}

	workDir := filepath.Dir(absPath)

	for _, rule := range configFile.Rules {
		// Mode 继承
		if rule.Mode == "" {
			if configFile.Mode != "" {
				rule.Mode = configFile.Mode
			} else {
				rule.Mode = ModeSymlink
			}
		}

		// SSH 模式：验证 server 引用，补全默认 port
		if rule.Mode == ModeSSH {
			if configFile.SSHServers == nil {
				return nil, fmt.Errorf("rule %q uses ssh mode but no ssh_servers defined in config", rule.Name)
			}
			srv, ok := configFile.SSHServers[rule.SSH]
			if !ok {
				return nil, fmt.Errorf("rule %q references unknown ssh server %q", rule.Name, rule.SSH)
			}
			if srv.Port == 0 {
				srv.Port = 22
			}
			if srv.Key != "" {
				expanded, kerr := fileutil.ParsePath("", srv.Key)
				if kerr == nil {
					srv.Key = expanded
				}
			}
		}

		for _, item := range rule.Items {
			// src | dest 变量替换
			item.Source, err = renderVars(configFile.Vars, item.Source)
			if err != nil {
				return nil, fmt.Errorf("failed to render %v: %w", item.Source, err)
			}

			item.Destination, err = renderVars(configFile.Vars, item.Destination)
			if err != nil {
				return nil, fmt.Errorf("failed to render %v: %w", item.Destination, err)
			}

			// src 路径绝对化
			item.Source, err = fileutil.ParsePath(workDir, item.Source)
			if err != nil {
				return nil, fmt.Errorf("failed to standard path %v: %w", item.Source, err)
			}

			// dest 路径绝对化：SSH 模式下 dest 是远程路径，不做本地处理
			if rule.Mode != ModeSSH {
				item.Destination, err = fileutil.ParsePath(workDir, item.Destination)
				if err != nil {
					return nil, fmt.Errorf("failed to standard path %v: %w", item.Destination, err)
				}
			}

			// 注意：跳过 source 文件存在性和类型检测（还原场景 source 不相关）
		}
	}

	return &configFile, nil
}

// unmarshall yaml config file content to struct
func readConfigFile(sp spinner.Spinner, absPath string) (c *ConfigFile) {
	configFile, err := ParseConfigFileOnly(absPath)
	if err != nil {
		sp.Failedf("%v", err)
		os.Exit(1)
	}

	// 额外校验：检测 source 文件是否存在并判断类型
	for _, rule := range configFile.Rules {
		for _, item := range rule.Items {
			exists, err := fileutil.IsFileExists(item.Source)
			if err != nil {
				sp.Failedf("failed to check if source file/folder exist %v: %v", item.Source, err)
				os.Exit(1)
			}

			if !exists {
				sp.Failedf("source file/folder do not exist %v", item.Source)
				os.Exit(1)
			}

			item.Type, err = fileutil.GetPathType(item.Source)
			if err != nil {
				sp.Failedf("failed to detect path type %v: %v", item.Source, err)
				os.Exit(1)
			}
		}
	}

	return configFile
}

// FilterRules returns the subset of rules that match any token in filter.
// Each token is either:
//   - a 1-based numeric string ("1", "2", …) → matched by original index
//   - a non-numeric string → matched against rule.Name (case-insensitive, exact)
//
// Unmatched tokens print a warning but do not abort.
// The returned slice preserves original order and has no duplicates.
func FilterRules(sp spinner.Spinner, rules []*Rule, filter []string) []*Rule {
	selected := make([]*Rule, 0, len(filter))
	seen := map[int]bool{}

	addRule := func(idx int) {
		if !seen[idx] {
			seen[idx] = true
			selected = append(selected, rules[idx])
		}
	}

	for _, token := range filter {
		matched := false
		if n, err := strconv.Atoi(token); err == nil {
			// 1-based index
			if n >= 1 && n <= len(rules) {
				addRule(n - 1)
				matched = true
			}
		} else {
			// name match (case-insensitive)
			for i, rule := range rules {
				if strings.EqualFold(rule.Name, token) {
					addRule(i)
					matched = true
				}
			}
		}
		if !matched {
			sp.CheckPoint(icon.IconInfo, color.ColorYellow,
				fmt.Sprintf("rule not found: %q", token), color.ColorReset)
		}
	}

	// sort selected by original index to preserve rule order
	result := make([]*Rule, 0, len(seen))
	for i, rule := range rules {
		if seen[i] {
			result = append(result, rule)
		}
	}
	return result
}

func renderVars(vars map[string]string, str string) (content string, err error) {
	matchedArgs := patternVar.FindAll([]byte(str), -1)

	if len(matchedArgs) == 0 {
		return str, nil
	}

	content = str

	for _, arg := range matchedArgs {
		argKey := strings.ToLower(string(arg[2 : len(arg)-1]))

		argVal, exists := vars[argKey]

		if !exists {
			return "", fmt.Errorf("can't find variable %v", argKey)
		}

		content = strings.ReplaceAll(content, string(arg), argVal)
	}

	return
}

// confirmBackupPath ask user to confirm original config files (if exists) backup path
func confirmBackupPath(sp spinner.Spinner) string {
	defaultBase, err := fileutil.ParsePath("", "~/.clink")
	if err != nil {
		defaultBase = ""
	}

	suggestedPath := filepath.Join(defaultBase, time.Now().Format("20060102_150405"))

	result, err := tui.RunInput("备份原有文件到", suggestedPath)
	if err != nil {
		sp.Failedf("failed to get backup path %v", err)
		os.Exit(1)
	}

	err = os.MkdirAll(result, 0700)
	if err != nil {
		sp.Failedf("failed to create backup path %v, %v", result, err)
		os.Exit(1)
	}

	return result
}

// buildConfirmBullets returns the bullet lines shown in the confirm box.
func buildConfirmBullets(configFile *ConfigFile, backupPath string) []string {
	// Count items by mode
	countByMode := map[Mode]int{}
	for _, rule := range configFile.Rules {
		for range rule.Items {
			countByMode[rule.Mode]++
		}
	}
	totalItems := 0
	var modeParts []string
	for _, mode := range []Mode{ModeSymlink, ModeCopy, ModeSSH} {
		n := countByMode[mode]
		if n > 0 {
			totalItems += n
			modeParts = append(modeParts, fmt.Sprintf("%s×%d", string(mode), n))
		}
	}
	itemLine := fmt.Sprintf("部署 %d 个 item (%s)", totalItems, strings.Join(modeParts, ", "))

	bullets := []string{itemLine}

	if backupPath != "" {
		bullets = append(bullets, fmt.Sprintf("备份原文件到 %s", backupPath))
	}

	// List SSH servers involved
	sshSeen := map[string]bool{}
	for _, rule := range configFile.Rules {
		if rule.Mode == ModeSSH && !sshSeen[rule.SSH] {
			sshSeen[rule.SSH] = true
			if srv, ok := configFile.SSHServers[rule.SSH]; ok {
				bullets = append(bullets, fmt.Sprintf("SSH 服务器: %s (%s@%s)", rule.SSH, srv.User, srv.Host))
			}
		}
	}

	return bullets
}

// ReadConfigForCheck parses config and prepares it for the --check command.
// Unlike ReadConfig it does not ask for a backup path or a "Proceed" confirmation,
// because --check is a read-only inspection and makes no changes to the filesystem.
func ReadConfigForCheck(configPath string, ruleFilter []string) (*Config, error) {
	sp := spinner.New()

	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config path %v: %w", configPath, err)
	}

	sp.CheckPoint(icon.IconInfo, color.ColorCyan, "Using: "+absConfigPath, color.ColorReset)

	sp.Start("Parsing config...")
	configFile, err := ParseConfigFileOnly(absConfigPath)
	sp.Stop()
	if err != nil {
		return nil, err
	}
	sp.CheckPoint(icon.IconCheck, color.ColorGreen, "Config loaded", color.ColorReset)

	if len(ruleFilter) > 0 {
		configFile.Rules = FilterRules(sp, configFile.Rules, ruleFilter)
	}

	// Prompt SSH passwords for servers that need them (needed to connect during check).
	PromptSSHPasswords(sp, configFile)

	workDIR := filepath.Dir(absConfigPath)
	cfg := &Config{
		DryRun:     false,
		WorkDIR:    workDIR,
		BackupPath: "",
		ConfigPath: absConfigPath,
		ConfigFile: configFile,
	}
	return cfg, nil
}
