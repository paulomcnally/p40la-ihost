---
title: "Títulos dinámicos en el header según la página cargada"
id: "SPEC-065"
status: "released"
author: "paulomcnally"
created: "2026-09-04"
updated: "2026-09-04"
github_issue: 66
---

# Títulos dinámicos en el header según la página cargada

**ID**: SPEC-065  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-09-04  
**Actualizado**: 2026-09-04

---

## 1. Resumen Ejecutivo

El header de la app (sticky top bar, visible especialmente en móvil) muestra siempre el título de la sección basado en el **primer segmento** de la URL: `t(`${activeBase}.title`)` en `frontend/src/components/DashboardLayout.tsx:50`. Esto hace que al navegar a una vista de detalle el título quede "congelado" en el nombre genérico de la sección. Ejemplos reportados por el usuario: en `/services/bills/:serviceId` el header sigue mostrando "Servicios" en lugar del nombre del servicio; en `/deudas/:id` sigue mostrando "Deudas" en lugar de la descripción de la deuda; y en las pestañas de Deudas (`?tab=`) el título no refleja el tab seleccionado.

Además, hay módulos que hoy **muestran un título incorrecto o roto**: `/institutions`, `/autos` y sus sub-rutas caen en el fallback `t('app.title')` ("p40la") porque **no existen claves i18n** para esos módulos; y todas las sub-rutas de `/pension/*` muestran "Pensión alimenticia" en lugar del título propio (Hijos, Salarios, Categorías, etc.).

La solución es un mecanismo simple por el cual **cada página establece el título del header que corresponde**, usando la columna `name`/`title`/`description` del registro cargado cuando es una vista de detalle o edición, las claves i18n de creación para los formularios "new", y el label del tab cuando la página tiene tabs. Impacto mínimo en iHost: un store de Zustand en memoria (sin DB, sin API, sin dependencias nuevas).

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Crear un mecanismo para que cada página establezca el título del header (store Zustand + hook `usePageTitle`), con fallback al comportamiento actual basado en ruta cuando la página no lo define.
2. **REQ-002**: `DashboardLayout` debe renderizar el título dinámico de la página cuando exista, y el título por sección como fallback.
3. **REQ-003**: En vistas de detalle, el header debe mostrar el nombre/descripción del registro: `/services/bills/:serviceId` → `service.name`; `/deudas/:id` → `debt.description`; `/autos/:id` → `${brand} ${model}`.
4. **REQ-004**: En la página Deudas, el header debe reflejar el tab seleccionado: "Análisis" (default), "Calendario" o "Deudas".
5. **REQ-005**: En formularios de edición, el header debe mostrar el nombre del registro cargado (decisión del usuario 2026-09-04).
6. **REQ-006**: En formularios de creación, el header debe mostrar la clave i18n de creación (ej: "Nueva casa", "Nuevo servicio", "Nueva Deuda").
7. **REQ-007**: Agregar las claves i18n faltantes para que ningún título caiga al fallback "p40la": secciones `autos.*`, `institutions.*`, `notifications.title`, `bills.edit` (y sus equivalentes en EN). Editar **ambas** copias `frontend/src/i18n/{es,en}.json` y `frontend/public/i18n/{es,en}.json`, correr `npm run build` y verificar que el runtime sirve las claves nuevas.
8. **REQ-008**: En `/pension/registros`, el header debe mostrar el período seleccionado (ej: "Septiembre 2026") y actualizarse al navegar de mes (decisión del usuario 2026-09-04).
9. **REQ-009**: En `/bills/new` y `/bills/edit/:id`, el header debe mostrar "Nueva factura" / "Editar factura" (decisión del usuario 2026-09-04).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-010**: En páginas secundarias de Settings (`/settings/language`, `/settings/currency`, `/settings/currency/:id`), mostrar el título de la subsección ("Idioma", "Monedas", nombre/código de la moneda en edición) en lugar de "Configuración".
2. **REQ-011**: En sub-rutas de Pensión, mostrar el título propio de cada sección (Hijos, Categorías, Salarios, Notificaciones) en lugar de "Pensión alimenticia".

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-012**: Evaluar si el título del header debe truncarse con `truncate` cuando el nombre del registro es largo (consistencia con la UI existente).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: El store es solo memoria, sin efectos en tiempo de respuesta ni carga de DB.
- **Seguridad**: Sin cambios de autenticación; los títulos provienen de datos ya cargados y autorizados por las APIs existentes.
- **Almacenamiento**: Sin cambios de esquema ni datos persistentes.
- **Disponibilidad**: Sin impacto; es cambio puramente de UI.
- **iHost**: Cero dependencias nuevas, cero consultas extra a SQLite. El store Zustand pesa ~1 KB. Cumple la restricción de mínimo consumo de memoria.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se auditó el ruteo completo en `frontend/src/App.tsx` y el mecanismo actual del header en `frontend/src/components/DashboardLayout.tsx:21` (`activeBase = location.pathname.split('/')[1]`) y línea 50 (`t(`${activeBase}.title`, t('app.title'))`). Se revisaron todas las páginas en `frontend/src/pages/` para determinar, por ruta, qué dato disponible usar como título. Se revisaron las claves i18n en `frontend/public/i18n/es.json` (la fuente de verdad; existe una copia idéntica en `frontend/src/i18n/` importada por el bundle, ambas deben actualizarse).

