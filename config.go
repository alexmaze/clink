package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/alexmaze/clink/lib/fileutil"
	"github.com/alexmaze/clink/lib/spinner"
	"github.com/manifoldco/promptui"
	"github.com/spf13/viper"
)

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

// AppConfig global configs
var AppConfig *Config

// SetupConfig initialize global configs
func SetupConfig(sp spinner.Spinner, dryRun bool, configPath string) {
	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		sp.Failedf("failed to resolve config path %v: %v", configPath, err)
		os.Exit(1)
	}

	sp.Successf("Using %v", absConfigPath)

	workDIR := filepath.Dir(absConfigPath)
	backupPath := confirmBackupPath(sp)
	configFile := readConfigFile(absConfigPath)

	cfg := &Config{
		DryRun:     dryRun,
		WorkDIR:    workDIR,
		BackupPath: backupPath,
		ConfigFile: configFile,
	}

	AppConfig = confirmConfig(sp, cfg)
}

// unmarshall yaml config file content to struct
func readConfigFile(absPath string) (c *ConfigFile) {
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

var VAR_REGEXP_PATTERN, _ = regexp.Compile(`\${[0-9a-zA-Z_-]+}`)

func renderVars(vars map[string]string, str string) (content string, err error) {

	matchedArgs := VAR_REGEXP_PATTERN.FindAll([]byte(str), -1)

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

// ask user to confirm original config files (if exists) backup path
func confirmBackupPath(sp spinner.Spinner) string {

	defaultPath, err := fileutil.ParsePath("", "~/.clink")
	if err != nil {
		defaultPath = ""
	}

	p := promptui.Prompt{
		Label:   "Please specify your backup path",
		Default: defaultPath,
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

// ask user to confirm variables defined in config file
// and render config file with absolute pathes
func confirmConfig(sp spinner.Spinner, cfg *Config) *Config {
	bts, _ := json.MarshalIndent(cfg, "", "  ")
	fmt.Println(string(bts))

	// TODO render variables
	return nil
}
