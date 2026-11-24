# 10 — Arquitectura y stack

> **Dominio**: stack, capas, estructura de carpetas, convenciones de código, límites del runtime.
> **Cargar cuando**: se diseña cualquier solución técnica.

---

## ARCH-STACK — Stack obligatorio

| Capa | Tecnología | Motivo |
|---|---|---|
| Lenguaje backend | Go | Binario nativo, bajo consumo de RAM, stdlib robusta |
| Base de datos | SQLite | Sin proceso separado, zero-config |
| HTTP | `net/http` de la stdlib | Suficiente para la API, sin framework |
| Frontend | React + Tailwind CSS, build con Vite | Componentes reutilizables, build fuera del iHost |
| Migraciones | Scripts SQL en `migrations/` | Ligeras y predecibles |
| Deploy | Add-on Docker para iHost | Contenedor ligero |

**Prohibido**: agregar dependencias pesadas sin justificarlo en la spec. **Prohibido**: Node.js en runtime del iHost (solo para build).

---

## ARCH-FOLDERS — Estructura de carpetas

```
/
├── cmd/server/            # Entrypoint: solo arranca el servidor
├── internal/
│   ├── api/               # Handlers HTTP (código legacy y router central)
│   ├── config/            # Configuración y env vars
│   ├── db/                # Conexión SQLite y migraciones
│   ├── models/            # Structs de dominio compartidos
│   ├── modules/           # Módulos nuevos con namespace (ARCH-MODULES)
│   ├── services/          # Lógica de negocio (código legacy)
│   └── storage/           # Queries SQLite (código legacy)
├── frontend/              # Código fuente React (build fuera del iHost)
├── public/                # Salida del build servida por el backend
├── migrations/            # SQL versionado
├── docs/                  # Specs y reglas
├── scripts/               # Utilidades de desarrollo y operación
├── AGENTS.md
├── Dockerfile
└── go.mod
```

---

## ARCH-LAYERS — Capas obligatorias

1. **`cmd/server/main.go`**: solo arranca el servidor. Sin lógica de negocio.
2. **`internal/api/`**: solo parsea requests, llama a services y devuelve responses.
3. **`internal/services/`**: única capa con lógica de negocio.
4. **`internal/storage/`**: única capa que habla con SQLite.
5. **`internal/models/`**: structs de dominio, sin dependencias de DB ni HTTP.

**Prohibido**:
- Lógica de negocio en handlers.
- SQL en services o handlers.
- Dependencias de HTTP dentro de models.

> Las capas siguen vigentes **dentro** de cada módulo de `internal/modules/` (ver `ARCH-MODULES`), donde `handlers.go`, `service.go`, `storage.go` y `model.go` son archivos separados del mismo paquete.

---

## ARCH-MODULES — Módulos con namespace canónico

Inspirado en los plugins de integración (`<proveedor>.<país>`, ej. `assa.nicaragua`). Aplica **solo a módulos nuevos**; el código existente en `internal/{api,services,storage,models}` no se migra.

### Identificador

| Tipo | Gramática | Ejemplo |
|---|---|---|
| Core (agnóstico) | `<dominio>.<producto>[.<capacidad>][.<acción>]` | `bank.creditcard` |
| Proveedor | `<dominio>.<proveedor>.<país>.<producto>[.<capacidad>][.<acción>]` | `bank.bac.ni.creditcard.statement.import` |

- Segmentos en minúsculas: `[a-z][a-z0-9_]*`.
- País: ISO 3166-1 alpha-2 en minúsculas (`ni`, `pa`).
- Máximo 6 segmentos. El ID es **inmutable** una vez publicado.

### Carpetas

La ruta espeja el ID desde `internal/modules/`:

```
internal/modules/bank/creditcard/                                   # bank.creditcard
internal/modules/bank/bac/ni/creditcard/statement/import/           # bank.bac.ni.creditcard.statement.import
```

### Archivos obligatorios por módulo

| Archivo | Responsabilidad |
|---|---|
| `module.go` | `Info()` con ID, versión semver, tipo, proveedor, país, nombre, descripción y migraciones que usa; registro en `init()` |
| `model.go` | Structs y constantes de dominio del módulo |
| `storage.go` | Único lugar con SQL del módulo |
| `service.go` | Única capa con lógica de negocio |
| `handlers.go` | Solo parseo de request/response HTTP |
| `*_test.go` | Tests del módulo |

