---
title: "Bancos: detalle de cuenta informativo, página de transferencias con tabs y toggles iOS obligatorios"
id: "SPEC-135"
status: "released"
author: "opencode"
created: "2026-11-17"
updated: "2026-11-17"
github_issue: 138
---

# Bancos: detalle de cuenta informativo, página de transferencias con tabs y toggles iOS obligatorios

**ID**: SPEC-135  
**Estado**: released  
**Autor**: opencode  
**Creado**: 2026-11-17  
**Actualizado**: 2026-11-17

---

## 1. Resumen Ejecutivo

El módulo de bancos (SPEC-133) resolvió el CRUD de bancos, cuentas y formatos de transferencia, pero la página de detalle de una cuenta (`/bancos/:id/cuentas/:accountId`) es en realidad el **formulario de edición**: inputs, selector de moneda, checkbox de cuenta activa, botones Guardar/Cancelar y, debajo, el editor de formatos con sus tabs. Eso rompe el patrón de la app: una página de detalle debe **mostrar información en modo lectura**, y editar debe ser una acción explícita que lleve a un formulario separado (como ya ocurre en autos, instituciones, deudas y servicios). Además, el editor de formatos mezclado con el formulario de la cuenta hace que una tarea de configuración bancaria (Transferencia / Transferencia ACH) quede escondida al final de un formulario de datos personales.

Esta spec separa las responsabilidades en tres pantallas: (1) **detalle de cuenta** de solo lectura, (2) **formulario de cuenta** para crear/editar, y (3) **página de transferencias** con dos tabs —`Transferencia` y `Transferencia ACH`— que conserva toda la funcionalidad actual del editor de formatos. La página de transferencias se abre **solo desde el menú de 3 puntos** de la cuenta (tanto en la card del banco como en el header del detalle), junto a las acciones **Editar** y **Eliminar**. De este modo la navegación sigue el patrón de acciones del proyecto y no aparecen links o botones sueltos dentro del contenido.

La spec también corrige el control de **cuenta activa**: hoy usa un `<input type="checkbox">` nativo; debe usar el `Toggle` estilo iOS del repositorio (`frontend/src/components/Toggle.tsx`), igual que Configuraciones y otras entidades. La regla se documenta de forma permanente en `docs/project-rules.md` y `AGENTS.md` para que no vuelva a ocurrir: **las páginas show nunca son formularios** y **todo booleano de entidad se muestra con Toggle iOS, nunca con checkbox nativo**. El cambio es exclusivamente de frontend (rutas, componentes, i18n y documentación): no hay migraciones, no se toca SQLite, no se agregan dependencias y el impacto en el iHost es nulo.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: La ruta `/bancos/:id/cuentas/:accountId` DEBE ser una página de **detalle de cuenta de solo lectura** (`BankAccountShowPage`). No puede contener `<form>`, `<input>`, `<textarea>`, `Select`, botones Guardar/Cancelar ni ningún control editable. Muestra: alias, banco, estado (Activa/Inactiva), número de cuenta, tipo de cuenta, moneda (código y nombre), beneficiario, notas, fecha de creación y última actualización.
2. **REQ-002**: La edición de una cuenta DEBE vivir en una ruta propia `/bancos/:id/cuentas/:accountId/edit` (`BankAccountFormPage`), accesible desde la acción **Editar** del menú de 3 puntos. Al guardar con éxito, navega al detalle de la cuenta y muestra el toast existente.
3. **REQ-003**: La creación de cuenta se mantiene en `/bancos/:id/cuentas/new` (`BankAccountFormPage`). Al crear, navega al detalle de la cuenta recién creada.
4. **REQ-004**: El formulario de cuenta DEBE dejar de contener el editor de formatos de transferencia, las tabs, la carga de variables y la carga de formatos. Solo gestiona los campos de la cuenta y el estado activo.
5. **REQ-005**: Nueva página de transferencias en `/bancos/:id/cuentas/:accountId/transferencias` (`BankAccountTransfersPage`) con dos tabs visibles: **Transferencia** (`transfer`) y **Transferencia ACH** (`ach`). El tab activo se refleja en el querystring (`?tab=transfer` por defecto, `?tab=ach`) para permitir deep links y conservar el estado al recargar.
6. **REQ-006**: La página de transferencias DEBE reutilizar `TransferFormatEditor` con toda su funcionalidad vigente: llaves/valores, llaves sugeridas por tipo, variables disponibles, vista previa del JSON, copiar JSON, guardar formato y eliminar formato con confirmación por palabra.
7. **REQ-007**: El acceso a la página de transferencias DEBE estar **solo** en el menú de 3 puntos (`CardMenu`): en la card de cada cuenta del detalle del banco y en el header del detalle de la cuenta. Opciones del menú de la cuenta: **Editar**, **Transferencias**, **Eliminar** (en ese orden, Editar con `app.edit`, Eliminar con `app.delete`, Transferencias con clave i18n propia). No se permiten links "Transferencias" ni botones directos dentro del contenido de la página.
8. **REQ-008**: El menú de la card de la cuenta DEBE detener la propagación del click para no disparar la navegación al detalle (`CardMenu` ya lo hace; se debe verificar que la nueva opción lo respete).
9. **REQ-009**: El campo **Cuenta activa** del formulario de cuenta DEBE usar el componente `Toggle` estilo iOS. Se elimina el `<input type="checkbox">` de `BankAccountFormPage`.
10. **REQ-010**: El campo **Activo** del formulario de banco (`BankFormPage`) también DEBE usar `Toggle` estilo iOS, por consistencia con la regla nueva.
11. **REQ-011**: Registrar en `BACK_ROUTES` (`frontend/src/components/DashboardLayout.tsx`) las rutas nuevas, respetando el orden de especificidad:
    - `/bancos/:id/cuentas/:accountId/transferencias` → detalle de la cuenta.
    - `/bancos/:id/cuentas/:accountId/edit` → detalle de la cuenta.
    - `/bancos/:id/cuentas/:accountId` → detalle del banco (`/bancos/:id`).
    - `/bancos/:id/cuentas/new` → detalle del banco (comportamiento actual).
