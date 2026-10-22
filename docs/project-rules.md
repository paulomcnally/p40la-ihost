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
| **Frontend** | React + Tailwind CSS (build estático con Vite) | Componentes reutilizables, DX superior, build fuera del iHost |
| **Migraciones DB** | `golang-migrate` o scripts SQL manuales | Ligero y predecible |
| **Deploy** | Add-on Docker para iHost | Empaquetado como contenedor ligero |

**Nota**: El build de React+Tailwind se realiza fuera del iHost (CI/CD o máquina de desarrollo). El iHost solo sirve archivos estáticos pre-build (HTML + JS bundle + CSS), sin Node.js en runtime.

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
├── frontend/                # Código fuente React (build fuera del iHost)
│   ├── src/                 # Componentes, páginas, stores
│   ├── package.json
│   └── vite.config.ts
├── public/                  # Frontend estático servido por el backend (output del build)
│   ├── index.html
│   ├── assets/
│   └── i18n/
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
- Node.js en runtime del iHost (solo para build en CI/CD o dev machine).

---

## 4. Reglas de UI

### 4.1 Patrón de creación

- **Todas** las páginas con acción de crear deben usar el componente `CreateMenu` (ícono de 3 puntos verticales) en el header.
- El menú despliega un dropdown con las opciones de creación disponibles.
- **Nunca** usar botones directos de "Crear" o "+" en el header.
- El componente `CreateMenu` se importa desde `frontend/src/components/CreateMenu.tsx`.

### 4.2 Estilo general

- Diseño inspirado en iOS: bordes redondeados (`rounded-ios`), sombras suaves (`shadow-ios`), colores del sistema Apple.
- Sidebar fijo a la izquierda con navegación por secciones.
- Header sticky con título de página y acciones a la derecha (settings, logout, create menu).
- Cards con hover effect y menú de acciones (3 puntos) que aparece al hacer hover.
- Modales de confirmación para acciones destructivas (eliminar).

### 4.3 Inputs y darkmode (obligatorio)

- **Todo** `input`, `select` y `textarea` DEBE usar los tokens de color del tema: `bg-card` para el fondo y `text-text`/`text-text-secondary` para el texto. Jamás colores hardcodeados (ej: `bg-white`, `dark:bg-[#2c2c2e]`, `text-black`).
- El color de texto de los form controls nativos lo fija la regla global en `frontend/src/index.css` (`input, select, textarea { color: rgb(var(--color-text)); background-color: rgb(var(--color-card)); color-scheme: light dark; }`). No duplicarla por formulario.
- **Todo formulario/modal nuevo debe verificarse en darkmode** antes de considerarse completo: texto y placeholders legibles (placeholder usa `--color-text-secondary`). Si se omite, la spec no pasa a `pending_release`.

### 4.4 Navegación de retorno en páginas de detalle (obligatorio)

- **Las páginas de detalle (y formularios de crear/editar) NO deben tener links "← Título" dentro del contenido.** La acción "volver a la lista" se muestra como flecha atrás en el header: en móvil reemplaza la hamburguesa que abre el sidebar (`lg:hidden`); en desktop la vuelta se hace desde el sidebar fijo.
- Para registrar una ruta con retorno, agregar una entrada al mapa `BACK_ROUTES` en `frontend/src/components/DashboardLayout.tsx` (patrón de pathname → ruta destino de la lista). Ver SPEC-063.
- En páginas de listado raíz (`/home`, `/services`, `/deudas`, `/autos`, `/settings`, etc.) el header sigue mostrando la hamburguesa en móvil: no agregar flecha atrás.
- Si se omite esta regla en una spec con páginas de detalle, la spec no pasa a `pending_release`.

### 4.5 Etiqueta de acciones de edición (obligatorio)

- **Toda acción de edición se rotula "Editar"**, sin importar la entidad ni la página: menús de 3 puntos, dropdowns, headers y botones usan la clave i18n genérica `app.edit` (`Editar` en `es`, `Edit` en `en`).
- **Prohibido** rotular acciones como "Editar servicio", "Editar deuda", "Editar casa" o "Editar [página]". Esas etiquetas confunden la acción con el título de la página y rompen la consistencia entre módulos.
- Los **títulos de formularios** sí pueden ser descriptivos ("Editar servicio", "Editar deuda") usando su clave específica; la regla aplica solo a etiquetas de acciones.
- Precedente: SPEC-114 usó "Editar servicio" en el menú del detalle de servicio; SPEC-115 lo corrige a "Editar" y documenta el estándar.

### 4.6 Dropdowns: componente `Select` obligatorio

