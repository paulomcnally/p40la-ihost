---
title: "Evitar releases sin cambios: app, MCP y wrapper Tailscale"
id: "SPEC-160"
status: "pending_release"
author: "opencode"
created: "2026-11-25"
updated: "2026-11-25"
github_issue: 163
---

# Evitar releases sin cambios: app, MCP y wrapper Tailscale

**ID**: SPEC-160
**Estado**: pending_release
**Autor**: opencode
**Creado**: 2026-11-25
**Actualizado**: 2026-11-25

---

## 1. Resumen Ejecutivo

`scripts/release.sh` publica hoy la app **siempre**, sin verificar si el contenido de la imagen cambió desde el último tag `v*`: aunque `main` solo tenga cambios de documentación (p. ej. `docs/specs/*.md`), el script bumpea `frontend/package.json` y `docker-compose.yml`, crea un tag `vX.Y.Z` y dispara un build multi-arch de la imagen `p40la-ihost`. El MCP sí se detecta por rutas, pero con una allowlist que **mezcla** el wrapper Tailscale (`deploy/mcp/` incluye `deploy/mcp/tailscale/`), y el wrapper Tailscale (`p40la-mcp-tailscale`) se republica con la **misma versión del MCP** y `latest` en **cada** release del MCP, aunque `deploy/mcp/tailscale/` no haya cambiado.

Esta spec introduce **detección de cambios independiente por artefacto** (app, MCP y wrapper Tailscale), cada uno con su allowlist de rutas y su propio tag git, de modo que solo se versiona/publica el artefacto cuyo contenido relevante cambió. Se define además un mecanismo explícito de *force* (versión explícita), una semántica de `--check` que no muta el estado local (sin `git fetch --tags`), y la propagación de fallos entre jobs para no reportar éxito incompleto.

Resultado esperado: `release.sh` deja de quemar versiones e imágenes sin cambios reales; `--check` representa fielmente las acciones y no escribe; un fallo en la publicación del wrapper Tailscale no queda oculto tras un `latest` publicado a medias.

No se ejecuta ningún release ni se publica ninguna imagen durante esta spec.

---

## 2. Cumplimiento de reglas

