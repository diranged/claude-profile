package profile

import (
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// Config holds user-customizable settings for a profile.
// Stored in <profile-dir>/claude-profile.yaml.
type Config struct {
	// Color is the ANSI 256-color code used for the banner and statusline.
	// Default: 108 (muted sage green). Examples: 204 (pink), 33 (blue), 208 (orange).
	Color int `yaml:"color,omitempty"`
}

// configFilename is the name of the YAML file storing per-profile settings.
// It lives in the profile's root directory (not the config/ subdirectory),
// keeping it separate from Claude Code's own configuration files.
const configFilename = "claude-profile.yaml"

// DefaultConfig returns the default profile configuration.
func DefaultConfig() Config {
	return Config{Color: 108}
}

// LoadConfig reads the profile's claude-profile.yaml.
// Returns defaults if the file doesn't exist.
func (p *Profile) LoadConfig() Config {
	cfg := DefaultConfig()

	data, err := os.ReadFile(filepath.Join(p.Dir, configFilename))
	if err != nil {
		return cfg
	}

	_ = yaml.Unmarshal(data, &cfg)

	// Ensure valid color
	if cfg.Color < 0 || cfg.Color > 255 {
		cfg.Color = DefaultConfig().Color
	}

	return cfg
}

// SaveConfig writes the profile's claude-profile.yaml.
func (p *Profile) SaveConfig(cfg Config) error {
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(p.Dir, configFilename), data, 0644)
}
