---
title: "Estandarización de formato y moneda en todos los montos del módulo de servicios y facturas"
id: "SPEC-113"
status: "draft"
author: "opencode"
created: "2026-10-21"
updated: "2026-10-21"
github_issue: 116
---

# Estandarización de formato y moneda en todos los montos del módulo de servicios y facturas

**ID**: SPEC-113  
**Estado**: draft  
**Autor**: opencode  
**Creado**: 2026-10-21  
**Actualizado**: 2026-10-21

---

## 1. Resumen Ejecutivo

En el detalle de facturas de un servicio (`/services/bills/:id`), concretamente en las pestañas **Análisis** (`?tab=analisis`) y **Facturas** (`?tab=facturas`), se muestran valores monetarios sin su moneda. El usuario entra directamente por URL (o recarga la página) y ve números como `1,234.56` o `C$ 1,234.56` de forma inconsistente, sin poder saber si corresponden a **NIO** o **USD**. El mismo problema aparece en la sección **Monto** del formulario de creación/edición de factura y en el modal de subida de facturas, donde el campo numérico no indica en qué moneda se está cargando el monto.

La causa raíz es doble. Primero, `BillsPage` resuelve el símbolo de moneda desde el catálogo global `useAppStore().currencies`, pero **no garantiza que ese catálogo esté cargado**: si se navega directo a la URL o se recarga, el array está vacío y `currency?.symbol` es `undefined`. Segundo, el formato de moneda se aplica de maneras distintas según el componente: `BillAnalysis` antepone manualmente el símbolo con un espacio (`${currencySymbol} ${formatMoney(...)}`), mientras que la tabla de facturas pasa el símbolo al helper; cuando el símbolo falta, ambos caminos degradan a un número "pelado". Además, el formulario de factura ignora la moneda del servicio aunque ya la tiene disponible en el estado.

Se propone una **regla única de presentación de montos**: todo valor monetario mostrado en el módulo de servicios/facturas debe pasar por el helper de formato configurado (`useCurrencyFormatStore().formatMoney`) y debe llevar su moneda, resuelta desde el registro del servicio (`services.currency_id` → catálogo `currencies`) con fallback al código ISO. La página que muestra dinero debe cargar el catálogo por sí misma, sin depender del historial de navegación. El formulario de factura y el modal de carga deben etiquetar el campo Monto con la moneda del servicio (solo lectura, sin selector nuevo). No se modifica el esquema de SQLite ni se agregan dependencias.

**Consideraciones iHost**: el cambio es exclusivamente de frontend (más un posible ajuste menor de tipos). No agrega queries nuevas relevantes: reutiliza `GET /api/currencies` (una sola llamada ya existente) y el formateo es JavaScript puro. Sin tablas nuevas, sin migraciones, sin librerías externas (se mantiene `Intl`-free: el helper propio `frontend/src/utils/currency.ts` ya implementa separadores configurables). Impacto en RAM/CPU despreciable y build multi-arch intacto.