> Las reglas se citan por ID desde `docs/rules/`; no se copian.

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Solo bash + git + gh + curl + jq (fallback actual); sin dependencias nuevas | `CA-ARCH-STACK` |
| `ARCH-IHOST` | No cambia runtime ni contenedor; solo tooling de release. `release.sh` no corre en el iHost | `CA-ARCH-IHOST` |
| `ARCH-MCP-TOOLS-MAP` | No agrega/elimina/cambia tools; manifiesto y contrato sin cambios | `CA-ARCH-MCP-TOOLS-MAP` |
| `DATA-*` | **N/A**: no toca DB, migraciones ni datos | — |
| `SEC-NO-PII` | El script no imprime tokens; usa la sesión `gh` existente | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Ejemplos con versiones ficticias (`0.5.22`, `0.7.0`, `0.2.0`) | `CA-SEC-SYNTHETIC` |
| `SEC-COMMIT` | `git diff --cached` revisado; sin secretos/DBs/PDFs | `CA-SEC-COMMIT` |
| `UI-*` / `UX-*` | **N/A**: no toca frontend ni pantallas | — |
| `TEST-LOCAL` | `bash -n scripts/release.sh` + `--check` + casos con fixtures en `/tmp` | `CA-TEST-LOCAL` |
| `TEST-MCP-CONTRACT` | **N/A**: no cambia tools, contrato ni backend | — |
| `DELIV-WORKTREE` | La spec vive en `../p40la-ihost-spec-160` sobre `feature/SPEC-160` | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | El release real se ejecuta con autorización explícita del usuario | `CA-DELIV-CONFIRM` |
| `DELIV-DOCKER` | Las tres imágenes se publican multi-arch (`amd64`, `arm/v7`, `arm64`) por CI | `CA-DELIV-DOCKER` |
| `DELIV-MCP-DOCS` | Se actualizan `docs/mcp/operations.md` (y referencias) en el mismo commit | `CA-DELIV-MCP-DOCS` |
| `DELIV-MCP-DOCS-SUBAGENT` | La actualización de `docs/mcp/` se delega a un subagente | `CA-DELIV-MCP-DOCS-SUBAGENT` |
| `DELIV-MCP-IMPACT` | Sección `## Impacto en tools MCP` completa; `--check-map` y `--gate` verdes | `CA-DELIV-MCP-IMPACT` |
| `DELIV-MCP-IMPACT-SUBAGENT` | Validación de un subagente `explore` documentada en la sección 6 | `CA-DELIV-MCP-IMPACT-SUBAGENT` |
| `DELIV-COMMIT` | Commits `SPEC-160: ...` | `CA-DELIV-COMMIT` |
| `DELIV-LABELS` | Issue con `spec/draft` → `spec/pending-execution` | `CA-DELIV-LABELS` |
| `DELIV-RELEASE` | Merge, push, tracker, label, cierre y limpieza en el mismo ciclo (fase posterior) | `CA-DELIV-RELEASE` |
| `DELIV-DEPLOY` | **N/A**: no se despliega al iHost en esta spec | — |

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: `release.sh` detecta cambios de la **app** desde el último tag `v*` contra una allowlist que representa el contenido de la imagen `p40la-ihost`. Sin cambios y sin versión explícita, **no** versiona ni publica la app.
2. **REQ-002**: `release.sh` detecta cambios del **MCP** desde el último tag `mcp-v*` contra una allowlist que representa el binario `p40la-mcp`. La allowlist deja de incluir `deploy/mcp/tailscale/` (pasa al wrapper Tailscale).
3. **REQ-003**: `release.sh` detecta cambios del **wrapper Tailscale** desde el último tag `tailscale-v*` contra `deploy/mcp/tailscale/`. Sin cambios y sin versión explícita, no se publica el wrapper.
4. **REQ-004**: Cada artefacto tiene su propio tag git independiente: `vX.Y.Z` (app), `mcp-vX.Y.Z` (MCP) y `tailscale-vX.Y.Z` (wrapper Tailscale). El wrapper deja de compartir la versión del MCP.
5. **REQ-005**: `release.sh` con versión explícita (argumento posicional o variable de entorno) **fuerza** la publicación de ese artefacto aunque no haya cambios, con un log de advertencia.
6. **REQ-006**: Si ningún artefacto cambia y no hay versiones explícitas, `release.sh` reporta «nada para liberar» y termina **sin** crear commits, tags ni publicar.
7. **REQ-007**: `--check` representa fielmente el plan (versiones, tags, decisión publicar/omitir por artefacto, workflows) **sin mutar** el estado local: sin commits, sin tags, sin push y **sin `git fetch --tags`** (usa `git ls-remote --tags` read-only).
8. **REQ-008**: El wrapper Tailscale se publica desde un workflow propio (`tailscale-publish.yml`, disparado por `tailscale-v*`) y se elimina el job `build-tailscale` de `mcp-publish.yml`.
9. **REQ-009**: Un fallo en la publicación de cualquiera de los artefactos disparados hace que `release.sh` salga con error (no reporta éxito) y deje el tag para reintentar el workflow; dentro de cada workflow, el job de manifest multi-arch depende de **todos** los jobs de build de su workflow.
10. **REQ-010**: El uso actual `./scripts/release.sh [app-version]` y `./scripts/release.sh [app-version] [mcp-version]` se conserva; el tercer argumento posicional pasa a ser la versión del wrapper Tailscale.

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-011**: Variables de entorno `MCP_VERSION` y `TAILSCALE_VERSION` fuerzan las versiones correspondientes; `MCP_VERSION=skip`/`--no-mcp` omiten el MCP y `TAILSCALE_VERSION=skip`/`--no-tailscale` omiten el wrapper (nuevo flag).
2. **REQ-012**: El `--check` imprime, para cada artefacto, el último tag, la versión que se crearía, la lista de archivos cambiados (o «sin cambios») y la decisión.
3. **REQ-013**: Validar idempotencia: abortar si el tag de cualquier artefacto ya existe local o remotamente antes de crear nada.

### 3.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-014**: Un flag `--list-tags` que muestre, para cada artefacto, su último tag y si hay cambios pendientes, sin armar plan de release.

### 3.4 Requerimientos No Funcionales

- **Compatibilidad**: bash 4+, Linux/macOS; `jq` opcional (se conserva el fallback actual).
- **Seguridad**: no imprime tokens ni secretos; usa los secretos de GitHub Actions en CI.
- **Idempotencia**: tags existentes abortan; re-ejecución no duplica.
- **Diagnóstico**: cada decisión (detectado/omitido/forzado, versión, tag, workflow) se loguea.
- **iHost**: sin impacto (el script no corre en el iHost).

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

