# Importación del export de automation y despliegue único

> **Spec**: SPEC-112  
> **Aplica a**: migración de `p40la-ihost-automation` → `p40la-ihost`  
> **Datos**: el export JSON contiene credenciales, webhooks, facturas y logs. **Nunca se commitea ni se comparte.**

Este runbook describe cómo importar el export JSON de `p40la-ihost-automation` en la DB de producción de `p40la-ihost` y cómo desplegar la imagen única. Todos los ejemplos usan rutas y datos sintéticos.

---

## 1. Formato del export

El archivo tiene este sobre (los valores son ilustrativos):

```json
{
  "export": {
    "format_version": 1,
    "app": "p40la-ihost-automation",
    "tables": {
      "settings": [],
      "apps": [],
      "plugins": [],
      "accounts": [],
      "bills": [],
      "webhook_logs": []
    }
  }
}
```

El importador valida `app`, `format_version` y las tablas. Las columnas `credentials`, `raw` y `payload` pueden venir como objeto/array JSON; el importador las serializa a texto.

Tablas destino (creadas por la migración `0039_merge_automation`):

| Origen | Destino |
|---|---|
| `apps` | `institutions` (match exacto, normalizado o `--institution-map`) |
| `plugins` | `automation_plugins` |
| `accounts` | `automation_accounts` |
| `bills` | `automation_bills` |
| `webhook_logs` | `automation_webhook_logs` |
| `settings` | `system_settings` (solo `webhook_enabled` y `language`) |
| `users` | **no se importa** (p40la tiene su propia autenticación) |

---

## 2. Precondiciones

- Contenedor `p40la-ihost` detenido (modo WAL: nunca escribir con el server corriendo).
- Backup verificado de `/app/data/app.db`.
- Export JSON descargado en el host de trabajo, en `/tmp`, con permisos `0600`.
- Mismo binario/versión de `import-automation` que la release a desplegar.
- Espacio libre suficiente para la copia de la DB y la SQLite temporal.

> **Regla**: la DB local de desarrollo no sirve para validar esta importación. Se trabaja siempre sobre una copia de producción.

---

## 3. Procedimiento recomendado (importar sobre una copia y subir la DB)

### 3.1 Detener el contenedor y respaldar

```bash
# En el iHost (SSH add-on, puerto 2222)
sudo sqlite3 /app/data/app.db ".backup '/app/data/backups/$(date +%Y%m%d-%H%M)-pre-automation/app.db'"
sudo sqlite3 /app/data/backups/.../app.db "PRAGMA integrity_check;"
```

Descargar la copia a la máquina de trabajo (`/tmp/.../app.db`). No usar copia plana con el `-wal` activo.

### 3.2 Construir el mapa de instituciones

Listar las apps del export (no expone credenciales):

```bash
jq -r '.export.tables.apps[].name' /tmp/p40la-export-XXXX.json
```

Comparar con las instituciones existentes:

```bash
sqlite3 /tmp/import/app.db "SELECT name FROM institutions ORDER BY name;"
```

Si un nombre de app no coincide exactamente o normalizado con una institución, crear un mapa **con el nombre exacto de la app**:

```json
{
  "Proveedor Demo": "Proveedor Demo Existente"
}
```

Guardarlo en `/tmp/import/institution-map.json` (no en el repo). Si no hay discrepancias, omitir `--institution-map`.

### 3.3 Importar sobre la copia

```bash
go run ./cmd/import-automation \
  --from-json /tmp/p40la-export-XXXX.json \
  --data-dir /tmp/import \
  --migrations ./migrations \
  --institution-map /tmp/import/institution-map.json \
  --verify
```

Para validar sin escribir:

```bash
go run ./cmd/import-automation \
  --from-json /tmp/p40la-export-XXXX.json \
  --data-dir /tmp/import \
  --migrations ./migrations \
  --dry-run
```

Salida esperada (ejemplo con conteos reales del export validado en SPEC-112):

```
accounts: leídas=6 insertadas=6 omitidas=0
bills: leídas=80 insertadas=80 omitidas=0
plugins: leídas=5 insertadas=5 omitidas=0
webhook_logs: leídas=838 insertadas=838 omitidas=0
instituciones: apps=5 reutilizadas=4 creadas=0 mapeadas=1
verify: automation_accounts=6 ok
verify: automation_bills=80 ok
verify: automation_webhook_logs=838 ok
```

Si `creadas` es mayor que 0, revisar los nombres: probablemente falte una entrada en el mapa.

### 3.4 Verificar la copia

```bash
sqlite3 /tmp/import/app.db "PRAGMA integrity_check; PRAGMA foreign_key_check;"
sqlite3 /tmp/import/app.db "SELECT COUNT(*) FROM automation_accounts;"
sqlite3 /tmp/import/app.db "SELECT COUNT(*) FROM automation_bills;"
sqlite3 /tmp/import/app.db "SELECT COUNT(*) FROM automation_webhook_logs;"
sqlite3 /tmp/import/app.db "SELECT COUNT(*) FROM automation_plugins;"
```

Revisar también que no queden servicios huérfanos:

```sql
SELECT DISTINCT automation_account_id
FROM services
WHERE automation_account_id IS NOT NULL
  AND automation_account_id NOT IN (SELECT id FROM automation_accounts);
```

