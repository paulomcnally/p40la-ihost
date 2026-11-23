---
title: "Hogar: reparaciones y mantenimientos con histórico de insumos y mano de obra"
id: "SPEC-141"
status: "released"
author: "opencode"
created: "2026-11-23"
updated: "2026-11-23"
github_issue: 144
---

# Hogar: reparaciones y mantenimientos con histórico de insumos y mano de obra

**ID**: SPEC-141  
**Estado**: released  
**Autor**: opencode  
**Creado**: 2026-11-23  
**Actualizado**: 2026-11-23

---

## 1. Resumen Ejecutivo

La aplicación ya administra viviendas (`homes`), instituciones y monedas, pero no conserva qué se reparó en cada casa, cuánto costó ni quién lo hizo. El usuario necesita un módulo de **reparaciones y mantenimientos del hogar** donde cada trabajo quede registrado como un hecho histórico con dos tipos de gasto: **insumos** (materiales comprados) y **mano de obra** (lo que cobra el técnico o contratista). No se trata de un cronograma de tareas pendientes: es un **histórico de reparaciones** consultable por vivienda, con totales agrupados por moneda.

Cada línea de detalle lleva **su propia moneda**, porque una reparación puede combinar córdobas y dólares (mano de obra local e insumos importados). La reparación no impone una moneda única y la app nunca suma monedas distintas: muestra un total por moneda.

Como el gasto en insumos casi siempre ocurre en un comercio concreto (ferretería, ferretería eléctrica, depósito de materiales), el módulo se **pega a las instituciones existentes**: cada reparación puede indicar el proveedor/contratista y cada línea de insumo puede indicar en qué institución se compró. No se crea un maestro de proveedores paralelo ni se duplican datos de contacto. Se agrega una categoría de institución `hardware` («Ferretería») para clasificar esos comercios, pero el vínculo no se restringe a esa categoría para no bloquear casos reales.

La implementación respeta los límites del iHost: **cero dependencias nuevas**, una migración aditiva de dos tablas SQLite, consultas indexadas por `home_id` y `performed_at`, cálculo de totales en SQL/Go sin cargar históricos completos en memoria, y UI con los componentes obligatorios (`Select`, `NumberInput`, `CardMenu`), i18n es/en, darkmode y page de detalle de solo lectura.

---

## 2. Cumplimiento de reglas