La validación multiagente se realizó con dos subagentes: `explore` (superficie del repo) y `general` (best practices externas).

**Superficie del repo (subagente `explore`, sesión `ses_dee8622dcffexdyuyyMrehejFS`):**

- La app (`cmd/server`) y el MCP (`cmd/mcp-server`) son **grafos de import Go separados**: ningún `.go` de producción fuera de `internal/mcp/` importa `internal/mcp` (la única aparición es `internal/api/mcp_backend_contract_test.go`, un `_test.go`). La comunicación MCP↔app es por HTTP en runtime (`internal/mcp/api/client.go`), no por import. Por lo tanto, un cambio en `internal/mcp/` **no** recompila el binario de la app.
- Contenido de la imagen de la app (`Dockerfile`): binario `./cmd/server`, `public/` (el build de Vite sobrescribe el `public/` commiteado) y `migrations/`. `.dockerignore` excluye `docs/`, `scripts/`, `*.md` (salvo `README.md`), `data/`, etc.
- Contenido de la imagen MCP (`Dockerfile.mcp`): solo el binario `./cmd/mcp-server` y el directorio `/data`; no copia `migrations/` ni `public/`.
- Wrapper Tailscale: contexto de build exclusivamente `deploy/mcp/tailscale/` (solo `Dockerfile` + `serve.json`), `FROM tailscale/tailscale:latest`.
- No existe ningún tag `tailscale-v*` (local ni remoto); el wrapper hereda la versión del MCP (`mcp-publish.yml` extrae `TAG` de `mcp-vX.Y.Z`).
- `scripts/build-mcp-image.sh` ya no existe (eliminado en SPEC-158). Además de `release.sh`, existen `scripts/push-dockerhub.sh` y `scripts/build.sh` (buildx local de `p40la-ihost`, dev).
- `VERSION=0.5.21` en `docker-compose.yml` es solo una env de runtime; no la usa el build.
- Últimos tags: `v0.5.21` (app) y `mcp-v0.6.1`; desde `v0.5.21` solo hay cambios de documentación (`docs/specs/README.md`, `docs/specs/SPEC-159-*.md`), pero `release.sh` planearía igual `v0.5.22`.
- No existe ningún test automatizado sobre `release.sh`; solo `bash -n` y el `--check` actual.

**Best practices externas (subagente `general`, sesión `ses_dee8622dbffeqcgi3P6Lp5Tihz`):**

