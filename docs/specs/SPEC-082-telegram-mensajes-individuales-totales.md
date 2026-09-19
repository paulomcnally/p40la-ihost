---
title: "Bot Telegram: un mensaje por servicio y por deuda + totales por moneda al final"
id: "SPEC-082"
status: "released"
author: "paulomcnally"
created: "2026-09-19"
updated: "2026-09-19"
github_issue: 85
---

# Bot Telegram: un mensaje por servicio y por deuda + totales por moneda al final

**ID**: SPEC-082  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-09-19  
**Actualizado**: 2026-09-19

---

## 1. Resumen Ejecutivo

El bot de Telegram (SPEC-079, migrado a Go) responde `/servicios_pendientes` y `/deudas_pendientes` con **un solo mensaje** que agrupa todos los servicios (o todas las deudas) en un texto concatenado. Cuando hay muchos registros el mensaje se vuelve largo y difícil de leer en el móvil, y no hay una visión rápida del total que se debe pagar.

Esta spec cambia el comportamiento para enviar **un mensaje por cada servicio** (en `/servicios_pendientes`) y **un mensaje por cada deuda** (en `/deudas_pendientes`), con el mismo contenido por ítem (nombre, casa/institución, cantidad y monto). Además, **al final de cada comando** se envía **un mensaje con los totales agrupados por moneda** (USD y NIO, o las monedas que tenga el sistema): total de facturas pendientes por moneda y total de cuotas pendientes por moneda.

