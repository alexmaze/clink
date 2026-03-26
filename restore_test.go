package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexmaze/clink/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── scanBackups tests ───────────────────────────────────────────────────────

func TestScanBackups_Empty(t *testing.T) {
	dir := t.TempDir()
	backups, err := scanBackups(dir)
	assert.NoError(t, err)
	assert.Empty(t, backups)
}

func TestScanBackups_NonexistentDir(t *testing.T) {
	backups, err := scanBackups("/nonexistent/path")
	assert.NoError(t, err)
	assert.Nil(t, backups)
}

func TestScanBackups_ValidBackups(t *testing.T) {
	dir := t.TempDir()

	// Create two backup dirs with files
	backup1 := filepath.Join(dir, "20240101_120000")
	backup2 := filepath.Join(dir, "20240102_130000")
	require.NoError(t, os.MkdirAll(backup1, 0755))
	require.NoError(t, os.MkdirAll(backup2, 0755))

	// Add files to make them non-empty
	require.NoError(t, os.WriteFile(filepath.Join(backup1, "file.txt"), []byte("a"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(backup2, "file.txt"), []byte("b"), 0644))

	backups, err := scanBackups(dir)
	require.NoError(t, err)
	assert.Len(t, backups, 2)

	// Should be sorted newest first
	assert.True(t, backups[0].Timestamp.After(backups[1].Timestamp))
	assert.Equal(t, "20240102_130000", filepath.Base(backups[0].Path))
	assert.Equal(t, "20240101_120000", filepath.Base(backups[1].Path))
}

func TestScanBackups_IgnoresInvalidDirs(t *testing.T) {
	dir := t.TempDir()

	// Valid backup with a file
	valid := filepath.Join(dir, "20240101_120000")
	require.NoError(t, os.MkdirAll(valid, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(valid, "file.txt"), []byte("a"), 0644))

	// Invalid directories
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "not-a-backup"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "2024-01-01"), 0755))
	// Regular file (not directory)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "20240103_120000"), []byte("file"), 0644))

	backups, err := scanBackups(dir)
	require.NoError(t, err)
	assert.Len(t, backups, 1)
}

func TestScanBackups_SkipsEmptyBackups(t *testing.T) {
	dir := t.TempDir()

	// Empty backup dir
	empty := filepath.Join(dir, "20240101_120000")
	require.NoError(t, os.MkdirAll(empty, 0755))

	backups, err := scanBackups(dir)
	require.NoError(t, err)
	assert.Empty(t, backups)
}

func TestScanBackups_WithConfigYaml(t *testing.T) {
	dir := t.TempDir()

	backup := filepath.Join(dir, "20240101_120000")
	require.NoError(t, os.MkdirAll(backup, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(backup, "config.yaml"), []byte("test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(backup, "data.txt"), []byte("data"), 0644))

	backups, err := scanBackups(dir)
	require.NoError(t, err)
	require.Len(t, backups, 1)
	assert.True(t, backups[0].HasConfig)
	assert.Equal(t, 1, backups[0].FileCount) // config.yaml excluded from count
}

// ── countFiles tests ────────────────────────────────────────────────────────

func TestCountFiles_Empty(t *testing.T) {
	dir := t.TempDir()
	assert.Equal(t, 0, countFiles(dir, false))
}

func TestCountFiles_WithFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "c.txt"), []byte("c"), 0644))

	assert.Equal(t, 3, countFiles(dir, false))
}

func TestCountFiles_ExcludesConfigYaml(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("cfg"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.txt"), []byte("data"), 0644))

	assert.Equal(t, 1, countFiles(dir, true))  // excludes config.yaml
	assert.Equal(t, 2, countFiles(dir, false)) // counts everything
}

// ── buildRestorePlan tests ──────────────────────────────────────────────────

func TestBuildRestorePlan_NoConfig(t *testing.T) {
	dir := t.TempDir()

	// Create backup structure: backup/home/user/.vimrc
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "home", "user"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "home", "user", ".vimrc"), []byte("set nu"), 0644))

	backup := BackupEntry{
		Path:      dir,
		Timestamp: time.Now(),
		HasConfig: false,
		FileCount: 1,
	}

	items, err := buildRestorePlan(backup, nil)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, config.ModeCopy, items[0].Mode)
	assert.Equal(t, "/home/user/.vimrc", items[0].Destination)
	assert.Equal(t, "(unknown)", items[0].RuleName)
}

