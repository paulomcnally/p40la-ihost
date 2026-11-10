---
title: "Módulo de bancos: cuentas y formatos de transferencia (Transferencia y ACH) con plantillas JSON"
id: "SPEC-133"
status: "in_progress"
author: "opencode"
created: "2026-11-09"
updated: "2026-11-09"
github_issue: 136
---

# Módulo de bancos: cuentas y formatos de transferencia (Transferencia y ACH) con plantillas JSON

**ID**: SPEC-133  
**Estado**: in_progress  
**Autor**: opencode  
**Creado**: 2026-11-09  
**Actualizado**: 2026-11-09

---

## 1. Resumen Ejecutivo

El proyecto ya tiene deudas y cuotas (SPEC-054) y plantillas de información de pago con variables dinámicas (SPEC-132). Falta el eslabón que describe **dónde y cómo se transfiere el dinero**: un módulo de bancos con sus cuentas y los formatos de transferencia que usan los bancos locales.

Esta spec define un **CRUD de bancos** donde cada banco puede tener **N cuentas bancarias** y cada cuenta puede tener hasta **dos formatos**: `Transferencia` y `Transferencia ACH`. Cada formato se guarda como un **JSON de pares llave/valor** (por ejemplo `from`, `to`, `bank`, `beneficiary`, `amount`, `currency`, `description`). Esos valores pueden ser literales o plantillas con variables de SPEC-132, de modo que un sistema interno —hoy manual y mañana automático— sepa exactamente qué propiedad inyectar en cada campo. El módulo no ejecuta transferencias: es la fuente de verdad de la estructura y de los datos de origen/destino.

La primera versión es solo lectura/escritura y copia del JSON resuelto. La asociación de una cuenta + formato a una deuda queda como P1, y la ejecución automática de pagos como P2. En el iHost el impacto es mínimo: tres tablas nuevas en SQLite (migración 0047), JSON en una columna TEXT, sin dependencias nuevas y con el mismo patrón de UI (cards, menú de 3 puntos, `Select`, modales de confirmación).

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: CRUD de bancos con: nombre (único, sin distinguir mayúsculas), código corto opcional, país opcional, notas opcionales y estado activo/inactivo.
2. **REQ-002**: Cada banco puede tener N cuentas bancarias. Datos de la cuenta: alias/etiqueta, número de cuenta, tipo (`savings` ahorro, `checking` corriente), moneda (FK a `currencies`), beneficiario y notas opcionales.
3. **REQ-003**: Cada cuenta puede tener hasta dos formatos de transferencia: `transfer` (Transferencia) y `ach` (Transferencia ACH). No se permiten dos formatos del mismo tipo para la misma cuenta.
4. **REQ-004**: Cada formato se guarda como un objeto JSON de pares llave/valor en `fields_json` (columna TEXT). Las llaves son los nombres de propiedad que el sistema interno va a reemplazar; los valores son literales o plantillas con variables.
5. **REQ-005**: Al crear un formato, sugerir llaves por tipo:
   - `transfer`: `from`, `to`, `amount`, `description`, `beneficiary`.
   - `ach`: `from`, `to`, `bank`, `beneficiary`, `amount`, `currency`, `description`.
   El usuario puede agregar, quitar y renombrar llaves libremente.