12. **REQ-012**: Documentar en `docs/project-rules.md` y `AGENTS.md` dos reglas permanentes:
    - **Regla de páginas Show**: una página de detalle muestra información en modo lectura; jamás es un formulario. Editar es una acción separada con ruta `/edit/:id` y etiqueta **Editar**.
    - **Regla de controles booleanos**: todo booleano de entidad o configuración (activo/inactivo, habilitado, generar automáticamente, etc.) usa `Toggle` estilo iOS; prohibido el checkbox nativo.
13. **REQ-013**: Agregar las claves i18n necesarias en `frontend/public/i18n/es.json` y `frontend/public/i18n/en.json` (título del detalle, secciones, estado activo/inactivo, acción Transferencias, tabs, textos de retorno, etc.). Sin textos hardcodeados en la UI.
14. **REQ-014**: El detalle de la cuenta DEBE mostrar un indicador de estado no editable (badge **Activa** / **Inactiva**). Si en el futuro se quiere cambiar el estado, se hace desde Editar; la página show no modifica datos.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-015**: Mostrar en el detalle de la cuenta un resumen **informativo** de los formatos configurados (por ejemplo, "Transferencia: configurado", "Transferencia ACH: sin configurar"), sin links ni controles. Sirve de orientación; la edición sigue estando solo en la página de transferencias.
2. **REQ-016**: Reemplazar las implementaciones inline de toggle (markup duplicado con `peer`/`sr-only`) por el componente `Toggle` en `SalaryFormPage`, `NotificationFormPage` y `ServiceFormPage`, manteniendo el mismo comportamiento.
3. **REQ-017**: Extender `frontend/scripts/check-ui-rules.cjs` para detectar `<input type="checkbox">` fuera de una allowlist justificada. La allowlist mínima: `frontend/src/components/Toggle.tsx` (implementación del toggle) y checkboxes de **selección múltiple** de filas/items (por ejemplo `BudgetTransactionsPage`, tablas de selección). El build debe fallar con un mensaje que apunte a `docs/project-rules.md` §4.10.
4. **REQ-018**: Mejorar la accesibilidad de `Toggle`: foco visible (`focus-visible` ring), `aria-label`/etiqueta asociada cuando el texto está fuera del componente, y estado `disabled` legible en darkmode. Sin cambiar su API actual de forma incompatible.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-019**: Evaluar el checkbox **Recordarme** de `LoginPage`: si se migra a `Toggle`, debe conservar el texto y el comportamiento de sesión; no es bloqueante para esta spec.
2. **REQ-020**: Enlaces profundos desde una cuota/deuda al formato de transferencia asociado (depende de SPEC-133 P1) podrían abrir la página de transferencias con `?tab=ach` o `?tab=transfer`; se documenta la ruta para que futuras specs la reutilicen.
3. **REQ-021**: Agregar tests de componentes para el detalle (no renderiza inputs) y para el mapeo de tabs (`?tab` inválido cae a `transfer`). Requiere incorporar infraestructura de tests al frontend si no existe; se deja como deuda documentada.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: una request adicional como máximo al abrir la página de transferencias (`GET /api/bank-accounts/{id}` ya devuelve la cuenta con sus formatos). Sin polling ni llamadas externas. El impacto en el iHost es nulo: solo cambia el bundle del frontend.
- **Almacenamiento**: sin migraciones ni cambios de esquema. Se reutilizan las tablas `banks`, `bank_accounts` y `bank_transfer_formats` de SPEC-133.
- **Dependencias**: ninguna nueva. Se reutilizan `react-router-dom`, `CardMenu`, `TransferFormatEditor`, `Toggle`, `Select` y `DeleteModal` existentes.
- **Seguridad**: todas las rutas siguen bajo `AuthGuard`. Números de cuenta y beneficiarios son datos sensibles: no se loguean ni se incluyen en mensajes de error. Los ejemplos de esta spec son sintéticos.
- **Accesibilidad**: tabs con `role="tablist"`/`role="tab"`/`aria-selected`, navegación por teclado y foco visible. El detalle usa jerarquía de encabezados coherente.
- **UX móvil**: acciones de 44px mínimo, tabs desplazables si no entran, menú de 3 puntos con portal y `z-50`, back arrow en el header (no links de retorno en el contenido).
- **UI**: tokens del tema (`bg-card`, `text-text`, `text-text-secondary`), darkmode verificado, `Select` del repo (nunca `<select>` nativo), modales conforme a SPEC-126.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Estado actual del código en `main` (commit `2e5b832`):

