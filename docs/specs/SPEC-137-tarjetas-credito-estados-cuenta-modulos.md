---
title: "Tarjetas de crédito, estados de cuenta e importación BAC con módulos namespaced"
id: "SPEC-137"
status: "released"
author: "OpenCode"
created: "2026-11-18"
updated: "2026-11-18"
github_issue: 140
---

# Tarjetas de crédito, estados de cuenta e importación BAC con módulos namespaced

**ID**: SPEC-137  
**Estado**: released  
**Autor**: OpenCode  
**Creado**: 2026-11-18  
**Actualizado**: 2026-11-18

---

## 1. Resumen Ejecutivo

Hoy el sistema modela bancos y cuentas (SPEC-133/134/135), pero no tiene un lugar para las **tarjetas de crédito**, que no son cuentas: tienen línea de crédito, ciclo de facturación, fecha de corte, pago mínimo y pago de contado, intereses y dos monedas en una misma línea (córdobas y dólares en BAC Nicaragua). El usuario pidió que, así como un banco tiene cuentas, también tenga tarjetas de crédito, y poder cargar el estado de cuenta mensual (PDF de BAC) para ver y editar sus movimientos.

Esta spec define:

1. Un módulo **core** `bank.creditcard` con el CRUD de tarjetas, estados de cuenta y movimientos.
2. Un módulo de proveedor `bank.bac.ni.creditcard.statement.import` que parsea el PDF "Transacciones del periodo" de BAC Nicaragua, con **preview obligatorio** antes de persistir.
3. La primera aplicación concreta de la arquitectura de **módulos con namespace canónico** (`ARCH-MODULES`), inspirada en los plugins de integración (`assa.nicaragua`, `claro.nicaragua`): la carpeta espeja el ID.

El resultado esperado: desde el detalle de un banco, tab **Tarjetas**, se crea una tarjeta, se importa su estado de cuenta, se revisan los movimientos con sus montos local/USD y se concilia el saldo impreso contra la suma de movimientos.

### Consideraciones de iHost

- **Sin dependencias nuevas**: el parseo usa `github.com/ledongthuc/pdf`, ya presente en `go.mod`.
- El PDF se procesa **en memoria** y se descarta; no se persiste ni se commitea (PII y datos financieros).
- Límite de subida 5 MB, una importación a la vez, pico de memoria objetivo < 32 MB.
- SQLite sigue siendo la única persistencia; tres tablas nuevas y una migración reversible.

---

## 2. Cumplimiento de reglas

Índice: `docs/rules/00-index.md`. La regla `ARCH-MODULES` se agrega en esta misma spec (es requisito previo a la implementación).

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Go + `net/http` + SQLite; sin dependencias nuevas; frontend React/Tailwind | — |
| `ARCH-FOLDERS` | Se agrega `internal/modules/` al árbol; el resto del layout no cambia | `CA-ARCH-MODULES` |
| `ARCH-LAYERS` | Dentro de cada módulo: `handlers.go` → `service.go` → `storage.go` → `model.go`, archivos separados | `CA-ARCH-LAYERS` |
| `ARCH-MODULES` | IDs `bank.creditcard` y `bank.bac.ni.creditcard.statement.import`; carpeta = ID; `module.go` + registry + `all/register.go` | `CA-ARCH-MODULES` |
| `ARCH-GO` | Nombres en inglés, `gofmt`, errores explícitos, `log/slog` | — |
| `ARCH-IHOST` | Sin deps nuevas, PDF en memoria, límite 5 MB, sin Node en runtime | `CA-NF-001` |
| `ARCH-CONFIG` | N/A: no cambia `opencode.json` ni la carga de instrucciones | — |
| `UI-SELECT` | Marca, moneda, categoría, color y tipo de movimiento usan `Select` (buscable cuando aplica); `check:ui` en el gate | `CA-UI-SELECT` |
| `UI-INPUT-TOKENS` | Inputs con `bg-card`/`text-text`; sin colores hardcodeados | `CA-UI-TOKENS` |
| `UI-DARKMODE` | Formularios, tab, listado de estados y preview de importación verificados en darkmode | `CA-UI-DARKMODE` |
| `UI-TOGGLE` | `active` de la tarjeta usa `Toggle` iOS, con `aria-label` | `CA-UI-TOGGLE` |
| `UI-LIST-PATTERN` | Tarjetas y estados de cuenta se listan en cards con `CardMenu`; sin formularios inline; `EmptyCard` si no hay datos | `CA-UI-LIST-PATTERN` |
| `UI-CREATE-MENU` | La creación de tarjeta/estado usa `CreateMenu` en el header del tab/sección | `CA-UI-CREATE-MENU` |
| `UI-EDIT-LABEL` | Todas las acciones de edición usan `app.edit` («Editar») | `CA-UI-EDIT-LABEL` |
| `UI-I18N` | Namespace `creditCards.*` en `frontend/public/i18n/{es,en}.json`; build y verificación por HTTP | `CA-UI-I18N` |
| `UI-GUARDRAIL` | `npm run check:ui` pasa; no hay `<select>` ni checkbox nativo | `CA-UI-SELECT` |
| `UX-BACK` | Rutas nuevas registradas en `BACK_ROUTES` (form → tarjeta, tarjeta → banco, estado → tarjeta); sin «← Título» en contenido | `CA-UX-BACK` |
| `UX-MODAL` | La importación es una **página** de 2 pasos, no un modal; los `DeleteModal` existentes mantienen overlay bloqueante y cierre visible | `CA-UX-MODAL` |
| `UX-MODAL-CONFIRM` | Borrar tarjeta/estado/movimiento usa `DeleteModal` con palabra `confirmo`, foco y Enter | `CA-UX-MODAL-CONFIRM` |
| `UX-DETAIL-ACTIONS` | Editar/Eliminar viven en el `CardMenu` del detalle; «Importar estado» es la acción primaria visible permitida | `CA-UX-DETAIL-ACTIONS` |
| `UX-DETAIL-READONLY` | Detalle de tarjeta y de estado en modo lectura; editar navega a `/edit/:id` | `CA-UX-DETAIL-READONLY` |
| `UX-PREREQ` | Tarjeta requiere banco existente (backend `bank_not_found` + frontend redirige); estado/movimiento requieren tarjeta/estado existentes | `CA-UX-PREREQ` |
| `UX-EMPTY` | Sin tarjetas, sin estados y sin movimientos: `EmptyCard` con acción | `CA-UX-EMPTY` |
| `UX-LOADING` | Skeleton/spinner en listados; botones deshabilitados durante parseo/guardado/importación | `CA-UX-LOADING` |
| `UX-FEEDBACK` | Toast de éxito/error; errores de validación junto al campo; warnings del parser visibles en el preview | `CA-UX-FEEDBACK` |
| `UX-FORMS` | Un Guardar + un Cancelar; validación bloquea; obligatorios marcados | `CA-UX-FORMS` |
| `DATA-SQLITE` | `journal_mode=WAL` ya activo; sin ORM; SQL crudo en `storage.go` | `CA-DATA-MIGRATIONS` |
| `DATA-MIGRATIONS` | `migrations/0048_create_credit_cards.up.sql` + `.down.sql`, reversible | `CA-DATA-MIGRATIONS` |
| `DATA-NO-DROP` | No se toca `data/app.db`; tests con DB temporal (`/tmp`) | `CA-DATA-PRODCOPY` |
| `DATA-PRODCOPY` | Validación contra copia de producción en `/tmp` antes de `pending_release` | `CA-DATA-PRODCOPY` |
| `DATA-PROD-WRITE` | N/A en esta spec: la migración se aplica por el arranque normal del server; no hay escritura manual a producción | — |
| `DATA-PROTECTED-SCHEMA` | N/A: no se modifica `debts`/`debt_bills` | — |
| `DATA-SSH` | N/A: no requiere acceso al iHost para implementar | — |
| `SEC-NO-PII` | Solo `last4`; prohibido PAN/CVV/PIN; el PDF no se persiste ni se cita; el titular no se importa del PDF | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Todos los ejemplos usan `6000-0000-0000-0000`, `Comercio Demo`, montos demo | `CA-SEC-SYNTHETIC` |
| `SEC-PRODCOPY` | Cualquier copia de DB para validar vive en `/tmp` y se borra | `CA-SEC-PRODCOPY` |
| `SEC-COMMIT` | `git diff --cached` revisado; sin PDFs, DBs, exports ni PII | `CA-SEC-COMMIT` |
| `SEC-HOOKS` | Hooks instalados; sin `--no-verify` | `CA-SEC-HOOKS` |
| `SEC-INCIDENT` | N/A: no se manejan secretos nuevos; protocolo vigente si ocurre | — |
| `SEC-LOGS` | Prohibido loguear contenido del PDF, líneas parseadas, PAN o nombre del titular | `CA-SEC-NO-PII` |
| `TEST-LOCAL` | `go build ./...`, `go test ./...`, `check:ui` y prueba manual | `CA-TEST-LOCAL` |
| `TEST-SERVER` | `./scripts/check-server.sh` antes de avisar al usuario | — |
| `TEST-KILLALL` | Si hay que reiniciar el server, `killall p40la-ihost` | — |
| `TEST-TESTUSER` | Endpoints probados con `./scripts/create-test-user.sh` y `/tmp/cookies.txt` | — |
| `TEST-USER-FLOW` | 🤖 Se pregunta antes de correr el server y antes de cerrar la spec | — |
| `TEST-ROBOT` | Preguntas de flujo con 🤖 | — |
| `TEST-CHECKLIST` | Checklist completo antes de `pending_release` | `CA-TEST-LOCAL` |
| `DELIV-WORKTREE` | Sesión en `/home/paulomcnally/github/p40la-ihost-spec-137` (rama `feature/SPEC-137`) | — |
| `DELIV-CONFIRM` | Merge y push solo con confirmación explícita | — |
| `DELIV-COMMIT` | Commits `SPEC-137: ...` | — |
| `DELIV-RELEASE` | Release con los 8 pasos; tracker y label en el release | — |
| `DELIV-CHECKRELEASE` | `/spec-check-release` antes de declarar terminado | — |
| `DELIV-HOOKS` | Hooks activos en el worktree | `CA-SEC-HOOKS` |
| `DELIV-DOCKER` | N/A: no cambia `Dockerfile`, base ni plataformas; sin deps nuevas | — |
| `DELIV-LABELS` | Issue con `spec/draft` → `spec/released`; sin label genérico | — |
| `DELIV-DEPLOY` | Deploy aprobado por el usuario; `/health` verificado | — |
| `DELIV-ESCALATION` | N/A al inicio; se activa si la spec se bloquea > 3 días | — |

