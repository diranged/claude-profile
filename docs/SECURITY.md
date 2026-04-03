# Security Considerations

This document describes the trust model, credential handling, and security properties of claude-profile.

## Trust Model

claude-profile is a thin wrapper that sets an environment variable (`CLAUDE_CONFIG_DIR`) and then replaces itself with the real `claude` binary via `syscall.Exec`. It:

- **Does not** intercept, read, or modify Claude Code's network traffic
- **Does not** handle authentication tokens directly (except for display via `show`)
- **Does not** persist any secrets itself
- **Does** read keychain entries for display purposes only (via the `security` CLI)
- **Does** have access to the same filesystem and keychain as the user running it

The security boundary is identical to running Claude Code directly. claude-profile adds no additional network exposure or privilege escalation.

## Credential Isolation

### macOS Keychain (Primary)

Claude Code stores OAuth credentials in the macOS Keychain as a generic password entry. The keychain service name is derived from `CLAUDE_CONFIG_DIR`:

```
Service: "Claude Code-credentials-<SHA256(CLAUDE_CONFIG_DIR)[:8]>"
Account: <current OS username>
```

Each profile gets a unique service name because each has a unique config directory path. The SHA-256 hash is deterministic -- the same path always maps to the same keychain entry.

**Properties:**
- Credentials are encrypted at rest by the macOS Keychain
- Access is controlled by macOS Keychain ACLs (typically tied to the Claude Code binary)
- claude-profile reads keychain entries using the `security find-generic-password` CLI tool
- Deletion uses `security delete-generic-password`
- The `security` CLI requires user-level access (no root needed for the login keychain)

### Plaintext Fallback (.credentials.json)

If the keychain is unavailable, Claude Code falls back to storing credentials in a `.credentials.json` file inside the config directory:

```
~/.claude-profiles/<name>/config/.credentials.json
```

**Risks:**
- The file contains OAuth tokens in plaintext JSON
- File permissions depend on the profile directory creation (`0700` for the config dir, `0644` for files created by Claude Code itself)
- Any process running as the same user can read this file
- The file is not encrypted

**Mitigations:**
- Profile directories are created with `0700` permissions
- The `AuthStatus()` method checks keychain first, falling back to file only if needed
- Users on macOS should ensure keychain integration is working for production use

## Keychain Access via `security` CLI

claude-profile shells out to the macOS `security` command-line tool for all keychain operations:

```go
// Read
exec.Command("security", "find-generic-password", "-s", serviceKey, "-a", username, "-w")

// Check existence
exec.Command("security", "find-generic-password", "-s", serviceKey, "-a", username)

// Delete
exec.Command("security", "delete-generic-password", "-s", serviceKey, "-a", username)
```

**Security properties:**
- The `security` CLI is a standard macOS system tool (`/usr/bin/security`)
- Password data (`-w` flag) is written to stdout of the child process
- The service key and username are passed as command arguments (visible in `ps` output briefly)
- No keychain passwords are passed on the command line -- they are read from stdout

**Consideration:** The OAuth token is briefly in memory as a Go string when `OAuthDetails()` parses it for display in the `show` command. It is not written to disk or logged.

## Environment Variable Exposure

claude-profile sets several environment variables that are inherited by the Claude process:

| Variable | Contains | Sensitivity |
|---|---|---|
| `CLAUDE_CONFIG_DIR` | Filesystem path | Low -- reveals directory structure |
| `CLAUDE_PROFILE_NAME` | Profile name | Low |
| `CLAUDE_PROFILE_AUTH` | "keychain"/"file"/"none" | Low |
| `CLAUDE_PROFILE_SUB` | "pro"/"max"/"free" | Low |
| `CLAUDE_PROFILE_COLOR` | ANSI color code | None |

None of these contain secrets. However, the user may pass sensitive variables through the environment:

| Variable | Contains | Sensitivity |
|---|---|---|
| `ANTHROPIC_API_KEY` | API secret key | **High** |
| `CLAUDE_CODE_USE_BEDROCK` | Feature flag | Low |
| `CLAUDE_CODE_USE_VERTEX` | Feature flag | Low |

