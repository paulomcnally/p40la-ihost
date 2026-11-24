---
title: "Despliegue del MCP remoto en el iHost con nodo Tailscale dedicado"
id: "SPEC-148"
status: "in_progress"
author: "opencode"
created: "2026-11-23"
updated: "2026-11-23"
github_issue: 151
---

# Despliegue del MCP remoto en el iHost con nodo Tailscale dedicado

**ID**: SPEC-148  
**Estado**: in_progress  
**Autor**: opencode  
**Creado**: 2026-11-23  
**Actualizado**: 2026-11-23

---

## 1. Resumen Ejecutivo

SPEC-147 liberó el servidor MCP remoto (transporte HTTP, OAuth 2.1, auditoría y doble gate de escritura), pero la topología validada fue la laptop. El requisito real es que el MCP corra **24/7 en el iHost**, sin depender de una laptop encendida.

El iHost ya tiene un nodo Tailscale (`ihost.<tailnet>.ts.net`) que publica el dashboard del 8088 con **Serve** en 443, solo dentro de la tailnet. Tailscale **no permite Serve y Funnel en el mismo puerto**: activar Funnel en ese nodo volvería público todo el 443, incluido el dashboard. Por eso esta spec agrega un **segundo nodo Tailscale dedicado** (`p40la-mcp.<tailnet>.ts.net`) con su propio estado y su propio Funnel, y deja el nodo actual **intacto**.

El resultado esperado: `https://p40la-mcp.<tailnet>.ts.net/mcp` disponible desde claude.ai, con OAuth, escrituras habilitadas y auditadas, corriendo en el iHost (ARM `armv7l`, sin `/dev/net/tun`) y sin afectar el acceso actual al dashboard.

---

