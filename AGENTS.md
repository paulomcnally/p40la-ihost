# AGENTS.md - Reglas del Repositorio

> **Última actualización**: 2026-08-13  
> **Proyecto**: p40la-ihost  
> **Sistema de specs**: v1.2

---

## 🎯 Propósito

Este repositorio utiliza un sistema formal de especificaciones técnicas (specs) para gestionar el desarrollo de funcionalidades. Todo cambio significativo DEBE pasar por el flujo de specs.

El proyecto corre en un **iHost con recursos muy limitados**, por lo que todas las decisiones técnicas deben priorizar simplicidad, bajo consumo de memoria y mínimas dependencias.

---

## ⚠️ Regla de Oro

**Antes de escribir código: la spec primero. Siempre.**  
Si la spec no existe, no se escribe código. Si la spec no contempla el cambio, la spec se actualiza primero. No hay excepciones.

---

## 📋 Reglas Fundamentales

### 1. Cero código sin spec

- **No se escribe una sola línea de código sin que exista una spec que lo defina.**
- El agente **JAMÁS** debe hacer cambios en el código que no estén contemplados en una spec existente o en creación.
- Si durante el desarrollo surge un cambio no contemplado (nuevo archivo, nuevo índice, cambio de modelo, etc.), la spec DEBE actualizarse **antes** de escribir el código.

### 2. ¿Qué necesita spec?

**TODO necesita spec, excepto**:
- Hotfixes críticos (servicio caído, seguridad)
- Cambios cosméticos (espacios, comentarios, formato)
- Actualizaciones de dependencias sin cambio de comportamiento

Cualquier cambio que toque lógica de negocio, base de datos, APIs, eventos, configuración de infraestructura o UI requiere spec **obligatoriamente**.

### 3. Flujo estricto

```
1. Identificar necesidad → 2. Crear spec (draft) → 3. Completar spec (pending_execution) → 4. Desarrollar
```

- **Paso 2 y 3**: Se crea la spec, se documenta todo (requerimientos, diseño, criterios de aceptación).
- **Paso 4**: Solo entonces se escribe código. Nunca a la inversa.
- Si se detecta algo no previsto durante el desarrollo → volver al paso 2, actualizar la spec, luego continuar.
- Si durante la evaluación manual local el usuario solicita cambios (incluso pequeños), el agente debe:
  1. Actualizar la spec para reflejar los cambios solicitados.
  2. Crear/actualizar la lista de tareas (`todo`) con cada cambio pendiente.
  3. Implementar los cambios y volver a dejar la app corriendo en local para reevaluación.
  4. Solo cuando el usuario confirme, se considera el spec cerrado.

### 4. Documentación mínima

Cada spec DEBE incluir:
- Requerimientos claros y priorizados (P0/P1/P2)
- Investigación técnica con fuentes cuando aplica
- Decisiones arquitectónicas justificadas
- Criterios de aceptación verificables
- Plan de implementación con estimaciones
- Consideraciones específicas de iHost (recursos limitados, SQLite, etc.)

### 5. IDs únicos y secuenciales

Los IDs de spec (`SPEC-XXX`) son inmutables y secuenciales. Nunca se reutiliza un número.

---

## 🔄 Flujo de Trabajo

### Fase 1: Requerimiento

1. Se identifica una necesidad
2. Se crea una spec usando la skill `spec-manager` o el template
3. Estado inicial: `draft`

### Fase 2: Definición

1. Investigar soluciones técnicas
2. Documentar decisiones y arquitectura
3. Definir criterios de aceptación
4. Validar con stakeholders
5. Mover a estado: `pending_execution`

### Fase 3: Desarrollo

1. Crear rama/feature desde la spec
2. Desarrollar según la especificación
3. Estado: `in_progress`

### Fase 4: Validación

1. Pasar criterios de aceptación
2. Code review
3. QA/Testing (tests automáticos + pruebas manuales locales)
4. El agente debe correr la aplicación en local y dejarla disponible para evaluación del usuario
5. El usuario confirma que los cambios están correctos tras probar manualmente
6. Estado: `pending_release`

