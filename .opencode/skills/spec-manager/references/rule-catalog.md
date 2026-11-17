# Catálogo de reglas base

Reglas genéricas que `init` escribe en `docs/rules/`. Cada proyecto las ajusta: puede renombrar el componente concreto, eliminar las que no apliquen o agregar las propias (manteniendo el ID).

> Este catálogo es la **base**; la fuente de verdad es siempre el archivo del proyecto en `docs/rules/`.

---

## Arquitectura (`ARCH-`)

| ID | Regla | Verificación |
|---|---|---|
| `ARCH-STACK` | Stack obligatorio y prohibición de dependencias sin justificar | Revisión en spec |
| `ARCH-FOLDERS` | Estructura de carpetas declarada | Revisión |
| `ARCH-LAYERS` | Separación de capas (transporte / negocio / datos) | Revisión + tests |
| `ARCH-CONV` | Formateo y convenciones de código | Guardrail de format/lint |
| `ARCH-CONFIG` | Configuración de agentes y reglas | Revisión |

## UI (`UI-`)

| ID | Regla | Verificación |
|---|---|---|
| `UI-COMPONENT-REUSE` | Reutilizar componentes existentes; no duplicar | Revisión |
| `UI-SELECT` | Selector oficial, prohibido `<select>` nativo | Guardrail de build |
| `UI-TOKENS` | Tokens de tema, sin colores hardcodeados | Revisión + darkmode |
| `UI-DARKMODE` | Verificación en darkmode | Manual |
| `UI-LIST` | Patrón de listado + estado vacío | Revisión |
| `UI-CREATE` | Acción de crear en header con componente oficial | Revisión |
| `UI-EDIT-LABEL` | Acción «Editar» con clave genérica `app.edit` | grep de i18n |
| `UI-I18N` | Fuente de verdad única de traducciones | Revisión + build |

## UX (`UX-`)

| ID | Regla | Verificación |
|---|---|---|
| `UX-BACK` | Flecha atrás en header, sin links «← Título» | Revisión + mapa de rutas |
| `UX-MODAL` | Sin cierre por click fuera, overlay bloqueante, escala z, cierre visible | Manual |
| `UX-MODAL-CONFIRM` | Confirmación por palabra con foco y Enter | Manual |
| `UX-DETAIL` | Detalle de solo lectura; editar en ruta separada | Revisión |
| `UX-PREREQ` | Prerrequisitos validados en backend y frontend | Tests + revisión |
| `UX-EMPTY` | Estado vacío con acción | Revisión |
| `UX-LOADING` | Estado de carga visible | Revisión |
| `UX-FEEDBACK` | Feedback de éxito/error | Revisión |
| `UX-FORMS` | Formulario con acción primaria/secundaria y validación | Revisión |

## Datos (`DATA-`)

| ID | Regla | Verificación |
|---|---|---|
| `DATA-DB` | Configuración de la base | Revisión |
| `DATA-MIGRATIONS` | Migraciones versionadas y reversibles | Revisión |
| `DATA-NO-DROP` | Nunca borrar la base del usuario | Revisión de comandos |
| `DATA-PRODCOPY` | Validar contra copia de producción en `/tmp` | Manual |
| `DATA-PROD-WRITE` | Procedimiento de escritura en producción | Runbook |
| `DATA-SCHEMA` | Esquema protegido | Revisión |

## Seguridad (`SEC-`)

| ID | Regla | Verificación |
|---|---|---|
| `SEC-NO-PII` | Sin PII, secretos ni volcados | Revisión de `git diff --cached` |
| `SEC-SYNTHETIC` | Datos sintéticos obligatorios | Revisión |
| `SEC-COMMIT` | Higiene previa al commit | Checklist |
| `SEC-INCIDENT` | Protocolo de incidente | Procedimiento |
| `SEC-LOGS` | Logs y capturas sanitizados | Revisión |

## Testing (`TEST-`)

| ID | Regla | Verificación |
|---|---|---|
| `TEST-LOCAL` | Pruebas locales obligatorias | Guardrail build/test |
| `TEST-SERVER` | Verificar servicio antes de avisar | Health check |
| `TEST-USER-FLOW` | Validación explícita del usuario | Flujo 🤖 |
| `TEST-CHECKLIST` | Checklist pre-release | Gate |

## Delivery (`DELIV-`)

| ID | Regla | Verificación |
|---|---|---|
| `DELIV-BRANCH` | Estrategia de ramas y aislamiento | Script del proyecto |
| `DELIV-CONFIRM` | Merge/push con confirmación; sin `--force` | Revisión |
| `DELIV-COMMIT` | Formato de commits | Revisión |
| `DELIV-RELEASE` | Pasos completos de release | Gate |
| `DELIV-LABELS` | Labels válidos | `gh` |
| `DELIV-DOCKER` | Multi-arquitectura (si aplica) | `buildx imagetools` |
| `DELIV-ESCALATION` | Escalación por bloqueo | Proceso |
