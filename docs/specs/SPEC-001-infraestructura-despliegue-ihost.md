---
title: "Infraestructura de Despliegue en SONOFF iHost: Docker, Volumen de Datos y Multi-Arquitectura"
id: "SPEC-001"
status: "pending_release"
author: "p40la-ihost-team"
created: "2026-08-12"
updated: "2026-08-12"
---

# Infraestructura de Despliegue en SONOFF iHost: Docker, Volumen de Datos y Multi-Arquitectura

**ID**: SPEC-001  
**Estado**: draft  
**Autor**: p40la-ihost-team  
**Creado**: 2026-08-12  
**Actualizado**: 2026-08-12

---

## 1. Resumen Ejecutivo

El proyecto `p40la-ihost` debe ejecutarse como un add-on Docker dentro de un SONOFF iHost (eWeLink CUBE). El iHost es un dispositivo con recursos limitados: procesador ARM de 32/64 bits, 2 GB o 4 GB de RAM, y almacenamiento interno compartido con el sistema operativo y otros add-ons.

Esta especificación define la infraestructura de empaquetado y despliegue: imagen Docker multi-arquitectura, persistencia de datos en un volumen montado en `/app/data`, configuración por variables de entorno, y scripts de build/publicación alineados con el flujo de iHost. El objetivo es que el add-on sea fácil de instalar, actualizar y mantener sin perder datos entre versiones.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: El proyecto debe empaquetarse como una imagen Docker que corra en iHost.
2. **REQ-002**: La imagen debe soportar las arquitecturas `linux/arm/v7`, `linux/arm64` y `linux/amd64`.
3. **REQ-003**: La base de datos SQLite y cualquier dato de configuración deben persistir en el volumen `/app/data`.
4. **REQ-004**: El contenedor debe exponer un puerto HTTP configurable (por defecto `8000`).
5. **REQ-005**: Debe existir un `Dockerfile` optimizado para tamaño y memoria, usando Go compilado.
6. **REQ-006**: Debe existir un `docker-compose.yml` para desarrollo/local.
7. **REQ-007**: Debe existir un script `./scripts/push-dockerhub.sh <version>` para publicar imagen multi-arch en Docker Hub.
8. **REQ-008**: El script de publicación debe usar versionado semántico (`MAJOR.MINOR.PATCH`) y **rechazar publicar un tag que ya existe** en Docker Hub o Git.
9. **REQ-009**: Debe existir documentación en `docs/ihost.md` con pasos de instalación en iHost.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-010**: Imagen base lo más liviana posible (distroless o alpine) para reducir consumo en iHost.
2. **REQ-011**: Health check HTTP ligero (`GET /health`) que iHost pueda usar para monitorear el contenedor.
3. **REQ-012**: Variables de entorno para: `PORT`, `DATA_DIR`, `LOG_LEVEL`.
4. **REQ-013**: Script `./scripts/run-local.sh` para levantar el proyecto en desarrollo sin Docker.
5. **REQ-014**: Script `./scripts/run-docker.sh` para levantar el proyecto con Docker en local.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-015**: Compatibilidad con network mode `host` además de `bridge` en iHost.
2. **REQ-016**: `.dockerignore` optimizado para no incluir archivos innecesarios en la imagen.
3. **REQ-017**: GitHub Actions para build y push automático en cada tag.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Imagen final menor a 30 MB comprimida. Tiempo de arranque menor a 3 segundos en iHost.
- **Memoria**: Consumo en idle menor a 20 MB de RAM.
- **Seguridad**: No exponer secrets en la imagen. No correr como root si es posible.
- **Disponibilidad**: Restart policy `unless-stopped`. Health check cada 30 segundos.
- **iHost**: Sin dependencias de red externa obligatorias. Todo offline-capable.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

El iHost ejecuta contenedores Docker sobre una arquitectura ARM y expone una UI para administrar add-ons. Los add-ons típicos requieren:

