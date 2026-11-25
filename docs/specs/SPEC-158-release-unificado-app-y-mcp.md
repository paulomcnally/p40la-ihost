---
title: "Release unificado: app e imagen MCP desde release.sh"
id: "SPEC-158"
status: "released"
author: "opencode"
created: "2026-11-24"
updated: "2026-11-24"
github_issue: 161
---

# Release unificado: app e imagen MCP desde release.sh

**ID**: SPEC-158<br>
**Estado**: released<br>
**Autor**: opencode<br>
**Creado**: 2026-11-24<br>
**Actualizado**: 2026-11-24<br>
**Release**: 2026-11-24 — merge a `main` `6839e217`; implementación `4bbe1c76`; `release.sh` libera app y MCP en el mismo ciclo; imágenes publicadas `p40la-ihost:0.5.20`, `p40la-mcp:0.6.0` y `p40la-mcp-tailscale:0.6.0` (multi-arch, workflows `success`); hotfix del polling de workflows (`2f82203f`); issue #161 cerrado.<br>

---

## 1. Resumen Ejecutivo

Hoy el release de la app y el de la imagen MCP están separados: `scripts/release.sh` versiona, tagea y publica la imagen web mediante GitHub Actions, mientras que `scripts/build-mcp-image.sh` construye y publica `p40la-mcp` localmente con `docker buildx`. Esa separación causó que SPEC-157 liberara la app sin publicar la imagen MCP que contiene las tools nuevas.

Esta spec mueve la responsabilidad del release MCP a `scripts/release.sh`: en el mismo ciclo, el script detecta si hay cambios que afectan al binario MCP desde el último tag `mcp-v*`, calcula o acepta una versión MCP, crea y pushea el tag `mcp-vX.Y.Z` y espera los dos workflows de GitHub Actions (imagen web y MCP). Si no hay cambios MCP, el release de la app sigue funcionando igual y el MCP se omite.

La publicación multi-arch queda en CI (`.github/workflows/mcp-publish.yml`), igual que la imagen principal: el release local no requiere Docker/QEMU ni `GH_TOKEN`, solo `bash`, `git`, `gh`, `curl` y los permisos ya usados por `release.sh`.

---

