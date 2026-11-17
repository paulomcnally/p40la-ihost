# 00 — Índice y manifiesto de reglas

> **Proyecto**: p40la-ihost
> **Última actualización**: 2026-11-17
> **Sistema**: reglas por dominio + IDs estables (SPEC-136)

Este archivo es el **punto de entrada** de las reglas del proyecto. La skill `spec-manager` lo lee primero (Paso 0) y desde acá decide qué dominios cargar.

---

## 1. Precedencia

1. Las **reglas de este repositorio** ganan sobre cualquier skill, template o instrucción genérica.
2. Si hay conflicto entre archivos de reglas, gana el archivo del dominio específico sobre este índice.
3. Si hay conflicto entre una spec y una regla, se actualiza la spec **antes** de escribir código.
4. Si una regla no existe y el cambio la necesita: se crea la regla (con ID) y después se implementa.

---

## 2. Perfil del proyecto

| Campo | Valor |
|---|---|
| Nombre | p40la-ihost |
| Tipo | Aplicación web autoalojada (add-on Docker) |
| Runtime | Go + SQLite, servido en un SONOFF iHost (ARM64, recursos limitados) |
| Frontend | React + Tailwind + Vite (build fuera del iHost) |
| Backend | `cmd/server` + `internal/{api,config,db,models,services,storage}` |
| Base de datos | SQLite con `journal_mode=WAL` |
| Frontend estático | `public/` (salida del build de `frontend/`) |
| i18n | `frontend/public/i18n/{es,en}.json` (fuente de verdad) |
| Specs | `docs/specs/` + GitHub Issues con labels `spec/*` |
| Puerto | `8088` (configurable con `PORT`) |

---

## 3. Dominios

| Archivo | Alcance | Cargar cuando… |
|---|---|---|
| [10-architecture.md](10-architecture.md) | Stack, capas, estructura de carpetas, convenciones Go, límites del iHost | Siempre que se diseñe una solución técnica |
| [20-ui.md](20-ui.md) | Componentes obligatorios, tokens de tema, patrones de listado, i18n, guardrails de UI | La spec toca frontend, formularios, listados o componentes |
| [30-ux.md](30-ux.md) | Navegación y retorno, modales, páginas de detalle, estados vacíos, feedback, prerequisitos | La spec toca pantallas, rutas, modales o flujos |
| [40-data.md](40-data.md) | SQLite, migraciones, copias de producción, esquema protegido | La spec toca DB, migraciones, importaciones o reportes |
| [50-security.md](50-security.md) | PII, secretos, datos sintéticos, higiene de commits, incidentes | **Siempre** (no negociable) |
| [60-testing.md](60-testing.md) | Pruebas locales, verificación del server, usuarios de prueba, flujo con el usuario | Siempre antes de `pending_release` |
| [70-delivery.md](70-delivery.md) | Worktrees, ramas, merge/push, release, Docker multi-arch, deploy | Antes de commit, merge o release |

---

## 4. Reglas no negociables (cargar siempre)

Estas reglas están detalladas en sus dominios, pero se aplican sin excepción y sin necesidad de lectura adicional:

| ID | Regla | Fuente |
|---|---|---|
| `SEC-NO-PII` | Prohibido commitear PII, datos financieros, secretos o volcados de producción | [50-security.md](50-security.md) |
| `SEC-SYNTHETIC` | Todo ejemplo/test usa datos sintéticos | [50-security.md](50-security.md) |
| `DATA-PRODCOPY` | Nunca se escribe en producción; se valida sobre copia en `/tmp` | [40-data.md](40-data.md) |
| `DATA-NO-DROP` | Nunca borrar ni reemplazar `data/app.db` del usuario | [40-data.md](40-data.md) |
| `DELIV-WORKTREE` | Cada sesión trabaja en su worktree; `main` es solo lectura para agentes | [70-delivery.md](70-delivery.md) |
| `DELIV-CONFIRM` | Merge y push requieren confirmación explícita del usuario | [70-delivery.md](70-delivery.md) |
| `SEC-INCIDENT` | Secreto/PII commiteado = incidente: rotar, purgar historia, documentar | [50-security.md](50-security.md) |

---

## 5. Guardrails ejecutables

Comandos que el gate de pre-release ejecuta cuando están disponibles. Si un comando no existe en el proyecto, se marca `N/A` con justificación.

| Guardrail | Regla | Comando | Cuándo |
|---|---|---|---|
| `check:ui` | `UI-SELECT` | `cd frontend && npm run check:ui` | La spec toca UI |
| `build` | `TEST-LOCAL` | `go build ./...` | Siempre |
| `tests` | `TEST-LOCAL` | `go test ./...` | Siempre |
| `server` | `TEST-SERVER` | `./scripts/check-server.sh` | Antes de avisar al usuario |
| `worktree` | `DELIV-WORKTREE` | `./scripts/check-worktree.sh` | Al arrancar |
| `diff` | `SEC-COMMIT` | `git diff --cached` (revisión manual) | Antes de commit |
| `hooks` | `DELIV-HOOKS` | `./scripts/install-hooks.sh` | Al clonar o crear worktree |
| `rules` | `DELIV-HOOKS` | `.opencode/skills/spec-manager/scripts/check-rules.sh .` | Al arrancar |

---

## 5.1 Enforcement automático

El proyecto tiene dos capas de prevención (SPEC-136):

| Capa | Archivo | Qué bloquea |
|---|---|---|
| Git hooks | `.githooks/pre-commit`, `.githooks/commit-msg` | DBs/PDFs/backups, secretos y PII, `public/i18n/`, `<select>`/checkbox nativo, mensajes sin `SPEC-XXX:` |
| Plugin OpenCode | `.opencode/plugins/rules-guard/` | `rm data/`, `pkill`, `git push --force`, `git reset --hard`, git de escritura en `main`, `public/i18n/`, `<select>`/checkbox nativo |

Los hooks se instalan con `./scripts/install-hooks.sh` (idempotente). El plugin se carga solo si `.opencode/node_modules` está instalado (`npm --prefix .opencode install`).

**Fail-safe**: si un hook o el plugin falla internamente, la operación continúa. Solo los patrones positivos bloquean.

---

## 6. IDs de regla

| Prefijo | Dominio |
|---|---|
| `ARCH-` | Arquitectura y stack |
| `UI-` | Interfaz y componentes |
| `UX-` | Experiencia, navegación y modales |
| `DATA-` | Datos y migraciones |
| `SEC-` | Seguridad y privacidad |
| `TEST-` | Pruebas y verificación |
| `DELIV-` | Entrega, git y deploy |

Formato: `PREFIJO-NOMBRE-CORTO` en mayúsculas (`UI-SELECT`, `UX-MODAL`, `DATA-PRODCOPY`).

Los criterios de aceptación de las specs usan el mismo prefijo con `CA-`: `CA-UI-SELECT`, `CA-UX-MODAL`, `CA-DATA-PRODCOPY`.

---

## 7. Mantenimiento

- Una regla se escribe **una sola vez** en su archivo de dominio.
- `AGENTS.md` y la skill **referencian**, no copian.
- Al agregar una regla: asignar ID, agregarla al dominio y, si es verificable, declarar el guardrail acá.
- Al cambiar una regla: actualizar su ID solo si cambia el significado; si es una aclaración, mantener el ID.
