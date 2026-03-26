package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── renderVars tests ────────────────────────────────────────────────────────

func TestRenderVars_NoVars(t *testing.T) {
	result, err := renderVars(nil, "/some/path")
	assert.NoError(t, err)
	assert.Equal(t, "/some/path", result)
}

func TestRenderVars_SingleVar(t *testing.T) {
	vars := map[string]string{"home": "/Users/test"}
	result, err := renderVars(vars, "${home}/.vimrc")
	assert.NoError(t, err)
	assert.Equal(t, "/Users/test/.vimrc", result)
}

func TestRenderVars_MultipleVars(t *testing.T) {
	vars := map[string]string{
		"home": "/Users/test",
		"app":  "myapp",
	}
	result, err := renderVars(vars, "${home}/.config/${app}")
	assert.NoError(t, err)
	assert.Equal(t, "/Users/test/.config/myapp", result)
}

func TestRenderVars_MissingVar(t *testing.T) {
	vars := map[string]string{}
	_, err := renderVars(vars, "${missing}/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}

func TestRenderVars_CaseInsensitive(t *testing.T) {
	vars := map[string]string{"home": "/Users/test"}
	result, err := renderVars(vars, "${HOME}/.vimrc")
	assert.NoError(t, err)
	assert.Equal(t, "/Users/test/.vimrc", result)
}

func TestRenderVars_VarWithDash(t *testing.T) {
	vars := map[string]string{"my-var": "value"}
	result, err := renderVars(vars, "prefix_${my-var}_suffix")
	assert.NoError(t, err)
	assert.Equal(t, "prefix_value_suffix", result)
}

func TestRenderVars_VarWithUnderscore(t *testing.T) {
	vars := map[string]string{"my_var": "value"}
	result, err := renderVars(vars, "${my_var}/path")
	assert.NoError(t, err)
	assert.Equal(t, "value/path", result)
}

func TestRenderVars_NoPattern(t *testing.T) {
	vars := map[string]string{"home": "/Users/test"}
	result, err := renderVars(vars, "/absolute/path/no/vars")
	assert.NoError(t, err)
	assert.Equal(t, "/absolute/path/no/vars", result)
}

func TestRenderVars_DuplicateVar(t *testing.T) {
	vars := map[string]string{"v": "X"}
	result, err := renderVars(vars, "${v}/${v}/${v}")
	assert.NoError(t, err)
	assert.Equal(t, "X/X/X", result)
}

// ── ParseConfigFileOnly tests ───────────────────────────────────────────────

func TestParseConfigFileOnly_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "dotfiles")
	require.NoError(t, os.MkdirAll(srcDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte("hello"), 0644))

	configContent := `
vars:
  dest: ` + dir + `/output

rules:
  - name: test-rule
    items:
      - src: ./dotfiles/test.txt
        dest: ${dest}/test.txt
`
	configPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	cf, err := ParseConfigFileOnly(configPath)
	require.NoError(t, err)
	require.NotNil(t, cf)
	assert.Len(t, cf.Rules, 1)
	assert.Equal(t, "test-rule", cf.Rules[0].Name)
	assert.Equal(t, ModeSymlink, cf.Rules[0].Mode) // default mode
	assert.Len(t, cf.Rules[0].Items, 1)
	assert.Equal(t, filepath.Join(dir, "output", "test.txt"), cf.Rules[0].Items[0].Destination)
	assert.Equal(t, filepath.Join(dir, "dotfiles", "test.txt"), cf.Rules[0].Items[0].Source)
}

func TestParseConfigFileOnly_ExplicitMode(t *testing.T) {
	dir := t.TempDir()

	configContent := `
mode: copy

rules:
  - name: copy-rule
    items:
      - src: /tmp/src
        dest: /tmp/dest
  - name: symlink-override
    mode: symlink
    items:
      - src: /tmp/src2
        dest: /tmp/dest2
`
	configPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	cf, err := ParseConfigFileOnly(configPath)
	require.NoError(t, err)
	assert.Equal(t, ModeCopy, cf.Rules[0].Mode)
	assert.Equal(t, ModeSymlink, cf.Rules[1].Mode)
}

