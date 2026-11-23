# 60 — Pruebas y verificación

> **Dominio**: pruebas locales, verificación del server, usuarios de prueba, flujo de validación con el usuario y checklist de calidad.
> **Cargar siempre antes de `pending_release`**.

---

## TEST-LOCAL — Pruebas locales obligatorias

- Todo cambio significativo se prueba **en local** (tests, build y ejecución básica) antes de commit o push.
- **No se commitea** código que no compile, que falle los tests o que no haya sido validado localmente.
- Si el resultado depende de datos reales, la prueba se corre contra una copia de la DB de producción (`DATA-PRODCOPY`), nunca contra `data/app.db` local.

---

## TEST-SERVER — Verificar el server antes de avisar

- Antes de informar al usuario que puede acceder, ejecutar `./scripts/check-server.sh`.
- El script consulta `GET /health` esperando `{"status":"ok"}`, reintenta y, si falla, muestra el log y sale con error.
- Solo cuando responde se le informa al usuario. Si no responde: revisar el log y reiniciar.

---

## TEST-KILLALL — Matar procesos con `killall`

- Usar **SIEMPRE** `killall` para matar procesos, **NUNCA** `pkill`.
- `pkill` falla silenciosamente en este entorno.
- Comando correcto: `killall p40la-ihost`.

---

## TEST-TESTUSER — Usuarios de prueba para autenticar

- Para probar endpoints protegidos, crear el usuario directamente en SQLite con `./scripts/create-test-user.sh`.
- Crea/actualiza el usuario (default `test@test.com` / `test1234`), hace login y deja las cookies en `/tmp/cookies.txt`.
- Personalizable con `EMAIL`, `PASSWORD`, `DB_PATH` y `PORT`.
- Uso posterior:
  ```bash
  curl -s -b /tmp/cookies.txt http://localhost:8088/api/...
  ```

---

## TEST-USER-FLOW — Flujo de validación con el usuario

1. Después de implementar los cambios de una spec, preguntar **SIEMPRE**:
   🤖 *«¿Querés que corra el server en local para que hagas pruebas?»*
2. **NO marcar tareas como completadas** hasta que el usuario confirme que las pruebas manuales fueron satisfactorias.
3. Preguntar explícitamente:
   🤖 *«¿Las pruebas fueron satisfactorias? ¿Puedo cerrar el spec?»*
4. Solo después de la confirmación se marcan los criterios como cumplidos y se cambia el estado.
5. Si el usuario reporta errores: corregir, volver a preguntar, repetir.

---

## TEST-ROBOT — Convención de comunicación

- El ícono 🤖 se usa **antes** de cualquier pregunta que sea parte del flujo de interacción (correr server, confirmar implementación, cerrar spec).
- Indica al usuario que es una pregunta de flujo, no información técnica.

---

## TEST-CHECKLIST — Checklist de calidad pre-release

Antes de pasar una spec a `pending_release`:

- [ ] Todos los requerimientos P0 están implementados.
- [ ] Los criterios de aceptación pasan.
- [ ] Hay tests unitarios/integración si aplica.
- [ ] La documentación técnica se actualizó si aplica.
- [ ] Code review hecho.
- [ ] No hay deuda técnica crítica sin documentar.
- [ ] Funciona en un entorno con recursos limitados (iHost).
- [ ] Los selectores usan `Select` y no `<select>` nativo (`npm run check:ui` pasa).
- [ ] Toda clave i18n usada existe en `es` y `en` (`npm run check:i18n` pasa).
- [ ] Los campos numéricos usan `NumberInput` y no `<input type="number">` (`npm run check:numbers` pasa).
- [ ] Los modales no se cierran al hacer click fuera; el overlay bloquea el fondo; el cierre visible existe en todos los estados; se respeta la escala de capas.
- [ ] Las páginas de detalle no tienen íconos de acción sueltos: Editar/Eliminar viven en el menú de 3 puntos (`CardMenu`).
- [ ] Si el comportamiento depende de datos, se validó contra una copia de la DB de producción.
- [ ] `git diff --cached` revisado: sin PII, secretos, DBs, PDFs ni exports.
- [ ] Hooks instalados (`core.hooksPath=.githooks`) y no se usó `--no-verify`.
- [ ] La tabla «Cumplimiento de reglas» de la spec está completa.
