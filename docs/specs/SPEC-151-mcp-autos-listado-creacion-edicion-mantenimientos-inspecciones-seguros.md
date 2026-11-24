---
title: "MCP de autos: listado, creación/edición y registro de mantenimientos, inspecciones y seguros"
id: "SPEC-151"
status: "pending_release"
author: "opencode"
created: "2026-11-24"
updated: "2026-11-24"
github_issue: 154
---

# MCP de autos: listado, creación/edición y registro de mantenimientos, inspecciones y seguros

**ID**: SPEC-151  
**Estado**: pending_release  
**Autor**: opencode  
**Creado**: 2026-11-24  
**Actualizado**: 2026-11-24

---

## 1. Resumen Ejecutivo

El servidor MCP de p40la-ihost hoy solo expone tools de facturas. Las operaciones sobre autos —listar el parque vehicular, dar de alta un vehículo, corregir sus datos y registrar mantenimientos, inspecciones mecánicas y seguros— solo se pueden hacer desde la UI, una entidad a la vez. Un agente no puede consultar ni cargar esos registros de forma desatendida.

Esta spec agrega seis tools MCP sobre la **API HTTP existente**: `autos_list`, `autos_create`, `autos_update`, `autos_maintenance_register`, `autos_inspection_register` y `autos_insurance_register`. No se agrega esquema, migración ni lógica de negocio nueva: las tools son adaptadores delgados que llaman los endpoints ya usados por el frontend (`/api/autos`, `/api/autos/{id}/maintenances`, `/api/autos/{id}/inspections`, `/api/autos/{id}/services`), de modo que las validaciones de taller, centro de inspección, moneda, cobertura y año de inspección siguen viviendo en `internal/services`.

Las tools de escritura conservan el contrato de la SPEC-145: `dry_run` activado por defecto, compuerta `P40LA_MCP_ALLOW_WRITES` (y `P40LA_MCP_REMOTE_WRITES` en HTTP) y errores accionables con código estable. El binario MCP corre fuera del add-on, por lo que el iHost no suma consumo de RAM; cada llamada hace una o dos peticiones HTTP secuenciales y acotadas.

---

## 2. Cumplimiento de reglas

