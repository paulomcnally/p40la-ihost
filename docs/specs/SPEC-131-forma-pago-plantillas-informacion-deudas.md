---
title: "Forma de pago por deuda y plantillas dinámicas de información"
id: "SPEC-131"
status: "pending_release"
author: "opencode"
created: "2026-11-08"
updated: "2026-11-08"
github_issue: 134
---

# Forma de pago por deuda y plantillas dinámicas de información

**ID**: SPEC-131
**Estado**: pending_release
**Autor**: opencode
**Creado**: 2026-11-08
**Actualizado**: 2026-11-08

---

## 1. Resumen Ejecutivo

El detalle de una deuda (`/deudas/:id`) permite pagar cada cuota con fecha, referencia y comprobantes PDF, pero **no tiene un lugar donde guardar y mostrar la forma de pago**. Para una transferencia bancaria el usuario necesita ver, en el momento de pagar, el monto de la cuota, una descripción que incluya el número de cuota, el banco, el número de cuenta y el nombre del titular. Hoy esos datos viven fuera de la aplicación y se copian manualmente cada mes.

Esta spec incorpora dos piezas que trabajan juntas:

1. **Un sistema de plantillas de información** construido por el usuario: plantillas reutilizables compuestas por campos (texto, texto largo, número, fecha, selector, casilla y campo calculado). Cada plantilla funciona como un formulario configurable, de forma análoga a como el formulario de automatizaciones cambia según el esquema de credenciales del plugin/connector.
2. **La asignación de una plantilla a cada deuda**, con sus valores guardados como JSON. Esa asignación representa la **forma de pago** de la deuda (por ejemplo, "Transferencia bancaria").

La administración de plantillas se accede desde la sección **Deudas**: menú de 3 puntos → **Plantillas** → hub de plantillas → **Plantillas de información** → listado y constructor. El hub separa los tipos de plantilla y evita llenar el menú de Deudas a medida que se agreguen otros tipos en el futuro.

Al abrir el modal de pago de una cuota se agrega una segunda pestaña llamada **Información**, que renderiza la plantilla asignada con los valores de la deuda y resuelve los campos calculados con la cuota que se está pagando. Así, el monto y la descripción (`Deuda - Cuota N`) siempre corresponden a la cuota seleccionada, sin duplicar datos.

La implementación usa dos tablas nuevas (`debt_information_templates` y `debt_payment_information`) y **no modifica el esquema de `debts` ni de `debt_bills`**. No agrega dependencias de Go ni de frontend: el renderizado de tokens es una función pura del lado del cliente, el almacenamiento es SQLite y la UI reutiliza los componentes existentes (`Select`, `CardMenu`, `CreateMenu`, `DeleteModal`, tokens de tema). El impacto en el iHost es mínimo: unas pocas filas, JSON de tamaño acotado y cero llamadas externas.

> **Privacidad**: los ejemplos de esta spec son sintéticos. No se reproducen nombres, cuentas, montos, descripciones ni URLs reales del usuario. Ver `docs/project-rules.md` §8.

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001 — CRUD de plantillas de información**: crear, listar, ver, editar y eliminar plantillas reutilizables. Cada plantilla tiene:
   - `name`: nombre visible (ej. "Transferencia bancaria"), obligatorio, único sin distinguir mayúsculas, 1-100 caracteres.
   - `description`: texto opcional, 0-500 caracteres.
   - `fields`: lista ordenada de 1 a 30 campos (ver REQ-002).
   - `usage_count`: cantidad de deudas que la tienen asignada (calculado, no persistido).
   - `created_at` / `updated_at`.
   La lista se ordena por nombre. No se crean plantillas por seed: el usuario las construye.
2. **REQ-002 — Modelo de campo de plantilla**: cada campo define:
   - `key`: clave interna, generada automáticamente a partir del label (slug ASCII, minúsculas, sin acentos, separada por `_`), única dentro de la plantilla, 1-40 caracteres, regex `^[a-z][a-z0-9_]{0,39}$`. Es de solo lectura una vez agregada; para cambiarla se elimina el campo y se crea otro (con pérdida del valor guardado de ese campo).
   - `label`: etiqueta visible, 1-100 caracteres.
   - `type`: uno de `text`, `textarea`, `number`, `date`, `select`, `checkbox`, `computed`.
   - `required`: booleano. Solo aplica a tipos de entrada; `computed` nunca es requerido.
   - `placeholder`: opcional, 0-200 caracteres, solo tipos de entrada.
   - `default_value`: opcional, 0-500 caracteres, solo tipos de entrada.
   - `help_text`: opcional, 0-300 caracteres.
   - `options`: para `select`, entre 1 y 50 opciones de 1-100 caracteres cada una, sin duplicados.
   - `value_template`: para `computed`, 1-500 caracteres de texto literal con tokens permitidos (REQ-006).
3. **REQ-003 — Navegación y constructor de plantillas en la UI**: la administración de plantillas se organiza en dos niveles dentro de Deudas:
   - En la página Deudas (`/deudas`), el menú de 3 puntos del header incluye la opción **Plantillas** (además de "Nueva Deuda"). Navega a `/deudas/plantillas`.
   - `/deudas/plantillas` es el **hub de plantillas de deudas**. Muestra cards con los tipos de plantilla disponibles; en P0 hay una sola card: **Plantillas de información** (descripción + cantidad de plantillas) que navega a `/deudas/plantillas/informacion`. El hub deja preparado el lugar para futuros tipos (por ejemplo, plantillas de contratos o de recordatorios) sin cambiar el menú de Deudas.
   - `/deudas/plantillas/informacion` es el listado de plantillas de información: cards con menú de 3 puntos (`Editar` con `app.edit`, `Eliminar`), `EmptyCard` cuando no hay plantillas y `CreateMenu` con "Nueva plantilla".
   - `/deudas/plantillas/informacion/new` y `/deudas/plantillas/informacion/edit/:id` son el constructor, con:
     - Datos generales (nombre, descripción) y lista de campos como cards apiladas.
     - Agregar campo, quitarlo y reordenarlo con botones **subir/bajar** (sin librerías de drag & drop).
     - Inputs específicos por tipo. Los selectores usan el componente `Select` del repo; el de tipo de campo y el de plantilla en la asignación deben ser `searchable` si superan 8 opciones.
     - Validación inline y con toast; no se permite guardar una plantilla inválida.
     - Aviso visible cuando la plantilla está en uso: "Los cambios afectan a N deuda(s)".
     - Quitar un campo de una plantilla ya guardada pide confirmación; en una plantilla nueva se quita directamente.
   - La card de **Plantillas de información** del hub muestra la cantidad de plantillas y, si hay alguna, un resumen de la última actualización. No tiene acción de crear: la creación vive en el listado.
