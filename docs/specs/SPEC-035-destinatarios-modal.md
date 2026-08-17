---
title: "Destinatarios con modal: alta/baja de emails con guardado automático"
id: "SPEC-035"
status: "released"
author: "paulomcnally"
created: "2026-08-16"
updated: "2026-08-16"
github_issue: 35
---

# Destinatarios con modal: alta/baja de emails con guardado automático

**ID**: SPEC-035  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-08-16  
**Actualizado**: 2026-08-16

---

## 1. Resumen Ejecutivo

La sección "Destinatarios" de Configuraciones (Alertas por Email) usa hoy un `<input type="text">` donde el usuario debe escribir una lista de correos separada por comas (`SettingsPage.tsx:413`). Esta experiencia es mala: el usuario no ve con claridad qué emails están cargados, comete errores de formato fácilmente y no hay validación visual de cada dirección.

Se propone reemplazar ese input por una lista de chips (tags) de emails con un modal para agregar cada dirección individualmente. Cada alta y cada baja se persiste automáticamente contra la API existente (`systemSettings.update`), sin necesidad de un botón "Guardar" extra. La validación de formato y la prevención de duplicados se hacen en el modal antes de agregar el chip.

El cambio es **frontend-only**: el backend sigue almacenando la lista comma-separated en la key `alert_emails` (sin migración de DB, sin nuevas dependencias), lo que respeta las restricciones de iHost (bajo consumo, SQLite, mínimas dependencias).

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Reemplazar el input texto comma-separated de "Destinatarios" por una lista de chips de emails (uno por destinatario).
2. **REQ-002**: Proporcionar un botón "Agregar" (o "Agregar email") que abre un modal para ingresar una única dirección de email.
3. **REQ-003**: El modal debe validar el formato del email antes de agregarlo. Si es inválido, mostrar toast de error y no agregar el chip.
4. **REQ-004**: El modal debe rechazar emails duplicados (comparación case-insensitive) con toast de error.
5. **REQ-005**: Cada chip debe poder eliminarse con un botón "X" (con modal de confirmación o acción directa según patrón existente de la app).
6. **REQ-006**: **Guardado automático**: cada alta y cada baja de email debe persistirse inmediatamente vía `api.systemSettings.update({ alert_emails })` con la lista recalculada. No se requiere botón "Guardar" extra.
7. **REQ-010**: El modal debe cerrarse automáticamente tras agregar un email con éxito. No debe cerrarse al hacer click fuera del modal (patrón SPEC-023).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-007**: Internacionalizar todas las cadenas nuevas en `frontend/public/i18n/{es,en}.json` (fuente de verdad, ver AGENTS.md) y correr el build de Vite para regenerar `public/`.
2. **REQ-008**: Mostrar estado de carga durante el guardado automático (evitar doble click / guardados concurrentes).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-009**: Al cargar la página, los emails existentes se muestran como chips desde `data.alert_emails`.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Cambio puramente visual; sin impacto medible en memoria/CPU del iHost.
- **Seguridad**: Sin cambios en autenticación ni datos sensibles. Los emails ya se consideran dato no sensible (se exponen en GET settings).
- **Almacenamiento**: Sin migración de DB; la key `alert_emails` mantiene el formato comma-separated.
- **Disponibilidad**: El guardado automático reutiliza el endpoint `PUT /api/system-settings` existente; si falla, se revierte el estado local y se muestra toast de error.
- **iHost**: Cero dependencias nuevas; solo JS en el bundle del frontend estático (build fuera del iHost).

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- `frontend/src/pages/SettingsPage.tsx:25` — estado `alertEmails: string` (lista comma-separated).
- `frontend/src/pages/SettingsPage.tsx:72` — carga desde `data.alert_emails`.
- `frontend/src/pages/SettingsPage.tsx:123-147` — `handleSaveEmail` persiste `alert_emails` como string.
- `frontend/src/pages/SettingsPage.tsx:410-414` — render del input actual con hint "Separados por coma".
- `internal/services/system_settings.go:185-197` — `GetAlertEmails`/`SetAlertEmails` parsean/unan listas comma-separated.
- `internal/api/system_settings_handlers.go:77,120,127-140` — `settingsRequest.AlertEmails *string`, `joinEmails()`, endpoint de update.
- Patrones de modal existentes en la app: `AddInsuranceModal.tsx`, `IconPickerModal.tsx`, `InstitutionCategoriesModal.tsx`, `UploadBillModal.tsx`.
- Fuente de verdad i18n: `frontend/public/i18n/` (AGENTS.md — `public/i18n/` es salida de build).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Guardar automático por alta/baja (sin botón Guardar) | UX inmediata, sin estado pendiente | Más llamadas a API | ✅ Seleccionada (decisión del usuario) |
| Guardar todo con el botón "Guardar" existente | Una sola llamada a API | Estado local sin persistir, confusión con SMTP | ❌ Rechazada |
| Modal sin validación | Menos código | Mantiene errores de formato | ❌ Rechazada |
| Validación de formato + sin duplicados | Evita correos inválidos/duplicados | Validación extra | ✅ Seleccionada (decisión del usuario) |
| Cambio de storage backend a JSON/array | Modelo más limpio | Migración DB + más código en iHost | ❌ Rechazada (backend intacto) |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Backend intacto (storage comma-separated)
- **Contexto**: El backend ya maneja listas comma-separated (`parseEmails`/`sanitizeEmails`/`joinEmails`). Cambiarlo agregaría migración y código en el iHost sin beneficio de UX.
- **Decisión**: El frontend transforma `string[]` ↔ string al comunicarse con la API. `alert_emails` sigue siendo un string.
- **Consecuencias**: Cero cambios en Go, cero migraciones. El frontend es el único responsable de construir la lista con `join(',')`.