| Elemento | Ubicación | Hallazgo |
|---|---|---|
| Rutas de bancos | `frontend/src/App.tsx` L165-L170 | `/bancos/:id/cuentas/:accountId` renderiza `BankAccountFormPage` (formulario de edición). |
| Formulario de cuenta | `frontend/src/pages/BankAccountFormPage.tsx` | Contiene el form completo, checkbox de activo (L245-L253) y el editor de formatos con tabs (L281-L313). |
| Detalle del banco | `frontend/src/pages/BankDetailPage.tsx` | Cards de cuentas con `CardMenu` (Editar/Delete); el click en la card navega a la ruta del formulario. |
| Editor de formatos | `frontend/src/components/TransferFormatEditor.tsx` | Componente autónomo con llaves/valores, variables, preview, copia y guardado; puede moverse a otra página sin reescribirse. |
| Toggle iOS | `frontend/src/components/Toggle.tsx` | Componente compartido (11×24 px, `sr-only` + `peer`), ya usado en Configuraciones y `CategoryFormPage`. Su API es `checked`, `onChange`, `disabled`. |
| Checkbox de banco | `frontend/src/pages/BankFormPage.tsx` L143-L150 | Checkbox nativo para `active`. |
| Rutas de retorno | `frontend/src/components/DashboardLayout.tsx` L16-L21 | `BACK_ROUTES` ya contempla `/bancos/:id/cuentas/(new|\d+)`; hay que ampliar con `edit` y `transferencias` antes de la regla genérica. |
| i18n | `frontend/public/i18n/{es,en}.json` | Existen `banks.*` para cuentas y formatos; faltan claves de detalle, estado y acción Transferencias. |
| Guardrail UI | `frontend/scripts/check-ui-rules.cjs` | Hoy solo prohíbe `<select>` nativo; no detecta checkboxes. |
| API | `frontend/src/api/index.ts` | `bankAccounts.get(id)` devuelve la cuenta con `formats`; `bankTransferVariables.list()` alimenta el editor. No requiere endpoints nuevos. |

Patrones de referencia en el proyecto:

- `AutoShowPage` / `InstitutionShowPage`: páginas de detalle de solo lectura con menú de 3 puntos y acciones (Editar/Eliminar) fuera del contenido.
- `DebtBillsPage` / `ServicesPage`: tabs con estado en querystring.
- `CategoryFormPage`, `SettingsPage`: uso del `Toggle` iOS para booleanos.
- `TransferFormatEditor`: ya resuelve variables, validación y confirmación por palabra; no debe duplicarse.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Mantener el formulario como "show" y ocultar inputs con CSS | Cero rutas nuevas | Sigue siendo un formulario disfrazado; rompe accesibilidad y el patrón; el usuario lo rechazó explícitamente | ❌ Rechazada |
| Un solo componente con modo `view`/`edit` según la ruta | Menos archivos | Mezcla lectura y edición, estados y validaciones; difícil de testear y de leer | ❌ Rechazada |
| Página show de solo lectura + formulario en `/edit/:id` | Patrón claro, accesible, consistente con autos/instituciones; reutiliza el form actual con mínimos cambios | Una ruta y un componente nuevo | ✅ Seleccionada |
| Tabs de transferencias dentro del show | Sin ruta nueva | Vuelve a mezclar información con edición; el usuario pidió una página nueva | ❌ Rechazada |
| Página de transferencias con tabs y deep link `?tab=` | Ruta dedicada, enlazable, mantiene el editor intacto | Una página y una ruta nuevas | ✅ Seleccionada |
| Página de transferencias accesible con botón dentro del show | Visible y directo | Viola el pedido del usuario y el patrón de acciones | ❌ Rechazada |
| Entrada solo desde el menú de 3 puntos | Consistente con el patrón de acciones | Menos descubrible; se mitiga con la opción en card y show | ✅ Seleccionada |
| Checkbox nativo para "activo" | Simple, nativo | Rompe el estilo iOS; el usuario lo rechazó; distinto a Configuraciones | ❌ Rechazada |
| `Toggle` iOS compartido | Consistente, accesible, ya existe | Requiere migrar dos formularios | ✅ Seleccionada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Separación estricta show/edición.

- **Contexto**: hoy el show es el formulario de edición y el usuario lo considera inaceptable.
- **Decisión**: `/bancos/:id/cuentas/:accountId` es solo lectura; la edición vive en `/edit`. El show no renderiza controles editables ni botones de guardado.
- **Consecuencias**: los enlaces existentes al detalle siguen funcionando y ahora muestran información; cualquier flujo que esperaba editar directo debe usar el menú **Editar**. Se gana claridad y se elimina el riesgo de cambios accidentales.

**ADR-002**: Transferencias como página propia con tabs.

- **Contexto**: el editor de formatos es una tarea de configuración distinta de los datos de la cuenta y hoy queda al final de un formulario largo.
- **Decisión**: nueva página `/transferencias` con tabs `transfer` y `ach`; el tab activo se guarda en `?tab=` para deep links y recarga.
- **Consecuencias**: menor carga cognitiva del formulario; el editor se reutiliza sin cambios funcionales. El estado de tabs no requiere store global.

**ADR-003**: Acceso exclusivo por menú de 3 puntos.

- **Contexto**: el usuario pidió que se llegue desde el dropdown de 3 puntos.
- **Decisión**: la opción **Transferencias** vive en el `CardMenu` de la card de cuenta y en el menú del header del show. No hay links ni botones en el contenido.
- **Consecuencias**: se respeta la regla de acciones del proyecto. La descubribilidad se cubre con dos puntos de entrada visibles (card y detalle).

**ADR-004**: Un único componente Toggle para booleanos.

- **Contexto**: hay toggles inline duplicados en varios formularios y checkboxes nativos en bancos.
- **Decisión**: `Toggle` es el único control válido para booleanos de entidad/configuración; se migran bancos en P0 y el resto en P1. Los checkboxes de selección múltiple quedan exceptuados.
- **Consecuencias**: consistencia visual y de comportamiento; el guardrail de build evita regresiones. La regla se documenta en `project-rules.md` §4.10 y `AGENTS.md`.

**ADR-005**: Sin cambios de backend.

- **Contexto**: los endpoints actuales ya devuelven todo lo necesario, incluidos los formatos.
- **Decisión**: no se agregan rutas API ni migraciones; sólo se agregan rutas de frontend y se reubican componentes.
- **Consecuencias**: cambio de bajo riesgo y despliegue simple; se mantiene el consumo del iHost.

## 4. Diseño Técnico

### 4.1 Rutas

```text
/bancos                                    BanksPage
/bancos/new                                BankFormPage (crear)
/bancos/edit/:id                           BankFormPage (editar)
/bancos/:id                                BankDetailPage
/bancos/:id/cuentas/new                    BankAccountFormPage (crear)
/bancos/:id/cuentas/:accountId             BankAccountShowPage (solo lectura)   ← cambia
/bancos/:id/cuentas/:accountId/edit        BankAccountFormPage (editar)         ← nueva
/bancos/:id/cuentas/:accountId/transferencias  BankAccountTransfersPage (?tab=) ← nueva
```

Orden recomendado en `App.tsx`: declarar `new`, `edit/:accountId` y `transferencias` antes de la ruta genérica `:accountId`. Aunque React Router v7 prioriza por especificidad, el orden explícito facilita la lectura.

### 4.2 `BACK_ROUTES`

Agregar antes de la regla genérica actual:

```ts
{
  pattern: /^\/bancos\/\d+\/cuentas\/\d+\/transferencias$/,
  to: (pathname) => `/bancos/${pathname.split('/')[2]}/cuentas/${pathname.split('/')[4]}`,
},
{
  pattern: /^\/bancos\/\d+\/cuentas\/\d+\/edit$/,
  to: (pathname) => `/bancos/${pathname.split('/')[2]}/cuentas/${pathname.split('/')[4]}`,
},
{
  pattern: /^\/bancos\/\d+\/cuentas\/(new|\d+)$/,
  to: (pathname) => `/bancos/${pathname.split('/')[2]}`,
},
```

Notas:

- El retorno de **transferencias** y **editar** es el detalle de la cuenta.
- El retorno del **show de cuenta** es el detalle del banco.
- No se agregan links "← Título" en el contenido (SPEC-063).

### 4.3 `BankAccountShowPage` (nuevo)

- Carga en paralelo `api.banks.get(bankId)` y `api.bankAccounts.get(accountId)`.
- Header:
  - Alias como título (`usePageTitle`).
  - Subtítulo: banco + tipo de cuenta.
  - Badge de estado: `Activa` (verde/success) / `Inactiva` (text-secondary).
  - `CardMenu` con opciones: **Editar** (`app.edit`), **Transferencias** (`banks.transfers`), **Eliminar** (`app.delete`, danger).
- Contenido en cards de solo lectura con pares etiqueta/valor:
  - **Cuenta**: número (mono), tipo, moneda (`código - nombre`), beneficiario.
  - **Estado**: badge + texto explicativo de que se cambia desde Editar.
  - **Notas**: texto o "Sin notas".
  - **Metadatos**: fecha de creación y última actualización (formato local).
  - **Formatos** (P1): estado por tipo, solo texto.
- No hay `<form>`, inputs, `Select`, botones de guardar ni links de retorno.
- Eliminación: reutiliza `DeleteModal` con confirmación por palabra; al confirmar, navega al banco.
- Estado de carga con `LoadingSpinner`; error con toast y contenido vacío.

### 4.4 `BankAccountFormPage` (modificado)

- Conserva los campos actuales: alias, número, tipo, moneda, beneficiario, notas y activo.
- **Se elimina**: carga de variables, tabs, `TransferFormatEditor`, `account.formats`, `reloadAccount` y todo lo relacionado con formatos.
- **Activo**: `<Toggle checked={active} onChange={setActive} />` con etiqueta `banks.active_account` al lado (no dentro de un `<label>` con input nativo).
- Título:
  - Crear: `banks.create_account`.
  - Editar: `banks.edit_account`.
- Al guardar:
  - Crear → navega a `/bancos/{bankId}/cuentas/{created.id}`.
  - Editar → navega a `/bancos/{bankId}/cuentas/{accountId}`.
- El formulario de edición no requiere cargar formatos; solo `api.bankAccounts.get` para hidratar los campos.
- Mantiene cancelar → detalle de la cuenta en edición; en creación → detalle del banco.

### 4.5 `BankAccountTransfersPage` (nuevo)

- Carga en paralelo: `api.banks.get(bankId)`, `api.bankAccounts.get(accountId)` y `api.bankTransferVariables.list()` (con fallback a las variables locales, igual que hoy).
- Header:
  - Título: `banks.transfers`.
  - Subtítulo: alias de la cuenta + banco.
  - Sin acciones de creación en el header.
- Tabs accesibles:
  ```tsx
  <div role="tablist" aria-label={t('banks.transfers')}>
    <button role="tab" aria-selected={tab === 'transfer'} ...>{t('banks.transfer')}</button>
    <button role="tab" aria-selected={tab === 'ach'} ...>{t('banks.ach')}</button>
  </div>
  ```
- `tab` se inicializa desde `searchParams.get('tab')`; valores inválidos caen a `'transfer'`. Al cambiar de tab se actualiza el querystring con `replace: true` para no llenar el historial.
- Renderiza `TransferFormatEditor` con `key={tab}`, `accountId`, `account`, `bank`, `kind`, `initialFormat` y `variables`.
- Al guardar/eliminar un formato, recarga la cuenta (`api.bankAccounts.get`) para reflejar `formats`.
- Si el formato no existe, el editor arranca vacío con llaves sugeridas (comportamiento actual).
- La página no duplica la lógica del editor ni agrega botones de cuenta.

### 4.6 Menús de 3 puntos

**Card de cuenta en `BankDetailPage`:**

