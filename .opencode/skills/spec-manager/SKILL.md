---
name: spec-manager
description: "Crea, actualiza y libera especificaciones técnicas (specs) con cumplimiento verificado de las reglas del proyecto. Úsala cuando el usuario pida crear/gestionar una spec, definir requerimientos, cambiar estados (draft/pending_execution/in_progress/pending_release/released), sincronizar con GitHub Issues, inicializar la estructura de reglas de un proyecto nuevo, o auditar qué reglas aplican a un cambio. Triggers: spec, especificación, requerimiento, tracking, contador, estado de spec, pending release, init rules."
---

# Spec Manager

Sistema de especificaciones con **reglas del proyecto como contrato**.

## Principios

1. **La spec primero.** No se escribe código sin una spec que lo defina.
2. **Una regla = un ID + una fuente canónica + un guardrail.** Esta skill **no copia reglas**: las referencia.
3. **Las reglas del proyecto ganan** sobre esta skill, sobre los templates y sobre la memoria del agente.
4. **Falla ruidosa.** Si faltan las reglas, se detiene y se ofrece `init`; nunca se opera a ciegas.
5. **Portable.** Nada de rutas, componentes ni tecnologías de un proyecto específico fuera de ejemplos marcados.

---

## PASO 0 — Rules Bootstrap (OBLIGATORIO)

Ejecutar **antes de cualquier comando o de escribir una spec**.

1. **Localizar** en la raíz del proyecto/worktree actual:
   - `AGENTS.md`
   - `docs/rules/00-index.md`

   Si el script existe, verificar de forma determinista:
   ```bash
   <skill-dir>/scripts/check-rules.sh <directorio>
   ```
2. **Leer el manifiesto** (`docs/rules/00-index.md`) y extraer:
   - perfil del proyecto (stack, rutas clave);
   - tabla de dominios (qué archivo cargar según el tipo de cambio);
   - guardrails ejecutables;
   - reglas no negociables.
3. **Cargar los dominios aplicables** (no hace falta cargar todo):

   | Tipo de cambio | Dominios |
   |---|---|
   | UI / frontend / formularios | `20-ui`, `30-ux` |
   | Rutas, pantallas, modales | `30-ux` |
   | DB, migraciones, datos reales | `40-data` |
   | Backend, arquitectura, dependencias | `10-architecture` |
   | Deploy, git, release | `70-delivery` |
   | **Siempre** | `50-security` + `60-testing` |

4. **Verificar** que existan los archivos que el manifiesto declara. Si falta alguno:

   ```
   ⛔ DETENIDO — Estructura de reglas incompleta
   Faltan: <rutas>
   No puedo crear ni modificar specs sin las reglas del proyecto.
   Opciones:
     1) /spec-init        → generar la estructura faltante (no destructivo)
     2) Indicarme la ruta de las reglas existentes para declararlas en el manifiesto
   ```

   No continuar hasta resolverlo. No inventar reglas ni asumir las de otro proyecto.

5. **Si no existe ninguna estructura** (`AGENTS.md` ni `docs/rules/`), asumir proyecto nuevo y ofrecer `/spec-init`.

> OpenCode ignora en silencio los archivos de reglas faltantes: esta verificación explícita es la única garantía.

---

## Contrato de estructura

Todo proyecto que use esta skill debe tener:

```
AGENTS.md                          # entrada auto-cargada + no negociables + índice
docs/rules/00-index.md             # manifiesto: perfil, dominios, guardrails, IDs
docs/rules/10-architecture.md      # stack, capas, carpetas, convenciones
docs/rules/20-ui.md                # componentes, tokens, patrones visuales
docs/rules/30-ux.md                # navegación, modales, detalle, estados, flujos
docs/rules/40-data.md              # datos, migraciones, copias de producción
docs/rules/50-security.md          # PII, secretos, datos sintéticos
docs/rules/60-testing.md           # pruebas, verificación, flujo con el usuario
docs/rules/70-delivery.md          # git, worktrees, release, deploy
.githooks/pre-commit               # enforcement en el commit (recomendado)
.githooks/commit-msg               # formato de mensaje (recomendado)
scripts/install-hooks.sh           # instalación idempotente
```

- Los **nombres de archivo son parte del contrato** (el manifiesto puede declarar variantes).
- `AGENTS.md` debe apuntar a `docs/rules/` y contener las reglas de seguridad no negociables.
- `opencode.json` debería declarar `instructions: ["AGENTS.md", "docs/rules/*.md"]`.

Templates listos para generar: `references/rules/` y `references/rules-index-template.md`.

---

## Enforcement (hooks)

