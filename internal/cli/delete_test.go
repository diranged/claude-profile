package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteCmd_ProfileNotExist(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	initLogger()
	cmd := newDeleteCmd()
	cmd.SetArgs([]string{"nonexistent"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestDeleteCmd_ForceDelete(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	// Create the profile
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "work", "config"), 0700))

	initLogger()
	cmd := newDeleteCmd()
	cmd.SetArgs([]string{"work", "--force"})
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "deleted")

	// Verify directory is gone
	_, err = os.Stat(filepath.Join(tmp, "work"))
	assert.True(t, os.IsNotExist(err))
}

func TestDeleteCmd_NoArgs(t *testing.T) {
	initLogger()
	cmd := newDeleteCmd()
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	assert.Error(t, err)
}
