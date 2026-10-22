---
title: "Unificación automation (2/4): backend integrado (rutas, plugins, scheduler y sync directo)"
id: "SPEC-110"
status: "pending_release"
author: "opencode"
created: "2026-10-21"
updated: "2026-10-21"
github_issue: 113
---

# Unificación automation (2/4): backend integrado (rutas, plugins, scheduler y sync directo)

**ID**: SPEC-110  
**Estado**: pending_release  
**Autor**: opencode  
**Creado**: 2026-10-21  
**Actualizado**: 2026-10-21

---

## 1. Resumen Ejecutivo

Segunda de las cuatro specs de la unificación (SPEC-108). SPEC-109 dejó en `app.db` las tablas `automation_*` y los datos migrados, pero el backend de automation sigue viviendo en el proyecto hermano (`p40la-ihost-automation`, puerto 8089): sus rutas, su autenticación, su scheduler y su cliente de plugins. Esta spec integra ese backend dentro de `p40la-ihost` para que exista **un solo proceso**: rutas `/api/automation/*` autenticadas con la sesión de p40la, scheduler en proceso, y sincronización directa (sin `AutomationClient` ni `automation_base_url`/`automation_api_key`).

Al terminar, la app principal podrá listar cuentas de automation por institución, administrar plugins y facturas, ejecutar el job diario y disparar la sincronización de un servicio sin depender del contenedor 8089. El frontend de esas pantallas queda para SPEC-111; el retiro del contenedor y del repo viejo, para SPEC-112.

**iHost**: se elimina un contenedor Go, un SQLite y un scheduler; el proceso único reutiliza la DB y el login existentes. Sin dependencias nuevas salvo el módulo privado de plugins, que ya usa automation.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Modelos de dominio para `automation_accounts`, `automation_plugins`, `automation_bills` y `automation_webhook_logs`, con `InstitutionID` en cuentas (SPEC-109).
2. **REQ-002**: Storage de automation sobre la DB de p40la: listar cuentas (join con `institutions`), obtener/crear/actualizar/borrar cuentas, listar plugins, listar facturas y logs por cuenta.
3. **REQ-003**: Endpoints autenticados por la sesión de p40la:
   - `GET /api/automation/accounts` (incluye `institution_name`)
   - `GET /api/automation/accounts/{id}`
   - `POST/PUT/DELETE /api/automation/accounts`
   - `GET /api/automation/plugins`
   - `GET /api/automation/accounts/{id}/bills`
   - `GET/PUT /api/automation/accounts/{id}/webhook`
   - `GET /api/automation/accounts/{id}/webhook/logs`
   - `GET/PUT /api/automation/settings/webhook`
