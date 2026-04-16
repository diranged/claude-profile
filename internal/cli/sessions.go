package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/diranged/claude-profile-go/internal/profile"
	"github.com/diranged/claude-profile-go/internal/sessions"
	"github.com/spf13/cobra"
)

// sessionGroup holds sessions that share the same working directory (cwd).
// The sessions command groups output by cwd so users can quickly scan
// which repos have recent activity.
type sessionGroup struct {
	// cwd is the working directory shared by all sessions in this group.
	// Displayed as the group heading (e.g. "=== /Users/matt/git/myapp ===").
	cwd string

	// sessions is the list of sessions in this group, already sorted by
	// ModTime descending (newest first) from FindByPrefix.
	sessions []sessions.Session
}

// newSessionsCmd builds the "sessions" subcommand, which lists all Claude Code
// sessions across every repository for the active profile.
//
// This solves the problem of not knowing which repo a session belongs to:
// when you work across many repos, you create sessions everywhere and later
// can't find them. This command shows everything in one place, grouped by
// directory, with the session's slug (pretty name) and first prompt to help
// identify each one.
//
// Flags:
//   - --since <duration>: only show sessions modified within this time window
//     (default "7d"). Supports Go durations (24h, 30m) and a custom "Nd" day
//     format (7d, 30d).
//   - --repo <substring>: case-insensitive filter on the session's cwd path.
//     Useful for narrowing to a specific project (e.g. --repo sproutbook).
func newSessionsCmd() *cobra.Command {
	var sinceFlag string
	var repoFlag string

	cmd := &cobra.Command{
		Use:   "sessions",
		Short: "List sessions across all repos",
		Long: `List all Claude Code sessions for the active profile, grouped by repository.
Shows session ID, git branch, timestamp, and first prompt for each session.

Requires a profile to be selected via -P or CLAUDE_PROFILE.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Resolve and validate the active profile — we need its ConfigDir
			// to know where session files are stored.
			name, err := resolveProfile()
			if err != nil {
				return err
			}

			p := profile.Load(name)
			if !p.Exists() {
				return fmt.Errorf("profile %q does not exist", name)
			}

			// Parse the --since flag into a Go duration. This supports both
			// standard Go durations ("24h", "30m") and our custom "Nd" day
			// format ("7d", "30d") via parseDuration().
			since, err := parseDuration(sinceFlag)
			if err != nil {
				return fmt.Errorf("invalid --since value %q: %w", sinceFlag, err)
			}

			// Fetch ALL sessions (empty prefix = match everything). Filtering
			// by time and repo happens in groupSessionsByCwd below.
			allSessions, err := sessions.FindByPrefix(p.ConfigDir, "")
			if err != nil {
				return fmt.Errorf("listing sessions: %w", err)
			}

			// Filter and group sessions by working directory.
			cutoff := time.Now().Add(-since)
			groups := groupSessionsByCwd(allSessions, cutoff, repoFlag)

			// Handle the empty case with a friendly message.
			out := cmd.OutOrStdout()
			if len(groups) == 0 {
				_, _ = fmt.Fprintf(out, "\n  No sessions found")
				if sinceFlag != "" {
					_, _ = fmt.Fprintf(out, " in the last %s", sinceFlag)
				}
				_, _ = fmt.Fprintf(out, ".\n\n")
				return nil
			}

			// Render the grouped session list with colors and formatting.
			printSessionGroups(cmd.OutOrStdout(), groups)
			return nil
		},
	}

	cmd.Flags().StringVar(&sinceFlag, "since", "7d", "show sessions modified within this duration (e.g. 1d, 12h, 30d)")
	cmd.Flags().StringVar(&repoFlag, "repo", "", "filter by repo path substring (case-insensitive)")

	return cmd
}

// groupSessionsByCwd takes a flat list of sessions and organises them into
// groups by working directory (cwd), applying time and repo filters.
//
// The grouping logic:
//  1. Skip sessions older than the cutoff time (--since filter).
//  2. Skip sessions whose cwd doesn't contain the repoFilter substring
//     (--repo filter, case-insensitive).
//  3. Group remaining sessions by cwd, preserving the original sort order
//     (newest first within each group, from FindByPrefix).
//  4. Sort groups alphabetically by cwd path so the output is stable and
//     easy to scan.
//
// Sessions with no recorded cwd (rare — only if the JSONL has no user record)
// are grouped under "(unknown)".
func groupSessionsByCwd(all []sessions.Session, cutoff time.Time, repoFilter string) []sessionGroup {
	var groups []sessionGroup

	// groupIdx maps cwd → index in the groups slice, so we can append
	// sessions to existing groups in O(1) instead of scanning the slice.
	groupIdx := make(map[string]int)

	for _, s := range all {
		// Apply the --since time filter.
		if s.ModTime.Before(cutoff) {
			continue
		}
		// Apply the --repo substring filter (case-insensitive).
		if repoFilter != "" && !containsCI(s.Cwd, repoFilter) {
			continue
		}

		// Normalise empty cwd to a placeholder for display.
		cwd := s.Cwd
		if cwd == "" {
			cwd = "(unknown)"
		}

		// Either append to an existing group or create a new one.
		if idx, ok := groupIdx[cwd]; ok {
			groups[idx].sessions = append(groups[idx].sessions, s)
		} else {
			groupIdx[cwd] = len(groups)
			groups = append(groups, sessionGroup{cwd: cwd, sessions: []sessions.Session{s}})
		}
	}

	// Sort groups alphabetically by cwd so output is deterministic and
	// easy to scan visually (repos from the same org cluster together).
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].cwd < groups[j].cwd
	})
	return groups
}

// printSessionGroups renders the grouped session list to the given writer.
// Output format:
//
//	 SESSIONS (21 sessions across 8 repos)
//
//	=== /Users/matt/git/myorg/api ===
//	  abc12345  2026-04-15 14:30  [main]  async-wishing-forest — fix the flaky test
//	  def67890  2026-04-14 09:15  [dev]   peppy-sparking-porcupine — add OAuth2 flow
//
//	=== /Users/matt/git/personal/blog ===
//	  ...
//
// Each group header is the working directory path. Sessions within each group
// are already sorted newest-first from FindByPrefix.
func printSessionGroups(w io.Writer, groups []sessionGroup) {
	// Count total sessions across all groups for the summary line.
	total := 0
	for _, g := range groups {
		total += len(g.sessions)
	}

	// Summary header: " SESSIONS (N sessions across M repos)"
	_, _ = fmt.Fprintf(w, "\n%s%s SESSIONS%s", colorBold, colorGreen, colorReset)
	_, _ = fmt.Fprintf(w, " %s(%d sessions across %d repos)%s\n",
		colorDim, total, len(groups), colorReset)

	// Render each group: heading + session rows.
	for _, g := range groups {
		_, _ = fmt.Fprintf(w, "\n%s=== %s ===%s\n", colorCyan, g.cwd, colorReset)
		for _, s := range g.sessions {
			printSessionLine(w, s)
		}
	}
	_, _ = fmt.Fprintln(w)
}

// printSessionLine renders a single session as one formatted row.
//
// Format:
//
//	  <id>  <timestamp>  [<branch>]  <label>
//
// Where <label> is the session slug (pretty name) if available, with the
// first prompt shown dimmed after it. If no slug exists, the first prompt
// is shown directly. If neither exists, "(no prompt)" is shown.
//
// Example with slug:
//
//	  abc12345  2026-04-15 14:30  [main]  async-wishing-forest — fix the flaky test
//
// Example without slug:
//
//	  abc12345  2026-04-15 14:30  [main]  fix the flaky integration test in auth middleware
func printSessionLine(w io.Writer, s sessions.Session) {
	ts := s.ModTime.Format("2006-01-02 15:04")

	// Format the git branch as "[branchname]" or empty string if none.
	branch := ""
	if s.GitBranch != "" {
		branch = fmt.Sprintf("[%s]", s.GitBranch)
	}

	// Build the label: prefer slug with dimmed prompt, fall back to prompt alone.
	label := s.FirstPrompt
	if s.Slug != "" {
		label = s.Slug
		if s.FirstPrompt != "" {
			// Show the slug prominently with the first prompt dimmed after "—"
			// so users can identify the session by name AND topic.
			label += fmt.Sprintf(" %s— %s%s", colorDim, s.FirstPrompt, colorReset)
		}
	}
	if label == "" {
		label = "(no prompt)"
	}

	// Render the row with fixed-width columns for alignment across sessions.
	// Column widths: ID=8, timestamp=16, branch=14, label=remainder.
	_, _ = fmt.Fprintf(w,
		"  %s%-8s%s  %s%-16s%s  %s%-14s%s  %s\n",
		colorAccent, shortID(s.ID), colorReset,
		colorDim, ts, colorReset,
		colorGreen, branch, colorReset,
		label,
	)
}

// parseDuration parses a human-friendly duration string. It extends Go's
// standard time.ParseDuration with support for a "Nd" day suffix, since
// Go's time package doesn't natively support days.
//
// Supported formats:
//   - "7d"    → 7 days (168 hours)
//   - "30d"   → 30 days
//   - "24h"   → 24 hours (standard Go)
//   - "30m"   → 30 minutes (standard Go)
//   - "2h30m" → 2.5 hours (standard Go)
//   - ""      → zero duration (no filtering)
func parseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	s = strings.TrimSpace(s)

	// Try standard Go duration first (handles h, m, s, ms, etc.)
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// Handle "Nd" day suffix — not supported by Go's time.ParseDuration.
	// We parse the integer prefix and multiply by 24 hours.
	if strings.HasSuffix(s, "d") {
		numStr := strings.TrimSuffix(s, "d")
		var days int
		if _, err := fmt.Sscanf(numStr, "%d", &days); err != nil {
			return 0, fmt.Errorf("cannot parse %q as duration", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	return 0, fmt.Errorf("cannot parse %q as duration (use e.g. 7d, 24h, 30m)", s)
}

// containsCI returns true if s contains substr, ignoring case.
// Used for the --repo filter to allow case-insensitive matching against
// directory paths (e.g. --repo "sproutbook" matches "/git/Sproutbook/App").
func containsCI(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
