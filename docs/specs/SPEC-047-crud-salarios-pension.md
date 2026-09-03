---
title: "CRUD de Salarios en módulo Pensión Alimenticia"
id: "SPEC-047"
status: "in_progress"
author: "p40la-ihost-team"
created: "2026-09-02"
updated: "2026-09-02"
github_issue: 47
---

# CRUD de Salarios en módulo Pensión Alimenticia

**ID**: SPEC-047  
**Estado**: in_progress  
**Autor**: p40la-ihost-team  
**Creado**: 2026-09-02  
**Actualizado**: 2026-09-02

---

## 1. Resumen Ejecutivo

El módulo de **Pensión Alimenticia** (base estructural creada en SPEC-044) tiene la sección **Salarios** navegando actualmente a una página placeholder. Esta spec convierte esa sección en un **CRUD completo de salarios**: registrar, listar, editar y eliminar cada ingreso/salario del usuario, con **empleador/fuente** (requerido), **monto** (requerido), **moneda** (select de las monedas ya existentes en el sistema), **día de pago** (requerido), **activo** (booleano, default `true`) y **nota** (opcional).

La página muestra la lista de registros en cards con el patrón ya establecido en el proyecto (Homes, Autos, Services, Hijos, Notificaciones): página de cards con `CreateMenu` (3 puntos en el header), menú de acciones de 3 puntos por card (Editar/Eliminar), modal de confirmación para eliminar y `EmptyCard` cuando no hay registros.

Respeto de iHost: una tabla SQLite pequeña, sin dependencias nuevas. El select de moneda reutiliza la tabla `currencies` existente (seed actual: NIO y USD) y el patrón ya usado en `ServiceFormPage`.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Tabla SQLite `salaries` con campos: id, employer (TEXT NOT NULL), amount (REAL NOT NULL), currency_id (INTEGER NOT NULL, FK → currencies.id), payment_day (INTEGER NOT NULL, 1-31), active (INTEGER NOT NULL DEFAULT 1), note (TEXT NULL), created_at, updated_at.
2. **REQ-002**: API REST CRUD protegida por auth: GET /api/salaries, GET /api/salaries/:id, POST /api/salaries, PUT /api/salaries/:id, DELETE /api/salaries/:id.
3. **REQ-003**: La sección Salarios (`/pension/salarios`) deja de ser placeholder y muestra la página de listado de registros con cards en grid (patrón `AutosPage`/`HijosPage`).
4. **REQ-004**: Formulario de creación/edición (`SalaryFormPage`) con campos: **Empleador / Fuente** (requerido, texto), **Monto** (requerido, numérico), **Moneda** (requerido, select de `currencies` existentes), **Día de pago** (requerido, 1-31), **Activo** (switch o toggle, default true) y **Nota** (opcional, textarea).
5. **REQ-005**: En cada card del listado se muestra el **empleador/fuente** como título, el **monto con símbolo de la moneda** y el **día de pago** como subtítulo, más un indicador visual del estado **active** (activo/inactivo).
6. **REQ-006**: Cada card tiene menú de acciones de 3 puntos (CardMenu) con opciones **Editar** y **Eliminar**.
7. **REQ-007**: Eliminar usa el modal de confirmación existente (DeleteModal).
8. **REQ-008**: `EmptyCard` cuando no hay registros, con botón para crear el primero.
9. **REQ-009**: Header de la página con título "Salarios" y `CreateMenu` (ícono 3 puntos) con opción de crear registro.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-010**: Validación de formulario: empleador no vacío, monto > 0, día de pago entre 1 y 31. Toast de error si falla la validación.
2. **REQ-011**: Validación de moneda: el `currency_id` debe existir en la tabla `currencies` (validación backend + frontend).
3. **REQ-012**: El flag **active** (default true) se persiste y se muestra visualmente en la card.
4. **REQ-013**: Toast de éxito/error al crear, editar y eliminar.
5. **REQ-014**: Loading spinner mientras se cargan los datos.
6. **REQ-015**: Etiquetas traducibles vía i18n es/en en `frontend/public/i18n/` (fuente de verdad), incluyendo el namespace `salaries` y actualizando `pension.salaries_empty` si es necesario.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-016**: Ordenar los registros por fecha de creación (más recientes primero) en el listado.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: CRUD ligero con queries simples sobre SQLite; el símbolo de moneda se resuelve en el frontend desde el store de `currencies`.
- **Seguridad**: Misma autenticación existente (`authMiddleware` en todas las rutas de la API).
- **Almacenamiento**: Una tabla SQLite nueva (`salaries`), registros de ~150-300 bytes cada uno.
- **Disponibilidad**: Sin cambios en health checks ni schedulers existentes.
- **iHost**: Sin dependencias nuevas; solo archivos estáticos React pre-build (Vite) servidos por el backend Go; sin Node.js en runtime.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se analizaron los patrones CRUD existentes para replicarlos exactamente:

- **SPEC-045 (CRUD de Hijos)**: referencia más reciente de CRUD en el mismo módulo — migración SQLite, `storage` + `service` + `handlers` en Go, rutas en `internal/api/routes.go`, página de listado y formulario.
- **SPEC-046 (CRUD de Notificaciones)**: referencia de CRUD con flag booleano `active` (default true) y switch en el formulario.
- **SPEC-044 (Menú Pensión Alimenticia)**: ya definió la sección **Salarios** navegando a `/pension/salarios` con página placeholder (`PensionPage.tsx`). Esta spec reemplaza el placeholder por el CRUD real.
- **Tabla `currencies`**: existe desde la migración `0002` con seed de NIO y USD; el select se implementa igual que en `ServiceFormPage.tsx` (componente `Select` con `currencies` desde el store).
- **`internal/models/service.go`**: referencia de validación de día de mes (`billing_day` con rango 1-31 en `internal/services/service.go:162-165`).
- **`frontend/src/pages/HijosPage.tsx`**: patrón de listado con cards en grid, `CreateMenu` (3 puntos) en el header, `CardMenu` (3 puntos por card con Editar/Eliminar), `DeleteModal` y `EmptyCard`.
- **`migrations/`**: última migración es `0016` (children, SPEC-045); la nueva será `0017` o `0018` según colisión con SPEC-046.
- **i18n**: fuente de verdad en `frontend/public/i18n/{es,en}.json` (se sobrescriben en el build). Ya existen `pension.salaries` y `pension.salaries_empty`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| CRUD completo (DB + API + listado + formulario) | Funcionalidad completa, consistente con Hijos/Autos | Más archivos a crear | ✅ Seleccionada |
| Solo frontend con datos en memoria/JSON | Rápido de implementar | No persiste, no escala | ❌ Rechazada |
| Mantener placeholder e implementar más adelante | Cero trabajo | No cumple el requerimiento | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Reuso de la tabla `currencies` para el select de moneda
- **Contexto**: El formulario requiere una moneda; el sistema ya tiene la tabla `currencies` (NIO/USD) gestionada en Settings y usada por `services.currency_id`.
- **Decisión**: `salaries.currency_id` es FK a `currencies.id`; el frontend llena el `Select` desde el store `appStore.currencies`, igual que `ServiceFormPage`.
- **Consecuencias**: Cero duplicación de datos de moneda; el select muestra `code (symbol)` consistente con el resto del sistema; si el usuario agrega monedas en Settings, aparecen automáticamente.

**ADR-002**: `payment_day` como entero 1-31
- **Contexto**: "Día de pago" indica el día del mes en que se cobra el salario.
- **Decisión**: Columna `payment_day INTEGER NOT NULL` con validación 1-31 (mismo patrón que `billing_day` de services). Se persiste el día numérico; el formato/label se maneja en i18n y frontend.
- **Consecuencias**: Validación simple y consistente; sin dependencias de fechas por ahora (a futuro los registros mensuales de SPEC-044 podrían usarlo).

**ADR-003**: Campo `active` para habilitar/deshabilitar
- **Contexto**: El usuario necesita marcar un salario como vigente o no sin eliminarlo; por defecto debe estar activo.
- **Decisión**: Columna `active` (INTEGER 0-1, default 1). El formulario expone un switch y la card muestra el estado visualmente. Sólo la nota es opcional; todos los demás campos son requeridos.
- **Consecuencias**: Un campo booleano más en la API y el formulario; permite apagar un salario sin borrarlo.

