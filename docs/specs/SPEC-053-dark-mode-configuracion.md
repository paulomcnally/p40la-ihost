---
title: "Modo oscuro configurable con default activo"
id: "SPEC-053"
status: "released"
author: "p40la-ihost-team"
created: "2026-09-03"
updated: "2026-09-03"
github_issue: 53
---

# Modo oscuro configurable con default activo

**ID**: SPEC-053  
**Estado**: released  
**Autor**: p40la-ihost-team  
**Creado**: 2026-09-03  
**Actualizado**: 2026-09-03

---

## 1. Resumen Ejecutivo

La aplicación actual tiene un tema claro fijo: los colores se definen en `tailwind.config.js` con valores hex hardcodeados y `body { background:#f2f2f7; color:#1c1c1e }` en `index.css`. No existe ninguna infraestructura de theming (`darkMode`, CSS variables, `prefers-color-scheme` ni clase `dark`). Se necesita habilitar un **modo oscuro** que se pueda activar/desactivar desde Configuraciones, con el **modo oscuro activo por defecto** para usuarios nuevos.

El reto principal es **no romper la UI existente**: el tema claro actual debe verse idéntico cuando el modo oscuro esté desactivado. La investigación determinó que ~90% de la superficie usa 9 tokens de paleta (`bg-bg`, `bg-card`, `border-border`, `text-text-secondary`, `bg-primary`, etc.), lo que permite adaptar la mayoría del tema automáticamente re-apuntando esos tokens a CSS variables con overrides en un bloque `.dark`. Los ~30 archivos restantes usan colores hardcodeados (inputs `bg-white`, badges `bg-green-100 text-green-700`, grises de estado inactivo, `Toggle`) y requieren variantes `dark:` puntuales.

Impacto en iHost: cero. El cambio es 100% frontend (CSS + clase en `<html>`), sin dependencias nuevas, sin cambios de backend ni de SQLite. El build sigue siendo estático servido por el binario Go. Memoria y almacenamiento inalterados.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Agregar un toggle de modo oscuro en la sección **General** de Configuraciones (`SettingsPage.tsx`), reutilizando el componente `Toggle`.
2. **REQ-002**: El modo oscuro debe estar **activo por defecto** cuando no existe preferencia guardada (usuarios nuevos).
3. **REQ-003**: La preferencia debe persistir en `localStorage` siguiendo el patrón existente de `hourFormat` (clave `darkMode`, valores `dark`/`light`).
4. **REQ-004**: Aplicar la clase `dark` en `document.documentElement` y configurar `darkMode: 'class'` en `tailwind.config.js`.
5. **REQ-005**: Refactorizar la paleta a CSS variables (`:root` claro + `.dark` oscuro) y añadir variantes `dark:` para los colores hardcodeados de los ~30 archivos afectados.
6. **REQ-006**: Agregar claves i18n `settings.dark_mode.*` en `frontend/public/i18n/{es,en}.json` (fuente de verdad) y regenerar el build.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-007**: Aplicar la clase `dark` antes del primer render (script inline en `frontend/index.html`) para evitar flash de tema incorrecto al cargar.
2. **REQ-008**: Cubrir todos los componentes con colores hardcodeados: inputs, badges de estado, sidebar, header, modales, `Toggle` (off-track y knob), `Toast` y `RegistrosPage`.
3. **REQ-009**: Verificar que las páginas `/login` y `/setup` (fuera del `DashboardLayout`) también se vean correctas en modo oscuro.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-010**: (Opcional) Transición suave de colores (`transition-colors`) en elementos clave. Solo si no introduce riesgo de performance en iHost.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Sin impacto medible en tiempo de respuesta. Solo CSS + una clase en `<html>`.
- **Seguridad**: Sin datos sensibles involucrados. Preferencia local del navegador.
- **Almacenamiento**: Sin cambios en SQLite. Una clave en `localStorage`.
- **Disponibilidad**: Sin cambios en backend, health checks ni deploy.
- **iHost**: Sin dependencias nuevas. Build estático igual que hoy. Memoria y CPU inalterados.
- **Compatibilidad**: El tema claro actual DEBE verse idéntico cuando el modo oscuro está desactivado (regresión cero).

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- `tailwind.config.js` (31 líneas): paleta de 9 tokens con hex hardcodeados, **sin `darkMode` configurado**, sin CSS variables.
- `src/index.css`: `body { background:#f2f2f7; color:#1c1c1e }` — únicos hex en el frontend.
- 44 archivos TSX usan clases de color: 26 páginas + 17 componentes + `App.tsx`.
  - **14 archivos** usan SOLO tokens de paleta → se adaptan automáticamente si los tokens se re-apuntan a variables.
  - **30 archivos** mezclan colores hardcodeados estándar de Tailwind (`bg-white`, `bg-green-100`, `text-red-500`, `bg-gray-100/200`, `bg-emerald-600/10`, etc.) → requieren `dark:`.