**Consideraciones de UI obligatorias**: no se crean formularios nuevos, pero los campos de monto existentes (`BillFormPage`, `UploadBillModal`) deben mantener los tokens del tema (`bg-card`, `text-text`/`text-text-secondary`) y verificarse en darkmode (SPEC-060). No se crean páginas de detalle nuevas; las rutas ya están registradas en `BACK_ROUTES` (`frontend/src/components/DashboardLayout.tsx`), por lo que no aplica un cambio de navegación (CA-BACK se cumple sin cambios).

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: En `/services/bills/:id?tab=analisis`, **todos** los montos (Total, Pagado, Pendiente, Promedio, evolución, cambios de período, mayor aumento, mayor disminución, tooltips y etiquetas del gráfico de barras) deben mostrar su moneda. Ningún número monetario puede renderizarse sin símbolo o código de moneda.
2. **REQ-002**: En `/services/bills/:id?tab=facturas`, la columna **Monto** de la tabla desktop y el monto de las cards móviles deben mostrar siempre la moneda del servicio.
3. **REQ-003**: La moneda de una factura se resuelve desde **el servicio dueño de la factura** (`service.currency_id`) contra el catálogo `currencies`; si el catálogo no está disponible o el `currency_id` no existe, se debe mostrar el código ISO como fallback (nunca un número sin unidad). No se debe asumir `NIO` por defecto.
4. **REQ-004**: La página de facturas debe **cargar el catálogo de monedas por sí misma** al montarse (o exponer una forma equivalente que no dependa de que otra página lo haya cargado antes). Debe funcionar con navegación directa por URL, recarga (F5) y modo incógnito autenticado.
5. **REQ-005**: En el formulario de creación/edición de factura (`BillFormPage`), el campo **Monto** debe indicar la moneda del servicio de forma visible y de solo lectura (por ejemplo, etiqueta "Monto (C$ · NIO)" o sufijo dentro del input). No se agrega un selector de moneda: la factura hereda la moneda del servicio.
6. **REQ-006**: En el modal de subida de facturas (`UploadBillModal`), cada campo **Monto** por archivo debe indicar la moneda del servicio con el mismo criterio que REQ-005.
7. **REQ-007**: El formato numérico (separador de miles, separador decimal y cantidad de decimales) debe salir **siempre** de la configuración global de formato de moneda (`currencyFormatStore`), sin `toFixed` ni concatenaciones manuales de símbolo en los componentes del módulo.
8. **REQ-008**: El indicador de moneda debe ser inequívoco: en encabezados/resúmenes de la pestaña Análisis se muestra el **código ISO** (ej. `NIO`, `USD`) junto al símbolo; en filas, cards, badges y gráficos alcanza con el símbolo configurado (ej. `C$`). Si el símbolo se resolvió por fallback, se usa el código ISO en lugar del símbolo.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-009**: Auditar el resto de las vistas que muestran dinero para detectar montos sin moneda (presupuesto, deudas, registros/salarios, autos/pólizas y modales asociados) y corregir los casos del mismo patrón. La auditoría debe documentarse con el listado de archivos y el criterio aplicado; los casos fuera del alcance de facturas pueden resolverse en esta spec si el cambio es de bajo riesgo.
2. **REQ-010**: Centralizar la resolución `currency_id → {symbol, code}` en un único helper/hook reutilizable (frontend) para evitar que cada página repita la lógica y vuelva a introducir fallbacks silenciosos.
3. **REQ-011**: En la pestaña Análisis, si en el futuro un servicio tuviera facturas en monedas distintas (dato heredado o inconsistente), la UI no debe sumar montos de monedas diferentes como si fueran homogéneos; debe agrupar por moneda o advertir. En el modelo actual las facturas heredan la moneda del servicio, por lo que este caso es defensivo.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-012**: Agregar una prueba visual/manual documentada (checklist) que recorra las dos pestañas y el formulario en modo claro y oscuro, en móvil y desktop.
2. **REQ-013**: Evaluar mostrar el código ISO también en filas de facturas si el usuario lo prefiere; por defecto no, para no sobrecargar tablas.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: una llamada adicional `GET /api/currencies` al entrar a la página (ya cacheada en el store global). El formateo es O(n) sobre la cantidad de valores renderizados. Sin impacto medible en iHost.
- **Seguridad**: sin endpoints nuevos ni cambios de permisos; se reutilizan endpoints autenticados existentes.
- **Almacenamiento**: sin cambios de esquema ni migraciones. No se escriben datos nuevos.
- **Disponibilidad**: si el catálogo de monedas falla, la UI debe seguir funcionando mostrando el fallback (código ISO si se conoce; en el peor caso, un placeholder `—`), sin romper el render ni mostrar `NaN`.
- **iHost**: sin dependencias externas nuevas; se mantiene Go stdlib + SQLite + React/Tailwind ya existentes. El build se hace fuera del iHost.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Archivos relevantes relevados en el repositorio:

