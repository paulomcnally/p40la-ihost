---
title: "Filtro de cuotas en página de deuda"
id: "SPEC-087"
status: "released"
author: "opencode"
created: "2026-09-19"
updated: "2026-09-19"
github_issue: 90
---

# Filtro de cuotas en página de deuda

**ID**: SPEC-087  
**Estado**: released  
**Autor**: opencode  
**Creado**: 2026-09-19  
**Actualizado**: 2026-09-19

---

## 1. Resumen Ejecutivo

La página de detalle de una deuda (`/deudas/:id`, componente `DebtBillsPage.tsx`) lista **todas** las cuotas de la deuda sin distinción de estado. En deudas de largo plazo (ej: Préstamo Auto con 96 cuotas hasta 2033), el usuario tiene solo una cuota pendiente al mes pero debe escanear una lista enorme de cuotas pagadas/futuras para encontrarla. Al marcar cuotas como pagadas en lote (operación vía SSH ya realizada), el listado queda dominado por cuotas `paid`, dificultando ver lo que realmente importa: lo pendiente.

Esta spec agrega **filtros por estado** con tabs en dos niveles:
1. **Detalle de deuda** (`/deudas/:id`): tabs **Todas / Pendientes / Pagadas** sobre las cuotas, con **`pending` como filtro por defecto** (el usuario ve solo lo pendiente al entrar, sin acción extra).
2. **Lista general** (`/deudas`): dentro de la pestaña "Deudas", tabs **Todas / Activas / Inactivas / Finalizadas** sobre las deudas, con **`activa` como filtro por defecto**.

Ambos filtros se reflejan en el **URL como query param** (`?status=pending`, `?status=paid`, `?status=all`, etc.), de modo que el estado del filtro es compartible, persistente al navegar y compatible con el patrón de tabs existente de `DeudasPage.tsx` (`?tab=`).

