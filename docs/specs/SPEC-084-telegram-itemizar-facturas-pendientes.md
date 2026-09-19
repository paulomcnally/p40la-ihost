---
title: "Bot Telegram: itemizar facturas pendientes con fecha de vencimiento"
id: "SPEC-084"
status: "draft"
author: "paulomcnally"
created: "2026-09-19"
updated: "2026-09-19"
github_issue: 87
---

# Bot Telegram: itemizar facturas pendientes con fecha de vencimiento

**ID**: SPEC-084  
**Estado**: draft  
**Autor**: paulomcnally  
**Creado**: 2026-09-19  
**Actualizado**: 2026-09-19

---

## 1. Resumen Ejecutivo

El comando `/servicios_pendientes` del bot de Telegram (SPEC-080 + SPEC-082) responde con **un mensaje por servicio** mostrando: nombre del servicio, casa, cantidad de facturas pendientes (`Facturas: N`) y el monto total acumulado (`Pendiente: <suma>`). El usuario reporta que **no se muestra cuándo vence cada factura**: "pendiente" es el total de la acumulación de todas, pero si quiere saber si es una o más facturas, necesita ver la fecha límite de pago de cada una.

El requerimiento confirmado con el usuario: **itemizar cada factura pendiente con su fecha de vencimiento y monto** dentro del mensaje de cada servicio, manteniendo el resumen (cantidad + total) y el mensaje de totales por moneda (SPEC-082). Es el mismo patrón que SPEC-083 aplicó a `/deudas_pendientes` con las cuotas pendientes.

**Formato aprobado por el usuario (validado en evaluación manual)**: cada factura itemizada lleva un **semáforo de vencimiento** (🟢/🟡/🔴) con la misma regla que la web (`DueDateBadge`, SPEC-081 REQ-008): verde si faltan >10 días, amarillo entre 1 y 10 días, rojo si vence hoy o ya venció. Fecha legible (`07 sep 2026`), orden por urgencia (vencidas primero, sin fecha al final), separador visual antes del resumen y aviso de días restantes ("Vence en N días" / "Vencida hace N días" / "Vence hoy"). Las facturas sin `due_date` se muestran con 🟢 y su periodo como referencia.

**Ajustes de evaluación manual (iteración 2)**: (1) `/servicios_pendientes` **filtra las facturas a "vencidas + mes actual"** (decisión del usuario): solo se muestran facturas cuyo `due_date` es anterior o igual al último día del mes en curso (en la zona horaria configurada, ADR-004); las facturas con `due_date` en meses futuros se excluyen del mensaje, del conteo (`Facturas: N`) y del total (`Pendiente: <suma>`). Las facturas sin `due_date` (periodo actual) se mantienen. (2) El **separador** pasa de `──────────────────` (box-drawing `─`, se renderiza gigante/ancho doble en el cliente de Telegram del usuario) a guiones ASCII `--------------------`.

**Ajustes de evaluación manual (iteración 3)**: la cantidad de guiones del separador pasa a ser **configurable** desde Configuración → Bot de Telegram con un campo entero `telegram_bot_separator_length` (default 20, el valor actual). El bot lee el setting en runtime al formatear `/servicios_pendientes`.

**Ajustes de evaluación manual (iteración 4)**: los textos del resumen **"Facturas" y "Pendiente" se muestran en negrita** (MarkdownV1 `*Facturas*: N` / `*Pendiente*: <monto>`). Aplica también al resumen de `/deudas_pendientes` (`Cuotas`/`Pendiente`) para mantener consistencia.

**Ajustes de evaluación manual (iteración 5)**: nueva configuración `telegram_bot_show_months` (entero 1-12, default 1 = mes actual) que controla el **rango de meses futuros** que muestra `/servicios_pendientes`. Regla confirmada con el usuario: las facturas **vencidas se muestran SIEMPRE** sin importar su antigüedad; las **no vencidas** se muestran si su `due_date` cae dentro de los próximos N meses desde el mes actual (N=1 → solo mes actual, N=2 → también el próximo mes, etc.). Las facturas sin `due_date` se mantienen siempre. El comportamiento con N=1 (default) es idéntico al de la iteración 2.

