---
title: "Formulario de tarjeta: límite en una moneda, fecha de bonificación y semántica de tasas"
id: "SPEC-138"
status: "draft"
author: "OpenCode"
created: "2026-11-18"
updated: "2026-11-18"
github_issue: 141
---

# Formulario de tarjeta: límite en una moneda, fecha de bonificación y semántica de tasas

**ID**: SPEC-138  
**Estado**: draft  
**Autor**: OpenCode  
**Creado**: 2026-11-18  
**Actualizado**: 2026-11-18

---

## 1. Resumen Ejecutivo

Al comparar el formulario de creación de tarjeta (SPEC-137) contra **estados de cuenta reales de dos bancos** (BAC Nicaragua y BANPRO) aparecen cuatro datos que el formulario hoy no puede capturar bien:

1. **El límite de crédito se publica en una sola moneda.** Los dos bancos revisados imprimen el límite únicamente en dólares, mientras la tarjeta opera en córdobas. El formulario pide *dos* límites (local y USD) sin decir cuál es el dato real, y el disponible local se calcula como `límite − saldo`; con el límite local en `0` el detalle muestra `C$ 0.00` de disponible, un dato falso que parece real.
2. **Las fechas límite pueden ser dos.** Uno de los estados publica dos fechas distintas: *límite para bonificar* (si pagás el total, te acreditan los intereses) y *límite para pagar* (si pagás el mínimo, no caés en mora). El otro publica una sola. El modelo de tarjeta solo tiene `payment_due_day`, así que la fecha de bonificación se pierde y la app no puede avisar «pagá antes de esta fecha para que te bonifiquen».
3. **La tasa moratoria impresa no es la tasa total.** En los dos casos revisados, la moratoria publicada es exactamente el **recargo** (la mitad de la corriente: 50 % → 25 %, 24 % → 12 %). El formulario la rotula «Tasa moratoria anual (%)» como si fuera el total; copiar el número impreso subestima la mora a la mitad.
4. **El % de pago mínimo no se publica.** Ninguno de los dos estados imprime un porcentaje: imprimen el monto de capital del pago mínimo y el pago mínimo total. El campo se llena a ojo y no hay ayuda para derivarlo.

El resultado esperado: crear una tarjeta copiando los datos del estado de cuenta sin inventar nada, con el límite expresado en la moneda que el banco publica, la fecha de bonificación cuando exista, las tasas con semántica explícita y los campos sin dato visibles como «no definido» en lugar de `0.00`.

**Fuera de alcance** (se propone como spec aparte): ampliar el modelo de **estados de cuenta** con los campos que estos dos bancos publican y la app no tiene (`interés bonificable`, `monto en mora`, `pagos atrasados`, `monto en exceso`), y el **importador BANPRO** (el parser actual solo entiende el layout «Transacciones del periodo» de BAC). Ver §9.

### Consideraciones de iHost

- Sin dependencias nuevas: solo Go stdlib, SQLite y React.
- Una migración aditiva reversible; sin reescritura de tablas.
- Sin cambios de rendimiento: 4 columnas nuevas en una tabla de decenas de filas.

---

## 2. Cumplimiento de reglas

