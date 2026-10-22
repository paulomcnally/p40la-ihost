---
title: "Unificación automation (1/4): migración de esquema y datos a tablas automation_* en app.db"
id: "SPEC-109"
status: "draft"
author: "opencode"
created: "2026-10-21"
updated: "2026-10-21"
github_issue: 112
---

# Unificación automation (1/4): migración de esquema y datos a tablas automation_* en app.db

**ID**: SPEC-109  
**Estado**: draft  
**Autor**: opencode  
**Creado**: 2026-10-21  
**Actualizado**: 2026-10-21

---

## 1. Resumen Ejecutivo

Primera de cuatro specs de la unificación definida en SPEC-108. Esta spec **solo crea el esquema y migra los datos** de `p40la-ihost-automation` a la DB de `p40la-ihost`, sin tocar rutas, UI ni el scheduler. Al terminar, `app.db` contiene las tablas `automation_*` con las apps, cuentas, plugins, facturas y logs de webhook, y `services.automation_account_id` queda listo para validarse contra `automation_accounts`.

El prototipo de SPEC-108 ya se ejecutó con éxito sobre copias: 5 apps, 5 cuentas, 5 plugins, 28 facturas y 100 logs migrados, `integrity_check` y `foreign_key_check` limpios. El hallazgo principal es que **el import debe hacerse con un comando offline** que lea una copia de la DB de automation (no con `ATTACH` hardcodeado en una migración) y que el backup debe usar `sqlite3 .backup`, porque copiar solo `app.db` pierde el contenido del `-wal` (en la prueba se perdieron 6 facturas y 52 logs).

**iHost**: solo agrega tablas e índices a la DB existente; no incorpora dependencias ni procesos nuevos. El import corre una vez, offline, y puede repetirse de forma idempotente.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Migración `0039_merge_automation.up.sql` que crea `automation_apps`, `automation_plugins`, `automation_accounts`, `automation_bills` y `automation_webhook_logs`, con las columnas, constraints y FKs internas (`automation_*` entre sí) del modelo de SPEC-108.
2. **REQ-002**: Índice `idx_services_automation_account_id` sobre `services(automation_account_id)` para el selector y los joins.
3. **REQ-003**: Comando offline `cmd/import-automation` que importa desde una copia de la DB de automation (`--from /ruta/automation.db`) a la DB destino (`DATA_DIR/app.db`), preservando IDs de apps, cuentas, plugins, facturas y logs.
4. **REQ-004**: El import no debe pisar `system_settings.webhook_api_key` de p40la; solo migra `webhook_enabled` y `language` si no existen o actualiza `webhook_enabled`.
5. **REQ-005**: El import debe ser **idempotente**: correrse dos veces no duplica filas ni cambia IDs (usar `INSERT ... ON CONFLICT`/verificación de existencia).
6. **REQ-006**: El comando debe soportar `--dry-run` (valida y reporta sin escribir) y mostrar un resumen: filas leídas/insertadas/omitidas por tabla.
7. **REQ-007**: Al finalizar, ejecutar `PRAGMA foreign_key_check` y `PRAGMA integrity_check` y fallar con código distinto de 0 si hay errores.
8. **REQ-008**: Documentar el procedimiento de backup y rollback: `sqlite3 app.db ".backup ..."` de ambas DB con los contenedores detenidos, restauración y verificación. Nunca usar copia plana de `app.db` sin `-wal`.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-009**: Validar en el import que `services.automation_account_id` (si no es `NULL`) exista en `automation_accounts`; reportar los IDs huérfanos (no borrarlos ni corregirlos automáticamente).
2. **REQ-010**: Preservar `credentials`, `webhook_url`, `schedule_*` y `last_run_at` de las cuentas, y `raw`/`http_status` de facturas y logs.
3. **REQ-011**: No migrar `users` ni `auth_cookie_secret` de automation: la autenticación pasa a ser la de p40la (SPEC-110).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-012**: Comando `--verify` que compara conteos origen/destino por tabla y devuelve un diff legible.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: el import debe tardar segundos con los volúmenes actuales (decenas de facturas/cientos de logs) y no bloquear el arranque normal del server.
- **Seguridad**: no loguear `credentials`, `webhook_api_key` ni tokens; el comando corre offline con la app detenida.
- **Almacenamiento**: el crecimiento de la DB es acotado (tablas nuevas + índice); los logs de webhook no crecen sin control porque la rotación se define en SPEC-110.
- **Disponibilidad**: el comando es de una sola ejecución; un fallo no deja la DB a medias (transacción única por import).
- **iHost**: sin dependencias nuevas; SQLite `modernc.org/sqlite`; binario estático.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- SPEC-108: evaluación, colisiones (`bills`, `users`, `settings`) y prototipo de migración validado.
- Automación: tablas `apps`, `accounts`, `plugins`, `bills`, `webhook_logs`, `settings`, `users`; migraciones `0001`–`0006`; SQLite `modernc`.
- p40la: `app.db` con `schema_migrations` (`0001`–`0038`), migraciones propias y `system_settings` KV.
- El prototipo con `ATTACH` funcionó, pero **no es apto para producción**: la ruta de la DB origen no existe dentro del contenedor de p40la y `ATTACH` en una migración acopla una ruta externa.
- El `-wal` de automation (881 KB en la copia local) contenía 6 facturas y 52 logs; una copia plana los perdió.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| `ATTACH` dentro de la migración `0039` | Simple, un solo archivo SQL | Ruta externa hardcodeada; falla si la DB origen no está montada; no idempotente | ❌ Rechazada para producción |
| Comando offline `cmd/import-automation` (Go) | Testeable, idempotente, `--dry-run`, resumen y validaciones; no depende de rutas dentro de migraciones | Requiere un paso operativo extra | ✅ Seleccionada |
| Volcar datos a CSV y cargarlos | Independiente del driver | Más pasos manuales, pierde tipos/escapes JSON, propenso a error | ❌ Rechazada |
| Copiar el archivo de DB de automation como DB separada | Cero migración | Mantiene dos DB y no cumple el objetivo de SPEC-108 | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: El import es un comando offline, no una migración SQL con `ATTACH`.
- **Contexto**: la DB origen vive en otro volumen y solo se copia al volumen de p40la durante la ventana de mantenimiento.
- **Decisión**: `cmd/import-automation --from /app/data/automation-import.db` abre ambas DB con `database/sql`, corre en una transacción y verifica al final.
- **Consecuencias**: Positivo: testeable e idempotente. Negativo: el operador debe copiar la DB origen al volumen y ejecutar el comando una vez.

