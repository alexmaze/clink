package main

import (
	"fmt"
	"time"

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

var sp = spinner.New()

func main() {
	// var opts ClinkOpts

	// flag.NewFlagSet(flag.Flag{}).ParseStruct(&opts, os.Args...)

	// SetupConfig(sp, opts.DryRun, opts.ConfigPath)

	prompt := promptui.Prompt{
		Label: "Number",
	}

	result, err := prompt.Run()
	fmt.Println(err, result)

	confirm := promptui.Prompt{
		Label:     "Is every ok?",
		IsConfirm: true,
	}
	result, err = confirm.Run()
	fmt.Println(err, result)

	sp := spinner.New().Start()

	stop1 := time.NewTimer(time.Second * 1)
	stop2 := time.NewTimer(time.Second * 2)
	stop3 := time.NewTimer(time.Second * 10)

OUT:
	for {
		select {
		case <-stop1.C:
			sp.SetMsg("Working on 1, 12345678901234567890123456789012345678901234567890123456789012345678901234567890")
		case <-stop2.C:
			sp.CheckPoint(icon.IconCheck, color.ColorBlue, "what", color.ColorPurple)
			// sp.CheckPoint("O", spinner.ColorBlue, "what", spinner.ColorPurple)
			sp.SetMsg("Working on 2, 12345678901234567890123456789012345678901234567890123456789012345678901234567890")
			// sp.SetSpinGap(50 * time.Millisecond)
		case <-stop3.C:
			sp.Success("Everything good!")
			sp.Success("Bye")
			break OUT
		}
	}

}