- **Los filtros `paths`/`paths-ignore` de `on.push` NO se evalúan para tags** (solo para branches). Confirmado en la doc oficial de GitHub Actions. Por eso la detección de cambios debe vivir en el script (pre-tag), no en el workflow. Fuentes: <https://docs.github.com/en/actions/writing-workflows/workflow-syntax-for-github-actions>; <https://stackoverflow.com/questions/76037078>.
- **Detección estándar**: `git diff --name-only <último-tag>..HEAD -- <paths>`; `git describe --tags --abbrev=0` falla (exit 128) sin tags; el fallback de «primera release» es difiar contra el árbol vacío (`4b825dc642cb6eb9a060e54bf8d69288fbee4904`). Fuente: <https://git-scm.com/docs/git-diff>, <https://git-scm.com/docs/git-describe>.
- **`git fetch --tags` muta `refs/tags/*`**; la alternativa read-only es `git ls-remote --tags <remote>`. Fuente: <https://git-scm.com/docs/git-fetch>, <https://git-scm.com/docs/git-ls-remote>.
- **Versionado independiente en monorepo** = tag por artefacto + detección por paths (patrón de `semantic-release-monorepo`, `changesets`, `python-semantic-release` con `path_filters`). Fuentes: <https://github.com/pmowrer/semantic-release-monorepo>, <https://changesets.dev/guide/technical-decisions>, <https://python-semantic-release.readthedocs.io/en/latest/configuration/configuration-guides/monorepos.html>.
- **Falla parcial en workflows**: un job que falla (sin `continue-on-error: true`) falla el run completo, pero un job final no-gateado reporta `success`. El patrón correcto es un job agregador con `needs` sobre todas las publicaciones + `if: ${{ !cancelled() }}` + verificación de `needs.<job>.result` (tratando `skipped` como OK). Preferir `!cancelled()` a `always()`. Fuentes: <https://docs.github.com/en/actions/using-jobs/using-jobs-in-a-workflow>, <https://www.kenmuse.com/blog/how-to-handle-step-and-job-errors-in-github-actions>.
- **No hay «skip» nativo por contenido**: para releases por tag, «no publicar» = no crear el tag (decisión pre-CI). Fuente: <https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/control-jobs-with-conditions>.

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| A. Detección por diff de rutas contra el último tag de cada artefacto (allowlist) | Simple, auditable, sin resolver grafos de import; cada artefacto independiente | Puede sobre-disparar si una ruta compartida (`go.mod`, `go.sum`) toca dos artefactos (correcto: ambos reconstruyen) | ✅ Seleccionada |
| B. Resolver grafos de import Go para decidir el binario afectado | Precisión máxima | Complejo, frágil, no cubre `public/`/`migrations/`/Dockerfiles | ❌ Rechazada |
| C. Filtrar por `paths`/`paths-ignore` en `on.push` de GitHub Actions | Cero lógica en el script | `paths` no aplica a tags (solo branches) — confirmado en la doc oficial | ❌ Rechazada |
| D. Publicar todo siempre (status quo) | Simple | Quema versiones e imágenes sin cambios | ❌ Rechazada |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001: detección independiente por allowlist de rutas**
- **Contexto**: tres imágenes con contenido distinto comparten un solo repo y un `go.mod`.
- **Decisión**: cada artefacto compara `git diff --name-only <último-tag>..HEAD -- <allowlist>`; para la app se excluye `internal/mcp/` del resultado (ver ADR-002). Sin tag previo, se difia contra el árbol vacío (`4b825dc642cb6eb9a060e54bf8d69288fbee4904`) y se trata como «cambió» (primera release).
- **Consecuencias**: `go.mod`/`go.sum` disparan app y MCP (ambos reconstruyen); un cambio solo de docs no dispara ninguno.

**ADR-002: la app excluye `internal/mcp/` de su allowlist**
- **Contexto**: el subagente `explore` confirmó que `cmd/server` no importa `internal/mcp` en producción (grafos separados).
- **Decisión**: la allowlist de la app usa `internal/` **menos** `internal/mcp/` (filtrado con `grep -v '^internal/mcp/'` sobre el diff).
- **Consecuencias**: un cambio solo del MCP no versiona la app; requiere mantener el filtro si algún día el grafo de import cruza `internal/mcp`.

**ADR-003: versionado independiente del wrapper Tailscale**
- **Contexto**: hoy el wrapper hereda la versión del MCP y se republica en cada release MCP.
- **Decisión**: tag `tailscale-vX.Y.Z` con bump de patch propio desde el último `tailscale-v*`; imagen `p40la-mcp-tailscale:X.Y.Z` + `latest`. Sin tag previo: `0.1.0`.
- **Consecuencias**: el wrapper solo se publica cuando cambia `deploy/mcp/tailscale/`; separa su historial del MCP.

**ADR-004: workflow propio para el wrapper Tailscale**
- **Contexto**: el wrapper tiene un ciclo de vida distinto del MCP y es una imagen trivial (2 archivos, sin compilación).
- **Decisión**: nuevo `.github/workflows/tailscale-publish.yml` (trigger `tailscale-v*` + `workflow_dispatch`) con un único job multi-arch (buildx `--platform linux/amd64,linux/arm/v7,linux/arm64 --push`); se elimina `build-tailscale` de `mcp-publish.yml`.
- **Consecuencias**: un fallo del wrapper no se mezcla con el resultado del MCP; cada tag → un workflow → una conclusión.

**ADR-005: `--check` read-only**
- **Contexto**: `git fetch --tags` muta `refs/tags` locales.
- **Decisión**: `--check` (y la detección en general) usa `git ls-remote --tags origin` (read-only) para resolver tag → SHA y `git diff <sha>..HEAD` para los cambios. No se hace `git fetch` en `--check`. En modo release real, el fetch se mantiene para asegurar tags frescos antes de crear los nuevos.
- **Consecuencias**: `--check` no muta; requiere que los commits apuntados por los tags estén presentes localmente (lo están tras un `git pull`/fetch previo).

---

## 5. Diseño Técnico

### 5.1 Flujo