func TestBuildRestorePlan_WithConfig(t *testing.T) {
	dir := t.TempDir()

	// Create backup file
	destPath := "/home/user/.vimrc"
	relPath := destPath[1:] // "home/user/.vimrc"
	require.NoError(t, os.MkdirAll(filepath.Join(dir, filepath.Dir(relPath)), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, relPath), []byte("data"), 0644))

	backup := BackupEntry{
		Path:      dir,
		Timestamp: time.Now(),
		HasConfig: true,
		FileCount: 1,
	}

	configFile := &config.ConfigFile{
		Rules: []*config.Rule{
			{
				Name: "vim config",
				Mode: config.ModeSymlink,
				Items: []*config.RuleItem{
					{Source: "/src/.vimrc", Destination: destPath},
				},
			},
		},
	}

	items, err := buildRestorePlan(backup, configFile)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "vim config", items[0].RuleName)
	assert.Equal(t, config.ModeSymlink, items[0].Mode)
	assert.Equal(t, destPath, items[0].Destination)
}

func TestBuildRestorePlan_SubdirectoryMatch(t *testing.T) {
	dir := t.TempDir()

	// Backup has /home/user/.vim/autoload/plug.vim
	relPath := "home/user/.vim/autoload/plug.vim"
	require.NoError(t, os.MkdirAll(filepath.Join(dir, filepath.Dir(relPath)), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, relPath), []byte("plug"), 0644))

	backup := BackupEntry{
		Path:      dir,
		Timestamp: time.Now(),
		FileCount: 1,
	}

	configFile := &config.ConfigFile{
		Rules: []*config.Rule{
			{
				Name: "vim dir",
				Mode: config.ModeCopy,
				Items: []*config.RuleItem{
					{Source: "/src/.vim", Destination: "/home/user/.vim"},
				},
			},
		},
	}

	items, err := buildRestorePlan(backup, configFile)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "vim dir", items[0].RuleName)
	assert.Equal(t, config.ModeCopy, items[0].Mode)
}

func TestBuildRestorePlan_SSHMode(t *testing.T) {
	dir := t.TempDir()

	relPath := "root/.bashrc"
	require.NoError(t, os.MkdirAll(filepath.Join(dir, filepath.Dir(relPath)), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, relPath), []byte("rc"), 0644))

	backup := BackupEntry{Path: dir, Timestamp: time.Now(), FileCount: 1}

	configFile := &config.ConfigFile{
		SSHServers: map[string]*config.SSHServer{
			"srv1": {Host: "1.2.3.4", User: "root", Port: 22},
		},
		Rules: []*config.Rule{
			{
				Name: "remote",
				Mode: config.ModeSSH,
				SSH:  "srv1",
				Items: []*config.RuleItem{
					{Source: "/local/.bashrc", Destination: "/root/.bashrc"},
				},
			},
		},
	}

	items, err := buildRestorePlan(backup, configFile)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, config.ModeSSH, items[0].Mode)
	assert.Equal(t, "srv1", items[0].SSHServer)
}

func TestBuildRestorePlan_UnmatchedFile(t *testing.T) {
	dir := t.TempDir()

	// File that doesn't match any rule
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "etc"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "etc", "hosts"), []byte("data"), 0644))

	backup := BackupEntry{Path: dir, Timestamp: time.Now(), FileCount: 1}

	configFile := &config.ConfigFile{
		Rules: []*config.Rule{
			{
				Name: "vim",
				Mode: config.ModeSymlink,
				Items: []*config.RuleItem{
					{Source: "/src/.vimrc", Destination: "/home/.vimrc"},
				},
			},
		},
	}

	items, err := buildRestorePlan(backup, configFile)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "(unmatched)", items[0].RuleName)
	assert.Equal(t, config.ModeCopy, items[0].Mode)
}