Índice: `docs/rules/00-index.md`.

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Go + `net/http` + SQLite; React/Tailwind; sin deps nuevas | — |
| `ARCH-FOLDERS` | Los cambios quedan dentro de `internal/modules/bank/creditcard/`, `migrations/` y `frontend/src/modules/bank/creditcard/` | — |
| `ARCH-LAYERS` | El cálculo del límite derivado y las validaciones viven en `service.go`; `storage.go` solo persiste; el handler solo parsea | `CA-ARCH-LAYERS` |
| `ARCH-MODULES` | No se crean módulos nuevos: se extiende `bank.creditcard` (ID inmutable, sin cambio de versión mayor → `1.1.0`) | `CA-ARCH-MODULES` |
| `ARCH-GO` | Nombres en inglés, `gofmt`, errores explícitos, `log/slog` | — |
| `ARCH-IHOST` | Sin deps nuevas; migración de 4 columnas; sin Node en runtime | — |
| `ARCH-CONFIG` | N/A: no cambia `opencode.json` | — |
| `UI-SELECT` | Moneda del límite, categoría y color usan `Select` (`searchable` donde aplica); `check:ui` en el gate | `CA-UI-SELECT` |
| `UI-INPUT-TOKENS` | Inputs con `bg-card`/`text-text`; sin colores hardcodeados | `CA-UI-TOKENS` |
| `UI-DARKMODE` | Formulario y detalle verificados en darkmode, incluidos los textos de ayuda | `CA-UI-DARKMODE` |
| `UI-TOGGLE` | `active` sigue con `Toggle` iOS; no se agregan booleanos nuevos | `CA-UI-TOGGLE` |
| `UI-LIST-PATTERN` | El listado de tarjetas no cambia de patrón (cards + `CardMenu`) | — |
| `UI-CREATE-MENU` | N/A: no se agrega acción de crear | — |
| `UI-EDIT-LABEL` | N/A: no se agregan acciones nuevas; «Editar» sigue usando `app.edit` | — |
| `UI-I18N` | Claves nuevas en `frontend/public/i18n/{es,en}.json`; build y verificación por HTTP | `CA-UI-I18N` |
| `UI-GUARDRAIL` | `npm run check:ui` pasa; sin `<select>` ni checkbox nativo | `CA-UI-SELECT` |
| `UX-BACK` | N/A: no se agregan rutas; las existentes conservan su entrada en `BACK_ROUTES` | — |
| `UX-MODAL` | N/A: no se agregan modales | — |
| `UX-MODAL-CONFIRM` | N/A: no se agregan confirmaciones por palabra | — |
| `UX-DETAIL-ACTIONS` | N/A: no se agregan acciones al detalle | — |
| `UX-DETAIL-READONLY` | El detalle sigue en modo lectura; los campos nuevos se muestran, no se editan ahí | `CA-UX-DETAIL-READONLY` |
| `UX-PREREQ` | N/A: no cambian prerrequisitos (la tarjeta sigue requiriendo banco) | — |
| `UX-EMPTY` | N/A: no se agregan listados | — |
| `UX-LOADING` | N/A: el formulario ya bloquea el guardado con `saving` | — |
| `UX-FEEDBACK` | Error de validación junto al campo (tipo de cambio, límite, días); toast en éxito/error | `CA-UX-FEEDBACK` |
| `UX-FORMS` | Un Guardar + un Cancelar; la validación bloquea el guardado; campos nuevos marcados como opcionales | `CA-UX-FORMS` |
| `DATA-SQLITE` | SQL crudo en `storage.go`; sin ORM | — |
| `DATA-MIGRATIONS` | `migrations/0049_credit_cards_limit_currency_bonus_day.up.sql` + `.down.sql`, aditiva y reversible | `CA-DATA-MIGRATIONS` |
| `DATA-NO-DROP` | No se toca `data/app.db`; pruebas con DB temporal en `/tmp` | `CA-DATA-PRODCOPY` |
| `DATA-PRODCOPY` | Validación del `ALTER TABLE` contra copia de producción en `/tmp` antes de `pending_release` | `CA-DATA-PRODCOPY` |
| `DATA-PROD-WRITE` | N/A: la migración la aplica el arranque del server; no hay escritura manual | — |
| `DATA-PROTECTED-SCHEMA` | N/A: no se toca `debts`/`debt_bills` | — |
| `DATA-SSH` | N/A: no se requiere acceso al iHost para implementar | — |
| `SEC-NO-PII` | La spec describe **patrones** de los estados de cuenta; no transcribe números de tarjeta, nombres, saldos ni movimientos | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Ejemplos con `XXXX-XXXX-XXXX-0000`, «Banco Demo», `U$ 1.000,00`, tasas demo | `CA-SEC-SYNTHETIC` |
| `SEC-PRODCOPY` | Copias de DB solo en `/tmp`, borradas al terminar | `CA-SEC-PRODCOPY` |
| `SEC-COMMIT` | `git diff --cached` revisado; sin PDFs, DBs ni capturas con datos reales | `CA-SEC-COMMIT` |
| `SEC-HOOKS` | Hooks instalados en el worktree; sin `--no-verify` | `CA-SEC-HOOKS` |
| `SEC-INCIDENT` | N/A: no se manejan secretos nuevos | — |
| `SEC-LOGS` | Prohibido loguear montos, límites o tasas del usuario | — |
| `TEST-LOCAL` | `go build ./...`, `go test ./...`, `check:ui` y prueba manual | `CA-TEST-LOCAL` |
| `TEST-SERVER` | `./scripts/check-server.sh` antes de avisar al usuario | — |
| `TEST-KILLALL` | Reinicio del server con `killall p40la-ihost` | — |
| `TEST-TESTUSER` | Endpoints probados con `./scripts/create-test-user.sh` | — |
| `TEST-USER-FLOW` | 🤖 Se pregunta antes de correr el server y antes de cerrar la spec | — |
| `TEST-ROBOT` | Preguntas de flujo con 🤖 | — |
| `TEST-CHECKLIST` | Checklist completo antes de `pending_release` | `CA-TEST-LOCAL` |
| `DELIV-WORKTREE` | Sesión en `../p40la-ihost-spec-138` (rama `feature/SPEC-138`) | — |
| `DELIV-CONFIRM` | Merge y push solo con confirmación explícita | — |
| `DELIV-COMMIT` | Commits `SPEC-138: ...` | — |
| `DELIV-RELEASE` | Release con los 8 pasos; tracker y label en el release | — |
| `DELIV-CHECKRELEASE` | `/spec-check-release` antes de declarar terminado | — |
| `DELIV-HOOKS` | Hooks activos en el worktree | `CA-SEC-HOOKS` |
| `DELIV-DOCKER` | N/A: no cambia `Dockerfile` | — |
| `DELIV-LABELS` | Issue con `spec/draft` → `spec/released` | — |
| `DELIV-DEPLOY` | Deploy aprobado por el usuario; `/health` verificado | — |
| `DELIV-ESCALATION` | N/A al inicio; se activa si se bloquea > 3 días | — |

