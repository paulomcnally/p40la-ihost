---
title: "Sistema de Envío de Mails y Alertas Diarias de Seguros Vencidos"
id: "SPEC-029"
status: "pending_release"
author: "paulomcnally"
created: "2026-08-16"
updated: "2026-08-16 (v4: pending_release, CAs pass)"
github_issue: 29
---

# Sistema de Envío de Mails y Alertas Diarias de Seguros Vencidos

**ID**: SPEC-029  
**Estado**: pending_release  
**Autor**: paulomcnally  
**Creado**: 2026-08-16  
**Actualizado**: 2026-08-16 (v2: eliminadas credenciales hardcoded, reforzada seguridad)

---

## 1. Resumen Ejecutivo

El sistema p40la-ihost actualmente gestiona autos y sus seguros asociados (vía `auto_services`), pero no existe ningún mecanismo que alerte al usuario cuando un auto queda sin seguro o cuando una póliza vence sin que haya una nueva activa. Esto genera riesgo de vehículos circulando sin cobertura, sin que el usuario se entere hasta que revisa manualmente la aplicación.

Esta spec introduce un **sistema de envío de mails informativos** con una **plantilla HTML única** (misma apariencia, título y contenido variables) y un **scheduler diario** que verifica el estado de los seguros de todos los autos registrados. Cuando detecta autos sin seguro o con pólizas vencidas sin reemplazo activo, envía un email de alerta a una lista de destinatarios configurables desde Settings.

La configuración SMTP (host, puerto, usuario, password, remitente) se configura **exclusivamente desde la UI de Settings** y se almacena en la tabla existente `system_settings` (SQLite key-value). **No se incluyen credenciales hardcoded ni defaults en el código.** El password SMTP es información sensible: **nunca se expone** en responses de la API ni en logs. El envío se realiza con `net/smtp` de la stdlib de Go — **cero dependencias nuevas**, alineado con las restricciones de memoria del iHost.

**Consideraciones de iHost**: No se agregan dependencias externas (se usa `net/smtp` de stdlib). No se crean tablas nuevas (se reutiliza `system_settings`). El scheduler sigue el mismo patrón del `BillingScheduler` existente (ticker horario + deduplicación por fecha). El consumo adicional de RAM es despreciable.

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: El sistema debe enviar emails vía SMTP usando `net/smtp` de la stdlib de Go, sin dependencias externas nuevas.
2. **REQ-002**: Debe existir una **única plantilla HTML** para todos los emails. La plantilla tiene la misma apariencia siempre; solo varían el **título** y el **contenido** (cuerpo HTML).
3. **REQ-003**: La configuración SMTP (host, port, user, password, from_email, from_name) debe guardarse en la tabla `system_settings` de SQLite y ser editable desde la UI de Settings.
4. **REQ-004**: La lista de emails destinatarios de alertas debe guardarse en `system_settings` (clave `alert_emails`, valor comma-separated) y ser editable desde la UI de Settings.
5. **REQ-005**: Debe existir un **AlertScheduler** que ejecute diariamente a la hora configurada en `billing_generation_hour` (ya existente en system_settings), verificando:
   - **REQ-005a**: Autos que **no tienen ningún servicio de seguro asociado** (sin registro en `auto_services`).
   - **REQ-005b**: Autos cuyos seguros asociados tienen `end_date` vencida (fecha < hoy) **y** no existe otro seguro activo para el mismo auto (otro `auto_services` cuyo servicio tenga `end_date` NULL o >= hoy).
