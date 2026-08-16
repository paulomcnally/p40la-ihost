---
title: "Integración de Specs con GitHub Issues para Gestión Asíncrona y Multi-Agente"
id: "SPEC-010"
status: "released"
author: "paulomcnally"
created: "2026-08-15"
updated: "2026-08-15"
github_issue: 11
---

# Integración de Specs con GitHub Issues para Gestión Asíncrona y Multi-Agente

**ID**: SPEC-010  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-08-15  
**Actualizado**: 2026-08-15 (implementación completa: labels, issues, SKILL.md, template)

---

## 1. Resumen Ejecutivo

El sistema actual de specs funciona con archivos Markdown locales y un tracker en `docs/specs/README.md`. Si bien es funcional para un flujo secuencial, tiene limitaciones cuando se quiere: (1) crear un backlog de specs para procesar asincrónicamente, (2) despachar múltiples agentes en paralelo, (3) tener un dashboard visual accesible desde cualquier lugar, y (4) forzar al agente a respetar el flujo de specs incluso cuando el usuario pide cambios iterativos sin mencionar la spec.

Esta spec propone integrar el sistema de specs con GitHub Issues, usando labels para mapear los estados del flujo de specs. Los archivos locales siguen siendo la fuente de verdad técnica; los issues actúan como espejo sincronizado con capacidad de gestión y visualización.

**No tiene impacto en iHost**: toda la integración se ejecuta en desarrollo vía `gh` CLI.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Al crear una spec con `/spec create`, se debe crear automáticamente un GitHub Issue con título, cuerpo resumido y label `spec/draft`. El número del issue se guarda en el frontmatter de la spec (`github_issue: <número>`).

2. **REQ-002**: Al cambiar el estado de una spec con `/spec status`, se debe actualizar el label del issue de GitHub correspondiente según la tabla de mapeo de estados.

3. **REQ-003**: Implementar comando `/spec list-issues [label]` que liste los issues de specs filtrados por estado/label, mostrando ID, título, estado y URL.

4. **REQ-004**: Implementar comando `/spec sync` que sincronice el estado de los issues de GitHub con los archivos locales. Si un issue fue cerrado manualmente, la spec pasa a `released`. Si un issue cambió de label, la spec actualiza su estado.

5. **REQ-005**: Al cerrar una spec (`/spec status SPEC-XXX released`), se debe cerrar el issue de GitHub correspondiente.

6. **REQ-006**: El agente DEBE verificar que existe una spec antes de escribir código significativo. Si el usuario pide cambios durante iteración que no están en la spec, el agente debe actualizar la spec (y el issue) antes de implementar, o al menos documentar los cambios como comentario en el issue.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

7. **REQ-007**: Implementar comando `/spec process <ID>` que tome una spec en `pending_execution`, la pase a `in_progress`, actualice el label del issue y genere un plan de tareas (todo) para el agente.

8. **REQ-008**: El cuerpo del issue de GitHub debe incluir un resumen estructurado de la spec: ID, estado, requerimientos P0, criterios de aceptación, y link al archivo completo en el repo.

9. **REQ-009**: Implementar comando `/spec pending` que muestre rápidamente cuántas specs están en cada estado no-terminal y sus URLs de issues.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

10. **REQ-010**: Al hacer cambios iterativos solicitados por el usuario que no están en la spec, agregar automáticamente un comentario al issue documentando el cambio solicitado y su implementación.