---

## 3. Requerimientos

### 3.1 Funcionales (P0 — obligatorios)

**Tarjetas**

1. **REQ-001**: Un banco puede tener N tarjetas de crédito. El detalle del banco muestra tabs **Cuentas** y **Tarjetas** (`?tab=`, default `cuentas`).
2. **REQ-002**: CRUD de tarjeta con: alias, banco (fijo del contexto), marca (`visa`, `mastercard`, `amex`, `diners`, `other`), últimos 4 dígitos, titular, moneda principal, moneda secundaria opcional (USD), límite local, límite USD, día de corte (1–31), día límite de pago (1–31), % pago mínimo, tasa corriente anual, tasa moratoria anual, tasas USD, color, icono, activa, notas.
3. **REQ-003**: Validaciones: `last4` exactamente 4 dígitos; nunca aceptar PAN completo, CVV ni PIN; días 1–31; tasas ≥ 0; `% pago mínimo` 0–100; moneda secundaria distinta de la principal; alias único por banco (case-insensitive).
4. **REQ-004**: El detalle de la tarjeta es de solo lectura y muestra límites, **disponible derivado** (local y USD), días de corte/pago, tasas y notas.

**Estados de cuenta**

5. **REQ-005**: CRUD de estados de cuenta por tarjeta con: periodo (inicio opcional, fin = fecha de corte), fecha límite de pago mínimo, fecha límite de pago de contado, saldo anterior (local/USD), intereses corrientes (local/USD), pago mínimo (local/USD), pago de contado (local/USD), saldo al corte (local/USD), estado (`unpaid`, `partial`, `paid`), notas.
6. **REQ-006**: Un solo estado por tarjeta y fecha de corte (`UNIQUE(card_id, period_end)`); conflicto → `409 statement_exists`.
7. **REQ-007**: El estado de cuenta muestra totales impresos, totales derivados por tipo de movimiento y una **conciliación**: `saldo_anterior + cargos − abonos` vs `saldo_al_corte`, con la diferencia visible (tolerancia 0.005).

**Movimientos**

8. **REQ-008**: CRUD de movimientos por estado de cuenta (o ciclo abierto si `statement_id` es NULL): fecha, concepto, categoría (reusa `categories`), tipo, monto local, monto USD, moneda original, estado (`posted`, `pending`), nota.
9. **REQ-009**: Tipos: `purchase`, `payment`, `payment_bonus`, `cash_advance`, `cash_advance_fee`, `interest_current`, `interest_late`, `fee`, `tax`, `bonus`, `refund`, `adjustment`, `other`.
10. **REQ-010**: Signo: positivo = cargo (aumenta deuda), negativo = abono (reduce deuda). El servicio rechaza cargos negativos para `purchase`/`fee`/`interest_*` y abonos positivos para `payment`/`payment_bonus`/`refund`.
11. **REQ-011**: La lista de movimientos permite filtrar por tipo y buscar por concepto; muestra fecha, concepto, categoría, tipo y montos local/USD.

**Importación BAC**

12. **REQ-012**: Importar el PDF "Transacciones del periodo" de BAC Nicaragua en **dos pasos**: `preview` (parsea, no persiste) y `confirm` (persiste en una transacción SQL).
13. **REQ-013**: El preview devuelve encabezado, movimientos, warnings y `preview_hash`; el frontend muestra todo antes de confirmar. El preview es **solo lectura**: la categoría y el tipo se editan después de importar (evita alterar el hash y mantiene el confirm determinista).
14. **REQ-014**: El parser extrae: producto enmascarado (solo `last4`), fecha de corte, fechas límite, saldo anterior, pago mínimo, pago de contado, intereses corrientes y saldo al corte (local/USD), más las filas de movimientos (fecha, concepto, local, USD).
15. **REQ-015**: El parser ignora líneas informativas (`PRICECASH`, `DISPONIBLE`, `SU DISPONIBLE`, `ACCIONES GANADAS`, `SORTEO`, `SERIE`, `Página #`, ecos de producto/titular con 0.00/0.00).
16. **REQ-016**: Clasificación por concepto: `SU PAGO RECIBIDO` → `payment`; `BONIFICACION` → `payment_bonus`; `RETIRO DE EFEC.` → `cash_advance`; `COMISION ADELANTO` → `cash_advance_fee`; `INTERES` → `interest_current`; `COMISION`/`MEMBRESA`/`SEGURO` → `fee`; resto → `purchase`.
17. **REQ-017**: Si el `last4` del PDF no coincide con la tarjeta destino, el confirm se rechaza con `card_mismatch` (se permite override explícito solo en preview, nunca silencioso).
18. **REQ-018**: Reimportar el mismo estado (misma tarjeta + corte) devuelve `409 statement_exists`; no se duplican movimientos.
19. **REQ-019**: El PDF **no se guarda** en disco, DB ni volumen; se descarta al terminar el request.

