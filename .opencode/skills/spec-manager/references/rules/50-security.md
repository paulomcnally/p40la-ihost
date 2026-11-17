# 50 — Seguridad, privacidad y datos sintéticos

> **Dominio**: PII, secretos, datos sintéticos, higiene de commits e incidentes.
> **Cargar siempre**. No admite excepciones.

---

## SEC-NO-PII — Prohibido commitear datos privados

Nunca se commitean:

- **PII**: nombres y apellidos, teléfonos, emails personales, direcciones, documentos de identidad, números de cuenta/contrato/factura reales, archivos reales de clientes, IPs/MACs domésticas.
- **Datos financieros**: montos, deudas, salarios, movimientos o extractos reales.
- **Secretos**: contraseñas, hashes, tokens, API keys, cookies, session secrets, credenciales, webhook keys y claves privadas.
- **Volcados**: DBs, `-wal`, `-shm`, backups, exports y capturas con datos reales.

Aplica a código, tests, fixtures, specs, docs, README, issues, PRs, commits, logs y capturas.

---

## SEC-SYNTHETIC — Datos sintéticos obligatorios

| Tipo | Ejemplo permitido |
|---|---|
| Identificador | `demo-account-001` |
| Email | `demo@example.com` |
| Teléfono | `+1 555 0100` |
| Entidad | `Proveedor Demo` |
| Documento | `F000000000000001` |
| Token | `demo-api-key`, `demo-token` |
| UUID | `00000000-0000-4000-8000-000000000000` |

- Prohibido usar datos reales «porque son de prueba».
- Si hay que reproducir un caso real, sanitizarlo primero y documentarlo.
- No pegar dumps ni filas reales: describir con conteos y nombres de columnas.

---

## SEC-COMMIT — Antes de commitear

1. Revisar `git diff --cached`.
2. Escanear el diff buscando PII/secretos (emails que no sean `example.com`, claves largas, `BEGIN ... PRIVATE KEY`, tokens, `password`).
3. Verificar que no se agreguen DBs, `-wal`, `-shm`, `backups/`, PDFs ni exports.
4. Si hay duda, no commitear: sanitizar primero.

---

## SEC-INCIDENT — Protocolo de incidente

1. Rotar el secreto de inmediato.
2. Eliminarlo del árbol **y de la historia**.
3. Documentar el incidente y avisar al usuario.

---

## SEC-LOGS — Logs y capturas

- Logs sanitizados: sin tokens, IDs, IPs ni datos de cuenta.
- Las capturas no deben mostrar datos reales.