func TestBuildRestorePlan_ExcludesConfigYaml(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("cfg"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.txt"), []byte("data"), 0644))

	backup := BackupEntry{Path: dir, Timestamp: time.Now(), HasConfig: true, FileCount: 1}

	items, err := buildRestorePlan(backup, nil)
	require.NoError(t, err)
	// config.yaml should be excluded
	for _, item := range items {
		assert.NotEqual(t, filepath.Join(dir, "config.yaml"), item.BackupFile)
	}
}

func TestBuildRestorePlan_EmptyBackup(t *testing.T) {
	dir := t.TempDir()

	backup := BackupEntry{Path: dir, Timestamp: time.Now(), FileCount: 0}

	items, err := buildRestorePlan(backup, nil)
	require.NoError(t, err)
	assert.Empty(t, items)
}

// ── filterRestoreItems tests ────────────────────────────────────────────────

func TestFilterRestoreItems_ByName(t *testing.T) {
	sp := &testSpinner{}
	items := []RestoreItem{
		{RuleName: "vim", Destination: "/a"},
		{RuleName: "bash", Destination: "/b"},
		{RuleName: "vim", Destination: "/c"},
	}

	filtered := filterRestoreItems(sp, items, []string{"vim"})
	assert.Len(t, filtered, 2)
	for _, item := range filtered {
		assert.Equal(t, "vim", item.RuleName)
	}
}

func TestFilterRestoreItems_ByIndex(t *testing.T) {
	sp := &testSpinner{}
	items := []RestoreItem{
		{RuleName: "vim", Destination: "/a"},
		{RuleName: "bash", Destination: "/b"},
		{RuleName: "zsh", Destination: "/c"},
	}

	// 1-based index: "2" should match "bash"
	filtered := filterRestoreItems(sp, items, []string{"2"})
	assert.Len(t, filtered, 1)
	assert.Equal(t, "bash", filtered[0].RuleName)
}

func TestFilterRestoreItems_CaseInsensitive(t *testing.T) {
	sp := &testSpinner{}
	items := []RestoreItem{
		{RuleName: "Vim Config", Destination: "/a"},
		{RuleName: "bash", Destination: "/b"},
	}

	filtered := filterRestoreItems(sp, items, []string{"vim config"})
	assert.Len(t, filtered, 1)
}

func TestFilterRestoreItems_NoMatch(t *testing.T) {
	sp := &testSpinner{}
	items := []RestoreItem{
		{RuleName: "vim", Destination: "/a"},
	}

	filtered := filterRestoreItems(sp, items, []string{"nonexistent"})
	assert.Empty(t, filtered)
}

// ── restoreLocal tests ──────────────────────────────────────────────────────

func TestRestoreLocal_File(t *testing.T) {
	dir := t.TempDir()

	backupFile := filepath.Join(dir, "backup.txt")
	require.NoError(t, os.WriteFile(backupFile, []byte("restored data"), 0644))

	destFile := filepath.Join(dir, "restored", "file.txt")

	item := RestoreItem{
		BackupFile:  backupFile,
		Destination: destFile,
		Mode:        config.ModeCopy,
	}

	err := restoreLocal(item)
	require.NoError(t, err)

	got, err := os.ReadFile(destFile)
	require.NoError(t, err)
	assert.Equal(t, []byte("restored data"), got)
}

func TestRestoreLocal_OverwriteExisting(t *testing.T) {
	dir := t.TempDir()

	backupFile := filepath.Join(dir, "backup.txt")
	require.NoError(t, os.WriteFile(backupFile, []byte("new"), 0644))

	destFile := filepath.Join(dir, "dest.txt")
	require.NoError(t, os.WriteFile(destFile, []byte("old"), 0644))

	item := RestoreItem{
		BackupFile:  backupFile,
		Destination: destFile,
		Mode:        config.ModeCopy,
	}

	err := restoreLocal(item)
	require.NoError(t, err)

	got, err := os.ReadFile(destFile)
	require.NoError(t, err)
	assert.Equal(t, []byte("new"), got)
}

func TestRestoreLocal_OverwriteSymlink(t *testing.T) {
	dir := t.TempDir()

	backupFile := filepath.Join(dir, "backup.txt")
	require.NoError(t, os.WriteFile(backupFile, []byte("restored"), 0644))

	destFile := filepath.Join(dir, "link.txt")
	require.NoError(t, os.Symlink("/some/target", destFile))

	item := RestoreItem{
		BackupFile:  backupFile,
		Destination: destFile,
		Mode:        config.ModeCopy,
	}

	err := restoreLocal(item)
	require.NoError(t, err)

	// Should be a regular file now, not a symlink
	info, err := os.Lstat(destFile)
	require.NoError(t, err)
	assert.False(t, info.Mode()&os.ModeSymlink != 0)

	got, err := os.ReadFile(destFile)
	require.NoError(t, err)
	assert.Equal(t, []byte("restored"), got)
}

// ── backupDirPattern tests ──────────────────────────────────────────────────

func TestBackupDirPattern(t *testing.T) {
	tests := []struct {
		input   string
		matches bool
	}{
		{"20240101_120000", true},
		{"20231231_235959", true},
		{"2024-01-01", false},
		{"backup", false},
		{"20240101", false},
		{"20240101_12000", false},  // 5 digits in time
		{"20240101_1200000", false}, // 7 digits in time
		{"", false},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.matches, backupDirPattern.MatchString(tt.input), "input: %q", tt.input)
	}
}