- Imagen multi-arquitectura que soporte ARMv7, ARM64 y AMD64.
- Volumen persistente para datos, montado en una ruta conocida dentro del contenedor.
- Puerto HTTP configurable y documentado.
- Publicación en un registro accesible desde el iHost (Docker Hub).
- Guía de instalación paso a paso para el usuario final.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **Go + distroless/static** | Imagen muy pequeña (<20 MB), sin shell, bajo ataque surface | Sin shell para debugging | ✅ Seleccionada para runtime |
| **Go + alpine** | Imagen pequeña (~25 MB), shell disponible para debug | Mayor surface que distroless | Considerar si se necesita debug en iHost |
| **Python + slim** (como referencia) | Familiar, fácil | Mayor imagen, mayor memoria, runtime pesado | ❌ Rechazada por consumo |
| **Node.js + alpine** | Ecosistema grande | Imagen y memoria más pesadas | ❌ Rechazada por consumo |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Imagen Docker multi-arquitectura
- **Contexto**: iHost usa ARMv7, pero el build y pruebas locales pueden ser ARM64 o AMD64.
- **Decisión**: Soportar `linux/arm/v7`, `linux/arm64`, `linux/amd64` en Docker Hub.
- **Consecuencias**: Build más lento, pero un solo tag funciona en todos los entornos.

**ADR-002**: Volumen persistente en `/app/data`
- **Contexto**: SQLite debe sobrevivir a actualizaciones del contenedor.
- **Decisión**: Montar volumen de iHost en `/app/data`. La app crea el directorio si no existe.
- **Consecuencias**: Actualizaciones no pierden datos. El usuario debe crear el volumen en iHost Docker UI.

**ADR-003**: Puerto por defecto 8000
- **Contexto**: Puerto poco común, fácil de recordar y con baja probabilidad de conflicto en iHost.
- **Decisión**: Puerto `8000` por defecto, configurable vía `PORT`.
- **Consecuencias**: El usuario puede cambiarlo si hay conflicto con otro add-on.

**ADR-004**: Network mode `host` recomendado
- **Contexto**: En iHost, `host` permite que el add-on alcance otros servicios locales (Node-RED, etc.) sin configurar IPs.
- **Decisión**: Documentar `host` como recomendado, `bridge` como alternativa.
- **Consecuencias**: Menos aislamiento, pero mayor simplicidad para el usuario.

**ADR-005**: Versionado semántico estricto para imágenes Docker
- **Contexto**: Se publicarán muchas imágenes a Docker Hub. Si se sobrescribe un tag publicado, los iHost pueden quedarse con una imagen cacheada inconsistente.
- **Decisión**: Usar Semantic Versioning (`MAJOR.MINOR.PATCH`). El script de publicación verifica que el tag de Git exista y que el tag de imagen no exista ya en Docker Hub. Nunca se sobrescribe `latest` sin también publicar una versión explícita.
- **Consecuencias**: Cada cambio publicado requiere bump de versión. Mayor previsibilidad para usuarios y para rollback.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
┌─────────────────────────────────────┐
│           SONOFF iHost              │
│  ┌─────────────────────────────┐    │
│  │   Docker add-on: p40la      │    │
│  │   ┌─────────────────────┐   │    │
│  │   │  Binario Go         │   │    │
│  │   │  HTTP :8000         │   │    │
│  │   └─────────────────────┘   │    │
│  │            │                │    │
│  │            ▼                │    │
│  │   ┌─────────────────────┐   │    │
│  │   │  /app/data/app.db   │   │    │
│  │   │  (volumen Docker)   │   │    │
│  │   └─────────────────────┘   │    │
│  └─────────────────────────────┘    │
└─────────────────────────────────────┘
```

### 4.2 Archivos a crear/modificar

#### `Dockerfile`

```dockerfile
# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server ./cmd/server

# Runtime stage
FROM gcr.io/distroless/static-debian12
WORKDIR /app
ENV DATA_DIR=/app/data
ENV PORT=8000
COPY --from=builder /app/server /app/server
COPY --from=builder /app/public /app/public
EXPOSE 8000
VOLUME ["/app/data"]
CMD ["/app/server"]
```

#### `docker-compose.yml`

```yaml
services:
  p40la-ihost:
    build: .
    image: paulomcnally/p40la-ihost:latest
    container_name: p40la-ihost
    restart: unless-stopped
    ports:
      - "8000:8000"
    volumes:
      - p40la-ihost-data:/app/data
    environment:
      - LOG_LEVEL=info

volumes:
  p40la-ihost-data:
```

#### `scripts/push-dockerhub.sh`

Script para publicar imagen multi-arch en Docker Hub. Debe:

- Validar que se pase un argumento de versión (`MAJOR.MINOR.PATCH`).
- Validar que exista un tag de Git local con esa versión.
- Validar que el tag de imagen `paulomcnally/p40la-ihost:<version>` no exista ya en Docker Hub.
- Construir y pushear tanto `paulomcnally/p40la-ihost:<version>` como `paulomcnally/p40la-ihost:latest`.
- Usar `docker buildx` con plataformas `linux/arm/v7`, `linux/arm64`, `linux/amd64`.

#### `docs/ihost.md`

Guía de instalación paso a paso en iHost, incluyendo creación de volumen, puerto, network y actualización.

### 4.3 Variables de entorno

| Variable | Default | Descripción |
|----------|---------|-------------|
| `PORT` | `8000` | Puerto HTTP del servidor |
| `DATA_DIR` | `/app/data` | Directorio de datos persistentes |
| `LOG_LEVEL` | `info` | Nivel de log (`debug`, `info`, `warn`, `error`) |

### 4.4 Health check

- Endpoint: `GET /health`
- Respuesta: `{"status":"ok","timestamp":"..."}`
- Docker healthcheck: `HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD ["/app/server", "-healthcheck"]` o curl si la imagen lo incluye.

### 4.5 Dependencias

- **Internas**: Ninguna (primera spec, no hay código previo).
- **Externas**: Docker, Docker Hub, Go 1.23, `gcr.io/distroless/static`.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: `docker build .` genera una imagen funcional en local.
- [ ] CA-002: `docker-compose up` levanta el contenedor y expone el puerto 8000.
- [ ] CA-003: El contenedor crea `/app/data` automáticamente si no existe.
- [ ] CA-004: SQLite persiste en `/app/data` entre reinicios del contenedor.
- [ ] CA-005: `./scripts/push-dockerhub.sh 0.1.0` publica imagen multi-arch.
- [ ] CA-006: `docs/ihost.md` incluye pasos claros de instalación en iHost.

### 5.2 No funcionales

- [ ] CA-NF-001: Imagen comprimida menor a 30 MB.
- [ ] CA-NF-002: Consumo de RAM en idle menor a 20 MB.
- [ ] CA-NF-003: Build soporta `linux/arm/v7`, `linux/arm64`, `linux/amd64`.

### 5.3 Testing

- **Manual**: Levantar contenedor local, verificar health check, reiniciar y confirmar datos persistentes.
- **iHost**: Instalar add-on en iHost, verificar acceso web, reiniciar contenedor, verificar persistencia.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Crear `Dockerfile` multi-stage con Go + distroless | 1h | Ninguna |
| 2 | Crear `docker-compose.yml` y `.dockerignore` | 30m | Fase 1 |
| 3 | Crear scripts `run-local.sh`, `run-docker.sh`, `push-dockerhub.sh` | 1h | Fase 2 |
| 4 | Crear `docs/ihost.md` y `docs/docker.md` | 1h | Fase 3 |
| 5 | Actualizar `docs/project-rules.md` con reglas de Docker/iHost | 30m | Fase 1 |
| 6 | Probar build local y validar tamaño de imagen | 1h | Fases 1-3 |

### 6.2 Milestones

1. **MVP**: Dockerfile y docker-compose funcionando en local.
2. **V1.0**: Imagen publicada en Docker Hub e instalable en iHost.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Build multi-arch lento o falla en CI/local | Media | Medio | Usar `docker buildx` con builder dedicado; documentar setup |
| Distroless dificulta debug en iHost | Media | Bajo | Opción de imagen alpine para debug; logs detallados |
| Volumen no persiste por mal montaje en iHost | Baja | Alto | Documentación clara con capturas/pasos exactos |
| Consumo de memoria excede lo esperado | Baja | Alto | Medir con `docker stats` y optimizar código Go |

## 8. Notas y Referencias

- Documentación iHost: https://help.sonoff.tech/docs/ihost
- Semantic Versioning: https://semver.org/lang/es/
- Imagen base: https://github.com/GoogleContainerTools/distroless
- Multi-arch buildx: https://docs.docker.com/build/building/multi-platform/

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-12 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-08-12 | p40la-ihost-team | Refuerza versionado semántico y elimina referencias externas |
| 2026-08-12 | p40la-ihost-team | v0.1.0 implementada: Dockerfile, docker-compose, scripts y publicación multi-arch a Docker Hub (`paulomcnally/p40la-ihost:0.1.0`)
