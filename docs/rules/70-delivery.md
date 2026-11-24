# 70 — Entrega: git, worktrees, release y deploy

> **Dominio**: flujo de git, aislamiento por worktree, release de specs, Docker multi-arch, deploy y escalación.
> **Cargar cuando**: se va a commitear, mergear, liberar una spec o desplegar.

---

## DELIV-WORKTREE — Un worktree por sesión

- El checkout principal (`main`) es de **SOLO LECTURA** para agentes: no se edita código ni se ejecutan `git commit`/`reset`/`checkout`/`switch` allí.
- Cada ventana/sesión trabaja en **su propio** worktree. El proyecto admite hasta 6 ventanas en paralelo sobre specs distintas.
- Al arrancar cualquier sesión: ejecutar `./scripts/check-worktree.sh`. **Si aborta, no se trabaja en ese checkout.**
- Para empezar: `./scripts/new-worktree.sh SPEC-XXX` crea `../p40la-ihost-spec-XXX` en la rama `feature/SPEC-XXX`.
- La spec se crea **dentro** del worktree (archivo + issue). El README tracker se actualiza solo en el release.

**Prohibido**: `git checkout`, `git switch`, `git reset`, `git stash`, `git clean` sobre el worktree principal o el de otra sesión.
**Prohibido**: `git reset --hard` salvo que el usuario lo pida explícitamente para su propio worktree.

Antes de cualquier operación git: confirmar el worktree actual con `git worktree list` y `git branch --show-current`.

---

## DELIV-CONFIRM — Merge y push requieren confirmación

- **JAMÁS** ejecutar `git push` a ramas protegidas sin confirmación explícita del usuario.
- **JAMÁS** mergear a `main` directamente desde una rama de feature sin confirmación.
- Para liberar una spec: mergear la rama del worktree a `main` **con confirmación del usuario**, luego actualizar label y cerrar el issue.
- **JAMÁS** ejecutar `--force` ni `--force-with-lease` en `main` ni en ramas de release.
- Solo el usuario puede aprobar deploys a producción/iHost.

---

## DELIV-COMMIT — Formato de commits

- Mensaje: `SPEC-XXX: descripción corta`.
- Los commits de implementación van acompañados, en la misma sesión, del commit de release que documenta el cambio de estado.
- No dejar una spec en `pending_release` después de mergear a `main`.

---

## DELIV-RELEASE — Flujo de release

Una spec pasa a `released` solo cuando se completan **todos** estos pasos en el mismo ciclo:

1. Merge de la rama del worktree a `main` (con confirmación).
2. Commit de todos los cambios pendientes y push.
3. Documentar commit/versión y fecha en la spec.
4. Actualizar `status` del frontmatter **y** el cuerpo (`**Estado**:`).
5. Actualizar el README tracker (`docs/specs/README.md`): tabla + contadores.
6. Cambiar el label de GitHub (`spec/<anterior>` → `spec/released`) y cerrar el issue.
7. Limpiar el worktree y la rama de la spec:
   ```bash
   git worktree remove ../p40la-ihost-spec-XXX
   git branch -d feature/SPEC-XXX   # SIEMPRE -d, nunca -D
   ```
8. Verificar con `git worktree list` que ya no aparece.

**Regla crítica**: implementación y release son inseparables. Si el código está en `main`, la spec DEBE estar `released`.

**«Cerrar» una spec JAMÁS significa cancelarla.** Cancelar (`cancelled`) es decisión exclusiva del usuario y requiere la palabra explícita «cancelar» o «descartar». Ante duda, preguntar antes de cambiar cualquier estado. Precedente: SPEC-085.

**Limpieza de worktree**: solo aplica a specs `released`. Los worktrees de specs en desarrollo nunca se tocan.

---

## DELIV-CHECKRELEASE — Verificación de specs colgadas

Antes de declarar terminada una spec y después de cada release, ejecutar `/spec-check-release`:

1. Listar specs locales y su estado.
2. Para cada spec en estado ≠ `released`, buscar commits con su nombre: `git log --all --oneline --grep="SPEC-XXX" -i`.
3. Cruzar con issues: `gh issue list --label "spec/<estado>" --state open`.
4. Reportar: specs colgadas, issues sin spec local o viceversa, inconsistencias local vs GitHub.
5. Si hay colgadas: no declarar la tarea terminada.

---

## DELIV-HOOKS — Enforcement automático

- Los hooks viven en `.githooks/` y se instalan con `./scripts/install-hooks.sh` (`git config core.hooksPath .githooks`). Es idempotente.
- `pre-commit` bloquea: DBs (`*.db`, `-wal`, `-shm`), PDFs, dumps, `backups/`, `data/`; secretos y PII; `public/i18n/` editado sin su fuente (el build legítimo que staged `frontend/public/i18n/` se permite); ejecuta el guardrail de UI si hay `.tsx` staged y exige `docs/mcp/` si se toca el MCP (`DELIV-MCP-DOCS`). También exige impacto declarado y tests MCP verdes si la superficie de tools cambió (`DELIV-MCP-IMPACT`).
- `commit-msg` exige `SPEC-XXX: descripción` (o `Merge `, `Revert `, `bump `, `hotfix:`).
- El plugin `.opencode/plugins/rules-guard/` bloquea en tiempo de ejecución: borrar `data/`, `pkill`, `git push --force`, `git reset --hard`, git de escritura en `main`, `public/i18n/`, `<select>` y checkbox nativo; además, specs nuevas sin `## Impacto en tools MCP` / `Validación subagente:` y tools nuevas sin entrada en `docs/mcp/tools/impact.json`.
- **Prohibido** `git commit --no-verify` salvo autorización explícita del usuario.
- **Fail-safe**: si un hook o el plugin falla internamente, la operación continúa; solo los patrones positivos bloquean.

---

## DELIV-MCP-DOCS — Todo cambio del MCP actualiza su documentación

- Se considera **superficie MCP**:
  - `cmd/mcp-server/` (código, no tests)
  - `internal/mcp/` (código, no tests)
  - `scripts/create-api-key.sh`
  - `internal/services/bill_bulk.go` y `internal/services/api_key.go`
  - `internal/api/bill_bulk_handlers.go` y `internal/api/api_key_handlers.go`
- Si se modifica cualquiera de esos archivos, el **mismo commit** DEBE incluir cambios en `docs/mcp/`.
- El mapeo mínimo: tools nuevas o cambios de schema → `docs/mcp/tools/`; configuración/API key → `configuration.md` y `api-key.md`; operación → `operations.md`; seguridad → `security.md`; arquitectura → `architecture.md`; onboarding → `README.md`.
- **Guardrail**: `./scripts/check-mcp-docs.sh --staged`, ejecutado por `.githooks/pre-commit`. Bloquea si hay superficie MCP sin `docs/mcp/`. Los `*_test.go` del MCP no disparan la regla por sí solos.
- **Fail-safe**: si el script no existe o no es ejecutable, el hook avisa y continúa.
- **Prohibido** `git commit --no-verify` para saltear este guardrail salvo autorización explícita del usuario.

---

## DELIV-MCP-DOCS-SUBAGENT — La documentación del MCP se delega a un subagente

- Toda actualización de `docs/mcp/` DEBE delegarse a un **subagente** (`general` o `explore`) en paralelo con la implementación, para optimizar el paralelismo.
- El brief del subagente DEBE incluir: archivos cambiados, contratos/tools nuevas, variables de entorno, ejemplos sintéticos (`SEC-SYNTHETIC`) y reglas `SEC-*` aplicables.
- El agente principal **revisa e integra** la documentación; no la redacta él mismo salvo que el subagente no esté disponible, y en ese caso lo justifica en la spec o en el mensaje de commit.
- Esta regla es de proceso: no tiene guardrail automático. El cumplimiento se declara en la tabla «Cumplimiento de reglas» de la spec.

