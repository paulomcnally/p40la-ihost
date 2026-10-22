---
title: "Evaluación: unificar p40la-ihost-automation dentro de p40la-ihost (una sola app y una sola DB)"
id: "SPEC-108"
status: "draft"
author: "opencode"
created: "2026-10-21"
updated: "2026-10-21"
github_issue: 111
---

# Evaluación: unificar p40la-ihost-automation dentro de p40la-ihost (una sola app y una sola DB)

**ID**: SPEC-108  
**Estado**: draft  
**Autor**: opencode  
**Creado**: 2026-10-21  
**Actualizado**: 2026-10-21

---

## 1. Resumen Ejecutivo

Hoy la sincronización de facturas vive en un proyecto separado, `p40la-ihost-automation` (puerto `8089`, su propia DB SQLite, su propio login, su propio frontend y su propio scheduler). `p40la-ihost` (puerto `8088`) solo lo invoca por HTTP usando `automation_base_url` + `automation_api_key`. Esa separación obliga a mantener dos bases de datos, dos despliegues, dos sistemas de specs y un vínculo manual: el usuario debe escribir a mano el **ID numérico de la cuenta de automation** en cada servicio (SPEC-092). Ese campo es la peor consecuencia del diseño actual: no hay forma de elegir la cuenta, hay que memorizar un número, y si se borra o recrea la cuenta el vínculo queda apuntando a la nada.

Esta spec es una **evaluación técnica y de decisión** (no implementa código): releva el inventario de ambos proyectos, identifica las colisiones de esquema y de autenticación, compara las alternativas (mantener dos servicios con un selector de cuentas, unificar todo en un solo binario/DB, o compartir DB con dos procesos) y deja una recomendación con plan por fases, migración de datos y rollback. El objetivo final es que exista **una sola app, un solo puerto, una sola DB y un solo login**, y que el vínculo servicio ↔ cuenta se resuelva con un **selector**, nunca escribiendo un ID.

**Consideraciones iHost**: el iHost corre con recursos limitados. Hoy hay dos contenedores Go + dos SQLite + dos frontends estáticos + dos schedulers. Unificar reduce RAM, disco, superficie de fallo y logs duplicados. Ambos proyectos ya usan el mismo stack (`Go`, `modernc.org/sqlite`, `React + Vite + Tailwind`, `distroless`, multi-arch), por lo que la fusión es viable sin nuevas dependencias pesadas.

