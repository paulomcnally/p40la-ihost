---
title: "Estados de cuenta multi-banco: interés bonificable, mora y exceso, conciliación de dos bases e importador BANPRO"
id: "SPEC-139"
status: "in_progress"
author: "OpenCode"
created: "2026-11-18"
updated: "2026-11-18"
github_issue: 142
---

# Estados de cuenta multi-banco: interés bonificable, mora y exceso, conciliación de dos bases e importador BANPRO

**ID**: SPEC-139  
**Estado**: in_progress  
**Autor**: OpenCode  
**Creado**: 2026-11-18  
**Actualizado**: 2026-11-18

---

## 1. Resumen Ejecutivo

SPEC-137 modeló el estado de cuenta con una sola base de conciliación (`saldo_anterior + cargos − abonos` contra `saldo al corte`) y con un único importador, el de BAC Nicaragua. Al revisar estados de cuenta reales de **dos bancos** (BAC y BANPRO) surgen tres problemas:

1. **La conciliación da descuadre en estados correctos.** Los dos bancos imprimen **dos totales distintos**: *pago de contado* (saldo anterior + cargos − abonos) y *saldo al corte*, que es el pago de contado **más el interés bonificable**. La app compara contra el saldo al corte sin sumar el bonificable, así que reporta una diferencia igual al interés del mes.
2. **Faltan agregados que los bancos publican:** *interés bonificable* (los dos bancos), *monto en mora*, *pagos atrasados* (cantidad) y *monto en exceso* (BANPRO). Sin ellos el estado de cuenta no refleja la situación real de la cuenta (mora, sobrecupo).
3. **Solo se puede importar BAC**, y el resolver de importadores devuelve el **primer** módulo que acepta PDF, no el que corresponde al archivo. Con dos importadores registrados, el segundo nunca se usaría.

Esta spec define:

1. Los campos nuevos del estado de cuenta (`interest_bonifiable`, `mora_amount`, `arrears_count`, `over_limit_amount`, en local y USD).
2. Una **conciliación de dos bases** (`cash` y `balance`), donde `balance = cash + interest_bonifiable`, válida para ambos bancos.
3. El módulo de proveedor **`bank.banpro.ni.creditcard.statement.import`** para el PDF «Estado de cuenta» de BANPRO, con inferencia de año (las filas usan `MM/DD` sin año).
4. La **resolución multi-importador** por contenido, con error tipado `unsupported_statement`.
5. Correcciones al clasificador de conceptos (mora, cargos administrativos y abonos).

El resultado esperado: importar el estado de cuenta de cualquiera de los dos bancos, ver la mora y el sobrecupo, y que la conciliación cierre por las dos vías sin descuadres falsos.

### Consideraciones de iHost

- Sin dependencias nuevas (`github.com/ledongthuc/pdf` ya está en `go.mod`).
- El PDF se procesa en memoria y se descarta; no se persiste ni se commitea (`SEC-NO-PII`).
- Con dos importadores, el archivo se parsea como máximo dos veces y solo en el `preview`: coste despreciable para PDFs de 2 páginas.

---

## 2. Cumplimiento de reglas