4. **REQ-004**: Las credenciales de las cuentas se guardan como JSON en `credentials`; el endpoint de revelado (`POST /api/automation/accounts/{id}/credentials:reveal`) exige la contraseña del usuario de p40la (mismo patrón de seguridad que automation).
5. **REQ-005**: Ejecutar los plugins (módulo privado `p40la-ihost-automation-plugins`, repo aparte) para consultar facturas de una cuenta, persistirlas en `automation_bills` y entregarlas **en proceso** al servicio vinculado (sin POST HTTP interno).
6. **REQ-006**: Scheduler en proceso: job diario por cuenta con `schedule_enabled`, `schedule_time`, `schedule_frequency`, `schedule_interval` y `schedule_days`, con la misma semántica que automation. Debe convivir con los schedulers existentes de p40la sin bloquear el arranque.
7. **REQ-007**: `POST /api/services/{id}/sync` deja de usar `AutomationClient` y llama directamente al servicio de automation: fetch del plugin + `WebhookService.UpsertBill` en proceso. No hay llamada HTTP a `/webhooks/{uuid}`.
8. **REQ-008**: El bot de Telegram (`/sincronizar`, `/sincronizar_servicio`) usa el mismo camino directo.
9. **REQ-009**: Migrar las claves `automation_base_url` y `automation_api_key` a obsoletas: se dejan de leer y se documenta su eliminación del frontend en SPEC-111.
10. **REQ-010**: Tests unitarios de storage y servicios, y de integración del sync con un plugin de prueba. La validación con datos reales se corre contra una copia de la DB de producción (regla de AGENTS.md).
11. **REQ-017**: El módulo privado de plugins requiere Go 1.26. Actualizar `go.mod` (`go 1.26.0`) y el Dockerfile a `golang:1.26-alpine`; el runtime del iHost no cambia (binario estático).
11b. **REQ-018**: Los endpoints de alta/edición de cuenta aceptan `service_id`: un ID > 0 vincula esa cuenta al servicio (limpiando vínculos previos de la misma cuenta), `0` desvincula y el campo ausente no modifica nada. La relación sigue viviendo en `services.automation_account_id`; no hay migración de esquema y el sync existente no cambia.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-011**: Al crear/editar una cuenta, validar que `institution_id` exista; el formulario de servicio (SPEC-111) consumirá `GET /api/automation/accounts?for=service` con etiqueta `institución · identificador · plugin`.
2. **REQ-012**: Al borrar una institución con cuentas de automation, impedir la operación (FK) o pedir confirmación explícita; no debe quedar una cuenta huérfana.
3. **REQ-013**: Límite y paginación de logs de webhook (`limit`, `offset`) y retención configurable para no crecer sin control en el iHost.
4. **REQ-014**: Los errores de plugin se persisten en `automation_bills.error`/`automation_webhook_logs` y se devuelven con un mensaje legible; nunca se loguean credenciales.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-015**: Endpoint `POST /api/automation/accounts/{id}/bills:fetch` para consultar facturas manualmente (equivalente al botón "Consultar facturas" de automation).
2. **REQ-016**: Mantener `POST /api/accounts/{id}/webhook:run` del proyecto viejo solo hasta SPEC-112, para compatibilidad durante la transición.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: el fetch del plugin puede tardar segundos; timeout de 30 s y ejecución fuera del request cuando sea posible. El scheduler no debe competir con el de facturación.
- **Seguridad**: sesión única de p40la; credenciales nunca en logs; revelado con contraseña; el módulo de plugins se compila con `GOPRIVATE` + secret `gh_token` (SPEC-112).
- **Almacenamiento**: sin tablas nuevas (SPEC-109); retención de logs para acotar crecimiento.
- **Disponibilidad**: si el plugin falla, el resto de la app sigue operativo; el scheduler registra el error y continúa con la próxima cuenta.
- **iHost**: un proceso menos; sin Node en runtime; sin servicios externos obligatorios.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- Automación: `internal/{api,services,storage,models}` con auth propia, scheduler y plugin registry. Rutas relevantes: `/api/apps`, `/api/apps/{id}/accounts`, `/api/accounts/{id}/bills`, `/api/accounts/{id}/webhook`, `/api/settings/webhook`, `/api/accounts/{id}/webhook:run`.
- p40la: `internal/api` (handlers + `AuthMiddleware`), `internal/services`, `internal/storage`, `internal/models`; DB única `app.db`; schedulers existentes en `cmd/server/main.go`.
- SPEC-109: tablas `automation_*` y `automation_accounts.institution_id`; import offline idempotente.
- Módulo privado de plugins `github.com/paulomcnally/p40la-ihost-automation-plugins` v1.1.0 (contrato `Plugin`, `Bill`, `Registry`).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Copiar automation dentro de `internal/automation/*` (paquete autocontenido) | Port casi directo; bajo riesgo de romper p40la | Duplica capas y rompe la estructura plana del proyecto | ❌ Rechazada |
| Integrar en las capas existentes (`internal/{models,storage,services,api}`) | Sigue las reglas del proyecto; reutiliza auth, helpers y settings | Requiere adaptar imports y nombres | ✅ Seleccionada |
| Mantener el contenedor 8089 y solo proxear rutas desde p40la | Mínimo cambio | No cumple el objetivo de una sola app/DB; sigue el doble proceso | ❌ Rechazada |
| Ejecutar plugins vía HTTP al proceso viejo | Cero port del plugin registry | Depende del contenedor que se quiere retirar | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Backend integrado en las capas planas de p40la.
- **Contexto**: las reglas del proyecto exigen `internal/{api,services,storage,models}`.
- **Decisión**: modelos en `internal/models/automation.go`, storage en `internal/storage/automation*.go`, servicios en `internal/services/automation*.go`, handlers en `internal/api/automation_handlers.go`, scheduler en `internal/services/automation_scheduler.go`.
- **Consecuencias**: consistencia con el resto del código; más adaptación de imports que un port literal.

**ADR-002**: Sin auth propia de automation.
- **Contexto**: automation tiene login, cookie y tabla `users`; p40la ya tiene sesión.
- **Decisión**: todas las rutas `/api/automation/*` van detrás de `AuthMiddleware`; se descarta la tabla `users` de automation (SPEC-109).
- **Consecuencias**: un solo login; desaparece el setup/login de automation.