## 2. Cumplimiento de reglas

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Se reutilizan las imágenes Go del MCP y `tailscale/tailscale`; sin dependencias nuevas | `CA-ARCH-STACK` |
| `ARCH-IHOST` | Contenedores livianos, estado en volúmenes, sin tun, presupuesto de RAM medido | `CA-ARCH-IHOST` |
| `ARCH-GO` | El binario del MCP no cambia; se publica la imagen ya construida | `CA-ARCH-GO` |
| `DATA-MIGRATIONS` | N/A: sin migraciones ni cambios de esquema | — |
| `DATA-NO-DROP` / `DATA-PRODCOPY` | No se toca `app.db`; las pruebas de datos usan copia en `/tmp` | `CA-DATA-PRODCOPY` |
| `SEC-NO-PII` | Spec y assets con placeholders; secretos solo en archivos `0600` del iHost | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | `demo-node.example.ts.net`, `192.0.2.10`, `demo-api-key` | `CA-SEC-SYNTHETIC` |
| `SEC-LOGS` | Auditoría sin argumentos; logs sin tokens ni API key | `CA-SEC-LOGS` |
| `SEC-INCIDENT` | Si se filtra la auth key o la API key: revocar, rotar y documentar | `CA-SEC-INCIDENT` |
| `TEST-LOCAL` | Build/tests del MCP verdes antes de publicar la imagen | `CA-TEST-LOCAL` |
| `TEST-SERVER` | `./scripts/check-server.sh` y `/health` del iHost antes de avisar | `CA-TEST-SERVER` |
| `TEST-USER-FLOW` | Validación del usuario en claude.ai antes de `pending_release` | `CA-TEST-USER-FLOW` |
| `DELIV-WORKTREE` | Trabajo en `../p40la-ihost-spec-148`, rama `feature/SPEC-148` | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | Deploy, Funnel y cambios en el iHost aprobados por el usuario | `CA-DELIV-CONFIRM` |
| `DELIV-DOCKER` | Imagen Docker Hub multi-arch: `linux/amd64,linux/arm/v7,linux/arm64` | `CA-DELIV-DOCKER` |
| `DELIV-MCP-DOCS` | El mismo commit actualiza `docs/mcp/` (despliegue iHost y Funnel) | `CA-DELIV-MCP-DOCS` |
| `DELIV-MCP-DOCS-SUBAGENT` | La redacción de docs se delega a un subagente | `CA-DELIV-MCP-DOCS-SUBAGENT` |
| `DELIV-DEPLOY` | Deploy al iHost aprobado; health check posterior | `CA-DELIV-DEPLOY` |
| `UI-*` / `UX-*` | N/A: no hay cambios de frontend | — |

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001 — Imagen en Docker Hub**: publicar `paulomcnally/p40la-mcp:0.1.0` (y `latest`) con manifests `linux/amd64`, `linux/arm/v7` y `linux/arm64`. El iHost usa `armv7l`.
2. **REQ-002 — MCP en el iHost**: contenedor `p40la-mcp` con `network_mode: host`, bind `127.0.0.1:8090`/`8091`, volumen persistente `/data`, `restart: unless-stopped`.
3. **REQ-003 — Tailscale dedicado**: contenedor `tailscale-mcp` con `TS_USERSPACE=true`, `TS_HOSTNAME=p40la-mcp`, estado y config propios, sin compartir volúmenes con el Tailscale actual.
4. **REQ-004 — Funnel solo del MCP**: `p40la-mcp.<tailnet>.ts.net:443` → `http://127.0.0.1:8090`, con `AllowFunnel: true`. Sin rutas al dashboard ni al listener admin.
5. **REQ-005 — Dashboard intacto**: no modificar el contenedor Tailscale existente, su `serve.json`, sus volúmenes ni su configuración. El acceso `https://ihost.<tailnet>.ts.net` debe seguir funcionando y seguir siendo privado (sin Funnel).
6. **REQ-006 — API key**: generar una API key global en el iHost con la app `0.5.17` o superior y guardarla en `/etc/p40la-mcp.env` con permisos `0600`.
7. **REQ-007 — OAuth**: `P40LA_MCP_PUBLIC_URL=https://p40la-mcp.<tailnet>.ts.net`, hash bcrypt de la contraseña de administración, `P40LA_MCP_TRANSPORT=http`.
8. **REQ-008 — Escrituras**: `P40LA_MCP_ALLOW_WRITES=true` y `P40LA_MCP_REMOTE_WRITES=true`; auditoría y rate limit activos.
9. **REQ-009 — Persistencia y arranque**: volúmenes para `/data` (OAuth/auditoría) y para el estado de Tailscale; ambos contenedores arrancan solos tras un reinicio del iHost.
10. **REQ-010 — Publicación**: verificar DNS público con **A IPv4**, TLS válido, discovery OAuth y `401` sin token.
11. **REQ-011 — Conector claude.ai**: agregar el conector remoto, completar OAuth y ejecutar una tool de lectura y una de escritura.
12. **REQ-012 — Rollback**: detener ambos contenedores y resetear el Funnel del nodo dedicado sin afectar el nodo del dashboard.

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-013 — Presupuesto de RAM**: medir RSS de ambos contenedores en el iHost; objetivo < 120 MB combinados y sin presión de memoria.
2. **REQ-014 — Runbook iHost**: documentar el alta por UI/API, variables, volúmenes, red, verificación y rollback.
3. **REQ-015 — Health check**: verificar `/healthz` del listener admin desde el propio iHost (UI/consola del contenedor) y `docker inspect` para el estado.
4. **REQ-016 — Backup**: respaldar `/data/oauth.json` y la config de Tailscale antes de actualizar imágenes.
5. **REQ-017 — Actualización**: documentar cómo actualizar la imagen del MCP y la del nodo Tailscale sin perder estado.

### 3.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-018 — Tag de Tailscale**: usar `tag:p40la-mcp` y ACLs que limiten Funnel solo a ese tag.
2. **REQ-019 — Monitoreo**: alerta si el contenedor MCP o el Funnel se caen (Uptime Kuma/health externo).
3. **REQ-020 — Rotación programada**: rotar la API key y la contraseña OAuth cada N meses.

### 3.4 Requerimientos No Funcionales

- **Disponibilidad**: 24/7; `restart: unless-stopped`; el iHost debe arrancar ambos contenedores.
- **Rendimiento**: `initialize` < 500 ms desde Internet; RSS combinado < 120 MB.
- **Seguridad**: endpoint público solo con OAuth; admin/app/dashboard no públicos; secretos `0600`.
- **Compatibilidad**: `armv7l`, sin `/dev/net/tun`, Docker del iHost (UI/API).
- **Privacidad**: los datos de las tools viajan a Anthropic; el usuario lo acepta.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto medido en el iHost

