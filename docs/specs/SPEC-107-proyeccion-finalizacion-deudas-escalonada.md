---
title: "Proyección de finalización de deudas: gráfico escalonado con granularidad mensual o anual"
id: "SPEC-107"
status: "released"
author: "opencode"
created: "2026-10-21"
updated: "2026-10-21"
github_issue: 110
---

# Proyección de finalización de deudas: gráfico escalonado con granularidad mensual o anual

**ID**: SPEC-107  
**Estado**: released  
**Autor**: opencode  
**Creado**: 2026-10-21  
**Actualizado**: 2026-10-21

---

## 1. Resumen Ejecutivo

El tab **Gráfica** de la página Deudas ya responde a dos preguntas: *cuándo termina cada deuda* (Cronograma), *cómo baja el saldo con el tiempo* (Saldo total) y *cómo evolucionó la deuda en los últimos 12 meses* (Historial, SPEC-105/106). Falta una vista que muestre de forma **agregada y legible** el **calendario de finalización**: en qué meses o años se van liberando los compromisos y cuánto saldo pendiente queda después de cada tramo.

Esta spec agrega una **cuarta vista "Proyección"** al segmented control del tab Gráfica. La vista dibuja un **gráfico escalonado (step chart)**: una línea que se mantiene plana y baja en cada período en el que finalizan una o más deudas, mostrando el **saldo pendiente remanente** hasta llegar a cero. La **granularidad es adaptativa**: si la deuda más lejana termina dentro de los próximos **12 meses o menos**, el eje X se agrupa **por mes**; si el horizonte supera los 12 meses, se agrupa **por año**. Así la gráfica evita 200 puntos mensuales en una hipoteca a 17 años y sigue siendo útil cuando todo se termina el año que viene. El usuario eligió explícitamente el gráfico **escalonado** y la métrica **monto pendiente**.

Todo el cálculo se hace en el **frontend** con los datos que `DeudasPage` ya carga (`pending_amount`, `last_due_date`, `pending_count` de SPEC-104): **no hay endpoint nuevo, ni migraciones, ni dependencias npm**. El gráfico reutiliza el patrón SVG/tooltip de `DebtChart` y `DebtHistoryChart` (SPEC-101/102/105), con tokens del tema y responsive por `viewBox`.

**Previsualización con datos reales**: antes de subir la feature, se clona la base de producción del iHost (backup consistente con `sqlite3 .backup`, `PRAGMA integrity_check`) y se levanta el server local apuntando `DATA_DIR` a una **copia** de ese clon. Nunca se escribe en la DB de producción. Con los datos reales de SPEC-106 (12 deudas, 702 cuotas, Hipoteca con cuotas pendientes hasta 2043), la vista debe abrir automáticamente en modo **anual**.

**Consideraciones iHost**: cambio 100% cliente (React/SVG), cero CPU/RAM adicional en el server, cero consultas nuevas, cero dependencias. El cálculo es O(n) sobre las deudas ya cargadas en memoria (decenas).

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Agregar una **cuarta vista "Proyección"** al segmented control del tab Gráfica de Deudas (`/deudas?tab=grafica`), junto a las existentes "Cronograma", "Saldo total" y "Últimos 12 meses". No se modifica el contenido, cálculo ni comportamiento de las otras tres vistas. La vista por defecto sigue siendo **"Últimos 12 meses"** (SPEC-106); "Proyección" se activa manualmente.
2. **REQ-002**: La vista "Proyección" muestra un **gráfico escalonado (step chart)** de una sola serie: el **saldo pendiente proyectado** en la moneda seleccionada. El valor inicial (extremo izquierdo) es la suma de `pending_amount` de las deudas abiertas (`pending_count > 0`). En cada período donde finalizan deudas, el saldo **baja** por la suma de sus `pending_amount`; el valor final del gráfico es **0**. La línea nunca sube.
3. **REQ-003**: **Granularidad adaptativa**:
   - Si la fecha de finalización más lejana del conjunto filtrado cae **dentro de los próximos 12 meses contados desde el mes actual (inclusive)**, agrupar **por mes** (`YYYY-MM`).
   - Si cae **más allá de ese horizonte**, agrupar **por año** (`YYYY`).
   - La granularidad se decide **una sola vez por moneda seleccionada**, no por deuda. El umbral es: `mes_fin_max <= mes_actual + 11` → mensual; en caso contrario → anual.