Índice: `docs/rules/00-index.md`.

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Go + `net/http` + SQLite; sin deps nuevas | — |
| `ARCH-FOLDERS` | Nuevo módulo en `internal/modules/bank/banpro/ni/creditcard/statement/import/` | `CA-ARCH-MODULES` |
| `ARCH-LAYERS` | Parser en el módulo; persistencia en `storage.go`; conciliación y validaciones en `service.go` | `CA-ARCH-LAYERS` |
| `ARCH-MODULES` | ID `bank.banpro.ni.creditcard.statement.import` (6 segmentos, minúsculas, país `ni`); `module.go` + registro en `all/register.go` | `CA-ARCH-MODULES` |
| `ARCH-GO` | `gofmt`, errores explícitos y tipados, `log/slog` | — |
| `ARCH-IHOST` | Sin deps nuevas; parseo en memoria; límite 10 MB vigente | — |
| `ARCH-CONFIG` | N/A: no cambia `opencode.json` | — |
| `UI-SELECT` | Filtro por tipo en movimientos sigue con `Select`; no se agregan selectores nativos | `CA-UI-SELECT` |
| `UI-INPUT-TOKENS` | Campos nuevos del formulario manual de estado usan `bg-card`/`text-text` | `CA-UI-TOKENS` |
| `UI-DARKMODE` | Detalle de estado, preview de importación y formulario manual verificados en darkmode | `CA-UI-DARKMODE` |
| `UI-TOGGLE` | N/A: no se agregan booleanos de entidad | — |
| `UI-LIST-PATTERN` | El listado de estados sigue en cards con `CardMenu`; badge de mora sin cambiar el patrón | — |
| `UI-CREATE-MENU` | N/A: no se agrega acción de crear | — |
| `UI-EDIT-LABEL` | N/A: no se agregan acciones de edición | — |
| `UI-I18N` | Claves nuevas (`creditCards.statement_*`) en `frontend/public/i18n/{es,en}.json`; build + verificación por HTTP | `CA-UI-I18N` |
| `UI-GUARDRAIL` | `npm run check:ui` pasa | `CA-UI-SELECT` |
| `UX-BACK` | N/A: no se agregan rutas | — |
| `UX-MODAL` | N/A: el preview sigue siendo página de 2 pasos | — |
| `UX-MODAL-CONFIRM` | N/A: no se agregan confirmaciones por palabra | — |
| `UX-DETAIL-ACTIONS` | N/A: no se agregan acciones al detalle | — |
| `UX-DETAIL-READONLY` | El detalle del estado sigue en modo lectura; los agregados nuevos se muestran | `CA-UX-DETAIL-READONLY` |
| `UX-PREREQ` | N/A: sin cambios de prerrequisitos | — |
| `UX-EMPTY` | N/A: el listado de estados ya tiene `EmptyCard` | — |
| `UX-LOADING` | N/A: el preview ya bloquea con estado de carga | — |
| `UX-FEEDBACK` | Si ningún importador reconoce el PDF: toast con mensaje accionable («descargá el PDF del estado de cuenta del banco, no el resumen de transacciones») | `CA-UX-FEEDBACK` |
| `UX-FORMS` | El formulario manual agrega campos nuevos opcionales; un Guardar + un Cancelar | `CA-UX-FORMS` |
| `DATA-SQLITE` | SQL crudo en `storage.go` | — |
| `DATA-MIGRATIONS` | `migrations/0050_card_statements_aggregates.up.sql` + `.down.sql`, aditiva y reversible | `CA-DATA-MIGRATIONS` |
| `DATA-NO-DROP` | No se toca `data/app.db`; tests con DB en `/tmp` | `CA-DATA-PRODCOPY` |
| `DATA-PRODCOPY` | Migración y conciliación probadas sobre copia de producción en `/tmp` | `CA-DATA-PRODCOPY` |
| `DATA-PROD-WRITE` | N/A: la migración la aplica el arranque del server | — |
| `DATA-PROTECTED-SCHEMA` | N/A: no se toca `debts`/`debt_bills` | — |
| `DATA-SSH` | N/A: no se requiere acceso al iHost para implementar | — |
| `SEC-NO-PII` | La spec describe el layout y los patrones, no transcribe números, titulares, saldos ni movimientos; los fixtures son sintéticos | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Ejemplos con `XXXX-XXXX-XXXX-0000`, «Banco Demo», «Comercio Demo», montos demo | `CA-SEC-SYNTHETIC` |
| `SEC-PRODCOPY` | Copias de DB solo en `/tmp` y borradas al terminar | `CA-SEC-PRODCOPY` |
| `SEC-COMMIT` | `git diff --cached` sin PDFs, DBs ni capturas reales | `CA-SEC-COMMIT` |
| `SEC-HOOKS` | Hooks activos; sin `--no-verify` | `CA-SEC-HOOKS` |
| `SEC-INCIDENT` | N/A: sin secretos nuevos | — |
| `SEC-LOGS` | Prohibido loguear líneas del PDF, referencias bancarias o montos | `CA-SEC-NO-PII` |
| `TEST-LOCAL` | `go build ./...`, `go test ./...`, `check:ui` y prueba manual | `CA-TEST-LOCAL` |
| `TEST-SERVER` | `./scripts/check-server.sh` antes de avisar al usuario | — |
| `TEST-KILLALL` | Reinicio con `killall p40la-ihost` | — |
| `TEST-TESTUSER` | Endpoints probados con `/tmp/cookies.txt` | — |
| `TEST-USER-FLOW` | 🤖 Se pregunta antes del server y antes de cerrar la spec | — |
| `TEST-ROBOT` | Preguntas de flujo con 🤖 | — |
| `TEST-CHECKLIST` | Checklist completo antes de `pending_release` | `CA-TEST-LOCAL` |
| `DELIV-WORKTREE` | Sesión en `../p40la-ihost-spec-139` (rama `feature/SPEC-139`) | — |
| `DELIV-CONFIRM` | Merge y push solo con confirmación explícita | — |
| `DELIV-COMMIT` | Commits `SPEC-139: ...` | — |
| `DELIV-RELEASE` | Release con los 8 pasos | — |
| `DELIV-CHECKRELEASE` | `/spec-check-release` antes de declarar terminado | — |
| `DELIV-HOOKS` | Hooks activos en el worktree | `CA-SEC-HOOKS` |
| `DELIV-DOCKER` | N/A: no cambia `Dockerfile` | — |
| `DELIV-LABELS` | Issue `spec/draft` → `spec/released` | — |
| `DELIV-DEPLOY` | Deploy aprobado por el usuario | — |
| `DELIV-ESCALATION` | N/A al inicio | — |