| Dato | Valor |
|---|---|
| `uname -m` | `armv7l` (userland 32-bit) |
| CPU / RAM | 4 cores · 3.7 GB totales · ~2.25 GB disponibles |
| Disco | 105 GB libres |
| `/dev/net/tun` | no existe |
| Docker CLI en SSH | no disponible (solo `wget`) |
| Nodo Tailscale actual | `ihost.<tailnet>.ts.net`, Serve 443 → `127.0.0.1:8088`, `AllowFunnel: false` |

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| A. Nodo Tailscale dedicado en el iHost | Dashboard intacto; Funnel solo MCP; siempre disponible | +50-80 MB RAM; segundo hostname | ✅ Seleccionada |
| B. Funnel en el nodo actual | Sin contenedor extra | Vuelve público todo el 443, incluido el dashboard | ❌ Rechazada |
| C. MCP en laptop + Funnel | Sin impacto en iHost | Depende de laptop 24/7 | ❌ Rechazada por requisito |
| D. Cloudflare Tunnel | URL propia y auth | Dependencia/cloud extra; no pedido | ❌ Rechazada |
| E. Serve tailnet-only | Sin exposición | claude.ai no alcanza la tailnet | ❌ No cumple |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001 — Dos nodos Tailscale, uno por servicio**
- **Contexto**: Funnel y Serve no comparten puerto; el dashboard usa 443 con Serve.
- **Decisión**: nodo `p40la-mcp` dedicado, con Funnel 443 → MCP; nodo `ihost` sin cambios.
- **Consecuencias**: un contenedor Tailscale extra y un hostname nuevo; aislamiento total entre dashboard y MCP.

**ADR-002 — `TS_USERSPACE=true`**
- **Contexto**: el iHost no tiene `/dev/net/tun`.
- **Decisión**: Tailscale en modo userspace, con su propio `TS_STATE_DIR` y `TS_SOCKET`.
- **Consecuencias**: no requiere privilegios de TUN; el rendimiento es suficiente para un MCP de bajo tráfico.

**ADR-003 — MCP en red host, Tailscale dedicado en red host (primario)**
- **Contexto**: el MCP necesita `127.0.0.1:8088` para la API y el Tailscale necesita alcanzar `127.0.0.1:8090`.
- **Decisión**: ambos con `network_mode: host`. El Tailscale en userspace no debería bindear el 443 del host; se valida que el 443 del dashboard siga intacto.
- **Fallback**: si el segundo Tailscale choca con el 443, detenerlo y evaluar un proxy en el host; el bridge simple no alcanza porque el MCP solo escucha loopback (ver `docs/mcp/ihost.md`).
- **Consecuencias**: configuración simple; el fallback requiere un proxy adicional en el host o cambiar el modo de red.

**ADR-004 — Imagen propia `0.1.0`**
- **Contexto**: el MCP tiene ciclo de vida distinto del add-on web.
- **Decisión**: publicar `paulomcnally/p40la-mcp:0.1.0` y `latest` en Docker Hub; no incluir el binario en la imagen del add-on.
- **Consecuencias**: actualizaciones independientes; el iHost solo cambia el tag del contenedor MCP.

---

## 5. Diseño Técnico

### 5.1 Diagrama

```
claude.ai
   │ HTTPS 443 + OAuth
   ▼
p40la-mcp.<tailnet>.ts.net        ← nodo Tailscale dedicado (contenedor tailscale-mcp)
   │ Funnel 443 → 127.0.0.1:8090
   ▼
p40la-mcp (contenedor, red host)
   │ HTTP + X-API-Key
   ▼
p40la-ihost :8088 (add-on actual)

ihost.<tailnet>.ts.net            ← nodo Tailscale actual (NO se toca)
   │ Serve 443 → 127.0.0.1:8088   (solo tailnet)
   ▼
dashboard
```

### 5.2 Contenedores

#### 5.2.1 `p40la-mcp`
- **Imagen**: `paulomcnally/p40la-mcp:0.1.0` (arm/v7).
- **Red**: `host`.
- **Volumen**: `p40la-mcp-data:/data`.
- **Env file**: `/etc/p40la-mcp.env` (`0600`).
- **Restart**: `unless-stopped`.
- **Puertos**: `127.0.0.1:8090` (público vía Funnel) y `127.0.0.1:8091` (admin local).

#### 5.2.2 `tailscale-mcp`
- **Imagen**: `tailscale/tailscale:latest` (arm/v7).
- **Red**: `host` (primario).
- **Volúmenes**: `tailscale-mcp-state:/var/lib/tailscale`, `tailscale-mcp-config:/config`.
- **Env**: `TS_AUTHKEY`, `TS_HOSTNAME=p40la-mcp`, `TS_USERSPACE=true`, `TS_STATE_DIR=/var/lib/tailscale`, `TS_SERVE_CONFIG=/config/serve.json`.
- **Restart**: `unless-stopped`.

