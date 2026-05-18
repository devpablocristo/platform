#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

bash "${ROOT_DIR}/scripts/validate-module-versions.sh"
python3 "${ROOT_DIR}/scripts/validate-internal-ts-deps.py"
bash "${ROOT_DIR}/scripts/test-go-modules.sh"
bash "${ROOT_DIR}/scripts/test-ts-modules.sh"