Reglas del proyecto aplicables a esta spec (índice: `docs/rules/00-index.md`).

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Las tools se implementan en Go con el SDK oficial MCP y el cliente `net/http` existente; sin Node.js ni dependencias nuevas. | `CA-ARCH-STACK` |
| `ARCH-FOLDERS` | Agrega `internal/mcp/tools/autos/`, extiende `internal/mcp/api/` y registra en `internal/mcp/tools/all/register.go`; documentación en `docs/mcp/tools/autos.md`. | `CA-ARCH-FOLDERS` |
| `ARCH-LAYERS` | Las tools no contienen SQL ni reglas de negocio: validan forma, llaman la API y propagan el resultado. Las validaciones de dominio permanecen en `internal/services`. | `CA-ARCH-LAYERS` |
| `ARCH-MODULES` | N/A justificado: el MCP es un adaptador externo, no un módulo de dominio con storage; consume los servicios legacy de autos que la regla no obliga a migrar. | — |
| `ARCH-GO` | Código en inglés, `gofmt`, errores explícitos, configuración por env y `slog` a stderr. | `CA-ARCH-GO` |
| `ARCH-IHOST` | El binario MCP no se ejecuta dentro del add-on; no hay migraciones ni procesos nuevos en el iHost. Las llamadas son secuenciales y sin paralelismo. | `CA-ARCH-IHOST` |
| `ARCH-CONFIG` | N/A justificado: no se modifican `opencode.json` ni las instrucciones del proyecto. | — |
| `UI-SELECT`, `UI-SELECT-ADD`, `UI-NUMBER-INPUT`, `UI-PHONE-INPUT`, `UI-INPUT-TOKENS`, `UI-DARKMODE`, `UI-TOGGLE`, `UI-LIST-PATTERN`, `UI-CREATE-MENU`, `UI-EDIT-LABEL`, `UI-I18N`, `UI-STYLE`, `UI-GUARDRAIL` | N/A justificado: la spec no toca frontend, formularios, listados, i18n ni componentes visuales. | — |
| `UX-BACK`, `UX-MODAL`, `UX-MODAL-CONFIRM`, `UX-DETAIL-ACTIONS`, `UX-DETAIL-READONLY`, `UX-EMPTY`, `UX-LOADING`, `UX-FORMS` | N/A justificado: no hay pantallas, rutas ni modales nuevos. | — |
| `UX-PREREQ` | Antes de registrar un mantenimiento, inspección o seguro, la tool verifica que el auto exista; el seguro verifica además que el servicio esté disponible. El backend revalida taller, centro de inspección, moneda y cobertura. | `CA-UX-PREREQ` |
| `UX-FEEDBACK` | Cada tool devuelve el registro creado/actualizado y, en `dry_run`, el payload planificado con `status: planned`. Los errores incluyen `hint` accionable. | `CA-UX-FEEDBACK` |
| `DATA-SQLITE` | N/A justificado: no se cambia configuración de SQLite; se usan las queries existentes vía API. | — |
| `DATA-MIGRATIONS` | N/A justificado: no hay tablas ni columnas nuevas; `autos`, `auto_maintenances`, `auto_inspections` y `auto_services` ya existen (migraciones 0009, 0042 y 0044). | — |
| `DATA-NO-DROP` | El MCP nunca abre ni reemplaza `data/app.db`; las pruebas usan `/tmp/test-app.db` o una API falsa. | `CA-DATA-NO-DROP` |
| `DATA-PRODCOPY` | Si la validación depende de datos reales, se corre contra una copia `.backup` en `/tmp`; nunca se escribe producción durante la validación. | `CA-DATA-PRODCOPY` |
| `DATA-PROD-WRITE` | Las escrituras reales van por la API, no por SQL directo. No se ejecuta SQL sobre el volumen. | `CA-DATA-PROD-WRITE` |
| `DATA-PROTECTED-SCHEMA` | N/A justificado: no se tocan `debts` ni `debt_bills`. | — |
| `SEC-NO-PII` | Ejemplos y tests usan placas, VIN, pólizas y descripciones sintéticas; logs solo con IDs y conteos. | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | `DEMO-001`, `VIN0000000000000001`, `Taller Demo`, `Aseguradora Demo`, `F000000000000001`. | `CA-SEC-SYNTHETIC` |
| `SEC-PRODCOPY` | Copias de producción solo en `/tmp` y se borran al terminar. | `CA-SEC-PRODCOPY` |
| `SEC-COMMIT` | Revisión manual de `git diff --cached` antes de cada commit. | `CA-SEC-COMMIT` |
| `SEC-HOOKS` | Hooks instalados en el worktree; prohibido `--no-verify` sin autorización. | `CA-SEC-HOOKS` |
| `SEC-INCIDENT` | Aplica el protocolo de `docs/mcp/security.md` si se filtra un secreto o PII. | `CA-SEC-INCIDENT` |
| `SEC-LOGS` | Logs del MCP a stderr, sin payloads sensibles; errores con código y mensaje, sin volcados. | `CA-SEC-LOGS` |
| `TEST-LOCAL` | `go build ./...` y `go test ./...` obligatorios; tests unitarios con API falsa y prueba manual local. | `CA-TEST-LOCAL` |
| `TEST-SERVER` | Antes de avisar al usuario se ejecuta `./scripts/check-server.sh`. | `CA-TEST-SERVER` |
| `TEST-KILLALL` | Uso exclusivo de `killall`, nunca `pkill`. | `CA-TEST-KILLALL` |
| `TEST-TESTUSER` | La prueba autenticada usa `./scripts/create-test-user.sh` y `/tmp/cookies.txt`. | `CA-TEST-TESTUSER` |
| `TEST-USER-FLOW` | Se pregunta 🤖 antes de correr el server y no se cierra la spec sin confirmación de pruebas. | `CA-TEST-USER-FLOW` |
| `TEST-ROBOT` | Las preguntas de flujo usan 🤖. | `CA-TEST-ROBOT` |
| `TEST-CHECKLIST` | Se completa la checklist completa antes de `pending_release`. | `CA-TEST-CHECKLIST` |
| `DELIV-WORKTREE` | La implementación se hace en `../p40la-ihost-spec-151` (`feature/SPEC-151`). | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | Merge, push y deploy requieren confirmación explícita; prohibido `--force`. | `CA-DELIV-CONFIRM` |
| `DELIV-COMMIT` | Commits con formato `SPEC-151: descripción corta`. | `CA-DELIV-COMMIT` |
| `DELIV-RELEASE` | Se sigue el flujo completo de release (merge, tracker, label, issue, limpieza). | `CA-DELIV-RELEASE` |
| `DELIV-CHECKRELEASE` | Se ejecuta `/spec-check-release` antes de declarar terminada la spec. | `CA-DELIV-CHECKRELEASE` |
| `DELIV-HOOKS` | Hooks instalados; sin `--no-verify`; incluye `check-mcp-docs.sh`. | `CA-DELIV-HOOKS` |
| `DELIV-MCP-DOCS` | El mismo commit incluye `docs/mcp/tools/autos.md` y la actualización del catálogo. | `CA-DELIV-MCP-DOCS` |
| `DELIV-MCP-DOCS-SUBAGENT` | La redacción de `docs/mcp/` se delega a un subagente en paralelo y el principal la revisa. | `CA-DELIV-MCP-DOCS-SUBAGENT` |
| `DELIV-DOCKER` | N/A justificado: no se crea ni publica imagen nueva; el binario MCP se compila con `go build`. | — |
| `DELIV-LABELS` | Issue con `spec/draft`; transiciones con labels válidos y cierre solo en release. | `CA-DELIV-LABELS` |
| `DELIV-DEPLOY` | El deploy al iHost lo aprueba el usuario y se verifica `/health` después. | `CA-DELIV-DEPLOY` |
| `DELIV-ESCALATION` | Si la spec queda bloqueada más de 3 días, se documenta y se escala. | `CA-DELIV-ESCALATION` |

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001 — Tool `autos_list`**: lista los autos registrados. Entrada opcional `query` que filtra por marca, modelo, placa o VIN sin distinguir mayúsculas ni acentos. Solo lectura.
2. **REQ-002 — Tool `autos_create`**: crea un auto con `year`, `brand`, `model`, `color` (paleta canónica de `AutoColors`), `icon`, `motor`, `chasis`, `vin` y `placa`. Valida campos obligatorios y año 1900–2100. `dry_run` por defecto `true`.
3. **REQ-003 — Tool `autos_update`**: actualiza un auto existente por `auto_id`. Recibe el mismo conjunto de campos que `autos_create` porque `PUT /api/autos/{id}` reemplaza el registro; `motor`, `chasis`, `vin`, `placa` e `icon` pueden ir vacíos. `dry_run` por defecto `true` y, en ese modo, verifica que el auto exista y devuelve el payload planificado.
4. **REQ-004 — Tool `autos_maintenance_register`**: registra un mantenimiento con `auto_id`, `performed_at` (`YYYY-MM-DD`, no futura), `mileage`, `institution_id` (taller), `currency_id` y `description` opcional. Acepta `items` opcionales (`labor`/`supplies`) con `description` y `amount`; se crean después del mantenimiento. `dry_run` por defecto `true`.
5. **REQ-005 — Tool `autos_inspection_register`**: registra una inspección mecánica con `auto_id`, `performed_at`, `inspection_year`, `institution_id` (centro de inspección), `result` (`approved`/`rejected`) y `notes` opcional. `dry_run` por defecto `true`.
6. **REQ-006 — Tool `autos_insurance_register`**: asocia un seguro a un auto con `auto_id`, `service_id`, `coverage_type` (`daños_a_terceros`/`full_cover`), `policy_number`, `insurer_number` y `certificate` opcional. Antes de escribir verifica que el servicio esté en `GET /api/autos/{id}/available-services`. `dry_run` por defecto `true`.
7. **REQ-007 — Compuerta de escritura**: las cinco tools de escritura solo escriben con `P40LA_MCP_ALLOW_WRITES=true`; en modo HTTP además `P40LA_MCP_REMOTE_WRITES=true`. Sin compuerta, `dry_run=true` funciona y `dry_run=false` devuelve `writes_disabled` con `hint`.
8. **REQ-008 — Registro único**: `internal/mcp/tools/all/register.go` sigue siendo el único punto que conoce el paquete `autos`; `cmd/mcp-server` no cambia.
9. **REQ-009 — Cliente API**: `internal/mcp/api` incorpora los tipos y métodos `ListAutos`, `GetAuto`, `CreateAuto`, `UpdateAuto`, `ListAvailableServices`, `CreateMaintenance`, `CreateMaintenanceItem`, `CreateInspection` y `CreateAutoService`, con timeout y límite de respuesta existentes.
10. **REQ-010 — Errores accionables**: códigos estables `invalid_request`, `invalid_date`, `not_found`, `prerequisite_missing`, `writes_disabled`, `auth_failed` y `upstream_error`; mensajes en español con `hint`.
11. **REQ-011 — Documentación**: `docs/mcp/tools/autos.md` con entrada, salida, ejemplos sintéticos, reglas y compuerta; el catálogo `docs/mcp/tools/README.md` lista las seis tools.
12. **REQ-012 — Anotaciones MCP**: `autos_list` con `readOnlyHint=true`; las tools de escritura con `destructiveHint=false`, `openWorldHint=false`. `autos_update` con `idempotentHint=true`; las de alta con `idempotentHint=false` porque repetir la llamada crea otro registro.
13. **REQ-013 — Corregir `ListAvailableServices`**: `GET /api/autos/{id}/available-services` falla con `sql: expected 23 destination arguments in Scan, not 25` porque la query no selecciona `automation_account_id` ni `is_insurance`. Se alinea la query con `serviceColumns` (columnas calificadas para el join) y se agrega un test de storage. Sin este arreglo `autos_insurance_register` no puede validar el prerrequisito.

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-014 — Ítems de mantenimiento**: `autos_maintenance_register` devuelve el detalle con costos de mano de obra, insumos y total; si un ítem falla, informa `partial: true` con el `maintenance_id` creado y el error del ítem.
2. **REQ-015 — Filtro por auto**: `autos_list` acepta `auto_id` para devolver un solo vehículo con su detalle.
3. **REQ-016 — Idempotencia de seguros**: si el `service_id` ya está asociado al auto, la tool devuelve `already_exists` en lugar de un error genérico del backend.
4. **REQ-017 — Listado de inspecciones**: las tools de inspección devuelven el cronograma calculado (`schedule`) que expone la API al crear, para informar la próxima fecha.

