#!/usr/bin/env bash
#
# Script de release automático para p40la-ihost (app), MCP y wrapper Tailscale.
#
# Cada artefacto se versiona y publica de forma INDEPENDIENTE: se compara el
# contenido relevante contra el último tag de ese artefacto y solo se crea tag
# y se dispara el workflow si hay cambios (o si se fuerza una versión).
#
#   App        -> tag vX.Y.Z          -> workflow docker-publish.yml
#   MCP        -> tag mcp-vX.Y.Z      -> workflow mcp-publish.yml
#   Tailscale  -> tag tailscale-vX.Y.Z -> workflow tailscale-publish.yml
#
# Uso:
#   ./scripts/release.sh                          # libera solo lo que cambió (bump patch)
#   ./scripts/release.sh 0.5.22                   # fuerza app 0.5.22; MCP/tailscale automáticos
#   ./scripts/release.sh 0.5.22 0.7.0             # fuerza app + MCP
#   ./scripts/release.sh 0.5.22 0.7.0 0.2.0       # fuerza app + MCP + wrapper Tailscale
#   MCP_VERSION=0.7.0 ./scripts/release.sh        # MCP explícito por entorno
#   TAILSCALE_VERSION=0.2.0 ./scripts/release.sh  # wrapper explícito por entorno
#   ./scripts/release.sh --no-mcp                 # omite el MCP
#   ./scripts/release.sh --no-tailscale           # omite el wrapper Tailscale
#   ./scripts/release.sh --check                  # plan read-only (sin commits, tags, push ni fetch)
#   ./scripts/release.sh --list-tags              # último tag y cambios pendientes por artefacto
#
# Requisitos: bash, git. En release real además: gh autenticado y curl.
# Opcionales: jq o python3 (para parsear JSON de Docker Hub y de gh).
#

set -euo pipefail

readonly REPO="paulomcnally/p40la-ihost"
readonly MCP_REPO="paulomcnally/p40la-mcp"
readonly TAILSCALE_REPO="paulomcnally/p40la-mcp-tailscale"
readonly DOCKER_HUB_API="https://hub.docker.com/v2/repositories/${REPO}/tags/?page_size=50"
readonly WORKFLOW_NAME="Build & Push Docker Image"
readonly WORKFLOW_FILE="docker-publish.yml"
readonly MCP_WORKFLOW_NAME="Build & Push MCP Image"
readonly MCP_WORKFLOW_FILE="mcp-publish.yml"
readonly MCP_WORKFLOW_PATH=".github/workflows/mcp-publish.yml"
readonly TAILSCALE_WORKFLOW_NAME="Build & Push Tailscale Image"
readonly TAILSCALE_WORKFLOW_FILE="tailscale-publish.yml"
readonly TAILSCALE_WORKFLOW_PATH=".github/workflows/tailscale-publish.yml"
readonly GITHUB_REPO="paulomcnally/p40la-ihost"

# Allowlist de la app: contenido real de la imagen `p40la-ihost`.
# `internal/mcp/` se excluye aparte (ver APP_EXCLUDE): el binario de la app no
# importa el paquete MCP, son grafos de build separados (ADR-002).
readonly APP_PATHS=(
  "cmd/server"
  "internal"
  "frontend"
  "public"
  "migrations"
  "go.mod"
  "go.sum"
  "Dockerfile"
  ".github/workflows/docker-publish.yml"
)
readonly APP_EXCLUDE='^internal/mcp/'

# Allowlist del MCP: contenido real de la imagen `p40la-mcp`.
readonly MCP_PATHS=(
  "cmd/mcp-server"
  "internal/mcp"
  "Dockerfile.mcp"
  ".github/workflows/mcp-publish.yml"
  "go.mod"
  "go.sum"
)

# Allowlist del wrapper Tailscale: contexto de build exclusivo.
readonly TAILSCALE_PATHS=(
  "deploy/mcp/tailscale"
)

# -----------------------------------------------------------------------------
# Helpers
# -----------------------------------------------------------------------------

log() {
  printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*" >&2
}

warn() {
  printf '[%s] AVISO: %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*" >&2
}

error() {
  printf '[%s] ERROR: %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*" >&2
  exit 1
}

