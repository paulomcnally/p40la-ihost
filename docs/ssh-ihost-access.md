# Guía de Acceso SSH a la DB del iHost

> **Última actualización**: 2026-09-04
> **Propósito**: Documentar, para agentes de IA y humanos, cómo acceder por SSH al volumen Docker de `p40la-ihost` en el SONOFF iHost para manipular la base de datos SQLite (`app.db`) de forma segura y verificable.

---

## 1. Contexto

- La app **p40la-ihost** corre como add-on Docker en un **SONOFF iHost** (eWeLink CUBE).
- La base de datos es **SQLite** y vive en un **volumen Docker llamado `p40la`**, montado en `/app/data` dentro del contenedor `p40la-ihost`.
- Archivos de la DB en el volumen: `app.db`, `app.db-wal`, `app.db-shm` (modo **WAL**).
- El iHost NO expone SSH de forma nativa: el puerto 22 del host está cerrado. **No existe un "Developer Mode" que active SSH.** El MaskROM mode (tocar 7 veces el device-id en la app eWeLink) sobreescribe el firmware y anula la garantía: **no usar**.
- La vía soportada por la comunidad es instalar el add-on Docker **`capiloky/sonoff-ssh`**, que levanta un servidor SSH dentro de un contenedor en el propio iHost.

---

## 2. Requisitos

1. El iHost debe estar accesible por red local como `ihost.local` (IP típica `192.168.1.191`).
2. El add-on **`capiloky/sonoff-ssh`** debe estar instalado y corriendo (imagen basada en `arm32v7/ubuntu`, compatible con el ARM64 del iHost por compatibilidad 32-bit).
3. El volumen `p40la` debe estar montado en el contenedor SSH en alguna ruta (la ruta exacta da igual; los archivos de la DB aparecen en la raíz del volumen montado).
4. La app `p40la-ihost` debe estar **DETENIDA** antes de cualquier escritura en la DB (ver sección 6).

---

## 3. Instalación del add-on SSH (una sola vez, vía UI del iHost)

En `http://ihost.local` → sección **Docker**:

1. Toca **+** → busca `capiloky/sonoff-ssh` → **Add**.
2. **Run** → configuración:
   - **Network**: `bridge`
   - **Port**: host `2222` → container `22`
   - **Volume**: selecciona el volumen `p40la` y proyéctalo a una ruta del contenedor (ej. `/app/data`).
3. **Run** para iniciarlo.

> ⚠️ El puerto host **no puede ser 80** (lo usa la UI del iHost) ni otro en uso. `2222` es la convención de este proyecto.

---

## 4. Credenciales y primer acceso

| Item | Valor |
|------|-------|
| Host | `ihost.local` |
| Puerto | `2222` |
| Usuario SSH | `sshuser` |
| Password SSH | `password` |
| Sudo | `sshuser` con password `password` (NOPASSWD en `/etc/sudoers`) |
| Usuario/Password del add-on filebrowser (si se usa) | `admin` / `admin` |

Comprobación de puerto abierto (desde la máquina de desarrollo):

```bash
timeout 2 bash -c "echo > /dev/tcp/ihost.local/2222" && echo "2222 OPEN"
```

---

## 5. Ejecutar comandos remotos (sin sshpass)

En la máquina de desarrollo **no hay `sshpass`** y el SSH pide password interactivamente. Usar **Python `paramiko`** (disponible):

```python
import paramiko
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect('ihost.local', port=2222, username='sshuser', password='password',
          timeout=10, allow_agent=False, look_for_keys=False)
_, out, err = c.exec_command('echo "password" | sudo -S id')
print(out.read().decode())
c.close()
```

- El `id` debe devolver `uid=0(root)`.
- `sudo` funciona con `echo "password" | sudo -S <cmd>` o `sudo -n` (las credenciales se cachean ~15 min).

### Primer acceso: instalar `sqlite3`

El contenedor SSH no trae `sqlite3`. Instalarlo una vez:

```bash
echo "password" | sudo -S apt-get update -qq
echo "password" | sudo -S apt-get install -y -qq sqlite3
```

### Subir archivos al contenedor

**SFTP NO está disponible** (subsistema deshabilitado). Para subir un archivo, usar stdin de `exec_command`:

```python
sql = open('import.sql','rb').read()
stdin, stdout, stderr = c.exec_command('cat > /tmp/import.sql && wc -l /tmp/import.sql', timeout=60)
stdin.write(sql)
stdin.channel.shutdown_write()
print(stdout.read().decode())
```

