---
title: "Título de la especificación"
id: "SPEC-XXX"
status: "draft"
author: ""
created: "YYYY-MM-DD"
updated: "YYYY-MM-DD"
github_issue: null
---

# {{title}}

**ID**: {{id}}  
**Estado**: {{status}}  
**Autor**: {{author}}  
**Creado**: {{created}}  
**Actualizado**: {{updated}}

---

## 1. Resumen Ejecutivo

Describir en 2-3 párrafos:
- Qué problema se resuelve
- Por qué es importante ahora
- Qué resultado se espera

Consideraciones específicas de iHost: impacto en memoria, almacenamiento, SQLite, etc.

> **Las reglas no se copian en la spec.** Se citan por ID desde `docs/rules/` y se declaran en la sección 2.

---

## 2. Cumplimiento de reglas

Reglas del proyecto aplicables a esta spec (índice: `docs/rules/00-index.md`).

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-*` | … o `N/A` justificado | `CA-ARCH-*` |
| `UI-SELECT` | … o `N/A` justificado | `CA-UI-SELECT` |
| `UI-TOKENS` / `UI-DARKMODE` | … o `N/A` justificado | `CA-UI-*` |
| `UI-EDIT-LABEL` | … o `N/A` justificado | `CA-UI-EDIT-LABEL` |
| `UI-I18N` | … o `N/A` justificado | `CA-UI-I18N` |
| `UX-BACK` | … o `N/A` justificado | `CA-UX-BACK` |
| `UX-MODAL` / `UX-MODAL-CONFIRM` | … o `N/A` justificado | `CA-UX-MODAL` |
| `UX-DETAIL` | … o `N/A` justificado | `CA-UX-DETAIL` |
| `UX-PREREQ` | … o `N/A` justificado | `CA-UX-PREREQ` |
| `DATA-PRODCOPY` | … o `N/A` justificado | `CA-DATA-PRODCOPY` |
| `SEC-SYNTHETIC` | Ejemplos con datos `demo-*` | `CA-SEC-SYNTHETIC` |

> Aplicar solo las reglas que correspondan al tipo de cambio. Si una regla aplica y no existe en `docs/rules/`: crearla antes de continuar.

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Descripción del requerimiento obligatorio
2. **REQ-002**: Descripción del requerimiento obligatorio

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-003**: Descripción del requerimiento importante

### 3.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-004**: Descripción del requerimiento deseable

### 3.4 Requerimientos No Funcionales

- **Rendimiento**: Tiempo de respuesta esperado, uso máximo de memoria
- **Seguridad**: Autenticación, autorización, datos sensibles
- **Almacenamiento**: Tamaño máximo en disco, rotación de logs
- **Disponibilidad**: Uptime esperado, health checks
- **iHost**: Consumo de RAM, CPU, dependencias minimizadas

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

Documentar qué se investigó:
- Fuentes consultadas (links, documentación)
- Tecnologías evaluadas
- Patrones considerados

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Opción A | ... | ... | ✅ Seleccionada |
| Opción B | ... | ... | ❌ Rechazada |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: [Título de la decisión]
- **Contexto**: Por qué se tomó esta decisión
- **Decisión**: Qué se decidió
- **Consecuencias**: Impactos positivos y negativos

---

## 5. Diseño Técnico

### 5.1 Diagrama de arquitectura

```
[Componente A] --(API REST)--> [Componente B]
     |                              |
     v                              v
[SQLite]                    [Frontend estático]
```

### 5.2 Componentes

#### 5.2.1 [Nombre del componente]
- **Responsabilidad**: Qué hace
- **Interfaz**: API expuesta, contratos
- **Dependencias**: Qué necesita
- **Ubicación**: Ruta en el repo

### 5.3 Modelo de datos

```
Entidad: [Nombre]
- campo_1: tipo (descripción)
- campo_2: tipo (descripción)
- Relaciones: [Entidad] (1:N)
```

### 5.4 APIs / Contratos

#### Endpoint: `METHOD /ruta`

**Request**:
```json
{
  "campo": "tipo"
}
```

**Response 200**:
```json
{
  "campo": "tipo"
}
```

**Response Error**:
```json
{
  "error": "código",
  "message": "descripción"
}
```

### 5.5 Dependencias

- **Internas**: Servicios/componentes del sistema que se modifican
- **Externas**: Librerías, APIs de terceros, servicios externos

---

## 6. Impacto en tools MCP

> Obligatorio (`DELIV-MCP-IMPACT`). Completar con `./scripts/check-mcp-impact.sh --files <archivos>` y la validación de un subagente (`DELIV-MCP-IMPACT-SUBAGENT`). Si no hay tools afectadas, escribir «ninguna» y justificarlo.

| Tool | ¿Afectada? | Motivo / ruta de dependencia | Contrato |
|---|---|---|---|
| `bills_list` | No | … | Sin cambios |
| `bills_generate` | No | … | Sin cambios |
| `bills_pay` | No | … | Sin cambios |

**Validación subagente**: `<explore|general>` — sesión `<ses_...>` — veredicto: …
**Manifest**: `docs/mcp/tools/impact.json`
**Contrato**: `docs/mcp/tools/contracts.json`

---

## 7. Criterios de Aceptación

> Usar IDs canónicos por dominio (`CA-<PREFIJO>-<NOMBRE>`) para que el cumplimiento sea auditable con grep.

### 7.1 Funcionales

- [ ] CA-001: Dado [contexto], cuando [acción], entonces [resultado esperado]
- [ ] CA-002: Dado [contexto], cuando [acción], entonces [resultado esperado]
- [ ] CA-UI-TOKENS: Los inputs usan tokens del tema y se verificó legibilidad en darkmode.
- [ ] CA-UI-SELECT: Los selectores usan el componente `Select` (buscable) y no hay `<select>` nativo.
- [ ] CA-UI-EDIT-LABEL: Las acciones de edición usan `app.edit` («Editar»).
- [ ] CA-UX-MODAL: El click fuera no cierra, el overlay bloquea el fondo, hay cierre visible en todos los estados, se respeta la escala de capas y la confirmación por palabra acepta Enter.
- [ ] CA-UX-BACK: El retorno va en el header vía `BACK_ROUTES`; no hay links "← Título" en el contenido.
- [ ] CA-UX-DETAIL: El detalle es de solo lectura; editar navega a `/edit/:id`.
- [ ] CA-UX-PREREQ: El prerrequisito se valida en backend y en frontend.
- [ ] CA-SEC-SYNTHETIC: Ejemplos y pruebas usan datos sintéticos.

### 7.2 No funcionales

- [ ] CA-NF-001: [Criterio de rendimiento/seguridad/etc.]

### 7.3 Testing

- **Unit tests**: Qué lógica debe cubrirse
- **Integration tests**: Qué flujos integrados probar
- **E2E tests**: Qué escenarios de usuario probar
- **Carga/Performance**: Qué métricas validar en iHost

---

## 8. Plan de Implementación

### 8.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | [Tarea] | X días | Ninguna |
| 2 | [Tarea] | Y días | Fase 1 |

### 8.2 Milestones

1. **MVP**: [Qué incluye el mínimo viable]
2. **V1.0**: [Versión completa]

---

## 9. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| [Descripción] | Alta/Media/Baja | Alto/Medio/Bajo | [Estrategia] |

---

## 10. Notas y Referencias

- Reglas aplicadas: `docs/rules/...`
- Links útiles
- Documentación relacionada
- Tickets/issues relacionados

---

## 11. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| YYYY-MM-DD | [Nombre] | Creación inicial de la especificación |
