package planner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alexmaze/clink/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildApplyPlanIncludesHooksAndManifest(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, ".zshrc")
	require.NoError(t, os.WriteFile(source, []byte("export SHELL"), 0644))

	cfg := &domain.Config{
		ConfigPath: filepath.Join(dir, "config.yaml"),
		Hooks:      &domain.Hooks{Pre: "echo pre", Post: "echo post"},
		Rules: []domain.Rule{
			{
				Name:  "shell",
				Mode:  domain.ModeSymlink,
				Hooks: &domain.Hooks{Pre: "echo rule pre", Post: "echo rule post"},
				Items: []domain.RuleItem{
					{Source: source, Destination: "/tmp/.zshrc"},
				},
			},
		},
	}

	plan, err := BuildApplyPlan(cfg)
	require.NoError(t, err)
	assert.Equal(t, "apply", plan.Command)
	assert.Equal(t, []string{"shell"}, plan.SelectedRules)
	require.NotEmpty(t, plan.BackupDir)
	require.Len(t, plan.Actions, 7)
	assert.Equal(t, domain.ActionRunHook, plan.Actions[0].Type)
	assert.Equal(t, domain.ActionBackupLocal, plan.Actions[2].Type)
	assert.Equal(t, domain.ActionDeploySymlink, plan.Actions[3].Type)
	assert.Equal(t, domain.ActionWriteManifest, plan.Actions[6].Type)
	assert.Contains(t, plan.Actions[2].BackupPath, filepath.Join("payload", "shell"))
}

func TestBuildCheckPlanCollectsSecretsForSSH(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "nginx.conf")
	require.NoError(t, os.WriteFile(source, []byte("worker_processes 1;"), 0644))

	cfg := &domain.Config{
		ConfigPath: filepath.Join(dir, "config.yaml"),
		SSHServers: map[string]domain.SSHServer{
			"prod": {Host: "example.com", User: "root"},
		},
		Rules: []domain.Rule{
			{
				Name: "remote",
				Mode: domain.ModeSSH,
				SSH:  "prod",
				Items: []domain.RuleItem{
					{Source: source, Destination: "/etc/nginx/nginx.conf"},
				},
			},
		},
	}

	plan, err := BuildCheckPlan(cfg)
	require.NoError(t, err)
	require.Len(t, plan.Actions, 1)
	assert.Equal(t, domain.ActionCheckSSH, plan.Actions[0].Type)
	assert.Equal(t, []string{"prod"}, plan.RequiredSecrets)
}

func TestBuildRestorePlanFiltersRules(t *testing.T) {
	manifest := domain.BackupManifest{
		ConfigPath: "/tmp/config.snapshot.yaml",
		Entries: []domain.BackupEntry{
			{RuleName: "shell", Mode: domain.ModeCopy, BackupPath: "/backup/shell", Destination: "/tmp/.zshrc", PathKind: domain.PathKindFile},
			{RuleName: "app", Mode: domain.ModeSSH, BackupPath: "/backup/app", Destination: "/etc/app.yaml", PathKind: domain.PathKindFile, SSHServer: "prod"},
		},
	}

	plan, err := BuildRestorePlan(manifest, "/backup", []string{"app"})
	require.NoError(t, err)
	require.Len(t, plan.Actions, 1)
	assert.Equal(t, domain.ActionRestoreSSH, plan.Actions[0].Type)
	assert.Equal(t, []string{"app"}, plan.SelectedRules)
}
