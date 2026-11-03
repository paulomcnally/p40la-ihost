---
title: "Home de Servicios con Análisis, Calendario y Gráfica (espejo de Deudas)"
id: "SPEC-120"
status: "draft"
author: "opencode"
created: "2026-11-03"
updated: "2026-11-03"
github_issue: 123
---

# Home de Servicios con Análisis, Calendario y Gráfica (espejo de Deudas)

**ID**: SPEC-120  
**Estado**: draft  
**Autor**: opencode  
**Creado**: 2026-11-03  
**Actualizado**: 2026-11-03

---

## 1. Resumen Ejecutivo

El home de Servicios (`/services`) hoy es un listado plano de cards: no permite responder preguntas básicas como cuánto se facturó este mes, cuánto está pagado, cuánto queda pendiente, qué vence cada día o cómo evolucionó el gasto en el último año. El home de Deudas (`/deudas`) ya resuelve esa necesidad con cuatro tabs (Análisis, Calendario, Deudas y Gráfica) y un selector de período. Este requerimiento pide **replicar esa experiencia en Servicios**, adaptada a la naturaleza de las facturas de servicios (recurrentes mensuales/anuales, multi-moneda y por casa).

El resultado esperado es que `/services` pase a tener las mismas cuatro tabs, con **Análisis** como tab por defecto: resumen del mes (total a pagar, pagado, pendiente y porcentaje pagado), desglose por servicio y listado de facturas del período; **Calendario** con las facturas por día de vencimiento; **Servicios** con el listado actual y filtros por estado; y **Gráfica** con la evolución de los últimos 12 meses y una proyección de los próximos 12. El selector de casa existente se mantiene y pasa a aplicar a las cuatro tabs.

Todo el estado de navegación (tab, casa, período, filtro y vista de gráfica) debe vivir en el **querystring**, de modo que cambiar de tab conserve la casa seleccionada, que un refresh mantenga el contexto y que cualquier vista se pueda compartir por URL. No se agregan dependencias: los gráficos siguen siendo SVG propios y las agregaciones se resuelven en SQLite/Go, respetando las restricciones de memoria del iHost.

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001 — Cuatro tabs en el home de Servicios.** `/services` debe mostrar las tabs `Análisis`, `Calendario`, `Servicios` y `Gráfica`, con el mismo patrón visual y de interacción que `/deudas` (botones con icono, mínimo 44px de alto, tab activa con `bg-primary text-white`). La tab por defecto es `análisis`.
2. **REQ-002 — Estado de navegación en la URL.** El estado debe persistir en el querystring con estas claves:
   - `tab`: `analisis` (default) | `calendario` | `servicios` | `grafica`.
   - `home`: id numérico de casa; ausente = todas las casas.
   - `status`: `pending` (default) | `paid` | `inactive` | `all` (solo aplica a la tab Servicios).
   - `year` y `month`: período seleccionado (default: año y mes actuales); los usan Análisis y Calendario.
   - `gview`: `history` (default) | `projection` (solo aplica a la tab Gráfica).
   Cambiar de tab, de casa, de período o de filtro debe actualizar únicamente la clave correspondiente y **preservar todas las demás**. Un refresh, un atajo de historial del navegador (atrás/adelante) o una URL compartida deben reconstruir exactamente la misma vista.
3. **REQ-003 — Selector de casa global.** El selector de casa existente (modal `HousePickerModal` con búsqueda) se mantiene en el header y pasa a escribir/leer el parámetro `home`. La casa seleccionada filtra las cuatro tabs: análisis, calendario, listado y gráfica. La opción "Todas las casas" elimina el parámetro `home`.
4. **REQ-004 — Tab Análisis: resumen del mes.** Debe mostrar un navegador de período (mes anterior/siguiente y acción "Mes actual"), el período en texto (ej. `noviembre 2026`) y, para la casa y el período seleccionados:
   - "Total a pagar" = suma de facturas del período.
   - "Pagado" = suma de facturas con `status = paid`.
   - "Pendiente" = suma de facturas con `status = pending`.
   - Porcentaje pagado del mes con barra de progreso.
   Los montos deben mostrarse **por moneda** (una línea por moneda, sin convertir ni sumar monedas distintas). Si hay más de una moneda debe existir un selector `Select` de moneda (no nativo) que filtre todo el tab, con opción "Todas las monedas".
