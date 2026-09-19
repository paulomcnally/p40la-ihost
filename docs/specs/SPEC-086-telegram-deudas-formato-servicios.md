---
title: "Bot Telegram: /deudas_pendientes con el mismo formato de /servicios_pendientes"
id: "SPEC-086"
status: "pending_release"
author: "paulomcnally"
created: "2026-09-19"
updated: "2026-09-19"
github_issue: 89
---

# Bot Telegram: /deudas_pendientes con el mismo formato de /servicios_pendientes

**ID**: SPEC-086  
**Estado**: pending_release  
**Autor**: paulomcnally  
**Creado**: 2026-09-19  
**Actualizado**: 2026-09-19

---

## 1. Resumen Ejecutivo

SPEC-084 definió el formato rico de `/servicios_pendientes`: cada factura itemizada con **semáforo de vencimiento 🟢/🟡/🔴** (regla SPEC-081 REQ-008), fecha legible ("15 jul 2026"), texto de días restantes ("Vence en N días" / "Vencida hace N días"), **línea en blanco entre registros**, orden por urgencia, separador configurable (`telegram_bot_separator_length`) y **negritas** en el resumen (`*Facturas*` / `*Pendiente*`). El comando `/deudas_pendientes` sigue usando el formato simple de SPEC-083 (solo `📅 YYYY-MM-DD — monto`, sin semáforo ni separación).

El usuario reporta (evaluación manual): *"me refiero al semáforo de emoji, el espacio entre cada registro, las negritas en los textos 'pendientes' etc"* — es decir, **aplicar exactamente el mismo formato de servicios a las cuotas pendientes**.

Esta spec replica el formato de SPEC-084 en `/deudas_pendientes` reutilizando los helpers ya existentes (`billStatus`, `daysUntilDue`, `dayWord`, `chunkBills`, `sortBillsByDue`, `currentMonthEnd`) y los settings (`telegram_bot_separator_length`, `telegram_bot_show_months`, zona horaria SPEC-078). Además, el filtro de cuotas pasa a usar la misma regla que servicios (`showMonths`, vencidas siempre), que con el default (1) equivale al filtro "mes en curso" de SPEC-085 (vencidas + mes actual), por lo que esta spec **absorbe el requerimiento de SPEC-085**.

