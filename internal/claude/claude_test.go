package claude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindBinary_InPath(t *testing.T) {
	// Create a temp directory with a fake claude binary
	tmp := t.TempDir()
	fakeClaude := filepath.Join(tmp, "claude")
	require.NoError(t, os.WriteFile(fakeClaude, []byte("#!/bin/sh\n"), 0755))

	// Prepend to PATH
	t.Setenv("PATH", tmp+":"+os.Getenv("PATH"))

	bin, err := FindBinary()
	require.NoError(t, err)
	assert.Equal(t, fakeClaude, bin)
}

func TestFindBinary_FallbackLocalBin(t *testing.T) {
	// Create a temp home dir with .local/bin/claude
	tmp := t.TempDir()
	localBin := filepath.Join(tmp, ".local", "bin")
	require.NoError(t, os.MkdirAll(localBin, 0755))
	fakeClaude := filepath.Join(localBin, "claude")
	require.NoError(t, os.WriteFile(fakeClaude, []byte("#!/bin/sh\n"), 0755))

	// Set HOME so the fallback check works, and clear PATH
	t.Setenv("HOME", tmp)
	t.Setenv("PATH", "/nonexistent")

	bin, err := FindBinary()
	require.NoError(t, err)
	assert.Equal(t, fakeClaude, bin)
}

func TestFindBinary_NotFound(t *testing.T) {
	t.Setenv("PATH", "/nonexistent-dir-for-test")
	t.Setenv("HOME", "/nonexistent-home-for-test")

	_, err := FindBinary()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "claude binary not found")
}

func TestBuildEnv_AddsConfigDir(t *testing.T) {
	// Clear CLAUDE_CONFIG_DIR from current env
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	_ = os.Unsetenv("CLAUDE_CONFIG_DIR")

	env := BuildEnv("/test/config/dir")

	found := false
	for _, e := range env {
		if e == "CLAUDE_CONFIG_DIR=/test/config/dir" {
			found = true
			break
		}
	}
	assert.True(t, found, "CLAUDE_CONFIG_DIR should be set in env")
}

func TestBuildEnv_ReplacesExistingConfigDir(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "/old/path")

	env := BuildEnv("/new/path")

	found := false
	count := 0
	for _, e := range env {
		if len(e) > 17 && e[:17] == "CLAUDE_CONFIG_DIR" {
			count++
			if e == "CLAUDE_CONFIG_DIR=/new/path" {
				found = true
			}
		}
	}
	assert.True(t, found, "CLAUDE_CONFIG_DIR should be updated to /new/path")
	assert.Equal(t, 1, count, "CLAUDE_CONFIG_DIR should appear exactly once")
}

func TestRunDirect_Success(t *testing.T) {
	code, err := RunDirect("echo", []string{"hello"}, os.Environ())
	require.NoError(t, err)
	assert.Equal(t, 0, code)
}

func TestRunDirect_ExitCode(t *testing.T) {
	code, err := RunDirect("sh", []string{"-c", "exit 42"}, os.Environ())
	require.NoError(t, err)
	assert.Equal(t, 42, code)
}

func TestRunDirect_BinaryNotFound(t *testing.T) {
	_, err := RunDirect("/nonexistent/binary", []string{}, os.Environ())
	assert.Error(t, err)
}

func TestBuildEnv_PreservesOtherVars(t *testing.T) {
	t.Setenv("MY_TEST_VAR_12345", "hello")

	env := BuildEnv("/some/path")

	found := false
	for _, e := range env {
		if e == "MY_TEST_VAR_12345=hello" {
			found = true
			break
		}
	}
	assert.True(t, found, "other env vars should be preserved")
}