### Fase 5: Release

1. Merge/deploy según lo acordado con el usuario
2. Documentar commit/versión en la spec
3. Estado: `released`

---

## 🚫 Restricciones de Despliegue (Agentes)

### Reglas absolutas

- **JAMÁS** ejecutar `git push` a ramas protegidas sin confirmación explícita del usuario.
- **JAMÁS** mergear a `main` directamente desde una rama de feature.
- **JAMÁS** ejecutar `--force` o `--force-with-lease` en `main` o ramas de release.

### Pruebas locales obligatorias

- **Todo cambio significativo debe probarse en local** (tests, build, ejecución básica) antes de intentar commits o push.
- **No se realiza commit** de código que no compile, que falle los tests o que no haya sido validado en el entorno de desarrollo local.
- Antes de solicitar confirmación para merge/push, el agente debe verificar que la aplicación levanta correctamente con la base de datos SQLite y que los criterios de aceptación del spec pasan en local.

### Flujo permitido

1. Desarrollar en rama `feature/SPEC-XXX-*` (local).
2. Ejecutar pruebas locales (tests, build y validación manual mínima).
3. Correr la aplicación en local para que el usuario evalúe los cambios manualmente.
4. El usuario confirma que todo está correcto; solo entonces se considera el spec listo para cerrar/publicar.
5. Solicitar confirmación al usuario para merge/push.
6. Solo el usuario puede aprobar deploys a producción/iHost.

---

## 🗂️ Estructura de Carpetas

```
/
├── .opencode/
│   └── skills/
│       └── spec-manager/
│           └── SKILL.md         # Skill de gestión de specs
├── docs/
│   ├── specs/
│   │   ├── README.md            # Tracker y contador
│   │   ├── templates/
│   │   │   └── spec-template.md # Template base
│   │   └── SPEC-XXX-*.md        # Especificaciones
│   ├── project-rules.md         # Reglas de stack y arquitectura
│   └── infrastructure.md        # Infraestructura y deploy
├── AGENTS.md                    # Este archivo
├── opencode.json               # Configuración de opencode
├── src/                        # Código fuente backend
├── public/                     # Frontend estático básico
└── package.json
```

---

## 🤖 Uso de la Skill `spec-manager`

La skill está registrada en `.opencode/skills/spec-manager/SKILL.md`.

### Comandos disponibles

| Comando | Descripción |
|---------|-------------|
| `/spec create "Título"` | Crear nueva spec con investigación automática |
| `/spec list [estado]` | Listar specs, opcionalmente filtradas |
| `/spec status <id> <estado>` | Cambiar estado de una spec |
| `/spec show <id>` | Mostrar resumen de una spec |

### Activación automática

La skill se activa automáticamente cuando el usuario menciona:
- "crear spec", "nueva especificación", "nuevo requerimiento"
- "estado de spec", "cambiar estado", "pending release"
- "contador de specs", "listar specs", "tracking"

---

## ✅ Checklist de Calidad

Antes de marcar una spec como `pending_release`:

- [ ] Todos los requerimientos P0 están implementados
- [ ] Los criterios de aceptación pasan
- [ ] Hay tests unitarios/integración si aplica
- [ ] Se actualizó la documentación técnica si aplica
- [ ] Se hizo code review
- [ ] No hay deuda técnica crítica sin documentar
- [ ] Se verificó que funciona en entornos con recursos limitados (iHost)

---

## 📞 Escalación

Si una spec se bloquea más de 3 días en cualquier estado, se debe:
1. Documentar el bloqueo en la spec (sección de riesgos)
2. Evaluar si se cancela o se redefine
3. Notificar al equipo

---

*Este documento es la fuente de verdad para el sistema de especificaciones. Cualquier cambio al flujo debe actualizar este archivo y la skill correspondiente.*
