#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

forbidden_pattern='(@devpablocristo/modules|github\.com/devpablocristo/modules|file:(\.\./)+modules|link:(\.\./)+modules|workspace:(\.\./)+modules|(^|["[:space:]:=])(\.\./)+modules([/"[:space:]]|$)|/Proyectos/[^"[:space:]]*/modules|/Projectos/[^"[:space:]]*/modules)'

violations="$(
  grep -RInE "${forbidden_pattern}" "${ROOT_DIR}" \
    --exclude-dir=.git \
    --exclude-dir=node_modules \
    --exclude=validate-boundaries.sh || true
)"

if [[ -n "${violations}" ]]; then
  echo "forbidden dependency direction: core must not reference modules" >&2
  echo "${violations}" >&2
  exit 1
fi

echo "core boundaries validated"
