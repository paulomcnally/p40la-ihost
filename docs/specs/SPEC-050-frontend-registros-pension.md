---
title: "Página de Registros Mensuales de Pensión Alimenticia (frontend, replicar child-support/records de P4OLA)"
id: "SPEC-050"
status: "released"
author: "p40la-ihost-team"
created: "2026-09-02"
updated: "2026-09-03"
github_issue: 50
---

# Página de Registros Mensuales de Pensión Alimenticia (frontend)

**ID**: SPEC-050  
**Estado**: released  
**Autor**: p40la-ihost-team  
**Creado**: 2026-09-02  
**Actualizado**: 2026-09-03

---

## 1. Resumen Ejecutivo

Esta spec construye la **página de Registros Mensuales** de Pensión Alimenticia en el frontend de p40la-ihost, replicando la página `child-support/records?year=2026&month=8` del proyecto P4OLA (`apps/dashboard/src/pages/child-support/RecordsPage.tsx`) pero **respetando las guías de estilo de p40la-ihost** (estilo iOS: `rounded-ios`, `shadow-ios`, cards, `CreateMenu`/`CardMenu`, `EmptyCard`, i18n es/en, `useToast`).

Es la **segunda de tres specs** (SPEC-049 backend → SPEC-050 frontend → SPEC-051 emails/generación). Consume exclusivamente la API definida en SPEC-049 (`/api/pension/records`, `/api/pension/salary-payments`, `/api/pension/closing`, upload/download de proof). **Sin análisis IA de comprobantes**: el modal "Marcar Pagado" permite subir el comprobante y llenar los campos manualmente (decisión acordada con el usuario).

La página reemplaza el placeholder actual: la ruta `/pension/registros` (SPEC-044) que hoy renderiza `PensionPage section="registros"` pasa a renderizar `RegistrosPage`.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Nueva página `frontend/src/pages/RegistrosPage.tsx` con:
   - **Header**: título "Registros mensuales" + selector de mes/año con flechas prev/next (persistido en query params `?year=&month=`, igual que P4OLA).
   - **Badge de mes cerrado** cuando `GET /api/pension/closing/:year/:month` devuelve `closed: true`.
   - **CreateMenu** (3 puntos, patrón del proyecto) con opciones: Crear registro manual, Cerrar mes (si hay registros y no está cerrado), Reabrir mes (si está cerrado).
2. **REQ-002**: **Cards resumen** (4 cards): Total, Pagado, Pendiente, Rechazado con montos calculados del listado (como P4OLA).
3. **REQ-003**: **Sección de salarios del mes** (si hay `salary_payments`): cards por salario con estado recibido/pendiente, botón "Marcar recibido" (abre modal) o "Marcar pendiente", días de atraso/adelanto vs día de pago esperado.
4. **REQ-004**: **Listado de registros** en cards (patrón grid/cards del proyecto). Cada card muestra categoría (título), hijo (subtítulo), monto con moneda, ícono de estado. Acciones según estado:
   - **pending**: editar (inline, monto+categoría), "Marcar Pagado" (modal), "Rechazar" (modal).
   - **paid**: muestra detalles de pago (fecha, método, referencia, comprobante link, conversión de moneda, notas) + botón "Marcar Pendiente".
   - **rejected**: muestra motivo + botón "Marcar Pendiente".
   - Si el mes está cerrado: sin acciones de edición/pago/rechazo (solo lectura + marcar pendiente deshabilitado).
