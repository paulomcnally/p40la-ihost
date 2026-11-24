---
title: "UI de API key global del MCP en Configuración"
id: "SPEC-149"
status: "pending_release"
author: "opencode"
created: "2026-11-23"
updated: "2026-11-23"
github_issue: 152
---

# UI de API key global del MCP en Configuración

**ID**: SPEC-149  
**Estado**: pending_release  
**Autor**: opencode  
**Creado**: 2026-11-23  
**Actualizado**: 2026-11-23

---

## 1. Resumen Ejecutivo

SPEC-145 agregó los endpoints para generar, consultar y revocar la API key global que usa el MCP, pero **no agregó interfaz**. Hoy el usuario tiene que abrir DevTools o usar `scripts/create-api-key.sh` para obtenerla. Eso es una fricción innecesaria para una operación básica de configuración.

Esta spec agrega una página en **Configuración → MCP** con estado de la key, botón para generarla, modal de una sola visualización con copiar, regeneración y revocación con confirmación. No cambia el backend: consume los endpoints existentes (`/api/api-key`), que ya están protegidos por sesión.

---

## 2. Cumplimiento de reglas

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `UI-INPUT-TOKENS` | Inputs con `bg-card`, `text-text`, tokens del tema | `CA-UI-INPUT-TOKENS` |
| `UI-DARKMODE` | Página y modales verificados en darkmode | `CA-UI-DARKMODE` |
| `UI-I18N` | Claves nuevas en `frontend/public/i18n/{es,en}.json`, paridad | `CA-UI-I18N` |
| `UI-STYLE` | Card iOS, `rounded-ios`, `shadow-ios`, acciones primarias/secundarias | `CA-UI-STYLE` |
| `UI-SELECT` / `UI-TOGGLE` / `UI-NUMBER-INPUT` | N/A: no hay selects, booleanos de entidad ni campos numéricos | — |
| `UX-MODAL` | Modal de key y confirmaciones con cierre visible, sin click fuera, z-index correcto | `CA-UX-MODAL` |
| `UX-MODAL-CONFIRM` | Regenerar y revocar usan `ConfirmWordModal` con foco y Enter | `CA-UX-MODAL-CONFIRM` |
| `UX-FEEDBACK` | Toasts de éxito/error; estado de carga y botones deshabilitados | `CA-UX-FEEDBACK` |
| `UX-LOADING` | Skeleton/spinner al cargar el estado de la key | `CA-UX-LOADING` |
| `SEC-NO-PII` | La key se muestra solo en memoria; no se loguea ni se persiste | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Ejemplos y tests con `demo-api-key` | `CA-SEC-SYNTHETIC` |
| `SEC-LOGS` | Sin logs de la key ni de headers | `CA-SEC-LOGS` |
| `TEST-LOCAL` | `npm run check:ui`, `check:i18n`, `check:numbers` y build | `CA-TEST-LOCAL` |
| `DELIV-WORKTREE` | Trabajo en `../p40la-ihost-spec-149`, rama `feature/SPEC-149` | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | Merge y release con confirmación del usuario | `CA-DELIV-CONFIRM` |
| `DELIV-DEPLOY` | El release publica la imagen para actualizar el iHost | `CA-DELIV-DEPLOY` |
| `DATA-*` | N/A: sin cambios de DB ni migraciones | — |

---

## 3. Requerimientos

### 3.1 Funcionales (P0)

