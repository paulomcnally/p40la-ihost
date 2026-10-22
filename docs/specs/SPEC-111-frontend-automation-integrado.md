---
title: "Frontend unificado de automation y selector de cuentas"
id: "SPEC-111"
status: "in_progress"
author: "opencode"
created: "2026-10-21"
updated: "2026-10-21"
github_issue: 114
---

# Frontend unificado de automation y selector de cuentas

**ID**: SPEC-111  
**Estado**: in_progress  
**Autor**: opencode  
**Creado**: 2026-10-21  
**Actualizado**: 2026-10-21

---

## 1. Resumen Ejecutivo

SPEC-110 dejó el backend de automation integrado en `p40la-ihost`: cuentas, credenciales, plugins, facturas, logs, scheduler y sync en proceso. Pero la SPA todavía no expone esas funciones: el único acceso es `Configuración → Automation`, que sigue apuntando al proyecto viejo (`automation_base_url` + `automation_api_key`), y `ServiceFormPage` pide un **ID numérico de cuenta** que el usuario debe averiguar a mano.

Esta spec incorpora la UI de automation a la SPA existente y reemplaza el ID manual por un **selector de cuentas**. Al terminar, el usuario podrá administrar cuentas y plugins, consultar facturas y logs, y vincular un servicio a una cuenta desde un desplegable, todo con el mismo login y el mismo diseño que el resto de p40la.

Impacto en iHost: nulo en runtime. Todo es React compilado a estáticos; no se agregan dependencias ni procesos. La UI consume los endpoints ya implementados en SPEC-110.

Consideraciones de UI obligatorias:
- Todo `input`/`select`/`textarea` usa tokens del tema (`bg-card`, `text-text`, `text-text-secondary`); nada de colores hardcodeados. Verificar en darkmode.
- Las páginas de detalle y formularios usan la flecha atrás del header vía `BACK_ROUTES`. No se crean links "← Título" en el contenido.

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Nueva sección **Automation** en el sidebar (`/automation`) con listado de cuentas en cards, menú de 3 puntos (editar/eliminar) y `CreateMenu` para crear. Estado vacío con `EmptyCard`.
2. **REQ-002**: Formulario de cuenta (`/automation/new`, `/automation/edit/:id`) con: institución (select de instituciones existentes), identificador, etiqueta, plugin (select del catálogo), credenciales dinámicas según `GET /api/automation/plugins/{name}/schema`, URL de webhook y programación (habilitado, hora, frecuencia, intervalo/días).
3. **REQ-003**: En edición, las credenciales existentes no se muestran ni se envían si no se modifican. El formulario indica qué campos secretos se conservan vacíos y permite revelarlas con contraseña mediante `POST /api/automation/accounts/{id}/credentials:reveal`.
4. **REQ-004**: Página de detalle (`/automation/:id`) con resumen de la cuenta, servicios vinculados, últimas facturas consultadas (`GET .../bills`), logs de entrega (`GET .../webhook/logs`) y acciones: consultar facturas (`POST .../bills:fetch`), sincronizar servicio vinculado (`POST /api/services/{id}/sync`), editar y eliminar.
5. **REQ-005**: Catálogo de plugins (`/automation/plugins`) de solo lectura: nombre, versión, origen, descripción y estado.
6. **REQ-006**: En `ServiceFormPage`, reemplazar el input numérico `automation_account_id` por un `Select` alimentado por `GET /api/automation/accounts?for=service`. La opción vacía representa `NULL` (servicio sin automation). Mostrar ayuda y, si no hay cuentas, un acceso directo a crear una.
7. **REQ-007**: Eliminar de la SPA la configuración legacy `Configuración → Automation` (`automation_base_url`/`automation_api_key`), su tarjeta en `SettingsPage`, su ruta y su página. La sección Automation del sidebar es el único acceso.
8. **REQ-008**: Registrar en `BACK_ROUTES`: `/automation/:id`, `/automation/new`, `/automation/edit/:id` → `/automation`; `/automation/plugins` → `/automation`.
9. **REQ-009**: Todas las cadenas visibles nuevas en `frontend/public/i18n/es.json` y `frontend/public/i18n/en.json` (fuente de verdad). Sin texto hardcodeado salvo nombres propios de plugins/proveedores.
10. **REQ-010**: Manejo de errores y estados: loading con `LoadingSpinner`, listas vacías, 409 al borrar una cuenta en uso (mensaje claro), 400/502 al consultar facturas, y toasts de éxito/error.
11. **REQ-011**: Formularios y modales verificados en darkmode (texto/placeholders legibles) y en móvil.
12. **REQ-018**: La UI de automation **no muestra** el campo `webhook_url` (ni en alta/edición ni en detalle). El valor existente se preserva internamente al editar como fallback de transición; no se elimina la columna, el fallback del backend ni el endpoint externo `POST /webhooks/{uuid}` del servicio.
13. **REQ-019**: Las acciones de la página de detalle de cuenta (editar, revelar credenciales, consultar ahora y eliminar) se agrupan en el menú de 3 puntos (`CardMenu`) del patrón de UI existente, no como botones sueltos en el header.
14. **REQ-020**: El formulario de alta/edición de cuentas se segmenta en 3 tabs: **Información**, **Plugin** y **Frecuencia**. Solo se muestra la sección activa; guardar/cancelar quedan siempre disponibles y, si la validación falla, se abre la tab del campo inválido.
15. **REQ-021**: **Guardar** permanece deshabilitado mientras falten campos requeridos (institución, identificador, plugin, credenciales requeridas en alta y programación válida si está habilitada). Las tabs con error muestran un punto rojo y un texto de ayuda junto al botón.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-012**: Copiar la URL de webhook de la cuenta con un botón y confirmación visual.
2. **REQ-013**: Mostrar en el detalle de la cuenta los servicios vinculados (filtrados por `automation_account_id` desde `GET /api/services`) con acceso a su detalle y botón de sync.
3. **REQ-014**: Resumen legible de la programación ("Diaria 07:00", "Cada 2 días 07:00", "Semanal lun 07:00", "Mensual día 15 07:00") y última ejecución.
4. **REQ-015**: En el listado de cuentas, mostrar institución, identificador/etiqueta, plugin y estado de programación.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-016**: Buscador en el listado de cuentas cuando supere 10 registros.
2. **REQ-017**: Mostrar el contador de facturas y el último estado en la card de la cuenta.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: la UI no agrega dependencias; el bundle no debe crecer más de ~30 KB gzip. Las listas se cargan bajo demanda.
- **Seguridad**: las credenciales nunca se guardan en estado global ni en `localStorage`; el reveal exige contraseña y se descarta al cerrar el modal.
- **Accesibilidad**: labels asociados, foco visible, botones con `title`/`aria-label` donde corresponda.
- **iHost**: solo estáticos; sin Node en runtime.
- **Compatibilidad**: React 19 + React Router 7 + Tailwind existentes.