---

## 3. Requerimientos

### 3.1 Funcionales (P0 — obligatorios)

**Límite en una moneda**

1. **REQ-001**: La tarjeta guarda en qué moneda está expresado el límite (`limit_currency`, `NIO` o `USD`). El formulario pide **un** límite, en la moneda elegida, y no dos campos equivalentes.
2. **REQ-002**: El formulario acepta un **tipo de cambio de referencia** opcional (`exchange_rate`, córdobas por dólar). Si se informa, el límite de la otra moneda se **deriva en el backend** (`round2(límite × tipo de cambio)` o `round2(límite ÷ tipo de cambio)` según corresponda) y se persiste en `credit_limit` / `credit_limit_usd` como hoy.
3. **REQ-003**: Si no hay tipo de cambio, el límite de la otra moneda queda en `0` y la UI lo trata como **no definido** (§REQ-005). El backend nunca inventa un tipo de cambio.
4. **REQ-004**: Si el usuario ingresa explícitamente ambos límites, se respetan y no se deriva nada (el dato del usuario gana). La derivación solo completa el que está en `0`.
5. **REQ-005**: `available_local` y `available_usd` solo se muestran cuando el límite correspondiente es `> 0`. Si el límite es `0`, la UI muestra `—` con la etiqueta `creditCards.limit_undefined`; **prohibido** mostrar `C$ 0.00` / `$0.00` como disponible.

**Fechas de pago**

