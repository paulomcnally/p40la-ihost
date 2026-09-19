---
title: "Bot Telegram: limitar /deudas_pendientes al mes en curso"
id: "SPEC-085"
status: "cancelled"
author: "paulomcnally"
created: "2026-09-19"
updated: "2026-09-19"
github_issue: 88
---

# Bot Telegram: limitar /deudas_pendientes al mes en curso

**ID**: SPEC-085  
**Estado**: cancelled  
**Autor**: paulomcnally  
**Creado**: 2026-09-19  
**Actualizado**: 2026-09-19

---

## 1. Resumen Ejecutivo

El comando `/deudas_pendientes` del bot de Telegram (SPEC-080/082/083) consulta `DebtBillStorage.ListPendingWithDetails`, que devuelve **todas** las cuotas con `status='pending'` sin importar su fecha de vencimiento. En deudas de largo plazo (ej: Hipoteca con cuotas hasta 2033) esto genera una lista enorme de cuotas **futuras** (de años venideros) que no aportan nada hoy: al usuario le interesa saber qué tiene pendiente **del pasado hasta el mes en curso**.

El usuario reporta: *"cuando consulto las deudas pendientes me envía el dato de todo el historial pendiente, eso no está bien, tiene que enviarme del pasado al presente, no me interesa saber deudas de un año futuro. Tienes que limitarte al mes en curso."*

La solución es filtrar en la query del storage: solo cuotas pendientes con `due_date <= último día del mes en curso`. El límite del mes se calcula en la **zona horaria configurada** (setting `timezone`, SPEC-078) con fallback UTC, reutilizando el helper `currentUserNow`. El formateo de mensajes (SPEC-083) no cambia: recibe menos datos y los muestra igual.

Consideraciones iHost: cambios acotados a `internal/storage/debt_bill.go`, `internal/services/telegram_bot.go` y tests. Sin cambios de esquema SQLite, API, frontend ni dependencias nuevas. El filtro SQL reduce además la memoria usada por el bot (menos filas escaneadas), alineado con el perfil de recursos limitados del iHost.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: `/deudas_pendientes` devuelve **solo** cuotas pendientes (`status='pending'`, `deleted_at IS NULL`) cuyo `due_date` sea menor o igual al **último día del mes en curso** (del pasado al presente). Las cuotas pendientes con vencimiento en meses futuros quedan excluidas.
2. **REQ-002**: El límite "último día del mes en curso" se calcula usando la **zona horaria configurada** del sistema (setting `timezone`, SPEC-078) vía el helper existente `currentUserNow` (`internal/services/scheduler_hour.go`). Si la zona no está configurada o es inválida, se usa UTC (fallback, mismo comportamiento que los schedulers).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-003**: El mensaje final de **totales por moneda** (SPEC-082 REQ-004) se calcula sobre la lista ya filtrada (solo mes en curso), sin cambios de código: refleja automáticamente las cuotas del mes.
2. **REQ-004**: Si no hay cuotas pendientes dentro del mes en curso (aunque existan cuotas futuras), se envía un mensaje único que aclare el alcance: `✅ No hay deudas pendientes en el mes en curso.` (reemplaza el mensaje genérico actual para evitar confusión).
3. **REQ-005**: Tests actualizados: storage (`ListPendingWithDetails` con cutoff) y validación manual local del bot con DB de prueba.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-006**: Evaluar si el mismo criterio aplica a `/servicios_pendientes` (facturas). **Fuera de alcance** de esta spec: el usuario reportó solo el caso de deudas; se documenta para una futura spec.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Query con filtro por `due_date <= ?` sobre índice existente de la tabla `debt_bills`; sin impacto medible en iHost. Menos filas procesadas → menos memoria en el formateo.
- **Seguridad**: Sin cambios de tokens, chat IDs ni secretos.
- **Almacenamiento**: Sin archivos nuevos ni cambios de esquema DB.
- **Disponibilidad**: Si falla el envío de un mensaje intermedio, se loguea y se continúa (comportamiento de `sendMany`, sin cambios).
- **iHost**: Sin dependencias nuevas. `time.Date` y `time.LoadLocation` ya están en uso (SPEC-078).

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Reporte del usuario**: `/deudas_pendientes` envía "todo el historial pendiente" incluyendo cuotas de años futuros; quiere solo del pasado al mes en curso.
- **Código actual**: `DebtBillStorage.ListPendingWithDetails` (`internal/storage/debt_bill.go:109`) filtra únicamente `db.status = 'pending' AND db.deleted_at IS NULL`, ordenado por `due_date ASC`. Es usado **solo** por el bot (`handleDeudasPendientes`, `internal/services/telegram_bot.go:231`) y su test (`internal/storage/debt_bill_pending_test.go`). El equivalente de facturas (`BillStorage.ListPendingWithDetails`) también lo usa `bill_summary_scheduler` (email diario) — por eso el cambio se acota a cuotas y no se toca facturas.
- **Formato de fechas**: `debt_bills.due_date` es TEXT `YYYY-MM-DD` (patrón usado en `ListByMonth` con `substr(due_date,1,7)`), por lo que la comparación lexicográfica `due_date <= 'YYYY-MM-DD'` es correcta.
- **Zona horaria**: El sistema ya tiene el setting `timezone` (SPEC-078) y el helper `currentUserNow(ctx, settings)` que devuelve `time.Now()` en la zona configurada (fallback UTC).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Filtrar en SQL con cutoff `due_date <= ?` en `ListPendingWithDetails` | Query mínima, consistente con arquitectura (queries en storage), menos memoria | Cambia la firma del método (único caller: el bot) | ✅ Seleccionada |
| Filtrar en memoria en el handler (traer todo y descartar) | No toca storage | Carga filas innecesarias (años futuros) en el iHost; lógica de negocio en el handler (viola reglas de capas) | ❌ Rechazada |
| Método nuevo `ListPendingWithDetailsUntil` conservando el actual | No rompe firma existente | Código duplicado; el método sin cutoff no se usa (solo el bot lo llama) | ❌ Rechazada |
| Límite fijo "próximos N días" desde hoy | Simple | El usuario pidió explícitamente "mes en curso" | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Cutoff = último día del mes en curso calculado con la zona horaria del usuario
- **Contexto**: El "mes en curso" depende de la zona horaria: en el límite de mes, UTC puede estar en un mes distinto al local del usuario.
- **Decisión**: Calcular `now = currentUserNow(ctx, settings)` (zona configurada, fallback UTC), luego `lastDay = time.Date(now.Year(), now.Month()+1, 1, ...).AddDate(0,0,-1)` y formatear `YYYY-MM-DD` como cutoff inclusivo. `time.Date` maneja el overflow de diciembre → enero del año siguiente.
- **Consecuencias**: Consistente con los schedulers (SPEC-078); sin lógica duplicada.

