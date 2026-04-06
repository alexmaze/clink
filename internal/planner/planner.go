package planner

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/alexmaze/clink/internal/domain"
	"github.com/alexmaze/clink/lib/fileutil"
)

func BuildApplyPlan(cfg *domain.Config) (*domain.Plan, error) {
	if len(cfg.Rules) == 0 {
		return nil, fmt.Errorf("config must define at least one rule")
	}
	backupBase, err := fileutil.ParsePath("", "~/.clink")
	if err != nil {
		return nil, fmt.Errorf("resolve backup base: %w", err)
	}

	now := time.Now()
	backupDir := filepath.Join(backupBase, now.Format("20060102_150405"))
	plan := &domain.Plan{
		Command:       "apply",
		ConfigPath:    cfg.ConfigPath,
		BackupDir:     backupDir,
		CreatedAt:     now,
		SelectedRules: selectedRuleNames(cfg.Rules),
	}

	addSecrets(cfg, plan)

	if cfg.Hooks != nil && cfg.Hooks.Pre != "" {
		plan.Actions = append(plan.Actions, domain.Action{
			Type:        domain.ActionRunHook,
			HookScope:   "pre_all",
			HookCommand: cfg.Hooks.Pre,
		})
	}

	for _, rule := range cfg.Rules {
		if rule.Hooks != nil && rule.Hooks.Pre != "" {
			plan.Actions = append(plan.Actions, domain.Action{
				Type:        domain.ActionRunHook,
				RuleName:    rule.Name,
				HookScope:   "pre_rule",
				HookCommand: rule.Hooks.Pre,
			})
		}

		for _, item := range rule.Items {
			kind, err := detectKind(item.Source)
			if err != nil {
				return nil, err
			}
			actionBackup := domain.Action{
				RuleName:    rule.Name,
				Mode:        rule.Mode,
				SSHServer:   rule.SSH,
				Source:      item.Source,
				Destination: item.Destination,
				PathKind:    kind,
				BackupPath:  payloadPath(backupDir, rule.Name, item.Destination),
			}
			if rule.Mode == domain.ModeSSH {
				actionBackup.Type = domain.ActionBackupRemote
			} else {
				actionBackup.Type = domain.ActionBackupLocal
			}
			plan.Actions = append(plan.Actions, actionBackup)

			actionDeploy := domain.Action{
				RuleName:    rule.Name,
				Mode:        rule.Mode,
				SSHServer:   rule.SSH,
				Source:      item.Source,
				Destination: item.Destination,
				PathKind:    kind,
			}
			switch rule.Mode {
			case domain.ModeCopy:
				actionDeploy.Type = domain.ActionDeployCopy
			case domain.ModeSSH:
				actionDeploy.Type = domain.ActionDeploySSH
			default:
				actionDeploy.Type = domain.ActionDeploySymlink
			}
			plan.Actions = append(plan.Actions, actionDeploy)
		}

		if rule.Hooks != nil && rule.Hooks.Post != "" {
			plan.Actions = append(plan.Actions, domain.Action{
				Type:        domain.ActionRunHook,
				RuleName:    rule.Name,
				HookScope:   "post_rule",
				HookCommand: rule.Hooks.Post,
			})
		}
	}

	if cfg.Hooks != nil && cfg.Hooks.Post != "" {
		plan.Actions = append(plan.Actions, domain.Action{
			Type:        domain.ActionRunHook,
			HookScope:   "post_all",
			HookCommand: cfg.Hooks.Post,
		})
	}

	plan.Actions = append(plan.Actions, domain.Action{
		Type:       domain.ActionWriteManifest,
		BackupPath: filepath.Join(backupDir, "manifest.json"),
	})
	return plan, nil
}

