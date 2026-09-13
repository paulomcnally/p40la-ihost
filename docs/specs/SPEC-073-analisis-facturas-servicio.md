---
title: "Análisis de facturas por servicio con detección de cambios de monto"
id: "SPEC-073"
status: "released"
author: "paulomcnally"
created: "2026-09-12"
updated: "2026-09-12"
github_issue: 76
---

# Análisis de facturas por servicio con detección de cambios de monto

**ID**: SPEC-073  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-09-12  
**Actualizado**: 2026-09-12

---

## 1. Resumen Ejecutivo

Al entrar a un servicio (`/bills/:serviceId`) el usuario ve únicamente el listado de sus facturas (`BillsPage`). No existe una vista que responda de un vistazo a las preguntas que más importan para controlar un servicio: *¿cómo evolucionó el monto facturado en el tiempo?* y, sobre todo, *¿hubo cambios en el monto entre períodos?* (por ejemplo, un salto de precio, un cobro retroactivo o un error de un webhook/analizador). Hoy eso solo se detecta comparando fila por fila en la tabla.

Esta spec agrega dos pestañas en la página del servicio: **Análisis** y **Facturas**. La pestaña **Facturas** es el listado actual (sin cambios funcionales, solo reubicado). La pestaña **Análisis** replica el espíritu de `DebtAnalysis` (SPEC-055), pero adaptado a facturas: como un servicio tiene una única moneda, **no aplica el selector multi-moneda**; en cambio el foco es la **evolución del monto por período y la detección de cambios** entre períodos consecutivos.