- Iconos usan `stroke: currentColor` → tema-aware automáticamente, sin trabajo.
- `Toggle.tsx`: off-track `bg-border`, knob `after:bg-white` → necesita variante oscura.
- Patrón de persistencia existente: `hourFormat` en `SettingsPage.tsx` (lazy initializer + `localStorage.setItem`).
- i18n: `settings` ya tiene `section_general` y `language`; una sección `settings.dark_mode.*` encaja como hermana.
- Sin `prefers-color-scheme`, sin clase `dark` en ninguna parte. Infraestructura de theming inexistente.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A: Paleta → CSS variables + bloque `.dark` | ~90% del tema se adapta solo; los 14 archivos de solo-tokens no se tocan; el claro actual queda idéntico (`:root` mantiene los valores actuales) | Requiere tocar `tailwind.config.js` e `index.css` con cuidado | ✅ Seleccionada |
| B: Solo variantes `dark:` en cada archivo | Sin refactor de config | Duplica el trabajo (~44 archivos); alto riesgo de olvidar clases | ❌ Rechazada |
| C: `darkMode: 'media'` (sigue al sistema) | Cero UI para configurar | No cumple REQ-001/002 (toggle manual + default dark) | ❌ Rechazada |
| D: Librería de theming (next-themes, etc.) | Comodidad | Dependencia nueva contraria a la política de mínimas dependencias en iHost | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Modo oscuro basado en clase `dark` en `<html>`
- **Contexto**: El requerimiento exige toggle manual con default activo; `media` no aplica.
- **Decisión**: `darkMode: 'class'` en Tailwind + toggle en Settings que agrega/remueve `dark` en `document.documentElement`.
- **Consecuencias**: Positivo: control total del tema. Negativo: requiere que cada elemento con colores hardcodeados tenga su variante `dark:`.

**ADR-002**: Paleta como CSS variables con overrides en `.dark`
- **Contexto**: Evitar tocar los 14 archivos que solo usan tokens y garantizar regresión cero en el tema claro.
- **Decisión**: Definir `--color-bg`, `--color-card`, `--color-border`, `--color-text-secondary`, etc. en `:root` (valores actuales) y en `.dark` (valores oscuros). `tailwind.config.js` referencia `var(--...)`.
- **Consecuencias**: Positivo: tema claro idéntico por construcción; negativo: hay que auditar los usos de `bg-primary/10` y opacidades para asegurar que las variables soporten `color-mix`/opacity en Tailwind (ver riesgo R-004).

**ADR-003**: Persistencia local con localStorage (patrón `hourFormat`)
- **Contexto**: La preferencia es del navegador del usuario; no hay backend involucrado.
- **Decisión**: Clave `darkMode` con valores `dark`/`light`. Ausencia de valor → `dark` (default). Escritura en el handler del toggle.
- **Consecuencias**: Simple y consistente con el código existente. Sin migraciones DB.

