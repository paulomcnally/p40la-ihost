---
title: "Bot Telegram: itemizar cuotas pendientes en /deudas_pendientes"
id: "SPEC-083"
status: "in_progress"
author: "paulomcnally"
created: "2026-09-19"
updated: "2026-09-19"
github_issue: 86
---

# Bot Telegram: itemizar cuotas pendientes en /deudas_pendientes

**ID**: SPEC-083  
**Estado**: in_progress  
**Autor**: paulomcnally  
**Creado**: 2026-09-19  
**Actualizado**: 2026-09-19

---

## 1. Resumen Ejecutivo

El comando `/deudas_pendientes` del bot de Telegram (SPEC-080 + SPEC-082) responde con **un mensaje por deuda** mostrando la cantidad de cuotas pendientes y un monto único "Pendiente", que es la **suma de las cuotas pendientes** (cada cuota incluye intereses futuros). En deudas con muchas cuotas restantes (ej: Hipoteca con 197 cuotas de C$620) ese monto supera el total de la deuda (C$122,140 vs C$58,050) y el usuario lo percibe como "el gran total de la deuda" en lugar de lo que está pendiente.

El usuario confirmó el comportamiento esperado: **itemizar cada cuota pendiente** (fecha de vencimiento + monto) por deuda, y al final el total. El mensaje de totales por moneda (SPEC-082 REQ-004) se mantiene.

Consideraciones iHost: cambios solo en `internal/services/telegram_bot.go` (formateo) y tests. Los datos ya están disponibles: `PendingDebtDetail` trae `DueDate` y `Amount` por cuota (agregado `CurrencyCode` en SPEC-082). Sin cambios de DB, API ni dependencias. Atención al límite de 4096 caracteres de Telegram: deudas con muchas cuotas pendientes requieren particionar el mensaje en bloques.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: `/deudas_pendientes` muestra, por cada deuda con cuotas pendientes, **cada cuota pendiente itemizada** con su fecha de vencimiento (`DueDate`) y monto, en el orden del query (por `due_date` ascendente).
2. **REQ-002**: Si la lista de cuotas de una deuda excede el límite de 4096 caracteres de Telegram, el mensaje de esa deuda se **particiona en varios mensajes** (bloques de ~25 cuotas por mensaje). El encabezado de la deuda solo va en el primer bloque; el último bloque cierra con el resumen.
3. **REQ-003**: Cada deuda termina con su resumen: cantidad de cuotas pendientes y monto total de las cuotas pendientes de esa deuda (formato `Cuotas: N` / `Pendiente: <monto>`).
4. **REQ-004**: Se mantiene el mensaje final de **totales por moneda** (SPEC-082): suma de cuotas pendientes agrupada por código de moneda.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-005**: Si no hay cuotas pendientes, se envía un único mensaje `✅ No hay deudas pendientes.` (sin totales).
2. **REQ-006**: Se mantiene el orden de deudas por monto total descendente (`sortDebtGroups`) y el formato de moneda configurado (SPEC-058).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-007**: Tests unitarios de los nuevos formateadores: itemización, particionado (>25 cuotas), caso vacío y multi-moneda.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Mismo patrón de envío secuencial (SPEC-082). Una deuda con N cuotas genera ⌈N/25⌉ mensajes; con volúmenes reales (máx ~197 cuotas) son ≤8 mensajes por deuda. Sin impacto en iHost.
- **Seguridad**: Sin cambios de tokens, chat IDs ni secretos.
- **Almacenamiento**: Sin archivos nuevos ni cambios de esquema DB.
- **Disponibilidad**: Si falla el envío de un mensaje intermedio, se loguea y se continúa (comportamiento de `sendMany`).
- **iHost**: Sin dependencias nuevas.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Reporte del usuario**: `/deudas_pendientes` "manda como que gran total de deuda, no las que están pendientes".
- **Datos de producción (iHost)**: verificado vía SSH. Ej: Hipoteca (total C$58,050, 240 cuotas de C$620) → 197 cuotas pendientes suman C$122,140; Toyota Hilux (total C$35,550, 96 cuotas de C$570) → 96 pendientes suman C$54,720. La cuota mensual incluye intereses, por eso la suma de cuotas pendientes supera el `total` de la deuda.
- **Código actual**: `formatDeudasPendientes` (SPEC-082) agrupa por deuda y muestra solo `Cuotas: N` + `Pendiente: <suma>`. `PendingDebtDetail` ya incluye `DueDate` y `Amount` por cuota; el query `ListPendingWithDetails` ordena por `due_date ASC`.
- **Límite de Telegram**: 4096 caracteres por mensaje. La Hipoteca con 197 cuotas itemizadas (línea `📅 2026-09-12 — C$620.00` ≈ 26 chars) superaría el límite en un único mensaje → particionado obligatorio.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Itemizar cuotas con particionado en bloques de 25 | Legible, respeta límite 4096, cubre deudas grandes | Múltiples mensajes por deuda con muchas cuotas | ✅ Seleccionada |
| Itemizar todas las cuotas en un solo mensaje | Una llamada por deuda | Supera 4096 chars en deudas grandes → Telegram rechaza el mensaje | ❌ Rechazada |
| Mostrar solo el saldo restante (total × pendientes/totales) | Un número "coherente" con el total | El usuario eligió itemizar las cuotas, no un saldo | ❌ Rechazada (decisión del usuario) |
| Un mensaje por cuota individual | Máxima legibilidad | Spam: 197 mensajes para una deuda | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Particionado por cantidad de cuotas (25 por bloque) en lugar de por longitud de caracteres
- **Contexto**: El límite real es 4096 chars, pero medir caracteres por mensaje agrega complejidad (conteo variable por símbolo/separador).
- **Decisión**: Fijar 25 cuotas por mensaje como constante. 25 líneas × ~30 chars ≈ 750 chars, muy por debajo del límite y cómodo de leer en móvil.
- **Consecuencias**: Mensajes predecibles y testables; una deuda con 197 cuotas genera 8 mensajes.

