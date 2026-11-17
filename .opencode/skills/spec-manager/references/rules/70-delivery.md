# 70 — Entrega: git, release y deploy

> **Dominio**: ramas, confirmaciones, release de specs, deploy y escalación.
> **Cargar cuando**: se va a commitear, mergear, liberar una spec o desplegar.

---

## DELIV-BRANCH — Ramas y aislamiento

- Estrategia del proyecto: `{{BRANCH_STRATEGY}}`.
- Si el proyecto usa worktrees por sesión: cada sesión trabaja en el suyo y el checkout principal es de solo lectura para agentes.
- Verificación al arrancar: `{{WORKTREE_CHECK_COMMAND}}`.

---

## DELIV-CONFIRM — Merge y push requieren confirmación

- **Jamás** push a ramas protegidas sin confirmación explícita del usuario.
- **Jamás** merge a la rama principal desde una rama de feature sin confirmación.
- **Jamás** `--force` ni `--force-with-lease` en ramas principales o de release.
- Solo el usuario aprueba un deploy a producción.

---

## DELIV-COMMIT — Formato de commits

- Formato: `{{COMMIT_FORMAT}}` (ej: `SPEC-XXX: descripción corta`).
- Los commits de implementación van acompañados del commit de release que documenta el estado.

---

## DELIV-RELEASE — Flujo de release

1. Merge a la rama principal con confirmación.
2. Commit y push de los cambios pendientes.
3. Documentar commit/versión y fecha en la spec.
4. Actualizar `status` (frontmatter + cuerpo).
5. Actualizar el tracker del proyecto.
6. Cambiar label y cerrar el issue.
7. Limpiar worktree/rama si el proyecto lo usa.
8. Verificar con `/spec-check-release`.

**Implementación y release son inseparables**: si el código está en la rama principal, la spec debe estar `released`.

**«Cerrar» jamás significa cancelar.** `cancelled` requiere orden explícita del usuario.

---

## DELIV-LABELS — Labels

- Labels válidos: `spec/draft`, `spec/pending-execution`, `spec/in-progress`, `spec/pending-release`, `spec/released`, `spec/cancelled`.
- Nunca labels genéricos.
- Actualizar el label antes de cerrar el issue.

---

## DELIV-DOCKER — Imágenes multi-arquitectura

- Si el proyecto publica imágenes Docker, declarar las plataformas obligatorias: `{{DOCKER_PLATFORMS}}`.
- Comando obligatorio: `{{DOCKER_BUILD_COMMAND}}`.
- Verificar los manifests con `docker buildx imagetools inspect <image>`.

> Si el proyecto no usa Docker, marcar esta sección como `N/A`.

---

## DELIV-ESCALATION — Escalación

Si una spec queda bloqueada más de `{{BLOCK_DAYS}}` días:

1. Documentar el bloqueo en riesgos.
2. Evaluar redefinición o cancelación (solo con decisión explícita del usuario).
3. Notificar al equipo.
