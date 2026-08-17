---
title: "Email informativo al generar factura automática"
id: "SPEC-030"
status: "released"
author: "paulomcnally"
created: "2026-08-16"
updated: "2026-08-16 (v3: released)"
github_issue: 30
---

# Email informativo al generar factura automática

**ID**: SPEC-030  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-08-16  
**Actualizado**: 2026-08-16 (v3: released)

---

## 1. Resumen Ejecutivo

El sistema p40la-ihost ya genera facturas automáticamente mediante el `BillingScheduler` (SPEC-008), pero el usuario no recibe ninguna notificación cuando esto ocurre. Si no revisa la aplicación, puede perder de vista qué facturas se generaron, cuándo vencen y por cuánto. Esto obliga a consultar manualmente la app para mantenerse al día con sus pagos recurrentes.

Esta spec agrega un **email informativo** que se envía **automáticamente** cada vez que el `BillingScheduler` genera una **nueva factura** de forma programada. El correo es de tipo informativo (no requiere acción inmediata) y comunica: que se generó una nueva factura, el período que cubre, el monto a pagar, si el monto es **fijo** o **variable** (editable), y la **institución** asociada al servicio. Se envía **un email por factura** generada, a los destinatarios ya configurados en `alert_emails` (reusando la infraestructura de SPEC-029).

**Consideraciones de iHost**: Se reutiliza íntegramente el `EmailService` existente (SPEC-029), que usa `net/smtp` de la stdlib — **cero dependencias nuevas**. No se crean tablas nuevas (se reutiliza `system_settings` con la clave `last_billing_generation` ya existente para deduplicación diaria). El envío se hace en la misma goroutine del scheduler, de forma no bloqueante para el servidor HTTP. El consumo adicional de RAM es despreciable.

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Cuando el `BillingScheduler` genera automáticamente una nueva factura para un servicio, debe enviarse **un email informativo** a los destinatarios configurados en `alert_emails` (clave ya existente de SPEC-029).
2. **REQ-002**: El email debe seguir el ejemplo informativo:
   - Saludo ("Hola,")
   - Anuncio de que se generó una nueva factura y que debe pagarse en el período correspondiente
   - **Monto** a pagar (con su moneda)
   - Indicación de si el monto es **fijo** o **variable** (monto editable)
   - **Institución** asociada al servicio
   - Período cubierto por la factura (mes/año o año para servicios anuales)
3. **REQ-003**: El envío se realiza únicamente para facturas generadas **automáticamente** por el scheduler. Las facturas generadas manualmente (vía `POST /api/services/:id/generate-bill` o desde la UI) **no** disparan email.
4. **REQ-004**: El envío usa el `EmailService` existente (`internal/services/email_service.go`) y su plantilla HTML única (`{{TITLE}}`, `{{CONTENT}}`, `{{DATE}}`), sin modificar su interfaz.
5. **REQ-005**: Si la configuración SMTP no está completa o no hay destinatarios configurados, el scheduler debe loguear un warning y **continuar** con la generación de facturas (el email no debe bloquear ni revertir la generación).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-006**: El asunto (subject) del email debe ser descriptivo, por ejemplo: `"Nueva factura generada — {nombre del servicio}"`.
2. **REQ-007**: El email debe incluir el monto formateado con la moneda del servicio (reusar lógica existente de `currency` si aplica) y el período (ej: "Agosto 2026" o "Año 2026" para servicios anuales).
3. **REQ-008**: El email debe indicar claramente "monto fijo" o "monto variable (podés editarlo)" según `billing_type` del servicio.
4. **REQ-009**: El email debe incluir la institución del servicio (campo `institution` del modelo `Service`, nombre de la institución si hay `institution_id`).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-010**: Incluir en el email un enlace directo a la app (ej: `http://<ihost-ip>:8088/bills`) para ver la factura.
2. **REQ-011**: Log de emails enviados en la tabla `email_logs` (mismo requerimiento P2 de SPEC-029, compartido).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: El envío de emails no debe bloquear la generación de facturas ni el servidor HTTP. Se ejecuta en la goroutine del scheduler (mismo patrón que SPEC-029).
- **Seguridad — Protección de información sensible**: Las mismas reglas de SPEC-029 aplican: el password SMTP **nunca** se expone en responses de API, logs ni mensajes de error. El email no incluye credenciales ni datos sensibles adicionales (solo datos del servicio y la factura).
- **Almacenamiento**: Sin tablas nuevas para P0/P1. Se reutilizan claves de `system_settings` existentes (`alert_emails`, config SMTP, `last_billing_generation`).
- **Disponibilidad**: Si el servidor SMTP no responde, el scheduler loguea el error (sin credenciales) y la generación de facturas continúa. No hay cola de reintentos (mantener simple para iHost).
- **iHost**: Cero dependencias nuevas. `net/smtp` está en stdlib. Consumo de RAM adicional < 1MB (solo el render de strings del email).

