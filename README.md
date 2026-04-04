# Claude Profile

**Claude Code... but with Profiles.**

A transparent wrapper around [Claude Code](https://docs.anthropic.com/en/docs/claude-code) that enables multiple subscription profiles. Each profile gets its own isolated config directory and macOS keychain entry, allowing concurrent sessions across different Claude subscriptions (e.g., work and personal).

<p align="center">
  <img src="docs/demo.gif" alt="Claude Profile Demo" width="800">
</p>

## How It Works

Claude Profile leverages Claude Code's built-in `CLAUDE_CONFIG_DIR` environment variable. When set, Claude Code automatically hashes the directory path (SHA-256) into its macOS keychain service name, producing a unique credential store per profile. No patching or hacking required -- it uses official, documented behavior.

```
Default keychain entry:       "Claude Code-credentials"
Profile keychain entry:       "Claude Code-credentials-<sha256[:8]>"
```

Each profile is a directory under `~/.claude-profiles/<name>/` containing:
- `config/` -- used as `CLAUDE_CONFIG_DIR` (Claude's settings, credentials, sessions)
- `claude-profile.yaml` -- profile-specific metadata (color code)

## Installation

### Quick Install (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/diranged/claude-profile/main/install.sh | sh
```

This detects your OS and architecture, downloads the latest release, verifies the SHA-256 checksum, and installs to `/usr/local/bin`.

To install a specific version or to a custom directory:

```bash
# Pin a version
curl -fsSL https://raw.githubusercontent.com/diranged/claude-profile/main/install.sh | VERSION=v0.1.0 sh

# Custom install directory
curl -fsSL https://raw.githubusercontent.com/diranged/claude-profile/main/install.sh | INSTALL_DIR=~/.local/bin sh
```

### From GitHub Releases

Download a prebuilt binary from the [Releases](https://github.com/diranged/claude-profile/releases) page. Binaries are available for Linux, macOS, and Windows on both amd64 and arm64.

### From Source

Requires Go 1.25+:

```bash
git clone https://github.com/diranged/claude-profile.git
cd claude-profile
make install    # Builds and copies to $GOPATH/bin
```

### From Go

```bash
go install github.com/diranged/claude-profile-go/cmd/main.go@latest
```

## Quick Start

1. **Create a profile:**

   ```bash
   claude-profile create work
   ```

   The interactive wizard will:
   - Create the isolated config directory
   - Offer to copy your existing `~/.claude` config files (CLAUDE.md, settings.json)
   - Let you pick a banner/statusline color
   - Configure the Claude Code statusline automatically

2. **Authenticate:**

   ```bash
   # OAuth (interactive)
   claude-profile -P work auth login

   # SSO
   claude-profile -P work auth login --sso

   # Or launch Claude and use /login inside the REPL
   claude-profile -P work
   ```

3. **Use it as a drop-in replacement for `claude`:**

   ```bash
   claude-profile -P work
   claude-profile -P work "explain this code"
   claude-profile -P personal --model opus
   ```

4. **Use an environment variable instead of `-p`:**

   ```bash
   export CLAUDE_PROFILE=work
   claude-profile "explain this code"
   ```

## Shell Aliases

Once your profiles are set up, shell aliases make switching effortless. Add these to your `~/.zshrc` or `~/.bashrc`:

```bash
# Personal Claude — just type "myclaude"
alias myclaude='claude-profile -P personal'

# Work Claude with a shared project config directory
alias workclaude='claude-profile -P work --add-dir ~/git/work/common-claude'

# Quick one-shot questions against different profiles
alias askwork='claude-profile -P work -p'
alias askpersonal='claude-profile -P personal -p'
```

You can pass any Claude flags through — the alias is just a shortcut for the profile selection:

```bash
myclaude                                 # Interactive REPL
myclaude "explain this function"         # One-shot question
workclaude --model opus -c "run tests"   # Specific model + command
workclaude auth login --sso              # Re-authenticate
```

This pairs well with `CLAUDE_PROFILE` for scripts:

```bash
# In a work-specific script or .envrc:
export CLAUDE_PROFILE=work
claude-profile "summarize the recent changes"
```

## Command Reference

### `claude-profile create <profile>`

Interactive wizard that creates a new profile. Steps through config bootstrapping, color selection, and statusline setup.

```bash
claude-profile create work
claude-profile create personal
```

### `claude-profile list`

Lists all profiles with their authentication status.

```bash
$ claude-profile list
  work                 (keychain)
  personal             (file)
  experiment           (none)
```

Auth status is one of:
- **keychain** -- OAuth credentials stored in macOS Keychain
- **file** -- credentials stored in `.credentials.json` (plaintext fallback)
- **none** -- no credentials found

### `claude-profile show <profile>`

Displays detailed information about a profile.

```bash
$ claude-profile show work
Profile:          work
Config dir:       /Users/you/.claude-profiles/work/config
Keychain service: Claude Code-credentials-6061db4b
Auth:             keychain
Subscription:     pro
Rate limit tier:  t3
Expires:          2025-01-15T12:00:00Z
Scopes:           user:inference, user:read
```

### `claude-profile delete <profile>`

Removes a profile's directory and its macOS keychain entry. Prompts for confirmation by default.

```bash
claude-profile delete old-profile
claude-profile delete old-profile --force   # Skip confirmation
```

### Passthrough Mode (Default)

When no subcommand matches, all arguments are forwarded to the real `claude` binary. This is the primary usage mode -- claude-profile acts as a transparent wrapper.

```bash
# These all pass through to claude:
claude-profile -P work
claude-profile -P work "explain this function"
claude-profile -P work --model opus -c "run tests"
claude-profile -P work auth login
claude-profile -P work auth login --sso
```

The `-p`/`--profile` flag (and `--profile=value` / `-pvalue` forms) is stripped before forwarding. All other flags and arguments pass through untouched. Claude Profile acts as a transparent wrapper.

### `claude-profile statusline [-- command args...]`

Acts as a Claude Code statusline provider. Prints a colored line showing the active profile name, auth method, and subscription type. Optionally chains to another statusline command.

```bash
# Standalone
claude-profile statusline

# Chaining with another statusline tool
claude-profile statusline -- bunx -y ccstatusline@latest
```

This is configured automatically by the `create` wizard in the profile's `settings.json`:

```json
{
  "statusLine": {
    "type": "command",
    "command": "/path/to/claude-profile statusline",
    "padding": 0
  }
}
```

If an existing statusline command is detected during creation, it is preserved by chaining:

```json
{
  "statusLine": {
    "type": "command",
    "command": "/path/to/claude-profile statusline -- bunx -y ccstatusline@latest",
    "padding": 0
  }
}
```

## Credential Isolation

Profile isolation relies on Claude Code's own keychain hashing behavior:

1. claude-profile sets `CLAUDE_CONFIG_DIR` to `~/.claude-profiles/<name>/config`
2. Claude Code computes `SHA-256(CLAUDE_CONFIG_DIR)` and takes the first 8 hex characters
3. The keychain service name becomes `Claude Code-credentials-<hash>`
4. Each profile gets a completely independent credential store

This matches Claude Code's internal `V51()` function. The hash is deterministic -- the same config directory always produces the same keychain entry.

**Example:**

```
Config dir:  /Users/you/.claude-profiles/work/config
SHA-256:     6061db4b...
Keychain:    Claude Code-credentials-6061db4b
```

## Statusline Integration

The statusline command reads profile info from environment variables set at launch time:

| Variable | Description |
|---|---|
| `CLAUDE_PROFILE_NAME` | Active profile name |
| `CLAUDE_PROFILE_AUTH` | Auth status (keychain/file/none) |
| `CLAUDE_PROFILE_SUB` | Subscription type (pro/max/free/unknown) |
| `CLAUDE_PROFILE_COLOR` | ANSI 256-color code for the profile |

These are injected into the Claude process environment by the passthrough handler, making them available to the statusline command when Claude Code invokes it.

## Color Customization

Each profile has an ANSI 256-color code used for the launch banner and statusline text. The `create` wizard offers these presets:

| Color | Name | Code |
|---|---|---|
| ![#87af87](https://img.shields.io/badge/●-87af87?style=flat-square&labelColor=87af87&color=87af87) | Green (default) | 108 |
| ![#0066ff](https://img.shields.io/badge/●-0066ff?style=flat-square&labelColor=0066ff&color=0066ff) | Blue | 33 |
| ![#ff8700](https://img.shields.io/badge/●-ff8700?style=flat-square&labelColor=ff8700&color=ff8700) | Orange | 208 |
| ![#ff5f87](https://img.shields.io/badge/●-ff5f87?style=flat-square&labelColor=ff5f87&color=ff5f87) | Pink | 204 |
| ![#00ffff](https://img.shields.io/badge/●-00ffff?style=flat-square&labelColor=00ffff&color=00ffff) | Cyan | 51 |
| ![#ff0000](https://img.shields.io/badge/●-ff0000?style=flat-square&labelColor=ff0000&color=ff0000) | Red | 196 |
| ![#af87ff](https://img.shields.io/badge/●-af87ff?style=flat-square&labelColor=af87ff&color=af87ff) | Purple | 141 |
| ![#ffff00](https://img.shields.io/badge/●-ffff00?style=flat-square&labelColor=ffff00&color=ffff00) | Yellow | 226 |
| | Custom | 0-255 |

The color is stored in `claude-profile.yaml` inside the profile directory and can be edited directly:

```yaml
color: 204
```

## Authentication Methods

Claude Profile does not handle authentication itself. It sets up the isolated environment and then delegates to Claude Code's built-in auth mechanisms.

### OAuth (Claude Subscription)

```bash
claude-profile -P work auth login
# Or inside Claude's REPL: /login
```

Credentials are stored in the macOS Keychain under a profile-specific service name.

### SSO (Enterprise)

```bash
claude-profile -P work auth login --sso
```

### API Key

```bash
ANTHROPIC_API_KEY=sk-ant-... claude-profile -P work
```

No keychain entry is created. The API key is passed through the environment.

### AWS Bedrock

```bash
CLAUDE_CODE_USE_BEDROCK=1 claude-profile -P work
```

Credentials come from AWS environment variables or IAM roles. Claude Profile detects this and skips the keychain credential check.

### Google Vertex AI

```bash
CLAUDE_CODE_USE_VERTEX=1 claude-profile -P work
```

Credentials come from GCP environment. Claude Profile detects this and skips the keychain credential check.

## Environment Variables Reference

| Variable | Description |
|---|---|
| `CLAUDE_PROFILE` | Default profile name (alternative to `-p` flag) |
| `CLAUDE_PROFILES_DIR` | Override profiles base directory (default: `~/.claude-profiles`) |
| `CLAUDE_PROFILE_DEBUG` | Enable debug logging (set to any non-empty value) |
| `CLAUDE_CONFIG_DIR` | Set automatically by claude-profile per-profile |
| `CLAUDE_CODE_USE_BEDROCK` | Signals Bedrock auth; skips keychain check |
| `CLAUDE_CODE_USE_VERTEX` | Signals Vertex auth; skips keychain check |
| `ANTHROPIC_API_KEY` | Direct API key auth; skips keychain check |
| `CLAUDE_PROFILE_NAME` | Set in Claude's env for statusline use |
| `CLAUDE_PROFILE_AUTH` | Set in Claude's env for statusline use |
| `CLAUDE_PROFILE_SUB` | Set in Claude's env for statusline use |
| `CLAUDE_PROFILE_COLOR` | Set in Claude's env for statusline use |

## License

See [LICENSE](LICENSE) for details.
