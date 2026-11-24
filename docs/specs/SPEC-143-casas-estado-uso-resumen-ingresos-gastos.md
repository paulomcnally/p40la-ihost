---
title: "Casas: estado de uso y resumen de ingresos y gastos"
id: "SPEC-143"
status: "pending_release"
author: "opencode"
created: "2026-11-23"
updated: "2026-11-23"
github_issue: 146
---

# Casas: estado de uso y resumen de ingresos y gastos

**ID**: SPEC-143
**Estado**: pending_release
**Autor**: opencode
**Creado**: 2026-11-23
**Actualizado**: 2026-11-23

---

## 1. Resumen Ejecutivo

Hoy una casa no tiene estado propio: solo existe el flag de publicación (`listing_enabled`) y un estado de alquiler derivado de reservas y contratos. El usuario no puede ver de un vistazo si una propiedad está **En alquiler**, **Reservada para alquiler**, **Alquilada**, **Habitada** (donde vive él) o **Sin uso** (por ejemplo un terreno que no cumple ninguna función pero genera gastos).

Además, la pantalla principal de casas no muestra el resultado económico de cada propiedad. El usuario necesita comparar **ingresos vs gastos** por casa y en total, con un criterio claro de qué suma cada lado, para detectar propiedades que solo generan gasto.

El resultado esperado es: un uso editable por casa (con estados de alquiler derivados automáticamente), un badge de estado en cada card y en el detalle, y una tarjeta de resumen mensual en `/home` con ingresos, gastos, neto y pendientes **por moneda**, sin conversión.

**iHost**: no se agregan dependencias ni procesos. El resumen se calcula con pocas consultas agregadas sobre SQLite, usando índices existentes y dos índices nuevos. La respuesta se limita a las casas del usuario (decenas, no miles).

---

## 2. Cumplimiento de reglas

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Solo Go + `net/http` + SQLite; React/Tailwind en el build. Sin dependencias nuevas | `CA-ARCH-STACK` |
| `ARCH-LAYERS` | Handler → service → storage en el módulo nuevo; `homes` mantiene su capa legacy | `CA-ARCH-LAYERS` |
| `ARCH-MODULES` | Nuevo módulo `realestate.portfolio` con `module/model/storage/service/handlers` y registro en `all/register.go` | `CA-ARCH-MODULES` |
| `ARCH-GO` | `gofmt`, nombres en inglés, errores explícitos | `CA-ARCH-GO` |
| `ARCH-IHOST` | Consultas agregadas, índices, sin Node en runtime, sin servicios externos | `CA-ARCH-IHOST` |
| `UI-SELECT` | El selector de uso usa `Select`; no hay `<select>` nativo | `CA-UI-SELECT` |
| `UI-INPUT-TOKENS` | El formulario usa `bg-card`, `text-text` y tokens del tema | `CA-UI-INPUT-TOKENS` |
| `UI-DARKMODE` | Badge, tarjeta de resumen y formulario verificados en darkmode | `CA-UI-DARKMODE` |
| `UI-TOGGLE` | N/A: el filtro usa filas de selección múltiple (excepción de la regla), no un booleano persistido | — |
| `UI-LIST-PATTERN` | El estado se muestra como badge en las cards; no se agregan formularios inline | `CA-UI-LIST-PATTERN` |
| `UI-EDIT-LABEL` | La acción de edición sigue usando `app.edit` | `CA-UI-EDIT-LABEL` |
| `UI-I18N` | Claves nuevas en `frontend/public/i18n/{es,en}.json` con paridad | `CA-UI-I18N` |
| `UI-STYLE` | Badge y tarjeta con `rounded-ios`/`shadow-ios`; colores de estado con tokens del tema | `CA-UI-STYLE` |
| `UI-NUMBER-INPUT` | N/A: no se agregan campos numéricos editables | — |
| `UI-PHONE-INPUT` | N/A: no hay teléfonos | — |
| `UX-BACK` | N/A: no se agregan rutas nuevas; `/home` es listado raíz | — |
| `UX-DETAIL-READONLY` | El detalle deja de editar la publicación: el uso se edita en `/home/edit/:id` y el detalle solo muestra el badge | `CA-UX-DETAIL-READONLY` |
| `UX-DETAIL-ACTIONS` | Editar/Eliminar siguen en el menú de 3 puntos | `CA-UX-DETAIL-ACTIONS` |
| `UX-MODAL` | El modal de filtro no cierra por click fuera, tiene X y Listo visibles, usa `z-[60]` y bloquea el fondo | `CA-UX-MODAL` |
| `UX-PREREQ` | Backend y frontend bloquean contratos/reservas sobre casas no disponibles y el cambio de uso con alquiler activo | `CA-UX-PREREQ` |
| `UX-EMPTY` | Sin casas se mantiene `EmptyCard`; sin movimientos el resumen muestra ceros | `CA-UX-EMPTY` |
| `UX-LOADING` | Skeleton del resumen y spinner existente del listado | `CA-UX-LOADING` |
| `UX-FEEDBACK` | Toast de éxito/error al guardar uso y al consultar el resumen | `CA-UX-FEEDBACK` |
| `UX-FORMS` | Una acción Guardar y una Cancelar; validación junto al campo | `CA-UX-FORMS` |
| `DATA-SQLITE` | WAL sin cambios; SQL crudo con `database/sql` | `CA-DATA-SQLITE` |
| `DATA-MIGRATIONS` | Migraciones `0058` y `0059` con `up` y `down` reversibles | `CA-DATA-MIGRATIONS` |
| `DATA-NO-DROP` | No se toca `data/app.db`; pruebas en `/tmp` | `CA-DATA-NO-DROP` |
| `DATA-PRODCOPY` | La validación del resumen y del backfill se hace sobre copia en `/tmp` | `CA-DATA-PRODCOPY` |
| `SEC-NO-PII` | Spec, tests y ejemplos sin datos reales | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Ejemplos con `Casa Demo`, `demo-account-001`, montos ficticios | `CA-SEC-SYNTHETIC` |
| `SEC-COMMIT` | Revisión de `git diff --cached` antes de commit | `CA-SEC-COMMIT` |
| `TEST-LOCAL` | `go build ./...`, `go test ./...`, `check:ui`, `check:i18n`, `check:numbers` | `CA-TEST-LOCAL` |
| `TEST-SERVER` | `./scripts/check-server.sh` antes de avisar al usuario | `CA-TEST-SERVER` |
| `TEST-KILLALL` | Reinicios con `killall p40la-ihost` | `CA-TEST-KILLALL` |
| `TEST-USER-FLOW` | Prueba manual del usuario antes de `pending_release` | `CA-TEST-USER-FLOW` |
| `TEST-CHECKLIST` | Checklist de `docs/rules/60-testing.md` completo | `CA-TEST-CHECKLIST` |
| `DELIV-WORKTREE` | Trabajo en `../p40la-ihost-spec-143` (`feature/SPEC-143`) | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | Merge y push solo con confirmación explícita | `CA-DELIV-CONFIRM` |
| `DELIV-COMMIT` | Commits `SPEC-143: ...` | `CA-DELIV-COMMIT` |
| `DELIV-HOOKS` | Hooks instalados, sin `--no-verify` | `CA-DELIV-HOOKS` |