claude-profile does not set or modify `ANTHROPIC_API_KEY` -- it passes through whatever the user has in their environment. The key is visible to any process that can inspect the environment of the Claude process (e.g., via `/proc/<pid>/environ` on Linux).

## syscall.Exec Implications

claude-profile uses `syscall.Exec` (Unix `execve`) to replace itself with the Claude binary:

```go
syscall.Exec(claudeBin, argv, env)
```

**Security properties:**
- The current process image is entirely replaced -- no claude-profile code remains in memory
- The Claude process inherits the same PID, file descriptors, and signal handlers
- The environment is explicitly constructed (not blindly inherited)
- The binary path comes from `exec.LookPath` or known filesystem paths

**Risk:** If an attacker can control the `PATH` or place a malicious `claude` binary in a higher-priority location, `FindBinary()` would execute it. This is the same risk as running `claude` directly.

## Shell Injection Analysis

claude-profile does not use `sh -c` or any shell evaluation for its core operations. Arguments are passed directly to `syscall.Exec` as an array:

```go
argv := append([]string{claudeBin}, claudeArgs...)
syscall.Exec(claudeBin, argv, env)
```

The statusline chaining feature does use `exec.Command` with user-provided arguments (everything after `--`), but these are passed as separate array elements, not concatenated into a shell string:

```go
child := exec.Command(args[0], args[1:]...)
```

The `security` CLI invocations also use `exec.Command` with explicit argument arrays. No shell interpolation occurs.

**Consideration:** The `configureStatusline()` function writes a command string into `settings.json` that Claude Code later executes. This string is constructed from the binary path (`os.Executable()`) and is not user-controlled at write time. However, if an attacker can modify `settings.json`, they could inject an arbitrary statusline command that Claude Code would execute.

## File Permissions

| Path | Permissions | Set By |
|---|---|---|
| `~/.claude-profiles/` | Inherited from parent | OS default (typically 0755) |
| `~/.claude-profiles/<name>/config/` | 0700 | `os.MkdirAll` in `EnsureDir()` |
| `~/.claude-profiles/<name>/claude-profile.yaml` | 0644 | `os.WriteFile` in `SaveConfig()` |
| `~/.claude-profiles/<name>/config/settings.json` | 0644 | `os.WriteFile` in `configureStatusline()` |
| `~/.claude-profiles/<name>/config/.credentials.json` | Varies | Created by Claude Code, not claude-profile |

**Recommendation:** The profiles base directory (`~/.claude-profiles`) should be `0700` to prevent other users from enumerating profiles. Users can set this manually:

```bash
chmod 700 ~/.claude-profiles
```

## Recommendations

1. **Prefer keychain authentication.** Use OAuth login (`auth login`) rather than API keys or plaintext credential files when possible.

2. **Protect API keys.** If using `ANTHROPIC_API_KEY`, be aware it is visible in the process environment. Consider using a secrets manager or shell integration that sets it only for the current command.

3. **Restrict profiles directory permissions.** Run `chmod 700 ~/.claude-profiles` after creating your first profile.

4. **Audit settings.json.** The statusline command in `settings.json` is executed by Claude Code. Verify it points to the expected binary.

5. **Keep Claude Code updated.** claude-profile depends on Claude Code's `CLAUDE_CONFIG_DIR` and keychain hashing behavior. Updates to Claude Code may change these internals.

6. **Use the `show` command to verify isolation.** Run `claude-profile show <name>` to confirm each profile has a distinct keychain service name.

## Threat Summary

| Threat | Impact | Likelihood | Mitigation |
|---|---|---|---|
| Malicious `claude` binary in PATH | Full compromise | Low | Use absolute paths, verify PATH |
| Plaintext `.credentials.json` read by other process | Token theft | Medium | Use keychain auth, restrict dir perms |
| API key visible in process environment | Key exposure | Medium | Use OAuth instead, or short-lived keys |
| Modified `settings.json` statusline command | Arbitrary execution | Low | Restrict file permissions, audit settings |
| Profile directory enumeration by other users | Privacy leak | Low | `chmod 700 ~/.claude-profiles` |