**ADR-002**: Guardado automático por evento
- **Contexto**: El usuario pidió que cada alta/baja persista inmediatamente.
- **Decisión**: Cada add/remove llama `api.systemSettings.update({ alert_emails: emails.join(',') })`. Ante error, se revierte el estado y se muestra toast.
- **Consecuencias**: Más requests HTTP, pero son ligeros (string corto). Sin botón extra.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[SettingsPage (chips + botón Agregar)]
        │
        ▼
[Modal Agregar email]  ──valida──▶ [chips local: string[]]
        │                                   │
        ▼                                   ▼
  toast de error (inválido/dup)   api.systemSettings.update
                                       { alert_emails: join(',') }
                                               │
                                               ▼
                                        backend: key "alert_emails"
```

### 4.2 Componentes

#### 4.2.1 SettingsPage (sección Destinatarios)
- **Responsabilidad**: Renderizar chips de destinatarios, botón "Agregar email" que abre el modal, y persistir cada cambio automáticamente.
- **Interfaz**: Reutiliza `api.systemSettings.get/update`.
- **Dependencias**: Modal nuevo (o inline en SettingsPage), `showToast`, i18n.
- **Ubicación**: `frontend/src/pages/SettingsPage.tsx`

#### 4.2.2 Modal Agregar email
- **Responsabilidad**: Capturar un email, validar formato y duplicados, agregar a la lista y disparar guardado.
- **Interfaz**: Props con callbacks `onAdd(email)` / `onClose` o similar, siguiendo el patrón de `AddInsuranceModal.tsx`.
- **Dependencias**: Ninguna nueva.
- **Ubicación**: `frontend/src/components/` (nuevo componente `EmailRecipientsModal.tsx` o inline en SettingsPage — decisión de implementación).

### 4.3 Modelo de datos

```
Key: alert_emails (system_settings)
- valor: string comma-separated (ej: "a@x.com,b@y.com")
- parse: []string via parseEmails()
- sin cambios de schema (0 migraciones)
```

### 4.4 APIs / Contratos

#### Endpoint: `PUT /api/system-settings` (existente, sin cambios)

**Request** (alta/baja de un email):
```json
{
  "alert_emails": "a@x.com,b@y.com"
}
```

**Response 200**:
```json
{
  "billing_generation_hour": 9,
  "alert_emails": "a@x.com,b@y.com"
}
```

**Response Error**:
```json
{
  "error": "internal_error",
  "message": "descripción"
}
```

### 4.5 Dependencias

- **Internas**: `frontend/src/pages/SettingsPage.tsx`, `frontend/src/api/index.ts` (sin cambios), `frontend/public/i18n/{es,en}.json`.
- **Externas**: Ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado un email válido, cuando se confirma en el modal, entonces aparece como chip y se persiste vía `systemSettings.update`.
- [ ] CA-002: Dado un email inválido (sin `@`, sin dominio), cuando se confirma, entonces se muestra toast de error y NO se agrega el chip.
- [ ] CA-003: Dado un email ya existente (case-insensitive), cuando se intenta agregar, entonces se muestra toast de error y NO se agrega duplicado.
- [ ] CA-004: Dado un chip existente, cuando se presiona eliminar, entonces el chip desaparece y la lista se persiste sin ese email.
- [ ] CA-005: Al recargar la página, los emails guardados aparecen como chips (desde `data.alert_emails`).
- [ ] CA-006: Ante error de red al guardar, el estado local se revierte y se muestra toast de error.

### 5.2 No funcionales

- [ ] CA-NF-001: El bundle del frontend compila con `npm run build` en `frontend/` y `public/` queda regenerado (i18n servido por `http://localhost:8088/i18n/es.json`).

