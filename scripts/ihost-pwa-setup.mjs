#!/usr/bin/env node
//
// SPEC-130: configura Tailscale Serve + Filebrowser en el iHost usando la API
// de la UI de Docker, sin `docker exec` y sin guardar credenciales.
//
// Requisitos:
//   - IHOST_TOKEN: token Bearer de una sesión autenticada de la UI del iHost.
//     Se obtiene en DevTools → Network → cualquier request a /api/v1/rest.
//   - Node.js 22+ (fetch global).
//
// Uso:
//   IHOST_TOKEN='...' node scripts/ihost-pwa-setup.mjs --dry-run
//   IHOST_TOKEN='...' node scripts/ihost-pwa-setup.mjs --apply
//
// Variables opcionales:
//   IHOST_URL=http://ihost.local
//   FILEBROWSER_PORT=8083
//   FILEBROWSER_PASSWORD=...   (si no se pasa, se lee de los logs del contenedor)
//   FILEBROWSER_ADDON_ID=9
//   TAILSCALE_ADDON_ID=21
//
// El script NO imprime el token, la password de Filebrowser ni claves de
// Tailscale. No borra volúmenes: solo detiene/elimina contenedores y los
// recrea. El volumen de estado de Tailscale se conserva.
//
import { readFileSync } from 'node:fs'

const IHOST_URL = (process.env.IHOST_URL || 'http://ihost.local').replace(/\/+$/, '')
const TOKEN = process.env.IHOST_TOKEN
const FILEBROWSER_PORT = process.env.FILEBROWSER_PORT || '8083'
const APPLY = process.argv.includes('--apply')
const DRY_RUN = !APPLY
const SERVE_FILE = new URL('../deploy/tailscale/serve.json', import.meta.url)

if (!TOKEN) {
  console.error('ERROR: falta IHOST_TOKEN. Obtenelo de la sesión autenticada de la UI del iHost.')
  process.exit(1)
}

const log = (...args) => console.log(...args)
const fail = (message) => {
  console.error(`ERROR: ${message}`)
  process.exit(1)
}
const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms))