### 3.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-018 — Tool `autos_delete`**: eliminación con confirmación explícita. Fuera de alcance de esta spec.
2. **REQ-019 — Tool `autos_maintenance_list`**: historial de mantenimientos por auto. Fuera de alcance.
3. **REQ-020 — Adjuntos**: subir comprobantes o certificados desde el MCP. Fuera de alcance por tamaño y tipo de contenido.

### 3.4 Requerimientos No Funcionales

- **Rendimiento**: `autos_list` debe responder en menos de 1 segundo en LAN con hasta 100 autos; las altas hacen como máximo 1 + N peticiones secuenciales, donde N es la cantidad de ítems de mantenimiento (máximo 20).
- **Memoria**: el binario MCP no corre dentro del add-on; no aumenta el RSS del server Go del iHost.
- **Seguridad**: autenticación con la API key global; nunca se registran payloads con datos personales ni números de póliza en logs.
- **Almacenamiento**: cero migraciones y cero archivos nuevos en el volumen.
- **Disponibilidad**: si la API no responde, la tool devuelve `upstream_error` con hint de `/health`.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

- Endpoints existentes verificados en `internal/api/routes.go`:
  - `GET/POST /api/autos`, `GET/PUT /api/autos/{id}`
  - `GET/POST /api/autos/{id}/maintenances`, `POST /api/autos/{id}/maintenances/{maintenance_id}/items`
  - `GET/POST /api/autos/{id}/inspections`
  - `GET/POST /api/autos/{id}/services`, `GET /api/autos/{id}/available-services`