---

## 3. Requerimientos

### 3.1 Funcionales (P0)

1. **REQ-001 — Uso de la casa editable.** Toda casa tiene un campo `usage` con tres valores: `rental` («En alquiler»), `inhabited` («Habitada») y `unused` («Sin uso»). Se crea con `rental` por defecto y se edita en el formulario de casa.
2. **REQ-002 — Estado derivado.** El estado mostrado se calcula sin guardarse:
   - `usage = unused` → **Sin uso**.
   - `usage = inhabited` → **Habitada**.
   - `usage = rental` + contrato `active` → **Alquilada**.
   - `usage = rental` + reserva `active` vigente → **Reservada para alquiler**.
   - `usage = rental` sin contrato ni reserva → **En alquiler**.
3. **REQ-003 — Estado inicial.** Una casa nueva queda **En alquiler** salvo que el formulario indique `Habitada` o `Sin uso`.
4. **REQ-004 — Badge de estado.** El listado `/home` muestra el badge de estado en cada card y el detalle lo muestra en el header (solo lectura).
5. **REQ-005 — Bloqueo de cambio de uso.** No se puede pasar a `Habitada`/`Sin uso` si la casa tiene contrato `active` o reserva `active` vigente. Backend responde `409 home_has_active_rental`; el frontend muestra el error junto al campo y un toast.
6. **REQ-006 — Bloqueo de alquiler sobre casas no disponibles.** No se puede crear contrato ni reserva para una casa con `usage != rental`. Backend `409 home_not_available`; el selector de casa en los formularios excluye o deshabilita esas casas con explicación.
7. **REQ-007 — Resumen mensual.** `GET /api/rental/portfolio/summary?year=&month=` devuelve, por casa y en totales: ingresos, gastos, neto, pendiente por cobrar y pendiente por pagar, **agrupados por moneda**.
8. **REQ-008 — Tarjeta de resumen en `/home`.** Arriba del listado se muestra el resumen del mes con navegación ‹ mes ›, totales por moneda y conteo de estados. Por defecto, mes actual.
9. **REQ-009 — Neto por card.** Cada card muestra el neto del mes seleccionado por moneda (ingresos − gastos). Las casas `Habitada`/`Sin uso` muestran solo gasto y neto negativo.
10. **REQ-010 — Criterio de ingresos y gastos.** Se aplica la tabla de la sección 5.4 sin ambigüedad, y el resumen explica el criterio en un texto de ayuda.
11. **REQ-011 — Sin conversión de moneda.** Los importes no se convierten entre monedas: se muestran y suman por moneda. No se calcula un neto único cross-currency.
12. **REQ-017 — Colores por estado.** Cada estado tiene un color propio y distinguible, visible en el badge del listado, del detalle y del filtro, y legible en darkmode:
    - **En alquiler** → verde (`success`).
    - **Reservada para alquiler** → ámbar (`warning`).
    - **Alquilada** → azul (`primary`).
    - **Habitada** → violeta (`accent`, token nuevo).
    - **Sin uso** → gris (`neutral`).
13. **REQ-018 — Filtro por estado con modal.** En `/home`, un control tipo dropdown («Filtrar») abre un modal con los cinco estados; el usuario elige uno o varios y la lista de casas se filtra. El modal:
    - no se cierra al hacer click fuera;
    - se cierra con la **X** o **Listo**, siempre visibles;
    - muestra el conteo por estado y permite «Todos»;
    - exige al menos un estado seleccionado (Listo deshabilitado si no hay ninguno);
    - usa la capa `z-[60]` y filas de selección múltiple (sin checkbox nativo).
    - El filtro afecta solo las cards del listado; el resumen financiero sigue siendo global.
14. **REQ-019 — Tab «Finanzas» en el detalle de la casa.** El detalle (`/home/:id`) suma un tab que muestra el detalle económico de esa casa: últimos 12 meses, mes seleccionable y movimientos.
15. **REQ-020 — Gráficos.** El tab muestra:
    - barras de ingresos vs gastos por mes (12 meses terminando en el mes seleccionado);
    - dona de composición de gastos del mes (servicios, reparaciones, comisión);
    - selector de moneda cuando hay más de una; nunca se convierten monedas.
16. **REQ-021 — Detalle de movimientos.** Tabla/lista del mes con fecha, tipo, descripción, estado e importes, con enlace a la entidad:
    - cobro → contrato;
    - seña → apartado;
    - factura → facturas del servicio;
    - reparación → detalle de la reparación;
    - comisión → contrato.