**ADR-002**: Encabezado solo en el primer bloque y resumen al final del último
- **Contexto**: Repetir el encabezado en cada bloque es ruido visual; el resumen en cada bloque duplica información.
- **Decisión**: El primer bloque de una deuda lleva `💳 *<deuda>*` + institución; los bloques intermedios son solo la lista de cuotas; el último bloque cierra con `Cuotas: N` / `Pendiente: <total>`.
- **Consecuencias**: Lectura fluida y sin repeticiones; el resumen siempre al final de la deuda.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Telegram Update /deudas_pendientes]
        │
        ▼
[handleDeudasPendientes]  (sin cambios de lógica)
        │  DebtBillStorage.ListPendingWithDetails()  (DueDate, Amount por cuota)
        ▼
[formatDeudasPendientes]  ─► []string: un mensaje por deuda (particionado) + 1 totales
        │
        ▼
[sendMany × M]  (secuencial, MarkdownV1)
```

### 4.2 Componentes

#### 4.2.1 `TelegramBotService` / `formatDeudasPendientes` (`internal/services/telegram_bot.go`)
- **Responsabilidad**: Construir la lista de mensajes de `/deudas_pendientes`.
- **Interfaz**: `formatDeudasPendientes(pending []appmodels.PendingDebtDetail, format CurrencyFormat) []string` (firma actual de SPEC-082, sin cambios externos).
- **Dependencias**: `PendingDebtDetail` (con `DueDate`, `Amount`, `CurrencyCode`).
- **Ubicación**: `internal/services/telegram_bot.go`.

#### 4.2.2 `debtGroup` (extendido)
- Agrega `installments []pendingInstallment` con `{dueDate string, amount float64}` por cuota, para itemizar.

### 4.3 Modelo de datos

Sin cambios de esquema ni de modelos. `PendingDebtDetail` ya expone:

```
PendingDebtDetail {
  DebtID, DebtDescription, InstitutionName, DueDate, Amount, CurrencySymbol, CurrencyCode
}
```

### 4.4 APIs / Contratos

Sin cambios de API REST ni de frontend. Contrato interno de mensajes:

```
Mensaje por deuda (primer bloque):
💳 *Hipoteca*
  Institución: <nombre>
  📅 2026-09-12 — C$620.00
  📅 2026-10-12 — C$620.00
  ... (hasta 25 cuotas)

Mensaje final de la deuda (último bloque):
  📅 2033-05-12 — C$620.00
  ...
  Cuotas: 197
  Pendiente: C$122,140.00

Mensaje de totales (final, SPEC-082):
💰 *Totales — Cuotas pendientes*
  NIO: C$228,xxx.xx
