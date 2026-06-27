#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

discover_go_modules() {
  find "${ROOT_DIR}" -type f -name go.mod | while IFS= read -r manifest; do
    dirname "${manifest}"
  done | sed "s#^${ROOT_DIR}/##" | sort
}

module_count=0

while IFS= read -r module; do
	[[ -n "${module}" ]] || continue
	module_count=$((module_count + 1))
	echo "==> go test ${module}"
	(
		cd "${ROOT_DIR}/${module}"
		go test ./...
	)
done < <(discover_go_modules)

if [[ "${module_count}" -eq 0 ]]; then
	echo "no go modules found" >&2
	exit 1
fi