- Validaciones existentes:
  - `AutoService.Create`: marca, modelo, color canónico y año 1900–2100 (`internal/services/auto.go`).
  - `AutoMaintenanceService.validate`: fecha no futura, kilometraje no negativo, moneda existente, institución de categoría taller (`internal/services/auto_maintenance.go`).
  - `AutoInspectionService.validate`: fecha no futura, resultado `approved`/`rejected`, año entre `auto.Year+2` y `performedAt.Year()+1`, institución de categoría inspección mecánica (`internal/services/auto_inspection.go`).
  - `AutoServiceService.Create`: cobertura `daños_a_terceros`/`full_cover`, póliza y número de aseguradora obligatorios (`internal/services/auto_service.go`).
- Patrón MCP vigente: `internal/mcp/tools/bills` (Toolset, `Register`, `dry_run`, `writes_disabled`, `jsonResult`, `toolError`, `apiError`) y registro en `internal/mcp/tools/all/register.go`.

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A. Tools MCP que llaman la API HTTP existente | Cero cambios de backend, reutiliza validaciones y auditoría; consistente con SPEC-145 | Una petición extra por operación | ✅ Seleccionada |
| B. Tools MCP que abren SQLite directamente | Menos latencia | Duplica validaciones, riesgo de corrupción WAL, rompe ARCH-LAYERS y el modelo de despliegue | ❌ Rechazada |
| C. Nuevos endpoints masivos específicos para autos | Contratos a medida | Duplica endpoints que ya existen; más superficie de mantenimiento | ❌ Rechazada |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001 — Adaptador delgado sobre la API**
- **Contexto**: los seis flujos ya existen en la UI y en los servicios.
- **Decisión**: las tools solo validan forma, verifican prerrequisitos y llaman la API.
- **Consecuencias**: las reglas de negocio quedan en un solo lugar; cualquier cambio de validación se refleja sin tocar el MCP.