### 5.3 `serve.json` del nodo dedicado

```json
{
  "TCP": { "443": { "HTTPS": true } },
  "Web": {
    "${TS_CERT_DOMAIN}:443": {
      "Handlers": {
        "/": { "Proxy": "http://127.0.0.1:8090" }
      }
    }
  },
  "AllowFunnel": { "${TS_CERT_DOMAIN}:443": true }
}
```

- `${TS_CERT_DOMAIN}` se sustituye por `p40la-mcp.<tailnet>.ts.net`.
- El fallback bridge requiere un proxy en el host; no apuntes el `serve.json` a `host.docker.internal:8090` sin ese proxy.

### 5.4 Variables del MCP en el iHost

```dotenv
P40LA_BASE_URL=http://127.0.0.1:8088
P40LA_API_KEY=demo-api-key
P40LA_MCP_TRANSPORT=http
P40LA_MCP_LISTEN=127.0.0.1:8090
P40LA_MCP_ADMIN_LISTEN=127.0.0.1:8091
P40LA_MCP_PUBLIC_URL=https://p40la-mcp.<tailnet>.ts.net
P40LA_MCP_OAUTH_PASSWORD_HASH=$2a$12$demo
P40LA_MCP_STATE_DIR=/data
P40LA_MCP_ALLOW_WRITES=true
P40LA_MCP_REMOTE_WRITES=true
P40LA_MCP_RATE_LIMIT=60
P40LA_MCP_AUDIT_MAX_MB=5
P40LA_TIMEZONE=America/Managua
```

### 5.5 Publicación de la imagen

- Workflow `.github/workflows/mcp-publish.yml` disparado por tag `mcp-v*` o `workflow_dispatch`.
- Reutiliza `DOCKERHUB_USERNAME`/`DOCKERHUB_TOKEN`; pasa `GH_TOKEN` como secreto de BuildKit.
- Alternativa local: `IMAGE=paulomcnally/p40la-mcp TAG=0.1.0 GH_TOKEN=... ./scripts/build-mcp-image.sh`.

### 5.6 Alta en el iHost

- El iHost no expone Docker CLI por SSH; se usa la UI de Docker (`#/docker/detail`) o la API `http://ihost.local/api/v1/rest` con un token de sesión.
- Orden: crear volúmenes → crear `p40la-mcp` → crear `tailscale-mcp` → verificar Funnel → agregar conector.
- Los secretos se cargan en el env file del contenedor, nunca en la spec ni en el repo.

### 5.7 Dependencias

- Imagen del MCP publicada (SPEC-147).
- App `0.5.17+` desplegada para la API key.
- Auth key de Tailscale con permiso de Funnel.
- HTTPS Certificates y MagicDNS habilitados en la tailnet.

---

## 6. Criterios de Aceptación

### 6.1 Funcionales

- [ ] **CA-IHOST-MCP-001**: `docker buildx imagetools inspect paulomcnally/p40la-mcp:0.1.0` muestra `amd64`, `arm/v7` y `arm64`.
- [ ] **CA-IHOST-MCP-002**: el contenedor `p40la-mcp` corre en el iHost (`armv7l`) con `restart: unless-stopped`.
- [ ] **CA-IHOST-MCP-003**: `/healthz` del listener admin responde `{"status":"ok"}` desde el iHost.
- [ ] **CA-IHOST-MCP-004**: el dashboard `https://ihost.<tailnet>.ts.net` sigue funcionando y **no** es público (Funnel sigue deshabilitado en ese nodo).
- [ ] **CA-IHOST-MCP-005**: `p40la-mcp.<tailnet>.ts.net` resuelve con **A IPv4** público y certificado válido.
- [ ] **CA-IHOST-MCP-006**: discovery OAuth responde y `/mcp` sin token devuelve `401` con `WWW-Authenticate`.
- [ ] **CA-IHOST-MCP-007**: el conector de claude.ai completa OAuth y lista las tools.
- [ ] **CA-IHOST-MCP-008**: una tool de lectura funciona y una de escritura con ambos flags se ejecuta y queda auditada.
- [ ] **CA-IHOST-MCP-009**: `/healthz`, `/admin/revoke`, el dashboard y el puerto 8088 no son alcanzables desde Internet.
- [ ] **CA-IHOST-MCP-010**: tras reiniciar el iHost, ambos contenedores vuelven a arrancar solos.
- [ ] **CA-IHOST-MCP-011**: el rollback (`docker stop` + reset del Funnel dedicado) deja el dashboard intacto.
- [ ] **CA-IHOST-MCP-012**: RSS combinado medido < 120 MB.