Reglas del proyecto aplicables a esta spec (índice: `docs/rules/00-index.md`). Las reglas no se copian: se citan por ID.

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Go + `net/http` + SQLite; React/Tailwind ya existentes. Sin librerías nuevas en backend ni frontend. | `CA-ARCH-STACK` |
| `ARCH-LAYERS` | Dentro del módulo: `handlers.go` (parseo), `service.go` (validación y totales), `storage.go` (SQL), `model.go` (structs). | `CA-ARCH-LAYERS` |
| `ARCH-MODULES` | Módulo nuevo `home.maintenance` en `internal/modules/home/maintenance/` con los 5 archivos obligatorios + tests; registrado en `internal/modules/all/register.go`; rutas montadas por el propio módulo. | `CA-ARCH-MODULES` |
| `ARCH-GO` | `gofmt`, nombres de código en inglés, errores explícitos, `log/slog` si hace falta. | `CA-ARCH-GO` |
| `ARCH-IHOST` | Migración aditiva, índices, sin cron ni proceso extra, sin dependencias de red; memoria acotada en listados. | `CA-ARCH-IHOST` |
| `UI-SELECT` | Tipo de mantenimiento (`kind`), área (`area`), unidad de insumo (`unit`), **moneda de cada línea** y proveedor/institución usan `Select`; los de muchas opciones son `searchable`. | `CA-UI-SELECT` |
| `UI-SELECT-ADD` | `N/A`: el selector de instituciones no crea entidades; muestra un enlace «Crear institución» fuera del dropdown. | — |
| `UI-NUMBER-INPUT` | Los montos (`amount`) y la cantidad de insumos (`quantity`) usan `NumberInput`. Prohibido `type="number"`. | `CA-UI-NUMBER-INPUT` |
| `UI-INPUT-TOKENS` | Inputs, textarea y selects usan `bg-card`, `text-text`, `text-text-secondary`, `border-border`. | `CA-UI-TOKENS` |
| `UI-DARKMODE` | Formulario, detalle, modales y tab se verifican en darkmode antes de `pending_release`. | `CA-UI-DARKMODE` |
| `UI-TOGGLE` | `N/A`: la spec no agrega booleanos de entidad ni de configuración. | — |
| `UI-LIST-PATTERN` | El histórico se muestra como cards con menú de 3 puntos; sin registros usa `EmptyCard` con acción. | `CA-UI-LIST-PATTERN` |
| `UI-CREATE-MENU` | `N/A`: el alta es la acción principal del tab de negocio y se ofrece como botón primario dentro del tab; no hay header de listado raíz con acción de crear. | — |
| `UI-EDIT-LABEL` | Las acciones de edición (mantenimiento y línea) usan `app.edit`. | `CA-UI-EDIT-LABEL` |
| `UI-I18N` | Claves nuevas solo en `frontend/public/i18n/{es,en}.json`, con paridad y sin apuntar a objetos padre. | `CA-UI-I18N` |
| `UI-STYLE` | Cards `rounded-ios shadow-ios`, sidebar existente, menú de acciones, confirmación de borrado. | `CA-UI-STYLE` |
| `UX-BACK` | `/home/:id/maintenances/new`, `/home/:id/maintenances/:id` y `/home/:id/maintenances/:id/edit` se registran en `BACK_ROUTES` con retorno dinámico a `/home/:id?tab=maintenance`. | `CA-UX-BACK` |
| `UX-MODAL` | Modal de línea y `DeleteModal` a `z-[60]`/`z-[80]`, sin cierre por click fuera, con X/Cancelar siempre visible. | `CA-UX-MODAL` |
| `UX-MODAL-CONFIRM` | El borrado usa `DeleteModal` (palabra `confirmo`), que ya enfoca el input y acepta Enter. | `CA-UX-MODAL-CONFIRM` |
| `UX-DETAIL-ACTIONS` | En el detalle, Editar/Eliminar del mantenimiento y de cada línea viven en `CardMenu`; «Agregar insumo/mano de obra» se mantiene como acción primaria del flujo. | `CA-UX-DETAIL-ACTIONS` |
| `UX-DETAIL-READONLY` | El detalle muestra los datos sin formularios ni Guardar/Cancelar; editar navega a `/edit`. | `CA-UX-DETAIL-READONLY` |
| `UX-PREREQ` | Vivienda, moneda e institución (si se indica) se validan en backend y frontend; vivienda inexistente devuelve 404. | `CA-UX-PREREQ` |
| `UX-EMPTY` | Histórico vacío muestra `EmptyCard` explicando qué es y con acción de alta. | `CA-UX-EMPTY` |
| `UX-LOADING` | Spinner/skeleton en listado y detalle; botones deshabilitados mientras guardan. | `CA-UX-LOADING` |
| `UX-FEEDBACK` | Toast de éxito/error al crear, editar, borrar y agregar líneas; errores junto al campo. | `CA-UX-FEEDBACK` |
| `UX-FORMS` | Un Guardar y un Cancelar; validación bloquea el guardado; obligatorios marcados. | `CA-UX-FORMS` |
| `DATA-SQLITE` | Tablas `home_maintenances` y `home_maintenance_items` con FKs, CHECK e índices; SQL crudo sin ORM. | `CA-DATA-SQLITE` |
| `DATA-MIGRATIONS` | `0057_home_maintenance.up.sql` y `.down.sql` reversibles; probadas con `go test`. | `CA-DATA-MIGRATIONS` |
| `DATA-PRODCOPY` | `N/A`: la migración es aditiva y no transforma datos existentes; no hay matching contra datos reales. Los tests usan DB sintética en `/tmp`. | — |
| `DATA-NO-DROP` / `DATA-PROTECTED-SCHEMA` | `N/A`: no se toca `data/app.db`, ni `debts`/`debt_bills`. | — |
| `SEC-NO-PII` | Spec, tests y fixtures usan solo datos sintéticos; no se commitean DBs ni capturas reales. | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Ejemplos con `Casa Demo`, `Proveedor Demo`, `F000000000000001`, montos ficticios. | `CA-SEC-SYNTHETIC` |
| `TEST-LOCAL` | `go build ./...`, `go test ./...` y los tres guardrails de frontend antes de commit. | `CA-TEST-LOCAL` |
| `TEST-SERVER` | Se valida con `./scripts/check-server.sh` antes de avisar al usuario. | `CA-TEST-SERVER` |
| `TEST-USER-FLOW` | Se pregunta 🤖 antes de probar y antes de cerrar la spec. | `CA-TEST-USER-FLOW` |
| `DELIV-WORKTREE` | Trabajo en `../p40la-ihost-spec-141`, rama `feature/SPEC-141`. | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | Merge y push solo con confirmación explícita del usuario. | `CA-DELIV-CONFIRM` |
| `DELIV-HOOKS` | Commits `SPEC-141: …` con hooks instalados, sin `--no-verify`. | `CA-DELIV-HOOKS` |
| `DELIV-DOCKER` / `DELIV-DEPLOY` | `N/A`: la spec no cambia Dockerfile ni despliega; el deploy lo aprueba el usuario. | — |

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Registrar reparaciones/mantenimientos por vivienda con: fecha de realización (`performed_at`, no futura), tipo (`kind`: `corrective` reparación / `preventive` mantenimiento), área (`area`) y descripción (`description`, opcional). **La moneda no vive en la reparación**: cada línea de detalle lleva la suya, porque una misma reparación puede mezclar monedas (ej. mano de obra en NIO e insumos comprados en USD).
2. **REQ-002**: Cada registro guarda líneas de detalle en `home_maintenance_items` con tipo `labor` (mano de obra) o `supplies` (insumos), descripción, **moneda obligatoria (`currency_id`)** y monto total de la línea (`amount >= 0`). Las líneas de insumos aceptan además `quantity` (> 0, default 1) y `unit` (clave corta: `unit`, `m`, `m2`, `kg`, `lb`, `l`, `gal`, `sack`, `box`); la mano de obra los ignora.
3. **REQ-003**: Calcular y exponer `totals_by_currency` por registro (con `labor_cost`, `supplies_cost` y `total` de cada moneda) y `totals_by_currency` para el histórico completo de la vivienda, agrupando siempre por moneda. **Nunca** se suman monedas distintas.
4. **REQ-004**: Permitir asociar la reparación completa a una institución (`institution_id`): el contratista, taller o proveedor principal. Es **opcional** (permite registro DIY). Si se indica, debe existir y se guarda por referencia, sin campo de texto duplicado.
5. **REQ-005**: Permitir asociar cada **línea de insumos** a una institución (`institution_id`) que identifica dónde se compró el material. Es opcional. Las líneas de mano de obra no aceptan institución (se rechaza `400`).
6. **REQ-006**: Agregar la categoría de institución `hardware` («Ferretería» / «Hardware store») en la migración, sin modificar categorías existentes. El selector de proveedor muestra todas las instituciones (búsqueda por nombre) y las de `hardware`/`maintenance` primero.
7. **REQ-007**: Listar el histórico de una vivienda ordenado por `performed_at DESC, id DESC`, con paginación simple opcional y filtro por año y tipo (P1).
8. **REQ-008**: CRUD completo de registros y de líneas, validando en backend que el registro y la línea pertenezcan a la vivienda indicada en la ruta (404 si no).
9. **REQ-009**: Agregar un tab **Mantenimiento** en el detalle de la vivienda (`/home/:id?tab=maintenance`) con el histórico en cards, totales por moneda y acción de alta.
10. **REQ-010**: Página de detalle de solo lectura (`/home/:id/maintenances/:maintenanceId`) con desglose de mano de obra e insumos, totales y acciones en `CardMenu`.
11. **REQ-011**: Páginas de formulario `/home/:id/maintenances/new` y `/home/:id/maintenances/:maintenanceId/edit`.
12. **REQ-012**: Validar en backend y frontend: vivienda existente, moneda existente **en cada línea**, institución existente cuando se indica, fecha no futura, monto numérico no negativo y descripciones no vacías en las líneas. Una línea sin `currency_id` se rechaza con `400`.
13. **REQ-013**: Al eliminar un mantenimiento, eliminar en cascada sus líneas. La vivienda usa borrado lógico (`homes.deleted_at`), por lo que el histórico no se borra; la FK `ON DELETE CASCADE` queda como red de seguridad si en el futuro se hace borrado físico.
14. **REQ-014**: Todas las rutas usan el middleware de autenticación existente.
15. **REQ-015**: Agregar todas las cadenas a `frontend/public/i18n/es.json` y `en.json`; sin texto hardcodeado.
16. **REQ-016**: Usar `Select` para tipo, área, **moneda de cada línea** e instituciones; `NumberInput` para montos y cantidades; tokens de tema y verificación en darkmode.

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-017**: Filtro por año y por tipo en el histórico, con total del filtro por moneda.
2. **REQ-018**: Resumen en el tab de la vivienda: total histórico por moneda, cantidad de reparaciones y última fecha.
3. **REQ-019**: Edición de una línea sin salir del detalle (modal) y reordenamiento estable por fecha de creación.
4. **REQ-020**: Estado vacío con enlace a crear institución cuando no hay ninguna cargada.

