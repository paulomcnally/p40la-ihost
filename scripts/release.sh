#!/usr/bin/env bash
#
# Script de release automático para p40la-ihost en Docker Hub.
#
# Libera la app web y, si corresponde, la imagen MCP en el mismo ciclo.
# La app se publica por el workflow "Build & Push Docker Image" (tag vX.Y.Z).
# El MCP se publica por "Build & Push MCP Image" (tag mcp-vX.Y.Z) cuando hay
# cambios que afectan al binario MCP desde el último tag mcp-v*.
#
# Uso:
#   ./scripts/release.sh                       # app: bump patch; MCP si hay cambios
#   ./scripts/release.sh 0.5.20                # app 0.5.20; MCP automático
#   ./scripts/release.sh 0.5.20 0.6.0          # app y MCP explícitos
#   MCP_VERSION=0.6.0 ./scripts/release.sh     # MCP explícito por entorno
#   ./scripts/release.sh --no-mcp              # solo app
#   MCP_VERSION=skip ./scripts/release.sh      # solo app
#   ./scripts/release.sh --check               # plan sin escrituras
#
# Requisitos: bash, curl, git, gh
# Opcionales: jq o python3 (para parsear JSON de Docker Hub)
#

set -euo pipefail

readonly REPO="paulomcnally/p40la-ihost"
readonly MCP_REPO="paulomcnally/p40la-mcp"
readonly DOCKER_HUB_API="https://hub.docker.com/v2/repositories/${REPO}/tags/?page_size=50"
readonly WORKFLOW_NAME="Build & Push Docker Image"
readonly WORKFLOW_FILE="docker-publish.yml"
readonly MCP_WORKFLOW_NAME="Build & Push MCP Image"
readonly MCP_WORKFLOW_FILE="mcp-publish.yml"
readonly MCP_WORKFLOW_PATH=".github/workflows/mcp-publish.yml"
readonly GITHUB_REPO="paulomcnally/p40la-ihost"
readonly MCP_PATHS=(
  "cmd/mcp-server"
  "internal/mcp"
  "Dockerfile.mcp"
  "deploy/mcp"
  ".github/workflows/mcp-publish.yml"
  "go.mod"
  "go.sum"
)

# -----------------------------------------------------------------------------
# Helpers
# -----------------------------------------------------------------------------

log() {
  printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*" >&2
}

error() {
  printf '[%s] ERROR: %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*" >&2
  exit 1
}

usage() {
  cat <<'EOF'
Uso:
  ./scripts/release.sh                       # app: bump patch; MCP si hay cambios
  ./scripts/release.sh 0.5.20                # app 0.5.20; MCP automático
  ./scripts/release.sh 0.5.20 0.6.0          # app y MCP explícitos
  MCP_VERSION=0.6.0 ./scripts/release.sh     # MCP explícito por entorno
  ./scripts/release.sh --no-mcp              # solo app
  MCP_VERSION=skip ./scripts/release.sh      # solo app
  ./scripts/release.sh --check               # plan sin escrituras
EOF
}

# Abre una URL en el navegador del sistema.
open_url() {
  local url="$1"
  local os

  os="$(uname -s)"
  case "${os}" in
    Linux)
      if command -v google-chrome >/dev/null 2>&1; then
        google-chrome "${url}" >/dev/null 2>&1 &
      elif command -v xdg-open >/dev/null 2>&1; then
        xdg-open "${url}" >/dev/null 2>&1 &
      else
        log "No se pudo abrir el navegador automáticamente. URL: ${url}"
      fi
      ;;
    Darwin)
      open "${url}"
      ;;
    CYGWIN* | MINGW* | MSYS*)
      cmd.exe /c start "" "${url}" >/dev/null 2>&1 || log "No se pudo abrir el navegador automáticamente. URL: ${url}"
      ;;
    *)
      log "Sistema operativo no soportado para abrir navegador. URL: ${url}"
      ;;
  esac
}

