---
title: "Sistema de facturación automática: monto fijo/variable y generación programada"
id: "SPEC-008"
status: "released"
author: "p40la-ihost-team"
created: "2026-08-15"
updated: "2026-08-15"
github_issue: 9
---

# Sistema de facturación automática: monto fijo/variable y generación programada

**ID**: SPEC-008  
**Estado**: draft  
**Autor**: p40la-ihost-team  
**Creado**: 2026-08-15  
**Actualizado**: 2026-08-15 (revisión: todos los servicios generan facturas, hora desde settings)

---

## 1. Resumen Ejecutivo

Esta especificación define un sistema de facturación automática para todos los servicios (fijos y variables) que permite la generación programada de facturas en estado pendiente. La diferencia entre ambos tipos radica en que los servicios de monto fijo generan facturas con un monto inmutable, mientras que los de monto variable generan facturas cuyo monto puede editarse posteriormente. Actualmente, los servicios y facturas existen en el sistema pero no hay un mecanismo para definir si un servicio tiene monto fijo o variable, ni para automatizar la creación periódica de facturas.

El sistema permitirá definir desde el formulario de creación/edición de servicios si el monto es fijo o variable, y configurar el día del mes en que se deben generar las facturas automáticamente. Todos los servicios pueden tener generación automática independientemente de su tipo de monto. La hora de generación se configura desde las settings del sistema (no variables de entorno), con un valor por defecto almacenado en la base de datos desde su creación. Esto es esencial para la gestión continua de servicios recurrentes como suscripciones, alquileres, o servicios periódicos.

Consideraciones específicas de iHost: el sistema de programación debe ser ligero (sin dependencias pesadas de cron), usar SQLite con WAL mode para concurrencia, y consumir mínima memoria RAM. La ejecución automática se realizará mediante un ticker en Go que verifique diariamente si hay facturas pendientes de generación. La configuración de la hora de generación se almacena en una tabla de settings en SQLite.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: El formulario de creación/edición de servicios debe permitir seleccionar el tipo de monto: `fijo` o `variable`.
2. **REQ-002**: Todos los servicios (fijos y variables) deben permitir configurar el día del mes (1-31) en que se generan las facturas automáticamente.
3. **REQ-003**: El sistema debe generar automáticamente facturas en estado `pending` para todos los servicios con `auto_generate: true` según el día configurado, independientemente de si son fijos o variables.
4. **REQ-004**: Las facturas generadas automáticamente deben heredar el monto, frecuencia (mensual/anual) y datos del servicio asociado. Las facturas de servicios variables deben permitir edición posterior del monto.
5. **REQ-005**: La UI debe mostrar claramente si un servicio es de monto fijo o variable, y el día de generación configurado.
6. **REQ-006**: La hora de generación automática debe configurarse desde las settings del sistema (no variables de entorno), con un valor por defecto almacenado en la base de datos desde su creación.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

6. **REQ-006**: Permitir activar/desactivar la generación automática por servicio (toggle on/off).
7. **REQ-007**: Las facturas generadas automáticamente deben tener una referencia al servicio que las originó y al período que cubren.
8. **REQ-008**: Validar que el día de generación sea válido para la frecuencia (ej: día 29-31 solo para meses que lo tengan, o ajustar al último día del mes).
9. **REQ-009**: Log de generación de facturas con fecha, servicio asociado y resultado (éxito/error).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

