---
title: "CRUD de Notificaciones en módulo Pensión Alimenticia"
id: "SPEC-046"
status: "draft"
author: "p40la-ihost-team"
created: "2026-09-02"
updated: "2026-09-02"
github_issue: 46
---

# CRUD de Notificaciones en módulo Pensión Alimenticia

**ID**: SPEC-046  
**Estado**: draft  
**Autor**: p40la-ihost-team  
**Creado**: 2026-09-02  
**Actualizado**: 2026-09-02

---

## 1. Resumen Ejecutivo

El módulo de **Pensión Alimenticia** (base estructural creada en SPEC-044) tiene la sección **Notificaciones** navegando actualmente a una página placeholder. Esta spec convierte esa sección en un **CRUD completo de notificaciones**: registrar, listar, editar y eliminar destinatarios de notificación, cada uno con **nombre**, **email** (ambos requeridos, email único) y un flag **active** (booleano) que permite habilitar o deshabilitar el envío de mails a ese destinatario.

La página muestra la lista de registros en cards y, debajo, un **histórico de emails enviados** que por ahora es una sección placeholder (futura característica de envío), quedando fuera de alcance de esta spec. El foco es exclusivamente el CRUD de registros.

El CRUD replica el patrón ya establecido en el proyecto (Homes, Autos, Services, Hijos): página de cards con `CreateMenu` (3 puntos en el header), menú de acciones de 3 puntos por card (Editar/Eliminar), modal de confirmación para eliminar y `EmptyCard` cuando no hay registros. Respeto de iHost: una tabla SQLite pequeña, sin dependencias nuevas.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Tabla SQLite `notifications` con campos: id, name, email (UNIQUE), active (boolean, default true), created_at, updated_at.
2. **REQ-002**: API REST CRUD protegida por auth: GET /api/notifications, GET /api/notifications/:id, POST /api/notifications, PUT /api/notifications/:id, DELETE /api/notifications/:id.
3. **REQ-003**: La sección Notificaciones (`/pension/notificaciones`) deja de ser placeholder y muestra la página de listado de registros con cards en grid (patrón `AutosPage`).
4. **REQ-004**: Formulario de creación/edición (`NotificationFormPage`) con campos: Nombre (requerido), Email (requerido, válido) y Active (switch o toggle).
5. **REQ-005**: En cada card del listado se muestra el **nombre** como título y el **email** como subtítulo, más un indicador visual del estado **active** (habilitado/deshabilitado).
6. **REQ-006**: Cada card tiene menú de acciones de 3 puntos (CardMenu) con opciones **Editar** y **Eliminar**.
7. **REQ-007**: Eliminar usa el modal de confirmación existente (DeleteModal).
8. **REQ-008**: `EmptyCard` cuando no hay registros, con botón para crear el primero.
9. **REQ-009**: Header de la página con título "Notificaciones" y `CreateMenu` (ícono 3 puntos) con opción de crear registro.
10. **REQ-010**: Debajo del listado, una sección **Histórico de emails enviados** que por ahora muestra un placeholder (futura característica de envío), sin lógica de negocio en esta spec.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-011**: Validación de formulario: nombre no vacío, email no vacío y formato válido. Toast de error si falla la validación.
2. **REQ-012**: Validación de unicidad de email: si el email ya existe en otro registro, toast de error y no se guarda (validación backend + frontend).
3. **REQ-013**: El flag **active** (default true) controla si el destinatario recibe o no emails; la lógica de envío futura debe consultar `active = 1` para decidir el envío.
4. **REQ-014**: Toast de éxito/error al crear, editar y eliminar.
5. **REQ-015**: Loading spinner mientras se cargan los datos.
6. **REQ-016**: Etiquetas traducibles vía i18n es/en en `frontend/public/i18n/` (fuente de verdad).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-017**: Ordenar los registros por fecha de creación (más recientes primero) en el listado.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: CRUD ligero con queries simples sobre SQLite.
- **Seguridad**: Misma autenticación existente (`authMiddleware` en todas las rutas de la API).
- **Almacenamiento**: Una tabla SQLite nueva (`notifications`), registros de ~100-200 bytes cada uno.
- **Disponibilidad**: Sin cambios en health checks ni schedulers existentes.
- **iHost**: Sin dependencias nuevas; solo archivos estáticos React pre-build (Vite) servidos por el backend Go; sin Node.js en runtime.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se analizaron los patrones CRUD existentes para replicarlos exactamente:

- **SPEC-045 (CRUD de Hijos)**: referencia más reciente de CRUD en el mismo módulo — migración SQLite, `storage` + `service` + `handlers` en Go, rutas en `internal/api/routes.go`, página de listado y formulario.
- **SPEC-024 (CRUD de Autos)**: referencia completa de CRUD — página de listado (`AutosPage.tsx`) y formulario (`AutoFormPage.tsx`).
- **SPEC-044 (Menú Pensión Alimenticia)**: ya definió la sección **Notificaciones** navegando a `/pension/notificaciones` con página placeholder (`PensionPage.tsx`). Esta spec reemplaza el placeholder por el CRUD real.
- **`frontend/src/pages/AutosPage.tsx`**: patrón de listado con cards en grid, `CreateMenu` (3 puntos) en el header, `CardMenu` (3 puntos por card con Editar/Eliminar), `DeleteModal` y `EmptyCard`.
- **`internal/api/routes.go`**: patrón de registro de rutas con `authMiddleware`.
- **`migrations/`**: última migración es `0015`; la nueva será `0016`.
- **i18n**: fuente de verdad en `frontend/public/i18n/{es,en}.json` (se sobrescriben en el build), espejo en `frontend/src/i18n/{es,en}.json`. Ya existen las claves `pension.notifications` y `pension.notifications_empty`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| CRUD completo (DB + API + listado + formulario) | Funcionalidad completa, consistente con Autos/Hijos | Más archivos a crear | ✅ Seleccionada |
| Solo frontend con datos en memoria/JSON | Rápido de implementar | No persiste, no escala | ❌ Rechazada |
| Mantener placeholder e implementar más adelante | Cero trabajo | No cumple el requerimiento | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Email único
- **Contexto**: El registro identifica a un destinatario de notificación; duplicar el email generaría envíos duplicados en el futuro histórico.
- **Decisión**: Columna `email` con constraint `UNIQUE` en SQLite, validación en el service (backend) antes del insert/update y mensaje de error claro en el frontend.
- **Consecuencias**: Índice único implícito sobre email; queries de verificación previas a guardar; toast de error específico para el caso de duplicado.

**ADR-002**: Nombre del modelo `notifications` (plural)
- **Contexto**: Los destinatarios de notificación podrían reutilizarse fuera del módulo Pensión Alimenticia (alertas por email, SPEC-033).
- **Decisión**: Tabla `notifications` a nivel global (no `pension_notifications`), permitiendo reuso futuro del modelo por el sistema de alertas.
- **Consecuencias**: Simplicidad; el modelo queda disponible para futuras features de envío sin migraciones adicionales.

**ADR-003**: Campo `active` para habilitar/deshabilitar envío
- **Contexto**: El usuario necesita decidir fácilmente en la lógica de envío de mails si un destinatario recibe o no emails, sin eliminar el registro.
- **Decisión**: Columna `active` (BOOLEAN/INTEGER 0-1, default 1) en la tabla `notifications`. La futura lógica de envío debe filtrar por `active = 1`. El formulario expone un switch para editar el estado y la card lo muestra visualmente.
- **Consecuencias**: Un campo booleano más en la API y el formulario; permite apagar destinatarios sin borrarlos; evita duplicar destinatarios con y sin envío.

**ADR-004**: Histórico de emails enviados como placeholder
- **Contexto**: El diseño muestra un histórico debajo de la lista, pero el envío de emails es una característica futura.
- **Decisión**: Se renderiza una sección con título "Histórico de emails enviados" y EmptyCard descriptivo, sin tablas ni lógica adicional en esta spec.
- **Consecuencias**: Se respeta el diseño solicitado sin implementar features fuera de alcance; se documenta como futura característica.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[NotificationFormPage] --(API REST)--> [NotificationHandlers] --> [NotificationService] --> [NotificationStorage]
      |                                                                                         |
      v                                                                                         v
