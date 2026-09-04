---
title: "Build nativo ARM64 con runner ubuntu-24.04-arm en GitHub Actions"
id: "SPEC-056"
status: "in_progress"
author: "p40la-ihost-team"
created: "2026-09-04"
updated: "2026-09-04"
github_issue: 56
---

# Build nativo ARM64 con runner ubuntu-24.04-arm en GitHub Actions

**ID**: SPEC-056  
**Estado**: in_progress  
**Autor**: p40la-ihost-team  
**Creado**: 2026-09-04  
**Actualizado**: 2026-09-04

---

## 1. Resumen Ejecutivo

El workflow `.github/workflows/docker-publish.yml` construye las 3 plataformas obligatorias (`linux/amd64`, `linux/arm/v7`, `linux/arm64`) en un runner x64 (`ubuntu-latest`) usando emulación QEMU vía Buildx. La plataforma `linux/arm64` se compila por emulación completa (QEMU + binfmt), lo que la hace la más lenta del pipeline y ocasionalmente propensa a fallos por timeouts o diferencias de comportamiento del emulador.

GitHub ofrece runners nativos ARM64 (`ubuntu-24.04-arm`, `ubuntu-22.04-arm`) y, desde el announcement de 2025, son **gratis e ilimitados para repositorios públicos**. El repo `paulomcnally/p40la-ihost` es público (verificado 2026-09-04), por lo que este cambio no tiene costo alguno.

Esta spec migra la construcción de `linux/arm64` a un runner nativo `ubuntu-24.04-arm`, manteniendo `linux/amd64` y `linux/arm/v7` en `ubuntu-latest` con QEMU. Resultado esperado: builds de arm64 más rápidos y confiables, y sin emulación para la arquitectura que corre en producción (el SONOFF iHost es aarch64). Es un cambio de configuración de CI exclusivamente: no toca código Go, base de datos, frontend ni el runtime del iHost, y no agrega dependencias ni consumo de recursos en el dispositivo.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: El job `build` de `docker-publish.yml` debe usar `runs-on: ubuntu-24.04-arm` para la plataforma `linux/arm64`, ejecutando el build de forma nativa sin QEMU.
2. **REQ-002**: Las plataformas `linux/amd64` y `linux/arm/v7` deben seguir corriendo en `ubuntu-latest` con QEMU.
3. **REQ-003**: Mantener intacta la regla multi-arch de 3 plataformas (`linux/amd64`, `linux/arm/v7`, `linux/arm64`) en la matriz del workflow.
4. **REQ-004**: Mantener el flujo de tags (`-amd64`/`-armv7`/`-arm64`), el job `merge` que crea el manifest multi-arch y el tag `latest`.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-005**: El paso `Set up QEMU` debe saltarse (o ser inofensivo) en el runner nativo ARM64, evitando trabajo innecesario.
2. **REQ-006**: No alterar inputs de las actions ya versionadas (checkout v5, qemu v4, buildx v4, login v4, build-push v7) salvo lo estrictamente necesario para el `runs-on` condicional.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-007**: (Opcional) Documentar en `docs/infrastructure.md` que el repo usa runners ARM64 gratuitos por ser público.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Reducir el tiempo de build de `linux/arm64` al eliminar la emulación QEMU (compilación nativa).
- **Seguridad**: Sin impacto; las actions mantienen sus versiones ya bumpadas a Node 24 (SPEC-052).
- **Almacenamiento**: Sin impacto.
- **Disponibilidad**: El job arm64 queda sujeto a la disponibilidad de runners ARM64 de GitHub; en horas pico puede haber cola, pero es lo mismo que con runners x64.
- **iHost**: Cero impacto en el dispositivo; las 3 imágenes multi-arch deben seguir generándose correctamente (el iHost usa la de arm64/aarch64).

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Documentación oficial**: https://docs.github.com/en/actions/reference/runners/github-hosted-runners — tabla "Standard GitHub-hosted runners for public repositories": Linux 4 CPU / 16 GB / 14 GB / arm64 con label `ubuntu-24.04-arm`. Uso **free e ilimitado** en repos públicos.
- **Announcement**: https://www.infoq.com/news/2025/02/github-actions-linux-arm64/ — runners Linux ARM64 en public preview; free para repos públicos; labels `ubuntu-24.04-arm`/`ubuntu-22.04-arm` **solo funcionan en repos públicos** (en privados el job falla).
- **Mantenimiento por GitHub**: https://github.com/actions/runner-images/issues/14100 — GitHub asumió el mantenimiento de las imágenes arm64 (`ubuntu-24.04-arm`), por lo que es un runner de primera clase.
- **Repos confirmado público**: `gh repo view paulomcnally/p40la-ihost` → `visibility: PUBLIC` (2026-09-04), habilitando el uso gratuito del runner ARM.
- **Precedente en repo**: SPEC-015 (CI/CD multi-arch) y SPEC-052 (bump de actions a Node 24, ya aplicado en `docker-publish.yml`).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **Runner ARM nativo para arm64 (`ubuntu-24.04-arm`)** | Build nativo sin QEMU; más rápido y confiable; gratis en repo público; arquitectura idéntica a producción (aarch64) | Runner compartido: posibles colas en horas pico | ✅ Seleccionada |
| Mantener todo en `ubuntu-latest` (x64 + QEMU) | Cero cambios; comportamiento conocido | arm64 sigue emulado (lento, propenso a timeouts) | ❌ Rechazada |
| Mover TODAS las plataformas al runner ARM | Un solo runner para todo | amd64 y arm/v7 se construirían emulados en ARM (peor que nativo x64) | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: `runs-on` condicional por plataforma vía matriz**
- **Contexto**: La matriz del job `build` itera sobre 3 plataformas; cada una debe correr en un runner distinto (arm64 en ARM nativo, el resto en x64).
- **Decisión**: Agregar un campo `runs_on` en el bloque `include:` de la matriz y usar `runs-on: ${{ matrix.runs_on }}` (soporte nativo de GitHub Actions para `runs-on` con expresión de matriz). Para `linux/arm64` → `ubuntu-24.04-arm`; para `linux/amd64` y `linux/arm/v7` → `ubuntu-latest`.
- **Consecuencias**: El job build se ejecuta en el runner adecuado por plataforma; el resto de pasos (checkout, versionado, buildx, login, push) permanece idéntico.

