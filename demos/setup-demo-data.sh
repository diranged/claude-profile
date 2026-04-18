#!/usr/bin/env bash
# setup-demo-data.sh — Creates a fully self-contained demo environment with
# fake profiles, credentials, and session data for deterministic VHS demos.
#
# Usage:
#   CLAUDE_PROFILES_DIR=/tmp/my-demo bash demos/setup-demo-data.sh
#
# Requires CLAUDE_PROFILES_DIR to be set. Fails if not.
#
# Everything is fake — no real credentials, no keychain access, no network
# calls. The profiles pass all auth checks because they have a valid
# .credentials.json file on disk.

set -euo pipefail

# ---------------------------------------------------------------------------
# Validate required environment
# ---------------------------------------------------------------------------

if [[ -z "${CLAUDE_PROFILES_DIR:-}" ]]; then
    echo "Error: CLAUDE_PROFILES_DIR must be set." >&2
    exit 1
fi

# ---------------------------------------------------------------------------
# Helper: create a fake profile with credentials
# ---------------------------------------------------------------------------
# Args:
#   $1 — profile name (e.g. "work", "personal")
#   $2 — ANSI 256-color code (e.g. 33 for blue, 208 for orange)
#   $3 — subscription type (e.g. "pro", "max", "team", "free")
create_profile() {
    local name="$1" color="$2" sub_type="$3"
    local profile_dir="${CLAUDE_PROFILES_DIR}/${name}"
    local config_dir="${profile_dir}/config"

    mkdir -p "${config_dir}"

    # Profile config (color for banner/statusline)
    cat > "${profile_dir}/claude-profile.yaml" <<YAML
color: ${color}
YAML

    # Fake credentials file — AuthStatus() checks os.Stat() on this file,
    # and OAuthDetails() parses the claudeAiOauth block for display.
    # No real tokens, no keychain, fully isolated.
    cat > "${config_dir}/.credentials.json" <<JSON
{
  "claudeAiOauth": {
    "subscriptionType": "${sub_type}",
    "scopes": ["user:inference", "user:read"],
    "expiresAt": 1893456000000,
    "rateLimitTier": "t3"
  }
}
JSON
}

# ---------------------------------------------------------------------------
# Helper: write a fake session JSONL file
# ---------------------------------------------------------------------------
# Args:
#   $1 — profile name
#   $2 — encoded cwd (dashes instead of slashes, e.g. "-Users-matt-git-myapp")
#   $3 — session UUID
#   $4 — cwd (absolute path)
#   $5 — git branch
#   $6 — first prompt text
#   $7 — slug (session pretty name, or "" for none)
#   $8 — mtime (touch -t format: YYYYMMDDHHmm)
write_session() {
    local profile_name="$1" encoded_cwd="$2" uuid="$3" cwd="$4" branch="$5" prompt="$6" slug="$7" mtime="$8"
    local config_dir="${CLAUDE_PROFILES_DIR}/${profile_name}/config"
    local dir="${config_dir}/projects/${encoded_cwd}"
    local file="${dir}/${uuid}.jsonl"

    mkdir -p "${dir}"

    # Permission mode record (always first in a session JSONL)
    echo '{"type":"permission-mode","permissionMode":"default","sessionId":"'"${uuid}"'"}' > "${file}"

    # User record — contains cwd, branch, and the first prompt
    echo '{"type":"user","cwd":"'"${cwd}"'","gitBranch":"'"${branch}"'","message":{"role":"user","content":"'"${prompt}"'"},"uuid":"user-001","timestamp":"2026-04-15T12:00:00Z","sessionId":"'"${uuid}"'"}' >> "${file}"

    # Assistant record with slug (if provided). The slug is Claude's
    # auto-generated pretty name for the session.
    if [[ -n "${slug}" ]]; then
        echo '{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Working on it..."}]},"uuid":"asst-001","timestamp":"2026-04-15T12:00:01Z","sessionId":"'"${uuid}"'","slug":"'"${slug}"'"}' >> "${file}"
    fi

    # Set the modification time for deterministic sort ordering
    touch -t "${mtime}" "${file}"
}

# ---------------------------------------------------------------------------
# Create profiles
# ---------------------------------------------------------------------------

