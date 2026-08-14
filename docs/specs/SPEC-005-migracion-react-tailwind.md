---
title: "Migración del Frontend a React + Tailwind CSS"
id: "SPEC-005"
status: "in_progress"
author: "p40la-ihost-team"
created: "2026-08-13"
updated: "2026-08-13"
---

# Migración del Frontend a React + Tailwind CSS

**ID**: SPEC-005  
**Estado**: draft  
**Autor**: p40la-ihost-team  
**Creado**: 2026-08-13  
**Actualizado**: 2026-08-13

---

## 1. Resumen Ejecutivo

El frontend actual del proyecto p40la-ihost utiliza HTML, CSS y JavaScript vanilla servidos directamente por el backend en Go. Esta aproximación ha demostrado ser difícil de mantener, con código duplicado, gestión de estado manual compleja y CSS sin sistema de diseño consistente.

Esta spec propone migrar el frontend a **React** como biblioteca de UI y **Tailwind CSS** como framework de estilos, con un proceso de build que genera archivos estáticos optimizados que el backend de Go sigue sirviendo sin cambios en su arquitectura HTTP.

El desafío principal es mantener el proyecto viable en un iHost con recursos limitados. La solución: el build de React+Tailwind se realiza **fuera del iHost** (en CI/CD o máquina de desarrollo), y el iHost solo sirve los archivos estáticos resultantes (HTML + JS bundle + CSS), sin necesidad de Node.js en runtime.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Setup de proyecto React con Vite como bundler (más ligero que Create React App)
2. **REQ-002**: Configuración de Tailwind CSS con purge/optimización para bundle mínimo
3. **REQ-003**: Migración completa de `login.html` a componente React con autenticación funcional
4. **REQ-004**: Migración completa de `dashboard.html` a componente React con sidebar, header, i18n, settings, home y módulo de servicios
5. **REQ-005**: Migración completa de `setup.html` a componente React
6. **REQ-006**: Sistema de i18n funcional (es/en) equivalente al actual
7. **REQ-007**: El build debe generar archivos estáticos en `public/` que el servidor Go sirva sin modificaciones
8. **REQ-008**: Actualización de `docs/project-rules.md` para autorizar React + Tailwind

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-009**: Router cliente-side para navegación sin recarga (React Router o solución ligera)
2. **REQ-010**: Gestión de estado centralizada (Context API o Zustand — preferir Zustand por simplicidad)
3. **REQ-011**: Componentes reutilizables para elementos comunes (botones, inputs, cards, modals)
4. **REQ-012**: El bundle JS final no debe superar 150KB gzipped
5. **REQ-013**: El bundle CSS final no debe superar 30KB gzipped

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-014**: Hot Module Replacement (HMR) para desarrollo local
2. **REQ-015**: TypeScript opcional para componentes críticos
3. **REQ-016**: Animaciones de transición entre vistas

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: First Contentful Paint < 1.5s en iHost, bundle JS < 150KB gzipped
- **Seguridad**: Mantener mismo modelo de autenticación (JWT/session), sin exponer tokens en URLs
- **Almacenamiento**: Los archivos de build estático no deben superar 500KB totales
- **iHost**: Cero dependencias de Node.js en runtime. Solo archivos estáticos pre-build
- **Build**: El proceso de build debe poder ejecutarse en CI/CD o máquina de desarrollo

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Frontend actual**: 3 archivos HTML + 4 archivos JS + 2 archivos CSS con lógica acoplada
- **SPEC-004** definió un dashboard complejo con sidebar, i18n, settings iOS-style, iconos, y módulo de facturas — demasiado para JS vanilla mantenible
- **iHost constraints**: RAM limitada, sin Node.js en runtime, SQLite como única DB
- **React**: Biblioteca de UI de Facebook, ~42KB gzipped (core + react-dom)
- **Tailwind CSS**: Framework utility-first, genera solo las clases usadas (purge)
- **Vite**: Bundler moderno, ~10x más rápido que Webpack, output estático limpio

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **React + Vite + Tailwind** | Ecosistema enorme, componentes reutilizables, DX excelente, build estático | Requiere Node.js para build (no en runtime) | ✅ Seleccionada |
| **React + Create React App + Tailwind** | Setup oficial | Bundle más pesado, build lento, menos configurable | ❌ Rechazada |
| **Preact + Tailwind** | ~3KB vs 42KB de React | Ecosistema menor, posibles incompatibilidades | ⚠️ Alternativa si el bundle es muy grande |
| **HTMX + Alpine.js** | Muy ligero, sin build step | No resuelve el problema de complejidad de estado | ❌ Rechazada |
| **Seguir con vanilla JS** | Cero dependencias | Ya demostrado como insostenible | ❌ Rechazada |
| **Vue + Vite** | Similar a React, buena DX | Ecosistema ligeramente menor para i18n/state | ⚠️ Alternativa viable |
| **Svelte + Tailwind** | Bundle más pequeño, sin runtime | Ecosistema más pequeño, menos maduro | ⚠️ Alternativa viable |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Build fuera del iHost, serve estático dentro
- **Contexto**: El iHost no tiene Node.js ni debe tenerlo. React necesita un build step.
- **Decisión**: El build se ejecuta en CI/CD o máquina de desarrollo. Los archivos resultantes (HTML, JS bundle, CSS) se copian a `public/` y el servidor Go los sirve como archivos estáticos.
- **Consecuencias**: El deploy requiere un paso de build previo. No se puede editar el frontend directamente en el iHost.