6. **REQ-006**: Nuevo campo opcional `bonus_due_day` (0–31; `0` = no aplica): día del mes en que vence el plazo para que el banco bonifique los intereses corrientes (pago de contado a tiempo).
7. **REQ-007**: El detalle de la tarjeta muestra «Límite para bonificar» solo si `bonus_due_day > 0`. Se distinguen visualmente las dos fechas: `payment_due_day` («Límite para pagar») y `bonus_due_day` («Límite para bonificar»).
8. **REQ-008**: `bonus_due_day` no puede ser igual a `payment_due_day` cuando ambos son `> 0` (validación con error junto al campo).

**Tasas**

9. **REQ-009**: `interest_late_rate` e `interest_late_rate_usd` se documentan y rotulan como **recargo moratorio** (porcentaje que se suma a la tasa corriente), no como tasa total. Etiquetas: «Recargo moratorio anual (%)» y «Recargo moratorio USD (%)», con texto de ayuda: *«Cargá el recargo que publica el banco (suele ser la mitad de la corriente). Si el banco publica la mora total, cargá total − corriente.»*
10. **REQ-010**: El detalle muestra la **tasa moratoria total** = corriente + recargo, con el desglose (`50 % + 25 % = 75 %`) y su equivalente en USD.
11. **REQ-011**: Sin migración de datos: los valores ya cargados se reinterpretan como recargo (es lo que publican los bancos). Se documenta en la migración y en el modelo (`model.go`).

**Pago mínimo**

12. **REQ-012**: `min_payment_pct` admite `0` = no definido; en ese caso se muestra `—`. Texto de ayuda: *«Si el estado de cuenta no publica el porcentaje, dejalo en 0. Podés estimarlo como capital del pago mínimo ÷ saldo al corte.»*
13. **REQ-013**: Validaciones del servicio: `limit_currency` ∈ {`NIO`,`USD`}; `exchange_rate` ≥ 0 y `> 0` si se usa para derivar; `bonus_due_day` 0–31; límites y tasas ≥ 0; `% pago mínimo` 0–100. Errores de dominio nuevos: `invalid_limit_currency`, `invalid_exchange_rate`, `invalid_bonus_day`.
14. **REQ-014**: La API devuelve los campos nuevos en `GET/POST/PUT /api/banks/{id}/cards` y en `GET /api/cards/{id}`, y expone los derivados `late_total_rate` y `late_total_rate_usd` (solo lectura) para que el frontend no recalcule.

### 3.2 Funcionales (P1 — importantes)

1. **REQ-015**: Campo opcional `category` (`classic`, `gold`, `platinum`, `black`, `business`, `other`): la categoría aparece en el nombre del producto de ambos bancos y define las tablas de seguros y el límite del producto.
2. **REQ-016**: Si el alias está vacío y hay marca + categoría, el formulario sugiere `«AMEX Dorada»` / `«Visa Platinum»` (editable, no se autoguarda).
3. **REQ-017**: El detalle muestra la categoría junto a la marca.

### 3.3 Funcionales (P2 — deseables)

1. **REQ-018**: `account_last4` opcional (últimos 4 de la cuenta), separado del plástico: hay bancos que identifican la cuenta y la tarjeta con números distintos, y el importador necesita saber cuál comparar.
2. **REQ-019**: Tarjetas adicionales agrupadas bajo una misma cuenta (una cuenta, N plásticos, un solo estado de cuenta).

### 3.4 No funcionales

- **Migración**: aditiva, sin rebuild de tabla, reversible con `ALTER TABLE ... DROP COLUMN`.
- **Compatibilidad**: tarjetas existentes siguen funcionando; `limit_currency` por defecto `NIO` y `bonus_due_day` por defecto `0` (no aplica).
- **Rendimiento**: sin impacto medible (4 columnas, sin índices nuevos).
- **Seguridad**: endpoints con `authMiddleware`; sin PAN/CVV/PIN; sin logs de montos.
- **iHost**: sin binarios externos, sin cgo, sin dependencias nuevas.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

