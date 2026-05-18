#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

discover_modules() {
  {
    find "${ROOT_DIR}" -type f -path '*/go/go.mod' -printf '%h\n'
    find "${ROOT_DIR}" -type f -path '*/python/pyproject.toml' -printf '%h\n'
    find "${ROOT_DIR}" -type f -path '*/rust/Cargo.toml' -printf '%h\n'
    find "${ROOT_DIR}" -type f -path '*/ts/package.json' -printf '%h\n'
  } | sed "s#^${ROOT_DIR}/##" | sort
}

printf "%-28s %-12s %s\n" "module" "version" "tag"
printf "%-28s %-12s %s\n" "------" "-------" "---"

while IFS= read -r module; do
  version_file="${ROOT_DIR}/${module}/VERSION"
  if [[ ! -f "${version_file}" ]]; then
    printf "%-28s %-12s %s\n" "${module}" "missing" "missing VERSION"
    continue
  fi

  version="$(tr -d '[:space:]' < "${version_file}")"
  printf "%-28s %-12s %s\n" "${module}" "${version}" "${module}/v${version}"
done < <(discover_modules)
