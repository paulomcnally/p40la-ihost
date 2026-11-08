// SPEC-129: service worker hand-written, sin dependencias.
//
// Estrategia:
// - Navegación: network-first con fallback al app shell cacheado.
// - /assets/*: cache-first (los nombres llevan hash).
// - /i18n/*, /icons/*, manifest: stale-while-revalidate.
// - /api/*, /webhooks/*, PDFs y todo lo que no sea GET: red, sin caché.
//
// Al cambiar assets o el shell, subir CACHE_VERSION para invalidar cachés.

const CACHE_PREFIX = 'p40la-'
const CACHE_VERSION = 'v2'
const SHELL_CACHE = `${CACHE_PREFIX}shell-${CACHE_VERSION}`
const ASSETS_CACHE = `${CACHE_PREFIX}assets-${CACHE_VERSION}`
const I18N_CACHE = `${CACHE_PREFIX}i18n-${CACHE_VERSION}`

const SHELL_URLS = [
  '/',
  '/index.html',
  '/manifest.webmanifest',
  '/icons/icon-192.png',
  '/icons/icon-512.png',
  '/icons/icon-maskable-512.png',
  '/icons/apple-touch-icon-180.png',
]

self.addEventListener('install', (event) => {
  event.waitUntil(precacheShell().then(() => self.skipWaiting()))
})

// Precarga el shell y los assets hasheados que referencia index.html, para
// que la primera instalación ya funcione offline (SPEC-129, REQ-015).
async function precacheShell() {
  const cache = await caches.open(SHELL_CACHE)
  await cache.addAll(SHELL_URLS)

  const index = await cache.match('/index.html')
  if (!index) return

  const html = await index.text()
  const assets = [...html.matchAll(/(?:src|href)="(\/assets\/[^"]+)"/g)].map((match) => match[1])
  await Promise.all(
    [...new Set(assets)].map(async (url) => {
      try {
        await cache.add(url)
      } catch {
        // Un asset faltante no debe romper la instalación del SW.
      }
    })
  )
}

self.addEventListener('activate', (event) => {
  const current = [SHELL_CACHE, ASSETS_CACHE, I18N_CACHE]
  event.waitUntil(
    caches
      .keys()
      .then((keys) =>
        Promise.all(
          keys
            .filter((key) => key.startsWith(CACHE_PREFIX) && !current.includes(key))
            .map((key) => caches.delete(key))
        )
      )
      .then(() => self.clients.claim())
  )
})

self.addEventListener('fetch', (event) => {
  const { request } = event
  if (request.method !== 'GET') return

  const url = new URL(request.url)
  if (url.origin !== self.location.origin) return
  if (url.pathname.startsWith('/api/') || url.pathname.startsWith('/webhooks/')) return
  if (url.pathname.endsWith('.pdf')) return

  if (request.mode === 'navigate') {
    event.respondWith(networkFirstNavigation(request))
    return
  }

  if (url.pathname.startsWith('/assets/')) {
    event.respondWith(cacheFirst(request, ASSETS_CACHE))
    return
  }

  if (url.pathname.startsWith('/i18n/')) {
    event.respondWith(staleWhileRevalidate(request, I18N_CACHE))
    return
  }

  if (url.pathname === '/manifest.webmanifest' || url.pathname.startsWith('/icons/')) {
    event.respondWith(staleWhileRevalidate(request, SHELL_CACHE))
  }
})

async function networkFirstNavigation(request) {
  try {
    const response = await fetch(request)
    if (response.ok) {
      const cache = await caches.open(SHELL_CACHE)
      await cache.put('/index.html', response.clone())
    }
    return response
  } catch {
    const cache = await caches.open(SHELL_CACHE)
    return (
      (await cache.match('/index.html')) ||
      (await cache.match('/')) ||
      new Response('Sin conexión', {
        status: 503,
        headers: { 'Content-Type': 'text/plain; charset=utf-8' },
      })
    )
  }
}

async function cacheFirst(request, cacheName) {
  const cache = await caches.open(cacheName)
  const cached = await cache.match(request)
  if (cached) return cached

  const response = await fetch(request)
  if (response.ok) await cache.put(request, response.clone())
  return response
}

async function staleWhileRevalidate(request, cacheName) {
  const cache = await caches.open(cacheName)
  const cached = await cache.match(request)

  const network = fetch(request)
    .then(async (response) => {
      if (response.ok) await cache.put(request, response.clone())
      return response
    })
    .catch(() => null)

  return cached || network || new Response('', { status: 504 })
}