**ADR-002: QEMU condicional solo donde hace falta**
- **Contexto**: El runner ARM nativo no necesita QEMU para compilar arm64; el runner x64 sí lo necesita para amd64 y arm/v7.
- **Decisión**: Agregar un flag `needs_qemu` en el bloque `include:` y condicionar el paso `Set up QEMU` con `if: matrix.needs_qemu == 'true'`.
- **Consecuencias**: En el runner ARM el paso se omite; en el runner x64 se mantiene el comportamiento actual.

**ADR-003: El job `merge` permanece en `ubuntu-latest`**
- **Contexto**: El job `merge` solo crea el manifest multi-arch con `docker buildx imagetools`, sin compilar nada.
- **Decisión**: Se deja en `ubuntu-latest` (x64); no requiere runner ARM.
- **Consecuencias**: Sin impacto; sigue siendo la etapa final que ensambla los 3 tags en el manifest `latest`/versión.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[.github/workflows/docker-publish.yml]  (trigger: tags v*)
        │
        ▼  job build (matriz con runs-on por plataforma)
┌──────────────────────┬──────────────────────────┬─────────────────────────┐
│ linux/amd64          │ linux/arm/v7             │ linux/arm64             │
│ ubuntu-latest        │ ubuntu-latest            │ ubuntu-24.04-arm        │
│ + QEMU (needs_qemu)  │ + QEMU (needs_qemu)      │ sin QEMU (build nativo) │
│ tag -amd64           │ tag -armv7               │ tag -arm64              │
└──────────────────────┴──────────────────────────┴─────────────────────────┘
        │                      │                            │
        ▼                      ▼                            ▼
        └──────────── job merge (ubuntu-latest) ────────────┘
                        manifest multi-arch
                        tag <version> + latest
