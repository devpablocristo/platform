#!/usr/bin/env bash
# Publish Python package tags for platform v0.1.0 release.
#
# Like publish-ts-tags.sh: only creates/pushes git tags. The actual PyPI
# publish is done by .github/workflows/publish-python-package.yml.
#
# Pre-requisite: PYPI_API_TOKEN configured as a repo secret in
# https://github.com/devpablocristo/platform/settings/secrets/actions

set -euo pipefail

DRY_RUN=false
if [[ "${1:-}" == "--dry-run" ]]; then
    DRY_RUN=true
fi

PACKAGES=(
    "http/python"
    "kernels/ai/runtime/python"
)

# Sanity checks
if ! git rev-parse --abbrev-ref HEAD | grep -q '^main$'; then
    echo "ERROR: not on main branch. Current: $(git rev-parse --abbrev-ref HEAD)"
    exit 1
fi
if ! git diff --quiet || ! git diff --cached --quiet; then
    echo "ERROR: working tree is dirty"
    exit 1
fi
git fetch origin main >/dev/null 2>&1
if [[ "$(git rev-parse HEAD)" != "$(git rev-parse origin/main)" ]]; then
    echo "ERROR: local main does not match origin/main"
    exit 1
fi

echo "==> Publishing ${#PACKAGES[@]} Python package tags from HEAD ($(git rev-parse HEAD))"
[[ "$DRY_RUN" == "true" ]] && echo "==> DRY RUN MODE"
echo

for pkg in "${PACKAGES[@]}"; do
    tag="$pkg/v0.1.0"
    version_file="$pkg/VERSION"
    pyproject="$pkg/pyproject.toml"

    if [[ ! -f "$version_file" ]]; then
        echo "  [WARN] $pkg: VERSION file missing — skipping"
        continue
    fi
    if [[ ! -f "$pyproject" ]]; then
        echo "  [WARN] $pkg: pyproject.toml missing — skipping"
        continue
    fi

    actual=$(tr -d '[:space:]' < "$version_file")
    if [[ "$actual" != "0.1.0" ]]; then
        echo "  [WARN] $pkg: VERSION=$actual, expected 0.1.0"
        continue
    fi

    if git rev-parse --verify "refs/tags/$tag" >/dev/null 2>&1; then
        echo "  [skip-local]  $tag exists locally"
    else
        if [[ "$DRY_RUN" == "true" ]]; then
            echo "  [would-tag]   $tag"
        else
            git tag "$tag" -m "$pkg: v0.1.0 — initial platform release"
            echo "  [tagged]      $tag"
        fi
    fi

    if git ls-remote --tags origin "refs/tags/$tag" 2>/dev/null | grep -q "$tag"; then
        echo "  [skip-remote] $tag already pushed"
    else
        if [[ "$DRY_RUN" == "true" ]]; then
            echo "  [would-push]  $tag"
        else
            git push origin "$tag" >/dev/null
            echo "  [pushed]      $tag (publish-python-package workflow will pick up)"
        fi
    fi
done

if [[ "$DRY_RUN" == "false" ]]; then
    echo
    echo "==> Watch:    https://github.com/devpablocristo/platform/actions"
    echo "==> Verify:   pip index versions devpablocristo-platform-http"
    echo "              pip index versions devpablocristo-platform-ai-runtime"
fi