Hallazgos clave:
- No existen secciones i18n `autos.*` ni `institutions.*`; `notifications` no tiene `title`. Por eso `/autos`, `/institutions` y `/pension/notificaciones` caen al fallback "p40la".
- `/pension/*` deriva `activeBase = 'pension'` y por eso todas las sub-rutas muestran "Pensión alimenticia".
- Las páginas de detalle (`BillsPage`, `DebtBillsPage`, `AutoShowPage`) ya cargan el registro completo (con `name`/`description`/`brand`+`model`) para renderizarlo, así que el título no requiere llamadas extra.
- `DeudasPage` ya conoce el tab activo desde `?tab=` (default `analisis`).
- `RegistrosPage` ya calcula `periodLabel` para su selector de mes.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **Store Zustand + hook `usePageTitle`** | Simple, encaja con el patrón existente (ya se usa Zustand), títulos dinámicos fáciles, se limpia solo al desmontar | Un archivo nuevo + tocar cada página | ✅ Seleccionada |
| Tabla de configuración de rutas en `DashboardLayout` | Centralizado | Las rutas con `:id` necesitan fetch de datos que ya hacen las páginas → duplicación y títulos lentos | ❌ Rechazada |
| Title en `document.title` vía react-helmet | Estándar para SEO | El header no es `document.title`; no resuelve el problema de UI | ❌ Rechazada |
| Prop `title` por ruta en el `<Route>` | Simple | No puede resolver nombres dinámicos asíncronos | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Store Zustand `pageTitleStore` + hook `usePageTitle`
- **Contexto**: Las páginas necesitan fijar un título que puede ser dinámico (nombre del registro, label del tab, período).
- **Decisión**: Crear `frontend/src/stores/pageTitleStore.ts` (estado `title: string | null` + `setTitle`) y un hook `usePageTitle(title: string | null)` que setea el título al montar/actualizar y lo limpia (`setTitle(null)`) al desmontar. `DashboardLayout` lee el store y usa `title ?? t(`${activeBase}.title`, t('app.title'))`.
- **Consecuencias**: Las páginas controlan su propio título de forma declarativa; al desmontar se restaura el fallback por sección. Costo mínimo en memoria.

**ADR-002**: El título del header usa el nombre del registro en vistas de detalle y edición
- **Contexto**: El usuario confirmó que en edición quiere el nombre del registro, y en detalle el nombre del registro.
- **Decisión**: Fuente de título por entidad: `Home.name`, `Service.name`, `Debt.description`, `Institution.name`, `Auto` → `${brand} ${model}`, `Child` → `${first_name} ${last_name}`, `Salary.employer`, `PensionCategory.name`, `Notification.name`, `Currency.name`.
- **Consecuencias**: Títulos informativos y consistentes; en formularios "new" se usa la clave i18n de creación porque aún no hay registro.

**ADR-003**: i18n en ambas copias (src y public) + build
- **Contexto**: El bundle importa `frontend/src/i18n/*.json` y el runtime fetchea `frontend/public/i18n/*.json`; el build de Vite regenera `public/` desde `frontend/public/`.
- **Decisión**: Editar las claves nuevas en las dos copias idénticas (`frontend/src/i18n/{es,en}.json` y `frontend/public/i18n/{es,en}.json`), correr `npm run build` en `frontend/` y verificar con `curl http://localhost:8088/i18n/es.json`.
- **Consecuencias**: Se evita el fallo histórico de SPEC-032/033 (claves que desaparecen en el build).

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Página (ej: BillsPage)]
        │  llama usePageTitle(service.name)
        ▼
