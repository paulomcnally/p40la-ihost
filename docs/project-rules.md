# Reglas del Proyecto — p40la-ihost

> **Última actualización**: 2026-08-12  
> **Vigente para**: Todo el repositorio `p40la-ihost`

---

## 1. Stack Tecnológico Obligatorio

El proyecto corre en un **iHost con recursos limitados**, por lo que el stack se eligió priorizando bajo consumo de memoria, pocas dependencias y facilidad de despliegue como add-on.

| Capa | Tecnología | Justificación |
|------|-----------|---------------|
| **Lenguaje** | Go | Binario nativo, bajo consumo de RAM, stdlib robusta |
| **Base de datos** | SQLite | Sin proceso separado, zero-config, ideal para iHost |
| **HTTP server** | `net/http` de Go | Incluido en stdlib, suficiente para API simple |
| **Frontend** | HTML + CSS + JS vanilla | Mínimo peso, sin build step ni dependencias pesadas |
| **Migraciones DB** | `golang-migrate` o scripts SQL manuales | Ligero y predecible |
| **Deploy** | Add-on Docker para iHost | Empaquetado como contenedor ligero |

**Regla**: Ningún framework pesado (Echo, Gin, Next.js, React, etc.) sin aprobación explícita y actualización de esta regla.

---

## 2. Estructura de Carpetas

```
/
├── cmd/
│   └── server/              # Punto de entrada del backend
│       └── main.go
├── internal/
│   ├── api/                 # Handlers HTTP y rutas
│   ├── config/              # Configuración y variables de entorno
│   ├── db/                  # Conexión SQLite y migraciones
│   ├── models/              # Entidades y structs de dominio
│   ├── services/            # Lógica de negocio
│   └── storage/             # SQLite queries y abstracción DB
├── public/                  # Frontend estático servido por el backend
│   ├── index.html
│   ├── css/
│   └── js/
├── docs/
│   ├── specs/               # Especificaciones técnicas
│   ├── project-rules.md     # Este archivo
│   └── infrastructure.md    # Deploy e infraestructura
├── migrations/              # Scripts SQL de migración
├── Dockerfile               # Contenedor ligero para iHost
├── docker-compose.yml       # Opcional, para desarrollo local
├── go.mod
└── README.md
```

---

## 3. Reglas de Arquitectura

### 3.1 Capas obligatorias

1. **`cmd/server/main.go`**: solo arranca el servidor. Sin lógica de negocio.
2. **`internal/api/`**: handlers HTTP. **Solo** parsean requests, llaman a services y devuelven responses.
3. **`internal/services/`**: única capa con lógica de negocio.
4. **`internal/storage/`**: única capa que interactúa con SQLite.
5. **`internal/models/`**: structs de dominio, sin dependencias de DB ni HTTP.

### 3.2 Prohibido

- Lógica de negocio en handlers.
- Queries SQL directamente desde services o handlers.
- Dependencias pesadas sin justificación en una spec.
- Frontend con framework SPA (React, Vue, etc.) salvo que una spec lo autorice.

---

## 4. Convenciones de Código Go

- Usar `gofmt` siempre.
- Nombres en inglés para código, español para specs y documentación.
- Manejo de errores explícito: nunca ignorar `error`.
- Configuración vía variables de entorno, con valores por defecto razonables.
- Logger simple: `log/slog` de stdlib.

---

## 5. Base de Datos

- SQLite con `journal_mode=WAL` para mejor concurrencia.
- Migraciones en `migrations/` con naming `NNNN_descripcion.up.sql` / `.down.sql`.
- No usar ORM. SQL crudo con `database/sql` y helpers propios livianos.

---

## 6. Consideraciones de iHost

- Imagen Docker base: `gcr.io/distroless/static` o `alpine:latest` si se necesita shell.
- Puerto por defecto: `8080` (configurable vía `PORT`).
- SQLite debe persistir en volumen montado (no dentro del contenedor).
- Health check ligero: `GET /health`.
- Sin dependencias de red externa obligatorias.
