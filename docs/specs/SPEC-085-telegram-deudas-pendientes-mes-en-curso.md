---
title: "Bot Telegram: limitar /deudas_pendientes al mes en curso"
id: "SPEC-085"
status: "pending_release"
author: "paulomcnally"
created: "2026-09-19"
updated: "2026-09-19"
github_issue: 88
---

# Bot Telegram: limitar /deudas_pendientes al mes en curso

**ID**: SPEC-085  
**Estado**: pending_release  
**Autor**: paulomcnally  
**Creado**: 2026-09-19  
**Actualizado**: 2026-09-19

---

## 1. Resumen Ejecutivo

El comando `/deudas_pendientes` del bot de Telegram (SPEC-080/082/083) consulta `DebtBillStorage.ListPendingWithDetails`, que devuelve **todas** las cuotas con `status='pending'` sin importar su fecha de vencimiento. En deudas de largo plazo (ej: Hipoteca con cuotas hasta 2033) esto genera una lista enorme de cuotas **futuras** (de años venideros) que no aportan nada hoy: al usuario le interesa saber qué tiene pendiente **del pasado hasta el mes en curso**.

El usuario reporta: *"cuando consulto las deudas pendientes me envía el dato de todo el historial pendiente, eso no está bien, tiene que enviarme del pasado al presente, no me interesa saber deudas de un año futuro. Tienes que limitarte al mes en curso."*

> **Nota de implementación (2026-09-19)**: el requerimiento de esta spec quedó implementado y liberado en `main` por **SPEC-086** (`/deudas_pendientes` con el mismo formato de `/servicios_pendientes`): el filtro `dueInRange` (vencidas SIEMPRE + no vencidas solo dentro de los próximos `showMonths` meses, default `1` = mes en curso) se aplica en `formatDeudasPendientes`, calculado en la zona horaria configurada (SPEC-084) con fallback UTC. El resultado visible es exactamente el pedido: del pasado al mes en curso, sin cuotas de años futuros. La solución de cutoff en SQL que proponía esta spec (ADR-002) queda descartada: el filtro en el formateo es configurable (`telegram_bot_show_months`) y evita duplicar lógica.

Consideraciones iHost: el filtro reduce las filas formateadas por el bot (menos memoria en el iHost). Sin cambios de esquema SQLite, API, frontend ni dependencias nuevas.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: `/deudas_pendientes` devuelve **solo** cuotas pendientes (`status='pending'`, `deleted_at IS NULL`) cuyo `due_date` sea menor o igual al **último día del mes en curso** (del pasado al presente). Las cuotas pendientes con vencimiento en meses futuros quedan excluidas.
2. **REQ-002**: El límite "último día del mes en curso" se calcula usando la **zona horaria configurada** del sistema (setting `timezone`, SPEC-078) con fallback UTC (mismo comportamiento que los schedulers).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-003**: El mensaje final de **totales por moneda** (SPEC-082 REQ-004) se calcula sobre la lista ya filtrada (solo mes en curso), sin cambios de código: refleja automáticamente las cuotas del mes.
2. **REQ-004**: Si no hay cuotas pendientes dentro del mes en curso (aunque existan cuotas futuras), se envía un mensaje único claro (`✅ No hay deudas pendientes.` en la implementación de SPEC-086).
3. **REQ-005**: Tests actualizados (storage y bot) y validación manual local.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-006**: Evaluar si el mismo criterio aplica a `/servicios_pendientes` (facturas). **Fuera de alcance** de esta spec: el usuario reportó solo el caso de deudas; se documenta para una futura spec. (Nota: SPEC-084/086 extendieron `showMonths` a servicios y deudas por igual.)

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Menos filas procesadas → menos memoria en el formateo del iHost.
- **Seguridad**: Sin cambios de tokens, chat IDs ni secretos.
- **Almacenamiento**: Sin archivos nuevos ni cambios de esquema DB.
- **Disponibilidad**: Si falla el envío de un mensaje intermedio, se loguea y se continúa (comportamiento de `sendMany`, sin cambios).
- **iHost**: Sin dependencias nuevas.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Reporte del usuario**: `/deudas_pendientes` envía "todo el historial pendiente" incluyendo cuotas de años futuros; quiere solo del pasado al mes en curso.
- **Código en main (SPEC-086)**: `formatDeudasPendientes` (`internal/services/telegram_bot.go`) aplica `dueInRange(p.DueDate, now, showMonths)` con `showMonths` default `1` (setting `telegram_bot_show_months`, SPEC-084 REQ-016): las vencidas se muestran siempre; las no vencidas solo si caen dentro de los próximos `showMonths` meses desde el mes actual. `now` se calcula con `GetTimezoneLocation` (SPEC-084, fallback UTC).
- **Formato de fechas**: `debt_bills.due_date` es TEXT `YYYY-MM-DD`.