usage() {
  cat <<'EOF'
Uso:
  ./scripts/release.sh                          # libera solo lo que cambió (bump patch)
  ./scripts/release.sh 0.5.22                   # fuerza app 0.5.22; MCP/tailscale automáticos
  ./scripts/release.sh 0.5.22 0.7.0             # fuerza app + MCP
  ./scripts/release.sh 0.5.22 0.7.0 0.2.0       # fuerza app + MCP + wrapper Tailscale
  MCP_VERSION=0.7.0 ./scripts/release.sh        # MCP explícito por entorno
  TAILSCALE_VERSION=0.2.0 ./scripts/release.sh  # wrapper explícito por entorno
  APP_VERSION=0.5.22 ./scripts/release.sh       # app explícita por entorno
  ./scripts/release.sh --no-mcp                 # omite el MCP (o MCP_VERSION=skip)
  ./scripts/release.sh --no-tailscale           # omite el wrapper (o TAILSCALE_VERSION=skip)
  ./scripts/release.sh --check                  # plan read-only: no escribe, no hace fetch
  ./scripts/release.sh --list-tags              # último tag y cambios pendientes por artefacto

Cada artefacto tiene su propio tag y su propio workflow:
  app       -> vX.Y.Z           -> docker-publish.yml
  MCP       -> mcp-vX.Y.Z       -> mcp-publish.yml
  Tailscale -> tailscale-vX.Y.Z -> tailscale-publish.yml
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

# Bumpea el patch de una versión semver X.Y.Z.
bump_patch_version() {
  local version="$1"
  local major minor patch
  IFS='.' read -r major minor patch <<< "${version}"
  echo "${major}.${minor}.$((patch + 1))"
}

# Versión declarada en frontend/package.json (fallback de primera release de la app).
package_json_version() {
  local version=""

  if [[ -f frontend/package.json ]]; then
    if command -v jq >/dev/null 2>&1; then
      version=$(jq -r '.version // empty' frontend/package.json 2>/dev/null || true)
    elif command -v python3 >/dev/null 2>&1; then
      version=$(python3 -c '
import json
with open("frontend/package.json") as fh:
    print(json.load(fh).get("version", ""))
' 2>/dev/null || true)
    fi
    if [[ -z "${version}" ]]; then
      version=$(grep -o '"version"[[:space:]]*:[[:space:]]*"[^"]*"' frontend/package.json 2>/dev/null \
        | head -1 | sed 's/.*"\([^"]*\)"$/\1/' || true)
    fi
  fi

  [[ "${version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || version="0.1.0"
  printf '%s' "${version}"
}

# Nombre del tag de un artefacto a partir de su prefijo y versión.
artifact_tag_name() {
  local prefix="$1"
  local version="$2"
  printf '%sv%s' "${prefix}" "${version}"
}

# Etiqueta legible del último tag de un artefacto.
artifact_last_tag_label() {
  local prefix="$1"
  local version="$2"
  if [[ -n "${version}" ]]; then
    artifact_tag_name "${prefix}" "${version}"
  else
    printf '(sin tag previo)'
  fi
}

# -----------------------------------------------------------------------------
# Tags remotos (read-only): se resuelven con `git ls-remote`, nunca con fetch
# salvo en el release real (ADR-005).
# -----------------------------------------------------------------------------

REMOTE_TAGS=""

load_remote_tags() {
  local raw
  if ! raw=$(git ls-remote --tags origin); then
    error "No se pudieron leer los tags de origin (git ls-remote --tags origin). Revisá la conectividad y el remoto."
  fi
  # Formato de salida: "<tag> <sha>" (una línea por tag, resuelve annotated a su commit).
  REMOTE_TAGS=$(printf '%s\n' "${raw}" | awk '
    NF >= 2 {
      sha = $1
      ref = $2
      sub(/^refs\/tags\//, "", ref)
      peeled = (ref ~ /\^\{\}$/)
      if (peeled) sub(/\^\{\}$/, "", ref)
      if (ref == "") next
      if (peeled || !(ref in shas)) shas[ref] = sha
    }
    END { for (t in shas) printf "%s %s\n", t, shas[t] }
  ')
}

# Última versión semver de un artefacto según los tags remotos, o vacío.
remote_latest_version() {
  local prefix="$1"
  local best=""
  local tag version

  while read -r tag _sha; do
    [[ -z "${tag}" ]] && continue
    case "${tag}" in
      "${prefix}v"[0-9]*) ;;
      *) continue ;;
    esac
    version="${tag#"${prefix}v"}"
    [[ "${version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || continue
    if [[ -z "${best}" ]] || [[ "$(printf '%s\n%s\n' "${best}" "${version}" | sort -V | tail -1)" == "${version}" ]]; then
      best="${version}"
    fi
  done <<< "${REMOTE_TAGS}"

  printf '%s' "${best}"
}

# SHA del commit apuntado por un tag remoto, o vacío.
remote_tag_sha() {
  local tag="$1"
  printf '%s\n' "${REMOTE_TAGS}" | awk -v t="${tag}" '$1 == t { print $2; exit }'
}

# Verifica que el commit base esté presente localmente antes de difiar.
verify_base_object() {
  local sha="$1"
  local tag="$2"

  [[ -z "${sha}" ]] && return 0
  if ! git cat-file -e "${sha}^{commit}" 2>/dev/null; then
    error "El tag ${tag} apunta a ${sha}, que no está presente localmente. Ejecutá 'git fetch --tags origin' (o 'git pull') y reintentá."
  fi
}

# Archivos cambiados desde un commit base dentro de una allowlist.
# Sin tag previo devuelve el centinela __FIRST__ (primera release).
changed_files() {
  local base_sha="$1"
  local exclude="$2"
  shift 2

  if [[ -z "${base_sha}" ]]; then
    printf '__FIRST__'
    return 0
  fi

  local files
  files=$(git diff --name-only "${base_sha}..HEAD" -- "$@")
  if [[ -n "${exclude}" && -n "${files}" ]]; then
    files=$(printf '%s\n' "${files}" | grep -Ev "${exclude}" || true)
  fi
  printf '%s' "${files}"
}

# Calcula el plan de un artefacto: "<release 0|1>|<versión>|<decisión>".
compute_plan() {
  local last_version="$1"
  local changed="$2"
  local explicit_version="$3"
  local skip="$4"
  local fallback_version="$5"
  local skip_reason="$6"

  if [[ "${skip}" -eq 1 ]]; then
    printf '0||omitir (%s)' "${skip_reason}"
    return 0
  fi
  if [[ -n "${explicit_version}" ]]; then
    printf '1|%s|publicar (forzado por versión explícita)' "${explicit_version}"
    return 0
  fi
  if [[ -n "${changed}" ]]; then
    if [[ -n "${last_version}" ]]; then
      printf '1|%s|publicar (cambios detectados)' "$(bump_patch_version "${last_version}")"
    else
      printf '1|%s|publicar (primera release)' "${fallback_version}"
    fi
    return 0
  fi
  printf '0||omitir (sin cambios)'
}

# Valida semver de una versión que se va a publicar.
validate_version() {
  local artifact="$1"
  local version="$2"
  if ! [[ "${version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    error "Versión de ${artifact} inválida: '${version}'. Debe seguir semver (X.Y.Z)."
  fi
}

# Aborta si el tag ya existe local o remotamente (idempotencia).
ensure_tag_available() {
  local tag="$1"

  if git rev-parse -q --verify "refs/tags/${tag}" >/dev/null 2>&1; then
    error "El tag ${tag} ya existe localmente. Elegí otra versión."
  fi
  if printf '%s\n' "${REMOTE_TAGS}" | awk -v t="${tag}" '$1 == t { found = 1 } END { exit !found }'; then
    error "El tag ${tag} ya existe en origin. Elegí otra versión."
  fi
}

# Imprime el detalle del plan de un artefacto (--check / --list-tags).
print_artifact_plan() {
  local label="$1"
  local tag_prefix="$2"
  local last_version="$3"
  local changed="$4"
  local release="$5"
  local version="$6"
  local decision="$7"

  log "[${label}] último tag: $(artifact_last_tag_label "${tag_prefix}" "${last_version}")"

  if [[ "${changed}" == "__FIRST__" ]]; then
    log "[${label}] cambios detectados: primera release (no hay tag previo)"
  elif [[ -z "${changed}" ]]; then
    log "[${label}] cambios detectados: sin cambios"
  else
    log "[${label}] cambios detectados: $(printf '%s\n' "${changed}" | wc -l | tr -d ' ') archivo(s)"
    printf '%s\n' "${changed}" | sed 's/^/    - /' >&2
  fi

  if [[ "${release}" -eq 1 ]]; then
    log "[${label}] versión a crear: $(artifact_tag_name "${tag_prefix}" "${version}")"
  else
    log "[${label}] versión a crear: (ninguna)"
  fi

  log "[${label}] decisión: ${decision}"
}

# -----------------------------------------------------------------------------
# Workflows
# -----------------------------------------------------------------------------

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
        --json url,headBranch,headSha \
        --limit 30 2>/dev/null \
        | jq -r --arg ref "${ref}" --arg sha "${HEAD_SHA}" '[.[] | select(.headBranch == $ref or .headSha == $sha)][0].url // empty' || true)
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

# Espera, verifica y deja la URL del workflow en WORKFLOW_RUN_URL.
# Marca WORKFLOW_FAILED=1 si no aparece el run o si la conclusión no es success.
# No se usa command substitution: el estado de WORKFLOW_FAILED debe sobrevivir.
check_workflow() {
  local workflow_file="$1"
  local ref="$2"
  local label="$3"

  WORKFLOW_RUN_URL=""
  log "Esperando que GitHub Actions registre el workflow de ${label} (${workflow_file}, tag ${ref})..."
  WORKFLOW_RUN_URL=$(wait_for_workflow_run "${workflow_file}" "${ref}" || true)

  if [[ -z "${WORKFLOW_RUN_URL}" || "${WORKFLOW_RUN_URL}" == "null" ]]; then
    log "ERROR: no se encontró el run del workflow de ${label} tras los reintentos."
    WORKFLOW_FAILED=1
    return 0
  fi

  if ! wait_for_workflow_conclusion "${WORKFLOW_RUN_URL}" "${label} ${ref}"; then
    WORKFLOW_FAILED=1
  fi
}

# Verifica en Docker Hub que la versión de la app no exista ya (solo release real).
verify_app_version_not_in_docker_hub() {
  local version="$1"
  local response=""
  local latest=""

  log "Consultando Docker Hub (${REPO}) para verificar que ${version} no exista..."
  if ! response=$(curl -fsSL "${DOCKER_HUB_API}" 2>/dev/null); then
    warn "No se pudo consultar Docker Hub; se continúa con la validación por tags."
    return 0
  fi

  latest=$(parse_latest_version "${response}")
  if [[ -n "${latest}" ]]; then
    log "Última versión publicada en Docker Hub: ${latest}"
  fi

  if version_exists_in_docker_hub "${response}" "${version}"; then
    error "La versión ${version} ya existe en Docker Hub (tag v${version}). Elegí otra versión."
  fi
}

# -----------------------------------------------------------------------------
# Argumentos
# -----------------------------------------------------------------------------

CHECK_ONLY=0
LIST_TAGS=0
NO_MCP=0
NO_TAILSCALE=0
APP_VERSION_ARG=""
MCP_VERSION_ARG=""
TAILSCALE_VERSION_ARG=""

APP_VERSION_REQUESTED="${APP_VERSION:-}"
MCP_VERSION_REQUESTED="${MCP_VERSION:-}"
TAILSCALE_VERSION_REQUESTED="${TAILSCALE_VERSION:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --check)
      CHECK_ONLY=1
      ;;
    --list-tags)
      LIST_TAGS=1
      ;;
    --no-mcp)
      NO_MCP=1
      ;;
    --no-tailscale)
      NO_TAILSCALE=1
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    -*)
      error "Opción desconocida: $1 (usá --help)"
      ;;
    *)
      if [[ -z "${APP_VERSION_ARG}" ]]; then
        APP_VERSION_ARG="$1"
      elif [[ -z "${MCP_VERSION_ARG}" ]]; then
        MCP_VERSION_ARG="$1"
      elif [[ -z "${TAILSCALE_VERSION_ARG}" ]]; then
        TAILSCALE_VERSION_ARG="$1"
      else
        error "Demasiados argumentos. Uso: ./scripts/release.sh [app-version] [mcp-version] [tailscale-version]"
      fi
      ;;
  esac
  shift
done

# Los argumentos posicionales ganan sobre las variables de entorno.
if [[ -n "${APP_VERSION_ARG}" ]]; then
  APP_VERSION_REQUESTED="${APP_VERSION_ARG}"
fi
if [[ -n "${MCP_VERSION_ARG}" ]]; then
  MCP_VERSION_REQUESTED="${MCP_VERSION_ARG}"
fi
if [[ -n "${TAILSCALE_VERSION_ARG}" ]]; then
  TAILSCALE_VERSION_REQUESTED="${TAILSCALE_VERSION_ARG}"
fi

# `skip`/`none`/`no` desactivan el artefacto.
normalize_skip() {
  case "$1" in
    skip | none | no) printf '1' ;;
    *) printf '0' ;;
  esac
}