### 3.5 Subir la DB importada

Con el contenedor **detenido**:

1. Subir `/tmp/import/app.db` a `/app/data/app.db` (SSH, stdin; no hay SFTP).
2. Eliminar `/app/data/app.db-wal` y `/app/data/app.db-shm` si existen.
3. Verificar permisos y `integrity_check` en el iHost.
4. Arrancar el contenedor y comprobar `GET /health`.

> Alternativa: ejecutar el importador en un contenedor one-off con la DB y el JSON montados. Si se elige esa vía, usar una imagen que incluya el binario y montar `/app/data` en solo lectura salvo la DB.

### 3.6 Verificar la aplicación

```bash
curl -s http://ihost.local:8088/health
```

Luego, en la UI:
- Login.
- `/automation`: 6 cuentas (o el conteo esperado).
- Detalle de una cuenta: facturas y logs.
- `/services`: editar un servicio y ver el selector de cuentas.
- Ejecutar un sync manual solo si el proveedor está disponible.

---

## 4. Fallback: inserts SQL directos

Solo si el importador no puede ejecutarse. El importador es la vía recomendada porque resuelve instituciones, stubs de plugins e idempotencia.

Esqueleto (sin datos reales; reemplazar placeholders):

```sql
BEGIN;

INSERT OR IGNORE INTO automation_plugins (id,name,version,source,description,status)
VALUES (1,'demo.provider','1.0.0','demo','Plugin demo','active');

-- institutions: resolver por nombre exacto/normalizado antes de insertar.
INSERT OR IGNORE INTO automation_accounts
  (id,institution_id,identifier,label,credentials,plugin_name,webhook_url,
   schedule_time,schedule_enabled,last_run_at,schedule_frequency,schedule_interval,schedule_days)
VALUES
  (1,1,'demo-account-001','Cuenta Demo','{"user":"demo"}','demo.provider',
   'http://demo.local/webhooks/00000000-0000-4000-8000-000000000000',
   '07:00',1,NULL,'daily',NULL,NULL);

INSERT OR IGNORE INTO automation_bills
  (id,account_id,plugin_name,plugin_version,source,status,error,raw,fetched_at,created_at)
VALUES
  (1,1,'demo.provider','1.0.0','demo','ok',NULL,'{"numFactura":"F000000000000001"}',
   CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);

COMMIT;
```

Reglas:
- Ensayar el script sobre una copia y correr `integrity_check` + `foreign_key_check`.
- Ejecutar en una sola transacción.
- No tocar el esquema de `debts`/`debt_bills`.

---

## 5. Rollback

1. Detener `p40la-ihost`.
2. Restaurar el backup previo:
   ```bash
   sudo cp /app/data/backups/<fecha>-pre-automation/app.db /app/data/app.db
   sudo rm -f /app/data/app.db-wal /app/data/app.db-shm
   ```
3. Verificar `integrity_check`.
4. Volver a la imagen anterior (`IMAGE_TAG` previo) y arrancar.
5. Confirmar `/health` y login.

---

## 6. Retiro de `p40la-ihost-automation`

Solo después de validar la app unificada:

1. Detener el contenedor `p40la-ihost-automation`.
2. Conservar su volumen/backup como respaldo (no borrar todavía).
3. Quitar el contenedor del compose/auto-start.
4. Archivar el repositorio viejo (README con puntero a `p40la-ihost`).
5. Pasado el período de gracia, eliminar el volumen viejo.

---

## 7. Limpieza y seguridad

- Borrar `/tmp/p40la-export-*.json`, `/tmp/import/` y el mapa al terminar.
- No pegar filas, credenciales, webhooks ni facturas en issues/PRs/chat.
- Si el JSON se copió a un equipo compartido, rotar las credenciales de los proveedores y la API key de webhooks.
- Verificar que `.gh_token`, `.env` y los exports estén fuera de git (`git status`).

---

## 8. Build y despliegue de la imagen única

El módulo privado de plugins requiere un token con lectura:

```bash
gh auth token > .gh_token   # no commitear
docker buildx build \
  --platform linux/amd64,linux/arm/v7,linux/arm64 \
  --secret id=gh_token,src=.gh_token \
  -t paulomcnally/p40la-ihost:<version> \
  --push .
```

Verificar los 3 manifests:

```bash
docker buildx imagetools inspect paulomcnally/p40la-ihost:<version>
```

En producción, el compose usa el volumen externo `p40la`:

```yaml
volumes:
  - p40la:/app/data
```

No cambiar el nombre del volumen: es donde vive la DB existente.

---

## 9. Checklist final

- [ ] Contenedor detenido y backup con `integrity_check = ok`.
- [ ] Export JSON en `/tmp` con permisos `0600`.
- [ ] `--dry-run` sin errores.
- [ ] Import real con `--verify` y conteos esperados.
- [ ] `creadas = 0` o justificado.
- [ ] `foreign_key_check` limpio.
- [ ] DB subida con el contenedor detenido; `-wal`/`-shm` eliminados.
- [ ] `/health` OK y login OK.
- [ ] `/automation`, facturas, logs y selector de servicios verificados.
- [ ] Contenedor viejo detenido y volumen conservado.
- [ ] `/tmp` limpio; secretos rotados si corresponde.
