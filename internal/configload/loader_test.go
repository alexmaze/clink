package configload

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alexmaze/clink/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadNormalizesConfigAndFiltersRules(t *testing.T) {
	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "dotfiles")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, ".zshrc"), []byte("export ZDOTDIR"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "app.yaml"), []byte("port: 8080"), 0644))

	configPath := filepath.Join(dir, "config.yaml")
	body := `
mode: copy
vars:
  app_home: /tmp/app
ssh_servers:
  prod:
    host: example.com
    user: root
    key: ./keys/id_rsa
rules:
  - name: shell
    mode: symlink
    items:
      - src: ./dotfiles/.zshrc
        dest: ~/.zshrc
  - name: app
    items:
      - src: ./dotfiles/app.yaml
        dest: ${APP_HOME}/app.yaml
`
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "keys"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "keys", "id_rsa"), []byte("key"), 0600))
	require.NoError(t, os.WriteFile(configPath, []byte(body), 0644))

	cfg, err := Load(configPath, []string{"app"})
	require.NoError(t, err)
	require.Len(t, cfg.Rules, 1)
	assert.Equal(t, "app", cfg.Rules[0].Name)
	assert.Equal(t, domain.ModeCopy, cfg.Rules[0].Mode)
	assert.Equal(t, filepath.Join(dir, "dotfiles", "app.yaml"), cfg.Rules[0].Items[0].Source)
	assert.Equal(t, "/tmp/app/app.yaml", cfg.Rules[0].Items[0].Destination)
	assert.Equal(t, filepath.Join(dir, "keys", "id_rsa"), cfg.SSHServers["prod"].Key)
}

func TestLoadForRestoreAllowsMissingSource(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.snapshot.yaml")
	body := `
rules:
  - name: remote
    items:
      - src: ./missing.txt
        dest: /tmp/missing.txt
`
	require.NoError(t, os.WriteFile(configPath, []byte(body), 0644))

	_, err := Load(configPath, nil)
	require.Error(t, err)

	cfg, err := LoadForRestore(configPath, nil)
	require.NoError(t, err)
	require.Len(t, cfg.Rules, 1)
	assert.Equal(t, domain.ModeSymlink, cfg.Rules[0].Mode)
}

func TestSaveWritesRelativeSources(t *testing.T) {
	dir := t.TempDir()
	cfg := &domain.Config{
		ConfigPath: filepath.Join(dir, "config.yaml"),
		WorkDir:    dir,
		Mode:       domain.ModeSymlink,
		SSHServers: map[string]domain.SSHServer{
			"prod": {
				Host: "example.com",
				User: "root",
				Key:  filepath.Join(dir, "keys", "id_rsa"),
			},
		},
		Rules: []domain.Rule{
			{
				Name: "shell",
				Mode: domain.ModeSymlink,
				Items: []domain.RuleItem{
					{
						Source:      filepath.Join(dir, ".clink", "sources", "shell", ".zshrc"),
						Destination: "/tmp/.zshrc",
					},
				},
			},
		},
	}

	require.NoError(t, Save(cfg))

	raw, err := os.ReadFile(cfg.ConfigPath)
	require.NoError(t, err)
	text := string(raw)
	assert.Contains(t, text, "src: ./.clink/sources/shell/.zshrc")
	assert.Contains(t, text, "key: ./keys/id_rsa")
}

func TestFilterRulesSupportsIndexAndName(t *testing.T) {
	rules := []domain.Rule{
		{Name: "shell"},
		{Name: "app"},
	}

	filtered, err := filterRules(rules, []string{"2", "shell"})
	require.NoError(t, err)
	require.Len(t, filtered, 2)
	assert.Equal(t, "shell", filtered[0].Name)
	assert.Equal(t, "app", filtered[1].Name)
}