```

### 4.5 Dependencias

- **Internas**: `internal/services/telegram_bot.go`, `internal/services/telegram_bot_test.go`.
- **Externas**: Ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado `/deudas_pendientes` con una deuda de 3 cuotas pendientes, cuando el usuario ejecuta el comando, entonces el mensaje de esa deuda lista las 3 cuotas con fecha y monto, y termina con `Cuotas: 3` y el total.
- [ ] CA-002: Dado una deuda con 60 cuotas pendientes, cuando se formatea el mensaje, entonces se generan 3 mensajes (25+25+10) sin encabezado repetido y con resumen en el último.
- [ ] CA-003: Dado que no hay cuotas pendientes, cuando el usuario ejecuta `/deudas_pendientes`, entonces se envía un único mensaje `✅ No hay deudas pendientes.` (sin totales).
- [ ] CA-004: Dado pendientes en USD y NIO, cuando se genera el mensaje de totales, entonces se muestran los totales por moneda correctos (SPEC-082 REQ-004).
- [ ] CA-005: Dado un fallo de red en un envío intermedio, cuando el bot continúa, entonces los mensajes restantes se envían y el error queda en el log.

### 5.2 No funcionales

- [ ] CA-NF-001: Ningún mensaje generado supera los 4096 caracteres (con bloque de 25 cuotas).
- [ ] CA-NF-002: Sin cambios de esquema SQLite ni dependencias nuevas.

### 5.3 Testing

- **Unit tests**: `formatDeudasPendientes` con 3 cuotas (itemización + resumen), 60 cuotas (particionado 25/25/10), caso vacío, multi-moneda, formato personalizado (SPEC-058).
- **Integration tests**: Ninguno nuevo (sin cambios de API).
- **E2E tests**: Prueba manual local con el bot: `/deudas_pendientes` contra la DB local (copia de producción) y verificación de los mensajes itemizados.
- **Carga/Performance**: Sin métricas nuevas; envío secuencial sin impacto en iHost.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Extender `debtGroup` con la lista de cuotas pendientes (dueDate + amount) | 0.25 días | Ninguna |
| 2 | Reescribir `formatDeudasPendientes`: itemización + particionado en bloques de 25 + resumen al final | 0.5 días | Fase 1 |
| 3 | Tests unitarios (itemización, particionado, vacío, multi-moneda) + validación manual local | 0.5 días | Fase 2 |

### 6.2 Milestones

1. **MVP**: `/deudas_pendientes` itemiza cada cuota pendiente con particionado y resumen por deuda + totales por moneda.
2. **V1.0**: Tests unitarios y validación manual con el usuario.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Deuda con cientos de cuotas genera muchos mensajes (spam) | Media | Medio | Bloque de 25 cuotas; volúmenes reales máx ~197 → ≤8 mensajes |
| El formato Markdown (`*bold*`, `—`) se rompe en Telegram | Baja | Bajo | Reutilizar patrón MarkdownV1 ya validado en SPEC-079/080/082 |
| Regresión del particionado por cambios de longitud de línea | Baja | Bajo | Constante de 25 cuotas por bloque, inmune a longitud de línea |
| El usuario esperaba otra cosa (ej: saldo restante) | Baja | Bajo | Requerimiento confirmado explícitamente con el usuario (itemizar cuotas) |

## 8. Notas y Referencias

- SPEC-079 (bot Go), SPEC-080 (comandos /servicios_pendientes + /deudas_pendientes), SPEC-082 (mensajes individuales + totales por moneda), SPEC-058 (formato de moneda).
- Datos de producción verificados vía SSH al iHost (`debt_bills` con `amount` por cuota, `due_date`).
- Límite de Telegram: 4096 caracteres por mensaje.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-19 | paulomcnally | Creación inicial de la especificación (requerimiento clarificado con el usuario: itemizar cuotas pendientes) |
| 2026-09-19 | paulomcnally | Implementación: `debtGroup` con lista de cuotas (`pendingInstallment`), `formatDeudasPendientes` itemiza cada cuota (fecha + monto), particionado en bloques de 25 (`chunkInstallments`, `maxInstallmentsPerMessage`), encabezado en primer bloque y resumen al final del último. Tests actualizados (itemización, particionado 60→3 bloques, vacío, multi-moneda). Estado → in_progress |