4. **REQ-004**: **Fecha de finalización de cada deuda**: se usa `debtPendingEndDate(debt)` (SPEC-104), que toma `last_due_date` (última cuota pendiente) y, si no existe, la fecha de cierre por `last_paid_at`. Se usan solo deudas con `pending_count > 0` y con una fecha de finalización válida. Las deudas cuya fecha de fin sea anterior al mes actual se agrupan en el **primer período** (mes o año actual) para no perder el monto ya vencido.
5. **REQ-005**: El eje X muestra los períodos en orden cronológico:
   - Modo mensual: desde el **mes actual** hasta el mes de la última finalización (máximo 12 etiquetas).
   - Modo anual: desde el **año actual** hasta el año de la última finalización.
   - En modo anual, `2026` incluye las deudas que finalizan este año; el saldo inicial se dibuja al comienzo del primer período (hoy).
6. **REQ-006**: El **eje Y** representa dinero, arranca en 0 y escala al saldo pendiente inicial total. Debe tener 4–5 líneas de grilla con etiquetas monetarias compactas (ej. `120k`, `1.2M`) usando el `symbol` de la moneda activa. El gráfico debe usar un `viewBox` responsive y scroll horizontal **solo dentro de la tarjeta** si hay demasiados años; no debe ensanchar la página.
7. **REQ-007**: **Tooltip por período** (mismo patrón flotante clampeado al viewport de `DebtChart`): nombre del período, **monto que finaliza en ese período**, **cantidad de deudas** que finalizan y **saldo restante después** del período. Si el período no tiene finalizaciones, el tooltip muestra saldo restante y "0" finalizaciones. El monto debe formatearse con `formatMoney` y el símbolo de la moneda.
8. **REQ-008**: **Selector de moneda** compartido con el resto de las vistas: solo se muestra si hay más de una moneda con deudas abiertas. El default es la moneda con mayor `pending_amount` actual (mismo criterio de SPEC-104/105). El gráfico, el tooltip y la tabla se recalculan al cambiar de moneda.
9. **REQ-009**: **Estado vacío** con el patrón existente (card centrada, ícono, título y subtítulo) cuando no hay deudas con `pending_count > 0` en la moneda seleccionada. La vista "Proyección" no debe romper el early-return de las vistas Cronograma/Saldo.
10. **REQ-010**: La vista debe incluir una **nota explicativa** al pie (texto secundario, `text-[10px]`) que aclare:
    - que la proyección se basa en la **última cuota pendiente registrada** de cada deuda;
    - que el criterio de granularidad es mensual si todo termina en ≤ 12 meses y anual si el horizonte es mayor;
    - que el saldo proyectado es una referencia y puede variar por mora, intereses o ajustes.
