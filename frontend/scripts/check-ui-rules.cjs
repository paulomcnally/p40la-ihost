#!/usr/bin/env node
//
// SPEC-118: guardrail de UI.
//
// Prohíbe el <select> nativo en el frontend. Los dropdowns DEBEN usar el
// componente reutilizable frontend/src/components/Select.tsx (dropdown custom
// con portal, tokens de tema y darkmode). El <select> nativo muestra el
// dropdown del sistema operativo y rompe el patrón visual de la app.
//
// Se ejecuta como parte de `npm run build` (y con `npm run check:ui`).
//
const fs = require('fs')
const path = require('path')

const srcDir = path.join(__dirname, '..', 'src')
const offenders = []
let scanned = 0

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
    const rel = path.relative(path.join(__dirname, '..'), full)
    const lines = fs.readFileSync(full, 'utf8').split('\n')
    lines.forEach((line, i) => {
      if (isCommentLine(line.trim())) return
      if (/<select[\s>]/.test(line)) {
        offenders.push(`${rel}:${i + 1}`)
      }
    })
  }
}

if (!fs.existsSync(srcDir)) {
  console.error(`[check-ui-rules] No existe ${srcDir}`)
  process.exit(1)
}

walk(srcDir)

if (offenders.length > 0) {
  console.error('\n[check-ui-rules] ERROR: <select> nativo detectado.\n')
  for (const o of offenders) console.error(`  - ${o}`)
  console.error(
    '\nUsá el componente custom: import Select from \'../components/Select\'.\n' +
      'Regla: AGENTS.md (dropdowns) y docs/project-rules.md §4.6. Ver SPEC-118.\n'
  )
  process.exit(1)
}

console.log(`[check-ui-rules] OK: ${scanned} archivos TSX/JSX sin <select> nativo.`)