if [[ "$(normalize_skip "${MCP_VERSION_REQUESTED}")" == "1" ]]; then
  NO_MCP=1
  MCP_VERSION_REQUESTED=""
fi
if [[ "$(normalize_skip "${TAILSCALE_VERSION_REQUESTED}")" == "1" ]]; then
  NO_TAILSCALE=1
  TAILSCALE_VERSION_REQUESTED=""
fi

# -----------------------------------------------------------------------------
# Validaciones de entorno
# -----------------------------------------------------------------------------

log "Validando herramientas necesarias..."
command -v git >/dev/null 2>&1 || error "git no está instalado"

if [[ "${CHECK_ONLY}" -eq 1 || "${LIST_TAGS}" -eq 1 ]]; then
  log "Modo de solo lectura: no se validan gh ni curl (no se escribe ni se publica nada)."
  if ! command -v gh >/dev/null 2>&1; then
    warn "gh no está instalado; el plan es válido pero el release real lo requiere."
  fi
else
  command -v curl >/dev/null 2>&1 || error "curl no está instalado"
  command -v gh >/dev/null 2>&1 || error "gh no está instalado. Instalalo desde: https://cli.github.com/"
  log "Validando autenticación con gh..."
  gh auth status >/dev/null 2>&1 || error "gh no está autenticado. Ejecutá: gh auth login"
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${ROOT_DIR}"

