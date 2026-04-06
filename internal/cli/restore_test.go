package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadManifestRejectsInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	require.NoError(t, os.WriteFile(path, []byte("{"), 0644))

	_, err := readManifest(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected end")
}