17. **REQ-022 — Comisión única.** La comisión de la inmobiliaria se devenga **una sola vez, en el mes de inicio del contrato**: porcentaje de una renta (`commission_percent × rent_amount`) si es porcentual, o `commission_amount` si es fija. Se muestra como costo del mes y se excluyen contratos borrador/cancelados.

### 3.2 Funcionales (P1)

1. **REQ-012 — Detalle del resumen.** Desglose por casa al tocar la tarjeta de resumen (misma data, sin endpoint nuevo).
2. **REQ-013 — Comisión de inmobiliaria.** Mostrar la comisión del contrato como gasto informativo del propietario en el resumen del mes.
3. **REQ-014 — Persistir el mes elegido** en el query string (`?year=&month=`) para conservarlo al navegar.

### 3.3 Funcionales (P2)

1. **REQ-015 — Rango de meses** (este mes, últimos 3, año) con comparación.
2. **REQ-016 — Exportar el resumen del mes** a CSV sintético.

### 3.4 No funcionales

- **Rendimiento**: el resumen se calcula con ≤ 6 consultas agregadas; objetivo < 150 ms con 50 casas y 5 años de datos en iHost.
- **Memoria**: respuesta acotada a casas + monedas (no se devuelven movimientos individuales).
- **Seguridad**: endpoint autenticado como el resto; sin PII en logs.
- **Almacenamiento**: solo dos migraciones; sin tablas nuevas.
- **iHost**: sin dependencias nuevas ni cambios en el build de frontend.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

- `homes` hoy: `id, name, address, sort_order, listing_enabled, deleted_at, created_at, updated_at`.
- `listing_enabled` se usa solo en `internal/storage/home.go`, `internal/modules/realestate/rental/*` y la UI de `HomeShowPage`.
- El estado de alquiler ya es derivado (SPEC-140): `available/reserved/rented/unlisted`.
- Gastos por casa ya existen en dos fuentes: `services` + `bills` (con `home_id`) y `home_maintenances` + `home_maintenance_items` (SPEC-141).
- Ingresos: `rental_payments` (cobros) y `rental_reservations.amount` (señas).
- `currencies` no tiene tipo de cambio; no hay conversión global en el proyecto.
- El usuario confirmó: todo inmueble puede alquilarse (no se agrega tipo de inmueble), estado derivado, `Habitada` manual, bloqueo de cambio con alquiler activo, resumen devengado con pendientes, por moneda.

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| Estado guardado y editable | Simple de leer | Se desincroniza con contratos/reservas; doble fuente de verdad | ❌ |
| Estado 100% derivado | Nunca miente | No puede representar «Habitada» | ❌ |
| `usage` manual + estados de alquiler derivados | Una sola fuente para lo manual y lo automático | Requiere migración y retirar `listing_enabled` | ✅ |
| Agregar `property_type` (casa/terreno) | Describe el inmueble | El usuario confirmó que todo puede alquilarse; no aporta al estado | ❌ |
| Mantener `listing_enabled` junto a `usage` | Menos cambios | Dos flags contradictorios (`usage=rental` + no publicado) | ❌ |
| Resumen dentro de `realestate.rental` | Menos archivos | SQL de gastos de otros dominios dentro del módulo de alquiler | ❌ |
| Módulo `realestate.portfolio` de solo lectura | Respeta ARCH-MODULES y separa responsabilidades | Un módulo nuevo pequeño | ✅ |
| Convertir monedas con tasa manual | Neto único | No hay tabla de tasas; agrega mantenimiento y errores | ❌ |
| Desglose por moneda | Correcto y sin supuestos | No hay un único número | ✅ |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001 — `usage` reemplaza a `listing_enabled`.**
- **Contexto**: «Sin uso» ya significa «no se ofrece en alquiler». Mantener ambos flags permite estados contradictorios.
- **Decisión**: `homes.usage TEXT NOT NULL DEFAULT 'rental' CHECK (usage IN ('rental','inhabited','unused'))`. Se elimina `listing_enabled`; el backfill marca `unused` donde `listing_enabled = 0`.
- **Consecuencias**: se retira `PUT /api/rental/homes/{id}/listing` y el toggle del detalle. El «no publicado» pasa a ser «Sin uso». La migración `down` recrea `listing_enabled` desde `usage`.

**ADR-002 — Estado derivado, nunca persistido.**
- **Contexto**: contratos y reservas ya determinan la disponibilidad.
- **Decisión**: `realestate.rental` expone una derivación única (`deriveState`) usada por el detalle y por el resumen.
- **Consecuencias**: cero desincronización; los tests cubren la matriz `usage × contrato × reserva`.

**ADR-003 — «Habitada» es manual y excluyente del alquiler.**
- **Contexto**: no se puede derivar que el dueño vive allí.
- **Decisión**: `usage = inhabited` bloquea contratos/reservas nuevos y el cambio se bloquea si ya hay alquiler activo.
- **Consecuencias**: los datos nunca quedan contradictorios; se requiere cerrar el contrato antes de mudarse.

**ADR-004 — Resumen devengado con pendientes, por moneda.**
- **Contexto**: el usuario quiere ver gasto vs ingreso del mes y también deudas.
- **Decisión**: ingresos y gastos del mes por fecha de negocio; pendientes informados aparte; sin conversión.
- **Consecuencias**: el neto es por moneda; el «por cobrar» es un saldo acumulado al cierre del mes (no solo del mes).

**ADR-005 — Módulo `realestate.portfolio` de solo lectura.**
- **Contexto**: el resumen cruza casas, alquileres, servicios y mantenimientos.
- **Decisión**: módulo propio que consulta la DB (sin importar `storage` de otros módulos) y recibe los estados vía el contrato `modules.HomeStateProvider` que implementa `realestate.rental`.
- **Consecuencias**: una sola derivación de estado; el módulo no escribe datos.