11. **REQ-011**: Soporte para asignar el issue a un usuario específico cuando se despacha a un agente.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Cada operación de sync no debe superar 5 segundos (depende de la API de GitHub).
- **Confiabilidad**: Si la API de GitHub falla, la operación local debe completarse igual y registrar un warning. Los archivos locales son la fuente de verdad.
- **iHost**: Cero impacto. Toda la integración usa `gh` CLI en desarrollo, no en runtime del iHost.
- **Rate limiting**: `gh` CLI maneja automáticamente el rate limit (5000 req/hora autenticado).

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **GitHub CLI (`gh`)**: Ya instalado y autenticado en el entorno (v2.97.0). Soporta todas las operaciones necesarias: `gh issue create`, `gh issue edit`, `gh issue list`, `gh issue close`, `gh issue reopen`, `gh issue comment`.
- **GitHub Labels API**: Permite crear, editar, listar labels con colores hexadecimales.
- **GitHub Issues como tracker**: Patrón común en proyectos open-source. Los issues son persistentes, tienen UI web, soportan labels, milestones, assignees, y comentarios.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| GitHub Issues + labels | Ya disponible, UI web, `gh` CLI, persistente, multi-agente | Depende de conexión a internet | ✅ Seleccionada |
| Solo archivos locales | Sin dependencias externas | Sin dashboard visual, difícil paralelizar | ❌ Rechazada |
| GitHub Projects | Más visual, tableros Kanban | Más complejo, overkill para este caso | ❌ Rechazada (por ahora) |
| Base de datos externa | Más flexible | Complejidad innecesaria, más dependencias | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-010-001**: Los archivos locales son la fuente de verdad; GitHub Issues son espejo sincronizado.
- **Contexto**: Si perdemos conexión a GitHub o la API falla, el sistema local debe seguir funcionando.
- **Decisión**: Siempre se actualiza el archivo local primero. La operación de GitHub es best-effort con warning si falla.
- **Consecuencias**: Puede haber desincronización temporal, resuelta por `/spec sync`.

**ADR-010-002**: Un issue cerrado = spec released o cancelled.
- **Contexto**: GitHub no tiene un estado "cerrado" semántico. Necesitamos mapear el cierre a un estado de spec.
- **Decisión**: Si el issue se cierra, `/spec sync` determina el estado basándose en el label actual: `spec/released` → `released`, `spec/cancelled` → `cancelled`, sin label conocido → log warning sin cambiar estado.
- **Consecuencias**: El cierre manual desde la UI de GitHub se refleja correctamente en local.

**ADR-010-003**: Labels con namespace `spec/` para evitar colisiones.
- **Contexto**: El repo puede tener otros labels para bugs, features, etc.
- **Decisión**: Todos los labels de spec usan el prefijo `spec/`: `spec/draft`, `spec/pending-execution`, `spec/in-progress`, `spec/pending-release`, `spec/released`, `spec/cancelled`.
- **Consecuencias**: Labels organizados y autocontenidos.

## 4. Diseño Técnico

### 4.1 Mapeo de estados

| Estado spec | Label GitHub | Color hex | Acción issue |
|-------------|-------------|-----------|--------------|
| `draft` | `spec/draft` | `#F4D03F` | Abierto |
| `pending_execution` | `spec/pending-execution` | `#3498DB` | Abierto |
| `in_progress` | `spec/in-progress` | `#9B59B6` | Abierto |
| `pending_release` | `spec/pending-release` | `#E67E22` | Abierto |
| `released` | `spec/released` | `#2ECC71` | **Cerrado** |
| `cancelled` | `spec/cancelled` | `#95A5A6` | **Cerrado** |

### 4.2 Labels a crear en GitHub

```bash
gh label create "spec/draft" --color "F4D03F" --description "Spec en redacción"
gh label create "spec/pending-execution" --color "3498DB" --description "Lista para desarrollo"
gh label create "spec/in-progress" --color "9B59B6" --description "En desarrollo activo"
gh label create "spec/pending-release" --color "E67E22" --description "Desarrollo completo, pendiente release"
gh label create "spec/released" --color "2ECC71" --description "En producción/iHost"
gh label create "spec/cancelled" --color "95A5A6" --description "Cancelada o descartada"
```

### 4.3 Flujo de operaciones

#### Crear spec (`/spec create "Título"`)

```
1. Generar ID (SPEC-XXX)
2. Crear archivo docs/specs/SPEC-XXX-titulo.md
3. Actualizar docs/specs/README.md
4. gh issue create \
     --title "SPEC-XXX: Título" \
     --label "spec/draft" \
     --body "<cuerpo estructurado con resumen de la spec>"
5. Extraer número del issue creado (del stdout de gh)
6. Actualizar frontmatter: github_issue: <número>
7. Confirmar al usuario
```

#### Cambiar estado (`/spec status SPEC-XXX nuevo_estado`)

