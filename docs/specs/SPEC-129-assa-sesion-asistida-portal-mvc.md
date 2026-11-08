---
title: "ASSA: sesión asistida y parsers del portal MVC nuevo"
id: "SPEC-129"
status: "draft"
author: "opencode"
created: "2026-11-08"
updated: "2026-11-08"
github_issue: 132
---

# ASSA: sesión asistida y parsers del portal MVC nuevo

**ID**: SPEC-129  
**Estado**: draft  
**Autor**: opencode  
**Creado**: 2026-11-08  
**Actualizado**: 2026-11-08

---

## 1. Resumen Ejecutivo

La SPEC-121 liberó el MVP P0 de diagnóstico: el plugin `assa.nicaragua` v1.1.1 ya distingue `ErrPortalMigrated` de `ErrCaptchaRequired`, evita falsos positivos con `var` y la app muestra un mensaje accionable. Eso no recupera los datos: el portal MVC nuevo exige reCAPTCHA y todavía no existe una sesión asistida ni parsers de las rutas autenticadas.

Esta spec cubre la Fase 2 y 3 que quedaron bloqueadas en SPEC-121: validar con una cuenta real una cookie de sesión, mapear las rutas autenticadas de pólizas/polifacturas/recibos, implementar el campo secreto `session_cookie`, detectar expiración y publicar el plugin `v1.2.0`. Sin una sesión real aportada por el usuario no se escribe scraping (REQ-005/REQ-008/CA-008 de SPEC-121).

La spec no agrega navegador, dependencias pesadas ni cambios de esquema SQLite. La cookie reutiliza `automation_accounts.credentials`, se enmascara en la UI y nunca se registra en logs, errores, `raw`, tests ni commits.

## 2. Requerimientos

### 2.1 P0 - Obligatorios

1. **REQ-001 — Validar la sesión real.** Con una cookie provista por el usuario (nunca commiteada), verificar que el portal MVC nuevo permite leer pólizas, polifacturas pendientes y recibos pagados. Medir la duración de la sesión y documentar el resultado en la spec.
2. **REQ-002 — Mapear rutas autenticadas.** Identificar las rutas y el HTML/JSON de las tres consultas. Si el mapeo no es viable o la sesión dura menos de 24 h, detener la implementación y volver a la API oficial o a la carga manual (SPEC-121 opciones E/F).
3. **REQ-003 — Campo `session_cookie`.** Agregar al `CredentialSchema` de ASSA un campo opcional `session_cookie`, `Type: "password"`, `Secret: true`, sin volver obligatorios `username`/`password`.
4. **REQ-004 — Uso y expiración.** Si `session_cookie` está presente, el plugin debe saltar el login legacy, validar la cookie contra una ruta autenticada, leer los datos y devolver `ErrAuthExpired` cuando el portal redirija al login.
5. **REQ-005 — Parsers tolerantes.** Construir parsers con fixtures sintéticos y sanitizados para pólizas, polifacturas y recibos. No incluir datos reales en tests ni specs.
6. **REQ-006 — UI de renovación.** Explicar en el formulario cómo renovar la cookie y permitir pegar una nueva sin recrear la cuenta. Inputs con `bg-card`/`text-text`, darkmode verificado y sin `<select>` nativo.
7. **REQ-007 — Seguridad.** La cookie no se devuelve en listados, no se guarda en `automation_bills.raw`, no aparece en logs/errores/tests/commits y el `reveal` sigue exigiendo la contraseña de la app.
8. **REQ-008 — Tests.** Cookie válida, cookie expirada, redirect al login, parsers con fixtures sintéticos y ausencia de secretos en errores/logs.
9. **REQ-009 — Versionado.** Publicar el plugin como `v1.2.0`, actualizar `go.mod`/`go.sum`, reconstruir la imagen multi-arch y verificar `automation_plugins.version` y `automation_bills.plugin_version`.

### 2.2 P1 - Importantes

1. **REQ-010 — Runbook.** Documentar login manual, extracción segura de la cookie, pegado, validación y renovación, sin valores reales.
2. **REQ-011 — Alertas.** Avisar en la página de automation cuando una cuenta acumule fallos de sesión y ofrecer la renovación.
3. **REQ-012 — Semántica de facturas.** Evaluar el upsert por `(service_id, year, month)` antes de reactivar ASSA; si requiere cambios, abrir una spec separada.

### 2.3 No funcionales

- Sin navegador headless ni solver de CAPTCHA.
- Timeout de 15 s por request; sin reintentos dentro del mismo job.
- Sin cambios de esquema SQLite.
- Compatibilidad con el contrato `plugins.Plugin` y los demás plugins.
- Validación con datos reales contra una copia de la DB de producción, nunca contra `data/app.db`.