**Módulos**

20. **REQ-020**: Registrar los módulos `bank.creditcard` y `bank.bac.ni.creditcard.statement.import` en un registry consultable (`GET /api/modules`), con ID, versión, tipo, proveedor, país y migraciones.
21. **REQ-021**: Los estados importados guardan `source_module_id` y `import_hash` para trazabilidad y deduplicación.

### 3.2 Funcionales (P1 — importantes)

1. **REQ-022**: Reglas de categorización por concepto (ej. `GOOGLE*` → categoría «Suscripciones»), editables.
2. **REQ-023**: Importar estados de otros bancos mediante un importador CSV genérico (`bank.generic.creditcard.statement.import`).
3. **REQ-024**: Exportar el estado de cuenta a CSV (sin datos de tarjeta más allá de `last4`).
4. **REQ-025**: Recordatorio de vencimiento en Home cuando `min_payment_due_date` está a ≤ 5 días y el saldo > 0.

### 3.3 Funcionales (P2 — deseables)

1. **REQ-026**: Cuotas/diferidos (`installments_total`, `installment_number`) y Tasa Cero.
2. **REQ-027**: Vínculo opcional con una cuenta de presupuesto (`accounts.type='credit_card'`).
3. **REQ-028**: Puntos/PriceCash y proyección de intereses.

### 3.4 No funcionales

- **Rendimiento**: preview de un estado de 1 página < 2 s; listados < 300 ms con 1.000 movimientos.
- **Memoria**: pico < 32 MB por importación; sin caché de PDFs; una importación concurrente.
- **Almacenamiento**: ~1 KB por movimiento; 5 años de una tarjeta ≈ 300 KB.
- **Seguridad**: endpoints con `authMiddleware`; upload limitado a 5 MB; solo `application/pdf` + firma `%PDF-`; sin PAN/CVV/PIN; sin logs de contenido.
- **iHost**: sin binarios externos, sin cgo, sin nuevas dependencias Go.
- **Disponibilidad**: `/health` sin cambios.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

**Open source (modelo de tarjetas)**

| Proyecto | Tarjeta como… | Estado de cuenta | Límite | Pago mínimo | Intereses | Multi-moneda |
|---|---|---|---|---|---|---|
| Firefly III | cuenta de activo con rol credit card | no existe (mes calendario) | virtual balance | no | no calcula | cuenta por moneda |
| Actual Budget | cuenta con saldo negativo | no existe | no | no | no | limitado |
| Maybe Finance | `CreditCard` sobre `Account` | no existe | `available_credit` | `minimum_payment` | `apr` | entries + FX |
| GnuCash | pasivo a corto plazo | conciliación manual | no | no | manual | cuenta por moneda |
| KMyMoney | tipo de cuenta `CreditCard` | conciliación manual | no | no | no | moneda por cuenta |

Conclusión: ninguno modela el **estado de cuenta** como entidad, y ninguno cubre el caso BAC (dos vencimientos, dos monedas, intereses por moneda). El valor del módulo está en persistir el documento mensual y conciliarlo.

**Convenciones BAC Nicaragua** (contrato de tarjeta de crédito y estado de cuenta):

- Fecha de corte mensual; estado de cuenta emitido dentro de los 7 días hábiles.
- Dos fechas límite: **pago mínimo** y **pago de contado** (pueden diferir y no siempre la de contado es posterior).
- Pago mínimo = amortización ≥ 4 % del principal del ciclo + intereses corrientes y moratorios (puede haber cuota mínima). Se guarda el valor **impreso**; el estimado es informativo.
- Intereses corrientes y moratorios **separados por moneda**.
- Una línea de crédito con carteras en córdobas y dólares, con equivalencia; el saldo local se indexa (mantenimiento de valor), por lo que **no se recalcula**: se guarda lo impreso.
- Comisión por adelanto de efectivo y bonificación por pago de contado.
- Líneas informativas de lealtad (PriceCash, sorteos) que **no** son movimientos.

**Viabilidad del parseo (spike, sin persistir datos)**

- El PDF de BAC está cifrado (RC4, password de usuario vacío, permisos de copia restringidos).
- `github.com/ledongthuc/pdf` (ya en `go.mod`) lo abre y extrae contenido; `GetPlainText` devuelve el texto sin saltos de línea útiles.
- `page.Content()` expone cada glifo con X/Y: agrupando por Y (±2) y ordenando por X se reconstruyen filas y columnas; insertando un espacio cuando el hueco entre glifos supera un umbral se recuperan los montos local/USD.
- Conclusión: **no hace falta dependencia nueva ni binario externo**; el parser se apoya en posiciones.

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| Firefly III-style: tarjeta = cuenta | simple, reusa presupuesto | no hay corte, mínimo, contado ni conciliación | ❌ |
| Tabla `credit_cards` + `card_statements` + `card_transactions` | refleja el documento del banco; conciliación real | 3 tablas nuevas | ✅ |
| Reutilizar `debts`/`debt_bills` | cero tablas nuevas | deuda fija con cuotas no es revolvente; contamina esquema protegido | ❌ |
| Montos en `INTEGER` (centavos) | exactitud | rompe la convención `REAL`/`float64` del repo; exige capa de conversión | ❌ |
| Montos en `REAL` (consistente) | uniformidad, menos código | error de redondeo; se mitiga con tolerancia y valores impresos | ✅ |
| Parser con binario `pdftotext` | robusto | suma ~30–50 MB a la imagen distroless y un `exec` | ❌ |
| Parser con `ledongthuc/pdf` + reconstrucción X/Y | sin deps nuevas, sin binarios, verificado | sensible a cambios de formato del banco | ✅ |
| Guardar el PDF adjunto | trazabilidad | PII y datos financieros en el volumen; riesgo de commit | ❌ |
| Preview + confirm | control del usuario, evita basura | dos endpoints | ✅ |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001 — La tarjeta es entidad propia, no una cuenta**

- **Contexto**: `accounts.type='credit_card'` existe para presupuesto, pero no tiene corte, límites ni estado de cuenta.
- **Decisión**: `credit_cards` cuelga de `banks`; no se reutiliza `accounts` en P0 (el vínculo queda en P2).
- **Consecuencias**: modelo fiel al banco; el presupuesto no ve la deuda de la tarjeta hasta P2.

**ADR-002 — Estados de cuenta como entidad de primera clase**

- **Contexto**: ningún OSS modela el documento mensual; BAC imprime totales autoritativos.
- **Decisión**: `card_statements` guarda los totales impresos; los derivados se calculan.
- **Consecuencias**: conciliación posible; una fila por mes y tarjeta.

**ADR-003 — Módulos con namespace canónico (`ARCH-MODULES`)**