---

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto

- Backend disponible (SPEC-110):
  - `GET /api/automation/accounts` y `?for=service`.
  - `POST/PUT/DELETE /api/automation/accounts[/{id}]`.
  - `POST /api/automation/accounts/{id}/credentials:reveal`.
  - `GET /api/automation/accounts/{id}`.
  - `GET /api/automation/accounts/{id}/bills`, `POST .../bills:fetch`.
  - `GET/PUT /api/automation/accounts/{id}/webhook`.
  - `GET /api/automation/accounts/{id}/webhook/logs`.
  - `GET /api/automation/plugins`, `GET /api/automation/plugins/{name}/schema`.
- El esquema de credenciales es dinámico (`plugins.CredentialField`: `key`, `label`, `type`, `required`, `secret`); la UI no debe conocer campos por proveedor.
- El campo `webhook_url` de la cuenta sigue existiendo como fallback de transición, pero en la UI se presenta como opcional y se explica que el vínculo preferido es el selector del servicio.
- `GET /api/services` ya devuelve `automation_account_id`, por lo que el detalle puede mostrar servicios vinculados sin endpoint nuevo.

### 3.2 Decisiones

**ADR-001: UI dinámica de credenciales**
- La página de cuenta carga el schema del plugin seleccionado y renderiza los campos. Para `secret`, usa `type="password"` y no rellena valores existentes.
- En edición, si un campo secreto queda vacío, no se envía en el JSON de credenciales (el backend conserva el valor). Si el usuario escribe, se envía solo ese campo.
- El reveal usa un modal que pide la contraseña del usuario logueado; el resultado se muestra en un textarea de solo lectura y se descarta al cerrar.

