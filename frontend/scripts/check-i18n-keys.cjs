#!/usr/bin/env node
// Guardrail UI-I18N (SPEC-140): toda clave usada por t('...') debe existir como
// string en es.json y en.json, y no puede apuntar a un objeto padre.
// También verifica paridad de claves entre ambos idiomas.

const fs = require('fs')
const path = require('path')

const root = path.resolve(__dirname, '..')
const i18nDir = path.join(root, 'public', 'i18n')
const srcDir = path.join(root, 'src')

function load(name) {
  return JSON.parse(fs.readFileSync(path.join(i18nDir, `${name}.json`), 'utf8'))
}

const dictionaries = { es: load('es'), en: load('en') }
const errors = []

function resolve(node, key) {
  return key.split('.').reduce((acc, part) => {
    if (acc == null || typeof acc !== 'object') return undefined
    return acc[part]
  }, node)
}

function flatten(node, prefix = '', out = {}) {
  for (const [key, value] of Object.entries(node)) {
    const full = prefix ? `${prefix}.${key}` : key
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      flatten(value, full, out)
    } else if (typeof value === 'string') {
      out[full] = value
    }
  }
  return out
}

// 1. Paridad es/en
const flatEs = flatten(dictionaries.es)
const flatEn = flatten(dictionaries.en)
for (const key of Object.keys(flatEs)) {
  if (!(key in flatEn)) errors.push(`falta en en.json: ${key}`)
}
for (const key of Object.keys(flatEn)) {
  if (!(key in flatEs)) errors.push(`falta en es.json: ${key}`)
}

// 2. Claves usadas en el código
function walk(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (entry.name === 'node_modules' || entry.name.startsWith('.')) continue
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) walk(full)
    else if (/\.(ts|tsx)$/.test(entry.name)) checkFile(full)
  }
}

function lineOf(source, index) {
  return source.slice(0, index).split('\n').length
}

function checkKey(key, file, source, index) {
  for (const lang of ['es', 'en']) {
    const value = resolve(dictionaries[lang], key)
    if (typeof value !== 'string') {
      const kind = value === undefined ? 'no existe' : 'apunta a un objeto'
      errors.push(`${path.relative(root, file)}:${lineOf(source, index)}: "${key}" ${kind} en ${lang}.json`)
      return
    }
  }
}

function checkFile(file) {
  const source = fs.readFileSync(file, 'utf8')

  const staticRe = /(?<![A-Za-z0-9_$])t\(\s*(['"])([^'"]+)\1\s*(?:,[^)]*)?\)/g
  let match
  while ((match = staticRe.exec(source)) !== null) {
    checkKey(match[2], file, source, match.index)
  }

  const dynamicRe = /(?<![A-Za-z0-9_$])t\(\s*`([^`]+)`\s*\)/g
  while ((match = dynamicRe.exec(source)) !== null) {
    const raw = match[1]
    if (!raw.includes('${')) {
      checkKey(raw, file, source, match.index)
      continue
    }
    let prefix = raw.slice(0, raw.indexOf('${')).replace(/\.$/, '')
    if (!prefix) continue
    if (typeof resolve(dictionaries.es, prefix) !== 'object') {
      const cut = prefix.lastIndexOf('.')
      prefix = cut > 0 ? prefix.slice(0, cut) : prefix
      if (typeof resolve(dictionaries.es, prefix) !== 'object') {
        errors.push(`${path.relative(root, file)}:${lineOf(source, match.index)}: prefijo dinámico "${prefix}" no existe en es.json`)
      }
    }
  }
}

walk(srcDir)

if (errors.length > 0) {
  console.error(`[check-i18n] ⛔ ${errors.length} problema(s):`)
  for (const error of errors.slice(0, 40)) console.error(`  - ${error}`)
  if (errors.length > 40) console.error(`  ... y ${errors.length - 40} más`)
  process.exit(1)
}

console.log(`[check-i18n] OK: ${Object.keys(flatEs).length} claves en es/en y todas las usadas existen.`)