**ADR-002**: Los IDs de automation se preservan.
- **Contexto**: `services.automation_account_id` ya apunta a IDs de cuentas.
- **Decisión**: insertar con el `id` explícito y no reasignar.
- **Consecuencias**: los vínculos existentes siguen válidos; si un ID quedó huérfano, se reporta (REQ-009).

**ADR-003**: `webhook_api_key` de p40la no se toca.
- **Contexto**: automation usa su key para autenticar el POST al webhook de p40la; al unificar, la entrega pasa a ser en proceso y esa key deja de usarse para ese fin.
- **Decisión**: conservar la key de p40la y migrar solo `webhook_enabled`/`language`.
- **Consecuencias**: los clientes externos que ya usan la key de p40la siguen funcionando; no se rompe nada.

## 4. Diseño Técnico

### 4.1 Diagrama

```
[automation container] --backup .backup--> automation-import.db
                                              |
                                              v
                                 [cmd/import-automation]
                                              |
                                              v
                                   /app/data/app.db (p40la)
                                   ├── automation_* (nuevo)
                                   └── services.automation_account_id
```

### 4.2 Componentes

#### 4.2.1 Migración `0039_merge_automation.up.sql`
- **Responsabilidad**: crear tablas `automation_*` e índice.
- **Ubicación**: `migrations/0039_merge_automation.up.sql` + `.down.sql` (drop de tablas nuevas).
- **Dependencias**: ninguna.

#### 4.2.2 `cmd/import-automation`
- **Responsabilidad**: importar datos desde la copia de automation.
- **Interfaz**: `--from <archivo>`, `--data-dir <dir>` (default `DATA_DIR`), `--dry-run`, `--verify`.
- **Ubicación**: `cmd/import-automation/main.go` + tests.
- **Dependencias**: `internal/db`, `modernc.org/sqlite`.

### 4.3 Modelo de datos

Idéntico al de SPEC-108 §4.3: `automation_apps`, `automation_accounts`, `automation_plugins`, `automation_bills`, `automation_webhook_logs`; FKs internas con `ON DELETE CASCADE`; `automation_accounts.plugin_name` referencia `automation_plugins(name)`.

### 4.4 APIs / Contratos

