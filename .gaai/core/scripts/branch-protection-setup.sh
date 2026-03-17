#!/usr/bin/env bash
set -euo pipefail

# ═══════════════════════════════════════════════════════════════════════════
# GAAI Branch Protection Setup — configure GitHub branch protection via CLI
# ═══════════════════════════════════════════════════════════════════════════
#
# Description:
#   Creates the staging branch (from main) if it doesn't exist, then
#   configures GitHub branch protection rules, repo merge settings, and
#   local git hooks. Idempotent — safe to re-run.
#
# Usage:
#   bash .gaai/core/scripts/branch-protection-setup.sh [options]
#
# Options:
#   --main-branch <name>    production/main branch (default: main)
#   --staging-branch <name> staging branch (default: staging)
#   --required-checks <csv> comma-separated CI check names
#                           (default: "Framework Integrity Check")
#   --required-approvals <n> reviewers required for production PRs (default: 1)
#   --dry-run               show what would be done without applying
#   --yes                   skip confirmation prompt
#
# Environment overrides:
#   GAAI_MAIN_BRANCH        override --main-branch
#   GAAI_STAGING_BRANCH     override --staging-branch
#
# Exit codes:
#   0 — all configuration applied successfully
#   1 — usage error or missing prerequisite
#   2 — GitHub API error
# ═══════════════════════════════════════════════════════════════════════════

# ── Defaults ─────────────────────────────────────────────────────────────

MAIN_BRANCH="${GAAI_MAIN_BRANCH:-main}"
STAGING_BRANCH="${GAAI_STAGING_BRANCH:-staging}"
REQUIRED_CHECKS="Framework Integrity Check"
REQUIRED_APPROVALS=1
DRY_RUN=false
YES=false

# ── Parse args ───────────────────────────────────────────────────────────

while [[ $# -gt 0 ]]; do
  case "$1" in
    --main-branch)       MAIN_BRANCH="$2";       shift 2 ;;
    --staging-branch)    STAGING_BRANCH="$2";     shift 2 ;;
    --required-checks)   REQUIRED_CHECKS="$2";    shift 2 ;;
    --required-approvals) REQUIRED_APPROVALS="$2"; shift 2 ;;
    --dry-run)           DRY_RUN=true;            shift ;;
    --yes)               YES=true;                shift ;;
    -h|--help)
      awk 'NR>6 && /^# ═/{exit} NR>6 && /^#/{sub(/^# ?/,""); print}' "$0"
      exit 0
      ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

# ── Helpers ──────────────────────────────────────────────────────────────

PASS=0
FAIL=0
SKIP=0

pass() { echo "  ✅ $1"; PASS=$((PASS + 1)); }
fail() { echo "  ❌ $1"; FAIL=$((FAIL + 1)); }
skip() { echo "  ⏭️  $1"; SKIP=$((SKIP + 1)); }
info() { echo "  → $1"; }

run_or_dry() {
  local desc="$1"
  shift
  if [[ "$DRY_RUN" == "true" ]]; then
    skip "(dry-run) $desc"
    return 0
  fi
  if "$@" 2>/dev/null; then
    pass "$desc"
    return 0
  else
    fail "$desc"
    return 1
  fi
}

# Build JSON array from comma-separated check names
checks_to_json_array() {
  local IFS=','
  local first=true
  printf '['
  for check in $1; do
    check="$(echo "$check" | sed 's/^ *//;s/ *$//')"
    if [[ "$first" == "true" ]]; then
      first=false
    else
      printf ','
    fi
    printf '"%s"' "$check"
  done
  printf ']'
}

# ── Locate project root ─────────────────────────────────────────────────

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CORE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
PROJECT_ROOT="$(cd "$CORE_DIR/../.." && pwd)"

# ── Pre-flight checks ───────────────────────────────────────────────────

echo ""
echo "GAAI Branch Protection Setup"
echo "  project:  $PROJECT_ROOT"
echo "  main:     $MAIN_BRANCH"
echo "  staging:  $STAGING_BRANCH"
echo "  checks:   $REQUIRED_CHECKS"
echo "  approvals: $REQUIRED_APPROVALS (production)"
if [[ "$DRY_RUN" == "true" ]]; then
  echo "  mode:     DRY RUN"
fi
echo "================================"

echo ""
echo "[ Prerequisites ]"

# gh CLI
if command -v gh &>/dev/null; then
  pass "gh CLI found ($(gh --version 2>&1 | head -1))"
