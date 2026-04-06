package cli

import (
	"errors"

	"github.com/alexmaze/clink/internal/configload"
	"github.com/alexmaze/clink/internal/executor"
	"github.com/alexmaze/clink/internal/planner"
	"github.com/alexmaze/clink/internal/report"
)

type checkOptions struct {
	commonFlags
}

func parseCheck(args []string) (checkOptions, error) {
	var opts checkOptions
	fs := newFlagSet("check")
	registerCommon(fs, &opts.commonFlags)
	return opts, fs.Parse(args)
}

func (a app) runCheck(opts checkOptions, err error) error {
	if err != nil {
		return err
	}
	cfg, err := configload.Load(defaultConfig(opts.ConfigPath), []string(opts.Rules))
	if err != nil {
		return err
	}
	uiImpl := chooseUI(opts.NonInteractive)
	if err := promptSecrets(uiImpl, cfg); err != nil {
		return err
	}
	plan, err := planner.BuildCheckPlan(cfg)
	if err != nil {
		return err
	}
	format := outputFormat(opts.Output)
	if err := report.PrintPlan(a.stdout, format, plan); err != nil {
		return err
	}
	result, err := executor.New(cfg, false).Run(plan)
	if printErr := report.PrintResult(a.stdout, format, result); printErr != nil {
		return printErr
	}
	if err != nil {
		return err
	}
	if result.Failed > 0 {
		return errors.New("check reported failures")
	}
	return nil
}
