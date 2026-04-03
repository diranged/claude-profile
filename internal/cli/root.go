// Package cli implements the cobra command tree for claude-profile.
//
// The CLI has two modes of operation:
//
//  1. Subcommands (login, list, show, delete) — manage profiles directly.
//  2. Passthrough mode — when no subcommand matches, all arguments are forwarded
//     to the real claude binary inside a PTY with a status bar.
//
// Profile selection uses -p/--profile flag or CLAUDE_PROFILE env var.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// log is the package-level structured logger. It is initialized by initLogger
	// and defaults to warn-level output unless CLAUDE_PROFILE_DEBUG is set.
	log *zap.SugaredLogger

	// profileFlag holds the value of the -p/--profile flag, bound via cobra's
	// PersistentFlags. It is also populated from the CLAUDE_PROFILE env var
	// through viper.
	profileFlag string
)

// newRootCmd builds the top-level cobra command for claude-profile.
//
// The root command doubles as the passthrough handler: when no subcommand
// matches, runPassthrough exec's the real claude binary with the resolved
// profile. Subcommands (create, list, show, delete, statusline) are registered
// as children.
//
// FParseErrWhitelist.UnknownFlags is enabled so that flags intended for the
// real claude binary (e.g., --model, -c) are not rejected by cobra.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "claude-profile",
		Short: "Run Claude Code with isolated subscription profiles",
		Long: `claude-profile is a transparent wrapper around Claude Code that enables
multiple subscription profiles. Each profile gets its own isolated config
directory and keychain entry, allowing concurrent sessions across different
Claude subscriptions.

Use as a drop-in replacement for claude:
  claude-profile -p work [claude args...]
  CLAUDE_PROFILE=work claude-profile [claude args...]

Or manage profiles directly:
  claude-profile login work
  claude-profile list`,

		// When no subcommand matches, treat all args as claude passthrough.
		// This is the primary usage mode — the user runs claude-profile exactly
		// like they would run claude, with an added -p flag.
		RunE:          runPassthrough,
		SilenceUsage:  true,
		SilenceErrors: true,

		// Allow flags intended for claude to pass through without error.
		// Without this, cobra rejects unknown flags like --model or -c.
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},

		// Accept arbitrary positional args so that unknown subcommands
		// (e.g., "remote-control", "mcp") are treated as args passed through
		// to claude via runPassthrough, rather than cobra errors.
		Args: cobra.ArbitraryArgs,
	}

	// Profile flag — the only flag claude-profile owns.
	// All other flags are passed through to claude.
	root.PersistentFlags().StringVarP(&profileFlag, "profile", "p", "",
		"profile name (or set CLAUDE_PROFILE env var)")

	// Bind CLAUDE_PROFILE env var as fallback for --profile flag
	_ = viper.BindEnv("profile", "CLAUDE_PROFILE")
	_ = viper.BindPFlag("profile", root.PersistentFlags().Lookup("profile"))

	root.AddCommand(
		newCreateCmd(),
		newListCmd(),
		newShowCmd(),
		newDeleteCmd(),
		newStatuslineCmd(),
	)

	return root
}

// Execute runs the root command. Called from main().
func Execute() error {
	initLogger()
	cmd := newRootCmd()
	err := cmd.Execute()
	if err != nil {
		// Print the error ourselves since SilenceErrors is on.
		// This avoids cobra printing "Error:" for exit codes from claude
		// passthrough, while still showing our own errors.
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	}
	return err
}

// resolveProfile returns the active profile name from flag or env var,
// or returns an error if neither is set.
func resolveProfile() (string, error) {
	p := viper.GetString("profile")
	if p == "" {
		return "", fmt.Errorf("no profile specified. Use -p <name> or set CLAUDE_PROFILE")
	}
	return p, nil
}

// initLogger configures the package-level zap logger. By default it logs at
// warn level. Setting the CLAUDE_PROFILE_DEBUG environment variable to any
// non-empty value enables debug-level output for troubleshooting.
func initLogger() {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)

	// Respect CLAUDE_PROFILE_DEBUG for verbose output during troubleshooting
	if os.Getenv("CLAUDE_PROFILE_DEBUG") != "" {
		cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}

	logger, err := cfg.Build()
	if err != nil {
		logger = zap.NewNop()
	}
	log = logger.Sugar()
}