---

## 3. Requerimientos

### 3.1 Funcionales (P0 — obligatorios)

**Modelo del estado de cuenta**

1. **REQ-001**: Agregar `interest_bonifiable` y `interest_bonifiable_usd` (interés corriente que el banco acredita si se paga el total antes del límite de bonificación). Ambos bancos lo imprimen como línea propia.
2. **REQ-002**: Agregar `mora_amount`, `mora_amount_usd`, `arrears_count` (cantidad de pagos atrasados) y `over_limit_amount`, `over_limit_amount_usd` (monto en exceso sobre el límite). Si el banco no los imprime, quedan en `0` y la UI los oculta.
3. **REQ-003**: `total_payment` mantiene su semántica de **pago de contado** y se documenta explícitamente en `model.go`; `new_balance` es el **saldo al corte** impreso.

**Conciliación de dos bases (reemplaza `REQ-007` de SPEC-137)**

4. **REQ-004**: La conciliación calcula dos resultados:
   - `expected_cash = previous_balance + cargos − abonos` → se compara con `total_payment`.
   - `expected_balance = expected_cash + interest_bonifiable` → se compara con `new_balance`.
   Ambas comparaciones con tolerancia `0.005`, en local y USD. `balanced` es verdadero solo si **las dos** cierran.
5. **REQ-005**: La UI del estado muestra el desglose (`cargos`, `abonos`, `pago de contado`, `interés bonificable`, `saldo al corte`) y, si alguna base no cierra, cuál y por cuánto. Prohibido mostrar un único «descuadre» cuando en realidad difiere el bonificable.

**Resolución de importadores**

6. **REQ-006**: El registry deja de devolver el primer importador que acepta PDFs. Nuevo contrato: `ParseStatements(filename, mimeType string, data []byte) (*models.ParsedStatement, modules.StatementImporter, error)` que recorre los importadores registrados y devuelve el primero cuyo `Parse` no devuelve `ErrUnsupportedStatement`.
7. **REQ-007**: Nuevo error tipado `modules.ErrUnsupportedStatement` (`errors.Is`). Los importadores lo envuelven cuando el layout no coincide (hoy devuelven texto libre como «faltan fechas del encabezado»).
8. **REQ-008**: Si ningún importador reconoce el archivo, el servicio devuelve `unsupported_statement` con mensaje accionable (§`UX-FEEDBACK`). El `unsupported_statement` deja de ser «no hay importador registrado» y pasa a significar «ningún importador reconoce este layout».

**Importador BANPRO**

