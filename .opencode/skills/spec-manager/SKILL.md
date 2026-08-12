---
name: spec-manager
description: "Use ONLY when the user asks to create, manage, update, or track technical specifications (specs). Triggers on keywords: spec, especificacion, especificación, requirement, requerimiento, tracking, contador, estado de spec, pending release. Manages the full spec lifecycle from creation to release for the p40la-ihost project."
---

# Spec Manager

Sistema de gestión de especificaciones técnicas para el repositorio `p40la-ihost`.

## Ubicación

- **Specs activas**: `docs/specs/`
- **Template**: `docs/specs/templates/spec-template.md`
- **Tracking**: `docs/specs/README.md`

## Estados de una Spec

```
draft → pending_execution → in_progress → pending_release → released
```

| Estado | Descripción | Color |
|--------|-------------|-------|
| `draft` | En proceso de redacción/investigación | 🟡 |
| `pending_execution` | Lista para desarrollo, no iniciada | 🔵 |
| `in_progress` | Actualmente en desarrollo | 🟣 |
| `pending_release` | Desarrollo completo, lista para release | 🟠 |
| `released` | Subida a iHost o producción | 🟢 |

## Flujo de trabajo

### 1. Crear una nueva spec

Cuando el usuario pide crear una spec:

1. **Investigar** el requerimiento usando websearch/webfetch si es necesario
2. **Generar ID**: Buscar el último número en `docs/specs/README.md` y asignar `SPEC-XXX` (3 dígitos, zero-padded)
3. **Crear archivo**: `docs/specs/SPEC-XXX-<titulo-breve>.md`
4. **Documentar completamente** usando el template como guía
5. **Actualizar tracker**: Agregar al `docs/specs/README.md` con estado inicial `draft`
6. **Confirmar**: Informar al usuario el ID asignado

### 2. Contador de specs

El contador está en `docs/specs/README.md` en la sección "## Resumen".

- Total de specs: conteo de todas las specs en el directorio
- Por estado: conteo agrupado por estado
- Último ID usado: último número secuencial

### 3. Actualizar estado de una spec

Cuando el usuario menciona que una spec cambió de estado:

1. **Actualizar README.md**: Cambiar el estado en la tabla de tracking
2. **Actualizar archivo de spec**: Cambiar el `status:` en el frontmatter del archivo `.md`
3. **Actualizar sección de historial**: Agregar entrada con fecha y descripción del cambio
4. **Verificar flujo**: Asegurar que el estado anterior y nuevo son válidos según el diagrama de estados

### 4. Reglas de documentación

Cada archivo de spec DEBE incluir:

- **Frontmatter** con: title, id, status, author, created, updated
- **Resumen ejecutivo**: Qué problema resuelve (2-3 párrafos)
- **Requerimientos**: Lista numerada y priorizada (P0, P1, P2)
- **Investigación**: Referencias, decisiones técnicas evaluadas, porqué de cada decisión
- **Diseño técnico**: Arquitectura, componentes, APIs, dependencias
- **Criterios de aceptación**: Lista verificable de condiciones que definen "terminado"
- **Plan de implementación**: Fases, estimación, dependencias entre tareas
- **Riesgos y mitigaciones**: Qué puede fallar y cómo manejarlo
- **Historial de cambios**: Fecha, autor, descripción de cada modificación

### 5. Release

Cuando una spec pasa a `released`:

1. Verificar que exista commit/versión que la suba a iHost
2. Documentar en la spec: commit/versión, fecha de deploy
3. El estado `released` significa "en iHost o producción"

## Comandos de la skill

### `/spec create "<título>"`
Crear nueva spec con título dado. Investigación automática.

### `/spec list [estado]`
Listar specs. Filtrar por estado si se proporciona.

### `/spec status <id> <nuevo_estado>`
Cambiar estado de una spec. Validar transición permitida.

### `/spec show <id>`
Mostrar resumen de una spec específica.

## Validaciones

- Un ID no se puede reutilizar
- `draft` puede ir a `pending_execution` o `cancelled`
- `pending_execution` puede ir a `in_progress` o `cancelled`
- `in_progress` puede ir a `pending_release` o volver a `pending_execution`
- `pending_release` puede ir a `released` o volver a `in_progress`
- `released` es estado final (o puede ir a `in_progress` para hotfixes)
- `cancelled` es estado terminal

## Formato de IDs

- Siempre `SPEC-XXX` donde XXX es 001-999
- Los números son secuenciales y no se reutilizan
- Extraer número del nombre de archivo: `SPEC-(\d{3})`