**ADR-004**: Un solo campo `employer` para "Empleador / Fuente"
- **Contexto**: El requerimiento pide "Empleador / Fuente" como un único concepto (quién paga o de dónde proviene el ingreso).
- **Decisión**: Columna `employer TEXT NOT NULL` con label i18n "Empleador / Fuente" ("Employer / Source" en inglés).
- **Consecuencias**: Modelo simple; si a futuro se necesitan separar empleador y fuente, será una migración posterior.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[SalaryFormPage] --(API REST)--> [SalaryHandlers] --> [SalaryService] --> [SalaryStorage]
      |                                                                              |
      v                                                                              v
[Frontend: SalariesPage (cards)]                                            [SQLite salaries]
      |                                                                        |
      +-- Select Moneda (currencies store) <--FK--> [SQLite currencies]
```

### 4.2 Componentes

#### 4.2.1 Backend - SalaryStorage
- **Responsabilidad**: Queries CRUD contra SQLite.
- **Interfaz**: List, GetByID, Create, Update, Delete.
- **Dependencias**: `database/sql`.
- **Ubicación**: `internal/storage/salary.go`.

#### 4.2.2 Backend - SalaryService
- **Responsabilidad**: Validación de negocio (employer no vacío; amount > 0; payment_day 1-31; currency_id existente; active booleano con default true).
- **Interfaz**: Lista, obtiene, crea, actualiza, elimina.
- **Dependencias**: SalaryStorage, CurrencyStorage (para validar currency_id).
- **Ubicación**: `internal/services/salary.go`.

#### 4.2.3 Backend - SalaryHandlers
- **Responsabilidad**: HTTP handlers para la API REST.
- **Interfaz**: HandleListSalaries, HandleGetSalary, HandleCreateSalary, HandleUpdateSalary, HandleDeleteSalary.
- **Dependencias**: SalaryService.
- **Ubicación**: `internal/api/salary_handlers.go`.

#### 4.2.4 Frontend - SalariesPage
- **Responsabilidad**: Listado de registros en cards con grid, menú de 3 puntos por card (Editar/Eliminar), `CreateMenu` en header, `EmptyCard` cuando no hay registros, indicador visual del estado active por card, símbolo de moneda desde el store.
- **Interfaz**: Componente React funcional.
- **Dependencias**: api, Icon, CreateMenu, CardMenu, DeleteModal, EmptyCard, LoadingSpinner, appStore (currencies).
- **Ubicación**: `frontend/src/pages/SalariesPage.tsx`.

#### 4.2.5 Frontend - SalaryFormPage
- **Responsabilidad**: Formulario de creación/edición con campos Empleador/Fuente, Monto, Moneda (Select), Día de pago, Activo (switch) y Nota (textarea).
- **Interfaz**: Componente React funcional, maneja `:id` param.
- **Dependencias**: api, useToast, useNavigate, Select, appStore (currencies).
- **Ubicación**: `frontend/src/pages/SalaryFormPage.tsx`.

#### 4.2.6 App / Sidebar
- **Responsabilidad**: Registrar rutas `/pension/salarios` → `SalariesPage` y `/pension/salarios/new` + `/pension/salarios/edit/:id` → `SalaryFormPage`. El sidebar ya tiene la entrada "Salarios" (SPEC-044).
- **Ubicación**: `frontend/src/App.tsx`.

### 4.3 Modelo de datos

```
Entidad: salary
- id: INTEGER PRIMARY KEY AUTOINCREMENT
- employer: TEXT NOT NULL (empleador/fuente, ej: "Empresa XYZ")
- amount: REAL NOT NULL (monto, ej: 15000.00)
- currency_id: INTEGER NOT NULL (FK → currencies.id, ej: 1 = NIO)
- payment_day: INTEGER NOT NULL (día de pago 1-31, ej: 15)
- active: INTEGER NOT NULL DEFAULT 1 (booleano 0/1; 1 = activo)
- note: TEXT NULL (nota opcional, ej: "Pago mensual")
- created_at: DATETIME DEFAULT CURRENT_TIMESTAMP
- updated_at: DATETIME DEFAULT CURRENT_TIMESTAMP

Relaciones: salary.currency_id → currencies.id (N:1)
```

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/salaries`

