---
title: "Aislamiento estricto de sesiones: worktrees obligatorios y main de solo lectura"
id: "SPEC-066"
status: "released"
author: "paulomcnally"
created: "2026-09-04"
updated: "2026-09-04"
github_issue: 67
---

# Aislamiento estricto de sesiones: worktrees obligatorios y main de solo lectura

**ID**: SPEC-066  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-09-04  
**Actualizado**: 2026-09-04

---

## 1. Resumen Ejecutivo

Durante el cierre de SPEC-064 ocurrió una colisión multi-sesión: dos sesiones de opencode operaban simultáneamente sobre el **checkout principal** (`main`). Una sesión ejecutó `git reset`/commits sobre `main` mientras la otra (esta) tenía cambios sin commitear allí, lo que dejó los commits propios huérfanos y mezcló el trabajo. El contenido se recuperó íntegro, pero la situación es exactamente la que AGENTS.md prohíbe y demuestra que **los worktrees solo protegen si se usan**: la regla existente no es suficiente porque nada impide que una sesión trabaje en `main`.

Esta spec endurece el flujo de trabajo para que el aislamiento sea **mecánico, no voluntario**:

1. **`main` pasa a ser de solo lectura para agentes**: ninguna sesión edita código ni ejecuta `git commit`/`reset`/`checkout` en el checkout principal. Todo el trabajo de una spec vive en su worktree (`p40la-ihost-spec-XXX`).
2. **La spec se crea dentro del worktree**, no en `main`: primero `/spec worktree SPEC-XXX`, y allí se crean el archivo de spec, README y código. El tracker README se actualiza **solo en el release** (al mergear); el estado vivo lo lleva el issue de GitHub.
3. **Guardas en scripts**: `new-worktree.sh` falla si `main` tiene cambios sin commitear, y un nuevo `scripts/check-worktree.sh` aborta al arrancar una sesión que no está en un worktree.