---

## 5. Diseño Técnico

### 5.1 Diagrama

```
HomesPage ──GET /api/homes──> HomeService ──> HomeStorage ──> homes(usage)
    │
    └──GET /api/rental/portfolio/summary──> portfolio.Service
                                                │
                    ┌───────────────────────────┼───────────────────────────┐
                    v                           v                           v
        modules.HomeStateProvider      bills + services           rental_payments
        (realestate.rental)            home_maintenances          rental_charges
                    │                  home_maintenance_items     rental_reservations
                    v
              homes + leases + reservations
```

### 5.2 Componentes

#### 5.2.1 Migraciones
- **Ubicación**: `migrations/0058_homes_usage.{up,down}.sql`, `migrations/0059_portfolio_indexes.{up,down}.sql`.
- `0058 up`:
  ```sql
  ALTER TABLE homes ADD COLUMN usage TEXT NOT NULL DEFAULT 'rental'
      CHECK (usage IN ('rental', 'inhabited', 'unused'));
  UPDATE homes SET usage = 'unused' WHERE listing_enabled = 0;
  ALTER TABLE homes DROP COLUMN listing_enabled;
  ```
- `0058 down` (reversible, sin pérdida de la clasificación):
  ```sql
  ALTER TABLE homes ADD COLUMN listing_enabled INTEGER NOT NULL DEFAULT 1;
  UPDATE homes SET listing_enabled = CASE WHEN usage = 'rental' THEN 1 ELSE 0 END;
  ALTER TABLE homes DROP COLUMN usage;
  ```
- `0059 up`:
  ```sql
  CREATE INDEX IF NOT EXISTS idx_rental_payments_paid_at ON rental_payments(paid_at);
  CREATE INDEX IF NOT EXISTS idx_rental_charges_due_date ON rental_charges(due_date);
  CREATE INDEX IF NOT EXISTS idx_home_maintenances_performed_at ON home_maintenances(performed_at);
  ```
- `0059 down`: `DROP INDEX` de los tres.
- **Nota**: `0059` se aplica **antes** de `0058 down`; `0058 down` no depende de `listing_enabled`.

#### 5.2.2 Hogares (capa legacy)
- `internal/models/home.go`: reemplazar `ListingEnabled` por `Usage string`.
- `internal/storage/home.go`: `SELECT`/`scan` con `usage`; `Create(name, address, usage)`; `Update(id, name, address, usage)`.
- `internal/services/home.go`:
  - Validar `usage ∈ {rental, inhabited, unused}`; default `rental` en creación.
  - `SetRentalGuard(HomeRentalGuard)` con `HasActiveRental(ctx, homeID) (bool, error)`.
  - Si cambia el uso a `inhabited`/`unused` y el guard responde `true` → `ErrHomeHasActiveRental`.
- `internal/api/home_handlers.go`: `homeRequest` incluye `usage`; mapear `ErrHomeHasActiveRental` a `409 home_has_active_rental` y validación a `400 invalid_usage`.

#### 5.2.3 Módulo `realestate.rental` (ajustes)
- `Storage.GetHomeListing`: `SELECT name, COALESCE(address,''), usage` → `HomeRentalStatus{Usage}`.
- `HomeRentalStatus`: `Usage` reemplaza `ListingEnabled`.
- Nueva derivación única:
  ```go
  func deriveState(usage string, hasLease, hasReservation bool) string {
      switch usage {
      case UsageInhabited: return StateInhabited
      case UsageUnused:    return StateUnused
      }
      if hasLease { return StateRented }
      if hasReservation { return StateReserved }
      return StateListed
  }
  ```
- `Service.GetHomeStatus` usa `deriveState`.
- `Service.ListHomeStates(ctx)` → `[]modules.HomeState` (una consulta con `EXISTS` de contrato/reserva).
- `Storage.ListHomeStates(ctx)` con:
  ```sql
  SELECT h.id, h.name, COALESCE(h.address,''), h.usage,
    EXISTS(SELECT 1 FROM rental_leases l
           WHERE l.home_id=h.id AND l.status='active' AND l.deleted_at IS NULL),
    EXISTS(SELECT 1 FROM rental_reservations r
           WHERE r.home_id=h.id AND r.status='active' AND r.expires_at >= ?)
  FROM homes h WHERE h.deleted_at IS NULL
  ORDER BY h.sort_order, h.name
  ```
- `Service.HasActiveRental(ctx, homeID)` usa el mismo `EXISTS` (contrato `active` o reserva `active` vigente).
- `Module` implementa `HomeStates` y `HasActiveRental` delegando al service.
- `prepareLease`/`prepareReservation`: si `usage != rental` → `ErrHomeNotAvailable` (409).
- Se elimina `SetListingEnabled` (storage, service, handler y ruta).

#### 5.2.4 Módulo `realestate.portfolio` (nuevo)
- **ID**: `realestate.portfolio`, versión `1.0.0`, tipo `core`.
- **Archivos**: `module.go`, `model.go`, `storage.go`, `service.go`, `handlers.go`, `service_test.go`.
- **Constructor**: `New(db *sql.DB, states modules.HomeStateProvider)`.
- **Responsabilidad**: calcular el resumen mensual; no escribe.
- **Registro**: en `internal/modules/all/register.go`, después de `rental.New(...)`:
  ```go
  rentalMod := rental.New(db, email, settings, dataDir)
  registry.Register(rentalMod)
  registry.Register(portfolio.New(db, rentalMod))
  ```