[pageTitleStore (Zustand, memoria)]
        │  title
        ▼
[DashboardLayout] ──▶ <h1>{title ?? t(`${activeBase}.title`, app.title)}</h1>
```

### 4.2 Componentes

#### 4.2.1 `frontend/src/stores/pageTitleStore.ts` (nuevo)
- **Responsabilidad**: Mantener el título dinámico del header.
- **Interfaz**: `usePageTitleStore((s) => s.title)` y `usePageTitleStore((s) => s.setTitle)`.
- **Dependencias**: `zustand`.
- **Ubicación**: `frontend/src/stores/pageTitleStore.ts`.

```ts
import { create } from 'zustand'

interface PageTitleState {
  title: string | null
  setTitle: (title: string | null) => void
}

export const usePageTitleStore = create<PageTitleState>((set) => ({
  title: null,
  setTitle: (title) => set({ title }),
}))
```

#### 4.2.2 `frontend/src/hooks/usePageTitle.ts` (nuevo)
- **Responsabilidad**: Hook que fija el título y lo limpia al desmontar.
- **Interfaz**: `usePageTitle(title: string | null)`.
- **Dependencias**: `react`, `pageTitleStore`.

```ts
import { useEffect } from 'react'
import { usePageTitleStore } from '../stores/pageTitleStore'

