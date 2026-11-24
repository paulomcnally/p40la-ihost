#!/usr/bin/env bash
#
# check-mcp-docs.sh — Guardrail DELIV-MCP-DOCS (SPEC-145).
#
# Bloquea un commit que toca el código del MCP sin actualizar docs/mcp/.
#
# Uso:
#   ./scripts/check-mcp-docs.sh --staged      # archivos staged (hook)
#   ./scripts/check-mcp-docs.sh <archivo>...  # lista explícita (tests/manual)
#
# Superficie MCP:
#   cmd/mcp-server/  internal/mcp/  scripts/create-api-key.sh
#   internal/services/bill_bulk.go  internal/services/api_key.go
#   internal/api/bill_bulk_handlers.go  internal/api/api_key_handlers.go
#
# Los archivos *_test.go del MCP no exigen documentación por sí solos.

set -euo pipefail

usage() {
  echo "Uso: $0 --staged | $0 <archivo>..." >&2
  exit 2
}

files=""
if [[ "${1:-}" == "--staged" ]]; then
  shift
  files="$(git diff --cached --name-only --diff-filter=ACMR 2>/dev/null || true)"
elif [[ $# -gt 0 ]]; then
  files="$(printf '%s\n' "$@")"
else
  usage
fi

[[ -z "${files}" ]] && exit 0

mcp_regex='^(cmd/mcp-server/|internal/mcp/|scripts/create-api-key\.sh$|internal/services/bill_bulk\.go$|internal/services/api_key\.go$|internal/api/bill_bulk_handlers\.go$|internal/api/api_key_handlers\.go$)'

mcp_files="$(printf '%s\n' "${files}" | grep -Ev '_test\.go$' | grep -E "${mcp_regex}" || true)"
docs_files="$(printf '%s\n' "${files}" | grep -E '^docs/mcp/' || true)"

if [[ -n "${mcp_files}" && -z "${docs_files}" ]]; then
  echo "" >&2
  echo "[check-mcp-docs] ⛔ MCP modificado sin actualizar docs/mcp/." >&2
  echo "  Regla: DELIV-MCP-DOCS (docs/rules/70-delivery.md)" >&2
  echo "  Archivos MCP detectados:" >&2
  while IFS= read -r file; do
    [[ -n "${file}" ]] && echo "    - ${file}" >&2
  done <<< "${mcp_files}"
  echo "" >&2
  echo "  Actualizá docs/mcp/ en el mismo commit. Delegá la redacción a un" >&2
  echo "  subagente en paralelo (DELIV-MCP-DOCS-SUBAGENT)." >&2
  exit 1
fi

exit 0
