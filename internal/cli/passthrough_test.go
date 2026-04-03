package cli

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractClaudeArgs_NoProfileFlag(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"claude-profile", "--model", "opus", "-c", "hello"}
	result := extractClaudeArgs()
	assert.Equal(t, []string{"--model", "opus", "-c", "hello"}, result)
}

func TestExtractClaudeArgs_DashP_Space(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"claude-profile", "-P", "work", "--model", "opus"}
	result := extractClaudeArgs()
	assert.Equal(t, []string{"--model", "opus"}, result)
}

func TestExtractClaudeArgs_DoubleDashProfile_Space(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"claude-profile", "--profile", "work", "--model", "opus"}
	result := extractClaudeArgs()
	assert.Equal(t, []string{"--model", "opus"}, result)
}

func TestExtractClaudeArgs_DoubleDashProfile_Equals(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"claude-profile", "--profile=work", "--model", "opus"}
	result := extractClaudeArgs()
	assert.Equal(t, []string{"--model", "opus"}, result)
}

func TestExtractClaudeArgs_DashP_Joined(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"claude-profile", "-Pwork", "--model", "opus"}
	result := extractClaudeArgs()
	assert.Equal(t, []string{"--model", "opus"}, result)
}

func TestExtractClaudeArgs_ProfileAtEnd(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"claude-profile", "--model", "opus", "-P", "work"}
	result := extractClaudeArgs()
	assert.Equal(t, []string{"--model", "opus"}, result)
}

func TestExtractClaudeArgs_MixedFlags(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"claude-profile", "-P", "work", "auth", "login", "--sso"}
	result := extractClaudeArgs()
	assert.Equal(t, []string{"auth", "login", "--sso"}, result)
}

func TestExtractClaudeArgs_NoArgs(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"claude-profile"}
	result := extractClaudeArgs()
	assert.Empty(t, result)
}

func TestExtractClaudeArgs_OnlyProfile(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"claude-profile", "-P", "work"}
	result := extractClaudeArgs()
	assert.Empty(t, result)
}

func TestExtractClaudeArgs_ProfileWithDashAtEnd(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// -P at the very end with no value following
	os.Args = []string{"claude-profile", "list", "-P"}
	result := extractClaudeArgs()
	assert.Equal(t, []string{"list"}, result)
}

func TestSetEnv_NewKey(t *testing.T) {
	env := []string{"FOO=bar", "BAZ=qux"}
	result := setEnv(env, "NEW", "val")
	assert.Contains(t, result, "NEW=val")
	assert.Contains(t, result, "FOO=bar")
	assert.Contains(t, result, "BAZ=qux")
}

func TestSetEnv_ReplaceExisting(t *testing.T) {
	env := []string{"FOO=bar", "BAZ=qux"}
	result := setEnv(env, "FOO", "newbar")
	assert.Contains(t, result, "FOO=newbar")
	assert.NotContains(t, result, "FOO=bar")
	assert.Len(t, result, 2)
}

func TestSetEnv_EmptyEnv(t *testing.T) {
	env := []string{}
	result := setEnv(env, "KEY", "value")
	assert.Equal(t, []string{"KEY=value"}, result)
}

func TestSetEnv_ValueWithEquals(t *testing.T) {
	env := []string{}
	result := setEnv(env, "KEY", "a=b=c")
	assert.Equal(t, []string{"KEY=a=b=c"}, result)
}

func TestRepeat(t *testing.T) {
	assert.Equal(t, "", repeat("x", 0))
	assert.Equal(t, "x", repeat("x", 1))
	assert.Equal(t, "xxx", repeat("x", 3))
	assert.Equal(t, "ababab", repeat("ab", 3))
}

func TestRepeat_Empty(t *testing.T) {
	assert.Equal(t, "", repeat("", 5))
}

func TestPrintBanner_DoesNotPanic(t *testing.T) {
	// Redirect stderr to capture output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	printBanner("testprofile", "/tmp/test/config", "keychain", "pro", 108)

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "testprofile")
	assert.Contains(t, output, "/tmp/test/config")
	assert.Contains(t, output, "keychain")
	assert.Contains(t, output, "pro")
}

func TestPrintBanner_CustomColor(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	printBanner("work", "/config", "file", "free", 204)

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "work")
	// Should contain the color escape sequence for 204
	assert.Contains(t, output, "\033[38;5;204m")
}

func TestPrintBanner_LongConfigDir(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	longDir := "/very/long/path/that/exceeds/typical/widths/for/config/directory/name/goes/here/and/keeps/going"
	printBanner("work", longDir, "none", "unknown", 33)

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, longDir)
}