11. **REQ-011**: **Sin cambios de backend**: no se agregan endpoints, rutas, migraciones ni consultas. La vista consume exclusivamente el array `debts` que `DeudasPage` ya pasa a `DebtChart`.
12. **REQ-012**: **Internacionalización**: todas las etiquetas nuevas deben agregarse en `frontend/public/i18n/es.json` y `frontend/public/i18n/en.json` (fuente de verdad). No editar `public/i18n/`. Claves mínimas: `chart_view_projection`, `chart_title_projection`, `chart_note_projection`, `projection_period_month`, `projection_period_year`, `projection_ends_in`, `projection_debts_count`, `projection_remaining_balance`.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-013**: **Tabla de apoyo colapsable** debajo del gráfico con columnas: período, **monto que finaliza**, **cantidad de deudas** y **saldo restante**. Sirve como respaldo numérico y para lectores de pantalla. En modo anual la tabla puede ser la vista principal de detalle; en modo mensual replica los 12 puntos.
2. **REQ-014**: Resaltar con color primario pleno el **tramo correspondiente al período actual** y con menor opacidad los tramos futuros. Marcar con línea punteada horizontal el **50% del saldo inicial** como referencia visual.
3. **REQ-015**: Mostrar un **badge/indicador de granularidad** junto al título (ej. "Mensual" / "Anual") para que el usuario entienda por qué cambió el eje X, con el mismo estilo de texto secundario del resto de la tarjeta.
4. **REQ-016**: En el tooltip, si el período tiene **hasta 3 deudas**, listar sus descripciones; si tiene más, mostrar "y N más". Evitar tooltips gigantes.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-017**: Resaltar en la tabla el **primer período en que el saldo remanente llega a 0** como "libre de deudas".
2. **REQ-018**: Evaluar (sin implementarlo) un toggle manual Mensual/Anual cuando el horizonte supere 12 meses; fuera de alcance de esta spec.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: cálculo O(n) sobre deudas abiertas (decenas), sin requests nuevos. Render SVG en el cliente; sin librerías de charts.
- **Seguridad**: sin endpoints nuevos; no se exponen datos adicionales. La página sigue bajo autenticación existente.
- **Almacenamiento**: **sin migraciones** ni cambios de esquema. No se escribe en `debts`/`debt_bills`.
- **Disponibilidad**: cambio aditivo en el cliente; no altera respuestas de API.
- **iHost**: cero dependencias nuevas, cero procesos en background, cero consumo extra de RAM en el server. El costo es despreciable en el navegador.
- **UI/darkmode**: todos los colores salen de tokens (`bg-card`, `text-text`, `text-text-secondary`, `border`, `primary`). El gráfico y la tabla deben verse correctamente en light y dark mode.
- **Responsive**: móvil, tablet y desktop; sin overflow horizontal de página (solo scroll interno del gráfico si aplica).

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Componente actual** (`frontend/src/components/DebtChart.tsx`, SPEC-101/102/104/105/106):
  - `View = 'cronograma' | 'saldo' | 'historial'`; estado inicial `historial` (SPEC-106).
  - `items` = deudas con `pending_count > 0`, con `endDate` calculada por `debtPendingEndDate` y ordenadas por fecha de fin.
  - `filteredItems` = `items` filtradas por moneda; default = moneda con mayor `pending_amount`.
  - `totalOwed` = suma de `pending_amount`; `lastEnd` = fecha de fin más lejana.
  - Ya existe `renderSaldo()`: un step chart de facto sobre una escala temporal continua, con un punto por deuda. **No agrupa** y usa una escala de tiempo (años) que con la Hipoteca llega a 2043.
  - El early-return por `items.length === 0` y el estado vacío ya están resueltos; la nueva vista debe convivir con ellos.
