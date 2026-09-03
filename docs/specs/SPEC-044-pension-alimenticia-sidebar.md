---
title: "Menú Pensión Alimenticia en el Sidebar con submenús y páginas en blanco"
id: "SPEC-044"
status: "released"
author: "p40la-ihost-team"
created: "2026-09-02"
updated: "2026-09-02"
github_issue: 44
---

# Menú Pensión Alimenticia en el Sidebar con submenús y páginas en blanco

**ID**: SPEC-044  
**Estado**: released  
**Autor**: p40la-ihost-team  
**Creado**: 2026-09-02  
**Actualizado**: 2026-09-02

---

## 1. Resumen Ejecutivo

Se necesita un nuevo módulo de **Pensión Alimenticia** en la aplicación, accesible desde el sidebar principal. Este módulo será la base para gestionar la pensión alimenticia de los hijos: registro de hijos, categorías de gastos, salarios involucrados, registros mensuales de pagos y notificaciones asociadas.

En esta primera iteración el requerimiento es estructural: agregar la entrada de menú "Pensión alimenticia" en el sidebar, con un submenú colapsable que despliegue cinco secciones (**Hijos**, **Categorías**, **Salarios**, **Registros mensuales**, **Notificaciones**), cada una navegando a una página en blanco (placeholder). No se requiere lógica de negocio, backend ni base de datos en esta fase.

La decisión de crear las páginas como placeholders permite validar la navegación y el patrón de UI antes de invertir en modelos de datos y APIs. Es coherente con la regla del proyecto de seguir el patrón de UI existente (sidebar + cards + i18n) y respeta las restricciones de iHost: bajo consumo de memoria, sin nuevas dependencias y sin Node.js en runtime.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Agregar la entrada de menú "Pensión alimenticia" en el sidebar, debajo de las entradas existentes (Casa, Servicios, Instituciones, Autos).
2. **REQ-002**: El menú debe ser expandible/colapsable y desplegar los siguientes submenús:
   - **Hijos** (`/pension/hijos`)
   - **Categorías** (`/pension/categorias`)
   - **Salarios** (`/pension/salarios`)
   - **Registros mensuales** (`/pension/registros`)
   - **Notificaciones** (`/pension/notificaciones`)
3. **REQ-003**: Cada submenú debe navegar a una página en blanco (placeholder) con título de página y EmptyCard indicando que no hay registros (siguiendo el patrón de UI existente, ver HomesPage.tsx).
4. **REQ-004**: Las rutas de las páginas deben estar registradas en `App.tsx` dentro del `DashboardLayout`.
5. **REQ-005**: Las etiquetas del menú y de las páginas deben ser traducibles vía i18n (es/en) en `frontend/public/i18n/` (fuente de verdad) y `frontend/src/i18n/`.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-006**: El submenú debe resaltar el item activo según la ruta actual (comportamiento consistente con el sidebar existente).
2. **REQ-007**: El menú debe ser responsive: en móvil se abre/cierra con el drawer existente, en desktop es fijo.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-008**: Definir iconos específicos para cada submenú (hijos, categorías, salarios, registros, notificaciones) dentro del catálogo existente de iconos.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: La adición del menú y las páginas placeholder no debe incrementar significativamente el tamaño del bundle ni el tiempo de carga. Sin lógica pesada.
- **Seguridad**: Las rutas nuevas deben quedar bajo el `AuthGuard` existente (autenticación requerida).
- **Almacenamiento**: Sin cambios en la base de datos. No se agregan tablas ni migraciones en esta fase.
- **Disponibilidad**: Sin impacto en health checks ni schedulers existentes.
- **iHost**: Sin nuevas dependencias. Solo archivos estáticos React build pre-generado (Vite) servidos por el backend Go. Sin Node.js en runtime.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se revisó la estructura actual del frontend React:

- `frontend/src/components/Sidebar.tsx`: define `items` como array plano `{ key, icon, label }` y renderiza botones con `handleNavigate`. No soporta submenús anidados actualmente.
- `frontend/src/App.tsx`: registra las rutas dentro de `<DashboardLayout>` envueltas por `AuthGuard`.
- `frontend/src/components/DashboardLayout.tsx`: calcula `activeBase` desde `location.pathname.split('/')[1]` para el título y resaltado del sidebar.
- `frontend/src/components/HomesPage.tsx`: referencia del patrón de listado (cards + EmptyCard).
- i18n: las claves de menú están en `frontend/src/i18n/es.json` bajo `menu.*` y la fuente de verdad de producción es `frontend/public/i18n/`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Sidebar plano con items separados | Implementación trivial | No cumple el requerimiento de submenús | ❌ Rechazada |
| Sidebar con submenú colapsable (grupo con hijos) | Cumple requerimiento, patrón común en UIs iOS | Requiere modificar `Sidebar.tsx` para render anidado y estado de expansión | ✅ Seleccionada |
| Librería de navegación externa | Funcionalidades avanzadas | Dependencia extra, viola restricción de mínimas dependencias en iHost | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Submenú colapsable nativo sin librerías externas
- **Contexto**: El sidebar actual es plano y simple; agregar submenús requiere expandir el componente. Se evalúa si usar una librería de menús.
- **Decisión**: Implementar el submenú con React puro y estado local (`useState` para la expansión) dentro de `Sidebar.tsx`, sin dependencias nuevas.
- **Consecuencias**: Código adicional pequeño y mantenible; cero impacto en el bundle del iHost; comportamiento consistente con el resto de la UI.

**ADR-002**: Páginas placeholder sin backend en esta fase
- **Contexto**: El módulo de pensión alimenticia tendrá lógica de negocio a futuro, pero el requerimiento actual es solo estructura de navegación.
- **Decisión**: Crear páginas en blanco (EmptyCard) sin modelos ni APIs, y documentar en esta spec que las fases futuras agregarán dominio.
- **Consecuencias**: Permite validar UX tempranamente; evita crear tablas SQLite que podrían cambiar según feedback del usuario.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Sidebar.tsx] --(navegación)--> [App.tsx rutas]
      |                                |
      |  submenú Pensión alimenticia   v
      +------------------------> [DashboardLayout (AuthGuard)]
                                        |
                                        v
                        [PensionPage placeholder] (Hijos/Categorías/Salarios/Registros/Notificaciones)