# Extrae la última versión semver pura (X.Y.Z) desde la respuesta JSON de Docker Hub.
parse_latest_version() {
  local response="$1"
  local versions

  if command -v jq >/dev/null 2>&1; then
    versions=$(echo "$response" | jq -r '.results[].name')
  elif command -v python3 >/dev/null 2>&1; then
    versions=$(echo "$response" | python3 -c '
import json, sys
data = json.load(sys.stdin)
for r in data.get("results", []):
    print(r.get("name", ""))
')
  else
    versions=$(echo "$response" | grep -o '"name":"[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*"' | grep -o '[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*')
  fi

  echo "$versions" | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -1 || true
}

# Indica si una versión ya existe en el JSON de Docker Hub.
version_exists_in_docker_hub() {
  local response="$1"
  local version="$2"

  if command -v jq >/dev/null 2>&1; then
    echo "$response" | jq -e --arg v "$version" '.results[] | select(.name == $v)' >/dev/null 2>&1
  elif command -v python3 >/dev/null 2>&1; then
    echo "$response" | python3 -c "
import json, sys
data = json.load(sys.stdin)
for r in data.get('results', []):
    if r.get('name') == '${version}':
        sys.exit(0)
sys.exit(1)
" >/dev/null 2>&1
  else
    echo "$response" | grep -q "\"name\":\"${version}\""
  fi
}

# Devuelve la última versión MCP publicada (sin el prefijo mcp-v), o vacío.
latest_mcp_version() {
  git tag --list 'mcp-v*' \
    | sed 's/^mcp-v//' \
    | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$' \
    | sort -V \
    | tail -1 || true
}

# Bumpea el patch de una versión semver X.Y.Z.
bump_patch_version() {
  local version="$1"
  local major minor patch
  IFS='.' read -r major minor patch <<< "${version}"
  echo "${major}.${minor}.$((patch + 1))"
}

# Espera el workflow disparado por un tag y devuelve su URL (o vacío).
wait_for_workflow_run() {
  local workflow="$1"
  local ref="$2"
  local url=""
  local attempt=0
  local max_attempts=20

  while [[ -z "${url}" && "${attempt}" -lt "${max_attempts}" ]]; do
    attempt=$((attempt + 1))
    if command -v jq >/dev/null 2>&1; then
      url=$(gh run list \
        --workflow="${workflow}" \
        --event push \
        --json url,headBranch \
        --limit 30 2>/dev/null \
        | jq -r --arg ref "${ref}" '[.[] | select(.headBranch == $ref)][0].url // empty' || true)
    else
      url=$(gh run list \
        --workflow="${workflow}" \
        --event push \
        --limit 30 2>/dev/null \
        | grep -o "https://github.com/${GITHUB_REPO}/actions/runs/[0-9]*" | head -1 || true)
    fi

    if [[ -z "${url}" ]]; then
      if [[ "${attempt}" -lt "${max_attempts}" ]]; then
        log "Intento ${attempt}/${max_attempts}: el workflow '${workflow}' todavía no aparece, reintentando en 10 segundos..."
        sleep 10
      fi
    else
      log "Workflow '${workflow}' encontrado en el intento ${attempt}/${max_attempts}"
    fi
  done

  printf '%s' "${url}"
}

# Extrae el ID del run desde la URL de GitHub Actions.
workflow_run_id() {
  local url="$1"
  printf '%s' "${url##*/}"
}

# Espera a que un run termine y falla si la conclusión no es success.
wait_for_workflow_conclusion() {
  local url="$1"
  local label="$2"
  local run_id
  local attempt=0
  local max_attempts=80
  local status=""
  local conclusion=""

  run_id=$(workflow_run_id "${url}")
  if ! [[ "${run_id}" =~ ^[0-9]+$ ]]; then
    log "No se pudo obtener el run ID de ${label}; revisá manualmente: ${url}"
    return 0
  fi

  while [[ "${attempt}" -lt "${max_attempts}" ]]; do
    attempt=$((attempt + 1))
    status=$(gh run view "${run_id}" --json status --jq '.status' 2>/dev/null || true)
    conclusion=$(gh run view "${run_id}" --json conclusion --jq '.conclusion' 2>/dev/null || true)

    if [[ "${status}" == "completed" ]]; then
      if [[ "${conclusion}" == "success" ]]; then
        log "${label}: workflow completado con éxito."
        return 0
      fi
      log "ERROR: ${label} terminó con conclusión '${conclusion}'. URL: ${url}"
      return 1
    fi

    log "${label}: estado ${status:-desconocido}; reintentando en 15 segundos (${attempt}/${max_attempts})..."
    sleep 15
  done

  log "ERROR: ${label} no terminó en el tiempo esperado. URL: ${url}"
  return 1
}

# -----------------------------------------------------------------------------
# Argumentos
# -----------------------------------------------------------------------------

CHECK_ONLY=0
NO_MCP=0
APP_VERSION_REQUESTED=""
MCP_VERSION_ARG=""
MCP_VERSION_REQUESTED="${MCP_VERSION:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --check)
      CHECK_ONLY=1
      ;;
    --no-mcp)
      NO_MCP=1
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    -*)
      error "Opción desconocida: $1 (usá --help)"
      ;;
    *)
      if [[ -z "${APP_VERSION_REQUESTED}" ]]; then
        APP_VERSION_REQUESTED="$1"
      elif [[ -z "${MCP_VERSION_ARG}" ]]; then
        MCP_VERSION_ARG="$1"
      else
        error "Demasiados argumentos. Uso: ./scripts/release.sh [app-version] [mcp-version]"
      fi
      ;;
  esac
  shift
