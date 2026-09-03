---
title: "Fix deprecación Node.js 20 en GitHub Actions del build Docker"
id: "SPEC-052"
status: "in_progress"
author: "p40la-ihost-team"
created: "2026-09-03"
updated: "2026-09-03"
github_issue: 52
---

# Fix deprecación Node.js 20 en GitHub Actions del build Docker

**ID**: SPEC-052  
**Estado**: in_progress  
**Autor**: p40la-ihost-team  
**Creado**: 2026-09-03  
**Actualizado**: 2026-09-03

---

## 1. Resumen Ejecutivo

El workflow `.github/workflows/docker-publish.yml` (build multi-arch del add-on para iHost) emite un warning de GitHub Actions en cada ejecución: **"Node.js 20 is deprecated. The following actions target Node.js 20 but are being forced to run on Node.js 24"**. GitHub anunció la deprecación de Node.js 20 en runners el 2025-09-19, y las acciones que fijan Node 20 serán forzadas a Node 24 o dejarán de funcionar eventualmente.

El warning afecta a 5 acciones del workflow: `actions/checkout@v4`, `docker/setup-qemu-action@v3`, `docker/setup-buildx-action@v3`, `docker/login-action@v3` y `docker/build-push-action@v6`. La solución es actualizar cada una a su versión mayor que usa **Node.js 24 como runtime por defecto**.

Es un cambio de configuración de CI exclusivamente: no toca código Go, base de datos, frontend ni el runtime del iHost. No agrega dependencias ni consumo de recursos en el dispositivo; solo elimina el warning y previene que el build deje de funcionar cuando GitHub retire Node 20 por completo.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Actualizar `actions/checkout` de `v4` a `v5` en el job `build` del workflow `docker-publish.yml`.
2. **REQ-002**: Actualizar `docker/setup-qemu-action` de `v3` a `v4` en el job `build`.
3. **REQ-003**: Actualizar `docker/setup-buildx-action` de `v3` a `v4` en TODAS las apariciones (jobs `build` y `merge`).
4. **REQ-004**: Actualizar `docker/login-action` de `v3` a `v4` en TODAS las apariciones (jobs `build` y `merge`).
5. **REQ-005**: Actualizar `docker/build-push-action` de `v6` a `v7` en el job `build`.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-006**: No cambiar ningún input/opción de las actions (context, platforms, tags, cache, provenance, secrets). El bump debe ser estrictamente de versión.
2. **REQ-007**: Mantener las 3 plataformas multi-arch obligatorias (`linux/amd64`, `linux/arm/v7`, `linux/arm64`).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-008**: (Opcional) Fijar las actions por SHA inmutable en vez de tag móvil, para builds reproducibles.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Sin impacto; el tiempo de build no cambia de forma relevante.
- **Seguridad**: Actualizar a versiones mayores mantenidas reduce superficie de vulnerabilidades conocidas en el runtime de las actions.
- **Almacenamiento**: Sin impacto.
- **Disponibilidad**: Previene que el pipeline de release falle/sea forzado a Node 24 sin soporte en el futuro.
- **iHost**: Cero impacto en el dispositivo; es solo infraestructura de CI. Las imágenes multi-arch deben seguir generándose correctamente.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Warning reportado** (build `linux/amd64`): "Node.js 20 is deprecated... forced to run on Node.js 24: actions/checkout@v4, docker/build-push-action@v6, docker/login-action@v3, docker/setup-buildx-action@v3, docker/setup-qemu-action@v3".
- **Anuncio oficial**: https://github.blog/changelog/2025-09-19-deprecation-of-node-20-on-github-actions-runners/
- **`actions/checkout`**: v5 es la versión con runtime node24 (requiere Actions Runner v2.327.1+, que es el caso en runners `ubuntu-latest`).
- **`docker/setup-buildx-action`**: v4.0.0 (2026-03-05) migra a Node 24 como runtime por defecto.
- **`docker/build-push-action`**: v7.0.0 migra a Node 24 como runtime por defecto.
- **`docker/login-action` y `docker/setup-qemu-action`**: la serie v4 usa Node 24.
- Versiones objetivo verificadas en los repos oficiales de Docker y en la README de cada action (ejemplos de uso actualizados ya referencian `checkout@v5`/`checkout@v6`, `build-push-action@v7`, `setup-buildx-action@v4`).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **Bump a mayor con runtime Node 24 (v4/v5/v7)** | Elimina el warning; versiones mantenidas; inputs compatibles | Cambio de versión mayor (revisar breaking changes) | ✅ Seleccionada |
| Fijar `runs-on` a runner con Node 20 | No requiere tocar actions | Contradice el plan de GitHub; Node 20 será retirado | ❌ Rechazada |
| Fijar actions por SHA (REQ-008) | Reproducible y seguro | Requiere actualizar SHA en cada release de las actions | ❌ No en MVP; se documenta como deseable |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Bump estricto de versión mayor sin cambios de inputs**
- **Contexto**: Las actions Docker `v3/v6` ya fueron deprecadas en su runtime y GitHub las fuerza a Node 24.
- **Decisión**: Actualizar solo el tag de versión en cada `uses:` (checkout v4→v5, qemu v3→v4, buildx v3→v4, login v3→v4, build-push v6→v7). No se modifican `with:` ni `run:`.
- **Consecuencias**: Warning eliminado; comportamiento del build idéntico; el manifest multi-arch se sigue generando con las 3 plataformas.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[.github/workflows/docker-publish.yml]
       │  bump de versiones (Node 24)
       ▼
