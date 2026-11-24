# Imagen del nodo Tailscale dedicado al MCP (SPEC-148).
# Incluye el serve.json con Funnel 443 -> MCP para que el iHost no necesite
# editar archivos dentro de volúmenes.
FROM tailscale/tailscale:latest
COPY serve.json /config/serve.json
ENV TS_SERVE_CONFIG=/config/serve.json
ENV TS_STATE_DIR=/var/lib/tailscale
ENV TS_USERSPACE=true
