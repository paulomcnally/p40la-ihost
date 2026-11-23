---
title: "Alquiler de viviendas: página de casa, apartado, contratos, cobros en PDF y recordatorios"
id: "SPEC-140"
status: "in_progress"
author: "OpenCode"
created: "2026-11-23"
updated: "2026-11-23"
github_issue: 143
---

# Alquiler de viviendas: página de casa, apartado, contratos, cobros en PDF y recordatorios

**ID**: SPEC-140  
**Estado**: in_progress  
**Autor**: OpenCode  
**Creado**: 2026-11-23  
**Actualizado**: 2026-11-23

---

## 1. Resumen Ejecutivo

Hoy el sistema administra casas (`homes`) como una etiqueta con nombre y dirección, sin página de detalle ni relación con contratos. Un propietario que alquila necesita gestionar el ciclo completo: publicar la vivienda, apartarla para un interesado, firmar un contrato con plazo definido (6, 12 o N meses), cobrar la renta mes a mes, emitir documentos de cobro y comprobantes, guardar la documentación legal y automatizar recordatorios de mantenimiento. Nada de eso existe.

Esta spec agrega una **página de detalle por casa** con una pestaña **Alquiler**, y un módulo namespaced `realestate.rental` que implementa: reservas con vencimiento que bloquean la promoción de la vivienda, contratos con partes notificables, generación de cargos mensuales, facturas internas de cobro en PDF con numeración inmutable, recibos de pago, comprobantes adjuntos, documentos legales del contrato y recordatorios recurrentes (jardinería, mantenimientos, inspecciones) enviados por email.

El sistema es **agnóstico de jurisdicción**: los documentos son internos de cobro, no facturas fiscales. La fiscalidad, la moneda y los datos del emisor se configuran; el software no asume un país. En el iHost todo debe funcionar con SQLite, sin crones externos ni Node en runtime, con generación de PDF en Go puro y un pico de memoria acotado.

---

## 2. Cumplimiento de reglas

Reglas del proyecto aplicables a esta spec (índice: `docs/rules/00-index.md`).

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Go + SQLite + `net/http`; única dependencia nueva es una librería PDF en Go puro, justificada en §4.2 | `CA-ARCH-STACK` |
| `ARCH-FOLDERS` | Módulo nuevo en `internal/modules/realestate/rental/`; migraciones en `migrations/`; frontend en `frontend/src/` | `CA-ARCH-FOLDERS` |
| `ARCH-LAYERS` | `handlers → service → storage → model` dentro del módulo; sin SQL en handlers ni services | `CA-ARCH-LAYERS` |
| `ARCH-MODULES` | ID `realestate.rental`, ruta espejo, `module.go/model.go/storage.go/service.go/handlers.go/*_test.go`, registro en `all/register.go` | `CA-ARCH-MODULES` |
| `ARCH-GO` | `gofmt`, nombres en inglés, errores explícitos, `log/slog` | `CA-ARCH-GO` |
| `ARCH-IHOST` | PDF en Go puro, envío acotado, sin servicios externos obligatorios; estado en el volumen | `CA-ARCH-IHOST` |
| `UI-SELECT` | Vivienda, inmobiliaria, moneda, rol de parte, tipo de documento y frecuencia usan `Select` (buscable cuando aplica) | `CA-UI-SELECT` |
| `UI-SELECT-ADD` | Las opciones «Agregar inmobiliaria / agente / persona / categoría» dentro de un `Select` se rotulan `app.add` («Agregar») con ícono `plus`, usando `addOption()` | `CA-UI-SELECT-ADD` |
| `UI-NUMBER-INPUT` | Montos, porcentajes, días y cantidades usan `NumberInput`; permite vacío al editar y limpia ceros a la izquierda. Guardrail `npm run check:numbers`; legacy documentado en `number-input-legacy.txt` | `CA-UI-NUMBER-INPUT` |
| `UI-INPUT-TOKENS` | Inputs con `bg-card` / `text-text`; sin colores hardcodeados | `CA-UI-INPUT-TOKENS` |
| `UI-DARKMODE` | Formularios, modales y pestaña Alquiler verificados en darkmode | `CA-UI-DARKMODE` |
| `UI-TOGGLE` | «Publicar vivienda», «Renovación automática», «Notificar por email», «Recordatorio activo» usan `Toggle` | `CA-UI-TOGGLE` |
| `UI-LIST-PATTERN` | Listados de contratos/cargos/documentos con cards y `CardMenu`; sin formularios inline; `EmptyCard` local | `CA-UI-LIST-PATTERN` |
| `UI-CREATE-MENU` | Crear contrato/apartado/recordatorio desde `CreateMenu` del header o de la pestaña | `CA-UI-CREATE-MENU` |
| `UI-EDIT-LABEL` | Toda acción de edición usa `app.edit` («Editar») | `CA-UI-EDIT-LABEL` |
| `UI-I18N` | Claves nuevas en `frontend/public/i18n/{es,en}.json`, namespace `rental.*`; nunca en `public/i18n/`. Toda clave usada existe como string en ambos idiomas y no apunta a objetos padre; guardrail `npm run check:i18n` | `CA-UI-I18N` |
| `UI-STYLE` | Cards `rounded-ios` / `shadow-ios`, badges de estado y menú de 3 puntos | `CA-UI-STYLE` |
| `UI-GUARDRAIL` | `npm run check:ui` limpio; sin `<select>` ni checkbox nativo | `CA-UI-GUARDRAIL` |
| `UX-BACK` | `/home/:id`, `/leases/:id`, `/leases/:id/edit`, `/reservations/:id`, `/invoices/:id` registradas en `BACK_ROUTES`; sin «← Título» en contenido | `CA-UX-BACK` |
| `UX-MODAL` | Modales de pago, envío, anulación y documento no cierran por click fuera; overlay bloqueante; cierre visible siempre; z-index según escala | `CA-UX-MODAL` |
| `UX-MODAL-CONFIRM` | Anular factura y cancelar contrato usan confirmación por palabra con autofocus y Enter | `CA-UX-MODAL-CONFIRM` |
| `UX-DETAIL-ACTIONS` | Editar, terminar, anular y eliminar viven en el `CardMenu`; solo «Pagar» y «Generar factura» quedan como acciones primarias | `CA-UX-DETAIL-ACTIONS` |
| `UX-DETAIL-READONLY` | Detalle de casa, contrato, apartado y factura en modo lectura; editar navega a `/edit/:id` | `CA-UX-DETAIL-READONLY` |
| `UX-PREREQ` | Contrato requiere casa existente; factura requiere cargos; apartado requiere casa libre. Se valida en backend y frontend | `CA-UX-PREREQ` |
| `UX-EMPTY` | Casa sin contrato, contrato sin cargos y lista sin recordatorios muestran `EmptyCard` con acción | `CA-UX-EMPTY` |
| `UX-LOADING` | Skeletons en pestaña Alquiler; botones de emitir/enviar/pagar se deshabilitan en curso | `CA-UX-LOADING` |
| `UX-FEEDBACK` | Toasts de éxito y error de validación junto al campo | `CA-UX-FEEDBACK` |
| `UX-FORMS` | Guardar/Cancelar únicos; validación bloqueante; obligatorios marcados | `CA-UX-FORMS` |
| `DATA-SQLITE` | WAL, SQL crudo, migraciones en el volumen | `CA-DATA-SQLITE` |
| `DATA-MIGRATIONS` | `0051`–`0054` con `up`/`down` reversibles y test de migración | `CA-DATA-MIGRATIONS` |
| `DATA-NO-DROP` | No se toca `data/app.db`; pruebas en `/tmp` | `CA-DATA-NO-DROP` |
| `DATA-PRODCOPY` | Validación funcional contra copia de producción en `/tmp` (solo lectura) | `CA-DATA-PRODCOPY` |
| `DATA-PROTECTED-SCHEMA` | No se modifica `debts` ni `debt_bills`; el módulo tiene tablas propias | `CA-DATA-PROTECTED-SCHEMA` |
| `DATA-PROD-WRITE` | N/A: la implementación no escribe en la DB de producción; el deploy aplica migraciones de forma normal | — |
| `DATA-SSH` | N/A: no hay operación manual sobre el iHost en esta spec | — |
| `SEC-NO-PII` | Nombres, emails, teléfonos y direcciones viven solo en la DB local; no van al repo, tests, specs ni logs | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Ejemplos con `Inquilino Demo`, `demo@example.com`, `+1 555 0100`, `F000000000000001` | `CA-SEC-SYNTHETIC` |
| `SEC-PRODCOPY` | Copias de producción solo en `/tmp` y se borran al terminar | `CA-SEC-PRODCOPY` |
| `SEC-COMMIT` | `git diff --cached` sin PII, DBs, PDFs ni exports antes de commitear | `CA-SEC-COMMIT` |
| `SEC-HOOKS` | Hooks instalados; sin `--no-verify` | `CA-SEC-HOOKS` |
| `SEC-LOGS` | Logs sin direcciones, importes ni contenido de documentos; solo IDs y conteos | `CA-SEC-LOGS` |
| `TEST-LOCAL` | `go build`, `go test`, `npm run check:ui` y prueba manual local | `CA-TEST-LOCAL` |
| `TEST-SERVER` | `./scripts/check-server.sh` antes de avisar al usuario | `CA-TEST-SERVER` |
| `TEST-KILLALL` | `killall` para detener el server | `CA-TEST-KILLALL` |
| `TEST-TESTUSER` | `./scripts/create-test-user.sh` para probar endpoints autenticados | `CA-TEST-TESTUSER` |
| `TEST-USER-FLOW` | Pruebas manuales del usuario antes de `pending_release` | `CA-TEST-USER-FLOW` |
| `TEST-CHECKLIST` | Checklist completo de `60-testing.md` al cerrar | `CA-TEST-CHECKLIST` |
| `DELIV-WORKTREE` | Trabajo en `../p40la-ihost-spec-140`, rama `feature/SPEC-140` | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | Merge y push solo con confirmación explícita | `CA-DELIV-CONFIRM` |
| `DELIV-COMMIT` | Commits `SPEC-140: descripción` | `CA-DELIV-COMMIT` |
| `DELIV-RELEASE` | Release con merge, tracker, label y limpieza de worktree | `CA-DELIV-RELEASE` |
| `DELIV-HOOKS` | Hooks activos en el worktree | `CA-DELIV-HOOKS` |
| `DELIV-LABELS` | Issue con `spec/draft` y transiciones a `spec/*` válidos | `CA-DELIV-LABELS` |
| `DELIV-DOCKER` | Si cambia la imagen, buildx multi-arch con las 3 plataformas | `CA-DELIV-DOCKER` |
| `DELIV-DEPLOY` | N/A: el deploy lo aprueba el usuario al liberar | — |
| `DELIV-ESCALATION` | N/A: no hay bloqueo al crear la spec | — |

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001 — Página de la casa.** Cada card de `HomesPage` navega a `/home/:id`. La página muestra nombre, dirección y estado de promoción en modo lectura, con pestañas `Resumen` y `Alquiler`.
2. **REQ-002 — Estado de promoción.** La casa expone uno de: `available` (disponible), `reserved` (apartada), `rented` (alquilada), `unlisted` (no publicada). Se calcula desde contratos y apartados vigentes, nunca se edita a mano. Un `Toggle` «Publicar vivienda» controla `unlisted`.
3. **REQ-003 — Apartado (reserva).** Se puede registrar un apartado sobre una casa libre con: interesado, agente inmobiliario opcional, monto y moneda de la seña, referencia de pago, fecha/hora de vencimiento, fecha de firma del contrato, fecha de mudanza, desglose de renta y mantenimiento, nota y política de devolución. Mientras está activo, la casa queda `reserved`, no se promociona y no admite otro apartado ni contrato superpuesto.

