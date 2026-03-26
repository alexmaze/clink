package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/alexmaze/clink/config"
	"github.com/alexmaze/clink/lib/color"
	"github.com/alexmaze/clink/lib/icon"
	"github.com/alexmaze/clink/lib/spinner"
	"github.com/cosiner/flag"
	"github.com/manifoldco/promptui"
)

// ClinkOpts command line options
type ClinkOpts struct {
	DryRun     bool   `names:"-d, --dry-run" usage:"dry-run mode, will only dispaly changes but will not execute"`
	ConfigPath string `names:"-c, --config"  usage:"specify config file path"`
}

// Metadata command line usages
func (t *ClinkOpts) Metadata() map[string]flag.Flag {
	const (
		usage   = "clink is a tool to help you centralized manage your configuration files or folders."
		version = `
		version: v0.0.1
		`
		desc = `
		With clink, you can put all your configuration files or folders like '.bashrc', '.vim/',
		"appconfig"... etc, in to any place you like, for example, one dropbox synced folder.
		Then what you need to do is write a "config.yaml" to specify where those config files
		or folders should be, and simply run "clink -c <path>/config.yaml", clink will automaticly
		sync your configs to where they should be.
		By using clink, you can conveniently to sync, share and backup your configs~
		`
	)
	return map[string]flag.Flag{
		"": {
			Usage:   usage,
			Version: version,
			Desc:    desc,
		},
		"-d": {
			Default: false,
		},
		"-c": {
			Default: "./config.yaml",
			Desc:    `e.g. -c ./config.yaml`,
		},
	}
}

// RuleResult holds per-rule execution statistics
type RuleResult struct {
	linked  int
	skipped int
	failed  int
}

func main() {
	var opts ClinkOpts

	flag.NewFlagSet(flag.Flag{}).ParseStruct(&opts, os.Args...)

	cfg := config.ReadConfig(opts.DryRun, opts.ConfigPath)
	sp := spinner.New()

	// pre-all hook
	if cfg.Hooks != nil && cfg.Hooks.Pre != "" {
		runHook(sp, "pre-all", cfg.Hooks.Pre)
	}

	totalRules := len(cfg.Rules)
	var totalLinked, totalSkipped, totalFailed int

	for i, rule := range cfg.Rules {
		result := executeRule(cfg, rule, i+1, totalRules)
		totalLinked += result.linked
		totalSkipped += result.skipped
		totalFailed += result.failed
	}

	// post-all hook
	if cfg.Hooks != nil && cfg.Hooks.Post != "" {
		runHook(sp, "post-all", cfg.Hooks.Post)
	}

	// 执行总结
	fmt.Println()
	sp.Successf("Done!  %s%d linked%s,  %s%d skipped%s,  %s%d failed%s.",
		color.ColorGreen, totalLinked, color.ColorReset,
		color.ColorYellow, totalSkipped, color.ColorReset,
		color.ColorRed, totalFailed, color.ColorReset)
}

// runHook executes a shell command as a hook (pre/post). On failure it prints
// an error message and exits the whole process with code 1.
func runHook(sp spinner.Spinner, label, cmd string) {
	sp.CheckPoint(icon.IconInfo, color.ColorYellow,
		fmt.Sprintf("  hook [%s]: %s", label, cmd), color.ColorReset)

	c := exec.Command("sh", "-c", cmd)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		sp.Failedf("hook [%s] failed: %v", label, err)
		os.Exit(1)
	}
}

func executeRule(cfg *config.Config, rule *config.Rule, ruleIndex, totalRules int) RuleResult {
	sp := spinner.New()
	result := RuleResult{}

	totalItems := len(rule.Items)
	sp.CheckPoint(icon.IconInfo, color.ColorCyan,
		fmt.Sprintf("[%d/%d] %s  (%d items)", ruleIndex, totalRules, rule.Name, totalItems),
		color.ColorReset)

	// pre-rule hook
	if rule.Hooks != nil && rule.Hooks.Pre != "" {
		runHook(sp, "pre", rule.Hooks.Pre)
	}

	for itemIndex, item := range rule.Items {
		// 每个 item 处理期间启动 spinner
		sp.Start(fmt.Sprintf("[%d/%d] processing %s ...", itemIndex+1, totalItems, item.Destination))

		// ── backup ──────────────────────────────────────────────────
		backupDest := path.Join(cfg.BackupPath, item.Destination)

		destExists, _ := fileutil_destExists(item.Destination)

		if !destExists {
			// 目标不存在，跳过备份
			sp.Stop()
			sp.CheckPoint(icon.IconInfo, color.ColorGray,
				fmt.Sprintf("  → backup  %s  (skipped, not exist)", item.Destination),
				color.ColorReset)
		} else {
			err := os.MkdirAll(path.Dir(backupDest), 0755)
			if err != nil {
				sp.Stop()
				sp.CheckPoint(icon.IconCross, color.ColorRed,
					fmt.Sprintf("  → backup  %s  failed to create dir: %s", item.Destination, err.Error()),
					color.ColorReset)
				sp.CheckPoint(icon.IconCross, color.ColorRed, "  → skip it.", color.ColorReset)
				result.skipped++
				continue
			}

			err = os.Rename(item.Destination, backupDest)
			if err != nil {
				sp.Stop()
				sp.CheckPoint(icon.IconCross, color.ColorRed,
					fmt.Sprintf("  → backup  %s  failed: %s", item.Destination, err.Error()),
					color.ColorReset)

				p := promptui.Prompt{
					Label:     "Continue anyway",
					IsConfirm: true,
				}
				_, confirmErr := p.Run()
				if confirmErr != nil {
					sp.Failed("Aborted")
					os.Exit(0)
				}
			} else {
				sp.Stop()
				sp.CheckPoint(icon.IconCheck, color.ColorGreen,
					fmt.Sprintf("  → backup  %s  ✔", item.Destination),
					color.ColorReset)
				sp.Start(fmt.Sprintf("[%d/%d] linking %s ...", itemIndex+1, totalItems, item.Destination))
			}
		}

		// ── link ─────────────────────────────────────────────────────
		err := os.MkdirAll(path.Dir(item.Destination), 0755)
		if err != nil {
			sp.Stop()
			sp.CheckPoint(icon.IconCross, color.ColorRed,
				fmt.Sprintf("  → link    %s  failed to create dir: %s", item.Destination, err.Error()),
				color.ColorReset)
			sp.CheckPoint(icon.IconCross, color.ColorRed, "  → skip it.", color.ColorReset)
			result.failed++
			continue
		}

		err = os.Symlink(item.Source, item.Destination)
		sp.Stop()
		if err != nil {
			sp.CheckPoint(icon.IconCross, color.ColorRed,
				fmt.Sprintf("  → link    %s  →  %s  failed: %s", item.Source, item.Destination, err.Error()),
				color.ColorReset)
			result.failed++
			continue
		}

		sp.CheckPoint(icon.IconCheck, color.ColorGreen,
			fmt.Sprintf("  → link    %s  →  %s  ✔", item.Source, item.Destination),
			color.ColorReset)
		result.linked++
	}

	// post-rule hook
	if rule.Hooks != nil && rule.Hooks.Post != "" {
		runHook(sp, "post", rule.Hooks.Post)
	}

	return result
}

// fileutil_destExists checks whether a path exists (file or symlink)
func fileutil_destExists(p string) (bool, error) {
	_, err := os.Lstat(p)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