4. **REQ-004 — Asignar forma de pago a una deuda**: desde el detalle `/deudas/:id`, agregar la acción **Forma de pago** al menú de acciones del header (junto a Contrato, Reconocer documento y Editar). Abre un modal que:
   - Muestra un estado vacío con acceso a crear una plantilla si no existe ninguna.
   - Permite elegir una plantilla con `Select` (buscable) y completar sus campos.
   - Si la deuda ya tiene una asignación, precarga la plantilla y los valores guardados.
   - Si se cambia de plantilla, se conservan los valores cuyas claves coincidan y se descartan los demás, previa confirmación si había valores cargados.
   - Los campos `computed` se muestran como vista previa de solo lectura usando la **próxima cuota pendiente** de la deuda (o la primera pendiente); si no hay cuota pendiente, se muestra la expresión con una leyenda de que se resuelve al pagar.
   - Guarda con `PUT /api/debts/{id}/payment-information` y refresca el detalle.
   - Permite **Quitar forma de pago** con confirmación (`DELETE`).
5. **REQ-005 — Pestaña Información en el modal de pago**: `DebtPayModal` pasa a tener dos pestañas:
   - **Pagar** (activa por defecto): el formulario actual de fecha de pago, referencia, factura y comprobante.
   - **Información**: renderiza la plantilla asignada a la deuda con los valores guardados y los campos calculados resueltos para **la cuota seleccionada**. Muestra el nombre de la plantilla, los campos en orden, valores faltantes como `—`, textareas con saltos de línea y textos largos con `break-words`.
   - Si la deuda no tiene forma de pago configurada, la pestaña muestra un estado vacío y un botón **Configurar** que cierra el modal de pago y abre el modal de forma de pago.
   - El pie del modal cambia según la pestaña: en **Pagar** se mantienen Cancelar/Pagar; en **Información** solo hay cierre (y opcionalmente "Volver a pagar" para cambiar de pestaña). **Nunca se dispara el pago desde la pestaña Información.**
   - La pestaña Información es de solo lectura en P0; la edición de valores se hace desde el modal de forma de pago.
6. **REQ-006 — Tokens dinámicos y resolución**:
   - Catálogo mínimo de tokens:
     | Token | Valor |
     |---|---|
     | `{{deuda.descripcion}}` | `Debt.description` |
     | `{{deuda.institucion}}` | `Debt.institution_name` |
     | `{{deuda.identificador}}` | `Debt.identifier` |
     | `{{deuda.moneda}}` | `Debt.currency_code` |
     | `{{deuda.total}}` | `Debt.total` sin formato |
     | `{{deuda.total_formateado}}` | `Debt.total` con formato de moneda |
     | `{{cuota.numero}}` | `DebtBill.installment_number` |
     | `{{cuota.monto}}` | `DebtBill.amount` sin formato |
     | `{{cuota.monto_formateado}}` | `DebtBill.amount` con formato de la moneda de la deuda |
     | `{{cuota.vencimiento}}` | `DebtBill.due_date` (ISO `YYYY-MM-DD`) |
     | `{{cuota.vencimiento_formateado}}` | `DebtBill.due_date` localizado (es/en) |
     | `{{cuota.estado}}` | Estado de la cuota (`Pendiente` / `Pagada`) |
     | `{{pago.fecha}}` | Fecha de pago seleccionada en el formulario (ISO); por defecto hoy |
     | `{{pago.fecha_formateada}}` | Fecha de pago seleccionada, localizada |
     | `{{sistema.moneda}}` | Símbolo de la moneda (`$`, `C$`, etc.) |
   - La resolución es interpolación de texto plano, sin HTML, sin `eval` y sin `dangerouslySetInnerHTML`.
   - El backend rechaza plantillas con tokens desconocidos o mal formados (`400 invalid_token`).
   - El renderer del frontend deja el token literal si lo desconoce (defensa ante datos viejos) y nunca interpreta HTML.
   - `{{cuota.monto_formateado}}`, `{{deuda.total_formateado}}` y `{{pago.fecha_formateada}}` usan el formato de moneda/fecha configurado en la app.
7. **REQ-007 — Persistencia en JSON**: los campos de la plantilla se guardan como `fields_json` (TEXT) y los valores de la asignación como `values_json` (TEXT). La API expone `fields` y `values` como JSON tipado. Límites: `fields_json` ≤ 32 KB por plantilla; `values_json` ≤ 16 KB por deuda; valores de texto ≤ 500 caracteres y de textarea ≤ 5000 caracteres.
8. **REQ-008 — Backend y API**:
   - Nuevas rutas autenticadas (todas con `authMiddleware`):
     - `GET /api/debt-information-templates` — lista con `field_count` y `usage_count`.
     - `POST /api/debt-information-templates` — crea.
     - `GET /api/debt-information-templates/{id}` — detalle con `fields` y `usage_count`.
     - `PUT /api/debt-information-templates/{id}` — actualiza.
     - `DELETE /api/debt-information-templates/{id}` — elimina o devuelve `409 template_in_use`.
     - `GET /api/debt-information-templates/tokens` — catálogo de tokens para el constructor.
     - `GET /api/debts/{id}/payment-information` — asignación actual; si no existe, `200` con `template_id: null`, `template: null`, `values: {}` (no 404).
     - `PUT /api/debts/{id}/payment-information` — guarda `template_id` y `values`.
     - `DELETE /api/debts/{id}/payment-information` — quita la asignación (`204`).
   - Errores consistentes: `400 validation_error`/`invalid_token`, `404 not_found`, `409 template_name_exists`/`template_in_use`.
   - No registrar en logs los valores de la información ni datos financieros; solo IDs, códigos de error y conteos.
9. **REQ-009 — Borrado seguro y nombres únicos**:
   - El nombre de plantilla es único sin distinguir mayúsculas (`COLLATE NOCASE`).
   - No se puede eliminar una plantilla asignada a una o más deudas; la UI muestra el conteo de uso y un toast explicativo.
   - Al eliminar una deuda, su asignación se elimina en cascada. Al eliminar una plantilla sin uso, no quedan registros huérfanos.
10. **REQ-010 — Convenciones de UI obligatorias**:
    - **Inputs**: tokens del tema (`bg-card`, `text-text`, `text-text-secondary`), sin colores hardcodeados. Verificar darkmode (texto y placeholders legibles).
    - **Select**: usar `frontend/src/components/Select.tsx`; prohibido `<select>` nativo. El build ejecuta `frontend/scripts/check-ui-rules.cjs`.
    - **Modales**: no se cierran con click fuera; overlay bloqueante en `z-[60]`; cierre visible (X/Cerrar/Cancelar) en todos los estados; confirmaciones anidadas en `z-[80]`; `Select` en `z-[90]`; toasts en `z-[100]`. El `DeleteModal` enfoca el input y acepta Enter cuando la palabra es válida.
    - **Navegación**: registrar en `BACK_ROUTES` la jerarquía Deudas → Plantillas → Plantillas de información → constructor. La vuelta va en la flecha del header, nunca como link "← Título" en el contenido:
       - `/deudas/plantillas` → `/deudas`.
       - `/deudas/plantillas/informacion` → `/deudas/plantillas`.
       - `/deudas/plantillas/informacion/new|edit/:id` → `/deudas/plantillas/informacion`.
    - **Listados**: cards + menú de 3 puntos + `EmptyCard` con título, descripción y botón. Acciones rotuladas "Editar" (`app.edit`), nunca "Editar plantilla" en el menú.
    - **i18n**: todas las cadenas nuevas en `frontend/public/i18n/es.json` y `frontend/public/i18n/en.json` (fuente de verdad). Después del cambio, `npm run build` en `frontend/` y verificación de que las claves se sirven.
