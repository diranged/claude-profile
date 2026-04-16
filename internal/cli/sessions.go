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

// sessionGroup holds sessions that share the same working directory.
type sessionGroup struct {
	cwd      string
	sessions []sessions.Session
}

// newSessionsCmd builds the "sessions" subcommand, which lists all Claude Code
// sessions across all repos for the active profile.
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
			name, err := resolveProfile()
			if err != nil {
				return err
			}

			p := profile.Load(name)
			if !p.Exists() {
				return fmt.Errorf("profile %q does not exist", name)
			}

			since, err := parseDuration(sinceFlag)
			if err != nil {
				return fmt.Errorf("invalid --since value %q: %w", sinceFlag, err)
			}

			allSessions, err := sessions.FindByPrefix(p.ConfigDir, "")
			if err != nil {
				return fmt.Errorf("listing sessions: %w", err)
			}

			cutoff := time.Now().Add(-since)
			groups := groupSessionsByCwd(allSessions, cutoff, repoFlag)

			out := cmd.OutOrStdout()
			if len(groups) == 0 {
				_, _ = fmt.Fprintf(out, "\n  No sessions found")
				if sinceFlag != "" {
					_, _ = fmt.Fprintf(out, " in the last %s", sinceFlag)
				}
				_, _ = fmt.Fprintf(out, ".\n\n")
				return nil
			}

			printSessionGroups(cmd.OutOrStdout(), groups)
			return nil
		},
	}

	cmd.Flags().StringVar(&sinceFlag, "since", "7d", "show sessions modified within this duration (e.g. 1d, 12h, 30d)")
	cmd.Flags().StringVar(&repoFlag, "repo", "", "filter by repo path substring (case-insensitive)")

	return cmd
}

// groupSessionsByCwd filters sessions by cutoff time and repo substring, then
// groups them by working directory, sorted alphabetically.
func groupSessionsByCwd(all []sessions.Session, cutoff time.Time, repoFilter string) []sessionGroup {
	var groups []sessionGroup
	groupIdx := make(map[string]int)

	for _, s := range all {
		if s.ModTime.Before(cutoff) {
			continue
		}
		if repoFilter != "" && !containsCI(s.Cwd, repoFilter) {
			continue
		}

		cwd := s.Cwd
		if cwd == "" {
			cwd = "(unknown)"
		}

		if idx, ok := groupIdx[cwd]; ok {
			groups[idx].sessions = append(groups[idx].sessions, s)
		} else {
			groupIdx[cwd] = len(groups)
			groups = append(groups, sessionGroup{cwd: cwd, sessions: []sessions.Session{s}})
		}
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].cwd < groups[j].cwd
	})
	return groups
}

// printSessionGroups renders the grouped session list to the given writer.
func printSessionGroups(w io.Writer, groups []sessionGroup) {
	total := 0
	for _, g := range groups {
		total += len(g.sessions)
	}

	_, _ = fmt.Fprintf(w, "\n%s%s SESSIONS%s", colorBold, colorGreen, colorReset)
	_, _ = fmt.Fprintf(w, " %s(%d sessions across %d repos)%s\n",
		colorDim, total, len(groups), colorReset)

	for _, g := range groups {
		_, _ = fmt.Fprintf(w, "\n%s=== %s ===%s\n", colorCyan, g.cwd, colorReset)
		for _, s := range g.sessions {
			printSessionLine(w, s)
		}
	}
	_, _ = fmt.Fprintln(w)
}

// printSessionLine renders a single session row.
func printSessionLine(w io.Writer, s sessions.Session) {
	ts := s.ModTime.Format("2006-01-02 15:04")

	branch := ""
	if s.GitBranch != "" {
		branch = fmt.Sprintf("[%s]", s.GitBranch)
	}

	// Show slug (pretty name) if available, otherwise the first prompt
	label := s.FirstPrompt
	if s.Slug != "" {
		label = s.Slug
		if s.FirstPrompt != "" {
			label += fmt.Sprintf(" %s— %s%s", colorDim, s.FirstPrompt, colorReset)
		}
	}
	if label == "" {
		label = "(no prompt)"
	}

	_, _ = fmt.Fprintf(w,
		"  %s%-8s%s  %s%-16s%s  %s%-14s%s  %s\n",
		colorAccent, shortID(s.ID), colorReset,
		colorDim, ts, colorReset,
		colorGreen, branch, colorReset,
		label,
	)
}

// parseDuration parses a human-friendly duration string like "7d", "24h", "30m".
// Supports d (days), h (hours), m (minutes) suffixes.
func parseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	s = strings.TrimSpace(s)

	// Try standard Go duration first (e.g. "24h", "30m")
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// Handle "Nd" day suffix
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

// containsCI returns true if s contains substr (case-insensitive).
func containsCI(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