### 3.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-021**: Adjuntar comprobantes/fotos por registro (reutilizando el patrón de adjuntos existente, con límites de tamaño y cantidad).
2. **REQ-022**: Programar mantenimientos preventivos recurrentes con recordatorio (fuera del alcance del histórico simple).
3. **REQ-023**: Importar una lista de precios del proveedor en CSV (manual, sin red) y guardar la factura de compra con número, fecha, RUC, subtotal, IVA 15% y total, adjuntando el PDF/foto del comprobante.
4. **REQ-024**: Generar una orden de compra en PDF a partir de insumos seleccionados y abrir un pedido prellenado por `wa.me`/`mailto` para enviarlo al proveedor. Sin automatizar el envío.

### 3.4 Requerimientos No Funcionales

- **Rendimiento**: listar 500 registros de una vivienda en < 150 ms; el listado no carga adjuntos ni detalles de líneas uno por uno (una sola consulta agrupada).
- **Memoria**: el histórico se pagina o se limita; no se mantienen en memoria los ítems de todos los registros a la vez.
- **Almacenamiento**: dos tablas nuevas + índices; crecimiento lineal con el uso (decenas de KB por año).
- **Seguridad**: autenticación obligatoria; validación de pertenencia vivienda→registro→línea.
- **iHost**: sin dependencias nuevas, sin cron, sin llamadas de red.
- **i18n**: paridad es/en y claves planas.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

Investigación con subagentes sobre CMMS (gestión de mantenimiento) y apps de propiedad del hogar. Hallazgos verificados:

1. **El mercado modela solicitud → orden de trabajo (OT) → líneas de costo → cierre**, no una entidad “reparación” única. UpKeep, MaintainX, Fiix, Limble, Property Meld y Housecall Pro coinciden en ese embudo.
2. **Los costos convergen en tres cubetas**: `LABOR` (horas × tarifa o monto fijo), `PARTS/MATERIALS` (cantidad × costo unitario) y `EXPENSE` (fletes, permisos, alquiler). Nuestra spec cubre las dos primeras, que son las que pidió el usuario; “otros” queda como P2 si aparece la necesidad.
3. **El histórico no es una tabla aparte**: es la OT cerrada consultada por activo/propiedad. Los reportes típicos son costo por activo y por año, historial del activo y fallas recurrentes. Confirma el enfoque “histórico por vivienda” de esta spec.
4. **Preventivo vs correctivo son dos ejes** (por qué se disparó y si estaba planificado). Para un histórico simple alcanza un enum de dos valores (`corrective`/`preventive`), sin estados de OT ni planificador.
5. **La recurrencia vive en el plan, no en la OT** (MaintainX/Grocy). Por eso los preventivos programados quedan explícitamente fuera del MVP (P2).
6. **Cantidad + unidad es crítico en ferretería**: los catálogos nicaragüenses venden por metro, m², libra, galón, saco y caja (SINSA publica precios por unidad de medida). Un insumo necesita `quantity` y `unit`, no solo un monto.
7. **Proveedores**: MaintainX usa un maestro de vendors con tres roles (ferretería, fabricante, técnico). En este proyecto ese maestro **ya existe** (`institutions` + `institution_categories`); crear otro violaría la simplicidad del iHost.
8. **LatAm/Nicaragua**: SINSA operó e-commerce sobre VTEX y EPA tiene portales por país, pero **no hay API B2B pública documentada** para ferreterías locales. La integración realista es carga manual/CSV de listas de precios y adjuntar el comprobante; no una integración en tiempo real.
9. **Facturación Nicaragua**: rige facturación computarizada autorizada por DGI (número correlativo, IVA desglosado). Guardar `invoice_number`/IVA es P2; el MVP solo guarda el costo.
10. **UX observada**: lista → detalle de solo lectura → formulario; OT como contenedor con pestañas (detalle, insumos, mano de obra, adjuntos); resumen de costos en bloques; `EmptyCard`; adjuntos con cámara. Nuestro diseño replica el patrón con dos tabs (mano de obra / insumos).
11. **No hay API B2B de ferreterías locales**: verificado en vivo (2026-11-23) que SINSA (VTEX) y Ferretería El Sol (WooCommerce) exponen solo **catálogo de tienda** sin autenticación, no cotización/compra; Do it Center y EPA (CR/GT/SV/VE, no Nicaragua) no exponen API pública. La integración realista es **carga manual/CSV** y adjuntar comprobantes.
12. **El flujo B2B real en la región es WhatsApp + PDF + Excel**: SINSA cotiza por WhatsApp y Do it PRO promete cotización por WhatsApp/celular. Una app personal no debe automatizar WhatsApp; alcanza con generar el pedido y abrir `wa.me`.
13. **“Proveedor” es una capacidad de la institución, no otra entidad** (Odoo/QuickBooks/Xero): identidad en `institutions`, perfil satélite 1:1 para RUC/contacto/plazo. El maestro ya existe; crear otro violaría la simplicidad del iHost.
14. **Nicaragua no exige factura electrónica a 2026**: se factura con talonario o sistema computarizado autorizado por DGI (IVA 15% desglosado, numeración correlativa). La app solo **almacena** comprobantes; no emite ni necesita conector fiscal.

Fuentes principales:
- UpKeep — costeo de OT: <https://help.onupkeep.com/en/articles/4925963-how-to-calculate-time-cost-to-a-work-order>
- UpKeep — detalle de OT (secciones y adjuntos): <https://help.onupkeep.com/en/articles/15545270-how-to-view-and-process-work-orders-on-web-new>
- MaintainX — datasets y campos de costo: <https://help.getmaintainx.com/datasets>
- MaintainX — planes de mantenimiento (fijo/flotante): <https://help.getmaintainx.com/about-maintenance-plans>
- MaintainX — vendors: <https://help.getmaintainx.com/about-vendors>
- Fiix — diccionario de datos de OT: <https://helpdesk.fiixsoftware.com/hc/en-us/articles/4407509790740-Data-dictionary-Work-order-audit-trail>
- Limble — reportes (historial de activo, costo por activo): <https://limble.com/learn/maintenance-reports>
- Property Meld — solicitudes y estados: <https://help.propertymeld.com/hc/en-us/articles/45827528593939-How-to-Create-Manage-and-Cancel-Meld-Requests>
- HomeZada — inventario, documentos y recordatorios: <https://www.homezada.com/faq>
- Grocy — recurrencia sin scheduler (vista `chores_current`): <https://github.com/grocy/grocy>
- SINSA — empresa y tienda (unidades de medida, C$): <https://www.sinsa.com.ni/nuestra-empresa>
- EPA — portales por país: <https://www.epaenlinea.com/>
- SINSA — catálogo VTEX (solo lectura, sin contrato B2B): <https://www.sinsa.com.ni/api/catalog_system/pub/products/search>
- Ferretería El Sol — Store API WooCommerce (solo lectura): <https://ferreteriaelsol.com/wp-json/wc/store/v1/products>
- Do it Center — cotización por WhatsApp (< 2 h): <https://www.doitcenter.com.pa/do-it-pro>
- Odoo — listas de precios de proveedor (`product.supplierinfo`): <https://www.odoo.com/documentation/20.0/applications/inventory_and_mrp/purchase/products/pricelist.html>
- DGI Nicaragua — facturación computarizada (Disposición Técnica 09-2007): <http://legislacion.asamblea.gob.ni/Normaweb.nsf>
- DGI Nicaragua — requisitos de factura y crédito fiscal: <https://www.dgi.gob.ni/FAQ/obligaciones_de_los_responsabl.htm>