CURRENT_BRANCH=$(git branch --show-current)
if [[ "${CURRENT_BRANCH}" != "main" ]]; then
  if [[ "${CHECK_ONLY}" -eq 1 || "${LIST_TAGS}" -eq 1 ]]; then
    warn "estás en ${CURRENT_BRANCH}; el modo de solo lectura no escribe y el plan no es ejecutable hasta mergear a main."
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

if [[ "${CHECK_ONLY}" -eq 1 || "${LIST_TAGS}" -eq 1 ]]; then
  log "Aviso: el modo de solo lectura omite la validación de ramas sin mergear."
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
# Sincronizar tags solo en release real
# -----------------------------------------------------------------------------

if [[ "${CHECK_ONLY}" -eq 1 || "${LIST_TAGS}" -eq 1 ]]; then
  log "Modo de solo lectura: se resuelven los tags con 'git ls-remote' (sin 'git fetch --tags')."
else
  log "Sincronizando tags con origin..."
  git fetch --tags origin >/dev/null 2>&1 || error "No se pudo hacer fetch de tags desde origin"
fi

log "Resolviendo el último tag de cada artefacto..."
load_remote_tags

APP_LAST_VERSION=$(remote_latest_version "")
MCP_LAST_VERSION=$(remote_latest_version "mcp-")
TAILSCALE_LAST_VERSION=$(remote_latest_version "tailscale-")