- **Contexto**: el usuario pidió identificar módulos como `bank.bac.module.foo.bar`, igual que los plugins de conexión (`assa.nicaragua`).
- **Decisión**: ID `dominio.proveedor.país.producto.capacidad.acción`; carpeta espeja el ID bajo `internal/modules/`; `module.go` + registry + `all/register.go`; solo módulos nuevos.
- **Consecuencias**: trazabilidad (`source_module_id`), importadores enchufables, y una regla nueva que convive con `ARCH-LAYERS` (capas dentro del módulo).

**ADR-004 — Dinero en `REAL`/`float64`**

- **Contexto**: todo el repo usa `REAL` (`debts`, `transactions`, `accounts`). Los centavos enteros serían más exactos.
- **Decisión**: `REAL`, redondeo a 2 decimales en el servicio y comparación con tolerancia 0.005.
- **Consecuencias**: consistencia y menos código; se documenta el riesgo de redondeo y se guarda lo impreso por el banco para no recomputar.

**ADR-005 — El PDF no se persiste y no se importa PII**

- **Contexto**: el estado de cuenta contiene nombre del titular, producto enmascarado y movimientos.
- **Decisión**: parseo en memoria; solo se guarda `last4`; el titular es un campo que el usuario escribe; no se loguea contenido.
- **Consecuencias**: menor trazabilidad del archivo; se compensa con `import_hash` + `source_module_id`.

**ADR-006 — Importación en dos pasos con hash de preview**

- **Contexto**: el parser puede equivocarse; confirmar a ciegas es riesgoso.
- **Decisión**: `preview` devuelve datos + `preview_hash`; `confirm` reenvía el payload y el server recomputa el hash; el estado y los movimientos se insertan en una sola transacción SQL.
- **Consecuencias**: sin estado en servidor, sin persistir el PDF; el usuario revisa y corrige antes de guardar.

**ADR-007 — Categorías existentes, sin taxonomía paralela**

- **Decisión**: `card_transactions.category_id` referencia `categories` (presupuesto) con `ON DELETE SET NULL`.
- **Consecuencias**: un solo catálogo; si no hay categorías, el campo queda vacío.

---

## 5. Diseño Técnico

### 5.1 Arquitectura

```
Frontend (React)
  └─ tab Tarjetas (BankDetail) → módulo UI creditCards
        │  fetch /api/...
        ▼
internal/api/routes.go  ──(authMiddleware)──►  internal/modules/all
                                                    │
                    ┌───────────────────────────────┴──────────────────────────┐
                    ▼                                                          ▼
        bank.creditcard                                        bank.bac.ni.creditcard.statement.import
  handlers.go → service.go → storage.go → model.go            parser.go → models.ParsedStatement
                    │                                                          │
                    └──────────────────► SQLite ◄──────────────────────────────┘
                              migrations/0048_create_credit_cards
```

### 5.2 Componentes

#### 5.2.1 Módulo core `bank.creditcard`

- **Responsabilidad**: CRUD de tarjetas, estados de cuenta y movimientos; derivados y conciliación.
- **Ubicación**: `internal/modules/bank/creditcard/`
- **Archivos**: `module.go`, `model.go`, `storage.go`, `service.go`, `handlers.go`, tests.
- **Dependencias**: `*sql.DB`, `internal/models` (moneda/categoría), `internal/services` (formato de moneda si aplica).

#### 5.2.2 Módulo importador `bank.bac.ni.creditcard.statement.import`

- **Responsabilidad**: detectar y parsear el PDF BAC; devolver `models.ParsedStatement`.
- **Ubicación**: `internal/modules/bank/bac/ni/creditcard/statement/import/`
- **Archivos**: `module.go`, `parser.go`, `lines.go`, `classify.go`, tests.
- **Contrato**: implementa `modules.StatementImporter` (`CanParse`, `Parse`).
- **Sin persistencia**: no toca SQLite.

#### 5.2.3 Registry

- **Ubicación**: `internal/modules/{registry.go,contract.go}` + `internal/modules/all/register.go`.
- **Tipos**:

```go
type Info struct {
    ID, Version, Kind, Provider, Country, Name, Description string
    Migrations []string
}

type Module interface{ Info() Info }

type StatementImporter interface {
    Module
    CanParse(filename, mimeType string) bool
    Parse(ctx context.Context, r io.Reader) (*models.ParsedStatement, error)
}
```

#### 5.2.4 Utilidad de líneas PDF

- **Ubicación**: `internal/modules/bank/bac/ni/creditcard/statement/import/lines.go` (privada del módulo).
- `reconstructLines(page pdf.Page) []Line` agrupa glifos por Y (±2), ordena por X e inserta espacios según huecos.
- Se testea con glifos sintéticos (sin PDF en el repo).

### 5.3 Modelo de datos

Migración `0048_create_credit_cards` (up/down). Fechas de negocio en `TEXT` ISO `YYYY-MM-DD`; timestamps `DATETIME`.

```sql
CREATE TABLE credit_cards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    bank_id INTEGER NOT NULL,
    alias TEXT NOT NULL COLLATE NOCASE,
    network TEXT NOT NULL CHECK (network IN ('visa','mastercard','amex','diners','other')),
    last4 TEXT NOT NULL CHECK (last4 GLOB '[0-9][0-9][0-9][0-9]'),
    holder TEXT NOT NULL DEFAULT '',
    currency_id INTEGER NOT NULL,
    secondary_currency_id INTEGER,
    credit_limit REAL NOT NULL DEFAULT 0,
    credit_limit_usd REAL NOT NULL DEFAULT 0,
    statement_day INTEGER NOT NULL DEFAULT 1 CHECK (statement_day BETWEEN 1 AND 31),
    payment_due_day INTEGER NOT NULL DEFAULT 1 CHECK (payment_due_day BETWEEN 1 AND 31),
    min_payment_pct REAL NOT NULL DEFAULT 4 CHECK (min_payment_pct BETWEEN 0 AND 100),
    interest_current_rate REAL NOT NULL DEFAULT 0 CHECK (interest_current_rate >= 0),
    interest_late_rate REAL NOT NULL DEFAULT 0 CHECK (interest_late_rate >= 0),
    interest_current_rate_usd REAL NOT NULL DEFAULT 0 CHECK (interest_current_rate_usd >= 0),
    interest_late_rate_usd REAL NOT NULL DEFAULT 0 CHECK (interest_late_rate_usd >= 0),
    color TEXT NOT NULL DEFAULT 'blue',
    icon TEXT NOT NULL DEFAULT 'credit',
    active INTEGER NOT NULL DEFAULT 1,
    notes TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (bank_id) REFERENCES banks(id) ON DELETE RESTRICT,
    FOREIGN KEY (currency_id) REFERENCES currencies(id),
    FOREIGN KEY (secondary_currency_id) REFERENCES currencies(id),
    UNIQUE (bank_id, alias),
    CHECK (secondary_currency_id IS NULL OR secondary_currency_id <> currency_id)
);
CREATE INDEX idx_credit_cards_bank ON credit_cards(bank_id);

CREATE TABLE card_statements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    card_id INTEGER NOT NULL,
    period_start TEXT,
    period_end TEXT NOT NULL,
    min_payment_due_date TEXT NOT NULL,
    full_payment_due_date TEXT NOT NULL,
    previous_balance REAL NOT NULL DEFAULT 0,
    previous_balance_usd REAL NOT NULL DEFAULT 0,
    interest_current REAL NOT NULL DEFAULT 0,
    interest_current_usd REAL NOT NULL DEFAULT 0,
    min_payment REAL NOT NULL DEFAULT 0,
    min_payment_usd REAL NOT NULL DEFAULT 0,
    total_payment REAL NOT NULL DEFAULT 0,
    total_payment_usd REAL NOT NULL DEFAULT 0,
    new_balance REAL NOT NULL DEFAULT 0,
    new_balance_usd REAL NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'unpaid' CHECK (status IN ('unpaid','partial','paid')),
    source TEXT NOT NULL DEFAULT 'manual' CHECK (source IN ('manual','import')),
    source_module_id TEXT NOT NULL DEFAULT '',
    import_hash TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (card_id) REFERENCES credit_cards(id) ON DELETE CASCADE,
    UNIQUE (card_id, period_end)
);
CREATE INDEX idx_card_statements_card ON card_statements(card_id, period_end DESC);
CREATE UNIQUE INDEX idx_card_statements_import ON card_statements(card_id, import_hash) WHERE import_hash <> '';

CREATE TABLE card_transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    card_id INTEGER NOT NULL,
    statement_id INTEGER,
    posted_at TEXT NOT NULL,
    concept TEXT NOT NULL,
    category_id INTEGER,
    type TEXT NOT NULL CHECK (type IN (
        'purchase','payment','payment_bonus','cash_advance','cash_advance_fee',
        'interest_current','interest_late','fee','tax','bonus','refund','adjustment','other')),
    amount_local REAL NOT NULL DEFAULT 0,
    amount_usd REAL NOT NULL DEFAULT 0,
    currency_original TEXT NOT NULL DEFAULT 'NIO' CHECK (currency_original IN ('NIO','USD')),
    status TEXT NOT NULL DEFAULT 'posted' CHECK (status IN ('posted','pending')),
    external_ref TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (card_id) REFERENCES credit_cards(id) ON DELETE CASCADE,
    FOREIGN KEY (statement_id) REFERENCES card_statements(id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL
);
CREATE INDEX idx_card_transactions_card ON card_transactions(card_id, posted_at DESC);
CREATE INDEX idx_card_transactions_statement ON card_transactions(statement_id);
CREATE UNIQUE INDEX idx_card_transactions_ref ON card_transactions(card_id, external_ref) WHERE external_ref <> '';
```