---

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **SPEC-029** ya implementó el `EmailService` (`internal/services/email_service.go`) con:
  - `Send(ctx, to, subject, htmlContent)` vía `net/smtp`
  - `RenderTemplate(title, contentHTML)` con plantilla HTML única
  - `SendTest(ctx, to)` para email de prueba
  - Config SMTP y `alert_emails` en `system_settings`, protegidos (password nunca expuesto)
- **SPEC-008** ya implementó el `BillingScheduler` (`internal/services/billing_scheduler.go`):
  - `checkAndGenerate()` genera facturas `pending` para servicios con `auto_generate`
  - Deduplicación diaria con clave `last_billing_generation`
  - `generateBillForService()` crea el `Bill` con `ServiceID`, `Year`, `Month`, `Amount` (desde `svc.SuggestedAmount`) y `Status: pending`
- **Modelo `Service`** (`internal/models/service.go`) incluye: `Name`, `Institution`, `InstitutionID`, `BillingType` (`fixed`/`variable`), `Frequency` (`monthly`/`yearly`), `SuggestedAmount`, `HomeID`.
- **Modelo `Bill`** (`internal/models/bill.go`) incluye: `ServiceID`, `Year`, `Month`, `Amount`, `Status`.
- **Modelo `Institution`** (`internal/models/institution.go`) incluye: `Name`, `CategoryID`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **A: Reusar `EmailService` existente desde `BillingScheduler`** | Cero dependencias nuevas, cero tablas nuevas, patrón probado, un solo punto de envío | Acopla scheduler de billing con envío de mails | ✅ Seleccionada |
| **B: Nuevo `BillEmailScheduler` separado** | Separación de responsabilidades | Más código, duplicación del patrón de ticker, necesita propia deduplicación | ❌ Rechazada (over-engineering para iHost) |
| **C: Email resumen diario con todas las facturas** | Un solo mail por día | No coincide con el requerimiento del usuario ("un email por factura") | ❌ Rechazada |
| **D: Email también para facturas manuales** | Mayor cobertura | No coincide con el requerimiento del usuario ("solo automáticas") | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Reutilizar `EmailService` existente y enviar desde `BillingScheduler`**
- **Contexto**: SPEC-029 ya construyó el `EmailService` con plantilla única y protección de credenciales. El `BillingScheduler` ya tiene el punto exacto donde se crea cada factura automática.
- **Decisión**: Inyectar `EmailService` en `BillingScheduler`. Después de cada `billStorage.Create` exitoso en `generateBillForService`, llamar a `EmailService.Send` con un email informativo por factura.
- **Consecuencias**:
  - ✅ Cero infraestructura nueva (misma goroutine, misma deduplicación diaria `last_billing_generation`)
  - ✅ Envío atómico con generación: si el mail falla, no se revierte la factura
  - ⚠️ El scheduler debe inyectarse `EmailService`; constructor cambia

