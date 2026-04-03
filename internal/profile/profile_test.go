package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfilesDir_Default(t *testing.T) {
	t.Setenv("CLAUDE_PROFILES_DIR", "")
	home, _ := os.UserHomeDir()
	assert.Equal(t, filepath.Join(home, ".claude-profiles"), ProfilesDir())
}

func TestProfilesDir_EnvOverride(t *testing.T) {
	t.Setenv("CLAUDE_PROFILES_DIR", "/tmp/custom-profiles")
	assert.Equal(t, "/tmp/custom-profiles", ProfilesDir())
}

func TestLoad(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := Load("work")
	assert.Equal(t, "work", p.Name)
	assert.Equal(t, filepath.Join(tmp, "work"), p.Dir)
	assert.Equal(t, filepath.Join(tmp, "work", "config"), p.ConfigDir)
	assert.NotEmpty(t, p.ServiceKey)
	// ServiceKey should have the hash suffix
	assert.Contains(t, p.ServiceKey, "Claude Code-credentials-")
}

func TestExists_NotExist(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := Load("nonexistent")
	assert.False(t, p.Exists())
}

func TestExists_AfterEnsureDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := Load("myprofile")
	assert.False(t, p.Exists())

	require.NoError(t, p.EnsureDir())
	assert.True(t, p.Exists())
}

func TestEnsureDir_CreatesNestedDirs(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := Load("nested")
	require.NoError(t, p.EnsureDir())

	info, err := os.Stat(p.ConfigDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestEnsureDir_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := Load("idem")
	require.NoError(t, p.EnsureDir())
	require.NoError(t, p.EnsureDir()) // second call should not fail
}

func TestAuthStatus_None(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := Load("empty")
	require.NoError(t, p.EnsureDir())

	assert.Equal(t, "none", p.AuthStatus())
}

func TestAuthStatus_File(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := Load("filecred")
	require.NoError(t, p.EnsureDir())

	// Create a credentials file
	credFile := filepath.Join(p.ConfigDir, ".credentials.json")
	require.NoError(t, os.WriteFile(credFile, []byte(`{}`), 0600))

	// On non-macOS or when keychain entry doesn't exist, should return "file"
	status := p.AuthStatus()
	// It will be either "keychain" (if security command finds it) or "file"
	assert.Contains(t, []string{"file", "keychain"}, status)
}

func TestKeychainService_KnownHash(t *testing.T) {
	// This is the specific test case from the instructions
	configDir := "/Users/diranged/.claude-profiles/work/config"
	svc := keychainService(configDir)
	assert.Equal(t, "Claude Code-credentials-6061db4b", svc)
}

func TestKeychainService_DifferentPaths(t *testing.T) {
	svc1 := keychainService("/path/one")
	svc2 := keychainService("/path/two")
	assert.NotEqual(t, svc1, svc2)
	assert.True(t, len(svc1) > len(keychainServicePrefix))
	assert.True(t, len(svc2) > len(keychainServicePrefix))
}

func TestKeychainService_Prefix(t *testing.T) {
	svc := keychainService("/any/path")
	assert.Contains(t, svc, keychainServicePrefix+"-")
}

func TestList_EmptyDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	names, err := List()
	require.NoError(t, err)
	assert.Empty(t, names)
}

func TestList_NonExistentDir(t *testing.T) {
	t.Setenv("CLAUDE_PROFILES_DIR", "/nonexistent/path/that/does/not/exist")

	names, err := List()
	require.NoError(t, err)
	assert.Nil(t, names)
}

func TestList_WithProfiles(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	// Create profile directories
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "work"), 0700))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "personal"), 0700))

	// Create a file (should be excluded)
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "not-a-profile"), []byte("x"), 0600))

	names, err := List()
	require.NoError(t, err)
	assert.Len(t, names, 2)
	assert.Contains(t, names, "work")
	assert.Contains(t, names, "personal")
}

func TestOAuthDetails_FromFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := Load("oauthtest")
	require.NoError(t, p.EnsureDir())

	creds := map[string]interface{}{
		"claudeAiOauth": map[string]interface{}{
			"subscriptionType": "pro",
			"scopes":           []string{"user:read", "user:write"},
			"expiresAt":        1700000000000,
			"rateLimitTier":    "tier1",
		},
	}
	data, err := json.Marshal(creds)
	require.NoError(t, err)

	credFile := filepath.Join(p.ConfigDir, ".credentials.json")
	require.NoError(t, os.WriteFile(credFile, data, 0600))

	info, err := p.OAuthDetails()
	require.NoError(t, err)
	assert.Equal(t, "pro", info.SubscriptionType)
	assert.Equal(t, []string{"user:read", "user:write"}, info.Scopes)
	assert.Equal(t, int64(1700000000000), info.ExpiresAt)
	assert.Equal(t, "tier1", info.RateLimitTier)
}

func TestOAuthDetails_NoCredentials(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := Load("nocreds")
	require.NoError(t, p.EnsureDir())

	_, err := p.OAuthDetails()
	assert.Error(t, err)
}

func TestOAuthDetails_InvalidJSON(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := Load("badjson")
	require.NoError(t, p.EnsureDir())

	credFile := filepath.Join(p.ConfigDir, ".credentials.json")
	require.NoError(t, os.WriteFile(credFile, []byte("not json"), 0600))

	_, err := p.OAuthDetails()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing credentials")
}

func TestDelete_RemovesDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := Load("todelete")
	require.NoError(t, p.EnsureDir())
	assert.True(t, p.Exists())

	require.NoError(t, p.Delete())

	// Directory should be gone
	_, err := os.Stat(p.Dir)
	assert.True(t, os.IsNotExist(err))
}

func TestCurrentUser(t *testing.T) {
	u := currentUser()
	assert.NotEmpty(t, u)
}

func TestSetEnv_NewKey(t *testing.T) {
	env := []string{"FOO=bar", "BAZ=qux"}
	env = setEnvHelper(env, "NEW", "val")
	assert.Contains(t, env, "NEW=val")
}

func TestSetEnv_ReplaceKey(t *testing.T) {
	env := []string{"FOO=bar", "BAZ=qux"}
	env = setEnvHelper(env, "FOO", "newbar")
	assert.Contains(t, env, "FOO=newbar")
	assert.NotContains(t, env, "FOO=bar")
}

// setEnvHelper replicates setEnv logic from passthrough.go for testing
// the profile package's own env-related behavior. We test setEnv
// separately in the cli package.
func setEnvHelper(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if len(e) >= len(prefix) && e[:len(prefix)] == prefix {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}
