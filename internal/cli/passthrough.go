package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"

	"golang.org/x/term"

	"github.com/diranged/claude-profile-go/internal/claude"
	"github.com/diranged/claude-profile-go/internal/profile"
	"github.com/diranged/claude-profile-go/internal/sessions"
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

	// Check --resume flag and validate that cwd matches the session's recorded
	// working directory, unless --resume-anywhere was specified.
	resumeID := extractResumeID(claudeArgs)
	resumeAnywhere := hasResumeAnywhereFlag(rawArgs())

	if resumeID != "" && !resumeAnywhere {
		if err := validateResumeCwd(p, resumeID); err != nil {
			return err
		}
	}

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

	// When the profile has SSO credentials, strip provider flags that would
	// override the auth method. The user's intent is to use the Anthropic API.
	if authStatus == "keychain" || authStatus == "file" {
		env = claude.UnsetEnv(env, "CLAUDE_CODE_USE_BEDROCK")
		env = claude.UnsetEnv(env, "CLAUDE_CODE_USE_VERTEX")
	}

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
		// Strip --resume-anywhere (our flag, not claude's)
		if arg == "--resume-anywhere" {
			continue
		}
		result = append(result, arg)
	}
	return result
}

// Version is set at build time via -ldflags (e.g., -ldflags "-X ...cli.Version=1.2.3").
// Falls back to VCS info embedded by Go, or "dev" if neither is available.
var Version string

func init() {
	if Version == "" {
		Version = buildVersion()
	}
}

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

// extractResumeID returns the session ID from --resume/−r args, or "" if
// the flag is absent or bare (no ID provided — let claude show its picker).
func extractResumeID(args []string) string {
	for i, arg := range args {
		// --resume <id>
		if arg == "--resume" && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			return args[i+1]
		}
		// --resume=<id>
		if strings.HasPrefix(arg, "--resume=") {
			return arg[len("--resume="):]
		}
		// -r <id>
		if arg == "-r" && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			return args[i+1]
		}
		// -r<id> (joined form, only if next char is not -)
		if len(arg) > 2 && arg[:2] == "-r" && arg[2] != '-' {
			return arg[2:]
		}
	}
	return ""
}

// hasResumeAnywhereFlag returns true if --resume-anywhere appears in the args.
func hasResumeAnywhereFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--resume-anywhere" {
			return true
		}
	}
	return false
}

// validateResumeCwd looks up the session by ID prefix and checks that the
// current working directory matches the session's recorded cwd.
func validateResumeCwd(p *profile.Profile, resumeID string) error {
	matches, err := sessions.FindByPrefix(p.ConfigDir, resumeID)
	if err != nil {
		log.Debugw("session lookup failed, passing through", "error", err)
		return nil
	}

	if len(matches) == 0 {
		log.Debugw("session not found in profile, passing through", "resumeID", resumeID)
		return nil
	}

	if len(matches) > 1 {
		msg := fmt.Sprintf("session prefix %q matches multiple sessions:\n", resumeID)
		for _, s := range matches {
			branch := ""
			if s.GitBranch != "" {
				branch = s.GitBranch
			}
			msg += fmt.Sprintf("  %-8s  %-40s  %-10s  %s\n",
				shortID(s.ID), s.Cwd, branch, s.ModTime.Format("2006-01-02 15:04"))
		}
		return fmt.Errorf("%s", msg)
	}

	session := matches[0]
	if session.Cwd == "" {
		log.Debugw("session has no recorded cwd, passing through", "resumeID", resumeID)
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}

	sessionCwd := resolvePath(session.Cwd)
	currentCwd := resolvePath(cwd)

	if sessionCwd == currentCwd {
		return nil
	}

	profileName := p.Name
	branch := ""
	if session.GitBranch != "" {
		branch = fmt.Sprintf("\n  Branch:       %s", session.GitBranch)
	}
	prompt := ""
	if session.FirstPrompt != "" {
		prompt = fmt.Sprintf("\n  First prompt: %s", session.FirstPrompt)
	}

	return fmt.Errorf(
		"session %s was started in a different directory.\n\n"+
			"  Session cwd:  %s\n"+
			"  Current cwd:  %s%s%s\n\n"+
			"  To resume, cd to the correct directory:\n"+
			"    cd %s && claude-profile -P %s --resume %s\n\n"+
			"  Or force-resume from this directory:\n"+
			"    claude-profile -P %s --resume-anywhere %s",
		shortID(session.ID),
		session.Cwd,
		cwd, branch, prompt,
		session.Cwd, profileName, shortID(session.ID),
		profileName, shortID(session.ID),
	)
}

// shortID returns the first 8 characters of a session UUID for display.
func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// resolvePath returns the canonical absolute path, resolving symlinks.
func resolvePath(p string) string {
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		return filepath.Clean(p)
	}
	return filepath.Clean(resolved)
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
