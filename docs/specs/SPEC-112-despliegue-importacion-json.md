---
title: "Despliegue único de automation y plan de importación del export JSON"
id: "SPEC-112"
status: "released"
author: "opencode"
created: "2026-10-21"
updated: "2026-10-21"
github_issue: 115
---

# Despliegue único de automation y plan de importación del export JSON

**ID**: SPEC-112  
**Estado**: released  
**Autor**: opencode  
**Creado**: 2026-10-21  
**Actualizado**: 2026-10-21

---

## 1. Resumen Ejecutivo

Las specs 109–111 dejaron el esquema, el backend y el frontend de automation dentro de `p40la-ihost`. Falta el cierre operativo: **una sola imagen multi-arch** que compile el módulo privado de plugins, **un solo contenedor** en el puerto `8088` usando el volumen `p40la`, y un **procedimiento repetible** para importar el export JSON de `p40la-ihost-automation` en la DB de producción del iHost.

El usuario entregó un export real (`p40la-export-*.json`, formato `p40la-ihost-automation`, `format_version: 1`). Este archivo **no se commitea**: contiene credenciales, webhooks, facturas y logs. La spec agrega `--from-json` al importador existente para consumirlo directamente, valida el resultado sobre una copia de producción y deja en el repo un runbook de despliegue/importación/rollback.

Resultado esperado: el día del deploy se detiene el contenedor viejo, se respalda la DB, se importa el JSON con un comando, se verifica integridad/conteos, se levanta la imagen única y se retira `p40la-ihost-automation`. Sin intervención manual sobre filas ni riesgo de perder cuentas, facturas o logs.

Consideraciones de iHost: un proceso menos, misma SQLite, imagen distroless, puerto `8088`, volumen `p40la`. El importador es offline y no corre en runtime.

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: `cmd/import-automation` acepta `--from-json <archivo>` y reutiliza toda la lógica existente (instituciones, plugins, cuentas, facturas, logs, settings, `--dry-run`, `--verify`, `--institution-map`, idempotencia).
2. **REQ-002**: Validar el export: `export.app == "p40la-ihost-automation"`, `export.format_version == 1` y presencia de `tables.{apps,plugins,accounts,bills,webhook_logs,settings}`. Error claro si no cumple.
3. **REQ-003**: Materializar el JSON en una SQLite temporal con el esquema viejo de automation, preservando IDs y columnas. Los valores JSON objeto/array (`credentials`, `raw`, `payload`) se serializan como texto JSON; `null` se preserva.
4. **REQ-004**: La SQLite temporal y su directorio se crean con permisos restrictivos y se eliminan siempre (éxito o error). Nunca se escribe el JSON ni datos derivados dentro del repo.
5. **REQ-005**: `--from` y `--from-json` son mutuamente excluyentes y exactamente uno es obligatorio.
6. **REQ-006**: Tests con un fixture JSON **sintético** que cubran importación feliz, idempotencia, export inválido (app/version/tablas) y flags incombinables.
7. **REQ-007**: Dockerfile con build del módulo privado mediante secreto de BuildKit (`GITHUB_TOKEN`), sin dejar el token en capas ni en la imagen final; `git` disponible en el builder.
8. **REQ-008**: Imagen multi-arch `linux/amd64,linux/arm/v7,linux/arm64`, puerto `8088`, `DATA_DIR=/app/data`, volumen `p40la`.
9. **REQ-009**: Runbook en `docs/deployment/automation-import.md` con: precondiciones, backup, stop del contenedor, importación, verificación, rollback, limpieza y fallback SQL directo.
10. **REQ-010**: Procedimiento de retiro del contenedor `p40la-ihost-automation` y archivo del repo viejo, posterior a la verificación.
11. **REQ-011**: Ningún agente escribe en producción ni despliega sin confirmación explícita del usuario. El runbook es para ejecución asistida.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-012**: `--dry-run` con `--from-json` reporta conteos del JSON sin tocar la DB destino.
2. **REQ-013**: `--verify` con `--from-json` compara los conteos del JSON contra `automation_*`.
3. **REQ-014**: El resumen imprime apps reutilizadas/creadas/mapeadas y advierte si una app creó una institución nueva (posible duplicado).
4. **REQ-015**: El runbook explica cómo construir el `--institution-map` con los nombres exactos de `tables.apps` para evitar duplicados.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-016**: Comando auxiliar documentado para listar las apps del export sin exponer credenciales (`jq '.export.tables.apps[].name'`).