Se revisaron **dos estados de cuenta reales de bancos distintos** (procesados en memoria, no persistidos ni citados; ver `SEC-NO-PII`):

| Observación | BAC Nicaragua | BANPRO |
|---|---|---|
| Moneda del límite impreso | Solo dólares | Solo dólares |
| Moneda de los movimientos | Córdobas + dólares | Córdobas (columna USD en 0) |
| Fechas límite | **Dos**: bonificar y pagar | **Una**: pagar antes de |
| Tasa corriente / mora impresa (local) | 49,92 % / 24,96 % | 50 % / 25 % |
| Tasa corriente / mora impresa (USD) | 24 % / 12 % | 30 % / 15 % |
| % pago mínimo impreso | No | No |
| Capital del pago mínimo impreso | Sí | Sí |
| Traslado por domingo/feriado | No documentado | Sí, al día hábil siguiente |

Patrón determinante: **la moratoria impresa es exactamente la mitad de la corriente en los cuatro pares de tasas**, lo que confirma que se publica como **recargo**, no como tasa total. Ningún banco publica el porcentaje de pago mínimo; ambos publican el monto de capital.

### 4.2 Opciones evaluadas

| Decisión | Opción A | Opción B | Elección |
|---|---|---|---|
| Límite en dos monedas | Mantener dos campos libres (status quo) | Un límite + moneda + tipo de cambio, con derivación | **B**: el dato real es uno; A obliga a inventar y produce disponibles falsos |
| Tipo de cambio | Configuración global del sistema | Por tarjeta | **B**: el banco aplica su propio tipo de cambio y la tarjeta puede diferir; una tasa global obligaría a un módulo de configuración que esta spec no necesita |
| Fecha de bonificación | Derivarla como `payment_due_day − N` | Campo propio opcional | **B**: los dos bancos no tienen un desfase fijo documentado |
| Semántica de la mora | Migrar los datos a tasa total | Reinterpretar como recargo y mostrar el total calculado | **B**: no reescribe datos del usuario y coincide con lo que publican los bancos |
| % pago mínimo | Obligatorio | `0` = no definido con ayuda para estimarlo | **B**: no se puede exigir un dato que el banco no publica |
| Categoría de tarjeta | Texto libre en notas | `Select` con catálogo cerrado | **B**: permite sugerir alias y, a futuro, validar tablas de seguros |

### 4.3 ADRs

- **ADR-001 — El límite tiene una moneda de origen.** Se persiste `limit_currency`; el otro límite es derivado y se marca como tal en la UI. Motivo: evita el disponible en `0.00` y hace explícito qué dato cargó el usuario.
- **ADR-002 — La derivación es del backend.** El `service` calcula el límite faltante con `round2`; el frontend no hace aritmética de dinero (`ARCH-LAYERS`).
- **ADR-003 — La mora se guarda como recargo.** No hay migración de datos: los valores cargados hoy ya son los publicados por los bancos. El total se calcula y se muestra.
- **ADR-004 — `bonus_due_day` a nivel tarjeta, fecha real a nivel estado de cuenta.** La tarjeta guarda el día habitual; el estado de cuenta seguirá guardando la fecha exacta (`full_payment_due_date`), porque los bancos la trasladan por domingos y feriados.

---

## 5. Diseño Técnico

### 5.1 Arquitectura

Sin cambios de arquitectura. Se extiende el módulo `bank.creditcard` (`ARCH-MODULES`):

```
internal/modules/bank/creditcard/
├── model.go       # +LimitCurrency, +ExchangeRate, +BonusDueDay, +Category, +LateTotalRate*
├── storage.go     # INSERT/UPDATE/SELECT con las columnas nuevas
├── service.go     # derivación del límite, validaciones, totales de mora
├── handlers.go    # payload sin cambios de forma (campos nuevos)
└── module.go      # versión 1.1.0

migrations/0049_credit_cards_limit_currency_bonus_day.{up,down}.sql
frontend/src/modules/bank/creditcard/pages/CreditCardFormPage.tsx
frontend/src/modules/bank/creditcard/pages/CreditCardShowPage.tsx
frontend/src/modules/bank/creditcard/types.ts
frontend/public/i18n/{es,en}.json
```

