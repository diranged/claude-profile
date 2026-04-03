# VHS Demo Tape for claude-profile
# Run: vhs demo.tape
# Requires: vhs, ttyd

Output docs/demo.gif

Set Shell "zsh"
Set FontSize 16
Set Width 1200
Set Height 700
Set Padding 20
Set TypingSpeed 50ms
Set Theme "Dracula"
Set WaitTimeout 120s

# --- Create a "work" profile (Bedrock) ---
Type "claude-profile create work"
Sleep 1s
Enter
Sleep 2s

# Prompt: copy config files? [Y/n]
Type "Y"
Enter
Sleep 1s

# Prompt: pick a color - Blue (option 2)
Type "2"
Enter
# First profile - give user time to read the output
Sleep 5s

# --- Create a "personal" profile ---
Type "claude-profile create personal"
Sleep 1s
Enter
Sleep 2s

# Prompt: copy config files? [Y/n]
Type "Y"
Enter
Sleep 1s

# Prompt: pick a color - Orange (option 3)
Type "3"
Enter
# Second profile - user gets the idea, less time needed
Sleep 3s

# --- List profiles ---
Type "claude-profile list"
Sleep 1s
Enter
Sleep 2s

# --- Authenticate the personal profile via OAuth ---
Type "CLAUDE_PROFILE=personal claude-profile auth login"
Sleep 1s
Enter
# VHS can't see claude's auth output - use a fixed sleep
# Approve the OAuth in the browser when it pops up
Sleep 5s

# --- Launch Claude with work profile via Bedrock (non-interactive) ---
Type "CLAUDE_CODE_USE_BEDROCK=1 claude-profile -P work -p 'Hi'"
Sleep 1s
Enter
Sleep 15s

# --- Clean up ---
Type "claude-profile delete -f work"
Sleep 1s
Enter
Sleep 1s
Type "claude-profile delete -f personal"
Sleep 1s
Enter
Sleep 1s

# --- Final list (empty) ---
Type "claude-profile list"
Sleep 1s
Enter
Sleep 2s