- **Utilidad** (`frontend/src/utils/debtMonthlyAggregation.ts`): `debtPendingEndDate(debt)` devuelve `{ year, month, day }` a partir de `last_due_date` (cuota pendiente más lejana) o `last_paid_at` (cierre). Es la fuente correcta para "según los registros".
- **Tipos** (`frontend/src/types/index.ts`): `Debt` ya expone `pending_count`, `pending_amount`, `last_due_date`, `last_paid_at` (SPEC-104). No hace falta backend.
- **UI de referencia**: `DebtHistoryChart.tsx` (SPEC-105) ya implementa `compactMoney`, `niceMax`, grilla con 5 líneas, tooltip clampeado, tabla de apoyo y selector de moneda. `DebtChart` aporta el patrón del segmented control y el tooltip `fixed`. Se reutilizan esos patrones, no se crea una librería de charts.
- **Datos reales de producción** (documentados en SPEC-106, copia `app-20261021-pre-spec106.db`, `integrity_check = ok`): 12 deudas (9 `activa`, 3 `finalizada`), 702 cuotas (311 pagadas, 391 pendientes). La Hipoteca tiene cuotas pendientes hasta `2043-01-12`, por lo que el horizonte real es de ~17 años → la vista debe abrir en **modo anual**. Las deudas de NIO se cerraron en octubre de 2026; al seleccionar NIO el horizonte es corto y podría caer en **modo mensual**, lo que valida el comportamiento adaptativo.
- **i18n**: las claves del tab Gráfica viven en el bloque `deudas` de `frontend/public/i18n/{es,en}.json`; el build de Vite copia ese directorio a `public/i18n/`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A. Reutilizar `renderSaldo()` sin agrupar | Cero código nuevo de cálculo | La escala temporal continua no responde "en qué mes/año termino"; con 17 años los puntos se amontonan; el usuario pidió granularidad adaptativa | ❌ Rechazada |
| B. Nueva vista "Proyección" con step chart y agrupación mensual/anual en el frontend | Reutiliza datos ya cargados, sin API ni dependencias; responde exactamente la pregunta; el eje X queda legible | Duplica parte del cálculo de agrupación en el cliente (pocas líneas) | ✅ Seleccionada |
| C. Endpoint backend `GET /api/debts/projection` | Centraliza el cálculo | No aporta: los datos ya están en el cliente; agrega superficie de API y latencia en iHost | ❌ Rechazada |
| D. Gráfico de barras por período (como Historial) | Consistente con SPEC-105 | El usuario pidió explícitamente **escalonada**; una barra no muestra el saldo remanente acumulado | ❌ Rechazada |
| E. Agrupar siempre por año | Simple | Pierde detalle cuando todo termina en menos de 12 meses | ❌ Rechazada |
| F. Agrupar siempre por mes | Detalle máximo | Ilegible con horizontes de 17 años (200+ puntos) | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Cálculo y agrupación en el frontend, sin endpoint nuevo
- **Contexto**: `DeudasPage` ya carga todas las deudas con sus agregados (`pending_amount`, `last_due_date`) y las pasa a `DebtChart`. Un endpoint nuevo obligaría a otra llamada y a duplicar la regla de "deuda abierta".
- **Decisión**: construir la serie de proyección en un `useMemo` dentro de `DebtChart` (o un componente `DebtProjectionChart` si el archivo crece demasiado), a partir de `filteredItems`.
- **Consecuencias**: cero latencia adicional y cero cambios de backend. La lógica queda acoplada al contrato de `Debt` de SPEC-104; si ese contrato cambia, hay que actualizar la vista.

**ADR-002**: Granularidad adaptativa con umbral de 12 meses
- **Contexto**: El usuario pidió "mensual si es en 12 meses o menos, anual si son muchos". Un umbral fijo evita un eje X saturado y mantiene la lectura rápida.
- **Decisión**: si `mes_fin_max <= mes_actual + 11`, agrupar por mes; si no, por año. La decisión se toma por moneda, después del filtro.
- **Consecuencias**: con la Hipoteca (2043) la vista real abre en anual; con conjuntos de deudas cortas abre en mensual. El cambio de moneda puede cambiar la granularidad, y el badge lo hace explícito.

**ADR-003**: El step chart representa el saldo remanente, no el monto que finaliza
- **Contexto**: "Proyección de cuándo finalizan las deudas" puede interpretarse como barras de montos que vencen por período o como saldo que queda. El usuario eligió "monto pendiente" y "escalonada".
- **Decisión**: la altura de la línea es el **saldo pendiente remanente**; cada escalón hacia abajo corresponde al monto de las deudas que finalizan en ese período. El monto que finaliza y la cantidad se muestran en el tooltip y la tabla.
- **Consecuencias**: la gráfica comunica de un vistazo "cuándo quedás libre" y cuánto pesa cada tramo. No confunde el monto que vence con el saldo total.

