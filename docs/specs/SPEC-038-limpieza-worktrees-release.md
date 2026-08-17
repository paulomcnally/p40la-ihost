---
title: "Limpieza automática de worktrees al liberar una spec"
id: "SPEC-038"
status: "in_progress"
author: "paulomcnally"
created: "2026-08-16"
updated: "2026-08-16"
github_issue: 38
---

# Limpieza automática de worktrees al liberar una spec

**ID**: SPEC-038  
**Estado**: in_progress  
**Autor**: paulomcnally  
**Creado**: 2026-08-16  
**Actualizado**: 2026-08-16

---

## 1. Resumen Ejecutivo

Cuando una spec llega al estado `released`, su rama `feature/SPEC-XXX` ya fue mergeada a `main` y el worktree asociado (`p40la-ihost-spec-XXX`) queda huérfano: un directorio aislado cuyo checkout ya no se usa. Si nadie lo limpia, se acumulan directorios y ramas locales que confunden a las sesiones y a `git worktree list`.

Se propone formalizar en el flujo de release (skill `spec-manager` y `AGENTS.md`) que **al liberar una spec se elimine su worktree y se borre la rama local mergeada**. El objetivo es mantener el entorno de worktrees siempre limpio: cada spec activa en desarrollo tiene exactamente un worktree, y las specs liberadas no dejan residuos.

El cambio es **solo de proceso/documentación**: no toca código de la app, DB, API ni frontend. No hay impacto en iHost.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Al completar el release de una spec (estado `released`), ejecutar la limpieza del worktree: `git worktree remove <ruta>` y eliminar la rama local mergeada (`git branch -d feature/SPEC-XXX`).
2. **REQ-002**: Documentar la limpieza como paso obligatorio del flujo de release en `AGENTS.md` (sección de worktrees) y en la skill `spec-manager` (sección Release y comando `/spec worktree`).
3. **REQ-003**: La limpieza solo aplica a worktrees de specs **liberadas**. Los worktrees de specs en desarrollo (`draft`/`in_progress`/`pending_release`) NO se tocan.
4. **REQ-004**: La rama solo se elimina si ya fue mergeada (`git branch -d` falla si no lo está, protegiendo contra pérdida accidental).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-005**: Documentar el comando manual de limpieza en la skill como referencia (`git worktree remove`, `git branch -d`), por si el usuario prefiere limpiar manualmente en lugar de automático.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-006**: Opcional: el script `new-worktree.sh` podría registrar un mensaje recordatorio de limpieza al crear el worktree.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Sin impacto (operaciones git puntuales).
- **Seguridad**: `git branch -d` (no `-D`) evita borrar ramas sin mergear.
- **Almacenamiento**: Libera espacio en disco al eliminar checkouts huérfanos.
- **Disponibilidad**: Sin cambios en endpoints ni en la app.
- **iHost**: N/A (proceso local de desarrollo, no corre en iHost).

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- AGENTS.md, regla "Worktrees por sesión": cada ventana/sesión trabaja en su PROPIO worktree; para liberar una spec se mergea la rama a `main`.
- Skill `spec-manager`, sección "Release" (pasos 1-8) y comando `/spec worktree`.
- Experiencia SPEC-036: tras liberar la spec, quedaron en disco `p40la-ihost-spec-036` (worktree) y la rama `feature/SPEC-036` (absorbida por main), sin limpieza automática documentada.
- `git worktree remove` elimina el checkout; `git branch -d` elimina la rama solo si está fully merged (seguro).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Limpieza manual documentada en el release | Simple, sin automatización | Depende del agente recordarlo | ❌ Rechazada |
| Paso obligatorio del flujo de release (skill + AGENTS.md) | Siempre se ejecuta, entorno limpio | Requiere disciplina del agente | ✅ Seleccionada |
| Script automatizado que detecte worktrees huérfanos | Cero esfuerzo manual | Más código, riesgo de borrar algo en uso | ❌ Rechazada |
| Borrar rama con `-D` | Elimina siempre | Puede perder ramas sin mergear | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: La limpieza es parte del release, no un script separado
- **Contexto**: El release ya ejecuta una secuencia obligatoria (frontmatter, label, issue, commits). La limpieza encaja como un paso más.
- **Decisión**: Agregar la limpieza como paso final del release en la skill y en AGENTS.md.
- **Consecuencias**: Entorno de worktrees siempre consistente; el agente no puede olvidarlo porque es parte del checklist.

