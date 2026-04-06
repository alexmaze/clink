package cli

import (
	"errors"
	"fmt"

	"github.com/alexmaze/clink/internal/configload"
	"github.com/alexmaze/clink/internal/executor"
	"github.com/alexmaze/clink/internal/planner"
	"github.com/alexmaze/clink/internal/report"
)

type applyOptions struct {
	commonFlags
}

func parseApply(args []string) (applyOptions, error) {
	var opts applyOptions
	fs := newFlagSet("apply")
	registerCommon(fs, &opts.commonFlags)
	return opts, fs.Parse(args)
}

func (a app) runApply(opts applyOptions, err error) error {
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
	plan, err := planner.BuildApplyPlan(cfg)
	if err != nil {
		return err
	}
	format := outputFormat(opts.Output)
	if err := report.PrintPlan(a.stdout, format, plan); err != nil {
		return err
	}
	if opts.DryRun {
		return nil
	}
	if !opts.Yes && uiImpl.Interactive() {
		ok, err := uiImpl.Confirm("Apply planned changes?", []string{
			fmt.Sprintf("Rules: %d", len(plan.SelectedRules)),
			fmt.Sprintf("Actions: %d", len(plan.Actions)),
			fmt.Sprintf("Backup directory: %s", plan.BackupDir),
		})
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("operation cancelled")
		}
	}
	if err := persistSnapshot(cfg, plan.BackupDir); err != nil {
		return err
	}
	result, err := executor.New(cfg, false).Run(plan)
	if printErr := report.PrintResult(a.stdout, format, result); printErr != nil {
		return printErr
	}
	return err
}
