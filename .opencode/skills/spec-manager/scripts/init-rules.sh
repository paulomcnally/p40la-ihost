#!/usr/bin/env bash
#
# init-rules.sh — Genera la estructura de reglas de un proyecto (SPEC-136).
#
# Uso:
#   ./init-rules.sh [directorio-destino]
#
# Comportamiento:
#   - Crea AGENTS.md y docs/rules/*.md SOLO si no existen.
#   - Nunca sobrescribe, nunca borra, nunca renombra.
#   - Idempotente: correrlo dos veces no cambia nada.
#   - No toca opencode.json ni crea specs.
#
# Salida: líneas "creado", "omitido" y un resumen final.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
TEMPLATES="${SKILL_DIR}/references"
DEST="${1:-$PWD}"

if [[ ! -d "${TEMPLATES}" ]]; then
  echo "ERROR: no se encontraron templates en ${TEMPLATES}" >&2
  exit 1
fi

if [[ ! -d "${DEST}" ]]; then
  echo "ERROR: el directorio destino no existe: ${DEST}" >&2
  exit 1
fi

created=0
omitted=0

copy_rule_file() {
  local src="$1"
  local rel="$2"
  local dst="${DEST}/${rel}"

  if [[ -e "${dst}" ]]; then
    echo "omitido   ${rel} (ya existe)"
    omitted=$((omitted + 1))
    return 0
  fi

  mkdir -p "$(dirname "${dst}")"
  cp "${src}" "${dst}"
  echo "creado    ${rel}"
  created=$((created + 1))
}

copy_rule_file "${TEMPLATES}/agents-template.md"        "AGENTS.md"
copy_rule_file "${TEMPLATES}/rules-index-template.md"   "docs/rules/00-index.md"
copy_rule_file "${TEMPLATES}/rules/10-architecture.md"  "docs/rules/10-architecture.md"
copy_rule_file "${TEMPLATES}/rules/20-ui.md"            "docs/rules/20-ui.md"
copy_rule_file "${TEMPLATES}/rules/30-ux.md"            "docs/rules/30-ux.md"
copy_rule_file "${TEMPLATES}/rules/40-data.md"          "docs/rules/40-data.md"
copy_rule_file "${TEMPLATES}/rules/50-security.md"      "docs/rules/50-security.md"
copy_rule_file "${TEMPLATES}/rules/60-testing.md"       "docs/rules/60-testing.md"
copy_rule_file "${TEMPLATES}/rules/70-delivery.md"     "docs/rules/70-delivery.md"
copy_rule_file "${TEMPLATES}/hooks/pre-commit"          ".githooks/pre-commit"
copy_rule_file "${TEMPLATES}/hooks/commit-msg"          ".githooks/commit-msg"
copy_rule_file "${TEMPLATES}/hooks/install-hooks.sh"    "scripts/install-hooks.sh"

chmod +x "${DEST}/.githooks/pre-commit" "${DEST}/.githooks/commit-msg" \
         "${DEST}/scripts/install-hooks.sh" 2>/dev/null || true

echo ""
echo "Resumen: ${created} creados, ${omitted} omitidos en ${DEST}"

if [[ ${created} -gt 0 ]]; then
  echo ""
  echo "Pendiente (manual, requiere confirmación del usuario):"
  echo "  - Completar los placeholders {{...}} de docs/rules/ y AGENTS.md."
  echo "  - Declarar 'instructions': [\"AGENTS.md\", \"docs/rules/*.md\"] en opencode.json."
  echo "  - Activar los hooks: ./scripts/install-hooks.sh"
fi
