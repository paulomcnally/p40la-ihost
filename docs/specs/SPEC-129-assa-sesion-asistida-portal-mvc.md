---
title: "ASSA: sesión asistida y parsers del portal MVC nuevo"
id: "SPEC-129"
status: "released"
author: "opencode"
created: "2026-11-08"
updated: "2026-11-08"
github_issue: 132
---

# ASSA: sesión asistida y parsers del portal MVC nuevo

**ID**: SPEC-129  
**Estado**: released  
**Autor**: opencode  
**Creado**: 2026-11-08  
**Actualizado**: 2026-11-08

---

## 1. Resumen Ejecutivo

La SPEC-121 liberó el MVP P0 de diagnóstico: el plugin `assa.nicaragua` v1.1.1 ya distingue `ErrPortalMigrated` de `ErrCaptchaRequired`, evita falsos positivos con `var` y la app muestra un mensaje accionable. Eso no recupera los datos: el login legacy ya no funciona y el portal MVC nuevo exige reCAPTCHA.

**Hallazgo del spike (2026-11-08):** una cookie de sesión del portal MVC nuevo permite acceder a las páginas SCA existentes (`Consulta_Polizas.aspx` y `Unidades.aspx`) sin el token `var`. Es decir, no hace falta mapear rutas MVC nuevas ni escribir parsers nuevos: alcanza con agregar `session_cookie`, usarla en el cliente HTTP y reutilizar los parsers actuales. Sin cookie, las mismas páginas devuelven `302` a `login_v.aspx?error=1`.

Esta spec implementa el campo secreto `session_cookie`, el modo cookie sin login legacy, la detección de expiración (`ErrAuthExpired`), la UI de renovación y el release del plugin `v1.2.0`. Sin una sesión real aportada por el usuario no se escribe scraping (REQ-005/REQ-008/CA-008 de SPEC-121).

La spec no agrega navegador, dependencias pesadas ni cambios de esquema SQLite. La cookie reutiliza `automation_accounts.credentials`, se enmascara en la UI y nunca se registra en logs, errores, `raw`, tests ni commits.

## 2. Requerimientos

### 2.1 P0 - Obligatorios

1. **REQ-001 — Validar la sesión real.** Con una cookie provista por el usuario (nunca commiteada), verificar que `Consulta_Polizas.aspx` y `Unidades.aspx` acepten la cookie sin `var` y devuelvan los grids. Sin cookie, ambas deben redirigir a `login_v.aspx?error=1`.
2. **REQ-002 — Confirmar el reuso de rutas SCA.** Documentar que las páginas SCA existentes siguen operativas con la cookie MVC y que no hace falta mapear rutas MVC nuevas. Si el portal deja de aceptar la cookie en esas rutas, detener la implementación y volver a la opción MVC/API oficial.
3. **REQ-003 — Campo `session_cookie`.** Agregar al `CredentialSchema` de ASSA un campo opcional `session_cookie`, `Type: "password"`, `Secret: true`, con una descripción de cómo renovarlo.
4. **REQ-004 — Uso y expiración.** Si `session_cookie` está presente, el plugin debe saltar el login legacy, aplicar la cookie al jar, consultar las rutas SCA sin `var` y devolver `ErrAuthExpired` cuando el portal redirija al login.
5. **REQ-005 — Reusar parsers.** Mantener los parsers actuales de pólizas, polifacturas y recibos; agregar tests con fixtures sintéticos para el modo cookie. No incluir datos reales en tests ni specs.
6. **REQ-006 — UI de renovación.** Mostrar la descripción del campo `session_cookie` en el formulario y permitir pegar una cookie nueva sin recrear la cuenta. Inputs con `bg-card`/`text-text`, darkmode verificado y sin `<select>` nativo.
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
        |        +--> aplicar cookies al jar
        |        +--> GET Consulta_Polizas.aspx (sin var)
        |        |        |
        |        |        +--> 302 login_v.aspx?error=1 --> ErrAuthExpired
        |        |        +--> 200 con grid --> parser existente
        |        |
        |        +--> GET Unidades.aspx?pol=&ipol= (sin var) --> parsers existentes
        |
        +--> sin cookie --> flujo P0 (ErrPortalMigrated / ErrCaptchaRequired)