9. **REQ-009**: Nuevo módulo `bank.banpro.ni.creditcard.statement.import` (tipo `provider`, proveedor `banpro`, país `ni`, versión `1.0.0`) que parsea el PDF «Estado de cuenta» de BANPRO.
10. **REQ-010**: El parser extrae, por anclas tolerantes a espacios y acentos: número de tarjeta enmascarado (`last4`), fecha de corte, fecha de pago, límite de crédito, saldo anterior, compras y retiros, cargos y débitos, pagos y créditos, pago de contado, pago mínimo, crédito disponible, interés corriente, interés bonificable, saldos al corte, monto en mora, pagos atrasados y monto en exceso (local y USD).
11. **REQ-011**: Las filas de movimiento usan `MM/DD` **sin año**: el año se infiere de la fecha de corte (si el mes de la fila es mayor al del corte, es del año anterior). `posted_at` se persiste como ISO.
12. **REQ-012**: Se ignoran los bloques informativos (`DETALLE COMPRA DIFERIDA`, `DETALLE FINANCIERO`, puntos, texto legal, ecos de producto y titular) sin generar movimientos ni warnings.
13. **REQ-013**: El parser devuelve `warnings` cuando un agregado impreso no coincide con la suma de movimientos, sin abortar el preview.
14. **REQ-014**: Si el `last4` impreso no coincide con la tarjeta destino, se rechaza con `card_mismatch` (misma regla que BAC).

**Clasificación de conceptos (aplica a los dos importadores)**

15. **REQ-015**: Conceptos con **mora** se clasifican como `interest_late` cuando son intereses y como `fee` cuando son cargos administrativos. La comprobación de «MORA» debe evaluarse **antes** que la genérica de «INTERES».
16. **REQ-016**: Abonos sin palabra clave (`CREDITO TRANSFERIDO`, `ABONO`, `PAGO`, `NOTA DE CREDITO`) se clasifican como `payment`; si el monto es negativo y no hay palabra clave, `other`. **Prohibido** clasificar un abono como `purchase` negativo: el servicio lo rechaza y el confirm falla.
17. **REQ-017**: `external_ref` usa el **número de referencia del banco** cuando el layout lo imprime (BANPRO); si no, el hash actual (BAC). La deduplicación sigue siendo por `(card_id, external_ref)`.

### 3.2 Funcionales (P1 — importantes)

1. **REQ-018**: Importar y mostrar los **puntos** del programa de lealtad (`points_previous`, `points_earned`, `points_used`, `points_available`) como dato informativo del estado.
2. **REQ-019**: El detalle del estado muestra un resumen visual de mora/sobrecupo (badge en el listado cuando `mora_amount > 0` o `arrears_count > 0`).

### 3.3 Funcionales (P2 — deseables)

1. **REQ-020**: Compras diferidas en tabla propia (`card_installments`: desembolso, tasa, plazo, interés del mes, principal del mes, cuota, saldo), importadas desde el bloque `DETALLE COMPRA DIFERIDA` / equivalente de BAC.
2. **REQ-021**: Proyección del plazo para saldar la deuda con el pago mínimo (dato que ambos bancos imprimen).

### 3.4 No funcionales

- **Migración** aditiva y reversible (`ALTER TABLE ... ADD COLUMN` / `DROP COLUMN`), sin rebuild.
- **Rendimiento**: preview de un estado de 2 páginas < 2 s con dos importadores registrados; ≤ 2 parseos por request.
- **Memoria**: pico < 32 MB por importación; sin caché de PDFs.
- **Seguridad**: `authMiddleware`; solo PDF con firma `%PDF-`; sin PAN/CVV/PIN; sin logs de contenido.
- **Compatibilidad**: estados ya importados conservan sus valores; los campos nuevos arrancan en `0`.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

Dos estados de cuenta reales, procesados en memoria y no citados (`SEC-NO-PII`):

| Observación | BAC Nicaragua | BANPRO |
|---|---|---|
| Layout | `Producto:` + `Previous Balance` + sección de movimientos | `RESUMEN` + tabla `REFERENCIA / FECHA / DESCRIPCIÓN` |
| Fechas de fila | `OCT/17` (mes/año corto) | `11/01` (mes/día, sin año) |
| Total de contado vs saldo al corte | Distintos; la diferencia es el interés bonificable | Ídem |
| Interés bonificable | Línea propia | Línea propia |
| Mora / atrasos / exceso | No impreso | Impreso |
| Referencia por movimiento | No | Sí (numérica larga) |
| Puntos | PriceCash / acciones | Saldo y movimientos de puntos |

**Verificación aritmética del modelo de dos bases** (con datos de ambos PDFs): `saldo anterior + cargos − abonos` coincide exactamente con el **pago de contado** impreso; `pago de contado + interés bonificable` coincide exactamente con el **saldo al corte** impreso. La conciliación actual compara contra el saldo al corte sin sumar el bonificable, por eso reporta un descuadre igual al interés del mes.