done

if [[ -n "${MCP_VERSION_ARG}" ]]; then
  MCP_VERSION_REQUESTED="${MCP_VERSION_ARG}"
fi

case "${MCP_VERSION_REQUESTED}" in
  skip | none | no)
    NO_MCP=1
    MCP_VERSION_REQUESTED=""
    ;;
esac

# -----------------------------------------------------------------------------
# Validaciones de entorno
# -----------------------------------------------------------------------------

log "Validando herramientas necesarias..."
command -v curl >/dev/null 2>&1 || error "curl no está instalado"
command -v git >/dev/null 2>&1 || error "git no está instalado"
command -v gh >/dev/null 2>&1 || error "gh no está instalado. Instalalo desde: https://cli.github.com/"

log "Validando autenticación con gh..."
gh auth status >/dev/null 2>&1 || error "gh no está autenticado. Ejecutá: gh auth login"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${ROOT_DIR}"

CURRENT_BRANCH=$(git branch --show-current)
if [[ "${CURRENT_BRANCH}" != "main" ]]; then
  if [[ "${CHECK_ONLY}" -eq 1 ]]; then
    log "Aviso: estás en ${CURRENT_BRANCH}; --check no escribe y el plan no es ejecutable hasta mergear a main."
  else
    error "No estás en la rama main. Rama actual: ${CURRENT_BRANCH}"
  fi
fi

log "Validando que no haya cambios sin commitear..."
if ! git diff --quiet || ! git diff --cached --quiet; then
  error "El working tree tiene cambios sin commitear. Commitea o stash antes de continuar."
fi

# -----------------------------------------------------------------------------
# Validar que no haya ramas de spec con código sin mergear a main
# -----------------------------------------------------------------------------

if [[ "${CHECK_ONLY}" -eq 1 ]]; then
  log "Aviso: --check omite la validación de ramas sin mergear."
else
  log "Validando que no haya ramas de feature con código sin mergear a main..."
  UNMERGED_BRANCHES=$(git branch --no-merged main --format='%(refname:short)' | grep -v '^main$' || true)
  if [[ -n "${UNMERGED_BRANCHES}" ]]; then
    error "Hay ramas de feature con código que main NO incluye:
$(echo "${UNMERGED_BRANCHES}" | sed 's/^/  - /')

La imagen se construye desde el tag sobre main, por lo que ese código NO se incluiría.
Mergeá cada rama a main (o liberá su spec) ANTES de correr release.sh."
  fi
fi

# -----------------------------------------------------------------------------
# Sincronizar tags
# -----------------------------------------------------------------------------