create_profile "work"     33  "team"     # Blue, team subscription
create_profile "personal" 208 "pro"      # Orange, pro subscription

# ---------------------------------------------------------------------------
# Create fake sessions for the "work" profile
# ---------------------------------------------------------------------------

# --- Repo 1: api-service (2 sessions) ---
write_session "work" \
    "-Users-matt-git-acme-api-service" \
    "a1b2c3d4-1111-2222-3333-444444444444" \
    "/Users/matt/git/acme/api-service" \
    "main" \
    "fix the flaky integration test in the auth middleware" \
    "sparkling-wondering-sloth" \
    "202604151430"

write_session "work" \
    "-Users-matt-git-acme-api-service" \
    "e5f6a7b8-1111-2222-3333-444444444444" \
    "/Users/matt/git/acme/api-service" \
    "feature/oauth-pkce" \
    "add OAuth2 PKCE flow to the login endpoint" \
    "peppy-hatching-reef" \
    "202604140915"

# --- Repo 2: frontend (1 session, recent) ---
write_session "work" \
    "-Users-matt-git-acme-frontend" \
    "b2c3d4e5-2222-3333-4444-555555555555" \
    "/Users/matt/git/acme/frontend" \
    "main" \
    "why is the bundle size 2MB larger after the last merge?" \
    "cozy-greeting-stonebraker" \
    "202604151100"

# --- Repo 3: infrastructure (2 sessions, one without slug) ---
write_session "work" \
    "-Users-matt-git-acme-infrastructure" \
    "c3d4e5f6-3333-4444-5555-666666666666" \
    "/Users/matt/git/acme/infrastructure" \
    "fix/dns-ttl" \
    "the DNS TTL for api.acme.com is too high, causing failover delays" \
    "golden-questing-zebra" \
    "202604141600"

write_session "work" \
    "-Users-matt-git-acme-infrastructure" \
    "d4e5f6a7-3333-4444-5555-777777777777" \
    "/Users/matt/git/acme/infrastructure" \
    "main" \
    "review the Terraform plan for the new staging environment" \
    "" \
    "202604131400"

# --- Repo 4: mobile-app (1 session) ---
write_session "work" \
    "-Users-matt-git-acme-mobile-app" \
    "f7a8b9c0-4444-5555-6666-888888888888" \
    "/Users/matt/git/acme/mobile-app" \
    "main" \
    "investigate the crash on iOS 18 when opening the photo picker" \
    "zippy-yawning-manatee" \
    "202604121000"

# --- Repo 5: data-pipeline (1 session, older) ---
write_session "work" \
    "-Users-matt-git-acme-data-pipeline" \
    "0a1b2c3d-5555-6666-7777-999999999999" \
    "/Users/matt/git/acme/data-pipeline" \
    "feature/backfill" \
    "backfill the missing events from March into the analytics warehouse" \
    "tranquil-hugging-beaver" \
    "202604101800"

# ---------------------------------------------------------------------------
# Create fake sessions for the "personal" profile
# ---------------------------------------------------------------------------

write_session "personal" \
    "-Users-matt-git-personal-blog" \
    "aa112233-6666-7777-8888-aaaaaaaaaaaa" \
    "/Users/matt/git/personal/blog" \
    "main" \
    "write a post about managing multiple Claude subscriptions" \
    "bubbly-skipping-sunrise" \
    "202604151200"

write_session "personal" \
    "-Users-matt-git-personal-dotfiles" \
    "bb223344-7777-8888-9999-bbbbbbbbbbbb" \
    "/Users/matt/git/personal/dotfiles" \
    "main" \
    "add zsh aliases for claude-profile" \
    "" \
    "202604140800"

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------

work_sessions=$(find "${CLAUDE_PROFILES_DIR}/work/config/projects" -name '*.jsonl' 2>/dev/null | wc -l | tr -d ' ')
personal_sessions=$(find "${CLAUDE_PROFILES_DIR}/personal/config/projects" -name '*.jsonl' 2>/dev/null | wc -l | tr -d ' ')

echo "Demo data created at: ${CLAUDE_PROFILES_DIR}"
echo "Profiles: work (${work_sessions} sessions), personal (${personal_sessions} sessions)"