Consideraciones iHost: cambios en `internal/models/bill.go` (agregar `DueDate` a `PendingBillDetail`), `internal/storage/bill.go` (seleccionar `b.due_date` en `ListPendingWithDetails`) y `internal/services/telegram_bot.go` (formateo) + tests. La columna `due_date` ya existe en `bills` desde SPEC-081, por lo que **no hay cambios de esquema DB ni migraciones**. El cálculo de "hoy" usa la zona horaria configurada (SPEC-078). Sin dependencias nuevas.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: `/servicios_pendientes` muestra, por cada servicio con facturas pendientes, **cada factura pendiente itemizada** con su fecha de vencimiento (`DueDate`) y monto. Si la factura no tiene `due_date` definida, se muestra el periodo (año-mes) como referencia de vencimiento.
2. **REQ-002**: Se mantiene el encabezado del servicio (nombre + casa) y el resumen final por servicio (`Facturas: N` / `Pendiente: <monto total>`), separado por una línea visual `──────`.
3. **REQ-003**: Se mantiene el mensaje final de **totales por moneda** (SPEC-082 REQ-003): suma de facturas pendientes agrupada por código de moneda.
4. **REQ-004**: El modelo `PendingBillDetail` se extiende con `DueDate` (string opcional, formato `YYYY-MM-DD`) y el query `ListPendingWithDetails` la selecciona de `bills.due_date`.
5. **REQ-009**: Cada factura itemizada muestra un **semáforo de vencimiento** con la misma regla que la web (SPEC-081 REQ-008): 🟢 si faltan más de 10 días, 🟡 si faltan entre 10 y 1 día, 🔴 si vence hoy o ya venció. Incluye texto de estado: "Vence en N días", "Vence hoy" o "Vencida hace N días" (singular/plural correcto).
6. **REQ-010**: La fecha de vencimiento se muestra en **formato legible** (`07 sep 2026`, meses en español abreviados). El cálculo de días usa la **zona horaria configurada** (SPEC-078) como "hoy".
7. **REQ-011**: Las facturas se ordenan **por urgencia**: vencidas/próximas a vencer primero (por fecha de vencimiento ascendente) y las sin `due_date` al final.
8. **REQ-012**: `/servicios_pendientes` **filtra las facturas a "vencidas + mes actual"**: se muestran solo facturas con `due_date` <= último día del mes en curso (zona configurada) o sin `due_date`. Las de meses futuros se excluyen del mensaje, del conteo y del total por servicio.
9. **REQ-013**: El separador antes del resumen usa guiones ASCII (`--------------------`), no box-drawing `─` (se renderiza mal en el cliente de Telegram del usuario).
10. **REQ-014**: La **cantidad de guiones del separador es configurable** vía setting `telegram_bot_separator_length` (entero, default 20, rango 0-100; 0 = sin separador). Se expone en la API de system settings y en la UI de Configuración → Bot de Telegram.
11. **REQ-015**: Los textos del resumen **"Facturas" y "Pendiente" van en negrita** (MarkdownV1 `*...*`). El resumen de `/deudas_pendientes` (`Cuotas`/`Pendiente`) también, por consistencia.
12. **REQ-016**: Nuevo setting `telegram_bot_show_months` (entero 1-12, default 1) que controla el **rango de meses futuros** de `/servicios_pendientes`: las facturas **vencidas se muestran siempre**; las **no vencidas** solo si `due_date` <= fin del mes actual + (N-1) meses. Sin `due_date` se mantienen siempre. N=1 (default) = comportamiento actual (vencidas + mes actual).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-005**: Si un servicio tiene más facturas pendientes de las que caben en un mensaje (límite 4096 caracteres de Telegram), el mensaje de ese servicio se **particiona en varios mensajes**, con encabezado solo en el primer bloque y resumen al final del último (patrón SPEC-083 ADR-001/002).
2. **REQ-006**: Si no hay facturas pendientes, se envía un único mensaje `✅ No hay facturas pendientes.` (sin totales, comportamiento actual).
3. **REQ-007**: Se mantiene el orden de servicios por monto total descendente (`sortGroups`) y el formato de moneda configurado (SPEC-058).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-008**: Tests unitarios de los nuevos formateadores: itemización con y sin `due_date`, particionado, caso vacío y multi-moneda.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Mismo patrón de envío secuencial (SPEC-082). Con volúmenes reales (una factura por mes por servicio) rara vez se supera un mensaje por servicio. Sin impacto en iHost.
- **Seguridad**: Sin cambios de tokens, chat IDs ni secretos.
- **Almacenamiento**: Sin archivos nuevos ni cambios de esquema DB (`due_date` ya existe en `bills` desde SPEC-081).
- **Disponibilidad**: Si falla el envío de un mensaje intermedio, se loguea y se continúa (comportamiento de `sendMany`).
- **iHost**: Sin dependencias nuevas.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Reporte del usuario**: el bot envía al pedir servicios pendientes "Casa, Facturas, Pendiente" pero "no me está enviando cuando vence cada factura". Pide: "si quiero saber si es una o más, poder ver cuándo es su fecha límite de pago; ese valor ya lo tenemos en la tabla".
- **Código actual**: `formatServiciosPendientes` (SPEC-082) agrupa por servicio y muestra solo `Facturas: N` + `Pendiente: <suma>`. `PendingBillDetail` **no expone** `DueDate` (a diferencia de `PendingDebtDetail` que sí lo expone, SPEC-083).
- **Datos**: SPEC-081 agregó `due_date` a `bills` (columna + modelo `Bill.DueDate *string`). El query `ListPendingWithDetails` (internal/storage/bill.go:211) no la selecciona aún. Verificado: el modelo `Bill` tiene `DueDate *string` desde SPEC-081.
- **Límite de Telegram**: 4096 caracteres por mensaje. Un servicio con muchas facturas itemizadas podría superarlo → particionado (patrón ya resuelto en SPEC-083 con 25 cuotas por bloque).
- **Precedente directo**: SPEC-083 itemizó las cuotas pendientes de `/deudas_pendientes` con el mismo problema y patrón de solución; esta spec replica el enfoque para facturas.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Itemizar cada factura con su `due_date` dentro del mensaje del servicio, reutilizando el patrón de SPEC-083 | Datos ya en la tabla (`bills.due_date`); legible; consistente con `/deudas_pendientes` | Requiere exponer `DueDate` en `PendingBillDetail` + query | ✅ Seleccionada |
| Mostrar solo la fecha más próxima de vencimiento por servicio | Mensaje corto | No responde a "si quiero saber si es una o más, ver cuándo vence cada una" | ❌ Rechazada (decisión del usuario) |
| Un mensaje por factura individual | Máxima claridad | Ruido: cada factura genera un mensaje separado; rompe el agrupado por servicio de SPEC-082 | ❌ Rechazada |
| Calcular vencimiento desde el periodo (año-mes) del servicio | Sin tocar modelo | La fecha real de vencimiento ya está en la tabla; usar periodo es menos preciso | ❌ Rechazada (fallback para facturas sin `due_date`) |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Fallback a periodo (año-mes) cuando `due_date` es NULL
- **Contexto**: SPEC-081 hizo `due_date` opcional; facturas existentes creadas antes pueden no tenerla. Telegram no puede mostrar un valor vacío útil.
- **Decisión**: Si `DueDate` viene NULL, la factura se muestra con su periodo `YYYY-MM` (ej: `📅 ago 2026 — C$620.00`) con semáforo 🟢 y texto "Sin fecha".
- **Consecuencias**: Mensajes siempre informativos; sin cambios de DB.

