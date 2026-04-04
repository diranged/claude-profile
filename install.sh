#!/bin/sh
# install.sh — install claude-profile from GitHub releases
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/diranged/claude-profile/main/install.sh | sh
#   curl -fsSL ... | VERSION=v0.2.0 sh
#
# Environment variables:
#   VERSION      — release tag to install (default: latest)
#   INSTALL_DIR  — where to place the binary (default: /usr/local/bin)

set -eu

REPO="diranged/claude-profile"
BINARY="claude-profile"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# --- helpers ----------------------------------------------------------------

info()  { printf '  \033[38;5;108m▸\033[0m %s\n' "$1"; }
warn()  { printf '  \033[33m▸\033[0m %s\n' "$1"; }
error() { printf '  \033[31m✗\033[0m %s\n' "$1" >&2; exit 1; }

need_cmd() {
    if ! command -v "$1" > /dev/null 2>&1; then
        error "required command not found: $1"
    fi
}

# --- detect platform --------------------------------------------------------

detect_os() {
    case "$(uname -s)" in
        Darwin) echo "darwin" ;;
        Linux)  echo "linux" ;;
        *)      error "unsupported OS: $(uname -s)" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *)             error "unsupported architecture: $(uname -m)" ;;
    esac
}

# --- resolve version --------------------------------------------------------

resolve_version() {
    if [ -n "${VERSION:-}" ]; then
        echo "$VERSION"
        return
    fi
    need_cmd curl
    # Follow the /releases/latest redirect and extract the tag from the final URL.
    # No API auth or rate limits required.
    local url
    url=$(curl -fsSL -o /dev/null -w '%{url_effective}' \
        "https://github.com/${REPO}/releases/latest")
    local tag="${url##*/}"
    if [ -z "$tag" ] || [ "$tag" = "latest" ]; then
        error "could not determine latest release"
    fi
    echo "$tag"
}

# --- main -------------------------------------------------------------------

main() {
    need_cmd curl
    need_cmd tar
    need_cmd uname

    printf '\n  \033[1mClaude Profile Installer\033[0m\n\n'

    local os arch version
    os=$(detect_os)
    arch=$(detect_arch)
    version=$(resolve_version)
    local version_num="${version#v}"

    info "Platform:  ${os}/${arch}"
    info "Version:   ${version}"
    info "Target:    ${INSTALL_DIR}/${BINARY}"
    printf '\n'

    local archive="${BINARY}_${version_num}_${os}_${arch}.tar.gz"
    local url="https://github.com/${REPO}/releases/download/${version}/${archive}"
    local checksum_url="https://github.com/${REPO}/releases/download/${version}/checksums.txt"

    # download to temp dir
    TMPDIR_CLEANUP=$(mktemp -d)
    trap 'rm -rf "$TMPDIR_CLEANUP"' EXIT
    local tmpdir="$TMPDIR_CLEANUP"

    info "Downloading ${archive}..."
    if ! curl -fsSL -o "${tmpdir}/${archive}" "$url"; then
        error "download failed — check that ${version} exists at https://github.com/${REPO}/releases"
    fi

    # verify checksum if sha256sum or shasum is available
    if command -v sha256sum > /dev/null 2>&1 || command -v shasum > /dev/null 2>&1; then
        info "Verifying checksum..."
        curl -fsSL -o "${tmpdir}/checksums.txt" "$checksum_url"
        local expected
        expected=$(grep "${archive}" "${tmpdir}/checksums.txt" | awk '{print $1}')
        if [ -z "$expected" ]; then
            warn "checksum entry not found for ${archive}, skipping verification"
        else
            local actual
            if command -v sha256sum > /dev/null 2>&1; then
                actual=$(sha256sum "${tmpdir}/${archive}" | awk '{print $1}')
            else
                actual=$(shasum -a 256 "${tmpdir}/${archive}" | awk '{print $1}')
            fi
            if [ "$expected" != "$actual" ]; then
                error "checksum mismatch!\n  expected: ${expected}\n  got:      ${actual}"
            fi
            info "Checksum verified ✓"
        fi
    else
        warn "sha256sum/shasum not found, skipping checksum verification"
    fi

    # extract
    info "Extracting..."
    tar -xzf "${tmpdir}/${archive}" -C "${tmpdir}"

    # install
    if [ -w "$INSTALL_DIR" ]; then
        mv "${tmpdir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    else
        info "Elevated permissions required to write to ${INSTALL_DIR}"
        sudo mv "${tmpdir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    fi
    chmod +x "${INSTALL_DIR}/${BINARY}"

    printf '\n  \033[38;5;108m✓\033[0m \033[1m%s %s\033[0m installed to %s\n\n' \
        "$BINARY" "$version" "$INSTALL_DIR"

    # verify it runs
    if command -v "$BINARY" > /dev/null 2>&1; then
        info "Run 'claude-profile --help' to get started"
    else
        warn "${INSTALL_DIR} may not be in your PATH"
        warn "Add it with: export PATH=\"${INSTALL_DIR}:\$PATH\""
    fi
    printf '\n'
}

main