**ADR-004**: Script inline anti-flash en `index.html`
- **Contexto**: El default es oscuro; sin aplicar la clase antes del primer paint se vería un flash claro.
- **Decisión**: Script inline en `<head>` que lee `localStorage` y agrega `dark` si no hay preferencia o es `dark`.
- **Consecuencias**: Positivo: sin flash. Negativo: script inline mínimo en el HTML (aceptable, sin dependencias).

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[SettingsPage] --toggle--> [handleDarkModeChange]
        |                        |
        |                        v
        |              localStorage 'darkMode' ('dark'|'light')
        |                        |
        |                        v
        |              document.documentElement.classList.toggle('dark')
        |                        |
        v                        v
[index.html script]    [tailwind darkMode:'class' + CSS vars]
   (anti-flash)                 |
                                v
                    :root (claro)  .dark (oscuro)  dark:* variants
                                |
                                v
                    Toda la UI (44 archivos TSX)
```

### 4.2 Componentes

#### 4.2.1 `frontend/tailwind.config.js`
- **Responsabilidad**: Habilitar `darkMode: 'class'` y re-apuntar la paleta a CSS variables.
- **Cambio**: `darkMode: 'class'`; colores → `rgb(var(--color-xxx) / <alpha-value>)` o `var(--color-xxx)`.
- **Dependencias**: Ninguna nueva.

#### 4.2.2 `frontend/src/index.css`
- **Responsabilidad**: Definir `:root` (valores actuales) y `.dark` (paleta oscura) + `body` con variables.
- **Cambio**: `body` usa `var(--color-bg)` / `var(--color-text)`; nuevos bloques `:root` y `.dark`.
- **Dependencias**: Nada.

#### 4.2.3 `frontend/src/pages/SettingsPage.tsx`
- **Responsabilidad**: Renderizar el toggle de modo oscuro en la sección General.
- **Cambio**: Estado `darkMode` (lazy init de localStorage, default `dark`), handler que escribe localStorage y toglea la clase en `<html>`, fila con `Toggle` + `t('settings.dark_mode.*')`.
- **Dependencias**: `Toggle`, `useI18nStore`.

#### 4.2.4 `frontend/index.html`
- **Responsabilidad**: Aplicar la clase `dark` antes del primer paint.
- **Cambio**: Script inline en `<head>`.
- **Dependencias**: Nada.

#### 4.2.5 Archivos con colores hardcodeados (~30)
- **Responsabilidad**: Añadir variantes `dark:` para inputs (`bg-white` → `dark:bg-...`), badges de estado, grises inactivos, `Toggle`, `Toast`, `RegistrosPage`.
- **Cambio**: Clases `dark:` puntuales, sin alterar el tema claro.
- **Dependencias**: Config de Tailwind (paso 1).

### 4.3 Modelo de datos

```
Sin cambios de base de datos (SQLite intacta).

localStorage:
- clave: 'darkMode'
- valores: 'dark' (default) | 'light'
```

### 4.4 APIs / Contratos

Sin cambios de API backend. El cambio es 100% cliente.

### 4.5 Dependencias

- **Internas**: `frontend/tailwind.config.js`, `frontend/src/index.css`, `frontend/src/pages/SettingsPage.tsx`, `frontend/index.html`, `frontend/src/components/Toggle.tsx` y los ~30 archivos con colores hardcodeados.
- **Externas**: Ninguna nueva (no se agrega librería de theming).

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado un usuario en Configuraciones, cuando abre la sección General, entonces ve un toggle "Modo oscuro" que refleja el estado actual.
- [ ] CA-002: Dado un usuario sin preferencia guardada (primer uso), cuando carga la app, entonces el tema es oscuro.
- [ ] CA-003: Dado un usuario con modo oscuro activo, cuando apaga el toggle, entonces la app cambia a tema claro inmediatamente sin recargar.
- [ ] CA-004: Dado un usuario que cambió el tema, cuando recarga la página, entonces la preferencia se mantiene.
- [ ] CA-005: Dado modo claro activo, cuando se recorre la app (dashboard, servicios, facturas, autos, pensión, configuraciones, modales, login), entonces la UI es idéntica a la actual (regresión cero).
- [ ] CA-006: Dado modo oscuro activo, cuando se recorre la app, entonces no hay texto ilegible ni elementos blancos sobre blanco.
- [ ] CA-007: Dado el toggle, cuando se navega, entonces los textos de título/subtítulo usan las claves i18n nuevas en español e inglés.
- [ ] CA-008: Dado el build regenerado, entonces `curl http://localhost:8088/i18n/es.json` y `/en.json` sirven las claves `settings.dark_mode.*`.

