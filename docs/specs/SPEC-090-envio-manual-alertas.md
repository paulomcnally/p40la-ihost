---
title: "Botón de envío manual de alertas desde Configuración → Alertas"
id: "SPEC-090"
status: "released"
author: "opencode"
created: "2026-09-19"
updated: "2026-09-19"
github_issue: 93
---

# Botón de envío manual de alertas desde Configuración → Alertas

**ID**: SPEC-090  
**Estado**: released  
**Autor**: opencode  
**Creado**: 2026-09-19  
**Actualizado**: 2026-09-19

---

## 1. Resumen Ejecutivo

Las alertas del sistema (seguros de autos vencidos, resumen diario de facturas, cuotas de deudas que vencen hoy y nueva factura generada) se despachan **exclusivamente** por los schedulers automáticos (`AlertScheduler`, `BillSummaryScheduler`, `DebtDueScheduler`, `BillingScheduler`), que solo corren a la hora configurada (`alert_check_hour`) y con deduplicación diaria (keys `last_*_check`). Esto impide al usuario **probar las alertas** cuando quiere: no puede forzar un envío para verificar que el mail, la voz (Alexa) o el push de Telegram llegan correctamente sin esperar al horario programado.

Esta spec agrega un botón **"Enviar alertas ahora"** en la página Configuración → Alertas que dispara los schedulers probables de inmediato, respetando los canales habilitados por alerta (mail / voz / telegram), pero **sin afectar el flujo automatizado**: no modifica la hora de check ni las keys `last_*_check`, por lo que el envío programado del día siguiente (o del mismo día, si aún no corrió) funciona exactamente igual.