5. **REQ-005 — Tab Análisis: desglose y facturas del período.** Debajo del resumen:
   - Donut SVG "Pendiente por servicio" sobre la moneda activa (top 6 servicios + "Otros").
   - Detalle por servicio con barra pagado/pendiente.
   - Listado de facturas del período con búsqueda por servicio, institución o número de factura; cada ítem muestra fecha de vencimiento, monto, estado, link al detalle del servicio y acción "Pagar" cuando está pendiente (reutilizando `PayBillModal`).
   - Si hay facturas anuales sin `due_date`, se asignan a enero del año (ver REQ-007) y se muestran en el período correspondiente.
6. **REQ-006 — Tab Calendario.** Calendario mensual equivalente al de Deudas:
   - Grilla de escritorio (7 columnas) con puntos de color (pagado/pendiente) por día; agenda en móvil.
   - Navegación de mes anterior/siguiente.
   - Día seleccionado con listado de facturas, total pendiente del día y acción "Pagar".
   - Las facturas sin `due_date` se ubican usando `billing_day` del servicio (ajustado al último día del mes); si no existe, día 1. Debe indicarse visualmente que la fecha es estimada cuando corresponda (P1).
7. **REQ-007 — Regla de período efectivo (determinista).** Para agrupar una factura en un mes/año se aplica este orden:
   1. Si `bills.due_date` existe y es válida → año/mes de `due_date`.
   2. Si `bills.month` está entre 1 y 12 → `bills.year` / `bills.month`.
   3. Si `bills.month = 0` (factura anual) y no hay `due_date` → **enero** de `bills.year` (coincide con la generación automática anual del scheduler, que corre en enero).
   Esta regla debe implementarse una sola vez en la capa de servicios/almacenamiento y usarse en Análisis, Calendario e Histórico para evitar resultados inconsistentes.
8. **REQ-008 — Tab Servicios: listado con filtros.** Mantener las cards actuales (icono, nombre, institución, casa, estado, webhook, sync, editar, eliminar) y agregar los filtros con contadores, espejo de Deudas:
   - `pending`: servicios activos con `latest_bill_status = pending`.
   - `paid`: servicios activos con `latest_bill_status = paid`.
   - `inactive`: servicios con `active = false` (sin importar el estado de la última factura).
   - `all`: todos.
   Los filtros son excluyentes y el filtro activo se persiste en `status`. Las cards conservan el menú de acciones y el orden alfabético. La tab debe funcionar igual con o sin casa seleccionada.
9. **REQ-009 — Tab Gráfica: histórico de 12 meses.** Vista `gview=history`:
   - Gráfico de barras por mes de los últimos 12 meses (incluyendo el mes actual) con monto facturado, pagado y pendiente.
   - Selector de moneda (`Select`) y tarjetas de resumen: total facturado del período, total pagado, total pendiente, promedio mensual, variación entre el primer y último mes con datos, cantidad de facturas.
   - Los meses sin facturas deben mostrarse en cero para no cortar la línea temporal.
   - No se permite sumar monedas distintas: el gráfico siempre muestra una moneda a la vez.
10. **REQ-010 — Tab Gráfica: proyección de 12 meses.** Vista `gview=projection`:
    - Proyección de los próximos 12 meses desde el período seleccionado, separando **confirmado** (facturas pendientes ya existentes) de **estimado** (servicios activos recurrentes sin factura en ese período, usando `suggested_amount`).
    - Los servicios anuales se estiman en enero de cada año (misma regla del scheduler); los mensuales, en cada mes dentro de `start_date`/`end_date`.
    - Si un servicio ya tiene una factura en el período, no se suma su estimado (evitar doble conteo).
    - Tabla de detalle por período con confirmado, estimado y total.
    - Nota visible: "Estimado con el monto sugerido; puede variar".
11. **REQ-011 — Títulos de página por tab.** El título del documento (`usePageTitle`) debe reflejar la tab activa (Análisis, Calendario, Servicios, Gráfica), igual que Deudas.
12. **REQ-012 — Estados vacíos.** Cada tab debe tener un `EmptyCard` con icono, título, descripción y acción cuando corresponda:
    - Sin casas → invitar a crear casa (comportamiento actual).
    - Sin servicios → invitar a crear servicio.
    - Sin facturas en el período → explicar que no hay facturas ese mes y ofrecer navegar al mes actual.
    - Sin datos para graficar → explicar que se necesitan facturas.
