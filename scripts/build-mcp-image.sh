#!/usr/bin/env bash
#
# Construye y publica la imagen multi-arch del servidor MCP (SPEC-147).
# Requiere: docker buildx, GH_TOKEN con lectura del módulo privado.
#
# Uso:
#   GH_TOKEN=... ./scripts/build-mcp-image.sh
#   IMAGE=ghcr.io/usuario/p40la-mcp TAG=0.5.17 ./scripts/build-mcp-image.sh
#
# Plataformas obligatorias (DELIV-DOCKER): amd64, arm/v7, arm64.

set -euo pipefail

IMAGE="${IMAGE:-ghcr.io/paulomcnally/p40la-mcp}"
TAG="${TAG:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
VERSION="${VERSION:-${TAG}}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm/v7,linux/arm64}"

if ! command -v docker >/dev/null 2>&1; then
  echo "ERROR: docker no está instalado" >&2
  exit 1
fi

secret_args=()
if [[ -n "${GH_TOKEN:-}" ]]; then
  secret_args=(--secret "id=gh_token,env=GH_TOKEN")
fi

docker buildx build \
  --platform "${PLATFORMS}" \
  --build-arg "VERSION=${VERSION}" \
  "${secret_args[@]}" \
  -f Dockerfile.mcp \
  -t "${IMAGE}:${TAG}" \
  --push \
  .

echo "Imagen publicada: ${IMAGE}:${TAG}"
echo "Verificá los manifests: docker buildx imagetools inspect ${IMAGE}:${TAG}"