#### 5.2.5 Frontend
- `frontend/src/pages/HomesPage.tsx`: carga hogares + resumen; tarjeta de resumen arriba; badge y neto en cada card.
- `frontend/src/pages/HomeFormPage.tsx`: `Select` de uso (3 opciones) + texto de ayuda; carga `home.usage` al editar; error de campo si el backend responde `home_has_active_rental`.
- `frontend/src/pages/HomeShowPage.tsx`: se elimina el toggle de publicación; el badge de estado usa el nuevo `listing_status`; el uso se edita desde «Editar».
- `frontend/src/components/home/PortfolioSummary.tsx` (nuevo): navegación ‹ mes ›, totales por moneda, pendientes y conteo de estados.
- `frontend/src/components/home/HomeStateFilterModal.tsx` (nuevo): filtro por estado con selección múltiple.
- `frontend/src/components/home/HomeFinancesTab.tsx` (nuevo): serie de 12 meses, KPIs, composición de gastos y movimientos del mes.
- `frontend/src/components/home/HomeFinanceCharts.tsx` (nuevo): barras y dona en SVG, sin dependencias.
- `frontend/src/components/home/HomeStateBadge.tsx` (nuevo): badge único reutilizado en listado y detalle.
- `frontend/src/pages/LeaseFormPage.tsx` y `ReservationFormPage.tsx`: el selector de casa solo ofrece `usage === 'rental'`; si no hay ninguna, muestra el estado vacío con enlace a editar la casa.
- `frontend/src/api/rental.ts` y `frontend/src/api/homes.ts`: `portfolio.summary(year, month)` y `usage` en create/update.
- `frontend/src/types`: `Home.usage`, `HomeUsage`, `HomeState`, tipos del resumen.

### 5.3 Modelo de datos

```
homes (existente + nueva columna)
- usage: TEXT NOT NULL DEFAULT 'rental'
         CHECK (usage IN ('rental','inhabited','unused'))
```

| usage | contrato activo | reserva activa | Estado mostrado |
|---|---|---|---|
| `rental` | sí | — | `rented` — **Alquilada** |
| `rental` | no | sí | `reserved` — **Reservada para alquiler** |
| `rental` | no | no | `listed` — **En alquiler** |
| `inhabited` | no | no | `inhabited` — **Habitada** |
| `unused` | no | no | `unused` — **Sin uso** |

- Los estados `inhabited`/`unused` con contrato o reserva activa son **inválidos** y se bloquean al editar.
- `listing_status` mantiene el nombre del campo por compatibilidad del contrato JSON, pero sus valores pasan a `listed/reserved/rented/inhabited/unused` (se retiran `available/unlisted`).

### 5.4 Criterio de ingresos, gastos y pendientes

| Concepto | Fuente | Filtro | Moneda |
|---|---|---|---|
| **Ingreso: cobros** | `rental_payments` | `paid_at` en `[inicio, fin)` del mes | `payments.currency` |
| **Ingreso: señas** | `rental_reservations.amount` | `starts_at` en el mes y `status IN ('active','forfeited')`; se excluyen `converted` (ya cuenta el depósito), `cancelled`, `expired` | `reservations.currency` |
| **Gasto: servicios** | `bills` ⋈ `services` | `bills.year/month` = mes, `bills.deleted_at IS NULL`, `services.deleted_at IS NULL` | `services.currency_id → currencies.code` |
| **Gasto: reparaciones** | `home_maintenance_items` ⋈ `home_maintenances` | `performed_at` en el mes (mano de obra + insumos) | `items.currency_id → currencies.code` |
| **Pendiente por cobrar** | `rental_charges` − `rental_payments` | saldo acumulado al último día del mes: cargos `status != 'void'` con `due_date < fin` − pagos con `paid_at < fin`; se informa si es > 0 | por moneda |
| **Pendiente por pagar** | `bills` | facturas del mes con `status = 'pending'` | moneda del servicio |

- **Neto** = ingresos − gastos, por moneda. Los pendientes no se suman al neto.
- **Habitada / Sin uso**: ingresos 0; sus gastos sí se suman.
- **No cuentan como ingreso**: facturas emitidas no cobradas (van a «por cobrar»), depósitos devueltos.
- **No cuentan como gasto**: la renta que paga el inquilino, comisiones de la agencia (P1).
- Los montos se devuelven con dos decimales tal como están en SQLite (`REAL`); el frontend los formatea con `Money`.

### 5.4.1 Colores de estado

| Estado | Token | Uso |
|---|---|---|
| `listed` — En alquiler | `success` (verde) | Badge y filtro |
| `reserved` — Reservada para alquiler | `warning` (ámbar) | Badge y filtro |
| `rented` — Alquilada | `primary` (azul) | Badge y filtro |
| `inhabited` — Habitada | `accent` (violeta, token nuevo) | Badge y filtro |
| `unused` — Sin uso | `neutral` (gris) | Badge y filtro |

- Se agrega `--color-accent` a `frontend/src/index.css` (claro y dark) y `accent` a `frontend/tailwind.config`.
- `Badge` (`frontend/src/components/rental/RentalUI.tsx`) incorpora el tono `accent`.
- `HomeStateBadge` agrega un punto de color (`bg-current`) para reforzar la distinción.

### 5.4.2 Filtro por estado

- Control «Filtrar» en el header de `/home`, junto a `CreateMenu`; muestra un contador cuando hay filtro activo.
- `HomeStateFilterModal` (nuevo, `frontend/src/components/home/`):
  - lista los 5 estados con su badge, conteo del mes y fila seleccionable (`aria-pressed`);
  - «Todos» selecciona los 5; «Listo» aplica; la X descarta;
  - si no hay ningún estado seleccionado, «Listo» se deshabilita y se muestra una ayuda;
  - overlay sin cierre por click, `z-[60]`, cierre visible siempre.
- Estado inicial del filtro: todos los estados. Se mantiene en memoria de la página (sin persistencia).
- El filtro solo se aplica a las cards; el resumen y sus totales no cambian.
- Si el resumen aún no cargó, se muestran todas las casas y el filtro se aplica al llegar los estados.

