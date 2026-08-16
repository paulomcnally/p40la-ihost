# p40la-ihost

Add-on para SONOFF iHost (eWeLink CUBE) para gestionar servicios, facturas, autos e instituciones.

---

## Releasing a new Docker version via tag

Para publicar una nueva imagen Docker multi-arquitectura (`linux/amd64`, `linux/arm/v7`, `linux/arm64`) en Docker Hub, seguí estos pasos desde una terminal:

### 1. Actualizar la versión en `docker-compose.yml`

```bash
# Editá docker-compose.yml y cambiá la variable VERSION, por ejemplo:
# VERSION=0.2.2
```

### 2. Commitear el cambio de versión

```bash
git add docker-compose.yml
git commit -m "bump version to 0.2.2"
git push origin main
```

### 3. Crear y pushear el tag

```bash
export VERSION=0.2.2
git tag -a v$VERSION -m "Release v$VERSION"
git push origin v$VERSION
```

> El workflow `docker-publish.yml` se dispara automáticamente con cualquier tag `v*`. Construye las 3 arquitecturas, crea el manifest multi-arch y publica:
> - `paulomcnally/p40la-ihost:$VERSION`
> - `paulomcnally/p40la-ihost:latest`

### 4. Verificar el release (opcional)

```bash
# Esperá ~2 minutos y verificá que existan los 3 manifests
docker buildx imagetools inspect paulomcnally/p40la-ihost:$VERSION
```

Deberías ver algo como:

```
Name:      docker.io/paulomcnally/p40la-ihost:0.2.2
MediaType: application/vnd.docker.distribution.manifest.list.v2+json
Digest:    sha256:...

Manifests:
  Name:      docker.io/paulomcnally/p40la-ihost:0.2.2-amd64
  Platform:  linux/amd64
  ...
  Name:      docker.io/paulomcnally/p40la-ihost:0.2.2-armv7
  Platform:  linux/arm/v7
  ...
  Name:      docker.io/paulomcnally/p40la-ihost:0.2.2-arm64
  Platform:  linux/arm64
  ...
```

---

## Instalación en SONOFF iHost

Ver [docs/ihost.md](docs/ihost.md).

---

## Desarrollo local

```bash
# Backend
go run ./cmd/server

# Frontend
cd frontend
npm install
npm run dev
```

El servidor expone la API en `http://localhost:8088`.

---

## Estructura del proyecto

- `cmd/server/` — punto de entrada del backend Go
- `internal/` — API, servicios, storage y modelos
- `frontend/` — aplicación React + Tailwind + Vite
- `migrations/` — migraciones SQLite
- `docs/` — documentación y specs