> No usar `base64` en la línea de comandos para archivos grandes: falla con `Argument list too long`.

---

## 6. Mecanismos de comprobación

### 6.1 El server de la app responde

```bash
curl -s http://ihost.local:8088/health
# → {"status":"ok","timestamp":"..."}
```

### 6.2 Acceso SSH + sudo funcionan

```python
# id == uid=0(root) tras sudo
```

### 6.3 La DB es legible y está íntegra

```bash
sudo sqlite3 /app/data/app.db "PRAGMA integrity_check;"   # → ok
sudo sqlite3 /app/data/app.db ".tables"                    # lista tablas
```

### 6.4 Verificación post-cambio

Siempre re-verificar con consultas de conteo (ej. `SELECT status, COUNT(*) FROM debt_bills GROUP BY status;`) y `PRAGMA integrity_check;` después de cualquier escritura.

---

## 7. Reglas críticas de operación (NO VIOLAR)

1. **NUNCA escribir en la DB mientras el contenedor `p40la-ihost` está corriendo.** En modo WAL hay datos en `app.db-wal`; escribir con el server activo corrompe la DB. Orden: **detener contenedor → operar → arrancar contenedor**.
2. **Respaldar SIEMPRE antes de modificar**:
   - En el iHost: `sudo sqlite3 /app/data/app.db ".backup '/app/data/backups/<fecha>-<motivo>/app.db'"` y verificar con `PRAGMA integrity_check;`.
   - Local (opcional pero recomendado): descargar el `.db` con el patrón `cat` vía `exec_command` (no SFTP) a `/home/paulomcnally/p40la-db-backups/`.
3. **No modificar el esquema SQLite.** La tabla `debts`/`debt_bills` se respeta tal cual; solo se insertan/actualizan datos (política del usuario). Ver `docs/specs/SPEC-054-modulo-deudas-calendario.md` para el modelo.
4. **Detener la escritura atómica**: envolver cambios multi-tabla en `BEGIN; ... COMMIT;` dentro de un solo `sqlite3 ".read archivo.sql"`.
5. **Usar `killall`, nunca `pkill`,** en procesos del iHost.
6. **Crear usuarios de prueba en la DB cuando se necesite autenticar** a endpoints protegidos (ver AGENTS.md §0). El hash bcrypt se genera en local con `go run ./scripts/genhash.go <password>`; el INSERT se aplica vía SSH a `/app/data/app.db` con `.parameter set` (el script `scripts/create-test-user.sh` es solo para la DB local).

---

## 8. Mapeo de datos (referencia Postgres → SQLite)

Contexto: la app `p4ola` (Postgres) no está en producción; los datos de deudas se importaron desde el backup `p4ola_20260726_030001.sql.gz`. La tabla SQLite `debts` usa `institution_id` (de `institutions`), `currency_id` (de `currencies`, `1=NIO`, `2=USD`) y estados `activa|inactiva|finalizada`. Las cuotas viven en `debt_bills` con `status pending|paid` y `UNIQUE(debt_id, installment_number)`.

---

## 9. Flujo típico de operación para un agente de IA

1. Verificar `curl http://ihost.local:8088/health` → `ok`.
2. Conectar por paramiko y verificar `sudo id` → root.
3. Verificar `sqlite3` instalado (instalar si falta).
4. **Pedir al usuario que detenga el contenedor `p40la-ihost`** (si la operación escribe).
5. **Respaldar** la DB (iHost + local).
6. Generar el SQL, probarlo en local sobre una copia del backup, y ejecutarlo en producción dentro de `BEGIN/COMMIT`.
7. Verificar conteos + `PRAGMA integrity_check;`.
8. Pedir al usuario que arranque el contenedor y confirme en la UI.
9. Reportar al usuario el resumen y ubicación de los backups.

---

## 10. Troubleshooting

| Problema | Causa / Solución |
|----------|------------------|
| `Connection refused` en 2222 | El add-on SSH no está corriendo. Revisar en UI de Docker del iHost. |
| `Permission denied` al SSH | Credenciales `sshuser`/`password` cambiadas o mal configuradas. |
| `sqlite3: command not found` | Instalar: `sudo apt-get install -y sqlite3` (una sola vez). |
| SFTP falla (`Channel closed`) | Esperado: SFTP no está habilitado. Usar `cat >` vía stdin. |
| `app.db-wal` creciendo / datos no visibles | La app está corriendo; detener el contenedor antes de operar. |
| `Argument list too long` | No pasar archivos grandes por argumentos; usar stdin. |