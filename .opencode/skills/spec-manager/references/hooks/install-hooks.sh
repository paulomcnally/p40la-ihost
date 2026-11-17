#!/usr/bin/env bash
#
# install-hooks.sh — Instala los hooks versionados de .githooks (SPEC-136).
# Copiado por spec-manager init. Idempotente.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [ -z "${ROOT}" ]; then
  echo "ERROR: no es un repositorio git." >&2
  exit 1
fi

cd "${ROOT}"

if [ ! -d ".githooks" ]; then
  echo "ERROR: no existe .githooks/ en ${ROOT}" >&2
  exit 1
fi

chmod +x .githooks/pre-commit .githooks/commit-msg 2>/dev/null || true

git config core.hooksPath .githooks

current="$(git config --get core.hooksPath || true)"
if [ "${current}" != ".githooks" ]; then
  echo "ERROR: no se pudo configurar core.hooksPath (actual: ${current:-vacío})." >&2
  exit 1
fi

echo "[install-hooks] OK: core.hooksPath=.githooks"
echo "  Hooks activos: pre-commit, commit-msg"