- `frontend/src/pages/BillsPage.tsx`: obtiene `currencies` de `useAppStore()` pero **no llama** a `loadAll()`/`loadCurrencies()`. Resuelve `const currency = currencies.find(c => c.id === service.currency_id)` y usa `formatMoney(bill.amount, currency?.symbol)` en cards (línea ~266) y tabla (línea ~308). Pasa `currency?.symbol` a `BillAnalysis`.
- `frontend/src/components/BillAnalysis.tsx`: recibe `currencySymbol?: string`; antepone manualmente `${currencySymbol || ''} ${formatMoney(...)}` en resúmenes, cambios y stats; en gráficos pasa el símbolo como segundo argumento. Si el símbolo es `undefined`, el resultado es un número sin moneda y con un espacio inicial.
- `frontend/src/stores/appStore.ts`: `currencies: []` por defecto; solo se llena con `loadAll()` o `loadCurrencies()`.
- `frontend/src/stores/currencyFormatStore.ts`: expone `formatMoney(amount, symbol?)`, que aplica separadores/decimales configurados y antepone el símbolo si existe.
- `frontend/src/utils/currency.ts`: implementación propia de formato (`formatCurrency`, `formatCurrencyWithSymbol`) sin `Intl`, con configuración global.
- `frontend/src/pages/BillFormPage.tsx`: destructura `currencies` pero no lo usa; el campo `bills.amount` no indica moneda.
- `frontend/src/components/UploadBillModal.tsx`: campos de monto por archivo sin indicación de moneda.
- Backend: `internal/models/bill.go` no incluye moneda; la factura hereda la moneda del servicio (`services.currency_id`, migración `0004_create_services.up.sql`). `internal/storage/bill.go` hace SELECT simple sobre `bills`.
- Precedente interno: `DebtAnalysis.tsx` usa `${b.currency_code || 'NIO'}` (fallback hardcodeado a NIO, no recomendable para nuevos desarrollos) y `DebtBillsPage.tsx` usa `currency?.symbol || debt.currency_code`. No existe hoy una regla única.

Verificación funcional: la página `BillsPage` se puede abrir directo por URL; como el store global arranca vacío, `currencies` no contiene el registro y `service.currency_id` no se puede mapear. Eso explica que los montos aparezcan sin moneda aun cuando el servicio tenga moneda configurada.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A. Solo cargar `loadAll()` en `BillsPage` | Cambio mínimo, resuelve el caso reportado | No corrige el formato manual de `BillAnalysis` ni el formulario; el problema puede reaparecer en otras páginas | ❌ Insuficiente sola |
| B. Cargar catálogo + helper único de resolución y formato en todas las vistas del módulo | Resuelve causa raíz y estandariza; sin backend; reutilizable | Requiere tocar varios componentes y agregar tests | ✅ Seleccionada |
| C. Enriquecer `Bill` en backend con `currency_symbol`/`currency_code` (JOIN a services/currencies) | Cada factura es autodescriptiva; robusto ante stores vacíos | Más cambios en storage (varias queries y scans), duplica la fuente de verdad y agrega costo de JOIN en listados | ⏸ Descartada para esta spec (posible evolución si se necesita exportar/notificar) |
| D. Usar `Intl.NumberFormat` | Estándar de la plataforma | Ignora la configuración de separadores/decimales del usuario y agrega variabilidad por locale | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: La moneda de una factura es la del servicio, no un atributo de la factura.
- **Contexto**: `bills` no tiene columna de moneda; el servicio la define al crearse (`services.currency_id`).
- **Decisión**: No se agrega columna ni JOIN en backend. La UI resuelve la moneda desde el servicio cargado en la misma pantalla.
- **Consecuencias**: Se evita una migración y se mantiene una única fuente de verdad. Requiere que la pantalla cargue el catálogo de monedas, lo cual se garantiza con REQ-004.

**ADR-002**: Un helper único de presentación de dinero.
- **Contexto**: Hoy conviven `formatMoney(amount, symbol)`, concatenaciones manuales con espacio y fallbacks hardcodeados a `NIO`.
- **Decisión**: Crear un helper/hook (ej. `useMoneyFormatter` o `resolveCurrency`) que reciba `currencyId`/`currencyCode` y el catálogo, devuelva `{symbol, code}` y exponga `format(amount)` usando el store de formato. Regla: nunca renderizar un número monetario sin pasar por este helper.
- **Consecuencias**: Consistencia visual y menos duplicación; una sola cosa que testear. Costo: pequeño refactor en componentes existentes.