else
  fail "gh CLI not found — install: https://cli.github.com/"
  echo ""
  echo "❌ Cannot continue without gh CLI."
  exit 1
fi

# gh auth
if gh auth status &>/dev/null 2>&1; then
  pass "gh authenticated"
else
  fail "gh not authenticated — run: gh auth login"
  echo ""
  echo "❌ Cannot continue without authentication."
  exit 1
fi

# git repo
if git -C "$PROJECT_ROOT" rev-parse --is-inside-work-tree &>/dev/null; then
  pass "Inside a git repository"
else
  fail "Not inside a git repository"
  echo ""
  echo "❌ Cannot continue outside a git repo."
  exit 1
fi

# Detect repo owner/name from remote
REPO_SLUG="$(gh repo view --json nameWithOwner -q '.nameWithOwner' 2>/dev/null || echo "")"
if [[ -n "$REPO_SLUG" ]]; then
  pass "GitHub repo: $REPO_SLUG"
else
  fail "Could not detect GitHub repo — ensure a remote is configured"
  echo ""
  echo "❌ Cannot continue without a GitHub remote."
  exit 1
fi

# Main branch exists
if git -C "$PROJECT_ROOT" rev-parse --verify "origin/$MAIN_BRANCH" &>/dev/null 2>&1; then
  pass "origin/$MAIN_BRANCH exists"
else
  fail "origin/$MAIN_BRANCH not found — push your main branch first"
  echo ""
  echo "❌ Cannot continue without the main branch on the remote."
  exit 1
fi

# ── Confirmation ─────────────────────────────────────────────────────────

if [[ "$YES" != "true" && "$DRY_RUN" != "true" ]]; then
  echo ""
  echo "This will:"
  echo "  • Create '$STAGING_BRANCH' branch from '$MAIN_BRANCH' (if needed)"
  echo "  • Apply branch protection to '$MAIN_BRANCH' and '$STAGING_BRANCH'"
  echo "  • Enable auto-merge, auto-delete branches, squash-only merges"
  echo "  • Set local git hooks path to .githooks (if directory exists)"
  echo ""
  read -r -p "  Proceed? [y/N] " CONFIRM
  if [[ ! "$CONFIRM" =~ ^[yY]$ ]]; then
    echo ""
    echo "Cancelled."
    exit 0
  fi
fi

# ── 1. Create staging branch ────────────────────────────────────────────

echo ""
echo "[ Staging Branch ]"

if git -C "$PROJECT_ROOT" rev-parse --verify "origin/$STAGING_BRANCH" &>/dev/null 2>&1; then
  pass "$STAGING_BRANCH already exists on remote"
elif git -C "$PROJECT_ROOT" rev-parse --verify "$STAGING_BRANCH" &>/dev/null 2>&1; then
  # Exists locally but not on remote — push it
  info "$STAGING_BRANCH exists locally, pushing to remote..."
  run_or_dry "Push $STAGING_BRANCH to origin" \
    git -C "$PROJECT_ROOT" push -u origin "$STAGING_BRANCH"
else
  # Does not exist — create from main
  info "Creating $STAGING_BRANCH from $MAIN_BRANCH..."
  if [[ "$DRY_RUN" == "true" ]]; then
    skip "(dry-run) Create $STAGING_BRANCH from origin/$MAIN_BRANCH"
  else
    if git -C "$PROJECT_ROOT" branch "$STAGING_BRANCH" "origin/$MAIN_BRANCH" && \
       git -C "$PROJECT_ROOT" push -u origin "$STAGING_BRANCH"; then
      pass "Created $STAGING_BRANCH from $MAIN_BRANCH and pushed to origin"
    else
      fail "Could not create $STAGING_BRANCH"
    fi
  fi
fi

# ── 2. Repo settings ────────────────────────────────────────────────────

echo ""
echo "[ Repo Settings ]"

run_or_dry "Enable auto-merge" \
  gh api -X PATCH "/repos/$REPO_SLUG" \
    -F allow_auto_merge=true \
    --silent

run_or_dry "Enable auto-delete head branches" \
  gh api -X PATCH "/repos/$REPO_SLUG" \
    -F delete_branch_on_merge=true \
    --silent

run_or_dry "Allow squash merges" \
  gh api -X PATCH "/repos/$REPO_SLUG" \
    -F allow_squash_merge=true \
    --silent

run_or_dry "Disable plain merge commits" \
  gh api -X PATCH "/repos/$REPO_SLUG" \
    -F allow_merge_commit=false \
    --silent

