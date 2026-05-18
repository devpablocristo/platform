#!/usr/bin/env bash
# Mark all legacy @devpablocristo/{core,modules}-* npm packages as deprecated.
# Run after platform-* equivalents are published. Idempotent.
#
# Pre-requisite: npm login (active session) with write access to @devpablocristo scope.

set -euo pipefail

DRY_RUN=false
OTP=""
for arg in "$@"; do
    case "$arg" in
        --dry-run) DRY_RUN=true ;;
        --otp=*) OTP="${arg#--otp=}" ;;
    esac
done

OTP_FLAG=()
[[ -n "$OTP" ]] && OTP_FLAG=(--otp="$OTP")

# old -> new (used to build the deprecation message)
declare -A MAP=(
    ["core-authn"]="platform-authn"
    ["core-browser"]="platform-browser"
    ["core-http"]="platform-http"
    ["core-fsm"]="platform-fsm"
    ["core-ai-contracts"]="platform-ai-contracts"
    ["modules-scheduling"]="platform-scheduling"
    ["modules-search"]="platform-search"
    ["modules-shell-sidebar"]="platform-shell-sidebar"
    ["modules-admin-insights"]="platform-admin-insights"
    ["modules-calendar-board"]="platform-calendar-board"
    ["modules-kanban-board"]="platform-kanban-board"
    ["modules-crud-ui"]="platform-crud-ui"
    ["modules-ui-data-display"]="platform-ui-data-display"
    ["modules-ui-modal"]="platform-ui-modal"
    ["modules-ui-section-hub"]="platform-ui-section-hub"
    ["modules-ui-page-shell"]="platform-ui-page-shell"
    ["modules-ui-conversation-inbox"]="platform-conversation-inbox"
    ["modules-ui-notification-feed"]="platform-notification-feed"
)

# These have no platform-* replacement (deprecated pre-migration, no consumers)
NO_REPLACEMENT=(
    "modules-ai-console"
    "modules-ui-filters"
    "modules-ui-forms"
    "modules-work-orders"
)

DOC_URL="https://github.com/devpablocristo/platform/blob/main/docs/migration/DEPRECATED_PACKAGES.md"

echo "==> Deprecating ${#MAP[@]} legacy npm packages (with replacement)"
[[ "$DRY_RUN" == "true" ]] && echo "==> DRY RUN MODE"
echo

for old in "${!MAP[@]}"; do
    new="${MAP[$old]}"
    msg="Moved to @devpablocristo/${new}"
    if [[ "$DRY_RUN" == "true" ]]; then
        echo "  [would-deprecate] @devpablocristo/${old} -> @devpablocristo/${new}"
    else
        echo "  [deprecating]     @devpablocristo/${old} -> @devpablocristo/${new}"
        npm deprecate "${OTP_FLAG[@]}" "@devpablocristo/${old}@*" "${msg}" 2>&1 | sed 's/^/    /' || true
    fi
done

echo
echo "==> Deprecating ${#NO_REPLACEMENT[@]} legacy npm packages (no replacement, no active consumers)"

for old in "${NO_REPLACEMENT[@]}"; do
    msg="Deprecated; no platform-* replacement available"
    if [[ "$DRY_RUN" == "true" ]]; then
        echo "  [would-deprecate] @devpablocristo/${old} (no replacement)"
    else
        echo "  [deprecating]     @devpablocristo/${old} (no replacement)"
        npm deprecate "${OTP_FLAG[@]}" "@devpablocristo/${old}@*" "${msg}" 2>&1 | sed 's/^/    /' || true
    fi
done

echo
echo "==> Done. Verify with:"
echo "    npm view @devpablocristo/core-browser deprecated"
echo "    npm view @devpablocristo/modules-scheduling deprecated"