Impacto en iHost: mínimo. El botón reutiliza la lógica de envío ya existente de los schedulers (no agrega dependencias, no agrega tablas ni columnas). Es un endpoint nuevo + refactor de una función por scheduler + un botón con feedback en la UI.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Botón **"Enviar alertas ahora"** en `SettingsAlertsPage.tsx`, arriba de la lista de alertas, con estado de carga (disabled mientras corre) y feedback al finalizar (toast de éxito/error y resumen de lo enviado).
2. **REQ-002**: Endpoint nuevo `POST /api/alerts/send-now` (autenticado) que dispara el envío manual de los 4 schedulers probables: seguros de autos (`AlertKeyInsurance`), resumen de facturas (`AlertKeyBillSummary`), cuotas que vencen hoy (`AlertKeyDebtDue`) y aviso de prueba de factura generada (`AlertKeyBillCreated`, sin generar facturas reales).
3. **REQ-003**: El envío manual **salta la hora configurada y el dedup diario** (no consulta `alert_check_hour` ni las keys `last_*_check`), pero **respeta los canales habilitados por alerta** (`mail_enabled` / `voice_enabled` / `telegram_enabled`) y los gates maestros existentes (SMTP+destinatarios, Voice Monkey activo+enviando, bot de Telegram habilitado+token+chat_ids). Cada scheduler ya aplica esos gates en su dispatch; el refactor los conserva.
4. **REQ-004**: El envío manual **NO escribe** las keys `last_alert_check`, `last_bill_summary_check`, `last_debt_due_check` ni `last_billing_generation`: el dedup del automático queda intacto (si el check del día ya corrió, el manual no lo "consume"; si no corrió, el automático seguirá mandando a su hora).
5. **REQ-005**: Para `AlertKeyBillCreated` el envío manual envía un **aviso de prueba genérico** (un aviso de "nueva factura generada") sin crear ni modificar facturas, usando el speech de la alerta (`AlertService.Speech`), el formatter de Telegram `formatBillCreatedAlert(1)` y un email de prueba simple.
6. **REQ-006**: El endpoint responde un **resumen por alerta**: por cada key, qué canales se despacharon, cuántos items (autos/cuotas/facturas) tenía el evento o si estaba deshabilitada/sin datos. La UI lo muestra en el toast o en un detalle breve.
7. **REQ-007**: i18n en `frontend/public/i18n/{es,en}.json` bajo `settings.alerts.*`: label del botón, hint, toasts de éxito/error y resumen.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-008**: Tests unitarios de la nueva lógica de envío forzado por scheduler (salta hora/dedup, respeta canales, no escribe `last_*`) y del handler `POST /api/alerts/send-now`.
2. **REQ-009**: Si el usuario abre la página mientras corre el envío, el botón queda deshabilitado para evitar doble disparo.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-010**: Indicador en el resumen de cuáles canales están deshabilitados por falta de prerequisitos (ej: "Telegram no enviado: bot no configurado"), espejo de los hints actuales de la página.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: el endpoint ejecuta los 4 schedulers secuencialmente (mismo patrón que los schedulers actuales). El envío de mails/voz/telegram ya es no bloqueante por canal (logs de error sin interrumpir). Tiempo esperado: segundos (envíos de red con timeout existente).
- **Seguridad**: endpoint autenticado con la sesión (`authMiddleware`), igual que `GET/PUT /api/alerts`. No expone tokens ni config de SMTP/Telegram.
- **Almacenamiento**: sin migraciones, sin tablas ni columnas nuevas.
- **Disponibilidad**: si un canal falla (SMTP caído, Telegram sin red, Voice Monkey off), el resto de los canales/alertas se envían igual (patrón no bloqueante actual).
- **iHost**: cero dependencias nuevas; solo lógica Go propia y un botón en el bundle de React ya compilado.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- `internal/services/alert_scheduler.go`, `bill_summary_scheduler.go`, `debt_due_scheduler.go`, `billing_scheduler.go`: cada uno tiene `Start()/Stop()/run()` con ticker de 1h y un método `checkAndX()` que: (1) valida canales habilitados de la alerta, (2) compara `currentUserNow().Hour()` con `alert_check_hour`, (3) compara `last_*_check` con la fecha de hoy, (4) recolecta datos y envía por los canales habilitados, (5) escribe `last_*_check = hoy`.
- `AlertScheduler`/`BillSummaryScheduler`/`DebtDueScheduler` ya tienen `CheckNow()` (usado por tests), pero **llaman a `checkAndX()` completo**, que incluye los gates de hora y dedup → hoy NO sirve para forzar un envío manual real.
- `internal/api/alerts_handlers.go`: `AlertsHandlers` recibe `*AlertService` y `*SystemSettingsService`; para el envío manual necesita referencia a los 4 schedulers (o a un agregador).
- `frontend/src/pages/SettingsAlertsPage.tsx`: lista de alertas con toggles Mail/Alexa/Telegram, gating por prerequisitos, sin botón de envío.
- `frontend/src/api/index.ts`: `api.alerts.list/update`; falta `sendNow`.
- `frontend/public/i18n/es.json` / `en.json`: claves `settings.alerts.*` (fuente de verdad; jamás editar `public/i18n/`).
- `internal/services/telegram_alerts.go`: `formatBillCreatedAlert(1)`, `formatInsuranceAlerts`, `formatDebtsDueToday`, etc. (reutilizables para el push manual).
- `internal/services/bill_email.go`: `buildBillCreatedEmail(svc, bill, symbol, format)` — para el aviso de prueba de `bill_created` se usará un contenido genérico (sin servicio/factura reales), ver ADR-003.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Refactor: extraer `sendXNow()` (sin gates de hora/dedup, sin escribir `last_*`) por scheduler + handler que los invoca | Reutiliza 100% la lógica de envío existente; el automático no cambia; los gates de canal se conservan | Refactor de 4 schedulers (mínimo: mover el cuerpo del envío a un método privado) | ✅ Seleccionada |
| Nuevo endpoint que reimplementa la recolección/envío por fuera de los schedulers | No toca los schedulers | Duplica lógica de recolección y de dispatch; dos fuentes de verdad del "qué enviar"; más código a mantener | ❌ Rechazada |
| Exponer `CheckNow()` tal cual (con hora/dedup) en la API | Cero refactor | No envía si la hora no coincide o si ya corrió hoy → el botón no cumpliría "enviar ahora" | ❌ Rechazada |
| Botón por alerta (en lugar de uno global) | Permite probar una alerta puntual | Más clicks; el usuario pidió un botón global ("Enviar alertas ahora"); se puede agregar luego | ❌ Rechazada (futuro) |
| Script CLI/curl fuera de la app | Rápido para el agente | El usuario quiere un botón en la UI, no comandos | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: El envío manual es un método `SendNow()` (o `ForceSend()`) por scheduler que **salta hora y dedup pero conserva los gates de canal** y **no escribe `last_*`**.
- **Contexto**: el cuerpo de `checkAndX()` mezcla gates (hora, dedup) con la lógica de recolección+envío. `CheckNow()` reutiliza todos los gates, lo que impide forzar el envío.
- **Decisión**: refactorizar cada scheduler para separar:
  - `checkAndX()` (ticker): gates de hora + dedup → llama `sendXNow()` → escribe `last_*`.
  - `SendNow()` (manual): llama directo `sendXNow()`, sin gates de hora/dedup y **sin escribir `last_*`**.
- **Consecuencias**: el automático queda 100% intacto; el manual reutiliza la misma recolección y el mismo dispatch por canal. Solo se toca la estructura interna de cada scheduler (no su contrato con el ticker).