> Los datos de ejemplo de esta spec son sintéticos. Los montos y comercios citados son referencias públicas de formato, no datos de una persona.

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| Reutilizar `auto_maintenances` con un “tipo de activo” | Cero tablas nuevas | Mezcla autos y casas, `mileage` obligatorio, rompe semántica y reportes | ❌ Rechazada |
| Tabla polimórfica `maintenances(owner_type, owner_id)` | Una sola tabla | Sin FKs reales, validaciones manuales, pierde integridad | ❌ Rechazada |
| Módulo `home.maintenance` con dos tablas propias | FKs, índices, semántica clara, sigue `ARCH-MODULES` | Duplica el patrón de autos | ✅ Seleccionada |
| Instituciones como maestro de proveedores paralelo | Control total | Duplica contactos y categorías ya existentes | ❌ Rechazada |
| Institución por registro y por línea de insumos | “Pegarse a instituciones” sin perder el detalle de compra | Dos FKs opcionales | ✅ Seleccionada |
| Integrar catálogo/precios de ferretería por API | Precios automáticos | APIs de storefront no oficiales, frágiles y sin contrato B2B; exige red en el iHost | ❌ Rechazada (P3 opcional, apagada por defecto) |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Módulo namespaced `home.maintenance`**
- **Contexto**: `ARCH-MODULES` exige módulos nuevos en `internal/modules/<ruta>/` con separación handlers/service/storage/model.
- **Decisión**: crear `internal/modules/home/maintenance/` y registrarlo en `internal/modules/all/register.go`. El módulo monta `/api/homes/{id}/maintenances…`.
- **Consecuencias**: aislamiento y testeabilidad; una ruta más de registro; no se toca el código legacy de autos.

**ADR-002: Histórico, no planificador**
- **Contexto**: el usuario pidió que los gastos “simplemente queden en un histórico de reparaciones”.
- **Decisión**: el MVP registra hechos consumados (fecha, costos). No hay tareas pendientes, vencimientos ni recordatorios.
- **Consecuencias**: alcance chico y claro; los preventivos programados quedan como P2.

**ADR-003: Institución opcional y sin restricción de categoría**
- **Contexto**: a diferencia de un taller mecánico, una reparación puede ser DIY, hacerla un conocido o comprarse en cualquier comercio.
- **Decisión**: `institution_id` es nullable tanto en el registro como en las líneas de insumos; si viene, se valida existencia pero no categoría. Se siembra `hardware` para clasificar ferreterías y ordenar el selector.
- **Consecuencias**: no se bloquean casos reales; los reportes por proveedor siguen funcionando.

**ADR-004: Moneda por línea, totales agrupados**
- **Contexto**: una misma reparación puede combinar monedas (mano de obra cobrada en córdobas e insumos comprados en dólares). Poner la moneda en la reparación obliga a partir el registro o a convertir mentalmente.
- **Decisión**: `currency_id` vive en `home_maintenance_items` (obligatoria). La reparación no tiene moneda. Las respuestas exponen `totals_by_currency` tanto por registro como para el histórico, con `labor_cost`, `supplies_cost` y `total` por moneda.
- **Consecuencias**: el registro refleja la realidad multi-moneda; la UI suma y muestra una línea por moneda y nunca mezcla valores. Las reparaciones de una sola moneda se ven igual que antes (una sola línea de total).

**ADR-005: Importes en `REAL`, no en centavos enteros**
- **Contexto**: la investigación de CMMS recomienda enteros de centavos para evitar errores de coma flotante. El proyecto ya almacena importes en `REAL` (servicios, autos, alquileres).
- **Decisión**: mantener `REAL` por consistencia con el esquema existente. El monto de cada línea se redondea a 2 decimales al escribir y los totales se calculan sumando esos valores ya redondeados.
- **Consecuencias**: un solo tipo de dinero en toda la app; se evita el error de mezclar `REAL` con centavos. Si el proyecto migra a centavos, este módulo se migra junto con el resto.

**ADR-006: Cantidad y unidad en insumos**
- **Contexto**: la ferretería vende por metro, m², libra, galón, saco y caja; un monto suelto pierde información útil del histórico.
- **Decisión**: agregar `quantity` y `unit` a las líneas de insumos desde la primera migración. `amount` sigue siendo el total de la línea (lo que el usuario pagó), no `quantity × unit_cost`.
- **Consecuencias**: el histórico explica “2 galones de pintura”; la UI muestra cantidad y unidad solo para insumos. No se guarda costo unitario en el MVP para no agregar un campo que el usuario tendría que completar dos veces.

**ADR-007: Sin integración de red; el vínculo es local**
- **Contexto**: las ferreterías de Nicaragua/Centroamérica no ofrecen API B2B; las APIs de storefront (VTEX/WooCommerce) no son contratos y pueden cambiar. El iHost debe funcionar offline.
- **Decisión**: el MVP no hace llamadas de red. “Pegarse a instituciones” significa FK a `institutions` (y a su categoría) + datos del comprobante cargados a mano. Importar listas de precios CSV, generar OC en PDF y registrar facturas quedan como P2/P3, siempre con acción manual y *feature flag* apagado.
- **Consecuencias**: cero dependencias externas y cero fragilidad; el usuario tipea el gasto, que es exactamente el alcance pedido (“simplemente quedan en un histórico”). Si más adelante quiere precios de catálogo, se agrega un importador opcional sin tocar este modelo.

**ADR-008: Borrar una institución no borra el histórico**
- **Contexto**: `institutions` se borra en duro y el proyecto tiene `foreign_keys=1`. Un FK por defecto bloquearía el borrado o, con `CASCADE`, destruiría reparaciones.
- **Decisión**: `institution_id` usa `ON DELETE SET NULL` en ambas tablas. La reparación y sus costos sobreviven; solo se pierde la referencia al proveedor.
- **Consecuencias**: el histórico nunca se pierde por limpiar instituciones. La UI muestra “—” cuando la institución ya no existe.

