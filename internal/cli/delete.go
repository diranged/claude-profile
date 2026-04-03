package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/diranged/claude-profile-go/internal/profile"
	"github.com/spf13/cobra"
)

// newDeleteCmd builds the "delete" subcommand, which removes a profile's
// directory tree and its macOS keychain entry. By default it prompts for
// confirmation; the -f/--force flag skips the prompt.
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

			out := cmd.OutOrStdout()

			if !force {
				_, _ = fmt.Fprintf(out, "\n  This will delete profile %s%s%s:\n\n", colorBold, name, colorReset)
				_, _ = fmt.Fprintf(out, "    %sDirectory:%s %s\n", colorCyan, colorReset, p.Dir)
				_, _ = fmt.Fprintf(out, "    %sKeychain:%s  %s\n\n", colorCyan, colorReset, p.ServiceKey)
				_, _ = fmt.Fprintf(out, "  Are you sure? %s[y/N]%s ", colorDim, colorReset)

				reader := bufio.NewReader(os.Stdin)
				input, _ := reader.ReadString('\n')
				if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(input)), "y") {
					_, _ = fmt.Fprintf(out, "  %s— Cancelled%s\n\n", colorDim, colorReset)
					return nil
				}
			}

			if err := p.Delete(); err != nil {
				return fmt.Errorf("deleting profile: %w", err)
			}

			_, _ = fmt.Fprintf(out, "  %s✓%s Profile %q deleted\n\n", colorGreen, colorReset, name)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation prompt")
	return cmd
}
