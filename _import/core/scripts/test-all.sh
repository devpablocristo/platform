#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

bash "${ROOT_DIR}/scripts/validate-runtime-layout.sh"
bash "${ROOT_DIR}/scripts/validate-boundaries.sh"
bash "${ROOT_DIR}/scripts/validate-module-versions.sh"
bash "${ROOT_DIR}/scripts/test-go-modules.sh"
bash "${ROOT_DIR}/scripts/test-rust-modules.sh"
bash "${ROOT_DIR}/scripts/test-ts-modules.sh"
bash "${ROOT_DIR}/scripts/test-http-python.sh"
bash "${ROOT_DIR}/scripts/test-ai.sh"
