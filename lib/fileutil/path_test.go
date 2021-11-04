package fileutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePath(t *testing.T) {

	var (
		err  error
		path string
	)

	// abs
	path, err = ParsePath("", "/xxx/xx")
	assert.Nil(t, err)
	assert.Equal(t, "/xxx/xx", path)

	// rel 1 normal
	path, err = ParsePath("/var", "xxx/xx")
	assert.Nil(t, err)
	assert.Equal(t, "/var/xxx/xx", path)

	// rel 2 error
	_, err = ParsePath("var", "xxx/xx")
	assert.NotNil(t, err)

	// rel 3 .
	path, err = ParsePath("/var", "./xxx")
	assert.Nil(t, err)
	assert.Equal(t, "/var/xxx", path)

	// rel home
	path, err = ParsePath("", "~/xxx")
	assert.Nil(t, err)
	homeDir, err := os.UserHomeDir()
	assert.Nil(t, err)
	assert.True(t, strings.HasPrefix(path, homeDir))
	assert.Equal(t, filepath.Join(homeDir, "xxx"), path)
}
