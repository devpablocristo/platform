#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

invalid=0

while IFS= read -r manifest; do
  [[ -n "${manifest}" ]] || continue
  rel="${manifest#${ROOT_DIR}/}"
  case "${rel}" in
    */go/go.mod|*/python/pyproject.toml|*/rust/Cargo.toml|*/ts/package.json)
      ;;
    *)
      echo "invalid runtime manifest path: ${rel}" >&2
      invalid=1
      ;;
  esac
done < <(find "${ROOT_DIR}" -type f \( -name go.mod -o -name pyproject.toml -o -name Cargo.toml -o -path '*/ts/package.json' \))

if [[ "${invalid}" -ne 0 ]]; then
  exit 1
fi

echo "runtime layout validated"
