---
title: "Formato de moneda configurable con default Nicaragua (1,000.00)"
id: "SPEC-058"
status: "released"
author: "paulomcnally"
created: "2026-09-04"
updated: "2026-09-04"
github_issue: 58
---

# Formato de moneda configurable con default Nicaragua (1,000.00)

**ID**: SPEC-058  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-09-04  
**Actualizado**: 2026-09-04

---

## 1. Resumen Ejecutivo

Hoy la aplicación muestra los montos con formato plano: `C$1500.00`, sin separador de miles. Para un usuario en Nicaragua (y en general hispanohablante) el monto correcto es `C$1,500.00`: separador de miles con coma, decimales con punto y dos dígitos decimales. Esta spec introduce un formato de moneda **configurable por formato** (no por país), con **default Nicaragua**: miles `,`, decimales `.`, 2 dígitos decimales.

La configuración debe poder cambiarse desde Settings y persistir en el backend (`system_settings` / SQLite), porque los correos automáticos (facturas, resúmenes, deudas, pensión) se generan en Go y también deben respetar el formato. Se implementa una utilidad compartida en el frontend (`utils/currency.ts`) que reemplaza los ~30 puntos inline de `.toFixed(2)` y la función local `formatMoney` de `DebtAnalysis.tsx`.

Impacto iHost: la spec es de bajo costo — una función de formateo pura en el frontend, 3 claves nuevas en `system_settings`, y reemplazo de cadenas de formato en los builders de email. No agrega dependencias ni consumo de memoria relevante. Aplica el precedente del modo oscuro y hourFormat (config simple), pero persistiendo en backend porque el servidor de emails lo necesita.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Crear utilidad compartida de formato de moneda en el frontend (`frontend/src/utils/currency.ts`) con parámetros: separador de miles, separador decimal y cantidad de dígitos decimales.
2. **REQ-002**: El formato por defecto debe ser el de Nicaragua: miles `,`, decimales `.`, 2 dígitos → `1500` → `1,500.00`.
3. **REQ-003**: Agregar sección "Formato de moneda" en Settings para configurar: separador de miles, separador decimal y dígitos decimales.
4. **REQ-004**: Reemplazar TODOS los puntos de renderizado de montos del frontend (~30 sitios, listados en 4.5) para usar la utilidad compartida, incluyendo eliminar `formatMoney` local de `DebtAnalysis.tsx` y los `$` hardcodeados de `AddInsuranceModal.tsx` y `AutoShowPage.tsx`.
5. **REQ-005**: Persistir la configuración en `system_settings` (SQLite) vía `PUT /api/system-settings`, con los 3 nuevos campos.
6. **REQ-006**: Aplicar el mismo formato en los emails generados en Go (`formatAmount` en `bill_email.go`, `bill_summary_email.go`, `debt_due_email.go`; `pensionAmount` en `pension_email.go`).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-007**: Añadir claves i18n (`settings.currency_format.*`) en `frontend/public/i18n/es.json` y `en.json` (nunca en `public/i18n/`).
2. **REQ-008**: Validación de entrada: los separadores deben ser caracteres únicos (`,`, `.`, ` `, `'` o ninguno) y los decimales entre 0 y 4.
3. **REQ-009**: El valor debe formatearse con redondeo correcto (`1234567.5` con 0 decimales → `1,234,568`).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-010**: Presets rápidos en Settings: Nicaragua (`1,000.00`), EE.UU. (`1,000.00`), Europa (`1.000,00`), India/sin separador.
2. **REQ-011**: Vista previa en vivo del formato dentro de Settings ("Así se verá: C$1,234,567.50").

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: La utilidad de formato es una función pura O(n) sobre el string; sin impacto perceptible. Memoria adicional insignificante.
- **Seguridad**: Sin datos sensibles nuevos. Los campos de formato son cadenas cortas validadas.
- **Almacenamiento**: 3 filas nuevas en `system_settings` (key/value) — < 1 KB.
- **Disponibilidad**: Si falta alguna clave de formato en DB, se usan los defaults de Nicaragua (nunca romper el render).
- **iHost**: Cero dependencias nuevas. Frontend sigue siendo build estático; backend solo usa `fmt.Sprintf` existente. Consumo de RAM/CPU despreciable.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Estado actual**: no existe una utilidad compartida de formato de moneda. Todo usa `.toFixed(2)` inline concatenando `{currency?.symbol}` (o el code). `DebtAnalysis.tsx:33-35` tiene `formatMoney(code, amount)` local. `AddInsuranceModal.tsx:86` y `AutoShowPage.tsx:151` tienen `$` hardcodeado.
- **Backend**: `internal/services/bill_email.go:70-75` `formatAmount(amount, symbol)` → `fmt.Sprintf("%s%.2f", symbol, amount)` (sin miles). `pension_email.go:22-24` `pensionAmount(amount, currency)` → `fmt.Sprintf("%s %.2f", currency, amount)`. Test existente: `bill_email_test.go:92-98` asume `"C$1500.00"`.
- **Settings**: coexisten `settings` (solo `language`) y `system_settings` (key/value con timestamps, patrón `PUT /api/system-settings` partial-update con struct de punteros). El frontend ya consume `api.systemSettings.get()/update()` en `SettingsPage.tsx`. Precedentes client-side puros: `darkMode` y `hourFormat` en localStorage. Para esta spec se necesita el backend (emails), así que se usa `system_settings`.
- **i18n**: fuente de verdad en `frontend/public/i18n/{es,en}.json` (455/454 líneas); build de Vite copia a `public/` con `emptyOutDir`. Nunca editar `public/i18n`.
- **Formato Nicaragua**: estándar hispanoamericano — miles con coma, decimales con punto, 2 decimales. `Intl.NumberFormat('es-NI')` produce exactamente `1,000.00`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A: `Intl.NumberFormat('es-NI')` en frontend + librería en Go | Correcto, soporte de locales | Requiere locale fija en frontend, y en Go ninguna librería estándar lo hace (habría que agregar dependencia) | ❌ Rechazada |
| B: Utilidad propia configurable (`utils/currency.ts`) + `fmt` con separadores manuales en Go | Sin dependencias, totalmente configurable por formato (miles, decimal, dígitos), consistente front/back | Requiere implementar agrupación manual de miles (poco código) | ✅ Seleccionada |
| C: Solo frontend (localStorage) | Mínimo esfuerzo | Los emails del backend no respetan el formato | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Configuración por formato, no por país.
- **Contexto**: El usuario quiere Nicaragua por defecto pero que sea configurable "por formato", no "por país". Agregar un selector de país dispararía una tabla de locales y complejidad innecesaria.
- **Decisión**: Tres parámetros independientes: separador de miles, separador decimal y dígitos decimales. Con presets opcionales (P2) como atajo.
- **Consecuencias**: Máxima flexibilidad con mínimo código. Cualquier convención de formato es alcanzable.