```
./scripts/release.sh [app-version] [mcp-version] [tailscale-version]
   |
   +-- validaciones (main, árbol limpio, gh; en --check solo aviso)
   +-- resolver último tag + diff por artefacto (git ls-remote + git diff)
   +-- por artefacto: ¿cambió? o ¿versión explícita? o ¿omitido?
   |      +-- app      → bump patch desde último v*   | versión explícita
   |      +-- mcp      → bump patch desde último mcp-v* | versión explícita / --no-mcp
   |      +-- tailscale→ bump patch desde último tailscale-v* | versión explícita / --no-tailscale
   +-- si --check: imprimir plan y salir (sin mutar)
   +-- si nada cambió: «nada para liberar» y salir 0
   +-- actualizar versiones (docker-compose.yml, frontend/package.json)
   +-- commit bump + tags (v*/mcp-v*/tailscale-v*) + push
   +-- esperar workflows (app, mcp, tailscale) y fallar si conclusión != success
```

### 5.2 Interfaz

```bash
./scripts/release.sh                          # libera lo que cambió (bump patch cada uno)
./scripts/release.sh 0.5.22                   # fuerza app 0.5.22; MCP/tailscale automáticos
./scripts/release.sh 0.5.22 0.7.0             # fuerza app + MCP
./scripts/release.sh 0.5.22 0.7.0 0.2.0       # fuerza app + MCP + tailscale
MCP_VERSION=0.7.0 ./scripts/release.sh        # MCP explícito
TAILSCALE_VERSION=0.2.0 ./scripts/release.sh  # tailscale explícito
APP_VERSION=0.5.22 ./scripts/release.sh       # app explícita por entorno (REQ-005)
./scripts/release.sh --no-mcp                 # omitir MCP
./scripts/release.sh --no-tailscale           # omitir wrapper (nuevo)
./scripts/release.sh --check                  # plan read-only
./scripts/release.sh --list-tags              # último tag + cambios pendientes (REQ-014)
```

### 5.3 Allowlists de detección

| Artefacto | Tag | Rutas | Imagen Docker Hub |
|---|---|---|---|
| App | `vX.Y.Z` | `cmd/server/`, `internal/` (menos `internal/mcp/`), `frontend/`, `public/`, `migrations/`, `go.mod`, `go.sum`, `Dockerfile`, `.github/workflows/docker-publish.yml` | `paulomcnally/p40la-ihost` |
| MCP | `mcp-vX.Y.Z` | `cmd/mcp-server/`, `internal/mcp/`, `Dockerfile.mcp`, `go.mod`, `go.sum`, `.github/workflows/mcp-publish.yml` | `paulomcnally/p40la-mcp` |
| Tailscale | `tailscale-vX.Y.Z` | `deploy/mcp/tailscale/` | `paulomcnally/p40la-mcp-tailscale` |

### 5.4 Workflows

| Imagen | Tag | Workflow |
|---|---|---|
| App | `vX.Y.Z` | `Build & Push Docker Image` (`docker-publish.yml`) |
| MCP | `mcp-vX.Y.Z` | `Build & Push MCP Image` (`mcp-publish.yml`, sin `build-tailscale`) |
| Tailscale | `tailscale-vX.Y.Z` | `Build & Push Tailscale Image` (`tailscale-publish.yml`, nuevo) |

- En `docker-publish.yml` y `mcp-publish.yml`, el job `merge` (manifest multi-arch) ya declara `needs: build`; se conserva y se documenta que debe cubrir **todos** los jobs de build. Si en el futuro un workflow tuviera más de un job de publicación, el job agregador debe declarar `needs` sobre todos y verificar `needs.<job>.result` con `if: ${{ !cancelled() }}` (tratando `skipped` como OK, `failure`/`cancelled` como error), sin usar `always()`.
- El wrapper es un único job multi-arch (sin matriz), por lo que no tiene problema de manifest parcial.

### 5.5 Archivos

| Archivo | Cambio |
|---|---|
| `scripts/release.sh` | detección por artefacto, `tailscale-v*`, `--check` read-only, `--no-tailscale`, `TAILSCALE_VERSION`, fallo global si algún workflow falla |
| `scripts/test-release-detection.sh` | **nuevo**: tests de la matriz de detección con fixtures git en `/tmp` (orientado a `scripts/release.sh`, sin red ni Docker Hub) |
| `.github/workflows/mcp-publish.yml` | eliminar job `build-tailscale` |
| `.github/workflows/tailscale-publish.yml` | **nuevo**: build multi-arch + push del wrapper `p40la-mcp-tailscale` |
| `docs/mcp/operations.md` | documentar detección independiente, tags `tailscale-v*`, `--check` read-only, `--no-tailscale` |
| `docs/mcp/architecture.md` | actualizar tabla de imágenes/versionado del wrapper |
| `docs/mcp/development.md` | actualizar tabla de imágenes/publicación |
| `docs/specs/SPEC-160-evitar-releases-sin-cambios-app-mcp-tailscale.md` | esta spec |
| `docs/specs/README.md` | tracker en el release |