**Derivados (no se almacenan)**

| Dato | Cálculo |
|---|---|
| Saldo actual local/USD | `último new_balance` + Σ movimientos con `posted_at > period_end` |
| Disponible local/USD | `credit_limit − saldo actual` (mínimo 0 al mostrar) |
| Utilización | `saldo actual / límite` |
| Vencimiento próximo | `min_payment_due_date − hoy` |
| Estado vencido | `min_payment_due_date < hoy` y saldo > 0 |
| Total por tipo | `SUM(amount_local)` / `SUM(amount_usd)` agrupado por `type` |
| Conciliación | `previous_balance + Σ cargos − Σ abonos` vs `new_balance` (diff) |

### 5.4 APIs / Contratos

Todas con `authMiddleware`. IDs inválidos → `400 invalid_id`.

#### Tarjetas

| Método | Ruta |
|---|---|
| GET | `/api/banks/{bank_id}/credit-cards` |
| POST | `/api/banks/{bank_id}/credit-cards` |
| GET | `/api/credit-cards/{id}` |
| PUT | `/api/credit-cards/{id}` |
| DELETE | `/api/credit-cards/{id}` |

`POST /api/banks/{bank_id}/credit-cards`:

```json
{
  "alias": "Tarjeta Demo",
  "network": "visa",
  "last4": "0000",
  "holder": "Titular Demo",
  "currency_id": 1,
  "secondary_currency_id": 2,
  "credit_limit": 50000,
  "credit_limit_usd": 1400,
  "statement_day": 15,
  "payment_due_day": 14,
  "min_payment_pct": 4,
  "interest_current_rate": 24,
  "interest_late_rate": 36,
  "interest_current_rate_usd": 18,
  "interest_late_rate_usd": 30,
  "color": "blue",
  "icon": "credit",
  "active": true,
  "notes": ""
}
```

`GET /api/credit-cards/{id}` agrega derivados:

```json
{
  "id": 1,
  "bank_id": 1,
  "alias": "Tarjeta Demo",
  "network": "visa",
  "last4": "0000",
  "balance_local": 1234.56,
  "balance_usd": 45.67,
  "available_local": 48765.44,
  "available_usd": 1354.33,
  "last_statement_id": 1,
  "statement_count": 1
}
```

Errores: `400 validation_error`, `404 bank_not_found` / `credit_card_not_found`, `409 duplicate_card`.

#### Estados de cuenta

| Método | Ruta |
|---|---|
| GET | `/api/credit-cards/{id}/statements` |
| POST | `/api/credit-cards/{id}/statements` |
| GET | `/api/credit-card-statements/{id}` |
| PUT | `/api/credit-card-statements/{id}` |
| DELETE | `/api/credit-card-statements/{id}` |

`GET /api/credit-card-statements/{id}` devuelve el estado + `summary` (por tipo) + `reconciliation`:

```json
{
  "id": 1,
  "card_id": 1,
  "period_end": "2026-03-15",
  "min_payment_due_date": "2026-04-14",
  "full_payment_due_date": "2026-04-07",
  "previous_balance": 1000.00,
  "new_balance": 1234.56,
  "summary": [
    {"type": "purchase", "amount_local": 500.00, "amount_usd": 14.00},
    {"type": "payment", "amount_local": -250.00, "amount_usd": 0}
  ],
  "reconciliation": {"expected": 1234.56, "actual": 1234.56, "difference": 0}
}
```

#### Movimientos

| Método | Ruta |
|---|---|
| GET | `/api/credit-card-statements/{id}/transactions` |
| POST | `/api/credit-card-statements/{id}/transactions` |
| PUT | `/api/credit-card-transactions/{id}` |
| DELETE | `/api/credit-card-transactions/{id}` |

Filtros: `?type=purchase&q=demo`.

#### Importación

**Preview** — `POST /api/credit-cards/{id}/statements/import/preview` (`multipart/form-data`, campo `file`, máx 5 MB):

```json
{
  "module_id": "bank.bac.ni.creditcard.statement.import",
  "module_version": "1.0.0",
  "statement": {
    "period_end": "2026-03-15",
    "min_payment_due_date": "2026-04-14",
    "full_payment_due_date": "2026-04-07",
    "previous_balance": 1000.00,
    "previous_balance_usd": 100.00,
    "interest_current": 10.00,
    "interest_current_usd": 1.00,
    "min_payment": 200.00,
    "min_payment_usd": 20.00,
    "total_payment": 1234.56,
    "total_payment_usd": 123.45,
    "new_balance": 1234.56,
    "new_balance_usd": 123.45
  },
  "card": {"last4": "0000", "matched": true},
  "transactions": [
    {
      "posted_at": "2026-03-02",
      "concept": "Comercio Demo",
      "type": "purchase",
      "amount_local": 123.45,
      "amount_usd": 0,
      "currency_original": "NIO",
      "external_ref": "demo-ref-0001"
    }
  ],
  "warnings": [
    {"code": "reconciliation_mismatch", "message": "Diferencia de 0.50 en moneda local"}
  ],
  "preview_hash": "sha256:0000000000000000000000000000000000000000000000000000000000000000"
}
```

Errores: `400 invalid_file` (no PDF, > 5 MB), `422 unsupported_statement`, `422 card_mismatch`, `409 statement_exists`.

**Confirm** — `POST /api/credit-cards/{id}/statements/import` (JSON con el mismo payload + `preview_hash`) → `201` con el estado creado. Recomputa el hash sobre el payload normalizado; si no coincide → `400 preview_mismatch`. Inserta estado + movimientos en una transacción SQL.

