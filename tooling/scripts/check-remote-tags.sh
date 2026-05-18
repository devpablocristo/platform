#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REMOTE="${1:-origin}"
TAGS_FILE="$(mktemp)"
trap 'rm -f "${TAGS_FILE}"' EXIT

discover_modules() {
  {
    find "${ROOT_DIR}" -type f -path '*/go/go.mod'
    find "${ROOT_DIR}" -type f -path '*/python/pyproject.toml'
    find "${ROOT_DIR}" -type f -path '*/rust/Cargo.toml'
    find "${ROOT_DIR}" -type f -path '*/ts/package.json'
  } | while IFS= read -r manifest; do
    dirname "${manifest}"
  done | sed "s#^${ROOT_DIR}/##" | sort
}

git ls-remote --tags "${REMOTE}" \
  | awk '{print $2}' \
  | sed 's#^refs/tags/##' \
  | sed 's/\^{}$//' \
  | sort -u > "${TAGS_FILE}"

if [[ ! -s "${TAGS_FILE}" ]]; then
  echo "no remote tags found for ${REMOTE}" >&2
  exit 1
fi

module_count=0
missing_count=0

while IFS= read -r module; do
  [[ -n "${module}" ]] || continue
  module_count=$((module_count + 1))

  version_file="${ROOT_DIR}/${module}/VERSION"
  if [[ ! -f "${version_file}" ]]; then
    echo "missing VERSION: ${module}" >&2
    exit 1
  fi

  version="$(tr -d '[:space:]' < "${version_file}")"
  expected_tag="${module}/v${version}"

  if ! grep -Fxq "${expected_tag}" "${TAGS_FILE}"; then
    echo "missing remote tag: ${expected_tag}"
    missing_count=$((missing_count + 1))
  fi
done < <(discover_modules)

if [[ "${module_count}" -eq 0 ]]; then
  echo "no modules discovered" >&2
  exit 1
fi

if [[ "${missing_count}" -gt 0 ]]; then
  echo "missing ${missing_count} remote tags in ${REMOTE}" >&2
  exit 1
fi

echo "all expected tags are present in ${REMOTE}"
