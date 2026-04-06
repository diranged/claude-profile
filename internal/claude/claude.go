// Package claude handles locating and invoking the real claude binary.
//
// claude-profile needs to call the actual claude binary (not shell aliases)
// to avoid interference from user aliases that add flags like --mcp-config.
// This package resolves the binary path and builds the environment for
// profile-isolated execution.
package claude

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// FindBinary locates the real claude binary on disk, bypassing shell aliases.
// It checks common installation paths in priority order.
func FindBinary() (string, error) {
	// First: check PATH using exec.LookPath (skips aliases, finds real binary)
	if p, err := exec.LookPath("claude"); err == nil {
		return p, nil
	}

	// Fallback: check well-known install locations
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".local", "bin", "claude"),
		"/usr/local/bin/claude",
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c, nil
		}
	}

	return "", fmt.Errorf("claude binary not found in PATH or common locations")
}

// BuildEnv returns an os.Environ() copy with CLAUDE_CONFIG_DIR set to the
// given profile config directory. This is the key mechanism for profile
// isolation — Claude Code hashes this path into its keychain service name.
func BuildEnv(configDir string) []string {
	env := os.Environ()
	found := false
	for i, e := range env {
		if len(e) > 17 && e[:17] == "CLAUDE_CONFIG_DIR" {
			env[i] = "CLAUDE_CONFIG_DIR=" + configDir
			found = true
			break
		}
	}
	if !found {
		env = append(env, "CLAUDE_CONFIG_DIR="+configDir)
	}
	return env
}

// UnsetEnv removes a named environment variable from an env slice. If the key
// is not present, the slice is returned unchanged.
func UnsetEnv(env []string, key string) []string {
	prefix := key + "="
	for i, e := range env {
		if len(e) >= len(prefix) && e[:len(prefix)] == prefix {
			return append(env[:i], env[i+1:]...)
		}
	}
	return env
}

// RunDirect executes the claude binary as a child process without a PTY
// wrapper, connecting stdin/stdout/stderr directly. It returns the child's exit
// code and any non-exit error. This is used by subcommands like "auth login"
// where the caller does not want to replace the current process via exec.
func RunDirect(binary string, args []string, env []string) (int, error) {
	cmd := exec.Command(binary, args...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
}
