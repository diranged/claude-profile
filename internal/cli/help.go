package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// ANSI color codes used in the help output.
const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
	colorGreen  = "\033[38;5;108m"
	colorCyan   = "\033[36m"
	colorYellow = "\033[33m"
	colorAccent = "\033[38;5;173m" // warm orange/brown matching Claude Code
)

// setCustomHelp overrides the default cobra help template with a colored,
// well-structured version that matches the visual style of the banner.
func setCustomHelp(root *cobra.Command) {
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		out := cmd.OutOrStdout()

		// Title
		_, _ = fmt.Fprintf(out, "\n%s%s Claude Profile v%s %s\n", colorBold, colorAccent, Version, colorReset)
		_, _ = fmt.Fprintf(out, "%s%s%s\n\n", colorDim, strings.Repeat("─", 50), colorReset)

		// Description
		_, _ = fmt.Fprintf(out, "  Transparent wrapper around Claude Code that enables multiple\n")
		_, _ = fmt.Fprintf(out, "  subscription profiles with isolated config and credentials.\n\n")

		// Quick start
		_, _ = fmt.Fprintf(out, "%s%s QUICK START%s\n\n", colorBold, colorGreen, colorReset)
		_, _ = fmt.Fprintf(out, "  %s$%s claude-profile create work          %s# set up a new profile%s\n", colorYellow, colorReset, colorDim, colorReset)
		_, _ = fmt.Fprintf(out, "  %s$%s claude-profile -P work              %s# launch Claude with profile%s\n", colorYellow, colorReset, colorDim, colorReset)
		_, _ = fmt.Fprintf(out, "  %s$%s CLAUDE_PROFILE=work claude-profile  %s# same, via env var%s\n\n", colorYellow, colorReset, colorDim, colorReset)

		// Profile management
		_, _ = fmt.Fprintf(out, "%s%s COMMANDS%s\n\n", colorBold, colorGreen, colorReset)

		commands := []struct {
			name, desc string
		}{
			{"create <profile>", "Set up a new profile (wizard)"},
			{"list", "List all profiles and their auth status"},
			{"show <profile>", "Show detailed profile info"},
			{"delete <profile>", "Remove a profile and its credentials"},
			{"statusline", "Statusline wrapper for Claude Code"},
		}
		for _, c := range commands {
			_, _ = fmt.Fprintf(out, "  %s%-22s%s %s\n", colorCyan, c.name, colorReset, c.desc)
		}

		_, _ = fmt.Fprintf(out, "\n%s%s FLAGS%s\n\n", colorBold, colorGreen, colorReset)
		_, _ = fmt.Fprintf(out, "  %s-P, --profile%s <name>   Profile to use (or set %sCLAUDE_PROFILE%s)\n", colorCyan, colorReset, colorYellow, colorReset)
		_, _ = fmt.Fprintf(out, "  %s-h, --help%s             Show this help\n", colorCyan, colorReset)

		_, _ = fmt.Fprintf(out, "\n%s%s AUTH METHODS%s\n\n", colorBold, colorGreen, colorReset)
		_, _ = fmt.Fprintf(out, "  All Claude auth methods work — just launch with your profile:\n\n")
		_, _ = fmt.Fprintf(out, "  %sOAuth/SSO%s      claude-profile -P work        %s# then /login inside Claude%s\n", colorBold, colorReset, colorDim, colorReset)
		_, _ = fmt.Fprintf(out, "  %sCLI auth%s       claude-profile -P work auth login [--sso]\n", colorBold, colorReset)
		_, _ = fmt.Fprintf(out, "  %sAPI key%s        %sANTHROPIC_API_KEY%s=sk-... claude-profile -P work\n", colorBold, colorReset, colorYellow, colorReset)
		_, _ = fmt.Fprintf(out, "  %sBedrock%s        %sCLAUDE_CODE_USE_BEDROCK%s=1 claude-profile -P work\n", colorBold, colorReset, colorYellow, colorReset)
		_, _ = fmt.Fprintf(out, "  %sVertex%s         %sCLAUDE_CODE_USE_VERTEX%s=1 claude-profile -P work\n", colorBold, colorReset, colorYellow, colorReset)

		_, _ = fmt.Fprintf(out, "\n%s%s ENVIRONMENT%s\n\n", colorBold, colorGreen, colorReset)
		_, _ = fmt.Fprintf(out, "  %sCLAUDE_PROFILE%s       Active profile name (alternative to -p)\n", colorYellow, colorReset)
		_, _ = fmt.Fprintf(out, "  %sCLAUDE_PROFILES_DIR%s  Profiles directory (default: ~/.claude-profiles)\n", colorYellow, colorReset)

		_, _ = fmt.Fprintf(out, "\n%s%s EXAMPLES%s\n\n", colorBold, colorGreen, colorReset)
		_, _ = fmt.Fprintf(out, "  %s# Create and authenticate a work profile%s\n", colorDim, colorReset)
		_, _ = fmt.Fprintf(out, "  %s$%s claude-profile create work\n", colorYellow, colorReset)
		_, _ = fmt.Fprintf(out, "  %s$%s claude-profile -P work auth login\n\n", colorYellow, colorReset)
		_, _ = fmt.Fprintf(out, "  %s# Run Claude with the work profile%s\n", colorDim, colorReset)
		_, _ = fmt.Fprintf(out, "  %s$%s claude-profile -P work\n\n", colorYellow, colorReset)
		_, _ = fmt.Fprintf(out, "  %s# Pass args through to Claude%s\n", colorDim, colorReset)
		_, _ = fmt.Fprintf(out, "  %s$%s claude-profile -P work --model opus -p /path/to/project\n\n", colorYellow, colorReset)
		_, _ = fmt.Fprintf(out, "  %s# See what profiles exist%s\n", colorDim, colorReset)
		_, _ = fmt.Fprintf(out, "  %s$%s claude-profile list\n\n", colorYellow, colorReset)

		_, _ = fmt.Fprintf(out, "  %sUse \"claude-profile [command] --help\" for more info on a command.%s\n\n", colorDim, colorReset)
	})
}