func TestParseConfigFileOnly_GlobalModeInheritance(t *testing.T) {
	dir := t.TempDir()

	configContent := `
mode: copy

rules:
  - name: inherits
    items:
      - src: /tmp/a
        dest: /tmp/b
`
	configPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	cf, err := ParseConfigFileOnly(configPath)
	require.NoError(t, err)
	assert.Equal(t, ModeCopy, cf.Rules[0].Mode)
}

func TestParseConfigFileOnly_DefaultSymlinkMode(t *testing.T) {
	dir := t.TempDir()

	configContent := `
rules:
  - name: default-mode
    items:
      - src: /tmp/a
        dest: /tmp/b
`
	configPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	cf, err := ParseConfigFileOnly(configPath)
	require.NoError(t, err)
	assert.Equal(t, ModeSymlink, cf.Rules[0].Mode)
}

func TestParseConfigFileOnly_SSHMode(t *testing.T) {
	dir := t.TempDir()

	configContent := `
ssh_servers:
  myserver:
    host: example.com
    user: root

rules:
  - name: ssh-rule
    mode: ssh
    ssh: myserver
    items:
      - src: /tmp/src
        dest: /remote/dest
`
	configPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	cf, err := ParseConfigFileOnly(configPath)
	require.NoError(t, err)
	assert.Equal(t, ModeSSH, cf.Rules[0].Mode)
	assert.Equal(t, "myserver", cf.Rules[0].SSH)
	assert.Equal(t, 22, cf.SSHServers["myserver"].Port)
}

func TestParseConfigFileOnly_SSHCustomPort(t *testing.T) {
	dir := t.TempDir()

	configContent := `
ssh_servers:
  myserver:
    host: example.com
    user: root
    port: 2222

rules:
  - name: ssh-rule
    mode: ssh
    ssh: myserver
    items:
      - src: /tmp/src
        dest: /remote/dest
`
	configPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	cf, err := ParseConfigFileOnly(configPath)
	require.NoError(t, err)
	assert.Equal(t, 2222, cf.SSHServers["myserver"].Port)
}