Consideraciones iHost: cambios solo en `internal/services/telegram_bot.go` (formato y envío de mensajes) y en los structs/queries de lectura de pendientes (`internal/models/bill.go`, `internal/models/debt_bill.go`, `internal/storage/bill.go`, `internal/storage/debt_bill.go`) para exponer el código de moneda. Sin cambios de esquema SQLite (la tabla `currencies` ya existe con `code` y `symbol`), sin dependencias nuevas y sin impacto en memoria (los mensajes se envían secuencialmente, uno por uno).

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: `/servicios_pendientes` envía **un mensaje por cada servicio** con facturas pendientes. Cada mensaje incluye: nombre del servicio, casa, cantidad de facturas pendientes y monto total de ese servicio (formato actual de `formatServiciosPendientes` por grupo).
2. **REQ-002**: `/deudas_pendientes` envía **un mensaje por cada deuda** con cuotas pendientes. Cada mensaje incluye: descripción de la deuda, institución, cantidad de cuotas pendientes y monto total de esa deuda.
3. **REQ-003**: Al final de `/servicios_pendientes` se envía **un mensaje de totales**: suma de facturas pendientes agrupada por moneda (ej: `Total USD: $1,200.00` y `Total NIO: C$8,000.00`). Solo se muestran las monedas presentes en los datos del sistema.
4. **REQ-004**: Al final de `/deudas_pendientes` se envía **un mensaje de totales**: suma de cuotas pendientes agrupada por moneda (mismas reglas que REQ-003).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-005**: Si no hay pendientes (0 servicios o 0 deudas), se envía un solo mensaje (`✅ No hay ...`) sin mensaje de totales.
2. **REQ-006**: Se mantiene el orden actual por monto descendente (`sortGroups`/`sortDebtGroups`) y el formato de moneda configurado (SPEC-058: separadores y dígitos decimales).
3. **REQ-007**: El agrupado de totales usa el **código de moneda** (`USD`, `NIO`, etc.) del sistema; si el código está vacío se agrupa por símbolo como fallback.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-008**: Tests unitarios de los nuevos formateadores (mensajes por ítem + mensaje de totales), incluyendo el caso sin pendientes y el caso multi-moneda.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: N+1 llamadas a `sendMessage` por comando (N = cantidad de servicios/deudas + 1 total). Envío secuencial; sin goroutines nuevas. Sin impacto medible en iHost.
- **Seguridad**: Sin cambios de tokens, chat IDs ni exposición de secretos.
- **Almacenamiento**: Sin archivos nuevos ni cambios de esquema DB.
- **Disponibilidad**: Si falla el envío de un mensaje intermedio, se loguea y se continúa con el siguiente (comportamiento actual de `reply`).
- **iHost**: Sin dependencias nuevas; solo lógica de formato y envío en el servicio existente.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- `internal/services/telegram_bot.go` (SPEC-079/080): `handleServiciosPendientes` y `handleDeudasPendientes` llaman a `reply` (un solo `sendMessage`) con `formatServiciosPendientes` / `formatDeudasPendientes`, que concatenan todos los grupos en un único texto Markdown.
- `internal/storage/bill.go:211` y `internal/storage/debt_bill.go:109`: `ListPendingWithDetails` devuelven `PendingBillDetail` y `PendingDebtDetail`, ambos con `CurrencySymbol` pero **sin** el código de moneda.
- `migrations/0002_create_currencies.up.sql`: la tabla `currencies` ya existe (`code`, `name`, `symbol`); default NIO (C$) y USD ($). No hay tabla de tasas de cambio.
- `go-telegram/bot` permite múltiples `SendMessage` sin estado compartido; el límite de Telegram es 30 msg/s (muy por encima del volumen esperado).
- Límite de 4096 chars por mensaje: los mensajes concatenados actuales pueden acercarse a ese límite; los mensajes por ítem lo eliminan de raíz.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Un mensaje por ítem + mensaje de totales | Legible en móvil, sin límite de 4096 chars, totales claros | Más llamadas a la Bot API | ✅ Seleccionada |
| Un solo mensaje con todos los ítems (actual) | Una sola llamada | Ilegible con muchos registros, riesgo de límite 4096 | ❌ Rechazada |
| Un mensaje por ítem sin totales | Mínimo cambio | No responde al requerimiento del usuario | ❌ Rechazada |
| Totales con conversión USD/NIO vía tasa de cambio | Total único en una moneda | No existe infraestructura de tasas en el sistema | ❌ Rechazada (se agrupa por moneda) |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Totales agrupados por moneda (sin conversión)
- **Contexto**: El usuario pide "el total en USD y el total en NIO (o la moneda según el sistema)". No existe tabla de tasas de cambio ni moneda única del sistema; cada servicio/deuda define su moneda vía `currency_id`.
- **Decisión**: El mensaje de totales suma los montos pendientes agrupados por código de moneda (`USD`, `NIO`, ...) y muestra un total por moneda. Solo aparecen las monedas presentes en los datos.
- **Consecuencias**: Simple, sin infraestructura nueva. Si en el futuro se quiere un total convertido a una moneda base, será una spec aparte con tasas de cambio.

**ADR-002**: Exponer `CurrencyCode` en los detalles de pendientes
- **Contexto**: Los structs `PendingBillDetail`/`PendingDebtDetail` tienen el símbolo pero no el código de moneda; agrupar por símbolo es frágil (dos monedas pueden compartir símbolo).
- **Decisión**: Agregar el campo `CurrencyCode` a ambos structs y al `SELECT` de `ListPendingWithDetails` (el `JOIN currencies` ya está en ambas queries).
- **Consecuencias**: Cambio mínimo en storage + tests existentes; agrupado robusto por código con fallback a símbolo.

**ADR-003**: Envío secuencial con el helper `reply` existente
- **Contexto**: El envío actual usa `reply` (MarkdownV1). Enviar N+1 mensajes no requiere primitivas nuevas.
- **Decisión**: Reemplazar la llamada única a `reply` por un loop que envía cada mensaje con la misma semántica de errores (log + continuar).
- **Consecuencias**: Comportamiento de error idéntico al actual; sin cambios de librería.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Telegram Update /servicios_pendientes]
        │
        ▼
[TelegramBotService.handleServiciosPendientes]
        │  BillStorage.ListPendingWithDetails()  (con CurrencyCode)
        ▼
