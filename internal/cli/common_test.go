package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindBackupsOnlyReturnsManifestDirectories(t *testing.T) {
	base := t.TempDir()
	backup1 := filepath.Join(base, "20260406_100000")
	backup2 := filepath.Join(base, "20260406_090000")
	require.NoError(t, os.MkdirAll(backup1, 0755))
	require.NoError(t, os.MkdirAll(backup2, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(backup1, "manifest.json"), []byte("{}"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(backup2, "manifest.json"), []byte("{}"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(base, "random.txt"), []byte("x"), 0644))

	backups, err := findBackups(base)
	require.NoError(t, err)
	require.Len(t, backups, 2)
	assert.Equal(t, backup1, backups[0])
	assert.Equal(t, backup2, backups[1])
}

func TestSlugNormalizesWhitespaceAndSymbols(t *testing.T) {
	assert.Equal(t, "hello-world", slug(" Hello, World! "))
	assert.Equal(t, "rule", slug("###"))
}

func TestResolveLocalPathUsesCWDForRelativePath(t *testing.T) {
	cwd := t.TempDir()
	configDir := t.TempDir()

	path, err := resolveLocalPath(cwd, configDir, "./dotfiles/.zshrc")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(cwd, "dotfiles", ".zshrc"), path)
}
