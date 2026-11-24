---
title: "MCP: búsqueda de entidades por nombre"
id: "SPEC-150"
status: "in_progress"
author: "opencode"
created: "2026-11-24"
updated: "2026-11-24"
github_issue: 153
---

# MCP: búsqueda de entidades por nombre

**ID**: SPEC-150  
**Estado**: in_progress  
**Autor**: opencode  
**Creado**: 2026-11-24  
**Actualizado**: 2026-11-24

---

## 1. Resumen Ejecutivo

Las tools MCP actuales obligan al agente a conocer IDs: `bills_list`, `bills_generate` y `bills_pay` reciben `service_id`, y las tools de alta de SPEC-146 resuelven nombres solo para crear. Un agente conectado desde claude.ai no tiene forma de descubrir el `id` de un servicio, hogar, banco o institución que ya existe, así que termina pidiéndole el ID al usuario o adivinando. Eso rompe el flujo para el que se construyó el MCP.

Esta spec agrega **una tool de solo lectura** (`entities_search`) que busca por nombre en servicios, hogares, bancos e instituciones y devuelve hasta N candidatos con su `id` y campos mínimos para desambiguar. La búsqueda ignora mayúsculas y acentos, ordena exactos → prefijo → contiene, y nunca elige por el agente: devuelve la lista para que el siguiente paso use el `id` correcto.

El impacto en el iHost es mínimo: no hay migraciones, no se agregan dependencias y cada búsqueda hace una sola llamada HTTP al backend existente. El proceso MCP sigue corriendo como contenedor liviano y la tool funciona con las escrituras deshabilitadas.

---

## 2. Cumplimiento de reglas

Reglas aplicables según `docs/rules/00-index.md`. La spec no toca UI ni rutas, así que los dominios `20-ui` y `30-ux` quedan fuera.

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Solo Go + `modelcontextprotocol/go-sdk` ya presente; sin dependencias nuevas | `CA-ARCH-STACK` |
| `ARCH-FOLDERS` | La tool vive en `internal/mcp/tools/entities/search.go`, junto a las tools de SPEC-146 | `CA-ARCH-FOLDERS` |
| `ARCH-LAYERS` | La tool solo normaliza, filtra y llama al cliente HTTP; sin SQL ni lógica de dominio | `CA-ARCH-LAYERS` |
| `ARCH-GO` | `gofmt`, errores explícitos, nombres en inglés, sin `panic` | `CA-ARCH-GO` |
| `ARCH-IHOST` | Una llamada HTTP por búsqueda, sin memoria adicional relevante, sin migraciones | `CA-ARCH-IHOST` |
| `DATA-PRODCOPY` | La validación manual de resultados se corre contra una copia de producción en `/tmp`; nunca se escribe en producción | `CA-DATA-PRODCOPY` |
| `SEC-NO-PII` | La tool no loguea el término de búsqueda ni devuelve datos fuera de los DTOs mínimos | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Ejemplos con `Servicio Demo`, `Hogar Demo`, `Banco Demo`, `Proveedor Demo` | `CA-SEC-SYNTHETIC` |
| `SEC-LOGS` | El logger HTTP sigue registrando solo método, path, status y duración | `CA-SEC-LOGS` |
| `TEST-LOCAL` | `go build ./...` y `go test ./...` verdes antes del commit | `CA-TEST-LOCAL` |
| `TEST-SERVER` | Si se prueba contra el server local, `./scripts/check-server.sh` antes de avisar | `CA-TEST-SERVER` |
| `TEST-USER-FLOW` | No se marca `pending_release` sin pruebas manuales confirmadas por el usuario | `CA-TEST-USER-FLOW` |
| `DELIV-WORKTREE` | Trabajo en `../p40la-ihost-spec-150`, rama `feature/SPEC-150` | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | Merge y push solo con confirmación explícita; sin `--force` | `CA-DELIV-CONFIRM` |
| `DELIV-DOCKER` | Imagen multi-arch `linux/amd64, linux/arm/v7, linux/arm64` al publicar | `CA-DELIV-DOCKER` |
| `DELIV-MCP-DOCS` | El mismo commit actualiza `docs/mcp/` (tool, catálogo y desarrollo) | `CA-DELIV-MCP-DOCS` |
| `DELIV-MCP-DOCS-SUBAGENT` | La redacción de `docs/mcp/` se delega a un subagente y se revisa | `CA-DELIV-MCP-DOCS-SUBAGENT` |
| `UI-*` / `UX-*` | N/A: no hay cambios de frontend, rutas, modales ni pantallas | — |