**Consideraciones de UI**: la spec no implementa UI, pero el objetivo funcional incluye eliminar el campo numérico "ID de cuenta en Automation" y reemplazarlo por un `<select>` con las cuentas reales. Cualquier implementación posterior debe usar tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y verificarse en darkmode.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Inventariar ambos proyectos: estructura de carpetas, tablas y columnas, endpoints, autenticación, migraciones, scheduler, sistema de plugins, build Docker y contratos de integración actuales (`webhook:run`, webhooks de facturas, `automation_account_id`).
2. **REQ-002**: Identificar todas las **colisiones de esquema** entre las dos DB (tablas y columnas con el mismo nombre y distinto significado), en particular `users`, `settings` vs `system_settings`, `bills` (facturas de proveedor vs facturas de servicio) y `plugins`.
3. **REQ-003**: Evaluar y documentar al menos estas alternativas: (A) mantener dos servicios y resolver el vínculo con un selector alimentado por un endpoint server-to-server; (B) unificar `automation` dentro de `p40la-ihost` (un binario, un puerto, una DB); (C) compartir el mismo archivo SQLite entre dos procesos.
4. **REQ-004**: Recomendar una alternativa con justificación explícita, costos, riesgos y descarte de las demás.
5. **REQ-005**: Definir el **mapeo de datos** de la alternativa recomendada: qué tablas se renombran/prefijan, qué datos se migran, qué se descarta (usuarios y sesiones de automation) y cómo se preservan cuentas, credenciales, plugins y logs de webhook.
6. **REQ-006**: Definir el plan de migración de datos con pasos verificables sobre una **copia** de la DB de producción (nunca sobre producción), incluyendo backup, `PRAGMA integrity_check` y rollback.
7. **REQ-007**: Definir cómo se elimina el campo manual `services.automation_account_id` como texto libre y se reemplaza por una selección de cuenta (dropdown) validada contra la tabla real de cuentas; el valor persistido debe ser una FK válida o `NULL`.
8. **REQ-008**: Definir el plan de despliegue: una sola imagen multi-arch (`linux/amd64`, `linux/arm/v7`, `linux/arm64`), un solo puerto (`8088`), un solo volumen (`p40la`), una sola ruta de datos (`/app/data/app.db`).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-009**: Definir cómo se unifican los schedulers (generación de facturas de `p40la` y jobs diarios de automation) en un solo proceso, sin duplicar ejecuciones ni bloquear el arranque.
2. **REQ-010**: Definir cómo se absorbe el sistema de plugins (módulo privado `p40la-ihost-automation-plugins`) sin romper el build: `GOPRIVATE`, secret `gh_token` de BuildKit y versionado del módulo.
3. **REQ-011**: Definir la convivencia de autenticación: automation tiene su tabla `users` y su cookie de sesión; la alternativa recomendada debe usar el login de `p40la-ihost` y descartar el login propio.
4. **REQ-012**: Definir qué pasa con la convivencia temporal durante la transición (¿se puede correr el automation viejo en paralelo? ¿se apaga antes de migrar? ¿cómo se evita doble entrega de facturas?).
5. **REQ-013**: Definir el destino del repo `p40la-ihost-automation`: archivar, dejar de publicar imágenes o mantener solo como fuente histórica, una vez liberada la unificación.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-014**: Evaluar si conviene migrar el frontend de automation a las páginas existentes de `p40la-ihost` (Servicios / Instituciones) o mantener secciones "Automation" separadas dentro del mismo frontend.
2. **REQ-015**: Evaluar si los logs de webhook deben retenerse con rotación (tamaño máximo) para no crecer sin límite en el iHost.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: la unificación no debe aumentar el uso de RAM del iHost; el objetivo es reducirlo (un proceso menos). El arranque del binario único no debe superar el arranque actual de `p40la-ihost`.
- **Seguridad**: una sola sesión y un solo `SESSION_SECRET`; no duplicar credenciales. Las credenciales de proveedores siguen cifradas/almacenadas como hoy. No se deben loguear claves.
- **Almacenamiento**: una sola DB en el volumen `p40la`; la migración debe documentar el crecimiento esperado (apps, cuentas, logs de webhook).
- **Disponibilidad**: la migración se hace con los contenedores detenidos y backup previo; el rollback debe ser posible restaurando el backup.
- **iHost**: sin dependencias nuevas; binario estático; multi-arch obligatorio; sin Node en runtime.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

**Proyecto `p40la-ihost-automation`** (repo hermano, `0.1.10`):
- Backend Go `1.26`, SQLite `modernc.org/sqlite`, migraciones propias con tabla `schema_migrations` (6 migraciones: `0001`–`0006`).
- Tablas: `users`, `settings`, `apps`, `accounts`, `plugins`, `bills`, `webhook_logs`.
- Servicios: `auth`, `apps`, `bills`, `scheduler`, `settings`, `webhook`.
- Endpoints propios: login/sesión, apps/cuentas, plugins, facturas, configuración de webhooks y logs, más `POST /api/accounts/{id}/webhook:run` (auth por token `X-Webhook-Key`, SPEC-017).
- Scheduler diario por cuenta: consulta el plugin del proveedor y hace POST al webhook de `p40la-ihost`.
- Plugins: contrato en módulo **privado** `github.com/paulomcnally/p40la-ihost-automation-plugins` (repo `p40la-ihost-automation-plugins`), importado por el backend; el Dockerfile lo descarga con `GOPRIVATE` + secret `gh_token`.
- Frontend React + Vite + Tailwind, servido estático por el backend; puerto `8089`; imagen multi-arch `paulomcnally/p40la-ihost-automation`.
- Volumen propio `p40la-ihost-automation-data`; container `p40la-ihost-automation`.

