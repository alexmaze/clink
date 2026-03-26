package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexmaze/clink/config"
	"github.com/alexmaze/clink/lib/color"
	"github.com/alexmaze/clink/lib/icon"
	"github.com/alexmaze/clink/lib/spinner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSpinner is a no-op implementation of spinner.Spinner for testing.
type testSpinner struct{}

func (s *testSpinner) Start(msg ...string) spinner.Spinner                                       { return s }
func (s *testSpinner) Stop()                                                                     {}
func (s *testSpinner) CheckPoint(_ icon.Icon, _ color.Color, _ string, _ color.Color)            {}
func (s *testSpinner) Success(_ string)                                                          {}
func (s *testSpinner) Failed(_ string)                                                           {}
func (s *testSpinner) Successf(_ string, _ ...interface{})                                       {}
func (s *testSpinner) Failedf(_ string, _ ...interface{})                                        {}
func (s *testSpinner) SetSpinGap(_ time.Duration) spinner.Spinner                                { return s }
func (s *testSpinner) SetIconFrames(_ []string) spinner.Spinner                                  { return s }
func (s *testSpinner) SetIconColor(_ color.Color) spinner.Spinner                                { return s }
func (s *testSpinner) SetMsg(_ string) spinner.Spinner                                           { return s }
func (s *testSpinner) SetMsgColor(_ color.Color) spinner.Spinner                                 { return s }

// ── modeActionLabel tests ───────────────────────────────────────────────────

func TestModeActionLabel(t *testing.T) {
	tests := []struct {
		mode     config.Mode
		expected string
	}{
		{config.ModeCopy, "copy  "},
		{config.ModeSSH, "upload"},
		{config.ModeSymlink, "link  "},
		{"", "link  "},       // default
		{"unknown", "link  "}, // unknown falls to default
	}

	for _, tt := range tests {
		result := modeActionLabel(tt.mode)
		assert.Equal(t, tt.expected, result, "mode: %q", tt.mode)
	}
}

// ── destExists tests ────────────────────────────────────────────────────────

func TestDestExists_FileExists(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "exists.txt")
	require.NoError(t, os.WriteFile(file, []byte("test"), 0644))

	exists, err := destExists(file)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestDestExists_FileNotExists(t *testing.T) {
	exists, err := destExists("/nonexistent/path/file.txt")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestDestExists_Symlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	require.NoError(t, os.WriteFile(target, []byte("target"), 0644))
	link := filepath.Join(dir, "link.txt")
	require.NoError(t, os.Symlink(target, link))

	exists, err := destExists(link)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestDestExists_BrokenSymlink(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "broken-link")
	require.NoError(t, os.Symlink("/nonexistent/target", link))

	exists, err := destExists(link)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestDestExists_Directory(t *testing.T) {
	dir := t.TempDir()
	exists, err := destExists(dir)
	assert.NoError(t, err)
	assert.True(t, exists)
}

// ── copyFile tests ──────────────────────────────────────────────────────────

func TestCopyFile_Basic(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	content := []byte("hello world\nline 2\n")
	require.NoError(t, os.WriteFile(src, content, 0644))

	err := copyFile(src, dst)
	require.NoError(t, err)

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, content, got)
}

func TestCopyFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "empty.txt")
	dst := filepath.Join(dir, "dst-empty.txt")

	require.NoError(t, os.WriteFile(src, []byte{}, 0644))

	err := copyFile(src, dst)
	require.NoError(t, err)

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestCopyFile_LargeFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "large.bin")
	dst := filepath.Join(dir, "large-dst.bin")

	data := make([]byte, 1<<20) // 1MB
	for i := range data {
		data[i] = byte(i % 256)
	}
	require.NoError(t, os.WriteFile(src, data, 0644))

	err := copyFile(src, dst)
	require.NoError(t, err)

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestCopyFile_SourceNotExists(t *testing.T) {
	dir := t.TempDir()
	err := copyFile("/nonexistent/file", filepath.Join(dir, "dst"))
	assert.Error(t, err)
}

