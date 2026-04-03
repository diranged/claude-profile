# Development Guide

## Prerequisites

- **Go 1.25+** (required by `go.mod`)
- **macOS** for keychain integration testing (the `security` CLI is macOS-specific)
- **Claude Code** installed somewhere in your PATH (for end-to-end testing)

## Building

```bash
# Build to bin/claude-profile
make build

# Build and install to $GOPATH/bin
make install

# Clean build artifacts
make clean
```

The build injects the version from `git describe` via `-ldflags`:

```bash
go build -ldflags "-X github.com/diranged/claude-profile-go/internal/cli.Version=$(git describe --tags --always --dirty)" -o bin/claude-profile ./cmd/main.go
```

## Testing

```bash
# Run all tests with coverage
make test

# View coverage summary
make cover

# Open HTML coverage report
make coverhtml
```

Tests use `t.TempDir()` and `t.Setenv()` for isolation -- they do not touch your real `~/.claude-profiles` directory or macOS keychain.

Key test patterns:
- `CLAUDE_PROFILES_DIR` is overridden to a temp directory
- `PATH` is manipulated to test binary discovery
- Keychain tests check both the presence and absence paths
- The `keychainService()` function has a known-hash test to verify compatibility with Claude Code's `V51()` function

## Linting

```bash
# Run golangci-lint (downloads automatically if needed)
make lint

# Run with auto-fix
make lint-fix
```

The linter configuration is in `.golangci.yml`. It uses golangci-lint v2.5.0 with these linters enabled:

- copyloopvar, dupl, errcheck, goconst, gocyclo, govet
- ineffassign, lll, misspell, nakedret, prealloc, revive
- staticcheck, unconvert, unparam, unused

Formatters: gofmt, goimports.

## Project Structure

```
.
├── cmd/
│   └── main.go                  Entry point
├── internal/
│   ├── cli/                     Cobra commands and CLI logic
│   │   ├── root.go              Root command, flag binding, logger
│   │   ├── passthrough.go       Default handler (banner + exec)
│   │   ├── login.go             "create" subcommand wizard
│   │   ├── list.go              "list" subcommand
│   │   ├── show.go              "show" subcommand
│   │   ├── delete.go            "delete" subcommand
│   │   ├── statusline.go        "statusline" subcommand
│   │   ├── args.go              Argument extraction helper
│   │   └── *_test.go            CLI tests
│   ├── profile/                 Profile management
│   │   ├── profile.go           Core Profile type and operations
│   │   ├── config.go            YAML config (color settings)
│   │   ├── bootstrap.go         Config file copying from ~/.claude
│   │   └── *_test.go            Profile tests
│   └── claude/                  Claude binary interaction
│       ├── claude.go            Binary discovery, env building
│       └── claude_test.go       Claude package tests
├── Makefile                     Build, test, lint targets
├── .golangci.yml                Linter configuration
├── .goreleaser.yml              Release configuration
├── .github/
│   └── workflows/
│       ├── ci.yml               CI pipeline (fmt, vet, lint, test)
│       └── release.yml          Tag-triggered release via GoReleaser
├── go.mod
└── go.sum
```

## Adding a New Command

1. Create a new file in `internal/cli/`, e.g., `rename.go`
2. Implement a constructor function following the pattern:

   ```go
   func newRenameCmd() *cobra.Command {
       return &cobra.Command{
           Use:   "rename <old> <new>",
           Short: "Rename a profile",
           Args:  cobra.ExactArgs(2),
           RunE: func(cmd *cobra.Command, args []string) error {
               // Implementation here
               return nil
           },
       }
   }
   ```

3. Register it in `root.go` by adding to the `root.AddCommand(...)` call
4. Add tests in a corresponding `rename_test.go` file
5. Update documentation

## Adding Profile Operations

New operations on profiles should be added as methods on the `*Profile` type in `internal/profile/profile.go`. The pattern is:

```go
func (p *Profile) NewOperation() error {
    // Use p.Dir for profile root, p.ConfigDir for Claude's config dir
    // Use p.ServiceKey for keychain operations
    return nil
}
```

## Release Process

Releases are automated via GitHub Actions and GoReleaser:

1. Ensure all CI checks pass on `main`
2. Create and push a version tag:

   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

3. The `release.yml` workflow triggers automatically and:
   - Builds binaries for linux/darwin/windows on amd64/arm64
   - Creates a GitHub Release with the binaries
   - Generates a changelog from conventional commits
   - Produces checksums

The GoReleaser config (`.goreleaser.yml`) builds with `CGO_ENABLED=0` for fully static binaries and injects the version via `-ldflags`.

### Changelog Filtering

The release changelog excludes commits prefixed with: `docs:`, `test:`, `ci:`, `chore:`, and merge commits.

## Code Style

- Follow standard Go conventions and `gofmt` formatting
- Every exported function, type, and package must have a doc comment
- Use `cmd.OutOrStdout()` for testable command output (not `fmt.Println`)
- Use `cmd.ErrOrStderr()` for error/warning output
- Banners and visual output go to stderr to avoid interfering with piped stdout
- Use `t.TempDir()` and `t.Setenv()` in tests for automatic cleanup
- Error messages should suggest actionable next steps where possible

## Dependencies

Core dependencies:
- `github.com/spf13/cobra` -- CLI command framework
- `github.com/spf13/viper` -- Configuration/env var binding
- `go.uber.org/zap` -- Structured logging
- `go.yaml.in/yaml/v3` -- YAML config parsing
- `golang.org/x/term` -- Terminal size detection
- `github.com/stretchr/testify` -- Test assertions

## CI Pipeline

The CI workflow (`.github/workflows/ci.yml`) runs on every push to `main` and all pull requests:

1. **Check formatting** -- `gofmt -l .`
2. **Vet** -- `go vet ./...`
3. **Lint** -- `golangci-lint` via the official GitHub Action
4. **Test** -- `go test ./...` with coverage
5. **Upload coverage** -- Coverage report stored as artifact for 14 days