Consideraciones iHost: cambios solo en `internal/services/telegram_bot.go` (formateo + handler) y tests. Sin cambios de DB, modelos, API ni dependencias nuevas. Los datos ya están disponibles: `PendingDebtDetail` tiene `DueDate` y `Amount` por cuota.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: `/deudas_pendientes` itemiza cada cuota pendiente con el **semáforo de vencimiento** (regla SPEC-081 REQ-008): 🟢 si faltan >10 días, 🟡 entre 1 y 10 días, 🔴 vence hoy o vencida, con texto "Vence en N días" / "Vence hoy" / "Vencida hace N días" (singular/plural correcto).
2. **REQ-002**: La fecha de vencimiento de cada cuota se muestra en **formato legible** (`15 jul 2026`, meses en español abreviados, reutilizando `esMonthAbbr`).
3. **REQ-003**: Entre cada cuota itemizada hay una **línea en blanco** (mismo espaciado que servicios).
4. **REQ-004**: Las cuotas se ordenan **por urgencia**: más vencida primero, por fecha de vencimiento ascendente (reutilizando `sortBillsByDue`).
5. **REQ-005**: El resumen por deuda mantiene **negritas** MarkdownV1: `*Cuotas*: N` y `*Pendiente*: <monto>`, con **separador configurable** (`telegram_bot_separator_length`, default 20, 0 = sin separador) antes del resumen (mismo patrón de servicios).
6. **REQ-006**: El filtro de cuotas usa la **misma regla que servicios** (`telegram_bot_show_months`, default 1): las **vencidas se muestran siempre**; las no vencidas solo si su `due_date` cae dentro de los próximos N meses desde el mes actual. Con N=1 (default) equivale a "vencidas + mes en curso" (comportamiento de SPEC-085).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-007**: Se mantiene el encabezado de la deuda (💳 nombre + institución) en el primer bloque y el resumen al final del último bloque cuando hay particionado (patrón SPEC-083 ADR-001/002, constante de 25 cuotas por bloque). La etiqueta `Institución` va en negrita (`*Institución*: <nombre>`), consistente con `*Cuotas*`/`*Pendiente*`.
2. **REQ-008**: Se mantiene el mensaje final de **totales por moneda** (SPEC-082 REQ-004), calculado sobre la lista ya filtrada.
3. **REQ-009**: Si no hay cuotas pendientes dentro del filtro, se envía un único mensaje `✅ No hay deudas pendientes.` (sin totales).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-010**: Tests unitarios de los nuevos formateadores: semáforo correcto, fecha legible, espaciado, negritas, separador, orden por urgencia y filtro con `showMonths`.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Mismo patrón de envío secuencial (SPEC-082); bloques de 25 cuotas. Sin impacto en iHost.
- **Seguridad**: Sin cambios de tokens, chat IDs ni secretos.
- **Almacenamiento**: Sin archivos nuevos ni cambios de esquema DB.
- **Disponibilidad**: Si falla el envío de un mensaje intermedio, se loguea y se continúa (`sendMany`).
- **iHost**: Sin dependencias nuevas.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Reporte del usuario**: el mensaje de `/deudas_pendientes` no tiene el formato de `/servicios_pendientes`: falta semáforo de emoji, espacio entre registros y negritas.
- **Causa raíz**: el worktree donde se probó el bot se creó antes del merge de SPEC-084 a `main`; además SPEC-085 solo aplicó el filtro del mes en curso sin tocar el formateo (heredado de SPEC-083).
- **Código actual**: `formatDeudasPendientes` (telegram_bot.go:615) itemiza con `📅 YYYY-MM-DD — monto` sin semáforo ni espaciado; `formatDebtGroup` (647) cierra con `*Cuotas*`/`*Pendiente*` ya en negrita (SPEC-084 REQ-015) pero sin separador configurable. SPEC-084 dejó helpers reutilizables: `billStatus`, `daysUntilDue`, `dayWord`, `billDueDate`, `currentMonthEnd`, `isCurrentPeriod`, `sortBillsByDue`, `chunkBills`, `esMonthAbbr`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Reutilizar helpers de SPEC-084 (`pendingBill`, `billStatus`, `sortBillsByDue`, `chunkBills`) para cuotas | Código único, consistencia garantizada con servicios | `pendingBill` está orientado a facturas; `isCurrentPeriod` recibe `PendingBillDetail` | ✅ Seleccionada |
| Duplicar la lógica de semáforo/fecha para deudas | Sin tocar código de servicios | Código duplicado, riesgo de divergencia | ❌ Rechazada |
| Cambiar `isCurrentPeriod` para aceptar cuotas | Reutilización total | Firma acoplada a dos modelos | ❌ Rechazada (se extrae helper sobre `dueDate string`) |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Unificar el tipo de ítem: las cuotas de `debtGroup` pasan a usar `pendingBill` (label/status/days/hasDue/amount), renombrando conceptualmente a "ítem itemizado" compartido
- **Contexto**: Las cuotas siempre tienen `due_date` (a diferencia de las facturas), pero el resto del formateo es idéntico.
- **Decisión**: `debtGroup.bills []pendingBill` (antes `installments []pendingInstallment`), reutilizando `sortBillsByDue`, `chunkBills` y `formatDebtGroup` con el mismo layout de servicios. El struct `pendingInstallment` y `chunkInstallments` se eliminan.
- **Consecuencias**: Una sola implementación del layout; `formatDebtGroup` difiere solo en encabezado (💳 deuda/institución) y resumen (`*Cuotas*` vs `*Facturas*`).