---

## 5. Diseño Técnico

### 5.1 Diagrama de arquitectura

```
[React: HomeShowPage tab] --> [HomeMaintenanceFormPage / ShowPage]
            |                              |
            +------ fetch /api/homes/{id}/maintenances ------+
                                                           |
                                    [módulo home.maintenance]
                                    handlers -> service -> storage
                                                           |
                                                      [SQLite]
                                    home_maintenances
                                    home_maintenance_items
```

### 5.2 Componentes

#### 5.2.1 Backend — módulo `home.maintenance`
- **Responsabilidad**: CRUD de registros y líneas, validación, cálculo de totales.
- **Interfaz**: ver §5.4.
- **Dependencias**: `*sql.DB`; helpers locales de fecha.
- **Ubicación**: `internal/modules/home/maintenance/`
  - `module.go`: `Info()` con ID `home.maintenance`, versión `1.0.0`, `Kind: core`, migración `0057_home_maintenance`; `RegisterRoutes`.
  - `model.go`: structs y constantes (`KindCorrective`, `KindPreventive`, `ItemLabor`, `ItemSupplies`).
  - `storage.go`: SQL del módulo; listado con totales, CRUD y `institution_name` por join.
  - `service.go`: validaciones y armado de totales.
  - `handlers.go`: parseo HTTP y mapeo de errores a 400/404/500.
  - `service_test.go` / `storage_test.go`: tests con DB en `/tmp`.

#### 5.2.2 Frontend
- **`frontend/src/types/homeMaintenance.ts`**: tipos `HomeMaintenance`, `HomeMaintenanceItem`, inputs y `HomeMaintenanceList`.
- **`frontend/src/api/homeMaintenance.ts`**: cliente del módulo (`list`, `get`, `create`, `update`, `delete`, `createItem`, `updateItem`, `deleteItem`).
- **`frontend/src/components/home/HomeMaintenanceTab.tsx`**: histórico dentro del detalle de la vivienda.
- **`frontend/src/components/home/HomeMaintenanceItemModal.tsx`**: alta/edición de línea (insumo/mano de obra).
- **`frontend/src/pages/HomeMaintenanceFormPage.tsx`**: formulario nuevo/editar.
- **`frontend/src/pages/HomeMaintenanceShowPage.tsx`**: detalle de solo lectura.
- **Integración**: `HomeShowPage` gana el tab `maintenance`; `App.tsx` monta las rutas; `DashboardLayout` registra `BACK_ROUTES`.

### 5.3 Modelo de datos

```sql
CREATE TABLE home_maintenances (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    home_id INTEGER NOT NULL,
    performed_at TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'corrective'
        CHECK (kind IN ('corrective', 'preventive')),
    area TEXT NOT NULL DEFAULT 'other'
        CHECK (area IN ('plumbing','electrical','roof','paint','carpentry',
                        'appliances','garden','cleaning','other')),
    institution_id INTEGER,
    description TEXT NOT NULL DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (home_id) REFERENCES homes(id) ON DELETE CASCADE,
    FOREIGN KEY (institution_id) REFERENCES institutions(id) ON DELETE SET NULL
);

CREATE TABLE home_maintenance_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    maintenance_id INTEGER NOT NULL,
    item_type TEXT NOT NULL CHECK (item_type IN ('labor','supplies')),
    description TEXT NOT NULL,
    quantity REAL NOT NULL DEFAULT 1 CHECK (quantity > 0),
    unit TEXT NOT NULL DEFAULT 'unit'
        CHECK (unit IN ('unit','m','m2','kg','lb','l','gal','sack','box')),
    amount REAL NOT NULL DEFAULT 0 CHECK (amount >= 0),
    currency_id INTEGER NOT NULL,
    institution_id INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (maintenance_id) REFERENCES home_maintenances(id) ON DELETE CASCADE,
    FOREIGN KEY (currency_id) REFERENCES currencies(id),
    FOREIGN KEY (institution_id) REFERENCES institutions(id) ON DELETE SET NULL
);
```

Índices: `(home_id, performed_at DESC, id DESC)`, `(institution_id)`, `(maintenance_id, item_type, id)`, `(currency_id)`.

### 5.4 APIs / Contratos

Base: `/api/homes/{homeID}/maintenances`, autenticadas.

#### `GET /api/homes/{homeID}/maintenances?year=YYYY&kind=corrective`
**Response 200**:
```json
{
  "items": [{
    "id": 1,
    "home_id": 1,
    "performed_at": "2026-03-15",
    "kind": "corrective",
    "area": "plumbing",
    "institution_id": 10,
    "institution_name": "Proveedor Demo",
    "description": "Cambio de llave de paso",
    "totals_by_currency": [
      { "currency_id": 2, "currency_code": "USD", "labor_cost": 40, "supplies_cost": 25.5, "total": 65.5 }
    ],
    "items": [],
    "created_at": "2026-03-15T10:00:00Z",
    "updated_at": "2026-03-15T10:00:00Z"
  }],
  "totals_by_currency": [
    { "currency_id": 1, "currency_code": "NIO", "labor_cost": 500, "supplies_cost": 900, "total": 1400 },
    { "currency_id": 2, "currency_code": "USD", "labor_cost": 150, "supplies_cost": 82.55, "total": 232.55 }
  ]
}
```