**ADR-002**: Vite como bundler en lugar de Webpack/ CRA
- **Contexto**: Necesitamos un bundler rápido y con output estático limpio.
- **Decisión**: Vite con `@vitejs/plugin-react` y `rollup` para producción.
- **Consecuencias**: Build más rápido, configuración más simple, output optimizado.

**ADR-003**: Tailwind CSS con purge/optimización
- **Contexto**: Tailwind genera miles de utilidades. Necesitamos solo las usadas.
- **Decisión**: Configurar `content` en tailwind.config.js para escanear solo archivos fuente. Output CSS mínimo.
- **Consecuencias**: CSS final de ~10-30KB en lugar de ~3MB.

**ADR-004**: Zustand para state management en lugar de Redux/Context API puro
- **Contexto**: Necesitamos gestión de estado para auth, i18n, settings, datos de servicios.
- **Decisión**: Zustand es ~1KB, API simple, sin boilerplate. Mejor que Context API para estado frecuente.
- **Consecuencias**: Dependencia mínima, código más limpio.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
┌─────────────────────────────────────────────────┐
│              CI/CD o Dev Machine                │
│                                                 │
│  src/frontend/ (React + TSX + Tailwind)         │
│       │                                         │
│       ▼                                         │
│  Vite build (npm run build)                     │
│       │                                         │
│       ▼                                         │
│  dist/ → public/ (HTML + JS bundle + CSS)       │
└──────────────────────┬──────────────────────────┘
                       │ (archivos estáticos)
                       ▼