**ADR-002: Selector de cuentas en servicios**
- Se usa el componente `Select` existente con `searchable` y opción vacía `""` → `null`.
- El endpoint `?for=service` devuelve `{id, label}` ya armado (institución · identificador · plugin), evitando lógica de presentación en el cliente.
- Si no hay cuentas, se muestra un texto con enlace a `/automation/new`; no se bloquea el alta del servicio.

**ADR-003: Retiro de la configuración legacy**
- `SettingsAutomationPage` y su tarjeta se eliminan de la SPA. El backend mantiene las claves por compatibilidad hasta SPEC-112, pero la UI ya no las usa.
- Se elimina el estado `automationConfigured` de `SettingsPage` y la clave `automation` de la lista de ajustes avanzados.

**ADR-004: Patrón de listado y detalle**
- Listado: cards + `CardMenu` + `CreateMenu` + `EmptyCard`, igual que `HomesPage`.
- Detalle: header con flecha atrás (`BACK_ROUTES`), tarjetas de resumen, tablas/listas simples y modales para acciones destructivas o reveal.
- Formulario: una sola columna en móvil, dos en desktop; inputs con tokens del tema.

**ADR-005: Sin backend nuevo**
- Esta spec no agrega endpoints ni migraciones. Si durante la implementación aparece una necesidad no cubierta, se actualiza SPEC-110/esta spec antes de escribir código.

---

## 4. Diseño Técnico

### 4.1 Archivos

| Archivo | Cambio |
|---|---|
| `frontend/src/types/index.ts` | Tipos `AutomationAccount`, `AutomationAccountOption`, `AutomationPlugin`, `AutomationBill`, `AutomationWebhookLog`, `AutomationCredentialField`, `AutomationAccountInput`. |
| `frontend/src/api/index.ts` | Grupo `automation` con list/get/create/update/delete/options/schema/credentials/bills/fetch/logs/webhook. |
| `frontend/src/pages/AutomationPage.tsx` | Listado de cuentas. |
| `frontend/src/pages/AutomationFormPage.tsx` | Alta/edición. |
| `frontend/src/pages/AutomationDetailPage.tsx` | Detalle, facturas, logs, acciones. |
| `frontend/src/pages/AutomationPluginsPage.tsx` | Catálogo de plugins. |
| `frontend/src/App.tsx` | Rutas nuevas. |
| `frontend/src/components/Sidebar.tsx` | Ítem Automation. |
| `frontend/src/components/DashboardLayout.tsx` | `BACK_ROUTES`. |
| `frontend/src/pages/ServiceFormPage.tsx` | Selector de cuentas. |
| `frontend/src/pages/SettingsPage.tsx` | Quitar tarjeta/estado legacy. |
| `frontend/src/pages/SettingsAutomationPage.tsx` | Eliminar. |
| `frontend/public/i18n/es.json`, `en.json` | Claves nuevas y limpieza de las legacy. |

### 4.2 Rutas

```
/automation                 → AutomationPage
/automation/new             → AutomationFormPage
/automation/edit/:id        → AutomationFormPage
/automation/plugins         → AutomationPluginsPage
/automation/:id             → AutomationDetailPage
```

### 4.3 API cliente

```ts
automation: {
  accounts: (forService?: boolean) => get<AutomationAccount[] | AutomationAccountOption[]>(
    forService ? '/api/automation/accounts?for=service' : '/api/automation/accounts'),
  get: (id) => get<AutomationAccount>(`/api/automation/accounts/${id}`),
  create: (body) => post<AutomationAccount>('/api/automation/accounts', body),
  update: (id, body) => put<AutomationAccount>(`/api/automation/accounts/${id}`, body),
  delete: (id) => del(`/api/automation/accounts/${id}`),
  revealCredentials: (id, password) => post<{credentials: string}>(`.../credentials:reveal`, {password}),
  plugins: () => get<AutomationPlugin[]>('/api/automation/plugins'),
  pluginSchema: (name) => get<AutomationCredentialField[]>(`/api/automation/plugins/${name}/schema`),
  bills: (id) => get<AutomationBill[]>(`.../bills`),
  fetchBills: (id) => post<{bills: AutomationBillItem[]}>(`.../bills:fetch`, {}),
  logs: (id, limit=20, offset=0) => get<AutomationWebhookLog[]>(`.../webhook/logs?limit=&offset=`),
  updateWebhook: (id, body) => put<AutomationAccount>(`.../webhook`, body),
}
```