**REQ-003.1 — Acuerdo de reserva (basado en documento real aportado por el usuario).** El apartado debe soportar: referencia bancaria de la seña, `applied_to_deposit` (la seña se imputa al depósito de garantía), fecha de firma y entrega de llaves, fecha de inicio de contrato, plazo pactado, desglose de renta + cuota de mantenimiento/vigilancia, reparto de la compensación por desistimiento entre propietario y agencia (`owner_split_percent` + `agency_split_percent`), política de devolución (`refundable`, `forfeitable`, `custom`) y lista de documentos exigidos al firmar (contrato de trabajo, estados de cuenta, identificación, etc.). El PDF del acuerdo se genera desde estos datos, sin texto fijo inventado.
4. **REQ-004 — Ciclo del apartado.** Estados: `active → converted | expired | cancelled | forfeited`. El scheduler marca `expired` al vencer y libera la casa. La conversión crea un contrato en `draft` vinculado, imputando la seña al depósito si el usuario lo indica.
5. **REQ-005 — Contrato de alquiler.** CRUD de contratos con: casa (fija cuando el formulario se abre desde `?home=ID` o en edición; selector solo si se crea sin contexto), inmobiliaria opcional, agente opcional, fecha de inicio, plazo en meses (`6`, `12` o `N` de 1 a 120), fecha de fin calculada, renta base, cuota mensual de mantenimiento/vigilancia, moneda, día de pago, depósito de garantía, mora opcional, renovación automática, días de preaviso, cuenta bancaria para el cobro y notas.
5.1. **REQ-005.1 — Formulario en tabs.** El alta y la edición del contrato se organizan en tres tabs persistidas en la URL: `?tab=general` (vivienda, inmobiliaria, agente y comisión), `?tab=terms` (condiciones económicas, mora, renovación, cobro y notas) y `?tab=parties` (personas). Guardar y Cancelar quedan siempre visibles; los errores de validación cambian a la tab correspondiente.
5.2. **REQ-005.2 — Edición de contrato activo.** Un contrato activo se puede editar en sus datos seguros: inmobiliaria, agente, comisión, mora, renovación, preaviso, cuenta de cobro, notas y personas. Las condiciones que ya generaron cargos o facturas (casa, fecha de inicio, plazo, renta, mantenimiento, moneda, día de pago y depósito) se muestran bloqueadas y el backend las preserva aunque lleguen en el payload. Terminar o reactivar sigue siendo una acción explícita.
6. **REQ-006 — Partes del contrato.** Personas asociadas con rol `tenant`, `guarantor`, `owner`, `agency_contact` o `agent`, con nombre, email, teléfono, documento, estado civil, domicilio, ocupación, empleador e ingreso mensual declarado (campos del perfil, opcionales). Cada parte define si recibe notificaciones (`notify_email`). Un contrato debe tener al menos un `tenant` con email válido para poder activarse. El formulario permite crear un agente en un modal y seleccionarlo sin salir del contrato.
7. **REQ-007 — Inmobiliaria como institución.** Las inmobiliarias se reutilizan desde `institutions` con la categoría `real_estate` («Inmobiliaria»), creada de forma idempotente por la migración `0055`. El selector del contrato **solo lista** instituciones de esa categoría y el backend **rechaza** cualquier otra con `422 agency_category`. El formulario permite crear una inmobiliaria en un modal con la categoría fija y solo el nombre, sin salir del contrato. **Sin inmobiliaria seleccionada no se muestran agente ni comisión**, y el backend los limpia si llegan en el payload. El contrato guarda `agency_institution_id`, el tipo de comisión (`percent` o `fixed`), el valor correspondiente (`commission_percent` o `commission_amount`) y la base (`collected` o `invoiced`, solo aplicable al porcentaje) para dejar registrada la intermediación. El modo fijo cubre casos como «un mes de alquiler» o un depósito pactado.
8. **REQ-008 — Cargos mensuales.** Al activar un contrato se generan los cargos de renta de cada período del plazo, de forma idempotente (único por contrato + período + tipo). Tipos: `rent`, `deposit`, `late_fee`, `utility`, `other`.
9. **REQ-009 — Factura interna de cobro.** Se emite una factura interna agrupando cargos pendientes del contrato. Tiene serie y número secuencial **inmutable**, fecha de emisión, vencimiento, líneas, total y moneda. Un documento emitido no se edita ni se borra. Con la facturación automática activa, el scheduler emite **una factura por período** cuando su vencimiento llega (mismo día) y, si el envío automático está activo y el SMTP configurado, la envía a inquilino y garantes con deduplicación diaria.
10. **REQ-010 — PDF de factura.** La factura se genera como PDF A4 con datos del emisor configurables, referencia al contrato y la vivienda, parte cobrada, detalle de líneas, total y leyenda «Documento interno de cobro — no constituye factura fiscal». Se guarda con hash SHA-256.
11. **REQ-011 — Envío por email.** La factura se puede enviar a las partes con `notify_email` usando el SMTP existente, adjuntando el PDF. Se registra el envío (fecha, destinatarios por rol, resultado) sin volcar direcciones a los logs.
12. **REQ-012 — Pagos y recibos.** Se registran pagos totales o parciales contra una factura o contra el contrato, con método, fecha y referencia. Al registrar un pago se puede emitir un recibo PDF con numeración propia. El saldo se recalcula; nunca se edita un recibo emitido.
13. **REQ-013 — Comprobantes de pago.** Cada pago admite adjuntos de comprobante (`pdf`, `png`, `jpg`), hasta 10 MB por archivo y 10 por pago, con hash y metadatos.
14. **REQ-014 — Documentos legales del contrato.** El contrato admite adjuntos PDF por tipo: `contract`, `addendum`, `inventory`, `id`, `income_proof`, `insurance`, `other`. Límite 10 MB por archivo y 20 por contrato.
15. **REQ-015 — Recordatorios.** Se pueden crear recordatorios por casa o por contrato con tipo `gardening`, `maintenance`, `inspection`, `payment_reminder` o `custom`, frecuencia (`once`, `monthly`, `every_n_days`, `yearly`), próxima fecha, días de anticipación y destinatarios. El scheduler envía email al cumplirse la anticipación.
16. **REQ-016 — Automatización diaria.** Un scheduler interno (patrón `Start/Stop`, ticker y deduplicación) ejecuta: vencimiento de apartados, **facturación automática de períodos vencidos**, envío automático de esas facturas y recordatorios. No depende de cron externo. Es idempotente: un período ya facturado no se vuelve a emitir y un email ya enviado no se repite.
16.1. **REQ-016.1 — Imputación automática de pagos.** Si un pago no indica factura, se imputa a la factura emitida impaga más antigua. Si no existe, el sistema emite una factura con los cargos vencidos del período más antiguo y la imputa. Si no hay cargos vencidos, el pago solo reduce el saldo del contrato.
17. **REQ-017 — Configuración de alquiler.** Sección de settings con datos del emisor (nombre, identificación, dirección, email), serie de facturas, serie de recibos, moneda por defecto, días de anticipación por defecto, remitente de recordatorios y toggles de **facturación automática** y **envío automático de facturas** (ambos activos por defecto).
18. **REQ-018 — Auditoría mínima.** Cada emisión, envío, anulación y pago queda registrado con fecha y usuario. Las anulaciones conservan el número y marcan el PDF como «ANULADA».
19. **REQ-019 — Aislamiento de datos.** Todas las tablas son nuevas y propias del módulo; no se altera el esquema de `debts`/`debt_bills`. La única modificación a una tabla existente es agregar `listing_enabled` a `homes` con default seguro.
20. **REQ-020 — Multimoneda.** Contrato, cargos, facturas, pagos y recibos guardan moneda ISO-4217 y formatean según la configuración existente. No se convierte entre monedas automáticamente.

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-021 — Listado global de alquileres.** Página `/leases` con contratos activos, por vencer y terminados, filtrable por casa.
2. **REQ-022 — Renovación y vencimiento.** Badge `expiring` cuando faltan ≤ 60 días; acción «Renovar» que crea un nuevo contrato encadenado sin solapar fechas.
3. **REQ-023 — Mora automática.** Si el contrato define mora y el cargo sigue impago tras la fecha de vencimiento, se agrega un cargo `late_fee` configurable (monto fijo o porcentaje, con tope).
4. **REQ-024 — Estado de cuenta del inquilino.** PDF con cargos, pagos y saldo por contrato y período.
5. **REQ-025 — Estado de cuenta del propietario.** Ingresos cobrados, comisión de la inmobiliaria y neto, por período.
6. **REQ-026 — Recordatorio de cobro.** Recordatorio automático N días antes del vencimiento de cada factura, sin duplicar envíos.