func TestCopyFile_DestDirNotExists(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	require.NoError(t, os.WriteFile(src, []byte("data"), 0644))

	dst := filepath.Join(dir, "nonexistent", "subdir", "dst.txt")
	err := copyFile(src, dst)
	assert.Error(t, err)
}

func TestCopyFile_OverwriteExisting(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	require.NoError(t, os.WriteFile(src, []byte("new content"), 0644))
	require.NoError(t, os.WriteFile(dst, []byte("old content"), 0644))

	err := copyFile(src, dst)
	require.NoError(t, err)

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, []byte("new content"), got)
}

// ── copyPath tests ──────────────────────────────────────────────────────────

func TestCopyPath_SingleFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "file.txt")
	dst := filepath.Join(dir, "copy.txt")

	require.NoError(t, os.WriteFile(src, []byte("content"), 0644))

	err := copyPath(src, dst)
	require.NoError(t, err)

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, []byte("content"), got)
}

func TestCopyPath_Directory(t *testing.T) {
	dir := t.TempDir()

	srcDir := filepath.Join(dir, "src")
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "sub"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("aaa"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "sub", "b.txt"), []byte("bbb"), 0644))

	dstDir := filepath.Join(dir, "dst")

	err := copyPath(srcDir, dstDir)
	require.NoError(t, err)

	gotA, err := os.ReadFile(filepath.Join(dstDir, "a.txt"))
	require.NoError(t, err)
	assert.Equal(t, []byte("aaa"), gotA)

	gotB, err := os.ReadFile(filepath.Join(dstDir, "sub", "b.txt"))
	require.NoError(t, err)
	assert.Equal(t, []byte("bbb"), gotB)
}

func TestCopyPath_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "empty-src")
	require.NoError(t, os.MkdirAll(srcDir, 0755))

	dstDir := filepath.Join(dir, "empty-dst")
	err := copyPath(srcDir, dstDir)
	require.NoError(t, err)

	info, err := os.Stat(dstDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestCopyPath_NestedDirectories(t *testing.T) {
	dir := t.TempDir()

	srcDir := filepath.Join(dir, "src")
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "a", "b", "c"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "a", "b", "c", "deep.txt"), []byte("deep"), 0644))

	dstDir := filepath.Join(dir, "dst")
	err := copyPath(srcDir, dstDir)
	require.NoError(t, err)

	got, err := os.ReadFile(filepath.Join(dstDir, "a", "b", "c", "deep.txt"))
	require.NoError(t, err)
	assert.Equal(t, []byte("deep"), got)
}

// ── snapshotConfig tests ────────────────────────────────────────────────────

func TestSnapshotConfig_DryRunSkips(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		DryRun:     true,
		ConfigPath: "/some/config.yaml",
		BackupPath: dir,
		ConfigFile: &config.ConfigFile{},
	}

	sp := &testSpinner{}
	snapshotConfig(sp, cfg)

	_, err := os.Stat(filepath.Join(dir, "config.yaml"))
	assert.True(t, os.IsNotExist(err))
}

func TestSnapshotConfig_EmptyPaths(t *testing.T) {
	cfg := &config.Config{
		DryRun:     false,
		ConfigPath: "",
		BackupPath: "",
		ConfigFile: &config.ConfigFile{},
	}

	sp := &testSpinner{}
	snapshotConfig(sp, cfg) // should not panic
}

func TestSnapshotConfig_CopiesConfig(t *testing.T) {
	dir := t.TempDir()
	srcConfig := filepath.Join(dir, "source-config.yaml")
	require.NoError(t, os.WriteFile(srcConfig, []byte("test: data"), 0644))

	backupDir := filepath.Join(dir, "backup")
	require.NoError(t, os.MkdirAll(backupDir, 0755))

	cfg := &config.Config{
		DryRun:     false,
		ConfigPath: srcConfig,
		BackupPath: backupDir,
		ConfigFile: &config.ConfigFile{},
	}

	sp := &testSpinner{}
	snapshotConfig(sp, cfg)

	got, err := os.ReadFile(filepath.Join(backupDir, "config.yaml"))
	require.NoError(t, err)
	assert.Equal(t, []byte("test: data"), got)
}
