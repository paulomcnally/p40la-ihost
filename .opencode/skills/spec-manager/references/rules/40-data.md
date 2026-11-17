# 40 — Datos y migraciones

> **Dominio**: base de datos, migraciones, copias de producción y operación sobre datos reales.
> **Cargar cuando**: la spec toca DB, migraciones, importaciones, matching o reportes.

---

## DATA-DB — Configuración de la base

- Motor: `{{DATABASE}}`.
- Convenciones de conexión y persistencia: `{{DB_CONFIG}}`.
- Sin ORM si el proyecto lo prohíbe; SQL explícito.

---

## DATA-MIGRATIONS — Migraciones

- Naming: `{{MIGRATION_NAMING}}`.
- Toda migración debe tener reversión o justificar por qué no.
- Se prueban sobre una copia de producción antes de aplicarse.

---

## DATA-NO-DROP — La base del usuario es producción

- **NUNCA** borrar ni reemplazar la base de datos del usuario.
- Para pruebas limpias, usar una base temporal (`{{TEST_DB_PATH}}`).
- La base local no es representativa para validar datos reales: solo sirve para tests con fixtures sintéticos.

---

## DATA-PRODCOPY — Validar contra copia de producción

- Toda prueba cuyo resultado dependa del contenido de la DB se corre contra una **copia de producción**.
- La copia se genera con el mecanismo del proyecto (`{{PROD_BACKUP_COMMAND}}`) y se descarga a `/tmp`.
- La copia vive **solo en `/tmp`** y se borra al terminar.
- **Nunca** se escribe en producción durante una validación.

---

## DATA-PROD-WRITE — Escritura sobre datos de producción

Si una petición implica modificar datos de producción, el agente DEBE:

1. Pedir al usuario detener el servicio antes de escribir.
2. Respaldar y verificar integridad.
3. Probar el SQL sobre una copia local.
4. Ejecutar en transacción (`BEGIN/COMMIT`).
5. Verificar conteos e integridad y pedir al usuario reiniciar.

Ajustar los comandos concretos en `{{PROD_DB_RUNBOOK}}`.

---

## DATA-SCHEMA — Esquema protegido

- Tablas/entidades que **no** se pueden alterar: `{{PROTECTED_SCHEMA}}`.
- Si no hay ninguna, declararlo explícitamente.