run_or_dry "Disable rebase merges" \
  gh api -X PATCH "/repos/$REPO_SLUG" \
    -F allow_rebase_merge=false \
    --silent

# ── 3. Production branch protection ─────────────────────────────────────

echo ""
echo "[ Production Branch Protection: $MAIN_BRANCH ]"

CHECKS_JSON=$(checks_to_json_array "$REQUIRED_CHECKS")

PRODUCTION_PAYLOAD=$(cat <<ENDJSON
{
  "required_status_checks": {
    "strict": true,
    "contexts": $CHECKS_JSON
  },
  "enforce_admins": true,
  "required_pull_request_reviews": {
    "dismiss_stale_reviews": true,
    "required_approving_review_count": $REQUIRED_APPROVALS
  },
  "restrictions": null,
  "allow_force_pushes": false,
  "allow_deletions": false
}
ENDJSON
)

if [[ "$DRY_RUN" == "true" ]]; then
  skip "(dry-run) Apply protection to $MAIN_BRANCH"
  info "Payload: $PRODUCTION_PAYLOAD"
else
  if echo "$PRODUCTION_PAYLOAD" | gh api -X PUT "/repos/$REPO_SLUG/branches/$MAIN_BRANCH/protection" \
      --input - --silent 2>/dev/null; then
    pass "Branch protection applied to $MAIN_BRANCH"
    info "PR required: yes, approvals: $REQUIRED_APPROVALS, dismiss stale: yes"
    info "Status checks: $REQUIRED_CHECKS (strict: up-to-date required)"
    info "Force push: blocked, deletions: blocked, admins: enforced"
  else
    fail "Could not apply branch protection to $MAIN_BRANCH"
  fi
fi

# ── 4. Staging branch protection ────────────────────────────────────────

echo ""
echo "[ Staging Branch Protection: $STAGING_BRANCH ]"

STAGING_PAYLOAD=$(cat <<ENDJSON
{
  "required_status_checks": {
    "strict": false,
    "contexts": $CHECKS_JSON
  },
  "enforce_admins": false,
  "required_pull_request_reviews": {
    "dismiss_stale_reviews": true,
    "required_approving_review_count": 0
  },
  "restrictions": null,
  "allow_force_pushes": false,
  "allow_deletions": false
}
ENDJSON
)

if [[ "$DRY_RUN" == "true" ]]; then
  skip "(dry-run) Apply protection to $STAGING_BRANCH"
  info "Payload: $STAGING_PAYLOAD"
else
  if echo "$STAGING_PAYLOAD" | gh api -X PUT "/repos/$REPO_SLUG/branches/$STAGING_BRANCH/protection" \
      --input - --silent 2>/dev/null; then
    pass "Branch protection applied to $STAGING_BRANCH"
    info "PR required: yes (admins can bypass for daemon status commits)"
    info "Status checks: $REQUIRED_CHECKS (strict: no — allows parallel deliveries)"
    info "Force push: blocked, deletions: blocked"
  else
    fail "Could not apply branch protection to $STAGING_BRANCH"
  fi
fi

# ── 5. Local git hooks ──────────────────────────────────────────────────

echo ""
echo "[ Local Git Hooks ]"

if [[ -d "$PROJECT_ROOT/.githooks" ]]; then
  CURRENT_HOOKS=$(git -C "$PROJECT_ROOT" config --get core.hooksPath 2>/dev/null || echo "")
  if [[ "$CURRENT_HOOKS" == ".githooks" ]]; then
    pass "git core.hooksPath already set to .githooks"
  else
    run_or_dry "Set git core.hooksPath to .githooks" \
      git -C "$PROJECT_ROOT" config core.hooksPath .githooks
  fi
else
  skip "No .githooks/ directory — pre-push safety hook not available"
fi

# ── Summary ──────────────────────────────────────────────────────────────

echo ""
echo "================================"
echo "Results: $PASS passed, $FAIL failed, $SKIP skipped"
echo ""

if [[ $FAIL -gt 0 ]]; then
  echo "❌ Setup incomplete — review the failures above."
  exit 2
else
  if [[ "$DRY_RUN" == "true" ]]; then
    echo "✅ Dry run complete. Re-run without --dry-run to apply."
  else
    echo "✅ Branch protection setup complete."
    echo ""
    echo "  Verify with:"
    echo "    gh api /repos/$REPO_SLUG/branches/$MAIN_BRANCH/protection | jq"
    echo "    gh api /repos/$REPO_SLUG/branches/$STAGING_BRANCH/protection | jq"
  fi
  echo ""
  exit 0
fi
