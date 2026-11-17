# 40 — Datos, SQLite y migraciones

> **Dominio**: base de datos, migraciones, copias de producción y operación sobre datos reales.
> **Cargar cuando**: la spec toca DB, migraciones, importaciones, matching de datos, reportes o cualquier comportamiento que dependa del contenido real.

---

## DATA-SQLITE — Configuración de SQLite

- `journal_mode=WAL` para mejor concurrencia.
- Sin ORM: SQL crudo con `database/sql` y helpers livianos propios.
- La base persiste en el volumen montado (`/app/data` en el contenedor), nunca dentro de la imagen.

---

## DATA-MIGRATIONS — Migraciones

- Naming obligatorio: `migrations/NNNN_descripcion.up.sql` y `.down.sql`.
- Toda migración debe tener `down` reversible o justificar por qué no.
- Las migraciones se prueban sobre una **copia de producción** antes de aplicarse.

---

## DATA-NO-DROP — La base del usuario es producción

- **NUNCA** ejecutar `rm -f data/app.db` ni cualquier comando que elimine o reemplace la base del usuario.
- La DB local del usuario es producción.
- Para pruebas limpias usar `/tmp/test-app.db`.
- La DB local **no** es representativa para validar datos reales: solo sirve para tests unitarios con fixtures sintéticos (precedente SPEC-109: `Tigo` apareció como institución nueva porque la DB local no la tenía).

---

## DATA-PRODCOPY — Validar contra copia de producción

- Toda prueba cuyo resultado dependa del contenido de la DB se corre contra una **copia de la DB de producción**, nunca contra `data/app.db` local.
- La copia se genera con `.backup` y se descarga a `/tmp`:
  ```bash
  sqlite3 /app/data/app.db ".backup '/tmp/<motivo>.db'"
  ```
- La copia vive **solo en `/tmp`** y se borra al terminar.
- **Nunca** se escribe en producción durante una validación.

---

## DATA-PROD-WRITE — Operación de escritura sobre la DB de producción

Si una petición implica modificar la DB del volumen, el agente DEBE:

1. Pedir al usuario **detener el contenedor** `p40la-ihost` antes de cualquier escritura (WAL activo + server corriendo = riesgo de corrupción).
2. **Respaldar siempre**:
   ```bash
   sudo sqlite3 /app/data/app.db ".backup '/app/data/backups/<fecha>-<motivo>/app.db'"
   ```
   Verificar con `PRAGMA integrity_check;` y descargar copia local.
3. Probar el SQL en local sobre una copia del backup.
4. Ejecutar cambios multi-tabla envueltos en `BEGIN/COMMIT`:
   ```bash
   sudo sqlite3 /app/data/app.db ".read /tmp/script.sql"
   ```
5. Verificar conteos e integridad y pedir al usuario arrancar el contenedor.

El acceso al iHost se documenta en `docs/ssh-ihost-access.md`.

---

## DATA-PROTECTED-SCHEMA — Esquema protegido

- **No modificar el esquema** de `debts` / `debt_bills` (política del usuario): solo insertar o actualizar datos.
- Modelo de deudas documentado en SPEC-054.

---

## DATA-SSH — Acceso al iHost

- El iHost no tiene SSH nativo: se accede con el add-on `capiloky/sonoff-ssh` en el puerto `2222`.
- Sin `sshpass` en la máquina de desarrollo: usar Python `paramiko`.
- No usar `base64` en argumentos (falla con «Argument list too long»); subir archivos por stdin de `exec_command`.
- SFTP no disponible.
- `sqlite3` no viene instalado en el contenedor SSH: instalar una vez con `sudo apt-get install -y sqlite3`.
- Verificar el server con `curl -s http://ihost.local:8088/health` → `{"status":"ok"}`.

**Guardrail**: `DATA-PRODCOPY` se declara en la spec cuando el resultado depende de datos.