10. **REQ-010**: Notificación visual en el dashboard cuando hay facturas pendientes generadas automáticamente.
11. **REQ-011**: Permitir reintentar la generación de facturas fallidas desde la UI.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: El ticker de verificación debe ejecutarse una vez al día, con consumo de CPU < 1% en iHost.
- **Almacenamiento**: Tabla `system_settings` con rotación mínima (solo claves de configuración). Logs de generación con rotación simple (máximo 30 días o 1000 entradas).
- **iHost**: Sin dependencias externas de cron o schedulers. Sin variables de entorno para configuración de billing. Usar `time.Ticker` de Go stdlib. SQLite con WAL mode.
- **Disponibilidad**: Si el sistema se reinicia, el ticker debe reanudarse automáticamente al iniciar el servidor, leyendo la hora desde `system_settings`.
- **Seguridad**: Solo usuarios autenticados pueden modificar la configuración de generación automática y settings.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **SPEC-004** define el módulo de servicios con facturas, pero no contempla automatización.
- **SPEC-007** maneja la distinción entre servicios mensuales y anuales para el formulario de facturas.
- Go stdlib incluye `time.Ticker` y `time.Timer` para programación periódica sin dependencias externas.
- SQLite con WAL mode permite lecturas concurrentes mientras se escriben facturas.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| `time.Ticker` en Go (daily check) | Sin dependencias, ligero, stdlib | Requiere que el servidor esté corriendo | ✅ Seleccionada |
| Cron del sistema operativo | Robusto, probado | Requiere acceso al host, no portable en Docker | ❌ Rechazada |
| Librería de scheduling (`robfig/cron`) | Más flexible | Dependencia externa, overkill para este caso | ❌ Rechazada |
| Worker separado con queue | Escalable | Complejidad innecesaria para iHost | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Tipo de monto como campo en el modelo de servicio
- **Contexto**: Necesidad de diferenciar servicios con monto fijo (inmutable) vs variable (editable después de generar).
- **Decisión**: Agregar campo `billing_type` (`fixed` o `variable`) y `billing_day` (1-31) al modelo `Service`. Todos los servicios pueden tener `auto_generate` independientemente del tipo.
- **Consecuencias**: Requiere migración de DB. Formularios deben actualizarse. La diferencia entre fijo y variable es solo si el monto se puede editar después de generar la factura.

**ADR-002**: Hora de generación configurable desde settings en la DB
- **Contexto**: Las facturas deben generarse una vez al día, en un horario configurable por el usuario desde la UI de settings.
- **Decisión**: Crear tabla `system_settings` en SQLite con clave `billing_generation_hour` (default 0 para medianoche). El scheduler lee esta configuración al iniciar y al cambiar. Sin variables de entorno.
- **Consecuencias**: Requiere migración de DB para crear tabla settings y valor por defecto. El scheduler debe poder recargar la configuración sin reiniciar. Se expone endpoint `GET/PUT /api/settings/billing-generation-hour`.

**ADR-003**: Facturas generadas con estado `pending` y referencia al servicio
- **Contexto**: Las facturas automáticas deben ser revisables y editables antes de marcarse como pagadas. Las facturas de servicios variables necesitan permitir edición del monto.
- **Decisión**: Generar facturas con `status: pending`, vinculadas al servicio origen mediante `service_id`. El período cubierto se calcula según la frecuencia del servicio. Las facturas de servicios variables permiten edición del monto mediante `PUT /api/bills/:id`.
- **Consecuencias**: Las facturas pendientes aparecen en el dashboard y pueden gestionarse manualmente. Trazabilidad completa servicio → factura. El endpoint de actualización de facturas ya existe (según SPEC-004).

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Frontend: ServiceForm] --(API REST)--> [internal/api/service.go]
       |                                        |
       v                                        v
  UI: billing_type, billing_day, auto_generate  [internal/services/service.go]
                                                    |
                                                    v
                                            [internal/services/billing_scheduler.go] (nuevo)
                                                    |
                                                    v
                                            [internal/storage/bill.go] --(SQLite)--> [bills table]
