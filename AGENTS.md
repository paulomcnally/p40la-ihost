# AGENTS.md — p40la-ihost

> **Última actualización**: 2026-11-17
> **Sistema de specs**: v2.0 (reglas por dominio — SPEC-136)

Este archivo es la **entrada** para agentes. Las reglas completas viven en `docs/rules/`, una sola vez cada una, con IDs estables.

---

## ⚠️ Regla de Oro

**Antes de escribir código: la spec primero. Siempre.**
Si la spec no existe, no se escribe código. Si la spec no contempla el cambio, se actualiza primero. Sin excepciones.

**Cerrar una spec es SIEMPRE un release, jamás una cancelación.**
`cancelled` solo existe si el usuario lo ordena explícitamente con las palabras «cancelar» o «descartar». Ante duda, preguntar antes de cambiar el estado. Precedente: SPEC-085.

---

## 🚨 No negociables (aplicar siempre)

| ID | Regla |
|---|---|
| `SEC-NO-PII` | Prohibido commitear PII, datos financieros, secretos o volcados de producción. Aplica a código, tests, specs, docs, issues, PRs, commits, logs y capturas. |
| `SEC-SYNTHETIC` | Todo ejemplo/test usa datos sintéticos (`demo@example.com`, `demo-account-001`, `+1 555 0100`, `Proveedor Demo`, `F000000000000001`). |
| `SEC-INCIDENT` | Secreto/PII commiteado = incidente: rotar, purgar del árbol **y de la historia**, documentar, avisar. |
| `DATA-NO-DROP` | Nunca borrar ni reemplazar `data/app.db`. La DB local del usuario es producción. |
| `DATA-PRODCOPY` | Validar contra copia de producción en `/tmp`, nunca contra la DB local. Nunca escribir en producción. |
| `DELIV-WORKTREE` | Cada sesión en su worktree. `main` es solo lectura para agentes. |
| `DELIV-CONFIRM` | Merge y push requieren confirmación explícita del usuario. Prohibido `--force`. |
| `TEST-KILLALL` | Usar `killall`, nunca `pkill`. |
| `TEST-SERVER` | Verificar `./scripts/check-server.sh` antes de avisar al usuario. |

Detalle: [`docs/rules/50-security.md`](docs/rules/50-security.md), [`40-data.md`](docs/rules/40-data.md), [`70-delivery.md`](docs/rules/70-delivery.md), [`60-testing.md`](docs/rules/60-testing.md).

---

## 📚 Reglas por dominio

| Archivo | Alcance | Cargar cuando… |
|---|---|---|
| [`docs/rules/00-index.md`](docs/rules/00-index.md) | Manifiesto, precedencia, guardrails, IDs | **Siempre** (Paso 0) |
| [`docs/rules/10-architecture.md`](docs/rules/10-architecture.md) | Stack, capas, carpetas, Go, límites del iHost | Se diseña una solución |
| [`docs/rules/20-ui.md`](docs/rules/20-ui.md) | Componentes, tokens, listados, i18n | Toca frontend/UI |
| [`docs/rules/30-ux.md`](docs/rules/30-ux.md) | Navegación, modales, detalle, estados, formularios | Toca pantallas o flujos |
| [`docs/rules/40-data.md`](docs/rules/40-data.md) | SQLite, migraciones, copias de producción | Toca DB o datos reales |
| [`docs/rules/50-security.md`](docs/rules/50-security.md) | PII, secretos, datos sintéticos | **Siempre** |
| [`docs/rules/60-testing.md`](docs/rules/60-testing.md) | Pruebas locales, verificación, flujo con el usuario | Antes de release |
| [`docs/rules/70-delivery.md`](docs/rules/70-delivery.md) | Git, worktrees, release, Docker, deploy | Antes de commit/merge |

> `docs/project-rules.md` quedó como **stub de compatibilidad** para las specs históricas. No agregar reglas allí.

---

## 🛡️ Enforcement automático

| Capa | Instalación | Qué bloquea |
|---|---|---|
| Git hooks (`.githooks/`) | `./scripts/install-hooks.sh` | DBs/PDFs/backups, secretos y PII, `public/i18n/`, `<select>`/checkbox nativo, commits sin `SPEC-XXX:` |
| Plugin (`.opencode/plugins/rules-guard/`) | `npm --prefix .opencode install` | `rm data/`, `pkill`, `git push --force`, `git reset --hard`, git de escritura en `main`, `public/i18n/`, `<select>`/checkbox nativo |

- **Prohibido** `git commit --no-verify` salvo autorización explícita del usuario.
- Fail-safe: si un hook o el plugin falla internamente, la operación continúa; solo los patrones positivos bloquean.
- Detalle: [`docs/rules/70-delivery.md`](docs/rules/70-delivery.md) (`DELIV-HOOKS`) y [`50-security.md`](docs/rules/50-security.md) (`SEC-HOOKS`).

---

## 🔄 Sistema de specs (resumen)

```
draft → pending_execution → in_progress → pending_release → released
```

- IDs `SPEC-XXX` inmutables y secuenciales. El próximo ID se calcula desde `docs/specs/SPEC-*.md` + issues, **no** desde el README.
- Toda spec vive en un worktree propio y tiene un issue de GitHub con label `spec/<estado>`.
- El README tracker (`docs/specs/README.md`) se actualiza **solo en el release**.
- Antes de `pending_release`: checklist de calidad de [`docs/rules/60-testing.md`](docs/rules/60-testing.md).
- Antes de declarar terminada una spec: `/spec-check-release`.

Flujo completo, comandos y gates: skill `spec-manager` (`.opencode/skills/spec-manager/SKILL.md`).

---

## ✅ Checklist mínima antes de `pending_release`

- [ ] Requerimientos P0 implementados y criterios de aceptación verificados.
- [ ] `go build ./...` y `go test ./...` pasan.
- [ ] `cd frontend && npm run check:ui && npm run check:i18n && npm run check:numbers` pasa (si hay UI).
- [ ] Probado en local y confirmado por el usuario (🤖).
- [ ] Si depende de datos: validado contra copia de producción.
- [ ] `git diff --cached` sin PII, secretos, DBs, PDFs ni exports.
- [ ] Hooks instalados (`./scripts/install-hooks.sh`) y sin `--no-verify`.
- [ ] Tabla «Cumplimiento de reglas» de la spec completa.

---

## 🗂️ Estructura

```
/
├── .opencode/skills/spec-manager/   # Skill + references (templates de reglas y specs)
├── cmd/server/                      # Entrypoint
├── internal/{api,config,db,models,services,storage}
├── frontend/                        # Fuente React (build fuera del iHost)
├── public/                          # Salida del build
├── docs/
│   ├── rules/                       # ← fuente de verdad de las reglas
│   ├── specs/                       # Specs + tracker
│   ├── ssh-ihost-access.md
│   └── project-rules.md             # stub de compatibilidad
├── migrations/
├── scripts/
└── AGENTS.md                        # este archivo
```

---

*Las reglas del proyecto prevalecen sobre cualquier skill o template genérico.*
