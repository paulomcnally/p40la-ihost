#!/usr/bin/env bash
#
# Crea un git worktree aislado para una spec.
#
# Cada sesión/ventana de opencode DEBE trabajar en su propio worktree
# para no pisar el trabajo sin commitear de otras sesiones.
#
# Uso:
#   ./scripts/new-worktree.sh <SPEC-ID>          # ej: SPEC-034
#   ./scripts/new-worktree.sh <SPEC-ID> <nombre> # nombre opcional de la rama
#
# Requisitos: bash, git
#
# Ejemplo:
#   ./scripts/new-worktree.sh SPEC-034
#   # crea: ../p40la-ihost-spec-034 en rama feature/SPEC-034
#   # luego: cd ../p40la-ihost-spec-034
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

log() {
  printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"
}

error() {
  printf '[%s] ERROR: %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*" >&2
  exit 1
}

# -----------------------------------------------------------------------------
# Validación de argumentos
# -----------------------------------------------------------------------------

if [[ $# -lt 1 ]]; then
  error "Uso: $0 <SPEC-ID> [nombre-rama]"
fi

SPEC_ID="$1"
if ! [[ "${SPEC_ID}" =~ ^SPEC-[0-9]{3}$ ]]; then
  error "SPEC-ID inválido: ${SPEC_ID}. Debe seguir el formato SPEC-XXX (ej: SPEC-034)."
fi

BRANCH_NAME="${2:-feature/${SPEC_ID}}"
WORKTREE_PATH="${ROOT_DIR}-${SPEC_ID,,}"  # ej: /ruta/p40la-ihost-spec-034

# -----------------------------------------------------------------------------
# Validaciones de estado
# -----------------------------------------------------------------------------

command -v git >/dev/null 2>&1 || error "git no está instalado"

if [[ -d "${WORKTREE_PATH}" ]]; then
  error "El directorio ya existe: ${WORKTREE_PATH}"
fi

if git worktree list | grep -q "${WORKTREE_PATH}"; then
  error "Ya existe un worktree en: ${WORKTREE_PATH}"
fi

if git branch --list "${BRANCH_NAME}" | grep -q "${BRANCH_NAME}"; then
  error "La rama ${BRANCH_NAME} ya existe. Elegí otro nombre: $0 ${SPEC_ID} feature/${SPEC_ID}-fix"
fi

# -----------------------------------------------------------------------------
# Validar que el checkout principal (main) esté limpio
# -----------------------------------------------------------------------------
# Evita la colisión multi-sesión: si `main` tiene cambios sin commitear (de esta
# u otra sesión), no se crea el worktree. Ver SPEC-066 y AGENTS.md.

MAIN_WT="$(git worktree list --porcelain 2>/dev/null | head -1 | sed 's/^worktree //')"
if [[ -n "${MAIN_WT}" && -n "$(git -C "${MAIN_WT}" status --porcelain 2>/dev/null)" ]]; then
  error "El checkout principal (${MAIN_WT}) tiene cambios sin commitear. Commiteá o revertí esos cambios antes de crear un worktree (ver AGENTS.md / SPEC-066)."
fi

# -----------------------------------------------------------------------------
# Sincronizar main con origin
# -----------------------------------------------------------------------------

log "Sincronizando main con origin/main..."
git fetch origin main >/dev/null 2>&1 || log "warning: no se pudo hacer fetch de origin/main"

BASE_REF="main"
if git rev-parse --verify origin/main >/dev/null 2>&1; then
  BASE_REF="origin/main"
fi
log "Usando base: ${BASE_REF}"

# -----------------------------------------------------------------------------
# Crear el worktree
# -----------------------------------------------------------------------------

log "Creando worktree: ${WORKTREE_PATH}"
log "Rama: ${BRANCH_NAME} (desde ${BASE_REF})"

git worktree add -b "${BRANCH_NAME}" "${WORKTREE_PATH}" "${BASE_REF}"

# -----------------------------------------------------------------------------
# Resumen
# -----------------------------------------------------------------------------

echo ""
log "=============================================="
log "Worktree creado correctamente."
log "Para trabajar en esta spec, abrí una NUEVA ventana de opencode en:"
log "  cd ${WORKTREE_PATH}"
log ""
log "Reglas críticas:"
log "  - NUNCA ejecutes git checkout/reset/switch en otro worktree ni en el principal compartido."
log "  - Todo commit/push de esta spec se hace DENTRO de ${WORKTREE_PATH}."
log "  - Para ver todos los worktrees: git worktree list (desde ${ROOT_DIR})"
log "=============================================="
echo ""
