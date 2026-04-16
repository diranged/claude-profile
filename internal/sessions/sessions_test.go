package sessions

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeSession creates a fake session JSONL file under
// configDir/projects/<encodedCwd>/<id>.jsonl with the given lines.
func writeSession(t *testing.T, configDir, encodedCwd, id string, lines []string) string {
	t.Helper()
	dir := filepath.Join(configDir, "projects", encodedCwd)
	require.NoError(t, os.MkdirAll(dir, 0755))
	path := filepath.Join(dir, id+".jsonl")
	var content string
	for _, l := range lines {
		content += l + "\n"
	}
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func TestFindByPrefix_ExactMatch(t *testing.T) {
	configDir := t.TempDir()
	writeSession(t, configDir, "-Users-test-repo", "abc12345-1111-2222-3333-444444444444", []string{
		`{"type":"permission-mode","permissionMode":"default","sessionId":"abc12345-1111-2222-3333-444444444444"}`,
		`{"type":"user","cwd":"/Users/test/repo","gitBranch":"main","message":{"role":"user","content":"hello world"}}`,
	})

	results, err := FindByPrefix(configDir, "abc12345-1111-2222-3333-444444444444")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "abc12345-1111-2222-3333-444444444444", results[0].ID)
	assert.Equal(t, "/Users/test/repo", results[0].Cwd)
	assert.Equal(t, "main", results[0].GitBranch)
	assert.Equal(t, "hello world", results[0].FirstPrompt)
}

func TestFindByPrefix_PrefixMatch(t *testing.T) {
	configDir := t.TempDir()
	writeSession(t, configDir, "-Users-test-repo", "abc12345-1111-2222-3333-444444444444", []string{
		`{"type":"user","cwd":"/Users/test/repo","gitBranch":"main","message":{"role":"user","content":"first session"}}`,
	})
	writeSession(t, configDir, "-Users-test-other", "abc99999-aaaa-bbbb-cccc-dddddddddddd", []string{
		`{"type":"user","cwd":"/Users/test/other","gitBranch":"dev","message":{"role":"user","content":"second session"}}`,
	})

	results, err := FindByPrefix(configDir, "abc")
	require.NoError(t, err)
	require.Len(t, results, 2)
}

func TestFindByPrefix_NoMatch(t *testing.T) {
	configDir := t.TempDir()
	writeSession(t, configDir, "-Users-test-repo", "abc12345-1111-2222-3333-444444444444", []string{
		`{"type":"user","cwd":"/Users/test/repo","gitBranch":"main","message":{"role":"user","content":"hello"}}`,
	})

	results, err := FindByPrefix(configDir, "zzz")
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestFindByPrefix_EmptyPrefix(t *testing.T) {
	configDir := t.TempDir()
	writeSession(t, configDir, "-Users-test-repo", "aaa11111-1111-1111-1111-111111111111", []string{
		`{"type":"user","cwd":"/Users/test/repo","gitBranch":"main","message":{"role":"user","content":"one"}}`,
	})
	writeSession(t, configDir, "-Users-test-other", "bbb22222-2222-2222-2222-222222222222", []string{
		`{"type":"user","cwd":"/Users/test/other","gitBranch":"dev","message":{"role":"user","content":"two"}}`,
	})

	results, err := FindByPrefix(configDir, "")
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestFindByPrefix_MalformedJSONL(t *testing.T) {
	configDir := t.TempDir()
	writeSession(t, configDir, "-Users-test-repo", "bad00000-0000-0000-0000-000000000000", []string{
		`not valid json at all`,
		`{"type":"user","cwd":"/Users/test/repo","gitBranch":"main","message":{"role":"user","content":"found it"}}`,
	})

	results, err := FindByPrefix(configDir, "bad00000")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "/Users/test/repo", results[0].Cwd)
	assert.Equal(t, "found it", results[0].FirstPrompt)
}

func TestFindByPrefix_NoUserRecord(t *testing.T) {
	configDir := t.TempDir()
	writeSession(t, configDir, "-Users-test-repo", "nope0000-0000-0000-0000-000000000000", []string{
		`{"type":"permission-mode","permissionMode":"default"}`,
		`{"type":"file-history-snapshot","messageId":"xxx"}`,
	})

	results, err := FindByPrefix(configDir, "nope0000")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "", results[0].Cwd)
	assert.Equal(t, "", results[0].FirstPrompt)
}

func TestFindByPrefix_ContentBlockArray(t *testing.T) {
	configDir := t.TempDir()
	writeSession(t, configDir, "-Users-test-repo", "blk00000-0000-0000-0000-000000000000", []string{
		`{"type":"user","cwd":"/Users/test/repo","gitBranch":"main","message":{"role":"user","content":[{"type":"text","text":"array content"}]}}`,
	})

	results, err := FindByPrefix(configDir, "blk00000")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "array content", results[0].FirstPrompt)
}

func TestFindByPrefix_MultilinePromptTruncated(t *testing.T) {
	configDir := t.TempDir()
	long := "this is a really long first line that goes on and on and on and on and on and on and on and on and on and on and on beyond 100 chars"
	writeSession(t, configDir, "-Users-test-repo", "lng00000-0000-0000-0000-000000000000", []string{
		`{"type":"user","cwd":"/x","gitBranch":"main","message":{"role":"user","content":"` + long + `"}}`,
	})

	results, err := FindByPrefix(configDir, "lng00000")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.LessOrEqual(t, len(results[0].FirstPrompt), 100)
	assert.True(t, len(results[0].FirstPrompt) > 0)
}

func TestFindByPrefix_SortedByModTimeDesc(t *testing.T) {
	configDir := t.TempDir()

	p1 := writeSession(t, configDir, "-Users-test-a", "aaa00000-0000-0000-0000-000000000001", []string{
		`{"type":"user","cwd":"/a","gitBranch":"main","message":{"role":"user","content":"older"}}`,
	})
	p2 := writeSession(t, configDir, "-Users-test-b", "aaa00000-0000-0000-0000-000000000002", []string{
		`{"type":"user","cwd":"/b","gitBranch":"main","message":{"role":"user","content":"newer"}}`,
	})

	// Make p1 older than p2
	info, err := os.Stat(p2)
	require.NoError(t, err)
	oldTime := info.ModTime().Add(-10 * time.Minute)
	require.NoError(t, os.Chtimes(p1, oldTime, oldTime))

	results, err := FindByPrefix(configDir, "aaa00000")
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "newer", results[0].FirstPrompt)
	assert.Equal(t, "older", results[1].FirstPrompt)
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "abc", truncate("abc", 10))
	assert.Equal(t, "abcdefg...", truncate("abcdefghijklm", 10))
	assert.Equal(t, "", truncate("", 10))
}

func TestFirstLine(t *testing.T) {
	assert.Equal(t, "hello", firstLine("hello\nworld"))
	assert.Equal(t, "single", firstLine("single"))
	assert.Equal(t, "trimmed", firstLine("  trimmed\n  second  "))
	assert.Equal(t, "", firstLine(""))
}

func TestFindByPrefix_NoProjectsDir(t *testing.T) {
	configDir := t.TempDir()
	// Don't create projects/ — should return empty, not error
	results, err := FindByPrefix(configDir, "abc")
	require.NoError(t, err)
	assert.Empty(t, results)
}