func BuildCheckPlan(cfg *domain.Config) (*domain.Plan, error) {
	if len(cfg.Rules) == 0 {
		return nil, fmt.Errorf("config must define at least one rule")
	}
	plan := &domain.Plan{
		Command:       "check",
		ConfigPath:    cfg.ConfigPath,
		CreatedAt:     time.Now(),
		SelectedRules: selectedRuleNames(cfg.Rules),
	}
	addSecrets(cfg, plan)

	for _, rule := range cfg.Rules {
		for _, item := range rule.Items {
			kind, err := detectKind(item.Source)
			if err != nil {
				return nil, err
			}
			action := domain.Action{
				RuleName:    rule.Name,
				Mode:        rule.Mode,
				SSHServer:   rule.SSH,
				Source:      item.Source,
				Destination: item.Destination,
				PathKind:    kind,
			}
			switch rule.Mode {
			case domain.ModeCopy:
				action.Type = domain.ActionCheckCopy
			case domain.ModeSSH:
				action.Type = domain.ActionCheckSSH
			default:
				action.Type = domain.ActionCheckSymlink
			}
			plan.Actions = append(plan.Actions, action)
		}
	}
	return plan, nil
}

func BuildRestorePlan(manifest domain.BackupManifest, selectedBackup string, filters []string) (*domain.Plan, error) {
	entries := manifest.Entries
	if len(filters) > 0 {
		filtered := make([]domain.BackupEntry, 0, len(entries))
		for _, entry := range entries {
			if containsRule(filters, entry.RuleName) {
				filtered = append(filtered, entry)
			}
		}
		entries = filtered
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("backup contains no restorable entries")
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].RuleName == entries[j].RuleName {
			return entries[i].Destination < entries[j].Destination
		}
		return entries[i].RuleName < entries[j].RuleName
	})

	plan := &domain.Plan{
		Command:       "restore",
		ConfigPath:    manifest.ConfigPath,
		BackupDir:     selectedBackup,
		CreatedAt:     time.Now(),
		SelectedRules: backupRuleNames(entries),
	}

	for _, entry := range entries {
		action := domain.Action{
			RuleName:    entry.RuleName,
			Mode:        entry.Mode,
			Source:      entry.BackupPath,
			Destination: entry.Destination,
			SSHServer:   entry.SSHServer,
			PathKind:    entry.PathKind,
			BackupPath:  entry.BackupPath,
		}
		if entry.Mode == domain.ModeSSH {
			action.Type = domain.ActionRestoreSSH
		} else {
			action.Type = domain.ActionRestoreLocal
		}
		plan.Actions = append(plan.Actions, action)
	}
	return plan, nil
}

func payloadPath(backupDir, ruleName, destination string) string {
	clean := destination
	if filepath.IsAbs(clean) {
		clean = clean[1:]
	}
	return filepath.Join(backupDir, "payload", ruleSlug(ruleName), clean)
}

func detectKind(path string) (domain.PathKind, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("detect path kind for %s: %w", path, err)
	}
	if info.IsDir() {
		return domain.PathKindDirectory, nil
	}
	return domain.PathKindFile, nil
}

func addSecrets(cfg *domain.Config, plan *domain.Plan) {
	seen := map[string]struct{}{}
	for _, rule := range cfg.Rules {
		if rule.Mode != domain.ModeSSH {
			continue
		}
		server := cfg.SSHServers[rule.SSH]
		if server.Key == "" && server.Password == "" {
			if _, ok := seen[rule.SSH]; !ok {
				seen[rule.SSH] = struct{}{}
				plan.RequiredSecrets = append(plan.RequiredSecrets, rule.SSH)
			}
		}
	}
}

func selectedRuleNames(rules []domain.Rule) []string {
	names := make([]string, 0, len(rules))
	for _, rule := range rules {
		names = append(names, rule.Name)
	}
	return names
}

func containsRule(filters []string, ruleName string) bool {
	for _, filter := range filters {
		if filter == ruleName {
			return true
		}
	}
	return false
}

func backupRuleNames(entries []domain.BackupEntry) []string {
	seen := map[string]struct{}{}
	names := []string{}
	for _, entry := range entries {
		if _, ok := seen[entry.RuleName]; ok {
			continue
		}
		seen[entry.RuleName] = struct{}{}
		names = append(names, entry.RuleName)
	}
	return names
}

func ruleSlug(input string) string {
	out := make([]rune, 0, len(input))
	lastDash := false
	for _, r := range input {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
			lastDash = false
		case r >= 'A' && r <= 'Z':
			out = append(out, r+'a'-'A')
			lastDash = false
		default:
			if !lastDash {
				out = append(out, '-')
				lastDash = true
			}
		}
	}
	if len(out) == 0 {
		return "rule"
	}
	return string(out)
}