**ADR-002**: Persistir en `system_settings`, no en localStorage.
- **Contexto**: El backend genera correos que deben respetar el formato. El precedente `darkMode`/`hourFormat` (localStorage) no sirve para esto.
- **Decisión**: 3 claves nuevas en `system_settings` leídas por el servicio de emails y por el frontend vía `GET /api/system-settings`. Defaults en Go si la clave no existe.
- **Consecuencias**: Un solo lugar de verdad; los emails y la UI siempre consistentes.

**ADR-003**: Utilidad de formato propia en vez de `Intl.NumberFormat`.
- **Contexto**: En Go no hay `Intl`. Para mantener front y back idénticos sin agregar dependencias, se implementa el mismo algoritmo en TypeScript y Go (~15 líneas cada uno).
- **Decisión**: Función pura `formatCurrency(amount, { thousandsSeparator, decimalSeparator, decimalDigits })` en frontend; helper equivalente en Go para emails.
- **Consecuencias**: Sin dependencias nuevas (regla iHost). El algoritmo es trivial (agrupar enteros en miles + unir con separador decimal + redondear a N decimales).

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[SettingsPage.tsx] --PUT /api/system-settings--> [SystemSettingsService] --> [SQLite: system_settings]
      |                                                    |
      | GET /api/system-settings                           v
      v                                     [bill_email.go, pension_email.go]
[currency.ts utility] <---- config ----            formatAmount / pensionAmount
      |
      v
[BillsPage, ServicesPage, SalariesPage, DeudasPage, DebtAnalysis,
 DebtCalendar, RegistrosPage, DebtBillsPage, forms...]