[Frontend: NotificationsPage (cards)]                                                 [SQLite notifications]
```

### 4.2 Componentes

#### 4.2.1 Backend - NotificationStorage
- **Responsabilidad**: Queries CRUD contra SQLite.
- **Interfaz**: List, GetByID, Create, Update, Delete, ExistsByEmail.
- **Dependencias**: `database/sql`.
- **Ubicación**: `internal/storage/notification.go`.

#### 4.2.2 Backend - NotificationService
- **Responsabilidad**: Validación de negocio (name y email requeridos; email con formato válido; email único; active booleano con default true).
- **Interfaz**: Lista, obtiene, crea, actualiza, elimina.
- **Dependencias**: NotificationStorage.
- **Ubicación**: `internal/services/notification.go`.

#### 4.2.3 Backend - NotificationHandlers
- **Responsabilidad**: HTTP handlers para la API REST.
- **Interfaz**: HandleListNotifications, HandleGetNotification, HandleCreateNotification, HandleUpdateNotification, HandleDeleteNotification.
- **Dependencias**: NotificationService.
- **Ubicación**: `internal/api/notification_handlers.go`.

#### 4.2.4 Frontend - NotificationsPage
- **Responsabilidad**: Listado de registros en cards con grid, menú de 3 puntos por card (Editar/Eliminar), `CreateMenu` en header, `EmptyCard` cuando no hay registros, indicador visual del estado active por card, y sección inferior con placeholder de histórico de emails enviados.
- **Interfaz**: Componente React funcional.
- **Dependencias**: api, Icon, CreateMenu, CardMenu, DeleteModal, EmptyCard, LoadingSpinner.
- **Ubicación**: `frontend/src/pages/NotificationsPage.tsx`.

#### 4.2.5 Frontend - NotificationFormPage
- **Responsabilidad**: Formulario de creación/edición con campos Nombre, Email y Active (switch).
- **Interfaz**: Componente React funcional, maneja `:id` param.
- **Dependencias**: api, useToast, useNavigate.
- **Ubicación**: `frontend/src/pages/NotificationFormPage.tsx`.

#### 4.2.6 App / Sidebar
- **Responsabilidad**: Registrar rutas `/pension/notificaciones` → `NotificationsPage` y `/pension/notificaciones/new` + `/pension/notificaciones/edit/:id` → `NotificationFormPage`. El sidebar ya tiene la entrada "Notificaciones" (SPEC-044).
- **Ubicación**: `frontend/src/App.tsx`.

### 4.3 Modelo de datos

```
Entidad: notification
- id: INTEGER PRIMARY KEY AUTOINCREMENT
- name: TEXT NOT NULL (nombre del destinatario, ej: "María Pérez")
- email: TEXT NOT NULL UNIQUE (email del destinatario, ej: "maria@example.com")
- active: INTEGER NOT NULL DEFAULT 1 (booleano 0/1; 1 = recibe emails, 0 = deshabilitado)
- created_at: DATETIME DEFAULT CURRENT_TIMESTAMP
- updated_at: DATETIME DEFAULT CURRENT_TIMESTAMP

Relaciones: Ninguna por ahora (destinatarios de notificación independientes; reusable por el sistema de alertas)
```

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/notifications`

**Response 200**:
```json
[
  {
    "id": 1,
    "name": "María Pérez",
    "email": "maria@example.com",
    "active": true,
    "created_at": "2026-09-02T10:00:00Z",
    "updated_at": "2026-09-02T10:00:00Z"
  }
]
```

#### Endpoint: `GET /api/notifications/:id`

**Response 200**: Objeto `notification` individual
**Response 404**: `{"error": "not_found", "message": "Registro no encontrado"}`

#### Endpoint: `POST /api/notifications`

**Request**:
```json
{
  "name": "María Pérez",
  "email": "maria@example.com",
  "active": true
}
```

**Response 201**: Objeto `notification` creado con ID (active default true si se omite)
**Response 400**: `{"error": "validation_error", "message": "Nombre y email son requeridos"}`
**Response 409**: `{"error": "duplicate_email", "message": "El email ya está registrado"}`

#### Endpoint: `PUT /api/notifications/:id`

**Request**: Mismo schema que POST
**Response 200**: Objeto `notification` actualizado
**Response 409**: `{"error": "duplicate_email", "message": "El email ya está registrado"}`

#### Endpoint: `DELETE /api/notifications/:id`

**Response 200**: `{"message": "Registro eliminado"}`

### 4.5 Dependencias

- **Internas**: `internal/api/routes.go` (registro de rutas), `cmd/server/main.go` (wiring del handler), `frontend/src/App.tsx` (rutas React), `frontend/src/pages/PensionPage.tsx` (se deja de usar para Notificaciones), componentes UI existentes (CreateMenu, CardMenu, DeleteModal, Toast, Icon, LoadingSpinner, EmptyCard), archivos i18n.
- **Externas**: Ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado el sidebar, cuando hago clic en "Pensión alimenticia → Notificaciones", se navega a `/pension/notificaciones` y se muestra la página de listado de registros.
- [ ] CA-002: Dado el listado vacío, cuando no hay registros, se muestra el EmptyCard con título, descripción y botón para crear el primero.
- [ ] CA-003: Dado el listado con registros, se muestran cards en grid con el nombre como título y el email como subtítulo, más un indicador visual del estado active (habilitado/deshabilitado).
- [ ] CA-004: Dado una card, cuando hago hover, aparece el menú de 3 puntos con opciones **Editar** y **Eliminar**.
- [ ] CA-005: Dado el formulario de creación, cuando completo Nombre y Email y guardo, se crea el registro (active default true) y se redirige al listado.
- [ ] CA-006: Dado el formulario con campos obligatorios vacíos, al guardar se muestra toast de error y no se crea el registro.
- [ ] CA-007: Dado el formulario con email de formato inválido, al guardar se muestra toast de error.
- [ ] CA-008: Dado un email ya registrado en otro registro, al crear/editar se muestra toast de error y no se guarda (backend + frontend).
- [ ] CA-009: Dado el formulario de edición, cuando modifico campos y/o el switch Active y guardo, se actualiza el registro.
- [ ] CA-009a: Dado un destinatario con active=false, la card lo muestra visualmente como deshabilitado y la futura lógica de envío no le envía mails (el filtro por active se documenta para la feature de envío).
- [ ] CA-010: Dado el menú de acciones, cuando selecciono "Eliminar", aparece el modal de confirmación y al confirmar se elimina el registro.
- [ ] CA-011: Debajo del listado se muestra la sección **Histórico de emails enviados** con placeholder (futura característica), sin errores.
- [ ] CA-012: Las etiquetas se muestran traducidas en español e inglés según el idioma seleccionado.

