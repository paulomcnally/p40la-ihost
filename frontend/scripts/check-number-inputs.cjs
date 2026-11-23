#!/usr/bin/env node
// Guardrail UI-NUMBER-INPUT (SPEC-140): los campos numéricos nuevos deben usar
// NumberInput, no <input type="number">. Los archivos legacy están listados en
// number-input-legacy.txt y se migran en una spec aparte.

const fs = require('fs')
const path = require('path')

const root = path.resolve(__dirname, '..')
const srcDir = path.join(root, 'src')
const legacyFile = path.join(__dirname, 'number-input-legacy.txt')

const legacy = new Set(
  fs.readFileSync(legacyFile, 'utf8')
    .split('\n')
    .map((line) => line.trim())
    .filter((line) => line && !line.startsWith('#'))
)

const violations = []
const stale = []

function walk(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (entry.name === 'node_modules' || entry.name.startsWith('.')) continue
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      walk(full)
      continue
    }
    if (!/\.(tsx|jsx)$/.test(entry.name)) continue
    const rel = path.relative(srcDir, full)
    const source = fs.readFileSync(full, 'utf8')
    const hasNumberInput = /\btype=["']number["']/.test(source)
    if (hasNumberInput && !legacy.has(rel)) {
      violations.push(rel)
    }
    if (!hasNumberInput && legacy.has(rel)) {
      stale.push(rel)
    }
  }
}

walk(srcDir)

if (stale.length > 0) {
  console.warn('[check-numbers] aviso: quitá de number-input-legacy.txt los archivos ya migrados:')
  for (const file of stale) console.warn(`  - ${file}`)
}

if (violations.length > 0) {
  console.error(`[check-numbers] ⛔ ${violations.length} archivo(s) con <input type="number"> fuera de la lista legacy:`)
  for (const file of violations) console.error(`  - ${file}`)
  console.error('  Usá frontend/src/components/NumberInput.tsx (regla UI-NUMBER-INPUT).')
  process.exit(1)
}

console.log(`[check-numbers] OK: sin nuevos <input type="number"> (legacy pendiente: ${legacy.size}).`)