**Impacto iHost**: solución 100% frontend, sin cambios de backend ni DB (los endpoints existentes ya devuelven todos los registros). Cero dependencias nuevas, filtrado en memoria con `Array.filter`, bundle mínimo. Aplica el patrón visual de tabs ya existente (botones pill con tokens del tema `bg-card`/`bg-primary`), sin nuevos form controls.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: En `/deudas/:id` (`DebtBillsPage.tsx`), agregar una fila de 3 tabs **Todas / Pendientes / Pagadas** entre el header de la deuda y la lista de cuotas. El filtro por defecto (sin query param) es **Pendientes** (`pending`): solo se muestran cuotas con `status='pending'`.
2. **REQ-002**: El tab activo del detalle de deuda se lee y escribe en el URL como query param `?status=all|pending|paid`. Al cambiar de tab se actualiza el param (patrón `useSearchParams`, como `setTab` de `DeudasPage.tsx`). Un URL como `/deudas/12?status=paid` abre la página con ese filtro activo.
3. **REQ-003**: En `/deudas` (`DeudasPage.tsx`), dentro de la pestaña "Deudas" (tab `deudas`), agregar una fila de tabs **Todas / Activas / Inactivas / Finalizadas** sobre las deudas. El filtro por defecto (sin query param) es **Activas** (`activa`).
4. **REQ-004**: El filtro de la lista general se refleja en el URL con el mismo query param `?status=all|activa|inactiva|finalizada`, sin romper el param existente `?tab=` (ambos coexisten; cambiar uno no debe borrar el otro).
5. **REQ-005**: Al filtrar, los contadores del detalle (progress donut "X de Y cuotas") siguen mostrando los totales reales de la deuda (no los del subconjunto filtrado); el filtro solo afecta la **lista de cuotas** renderizada.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-006**: Si el filtro activo no tiene resultados (ej: tab "Pagadas" con 0 cuotas pagadas), mostrar el estado vacío existente (`deudas.bills_empty`) en lugar de un listado vacío sin mensaje.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-007**: Las claves i18n nuevas (`deudas.filter_all`, `deudas.filter_pending`, `deudas.filter_paid`, `deudas.filter_active`, `deudas.filter_inactive`, `deudas.filter_finished`) se agregan en `frontend/public/i18n/{es,en}.json` (fuente de verdad, NUNCA `public/i18n/`).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Filtrado en memoria O(n) sobre listas ya cargadas (≤ 96 cuotas por deuda); sin requests adicionales ni latencia.
- **Seguridad**: Sin cambios de autenticación ni datos sensibles; filtros puramente visuales.
- **Almacenamiento**: Sin cambios de esquema ni datos nuevos.
- **Disponibilidad**: Sin impacto en server/health checks; cambios solo en bundle estático.
- **iHost**: Cero dependencias nuevas, sin lógica backend, sin queries SQL adicionales. Bundle impactado solo por el JS de los tabs (mínimo).

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- `DebtBillsPage.tsx` (`/deudas/:id`): carga `api.debts.get(id)` + `api.debts.listBills(id)` (todas las cuotas, sin filtro server-side) y renderiza la lista completa con el patrón CardMenu + badges de estado. El donut de progreso ya computa `paidCount`/`pendingCount` sobre todas las cuotas.
- `DeudasPage.tsx` (`/deudas`): ya implementa tabs con `useSearchParams` y `setSearchParams({ tab: key })` (patrón `TabKey = 'calendario' | 'deudas' | 'analisis'`, default `analisis`). El listado de deudas renderiza todas con `debts.map(...)`, diferenciando visualmente `activa` (bg-card) de las demás (opacity-60).
- `frontend/src/api/index.ts` expone `debts.listBills(debtId)` → todas las cuotas y `debts.list()` → todas las deudas (sin params de estado).
- Los endpoints backend no soportan filtro por estado (ni `bills` ni `debt_bills`), pero no hace falta: los volúmenes son acotados (cuotas ≤ ~96 por deuda; deudas pocas) y el filtrado en memoria es suficiente y más liviano para iHost.
- i18n: fuente de verdad `frontend/public/i18n/{es,en}.json` (regla AGENTS.md: `public/i18n/` es salida de build y se pierde).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **Filtro frontend con `useSearchParams` + `Array.filter`** | Cero backend, cero DB, instantáneo, patrón ya usado en `DeudasPage` (`?tab=`) | Los datos viajan completos (volúmenes chicos, aceptable) | ✅ Seleccionada |
| Filtro server-side (query param en API) | Menos datos en payload | Cambios en backend + storage + tests; latencia extra en iHost; innecesario para volúmenes acotados | ❌ Rechazada |
| Tabs con estado local (useState, sin URL) | Más simple | No compartible, se pierde al navegar/refrescar, contradice pedido explícito de afectar el URL | ❌ Rechazada |
| Componente `Select` dropdown | Ocupa menos espacio | Rompe patrón de tabs visual, peor UX móvil, token `select` sin necesidad | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Filtro por estado implementado 100% en frontend, en memoria.
- **Contexto**: Los endpoints de deudas/cuotas no filtran por estado y el usuario quiere filtro visual con default pendiente. Los volúmenes son acotados (≤96 cuotas/deuda, decenas de deudas) y el iHost es de recursos limitados: evitar requests extra y lógica server-side innecesaria.
- **Decisión**: `DebtBillsPage` y `DeudasPage` leen `?status=` con `useSearchParams` (default `pending` y `activa` respectivamente), filtran con `Array.filter` el array ya cargado y actualizan el param al cambiar de tab.
- **Consecuencias**: Sin cambios de API/DB/tests backend. El payload sigue trayendo todo (aceptable por volumen). El filtro es compartible vía URL.

**ADR-002**: Reutilizar el query param `status` (con valores `all|pending|paid` en detalle; `all|activa|inactiva|finalizada` en lista) y coexistir con el `tab` existente.
- **Contexto**: `DeudasPage` ya usa `?tab=`; agregar `?status=` debe preservar ambos params al navegar entre tabs y al cambiar filtro.
- **Decisión**: En `DeudasPage`, `setTab` hace `setSearchParams` preservando el `status` actual; el setter de filtro preserva el `tab` actual. En `DebtBillsPage` solo existe `status`.
- **Consecuencias**: URLs combinadas tipo `/deudas?tab=deudas&status=activa` funcionan y son estables al navegar.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[DebtBillsPage (/deudas/:id)]
   useSearchParams (?status=) ──default 'pending'──► Array.filter(bills, status)
        │                                                  │
        └── Tabs [Todas|Pendientes|Pagadas] ◄──► setSearchParams ─┘
                                                                   ▼
                                                     Lista de cuotas filtrada (CardMenu + badges)

