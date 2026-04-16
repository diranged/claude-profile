// Package sessions discovers and inspects Claude Code session files stored
// under a profile's config directory. Each session is a JSONL file at
// <configDir>/projects/<encoded-cwd>/<uuid>.jsonl. The first "type":"user"
// record in each file contains the session's working directory, git branch,
// and initial prompt text.
package sessions

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Session holds metadata extracted from a Claude Code session JSONL file.
type Session struct {
	// ID is the session UUID (the jsonl filename without extension).
	ID string

	// Slug is the human-readable session name (e.g. "async-wishing-forest").
	// Claude generates this and stores it on assistant message records.
	Slug string

	// Cwd is the working directory recorded on the first user message.
	Cwd string

	// GitBranch is the git branch recorded on the first user message.
	GitBranch string

	// FirstPrompt is the truncated text of the first user message.
	FirstPrompt string

	// ModTime is the file's last modification time.
	ModTime time.Time

	// Path is the absolute path to the .jsonl file.
	Path string
}

// FindByPrefix searches all project directories under configDir for session
// files whose UUID starts with the given prefix. An empty prefix matches all
// sessions. Results are sorted by ModTime descending (newest first).
func FindByPrefix(configDir, prefix string) ([]Session, error) {
	projectsDir := filepath.Join(configDir, "projects")
	pattern := filepath.Join(projectsDir, "*", prefix+"*.jsonl")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var sessions []Session
	for _, path := range matches {
		s, err := parseSessionMeta(path)
		if err != nil {
			continue // skip unreadable/malformed files
		}
		sessions = append(sessions, s)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ModTime.After(sessions[j].ModTime)
	})
	return sessions, nil
}

// jsonlRecord is the minimal JSON structure we need from lines in the
// session JSONL. User records have cwd/gitBranch/message; assistant
// records have the slug (pretty session name).
type jsonlRecord struct {
	Type      string      `json:"type"`
	Cwd       string      `json:"cwd"`
	GitBranch string      `json:"gitBranch"`
	Slug      string      `json:"slug"`
	Message   userMessage `json:"message"`
}

type userMessage struct {
	Content json.RawMessage `json:"content"`
}

// parseSessionMeta reads a session JSONL file and extracts metadata from
// the first "type":"user" record. It reads line-by-line and exits early
// for speed — session files can be very large.
func parseSessionMeta(path string) (Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return Session{}, err
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return Session{}, err
	}

	id := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	s := Session{
		ID:      id,
		ModTime: info.ModTime(),
		Path:    path,
	}

	scanner := bufio.NewScanner(f)
	// Session files can have very long lines (tool results, etc.)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	foundUser := false
	foundSlug := false

	for scanner.Scan() {
		var rec jsonlRecord
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue
		}

		// Extract cwd, branch, and prompt from the first user message
		if !foundUser && rec.Type == "user" {
			s.Cwd = rec.Cwd
			s.GitBranch = rec.GitBranch
			s.FirstPrompt = extractPromptText(rec.Message.Content)
			foundUser = true
		}

		// Extract the slug (pretty name) — appears on assistant messages
		if !foundSlug && rec.Slug != "" {
			s.Slug = rec.Slug
			foundSlug = true
		}

		if foundUser && foundSlug {
			break
		}
	}

	return s, nil
}

// extractPromptText pulls a short text snippet from the message content,
// which can be either a plain string or an array of content blocks.
func extractPromptText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Try plain string first
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return truncate(firstLine(text), 100)
	}

	// Try array of content blocks
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				return truncate(firstLine(b.Text), 100)
			}
		}
	}

	return ""
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