### 4.2 Opciones evaluadas

| Decisión | Opción A | Opción B | Elección |
|---|---|---|---|
| Conciliación | Comparar contra el saldo al corte y tolerar la diferencia del bonificable | Dos bases explícitas (`cash` y `balance`) | **B**: la diferencia es un dato, no ruido |
| Selección de importador | Elegir por banco de la tarjeta | Intentar todos y quedarse con el que parsea | **B**: la tarjeta puede estar creada sin banco correcto y el layout es la fuente de verdad; además permite importar estados de bancos no modelados |
| Error de layout | Texto libre | Error tipado `ErrUnsupportedStatement` | **B**: el registry necesita distinguir «no es mío» de «está roto» |
| Año de las filas BANPRO | Exigir estado con año | Inferir del corte | **B**: el PDF no imprime año |
| Referencia bancaria | Solo hash | Referencia del banco con fallback a hash | **B**: sobrevive a reimportaciones y a cambios de formato |
| Abonos desconocidos | Rechazar el confirm | Clasificar como `payment` u `other` | **B**: no se puede bloquear la importación por un concepto nuevo |

### 4.3 ADRs

- **ADR-001 — Dos bases de conciliación.** `cash` valida la suma de movimientos; `balance` valida el saldo impreso incluyendo el bonificable. Es la única forma de que ambos bancos cierren.
- **ADR-002 — El importador se elige por layout, no por banco.** El `last4` y el parseo son la validación real; el banco de la tarjeta es metadato.
- **ADR-003 — La mora se clasifica antes que el interés genérico.** «CARGO DE INTERESES POR MORA» debe ser `interest_late`, no `interest_current`.
- **ADR-004 — Los agregados son del estado, no de la tarjeta.** Mora, atrasos y exceso cambian mes a mes: viven en `card_statements`.

---

## 5. Diseño Técnico

### 5.1 Arquitectura

```
internal/modules/
├── contract.go                 # + ErrUnsupportedStatement, ajuste de StatementImporter
├── registry.go                 # + ParseStatements (recorre importadores)
├── bank/creditcard/
│   ├── model.go                # + agregados, Reconciliation de dos bases
│   ├── service.go              # conciliación, uso del nuevo resolver
│   └── storage.go              # columnas nuevas
├── bank/bac/ni/creditcard/statement/import/
│   ├── parser.go               # envuelve ErrUnsupportedStatement; clasificación de mora
│   └── classify.go             # + mora, abonos
└── bank/banpro/ni/creditcard/statement/import/
    ├── module.go
    ├── parser.go
    ├── classify.go             # reusa el clasificador de BAC si se extrae a un paquete común
    └── parser_test.go          # fixtures sintéticos

migrations/0050_card_statements_aggregates.{up,down}.sql
frontend/src/modules/bank/creditcard/{pages,components,types.ts}
frontend/public/i18n/{es,en}.json
```

### 5.2 Modelo de datos

Migración `0050` (aditiva):

```sql
ALTER TABLE card_statements ADD COLUMN interest_bonifiable REAL NOT NULL DEFAULT 0;
ALTER TABLE card_statements ADD COLUMN interest_bonifiable_usd REAL NOT NULL DEFAULT 0;
ALTER TABLE card_statements ADD COLUMN mora_amount REAL NOT NULL DEFAULT 0;
ALTER TABLE card_statements ADD COLUMN mora_amount_usd REAL NOT NULL DEFAULT 0;
ALTER TABLE card_statements ADD COLUMN arrears_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE card_statements ADD COLUMN over_limit_amount REAL NOT NULL DEFAULT 0;
ALTER TABLE card_statements ADD COLUMN over_limit_amount_usd REAL NOT NULL DEFAULT 0;
ALTER TABLE card_statements ADD COLUMN points_previous INTEGER NOT NULL DEFAULT 0;
ALTER TABLE card_statements ADD COLUMN points_earned INTEGER NOT NULL DEFAULT 0;
ALTER TABLE card_statements ADD COLUMN points_used INTEGER NOT NULL DEFAULT 0;
ALTER TABLE card_statements ADD COLUMN points_available INTEGER NOT NULL DEFAULT 0;
```

`down`: `DROP COLUMN` de las once columnas (precedente `0041`).