**ADR-002**: Particionado por cantidad de facturas por bloque para servicios con muchas pendientes
- **Contexto**: El límite real es 4096 chars; medir caracteres agrega complejidad.
- **Decisión**: Reutilizar la constante de 25 ítems por bloque (SPEC-083 ADR-001). Con una factura por mes por servicio, el caso normal es 1 bloque; el particionado es red de seguridad.
- **Consecuencias**: Mensajes predecibles y testables; sin lógica nueva de particionado (reutiliza `chunkInstallments`).

**ADR-003**: Semáforo de vencimiento con la regla de la web (`DueDateBadge`, SPEC-081 REQ-008)
- **Contexto**: El usuario pidió emoji de círculo verde/amarillo/rojo "según la regla actual que habíamos mencionado para los servicios".
- **Decisión**: Replicar exactamente la regla de `DueDateBadge.tsx`: verde si faltan >10 días, amarillo entre 1 y 10, rojo si vence hoy o ya venció. Telegram no soporta colores, se usa emoji 🟢/🟡/🔴.
- **Consecuencias**: Consistencia entre web y bot; regla única y verificable en tests.

**ADR-004**: "Hoy" en la zona horaria configurada (SPEC-078)
- **Contexto**: El iHost corre en UTC por defecto; la zona configurada puede diferir (ej: America/Managua).
- **Decisión**: `handleServiciosPendientes` resuelve `GetTimezoneLocation` y pasa "hoy" a los formateadores (fallback UTC si no hay zona configurada).
- **Consecuencias**: Coincidencia con la web (que usa la misma zona, SPEC-081 ADR-004); sin dependencias nuevas.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Telegram Update /servicios_pendientes]
        │
        ▼