1. **REQ-001 — Acceso**: fila **MCP** en Configuración (sección avanzada) que navega a `/settings/mcp`.
2. **REQ-002 — Estado**: la página muestra si la key está configurada, el prefijo, la fecha de creación y el último uso.
3. **REQ-003 — Generar**: si no hay key, botón **Generar API key** que llama `POST /api/api-key/regenerate`.
4. **REQ-004 — Mostrar una vez**: la key completa se muestra en un modal con botón **Copiar** y advertencia de que no se vuelve a mostrar. No se guarda en `localStorage` ni en el estado global.
5. **REQ-005 — Regenerar**: si ya hay key, acción **Regenerar** con confirmación por palabra; invalida la anterior.
6. **REQ-006 — Revocar**: acción **Revocar** con confirmación por palabra; llama `DELETE /api/api-key`.
7. **REQ-007 — Feedback**: toasts de éxito/error; errores claros; botones deshabilitados durante la operación.
8. **REQ-008 — i18n**: claves en español e inglés.
9. **REQ-009 — Darkmode**: legible en ambos temas.

### 3.2 No funcionales

- Sin dependencias nuevas.
- Sin cambios de backend ni de esquema.
- La key nunca se escribe en logs, URL ni localStorage.
- La página funciona en móvil y desktop.

---

## 4. Diseño Técnico

### 4.1 Componentes

- `frontend/src/pages/SettingsMCPPage.tsx` (nuevo).
- `frontend/src/api/index.ts`: `mcpAPIKey.get/regenerate/revoke`.
- `frontend/src/App.tsx`: ruta `settings/mcp`.
- `frontend/src/pages/SettingsPage.tsx`: fila de navegación.
- `frontend/public/i18n/{es,en}.json`: `settings.mcp.*`.

### 4.2 Contratos

- `GET /api/api-key` → `{ configured, prefix, created_at, last_used_at }`.
- `POST /api/api-key/regenerate` → `{ api_key, prefix, created_at }`.
- `DELETE /api/api-key` → `{ message }`.

### 4.3 Flujo

1. Al entrar, cargar estado.
2. Sin key → botón Generar. Con key → Regenerar y Revocar.
3. Generar/Regenerar → modal con la key, copiar, Listo.
4. Revocar → confirmación por palabra → estado “sin configurar”.

---

## 5. Criterios de Aceptación

- [ ] **CA-MCP-UI-001**: Configuración muestra la fila MCP y navega a `/settings/mcp`.
- [ ] **CA-MCP-UI-002**: sin key, se ve “Sin configurar” y el botón Generar.
- [ ] **CA-MCP-UI-003**: al generar, se muestra la key una sola vez con Copiar y Listo; no queda en localStorage.
- [ ] **CA-MCP-UI-004**: regenerar pide confirmación y actualiza el prefijo.
- [ ] **CA-MCP-UI-005**: revocar pide confirmación y deja el estado sin configurar.
- [ ] **CA-MCP-UI-006**: toasts de éxito/error y botones deshabilitados durante la operación.
- [ ] **CA-MCP-UI-007**: `check:ui`, `check:i18n`, `check:numbers` y build pasan.
- [ ] **CA-MCP-UI-008**: darkmode legible.
- [ ] **CA-UX-MODAL**: modales con cierre visible y sin click fuera.
- [ ] **CA-UX-MODAL-CONFIRM**: confirmación por palabra con foco y Enter.
- [ ] **CA-SEC-SYNTHETIC**: ejemplos `demo-api-key`.

---

## 6. Plan

| Fase | Descripción | Estimación |
|---|---|---|
| 1 | Cliente API + página + ruta + fila en Configuración | 0.5 día |
| 2 | i18n es/en + modales + toasts | 0.5 día |
| 3 | Checks frontend, build, validación manual y release | 0.5 día |

---

## 7. Riesgos

| Riesgo | Mitigación |
|---|---|
| Exponer la key en logs o estado persistente | Solo estado local del modal; no logs; no localStorage |
| Regeneración accidental | Confirmación por palabra |
| Claves i18n faltantes | `check:i18n` |
| Regresión visual en darkmode | Verificación manual |

---

## 8. Historial

| Fecha | Autor | Descripción |
|---|---|---|
| 2026-11-23 | opencode | Creación inicial |
| 2026-11-23 | opencode | Implementación de la página, ruta, cliente API, i18n y checks frontend |
