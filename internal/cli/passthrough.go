package cli

import (
	"fmt"
	"os"
	"runtime/debug"
	"syscall"

	"golang.org/x/term"

	"github.com/diranged/claude-profile-go/internal/claude"
	"github.com/diranged/claude-profile-go/internal/profile"
	"github.com/spf13/cobra"
)

// runPassthrough is the default command handler. It resolves the active profile,
// prints a banner, and exec's claude (replacing this process).
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

	// Allow passthrough without credentials when:
	// - Running an auth command (e.g., "auth login")
	// - Using Bedrock/Vertex (credentials come from AWS/GCP env, not keychain)
	// - Using an API key directly
	claudeArgs := extractClaudeArgs()
	isAuthCmd := len(claudeArgs) > 0 && claudeArgs[0] == "auth"
	hasExternalAuth := os.Getenv("CLAUDE_CODE_USE_BEDROCK") != "" ||
		os.Getenv("CLAUDE_CODE_USE_VERTEX") != "" ||
		os.Getenv("ANTHROPIC_API_KEY") != ""

	if authStatus == "none" && !isAuthCmd && !hasExternalAuth {
		return fmt.Errorf("profile %q has no credentials. Run: claude-profile -P %s auth login", name, name)
	}

	claudeBin, err := claude.FindBinary()
	if err != nil {
		return err
	}

	// Collect subscription info for the banner
	subType := "unknown"
	if info, err := p.OAuthDetails(); err == nil {
		subType = info.SubscriptionType
	}

	cfg := p.LoadConfig()

	env := claude.BuildEnv(p.ConfigDir)
	env = setEnv(env, "CLAUDE_PROFILE_NAME", name)
	env = setEnv(env, "CLAUDE_PROFILE_AUTH", authStatus)
	env = setEnv(env, "CLAUDE_PROFILE_SUB", subType)
	env = setEnv(env, "CLAUDE_PROFILE_COLOR", fmt.Sprintf("%d", cfg.Color))

	// Print profile banner matching Claude Code's box-drawing style
	printBanner(name, p.ConfigDir, authStatus, subType, cfg.Color)

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
	result := make([]string, 0, len(args))
	skip := false
	for i, arg := range args {
		if skip {
			skip = false
			continue
		}
		// Skip our own -P/--profile flag and its value
		if arg == "-P" || arg == "--profile" {
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
		// Handle -P<value> (no space) form
		if len(arg) > 2 && arg[:2] == "-P" && arg[2] != '-' {
			continue
		}
		result = append(result, arg)
	}
	return result
}

// Version is set at build time via -ldflags (e.g., -ldflags "-X ...cli.Version=1.2.3").
// Falls back to VCS info embedded by Go, or "dev" if neither is available.
var Version = buildVersion()

func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	var revision, dirty string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) >= 7 {
				revision = s.Value[:7]
			}
		case "vcs.modified":
			if s.Value == "true" {
				dirty = "-dirty"
			}
		}
	}
	if revision != "" {
		return "dev-" + revision + dirty
	}
	return "dev"
}

// maxBannerWidth caps the banner width to match Claude Code's own box-drawing
// style, which tops out at 120 columns.
const maxBannerWidth = 120

// printBanner renders a Unicode box-drawing banner to stderr showing the active
// profile's name, config directory, auth status, and subscription type. The
// banner is colored using the profile's configured ANSI 256-color code and
// adapts its width to the terminal (up to maxBannerWidth). Output goes to
// stderr so it does not interfere with Claude's stdout in piped/non-interactive
// usage.
func printBanner(name, configDir, auth, sub string, colorCode int) {
	const (
		reset  = "\033[0m"
		accent = "\033[38;5;173m" // Claude's warm orange/brown
	)
	color := fmt.Sprintf("\033[38;5;%dm", colorCode)

	// Get terminal width, capped to match Claude Code's box width
	width := 80
	if w, _, err := term.GetSize(int(os.Stderr.Fd())); err == nil && w > 0 {
		width = w
	}
	if width > maxBannerWidth {
		width = maxBannerWidth
	}

	title := fmt.Sprintf(" Claude Profile v%s ", Version)
	labelWidth := 16

	lines := []struct{ label, value string }{
		{"Profile", name},
		{"Config", configDir},
		{"Auth", auth},
		{"Subscription", sub},
	}

	innerWidth := width - 2

	// Top border: ╭─── Claude Profile v0.1.0 ───────────────╮
	topDashes := width - 6 - len(title)
	if topDashes < 1 {
		topDashes = 1
	}
	fmt.Fprintf(os.Stderr, "\n%s╭───%s%s%s%s %s╮%s\n",
		color, reset, accent, title, color, repeat("─", topDashes), reset)

	// Content lines
	for _, l := range lines {
		text := fmt.Sprintf(" %s%-*s%s%s", color, labelWidth, l.label+":", reset+color, l.value)
		visibleLen := 1 + labelWidth + len(l.value)
		pad := innerWidth - visibleLen
		if pad < 0 {
			pad = 0
		}
		fmt.Fprintf(os.Stderr, "%s│%s%s%s%s│%s\n",
			color, reset, text, reset+repeat(" ", pad), color, reset)
	}

	// Bottom border
	fmt.Fprintf(os.Stderr, "%s╰%s╯%s\n\n", color, repeat("─", innerWidth), reset)
}

// setEnv sets or replaces an environment variable in an env slice ([]string
// of "KEY=VALUE" entries). If the key already exists, its value is updated
// in-place; otherwise a new entry is appended. This avoids mutating
// os.Environ() directly and lets us build a custom environment for
// syscall.Exec.
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

// repeat returns s concatenated n times. Used for generating box-drawing
// border lines in the banner.
func repeat(s string, n int) string {
	out := ""
	for range n {
		out += s
	}
	return out
}
