---
title: "Flecha atrás en el header para páginas de detalle en lugar del link '← Título' + regla de UI permanente"
id: "SPEC-063"
status: "released"
author: "paulomcnally"
created: "2026-09-04"
updated: "2026-09-04"
github_issue: 64
---

# Flecha atrás en el header para páginas de detalle en lugar del link '← Título' + regla de UI permanente

**ID**: SPEC-063  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-09-04  
**Actualizado**: 2026-09-04

---

## 1. Resumen Ejecutivo

Al entrar a una deuda específica (`/deudas/:id`) hoy existe, dentro del contenido de la página, un link con flecha hacia atrás y el título de la sección (`← Deudas`). El mismo patrón se repite en `BillsPage` (`← Servicios`) y `AutoShowPage` (`← Volver`). Esta spec elimina esa sección de los contenidos y traslada la navegación "volver a la lista" al header de la aplicación: en las páginas donde existe una lista padre, el ícono de hamburguesa (menú que abre el sidebar, visible solo en móvil) se reemplaza por una flecha atrás que vuelve a la lista.

La motivación es de consistencia y UX: el header es el lugar natural para la acción de retroceder (patrón iOS), y el link dentro del contenido duplica la navegación y consume espacio vertical en páginas de detalle. Además, se documenta el patrón como regla de UI del proyecto para que todas las páginas futuras de detalle lo repliquen sin volver a crear links de retorno dentro del contenido.

Impacto iHost: cambio 100% frontend (React), sin dependencias nuevas, sin impacto en memoria/CPU/SQLite. El backend no se toca.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Eliminar la sección del link "← Título" (flecha atrás + texto del título de la sección) del contenido de las páginas de detalle que hoy lo tienen: `DebtBillsPage.tsx` (líneas 55-63, `← Deudas`), `BillsPage.tsx` (líneas 76-84, `← Servicios`) y `AutoShowPage.tsx` (líneas 59-65, `← Volver`).
2. **REQ-002**: En el header (`DashboardLayout.tsx`), reemplazar el ícono de hamburguesa (menú que abre el sidebar) por una flecha atrás en las páginas donde existe una ruta "volver a la lista". La flecha navega a la lista padre. En páginas sin lista padre (listados raíz, home, settings, etc.) el header sigue mostrando la hamburguesa en móvil.
3. **REQ-003**: Definir un mapa central de rutas "back" (patrón de pathname → ruta destino de la lista) en `DashboardLayout.tsx` para que el header sepa cuándo mostrar la flecha. El mapa cubre al menos las 3 páginas de detalle actuales:
   - `/deudas/:id` → `/deudas`
   - `/services/bills/:serviceId` → `/services`
   - `/autos/:id` → `/autos`
4. **REQ-004**: Mantener el título de sección del header intacto en estas páginas (el título se sigue derivando de `activeBase`; no se agrega el nombre del registro al header).
5. **REQ-005**: Documentar la regla como estilo/regla de UI del proyecto en `docs/project-rules.md` (sección 4 Reglas de UI), en `AGENTS.md` (Reglas Fundamentales) y en `docs/specs/templates/spec-template.md` (consideraciones/criterios), para que futuras páginas de detalle usen la flecha del header y **nunca** creen links "← Título" dentro del contenido.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-006**: Extender el mapa de rutas "back" a las páginas de formulario/edición que entran desde una lista, para consistencia: `/home/new` y `/home/edit/:id` → `/home`; `/services/new` y `/services/edit/:id` → `/services`; `/institutions/new` y `/institutions/edit/:id` → `/institutions`; `/autos/new` y `/autos/edit/:id` → `/autos`; `/deudas/new` y `/deudas/edit/:id` → `/deudas`; rutas del módulo pensión (hijos/categorías/salarios/notificaciones) → su listado; `/bills/new` y `/bills/edit/:id` → `/services` (o `/services/bills/:serviceId` cuando venga de ahí).
2. **REQ-007**: Evaluar sub-páginas de settings (`/settings/language`, `/settings/currency`, `/settings/currency/:id`) con retorno a `/settings`.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-008**: Evaluar mostrar la flecha atrás también en desktop (lg+) en páginas de detalle, como affordance adicional de navegación (el sidebar sigue visible).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Sin impacto. Lógica de un `match` de rutas en el header; sin JS adicional en runtime.
- **Seguridad**: Sin cambios de datos ni de APIs.
- **Almacenamiento**: Sin cambios en SQLite. Cero migraciones.
- **Disponibilidad**: No hay regresión en navegación: la acción de volver sigue disponible (flecha del header en móvil, sidebar en desktop, botón del navegador).
- **iHost**: Cero dependencias nuevas. Bundle crece < 1 KB. Backend intacto.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Header actual**: `DashboardLayout.tsx:41-50` renderiza un `<button>` con ícono `menu` (hamburguesa) que abre el sidebar, visible solo en móvil (`lg:hidden`). El título se deriva de `activeBase` (primer segmento del pathname) con `t(`${activeBase}.title`)`.
- **Páginas con link "← Título" en el contenido** (patrón a eliminar):
  - `DebtBillsPage.tsx:55-63`: botón con `Icon name="back"` + `t('menu.deudas')` → `navigate('/deudas')`.
  - `BillsPage.tsx:76-84`: botón con `Icon name="back"` + `t('menu.services')` → `navigate('/services')`.
  - `AutoShowPage.tsx:59-65`: botón con `Icon name="back"` + texto "Volver" → `navigate('/autos')`.