**ADR-003**: Fallback explícito, sin asumir moneda.
- **Contexto**: Varias vistas asumen `NIO` cuando falta el código.
- **Decisión**: Si no se puede resolver la moneda, mostrar el código ISO conocido; si tampoco hay código, mostrar `—` y registrar un warning de desarrollo. Prohibido inventar `NIO`.
- **Consecuencias**: Evita mostrar datos financieros incorrectos; obliga a que el catálogo esté disponible.

**ADR-004**: Formato canónico `símbolo + número` en filas y `código ISO` en encabezados.
- **Contexto**: El helper existente antepone el símbolo sin espacio (`C$1,234.56`); `BillAnalysis` agrega un espacio manual y a veces queda doble.
- **Decisión**: Mantener el helper sin cambios de formato y dejar de concatenar símbolos a mano. En encabezados de resumen e indicadores de sección mostrar `NIO`/`USD` como etiqueta secundaria.
- **Consecuencias**: Consistencia con el resto de la app sin cambiar la configuración global ni migrar datos.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Service (services.currency_id)]
              │
              ▼
   [GET /api/services/:id] ──► [BillsPage]
              │                     │
              │                     ├── loadCurrencies() ──► [GET /api/currencies]
              │                     │                              │
              ▼                     ▼                              ▼
      [GET /api/bills?service]  [resolveCurrency(currency_id, catálogo)]
                                        │
                                        ▼
                             [formatMoney(amount, symbol)]
                                        │
                     ┌──────────────────┴───────────────────┐
                     ▼                                      ▼
              [BillAnalysis]                        [Tabla/Cards facturas]
                     │                                      │
                     ▼                                      ▼
        [Resúmenes/gráficos con código ISO]        [Monto con símbolo]
```

### 4.2 Componentes

#### 4.2.1 `frontend/src/utils/money.ts` (nuevo) o extensión de `currency.ts`
- **Responsabilidad**: resolver una moneda a partir de `currencyId` y/o `currencyCode` sobre el catálogo, y exponer un formateador con el formato global.
- **Interfaz**:
  ```ts
  interface ResolvedCurrency { symbol?: string; code?: string }
  resolveCurrency(currencies: Currency[], opts: { id?: number; code?: string }): ResolvedCurrency
  formatMoneyWithCurrency(amount: number, resolved: ResolvedCurrency, cfg: CurrencyFormatConfig): string
  ```
- **Dependencias**: `utils/currency.ts`, tipos `Currency`.
- **Ubicación**: `frontend/src/utils/`.

#### 4.2.2 `BillsPage.tsx`
- **Responsabilidad**: cargar servicio, facturas y catálogo de monedas; resolver la moneda una vez y pasarla a las pestañas.
- **Cambios**: llamar a `loadCurrencies()` (o `loadAll()`) en el `useEffect` inicial; usar el helper para `currency`; pasar `currencyCode` además de `symbol` a `BillAnalysis`; reemplazar `formatMoney(bill.amount, currency?.symbol)` por el helper (mismo resultado pero con fallback).
- **Dependencias**: `useAppStore`, `currencyFormatStore`, helper nuevo.

#### 4.2.3 `BillAnalysis.tsx`
- **Responsabilidad**: mostrar análisis de facturas con montos formateados y moneda.
- **Cambios**: recibir `currencyCode`; eliminar las concatenaciones `${symbol} ${formatMoney(...)}` y usar el helper una sola vez; en resúmenes mostrar el código ISO como etiqueta; en gráficos pasar el símbolo resuelto.
- **Dependencias**: helper nuevo, `formatMoney`.

#### 4.2.4 `BillFormPage.tsx` y `UploadBillModal.tsx`
- **Responsabilidad**: indicar la moneda al cargar el monto.
- **Cambios**: resolver la moneda del servicio (cargar catálogo si hace falta) y mostrar la etiqueta/sufijo en el campo Monto (ej. `Monto (C$ · NIO)`), solo lectura; mantener el input numérico actual.
- **Dependencias**: helper nuevo, `loadCurrencies`.

### 4.3 Modelo de datos

Sin cambios. Se reutiliza:

```
Entidad: Currency
- id: INTEGER
- code: TEXT (ISO, ej. NIO/USD)
- name: TEXT
- symbol: TEXT (ej. C$)

