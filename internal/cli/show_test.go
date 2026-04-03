package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShowCmd_ProfileNotExist(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	initLogger()
	cmd := newShowCmd()
	cmd.SetArgs([]string{"nonexistent"})
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestShowCmd_ProfileExists(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	// Create the profile
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "work", "config"), 0700))

	initLogger()
	cmd := newShowCmd()
	cmd.SetArgs([]string{"work"})
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "PROFILE: work")
	assert.Contains(t, output, "Config dir")
	assert.Contains(t, output, "Keychain service")
	assert.Contains(t, output, "Auth")
}

func TestShowCmd_WithCredentials(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	// Create profile with credentials file
	configDir := filepath.Join(tmp, "credprofile", "config")
	require.NoError(t, os.MkdirAll(configDir, 0700))

	creds := `{"claudeAiOauth":{"subscriptionType":"pro","scopes":["user:read"],"expiresAt":1700000000000,"rateLimitTier":"tier1"}}`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, ".credentials.json"), []byte(creds), 0600))

	initLogger()
	cmd := newShowCmd()
	cmd.SetArgs([]string{"credprofile"})
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Subscription")
	assert.Contains(t, output, "pro")
	assert.Contains(t, output, "Rate limit tier")
	assert.Contains(t, output, "tier1")
	assert.Contains(t, output, "Expires")
	assert.Contains(t, output, "Scopes")
	assert.Contains(t, output, "user:read")
}

func TestShowCmd_NoArgs(t *testing.T) {
	initLogger()
	cmd := newShowCmd()
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	assert.Error(t, err)
}
