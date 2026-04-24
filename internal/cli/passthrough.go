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
	// working directory.
	resumeID := extractResumeID(claudeArgs)
	if resumeID != "" {
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
		if arg == flagProfileShort || arg == flagProfile {
			// Next arg is the value — skip it too
			if i+1 < len(args) {
				skip = true
			}
			continue
		}
		// Handle --profile=value form
		if len(arg) > len(flagProfilePrefix) && arg[:len(flagProfilePrefix)] == flagProfilePrefix {
			continue
		}
		// Handle -P<value> (no space) form
		if len(arg) > len(flagProfileShort) && arg[:len(flagProfileShort)] == flagProfileShort && arg[len(flagProfileShort)] != '-' {
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

// extractResumeID scans the arg list for Claude Code's --resume / -r flag
// and returns the session ID that follows it, or "" if:
//   - The flag is not present at all
//   - The flag is bare (no ID after it) — this means "show the session picker",
//     which we let claude handle natively
//
// All four syntactic forms are handled:
//
//	--resume <id>     long form with space separator
//	--resume=<id>     long form with equals sign
//	-r <id>           short form with space separator
//	-r<id>            short form with joined value (no space)
//
// The next-arg check (!strings.HasPrefix(args[i+1], "-")) prevents treating
// another flag as the session ID. This means bare --resume (no ID, followed
// by another flag like --model) correctly returns "".
func extractResumeID(args []string) string {
	for i, arg := range args {
		// --resume <id> (long form, space-separated)
		if arg == flagResume && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			return args[i+1]
		}
		// --resume=<id> (long form, equals-sign)
		if strings.HasPrefix(arg, flagResumePrefix) {
			return arg[len(flagResumePrefix):]
		}
		// -r <id> (short form, space-separated)
		if arg == flagResumeShort && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			return args[i+1]
		}
		// -r<id> (short form, joined value — e.g. "-rabc123")
		if len(arg) > len(flagResumeShort) && arg[:len(flagResumeShort)] == flagResumeShort && arg[len(flagResumeShort)] != '-' {
			return arg[len(flagResumeShort):]
		}
	}
	return ""
}

// validateResumeCwd is the core of the --resume directory safety check.
// Given a session ID (or prefix), it:
//
//  1. Looks up matching sessions in the profile's config directory.
//  2. If no matches: logs a debug message and returns nil (pass through to
//     claude, which may have its own lookup logic).
//  3. If multiple matches: returns an error listing all candidates so the
//     user can pick a more specific prefix.
//  4. If exactly one match: compares the session's recorded cwd with the
//     current working directory (both resolved through symlinks). If they
//     match, returns nil. If they differ, returns a formatted error with
//     the correct cd command.
//
// The function is intentionally lenient: when in doubt (lookup error, no cwd
// recorded, can't determine current dir), it passes through rather than
// blocking the user. The safety check is advisory, not a hard gate.
func validateResumeCwd(p *profile.Profile, resumeID string) error {
	matches, err := sessions.FindByPrefix(p.ConfigDir, resumeID)
	if err != nil {
		// Lookup failure (e.g. permissions, corrupt directory) — don't block
		// the user, just log and let claude try its own resolution.
		log.Debugw("session lookup failed, passing through", "error", err)
		return nil
	}

	// No matches in this profile's sessions. The session might exist in a
	// different profile, or claude may have its own lookup path. Pass through.
	if len(matches) == 0 {
		log.Debugw("session not found in profile, passing through", "resumeID", resumeID)
		return nil
	}

	// Multiple matches — the prefix is ambiguous. Show the user all candidates
	// so they can use a longer prefix to disambiguate.
	if len(matches) > 1 {
		return formatAmbiguousResumeError(resumeID, matches)
	}

	// Exactly one match — check if the cwd lines up.
	session := matches[0]

	// If the session has no recorded cwd (rare edge case — JSONL with no user
	// record), we can't validate, so pass through.
	if session.Cwd == "" {
		log.Debugw("session has no recorded cwd, passing through", "resumeID", resumeID)
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		// Can't determine current directory — don't block, pass through.
		return nil
	}

	// Compare paths after resolving symlinks and cleaning, so that
	// /var → /private/var (macOS) and trailing slashes don't cause
	// false mismatches.
	if resolvePath(session.Cwd) == resolvePath(cwd) {
		return nil
	}

	// Cwd mismatch — build a helpful error with the correct cd command.
	return formatCwdMismatchError(session, cwd, p.Name)
}