## 2. Cumplimiento de reglas

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Solo bash + git + gh + curl; sin dependencias nuevas | `CA-ARCH-STACK` |
| `ARCH-IHOST` | No cambia el runtime ni el contenedor; solo automatiza tags de release | `CA-ARCH-IHOST` |
| `ARCH-MCP-TOOLS-MAP` | No agrega ni cambia tools; el manifiesto no cambia | `CA-ARCH-MCP-TOOLS-MAP` |
| `DATA-*` | **N/A**: no toca DB, migraciones ni datos | — |
| `SEC-NO-PII` | El script no imprime tokens; usa la sesión de `gh` existente | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Ejemplos con versiones ficticias (`0.5.20`, `0.6.0`) | `CA-SEC-SYNTHETIC` |
| `SEC-COMMIT` | `git diff --cached` revisado; sin secretos ni DBs | `CA-SEC-COMMIT` |
| `UI-*` / `UX-*` | **N/A**: no toca frontend ni pantallas | — |
| `TEST-LOCAL` | `bash -n scripts/release.sh` y modo `--check` sin escrituras | `CA-TEST-LOCAL` |
| `TEST-MCP-CONTRACT` | **N/A**: no cambia tools, contrato ni backend; solo el release de la imagen | — |
| `DELIV-WORKTREE` | La spec vive en `../p40la-ihost-spec-158` sobre `feature/SPEC-158` | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | El release real se ejecuta con autorización explícita del usuario | `CA-DELIV-CONFIRM` |
| `DELIV-DOCKER` | La imagen MCP se publica multi-arch (`amd64`, `arm/v7`, `arm64`) por CI | `CA-DELIV-DOCKER` |
| `DELIV-MCP-DOCS` | Se actualizan `docs/mcp/operations.md`, `http.md`, `architecture.md` y `development.md` en el mismo commit | `CA-DELIV-MCP-DOCS` |
| `DELIV-MCP-DOCS-SUBAGENT` | La actualización de `docs/mcp/` se delega a un subagente | `CA-DELIV-MCP-DOCS-SUBAGENT` |
| `DELIV-MCP-IMPACT` | Sección `## Impacto en tools MCP` completa; `--check-map` y `--gate` verdes | `CA-DELIV-MCP-IMPACT` |
| `DELIV-MCP-IMPACT-SUBAGENT` | Validación de un subagente `explore` documentada en la sección 6 | `CA-DELIV-MCP-IMPACT-SUBAGENT` |
| `DELIV-COMMIT` | Commits `SPEC-158: ...` | `CA-DELIV-COMMIT` |
| `DELIV-LABELS` | Issue con `spec/draft` y transición a `spec/in-progress`/`spec/released` | `CA-DELIV-LABELS` |
| `DELIV-RELEASE` | Merge, push, tracker, label, cierre y limpieza en el mismo ciclo | `CA-DELIV-RELEASE` |
| `DELIV-DEPLOY` | **N/A**: no se despliega al iHost en esta spec | — |

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: `scripts/release.sh` conserva el release actual de la app: calcula/bump versiona, actualiza `docker-compose.yml` y `frontend/package.json`, commitea, tagea `vX.Y.Z`, pushea y espera el workflow `Build & Push Docker Image`.
2. **REQ-002**: Detectar si hay cambios que afectan al binario MCP desde el último tag `mcp-v*`. Rutas: `cmd/mcp-server/`, `internal/mcp/`, `Dockerfile.mcp`, `deploy/mcp/`, `.github/workflows/mcp-publish.yml`, `go.mod`, `go.sum`.
3. **REQ-003**: Si hay cambios MCP, calcular una versión MCP (bump de patch por defecto) o aceptar una explícita, crear el tag anotado `mcp-vX.Y.Z`, pushearlo para disparar `.github/workflows/mcp-publish.yml` y verificar que el workflow termine con conclusión `success`.
4. **REQ-004**: Validar que el tag MCP no exista localmente ni en `origin` antes de crearlo.
5. **REQ-005**: Si no hay cambios MCP, omitir el release MCP con un log claro; el release de la app continúa.
6. **REQ-006**: Permitir forzar la omisión del MCP con `--no-mcp` o `MCP_VERSION=skip`.
7. **REQ-007**: Permitir una versión MCP explícita como segundo argumento o con `MCP_VERSION`; validar semver `X.Y.Z`.
8. **REQ-008**: Esperar y reportar las URLs de ambos workflows (`Build & Push Docker Image` y `Build & Push MCP Image`) cuando corresponda.
9. **REQ-009**: Eliminar `scripts/build-mcp-image.sh` y actualizar la documentación que lo referencia para que apunte a `release.sh` o al `workflow_dispatch` del MCP.
10. **REQ-010**: Mantener el uso actual `./scripts/release.sh [app-version]` sin romperlo.

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-011**: Agregar un modo `--check` que muestre versiones, tag MCP detectado, decisión (publicar/omitir) y comandos previstos, sin modificar archivos, commits, tags ni remotos.
2. **REQ-012**: Si el usuario pasa una versión MCP explícita y no hay cambios, publicar igual (override explícito) con un warning.
3. **REQ-013**: Verificar que `.github/workflows/mcp-publish.yml` exista cuando se va a publicar el MCP; si falta, abortar antes de crear el tag.

### 3.3 Requerimientos No Funcionales

- **Compatibilidad**: bash 4+, Linux/macOS; sin `jq` obligatorio (se mantiene el fallback actual).
- **Seguridad**: no se imprimen tokens ni se pasan secretos por argumentos; el build usa los secretos de GitHub Actions.
- **Idempotencia**: si el tag MCP ya existe, el script aborta sin publicar.
- **Diagnóstico**: cada decisión (detectado/omitido, versión, tag, workflow) se loguea.
- **iHost**: sin impacto; el script no corre en el iHost.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

- `scripts/release.sh` ya valida ramas sin mergear, versión en Docker Hub, tags git, hace bump, tag y push, y espera el workflow principal.
- `.github/workflows/mcp-publish.yml` se dispara con tags `mcp-v*` y construye tres plataformas + manifest multi-arch; también acepta `workflow_dispatch`.
- `scripts/build-mcp-image.sh` publica localmente con `docker buildx` y requiere Docker/QEMU y `GH_TOKEN`; el binario MCP solo importa `cmd/mcp-server` e `internal/mcp`, así que esa es la superficie real de detección.
- Tags MCP existentes: `mcp-v0.1.0`, `mcp-v0.5.0`.
- SPEC-157 liberó la app sin publicar la imagen MCP; el catálogo de tools quedó en 44 pero la imagen seguía en 42.

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| A. `release.sh` llama a `build-mcp-image.sh` | Cambio mínimo | Mantiene dos scripts; exige Docker/QEMU/GH_TOKEN local; no usa el CI existente | ❌ Rechazada |
| B. `release.sh` crea/pushea `mcp-v*` y espera el workflow MCP | Usa el CI multi-arch existente; un solo comando; sin Docker local | Depende de GitHub Actions | ✅ Seleccionada |
| C. Publicar el MCP en cada release aunque no haya cambios | Simple | Quema versiones e imágenes sin cambios | ❌ Rechazada |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001: detección por diff desde el último tag `mcp-v*`**
- **Contexto**: no hay una carpeta única que represente el binario MCP; depende de `cmd/mcp-server`, `internal/mcp`, el Dockerfile, deploy y módulos Go.
- **Decisión**: comparar `git diff --name-only <último-tag-mcp>..HEAD` contra una allowlist de rutas.
- **Consecuencias**: un cambio en el backend web que no toca esas rutas no publica MCP; un cambio en `internal/mcp` sí. La lista se puede ampliar si aparece una dependencia nueva.

