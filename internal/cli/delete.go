package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/diranged/claude-profile-go/internal/profile"
	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <profile>",
		Short: "Delete a saved profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			p := profile.Load(name)

			if !p.Exists() {
				return fmt.Errorf("profile %q does not exist", name)
			}

			if !force {
				fmt.Fprintf(cmd.OutOrStdout(), "This will delete profile %q:\n", name)
				fmt.Fprintf(cmd.OutOrStdout(), "  Directory: %s\n", p.Dir)
				fmt.Fprintf(cmd.OutOrStdout(), "  Keychain:  %s\n", p.ServiceKey)
				fmt.Fprint(cmd.OutOrStdout(), "Are you sure? [y/N] ")

				reader := bufio.NewReader(os.Stdin)
				input, _ := reader.ReadString('\n')
				if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(input)), "y") {
					fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
					return nil
				}
			}

			if err := p.Delete(); err != nil {
				return fmt.Errorf("deleting profile: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Profile %q deleted.\n", name)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation prompt")
	return cmd
}