export function usePageTitle(title: string | null) {
  const setTitle = usePageTitleStore((s) => s.setTitle)
  useEffect(() => {
    setTitle(title)
    return () => setTitle(null)
  }, [title, setTitle])
}
```

#### 4.2.3 `frontend/src/components/DashboardLayout.tsx` (modificar)
- **Responsabilidad**: Leer el título dinámico y usarlo con fallback.
- **Cambio** en la línea 50:

```tsx
const pageTitle = usePageTitleStore((s) => s.title)
// ...
<h1 className="text-base sm:text-lg font-semibold">{pageTitle ?? t(`${activeBase}.title`, t('app.title'))}</h1>
```

#### 4.2.4 Páginas (modificar todas en `frontend/src/pages/`)
- **Responsabilidad**: Llamar `usePageTitle(...)` con el título correcto según el estado (lista, detalle, tab, form new/edit).
- La llamada va **al inicio** del componente (antes de los early returns de loading/empty), respetando las reglas de hooks.

### 4.3 Matriz de títulos del header por página (validada con el usuario)

| Módulo | Ruta | Estado | Título del header | Fuente |
|--------|------|--------|-------------------|--------|
| Casa | `/home` | Lista | Casa | `home.title` (sin cambio) |
| Casa | `/home/new` | Form new | Nueva casa | `home.create` |
| Casa | `/home/edit/:id` | Form edit | Nombre de la casa | `home.name` |
| Servicios | `/services` | Lista | Servicios | `services.title` (sin cambio) |
| Servicios | `/services/new` | Form new | Nuevo servicio | `services.create` |
| Servicios | `/services/edit/:id` | Form edit | Nombre del servicio | `service.name` |
| Servicios | `/services/bills/:serviceId` | Detalle (facturas) | Nombre del servicio | `service.name` |
| Facturas | `/bills/new` | Form new | Nueva factura | `bills.create` |
| Facturas | `/bills/edit/:id` | Form edit | Editar factura | `bills.edit` (nueva clave) |
| Deudas | `/deudas?tab=analisis` | Tab Análisis | Análisis | `deudas.tab_analysis` |
| Deudas | `/deudas?tab=calendario` | Tab Calendario | Calendario | `deudas.tab_calendar` |
| Deudas | `/deudas?tab=deudas` | Tab Deudas | Deudas | `deudas.tab_debts` |
| Deudas | `/deudas/new` | Form new | Nueva Deuda | `deudas.create` |
| Deudas | `/deudas/edit/:id` | Form edit | Descripción de la deuda | `debt.description` |
| Deudas | `/deudas/:id` | Detalle (cuotas) | Descripción de la deuda | `debt.description` |
| Instituciones | `/institutions` | Lista | Instituciones | `institutions.title` (nueva clave) |
| Instituciones | `/institutions/new` | Form new | Nueva institución | `institutions.create` (nueva clave) |
| Instituciones | `/institutions/edit/:id` | Form edit | Nombre de la institución | `institution.name` |
| Autos | `/autos` | Lista | Autos | `autos.title` (nueva clave) |
| Autos | `/autos/new` | Form new | Nuevo auto | `autos.create` (nueva clave) |
| Autos | `/autos/edit/:id` | Form edit | Marca + Modelo | `auto.brand` + `auto.model` |
| Autos | `/autos/:id` | Detalle | Marca + Modelo | `auto.brand` + `auto.model` |
| Pensión | `/pension/hijos` | Lista | Hijos | `hijos.title` |
| Pensión | `/pension/hijos/new` | Form new | Nuevo hijo | `hijos.create` |
| Pensión | `/pension/hijos/edit/:id` | Form edit | Nombre completo | `child.first_name` + `child.last_name` |
| Pensión | `/pension/categorias` | Lista | Categorías | `categorias.title` |
| Pensión | `/pension/categorias/new` | Form new | Nueva categoría | `categorias.create` |
| Pensión | `/pension/categorias/edit/:id` | Form edit | Nombre | `category.name` |
| Pensión | `/pension/salarios` | Lista | Salarios | `salaries.title` |
| Pensión | `/pension/salarios/new` | Form new | Nuevo salario | `salaries.create` |
| Pensión | `/pension/salarios/edit/:id` | Form edit | Empleador | `salary.employer` |
| Pensión | `/pension/registros` | Lista con período | Período seleccionado (ej: "Septiembre 2026") | `periodLabel` |
| Pensión | `/pension/notificaciones` | Lista | Notificaciones | `notifications.title` (nueva clave) |
| Pensión | `/pension/notificaciones/new` | Form new | Nueva notificación | `notifications.create` |
| Pensión | `/pension/notificaciones/edit/:id` | Form edit | Nombre | `notification.name` |
| Configuración | `/settings` | Lista | Configuración | `settings.title` (sin cambio) |
| Configuración | `/settings/language` | Subsección | Idioma | `settings.language.title` |
| Configuración | `/settings/currency` | Form new | Monedas | `settings.currencies.title` |
| Configuración | `/settings/currency/:id` | Form edit | Nombre de la moneda | `currency.name` |

### 4.4 APIs / Contratos

Sin cambios de API. Toda la información de títulos ya está disponible en las respuestas existentes:
- `GET /api/services/:id` → `{ name }`
- `GET /api/debts/:id` → `{ description }`
- `GET /api/autos/:id` → `{ brand, model }`
- etc.

### 4.5 Dependencias

- **Internas**: `DashboardLayout`, las páginas de `frontend/src/pages/`, `frontend/src/stores/`, `frontend/src/hooks/` (nuevo), i18n `es/en.json` (src + public).
- **Externas**: Ninguna. Solo `zustand` (ya incluido).

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Al navegar a `/services/bills/:serviceId`, el header muestra el nombre del servicio (ej: "Internet Claro"), no "Servicios".
- [x] CA-002: Al navegar a `/deudas/:id`, el header muestra la descripción de la deuda (ej: "Préstamo Banco LAFISE"), no "Deudas".
- [x] CA-003: En `/deudas`, el header refleja el tab: "Análisis", "Calendario" o "Deudas", y cambia al cambiar de tab.
- [x] CA-004: En `/autos/:id` el header muestra "Marca Modelo"; en `/autos` muestra "Autos" (ya no "p40la").
- [x] CA-005: En `/institutions` y sus formularios, el header muestra títulos correctos (ya no "p40la").
- [x] CA-006: En sub-rutas de Pensión, el header muestra el título propio de cada sección (Hijos, Categorías, Salarios, Notificaciones), no "Pensión alimenticia".
- [x] CA-007: En `/pension/registros`, el header muestra el período seleccionado y se actualiza al navegar entre meses.
- [x] CA-008: En formularios de edición, el header muestra el nombre del registro cargado.
- [x] CA-009: En formularios de creación, el header muestra la clave i18n de creación (ej: "Nueva casa", "Nuevo servicio", "Nueva factura", "Editar factura").
- [x] CA-010: En `/settings/language` y `/settings/currency`, el header muestra "Idioma" y "Monedas" respectivamente (no "Configuración").
- [x] CA-011: Tras `npm run build`, `curl http://localhost:8088/i18n/es.json` sirve las claves nuevas (`autos.*`, `institutions.*`, `notifications.title`, `bills.edit`) y el header no cae al fallback "p40la" en ninguna ruta.
- [x] CA-012: Al desmontar una página con título dinámico (navegar a otra sección), el header vuelve al fallback por sección sin títulos "pegados".
- [x] CA-DARK: No aplica a formularios nuevos (no se crean inputs), pero el header usa tokens del tema ya existentes (`text-text`, `bg-card`); verificar legibilidad en darkmode.