6. **REQ-006**: Los valores de las llaves pueden contener variables con la sintaxis `{clave_en_ingles}` de SPEC-132. Se agregan variables propias del contexto bancario: `bank_name`, `bank_short_code`, `account_alias`, `account_number`, `account_type`, `account_beneficiary`, `account_currency`, `account_currency_code`, `account_currency_symbol`.
7. **REQ-007**: Validar llaves y valores: llave con formato válido y única dentro del formato, valor de hasta 500 caracteres, JSON total de hasta 16 KB, variables desconocidas o mal formadas rechazadas con mensaje claro.
8. **REQ-008**: Vista previa del JSON del formato con los valores resueltos (variables de cuenta) y acción "Copiar JSON". En P0 las variables de deuda/cuota de SPEC-132 quedan visibles sin resolver si no hay contexto de deuda; se resuelven cuando exista asociación (P1).
9. **REQ-009**: Módulo "Bancos" como sección propia del sidebar, con listado de bancos (cards + menú de acciones de 3 puntos + `EmptyCard`), detalle del banco con sus cuentas y formulario de cuenta con las dos pestañas de formato.
10. **REQ-010**: Eliminar un banco con cuentas debe bloquearse (primero se eliminan las cuentas). Eliminar una cuenta elimina sus formatos en cascada, siempre con confirmación por palabra (`DeleteModal` con `confirmo`).
11. **REQ-011**: i18n completo en `frontend/public/i18n/es.json` y `frontend/public/i18n/en.json`; el build sincroniza a `frontend/src/i18n` (`sync-i18n.cjs`). Sin textos hardcodeados en la UI.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-012**: Asociar una cuenta bancaria + formato de transferencia a la forma de pago de una deuda (nuevo campo/tipo o selector en `DebtPaymentInformationModal`), para que la información de pago de la cuota incluya el JSON listo para usar.
2. **REQ-013**: Vista previa del formato con contexto real de deuda y cuota (selector de deuda + cuota), resolviendo también las variables de SPEC-132.
3. **REQ-014**: Duplicar un formato existente (mismas llaves/valores) hacia el otro tipo de la misma cuenta, con ajuste de llaves sugeridas.
4. **REQ-015**: Marcar una cuenta y un formato como predeterminados del banco.
5. **REQ-016**: Exportar el JSON del formato a archivo `.json` además de copiarlo.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-017**: Ejecutar/encolar transferencias usando cuenta + formato + valores resueltos (requiere integración con un sistema de pagos; fuera del alcance de esta spec).
2. **REQ-018**: Importar un formato pegando un JSON crudo, con validación y mapeo de llaves.
3. **REQ-019**: Historial de versiones de los formatos (quién y cuándo cambió cada llave/valor).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: listados paginados o con límite razonable (bancos ≤ 100, cuentas ≤ 50 por banco). La resolución de variables es una regex sobre textos cortos. Sin llamadas externas.
- **Almacenamiento**: migración 0047 con tres tablas (`banks`, `bank_accounts`, `bank_transfer_formats`). `fields_json` TEXT con tope de 16 KB por formato. Sin archivos en disco.
- **Seguridad**: todos los endpoints requieren sesión (mismo `authMiddleware`). Números de cuenta y beneficiarios son datos sensibles: no se loguean, no se incluyen en mensajes de error y no se commitean. Se muestran completos solo dentro de la app autenticada.
- **Privacidad**: specs, tests y ejemplos usan datos sintéticos (`Banco Demo`, `0000000001`, `Beneficiario Demo`, `COR`). Nunca datos reales.
- **iHost**: sin dependencias nuevas; SQLite con WAL; tres tablas e índices simples; sin workers ni caché.
- **UI**: inputs con tokens del tema (`bg-card`, `text-text`, `text-text-secondary`), darkmode verificado, `Select` del repo (nunca `<select>` nativo), modales conforme a SPEC-126.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Deudas y cuotas** (SPEC-054): `debts` y `debt_bills`; `debt_bills` ya tiene `installment_number`, `amount`, `due_date` y `currency`.
- **Plantillas de información** (SPEC-131): `debt_information_templates` y `debt_payment_information`; `fields_json` TEXT ya se usa para guardar definiciones estructuradas.
- **Variables dinámicas** (SPEC-132): catálogo en inglés con etiquetas en español, sintaxis `{clave}`, resolución en el frontend con contexto deuda + cuota.
- **Monedas**: tabla `currencies` y endpoints `/api/currencies` ya existentes; las cuentas pueden referenciar `currency_id`.
- **UI existente**: `HomesPage`/`InstitutionsPage` para listados con cards y menú de 3 puntos; `Select` para dropdowns; `DeleteModal`/`ConfirmWordModal` para confirmaciones; `sidebar` con items en `frontend/src/components/Sidebar.tsx`; `BACK_ROUTES` en `DashboardLayout.tsx`.
- **Migraciones**: última migración `0046_create_debt_information_templates`; la nueva es `0047`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Formatos como columnas fijas (`from`, `to`, `bank`, ...) | Consultables y tipados | Rígido: cada banco/ACH varía; el pedido es llave/valor libre | ❌ Rechazada |
| Formatos como JSON libre en `fields_json` | Flexible, sin migraciones por cada banco, coincide con el pedido | Requiere validación y UI de pares llave/valor | ✅ Seleccionada |
| Un formato por cuenta (un solo JSON) con variantes internas | Menos filas | No distingue Transferencia vs ACH, que es un requisito explícito | ❌ Rechazada |
| Dos filas por cuenta con `kind` (`transfer`/`ach`) | Modelo explícito, UNIQUE por cuenta+tipo, fácil de validar | Dos filas por cuenta | ✅ Seleccionada |
| Formatos a nivel banco (compartidos por cuentas) | Menos duplicación | Cada cuenta tiene su `to`/beneficiario; el pedido dice "cada una puede tener 2 formatos" | ❌ Rechazada |
| Asociación a deudas dentro de esta spec | Cierra el flujo completo | Aumenta el alcance y toca SPEC-131/132; el usuario eligió CRUD primero | ⏸ P1 |
| Ejecución de transferencias dentro de esta spec | Automatización total | Requiere integración bancaria y seguridad; fuera del alcance | ⏸ P2 |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: JSON llave/valor libre con sugerencias por tipo.
- **Contexto**: los ejemplos de ACH y transferencia difieren en llaves (`bank`, `currency`).
- **Decisión**: `fields_json` guarda un objeto JSON; la UI parte de las llaves sugeridas y permite agregar/quitar/renombrar. Las llaves recomendadas son solo un punto de partida, no un esquema cerrado.
- **Consecuencias**: máxima flexibilidad; la validación debe cuidar duplicados, formato y tamaño. El orden de las llaves no es semántico; la UI muestra primero las sugeridas y luego las personalizadas.

