---
title: "Gráfica de deudas: abrir por defecto en Últimos 12 meses"
id: "SPEC-106"
status: "released"
author: "opencode"
created: "2026-10-21"
updated: "2026-10-21"
github_issue: 109
---

# Gráfica de deudas: abrir por defecto en Últimos 12 meses

**ID**: SPEC-106  
**Estado**: released  
**Autor**: opencode  
**Creado**: 2026-10-21  
**Actualizado**: 2026-10-21

---

## 1. Resumen Ejecutivo

La SPEC-105 agregó la vista **«Últimos 12 meses»** al tab Gráfica de Deudas, pero dejó **«Cronograma»** como vista por defecto. El resultado práctico es que, al entrar a la Gráfica, el usuario ve una proyección a futuro que llega hasta **2043** (la Hipoteca tiene cuotas pendientes hasta `2043-01-12`), que no responde a la pregunta que motivó la spec original: *¿la deuda total subió o bajó en los últimos 12 meses?*. La vista histórica existe y calcula bien, pero queda escondida detrás de un click.

Esta spec cambia únicamente la **vista inicial** del tab Gráfica a **«Últimos 12 meses»**. Cronograma y Saldo total siguen disponibles en el segmented control, sin cambios en su contenido, cálculo ni contrato de backend. No se toca la API `/api/debts/history`, ni la base de datos, ni se agregan dependencias.

**Verificación con datos reales de producción** (copia `app-20261021-pre-spec106.db`, 12 deudas, 702 cuotas): el problema es exclusivamente de vista inicial. La serie histórica real es USD `8,8,9,9,9,9,9,9,9,9,8,8` (deudas abiertas al cierre de cada mes, nov-2025 → oct-2026) y NIO `1,1,1,1,1,1,3,3,3,3,3,0`; el Cronograma, en cambio, proyecta la Hipoteca hasta el año 2043. El usuario eligió **Historial por defecto**, manteniendo las otras dos vistas disponibles.

**Consideraciones iHost**: cambio puramente de estado inicial en el cliente React; cero costo de CPU/RAM, cero consultas adicionales, cero cambios de esquema. La vista histórica sigue usando el endpoint liviano de SPEC-105 (`O(12 × n)`, 2 consultas SQL).

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: El tab **Gráfica** de la página Deudas (`/deudas?tab=grafica`) DEBE iniciar en la vista **«Últimos 12 meses»** (`historial`), no en «Cronograma».
2. **REQ-002**: Las vistas **«Cronograma»** y **«Saldo total»** siguen disponibles en el segmented control y deben conservar su comportamiento actual (contenido, cálculos, tooltips, selector de moneda), sin regresiones.
3. **REQ-003**: El cambio aplica **solo al montaje inicial** del componente. Si el usuario cambia manualmente a Cronograma o Saldo, esa selección se respeta mientras permanezca en la página; no se agrega persistencia entre sesiones ni `localStorage`.
4. **REQ-004**: No se modifican el endpoint `GET /api/debts/history`, los tipos `DebtHistoryResponse`/`DebtHistoryCurrency`, ni la lógica de cálculo del backend.
5. **REQ-005**: El segmented control debe seguir mostrando las tres opciones en el mismo orden actual («Cronograma», «Saldo total», «Últimos 12 meses»), con «Últimos 12 meses» resaltada como activa al entrar.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-006**: Agregar/ajustar el test o verificación mínima de que el estado inicial del componente es `historial` (por ejemplo, un test de render del componente o una aserción directa del valor inicial si el proyecto no tiene infraestructura de tests de React).
2. **REQ-007**: Revisar que el early-return de estado vacío de Cronograma/Saldo no afecte a la vista histórica (comportamiento ya implementado en SPEC-105 REQ-001/CA-016); no debe romperse con el nuevo default.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-008**: Evaluar (sin implementarlo en esta spec) si en el futuro conviene persistir la última vista elegida por el usuario. Fuera de alcance.

### 2.4 Cambio solicitado durante la implementación (P0)

