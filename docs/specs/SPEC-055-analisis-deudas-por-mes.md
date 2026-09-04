---
title: "Análisis de Deudas por Mes con Gráficos en la página de Deudas"
id: "SPEC-055"
status: "in_progress"
author: "p40la-ihost-team"
created: "2026-09-04"
updated: "2026-09-04"
github_issue: 55
---

# Análisis de Deudas por Mes con Gráficos en la página de Deudas

**ID**: SPEC-055  
**Estado**: in_progress  
**Autor**: p40la-ihost-team  
**Creado**: 2026-09-04  
**Actualizado**: 2026-09-04

---

## 1. Resumen Ejecutivo

El módulo Deudas (SPEC-054) permite registrar deudas con su plan de cuotas (`debt_bills`) y ofrece dos vistas en la página `/deudas`: el **Calendario** y el listado de **Deudas**. Sin embargo, no existe una vista que responda de un vistazo a la pregunta financiera clave del usuario: *"¿cuánto dinero necesito este mes para pagar todas mis deudas?"*. Actualmente el usuario debe revisar cuota por cuota en el calendario para deducir lo pendiente.

Esta spec agrega una tercera pestaña **Análisis** en la página de Deudas que presenta todos los registros de cuotas del mes (con estado pagado/pendiente), desglosa cuánto está pagado y cuánto queda pendiente, muestra el total que se necesita para saldar el mes, y agrega gráficos relevantes (incluido un **gráfico de pastel** que muestra qué deuda consume más dinero). Se incorpora la navegación por mes (mes actual con flechas anterior/siguiente) replicando el patrón ya existente en la pensión (`RegistrosPage`) y en el propio `DebtCalendar`.

Como ya existe el endpoint `GET /api/debt-bills?year=&month=` (usado por el calendario) que devuelve todas las cuotas de un mes con descripción, institución, moneda, monto y estado, **no se requieren cambios de backend, migraciones ni tablas nuevas**. Toda la agregación se hace en el frontend. Los gráficos se implementan como SVG/CSS ligeros sin librerías externas para respetar las restricciones de iHost (mínimas dependencias, bajo consumo de memoria).

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Agregar una tercera pestaña **Análisis** en la página `/deudas` (junto a Calendario y Deudas), sincronizada con el mismo patrón de tabs existente.
2. **REQ-002**: En la pestaña Análisis, mostrar el **total del mes** y el desglose: total a pagar del mes, total ya pagado, y **total pendiente** (lo que se necesita para saldar el mes). Agrupado por moneda cuando haya más de una.
3. **REQ-003**: Listar todos los registros de cuotas del mes con su estado (pagado/pendiente), número de cuota, descripción de la deuda, institución, fecha de vencimiento y monto.
4. **REQ-004**: Navegación por mes: mostrar el mes en curso por defecto, con flechas anterior/siguiente y botón para volver al mes actual (patrón de `DebtCalendar` y `RegistrosPage`).
5. **REQ-005**: Agregar un **gráfico de pastel (donut)** que muestre qué deuda (agrupada por descripción) consume más dinero en el mes seleccionado, permitiendo identificar de un vistazo la deuda más costosa.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-006**: Gráfico de barras (opcional) que compare el total pendiente vs pagado por deuda en el mes, o el resumen mes a mes, para dar contexto adicional.
2. **REQ-007**: Indicador visual de progreso (porcentaje pagado del mes) junto al total pendiente.
3. **REQ-008**: Acción de marcar como pagada una cuota pendiente directamente desde la vista Análisis (reutilizando `DebtPayModal`).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-009**: Exportar/descargar el resumen del mes (CSV) para llevar registro externo.
2. **REQ-010**: Comparativa entre meses (mini histórico) para ver la evolución de deudas.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: La pestaña debe renderizar sin jank en el iHost; solo se cargan las cuotas del mes (volumen pequeño).
- **Seguridad**: Autenticación existente (authMiddleware). Sin datos sensibles adicionales.
- **Almacenamiento**: Sin cambios de esquema ni migraciones; sin almacenamiento adicional.
- **Disponibilidad**: Reutiliza `GET /api/debt-bills` existente; sin nuevos servicios.
- **iHost**: **Cero dependencias nuevas** (sin librería de gráficos). Gráficos SVG/CSS ligeros. No agregar peso al bundle más allá de lo estrictamente necesario. Sin cambios de runtime.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- El backend ya expone `GET /api/debt-bills?year=&month=` (`internal/api/debt_handlers.go:158`, `internal/storage/debt_bill.go:91`) que devuelve `[]DebtBill` de un mes con `debt_description`, `institution_name`, `currency_code`, `installment_number`, `due_date`, `amount`, `status`, `paid_at`.
- `DebtBill` (`internal/models/debt_bill.go`, `frontend/src/types/index.ts:254`) ya contiene todos los campos necesarios para agregar totales por estado.
- El patrón de navegación mensual ya existe en `frontend/src/components/DebtCalendar.tsx` (año/mes, flechas) y en `frontend/src/pages/RegistrosPage.tsx` (header con selector de mes).
- `frontend/package.json` NO incluye ninguna librería de gráficos (solo react, react-dom, react-router-dom, zustand). Agregar una (ej. recharts) aumentaría el bundle y el consumo de RAM, contradiciendo las restricciones de iHost.
- La página `/deudas` usa tabs controlados por `searchParams` (`DeudasPage.tsx:14`, `TabKey = 'calendario' | 'deudas'`).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Reutilizar `GET /api/debt-bills` y agregar en frontend | Cero backend, cero migraciones, rápido de implementar, volumen por mes pequeño | Cálculo de totales en cliente | ✅ Seleccionada |
| Nuevo endpoint de agregación backend | Cálculo servido, menor lógica cliente | Nuevo handler + storage + rutas + tests; más superficie | ❌ Rechazada (innecesario para pocas filas/mes) |
| Librería de gráficos (recharts/chart.js) | Gráficos listos y ricos | Bundle pesado, más RAM, contra restricciones iHost | ❌ Rechazada |
| Gráficos SVG/CSS propios (donut/barras) | Zero dependencias, ligero, control total del estilo iOS | Implementación manual | ✅ Seleccionada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Agregación y gráficos en el frontend.
- **Contexto**: Ya existe un endpoint que devuelve las cuotas de un mes; el volumen por mes es pequeño (decenas de filas).
- **Decisión**: Reutilizar `GET /api/debt-bills` y computar totales, agrupaciones y datos de gráficos en React con `useMemo`.
- **Consecuencias**: Sin cambios de backend ni migraciones; menos riesgo; la lógica de agregación vive en el cliente.

