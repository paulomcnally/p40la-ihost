#!/usr/bin/env bash
#
# SPEC-129: genera los íconos PWA a partir de frontend/public/favicon.svg.
# Los PNG resultantes se commitean; este script es para regenerarlos.
#
# Requisitos: ImageMagick (convert). No se ejecuta en el build ni en runtime.
#
# Uso: ./scripts/generate-pwa-icons.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
SRC="${ROOT_DIR}/frontend/public/favicon.svg"
OUT="${ROOT_DIR}/frontend/public/icons"

command -v convert >/dev/null 2>&1 || {
  echo "ERROR: ImageMagick (convert) no está instalado" >&2
  exit 1
}

[[ -f "${SRC}" ]] || {
  echo "ERROR: no existe ${SRC}" >&2
  exit 1
}

mkdir -p "${OUT}"

# Íconos normales: fondo transparente, esquinas redondeadas del SVG.
convert -background none "${SRC}" -resize 192x192 -depth 8 "${OUT}/icon-192.png"
convert -background none "${SRC}" -resize 512x512 -depth 8 "${OUT}/icon-512.png"

# Maskable: fondo azul a sangre y el rayo centrado al ~62% para respetar la
# safe zone del 40% central (el resto puede recortarse).
convert -size 512x512 xc:'#007AFF' \
  \( -background none "${SRC}" -resize 320x320 \) \
  -gravity center -composite -depth 8 "${OUT}/icon-maskable-512.png"

# Apple touch icons: opacos, sin transparencia (iOS aplica su propia máscara).
for size in 180 167 152; do
  inner=$((size * 62 / 100))
  convert -size "${size}x${size}" xc:'#007AFF' \
    \( -background none "${SRC}" -resize "${inner}x${inner}" \) \
    -gravity center -composite -depth 8 "${OUT}/apple-touch-icon-${size}.png"
done

# Compatibilidad con el archivo histórico.
cp "${OUT}/apple-touch-icon-180.png" "${ROOT_DIR}/frontend/public/apple-touch-icon.png"

echo "Íconos PWA generados en ${OUT}"
file "${OUT}"/*.png