#### Módulos

`GET /api/modules` → lista de `Info` del registry. Solo metadatos; sin credenciales.

### 5.5 Parser BAC (contrato interno)

```go
type ParsedStatement struct {
    ModuleID      string
    ModuleVersion string
    PeriodEnd     string
    MinPaymentDue string
    FullPaymentDue string
    PreviousBalance, PreviousBalanceUSD float64
    InterestCurrent, InterestCurrentUSD float64
    MinPayment, MinPaymentUSD           float64
    TotalPayment, TotalPaymentUSD       float64
    NewBalance, NewBalanceUSD           float64
    Last4          string
    Transactions   []ParsedTransaction
    Warnings       []ParseWarning
}
```

**Algoritmo**

1. Abrir con `pdf.NewReader` (soporta el cifrado con password vacío).
2. Por página: `GetPlainText` (conserva palabras y espacios; no conserva saltos de línea).
3. Recortar la sección de movimientos entre `Previous balance` y `INTERESES CORRIENTES DEL…`.
4. Encabezado por regex tolerantes a espacios (`\s*`) sobre el texto completo: `Producto`, `Fecha de corte`, `Pago Mínimo/Fecha límite`, `Pago Contado/Fecha limite`, `Saldo Al Corte/Local`, `Saldo Al Corte/Dólares`, `INTERESES CORRIENTES DEL MES/LOCAL`, `/DÓLARES`, `Previous balance`.
5. Filas con fecha: regex perezosa `fecha + concepto + monto local + monto USD` (soporta montos pegados entre sí).
6. Bonificaciones sin fecha: regex sobre `BONIFICACION…` con dos montos; se imputan a la fecha de corte.
7. Ignorar líneas informativas (`PRICECASH`, `DISPONIBLE`, `SORTEOS`, `SERIE`, ecos de producto/titular).
8. Clasificar y normalizar (`external_ref` = `sha256(fecha|concepto|local|usd|índice)`).
9. Sanidad: si Σ movimientos no cuadra con los totales impresos, agregar warning (no abortar).
10. Devolver `ParsedStatement` + `preview_hash` (SHA-256 del JSON normalizado, sin el hash).

> Nota de implementación: `Content()` de `ledongthuc/pdf` entrega la X del *run*, no de cada glifo, por lo que la reconstrucción posicional no es fiable. `GetPlainText` + anclas resultó robusto y fue validado contra un estado de cuenta real (26 movimientos, conciliación exacta).

**Restricciones**: no leer el nombre del titular; solo `last4`; no loguear líneas; no persistir el PDF; un parser que no reconoce el formato devuelve `unsupported_statement` con mensaje accionable.

### 5.6 Frontend

Estructura del módulo UI (espeja el namespace):

```
frontend/src/modules/bank/creditcard/
├── pages/CreditCardFormPage.tsx
├── pages/CreditCardShowPage.tsx
├── pages/CreditCardStatementPage.tsx
├── pages/CreditCardStatementFormPage.tsx
├── pages/CreditCardImportPage.tsx
├── components/CreditCardList.tsx
├── components/StatementList.tsx
├── components/TransactionList.tsx
├── api.ts
├── types.ts
└── utils.ts
```

Rutas nuevas:

| Ruta | Página |
|---|---|
| `/bancos/:id?tab=tarjetas` | `BankDetailPage` (tab) |
| `/bancos/:id/tarjetas/new` | `CreditCardFormPage` |
| `/bancos/:id/tarjetas/:cardId/edit` | `CreditCardFormPage` |
| `/bancos/:id/tarjetas/:cardId` | `CreditCardShowPage` |
| `/bancos/:id/tarjetas/:cardId/importar` | `CreditCardImportPage` |
| `/bancos/:id/tarjetas/:cardId/estados/new` | `CreditCardStatementFormPage` |
| `/bancos/:id/tarjetas/:cardId/estados/:statementId/edit` | `CreditCardStatementFormPage` |
| `/bancos/:id/tarjetas/:cardId/estados/:statementId` | `CreditCardStatementPage` |

`BACK_ROUTES` (antes de la regla `/bancos/:id`):

```ts
{ pattern: /^\/bancos\/\d+\/tarjetas\/new$/, to: (p) => `/bancos/${p.split('/')[2]}?tab=tarjetas` },
{ pattern: /^\/bancos\/\d+\/tarjetas\/\d+\/edit$/, to: (p) => `/bancos/${p.split('/')[2]}/tarjetas/${p.split('/')[4]}` },
{ pattern: /^\/bancos\/\d+\/tarjetas\/\d+\/importar$/, to: (p) => `/bancos/${p.split('/')[2]}/tarjetas/${p.split('/')[4]}` },
{ pattern: /^\/bancos\/\d+\/tarjetas\/\d+\/estados\/new$/, to: (p) => `/bancos/${p.split('/')[2]}/tarjetas/${p.split('/')[4]}` },
{ pattern: /^\/bancos\/\d+\/tarjetas\/\d+\/estados\/\d+\/edit$/, to: (p) => `/bancos/${p.split('/')[2]}/tarjetas/${p.split('/')[4]}/estados/${p.split('/')[6]}` },
{ pattern: /^\/bancos\/\d+\/tarjetas\/\d+\/estados\/\d+$/, to: (p) => `/bancos/${p.split('/')[2]}/tarjetas/${p.split('/')[4]}` },
{ pattern: /^\/bancos\/\d+\/tarjetas\/\d+$/, to: (p) => `/bancos/${p.split('/')[2]}?tab=tarjetas` },
```

UI:

- **Tab Tarjetas** en `BankDetailPage`: cards con alias, `**** 0000`, marca, límite, disponible, badge activa; `CardMenu` (Editar / Eliminar); `CreateMenu` (Nueva tarjeta / Importar estado).
- **Detalle tarjeta**: lectura; `CardMenu` (Editar / Eliminar); acción primaria «Importar estado de cuenta»; lista de estados con periodo, corte, saldo local/USD y badge.
- **Importación**: página con pasos 1 (archivo) y 2 (revisión). Paso 2 muestra encabezado, tabla de movimientos en modo lectura, warnings, y botones «Importar» / «Elegir otro archivo». Muestra `module_id` y versión al pie.
- Reutiliza: `Select`, `Toggle`, `CardMenu`, `CreateMenu`, `DeleteModal`, `IconPickerModal`, `LoadingSpinner`, `useCurrencyFormatStore`, `EmptyCard`.
- Categorías: `GET /api/budget/category-groups` aplanado (patrón `BudgetTransactionsPage`).

### 5.7 i18n

Namespace `creditCards.*` en `frontend/public/i18n/es.json` y `en.json` (fuente de verdad). Claves mínimas: `title`, `tab`, `create`, `edit`, `delete_*`, `alias`, `network`, `last4`, `holder`, `limit_local`, `limit_usd`, `available`, `statement_day`, `payment_due_day`, `min_payment_pct`, `interest_rate`, `interest_late_rate`, `color`, `icon`, `active`, `notes`, `statements`, `statement_*`, `transactions`, `transaction_*`, `type_*` (13 tipos), `import_*`, `preview_*`, `warning_*`, `reconciliation_*`, `empty_*`, `validation_*`, `saved`, `deleted`.

Reutiliza `app.edit`, `app.delete`, `app.save`, `app.cancel`, `app.search`, y `banks.*` donde aplique.

### 5.8 Errores de dominio