```

### 4.2 Componentes

#### 4.2.1 Sidebar (`frontend/src/components/Sidebar.tsx`)
- **Responsabilidad**: Renderizar navegación principal. Se modifica para soportar un grupo de submenús colapsable para "Pensión alimenticia".
- **Interfaz**: Mantener las props `activeBase`, `isOpen`, `onClose`. Internamente, el grupo define `children: [{ key, icon, label }]`.
- **Dependencias**: `useNavigate`, `useI18nStore`, componente `Icon`.
- **Ubicación**: `frontend/src/components/Sidebar.tsx`.

#### 4.2.2 Página placeholder genérica (`frontend/src/pages/PensionPage.tsx`)
- **Responsabilidad**: Renderizar el placeholder de una sección del módulo, mostrando título y EmptyCard.
- **Interfaz**: Recibe la sección a renderizar (vía props o vía ruta) y muestra `t('pension.<seccion>.title')` y `t('pension.<seccion>.empty')`.
- **Dependencias**: `useI18nStore`.
- **Ubicación**: `frontend/src/pages/PensionPage.tsx`.

#### 4.2.3 App (`frontend/src/App.tsx`)
- **Responsabilidad**: Registrar las rutas nuevas del módulo dentro del `DashboardLayout`.
- **Interfaz**: Rutas `/pension/hijos`, `/pension/categorias`, `/pension/salarios`, `/pension/registros`, `/pension/notificaciones`.
- **Dependencias**: `PensionPage`.
- **Ubicación**: `frontend/src/App.tsx`.

### 4.3 Modelo de datos

Sin cambios. No se agregan tablas en esta fase.

### 4.4 APIs / Contratos

Sin cambios. No se agregan endpoints en esta fase.

### 4.5 Dependencias

- **Internas**: `Sidebar.tsx`, `App.tsx`, archivos i18n (`frontend/public/i18n/{es,en}.json` y `frontend/src/i18n/{es,en}.json`). Nueva página `PensionPage.tsx`.
- **Externas**: Ninguna.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: En el sidebar aparece la entrada "Pensión alimenticia" debajo de Autos.
- [x] CA-002: Al hacer clic en "Pensión alimenticia" se despliega/colapsa el submenú con las 5 secciones: Hijos, Categorías, Salarios, Registros mensuales, Notificaciones.
- [x] CA-003: Cada submenú navega a su ruta correspondiente (`/pension/hijos`, `/pension/categorias`, `/pension/salarios`, `/pension/registros`, `/pension/notificaciones`) y muestra una página en blanco con título y EmptyCard.
- [x] CA-004: Las rutas están protegidas por autenticación (requieren login).
- [x] CA-005: Las etiquetas se muestran traducidas en español e inglés según el idioma seleccionado.
- [x] CA-006: El item del submenú activo se resalta según la ruta actual.
- [x] CA-007: El comportamiento responsive se mantiene (drawer en móvil, sidebar fijo en desktop).

### 5.2 No funcionales

- [x] CA-NF-001: El build de Vite (`npm run build` en `frontend/`) compila sin errores y el bundle no crece significativamente.
- [x] CA-NF-002: No se agregan dependencias nuevas al proyecto.

### 5.3 Testing

- **Unit tests**: No aplica lógica de negocio nueva; validar compilación TypeScript (`tsc -b`).
- **Integration tests**: Navegación manual de cada submenú a su página.
- **E2E tests**: Flujo de login → abrir menú → navegar a cada sección → verificar EmptyCard.
- **Carga/Performance**: Sin métricas nuevas; verificar que la app levanta en local con `./start.sh`.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Modificar `Sidebar.tsx` para soportar grupo de submenús colapsable con "Pensión alimenticia" | 0.5 días | Ninguna |
| 2 | Crear `PensionPage.tsx` (placeholder con EmptyCard) y registrar las 5 rutas en `App.tsx` | 0.5 días | Fase 1 |
| 3 | Agregar claves i18n es/en (fuente en `frontend/public/i18n/` y espejo en `frontend/src/i18n/`) y build | 0.5 días | Fase 2 |
| 4 | Pruebas locales (`npm run build`, levantar server, navegación manual) | 0.5 días | Fase 3 |

### 6.2 Milestones

1. **MVP**: Menú desplegable con 5 submenús navegando a páginas placeholder en blanco con EmptyCard, con i18n es/en.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| El sidebar plano actual no soporta submenús y la modificación rompe navegación existente | Media | Alto | Cambios mínimos e incrementales; probar navegación existente después del cambio |
| Las claves i18n editadas en `frontend/src/i18n/` no se reflejan en producción | Media | Medio | Editar SIEMPRE `frontend/public/i18n/` (fuente de verdad) y correr `npm run build`; verificar con `curl` |
| `activeBase` usa solo el primer segmento de la ruta y no resalta submenús anidados | Alta | Bajo | Ajustar la lógica de resaltado para rutas `/pension/*` |

## 8. Notas y Referencias

- Patrón de UI existente: `frontend/src/pages/HomesPage.tsx` (cards + EmptyCard + CreateMenu).
- Sidebar actual: `frontend/src/components/Sidebar.tsx`.
- Reglas de i18n del proyecto: fuente de verdad en `frontend/public/i18n/` (ver AGENTS.md, sección "Reglas críticas").

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-02 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-09-02 | p40la-ihost-team | Estado cambiado de draft a pending_execution y luego a in_progress |
| 2026-09-02 | p40la-ihost-team | Implementación completa y validada por el usuario. Release a main: commit `7deb6c8` (merge feature/SPEC-044), versión 0.4.14. Estado → released |