**ADR-002**: El botón global dispara los 4 schedulers probables (seguros, resumen, deudas hoy, factura generada de prueba). No incluye las alertas de pensión (evento-driven, dependen de datos de un evento puntual y no de un check diario).
- **Contexto**: el usuario pidió "un botón que permita el envío de las alertas manual para probarlas". Las alertas de pensión (`pension_records_created`, `pension_record_paid`, etc.) se disparan por eventos de negocio (generar mes, marcar pagado), no por un check; forzarlas requeriría fabricar eventos falsos.
- **Decisión**: alcance = 4 schedulers con check. Las alertas de pensión quedan fuera (P2/futuro, con su propia lógica de "probar con datos de ejemplo").
- **Consecuencias**: el botón prueba las alertas que el usuario más menciona (seguros, facturas, deudas); pensión se documenta como no incluida.

**ADR-003**: `AlertKeyBillCreated` se incluye como **aviso de prueba genérico** sin generar facturas.
- **Contexto**: `BillingScheduler` solo envía esa alerta cuando efectivamente genera facturas automáticas (no es un check). El usuario pidió incluirla.
- **Decisión**: el envío manual de `bill_created` envía el speech de la alerta (voz), `formatBillCreatedAlert(1)` (Telegram) y un email de prueba con contenido genérico (título "Nueva factura generada", cuerpo simple) — **sin crear facturas ni tocar `last_billing_generation`**.
- **Consecuencias**: se puede verificar el canal completo (mail/voz/telegram) de la alerta de factura sin efectos colaterales en la facturación.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
Frontend SettingsAlertsPage
   │  click "Enviar alertas ahora"
   ▼
POST /api/alerts/send-now (authMiddleware)
   │
   ▼
AlertsHandlers.SendNow ──► scheduler.AlertScheduler.SendNow()
                     ──► scheduler.BillSummaryScheduler.SendNow()
                     ──► scheduler.DebtDueScheduler.SendNow()
                     ──► scheduler.BillingScheduler.SendNowBillCreated()
   │  cada uno respeta canales habilitados por alerta
   ▼
dispatch mail (EmailService) / voz (VoiceMonkeyService) / telegram (dispatchTelegram)
   │
   ▼
