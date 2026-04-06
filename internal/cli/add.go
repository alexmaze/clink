package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexmaze/clink/internal/configload"
	"github.com/alexmaze/clink/internal/domain"
	"github.com/alexmaze/clink/internal/executor"
	"github.com/alexmaze/clink/internal/report"
)

type addOptions struct {
	commonFlags
	Source string
	Dest   string
	Rule   string
	Name   string
	Mode   string
}

type addSpec struct {
	RuleName      string
	Mode          domain.Mode
	Source        string
	ManagedSource string
	Destination   string
	Kind          domain.PathKind
}

func parseAdd(args []string) (addOptions, error) {
	var opts addOptions
	fs := newFlagSet("add")
	registerCommon(fs, &opts.commonFlags)
	fs.StringVar(&opts.Dest, "dest", "", "destination path")
	fs.StringVar(&opts.Rule, "rule", "", "append to an existing rule")
	fs.StringVar(&opts.Name, "name", "", "create a new rule name")
	fs.StringVar(&opts.Mode, "mode", "", "mode: symlink or copy")
	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	if fs.NArg() != 1 {
		return opts, fmt.Errorf("usage: clink add [flags] <source>")
	}
	opts.Source = fs.Arg(0)
	return opts, nil
}

func (a app) runAdd(opts addOptions, err error) error {
	if err != nil {
		return err
	}
	cfg, err := configload.Load(defaultConfig(opts.ConfigPath), nil)
	if err != nil {
		return err
	}
	uiImpl := chooseUI(opts.NonInteractive)
	spec, err := buildAddSpec(cfg, opts)
	if err != nil {
		return err
	}
	plan := &domain.Plan{
		Command:       "add",
		ConfigPath:    cfg.ConfigPath,
		CreatedAt:     time.Now(),
		SelectedRules: []string{spec.RuleName},
		Actions: []domain.Action{
			{
				Type:        domain.ActionDeployCopy,
				RuleName:    spec.RuleName,
				Mode:        spec.Mode,
				Source:      spec.Source,
				Destination: spec.ManagedSource,
				PathKind:    spec.Kind,
			},
		},
	}
	format := outputFormat(opts.Output)
	if err := report.PrintPlan(a.stdout, format, plan); err != nil {
		return err
	}
	if !opts.Yes && uiImpl.Interactive() {
		ok, err := uiImpl.Confirm("Add source into clink management?", []string{
			fmt.Sprintf("Rule: %s", spec.RuleName),
			fmt.Sprintf("Mode: %s", spec.Mode),
			fmt.Sprintf("Managed source: %s", spec.ManagedSource),
			fmt.Sprintf("Destination: %s", spec.Destination),
		})
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("operation cancelled")
		}
	}
	exec := executor.New(cfg, opts.DryRun)
	result, err := exec.Run(plan)
	if err != nil {
		if result != nil {
			_ = report.PrintResult(a.stdout, format, result)
		}
		return err
	}
	appendRule(cfg, spec)
	if !opts.DryRun {
		if err := configload.Save(cfg); err != nil {
			return err
		}
	}
	return report.PrintResult(a.stdout, format, result)
}

func buildAddSpec(cfg *domain.Config, opts addOptions) (*addSpec, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	source, err := resolveLocalPath(cwd, cfg.WorkDir, opts.Source)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(source)
	if err != nil {
		return nil, err
	}
	kind := domain.PathKindFile
	if info.IsDir() {
		kind = domain.PathKindDirectory
	}

	ruleName := strings.TrimSpace(opts.Rule)
	mode := domain.Mode(opts.Mode)
	if ruleName != "" {
		rule, found := findRule(cfg.Rules, ruleName)
		if !found {
			return nil, fmt.Errorf("rule not found: %s", ruleName)
		}
		if rule.Mode == domain.ModeSSH {
			return nil, fmt.Errorf("cannot append to ssh rule: %s", rule.Name)
		}
		mode = rule.Mode
		ruleName = rule.Name
	}

	if ruleName == "" {
		ruleName = strings.TrimSpace(opts.Name)
		if ruleName == "" {
			ruleName = filepath.Base(source)
		}
		if mode == "" {
			mode = domain.ModeSymlink
		}
		if _, found := findRule(cfg.Rules, ruleName); found {
			return nil, fmt.Errorf("rule already exists: %s", ruleName)
		}
	}

	dest := opts.Dest
	if dest == "" {
		dest = source
	}
	dest, err = resolveLocalPath(cwd, cfg.WorkDir, dest)
	if err != nil {
		return nil, err
	}

	managedSource := filepath.Join(cfg.WorkDir, ".clink", "sources", slug(ruleName), filepath.Base(source))
	for _, rule := range cfg.Rules {
		for _, item := range rule.Items {
			if item.Destination == dest {
				return nil, fmt.Errorf("destination already managed: %s", dest)
			}
			if item.Source == source {
				return nil, fmt.Errorf("source already managed by rule %s", rule.Name)
			}
		}
	}
	return &addSpec{
		RuleName:      ruleName,
		Mode:          mode,
		Source:        source,
		ManagedSource: managedSource,
		Destination:   dest,
		Kind:          kind,
	}, nil
}

func appendRule(cfg *domain.Config, spec *addSpec) {
	rule, found := findRule(cfg.Rules, spec.RuleName)
	item := domain.RuleItem{
		Source:      spec.ManagedSource,
		Destination: spec.Destination,
		Kind:        spec.Kind,
	}
	if found {
		rule.Items = append(rule.Items, item)
		for idx := range cfg.Rules {
			if strings.EqualFold(cfg.Rules[idx].Name, spec.RuleName) {
				cfg.Rules[idx] = rule
				return
			}
		}
		return
	}
	cfg.Rules = append(cfg.Rules, domain.Rule{
		Name: spec.RuleName,
		Mode: spec.Mode,
		Items: []domain.RuleItem{
			item,
		},
	})
}

func findRule(rules []domain.Rule, name string) (domain.Rule, bool) {
	for _, rule := range rules {
		if strings.EqualFold(rule.Name, name) {
			return rule, true
		}
	}
	return domain.Rule{}, false
}
