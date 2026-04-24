package cli

// Flag and argument constants used across the CLI. Centralised here so that
// every piece of code that matches, strips, or emits these flag strings
// references a single source of truth.
//
// These constants represent the literal strings as they appear on the command
// line (e.g. "--resume", not "resume"). They are used in two places:
//
//  1. extractClaudeArgs() — to identify and strip (or replace) flags that
//     belong to claude-profile before passing the remaining args to claude.
//
//  2. extractResumeID() — to detect resume-related flags in the arg list
//     and extract the session ID.
//
// Note: cobra flag registration in root.go uses bare names ("profile", "P")
// without the "--"/"-" prefix — those are cobra API conventions, not raw arg
// strings, so they don't use these constants.
const (
	// flagProfile is the long form of claude-profile's own profile flag.
	// Appears as: --profile <name>
	flagProfile = "--profile"

	// flagProfileShort is the short form of the profile flag.
	// Appears as: -P <name> or -P<name> (no space)
	flagProfileShort = "-P"

	// flagProfilePrefix is the equals-sign form of the profile flag.
	// Appears as: --profile=<name>
	flagProfilePrefix = "--profile="

	// flagResume is Claude Code's --resume flag, which we intercept to
	// validate the working directory before passing through.
	// Appears as: --resume <id> or --resume=<id>
	flagResume = "--resume"

	// flagResumeShort is the short form of Claude Code's resume flag.
	// Appears as: -r <id> or -r<id> (no space)
	flagResumeShort = "-r"

	// flagResumePrefix is the equals-sign form of the resume flag.
	// Appears as: --resume=<id>
	flagResumePrefix = "--resume="
)
