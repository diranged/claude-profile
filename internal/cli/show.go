package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/diranged/claude-profile-go/internal/profile"
	"github.com/spf13/cobra"
)

// newShowCmd builds the "show" subcommand, which prints detailed information
// about a single profile including its config directory path, keychain service
// name, authentication status, and (if OAuth credentials exist) subscription
// type, rate limit tier, token expiry, and granted scopes.
func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <profile>",
		Short: "Show profile details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			p := profile.Load(name)

			if !p.Exists() {
				return fmt.Errorf("profile %q does not exist", name)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Profile:          %s\n", name)
			fmt.Fprintf(out, "Config dir:       %s\n", p.ConfigDir)
			fmt.Fprintf(out, "Keychain service: %s\n", p.ServiceKey)
			fmt.Fprintf(out, "Auth:             %s\n", p.AuthStatus())

			if info, err := p.OAuthDetails(); err == nil {
				fmt.Fprintf(out, "Subscription:     %s\n", info.SubscriptionType)
				fmt.Fprintf(out, "Rate limit tier:  %s\n", info.RateLimitTier)
				if info.ExpiresAt > 0 {
					t := time.UnixMilli(info.ExpiresAt)
					fmt.Fprintf(out, "Expires:          %s\n", t.Format(time.RFC3339))
				}
				if len(info.Scopes) > 0 {
					fmt.Fprintf(out, "Scopes:           %s\n", strings.Join(info.Scopes, ", "))
				}
			}

			return nil
		},
	}
}
