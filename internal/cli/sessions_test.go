package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTestSession creates a fake session JSONL file and returns its path.
func writeTestSession(t *testing.T, configDir, encodedCwd, id, cwd, branch, prompt string) string {
	t.Helper()
	dir := filepath.Join(configDir, "projects", encodedCwd)
	require.NoError(t, os.MkdirAll(dir, 0755))
	path := filepath.Join(dir, id+".jsonl")
	line := `{"type":"user","cwd":"` + cwd + `","gitBranch":"` + branch + `","message":{"role":"user","content":"` + prompt + `"}}` + "\n"
	require.NoError(t, os.WriteFile(path, []byte(line), 0644))
	return path
}

func TestSessionsCmd_ListsSessions(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)
	t.Setenv("CLAUDE_PROFILE", "test")

	configDir := filepath.Join(tmp, "test", "config")
	require.NoError(t, os.MkdirAll(configDir, 0700))

	writeTestSession(t, configDir, "-Users-test-repo-a", "aaa11111-1111-1111-1111-111111111111",
		"/Users/test/repo-a", "main", "hello from repo a")
	writeTestSession(t, configDir, "-Users-test-repo-b", "bbb22222-2222-2222-2222-222222222222",
		"/Users/test/repo-b", "dev", "hello from repo b")

	initLogger()
	cmd := newRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"sessions", "--since", "1d"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "/Users/test/repo-a")
	assert.Contains(t, output, "/Users/test/repo-b")
	assert.Contains(t, output, "hello from repo a")
	assert.Contains(t, output, "hello from repo b")
	assert.Contains(t, output, "aaa11111")
	assert.Contains(t, output, "bbb22222")
	assert.Contains(t, output, "[branch: main]")
	assert.Contains(t, output, "[branch: dev]")
}

func TestSessionsCmd_RepoFilter(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)
	t.Setenv("CLAUDE_PROFILE", "test")

	configDir := filepath.Join(tmp, "test", "config")
	require.NoError(t, os.MkdirAll(configDir, 0700))

	writeTestSession(t, configDir, "-Users-test-repo-a", "aaa11111-1111-1111-1111-111111111111",
		"/Users/test/repo-a", "main", "first")
	writeTestSession(t, configDir, "-Users-test-repo-b", "bbb22222-2222-2222-2222-222222222222",
		"/Users/test/repo-b", "dev", "second")

	initLogger()
	cmd := newRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"sessions", "--repo", "repo-a"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "repo-a")
	assert.NotContains(t, output, "repo-b")
}

func TestSessionsCmd_SinceFilter(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)
	t.Setenv("CLAUDE_PROFILE", "test")

	configDir := filepath.Join(tmp, "test", "config")
	require.NoError(t, os.MkdirAll(configDir, 0700))

	recentPath := writeTestSession(t, configDir, "-recent", "aaa11111-1111-1111-1111-111111111111",
		"/recent", "main", "recent session")
	oldPath := writeTestSession(t, configDir, "-old", "bbb22222-2222-2222-2222-222222222222",
		"/old", "main", "old session")

	// Make the old session actually old
	oldTime := time.Now().Add(-30 * 24 * time.Hour)
	require.NoError(t, os.Chtimes(oldPath, oldTime, oldTime))
	// Touch the recent one to ensure it's fresh
	require.NoError(t, os.Chtimes(recentPath, time.Now(), time.Now()))

	initLogger()
	cmd := newRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"sessions", "--since", "7d"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "recent session")
	assert.NotContains(t, output, "old session")
}

func TestSessionsCmd_NoSessions(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)
	t.Setenv("CLAUDE_PROFILE", "test")

	configDir := filepath.Join(tmp, "test", "config")
	require.NoError(t, os.MkdirAll(configDir, 0700))

	initLogger()
	cmd := newRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"sessions"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No sessions found")
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
		err   bool
	}{
		{"7d", 7 * 24 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"30d", 30 * 24 * time.Hour, false},
		{"24h", 24 * time.Hour, false},
		{"30m", 30 * time.Minute, false},
		{"2h30m", 2*time.Hour + 30*time.Minute, false},
		{"", 0, false},
		{"abc", 0, true},
		{"d", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseDuration(tt.input)
			if tt.err {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestContainsCI(t *testing.T) {
	assert.True(t, containsCI("/Users/test/Sproutbook", "sproutbook"))
	assert.True(t, containsCI("/Users/test/sproutbook", "Sproutbook"))
	assert.False(t, containsCI("/Users/test/repo", "sproutbook"))
	assert.True(t, containsCI("anything", ""))
}
