package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"

	"github.com/spf13/cobra"
)

// newStatuslineCmd builds the "statusline" subcommand, which acts as a Claude
// Code statusline provider. Claude Code invokes the configured statusline
// command, piping JSON session data to stdin and reading display text from
// stdout.
//
// This command:
//  1. Reads all of stdin (the JSON session blob Claude provides).
//  2. Prints a colored line showing the active profile name, auth status, and
//     subscription type (read from CLAUDE_PROFILE_* env vars set at launch).
//  3. Optionally chains to a wrapped command (everything after "--") by piping
//     the same stdin JSON into it, allowing the user's original statusline tool
//     to still function.
//
// DisableFlagParsing is set so that flags after "--" are not consumed by cobra.
func newStatuslineCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "statusline [-- command args...]",
		Short: "Statusline wrapper that prepends profile info",
		Long: `Use as your Claude Code statusLine command to display profile info
alongside your existing statusline tool.

Configure in settings.json:
  "statusLine": {
    "type": "command",
    "command": "claude-profile statusline -- bunx -y ccstatusline@latest"
  }

Reads Claude's JSON session data from stdin, outputs a profile info line,
then passes the JSON through to the wrapped command (if any).

Profile info is read from environment variables set by claude-profile
at launch time (CLAUDE_PROFILE_NAME, CLAUDE_PROFILE_AUTH, CLAUDE_PROFILE_SUB).`,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Read all of stdin (Claude's JSON session data)
			input, _ := io.ReadAll(os.Stdin)

			// Output our profile info line
			name := os.Getenv("CLAUDE_PROFILE_NAME")
			auth := os.Getenv("CLAUDE_PROFILE_AUTH")
			sub := os.Getenv("CLAUDE_PROFILE_SUB")

			if name != "" {
				colorCode := 108
				if c := os.Getenv("CLAUDE_PROFILE_COLOR"); c != "" {
					if v, err := strconv.Atoi(c); err == nil && v >= 0 && v <= 255 {
						colorCode = v
					}
				}
				color := fmt.Sprintf("\033[38;5;%dm", colorCode)
				reset := "\033[0m"
				fmt.Printf("%sProfile: %s | Auth: %s | Subscription: %s%s\n",
					color, name, auth, sub, reset)
			}

			// Strip leading "--" if present
			if len(args) > 0 && args[0] == "--" {
				args = args[1:]
			}

			// Chain to wrapped command if provided
			if len(args) > 0 {
				child := exec.Command(args[0], args[1:]...)
				child.Stdin = bytes.NewReader(input)
				child.Stdout = os.Stdout
				child.Stderr = os.Stderr
				_ = child.Run()
			}

			return nil
		},
	}
}