**Proyecto `p40la-ihost`** (este repo, `0.4.52`):
- Backend Go `1.24`, misma DB (`modernc.org/sqlite`), migraciones `0001`–`0038`.
- Tablas propias: `users`, `homes`, `institutions`, `services`, `bills`, `debts`, `debt_bills`, `children`, `categories`, `budget_*`, `system_settings`, `notifications`, etc.
- Autenticación por sesión/cookie propia; frontend React + Vite + Tailwind; puerto `8088`; volumen `p40la`; imagen multi-arch.
- Integración actual con automation: `automation_base_url`, `automation_api_key` en `system_settings`; `services.automation_account_id` (entero nullable, migración `0034`); `AutomationClient.SyncAccount` llama a `POST {base}/api/accounts/{id}/webhook:run` (SPEC-092). El bot de Telegram expone `/sincronizar` y `/sincronizar_servicio` (SPEC-103).

**Colisiones detectadas a priori**:

| Tabla | `p40la-ihost` | `automation` | Conflicto |
|---|---|---|---|
| `users` | usuarios de la app | usuarios de automation | 🔴 Mismo nombre y propósito distinto → usar solo la de p40la |
| `settings` | — | KV de automation (`webhook_api_key`, `webhook_enabled`, `language`) | 🟡 Convive con `system_settings`; unificar en `system_settings` |
| `bills` | facturas de servicios | facturas obtenidas del proveedor | 🔴 Mismo nombre, semántica distinta → renombrar a `automation_bills` |
| `plugins` | — | catálogo de plugins | 🟢 Sin colisión si se mantiene nombre |
| `apps` / `accounts` | — | apps y cuentas de proveedores | 🟢 Sin colisión, pero conviene prefijo `automation_` |
| `webhook_logs` | — | logs de entrega | 🟢 Sin colisión |
| `schema_migrations` | `0001`–`0038` | `0001`–`0006` | 🔴 Numeración duplicada → renumerar/renombrar al migrar |

**Otras diferencias**: dos cookies de sesión, dos `SESSION_SECRET`, dos Dockerfiles, dos volúmenes, dos UIs y dos sistemas de specs.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **A. Mantener dos servicios y agregar selector de cuentas** (endpoint server-to-server que liste cuentas; `p40la` lo consume para el dropdown) | Cambio pequeño; no toca datos; resuelve el ID manual ya; bajo riesgo | Sigue habiendo 2 DB, 2 despliegues, 2 logins, 2 schedulers y deriva de esquema; el iHost sigue con doble consumo | ❌ Rechazada como solución final (sirve como mitigación inmediata) |
| **B. Unificar `automation` dentro de `p40la-ihost`** (un binario, un puerto, una DB, un login) | Una sola app/DB/login/puerto/volumen; elimina el ID manual de raíz; menos RAM y disco; un solo scheduler; un solo despliegue multi-arch | Trabajo grande de migración de datos y de UI; acopla plugins privados al build; requiere ventana de mantenimiento y pruebas cuidadosas | ✅ **Seleccionada** |
| **C. Compartir el mismo archivo SQLite entre dos procesos** | No se migra el esquema; cada app sigue con su código | WAL con dos procesos y dos pools de conexiones aumenta riesgo de `SQLITE_BUSY`/locks; no elimina el doble login ni el doble deploy; acopla versiones del esquema | ❌ Rechazada |
| **D. Unificar al revés** (traer `p40la` dentro de `automation`) | — | `automation` es el proyecto más chico y sin hogar/deudas/budget; mover el núcleo sería absurdo | ❌ Rechazada |
| **E. No hacer nada y documentar el ID manual** | Cero trabajo | Mantiene la peor UX y la deuda de dos sistemas | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: El destino es un solo binario Go y una sola DB (`app.db`) en el puerto `8088`.
- **Contexto**: dos procesos con el mismo stack y datos relacionados (cuentas ↔ servicios) generan doble mantenimiento y el vínculo manual por ID.
- **Decisión**: absorber los paquetes de automation (`apps`, `accounts`, `plugins`, `bills`, `webhook`, `scheduler`) dentro de `p40la-ihost`, con prefijo `automation_` en tablas nuevas y rutas `/automation/*` en el backend y el frontend.
- **Consecuencias**: Positivo: un solo despliegue, una sola sesión, menos RAM, dropdown natural contra la tabla de cuentas. Negativo: migración de datos y de UI, y el build pasa a depender del módulo privado de plugins.