### 2.4 Requerimientos No Funcionales

- **Seguridad**: el JSON y la SQLite temporal contienen secretos; se manejan en `/tmp`, permisos `0600`/`0700`, y se borran. No se loguean valores.
- **Integridad**: import en una transacción; `PRAGMA integrity_check` y `foreign_key_check` al final.
- **Idempotencia**: reejecutar no duplica cuentas/facturas/logs (claves primarias/`INSERT OR IGNORE`).
- **iHost**: imagen distroless, sin shell en runtime, `CGO_ENABLED=0`, multi-arch.
- **Reproducibilidad**: build con `go 1.26`, módulo privado fijado por versión en `go.mod`.

---

## 3. Investigación y Decisiones Técnicas

### 3.1 Formato del export

Estructura observada (sin datos):

```json
{
  "export": {
    "format_version": 1,
    "app": "p40la-ihost-automation",
    "exported_at": "...",
    "exported_by": "...",
    "schema_migrations": ["..."],
    "counts": { "accounts": 6, "apps": 5, "bills": 80, "plugins": 5, "settings": 3, "users": 1, "webhook_logs": 838 },
    "tables": {
      "users": [ ... ],
      "settings": [ { "key": "...", "value": "..." } ],
      "apps": [ { "id": 1, "name": "..." } ],
      "accounts": [ { "id": 1, "app_id": 1, "identifier": "...", "credentials": { ... }, ... } ],
      "plugins": [ ... ],
      "bills": [ { "id": 1, "account_id": 1, "raw": { ... }, ... } ],
      "webhook_logs": [ { "id": 1, "payload": { ... }, ... } ]
    }
  }
}
```

Columnas por tabla: `apps(id,name,created_at,updated_at)`, `accounts(id,app_id,identifier,label,credentials,plugin_name,webhook_url,schedule_time,schedule_enabled,last_run_at,schedule_frequency,schedule_interval,schedule_days,created_at,updated_at)`, `plugins(id,name,version,source,description,status,created_at,updated_at)`, `bills(id,account_id,plugin_name,plugin_version,source,status,error,raw,fetched_at,created_at)`, `webhook_logs(id,account_id,webhook_url,status,payload,response,ran_at,http_status)`, `settings(key,value)`.

### 3.2 Decisiones

**ADR-001: Reutilizar el importador SQLite**
- `--from-json` materializa el JSON en una SQLite temporal con el esquema viejo y luego ejecuta el mismo `runImport` que `--from`.
- Evita duplicar la lógica de matching de instituciones, stubs de plugins, idempotencia y verificación.
- La SQLite temporal se elimina al terminar.

**ADR-002: No importar usuarios ni secretos de auth**
- `users` del export se ignora: p40la tiene su propia autenticación.
- `settings` solo migra `webhook_enabled` y `language` (comportamiento actual de SPEC-109); no toca `webhook_api_key` de p40la.

**ADR-003: Build del módulo privado con secreto**
- El módulo `github.com/paulomcnally/p40la-ihost-automation-plugins` es privado.
- El builder configura `git url.insteadOf` con un token pasado como `--secret id=github_token`; el token no queda en la imagen final (distroless) ni en el historial de capas.
- `GOPRIVATE=github.com/paulomcnally/*` evita el proxy público.

**ADR-004: Volumen de producción `p40la`**
- El compose de desarrollo usa un volumen nombrado; el de producción debe declarar `p40la` como externo para conservar la DB existente.
- El contenedor viejo se detiene antes de importar; su volumen se conserva como respaldo hasta validar.

