---
title: "Dropdowns de UI para horas (generación y check de alertas) con formato AM/PM y 24h"
id: "SPEC-039"
status: "released"
author: "p40la-ihost-team"
created: "2026-08-16"
updated: "2026-08-16"
github_issue: 39
---

# Dropdowns de UI para horas (generación y check de alertas) con formato AM/PM y 24h

**ID**: SPEC-039  
**Estado**: released  
**Autor**: p40la-ihost-team  
**Creado**: 2026-08-16  
**Actualizado**: 2026-08-16

---

## 1. Resumen Ejecutivo

En la sección "Facturación automática" de Configuraciones, el campo "Hora de generación" usa un `<select>` nativo del navegador que renderiza las 24 horas en formato militar (00:00–23:00). Este control depende del OS (apariencia y comportamiento no controlables) y no es consistente con el patrón de UI iOS del resto de la app, que ya dispone del componente custom `Select` (`frontend/src/components/Select.tsx`) usado en formularios de servicios, facturas e instituciones.

Esta spec reemplaza el `<select>` nativo por el componente UI `Select`, y agrega un toggle de preferencia de formato de visualización: **AM/PM** (por defecto, ej. "1:00 PM") o **24h** (ej. "13:00"). El valor que se persiste en el backend (`billing_generation_hour`, entero 0–23) no cambia: la conversión de formato es puramente visual en el frontend.

Además, la spec expone y da visibilidad a la **hora de check de alertas** (`alert_check_hour`), que ya existe en el backend pero no se expone en el GET de settings ni se muestra en la UI. Se agrega un segundo dropdown de UI en Settings (mismo formato AM/PM/24h) para configurarla, exponiendo la key en el GET. Ambas horas se configuran con el mismo patrón visual.

La preferencia de formato es un cambio de UI puramente cosmético, por lo que se guarda en `localStorage` del navegador y NO en la base de datos SQLite, evitando migraciones y consumo innecesario en el iHost. El resultado esperado es una experiencia visual consistente, sin dependencias nuevas ni cambios de API.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Reemplazar el `<select>` nativo de "Hora de generación" en `SettingsPage.tsx` por el componente custom `Select` (`../components/Select`), respetando el patrón de UI existente.
2. **REQ-002**: El valor enviado al backend debe seguir siendo el entero 0–23 (`billing_generation_hour`), sin cambios en la API ni en el modelo de datos.
3. **REQ-003**: Mostrar las horas en formato **AM/PM** por defecto (12:00 AM, 1:00 AM, ..., 11:00 AM, 12:00 PM, 1:00 PM, ..., 11:00 PM).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-004**: Agregar un toggle (Switch) de preferencia de formato "24h / AM/PM" junto a los dropdowns. Al cambiar, las etiquetas de ambos dropdowns se actualizan al instante sin perder la selección actual.
2. **REQ-005**: Persistir la preferencia de formato en `localStorage` (clave tipo `hourFormat`) para que se mantenga entre sesiones.
3. **REQ-006**: Exponer `alert_check_hour` en el endpoint `GET /api/settings` (hoy se puede escribir vía PUT pero no se lee).
4. **REQ-007**: Agregar en Settings un dropdown de UI "Hora de check de alertas" (`alert_check_hour`, entero 0–23) con el mismo componente `Select` y formato AM/PM/24h que la hora de generación.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-008**: Respetar el idioma/i18n para las etiquetas estáticas del bloque "Facturación automática" y del nuevo bloque de alertas si ya existen claves de traducción, o documentar su estado actual (hoy el bloque está hardcodeado en español).

### 2.4 Requerimientos Funcionales (P2 - Mejoras iterativas solicitadas por usuario)