6. **REQ-006**: Cuando el AlertScheduler detecta autos en las condiciones de REQ-005, debe enviar un único email de alerta a todos los destinatarios configurados, listando los autos afectados con el motivo (sin seguro / seguro vencido).
7. **REQ-007**: El AlertScheduler debe usar deduplicación por fecha (clave `last_alert_check` en system_settings) para no enviar alertas duplicadas el mismo día.
8. **REQ-008**: Si no hay autos en condición de alerta, no se envía ningún email.
9. **REQ-009**: Si la configuración SMTP no está completa (falta host, user, password, o from_email), el scheduler debe loguear un warning y no intentar el envío.
10. **REQ-010**: Debe existir un endpoint `POST /api/system-settings/test-email` que envíe un email de prueba a los destinatarios configurados, para validar la configuración SMTP desde la UI.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-011**: La UI de Settings debe tener una sección "Alertas por Email" con campos para: destinatarios (lista editable), y un botón "Enviar email de prueba".
2. **REQ-012**: La UI de Settings debe tener una sección "Configuración SMTP" con campos para: host, port, user, password (tipo password), from_email, from_name.
3. **REQ-013**: El email de alerta debe incluir para cada auto: marca, modelo, año, placa, y motivo de la alerta (Sin seguro / Seguro vencido el DD/MM/YYYY).
4. **REQ-014**: El password SMTP debe mostrarse enmascarado (type="password") en la UI, con un botón para mostrar/ocultar.
5. **REQ-015**: **No se incluyen credenciales SMTP hardcoded ni defaults en el código.** Toda configuración SMTP se ingresa manualmente desde la UI. El sistema funciona en modo "no configurado" hasta que el usuario complete los campos.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-016**: Log de emails enviados en una tabla `email_logs` (timestamp, subject, recipients, status, error). Útil para debugging.
2. **REQ-017**: Posibilidad de configurar una hora independiente para el check de alertas (`alert_check_hour`), separada de `billing_generation_hour`.
3. **REQ-018**: Incluir en el email de alerta un enlace directo a la app (ej: `http://<ihost-ip>:8088/autos/<id>`) para acceso rápido.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: El envío de emails no debe bloquear el servidor HTTP. El AlertScheduler corre en una goroutine separada (igual que BillingScheduler).
- **Seguridad — Protección de información sensible**:
  - El password SMTP es información sensible y **NUNCA** se expone en: responses de la API (GET), logs (slog), mensajes de error, ni traces. Las queries de storage y servicios deben enmascarar el password al loguear.
  - Las credenciales SMTP se almacenan en SQLite (`system_settings`). Esto es aceptable porque: (a) el iHost es un dispositivo local sin acceso externo, (b) es un sistema single-user. Ver ADR-001 para análisis.
  - **No se guardan credenciales en el código fuente ni en el repositorio.** Toda config se ingresa via UI en runtime.
  - El endpoint `GET /api/system-settings` devuelve un flag booleano `smtp_configured` (indica si hay password guardado) pero **nunca** devuelve el valor del password ni del user.
- **Almacenamiento**: Sin tablas nuevas para P0/P1 (se reutiliza `system_settings`). P2 agrega `email_logs` (~1KB por email, sin info sensible en error).
- **Disponibilidad**: Si el servidor SMTP no responde, el scheduler loguea el error (sin credenciales) y reintenta al día siguiente. No hay cola de reintentos (mantener simple para iHost).
- **iHost**: Cero dependencias nuevas. `net/smtp` está en stdlib. Consumo de RAM adicional < 1MB (solo la goroutine del scheduler + string del template).

---

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Sistema P4OLA existente** (`/home/paulomcnally/P4OLA/scripts/send-email.sh`): Usa Mailgun SMTP (`smtp.mailgun.org:587`) con credenciales hardcoded en el script. Envía HTML + texto plano vía Python `smtplib`. From: `P4OLA <postmaster@paulomcnally.com>`. **Este patrón (credenciales en código) NO se replica en p40la-ihost** — las credenciales se configuran exclusivamente desde la UI.
- **Sistema P4OLA NestJS** (`.claude/EMAIL_RECOVERY.md`): Usa variables de entorno (`SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM`, `SMTP_REPLY_TO`), BullMQ/Redis para cola de emails, tabla `notification_logs` para tracking. Modo DRY RUN si no hay credenciales.
- **Go stdlib `net/smtp`**: Disponible desde Go 1.0. Soporta STARTTLS, AUTH PLAIN/LOGIN. Suficiente para envío simple vía Mailgun. No requiere dependencias externas.
- **Patrón de scheduler existente** (`internal/services/billing_scheduler.go`): Ticker horario, compara `now.Hour()` con setting, deduplicación por fecha. Patrón probado y funcionando.
- **Tabla `system_settings`** (`internal/storage/system_settings.go`): Key-value con UPSERT. Ya usada para `billing_generation_hour`. Perfecto para guardar config SMTP sin migraciones.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **A: `net/smtp` stdlib + config en SQLite (sin hardcoded)** | Cero dependencias, configurable desde UI, cero credenciales en repositorio, user/password no se exponen en API ni logs | Password en texto plano en SQLite; requiere config manual inicial | ✅ Seleccionada |
| **B: `net/smtp` stdlib + password en env var** | Password no en DB | No configurable desde UI sin restart, complejidad extra, user sigue en DB | ❌ Rechazada (alternativa documentada en ADR-001) |
| **C: `gopkg.in/mail.v2` (lib externa)** | API más ergonómica, soporte attachments | Dependencia nueva, más RAM, innecesario para emails simples | ❌ Rechazada |
| **D: Llamar script `send-email.sh` de P4OLA** | Reutiliza código existente | Dependencia de P4OLA en el mismo host (iHost no lo tiene), requiere Python en runtime | ❌ Rechazada |
| **E: Webhook a API externa (Mailgun HTTP API)** | Sin credenciales SMTP locales | Requiere dependencia HTTP client + JSON parsing, más complejo que SMTP | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Almacenar credenciales SMTP en SQLite (`system_settings`) — sin defaults en código**
- **Contexto**: El usuario necesita configurar SMTP desde la UI sin reiniciar el servicio. El iHost es local, single-user, sin acceso externo. El sistema P4OLA tiene credenciales hardcoded, pero **ese patrón NO se replica aquí**: p40la-ihost no incluye credenciales en el código fuente ni en el repositorio.
- **Decisión**: Guardar host, port, user, password, from_email, from_name en `system_settings` (ingresados manualmente via UI). Sin defaults, sin seed, sin hardcoded. El password se trata como información sensible: no se devuelve en GET, no se loguea.
- **Consecuencias**:
  - ✅ Configurable desde UI sin restart
  - ✅ Simple, sin migraciones (tabla key-value ya existe)
  - ✅ Consistente con el contexto local del iHost
  - ✅ **Cero credenciales en el repositorio** — seguro para control de versiones
  - ✅ Password protegido: no expuesto en API responses ni logs
  - ⚠️ Password en texto plano en SQLite (aceptable dado contexto local single-user)
  - ⚠️ Requiere configuración manual inicial antes de que funcionen las alertas
  - **Alternativa documentada**: Si en el futuro se requiere mayor seguridad, mover `smtp_password` a env var `SMTP_PASS` y leerlo en `config.go`. El resto puede quedar en SQLite.