La solución es **100% frontend**: ya existe `GET /api/services/{id}/bills` que devuelve todas las facturas del servicio (`year`, `month`, `amount`, `status`, `invoice_number`, `paid_at`), por lo que no se requieren cambios de backend, migraciones ni tablas nuevas. Los gráficos se implementan como SVG/CSS propios (sin librerías) para respetar las restricciones del iHost (mínimas dependencias, bajo consumo de memoria y bundle contenido).

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Agregar dos pestañas en `BillsPage` (`/bills/:serviceId`): **Análisis** y **Facturas**. La pestaña **Análisis es la pestaña por defecto**. El estado de la pestaña se controla con `searchParams` (`?tab=analisis|facturas`), siguiendo el patrón de `DeudasPage` (SPEC-055). La pestaña **Facturas** conserva el listado actual (cards en móvil, tabla en desktop) y todas sus acciones (editar, pagar, historial, eliminar, subir, webhook).
2. **REQ-002**: La pestaña Análisis muestra un **resumen del servicio**: cantidad de facturas registradas, total facturado, promedio por período, total pagado y total pendiente. El monto se muestra con la moneda/símbolo del servicio (una sola moneda; sin selector de moneda).
3. **REQ-003**: Mostrar un **gráfico de evolución del monto por período** (línea/área SVG) ordenado cronológicamente por `year` y `month` (los servicios anuales con `month = 0` se ubican al inicio de su año y se etiquetan como "Anual"). Debe permitir identificar visualmente subidas/bajadas del monto a lo largo del tiempo.
4. **REQ-004**: **Detección de cambios de monto**: comparar cada período con el período inmediatamente anterior y marcar los períodos donde el monto cambió, mostrando el **delta absoluto** y el **porcentaje de variación**. Se debe distinguir visualmente subas (↑) de bajas (↓), y listar los cambios con período, monto anterior, monto nuevo y variación.
5. **REQ-005**: **Filtro/selector de año** en Análisis: opción "Todos" (histórico completo) más cada año con facturas; al cambiar, el gráfico, el resumen y la lista de cambios se recalculan. La opción por defecto es "Todos" (o el año más reciente; se definirá en diseño priorizando la utilidad de detección de cambios).
6. **REQ-006**: Todo el texto nuevo debe usar i18n (`frontend/public/i18n/{es,en}.json`, fuente de verdad) y todos los inputs/selects deben usar los tokens del tema (`bg-card`, `text-text`, `text-text-secondary`), verificando legibilidad en darkmode. Los gráficos usan los colores del tema/paleta ya definida en `DebtAnalysis`.
7. **REQ-007**: Estado vacío coherente con el patrón existente (título, descripción, sin gráficos rotos) cuando el servicio no tiene facturas, reutilizando el patrón de EmptyCard/`BillsPage`.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-008**: **Gráfico de barras por período** complementario al de línea, para comparar montos lado a lado (útil cuando hay pocos períodos). Barra resaltada para el período con mayor monto.
2. **REQ-009**: Indicadores estadísticos del período seleccionado: monto **mínimo**, **máximo**, **promedio** y **mayor variación** (subida y bajada), con la etiqueta del período correspondiente.
3. **REQ-010**: Cada período del gráfico/lista de cambios es **clickeable** y navega a la factura correspondiente (edición) o al listado de Facturas filtrado por ese período, reutilizando la navegación existente.
4. **REQ-011**: Toggle de agrupación **"Por período" / "Por año"** (suma anual) para ver la evolución tanto mes a mes como año a año.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-012**: Indicar el **origen del cambio** (dashboard/webhook) para los períodos donde exista un evento `updated` en `bill_history` (SPEC-070), enriqueciendo la detección de cambios (por ejemplo, detectar un webhook que cambió el monto).
2. **REQ-013**: Exportar el análisis (CSV) con período, monto, variación absoluta y porcentual.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: El análisis opera sobre las facturas de un solo servicio (volumen pequeño, típicamente decenas). La agregación se hace con `useMemo`; render sin jank en iHost.
- **Seguridad**: Se reutiliza el endpoint autenticado existente; sin datos sensibles adicionales.
- **Almacenamiento**: Sin cambios de esquema, sin migraciones, sin almacenamiento adicional.
- **Disponibilidad**: Reutiliza `GET /api/services/{id}/bills`; sin nuevos servicios ni endpoints.
- **iHost**: **Cero dependencias nuevas** (sin librería de gráficos). Gráficos SVG/CSS ligeros. No agregar peso al bundle más allá de lo necesario. Sin cambios de runtime.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- `BillsPage` (`frontend/src/pages/BillsPage.tsx`) es la página de detalle de un servicio: carga `api.services.get(serviceId)` y `api.bills.list(serviceId)` y renderiza cards (móvil) / tabla (desktop). No tiene tabs.
- `frontend/src/api/index.ts:81` expone `bills.list(serviceId)` → `GET /api/services/{serviceId}/bills`, que devuelve `Bill[]` con `id, service_id, year, month, amount, invoice_number, status, drive_url, paid_at, payment_reference` (`frontend/src/types/index.ts:89`).
- El patrón de tabs con `searchParams` ya existe en `DeudasPage.tsx:17-60` (`TabKey`, `tab` por defecto, `setTab`), con iconos y `usePageTitle` dinámico.
- El patrón de análisis y gráficos SVG propios (donut, barras, resumen, paleta) ya existe en `frontend/src/components/DebtAnalysis.tsx` (SPEC-055). Se puede extraer/replicar el enfoque sin introducir dependencias.
- `SPEC-070` agregó `bill_history` (`GET /api/bills/{id}/history`) con eventos `created|updated|paid`, fuente `dashboard|webhook` y diffs de campos, incluyendo `amount`. Es la fuente opcional para el REQ-012 (P2).
- `frontend/package.json` no incluye librerías de gráficos (solo react, react-dom, react-router-dom, zustand). Agregar una (recharts/chart.js) aumentaría bundle y RAM, contra las restricciones de iHost.
- `BillsPage` ya es una página de detalle con flecha atrás en el header (SPEC-063), por lo que agregar tabs internas **no** requiere nuevas entradas en `BACK_ROUTES` ni links "← Título".

### 3.2 Análisis de gráficos: ¿cuál es relevante?

El requerimiento central no es "mostrar montos" (eso ya se ve en la tabla), sino **detectar cambios en los montos a lo largo del tiempo**. Evaluación de opciones:

| Tipo de gráfico | Utilidad para el objetivo | Decisión |
|-----------------|---------------------------|----------|
| **Línea/área de evolución del monto por período** | Excelente: muestra tendencia y hace evidentes saltos/bajadas de un vistazo. Es el mejor para "detectar cambios". | ✅ **Principal (REQ-003)** |
| **Variación (delta % período a período)** como anotación sobre la línea o mini-gráfico de barras | Excelente: responde directamente "¿cambió el monto y cuánto?". Complementa la línea marcando los puntos de quiebre. | ✅ **Complementario (REQ-004)** |
| **Barras por período** | Buena para comparar montos lado a lado cuando hay pocos períodos; más débil para tendencias largas. | ✅ Complementario P1 (REQ-008) |
| **Pastel/donut por categoría** | No aplica: no hay categorías ni multi-moneda en un servicio; sería engañoso. | ❌ Rechazado |
| **Multi-moneda (como Deudas)** | No aplica: un servicio tiene una sola moneda. | ❌ Rechazado (explícito en el requerimiento) |
| **Librería de gráficos (recharts/chart.js)** | Rica pero pesada; viola mínimas dependencias de iHost. | ❌ Rechazado |