┌─────────────────────────────────────────────────┐
│                    iHost                        │
│                                                 │
│  Go Server (net/http)                           │
│       │                                         │
│       ├── Serve static: public/*.html            │
│       ├── Serve static: public/assets/*.js       │
│       ├── Serve static: public/assets/*.css      │
│       └── API endpoints: /api/*                  │
│              │                                  │
│              ▼                                  │
│         SQLite (modernc.org/sqlite)             │
└─────────────────────────────────────────────────┘
```

### 4.2 Componentes

#### 4.2.1 Estructura del frontend

```
src/frontend/
├── index.html                    # Entry point (Vite)
├── package.json
├── vite.config.js
├── tailwind.config.js
├── postcss.config.js
├── tsconfig.json                 # Opcional pero recomendado
├── src/
│   ├── main.tsx                  # Entry point React
│   ├── App.tsx                   # Root component con router
│   ├── styles/
│   │   └── index.css             # Tailwind imports
│   ├── components/
│   │   ├── ui/                   # Componentes reutilizables
│   │   │   ├── Button.tsx
│   │   │   ├── Input.tsx
│   │   │   ├── Card.tsx
│   │   │   ├── Modal.tsx
│   │   │   └── Sidebar.tsx
│   │   ├── auth/
│   │   │   ├── LoginForm.tsx
│   │   │   └── SetupForm.tsx
│   │   └── dashboard/
│   │       ├── DashboardLayout.tsx
│   │       ├── Header.tsx
│   │       ├── HomeView.tsx
│   │       ├── SettingsView.tsx
│   │       └── ServicesView.tsx
│   ├── hooks/
│   │   ├── useAuth.ts
│   │   └── useI18n.ts
│   ├── stores/
│   │   ├── authStore.ts          # Zustand: auth state
│   │   ├── settingsStore.ts      # Zustand: settings state
│   │   └── i18nStore.ts          # Zustand: i18n state
│   ├── i18n/
│   │   ├── es.json
│   │   ├── en.json
│   │   └── index.ts              # i18n helper
│   ├── api/
│   │   └── client.ts             # Fetch wrapper para /api/*
│   └── types/
│       └── index.ts              # TypeScript types
└── public/                       # Output del build → mapeado al public/ del Go server
```

#### 4.2.2 Integración con Go

El servidor Go se modificó en `internal/api/routes.go`:
- Se eliminaron los handlers individuales para `/setup`, `/login`, `/dashboard`
- Se reemplazaron con un `spaHandler` que sirve `index.html` como fallback SPA
- Se agregó `http.FileServer` para `/assets/` (JS/CSS bundles)
- Se mantiene `http.StripPrefix` para `/i18n/` (archivos JSON)
- Todas las APIs REST permanecen sin cambios

### 4.2.3 Estructura real implementada

```
frontend/
├── index.html
├── package.json
├── vite.config.ts
├── tailwind.config.js
├── postcss.config.js
├── tsconfig.json
├── tsconfig.node.json
└── src/
    ├── main.tsx
    ├── App.tsx
    ├── index.css
    ├── types/
    │   └── index.ts
    ├── api/
    │   └── index.ts
    ├── components/
    │   ├── Icons.tsx
    │   ├── Sidebar.tsx
    │   ├── DashboardLayout.tsx
    │   └── DeleteModal.tsx
    ├── stores/
    │   ├── authStore.ts
    │   ├── i18nStore.ts
    │   └── appStore.ts
    └── pages/
        ├── LoginPage.tsx
        ├── SetupPage.tsx
        ├── HomesPage.tsx
        ├── HomeFormPage.tsx
        ├── ServicesPage.tsx
        ├── ServiceFormPage.tsx
        ├── BillsPage.tsx
        ├── BillFormPage.tsx
        ├── SettingsPage.tsx
        ├── LanguagePage.tsx
        └── CurrencyFormPage.tsx
```

### 4.3 Modelo de datos

Sin cambios en el modelo de datos. El frontend consume las mismas APIs REST existentes.

### 4.4 APIs / Contratos

Sin cambios en las APIs existentes. El frontend React consume los mismos endpoints:

- `POST /api/login`
- `POST /api/setup`
- `GET /api/me`
- `GET /api/settings`
- `POST /api/settings/language`
- `GET /api/currencies`
- `POST /api/currencies`
- `PUT /api/currencies/{id}`
- `DELETE /api/currencies/{id}`
- `GET /api/homes`
- `GET /api/homes/{id}`
- `POST /api/homes`
- `PUT /api/homes/{id}`
- `DELETE /api/homes/{id}`
- `GET /api/services`
- `GET /api/services/{id}`
- `POST /api/services`
- `PUT /api/services/{id}`
- `DELETE /api/services/{id}`
- `GET /api/services/{service_id}/bills`
- `GET /api/bills/{id}`
- `POST /api/bills`
- `PUT /api/bills/{id}`
- `DELETE /api/bills/{id}`
- `GET /i18n/{lang}.json`

### 4.5 Dependencias

**Internas**:
- `internal/api/routes.go` — modificado para servir SPA en lugar de HTMLs individuales
- `public/` — reemplazado por output del build de Vite (index.html + assets/)
- `public/i18n/` — conservado de frontend vanilla para archivos JSON

**Externas (frontend)**:
| Paquete | Tamaño gzipped | Propósito |
|---------|---------------|-----------|
| react | ~42KB | Biblioteca de UI |
| react-dom | incluido en react | Renderizado DOM |
| zustand | ~1KB | State management |
| react-router-dom | ~10KB | Routing cliente |
| tailwindcss | 0KB (build-time) | CSS framework |

**Externas (build)**:
| Paquete | Propósito |
|---------|-----------|
| vite | Bundler |
| @vitejs/plugin-react | Plugin React para Vite |
| autoprefixer | PostCSS plugin |
| postcss | Procesador CSS |
| typescript | Type checking |

### 4.6 Archivos eliminados

- `public/login.html`
- `public/setup.html`
- `public/dashboard.html`
- `public/css/auth.css`
- `public/css/dashboard.css`
- `public/js/api.js`
- `public/js/app.js`
- `public/js/auth.js`
- `public/js/i18n.js`
- `public/js/icons.js`

### 4.7 Resultados del build

| Archivo | Tamaño | Gzip |
|---------|--------|------|
| index.html | 0.39 KB | 0.27 KB |
| index-*.js | 285.30 KB | 84.24 KB |
| index-*.css | 14.43 KB | 3.65 KB |
| **Total** | **300.12 KB** | **88.16 KB** |

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Dado un usuario en la página de login, cuando ingresa credenciales válidas, entonces es redirigido al dashboard sin recarga de página
- [x] CA-002: Dado un usuario autenticado, cuando navega entre vistas (home, settings, services), entonces la navegación es instantánea sin recarga
- [x] CA-003: Dado un usuario en el dashboard, cuando cambia el idioma, entonces toda la UI se actualiza al idioma seleccionado
- [x] CA-004: Dado un usuario en settings, cuando modifica una configuración, entonces el cambio se persiste y refleja inmediatamente
- [x] CA-005: Dado un usuario no autenticado, cuando intenta acceder al dashboard, entonces es redirigido al login
- [x] CA-006: El setup inicial funciona correctamente con el formulario de configuración
- [x] CA-007: El módulo de servicios muestra facturas y datos equivalentes al frontend actual

### 5.2 No funcionales

- [ ] CA-NF-001: El bundle JS total no supera 150KB gzipped
- [ ] CA-NF-002: El bundle CSS total no supera 30KB gzipped
- [ ] CA-NF-003: El tiempo de carga inicial (FCP) es menor a 1.5s en iHost
- [ ] CA-NF-004: No se requiere Node.js en runtime en el iHost
- [ ] CA-NF-005: El servidor Go sirve los archivos estáticos sin modificaciones en su lógica

### 5.3 Testing

- **Unit tests**: Componentes React críticos (LoginForm, auth hooks, stores)
- **Integration tests**: Flujo completo login → dashboard → navegación
- **E2E tests**: Escenarios de usuario con Playwright o similar
- **Performance**: Medir bundle size y FCP en entorno iHost

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Setup de proyecto React + Vite + Tailwind | 1 día | Ninguna |
| 2 | Migración de login y setup (auth flow) | 2 días | Fase 1 |
| 3 | Migración de dashboard layout (sidebar, header) | 2 días | Fase 2 |
| 4 | Migración de vistas (home, settings, services) | 3 días | Fase 3 |
| 5 | Migración de i18n y state management | 1 día | Fase 3 |
| 6 | Integración con Go server + optimización de build | 1 día | Fase 4, 5 |
| 7 | Testing, QA y ajuste de performance | 2 días | Fase 6 |

### 6.2 Milestones

1. **MVP**: Login + dashboard básico funcionando con React, served por Go
2. **V1.0**: Todas las vistas migradas, i18n funcional, bundle optimizado, tests pasando

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Bundle final excede límites de tamaño | Media | Alto | Usar code splitting, tree shaking, analizar con `rollup-plugin-visualizer` |
| Complejidad del build step para CI/CD | Baja | Medio | Documentar proceso de build, usar Docker multi-stage |
| Pérdida de funcionalidad durante migración | Media | Alto | Migrar vista por vista, mantener frontend actual como fallback |
| Performance peor que vanilla JS | Baja | Alto | Medir antes/después, optimizar con lazy loading y code splitting |
| Tailwind genera CSS muy grande | Baja | Medio | Configurar purge correctamente, usar `content` restrictivo |

## 8. Notas y Referencias

- Vite docs: https://vitejs.dev/
- Tailwind CSS docs: https://tailwindcss.com/docs
- Zustand docs: https://zustand-demo.pmnd.rs/
- React docs: https://react.dev/
- SPEC-004: Dashboard con Sidebar, Header, i18n, Settings — funcionalidad a migrar
- project-rules.md: Requiere actualización para autorizar React + Tailwind

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-13 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-08-13 | p40la-ihost-team | Estado cambiado a `in_progress`. Frontend React+Vite+Tailwind implementado completamente. Build exitoso: JS 84KB gzip, CSS 3.65KB gzip. Server corriendo en :8088. |
| 2026-08-13 | p40la-ihost-team | Fix: MIME type incorrecto en assets. Se cambió patrón de rutas Go de `GET /assets/` a `GET /assets/{file...}` con `http.StripPrefix` para que `http.FileServer` sirva JS/CSS con Content-Type correcto (`text/javascript`, `text/css`). |
| 2026-08-13 | p40la-ihost-team | Fix: Campo `setup` vs `setup_completed`. El API devuelve `{"setup_completed": true}` pero el frontend buscaba `setup`. Se corrigió en `authStore.ts` a `status?.setup_completed`. |
| 2026-08-13 | p40la-ihost-team | Fix: i18n no cargaba. Diccionario empezaba vacío y los componentes renderizaban antes del fetch async. Solución: se embebieron los JSON (`es.json`, `en.json`) directamente en el bundle via import en `i18nStore.ts`, y se copiaron a `public/i18n/` para el endpoint del server. |
| 2026-08-13 | p40la-ihost-team | UI: Botones de "Crear" reemplazados por menú de 3 puntos (`CreateMenu`). Aplica a HomesPage, ServicesPage, BillsPage. El menú despliega opciones de creación en dropdown. |
| 2026-08-13 | p40la-ihost-team | Docker: `Dockerfile` actualizado con multi-stage build. Stage 1: `node:20-alpine` hace `npm ci` + `npm run build` del frontend. Stage 2: `golang:1.23-alpine` copia el build de React a `public/` y compila el binario Go. Stage 3: `distroless` runtime con server + public + migrations. |
| 2026-08-13 | p40la-ihost-team | UI: Dropdowns nativos (`<select>`) reemplazados por componente `Select` custom con búsqueda. Aplica a filtro de casas en ServicesPage, y selects en ServiceFormPage (casa, moneda, frecuencia) y BillFormPage (mes, estado). |
| 2026-08-13 | p40la-ihost-team | UI: Menús de 3 puntos en cards no visibles. Se creó componente `CardMenu` reutilizable con estado React (no DOM manipulation). Aplica a HomesPage, ServicesPage, BillsPage. Z-index y posicionamiento corregidos. |
| 2026-08-13 | p40la-ihost-team | Fix: Dropdown de CardMenu en tabla de BillsPage cortado por `overflow-hidden` del contenedor. Se removió `overflow-hidden` del wrapper de la tabla para permitir que el menú se renderice fuera de los límites. |
| 2026-08-13 | p40la-ihost-team | Fix: Refrescar la página redirigía al login porque el estado `isAuthenticated` se perdía. Se agregó `checkSession()` que valida la cookie de sesión vía `GET /api/me` al inicializar la app. El backend usa cookies (no JWT). |