**ADR-002**: Gráficos SVG/CSS propios, sin librerías.
- **Contexto**: No hay librería de gráficos instalada y el proyecto exige mínimas dependencias (AGENTS.md, project-rules).
- **Decisión**: Implementar un donut/pie SVG y barras CSS con Tailwind, sin dependencias.
- **Consecuencias**: Bundle contenido, sin riesgo de breaking changes de librerías, estilo consistente.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[DebtCalendar]  [Listado Deudas]  [NUEVA pestaña Análisis]
         \            |              /
          \           v             /
       [DeudasPage: tabs via searchParams]
                 |
                 v
     [DebtAnalysis: mes actual + flechas]
                 |
                 v
   GET /api/debt-bills?year=&month=  (existente)
                 |
                 v
       [SQLite: debt_bills]  (sin cambios)
```

### 4.2 Componentes

#### 4.2.1 Nuevo componente `DebtAnalysis` (`frontend/src/components/DebtAnalysis.tsx`)
- **Responsabilidad**: Vista Análisis por mes de las cuotas de deuda.
- **Interfaz**: Sin props; usa `api.debts.billsByMonth(year, month)` y `useI18nStore`.
- **Dependencias**: `api`, `useI18nStore`, `Icon`, `DebtPayModal` (para marcar pagado).
- **Ubicación**: `frontend/src/components/`.

#### 4.2.2 Nuevo subcomponente de gráficos `DebtPieChart` (dentro de `DebtAnalysis.tsx`)
- **Responsabilidad**: Donut SVG que muestra el peso de cada deuda (por descripción) en el total del mes.
- **Dependencias**: Solo React + `Icon` opcional para leyenda.

### 4.3 Modelo de datos

Sin cambios. Se usa el modelo existente `DebtBill`:

```
Entidad: DebtBill (existente, sin modificar)
- debt_description, institution_name, currency_code
- installment_number, due_date, amount
- status: pending | paid, paid_at
```

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/debt-bills?year=YYYY&month=MM` (existente, sin cambios)

**Request**: `GET /api/debt-bills?year=2026&month=9`

**Response 200**: `DebtBill[]` (los mismos datos que consume `DebtCalendar`).

Sin endpoints nuevos. No hay contratos que modificar.

### 4.5 Dependencias