### 6.2 No funcionales

- [ ] **CA-NF-001**: `go build ./...` y `go test ./...` verdes.
- [ ] **CA-NF-002**: imagen multi-arch verificada; sin dependencias nuevas.
- [ ] **CA-NF-003**: secretos solo en archivos `0600` del iHost; repo sin tokens ni auth keys.
- [ ] **CA-NF-004**: docs `docs/mcp/` actualizadas con el runbook de iHost.
- [ ] **CA-NF-005**: el nodo Tailscale actual no se modifica (diff de contenedor/volúmenes vacío).

### 6.3 Testing

- **Local**: build/tests del MCP; build multi-arch; smoke test del contenedor.
- **Docker Hub**: manifests y pull en arm/v7.
- **iHost**: logs de ambos contenedores; `/healthz`; Funnel status; DNS público.
- **claude.ai**: conector, OAuth, tools de lectura y escritura.
- **Rollback**: detener contenedores, reset de Funnel, verificar dashboard.

---

## 7. Plan de Implementación

### 7.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Workflow de publicación + tag `mcp-v0.1.0` + imagen en Docker Hub | 0.5 día | SPEC-147 |
| 2 | Assets de despliegue (`deploy/mcp/`), env example y runbook | 0.5 día | Fase 1 |
| 3 | Alta de contenedores en el iHost y validación del 443/dashboard | 0.5 día | App 0.5.17 + auth key |
| 4 | API key, OAuth, Funnel y verificación pública | 0.5 día | Fase 3 |
| 5 | Conector claude.ai + prueba de lectura/escritura | 0.5 día | Fase 4 |
| 6 | Medición de RSS, documentación y release | 0.5 día | Fase 5 |

### 7.2 Milestones

1. **MVP**: MCP en iHost accesible por Funnel con OAuth y lectura.
2. **V1.0**: escrituras habilitadas, auditoría, reinicio automático y conector validado.

---

## 8. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| El segundo Tailscale choca con el 443 del dashboard | Media | Alto | Validar primero; si choca, detenerlo y montar proxy en el host (ver `docs/mcp/ihost.md`) |
| Pérdida del acceso actual al dashboard | Baja | Alto | No tocar el contenedor/volúmenes/serve.json actuales; verificar antes/después |
| RAM insuficiente | Baja | Medio | 2.25 GB disponibles; medir RSS; límites de contenedor |
| Auth key de Tailscale filtrada | Baja | Alto | Tag dedicado, key de un solo uso, rotación, `SEC-INCIDENT` |
| DNS público sin A IPv4 | Media | Alto | Verificar con resolutor público; esperar propagación; reintentar |
| Docker del iHost sin `host-gateway` | Baja | Medio | Fallback a IP LAN o red compartida del MCP |
| Imagen arm/v7 no publicada | Baja | Alto | Workflow multi-arch + verificación de manifests |
| Escrituras desde un endpoint público | Media | Alto | OAuth + doble flag + auditoría + rate limit; revocación |
| API key global sin scopes | Alta | Medio | Aceptar en V1; scopes en P2 |
| Actualización de imagen pierde estado | Baja | Medio | Volúmenes persistentes + backup de `/data` y Tailscale |

---

## 9. Notas y Referencias

- SPEC-147: MCP remoto (HTTP, OAuth, auditoría, doble gate).
- `docs/mcp/funnel.md`: conflicto Serve/Funnel y nodo dedicado.
- `docs/ihost.md` y `docs/ihost-pwa-tailscale-setup.md`: Tailscale actual del dashboard.
- Tailscale Funnel: 443/8443/10000; Claude solo 443; IPv4-only.
- `Dockerfile.mcp`, `scripts/build-mcp-image.sh`, `deploy/tailscale/serve-funnel.example.json`.
- Reglas: `docs/rules/{10-architecture,40-data,50-security,60-testing,70-delivery}.md`.

---

## 10. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-23 | opencode | Creación inicial de la especificación |
| 2026-11-23 | opencode | Inicio de implementación: publicación de imagen, assets de despliegue y nodo Tailscale dedicado |