**ADR-002**: Filtro compartido por fecha (string) en lugar de por modelo
- **Contexto**: `isCurrentPeriod` recibe `PendingBillDetail`; las cuotas son `PendingDebtDetail` con `DueDate string` (no puntero).
- **Decisión**: Extraer `dueInRange(dueDate string, now time.Time, showMonths int) bool` (misma regla: vencidas siempre; no vencidas dentro de los próximos N meses; sin fecha → siempre) y usarla desde ambos formateadores.
- **Consecuencias**: Regla única y testable; SPEC-085 queda absorbida (su cutoff fijo es el caso showMonths=1).

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Telegram Update /deudas_pendientes]
        │
        ▼
[handleDeudasPendientes]
        │  now (zona SPEC-078), sepLen, showMonths (settings)
        ▼
[ListPendingWithDetails()]  ──► []PendingDebtDetail (sin cambios)
        ▼
[formatDeudasPendientes(pending, format, now, sepLen, showMonths)]
        │  filtro dueInRange + itemización con pendingBill (semáforo, fecha legible)
        ▼
[formatDebtGroup × N]  (mismo layout que formatServiceGroup, 💳 + *Cuotas*)
        ▼
[sendMany × M]  (secuencial, MarkdownV1)
```

### 4.2 Componentes

#### 4.2.1 `TelegramBotService.handleDeudasPendientes` (`internal/services/telegram_bot.go`)
- **Responsabilidad**: Resolver `now` (zona configurada, fallback UTC), `sepLen` (`telegram_bot_separator_length`) y `showMonths` (`telegram_bot_show_months`) — idéntico a `handleServiciosPendientes` — y pasarlos al formateador.
- **Interfaz**: sin cambios externos.

#### 4.2.2 `formatDeudasPendientes` / `formatDebtGroup`
- **Firma**: `formatDeudasPendientes(pending []appmodels.PendingDebtDetail, format CurrencyFormat, now time.Time, sepLen, showMonths int) []string`.
- **Layout por cuota** (idéntico a servicios):
```
💳 *Préstamo LAFISE*
  Institución: Banco LAFISE

  🔴 Vencida hace 66 días
  📅 15 jul 2026 — C$1,666.67

  🟡 Vence en 5 días
  📅 15 sep 2026 — C$1,666.67

  --------------------
  *Cuotas*: 2
  *Pendiente*: C$3,333.34