**ADR-004**: Previsualización con un clon de la DB de producción, sin tocar producción
- **Contexto**: El usuario pidió ver cómo se ve con datos reales antes de subir la feature. La DB de producción vive en el volumen Docker del iHost y está en modo WAL.
- **Decisión**: generar un backup consistente en el iHost con `sqlite3 .backup`, verificar `PRAGMA integrity_check`, descargarlo a `/home/paulomcnally/p40la-db-backups/`, copiarlo a un `DATA_DIR` temporal local (ej. `/tmp/opencode/spec107-data/app.db`) y correr el server local contra esa copia. La DB de producción **no se abre para escritura** y la DB local del usuario (`data/app.db`) **no se sobrescribe**.
- **Consecuencias**: la validación manual refleja los datos reales y es repetible. El clon puede contener datos personales: se mantiene solo en la máquina de desarrollo y no se commitea.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
DeudasPage
  └── api.debts.list()  (sin cambios)
        └── DebtChart (debts, currencies)
              ├── segmented: Cronograma | Saldo total | Últimos 12 meses | Proyección
              └── Proyección (nuevo)
                    ├── useMemo: agrupar filteredItems por mes/año
                    ├── SVG step chart (saldo remanente)
                    ├── tooltip por período
                    └── tabla de apoyo

