// Package main is the entrypoint for claude-profile.
//
// claude-profile is a transparent wrapper around Claude Code that enables
// multiple subscription profiles. Each profile gets its own isolated
// CLAUDE_CONFIG_DIR and keychain entry, allowing concurrent sessions
// across different Claude subscriptions (e.g., work and personal).
//
// Usage:
//
//	claude-profile -P work [claude args...]
//	CLAUDE_PROFILE=work claude-profile [claude args...]
//	claude-profile login work
//	claude-profile list
package main

import (
	"os"

	"github.com/diranged/claude-profile-go/internal/cli"
)

// main initializes the CLI and exits with a non-zero status on error.
// All real logic lives in the cli package; main simply delegates to cli.Execute.
func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