**ADR-002: versionado MCP independiente**
- **Contexto**: la app y el MCP tienen ciclos y tags distintos (`v0.5.19` vs `mcp-v0.5.0`).
- **Decisión**: mantener tags `mcp-vX.Y.Z`; sin versión explícita, bumpear patch desde el último tag MCP. Las features pueden pasar `MCP_VERSION=0.6.0`.
- **Consecuencias**: el release de la app no obliga a bumpear el MCP; el MCP puede quedar en una versión anterior si no cambió.

**ADR-003: publicación por CI, no local**
- **Contexto**: el workflow MCP ya construye las tres plataformas y el manifest; el build local duplica esa lógica y exige herramientas que el release de la app no necesita.
- **Decisión**: `release.sh` solo crea/pushea el tag; el CI publica. Se elimina `build-mcp-image.sh`.
- **Consecuencias**: el release es más rápido y consistente; para una publicación manual queda `workflow_dispatch`.

---

## 5. Diseño Técnico

### 5.1 Flujo

```
./scripts/release.sh [APP_VERSION] [MCP_VERSION]
   |
   +-- validaciones (main limpio, ramas mergeadas, gh)
   +-- release app: bump -> commit -> tag vX.Y.Z -> push -> workflow app
   +-- detectar cambios MCP desde mcp-v*
   |      |
   |      +-- sin cambios y sin override -> omitir
   |      +-- con cambios u override -> versión -> tag mcp-vX.Y.Z -> push -> workflow MCP
   +-- esperar workflows y mostrar URLs
```

### 5.2 Interfaz

```bash
# Release de app; MCP solo si hay cambios
./scripts/release.sh

# App 0.5.20; MCP automático si hay cambios
./scripts/release.sh 0.5.20

# App 0.5.20 y MCP 0.6.0 explícitos
./scripts/release.sh 0.5.20 0.6.0
MCP_VERSION=0.6.0 ./scripts/release.sh 0.5.20

# Forzar solo app
./scripts/release.sh --no-mcp
MCP_VERSION=skip ./scripts/release.sh

# Ver el plan sin escribir nada
./scripts/release.sh --check
```

### 5.3 Detección

```bash
MCP_LAST_TAG=$(git tag --list 'mcp-v*' | sort -V | tail -1)
MCP_CHANGED=$(git diff --name-only "${MCP_LAST_TAG}"..HEAD -- \
  cmd/mcp-server internal/mcp Dockerfile.mcp deploy/mcp \
  .github/workflows/mcp-publish.yml go.mod go.sum)
```

Si no hay tag previo, se considera que aplica.

### 5.4 Workflows

| Imagen | Tag | Workflow |
|---|---|---|
| App web | `vX.Y.Z` | `Build & Push Docker Image` |
| MCP | `mcp-vX.Y.Z` | `Build & Push MCP Image` |

### 5.5 Archivos

| Archivo | Cambio |
|---|---|
| `scripts/release.sh` | detección, versión y tag MCP, espera de ambos workflows, `--check`, `--no-mcp` |
| `scripts/build-mcp-image.sh` | eliminar; su lógica de release pasa a `release.sh` y el build multi-arch queda en CI |
| `docs/mcp/operations.md` | reemplazar el uso del script eliminado por `release.sh`/`workflow_dispatch` |
| `docs/mcp/http.md` | actualizar la referencia de publicación |
| `docs/mcp/architecture.md` | actualizar la referencia de imagen/publicación |
| `docs/mcp/development.md` | actualizar la tabla de imágenes |
| `docs/specs/SPEC-158-release-unificado-app-y-mcp.md` | esta spec |
| `docs/specs/README.md` | tracker en el release |

**Sin migraciones ni dependencias nuevas.**

---

## 6. Impacto en tools MCP

Ninguna tool MCP se agrega, elimina ni cambia de contrato. El cambio es exclusivamente de tooling de release (`scripts/release.sh`) y documentación; no toca `internal/mcp/`, `cmd/mcp-server/`, el manifiesto `docs/mcp/tools/impact.json` ni `docs/mcp/tools/contracts.json`.

| Tool | ¿Afectada? | Motivo | Contrato |
|---|---|---|---|
| Todas las tools del catálogo (44) | No | El script solo publica la imagen; no cambia código, schema, auth ni endpoints | Sin cambios |