**ADR-002 — `dry_run` para altas**
- **Contexto**: la compuerta de escritura de SPEC-145 exige `dry_run` en tools que escriben.
- **Decisión**: `dry_run=true` hace preflight local (campos, formato de fecha, existencia del auto y disponibilidad del servicio) y devuelve `status: planned`; el backend revalida al escribir.
- **Consecuencias**: el agente puede mostrar el plan al usuario antes de confirmar; el preflight no garantiza categorías de institución, que solo el backend conoce.

**ADR-003 — Sin migraciones**
- **Contexto**: todas las tablas y endpoints necesarios existen.
- **Decisión**: no se agrega esquema ni se modifica el backend.
- **Consecuencias**: release de bajo riesgo; el alcance queda acotado a `internal/mcp/` y `docs/mcp/`.

---

## 5. Diseño Técnico

### 5.1 Diagrama de arquitectura

```
Cliente MCP (OpenCode / Claude)
        |
        v
internal/mcp/tools/autos  --(HTTP + API key)-->  p40la-ihost API
        |                                             |
        |                                             v
internal/mcp/api (tipos y métodos)            internal/services
                                                      |
                                                      v
                                                   SQLite
```

### 5.2 Componentes

#### 5.2.1 `internal/mcp/tools/autos`
- **Responsabilidad**: definir las seis tools, sus schemas, validación de forma, `dry_run` y respuestas.
- **Interfaz**: `Register(server, client, cfg)` y un `Toolset` con `now func() time.Time` para tests.
- **Dependencias**: `internal/mcp/api`, `internal/mcp/config`, SDK MCP.
- **Ubicación**: `internal/mcp/tools/autos/`.

#### 5.2.2 `internal/mcp/api` (extensión)
- **Responsabilidad**: tipos `Auto`, `Maintenance`, `Inspection`, `AutoService` y métodos HTTP contra los endpoints existentes.
- **Interfaz**: se agregan métodos a la interfaz `API`; el cliente real y los fakes de test los implementan.
- **Dependencias**: `net/http`, `encoding/json`.
- **Ubicación**: `internal/mcp/api/autos.go` y `types.go`.

#### 5.2.3 `docs/mcp/tools/autos.md`
- **Responsabilidad**: documentar las seis tools con ejemplos sintéticos y la compuerta de escritura.
- **Ubicación**: `docs/mcp/tools/autos.md`.

#### 5.2.4 `internal/storage/auto_service.go`
- **Responsabilidad**: corregir `ListAvailableServices` para que seleccione las mismas 25 columnas que `scanServices` espera (`automation_account_id` e `is_insurance`), usando columnas calificadas por el join.
- **Interfaz**: `GET /api/autos/{id}/available-services` deja de devolver `internal_error`.
- **Ubicación**: `internal/storage/service.go` (`serviceColumnsQualified`) y `internal/storage/auto_service.go`.

### 5.3 Modelo de datos

Sin cambios de esquema. Se reutilizan:

```
Auto: id, year, model, brand, color, icon, motor, chasis, vin, placa
AutoMaintenance: id, auto_id, performed_at, mileage, institution_id, description, currency_id
AutoMaintenanceItem: id, maintenance_id, item_type (labor|supplies), description, amount
AutoInspection: id, auto_id, performed_at, inspection_year, institution_id, result (approved|rejected), notes
AutoService: id, auto_id, service_id, coverage_type, policy_number, certificate, insurer_number
```

### 5.4 APIs / Contratos

#### Tool `autos_list`

**Entrada**:
```json
{
  "query": "demo",
  "auto_id": 0
}
```

**Salida**:
```json
{
  "count": 1,
  "autos": [
    {
      "id": 1,
      "year": 2024,
      "brand": "Marca Demo",
      "model": "Modelo Demo",
      "color": "white",
      "icon": "vehicle",
      "placa": "DEMO-001",
      "vin": "VIN0000000000000001"
    }
  ]
}
```

#### Tool `autos_create`

**Entrada**:
```json
{
  "year": 2024,
  "brand": "Marca Demo",
  "model": "Modelo Demo",
  "color": "white",
  "placa": "DEMO-001",
  "dry_run": true
}
```

**Salida (dry_run)**:
```json
{
  "status": "planned",
  "dry_run": true,
  "auto": { "year": 2024, "brand": "Marca Demo", "model": "Modelo Demo", "color": "white", "placa": "DEMO-001", "icon": "vehicle" }
}
```

Con `dry_run=false` devuelve `status: "created"` y el auto con `id` y timestamps.

#### Tool `autos_update`

**Entrada**:
```json
{
  "auto_id": 1,
  "year": 2024,
  "brand": "Marca Demo",
  "model": "Modelo Demo Actualizado",
  "color": "black",
  "placa": "DEMO-001",
  "dry_run": true
}
```

**Salida**: `status: planned` o `updated` con el auto resultante.

#### Tool `autos_maintenance_register`

**Entrada**:
```json
{
  "auto_id": 1,
  "performed_at": "2026-11-20",
  "mileage": 45000,
  "institution_id": 7,
  "currency_id": 1,
  "description": "Cambio de aceite y filtros",
  "items": [
    { "item_type": "labor", "description": "Mano de obra", "amount": 500.00 },
    { "item_type": "supplies", "description": "Aceite y filtros", "amount": 1200.00 }
  ],
  "dry_run": true
}
```

**Salida**: `status: planned` o `created`, `maintenance` con `id`, costos `labor_cost`, `supplies_cost`, `total_cost`, `items` y `partial` si algún ítem falló.

#### Tool `autos_inspection_register`

**Entrada**:
```json
{
  "auto_id": 1,
  "performed_at": "2026-11-20",
  "inspection_year": 2026,
  "institution_id": 9,
  "result": "approved",
  "notes": "Inspección demo aprobada",
  "dry_run": true
}
```

**Salida**: `status: planned` o `created`, `inspection` y `schedule` con `next_due_year`, `next_due_date`, `status`, `days_until_due`.

#### Tool `autos_insurance_register`

**Entrada**:
```json
{
  "auto_id": 1,
  "service_id": 12,
  "coverage_type": "full_cover",
  "policy_number": "POL-DEMO-0001",
  "insurer_number": "ASEG-DEMO-001",
  "certificate": "CERT-DEMO-001",
  "dry_run": true
}
```

**Salida**: `status: planned` o `created` con el seguro y los datos del servicio.

#### Errores

```json
{
  "error": "prerequisite_missing",
  "message": "el servicio no está disponible para asociar al auto",
  "hint": "Verificá service_id con GET /api/autos/{id}/available-services."
}
```

### 5.5 Dependencias

- **Internas**: `internal/mcp/api`, `internal/mcp/config`, `internal/mcp/registry`, `internal/mcp/tools/all`.
- **Externas**: `github.com/modelcontextprotocol/go-sdk/mcp` (ya presente). Sin dependencias nuevas.

---

## 6. Criterios de Aceptación

> IDs canónicos por dominio (`CA-<PREFIJO>-<NOMBRE>`) para auditoría con grep.

### 6.1 Funcionales

