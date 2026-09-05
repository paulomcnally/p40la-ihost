---
title: "Scripts para tareas recurrentes del agente (test-user y check-server) para reducir tokens del prompt"
id: "SPEC-059"
status: "released"
author: "p40la-ihost-team"
created: "2026-09-04"
updated: "2026-09-04"
github_issue: 60
---

# Scripts para tareas recurrentes del agente (test-user y check-server) para reducir tokens del prompt

**ID**: SPEC-059  
**Estado**: released  
**Autor**: p40la-ihost-team  
**Creado**: 2026-09-04  
**Actualizado**: 2026-09-04

---

## 1. Resumen Ejecutivo

AGENTS.md (la fuente de verdad de las reglas del agente) se carga en contexto en **cada sesión**, y contiene varios procedimientos operativos largos escritos como pasos de prompt. El más costoso es el de "crear usuario de prueba y autenticar": ~1.150 caracteres (~300 tokens por sesión) que describen 4 pasos manuales (generar hash bcrypt vía `go run /tmp/genhash.go`, insertar en SQLite interpolando `$HASH`, login con curl, y uso de cookies). Además de ser costoso en tokens, depende de `/tmp/genhash.go`, un archivo efímero que desaparece con cada reinicio y que el agente debe recrear a mano ("requiere un Go file con..."), lo que genera más tokens en runtime y riesgo de errores de quoting/interpolación del hash (`$2a$10$...` en strings con comillas dobles).

Esta spec convierte esos procedimientos en **scripts ejecutables del repo** (`scripts/`), de modo que el prompt solo diga *"ejecutá `./scripts/create-test-user.sh`"* (≈50 tokens). Ahorro estimado: **~250 tokens por sesión** de forma permanente, más eliminar la fragilidad de `/tmp/genhash.go` y los errores de interpolación. Se incluye también `scripts/check-server.sh` para reemplazar la regla de verificación post-server (health + revisión de logs), que hoy son 3 líneas de prompt.