[handleServiciosPendientes]  (sin cambios de lógica)
        │  BillStorage.ListPendingWithDetails()  (nuevo: due_date por factura)
        ▼
[formatServiciosPendientes]  ─► []string: un mensaje por servicio (particionado) + 1 totales
        │
        ▼
[sendMany × M]  (secuencial, MarkdownV1)
```

### 4.2 Componentes

#### 4.2.1 `BillStorage.ListPendingWithDetails` (`internal/storage/bill.go`)
- **Responsabilidad**: Devolver facturas pendientes con contexto de casa, institución, servicio y moneda (SPEC-031).
- **Cambio**: Agregar `b.due_date` al SELECT y escanearlo en `PendingBillDetail.DueDate` (como `sql.NullString`, mapeando a `*string` o string vacío según convención del modelo `Bill`).
- **Dependencias**: Columna `bills.due_date` (existe desde SPEC-081).

#### 4.2.2 `PendingBillDetail` (`internal/models/bill.go`)
- **Cambio**: Agregar campo `DueDate *string` (o `string` con convención `omitempty`) + tag `json:"due_date,omitempty"`.
- **Dependencias**: Ninguna nueva.

#### 4.2.3 `formatServiciosPendientes` + `pendingGroup` (`internal/services/telegram_bot.go`)
- **Responsabilidad**: Construir la lista de mensajes de `/servicios_pendientes`.
- **Cambio**: Extender `pendingGroup` con `bills []pendingBill` (dueDate + amount), itemizar cada factura y cerrar con el resumen; particionar si excede el límite (reutilizando `chunkInstallments` y la constante de 25).
- **Interfaz**: `formatServiciosPendientes(pending []appmodels.PendingBillDetail, format CurrencyFormat) []string` (firma actual, sin cambios externos).

### 4.3 Modelo de datos

Sin cambios de esquema. `PendingBillDetail` se extiende:

```
PendingBillDetail {
  BillID, ServiceID, HomeID, HomeName, Institution, ServiceName,
  CurrencySymbol, CurrencyCode, Year, Month, Amount, Status, CreatedAt,
  DueDate *string   // NUEVO: bills.due_date (SPEC-081), formato YYYY-MM-DD o NULL
}
```

### 4.4 APIs / Contratos

Sin cambios de API REST ni de frontend. Contrato interno de mensajes:

```
Mensaje por servicio (bloque 1):
⚡ *Internet Claro* — Casa A

  🔴 Vencida hace 3 días
  📅 20 ago 2026 — C$1,050.00

  🟡 Vence en 5 días
  📅 25 sep 2026 — C$1,050.00

  🟢 Vence en 30 días
  📅 20 oct 2026 — C$1,050.00

  🟢 Sin fecha — ago 2026
  📅 ago 2026 — C$99.90

  ──────────────────
  Facturas: 4
  Pendiente: C$3,249.90

Mensaje de totales (final, SPEC-082):
💰 *Totales — Facturas pendientes*
  NIO: C$xxx,xxx.xx
