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
			auth := p.AuthStatus()
			ac := authColor(auth)

			_, _ = fmt.Fprintf(out, "\n%s%s PROFILE: %s%s\n\n", colorBold, colorGreen, name, colorReset)
			_, _ = fmt.Fprintf(out, "  %sConfig dir%s       %s\n", colorCyan, colorReset, p.ConfigDir)
			_, _ = fmt.Fprintf(out, "  %sKeychain service%s %s\n", colorCyan, colorReset, p.ServiceKey)
			_, _ = fmt.Fprintf(out, "  %sAuth%s             %s%s%s\n", colorCyan, colorReset, ac, auth, colorReset)

			if info, err := p.OAuthDetails(); err == nil {
				_, _ = fmt.Fprintf(out, "  %sSubscription%s     %s\n", colorCyan, colorReset, info.SubscriptionType)
				_, _ = fmt.Fprintf(out, "  %sRate limit tier%s  %s\n", colorCyan, colorReset, info.RateLimitTier)
				if info.ExpiresAt > 0 {
					_, _ = fmt.Fprintf(out, "  %sExpires%s          %s\n", colorCyan, colorReset, time.UnixMilli(info.ExpiresAt).Format(time.RFC3339))
				}
				if len(info.Scopes) > 0 {
					_, _ = fmt.Fprintf(out, "  %sScopes%s           %s\n", colorCyan, colorReset, strings.Join(info.Scopes, ", "))
				}
			}

			_, _ = fmt.Fprintln(out)
			return nil
		},
	}
}