```

### 4.2 Componentes

#### 4.2.1 Utilidad `frontend/src/utils/currency.ts`
- **Responsabilidad**: Formatear montos según configuración.
- **Interfaz**: `formatCurrency(amount: number, cfg?: CurrencyFormatConfig): string` donde `CurrencyFormatConfig = { thousandsSeparator?: string; decimalSeparator?: string; decimalDigits?: number }`. Exportar también `DEFAULT_CURRENCY_FORMAT = { thousandsSeparator: ',', decimalSeparator: '.', decimalDigits: 2 }` (Nicaragua) y `formatCurrencyWithSymbol(amount, symbol, cfg)` para los casos `C$ 1,500.00`.
- **Dependencias**: Ninguna (función pura).
- **Ubicación**: `frontend/src/utils/currency.ts`.

#### 4.2.2 Sección en `SettingsPage.tsx`
- **Responsabilidad**: UI de configuración del formato con inputs: separador de miles (select: `,` / `.` / espacio / `'` / ninguno), separador decimal (select), dígitos decimales (select 0–4) y preview en vivo.
- **Interfaz**: reutiliza `api.systemSettings.get()/update()`.
- **Dependencias**: `utils/currency.ts`, claves i18n nuevas.
- **Ubicación**: `frontend/src/pages/SettingsPage.tsx`.

#### 4.2.3 Backend `SystemSettingsService` + handlers
- **Responsabilidad**: Persistir/leer las 3 claves de formato.
- **Interfaz**: nuevos campos en `settingsRequest` (JSON): `currency_thousands_separator`, `currency_decimal_separator`, `currency_decimal_digits` (todos punteros, patrón partial-update existente). Respuesta `GET /api/system-settings` los incluye.
- **Dependencias**: `internal/services/system_settings.go`, `internal/api/system_settings_handlers.go`.
- **Ubicación**: `internal/services/system_settings.go`, `internal/api/system_settings_handlers.go`, `migrations/0025_*.up.sql` (insert de defaults).

#### 4.2.4 Emails en Go
- **Responsabilidad**: Aplicar el formato configurado en correos.
- **Interfaz**: helper `formatAmount(amount, symbol, cfg)` y `pensionAmount(amount, currency, cfg)` leen `currency_format` del `SystemSettingsService` (o reciben config). Default Nicaragua si ausente.
- **Dependencias**: `SystemSettingsService`.
- **Ubicación**: `internal/services/bill_email.go`, `bill_summary_email.go`, `debt_due_email.go`, `pension_email.go`.

### 4.3 Modelo de datos

