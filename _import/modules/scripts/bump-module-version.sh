#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SEMVER_REGEX='^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$'

if [[ "${#}" -ne 2 ]]; then
  echo "usage: $0 <module/runtime> <version>" >&2
  echo "example: $0 crud/ui/ts 0.2.0" >&2
  exit 1
fi

module="${1}"
version="${2}"

if [[ "${module}" = /* || "${module}" == ".." || "${module}" == *"../"* || "${module}" == *"/.."* ]]; then
  echo "invalid module path: ${module}" >&2
  exit 1
fi

module_dir="${ROOT_DIR}/${module}"

if [[ ! -d "${module_dir}" ]]; then
  echo "module not found: ${module}" >&2
  exit 1
fi

if [[ ! "${version}" =~ ${SEMVER_REGEX} ]]; then
  echo "invalid semver: ${version}" >&2
  exit 1
fi

version_file="${module_dir}/VERSION"
if [[ ! -f "${version_file}" ]]; then
  echo "missing VERSION file in ${module}" >&2
  exit 1
fi

printf "%s\n" "${version}" > "${version_file}"

if [[ -f "${module_dir}/package.json" ]]; then
  MODULE_DIR="${module_dir}" VERSION_VALUE="${version}" python3 <<'PY'
import json
import os
from pathlib import Path

module_dir = Path(os.environ["MODULE_DIR"])
version = os.environ["VERSION_VALUE"]
package_json = module_dir / "package.json"

data = json.loads(package_json.read_text())
data["version"] = version
package_json.write_text(json.dumps(data, indent=2) + "\n")
PY
fi

echo "updated ${module} to ${version}"
echo "next tag: ${module}/v${version}"