```

### 4.2 Componentes

#### 4.2.1 Modelo Service (actualizado)
- **Responsabilidad**: Representar un servicio con configuración de facturación.
- **Ubicación**: `internal/models/service.go`
- **Nuevos campos**:
  - `BillingType string` (`fixed` o `variable`)
  - `BillingDay int` (1-31, nullable, día de generación automática)
  - `AutoGenerate bool` (true/false, si se generan facturas automáticamente)

#### 4.2.2 BillingScheduler (nuevo)
- **Responsabilidad**: Programar y ejecutar la generación automática de facturas.
- **Interfaz**: 
  - `Start()` - Inicia el ticker
  - `Stop()` - Detiene el ticker
  - `GeneratePendingBills()` - Genera facturas pendientes para servicios configurados
- **Dependencias**: `internal/services/service.go`, `internal/storage/bill.go`
- **Ubicación**: `internal/services/billing_scheduler.go`

#### 4.2.3 API Endpoints (actualizados)
- **Responsabilidad**: Exponer configuración de facturación automática y settings.
- **Ubicación**: `internal/api/service.go`
- **Endpoints existentes actualizados**:
  - `POST /api/services` - Acepta nuevos campos `billing_type`, `billing_day`, `auto_generate`
  - `PUT /api/services/:id` - Permite actualizar configuración de facturación
- **Nuevos endpoints**:
  - `POST /api/services/:id/generate-bill` - Genera factura manualmente para cualquier servicio
  - `GET /api/settings/billing-generation-hour` - Obtiene hora de generación configurada
  - `PUT /api/settings/billing-generation-hour` - Actualiza hora de generación (notifica al scheduler)

### 4.3 Modelo de datos

```
Entidad: Service (actualizado)
- id: INTEGER PRIMARY KEY
- name: TEXT
- icon: TEXT
- billing_type: TEXT (`fixed` o `variable`, default `variable`)
- billing_day: INTEGER (1-31, default 1, día de generación automática)
- auto_generate: BOOLEAN (default false)
- frequency: TEXT (`monthly` o `yearly`)
- amount: REAL
- created_at: DATETIME
- updated_at: DATETIME

Entidad: system_settings (nueva)
- id: INTEGER PRIMARY KEY
- key: TEXT UNIQUE (ej: 'billing_generation_hour')
- value: TEXT (ej: '0' para medianoche)
- created_at: DATETIME
- updated_at: DATETIME