[formatServiciosPendientes] ──► N mensajes (uno por servicio)
[formatServiciosTotales]    ──► 1 mensaje (totales por moneda)
        │
        ▼
[bot.SendMessage × N+1]  (secuencial, MarkdownV1)
```

### 4.2 Componentes

#### 4.2.1 `TelegramBotService` (`internal/services/telegram_bot.go`)
- **Responsabilidad**: Recibir el comando, consultar pendientes y enviar N+1 mensajes.
- **Interfaz**: Handlers existentes `handleServiciosPendientes` / `handleDeudasPendientes` (sin cambios de firma).
- **Dependencias**: `BillStorage`, `DebtBillStorage`, `SystemSettingsService`.
- **Ubicación**: `internal/services/telegram_bot.go`.

#### 4.2.2 Formateadores (`internal/services/telegram_bot.go`)
- `formatServiciosPendientes` → pasa a devolver `[]string` (un mensaje por servicio) o se reemplaza por un builder por ítem.
- Nuevo `formatServiciosTotales(pending, format) string`: suma por `CurrencyCode` y construye `💰 *Totales*\n  USD: $X\n  NIO: C$Y` (solo monedas presentes).
- Análogos para deudas: `formatDeudasPendientes` → `[]string` y nuevo `formatDeudasTotales`.

### 4.3 Modelo de datos

```
Entidad: PendingBillDetail (internal/models/bill.go)
- CurrencySymbol: string (existente)
- CurrencyCode:   string (NUEVO — código de moneda, ej: "USD"/"NIO")

Entidad: PendingDebtDetail (internal/models/debt_bill.go)
- CurrencySymbol: string (existente)
- CurrencyCode:   string (NUEVO — código de moneda)
```

Sin cambios de esquema: `currencies` ya tiene `code`. Los `SELECT` de `ListPendingWithDetails` agregan `c.code` al escaneo.

### 4.4 APIs / Contratos

No hay cambios de API REST ni de frontend. Contrato interno:

```
[BillStorage.ListPendingWithDetails] → []PendingBillDetail { BillID, ServiceID, HomeID,
  HomeName, Institution, ServiceName, CurrencySymbol, CurrencyCode, Year, Month, Amount, Status, CreatedAt }

[DebtBillStorage.ListPendingWithDetails] → []PendingDebtDetail { DebtID, DebtDescription,
  InstitutionName, DueDate, Amount, CurrencySymbol, CurrencyCode }
