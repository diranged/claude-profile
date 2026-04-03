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
			_, _ = fmt.Fprintf(out, "Profile:          %s\nConfig dir:       %s\nKeychain service: %s\nAuth:             %s\n",
				name, p.ConfigDir, p.ServiceKey, p.AuthStatus())

			if info, err := p.OAuthDetails(); err == nil {
				_, _ = fmt.Fprintf(out, "Subscription:     %s\nRate limit tier:  %s\n",
					info.SubscriptionType, info.RateLimitTier)
				if info.ExpiresAt > 0 {
					_, _ = fmt.Fprintf(out, "Expires:          %s\n", time.UnixMilli(info.ExpiresAt).Format(time.RFC3339))
				}
				if len(info.Scopes) > 0 {
					_, _ = fmt.Fprintf(out, "Scopes:           %s\n", strings.Join(info.Scopes, ", "))
				}
			}

			return nil
		},
	}
}
