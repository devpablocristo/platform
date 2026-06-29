#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

forbidden_pattern='(@devpablocristo/(core|modules)|github\.com/devpablocristo/(core|modules)|file:(\.\./)+(core|modules)|link:(\.\./)+(core|modules)|workspace:(\.\./)+(core|modules)|(^|["[:space:]:=])(\.\./)+(core|modules)([/"[:space:]]|$)|/Proyectos/[^"[:space:]]*/(core|modules)|/Projectos/[^"[:space:]]*/(core|modules))'

violations="$(
  grep -RInE "${forbidden_pattern}" "${ROOT_DIR}" \
    --exclude-dir=.git \
    --exclude-dir=node_modules \
    --exclude-dir=docs \
    --exclude-dir=tooling \
    --include='*.go' \
    --include='go.mod' \
    --include='*.ts' \
    --include='*.tsx' \
    --include='*.css' \
    --include='*.js' \
    --include='*.json' \
    --include='*.md' \
    --include='Dockerfile*' \
    --include='Cargo.toml' \
    --include='pyproject.toml' \
    --include='package.json' \
    --exclude=AUDIT.md || true
)"

if [[ -n "${violations}" ]]; then
  echo "forbidden legacy dependency: platform code must not reference core/modules" >&2
  echo "${violations}" >&2
  exit 1
fi

echo "platform boundaries validated"