**ADR-005: Importación asistida, no automática**
- El agente prepara el comando y valida sobre copia, pero la escritura en producción la ejecuta el usuario o el agente con confirmación explícita y contenedor detenido.

---

## 4. Diseño Técnico

### 4.1 Importador `--from-json`

Archivos:
- `cmd/import-automation/main.go`: flag `--from-json`, validación mutua, `materializeJSONSource`.
- `cmd/import-automation/json_source.go`: parseo, esquema temporal, inserción.
- `cmd/import-automation/json_source_test.go`: fixture sintético.
- `cmd/import-automation/main_test.go`: caso de flags.

Flujo:
```
main → opts.sourceJSONPath
  → materializeJSONSource(path) → tempDB path + cleanup
  → opts.sourcePath = tempDB
  → runImport(...) (sin cambios)
  → cleanup()
```

Esquema temporal: `CREATE TABLE` de `apps`, `plugins`, `accounts`, `bills`, `webhook_logs`, `settings` (más `users` vacía si se necesita compatibilidad; el importador no la lee).

Validaciones:
- JSON malformado → error con la ruta.
- `app` distinto → error.
- `format_version` distinto de 1 → error.
- tabla requerida ausente → error.
- columna desconocida → se ignora (solo se insertan columnas conocidas); columna requerida ausente → `NULL`/default.

### 4.2 Dockerfile

```dockerfile
FROM golang:1.26-alpine AS backend-builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=secret,id=github_token \
    GITHUB_TOKEN="$(cat /run/secrets/github_token)" \
    GOPRIVATE=github.com/paulomcnally/* \
    git config --global url."https://x-access-token:${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/" && \
    go mod download
...
```

Build:
```bash
docker buildx build \
  --platform linux/amd64,linux/arm/v7,linux/arm64 \
  --secret id=github_token,src="$HOME/.config/p40la/github_token" \
  -t paulomcnally/p40la-ihost:<version> \
  --push .
```

### 4.3 Compose de producción

```yaml
services:
  p40la-ihost:
    image: paulomcnally/p40la-ihost:<version>
    container_name: p40la-ihost
    restart: unless-stopped
    ports: ["8088:8088"]
    volumes:
      - p40la:/app/data
    environment:
      - PORT=8088
      - DATA_DIR=/app/data
volumes:
  p40la:
    external: true
```

### 4.4 Runbook

`docs/deployment/automation-import.md` con las fases:
1. Precondiciones y copia del export a `/tmp` (permisos `0600`).
2. Detener `p40la-ihost`; backup `.backup` + `integrity_check` + descarga local.
3. Copiar el export al host de trabajo y ejecutar `import-automation --from-json` sobre la copia descargada.
4. Verificar conteos e integridad; comparar `--verify`.
5. Subir la DB modificada al volumen `p40la` (contenedor detenido) o ejecutar el importador en un contenedor one-off si el usuario lo prefiere.
6. Levantar la imagen única; `GET /health`; login; revisar `/automation`.
7. Detener/eliminar el contenedor `p40la-ihost-automation`; conservar su volumen/backup.
8. Limpiar `/tmp` (JSON, DB temporal, map) y rotar credenciales si el JSON circuló.
9. Rollback: restaurar el backup y volver a la imagen anterior.

Fallback SQL directo: solo si el importador no puede ejecutarse; se documenta el esqueleto `BEGIN; INSERT OR IGNORE ...; COMMIT;` con placeholders, sin datos reales.

### 4.5 Impacto en iHost

- Un contenedor menos (se retira automation).
- Misma DB SQLite, mismas migraciones.
- Sin cambios de rendimiento en runtime; el importador es offline.

---

## 5. Criterios de Aceptación