> Reglas fuente: `docs/rules/10-architecture.md`, `40-data.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001 — Tool `entities_search`**: una sola tool de solo lectura con `entity` (`service`, `home`, `bank`, `institution`), `query` opcional y `limit` opcional (default 10, máximo 50).
2. **REQ-002 — Búsqueda de hogares**: por nombre, sin distinguir mayúsculas ni acentos, con coincidencia exacta, prefijo o contenido. Devuelve `id`, `name`, `address` y `usage`.
3. **REQ-003 — Búsqueda de servicios**: por nombre, global o filtrada por hogar (`home` o `home_id`). Devuelve `id`, `name`, `home_id`, `institution`, `currency_id`, `frequency`, `suggested_amount` y `active`.
4. **REQ-004 — Búsqueda de bancos**: por nombre o `short_code`, con `id`, `name`, `short_code`, `country` y `active`.
5. **REQ-005 — Búsqueda de instituciones**: por nombre, con `id`, `name` y `category_id`.
6. **REQ-006 — Solo lectura**: la tool no escribe, no exige `P40LA_MCP_ALLOW_WRITES` y funciona con las escrituras deshabilitadas. Se anota con `ReadOnlyHint: true`.
7. **REQ-007 — Orden determinista**: exactos primero, luego prefijo, luego contenido; dentro de cada grupo por nombre y `id` ascendente.
8. **REQ-008 — Sin resultados no es error**: responde `count: 0`, `items: []` y un `hint` con sugerencias; nunca `isError: true` por una búsqueda vacía.
9. **REQ-009 — Registro**: se monta en `internal/mcp/tools/all/register.go` y aparece en `tools/list` junto a las 7 tools existentes (total 8).
10. **REQ-010 — Documentación**: `docs/mcp/tools/entities.md` (o `search.md`) documenta entradas, salidas, orden, límites y ejemplos; se actualizan catálogo y guía de desarrollo.

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-011 — Filtro de hogar por nombre**: `home` se resuelve con la misma normalización de SPEC-146; si no existe → `not_found` con hint; si es ambiguo → `ambiguous_reference` con candidatos y se sugiere `home_id`.
2. **REQ-012 — `truncated`**: si hay más resultados que `limit`, la respuesta incluye `truncated: true` y `total` con el conteo completo.
3. **REQ-013 — `query` vacía**: lista las primeras `limit` entidades ordenadas por nombre, para descubrimiento inicial.
4. **REQ-014 — Límite validado**: `limit` fuera de 1..50 responde `invalid_value` sin llamar a la API.

### 3.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-015 — Búsqueda por categoría/país**: filtros opcionales `category` para instituciones y `country` para bancos. Se difiere.
2. **REQ-016 — Tolerancia a errores de tipeo**: distancia de Levenshtein o trigramas. Se difiere; primero medir necesidad real.
3. **REQ-017 — Paginación**: `offset` cuando el backend exponga paginación. Se difiere.

### 3.4 Requerimientos No Funcionales

- **Rendimiento**: una llamada HTTP por búsqueda (dos si se resuelve `home` por nombre); respuesta < 500 ms en LAN con hasta 1.000 entidades.
- **Memoria**: sin caché ni índices en memoria; el filtrado recorre la respuesta del backend y se descarta.
- **Seguridad**: requiere la API key global existente; no agrega superficie pública ni escrituras; no loguea el término de búsqueda.
- **Almacenamiento**: cero migraciones y cero archivos nuevos de datos.
- **iHost**: sin cambios en el server web; el MCP sigue siendo el mismo binario liviano.
- **Compatibilidad**: no rompe las tools de facturas ni las de alta de SPEC-146.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

- SPEC-145: `bills_*` reciben `service_id`; no existe forma de descubrirlo desde el MCP.
- SPEC-146: tools de alta resuelven nombres contra `GET /api/{homes,currencies,institution-categories}` y detectan duplicados, pero no exponen búsquedas.
- Endpoints existentes y reutilizables:
  - `GET /api/services` (filtro `home_id`), `GET /api/homes`, `GET /api/banks`, `GET /api/institutions`.
  - Todos devuelven listas completas con los campos necesarios; no hay endpoint de búsqueda.
- `internal/mcp/tools/entities` ya tiene `normalize`, `findByName` y resolución de hogar; reutilizarlos evita duplicar reglas.

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| A. Una tool `entities_search` con enum `entity` | Menos tools, schema único, fácil de descubrir | Un `switch` por entidad | ✅ Seleccionada |
| B. Cuatro tools (`homes_search`, `services_search`, …) | Schemas específicos | Cuatro entradas más en `tools/list` y docs duplicadas | ❌ Rechazada |
| C. Endpoint backend `/api/search` | Búsqueda del lado SQL, escalable | Toca server, SQL y tests; no hace falta para el volumen actual | ❌ Rechazada |
| D. Resolver nombres dentro de `bills_*` | Cero tools nuevas | Duplica resolución en cada tool y sigue sin descubrir IDs | ❌ Rechazada |
| E. Búsqueda en el cliente MCP con caché | Rápida tras la primera llamada | Caché obsoleta y memoria extra en el iHost | ❌ Rechazada |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001 — Una tool con enum en lugar de cuatro tools**
- **Contexto**: el agente necesita descubrir IDs de cuatro entidades; cuatro tools aumentan la lista y la documentación.
- **Decisión**: `entities_search` con `entity` obligatorio y `query`/`limit` opcionales.
- **Consecuencias**: una entrada más en `tools/list` (total 8); el handler usa un `switch` y comparte validación, orden y formato.

**ADR-002 — Búsqueda del lado MCP sobre los listados existentes**
- **Contexto**: el volumen actual (cientos de entidades) no justifica un endpoint nuevo; tocar el server suma SQL, migraciones y superficie.
- **Decisión**: traer la lista por HTTP y filtrar/ordenar en el MCP.
- **Consecuencias**: cero cambios en backend; si el volumen crece a decenas de miles, se evalúa paginación o endpoint dedicado (REQ-017).

**ADR-003 — Devolver candidatos, nunca elegir**
- **Contexto**: dos servicios pueden llamarse igual en hogares distintos; auto-elegir produciría facturas en el servicio equivocado.
- **Decisión**: la tool devuelve todos los candidatos dentro del `limit`, ordenados; el agente elige y usa el `id`.
- **Consecuencias**: un paso adicional para el agente en casos ambiguos; cero riesgo de escritura sobre la entidad equivocada.

**ADR-004 — La búsqueda no requiere el gate de escritura**
- **Contexto**: `P40LA_MCP_ALLOW_WRITES` protege cambios; buscar es lectura.
- **Decisión**: `entities_search` funciona con escrituras deshabilitadas y se anota `ReadOnlyHint`.
- **Consecuencias**: el agente puede descubrir IDs incluso en modo seguro; no hay riesgo de mutación.

---

## 5. Diseño Técnico

### 5.1 Diagrama de arquitectura

```
claude.ai / agente
        │ tools/call entities_search
        ▼