**ADR-002: Un email por factura, solo para generación automática**
- **Contexto**: El usuario pidió un correo tipo "se ha generado una nueva factura" cuando el sistema genera facturas automáticamente.
- **Decisión**: Cada factura creada por `generateBillForService` (que solo corre para `auto_generate: true`) dispara su propio email. La generación manual no dispara email (pasa por otra ruta: handler `POST /api/services/:id/generate-bill`).
- **Consecuencias**:
  - ✅ Cumple exactamente el requerimiento
  - ⚠️ Si un día se generan muchas facturas, se envían varios emails (volumen bajo, aceptable para uso personal)

**ADR-003: Destinatarios = `alert_emails` existente**
- **Contexto**: SPEC-029 ya define la lista de destinatarios de alertas en `system_settings`.
- **Decisión**: Reusar la misma lista `alert_emails` para los emails de nuevas facturas. Cero configuración nueva para el usuario.
- **Consecuencias**:
  - ✅ Sin migraciones ni nuevas claves de settings
  - ⚠️ Si en el futuro se requieren destinatarios distintos, agregar clave separada (P2)

**ADR-004: Monto variable se informa como "estimado/editable"**
- **Contexto**: Las facturas de servicios variables se generan con `SuggestedAmount` como referencia, pero el monto final puede editarse luego.
- **Decisión**: En el email, para servicios con `billing_type: variable`, indicar que el monto es variable y puede editarse. Para `fixed`, indicar monto fijo.
- **Consecuencias**:
  - ✅ El usuario sabe si el monto es definitivo o editable
  - ⚠️ El monto mostrado para variables es el estimado sugerido al momento de generación

---

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
┌──────────────────────────────────────────────────────────┐
│                BillingScheduler (existente)              │
│  checkAndGenerate() → generateBillForService()           │
└───────────┬──────────────────────────────────────────────┘
            │
            ▼  (por cada factura creada)
┌──────────────────────────────────────────────────────────┐
│  EmailService (existente, SPEC-029)                      │
│  Send(to=alert_emails, subject, RenderTemplate(title,    │
│     contentHTML))                                        │
└───────────┬──────────────────────────────────────────────┘
            │
            ▼
┌──────────────────────────────────────────────────────────┐
│  SMTP Server (config desde UI, no hardcoded)             │
└──────────────────────────────────────────────────────────┘
```

### 4.2 Componentes

#### 4.2.1 `internal/services/billing_scheduler.go` (EXTENDIDO)
- **Responsabilidad**: Añadir envío de email tras crear cada factura automática.
- **Cambios**:
  - Agregar campo `emailService *EmailService` al struct
  - `NewBillingScheduler` recibe `emailService`
  - En `generateBillForService`, después del `Create` exitoso, llamar a un nuevo helper `s.sendBillCreatedEmail(ctx, bill, svc)`
  - El helper construye título/contenido y llama `s.emailService.Send(ctx, recipients, subject, html)`
- **Comportamiento de errores**: Si `sendBillCreatedEmail` falla, loguear error (sin credenciales) y **continuar** (la factura ya está creada). No se revierte.
- **Ubicación**: `internal/services/billing_scheduler.go`

#### 4.2.2 `internal/services/bill_email.go` (NUEVO, opcional)
- **Responsabilidad**: Construir el contenido HTML del email de nueva factura. Si es pequeño, puede ir dentro de `billing_scheduler.go`; se separa para testabilidad.
- **Interfaz**:
  ```go
  func buildBillCreatedEmail(svc *models.Service, bill *models.Bill, currency string) (title, contentHTML string)
  ```
- **Ubicación**: `internal/services/bill_email.go`

#### 4.2.3 `internal/services/system_settings.go` (sin cambios)
- **Responsabilidad**: Ya expone `GetAlertEmails(ctx) ([]string, error)` (de SPEC-029). Se reutiliza tal cual.

#### 4.2.4 `internal/services/email_service.go` (sin cambios)
- **Responsabilidad**: Se reutiliza `Send`, `RenderTemplate` y `SendTest` sin modificaciones.

### 4.3 Modelo de datos

Sin cambios. Se reutilizan:
- `system_settings` → claves `alert_emails`, SMTP config, `last_billing_generation` (existente)
- Tablas `services`, `bills`, `institutions` (solo lectura)

### 4.4 Contenido del Email

**Subject**: `Nueva factura generada — {Nombre del servicio}`

**Contenido HTML** (usando la plantilla única de SPEC-029):

```html
<p>Hola,</p>