---

## DELIV-MCP-IMPACT — Impacto obligatorio en tools MCP

- Toda spec que crea o modifica una feature DEBE incluir la sección `## Impacto en tools MCP`, incluso si el veredicto es «ninguna tool afectada». El gate también valida toda spec nueva staged, aunque el cambio no toque una tool.
- La fuente de verdad es `docs/mcp/tools/impact.json`. Antes de escribir código, ejecutar:
  ```bash
  ./scripts/check-mcp-impact.sh --files <archivos previstos>
  ```
- Si hay tools afectadas, la sección DEBE declarar cada una, la ruta de dependencia (tool → cliente → endpoint → handler → servicio → storage → tabla), el impacto en el contrato (`docs/mcp/tools/contracts.json`) y los tests que se correrán.
- Si se agrega una tool: manifiesto, documentación, contrato y tests en el mismo ciclo.
- El gate de `pre-commit` es `./scripts/check-mcp-impact.sh --staged --gate`; bloquea si:
  - la tool registrada no está en el manifiesto;
  - la spec en curso no tiene la sección de impacto;
  - falta mencionar una tool afectada;
  - no hay `Validación subagente:` válida;
  - `go test ./internal/mcp/...` falla;
  - el contrato backend↔MCP (`go test ./internal/api -run TestMCPBackendContract`) falla.
- En un merge (`MERGE_HEAD` presente) el gate de spec se omite porque los commits integrados ya fueron validados; el manifiesto se valida igual.
- El sesgo es conservador: un falso positivo se resuelve declarando el impacto en la spec; un falso negativo se corrige ampliando el manifiesto.

**Precedente**: SPEC-153.

---

## DELIV-MCP-IMPACT-SUBAGENT — La validación de impacto se delega a un subagente

- Antes de completar la sección `## Impacto en tools MCP`, el agente principal DEBE delegar el análisis a un subagente (`explore` o `general`).
- El brief DEBE incluir: feature, archivos previstos, tools candidatas, endpoints, tablas, reglas `SEC-*` y el manifiesto `docs/mcp/tools/impact.json`.
- La spec DEBE registrar `Validación subagente:` con agente, sesión y veredicto. Valores como `pendiente`, `no` o `n/a` no son válidos.
- El agente principal revisa e integra el veredicto; no se autovalida.
- Esta regla es de proceso: el gate verifica la evidencia declarada, no la ejecución real del subagente.

**Precedente**: SPEC-153.

---

## DELIV-DOCKER — Docker multi-arch obligatorio

- El iHost usa ARM64 (aarch64).
- **TODAS** las imágenes Docker DEBEN ser multi-arch con 3 plataformas: `linux/amd64`, `linux/arm/v7`, `linux/arm64`.
- Comando obligatorio:
  ```bash
  docker buildx build --platform linux/amd64,linux/arm/v7,linux/arm64 --push .
  ```
- Verificar con `docker buildx imagetools inspect <image>` que existan los 3 manifests.
- Aplica a cualquier imagen del proyecto, sin excepciones.

---

## DELIV-LABELS — Labels de GitHub

- Labels válidos: `spec/draft`, `spec/pending-execution`, `spec/in-progress`, `spec/pending-release`, `spec/released`, `spec/cancelled`.
- **NUNCA** agregar labels genéricos como `spec`.
- Al cambiar de estado: primero actualizar el label, después cerrar el issue.

---

## DELIV-DEPLOY — Deploy

- El deploy a producción/iHost lo aprueba el usuario.
- Verificar el server después del deploy: `curl -s http://ihost.local:8088/health` → `{"status":"ok"}`.
- Documentar versión y fecha en la spec liberada.

---

## DELIV-ESCALATION — Escalación

Si una spec queda bloqueada más de 3 días en cualquier estado:

1. Documentar el bloqueo en la sección de riesgos.
2. Evaluar si se redefine o se cancela (solo con decisión explícita del usuario).
3. Notificar al equipo.