Consideraciones iHost: sin impacto en runtime (cambios de proceso/docs/scripts de desarrollo). No toca el build multi-arch ni la DB.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: **Regla dura "main de solo lectura"** documentada en AGENTS.md: ninguna sesión de opencode edita código ni ejecuta git ops destructivas/commit en el checkout principal (`/home/paulomcnally/github/p40la-ihost`). Todo commit de una spec sale de su worktree vía merge de release.
2. **REQ-002**: **Crear la spec dentro del worktree**: cambiar el flujo para que `docs/specs/SPEC-XXX.md` y el README tracker se creen/editen **dentro** del worktree de la spec (rama `feature/SPEC-XXX`), no en `main`. El README tracker se actualiza al hacer release (merge a `main`); el issue de GitHub es el tracker de estado mientras la spec está abierta.
3. **REQ-003**: **`new-worktree.sh` valida que `main` esté limpio**: si `git status` del checkout principal tiene cambios sin commitear (tracked o untracked), el script **falla con mensaje claro** pidiendo commitear/revertir antes de crear el worktree.
4. **REQ-004**: **Script de arranque `scripts/check-worktree.sh`**: verifica que la sesión está corriendo en un worktree de spec (ruta con sufijo `-spec-XXX` **o** rama `feature/SPEC-*`). Si está en el checkout principal (`main`) o en otra rama, **aborta** con instrucciones de cómo crear/usar el worktree. Este script se ejecuta automáticamente al iniciar la sesión (referenciado en AGENTS.md como paso obligatorio).
5. **REQ-005**: **Actualizar la skill `spec-manager`** (`.opencode/skills/spec-manager/SKILL.md`) y `AGENTS.md` para reflejar el nuevo flujo: worktree obligatorio antes de tocar código, espec creada en el worktree, y los comandos `/spec worktree` / guardas como parte del ciclo normal.
6. **REQ-006**: **README tracker actualizado solo en release**: quitar la obligación de actualizar `docs/specs/README.md` al momento de crear la spec. La creación solo crea el archivo de spec + issue de GitHub (que ya refleja el estado vía labels). Al liberar, el merge trae la fila del tracker + contadores.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-007**: **Hook de git protector en `main`** (`.git/hooks/pre-commit`, `pre-reset`, etc. o config recomendada) que bloquee commits directos en `main` con mensaje de advertencia, dejando los merges de release como única vía. Opcional si interfiere con flujos manuales del usuario.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-008**: Comando `/spec ensure-worktree` en la skill que envuelva `check-worktree.sh` + `new-worktree.sh` para crear y saltar al worktree en un solo paso desde cualquier checkout.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Sin impacto en la app; scripts de desarrollo se ejecutan una vez por sesión (< 100ms).
- **Seguridad**: Sin cambios de permisos; los guards son locales al repo.
- **Almacenamiento**: Sin cambios de esquema.
- **Disponibilidad**: Sin impacto en runtime del iHost.
- **iHost**: Cambios solo de proceso de desarrollo (docs + scripts). El build multi-arch (`linux/amd64`, `linux/arm/v7`, `linux/arm64`) no se toca.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Incidente SPEC-064 (2026-09-04)**: dos sesiones en el checkout principal; una hizo `git reset --hard <commit>` y commits `git add -A`, dejando huérfanos los commits de la otra sesión. Recuperado vía contenido preservado en el commit de la otra sesión, pero el riesgo es estructural.
- AGENTS.md ya documenta: "Cada ventana/sesión de opencode trabaja en SU PROPIO git worktree" y prohíbe `checkout/reset/switch/stash/clean` sobre el principal. Sin embargo, **nada impide que una sesión se abra en el checkout principal** ni que haga commits directos allí, y el flujo actual crea la spec en `main` (forzando a tocarlo).
- `scripts/new-worktree.sh` (2026): crea `feature/SPEC-XXX` desde `origin/main`. No valida que `main` esté limpio. El flujo actual de creación de spec (skill spec-manager) escribe `docs/specs/SPEC-XXX.md` y `docs/specs/README.md` **antes** de crear el worktree, lo que deja cambios sin commitear en `main`.
- La skill expone `/spec worktree <ID>` pero la creación de spec no lo invoca por defecto; es un paso manual posterior.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **Worktrees obligatorios + main read-only (esta spec)** | Aislamiento mecánico, cero cambios de infra | Requiere disciplina de proceso + scripts | ✅ Seleccionada |
| Confiar solo en la regla escrita (estado actual) | Cero trabajo | Ya falló en SPEC-064; depende de la voluntad | ❌ Rechazada |
| Rama por sesión en el mismo checkout (sin worktree) | Sin directorios extra | Sigue compartiendo working tree; checkout de ramas pisa archivos | ❌ Rechazada |
| Servidor de builds / CI por PR | Garantía de integración | Proceso pesado para un repo chico; no evita colisiones locales | ❌ Rechazada |
| **Spec creada en el worktree (flujo)** | `main` nunca se ensucia con docs de specs en curso | El README tracker no refleja specs abiertas hasta el release (se compensa con issue de GitHub) | ✅ Seleccionada |
| Hook git que bloquea commits/resets en `main` | Defensa extra | Puede molestar flujos manuales de release | ⚪ P1 (opcional) |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: `main` es de solo lectura para agentes.**
- **Contexto**: La colisión de SPEC-064 ocurrió porque dos sesiones escribieron sobre el mismo checkout.
- **Decisión**: Ningún agente edita código ni ejecuta git ops de escritura en el checkout principal. Los únicos cambios a `main` provienen de merges de release (que idealmente ejecuta el usuario por CLI).
- **Consecuencias**: Elimina la fuente de colisiones. Requiere que toda sesión se abra en un worktree (guardado por script).

**ADR-002: La spec y su código nacen en el worktree; el README tracker solo se toca al release.**
- **Contexto**: El flujo actual crea la spec en `main`, ensuciándolo con cambios sin commitear.
- **Decisión**: `/spec worktree SPEC-XXX` es el primer paso de toda spec. Dentro del worktree se crean `SPEC-XXX.md`, las ediciones de README y el código. Al liberar, el merge a `main` lleva la fila del tracker y los contadores.
- **Consecuencias**: `main` queda limpio salvo releases. El estado de las specs abiertas se consulta en GitHub (labels) o con `/spec list`.