1. **REQ-009**: Corregir el overflow del menú desplegable: el menú del `Select` de "Hora de check de alertas" (y el de generación) se recorta dentro del contenedor con `overflow-hidden`. El menú debe renderizarse por encima de todo (portal) o posicionarse fuera del contenedor recortado.
2. **REQ-010**: Activar el buscador (`searchable`) en ambos dropdowns de hora para poder escribir, filtrar por autocompletado y seleccionar con menos interacción.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Sin impacto medible; cambio solo de presentación en el frontend.
- **Seguridad**: Sin cambios de autenticación ni datos sensibles. La preferencia en `localStorage` es del navegador del usuario.
- **Almacenamiento**: Sin cambios en SQLite. Solo ~20 bytes en `localStorage`.
- **Disponibilidad**: Sin cambios de API; el endpoint `system_settings` queda intacto.
- **iHost**: Cero nuevas dependencias. `localStorage` en vez de DB evita migraciones y escritura innecesaria en el iHost.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Implementación actual**: `SettingsPage.tsx` líneas 289–292 construye el array `hours` con labels 24h (`00:00`–`23:00`) y líneas 341–349 renderiza un `<select>` nativo. `billingHour` (0–23) se carga desde `api.systemSettings.get()` y se guarda con `api.systemSettings.update({ billing_generation_hour: hour })`.
- **Componente existente**: `frontend/src/components/Select.tsx` ya implementa el dropdown custom de UI (con click-outside para cerrar, chevron animado, hover states) y se usa en `BillFormPage`, `ServiceFormPage`, `InstitutionFormPage` y `ServicesPage`. No requiere cambios.
- **Toggle existente**: `frontend/src/components/Toggle.tsx` es el switch usado en el resto de la app (email alerts, voice monkey). Se reutiliza para el formato.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Usar componente custom `Select` + toggle localStorage | Consistencia visual, cero dependencias, sin migración DB, preferencia persistente | Cambio menor de código en SettingsPage | ✅ Seleccionada |
| Seguir con `<select>` nativo | Cero código | Depende del OS, inconsistente con la UI | ❌ Rechazada |
| Persistir preferencia en `system_settings` (SQLite) | Preferencia global | Migración/columna nueva, escritura en iHost innecesaria, complejidad | ❌ Rechazada |
| Librería de datepicker/dropdown de terceros | Features avanzadas | Peso extra en bundle, dependencia innecesaria | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Preferencia de formato en localStorage, no en SQLite**
- **Contexto**: El formato de visualización de la hora (AM/PM vs 24h) es una preferencia puramente visual del usuario en un dispositivo.
- **Decisión**: Guardar en `localStorage` del navegador una clave simple (ej. `hourFormat` con valor `"12h"` o `"24h"`).
- **Consecuencias**: Positivas: cero migraciones, cero cambios de API, cero costo en iHost. Negativas: la preferencia es por-navegador (si el usuario entra desde otro dispositivo, vuelve al default AM/PM), lo cual es aceptable para este caso.

**ADR-002: El valor canónico sigue siendo 0–23 en el backend**
- **Contexto**: El backend y el scheduler de generación automática y de check de alertas trabajan con horas 0–23.
- **Decisión**: Mantener el contrato de API intacto. Solo las etiquetas visibles cambian según el formato elegido; al seleccionar una opción se mapea de vuelta al entero 0–23.
- **Consecuencias**: La conversión (hora ↔ label) es una función pura en el frontend, fácil de testear. No hay riesgo de inconsistencia de datos.

**ADR-003: Exponer `alert_check_hour` en el GET de settings**
- **Contexto**: `alert_check_hour` ya existe en el backend (`SetAlertCheckHour`) y se persiste vía PUT, pero el GET no la devuelve, por lo que la UI no puede mostrarla ni leer el valor actual.
- **Decisión**: Agregar `alert_check_hour` al response del `GET /api/settings` usando `GetAlertCheckHour` (que ya aplica fallback a `billing_generation_hour`).
- **Consecuencias**: La UI puede leer y editar la hora de check de alertas sin migraciones ni cambios de esquema.

**ADR-004: Menú del Select en portal para evitar recorte por `overflow-hidden`**
- **Contexto**: El menú desplegable usa `position: absolute` dentro del componente `Select`, que a su vez está dentro de un card con `overflow-hidden` (patrón iOS `rounded-ios shadow-ios overflow-hidden`). El menú se recorta dentro del contenedor.
- **Decisión**: Renderizar el menú vía `createPortal` de `react-dom` al `document.body`, con posicionamiento `fixed` calculado desde el `getBoundingClientRect` del botón. Esto reutiliza `react-dom` (ya presente), sin dependencias nuevas.
- **Consecuencias**: El menú siempre se dibuja por encima de cualquier contenedor con `overflow-hidden`. Se debe recalcular la posición en scroll/resize para mantener la alineación.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[SettingsPage.tsx] --billing_generation_hour(0-23)--> [api.systemSettings] --> [Backend Go] --> [SQLite]
      |          --alert_check_hour(0-23)--------->
      |
      |  <Select options=[{value:0..23, label: AM/PM o 24h}] />   (x2: generación + check de alertas)
      |  <Toggle 24h/AM/PM />  --> localStorage(hourFormat)
