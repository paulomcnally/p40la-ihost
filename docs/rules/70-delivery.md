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
