package cli

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/term"

	"github.com/diranged/claude-profile-go/internal/claude"
	"github.com/diranged/claude-profile-go/internal/profile"
	"github.com/diranged/claude-profile-go/internal/statusline"
	"github.com/spf13/cobra"
)

// runPassthrough is the default command handler. It resolves the active profile,
// sets up the statusline wrapper, and exec's claude (replacing this process).
func runPassthrough(cmd *cobra.Command, _ []string) error {
	name, err := resolveProfile()
	if err != nil {
		return err
	}

	p := profile.Load(name)
	if !p.Exists() {
		return fmt.Errorf("profile %q does not exist. Run: claude-profile login %s", name, name)
	}

	authStatus := p.AuthStatus()
	if authStatus == "none" {
		return fmt.Errorf("profile %q has no credentials. Run: claude-profile login %s", name, name)
	}

	claudeBin, err := claude.FindBinary()
	if err != nil {
		return err
	}

	// Collect subscription info for the statusline
	subType := "unknown"
	if info, err := p.OAuthDetails(); err == nil {
		subType = info.SubscriptionType
	}

	// Set up the statusline wrapper in the profile's config dir.
	// This writes the wrapper script and updates settings.json.
	_, origCmd, err := statusline.EnsureWrapper(p.ConfigDir)
	if err != nil {
		// Non-fatal: statusline is nice-to-have, not required
		log.Warnw("failed to set up statusline", "error", err)
	}

	// Build environment with profile isolation + statusline env vars
	env := claude.BuildEnv(p.ConfigDir)
	env = setEnv(env, statusline.EnvProfileName, name)
	env = setEnv(env, statusline.EnvProfileAuth, authStatus)
	env = setEnv(env, statusline.EnvProfileSub, subType)
	if origCmd != "" {
		env = setEnv(env, statusline.EnvOrigCommand, origCmd)
	}

	// Build claude args: everything after our flags.
	claudeArgs := extractClaudeArgs()

	// Print profile banner matching Claude Code's box-drawing style
	printBanner(name, p.ConfigDir, authStatus, subType)

	log.Debugw("launching claude",
		"profile", name,
		"configDir", p.ConfigDir,
		"binary", claudeBin,
		"args", claudeArgs,
	)

	// Replace this process with claude. This is the cleanest approach:
	// no PTY wrapper, no output filtering, no scroll region hacks.
	// Claude gets full control of the terminal as it expects.
	argv := append([]string{claudeBin}, claudeArgs...)
	return syscall.Exec(claudeBin, argv, env)
}

// extractClaudeArgs pulls out the args to pass to claude from os.Args.
// It strips the binary name and any -p/--profile flags (with their values)
// that belong to claude-profile, passing everything else through.
func extractClaudeArgs() []string {
	args := rawArgs()
	var result []string
	skip := false
	for i, arg := range args {
		if skip {
			skip = false
			continue
		}
		// Skip our own -p/--profile flag and its value
		if arg == "-p" || arg == "--profile" {
			// Next arg is the value — skip it too
			if i+1 < len(args) {
				skip = true
			}
			continue
		}
		// Handle --profile=value form
		if len(arg) > 10 && arg[:10] == "--profile=" {
			continue
		}
		// Handle -p<value> (no space) form
		if len(arg) > 2 && arg[:2] == "-p" && arg[2] != '-' {
			continue
		}
		result = append(result, arg)
	}
	return result
}

// Version is set at build time via -ldflags.
var Version = "dev"

// printBanner renders a styled profile info box matching Claude Code's aesthetic.
// Uses full terminal width like Claude's "── Claude Code v2.1.91 ──" header.
func printBanner(name, configDir, auth, sub string) {
	const (
		dim    = "\033[2m"
		reset  = "\033[0m"
		accent = "\033[38;5;173m" // Claude's warm orange/brown
	)

	// Get terminal width, fall back to 80
	width := 80
	if w, _, err := term.GetSize(int(os.Stderr.Fd())); err == nil && w > 0 {
		width = w
	}

	title := fmt.Sprintf(" Claude Profile v%s ", Version)
	labelWidth := 16 // "Subscription:   " padded

	lines := []struct{ label, value string }{
		{"Profile", name},
		{"Config", configDir},
		{"Auth", auth},
		{"Subscription", sub},
	}

	// Inner width is terminal width minus the two border chars
	innerWidth := width - 2

	green := "\033[38;5;108m" // Muted sage green

	// Top border: ╭─── Claude Profile v0.1.0 ───────────────╮
	// Visual width: ╭(1) + ───(3) + title + space(1) + dashes(N) + ╮(1) = width
	topDashes := width - 6 - len(title)
	if topDashes < 1 {
		topDashes = 1
	}
	fmt.Fprintf(os.Stderr, "\n%s╭───%s%s%s%s %s╮%s\n",
		green, reset, accent, title, green, repeat("─", topDashes), reset)

	// Content lines padded to full width
	// Visual width: │(1) + text(innerWidth) + │(1) = width
	for _, l := range lines {
		text := fmt.Sprintf(" %s%-*s%s%s", green, labelWidth, l.label+":", reset+green, l.value)
		visibleLen := 1 + labelWidth + len(l.value)
		pad := innerWidth - visibleLen
		if pad < 0 {
			pad = 0
		}
		fmt.Fprintf(os.Stderr, "%s│%s%s%s%s│%s\n",
			green, reset, text, reset+repeat(" ", pad), green, reset)
	}

	// Bottom border: ╰─────────────────────────────────────────╯
	fmt.Fprintf(os.Stderr, "%s╰%s╯%s\n\n", green, repeat("─", innerWidth), reset)
}

func repeat(s string, n int) string {
	out := ""
	for range n {
		out += s
	}
	return out
}

// setEnv sets or replaces an environment variable in an env slice.
func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if len(e) >= len(prefix) && e[:len(prefix)] == prefix {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}
