package cli

// Flag and argument constants used across the CLI. Centralised here to avoid
// magic strings scattered through the arg-parsing and validation code.
const (
	// claude-profile's own flags
	flagProfile        = "--profile"
	flagProfileShort   = "-P"
	flagProfilePrefix  = "--profile="
	flagResumeAnywhere = "--resume-anywhere"

	// Claude Code flags that we intercept
	flagResume       = "--resume"
	flagResumeShort  = "-r"
	flagResumePrefix = "--resume="
)