**Response 200**:
```json
[
  {
    "id": 1,
    "employer": "Empresa XYZ",
    "amount": 15000.00,
    "currency_id": 1,
    "payment_day": 15,
    "active": true,
    "note": "Pago mensual",
    "created_at": "2026-09-02T10:00:00Z",
    "updated_at": "2026-09-02T10:00:00Z"
  }
]
```

#### Endpoint: `GET /api/salaries/:id`

**Response 200**: Objeto `salary` individual
**Response 404**: `{"error": "not_found", "message": "Registro no encontrado"}`

#### Endpoint: `POST /api/salaries`

**Request**:
```json
{
  "employer": "Empresa XYZ",
  "amount": 15000.00,
  "currency_id": 1,
  "payment_day": 15,
  "active": true,
  "note": "Pago mensual"
}
```

**Response 201**: Objeto `salary` creado con ID (active default true si se omite)
**Response 400**: `{"error": "validation_error", "message": "Empleador, monto, moneda y día de pago son requeridos"}`
**Response 400**: `{"error": "invalid_payment_day", "message": "El día de pago debe estar entre 1 y 31"}`
**Response 400**: `{"error": "invalid_currency", "message": "La moneda no existe"}`

#### Endpoint: `PUT /api/salaries/:id`

**Request**: Mismo schema que POST
**Response 200**: Objeto `salary` actualizado
**Response 400**: Mismos errores de validación que POST

#### Endpoint: `DELETE /api/salaries/:id`

**Response 200**: `{"message": "Registro eliminado"}`

### 4.5 Dependencias

- **Internas**: `internal/api/routes.go` (registro de rutas), `cmd/server/main.go` (wiring del handler), `frontend/src/App.tsx` (rutas React), `frontend/src/pages/PensionPage.tsx` (se deja de usar para Salarios), `internal/storage/currency.go` (validación de currency_id), componentes UI existentes (CreateMenu, CardMenu, DeleteModal, Toast, Icon, LoadingSpinner, EmptyCard, Select), archivos i18n.
- **Externas**: Ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado el sidebar, cuando hago clic en "Pensión alimenticia → Salarios", se navega a `/pension/salarios` y se muestra la página de listado de registros.
- [ ] CA-002: Dado el listado vacío, cuando no hay registros, se muestra el EmptyCard con título, descripción y botón para crear el primero.
- [ ] CA-003: Dado el listado con registros, se muestran cards en grid con el empleador/fuente como título, el monto con símbolo de moneda y el día de pago como subtítulo, más un indicador visual del estado active (activo/inactivo).
- [ ] CA-004: Dado una card, cuando hago hover, aparece el menú de 3 puntos con opciones **Editar** y **Eliminar**.
- [ ] CA-005: Dado el formulario de creación, cuando completo Empleador/Fuente, Monto, Moneda (select de monedas existentes) y Día de pago, y guardo, se crea el registro (active default true) y se redirige al listado.
- [ ] CA-006: Dado el formulario con campos obligatorios vacíos, al guardar se muestra toast de error y no se crea el registro.
- [ ] CA-007: Dado el formulario con día de pago fuera de rango (0 o 32), al guardar se muestra toast de error.
- [ ] CA-008: Dado el formulario de edición, cuando modifico campos y/o el switch Activo y guardo, se actualiza el registro.
- [ ] CA-009: Dado un salario con active=false, la card lo muestra visualmente como inactivo.
- [ ] CA-010: Dado el menú de acciones, cuando selecciono "Eliminar", aparece el modal de confirmación y al confirmar se elimina el registro.
- [ ] CA-011: El select de moneda lista las monedas existentes del sistema (currencies store) mostrando `code (symbol)`; las monedas agregadas en Settings aparecen en el select.
- [ ] CA-012: Las etiquetas se muestran traducidas en español e inglés según el idioma seleccionado.

### 5.2 No funcionales

- [ ] CA-NF-001: El build de Vite (`npm run build` en `frontend/`) compila sin errores.
- [ ] CA-NF-002: No se agregan dependencias nuevas al proyecto.
- [ ] CA-NF-003: El backend compila (`go build ./...`) y los tests de Go pasan (`go test ./...`).

### 5.3 Testing