**ADR-002**: Los valores son plantillas, no datos resueltos.
- **Contexto**: el sistema interno llenará `from`, `amount`, etc. en el futuro.
- **Decisión**: guardar literales o `{variable}` en el valor; no guardar nunca el JSON resuelto. La resolución es de solo lectura.
- **Consecuencias**: el mismo formato sirve para cualquier cuota/deuda; la copia del JSON resuelto es una acción explícita del usuario.

**ADR-003**: Variables de dos contextos (cuenta y deuda/cuota).
- **Contexto**: los formatos necesitan datos de la cuenta (`account_number`, `beneficiary`) y, más adelante, del pago (`installment_number`, `amount`).
- **Decisión**: un contexto combinado en el frontend: variables `account_*`/`bank_*` siempre disponibles; variables de deuda/cuota de SPEC-132 disponibles cuando hay deuda/cuota seleccionada. El backend valida contra la unión de catálogos.
- **Consecuencias**: P0 puede previsualizar y copiar con datos de cuenta; P1 agrega el contexto de deuda sin cambiar el formato guardado.

**ADR-004**: Borrado restrictivo de bancos.
- **Contexto**: borrar un banco con cuentas podría eliminar datos sensibles y formatos por accidente.
- **Decisión**: `DELETE /api/banks/{id}` falla si tiene cuentas; el usuario debe borrar las cuentas primero. El borrado de cuenta sí elimina sus formatos en cascada, con confirmación por palabra.
- **Consecuencias**: más pasos para el usuario, pero menos riesgo de pérdida accidental.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Sidebar: Bancos]
   │
   ▼
[BanksPage] ── CRUD ──► /api/banks
   │
   ▼
[BankDetailPage] ── CRUD cuentas ──► /api/banks/{id}/accounts
   │
   ▼
[BankAccountFormPage] ── pestañas Transferencia / ACH
   │            │
   │            └── JSON llave/valor ──► /api/bank-accounts/{id}/transfer-formats/{kind}
   │
   ▼
[Vista previa + Copiar JSON] ── resuelve variables {account_*} y (P1) {installment_*}
   │
   ▼
