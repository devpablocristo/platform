#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
AI_DIR="${ROOT_DIR}/ai/python"
HTTP_DIR="${ROOT_DIR}/http/python"
VENV_DIR="${AI_DIR}/.venv"

if [[ ! -x "${VENV_DIR}/bin/pip" ]] || ! "${VENV_DIR}/bin/python" -V >/dev/null 2>&1 || ! "${VENV_DIR}/bin/pip" --version >/dev/null 2>&1; then
  rm -rf "${VENV_DIR}"
  python3 -m venv "${VENV_DIR}"
fi

"${VENV_DIR}/bin/pip" install --upgrade pip >/dev/null
"${VENV_DIR}/bin/pip" install -e "${HTTP_DIR}" >/dev/null
"${VENV_DIR}/bin/pip" install -e "${AI_DIR}[test]" >/dev/null
PYTHONPYCACHEPREFIX="${AI_DIR}/.pycache" "${VENV_DIR}/bin/python" -m compileall "${AI_DIR}/src"
(
  cd "${AI_DIR}"
  PYTHONPATH=src "${VENV_DIR}/bin/python" -m pytest tests -q
)