### 5.3 Conciliación

```go
type Reconciliation struct {
    ExpectedCash    float64 `json:"expected_cash"`
    PrintedCash     float64 `json:"printed_cash"`
    DifferenceCash  float64 `json:"difference_cash"`
    ExpectedBalance float64 `json:"expected_balance"`
    PrintedBalance  float64 `json:"printed_balance"`
    DifferenceBalance float64 `json:"difference_balance"`
    // Equivalentes en USD con los mismos campos *_usd.
    Balanced        bool    `json:"balanced"`
}
```

```
expected_cash    = previous + Σcargos − Σabonos
expected_balance = expected_cash + interest_bonifiable
balanced         = |diff_cash| ≤ 0.005 && |diff_balance| ≤ 0.005   (local y USD)
```

Reemplaza `ExpectedLocal/ActualLocal/...` de SPEC-137; el frontend del detalle se actualiza en la misma spec.

### 5.4 Resolver de importadores

```go
// contract.go
var ErrUnsupportedStatement = errors.New("unsupported_statement")

type StatementImporter interface {
    Info() Info
    CanParse(filename, mimeType string) bool
    Parse(ctx context.Context, r io.Reader) (*models.ParsedStatement, error) // ErrUnsupportedStatement si no es su layout
}

// registry.go
func (r *Registry) ParseStatements(ctx context.Context, filename, mimeType string, data []byte) (
    *models.ParsedStatement, StatementImporter, error)
```

Regla: se intentan los importadores en orden de registro; se devuelve el primero que parsea sin `ErrUnsupportedStatement`; cualquier otro error se propaga (un PDF corrupto no debe enmascararse como «layout desconocido»). El servicio `PreviewImport` usa este método y mantiene las validaciones posteriores (`normalizeParsed`, `last4`, duplicado, hash).

### 5.5 Parser BANPRO (contrato interno)

1. Abrir con `ledongthuc/pdf` (`GetPlainText`) y concatenar páginas.
2. **Anclas de encabezado**: nombre del banco, `NÚMERO DE TARJETA`, `FECHA DE CORTE`, `FECHA DE PAGO`, `LÍMITE DE CREDITO`.
3. **Anclas de resumen**: `Saldo Anterior`, `Compras y Retiros`, `Cargos y Débitos`, `Pagos y Créditos`, `Pago de Contado`, `Pago Mínimo`, `Crédito Disponible`.
4. **Región de movimientos**: entre `REFERENCIA` (cabecera de tabla) y `INTERÉS CORRIENTES` / `SALDOS AL CORTE`.
5. Filas: `referencia? MM/DD concepto monto_local monto_usd?`; los montos pueden venir sin columna USD.
6. **Anclas de totales**: `INTERÉS CORRIENTES`, `INTERÉS BONIFICABLE`, `SALDOS AL CORTE`, `Monto en Mora`, `Pagos Atrasados`, `Monto en Exceso`.
7. **Puntos**: `Saldo en puntos anterior`, `Puntos Acreditados`, `Puntos Debitados`, `Puntos Disponibles`.
8. Clasificar, inferir año, normalizar y calcular `external_ref` con la referencia bancaria.
9. Sanidad: comparar la suma de movimientos con `Cargos y Débitos` y `Pagos y Créditos`; si no cuadra, `warning` (no aborta).
10. Si faltan las anclas mínimas (corte, pago, región de movimientos) → `ErrUnsupportedStatement`.

### 5.6 Frontend

- **`CreditCardStatementShowPage.tsx`**: tabla de conciliación con dos filas (contado y saldo al corte) y diferencias; bloque de agregados (bonificable, mora, atrasos, exceso) solo si son distintos de `0`; puntos si existen.
- **`CreditCardStatementFormPage.tsx`**: campos nuevos opcionales agrupados en «Agregados del banco».
- **`CreditCardImportPage.tsx`**: preview con los agregados; mensaje accionable si `unsupported_statement`.
- **`CreditCardStatementShowPage` / listado**: badge «En mora» cuando `mora_amount > 0` o `arrears_count > 0`.

### 5.7 i18n

Claves nuevas: `statement_interest_bonifiable`, `statement_mora`, `statement_arrears`, `statement_over_limit`, `statement_points`, `statement_reconciliation_cash`, `statement_reconciliation_balance`, `statement_reconciliation_ok`, `statement_reconciliation_diff`, `import_unsupported_help`.