1. **REQ-009**: La vista «Últimos 12 meses» debe mostrar el **monto total adeudado de cada mes**, no solo la cantidad de deudas. Las **barras pasan a representar el monto** (saldo pendiente total de las deudas abiertas al cierre del mes) en la moneda seleccionada. La **cantidad de deudas se mantiene** en el tooltip, la tabla y las tarjetas.
2. **REQ-010**: El monto de cada mes se calcula por deuda como:
   - `outstanding(m) = SUM(amount) de las cuotas no eliminadas de la deuda − SUM(amount) de las cuotas pagadas cuyo mes de pago es ≤ m` (el total de cuotas es el monto financiado real, con intereses).
   - Si la deuda no tiene cuotas registradas, `outstanding(m) = deuda.total` mientras esté abierta.
   - El resultado se clampea a `>= 0` y se redondea a 2 decimales. Solo se suman deudas abiertas en `m` según las reglas de SPEC-105 (REQ-003/REQ-004).
3. **REQ-011**: El resumen debe agregar `start_amount`, `end_amount`, `change_amount` y `change_pct_amount` (porcentaje de variación del monto, `null` si el monto inicial es 0), además de los conteos existentes (`start_count`, `end_count`, `change_count`, `closed_count`) y `pending_amount_current`.
4. **REQ-012**: El **tooltip** de cada barra muestra mes, **monto adeudado**, cantidad de deudas abiertas y cantidad de deudas cerradas/eliminadas. La **tabla de apoyo** agrega la columna de monto. El **eje Y** usa etiquetas monetarias compactas (ej. `120k`) y el gráfico se escala al máximo monto del período.
5. **REQ-013**: Las **tarjetas** muestran el monto al inicio y al cierre (con la cantidad de deudas como subtítulo), la variación de monto (con porcentaje) y las deudas cerradas. Se mantiene la tarjeta de saldo pendiente actual como referencia.
6. **REQ-014**: El backend sigue resolviendo la historia en **máximo 2 consultas SQL** (deudas + pagos agrupados por deuda y mes) y el contrato sigue siendo una sola respuesta por todas las monedas.

### 2.5 Requerimientos No Funcionales

- **Rendimiento**: sin impacto medible; no hay requests nuevos ni recálculos.
- **Seguridad**: sin cambios; la vista histórica ya está bajo `authMiddleware`.
- **Almacenamiento**: sin migraciones ni cambios de esquema.
- **Disponibilidad**: cambio aditivo/cosmético-funcional en el cliente; no altera endpoints.
- **iHost**: cero dependencias nuevas, cero proceso en background, cero consumo extra de RAM.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Estado actual** (`frontend/src/components/DebtChart.tsx`): `const [view, setView] = useState<View>('cronograma')`. El tipo es `'cronograma' | 'saldo' | 'historial'` y el segmented control renderiza las tres opciones. La vista `historial` renderiza `DebtHistoryChart` (SPEC-105), que a su vez llama a `GET /api/debts/history?months=12`.
- **Origen del «2043»**: en la vista Cronograma, `lastEnd` es la fecha de fin de la deuda con cuotas pendientes más lejana y el eje Y se dibuja desde hoy hasta esa fecha. Con los datos reales, la Hipoteca (id 1, USD, 240 cuotas desde 2023-02-12, 196 pendientes, última cuota `2043-01-12`) define el extremo superior, por eso el eje llega a 2043. No es un bug de cálculo: es el propósito del Cronograma, pero no lo que el usuario esperaba al entrar.
- **Datos reales de producción** (copia consistente vía `sqlite3 .backup`, `PRAGMA integrity_check = ok`, 552.960 bytes): 12 deudas (9 `activa`, 3 `finalizada`), 702 cuotas (311 `paid`, 391 `pending`). Ventana nov-2025 → oct-2026:
  - USD: 8 deudas abiertas al inicio y 8 al cierre; pico de 9 entre enero y agosto de 2026.
  - NIO: 1 deuda abierta al inicio; 3 desde mayo de 2026; 0 en octubre de 2026 (las tres de NIO se cerraron ese mes).
  - `Seguro vehículo - Póliza 02B...` (id 10) está `activa`, sin cuotas pendientes ni pagadas; por REQ-003 de SPEC-105 no se considera cerrada, pero tampoco aporta cuotas. No afecta esta spec.