- [x] CA-001: `autos_list` devuelve todos los autos y filtra por `query` en marca, modelo, placa y VIN.
- [x] CA-002: `autos_create` con `dry_run=true` no escribe y devuelve `status: planned`; con `dry_run=false` y compuerta activa crea el auto.
- [x] CA-003: `autos_update` actualiza el auto por `auto_id` y devuelve el registro resultante; un ID inexistente devuelve `not_found`.
- [x] CA-004: `autos_maintenance_register` crea el mantenimiento y sus ítems; devuelve costos calculados.
- [x] CA-005: `autos_maintenance_register` rechaza fecha futura, kilometraje negativo y moneda inexistente con `invalid_request`.
- [x] CA-006: `autos_inspection_register` crea la inspección y devuelve el cronograma; respeta las reglas de año y resultado.
- [x] CA-007: `autos_insurance_register` verifica `available-services` y crea la asociación con cobertura válida.
- [x] CA-008: las cinco tools de escritura devuelven `writes_disabled` cuando la compuerta está apagada y `dry_run=false`.
- [x] CA-009: todas las tools aparecen en el servidor MCP tras registrarse en `all/register.go`.
- [x] CA-010: los errores de la API se traducen a códigos estables con `hint`.
- [x] CA-011: `GET /api/autos/{id}/available-services` responde 200 con la lista de servicios de seguro no asociados, sin error de escaneo.
- [x] CA-ARCH-STACK: no se agregan dependencias externas nuevas.
- [x] CA-ARCH-FOLDERS: el código nuevo vive en `internal/mcp/tools/autos/` y `internal/mcp/api/autos.go`.
- [x] CA-ARCH-LAYERS: las tools no contienen SQL ni validaciones de dominio duplicadas.
- [x] CA-ARCH-GO: `gofmt` y `go vet` sin cambios pendientes.
- [x] CA-ARCH-IHOST: no hay migraciones ni procesos nuevos en el iHost.
- [x] CA-UX-PREREQ: se valida la existencia del auto y la disponibilidad del servicio antes de escribir.
- [x] CA-UX-FEEDBACK: `dry_run` devuelve `status: planned` y las escrituras devuelven el registro creado.
- [x] CA-DATA-NO-DROP: ninguna prueba abre `data/app.db`.
- [x] CA-DATA-PRODCOPY: si se valida con datos, se usa copia en `/tmp`.
- [x] CA-DATA-PROD-WRITE: no se ejecuta SQL sobre el volumen.
- [x] CA-SEC-NO-PII: ejemplos y tests sin datos reales.
- [x] CA-SEC-SYNTHETIC: se usan `DEMO-001`, `Taller Demo`, `Aseguradora Demo`.
- [x] CA-SEC-PRODCOPY: copias temporales borradas al terminar.
- [x] CA-SEC-COMMIT: `git diff --cached` revisado.
- [x] CA-SEC-HOOKS: hooks instalados y sin `--no-verify`.
- [x] CA-SEC-INCIDENT: protocolo documentado en `docs/mcp/security.md`.
- [x] CA-SEC-LOGS: logs sin payloads sensibles.
- [x] CA-TEST-LOCAL: `go build ./...` y `go test ./...` pasan.
- [ ] CA-TEST-SERVER: `./scripts/check-server.sh` pasa antes de avisar.
- [x] CA-TEST-KILLALL: se usa `killall` si hace falta detener el server.
- [x] CA-TEST-TESTUSER: prueba autenticada con `create-test-user.sh` si aplica.
- [ ] CA-TEST-USER-FLOW: el usuario confirma las pruebas manuales antes del release.
- [ ] CA-TEST-ROBOT: preguntas de flujo con 🤖.
- [ ] CA-TEST-CHECKLIST: checklist de `60-testing.md` completa.
- [ ] CA-DELIV-WORKTREE: trabajo en `feature/SPEC-151`.
- [ ] CA-DELIV-CONFIRM: merge/push con confirmación explícita.
- [ ] CA-DELIV-COMMIT: mensajes `SPEC-151: ...`.
- [ ] CA-DELIV-RELEASE: release completo en el mismo ciclo.
- [ ] CA-DELIV-CHECKRELEASE: `/spec-check-release` sin colgadas.
- [ ] CA-DELIV-HOOKS: hooks activos y `check-mcp-docs.sh` en verde.
- [ ] CA-DELIV-MCP-DOCS: `docs/mcp/` actualizado en el mismo commit.
- [ ] CA-DELIV-MCP-DOCS-SUBAGENT: documentación redactada por subagente y revisada.
- [ ] CA-DELIV-LABELS: issue con `spec/draft` y transición a `spec/released`.
- [ ] CA-DELIV-DEPLOY: deploy aprobado por el usuario.
- [ ] CA-DELIV-ESCALATION: bloqueo > 3 días documentado.