Resumen por alerta → JSON → toast en la UI
```

### 4.2 Componentes

#### 4.2.1 Schedulers (4 archivos)
- **Responsabilidad**: separar la lógica de envío de los gates de hora/dedup.
- **Cambios**:
  - `internal/services/alert_scheduler.go`: extraer `sendXNow()` con recolección + dispatch (mail/voz/telegram) y exponer `SendNow() error` (o con resumen). `checkAndAlert()` conserva los gates de hora/dedup, llama a la lógica extraída y escribe `last_alert_check`.
  - `internal/services/bill_summary_scheduler.go`: idem con `SendNow()`; `checkAndSend()` conserva hora/dedup y escribe `last_bill_summary_check`.
  - `internal/services/debt_due_scheduler.go`: idem con `SendNow()`; conserva `last_debt_due_check`.
  - `internal/services/billing_scheduler.go`: agregar `SendNowBillCreated()` que envía el aviso de prueba genérico (ADR-003) sin generar facturas ni escribir `last_billing_generation`.

#### 4.2.2 `internal/api/alerts_handlers.go`
- **Responsabilidad**: exponer el endpoint de envío manual.
- **Cambios**: `AlertsHandlers` recibe también los 4 schedulers (o un `AlertSendService` agregador). Nuevo handler `SendNow` que los invoca, junta los resultados y responde el resumen. `internal/api/handlers.go` y `cmd/server/main.go` se actualizan para pasar las dependencias.

#### 4.2.3 `internal/api/routes.go`
- **Cambios**: `mux.Handle("POST /api/alerts/send-now", authMiddleware(http.HandlerFunc(handler.alerts.SendNow)))`.

#### 4.2.4 `frontend/src/pages/SettingsAlertsPage.tsx`
- **Responsabilidad**: botón global de envío manual.
- **Cambios**: botón "Enviar alertas ahora" arriba de la lista (patrón visual consistente con los botones de la app, ej. `SettingsEmailAlertsPage`), con estado `sending` (disabled + spinner/texto), y toast con el resumen devuelto por el endpoint.

#### 4.2.5 `frontend/src/api/index.ts`
- **Cambios**: `api.alerts.sendNow()` → `POST /api/alerts/send-now`, tipado del resumen.

#### 4.2.6 i18n
- **Cambios**: claves nuevas en `frontend/public/i18n/es.json` / `en.json` bajo `settings.alerts.*` (label, hint, enviando, éxito, error, resumen). **Editar siempre `frontend/public/i18n/`, nunca `public/i18n/`** (regla de AGENTS.md).

### 4.3 Modelo de datos

Sin cambios de esquema. Sin migraciones. Se reutilizan la tabla `alerts`, `system_settings` y los storages existentes (`AutoStorage`, `BillStorage`, `DebtBillStorage`).

### 4.4 APIs / Contratos

#### Endpoint: `POST /api/alerts/send-now`

**Request**: sin cuerpo.

**Response 200**:
```json
{
  "message": "Alertas enviadas",
  "results": [
    {
      "key": "insurance",
      "title": "Seguros de autos vencidos",
      "sent_channels": ["mail", "telegram"],
      "items": 2,
      "detail": "2 vehículos en alerta"
    },
    {
      "key": "bill_summary",
      "title": "Resumen diario de facturas",
      "sent_channels": ["telegram"],
      "items": 5,
      "detail": "5 facturas pendientes"
    },
    {
      "key": "debt_due",
      "title": "Cuotas de deudas que vencen hoy",
      "sent_channels": [],
      "items": 0,
      "detail": "No hay cuotas que vencen hoy"
    },
    {
      "key": "bill_created",
      "title": "Nueva factura generada",
      "sent_channels": ["mail", "voice"],
      "items": 1,
      "detail": "Aviso de prueba enviado (sin generar facturas)"
    }
  ]
}
```

**Response Error**:
```json
{
  "error": "internal_error",
  "message": "descripción"
}
```

Notas:
- `sent_channels` solo incluye canales efectivamente despachados (con la alerta habilitada y el gate maestro cumplido).
- Si una alerta está deshabilitada en todos los canales, `sent_channels` queda vacío y `detail` lo explica (no es error).
- El handler nunca falla por un canal roto: los errores de SMTP/Telegram/Voice Monkey se registran por scheduler (patrón actual) y el resumen refleja lo enviado.

### 4.5 Dependencias

- **Internas**: `AlertScheduler`, `BillSummaryScheduler`, `DebtDueScheduler`, `BillingScheduler`, `AlertsHandlers`, `Handler`, `main.go`, `routes.go`, `SettingsAlertsPage.tsx`, `api/index.ts`, i18n es/en.
- **Externas**: ninguna.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Dado la página Configuración → Alertas, cuando se pulsa "Enviar alertas ahora", entonces se despachan las 4 alertas (seguros, resumen, deudas hoy, factura de prueba) por los canales habilitados de cada una y se muestra un resumen/feedback.
- [ ] CA-002: Dado que una alerta tiene `mail_enabled=1` con SMTP y destinatarios configurados, cuando se pulsa el botón, entonces llega el mail correspondiente a los destinatarios (contenido real de la alerta). *(validación manual con SMTP real pendiente)*
- [ ] CA-003: Dado que una alerta tiene `voice_enabled=1` con Voice Monkey activo y "enviar alertas" on, cuando se pulsa el botón, entonces se anuncia el speech de esa alerta por el altavoz. *(validación manual con altavoz pendiente)*
- [ ] CA-004: Dado que una alerta tiene `telegram_enabled=1` con bot habilitado + token + chat_ids, cuando se pulsa el botón, entonces llega el push a los chat_ids con el formato de los comandos. *(validación manual con bot real pendiente)*
- [x] CA-005: Dado que una alerta está deshabilitada o sin prerequisitos (SMTP off, VM off, bot off), cuando se pulsa el botón, entonces esa alerta no envía nada por el canal faltante (y el resto sigue enviando); el resumen lo refleja sin error.
- [x] CA-006: Dado que el check automático ya corrió hoy (o no), cuando se pulsa el botón manual, entonces las keys `last_alert_check` / `last_bill_summary_check` / `last_debt_due_check` / `last_billing_generation` NO cambian: el scheduler automático del día siguiente (o de la próxima hora coincidente) envía igual.
- [x] CA-007: Dado el envío manual de `bill_created`, cuando se pulsa el botón, entonces NO se crean/modifican facturas ni se toca `last_billing_generation`; solo se envía el aviso de prueba.
- [x] CA-008: Dado un canal que falla (SMTP caído, Telegram sin red), cuando se pulsa el botón, entonces el resto de canales/alertas se envían igual y el endpoint responde 200 con el resumen de lo enviado.
- [ ] CA-DARK: El botón y los estados (disabled, textos) usan tokens del tema y se verificó legibilidad en darkmode. *(verificación visual pendiente)*

### 5.2 No funcionales

- [x] CA-NF-001: Cero dependencias nuevas y cero migraciones; el binario no crece con librerías.
- [x] CA-NF-002: El endpoint está protegido por sesión (sin cookie → 401, igual que el resto de `/api/alerts`).

### 5.3 Testing

- **Unit tests**: por cada scheduler, `SendNow()` salta hora/dedup, respeta canales habilitados y no escribe `last_*`; `BillingScheduler.SendNowBillCreated` no crea facturas. Handler `POST /api/alerts/send-now`: respuesta con resumen por alerta, 401 sin sesión, no falla con canales rotos (services mock).
- **Integration tests**: scheduler con mocks de EmailService/VoiceMonkeyService/TelegramBotService que verifican destinatarios/textos; verificar que `last_*` queda intacto tras `SendNow()`.
- **E2E tests**: manual en local — activar canales, pulsar el botón, verificar llegada de mail/voz/telegram; deshabilitar una alerta y ver que no envía; verificar en darkmode.
- **iHost**: verificar que el botón funciona contra la DB del volumen sin aumento de memoria relevante y que el scheduler automático sigue corriendo a su hora.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Refactor de los 4 schedulers: extraer lógica de envío + métodos `SendNow()` / `SendNowBillCreated()` (sin gates hora/dedup, sin escribir `last_*`) | 0.5 día | Ninguna |
| 2 | Handler `POST /api/alerts/send-now` + wiring (`AlertsHandlers`, `Handler`, `main.go`, `routes.go`) | 0.5 día | Fase 1 |
| 3 | Frontend: botón + estado de carga + resumen en `SettingsAlertsPage.tsx`, `api.alerts.sendNow()`, i18n es/en | 0.5 día | Fase 2 |
| 4 | Tests unitarios/integración + build + validación manual local (server corriendo, canales reales, darkmode) | 0.5 día | Fases 1-3 |

### 6.2 Milestones

1. **MVP**: Refactor de schedulers + endpoint + botón con toast de éxito/error (sin resumen detallado).
2. **V1.0**: Resumen por alerta en la respuesta/toast + tests + validación manual completa (REQ-006/007).

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| El refactor altera el comportamiento del scheduler automático | Media | Alto | Extraer el cuerpo de envío tal cual (sin cambiar lógica); tests existentes de schedulers (`*_scheduler_test.go`) deben seguir verdes; validación manual del automático a su hora |
| Envío manual duplicado con el automático del mismo día | Baja | Bajo | Es el comportamiento pedido (prueba); el manual no escribe `last_*`, así que el automático decide por sí mismo |
| Botón sin feedback claro (spam de mails por doble click) | Baja | Medio | Estado disabled durante la ejecución (REQ-009) |
| Espera larga si SMTP/Telegram cuelgan | Media | Medio | Reutilizar timeouts existentes de EmailService/Telegram; envío no bloqueante por canal (log de error) |
| El usuario confunde "enviar ahora" con "generar facturas" (bill_created) | Media | Bajo | Detalle explícito en el resumen: "aviso de prueba (sin generar facturas)" y en el hint del botón |

## 8. Notas y Referencias

- SPEC-029/030/031/032/033: sistema de alertas multicanal (mail/voz), schedulers y dedup diario.
- SPEC-037: gating de toggles por canal (prerequisitos de SMTP/VM/bot).
- SPEC-054: scheduler de cuotas de deudas que vencen hoy.
- SPEC-088: canal telegram en alertas, `dispatchTelegram`, formatters reutilizables (`formatInsuranceAlerts`, `formatDebtsDueToday`, `formatBillCreatedAlert`).
- SPEC-078: zona horaria de los horarios de envío (los gates de hora usan `currentUserNow`).
- `frontend/public/i18n/es.json` / `en.json`: fuente de verdad del i18n (editar ahí, nunca en `public/i18n/`).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-19 | opencode | Creación inicial de la especificación |
| 2026-09-19 | opencode | Implementación: refactor de los 4 schedulers (SendNow/SendNowBillCreated sin hora/dedup/last_*, dispatch devuelve bool), endpoint POST /api/alerts/send-now (interfaces AlertSenders/BillCreatedSender + wiring en main.go/routes.go), botón "Enviar alertas ahora" en SettingsAlertsPage con resumen, api.alerts.sendNow(), i18n es/en, tests unitarios (schedulers + handler). Build y tests verdes; validación manual local del endpoint (canales, dedup intacto, SMTP caído no bloquea, 401 sin sesión). |
| 2026-09-19 | opencode | Release: merge de feature/SPEC-090 a main, push, issue #93 cerrado con label spec/released |