APP_BASE_SHA=""
MCP_BASE_SHA=""
TAILSCALE_BASE_SHA=""

if [[ -n "${APP_LAST_VERSION}" ]]; then
  APP_BASE_SHA=$(remote_tag_sha "$(artifact_tag_name "" "${APP_LAST_VERSION}")")
  verify_base_object "${APP_BASE_SHA}" "$(artifact_tag_name "" "${APP_LAST_VERSION}")"
fi
if [[ -n "${MCP_LAST_VERSION}" ]]; then
  MCP_BASE_SHA=$(remote_tag_sha "$(artifact_tag_name "mcp-" "${MCP_LAST_VERSION}")")
  verify_base_object "${MCP_BASE_SHA}" "$(artifact_tag_name "mcp-" "${MCP_LAST_VERSION}")"
fi
if [[ -n "${TAILSCALE_LAST_VERSION}" ]]; then
  TAILSCALE_BASE_SHA=$(remote_tag_sha "$(artifact_tag_name "tailscale-" "${TAILSCALE_LAST_VERSION}")")
  verify_base_object "${TAILSCALE_BASE_SHA}" "$(artifact_tag_name "tailscale-" "${TAILSCALE_LAST_VERSION}")"
fi

# -----------------------------------------------------------------------------
# Detección de cambios por artefacto
# -----------------------------------------------------------------------------

log "Detectando cambios desde el último tag de cada artefacto..."
APP_CHANGED=$(changed_files "${APP_BASE_SHA}" "${APP_EXCLUDE}" "${APP_PATHS[@]}")
MCP_CHANGED=$(changed_files "${MCP_BASE_SHA}" "" "${MCP_PATHS[@]}")
TAILSCALE_CHANGED=$(changed_files "${TAILSCALE_BASE_SHA}" "" "${TAILSCALE_PATHS[@]}")

# -----------------------------------------------------------------------------
# Plan por artefacto
# -----------------------------------------------------------------------------

APP_FALLBACK_VERSION="$(package_json_version)"

IFS='|' read -r APP_RELEASE APP_VERSION APP_DECISION <<< "$(compute_plan \
  "${APP_LAST_VERSION}" "${APP_CHANGED}" "${APP_VERSION_REQUESTED}" 0 "${APP_FALLBACK_VERSION}" "")"
IFS='|' read -r MCP_RELEASE MCP_VERSION MCP_DECISION <<< "$(compute_plan \
  "${MCP_LAST_VERSION}" "${MCP_CHANGED}" "${MCP_VERSION_REQUESTED}" "${NO_MCP}" "0.1.0" "--no-mcp")"
IFS='|' read -r TAILSCALE_RELEASE TAILSCALE_VERSION TAILSCALE_DECISION <<< "$(compute_plan \
  "${TAILSCALE_LAST_VERSION}" "${TAILSCALE_CHANGED}" "${TAILSCALE_VERSION_REQUESTED}" "${NO_TAILSCALE}" "0.1.0" "--no-tailscale")"

if [[ "${APP_RELEASE}" -eq 1 ]]; then
  validate_version "app" "${APP_VERSION}"
fi
if [[ "${MCP_RELEASE}" -eq 1 ]]; then
  validate_version "MCP" "${MCP_VERSION}"
  [[ -f "${MCP_WORKFLOW_PATH}" ]] || error "Falta ${MCP_WORKFLOW_PATH}; no se puede publicar el MCP."