```

### 4.5 Dependencias

- **Internas**: `internal/models/bill.go`, `internal/storage/bill.go`, `internal/services/telegram_bot.go`, `internal/services/telegram_bot_test.go`, `internal/storage/bill_pending_test.go`.
- **Externas**: Ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Dado `/servicios_pendientes` con un servicio con 2 facturas pendientes con `due_date`, cuando el usuario ejecuta el comando, entonces el mensaje del servicio lista las 2 facturas con fecha legible de vencimiento y monto, con su semáforo, y termina con separador, `Facturas: 2` y el total.
- [x] CA-002: Dado una factura pendiente sin `due_date`, cuando se formatea el mensaje, entonces se muestra con 🟢, texto "Sin fecha" y su periodo (año-mes) como referencia, al final de la lista.
- [x] CA-003: Dado facturas con distintos vencimientos, cuando se formatea el mensaje, entonces el semáforo es correcto según SPEC-081 REQ-008 (>10 días 🟢, 1-10 días 🟡, hoy/vencida 🔴) con texto "Vence en N días", "Vence hoy" o "Vencida hace N días" (singular/plural correcto).
- [x] CA-004: Dado facturas con y sin `due_date`, cuando se formatea el mensaje, entonces se ordenan por urgencia (vencidas/próximas primero por fecha ascendente, sin fecha al final).
- [x] CA-005: Dado un servicio con más de 25 facturas pendientes, cuando se formatea el mensaje, entonces se particiona en varios mensajes sin encabezado repetido y con resumen al final del último.
- [x] CA-006: Dado que no hay facturas pendientes, cuando el usuario ejecuta `/servicios_pendientes`, entonces se envía un único mensaje `✅ No hay facturas pendientes.` (sin totales).
- [x] CA-007: Dado pendientes en USD y NIO, cuando se genera el mensaje de totales, entonces se muestran los totales por moneda correctos (SPEC-082 REQ-003).
- [x] CA-008: Dado un fallo de red en un envío intermedio, cuando el bot continúa, entonces los mensajes restantes se envían y el error queda en el log.
- [x] CA-009: Dado un servicio con facturas vencidas, del mes actual y de meses futuros, cuando se ejecuta `/servicios_pendientes`, entonces solo se muestran las vencidas y las del mes actual; las futuras no aparecen ni en el conteo ni en el total.
- [x] CA-010: Dado el mensaje de un servicio, cuando se renderiza el separador, entonces se ve como `--------------------` (guiones ASCII) y no como box-drawing.
- [x] CA-011: Dado `telegram_bot_separator_length = 30`, cuando se ejecuta `/servicios_pendientes`, entonces el separador tiene 30 guiones; con 0 no se muestra separador; con valor vacío/sin config se usa 20 (default).
- [x] CA-012: Dado el resumen de un servicio y de una deuda, cuando se formatea, entonces "Facturas"/"Pendiente" (y "Cuotas"/"Pendiente" en deudas) aparecen en negrita con MarkdownV1.
- [x] CA-013: Dado `telegram_bot_show_months = 1` (default) con facturas vencidas, del mes actual y de meses futuros, cuando se ejecuta `/servicios_pendientes`, entonces se muestran vencidas y mes actual; las futuras se excluyen.
- [x] CA-014: Dado `telegram_bot_show_months = 2`, cuando se ejecuta `/servicios_pendientes`, entonces además de vencidas y mes actual se muestran las facturas del próximo mes; con =3 también las del siguiente; las vencidas de cualquier antigüedad se muestran siempre.
- [x] CA-015: Dado `telegram_bot_show_months = 0 o 13`, cuando se intenta guardar, entonces la API rechaza con 400 y el valor anterior se mantiene.

### 5.2 No funcionales

- [x] CA-NF-001: Ningún mensaje generado supera los 4096 caracteres (con bloque de 25 facturas).
- [x] CA-NF-002: Sin cambios de esquema SQLite ni dependencias nuevas.

### 5.3 Testing

- **Unit tests**: `formatServiciosPendientes` con 2 facturas (itemización + resumen), factura sin `due_date` (fallback periodo), 30 facturas (particionado 25/5), caso vacío, multi-moneda, formato personalizado (SPEC-058). Storage: `ListPendingWithDetails` devuelve `DueDate` correctamente (con y sin valor).
- **Integration tests**: Ninguno nuevo (sin cambios de API).
- **E2E tests**: Prueba manual local con el bot: `/servicios_pendientes` contra la DB local (copia de producción) y verificación de los mensajes itemizados.
- **Carga/Performance**: Sin métricas nuevas; envío secuencial sin impacto en iHost.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Agregar `DueDate` a `PendingBillDetail` + actualizar query `ListPendingWithDetails` (SELECT + scan) | 0.25 días | Ninguna |
| 2 | Extender `pendingGroup` con la lista de facturas (dueDate + amount) y reescribir `formatServiciosPendientes`: itemización + particionado (reutilizando `chunkInstallments`) + resumen al final | 0.5 días | Fase 1 |
| 3 | Semáforo 🟢/🟡/🔴 (regla SPEC-081 REQ-008), fecha legible, días restantes, zona horaria (SPEC-078) y orden por urgencia | 0.5 días | Fase 2 |
| 4 | Tests unitarios (semáforo, fallback sin due_date, particionado, orden, vacío, multi-moneda) + validación manual local | 0.5 días | Fase 3 |

### 6.2 Milestones

1. **MVP**: `/servicios_pendientes` itemiza cada factura pendiente con su fecha de vencimiento, con particionado y resumen por servicio + totales por moneda.
2. **V1.0**: Tests unitarios y validación manual con el usuario.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Muchas facturas pendientes generan muchos mensajes (spam) | Baja | Medio | Bloque de 25 facturas; caso normal: 1 factura/mes/servicio → 1 mensaje |
| Facturas sin `due_date` muestran datos vacíos o confusos | Media | Bajo | Fallback a periodo año-mes (ADR-001) |
| El formato Markdown (`*bold*`, `—`) se rompe en Telegram | Baja | Bajo | Reutilizar patrón MarkdownV1 ya validado en SPEC-079/080/082 |
| Regresión en el orden de servicios o totales (SPEC-082) | Baja | Medio | Tests existentes de totales y orden se mantienen y se actualizan |

## 8. Notas y Referencias

- SPEC-079 (bot Go), SPEC-080 (comandos /servicios_pendientes + /deudas_pendientes), SPEC-082 (mensajes individuales + totales por moneda), SPEC-083 (itemización de cuotas pendientes, patrón a replicar), SPEC-081 (columna `due_date` en `bills`), SPEC-058 (formato de moneda).
- Límite de Telegram: 4096 caracteres por mensaje.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-19 | paulomcnally | Creación inicial de la especificación (requerimiento del usuario: ver fecha límite de pago de cada factura pendiente en /servicios_pendientes) |
| 2026-09-19 | paulomcnally | Implementación inicial: `PendingBillDetail.DueDate` + query con `b.due_date`, `formatServiciosPendientes` itemiza cada factura (fecha + monto) con particionado en bloques de 25 y fallback a periodo. Tests verdes + validación manual local con el bot (conflicto 409 con iHost resuelto deteniendo el add-on). |
| 2026-09-19 | paulomcnally | Cambio iterativo solicitado por el usuario en evaluación manual: formato mejorado con semáforo 🟢/🟡/🔴 (regla SPEC-081 REQ-008), fecha legible "07 sep 2026", orden por urgencia, separador visual antes del resumen, texto de días restantes y zona horaria configurada (SPEC-078). Spec actualizada (REQ-009/010/011, ADR-003/004, CA-003/004). |
| 2026-09-19 | paulomcnally | Iteración 2: filtro "vencidas + mes actual" (REQ-012) excluyendo facturas de meses futuros del mensaje/conteo/total; separador cambiado de box-drawing a guiones ASCII (REQ-013). CAs CA-009/CA-010. |
| 2026-09-19 | paulomcnally | Iteración 3-4: separador configurable `telegram_bot_separator_length` (default 20, 0-100) + negritas en Facturas/Pendiente/Cuotas (REQ-014/015). Backend (setting + API + bot) y frontend (SettingsTelegramBotPage + i18n es/en). |
| 2026-09-19 | paulomcnally | Iteración 5: setting `telegram_bot_show_months` (1-12, default 1) — las vencidas se muestran siempre; las no vencidas solo dentro de los próximos N meses (REQ-016). API valida rango con 400. Tests de formateador y settings. Validación manual con el usuario: ✅ funciona. Todos los criterios de aceptación en pass. |