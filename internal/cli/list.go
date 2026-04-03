package cli

import (
	"fmt"

	"github.com/diranged/claude-profile-go/internal/profile"
	"github.com/spf13/cobra"
)

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

			if len(names) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No profiles found. Create one with: claude-profile login <name>")
				return nil
			}

			for _, name := range names {
				p := profile.Load(name)
				auth := p.AuthStatus()
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %-20s (%s)\n", name, auth)
			}
			return nil
		},
	}
}