fi
if [[ "${TAILSCALE_RELEASE}" -eq 1 ]]; then
  validate_version "wrapper Tailscale" "${TAILSCALE_VERSION}"
  [[ -f "${TAILSCALE_WORKFLOW_PATH}" ]] || error "Falta ${TAILSCALE_WORKFLOW_PATH}; no se puede publicar el wrapper Tailscale."
fi

# Avisos de versión explícita sin cambios (REQ-005).
if [[ "${APP_RELEASE}" -eq 1 && -n "${APP_VERSION_REQUESTED}" && -z "${APP_CHANGED}" ]]; then
  warn "app: no se detectaron cambios, pero se publica v${APP_VERSION} por versión explícita."
fi
if [[ "${MCP_RELEASE}" -eq 1 && -n "${MCP_VERSION_REQUESTED}" && -z "${MCP_CHANGED}" ]]; then
  warn "MCP: no se detectaron cambios, pero se publica mcp-v${MCP_VERSION} por versión explícita."
fi
if [[ "${TAILSCALE_RELEASE}" -eq 1 && -n "${TAILSCALE_VERSION_REQUESTED}" && -z "${TAILSCALE_CHANGED}" ]]; then
  warn "Tailscale: no se detectaron cambios, pero se publica tailscale-v${TAILSCALE_VERSION} por versión explícita."
fi

# Idempotencia: ningún tag a crear puede existir (local o remoto) antes de escribir.
if [[ "${APP_RELEASE}" -eq 1 ]]; then
  ensure_tag_available "$(artifact_tag_name "" "${APP_VERSION}")"
fi
if [[ "${MCP_RELEASE}" -eq 1 ]]; then
  ensure_tag_available "$(artifact_tag_name "mcp-" "${MCP_VERSION}")"
fi
if [[ "${TAILSCALE_RELEASE}" -eq 1 ]]; then
  ensure_tag_available "$(artifact_tag_name "tailscale-" "${TAILSCALE_VERSION}")"
fi

# -----------------------------------------------------------------------------
# Salidas de solo lectura: --check y --list-tags
# -----------------------------------------------------------------------------

if [[ "${LIST_TAGS}" -eq 1 ]]; then
  echo ""
  log "=============================================="
  log "Tags por artefacto (--list-tags): no se escribe nada"
  for entry in "app:" "MCP:mcp-" "Tailscale:tailscale-"; do
    label="${entry%%:*}"
    prefix="${entry#*:}"
    case "${prefix}" in
      "") last="${APP_LAST_VERSION}" changed="${APP_CHANGED}" ;;
      "mcp-") last="${MCP_LAST_VERSION}" changed="${MCP_CHANGED}" ;;
      *) last="${TAILSCALE_LAST_VERSION}" changed="${TAILSCALE_CHANGED}" ;;
    esac
    if [[ "${changed}" == "__FIRST__" ]]; then
      pending="sí (primera release, sin tag previo)"
    elif [[ -n "${changed}" ]]; then
      pending="sí ($(printf '%s\n' "${changed}" | wc -l | tr -d ' ') archivo(s))"
    else
      pending="no"
    fi
    log "[${label}] último tag: $(artifact_last_tag_label "${prefix}" "${last}") — cambios pendientes: ${pending}"
  done
  log "=============================================="
  echo ""
  exit 0
fi

if [[ "${CHECK_ONLY}" -eq 1 ]]; then
  echo ""
  log "=============================================="
  log "Plan de release (--check): no se escribe nada"
  print_artifact_plan "app" "" "${APP_LAST_VERSION}" "${APP_CHANGED}" "${APP_RELEASE}" "${APP_VERSION}" "${APP_DECISION}"
  print_artifact_plan "MCP" "mcp-" "${MCP_LAST_VERSION}" "${MCP_CHANGED}" "${MCP_RELEASE}" "${MCP_VERSION}" "${MCP_DECISION}"
  print_artifact_plan "Tailscale" "tailscale-" "${TAILSCALE_LAST_VERSION}" "${TAILSCALE_CHANGED}" "${TAILSCALE_RELEASE}" "${TAILSCALE_VERSION}" "${TAILSCALE_DECISION}"

  TAGS_PLAN=""
  if [[ "${APP_RELEASE}" -eq 1 ]]; then
    TAGS_PLAN="$(artifact_tag_name "" "${APP_VERSION}")"
  fi
  if [[ "${MCP_RELEASE}" -eq 1 ]]; then
    TAGS_PLAN="${TAGS_PLAN:+${TAGS_PLAN}, }$(artifact_tag_name "mcp-" "${MCP_VERSION}")"
  fi
  if [[ "${TAILSCALE_RELEASE}" -eq 1 ]]; then
    TAGS_PLAN="${TAGS_PLAN:+${TAGS_PLAN}, }$(artifact_tag_name "tailscale-" "${TAILSCALE_VERSION}")"
  fi

  if [[ -z "${TAGS_PLAN}" ]]; then
    log "Tags que se crearían: ninguno (nada para liberar)"
    log "Archivos que se actualizarían: ninguno"
  else
    log "Tags que se crearían: ${TAGS_PLAN}"
    if [[ "${APP_RELEASE}" -eq 1 ]]; then
      log "Archivos que se actualizarían: docker-compose.yml, frontend/package.json"
    else
      log "Archivos que se actualizarían: ninguno (solo tags)"
    fi
  fi
  log "=============================================="
  echo ""
  exit 0
