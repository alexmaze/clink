package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alexmaze/clink/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAddSpecForNewRule(t *testing.T) {
	configDir := t.TempDir()
	sourceDir := t.TempDir()
	source := filepath.Join(sourceDir, ".vimrc")
	require.NoError(t, os.WriteFile(source, []byte("set nu"), 0644))

	oldwd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldwd) }()
	require.NoError(t, os.Chdir(sourceDir))

	cfg := &domain.Config{
		ConfigPath: filepath.Join(configDir, "config.yaml"),
		WorkDir:    configDir,
	}

	spec, err := buildAddSpec(cfg, addOptions{Source: "./.vimrc"})
	require.NoError(t, err)
	assert.Equal(t, ".vimrc", spec.RuleName)
	assert.Equal(t, domain.ModeSymlink, spec.Mode)
	assert.Equal(t, source, spec.Source)
	assert.Equal(t, source, spec.Destination)
	assert.Equal(t, filepath.Join(configDir, ".clink", "sources", "vimrc", ".vimrc"), spec.ManagedSource)
}

func TestBuildAddSpecRejectsDuplicateDestination(t *testing.T) {
	configDir := t.TempDir()
	sourceDir := t.TempDir()
	source := filepath.Join(sourceDir, ".zshrc")
	require.NoError(t, os.WriteFile(source, []byte("export EDITOR=vim"), 0644))

	oldwd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldwd) }()
	require.NoError(t, os.Chdir(sourceDir))

	cfg := &domain.Config{
		WorkDir: configDir,
		Rules: []domain.Rule{
			{
				Name: "shell",
				Mode: domain.ModeSymlink,
				Items: []domain.RuleItem{
					{Source: filepath.Join(configDir, ".clink", "sources", "shell", ".bashrc"), Destination: source},
				},
			},
		},
	}

	_, err = buildAddSpec(cfg, addOptions{Source: "./.zshrc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "destination already managed")
}

func TestAppendRuleAddsAndAppends(t *testing.T) {
	cfg := &domain.Config{}
	spec1 := &addSpec{
		RuleName:      "shell",
		Mode:          domain.ModeSymlink,
		ManagedSource: "/tmp/source1",
		Destination:   "/tmp/dest1",
		Kind:          domain.PathKindFile,
	}
	appendRule(cfg, spec1)
	require.Len(t, cfg.Rules, 1)
	require.Len(t, cfg.Rules[0].Items, 1)

	spec2 := &addSpec{
		RuleName:      "shell",
		Mode:          domain.ModeSymlink,
		ManagedSource: "/tmp/source2",
		Destination:   "/tmp/dest2",
		Kind:          domain.PathKindFile,
	}
	appendRule(cfg, spec2)
	require.Len(t, cfg.Rules, 1)
	require.Len(t, cfg.Rules[0].Items, 2)
}