**ADR-002: Reutilizar `billing_generation_hour` como hora del AlertScheduler**
- **Contexto**: El usuario indicó "la hora de envío ya está configurada en configuraciones". Existe `billing_generation_hour` en `system_settings`.
- **Decisión**: El AlertScheduler usa `billing_generation_hour` como hora de ejecución. Se ejecuta a la misma hora que la generación de facturas.
- **Consecuencias**:
  - ✅ Cero configuración extra para el usuario
  - ✅ Reutiliza infraestructura existente
  - ⚠️ Si se necesita hora independiente en el futuro, agregar `alert_check_hour` (P2/REQ-017)

**ADR-003: Usar `net/smtp` de stdlib en lugar de lib externa**
- **Contexto**: El iHost tiene recursos limitados. Las dependencias externas aumentan el tamaño del binario y el consumo de RAM.
- **Decisión**: Usar `net/smtp` de la stdlib de Go. Construir el mensaje MIME manualmente (headers + boundaries) para soportar HTML + texto plano.
- **Consecuencias**:
  - ✅ Cero dependencias nuevas, cero aumento de tamaño del binario
  - ✅ Control total sobre el formato del mensaje
  - ⚠️ Código más verbose para construir MIME multipart (but trivial, ~30 líneas)

**ADR-004: Plantilla HTML única embebida en código Go**
- **Contexto**: Los emails son informativos y comparten apariencia. Solo varían título y contenido.
- **Decisión**: Definir la plantilla HTML como un string constant en `internal/services/email_service.go`. Usar `strings.ReplaceAll` para inyectar título y contenido.
- **Consecuencias**:
  - ✅ Sin dependencia de motor de templates
  - ✅ Sin archivos externos que cargar desde disco
  - ✅ Cambios a la plantilla requieren recompilar (aceptable, es infrecuente)

---

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
┌─────────────────────────────────────────────────────────┐
│                    cmd/server/main.go                     │
│  Arranca AlertScheduler + EmailService (goroutines)       │
└───────────┬─────────────────────────────────┬───────────┘
            │                                 │
            ▼                                 ▼
┌──────────────────────┐         ┌──────────────────────┐
│   AlertScheduler     │         │    EmailService       │
│  (ticker horario)    │         │  (net/smtp stdlib)    │
│                      │         │                       │
│  1. Lee hour setting │         │  Send(to, subject,    │
│  2. Verifica fecha   │────────▶│    htmlContent)       │
│  3. Query autos      │         │                       │
│  4. Renderiza email  │         │  RenderTemplate(      │
│  5. EmailService.Send│         │    title, content)    │
└──────────┬───────────┘         └──────────┬───────────┘
           │                                │
           ▼                                ▼
┌──────────────────────┐         ┌──────────────────────┐
│  AutoStorage         │         │  SMTP Server          │
│  (queries de alerta) │         │  (configurable desde  │
│                      │         │   UI, no hardcoded)   │
│  - ListWithoutInsurance│        │                       │
│  - ListWithExpiredInsurance    └──────────────────────┘
└──────────┬───────────┘
           │
           ▼