Entidad: Bill (sin cambios estructurales, solo uso)
- id: INTEGER PRIMARY KEY
- service_id: INTEGER (FK → services.id)
- month: INTEGER (1-12, 0 para yearly)
- year: INTEGER
- amount: REAL
- status: TEXT (`pending`, `paid`, `cancelled`)
- due_date: DATETIME
- created_at: DATETIME
- generated_by: TEXT (`manual` o `auto`)
```

### 4.4 APIs / Contratos

#### Endpoint: `POST /api/services`

**Request**:
```json
{
  "name": "Servicio de internet",
  "icon": "wifi",
  "billing_type": "fixed",
  "billing_day": 15,
  "auto_generate": true,
  "frequency": "monthly",
  "amount": 50.00
}
```

**Response 201**:
```json
{
  "id": 1,
  "name": "Servicio de internet",
  "icon": "wifi",
  "billing_type": "fixed",
  "billing_day": 15,
  "auto_generate": true,
  "frequency": "monthly",
  "amount": 50.00,
  "created_at": "2026-08-15T00:00:00Z",
  "updated_at": "2026-08-15T00:00:00Z"
}
```

#### Endpoint: `PUT /api/services/:id`

**Request**:
```json
{
  "billing_type": "fixed",
  "billing_day": 1,
  "auto_generate": true
}
```

**Response 200**: Mismo schema que POST.

#### Endpoint: `POST /api/services/:id/generate-bill`

**Request**: (vacío, usa datos del servicio)

**Response 201**:
```json
{
  "id": 42,
  "service_id": 1,
  "month": 8,
  "year": 2026,
  "amount": 50.00,
  "status": "pending",
  "due_date": "2026-08-15T00:00:00Z",
  "created_at": "2026-08-15T00:00:00Z",
  "generated_by": "manual"
}
```

#### Endpoint: `GET /api/settings/billing-generation-hour`

**Response 200**:
```json
{
  "key": "billing_generation_hour",
  "value": "0",
  "description": "Hora del día (0-23) para generación automática de facturas"
}
```

#### Endpoint: `PUT /api/settings/billing-generation-hour`

**Request**:
```json
{
  "value": "2"
}
```

**Response 200**:
```json
{
  "key": "billing_generation_hour",
  "value": "2",
  "message": "Hora de generación actualizada. El scheduler se reiniciará en el próximo ciclo."
}
```

### 4.5 Dependencias

- **Internas**: 
  - `internal/models/service.go` (nuevos campos)
  - `internal/models/setting.go` (nuevo modelo para system_settings)
  - `internal/services/service.go` (lógica de validación)
  - `internal/services/billing_scheduler.go` (nuevo)
  - `internal/services/settings.go` (nuevo o actualizado)
  - `internal/storage/bill.go` (generación de facturas)
  - `internal/storage/setting.go` (nuevo, CRUD de settings)
  - `internal/api/service.go` (endpoints actualizados)
  - `internal/api/settings.go` (nuevo o actualizado)
  - `frontend/src/pages/ServiceFormPage.tsx` (nuevos campos en formulario)
  - `frontend/src/pages/SettingsPage.tsx` (campo para hora de generación)
- **Externas**: 
  - Ninguna nueva. Solo `time` package de Go stdlib.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado un servicio nuevo, cuando el usuario selecciona `billing_type`, entonces debe poder ingresar `billing_day` (1-31) y `auto_generate` (true/false) independientemente del tipo.
- [ ] CA-002: Dado un servicio fijo con `auto_generate: true` y `billing_day: 15`, cuando el sistema llega al día 15 del mes a la hora configurada, entonces se genera una factura `pending` con el monto del servicio.
- [ ] CA-003: Dado un servicio variable con `auto_generate: true` y `billing_day: 10`, cuando el sistema llega al día 10 del mes a la hora configurada, entonces se genera una factura `pending` cuyo monto puede editarse posteriormente.
- [ ] CA-004: Dado un servicio anual con `auto_generate: true`, cuando el sistema llega al mes y día configurados, entonces se genera una factura `pending` con `month: 0`.
- [ ] CA-005: Dado un servicio con `billing_day: 31`, cuando el mes no tiene 31 días, entonces la factura se genera el último día del mes.
- [ ] CA-006: Dado el dashboard de facturas, cuando hay facturas `pending` generadas automáticamente, entonces se muestran con un indicador visual de origen automático.
- [ ] CA-007: Dado el panel de settings, cuando el usuario modifica la hora de generación, entonces el scheduler actualiza su horario sin necesidad de reiniciar el servidor.
- [ ] CA-008: Dada una nueva instalación, cuando se crea la base de datos, entonces la tabla `system_settings` existe con `billing_generation_hour = 0` (medianoche).

### 5.2 No funcionales

- [ ] CA-NF-001: El ticker de generación consume < 1% de CPU en iHost durante la verificación diaria.
- [ ] CA-NF-002: La migración de base de datos se ejecuta en < 1 segundo para hasta 1000 servicios.
- [ ] CA-NF-003: Si el servidor se reinicia, el scheduler se reanuda automáticamente al iniciar sin intervención manual.

### 5.3 Testing

- **Unit tests**: 
  - Lógica de cálculo de día válido para meses con diferente cantidad de días.
  - Generación de facturas para servicios mensuales vs anuales, fijos vs variables.
  - Validación de campos `billing_type`, `billing_day`, `auto_generate`.
  - CRUD de `system_settings` (lectura/escritura de `billing_generation_hour`).
- **Integration tests**: 
  - Flujo completo: crear servicio fijo/variable → esperar día de generación → verificar factura creada.
  - Cambiar hora de generación en settings → verificar que scheduler actualiza horario.
  - Generar factura de servicio variable → editar monto → verificar persistencia.
- **E2E tests**: 
  - Formulario de servicio: seleccionar tipo, ingresar día, activar auto_generate, guardar, verificar en DB.
  - Settings: modificar hora de generación, verificar que se persiste en DB.
  - Dashboard: verificar que facturas automáticas aparecen con indicador correcto.
- **Carga/Performance**: Verificar que el ticker no impacta el rendimiento del servidor en iHost con 100+ servicios.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración DB: agregar campos `billing_type`, `billing_day`, `auto_generate` a services + tabla `system_settings` con default | 0.5 días | Ninguna |
| 2 | Actualizar modelo Service y crear modelo Setting en Go | 0.5 días | Fase 1 |
| 3 | Crear `billing_scheduler.go` con lógica de ticker y generación automática (lee hora desde settings) | 1 día | Fase 2 |
| 4 | Actualizar endpoints de servicios (POST/PUT) para aceptar nuevos campos | 0.5 días | Fase 2 |
| 5 | Crear endpoints de settings (GET/PUT billing_generation_hour) | 0.5 días | Fase 2 |
| 6 | Actualizar formulario de servicio en React (campos billing_type, billing_day, auto_generate) | 1 día | Fase 2 |
| 7 | Agregar campo de hora de generación en SettingsPage | 0.5 días | Fase 5 |
| 8 | Integrar scheduler en `cmd/server/main.go` | 0.5 días | Fase 3 |
| 9 | Tests unitarios e integración | 1 día | Fases 1-8 |
| 10 | Validación manual en local y ajustes | 0.5 días | Fase 9 |

### 6.2 Milestones

1. **MVP**: Servicio con campos de facturación fijos/variables + tabla system_settings + generación manual de facturas (Fases 1-7).
2. **V1.0**: Scheduler automático integrado + settings configurables desde UI + tests completos + validación en iHost (Fases 8-10).

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Servidor apagado en día de generación | Media | Medio | Al reiniciar, verificar si hay facturas pendientes del período y generarlas. |
| Mes con menos días que `billing_day` | Alta | Bajo | Ajustar al último día del mes (ej: 31 → 28/29/30 según corresponda). |
| Duplicación de facturas por reinicio del scheduler | Baja | Alto | Verificar existencia de factura para el período antes de generar (unique constraint service_id + month + year). |
| Consumo excesivo de memoria por ticker | Baja | Bajo | Usar `time.Ticker` con intervalo de 24h, sin goroutines adicionales. |
| Confusión del usuario entre monto fijo y variable | Media | Medio | Tooltips claros en la UI: fijo = monto inmutable, variable = monto editable después de generar. |
| Cambio de hora de settings no se refleja inmediatamente | Baja | Bajo | El scheduler recarga la configuración en cada ciclo o al recibir notificación de cambio. |

## 8. Notas y Referencias

- SPEC-004: Dashboard con módulo de servicios y facturas (contexto base).
- SPEC-007: Manejo de mes para servicios anuales (relacionado con frecuencia).
- Documentación Go `time.Ticker`: https://pkg.go.dev/time#Ticker
- SQLite WAL mode: https://www.sqlite.org/wal.html

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-15 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-08-15 | p40la-ihost-team | Actualización: todos los servicios (fijos y variables) generan facturas automáticamente. Hora de generación manejada desde settings en DB (no variables de entorno). |

## 10. Lecciones Aprendidas

### Errores cometidos durante la implementación

1. **NUNCA ejecutar `rm -f data/app.db`**: Se borró la base de datos local del usuario al probar migraciones. La DB local contiene datos de producción del usuario. Las migraciones deben probarse contra la DB existente o en un archivo temporal separado (`/tmp/test-app.db`).

2. **Seguir el patrón de UI existente**: Las páginas de listado deben usar el patrón card con EmptyCard (título, descripción, botón) cuando no hay registros. No mostrar formularios inline directamente. Referencia: `HomesPage.tsx` como patrón correcto.

3. **Validación de prerequisitos en backend Y frontend**: Al igual que se valida que existan homes antes de crear servicios, se debe validar que existan instituciones. Esto requiere:
   - Backend: Check en `CreateService` que `institutions` tenga al menos 1 registro
   - Frontend: Redirect a `/institutions/new` si no hay instituciones al intentar crear servicio
   - Storage: Método `Count()` en `InstitutionStorage`