<p>
  Se ha generado una nueva factura que debes pagar en el período
  <strong>{Período}</strong> por el monto de <strong>{Monto}</strong>.
</p>

<p>
  Este monto es <strong>{fijo | variable (podés editarlo)}</strong>.
</p>

<p>
  <strong>Institución:</strong> {Nombre de la institución}<br/>
  <strong>Servicio:</strong> {Nombre del servicio}<br/>
  <strong>Período:</strong> {Mes Año | Año YYYY}<br/>
  <strong>Monto:</strong> {Monto formateado con moneda}
</p>

<p style="margin-top:24px;color:#8e8e93;font-size:13px;">
  Esta es una notificación automática del sistema de facturación.
</p>
```

Ejemplo concreto para servicio de internet fijo:

```html
<p>Hola,</p>

<p>
  Se ha generado una nueva factura que debes pagar en el período
  <strong>Agosto 2026</strong> por el monto de <strong>$1.500,00</strong>.
</p>

<p>Este monto es <strong>fijo</strong>.</p>

<p>
  <strong>Institución:</strong> Claro<br/>
  <strong>Servicio:</strong> Internet residencial<br/>
  <strong>Período:</strong> Agosto 2026<br/>
  <strong>Monto:</strong> $1.500,00
</p>
```

### 4.5 Dependencias

- **Internas**:
  - `EmailService` (existente, SPEC-029)
  - `SystemSettingsService.GetAlertEmails` (existente, SPEC-029)
  - `BillingScheduler` (modificado)
  - Modelos `Service`, `Bill`, `Institution` (solo lectura)
- **Externas**:
  - **NINGUNA nueva**. Se usa `net/smtp` de la stdlib (ya en el proyecto).

---

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Dado un servicio con `auto_generate: true` y factura creada por el `BillingScheduler`, cuando se genera la factura, entonces se envía un email informativo a los destinatarios de `alert_emails`.
- [x] CA-002: Dado el email recibido, entonces contiene saludo ("Hola"), anuncio de nueva factura, período, monto, indicación de monto fijo/variable e institución.
- [x] CA-003: Dado un servicio con `billing_type: fixed`, cuando se genera la factura, entonces el email indica "monto fijo".
- [x] CA-004: Dado un servicio con `billing_type: variable`, cuando se genera la factura, entonces el email indica que el monto es variable y editable.
- [x] CA-005: Dado un servicio con institución, cuando se genera la factura, entonces el email incluye el nombre de la institución.
- [x] CA-006: Dado un servicio anual, cuando se genera la factura, entonces el email muestra el período como "Año YYYY" (month=0).
- [x] CA-007: Dado que se genera una factura manualmente (vía `POST /api/services/:id/generate-bill`), entonces **no** se envía email.
- [x] CA-008: Dado que no hay destinatarios configurados (`alert_emails` vacío) o SMTP incompleto, cuando el scheduler genera una factura, entonces loguea un warning, **la factura se crea igualmente** y no se envía email.
- [x] CA-009: Dado que una factura ya existe para el período (deduplicación de `last_billing_generation` / `FindByServicePeriod`), cuando el scheduler corre, entonces no se reenvía email duplicado.

### 5.2 No funcionales

- [x] CA-NF-001: El binario compilado no aumenta más de 50KB (sin dependencias nuevas).
- [x] CA-NF-002: El envío de email no bloquea la generación de facturas ni el servidor HTTP.
- [x] CA-NF-003: El build compila sin errores con `go build ./...`.
- [x] CA-NF-004: `go vet ./...` no reporta errores.
- [x] CA-NF-005: `grep -rn "smtp_password\|smtp_user\|postmaster" internal/ cmd/` no encuentra credenciales hardcoded (se mantiene la protección de SPEC-029).
- [x] CA-NF-006: El email no incluye credenciales ni datos sensibles (solo datos de servicio/factura).

### 5.3 Testing

- **Unit tests**:
  - `bill_email_test.go`: `buildBillCreatedEmail` produce título/contenido correcto para fijo/variable, mensual/anual, con/sin institución.
  - `billing_scheduler_test.go`: mock de `EmailService` verifica que se llama `Send` una vez por factura creada; verifica que no se llama para facturas manuales ni para servicios sin `auto_generate`.
- **Integration tests**:
  - Flujo completo: crear servicio auto → ejecutar scheduler → verificar email enviado (mock) + factura creada.
  - SMTP no configurado → factura creada igual, warning logueado, sin email.
  - Deduplicación: ejecutar scheduler dos veces el mismo día → segundo no envía email.
- **E2E tests (manuales)**:
  - Configurar SMTP y `alert_emails` desde UI → generar factura automática → verificar recepción del email en Gmail/Apple Mail.
  - Verificar que el email se ve bien con los estilos iOS de la plantilla.
  - Generar factura manual → verificar que NO llega email.
- **Carga/Performance**: N/A (volumen de facturas por día es bajo, < 100).

---

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Helper `buildBillCreatedEmail` + unit tests (fijo/variable, mensual/anual, institución) | 0.5 día | Ninguna |
| 2 | Inyectar `EmailService` en `BillingScheduler` + envío tras crear factura + tests con mock | 0.5 día | Fase 1 |
| 3 | Validación manual local: generar factura auto → verificar email (o warning si SMTP no configurado) | 0.25 día | Fase 2 |
| 4 | Build + vet + tests completos + validación en iHost | 0.25 día | Fase 2 |

**Estimación total**: ~1.5 días

### 6.2 Milestones

1. **MVP**: Email por factura automática con monto (fijo/variable), período e institución, reutilizando `EmailService` y `alert_emails`.
2. **V1.0**: MVP + enlace a la app (P2) + logs si aplica.

---

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| SMTP mal configurado: emails no llegan | Media | Medio | Se loguea warning sin bloquear la generación. El endpoint `test-email` de SPEC-029 permite validar la config |
| Volumen de emails si hay muchos servicios automáticos | Baja | Bajo | Uso personal (iHost), volumen bajo. Si crece, migrar a email resumen (P2) |
| Monto variable mostrado como estimado puede confundir | Media | Bajo | El email aclara "este monto es variable (podés editarlo)" |
| Institución ausente en el servicio | Media | Bajo | Mostrar solo "Servicio" sin la línea de institución si no existe |
| Email HTML se ve mal en algunos clientes | Media | Bajo | Plantilla existente con estilos inline (probada en Gmail/Apple Mail en SPEC-029) |
| Scheduler intenta enviar antes de que la DB esté lista | Baja | Medio | main.go arranca scheduler después de abrir DB (ya es así con BillingScheduler) |

---

## 8. Notas y Referencias

- SPEC-029: `EmailService`, plantilla única, config SMTP y `alert_emails` (base reutilizable).
- SPEC-008: `BillingScheduler`, generación automática y deduplicación por día.
- `internal/services/email_service.go` (existente)
- `internal/services/billing_scheduler.go` (a modificar)
- `internal/models/service.go`, `internal/models/bill.go`, `internal/models/institution.go`

---

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-16 | paulomcnally | Creación inicial de la especificación. Decisiones: un email por factura, destinatarios = `alert_emails`, solo generación automática. |
| 2026-08-16 | paulomcnally | Estado a `in_progress`. Desarrollo iniciado. |
| 2026-08-16 | paulomcnally | **Released** junto con SPEC-031/032/033 (commit único del sistema de alertas multicanal). Validado en local por el usuario. |