Impacto iHost: nulo. Son herramientas de desarrollo del agente (dev machine), no forman parte del binario servidor. El único cambio de código Go es un helper mínimo `scripts/genhash.go` que reutiliza `golang.org/x/crypto/bcrypt`, dependencia ya presente en `go.mod` (v0.26.0).

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Crear `scripts/create-test-user.sh` que, en un solo comando: genere el hash bcrypt del password, cree/actualice el usuario de prueba en la DB, haga login y deje las cookies listas en `/tmp/cookies.txt`.
2. **REQ-002**: El script debe aceptar configuración por variables de entorno con defaults sensatos: `DB_PATH` (`data/app.db`), `EMAIL` (`test@test.com`), `PASSWORD` (`test1234`), `PORT` (`8088`), `COOKIE_JAR` (`/tmp/cookies.txt`).
3. **REQ-003**: Crear helper `scripts/genhash.go` (Go, `package main`) que imprime el hash bcrypt por stdout. Debe ser idempotente para cualquier password pasado como argumento.
4. **REQ-004**: Actualizar AGENTS.md sección 0: reemplazar los 4 pasos del proceso de creación de usuario de prueba por una referencia de una línea al script.
5. **REQ-005**: El INSERT debe ser idempotente: `INSERT ... ON CONFLICT(email) DO UPDATE` (la tabla `users` ya tiene `email TEXT UNIQUE NOT NULL`).
6. **REQ-006**: Evitar errores de quoting del hash: usar parámetros ligados de sqlite3 (`?1`, `?2`) en vez de interpolación de `$HASH` en comillas dobles.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-007**: Crear `scripts/check-server.sh` que verifique `GET /health` esperando `"status":"ok"` con reintentos (hasta ~10s), y ante fallo muestre las últimas líneas del log `/tmp/p40la-server.log` y salga con código distinto de 0.
2. **REQ-008**: Actualizar AGENTS.md para reemplazar la regla de verificación post-server (línea del `curl /health` + revisión de logs) por referencia a `scripts/check-server.sh`.
3. **REQ-009**: El script `create-test-user.sh` debe imprimir al final un comando listo para copiar/pegar usando las cookies (ej: `curl -b /tmp/cookies.txt http://localhost:8088/api/...`).
4. **REQ-010**: Verificar el comando de ejemplo contra el login real (`POST /api/login` devuelve cookie de sesión).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-011**: Soporte remoto (iHost vía SSH) para crear el usuario de prueba en el volumen `p40la` (`/app/data/app.db`) siguiendo `docs/ssh-ihost-access.md`. Se deja fuera del alcance v1 y se documenta como trabajo futuro.
2. **REQ-012**: Migrar otros procedimientos del prompt a scripts (p. ej. backup/restore de la DB local y remota) en iteraciones posteriores.
3. **REQ-013**: Arreglar el test time-dependent `TestBillPayBill` en `internal/services/finance_test.go`: hardcodea los periodos 2026-09/2026-10, que colisionan con la factura auto-generada cuando el mes actual coincide (fallo detectado el 2026-09-04). Los periodos de las facturas segunda y tercera deben derivarse del periodo de la factura auto-generada (usa `time.Now`), no hardcodearse.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: `go run` del helper tiene cold-start (~1-2s), aceptable para una herramienta de dev. Sin impacto en runtime del server.
- **Seguridad**: El password default es solo para pruebas locales. El script no debe loguear el hash ni el password.
- **Almacenamiento**: Dos scripts shell (~40 líneas) + un helper Go (~20 líneas). Insignificante.
- **Disponibilidad**: Si `sqlite3` o `go` no están instalados, el script debe fallar con mensaje claro y exit code ≠ 0.
- **iHost**: Sin cambios en el binario servidor ni en la imagen Docker. Cero impacto en el iHost.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- AGENTS.md sección 0 (líneas 28-42): reglas críticas con procedimientos embebidos en prompt.
- Tabla `users` (`migrations/0001_create_users_and_settings.up.sql`): `email TEXT UNIQUE NOT NULL` → permite `ON CONFLICT(email) DO UPDATE`.
- `POST /api/login` (`internal/api/handlers.go:145`): body `{email, password}`, setea cookie de sesión, responde `{email}`.
- `GET /health` (`internal/api/handlers.go:179`): responde `{"status":"ok", "timestamp": ...}`.
- `go.mod`: `golang.org/x/crypto v0.26.0` ya es dependencia directa → el helper bcrypt es gratis.
- `/tmp/genhash.go` actual: efímero (se pierde en cada reinicio), el prompt obliga a recrearlo a mano.
- Precedentes de scripts del repo: `scripts/build.sh`, `scripts/run-local.sh`, `scripts/release.sh` (todos bash con `set -euo pipefail` y `SCRIPT_DIR`/`ROOT_DIR`).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Script bash `scripts/create-test-user.sh` | Un comando, cero runtime server, sigue precedente de `scripts/` | Requiere `sqlite3` y `go` en la dev machine | ✅ Seleccionada |
| Endpoint de registro en la API | No requiere sqlite3 | Agrega superficie de ataque + código de negocio sin necesidad real | ❌ Rechazada |
| Python3 para generar hash (`python3 -c import bcrypt`) | Rápido, sin cold-start Go | `bcrypt` de Python no está garantizado en todo entorno; Go ya es dependencia del repo | ❌ Rechazada |
| Interpolación `$HASH` en comillas dobles (como hoy) | Cero cambios | Corrompe el hash (`$2a$10$...` → `$2` expande a posicional vacío) | ❌ Rechazada |
| Parámetros ligados de sqlite3 (`?1`, `?2`) | Sin riesgo de quoting, robusto | Requiere sqlite3 ≥ 3.14 (presente en cualquier sistema moderno) | ✅ Seleccionada para el INSERT |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Scripts bash autónomos en `scripts/`**
- **Contexto**: Los procedimientos del agente viven hoy en AGENTS.md como texto de prompt repetido en cada sesión.
- **Decisión**: Moverlos a scripts ejecutables del repo siguiendo el patrón existente (`set -euo pipefail`, `SCRIPT_DIR`/`ROOT_DIR`, variables de entorno con defaults).
- **Consecuencias**: El prompt se reduce a una línea por procedimiento; los scripts son versionados, testables y mantenibles. Costo: mantenimiento del script.

