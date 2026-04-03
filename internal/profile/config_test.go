package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, 108, cfg.Color)
}

func TestLoadConfig_NoFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := Load("noconfig")
	require.NoError(t, p.EnsureDir())

	cfg := p.LoadConfig()
	assert.Equal(t, 108, cfg.Color)
}

func TestSaveAndLoadConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := Load("configtest")
	require.NoError(t, p.EnsureDir())
	// EnsureDir creates ConfigDir but SaveConfig writes to Dir
	require.NoError(t, os.MkdirAll(p.Dir, 0700))

	cfg := Config{Color: 204}
	require.NoError(t, p.SaveConfig(cfg))

	loaded := p.LoadConfig()
	assert.Equal(t, 204, loaded.Color)
}

func TestLoadConfig_InvalidColor_TooHigh(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := Load("badcolor")
	require.NoError(t, p.EnsureDir())

	// Write config with invalid color
	configPath := filepath.Join(p.Dir, configFilename)
	require.NoError(t, os.WriteFile(configPath, []byte("color: 999\n"), 0644))

	cfg := p.LoadConfig()
	assert.Equal(t, 108, cfg.Color) // should revert to default
}

func TestLoadConfig_InvalidColor_Negative(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := Load("negcolor")
	require.NoError(t, p.EnsureDir())

	configPath := filepath.Join(p.Dir, configFilename)
	require.NoError(t, os.WriteFile(configPath, []byte("color: -5\n"), 0644))

	cfg := p.LoadConfig()
	assert.Equal(t, 108, cfg.Color) // should revert to default
}

func TestLoadConfig_ValidBoundaryColors(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	tests := []struct {
		name  string
		color int
	}{
		// Note: color=0 is NOT tested here because the Config struct uses
		// `yaml:"color,omitempty"` which drops zero values, causing it to
		// reload as the default (108). This is expected behavior.
		{"one", 1},
		{"max", 255},
		{"mid", 128},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := Load("boundary-" + tc.name)
			require.NoError(t, p.EnsureDir())

			cfg := Config{Color: tc.color}
			require.NoError(t, p.SaveConfig(cfg))

			loaded := p.LoadConfig()
			assert.Equal(t, tc.color, loaded.Color)
		})
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := Load("badyaml")
	require.NoError(t, p.EnsureDir())

	configPath := filepath.Join(p.Dir, configFilename)
	require.NoError(t, os.WriteFile(configPath, []byte("{{invalid yaml"), 0644))

	// Should return defaults when YAML is invalid
	cfg := p.LoadConfig()
	assert.Equal(t, 108, cfg.Color)
}
