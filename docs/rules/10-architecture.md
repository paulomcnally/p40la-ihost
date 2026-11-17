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
│   ├── api/               # Handlers HTTP
│   ├── config/            # Configuración y env vars
│   ├── db/                # Conexión SQLite y migraciones
│   ├── models/            # Structs de dominio
│   ├── services/          # Lógica de negocio
│   └── storage/           # Queries SQLite
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