[SQLite: banks, bank_accounts, bank_transfer_formats]
```

### 4.2 Componentes

#### 4.2.1 Modelo y migración
- **Responsabilidad**: persistir bancos, cuentas y formatos.
- **Interfaz**: tablas `banks`, `bank_accounts`, `bank_transfer_formats`.
- **Dependencias**: `currencies` (FK).
- **Ubicación**: `migrations/0047_create_banks.up.sql` / `.down.sql`, `internal/models/bank.go`.

#### 4.2.2 Storage y service
- **Responsabilidad**: consultas SQL y reglas de negocio (unicidad, validación de JSON, borrado restrictivo/cascada).
- **Interfaz**: `BankStorage`, `BankAccountStorage`, `BankTransferFormatStorage` o un único `BankStorage` con métodos por entidad; `BankService`.
- **Dependencias**: `database/sql`, `encoding/json`, validadores de variables de SPEC-132.
- **Ubicación**: `internal/storage/bank.go`, `internal/services/bank.go`.

#### 4.2.3 API
- **Responsabilidad**: handlers finos que delegan en el service.
- **Interfaz**: endpoints de la sección 4.5.
- **Dependencias**: `authMiddleware`.
- **Ubicación**: `internal/api/bank_handlers.go`, `internal/api/routes.go`.

#### 4.2.4 Frontend
- **Responsabilidad**: CRUD, editor de pares llave/valor, sugerencias, vista previa y copia.
- **Interfaz**: `BanksPage`, `BankFormPage`, `BankDetailPage`, `BankAccountFormPage`, `TransferFormatEditor`.
- **Dependencias**: `Select`, `Icon`, `DeleteModal`/`ConfirmWordModal`, `useToast`, `api`.
- **Ubicación**: `frontend/src/pages/`, `frontend/src/components/`, `frontend/src/api/index.ts`, `frontend/src/types/index.ts`.

### 4.3 Modelo de datos

Migración `0047_create_banks`:

```
Entidad: banks
- id: INTEGER PK AUTOINCREMENT
- name: TEXT NOT NULL UNIQUE COLLATE NOCASE
- short_code: TEXT NOT NULL DEFAULT ''
- country: TEXT NOT NULL DEFAULT ''
- notes: TEXT NOT NULL DEFAULT ''
- active: INTEGER NOT NULL DEFAULT 1
- created_at, updated_at: DATETIME

Entidad: bank_accounts
- id: INTEGER PK AUTOINCREMENT
- bank_id: INTEGER NOT NULL FK banks(id) ON DELETE RESTRICT
- alias: TEXT NOT NULL
- account_number: TEXT NOT NULL
- account_type: TEXT NOT NULL CHECK (account_type IN ('savings','checking'))
- currency_id: INTEGER NOT NULL FK currencies(id)
- beneficiary: TEXT NOT NULL
- notes: TEXT NOT NULL DEFAULT ''
- active: INTEGER NOT NULL DEFAULT 1
- created_at, updated_at: DATETIME
- UNIQUE(bank_id, account_number)
- INDEX(bank_id)