```
1. Validar transición de estados
2. Leer spec, obtener github_issue del frontmatter
3. Si github_issue existe:
   a. Determinar label nuevo según mapeo
   b. gh issue edit <número> --remove-label "spec/<estado_anterior>" --add-label "spec/<nuevo_estado>"
   c. Si nuevo_estado es released o cancelled:
      gh issue close <número>
4. Actualizar archivo local (frontmatter + historial)
5. Actualizar README.md
6. Confirmar
```

#### Listar issues (`/spec list-issues [label]`)

```
1. gh issue list --label "spec/<label>" --json number,title,labels,state,url --limit 50
2. Parsear y formatear output mostrando: ID spec, título, estado, URL
3. Si no hay label, mostrar todos los issues con label que empiece con "spec/"
```

#### Sync (`/spec sync`)

```
1. gh issue list --label "spec/" --json number,title,labels,state
2. Para cada issue:
   a. Buscar spec local que tenga github_issue == issue.number
   b. Si no existe spec local → log warning (issue huérfano)
   c. Comparar estado del issue (labels + closed/open) con estado de la spec
   d. Si hay diferencia:
      - Determinar nuevo estado según labels
      - Si issue está cerrado y tiene label spec/released → spec.released
      - Si issue está cerrado y tiene label spec/cancelled → spec.cancelled
      - Actualizar archivo local + README.md
      - Log: "SPEC-XXX sincronizado: <estado_anterior> → <nuevo_estado>"
3. Reporte final: X specs sincronizadas, Y sin cambios, Z warnings
```

#### Process (`/spec process SPEC-XXX`)

```
1. Validar que spec esté en pending_execution
2. Cambiar estado a in_progress (actualizar archivo, label, README)
3. Leer criterios de aceptación de la spec
4. Generar lista de tareas (todo) para el agente
5. Confirmar y mostrar plan
```

### 4.4 Cuerpo del issue de GitHub

```markdown
## SPEC-XXX: Título

**Estado**: draft | pending_execution | in_progress | pending_release | released
**Archivo**: [docs/specs/SPEC-XXX-titulo.md](link al archivo en el repo)

---

### Resumen
[2-3 párrafos del resumen ejecutivo]

### Requerimientos P0
1. REQ-001: ...
2. REQ-002: ...

### Criterios de Aceptación
- [ ] CA-001: ...
- [ ] CA-002: ...

### Plan de Implementación
| Fase | Descripción | Estimación |
|------|-------------|------------|
| 1 | ... | ... |

---
*Este issue es gestionado automáticamente por el sistema de specs. No cambiar labels manualmente.*
```

### 4.5 Actualización del SKILL.md

La skill `spec-manager` debe actualizarse para incluir:

1. **Nuevo comando**: `/spec list-issues [label]`
2. **Nuevo comando**: `/spec sync`
3. **Nuevo comando**: `/spec process <ID>`
4. **Nuevo comando**: `/spec pending`
5. **Regla obligatoria**: Antes de cualquier operación de cambio de estado, intentar actualizar el issue de GitHub. Si falla, continuar con warning.
6. **Regla obligatoria**: Al crear spec, crear issue de GitHub.
7. **Regla de flujo forzado**: El agente DEBE verificar existencia de spec antes de escribir código. Si el usuario pide cambios iterativos no contemplados, actualizar spec + issue (o comentar en issue) antes de implementar.

### 4.6 Archivos a modificar

| Archivo | Cambio |
|---------|--------|
| `.opencode/skills/spec-manager/SKILL.md` | Agregar nuevos comandos, reglas de GitHub, flujo forzado |
| `docs/specs/templates/spec-template.md` | Agregar campo `github_issue` al frontmatter |
| `docs/specs/README.md` | Actualizar tracker (ya se hace, sin cambio adicional) |

### 4.7 Dependencias