**Conclusión**: el gráfico más relevante es una **línea/área de evolución del monto por período**, con **puntos resaltados y etiquetas en los períodos donde el monto cambió** respecto del período anterior (delta absoluto y %), acompañada de una **lista de cambios** ("¿cuándo y cuánto cambió?") y, como apoyo, **barras por período**. Esto responde directamente a "saber si hay cambios en montos de facturas".

### 3.3 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Reutilizar `GET /api/services/{id}/bills` y agregar en frontend | Cero backend, cero migraciones, rápido, volumen pequeño | Cálculo en cliente | ✅ Seleccionada |
| Nuevo endpoint de agregación backend | Cálculo servido | Nuevo handler + storage + rutas + tests; innecesario para pocas filas | ❌ Rechazada |
| Librería de gráficos | Gráficos ricos | Bundle/RAM; contra restricciones iHost | ❌ Rechazada |
| Gráficos SVG/CSS propios (línea, barras) | Zero dependencias, ligero, estilo iOS consistente | Implementación manual | ✅ Seleccionada |

### 3.4 Decisiones arquitectónicas (ADRs)

**ADR-001**: Agregación y gráficos en el frontend.
- **Contexto**: Ya existe el endpoint que devuelve todas las facturas del servicio; el volumen es pequeño.
- **Decisión**: Reutilizar `api.bills.list(serviceId)` y computar resumen, evolución y cambios en React con `useMemo`.
- **Consecuencias**: Sin cambios de backend ni migraciones; la lógica de agregación vive en el cliente.

**ADR-002**: Gráficos SVG/CSS propios, sin librerías.
- **Contexto**: El proyecto exige mínimas dependencias y no hay librería de gráficos instalada.
- **Decisión**: Implementar línea/área y barras en SVG/CSS con Tailwind, reutilizando el enfoque y la paleta de `DebtAnalysis`.
- **Consecuencias**: Bundle contenido, sin breaking changes de terceros, estilo consistente.

**ADR-003**: La detección de cambios se basa en la comparación período-a-período de `amount`.
- **Contexto**: El objetivo es identificar cambios en montos, no solo ver valores.
- **Decisión**: Ordenar cronológicamente y comparar cada período con el anterior; exponer delta absoluto y %; marcar subas/bajas. El `bill_history` (SPEC-070) queda como enriquecimiento P2 para indicar el origen.
- **Consecuencias**: Simple, sin backend, y responde directo al requerimiento. No distingue la causa sin el P2.

**ADR-004**: Tabs controladas por `searchParams` en `BillsPage`.
- **Contexto**: `BillsPage` es una página de detalle; se quiere mantener el patrón del resto de la app.
- **Decisión**: Usar `?tab=analisis|facturas` con Análisis por defecto, sin romper las rutas de edición/pago existentes.
- **Consecuencias**: URLs compartibles y consistentes con `DeudasPage`; bajo riesgo de regresión.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[BillsPage /bills/:serviceId]
        |
        |-- tab "analisis" --> [BillAnalysis]  (nuevo)
        |                          |
        |                          v
        |              api.bills.list(serviceId)  (existente)
        |                          |
        |                          v
        |          agregación en cliente: resumen, evolución, cambios
        |
        |-- tab "facturas" --> [listado actual: cards/tabla + acciones]
        |
        v
   [SQLite: bills]  (sin cambios)
```

### 4.2 Componentes

#### 4.2.1 `BillsPage` (modificado, `frontend/src/pages/BillsPage.tsx`)
- **Responsabilidad**: Agregar las tabs **Análisis** / **Facturas** (con `searchParams`), mantener el listado actual en la pestaña Facturas y montar `BillAnalysis` en la pestaña Análisis.
- **Dependencias**: `useSearchParams`, `Icon`, `BillAnalysis`.

#### 4.2.2 `BillAnalysis` (nuevo, `frontend/src/components/BillAnalysis.tsx`)
- **Responsabilidad**: Vista de análisis de las facturas del servicio: resumen, gráfico de evolución con marcas de cambio, lista de cambios y (P1) barras/estadísticas.
- **Props**: `{ serviceId: number; currencySymbol?: string }` (o `service`), para formatear montos con la moneda del servicio.
- **Dependencias**: `api`, `useI18nStore`, `useCurrencyFormatStore`, `Icon`, `LoadingSpinner`.
- **Ubicación**: `frontend/src/components/BillAnalysis.tsx`.

#### 4.2.3 Subcomponentes de gráficos (dentro de `BillAnalysis.tsx`)
- `BillLineChart`: SVG de línea/área con puntos y marcas de cambio.
- `BillBarChart` (P1): barras SVG por período.
- `ChangeList`: lista de cambios con delta absoluto y %.

### 4.3 Modelo de datos

Sin cambios. Se usa `Bill` existente:

```
Entidad: Bill (existente, sin modificar)
- id, service_id, year, month (0 = anual)
- amount, invoice_number, status (pending|paid), drive_url
- paid_at, payment_reference
```

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/services/{id}/bills` (existente, sin cambios)