13. **REQ-013 — i18n.** Todas las cadenas nuevas deben agregarse en `frontend/public/i18n/es.json` y `frontend/public/i18n/en.json`, bajo el namespace `services` (o sub-namespaces `services.analysis_*`, `services.calendar_*`, `services.chart_*`, `services.view_*`). No se permiten textos hardcodeados en los componentes nuevos. Los textos hardcodeados existentes en `ServicesPage.tsx` (`Inactivo`, `Vencido`, `Nueva institución`, `Nueva moneda`) deben migrarse a claves i18n.
14. **REQ-014 — Sin cambios de esquema.** La funcionalidad debe resolverse con las tablas existentes (`services`, `bills`, `homes`, `currencies`). No se agregan columnas ni tablas. Si las mediciones con una copia de producción muestran consultas lentas, se evaluará un índice en una spec aparte (ver REQ-NF-004).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-015 — Indicador de fecha estimada.** En Calendario y Análisis, marcar las facturas cuya fecha de vencimiento fue inferida desde `billing_day` (no existe `due_date`) con un badge "Fecha estimada".
2. **REQ-016 — Ranking de gasto por servicio.** En la tab Gráfica, sub-vista o sección "Gasto por servicio" del período seleccionado, ordenada de mayor a menor, con monto y porcentaje.
3. **REQ-017 — Contexto preservado al navegar.** Al abrir el detalle de un servicio o una factura desde el home, incluir el querystring de origen para que la flecha atrás del header vuelva al mismo tab/casa/período. Si la implementación de `BACK_ROUTES` no permite query dinámica, se acepta volver a `/services` sin filtros y se documenta como limitación.
4. **REQ-018 — Refresco silencioso.** Al pagar una factura desde Análisis o Calendario, actualizar sólo los datos del tab visible (no recargar toda la página) y mostrar toast de éxito.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-019 — Exportar el análisis del período.** Botón para copiar/descargar un resumen CSV del período (servicio, vencimiento, monto, moneda, estado). Sin dependencias nuevas.
2. **REQ-020 — Comparación año contra año.** En Gráfica, toggle "vs. año anterior" que superpone los 12 meses del año anterior.

### 2.4 Requerimientos No Funcionales

- **REQ-NF-001 — iHost / memoria.** Sin dependencias nuevas de frontend ni backend. Los gráficos son SVG propios; no se incorporan librerías de charts. El bundle no debe crecer más de ~15 KB gzip.
- **REQ-NF-002 — Performance de API.** `GET /api/service-bills` y `GET /api/service-bills/summary` deben responder en < 300 ms con hasta 5.000 facturas en una copia de producción, con una sola consulta agregada (sin N+1).
- **REQ-NF-003 — Payload.** El listado mensual no debe superar ~200 KB con 300 facturas. El histórico se devuelve agregado por período/moneda, nunca con todas las facturas crudas de 12 meses.
- **REQ-NF-004 — Índices.** No se crean índices en esta spec. Durante la validación se ejecuta `EXPLAIN QUERY PLAN`; si el filtro por período efectivo hace un scan costoso, se documenta y se propone migración en una spec aparte.
- **REQ-NF-005 — Seguridad.** Todos los endpoints nuevos usan el `authMiddleware` existente; no exponen datos de otras casas ni permiten escrituras (solo lectura).
- **REQ-NF-006 — Responsive.** Las cuatro tabs deben funcionar en móvil (375px), tablet y desktop, con scroll horizontal controlado en gráficos y tablas.
- **REQ-NF-007 — Darkmode.** Todos los componentes nuevos usan tokens del tema (`bg-card`, `text-text`, `text-text-secondary`, `border-border`); prohibido `bg-white`, `text-black` o colores hardcodeados de fondo.

---

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se revisó la implementación actual de ambos homes y de los componentes reutilizables:

- `frontend/src/pages/DeudasPage.tsx`: tabs con `useSearchParams`, vistas con contadores, cards, historial agrupado por año.
- `frontend/src/components/DebtAnalysis.tsx`: resumen mensual, multi-moneda, donut, detalle por deuda, búsqueda y pago.
- `frontend/src/components/DebtCalendar.tsx`: calendario por `due_date`, agenda móvil, pago.
- `frontend/src/components/DebtChart.tsx`, `DebtHistoryChart.tsx`, `DebtProjectionChart.tsx`: gráficos SVG sin dependencias.
- `frontend/src/pages/ServicesPage.tsx`: listado plano, selector de casa, acciones de webhook/sync/editar/eliminar.
- `frontend/src/components/BillAnalysis.tsx`: análisis por servicio (evolución, cambios de monto), reutilizable conceptualmente pero no como vista global.
- `internal/api/debt_handlers.go` + `internal/storage/debt_bill.go`: patrón de endpoint `/api/debt-bills?year=&month=` y consulta por `substr(due_date,1,7)`.
- `internal/services/billing_scheduler.go`: las facturas anuales se generan en enero con `month = 0`; las mensuales usan `billing_day`.
- `internal/models/bill.go` y `internal/models/service.go`: campos disponibles (`due_date`, `issue_date`, `paid_at`, `billing_day`, `frequency`, `suggested_amount`, `active`, `is_recurring`, `start_date`, `end_date`).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A. Reutilizar los componentes de Deudas parametrizándolos por tipo de entidad | Menos código duplicado | Los modelos y semánticas difieren (cuotas vs facturas, anuales, casas); acopla dos módulos y eleva el riesgo de regresión | ❌ Rechazada |
| B. Crear componentes espejo (`ServiceAnalysis`, `ServiceCalendar`, `ServiceChart`) con los mismos patrones visuales | Bajo riesgo, cambios aislados, evolución independiente, estilo idéntico | Alguna duplicación de layout | ✅ Seleccionada |
| C. Mantener el listado y agregar sólo un dashboard arriba | Menor esfuerzo | No cumple "tal cual" ni la navegación por tabs/URL | ❌ Rechazada |
| D. Agregar una librería de gráficos (recharts/chart.js) | Más rápido de escribir | Bundle grande, RAM/CPU en iHost, viola minimización de dependencias | ❌ Rechazada |
| E. Calcular agregados en el frontend trayendo todas las facturas | Menos endpoints | Payload y memoria crecientes; lógica de negocio fuera de `internal/services` | ❌ Rechazada |
| F. Agregados en SQL + endpoints de lectura | Payload chico, rápido en iHost, lógica en la capa correcta | Requiere 2 endpoints nuevos | ✅ Seleccionada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001 — Componentes espejo, no parametrización de Deudas.**
- **Contexto**: Deudas y Servicios comparten el patrón de tabs, pero los modelos difieren (cuotas con `installment_number` vs facturas con `month=0` anual, `due_date` opcional, casa y webhook).
- **Decisión**: crear componentes nuevos en `frontend/src/components/` reutilizando los patrones y clases visuales de Deudas, y compartir sólo utilidades puras (fechas, formato, agrupación).
- **Consecuencias**: duplicación acotada de JSX; menor riesgo de romper Deudas; ambos módulos pueden evolucionar por separado.

**ADR-002 — Estado 100% en querystring.**
- **Contexto**: El requerimiento exige que la casa se mantenga al cambiar de tab y que las vistas sean compartibles.
- **Decisión**: `useSearchParams` como única fuente de verdad para `tab`, `home`, `status`, `year`, `month` y `gview`. No se usa estado local para filtros persistibles; sólo para datos de UI efímeros (modal abierto, día seleccionado, búsqueda).
- **Consecuencias**: back/forward y refresh funcionan; los links son compartibles; hay que validar y normalizar valores inválidos.

**ADR-003 — Período efectivo calculado en backend.**
- **Contexto**: `due_date` es opcional (SPEC-081) y las facturas anuales usan `month = 0`. Sin una regla única, Análisis, Calendario e Histórico darían totales distintos.
- **Decisión**: implementar la regla de REQ-007 en una función de la capa de servicios y exponer en la respuesta `period_year`, `period_month` y `period_date` (fecha efectiva). La UI no recalcula la regla.
- **Consecuencias**: consistencia entre tabs; el contrato de API queda explícito; los tests unitarios cubren los tres casos.

**ADR-004 — Multi-moneda sin conversión.**
- **Contexto**: El proyecto no maneja tipos de cambio y los servicios pueden estar en NIO/USD.
- **Decisión**: nunca sumar montos de monedas distintas. Las respuestas agrupan por `currency_code`; la UI muestra líneas por moneda y obliga a elegir una para gráficos.
- **Consecuencias**: totales honestos; el usuario debe cambiar de moneda para ver el detalle de la otra.