### 5.2 Modelo de datos

Migración `0049` (aditiva, `ALTER TABLE`):

```sql
ALTER TABLE credit_cards ADD COLUMN limit_currency TEXT NOT NULL DEFAULT 'NIO'
    CHECK (limit_currency IN ('NIO', 'USD'));
ALTER TABLE credit_cards ADD COLUMN exchange_rate REAL NOT NULL DEFAULT 0
    CHECK (exchange_rate >= 0);
ALTER TABLE credit_cards ADD COLUMN bonus_due_day INTEGER NOT NULL DEFAULT 0
    CHECK (bonus_due_day BETWEEN 0 AND 31);
ALTER TABLE credit_cards ADD COLUMN category TEXT NOT NULL DEFAULT ''
    CHECK (category IN ('', 'classic', 'gold', 'platinum', 'black', 'business', 'other'));
```

`down`: `ALTER TABLE credit_cards DROP COLUMN ...` para las cuatro columnas (precedente: `0041_create_pension_import.down.sql`).

Campos nuevos en `CreditCard`:

| Campo | Tipo | Semántica |
|---|---|---|
| `limit_currency` | `string` | Moneda en la que está expresado el límite que cargó el usuario |
| `exchange_rate` | `float64` | Córdobas por dólar usado para derivar el otro límite (`0` = no informado) |
| `bonus_due_day` | `int` | Día límite para bonificar (`0` = no aplica) |
| `category` | `string` | Categoría del producto (`""` = no definida) |
| `late_total_rate` | `float64` (derivado) | `interest_current_rate + interest_late_rate` |
| `late_total_rate_usd` | `float64` (derivado) | `interest_current_rate_usd + interest_late_rate_usd` |

### 5.3 API

`POST /api/banks/{bank_id}/cards` y `PUT /api/cards/{card_id}` aceptan:

```json
{
  "alias": "AMEX Dorada",
  "network": "amex",
  "last4": "0000",
  "holder": "Titular Demo",
  "currency_id": 1,
  "secondary_currency_id": 2,
  "limit_currency": "USD",
  "credit_limit": 0,
  "credit_limit_usd": 1000,
  "exchange_rate": 36.5,
  "statement_day": 15,
  "payment_due_day": 14,
  "bonus_due_day": 7,
  "min_payment_pct": 0,
  "interest_current_rate": 50,
  "interest_late_rate": 25,
  "interest_current_rate_usd": 30,
  "interest_late_rate_usd": 15,
  "category": "gold",
  "color": "gold",
  "icon": "credit",
  "active": true,
  "notes": ""
}
```

Regla de normalización (`service.Normalize`):

```
si limit_currency == "USD":
    si credit_limit == 0 y exchange_rate > 0: credit_limit = round2(credit_limit_usd * exchange_rate)
si limit_currency == "NIO":
    si credit_limit_usd == 0 y exchange_rate > 0: credit_limit_usd = round2(credit_limit / exchange_rate)
```

Errores nuevos: `invalid_limit_currency` (400), `invalid_exchange_rate` (400), `invalid_bonus_day` (400).

### 5.4 Frontend

**`CreditCardFormPage.tsx`**

- Sección *Límites*:
  - `Select` **Moneda del límite** (`NIO`/`USD`) a la izquierda.
  - Un solo input de límite rotulado según la moneda elegida (`Límite de crédito (C$)` / `Límite de crédito (US$)`).
  - Input **Tipo de cambio (C$ por US$)** con ayuda: *«Opcional. Si lo informás, calculamos el límite en la otra moneda.»*
  - Texto derivado, solo lectura: *«Límite equivalente: C$ 36.500,00 (derivado)»* cuando aplica.
  - `Día de corte`, `Límite para pagar`, `Límite para bonificar` (opcional, `0` = no aplica) y `% pago mínimo`.