- **Internas**: `frontend/src/pages/DeudasPage.tsx` (agregar tab), `frontend/src/components/DebtCalendar.tsx` (patrón de navegación de referencia), `frontend/src/api/index.ts` (ya expone `billsByMonth`), `frontend/src/components/DebtPayModal.tsx` (reutilizar), `frontend/public/i18n/{es,en}.json` (nuevas claves).
- **Externas**: Ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado un usuario en `/deudas`, cuando selecciona la pestaña **Análisis**, entonces se muestra la vista por mes con el mes en curso por defecto.
- [ ] CA-002: Dado un mes con cuotas, cuando se abre Análisis, entonces se muestran los totales del mes (a pagar, pagado, pendiente) agrupados por moneda y el monto pendiente indica claramente cuánto se necesita para saldar el mes.
- [ ] CA-003: Dado un mes con cuotas, cuando se ve la lista, entonces cada registro muestra estado (pagado/pendiente), número de cuota, descripción, institución, vencimiento y monto.
- [ ] CA-004: Dado el selector de mes, cuando se navega con las flechas anterior/siguiente, entonces los totales y la lista se actualizan al mes correspondiente y se puede volver al mes actual.
- [ ] CA-005: Dado un mes con más de una deuda, cuando se abre Análisis, entonces el gráfico de pastel muestra la proporción de dinero que consume cada deuda, identificando a simple vista la más costosa.
- [ ] CA-006: Dado un mes sin cuotas, cuando se abre Análisis, entonces se muestra un estado vacío coherente con el patrón de EmptyCard.
- [ ] CA-007: Dado un mes con cuotas pendientes, cuando el usuario marca una como pagada desde Análisis, entonces el estado, los totales y el gráfico se actualizan.

### 5.2 No funcionales

- [ ] CA-NF-001: El build de frontend (`npm run build`) compila sin errores y sin nuevas dependencias en `package.json`.
- [ ] CA-NF-002: No se realizan cambios de backend, migraciones ni esquema de DB.
- [ ] CA-NF-003: Las claves i18n nuevas se sirven correctamente desde `/i18n/es.json` y `/i18n/en.json` tras el build.

### 5.3 Testing

- **Unit tests**: Lógica de agregación de totales por estado y por moneda; cálculo de datos del donut (porcentajes por descripción).
- **Integration tests**: Flujo de carga de un mes vía `api.debts.billsByMonth` y render de la pestaña Análisis.
- **E2E tests**: Navegación entre tabs, navegación mensual, marcado de pago y actualización de totales/gráfico.
- **Carga/Performance**: Verificar render fluido en iHost (bundle sin librería de gráficos).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Agregar tab **Análisis** en `DeudasPage` y ruta de estado | 0.5 día | Ninguna |
| 2 | Componente `DebtAnalysis` con navegación mensual, totales por estado/moneda y listado | 1 día | Fase 1 |
| 3 | Gráfico de pastel (donut) SVG + barras opcionales | 1 día | Fase 2 |
| 4 | Reutilizar `DebtPayModal` para marcar pagado y refrescar | 0.5 día | Fase 2 |
| 5 | Claves i18n (es/en), build y validación en local | 0.5 día | Fases 2-4 |

### 6.2 Milestones

1. **MVP**: Tab Análisis con mes en curso, totales (a pagar/pagado/pendiente), listado con estado y gráfico de pastel de deudas.
2. **V1.0**: Navegación mensual completa + acción de marcar pagado + i18n + barras de contexto.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Múltiples monedas en el mes complican el "total a pagar" | Media | Medio | Agrupar totales por moneda y mostrar cada grupo claramente |
| Donut SVG complejo con muchos segmentos | Baja | Bajo | Limitar a top N + "otros"; tooltip/leyenda clara |
| Tabs controlados por `searchParams` generan URLs largas | Baja | Bajo | Solo persistir `tab`, no `year/month` (o persistir de forma limpia) |
| Conflicto con otras sesiones por editar `DeudasPage.tsx` | Media | Alto | Trabajar en worktree aislado de la spec; seguir AGENTS.md |

## 8. Notas y Referencias

- SPEC-054 (módulo Deudas) — base de este requerimiento.
- `frontend/src/pages/DeudasPage.tsx` — tabs existentes.
- `frontend/src/components/DebtCalendar.tsx` y `frontend/src/pages/RegistrosPage.tsx` — patrón de navegación mensual a replicar.
- `internal/api/debt_handlers.go:158` y `internal/storage/debt_bill.go:91` — endpoint reutilizado.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-04 | p40la-ihost-team | Creación inicial de la especificación. Requerimiento de usuario: pestaña Análisis por mes en Deudas. Se añadió gráfico de pastel para identificar la deuda que consume más dinero (solicitud de usuario). |