11. **REQ-011 — Privacidad y seguridad**:
    - No commitear PII, cuentas bancarias, nombres de titulares, montos ni descripciones reales en código, tests, fixtures, specs, issues, capturas o logs.
    - Todos los endpoints requieren sesión autenticada; no hay endpoints públicos nuevos.
    - Los valores se guardan en texto plano en SQLite, igual que el resto de los datos financieros de la app. Esta spec no agrega cifrado en reposo; se documenta como riesgo y se mitiga con autenticación, acceso de red restringido y ausencia de logs.
    - El renderer no interpreta HTML ni ejecuta expresiones; los valores se muestran como texto.
12. **REQ-012 — Pruebas y validación**:
    - Tests unitarios Go de validación, storage, servicio y handlers (incluida autenticación y errores 400/404/409).
    - Test de migración `0046` up/down (crear, verificar tablas/índices, revertir sin tocar `debts`/`debt_bills`).
    - Tests de utilidades frontend con `node --test`: slugify de claves, resolución de tokens, validación de valores y límites.
    - Prueba manual contra una **copia de la DB de producción en `/tmp`** (nunca contra `data/app.db` local ni escribiendo en producción), con datos sintéticos agregados a la copia.
    - `go test ./...`, `go vet ./...`, `gofmt`, `cd frontend && npm run build` (incluye `check:ui`) y `npm run check:ui`.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-013**: Copiar al portapapeles un valor individual y "Copiar todo" en la pestaña Información, con feedback por toast.
2. **REQ-014**: Vista previa en vivo en el constructor de plantillas, usando datos sintéticos de ejemplo, para ver cómo se renderiza antes de guardar.
3. **REQ-015**: Duplicar una plantilla existente (útil para variantes por banco) y exportar/importar la definición como JSON.
4. **REQ-016**: Guardar una instantánea de la información mostrada al confirmar el pago (nueva tabla o campo JSON por cuota pagada), para conservar qué datos se usaron en cada pago aunque la plantilla o los valores cambien después.
5. **REQ-017**: Permitir editar los valores de la información desde la pestaña Información del modal de pago, con guardado explícito y validación, sin salir del flujo de pago.
6. **REQ-018**: Mostrar un indicador en la card y en el detalle de la deuda cuando tiene forma de pago configurada, con el nombre de la plantilla.
7. **REQ-019**: Selector de tokens en el constructor (insertar en el cursor) y listado de tokens con ejemplo y descripción.
8. **REQ-020**: Búsqueda/filtro de plantillas por nombre y orden configurable.
9. **REQ-021**: Marcar una plantilla como predeterminada para precargarla al asignar una deuda nueva.
10. **REQ-022**: Reutilizar el renderer de información para otros módulos (servicios/facturas) sin duplicar lógica.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-023**: Campos condicionales (mostrar/ocultar según otro campo).
2. **REQ-024**: Nuevos tipos de campo: `url`, `email`, `phone`, archivo/imagen.
3. **REQ-025**: Versionado de plantillas con migración de valores y snapshot por versión.
4. **REQ-026**: Múltiples formas de pago por deuda con selección en el momento de pagar.
5. **REQ-027**: Generación de PDF/QR con los datos de la forma de pago (solo si se define una spec aparte).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: renderizar una plantilla es O(cantidad de campos), con un máximo de 30; no agrega llamadas externas. Las consultas de lista usan `COUNT` agregado, sin N+1.
- **Memoria**: sin dependencias nuevas; JSON pequeño y acotado; impacto esperado < 30 KB de código en el bundle.
- **Almacenamiento**: dos tablas nuevas; como máximo 30 campos por plantilla y 16 KB de valores por deuda. Sin archivos ni binarios.
- **Seguridad**: autenticación en todos los endpoints; sin logs de valores; sin HTML/`eval`; validación estricta en backend y frontend.
- **iHost**: solo Go stdlib + SQLite; sin Node en runtime; migración compatible con SQLite; build multi-arch (`amd64`, `arm/v7`, `arm64`) intacto.
- **Disponibilidad**: funciona sin red externa; no bloquea el pago si la información no está configurada.
- **Compatibilidad**: móvil desde 375 px, darkmode, i18n es/en, teclado y lectores de pantalla básicos (labels asociados, tabs con `role`/`aria-selected`).

---

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se analizó el módulo de deudas y el patrón de formularios dinámicos de automatizaciones en `main`:

- **Detalle de deuda**: `frontend/src/pages/DebtBillsPage.tsx` carga `api.debts.get(id)` y `api.debts.listBills(id)`, muestra las cuotas y abre `DebtPayModal` desde el menú de cada cuota.
- **Modal de pago actual**: `frontend/src/components/DebtPayModal.tsx` solo captura `paid_at`, `payment_reference`, factura y comprobante; no existe un campo de forma de pago ni una pestaña de información.
- **Modelo de deuda**: `internal/models/debt.go` y `internal/models/debt_bill.go`; la tabla `debts` y `debt_bills` se crea en `migrations/0024_create_debts_and_debt_bills.up.sql`. La política del proyecto prohíbe modificar el esquema de estas dos tablas; los cambios deben ir en tablas nuevas.
- **API de deudas**: `internal/api/debt_handlers.go` y las rutas de `internal/api/routes.go` (listar, obtener, crear, editar, borrar, cuotas, pagar, comprobantes).
- **Patrón de formulario dinámico**: `frontend/src/pages/AutomationFormPage.tsx` obtiene `AutomationCredentialField[]` con `api.automation.pluginSchema(plugin_name)` y renderiza los campos según `type`/`required`/`description`. El tipo `AutomationCredentialField` (`frontend/src/types/index.ts`) es `{ key, label, type: 'text' | 'password', required, secret, description }`. Este es el antecedente directo del "formulario distinto según el connector": el esquema define qué campos se muestran.
- **Componentes reutilizables**: `Select` (dropdown custom con portal y darkmode), `CardMenu`, `CreateMenu`, `DeleteModal` (confirmación por palabra con foco y Enter), `EmptyCard`/patrón de listados, `PdfUploadField`.
- **Navegación**: `BACK_ROUTES` en `frontend/src/components/DashboardLayout.tsx`; `App.tsx` define las rutas de `deudas`.
- **Reglas de UI**: `docs/project-rules.md` §4.3-§4.7 y `AGENTS.md` (tokens de tema, `Select`, modales, vuelta atrás, i18n, privacidad).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| Campos bancarios hardcodeados en el modal de pago | Rápido, sin tablas nuevas | No cubre otras formas de pago, no es un sistema de plantillas y no cumple "formularios construibles" | ❌ Rechazada |
| Plantillas reutilizables + valores JSON por deuda | Flexible, cumple el requerimiento, bajo costo en SQLite, sin dependencias | Requiere constructor y validación | ✅ Seleccionada |
| Tablas normalizadas para campos y valores | Consultable, integridad referencial por campo | Más tablas, más joins y más código; no se necesita para renderizar | ❌ Rechazada para P0 |
| Plantillas por deuda (no reutilizables) | Aisladas | Duplicación y mantenimiento; el usuario pidió un sistema de plantillas | ❌ Rechazada |
| Plantillas globales + asignación por deuda | Reutilización, una deuda = una forma de pago | Editar una plantilla afecta a las deudas que la usan | ✅ Seleccionada (con aviso y pruebas) |
| Agregar columnas a `debts`/`debt_bills` | Menos tablas | Prohibido por política del proyecto y frágil para múltiples campos | ❌ Rechazada |
| Renderizado en backend | Un solo lugar de verdad | Necesita ida y vuelta por cuota, fecha de pago y moneda; más CPU en iHost | ❌ Rechazada |
| Renderizado en frontend con tokens | Instantáneo, offline, sin requests extra | Hay que mantener el catálogo de tokens en Go y TS | ✅ Seleccionada |
| Librería de formularios/drag & drop | Menos código de UI | Dependencia y bundle extra en un iHost con recursos limitados | ❌ Rechazada |
| Botones subir/bajar nativos | Sin dependencias, accesible | Menos "moderno" que drag & drop | ✅ Seleccionada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001 — Plantillas como esquema declarativo de formulario**
- **Contexto**: el usuario necesita construir formularios distintos para cada forma de pago, como los formularios de automatización que dependen del connector.
- **Decisión**: una plantilla es un `name` + `description` + lista ordenada de campos tipados. El frontend renderiza el formulario y la vista de información a partir de ese esquema, sin código específico por banco o método.
- **Consecuencias**: agregar un tipo de campo requiere tocar el renderer y la validación; no requiere migraciones ni nuevas pantallas. El catálogo de tipos queda acotado a 7 para P0.

