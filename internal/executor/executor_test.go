package executor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexmaze/clink/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyPathPreservesSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	link := filepath.Join(dir, "source-link")
	dest := filepath.Join(dir, "copied-link")

	require.NoError(t, os.WriteFile(target, []byte("payload"), 0644))
	require.NoError(t, os.Symlink(target, link))

	require.NoError(t, copyPath(link, dest))

	info, err := os.Lstat(dest)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0)
	readTarget, err := os.Readlink(dest)
	require.NoError(t, err)
	assert.Equal(t, target, readTarget)
}

func TestCheckCopyDetectsHashDrift(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dest := filepath.Join(dir, "dest.txt")
	require.NoError(t, os.WriteFile(src, []byte("hello"), 0644))
	require.NoError(t, os.WriteFile(dest, []byte("world"), 0644))

	status, checkStatus, detail := checkCopy(domain.Action{
		Source:      src,
		Destination: dest,
	})

	assert.Equal(t, "FAILED", status)
	assert.Equal(t, domain.CheckStatusDrifted, checkStatus)
	assert.Contains(t, detail, "hash mismatch")
}

func TestRestoreLocalReplacesDestinationAtomically(t *testing.T) {
	dir := t.TempDir()
	backupFile := filepath.Join(dir, "backup.txt")
	dest := filepath.Join(dir, "dest.txt")
	require.NoError(t, os.WriteFile(backupFile, []byte("new"), 0644))
	require.NoError(t, os.WriteFile(dest, []byte("old"), 0644))

	require.NoError(t, restoreLocal(domain.Action{
		Source:      backupFile,
		Destination: dest,
	}))

	body, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, []byte("new"), body)
}

func TestBackupLocalWritesManifestEntry(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.txt")
	dest := filepath.Join(dir, "dest.txt")
	backup := filepath.Join(dir, "backup", "dest.txt")
	require.NoError(t, os.WriteFile(source, []byte("src"), 0644))
	require.NoError(t, os.WriteFile(dest, []byte("dest"), 0644))

	exec := New(&domain.Config{ConfigPath: filepath.Join(dir, "config.yaml")}, false)
	status, detail, entry, err := exec.backupLocal(domain.Action{
		RuleName:    "shell",
		Mode:        domain.ModeCopy,
		Source:      source,
		Destination: dest,
		BackupPath:  backup,
		PathKind:    domain.PathKindFile,
	})
	require.NoError(t, err)
	assert.Equal(t, "OK", status)
	assert.Equal(t, "backup captured", detail)
	require.NotNil(t, entry)
	assert.NotEmpty(t, entry.SHA256)

	body, err := os.ReadFile(backup)
	require.NoError(t, err)
	assert.Equal(t, []byte("dest"), body)
}

func TestWriteManifestPersistsJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	manifest := &domain.BackupManifest{
		Version:    1,
		Command:    "apply",
		ConfigPath: "/tmp/config.yaml",
		Entries: []domain.BackupEntry{
			{RuleName: "shell", Destination: "/tmp/.zshrc"},
		},
	}

	require.NoError(t, writeManifest(path, manifest))

	body, err := os.ReadFile(path)
	require.NoError(t, err)
	var decoded domain.BackupManifest
	require.NoError(t, json.Unmarshal(body, &decoded))
	assert.Equal(t, "apply", decoded.Command)
	assert.Len(t, decoded.Entries, 1)
}