#### `POST /api/homes/{homeID}/maintenances`
```json
{ "performed_at": "2026-03-15", "kind": "corrective", "area": "plumbing",
  "institution_id": 10, "description": "Cambio de llave de paso" }
```
`201` con el registro creado. Errores: `400` validación, `404` vivienda/institución inexistente.

#### `GET|PUT|DELETE /api/homes/{homeID}/maintenances/{id}`
`PUT` acepta el mismo body; `DELETE` responde `204`.

#### `POST /api/homes/{homeID}/maintenances/{id}/items`
```json
{ "item_type": "supplies", "description": "Pintura blanca", "quantity": 2, "unit": "gal",
  "amount": 25.5, "currency_id": 2, "institution_id": 11 }
```
`201`; `400` si falta `currency_id`, si `item_type=labor` con `institution_id`, si `quantity <= 0` o si `unit` no es una clave válida; `404` si la moneda o la institución no existen.

#### `PUT|DELETE /api/homes/{homeID}/maintenances/{id}/items/{itemID}`
`PUT` mismo body; `DELETE` responde `204`.

### 5.5 Dependencias

- **Internas**: `homes`, `currencies`, `institutions`, `institution_categories`, middleware de auth, componentes `Select`/`NumberInput`/`CardMenu`/`DeleteModal`/`Toast`.
- **Externas**: ninguna nueva.

---

## 6. Criterios de Aceptación

### 6.1 Funcionales

- [x] `CA-HM-001`: Desde el tab Mantenimiento de una vivienda se crea una reparación con fecha, tipo, área y descripción; queda visible en el histórico.
- [x] `CA-HM-002`: Se agregan líneas de mano de obra e insumos con su moneda; el detalle muestra `labor_cost`, `supplies_cost` y `total` de cada moneda.
- [x] `CA-HM-003`: El histórico muestra un total por cada moneda usada, sin sumar monedas distintas.
- [x] `CA-HM-004`: Se asocia un proveedor a la reparación y una ferretería a una línea de insumos; el nombre se muestra en el detalle.
- [x] `CA-HM-005`: Una línea de mano de obra con `institution_id` es rechazada con `400`.
- [x] `CA-HM-006`: Editar y eliminar registros y líneas funciona y respeta la pertenencia vivienda→registro→línea.
- [x] `CA-HM-007`: Eliminar un registro elimina sus líneas (cascada verificada por test). La vivienda se borra de forma lógica (`deleted_at`), así que su histórico permanece en la DB y deja de ser accesible.
- [x] `CA-HM-008`: Fecha futura, monto negativo o descripción vacía de línea se rechazan en backend y se muestran junto al campo en frontend.
- [x] `CA-HM-009`: Filtro por año y tipo (P1) devuelve el subconjunto correcto y su total.
- [x] `CA-HM-010`: Los insumos guardan y muestran cantidad y unidad (ej. «2 gal»); la mano de obra no. `quantity <= 0` o unidad inválida se rechazan con `400`.
- [x] `CA-HM-011`: Borrar una institución referenciada pone `institution_id = NULL` y conserva la reparación y sus costos.
- [x] `CA-HM-012`: Una misma reparación puede tener mano de obra en NIO e insumos en USD; el detalle y el histórico muestran los totales separados por moneda y no los mezclan. Una línea sin moneda se rechaza con `400`.
- [x] `CA-ARCH-MODULES`: `Info()` devuelve `home.maintenance`, versión semver y la migración declarada; el módulo está registrado en `all/register.go`.
- [x] `CA-ARCH-LAYERS`: `handlers.go` no contiene SQL; `storage.go` es el único con consultas.
- [x] `CA-UI-SELECT`: tipo, área, moneda de cada línea e institución usan `Select`; `npm run check:ui` pasa.
- [x] `CA-UI-NUMBER-INPUT`: los montos usan `NumberInput`; `npm run check:numbers` pasa.
- [x] `CA-UI-TOKENS`: inputs y textarea usan tokens del tema; verificado en darkmode.
- [x] `CA-UI-I18N`: `npm run check:i18n` pasa y no hay claves faltantes en es/en.
- [x] `CA-UI-EDIT-LABEL`: las acciones de edición dicen «Editar» (`app.edit`).
- [x] `CA-UI-LIST-PATTERN`: histórico en cards con `CardMenu`; vacío con `EmptyCard`.
- [x] `CA-UX-MODAL`: el modal de línea y `DeleteModal` no cierran por click fuera y tienen cierre visible.
- [x] `CA-UX-MODAL-CONFIRM`: `DeleteModal` enfoca el input y acepta Enter con `confirmo`.
- [x] `CA-UX-DETAIL-ACTIONS`: editar/eliminar del registro y de las líneas están en `CardMenu`, sin íconos sueltos.
- [x] `CA-UX-DETAIL-READONLY`: el detalle no tiene formularios; Editar navega a `/edit`.
- [x] `CA-UX-BACK`: el retorno va en el header vía `BACK_ROUTES` a `/home/:id?tab=maintenance`.
- [x] `CA-UX-PREREQ`: vivienda/moneda/institución se validan en ambos lados.
- [x] `CA-UX-EMPTY`: histórico vacío muestra `EmptyCard` con acción.
- [x] `CA-UX-LOADING`: spinner en carga y botones deshabilitados al guardar.
- [x] `CA-UX-FEEDBACK`: toast de éxito y error en todas las acciones.
- [x] `CA-DATA-MIGRATIONS`: migración up/down reversible y test dedicado.
- [x] `CA-SEC-SYNTHETIC`: tests y ejemplos usan `Casa Demo`, `Proveedor Demo`, `demo-account-001`.
- [x] `CA-TEST-LOCAL`: `go build ./...`, `go test ./...`, `check:ui`, `check:i18n`, `check:numbers` pasan.