**ADR-005 — Endpoints de sólo lectura y agregados en SQL.**
- **Contexto**: iHost con recursos limitados y regla de capas del proyecto.
- **Decisión**: `GET /api/service-bills` (detalle del mes) y `GET /api/service-bills/summary` (histórico agregado), más `GET /api/service-bills/projection` (proyección estimada). Toda la lógica en `internal/services`, SQL en `internal/storage`.
- **Consecuencias**: payload chico, sin N+1, fácil de testear.

---

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[ServicesPage (React)]
   |  tab/home/status/year/month/gview en querystring
   +--> ServiceAnalysis ----> GET /api/service-bills?year&month&home_id
   +--> ServiceCalendar ----> GET /api/service-bills?year&month&home_id
   +--> ServicesList   ----> GET /api/services?home_id
   +--> ServiceChart   ----> GET /api/service-bills/summary?from&to&home_id
                        \--> GET /api/service-bills/projection?from&months&home_id

[internal/api] --(handlers sólo parsean/responden)--> [internal/services]
                                                        |
                                                        v
                                                  [internal/storage]
                                                        |
                                                        v
                                                    [SQLite]
```

### 4.2 Componentes

#### 4.2.1 `ServicesPage.tsx` (refactor)
- **Responsabilidad**: shell del home, tabs, lectura/escritura del querystring, carga de casas/servicios, títulos y estados vacíos globales.
- **Interfaz**: `useSearchParams`; renderiza `ServiceAnalysis`, `ServiceCalendar`, listado y `ServiceChart`.
- **Ubicación**: `frontend/src/pages/ServicesPage.tsx`.

#### 4.2.2 `ServiceAnalysis.tsx` (nuevo)
- **Responsabilidad**: resumen mensual, multi-moneda, donut por servicio, detalle pagado/pendiente, búsqueda y pago.
- **Dependencias**: `Select`, `Icon`, `LoadingSpinner`, `PayBillModal`, `useCurrencyFormatStore`.
- **Ubicación**: `frontend/src/components/ServiceAnalysis.tsx`.

#### 4.2.3 `ServiceCalendar.tsx` (nuevo)
- **Responsabilidad**: calendario/agenda mensual por fecha efectiva, totales del día y pago.
- **Dependencias**: `PayBillModal`, `Icon`.
- **Ubicación**: `frontend/src/components/ServiceCalendar.tsx`.

#### 4.2.4 `ServiceChart.tsx` (nuevo)
- **Responsabilidad**: histórico de 12 meses y proyección de 12 meses, selector de moneda y sub-vista `gview`.
- **Dependencias**: `Select`, SVG propio.
- **Ubicación**: `frontend/src/components/ServiceChart.tsx`.

#### 4.2.5 `ServicesList` (extraído de `ServicesPage`)
- **Responsabilidad**: cards de servicios con filtros y acciones actuales.
- **Ubicación**: `frontend/src/components/ServicesList.tsx` (o dentro de `ServicesPage` si se prefiere mantener el archivo único; la spec permite ambas).

#### 4.2.6 Backend
- `internal/models/service_bill.go`: structs `ServiceBillDetail`, `ServiceBillSummaryRow`, `ServiceProjectionPeriod`.
- `internal/storage/service_bill.go`: consultas de detalle por período, agregado por rango y servicios con/sin factura.
- `internal/services/service_bill.go`: regla de período efectivo, agrupación multi-moneda y proyección.
- `internal/api/service_bill_handlers.go`: handlers de sólo lectura.
- `internal/api/routes.go`: registro de rutas bajo `authMiddleware`.

### 4.3 Modelo de datos

No hay cambios de esquema. Se usan:

```
services
- id, home_id, name, institution, currency_id, frequency ('monthly'|'yearly')
- suggested_amount, active, billing_day, is_recurring, start_date, end_date
- deleted_at

bills
- id, service_id, year, month (0 = anual), amount, status ('pending'|'paid')
- issue_date, due_date, paid_at, invoice_number, payment_reference, deleted_at

homes / currencies
- homes.id, homes.name
- currencies.id, currencies.code, currencies.symbol
```

Regla de período efectivo (REQ-007):

```
period_year  = year(due_date)        si due_date != NULL
             = bills.year            si month BETWEEN 1 AND 12
             = bills.year            si month = 0 (anual sin vencimiento)

period_month = month(due_date)       si due_date != NULL
             = bills.month           si month BETWEEN 1 AND 12
             = 1                     si month = 0 (anual sin vencimiento)

period_date  = due_date              si due_date != NULL
             = period_year-MM-billing_day (clampeado al último día del mes)
             = period_year-MM-01     si billing_day es NULL
