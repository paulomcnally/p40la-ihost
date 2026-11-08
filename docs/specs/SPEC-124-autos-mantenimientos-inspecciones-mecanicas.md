---
title: "Autos: mantenimientos por kilometraje e inspecciones mecánicas anuales con PDFs"
id: "SPEC-124"
status: "in_progress"
author: "opencode"
created: "2026-11-07"
updated: "2026-11-07"
github_issue: 127
---

# Autos: mantenimientos por kilometraje e inspecciones mecánicas anuales con PDFs

**ID**: SPEC-124  
**Estado**: in_progress  
**Autor**: opencode  
**Creado**: 2026-11-07  
**Actualizado**: 2026-11-07

---

## 1. Resumen Ejecutivo

El módulo de Autos hoy permite registrar el vehículo y sus pólizas de seguro (SPEC-024, SPEC-025, SPEC-028, SPEC-067, SPEC-117, SPEC-118), pero no conserva el historial de mantenimientos ni el cumplimiento de las inspecciones mecánicas. El usuario necesita registrar cada mantenimiento con su kilometraje, taller, mano de obra e insumos, y cada inspección mecánica con su institución, resultado y comprobantes PDF. Ambos tipos de registro deben aceptar **todos los PDFs que hagan falta**, porque una factura, una orden de trabajo o un certificado pueden venir en varios archivos.

Los mantenimientos se rigen por kilometraje y **no** llevan una alerta automática en esta spec: se registran como historial y el kilometraje más reciente se muestra como referencia. Las inspecciones mecánicas, en cambio, se rigen por fecha con una regla explícita: para un vehículo modelo `Y`, la primera inspección vence el **1 de enero de `Y + 2`** y luego todos los años. La aplicación debe calcular la próxima inspección pendiente y mostrar un estado visible (al día, próxima o vencida) en el detalle del auto y en el listado, sin depender de un cron ni de una notificación externa.

