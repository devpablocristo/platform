#!/usr/bin/env bash
# Publish all Go module tags for platform v0.1.0 release (Fase A7).
#
# This script:
#   1. Verifies each go.mod has VERSION 0.1.0.
#   2. Creates and pushes a tag <module-path>/v0.1.0 for each Go module.
#   3. Skips tags that already exist (idempotent).
#   4. Stops on first failure so you can inspect.
#
# Usage:
#   bash tooling/scripts/publish-go-tags.sh           # actually publish
#   bash tooling/scripts/publish-go-tags.sh --dry-run # show what would happen
#
# Pre-requisites:
#   - cwd is the platform repo root.
#   - Git remote 'origin' points to github.com/devpablocristo/platform.
#   - Local HEAD is on main with all changes pushed.
#   - errors/go/v0.1.0 already published as smoke test (will be skipped here).

set -euo pipefail

DRY_RUN=false
if [[ "${1:-}" == "--dry-run" ]]; then
    DRY_RUN=true
fi

# Topological order: no internal deps first, then deps grow upward.
MODULES=(
    # Round 1 — leaves (no internal platform deps, or already published)
    "errors/go"                       # already published; will skip
    "validate/go"
    "config/go"
    "observability/go"
    "concurrency/go"
    "contracts/ai/go"
    "calendar/ics/go"
    "ingestion/go"
    "eventing/go"
    "webhook/go"
    "jobs/go"
    "databases/dynamodb/go"
    "features/crud/paths/go"

    # Round 2 — depend on errors
    "databases/postgres/go"
    "features/crud/archive/go"
    "http/go"

    # Round 3 — depend on http and others
    "security/go"
    "authn/go"
    "authz/go"
    "notifications/go"
    "http/gin/go"

    # Round 4 — SDKs (external services)
    "sdks/aws/lambda/go"
    "sdks/aws/s3/go"
    "sdks/aws/sqs/go"
    "sdks/google-calendar/go"

    # Round 5 — kernels
    "kernels/activity/go"
    "kernels/ai/go"
    "kernels/artifact/go"
    "kernels/governance/go"
    "kernels/saas/go"

    # Round 6 — feature verticals
    "features/scheduling/go"
)

# Sanity checks
if ! git rev-parse --abbrev-ref HEAD | grep -q '^main$'; then
    echo "ERROR: not on main branch. Current: $(git rev-parse --abbrev-ref HEAD)"
    exit 1
fi

if ! git diff --quiet || ! git diff --cached --quiet; then
    echo "ERROR: working tree is dirty. Commit or stash first."
    exit 1
fi

# Make sure local main is in sync with remote
if ! git fetch origin main >/dev/null 2>&1; then
    echo "ERROR: failed to fetch origin/main"
    exit 1
fi
LOCAL_HEAD=$(git rev-parse HEAD)
REMOTE_HEAD=$(git rev-parse origin/main)
if [[ "$LOCAL_HEAD" != "$REMOTE_HEAD" ]]; then
    echo "ERROR: local main ($LOCAL_HEAD) does not match origin/main ($REMOTE_HEAD)"
    echo "Run: git pull --ff-only origin main"
    exit 1
fi

echo "==> Publishing ${#MODULES[@]} Go module tags from HEAD ($LOCAL_HEAD)"
[[ "$DRY_RUN" == "true" ]] && echo "==> DRY RUN MODE (no tags will be created or pushed)"
echo

CREATED=0
SKIPPED=0
PUSHED=0
FAILED=()

for mod in "${MODULES[@]}"; do
    tag="$mod/v0.1.0"
    version_file="$mod/VERSION"

    if [[ ! -f "$version_file" ]]; then
        echo "  [WARN] $mod: VERSION file not found, skipping"
        FAILED+=("$mod (no VERSION)")
        continue
    fi

    actual=$(cat "$version_file" | tr -d '[:space:]')
    if [[ "$actual" != "0.1.0" ]]; then
        echo "  [WARN] $mod: VERSION=$actual, expected 0.1.0, skipping"
        FAILED+=("$mod (VERSION mismatch: $actual)")
        continue
    fi

    if git rev-parse --verify "refs/tags/$tag" >/dev/null 2>&1; then
        echo "  [skip-local]  $tag already exists locally"
    else
        if [[ "$DRY_RUN" == "true" ]]; then
            echo "  [would-tag]   $tag"
        else
            git tag "$tag" -m "$mod: v0.1.0 — initial platform release"
            echo "  [tagged]      $tag"
            CREATED=$((CREATED+1))
        fi
    fi

    # Check if tag exists on remote
    if git ls-remote --tags origin "refs/tags/$tag" 2>/dev/null | grep -q "$tag"; then
        echo "  [skip-remote] $tag already pushed"
        SKIPPED=$((SKIPPED+1))
    else
        if [[ "$DRY_RUN" == "true" ]]; then
            echo "  [would-push]  $tag"
        else
            git push origin "$tag" >/dev/null
            echo "  [pushed]      $tag"
            PUSHED=$((PUSHED+1))
        fi
    fi
done

echo
echo "==> Summary"
echo "    Tags created locally: $CREATED"
echo "    Tags pushed:          $PUSHED"
echo "    Tags already remote:  $SKIPPED"
if [[ ${#FAILED[@]} -gt 0 ]]; then
    echo "    Modules with issues:  ${#FAILED[@]}"
    for f in "${FAILED[@]}"; do
        echo "      - $f"
    done
fi

if [[ "$DRY_RUN" == "false" && $PUSHED -gt 0 ]]; then
    echo
    echo "==> Next: verify the proxy can resolve a sample with:"
    echo "    GOPROXY=https://proxy.golang.org go list -m github.com/devpablocristo/platform/validate/go@v0.1.0"
fi
