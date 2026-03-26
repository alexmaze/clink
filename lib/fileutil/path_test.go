package fileutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── ParsePath tests ─────────────────────────────────────────────────────────

func TestParsePath_Absolute(t *testing.T) {
	path, err := ParsePath("", "/xxx/xx")
	assert.Nil(t, err)
	assert.Equal(t, "/xxx/xx", path)
}

func TestParsePath_RelativeNormal(t *testing.T) {
	path, err := ParsePath("/var", "xxx/xx")
	assert.Nil(t, err)
	assert.Equal(t, "/var/xxx/xx", path)
}

func TestParsePath_RelativeBaseNotAbsolute(t *testing.T) {
	_, err := ParsePath("var", "xxx/xx")
	assert.NotNil(t, err)
}

func TestParsePath_RelativeDot(t *testing.T) {
	path, err := ParsePath("/var", "./xxx")
	assert.Nil(t, err)
	assert.Equal(t, "/var/xxx", path)
}

func TestParsePath_HomeDir(t *testing.T) {
	path, err := ParsePath("", "~/xxx")
	assert.Nil(t, err)
	homeDir, err := os.UserHomeDir()
	assert.Nil(t, err)
	assert.True(t, strings.HasPrefix(path, homeDir))
	assert.Equal(t, filepath.Join(homeDir, "xxx"), path)
}

func TestParsePath_HomeDirNested(t *testing.T) {
	path, err := ParsePath("", "~/a/b/c")
	assert.NoError(t, err)
	homeDir, _ := os.UserHomeDir()
	assert.Equal(t, filepath.Join(homeDir, "a", "b", "c"), path)
}

func TestParsePath_EmptyRelativePath(t *testing.T) {
	path, err := ParsePath("/base", "")
	assert.NoError(t, err)
	assert.Equal(t, "/base", path)
}

func TestParsePath_DotDotRelative(t *testing.T) {
	path, err := ParsePath("/var/lib", "../xxx")
	assert.NoError(t, err)
	assert.Equal(t, "/var/xxx", path)
}

// ── IsFileExists tests ──────────────────────────────────────────────────────

func TestIsFileExists_FileExists(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(file, []byte("data"), 0644))

	exists, err := IsFileExists(file)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestIsFileExists_FileNotExists(t *testing.T) {
	exists, err := IsFileExists("/nonexistent/path/file.txt")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestIsFileExists_DirectoryExists(t *testing.T) {
	dir := t.TempDir()
	exists, err := IsFileExists(dir)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestIsFileExists_Symlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	require.NoError(t, os.WriteFile(target, []byte("data"), 0644))

	link := filepath.Join(dir, "link.txt")
	require.NoError(t, os.Symlink(target, link))

	exists, err := IsFileExists(link)
	assert.NoError(t, err)
	assert.True(t, exists)
}

// ── GetPathType tests ───────────────────────────────────────────────────────

func TestGetPathType_File(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(file, []byte("data"), 0644))

	pt, err := GetPathType(file)
	assert.NoError(t, err)
	assert.Equal(t, PathTypeFile, pt)
}

func TestGetPathType_Directory(t *testing.T) {
	dir := t.TempDir()

	pt, err := GetPathType(dir)
	assert.NoError(t, err)
	assert.Equal(t, PathTypeFolder, pt)
}

func TestGetPathType_NonexistentPath(t *testing.T) {
	_, err := GetPathType("/nonexistent/path")
	assert.Error(t, err)
}

func TestGetPathType_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "empty.txt")
	require.NoError(t, os.WriteFile(file, []byte{}, 0644))

	pt, err := GetPathType(file)
	assert.NoError(t, err)
	assert.Equal(t, PathTypeFile, pt)
}

// ── PathType constants tests ────────────────────────────────────────────────

func TestPathTypeConstants(t *testing.T) {
	assert.Equal(t, PathType("file"), PathTypeFile)
	assert.Equal(t, PathType("folder"), PathTypeFolder)
}