Las reglas no se sostienen solo con instrucciones: el proyecto debe tener prevención automática.

| Capa | Archivo | Qué bloquea |
|---|---|---|
| Git hook `pre-commit` | `.githooks/pre-commit` | Archivos prohibidos (DBs, PDFs, backups), secretos/PII, i18n generado, guardrail de UI |
| Git hook `commit-msg` | `.githooks/commit-msg` | Mensajes fuera del formato del proyecto |
| Plugin de OpenCode | `.opencode/plugins/rules-guard/` | Comandos destructivos y escrituras prohibidas del agente |

Reglas:
- **Fail-safe**: si el hook o el plugin falla internamente, la operación continúa. Solo los patrones positivos bloquean.
- **Prohibido** `--no-verify` salvo autorización explícita del usuario.
- `init` genera los hooks; `scripts/install-hooks.sh` los activa (`git config core.hooksPath .githooks`).
- Si el proyecto no tiene hooks, el gate de pre-release lo reporta como advertencia (no bloqueante).
- Los hooks **complementan** la revisión: no reemplazan la tabla de cumplimiento ni la validación del usuario.

---

## IDs de regla y de criterios

| Prefijo | Dominio |
|---|---|
| `ARCH-` | arquitectura y stack |
| `UI-` | interfaz y componentes |
| `UX-` | experiencia, navegación, modales |
| `DATA-` | datos y migraciones |
| `SEC-` | seguridad y privacidad |
| `TEST-` | pruebas y verificación |
| `DELIV-` | entrega, git y deploy |

- Regla: `UI-SELECT`, `UX-MODAL`, `DATA-PRODCOPY`.
- Criterio de aceptación: `CA-UI-SELECT`, `CA-UX-MODAL`, `CA-DATA-PRODCOPY`.
- **Obligatorio**: toda spec incluye una sección **«Cumplimiento de reglas»**:

  ```markdown
  ## Cumplimiento de reglas

  | Regla | Cómo la cumple la spec | Criterio |
  |---|---|---|
  | UI-SELECT | El selector de moneda usa el componente Select, searchable | CA-UI-SELECT |
  | UX-MODAL | El modal no cierra por click fuera; z-[60]; cierre visible | CA-UX-MODAL |
  | SEC-SYNTHETIC | Ejemplos con datos `demo-*` | CA-SEC-SYNTHETIC |
  | UX-BACK | N/A: la spec no agrega rutas con lista padre | — |
  ```

- `N/A` solo con justificación.
- El ID de regla debe existir en `docs/rules/`. Si no existe y aplica: crear la regla antes de la spec.

---

## Comandos

| Comando | Descripción |
|---|---|
| `/spec-init` | Genera la estructura de reglas faltante (no destructivo) |
| `/spec-create "<título>"` | Crea spec + issue con ID secuencial y tabla de cumplimiento |
| `/spec-list [estado]` | Lista specs locales, filtradas por estado |
| `/spec-status <id> <estado>` | Cambia estado con validación de transición |
| `/spec-show <id>` | Resumen de una spec |
| `/spec-worktree <id>` | Crea worktree aislado (si el proyecto usa worktrees) |
| `/spec-check-release` | Detecta specs colgadas (código en main sin release) |
| `/spec-audit <id>` | Audita una spec contra las reglas del manifiesto |

> Si los comandos de OpenCode del proyecto usan otro nombre (`/spec-create` vs `/spec create`), usar el registrado en `opencode.json`. No inventar comandos.

---

## 1. `/spec-init`

Crea la estructura de reglas en un proyecto nuevo o incompleto.

### Procedimiento

1. Ejecutar Paso 0 en modo **diagnóstico** (sin abortar).
2. Detectar qué falta de la lista del contrato.
3. **Nunca sobrescribir**: si un archivo existe, omitirlo y reportarlo.
4. Generar con el script determinista:
   ```bash
   <skill-dir>/scripts/init-rules.sh <directorio-del-proyecto>
   ```
   El script copia desde `references/`:
   - `AGENTS.md` ← `agents-template.md`
   - `docs/rules/00-index.md` ← `rules-index-template.md`
   - `docs/rules/{10,20,30,40,50,60,70}-*.md` ← `rules/`
   - `.githooks/{pre-commit,commit-msg}` ← `hooks/`
   - `scripts/install-hooks.sh` ← `hooks/install-hooks.sh`
   Es idempotente y no destructivo. Si no se puede ejecutar (permisos, Windows), copiar manualmente respetando el mismo criterio de no-sobrescritura.
5. Si existe un `AGENTS.md` o `project-rules.md` monolítico previo:
   - **no modificarlo**;
   - ofrecer `--migrate` para mapear secciones a dominios (solo propuesta, requiere confirmación).