### 5.4.3 Tab «Finanzas» de la casa

- **UI**: `HomeFinancesTab` dentro de `HomeShowPage`, con tab propio (`?tab=finances`) y selector de mes ‹ ›. El estado del tab se conserva en el query string como los demás.
- **API**: `GET /api/rental/portfolio/homes/{id}/finances?year=YYYY&month=M` (autenticada). Valida el período (`400 invalid_period`) y la casa (`404 home_not_found`).
- **Respuesta**:
  ```json
  {
    "home_id": 1,
    "home_name": "Casa Demo",
    "usage": "rental",
    "state": "rented",
    "year": 2026,
    "month": 11,
    "currencies": ["NIO", "USD"],
    "months": [
      { "year": 2026, "month": 10,
        "income": [{ "currency": "USD", "amount": 500 }],
        "expenses": [{ "currency": "USD", "amount": 320 }],
        "net": [{ "currency": "USD", "amount": 180 }] }
    ],
    "income": [], "expenses": [], "net": [],
    "pending_income": [], "pending_expenses": [],
    "breakdown": [
      { "kind": "services", "amounts": [{ "currency": "NIO", "amount": 1200 }] },
      { "kind": "maintenance", "amounts": [{ "currency": "USD", "amount": 300 }] },
      { "kind": "commission", "amounts": [{ "currency": "USD", "amount": 250 }] }
    ],
    "items": [
      { "type": "payment", "date": "2026-11-06", "description": "Cobro de alquiler",
        "detail": "Transferencia · demo-ref", "status": "paid",
        "amounts": [{ "currency": "USD", "amount": 300 }], "lease_id": 1 },
      { "type": "bill", "date": "2026-11-01", "description": "Luz Demo",
        "detail": "Proveedor Demo", "status": "pending",
        "amounts": [{ "currency": "NIO", "amount": 1200 }], "bill_id": 3, "service_id": 2 }
    ]
  }
  ```
  - `months` contiene exactamente 12 entradas, de `mes-11` a `mes` (inclusive), con ceros omitidos.
  - `items` se ordena por fecha descendente; un mantenimiento con ítems en varias monedas devuelve un solo ítem con varios importes.
  - El frontend elige por defecto la moneda con mayor actividad del período y permite cambiarla con `Select`.
  - Los enlaces se construyen en el frontend con los IDs: `/leases/{lease_id}`, `/reservations/{id}`, `/services/bills/{service_id}`, `/home/{home_id}/maintenances/{id}`.
- **Gráficos** (SVG propio, sin dependencias):
  - `FinanceBars`: 12 meses, dos barras por mes (ingreso verde, gasto rojo), eje con máximo redondeado y `<title>` por barra.
  - `FinanceDonut`: composición de gastos del mes seleccionado; colores `primary` (servicios), `warning` (reparaciones) y `accent` (comisión), con leyenda e importes.
- **Cálculo**: reutiliza las fuentes de la sección 5.4 sobre una ventana de 12 meses, agrupando en Go por mes y moneda. La comisión se calcula por contrato con `start_date` dentro del mes (REQ-022) y no se repite.
- **iHost**: 5 consultas por apertura del tab (pagos, señas, facturas, mantenimientos y comisiones de la ventana); sin N+1 y sin nuevas dependencias.

### 5.5 APIs / Contratos

#### `GET /api/rental/portfolio/summary?year=YYYY&month=M`

- Auth: obligatoria (mismo middleware).
- Validación: `year` 2000–2100, `month` 1–12 → `400 invalid_period`.

**Response 200**:
```json
{
  "year": 2026,
  "month": 11,
  "homes": [
    {
      "home_id": 1,
      "name": "Casa Demo",
      "address": "Calle Demo 123",
      "usage": "rental",
      "state": "rented",
      "income": [{ "currency": "USD", "amount": 500.00 }],
      "expenses": [{ "currency": "NIO", "amount": 1200.00 }],
      "net": [
        { "currency": "NIO", "amount": -1200.00 },
        { "currency": "USD", "amount": 500.00 }
      ],
      "pending_income": [{ "currency": "USD", "amount": 250.00 }],
      "pending_expenses": []
    }
  ],
  "totals": {
    "income": [{ "currency": "USD", "amount": 500.00 }],
    "expenses": [{ "currency": "NIO", "amount": 1200.00 }],
    "net": [
      { "currency": "NIO", "amount": -1200.00 },
      { "currency": "USD", "amount": 500.00 }
    ],
    "pending_income": [{ "currency": "USD", "amount": 250.00 }],
    "pending_expenses": []
  },
  "state_counts": {
    "listed": 2,
    "reserved": 1,
    "rented": 3,
    "inhabited": 1,
    "unused": 2
  }
}
```

- Las listas `income/expenses/net/pending_*` se ordenan por `currency` ascendente y omiten montos 0.
- `homes` respeta `sort_order` del listado.

#### `POST /api/homes` y `PUT /api/homes/{id}`

**Request** (nuevo campo `usage`, opcional; default `rental`):
```json
{ "name": "Casa Demo", "address": "Calle Demo 123", "usage": "rental" }
```

**Errores nuevos**:
```json
{ "error": "invalid_usage", "message": "Uso inválido" }
{ "error": "home_has_active_rental", "message": "Cerrá el contrato o la reserva antes de cambiar el uso" }
```

#### Creación de contrato/reserva

- `POST /api/rental/leases` y `POST /api/rental/reservations` sobre una casa con `usage != rental` → `409 home_not_available`.

#### Endpoint retirado

- `PUT /api/rental/homes/{id}/listing` deja de existir; el uso se cambia por `PUT /api/homes/{id}`.

### 5.6 Dependencias

- **Internas**: `homes` (legacy), `realestate.rental`, `home.maintenance` (tablas), `services`/`bills`, `currencies`.
- **Externas**: ninguna nueva.