**Response 200**: `Bill[]` (las mismas facturas que consume el listado actual).

Sin endpoints nuevos. No hay contratos que modificar.

### 4.5 Dependencias

- **Internas**: `frontend/src/pages/BillsPage.tsx` (tabs), `frontend/src/components/DebtAnalysis.tsx` (patrón de referencia, no se modifica), `frontend/src/api/index.ts` (`bills.list` existente), `frontend/src/components/Icons.tsx` (iconos `chart`, `bill`, `calendar`), `frontend/public/i18n/{es,en}.json` (nuevas claves).
- **Externas**: Ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado un usuario en `/bills/:serviceId`, entonces ve dos pestañas (**Análisis** y **Facturas**) y la pestaña por defecto es **Análisis**.
- [ ] CA-002: Dado un servicio con facturas, cuando se abre Análisis, entonces se muestra el resumen (cantidad, total facturado, promedio, pagado y pendiente) con la moneda del servicio.
- [ ] CA-003: Dado un servicio con varios períodos, cuando se abre Análisis, entonces se muestra un gráfico de evolución del monto ordenado cronológicamente (incluyendo el caso `month = 0` como "Anual").
- [ ] CA-004: Dado un cambio de monto entre períodos consecutivos, cuando se abre Análisis, entonces el período cambiado se marca visualmente (subida ↑ / bajada ↓) y se muestra su delta absoluto y porcentaje en la lista de cambios.
- [ ] CA-005: Dado el selector de año, cuando el usuario elige un año (o "Todos"), entonces el resumen, el gráfico y la lista de cambios se recalculan para ese conjunto.
- [ ] CA-006: Dado un servicio sin facturas, cuando se abre Análisis, entonces se muestra un estado vacío coherente (sin gráficos rotos).
- [ ] CA-007: Dado que el usuario está en la pestaña Facturas, entonces ve el listado actual (cards en móvil, tabla en desktop) y todas sus acciones siguen funcionando (editar, pagar, historial, eliminar, subir, webhook).
- [ ] CA-008: Dado un período con monto mayor/menor que el anterior, cuando se ve el gráfico/lista, entonces se distingue claramente la suba de la baja y la variación porcentual es correcta.
- [ ] CA-009: (P1) Dado el gráfico de barras por período, cuando hay pocos períodos, entonces los montos se comparan lado a lado y se resalta el mayor.
- [ ] CA-010: (P1) Dado el conjunto de facturas, entonces se muestran mínimo, máximo, promedio y mayor variación (suba y bajada) con su período.
- [ ] CA-011: (P1) Dado un período en el análisis, cuando el usuario hace click, entonces navega a la factura correspondiente o al listado de Facturas de ese período.
- [ ] CA-012: (P1) Dado el toggle "Por período / Por año", cuando se cambia, entonces la evolución se agrupa por mes/período o por suma anual.
- [ ] CA-DARK: Los inputs/selects nuevos usan tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y se verificó legibilidad en darkmode (texto y placeholders). Gráficos legibles en claro y oscuro.
- [ ] CA-BACK: `BillsPage` mantiene la flecha atrás del header (ya registrada en `BACK_ROUTES`); NO se crean links "← Título" dentro del contenido.

### 5.2 No funcionales

- [ ] CA-NF-001: El build de frontend (`npm run build`) compila sin errores y sin nuevas dependencias en `package.json`.
- [ ] CA-NF-002: No se realizan cambios de backend, migraciones ni esquema de DB.
- [ ] CA-NF-003: Las claves i18n nuevas se sirven correctamente desde `/i18n/es.json` y `/i18n/en.json` tras el build.
- [ ] CA-NF-004: La carga del análisis no bloquea el listado y usa el mismo fetch existente (sin llamadas duplicadas innecesarias).