**ADR-003**: Scheduler en proceso con registro idempotente.
- **Contexto**: el scheduler de automation corre en su propio contenedor y hace POST al webhook.
- **Decisión**: portar la lógica a un scheduler de p40la que consulta las cuentas con `schedule_enabled` y ejecuta el job directo (plugin → upsert de facturas), registrando `last_run_at` y logs. Un lock por cuenta evita solapamientos.
- **Consecuencias**: un proceso menos; hay que cuidar el orden de arranque y los timeouts.

**ADR-004**: Compatibilidad temporal del endpoint `webhook:run`.
- **Contexto**: durante la transición, el p40la viejo puede seguir llamando a automation.
- **Decisión**: mantener el endpoint en el proyecto viejo hasta SPEC-112; el nuevo backend de p40la no lo necesita.
- **Consecuencias**: convivencia breve sin romper el sync actual.

**ADR-006**: El vínculo cuenta → servicio se administra desde automation.
- **Contexto**: el usuario agrega primero el servicio y luego la automatización; el selector en el formulario de servicio invierte ese flujo.
- **Decisión**: la UI de automation elige el servicio y el backend escribe `services.automation_account_id` a partir de `service_id`. No se agrega una columna nueva.
- **Consecuencias**: el formulario de servicio deja de ofrecer el selector (solo preserva el valor existente al editar); la relación sigue siendo una cuenta → un servicio y el sync no cambia.

**ADR-005**: El sync interno no usa webhooks HTTP.
- **Contexto**: al estar todo en el mismo proceso, el POST a `/webhooks/{uuid}` es un salto innecesario; además `automation_webhook_logs` ya existe para la historia.
- **Decisión**: el sync llama a `WebhookService.UpsertBill` directamente. El servicio destino se resuelve por `services.automation_account_id` (preferido) o, como fallback de transición, por el `uuid` del `webhook_url` de la cuenta. El endpoint `POST /webhooks/{uuid}` se mantiene **solo para clientes externos** (SPEC-069).
- **Consecuencias**: menos latencia y menos puntos de fallo; los clientes externos no se ven afectados. Si una cuenta no tiene servicio vinculado ni webhook_url válido, el sync falla con un mensaje claro.

## 4. Diseño Técnico

### 4.1 Diagrama

```
Frontend (SPEC-111) / Bot Telegram / SyncService
        │  sesión p40la
        v
[internal/api/automation_handlers.go]
        │
        v
[internal/services/automation*.go] ──> [plugins registry (módulo privado)]
        │                                      │
        v                                      v
[internal/storage/automation*.go]        proveedor externo
        │
        v
   app.db (automation_* + institutions + services)
```

### 4.2 Componentes

- `internal/models/automation.go`: `AutomationAccount`, `AutomationPlugin`, `AutomationBill`, `AutomationWebhookLog`.
- `internal/storage/automation.go`: CRUD y consultas (join instituciones).
- `internal/services/automation.go`: reglas de negocio, validaciones, sync y fetch.
- `internal/services/automation_scheduler.go`: jobs diarios.
- `internal/api/automation_handlers.go`: endpoints.
- `cmd/server/main.go`: wiring y arranque del scheduler.

### 4.3 Modelo de datos

El de SPEC-109 §4.3. Cuentas con `institution_id`, plugins, facturas y logs. Sin tablas nuevas.

### 4.4 APIs / Contratos

- `GET /api/automation/accounts` → `[{id, institution_id, institution_name, identifier, label, plugin_name, webhook_url, schedule_*}]`
- `GET /api/automation/accounts?for=service` → `[{id, label: "Institución · identificador · plugin"}]`
- `GET/POST/PUT/DELETE /api/automation/accounts...`
- `POST /api/automation/accounts/{id}/credentials:reveal` (body `{password}`)
- `GET /api/automation/plugins`
- `GET /api/automation/accounts/{id}/bills`
- `GET/PUT /api/automation/accounts/{id}/webhook`
- `GET /api/automation/accounts/{id}/webhook/logs?limit=&offset=`
- `GET/PUT /api/automation/settings/webhook`
- `POST /api/automation/accounts/{id}/bills:fetch`

### 4.5 Dependencias

