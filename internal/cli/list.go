package cli

import (
	"fmt"

	"github.com/diranged/claude-profile-go/internal/profile"
	"github.com/spf13/cobra"
)

// authColor returns an ANSI color code appropriate for the auth status.
func authColor(status string) string {
	switch status {
	case "keychain":
		return colorGreen
	case "file":
		return colorYellow
	default:
		return colorDim
	}
}

// newListCmd builds the "list" subcommand, which enumerates all profiles found
// in the profiles directory (default: ~/.claude-profiles/) and prints each
// profile's name alongside its authentication status (keychain, file, or none).
func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := profile.List()
			if err != nil {
				return fmt.Errorf("listing profiles: %w", err)
			}

			out := cmd.OutOrStdout()

			if len(names) == 0 {
				_, _ = fmt.Fprintf(out, "\n  No profiles found. Create one with:\n\n")
				_, _ = fmt.Fprintf(out, "    %s$%s claude-profile create <name>\n\n", colorYellow, colorReset)
				return nil
			}

			_, _ = fmt.Fprintf(out, "\n%s%s PROFILES%s\n\n", colorBold, colorGreen, colorReset)
			for _, name := range names {
				p := profile.Load(name)
				auth := p.AuthStatus()
				ac := authColor(auth)
				_, _ = fmt.Fprintf(out, "  %s%-20s%s %s%s%s\n", colorCyan, name, colorReset, ac, auth, colorReset)
			}
			_, _ = fmt.Fprintln(out)
			return nil
		},
	}
}