fi

# -----------------------------------------------------------------------------
# Nada para liberar
# -----------------------------------------------------------------------------

if [[ "${APP_RELEASE}" -eq 0 && "${MCP_RELEASE}" -eq 0 && "${TAILSCALE_RELEASE}" -eq 0 ]]; then
  log "Nada para liberar: ningún artefacto tiene cambios relevantes y no hay versiones explícitas."
  log "No se crean commits, tags ni workflows."
  exit 0
fi

# -----------------------------------------------------------------------------
# Guarda extra: la versión de la app no debe existir en Docker Hub
# -----------------------------------------------------------------------------

if [[ "${APP_RELEASE}" -eq 1 ]]; then
  verify_app_version_not_in_docker_hub "${APP_VERSION}"
fi

# -----------------------------------------------------------------------------
# Actualizar archivos (solo si se publica la app)
# -----------------------------------------------------------------------------

COMMIT_CREATED=0
if [[ "${APP_RELEASE}" -eq 1 ]]; then
  log "Actualizando docker-compose.yml (VERSION=${APP_VERSION})..."
  sed -i.bak "s/\(- VERSION=\).*/\1${APP_VERSION}/" docker-compose.yml
  rm -f docker-compose.yml.bak

  log "Actualizando frontend/package.json (version=${APP_VERSION})..."
  if command -v jq >/dev/null 2>&1; then
    TMP_PKG=$(mktemp)
    jq --arg v "${APP_VERSION}" '.version = $v' frontend/package.json > "${TMP_PKG}"
    mv "${TMP_PKG}" frontend/package.json
  else
    sed -i.bak "s/\"version\": \"[^\"]*\"/\"version\": \"${APP_VERSION}\"/" frontend/package.json
    rm -f frontend/package.json.bak
  fi

  if git diff --quiet; then
    warn "Los archivos ya tenían la versión ${APP_VERSION}; se omite el commit de bump."
  else
    log "Creando commit de bump..."
    git add docker-compose.yml frontend/package.json
    git commit -m "bump version to ${APP_VERSION}"
    COMMIT_CREATED=1
  fi
else
  log "app: sin release; no se tocan docker-compose.yml ni frontend/package.json."
fi

# -----------------------------------------------------------------------------
# Tags
# -----------------------------------------------------------------------------

if [[ "${APP_RELEASE}" -eq 1 ]]; then
  log "Creando tag anotado $(artifact_tag_name "" "${APP_VERSION}")..."
  git tag -a "$(artifact_tag_name "" "${APP_VERSION}")" -m "Release v${APP_VERSION}"
fi

if [[ "${MCP_RELEASE}" -eq 1 ]]; then
  log "Creando tag anotado $(artifact_tag_name "mcp-" "${MCP_VERSION}")..."
  git tag -a "$(artifact_tag_name "mcp-" "${MCP_VERSION}")" -m "Release MCP v${MCP_VERSION}"
fi

if [[ "${TAILSCALE_RELEASE}" -eq 1 ]]; then
  log "Creando tag anotado $(artifact_tag_name "tailscale-" "${TAILSCALE_VERSION}")..."
  git tag -a "$(artifact_tag_name "tailscale-" "${TAILSCALE_VERSION}")" -m "Release Tailscale wrapper v${TAILSCALE_VERSION}"
fi

HEAD_SHA=$(git rev-parse HEAD)

# -----------------------------------------------------------------------------
# Push
# -----------------------------------------------------------------------------

if [[ "${COMMIT_CREATED}" -eq 1 ]]; then
  log "Pusheando commit a origin/main..."
  git push origin main
else
  log "Sin commit de bump; main no necesita push."
fi

if [[ "${APP_RELEASE}" -eq 1 ]]; then
  log "Pusheando tag $(artifact_tag_name "" "${APP_VERSION}")..."
  git push origin "$(artifact_tag_name "" "${APP_VERSION}")"
fi