### 4.4 Formulario de cuenta

Campos:
- **Institución** (`Select`, requerido). Si no hay instituciones, mostrar aviso con enlace a `/institutions/new`.
- **Identificador** (texto, requerido).
- **Etiqueta** (texto, opcional).
- **Plugin** (`Select` del catálogo, requerido). Al cambiar, recargar schema y limpiar credenciales.
- **Credenciales** (dinámicas): `text`/`password` según schema; requeridas si `required` y no existe valor previo.
- **Programación**: switch habilitado, hora (`input type="time"`), frecuencia (`daily`, `every_n_days`, `weekly`, `monthly`), intervalo (solo `every_n_days`), días (semanal: checkboxes lun–dom; mensual: número 1–28).
- En edición: credenciales secretas vacías = conservar; texto de ayuda.

Validaciones cliente:
- institución y plugin requeridos; identificador no vacío.
- hora `HH:MM` si la programación está habilitada.
- intervalo ≥ 1; día mensual 1–28; al menos un día si es semanal.

### 4.5 Detalle

- Card resumen: institución, identificador, etiqueta, plugin/versión, programación, última ejecución, webhook URL con copiar.
- Servicios vinculados (P1): nombre, institución, estado, botón sync.
- Facturas: últimas 20, columnas estado, plugin, versión, error, fecha; botón "Consultar ahora".
- Logs: últimas 20, estado, respuesta, fecha; paginación simple "Ver más".
- Acciones: editar, eliminar (con `DeleteModal`), revelar credenciales.

### 4.6 i18n

Nuevo namespace `automation` en `es.json`/`en.json` con subclaves: `title`, `subtitle`, `create`, `edit`, `empty`, `empty_desc`, `accounts`, `plugins`, `plugin`, `institution`, `identifier`, `label`, `credentials`, `credentials_keep`, `credentials_reveal`, `password`, `password_desc`, `webhook`, `schedule`, `schedule_enabled`, `time`, `frequency`, `interval`, `days`, `bills`, `logs`, `fetch`, `linked_services`, `delete_in_use`, etc.
Se eliminan `settings.automation.*` y `services.automation_account_placeholder`/`_hint`, reemplazadas por `services.automation_account_hint` y `automation.*`.

### 4.7 Impacto en iHost

Sin cambios de runtime ni de DB. El build de Vite sigue generando estáticos; no se agregan dependencias.

---

## 5. Criterios de Aceptación

- [ ] **CA-001**: `/automation` lista cuentas con cards, menú de acciones y `CreateMenu`; con 0 cuentas muestra `EmptyCard` y botón de alta.
- [ ] **CA-002**: Se puede crear una cuenta con institución, identificador, plugin, credenciales, webhook y programación; se persiste y aparece en el listado.
- [ ] **CA-003**: Se puede editar una cuenta sin reenviar credenciales secretas; si se dejan vacías, se conservan.
- [ ] **CA-004**: El reveal de credenciales exige contraseña y muestra el JSON; al cerrar, no queda en el estado.
- [ ] **CA-005**: El detalle muestra facturas y logs, y "Consultar ahora" ejecuta el plugin y actualiza la lista.
- [ ] **CA-006**: Borrar una cuenta en uso devuelve 409 y la UI muestra un mensaje claro sin eliminarla.
- [ ] **CA-007**: `/automation/plugins` muestra los 5 plugins compilados con versión y estado.
- [ ] **CA-008**: `ServiceFormPage` muestra un `Select` de cuentas; guarda `null` con la opción vacía y el ID correcto al elegir una.
- [ ] **CA-009**: No queda ninguna referencia a `automation_base_url`, `automation_api_key` ni `SettingsAutomationPage` en la SPA.
- [ ] **CA-010**: `BACK_ROUTES` funciona en móvil para detalle, alta, edición y plugins.
- [ ] **CA-011**: Todos los inputs usan tokens del tema y se verificaron en darkmode (texto y placeholders legibles).
- [ ] **CA-012**: `npm run build` pasa sin errores de TypeScript y las claves i18n se sirven desde `public/i18n` tras el build.
- [ ] **CA-013**: La app corre en local sobre una copia de producción y el flujo completo (listar, crear, editar, vincular servicio, consultar facturas) funciona con datos reales.
- [ ] **CA-014**: El formulario y el detalle de cuenta no muestran `webhook_url`; al editar una cuenta existente el valor se conserva en la DB (fallback) y no se borra.
- [ ] **CA-015**: El detalle de cuenta usa un `CardMenu` de 3 puntos con Editar, Revelar credenciales, Consultar ahora y Eliminar; no hay botones sueltos para esas acciones.
- [ ] **CA-016**: El formulario de cuenta muestra 3 tabs (Información, Plugin, Frecuencia); cada tab contiene solo sus campos, la validación salta a la tab del error y Guardar/Cancelar siguen accesibles.
- [ ] **CA-017**: Con campos requeridos vacíos o una programación inválida, el botón Guardar está deshabilitado; las tabs afectadas muestran un punto rojo y el texto de ayuda.