Entidad: bank_transfer_formats
- id: INTEGER PK AUTOINCREMENT
- bank_account_id: INTEGER NOT NULL FK bank_accounts(id) ON DELETE CASCADE
- kind: TEXT NOT NULL CHECK (kind IN ('transfer','ach'))
- fields_json: TEXT NOT NULL DEFAULT '{}'
- created_at, updated_at: DATETIME
- UNIQUE(bank_account_id, kind)
- INDEX(bank_account_id)
```

Ejemplo de `fields_json` (datos sintéticos):

```json
{
  "from": "0000000000",
  "to": "{account_number}",
  "bank": "{bank_name}",
  "beneficiary": "{account_beneficiary}",
  "amount": "{installment_amount}",
  "currency": "{account_currency}",
  "description": "Pago Lote Demo 13 - Cuota {installment_number}"
}
```

`account_number` es el número de la cuenta destino; `from` es la cuenta pagadora, que puede ser literal o una variable que el sistema interno complete. Ningún valor se resuelve ni se escribe en la DB.

### 4.4 Catálogo de variables del contexto bancario

Se suman a las variables de SPEC-132. Etiquetas en español, claves en inglés:

| Clave | Etiqueta sugerida (es) | Ejemplo |
|---|---|---|
| `bank_name` | Nombre del banco | `Banco Demo` |
| `bank_short_code` | Código corto del banco | `BDEMO` |
| `account_alias` | Alias de la cuenta | `Cuenta principal` |
| `account_number` | Número de cuenta | `0000000001` |
| `account_type` | Tipo de cuenta (localizado) | `Ahorro` |
| `account_beneficiary` | Beneficiario de la cuenta | `Beneficiario Demo` |
| `account_currency` | Moneda de la cuenta (código) | `COR` |
| `account_currency_symbol` | Símbolo de la moneda | `C$` |
| `account_notes` | Notas de la cuenta | `Cuenta para pagos` |

Sugerencias de llaves por tipo (no obligatorias):

- `transfer`: `from`, `to`, `amount`, `description`, `beneficiary`
- `ach`: `from`, `to`, `bank`, `beneficiary`, `amount`, `currency`, `description`

Reglas de validación de llaves: `^[a-z][a-z0-9_]{0,39}$` (minúsculas, números y guion bajo), sin duplicados, máximo 30 llaves por formato.

### 4.5 APIs / Contratos

Todos los endpoints requieren sesión autenticada y devuelven JSON.

#### Bancos

| Método | Ruta | Descripción |
|---|---|---|
| GET | `/api/banks` | Lista bancos con `account_count` |
| POST | `/api/banks` | Crea un banco |
| GET | `/api/banks/{id}` | Banco |
| PUT | `/api/banks/{id}` | Actualiza un banco |
| DELETE | `/api/banks/{id}` | Elimina si no tiene cuentas; si tiene, `409 bank_has_accounts` |

#### Cuentas

| Método | Ruta | Descripción |
|---|---|---|
| GET | `/api/banks/{id}/accounts` | Lista cuentas del banco |
| POST | `/api/banks/{id}/accounts` | Crea una cuenta |
| GET | `/api/bank-accounts/{id}` | Cuenta + formatos |
| PUT | `/api/bank-accounts/{id}` | Actualiza una cuenta |
| DELETE | `/api/bank-accounts/{id}` | Elimina cuenta y sus formatos |

#### Formatos de transferencia

| Método | Ruta | Descripción |
|---|---|---|
| GET | `/api/bank-accounts/{id}/transfer-formats` | Lista los formatos (`transfer`, `ach`) |
| PUT | `/api/bank-accounts/{id}/transfer-formats/{kind}` | Crea o actualiza el formato (`upsert`) |
| DELETE | `/api/bank-accounts/{id}/transfer-formats/{kind}` | Elimina el formato |

#### Catálogo de variables

`GET /api/bank-transfer-variables` devuelve la unión de variables bancarias y de SPEC-132:

```json
[
  { "key": "account_number", "label": "Número de cuenta", "example": "0000000001", "description": "Número de la cuenta bancaria" },
  { "key": "installment_number", "label": "Número de cuota", "example": "5", "description": "Número de la cuota que se está pagando" }
]
```

#### Contrato de formato

**PUT** `/api/bank-accounts/1/transfer-formats/ach`

```json
{
  "fields": {
    "from": "0000000000",
    "to": "{account_number}",
    "bank": "{bank_name}",
    "beneficiary": "{account_beneficiary}",
    "amount": "{installment_amount}",
    "currency": "{account_currency}",
    "description": "Pago Lote Demo 13 - Cuota {installment_number}"
  }
}
```

**Response 200**:

```json
{
  "id": 10,
  "bank_account_id": 1,
  "kind": "ach",
  "fields": { "from": "{account_number}", "to": "00000000000001" },
  "created_at": "2026-11-09T12:00:00Z",
  "updated_at": "2026-11-09T12:00:00Z"
}
```

**Response Error**:

```json
{
  "error": "invalid_variable",
  "message": "La variable {installment_numer} no es válida en el campo 'description'."
}
```

Errores previstos: `validation_error`, `invalid_variable`, `bank_not_found`, `account_not_found`, `format_not_found`, `duplicate_account`, `bank_has_accounts`, `duplicate_format`.

### 4.6 Frontend: rutas y UI

- **Sidebar**: nuevo item `banks` con ícono `bank`, etiqueta `menu.banks` ("Bancos"/"Banks"), entre Deudas e Instituciones.
- **Rutas**:
  - `/bancos` — listado de bancos.
  - `/bancos/new`, `/bancos/edit/:id` — formulario de banco.
  - `/bancos/:id` — detalle del banco con cuentas.
  - `/bancos/:id/cuentas/new`, `/bancos/:id/cuentas/:accountId` — formulario de cuenta con pestañas Transferencia / ACH.
- **BACK_ROUTES** (en `DashboardLayout.tsx`):
  - `/bancos/(new|edit/:id)` → `/bancos`
  - `/bancos/:id` → `/bancos`
  - `/bancos/:id/cuentas/(new|:accountId)` → `/bancos/:id`
- **Listados**: cards con menú de 3 puntos y `EmptyCard`; nunca formularios inline.
- **Formulario de cuenta**: datos de la cuenta + dos pestañas de formato. Cada pestaña muestra filas llave/valor con: llave (input), valor (input), botón de variables (panel con etiquetas en español y clave inglesa) y botón de eliminar fila. Botón "Agregar campo" y "Restaurar sugeridas".
- **Vista previa**: bloque `bg-bg/60` con el JSON resuelto y botón "Copiar JSON". En P0 resuelve variables de cuenta; deja las de deuda/cuota visibles tal cual si no hay contexto.
- **Selects**: tipo de cuenta y moneda con el componente `Select` (moneda `searchable`).
- **Modales**: confirmaciones de borrado con `DeleteModal`/`ConfirmWordModal` (palabra `confirmo`, foco automático, Enter, sin cierre por click fuera, `z-[60]`/`z-[80]` según corresponda).
- **Darkmode**: verificar inputs, placeholders, pestañas, chips de variables y JSON.

### 4.7 i18n

Editar **`frontend/public/i18n/es.json`** y **`frontend/public/i18n/en.json`** (fuente de verdad). El build ejecuta `sync-i18n.cjs` y copia a `frontend/src/i18n`. Claves nuevas mínimas: `menu.banks`, `banks.title`, `banks.empty_*`, `banks.create`, `banks.edit`, `banks.name`, `banks.short_code`, `banks.country`, `banks.notes`, `banks.accounts`, `banks.account_*`, `banks.transfer`, `banks.ach`, `banks.format_*`, `banks.variables`, `banks.preview`, `banks.copy_json`, `banks.validation_*`, `banks.delete_*`.

### 4.8 Dependencias

- **Internas**: SPEC-132 (variables), SPEC-054 (deudas/cuotas), SPEC-126 (modales), SPEC-118/004 (Select/UI).
- **Externas**: ninguna.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Se puede crear, listar, editar y eliminar un banco; el nombre duplicado (sin distinguir mayúsculas) se rechaza.
- [ ] CA-002: Un banco con cuentas no se puede eliminar; la API devuelve `409 bank_has_accounts` y la UI muestra el motivo.
- [ ] CA-003: Se pueden crear N cuentas por banco con alias, número, tipo, moneda y beneficiario; el número duplicado dentro del mismo banco se rechaza.
- [ ] CA-004: Cada cuenta permite guardar un formato `transfer` y un formato `ach`; un segundo formato del mismo tipo se rechaza (`duplicate_format`).
- [ ] CA-005: Al crear un formato se muestran las llaves sugeridas del tipo y se pueden agregar, quitar y renombrar llaves.
- [ ] CA-006: El JSON se persiste exactamente como pares llave/valor en `fields_json`; al reabrir el formato se recupera igual.
- [ ] CA-007: Una llave duplicada, inválida o un JSON que supere 16 KB se rechaza con un mensaje claro.
- [ ] CA-008: Los valores aceptan variables `{account_number}`, `{bank_name}`, `{installment_number}`, etc.; una variable desconocida se rechaza en UI y API.
- [ ] CA-009: "Vista previa" muestra el JSON con las variables de cuenta resueltas y "Copiar JSON" copia ese contenido.
- [ ] CA-010: Eliminar una cuenta pide confirmación por palabra (`confirmo`), enfoca el input, acepta Enter y elimina también sus formatos.
- [ ] CA-011: El listado de bancos usa cards con menú de 3 puntos y `EmptyCard` cuando no hay registros.
- [ ] CA-012: Las rutas de detalle/formulario muestran flecha atrás en el header (registradas en `BACK_ROUTES`), nunca un link "← Título" en el contenido.
- [ ] CA-013: Todos los dropdowns usan el componente `Select`; `npm run check:ui` pasa.
- [ ] CA-014: Todos los textos visibles están en i18n (`es`/`en`), sin hardcodeos.
- [ ] CA-DARK: Inputs, placeholders, pestañas, chips y JSON usan tokens del tema y son legibles en darkmode.
- [ ] CA-MODAL: Los modales de confirmación no se cierran por click fuera, el overlay bloquea el fondo, el cierre visible existe en todos los estados, se respeta la escala de capas y la confirmación por palabra enfoca el input y acepta Enter.
- [ ] CA-BACK: Las rutas nuevas están registradas en `BACK_ROUTES` y no hay links de retorno dentro del contenido.

### 5.2 No funcionales

- [ ] CA-NF-001: Migración 0047 `up`/`down` idempotente y reversible; `PRAGMA integrity_check` OK.
- [ ] CA-NF-002: `go test ./...` pasa y `npm run build` (incluye `check:ui`, `check:pwa`, `tsc -b`) pasa.
- [ ] CA-NF-003: Sin dependencias nuevas ni llamadas de red externas.
- [ ] CA-NF-004: Números de cuenta y beneficiarios no aparecen en logs ni en mensajes de error; los tests usan datos sintéticos.
- [ ] CA-NF-005: Validación de datos con una copia de la DB de producción si el comportamiento depende de datos reales (listados, monedas existentes); nunca contra `data/app.db` local.

### 5.3 Testing

- **Unit tests (Go)**: validación de banco/cuenta/formato (nombres, duplicados, `kind`, llaves, variables, tamaño JSON); borrado restrictivo/cascada.
- **Integration tests (Go)**: CRUD completo de banco, cuenta y formatos; errores 409 y 404; catálogo de variables.
- **Manual (frontend)**: crear un banco con dos cuentas, guardar los dos formatos, verificar sugerencias, variables, vista previa, copia, borrados con confirmación, darkmode y responsive.
- **Performance**: no aplica carga; verificar que el listado y el editor responden rápido en el iHost.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración 0047, modelos y storage (`banks`, `bank_accounts`, `bank_transfer_formats`) | 0.5 día | SPEC-132 para variables |
| 2 | Service con validaciones, borrado restrictivo/cascada y catálogo de variables | 0.5 día | Fase 1 |
| 3 | Handlers y rutas API | 0.5 día | Fase 2 |
| 4 | Frontend: API client, tipos, sidebar, rutas y `BACK_ROUTES` | 0.5 día | Fase 3 |
| 5 | Páginas de bancos, cuentas y editor de formatos con vista previa/copia | 1.5 días | Fase 4 |
| 6 | i18n, build, pruebas manuales y darkmode | 0.5 día | Fase 5 |

### 6.2 Milestones

1. **MVP**: CRUD de bancos y cuentas + formatos JSON con sugerencias, validación y copia.
2. **V1.0**: vista previa con variables de cuenta, i18n, darkmode y pruebas completas.
3. **P1**: asociación de cuenta + formato a la forma de pago de una deuda y vista previa con cuota real.
4. **P2**: ejecución de transferencias e historial de versiones.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Exposición de números de cuenta/beneficiarios en logs o errores | Media | Alto | No loguear payloads; mensajes genéricos; tests con datos sintéticos |
| Formatos demasiado grandes o con llaves basura | Media | Medio | Límite de 30 llaves y 16 KB; validación de patrón y duplicados |
| Divergencia entre las llaves sugeridas y lo que espera el banco | Alta | Medio | Sugerencias editables y documentación por banco en `notes`; importación JSON (P2) |
| Borrado accidental de cuentas/formatos | Media | Alto | Banco restrictivo, cuenta con confirmación por palabra y cascada documentada |
| Dependencia de SPEC-132 para variables | Media | Medio | Si SPEC-132 no está implementada, el editor acepta solo variables de cuenta y muestra las de deuda como P1 |
| Conflictos de i18n entre `public/i18n` y `src/i18n` | Baja | Medio | Editar solo `frontend/public/i18n` y correr el build con `sync-i18n.cjs` |
| Multi-moneda mal manejada en ACH (`currency`) | Media | Medio | `currency_id` en la cuenta + llave `currency` con variable `{account_currency}` |

## 8. Notas y Referencias

- SPEC-132: `docs/specs/SPEC-132-variables-dinamicas-plantillas-informacion-deudas.md` (variables `{clave}` en inglés).
- SPEC-131: plantillas de información de deudas.
- SPEC-054: deudas y cuotas (`debts`, `debt_bills`).
- SPEC-126: reglas de modales y confirmación por palabra.
- SPEC-004/118: `Select` obligatorio y UI iOS.
- Código de referencia:
  - `frontend/src/pages/HomesPage.tsx`, `frontend/src/pages/InstitutionsPage.tsx`
  - `frontend/src/components/Select.tsx`, `DeleteModal.tsx`, `ConfirmWordModal.tsx`
  - `frontend/src/components/Sidebar.tsx`, `DashboardLayout.tsx`
  - `internal/api/routes.go`, `internal/services/debt_information.go`
  - `migrations/0046_create_debt_information_templates.up.sql`

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-09 | opencode | Creación inicial de la especificación |
| 2026-11-09 | opencode | Implementación: migración 0047, CRUD de bancos/cuentas, formatos Transferencia/ACH en JSON llave/valor, catálogo de variables bancarias, UI con sección en el sidebar, listados, formularios y copia del JSON resuelto. Pendiente de validación manual. |