### 5.3 Testing

- **Unit tests**: Ordenamiento cronológico de períodos (incluyendo `month = 0`); cálculo de delta absoluto y porcentual; cálculo de resumen (total, promedio, pagado, pendiente); agrupación por año.
- **Integration tests**: Render de la pestaña Análisis con `api.bills.list` mockeado; cambio de tab vía `searchParams`.
- **E2E tests**: Navegar entre Análisis/Facturas, cambiar de año, verificar marcas de cambio y que las acciones del listado sigan operativas.
- **Carga/Performance**: Verificar render fluido en iHost (bundle sin librería de gráficos).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Tabs Análisis/Facturas en `BillsPage` con `searchParams` (Análisis por defecto) | 0.5 día | Ninguna |
| 2 | Componente `BillAnalysis`: carga, ordenamiento cronológico, resumen y estado vacío | 1 día | Fase 1 |
| 3 | Gráfico de evolución (línea/área) + detección de cambios (marcas, delta, %) y lista | 1 día | Fase 2 |
| 4 | Selector de año, gráfico de barras, estadísticas y período clickeable (P1) | 1 día | Fase 3 |
| 5 | i18n (es/en), tokens de tema/darkmode, build y pruebas locales | 0.5 día | Fases 2-4 |
| 6 | (P2) Enriquecimiento con `bill_history` (origen del cambio) y export CSV | 1 día | Fase 3 |

### 6.2 Milestones

1. **MVP**: Tabs + Análisis con resumen, evolución del monto y detección de cambios (delta absoluto y %).
2. **V1.0**: Selector de año, barras, estadísticas, período clickeable, toggle por año, i18n y darkmode verificados.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Facturas duplicadas para un mismo período distorsionan la evolución | Media | Medio | Agrupar/sumar por período o tomar el registro más reciente; documentar la regla elegida |
| Servicios anuales (`month = 0`) desordenan la línea temporal | Media | Medio | Ubicar el período anual al inicio de su año y etiquetarlo "Anual"; cubierto por test unitario |
| Gráfico SVG con muchos períodos se vuelve ilegible | Baja | Bajo | Limitar el rango visible por año, scroll/zoom o agregación anual (toggle P1) |
| Tocar `BillsPage` introduce regresiones en acciones del listado | Media | Alto | Mantener el listado sin cambios funcionales; tabs con `searchParams`; pruebas manuales del flujo completo |
| Conflicto con otras sesiones por editar `BillsPage.tsx` | Media | Alto | Trabajar en el worktree aislado de la spec (SPEC-073); seguir AGENTS.md/SPEC-066 |

## 8. Notas y Referencias

- SPEC-055: Análisis de Deudas por Mes con Gráficos (patrón de referencia; multi-moneda **no** aplica aquí).
- SPEC-070: Historial de cambios de facturas (fuente opcional para el origen del cambio, REQ-012).
- SPEC-069: Webhooks por servicio para facturas (origen de cambios de monto).
- `frontend/src/pages/BillsPage.tsx` — página a modificar.
- `frontend/src/components/DebtAnalysis.tsx` — patrón de gráficos SVG sin librerías.
- `frontend/src/pages/DeudasPage.tsx` — patrón de tabs con `searchParams`.
- `frontend/src/api/index.ts:81` — `bills.list` reutilizado.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-12 | paulomcnally | Creación inicial de la especificación. Requerimiento de usuario: dos tabs (Análisis \| Facturas) en la página de un servicio, con análisis similar al de Deudas pero sin multi-moneda y enfocado en detectar cambios de monto entre períodos. Se incluyó análisis del gráfico relevante (línea/área de evolución + variación período a período). |
| 2026-09-12 | paulomcnally | Implementación (in_progress): tabs Análisis/Facturas en `BillsPage` vía `searchParams` (Análisis por defecto); nuevo componente `BillAnalysis` con resumen (total/pagado/pendiente, cantidad, promedio), gráfico de evolución SVG con marcas de cambio (subida ↑ rojo / bajada ↓ verde), lista de cambios con delta absoluto y %, selector de año, toggle por período/año, barras por período y estadísticas (mayor subida/bajada). i18n es/en (24 claves). Cero dependencias nuevas. Build frontend y backend OK. |
| 2026-09-12 | paulomcnally | Release: merge `feature/SPEC-073` a `main` (commit `93d3aee`). Estado `released`. |