### 6.2 No funcionales

- [ ] CA-NF-001: `autos_list` responde en menos de 1 segundo con 100 autos en LAN.
- [ ] CA-NF-002: el binario MCP no suma procesos al add-on del iHost.
- [ ] CA-NF-003: una llamada no hace más de 21 peticiones HTTP secuenciales (1 + 20 ítems).

### 6.3 Testing

- **Unit tests**: `internal/mcp/tools/autos` con API falsa: listado y filtro, dry-run, compuerta, prerequisitos, mapeo de errores, ítems de mantenimiento.
- **Integration tests**: cliente API contra `httptest.Server` para serialización de payloads y códigos de error.
- **E2E**: server local + MCP stdio con `autos_list` y un `dry_run` de alta.
- **Carga/Performance**: listado con 100 autos sintéticos.

---

## 7. Plan de Implementación

### 7.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Tipos y métodos en `internal/mcp/api` | 0.5 días | Ninguna |
| 2 | Paquete `internal/mcp/tools/autos` y registro | 1 día | Fase 1 |
| 3 | Tests unitarios y de contrato | 0.5 días | Fase 2 |
| 4 | Documentación `docs/mcp/tools/autos.md` y catálogo | 0.5 días | Fase 2 |
| 5 | Verificación local y release | 0.5 días | Fases 3 y 4 |

### 7.2 Milestones

1. **MVP**: seis tools registradas, compuerta de escritura y tests unitarios.
2. **V1.0**: documentación publicada, release y deploy aprobado.

### 7.3 Estado de implementación

- `go build ./...`, `go vet ./internal/mcp/... ./internal/storage/...` y `go test ./...` en verde.
- Tests nuevos: `internal/mcp/tools/autos` (unidad y protocolo MCP en memoria), `internal/mcp/api` (cliente HTTP) y `internal/storage` (regresión de `ListAvailableServices`).
- Prueba E2E real contra un server temporal en `/tmp` con DATA_DIR aislado y API key sintética: `autos_list`, `autos_create` (dry_run y escritura), `autos_update`, `autos_maintenance_register` con dos ítems (total 1700), `autos_inspection_register` con cronograma, `autos_insurance_register` y rechazo `already_exists` al repetir el seguro.
- El bug preexistente de `GET /api/autos/{id}/available-services` se reprodujo en la E2E y quedó corregido con test de regresión.
- Los temporales de la prueba se eliminaron; no se tocó `data/app.db` ni producción.

---

## 8. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Duplicar registros al repetir un alta | Media | Medio | `idempotentHint=false` y respuesta con `id`; documentar que el alta no es idempotente |
| El preflight de `dry_run` no detecta categoría de institución inválida | Alta | Bajo | El backend rechaza con `institution_category`; la tool propaga el error con hint |
| Crecimiento del paquete de tools | Baja | Bajo | Registro por dominio en `all/register.go`; sin lógica compartida duplicada |
| Fuga de datos en logs | Baja | Alto | Logs solo con IDs y conteos; sin payloads (`SEC-LOGS`) |
| Dependencia de endpoints no estables | Baja | Medio | Tests de contrato con `httptest` y verificación en local |

---

## 9. Notas y Referencias

- Reglas aplicadas: `docs/rules/10-architecture.md`, `20-ui.md`, `30-ux.md`, `40-data.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.
- Specs relacionadas: SPEC-124 (mantenimientos e inspecciones), SPEC-145 (servidor MCP y tools de facturas), SPEC-147 (MCP HTTP/OAuth), SPEC-118 (colores de autos).
- Endpoints: `internal/api/routes.go` líneas 146–173 y 230–235.
- Documentación MCP: `docs/mcp/README.md`, `docs/mcp/tools/README.md`.

---

## 10. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-24 | opencode | Creación inicial de la especificación |
| 2026-11-24 | opencode | Implementación de las seis tools, fix de `ListAvailableServices`, tests y documentación |
| 2026-11-24 | opencode | Verificación E2E real con API temporal: alta, edición, mantenimiento, inspección, seguro y duplicado |
