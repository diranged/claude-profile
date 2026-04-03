package cli

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRawArgs_NoArgs(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"claude-profile"}
	assert.Nil(t, rawArgs())
}

func TestRawArgs_SingleArg(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"claude-profile", "list"}
	assert.Equal(t, []string{"list"}, rawArgs())
}

func TestRawArgs_MultipleArgs(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"claude-profile", "-p", "work", "--model", "opus"}
	assert.Equal(t, []string{"-p", "work", "--model", "opus"}, rawArgs())
}

func TestRawArgs_OnlyBinary(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"claude-profile"}
	result := rawArgs()
	assert.Nil(t, result)
}