- **Conclusión**: la vista histórica funciona y refleja razonablemente la realidad. El único defecto es el **default** de la vista.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A. Cambiar el default a `historial` | Cambio mínimo (1 línea), resuelve exactamente lo pedido, conserva el resto | Quien prefiera Cronograma debe hacer un click | ✅ Seleccionada |
| B. Eliminar Cronograma y Saldo | Elimina la confusión de raíz | Pierde funcionalidad ya liberada y útil (proyección de fin de deudas) | ❌ Rechazada por el usuario |
| C. Limitar el eje del Cronograma a 12/24 meses | Evita ver 2043 | No resuelve que el usuario quiere ver la evolución pasada al entrar; cambia el significado del Cronograma | ❌ Rechazada por el usuario |
| D. Persistir la última vista en `localStorage` | Recuerda la preferencia | Agrega estado persistente y casos de migración; fuera de alcance | ❌ Rechazada (P2, futuro) |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: El default es una decisión de presentación, no de datos
- **Contexto**: la API y el cálculo histórico ya son correctos (verificado con datos reales). El problema es qué vista se muestra primero.
- **Decisión**: cambiar `useState<View>('cronograma')` a `useState<View>('historial')` en `DebtChart`, sin tocar backend, tipos ni `DebtHistoryChart`.
- **Consecuencias**: cambio de una línea, reversible y de bajo riesgo. Cronograma y Saldo total siguen accesibles. No hay migración ni impacto en la API.

**ADR-002**: No persistir la preferencia
- **Contexto**: el usuario pidió que al entrar vea los últimos 12 meses; no pidió recordar la última vista.
- **Decisión**: el default aplica en cada montaje del componente; la selección manual dura mientras el componente esté montado.
- **Consecuencias**: comportamiento predecible y simple. Si más adelante se quiere persistencia, se evalúa en otra spec (REQ-008).

**ADR-003**: El monto histórico se reconstruye con las cuotas, no con `debts.total`
- **Contexto**: verificado con datos reales, `debts.total` no siempre coincide con el monto financiado: la Hipoteca tiene `total = 58.050` pero la suma de sus 240 cuotas es `148.800`; el Préstamo Moto tiene `total = 3.389,69` y ya pagó `3.739,96` en cuotas. Usar `total − pagos` daría montos negativos o irreales.
- **Decisión**: el saldo de cada mes se calcula con las cuotas no eliminadas: `SUM(cuotas) − SUM(cuotas pagadas hasta el mes)`. Si la deuda no tiene cuotas, se usa `debt.total`. Se mantiene la regla de deuda abierta de SPEC-105.
- **Consecuencias**: el monto refleja el financiamiento real con intereses y es coherente con las cuotas que el usuario ve en el detalle. Requiere una segunda consulta agregada por `debt_id` y mes de pago (sin N+1), dentro del límite de 2 consultas.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[DeudasPage]
     |
     v
[DebtChart]  view inicial = 'historial'
     |
     +--> 'historial'  --> [DebtHistoryChart] --> GET /api/debts/history?months=12
     +--> 'cronograma' --> SVG proyectado a futuro (sin cambios)
     +--> 'saldo'      --> SVG saldo proyectado (sin cambios)