**Sin migraciones ni dependencias nuevas.**

---

## 6. Impacto en tools MCP

Ninguna tool MCP se agrega, elimina ni cambia de contrato. El cambio es exclusivamente de tooling de release (`scripts/release.sh`), workflows (`mcp-publish.yml`, nuevo `tailscale-publish.yml`) y documentación `docs/mcp/`; no toca `internal/mcp/`, `cmd/mcp-server/`, `docs/mcp/tools/impact.json` ni `docs/mcp/tools/contracts.json`.

| Tool | ¿Afectada? | Motivo / ruta de dependencia | Contrato |
|---|---|---|---|
| Todas las tools del catálogo | No | Solo cambia el proceso de publicación de la imagen; no código, schema, auth ni endpoints | Sin cambios |

Validación subagente: `explore` — sesión `ses_dee8622dcffexdyuyyMrehejFS` — veredicto: **ninguna tool afectada**; los archivos modificados (`scripts/release.sh`, `.github/workflows/mcp-publish.yml`, nuevo `.github/workflows/tailscale-publish.yml`, `docs/mcp/*.md`) no están en el manifiesto `docs/mcp/tools/impact.json` ni en `shared.affects_all`; el binario MCP (`cmd/mcp-server`) y el contrato (`contracts.json`) no cambian.
**Manifest**: `docs/mcp/tools/impact.json`
**Contrato**: `docs/mcp/tools/contracts.json`

---

## 7. Criterios de Aceptación

### 7.1 Funcionales

- [ ] `CA-REQ-001`: con solo cambios en `docs/` (sin tocar las allowlists), `release.sh` **no** bumpea ni publica la app.
- [ ] `CA-REQ-002`: con cambios en `internal/mcp/` y sin cambios en el resto de la app, el MCP se publica y la app **no** se versiona.
- [ ] `CA-REQ-003`: con cambios en `deploy/mcp/tailscale/` y sin cambios en MCP, solo se publica el wrapper `tailscale-vX.Y.Z`; el MCP no.
- [ ] `CA-REQ-004`: con cambios en `deploy/mcp/tailscale/` y en `internal/mcp/`, se publican MCP y wrapper con tags independientes.
- [ ] `CA-REQ-005`: `release.sh 0.5.22` fuerza la app aunque no haya cambios, con warning; `TAILSCALE_VERSION=0.2.0` fuerza el wrapper.
- [ ] `CA-REQ-006`: sin cambios y sin versiones explícitas, `release.sh` imprime «nada para liberar» y no crea commits, tags ni workflows.
- [ ] `CA-REQ-007`: `--check` no crea commits, tags ni push, **y** no ejecuta `git fetch --tags` (verificable comparando `git show-ref` antes y después).
- [ ] `CA-REQ-008`: no existe el job `build-tailscale` en `mcp-publish.yml`; existe `tailscale-publish.yml` con build multi-arch + push.
- [ ] `CA-REQ-009`: si un workflow disparado falla, `release.sh` sale con error y no reporta éxito.
- [ ] `CA-REQ-010`: `./scripts/release.sh 0.5.22 0.7.0` sigue funcionando y el tercer argumento pasa a ser la versión del wrapper.
- [ ] `CA-REQ-013`: un tag existente (local o remoto) aborta antes de crear nada.

### 7.2 No funcionales

- [ ] `CA-NF-001`: `bash -n scripts/release.sh` pasa sin errores.
- [ ] `CA-NF-002`: `--check` termina en tiempo razonable sin llamadas de red que muten estado (solo `git ls-remote` read-only).

### 7.3 Testing