- **Rutas involucradas** (`App.tsx:86,99,116`): `services/bills/:serviceId`, `autos/:id`, `deudas/:id`.
- **Ícono `back` ya existe** en `Icons.tsx` (usado por las 3 páginas), se reutiliza en el header.
- **Patrón iOS**: las pantallas de detalle muestran una flecha atrás en el header (top bar), no links dentro del contenido. Es el patrón de referencia para la decisión.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A: Mapa central de rutas "back" en `DashboardLayout.tsx` (patrón de pathname → ruta destino) | Un solo punto de verdad; el header decide sin tocar cada página; fácil de auditar/extender | Requiere mantener el mapa al agregar páginas de detalle nuevas (mitigado por la regla de proyecto) | ✅ Seleccionada |
| B: Declaración por página (contexto `BackRoute` que cada página setea) | Flexible por página | Más código por página; fácil de olvidar; el patrón se degrada sin la regla | ❌ Rechazada |
| C: Usar `history.back()` en el header para todas las páginas | Cero configuración | Comportamiento impredecible (puede salir de la app o ir a una página sin contexto); no garantiza volver a la lista | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: La flecha atrás se decide desde un mapa central de rutas en el header, no desde cada página.
- **Contexto**: Las páginas de detalle hoy duplican la navegación de retorno con links en el contenido. El header ya es el único lugar donde está la hamburguesa en móvil; centralizar la decisión evita que cada página nueva tenga que acordarse de configurar su back.
- **Decisión**: En `DashboardLayout.tsx`, una constante `BACK_ROUTES: { pattern: RegExp; to: string }[]` se evalúa contra `location.pathname`. Si hay match, el botón izquierdo del header es una flecha que navega a `to`; si no, se muestra la hamburguesa (móvil).
- **Consecuencias**: Agregar una página de detalle nueva implica agregar una entrada al mapa (documentado como regla). La navegación de retorno queda consistente y testeable.

**ADR-002**: La flecha atrás reemplaza la hamburguesa solo en móvil (`lg:hidden`); en desktop el sidebar fijo sigue siendo la navegación.
- **Contexto**: La hamburguesa existe únicamente en móvil porque en desktop el sidebar está siempre visible. La petición del usuario es "en vez de mostrar el ícono de sandwich, mostrar una flecha atrás".
- **Decisión**: En páginas con ruta back, el slot izquierdo del header muestra la flecha atrás con la misma visibilidad que tenía la hamburguesa (`lg:hidden`). En desktop no se agregan elementos.
- **Consecuencias**: Cambio mínimo y fiel a la petición; en desktop la vuelta a la lista se hace desde el sidebar. La mejora de mostrar la flecha también en desktop queda como P2 (REQ-008).

**ADR-003**: Regla permanente de UI en project-rules.md + AGENTS.md + template de specs.
- **Contexto**: Este patrón (link de retorno dentro del contenido) es un defecto recurrente de proceso que apareció en varias páginas por separado. La regla debe vivir en las fuentes de verdad del proyecto para que futuras páginas de detalle usen la flecha del header.
- **Decisión**: Documentar en `docs/project-rules.md` (Reglas de UI) que las páginas de detalle deben usar la flecha del header (vía mapa `BACK_ROUTES`) y no crear links "← Título" en el contenido; reflejar en `AGENTS.md` y en el template de specs.
- **Consecuencias**: El checklist de calidad obliga a verificar el uso de la flecha del header en páginas de detalle; si se omite, la spec no pasa a pending_release.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[DashboardLayout.tsx]  --BACK_ROUTES (pathname → to)-->  decide slot izquierdo del header
        |                                                         |
        v                                                         v
