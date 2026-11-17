---
title: "{{TITLE}}"
id: "{{ID}}"
status: "draft"
author: "{{AUTHOR}}"
created: "{{DATE}}"
updated: "{{DATE}}"
github_issue: null
---

# {{TITLE}}

**ID**: {{ID}}
**Estado**: {{STATUS}}
**Autor**: {{AUTHOR}}
**Creado**: {{DATE}}
**Actualizado**: {{DATE}}

---

## 1. Resumen Ejecutivo

Qué problema se resuelve, por qué es importante ahora y qué resultado se espera (2-3 párrafos).

Consideraciones del runtime/entorno: memoria, almacenamiento, dependencias, límites conocidos.

---

## 2. Cumplimiento de reglas

Reglas del proyecto aplicables a esta spec (ver `docs/rules/00-index.md`).

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-*` | … | `CA-ARCH-*` |
| `UI-*` | … | `CA-UI-*` |
| `UX-*` | … | `CA-UX-*` |
| `DATA-*` | … | `CA-DATA-*` |
| `SEC-*` | … | `CA-SEC-*` |
| `TEST-*` | … | `CA-TEST-*` |
| `DELIV-*` | … | `CA-DELIV-*` |

- `N/A` solo con justificación explícita.
- Si una regla aplica y no existe en `docs/rules/`: crear la regla antes de continuar.

---

## 3. Requerimientos

### 3.1 Funcionales (P0 - Obligatorios)
1. **REQ-001**: …

### 3.2 Funcionales (P1 - Importantes)
1. **REQ-002**: …

### 3.3 Funcionales (P2 - Deseables)
1. **REQ-003**: …

### 3.4 No funcionales
- **Rendimiento**: …
- **Seguridad**: …
- **Almacenamiento**: …
- **Disponibilidad**: …

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado
Fuentes, tecnologías evaluadas, patrones considerados.

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| A | … | … | ✅ |
| B | … | … | ❌ |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: …
- **Contexto**:
- **Decisión**:
- **Consecuencias**:

---

## 5. Diseño Técnico

### 5.1 Diagrama
```
{{DIAGRAMA}}
```

### 5.2 Componentes
#### 5.2.1 {{COMPONENTE}}
- **Responsabilidad**:
- **Interfaz**:
- **Dependencias**:
- **Ubicación**:

### 5.3 Modelo de datos
```
{{MODELO}}
```

### 5.4 APIs / Contratos
#### `METHOD /ruta`
**Request** / **Response 200** / **Response Error**

### 5.5 Dependencias
- **Internas**:
- **Externas**:

---

## 6. Criterios de Aceptación

### 6.1 Funcionales
- [ ] CA-001: Dado …, cuando …, entonces …
- [ ] CA-UI-001: …
- [ ] CA-UX-001: …
- [ ] CA-DATA-001: …
- [ ] CA-SEC-001: …

> Usar IDs canónicos por dominio (`CA-<PREFIJO>-<NOMBRE>`) para que el cumplimiento sea auditable.

### 6.2 No funcionales
- [ ] CA-NF-001: …

### 6.3 Testing
- **Unit**:
- **Integration**:
- **E2E**:
- **Carga/Performance**:

---

## 7. Plan de Implementación

| Fase | Descripción | Estimación | Dependencias |
|---|---|---|---|
| 1 | … | … | — |

**Milestones**: MVP / V1.0

---

## 8. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| … | Alta/Media/Baja | Alto/Medio/Bajo | … |

---

## 9. Notas y Referencias

- Reglas aplicadas: `docs/rules/…`
- Specs relacionadas:
- Issues:

---

## 10. Historial de Cambios

| Fecha | Autor | Descripción |
|---|---|---|
| {{DATE}} | {{AUTHOR}} | Creación inicial |
