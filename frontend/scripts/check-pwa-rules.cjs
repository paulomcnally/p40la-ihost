#!/usr/bin/env node
//
// SPEC-130: guardrail de PWA.
//
// Valida que el manifest, los íconos y el service worker existan y cumplan
// los requisitos mínimos de instalabilidad de Chrome Android. Se ejecuta en
// `npm run build` (y con `npm run check:pwa`).
//
const fs = require('fs')
const path = require('path')

const publicDir = path.join(__dirname, '..', 'public')
const manifestPath = path.join(publicDir, 'manifest.webmanifest')
const swPath = path.join(publicDir, 'sw.js')
const indexPath = path.join(__dirname, '..', 'index.html')
const errors = []

function fail(message) {
  errors.push(message)
}

function readPngSize(file) {
  if (!fs.existsSync(file)) {
    fail(`falta el ícono ${path.relative(publicDir, file)}`)
    return null
  }
  const buf = fs.readFileSync(file)
  const signature = buf.subarray(0, 8).toString('hex')
  if (signature !== '89504e470d0a1a0a') {
    fail(`${path.relative(publicDir, file)} no es un PNG válido`)
    return null
  }
  return { width: buf.readUInt32BE(16), height: buf.readUInt32BE(20) }
}

if (!fs.existsSync(manifestPath)) {
  fail('falta frontend/public/manifest.webmanifest')
} else {
  let manifest
  try {
    manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'))
  } catch (err) {
    fail(`manifest.webmanifest no es JSON válido: ${err.message}`)
  }

  if (manifest) {
    if (!manifest.name && !manifest.short_name) fail('el manifest necesita name o short_name')
    if (manifest.id !== '/') fail('el manifest debe declarar id: "/"')
    if (manifest.start_url !== '/') fail('el manifest debe declarar start_url: "/"')
    if (manifest.scope !== '/') fail('el manifest debe declarar scope: "/"')
    if (!['standalone', 'fullscreen', 'minimal-ui'].includes(manifest.display)) {
      fail('display debe ser standalone, fullscreen o minimal-ui')
    }
    if (manifest.prefer_related_applications !== false) {
      fail('prefer_related_applications debe ser false')
    }

    const icons = Array.isArray(manifest.icons) ? manifest.icons : []
    const has = (size, purpose) =>
      icons.some((icon) => {
        const purposes = String(icon.purpose || 'any').split(/\s+/)
        return icon.sizes === size && purposes.includes(purpose)
      })

    if (!has('192x192', 'any')) fail('falta un ícono 192x192 con purpose "any"')
    if (!has('512x512', 'any')) fail('falta un ícono 512x512 con purpose "any"')
    if (!has('512x512', 'maskable')) fail('falta un ícono 512x512 con purpose "maskable"')

    for (const icon of icons) {
      const file = path.join(publicDir, icon.src.replace(/^\//, ''))
      const size = readPngSize(file)
      if (!size) continue
      const [expectedWidth, expectedHeight] = String(icon.sizes).split('x').map(Number)
      if (size.width !== expectedWidth || size.height !== expectedHeight) {
        fail(
          `${icon.src} mide ${size.width}x${size.height}, el manifest declara ${icon.sizes}`
        )
      }
    }
  }
}

if (!fs.existsSync(swPath)) {
  fail('falta frontend/public/sw.js')
} else {
  const sw = fs.readFileSync(swPath, 'utf8')
  if (!sw.includes("addEventListener('fetch'")) {
    fail('sw.js no registra un handler de fetch')
  }
}

const appleIcon = path.join(publicDir, 'icons', 'apple-touch-icon-180.png')
const appleSize = readPngSize(appleIcon)
if (appleSize && (appleSize.width !== 180 || appleSize.height !== 180)) {
  fail(`apple-touch-icon-180.png mide ${appleSize.width}x${appleSize.height}`)
}

if (!fs.existsSync(indexPath)) {
  fail('falta frontend/index.html')
} else {
  const html = fs.readFileSync(indexPath, 'utf8')
  if (!/rel="manifest"/.test(html)) fail('index.html no declara <link rel="manifest">')
  if (!/apple-touch-icon/.test(html)) fail('index.html no declara apple-touch-icon')
  if (!/viewport-fit=cover/.test(html)) fail('index.html no declara viewport-fit=cover')
}

if (errors.length > 0) {
  console.error('\n[check-pwa-rules] ERROR: la PWA no cumple los requisitos mínimos.\n')
  for (const error of errors) console.error(`  - ${error}`)
  console.error('\nVer SPEC-130 y docs/ihost.md.\n')
  process.exit(1)
}

console.log('[check-pwa-rules] OK: manifest, íconos, service worker y meta tags presentes.')