### 5.2 No funcionales

- [ ] CA-NF-001: El build de Vite (`npm run build` en `frontend/`) compila sin errores.
- [ ] CA-NF-002: No se agregan dependencias nuevas al proyecto.
- [ ] CA-NF-003: El backend compila (`go build ./...`) y los tests de Go pasan (`go test ./...`).

### 5.3 Testing

- **Unit tests**: Validación del service (campos vacíos, email inválido, email duplicado, active booleano).
- **Integration tests**: CRUD completo contra SQLite en memoria, incluido el constraint UNIQUE de email y el default de active.
- **E2E tests**: Flujo de usuario: crear registro → listar → editar (incluido toggle active) → eliminar; caso de email duplicado.
- **Carga/Performance**: Validar tiempo de respuesta del listado con 50+ registros (queries simples, sin joins).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración SQLite `0016_create_notifications` (up/down) + modelo Go | 15 min | Ninguna |
| 2 | NotificationStorage + NotificationService + NotificationHandlers | 30 min | Fase 1 |
| 3 | Registro de rutas API en `routes.go` + wiring en `main.go` | 15 min | Fase 2 |
| 4 | Tipo `Notification` en `frontend/src/types/index.ts` + métodos en api client | 15 min | Fase 3 |
| 5 | NotificationsPage (listado con cards, menú 3 puntos, indicador active, EmptyCard, sección histórico placeholder) | 30 min | Fase 4 |
| 6 | NotificationFormPage (formulario con validación, switch Active y manejo de email duplicado) | 30 min | Fase 4 |
| 7 | Rutas React en `App.tsx` (reemplazar placeholder de Notificaciones) | 15 min | Fase 5, 6 |
| 8 | Claves i18n es/en (`frontend/public/i18n/` + `frontend/src/i18n/`) y build | 15 min | Fase 7 |
| 9 | Pruebas locales (`go test`, `npm run build`, levantar server, validación manual) | 30 min | Fase 8 |
| **Total** | | **~3.25 horas** | |

### 6.2 Milestones

1. **MVP**: CRUD funcional completo de notificaciones con cards (Fases 1-8).
2. **V1.0**: MVP + pruebas locales + validación manual del usuario.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Colisión con SPEC-045 (CRUD de Hijos) si ambas corren en paralelo sobre la migración `0016` | Media | Medio | Coordinar el número de migración; si SPEC-045 ya usó `0016`, esta spec usa `0017` |
| Email duplicado no detectado (race condition) | Baja | Medio | Constraint UNIQUE en SQLite como última línea de defensa + validación en service |
| Claves i18n editadas solo en `frontend/src/i18n/` se pierden en el build | Media | Medio | Editar SIEMPRE `frontend/public/i18n/` (fuente de verdad) y correr `npm run build`; verificar con `curl` (ver AGENTS.md) |
| Reemplazar el placeholder de SPEC-044 rompe la ruta `/pension/notificaciones` | Baja | Medio | Mantener la misma ruta; verificar navegación del resto de secciones de Pensión |

## 8. Notas y Referencias

- Patrón CRUD de referencia: `docs/specs/SPEC-045-crud-hijos-pension.md`, `docs/specs/SPEC-024-autos-crud.md`, `frontend/src/pages/AutosPage.tsx`, `frontend/src/pages/AutoFormPage.tsx`.
- Backend CRUD: `internal/storage/auto.go`, `internal/services/auto.go`, `internal/api/auto_handlers.go`.
- Base del módulo Pensión Alimenticia: `docs/specs/SPEC-044-pension-alimenticia-sidebar.md` (sección Notificaciones en `/pension/notificaciones`).
- Componentes UI: `frontend/src/components/CreateMenu.tsx`, `CardMenu.tsx`, `DeleteModal.tsx`, `EmptyCard` (inline en `AutosPage.tsx`).
- Migraciones: `migrations/` (última: `0015`; la nueva será `0016` o `0017` según colisión con SPEC-045).
- Reglas de i18n: fuente de verdad en `frontend/public/i18n/` (ver AGENTS.md, sección "Reglas críticas").

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-02 | p40la-ihost-team | Creación inicial de la especificación |