package main

import (
	"os"
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

func main() {
	var opts ClinkOpts

	flag.NewFlagSet(flag.Flag{}).ParseStruct(&opts, os.Args...)

	cfg := config.ReadConfig(opts.DryRun, opts.ConfigPath)

	for _, rule := range cfg.Rules {
		executeRule(cfg, rule)
	}
}

func executeRule(cfg *config.Config, rule *config.Rule) {
	sp := spinner.New()
	sp.CheckPoint(icon.IconInfo, color.ColorCyan, "Execute rule: "+rule.Name, color.ColorReset)

	for _, item := range rule.Items {

		// backup
		backupDest := path.Join(cfg.BackupPath, item.Destination)

		sp.CheckPoint(icon.IconInfo, color.ColorCyan, "\tbackup: "+item.Destination+" to "+backupDest, color.ColorReset)

		err := os.MkdirAll(path.Dir(backupDest), 0755)
		if err != nil {
			sp.CheckPoint(icon.IconCross, color.ColorRed, "Failed to create dir: "+item.Destination+": "+err.Error(), color.ColorReset)
			sp.CheckPoint(icon.IconCross, color.ColorRed, "Skip it.", color.ColorReset)
			continue
		}

		err = os.Rename(item.Destination, backupDest)
		if err != nil {
			sp.CheckPoint(icon.IconCross, color.ColorRed, "Failed to backup "+item.Source+": "+err.Error(), color.ColorReset)

			p := promptui.Prompt{
				Label:     "Continue anyway",
				IsConfirm: true,
			}
			_, err = p.Run()
			if err != nil {
				sp.Failed("Aborted")
				os.Exit(0)
			}
		}

		// link

		err = os.MkdirAll(path.Dir(item.Destination), 0755)
		if err != nil {
			sp.CheckPoint(icon.IconCross, color.ColorRed, "Failed to create dir: "+item.Destination+": "+err.Error(), color.ColorReset)
			sp.CheckPoint(icon.IconCross, color.ColorRed, "Skip it.", color.ColorReset)
			continue
		}

		err = os.Symlink(item.Source, item.Destination)
		if err != nil {
			sp.CheckPoint(icon.IconCross, color.ColorRed, "Failed to link "+item.Source+" to "+item.Destination+": "+err.Error(), color.ColorReset)
			sp.CheckPoint(icon.IconCross, color.ColorRed, "Skip it.", color.ColorReset)
		}
		sp.CheckPoint(icon.IconCheck, color.ColorCyan, "\tlink: "+item.Source+" to "+item.Destination, color.ColorReset)
	}
}
