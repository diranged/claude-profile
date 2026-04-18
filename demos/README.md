# Demo Recordings

This directory contains [VHS](https://github.com/charmbracelet/vhs) tape files
for recording demo GIFs of claude-profile features. All demos use a shared
temporary profile directory so they never touch your real `~/.claude-profiles`.

## Prerequisites

- [VHS](https://github.com/charmbracelet/vhs) installed (`brew install vhs`)
- claude-profile binary built (`make build` from repo root)

## Quick Start

```bash
# Build the binary
make build

# From the demos/ directory:
cd demos

# Record the sessions demo (fully automated)
make sessions

# Record the profiles demo (interactive — needs human for OAuth)
make profiles
```

Output GIFs land in `docs/`:
- `docs/demo.gif` — profiles demo
- `docs/sessions-demo.gif` — sessions demo

All demos use `$TMPDIR/claude-profile-test` as the profiles directory so they
never touch your real `~/.claude-profiles`. Each target wipes and recreates
the directory from scratch, so every recording starts clean.

## Targets

### `make sessions`

Records `sessions.tape` — demonstrates the session management features:

1. `claude-profile sessions` — list all sessions across repos
2. `--repo` filter — narrow to a specific project
3. Per-profile sessions — show that each profile has its own sessions
4. `--resume` cwd check — the error when resuming from the wrong directory
5. `--resume-anywhere` — the escape hatch

Fully automated — no human interaction needed.

### `make profiles`

Records `profiles.tape` — the original demo showing profile creation, listing,
OAuth login, and launching Claude. **This is interactive**: the OAuth step
opens a browser and requires you to click "Authorize" before the VHS timeout.

### `make clean`

Removes the generated GIF files from `docs/` and the temp directory.

## Files

| File | Purpose |
|------|---------|
| `Makefile` | Orchestrates setup, recording, and cleanup |
| `setup-demo-data.sh` | Creates fake profiles, credentials, and sessions |
| `profiles.tape` | VHS tape for profile management demo |
| `sessions.tape` | VHS tape for session management demo |

## Adding New Demos

1. If your demo needs additional fake data, add `write_session` calls to
   `setup-demo-data.sh` (or create a new profile with `create_profile`).
2. Create a new `.tape` file. Use `$(DEMO_MARKER)` as a prerequisite in
   the Makefile to ensure setup has been run.
3. Set `Output docs/<feature>-demo.gif`.
4. Add a Makefile target and update this README.
