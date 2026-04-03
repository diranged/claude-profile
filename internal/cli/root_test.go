package cli

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveProfile_NoProfileSet(t *testing.T) {
	// Reset viper state
	viper.Reset()
	profileFlag = ""

	_, err := resolveProfile()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no profile specified")
}

func TestResolveProfile_FromEnvVar(t *testing.T) {
	viper.Reset()
	t.Setenv("CLAUDE_PROFILE", "envprofile")

	// Re-bind the env var
	_ = viper.BindEnv("profile", "CLAUDE_PROFILE")

	name, err := resolveProfile()
	require.NoError(t, err)
	assert.Equal(t, "envprofile", name)
}

func TestResolveProfile_FromViper(t *testing.T) {
	viper.Reset()
	viper.Set("profile", "viperprofile")

	name, err := resolveProfile()
	require.NoError(t, err)
	assert.Equal(t, "viperprofile", name)
}

func TestNewRootCmd_Structure(t *testing.T) {
	initLogger()
	cmd := newRootCmd()

	assert.Equal(t, "claude-profile", cmd.Use)
	assert.True(t, cmd.SilenceUsage)
	assert.True(t, cmd.SilenceErrors)

	// Check subcommands exist
	subCmds := cmd.Commands()
	names := make([]string, len(subCmds))
	for i, c := range subCmds {
		names[i] = c.Name()
	}
	assert.Contains(t, names, "create")
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "show")
	assert.Contains(t, names, "delete")
	assert.Contains(t, names, "statusline")
}

func TestNewRootCmd_ProfileFlag(t *testing.T) {
	initLogger()
	cmd := newRootCmd()

	f := cmd.PersistentFlags().Lookup("profile")
	require.NotNil(t, f)
	assert.Equal(t, "p", f.Shorthand)
}

func TestInitLogger_Default(t *testing.T) {
	t.Setenv("CLAUDE_PROFILE_DEBUG", "")
	initLogger()
	assert.NotNil(t, log)
}

func TestInitLogger_Debug(t *testing.T) {
	t.Setenv("CLAUDE_PROFILE_DEBUG", "1")
	initLogger()
	assert.NotNil(t, log)
}

func TestExecute_NoProfile(t *testing.T) {
	// Reset viper to avoid leaking state
	viper.Reset()
	t.Setenv("CLAUDE_PROFILE", "")

	err := Execute()
	// Should fail because no profile is specified
	assert.Error(t, err)
}