- **Todo** selector de opciones (color, mes, moneda, estado, institución, etc.) DEBE usar el componente reutilizable `frontend/src/components/Select.tsx`: dropdown custom con portal, tokens de tema (`bg-card`, `text-text`, `text-text-secondary`) y soporte de darkmode.
- **Prohibido** `<select>`/`<option>` nativo. El dropdown del sistema operativo rompe el patrón visual iOS de la app y no respeta el tema.
- Los selectores con muchas opciones (colores, monedas, instituciones, etc.) DEBEN ser `searchable`: el buscador se enfoca al abrir y filtra por etiqueta sin distinguir mayúsculas ni acentos.
- El build de `frontend/` ejecuta `frontend/scripts/check-ui-rules.cjs` **antes** de compilar y falla si detecta un `<select>` nativo. También se puede correr aislado con `npm run check:ui`.
- Precedentes: SPEC-004 CA-022 (dropdowns custom, no el del OS), SPEC-111 CA-018 (frecuencia con `Select`), SPEC-118 (selector de color de autos y guardrail de build).

---

## 5. Convenciones de Código Go

- Usar `gofmt` siempre.
- Nombres en inglés para código, español para specs y documentación.
- Manejo de errores explícito: nunca ignorar `error`.
- Configuración vía variables de entorno, con valores por defecto razonables.
- Logger simple: `log/slog` de stdlib.

---

## 6. Base de Datos

- SQLite con `journal_mode=WAL` para mejor concurrencia.
- Migraciones en `migrations/` con naming `NNNN_descripcion.up.sql` / `.down.sql`.
- No usar ORM. SQL crudo con `database/sql` y helpers propios livianos.

### 6.1 Validación con datos reales (obligatorio)

- **Toda prueba cuyo resultado dependa del contenido de la DB se corre contra una copia de la DB de producción**, no contra `data/app.db` local.
- Procedimiento: `sqlite3 /app/data/app.db ".backup '/tmp/<motivo>.db'"` vía SSH, descargar la copia y ejecutar sobre ella. Nunca escribir en producción durante la validación.
- La DB local solo sirve para tests unitarios con fixtures sintéticos; está desactualizada y puede cambiar el resultado (precedente SPEC-109: `Tigo` apareció como institución nueva por no existir en la DB local).

---

## 7. Consideraciones de iHost

- Imagen Docker base: `gcr.io/distroless/static` o `alpine:latest` si se necesita shell.
- Puerto por defecto: `8088` (configurable vía `PORT`).
- SQLite debe persistir en volumen montado (no dentro del contenedor).
- Health check ligero: `GET /health`.
- Sin dependencias de red externa obligatorias.

---

## 8. Datos privados y secretos (obligatorio)

### 8.1 Prohibido en el repositorio

Nunca se commitean:

- **Datos personales (PII)**: nombres y apellidos, teléfonos, emails personales, direcciones, cédulas/DNI, RUC personal, números de cuenta/medidor/contrato, números de factura reales, facturas o PDFs reales, chat IDs de Telegram, IPs/MACs domésticas.
- **Datos financieros**: montos, deudas, salarios, movimientos o cualquier extracto real.
- **Secretos**: contraseñas, hashes de contraseña, tokens (Telegram/APIs), API keys, cookies, `SESSION_SECRET`, credenciales SSH, webhook UUIDs/keys y claves privadas.
- **Volcados**: DBs (`*.db`, `-wal`, `-shm`), backups, exports JSON/CSV y capturas con datos reales.

Aplica a código, tests, fixtures, specs, docs, README, issues, PRs, mensajes de commit, logs y capturas.

### 8.2 Datos sintéticos obligatorios

En tests, ejemplos y specs usar siempre datos ficticios:

| Tipo | Ejemplo permitido |
|---|---|
| Identificador de cuenta | `demo-account-001` |
| Email | `demo@example.com` |
| Teléfono | `+1 555 0100` |
| Institución/proveedor | `Proveedor Demo` |
| Factura | `F000000000000001` |
| API key/token | `demo-api-key`, `demo-token` |
| UUID | `00000000-0000-4000-8000-000000000000` |

Está prohibido usar datos reales "porque son de prueba". Si hace falta reproducir un caso real, se sanitiza primero (nombres, números, direcciones, montos) y se documenta que es sintético.

### 8.3 Copias de producción

- Las copias de la DB de producción viven **solo en `/tmp`** (ver §6.1), nunca en el repo.
- Se borran al terminar la validación.
- No se pegan dumps, filas ni valores reales en specs, issues, PRs ni mensajes de chat/commit. Los resultados se describen con conteos y nombres de columnas, no con datos.

### 8.4 Antes de commitear

1. Revisar `git diff --cached`.
2. Escanear el diff buscando PII/secretos (teléfonos, emails no `example.com`, claves hex largas, `BEGIN ... PRIVATE KEY`, tokens de Telegram, `password`).
3. Verificar que no se agreguen `*.db`, `*.db-wal`, `*.db-shm`, `backups/`, `*.pdf` ni exports.
4. Si hay duda, no commitear: sanitizar primero.

### 8.5 Incidente

Si se commiteó un dato privado o un secreto:

1. Rotar el secreto de inmediato (API key, token, contraseña).
2. Eliminarlo del árbol **y de la historia** (no alcanza con un commit posterior).
3. Documentar el incidente y avisar al usuario.