5. **REQ-005**: **Modal "Marcar Pagado"**: subida de comprobante (multipart → `upload-proof`), campos paid_at (datetime-local), payment_method (select: bank_transfer, cash, check, mobile, other), payment_reference, evidence_notes, y conversión de moneda opcional (original_amount, original_currency, exchange_rate con cálculo automático de tasa). Sin análisis IA.
6. **REQ-006**: **Modal "Marcar Salario Recibido"**: received_at (datetime-local, requerido), received_amount, notes. Validación de diferencia vs monto esperado (info visual).
7. **REQ-007**: **Modal "Rechazar"**: textarea con motivo (requerido) → `mark-rejected`.
8. **REQ-008**: **Modal "Reabrir mes"** con confirmación por palabra (patrón existente del proyecto — ver `DeleteModal`/confirmación por palabra; en P4OLA es `ReopenMonthModal`).
9. **REQ-009**: **Formulario inline "Crear registro manual"**: select de hijo, select de categoría, monto, moneda (default NIO) → `POST /api/pension/records`.
10. **REQ-010**: `EmptyCard` cuando no hay registros ni salarios en el mes (título, descripción, botón crear) — patrón `HijosPage`.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-011**: Loading states (LoadingSpinner / skeletons) mientras cargan datos.
2. **REQ-012**: Toasts de éxito/error en todas las mutaciones (useToast del proyecto).
3. **REQ-013**: Claves i18n es/en nuevas bajo namespace `pension.records.*` (y `pension.salaryPayments.*`, `pension.closing.*`) en `frontend/public/i18n/{es,en}.json` (fuente de verdad) + `frontend/src/i18n/`.
4. **REQ-014**: Cambiar la ruta `/pension/registros` en `App.tsx` de `PensionPage section="registros"` a `RegistrosPage`.
5. **REQ-015**: Select de hijo/categoría/moneda reutiliza el componente `Select` del proyecto (NO react-select; seguir UI rules de p40la-ihost).
6. **REQ-016**: Comprobante subido se muestra como link de descarga (`GET /api/pension/records/{id}/proof`) en la card del registro pagado.
7. **REQ-017**: Días de atraso de salario: "esperado el día X", badge rojo "N días de atraso" si corresponde; si recibido, diferencia +N/N días vs fecha esperada (patrón P4OLA).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-018**: Endpoint de resumen `GET /api/pension/records/summary` (SPEC-049 REQ-015) usado para las cards si está disponible; si no, calcular en el frontend.
2. **REQ-019**: Botón "Generar mes" en el menú (depende de SPEC-051; esta spec deja el espacio/acción si SPEC-051 ya está lista).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Bundle estático sin dependencias nuevas; sin librerías de fecha externas (usar `Intl`/toLocaleDateString como en HijosPage).
- **Seguridad**: Misma auth existente (el api client redirige a /login en 401).
- **i18n**: Editar SIEMPRE `frontend/public/i18n/` (fuente de verdad) y correr `npm run build`; verificar con `curl /i18n/es.json` (regla crítica AGENTS.md).
- **iHost**: Sin Node en runtime; solo estáticos pre-build servidos por Go.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **P4OLA RecordsPage.tsx** (955 líneas): analizada en detalle. Componentes que usa: `useSupportRecords`, `useMonthClosing`, `useSalaryPayments`, `useChildren`, `useSupportCategories`, `useAnalyzePayment` (IA — se omite), `useUploadProof`, `useCloseMonth/useReopenMonth`, `EmptyState`, `ActionMenu`, `ReopenMonthModal`.
- **Estilo p40la-ihost**: patrón de listados con cards (HijosPage, AutosPage), `CreateMenu`, `CardMenu`, `DeleteModal`, `EmptyCard` (inline en HijosPage), `LoadingSpinner`, `Toast`, `Select` (componente propio). El AGENTS.md prohíbe formularios inline en listados, pero el "crear registro manual" en P4OLA es un panel inline colapsable; se adapta como **modal** o **panel colapsable** respetando el patrón del proyecto (se decide: panel colapsable bajo el header, igual que P4OLA, porque es un alta rápida dentro del mes; se confirma en revisión con el usuario).
- **Tipos y api client**: SPEC-049 define el contrato en `frontend/src/types/index.ts` y `frontend/src/api/index.ts`.
- **i18n actual**: ya existen `pension.records` y `pension.records_empty`; se agregan las claves de detalle.
- **Routing**: `App.tsx` línea 106 usa `PensionPage section="registros"` → se reemplaza por `RegistrosPage`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Pagina nueva `RegistrosPage.tsx` con la API de SPEC-049 | Foco claro, sin deuda | Más archivos | ✅ Seleccionada |
| Modificar `PensionPage` para cargar datos | Rápido | Página genérica que mezcla secciones | ❌ Rechazada |
| React Query (como P4OLA) | Cache/listo | El proyecto usa state local + fetch | ❌ Rechazada (se usa patrón existente) |
| react-select para selects | UX en P4OLA | Contradice UI rules del proyecto | ❌ Rechazada (componente `Select` propio) |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Patrón de datos con hooks + estado local (no React Query)
- **Contexto**: P4OLA usa TanStack Query; p40la-ihost usa `useState` + `useEffect` + `api.*`.
- **Decisión**: `RegistrosPage` usa estado local con función `load*` y refetch tras mutaciones (patrón `HijosPage.loadChildren`).
- **Consecuencias**: Consistente con el proyecto; sin dependencias nuevas.