**ADR-002 — Plantillas globales, asignación por deuda**
- **Contexto**: la misma forma de pago (por ejemplo, transferencia a un banco) puede repetirse en varias deudas; los valores (cuenta, titular) pueden ser distintos.
- **Decisión**: las plantillas se definen una vez y se asignan a cada deuda con valores propios. Una deuda tiene como máximo una asignación en P0.
- **Consecuencias**: reutilización y consistencia; editar la plantilla impacta a todas las deudas asignadas, por eso el constructor avisa el uso y P1 evalúa versionado.

**ADR-003 — JSON para definición y valores**
- **Contexto**: el renderizado necesita el esquema completo y los valores por clave; no se consultan campos individuales desde SQL.
- **Decisión**: `fields_json` y `values_json` como TEXT con JSON validado. Se agregan límites de tamaño y cantidad.
- **Consecuencias**: menos tablas y consultas; se pierde la capacidad de hacer SQL sobre campos individuales, que no es un requisito. Los valores se validan en la capa de servicio antes de persistir.

**ADR-004 — Tablas nuevas, sin tocar `debts`/`debt_bills`**
- **Contexto**: la política del proyecto prohíbe modificar el esquema de `debts` y `debt_bills`; además, una relación 1:1 y una 1:N se modelan naturalmente aparte.
- **Decisión**: crear `debt_information_templates` y `debt_payment_information` en la migración `0046`. No hay `ALTER TABLE` sobre las tablas existentes.
- **Consecuencias**: migración segura y reversible; el borrado de una deuda elimina su asignación por FK; el borrado de una plantilla en uso se bloquea.

**ADR-005 — Resolución de tokens en el cliente**
- **Contexto**: los valores calculados dependen de la cuota seleccionada, la fecha de pago que el usuario está eligiendo y el formato de moneda configurado; el backend no necesita renderizar.
- **Decisión**: el backend valida los tokens y guarda la expresión; el frontend resuelve la interpolación con `debt`, `bill`, `symbol`, `lang` y `paidAt`.
- **Consecuencias**: render instantáneo y offline; el catálogo de tokens se mantiene en Go (validación) y TS (render), con tests que cubren ambos. No hay `eval` ni HTML.

**ADR-006 — Plantillas dentro de Deudas, con un hub intermedio**
- **Contexto**: las plantillas son específicas de la información de pago de deudas; el usuario las va a administrar junto con sus deudas y pidió que el acceso sea **Deudas → 3 puntos → Plantillas → Plantillas de información**. Se prevé que más adelante existan otros tipos de plantillas de deudas (contratos, recordatorios, etc.).
- **Decisión**: el menú de acciones de Deudas incluye **Plantillas**, que abre el hub `/deudas/plantillas`. El hub muestra cards por tipo; en P0 una sola card, **Plantillas de información**, que abre el listado `/deudas/plantillas/informacion`. La creación/edición vive en `/deudas/plantillas/informacion/new|edit/:id`. Sin entrada nueva en el sidebar.
- **Consecuencias**: un nivel extra de navegación ahora, pero el menú de Deudas no se llena de opciones cuando se agreguen otros tipos; hay que registrar las tres rutas en `BACK_ROUTES` y cuidar el orden de `/deudas/:id` vs `/deudas/plantillas` (React Router prioriza el segmento estático, y además se declaran antes).

**ADR-007 — Pestaña Información de solo lectura en P0**
- **Contexto**: el flujo principal es pagar; la información se configura una vez por deuda y se consulta al pagar. Editar valores en el modal de pago agrega validación y estados.
- **Decisión**: la pestaña Información solo renderiza; la edición se hace en el modal de forma de pago. P1 evalúa edición inline.
- **Consecuencias**: flujo de pago más seguro (no se puede disparar el pago desde Información) y menor superficie de error.

**ADR-008 — Sin dependencias nuevas**
- **Contexto**: iHost con recursos limitados; el repo ya tiene componentes suficientes.
- **Decisión**: formularios controlados con React, reordenamiento con botones, validación propia y `Select`/modales existentes. Backend con `net/http`, `database/sql` y `encoding/json`.
- **Consecuencias**: más código propio de validación, pero sin peso de bundle ni superficie de dependencias.