```

### 4.5 Dependencias

- **Internas**: `internal/models/bill.go`, `internal/models/debt_bill.go`, `internal/storage/bill.go`, `internal/storage/debt_bill.go`, `internal/services/telegram_bot.go`, tests asociados (`telegram_bot_test.go`, `bill_pending_test.go`, `debt_bill_pending_test.go`).
- **Externas**: Ninguna nueva. `go-telegram/bot` (existente).

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Dado `/servicios_pendientes` con N servicios con facturas pendientes, cuando el usuario ejecuta el comando, entonces el bot envía **N mensajes** (uno por servicio, con nombre, casa, cantidad de facturas y monto total) y **1 mensaje de totales por moneda**.
- [x] CA-002: Dado `/deudas_pendientes` con M deudas con cuotas pendientes, cuando el usuario ejecuta el comando, entonces el bot envía **M mensajes** (uno por deuda, con descripción, institución, cantidad de cuotas y monto total) y **1 mensaje de totales por moneda**.
- [x] CA-003: Dado que no hay facturas pendientes, cuando el usuario ejecuta `/servicios_pendientes`, entonces el bot envía un único mensaje `✅ No hay facturas pendientes.` (sin mensaje de totales).
- [x] CA-004: Dado que no hay cuotas pendientes, cuando el usuario ejecuta `/deudas_pendientes`, entonces el bot envía un único mensaje `✅ No hay deudas pendientes.` (sin mensaje de totales).
- [x] CA-005: Dado pendientes en USD y NIO, cuando se genera el mensaje de totales, entonces se muestran `Total USD` y `Total NIO` con montos correctos según el formato configurado (SPEC-058).
- [x] CA-006: Dado un fallo de red en el envío de un mensaje intermedio, cuando el bot continúa, entonces los mensajes restantes se envían y el error queda en el log.

### 5.2 No funcionales

- [ ] CA-NF-001: El agrupado por moneda usa `CurrencyCode` con fallback a `CurrencySymbol`.
- [ ] CA-NF-002: Sin cambios de esquema SQLite ni dependencias nuevas.

### 5.3 Testing

- **Unit tests**: `formatServiciosPendientes`/`formatDeudasPendientes` (nuevo contrato `[]string`), `formatServiciosTotales`/`formatDebtTotales` (multi-moneda, moneda vacía, caso vacío). Actualizar tests existentes de storage (`bill_pending_test.go`, `debt_bill_pending_test.go`) con `CurrencyCode`.
- **Integration tests**: Ninguno nuevo (sin cambios de API).
- **E2E tests**: Prueba manual local con bot habilitado: ejecutar ambos comandos y verificar N+1 mensajes legibles + totales.
- **Carga/Performance**: Sin métricas nuevas; el envío secuencial no impacta a iHost.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Storage: agregar `CurrencyCode` a `PendingBillDetail`/`PendingDebtDetail` y a los `SELECT` de `ListPendingWithDetails` (ambos storages) + actualizar tests | 0.5 días | Ninguna |
| 2 | Services: convertir `formatServiciosPendientes`/`formatDebtPendientes` a lista de mensajes por ítem + nuevos `format*Totales` por moneda | 0.5 días | Fase 1 |
| 3 | Services: handlers envían N+1 mensajes secuencialmente | 0.25 días | Fase 2 |
| 4 | Tests unitarios de los formateadores y validación manual local (correr server, probar con bot real) | 0.5 días | Fase 3 |

### 6.2 Milestones

1. **MVP**: `/servicios_pendientes` y `/deudas_pendientes` envían mensajes individuales + mensaje de totales por moneda.
2. **V1.0**: Cobertura de tests unitarios y validación manual con el usuario.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Rate limit de Telegram (30 msg/s) | Baja | Medio | Envío secuencial; volúmenes esperados son decenas, no miles |
| Cambio de contrato de formateadores rompe tests existentes | Media | Bajo | Actualizar tests en la misma fase (Fase 1-2) |
| Monedas sin código (NULL) agrupan mal | Baja | Bajo | Fallback a `CurrencySymbol` (REQ-007) |
| El usuario esperaba conversión USD↔NIO | Media | Medio | ADR-001 documentado; si se requiere, spec futura con tasas de cambio |

## 8. Notas y Referencias

- SPEC-079 (Bot Telegram integrado al server), SPEC-080 (fix comandos + SetMyCommands), SPEC-058 (formato de moneda configurable).
- `docs/ssh-ihost-access.md` (acceso a producción en iHost, si aplica para validación).
- Documentación de `go-telegram/bot`: `SendMessage` con `ParseModeMarkdownV1` (ya usado en `reply`).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-19 | paulomcnally | Creación inicial de la especificación |
| 2026-09-19 | paulomcnally | Implementación: `CurrencyCode` en `PendingBillDetail`/`PendingDebtDetail` + SELECTs; `formatServiciosPendientes`/`formatDeudasPendientes` → `[]string` (un mensaje por ítem); nuevos `formatServiciosTotales`/`formatDeudasTotales` (totales por moneda); `sendMany` para envío secuencial N+1. Tests actualizados (storage + formateadores). Estado → in_progress |
| 2026-09-19 | paulomcnally | Validación manual del usuario satisfactoria. Criterios de aceptación en pass. Estado → pending_release |
| 2026-09-19 | paulomcnally | Release: merge `feature/SPEC-082` → `main` (commit 4c963e3), issue #85 cerrado con label `spec/released`. Estado → released |