- **Internas**: Ninguna nueva
- **Externas**: `gh` CLI (ya instalado v2.97.0), acceso a GitHub API

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado que ejecuto `/spec create "Test"`, cuando se completa, entonces existe un archivo local Y un issue de GitHub con label `spec/draft` Y el frontmatter tiene `github_issue` con el número correcto.
- [ ] CA-002: Dado una spec SPEC-XXX con issue asociado, cuando ejecuto `/spec status SPEC-XXX in_progress`, entonces el issue cambia su label de `spec/draft` a `spec/in-progress`.
- [ ] CA-003: Dado una spec SPEC-XXX en `pending_release`, cuando ejecuto `/spec status SPEC-XXX released`, entonces el issue se cierra Y tiene label `spec/released`.
- [ ] CA-004: Dado que cierro manualmente un issue de spec desde la UI de GitHub, cuando ejecuto `/spec sync`, entonces la spec local pasa a `released` o `cancelled` según el label.
- [ ] CA-005: Dado que ejecuto `/spec list-issues pending-execution`, entonces se muestran solo los issues con label `spec/pending-execution` con ID, título y URL.
- [ ] CA-006: Dado que ejecuto `/spec sync`, entonces todas las specs con issue asociado tienen su estado sincronizado con GitHub.
- [ ] CA-007: Dado una spec en `pending_execution`, cuando ejecuto `/spec process SPEC-XXX`, entonces la spec pasa a `in_progress`, el label se actualiza, y se muestra un plan de tareas.

### 5.2 No funcionales

- [ ] CA-NF-001: Si GitHub API no responde, la operación local se completa igual con un warning visible.
- [ ] CA-NF-002: Cada operación de sync completa en menos de 10 segundos con hasta 50 issues.
- [ ] CA-NF-003: Los 6 labels se crean correctamente con los colores especificados.

### 5.3 Testing

- **Pruebas manuales**: Crear spec, verificar issue, cambiar estados, verificar labels, cerrar issue, sync.
- **Pruebas de resiliencia**: Desconectar red, verificar que operaciones locales funcionan con warning.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Crear labels en GitHub | 2 min | Ninguna |
| 2 | Actualizar template de spec con campo `github_issue` | 5 min | Ninguna |
| 3 | Actualizar SKILL.md con nuevos comandos y reglas | 15 min | Fase 2 |
| 4 | Implementar flujo de creación de issue en `/spec create` | 10 min | Fase 3 |
| 5 | Implementar actualización de labels en `/spec status` | 10 min | Fase 4 |
| 6 | Implementar `/spec list-issues` | 10 min | Fase 3 |
| 7 | Implementar `/spec sync` | 15 min | Fase 3 |
| 8 | Implementar `/spec process` | 10 min | Fase 3 |
| 9 | Implementar `/spec pending` | 5 min | Fase 3 |
| 10 | Pruebas manuales completas | 15 min | Fases 1-9 |

### 6.2 Milestones

1. **MVP (Fases 1-5)**: Crear spec con issue, cambiar estados con sync de labels, cerrar issue al release.
2. **V1.0 (Fases 6-10)**: Todos los comandos adicionales, sync bidireccional, pruebas completas.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| GitHub API rate limit | Baja | Medio | `gh` maneja rate limit automáticamente. 5000 req/hora es más que suficiente. |
| Desincronización local ↔ GitHub | Media | Bajo | `/spec sync` resuelve. Archivos locales son fuente de verdad. |
| Issue cerrado accidentalmente | Baja | Medio | `/spec sync` detecta y actualiza. Se puede reabrir con `/spec status`. |
| `gh` CLI no disponible | Baja | Alto | Operaciones locales funcionan igual. Warning visible. Specs sin issue se pueden vincular después. |
| Agente ignora flujo de specs | Media | Alto | Regla explícita en SKILL.md + validación antes de escribir código. |

## 8. Notas y Referencias

- Documentación de GitHub CLI: https://cli.github.com/manual/
- GitHub Issues API: https://docs.github.com/rest/issues/issues
- Esta spec reemplaza el sistema de solo archivos locales como fuente única, añadiendo GitHub como espejo sincronizado.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-15 | paulomcnally | Creación inicial de la especificación |
| 2026-08-15 | paulomcnally | Labels creados, issues SPEC-001 a SPEC-009 migrados, SKILL.md actualizado |
| 2026-08-15 | paulomcnally | Spec released |