### 6.2 No funcionales

- [x] `CA-NF-001`: listar 500 registros de una vivienda responde en < 150 ms en la iHost con índices.
- [x] `CA-NF-002`: el listado no dispara una consulta por registro (sin N+1).
- [x] `CA-NF-003`: no se agregan dependencias a `go.mod` ni a `package.json`.

### 6.3 Testing

- **Unit tests (Go)**: validación de fecha/moneda/institución; rechazo de institución en mano de obra; cálculo de totales; cascada al borrar.
- **Integration tests**: migración 0057 up/down; CRUD vía handler con DB en `/tmp`.
- **Frontend**: build + `check:ui` + `check:i18n` + `check:numbers`; prueba manual de darkmode y móvil.
- **Datos**: DB sintética; no requiere copia de producción (`DATA-PRODCOPY` N/A).

---

## 7. Plan de Implementación

### 7.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|---|---|---|---|
| 1 | Migración 0057 + test up/down | 0.5 día | Ninguna |
| 2 | Módulo Go `home.maintenance` (model/storage/service/handlers) + registro | 1.5 días | Fase 1 |
| 3 | Tests Go (validaciones, totales, cascada) | 0.5 día | Fase 2 |
| 4 | Tipos + API client frontend | 0.5 día | Fase 2 |
| 5 | Tab, formulario y detalle + i18n es/en | 1.5 días | Fase 4 |
| 6 | Guardrails, darkmode, móvil y pruebas del usuario | 0.5 día | Fase 5 |

### 7.2 Milestones

1. **MVP**: P0 completo (registro + líneas + histórico + proveedor + i18n + guardrails).
2. **V1.0**: filtros por año/tipo, resumen en la vivienda y adjuntos de comprobantes.
3. **Futuro**: preventivos programados y órdenes de compra a proveedores.

---

## 8. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| El histórico crece y el tab se vuelve lento | Media | Medio | Índices por `home_id` + paginación y totales en una sola consulta |
| Usuarios esperan planificador de tareas, no histórico | Media | Medio | Nombrar el tab «Mantenimiento» y documentar que es histórico; preventivos en P2 |
| Duplicar proveedores en instituciones | Baja | Medio | Selector con búsqueda y enlace a crear institución; sin tabla paralela |
| Categoría `hardware` no existe en DBs viejas | Baja | Bajo | `INSERT … ON CONFLICT DO NOTHING` en la migración |
| Mezclar monedas en un total | Baja | Alto | `totals_by_currency`; test que lo cubre |
| Borrar una ferretería deja reparaciones sin proveedor | Baja | Bajo | `ON DELETE SET NULL` preserva costos y descripción; la UI muestra “—” |
| Presión por integrar precios de ferretería en línea | Media | Medio | Declarado P3 con flag apagado; el MVP es offline por diseño |
| Romper el detalle de vivienda al agregar el tab | Media | Medio | Tab aditivo, `BACK_ROUTES` y pruebas de regresión de `HomeShowPage` |

---

## 9. Notas y Referencias

- Reglas aplicadas: `docs/rules/00-index.md`, `10-architecture.md`, `20-ui.md`, `30-ux.md`, `40-data.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.
- Precedente directo: SPEC-124 (mantenimientos de autos con mano de obra e insumos).
- Precedente de módulo namespaced: `internal/modules/realestate/rental/` (SPEC-140) y SPEC-137.
- Precedente de UI: `AutoMaintenanceShowPage.tsx`, `MaintenanceModal.tsx` (sin copiar su `<input type="number">`, que viola `UI-NUMBER-INPUT`).

### Release

- **Estado**: released (2026-11-23)
- **Commits**: `5bb4568f`, `1a470c2d`, `4f10798a`
- **Merge a `main`**: `cc14ffe5`
- **Migración**: `0057_home_maintenance`
- **Issue**: #144 (cerrado)
- **Validación**: pruebas manuales del usuario en `http://localhost:8091` con base demo sintética; `go build ./...`, `go test ./...`, `check:ui`, `check:i18n`, `check:numbers` y build de Vite en verde.

---

## 10. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-23 | opencode | Creación inicial de la especificación |
| 2026-11-23 | opencode | Investigación con subagentes (CMMS/hogar y proveedores) integrada en §4; ajustes de modelo: `quantity`/`unit`, FKs `ON DELETE SET NULL`, ADR-005 a ADR-008 |
| 2026-11-23 | opencode | Implementación P0: migración `0057`, módulo `home.maintenance` (model/storage/service/handlers/tests), tab en la vivienda, formulario, detalle, modal de líneas, i18n es/en y guardrails. Pendiente validación del usuario |
| 2026-11-23 | opencode | Cambio de diseño pedido por el usuario: la moneda pasa de la reparación a **cada línea** (`home_maintenance_items.currency_id`); totales por moneda por registro y por vivienda (ADR-004 reescrito, REQ-001/002/003/012/016 y CA-HM-001/002/012 ajustados) |
| 2026-11-23 | opencode | Release: merge `cc14ffe5` a `main`, estado `released`, issue #144 cerrado y tracker actualizado |