Sin backend nuevo · Sin SQLite nuevo · Sin dependencias npm
```

### 4.2 Componentes

#### 4.2.1 `DebtChart.tsx` (modificado)
- **Responsabilidad**: agregar la vista `'proyeccion'` al tipo `View`, el cuarto botón del segmented control, el título/nota y el render.
- **Interfaz**: recibe `debts: Debt[]` y `currencies: Currency[]` (sin cambios).
- **Dependencias**: `debtPendingEndDate`, `formatMoney`, i18n.
- **Ubicación**: `frontend/src/components/DebtChart.tsx`.

#### 4.2.2 `DebtProjectionChart.tsx` (nuevo, opcional si el render crece)
- **Responsabilidad**: recibir `items` filtrados por moneda y `currencies`, calcular la serie agrupada y dibujar el step chart, tooltip y tabla.
- **Interfaz**:
  ```ts
  interface Props {
    items: { debt: Debt; endDate: Date }[]
    currencies: Currency[]
    effectiveCurrency: string
  }
  ```
- **Dependencias**: React, `useI18nStore`, `useCurrencyFormatStore`, helpers de formato.
- **Ubicación**: `frontend/src/components/DebtProjectionChart.tsx`.

> Decisión de implementación: si el bloque supera ~150 líneas dentro de `DebtChart`, extraerlo a `DebtProjectionChart.tsx`. Mantener un solo componente es preferible si el render es compacto.

### 4.3 Modelo de datos (en memoria)

```
ProjectionPoint {
  key: string           // "2027-03" (mensual) | "2027" (anual)
  label: string         // "mar 27" | "2027"
  year: number
  month?: number        // solo mensual
  endingAmount: number  // suma de pending_amount de deudas que finalizan en el período
  endingCount: number   // cantidad de deudas
  remainingAmount: number // saldo después del período
}
```

- Se ordena por `key` ascendente.
- `remainingAmount` se calcula acumulando: `remaining -= endingAmount`, clampeado a `>= 0`.
- El punto inicial (`remainingAmount = totalOwed`, `endingAmount = 0`) se dibuja al comienzo del primer período (hoy).
- Montos redondeados a 2 decimales al mostrar; el cálculo usa los valores completos.

### 4.4 APIs / Contratos

**No se agregan endpoints.** Se reutiliza `GET /api/debts` (ya consumido por `DeudasPage`) y los campos calculados de `Debt`:
- `pending_count: number`
- `pending_amount: number`
- `last_due_date?: string` (`YYYY-MM-DD` de la última cuota pendiente)
- `last_paid_at?: string`

### 4.5 Dependencias

- **Internas**: `DebtChart.tsx`, `debtMonthlyAggregation.ts`, `types/index.ts`, `i18nStore`, `currencyFormatStore`, `frontend/public/i18n/{es,en}.json`.
- **Externas**: ninguna nueva. SVG nativo de React.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: El tab Gráfica muestra **cuatro** vistas en el segmented control; "Proyección" está disponible y "Últimos 12 meses" sigue siendo la vista inicial (SPEC-106 no se rompe).
- [x] CA-002: Con un conjunto de deudas cuyo fin más lejano cae dentro de los próximos 12 meses, el eje X se agrupa **por mes** y muestra como máximo 12 etiquetas.
- [x] CA-003: Con un conjunto cuyo fin más lejano supera los 12 meses (caso real: Hipoteca 2043), el eje X se agrupa **por año** y el badge indica "Anual".
- [x] CA-004: El step chart parte del **saldo pendiente total** de la moneda seleccionada, baja en cada período con finalizaciones y termina en **0**; nunca sube.
- [x] CA-005: El tooltip de cada período muestra período, monto que finaliza, cantidad de deudas y saldo restante; se posiciona dentro del viewport en móvil.
- [x] CA-006: El selector de moneda cambia la serie y puede cambiar la granularidad; el default es la moneda con mayor saldo pendiente.
- [x] CA-007: La tabla de apoyo muestra período, monto que finaliza, cantidad y saldo restante, y coincide con el gráfico.
- [x] CA-008: La nota al pie explica la fuente (última cuota pendiente), el criterio mensual/anual y que es una referencia.
- [x] CA-009: Sin deudas abiertas en la moneda seleccionada, se muestra el estado vacío existente sin errores de render.
- [x] CA-010: No se agregan endpoints, migraciones ni consultas; `git diff` no toca `internal/`, `migrations/` ni `cmd/`.
- [x] CA-011: Las claves i18n nuevas existen en `frontend/public/i18n/es.json` y `en.json` (no en `public/i18n/`) y se sirven tras `npm run build`.
- [x] CA-DARK: El gráfico, el segmented control, el selector y la tabla usan tokens del tema y son legibles en darkmode (texto, grillas, tooltip).
- [x] CA-BACK: La spec no crea páginas de detalle ni formularios con lista padre; no aplica `BACK_ROUTES`.

### 5.2 No funcionales

- [x] CA-NF-001: `go build ./...` y `go test ./...` siguen pasando (sin cambios de backend).
- [x] CA-NF-002: `npm run build` en `frontend/` compila sin errores de TypeScript y sin warnings nuevos.
- [x] CA-NF-003: La página Deudas no genera scroll horizontal a nivel de página en móvil; si el gráfico lo necesita, el scroll queda dentro de la tarjeta.
- [x] CA-NF-004: El cálculo de la serie no hace requests adicionales ni bloquea el render de las otras vistas.

### 5.3 Testing

- **Unit tests**: si se extrae una función pura `buildProjection(items, monthsThreshold)`, cubrir: agrupación mensual, agrupación anual, deudas vencidas en el primer período, una sola deuda, saldo final 0 y redondeo.
- **Manual (obligatoria)**: levantar el server local con el **clon de la DB de producción** y verificar:
  1. La vista "Proyección" abre en modo **anual** con la Hipoteca hasta 2043.
  2. Al cambiar a NIO, la granularidad y la serie son coherentes.
  3. El saldo inicial coincide con la tarjeta "Saldo pendiente" de la vista.
  4. El último período muestra saldo 0.
  5. Tooltip y tabla coinciden.
  6. Darkmode y móvil (DevTools) sin overflow.
- **E2E**: no aplica infraestructura E2E en el repo; se reemplaza por la prueba manual con clon.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Agregar `'proyeccion'` al tipo `View`, cuarto botón y claves i18n ES/EN | 0.25 día | Ninguna |
| 2 | Implementar cálculo de agrupación mensual/anual y step chart SVG | 0.5 día | Fase 1 |
| 3 | Tooltip, badge de granularidad, tabla de apoyo y nota | 0.25 día | Fase 2 |
| 4 | Pruebas locales: build, tests, clon de DB de producción y validación manual/darkmode | 0.25 día | Fase 3 |

### 6.2 Milestones

1. **MVP**: vista "Proyección" con step chart, granularidad adaptativa y tooltip.
2. **V1.0**: badge, tabla de apoyo, nota y validación con clon de producción.

### 6.3 Procedimiento de clonado de la DB de producción (validación)

1. En el iHost, con el contenedor de la app detenido o desde una copia consistente, ejecutar:
   ```bash
   sudo sqlite3 /app/data/app.db ".backup '/app/data/backups/20261021-spec107/app.db'"
   sudo sqlite3 /app/data/backups/20261021-spec107/app.db "PRAGMA integrity_check;"
   ```
2. Descargar el backup a `/home/paulomcnally/p40la-db-backups/` (vía `paramiko`, ya que SFTP no está disponible).
3. Copiarlo a un `DATA_DIR` temporal local:
   ```bash
   mkdir -p /tmp/opencode/spec107-data
   cp /home/paulomcnally/p40la-db-backups/<archivo>.db /tmp/opencode/spec107-data/app.db
   ```
4. Levantar el server local **sin tocar** `data/app.db`:
   ```bash
   DATA_DIR=/tmp/opencode/spec107-data PORT=8088 ./scripts/run-local.sh
   ```
5. Crear/actualizar el usuario de prueba sobre la copia:
   ```bash
   DB_PATH=/tmp/opencode/spec107-data/app.db ./scripts/create-test-user.sh
   ```
6. Verificar `./scripts/check-server.sh` y navegar a `/deudas?tab=grafica` → "Proyección".
7. Al terminar, borrar el `DATA_DIR` temporal si el usuario lo autoriza. **Nunca** copiar el clon sobre `data/app.db`.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Deudas con `pending_count > 0` pero sin `last_due_date` ni `last_paid_at` | Baja | Medio | Excluirlas de la serie y documentarlo en la nota; opcionalmente contarlas en el primer período. Validar con el clon. |
| Confusión entre "monto que finaliza" y "saldo remanente" | Media | Medio | Tooltip y tabla muestran ambos; la nota explica que la línea es el saldo que queda. |
| Con moneda NIO el horizonte es corto y cambia a mensual, mientras USD queda anual | Media | Bajo | Es el comportamiento deseado; el badge de granularidad lo hace explícito. |
| Demasiados años en el eje X (ej. 18) y etiquetas superpuestas | Baja | Bajo | Rotación/alternancia de etiquetas y scroll interno; ancho de diseño con `viewBox`. |
| Overflow horizontal en móvil | Media | Medio | `overflow-x-auto` solo en el contenedor del gráfico; verificar con DevTools. |
| Clon de producción con datos personales | Baja | Medio | Mantenerlo fuera del repo, en `/tmp/opencode` y `/home/paulomcnally/p40la-db-backups/`; no commitear. |
| El usuario espera ver meses aunque el horizonte sea 2043 | Media | Bajo | Badge "Anual" + nota; la spec documenta el criterio acordado (P2 deja abierto un toggle futuro). |

## 8. Notas y Referencias

- SPEC-100: Deudas — tab Gráfica de barras.
- SPEC-101/102: prototipo y responsive de gráficos de deudas.
- SPEC-104: saldo pendiente y fin proyectado por última cuota pendiente.
- SPEC-105: evolución de deudas en los últimos 12 meses (patrón de UI y cálculo de series).
- SPEC-106: vista inicial por defecto del tab Gráfica.
- `frontend/src/components/DebtChart.tsx`, `DebtHistoryChart.tsx`.
- `frontend/src/utils/debtMonthlyAggregation.ts`.
- `docs/ssh-ihost-access.md` (acceso SSH y backups de la DB del iHost).

## 9. Resultado de la implementación (verificación local)

**Fecha**: 2026-10-21  
**Estado**: implementado, pendiente de confirmación manual del usuario.

### 9.1 Archivos modificados/creados

- `frontend/src/components/DebtProjectionChart.tsx` (nuevo): cálculo de la serie agrupada, step chart SVG, tooltip, tabla y nota.
- `frontend/src/components/DebtChart.tsx`: cuarta vista `proyeccion`, botón en el segmented control, título y render; segmented con `flex-wrap` para móvil.
- `frontend/public/i18n/es.json` y `en.json`: claves nuevas (el build las sincroniza a `frontend/src/i18n/`).
- `docs/specs/SPEC-107-proyeccion-finalizacion-deudas-escalonada.md`: esta spec.

### 9.2 Verificación con clon de producción

- Backup remoto: `/app/data/backups/20261021-spec107/app.db` (`.backup`, 552.960 bytes).
- `PRAGMA integrity_check` remoto y local: `ok`.
- Copia local: `/home/paulomcnally/p40la-db-backups/20261021-spec107/app.db` y `/tmp/opencode/spec107-data/app.db`.
- Contenido: 12 deudas, 702 cuotas, 8 deudas abiertas con `last_due_date`.
- Server local: `DATA_DIR=/tmp/opencode/spec107-data PORT=8090`, usuario `test@test.com` / `test1234`.
- Resultado real: fin más lejano `2043-01-12` (Hipoteca) → **modo anual**, 18 períodos (2026–2043), saldo inicial `$197.794,40`, último período `$0,00`.
- Tooltip verificado (2028): finaliza `$13.223,88`, 2 deudas, saldo restante `$181.986,35`.
- Tabla de apoyo: 18 filas, última en 2043 con `$0,00`.
- Modo mensual verificado con una copia sintética del clon (fines entre dic-2026 y jun-2027): badge **"Mensual"**, eje X mensual correcto.
- Móvil 390 px: `scrollWidth == innerWidth == 390` (sin overflow de página); segmented se acomoda en dos líneas. Sin errores de consola.
- `npm run build` OK; `go build ./...`, `go test ./...` y `node --test frontend/tests/*.test.ts` OK.
- La DB de producción **no se modificó** y `data/app.db` local no se sobrescribió.

### 9.3 Confirmación y release

- Pruebas manuales del usuario: **satisfactorias** (2026-10-21).
- Release: merge de `feature/SPEC-107` a `main`, tracker actualizado, label `spec/released` y issue cerrado.
- Deploy: no se subió al iHost; queda a decisión del usuario.

## 10. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-10-21 | opencode | Creación inicial de la especificación. Requerimiento del usuario: proyección de finalización de deudas en gráfico escalonado, mensual si el horizonte es ≤ 12 meses y anual si es mayor, con validación previa sobre un clon de la DB de producción. |
| 2026-10-21 | opencode | Inicio de implementación (`in_progress`): cuarta vista "Proyección", gráfico escalonado y granularidad adaptativa. |
| 2026-10-21 | opencode | Implementación completa (`pending_release`): nuevo `DebtProjectionChart`, i18n ES/EN y verificación con clon de producción (modo anual 2026–2043 y modo mensual sintético). Pendiente confirmación manual. |
| 2026-10-21 | opencode | Pruebas manuales del usuario satisfactorias. Release aprobado (`released`). Commit de release: `0b8ea72`. |