**ADR-002**: Las tablas de automation se renombran con prefijo `automation_` para evitar colisiones.
- **Contexto**: `bills`, `users` y `settings` ya existen en `p40la-ihost` con otro significado.
- **Decisión**: `apps` → `automation_apps`, `accounts` → `automation_accounts`, `plugins` → `automation_plugins`, `bills` → `automation_bills`, `webhook_logs` → `automation_webhook_logs`; `settings` de automation se fusiona en `system_settings`; `users` de automation se descarta (se usa el login de p40la).
- **Consecuencias**: migración explícita tabla por tabla; las consultas de automation cambian de nombre, pero el resto de p40la no se toca.

**ADR-003**: El vínculo servicio ↔ cuenta se valida contra la tabla real, no es un número tipeado.
- **Contexto**: `services.automation_account_id` es un entero libre (SPEC-092) que el usuario debe conocer de memoria.
- **Decisión**: en el formulario de servicio, reemplazar el input numérico por un `<select>` poblado desde `automation_accounts` (con app/proveedor como etiqueta). Persistir el ID y **validar en backend** que exista (la columna se mantiene por compatibilidad; agregar una FK física a una tabla existente en SQLite requiere un rebuild de `services`, que se evalúa en la spec de implementación). Mostrar "Sin cuenta" para `NULL`.
- **Consecuencias**: se elimina el error de ID inválido y la necesidad de mirar la URL de automation. El prototipo demostró que un ID inválido no rompe la consulta pero se degrada silenciosamente a "Sin cuenta" (LEFT JOIN), por lo que la validación en backend es obligatoria.

**ADR-004**: El scheduler de automation se ejecuta dentro del proceso de `p40la-ihost`.
- **Contexto**: hoy hay dos schedulers independientes; al unificar, el job diario por cuenta debe seguir corriendo sin contenedor aparte.
- **Decisión**: reutilizar el patrón de schedulers existente de p40la (goroutines con ticker y hora configurada), con las mismas garantías de idempotencia del upsert de facturas (SPEC-069/SPEC-071).
- **Consecuencias**: menos procesos; hay que cuidar que la ejecución no bloquee el arranque ni se solape con el scheduler de facturación.

## 4. Diseño Técnico

### 4.6 Prototipo de migración (resultado)

Se ejecutó la migración propuesta sobre **copias** de las dos DB (nunca sobre producción):

| Paso | Resultado |
|---|---|
| Backup con `sqlite3 .backup` de `p40la` y `automation` | ✅ |
| Creación de `automation_apps`, `automation_plugins`, `automation_accounts`, `automation_bills`, `automation_webhook_logs` | ✅ |
| Copia de datos preservando IDs | ✅ 5 apps, 5 cuentas, 5 plugins, 28 facturas, 100 logs |
| Migración de settings (`webhook_enabled`, `language`) sin pisar `webhook_api_key` de p40la | ✅ |
| `PRAGMA integrity_check` | `ok` |
| `PRAGMA foreign_key_check` | sin errores |
| Consulta del selector (app + identificador + plugin) | ✅ 5 opciones con etiqueta legible |
| Servicio con ID inválido (`999`) | Se degrada a "Sin cuenta"; **requiere validación en backend** |

**Hallazgo crítico del prototipo**: copiar solo `app.db` pierde el contenido del `-wal`. En la primera prueba la DB de automation tenía 6 facturas y 52 logs sin checkpointear; el backup plano los perdió. Toda migración y backup DEBE usar `sqlite3 .backup` (o copiar `app.db` + `app.db-wal` + `app.db-shm` con los procesos detenidos). Esto se incorpora como paso obligatorio del plan.