Entidad: Service
- id: INTEGER
- currency_id: INTEGER → currencies(id)

Entidad: Bill
- id, service_id, year, month, amount, ...
- Relación: Bill N:1 Service (hereda moneda)
```

### 4.4 APIs / Contratos

No se crean endpoints. Se consumen los existentes:

#### Endpoint: `GET /api/currencies`
**Response 200**:
```json
[
  { "id": 1, "code": "NIO", "name": "Córdoba", "symbol": "C$" },
  { "id": 2, "code": "USD", "name": "Dólar", "symbol": "US$" }
]
```

#### Endpoint: `GET /api/services/{id}`
**Response 200** (campos relevantes):
```json
{ "id": 3, "name": "Servicio Demo", "currency_id": 1 }
```

#### Endpoint: `GET /api/bills?service_id={id}`
**Response 200** (sin cambios; la moneda se resuelve en el cliente):
```json
[
  { "id": 10, "service_id": 3, "year": 2026, "month": 9, "amount": 1234.56, "status": "pending" }
]
```

**Response Error** (formato existente):
```json
{ "error": "code", "message": "descripción" }
```

### 4.5 Dependencias

- **Internas**: `BillsPage`, `BillAnalysis`, `BillFormPage`, `UploadBillModal`, `appStore`, `currencyFormatStore`, `utils/currency.ts`, i18n `frontend/public/i18n/{es,en}.json`.
- **Externas**: ninguna nueva. Sin `Intl` obligatorio, sin librerías de formato.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado un servicio con moneda `NIO` y al menos una factura, cuando se abre **directamente** `/services/bills/3?tab=analisis` (URL nueva o recarga), entonces todos los montos muestran `C$` y los resúmenes indican `NIO`; ningún monto queda como número pelado.
- [ ] CA-002: Dado el mismo servicio, cuando se abre `/services/bills/3?tab=facturas`, entonces la columna Monto (desktop) y el monto de cada card (móvil) muestran la moneda.
- [ ] CA-003: Dado un servicio con moneda `USD`, cuando se abren ambas pestañas, entonces los montos muestran `US$`/`USD` y no `C$` ni `NIO`.
- [ ] CA-004: Dado un catálogo de monedas que no se pudo cargar, cuando se abre la página, entonces los montos muestran el código ISO si está disponible y la vista no se rompe (sin `NaN`, sin `undefined`).
- [ ] CA-005: Cuando se crea o edita una factura, el campo Monto indica la moneda del servicio (etiqueta o sufijo) sin permitir cambiarla desde ese formulario.
- [ ] CA-006: Cuando se sube una factura con el modal, cada campo Monto indica la moneda del servicio.
- [ ] CA-007: Los separadores de miles, decimales y cantidad de decimales respetan la configuración de `Settings → Formato de moneda` (ej. `1.234,56` si así se configuró).
- [ ] CA-008: No se muestran espacios dobles ni símbolos duplicados (`C$ C$1,234.56`) en ningún monto.
- [ ] CA-009: La pestaña Análisis mantiene sus cálculos y gráficos funcionando; solo cambia la presentación de moneda.
- [ ] CA-DARK: Los campos de monto y las etiquetas de moneda se verificaron en darkmode (texto y placeholders legibles, tokens `bg-card`/`text-text`).
- [ ] CA-BACK: No se crean páginas nuevas; las rutas de detalle existentes siguen usando la flecha atrás del header vía `BACK_ROUTES`, sin links "← Título" en el contenido.

### 5.2 No funcionales

- [ ] CA-NF-001: El cambio no agrega más de una llamada de red adicional por carga de página (`GET /api/currencies`) y no introduce nuevas dependencias en `package.json`.
- [ ] CA-NF-002: El bundle de frontend no aumenta de forma significativa (< 5 KB gzip) y el build de Vite compila sin warnings nuevos.
- [ ] CA-NF-003: `go test ./...` sigue pasando y `npm run build` (en `frontend/`) finaliza correctamente.
- [ ] CA-NF-004: La app se validó en local contra una **copia de la DB de producción** para verificar que los servicios reales con moneda NIO/USD se muestran correctamente (sin escribir en producción).

### 5.3 Testing

- **Unit tests (frontend)**: `resolveCurrency` con id válido, id inexistente, catálogo vacío y code-only; `formatMoney` con los tres separadores y 0/2 decimales; ausencia de `undefined`/`NaN` en la salida.
- **Integration tests**: render de `BillAnalysis` con `currencySymbol` y `currencyCode` definidos/indefinidos; render de la tabla de facturas con y sin catálogo cargado.
- **E2E/manual**: navegación directa por URL a ambas pestañas, recarga, móvil/desktop, darkmode, servicio NIO y servicio USD, formulario de factura y modal de subida.
- **Carga/Performance**: verificar en el iHost que la página sigue respondiendo en tiempos similares (sin medición formal; comparación antes/después).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Helper de resolución/formato + tests unitarios | 0.5 día | Ninguna |
| 2 | `BillsPage` + `BillAnalysis`: carga de catálogo y formato unificado | 0.5 día | Fase 1 |
| 3 | `BillFormPage` + `UploadBillModal`: etiqueta de moneda en Monto | 0.25 día | Fase 1 |
| 4 | Auditoría del resto de vistas (REQ-009) y correcciones de bajo riesgo | 0.5 día | Fase 2 |
| 5 | i18n, build, pruebas manuales (claro/oscuro, móvil/desktop), validación con copia de producción | 0.5 día | Fases 2-4 |

### 6.2 Milestones

1. **MVP**: pestañas Análisis y Facturas muestran siempre la moneda, incluyendo navegación directa/recarga.
2. **V1.0**: helper único adoptado en el módulo de facturas, formulario y modal de subida; auditoría del resto de vistas documentada; pruebas y build en verde.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| El catálogo de monedas tarda en cargar y se muestra un frame sin moneda | Media | Bajo | Resolver con estado de carga y no renderizar montos hasta tener servicio + moneda; o usar fallback estable |
| `currency_id` apunta a una moneda inexistente/eliminada | Baja | Medio | Fallback a código ISO y log de advertencia; no inventar NIO |
| Refactor de `BillAnalysis` altera cálculos | Baja | Medio | Tests de render y comparación de totales antes/después; cambio limitado a presentación |
| Otras vistas usan `|| 'NIO'` y quedan inconsistentes | Media | Bajo | Incluir en la auditoría (REQ-009) y documentar los casos no corregidos |
| Cambios de i18n editados en `public/i18n` en vez de `frontend/public/i18n` | Baja | Medio | Editar solo `frontend/public/i18n/{es,en}.json` y verificar tras `npm run build` (regla del repo) |
| Datos reales usados en pruebas/specs | Baja | Alto | Usar solo datos sintéticos; validar sobre copia en `/tmp`; no pegar montos ni nombres reales |

## 8. Notas y Referencias

- `frontend/src/utils/currency.ts` — helper de formato existente.
- `frontend/src/stores/currencyFormatStore.ts` — configuración global de formato.
- `frontend/src/stores/appStore.ts` — catálogo de monedas.
- `frontend/src/pages/BillsPage.tsx`, `frontend/src/components/BillAnalysis.tsx` — superficies afectadas.
- `frontend/src/pages/BillFormPage.tsx`, `frontend/src/components/UploadBillModal.tsx` — campos Monto.
- `internal/models/bill.go`, `internal/storage/bill.go` — modelo de facturas sin moneda propia.
- SPEC-060 (darkmode), SPEC-063 (navegación de retorno), SPEC-098 (facturas de servicios).
- Regla del repo: i18n en `frontend/public/i18n/{es,en}.json`; validación con copia de la DB de producción.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-10-21 | opencode | Creación inicial de la especificación a partir del reporte del usuario sobre montos sin moneda en `/services/bills/:id` (pestañas Análisis y Facturas) y en el campo Monto del formulario. |
