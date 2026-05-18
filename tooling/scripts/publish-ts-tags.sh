#!/usr/bin/env bash
# Publish all TypeScript package tags for platform v0.1.0 release.
#
# This script ONLY creates and pushes git tags. The actual npm publish is
# handled by the GitHub Actions workflow `.github/workflows/publish-ts-package.yml`
# which triggers on these tags.
#
# Pre-requisites:
#   - Repo secret NPM_TOKEN configured at
#     https://github.com/devpablocristo/platform/settings/secrets/actions
#   - Local HEAD is on main with workflow file pushed.
#
# Usage:
#   bash tooling/scripts/publish-ts-tags.sh           # actually push
#   bash tooling/scripts/publish-ts-tags.sh --dry-run # preview only

set -euo pipefail

DRY_RUN=false
if [[ "${1:-}" == "--dry-run" ]]; then
    DRY_RUN=true
fi

PACKAGES=(
    # L0 capabilities
    "authn/ts"
    "browser/ts"
    "concurrency/fsm/ts"
    "http/ts"

    # Cross-language contracts
    "contracts/ai/ts"

    # Feature verticals
    "features/admin-insights/ts"
    "features/calendar-board/ts"
    "features/conversation-inbox/ts"
    "features/crud/ui/ts"
    "features/kanban-board/ts"
    "features/notification-feed/ts"
    "features/scheduling/ts"
    "features/search/ts"

    # UI primitives
    "ui/data-display/ts"
    "ui/modal/ts"
    "ui/page-shell/ts"
    "ui/section-hub/ts"
    "ui/shell-sidebar/ts"
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
git fetch origin main >/dev/null 2>&1
if [[ "$(git rev-parse HEAD)" != "$(git rev-parse origin/main)" ]]; then
    echo "ERROR: local main does not match origin/main. Run git pull --ff-only origin main"
    exit 1
fi

echo "==> Publishing ${#PACKAGES[@]} TS package tags from HEAD ($(git rev-parse HEAD))"
[[ "$DRY_RUN" == "true" ]] && echo "==> DRY RUN MODE"
echo
echo "NOTE: pushing each tag triggers the publish-ts-package GitHub Actions"
echo "workflow which performs the actual npm publish. Check progress at:"
echo "  https://github.com/devpablocristo/platform/actions"
echo

CREATED=0; PUSHED=0; SKIPPED=0; FAILED=()

for pkg in "${PACKAGES[@]}"; do
    tag="$pkg/v0.1.0"
    version_file="$pkg/VERSION"
    pkg_json="$pkg/package.json"

    if [[ ! -f "$version_file" ]]; then
        echo "  [WARN] $pkg: VERSION file missing"
        FAILED+=("$pkg (no VERSION)")
        continue
    fi
    if [[ ! -f "$pkg_json" ]]; then
        echo "  [WARN] $pkg: package.json missing"
        FAILED+=("$pkg (no package.json)")
        continue
    fi

    actual=$(tr -d '[:space:]' < "$version_file")
    if [[ "$actual" != "0.1.0" ]]; then
        echo "  [WARN] $pkg: VERSION=$actual, expected 0.1.0"
        FAILED+=("$pkg (VERSION mismatch: $actual)")
        continue
    fi

    if git rev-parse --verify "refs/tags/$tag" >/dev/null 2>&1; then
        echo "  [skip-local]  $tag already exists locally"
    else
        if [[ "$DRY_RUN" == "true" ]]; then
            echo "  [would-tag]   $tag"
        else
            git tag "$tag" -m "$pkg: v0.1.0 — initial platform release"
            echo "  [tagged]      $tag"
            CREATED=$((CREATED+1))
        fi
    fi

    if git ls-remote --tags origin "refs/tags/$tag" 2>/dev/null | grep -q "$tag"; then
        echo "  [skip-remote] $tag already pushed"
        SKIPPED=$((SKIPPED+1))
    else
        if [[ "$DRY_RUN" == "true" ]]; then
            echo "  [would-push]  $tag"
        else
            git push origin "$tag" >/dev/null
            echo "  [pushed]      $tag (workflow will pick it up)"
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
    for f in "${FAILED[@]}"; do echo "      - $f"; done
fi

if [[ "$DRY_RUN" == "false" && $PUSHED -gt 0 ]]; then
    echo
    echo "==> Watch workflow runs at:"
    echo "    https://github.com/devpablocristo/platform/actions"
    echo
    echo "==> Once green, smoke-test resolution with:"
    echo "    npm view @devpablocristo/platform-browser version"
fi