func TestParseConfigFileOnly_SSHMissingServer(t *testing.T) {
	dir := t.TempDir()

	configContent := `
rules:
  - name: bad-ssh
    mode: ssh
    ssh: nonexistent
    items:
      - src: /tmp/src
        dest: /remote/dest
`
	configPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	_, err := ParseConfigFileOnly(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no ssh_servers defined")
}

func TestParseConfigFileOnly_SSHUnknownServer(t *testing.T) {
	dir := t.TempDir()

	configContent := `
ssh_servers:
  real-server:
    host: example.com
    user: root

rules:
  - name: bad-ref
    mode: ssh
    ssh: wrong-name
    items:
      - src: /tmp/src
        dest: /remote/dest
`
	configPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	_, err := ParseConfigFileOnly(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown ssh server")
}

func TestParseConfigFileOnly_VarSubstitutionError(t *testing.T) {
	dir := t.TempDir()

	configContent := `
rules:
  - name: bad-var
    items:
      - src: /tmp/src
        dest: ${undefined_var}/dest
`
	configPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	_, err := ParseConfigFileOnly(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "undefined_var")
}

func TestParseConfigFileOnly_NonexistentFile(t *testing.T) {
	_, err := ParseConfigFileOnly("/nonexistent/config.yaml")
	assert.Error(t, err)
}

func TestParseConfigFileOnly_EmptyRules(t *testing.T) {
	dir := t.TempDir()
	configContent := `
rules: []
`
	configPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	cf, err := ParseConfigFileOnly(configPath)
	require.NoError(t, err)
	assert.Empty(t, cf.Rules)
}

func TestParseConfigFileOnly_Hooks(t *testing.T) {
	dir := t.TempDir()

	configContent := `
hooks:
  pre: echo hello
  post: echo bye

rules:
  - name: with-hooks
    hooks:
      pre: echo rule-pre
      post: echo rule-post
    items:
      - src: /tmp/src
        dest: /tmp/dest
`
	configPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	cf, err := ParseConfigFileOnly(configPath)
	require.NoError(t, err)
	assert.NotNil(t, cf.Hooks)
	assert.Equal(t, "echo hello", cf.Hooks.Pre)
	assert.Equal(t, "echo bye", cf.Hooks.Post)
	assert.NotNil(t, cf.Rules[0].Hooks)
	assert.Equal(t, "echo rule-pre", cf.Rules[0].Hooks.Pre)
	assert.Equal(t, "echo rule-post", cf.Rules[0].Hooks.Post)
}

func TestParseConfigFileOnly_RelativeSourcePath(t *testing.T) {
	dir := t.TempDir()

	configContent := `
rules:
  - name: relative
    items:
      - src: ./dotfiles/test.txt
        dest: /tmp/dest.txt
`
	configPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	cf, err := ParseConfigFileOnly(configPath)
	require.NoError(t, err)
	// Source should be resolved relative to config dir
	assert.Equal(t, filepath.Join(dir, "dotfiles", "test.txt"), cf.Rules[0].Items[0].Source)
}

func TestParseConfigFileOnly_SSHDestNotExpanded(t *testing.T) {
	dir := t.TempDir()

	configContent := `
ssh_servers:
  srv:
    host: example.com
    user: root

rules:
  - name: ssh
    mode: ssh
    ssh: srv
    items:
      - src: /local/path
        dest: /remote/path
`
	configPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	cf, err := ParseConfigFileOnly(configPath)
	require.NoError(t, err)
	// SSH dest should remain as-is (not expanded locally)
	assert.Equal(t, "/remote/path", cf.Rules[0].Items[0].Destination)
}

// ── Mode constants tests ────────────────────────────────────────────────────

func TestModeConstants(t *testing.T) {
	assert.Equal(t, Mode("symlink"), ModeSymlink)
	assert.Equal(t, Mode("copy"), ModeCopy)
	assert.Equal(t, Mode("ssh"), ModeSSH)
}

// ── patternVar regex tests ──────────────────────────────────────────────────

func TestPatternVar_Matches(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"${home}", []string{"${home}"}},
		{"${HOME}", []string{"${HOME}"}},
		{"${a-b}", []string{"${a-b}"}},
		{"${a_b}", []string{"${a_b}"}},
		{"${v1}/${v2}", []string{"${v1}", "${v2}"}},
		{"no vars here", nil},
		{"${123}", []string{"${123}"}},
	}

	for _, tt := range tests {
		matches := patternVar.FindAllString(tt.input, -1)
		assert.Equal(t, tt.expected, matches, "input: %s", tt.input)
	}
}

// ── Struct types tests ──────────────────────────────────────────────────────

func TestConfigFileStruct(t *testing.T) {
	cf := &ConfigFile{
		Mode: ModeSymlink,
		Hooks: &Hooks{
			Pre:  "echo pre",
			Post: "echo post",
		},
		SSHServers: map[string]*SSHServer{
			"test": {Host: "localhost", Port: 22, User: "root"},
		},
		Vars: map[string]string{"key": "value"},
		Rules: []*Rule{
			{
				Name: "test",
				Mode: ModeCopy,
				Items: []*RuleItem{
					{Source: "/src", Destination: "/dest"},
				},
			},
		},
	}

	assert.Equal(t, ModeSymlink, cf.Mode)
	assert.NotNil(t, cf.Hooks)
	assert.Len(t, cf.SSHServers, 1)
	assert.Len(t, cf.Vars, 1)
	assert.Len(t, cf.Rules, 1)
}