**ADR-002: Helper bcrypt en Go dentro del repo**
- **Contexto**: Se necesita generar hashes bcrypt; `/tmp/genhash.go` es efímero.
- **Decisión**: Crear `scripts/genhash.go` (`package main`) que usa `golang.org/x/crypto/bcrypt` (ya en `go.mod`). `go run ./scripts/genhash.go <password>` imprime el hash por stdout.
- **Consecuencias**: `go build ./...` incluirá un binario helper más (inofensivo). Elimina la dependencia de archivos en `/tmp`.

**ADR-003: Parámetros ligados de sqlite3 para el INSERT**
- **Contexto**: El hash bcrypt contiene `$` que interpolar en comillas dobles rompe el valor.
- **Decisión**: `sqlite3 "$DB" "INSERT ... VALUES (?1, ?2) ON CONFLICT(email) DO UPDATE SET password_hash=?2;" "$EMAIL" "$HASH"`.
- **Consecuencias**: Sin riesgo de quoting ni inyección; el script es seguro para cualquier password.

## 4. Diseño Técnico

### 4.1 Diagrama de flujo — `create-test-user.sh`

```
[env: DB_PATH/EMAIL/PASSWORD/PORT/COOKIE_JAR]
        │
        v
[go run scripts/genhash.go $PASSWORD] ──► HASH (stdout)
        │
        v
[sqlite3 $DB "INSERT ... VALUES (?1,?2) ON CONFLICT(email) DO UPDATE SET password_hash=?2" $EMAIL $HASH]
        │
        v
[curl -s -c $COOKIE_JAR $BASE/api/login -d '{"email":..., "password":...}']
        │
        v
[echo: cookies listas + ejemplo curl -b $COOKIE_JAR]
```

### 4.2 Componentes

#### 4.2.1 `scripts/create-test-user.sh`
- **Responsabilidad**: Ciclo completo crear-usuario + login en un comando.
- **Interfaz**: Variables de entorno `DB_PATH`, `EMAIL`, `PASSWORD`, `PORT`, `COOKIE_JAR` (todas con default). Exit 0 con mensaje de éxito; exit ≠ 0 si falta `sqlite3`/`go`/DB o si el login falla.
- **Dependencias**: `go`, `sqlite3`, `curl`.
- **Ubicación**: `scripts/create-test-user.sh`

#### 4.2.2 `scripts/genhash.go`
- **Responsabilidad**: Imprimir hash bcrypt del password (arg 1, default `test1234`).
- **Interfaz**: `go run scripts/genhash.go [password]` → stdout hash.
- **Dependencias**: `golang.org/x/crypto/bcrypt` (ya en go.mod).
- **Ubicación**: `scripts/genhash.go`

#### 4.2.3 `scripts/check-server.sh`
- **Responsabilidad**: Verificar que el server responde `{"status":"ok"}` con reintentos; en fallo, mostrar el log.
- **Interfaz**: Variables `PORT`, `LOG_FILE`. Exit 0 si responde; exit 1 si no.
- **Dependencias**: `curl`.
- **Ubicación**: `scripts/check-server.sh`

### 4.3 Modelo de datos

```
Entidad: users (sin cambios de esquema)
- email: TEXT UNIQUE NOT NULL  (conflicto resuelto con ON CONFLICT DO UPDATE)
- password_hash: TEXT NOT NULL
```

### 4.4 APIs / Contratos

#### `POST /api/login` (consumido por el script)
**Request**:
```json
{ "email": "test@test.com", "password": "test1234" }
```
**Response 200**: `{ "email": "test@test.com" }` + cookie de sesión (guardada en `COOKIE_JAR`).

#### `GET /health` (consumido por `check-server.sh`)
**Response 200**: `{ "status": "ok", "timestamp": "..." }`

