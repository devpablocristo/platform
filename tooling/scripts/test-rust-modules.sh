#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if ! command -v cargo >/dev/null 2>&1 && [[ -f "${HOME}/.cargo/env" ]]; then
  # shellcheck source=/dev/null
  source "${HOME}/.cargo/env"
fi

if ! command -v cargo >/dev/null 2>&1; then
  echo "cargo not found: install Rust to test rust modules" >&2
  exit 1
fi

mapfile -t modules < <(find "${ROOT_DIR}" -type f -path '*/rust/Cargo.toml' -printf '%h\n' | sort)

if [[ "${#modules[@]}" -eq 0 ]]; then
  echo "no rust modules found"
  exit 0
fi

for module in "${modules[@]}"; do
  rel="${module#${ROOT_DIR}/}"
  echo "==> cargo test ${rel}"
  (
    cd "${module}"
    cargo test
  )
done