**ADR-002**: Cambiar la firma de `ListPendingWithDetails` (agregar `until string`) en lugar de crear un método nuevo
- **Contexto**: El método tiene un único consumidor (el bot) y un test.
- **Decisión**: `ListPendingWithDetails(ctx, until string)` con `AND db.due_date <= ?` en el WHERE. El parámetro es la fecha límite `YYYY-MM-DD` (inclusiva).
- **Consecuencias**: API de storage mínima y sin código muerto; el test existente se actualiza.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Telegram Update /deudas_pendientes]
        │
        ▼
[handleDeudasPendientes]
        │  cutoff = último día del mes en curso (currentUserNow, SPEC-078)
        ▼
[DebtBillStorage.ListPendingWithDetails(ctx, cutoff)]
        │  WHERE status='pending' AND deleted_at IS NULL AND due_date <= cutoff
        ▼
[formatDeudasPendientes]  ─► []string (sin cambios, SPEC-083)
        │
        ▼
[sendMany × M]  (secuencial, MarkdownV1)
```

### 4.2 Componentes

#### 4.2.1 `DebtBillStorage.ListPendingWithDetails` (`internal/storage/debt_bill.go`)
- **Responsabilidad**: Devolver cuotas pendientes con contexto (deuda, institución, moneda) **hasta** una fecha límite inclusiva.
- **Interfaz**: `ListPendingWithDetails(ctx context.Context, until string) ([]models.PendingDebtDetail, error)` — `until` en formato `YYYY-MM-DD`.
- **Dependencias**: `models.PendingDebtDetail` (sin cambios).
- **Cambio SQL**: agregar `AND db.due_date <= ?` al WHERE existente.

#### 4.2.2 `TelegramBotService.handleDeudasPendientes` (`internal/services/telegram_bot.go`)
- **Responsabilidad**: Calcular el cutoff del mes en curso y pasarlo al storage.
- **Interfaz**: sin cambios externos.
- **Cambio**: 
  ```go
  now, err := currentUserNow(ctx, s.settings)
  if err != nil {
      slog.Warn("telegram_bot: zona horaria inválida, usando UTC para el mes en curso", "error", err)
  }
  lastDay := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location()).AddDate(0, 0, -1)
  cutoff := lastDay.Format("2006-01-02")
  pending, err := s.debtBills.ListPendingWithDetails(ctx, cutoff)
  ```
- **REQ-004**: en `formatDeudasPendientes`, el caso vacío devuelve `✅ No hay deudas pendientes en el mes en curso.`

### 4.3 Modelo de datos

Sin cambios de esquema ni de modelos. `debt_bills.due_date` (TEXT `YYYY-MM-DD`) se compara lexicográficamente contra el cutoff.

### 4.4 APIs / Contratos

Sin cambios de API REST ni de frontend. Contrato interno del storage:

```go
// Antes
ListPendingWithDetails(ctx context.Context) ([]models.PendingDebtDetail, error)
// Después
ListPendingWithDetails(ctx context.Context, until string) ([]models.PendingDebtDetail, error)
```

### 4.5 Dependencias

- **Internas**: `internal/storage/debt_bill.go`, `internal/services/telegram_bot.go`, `internal/services/telegram_bot_test.go`, `internal/storage/debt_bill_pending_test.go`.
- **Externas**: Ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado un mes actual = 2026-09 y cuotas pendientes con `due_date` 2026-08-15, 2026-09-30, 2026-10-12 y 2027-03-10, cuando se ejecuta `/deudas_pendientes`, entonces el bot responde solo con las cuotas del 2026-08-15 y 2026-09-30 (pasado + mes en curso).
- [ ] CA-002: Dado una cuota pendiente con `due_date` = último día del mes en curso, cuando se ejecuta el comando, entonces esa cuota SÍ se incluye (límite inclusivo).
- [ ] CA-003: Dado `timezone = America/Managua` configurado, cuando el mes en la zona del usuario difiere del mes UTC (borde de mes), entonces el cutoff se calcula con la zona del usuario.
- [ ] CA-004: Dado que no hay cuotas pendientes con `due_date <= cutoff` (pero sí existen cuotas futuras pendientes), cuando se ejecuta `/deudas_pendientes`, entonces se envía `✅ No hay deudas pendientes en el mes en curso.` (sin mensaje de totales).
- [ ] CA-005: Dado pendientes filtrados en USD y NIO, cuando se genera el mensaje de totales, entonces los totales por moneda suman solo las cuotas del mes en curso.

### 5.2 No funcionales

- [ ] CA-NF-001: Sin cambios de esquema SQLite ni dependencias nuevas.
- [ ] CA-NF-002: La query filtrada responde en < 10ms con 1000 cuotas (mismo perfil que la actual).

### 5.3 Testing

- **Unit tests**: `debt_bill_pending_test.go` — cutoff excluye cuotas futuras, incluye la fecha límite; `telegram_bot_test.go` — caso vacío con mensaje nuevo.
- **Integration tests**: `ListPendingWithDetails(ctx, cutoff)` con DB de test (`:memory:` + migraciones), verificando el WHERE con `due_date <= cutoff`.
- **E2E tests**: Prueba manual local del bot con `/deudas_pendientes` contra una DB de prueba que contenga cuotas de varios meses (pasado, mes actual, futuras).
- **Carga/Performance**: Sin métricas nuevas; menos filas que antes.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Storage: agregar parámetro `until` a `ListPendingWithDetails` + filtro `due_date <= ?` + actualizar test de storage | 0.25 días | Ninguna |
| 2 | Bot: cálculo del cutoff del mes en curso con `currentUserNow` + mensaje vacío con alcance del mes | 0.25 días | Fase 1 |
| 3 | Tests (unitarios + manual local) y validación con el usuario | 0.5 días | Fase 2 |

### 6.2 Milestones

1. **MVP**: `/deudas_pendientes` responde solo cuotas hasta el último día del mes en curso (zona del usuario).
2. **V1.0**: Tests actualizados y validación manual con el usuario.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Corte incorrecto en borde de mes por zona horaria | Baja | Medio | Cutoff con `currentUserNow` (SPEC-078); test CA-003 |
| El usuario espera ver cuotas del mes siguiente "próximas a vencer" | Media | Bajo | Requerimiento explícito del usuario: "del pasado al presente, mes en curso"; si cambia de opinión se ajusta la spec |
| Mensaje vacío confuso si solo hay cuotas futuras | Media | Bajo | REQ-004: mensaje aclara el alcance ("en el mes en curso") |
| Confusión con /servicios_pendientes (sin filtrar) | Baja | Bajo | Documentado como fuera de alcance (REQ-006) |

## 8. Notas y Referencias

- SPEC-078 (zona horaria y `currentUserNow`), SPEC-080 (`/deudas_pendientes`), SPEC-082 (totales por moneda), SPEC-083 (itemización de cuotas).
- `internal/storage/debt_bill.go:109` — query actual sin filtro de fecha.
- `internal/services/telegram_bot.go:226-245` — `handleDeudasPendientes` (único caller del método).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-19 | paulomcnally | Creación inicial de la especificación (requerimiento reportado por el usuario: limitar /deudas_pendientes al mes en curso) |
| 2026-09-19 | paulomcnally | Implementación: `ListPendingWithDetails(ctx, until)` con filtro `due_date <= ?` (storage), helper `currentMonthEnd` (zona del usuario vía `currentUserNow`, fallback UTC) en `handleDeudasPendientes`, mensaje vacío "No hay deudas pendientes en el mes en curso". Tests: cutoff en storage, `TestCurrentMonthEnd`, E2E con `ProcessUpdate` + seed de cuotas pasado/actual/futuro. Estado → in_progress |
| 2026-09-19 | paulomcnally | **Cancelada**: el requerimiento (limitar /deudas_pendientes al mes en curso) fue absorbido por SPEC-086, que aplica el filtro `telegram_bot_show_months` (vencidas siempre + próximos N meses; default 1 = mes en curso) con el formato completo de servicios. La implementación de esta spec (cutoff fijo en storage) queda descartada. Estado → cancelled |
