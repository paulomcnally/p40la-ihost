#!/usr/bin/env bash
#
# Crea o rota la API key global del sistema p40la-ihost (SPEC-145).
#
# La key se muestra una sola vez. La sesión de usuario se usa únicamente para
# provisionar; el MCP después usa solo P40LA_API_KEY.
#
# Uso:
#   EMAIL=tu@example.com PASSWORD='...' ./scripts/create-api-key.sh
#
# Variables:
#   BASE_URL   URL del server (default: http://localhost:${PORT:-8088})
#   PORT       Puerto si no se define BASE_URL (default 8088)
#   EMAIL      Email del usuario administrador (obligatorio)
#   PASSWORD   Contraseña del usuario (obligatorio)
#   KEY_FILE   Archivo opcional para guardar la key con permisos 0600
#   COOKIE_JAR Archivo temporal de cookies (default: mktemp)
#
# Requisitos: curl, jq

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:${PORT:-8088}}"
EMAIL="${EMAIL:-}"
PASSWORD="${PASSWORD:-}"
KEY_FILE="${KEY_FILE:-}"
COOKIE_JAR="${COOKIE_JAR:-$(mktemp)}"

cleanup() {
  rm -f "${COOKIE_JAR}"
}
trap cleanup EXIT

error() {
  echo "ERROR: $*" >&2
  exit 1
}

command -v curl >/dev/null 2>&1 || error "curl no está instalado"
command -v jq >/dev/null 2>&1 || error "jq no está instalado"

[[ -n "${EMAIL}" ]] || error "definí EMAIL"
[[ -n "${PASSWORD}" ]] || error "definí PASSWORD"

if ! curl -sf -o /dev/null -c "${COOKIE_JAR}" "${BASE_URL}/api/login" \
  -X POST -H 'Content-Type: application/json' \
  -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\"}"; then
  error "login falló contra ${BASE_URL} (¿server corriendo y credenciales válidas?)"
fi

response="$(curl -sf -b "${COOKIE_JAR}" -X POST "${BASE_URL}/api/api-key/regenerate" \
  -H 'Content-Type: application/json')" || error "no se pudo generar la API key"

key="$(printf '%s' "${response}" | jq -r '.api_key // empty')"
[[ -n "${key}" ]] || error "la respuesta no incluyó api_key"

if [[ -n "${KEY_FILE}" ]]; then
  umask 077
  printf '%s' "${key}" > "${KEY_FILE}"
  chmod 600 "${KEY_FILE}"
  echo "API key guardada en ${KEY_FILE} (permisos 0600)."
else
  echo "${key}"
  echo "Guardala ahora: no se vuelve a mostrar." >&2
fi