### 4.5 Dependencias

- **Internas**: `scripts/` (nuevos archivos), `AGENTS.md` (reglas), `docs/ssh-ihost-access.md` (referencia). Sin cambios en `cmd/`, `internal/`, `migrations/` ni frontend.
- **Externas**: `golang.org/x/crypto` (ya presente). `sqlite3` y `curl` (CLI, ya usados por el repo).

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Ejecutar `./scripts/create-test-user.sh` sobre una DB de prueba crea el usuario `test@test.com`, hace login y deja `/tmp/cookies.txt` con cookie válida.
- [x] CA-002: Ejecutar el script dos veces es idempotente (no duplica el usuario; actualiza el hash).
- [x] CA-003: Con `EMAIL`/`PASSWORD` personalizados, el login contra la API funciona con esas credenciales.
- [x] CA-004: El hash en DB no está corrompido (login exitoso lo demuestra; `$` del hash intacto).
- [x] CA-005: `./scripts/check-server.sh` con server arriba responde OK en ≤10s.
- [x] CA-006: `./scripts/check-server.sh` con server caído imprime las últimas líneas del log y sale con código 1.
- [x] CA-007: AGENTS.md referencia los scripts en una línea y ya no contiene los 4 pasos manuales ni el proceso de `/tmp/genhash.go`.
- [x] CA-008: `go test ./...` pasa completo (incluye el fix de `TestBillPayBill`).

### 5.2 No funcionales

- [x] CA-NF-001: Sin cambios en el binario servidor, la imagen Docker ni la DB de producción.
- [x] CA-NF-002: El script no imprime el hash ni el password en claro.

### 5.3 Testing

- **Unit tests**: No aplica (scripts bash; se valida por ejecución).
- **Integration tests**: Ejecutar el flujo completo contra una DB de prueba (`/tmp/test-app.db`) y el server local; verificar login con curl.
- **E2E tests**: No aplica.
- **Carga/Performance**: N/A.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | `scripts/genhash.go` + `scripts/create-test-user.sh` | 0.5 día | Ninguna |
| 2 | `scripts/check-server.sh` | 0.25 día | Ninguna |
| 3 | Actualizar AGENTS.md y docs/ssh-ihost-access.md | 0.25 día | Fase 1 y 2 |
| 4 | Pruebas locales + validación con el usuario | 0.5 día | Fase 3 |

### 6.2 Milestones

1. **MVP**: `create-test-user.sh` + AGENTS.md actualizado (P0).
2. **V1.0**: + `check-server.sh` (P1).

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| `sqlite3`/`go` no instalados en la dev machine | Media | Medio | Mensaje de error claro con exit ≠ 0; verificar presencia con `command -v` |
| Quoting del hash bcrypt | Baja (mitigado) | Alto | Parámetros ligados `?1`/`?2` de sqlite3 |
| Script apunte a la DB de producción por error | Baja | Alto | Validar que la DB exista; default local `data/app.db`; documentar riesgo en AGENTS.md |
| `go build ./...` falle por el helper | Baja | Medio | Helper es `package main` estándar; verificar compilación en CI/local |

## 8. Notas y Referencias

- AGENTS.md sección 0 (reglas críticas, fuente de la regla de tokens).
- `migrations/0001_create_users_and_settings.up.sql` (esquema `users`).
- `internal/api/handlers.go` (`Login` línea 145, `Health` línea 179).
- `go.mod` (`golang.org/x/crypto v0.26.0`).
- Precedente scripts: `scripts/run-local.sh`, `scripts/release.sh`.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-04 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-09-04 | p40la-ihost-team | REQ-013: fix test time-dependent `TestBillPayBill` (periodos derivados de la factura auto-generada) |
| 2026-09-04 | p40la-ihost-team | Implementación (commit `0800926`): `scripts/create-test-user.sh`, `scripts/check-server.sh`, `scripts/genhash.go`, AGENTS.md y ssh-ihost-access actualizados, fix `TestBillPayBill`. Merge a `main` y release. |