La implementación debe respetar las limitaciones del iHost: cero dependencias nuevas, dos tablas SQLite nuevas más una tabla de adjuntos, subida de PDFs en streaming a disco (sin cargar archivos completos en memoria), límites de tamaño y cantidad, y consultas indexadas. La UI reutiliza los patrones existentes: secciones dentro del detalle del auto, modales de alta/edición, `CardMenu` para acciones, componente `Select` (nunca `<select>` nativo), tokens de tema y verificación en darkmode.

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Registrar mantenimientos por auto con los campos obligatorios: fecha de realización (`performed_at`), kilometraje (`mileage`), institución de categoría **Taller Mecánico** (`institution_id`) y moneda (`currency_id`). La mano de obra y los insumos se registran como líneas de detalle (`auto_maintenance_items`), cada una con tipo (`labor`/`supplies`), descripción y valor. Los totales de mano de obra, insumos y general se calculan sumando las líneas.
2. **REQ-002**: El mantenimiento debe estar asociado a una institución de categoría **Taller Mecánico**. La institución es obligatoria; el nombre del taller se obtiene de la relación `institution_id` y no existe un campo de texto separado ni una columna `workshop_name`.
3. **REQ-003**: Permitir descripción libre del trabajo realizado (`description`, opcional) y ordenar los mantenimientos por fecha descendente.
4. **REQ-004**: Permitir subir **múltiples PDFs** por mantenimiento (0 a N, hasta el límite técnico definido en REQ-014), listarlos, descargarlos/abrirlos y eliminarlos individualmente.
5. **REQ-005**: Registrar inspecciones mecánicas por auto con los campos obligatorios: fecha de realización (`performed_at`), año de la inspección (`inspection_year`), institución de categoría **Inspección Mecánica** y resultado (`approved`/`rejected`). `notes` es opcional.
6. **REQ-006**: Validar en backend y frontend que la institución de una inspección pertenezca a la categoría `mechanical_inspection`. Una institución de otra categoría debe rechazarse con `400`.
7. **REQ-007**: Calcular el calendario de inspecciones a partir del año del auto: primera inspección en `auto.year + 2`, con fecha de vencimiento **1 de enero** de ese año, y luego una por año. El cálculo se hace al vuelo (no se persiste un cronograma).
8. **REQ-008**: Determinar la próxima inspección pendiente como el primer año del calendario que no tenga una inspección **aprobada** registrada. Una inspección rechazada no completa el año.
9. **REQ-009**: Exponer el estado de la próxima inspección: `overdue` (vencida), `due_soon` (vence en los próximos 60 días, incluido el mismo día de vencimiento) u `ok` (al día). Mostrar en el detalle del auto el estado, la fecha de vencimiento y el año pendiente.
10. **REQ-010**: Mostrar en el listado de autos un indicador compacto del estado de inspección de cada vehículo, para que una inspección vencida o próxima sea fácil de identificar sin abrir el detalle.
11. **REQ-011**: Permitir subir **múltiples PDFs** por inspección (0 a N), listarlos, descargarlos/abrirlos y eliminarlos individualmente.
12. **REQ-012**: Sembrar dos categorías de institución con claves estables: `workshop` ("Taller Mecánico") y `mechanical_inspection` ("Inspección Mecánica"). Los selectores de institución de cada formulario se filtran por la categoría correspondiente.
13. **REQ-013**: Al eliminar un mantenimiento, una inspección o un auto, eliminar en cascada sus filas de adjuntos y los archivos PDF asociados en disco. Al eliminar un adjunto individual, eliminar su fila y su archivo.
14. **REQ-014**: Aplicar límites de archivo: solo PDF, máximo **10 MB por archivo**, máximo **20 adjuntos por registro**, **un archivo por request**. La subida debe escribir a un archivo temporal y renombrar de forma atómica; no debe leer el archivo completo en memoria.
15. **REQ-015**: Validar los PDFs por extensión (`.pdf`, case-insensitive), tamaño mayor a 0 y firma mágica `%PDF-` en los primeros 5 bytes. Descargar con `Content-Type: application/pdf`, `Content-Disposition` inline (o `attachment` con `?download=1`) y `X-Content-Type-Options: nosniff`.
16. **REQ-016**: Validar todos los endpoints con el middleware de autenticación existente y verificar que el registro y el adjunto pertenezcan al auto indicado en la ruta.
17. **REQ-017**: Mostrar el kilometraje más reciente del auto en la sección de mantenimientos (calculado del mantenimiento con mayor kilometraje, no necesariamente el de fecha más reciente).
18. **REQ-018**: Agregar todas las cadenas nuevas a `frontend/public/i18n/es.json` y `frontend/public/i18n/en.json`, y usarlas en la UI. No agregar texto hardcodeado en las secciones nuevas.
19. **REQ-019**: Usar el componente `Select` para institución, moneda, año de inspección y resultado. Prohibido `<select>` nativo. Todos los inputs deben usar tokens del tema (`bg-card`, `text-text`, `text-text-secondary`, `border-border`).
20. **REQ-020**: La UI debe ser responsive (móvil y desktop) y verificarse en darkmode (texto y placeholders legibles).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-021**: Editar mantenimientos e inspecciones existentes, incluida la fecha, el kilometraje, la institución, los montos, el resultado y las notas.
2. **REQ-022**: Mostrar la cantidad y el tamaño total de adjuntos por registro, y advertir cuando se alcanza el máximo de 20.
3. **REQ-023**: Si no existe ninguna institución con la categoría requerida, mostrar un estado vacío con acción para crear una institución, en lugar de un selector vacío sin explicación.
4. **REQ-024**: Filtrar el historial de inspecciones por año y mostrar el cronograma completo (años cumplidos, pendiente y futuros) en el detalle del auto.
5. **REQ-025**: Mostrar el costo total de mantenimientos por moneda en la sección de mantenimientos (suma de mano de obra e insumos de los registros mostrados).
6. **REQ-026**: Permitir descargar un adjunto con nombre original y `?download=1`; por defecto abrirlo inline en una pestaña nueva.
7. **REQ-027**: Al cambiar el año del auto, recalcular el cronograma de inspecciones. Si el nuevo año deja años de inspección registrados por debajo del mínimo (`auto.year + 2`), mostrar una advertencia y no borrar los registros existentes.
8. **REQ-032**: Organizar el detalle del auto en tres tabs —**Seguros**, **Mantenimientos** e **Inspecciones**— para evitar una sola página con demasiada información. El tab activo se refleja en el querystring (`?tab=seguros|mantenimientos|inspecciones`), por defecto `seguros`. Cada tab muestra su contador y solo se renderiza el panel activo; el tab de inspecciones muestra un indicador de estado cuando está vencida o próxima.
9. **REQ-033**: Mostrar en cada fila de mantenimiento e inspección una acción visible **Adjuntar PDF** que abra un modal dedicado para gestionar los comprobantes de un registro ya creado, sin tener que editar el resto de sus campos. El modal permite subir varios PDFs y eliminar los existentes, y el contador de la fila se actualiza al cerrar.
10. **REQ-034**: Mostrar en cada fila de mantenimiento e inspección una acción visible **Editar** además del menú de 3 puntos. En el banner de inspecciones, cuando el estado es `overdue` o `due_soon`, incluir una acción directa **Registrar inspección de {año}** que abra el modal con el año pendiente preseleccionado. El estado “Vencida” es el resultado calculado del ciclo pendiente y no un registro editable: si no existe una inspección para ese año, el usuario debe crearla.
11. **REQ-035**: Organizar los modales de crear/editar mantenimiento e inspección en dos tabs: **Información** y **Facturas**. Información contiene los campos del registro; Facturas contiene la gestión de PDFs (subir, abrir, eliminar y pendientes). El tab Facturas muestra el contador de adjuntos existentes más pendientes. Si se intenta guardar desde Facturas con datos obligatorios incompletos, el modal vuelve a Información y muestra el error.
12. **REQ-036**: No almacenar el nombre del taller como columna. La migración `0043` elimina `workshop_name`, migra los talleres libres existentes a instituciones de categoría `workshop` y deja `institution_id` como `NOT NULL`. El formulario solo muestra el selector de taller y la API rechaza un mantenimiento sin `institution_id`.
13. **REQ-037**: Registrar la mano de obra y los insumos como líneas únicas en `auto_maintenance_items` (`item_type`, `description`, `amount`), asociadas al mantenimiento. Un mantenimiento puede crearse sin líneas y agregarlas después desde su detalle; cada línea debe tener descripción y valor no negativo. Los totales `labor_cost`, `supplies_cost` y `total_cost` se calculan en la respuesta y no se persisten. La migración `0044` crea la tabla, migra los escalares existentes a líneas “Mano de obra” e “Insumos” y elimina las columnas escalares.
14. **REQ-038**: Tener una página de detalle por mantenimiento en `/autos/:id/maintenances/:maintenanceId`, accesible desde la opción **Ver** del menú de 3 puntos de la fila. En la lista, el menú de la fila muestra **Ver** y **Eliminar**; la edición se hace desde el detalle. El detalle se organiza en cuatro tabs —**Información**, **Mano de obra**, **Insumos** y **Facturas**— con el tab activo en `?tab=informacion|mano_obra|insumos|facturas` (por defecto `informacion`). Permite editar la información básica, agregar/editar/eliminar líneas y subir/eliminar PDFs. El modal de crear/editar mantenimiento queda solo con la información básica (fecha, kilometraje, taller, descripción y moneda), sin conceptos ni archivos.
15. **REQ-039**: Exponer CRUD de líneas de mantenimiento (`POST`, `PUT`, `DELETE` en `/api/autos/{id}/maintenances/{maintenance_id}/items[/{item_id}]`) y `GET /api/autos/{id}/maintenances/{maintenance_id}` para el detalle. Cada operación debe verificar que el mantenimiento y la línea pertenezcan al auto de la ruta.
16. **REQ-040**: Aplicar las reglas de modales vigentes en `main` (SPEC-122, `docs/project-rules.md` §4.7): ningún modal se cierra al hacer click fuera; el overlay usa `z-[60]` o superior y bloquea la página de fondo; siempre hay X o Cancelar/Cerrar; `DeleteModal` usa `z-[70]`, enfoca el input al abrir y confirma con Enter cuando el texto es `confirmo`.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-028**: Incluir las inspecciones próximas y vencidas en el resumen diario de alertas por email y Telegram, reutilizando la infraestructura de alertas existente (SPEC-029, SPEC-033, SPEC-088).
2. **REQ-029**: Estimar el próximo mantenimiento por intervalo de kilometraje configurable (por ejemplo, cada 5.000 km). Esta spec **no** implementa alertas de mantenimiento; solo deja el kilometraje registrado.
3. **REQ-030**: Reporte de gastos de mantenimiento por auto, año y moneda, con gráfico simple.
4. **REQ-031**: Registrar el costo de la inspección mecánica (monto y moneda) además del resultado y el comprobante.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: listar mantenimientos/inspecciones de un auto en menos de 150 ms con 500 registros y 2.000 adjuntos; el cálculo del cronograma debe ser O(años) y no consultar la base por cada año.
- **Memoria**: la subida de PDFs no debe mantener más de ~1 MB en memoria por request; usar `http.MaxBytesReader`, `multipart` con archivo temporal y `io.Copy` con `LimitReader`.
- **Almacenamiento**: PDFs en el volumen `DATA_DIR`, fuera de SQLite. La base guarda solo metadatos. Tamaño máximo teórico por registro: 20 × 10 MB = 200 MB; el volumen de datos debe respaldarse completo (la DB sola no incluye los PDFs).
- **Seguridad**: autenticación en todos los endpoints, validación de pertenencia auto → registro → adjunto, nombres de archivo generados por ID (nunca por nombre del usuario), validación de firma PDF, `nosniff`.
- **iHost**: sin dependencias Go nuevas; solo `net/http`, `mime/multipart`, `database/sql`, `os`, `io`, `path/filepath`. Sin procesos ni workers adicionales.
- **Base de datos**: SQLite con WAL; las tablas nuevas usan `ON DELETE CASCADE` para `auto_id` y están indexadas por `auto_id` y por la clave de orden.
- **Disponibilidad**: las operaciones de archivo no bloquean las consultas; no se agregan locks globales.
- **Compatibilidad**: no romper los endpoints existentes de autos ni el contrato de `GET /api/autos`.

