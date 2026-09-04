---
title: "CI/CD con GitHub Actions: build multi-arch por tag"
id: "SPEC-015"
status: "released"
author: "paulomcnally"
created: "2026-08-15"
updated: "2026-09-04 (released)"
github_issue: 59
---

# CI/CD con GitHub Actions: build multi-arch por tag

**ID**: SPEC-015  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-08-15  
**Actualizado**: 2026-09-04 (released)  

---

## 1. Resumen Ejecutivo

El build de Docker multi-arch (amd64, arm64, armv7) es extremadamente lento en la máquina de desarrollo (~20 min por cross-compilation con QEMU). Esta spec configura GitHub Actions para:

- Construir y publicar automáticamente cuando se crea un tag `v*`
- Multi-arch nativo usando GitHub Actions runners (mucho más rápido que QEMU local)
- Versionamiento semántico extraído del tag

## 2. Requerimientos

### P0 - Obligatorios
1. Build multi-arch (linux/amd64, linux/arm/v7, linux/arm64)
2. Trigger solo en tag push (no en cada commit a main)
3. Extraer versión del tag y usarla como Docker tag
4. Push a Docker Hub

### P1 - Importantes
1. Cache de capas Docker entre builds
2. Build del frontend dentro del container
3. Health check en la imagen final

## 3. Diseño Técnico

### Trigger
```
on:
  push:
    tags: ["v*"]
```

### Flujo
1. Checkout del código
2. Set up Docker Buildx
3. Login a Docker Hub
4. Build y push multi-arch con tag de versión

### Versionamiento
- Tag `v0.2.3` → Docker image `paulomcnally/p40la-ihost:0.2.3`
- Tag `v0.2.3` → Docker image `paulomcnally/p40la-ihost:latest`

## 4. Criterios de Aceptación

- [ ] Al hacer `git tag v0.3.0 && git push origin v0.3.0` se construye la imagen
- [ ] La imagen tiene 3 plataformas: amd64, arm64, armv7
- [ ] La imagen se publica en Docker Hub con tag de versión
- [ ] La imagen se publica también como `latest`

## 5. Archivos a crear

- `.github/workflows/docker-publish.yml`

## 6. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-15 | paulomcnally | Creación inicial de la especificación: CI/CD con GitHub Actions, build multi-arch por tag `v*`. |
| 2026-08-15 | paulomcnally | Implementación: workflow `docker-publish.yml`, Dockerfile a Go 1.24, `.dockerignore` y `scripts/build.sh` (commit `e78c32a`). Evolucionada luego por SPEC-027 (script de release con bump) y SPEC-056 (build nativo ARM64). |
| 2026-09-04 | paulomcnally | Release. Workflow en uso en producción (releases a Docker Hub con tags `v*`). Código en `main`.
