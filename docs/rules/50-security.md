# 50 — Seguridad, privacidad y datos sintéticos

> **Dominio**: PII, secretos, datos sintéticos, higiene de commits y protocolo de incidentes.
> **Cargar siempre**. Estas reglas no son negociables y no admiten excepciones.

---

## SEC-NO-PII — Prohibido commitear datos privados

Nunca se commitean:

- **Datos personales (PII)**: nombres y apellidos, teléfonos, emails personales, direcciones, cédulas/DNI, RUC personal, números de cuenta/medidor/contrato, números de factura reales, facturas o PDFs reales, chat IDs de Telegram, IPs/MACs domésticas.
- **Datos financieros**: montos, deudas, salarios, movimientos o cualquier extracto real.
- **Secretos**: contraseñas, hashes de contraseña, tokens (Telegram/APIs), API keys, cookies, `SESSION_SECRET`, credenciales SSH, webhook UUIDs/keys y claves privadas.
- **Volcados**: DBs (`*.db`, `-wal`, `-shm`), backups, exports JSON/CSV y capturas con datos reales.

Aplica a **código, tests, fixtures, specs, docs, README, issues, PRs, mensajes de commit, logs y capturas**.

---

## SEC-SYNTHETIC — Datos sintéticos obligatorios

En tests, ejemplos y specs usar siempre datos ficticios:

| Tipo | Ejemplo permitido |
|---|---|
| Identificador de cuenta | `demo-account-001` |
| Email | `demo@example.com` |
| Teléfono | `+1 555 0100` |
| Institución/proveedor | `Proveedor Demo` |
| Factura | `F000000000000001` |
| API key/token | `demo-api-key`, `demo-token` |
| UUID | `00000000-0000-4000-8000-000000000000` |

- Está prohibido usar datos reales «porque son de prueba».
- Si hace falta reproducir un caso real, se **sanitiza primero** (nombres, números, direcciones, montos) y se documenta que es sintético.
- No se pegan dumps, filas ni valores reales en specs, issues, PRs ni mensajes de chat/commit. Los resultados se describen con **conteos y nombres de columnas**.

---

## SEC-PRODCOPY — Copias de producción

- Las copias de la DB de producción viven **solo en `/tmp`**, nunca en el repo.
- Se borran al terminar la validación.
- Nunca se commitean DBs, `-wal`, `-shm`, backups, PDFs ni exports.

Detalle operativo en [40-data.md](40-data.md) (`DATA-PRODCOPY`, `DATA-PROD-WRITE`).

---

## SEC-COMMIT — Antes de commitear

1. Revisar `git diff --cached`.
2. Escanear el diff buscando PII/secretos: teléfonos, emails que no sean `example.com`, claves hex largas, `BEGIN ... PRIVATE KEY`, tokens de Telegram, `password`.
3. Verificar que no se agreguen `*.db`, `*.db-wal`, `*.db-shm`, `backups/`, `*.pdf` ni exports.
4. Si hay duda, **no commitear**: sanitizar primero.

**Guardrail**: `diff` (regla `SEC-COMMIT`) → revisión manual de `git diff --cached`.

---

## SEC-INCIDENT — Protocolo de incidente

Si se commiteó un dato privado o un secreto:

1. **Rotar** el secreto de inmediato (API key, token, contraseña).
2. Eliminarlo del árbol **y de la historia** (no alcanza con un commit posterior).
3. Documentar el incidente y avisar al usuario.

---

## SEC-LOGS — Logs y capturas

- Los logs pegados en issues o specs deben estar sanitizados (sin tokens, IDs de chat, IPs ni datos de cuenta).
- Las capturas de pantalla no deben mostrar datos reales.

**Precedente**: SPEC-106/107 incluyeron montos y nombres de deudas reales en la spec (violación de esta regla; se documenta como antecedente negativo).