async function api(path, method = 'GET', body) {
  const res = await fetch(`${IHOST_URL}/api/v1/rest${path}`, {
    method,
    headers: {
      Authorization: `Bearer ${TOKEN}`,
      ...(body ? { 'Content-Type': 'application/json' } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
    signal: AbortSignal.timeout(20000),
  })
  const text = await res.text()
  let parsed
  try {
    parsed = JSON.parse(text)
  } catch {
    parsed = text
  }
  if (!res.ok) throw new Error(`HTTP ${res.status} en ${method} ${path}: ${String(text).slice(0, 200)}`)
  return parsed
}

async function apiOk(path, method = 'GET', body) {
  const result = await api(path, method, body)
  if (result && result.error && result.error !== 0) {
    throw new Error(result.msg || `error ${result.error} en ${method} ${path}`)
  }
  return result
}

function envValue(envs, key) {
  const found = (envs || []).find((env) => String(env).startsWith(`${key}=`))
  return found ? String(found).slice(key.length + 1) : null
}

function envKeys(envs) {
  return (envs || []).map((env) => String(env).split('=')[0]).join(', ')
}

function volumeSummary(volumes) {
  return (volumes || [])
    .map((volume) => `${volume.volume_name || '(anónimo)'}:${volume.addon_mountpoint}`)
    .join(', ')
}

function findContainer(containers, name) {
  return containers.find((container) => container.container_name === name)
}

function findAddon(addons, name) {
  return addons.find((addon) => addon.name === name)
}

async function ensureVolume(name) {
  const volumes = await apiOk('/docker/volumes')
  const exists = (volumes.data || []).some((volume) => (volume.volume_name || volume.name) === name)
  if (exists) {
    log(`  volumen ${name}: ya existe`)
    return
  }
  if (DRY_RUN) {
    log(`  volumen ${name}: se crearía`)
    return
  }
  await apiOk('/docker/volumes', 'POST', { volume_name: name })
  log(`  volumen ${name}: creado`)
}

async function stopAndDelete(container, label) {
  if (!container) return
  if (DRY_RUN) {
    log(`  ${label}: se detendría y eliminaría (${container.container_id})`)
    return
  }
  try {
    await apiOk(`/docker/containers/${container.container_id}`, 'PUT', { state: 'stop' })
  } catch (error) {
    log(`  ${label}: stop devolvió ${error.message} (puede estar ya detenido)`)
  }
  await sleep(2000)
  await apiOk(`/docker/containers/${container.container_id}`, 'PUT', { state: 'delete' })
  log(`  ${label}: detenido y eliminado`)
}

async function getContainerLogs(containerId, tail = 300) {
  const result = await apiOk(`/docker/containers/${containerId}/log?tail=${tail}`)
  return result.data?.logs || []
}

async function filebrowserLogin(password) {
  const url = new URL(IHOST_URL)
  url.port = FILEBROWSER_PORT
  const res = await fetch(`${url.origin}/api/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: 'admin', password }),
    signal: AbortSignal.timeout(15000),
  })
  if (!res.ok) throw new Error(`Filebrowser login HTTP ${res.status}`)
  const text = await res.text()
  try {
    const parsed = JSON.parse(text)
    return parsed.token || parsed
  } catch {
    return text
  }
}

async function filebrowserCreateFile(token, content) {
  const url = new URL(IHOST_URL)
  url.port = FILEBROWSER_PORT
  const res = await fetch(`${url.origin}/api/resources/serve.json?override=true`, {
    method: 'POST',
    headers: { 'X-Auth': token, 'Content-Type': 'text/plain' },
    body: content,
    signal: AbortSignal.timeout(15000),
  })
  if (!res.ok) throw new Error(`Filebrowser create HTTP ${res.status}`)
  const verify = await fetch(`${url.origin}/api/resources/serve.json`, {
    headers: { 'X-Auth': token },
    signal: AbortSignal.timeout(15000),
  })
  if (!verify.ok) throw new Error(`Filebrowser verify HTTP ${verify.status}`)
  const verified = await verify.text()
  if (!verified.includes('TS_CERT_DOMAIN')) throw new Error('serve.json verificado no contiene TS_CERT_DOMAIN')
  log('  serve.json: creado y verificado')
}

async function main() {
  log(`[ihost-pwa-setup] ${DRY_RUN ? 'DRY-RUN (sin cambios)' : 'APLICANDO CAMBIOS'}`)
  log(`[ihost-pwa-setup] iHost: ${IHOST_URL}`)

  const [addonsResult, containersResult, volumesResult] = await Promise.all([
    apiOk('/docker/addons'),
    apiOk('/docker/containers'),
    apiOk('/docker/volumes'),
  ])
  const addons = addonsResult.data || []
  const containers = containersResult.data || []
  const volumes = (volumesResult.data || []).map((volume) => volume.volume_name || volume.name)

  const filebrowserAddon = findAddon(addons, 'filebrowser/filebrowser')
  const tailscaleAddon = findAddon(addons, 'tailscale/tailscale')
  if (!filebrowserAddon || !tailscaleAddon) fail('No se encontraron los add-ons filebrowser/filebrowser y tailscale/tailscale')

  const filebrowser = findContainer(containers, 'filebrowser/filebrowser')
  const tailscale = findContainer(containers, 'tailscale/tailscale')
  if (!tailscale) fail('No existe el contenedor tailscale/tailscale')

  const tailscaleStateDir = envValue(tailscale.environments, 'TS_STATE_DIR')
  if (!tailscaleStateDir) fail('El contenedor Tailscale no tiene TS_STATE_DIR')

  // Se preservan todas las variables existentes (incluido TS_HOSTNAME si
  // estuviera) y solo se agrega TS_SERVE_CONFIG. No se imprimen los valores.
  const tailscaleEnvs = (tailscale.environments || []).filter(
    (env) => !String(env).startsWith('TS_SERVE_CONFIG=')
  )
  if (!tailscaleEnvs.some((env) => String(env).startsWith('PATH='))) {
    tailscaleEnvs.push('PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin')
  }
  tailscaleEnvs.push('TS_SERVE_CONFIG=/config/serve.json')

  const stateVolume = (tailscale.volume_list || []).find(
    (volume) => volume.addon_mountpoint === tailscaleStateDir
  ) || (tailscale.volume_list || [])[0]
  if (!stateVolume?.volume_name) fail('No se pudo identificar el volumen de estado de Tailscale')

  log('\n[estado actual]')
  log(`  Filebrowser addon_id=${filebrowserAddon.id} container=${filebrowser?.container_id || 'ausente'}`)
  log(`  Filebrowser puerto=${(filebrowser?.port_list || []).join(',') || '—'} volúmenes=${volumeSummary(filebrowser?.volume_list)}`)
  log(`  Filebrowser env keys=${envKeys(filebrowser?.environments)}`)
  log(`  Tailscale addon_id=${tailscaleAddon.id} container=${tailscale.container_id}`)
  log(`  Tailscale network=${tailscale.network} volúmenes=${volumeSummary(tailscale.volume_list)}`)
  log(`  Tailscale env keys=${envKeys(tailscale.environments)}`)
  log(`  Volumen de estado detectado: ${stateVolume.volume_name} → ${stateVolume.addon_mountpoint}`)

  log('\n[plan]')
  log('  1. Crear/verificar volúmenes: tailscale-config, filebrowser-db, filebrowser-config')
  await ensureVolume('tailscale-config')
  await ensureVolume('filebrowser-db')
  await ensureVolume('filebrowser-config')

  log('  2. Recrear Filebrowser en /srv con tailscale-config, /database con filebrowser-db, /config con filebrowser-config')
  await stopAndDelete(filebrowser, 'Filebrowser')

  if (!DRY_RUN) {
    const filebrowserEnvs = (filebrowser?.environments || []).filter(
      (env) => !/^(UID|GID)=/.test(String(env))
    )
    if (!filebrowserEnvs.some((env) => String(env).startsWith('PATH='))) {
      filebrowserEnvs.push('PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin')
    }
    filebrowserEnvs.push('UID=0', 'GID=0')

    const filebrowserPayload = {
      addon_id: filebrowserAddon.id,
      dev_snd: false,
      usb_devices: [],
      port_list: [{ host_port: FILEBROWSER_PORT, container_port: '80' }],
      network: 'bridge',
      volume_list: [
        { volume_name: 'tailscale-config', mountpoint: '/srv' },
        { volume_name: 'filebrowser-db', mountpoint: '/database' },
        { volume_name: 'filebrowser-config', mountpoint: '/config' },
      ],
      environments: filebrowserEnvs,
    }
    const created = await apiOk('/docker/containers', 'POST', filebrowserPayload)
    log(`  Filebrowser: creado ${created.data?.container_id}`)
    await sleep(5000)

    const currentContainers = await apiOk('/docker/containers')
    const newFilebrowser = findContainer(currentContainers.data || [], 'filebrowser/filebrowser')
    if (!newFilebrowser) fail('Filebrowser no aparece después de crearlo')

    let password = process.env.FILEBROWSER_PASSWORD
    if (!password) {
      for (let i = 0; i < 8 && !password; i++) {
        const logs = await getContainerLogs(newFilebrowser.container_id, 200)
        const line = logs.find((entry) => entry.includes('randomly generated password:'))
        if (line) password = line.split('randomly generated password:')[1].trim().split(/\s/)[0]
        else await sleep(1500)
      }
    }
    if (!password) fail('No se pudo obtener la password de Filebrowser; pasá FILEBROWSER_PASSWORD')

    const filebrowserToken = await filebrowserLogin(password)
    await filebrowserCreateFile(filebrowserToken, readFileSync(SERVE_FILE, 'utf8'))
    await apiOk(`/docker/containers/${newFilebrowser.container_id}`, 'PUT', { state: 'stop' })
    log('  Filebrowser: detenido después de crear serve.json')
  }

  log('  3. Recrear Tailscale con el volumen de estado + tailscale-config y TS_SERVE_CONFIG')
  await stopAndDelete(tailscale, 'Tailscale')

  if (!DRY_RUN) {
    const tailscalePayload = {
      addon_id: tailscaleAddon.id,
      dev_snd: false,
      usb_devices: [],
      port_list: [],
      network: 'host',
      volume_list: [
        { volume_name: stateVolume.volume_name, mountpoint: tailscaleStateDir },
        { volume_name: 'tailscale-config', mountpoint: '/config' },
      ],
      environments: tailscaleEnvs,
    }
    const created = await apiOk('/docker/containers', 'POST', tailscalePayload)
    log(`  Tailscale: creado ${created.data?.container_id}`)
    await sleep(12000)

    const currentContainers = await apiOk('/docker/containers')
    const newTailscale = findContainer(currentContainers.data || [], 'tailscale/tailscale')
    if (!newTailscale) fail('Tailscale no aparece después de crearlo')

    const logs = await getContainerLogs(newTailscale.container_id, 300)
    const relevant = logs.filter((line) => /serve|cert|tls|startup|error/i.test(line))
    for (const line of relevant.slice(-20)) {
      log(
        `  log: ${line
          .replace(/[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+/g, '***@***')
          .replace(/(nodekey|authkey|tskey-)[^ ]+/gi, '$1***')
          .slice(0, 240)}`
      )
    }
    if (logs.some((line) => line.includes('not able to issue TLS certs'))) {
      log('\n  ⚠️  Falta habilitar HTTPS Certificates en la tailnet:')
      log('     https://login.tailscale.com/admin/dns → HTTPS Certificates → Enable')
      log('     Después reiniciá el add-on Tailscale.')
    } else {
      log('\n  Tailscale recreado. Verificá https://<nodo>.<tailnet>.ts.net con TLS activo.')
    }
  }

  log('\n[fin]')
  if (DRY_RUN) log('No se cambió nada. Ejecutá con --apply para aplicar.')
}

main().catch((error) => {
  console.error(`ERROR: ${error.message}`)
  process.exit(1)
})