- **Unit tests**: Validación del service (empleador vacío, monto inválido, día fuera de rango, moneda inexistente, active booleano).
- **Integration tests**: CRUD completo contra SQLite en memoria, incluido el FK a currencies y el default de active.
- **E2E tests**: Flujo de usuario: crear salario (con select de moneda) → listar con símbolo → editar (incluido toggle active) → eliminar; casos de validación.
- **Carga/Performance**: Validar tiempo de respuesta del listado con 50+ registros (queries simples, sin joins).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración SQLite `0017_create_salaries` (up/down) + modelo Go | 15 min | Ninguna |
| 2 | SalaryStorage + SalaryService + SalaryHandlers | 30 min | Fase 1 |
| 3 | Registro de rutas API en `routes.go` + wiring en `main.go` | 15 min | Fase 2 |
| 4 | Tipo `Salary` en `frontend/src/types/index.ts` + métodos en api client | 15 min | Fase 3 |
| 5 | SalariesPage (listado con cards, menú 3 puntos, indicador active, EmptyCard) | 30 min | Fase 4 |
| 6 | SalaryFormPage (formulario con validación, Select de moneda, switch Active) | 30 min | Fase 4 |
| 7 | Rutas React en `App.tsx` (reemplazar placeholder de Salarios) | 15 min | Fase 5, 6 |
| 8 | Claves i18n es/en (`frontend/public/i18n/` + `frontend/src/i18n/`) y build | 15 min | Fase 7 |
| 9 | Pruebas locales (`go test`, `npm run build`, levantar server, validación manual) | 30 min | Fase 8 |
| **Total** | | **~3.25 horas** | |

### 6.2 Milestones

1. **MVP**: CRUD funcional completo de salarios con cards (Fases 1-8).
2. **V1.0**: MVP + pruebas locales + validación manual del usuario.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Colisión de número de migración con SPEC-046 (CRUD de Notificaciones) si ambas corren en paralelo | Media | Medio | Coordinar el número: si SPEC-046 usa `0017`, esta spec usa `0018` |
| El select de moneda muestra monedas eliminadas (soft delete) | Baja | Medio | Listar solo `deleted_at IS NULL` (mismo criterio que `CurrencyStorage.List`) |
| Monto con decimales y precisión | Media | Bajo | Usar REAL en SQLite y formateo con `toFixed(2)` en el frontend; documentar límites |
| Claves i18n editadas solo en `frontend/src/i18n/` se pierden en el build | Media | Medio | Editar SIEMPRE `frontend/public/i18n/` (fuente de verdad) y correr `npm run build`; verificar con `curl` (ver AGENTS.md) |
| Reemplazar el placeholder de SPEC-044 rompe la ruta `/pension/salarios` | Baja | Medio | Mantener la misma ruta; verificar navegación del resto de secciones de Pensión |

## 8. Notas y Referencias

- Patrón CRUD de referencia: `docs/specs/SPEC-045-crud-hijos-pension.md`, `docs/specs/SPEC-046-crud-notificaciones-pension.md`, `docs/specs/SPEC-024-autos-crud.md`, `frontend/src/pages/HijosPage.tsx`, `frontend/src/pages/AutosPage.tsx`.
- Backend CRUD: `internal/storage/child.go`, `internal/services/child.go`, `internal/api/child_handlers.go`.
- Select de moneda: `frontend/src/pages/ServiceFormPage.tsx` (componente `Select` + store `currencies`).
- Validación de día de mes: `internal/services/service.go` (rango 1-31 para `billing_day`).
- Base del módulo Pensión Alimenticia: `docs/specs/SPEC-044-pension-alimenticia-sidebar.md` (sección Salarios en `/pension/salarios`).
- Componentes UI: `frontend/src/components/CreateMenu.tsx`, `CardMenu.tsx`, `DeleteModal.tsx`, `EmptyCard` (inline en `HijosPage.tsx`), `Select` (usado en `ServiceFormPage.tsx`).
- Migraciones: `migrations/` (última: `0016`; la nueva será `0017` o `0018` según colisión con SPEC-046).
- Reglas de i18n: fuente de verdad en `frontend/public/i18n/` (ver AGENTS.md, sección "Reglas críticas").

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-02 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-09-02 | p40la-ihost-team | Estado cambiado a in_progress; inicio de desarrollo en worktree feature/SPEC-047 |