```

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/service-bills`

**Query params**: `year` (int, obligatorio), `month` (1-12, obligatorio), `home_id` (int, opcional).

**Response 200**:
```json
[
  {
    "id": 1,
    "service_id": 10,
    "service_name": "Servicio Demo",
    "institution": "Proveedor Demo",
    "home_id": 1,
    "home_name": "Casa Demo",
    "icon_key": "wifi",
    "currency_id": 1,
    "currency_code": "NIO",
    "currency_symbol": "C$",
    "frequency": "monthly",
    "billing_day": 15,
    "active": true,
    "year": 2026,
    "month": 11,
    "period_year": 2026,
    "period_month": 11,
    "period_date": "2026-11-15",
    "amount": 1000.00,
    "invoice_number": "F000000000000001",
    "status": "pending",
    "issue_date": "2026-11-01",
    "due_date": null,
    "paid_at": null,
    "payment_reference": ""
  }
]
```

**Errores**: `400 invalid_year`, `400 invalid_month`, `401 unauthorized`.

Notas:
- Excluye `bills.deleted_at IS NOT NULL` y servicios con `deleted_at IS NOT NULL`.
- Aplica `home_id` sobre `services.home_id`.
- Ordena por `period_date`, nombre de servicio.

#### Endpoint: `GET /api/service-bills/summary`

**Query params**: `from=YYYY-MM` (obligatorio), `to=YYYY-MM` (obligatorio), `home_id` (opcional).

**Response 200**:
```json
{
  "from": "2025-12",
  "to": "2026-11",
  "periods": [
    {
      "period": "2026-11",
      "year": 2026,
      "month": 11,
      "currency_code": "NIO",
      "currency_symbol": "C$",
      "total": 1500.00,
      "paid": 1000.00,
      "pending": 500.00,
      "count": 3
    },
    {
      "period": "2026-11",
      "year": 2026,
      "month": 11,
      "currency_code": "USD",
      "currency_symbol": "$",
      "total": 120.00,
      "paid": 120.00,
      "pending": 0.00,
      "count": 2
    }
  ]
}
```

**Errores**: `400 invalid_range`, `400 invalid_period`, `401 unauthorized`.

Notas:
- Devuelve filas planas por período + moneda; el frontend completa los períodos faltantes con cero.
- Una sola consulta con `GROUP BY period, currency_code`.

#### Endpoint: `GET /api/service-bills/projection`

**Query params**: `from=YYYY-MM` (default: mes actual), `months` (default 12, rango 1-24), `home_id` (opcional).

**Response 200**:
```json
{
  "from": "2026-11",
  "months": 12,
  "periods": [
    {
      "period": "2026-11",
      "year": 2026,
      "month": 11,
      "currency_code": "NIO",
      "currency_symbol": "C$",
      "confirmed": 500.00,
      "estimated": 300.00,
      "total": 800.00,
      "services": [
        { "service_id": 10, "name": "Servicio Demo", "amount": 200.00, "source": "bill" },
        { "service_id": 11, "name": "Servicio Demo 2", "amount": 100.00, "source": "bill" },
        { "service_id": 12, "name": "Servicio Demo 3", "amount": 300.00, "source": "estimate" }
      ]
    }
  ],
  "estimate_note": true
}
```

**Errores**: `400 invalid_period`, `400 invalid_months`, `401 unauthorized`.

Notas:
- `confirmed` = facturas pendientes del período (existentes).
- `estimated` = suma de `suggested_amount` de servicios activos, recurrentes, vigentes y sin factura en el período.
- `services[].source` = `bill` | `estimate`.
- Se devuelven filas por moneda; el frontend elige una moneda para graficar.

### 4.5 Dependencias

- **Internas**: `internal/services` (lógica de período/proyección), `internal/storage` (consultas), `internal/api` (handlers/rutas), `frontend/src/pages/ServicesPage.tsx`, componentes nuevos, `frontend/public/i18n/{es,en}.json`.
- **Externas**: ninguna nueva. Se reutilizan React, React Router, Tailwind y SVG nativo.