| Código | HTTP | Origen |
|---|---|---|
| `validation_error` | 400 | service |
| `invalid_id` | 400 | api |
| `invalid_file` | 400 | api/upload |
| `preview_mismatch` | 400 | service |
| `bank_not_found` | 404 | storage |
| `credit_card_not_found` | 404 | storage |
| `statement_not_found` | 404 | storage |
| `transaction_not_found` | 404 | storage |
| `duplicate_card` | 409 | storage (UNIQUE) |
| `statement_exists` | 409 | storage (UNIQUE) |
| `card_mismatch` | 422 | importer |
| `unsupported_statement` | 422 | importer |
| `internal_error` | 500 | api |

### 5.9 Dependencias

- **Internas**: `banks`, `currencies`, `categories`, `internal/models`, `internal/modules`, `internal/api` (router y `authMiddleware`).
- **Externas**: ninguna nueva. `ledongthuc/pdf` ya está en `go.mod`.
- **Frontend**: ninguna nueva; Vite/Tailwind/React existentes.

---

## 6. Criterios de Aceptación

### 6.1 Funcionales

**Módulos y arquitectura**

- [x] **CA-ARCH-MODULES**: existen `internal/modules/bank/creditcard/` y `internal/modules/bank/bac/ni/creditcard/statement/import/`; cada uno tiene `module.go` con `Info()`; `GET /api/modules` devuelve ambos IDs con versión; la carpeta espeja el ID.
- [x] **CA-ARCH-LAYERS**: en cada módulo, el SQL está solo en `storage.go`, la lógica solo en `service.go`, los handlers solo parsean HTTP.

**Tarjetas**

- [x] **CA-CARD-001**: crear una tarjeta desde el tab Tarjetas la persiste y aparece en el listado sin recargar.
- [x] **CA-CARD-002**: `last4` con menos/más de 4 dígitos o con letras devuelve `400 validation_error` y el error se muestra junto al campo.
- [x] **CA-CARD-003**: alias duplicado en el mismo banco devuelve `409 duplicate_card`.
- [x] **CA-CARD-004**: borrar una tarjeta con estados de cuenta pide confirmación por palabra y elimina en cascada estados y movimientos.
- [x] **CA-CARD-005**: el detalle muestra disponible local/USD derivado y no permite editar inline.

**Estados de cuenta**

- [x] **CA-STMT-001**: crear dos estados con la misma tarjeta y `period_end` devuelve `409 statement_exists`.
- [x] **CA-STMT-002**: el detalle muestra totales por tipo y la conciliación con diferencia 0.00 para datos consistentes.
- [x] **CA-STMT-003**: una diferencia > 0.005 se muestra como warning visible, sin modificar los totales impresos.
- [x] **CA-STMT-004**: borrar un estado elimina sus movimientos (CASCADE) y actualiza el saldo derivado.

**Movimientos**

- [x] **CA-TX-001**: crear un movimiento manual con tipo `payment` y monto positivo devuelve error de validación.
- [x] **CA-TX-002**: un movimiento `purchase` en USD con `amount_usd` > 0 y local 0 se muestra correctamente en la columna dólares.
- [x] **CA-TX-003**: el filtro por tipo y la búsqueda por concepto funcionan combinados.
- [x] **CA-TX-004**: editar la categoría de un movimiento no altera montos ni conciliación.

**Importación**

- [x] **CA-IMP-001**: subir el PDF de BAC devuelve preview con `period_end`, totales, movimientos, `warnings` y `preview_hash`.
- [x] **CA-IMP-002**: el preview no crea filas en la DB (verificado por conteo antes/después).
- [x] **CA-IMP-003**: confirmar inserta 1 estado + N movimientos en una sola transacción; si falla un movimiento, no queda nada.
- [x] **CA-IMP-004**: reimportar el mismo corte devuelve `409 statement_exists`.
- [x] **CA-IMP-005**: un PDF que no es de BAC devuelve `422 unsupported_statement` sin crear datos.
- [x] **CA-IMP-006**: un PDF con `last4` distinto al de la tarjeta devuelve `422 card_mismatch`.
- [x] **CA-IMP-007**: modificar el payload entre preview y confirm devuelve `400 preview_mismatch`.
- [x] **CA-IMP-008**: el PDF no aparece en `data/`, en el repo ni en logs después de la importación.
- [x] **CA-IMP-009**: los estados importados guardan `source = 'import'`, `source_module_id` e `import_hash`.

**Saldo y derivados**

- [x] **CA-DER-001**: el disponible local = límite local − (último saldo al corte + movimientos posteriores), acotado a ≥ 0 al mostrar.
- [x] **CA-DER-002**: sin estados de cuenta, el disponible = límite.

### 6.2 Reglas de UI/UX

- [x] **CA-UI-SELECT**: marca, moneda, categoría, color y tipo usan `Select`; `npm run check:ui` pasa.
- [x] **CA-UI-TOKENS**: inputs con tokens del tema; sin colores hardcodeados.
- [x] **CA-UI-DARKMODE**: formularios, tab, listados y preview verificados en darkmode.
- [x] **CA-UI-TOGGLE**: `active` usa `Toggle` con `aria-label`.
- [x] **CA-UI-LIST-PATTERN**: listados con cards y `CardMenu`; sin formularios inline; `EmptyCard` cuando no hay datos.
- [x] **CA-UI-CREATE-MENU**: creación desde `CreateMenu`, no botón «+».
- [x] **CA-UI-EDIT-LABEL**: acciones de edición rotuladas `app.edit` («Editar»).
- [x] **CA-UI-I18N**: claves `creditCards.*` en `frontend/public/i18n/{es,en}.json`; `npm run build` y `curl /i18n/es.json` las sirven.
- [x] **CA-UX-BACK**: flecha atrás en el header según `BACK_ROUTES`; sin «← Título».
- [x] **CA-UX-MODAL**: ningún modal cierra por click fuera; overlay bloquea; cierre visible.
- [x] **CA-UX-MODAL-CONFIRM**: `DeleteModal` enfoca el input y acepta Enter con `confirmo`.
- [x] **CA-UX-DETAIL-ACTIONS**: Editar/Eliminar solo en `CardMenu`; importar es acción primaria permitida.
- [x] **CA-UX-DETAIL-READONLY**: detalle sin `<form>`; editar navega a `/edit/:id`.
- [x] **CA-UX-PREREQ**: backend valida banco/tarjeta/estado; frontend redirige si falta el prerrequisito.
- [x] **CA-UX-EMPTY**: estados vacíos con acción en tarjetas, estados y movimientos.
- [x] **CA-UX-LOADING**: spinner/skeleton y botones deshabilitados en operaciones asíncronas.
- [x] **CA-UX-FEEDBACK**: toast de éxito/error; warnings del parser visibles antes de confirmar.
- [x] **CA-UX-FORMS**: una acción primaria, validación con error junto al campo.

### 6.3 Datos y seguridad

- [x] **CA-DATA-MIGRATIONS**: `0048` aplica y revierte limpio sobre una copia en `/tmp`; `PRAGMA integrity_check` OK.
- [x] **CA-DATA-PRODCOPY**: validación sobre copia de producción; la copia se borra al terminar.
- [x] **CA-SEC-NO-PII**: solo `last4`; no se guardan PAN/CVV/PIN; el titular no se autocompleta desde el PDF; no hay PDFs ni volcados en el diff.
- [x] **CA-SEC-SYNTHETIC**: ejemplos y tests usan `6000-0000-0000-0000`, `Comercio Demo`, `demo@example.com`.
- [x] **CA-SEC-PRODCOPY**: ninguna copia de DB en el repo.
- [x] **CA-SEC-COMMIT**: `git diff --cached` sin PII, secretos, DBs, PDFs ni exports.
- [x] **CA-SEC-HOOKS**: hooks activos; sin `--no-verify`.

