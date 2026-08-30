// Package profile manages Claude Code authentication profiles.
//
// Each profile is a named directory under ~/.claude-profiles/<name>/ containing
// a config/ subdirectory used as CLAUDE_CONFIG_DIR. When CLAUDE_CONFIG_DIR is set,
// Claude Code automatically derives a per-profile credential location: on macOS
// this is the keychain service name "Claude Code-credentials-<sha256[:8]>", and
// on Linux this is the .credentials.json file inside CLAUDE_CONFIG_DIR.
//
// Profile isolation requires zero hacks — it leverages Claude Code's built-in
// per-CLAUDE_CONFIG_DIR behavior on both platforms.
//
// Keychain integration (macOS): the keychain service name is:
//
//	Default:              "Claude Code-credentials"
//	With CLAUDE_CONFIG_DIR: "Claude Code-credentials-<sha256[:8]>"
//
// File integration (Linux and other non-darwin builds): credentials live at
// <CLAUDE_CONFIG_DIR>/.credentials.json (mode 0600), written by Claude Code
// itself when no platform keychain is available. The keychainHas/keychainRead/
// keychainDelete helpers are stubs on non-darwin builds, so AuthStatus() and
// related callers fall through to the file path automatically.
package profile

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// keychainServicePrefix is the base macOS keychain service name used by Claude
// Code. When CLAUDE_CONFIG_DIR is set, Claude appends a SHA-256 hash suffix to
// this prefix to produce a unique keychain entry per config directory.
// Only consulted on darwin builds; computed unconditionally because it's cheap.
const keychainServicePrefix = "Claude Code-credentials"

// Profile represents a named Claude Code authentication profile. Each profile
// maps to an isolated directory tree and (on macOS) a unique keychain entry,
// enabling concurrent sessions across different Claude subscriptions.
type Profile struct {
	// Name is the user-chosen profile identifier (e.g., "work", "personal").
	Name string

	// Dir is the root profile directory: <profiles_dir>/<name>.
	// Contains claude-profile.yaml and the config/ subdirectory.
	Dir string

	// ConfigDir is the value assigned to CLAUDE_CONFIG_DIR:
	// <profiles_dir>/<name>/config. Claude Code stores its settings,
	// credentials file, and session data here.
	ConfigDir string

	// ServiceKey is the macOS keychain service name for this profile's
	// OAuth credentials. It is derived by appending the first 8 hex chars
	// of SHA-256(ConfigDir) to keychainServicePrefix, matching the algorithm
	// in Claude Code's internal V51() function. Only used on darwin builds.
	ServiceKey string
}

// OAuthInfo holds displayable metadata extracted from Claude Code's stored
// OAuth credentials. The credentials are stored either in the macOS keychain
// (darwin) or as a .credentials.json file inside the profile's config
// directory (Linux, or darwin when no keychain entry is present). The JSON
// structure nests these fields under a "claudeAiOauth" key.
//
// Field types match the upstream JSON schema: Scopes is a string array and
// ExpiresAt is a Unix epoch timestamp in milliseconds (not seconds).
type OAuthInfo struct {
	// SubscriptionType is the Claude subscription plan (e.g., "pro", "max", "free").
	SubscriptionType string `json:"subscriptionType"`

	// Scopes lists the OAuth scopes granted to the token (e.g., "user:inference").
	Scopes []string `json:"scopes"`

	// ExpiresAt is the token expiration time as a Unix epoch in milliseconds.
	// A value of 0 means the field was absent from the credentials.
	ExpiresAt int64 `json:"expiresAt"`

	// RateLimitTier indicates the API rate-limit tier associated with the
	// subscription (e.g., "t3", "t4").
	RateLimitTier string `json:"rateLimitTier"`
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
// "keychain" (darwin only), "file", or "none".
func (p *Profile) AuthStatus() string {
	if p.keychainHas() {
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

	if p.keychainHas() {
		out, err := p.keychainRead()
		if err != nil {
			return nil, fmt.Errorf("reading keychain: %w", err)
		}
		raw = out
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

// Delete removes the profile directory and (on darwin) its keychain entry.
func (p *Profile) Delete() error {
	// Remove keychain entry (ignore errors — may not exist or not applicable)
	_ = p.keychainDelete()
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
