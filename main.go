package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/alexmaze/clink/config"
	"github.com/alexmaze/clink/lib/color"
	"github.com/alexmaze/clink/lib/icon"
	"github.com/alexmaze/clink/lib/spinner"
	"github.com/alexmaze/clink/lib/sshutil"
	"github.com/cosiner/flag"
	"github.com/manifoldco/promptui"
)

// Version is set at build time via -ldflags
var Version = "dev"

// ClinkOpts command line options
type ClinkOpts struct {
	DryRun     bool     `names:"-d, --dry-run" usage:"dry-run mode, will only display changes but will not execute"`
	ConfigPath string   `names:"-c, --config"  usage:"specify config file path"`
	Rules      []string `names:"-r, --rule"    usage:"only run rules matching the given name or 1-based index (can be specified multiple times)"`
	Restore    bool     `names:"--restore"     usage:"interactively restore files from a previous backup"`
	Check      bool     `names:"--check"       usage:"check whether configured links are correctly established (read-only)"`
}

// Metadata command line usages
func (t *ClinkOpts) Metadata() map[string]flag.Flag {
	const (
		usage = "clink is a tool to help you centralized manage your configuration files or folders."
		desc  = `
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
			Version: "\n\tversion: " + Version + "\n\t",
			Desc:    desc,
		},
		"-d": {
			Default: false,
		},
		"-c": {
			Default: "./config.yaml",
			Desc:    `e.g. -c ./config.yaml`,
		},
		"-r": {
			Default: []string{},
			Desc:    `e.g. -r "vim 配置" -r 2`,
		},
		"--restore": {
			Default: false,
			Desc:    `interactively select a backup to restore`,
		},
		"--check": {
			Default: false,
			Desc:    `read-only health-check; prints which items are correctly linked/copied/uploaded`,
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

	if opts.Restore {
		runRestore(opts)
		return
	}

	if opts.Check {
		runCheck(opts)
		return
	}

	cfg := config.ReadConfig(opts.DryRun, opts.ConfigPath, opts.Rules)
	sp := spinner.New()

	// 快照 config.yaml 到备份目录
	snapshotConfig(sp, cfg)

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

	// SSH 模式：建立连接（规则内复用同一连接）
	var sshClient *sshutil.Client
	if rule.Mode == config.ModeSSH {
		srv := cfg.SSHServers[rule.SSH]
		var err error
		sp.Start(fmt.Sprintf("Connecting to %s@%s ...", srv.User, srv.Host))
		sshClient, err = sshutil.NewClient(srv)
		sp.Stop()
		if err != nil {
			sp.CheckPoint(icon.IconCross, color.ColorRed,
				fmt.Sprintf("  SSH connect failed: %s", err.Error()), color.ColorReset)
			result.failed += totalItems
			return result
		}
		defer sshClient.Close()
		sp.CheckPoint(icon.IconCheck, color.ColorGreen,
			fmt.Sprintf("  SSH connected to %s@%s", srv.User, srv.Host), color.ColorReset)
	}

	for itemIndex, item := range rule.Items {
		sp.Start(fmt.Sprintf("[%d/%d] processing %s ...", itemIndex+1, totalItems, item.Destination))

		skipped, err := backupItem(sp, cfg, item, rule.Mode, sshClient)
		if skipped {
			sp.Stop()
			result.skipped++
			continue
		}
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
		}

		err = deployItem(sp, cfg, item, rule.Mode, sshClient)
		sp.Stop()
		if err != nil {
			sp.CheckPoint(icon.IconCross, color.ColorRed,
				fmt.Sprintf("  → deploy  %s  failed: %s", item.Destination, err.Error()),
				color.ColorReset)
			result.failed++
			continue
		}

		actionLabel := modeActionLabel(rule.Mode)
		sp.CheckPoint(icon.IconCheck, color.ColorGreen,
			fmt.Sprintf("  → %s  %s  →  %s  ✔", actionLabel, item.Source, item.Destination),
			color.ColorReset)
		result.linked++
	}

	// post-rule hook
	if rule.Hooks != nil && rule.Hooks.Post != "" {
		runHook(sp, "post", rule.Hooks.Post)
	}

	return result
}

// modeActionLabel returns a short human-readable label for the deploy action.
func modeActionLabel(mode config.Mode) string {
	switch mode {
	case config.ModeCopy:
		return "copy  "
	case config.ModeSSH:
		return "upload"
	default:
		return "link  "
	}
}

// backupItem backs up the existing destination (if any).
//
//   - symlink / copy mode: os.Rename the local file into the backup directory.
//   - ssh mode: download the remote file to the local backup directory.
//
// Returns (true, nil) when there is nothing to back up (destination absent).
// Returns (false, err) on a hard backup failure that the caller should handle.
func backupItem(sp spinner.Spinner, cfg *config.Config, item *config.RuleItem, mode config.Mode, client *sshutil.Client) (skipped bool, err error) {
	backupDest := filepath.Join(cfg.BackupPath, item.Destination)

	switch mode {
	case config.ModeSSH:
		exists, err := client.Exists(item.Destination)
		if err != nil {
			return false, fmt.Errorf("check remote: %w", err)
		}
		if !exists {
			sp.Stop()
			sp.CheckPoint(icon.IconInfo, color.ColorGray,
				fmt.Sprintf("  → backup  %s  (skipped, not exist)", item.Destination),
				color.ColorReset)
			sp.Start(fmt.Sprintf("[processing] uploading %s ...", item.Destination))
			return false, nil
		}
		if cfg.DryRun {
			sp.Stop()
			sp.CheckPoint(icon.IconInfo, color.ColorYellow,
				fmt.Sprintf("  → [dry-run] would backup  %s  →  %s", item.Destination, backupDest),
				color.ColorReset)
			sp.Start(fmt.Sprintf("[dry-run] uploading %s ...", item.Destination))
			return false, nil
		}
		// Download remote → local backup
		if mkErr := os.MkdirAll(filepath.Dir(backupDest), 0755); mkErr != nil {
			return false, fmt.Errorf("create backup dir: %w", mkErr)
		}
		if dlErr := client.Download(item.Destination, backupDest); dlErr != nil {
			return false, fmt.Errorf("download backup: %w", dlErr)
		}
		sp.Stop()
		sp.CheckPoint(icon.IconCheck, color.ColorGreen,
			fmt.Sprintf("  → backup  %s  ✔", item.Destination),
			color.ColorReset)
		sp.Start(fmt.Sprintf("[processing] uploading %s ...", item.Destination))
		return false, nil

	default: // symlink or copy
		destExists, destErr := destExists(item.Destination)
		if destErr != nil {
			return false, fmt.Errorf("check destination: %w", destErr)
		}
		if !destExists {
			sp.Stop()
			sp.CheckPoint(icon.IconInfo, color.ColorGray,
				fmt.Sprintf("  → backup  %s  (skipped, not exist)", item.Destination),
				color.ColorReset)
			sp.Start(fmt.Sprintf("[processing] deploying %s ...", item.Destination))
			return false, nil
		}
		if cfg.DryRun {
			sp.Stop()
			sp.CheckPoint(icon.IconInfo, color.ColorYellow,
				fmt.Sprintf("  → [dry-run] would backup  %s  →  %s", item.Destination, backupDest),
				color.ColorReset)
			sp.Start(fmt.Sprintf("[dry-run] deploying %s ...", item.Destination))
			return false, nil
		}
		if mkErr := os.MkdirAll(filepath.Dir(backupDest), 0755); mkErr != nil {
			return false, fmt.Errorf("create backup dir: %w", mkErr)
		}
		if renameErr := os.Rename(item.Destination, backupDest); renameErr != nil {
			return false, renameErr
		}
		sp.Stop()
		sp.CheckPoint(icon.IconCheck, color.ColorGreen,
			fmt.Sprintf("  → backup  %s  ✔", item.Destination),
			color.ColorReset)
		sp.Start(fmt.Sprintf("[processing] deploying %s ...", item.Destination))
		return false, nil
	}
}

// deployItem deploys the source to its destination according to the rule mode.
func deployItem(sp spinner.Spinner, cfg *config.Config, item *config.RuleItem, mode config.Mode, client *sshutil.Client) error {
	if cfg.DryRun {
		actionLabel := modeActionLabel(mode)
		sp.CheckPoint(icon.IconInfo, color.ColorYellow,
			fmt.Sprintf("  → [dry-run] would %s  %s  →  %s", actionLabel, item.Source, item.Destination),
			color.ColorReset)
		return nil
	}

	switch mode {
	case config.ModeCopy:
		if err := os.MkdirAll(filepath.Dir(item.Destination), 0755); err != nil {
			return fmt.Errorf("create dest dir: %w", err)
		}
		return copyPath(item.Source, item.Destination)

	case config.ModeSSH:
		return client.Upload(item.Source, item.Destination)

	default: // symlink
		if err := os.MkdirAll(filepath.Dir(item.Destination), 0755); err != nil {
			return fmt.Errorf("create dest dir: %w", err)
		}
		return os.Symlink(item.Source, item.Destination)
	}
}

// copyPath recursively copies src to dest.
// If src is a file, dest is created/overwritten as a file.
// If src is a directory, dest is created as a directory and all contents are copied.
func copyPath(src, dest string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return copyFile(src, dest)
	}

	if err := os.MkdirAll(dest, info.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcChild := filepath.Join(src, entry.Name())
		destChild := filepath.Join(dest, entry.Name())
		if err := copyPath(srcChild, destChild); err != nil {
			return err
		}
	}
	return nil
}

// copyFile copies a single regular file from src to dest.
func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// destExists checks whether a path exists (file or symlink).
func destExists(p string) (bool, error) {
	_, err := os.Lstat(p)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// snapshotConfig copies the original config.yaml into the backup directory so
// that future restores know which rules/modes/SSH servers were used.
// Failures are non-fatal — a warning is printed but the deploy continues.
func snapshotConfig(sp spinner.Spinner, cfg *config.Config) {
	if cfg.DryRun || cfg.ConfigPath == "" || cfg.BackupPath == "" {
		return
	}
	dest := filepath.Join(cfg.BackupPath, "config.yaml")
	if err := copyFile(cfg.ConfigPath, dest); err != nil {
		sp.CheckPoint(icon.IconInfo, color.ColorYellow,
			fmt.Sprintf("Warning: failed to snapshot config.yaml: %v", err),
			color.ColorReset)
	}
}