- Sección *Tasas*: etiquetas «Recargo moratorio…», ayuda de §REQ-009 y línea calculada «Moratoria total: 75 %».
- Sección *Información*: `Select` **Categoría** (opcional) + sugerencia de alias (§REQ-016).
- Validación en cliente: `bonus_due_day ≠ payment_due_day` cuando ambos > 0; `exchange_rate > 0` si se usa.

**`CreditCardShowPage.tsx`**

- Métrica de límite: muestra solo la moneda de origen y, si existe, el equivalente marcado como derivado.
- Disponible: `—` + `creditCards.limit_undefined` cuando el límite es `0`.
- Fechas: «Límite para pagar» y (si aplica) «Límite para bonificar».
- Tasas: corriente, recargo y **total**.

**`CreditCardsTab.tsx`**: sin cambios funcionales; el subtítulo usa el límite de origen.

### 5.5 i18n

Claves nuevas en `creditCards.*` (es/en): `limit_currency`, `limit_amount`, `exchange_rate`, `exchange_rate_help`, `limit_derived`, `limit_undefined`, `bonus_due_day`, `bonus_due_day_help`, `payment_due_day_label`, `late_surcharge`, `late_surcharge_help`, `late_total`, `min_payment_pct_help`, `category`, `category_classic|gold|platinum|black|business|other`, `alias_suggestion`.

### 5.6 Errores de dominio

| Código | HTTP | Cuándo |
|---|---|---|
| `invalid_limit_currency` | 400 | `limit_currency` fuera de `NIO`/`USD` |
| `invalid_exchange_rate` | 400 | `exchange_rate < 0`, o `= 0` habiendo límite solo en la otra moneda y sin valor que derive |
| `invalid_bonus_day` | 400 | `bonus_due_day` fuera de 0–31 o igual a `payment_due_day` |

---

## 6. Criterios de Aceptación

### 6.1 Funcionales

- **CA-REQ-001**: Crear una tarjeta con `limit_currency=USD` y `credit_limit_usd=1000` sin tipo de cambio persiste `credit_limit=0` y el detalle muestra «—» en el disponible local.
- **CA-REQ-002**: Con `exchange_rate=36.5`, la misma tarjeta persiste `credit_limit=36500` y el detalle marca el valor como derivado.
- **CA-REQ-003**: Si el usuario ingresa ambos límites manualmente, la derivación no los sobrescribe (test unitario del servicio).
- **CA-REQ-004**: `available_local` nunca se renderiza como `0.00` cuando el límite es `0`.
- **CA-REQ-005**: `bonus_due_day=7` con `payment_due_day=14` muestra ambas fechas en el detalle; `bonus_due_day=14` devuelve `invalid_bonus_day`.
- **CA-REQ-006**: Con corriente 50 y recargo 25, el detalle muestra «Moratoria total 75 %».
- **CA-REQ-007**: `min_payment_pct=0` se muestra como «—» y guarda sin error.

### 6.2 Reglas de UI/UX

- **CA-UI-SELECT**: moneda del límite y categoría usan `Select`; `npm run check:ui` pasa.
- **CA-UI-TOKENS**: inputs con `bg-card`/`text-text`; sin colores hardcodeados.
- **CA-UI-DARKMODE**: formulario y detalle verificados en darkmode, incluidos los textos de ayuda.
- **CA-UX-FEEDBACK**: los errores de `bonus_due_day` y `exchange_rate` se muestran junto al campo.
- **CA-UX-FORMS**: un Guardar + un Cancelar; los campos opcionales están marcados.

### 6.3 Datos y seguridad