---

## 6. Plan de Implementación

| Fase | Descripción | Estimación |
|---|---|---|
| 1 | Tipos y cliente API (`types`, `api`) | 0.5 h |
| 2 | Listado `/automation` + sidebar + rutas + `BACK_ROUTES` | 1 h |
| 3 | Formulario de cuenta con credenciales dinámicas y programación | 2 h |
| 4 | Detalle: resumen, facturas, logs, reveal, sync, delete | 2 h |
| 5 | Catálogo de plugins | 0.5 h |
| 6 | Selector en `ServiceFormPage` y retiro de settings legacy | 1 h |
| 7 | i18n, darkmode, build, E2E local sobre copia de producción | 1.5 h |

Dependencias: SPEC-110 (backend) debe estar mergeada en esta rama. No hay dependencias externas nuevas.

---

## 7. Riesgos y Mitigaciones

| Riesgo | Mitigación |
|---|---|
| El schema de credenciales cambia entre plugins | La UI se guía por `CredentialField`; no hardcodea campos. |
| El usuario deja credenciales secretas vacías y cree que se borraron | Texto explícito "dejar vacío para conservar" y no incluir claves vacías en el payload. |
| Cuentas con más de un servicio vinculado | El detalle lista todos y el backend rechaza el sync ambiguo con mensaje claro. |
| Pérdida de la configuración legacy antes de SPEC-112 | No se borran claves del backend; solo se retira la UI. |
| i18n editado en `public/i18n` por error | Editar siempre `frontend/public/i18n` y verificar tras `npm run build`. |
| Regresión en `ServiceFormPage` | Mantener `automation_account_id` opcional y probar alta/edición con y sin cuenta. |

---

## 8. Historial de Cambios

| Fecha | Autor | Cambio |
|---|---|---|
| 2026-10-21 | opencode | Creación de la spec e inicio de implementación (issue #114). |
| 2026-10-21 | opencode | Implementación completa: tipos y cliente API, listado, formulario con credenciales dinámicas y programación, detalle con facturas/logs/reveal/sync/delete, catálogo de plugins, selector en `ServiceFormPage` y retiro de la configuración legacy. `npm run build` pasa. E2E sobre copia de producción con el export real de automation: 6 cuentas, 80 facturas, 838 logs, 5 plugins; el selector devuelve 6 opciones. Pendiente revisión visual/darkmode del usuario. |
| 2026-10-21 | opencode | Cambio iterativo solicitado por el usuario: se retira `webhook_url` de la UI de automation (formulario y detalle). Se conserva el valor en la DB al editar como fallback de transición; el endpoint externo del servicio no se toca. |
| 2026-10-21 | opencode | Cambio iterativo: las acciones del detalle (editar, revelar credenciales, consultar ahora, eliminar) pasan al `CardMenu` de 3 puntos, siguiendo el patrón de UI del proyecto. |
| 2026-10-21 | opencode | Cambio iterativo: el formulario de cuenta se segmenta en tabs Información / Plugin / Frecuencia, mostrando solo la sección activa. |
| 2026-10-21 | opencode | Cambio iterativo: Guardar se deshabilita mientras falten campos requeridos; tabs con error marcadas y texto de ayuda. |
