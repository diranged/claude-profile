# VHS Demo Tape: Session Management
# Run: cd demos && make sessions
# Requires: vhs, claude-profile binary built (make build)
#
# This demo uses pre-generated fake session data created by the Makefile
# via setup-demo-data.sh. CLAUDE_PROFILES_DIR is set by the Makefile to
# point at the temporary demo directory.

Output ../docs/sessions-demo.gif

Set Shell "zsh"
Set FontSize 15
Set Width 1400
Set Height 800
Set Padding 20
Set TypingSpeed 40ms
Set Theme "Dracula"
Set WaitTimeout 30s

# CLAUDE_PROFILES_DIR is set by the Makefile and inherited by the shell
# VHS spawns, so all commands use the fake demo data automatically.

# Allow commenting
Type "setopt INTERACTIVE_COMMENTS && clear"
Enter

# ---------------------------------------------------------------------------
# Step 1: List all sessions across repos
# ---------------------------------------------------------------------------
Type "# List all sessions across every repo"
Enter
Sleep 500ms

Type "claude-profile -P work sessions --since 30d"
Enter
Sleep 4s

# ---------------------------------------------------------------------------
# Step 2: Filter by repo name
# ---------------------------------------------------------------------------
Type "# Filter to a specific repo"
Enter
Sleep 500ms

Type "claude-profile -P work sessions --repo infrastructure"
Enter
Sleep 3s

# ---------------------------------------------------------------------------
# Step 3: Show a different profile's sessions
# ---------------------------------------------------------------------------
Type "# Each profile has its own sessions"
Enter
Sleep 500ms

Type "claude-profile -P personal sessions --since 30d"
Enter
Sleep 3s

# ---------------------------------------------------------------------------
# Step 4: Resume from wrong directory — cwd safety check
# ---------------------------------------------------------------------------
Type "# Try to resume a session from the wrong directory"
Enter
Sleep 500ms

Type "claude-profile -P work --resume a1b2c3d4"
Enter
Sleep 4s