### 4.7 Estrategia de entrega en varias specs

Para no hacer un cambio gigantesco, la unificación se parte en specs secuenciales. **SPEC-108 queda como spec paraguas** (evaluación y plan) y las implementaciones se crean al iniciar cada fase, cada una en su worktree:

| Spec propuesta | Alcance | Depende de |
|---|---|---|
| **SPEC-109** | **Esquema y datos**: migración `0039+` con tablas `automation_*`, copia de datos, settings, índice de `services.automation_account_id`, script de verificación y rollback. Sin UI ni rutas nuevas. | SPEC-108 |
| **SPEC-110** | **Backend unificado**: mover `apps/accounts/plugins/bills/webhook` a `internal/automation`, rutas `/api/automation/*` con la sesión de p40la, scheduler en proceso y eliminación gradual de `AutomationClient` / `automation_base_url` / `automation_api_key`. | SPEC-109 |
| **SPEC-111** | **Frontend unificado**: páginas de automation dentro de la SPA, selector de cuentas en `ServiceFormPage`, retiro del campo numérico, `BACK_ROUTES` y darkmode. | SPEC-110 |
| **SPEC-112** | **Despliegue único y retiro**: Dockerfile único con plugins privados, multi-arch, puerto 8088, volumen `p40la`, apagado del contenedor `p40la-ihost-automation` y archivo del repo viejo. | SPEC-110/111 |

Cada spec de implementación se crea en su propio worktree y se libera por separado. La DB vieja de automation se conserva como respaldo hasta completar SPEC-112.


### 4.1 Diagrama de arquitectura (estado objetivo)

```
                    iHost (8088, multi-arch)
┌──────────────────────────────────────────────────────────┐
│  p40la-ihost (un solo binario Go)                        │
│                                                          │
│  Frontend React (una SPA)                                │
│    ├── Servicios / Deudas / Budget / ...                 │
│    └── Automation: Apps, Cuentas, Plugins, Logs          │
│                                                          │
│  API HTTP ── AuthMiddleware (una sesión)                 │
│    ├── /api/...                 (dominio p40la)          │
│    └── /api/automation/...      (apps, cuentas, jobs)    │
│                                                          │
│  Schedulers (un proceso)                                 │
│    ├── facturación / alertas / deudas                    │
│    └── jobs diarios de automation                        │
│                                                          │
│  SQLite única: /app/data/app.db (WAL)                    │
│    tablas p40la + automation_* (prefijo)                 │
└──────────────────────────────────────────────────────────┘
        │
        └── plugins (módulo privado, compilado en el binario)
```

### 4.2 Componentes a absorber

#### 4.2.1 Paquete `automation` (nuevo)
- **Responsabilidad**: agrupar las capas de automation (`api`, `services`, `storage`, `models`) dentro de p40la.
- **Interfaz**: rutas `/api/automation/*` autenticadas por la sesión de p40la.
- **Dependencias**: `AutomationClient` actual, `system_settings`, storage de p40la.
- **Ubicación propuesta**: `internal/automation/{api,services,storage,models}`.

#### 4.2.2 Frontend
- **Responsabilidad**: mover las pantallas de automation (Apps, detalle de cuenta/facturas, Plugins, Settings de webhooks) a la SPA de p40la.
- **Interfaz**: rutas `/automation`, `/automation/apps/:appId`, `/automation/apps/:appId/accounts/:accountId/bills`, `/automation/plugins`.
- **Notas**: registrar las rutas de detalle en `BACK_ROUTES` (SPEC-063). Inputs con tokens del tema y verificación en darkmode.

#### 4.2.3 Scheduler
- **Responsabilidad**: job diario por cuenta (`fetch del plugin` → `webhook de p40la`).
- **Interfaz**: misma lógica de `RunJob`; sin llamada HTTP externa cuando el destino es el webhook local (se puede invocar el servicio directamente).
- **Consecuencias**: se elimina la dependencia de `automation_base_url`/`automation_api_key` y el endpoint `webhook:run`.