```
Tabla: system_settings (existente, key/value)
- clave: currency_thousands_separator  (TEXT, default ',')
- clave: currency_decimal_separator   (TEXT, default '.')
- clave: currency_decimal_digits      (TEXT, default '2')

Migración: migrations/0025_add_currency_format_settings.up.sql
  INSERT INTO system_settings (key, value) VALUES
    ('currency_thousands_separator', ','),
    ('currency_decimal_separator', '.'),
    ('currency_decimal_digits', '2');
```

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/system-settings`

**Response 200** (campos nuevos agregados al mapa existente):
```json
{
  "currency_thousands_separator": ",",
  "currency_decimal_separator": ".",
  "currency_decimal_digits": 2,
  "...": "..."
}
```

#### Endpoint: `PUT /api/system-settings`

**Request** (partial update, punteros):
```json
{
  "currency_thousands_separator": ",",
  "currency_decimal_separator": ".",
  "currency_decimal_digits": 2
}
```

**Response 200**: mapa completo actualizado.

**Response Error**:
```json
{
  "error": "invalid_currency_format",
  "message": "Separadores inválidos"
}
```

### 4.5 Alcance de reemplazo en frontend (sitios de render de montos)

| Archivo | Líneas |
|---------|--------|
| `frontend/src/pages/BillsPage.tsx` | 119, 158 |
| `frontend/src/pages/ServicesPage.tsx` | 153 |
| `frontend/src/pages/SalariesPage.tsx` | 99 |
| `frontend/src/pages/DebtBillsPage.tsx` | 74, 107 |
| `frontend/src/pages/DeudasPage.tsx` | 190, 193 |
| `frontend/src/components/DebtAnalysis.tsx` | 33-35 (formatMoney), 211, 235, 240, 245, 268, 278, 301, 308-309, 343 |
| `frontend/src/components/DebtCalendar.tsx` | 143, 162 |
| `frontend/src/pages/RegistrosPage.tsx` | 694, 789, 811, 843, 887, 948 |
| `frontend/src/components/AddInsuranceModal.tsx` | 86 (hardcoded `$`) |
| `frontend/src/pages/AutoShowPage.tsx` | 151 (hardcoded `$`) |

### 4.6 Dependencias

- **Internas**: `SettingsPage.tsx`, `api/index.ts` (tipos de `systemSettings`), todos los archivos de la tabla 4.5, servicios de email en Go, tests `bill_email_test.go`.
- **Externas**: Ninguna (regla iHost: sin dependencias nuevas).

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Con la config por defecto, un monto `1500` se renderiza como `1,500.00` en todas las páginas (Bills, Services, Salaries, Deudas, DebtAnalysis, DebtCalendar, Registros, DebtBills).
- [ ] CA-002: Existe una sección "Formato de moneda" en Settings con separador de miles, separador decimal y dígitos decimales.
- [ ] CA-003: Cambiar el formato en Settings se guarda vía `PUT /api/system-settings` y persiste en `system_settings` (SQLite) tras recargar.
- [ ] CA-004: Al cambiar la config, todos los montos de la UI se actualizan al nuevo formato (p. ej. `1.000,00` para miles `.` decimal `,`).
- [ ] CA-005: Con `currency_decimal_digits = 0`, un monto `1234567.5` se muestra como `1,234,568` (redondeo correcto).
- [ ] CA-006: Los correos automáticos (factura creada, resumen diario, deuda vencida, pensión) usan el mismo formato configurado; con default producen `C$1,500.00`.
- [ ] CA-007: Si las claves de formato no existen en DB, la app usa los defaults Nicaragua sin errores.
- [ ] CA-008: `AddInsuranceModal` y `AutoShowPage` dejan de mostrar `$` hardcodeado y usan el formateador configurado.
- [ ] CA-009: Las claves i18n nuevas existen en `es.json` y `en.json` (en `frontend/public/i18n/`) y se sirven tras `npm run build`.

### 5.2 No funcionales

- [ ] CA-NF-001: El bundle del frontend no crece de forma apreciable (utilidad ~0.5 KB) y no se agregan dependencias.
- [ ] CA-NF-002: El backend sigue arrancando en iHost sin migraciones fallidas; la migración 0025 es idempotente/segura.

### 5.3 Testing

- **Unit tests**: `formatCurrency` en TS (TS node o vitest si ya existe setup) cubriendo: default Nicaragua, redondeo, 0 decimales, separadores personalizados. En Go: actualizar/extender `bill_email_test.go` para el nuevo helper con miles.
- **Integration tests**: `PUT /api/system-settings` guarda y `GET` devuelve los 3 campos.
- **E2E tests**: flujo manual en local — cambiar formato en Settings, verificar montos en Deudas, Bills y un email generado.
- **Carga/Performance**: N/A (formateo trivial).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración 0025 + campos en `settingsRequest` y response de `system_settings` (backend) | 0.5 día | Ninguna |
| 2 | Helper Go `formatAmount`/`pensionAmount` con config + aplicación en emails + tests | 0.5 día | Fase 1 |
| 3 | Utilidad `utils/currency.ts` + tipo `CurrencyFormatConfig` + extender `api.systemSettings` types | 0.5 día | Ninguna |
| 4 | Sección "Formato de moneda" en SettingsPage + i18n + preview en vivo | 0.5 día | Fase 1, 3 |
| 5 | Reemplazar los ~30 sitios de render por la utilidad (incluye hardcoded `$`) | 1 día | Fase 3 |
| 6 | Validación manual en local (server corriendo) + ajustes según feedback | 0.5 día | Fases 1-5 |

**Estimación total**: ~3.5 días.

### 6.2 Milestones

1. **MVP**: Fases 1-3 (backend + utilidad + emails) — el default Nicaragua ya aplica en todos lados.
2. **V1.0**: Fases 4-6 (configurable desde Settings en frontend + reemplazo total de renders).

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Faltar algún sitio de render de montos (quede sin miles) | Media | Medio | Lista exhaustiva en 4.5; revisión con grep de `.toFixed(` y `formatMoney(` tras implementar |
| Los emails cambian de formato y rompen tests existentes | Media | Bajo | Actualizar `bill_email_test.go` y agregar tests del nuevo helper |
| Validación de separadores con inputs maliciosos/raros | Baja | Bajo | Whitelist de separadores válidos en backend y frontend |
| i18n roto por editar `public/i18n` por error | Baja | Alto | Regla AGENTS.md: solo `frontend/public/i18n/`; verificar con `curl http://localhost:8088/i18n/es.json` |

## 8. Notas y Referencias

- Precedentes de settings client-side: `darkMode`/`hourFormat` en `SettingsPage.tsx` (localStorage).
- Formato de referencia: `Intl.NumberFormat('es-NI')` → `1,000.00`.
- Regla AGENTS.md: la fuente de verdad de i18n es `frontend/public/i18n/`, no `public/i18n/`.
- Migraciones: naming `NNNN_descripcion.up.sql`/`.down.sql`; próxima libre `0025`.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-04 | paulomcnally | Creación inicial de la especificación |
| 2026-09-04 | paulomcnally | Release: implementación en commit `bc0b2f1` (SPEC-058: formato de moneda configurable con default Nicaragua) |