## 3. Diseño Técnico

### 3.1 Flujo

```
[AutomationService.FetchBills]
        |
        v
[assa.nicaragua v1.2.0]
        |
        +--> session_cookie presente?
        |        |
        |        +--> GET ruta autenticada --> 302 login --> ErrAuthExpired
        |        +--> GET pólizas/facturas/recibos --> parsers MVC
        |
        +--> sin cookie --> flujo P0 (ErrPortalMigrated / ErrCaptchaRequired)
```

### 3.2 Componentes

- **Plugin privado** (`assa/nicaragua/nicaragua.go`): credencial `session_cookie`, validación de sesión, parsers nuevos y errores existentes.
- **App**: sin endpoints nuevos; `GET /api/automation/plugins/{name}/schema` expone el campo y `PUT /api/automation/accounts/{id}` conserva credenciales si el campo viene vacío.
- **Frontend**: `AutomationFormPage.tsx` genera el campo desde el schema; `AutomationDetailPage.tsx` muestra el estado y la acción de renovación.
- **Base de datos**: `automation_accounts.credentials` guarda el JSON; sin migraciones.

### 3.3 Credenciales (ejemplo sintético)

```json
{
  "username": "demo-user",
  "password": "demo-pass",
  "session_cookie": "demo-cookie"
}
```

## 4. Criterios de Aceptación

- [ ] **CA-001**: Una cookie válida obtenida manualmente permite leer y entregar pólizas, polifacturas y recibos al servicio.
- [ ] **CA-002**: Una cookie expirada devuelve `ErrAuthExpired` y la UI muestra el mensaje de renovación.
- [ ] **CA-003**: El campo `session_cookie` aparece en el schema, es opcional y secreto; no se muestra en listados ni `raw`.
- [ ] **CA-004**: Los parsers nuevos tienen tests con fixtures sintéticos y no incluyen datos reales.
- [ ] **CA-005**: La cookie no aparece en logs, errores, tests, `automation_bills.raw` ni commits.
- [ ] **CA-006**: El formulario se verifica en darkmode y usa los tokens del tema.
- [ ] **CA-007**: La imagen se publica con los tres manifests (`linux/amd64`, `linux/arm/v7`, `linux/arm64`).
- [ ] **CA-008**: La validación con datos reales se hace sobre una copia de producción; `PRAGMA integrity_check` y `foreign_key_check` quedan OK.

## 5. Plan de Implementación

| Fase | Descripción | Estimación | Dependencia |
|---|---|---|---|
| 1 | Spike con sesión real: login manual, cookie, duración y rutas | 0.5–1 día | Cookie del usuario |
| 2 | Plugin `v1.2.0`: `session_cookie`, validación, expiración y parsers | 2–4 días | Fase 1 go |
| 3 | UI de renovación y tests de no exposición | 0.5–1 día | Fase 2 |
| 4 | Release: tag `v1.2.0`, `go.mod`, imagen multi-arch y verificación | 0.5 día | Fase 3 |

Si la Fase 1 da no-go, se elige la API oficial de ASSA o la carga manual/PDF (SPEC-121) y se actualiza esta spec antes de escribir código.

## 6. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| La sesión dura muy poco | Media | Alto | Medir en Fase 1; si es < 24 h, no implementar y volver a API oficial/manual |
| El portal cambia otra vez | Alta | Alto | Parsers tolerantes, fixtures, detección de errores y versionado |
| Fuga de la cookie | Baja | Alto | Campo secreto, `reveal` con contraseña, tests y revisión de diff |
| El upsert por mes colapsa facturas | Media | Medio | Evaluar REQ-012 antes de reactivar ASSA |

## 7. Notas y Referencias

- SPEC-121: MVP P0 liberado (`v1.1.1`), migración y CAPTCHA detectados.
- Portal nuevo: `https://miassanet.assanet.com/ni/MiAssan-NI/login`.
- Contrato: `plugins.Plugin`, `plugins.CredentialField`.
- App: `internal/services/automation_sync.go`, `frontend/src/pages/AutomationFormPage.tsx`.
- Sin datos reales: usar `demo-user`, `demo-pass`, `demo-cookie`, `demo-policy-001`.

## 8. Historial de Cambios

| Fecha | Autor | Descripción |
|---|---|---|
| 2026-11-08 | opencode | Creación como seguimiento de SPEC-121. Se documenta la Fase 2/3 pendiente: sesión asistida, parsers del portal MVC y plugin `v1.2.0`. |