internal/mcp/tools/entities/search.go
        │ GET /api/{services,homes,banks,institutions}
        ▼
p40la-ihost :8088  →  internal/api → internal/services → SQLite
```

### 5.2 Componentes

#### 5.2.1 `internal/mcp/tools/entities/search.go`
- **Responsabilidad**: validar entrada, resolver el filtro de hogar, listar, normalizar, ordenar y recortar.
- **Interfaz**: `entities_search` registrada en `entities.Register`.
- **Dependencias**: `internal/mcp/api`, `internal/mcp/config`, `internal/mcp/tools/toolkit`, helpers de `entities` (`normalize`, `resolveHome`).
- **Ubicación**: `internal/mcp/tools/entities/search.go`.

#### 5.2.2 `internal/mcp/tools/all/register.go`
- **Responsabilidad**: sin cambios: `entities.Register` ya monta el paquete; la tool nueva se suma dentro del mismo `Register`.

#### 5.2.3 `internal/mcp/api`
- **Responsabilidad**: sin métodos nuevos; se reutilizan `ListServices`, `ListHomes`, `ListBanks` y `ListInstitutions`.
- **Ubicación**: `internal/mcp/api/entities.go`.

### 5.3 Modelo de datos

No hay cambios de esquema ni DTOs nuevos en el backend. La salida usa un ítem genérico con campos opcionales:

```
SearchItem:
- id: int64
- name: string
- address, usage: string        (home)
- home_id, currency_id: int64   (service)
- institution: string           (service)
- frequency: string             (service)
- suggested_amount: float       (service)
- active: bool                  (service, bank)
- short_code, country: string   (bank)
- category_id: int64            (institution)
```

### 5.4 APIs / Contratos

#### Tool: `entities_search`

**Entrada**
```json
{
  "entity": "service",
  "query": "internet",
  "home": "Hogar Demo",
  "limit": 10
}
```

- `entity` obligatorio: `service`, `home`, `bank` o `institution`.
- `query` opcional: si está vacío, lista las primeras `limit` entidades por nombre.
- `home`/`home_id` solo para `service`; si se envían ambos, gana `home_id`.
- `limit` 1..50, default 10.

**Salida 200**
```json
{
  "entity": "service",
  "query": "internet",
  "count": 1,
  "total": 1,
  "truncated": false,
  "items": [
    {
      "id": 12,
      "name": "Internet Demo",
      "home_id": 7,
      "institution": "Proveedor Demo",
      "currency_id": 1,
      "frequency": "monthly",
      "suggested_amount": 1000,
      "active": true
    }
  ]
}
```

**Error**
```json
{
  "error": "invalid_value",
  "message": "entity debe ser service, home, bank o institution.",
  "hint": "Valores válidos: service, home, bank, institution."
}
```

| Código | Cuándo |
|---|---|
| `invalid_request` | `entity` ausente o `query` demasiado larga |
| `invalid_value` | `entity` o `limit` fuera de rango |
| `not_found` | `home`/`home_id` del filtro no existe (con hint) |
| `ambiguous_reference` | `home` del filtro coincide con más de un hogar |
| `api_error` | Error HTTP del backend |
| `upstream_error` | Backend inalcanzable |

> Una búsqueda sin coincidencias **no** es error: `count: 0` con `hint`.

### 5.5 Dependencias

- **Internas**: SPEC-146 (paquete `entities`, `toolkit` y cliente con `List*`) y SPEC-148 (despliegue/OAuth para la validación en vivo). Si SPEC-146 no está en `main`, la implementación parte de su rama o se rebasa.
- **Externas**: ninguna nueva.

---

## 6. Criterios de Aceptación

### 6.1 Funcionales

- [ ] **CA-MCP-SEARCH-001**: `tools/list` incluye `entities_search` con schema válido y descripción en español.
- [ ] **CA-MCP-SEARCH-002**: buscar hogares por un fragmento devuelve coincidencias sin distinguir mayúsculas ni acentos y con `id`.
- [ ] **CA-MCP-SEARCH-003**: buscar servicios sin filtro devuelve coincidencias globales; con `home`/`home_id` solo del hogar indicado.
- [ ] **CA-MCP-SEARCH-004**: buscar bancos por nombre o `short_code` y instituciones por nombre devuelve `id` y campos mínimos.
- [ ] **CA-MCP-SEARCH-005**: una búsqueda sin coincidencias responde `count: 0`, `items: []`, `hint` y **no** `isError`.
- [ ] **CA-MCP-SEARCH-006**: dos servicios con el mismo nombre devuelven ambos candidatos con `home_id`; la tool no elige.
- [ ] **CA-MCP-SEARCH-007**: el orden es exacto → prefijo → contenido, y dentro del grupo por nombre e `id`.
- [ ] **CA-MCP-SEARCH-008**: `limit` default 10, máximo 50; al superar el total, `truncated: true` y `total` correcto.
- [ ] **CA-MCP-SEARCH-009**: con `P40LA_MCP_ALLOW_WRITES=false` la tool funciona y devuelve resultados.
- [ ] **CA-MCP-SEARCH-010**: un `home` inexistente responde `not_found` con hint; uno ambiguo responde `ambiguous_reference` con candidatos.
- [ ] **CA-MCP-SEARCH-011**: `docs/mcp/` documenta la tool, el orden, los límites y el flujo “buscar → usar id”.
- [ ] **CA-MCP-SEARCH-012**: las tools de facturas y de alta siguen pasando sus tests.

### 6.2 No funcionales

- [ ] **CA-NF-001**: ninguna tool contiene SQL ni importa `internal/storage`/`internal/services`.
- [ ] **CA-NF-002**: `go build ./...` y `go test ./...` pasan; sin dependencias nuevas.
- [ ] **CA-NF-003**: una búsqueda hace como máximo 2 llamadas HTTP y responde en < 500 ms en LAN.
- [ ] **CA-NF-004**: la documentación y los tests no exponen PII, secretos ni datos reales.

### 6.3 Testing

- **Unit tests**: normalización, ranking, `limit`/`truncated`, query vacía, validaciones y filtro de hogar con una API falsa.
- **Protocol tests**: `tools/list` con `entities_search` y una llamada end-to-end con API falsa.
- **Client tests**: N/A (no hay métodos nuevos; se cubren los existentes).
- **Integración manual**: contra una copia de producción en `/tmp`, buscar un servicio real por fragmento y usar su `id` en `bills_list`.

---

## 7. Plan de Implementación

### 7.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | `search.go`: schema, validaciones, normalización y ranking | 0.5 día | SPEC-146 |
| 2 | Filtro de hogar, `truncated`/`total` y tests unitarios | 0.5 día | Fase 1 |
| 3 | Protocol test, registro y verificación de 8 tools | 0.25 día | Fase 2 |
| 4 | Docs MCP (subagente), imagen multi-arch y deploy | 0.5 día | Fase 3 |
| 5 | Validación manual contra copia de producción y confirmación | 0.5 día | Fase 4 |

### 7.2 Milestones

1. **MVP**: `entities_search` funciona para las cuatro entidades con tests y aparece en `tools/list`.
2. **V1.0**: filtro de hogar, ranking completo, docs y validación en vivo.

---

## 8. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| SPEC-146 no está en `main` al implementar | Alta | Medio | Partir de `feature/SPEC-146` o rebasar tras su release; declarado en dependencias |
| Dos entidades con el mismo nombre | Media | Alto | Devolver candidatos con contexto (`home_id`, `short_code`) y nunca auto-elegir |
| Listados grandes degradan la respuesta | Baja | Medio | `limit` máximo 50 y filtro `home_id`; evaluar paginación si crece |
| El término de búsqueda contiene PII en logs | Baja | Alto | No loguear `query`; el logger HTTP ya no registra argumentos |
| Resultados dependen de datos reales | Media | Medio | Validar contra copia de producción en `/tmp`; nunca escribir |
| Drift con las tools de alta | Baja | Bajo | Reutilizar `normalize`, `resolveHome` y `toolkit` del mismo paquete |

---

## 9. Notas y Referencias

- Reglas aplicadas: `docs/rules/10-architecture.md`, `40-data.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.
- SPEC-145: tools de facturas y API key global.
- SPEC-146: tools de alta y helpers de resolución por nombre.
- SPEC-147/148: servidor MCP remoto, OAuth y despliegue en el iHost.
- Endpoints: `internal/api/routes.go` (`homes`, `services`, `banks`, `institutions`).
- Documentación: `docs/mcp/tools/entities.md`, `docs/mcp/development.md`.
- Ejemplos sintéticos: `Servicio Demo`, `Hogar Demo`, `Banco Demo`, `Proveedor Demo`, `NIO`.

---

## 10. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-24 | opencode | Creación inicial de la especificación |
| 2026-11-24 | opencode | Inicio de implementación: `entities_search` con ranking, límites y tests |
