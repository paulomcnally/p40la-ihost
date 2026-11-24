#!/usr/bin/env bash
#
# check-mcp-impact.sh — Guardrail DELIV-MCP-IMPACT (SPEC-153).
#
# Cruza los archivos cambiados contra docs/mcp/tools/impact.json y reporta
# qué tools MCP pueden verse afectadas. Con --gate exige que la spec en curso
# declare el impacto y la validación de un subagente, y corre los tests MCP.
#
# Uso:
#   ./scripts/check-mcp-impact.sh --check-map
#   ./scripts/check-mcp-impact.sh --files internal/services/bill_bulk.go
#   ./scripts/check-mcp-impact.sh --staged
#   ./scripts/check-mcp-impact.sh --staged --gate
#   ./scripts/check-mcp-impact.sh --staged --json
#
# El comando es de desarrollo: no se ejecuta en el iHost.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

exec go run ./scripts/mcp-impact "$@"
