package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/diranged/claude-profile-go/internal/profile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureStatusline_NoExistingSettings(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := profile.Load("sltest")
	require.NoError(t, p.EnsureDir())

	err := configureStatusline(p)
	require.NoError(t, err)

	// Read the settings file
	settingsPath := filepath.Join(p.ConfigDir, "settings.json")
	data, err := os.ReadFile(settingsPath)
	require.NoError(t, err)

	var settings map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &settings))

	sl, ok := settings["statusLine"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "command", sl["type"])
	assert.Contains(t, sl["command"], "statusline")
	assert.Equal(t, float64(0), sl["padding"])
}

func TestConfigureStatusline_ExistingSettings(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := profile.Load("slexisting")
	require.NoError(t, p.EnsureDir())

	// Write existing settings with other keys
	settingsPath := filepath.Join(p.ConfigDir, "settings.json")
	existing := map[string]interface{}{
		"someOtherKey": "value",
	}
	data, _ := json.Marshal(existing)
	require.NoError(t, os.WriteFile(settingsPath, data, 0644))

	err := configureStatusline(p)
	require.NoError(t, err)

	data, err = os.ReadFile(settingsPath)
	require.NoError(t, err)

	var settings map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &settings))

	// Other keys should be preserved
	assert.Equal(t, "value", settings["someOtherKey"])
	// Statusline should be set
	sl, ok := settings["statusLine"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "command", sl["type"])
}

func TestConfigureStatusline_WrapExistingStatusline(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := profile.Load("slwrap")
	require.NoError(t, p.EnsureDir())

	// Write existing settings with a statusline command
	settingsPath := filepath.Join(p.ConfigDir, "settings.json")
	existing := map[string]interface{}{
		"statusLine": map[string]interface{}{
			"type":    "command",
			"command": "bunx ccstatusline@latest",
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	require.NoError(t, os.WriteFile(settingsPath, data, 0644))

	err := configureStatusline(p)
	require.NoError(t, err)

	data, err = os.ReadFile(settingsPath)
	require.NoError(t, err)

	var settings map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &settings))

	sl, ok := settings["statusLine"].(map[string]interface{})
	require.True(t, ok)

	cmd, ok := sl["command"].(string)
	require.True(t, ok)
	// Should wrap the original command
	assert.Contains(t, cmd, "statusline -- bunx ccstatusline@latest")
}

func TestConfigureStatusline_InvalidJSON(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := profile.Load("slbadjson")
	require.NoError(t, p.EnsureDir())

	// Write invalid JSON
	settingsPath := filepath.Join(p.ConfigDir, "settings.json")
	require.NoError(t, os.WriteFile(settingsPath, []byte("not json"), 0644))

	// Should still succeed — it treats invalid JSON as empty settings
	err := configureStatusline(p)
	require.NoError(t, err)

	data, err := os.ReadFile(settingsPath)
	require.NoError(t, err)

	var settings map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &settings))
	assert.NotNil(t, settings["statusLine"])
}