### 3.2 Opciones evaluadas (original de esta spec)

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Filtrar en SQL con cutoff `due_date <= ?` en `ListPendingWithDetails` | Query mínima, menos memoria | Cambia la firma del método (único caller: el bot) | ~~Seleccionada~~ descartada |
| Filtrar en memoria en el handler (traer todo y descartar) | No toca storage | Carga filas innecesarias; lógica de negocio en el handler | ❌ Rechazada |
| Método nuevo `ListPendingWithDetailsUntil` conservando el actual | No rompe firma existente | Código duplicado | ❌ Rechazada |
| Límite fijo "próximos N días" desde hoy | Simple | El usuario pidió explícitamente "mes en curso" | ❌ Rechazada |

**Decisión final (SPEC-086, adoptada por esta spec)**: filtro en el **formateo** (`dueInRange` + `showMonths`), configurable por el usuario y reutilizado por servicios y deudas. Evita duplicar lógica de fecha en storage y da control del horizonte de meses.

### 3.3 Decisiones arquitectónicas

- **ADR-001**: El "mes en curso" se calcula en la **zona horaria configurada** (`GetTimezoneLocation`, SPEC-084) con fallback UTC, consistente con los schedulers (SPEC-078).
- **ADR-002 (revisado)**: La firma de `ListPendingWithDetails` **no cambia**; el filtro de rango se aplica en `formatDeudasPendientes` (SPEC-086). El cutoff SQL propuesto originalmente queda descartado como implementación redundante.

## 4. Diseño Técnico (implementado en main vía SPEC-086)

```
[Telegram Update /deudas_pendientes]
        │
        ▼
[handleDeudasPendientes]
        │  now = GetTimezoneLocation (SPEC-084) · showMonths = telegram_bot_show_months (default 1)
        ▼
[DebtBillStorage.ListPendingWithDetails(ctx)]   (sin filtro SQL)
        │  WHERE status='pending' AND deleted_at IS NULL
        ▼
[formatDeudasPendientes(pending, format, now, sepLen, showMonths)]
        │  dueInRange: vencidas SIEMPRE · no vencidas ≤ fin del mes actual + (showMonths-1) meses
        ▼
[sendMany × M]  (secuencial, MarkdownV1)
```

### 4.1 Componentes (en main, SPEC-086)

- **`formatDeudasPendientes`** (`internal/services/telegram_bot.go`): filtra con `dueInRange(p.DueDate, now, showMonths)`, itemiza por deuda con semáforo/fecha legible y agrega totales por moneda sobre la lista filtrada.
- **`dueInRange`**: vencidas siempre; no vencidas dentro del rango `showMonths` (default 1 = mes en curso).
- **`handleDeudasPendientes`**: resuelve `now` (zona del usuario) y `showMonths` desde settings.

### 4.2 Modelo de datos / APIs / Dependencias

Sin cambios de esquema, modelos ni API REST. Sin dependencias nuevas.

## 5. Criterios de Aceptación

### 5.1 Funcionales (verificados contra main, SPEC-086)