---

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
┌─────────────────────────────┐        REST (JSON)        ┌──────────────────────────────┐
│ Frontend React              │ ────────────────────────> │ Backend Go                   │
│                             │                           │                              │
│ DeudasPage                  │  /api/debt-information-   │ debt_information_handlers    │
│  └─ TemplatesHubPage        │  templates                │  └─ DebtInformationService   │
│      └─ InformationTemplates│                           │      ├─ validación           │
│          └─ TemplateForm    │                           │      └─ DebtInformationStorage│
│ Deudas/:id                  │  /api/debts/{id}/         │           └─ SQLite          │
│  ├─ DebtPaymentInfoModal    │  payment-information      │                              │
│  └─ DebtPayModal            │                           │                              │
│      └─ DebtInformationView │                           └──────────────────────────────┘
│          └─ token resolver  │
└─────────────────────────────┘
```

### 4.2 Componentes

#### 4.2.1 Backend

| Componente | Responsabilidad | Ubicación |
|---|---|---|
| Migración 0046 | Crear tablas e índices; down reversible | `migrations/0046_create_debt_information_templates.up.sql` / `.down.sql` |
| Modelos | Structs de plantilla, campo, asignación y tokens | `internal/models/debt_information.go` |
| Storage | CRUD SQL, conteo de uso, upsert por deuda, borrado | `internal/storage/debt_information.go` |
| Servicio | Validación de campos/valores/tokens, reglas de negocio | `internal/services/debt_information.go` |
| Handlers | Parseo HTTP, respuestas y códigos de error | `internal/api/debt_information_handlers.go` |
| Rutas | Registro con `authMiddleware` | `internal/api/routes.go` |
| Wiring | Construcción e inyección del servicio/handler | `internal/api/handlers.go` y `cmd/server/main.go` (según patrón existente) |
| Tests | Validación, storage, handlers, migración | `internal/services/debt_information_test.go`, `internal/api/debt_information_handlers_test.go`, `internal/db/migrate_046_test.go` |

#### 4.2.2 Frontend

| Componente | Responsabilidad | Ubicación |
|---|---|---|
| Tipos | `DebtInformationTemplate`, `DebtInformationField`, `DebtPaymentInformation`, tokens | `frontend/src/types/index.ts` |
| Cliente API | Métodos de plantillas, tokens y asignación | `frontend/src/api/index.ts` |
| Utilidades | Slugify, resolución de tokens, validación, formato | `frontend/src/utils/debtInformation.ts` |
| Renderer de campos | Input por tipo + vista de solo lectura | `frontend/src/components/DebtInformationField.tsx` |
| Vista de información | Lista label/valor con valores resueltos | `frontend/src/components/DebtInformationView.tsx` |
| Hub de plantillas | Cards por tipo de plantilla; P0: "Plantillas de información" | `frontend/src/pages/DebtTemplatesHubPage.tsx` |
| Lista de plantillas de información | Cards, menú, `EmptyCard`, alta/edición | `frontend/src/pages/DebtInformationTemplatesPage.tsx` |
| Constructor | Formulario de plantilla y campos | `frontend/src/pages/DebtInformationTemplateFormPage.tsx` |
| Modal de asignación | Elegir plantilla, completar valores, guardar/quitar | `frontend/src/components/DebtPaymentInformationModal.tsx` |
| Modal de pago | Pestañas Pagar/Información | `frontend/src/components/DebtPayModal.tsx` (modificar) |
| Detalle de deuda | Acción "Forma de pago" y carga de la asignación | `frontend/src/pages/DebtBillsPage.tsx` (modificar) |
| Página de deudas | Opción "Plantillas" en el menú de 3 puntos | `frontend/src/pages/DeudasPage.tsx` (modificar) |
| Rutas | `/deudas/plantillas`, `/deudas/plantillas/informacion`, `/deudas/plantillas/informacion/new`, `/deudas/plantillas/informacion/edit/:id` | `frontend/src/App.tsx` |
| Vuelta atrás | Patrones de rutas de plantillas | `frontend/src/components/DashboardLayout.tsx` |
| i18n | Claves es/en | `frontend/public/i18n/es.json`, `frontend/public/i18n/en.json` |
| Tests | Utilidades puras | `frontend/tests/debtInformation.test.ts` (patrón `node --test`) |

### 4.3 Modelo de datos

```sql
-- migrations/0046_create_debt_information_templates.up.sql
-- Plantillas reutilizables de información de pago (SPEC-131).
-- No modifica debts ni debt_bills.