**ADR-003: Guardas mecánicas en scripts en vez de solo reglas escritas.**
- **Contexto**: La regla escrita no se cumplió; hace falta prevención automática.
- **Decisión**: `new-worktree.sh` valida `main` limpio; `check-worktree.sh` aborta sesiones fuera de un worktree. Ambos se referencian en AGENTS.md como obligatorios.
- **Consecuencias**: El costo de la prevención recae en scripts livianos; los releases manuales siguen siendo posibles para el usuario.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Arranque de sesión opencode]
        |
        v
[scripts/check-worktree.sh]  --falla si NO está en p40la-ihost-spec-XXX / feature/SPEC-*
        |
        v
[Crear spec]  (dentro del worktree)
   1. /spec worktree SPEC-XXX  (valida main limpio)
   2. Crear docs/specs/SPEC-XXX.md + issue GitHub (labels = estado vivo)
   3. Desarrollar código en el worktree
        |
        v
[Release]  (merge feature/SPEC-XXX -> main)
   - El merge trae: código + SPEC-XXX.md + README tracker (fila + contadores)
   - Actualizar labels GitHub + cerrar issue
```

### 4.2 Componentes

#### 4.2.1 `scripts/new-worktree.sh` (modificar)
- **Responsabilidad**: Crear el worktree aislado; ahora valida que el checkout principal esté limpio antes de crear.
- **Interfaz**: `./scripts/new-worktree.sh <SPEC-ID> [nombre-rama]`
- **Cambio**: Agregar chequeo `git -C <root> status --porcelain` → si no está vacío, fallar con mensaje instructivo.
- **Ubicación**: `scripts/new-worktree.sh`.

#### 4.2.2 `scripts/check-worktree.sh` (nuevo)
- **Responsabilidad**: Verificar que la sesión corre en un worktree de spec.
- **Interfaz**: `./scripts/check-worktree.sh` (exit 0 ok / exit 1 aborta con instrucciones).
- **Lógica**: Aceptar si `git rev-parse --show-toplevel` coincide con un worktree registrado cuya rama empieza con `feature/SPEC-` (o el directorio termina en `-spec-XXX`). Rechazar si es el checkout principal o cualquier otro.
- **Ubicación**: `scripts/check-worktree.sh`.

#### 4.2.3 `AGENTS.md` y `.opencode/skills/spec-manager/SKILL.md` (modificar)
- **Responsabilidad**: Documentar el nuevo flujo y referenciar las guardas como obligatorias.
- **Cambios**: Sección "Regla CRÍTICA: Worktrees por sesión" reforzada con "main = solo lectura" + paso de arranque `check-worktree.sh`; en la skill, el flujo de creación ahora empieza con `/spec worktree`.

#### 4.2.4 (P1) Hook de git opcional
- **Responsabilidad**: Bloquear commits directos en `main`.
- **Ubicación**: `.git/hooks/pre-commit` (local, no versionado) o documentación de instalación.

### 4.3 Modelo de datos

Sin cambios. Proceso y scripts de desarrollo únicamente.

### 4.4 APIs / Contratos

Sin cambios de API. Los scripts son la interfaz nueva:
- `./scripts/check-worktree.sh` → exit 0 (ok) / exit 1 + mensaje (aborta).
- `./scripts/new-worktree.sh <SPEC-ID>` → sigue creando el worktree, pero ahora falla si `main` está sucio.

### 4.5 Dependencias

- **Internas**:
  - `scripts/new-worktree.sh` (modificar)
  - `AGENTS.md` (regla main read-only + paso de arranque)
  - `.opencode/skills/spec-manager/SKILL.md` (flujo: worktree primero, README solo al release)
  - `docs/specs/README.md` (nota de que el tracker se actualiza solo en release)
- **Externas**: Ninguna. `git` (ya disponible).

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado una sesión que corre en el checkout principal (`main`), cuando se ejecuta `./scripts/check-worktree.sh`, entonces aborta con exit 1 y mensaje instructivo.
- [ ] CA-002: Dado una sesión en `p40la-ihost-spec-XXX` (rama `feature/SPEC-XXX`), cuando se ejecuta `./scripts/check-worktree.sh`, entonces pasa con exit 0.
- [ ] CA-003: Dado el checkout principal con cambios sin commitear, cuando se ejecuta `./scripts/new-worktree.sh SPEC-XXX`, entonces falla con mensaje claro (no crea el worktree).
- [ ] CA-004: Dado el checkout principal limpio, cuando se ejecuta `./scripts/new-worktree.sh SPEC-XXX`, entonces crea el worktree como antes.
- [ ] CA-005: AGENTS.md documenta la regla "main de solo lectura para agentes" y el paso obligatorio de `check-worktree.sh` al arrancar sesión.
- [ ] CA-006: La skill `spec-manager` refleja el flujo: crear la spec **dentro del worktree** y actualizar el README tracker **solo al release** (el issue de GitHub es el tracker de estado en curso).

### 5.2 No funcionales

- [ ] CA-NF-001: Los scripts no introducen dependencias nuevas (solo bash + git).
- [ ] CA-NF-002: No hay cambios en el runtime de la app, build multi-arch ni DB.
- [ ] CA-NF-003: Documentación en español, consistente con AGENTS.md.

### 5.3 Testing

- **Unit tests**: Probar manualmente los dos scripts (main limpio/sucio, worktree sí/no, rama correcta/incorrecta).
- **Integration tests**: Flujo completo: crear worktree → crear spec dentro → desarrollar → release por merge, sin tocar `main` en el medio.
- **E2E tests**: Reproducir el escenario de SPEC-064 (dos sesiones) y verificar que los guards impiden la colisión.
- **Carga/Performance**: N/A (scripts de una sola ejecución).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Modificar `scripts/new-worktree.sh` (validar main limpio) | 0.25 día | Ninguna |
| 2 | Crear `scripts/check-worktree.sh` (guard de arranque) | 0.25 día | Ninguna |
| 3 | Actualizar AGENTS.md (main read-only + paso de arranque) | 0.25 día | Fases 1-2 |
| 4 | Actualizar skill `spec-manager` (flujo: worktree primero, README solo al release) | 0.5 día | Fase 3 |
| 5 | Pruebas manuales de los guards + validación con el usuario | 0.25 día | Fases 1-4 |
| 6 | (P1) Hook opcional de bloqueo de commits en `main` | 0.25 día | Fase 3 |

### 6.2 Milestones

1. **MVP**: Fases 1-4 (guards + docs) — el flujo queda mecánicamente protegido.
2. **V1.0**: Fase 6 (hook opcional) + validación del flujo completo de una spec de ejemplo.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Los guards son evadibles (un usuario fuerza la sesión en main) | Media | Medio | Regla documentada + el guard avisa; el release manual es del usuario |
| Cambiar el flujo de creación de spec rompe hábitos existentes | Media | Medio | README tracker se mantiene, solo se mueve su actualización al release; se documenta en la skill |
| `check-worktree.sh` bloquea sesiones legítimas | Baja | Bajo | Lógica clara (worktree `-spec-XXX` o rama `feature/SPEC-*`); mensaje instructivo |
| El hook pre-commit en main bloquea releases del usuario | Baja | Bajo | Hook P1/opcional; se puede instalar solo en el checkout principal o documentar excepción |

## 8. Notas y Referencias

- Incidente: cierre de SPEC-064 (2026-09-04) — colisión multi-sesión en el checkout principal.
- Reglas actuales: AGENTS.md sección "Regla CRÍTICA: Worktrees por sesión (multi-ventana)".
- Script existente: `scripts/new-worktree.sh`.
- Skill: `.opencode/skills/spec-manager/SKILL.md` (comando `/spec worktree`).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-04 | paulomcnally | Creación inicial de la especificación. Propuesta tras el incidente de colisión multi-sesión en SPEC-064: worktrees obligatorios, main de solo lectura para agentes, guards mecánicos en scripts y creación de specs dentro del worktree. |
| 2026-09-04 | paulomcnally | Implementación (en worktree `p40la-ihost-spec-066`): `check-worktree.sh` (guard de arranque), `new-worktree.sh` valida que `main` esté limpio, AGENTS.md con regla "main de solo lectura" y skill spec-manager con flujo worktree-primero (README tracker solo al release). Guards probados: worktree→OK, main→aborta, main sucio→aborta. |
| 2026-09-04 | paulomcnally | **Release**: merge fast-forward de `feature/SPEC-066` a `main` (commit `ea25a76`). Validado por el usuario en local. Estado `released`. |