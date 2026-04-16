package cli

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/diranged/claude-profile-go/internal/profile"
	"github.com/diranged/claude-profile-go/internal/sessions"
	"github.com/spf13/cobra"
)

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

			cutoff := time.Now().Add(-since)

			allSessions, err := sessions.FindByPrefix(p.ConfigDir, "")
			if err != nil {
				return fmt.Errorf("listing sessions: %w", err)
			}

			// Group by cwd, filter by time and repo
			type group struct {
				cwd      string
				sessions []sessions.Session
			}
			var groups []group
			groupIdx := make(map[string]int)

			for _, s := range allSessions {
				if s.ModTime.Before(cutoff) {
					continue
				}
				if repoFlag != "" && !containsCI(s.Cwd, repoFlag) {
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
					groups = append(groups, group{cwd: cwd, sessions: []sessions.Session{s}})
				}
			}

			sort.Slice(groups, func(i, j int) bool {
				return groups[i].cwd < groups[j].cwd
			})

			out := cmd.OutOrStdout()

			if len(groups) == 0 {
				_, _ = fmt.Fprintf(out, "\n  No sessions found")
				if sinceFlag != "" {
					_, _ = fmt.Fprintf(out, " in the last %s", sinceFlag)
				}
				_, _ = fmt.Fprintf(out, ".\n\n")
				return nil
			}

			total := 0
			for _, g := range groups {
				total += len(g.sessions)
			}

			_, _ = fmt.Fprintf(out, "\n%s%s SESSIONS%s", colorBold, colorGreen, colorReset)
			_, _ = fmt.Fprintf(out, " %s(%d sessions across %d repos)%s\n",
				colorDim, total, len(groups), colorReset)

			for _, g := range groups {
				_, _ = fmt.Fprintf(out, "\n%s=== %s ===%s\n", colorCyan, g.cwd, colorReset)
				for _, s := range g.sessions {
					ts := s.ModTime.Format("2006-01-02 15:04")
					branch := ""
					if s.GitBranch != "" {
						branch = fmt.Sprintf("[%s]", s.GitBranch)
					}
					prompt := s.FirstPrompt
					if prompt == "" {
						prompt = "(no prompt)"
					}
					_, _ = fmt.Fprintf(out, "  %s%-8s%s  %s%-16s%s  %s%-14s%s  %s\n",
						colorAccent, shortID(s.ID), colorReset,
						colorDim, ts, colorReset,
						colorGreen, branch, colorReset,
						prompt,
					)
				}
			}
			_, _ = fmt.Fprintln(out)
			return nil
		},
	}

	cmd.Flags().StringVar(&sinceFlag, "since", "7d", "show sessions modified within this duration (e.g. 1d, 12h, 30d)")
	cmd.Flags().StringVar(&repoFlag, "repo", "", "filter by repo path substring (case-insensitive)")

	return cmd
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