CREATE TABLE IF NOT EXISTS debt_information_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL COLLATE NOCASE UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    fields_json TEXT NOT NULL DEFAULT '[]',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS debt_payment_information (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    debt_id INTEGER NOT NULL UNIQUE,
    template_id INTEGER NOT NULL,
    values_json TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (debt_id) REFERENCES debts(id) ON DELETE CASCADE,
    FOREIGN KEY (template_id) REFERENCES debt_information_templates(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_debt_payment_information_template
    ON debt_payment_information(template_id);
```

```sql
-- migrations/0046_create_debt_information_templates.down.sql
DROP TABLE IF EXISTS debt_payment_information;
DROP TABLE IF EXISTS debt_information_templates;
```

Entidades:

```
DebtInformationTemplate
- id: int64
- name: string (único, NOCASE)
- description: string
- fields: []DebtInformationField (JSON)
- field_count: int (calculado)
- usage_count: int (calculado)
- created_at / updated_at: time.Time

DebtInformationField
- key: string
- label: string
- type: text | textarea | number | date | select | checkbox | computed
- required: bool
- placeholder: string (opcional)
- default_value: string (opcional)
- help_text: string (opcional)
- options: []string (solo select)
- value_template: string (solo computed)

DebtPaymentInformation
- debt_id: int64 (único)
- template_id: int64
- values: map[string]any (JSON)
- created_at / updated_at: time.Time
```

### 4.4 Ejemplo de plantilla y renderizado

Plantilla sintética "Transferencia bancaria":

```json
{
  "name": "Transferencia bancaria",
  "description": "Datos para transferir al acreedor",
  "fields": [
    {
      "key": "monto",
      "label": "Monto",
      "type": "computed",
      "value_template": "{{cuota.monto_formateado}}"
    },
    {
      "key": "descripcion",
      "label": "Descripción",
      "type": "computed",
      "value_template": "{{deuda.descripcion}} - Cuota {{cuota.numero}}"
    },
    {
      "key": "banco",
      "label": "Banco",
      "type": "text",
      "required": true
    },
    {
      "key": "cuenta",
      "label": "# Cuenta",
      "type": "text",
      "required": true
    },
    {
      "key": "nombre",
      "label": "Nombre",
      "type": "text",
      "required": true
    }
  ]
}
```

Valores sintéticos asignados a una deuda:

```json
{
  "banco": "Banco Demo",
  "cuenta": "demo-account-001",
  "nombre": "Titular Demo"
}
```

Render en la pestaña Información al pagar la cuota 12 de "Lote Demo 1":

```
Monto: $100.00
Descripción: Lote Demo 1 - Cuota 12
Banco: Banco Demo
# Cuenta: demo-account-001
Nombre: Titular Demo
```

> Todos los datos del ejemplo son sintéticos (`Banco Demo`, `demo-account-001`, `Titular Demo`, `Lote Demo 1`, `$100.00`). No corresponden a una persona, cuenta o deuda real.

### 4.5 APIs / Contratos

#### `GET /api/debt-information-templates`

Response 200:

```json
[
  {
    "id": 1,
    "name": "Transferencia bancaria",
    "description": "Datos para transferir al acreedor",
    "field_count": 5,
    "usage_count": 2,
    "created_at": "2026-11-08T10:00:00Z",
    "updated_at": "2026-11-08T10:00:00Z"
  }
]
```

#### `GET /api/debt-information-templates/{id}`

Response 200:

```json
{
  "id": 1,
  "name": "Transferencia bancaria",
  "description": "Datos para transferir al acreedor",
  "fields": [
    { "key": "monto", "label": "Monto", "type": "computed", "value_template": "{{cuota.monto_formateado}}" },
    { "key": "descripcion", "label": "Descripción", "type": "computed", "value_template": "{{deuda.descripcion}} - Cuota {{cuota.numero}}" },
    { "key": "banco", "label": "Banco", "type": "text", "required": true },
    { "key": "cuenta", "label": "# Cuenta", "type": "text", "required": true },
    { "key": "nombre", "label": "Nombre", "type": "text", "required": true }
  ],
  "usage_count": 2,
  "created_at": "2026-11-08T10:00:00Z",
  "updated_at": "2026-11-08T10:00:00Z"
}
```

#### `POST /api/debt-information-templates` / `PUT /api/debt-information-templates/{id}`

Request:

```json
{
  "name": "Transferencia bancaria",
  "description": "Datos para transferir al acreedor",
  "fields": [
    { "key": "monto", "label": "Monto", "type": "computed", "value_template": "{{cuota.monto_formateado}}" },
    { "key": "banco", "label": "Banco", "type": "text", "required": true }
  ]
}
```

Response 200/201: el objeto completo de la plantilla.

Errores:

```json
{ "error": "validation_error", "message": "El campo 'banco' es obligatorio" }
{ "error": "invalid_token", "message": "Token desconocido: {{cuota.total}}" }
{ "error": "template_name_exists", "message": "Ya existe una plantilla con ese nombre" }
{ "error": "template_in_use", "message": "La plantilla está asignada a 2 deuda(s)" }
```

#### `GET /api/debt-information-templates/tokens`

Response 200:

```json
[
  { "key": "deuda.descripcion", "label": "Descripción de la deuda", "example": "Lote Demo 1" },
  { "key": "cuota.numero", "label": "Número de cuota", "example": "12" },
  { "key": "cuota.monto_formateado", "label": "Monto de la cuota", "example": "$100.00" }
]
```

#### `GET /api/debts/{id}/payment-information`

Response 200 (sin asignación):

```json
{
  "debt_id": 10,
  "template_id": null,
  "template": null,
  "values": {}
}
```

Response 200 (con asignación):

```json
{
  "debt_id": 10,
  "template_id": 1,
  "template": {
    "id": 1,
    "name": "Transferencia bancaria",
    "description": "Datos para transferir al acreedor",
    "fields": [
      { "key": "monto", "label": "Monto", "type": "computed", "value_template": "{{cuota.monto_formateado}}" },
      { "key": "banco", "label": "Banco", "type": "text", "required": true }
    ]
  },
  "values": {
    "banco": "Banco Demo"
  },
  "updated_at": "2026-11-08T10:00:00Z"
}
```

#### `PUT /api/debts/{id}/payment-information`

Request:

```json
{
  "template_id": 1,
  "values": {
    "banco": "Banco Demo",
    "cuenta": "demo-account-001",
    "nombre": "Titular Demo"
  }
}
```

Response 200: el mismo formato que el GET.

#### `DELETE /api/debts/{id}/payment-information`

Response 204.

### 4.6 Flujos de UI

1. **Crear plantilla**: Deudas → menú de acciones (3 puntos) → **Plantillas** → hub de plantillas → **Plantillas de información** → "Nueva plantilla" → nombre/descripción → agregar campos → guardar. La lista queda disponible para asignar. El hub es el único punto de entrada desde Deudas; la card "Plantillas de información" muestra la cantidad de plantillas existentes.
2. **Asignar forma de pago**: `/deudas/:id` → menú de acciones → "Forma de pago" → elegir plantilla → completar valores → guardar. Si no hay plantillas, el modal muestra un estado vacío con "Crear plantilla".
3. **Pagar y consultar**: cuota pendiente → menú → "Pagar" → pestaña **Pagar** (flujo actual) o pestaña **Información** (datos resueltos para esa cuota). El botón Pagar solo existe en la pestaña Pagar.
4. **Editar plantilla en uso**: se muestra el aviso "Los cambios afectan a N deuda(s)"; al guardar, las deudas asignadas usan la nueva definición. Los valores de claves eliminadas se conservan en JSON pero dejan de renderizarse; P1 evaluará versionado/snapshot.

### 4.7 Validaciones y límites

| Regla | Backend | Frontend |
|---|---|---|
| Nombre de plantilla 1-100, único NOCASE | ✅ | ✅ |
| 1-30 campos | ✅ | ✅ |
| `key` regex y única | ✅ | ✅ (generada) |
| Tipos permitidos | ✅ | ✅ |
| `select` con 1-50 opciones | ✅ | ✅ |
| `computed` con tokens válidos | ✅ | ✅ |
| Valores requeridos y de tipo correcto | ✅ | ✅ |
| `text` ≤ 500 / `textarea` ≤ 5000 | ✅ | ✅ |
| `values_json` ≤ 16 KB | ✅ | ✅ (aviso) |
| Plantilla en uso no se elimina | ✅ | ✅ (deshabilita/toast) |
| Valores desconocidos al guardar | Se ignoran (solo se persisten claves del template) | — |

### 4.8 i18n (claves nuevas)

Agregar en `frontend/public/i18n/es.json` y `en.json`:

- Navegación: `deudas.templates_menu` ("Plantillas"/"Templates"), `deudas.templates_hub_title` ("Plantillas"/"Templates"), `deudas.templates_info` ("Plantillas de información"/"Information templates"), `deudas.templates_info_description` ("Formularios reutilizables con los datos de la forma de pago"/"Reusable forms with payment method details"), `deudas.templates_info_count` (`{n} plantilla(s)`/`{n} template(s)`), `deudas.templates_info_updated` ("Última actualización"/"Last updated").
- Listado y constructor: `deudas.template_create`, `deudas.template_edit`, `deudas.template_empty`, `deudas.template_empty_hint`, `deudas.template_name`, `deudas.template_description`, `deudas.template_usage`, `deudas.template_fields`, `deudas.template_add_field`, `deudas.template_field_*` (label, key, type, required, placeholder, default, help, options, expression, remove, move_up, move_down), `deudas.field_type_*` (text, textarea, number, date, select, checkbox, computed), `deudas.template_saved`, `deudas.template_deleted`, `deudas.template_in_use`, `deudas.template_name_exists`.
- `deudas.payment_method`, `deudas.payment_method_empty`, `deudas.payment_method_configure`, `deudas.payment_method_saved`, `deudas.payment_method_removed`, `deudas.payment_method_clear`, `deudas.payment_method_clear_confirm`, `deudas.payment_method_template`, `deudas.payment_method_values`, `deudas.payment_method_preview`.
- `deudas.pay_tab_pay` ("Pagar"/"Pay"), `deudas.pay_tab_information` ("Información"/"Information"), `deudas.payment_info_empty`, `deudas.payment_info_empty_hint`, `deudas.payment_info_configure`, `deudas.payment_info_value_missing`.
- `deudas.info_copy`, `deudas.info_copied`, `deudas.info_copy_all` (P1).
- Mensajes de validación: `deudas.validation_required`, `deudas.validation_key`, `deudas.validation_duplicate_key`, `deudas.validation_options`, `deudas.validation_token`, `deudas.validation_max_fields`, `deudas.validation_value_length`.
- Reutilizar `app.edit`, `app.delete`, `app.cancel`, `app.close`, `app.save`, `app.loading`, `app.actions`, `errors.generic` cuando existan.

### 4.9 Dependencias

- **Internas**: `DebtService`/`DebtStorage` (para verificar que la deuda exista), `authMiddleware`, `Select`, `CardMenu`, `CreateMenu`, `DeleteModal`, `EmptyCard`, `useToast`, `useI18nStore`, `useCurrencyFormatStore`.
- **Externas**: ninguna nueva. Go stdlib (`encoding/json`, `database/sql`, `net/http`) y React/Tailwind ya presentes.

---

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Se puede crear una plantilla con nombre, descripción y al menos un campo de cada tipo soportado (`text`, `textarea`, `number`, `date`, `select`, `checkbox`, `computed`), guardarla y verla en la lista con su cantidad de campos.
- [ ] CA-002: Se puede editar una plantilla, agregar/quitar/reordenar campos con los botones subir/bajar y guardar; al recargar, el orden y la configuración persisten.
- [ ] CA-003: Se puede asignar una plantilla a una deuda desde el menú "Forma de pago", completar los valores, guardar y volver a abrir el modal con los valores precargados.
- [ ] CA-004: El modal de pago muestra dos pestañas, **Pagar** e **Información**; la pestaña Información renderiza la plantilla asignada con los valores guardados y el nombre de la plantilla.
- [ ] CA-005: Con una deuda de varias cuotas pendientes, al abrir la pestaña Información de la cuota N, los campos calculados muestran el monto y el número de cuota N (por ejemplo `Descripción: Lote Demo 1 - Cuota 12` y `Monto: $100.00`). Al abrir la cuota N+1, los valores cambian a N+1 y al monto correspondiente.
- [ ] CA-006: Los campos `computed` resuelven `{{deuda.*}}`, `{{cuota.*}}`, `{{pago.*}}` y `{{sistema.moneda}}`; los tokens desconocidos se rechazan al guardar la plantilla con un mensaje claro.
- [ ] CA-007: Los campos requeridos y los tipos se validan en frontend y backend; un payload inválido devuelve `400 validation_error` y no persiste.
- [ ] CA-008: Si la deuda no tiene forma de pago, la pestaña Información muestra un estado vacío y el botón Configurar; el flujo de pago existente sigue funcionando sin bloqueos.
- [ ] CA-009: Desde la pestaña Información no se puede confirmar el pago; el botón Pagar solo está en la pestaña Pagar.
- [ ] CA-010: No se puede eliminar una plantilla asignada a una deuda (`409 template_in_use`); el nombre de plantilla duplicado devuelve `409 template_name_exists`.
- [ ] CA-011: Quitar la forma de pago de una deuda elimina la asignación (`204`) y la pestaña Información vuelve al estado vacío.
- [ ] CA-012: Los datos sobreviven a un reinicio del server y a una recarga del navegador; se almacenan en `debt_information_templates.fields_json` y `debt_payment_information.values_json`.
- [ ] CA-013: La migración `0046` crea las dos tablas y el índice; el `down` las elimina en orden y no altera `debts` ni `debt_bills`.
- [ ] CA-014: Todos los endpoints nuevos exigen sesión; sin cookie devuelven `401`.
- [ ] CA-015: La lista de plantillas de información usa cards + menú de 3 puntos y `EmptyCard` cuando no hay registros; la acción de edición se rotula "Editar" (`app.edit`). El hub `/deudas/plantillas` muestra la card "Plantillas de información" con la cantidad de plantillas.
- [ ] CA-NAV: En `/deudas`, el menú de 3 puntos incluye **Plantillas**; al abrirlo se llega al hub `/deudas/plantillas`; al entrar a "Plantillas de información" se llega al listado; desde ahí se crea/edita una plantilla. La jerarquía es Deudas → Plantillas → Plantillas de información → constructor.
- [ ] CA-DARK: Los inputs, textareas, selects y cards usan tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y se verificó legibilidad de texto y placeholders en darkmode.
- [ ] CA-MODAL: El modal de forma de pago y el de pago no se cierran con click fuera; el overlay bloquea el fondo (`z-[60]`), hay cierre visible en todos los estados, las confirmaciones anidadas usan `z-[80]`, el `Select` `z-[90]` y los toasts `z-[100]`; el `DeleteModal` enfoca el input y acepta Enter con la palabra válida.
- [ ] CA-BACK: `/deudas/plantillas` vuelve a `/deudas`; `/deudas/plantillas/informacion` vuelve a `/deudas/plantillas`; `/deudas/plantillas/informacion/new|edit/:id` vuelve a `/deudas/plantillas/informacion`, mediante la flecha del header registrada en `BACK_ROUTES`; no hay links "← Título" en el contenido.
- [ ] CA-UI-SELECT: No se agregó ningún `<select>` nativo; `npm run check:ui` y el build de `frontend/` pasan.
- [ ] CA-I18N: Todas las cadenas nuevas están en `frontend/public/i18n/es.json` y `en.json`; el build las sirve y no hay textos hardcodeados visibles.
- [ ] CA-RESPONSIVE: Las pantallas de plantillas, el constructor y la pestaña Información son usables a 375 px, sin desbordes horizontales; los textos largos y números de cuenta se cortan o envuelven correctamente.
- [ ] CA-COMPAT: La deuda sin cuotas pendientes (finalizada) puede conservar/editar su forma de pago; la vista previa de campos calculados muestra la expresión cuando no hay cuota.
- [ ] CA-IHOST: No se agregaron dependencias; `go test ./...`, `go vet`, `gofmt` y `npm run build` pasan; el build multi-arch no cambia.
- [ ] CA-PRIVACY: No hay PII, cuentas, nombres, montos ni URLs reales en la spec, tests, fixtures, issues, capturas ni logs; los ejemplos usan `demo-account-001`, `Banco Demo`, `Titular Demo` y `Lote Demo 1`.

### 5.2 No funcionales

- [ ] CA-NF-001: Una plantilla de 30 campos se renderiza en < 50 ms en el iHost y no genera requests adicionales al abrir la pestaña Información.
- [ ] CA-NF-002: El endpoint de lista no hace N+1; usa agregados para `field_count`/`usage_count` (o una consulta por plantilla acotada).
- [ ] CA-NF-003: Los límites de tamaño se aplican en backend y el server responde con error controlado, sin panic ni consumo desmedido de memoria.
- [ ] CA-NF-004: No se registran valores de información en logs; solo IDs y códigos de error.

### 5.3 Testing

- **Unit tests Go**: `ValidateTemplate`, `ValidateValues`, `ResolveTokens` (validación), `SlugifyKey`; casos de límites, tokens inválidos, tipos incorrectos, required, select fuera de opciones y JSON corrupto.
- **Integration tests Go**: CRUD de plantillas, asignación/upsert por deuda, borrado bloqueado en uso, cascada al borrar deuda, auth, migración 0046 up/down.
- **Unit tests Frontend**: `resolveDebtInformationTokens`, `validateTemplateFields`, `validateDebtInformationValues`, `slugifyFieldKey`; casos con todos los tokens, valores faltantes, fechas, booleanos y textos largos.
- **Manual**: copia de producción en `/tmp`, crear plantilla sintética, asignarla a una deuda con varias cuotas, verificar cantidad/descripción por cuota, darkmode, móvil, modales, back y borrado bloqueado. Borrar la copia al terminar.
- **Regresión**: el pago normal (fecha/referencia/PDFs) sigue funcionando sin información configurada.

---

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|---|---|---|---|
| 1 | Migración 0046, modelos, storage y test de migración | 0.5 día | Ninguna |
| 2 | Servicio de validación, tokens y tests unitarios | 1 día | Fase 1 |
| 3 | Handlers, rutas, wiring y tests de API/auth | 0.5 día | Fase 2 |
| 4 | Tipos, cliente API, utilidades de tokens/validación y tests frontend | 0.5 día | Fase 3 |
| 5 | Hub de plantillas, listado de plantillas de información y constructor (cards, campos, orden, validación) | 1.5 días | Fase 4 |
| 6 | Modal de forma de pago e integración en el detalle de deuda | 1 día | Fase 5 |
| 7 | Pestañas Pagar/Información en `DebtPayModal` + renderer | 0.5 día | Fase 6 |
| 8 | i18n, darkmode, responsive, pruebas manuales contra copia de producción y ajustes | 0.5 día | Fase 7 |
| **Total** | | **~6 días** | |

### 6.2 Milestones

1. **MVP (P0)**: plantillas CRUD, asignación por deuda, pestaña Información con tokens dinámicos, validación, migración y pruebas. Es lo que se libera en esta spec.
2. **V1.0 (P1)**: copiar valores, vista previa, duplicar/exportar, snapshot por pago, edición desde la pestaña Información, indicador en cards y selector de tokens.
3. **P2**: campos condicionales, nuevos tipos, versionado, múltiples formas de pago y PDF/QR (cada uno con su propia spec si aplica).

### 6.3 Dependencias y coordinación

- **SPEC-054**: modelo y CRUD de deudas/cuotas.
- **SPEC-043**: acción Pagar y referencia de pago.
- **SPEC-122/123**: PDFs de comprobantes y contratos; no se modifican.
- **SPEC-126**: reglas de modales, capas y confirmación por palabra; cumplir.
- **SPEC-127/128**: cambios recientes en el modal de pago y la tarjeta de cuota; al implementar, partir de `main` actualizado y evitar duplicar helpers (`paymentTiming`, etc.).
- **SPEC-110/111**: patrón de formularios por esquema de plugin/connector; se reutiliza el concepto, no el código.

---

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| Exposición de datos bancarios/PII en el repo o logs | Media | Alto | Ejemplos sintéticos, revisión de `git diff --cached`, prohibición de loguear valores, auth en endpoints |
| XSS por plantillas o valores maliciosos | Baja | Alto | Interpolación de texto plano, React escapa, sin `dangerouslySetInnerHTML`, sin `eval`, validación de tokens |
| Editar una plantilla rompe valores existentes | Media | Medio | Valores por clave, claves inmutables, aviso de uso, renderer tolerante; P1 versionado/snapshot |
| Borrar una plantilla en uso deja deudas sin información | Baja | Medio | Bloqueo `409 template_in_use` y conteo de uso visible |
| Plantillas/valores demasiado grandes afectan al iHost | Baja | Medio | Límites de 30 campos, 32 KB/16 KB, textos acotados y validación backend |
| Cambios en `debts`/`debt_bills` por error | Baja | Alto | Migración solo con tablas nuevas; test que verifica que no hay `ALTER` y que las columnas existentes siguen igual |
| Conflictos con specs recientes de deudas | Media | Medio | Rebase sobre `main`, reutilizar `PaymentTimingLabel`/helpers existentes, no duplicar componentes |
| Modal de pago con pestañas rompe el flujo actual | Media | Medio | Tests manuales de pago normal, botón Pagar solo en la pestaña Pagar, estados de loading/saving |
| Selector nativo o modales que se cierran fuera | Baja | Medio | `Select` del repo, `check:ui`, revisión de overlay `z-[60]` y cierre visible |
| Pérdida de datos al cambiar de plantilla | Media | Medio | Conservar valores por clave, confirmar si hay valores, guardado explícito |
| Migración no reversible | Baja | Medio | Test up/down; drop en orden hijo→padre; no tocar tablas existentes |

---

## 8. Notas y Referencias

- **Archivos clave**:
  - `frontend/src/pages/DebtBillsPage.tsx`
  - `frontend/src/components/DebtPayModal.tsx`
  - `frontend/src/pages/DeudasPage.tsx`
  - `frontend/src/components/Select.tsx`
  - `frontend/src/components/DeleteModal.tsx`
  - `frontend/src/components/DashboardLayout.tsx` (`BACK_ROUTES`)
  - `frontend/src/pages/AutomationFormPage.tsx` (patrón de formulario por esquema)
  - `frontend/src/types/index.ts` (`AutomationCredentialField`)
  - `internal/models/debt.go`, `internal/models/debt_bill.go`
  - `internal/api/routes.go`, `internal/api/debt_handlers.go`
  - `migrations/0024_create_debts_and_debt_bills.up.sql`
  - `docs/project-rules.md` §4.3-§4.7 y §8
- **Specs relacionadas**: SPEC-043 (acción Pagar), SPEC-054 (deudas), SPEC-110/111 (automatizaciones y plugins), SPEC-122/123 (documentos), SPEC-126 (modales), SPEC-127/128 (pago y puntualidad).
- **Supuestos**:
  - Las plantillas son globales y reutilizables; una deuda tiene como máximo una forma de pago asignada en P0.
  - El nombre de la plantilla identifica el método (ej. "Transferencia bancaria"); no se agrega un enum separado de métodos.
  - La administración de plantillas vive bajo Deudas, no en Configuración, con la jerarquía **Deudas → 3 puntos → Plantillas → Plantillas de información**.
  - El hub `/deudas/plantillas` es una página de categorías; hoy tiene una sola card ("Plantillas de información") y queda preparado para futuros tipos de plantilla.
  - La pestaña Información es de solo lectura en P0.
- **Fuera de alcance**:
  - Ejecutar transferencias, integrarse con bancos, generar PDFs/QR o enviar datos por email/Telegram.
  - Cifrado en reposo de los valores.
  - Roles/permisos multiusuario.
  - Formas de pago para servicios/facturas (P1/P2 de reutilización).
  - Edición de la información desde la pestaña Información (P1).
- **Datos sintéticos**: cualquier ejemplo debe usar `demo-account-001`, `demo@example.com`, `Banco Demo`, `Titular Demo`, `Lote Demo 1`, `$100.00`, `F000000000000001`. No usar datos reales "porque son de prueba".

---

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|---|---|---|
| 2026-11-08 | opencode | Creación inicial de la especificación: forma de pago por deuda, plantillas de información construibles y pestaña Información en el modal de pago. |
| 2026-11-08 | opencode | Ajuste de navegación solicitado por el usuario: Deudas → menú de 3 puntos → **Plantillas** → hub → **Plantillas de información** → listado/constructor. |
| 2026-11-08 | opencode | Implementación P0 completa: migración 0046, modelos/storage/servicio/handlers/rutas, hub/listado/constructor, modal de forma de pago, pestaña Información, i18n es/en, tests Go/frontend y build. Servidor local de prueba en `:8088` con DB sintética; pendiente validación manual del usuario. |
| 2026-11-08 | opencode | Validación manual confirmada por el usuario. Se inicia el release: estado `pending_release`, merge a `main`, bump de versión e imagen Docker. |