**Validación subagente:** explore — sesión `ses_deedce7d7ffeEPMFyqAa8B8cLB` — veredicto: **viable con cambios, ninguna tool afectada**; `scripts/release.sh`, `build-mcp-image.sh` y `docs/mcp/*.md` no están en el manifiesto ni en `shared.affects_all`, por lo que el contrato y el golden no cambian; el binario MCP solo importa `cmd/mcp-server` e `internal/mcp`; se recomienda incluir `.github/workflows/mcp-publish.yml` en la detección y verificar la conclusión de los workflows para evitar releases parciales.

**Manifest**: `docs/mcp/tools/impact.json`
**Contrato**: `docs/mcp/tools/contracts.json`

---

## 7. Criterios de Aceptación

- [x] `CA-REQ-001`: `release.sh` sin argumentos sigue liberando la app como antes.
- [x] `CA-REQ-002`: con cambios en `internal/mcp/` desde el último `mcp-v*`, el script detecta que el MCP aplica.
- [x] `CA-REQ-003`: crea y pushea `mcp-vX.Y.Z`, espera el workflow `Build & Push MCP Image` y falla si la conclusión no es `success`.
- [x] `CA-REQ-004`: un tag MCP existente aborta antes de publicar.
- [x] `CA-REQ-005`: sin cambios MCP, el script omite el MCP y sigue con la app.
- [x] `CA-REQ-006`: `--no-mcp` y `MCP_VERSION=skip` omiten el MCP.
- [x] `CA-REQ-007`: `MCP_VERSION=0.6.0` y el segundo argumento usan esa versión.
- [x] `CA-REQ-008`: el resumen muestra las URLs de ambos workflows.
- [x] `CA-REQ-009`: `build-mcp-image.sh` ya no existe y la documentación apunta a `release.sh`/CI.
- [x] `CA-REQ-010`: `./scripts/release.sh 0.5.20` sigue funcionando.
- [x] `CA-REQ-011`: `--check` no modifica archivos, commits, tags ni remotos.
- [x] `CA-TEST-LOCAL`: `bash -n scripts/release.sh` pasa.
- [x] `CA-DELIV-DOCKER`: el workflow MCP conserva las tres plataformas y el manifest.
- [x] `CA-SEC-SYNTHETIC`: los ejemplos usan versiones ficticias.

---

## 8. Plan de Implementación

| Fase | Descripción | Estimación | Dependencias |
|---|---|---|---|
| 1 | Detección y versionado MCP en `release.sh` | 0.5 día | Ninguna |
| 2 | Tags, push y espera de ambos workflows; `--check`/`--no-mcp` | 0.5 día | Fase 1 |
| 3 | Eliminar `build-mcp-image.sh` y actualizar docs MCP | 0.25 día | Fase 2 |
| 4 | Validación local (`bash -n`, `--check`) y release | 0.25 día | Fase 3 |

---

## 9. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| La detección omite un cambio que sí afecta al MCP | Baja | Medio | Allowlist amplia y revisable; `MCP_VERSION` permite forzar |
| El workflow MCP falla después del push del tag | Baja | Medio | `release.sh` verifica `conclusion=success` y sale con error; el tag queda para reintentar el workflow desde GitHub Actions |
| Se publica MCP sin cambios reales | Baja | Bajo | Detección por diff y opción `--no-mcp` |
| El tag MCP ya existe | Baja | Bajo | Validación local y remota antes de crear |
| Divergencia de versiones app/MCP | Media | Bajo | Versiones independientes documentadas; el resumen muestra ambas |

---

## 10. Notas y Referencias

- `scripts/release.sh`, `scripts/build-mcp-image.sh`, `.github/workflows/mcp-publish.yml`.
- `docs/mcp/operations.md`, `docs/mcp/http.md`, `docs/mcp/architecture.md`, `docs/mcp/development.md`.
- Specs relacionadas: SPEC-027 (release.sh), SPEC-147 (MCP remoto), SPEC-148 (publicación MCP), SPEC-155/SPEC-157 (imágenes MCP).

---

## 11. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-24 | opencode | Creación inicial de la especificación; issue #161 |
| 2026-11-24 | opencode | Implementación: detección MCP por diff desde `mcp-v*`, versión automática/explícita, tag `mcp-vX.Y.Z`, espera y verificación de conclusión de ambos workflows, `--check` y `--no-mcp`; eliminado `build-mcp-image.sh`; docs MCP actualizadas. `bash -n` verde |
| 2026-11-24 | opencode | Release: merge a `main` `6839e217`; tracker, label y cierre del issue #161; `--check` validado con v0.5.20 y mcp-v0.6.0 |
| 2026-11-24 | opencode | Hotfix: `release.sh` reportó timeout aunque los workflows terminaron bien. Causas: `gh run view --jq -r` inválido, timeout de 10 min (el workflow de la app tardó ~15 min) y logs capturados por command substitution. Se corrige el polling, el timeout (20 min) y el log a stderr; release `2f82203f` |