---

## 6. Criterios de Aceptación

### 6.1 Funcionales y de estado

- [x] `CA-001`: Al crear una casa sin indicar uso, queda `usage=rental` y el badge muestra **En alquiler**.
- [x] `CA-002`: Una casa con reserva `active` vigente muestra **Reservada para alquiler**.
- [x] `CA-003`: Una casa con contrato `active` muestra **Alquilada** aunque además tenga reserva.
- [x] `CA-004`: Una casa con `usage=inhabited` muestra **Habitada** y nunca estados de alquiler.
- [x] `CA-005`: Una casa con `usage=unused` muestra **Sin uso** y nunca estados de alquiler.
- [x] `CA-006`: Cambiar a `inhabited`/`unused` con contrato `active` devuelve `409 home_has_active_rental` y no persiste el cambio.
- [x] `CA-007`: Cambiar a `inhabited`/`unused` con reserva `active` vigente devuelve `409 home_has_active_rental`.
- [x] `CA-008`: Crear contrato o reserva sobre una casa `inhabited`/`unused` devuelve `409 home_not_available`.
- [x] `CA-009`: El formulario de contrato/reserva no ofrece casas con `usage != rental` y explica por qué si no hay ninguna disponible.
- [x] `CA-010`: El resumen del mes suma cobros y señas como ingreso, y facturas + reparaciones como gasto, según la sección 5.4.
- [x] `CA-011`: Los importes se muestran por moneda y nunca se convierte ni se suma entre monedas distintas.
- [x] `CA-012`: El neto por casa y total es `ingresos − gastos` por moneda.
- [x] `CA-013`: «Por cobrar» refleja el saldo acumulado al cierre del mes y «por pagar» las facturas del mes pendientes.
- [x] `CA-014`: Casas `inhabited`/`unused` aportan gastos al total y cero ingresos.
- [x] `CA-015`: La navegación ‹ mes › cambia el resumen y los netos de las cards sin recargar la página.
- [x] `CA-016`: Sin movimientos en el mes, el resumen muestra ceros y no rompe.
- [x] `CA-017`: Los cinco estados se distinguen por color (verde, ámbar, azul, violeta, gris), con punto de color, y se leen bien en darkmode.
- [x] `CA-018`: El botón «Filtrar» abre un modal con los cinco estados; el modal no cierra por click fuera, tiene X y Listo visibles y usa `z-[60]`.
- [x] `CA-019`: El filtro permite seleccionar varios estados, muestra el conteo por estado, exige al menos uno y filtra las cards sin alterar el resumen.
- [x] `CA-020`: Al quitar el filtro («Todos») vuelven a mostrarse todas las casas.
- [x] `CA-021`: El detalle de la casa tiene un tab «Finanzas» que carga sin recargar la página y respeta el retorno del header.
- [x] `CA-022`: El gráfico de barras muestra 12 meses (ingresos vs gastos) en la moneda elegida y navega con el mes seleccionado.
- [x] `CA-023`: La dona muestra la composición de gastos del mes (servicios, reparaciones, comisión) en la moneda elegida.
- [x] `CA-024`: La lista de movimientos muestra cobros, señas, facturas, reparaciones y comisión del mes, con enlace a su detalle.
- [x] `CA-025`: La comisión aparece solo en el mes de inicio del contrato y no se repite en los meses siguientes.
- [x] `CA-026`: Si no hay movimientos o no hay gastos, se muestran estados vacíos claros; con más de una moneda hay selector y no se suman monedas distintas.

### 6.2 Reglas de UI/UX

- [x] `CA-ARCH-MODULES`: `realestate.portfolio` tiene los archivos obligatorios y se registra solo en `all/register.go`.
- [x] `CA-ARCH-LAYERS`: handlers sin SQL ni lógica; service sin SQL; storage sin HTTP.
- [x] `CA-ARCH-IHOST`: el resumen ejecuta ≤ 6 consultas y no agrega dependencias.
- [x] `CA-DATA-MIGRATIONS`: `0058` y `0059` tienen `up` y `down`; se probaron sobre copia en `/tmp`.
- [x] `CA-DATA-PRODCOPY`: la validación con datos reales se hizo sobre copia y se borró al terminar.
- [x] `CA-UI-SELECT`: el uso se elige con el componente `Select`; no hay `<select>` nativo.
- [x] `CA-UI-TOGGLE`: N/A justificado en la tabla de cumplimiento (filtro con filas de selección múltiple).
- [x] `CA-UI-STYLE`: los colores de estado usan tokens del tema y el badge conserva `rounded-ios`.
- [x] `CA-UI-INPUT-TOKENS`: formulario, badge y tarjeta usan tokens del tema, sin colores hardcodeados.
- [x] `CA-UI-DARKMODE`: badge, resumen y formulario legibles en darkmode.
- [x] `CA-UI-LIST-PATTERN`: el estado y el neto viven en cards; no hay formularios inline.
- [x] `CA-UI-EDIT-LABEL`: la acción de edición sigue rotulada «Editar» (`app.edit`).
- [x] `CA-UI-I18N`: claves nuevas en `es.json` y `en.json`, con paridad y sin apuntar a objetos padre.
- [x] `CA-UX-DETAIL-READONLY`: el detalle no edita el uso ni la publicación; solo muestra el badge.
- [x] `CA-UX-DETAIL-ACTIONS`: Editar/Eliminar permanecen en el menú de 3 puntos.
- [x] `CA-UX-PREREQ`: los bloqueos se validan en backend y en frontend.
- [x] `CA-UX-MODAL`: el modal de filtro no cierra por click fuera, el overlay bloquea el fondo, X y Listo siempre visibles y `z-[60]`.
- [x] `CA-UX-EMPTY`: sin casas se mantiene `EmptyCard`; sin movimientos se muestran ceros.
- [x] `CA-UX-LOADING`: el resumen muestra skeleton mientras carga; Guardar se deshabilita al enviar.
- [x] `CA-UX-FEEDBACK`: éxito y error de guardado muestran toast; el error de uso se muestra junto al campo.
- [x] `CA-UX-FORMS`: Guardar/Cancelar únicos; validación bloquea el guardado.
- [x] `CA-SEC-NO-PII`: sin datos reales en spec, tests, commits ni capturas.
- [x] `CA-SEC-SYNTHETIC`: ejemplos con `Casa Demo`, `demo@example.com`, montos ficticios.