6. Actualizar `opencode.json` **solo con confirmación**: `instructions` con `AGENTS.md` y `docs/rules/*.md`.
7. Reportar:

   ```
   Estructura de reglas
   ✓ creados:   docs/rules/00-index.md, docs/rules/10-architecture.md, ...
   = omitidos:  AGENTS.md (ya existía)
   ⚠ pendiente: completar la tabla de dominios del manifiesto
   ```

### Reglas de `init`

- No crea specs, código, ramas ni issues.
- No borra ni renombra archivos.
- Es idempotente: correrlo dos veces no cambia nada.
- No asume stack: los templates usan placeholders `{{...}}` que el proyecto completa.
- **Anti-drift**: si detecta el mismo enunciado de regla en más de un archivo, advertirlo y proponer dejar una sola copia canónica (con ID) y referencias en el resto.

---

## 2. `/spec-create "<título>"`

1. **Paso 0** (obligatorio). Si falta estructura → abortar y ofrecer `init`.
2. **Worktree** (si el proyecto lo exige): ejecutar el script de verificación declarado en el manifiesto. Si aborta, no trabajar en ese checkout.
3. **Investigar** el requerimiento (web/docs si aplica).
4. **Calcular el ID**: máximo `SPEC-(\d{3})` entre `docs/specs/SPEC-*.md` **y** los issues abiertos/cerrados con título `SPEC-XXX`. Nunca desde el README tracker (se actualiza solo en release).
5. **Impacto en tools MCP** (si el proyecto tiene manifiesto, p. ej. `docs/mcp/tools/impact.json`):
   - delegar a un subagente (`explore` o `general`) el análisis de qué tools existentes pueden romperse;
   - correr `./scripts/check-mcp-impact.sh --files <archivos previstos>` si existe;
   - completar `## Impacto en tools MCP` con tools afectadas, ruta de dependencia, contrato y `Validación subagente:` (agente, sesión, veredicto).
6. **Crear el archivo** `docs/specs/SPEC-XXX-<slug>.md` con el template del proyecto (o `references/spec-template.md`).
7. **Completar** las secciones obligatorias (ver abajo), incluida **Cumplimiento de reglas** e **Impacto en tools MCP**.
8. **Crear el issue** con label `spec/draft` y guardar `github_issue` en el frontmatter.
9. **Confirmar** al usuario: ID, ruta y URL del issue.

### Secciones obligatorias de una spec

- Frontmatter: `title`, `id`, `status`, `author`, `created`, `updated`, `github_issue`.
- Resumen ejecutivo (problema, por qué ahora, resultado esperado).
- Requerimientos priorizados P0/P1/P2.
- **Cumplimiento de reglas** (tabla con IDs).
- **Impacto en tools MCP** (si el proyecto tiene tools MCP): tools afectadas o «ninguna», ruta de dependencia, contrato y `Validación subagente:`.
- Investigación y decisiones (ADRs cuando hay trade-offs).
- Diseño técnico (componentes, APIs, datos, archivos).
- Criterios de aceptación con IDs canónicos (`CA-<DOMINIO>-<NOMBRE>`).
- Plan de implementación.
- Riesgos y mitigaciones.
- Historial de cambios.

---

## 3. `/spec-status <id> <estado>`

1. Validar la transición (ver diagrama).
2. Leer `github_issue` del frontmatter.
3. Actualizar GitHub: `gh issue edit <n> --remove-label "spec/<anterior>" --add-label "spec/<nuevo>"`.
   - Si el nuevo estado es `released`: cerrar el issue **después** de actualizar el label.
   - Si `gh` falla: continuar con warning visible.
4. Actualizar `status` en el frontmatter **y** en el cuerpo (`**Estado**:`).
5. Agregar entrada al historial.
6. **No** actualizar el README tracker en estados intermedios (solo en release, si el proyecto lo define así).
7. Confirmar al usuario.

### Transiciones válidas

```
draft            → pending_execution | cancelled
pending_execution→ in_progress | cancelled
in_progress      → pending_release | pending_execution
pending_release  → released | in_progress
released         → in_progress (hotfix)
cancelled        → terminal
```

**`cancelled` es decisión exclusiva del usuario** (palabras «cancelar»/«descartar»). «Cerrar» = `released`.

---

## 4. Gate de pre-release

Antes de pasar a `pending_release` o `released`, verificar y reportar:

1. **Guardrails del manifiesto**: ejecutar cada comando declarado y capturar el resultado. Si uno falla → **bloquear**.
2. **Impacto MCP**: si existe `docs/mcp/tools/impact.json`, correr `./scripts/check-mcp-impact.sh --staged --gate` y `go test ./internal/mcp/...`; verificar que la sección `## Impacto en tools MCP` cubra todas las tools afectadas y tenga `Validación subagente:`.
3. **Cobertura de reglas**: cada regla aplicable al cambio está en la tabla de cumplimiento con criterio o `N/A` justificado.
4. **Checklist del proyecto**: leer `docs/rules/60-testing.md` y confirmar cada ítem.
5. **Higiene de datos**: `git diff --cached` sin PII, secretos, DBs, PDFs ni exports.
6. **Validación del usuario**: no avanzar sin confirmación explícita de las pruebas manuales.

Salida del gate:

```
Gate de pre-release — SPEC-XXX
✓ guardrails: check:ui, build, test
✗ reglas sin cubrir: UX-BACK (falta CA o N/A)
⛔ BLOQUEADO: completar la tabla de cumplimiento antes de continuar
```

---

## 5. `/spec-check-release`

Detecta specs colgadas (implementación en main sin release).

1. Listar specs locales y su `status`.
2. Para cada spec ≠ `released`: buscar commits `git log --all --oneline --grep="SPEC-XXX" -i`.
3. Cruzar con GitHub: `gh issue list --label "spec/<estado>" --state open`.
4. Reportar: specs colgadas, issues sin spec local o viceversa, inconsistencias frontmatter ↔ labels.
5. Si hay colgadas: **no** declarar terminada la tarea.

---

## 6. Release

Una spec pasa a `released` solo con **todos** estos pasos en el mismo ciclo:

1. Merge a `main` **con confirmación explícita del usuario**.
2. Commit y push de todos los cambios pendientes (formato del proyecto, ej: `SPEC-XXX: descripción`).
3. Documentar commit/versión y fecha en la spec.
4. Actualizar `status` (frontmatter + cuerpo).
5. Actualizar el tracker del proyecto (si existe).
6. Cambiar label a `spec/released` y cerrar el issue.
7. Limpiar el worktree/rama de la spec (solo si está `released`).
8. Verificar con `/spec-check-release`.

**Implementación y release son inseparables**: si el código está en `main`, la spec debe estar `released`.

---

## 7. Integración con GitHub Issues

| Operación | Comando |
|---|---|
| Crear | `gh issue create --title "SPEC-XXX: Título" --label "spec/<estado>" --body "..."` |
| Cambiar label | `gh issue edit <n> --remove-label "spec/<anterior>" --add-label "spec/<nuevo>"` |
| Cerrar | `gh issue close <n>` |
| Reabrir | `gh issue reopen <n>` |
| Comentar | `gh issue comment <n> --body "..."` |
| Listar | `gh issue list --label "spec/<estado>" --state open` |

Labels válidos: `spec/draft`, `spec/pending-execution`, `spec/in-progress`, `spec/pending-release`, `spec/released`, `spec/cancelled`. Nunca labels genéricos.

Si `gh` no está disponible: continuar con warning y dejar constancia de la operación pendiente.

---

## Anti-patrones (prohibidos)

- Crear una spec sin haber leído `docs/rules/00-index.md`.
- Crear una spec que toca tools MCP sin la sección `## Impacto en tools MCP` o sin validación de un subagente.
- Copiar reglas dentro de la spec o de la skill (solo IDs + referencia).
- Inventar reglas cuando el proyecto no las tiene.
- Cerrar/liberar sin confirmación del usuario.
- Cancelar una spec sin orden explícita del usuario.
- Suponer que un archivo de reglas existe (OpenCode lo ignora en silencio).
- Declarar terminada una spec con guardrails fallando.

---

## Referencias incluidas

| Archivo | Uso |
|---|---|
| `references/agents-template.md` | `AGENTS.md` base para proyectos nuevos |
| `references/rules-index-template.md` | Manifiesto `docs/rules/00-index.md` |
| `references/rules/10-architecture.md` … `70-delivery.md` | Templates de dominio con reglas base genéricas |
| `references/spec-template.md` | Template de spec con «Cumplimiento de reglas» |
| `references/rule-catalog.md` | Catálogo de IDs de regla base y guardrails sugeridos |
| `references/hooks/pre-commit` | Hook de commit con guardrails |
| `references/hooks/commit-msg` | Hook de formato de mensaje |
| `references/hooks/install-hooks.sh` | Instalador idempotente de hooks |
| `scripts/check-rules.sh` | Verifica la estructura de reglas (Paso 0) |
| `scripts/init-rules.sh` | Genera la estructura faltante sin sobrescribir |