---

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Al abrir `/services` sin parámetros se muestra la tab Análisis, la casa "Todas" y el mes actual.
- [ ] CA-002: Al cambiar de tab, el parámetro `home` (y `year`, `month`, `status`) se conserva en la URL. Ejemplo: `/services?tab=analisis&home=1&year=2026&month=11` → click en Calendario → `/services?tab=calendario&home=1&year=2026&month=11`.
- [ ] CA-003: Un refresh o un atajo atrás/adelante reconstruye exactamente la misma tab, casa, período y filtro.
- [ ] CA-004: Una URL compartida con `tab`, `home`, `status`, `year`, `month` y `gview` abre la vista correspondiente; los valores inválidos caen a los defaults sin romper la página.
- [ ] CA-005: El selector de casa aplica a las cuatro tabs. Con una casa seleccionada, Análisis, Calendario, Servicios y Gráfica sólo muestran datos de esa casa.
- [ ] CA-006: En Análisis, `total = pagado + pendiente` para cada moneda; el porcentaje pagado coincide con `pagado / total`.
- [ ] CA-007: Una factura con `due_date` se agrupa en el mes de `due_date`; una mensual sin `due_date`, en `bills.year/month`; una anual (`month=0`) sin `due_date`, en enero de `bills.year`. Los tres casos coinciden en Análisis, Calendario e Histórico.
- [ ] CA-008: El Calendario muestra las facturas en su fecha efectiva, con puntos verdes (pagadas) y amarillos (pendientes), y el detalle del día con total pendiente y acción Pagar.
- [ ] CA-009: La tab Servicios conserva las cards y sus acciones (webhook, sync, editar, eliminar) y agrega los filtros con contadores; el filtro activo se refleja en `status`.
- [ ] CA-010: La Gráfica histórica muestra 12 meses (sin huecos), con total/pagado/pendiente por mes, y nunca suma monedas distintas.
- [ ] CA-011: La proyección muestra 12 meses con `confirmed + estimated = total`, no duplica servicios con factura existente y respeta servicios anuales en enero.
- [ ] CA-012: Pagar una factura desde Análisis o Calendario actualiza el estado sin recargar toda la página y muestra toast.
- [ ] CA-013: Los estados vacíos (sin casas, sin servicios, sin facturas del mes, sin datos de gráfica) muestran un `EmptyCard` con acción útil.
- [ ] CA-014: Todos los textos nuevos existen en `es.json` y `en.json`; no hay strings hardcodeados en los componentes nuevos.
- [ ] CA-015: El título del documento cambia según la tab activa.
- [ ] CA-DARK: Los componentes nuevos usan tokens del tema (`bg-card`, `text-text`, `text-text-secondary`, `border-border`) y se verificó legibilidad en darkmode (texto, placeholders y gráficos).
- [ ] CA-SELECT: Todos los selectores (mes/período, moneda, filtros) usan el componente `Select` del repo; no hay `<select>` nativo (`npm run check:ui` pasa).
- [ ] CA-BACK: No se crean páginas de detalle nuevas. Las rutas existentes `/services/bills/:id`, `/services/new` y `/services/edit/:id` mantienen su flecha atrás en el header vía `BACK_ROUTES`; no se agregan links "← Título" en el contenido.

### 5.2 No funcionales

- [ ] CA-NF-001: `GET /api/service-bills` y `GET /api/service-bills/summary` responden en < 300 ms con una copia de producción de hasta 5.000 facturas.
- [ ] CA-NF-002: El build de frontend no incorpora dependencias nuevas y el bundle crece < 15 KB gzip.
- [ ] CA-NF-003: Las consultas no generan N+1; se verifica con logs/`EXPLAIN QUERY PLAN`.
- [ ] CA-NF-004: La página funciona en 375px, 768px y 1280px sin overflow horizontal fuera de los contenedores con scroll.
- [ ] CA-NF-005: Los endpoints nuevos exigen sesión y sólo devuelven datos de lectura.

### 5.3 Testing

- **Unit tests (Go)**:
  - `periodYearMonth` / `periodDate`: due_date válido, mensual sin due_date, anual sin due_date, `billing_day` mayor al último día del mes, `billing_day` nulo, `due_date` inválido.
  - Agregado `summary`: total/pagado/pendiente por moneda y por período; meses sin datos.
  - Proyección: servicio mensual, anual, inactivo, no recurrente, fuera de `start_date`/`end_date`, servicio con factura existente (no duplica).
  - Filtro por `home_id` y exclusión de soft-deleted.
- **Integration tests (Go)**:
  - `GET /api/service-bills` con y sin `home_id`, parámetros inválidos (400).
  - `GET /api/service-bills/summary` con rango inválido.
  - `GET /api/service-bills/projection` con `months` fuera de rango.