┌──────────────────────┐
│  SystemSettingsStorage│
│  (SMTP config,        │
│   alert_emails,       │
│   last_alert_check)   │
└──────────────────────┘
```

### 4.2 Componentes

#### 4.2.1 `internal/services/email_service.go` (NUEVO)
- **Responsabilidad**: Enviar emails vía SMTP usando stdlib. Renderizar plantilla HTML única.
- **Interfaz**:
  ```go
  type EmailService struct { ... }
  func NewEmailService(settings *SystemSettingsService) *EmailService
  func (s *EmailService) Send(ctx context.Context, to []string, subject, htmlContent string) error
  func (s *EmailService) RenderTemplate(title, contentHTML string) string
  func (s *EmailService) IsConfigured(ctx context.Context) (bool, error)
  ```
- **Dependencias**: `SystemSettingsService` (lee config SMTP)
- **Ubicación**: `internal/services/email_service.go`

#### 4.2.2 `internal/services/alert_scheduler.go` (NUEVO)
- **Responsabilidad**: Scheduler diario que verifica seguros de autos y envía alertas por email.
- **Interfaz**:
  ```go
  type AlertScheduler struct { ... }
  func NewAlertScheduler(autoStorage *storage.AutoStorage, autoServiceStorage *storage.AutoServiceStorage, emailService *EmailService, settingsService *SystemSettingsService) *AlertScheduler
  func (s *AlertScheduler) Start()
  func (s *AlertScheduler) Stop()
  ```
- **Patrón**: Idéntico a `BillingScheduler` (ticker horario, check hora, deduplicación por fecha).
- **Dependencias**: `AutoStorage`, `AutoServiceStorage`, `EmailService`, `SystemSettingsService`
- **Ubicación**: `internal/services/alert_scheduler.go`

#### 4.2.3 `internal/storage/auto_storage.go` (EXTENDIDO)
- **Responsabilidad**: Agregar queries para detectar autos sin seguro o con seguro vencido.
- **Nuevos métodos**:
  ```go
  func (s *AutoStorage) ListWithoutInsurance(ctx context.Context) ([]models.AutoAlert, error)
  func (s *AutoStorage) ListWithExpiredInsurance(ctx context.Context) ([]models.AutoAlert, error)
  ```
- **Ubicación**: Extender `internal/storage/auto_storage.go`

#### 4.2.4 `internal/models/auto.go` (EXTENDIDO)
- **Nuevo struct**:
  ```go
  type AutoAlert struct {
      Auto        Auto    `json:"auto"`
      AlertType   string  `json:"alert_type"`  // "no_insurance" | "expired"
      EndDate     string  `json:"end_date,omitempty"` // solo para "expired"
      ServiceName string  `json:"service_name,omitempty"` // nombre del seguro vencido
  }
  ```

#### 4.2.5 `internal/services/system_settings.go` (EXTENDIDO)
- **Nuevos métodos**:
  ```go
  func (s *SystemSettingsService) GetSMTPConfig(ctx context.Context) (*models.SMTPConfig, error)        // interno, incluye user+password
  func (s *SystemSettingsService) GetSMTPConfigPublic(ctx context.Context) (*models.SMTPConfigPublic, error) // para API, SIN user ni password
  func (s *SystemSettingsService) SetSMTPConfig(ctx context.Context, cfg *models.SMTPConfig) error
  func (s *SystemSettingsService) IsSMTPConfigured(ctx context.Context) (bool, error)
  func (s *SystemSettingsService) GetAlertEmails(ctx context.Context) ([]string, error)
  func (s *SystemSettingsService) SetAlertEmails(ctx context.Context, emails []string) error
  func (s *SystemSettingsService) GetAlertCheckHour(ctx context.Context) (int, error) // fallback a billing_generation_hour
  ```
- **Notas de seguridad**: `GetSMTPConfig` (con user+password) solo se usa internamente por `EmailService` para enviar emails. `GetSMTPConfigPublic` (sin user ni password) se usa en los handlers de API para el GET. **Ningún log debe incluir user ni password.**

#### 4.2.6 `internal/models/smtp_config.go` (NUEVO)
- **Responsabilidad**: Structs para configuración SMTP.
  ```go
  // SMTPConfig — completo, uso interno. NUNCA se serializa a JSON para la API.
  type SMTPConfig struct {
      Host       string
      Port       int
      User       string
      Password   string
      FromEmail  string
      FromName   string
  }

  // SMTPConfigPublic — versión segura para API responses. Sin credenciales.
  type SMTPConfigPublic struct {
      Host          string `json:"smtp_host"`
      Port          int    `json:"smtp_port"`
      User          string `json:"smtp_user"`           // siempre "" en responses
      FromEmail     string `json:"smtp_from_email"`
      FromName      string `json:"smtp_from_name"`
      Configured    bool   `json:"smtp_configured"`     // true si user+password están set
  }
  ```

#### 4.2.7 `internal/api/system_settings_handlers.go` (EXTENDIDO)
- **Responsabilidad**: Exponer SMTP config y alert emails via API. Endpoint de test email.
- **Cambios en `GetSystemSettings`**: Agregar `smtp_host`, `smtp_port`, `smtp_from_email`, `smtp_from_name`, `smtp_configured`, `alert_emails`. **`smtp_user` y `smtp_password` NUNCA se devuelven** (se omiten o van como "").
- **Cambios en `UpdateSystemSettings`**: Aceptar campos SMTP y `alert_emails`. Si `smtp_user` o `smtp_password` vienen vacíos/ausentes, se mantiene el valor actual.
- **Nuevo endpoint**: `POST /api/system-settings/test-email`
- **Logs**: Los handlers y servicios deben loguear solo `smtp_host` y `smtp_configured`, **nunca** user ni password.

### 4.3 Modelo de datos

No se crean tablas nuevas (P0/P1). Se usan claves nuevas en `system_settings` existente:

```
Tabla: system_settings (existente, key-value)