#### 4.2.4 Selector de cuenta en Servicio
- **Responsabilidad**: reemplazar el campo `automation_account_id` numérico por un `<select>`.
- **Interfaz**: opciones `{id, label}` donde el label combina app + identificador de cuenta + plugin.
- **Validación**: backend verifica que la cuenta exista antes de guardar.

### 4.3 Modelo de datos (estado objetivo)

```
automation_apps
- id, name, created_at, updated_at

automation_accounts
- id, app_id (FK automation_apps), identifier, label, credentials (JSON), plugin_name, created_at, updated_at

automation_plugins
- id, name, version, source, status, created_at, updated_at

automation_bills
- id, account_id (FK automation_accounts), period, amount, due_date, raw (JSON), created_at

automation_webhook_logs
- id, account_id, status, http_status, payload, response, created_at

services
- automation_account_id -> FK automation_accounts(id) NULL  (mismo nombre, semántica nueva)
```

Los settings de automation (`webhook_api_key`, `webhook_enabled`, `language`) se migran a `system_settings` con las claves actuales; `users` de automation se descarta.

### 4.4 APIs / Contratos

- `GET /api/automation/accounts?for=service` → `[{id, label, app_name, plugin_name}]` para poblar el selector.
- `GET /api/automation/apps`, `GET /api/automation/apps/{id}/accounts`, `GET /api/automation/accounts/{id}/bills`.
- `GET/PUT /api/automation/settings` (webhooks) usando la sesión de p40la.
- Se **eliminan** `GET/PUT /api/settings/webhook` de automation y el endpoint `webhook:run` (ya no hay llamada server-to-server).
- Se mantiene `POST /webhooks/{uuid}` de p40la para clientes externos (SPEC-069).

### 4.5 Dependencias

- **Internas**: `internal/api`, `internal/services`, `internal/storage`, `internal/models`, `cmd/server`, `frontend/src`, `migrations/`.
- **Externas**: `github.com/paulomcnally/p40la-ihost-automation-plugins` (privado), `modernc.org/sqlite`, `golang.org/x/crypto`, `go-telegram/bot` (ya presente). Sin dependencias nuevas.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Existe un inventario completo y verificado de tablas, endpoints, migraciones y componentes de ambos proyectos.
- [ ] CA-002: Está documentada la matriz de colisiones de esquema con la decisión de renombrado/fusión/descarte para cada tabla.
- [ ] CA-003: Está documentada la recomendación (Opción B) con justificación y el descarte explícito de A, C, D y E.
- [ ] CA-004: Existe un plan de migración de datos paso a paso, probado sobre una copia de la DB, con backup previo, `integrity_check` y rollback. *(Avance: prototipo ejecutado con `integrity_check = ok`, `foreign_key_check` limpio y conteos preservados.)*
- [ ] CA-005: Está definido el reemplazo del campo numérico por un selector de cuentas, con validación en backend y `NULL` permitido.
- [ ] CA-006: Está definido el plan de despliegue de una sola imagen multi-arch en el puerto `8088` con un solo volumen.
- [ ] CA-DARK: Si la implementación posterior toca formularios/modales, los inputs usan tokens del tema (`bg-card`, `text-text`) y se verificó legibilidad en darkmode.
- [ ] CA-BACK: Si la implementación posterior crea páginas de detalle, el retorno se hace con la flecha atrás del header (registrada en `BACK_ROUTES`).

### 5.2 No funcionales

- [ ] CA-NF-001: La estimación de esfuerzo y la lista de riesgos están completas y revisadas.
- [ ] CA-NF-002: El plan mantiene las 3 arquitecturas Docker (`amd64`, `arm/v7`, `arm64`) y la viabilidad de build con el módulo privado.
- [ ] CA-NF-003: El plan no incorpora dependencias nuevas y reduce el consumo de recursos del iHost (un proceso menos).

### 5.3 Testing

- **Unit tests**: definidos en la spec de implementación posterior (migración de storage y selector).
- **Integration tests**: migración sobre copia de DB; verificación de que cuentas, credenciales y logs sobreviven.
- **E2E**: alta/edición de servicio con selector; sincronización manual; job diario; login único.
- **Carga/Performance**: comparar RSS del iHost antes/después; el objetivo es menor o igual.