[actions/checkout@v5]
[docker/setup-qemu-action@v4]  ──▶ linux/amd64
[docker/setup-buildx-action@v4] ──▶ linux/arm/v7   (multi-arch, 3 plataformas)
[docker/login-action@v4]         ──▶ linux/arm64
[docker/build-push-action@v7]
       │
       ▼
[merge job] ──▶ manifest multi-arch + tags latest
```

### 4.2 Componentes

#### 4.2.1 `.github/workflows/docker-publish.yml` (modificar)
- **Responsabilidad**: Build y push multi-arch de la imagen Docker para iHost.
- **Interfaz**: Sin cambios de contrato; solo se actualizan los tags de las actions.
- **Dependencias**: `actions/checkout@v5`, `docker/setup-qemu-action@v4`, `docker/setup-buildx-action@v4`, `docker/login-action@v4`, `docker/build-push-action@v7`.
- **Ubicación**: `.github/workflows/docker-publish.yml`.

### 4.3 Modelo de datos

Sin cambios. No aplica.

### 4.4 APIs / Contratos

Sin cambios. No aplica.

### 4.5 Dependencias

- **Internas**: Solo `.github/workflows/docker-publish.yml`.
- **Externas**: Actions de GitHub (checkout v5, docker actions v4/v7) con runtime Node 24. Ninguna dependencia nueva para el proyecto.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: `actions/checkout` queda en `@v5` en el workflow.
- [ ] CA-002: `docker/setup-qemu-action` queda en `@v4`.
- [ ] CA-003: `docker/setup-buildx-action` queda en `@v4` en los jobs `build` y `merge`.
- [ ] CA-004: `docker/login-action` queda en `@v4` en los jobs `build` y `merge`.
- [ ] CA-005: `docker/build-push-action` queda en `@v7`.
- [ ] CA-006: No se alteró ningún input (`with:`) ni paso `run:` del workflow.

### 5.2 No funcionales

- [ ] CA-NF-001: Al ejecutar el workflow (tag `v*`), el warning "Node.js 20 is deprecated" desaparece de los logs del job `build` (`linux/amd64`).
- [ ] CA-NF-002: Las 3 plataformas (`linux/amd64`, `linux/arm/v7`, `linux/arm64`) se construyen y el manifest multi-arch se crea correctamente.

### 5.3 Testing

- **Unit tests**: No aplica (solo YAML de CI).
- **Integration tests**: Ejecutar el workflow en GitHub Actions con un tag de prueba (`v0.x.y`) y verificar que los 3 jobs `build` pasan sin warnings de Node 20.
- **E2E tests**: Verificar en Docker Hub que el manifest multi-arch (`imagetools inspect`) contiene los 3 manifests.
- **Carga/Performance**: No aplica.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Editar `docker-publish.yml`: bump de las 5 actions a versiones Node 24 | 0.25 días | Ninguna |
| 2 | Validar YAML (parseo local) y ejecutar workflow con tag de prueba en GitHub Actions | 0.25 días | Fase 1 |
| 3 | Verificar en Docker Hub los 3 manifests del manifest multi-arch | 0.25 días | Fase 2 |

### 6.2 Milestones

1. **MVP**: Bump de versiones de las 5 actions y workflow sin warnings de Node 20.
2. **V1.0**: Verificación del manifest multi-arch con las 3 plataformas en Docker Hub.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Alguna action v4/v7 cambió inputs de forma breaking | Baja | Medio | Revisar release notes de cada bump; el workflow usa inputs básicos estables (context, platforms, tags, cache, provenance) |
| El build con Node 24 rompe por runtime más estricto | Baja | Medio | Las actions v4/v7 están diseñadas para Node 24; ejecutar un tag de prueba antes de un release real |
| Olvidar una aparición de `buildx`/`login` en el job `merge` | Media | Bajo | Revisar que TODAS las apariciones se actualicen (CA-003/CA-004) |

## 8. Notas y Referencias

- Anuncio de deprecación Node 20: https://github.blog/changelog/2025-09-19-deprecation-of-node-20-on-github-actions-runners/
- `actions/checkout` v5 (runtime node24): https://github.com/actions/checkout
- `docker/build-push-action` v7: https://github.com/docker/build-push-action
- `docker/setup-buildx-action` v4: https://github.com/docker/setup-buildx-action
- `docker/login-action` v4: https://github.com/docker/login-action
- `docker/setup-qemu-action` v4: https://github.com/docker/setup-qemu-action
- SPEC-027: Script de release automático para Docker Hub con bump de versión.
- AGENTS.md: regla de multi-arch con 3 plataformas obligatorias (`linux/amd64` + `linux/arm/v7` + `linux/arm64`).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-03 | p40la-ihost-team | Creación inicial de la especificación. |
| 2026-09-03 | p40la-ihost-team | Cambio de estado a `in_progress` para inicio de desarrollo. |