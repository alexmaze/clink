package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunPrintsUsageWithoutCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"clink"}, &stdout, &stderr)
	require.Error(t, err)
	assert.True(t, IsSilent(err))
	assert.Contains(t, stdout.String(), "Usage:")
	assert.Empty(t, stderr.String())
}

func TestRunPrintsUsageForUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"clink", "unknown"}, &stdout, &stderr)
	require.Error(t, err)
	assert.True(t, IsSilent(err))
	assert.Contains(t, stdout.String(), "clink add [flags] <source>")
	assert.Empty(t, stderr.String())
}

func TestRunPrintsVersion(t *testing.T) {
	oldVersion := Version
	Version = "1.2.3"
	defer func() { Version = oldVersion }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"clink", "version"}, &stdout, &stderr)
	require.NoError(t, err)
	assert.Equal(t, "1.2.3\n", stdout.String())
	assert.Empty(t, stderr.String())
}