```ts
options = [
  { label: t('app.edit'), icon: 'edit', onClick: () => navigate(`/bancos/${bank.id}/cuentas/${account.id}/edit`) },
  { label: t('banks.transfers'), icon: 'transfer' | 'wallet', onClick: () => navigate(`/bancos/${bank.id}/cuentas/${account.id}/transferencias`) },
  { label: t('app.delete'), icon: 'delete', danger: true, onClick: () => setDeleteTarget(account) },
]
```

El click en la card navega al detalle (`/bancos/{bankId}/cuentas/{accountId}`), que ahora es informativo.

**Header del `BankAccountShowPage`:** mismo `CardMenu` con Editar, Transferencias y Eliminar.

> Nota de consistencia: SPEC-134 (en otra ventana) también elimina el ícono de editar suelto del detalle del banco. Esta spec aplica el mismo criterio en el detalle de la cuenta: un único menú de 3 puntos, sin íconos de acción sueltos al lado.

### 4.7 Toggle iOS

- Usar `frontend/src/components/Toggle.tsx` en `BankAccountFormPage` (activo) y `BankFormPage` (activo).
- Estructura recomendada:
  ```tsx
  <div className="flex items-center justify-between gap-3 min-h-[44px]">
    <span className="text-sm font-medium">{t('banks.active_account')}</span>
    <Toggle checked={active} onChange={setActive} ariaLabel={t('banks.active_account')} />
  </div>
  ```
- P1: el componente acepta `ariaLabel` opcional y agrega `focus-visible` ring. El texto visible queda fuera del `label` del componente, por lo que el `aria-label` es obligatorio para lectores de pantalla.
- Prohibido `<input type="checkbox">` para este caso y para cualquier booleano de entidad/configuración.

### 4.8 Reglas de UI a documentar

**`docs/project-rules.md`** — agregar:

- **§4.9 Páginas de detalle (show): información, nunca formulario.**
  - Una página de detalle muestra datos en modo lectura; no incluye `<form>`, inputs, selects, textareas ni botones Guardar/Cancelar.
  - Editar es una acción explícita (menú de 3 puntos o header) que navega a una ruta `/edit/:id` separada.
  - La acción se rotula **Editar** (`app.edit`).
  - Los cambios de estado se hacen desde Editar; el show solo muestra el estado con un badge.
  - Precedente: SPEC-135 (cuenta bancaria).

- **§4.10 Booleanos: Toggle iOS obligatorio, checkbox nativo prohibido.**
  - Todo booleano de entidad o configuración (`active`, `enabled`, `auto_generate`, `is_recurring`, etc.) usa `frontend/src/components/Toggle.tsx`.
  - Prohibido `<input type="checkbox">` para estos casos.
  - Excepción: checkboxes de **selección múltiple** de filas/items en tablas o listas, que no representan un estado persistido.
  - El toggle debe tener etiqueta visible o `aria-label`, foco visible y verificarse en darkmode.
  - El guardrail `frontend/scripts/check-ui-rules.cjs` (P1) falla ante checkbox nativo fuera de la allowlist.

**`AGENTS.md`** — agregar dos bullets en "Reglas críticas":

- **Páginas show jamás son formularios.** El detalle muestra información; editar vive en `/edit/:id` y se accede por la acción **Editar**.
- **Booleanos siempre con `Toggle` iOS.** Nunca checkbox nativo para activo/inactivo ni flags de entidad/configuración.

**`.opencode/skills/spec-manager/SKILL.md`** (P1): sumar ambas reglas a la lista de consideraciones obligatorias para nuevas specs.

### 4.9 i18n

Claves nuevas propuestas (nombres definitivos a ajustar en implementación):

| Clave | ES | EN |
|---|---|---|
| `banks.account_detail` | `Detalle de la cuenta` | `Account details` |
| `banks.transfers` | `Transferencias` | `Transfers` |
| `banks.account_status` | `Estado` | `Status` |
| `banks.active_status` | `Activa` | `Active` |
| `banks.inactive_status` | `Inactiva` | `Inactive` |
| `banks.account_info` | `Información de la cuenta` | `Account information` |
| `banks.account_number_label` | `Número de cuenta` | `Account number` |
| `banks.notes_section` | `Notas` | `Notes` |
| `banks.no_notes` | `Sin notas` | `No notes` |
| `banks.created_at` | `Creada` | `Created` |
| `banks.updated_at` | `Última actualización` | `Last updated` |
| `banks.formats_summary` | `Formatos` | `Formats` |
| `banks.format_configured` | `Configurado` | `Configured` |
| `banks.format_not_configured` | `Sin configurar` | `Not configured` |
| `banks.status_edit_hint` | `Para cambiar el estado, usá Editar.` | `To change the status, use Edit.` |