- **Internas**: `internal/db`, `internal/storage`, `internal/services`, `internal/api`, `internal/models`, `cmd/server`, bot de Telegram.
- **Externas**: `github.com/paulomcnally/p40la-ihost-automation-plugins` v1.1.0 (privado, repo aparte; requiere Go 1.26), `modernc.org/sqlite` (ya presente). Sin otras.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Las rutas `/api/automation/*` responden autenticadas por sesión y 401 sin sesión.
- [ ] CA-002: `GET /api/automation/accounts` devuelve institución, identificador, plugin y schedule, sin exponer credenciales.
- [ ] CA-003: CRUD de cuentas valida `institution_id` y no deja huérfanos.
- [ ] CA-004: El fetch de un plugin persiste facturas en `automation_bills` y registra el error si falla.
- [ ] CA-005: El scheduler ejecuta el job de una cuenta habilitada y actualiza `last_run_at`/logs.
- [ ] CA-006: `POST /api/services/{id}/sync` funciona sin `automation_base_url`/`automation_api_key`.
- [ ] CA-007: `/sincronizar` y `/sincronizar_servicio` usan el servicio directo.
- [ ] CA-008: No se loguean credenciales ni claves.
- [ ] CA-009: `service_id` en create/update vincula/desvincula el servicio correctamente (0 desvincula, ausente preserva) y el detalle muestra el vínculo resultante.
- [ ] CA-DARK: (aplica a SPEC-111; acá no hay UI nueva).

### 5.2 No funcionales

- [ ] CA-NF-001: `go build ./...` y `go test ./...` pasan.
- [ ] CA-NF-002: La validación con datos reales se corre contra una copia de la DB de producción.
- [ ] CA-NF-003: El scheduler no bloquea el arranque ni se solapa con los existentes.

### 5.3 Testing

- **Unit tests**: storage (join instituciones), validaciones, scheduler (cálculo de próxima ejecución), sync con plugin de prueba.
- **Integration tests**: rutas con sesión; fetch y persistencia; copia de producción para el matching.
- **E2E**: sync desde el menú de servicios y desde el bot.

## 6. Plan de Implementación

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Modelos + storage + listado de cuentas/plugins | 1 día | SPEC-109 |
| 2 | Handlers y rutas `/api/automation/*` + selector | 1 día | Fase 1 |
| 3 | Plugin registry + fetch de facturas + webhook/logs | 1,5 días | Fase 2 |
| 4 | Scheduler en proceso + sync directo + bot | 1,5 días | Fase 3 |
| 5 | Tests, limpieza de `AutomationClient` y validación con copia de producción | 1 día | Fase 4 |

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| Dependencia del módulo privado de plugins en el build | Media | Alto | `GOPRIVATE` + secret `gh_token`; documentado en SPEC-112 |
| Diferencias de comportamiento del scheduler | Media | Medio | Portar tests de automation y comparar ejecuciones |
| Doble ejecución durante la transición (8089 y p40la) | Media | Alto | Deshabilitar el scheduler viejo antes de arrancar el nuevo; ventana de mantenimiento |
| Credenciales expuestas en logs | Baja | Alto | Revisión de logs y tests que verifiquen que no se imprimen |
| Crecimiento de logs de webhook | Media | Bajo | Retención/paginación (REQ-013) |

## 8. Notas y Referencias

- SPEC-108 (paraguas), SPEC-109 (esquema/datos), SPEC-111 (frontend), SPEC-112 (deploy).
- `p40la-ihost-automation`: `internal/services/{webhook,scheduler,bills}.go`, `internal/api/*`.
- Regla AGENTS.md: validar con copia de producción.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-10-21 | opencode | Creación inicial: backend integrado de automation (rutas, plugins, scheduler y sync directo). |
| 2026-10-21 | opencode | Fase 1 implementada: modelos, storage con join a `institutions`, servicios y endpoints de lectura (`/api/automation/accounts`, `?for=service`, plugins, facturas, logs) con tests y validación contra copia de producción. |
| 2026-10-21 | opencode | Fase 2 implementada: CRUD de cuentas con validación de institución/plugin y unicidad, revelado de credenciales con contraseña, webhook/schedule por cuenta y bloqueo de borrado si hay servicios vinculados. E2E contra copia de producción (201/200/401/409/400). |
| 2026-10-21 | opencode | Fase 3 implementada: registry del módulo privado de plugins (repo aparte), catálogo de plugins, fetch y persistencia de facturas, sync directo a `WebhookService.UpsertBill` (sin webhook HTTP interno), scheduler en proceso y bot/endpoint de sync usando `AutomationSyncer`. Go 1.26 y Dockerfile actualizados por requisito del módulo. E2E: 5 plugins registrados y scheduler iniciado sobre copia de producción. Se agregaron `GET /api/automation/plugins/{name}/schema` y `POST /api/automation/accounts/{id}/bills:fetch`. |
| 2026-10-21 | opencode | Backend completo y validado: CRUD, plugins, sync directo, scheduler, bot y endpoints auxiliares. `go test ./...` y build OK. Pasa a pending_release a la espera del release conjunto 109–112. |
| 2026-10-21 | opencode | Cambio iterativo: create/update de cuenta aceptan `service_id` para vincular/desvincular el servicio desde automation, sin cambiar el esquema ni el sync. |