### 3.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-027 — Nota de crédito.** Corrección formal de una factura anulada con documento que referencia el original.
2. **REQ-028 — Canales alternativos.** Enviar recordatorios también por Telegram o WhatsApp usando la infraestructura de alertas existente.
3. **REQ-029 — Firma electrónica** de contratos y adjuntos.
4. **REQ-030 — Portal del inquilino** de solo lectura con sus facturas y recibos.
5. **REQ-031 — Indexación anual** de la renta por índice configurable.
6. **REQ-032 — Exportación CSV** de contratos y cobros.

### 3.4 Requerimientos No Funcionales

- **Rendimiento**: generar un PDF en ≤ 1 s y ≤ 30 MB de pico de memoria en ARM64; la pestaña Alquiler responde en ≤ 300 ms con hasta 200 contratos.
- **Concurrencia**: numeración de documentos atómica; dos emisiones simultáneas nunca repiten número. Transacciones SQLite cortas con WAL.
- **Almacenamiento**: PDFs y adjuntos en el volumen (`DATA_DIR/rental/...`), metadatos y hashes en SQLite. Cuota lógica: 20 adjuntos por contrato y 10 por pago.
- **Seguridad**: todos los endpoints autenticados; descargas por ID, sin rutas provistas por el cliente; validación de tipo real por magic bytes.
- **Privacidad**: nombres, emails y direcciones no se escriben en logs ni se commitean; los ejemplos son sintéticos.
- **Disponibilidad**: scheduler idempotente; un fallo de email no bloquea la operación ni duplica envíos.
- **iHost**: sin Chromium, sin Node en runtime, sin cgo; dependencia nueva máxima de una librería PDF liviana.
- **Compatibilidad**: migraciones `up` y `down` probadas; sin pérdida de datos al revertir (solo se eliminan tablas nuevas).

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

Investigación de dominio (subagentes, 2026-11-23) y del código existente:

- **Roles**: propietario (*landlord/owner*), inmobiliaria (*brokerage/property manager*) e inquilino (*tenant*). La inmobiliaria cobra por mandato y rinde cuentas; las comisiones típicas son 8–12 % de la renta cobrada, con variantes por colocación y renovación. Fuente: [iPropertyManagement](https://ipropertymanagement.com/research/average-property-management-fees), [NCHM](https://www.nchm.org/how-much-does-a-property-manager-cost).
- **Apartado**: el *holding deposit* retira la unidad del mercado por una ventana corta (típicamente 15 días, con topes como una semana de renta en UK). Se documenta por escrito con monto, partes, vencimiento, condiciones y reembolsabilidad. Es distinto del **depósito de garantía** (*security deposit*), que existe desde la firma y dura todo el contrato. Fuente: [Nolo](https://www.nolo.com/legal-encyclopedia/how-handle-tenant-holding-deposits.html), [Shelter](https://england.shelter.org.uk/housing_advice/tenancy_deposits/holding_deposits).
- **Contratos**: plazo fijo (6/12 meses), periódico (*month-to-month*) y renovación. Al vencer suele convertirse en periódico. Documentos habituales: contrato firmado, identificación, comprobante de ingresos, inventario/*condition report*, certificados y pólizas. Fuente: [Consumer Affairs Victoria](https://www.consumer.vic.gov.au/housing/renting/starting-and-changing-rental-agreements/different-rental-agreements/residential-rental-agreements).
- **Cobros**: *invoice* es el cargo; *receipt* es la prueba de pago. No son intercambiables; un pago parcial emite recibo por lo pagado y deja saldo. Los comprobantes de transferencia se concilian aparte.
- **Numeración e inmutabilidad**: numeración única y secuencial por serie, sin reutilizar números; un documento emitido no se edita, se corrige con nota de crédito. Retención de 6–10 años según jurisdicción. Fuente: [HMRC VATREC5010](https://www.gov.uk/hmrc-internal-manuals/vat-trader-records/vatrec5010).
- **Fiscalidad**: quién es emisor y qué documento tiene validez fiscal **varía por país**. Por eso esta spec define documentos internos de cobro y deja los datos fiscales como configuración.
- **Código existente**: `homes` solo tiene `name`/`address` y no tiene página de detalle; `internal/api/contract_handlers.go` es carga de PDF de contratos de deudas, no una entidad contractual; no existe generación de PDF (solo lectura con `ledongthuc/pdf`); el envío de email no soporta adjuntos; los schedulers siguen el patrón `Start/Stop` + ticker + dedup diaria; los módulos namespaced viven en `internal/modules/` con registro en `all/register.go`; la última migración es `0050`.
- **Documento de referencia del usuario**: se analizó un acuerdo de reserva escaneado (3 páginas) aportado como ejemplo. Define partes, propiedad, renta base + cuota de mantenimiento, monto y referencia de la seña, imputación al depósito, fecha de firma/entrega de llaves, fecha de inicio, plazo de 1 año, documentos exigidos al inquilino, reparto de compensación propietario/agencia y credencial del agente inmobiliario. **No se copiaron datos personales ni importes**; solo se derivaron campos y flujos.

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| Módulo único `realestate.rental` con archivos internos separados | Coherencia transaccional, sin dependencias cruzadas entre módulos, menos boilerplate en iHost | Un módulo grande | ✅ Seleccionada |
| Tres módulos (`rental.lease`, `rental.billing`, `rental.reminders`) | Separación por capacidad | `ARCH-MODULES` prohíbe que un módulo importe storage de otro; obliga a duplicar o a contratos artificiales | ❌ Rechazada |
| Entidad «agencia» nueva | Modelo explícito | Duplica `institutions`, que ya tiene categorías y contactos | ❌ Rechazada |
| Inmobiliaria = institución con categoría `real_estate` | Reutiliza CRUD, búsqueda y contacto existentes; menos tablas | La comisión se guarda en el contrato, no en la institución | ✅ Seleccionada |
| PDF con navegador headless (HTML→PDF) | Fidelidad visual | Chromium en iHost es inviable (RAM, tamaño de imagen) y viola `ARCH-IHOST` | ❌ Rechazada |
| PDF con `go-pdf/fpdf` (Go puro) | Sin cgo, ARM64, memoria baja, fuentes core y TTF embebidas | Maquetación manual, sin CSS | ✅ Seleccionada |
| PDF con `maroto` | API de alto nivel | Más dependencias y peso; menos control de memoria | ❌ Rechazada |
| Estado de promoción almacenado | Consulta simple | Se desincroniza si falla el scheduler o se edita a mano | ❌ Rechazada |
| Estado de promoción derivado | Única fuente de verdad; nunca miente | Requiere joins en la consulta | ✅ Seleccionada |
| Un solo `deposit_amount` para seña y garantía | Menos tablas | Mezcla dos conceptos con ciclos de vida distintos | ❌ Rechazada |
| Entidades separadas `reservation` y `deposit` | Ciclos y estados correctos | Más tablas | ✅ Seleccionada |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001 — Módulo único `realestate.rental`.**
- **Contexto**: el flujo contrato → cargos → factura → pago → recibo es una sola transacción de negocio y `ARCH-MODULES` impide dependencias de storage entre módulos.
- **Decisión**: un módulo con ID `realestate.rental`, ruta `internal/modules/realestate/rental/`, y archivos internos adicionales (`billing.go`, `pdf.go`, `reminders.go`) además de los obligatorios. Todo el SQL en `storage*.go` del mismo paquete.
- **Consecuencias**: transacciones simples y consistentes; el paquete crece y debe mantenerse con archivos temáticos.

**ADR-002 — Estado de promoción derivado.**
- **Contexto**: «no promocionar más la vivienda» debe ser verdad incluso si un proceso falla.
- **Decisión**: `listing_status` se calcula en el servicio a partir de `homes.listing_enabled`, apartados `active` no vencidos y contratos vigentes. No se persiste.
- **Consecuencias**: nunca queda una casa bloqueada por estado obsoleto; la consulta hace 2 joins indexados.

**ADR-003 — Documentos internos, no fiscales.**
- **Contexto**: la validez fiscal depende del país y no está definida por el usuario.
- **Decisión**: factura interna de cobro con serie/número propio, leyenda explícita y datos del emisor configurables. No se declara «factura fiscal».
- **Consecuencias**: cubre el cobro y la trazabilidad; si más adelante se necesita fiscalidad, se agrega un perfil por país sin migrar los documentos existentes.

**ADR-004 — Numeración atómica con tabla de secuencias.**
- **Contexto**: la numeración no puede repetirse ni reutilizarse.
- **Decisión**: `rental_sequences(kind, series, year, next_number)`; la emisión reserva el número dentro de la misma transacción que crea el documento (`UPDATE ... ; SELECT`), con `UNIQUE(kind, series, year, number)` como red de seguridad.
- **Consecuencias**: puede haber huecos si una transacción falla; es aceptable y preferible a repetir.

**ADR-005 — Inmutabilidad por estado, no por borrado.**
- **Contexto**: un documento emitido no se edita.
- **Decisión**: `issued` no admite `PUT`/`DELETE`; la anulación cambia a `void`, guarda motivo y regenera el PDF con marca «ANULADA». El número se conserva.
- **Consecuencias**: auditoría clara; el almacenamiento crece con cada versión anulada.

**ADR-006 — Importes en REAL con redondeo a 2 decimales.**
- **Contexto**: el resto del proyecto (facturas, deudas, tarjetas) usa `REAL` y `round2`. Introducir centavos enteros solo en este módulo duplicaría conversiones en API, PDF y frontend.
- **Decisión**: `REAL` con `round2` en cada operación y comparación de saldos con tolerancia de 0.005. La numeración y los totales se calculan en el servicio, nunca en el frontend.
- **Consecuencias**: consistencia con el código existente; los tests cubren redondeo y saldos.

**ADR-007 — El acuerdo de reserva se modela con campos, no con texto fijo.**
- **Contexto**: el documento real aportado por el usuario mezcla cláusulas de retención y devolución y agrega datos que ungen el flujo (referencia de pago, reparto de compensación, documentos exigidos).
- **Decisión**: todos esos datos son columnas del apartado y el PDF se arma desde ellos. La política de devolución es un enum explícito (`refundable`, `forfeitable`, `custom`) para no inventar una regla legal.
- **Consecuencias**: el documento es configurable por operación; no se hardcodea una jurisdicción. Los datos del ejemplo aportado no se copian al repo.

---

## 5. Diseño Técnico

### 5.1 Diagrama de arquitectura

```
[React: HomeShowPage / LeaseShowPage / ReservationPage / InvoiceShowPage]
                 │  (fetch JSON + PDF)
                 v
[internal/api/routes.go] ──> [módulo realestate.rental]
                                 handlers → service → storage → model
                                 │            │
                                 │            ├─> services.EmailService (SMTP)
                                 │            ├─> services.SystemSettings (emisor/moneda)
                                 │            └─> pdf (Go puro)
                                 v
                          [SQLite WAL]  +  [DATA_DIR/rental/... (PDFs y adjuntos)]
                                 ^
                                 │
                    [RentalScheduler: apartados, recordatorios, mora]
```

### 5.2 Componentes

#### 5.2.1 Módulo `realestate.rental`
- **Responsabilidad**: contratos, apartados, partes, cargos, facturas, recibos, pagos, documentos y recordatorios.
- **Interfaz**: `Info()` con `ID: "realestate.rental"`, versión `1.0.0`, `Kind: "core"`, `Migrations: ["0051_rental_leases", "0052_rental_reservations", "0053_rental_billing", "0054_rental_reminders", "0055_real_estate_category", "0056_rental_commission_type"]`; `RegisterRoutes(mux, auth)`.
- **Dependencias**: `*sql.DB`, `*services.EmailService`, `*services.SystemSettings`, `DATA_DIR`, reloj inyectable.
- **Ubicación**: `internal/modules/realestate/rental/`.
- **Archivos**: `module.go`, `model.go`, `storage_lease.go`, `storage_billing.go`, `storage_reminders.go`, `service.go`, `billing.go`, `pdf.go`, `reminders.go`, `handlers.go`, `*_test.go`.
- **Registro**: import en `internal/modules/all/register.go`; sin cambios de lógica en `internal/api/routes.go`.

#### 5.2.2 PDF (`pdf.go`)
- **Responsabilidad**: renderizar factura, recibo y estado de cuenta.
- **Interfaz**: `RenderInvoice(inv InvoiceSnapshot) ([]byte, error)`, `RenderReceipt(...)`, `RenderStatement(...)`.
- **Dependencias**: `github.com/go-pdf/fpdf` (MIT, Go puro, sin cgo).
- **Detalles**: A4, fuente DejaVuSans embebida con `//go:embed` para acentos y símbolos; salida a `[]byte`; escritura atómica (`*.tmp` + `rename`); semáforo de 2 generaciones concurrentes.
- **Ubicación**: `internal/modules/realestate/rental/pdf.go` y `assets/`.

#### 5.2.3 Envío de email con adjuntos
- **Responsabilidad**: extender `EmailService` con `SendWithAttachments(ctx, to, subject, html, []Attachment)`, donde `Attachment{Name, ContentType, Data}`.
- **Interfaz**: `multipart/mixed` con `multipart/alternative` interno; base64; límite total 10 MB; solo `application/pdf` en esta spec.
- **Dependencias**: `net/smtp` existente; sin dependencias nuevas.
- **Ubicación**: `internal/services/email_service.go` (método nuevo, sin romper `Send`).

#### 5.2.4 `RentalScheduler`
- **Responsabilidad**: cada 15 min (configurable) ejecuta tareas idempotentes: vencer apartados, enviar recordatorios cuya anticipación se cumplió, marcar cargos vencidos y (P1) aplicar mora.
- **Interfaz**: `NewRentalScheduler(service, logger, interval)`, `Start()`, `Stop()`, `RunOnce(ctx)`.
- **Dependencias**: servicio del módulo; tabla de eventos para dedup.
- **Ubicación**: `internal/modules/realestate/rental/scheduler.go`; arranque en `cmd/server/main.go`.
- **Idempotencia**: `rental_reminder_events UNIQUE(reminder_id, occurrence_key)` y `rental_email_log UNIQUE(kind, owner_id, occurrence_key)`.

#### 5.2.5 Frontend
- **Responsabilidad**: página de casa con pestaña Alquiler, formularios y detalles.
- **Componentes**: `HomeShowPage`, `LeaseFormPage`, `LeaseShowPage`, `ReservationFormPage`, `ReservationShowPage`, `InvoiceShowPage`, `RentalSettingsPage`, más `LeaseCard`, `PartyEditor`, `AttachmentList`, `PaymentModal`, `SendInvoiceModal`.
- **Ubicación**: `frontend/src/pages/` y `frontend/src/components/rental/`.

### 5.3 Modelo de datos

> Todas las tablas nuevas. Importes en `INTEGER` de unidades mínimas (centavos) para evitar errores de coma flotante. Fechas en `TEXT` RFC 3339 UTC; fechas de calendario en `TEXT YYYY-MM-DD`.

```
home (existente)
- + listing_enabled: INTEGER NOT NULL DEFAULT 1

rental_people
- id, full_name, email, phone, tax_id
- address, occupation, employer, monthly_income, marital_status
- notes, deleted_at, created_at, updated_at

rental_leases
- id, home_id (FK homes), agency_institution_id (FK institutions, nullable), agent_person_id (nullable)
- status: draft|active|ended|terminated|cancelled
- start_date, end_date, term_months (1..120)
- rent_amount, maintenance_amount, currency, payment_day (1..28)
- deposit_amount, late_fee_type (none|fixed|percent), late_fee_value, late_fee_cap
- auto_renew, notice_days, commission_type (percent|fixed), commission_percent, commission_amount, commission_base (collected|invoiced)
- owner_bank_account, notes, created_at, updated_at, deleted_at
- CHECK: end_date > start_date; term_months between 1 and 120

rental_lease_parties
- id, lease_id, person_id, role: tenant|guarantor|owner|agency_contact|agent
- notify_email INTEGER NOT NULL DEFAULT 1
- UNIQUE(lease_id, person_id, role)

rental_reservations
- id, home_id, applicant_person_id, agent_person_id (nullable), converted_lease_id (nullable)
- status: active|converted|expired|cancelled|forfeited
- amount, currency, reference, applied_to_deposit
- starts_at, expires_at, contract_date, move_in_date
- rent_amount, maintenance_amount
- owner_split_percent, agency_split_percent
- refund_policy: refundable|forfeitable|custom
- required_documents (TEXT), notes
- created_at, updated_at
- Índice único parcial: una sola reserva active por home
- CHECK: expires_at > starts_at

rental_charges
- id, lease_id, kind: rent|maintenance|deposit|late_fee|utility|other
- period: TEXT YYYY-MM, description, amount, currency
- due_date, status: pending|invoiced|partially_paid|paid|void
- invoice_id (nullable), created_at, updated_at
- UNIQUE(lease_id, period, kind)

rental_sequences
- kind: invoice|receipt, series, year, next_number
- PRIMARY KEY(kind, series, year)

rental_invoices
- id, lease_id, series, number, year
- status: draft|issued|void
- issue_date, due_date, currency
- subtotal_minor, total_minor
- snapshot_json (emisor, receptor, vivienda, contrato congelados)
- pdf_path, pdf_sha256, pdf_size_bytes
- sent_at, void_reason, voided_at
- created_at, UNIQUE(series, year, number)

rental_invoice_lines
- id, invoice_id, charge_id (nullable), description, amount_minor

rental_payments
- id, lease_id, invoice_id (nullable), paid_at
- amount_minor, currency, method: cash|transfer|check|card|other
- reference, notes, created_at

rental_payment_proofs
- id, payment_id, original_name, content_type (pdf|png|jpeg)
- size_bytes, sha256, storage_path, created_at

rental_receipts
- id, payment_id UNIQUE, series, number, year
- status: issued|void, issue_date, amount_minor, currency
- pdf_path, pdf_sha256, pdf_size_bytes, sent_at, created_at

rental_lease_documents
- id, lease_id, doc_type: contract|addendum|inventory|id|income_proof|insurance|other
- original_name, content_type (pdf), size_bytes, sha256, storage_path, created_at

rental_reminders
- id, home_id (nullable), lease_id (nullable)
- kind: gardening|maintenance|inspection|payment_reminder|custom
- title, description
- schedule_type: once|monthly|every_n_days|yearly
- interval_n, anchor_date, next_due_at, lead_days
- notify_roles (CSV: tenant,guarantor,owner,agency_contact)
- extra_emails (CSV, opcional)
- enabled INTEGER NOT NULL DEFAULT 1
- created_at, updated_at
- CHECK: al menos uno de home_id o lease_id

rental_reminder_events
- id, reminder_id, occurrence_key (TEXT), due_at, sent_at, result (sent|failed|skipped), error_code
- UNIQUE(reminder_id, occurrence_key)

rental_email_log
- id, kind (invoice|receipt|reminder|statement), owner_id, occurrence_key
- recipients_count, recipient_roles, result, error_code, created_at
- UNIQUE(kind, owner_id, occurrence_key)
```

**Relaciones**: `home 1—N lease`; `home 1—N reservation`; `lease 1—N party`, `1—N charge`, `1—N invoice`, `1—N payment`, `1—N document`, `1—N reminder`; `invoice 1—N line`, `1—N payment`; `payment 1—1 receipt`, `1—N proof`.

**Estado de promoción** (derivado):
```
unlisted  si homes.listing_enabled = 0
rented    si existe lease active con start_date <= hoy <= end_date
reserved  si existe reservation active con expires_at > ahora
available en cualquier otro caso
```

### 5.4 APIs / Contratos

Todas bajo `/api/rental`, autenticadas con el middleware existente. Errores con el formato estándar `{"error":"codigo","message":"..."}`.

#### Vivienda y apartados

| Método | Ruta | Descripción |
|---|---|---|
| `GET` | `/api/rental/homes/{homeID}` | Estado de promoción, contrato vigente, apartado vigente y próximos vencimientos |
| `PUT` | `/api/rental/homes/{homeID}/listing` | `{"listing_enabled": false}` |
| `GET` | `/api/rental/reservations?home_id=&status=` | Listado |
| `POST` | `/api/rental/reservations` | Crear apartado |
| `GET` | `/api/rental/reservations/{id}` | Detalle |
| `PUT` | `/api/rental/reservations/{id}` | Editar mientras `active` |
| `POST` | `/api/rental/reservations/{id}/convert` | Crear contrato `draft` y marcar `converted` |
| `POST` | `/api/rental/reservations/{id}/cancel` | `{"reason":"..."}` |
| `POST` | `/api/rental/reservations/{id}/forfeit` | Retener la seña con motivo |

**Request — crear apartado**
```json
{
  "home_id": 1,
  "applicant": {"full_name": "Inquilino Demo", "email": "demo@example.com", "phone": "+1 555 0100"},
  "amount_minor": 150000,
  "currency": "USD",
  "expires_at": "2026-12-07T23:59:59Z",
  "notes": "Reserva sintética de ejemplo"
}
```
**Response 201**
```json
{"id": 10, "status": "active", "expires_at": "2026-12-07T23:59:59Z", "home_id": 1}
```
**Error 409**: `{"error":"home_not_available","message":"La vivienda ya tiene un apartado o contrato vigente"}`.

#### Contratos

| Método | Ruta | Descripción |
|---|---|---|
| `GET` | `/api/rental/leases?status=&home_id=` | Listado |
| `POST` | `/api/rental/leases` | Crear `draft` |
| `GET` | `/api/rental/leases/{id}` | Detalle con partes, cargos y saldo |
| `PUT` | `/api/rental/leases/{id}` | Editar solo en `draft` (fechas/monto bloqueados si está `active`, salvo notas) |
| `POST` | `/api/rental/leases/{id}/activate` | Valida casa libre, partes y email; genera cargos |
| `POST` | `/api/rental/leases/{id}/terminate` | `{"reason":"...","end_date":"..."}` |
| `DELETE` | `/api/rental/leases/{id}` | Solo `draft`; soft-delete |
| `GET/POST` | `/api/rental/leases/{id}/parties` | Listar/agregar parte |
| `DELETE` | `/api/rental/leases/{id}/parties/{partyID}` | Quitar parte |
| `GET/POST` | `/api/rental/leases/{id}/documents` | Listar/subir (multipart `file`, `doc_type`) |
| `GET` | `/api/rental/leases/{id}/documents/{docID}` | Descargar (auth; `?download=1`) |
| `DELETE` | `/api/rental/leases/{id}/documents/{docID}` | Borrar adjunto |
| `GET/POST` | `/api/rental/leases/{id}/charges` | Listar/generar cargos (`{"periods":["2026-12"]}`) |
| `GET/POST` | `/api/rental/leases/{id}/reminders` | Listar/crear recordatorio |

**Request — crear contrato**
```json
{
  "home_id": 1,
  "agency_institution_id": 3,
  "start_date": "2026-12-01",
  "term_months": 12,
  "rent_amount_minor": 50000,
  "currency": "USD",
  "payment_day": 5,
  "deposit_amount_minor": 100000,
  "auto_renew": false,
  "notice_days": 30,
  "commission_type": "percent",
  "commission_percent": 8.5,
  "commission_amount": 0,
  "commission_base": "collected",
  "parties": [
    {"person": {"full_name": "Inquilino Demo", "email": "demo@example.com"}, "role": "tenant", "notify_email": true},
    {"person": {"full_name": "Garante Demo", "email": "garante@example.com"}, "role": "guarantor", "notify_email": true}
  ]
}
```

#### Cobros

| Método | Ruta | Descripción |
|---|---|---|
| `GET` | `/api/rental/invoices?lease_id=&status=` | Listado |
| `POST` | `/api/rental/invoices` | `{"lease_id":1,"charge_ids":[1,2],"due_date":"2026-12-05"}` → emite número y PDF |
| `GET` | `/api/rental/invoices/{id}` | Detalle |
| `GET` | `/api/rental/invoices/{id}/pdf` | PDF (inline o descarga) |
| `POST` | `/api/rental/invoices/{id}/send` | Envía a partes notificables; adjunta PDF |
| `POST` | `/api/rental/invoices/{id}/void` | `{"reason":"..."}`; regenera PDF «ANULADA» |
| `GET/POST` | `/api/rental/invoices/{id}/payments` | Listar/registrar pago |
| `GET` | `/api/rental/payments/{id}/receipt.pdf` | Recibo emitido |
| `POST` | `/api/rental/payments/{id}/proofs` | Subir comprobante (multipart) |
| `GET` | `/api/rental/payments/{id}/proofs/{proofID}` | Descargar comprobante |
| `GET` | `/api/rental/leases/{id}/balance` | Cargos, pagos y saldo |

**Request — emitir factura**
```json
{"lease_id": 1, "charge_ids": [1, 2], "due_date": "2026-12-05"}
```
**Response 201**
```json
{
  "id": 7, "series": "ALQ", "number": 1, "year": 2026,
  "status": "issued", "total_minor": 50000, "currency": "USD",
  "pdf_sha256": "0000000000000000000000000000000000000000000000000000000000000000"
}
```

#### Personas, settings y scheduler

| Método | Ruta | Descripción |
|---|---|---|
| `GET` | `/api/rental/people?q=` | Búsqueda para pickers |
| `POST` | `/api/rental/people` | Alta rápida |
| `GET/PUT` | `/api/rental/settings` | Emisor, series, moneda, anticipación |
| `POST` | `/api/rental/reminders/{id}/run` | Enviar ahora (manual) |

### 5.5 Dependencias

- **Internas**: `internal/modules/registry.go` (sin cambios), `internal/modules/all/register.go` (nuevo import), `internal/services/email_service.go` (método de adjuntos), `internal/services/system_settings.go` (settings nuevos), `cmd/server/main.go` (scheduler), `frontend/src/App.tsx`, `DashboardLayout.tsx`, `api/index.ts`, `types/index.ts`, i18n.
- **Externas**: `github.com/go-pdf/fpdf` v0.9.0 (MIT, Go puro, sin cgo). Justificación: única forma de cumplir PDF nativo en iHost sin Chromium; impacto ~2 MB en el binario y sin procesos externos. Ya incorporada en `go.mod`.

---

## 6. Criterios de Aceptación

### 6.1 Funcionales

- [ ] CA-RENTAL-HOME-PAGE: desde `/home` se abre `/home/:id`; la pestaña Alquiler muestra estado, contrato, apartado y próximos vencimientos.
- [ ] CA-RENTAL-LISTING: con apartado activo o contrato vigente, `listing_status` es `reserved`/`rented` y el formulario de nuevo apartado responde 409.
- [ ] CA-RENTAL-RESERVATION: crear un apartado con vencimiento futuro lo deja `active`; al pasar el vencimiento el scheduler lo marca `expired` y la casa vuelve a `available`.
- [ ] CA-RENTAL-RESERVATION-CONVERT: convertir un apartado crea un contrato `draft` con la misma casa y partes, y marca el apartado `converted`.
- [ ] CA-RENTAL-LEASE-TERM: un contrato de 6 meses desde `2026-12-01` termina `2027-05-31`; uno de 12 meses termina `2027-11-30`; un plazo personalizado de 18 meses calcula el fin correcto.
- [ ] CA-RENTAL-LEASE-ACTIVATE: activar sin `tenant` con email devuelve 422 con error junto al campo; activar correctamente genera un cargo `rent` por período y es idempotente al reintentar.
- [ ] CA-RENTAL-LEASE-EDIT: un contrato activo muestra la acción Editar; se guardan inmobiliaria, agente, comisión, notas y personas, mientras renta, plazo, fechas, moneda, día de pago y depósito permanecen intactos aunque el payload los cambie.
- [ ] CA-RENTAL-AGENCY: la inmobiliaria se elige con `Select` buscable **solo** entre instituciones de categoría `real_estate`; una institución de otra categoría no aparece en la lista y el backend responde `422 agency_category`. Sin inmobiliaria no se muestran ni persisten agente ni comisión.
- [ ] CA-RENTAL-COMMISSION: la comisión se guarda como porcentaje (0–100 con base cobrado/facturado) o como monto fijo en la moneda del contrato; el tipo inválido se rechaza y el detalle muestra el valor correcto.
- [ ] CA-RENTAL-AUTO-INVOICE: con la facturación automática activa, al llegar el vencimiento del período se emite una única factura con sus cargos y no se duplica al reejecutar el scheduler. Si el envío automático está activo y hay SMTP, se envía una sola vez por día.
- [ ] CA-RENTAL-PAYMENT-MATCH: un pago sin factura se imputa a la impaga más antigua; si no existe, genera la factura del período vencido más antiguo; los cargos quedan pagados o parciales según el monto.
- [ ] CA-RENTAL-INVOICE-NUMBER: dos emisiones concurrentes producen números distintos y consecutivos; el número no se reutiliza al anular.
- [ ] CA-RENTAL-INVOICE-IMMUTABLE: una factura `issued` no admite edición ni borrado (405/409); anular exige motivo y conserva el número.
- [ ] CA-RENTAL-PDF: el PDF de factura se genera en ≤ 1 s, contiene emisor, número, vivienda, parte, líneas, total y la leyenda de documento interno; el hash coincide con el archivo.
- [ ] CA-RENTAL-EMAIL: enviar una factura adjunta el PDF a las partes con `notify_email`; un envío repetido del mismo día no duplica por la clave de dedup.
- [ ] CA-RENTAL-PAYMENT: un pago parcial deja saldo y estado `partially_paid`; el pago total lo deja `paid`; el recibo emitido no se edita.
- [ ] CA-RENTAL-PROOFS: se aceptan comprobantes PDF/PNG/JPEG ≤ 10 MB; se rechazan otros tipos y se detecta por magic bytes.
- [ ] CA-RENTAL-DOCUMENTS: se adjuntan documentos legales por tipo; el 21.º devuelve 422; la descarga requiere sesión.
- [ ] CA-RENTAL-REMINDERS: un recordatorio mensual con `lead_days=3` envía 3 días antes, registra el evento y calcula la ocurrencia siguiente; «Enviar ahora» funciona aunque no esté vencido.
- [ ] CA-RENTAL-SETTINGS: cambiar la serie de facturas afecta solo a documentos nuevos y no altera los ya emitidos.
- [ ] CA-RENTAL-BALANCE: el saldo del contrato es suma de cargos menos pagos, sin negativos silenciosos.

### 6.2 No funcionales / reglas

- [ ] CA-ARCH-STACK / CA-ARCH-FOLDERS / CA-ARCH-LAYERS / CA-ARCH-MODULES / CA-ARCH-GO: estructura y capas verificadas; `go vet` y build limpios.
- [ ] CA-ARCH-IHOST: sin cgo, sin Node runtime, pico de memoria de PDF medido y documentado.
- [ ] CA-UI-SELECT / CA-UI-GUARDRAIL: `npm run check:ui` pasa; no hay `<select>` ni checkbox nativo.
- [ ] CA-UI-INPUT-TOKENS / CA-UI-DARKMODE: formularios y modales legibles en darkmode.
- [ ] CA-UI-TOGGLE / CA-UI-CREATE-MENU / CA-UI-EDIT-LABEL / CA-UI-I18N / CA-UI-LIST-PATTERN / CA-UI-STYLE.
- [ ] CA-UX-BACK / CA-UX-MODAL / CA-UX-MODAL-CONFIRM / CA-UX-DETAIL-ACTIONS / CA-UX-DETAIL-READONLY / CA-UX-PREREQ / CA-UX-EMPTY / CA-UX-LOADING / CA-UX-FEEDBACK / CA-UX-FORMS.
- [ ] CA-DATA-SQLITE / CA-DATA-MIGRATIONS / CA-DATA-NO-DROP / CA-DATA-PRODCOPY / CA-DATA-PROTECTED-SCHEMA.
- [ ] CA-SEC-NO-PII / CA-SEC-SYNTHETIC / CA-SEC-PRODCOPY / CA-SEC-COMMIT / CA-SEC-HOOKS / CA-SEC-LOGS.
- [ ] CA-TEST-LOCAL / CA-TEST-SERVER / CA-TEST-KILLALL / CA-TEST-TESTUSER / CA-TEST-USER-FLOW / CA-TEST-CHECKLIST.
- [ ] CA-DELIV-WORKTREE / CA-DELIV-CONFIRM / CA-DELIV-COMMIT / CA-DELIV-RELEASE / CA-DELIV-HOOKS / CA-DELIV-LABELS / CA-DELIV-DOCKER.

### 6.3 Testing

- **Unit tests**: cálculo de `end_date` para 6/12/18 meses y fin de mes; generación idempotente de cargos; transición y vencimiento de apartados; estado de promoción derivado; totales y redondeo en centavos; saldo; hash de PDF; numeración concurrente (goroutines contra SQLite temporal).
- **Integration tests**: CRUD de contrato con partes; activar y emitir factura; pago parcial y recibo; subida y descarga de adjuntos; envío con SMTP falso; dedup de recordatorios; `409` por casa ocupada; `422` por límites.
- **Migration tests**: `0051`–`0054` `up` y `down` sobre DB temporal; verificar que `homes` conserva filas y que revertir no toca tablas existentes.
- **Frontend**: `npm run check:ui`; navegación y pestañas; formularios con `Select`/`Toggle`; darkmode; estados vacíos y de carga.
- **Manual con el usuario**: casa → apartado → contrato → cargos → factura PDF → email → pago → recibo → recordatorio → vencimiento automático.
- **Performance**: medir tiempo y RSS de generar 20 PDFs seguidos en ARM64; documentar en la spec.

---

## 7. Plan de Implementación

### 7.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|---|---|---|---|
| 1 | Migraciones `0051`–`0054`, modelos y storage | 2 días | Ninguna |
| 2 | Servicio de contratos, partes, apartados y estado de promoción | 3 días | Fase 1 |
| 3 | Cargos, numeración atómica, facturas y recibos | 3 días | Fase 2 |
| 4 | Motor PDF y email con adjuntos | 2 días | Fase 3 |
| 5 | Documentos legales y comprobantes de pago | 1 día | Fase 1 |
| 6 | Recordatorios + `RentalScheduler` | 2 días | Fase 2 |
| 7 | Frontend: HomeShowPage, pestaña Alquiler, contratos, cobros, settings | 5 días | Fases 2–6 |
| 8 | Tests, i18n, darkmode, performance y validación con el usuario | 2 días | Todo |

### 7.2 Milestones

1. **MVP**: casa con pestaña Alquiler, apartado con vencimiento, contrato, cargos, factura PDF, pago y recibo, documentos y recordatorios manuales.
2. **V1.0**: scheduler automático, envío por email, settings del emisor, P1 de listado global y renovación.

---

## 8. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| PII (nombres, emails, direcciones) termina en repo o logs | Media | Alto | Solo DB local; tests con `demo-*`; logs con IDs y conteos; revisión de `git diff --cached` |
| El usuario espera validez fiscal del PDF | Media | Alto | Leyenda explícita, nombre «documento interno de cobro» y settings de emisor; documentar el límite en la UI |
| Librería PDF con problemas en ARM64 | Baja | Alto | `go-pdf/fpdf` es Go puro; prueba de build y ejecución en el iHost antes de `pending_release` |
| Duplicación de emails por reintentos | Media | Medio | `UNIQUE(kind, owner_id, occurrence_key)`; scheduler idempotente |
| Números de factura repetidos por concurrencia | Baja | Alto | Reserva de número en transacción + `UNIQUE` como red de seguridad |
| Crecimiento de disco por adjuntos y PDFs anulados | Media | Medio | Cuotas por entidad, retención configurable y visibilidad del uso |
| Bloqueo de SQLite por transacciones largas | Media | Medio | WAL, transacciones cortas, PDF fuera de la transacción de numeración |
| Ambigüedad de alcance (venta, fiscalidad, firma) | Alta | Medio | P2 explícito; esta spec solo alquiler y documentos internos |
| Fechas y zona horaria mal calculadas | Media | Medio | Fechas de calendario `YYYY-MM-DD`, `expires_at` en UTC, reloj inyectable y tests de borde |
| Casa bloqueada por un apartado olvidado | Media | Bajo | Vencimiento automático, aviso previo y cancelación manual con motivo |

---

## 9. Notas y Referencias

- Reglas aplicadas: `docs/rules/10-architecture.md`, `20-ui.md`, `30-ux.md`, `40-data.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.
- Estructura de módulos: `internal/modules/registry.go`, `internal/modules/all/register.go`, módulo de referencia `internal/modules/bank/creditcard/`.
- Casas: `internal/models/home.go`, `internal/storage/home.go`, `internal/api/home_handlers.go`, `frontend/src/pages/HomesPage.tsx`.
- Adjuntos de referencia: `internal/services/auto_attachment.go`, `internal/api/auto_attachment_handlers.go`.
- Email: `internal/services/email_service.go`, `internal/services/system_settings.go`.
- Schedulers de referencia: `internal/services/alert_scheduler.go`, `automation_scheduler.go`, arranque en `cmd/server/main.go`.
- Investigación de dominio: [Nolo — holding deposits](https://www.nolo.com/legal-encyclopedia/how-handle-tenant-holding-deposits.html), [Shelter — holding deposits](https://england.shelter.org.uk/housing_advice/tenancy_deposits/holding_deposits), [Consumer Affairs Victoria — rental agreements](https://www.consumer.vic.gov.au/housing/renting/starting-and-changing-rental-agreements/different-rental-agreements/residential-rental-agreements), [HMRC VATREC5010 — numeración](https://www.gov.uk/hmrc-internal-manuals/vat-trader-records/vatrec5010), [iPropertyManagement — fees](https://ipropertymanagement.com/research/average-property-management-fees).
- Relacionadas: SPEC-024/025 (patrón de detalle y adjuntos), SPEC-133/134/135 (módulos, detalle y menú de acciones), SPEC-126 (confirmación por palabra).

---

## 10. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-23 | OpenCode | Creación inicial de la especificación (draft) |
| 2026-11-23 | OpenCode | Implementación: migraciones 0051–0054, módulo `realestate.rental`, PDF, email con adjuntos, scheduler y frontend. Corrección: los cargos se generan por `term_months` exactos aunque el contrato empiece a mitad de mes. Estado → `in_progress`. |
| 2026-11-23 | OpenCode | Corrección del selector de inmobiliarias: migración `0055` crea la categoría `real_estate`, el formulario filtra por esa categoría y el backend rechaza instituciones de otro tipo. También se corrigió la verificación de existencia de instituciones (la tabla no tiene `deleted_at`). |
| 2026-11-23 | OpenCode | Alta rápida desde el formulario de contrato: «Agregar inmobiliaria» (categoría fija) y «Agregar agente» en un modal, con selección automática del registro creado. |
| 2026-11-23 | OpenCode | Regla `UI-SELECT-ADD`: toda opción de alta dentro de un dropdown usa «Agregar» + ícono `plus` mediante `addOption()`. Se aplicó a inmobiliaria, agente, persona y categoría de importación. |
| 2026-11-23 | OpenCode | Formularios de contrato y apartado: la vivienda se muestra como dato fijo (sin dropdown) cuando se llega desde `?home=ID` o en edición. |
| 2026-11-23 | OpenCode | Comisión de intermediación con dos modos: porcentaje (con base cobrado/facturado) o monto fijo (por ejemplo, un mes de renta). Migración `0056`, validación y visualización en el detalle. |
| 2026-11-23 | OpenCode | Facturación automática por período al vencer, envío automático con deduplicación diaria y toggles en Configuración → Alquiler. Pagos sin factura se imputan a la impaga más antigua o generan la del período vencido. |
| 2026-11-23 | OpenCode | Guardrail `check:i18n`: valida que toda clave estática usada exista como string en es/en, que no apunte a objetos padre y que ambos idiomas tengan paridad. Se integró al build y al pre-commit; se corrigieron `rental.reference`, `rental.charges.empty*` y claves de presupuesto. |
| 2026-11-23 | OpenCode | Edición de contratos activos: acción Editar en el menú, campos económicos bloqueados en UI y preservados en backend; se editan inmobiliaria, agente, comisión, mora, preaviso, cuenta de cobro, notas y personas. |
| 2026-11-23 | OpenCode | Formulario de contrato en tres tabs (Vivienda, Condiciones, Personas) con la tab activa en `?tab=` y salto automático a la tab con error. |
| 2026-11-23 | OpenCode | Agente y comisión dependen de la inmobiliaria: se ocultan sin agencia, se resetean al quitarla y el backend los limpia. La sección Intermediación del detalle solo aparece con inmobiliaria. |
| 2026-11-23 | OpenCode | Regla `UI-NUMBER-INPUT`: componente `NumberInput` que permite vacío y elimina ceros a la izquierda. Migrados los formularios de alquiler. Los 21 archivos legacy quedan listados en `frontend/scripts/number-input-legacy.txt` para una spec de migración. |
| 2026-11-23 | OpenCode | Quitar una persona del contrato ahora abre `DeleteModal` con confirmación por palabra («confirmo») antes de eliminarla de la lista. |
| 2026-11-23 | OpenCode | Alta de personas del contrato en modal: rol, persona existente o nueva, email/teléfono y notificación. Al guardar se crea la persona si es nueva, se refresca el listado y se agrega a la pestaña Personas. El primer modal solo elige rol, persona y notificación; la opción «Agregar» abre un **segundo modal anidado** (`z-[70]`) con Nombre completo, Email y Teléfono. Al crear, se refresca el listado y la persona queda seleccionada en el primer modal. |
