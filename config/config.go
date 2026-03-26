package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/alexmaze/clink/lib/color"
	"github.com/alexmaze/clink/lib/fileutil"
	"github.com/alexmaze/clink/lib/icon"
	"github.com/alexmaze/clink/lib/spinner"
	"github.com/manifoldco/promptui"
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
		configFile.Rules = filterRules(sp, configFile.Rules, ruleFilter)
	}

	// 3. 友好展示配置预览（此时用户可以看清楚要操作的内容）
	printConfigPreview(sp, configFile)

	// 4. 询问备份路径（用户已知道要备份什么）
	backupPath := confirmBackupPath(sp)

	// 5. 对 SSH 模式中未配置密钥/密码的服务器统一 prompt 输入密码
	promptSSHPasswords(sp, configFile)

	// 6. 再次确认执行
	p := promptui.Prompt{
		Label:     "Proceed",
		IsConfirm: true,
	}
	_, err = p.Run()
	if err != nil {
		sp.Failed("Canceled")
		os.Exit(0)
	}

	workDIR := filepath.Dir(absConfigPath)
	cfg := &Config{
		DryRun:     dryRun,
		WorkDIR:    workDIR,
		BackupPath: backupPath,
		ConfigFile: configFile,
	}

	sp.Success("Ready!")
	return cfg
}

// promptSSHPasswords prompts for passwords for SSH servers that have no key and no password set.
func promptSSHPasswords(sp spinner.Spinner, configFile *ConfigFile) {
	// collect server names used in ssh-mode rules that need a password prompt
	needed := map[string]bool{}
	for _, rule := range configFile.Rules {
		if rule.Mode != ModeSSH {
			continue
		}
		srv, ok := configFile.SSHServers[rule.SSH]
		if !ok {
			continue
		}
		if srv.Key == "" && srv.Password == "" {
			needed[rule.SSH] = true
		}
	}

	for name := range needed {
		srv := configFile.SSHServers[name]
		p := promptui.Prompt{
			Label: fmt.Sprintf("Password for %s@%s (server: %s)", srv.User, srv.Host, name),
			Mask:  '*',
		}
		pwd, err := p.Run()
		if err != nil {
			sp.Failedf("failed to read password for server %s: %v", name, err)
			os.Exit(1)
		}
		srv.Password = pwd
	}
}

// printConfigPreview 以人类可读格式展示配置预览，不依赖 yaml.Marshal
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

	for i, rule := range configFile.Rules {
		// 构建 mode 标签
		modeLabel := buildModeLabel(configFile, rule)

		fmt.Printf("  %s[%d]%s %s  %s[%s]%s\n",
			color.ColorCyan,
			i+1,
			color.ColorReset,
			color.ColorWhite.Color(rule.Name),
			color.ColorGray, modeLabel, color.ColorReset)

		if rule.Hooks != nil {
			if rule.Hooks.Pre != "" {
				fmt.Printf("      %spre-hook:%s  %s\n",
					color.ColorYellow, color.ColorReset, rule.Hooks.Pre)
			}
			if rule.Hooks.Post != "" {
				fmt.Printf("      %spost-hook:%s %s\n",
					color.ColorYellow, color.ColorReset, rule.Hooks.Post)
			}
		}
		for _, item := range rule.Items {
			fmt.Printf("      • %s  %s[%s]%s\n        %s→%s  %s\n",
				item.Source,
				color.ColorGray, string(item.Type), color.ColorReset,
				color.ColorGray, color.ColorReset,
				item.Destination)
		}
		fmt.Println()
	}
}

// buildModeLabel constructs the display label like "symlink", "copy", or "ssh → user@host"
func buildModeLabel(configFile *ConfigFile, rule *Rule) string {
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

// unmarshall yaml config file content to struct
func readConfigFile(sp spinner.Spinner, absPath string) (c *ConfigFile) {
	viper.SetConfigType("yaml")

	f, err := os.Open(absPath)
	if err != nil {
		sp.Failedf("failed to open config file %v: %v", absPath, err)
		os.Exit(1)
	}
	defer f.Close()

	if err = viper.ReadConfig(f); err != nil {
		sp.Failedf("failed to read config file %v: %v", absPath, err)
		os.Exit(1)
	}

	var configFile ConfigFile
	if err = viper.Unmarshal(&configFile); err != nil {
		sp.Failedf("wrong config file format %v: %v", absPath, err)
		os.Exit(1)
	}

	workDir := filepath.Dir(absPath)

	// 解析 Type, 替换绝对路径
	for _, rule := range configFile.Rules {
		// Mode 继承：rule 未设置则继承全局，全局也未设置则默认 symlink
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
				sp.Failedf("rule %q uses ssh mode but no ssh_servers defined in config", rule.Name)
				os.Exit(1)
			}
			srv, ok := configFile.SSHServers[rule.SSH]
			if !ok {
				sp.Failedf("rule %q references unknown ssh server %q", rule.Name, rule.SSH)
				os.Exit(1)
			}
			if srv.Port == 0 {
				srv.Port = 22
			}
			// expand key path if provided
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
				sp.Failedf("failed to render %v: %v", item.Source, err)
				os.Exit(1)
			}

			item.Destination, err = renderVars(configFile.Vars, item.Destination)
			if err != nil {
				sp.Failedf("failed to render %v: %v", item.Destination, err)
				os.Exit(1)
			}

			// src 路径绝对化（src 始终是本地路径）
			item.Source, err = fileutil.ParsePath(workDir, item.Source)
			if err != nil {
				sp.Failedf("failed to standard path %v: %v", item.Source, err)
				os.Exit(1)
			}

			// dest 路径绝对化：SSH 模式下 dest 是远程路径，不做本地处理
			if rule.Mode != ModeSSH {
				item.Destination, err = fileutil.ParsePath(workDir, item.Destination)
				if err != nil {
					sp.Failedf("failed to standard path %v: %v", item.Destination, err)
					os.Exit(1)
				}
			}

			// 检测 src 是否存在
			exists, err := fileutil.IsFileExists(item.Source)
			if err != nil {
				sp.Failedf("failed to check if source file/folder exist %v: %v", item.Source, err)
				os.Exit(1)
			}

			if !exists {
				sp.Failedf("source file/folder do not exist %v", item.Source)
				os.Exit(1)
			}

			// 判断 src 类型：文件 | 文件夹
			item.Type, err = fileutil.GetPathType(item.Source)
			if err != nil {
				sp.Failedf("failed to detect path type %v: %v", item.Source, err)
				os.Exit(1)
			}
		}
	}

	return &configFile
}

// filterRules returns the subset of rules that match any token in filter.
// Each token is either:
//   - a 1-based numeric string ("1", "2", …) → matched by original index
//   - a non-numeric string → matched against rule.Name (case-insensitive, exact)
//
// Unmatched tokens print a warning but do not abort.
// The returned slice preserves original order and has no duplicates.
func filterRules(sp spinner.Spinner, rules []*Rule, filter []string) []*Rule {
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

	// 生成含时间戳的建议路径，让用户直接 Enter 接受即可
	suggestedPath := path.Join(defaultBase, time.Now().Format("20060102_150405"))

	p := promptui.Prompt{
		Label:   "Backup existing files to",
		Default: suggestedPath,
	}

	result, err := p.Run()
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