### 6.3 Testing

- **Unit tests**:
  - `deriveState` para las 6 combinaciones válidas y las 2 inválidas bloqueadas.
  - `HomeService.Update` con guard: bloquea `inhabited`/`unused` con alquiler activo; permite el resto.
  - `portfolio.Service.Summary`: ingresos (cobros + señas, excluye `converted`), gastos (bills + mantenimientos), pendientes, monedas separadas, límites de mes.
- **Integration tests**: `go test ./...` con SQLite en `/tmp` y datos sintéticos; migraciones `up`/`down`/`up`.
- **E2E manual**: crear casa, cambiar uso, crear reserva → estado, activar contrato → estado, intentar cambiar uso → error, ver resumen del mes.
- **Performance**: medir el endpoint con 50 casas y 3 años de datos sintéticos.

---

## 7. Plan de Implementación

| Fase | Descripción | Dependencias |
|---|---|---|
| 1 | Migraciones `0058`/`0059` + `models.Home.Usage` + storage/service/handler de hogares | — |
| 2 | Ajustes en `realestate.rental`: `deriveState`, `HomeStates`, `HasActiveRental`, bloqueos, retiro de `SetListingEnabled` | 1 |
| 3 | Módulo `realestate.portfolio` + contrato `HomeStateProvider` + registro | 2 |
| 4 | Frontend: tipos, API, `Select` de uso, badge con colores, filtro por estado, tarjeta de resumen, neto por card | 3 |
| 5 | i18n `es`/`en` + guardrails `check:ui`, `check:i18n`, `check:numbers` | 4 |
| 6 | Tests, build, validación sobre copia de producción y prueba manual | 5 |

**MVP**: fases 1–5 con P0 (estado + resumen del mes).
**V1.0**: P1 (desglose y query string).

---

## 8. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| Retirar `listing_enabled` rompe consumidores internos | Media | Medio | `grep` completo, migración `down` reversible, tests de listado/detalle |
| Backfill incorrecto de casas no publicadas | Baja | Medio | Validar sobre copia de producción: comparar conteos antes/después |
| Neto confuso al haber NIO y USD | Alta | Medio | Nunca sumar entre monedas; mostrar una línea por moneda y texto de ayuda |
| «Por cobrar» acumulado se lee como del mes | Media | Bajo | Etiqueta «Saldo por cobrar al cierre» y ayuda contextual |
| Doble conteo seña + depósito | Media | Medio | Excluir reservas `converted` del ingreso |
| Rendimiento con muchos mantenimientos | Baja | Medio | Índices `0059`, `GROUP BY` por casa/moneda, sin N+1 |
| Datos reales en pruebas | Baja | Alto | Copia en `/tmp`, datos sintéticos, `SEC-COMMIT` |
| Cambio de uso bloqueado genera fricción | Media | Bajo | Mensaje claro y enlace al contrato/reserva activa |

---

## 9. Notas y Referencias

- Reglas: `docs/rules/00-index.md`, `10-architecture.md`, `20-ui.md`, `30-ux.md`, `40-data.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.
- Specs relacionadas: SPEC-140 (alquiler), SPEC-141/142 (reparaciones y cards), SPEC-063 (retorno), SPEC-122/126 (modales), SPEC-135 (detalle informativo).
- Tablas fuente: `homes`, `services`, `bills`, `home_maintenances`, `home_maintenance_items`, `rental_leases`, `rental_reservations`, `rental_charges`, `rental_payments`, `currencies`.
- El detalle del criterio económico vive en la sección 5.4 y debe reflejarse en la ayuda de la UI.

### 9.1 Validación de implementación

- `go build ./...` y `go test ./...`: OK, con tests nuevos de `realestate.portfolio`, del guard de uso y de reversibilidad de migraciones.
- Frontend: `npm run build` con `check:ui`, `check:i18n` y `check:numbers`: OK.
- API local con DB sintética: estados `listed`/`rented`/`inhabited`, bloqueo `409 home_has_active_rental` y `409 home_not_available`, resumen por moneda y pendientes: OK.
- Copia de producción en `/tmp` (0050 → 0059): 4 casas conservadas, columna `usage` creada, `listing_enabled` retirada, sin pérdida de filas. Sin datos reales copiados al repo.
- Colores por estado y filtro con modal: `HomeStateBadge` con 5 tonos (incluye `accent`) y `HomeStateFilterModal`; el build de frontend con `check:ui` e `check:i18n` pasa.
- Tab Finanzas validado por API: 12 meses, movimientos del mes, breakdown por categoría y comisión solo en el mes de inicio del contrato.
- Prueba manual del usuario: aprobada; pidió el release de la spec.

---

## 10. Historial de Cambios

| Fecha | Autor | Descripción |
|---|---|---|
| 2026-11-23 | opencode | Creación inicial de la especificación (draft) con decisiones del usuario |
| 2026-11-23 | opencode | Implementación: migraciones, `usage`, estados derivados, módulo `realestate.portfolio`, resumen en `/home` y pruebas |
| 2026-11-23 | opencode | Colores por estado y filtro por estado con modal |
| 2026-11-23 | opencode | Tab Finanzas por casa: barras, dona, movimientos y comisión única al inicio |
| 2026-11-23 | opencode | Implementación completa y validada; estado pasa a `pending_release` |