### 5.2 No funcionales

- [ ] CA-NF-001: El bundle de JS no agrega dependencias runtime nuevas; el tamaño aumenta solo por el CSS de las variantes `dark:`.
- [ ] CA-NF-002: No hay cambios en backend, migraciones ni esquema SQLite.
- [ ] CA-NF-003: No hay flash de tema claro al cargar la app con modo oscuro activo.

### 5.3 Testing

- **Unit tests**: Handler del toggle (lectura/escritura localStorage, aplicación de clase) si el proyecto tiene infra de tests de frontend.
- **Integration tests**: Persistencia tras recarga; default dark sin preferencia.
- **E2E tests**: Recorrido manual de las páginas listadas en CA-005 en ambos temas.
- **Carga/Performance**: Verificar en iHost que el toggle no cause re-renders globales costosos.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Config: `darkMode: 'class'` + paleta a CSS variables en `tailwind.config.js` e `index.css` (manteniendo claro idéntico) | 0.5 día | Ninguna |
| 2 | Script anti-flash en `index.html` + aplicación inicial de la clase `dark` | 0.5 día | Fase 1 |
| 3 | Toggle en `SettingsPage.tsx` + claves i18n `settings.dark_mode.*` en es/en + build | 0.5 día | Fase 1 |
| 4 | Variantes `dark:` en los ~30 archivos con colores hardcodeados (inputs, badges, Toggle, Toast, RegistrosPage, login/setup) | 1 día | Fase 1 |
| 5 | Validación manual completa en ambos temas + build + pruebas en local | 0.5 día | Fases 2-4 |

### 6.2 Milestones

1. **MVP**: Fases 1-3 (toggle funcional con default dark y persistencia, con ajustes básicos de inputs).
2. **V1.0**: Fases 1-5 (toda la UI legible en ambos temas, sin regresiones).

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| R-001: Regresión en el tema claro al re-apuntar tokens a CSS variables | Media | Alto | Mantener exactamente los hex actuales en `:root`; validar CA-005 con capturas antes/después |
| R-002: Olvidar un `dark:` en algún archivo → texto ilegible | Media | Medio | Checklist por página (CA-006) y recorrido manual completo en ambos temas |
| R-003: Flash de tema claro al cargar | Media | Medio | Script inline anti-flash en `<head>` (REQ-007) |
| R-004: Opacidades como `bg-primary/10` no funcionen con variables | Media | Medio | Usar sintaxis `rgb(var(--color-primary) / <alpha-value>)` en Tailwind para soportar `/opacity`; verificar Sidebar (active `bg-primary/10`) |
| R-005: Toast y badges de color con fondos claros poco legibles en oscuro | Media | Bajo | Variantes `dark:` específicas por componente |

## 8. Notas y Referencias

- Precedente i18n (SPEC-032/033): la fuente de verdad es `frontend/public/i18n/`, NO `public/i18n/` (build output que se borra con `emptyOutDir`).
- Patrón de persistencia existente: `hourFormat` en `frontend/src/pages/SettingsPage.tsx`.
- Tailwind `darkMode: 'class'`: https://tailwindcss.com/docs/dark-mode
- CSS variables en Tailwind: https://tailwindcss.com/docs/customizing-colors#using-css-variables

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-03 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-09-03 | p40la-ihost-team | Transición draft → pending_execution → in_progress. Inicio de implementación |
| 2026-09-03 | p40la-ihost-team | Release: commit `e343a5b` en `main` (feature/SPEC-053 mergeado). Pruebas manuales del usuario satisfactorias. Estado → released |