### 5.8 Errores de dominio

| Código | HTTP | Cuándo |
|---|---|---|
| `unsupported_statement` | 400 | Ningún importador reconoce el layout |
| `unsupported_statement` | 400 | Módulo del banco no registrado (mensaje distinto, mismo código) |
| `card_mismatch` | 409 | `last4` impreso ≠ tarjeta destino |
| `statement_exists` | 409 | Mismo `(card_id, period_end)` |

---

### 5.9 Notas de implementación

Decisiones tomadas durante la implementación (validadas contra un PDF real de BANPRO, procesado en memoria y no persistido):

1. **Codificación heredada.** El PDF de BANPRO no trae `ToUnicode`: los glifos llegan desplazados 31 posiciones y los dígitos como caracteres de control. Se agrega `statementtext.DecodeLegacyEncoding`, que detecta la corrupción (> 5 caracteres de control no blancos) y desplaza solo en ese caso. Los acentos sin mapeo se descartan (`MEMBRESIA ANUAL SEGUNDO AÑO` → `... SEGUNDO AO`); es preferible a romper las anclas.
2. **Una sola fecha de pago.** BANPRO imprime únicamente la fecha límite de pago mínimo. `full_payment_due_date` se completa con la fecha de corte (el pago de contado vence al corte) y se emite el warning `single_payment_date`.
3. **BAC no completa los agregados nuevos.** Su importador procesa el PDF «Transacciones del periodo», que no imprime bonificable, mora ni exceso: quedan en `0` y la UI los oculta. El campo existe en el contrato para cuando ese layout los incluya.
4. **Bases aplicables.** La conciliación marca `cash_applicable` / `balance_applicable` solo cuando el banco imprimió el total correspondiente; un estado manual sin `total_payment` no reporta un descuadre falso.
5. **Firma del resolver.** `Registry.ParseStatements` recibe `[]byte` (el handler ya lee y valida el PDF con límite de 5 MB) y `PreviewImport` cambió de `io.Reader` a `[]byte`.
6. **Verificación real.** Con un estado BANPRO de 4 movimientos: cargos y abonos coinciden al centavo, `previous + cargos − abonos = pago de contado`, `+ interés bonificable = saldo al corte`, sin warnings de conciliación.

---

## 6. Criterios de Aceptación

### 6.1 Funcionales

- **CA-REQ-001**: Importar un estado con `interest_bonifiable > 0` y verificar que `expected_balance − printed_balance = 0` y que `balanced = true` (fixture sintético con la misma estructura de ambos bancos).
- **CA-REQ-002**: Un estado sin los agregados de mora se guarda con `mora_amount = 0`, `arrears_count = 0`, `over_limit_amount = 0` y la UI los oculta.
- **CA-REQ-003**: Con dos importadores registrados, un PDF BAC se parsea con BAC y uno BANPRO con BANPRO (test de integración con fixtures sintéticos).
- **CA-REQ-004**: Un PDF que no es de ningún banco devuelve `unsupported_statement` y no deja datos.
- **CA-REQ-005**: Una fila `12/28` con corte `11/17` se imputa al año anterior (test unitario del inferidor de año).
- **CA-REQ-006**: «CARGO DE INTERESES POR MORA» → `interest_late`; «CARGO ADMINISTRATIVO DE MORA» → `fee`; «CREDITO TRANSFERIDO…» negativo → `payment` (test unitario).
- **CA-REQ-007**: El confirm de un estado con un abono desconocido no falla (se importa como `payment` u `other`).

### 6.2 Reglas de UI/UX

- **CA-UI-SELECT**: `npm run check:ui` pasa; sin `<select>` nativo.
- **CA-UI-DARKMODE**: detalle, formulario y preview verificados en darkmode.
- **CA-UX-FEEDBACK**: el error de layout desconocido muestra el mensaje accionable, no el texto crudo del parser.
- **CA-UX-DETAIL-READONLY**: el detalle no incorpora inputs.

### 6.3 Datos y seguridad

