package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/diranged/claude-profile-go/internal/claude"
	"github.com/diranged/claude-profile-go/internal/profile"
	"github.com/spf13/cobra"
)

func newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login <profile> [flags]",
		Short: "Login and save credentials for a profile",
		Long: `Runs 'claude auth login' with an isolated config directory for the
given profile. After login, credentials are stored in the macOS keychain
under a profile-specific service name.

If ~/.claude exists with configuration files (CLAUDE.md, settings.json, etc.),
you'll be offered the option to copy them into the new profile.

Any extra flags are passed through to 'claude auth login' (e.g., --sso, --console).`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			extra := args[1:] // passthrough flags like --sso, --console

			p := profile.Load(name)
			isNew := !p.Exists()

			if err := p.EnsureDir(); err != nil {
				return fmt.Errorf("creating profile directory: %w", err)
			}

			// For new profiles, offer to bootstrap from default config
			if isNew {
				if err := offerBootstrap(p); err != nil {
					return err
				}
			}

			claudeBin, err := claude.FindBinary()
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Logging in for profile: %s\n", name)
			fmt.Fprintf(out, "Config directory:       %s\n", p.ConfigDir)
			fmt.Fprintf(out, "Keychain service:       %s\n", p.ServiceKey)
			fmt.Println()

			loginArgs := append([]string{"auth", "login"}, extra...)
			env := claude.BuildEnv(p.ConfigDir)

			exitCode, err := claude.RunDirect(claudeBin, loginArgs, env)
			if err != nil {
				return fmt.Errorf("claude auth login failed: %w", err)
			}
			if exitCode != 0 {
				return fmt.Errorf("claude auth login exited with code %d", exitCode)
			}

			fmt.Println()

			auth := p.AuthStatus()
			switch auth {
			case "keychain":
				fmt.Fprintf(out, "Profile %q credentials saved to keychain (service: %s)\n", name, p.ServiceKey)
			case "file":
				fmt.Fprintf(out, "Profile %q credentials saved to file\n", name)
			default:
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not verify credentials were saved for profile %q\n", name)
			}

			fmt.Fprintf(out, "\nUsage: claude-profile -p %s\n", name)
			return nil
		},
	}
}

// offerBootstrap checks if the default ~/.claude directory has config files
// that could be copied into a new profile, and prompts the user.
func offerBootstrap(p *profile.Profile) error {
	files := profile.BootstrapFiles()
	if len(files) == 0 {
		return nil
	}

	fmt.Printf("Found config files in %s:\n", profile.DefaultConfigDir())
	for _, f := range files {
		fmt.Printf("  - %s\n", f)
	}
	fmt.Print("Copy these into the new profile? [Y/n] ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	// Default to yes (empty input or "y")
	if input == "" || input == "y" || input == "yes" {
		if err := p.CopyBootstrapFiles(files); err != nil {
			return fmt.Errorf("copying config files: %w", err)
		}
		fmt.Printf("Copied %d file(s) into profile.\n\n", len(files))
	} else {
		fmt.Println("Skipped.")
	}

	return nil
}
