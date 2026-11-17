#!/usr/bin/env node
//
// Guardrails de UI.
//
// 1) SPEC-118: prohíbe el <select> nativo. Los dropdowns DEBEN usar el
//    componente reutilizable frontend/src/components/Select.tsx (dropdown custom
//    con portal, tokens de tema y darkmode). El <select> nativo muestra el
//    dropdown del sistema operativo y rompe el patrón visual de la app.
//
// 2) SPEC-135: prohíbe <input type="checkbox"> fuera de Toggle.tsx. Todo
//    booleano de entidad/configuración DEBE usar el Toggle estilo iOS. Los
//    checkboxes de selección múltiple de filas/items se agregan a la allowlist
//    con justificación.
//
// Se ejecuta como parte de `npm run build` (y con `npm run check:ui`).
//
const fs = require('fs')
const path = require('path')

const srcDir = path.join(__dirname, '..', 'src')
const selectOffenders = []
const checkboxOffenders = []
let scanned = 0

// Implementación del toggle: es el único lugar donde vive el checkbox nativo.
const CHECKBOX_ALLOWLIST = new Set(['src/components/Toggle.tsx'])

// Comentarios de línea o bloques de comentario JSX que solo mencionan el tag.
function isCommentLine(trimmed) {
  return (
    trimmed.startsWith('//') ||
    trimmed.startsWith('*') ||
    trimmed.startsWith('/*') ||
    trimmed.startsWith('{/*')
  )
}

function walk(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      walk(full)
      continue
    }
    if (!/\.(tsx|jsx)$/.test(entry.name)) continue

    scanned++
    const rel = path.relative(path.join(__dirname, '..'), full).replaceAll('\\', '/')
    const lines = fs.readFileSync(full, 'utf8').split('\n')
    lines.forEach((line, i) => {
      if (isCommentLine(line.trim())) return
      if (/<select[\s>]/.test(line)) {
        selectOffenders.push(`${rel}:${i + 1}`)
      }
      if (/type=("|')checkbox\1/.test(line) && !CHECKBOX_ALLOWLIST.has(rel)) {
        checkboxOffenders.push(`${rel}:${i + 1}`)
      }
    })
  }
}

if (!fs.existsSync(srcDir)) {
  console.error(`[check-ui-rules] No existe ${srcDir}`)
  process.exit(1)
}

walk(srcDir)

if (selectOffenders.length > 0) {
  console.error('\n[check-ui-rules] ERROR: <select> nativo detectado.\n')
  for (const o of selectOffenders) console.error(`  - ${o}`)
  console.error(
    "\nUsá el componente custom: import Select from '../components/Select'.\n" +
      'Regla: AGENTS.md (dropdowns) y docs/project-rules.md §4.6. Ver SPEC-118.\n'
  )
  process.exit(1)
}

if (checkboxOffenders.length > 0) {
  console.error('\n[check-ui-rules] ERROR: checkbox nativo detectado.\n')
  for (const o of checkboxOffenders) console.error(`  - ${o}`)
  console.error(
    "\nUsá el componente Toggle: import Toggle from '../components/Toggle'.\n" +
      'Regla: AGENTS.md (booleanos) y docs/project-rules.md §4.10. Ver SPEC-135.\n' +
      'Solo las selecciones múltiples de filas/items pueden exceptuarse en CHECKBOX_ALLOWLIST.\n'
  )
  process.exit(1)
}

console.log(
  `[check-ui-rules] OK: ${scanned} archivos TSX/JSX sin <select> nativo ni checkbox nativo.`
)
