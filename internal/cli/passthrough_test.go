package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/diranged/claude-profile-go/internal/profile"
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

// --- extractResumeID tests ---

func TestExtractResumeID(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no resume", []string{"--model", "opus"}, ""},
		{"bare --resume", []string{"--resume"}, ""},
		{"bare -r", []string{"-r"}, ""},
		{"--resume space", []string{"--resume", "abc123"}, "abc123"},
		{"--resume=value", []string{"--resume=abc123"}, "abc123"},
		{"-r space", []string{"-r", "abc123"}, "abc123"},
		{"-r joined", []string{"-rabc123"}, "abc123"},
		{"resume with other flags", []string{"--model", "opus", "--resume", "abc123", "-c", "hi"}, "abc123"},
		{"resume at end", []string{"--model", "opus", "--resume", "abc123"}, "abc123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractResumeID(tt.args)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidateResumeCwd_WrongDirectory(t *testing.T) {
	initLogger()
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := profile.Load("test")
	require.NoError(t, p.EnsureDir())

	// Create a fake session in a different cwd
	projDir := filepath.Join(p.ConfigDir, "projects", "-some-other-repo")
	require.NoError(t, os.MkdirAll(projDir, 0755))
	sessionID := "abc12345-1111-2222-3333-444444444444"
	jsonl := `{"type":"user","cwd":"/some/other/repo","gitBranch":"main","message":{"role":"user","content":"hello"}}` + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(projDir, sessionID+".jsonl"), []byte(jsonl), 0644))

	err := validateResumeCwd(p, "abc12345")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "was started in a different directory")
	assert.Contains(t, err.Error(), "/some/other/repo")
}

func TestValidateResumeCwd_CorrectDirectory(t *testing.T) {
	initLogger()
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := profile.Load("test")
	require.NoError(t, p.EnsureDir())

	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Encode cwd as Claude does (/ -> -)
	projDir := filepath.Join(p.ConfigDir, "projects", "-this-project")
	require.NoError(t, os.MkdirAll(projDir, 0755))
	sessionID := "match000-1111-2222-3333-444444444444"
	jsonl := `{"type":"user","cwd":"` + cwd + `","gitBranch":"main","message":{"role":"user","content":"hello"}}` + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(projDir, sessionID+".jsonl"), []byte(jsonl), 0644))

	err = validateResumeCwd(p, "match000")
	assert.NoError(t, err)
}

func TestValidateResumeCwd_SessionNotFound(t *testing.T) {
	initLogger()
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := profile.Load("test")
	require.NoError(t, p.EnsureDir())

	// No session files — should pass through (nil error)
	err := validateResumeCwd(p, "doesnotexist")
	assert.NoError(t, err)
}

func TestValidateResumeCwd_MultipleMatches(t *testing.T) {
	initLogger()
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := profile.Load("test")
	require.NoError(t, p.EnsureDir())

	for _, dir := range []string{"-repo-a", "-repo-b"} {
		projDir := filepath.Join(p.ConfigDir, "projects", dir)
		require.NoError(t, os.MkdirAll(projDir, 0755))
		id := "dup00000-" + dir + "-2222-3333-444444444444"
		jsonl := `{"type":"user","cwd":"/` + dir + `","gitBranch":"main","message":{"role":"user","content":"hi"}}` + "\n"
		require.NoError(t, os.WriteFile(filepath.Join(projDir, id+".jsonl"), []byte(jsonl), 0644))
	}

	err := validateResumeCwd(p, "dup00000")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "matches multiple sessions")
}

func TestShortID(t *testing.T) {
	assert.Equal(t, "abc12345", shortID("abc12345-1111-2222-3333-444444444444"))
	assert.Equal(t, "short", shortID("short"))
	assert.Equal(t, "12345678", shortID("12345678"))
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