log "Sincronizando tags con origin..."
git fetch --tags origin >/dev/null 2>&1 || error "No se pudo hacer fetch de tags desde origin"

# -----------------------------------------------------------------------------
# Determinar si el MCP aplica
# -----------------------------------------------------------------------------

MCP_LAST_VERSION="$(latest_mcp_version)"
MCP_CHANGED=""
if [[ -n "${MCP_LAST_VERSION}" ]]; then
  MCP_CHANGED=$(git diff --name-only "mcp-v${MCP_LAST_VERSION}"..HEAD -- "${MCP_PATHS[@]}" || true)
else
  MCP_CHANGED="sin-tag-previo"
fi

MCP_RELEASE=0
MCP_VERSION=""
if [[ "${NO_MCP}" -eq 1 ]]; then
  log "MCP: omitido por --no-mcp/MCP_VERSION=skip."
elif [[ -n "${MCP_VERSION_REQUESTED}" ]]; then
  MCP_VERSION="${MCP_VERSION_REQUESTED}"
  MCP_RELEASE=1
  if [[ -z "${MCP_CHANGED}" ]]; then
    log "MCP: no se detectaron cambios desde ${MCP_LAST_VERSION:+mcp-v${MCP_LAST_VERSION}}, pero se publica por versión explícita."
  fi
elif [[ -n "${MCP_CHANGED}" ]]; then
  MCP_RELEASE=1
  if [[ -n "${MCP_LAST_VERSION}" ]]; then
    MCP_VERSION="$(bump_patch_version "${MCP_LAST_VERSION}")"
  else
    MCP_VERSION="0.1.0"
  fi
else
  log "MCP: sin cambios desde mcp-v${MCP_LAST_VERSION:-el inicio}; no se publica."
fi

