package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCmd_NoProfiles(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	initLogger()
	cmd := newListCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No profiles found")
}

func TestListCmd_WithCredentialedProfiles(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	// Create profile with credentials file
	configDir := filepath.Join(tmp, "work", "config")
	require.NoError(t, os.MkdirAll(configDir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, ".credentials.json"), []byte(`{}`), 0600))

	// Create profile without credentials
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "personal", "config"), 0700))

	initLogger()
	cmd := newListCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "work")
	assert.Contains(t, output, "personal")
}

func TestListCmd_WithProfiles(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	// Create profile directories with config subdirs
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "work", "config"), 0700))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "personal", "config"), 0700))

	initLogger()
	cmd := newListCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "work")
	assert.Contains(t, output, "personal")
}