- [x] CA-001: Dado mes actual = 2026-09 y cuotas pendientes con `due_date` 2026-08-15, 2026-09-30, 2026-10-12 y 2027-03-10, cuando se ejecuta `/deudas_pendientes`, entonces el bot responde solo con las cuotas del 2026-08-15 y 2026-09-30 (pasado + mes en curso). *(`dueInRange` con showMonths=1: vencidas + mes actual)*
- [x] CA-002: Dado una cuota pendiente con `due_date` = último día del mes en curso, cuando se ejecuta el comando, entonces esa cuota SÍ se incluye (límite inclusivo). *(`currentMonthEnd(now).AddDate(0, showMonths-1, 0)`)*
- [x] CA-003: Dado `timezone = America/Managua` configurado, cuando el mes en la zona del usuario difiere del mes UTC (borde de mes), entonces el filtro se calcula con la zona del usuario. *(`GetTimezoneLocation`, SPEC-084)*
- [x] CA-004: Dado que no hay cuotas pendientes en rango (pero sí existen cuotas futuras pendientes), cuando se ejecuta `/deudas_pendientes`, entonces se envía un mensaje único claro sin totales. *(`✅ No hay deudas pendientes.`)*
- [x] CA-005: Dado pendientes filtrados en USD y NIO, cuando se genera el mensaje de totales, entonces los totales por moneda suman solo las cuotas en rango. *(`formatDeudasTotales(filtered, ...)`)*

### 5.2 No funcionales

- [x] CA-NF-001: Sin cambios de esquema SQLite ni dependencias nuevas.
- [x] CA-NF-002: Sin impacto medible de rendimiento en iHost (menos filas formateadas).

### 5.3 Testing

- **Unit tests (main)**: `telegram_bot_test.go` — `dueInRange`, `formatDeudasPendientes` con rango, caso vacío; `debt_bill_pending_test.go` — storage sin regresión.
- **E2E**: prueba manual local del bot con `/deudas_pendientes` contra DB de prueba con cuotas de varios meses.

## 6. Plan de Implementación (ejecutado)

| Fase | Descripción | Estado |
|------|-------------|--------|
| 1 | Implementación del filtro de rango en `formatDeudasPendientes` con `dueInRange` + `showMonths` (SPEC-086) | ✅ en main |
| 2 | Tests (unitarios + validación manual local) | ✅ en main |
| 3 | Release formal de esta spec (merge doc a main, estado released) | ⏳ este commit |

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| El usuario espera ver cuotas del mes siguiente "próximas a vencer" | Media | Bajo | `showMonths` es configurable (setting `telegram_bot_show_months`); default 1 = mes en curso |
| Mensaje vacío confuso si solo hay cuotas futuras | Media | Bajo | Mensaje único claro sin totales |
| Confusión con /servicios_pendientes | Baja | Bajo | Mismo criterio `dueInRange` en ambos (SPEC-084/086) |

## 8. Notas y Referencias

- SPEC-078 (zona horaria), SPEC-080 (`/deudas_pendientes`), SPEC-082 (totales por moneda), SPEC-083 (itemización de cuotas), SPEC-084 (`showMonths`, semáforo, fecha legible), SPEC-086 (formato de servicios aplicado a deudas, filtro `dueInRange`).
- `internal/services/telegram_bot.go` — `formatDeudasPendientes`, `dueInRange`, `handleDeudasPendientes`.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-19 | paulomcnally | Creación inicial de la especificación (requerimiento reportado por el usuario: limitar /deudas_pendientes al mes en curso) |
| 2026-09-19 | paulomcnally | Implementación original (descartada luego): `ListPendingWithDetails(ctx, until)` con filtro `due_date <= ?` (storage), helper `currentMonthEnd` (zona del usuario vía `currentUserNow`, fallback UTC) en `handleDeudasPendientes`. Estado → in_progress |
| 2026-09-19 | paulomcnally | **Cancelada (error de proceso, revertida)**: el requerimiento fue absorbido por SPEC-086. La cancelación fue un error: "cerrar" una spec jamás implica cancelarla (ver AGENTS.md). La implementación quedó descartada |
| 2026-09-19 | paulomcnally | **Reabierta** por decisión del usuario: el requerimiento está implementado y liberado en main por SPEC-086 (`dueInRange` + `showMonths` default 1 aplicado en `formatDeudasPendientes`). Criterios de aceptación verificados contra main. Estado → pending_release |