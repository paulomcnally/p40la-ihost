---
title: "Desarrollo de tools"
sidebar_position: 6
description: "Cómo agregar una tool nueva al servidor MCP"
---

# Desarrollo de tools

## Estructura

```
cmd/mcp-server/                 # entrypoint stdio
internal/mcp/
├── config/                     # variables de entorno
├── api/                        # cliente HTTP + tipos DTO + interfaz API
├── registry/                   # montaje de conjuntos de tools
└── tools/
    ├── all/register.go         # único punto que importa los dominios
    └── bills/                  # tools de facturas
        ├── tools.go            # Toolset + Register
        ├── results.go          # helpers de resultado/error
        ├── period.go           # períodos YYYY-MM
        ├── list.go
        ├── generate.go
        └── pay.go
```

## Agregar una tool

1. **Definí el contrato HTTP** en el backend si la operación requiere lógica de negocio. La tool no debe contener SQL ni reglas de dominio.
2. **Agregá el método** en `internal/mcp/api` (tipo DTO + método del cliente + interfaz `API`).
3. **Creá el paquete de tools** `internal/mcp/tools/<dominio>` con un `Toolset` y un `Register`.
4. **Registrá el dominio** en `internal/mcp/tools/all/register.go` con un `r.Add(...)`.
5. **Escribí tests** con una API falsa y, si aplica, un test de protocolo con `mcp.NewInMemoryTransports`.
6. **Documentá** la tool en `docs/mcp/tools/<dominio>.md` y actualizá el catálogo.

## Contrato de una tool

```go
type ExampleInput struct {
    ServiceID int64   `json:"service_id" jsonschema:"ID del servicio (obligatorio)"`
    Note      *string `json:"note,omitempty" jsonschema:"Texto opcional"`
}

func (t *Toolset) example(ctx context.Context, _ *mcp.CallToolRequest, input ExampleInput) (*mcp.CallToolResult, any, error) {
    if input.ServiceID <= 0 {
        return toolError("invalid_request", "service_id debe ser mayor que cero", "")
    }
    result, err := t.client.Example(ctx, input.ServiceID)
    if err != nil {
        return apiError(err)
    }
    return jsonResult(result)
}
```

Reglas:

- Los campos obligatorios no llevan `omitempty`; los opcionales usan punteros.
- Las descripciones de `jsonschema` son la documentación que ve el modelo.
- Los errores de negocio devuelven `IsError=true` con `{error, message, hint}`.
- Los errores de la API se traducen con `apiError` para no filtrar detalles.
- Las escrituras deben pasar por `writeMode` y respetar `P40LA_MCP_ALLOW_WRITES`.
- Toda tool que escribe debe ser idempotente o explicar por qué no lo es.

## Registro

```go
// internal/mcp/tools/all/register.go
func Register(r *registry.Registry, client api.API, cfg config.Config) {
    r.Add(func(server *mcp.Server) {
        bills.Register(server, client, cfg)
    })
    // r.Add(func(server *mcp.Server) { debts.Register(server, client, cfg) })
}
```

`cmd/mcp-server` no conoce los paquetes de dominio: solo monta el registry.

## Tests

- **Cliente HTTP**: `httptest` con login, cookie, 401 + relogin y mapeo de errores.
- **Tool**: API falsa que captura requests y devuelve respuestas; reloj fijo para períodos.
- **Protocolo**: `initialize`, `tools/list` y `tools/call` con transporte en memoria.
- **Backend**: DB temporal con migraciones; nunca la DB local del usuario.

```bash
go test ./internal/mcp/...
go test ./internal/services/ ./internal/api/
go build ./...
```

## Checklist de una tool nueva

- [ ] Input schema claro y con campos obligatorios/opcionales correctos.
- [ ] Descripción con límites, defaults y requisitos de escritura.
- [ ] Errores con código estable.
- [ ] Tests de éxito, validación y error de API.
- [ ] Documentación en `docs/mcp/tools/`.
- [ ] Sin secretos, PII ni montos reales en tests o docs.
- [ ] `go build ./...` y `go test ./...` pasan.
