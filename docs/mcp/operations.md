---
title: "Operación"
sidebar_position: 6
description: "Build, ejecución, API key, diagnóstico y respaldo del servidor MCP"
---

# Operación

## Build

```bash
# Binario local
go build -o bin/p40la-mcp ./cmd/mcp-server

# Versión embebida
go build -ldflags="-X main.version=0.1.0" -o bin/p40la-mcp ./cmd/mcp-server

# Multi-plataforma (ejemplo)
GOOS=linux GOARCH=arm64 go build -o bin/p40la-mcp-linux-arm64 ./cmd/mcp-server
GOOS=darwin GOARCH=arm64 go build -o bin/p40la-mcp-darwin-arm64 ./cmd/mcp-server
```

El binario MCP **no** se incluye en la imagen Docker del servidor ni se inicia con el add-on. Se ejecuta en la máquina del cliente MCP y se conecta por HTTP al iHost. Esto evita sumar RAM y procesos al dispositivo.

## Provisionar la API key

```bash
EMAIL="demo@example.com" PASSWORD="demo-password" ./scripts/create-api-key.sh
```

La key se imprime una sola vez. Guardala en un gestor de secretos o en un archivo `0600`:

```bash
EMAIL="demo@example.com" PASSWORD="demo-password" \
KEY_FILE="$HOME/.config/p40la/api-key" \
./scripts/create-api-key.sh
```

La contraseña se usa solo para esta operación; el MCP no la necesita. Detalle en [api-key.md](api-key.md).

## Ejecución

```bash
P40LA_BASE_URL="http://ihost.local:8088" \
P40LA_API_KEY="p40la_..." \
./bin/p40la-mcp
```

El proceso espera mensajes JSON-RPC por stdin. Todo log va a stderr para no romper el protocolo.

## Verificación previa

```bash
curl -s http://ihost.local:8088/health
# {"status":"ok","timestamp":"..."}

# Verificar que la key responde
curl -s -H "Authorization: Bearer $P40LA_API_KEY" \
  http://ihost.local:8088/api/services | head
```

Si el health falla, el MCP no puede operar. Revisá red, URL base y que el contenedor esté corriendo.

## Rotación y revocación

- Rotar: `POST /api/api-key/regenerate` o el script. La key anterior deja de funcionar de inmediato.
- Revocar: `DELETE /api/api-key`.
- Después de rotar, actualizá `P40LA_API_KEY` en el cliente MCP y reiniciá el proceso.

## Procedimiento de backfill

Ejemplo para un servicio mensual, enero a la fecha:

1. `bills_list` con `service_id` para ver qué existe.
2. `bills_generate` con `from=2026-01`, `to=2026-11`, `dry_run=true` (default). Debe reportar 11 planificadas si no hay ninguna.
3. Mostrar el resumen al usuario y pedir confirmación.
4. Respaldar la DB del iHost según `docs/rules/40-data.md`.
5. Repetir con `dry_run=false` y `P40LA_MCP_ALLOW_WRITES=true`.
6. Verificar con `bills_list` que hay 11 facturas y que una muestra `source=mcp` en su historial.

No generes meses futuros: el backend los rechaza. No repitas a ciegas si hubo un error parcial: la operación es idempotente y el segundo intento reporta lo ya creado como `skipped`.

## Diagnóstico

| Síntoma | Causa probable | Acción |
|---|---|---|
| `auth_failed` | Key inválida, revocada o rotada | Regenerar la key y actualizar `P40LA_API_KEY` |
| `writes_disabled` | Compuerta apagada | Definir `P40LA_MCP_ALLOW_WRITES=true` o usar `dry_run=true` |
| `upstream_error` | URL, red o servidor caído | Probar `/health`, revisar `P40LA_BASE_URL` |
| `service_not_found` | `service_id` incorrecto | Verificar en la UI o con `bills_list` |
| `service_inactive` | Servicio desactivado | Activarlo antes de generar |
| `amount_unresolved` | Sin monto explícito, sin factura previa y sin monto sugerido | Pasar `amount` o configurar el servicio |
| `deleted_slot` | Factura eliminada ocupando el período | Restaurarla o eliminar el registro desde la UI; no se recrea sola |
| Timeout | Red lenta o servidor cargado | Subir `P40LA_TIMEOUT` hasta 60s y revisar el iHost |

## Recursos

- El binario no abre la DB ni mantiene caché en disco.
- Una generación de 11 facturas hace una sola llamada HTTP y el backend procesa secuencialmente.
- Límite de 36 períodos por generación.
- No hay reintentos automáticos de escrituras.

## Tests

```bash
go build ./...
go test ./internal/services/ ./internal/api/ ./internal/mcp/...
```

Los tests de integración usan una DB temporal (`t.TempDir()`), nunca `data/app.db`.

## Logs

- Nivel informativo al iniciar: versión, URL base, si las escrituras están habilitadas y la zona horaria.
- Sin API key, montos, emails ni cuerpos de request/response.
- Los errores HTTP se muestran al agente con código y mensaje, no en logs con payload.
