#!/usr/bin/env bash
#
# Guard de arranque de sesión: verifica que opencode esté corriendo en un
# worktree de spec (p40la-ihost-spec-XXX en rama feature/SPEC-*).
#
# El checkout principal (main) es de SOLO LECTURA para agentes: todo trabajo de
# una spec vive en su worktree. Ver SPEC-066 y AGENTS.md.
#
# Uso:
#   ./scripts/check-worktree.sh      # exit 0 = ok, exit 1 = aborta
#
# Se ejecuta automáticamente al iniciar cada sesión (paso obligatorio).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || echo '')"
BRANCH="$(git branch --show-current 2>/dev/null || echo '')"

if [[ -z "${ROOT}" || -z "${BRANCH}" ]]; then
  echo "[check-worktree] ERROR: no es un repo git válido. Abrí opencode en un worktree de spec (p40la-ihost-spec-XXX)." >&2
  exit 1
fi

BASE="$(basename "${ROOT}")"
IS_SPEC_WT=""
if [[ "${BASE}" == p40la-ihost-spec-* ]]; then
  IS_SPEC_WT=1
fi
if [[ "${BRANCH}" == feature/SPEC-* ]]; then
  IS_SPEC_WT=1
fi

if [[ -n "${IS_SPEC_WT}" ]]; then
  echo "[check-worktree] OK: sesión en worktree de spec (${ROOT} / ${BRANCH})."
  exit 0
fi

echo "[check-worktree] ERROR: esta sesión NO está en un worktree de spec." >&2
echo "  Raíz: ${ROOT}" >&2
echo "  Rama: ${BRANCH}" >&2
echo "  Para trabajar en una spec, usá: ./scripts/new-worktree.sh SPEC-XXX" >&2
echo "  y abrí opencode en ../p40la-ihost-spec-XXX." >&2
echo "  El checkout principal (main) es de SOLO LECTURA para agentes (ver AGENTS.md / SPEC-066)." >&2
exit 1