Reglas: editar siempre en `frontend/public/i18n/{es,en}.json`; correr `npm run build` en `frontend/`; verificar que las claves se sirvan desde `/i18n/es.json`. Sin textos hardcodeados.

### 4.10 Backend / Base de datos

- Sin cambios en `internal/api/bank_handlers.go`, `internal/services`, `internal/storage` ni `migrations/`.
- Se reutilizan:
  - `GET /api/banks/{id}`
  - `GET /api/banks/{id}/accounts`
  - `GET /api/bank-accounts/{id}` (incluye `formats`)
  - `PUT /api/bank-accounts/{id}`
  - `PUT /api/bank-accounts/{id}/transfer-formats/{kind}`
  - `DELETE /api/bank-accounts/{id}/transfer-formats/{kind}`
  - `GET /api/bank-transfer-variables`

### 4.11 Flujo de usuario

```text
Bancos → Banco → card de cuenta
  ├─ click en card ───────────────→ Detalle de cuenta (solo lectura)
  │                                   ├─ menú ⋯ → Editar ─────────→ Formulario de cuenta
  │                                   ├─ menú ⋯ → Transferencias ──→ Página de transferencias
  │                                   │                              ├─ tab Transferencia
  │                                   │                              └─ tab Transferencia ACH
  │                                   └─ menú ⋯ → Eliminar ────────→ DeleteModal (confirmo)
  └─ menú ⋯ → Editar / Transferencias / Eliminar (acciones directas)
```

## 5. Criterios de Aceptación

- [x] **CA-001**: Al entrar a `/bancos/:id/cuentas/:accountId` se ve una página de detalle de solo lectura; el DOM no contiene `<form>`, `input`, `textarea`, `Select` ni botones Guardar/Cancelar. Verificable con inspección y búsqueda en el componente.
- [x] **CA-002**: El detalle muestra alias, banco, estado (Activa/Inactiva), número de cuenta, tipo, moneda, beneficiario, notas, fecha de creación y última actualización.
- [x] **CA-003**: El menú de 3 puntos del detalle contiene **Editar**, **Transferencias** y **Eliminar**; Editar navega a `/edit` y Transferencias a `/transferencias`.
- [x] **CA-004**: El formulario de cuenta en `/new` y `/edit/:accountId` ya no muestra tabs ni editor de formatos; solo los campos de cuenta y el Toggle de activo.
- [x] **CA-005**: Crear una cuenta navega a su detalle; editar y guardar navega al detalle con el toast de éxito. Cancelar en edición vuelve al detalle; cancelar en creación vuelve al banco.
- [x] **CA-006**: `/bancos/:id/cuentas/:accountId/transferencias` muestra dos tabs: **Transferencia** y **Transferencia ACH**.
- [x] **CA-007**: El tab activo se refleja en `?tab=transfer|ach`, sobrevive a la recarga y un valor inválido cae a `transfer`.
- [x] **CA-008**: El editor de formatos conserva: llaves sugeridas, agregar/quitar/renombrar llaves, variables, vista previa, copiar JSON, guardar y eliminar con confirmación por palabra.
- [x] **CA-009**: La página de transferencias solo es accesible desde el menú de 3 puntos; no hay links ni botones sueltos hacia ella en el contenido.
- [x] **CA-010**: El menú de la card de cuenta incluye Transferencias y el click en la opción no dispara la navegación al detalle.
- [x] **CA-011**: El campo **Cuenta activa** usa `Toggle` iOS; no existe `type="checkbox"` en `BankAccountFormPage` ni en `BankFormPage`.
- [x] **CA-012**: El Toggle funciona con teclado (Tab + Espacio), muestra foco visible y tiene etiqueta accesible.
- [x] **CA-013**: `BACK_ROUTES` devuelve: transferencias → detalle de cuenta; editar cuenta → detalle de cuenta; detalle de cuenta → banco. En móvil se ve la flecha atrás; no hay links "← Título".
- [x] **CA-014**: No hay íconos de acción sueltos en el header del detalle de la cuenta; las acciones están en el menú de 3 puntos.
- [x] **CA-015**: `docs/project-rules.md` incluye §4.9 (show informativo) y §4.10 (toggles iOS) y `AGENTS.md` incluye ambos bullets.
- [x] **CA-016**: `frontend/public/i18n/es.json` y `en.json` tienen las claves nuevas; no hay textos hardcodeados en las páginas nuevas.
- [x] **CA-017**: `npm run check:ui` y `npm run build` pasan en `frontend/`; el `tsc -b` no reporta errores.
- [x] **CA-018**: El detalle y el formulario se verifican en lightmode y darkmode (texto, placeholders y toggles legibles).
- [x] **CA-019**: Sin migraciones ni cambios de backend; `git diff` no toca `internal/` ni `migrations/`.
- [x] **CA-020**: Prueba manual con datos sintéticos (`Banco Demo`, `demo-account-001`, `Beneficiario Demo`) sobre una copia de la DB de producción si el flujo depende de datos existentes (no aplica: el flujo de UI se validó con datos sintéticos).

