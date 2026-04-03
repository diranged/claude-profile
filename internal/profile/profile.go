// Package profile manages Claude Code authentication profiles.
//
// Each profile is a named directory under ~/.claude-profiles/<name>/ containing
// a config/ subdirectory used as CLAUDE_CONFIG_DIR. When CLAUDE_CONFIG_DIR is set,
// Claude Code automatically appends a SHA-256 hash of the directory path to its
// macOS keychain service name, giving each profile its own isolated credentials.
//
// This means profile isolation requires zero hacks — it leverages Claude Code's
// built-in behavior where the keychain service name is:
//
//	Default:              "Claude Code-credentials"
//	With CLAUDE_CONFIG_DIR: "Claude Code-credentials-<sha256[:8]>"
package profile

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

const keychainServicePrefix = "Claude Code-credentials"

// Profile represents a named Claude Code authentication profile.
type Profile struct {
	Name       string
	Dir        string // root profile dir: <profiles_dir>/<name>
	ConfigDir  string // CLAUDE_CONFIG_DIR value: <profiles_dir>/<name>/config
	ServiceKey string // macOS keychain service name (hash-suffixed)
}

// OAuthInfo holds displayable info extracted from stored credentials.
// Field types must match the actual JSON: scopes is an array, expiresAt
// is a numeric epoch timestamp in milliseconds.
type OAuthInfo struct {
	SubscriptionType string   `json:"subscriptionType"`
	Scopes           []string `json:"scopes"`
	ExpiresAt        int64    `json:"expiresAt"`
	RateLimitTier    string   `json:"rateLimitTier"`
}

// ProfilesDir returns the base directory for all profiles.
// Override with CLAUDE_PROFILES_DIR env var.
func ProfilesDir() string {
	if d := os.Getenv("CLAUDE_PROFILES_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude-profiles")
}

// Load returns a Profile for the given name. The profile may not exist yet.
func Load(name string) *Profile {
	dir := filepath.Join(ProfilesDir(), name)
	configDir := filepath.Join(dir, "config")
	return &Profile{
		Name:       name,
		Dir:        dir,
		ConfigDir:  configDir,
		ServiceKey: keychainService(configDir),
	}
}

// Exists returns true if the profile config directory exists on disk.
func (p *Profile) Exists() bool {
	_, err := os.Stat(p.ConfigDir)
	return err == nil
}

// EnsureDir creates the profile config directory if it doesn't exist.
func (p *Profile) EnsureDir() error {
	return os.MkdirAll(p.ConfigDir, 0700)
}

// AuthStatus returns where credentials are stored for this profile:
// "keychain", "file", or "none".
func (p *Profile) AuthStatus() string {
	if p.hasKeychainEntry() {
		return "keychain"
	}
	credFile := filepath.Join(p.ConfigDir, ".credentials.json")
	if _, err := os.Stat(credFile); err == nil {
		return "file"
	}
	return "none"
}

// OAuthDetails reads credential metadata from keychain or fallback file.
// Returns subscription type, scopes, expiry, etc. for display purposes.
func (p *Profile) OAuthDetails() (*OAuthInfo, error) {
	var raw string

	if p.hasKeychainEntry() {
		out, err := exec.Command("security", "find-generic-password",
			"-s", p.ServiceKey, "-a", currentUser(), "-w").Output()
		if err != nil {
			return nil, fmt.Errorf("reading keychain: %w", err)
		}
		raw = strings.TrimSpace(string(out))
	} else {
		credFile := filepath.Join(p.ConfigDir, ".credentials.json")
		data, err := os.ReadFile(credFile)
		if err != nil {
			return nil, fmt.Errorf("reading credentials file: %w", err)
		}
		raw = string(data)
	}

	var creds struct {
		ClaudeAiOauth OAuthInfo `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal([]byte(raw), &creds); err != nil {
		return nil, fmt.Errorf("parsing credentials: %w", err)
	}
	return &creds.ClaudeAiOauth, nil
}

// Delete removes the profile directory and its keychain entry.
func (p *Profile) Delete() error {
	// Remove keychain entry (ignore errors — may not exist)
	_ = exec.Command("security", "delete-generic-password",
		"-s", p.ServiceKey, "-a", currentUser()).Run()
	return os.RemoveAll(p.Dir)
}

// List returns all profile names found in the profiles directory.
func List() ([]string, error) {
	dir := ProfilesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

func (p *Profile) hasKeychainEntry() bool {
	err := exec.Command("security", "find-generic-password",
		"-s", p.ServiceKey, "-a", currentUser()).Run()
	return err == nil
}

// keychainService computes the same service name that Claude Code uses
// internally when CLAUDE_CONFIG_DIR is set. The format is:
//
//	"Claude Code-credentials-<first 8 hex chars of SHA-256 of config dir>"
//
// This must stay in sync with Claude Code's V51() function in cli.js.
func keychainService(configDir string) string {
	h := sha256.Sum256([]byte(configDir))
	return fmt.Sprintf("%s-%x", keychainServicePrefix, h[:4])
}

func currentUser() string {
	u, err := user.Current()
	if err != nil {
		return os.Getenv("USER")
	}
	return u.Username
}