### 5.3 Testing

- **Unit tests**: Validación de formato de email y deduplicación (si se extrae a util).
- **Integration tests**: No aplican nuevos endpoints; flujo cubierto por pruebas manuales.
- **E2E tests**: Escenarios de usuario: agregar email válido, inválido, duplicado, eliminar chip, recargar página.
- **Carga/Performance**: Sin métricas nuevas; cambio no afecta rendimiento del iHost.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Crear modal de agregar email + refactor de `alertEmails` a `string[]` + chips con X + guardado automático | 0.5 día | Ninguna |
| 2 | i18n (es/en en `frontend/public/i18n/`), `npm run build`, verificación con `curl` | 0.25 día | Fase 1 |

### 6.2 Milestones

1. **MVP**: Chips + modal + validación + guardado automático + i18n funcionando en local.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Guardados concurrentes al eliminar rápido | Media | Bajo | Estado `savingEmails` que deshabilita acciones durante el request (REQ-008) |
| Perder claves i18n editadas en `public/i18n/` (no fuente) | Baja | Medio | Editar SOLO en `frontend/public/i18n/` (regla AGENTS.md), build y verificación con `curl` |
| Parseo de listas previas mal formadas | Baja | Bajo | Reutilizar `parseEmails` del backend / sanitización existente |

## 8. Notas y Referencias

- Patrón de modal: `frontend/src/components/AddInsuranceModal.tsx`, `IconPickerModal.tsx`
- Regla i18n: AGENTS.md (fuente de verdad `frontend/public/i18n/`)
- Regla de UI: AGENTS.md (patrón cards + modales, nunca formularios inline en listados)

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-16 | paulomcnally | Creación inicial de la especificación |
| 2026-08-16 | paulomcnally | Procesado: transición de draft → in_progress para iniciar desarrollo. |
| 2026-08-16 | paulomcnally | Cambio iterativo (usuario): modal debe cerrarse tras agregar con éxito y NO cerrarse al hacer click fuera (REQ-010). |
| 2026-08-16 | paulomcnally | Released. Commit mergeado a `main` desde `feature/SPEC-035`. Issue #35 cerrado. |