```

### 4.3 Modelo de datos

Sin cambios de esquema ni de modelos.

### 4.4 APIs / Contratos

Sin cambios de API REST ni de frontend. Cambios internos:
- `debtGroup.installments []pendingInstallment` → `debtGroup.bills []pendingBill`
- Eliminar `pendingInstallment` y `chunkInstallments` (reemplazados por `pendingBill` y `chunkBills`)
- Nuevo helper `dueInRange(dueDate string, now time.Time, showMonths int) bool`; `isCurrentPeriod` lo delega.

### 4.5 Dependencias

- **Internas**: `internal/services/telegram_bot.go`, `internal/services/telegram_bot_test.go`.
- **Externas**: Ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Dado `/deudas_pendientes` con cuotas vencida (hace N días), del mes (🟡/🟢) y futura fuera de rango, cuando se ejecuta el comando, entonces se muestran la vencida y la del mes con semáforo y fecha legible; la futura no aparece ni en el conteo ni en el total.
- [x] CA-002: Dado el mensaje de una deuda, cuando se formatea, entonces cada cuota tiene semáforo correcto (regla SPEC-081 REQ-008) y texto "Vence en N días"/"Vence hoy"/"Vencida hace N días" (singular/plural).
- [x] CA-003: Dado el mensaje de una deuda, cuando se formatea, entonces hay una línea en blanco entre cuota y cuota, la fecha es legible ("15 jul 2026") y el resumen cierra con separador (guiones ASCII), `*Cuotas*` y `*Pendiente*` en negrita.
- [x] CA-004: Dado un `telegram_bot_separator_length = 30`, cuando se ejecuta el comando, entonces el separador tiene 30 guiones; con 0 no se muestra; con vacío/sin config se usa 20.
- [x] CA-005: Dado `telegram_bot_show_months = 2`, cuando se ejecuta el comando, entonces además de vencidas y mes actual se muestran las cuotas del próximo mes; las vencidas de cualquier antigüedad se muestran siempre.
- [x] CA-006: Dado que no hay cuotas en rango, cuando se ejecuta el comando, entonces se envía `✅ No hay deudas pendientes.` (sin totales).
- [x] CA-007: Dado pendientes en USD y NIO, cuando se genera el mensaje de totales, entonces los totales por moneda suman solo las cuotas en rango.

### 5.2 No funcionales

- [x] CA-NF-001: Ningún mensaje generado supera los 4096 caracteres (bloque de 25 cuotas).
- [x] CA-NF-002: Sin cambios de esquema SQLite ni dependencias nuevas.

### 5.3 Testing

- **Unit tests**: `formatDeudasPendientes` con la nueva firma: semáforo (vencida/mes/futura), espaciado, negritas, separador configurable, orden por urgencia, particionado, caso vacío, multi-moneda, `dueInRange` con showMonths 1 y 2.
- **Integration tests**: Ninguno nuevo (sin cambios de API).
- **E2E tests**: Prueba manual local con el bot: `/deudas_pendientes` contra la DB local de prueba y verificación visual del formato (semáforo, espaciado, negritas).
- **Carga/Performance**: Sin métricas nuevas.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Handler: resolver now/sepLen/showMonths y pasar al formateador | 0.25 días | Ninguna |
| 2 | Refactor: `debtGroup.bills []pendingBill`, eliminar `pendingInstallment`/`chunkInstallments`, helper `dueInRange`, reescribir `formatDeudasPendientes`/`formatDebtGroup` con layout de servicios | 0.5 días | Fase 1 |
| 3 | Tests unitarios + validación manual local con el bot | 0.5 días | Fase 2 |

### 6.2 Milestones

1. **MVP**: `/deudas_pendientes` con semáforo, espaciado, fecha legible, separador y negritas.
2. **V1.0**: Tests actualizados y validación manual con el usuario.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Regresión en servicios por el refactor compartido | Baja | Medio | Tests existentes de servicios se mantienen; helpers sin cambio de comportamiento |
| Markdown con emoji/asteriscos se rompe en Telegram | Baja | Bajo | Mismo patrón ya validado en SPEC-084 (servicios) |
| El usuario espera el filtro exacto de SPEC-085 y showMonths>1 cambia el alcance | Baja | Bajo | Default showMonths=1 = comportamiento de SPEC-085; configurable en Settings |

## 8. Notas y Referencias

- SPEC-084 (formato de servicios a replicar), SPEC-083 (itemización de cuotas), SPEC-085 (filtro mes en curso, absorbido), SPEC-082 (totales por moneda), SPEC-081 REQ-008 (regla del semáforo), SPEC-078 (zona horaria), SPEC-058 (formato de moneda).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-19 | paulomcnally | Creación inicial de la especificación (requerimiento del usuario: copiar el formato de /servicios_pendientes a /deudas_pendientes — semáforo, espaciado, negritas) |
| 2026-09-19 | paulomcnally | Implementación: handler resuelve now/sepLen/showMonths; `debtGroup.bills []pendingBill` (reemplaza `pendingInstallment`/`chunkInstallments` por `chunkBills`/`sortBillsByDue`), `formatDebtGroup` con layout de servicios (semáforo + fecha legible + línea en blanco + separador + negritas), helper compartido `dueInRange` (isCurrentPeriod lo delega). Tests actualizados (semáforo, espaciado, negritas, separador, filtro, showMonths, particionado). Estado → in_progress |
| 2026-09-19 | paulomcnally | Cambio iterativo solicitado por el usuario en evaluación manual: etiqueta `*Institución*` en negrita en el encabezado de cada deuda (REQ-007). Test actualizado. |
| 2026-09-19 | paulomcnally | Validación manual con el usuario: ✅ todo funciona correctamente. Todos los criterios de aceptación en pass. Estado → pending_release |