**ADR-002**: Usar `git branch -d` (nunca `-D`)
- **Contexto**: La rama debe eliminarse solo si ya fue mergeada a main.
- **Decisión**: `git branch -d` falla si la rama no está mergeada, protegiendo contra pérdida.
- **Consecuencias**: Si el merge aún no llegó a la rama local, el comando avisa en vez de borrar.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Spec liberada (released)]
        │
        ▼
1. Merge feature/SPEC-XXX → main (ya existente)
        │
        ▼
2. git worktree remove ../p40la-ihost-spec-XXX   ← NUEVO
        │
        ▼
3. git branch -d feature/SPEC-XXX                 ← NUEVO
        │
        ▼
4. Verificar: git worktree list sin el worktree
```

### 4.2 Componentes

#### 4.2.1 AGENTS.md — Regla "Worktrees por sesión"
- **Responsabilidad**: Documentar la limpieza como parte del release.
- **Ubicación**: `AGENTS.md`, sección "Regla CRÍTICA: Worktrees por sesión".

#### 4.2.2 Skill `spec-manager` — Sección Release y `/spec worktree`
- **Responsabilidad**: Documentar la limpieza como paso obligatorio del release y el comando manual.
- **Ubicación**: `.opencode/skills/spec-manager/SKILL.md`.

### 4.3 Modelo de datos

N/A (no hay DB).

### 4.4 APIs / Contratos

N/A (no hay API).

### 4.5 Dependencias

- **Internas**: `AGENTS.md`, `.opencode/skills/spec-manager/SKILL.md`.
- **Externas**: `git worktree`, `git branch` (CLI estándar).

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: El flujo de release de la skill incluye la limpieza del worktree y la rama como paso obligatorio.
- [ ] CA-002: AGENTS.md documenta que al liberar una spec se ejecuta `git worktree remove` + `git branch -d`.
- [ ] CA-003: La skill documenta el comando manual de limpieza como referencia.
- [ ] CA-004: La limpieza solo aplica a specs liberadas; specs en desarrollo no se ven afectadas.

### 5.2 No funcionales

- [ ] CA-NF-001: El comando documentado usa `git branch -d` (no `-D`), protegiendo ramas sin mergear.

### 5.3 Testing

- **Unit tests**: N/A (documentación).
- **Integration tests**: Verificar en local que tras liberar SPEC-036 se pueda eliminar su worktree y rama con los comandos documentados.
- **E2E tests**: Ejecutar el flujo de release completo de una spec dummy y verificar que el worktree desaparece.
- **Carga/Performance**: N/A.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Actualizar skill `spec-manager` (sección Release + `/spec worktree`) | 0.25 día | Ninguna |
| 2 | Actualizar AGENTS.md (regla de worktrees) | 0.25 día | Ninguna |
| 3 | Validación: liberar una spec dummy y verificar limpieza | 0.25 día | Fase 1, 2 |

### 6.2 Milestones

1. **MVP**: Documentación de la limpieza en skill + AGENTS.md.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Borrar un worktree de una spec aún activa | Baja | Alto | La limpieza se ejecuta SOLO al pasar a `released`, y usa `git worktree remove` que falla si hay cambios sin commitear |
| Borrar rama sin mergear | Baja | Alto | `git branch -d` (no `-D`) falla si la rama no está fully merged |
| AGENTE olvide la limpieza | Media | Medio | Es paso obligatorio del checklist de release en skill + AGENTS.md |

## 8. Notas y Referencias

- Experiencia SPEC-036: worktree `p40la-ihost-spec-036` y rama `feature/SPEC-036` quedaron huérfanos tras el release.
- `git worktree list` muestra todos los worktrees y sus ramas.
- `git worktree remove <ruta>` y `git branch -d <rama>` son operaciones locales, sin push.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-16 | paulomcnally | Creación inicial de la especificación |