```

### 4.2 Componentes

#### 4.2.1 `.github/workflows/docker-publish.yml` (modificar)
- **Responsabilidad**: Build y push multi-arch de la imagen Docker para iHost, ahora con build nativo de arm64.
- **Interfaz**: Cambios solo en la definición de la matriz y el `runs-on`; el contrato de tags y manifest no cambia.
- **Dependencias**: Actions ya versionadas (checkout@v5, setup-qemu@v4, setup-buildx@v4, login@v4, build-push@v7) — sin bump adicional.
- **Ubicación**: `.github/workflows/docker-publish.yml`.

### 4.3 Modelo de datos

Sin cambios. No aplica.

### 4.4 APIs / Contratos

Sin cambios. No aplica.

### 4.5 Dependencias

- **Internas**: Solo `.github/workflows/docker-publish.yml`.
- **Externas**: GitHub Actions (runner público `ubuntu-24.04-arm`). Ninguna dependencia nueva para el proyecto ni para el iHost.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: El job `build` usa `runs-on: ${{ matrix.runs_on }}` con `ubuntu-24.04-arm` para `linux/arm64` y `ubuntu-latest` para `linux/amd64`/`linux/arm/v7`.
- [ ] CA-002: El paso `Set up QEMU` se ejecuta solo cuando `matrix.needs_qemu == 'true'` (amd64 y arm/v7), y se omite en el runner ARM.
- [ ] CA-003: La matriz conserva las 3 plataformas (`linux/amd64`, `linux/arm/v7`, `linux/arm64`) y sus tags (`-amd64`, `-armv7`, `-arm64`).
- [ ] CA-004: El job `merge` sigue creando el manifest multi-arch con tags `<version>` y `latest`.
- [ ] CA-005: El workflow se dispara con un tag `v*` de prueba y los 3 jobs `build` pasan correctamente.

### 5.2 No funcionales

- [ ] CA-NF-001: El build de `linux/arm64` corre en un runner `ubuntu-24.04-arm` (verificable en los logs del job: runner label/arch arm64).
- [ ] CA-NF-002: En Docker Hub, el manifest multi-arch (`imagetools inspect`) contiene los 3 manifests (amd64, armv7, arm64).

### 5.3 Testing

- **Unit tests**: No aplica (solo YAML de CI).
- **Integration tests**: Ejecutar el workflow con un tag de prueba (`v*`) y verificar que el job arm64 se ejecuta en runner ARM nativo y los otros dos en x64.
- **E2E tests**: Verificar en Docker Hub el manifest multi-arch con `docker buildx imagetools inspect <image>:<tag>` y confirmar los 3 manifests.
- **Carga/Performance**: Comparar tiempo de build de arm64 nativo vs el anterior emulado (expectativa: reducción).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Editar `docker-publish.yml`: agregar `runs_on` y `needs_qemu` al `include:` de la matriz, `runs-on: ${{ matrix.runs_on }}` y condicionar el paso QEMU | 0.25 días | Ninguna |
| 2 | Validar YAML (parseo local) y ejecutar workflow con tag de prueba en GitHub Actions | 0.25 días | Fase 1 |
| 3 | Verificar en Docker Hub los 3 manifests del manifest multi-arch y que arm64 corrió en runner ARM | 0.25 días | Fase 2 |

### 6.2 Milestones

1. **MVP**: Workflow con build nativo arm64 en `ubuntu-24.04-arm` y los 3 jobs pasando.
2. **V1.0**: Verificación del manifest multi-arch con las 3 plataformas en Docker Hub.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| El label `ubuntu-24.04-arm` no disponible por ser repo privado | Baja (repo es público, verificado) | Alto | Verificar visibilidad del repo; si cambia a privado, los jobs ARM fallan → documentar en la spec y volver a QEMU |
| Colas en runners ARM en horas pico | Media | Bajo | Aceptable; es el mismo comportamiento que runners x64 compartidos; el build sigue corriendo |
| `runs-on` con expresión de matriz no soportada | Baja (soporte nativo desde 2023) | Medio | Validar con tag de prueba; fallback: separar job arm64 en un job propio con `ubuntu-24.04-arm` |
| QEMU necesario también en ARM para arm/v7 si se mueve | N/A | N/A | No se mueve: arm/v7 queda en x64 con QEMU, donde ya funciona |
| Diferencias de build nativo vs emulado generen artefactos distintos | Baja | Medio | El build nativo es el deseado (igual a producción aarch64); verificar manifest y health check de la imagen |

## 8. Notas y Referencias

- GitHub-hosted runners reference (tabla public repos): https://docs.github.com/en/actions/reference/runners/github-hosted-runners
- Linux arm64 hosted runners free para public repos: https://github.com/orgs/community/discussions/148648
- InfoQ — GitHub Actions Linux ARM64: https://www.infoq.com/news/2025/02/github-actions-linux-arm64/
- Arm64 runner images mantenidas por GitHub: https://github.com/actions/runner-images/issues/14100
- SPEC-015: CI/CD con GitHub Actions: build multi-arch por tag.
- SPEC-052: Fix deprecación Node.js 20 en GitHub Actions del build Docker (actions ya en v4/v5/v7).
- AGENTS.md: regla de multi-arch con 3 plataformas obligatorias (`linux/amd64` + `linux/arm/v7` + `linux/arm64`).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-04 | p40la-ihost-team | Creación inicial de la especificación. |
| 2026-09-04 | p40la-ihost-team | Cambio de estado a `in_progress` para inicio de desarrollo. |
| 2026-09-04 | p40la-ihost-team | Implementación completada en `.github/workflows/docker-publish.yml`: matriz con `runs_on`/`needs_qemu`, `runs-on: ${{ matrix.runs_on }}` y QEMU condicional. Pendiente validación con tag de prueba. |