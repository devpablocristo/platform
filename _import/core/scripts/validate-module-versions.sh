#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SEMVER_REGEX='^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$'

discover_modules() {
  {
    find "${ROOT_DIR}" -type f -path '*/go/go.mod' -printf '%h\n'
    find "${ROOT_DIR}" -type f -path '*/python/pyproject.toml' -printf '%h\n'
    find "${ROOT_DIR}" -type f -path '*/rust/Cargo.toml' -printf '%h\n'
    find "${ROOT_DIR}" -type f -path '*/ts/package.json' -printf '%h\n'
  } | sed "s#^${ROOT_DIR}/##" | sort
}

module_count=0

while IFS= read -r module; do
  [[ -n "${module}" ]] || continue
  module_count=$((module_count + 1))

  version_file="${ROOT_DIR}/${module}/VERSION"
  if [[ ! -f "${version_file}" ]]; then
    echo "missing VERSION: ${module}" >&2
    exit 1
  fi

  version="$(tr -d '[:space:]' < "${version_file}")"
  if [[ ! "${version}" =~ ${SEMVER_REGEX} ]]; then
    echo "invalid semver in ${module}/VERSION: ${version}" >&2
    exit 1
  fi

  case "${module}" in
    */go)
      go_mod="${ROOT_DIR}/${module}/go.mod"
      expected_module="github.com/devpablocristo/core/${module}"
      actual_module="$(sed -n 's/^module //p' "${go_mod}" | head -n1 | tr -d '[:space:]')"
      if [[ "${actual_module}" != "${expected_module}" ]]; then
        echo "go.mod module mismatch in ${module}: expected ${expected_module}, got ${actual_module}" >&2
        exit 1
      fi
      ;;
    */python)
      pyproject="${ROOT_DIR}/${module}/pyproject.toml"
      actual_version="$(awk -F'\"' '/^[[:space:]]*version = \"/ {print $2; exit}' "${pyproject}")"
      if [[ -z "${actual_version}" ]]; then
        echo "missing [project].version in ${module}/pyproject.toml" >&2
        exit 1
      fi
      if [[ "${actual_version}" != "${version}" ]]; then
        echo "pyproject version mismatch in ${module}: VERSION=${version} pyproject=${actual_version}" >&2
        exit 1
      fi
      ;;
    */rust)
      cargo_toml="${ROOT_DIR}/${module}/Cargo.toml"
      actual_version="$(awk -F'\"' '/^[[:space:]]*version = \"/ {print $2; exit}' "${cargo_toml}")"
      if [[ -z "${actual_version}" ]]; then
        echo "missing package.version in ${module}/Cargo.toml" >&2
        exit 1
      fi
      if [[ "${actual_version}" != "${version}" ]]; then
        echo "Cargo.toml version mismatch in ${module}: VERSION=${version} cargo=${actual_version}" >&2
        exit 1
      fi
      ;;
    */ts)
      package_json="${ROOT_DIR}/${module}/package.json"
      actual_version="$(awk -F'\"' '/^[[:space:]]*\"version\": \"/ {print $4; exit}' "${package_json}")"
      if [[ -z "${actual_version}" ]]; then
        echo "missing version in ${module}/package.json" >&2
        exit 1
      fi
      if [[ "${actual_version}" != "${version}" ]]; then
        echo "package.json version mismatch in ${module}: VERSION=${version} package=${actual_version}" >&2
        exit 1
      fi
      ;;
    *)
      echo "unknown module runtime: ${module}" >&2
      exit 1
      ;;
  esac
done < <(discover_modules)

if [[ "${module_count}" -eq 0 ]]; then
  echo "no modules discovered" >&2
  exit 1
fi

echo "validated ${module_count} independently versioned modules"
