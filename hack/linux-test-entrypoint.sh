#!/usr/bin/env bash
# Entrypoint for the Linux smoke-test container.
#
# Builds claude-profile from /src, installs it on PATH, prints a quick banner
# with version info, then exec's whatever command was passed (default: bash).
set -euo pipefail

if [[ ! -d /src ]]; then
    echo "expected /src bind mount with the claude-profile source tree" >&2
    exit 1
fi

cd /src
echo "Building claude-profile for $(go env GOOS)/$(go env GOARCH).."
go build -o /usr/local/bin/claude-profile ./cmd/main.go

echo
echo "==> claude-profile version"
claude-profile --version || true
echo "==> claude binary"
which claude || echo "(claude not on PATH)"
echo "==> uname"
uname -a
echo

exec "$@"