### 6.4 Testing

- **Unit tests**: parser sobre líneas/glifos sintéticos (encabezado, filas, informativas, signos, clasificación); validaciones de tarjeta/estado/movimiento; conciliación y derivados.
- **Integration tests**: migración 0048; CRUD completo; import preview+confirm; dedupe; `card_mismatch`; rollback de import.
- **E2E/manual**: crear tarjeta → importar PDF real (local, sin commit) → revisar preview → confirmar → ver movimientos → conciliar → editar categoría → borrar.
- **Performance**: preview < 2 s; memoria < 32 MB (medida con `runtime.MemStats` en test o perfil manual).

---

## 7. Plan de Implementación

### 7.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|---|---|---|---|
| 0 | Regla `ARCH-MODULES` + registry + `all/register.go` + `GET /api/modules` | 0.5 día | — |
| 1 | Migración `0048` + `model.go` + `storage.go` del módulo core | 1 día | Fase 0 |
| 2 | `service.go` + validaciones + derivados + conciliación | 1 día | Fase 1 |
| 3 | `handlers.go` + wiring de rutas + tests de API | 0.5 día | Fase 2 |
| 4 | Frontend: tab Tarjetas + CRUD de tarjeta + i18n | 1.5 días | Fase 3 |
| 5 | Frontend: estados de cuenta + movimientos + conciliación | 1.5 días | Fase 4 |
| 6 | Módulo importador BAC: líneas X/Y + parser + preview/confirm + UI | 2 días | Fase 5 |
| 7 | Tests, darkmode, gates, validación con copia de producción | 1 día | Fase 6 |

### 7.2 Milestones

1. **MVP**: tarjetas + estados + movimientos manuales + tab en banco.
2. **V1.0**: importación BAC con preview/confirm y conciliación.
3. **Futuro (P1/P2)**: CSV genérico, reglas de categorización, cuotas, vínculo con presupuesto.

---

## 8. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| BAC cambia el layout del PDF | Media | Alto | Parser aislado en su módulo, versión en `Info()`, warnings, error accionable y carga manual como fallback |
| El texto extraído pega columnas y montos | Media | Alto | Reconstrucción por X/Y + tests sintéticos de glifos; validación de conciliación |
| PDFs con otra versión de cifrado | Baja | Medio | Error `unsupported_statement` claro; no se intenta fuerza bruta |
| Redondeo `float64` | Media | Medio | Tolerancia 0.005, redondeo a 2 decimales, guardar lo impreso |
| Duplicar movimientos al reimportar | Media | Alto | `UNIQUE(card_id, period_end)` + `import_hash` + `external_ref` |
| PII en logs o commits | Baja | Muy alto | No loguear contenido; solo `last4`; hooks + revisión de diff |
| Crecimiento del alcance (cuotas, presupuesto) | Alta | Medio | P1/P2 explícitos fuera del MVP |
| Memoria en iHost | Baja | Medio | 5 MB máx, una importación, sin caché, streaming a buffer acotado |
| Regla `ARCH-MODULES` en conflicto con `ARCH-LAYERS` | Media | Medio | Capas dentro del módulo; actualización explícita de `10-architecture.md` en esta spec |

---

## 9. Notas y Referencias

- Reglas: `docs/rules/00-index.md`, `docs/rules/10-architecture.md` (`ARCH-MODULES`), `20-ui.md`, `30-ux.md`, `40-data.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.
- Precedentes internos: SPEC-133 (bancos/cuentas), SPEC-134 (país/acciones), SPEC-135 (detalle informativo/toggle), SPEC-099 (tarjeta como cuenta de presupuesto), SPEC-122/123/127 (documentos y PDFs).
- Firefly III, tipos de cuenta: <https://docs.firefly-iii.org/references/firefly-iii/account-types/>
- Actual Budget, tarjetas de crédito: <https://actualbudget.org/docs/budgeting/credit-cards/>
- Maybe Finance, modelo `CreditCard`: <https://github.com/maybe-finance/maybe/blob/main/app/models/credit_card.rb>
- GnuCash, tarjetas: <https://www.gnucash.org/docs/v5/C/gnucash-guide/chapter_cc.html>
- BAC Nicaragua, contrato de tarjeta de crédito: <https://www.baccredomatic.com/sites/default/files/2024-01/NI-Contrato%20de%20Tarjeta%20de%20Credito%20BAC%20Nicaragua_0.pdf>
- BAC Nicaragua, tabla de costos: <https://www.baccredomatic.com/es-ni/personas/informativo/tabla-de-costos-tarjeta-de-credito-0>
- `ledongthuc/pdf`: <https://github.com/ledongthuc/pdf>
- Nota de privacidad: la estructura del estado de cuenta se verificó con un archivo real del usuario; no se persistió, no se citó y no se commitea. Todos los ejemplos de esta spec son sintéticos.

---

## 10. Historial de Cambios

| Fecha | Autor | Descripción |
|---|---|---|
| 2026-11-18 | OpenCode | Creación inicial de la especificación (SPEC-137) |
| 2026-11-18 | OpenCode | Implementación backend + frontend; parser ajustado a `GetPlainText` + anclas; preview de solo lectura |
| 2026-11-18 | OpenCode | Release: merge a `main` de `feature/SPEC-137` (commit `c11958c0`, merge `0358161f`) |

---

## 11. Verificación de release

| Criterio | Evidencia |
|---|---|
| `CA-ARCH-MODULES` | `GET /api/modules` devuelve `bank.creditcard` y `bank.bac.ni.creditcard.statement.import`; carpetas espejan los IDs |
| `CA-ARCH-LAYERS` | `model.go`, `storage.go`, `service.go`, `handlers.go` separados por módulo; sin SQL fuera de storage |
| `CA-CARD-*` | `go test ./internal/modules/bank/creditcard` (CRUD, validaciones, duplicados, cascada) |
| `CA-STMT-*` | Tests de corte duplicado, resumen y conciliación; estado con diferencia visible |
| `CA-TX-*` | Tests de signo, filtros y categoría; edición sin alterar montos |
| `CA-IMP-*` | Prueba end-to-end por HTTP con PDF real: 26 movimientos, 0 warnings, confirm `201`, reimport `409`, `last4` verificado, PDF no persistido |
| `CA-DER-*` | Tests de saldo y disponible derivados |
| `CA-UI-*` | `npm run build` y `check:ui` OK (127 archivos); selectores con `Select`, toggles con `Toggle`; verificado en darkmode por el usuario |
| `CA-UX-*` | Back en `BACK_ROUTES`, detalle de solo lectura, acciones en `CardMenu`, `EmptyCard`, toasts y modales bloqueantes |
| `CA-DATA-MIGRATIONS` | `TestMigrate048CreditCards`: tablas, índices y CHECK de `last4` |
| `CA-DATA-PRODCOPY` | Validación en DB temporal; `data/app.db` no se tocó |
| `CA-SEC-*` | `git diff --cached` sin PII, PDFs, DBs ni exports; solo `last4`; sin logs de contenido |
| `CA-TEST-LOCAL` | `go build ./...`, `go test ./internal/...`, `npm run build` y `check:ui` OK |
