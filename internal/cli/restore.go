package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexmaze/clink/internal/configload"
	"github.com/alexmaze/clink/internal/domain"
	"github.com/alexmaze/clink/internal/executor"
	"github.com/alexmaze/clink/internal/planner"
	"github.com/alexmaze/clink/internal/report"
	"github.com/alexmaze/clink/internal/ui"
	"github.com/alexmaze/clink/lib/fileutil"
)

type restoreOptions struct {
	commonFlags
	Backup string
}

func parseRestore(args []string) (restoreOptions, error) {
	var opts restoreOptions
	fs := newFlagSet("restore")
	registerCommon(fs, &opts.commonFlags)
	fs.StringVar(&opts.Backup, "backup", "", "backup directory name or absolute path")
	return opts, fs.Parse(args)
}

func (a app) runRestore(opts restoreOptions, err error) error {
	if err != nil {
		return err
	}
	uiImpl := chooseUI(opts.NonInteractive)
	backupDir, manifest, err := selectBackup(uiImpl, opts.Backup)
	if err != nil {
		return err
	}
	cfg, err := configload.LoadForRestore(defaultConfig(opts.configPathOrManifest(manifest)), nil)
	if err != nil {
		return err
	}
	if err := promptSecrets(uiImpl, cfg); err != nil {
		return err
	}
	plan, err := planner.BuildRestorePlan(*manifest, backupDir, []string(opts.Rules))
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
		ok, err := uiImpl.Confirm("Restore backup contents?", []string{
			fmt.Sprintf("Backup: %s", backupDir),
			fmt.Sprintf("Entries: %d", len(plan.Actions)),
		})
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("operation cancelled")
		}
	}
	result, err := executor.New(cfg, false).Run(plan)
	if printErr := report.PrintResult(a.stdout, format, result); printErr != nil {
		return printErr
	}
	return err
}

func selectBackup(uiImpl ui.UI, requested string) (string, *domain.BackupManifest, error) {
	base, err := fileutil.ParsePath("", "~/.clink")
	if err != nil {
		return "", nil, err
	}
	backups, err := findBackups(base)
	if err != nil {
		return "", nil, err
	}
	if len(backups) == 0 {
		return "", nil, fmt.Errorf("no backups found")
	}

	var selected string
	if requested != "" {
		if filepath.IsAbs(requested) {
			selected = requested
		} else {
			selected = filepath.Join(base, requested)
		}
	} else if uiImpl.Interactive() {
		items := make([]string, 0, len(backups))
		for _, backup := range backups {
			items = append(items, filepath.Base(backup))
		}
		index, err := uiImpl.Select("Select a backup", items)
		if err != nil {
			return "", nil, err
		}
		selected = backups[index]
	} else {
		selected = backups[0]
	}

	manifest, err := readManifest(filepath.Join(selected, "manifest.json"))
	if err != nil {
		return "", nil, err
	}
	return selected, manifest, nil
}

func readManifest(path string) (*domain.BackupManifest, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest domain.BackupManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func (opts restoreOptions) configPathOrManifest(manifest *domain.BackupManifest) string {
	if opts.ConfigPath != "" {
		return opts.ConfigPath
	}
	snapshot := manifest.ConfigSnapshot
	if snapshot != "" {
		return snapshot
	}
	return manifest.ConfigPath
}