- **Frontend**:
  - `npm run build` y `npm run check:ui` pasan.
  - Prueba manual: navegación de tabs con casa seleccionada, refresh, back/forward, deep link, darkmode, móvil.
- **Producción (copia)**:
  - Validar contra una copia de la DB de producción (`.backup` vía SSH a `/tmp`), nunca contra `data/app.db` local, comparando totales de Análisis/Calendario/Gráfica con consultas SQL independientes.

---

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Backend: modelos, storage, servicios (período efectivo, summary, proyección) y handlers | 2 días | Ninguna |
| 2 | Refactor de `ServicesPage`: tabs, querystring, selector de casa global, listado con filtros | 1.5 días | Fase 1 |
| 3 | `ServiceAnalysis` (resumen, donut, detalle, búsqueda, pago) | 1.5 días | Fase 2 |
| 4 | `ServiceCalendar` (grilla, agenda móvil, pago) | 1 día | Fase 2 |
| 5 | `ServiceChart` (histórico 12 meses + proyección 12 meses + tabla) | 2 días | Fase 1 |
| 6 | i18n es/en, estados vacíos, responsive, darkmode y QA | 1 día | Fases 2-5 |
| 7 | Validación contra copia de producción y ajustes | 0.5 días | Fase 6 |

**Total estimado**: ~9.5 días de agente.

### 6.2 Milestones

1. **MVP**: tabs + URL state + Análisis mensual + listado con filtros. (Fases 1-3)
2. **V1.0**: Calendario + Gráfica histórica + proyección + i18n + responsive/darkmode. (Fases 4-7)

---

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Totales distintos entre tabs por reglas de período ambiguas | Media | Alto | ADR-003: una sola función de período efectivo, testeada y usada por todos los endpoints |
| Facturas anuales sin `due_date` invisibles o mal ubicadas | Alta | Medio | Regla explícita: enero del año (coincide con el scheduler); badge "Fecha estimada" cuando aplique |
| Mezcla de monedas en totales/gráficos | Media | Alto | Agrupar siempre por `currency_code`; obligar a elegir moneda para gráficos; tests con NIO/USD |
| Doble conteo en la proyección (factura + estimado del mismo servicio) | Media | Medio | Excluir el estimado si el servicio ya tiene factura en el período; test unitario específico |
| Regresión en las acciones de las cards (webhook, sync, editar, eliminar) | Media | Medio | Mantener el componente de card y sus menús; pruebas manuales de cada acción |
| Payload/memoria en iHost con muchos años de facturas | Baja | Medio | Endpoints agregados, rango acotado, sin cargar facturas crudas de 12 meses |
| Pérdida de contexto al volver de un detalle | Media | Bajo | REQ-017: propagar querystring; si no es viable, documentar la limitación |
| Confusión entre `status` de la vista y `status` de factura | Baja | Bajo | Documentar que `status` en la URL es el filtro de la tab Servicios; las facturas usan `bills.status` |
| Valores inválidos en la URL rompen la vista | Media | Medio | Validación con fallback a defaults y normalización silenciosa |

---

## 8. Notas y Referencias

- **Precedentes**: SPEC-054 (módulo Deudas), SPEC-055 (análisis de deudas por mes), SPEC-100/101/102 (gráficas y responsividad de Deudas), SPEC-104/105/106/107 (historial, evolución y proyección), SPEC-113 (formato de moneda), SPEC-081 (fechas de emisión/vencimiento).
- **Patrones de UI**: `frontend/src/pages/DeudasPage.tsx`, `frontend/src/components/DebtAnalysis.tsx`, `DebtCalendar.tsx`, `DebtChart.tsx`, `DebtHistoryChart.tsx`, `DebtProjectionChart.tsx`.
- **Reglas del repo**: `AGENTS.md` (worktrees, UI, i18n, darkmode, `Select` obligatorio), `docs/project-rules.md` §4 y §8.
- **Datos sintéticos** para ejemplos y tests: `Servicio Demo`, `Proveedor Demo`, `Casa Demo`, `F000000000000001`, `demo-account-001`.
- **Fuera de alcance**: conversión de monedas, edición de facturas desde el home global, notificaciones, cambios de esquema, migraciones, exportación PDF.

---

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-03 | opencode | Creación inicial de la especificación a pedido del usuario (replicar el home de Deudas en Servicios, con estado en querystring y casa persistente). |