### Registro y rutas

- `internal/modules/registry.go` define `Info`, `Module`, `Registry` y las interfaces de capacidad (ej. `StatementImporter`, `Router`).
- `internal/modules/all/register.go` es el **único punto de entrada** que importa los módulos y los registra (patrón `automation-plugins/all` y `analyzers/all`).
- `cmd/server/main.go` construye el registry con las dependencias (`*sql.DB`) y lo pasa a `internal/api`.
- `internal/api/routes.go` solo invoca el registro de rutas del módulo; el módulo monta sus propios paths.
- Migraciones: los `.sql` viven en `migrations/` (fuente única, `DATA-MIGRATIONS`) y el módulo los declara en `Info().Migrations`.

### Dependencias

- Dirección obligatoria: `handlers → service → storage → model`.
- Un módulo **no** importa `storage` ni `handlers` de otro módulo. La comunicación es por contratos en `internal/modules` o structs en `internal/models`.
- Un módulo puede usar servicios compartidos de `internal/services` (moneda, formato) sin duplicarlos.

**Guardrail propuesto**: `modules` (regla `ARCH-MODULES`) → `.opencode/skills/spec-manager/scripts/check-modules.sh .` — se implementa en SPEC-137; hasta entonces la verificación es manual y se declara en la spec.

---

## ARCH-MCP-TOOLS-MAP — Manifiesto de tools MCP

- Toda tool registrada con `mcp.AddTool` DEBE tener una entrada en `docs/mcp/tools/impact.json`.
- El manifiesto declara, por tool: `docs`, `entrypoints`, `dependencies`, `endpoints` y `tests`. En `shared.affects_all` declara los archivos o directorios cuyo cambio afecta a todas las tools (rutas, middleware, auth, modelos, migraciones, SDK).
- Las dependencias pueden ser archivos exactos, directorios con `/` final o globs (`internal/services/bill*.go`).
- El registro DEBE usar `mcp.AddTool(server, &mcp.Tool{Name: "..."}, ...)` con el nombre literal bajo `internal/mcp/tools/`; el analizador lee el nombre con `go/parser` y no resuelve variables.
- Al agregar, renombrar o eliminar una tool: actualizar el manifiesto, `docs/mcp/tools/contracts.json`, `docs/mcp/` y la spec en el mismo ciclo.
- El manifiesto es la fuente de verdad para el análisis de impacto; no se duplica en la spec.

**Guardrail**: `mcp-impact` (regla `DELIV-MCP-IMPACT`) → `./scripts/check-mcp-impact.sh --check-map`. Falla si una tool registrada no está en el manifiesto, si el manifiesto declara una tool inexistente o si una ruta declarada no existe.

**Precedente**: SPEC-153.

---

## ARCH-GO — Convenciones de código Go

- `gofmt` siempre.
- Nombres en inglés en el código; español en specs y documentación.
- Manejo de errores explícito: nunca ignorar un `error`.
- Configuración por variables de entorno con defaults razonables.
- Logger con `log/slog` de la stdlib.

---

## ARCH-IHOST — Límites del runtime

- El iHost tiene recursos limitados: priorizar bajo uso de memoria y mínimas dependencias.
- El binario y el frontend se construyen **fuera** del iHost.
- El estado persistente vive en un volumen montado, nunca dentro del contenedor.
- No hay dependencias de red externas obligatorias.
- El frontend se sirve como archivos estáticos pre-compilados.
- Imagen Docker base: `gcr.io/distroless/static` o `alpine:latest` si se necesita shell.
- Puerto por defecto: `8088` (configurable con `PORT`).
- Health check ligero: `GET /health`.

---

## ARCH-CONFIG — Configuración de OpenCode del proyecto

- `AGENTS.md` es la entrada auto-cargada.
- `opencode.json` declara `instructions` con `AGENTS.md` y el glob `docs/rules/*.md`.
- Las reglas viven en `docs/rules/`; no se duplican en la skill.

**Guardrail**: `build` (regla `TEST-LOCAL`) → `go build ./...`.
