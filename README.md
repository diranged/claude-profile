# claude-profile

A transparent wrapper around [Claude Code](https://docs.anthropic.com/en/docs/claude-code) that enables multiple subscription profiles. Each profile gets its own isolated config directory and macOS keychain entry, allowing concurrent sessions across different Claude subscriptions (e.g., work and personal).

## How It Works

claude-profile leverages Claude Code's built-in `CLAUDE_CONFIG_DIR` environment variable. When set, Claude Code automatically hashes the directory path (SHA-256) into its macOS keychain service name, producing a unique credential store per profile. No patching or hacking required -- it uses official, documented behavior.

```
Default keychain entry:       "Claude Code-credentials"
Profile keychain entry:       "Claude Code-credentials-<sha256[:8]>"
```

Each profile is a directory under `~/.claude-profiles/<name>/` containing:
- `config/` -- used as `CLAUDE_CONFIG_DIR` (Claude's settings, credentials, sessions)
- `claude-profile.yaml` -- profile-specific metadata (color code)

## Installation

### From Source

Requires Go 1.25+:

```bash
git clone https://github.com/diranged/claude-profile-go.git
cd claude-profile-go
make install    # Builds and copies to $GOPATH/bin
```

### From GitHub Releases

Download a prebuilt binary from the [Releases](https://github.com/diranged/claude-profile-go/releases) page. Binaries are available for Linux, macOS, and Windows on both amd64 and arm64.

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
   claude-profile -p work auth login

   # SSO
   claude-profile -p work auth login --sso

   # Or launch Claude and use /login inside the REPL
   claude-profile -p work
   ```

3. **Use it as a drop-in replacement for `claude`:**

   ```bash
   claude-profile -p work
   claude-profile -p work "explain this code"
   claude-profile -p personal --model opus
   ```

4. **Use an environment variable instead of `-p`:**

   ```bash
   export CLAUDE_PROFILE=work
   claude-profile "explain this code"
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
claude-profile -p work
claude-profile -p work "explain this function"
claude-profile -p work --model opus -c "run tests"
claude-profile -p work auth login
claude-profile -p work auth login --sso
```

The `-p`/`--profile` flag (and `--profile=value` / `-pvalue` forms) is stripped before forwarding. All other flags and arguments pass through untouched.

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

| Name | Code |
|---|---|
| Green (default) | 108 |
| Blue | 33 |
| Orange | 208 |
| Pink | 204 |
| Cyan | 51 |
| Red | 196 |
| Purple | 141 |
| Yellow | 226 |
| Custom | 0-255 |

The color is stored in `claude-profile.yaml` inside the profile directory and can be edited directly:

```yaml
color: 204
```

## Authentication Methods

claude-profile itself does not handle authentication. It sets up the isolated environment and then delegates to Claude Code's built-in auth mechanisms.

### OAuth (Claude Subscription)

```bash
claude-profile -p work auth login
# Or inside Claude's REPL: /login
```

Credentials are stored in the macOS Keychain under a profile-specific service name.

### SSO (Enterprise)

```bash
claude-profile -p work auth login --sso
```

### API Key

```bash
ANTHROPIC_API_KEY=sk-ant-... claude-profile -p work
```

No keychain entry is created. The API key is passed through the environment.

### AWS Bedrock

```bash
CLAUDE_CODE_USE_BEDROCK=1 claude-profile -p work
```

Credentials come from AWS environment variables or IAM roles. claude-profile detects this and skips the keychain credential check.

### Google Vertex AI

```bash
CLAUDE_CODE_USE_VERTEX=1 claude-profile -p work
```

Credentials come from GCP environment. claude-profile detects this and skips the keychain credential check.

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