if [[ "${MCP_RELEASE}" -eq 1 ]]; then
  log "Pusheando tag $(artifact_tag_name "mcp-" "${MCP_VERSION}")..."
  git push origin "$(artifact_tag_name "mcp-" "${MCP_VERSION}")"
fi

if [[ "${TAILSCALE_RELEASE}" -eq 1 ]]; then
  log "Pusheando tag $(artifact_tag_name "tailscale-" "${TAILSCALE_VERSION}")..."
  git push origin "$(artifact_tag_name "tailscale-" "${TAILSCALE_VERSION}")"
fi

# -----------------------------------------------------------------------------
# Esperar los workflows disparados
# -----------------------------------------------------------------------------

WORKFLOW_FAILED=0
APP_RUN_URL=""
MCP_RUN_URL=""
TAILSCALE_RUN_URL=""

if [[ "${APP_RELEASE}" -eq 1 ]]; then
  check_workflow "${WORKFLOW_FILE}" "$(artifact_tag_name "" "${APP_VERSION}")" "App"
  APP_RUN_URL="${WORKFLOW_RUN_URL}"
fi

if [[ "${MCP_RELEASE}" -eq 1 ]]; then
  check_workflow "${MCP_WORKFLOW_FILE}" "$(artifact_tag_name "mcp-" "${MCP_VERSION}")" "MCP"
  MCP_RUN_URL="${WORKFLOW_RUN_URL}"
fi

if [[ "${TAILSCALE_RELEASE}" -eq 1 ]]; then
  check_workflow "${TAILSCALE_WORKFLOW_FILE}" "$(artifact_tag_name "tailscale-" "${TAILSCALE_VERSION}")" "Tailscale"
  TAILSCALE_RUN_URL="${WORKFLOW_RUN_URL}"
fi

# -----------------------------------------------------------------------------
# Resumen
# -----------------------------------------------------------------------------

echo ""
log "=============================================="
if [[ "${APP_RELEASE}" -eq 1 ]]; then
  log "App v${APP_VERSION} procesado"
  log "App Docker Hub: https://hub.docker.com/r/${REPO}/tags"
  if [[ -n "${APP_RUN_URL}" && "${APP_RUN_URL}" != "null" ]]; then
    log "App workflow: ${APP_RUN_URL}"
  else
    log "App workflow: https://github.com/${GITHUB_REPO}/actions"
    log "(No se pudo obtener la URL exacta; revisá: gh run list --workflow=\"${WORKFLOW_NAME}\")"
  fi
else
  log "App: sin cambios, no se publicó."
fi

if [[ "${MCP_RELEASE}" -eq 1 ]]; then
  log "MCP mcp-v${MCP_VERSION}: https://hub.docker.com/r/${MCP_REPO}/tags"
  if [[ -n "${MCP_RUN_URL}" && "${MCP_RUN_URL}" != "null" ]]; then
    log "MCP workflow: ${MCP_RUN_URL}"
  else
    log "MCP workflow: https://github.com/${GITHUB_REPO}/actions"
    log "(No se pudo obtener la URL exacta; revisá: gh run list --workflow=\"${MCP_WORKFLOW_NAME}\")"
  fi
else
  log "MCP: sin cambios, no se publicó."
fi

if [[ "${TAILSCALE_RELEASE}" -eq 1 ]]; then
  log "Tailscale tailscale-v${TAILSCALE_VERSION}: https://hub.docker.com/r/${TAILSCALE_REPO}/tags"
  if [[ -n "${TAILSCALE_RUN_URL}" && "${TAILSCALE_RUN_URL}" != "null" ]]; then
    log "Tailscale workflow: ${TAILSCALE_RUN_URL}"
  else
    log "Tailscale workflow: https://github.com/${GITHUB_REPO}/actions"
    log "(No se pudo obtener la URL exacta; revisá: gh run list --workflow=\"${TAILSCALE_WORKFLOW_NAME}\")"
  fi
else
  log "Tailscale: sin cambios, no se publicó."
fi
log "=============================================="
echo ""

if [[ "${WORKFLOW_FAILED}" -eq 1 ]]; then
  error "Uno o más workflows fallaron; los tags quedan creados para reintentar el workflow desde GitHub Actions. Revisá las URLs anteriores."
fi

if [[ -n "${APP_RUN_URL}" && "${APP_RUN_URL}" != "null" ]]; then
  open_url "${APP_RUN_URL}"
elif [[ -n "${MCP_RUN_URL}" && "${MCP_RUN_URL}" != "null" ]]; then
  open_url "${MCP_RUN_URL}"
elif [[ -n "${TAILSCALE_RUN_URL}" && "${TAILSCALE_RUN_URL}" != "null" ]]; then
  open_url "${TAILSCALE_RUN_URL}"
fi
