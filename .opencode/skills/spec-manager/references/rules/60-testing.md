# 60 — Pruebas y verificación

> **Dominio**: pruebas locales, verificación del servicio, flujo de validación con el usuario y checklist.
> **Cargar siempre antes de `pending_release`**.

---

## TEST-LOCAL — Pruebas locales obligatorias

- Todo cambio significativo se prueba en local: build, tests y ejecución básica.
- **No se commitea** código que no compile, que falle tests o que no se haya validado.
- Si el resultado depende de datos reales, se prueba contra copia de producción (`DATA-PRODCOPY`).

---

## TEST-SERVER — Verificar el servicio antes de avisar

- Antes de informar al usuario que puede acceder, verificar que el servicio responde: `{{HEALTH_CHECK_COMMAND}}`.
- Solo entonces informarlo. Si falla: revisar logs y reiniciar.

---

## TEST-USER-FLOW — Flujo de validación con el usuario

1. Después de implementar, preguntar siempre:
   🤖 *«¿Querés que corra el server en local para que hagas pruebas?»*
2. **No marcar tareas como completadas** hasta la confirmación del usuario.
3. Preguntar: 🤖 *«¿Las pruebas fueron satisfactorias? ¿Puedo cerrar el spec?»*
4. Solo con la confirmación se marcan criterios y se cambia el estado.
5. Si el usuario reporta errores: corregir, volver a preguntar, repetir.

---

## TEST-ROBOT — Convención de comunicación

- El ícono 🤖 precede a toda pregunta que sea parte del flujo de interacción.

---

## TEST-CHECKLIST — Checklist pre-release

- [ ] Requerimientos P0 implementados.
- [ ] Criterios de aceptación verificados.
- [ ] `{{BUILD_COMMAND}}` pasa.
- [ ] `{{TEST_COMMAND}}` pasa.
- [ ] Guardrails de UI pasan (si aplica).
- [ ] Probado en local y confirmado por el usuario (🤖).
- [ ] Si depende de datos: validado contra copia de producción.
- [ ] `git diff --cached` sin PII, secretos ni volcados.
- [ ] Tabla «Cumplimiento de reglas» completa.