## 6. Plan de Implementación

### 6.1 Fases (de esta evaluación)

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Relevamiento e inventario de ambos repos (tablas, rutas, servicios, build) | 1 día | Ninguna |
| 2 | Matriz de colisiones y decisión de renombrado/fusión/descarte | 0,5 día | Fase 1 |
| 3 | Prototipo de migración sobre copia de DB + `integrity_check` | 1–2 días | Fase 2 |
| 4 | Plan de implementación detallado (backend, frontend, scheduler, Docker, rollback) y estimación final | 1 día | Fase 3 |

### 6.2 Milestones

1. **Decisión aprobada**: spec en `pending_execution` con la Opción B y el plan de migración validado.
2. **MVP unificado**: una sola app/DB/puerto, login único, cuentas y plugins operativos, selector de cuenta en servicios, scheduler de automation corriendo en proceso.
3. **Retiro del proyecto viejo**: repo `p40la-ihost-automation` archivado y su contenedor apagado, con la DB vieja respaldada.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Colisión de tablas (`bills`, `users`, `settings`) | Alta | Alto | Prefijo `automation_` + migración explícita; probar sobre copia |
| Migración de numeración de migraciones (`0006` vs `0038`) | Alta | Medio | Renumerar como `0039+` y documentar el mapeo; nunca reutilizar IDs |
| Build con módulo privado de plugins en CI/Docker | Media | Alto | Mantener `GOPRIVATE` + secret `gh_token`; documentar el requisito |
| Doble entrega de facturas durante la transición (ambos schedulers) | Media | Alto | Apagar automation antes de arrancar el unificado; ventana de mantenimiento; idempotencia del upsert (SPEC-069/071) |
| Pérdida de datos en la migración | Baja | Alto | Backup obligatorio + `integrity_check` + rollback documentado |
| Backup incompleto por `-wal` sin checkpointear | Media | Alto | Usar siempre `sqlite3 .backup` (validado en el prototipo: una copia plana perdió 6 facturas y 52 logs) |
| Scheduler único bloquea el arranque o se solapa | Media | Medio | Reutilizar el patrón de schedulers de p40la; ejecutar jobs en goroutines con timeout |
| Crecimiento de `automation_webhook_logs` | Media | Bajo | Retención/rotación por antigüedad o tamaño (REQ-015) |
| Divergencia de specs entre los dos repos | Alta | Bajo | Esta spec absorbe el seguimiento; el repo viejo queda congelado |

## 8. Notas y Referencias

- SPEC-092: Sincronización manual con plugins de automation (`automation_base_url`, `automation_api_key`, `services.automation_account_id`).
- SPEC-103: `/sincronizar` y botones inline en Telegram.
- SPEC-069: Webhooks de facturas por servicio.
- SPEC-071: Reactivación de facturas soft-deleted (idempotencia del upsert).
- SPEC-063: Navegación de retorno (`BACK_ROUTES`).
- Repo `p40la-ihost-automation` (v0.1.10): `internal/*`, `migrations/0001`–`0006`, Dockerfile multi-arch, puerto `8089`.
- Repo privado `p40la-ihost-automation-plugins` (contrato `plugins.Plugin`, `Bill`, `Registry`).
- Mitigación inmediata (si se aprueba antes de la unificación): endpoint server-to-server en automation para listar cuentas y dropdown en `ServiceFormPage`; elimina el ID manual sin esperar la fusión.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-10-21 | opencode | Creación inicial: evaluación de unificación de `p40la-ihost-automation` dentro de `p40la-ihost` para eliminar el ID manual de cuenta y las dos DB. |
| 2026-10-21 | opencode | Prototipo de migración ejecutado sobre copias: tablas `automation_*`, 28 facturas y 100 logs preservados, `integrity_check` ok. Se documentó la pérdida de datos si se copia la DB sin el `-wal` y se definió la entrega en SPEC-109…SPEC-112. |