- **CA-DATA-MIGRATIONS**: `0050` sube y baja sin pérdida.
- **CA-DATA-PRODCOPY**: migración y conciliación probadas sobre copia en `/tmp`, borrada al terminar.
- **CA-SEC-NO-PII**: spec, tests y fixtures sin números, titulares, saldos ni referencias reales.
- **CA-SEC-SYNTHETIC**: fixtures con `XXXX-XXXX-XXXX-0000`, «Banco Demo», referencia `00000000000000000000000`.

### 6.4 Testing

- **CA-TEST-LOCAL**: `go build ./...`, `go test ./...`, `check:ui` pasan.
- **CA-TEST-USER-FLOW**: el usuario confirma la importación de sus dos estados antes de `pending_release`.

---

## 7. Plan de Implementación

### 7.1 Fases

1. **Contrato y resolver**: `ErrUnsupportedStatement`, `ParseStatements`, ajuste del importador BAC para envolver el error.
2. **Backend del estado**: migración `0050`, columnas, conciliación de dos bases y tests con fixtures sintéticos de ambos layouts.
3. **Importador BANPRO**: parser + clasificador + tests (incluida inferencia de año).
4. **Frontend**: detalle con dos bases y agregados, formulario manual, preview y mensajes.
5. **i18n y build**; migración y conciliación sobre copia de producción en `/tmp`.
6. **Prueba manual** con los dos estados reales (PDF en memoria, nunca en el repo).

### 7.2 Milestones

| Hito | Criterio |
|---|---|
| M1 | Resolver multi-importador con tests |
| M2 | Migración `0050` reversible y conciliación de dos bases cerrando en ambos layouts |
| M3 | Importador BANPRO con preview correcto sobre fixture sintético |
| M4 | Detalle y formulario completos, darkmode verificado |
| M5 | Importación real de los dos estados confirmada por el usuario |

---

## 8. Riesgos y Mitigaciones

| Riesgo | Impacto | Mitigación |
|---|---|---|
| El layout BANPRO cambia entre ciclos o productos | Alto | Anclas tolerantes, `unsupported_statement` accionable y fixture que se actualiza con cada variante detectada |
| Inferencia de año falla en el corte de enero | Medio | Test unitario explícito del borde diciembre/enero |
| Dos parseos por request encarecen el preview | Bajo | Se intenta primero el importador cuyo `CanParse` coincide con el banco de la tarjeta, luego el resto |
| La nueva conciliación rompe la UI existente | Medio | Se actualiza el detalle en la misma spec y se prueba con un estado ya importado (copia de producción en `/tmp`) |
| `external_ref` con referencia bancaria duplica movimientos históricos | Medio | Si el estado ya existe (`statement_exists`), no se reimporta; para estados viejos se mantiene el hash |
| El clasificador común entre bancos acopla los módulos | Bajo | El paquete común solo expone funciones puras de concepto → tipo (`ARCH-MODULES` permite reuso de utilidades) |

---

## 9. Notas y Referencias

- **Spec previa**: `SPEC-137` (modelo de tarjeta/estado, parser BAC). Esta spec **reemplaza** su `REQ-007` (conciliación) y amplía su `REQ-014`–`REQ-017`.
- **Spec relacionada**: `SPEC-138` (formulario de tarjeta con límite en una moneda y semántica de tasas), creada a partir de los mismos dos estados de cuenta.
- **Esfuerzo estimado**: contrato + resolver 0,5 día; backend/conciliación 1 día; parser BANPRO 1,5 día; frontend 1 día; verificación 0,5 día.

---

## 10. Historial de Cambios

| Fecha | Versión | Cambio |
|---|---|---|
| 2026-11-18 | 1.0 | Creación inicial a partir de dos estados de cuenta reales (BAC y BANPRO) |
| 2026-11-18 | 1.1 | Implementación: resolver multi-importador, conciliación de dos bases, agregados, importador BANPRO y `DecodeLegacyEncoding`. Estado → `in_progress` |

---

## 11. Verificación de release

- [ ] Migración `0050` aplicada y revertida sobre copia de producción en `/tmp`.
- [ ] `go build ./...` y `go test ./...` pasan, incluidos fixtures de ambos layouts.
- [ ] `npm run check:ui` pasa y darkmode verificado.
- [ ] 🤖 Importación de los dos estados reales confirmada por el usuario.
- [ ] `git diff --cached` sin PII, secretos, DBs ni PDFs.
- [ ] Label `spec/released` y issue cerrado; tracker actualizado.