```

### 4.2 Componentes

#### 4.2.1 `DebtChart` (modificado)
- **Responsabilidad**: iniciar en la vista histórica y seguir permitiendo cambiar a las otras dos vistas.
- **Cambio**: `const [view, setView] = useState<View>('historial')`.
- **Ubicación**: `frontend/src/components/DebtChart.tsx`.
- **Dependencias**: las existentes (`DebtHistoryChart`, `Select`, stores de i18n y moneda).

#### 4.2.2 `DebtHistoryChart` (sin cambios)
- **Responsabilidad**: cargar `GET /api/debts/history?months=12` y dibujar barras, tarjetas, tooltip y tabla.
- **Ubicación**: `frontend/src/components/DebtHistoryChart.tsx`.

#### 4.2.3 Backend (modificado)
- `internal/storage/debt.go`: `ListForHistory` agrega `bills_total` (suma de cuotas no eliminadas) y una segunda consulta de pagos agrupados por `debt_id` y `substr(paid_at,1,7)`; el resultado se adjunta a cada fila.
- `internal/services/debt_history.go`: calcula `outstanding_amount` por mes y los montos del resumen.
- `internal/models/debt_history.go`: agrega `outstanding_amount` a `DebtHistoryPoint` y los campos de monto a `DebtHistorySummary`.
- `internal/api/debt_handlers.go` y la ruta no cambian.

### 4.3 Modelo de datos

Sin cambios de esquema. Se reutilizan `debts`, `debt_bills` y `currencies` existentes.

### 4.4 APIs / Contratos

Sin cambios. Se mantiene el contrato de SPEC-105 §4.4:

#### Endpoint: `GET /api/debts/history?months=12`

**Response 200** (resumen):
```json
{
  "months": 12,
  "generated_at": "2026-10-21T12:00:00Z",
  "currencies": [
    {
      "code": "USD",
      "symbol": "$",
      "series": [
        { "month": "2025-11", "open_count": 8, "closed_count": 0, "outstanding_amount": 214500.0 },
        { "month": "2026-10", "open_count": 8, "closed_count": 1, "outstanding_amount": 198603.08 }
      ],
      "summary": {
        "start_count": 8,
        "end_count": 8,
        "change_count": 0,
        "change_pct": 0,
        "closed_count": 1,
        "pending_amount_current": 198603.08,
        "start_amount": 214500.0,
        "end_amount": 198603.08,
        "change_amount": -15896.92,
        "change_pct_amount": -7.41
      }
    }
  ]
}
```

### 4.5 Dependencias

- **Internas**: `frontend/src/components/DebtChart.tsx`.
- **Externas**: ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Al abrir `/deudas` y entrar al tab **Gráfica**, se muestra directamente el gráfico de **«Últimos 12 meses»** con 12 barras mensuales (nov-2025 → oct-2026 con los datos reales de producción), sin hacer click en ninguna opción.
- [x] CA-002: El segmented control muestra las tres opciones y «Últimos 12 meses» aparece resaltada como activa al entrar.
- [x] CA-003: Al hacer click en «Cronograma», se ve la vista proyectada actual (incluido el eje hasta 2043 con los datos reales) sin errores; al volver a «Últimos 12 meses», la serie histórica se vuelve a mostrar correctamente.
- [x] CA-004: Al hacer click en «Saldo total», se ve la vista de saldo proyectado actual sin regresiones.
- [x] CA-005: Con la base real, la vista histórica muestra USD con 8 deudas al inicio y 8 al cierre, y NIO con 1 al inicio y 0 al cierre (los valores pueden variar si cambian los datos, pero deben coincidir con la serie y la DB).
- [x] CA-006: Con todas las deudas cerradas (sin `pending_count > 0`), el tab Gráfica sigue mostrando el segmented control y la vista histórica (no el estado vacío de Cronograma/Saldo).
- [x] CA-007: No hay cambios en el endpoint ni en el payload; `GET /api/debts/history` sigue respondiendo 200 con sesión y 401 sin sesión.
- [x] CA-008: Las barras representan el monto adeudado de cada mes; con los datos reales, el monto de USD baja de forma coherente mes a mes y NIO llega a 0 en octubre 2026. La cantidad de deudas sigue visible en tooltip, tabla y tarjetas.
- [x] CA-009: El tooltip muestra monto + cantidad abiertas + cerradas; la tabla incluye la columna de monto; el eje Y usa etiquetas monetarias y el gráfico se escala al máximo monto.
- [x] CA-010: Las tarjetas muestran monto al inicio, monto actual, variación de monto (con %), deudas cerradas y saldo pendiente actual de referencia.
- [x] CA-011: El endpoint sigue usando como máximo 2 consultas SQL y devuelve `series[].outstanding_amount` y los campos `start_amount`/`end_amount`/`change_amount`/`change_pct_amount`.

### 5.2 No funcionales

- [x] CA-NF-001: `npm run build` en `frontend/` pasa sin errores de TypeScript.
- [x] CA-NF-002: No se agregan dependencias npm ni Go; no hay migraciones.
- [x] CA-NF-003: La vista es usable en móvil (360 px), tablet y desktop; el segmented control mantiene áreas táctiles ≥44 px.
- [x] CA-DARK: El segmented control y el selector de moneda usan tokens del tema y se verificaron en darkmode.
- [x] CA-BACK: No se crean páginas de detalle ni links «← Título»; no requiere cambios en `BACK_ROUTES`.

### 5.3 Testing

- **Unit tests**: si existe infraestructura de tests de React, aserción del valor inicial `historial`; si no, verificación manual documentada del estado inicial.
- **Integration tests**: `GET /api/debts/history?months=12` contra la copia de producción en `/home/paulomcnally/p40la-db-backups/app-20261021-pre-spec106.db`; contrastar USD/NIO con las series calculadas.
- **Manual**: abrir `/deudas?tab=grafica` en local, confirmar CA-001 a CA-005, cambiar entre las tres vistas, probar darkmode y 360/768/1280 px.
- **Datos reales**: usar la copia de producción descargada el 2026-10-21; no usar datos sintéticos para la validación final.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Cambiar el default de `view` a `historial` en `DebtChart` | 5 min | Ninguna |
| 2 | Verificación con datos reales (endpoint + UI) y `npm run build` | 30 min | Fase 1 |
| 3 | Pruebas manuales del usuario y ajustes | 15 min | Fase 2 |

### 6.2 Milestones

1. **MVP**: el tab Gráfica abre en «Últimos 12 meses» con datos reales.
2. **V1.0**: verificación completa de las tres vistas, darkmode y responsive.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Usuarios que esperaban Cronograma al entrar ahora ven el historial | Media | Bajo | Las tres vistas siguen visibles y a un click; el cambio responde al pedido del usuario |
| Regresión en las vistas Cronograma/Saldo por tocar el estado | Baja | Medio | El cambio es solo el valor inicial; probar las tres vistas manualmente (CA-003/CA-004) |
| La vista histórica muestra un gráfico plano si no hay cambios | Media | Bajo | Es el dato real; se mantienen tarjetas de totales y tabla de apoyo de SPEC-105 |
| Diferencia entre datos de producción y locales | Baja | Bajo | Validar contra la copia de producción descargada y documentar la fecha |

## 8. Notas y Referencias

- `frontend/src/components/DebtChart.tsx` — `useState<View>('cronograma')` (causa del comportamiento actual).
- `frontend/src/components/DebtHistoryChart.tsx` — vista histórica de SPEC-105.
- `internal/services/debt_history.go`, `internal/storage/debt.go`, `internal/api/debt_handlers.go` — endpoint y cálculo, sin cambios.
- SPEC-105 — evolución de deudas en los últimos 12 meses (origen de la vista histórica).
- Copia de producción para pruebas: `/home/paulomcnally/p40la-db-backups/app-20261021-pre-spec106.db` (sha256 `af935e7c5b48cf206bb2e7d9d535107622760cacb15f0385228da29dcc2fee14`).
- Decisión del usuario (2026-10-21): «Historial por defecto», manteniendo Cronograma y Saldo disponibles.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-10-21 | opencode | Creación inicial de la especificación con datos reales de producción |
| 2026-10-21 | opencode | Inicio de implementación: default de la vista cambiado a `historial`; build y endpoint verificados con la copia de producción |
| 2026-10-21 | opencode | Cambio solicitado por el usuario: barras de monto adeudado por mes, manteniendo cantidad en tooltip/tabla/tarjetas (REQ-009..REQ-014, ADR-003) |
| 2026-10-21 | opencode | Implementados montos históricos en backend/frontend; verificado con la copia de producción (USD 236.362→197.794, NIO 62.221→0) |
| 2026-10-21 | opencode | Pruebas manuales del usuario satisfactorias («listo funciona»); criterios de aceptación validados. Estado: pending_release |
| 2026-10-21 | opencode | Release completo: merge a `main`, issue cerrado y worktree eliminado |
