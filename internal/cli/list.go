package cli

import (
	"fmt"

	"github.com/diranged/claude-profile-go/internal/profile"
	"github.com/spf13/cobra"
)

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
				fmt.Fprintln(cmd.OutOrStdout(), "No profiles found. Create one with: claude-profile login <name>")
				return nil
			}

			for _, name := range names {
				p := profile.Load(name)
				auth := p.AuthStatus()
				fmt.Fprintf(cmd.OutOrStdout(), "  %-20s (%s)\n", name, auth)
			}
			return nil
		},
	}
}