```

### 3.2 Componentes

- **Plugin privado** (`assa/nicaragua/nicaragua.go`): credencial `session_cookie`, carga de cookies en el jar, consultas SCA sin `var`, detección de expiración y reuso de parsers.
- **Contrato** (`plugins/types.go`): `CredentialField.Description` opcional para explicar la renovación en la UI.
- **App**: sin endpoints nuevos; `GET /api/automation/plugins/{name}/schema` expone el campo y `PUT /api/automation/accounts/{id}` conserva credenciales si el campo viene vacío.
- **Frontend**: `AutomationFormPage.tsx` renderiza la descripción del campo; `AutomationDetailPage.tsx` ya muestra el estado de sesión expirada.
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

- [x] **CA-001**: Una cookie válida obtenida manualmente permite leer y entregar pólizas, polifacturas y recibos al servicio reutilizando las rutas SCA. Validado con la cuenta real: 2 pólizas, 18 facturas/recibos; el endpoint de fetch devolvió 6 facturas para la póliza demo.
- [x] **CA-002**: Una cookie expirada devuelve `ErrAuthExpired` y la UI muestra el mensaje de renovación. Verificado con `TestFetchBills_SessionCookieExpired` y el mapeo existente de `auth_expired`.
- [x] **CA-003**: El campo `session_cookie` aparece en el schema, es opcional y secreto; no se muestra en listados ni `raw`. Verificado con `TestCredentialSchema` y `GET /api/automation/plugins/assa.nicaragua/schema`.
- [x] **CA-004**: El modo cookie tiene tests con fixtures sintéticos y no incluye datos reales; los parsers existentes siguen pasando (`go test ./...` en el módulo de plugins).
- [x] **CA-005**: La cookie no aparece en logs, errores, tests, `automation_bills.raw` ni commits. Tests de no filtración y revisión de diff.
- [x] **CA-006**: El formulario muestra la descripción de renovación (`CredentialField.Description`) y el input usa `bg-card`/`text-text`; build de frontend OK.
- [x] **CA-007**: La imagen se publica con los tres manifests (`linux/amd64`, `linux/arm/v7`, `linux/arm64`). Verificado en el release.
- [x] **CA-008**: La validación con datos reales se hizo solo con GET de lectura; la cookie se borró de `/tmp` al terminar.

## 5. Plan de Implementación

| Fase | Descripción | Estimación | Dependencia |
|---|---|---|---|
| 1 | Spike con sesión real: cookie, acceso a SCA sin `var`, expiración | 0.5 día | Cookie del usuario (hecho) |
| 2 | Plugin `v1.2.0`: `session_cookie`, jar, consultas sin `var`, `ErrAuthExpired` | 1–2 días | Fase 1 go |
| 3 | UI de renovación (descripción del campo), tests de no exposición y modo cookie | 0.5 día | Fase 2 |
| 4 | Release: tag `v1.2.0`, `go.mod`, imagen multi-arch y verificación | 0.5 día | Fase 3 |

Si el portal deja de aceptar la cookie en las rutas SCA, se actualiza esta spec y se vuelve a la opción MVC/API oficial antes de escribir código.

## 6. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| La sesión dura muy poco | Media | Alto | Medir en Fase 1; si es < 24 h, documentar y evaluar renovación asistida |
| El portal deja de aceptar la cookie en SCA | Media | Alto | Detectar `302` a login y volver a MVC/API oficial |
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
| 2026-11-08 | opencode | Spike con sesión real: la cookie MVC permite acceder a `Consulta_Polizas.aspx` y `Unidades.aspx` sin `var`; se reutilizan los parsers existentes. Se descarta mapear rutas MVC nuevas y se enfoca la implementación en `session_cookie`. |
| 2026-11-08 | opencode | Release: plugin `v1.2.0` con `session_cookie`, modo cookie sin login legacy y `ErrAuthExpired`; app con descripción del campo y `go.mod` v1.2.0. Validación real read-only: 2 pólizas, 18 facturas/recibos, fetch de 6 facturas; imagen `v0.5.6` con tres manifests. |