[DeudasPage (/deudas)]
   useSearchParams (?tab= + ?status=) ──default 'activa'──► Array.filter(debts, status)
        │                                                       │
        └── Tabs [Todas|Activas|Inactivas|Finalizadas] ◄───────┘
                                                                  ▼
                                                    Grid de cards de deudas filtrado
```

### 4.2 Componentes

#### 4.2.1 `DebtBillsPage.tsx` (modificado)
- **Responsabilidad**: Agregar tabs de filtro de cuotas y filtrar la lista renderizada.
- **Interfaz**: `status` de `useSearchParams`, default `'pending'`; valores `all|pending|paid`. Tabs idénticos al patrón de `DeudasPage` (botones pill `bg-primary` activo / `bg-card` inactivo, `min-h-[44px]`).
- **Dependencias**: `useSearchParams` (react-router-dom, ya en uso en `DeudasPage`).
- **Ubicación**: `frontend/src/pages/DebtBillsPage.tsx`.

#### 4.2.2 `DeudasPage.tsx` (modificado)
- **Responsabilidad**: En el tab `deudas`, agregar tabs de filtro por estado de deuda y filtrar el grid.
- **Interfaz**: `status` de `useSearchParams` (solo relevante en `tab === 'deudas'`), default `'activa'`; valores `all|activa|inactiva|finalizada`. Preservar `tab` y `status` al setear cualquiera de los dos.
- **Dependencias**: `useSearchParams` (ya importado).
- **Ubicación**: `frontend/src/pages/DeudasPage.tsx`.

#### 4.2.3 i18n (`frontend/public/i18n/{es,en}.json` — modificado)
- **Responsabilidad**: Claves de labels de los tabs.
- **Claves nuevas**: `deudas.filter_all`, `deudas.filter_pending`, `deudas.filter_paid`, `deudas.filter_active`, `deudas.filter_inactive`, `deudas.filter_finished`.
- **Ubicación**: `frontend/public/i18n/es.json` y `frontend/public/i18n/en.json`. **Nunca** `public/i18n/` (se regenera en build).

### 4.3 Modelo de datos

Sin cambios. Se reutilizan los campos existentes:
- `DebtBill.status`: `'pending' | 'paid'`
- `Debt.status`: `'activa' | 'inactiva' | 'finalizada'`

### 4.4 APIs / Contratos

Sin cambios de API. Se reutilizan:
- `GET /api/debts/{id}/bills` (todas las cuotas de la deuda)
- `GET /api/debts` (todas las deudas)

### 4.5 Dependencias

- **Internas**: `DebtBillsPage.tsx`, `DeudasPage.tsx`, `frontend/public/i18n/{es,en}.json`.
- **Externas**: Ninguna. `useSearchParams` ya viene de `react-router-dom` (dependencia existente).

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Dado `/deudas/12` sin query param, cuando se abre la página, entonces solo se muestran las cuotas con `status='pending'` (default pendientes) y el tab "Pendientes" aparece activo.
- [x] CA-002: Dado el detalle de una deuda, cuando se toca el tab "Pagadas", entonces el URL cambia a `?status=paid` y solo se listan cuotas `paid`; al tocar "Todas", el URL es `?status=all` y se listan todas.
- [x] CA-003: Dado el URL compartido `/deudas/12?status=paid`, cuando se navega a él, entonces la página abre con el tab "Pagadas" activo y la lista filtrada (deep-link funcional).
- [x] CA-004: Dado `/deudas` sin query param, cuando se está en la pestaña "Deudas", entonces solo se muestran deudas `activa` por defecto (tab "Activas" activo).
- [x] CA-005: Dado `/deudas?tab=deudas&status=inactiva`, cuando se cambia de pestaña (ej: a "Calendario") y se vuelve a "Deudas", entonces el filtro `status=inactiva` se preserva en el URL y el listado sigue filtrado.
- [x] CA-006: Dado el detalle con cuotas pagadas y pendientes, cuando el filtro activo no tiene resultados (ej: 0 pagadas), entonces se muestra el estado vacío existente (`deudas.bills_empty`) con mensaje, no una lista vacía muda.
- [x] CA-007: Dado el detalle de una deuda con filtro activo, cuando se revisa el donut de progreso, entonces los contadores "X de Y cuotas" y montos reflejan el total real de la deuda (no el subconjunto filtrado).
- [x] CA-DARK: Los tabs usan tokens del tema (`bg-card`, `text-text-secondary`, `bg-primary`, `text-white`) y se verificó su legibilidad en darkmode (no aplican inputs nuevos; el patrón pill ya existe en `DeudasPage`).
- [x] CA-BACK: No aplica (no se crean páginas de detalle/formularios nuevos; la navegación de retorno de `/deudas/:id` ya está registrada en `BACK_ROUTES`).

### 5.2 No funcionales

- [x] CA-NF-001: Sin requests de red adicionales al filtrar (filtrado en memoria); `npm run build` en `frontend/` sin errores y el bundle sirve las claves i18n nuevas (`curl -s http://localhost:8088/i18n/es.json` las incluye).

