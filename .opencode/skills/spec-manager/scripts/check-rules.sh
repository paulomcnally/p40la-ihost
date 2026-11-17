#!/usr/bin/env bash
#
# check-rules.sh — Verifica que exista la estructura de reglas del proyecto.
# Parte del Paso 0 (Rules Bootstrap) de la skill spec-manager (SPEC-136).
#
# Uso:
#   ./check-rules.sh [directorio]     # exit 0 = OK, exit 1 = falta estructura
#
# No modifica nada. No valida el contenido: solo la existencia del contrato.

set -euo pipefail

DIR="${1:-$PWD}"

REQUIRED=(
  "AGENTS.md"
  "docs/rules/00-index.md"
  "docs/rules/10-architecture.md"
  "docs/rules/20-ui.md"
  "docs/rules/30-ux.md"
  "docs/rules/40-data.md"
  "docs/rules/50-security.md"
  "docs/rules/60-testing.md"
  "docs/rules/70-delivery.md"
)

missing=()
for f in "${REQUIRED[@]}"; do
  [[ -f "${DIR}/${f}" ]] || missing+=("${f}")
done

if [[ ${#missing[@]} -eq 0 ]]; then
  echo "[check-rules] OK: estructura de reglas completa en ${DIR}"
  exit 0
fi

echo "[check-rules] ⛔ DETENIDO — Estructura de reglas incompleta en ${DIR}" >&2
echo "  Faltan:" >&2
for f in "${missing[@]}"; do
  echo "    - ${f}" >&2
done
echo "" >&2
echo "  No se deben crear ni modificar specs sin las reglas del proyecto." >&2
echo "  Opciones:" >&2
echo "    1) Ejecutar scripts/init-rules.sh ${DIR}   (no destructivo)" >&2
echo "    2) Declarar las rutas reales en docs/rules/00-index.md" >&2
exit 1