---

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se revisó el estado actual del repositorio (rama `main`, commit `ec19532`):

- **Autos**: `internal/models/auto.go`, `internal/storage/auto.go`, `internal/services/auto.go`, `internal/api/auto_handlers.go`. La tabla `autos` no tiene kilometraje ni fechas de mantenimiento.
- **Seguros**: `auto_services` (migración `0010`), con columnas agregadas en `0012`; el detalle del auto (`frontend/src/pages/AutoShowPage.tsx`) ya muestra una sección "Seguros" con `CardMenu` y modales de alta/edición. Es el patrón visual a replicar.
- **Instituciones y categorías**: `institutions.category_id` referencia `institution_categories` (`0011`). Las claves sembradas son en inglés (`insurance`, `transport`, `maintenance`, etc.). **No existen** las categorías `workshop` ni `mechanical_inspection`. El filtro por categoría ya tiene precedente en `internal/storage/auto_service.go` (`ic.key = 'insurance'`).
- **Adjuntos PDF**: hay tres precedentes:
  - Comprobantes de pensión: `{DATA_DIR}/uploads/payment-proofs/{year}/{month}/{record_id}.ext`, un archivo por registro, 10 MB, sin validación de firma PDF (SPEC-051).
  - Comprobantes de pago de deudas (SPEC-122): `{DATA_DIR}/documents/{debt_id}/receipt_{installment}.pdf`, un archivo por cuota, 10 MB, validación `%PDF-`, escritura atómica con `os.CreateTemp` + `os.Rename`.
  - Contratos (SPEC-123): `{DATA_DIR}/contracts/...`, un archivo fijo `contract.pdf` por entidad.
  Ningún precedente soporta **múltiples archivos por registro**; esta spec introduce una tabla de metadatos y una carpeta por registro.
- **Migraciones**: la última es `0041_create_pension_import`. Las nuevas migraciones son `0042_create_auto_maintenance_inspection`, `0043_auto_maintenance_institution_reference` (elimina `workshop_name` y hace `institution_id` obligatorio) y `0044_auto_maintenance_items` (líneas de mano de obra e insumos).
- **UI**: el detalle del auto usa modales (`AddInsuranceModal`, `DeleteModal`) y `CardMenu`. No hay componente de tabs compartido; para no agregar rutas nuevas se mantienen secciones y modales.
- **Moneda**: `currencies` (`0002`) y el selector `Select` con `api.currencies.list()`; los formularios de deuda y servicio inicializan con `currencies[0]`.
- **i18n**: fuente de verdad `frontend/public/i18n/{es,en}.json`; el build copia y valida. SPEC-060/063/004/118 fijan las reglas de darkmode, flecha atrás y `Select`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Páginas nuevas `/autos/:id/mantenimientos` e `/autos/:id/inspecciones` | Más espacio, URLs estables | Más rutas, `BACK_ROUTES`, más componentes y navegación; el detalle ya resuelve seguros con modales | ❌ Rechazada para el MVP |
| Secciones + modales dentro de `AutoShowPage` | Consistente con seguros, menos rutas, menos código, sin cambios de `BACK_ROUTES` | El modal puede quedar largo en desktop; se resuelve con scroll interno | ✅ Seleccionada |
| Un PDF por registro (patrón actual) | Simple, reutiliza precedentes | No cumple "los que se necesiten" | ❌ Rechazada |
| Tabla `auto_attachments` con `owner_type`/`owner_id` | Una sola implementación de upload/download/delete para ambos módulos; menos código | Sin FK polimórfica; la integridad se valida en el servicio | ✅ Seleccionada |
| Dos tablas de adjuntos separadas | FK real por tabla | Duplica storage, handlers, tests y rutas | ❌ Rechazada |
| Guardar PDFs como BLOB en SQLite | Un solo backup | Aumenta la DB, WAL y memoria del iHost; SQLite no está pensada para archivos grandes | ❌ Rechazada |
| PDFs en disco + metadatos en DB | Bajo uso de memoria, backup del volumen, streaming | Hay que limpiar archivos huérfanos | ✅ Seleccionada |
| Calendario de inspecciones persistido (tabla/cron) | Consulta directa | Duplica estado, se desincroniza al editar el auto y requiere migraciones de datos | ❌ Rechazada |
| Calendario calculado al vuelo desde `auto.year` + inspecciones aprobadas | Sin estado duplicado, se recalcula al cambiar el año | Hay que testear bien los bordes | ✅ Seleccionada |
| Dos categorías nuevas (`workshop`, `mechanical_inspection`) | Filtros claros y validación por clave estable | Hay que asignar categoría a instituciones existentes | ✅ Seleccionada |
| Reutilizar `maintenance`/`transport` | Sin migración de categorías | `maintenance` es mantenimiento del hogar; `transport` incluye gasolina/peajes; impreciso | ❌ Rechazada |
| Alertas de inspección por email/Telegram en el MVP | Aviso proactivo | Aumenta alcance y depende del scheduler; el usuario no lo pidió explícitamente | ⏳ P2 |
| Alerta de mantenimiento por kilometraje | Proactiva | No hay odómetro ni lectura automática; el usuario indicó que no es fácil de identificar | ❌ Fuera de alcance |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: PDFs en disco y metadatos en SQLite.
- **Contexto**: el iHost tiene recursos limitados y SQLite con WAL; guardar PDFs como BLOB infla la DB y las escrituras.
- **Decisión**: guardar los archivos bajo `{DATA_DIR}/autos/{auto_id}/{maintenances|inspections}/{owner_id}/{attachment_id}.pdf` y persistir solo metadatos en `auto_attachments`.
- **Consecuencias**: el backup debe incluir todo `DATA_DIR`; se necesita limpieza explícita de archivos al borrar. Se evita cargar archivos en memoria y se sirven con `http.ServeFile`.

**ADR-002**: Cronograma de inspecciones calculado, no persistido.
- **Contexto**: la regla depende del año del auto y de las inspecciones aprobadas; persistir un cronograma obliga a recalcularlo al editar el auto.
- **Decisión**: calcular `first_due_year = auto.year + 2`, `next_due_year` como el primer año no aprobado y `next_due_date = next_due_year-01-01` en cada consulta.
- **Consecuencias**: no hay migración de datos ni cron; el cálculo debe estar cubierto por tests de borde. Si se cambia el año del auto, el cronograma cambia automáticamente.

**ADR-003**: Dos categorías de institución nuevas con claves en inglés.
- **Contexto**: las claves de `institution_categories` son estables, en minúsculas y sin acentos (`insurance`, `maintenance`).
- **Decisión**: sembrar `workshop` ("Taller Mecánico") y `mechanical_inspection` ("Inspección Mecánica") con `ON CONFLICT(key) DO NOTHING`.
- **Consecuencias**: las instituciones existentes deben asignarse manualmente a la categoría; los selectores muestran un estado vacío con acción de creación. No se reutilizan categorías semánticamente distintas.