Claves nuevas:
- smtp_host         TEXT   -- ej: "smtp.example.com"
- smtp_port         TEXT   -- ej: "587"
- smtp_user         TEXT   -- SENSIBLE: ej: "postmaster@example.com"
- smtp_password     TEXT   -- SENSIBLE: ej: "<smtp-api-key>"
- smtp_from_email   TEXT   -- ej: "alerts@example.com"
- smtp_from_name    TEXT   -- ej: "P40LA"
- alert_emails      TEXT   -- ej: "paulo@example.com,admin@example.com"
- last_alert_check  TEXT   -- ej: "2026-08-16" (deduplicación diaria)
```

Para P2 (REQ-016), tabla opcional:

```
Tabla: email_logs (P2 - opcional)
- id          INTEGER PRIMARY KEY
- subject     TEXT
- recipients  TEXT      -- comma-separated
- status      TEXT      -- "sent" | "failed"
- error       TEXT      -- NULL si success
- created_at  TIMESTAMP
```

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/system-settings` (EXTENDIDO)

**Response 200** (campos nuevos marcados con ★):
```json
{
  "billing_generation_hour": 8,
  "smtp_host": "smtp.example.com",             // ★
  "smtp_port": 587,                            // ★
  "smtp_user": "",                             // ★ NUNCA se devuelve (info sensible)
  "smtp_from_email": "alerts@example.com",     // ★
  "smtp_from_name": "P40LA",                   // ★
  "smtp_configured": true,                     // ★ (bool: indica si user+password están set)
  "alert_emails": "paulo@example.com"          // ★
}
```

> **Seguridad**: `smtp_user` y `smtp_password` son información sensible (credenciales de autenticación SMTP) y **NUNCA** se devuelven en el GET. Se envían como string vacío. La UI usa `smtp_configured` (bool) para saber si ya hay credenciales guardadas. Los campos `smtp_host`, `smtp_port`, `smtp_from_email`, `smtp_from_name` y `alert_emails` no son sensibles (son info de conexión/remitente, no de autenticación) y sí se devuelven para mostrar en la UI.

#### Endpoint: `PUT /api/system-settings` (EXTENDIDO)

**Request** (campos opcionales, solo los que se actualizan):
```json
{
  "billing_generation_hour": 8,
  "smtp_host": "smtp.example.com",
  "smtp_port": 587,
  "smtp_user": "<smtp-user>",            // info sensible, solo se envía para guardar
  "smtp_password": "<smtp-password>",    // info sensible, solo se envía si se cambia; si no se envía se mantiene el actual
  "smtp_from_email": "alerts@example.com",
  "smtp_from_name": "P40LA",
  "alert_emails": "paulo@example.com,admin@example.com"
}
```

> **Seguridad**: `smtp_user` y `smtp_password` solo viajan del cliente al servidor en el PUT para ser guardados. Nunca se devuelven en GET. Si `smtp_user` o `smtp_password` no se envían en el PUT (campos ausentes o vacíos), se mantiene el valor actual guardado en SQLite.

**Response 200**:
```json
{
  "billing_generation_hour": 8,
  "smtp_configured": true,
  "message": "Configuración actualizada"
}
```

