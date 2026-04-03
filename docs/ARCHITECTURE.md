# Architecture

This document describes the internal design of claude-profile.

## Overview

claude-profile is a thin, transparent wrapper around Claude Code. Its primary job is to set the `CLAUDE_CONFIG_DIR` environment variable before `exec`-ing the real `claude` binary, which causes Claude Code to use an isolated config directory and keychain entry. The wrapper then replaces itself with Claude via `syscall.Exec` -- it does not stay resident as a parent process.

## Core Isolation Mechanism: CLAUDE_CONFIG_DIR

Claude Code supports a `CLAUDE_CONFIG_DIR` environment variable that overrides the default `~/.claude` config directory. When this variable is set, Claude Code also changes its macOS keychain service name by appending a SHA-256 hash of the directory path:

```
Default:              "Claude Code-credentials"
With CLAUDE_CONFIG_DIR: "Claude Code-credentials-<sha256[:8]>"
```

The hash uses the first 8 hex characters (4 bytes) of the SHA-256 digest. This behavior is implemented in Claude Code's internal `V51()` function (in `cli.js`).

claude-profile replicates this hash computation in `keychainService()` to:
- Predict the keychain service name for each profile (for display and cleanup)
- Verify credential presence without launching Claude

### Hash Example

```
Input:     /Users/you/.claude-profiles/work/config
SHA-256:   6061db4b...
Service:   Claude Code-credentials-6061db4b
```

## Process Lifecycle

The passthrough flow (the primary usage mode) follows this sequence:

```
1. User runs:          claude-profile -P work --model opus
2. cobra parses:       profileFlag = "work", remaining args unknown to cobra
3. resolveProfile():   Returns "work" from flag or CLAUDE_PROFILE env
4. profile.Load():     Constructs Profile struct with paths and keychain key
5. Profile.Exists():   Checks config directory exists on disk
6. AuthStatus():       Checks keychain (via `security` CLI), then .credentials.json
7. extractClaudeArgs():Strips -P/--profile from os.Args, keeps everything else
8. claude.FindBinary():Locates real claude binary via PATH then fallback paths
9. claude.BuildEnv():  Copies os.Environ(), sets CLAUDE_CONFIG_DIR
10. setEnv():          Adds CLAUDE_PROFILE_NAME/AUTH/SUB/COLOR to env
11. printBanner():     Renders Unicode box-drawing banner to stderr
12. syscall.Exec():    Replaces current process with claude binary
```

After step 12, the claude-profile process no longer exists. The `claude` binary runs with PID unchanged, full terminal control, and the modified environment.

### Why syscall.Exec Instead of exec.Command

`syscall.Exec` replaces the current process entirely (Unix `execve`). This is preferred over running Claude as a child process because:

- Claude Code gets direct terminal control (no PTY proxy needed)
- Signal handling works naturally (Ctrl-C goes directly to Claude)
- No zombie process or extra PID
- Exit codes propagate automatically to the caller's shell
- No stdout/stderr buffering or interleaving issues

## Package Structure

```
cmd/
  main.go                 Entry point. Delegates to cli.Execute().

internal/
  cli/
    root.go               Cobra command tree, flag binding, logger init.
    passthrough.go        Default handler: banner + syscall.Exec to claude.
    login.go              "create" subcommand: interactive profile wizard.
    list.go               "list" subcommand: enumerate profiles with auth status.
    show.go               "show" subcommand: detailed profile info display.
    delete.go             "delete" subcommand: remove profile + keychain entry.
    statusline.go         "statusline" subcommand: Claude Code statusline provider.
    args.go               rawArgs() helper for extracting os.Args.
    *_test.go             Unit tests for CLI commands.

  profile/
    profile.go            Profile struct, Load/Exists/Delete/List/AuthStatus/OAuthDetails.
    config.go             Per-profile YAML config (color). Load/Save.
    bootstrap.go          Copy config files from default ~/.claude into new profile.
    *_test.go             Unit tests for profile logic.

  claude/
    claude.go             FindBinary (PATH + fallbacks), BuildEnv, RunDirect.
    claude_test.go        Unit tests for binary discovery and env building.
```

## Data Flow Diagrams

### Profile Creation (`create`)

```
User input --> Wizard prompts
                |
                v
          profile.EnsureDir()       Creates ~/.claude-profiles/<name>/config/
                |
                v
          offerBootstrap()          Copies CLAUDE.md, settings.json from ~/.claude
                |
                v
          pickColor()               Interactive ANSI color selection
                |
                v
          profile.SaveConfig()      Writes claude-profile.yaml
                |
                v
          configureStatusline()     Updates settings.json with statusline command
                |
                v
          Print next-step instructions
```

### Profile Passthrough (Default)

```
os.Args  -->  cobra parse  -->  extractClaudeArgs()  -->  stripped args
                  |
                  v
           resolveProfile()  -->  profile.Load()  -->  Profile struct
                                       |
                                       v
                                 Exists? AuthStatus?
                                       |
                                       v
                              claude.FindBinary()  -->  binary path
                                       |
                                       v
                              claude.BuildEnv()  -->  env with CLAUDE_CONFIG_DIR
                                       |
                                       v
                              printBanner() to stderr
                                       |
                                       v
                              syscall.Exec(binary, args, env)
```

### Statusline

```
Claude Code invokes statusline command
         |
         v
   Read stdin (JSON session blob from Claude)
         |
         v
   Read CLAUDE_PROFILE_* env vars (set at launch)
         |
         v
   Print colored profile info line to stdout
         |
         v
   If args after "--": pipe stdin JSON into child command
         |
         v
   Child's stdout appended to our stdout
```

## Settings.json Manipulation

The `create` wizard modifies the profile's `settings.json` to register the statusline command. It:

1. Reads existing `settings.json` (if any, possibly bootstrapped from `~/.claude`)
2. Checks for an existing `statusLine.command` entry
3. If found, wraps it: `claude-profile statusline -- <original command>`
4. If not found, sets: `claude-profile statusline`
5. Writes back with `json.MarshalIndent` for readability

The statusline entry format matches Claude Code's expected schema:

```json
{
  "statusLine": {
    "type": "command",
    "command": "/path/to/claude-profile statusline",
    "padding": 0
  }
}
```

## Key Design Decisions

### Profile Directory Layout

```
~/.claude-profiles/
  work/
    claude-profile.yaml     # Our metadata (color)
    config/                 # CLAUDE_CONFIG_DIR target
      settings.json         # Claude Code settings
      .credentials.json     # Plaintext credential fallback
      CLAUDE.md             # Claude Code instructions
      ...                   # Claude's sessions, projects, etc.
```

The `claude-profile.yaml` file lives in the profile root (not inside `config/`) to keep it separate from Claude Code's own files. Claude Code only sees the `config/` subdirectory.

### Flag Parsing Strategy

cobra's `FParseErrWhitelist.UnknownFlags` is enabled so that flags intended for Claude (like `--model`, `-c`, `--dangerously-skip-permissions`) are not rejected. The `extractClaudeArgs()` function then manually strips only the `-P`/`--profile` flag from `os.Args` before passing the rest to Claude. Uppercase `-P` is used to avoid conflicting with Claude Code's own `-p` (prompt) flag.

The function handles all flag forms: `-P value`, `--profile value`, `--profile=value`, and `-Pvalue`.

### Version Injection

The `Version` variable in `passthrough.go` defaults to `"dev"` and is overridden at build time via `-ldflags`:

```
-X github.com/diranged/claude-profile-go/internal/cli.Version=1.2.3
```

This is used in the banner display and set by both the Makefile (`git describe`) and GoReleaser (`{{.Version}}`).