- **Automatizado**: `./scripts/test-release-detection.sh` construye fixtures git descartables en `$TMPDIR` (repo + origen bare local), aplica la matriz de detección y valida 27 casos sin red ni Docker Hub (`bash -n`, estructura de workflows, detección, `--check` read-only, versiones forzadas, `--list-tags`, colisión de tags local/remota y propagación de fallos de workflow con `gh` simulado).
- **Estáticos**: `bash -n scripts/release.sh`; `shellcheck` si está disponible (opcional).
- **Casos de detección**: con un fixture git en `/tmp` (repositorio temporal con tags `v*`/`mcp-v*`/`tailscale-v*` y commits sobre rutas específicas), verificar la matriz de detección (docs-only, mcp-only, tailscale-only, combinados) sin tocar Docker Hub.
- **`--check` read-only**: capturar `git show-ref` antes/después y verificar que no cambia.
- **Workflows**: validar sintaxis YAML (no se ejecuta publicación real durante la spec).

---

## 8. Plan de Implementación

### 8.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|---|---|---|---|
| 1 | Detección por artefacto (app/MCP/tailscale) en `release.sh` + `--check` read-only | 0.5 día | Ninguna |
| 2 | Versionado `tailscale-v*`, tercer argumento, `--no-tailscale`, `TAILSCALE_VERSION`, fallo global | 0.5 día | Fase 1 |
| 3 | Workflow `tailscale-publish.yml` + quitar `build-tailscale` de `mcp-publish.yml` | 0.25 día | Fase 2 |
| 4 | Docs `docs/mcp/` (subagente) + casos de detección con fixture en `/tmp` | 0.25 día | Fase 3 |
| 5 | Validación local (`bash -n`, `--check`, matriz de detección) y release | 0.25 día | Fase 4 |

### 8.2 Milestones

1. **MVP**: detección independiente + `--check` read-only + `tailscale-v*`.
2. **V1.0**: workflow propio del wrapper + propagación de fallos + docs.

---

## 9. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| Allowlist omite un cambio real (release no publicado) | Media | Alto | Allowlists amplias y revisables; versión explícita fuerza; `--check` muestra los archivos detectados |
| Allowlist sobre-dispara (release innecesario) | Media | Bajo | ADR-002 excluye `internal/mcp/` de la app; revisar con subagente |
| `git diff <sha>..HEAD` falla por commit no presente | Baja | Medio | Documentar que `--check` requiere refs frescas; release real hace `git fetch` primero |
| Tag `tailscale-v*` colisiona con esquema previo inexistente | Baja | Bajo | No hay tags `tailscale-v*` hoy; validación de existencia antes de crear |
| Fallo parcial del wrapper no detectado | Baja | Alto | Workflow propio + `release.sh` espera su conclusión y falla si no es `success` |
| Re-ejecución duplica versión | Baja | Bajo | Validación de tag local/remoto antes de crear |

---

## 10. Notas y Referencias

- Reglas aplicadas: `docs/rules/10-architecture.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.
- `scripts/release.sh`, `.github/workflows/docker-publish.yml`, `.github/workflows/mcp-publish.yml`, `deploy/mcp/tailscale/`.
- `docs/mcp/operations.md`, `docs/mcp/architecture.md`, `docs/mcp/development.md`.
- Specs relacionadas: SPEC-027 (release.sh), SPEC-147/SPEC-148 (MCP remoto y nodo Tailscale), SPEC-158 (release unificado app y MCP).
- Issue raíz: AGE-13 (ef07b389-0254-4fe5-ba6c-f08125dcac61).
- Fuentes externas: doc oficial de GitHub Actions (paths/tags, `needs`, `continue-on-error`, `if`), `git-diff`/`git-ls-remote`/`git-fetch`/`git-describe`, `semantic-release-monorepo`, `changesets`, `python-semantic-release` (monorepo), Ken Muse (errores de jobs).

---

## 11. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-25 | opencode | Creación inicial de la especificación (draft); validación multiagente (explore + general) integrada |
| 2026-11-25 | opencode | Estado `pending_execution`; issue de GitHub #163 |
| 2026-11-25 | opencode | Estado `in_progress` (implementación): se agrega `scripts/test-release-detection.sh` como suite automatizada de la matriz de detección (§7.3), se documenta `APP_VERSION` y `--list-tags` en la interfaz (§5.2) y el archivo de tests en §5.5 |
| 2026-11-25 | DevOps | Estado `pending_release`: QA y Security aprobaron SPEC-160; worktree verificado y plan read-only confirma publicar MCP `mcp-v0.6.2` y wrapper `tailscale-v0.1.0`, sin versionar la app |