**ADR-004**: Un archivo por request, escritura atómica y streaming.
- **Contexto**: el iHost tiene RAM limitada; `r.FormFile` con `ParseMultipartForm` puede volcar a disco, pero leer todo con `io.ReadAll` consume memoria.
- **Decisión**: `POST` multipart con un solo `file`; `http.MaxBytesReader` a 11 MB; `io.Copy` con `LimitReader` a un temporal en el directorio destino; validar `%PDF-`; `os.Rename` atómico; `defer` cleanup.
- **Consecuencias**: el frontend sube los archivos en secuencia después de crear el registro; si un archivo falla, los anteriores quedan y se informa el error.

**ADR-005**: Tabla genérica de adjuntos con integridad en el servicio.
- **Contexto**: SQLite no soporta FKs polimórficas y duplicar tablas aumenta el código.
- **Decisión**: `auto_attachments(auto_id, owner_type, owner_id, ...)` con `CHECK owner_type IN ('maintenance','inspection')`; el servicio valida que `owner_id` exista, pertenezca a `auto_id` y que el tipo sea correcto. `auto_id` tiene FK real con `ON DELETE CASCADE`.
- **Consecuencias**: las filas huérfanas solo pueden originarse por bugs; se cubren con tests y con la limpieza en las eliminaciones.

**ADR-006**: Estado de inspección calculado y umbral de 60 días.
- **Contexto**: se necesita una señal visual simple, sin scheduler.
- **Decisión**: `overdue` si `hoy > next_due_date`; `due_soon` si `0 <= días <= 60`; `ok` en el resto. El mismo cálculo se usa en detalle y listado.
- **Consecuencias**: el estado depende de la fecha actual y no se persiste; no hay notificaciones automáticas en el MVP (P2).

---

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
┌──────────────────────────────┐
│ React (AutoShowPage)         │
│  ├─ Sección Mantenimientos   │
│  ├─ Sección Inspecciones     │
│  └─ Modales + adjuntos PDF   │
└──────────────┬───────────────┘
               │ JSON + multipart/form-data
┌──────────────▼───────────────┐
│ internal/api                 │
│  auto_maintenance_handlers   │
│  auto_inspection_handlers    │
│  auto_attachment_handlers    │
└──────────────┬───────────────┘
               │
┌──────────────▼───────────────┐
│ internal/services            │
│  AutoMaintenanceService      │
│  AutoInspectionService       │
│  AutoAttachmentService       │
│  InspectionSchedule (regla)  │
└───────┬──────────────┬───────┘
        │              │
┌───────▼───────┐ ┌────▼────────────────────┐
│ storage       │ │ DATA_DIR/autos/...      │
│ SQLite        │ │ PDFs (streaming)        │
└───────────────┘ └─────────────────────────┘
```

### 4.2 Componentes

#### 4.2.1 Migraciones `0042`, `0043` y `0044`
- **Responsabilidad**: `0042` crea las tablas, índices y categorías de institución. `0043` migra los talleres libres existentes a instituciones, elimina la columna `workshop_name` y deja `institution_id` como `NOT NULL`. `0044` crea `auto_maintenance_items`, migra `labor_cost`/`supplies_cost` a líneas y elimina esas columnas escalares.
- **Ubicación**: `migrations/0042_create_auto_maintenance_inspection.{up,down}.sql`, `migrations/0043_auto_maintenance_institution_reference.{up,down}.sql` y `migrations/0044_auto_maintenance_items.{up,down}.sql`.

#### 4.2.2 Modelos
- **Responsabilidad**: structs de dominio y DTOs de respuesta.
- **Archivos**:
  - `internal/models/auto_maintenance.go`: `AutoMaintenance`, `AutoMaintenanceDetail`, `AutoMaintenanceInput`.
  - `internal/models/auto_inspection.go`: `AutoInspection`, `AutoInspectionDetail`, `AutoInspectionInput`, `InspectionSchedule`.
  - `internal/models/auto_attachment.go`: `AutoAttachment`.
- **Campos calculados** (no persistidos): `institution_name`, `currency_code`, `total_cost`, `attachments`, `next_due_date`, `status`, `days_until_due`.

#### 4.2.3 Storage
- **Responsabilidad**: SQL crudo, sin lógica de negocio.
- **Archivos**:
  - `internal/storage/auto_maintenance.go`: `ListByAuto`, `GetByID`, `Create`, `Update`, `Delete`, `LatestMileage`, `TotalsByCurrency`, `ListItemsByAuto`, `ListItemsByMaintenance`, `CreateItem`, `UpdateItem`, `DeleteItem`.
  - `internal/storage/auto_inspection.go`: `ListByAuto`, `GetByID`, `Create`, `Update`, `Delete`, `ApprovedYears`, `ListAllSchedules` (para el indicador del listado).
  - `internal/storage/auto_attachment.go`: `ListByOwner`, `ListByAuto`, `GetByID`, `Create`, `Delete`, `CountByOwner`, `DeleteByAuto`.
  - `internal/storage/institution.go` (extensión): `GetCategoryKey(ctx, id)` para validar la categoría.
- **Índices**: ver DDL en §4.3.

#### 4.2.4 Servicios
- **Responsabilidad**: validaciones, regla de Inspección, coordinación de archivos.
- **Archivos**:
  - `internal/services/auto_maintenance.go`: valida auto, fechas, kilometraje, moneda e institución/taller; CRUD de líneas; calcula totales; delega storage.
  - `internal/services/auto_inspection.go`: valida auto, año de inspección, institución de categoría `mechanical_inspection`, resultado; calcula `InspectionSchedule`; valida años de inspección contra `auto.year + 2`.
  - `internal/services/auto_attachment.go`: `MaxAutoAttachmentBytes = 10 * 1024 * 1024`, `MaxAutoAttachmentsPerRecord = 20`; guarda/valida/borra PDFs; resuelve rutas; limpia el directorio del auto al eliminarlo.
  - `internal/services/inspection_schedule.go`: función pura `BuildInspectionSchedule(autoYear int64, approvedYears []int, today time.Time) models.InspectionSchedule` (testeable sin DB).
- **Errores centinela**: `ErrAutoNotFound`, `ErrMaintenanceNotFound`, `ErrInspectionNotFound`, `ErrAttachmentNotFound`, `ErrAttachmentInvalid`, `ErrAttachmentTooLarge`, `ErrTooManyAttachments`, `ErrInstitutionCategory`.

#### 4.2.5 API
- **Responsabilidad**: parseo de requests, status codes, llamadas a services.
- **Archivos**:
  - `internal/api/auto_maintenance_handlers.go`
  - `internal/api/auto_inspection_handlers.go`
  - `internal/api/auto_attachment_handlers.go`
- **Wiring**: `cmd/server/main.go` crea storages, services y handlers; `internal/api/handlers.go` agrega los campos; `internal/api/routes.go` registra las rutas con `authMiddleware`.

#### 4.2.6 Frontend
- **Responsabilidad**: secciones, formularios, adjuntos y estado.
- **Archivos**:
  - `frontend/src/pages/AutoShowPage.tsx`: tabs de Seguros/Mantenimientos/Inspecciones, listados y opción **Ver**.
  - `frontend/src/pages/AutoMaintenanceShowPage.tsx`: detalle de mantenimiento con información básica, líneas y archivos.
  - `frontend/src/components/MaintenanceModal.tsx`: alta/edición de la información básica del mantenimiento.
  - `frontend/src/components/MaintenanceItemModal.tsx`: alta/edición de líneas de mano de obra e insumos.
  - `frontend/src/components/InspectionModal.tsx`: alta/edición de inspección con adjuntos.
  - `frontend/src/components/AutoAttachmentsField.tsx`: selector múltiple, staging, lista de adjuntos existentes, eliminar.
  - `frontend/src/components/InspectionStatusBadge.tsx`: badge reutilizable (detalle y listado).
  - `frontend/src/pages/AutosPage.tsx`: indicador de inspección en cada card (P1/REQ-010).
  - `frontend/src/api/index.ts`: métodos nuevos y helper multipart.
  - `frontend/src/types/index.ts`: tipos nuevos.
  - `frontend/public/i18n/es.json` y `en.json`: claves nuevas.
- **Dependencias**: ninguna librería nueva; `CardMenu`, `Select`, `DeleteModal`, `Icon`, `useCurrencyFormatStore` existentes.

### 4.3 Modelo de datos

#### `auto_maintenances`

```sql
CREATE TABLE IF NOT EXISTS auto_maintenances (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    auto_id INTEGER NOT NULL,
    performed_at TEXT NOT NULL,              -- YYYY-MM-DD
    mileage INTEGER NOT NULL CHECK (mileage >= 0),
    institution_id INTEGER NOT NULL,         -- categoría workshop
    description TEXT NOT NULL DEFAULT '',
    currency_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (auto_id) REFERENCES autos(id) ON DELETE CASCADE,
    FOREIGN KEY (institution_id) REFERENCES institutions(id),
    FOREIGN KEY (currency_id) REFERENCES currencies(id)
);