**ADR-002**: Selects con componente `Select` propio
- **Contexto**: P4OLA usa react-select estilizado en oscuro.
- **Decisión**: Usar el componente `Select` existente del proyecto (ServiceFormPage, etc.) para hijo/categoría/método de pago.
- **Consecuencias**: Consistencia visual; menos JS en el bundle.

**ADR-003**: Creación manual como panel colapsable (no modal)
- **Contexto**: P4OLA usa un panel inline bajo el header.
- **Decisión**: Panel colapsable con botón "Crear registro" del CreateMenu; respeta el flujo de alta rápida sin navegación.
- **Consecuencias**: Se evita un formulario en ruta propia; el AGENTS.md de P4OLA prohíbe formularios inline en listados, pero el AGENTS.md de p40la-ihost solo define el patrón de EmptyCard/CreateMenu — se alinea con el patrón P4OLA de alta rápida y se revisa con el usuario en validación.

**ADR-004**: Query params para mes/año
- **Contexto**: P4OLA persiste `?year=&month=` en la URL.
- **Decisión**: Igual en p40la-ihost usando `useSearchParams` (react-router ya disponible).
- **Consecuencias**: URLs compartibles y navegación con botones atrás/adelante.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Sidebar /pension/registros] --> [RegistrosPage.tsx]
   |-- GET  /api/pension/records?year&month
   |-- GET  /api/pension/salary-payments?year&month
   |-- GET  /api/pension/closing/:year/:month
   |-- GET  /api/children  |  /api/pension-categories
   |-- POST/PUT/POST ... /mark-* / closing / upload-proof