#### Endpoint: `POST /api/system-settings/test-email` (NUEVO)

**Request**: vacío (envía a los `alert_emails` configurados)

**Response 200**:
```json
{
  "message": "Email de prueba enviado",
  "recipients": "paulo@example.com"
}
```

**Response 400** (SMTP no configurado o sin destinatarios):
```json
{
  "error": "smtp_not_configured",
  "message": "Configure SMTP y destinatarios antes de probar"
}
```

### 4.5 Plantilla de Email

HTML único embebido en `email_service.go`. Diseño limpio, iOS-inspired, compatible con clientes de email:

```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="margin:0;padding:0;background-color:#f5f5f7;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;">
  <table width="100%" cellpadding="0" cellspacing="0" style="background-color:#f5f5f7;padding:24px 0;">
    <tr>
      <td align="center">
        <table width="600" cellpadding="0" cellspacing="0" style="background-color:#ffffff;border-radius:16px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.08);">
          <!-- Header -->
          <tr>
            <td style="background-color:#007aff;padding:24px 32px;">
              <h1 style="margin:0;color:#ffffff;font-size:22px;font-weight:700;">P40LA</h1>
            </td>
          </tr>
          <!-- Title -->
          <tr>
            <td style="padding:32px 32px 0 32px;">
              <h2 style="margin:0;color:#1d1d1f;font-size:20px;font-weight:600;">{{TITLE}}</h2>
            </td>
          </tr>
          <!-- Content -->
          <tr>
            <td style="padding:16px 32px 32px 32px;color:#1d1d1f;font-size:15px;line-height:1.6;">
              {{CONTENT}}
            </td>
          </tr>
          <!-- Footer -->
          <tr>
            <td style="padding:16px 32px;border-top:1px solid #e5e5ea;background-color:#fafafa;">
              <p style="margin:0;color:#8e8e93;font-size:12px;">
                P40LA iHost — Alerta automática generada el {{DATE}}
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>
```

**Placeholders**: `{{TITLE}}`, `{{CONTENT}}`, `{{DATE}}` — reemplazados via `strings.ReplaceAll`.

### 4.6 Contenido del Email de Alerta

El `{{CONTENT}}` del email de alerta se construye dinámicamente:

```html
<p>Se detectaron los siguientes vehículos que requieren atención:</p>

<table width="100%" cellpadding="8" cellspacing="0" style="border-collapse:collapse;margin-top:16px;">
  <tr style="background-color:#f5f5f7;">
    <th style="text-align:left;padding:12px;border-bottom:2px solid #e5e5ea;font-size:13px;">Vehículo</th>
    <th style="text-align:left;padding:12px;border-bottom:2px solid #e5e5ea;font-size:13px;">Placa</th>
    <th style="text-align:left;padding:12px;border-bottom:2px solid #e5e5ea;font-size:13px;">Motivo</th>
  </tr>
  <tr>
    <td style="padding:12px;border-bottom:1px solid #e5e5ea;">Toyota Corolla 2020</td>
    <td style="padding:12px;border-bottom:1px solid #e5e5ea;">ABC-123</td>
    <td style="padding:12px;border-bottom:1px solid #e5e5ea;color:#ff3b30;">Sin seguro asociado</td>
  </tr>
  <tr>
    <td style="padding:12px;border-bottom:1px solid #e5e5ea;">Honda Civic 2019</td>
    <td style="padding:12px;border-bottom:1px solid #e5e5ea;">XYZ-789</td>
    <td style="padding:12px;border-bottom:1px solid #e5e5ea;color:#ff9500;">Seguro vencido el 10/08/2026</td>
  </tr>
</table>

<p style="margin-top:24px;color:#8e8e93;font-size:13px;">
  Ingresá a <a href="http://ihost:8088/autos" style="color:#007aff;">P40LA</a> para revisar y asociar un nuevo seguro.
</p>
```

### 4.7 Dependencias

- **Internas**: 
  - `SystemSettingsService` / `SystemSettingsStorage` (extendido)
  - `AutoStorage` (extendido con queries de alerta)
  - `AutoServiceStorage` (para queries de seguros asociados)
  - `BillingScheduler` (patrón a seguir, no se modifica)
- **Externas**: 
  - **NINGUNA nueva**. Se usa `net/smtp` y `crypto/tls` de la stdlib de Go.

