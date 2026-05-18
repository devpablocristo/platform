#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

mapfile -t modules < <(
  find "${ROOT_DIR}" -type f -path '*/ts/package.json' | while IFS= read -r manifest; do
    dirname "${manifest}"
  done | sort
)

if [[ "${#modules[@]}" -eq 0 ]]; then
  echo "no ts modules found"
  exit 0
fi

for module in "${modules[@]}"; do
  rel="${module#${ROOT_DIR}/}"
  echo "==> testing ${rel}"
  (cd "${module}" && npm ci --no-audit --no-fund && npm run typecheck && npm test)
done