## 6. Plan de Implementación

| Fase | Descripción | Estimación |
|------|-------------|------------|
| 1 | Documentar reglas en `project-rules.md` y `AGENTS.md` (§4.9 y §4.10) | 30 min |
| 2 | Crear `BankAccountShowPage` de solo lectura y sus estilos | 2 h |
| 3 | Crear `BankAccountTransfersPage` con tabs, querystring y `TransferFormatEditor` | 2 h |
| 4 | Simplificar `BankAccountFormPage` (quitar formatos, rutas `/new` y `/edit`, navegación post-guardado) | 1.5 h |
| 5 | Rutas en `App.tsx`, `BACK_ROUTES`, menús de 3 puntos en card y show | 1 h |
| 6 | Migrar activo de cuenta y banco a `Toggle`; P1: toggles inline restantes y accesibilidad | 1 h |
| 7 | i18n ES/EN, `npm run build`, verificación en darkmode y prueba manual | 1.5 h |
| 8 | P1: guardrail `check-ui-rules.cjs` para checkboxes nativos + allowlist | 1 h |
| **Total P0** | | **~8 h** |
| **Total con P1** | | **~10 h** |

Dependencias: ninguna externa. SPEC-133 debe estar liberada (lo está). SPEC-134 toca el detalle del banco en otra ventana; no hay solapamiento de archivos salvo `BankDetailPage` (menú de cuenta) y `docs/project-rules.md`/`AGENTS.md`, que deben mergearse con cuidado.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Usuarios con enlaces guardados al detalle esperan editar | Media | Bajo | El detalle ahora es informativo; **Editar** está a un click en el menú. Se documenta en el resumen del release. |
| Conflictos de merge con SPEC-134 en `BankDetailPage` y reglas | Media | Medio | Mergear primero la spec que llegue a `main`; resolver a mano conservando ambas reglas (país + detalle/toggles). |
| Duplicación de lógica al mover formatos | Baja | Medio | Reutilizar `TransferFormatEditor` sin copiarlo; la página solo lo monta y recarga datos. |
| `BACK_ROUTES` con regex genérica captura `/edit` o `/transferencias` | Media | Bajo | Declarar las rutas específicas antes de la genérica y probar los tres retornos en móvil. |
| Pérdida de estado al cambiar de tab | Baja | Bajo | `key={tab}` remonta el editor; el tab vive en el querystring y los datos se recargan desde la API. |
| Toggle sin foco visible o sin etiqueta accesible | Media | Bajo | P1 agrega `ariaLabel` y `focus-visible`; P0 usa texto visible y verifica teclado. |
| Guardrail de checkbox con falsos positivos | Media | Bajo | Allowlist explícita para `Toggle.tsx` y checkboxes de selección múltiple; mensaje con la regla. |
| Datos sensibles en capturas o pruebas | Baja | Alto | Usar solo datos sintéticos; no commitear DBs ni capturas con datos reales (project-rules §8). |

## 8. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-17 | opencode | Creación de la spec en estado `draft`: detalle de cuenta de solo lectura, página de transferencias con tabs accesibles por menú de 3 puntos, Toggle iOS obligatorio y reglas de UI permanentes. |
| 2026-11-17 | opencode | Implementación P0 + P1 en `feature/SPEC-135`: `BankAccountShowPage`, `BankAccountTransfersPage` con `?tab=`, formulario de cuenta simplificado, rutas y `BACK_ROUTES`, menús de 3 puntos (card y detalle), Toggle en bancos/cuenta y migración de los checkboxes restantes a `Toggle` (salarios, notificaciones, servicios, presupuesto, plantillas, login), reglas §4.9/§4.10 en `project-rules.md`, `AGENTS.md` y skill, guardrail de checkbox en `check-ui-rules.cjs`, i18n ES/EN. Validado con `npm run build`, `go test ./...` y navegación headless en darkmode/lightmode. Estado a `in_progress` a la espera de confirmación manual del usuario. |
| 2026-11-17 | opencode | Release: implementación commiteada en `4ae981c` y merge `07372e1` a `main`; spec a `released`, issue #138 cerrado. Sin cambios de backend ni despliegue Docker en este ciclo. |