// formatAmbiguousResumeError builds the error shown when a session ID prefix
// matches more than one session. It lists all candidates with their cwd,
// branch, and timestamp so the user can pick a more specific prefix.
//
// Example output:
//
//	session prefix "abc" matches multiple sessions:
//	  abc12345  /Users/matt/git/myorg/api    main  2026-04-15 14:30
//	  abc99999  /Users/matt/git/personal/blog main  2026-04-13 20:45
func formatAmbiguousResumeError(prefix string, matches []sessions.Session) error {
	var b strings.Builder
	fmt.Fprintf(&b, "session prefix %q matches multiple sessions:\n", prefix)
	for _, s := range matches {
		fmt.Fprintf(&b, "  %-8s  %-40s  %-10s  %s\n",
			shortID(s.ID), s.Cwd, s.GitBranch, s.ModTime.Format("2006-01-02 15:04"))
	}
	return fmt.Errorf("%s", b.String())
}

// formatCwdMismatchError builds the error shown when the user tries to resume
// a session from the wrong directory. The error includes:
//   - The session's recorded cwd and the user's current cwd
//   - The git branch and first prompt (if available) for identification
//   - A copy-pasteable cd + resume command to get to the right place
//
// Example output:
//
//	session abc12345 was started in a different directory.
//
//	  Session cwd:  /Users/matt/git/myorg/api
//	  Current cwd:  /Users/matt/git/personal/blog
//	  Branch:       main
//	  First prompt: fix the flaky integration test
//
//	  To resume, cd to the correct directory:
//	    cd /Users/matt/git/myorg/api && claude-profile -P me --resume abc12345
func formatCwdMismatchError(session sessions.Session, currentCwd, profileName string) error {
	sid := shortID(session.ID)

	var b strings.Builder
	fmt.Fprintf(&b, "session %s was started in a different directory.\n\n", sid)
	fmt.Fprintf(&b, "  Session cwd:  %s\n", session.Cwd)
	fmt.Fprintf(&b, "  Current cwd:  %s\n", currentCwd)
	if session.GitBranch != "" {
		fmt.Fprintf(&b, "  Branch:       %s\n", session.GitBranch)
	}
	if session.FirstPrompt != "" {
		fmt.Fprintf(&b, "  First prompt: %s\n", session.FirstPrompt)
	}
	fmt.Fprintf(&b, "\n  To resume, cd to the correct directory:\n")
	fmt.Fprintf(&b, "    cd %s && claude-profile -P %s %s %s", session.Cwd, profileName, flagResume, sid)

	return fmt.Errorf("%s", b.String())
}

// shortID returns the first 8 characters of a session UUID for display.
// Session UUIDs are full v4 UUIDs (36 chars), but 8 characters is enough
// to identify them uniquely in practice and keeps the output compact.
// If the ID is already 8 chars or shorter, it's returned as-is.
func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// resolvePath returns the canonical absolute path after resolving symlinks.
// This is used when comparing the session's recorded cwd with the user's
// current cwd to avoid false mismatches caused by:
//   - macOS /var → /private/var symlink
//   - User-created symlinks to repos
//   - Trailing slashes or . / .. components
//
// If symlink resolution fails (e.g. the path doesn't exist), falls back
// to filepath.Clean which at least normalises the path string.
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
