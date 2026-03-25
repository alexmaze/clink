package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
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
	Vars  map[string]string `mapstructure:"vars"`
	Rules []*Rule           `mapstructure:"rules"`
}

// Config clink configs
type Config struct {
	WorkDIR    string // path to `config.yaml`
	DryRun     bool
	BackupPath string

	*ConfigFile
}

// ReadConfig initialize global configs
func ReadConfig(dryRun bool, configPath string) *Config {
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

	// 3. 友好展示配置预览（此时用户可以看清楚要操作的内容）
	printConfigPreview(sp, configFile)

	// 4. 询问备份路径（用户已知道要备份什么）
	backupPath := confirmBackupPath(sp)

	// 5. 再次确认执行
	p := promptui.Prompt{
		Label:     "Proceed with linking",
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

	for i, rule := range configFile.Rules {
		fmt.Printf("  %s[%d]%s %s\n",
			color.ColorCyan,
			i+1,
			color.ColorReset,
			color.ColorWhite.Color(rule.Name))
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

	// 解析 Type, 替换绝对路径
	for _, rule := range configFile.Rules {
		for _, item := range rule.Items {
			// src | dest 变量替换
			item.Source, err = renderVars(configFile.Vars, item.Source)
			if err != nil {
				sp.Failedf("failed to render %v: %v", item.Source, err)
				os.Exit(1)
			}

			item.Destination, err = renderVars(configFile.Vars, item.Destination)
			if err != nil {
				sp.Failedf("failed to render %v: %v", item.Source, err)
				os.Exit(1)
			}

			// 路径绝对化
			workDir := filepath.Dir(absPath)
			item.Source, err = fileutil.ParsePath(workDir, item.Source)
			if err != nil {
				sp.Failedf("failed to standard path %v: %v", item.Source, err)
				os.Exit(1)
			}

			item.Destination, err = fileutil.ParsePath(workDir, item.Destination)
			if err != nil {
				sp.Failedf("failed to standard path %v: %v", item.Destination, err)
				os.Exit(1)
			}

			// 检测 src 是否存在
			exists, err := fileutil.IsFileExists(item.Source)
			if err != nil {
				sp.Failedf("failed to check if source file/folder exist %v: %v", item.Source, err)
				os.Exit(1)
			}

			if !exists {
				sp.Failedf("source file/folder do not exist %v: %v", item.Source, err)
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