```

### 4.2 Componentes

#### 4.2.1 RegistrosPage
- **Responsabilidad**: Coordina todo el estado de la página.
- **Ubicación**: `frontend/src/pages/RegistrosPage.tsx`.
- **Subcomponentes (dentro del mismo archivo o en `components/` si se reutilizan)**: `MonthSelector`, `SummaryCards`, `SalaryCard`, `RecordCard`, `PayRecordModal`, `SalaryReceivedModal`, `RejectModal`, `ReopenMonthModal`, `CreateRecordForm`.

#### 4.2.2 App.tsx
- **Responsabilidad**: Ruta `/pension/registros` → `RegistrosPage` (reemplaza placeholder).

### 4.3 Modelo de datos (contrato desde SPEC-049)

```
SupportRecord { id, child_id, child_name, pension_category_id, category_name, year, month, amount, currency, status, paid_at, payment_method, payment_reference, evidence_notes, notes, proof_file_name, original_amount, original_currency, exchange_rate }
SalaryPayment { id, salary_id, employer, year, month, amount, currency, status, received_amount, received_at, notes }
MonthClosing  { closed: bool, closed_at: string|null }
```

### 4.4 APIs / Contratos (consumidas)

- `GET /api/pension/records?year&month`
- `POST /api/pension/records` (child_id, pension_category_id, year, month, amount, currency, notes)
- `PUT /api/pension/records/{id}` (amount, pension_category_id)
- `POST /api/pension/records/{id}/mark-paid` (paid_at, payment_method, payment_reference, evidence_notes, original_amount, original_currency, exchange_rate)
- `POST /api/pension/records/{id}/mark-pending` | `mark-rejected` (reason)
- `GET /api/pension/salary-payments?year&month`
- `POST /api/pension/salary-payments/{id}/mark-received` (received_at, received_amount, notes) | `mark-pending`
- `GET/POST/DELETE /api/pension/closing/{year}/{month}`
- `POST /api/pension/records/{id}/upload-proof` (multipart `file`)
- `GET /api/pension/records/{id}/proof`
- `GET /api/children` | `GET /api/pension-categories` (existentes)

### 4.5 Dependencias

- **Internas**: `frontend/src/types/index.ts` + `frontend/src/api/index.ts` (SPEC-049), componentes `Select`, `CreateMenu`, `CardMenu`, `DeleteModal`, `LoadingSpinner`, `Toast`, `Icon`, `EmptyCard`, store i18n, `useSearchParams`.
- **Externas**: Ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: `/pension/registros` muestra la página con header, selector mes/año (inicial = mes actual) y las cards resumen.
- [ ] CA-002: Con registros cargados, el listado muestra cards con categoría, hijo, monto y estado; acciones según estado (editar/pagar/rechazar para pending; detalle de pago + marcar pendiente para paid; motivo + marcar pendiente para rejected).
- [ ] CA-003: El CreateMenu permite crear un registro manual (hijo + categoría + monto + moneda) y aparece en el listado sin recargar.
- [ ] CA-004: El modal "Marcar Pagado" permite subir un comprobante (se muestra link descargable en la card tras guardar), llenar fecha/método/referencia/notas y conversión de moneda opcional.
- [ ] CA-005: El modal "Marcar Salario Recibido" requiere fecha y guarda monto recibido/notas; el card de salario muestra el estado.
- [ ] CA-006: Cerrar mes persiste; con el mes cerrado se muestra el badge y las acciones de edición/pago/rechazo desaparecen (o quedan deshabilitadas).
- [ ] CA-007: Reabrir mes usa confirmación por palabra y reabre correctamente.
- [ ] CA-008: EmptyCard cuando no hay registros ni salarios en el mes, con botón para crear el primero.
- [ ] CA-009: Navegación prev/next cambia el mes y actualiza la URL con `?year=&month=`.
- [ ] CA-010: Las etiquetas se traducen es/en correctamente.

### 5.2 No funcionales

- [ ] CA-NF-001: `npm run build` en `frontend/` compila sin errores.
- [ ] CA-NF-002: Sin dependencias npm nuevas.
- [ ] CA-NF-003: Las claves i18n nuevas se sirven desde `frontend/public/i18n/` (verificar con `curl /i18n/es.json`).

### 5.3 Testing

- **Unit tests**: No se usan test runners en frontend del proyecto (build + validación manual); si existe algún patrón de test frontend, cubrir funciones puras (cálculo de resumen, días de atraso).
- **Integration tests**: Flujo con API real en local (SPEC-049): login → crear registros/salarios → ver en página.
- **E2E tests**: Recorrido completo de la página con el usuario (flujo de prueba definido en Fase de validación).
- **Carga/Performance**: Pagina responde fluida con un mes de 20+ registros (bundle estático, sin librerías pesadas).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | RegistrosPage: header, MonthSelector, cards resumen, EmptyCard | 45 min | SPEC-049 |
| 2 | Sección salarios + cards de salario | 30 min | Fase 1 |
| 3 | Listado de registros + acciones por estado + edición inline | 45 min | Fase 1 |
| 4 | Modales: Marcar Pagado (con proof upload), Salario Recibido, Rechazar, Reabrir | 60 min | Fase 2, 3 |
| 5 | Formulario crear registro manual + CreateMenu (crear/cerrar/reabrir) | 30 min | Fase 3 |
| 6 | Ruta en App.tsx + tipos/api client si faltan (espejo SPEC-049) | 15 min | SPEC-049 |
| 7 | i18n es/en (`frontend/public/i18n/` + `src/i18n/`) + build | 20 min | Fase 6 |
| 8 | Pruebas locales: build + server + validación manual con usuario | 45 min | Fase 7 |
| **Total** | | **~4.8 horas** | |

### 6.2 Milestones

1. **MVP**: Página funcional con listado, salarios, resumen y acciones de estado (Fases 1-5).
2. **V1.0**: MVP + modales completos + i18n + validación manual.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| El api client/contrato de SPEC-049 cambia campos | Media | Medio | Contrato congelado en SPEC-049 (Fase 7); esta spec lo consume tal cual |
| Página muy larga (como P4OLA, 955 líneas) | Media | Medio | Dividir en componentes internos reutilizables dentro de `RegistrosPage.tsx` |
| Selects/inputs que violan UI rules del proyecto | Media | Medio | Usar componentes existentes (`Select`, clases iOS del proyecto); revisar en code review |
| i18n editado solo en `src/i18n/` se pierde en build | Media | Medio | Editar `frontend/public/i18n/` SIEMPRE y correr build (regla AGENTS.md) |
| Confirmación por palabra de reapertura no existe aún | Baja | Bajo | Reusar patrón de `DeleteModal` (confirmación por palabra) o crear `ReopenMonthModal` propio siguiendo la UI del proyecto |

## 8. Notas y Referencias

- Referencia de dominio (P4OLA): `apps/dashboard/src/pages/child-support/RecordsPage.tsx`, `apps/dashboard/src/hooks/{useSupportRecords,useMonthClosing,useSalaryPayments,useChildSupportImport,useChildren,useSupportCategories}.ts`, `apps/dashboard/src/components/ui/{ActionMenu,ReopenMonthModal,EmptyState}.tsx`.
- Estilo p40la-ihost: `frontend/src/pages/HijosPage.tsx`, `AutosPage.tsx`, `components/{CreateMenu,CardMenu,DeleteModal,Select,Toast,LoadingSpinner,Icons}.tsx`, `index.css` (rounded-ios, shadow-ios).
- Specs: SPEC-044 (ruta `/pension/registros`), SPEC-049 (backend consumido), SPEC-051 (emails + generación).
- Reglas críticas AGENTS.md: i18n en `frontend/public/i18n/`, build de Vite con `emptyOutDir`, verificar `/health` antes de reportar.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-02 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-09-02 | p40la-ihost-team | Estado cambiado a pending_execution (aprobada para desarrollo) |
| 2026-09-02 | p40la-ihost-team | Estado cambiado a in_progress; inicio de desarrollo en worktree feature/SPEC-050 |
| 2026-09-02 | p40la-ihost-team | Implementación completa: RegistrosPage.tsx (header mes/año, badge cerrado, CreateMenu, cards resumen, salarios, listado por estado, edición inline, modales Marcar Pagado/Salario Recibido/Rechazar/Reabrir, crear manual, EmptyCard), ruta /pension/registros, i18n es/en (registros, salaryPayments, closing). Validado en local (npm run build, server con backend SPEC-049 + frontend). Pendiente validación manual del usuario |
| 2026-09-03 | p40la-ihost-team | Release: merge feature/SPEC-050 a main (commit 851e29f), validación manual del usuario aprobada, issue #50 cerrado, worktree limpiado |