Esta spec no expone endpoints nuevos. Solo el comando CLI:

```
import-automation --from /app/data/automation-import.db [--dry-run] [--verify]
```

Salida esperada:

```
apps: leídas=5 insertadas=5 omitidas=0
accounts: leídas=5 insertadas=5 omitidas=0
plugins: leídas=5 insertadas=5 omitidas=0
bills: leídas=28 insertadas=28 omitidas=0
webhook_logs: leídas=100 insertadas=100 omitidas=0
settings: webhook_enabled actualizado, language omitido (ya existe)
huérfanos: services.automation_account_id -> []
integrity_check: ok
foreign_key_check: ok
```

### 4.5 Procedimiento operativo (runbook)

1. Detener `p40la-ihost` y `p40la-ihost-automation`.
2. Backup: `sqlite3 /app/data/app.db ".backup '/app/data/backups/<fecha>-pre-merge/app.db'"` (p40la) y el equivalente para automation.
3. Copiar el backup de automation al volumen de p40la como `automation-import.db`.
4. Ejecutar `import-automation --from /app/data/automation-import.db --dry-run`.
5. Ejecutar sin `--dry-run`.
6. Verificar conteos e `integrity_check`; arrancar `p40la-ihost`.
7. Rollback: restaurar el backup de p40la y volver a arrancar la versión anterior.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: `0039` crea las 5 tablas `automation_*` y el índice, sin tocar tablas existentes.
- [ ] CA-002: El import preserva IDs y copia exactamente los conteos de apps, cuentas, plugins, facturas y logs.
- [ ] CA-003: Correr el import dos veces no duplica filas (idempotencia).
- [ ] CA-004: `webhook_api_key` de p40la queda intacta; `webhook_enabled`/`language` migrados.
- [ ] CA-005: `PRAGMA integrity_check` = `ok` y `foreign_key_check` sin filas.
- [ ] CA-006: Los `services.automation_account_id` huérfanos se reportan sin abortar el import.
- [ ] CA-007: `--dry-run` no escribe nada y reporta el mismo resumen.
- [ ] CA-008: El runbook de backup/rollback está probado sobre copias.

### 5.2 No funcionales

- [ ] CA-NF-001: El import no loguea credenciales ni claves.
- [ ] CA-NF-002: El comando termina en segundos con los volúmenes actuales y no agrega dependencias.
- [ ] CA-NF-003: La migración `.down.sql` revierte el esquema sin afectar tablas de p40la.

### 5.3 Testing

- **Unit tests**: import idempotente, preservación de IDs, no pisar `webhook_api_key`, detección de huérfanos.
- **Integration tests**: sobre copias reales de ambas DB (fixtures), `integrity_check` y `foreign_key_check`.
- **E2E**: runbook completo en el iHost con contenedores detenidos (ventana de mantenimiento).
- **Performance**: tiempo de import y tamaño final de `app.db`.

## 6. Plan de Implementación

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración `0039` + `.down.sql` | 0,5 día | SPEC-108 |
| 2 | `cmd/import-automation` con `--dry-run`/`--verify` e idempotencia | 1 día | Fase 1 |
| 3 | Tests con copias de las DB y verificación de conteos | 0,5 día | Fase 2 |
| 4 | Runbook y prueba en iHost (sin arrancar aún el backend unificado) | 0,5 día | Fase 3 |

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| Backup plano sin `-wal` | Media | Alto | Usar `.backup`; validado en el prototipo |
| IDs duplicados al reimportar | Media | Medio | `ON CONFLICT` + verificación de existencia + test de idempotencia |
| `credentials` con JSON inválido | Baja | Medio | Copiar `TEXT` tal cual; no parsear en el import |
| Huérfanos en `services.automation_account_id` | Media | Bajo | Reportar y resolver en la UI (SPEC-111) |
| Correr el import con la app arriba | Baja | Alto | Runbook exige contenedores detenidos |

## 8. Notas y Referencias

- SPEC-108: evaluación y estrategia multi-spec.
- SPEC-092: origen del vínculo `automation_account_id`.
- SPEC-110: backend unificado (consume estas tablas).
- SPEC-111: selector de cuentas en la UI.
- Prototipo validado en `/tmp/opencode/spec108-migration/0039_merge_automation.up.sql` (SPEC-108).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-10-21 | opencode | Creación inicial: esquema `automation_*` + comando de import offline, derivado de SPEC-108. |