[location.pathname]                                [flecha atrás (lg:hidden) | hamburguesa (lg:hidden)]
                                                         |
                                                         v
[Páginas de detalle]  --elimina link '← Título'-->  [navegación 'volver a lista' solo en header]
```

### 4.2 Componentes

#### 4.2.1 Header de `DashboardLayout.tsx`
- **Responsabilidad**: Decidir el slot izquierdo del header según la ruta actual.
- **Interfaz**:
  ```tsx
  const BACK_ROUTES: { pattern: RegExp; to: string }[] = [
    { pattern: /^\/deudas\/\d+$/, to: '/deudas' },
    { pattern: /^\/services\/bills\/\d+$/, to: '/services' },
    { pattern: /^\/autos\/\d+$/, to: '/autos' },
  ]
  const backRoute = BACK_ROUTES.find(r => r.pattern.test(location.pathname))
  ```
  - Si `backRoute`: `<button onClick={() => navigate(backRoute.to)} className="lg:hidden ...">` con `<Icon name="back" />`.
  - Si no: el botón de hamburguesa actual (abre sidebar).
- **Dependencias**: `useLocation`, `useNavigate`, `Icon` (ya disponibles).
- **Ubicación**: `frontend/src/components/DashboardLayout.tsx`.

#### 4.2.2 Páginas de detalle
- **Responsabilidad**: Quitar el link de retorno del contenido.
- **Interfaz**: Eliminar el bloque `{/* back link */}` (flecha + texto) de `DebtBillsPage.tsx`, `BillsPage.tsx` y `AutoShowPage.tsx`. El header ya provee la flecha.
- **Ubicación**: `frontend/src/pages/DebtBillsPage.tsx`, `frontend/src/pages/BillsPage.tsx`, `frontend/src/pages/AutoShowPage.tsx`.

#### 4.2.3 Documentación (regla de UI)
- **Responsabilidad**: Evitar recurrencia del patrón.
- **Interfaz**: Entradas en `docs/project-rules.md` (sección 4 Reglas de UI), `AGENTS.md` (Reglas Fundamentales) y `docs/specs/templates/spec-template.md` (consideraciones/criterios de aceptación).
- **Ubicación**: raíz y `docs/`.

### 4.3 Modelo de datos

Sin cambios. No hay migraciones, ni tablas, ni claves de settings.

### 4.4 APIs / Contratos

Sin cambios. No hay endpoints nuevos ni modificados.

### 4.5 Dependencias

- **Internas**: `frontend/src/components/DashboardLayout.tsx`, `frontend/src/pages/DebtBillsPage.tsx`, `frontend/src/pages/BillsPage.tsx`, `frontend/src/pages/AutoShowPage.tsx`, `docs/project-rules.md`, `AGENTS.md`, `docs/specs/templates/spec-template.md`.
- **Externas**: Ninguna (regla iHost: sin dependencias nuevas).

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: En `/deudas/:id` ya NO existe el link "← Deudas" dentro del contenido; el header (móvil) muestra una flecha atrás que navega a `/deudas`.
- [x] CA-002: En `/services/bills/:serviceId` ya NO existe el link "← Servicios" dentro del contenido; el header (móvil) muestra una flecha atrás que navega a `/services`.
- [x] CA-003: En `/autos/:id` ya NO existe el link "← Volver" dentro del contenido; el header (móvil) muestra una flecha atrás que navega a `/autos`.
- [x] CA-004: En páginas de listado raíz (ej: `/home`, `/services`, `/deudas`, `/autos`, `/settings`, `/institutions`) el header sigue mostrando la hamburguesa en móvil (no hay flecha atrás).
- [x] CA-005: El título de sección del header se mantiene correcto en las páginas de detalle (derivado de `activeBase`).
- [x] CA-006: En desktop (lg+) no hay regresión: el sidebar sigue fijo y no aparecen elementos nuevos en el header.
- [x] CA-007: (P1) El mapa `BACK_ROUTES` cubre las páginas de formulario/edición de REQ-006 y el header muestra la flecha en cada una.
- [x] CA-008: Existe la regla en `docs/project-rules.md` (Reglas de UI) que obliga a usar la flecha del header en páginas de detalle y prohíbe links "← Título" en el contenido.
- [x] CA-009: Existe la referencia a la regla en `AGENTS.md` y en el template de specs (criterios de aceptación para páginas de detalle).
- [x] CA-DARK: Si se tocan componentes visuales con texto/placeholders, se mantienen los tokens del tema (`bg-card`, `text-text`); verificar legibilidad en darkmode.

### 5.2 No funcionales

- [x] CA-NF-001: El bundle del frontend no crece de forma apreciable (< 1 KB) y no se agregan dependencias.
- [x] CA-NF-002: El build de Vite (`npm run build` en `frontend/`) pasa sin errores y el server local sirve el bundle actualizado.

### 5.3 Testing

- **Unit tests**: N/A (cambio de UI puro; la lógica del mapa de rutas es trivial). Si se extrae `BACK_ROUTES` a un helper, puede testearse el match por pathname.
- **Integration tests**: N/A (sin API).
- **E2E tests**: Prueba manual en local: recorrer `/deudas/:id`, `/services/bills/:serviceId` y `/autos/:id` en móvil (o viewport angosto) verificando flecha atrás en el header y ausencia del link en el contenido; repetir en desktop.
- **Carga/Performance**: N/A.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Implementar `BACK_ROUTES` + flecha atrás en `DashboardLayout.tsx` (reemplazo de la hamburguesa en móvil) | 0.25 día | Ninguna |
| 2 | Eliminar los links "← Título" del contenido en `DebtBillsPage.tsx`, `BillsPage.tsx` y `AutoShowPage.tsx` | 0.25 día | Fase 1 |
| 3 | (P1) Extender `BACK_ROUTES` a formularios/edición y evaluar settings | 0.25 día | Fase 1 |
| 4 | Reglas en `docs/project-rules.md`, `AGENTS.md` y `docs/specs/templates/spec-template.md` | 0.25 día | Ninguna |
| 5 | Correr server en local (`npm run build` + backend) para validación del usuario | 0.25 día | Fases 1-4 |

**Estimación total**: ~1 día (P0: fases 1-2 + 4-5).

### 6.2 Milestones

1. **MVP**: Fases 1-2 + 4 (flecha atrás en las 3 páginas de detalle + regla de proyecto).
2. **V1.0**: Fases 3 y 5 (cobertura total + validación con el usuario).

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| El patrón regex de `BACK_ROUTES` matchea rutas no deseadas (ej: `/deudas/new`, `/deudas/edit/:id`) | Media | Medio | Usar regex acotados con `\d+` para ids numéricos y evitar match de `/new`/`/edit`; verificar CA-004 |
| En móvil, las páginas de detalle pierden acceso al sidebar (la hamburguesa se reemplaza por la flecha) | Media | Bajo | Es la petición explícita del usuario; el sidebar sigue accesible desde cualquier página raíz y en desktop es fijo |
| Futuras páginas de detalle vuelven a crear links "← Título" en el contenido | Media | Alto | Regla permanente en project-rules.md + AGENTS.md + template de specs; revisión en code review |
| El build de Vite no regenera `public/` y se prueba contra el bundle viejo | Baja | Medio | Regla AGENTS.md: correr `npm run build` y verificar con el server local |

## 8. Notas y Referencias

- Páginas afectadas (patrón a eliminar): `DebtBillsPage.tsx:55-63`, `BillsPage.tsx:76-84`, `AutoShowPage.tsx:59-65`.
- Header actual: `DashboardLayout.tsx:41-50`.
- Rutas: `App.tsx:86` (`services/bills/:serviceId`), `App.tsx:99` (`autos/:id`), `App.tsx:116` (`deudas/:id`).
- Precedente de regla de proceso en UI: SPEC-060 (darkmode de inputs) documentó reglas permanentes en project-rules.md + AGENTS.md + template de specs. Esta spec sigue el mismo mecanismo para el patrón de navegación.
- Reglas AGENTS.md aplicables: "Seguir siempre el patrón de UI existente", "Cero código sin spec".

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-04 | paulomcnally | Creación inicial de la especificación |
| 2026-09-04 | paulomcnally | Implementación: `BACK_ROUTES` + flecha atrás en header de `DashboardLayout` (reemplaza hamburguesa en móvil en detalle/formularios), eliminados links "← Título" del contenido en `DebtBillsPage`, `BillsPage` y `AutoShowPage`, clave i18n `app.back`, regla de UI en project-rules.md/AGENTS.md/spec-template.md. Criterios de aceptación en pass tras pruebas manuales del usuario. |
| 2026-09-04 | paulomcnally | Released en main via merge `2b3b45d` (rama `feature/SPEC-063`, frontend-only, sin backend ni DB). Issue #64 cerrado. |