---

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Dado un auto sin ningún seguro asociado, cuando el AlertScheduler ejecuta, entonces el auto aparece en el email de alerta con motivo "Sin seguro asociado".
- [x] CA-002: Dado un auto con un seguro cuya `end_date` es anterior a hoy, y no tiene otro seguro activo, cuando el AlertScheduler ejecuta, entonces el auto aparece en el email con motivo "Seguro vencido el DD/MM/YYYY".
- [x] CA-003: Dado un auto con un seguro vencido PERO con otro seguro activo (end_date NULL o >= hoy), cuando el AlertScheduler ejecuta, entonces el auto NO aparece en el email.
- [x] CA-004: Dado que no hay autos en condición de alerta, cuando el AlertScheduler ejecuta, entonces no se envía ningún email.
- [x] CA-005: Dado que el AlertScheduler ya ejecutó hoy (last_alert_check = hoy), cuando se llama nuevamente, entonces no se envía email duplicado.
- [x] CA-006: Dado que la config SMTP está incompleta (falta password), cuando el AlertScheduler intenta enviar, entonces loguea un warning y no envía.
- [x] CA-007: Dado la config SMTP completa, cuando se llama `POST /api/system-settings/test-email`, entonces se envía un email de prueba a los destinatarios configurados.
- [x] CA-008: Dado la UI de Settings, cuando el usuario edita los campos SMTP y destinatarios, entonces se guardan en SQLite y persisten tras reiniciar.
- [x] CA-009: Dado el email recibido, cuando se abre en un cliente de email, entonces muestra la plantilla P40LA con header azul, título, tabla de autos y footer.
- [x] CA-010: Dado que se hace `GET /api/system-settings`, entonces la respuesta **NO incluye** `smtp_user` ni `smtp_password` (campos ausentes o string vacío). Solo incluye `smtp_configured` (bool).
- [x] CA-011: Dado que se hace `PUT /api/system-settings` sin enviar `smtp_user` o `smtp_password` (campos ausentes), entonces se mantienen los valores actuales guardados en SQLite (no se sobrescriben con vacío).
- [x] CA-012: Dado que se buscan credenciales SMTP en el código fuente (grep de `smtp_password`, `postmaster`, API keys), entonces **no se encuentran** credenciales hardcoded en ningún archivo del repositorio.
- [x] CA-013: Dado que el AlertScheduler o EmailService encuentran un error SMTP, cuando se loguea con `slog`, entonces el log **no contiene** `smtp_user` ni `smtp_password` (verificar con test que captura logs).

### 5.2 No funcionales

- [x] CA-NF-001: El binario compilado no aumenta más de 50KB (sin dependencias nuevas).
- [x] CA-NF-002: El AlertScheduler no bloquea el servidor HTTP (corre en goroutine).
- [x] CA-NF-003: El consumo de RAM adicional es < 1MB en idle.
- [x] CA-NF-004: El build compila sin errores con `go build ./...`.
- [x] CA-NF-005: `go vet ./...` no reporta errores.
- [x] CA-NF-006: `grep -rn "smtp_password\|smtp_user\|postmaster\|0bc52df2" internal/ cmd/` no encuentra credenciales hardcoded en el código fuente.

### 5.3 Testing

- **Unit tests**:
  - `email_service_test.go`: RenderTemplate con título/contenido variables, validación de placeholders reemplazados.
  - `alert_scheduler_test.go`: Lógica de detección de autos sin seguro / con seguro vencido (mock storage).
  - `system_settings_test.go`: Get/Set SMTP config, Get/Set alert_emails, parseo de comma-separated. **Verificar que GetSMTPConfigPublic no incluye user ni password.**
  - `security_test.go`: **Test que captura logs de slog y verifica que smtp_user y smtp_password no aparecen en ningún mensaje de log** (incluyendo errores SMTP).
- **Integration tests**:
  - Flujo completo: insertar auto sin seguro → ejecutar scheduler → verificar que se llama EmailService.Send.
  - Deduplicación: ejecutar scheduler dos veces el mismo día → segundo no envía.
  - **PUT sin user/password → valores actuales se mantienen (no se sobrescriben con vacío).**
  - **GET → response no contiene smtp_user ni smtp_password.**
- **E2E tests (manuales)**:
  - Configurar SMTP desde UI → enviar test email → verificar recepción.
  - Crear auto sin seguro → esperar ejecución del scheduler → verificar email de alerta.
  - Email se ve correctamente en Gmail/Apple Mail.
  - **Verificar que `grep -rn "postmaster\|0bc52df2\|smtp_password" internal/ cmd/` no encuentra credenciales en código.**
- **Carga/Performance**: N/A (volumen de autos es bajo, < 100).