### 5.2 No funcionales

- [x] CA-NF-001: La solución agrega ~1 KB en memoria (store Zustand), sin queries extra a SQLite ni dependencias nuevas.
- [x] CA-NF-002: `npm run build` (frontend) y `go build` pasan sin errores; `npm run lint`/`tsc` sin warnings nuevos.

### 5.3 Testing

- **Unit tests**: Si existe infra de test frontend, verificar `usePageTitle` setea/limpia el título del store.
- **Integration tests**: Recorrido manual de las rutas de la matriz 4.3 verificando el texto del header.
- **E2E tests**: Flujo móvil: Servicios → entrar a un servicio → ver nombre en header; Deudas → cambiar tabs → ver label del tab; Deudas → entrar a una deuda → ver descripción.
- **Carga/Performance**: N/A (cambio de UI sin impacto medible).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Crear `pageTitleStore` + hook `usePageTitle` + modificar `DashboardLayout` | 0.5 días | Ninguna |
| 2 | Agregar claves i18n nuevas en `es/en.json` (src + public) y correr build | 0.5 días | Fase 1 |
| 3 | Aplicar `usePageTitle` en páginas de detalle y tabs (Bills, DebtBills, AutoShow, Deudas, Registros) | 1 día | Fase 1 |
| 4 | Aplicar `usePageTitle` en listas y formularios new/edit restantes (Casa, Servicios, Facturas, Instituciones, Autos, Pensión, Settings) | 1 día | Fase 2, 3 |
| 5 | Pruebas manuales locales en móvil y desktop, darkmode, build y release | 0.5 días | Fases 1-4 |

### 6.2 Milestones

1. **MVP**: Fases 1-3 — casos reportados por el usuario (Servicios, Deudas, tabs) + Autos/Instituciones sin fallback "p40la".
2. **V1.0**: Fases 4-5 — matriz completa de la sección 4.3.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Título flash: al cargar detalle, el header muestra brevemente el fallback hasta que llega el nombre | Media | Bajo | El hook acepta `null` mientras carga; el flash es de ms. Aceptable |
| Editar solo una copia de i18n (src o public) y romper en build | Media | Medio | Regla de AGENTS.md: editar ambas copias y verificar con curl tras build (CA-011) |
| Título largo del registro desborda el header en móvil | Baja | Bajo | REQ-012: aplicar `truncate` al `h1` si se confirma |
| Hook llamado después de early return violando reglas de React | Baja | Medio | Colocar `usePageTitle` siempre al inicio del componente, antes de los early returns |
| Colisión de ID con otra sesión paralela (ocurrió: SPEC-063 ya existía) | Alta | Medio | Verificar `ls docs/specs/` antes de fijar ID y usar `git worktree list` |

## 8. Notas y Referencias

- Mecanismo actual del header: `frontend/src/components/DashboardLayout.tsx:50`.
- Regla de i18n (fuente de verdad `frontend/public/i18n/` y build): AGENTS.md sección "Reglas críticas".
- Precedente de fallo i18n en build: SPEC-032/033.
- Stores existentes (patrón Zustand): `frontend/src/stores/i18nStore.ts`.
- SPEC-063 (otra sesión, "Flecha atrás en el header") es un feature relacionado pero independiente.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-04 | paulomcnally | Creación inicial de la especificación (ID renumerado de 063 a 065 por colisión con otra sesión) |
| 2026-09-04 | paulomcnally | Estado a in_progress. Implementación: store `pageTitleStore` + hook `usePageTitle`, modificación de `DashboardLayout`, claves i18n nuevas (`autos.*`, `institutions.*`, `notifications.title`) en es/en, y `usePageTitle` aplicado en todas las páginas (detalles, tabs, listas y forms new/edit). Build OK. Server local en :8090 para evaluación. |
| 2026-09-04 | paulomcnally | Release: criterios de aceptación en pass, estado a `released`. Commit de implementación `2bd5a32` mergeado a `main` (issue #66 cerrado). |