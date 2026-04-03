package cli

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatuslineCmd_WithEnvVars(t *testing.T) {
	t.Setenv("CLAUDE_PROFILE_NAME", "work")
	t.Setenv("CLAUDE_PROFILE_AUTH", "keychain")
	t.Setenv("CLAUDE_PROFILE_SUB", "pro")
	t.Setenv("CLAUDE_PROFILE_COLOR", "204")

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Provide empty stdin
	oldStdin := os.Stdin
	stdinR, stdinW, _ := os.Pipe()
	stdinW.Close()
	os.Stdin = stdinR

	cmd := newStatuslineCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout
	os.Stdin = oldStdin

	require.NoError(t, err)

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "Profile: work")
	assert.Contains(t, output, "Auth: keychain")
	assert.Contains(t, output, "Subscription: pro")
}

func TestStatuslineCmd_NoEnvVars(t *testing.T) {
	t.Setenv("CLAUDE_PROFILE_NAME", "")
	t.Setenv("CLAUDE_PROFILE_AUTH", "")
	t.Setenv("CLAUDE_PROFILE_SUB", "")
	t.Setenv("CLAUDE_PROFILE_COLOR", "")

	// Unset the env vars
	os.Unsetenv("CLAUDE_PROFILE_NAME")
	os.Unsetenv("CLAUDE_PROFILE_AUTH")
	os.Unsetenv("CLAUDE_PROFILE_SUB")
	os.Unsetenv("CLAUDE_PROFILE_COLOR")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldStdin := os.Stdin
	stdinR, stdinW, _ := os.Pipe()
	stdinW.Close()
	os.Stdin = stdinR

	cmd := newStatuslineCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout
	os.Stdin = oldStdin

	require.NoError(t, err)

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// With no CLAUDE_PROFILE_NAME, should output nothing
	assert.Empty(t, output)
}

func TestStatuslineCmd_DefaultColor(t *testing.T) {
	t.Setenv("CLAUDE_PROFILE_NAME", "test")
	t.Setenv("CLAUDE_PROFILE_AUTH", "none")
	t.Setenv("CLAUDE_PROFILE_SUB", "free")
	// No color set — should default to 108

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldStdin := os.Stdin
	stdinR, stdinW, _ := os.Pipe()
	stdinW.Close()
	os.Stdin = stdinR

	cmd := newStatuslineCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout
	os.Stdin = oldStdin

	require.NoError(t, err)

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "Profile: test")
	// Default color 108
	assert.Contains(t, output, "\033[38;5;108m")
}

func TestStatuslineCmd_InvalidColor(t *testing.T) {
	t.Setenv("CLAUDE_PROFILE_NAME", "test")
	t.Setenv("CLAUDE_PROFILE_AUTH", "none")
	t.Setenv("CLAUDE_PROFILE_SUB", "free")
	t.Setenv("CLAUDE_PROFILE_COLOR", "notanumber")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldStdin := os.Stdin
	stdinR, stdinW, _ := os.Pipe()
	stdinW.Close()
	os.Stdin = stdinR

	cmd := newStatuslineCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout
	os.Stdin = oldStdin

	require.NoError(t, err)

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Should fall back to default color 108
	assert.Contains(t, output, "\033[38;5;108m")
}

func TestStatuslineCmd_ColorOutOfRange(t *testing.T) {
	t.Setenv("CLAUDE_PROFILE_NAME", "test")
	t.Setenv("CLAUDE_PROFILE_AUTH", "none")
	t.Setenv("CLAUDE_PROFILE_SUB", "free")
	t.Setenv("CLAUDE_PROFILE_COLOR", "999")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldStdin := os.Stdin
	stdinR, stdinW, _ := os.Pipe()
	stdinW.Close()
	os.Stdin = stdinR

	cmd := newStatuslineCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout
	os.Stdin = oldStdin

	require.NoError(t, err)

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Should fall back to default color 108
	assert.Contains(t, output, "\033[38;5;108m")
}

func TestStatuslineCmd_WithChainedCommand(t *testing.T) {
	t.Setenv("CLAUDE_PROFILE_NAME", "work")
	t.Setenv("CLAUDE_PROFILE_AUTH", "keychain")
	t.Setenv("CLAUDE_PROFILE_SUB", "pro")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldStdin := os.Stdin
	stdinR, stdinW, _ := os.Pipe()
	stdinW.Close()
	os.Stdin = stdinR

	cmd := newStatuslineCmd()
	// Chain with echo command
	cmd.SetArgs([]string{"--", "echo", "chained"})
	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout
	os.Stdin = oldStdin

	require.NoError(t, err)

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "Profile: work")
	assert.Contains(t, output, "chained")
}