---

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Modelo `SMTPConfig` + extender `SystemSettingsService` y `SystemSettingsStorage` con getters/setters SMTP y alert_emails | 0.5 día | Ninguna |
| 2 | `EmailService` con `net/smtp`: Send, RenderTemplate, IsConfigured. Plantilla HTML embebida. Unit tests de RenderTemplate. | 1 día | Fase 1 |
| 3 | Extender `AutoStorage` con queries `ListWithoutInsurance` y `ListWithExpiredInsurance`. Modelo `AutoAlert`. Unit tests de queries. | 0.5 día | Ninguna |
| 4 | `AlertScheduler`: ticker, deduplicación, lógica de detección, render de email de alerta. Unit tests con mocks. | 1 día | Fases 2, 3 |
| 5 | Extender `system_settings_handlers.go`: GET/PUT con campos SMTP + alert_emails. Endpoint `POST /test-email`. | 0.5 día | Fase 1 |
| 6 | Integrar `EmailService` y `AlertScheduler` en `main.go` (Start/Stop). | 0.25 día | Fases 2, 4 |
| 7 | Frontend: sección "Configuración SMTP" + sección "Alertas por Email" en SettingsPage. API client. i18n. | 1 día | Fase 5 |
| 8 | Tests de integración + build + vet. Pruebas manuales locales. | 0.5 día | Todas |

**Estimación total**: ~4.75 días

### 6.2 Milestones

1. **MVP**: EmailService + AlertScheduler + API + integración en main.go. Sin UI (configurable via API/curl). Verificación manual con curl.
2. **V1.0**: MVP + UI de Settings completa + tests de integración. Sin seed de defaults — el usuario configura SMTP manualmente al primer uso.

---

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| SMTP mal configurado por el usuario (host/puerto incorrectos) | Media | Medio | Endpoint de test-email para validar antes de habilitar alertas. Logs con errores específicos (sin exponer credenciales) |
| Email HTML se ve mal en algunos clientes | Media | Bajo | Plantilla simple con estilos inline (compatibilidad máxima), probar en Gmail/Apple Mail |
| iHost sin conexión a internet al ejecutar scheduler | Media | Bajo | Scheduler loguea error (sin credenciales) y reintenta día siguiente. No hay cola (mantener simple) |
| Password y user SMTP en texto plano en SQLite | Baja | Bajo | Aceptable en contexto local single-user. ADR-001 documenta alternativa via env var. No se exponen en API ni logs |
| Scheduler ejecuta antes de que la DB esté lista | Baja | Medio | main.go arranca scheduler después de abrir DB (igual que BillingScheduler) |
| Múltiples destinatarios con uno inválido rechaza todo el envío | Media | Medio | Usar `smtp.SendMail` con múltiples RCPT TO. Si uno falla, loguear y continuar |
| Credenciales sensibles filtradas en logs por error de programación | Media | Alto | Code review enfocado en seguridad. Regla explícita: slog nunca recibe user ni password. Tests que verifican ausencia de credenciales en logs |

---

## 8. Notas y Referencias

- Script de email P4OLA (referencia de patrón SMTP, **NO se replican credenciales**): `/home/paulomcnally/P4OLA/scripts/send-email.sh`
- Doc de recuperación de emails P4OLA: `/home/paulomcnally/P4OLA/.claude/EMAIL_RECOVERY.md`
- Scheduler existente (patrón a seguir): `internal/services/billing_scheduler.go`
- Storage de system settings: `internal/storage/system_settings.go`
- Go stdlib `net/smtp`: https://pkg.go.dev/net/smtp
- Go stdlib `crypto/tls`: https://pkg.go.dev/crypto/tls (para STARTTLS)
- MIME multipart message format: RFC 2045/2046

> **IMPORTANTE**: Las credenciales SMTP (user y password) se configuran exclusivamente desde la UI de Settings. No existen defaults hardcoded, no hay seed automático, y no se incluyen credenciales en el código fuente ni en este documento.

---

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-16 | paulomcnally | Creación inicial de la especificación |
| 2026-08-16 | paulomcnally | Eliminadas credenciales hardcoded de Mailgun. Eliminado seed de defaults (REQ-015 original). ADR-001 actualizado: cero credenciales en repositorio. Reforzada protección de info sensible: smtp_user y smtp_password nunca se exponen en API ni logs. Agregados SMTPConfigPublic (sin credenciales) y métodos separados Get/GetPublic. Nuevos CAs sobre no-exposición en logs. |
| 2026-08-16 | paulomcnally | Estado a `in_progress`. Desarrollo iniciado en rama `feature/SPEC-029-email-alerts`. |
| 2026-08-16 | paulomcnally | Desarrollo completado: EmailService, AlertScheduler, queries de alerta, API handlers, frontend Settings UI, tests. Criterios de aceptación verificados en local (tests + validación manual del usuario). Estado a `pending_release`. |