```

### 4.2 Componentes

#### 4.2.1 SettingsPage.tsx (modificado)
- **Responsabilidad**: Reemplazar el bloque de "Hora de generación" (líneas 333–352) y agregar el bloque "Hora de check de alertas".
- **Interfaz**: Usa `Select` (options con `value: number 0–23` y `label` formateado) y `Toggle` para el formato.
- **Dependencias**: `Select.tsx`, `Toggle.tsx`, `useI18nStore` (opcional para etiquetas).
- **Ubicación**: `frontend/src/pages/SettingsPage.tsx`.

#### 4.2.2 Función utilitaria de formato (nueva)
- **Responsabilidad**: Convertir hora 0–23 a label según formato.
- **Interfaz**: `formatHourLabel(hour: number, format: '12h' | '24h'): string`
  - `12h`: 0 → "12:00 AM", 1 → "1:00 AM", 12 → "12:00 PM", 13 → "1:00 PM", 23 → "11:00 PM".
  - `24h`: hour → `String(hour).padStart(2, '0') + ':00'`.
- **Dependencias**: Ninguna (función pura). Puede vivir como helper en `SettingsPage.tsx` o en `frontend/src/utils/`.
- **Ubicación**: `frontend/src/pages/SettingsPage.tsx` (o `frontend/src/utils/time.ts` si se prefiere testabilidad).

#### 4.2.3 Backend (modificado)
- **Responsabilidad**: Exponer `alert_check_hour` en el GET de settings.
- **Interfaz**: Agregar la lectura de `GetAlertCheckHour` en `GetSystemSettings` y añadir `"alert_check_hour": alertHour` al response JSON.
- **Dependencias**: `services.SystemSettingsService.GetAlertCheckHour` (ya existe).
- **Ubicación**: `internal/api/system_settings_handlers.go`.

### 4.3 Modelo de datos

No hay cambios de modelo de datos. `billing_generation_hour` y `alert_check_hour` en `system_settings` permanecen como enteros 0–23.

```
Entidad: system_settings (sin cambios)
- billing_generation_hour: integer (0-23)
- alert_check_hour: integer (0-23)  -- ya existe en backend (GetAlertCheckHour/SetAlertCheckHour)
```

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/settings` (modificado: agrega `alert_check_hour`)

**Response 200**:
```json
{
  "billing_generation_hour": 13,
  "alert_check_hour": 8,
  "smtp_host": "...",
  "..."
}
```

#### Endpoint: `PUT /api/settings` (sin cambios de contrato, ya soporta `alert_check_hour`)

**Request**:
```json
{
  "billing_generation_hour": 13,
  "alert_check_hour": 8
}
```

### 4.5 Dependencias

- **Internas**: `frontend/src/components/Select.tsx`, `frontend/src/components/Toggle.tsx` (no se modifican); `internal/api/system_settings_handlers.go` y `internal/services/system_settings.go` (solo lectura/uso de métodos existentes).
- **Externas**: Ninguna nueva. Solo `localStorage` del navegador.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado el bloque "Hora de generación" en Configuraciones, cuando se abre el dropdown, entonces se renderiza el componente UI `Select` (no un `<select>` nativo del navegador).
- [ ] CA-002: Dado el formato por defecto AM/PM, cuando se lista el dropdown, entonces las opciones se muestran como 12:00 AM, 1:00 AM, ..., 11:00 PM.
- [ ] CA-003: Dado un valor guardado de 13, cuando se abre Configuraciones con formato 12h, entonces el dropdown muestra "1:00 PM" seleccionado.
- [ ] CA-004: Dado el toggle de formato, cuando se cambia de AM/PM a 24h, entonces las opciones se muestran como 00:00, 01:00, ..., 23:00 y la selección actual se conserva.
- [ ] CA-005: Dado un cambio de hora con formato AM/PM, cuando se selecciona "1:00 PM", entonces se persiste `billing_generation_hour: 13` vía `api.systemSettings.update`.
- [ ] CA-006: Dado que el usuario cambió la preferencia de formato, cuando recarga la página, entonces el formato elegido se mantiene (persistencia en `localStorage`).
- [ ] CA-007: Dado que no hay formato guardado en `localStorage`, cuando se abre Configuraciones, entonces se usa AM/PM por defecto.
- [ ] CA-008: Dado `GET /api/settings`, cuando se consulta, entonces el response incluye `alert_check_hour` con el valor configurado (o el fallback a `billing_generation_hour` si no está seteado).
- [ ] CA-009: Dado el nuevo bloque "Hora de check de alertas" en Configuraciones, cuando se cambia su valor con formato AM/PM, entonces se persiste `alert_check_hour` vía `api.systemSettings.update`.
- [ ] CA-010: Dado el toggle de formato, cuando se cambia de formato, entonces AMBOS dropdowns (generación y check de alertas) se actualizan de forma consistente.
- [ ] CA-011: Dado el dropdown de hora (generación o check de alertas) dentro del card con `overflow-hidden`, cuando se abre el menú, entonces el menú se muestra completo sin recortarse.
- [ ] CA-012: Dado un dropdown de hora, cuando se escribe texto en el buscador, entonces las opciones se filtran por autocompletado y se puede seleccionar una con un click.