- [ ] **CA-001**: `go test ./cmd/import-automation` pasa con el fixture JSON sintético.
- [ ] **CA-002**: `--from` y `--from-json` juntos fallan con mensaje claro; ninguno también falla.
- [ ] **CA-003**: Un JSON con `app` o `format_version` incorrectos falla sin escribir en destino.
- [ ] **CA-004**: El export real se importa sobre una copia de producción: 6 cuentas, 80 facturas, 838 logs, 5 plugins; 4 instituciones reutilizadas + 1 mapeada, 0 creadas; `integrity_check` y `verify` OK.
- [ ] **CA-005**: Reejecutar el import no duplica filas.
- [ ] **CA-006**: `docker buildx build` multi-arch documentado y Dockerfile compila con el secreto; el token no aparece en la imagen final.
- [ ] **CA-007**: `docker-compose.yml` usa el volumen externo `p40la` y el puerto `8088`.
- [ ] **CA-008**: `docs/deployment/automation-import.md` existe, no contiene datos reales y cubre backup/import/verify/rollback/cleanup.
- [ ] **CA-009**: No se commitea el JSON, la SQLite temporal ni el `--institution-map` con datos.
- [ ] **CA-010**: El retiro del contenedor viejo queda documentado y condicionado a la verificación.

---

## 6. Plan de Implementación

| Fase | Descripción | Estimación |
|---|---|---|
| 1 | `--from-json`: parser, esquema temporal y validaciones | 1.5 h |
| 2 | Tests sintéticos (happy path, idempotencia, errores, flags) | 1 h |
| 3 | Validación con el export real sobre copia de producción | 0.5 h |
| 4 | Dockerfile con secreto y compose con volumen `p40la` | 1 h |
| 5 | Runbook de importación/despliegue/rollback | 1 h |
| 6 | Verificación final (`go test`, build, multi-arch dry-run) | 0.5 h |

Dependencias: SPEC-109 (importador), SPEC-110/111 (app unificada). Sin dependencias nuevas de runtime.

---

## 7. Riesgos y Mitigaciones

| Riesgo | Mitigación |
|---|---|
| El export cambia de formato | Validar `format_version` y fallar con mensaje claro; documentar la versión soportada. |
| Duplicar instituciones por nombres distintos | `--institution-map` con nombres exactos de `tables.apps`; el resumen advierte instituciones creadas. |
| Filtrar credenciales en logs/temporales | No loguear valores; `/tmp` con permisos restrictivos; borrar al terminar. |
| Token de GitHub en la imagen | Build secret de BuildKit; stage final distroless sin git ni token. |
| Importar sobre la DB en uso | Contenedor detenido + backup + `integrity_check`; nunca WAL activo. |
| Perder la DB de automation | No se borra el volumen viejo hasta validar; backup previo. |
| Build arm/v7 falla por dependencias | `CGO_ENABLED=0` y `modernc.org/sqlite` puro Go; probar los 3 manifests. |

---

## 8. Historial de Cambios

| Fecha | Autor | Cambio |
|---|---|---|
| 2026-10-21 | opencode | Creación de la spec e inicio de implementación (issue #115). |
| 2026-10-21 | opencode | Implementado `--from-json` (materialización a SQLite temporal, validación de app/format_version/tablas, limpieza, tests sintéticos). Validado con el export real sobre copia de producción: 6 cuentas, 80 facturas, 838 logs, 5 plugins, 4 instituciones reutilizadas + 1 mapeada, 0 creadas, `integrity_check` y `--verify` OK. Dockerfile con secreto `gh_token`, compose con volumen externo `p40la`, workflow con `P40LA_PLUGINS_TOKEN` y runbook `docs/deployment/automation-import.md`. `docker compose config` OK; el build de imagen no se pudo ejecutar por timeout de DNS contra `registry-1.docker.io`. |
| 2026-10-21 | opencode | Release aprobado por el usuario. Import validado con el export real sobre copia de producción. Código mergeado a `main`; publicación de la imagen 0.5.0 pendiente de configurar `P40LA_PLUGINS_TOKEN` en GitHub Actions. Estado: released. |