- **CA-DATA-MIGRATIONS**: `0049` sube y baja sin pérdida; `go test ./...` con DB en `/tmp`.
- **CA-DATA-PRODCOPY**: `ALTER TABLE` probado sobre copia de producción en `/tmp`, borrada al terminar.
- **CA-SEC-NO-PII**: la spec, los tests y los fixtures no contienen números de tarjeta, titulares, saldos ni movimientos reales.
- **CA-SEC-SYNTHETIC**: fixtures con `XXXX-XXXX-XXXX-0000`, «Banco Demo», montos demo.

### 6.4 Testing

- **CA-TEST-LOCAL**: `go build ./...`, `go test ./...` y `cd frontend && npm run check:ui` pasan.
- **CA-TEST-USER-FLOW**: el usuario confirma las pruebas manuales antes de `pending_release`.

---

## 7. Plan de Implementación

### 7.1 Fases

1. **Backend**: migración `0049`, campos en `model.go`, `storage.go`, derivación y validaciones en `service.go`, errores nuevos, tests unitarios.
2. **Frontend**: `types.ts`, `api` (payload), `CreditCardFormPage` (límite único, tipo de cambio, fechas, tasas, categoría), `CreditCardShowPage` (derivados y «—»).
3. **i18n**: claves es/en + `npm run build` + verificación por HTTP.
4. **Validación**: copia de producción en `/tmp` para la migración; prueba manual con una tarjeta real del usuario.

### 7.2 Milestones

| Hito | Criterio |
|---|---|
| M1 | Migración aplicada y reversible en `/tmp` |
| M2 | API con campos nuevos y derivación testeada |
| M3 | Formulario y detalle completos, darkmode verificado |
| M4 | Prueba manual del usuario con las dos tarjetas reales |

---

## 8. Riesgos y Mitigaciones

| Riesgo | Impacto | Mitigación |
|---|---|---|
| Reinterpretar `interest_late_rate` como recargo confunde datos ya cargados por el usuario | Medio | Texto de ayuda explícito y total calculado visible; el usuario revisa cada tarjeta en la prueba manual |
| El tipo de cambio queda desactualizado y el límite derivado se ve raro | Bajo | Se muestra marcado como derivado y se puede volver a editar |
| `ALTER TABLE ADD COLUMN` con `CHECK` en SQLite antiguo | Bajo | Probar en la versión del iHost antes de `pending_release` |
| `bonus_due_day` no aplica a todos los bancos | Bajo | `0` = no aplica y la UI no muestra la fecha |
| La spec no resuelve la importación de BANPRO | Medio | Declarado fuera de alcance; se propone SPEC-139 (§9) |

---

## 9. Notas y Referencias

- **Spec previa**: `docs/specs/SPEC-137-tarjetas-credito-estados-cuenta-modulos.md` (modelo de tarjeta, estados y parser BAC).
- **Propuesta de spec siguiente (SPEC-139)**: estados de cuenta multi-banco — `interest_bonifiable` (ambos bancos lo imprimen), `mora_amount` / `arrears_count` / `over_limit_amount` (BANPRO), y la diferencia entre *pago de contado* y *saldo al corte* que hoy rompería la conciliación (`REQ-007` de SPEC-137); más el importador del layout BANPRO (el parser actual exige anclas que ese banco no imprime).
- **Estimaciones de esfuerzo**: backend 1 día, frontend 1 día, i18n y verificación 0,5 día.

---

## 10. Historial de Cambios

| Fecha | Versión | Cambio |
|---|---|---|
| 2026-11-18 | 1.0 | Creación inicial a partir de dos estados de cuenta reales de BAC y BANPRO |

---

## 11. Verificación de release

- [ ] Migración `0049` aplicada sobre copia de producción en `/tmp` y revertida.
- [ ] `go build ./...` y `go test ./...` pasan.
- [ ] `npm run check:ui` pasa y darkmode verificado.
- [ ] 🤖 Pruebas manuales confirmadas por el usuario.
- [ ] `git diff --cached` sin PII, secretos, DBs ni PDFs.
- [ ] Label `spec/released` y issue cerrado; tracker actualizado.