if [[ "${MCP_RELEASE}" -eq 1 ]]; then
  if ! [[ "${MCP_VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    error "Versión MCP inválida: ${MCP_VERSION}. Debe seguir semver (X.Y.Z)."
  fi
  [[ -f "${MCP_WORKFLOW_PATH}" ]] || error "Falta ${MCP_WORKFLOW_PATH}; no se puede publicar el MCP."
  if git tag -l "mcp-v${MCP_VERSION}" | grep -q "mcp-v${MCP_VERSION}"; then
    error "El tag mcp-v${MCP_VERSION} ya existe localmente"
  fi
  if git ls-remote --tags origin "refs/tags/mcp-v${MCP_VERSION}" | grep -q "refs/tags/mcp-v${MCP_VERSION}"; then
    error "El tag mcp-v${MCP_VERSION} ya existe en origin"
  fi
fi

# -----------------------------------------------------------------------------
# Consultar Docker Hub y calcular la versión de la app
# -----------------------------------------------------------------------------

log "Consultando tags en Docker Hub: ${REPO}..."
RESPONSE=$(curl -fsSL "${DOCKER_HUB_API}" 2>/dev/null) || error "No se pudo consultar Docker Hub API en ${DOCKER_HUB_API}"

LATEST_VERSION=$(parse_latest_version "${RESPONSE}")

if [[ -n "${LATEST_VERSION}" ]]; then
  log "Última versión publicada en Docker Hub: ${LATEST_VERSION}"
else
  log "No hay versiones publicadas en Docker Hub (primera release)."
fi

if [[ -n "${APP_VERSION_REQUESTED}" ]]; then
  NEW_VERSION="${APP_VERSION_REQUESTED}"
  if ! [[ "${NEW_VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    error "Versión inválida: ${NEW_VERSION}. Debe seguir semver (X.Y.Z)."
  fi
  log "Usando versión manual: ${NEW_VERSION}"
elif [[ -n "${LATEST_VERSION}" ]]; then
  NEW_VERSION="$(bump_patch_version "${LATEST_VERSION}")"
  log "Calculando bump automático de patch: ${LATEST_VERSION} → ${NEW_VERSION}"
else
  INITIAL_VERSION=$(jq -r '.version' frontend/package.json 2>/dev/null || echo "0.1.0")
  NEW_VERSION="${INITIAL_VERSION}"
  log "Primera release: usando versión inicial ${NEW_VERSION}"
fi

# -----------------------------------------------------------------------------
# Validar que la versión de la app no exista
# -----------------------------------------------------------------------------

log "Verificando que v${NEW_VERSION} no exista como tag git..."
if git tag -l "v${NEW_VERSION}" | grep -q "v${NEW_VERSION}"; then
  error "El tag v${NEW_VERSION} ya existe localmente"
fi

if git ls-remote --tags origin "refs/tags/v${NEW_VERSION}" | grep -q "refs/tags/v${NEW_VERSION}"; then
  error "El tag v${NEW_VERSION} ya existe en origin"
fi

log "Verificando que ${NEW_VERSION} no exista en Docker Hub..."
if version_exists_in_docker_hub "${RESPONSE}" "${NEW_VERSION}"; then
  error "La versión ${NEW_VERSION} ya existe en Docker Hub"
fi

# -----------------------------------------------------------------------------
# Plan (--check)
# -----------------------------------------------------------------------------

if [[ "${CHECK_ONLY}" -eq 1 ]]; then
  echo ""
  log "=============================================="
  log "Plan de release (--check): no se escribe nada"
  log "App: v${NEW_VERSION}"
  if [[ "${MCP_RELEASE}" -eq 1 ]]; then
    if [[ -n "${MCP_LAST_VERSION}" ]]; then
      log "MCP: mcp-v${MCP_VERSION} (último tag: mcp-v${MCP_LAST_VERSION})"
    else
      log "MCP: mcp-v${MCP_VERSION} (sin tag previo)"
    fi
    if [[ -n "${MCP_CHANGED}" && "${MCP_CHANGED}" != "sin-tag-previo" ]]; then
      log "Cambios MCP detectados:"
      echo "${MCP_CHANGED}" | sed 's/^/  - /'
    elif [[ "${MCP_CHANGED}" == "sin-tag-previo" ]]; then
      log "MCP: sin tag previo, se publica la primera versión."
    fi
  else
    log "MCP: se omite."
  fi
  log "Archivos que se actualizarían: docker-compose.yml, frontend/package.json"
  TAGS_PLAN="v${NEW_VERSION}"
  if [[ "${MCP_RELEASE}" -eq 1 ]]; then
    TAGS_PLAN="${TAGS_PLAN}, mcp-v${MCP_VERSION}"
  fi
  log "Tags que se crearían: ${TAGS_PLAN}"
  log "=============================================="
  echo ""
  exit 0
fi

# -----------------------------------------------------------------------------
# Actualizar archivos
# -----------------------------------------------------------------------------

log "Actualizando docker-compose.yml (VERSION=${NEW_VERSION})..."
sed -i.bak "s/\(- VERSION=\).*/\1${NEW_VERSION}/" docker-compose.yml
rm -f docker-compose.yml.bak

log "Actualizando frontend/package.json (version=${NEW_VERSION})..."
if command -v jq >/dev/null 2>&1; then
  TMP_PKG=$(mktemp)
  jq --arg v "${NEW_VERSION}" '.version = $v' frontend/package.json > "${TMP_PKG}"
  mv "${TMP_PKG}" frontend/package.json
else
  sed -i.bak "s/\"version\": \"[^\"]*\"/\"version\": \"${NEW_VERSION}\"/" frontend/package.json
  rm -f frontend/package.json.bak
fi

if git diff --quiet; then
  error "No se detectaron cambios después de actualizar la versión"
fi

# -----------------------------------------------------------------------------
# Commit y tags
# -----------------------------------------------------------------------------

log "Creando commit de bump..."
git add docker-compose.yml frontend/package.json
git commit -m "bump version to ${NEW_VERSION}"

log "Creando tag anotado v${NEW_VERSION}..."
git tag -a "v${NEW_VERSION}" -m "Release v${NEW_VERSION}"

if [[ "${MCP_RELEASE}" -eq 1 ]]; then
  log "Creando tag anotado mcp-v${MCP_VERSION}..."
  git tag -a "mcp-v${MCP_VERSION}" -m "Release MCP v${MCP_VERSION}"
fi

# -----------------------------------------------------------------------------
# Push
# -----------------------------------------------------------------------------

log "Pusheando commit a origin/main..."
git push origin main

log "Pusheando tag v${NEW_VERSION}..."
git push origin "v${NEW_VERSION}"

if [[ "${MCP_RELEASE}" -eq 1 ]]; then
  log "Pusheando tag mcp-v${MCP_VERSION}..."
  git push origin "mcp-v${MCP_VERSION}"
fi

# -----------------------------------------------------------------------------
# Esperar los workflows
# -----------------------------------------------------------------------------

log "Esperando que GitHub Actions registre el workflow de la app..."
APP_RUN_URL=$(wait_for_workflow_run "${WORKFLOW_FILE}" "v${NEW_VERSION}" || true)

MCP_RUN_URL=""
if [[ "${MCP_RELEASE}" -eq 1 ]]; then
  log "Esperando que GitHub Actions registre el workflow del MCP..."
  MCP_RUN_URL=$(wait_for_workflow_run "${MCP_WORKFLOW_FILE}" "mcp-v${MCP_VERSION}" || true)
fi

WORKFLOW_FAILED=0
if [[ -z "${APP_RUN_URL}" || "${APP_RUN_URL}" == "null" ]]; then
  log "ERROR: no se encontró el run del workflow de la app tras los reintentos."
  WORKFLOW_FAILED=1
else
  wait_for_workflow_conclusion "${APP_RUN_URL}" "App v${NEW_VERSION}" || WORKFLOW_FAILED=1
fi
if [[ "${MCP_RELEASE}" -eq 1 ]]; then
  if [[ -z "${MCP_RUN_URL}" || "${MCP_RUN_URL}" == "null" ]]; then
    log "ERROR: no se encontró el run del workflow del MCP tras los reintentos."
    WORKFLOW_FAILED=1
  else
    wait_for_workflow_conclusion "${MCP_RUN_URL}" "MCP v${MCP_VERSION}" || WORKFLOW_FAILED=1
  fi
fi

# -----------------------------------------------------------------------------
# Resumen
# -----------------------------------------------------------------------------

echo ""
log "=============================================="
log "Release v${NEW_VERSION} procesado"
log "App Docker Hub: https://hub.docker.com/r/${REPO}/tags"
if [[ -n "${APP_RUN_URL}" && "${APP_RUN_URL}" != "null" ]]; then
  log "App workflow: ${APP_RUN_URL}"
else
  log "App workflow: https://github.com/${GITHUB_REPO}/actions"
  log "(No se pudo obtener la URL exacta; revisá: gh run list --workflow=\"${WORKFLOW_NAME}\")"
fi

if [[ "${MCP_RELEASE}" -eq 1 ]]; then
  log "MCP mcp-v${MCP_VERSION}: https://hub.docker.com/r/${MCP_REPO}/tags"
  if [[ -n "${MCP_RUN_URL}" && "${MCP_RUN_URL}" != "null" ]]; then
    log "MCP workflow: ${MCP_RUN_URL}"
  else
    log "MCP workflow: https://github.com/${GITHUB_REPO}/actions"
    log "(No se pudo obtener la URL exacta; revisá: gh run list --workflow=\"${MCP_WORKFLOW_NAME}\")"
  fi
fi
log "=============================================="
echo ""

if [[ "${WORKFLOW_FAILED}" -eq 1 ]]; then
  error "Uno o más workflows fallaron; revisá las URLs anteriores y reintentá el workflow desde GitHub Actions."
fi

if [[ -n "${APP_RUN_URL}" && "${APP_RUN_URL}" != "null" ]]; then
  open_url "${APP_RUN_URL}"
elif [[ -n "${MCP_RUN_URL}" && "${MCP_RUN_URL}" != "null" ]]; then
  open_url "${MCP_RUN_URL}"
fi