CREATE INDEX IF NOT EXISTS idx_auto_maintenances_auto_date
    ON auto_maintenances(auto_id, performed_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_auto_maintenances_institution
    ON auto_maintenances(institution_id);
```

#### `auto_maintenance_items`

```sql
CREATE TABLE IF NOT EXISTS auto_maintenance_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    maintenance_id INTEGER NOT NULL,
    item_type TEXT NOT NULL CHECK (item_type IN ('labor', 'supplies')),
    description TEXT NOT NULL,
    amount REAL NOT NULL DEFAULT 0 CHECK (amount >= 0),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (maintenance_id) REFERENCES auto_maintenances(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_auto_maintenance_items_maintenance
    ON auto_maintenance_items(maintenance_id, item_type, id);
```

#### `auto_inspections`

```sql
CREATE TABLE IF NOT EXISTS auto_inspections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    auto_id INTEGER NOT NULL,
    performed_at TEXT NOT NULL,              -- YYYY-MM-DD
    inspection_year INTEGER NOT NULL,        -- año del ciclo que cumple
    institution_id INTEGER NOT NULL,         -- categoría mechanical_inspection
    result TEXT NOT NULL CHECK (result IN ('approved', 'rejected')),
    notes TEXT NOT NULL DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (auto_id) REFERENCES autos(id) ON DELETE CASCADE,
    FOREIGN KEY (institution_id) REFERENCES institutions(id)
);

CREATE INDEX IF NOT EXISTS idx_auto_inspections_auto_year
    ON auto_inspections(auto_id, inspection_year DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_auto_inspections_institution
    ON auto_inspections(institution_id);
```

#### `auto_attachments`

```sql
CREATE TABLE IF NOT EXISTS auto_attachments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    auto_id INTEGER NOT NULL,
    owner_type TEXT NOT NULL CHECK (owner_type IN ('maintenance', 'inspection')),
    owner_id INTEGER NOT NULL,
    file_name TEXT NOT NULL,                 -- nombre original, sanitizado
    size_bytes INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (auto_id) REFERENCES autos(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_auto_attachments_owner
    ON auto_attachments(owner_type, owner_id, id);
CREATE INDEX IF NOT EXISTS idx_auto_attachments_auto
    ON auto_attachments(auto_id);
```

Rutas de archivo (no se guardan en la DB, se derivan):

```
{DATA_DIR}/autos/{auto_id}/maintenances/{maintenance_id}/{attachment_id}.pdf
{DATA_DIR}/autos/{auto_id}/inspections/{inspection_id}/{attachment_id}.pdf
```

#### Categorías de institución

```sql
INSERT INTO institution_categories (key, name, description, icon_key)
VALUES
    ('workshop', 'Taller Mecánico',
     'Talleres de mantenimiento y reparación vehicular', 'wrench'),
    ('mechanical_inspection', 'Inspección Mecánica',
     'Centros de inspección técnica vehicular', 'vehicle')
ON CONFLICT(key) DO NOTHING;
```

`down.sql`:

1. `UPDATE institutions SET category_id = NULL WHERE category_id IN (SELECT id FROM institution_categories WHERE key IN ('workshop','mechanical_inspection'));`
2. `DELETE FROM institution_categories WHERE key IN ('workshop','mechanical_inspection');`
3. `DROP TABLE IF EXISTS auto_attachments;`
4. `DROP TABLE IF EXISTS auto_inspections;`
5. `DROP TABLE IF EXISTS auto_maintenances;`

> Nota: el `down` elimina las categorías solo después de desasociar instituciones; no borra instituciones ni registros de autos. Los archivos PDF del volumen no se eliminan en el `down` (lo hace el servicio de aplicación).

### 4.4 APIs / Contratos

Todos los endpoints requieren sesión válida (401 sin sesión).

#### Mantenimientos

| Método | Ruta | Descripción |
|---|---|---|
| GET | `/api/autos/{id}/maintenances` | Lista mantenimientos del auto, con conceptos y adjuntos |
| POST | `/api/autos/{id}/maintenances` | Crea mantenimiento (información básica) |
| GET | `/api/autos/{id}/maintenances/{maintenance_id}` | Detalle del mantenimiento |
| PUT | `/api/autos/{id}/maintenances/{maintenance_id}` | Actualiza la información básica |
| DELETE | `/api/autos/{id}/maintenances/{maintenance_id}` | Elimina mantenimiento + conceptos + adjuntos |
| POST | `/api/autos/{id}/maintenances/{maintenance_id}/items` | Agrega una línea de mano de obra o insumos |
| PUT | `/api/autos/{id}/maintenances/{maintenance_id}/items/{item_id}` | Actualiza una línea |
| DELETE | `/api/autos/{id}/maintenances/{maintenance_id}/items/{item_id}` | Elimina una línea |
| POST | `/api/autos/{id}/maintenances/{maintenance_id}/attachments` | Sube un PDF (multipart `file`) |
| GET | `/api/autos/{id}/maintenances/{maintenance_id}/attachments/{attachment_id}` | Abre/descarga PDF (`?download=1`) |
| DELETE | `/api/autos/{id}/maintenances/{maintenance_id}/attachments/{attachment_id}` | Elimina un adjunto |

**POST/PUT request** (`application/json`):

```json
{
  "performed_at": "2026-10-15",
  "mileage": 42000,
  "institution_id": 12,
  "description": "Cambio de aceite y filtros",
  "currency_id": 1
}
```

**POST/PUT item request** (`application/json`):

```json
{
  "item_type": "labor",
  "description": "Pintura",
  "amount": 45.0
}
```

**Response 200/201**:

```json
{
  "id": 1,
  "auto_id": 1,
  "performed_at": "2026-10-15",
  "mileage": 42000,
  "institution_id": 12,
  "institution_name": "Taller Demo",
  "description": "Cambio de aceite y filtros",
  "labor_cost": 45.0,
  "supplies_cost": 80.0,
  "total_cost": 125.0,
  "currency_id": 1,
  "currency_code": "NIO",
  "items": [
    { "id": 1, "maintenance_id": 1, "item_type": "labor", "description": "Pintura", "amount": 45.0 },
    { "id": 2, "maintenance_id": 1, "item_type": "supplies", "description": "Pintura", "amount": 50.0 },
    { "id": 3, "maintenance_id": 1, "item_type": "supplies", "description": "Tornillos", "amount": 30.0 }
  ],
  "created_at": "2026-10-15T10:00:00Z",
  "updated_at": "2026-10-15T10:00:00Z",
  "attachments": [
    {
      "id": 7,
      "owner_type": "maintenance",
      "owner_id": 1,
      "file_name": "factura-demo.pdf",
      "size_bytes": 12345,
      "created_at": "2026-10-15T10:01:00Z"
    }
  ]
}
```

**GET lista** devuelve un array de los objetos anteriores, ordenado por `performed_at DESC, id DESC`, más `latest_mileage` y totales por moneda en un objeto:

```json
{
  "items": [ /* ... */ ],
  "latest_mileage": 42000,
  "totals_by_currency": [
    { "currency_id": 1, "currency_code": "NIO", "total": 125.0 }
  ]
}
```

#### Inspecciones

| Método | Ruta | Descripción |
|---|---|---|
| GET | `/api/autos/{id}/inspections` | Lista inspecciones + cronograma |
| POST | `/api/autos/{id}/inspections` | Crea inspección |
| PUT | `/api/autos/{id}/inspections/{inspection_id}` | Actualiza inspección |
| DELETE | `/api/autos/{id}/inspections/{inspection_id}` | Elimina inspección + adjuntos |
| POST | `/api/autos/{id}/inspections/{inspection_id}/attachments` | Sube un PDF (multipart `file`) |
| GET | `/api/autos/{id}/inspections/{inspection_id}/attachments/{attachment_id}` | Abre/descarga PDF (`?download=1`) |
| DELETE | `/api/autos/{id}/inspections/{inspection_id}/attachments/{attachment_id}` | Elimina un adjunto |
| GET | `/api/autos/inspection-alerts` | Estado de inspección de todos los autos (para el listado) |

**POST/PUT request**:

```json
{
  "performed_at": "2028-01-10",
  "inspection_year": 2028,
  "institution_id": 15,
  "result": "approved",
  "notes": "Sin observaciones"
}
```

**Response 200/201**:

```json
{
  "id": 3,
  "auto_id": 1,
  "performed_at": "2028-01-10",
  "inspection_year": 2028,
  "institution_id": 15,
  "institution_name": "Centro Demo",
  "result": "approved",
  "notes": "Sin observaciones",
  "created_at": "2028-01-10T09:00:00Z",
  "updated_at": "2028-01-10T09:00:00Z",
  "attachments": []
}
```

**GET `/api/autos/{id}/inspections` response**:

```json
{
  "items": [ /* inspecciones */ ],
  "schedule": {
    "first_due_year": 2028,
    "next_due_year": 2029,
    "next_due_date": "2029-01-01",
    "status": "ok",
    "days_until_due": 420,
    "approved_years": [2028],
    "rejected_years": []
  }
}
```

`status`: `overdue`, `due_soon` (0–60 días) u `ok`. Si no hay años pendientes dentro del rango evaluado, `next_due_year` es el siguiente año del calendario.

**GET `/api/autos/inspection-alerts` response**:

```json
[
  {
    "auto_id": 1,
    "next_due_year": 2029,
    "next_due_date": "2029-01-01",
    "status": "ok",
    "days_until_due": 420
  }
]
```

#### Adjuntos

**POST** (multipart/form-data):

- Campo: `file` (un solo archivo).
- Extensiones: `.pdf` (case-insensitive).
- Tamaño: 1 byte a 10 MB.
- Firma: primeros 5 bytes `%PDF-`.
- Respuesta 201: metadato del adjunto (mismo shape que el array `attachments`).
- Errores: `400 invalid_file`, `413 request_too_large`, `409 attachment_limit`, `404 not_found`.

**GET**: `Content-Type: application/pdf`; `Content-Disposition: inline; filename="<original>"` (o `attachment` con `?download=1`); `X-Content-Type-Options: nosniff`; `http.ServeFile`.

**DELETE**: `200 {"message":"Adjunto eliminado"}`.

#### Errores comunes

```json
{
  "error": "invalid_request",
  "message": "el taller es requerido"
}
```

Códigos: `invalid_id`, `invalid_request`, `not_found`, `invalid_file`, `request_too_large`, `attachment_limit`, `institution_category`, `internal_error`.

### 4.5 Dependencias

- **Internas**: `AutoService`/`AutoStorage` (existencia y año del auto), `InstitutionStorage` (categoría), `CurrencyStorage` (código de moneda), `authMiddleware`, `DeleteModal`, `Select`, `CardMenu`, `useCurrencyFormatStore`.
- **Externas**: ninguna nueva. Solo standard library de Go y React/Tailwind ya instalados.

---

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado un auto existente, cuando se crea un mantenimiento con fecha, kilometraje, taller, mano de obra, insumos y moneda válidos, entonces responde `201` y el registro aparece en la sección.
- [ ] CA-002: Si falta `mileage`, `institution_id`, la moneda o algún monto es negativo, la API responde `400` con un mensaje claro y no inserta nada.
- [ ] CA-003: El formulario exige una institución de categoría `workshop`; el nombre del taller mostrado y guardado se obtiene de esa institución, sin campo de texto editable.
- [ ] CA-004: El total mostrado es la suma de las líneas de mano de obra e insumos, con el formato de moneda configurado.
- [ ] CA-005: Se pueden subir al menos 3 PDFs al mismo mantenimiento, listarlos, abrirlos y eliminarlos individualmente.
- [ ] CA-006: Un archivo que no es PDF, uno de 0 bytes, uno mayor a 10 MB o uno sin firma `%PDF-` se rechaza y no deja archivo en disco.
- [ ] CA-007: Superar 20 adjuntos en un registro devuelve `409` y no escribe el archivo.
- [ ] CA-008: Para un auto `year = 2026` sin inspecciones, el cronograma indica `first_due_year = 2028`, `next_due_date = 2028-01-01` y el estado correspondiente a la fecha actual.
- [ ] CA-009: Al registrar una inspección `approved` para `inspection_year = 2028`, la próxima inspección pasa a `2029-01-01`.
- [ ] CA-010: Una inspección `rejected` para 2028 **no** avanza el cronograma; el año 2028 sigue pendiente y el estado refleja la fecha.
- [ ] CA-011: Una inspección con `inspection_year < auto.year + 2` o con `inspection_year > performed_at.Year()+1` se rechaza con `400`.
- [ ] CA-012: Una inspección con una institución que no es de categoría `mechanical_inspection` se rechaza en backend y no aparece en el selector del frontend.
- [ ] CA-013: El detalle del auto muestra estado (`overdue`/`due_soon`/`ok`), año y fecha de la próxima inspección; el listado de autos muestra el indicador compacto.
- [ ] CA-014: Al eliminar un mantenimiento o una inspección se eliminan sus adjuntos de la DB y del disco.
- [ ] CA-015: Al eliminar un auto, se eliminan sus mantenimientos, inspecciones, adjuntos y el directorio `{DATA_DIR}/autos/{auto_id}` (si el borrado del auto es exitoso).
- [ ] CA-016: Las categorías `workshop` y `mechanical_inspection` existen después de migrar y se muestran en el formulario de instituciones.
- [ ] CA-017: Todos los endpoints nuevos responden `401` sin sesión y `404` cuando el auto, registro o adjunto no existe o no pertenece al auto de la ruta.
- [ ] CA-018: Las migraciones `0042`, `0043` y `0044` suben, bajan y vuelven a subir sin errores, y el `down` de `0042` no falla si hay instituciones asociadas. `0043` migra un taller libre a una institución antes de eliminar `workshop_name`; `0044` migra los escalares a líneas antes de eliminarlos.
- [ ] CA-019: Los formatos de fecha `YYYY-MM-DD` se validan; una fecha inválida o futura para un registro realizado se rechaza.
- [ ] CA-020: La UI usa `Select` para institución, moneda, año y resultado; `npm run check:ui` pasa.
- [ ] CA-021: Las claves nuevas existen en `es.json` y `en.json` y se sirven desde `/i18n/es.json` tras el build.
- [ ] CA-DARK: Los inputs, selects y textareas de los modales usan `bg-card`/`text-text`/`text-text-secondary`; texto y placeholders son legibles en darkmode.
- [ ] CA-BACK: No se crean páginas nuevas; el detalle del auto conserva su flecha atrás registrada en `BACK_ROUTES`. Si finalmente se agregan rutas, se registran en `BACK_ROUTES` y no se usan links "← Título".
- [ ] CA-RESP: Los modales y secciones se ven correctamente en móvil (375 px) y desktop, sin overflow horizontal.
- [ ] CA-022: El detalle del auto muestra tres tabs (Seguros, Mantenimientos, Inspecciones), el tab activo se conserva en `?tab=`, solo se renderiza el panel activo y el tab de inspecciones refleja el estado vencida/próxima.
- [ ] CA-023: Desde una inspección ya creada se puede usar la acción visible **Adjuntar PDF**, subir uno o más archivos y ver el contador actualizado al cerrar; el mismo comportamiento aplica a los mantenimientos.
- [ ] CA-024: Cada fila de inspección/mantenimiento tiene una acción visible **Editar**. El banner de inspección con estado vencido/próximo muestra **Registrar inspección de {año}** y abre el modal con ese año preseleccionado; crear la inspección pendiente recalcula el cronograma.
- [ ] CA-025: Los modales de crear/editar mantenimiento e inspección muestran tabs **Información** y **Facturas**. Facturas lista los PDFs existentes, permite subir varios y eliminar, y muestra el contador; guardar desde Facturas sin los datos obligatorios vuelve a Información con el error visible.
- [ ] CA-026: `auto_maintenances` no tiene columna `workshop_name` y `institution_id` es `NOT NULL`. El formulario solo muestra el selector de taller; la API rechaza un mantenimiento sin institución. La migración `0043` migra talleres libres existentes a instituciones antes de eliminar la columna.
- [ ] CA-027: Un mantenimiento puede crearse con la información básica y luego agregar líneas de `auto_maintenance_items` de tipo `labor`/`supplies`, cada una con descripción y valor. La API de líneas rechaza descripciones vacías, tipos inválidos y valores negativos. Los totales por tipo y el total general se calculan sumando las líneas. La migración `0044` convierte los escalares existentes en líneas “Mano de obra” e “Insumos”.
- [ ] CA-028: El menú de la fila muestra **Ver** y **Eliminar** (sin **Editar**). **Ver** abre `/autos/:id/maintenances/:maintenanceId`. El detalle usa tabs **Información**, **Mano de obra**, **Insumos** y **Facturas**, con el tab activo en `?tab=` y contadores; solo se renderiza el panel activo. Información permite editar/eliminar el mantenimiento; Mano de obra e Insumos permiten agregar/editar/eliminar líneas; Facturas permite subir/eliminar PDFs. El modal de crear/editar solo contiene la información básica y la moneda. El header muestra la flecha atrás hacia `/autos/:id?tab=mantenimientos`.
- [ ] CA-029: Los modales de mantenimiento, concepto e inspección no se cierran al hacer click fuera, usan overlay `z-[60]` y tienen X/Cancelar. `DeleteModal` usa `z-[70]`, enfoca el input y confirma con Enter cuando el texto es `confirmo`.

### 5.2 No funcionales

- [ ] CA-NF-001: Subir un PDF de 10 MB no incrementa el RSS del server en más de ~20 MB.
- [ ] CA-NF-002: Listar 500 mantenimientos con 2.000 adjuntos responde en menos de 150 ms en el iHost.
- [ ] CA-NF-003: No se agregan dependencias Go ni npm.
- [ ] CA-NF-004: `go test ./...` y `npm run build` pasan.
- [ ] CA-NF-005: No se commitean PDFs, DBs ni datos reales; los tests usan PDFs sintéticos y datos `Demo`.

### 5.3 Testing

- **Unit tests**:
  - `BuildInspectionSchedule` con casos: año futuro, sin inspecciones, aprobadas consecutivas, año rechazado, inspección adelantada, fecha exacta de vencimiento, 60/61 días.
  - Validaciones de mantenimiento (mileage, montos, taller, moneda, fechas).
  - Validaciones de inspección (categoría, año mínimo/máximo, resultado).
- **Integration tests**:
  - Storage CRUD + listados con adjuntos agrupados.
  - Handlers multipart: PDF válido, no PDF, 0 bytes, >10 MB, límite de 20, descarga inline/attachment, 401.
  - Migraciones `0042`, `0043` y `0044` up/down/up, incluida la migración de un taller libre a institución y de los escalares a líneas.
- **E2E/manual**:
  - Auto 2026 → crear inspección 2028 → verificar próximo 2029.
  - Inspección rechazada → verificar que no avanza.
  - Subir 3 PDFs, abrir, descargar y eliminar.
  - Darkmode y móvil.
- **Carga/Performance**: no se requiere benchmark formal; medir el listado con datos sintéticos.

---

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migraciones `0042` (tablas, índices, categorías), `0043` (referencia de institución) y `0044` (líneas de detalle) + modelos + tests de migración | 0.5 día | Ninguna |
| 2 | Storage y services (validaciones, cronograma, adjuntos) + tests unitarios | 1.5 días | Fase 1 |
| 3 | Handlers, rutas y wiring (JSON + multipart) + tests de API | 1 día | Fase 2 |
| 4 | Frontend (secciones, modales, adjuntos, badge, API client, tipos) | 1.5 días | Fase 3 |
| 5 | i18n es/en, darkmode, responsive, `check:ui` y build | 0.5 día | Fase 4 |
| 6 | QA manual con copia de la DB de producción, ajustes y documentación | 0.5–1 día | Fase 5 |

### 6.2 Milestones

1. **MVP**: tablas + backend CRUD + adjuntos + cronograma + UI básica en el detalle. Sin indicador en el listado ni totales por moneda.
2. **V1.0**: MVP + indicador en listado + totales + edición + i18n + darkmode/responsive + tests completos.

---

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Crecimiento de disco por PDFs en el iHost | Alta | Medio | Límite 10 MB/archivo y 20/registro; mostrar tamaño total; documentar que el backup del volumen incluye PDFs; evaluar purge manual en una spec futura |
| Archivos huérfanos si falla la limpieza | Media | Bajo | Borrar fila y archivo en el mismo flujo; tests de eliminación; log de errores de `os.Remove` sin fallar la request |
| Instituciones existentes sin la nueva categoría | Alta | Bajo | Estado vacío con acción "Crear institución"; texto de ayuda; no auto-asignar categorías |
| Regla de inspección mal interpretada (año modelo vs año de compra) | Media | Alto | Documentar `auto.year + 2`; tests de borde; mostrar el año de vencimiento y el año pendiente en la UI |
| Inspección rechazada que no avanza el cronograma | Baja | Medio | Test explícito CA-010; badge de vencida si el año sigue pendiente |
| Upload multipart con memoria alta | Media | Alto | `MaxBytesReader`, temporal en disco, `io.Copy` con `LimitReader`; un archivo por request |
| Cambio de año del auto deja inspecciones inválidas | Baja | Medio | Advertencia en UI; no borrar datos; permitir corregir el año de la inspección |
| Confusión con la categoría `maintenance` existente (hogar) | Media | Bajo | Claves nuevas y nombres explícitos; el selector filtra por clave |
| PDFs con nombre malicioso/path traversal | Baja | Alto | Ignorar el nombre para la ruta; usar `{attachment_id}.pdf`; `filepath.Base` solo para el nombre original mostrado |
| Migración `down` falla por FKs de instituciones | Media | Medio | `UPDATE institutions SET category_id = NULL` antes de borrar categorías; test de migración |
| Conflictos multi-sesión en el worktree | Baja | Alto | Trabajar solo en `../p40la-ihost-spec-124`; main de solo lectura (SPEC-066) |

---

## 8. Notas y Referencias

- SPEC-024: Módulo CRUD de Autos.
- SPEC-025: Página Show de Autos con Seguros.
- SPEC-028: Extensión de campos para autos y pólizas.
- SPEC-043: Acción Pagar en facturas con comprobante y referencia.
- SPEC-051: Emails y generación mensual de pensión (precedente de upload).
- SPEC-054: Módulo Deudas.
- SPEC-060: Fix global de inputs en darkmode.
- SPEC-063: Flecha atrás en páginas de detalle (`BACK_ROUTES`).
- SPEC-091: Pólizas y ciclos de servicio.
- SPEC-117: Menú de acciones en el detalle del auto.
- SPEC-118: Campos opcionales y color predefinido.
- SPEC-122: Comprobantes PDF en pagos de deudas (precedente de validación `%PDF-` y escritura atómica).
- SPEC-123: Contratos PDF para deudas y servicios (precedente de almacenamiento por entidad).
- `docs/project-rules.md` §4 (UI), §6 (DB), §8 (datos privados y secretos).
- `AGENTS.md` (worktrees, validación con copia de producción, reglas de UI).

**Fuera de alcance de esta spec**:
- Alertas de mantenimiento por kilometraje (no hay lectura automática de odómetro).
- OCR o extracción de datos de los PDFs.
- Notificaciones email/Telegram de inspecciones (P2, REQ-028).
- Gestión de talleres/centros como entidades propias: se reutilizan instituciones y categorías.
- Cambios al esquema de `autos` o `auto_services`.

---

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-07 | opencode | Creación inicial de la especificación (mantenimientos por kilometraje e inspecciones mecánicas anuales con PDFs). |
| 2026-11-07 | opencode | Inicio de implementación: estado `in_progress`. |
| 2026-11-08 | opencode | Implementación completa: migración `0042`, backend (storage/services/API), frontend (secciones, modales, adjuntos y badge), i18n es/en, tests Go y build del frontend. Migración validada sobre copia de producción: 3 autos y 16 instituciones; `up`/`down`/`up` idempotente e `integrity_check = ok`. |
| 2026-11-08 | opencode | Cambio iterativo solicitado por el usuario durante la evaluación: organizar el detalle del auto en tabs Seguros, Mantenimientos e Inspecciones (`?tab=`), mostrando solo el panel activo. Se agrega REQ-032 y CA-022. |
| 2026-11-08 | opencode | Cambio iterativo: acción visible “Adjuntar PDF” por fila con modal dedicado para gestionar comprobantes después de crear el mantenimiento/inspección. Se agrega REQ-033 y CA-023. |
| 2026-11-08 | opencode | Cambio iterativo: acciones visibles “Editar” en cada fila y CTA “Registrar inspección de {año}” en el banner de estado para crear el ciclo pendiente. Se agrega REQ-034 y CA-024. |
| 2026-11-08 | opencode | Cambio iterativo: modales de crear/editar mantenimiento e inspección con tabs Información y Facturas. Se agrega REQ-035 y CA-025. |
| 2026-11-08 | opencode | Cambio iterativo: eliminar la duplicación del nombre del taller. Con institución seleccionada se usa su nombre y se oculta el campo libre; sin institución se exige el nombre manual. Se agrega REQ-036 y CA-026. |
| 2026-11-08 | opencode | Corrección de modelo: el taller es solo la referencia a la institución. Migración `0043` elimina la columna `workshop_name`, migra talleres libres a instituciones y deja `institution_id` `NOT NULL`. Se actualizan REQ-001/REQ-002/REQ-036 y CA-002/CA-003/CA-026. |
| 2026-11-08 | opencode | Mano de obra e insumos como líneas de detalle en `auto_maintenance_items` (`item_type`, `description`, `amount`). Migración `0044`, totales calculados. Se agrega REQ-037 y CA-027; la moneda sigue en Información. |
| 2026-11-08 | opencode | Página de detalle de mantenimiento con edición básica, CRUD de líneas y archivos. El modal de crear/editar queda solo con información básica y se agrega **Ver** al menú. Se agregan REQ-038/REQ-039 y CA-028. |
| 2026-11-08 | opencode | Detalle de mantenimiento organizado en tabs Información, Mano de obra, Insumos y Facturas (`?tab=`), con contadores y solo el panel activo. Se actualiza REQ-038 y CA-028. |
| 2026-11-08 | opencode | El menú de la fila de mantenimiento queda con **Ver** y **Eliminar**; **Editar** se mueve al detalle. Se actualizan REQ-038 y CA-028. |
| 2026-11-08 | opencode | Aplicación de las reglas de modales de `main`/SPEC-122: sin cierre por click fuera, overlays `z-[60]`, cierre explícito y `DeleteModal` `z-[70]` con foco y Enter. Se agrega REQ-040 y CA-029. |