### 5.2 No funcionales

- [ ] CA-NF-001: No se agregan dependencias nuevas al `package.json` (el bundle no crece significativamente).
- [ ] CA-NF-002: No hay migraciones de base de datos.
- [ ] CA-NF-003: La app sigue funcionando correctamente en iHost con recursos limitados (misma huella de memoria que antes).

### 5.3 Testing

- **Unit tests**: Función `formatHourLabel` (0 → 12:00 AM, 12 → 12:00 PM, 13 → 1:00 PM / 13:00, 23 → 11:00 PM / 23:00, round-trip 0–23). En Go: `GetAlertCheckHour` con fallback a `billing_generation_hour` cuando `alert_check_hour` está vacío.
- **Integration tests**: Verificar que seleccionar un label 12h persiste el entero correcto 0–23 en el endpoint para ambas keys; verificar que el GET devuelve `alert_check_hour`.
- **E2E tests**: Flujo manual en Configuraciones: abrir ambos dropdowns, cambiar formato con toggle, recargar página y confirmar persistencia.
- **Carga/Performance**: N/A (sin impacto medible en iHost).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Exponer `alert_check_hour` en `GET /api/settings` (backend) | 0.25 días | Ninguna |
| 2 | Crear helper `formatHourLabel` y construir options según formato | 0.5 días | Ninguna |
| 3 | Reemplazar `<select>` por `Select`, agregar `Toggle` de formato con persistencia en `localStorage` y el dropdown de check de alertas | 0.5 días | Fase 1, 2 |
| 4 | Fix overflow: menú del `Select` en portal (`createPortal` + `position: fixed`) | 0.5 días | Fase 3 |
| 5 | Activar `searchable` en ambos dropdowns de hora | 0.25 días | Fase 4 |
| 6 | Build de frontend (`npm run build`) y validación manual local | 0.5 días | Fase 5 |
| 7 | Pruebas del usuario + release | 0.5 días | Fase 6 |

### 6.2 Milestones

1. **MVP**: Dropdown UI con formato AM/PM para la hora de generación (REQ-001, REQ-002, REQ-003).
2. **V1.0**: Toggle 24h/AM/PM con persistencia en `localStorage` + dropdown de check de alertas y exposición en GET (REQ-004, REQ-005, REQ-006, REQ-007).
3. **V1.1**: Fix de overflow del menú + buscador en dropdowns (REQ-009, REQ-010).

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Romper la persistencia del valor al cambiar el mapeo de labels | Media | Alto | Mantener el `value` de cada option como entero 0–23; el label es solo presentación |
| El dropdown custom no cierra bien en móvil | Baja | Medio | `Select.tsx` ya maneja click-outside; validar en responsive |
| Usuario con formato 24h prefiere default distinto | Baja | Bajo | La preferencia queda en localStorage; default AM/PM es decisión de producto |
| Romper el fallback de `alert_check_hour` a `billing_generation_hour` al exponerlo en GET | Baja | Medio | Reutilizar `GetAlertCheckHour` (ya implementa el fallback); añadir test en Go |

## 8. Notas y Referencias

- Componente `Select` existente: `frontend/src/components/Select.tsx`
- Componente `Toggle` existente: `frontend/src/components/Toggle.tsx`
- Página a modificar: `frontend/src/pages/SettingsPage.tsx` (bloque "Hora de generación", líneas ~289–352)
- Backend: `internal/api/system_settings_handlers.go` (`GetSystemSettings`, `UpdateSystemSettings`) y `internal/services/system_settings.go` (`GetAlertCheckHour`/`SetAlertCheckHour`, líneas ~199–221)
- Regla AGENTS.md: la fuente de verdad del i18n es `frontend/public/i18n/`; si se agregan claves de traducción, editar ahí y rebuildear.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-16 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-08-16 | p40la-ihost-team | Alcance ampliado: exponer `alert_check_hour` en GET y agregar dropdown de UI para hora de check de alertas (solicitado por usuario) |
| 2026-08-16 | p40la-ihost-team | Estado: draft → in_progress. Inicio del desarrollo |
| 2026-08-16 | p40la-ihost-team | Mejoras iterativas (usuario): fix overflow del menú del Select (portal) + buscador `searchable` en ambos dropdowns de hora (REQ-009, REQ-010) |
| 2026-08-16 | p40la-ihost-team | Released. Código en `main` (merge ed7950b), issue #39 cerrado con label `spec/released`. Criterios de aceptación CA-001 a CA-012 pasan |