### 5.3 Testing

- **Unit tests**: No aplica lógica backend nueva; el filtrado es inline (`Array.filter`) — se valida manualmente.
- **Integration tests**: `npm run build` + server local + verificación de URLs (`?status=paid`, `?status=all`, default sin param) y coexistencia `tab`+`status`.
- **E2E tests**: Escenarios manuales: default pendientes, cambio de tab actualiza URL, deep-link, preservación de filtro al cambiar de pestaña en `/deudas`, empty state.
- **Carga/Performance**: Con la deuda de 96 cuotas, el filtrado y render no agregan latencia perceptible en iHost.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Agregar tabs + filtro en `DebtBillsPage.tsx` (REQ-001/002/005/006) | 0.5 día | Ninguna |
| 2 | Agregar tabs + filtro en `DeudasPage.tsx` preservando `tab` (REQ-003/004) | 0.5 día | Fase 1 |
| 3 | Claves i18n es/en (REQ-007) + `npm run build` + verificación servida | 0.25 día | Fase 1-2 |
| 4 | Pruebas manuales locales con server corriendo (defaults, URLs, deep-links, darkmode) | 0.25 día | Fase 3 |

### 6.2 Milestones

1. **MVP**: Fases 1-3 — filtros funcionales con URL y i18n.
2. **V1.0**: Fase 4 — validación manual completa del usuario en local.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Perder el param `status` al cambiar `tab` en `/deudas` | Media | Medio | `setTab` y setter de filtro preservan el otro param (ADR-002) |
| i18n editado en `public/i18n/` y perdido en build | Media | Alto | Editar solo `frontend/public/i18n/` (regla AGENTS.md) y verificar con curl post-build |
| Default `pending`/`activa` confunde a quien quiere ver todo | Baja | Bajo | Tabs visibles con la opción "Todas" a un toque; URL permite deep-link |
| Valor inválido en `?status=` (ej: `?status=foo`) | Baja | Bajo | Coercer a default (`pending`/`activa`) si el valor no está en el set permitido |

## 8. Notas y Referencias

- `frontend/src/pages/DeudasPage.tsx` — patrón existente de tabs con `useSearchParams` (`?tab=`, `setTab`).
- `frontend/src/pages/DebtBillsPage.tsx` — página a modificar (lista de cuotas sin filtro).
- `frontend/public/i18n/{es,en}.json` — fuente de verdad de traducciones (AGENTS.md §0).
- SPEC-054 — modelo de deudas (`debt_bills.status` pending/paid, `debts.status` activa/inactiva/finalizada).
- SPEC-085/086 — precedente de "pendientes del mes en curso" en Telegram (misma motivación de priorizar lo pendiente).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-19 | opencode | Creación inicial de la especificación |
| 2026-09-19 | opencode | Implementación (tabs + filtros en DebtBillsPage y DeudasPage, i18n es/en) y validación manual del usuario |
| 2026-09-19 | opencode | Release: merge a main, issue #90 cerrado con label spec/released |