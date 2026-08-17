---
title: "Alertas multicanal con Voice Monkey (Alexa): sistema robusto de alertas con formato mail y voz"
id: "SPEC-033"
status: "pending_release"
author: "paulomcnally"
created: "2026-08-16"
updated: "2026-08-16 (pending_release)"
github_issue: 33
---

# Alertas multicanal con Voice Monkey (Alexa): sistema robusto de alertas con formato mail y voz

**ID**: SPEC-033  
**Estado**: pending_release  
**Autor**: paulomcnally  
**Creado**: 2026-08-16  
**Actualizado**: 2026-08-16 (pending_release)

---

## 1. Resumen Ejecutivo

El sistema p40la-ihost envía hoy alertas únicamente por **email**: SPEC-029 (seguros de autos vencidos), SPEC-030 (email al generar factura automática) y SPEC-031 (resumen diario de facturas pendientes). SPEC-032 (draft) agrega la sección "Alertas" en Configuraciones con toggles opt-in por funcionalidad de mail. El usuario quiere además que **cada alerta pueda anunciarse por voz en un altavoz Alexa** usando [Voice Monkey](https://www.voicemonkey.io/) (API TTS), con un toggle independiente del de mail.

Esta spec agrega el **canal de voz (Alexa)** al sistema de alertas y lo hace **más robusto y extensible**: introduce una **tabla interna `alerts`** (SQLite, sembrada desde código Go) que es la fuente de verdad sobre *qué alerta es cuál* y *qué formatos/canales tiene* (`mail_enabled`, `voice_enabled`). Esto permite escalar a futuros canales (WhatsApp, Telegram, etc.) simplemente agregando un campo/flag y un servicio de envío, sin reestructurar los schedulers.

En Settings se agrega una sección **"Voice Monkey (Alexa)"** que está **inactiva por defecto** (opt-in). Al activarla, se permite configurar un **API token** y un **device**, más un **toggle "enviar alertas"**. Cada alerta de la sección "Alertas" (SPEC-032) tendrá dos toggles: **Mail** y **Alexa**, independientes. Cuando una alerta está activa para voz y Voice Monkey está configurado, el scheduler correspondiente anuncia un **texto corto TTS** (speech) con la voz "Lucia" en español.

**Consideraciones de iHost**: Una tabla SQLite nueva pequeña (`alerts`, 3 filas). El servicio de voz usa **solo stdlib** (`net/http`, `encoding/json`). Consumo de RAM/CPU despreciable (una llamada HTTP de 10s máxima por alerta). El token de Voice Monkey se trata como información sensible: **nunca** se expone en API ni logs.

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Debe existir una sección **"Voice Monkey (Alexa)"** en Settings, con un toggle maestro **inactivo por defecto**. Al activarlo se muestran: campo **API token**, campo **Device** y un toggle **"Enviar alertas"** (envío de alertas por voz).
2. **REQ-002**: Cada alerta de SPEC-032 (seguros de autos, factura generada, resumen diario) debe tener, además del toggle de **Mail**, un toggle **Alexa** independiente.
3. **REQ-003**: Debe existir un **`VoiceMonkeyService`** que envíe un anuncio TTS a `POST https://api-v3.voicemonkey.io/announce` con `token`, `device`, `speech`, `voice="Lucia"` (español) y `chime` opcional, usando stdlib (`net/http` + `encoding/json`), timeout de 10s. **Cero dependencias nuevas.**
4. **REQ-004**: Debe existir una **tabla `alerts`** (migración `0013`) sembrada desde código Go (idempotente, `INSERT OR IGNORE`) que defina: `key`, `title`, `description`, `mail_enabled`, `voice_enabled`, `speech` (texto TTS).
5. **REQ-005**: Cada scheduler, al detectar una alerta aplicable, debe **despachar a los canales habilitados**: mail (si `mail_enabled`) **y** voz (si `voice_enabled` + Voice Monkey activo). Los canales son **independientes y no bloqueantes**: si la voz falla, se loguea el error y no se afecta el mail (y viceversa).
6. **REQ-006**: La configuración Voice Monkey se guarda en `system_settings`: `voicemonkey_enabled`, `voicemonkey_send_alerts`, `voicemonkey_token`, `voicemonkey_device`. Por defecto `voicemonkey_enabled` = ausente/apagado.
7. **REQ-007**: Debe existir `POST /api/system-settings/test-voice` (botón "Probar aviso") que anuncie un mensaje de prueba por TTS, independiente de los toggles de alerta.
8. **REQ-008**: **Seguridad**: `voicemonkey_token` y `voicemonkey_device` son información sensible. **NUNCA** se devuelven en `GET /api/system-settings` ni se loguean. Solo se exponen flags booleanos: `voicemonkey_enabled`, `voicemonkey_send_alerts`, `voicemonkey_configured` (true si token+device están set).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-009**: Debe existir `GET /api/alerts` que liste cada alerta con `key`, `title`, `description`, `mail_enabled`, `voice_enabled`, `speech`.
2. **REQ-010**: Debe existir `PUT /api/alerts/{key}` (o `PUT /api/alerts`) para actualizar `mail_enabled` y `voice_enabled` por alerta.
3. **REQ-011**: El seed de `alerts` debe correr al arrancar `main.go`, insertando solo las filas faltantes sin sobrescribir los toggles existentes (el usuario no pierde config al actualizar).
4. **REQ-012**: El `speech` por alerta es un **texto corto fijo** definido en el seed, con placeholders dinámicos opcionales (p. ej. cantidad de facturas pendientes, nombre de servicio).
5. **REQ-013**: Si el toggle maestro de Voice Monkey está off, o `voicemonkey_send_alerts` está off, o falta token/device, el scheduler loguea (debug/info) que la voz está deshabilitada y no intenta el envío.
6. **REQ-014**: La UI debe reflejar el estado de Voice Monkey al cargar (master, token configurado, send_alerts, toggles por alerta) y persistir cambios inmediatamente o al guardar, según el patrón de Settings.
7. **REQ-018**: Los toggles **Alexa** de la sección Alertas deben estar **deshabilitados** (grises, no interactivos) mientras Voice Monkey no esté plenamente activo (`voicemonkey_enabled=true`, `voicemonkey_configured=true` y `voicemonkey_send_alerts=true`). Cuando estén deshabilitados, se muestra un aviso guiando al usuario a activar Voice Monkey (token + device) y el toggle "Enviar alertas".
8. **REQ-019**: El toggle **"Enviar alertas"** de la sección Voice Monkey debe estar **deshabilitado** (gris, no interactivo) mientras `voicemonkey_configured` sea `false` (faltan token o device). Solo se habilita cuando token y device están definidos.
9. **REQ-020**: Cuando token y device estén configurados (`voicemonkey_configured=true`), la sección Voice Monkey debe mostrar un **estado "Configurado"** (sin inputs de token/device) con un botón **"Reconfigurar"**. Al pulsarlo se **limpian token y device** y se **resetean `voicemonkey_enabled` y `voicemonkey_send_alerts` a OFF**, volviendo a mostrar el formulario de configuración.
10. **REQ-021**: Los toggles `voicemonkey_enabled` y `voicemonkey_send_alerts` deben persistir de forma **independiente**: enviar un toggle en `PUT /api/system-settings` NO debe resetear el otro, ni un PUT de otros settings (SMTP, billing hour, etc.) debe resetear los toggles de Voice Monkey.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-015**: Mostrar en Settings un aviso si Voice Monkey está configurado pero no hay ninguna alerta con toggle Alexa activado.
2. **REQ-016**: Extender la tabla `alerts` con flags para canales futuros (p. ej. `whatsapp_enabled`) sin migraciones destructivas, como demostración del modelo extensible.
3. **REQ-017**: Log de anuncios enviados en `system_settings` o `email_logs` estilo P2 (timestamp, alerta, status, error).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: El envío por voz es una llamada HTTP con timeout 10s que **no bloquea** el servidor (corre en goroutine del scheduler). Impacto despreciable.
- **Seguridad — Protección de información sensible**: `voicemonkey_token` y `voicemonkey_device` se tratan como credenciales (mismo nivel que `smtp_password`): **nunca** en GET, **nunca** en logs, **nunca** en errores de respuesta. `GET /api/system-settings` devuelve solo `voicemonkey_configured` (bool). Si un log llega a incluir el token por error, es un bug a corregir en review.
- **Almacenamiento**: Una tabla `alerts` pequeña (3 filas) + 4 claves nuevas en `system_settings`. Sin crecimiento significativo.
- **Disponibilidad**: Si Voice Monkey no responde o devuelve error, se loguea (sin token) y la próxima ejecución reintenta. No hay cola de reintentos (mantener simple).
- **iHost**: Cero dependencias nuevas. `net/http` y `encoding/json` están en stdlib. RAM adicional < 1MB.
- **Dependencia de red externa**: Voice Monkey requiere conexión a internet saliente (HTTPS). No es bloqueante si no hay red: se loguea el error.

---

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Voice Monkey API v3** (referencia provista por el usuario en el requerimiento):
  - Endpoint: `POST https://api-v3.voicemonkey.io/announce`
  - Body JSON: `{"token": "<api-token>", "device": "<device-id>", "speech": "<texto TTS>", "voice": "Lucia", "chime": "<opcional>"}`
  - Respuesta: `{"success": bool, "data": string, "error": string}`
  - Autenticación: `voicemonkey.io/docs/api/authentication` (API token por cuenta). El **device ID** identifica el altavoz Echo destino (ej: `echo-show-7dmsr`).
  - La voz `"Lucia"` es un TTS en español (configurable).
- **SPEC-032 (draft)**: define la sección "Alertas" en Settings con toggles de mail opt-in por funcionalidad y el modelo de toggles en `system_settings` (claves `alert_*_enabled`).
- **Schedulers existentes**: `AlertScheduler` (SPEC-029), `BillingScheduler` (SPEC-030), `BillSummaryScheduler` (SPEC-031). Todos siguen el patrón ticker horario + deduplicación por fecha.
- **`EmailService`** (`internal/services/email_service.go`): patrón de servicio de envío con config en `system_settings`. El `VoiceMonkeyService` lo replica con HTTP.
- **Migraciones**: patrón `NNNN_descripcion.up.sql`/`.down.sql`. La siguiente es `0013`.
- **Seguridad SMTP**: patrones ya establecidos (SPEC-029) para credenciales sensibles que se reutilizan para el token de Voice Monkey.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **A: `VoiceMonkeyService` con stdlib + config en `system_settings`** | Cero dependencias, configurable desde UI, token no expuesto, consistente con `EmailService` | Token en texto plano en SQLite; requiere conexión a internet | ✅ Seleccionada |
| **B: Lib externa de TTS/Alexa (ej. alexa-remote2, go-alexa)** | Menos código propio | Dependencias nuevas, binario más grande, más RAM — contra las reglas del iHost | ❌ Rechazada |
| **C: Webhook a IFTTT/otro servicio de voz** | Simple | Dependencia de terceros con otra cuenta, menos control, sin API oficial TTS | ❌ Rechazada |
| **D: Solo email (sin voz)** | Nada que implementar | No cumple el requerimiento del usuario | ❌ Rechazada |
| **E: Toggles de voz en `system_settings` planas (sin tabla `alerts`)** | Sin migración | No cumple el requerimiento explícito de "base de datos interna robusta" para saber qué alerta es cuál y su formato | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Tabla `alerts` sembrada desde código Go como fuente de verdad de las alertas**
- **Contexto**: El usuario pidió "hacer más robusto el sistema de alertas para manejar una base de datos interna (en conjunto con su código fuente para que funcione) de tal forma sepamos qué alerta es cuál y cuándo tiene formato mail o formato voz".
- **Decisión**: Nueva tabla `alerts` (`key` UNIQUE) con `mail_enabled` y `voice_enabled`. El catálogo de alertas (keys, títulos, descripciones, speech) se define en una constante/map de Go y se siembra con `INSERT OR IGNORE` al arrancar. Agregar una alerta nueva = código + seed. Agregar un canal futuro = columna + servicio de envío.
- **Consecuencias**:
  - ✅ Extensible a futuros canales (WhatsApp, Telegram)
  - ✅ Los schedulers consultan una sola fuente (`alerts`) en vez de claves planas dispersas
  - ✅ El seed idempotente no borra toggles del usuario
  - ⚠️ Una migración nueva (0013) — mínima

**ADR-002: `VoiceMonkeyService` replicando el patrón de `EmailService`, con stdlib**
- **Contexto**: Se necesita enviar anuncios TTS sin dependencias nuevas.
- **Decisión**: `internal/services/voicemonkey.go` con método `Announce(ctx, speech)`, config leída de `system_settings` (token/device), request `POST /announce`, timeout 10s, respuestas `{success, error}`. Reutiliza `SystemSettingsService` para la config.
- **Consecuencias**:
  - ✅ Cero dependencias nuevas, binario mínimo
  - ✅ Testeable con `httptest.Server`
  - ⚠️ Código propio para el cliente HTTP (trivial, ~60 líneas)

**ADR-003: Token y device de Voice Monkey como información sensible (patrón SMTP)**
- **Contexto**: Un token de API es una credencial.
- **Decisión**: Mismo tratamiento que `smtp_password`: `GetVoiceMonkeyConfigPublic` nunca devuelve token/device; solo `voicemonkey_configured` (bool). Los logs y errores nunca incluyen el token.
- **Consecuencias**:
  - ✅ Misma disciplina de seguridad ya establecida
  - ✅ `voicemonkey_configured` permite a la UI mostrar estado sin exponer el valor

**ADR-004: Dispatch multicanal no bloqueante en los schedulers**
- **Contexto**: Una alerta puede ir por mail y por voz.
- **Decisión**: Cada scheduler construye el "mensaje" de la alerta (HTML para mail, speech para voz) y despacha a cada canal habilitado en secuencia simple. El error de un canal **no** aborta el otro ni la lógica de negocio (generación de facturas, deduplicación).
- **Consecuencias**:
  - ✅ Independencia de canales
  - ✅ Simplicidad (sin colas, sin reintentos — correcto para iHost)

**ADR-005: La tabla `alerts` unifica el almacenamiento de toggles de mail (SPEC-032) y voz**
- **Contexto**: SPEC-032 (draft) propone claves planas `alert_*_enabled` en `system_settings`. SPEC-033 introduce `alerts` con `mail_enabled`/`voice_enabled`.
- **Decisión**: La tabla `alerts` es el almacenamiento real de ambos toggles. SPEC-032 sigue siendo la spec de la sección UI "Alertas" y sus criterios; SPEC-033 refina el modelo de datos. Se documenta esta relación y se valida con el usuario durante el desarrollo.
- **Consecuencias**:
  - ✅ Un solo mecanismo de persistencia (menos código, menos claves)
  - ✅ Compatible con los criterios de aceptación de SPEC-032 (los toggles persisten y sobreviven reinicios)
  - ⚠️ Desvío menor del diseño de almacenamiento de SPEC-032 (se registra aquí)

---

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
┌─────────────────────────────────────────────────────────────────┐
│                     SettingsPage.tsx (frontend)                  │
│  Sección "Alertas" (SPEC-032)                                   │
│  ├─ Seguros de autos      [Mail] [Alexa]                         │
│  ├─ Nueva factura         [Mail] [Alexa]                         │
│  └─ Resumen diario        [Mail] [Alexa]                         │
│  Sección "Voice Monkey (Alexa)"                                  │
│  ├─ [Toggle master] Activar (off por defecto)                    │
│  ├─ API token      [campo]                                      │
│  ├─ Device         [campo]                                      │
│  ├─ [Toggle] Enviar alertas                                      │
│  └─ [Botón] Probar aviso                                         │
└───────────────┬─────────────────────────────────────────────────┘
                │ GET/PUT /api/system-settings · GET/PUT /api/alerts
                ▼
┌─────────────────────────────────────────────────────────────────┐
│  Handlers (EXTENDIDO)                                            │
│  ├─ system_settings_handlers.go (+voicemonkey_*, +test-voice)    │
│  └─ alerts_handlers.go (NUEVO: GET/PUT /api/alerts)              │
└───────────────┬─────────────────────────────────────────────────┘
                ▼
┌─────────────────────────────────────────────────────────────────┐
│  SystemSettingsService (EXTENDIDO)    AlertService (NUEVO)       │
│  ├─ Get/SetVoiceMonkey*               ├─ List/Get/Update         │
│  └─ IsVoiceMonkeyConfigured           └─ Seed (INSERT OR IGNORE) │
└───────────────┬───────────────────────┬─────────────────────────┘
                ▼                       ▼
┌─────────────────────────┐   ┌───────────────────────────────┐
│  system_settings (SQLite)│   │  alerts (SQLite, migración    │
│  voicemonkey_enabled     │   │  0013): key,title,desc,      │
│  voicemonkey_send_alerts │   │  mail_enabled, voice_enabled,│
│  voicemonkey_token  🔒   │   │  speech                       │
│  voicemonkey_device  🔒  │   └───────────────────────────────┘
└─────────────┬───────────┘
              ▼
┌─────────────────────────────────────────────────────────────────┐
│  Schedulers (MODIFICADO) — dispatch multicanal                  │
│  ├─ AlertScheduler       → alerta 'insurance'                   │
│  ├─ BillingScheduler     → alerta 'bill_created'                │
│  └─ BillSummaryScheduler → alerta 'bill_summary'                │
│     mail_enabled? → EmailService.Send (existente)               │
│     voice_enabled?→ VoiceMonkeyService.Announce (NUEVO)         │
└─────────────────────────────────────────────────────────────────┘
```

### 4.2 Componentes

#### 4.2.1 `internal/models/alert.go` (NUEVO)
- **Responsabilidad**: Struct de dominio de alerta.
  ```go
  type Alert struct {
      ID           int64     `json:"id"`
      Key          string    `json:"key"`
      Title        string    `json:"title"`
      Description  string    `json:"description"`
      MailEnabled  bool      `json:"mail_enabled"`
      VoiceEnabled bool      `json:"voice_enabled"`
      Speech       string    `json:"speech"`
      CreatedAt    time.Time `json:"created_at"`
      UpdatedAt    time.Time `json:"updated_at"`
  }
  // Catálogo sembrado desde código (seed): keys estables.
  const (
      AlertKeyInsurance    = "insurance"
      AlertKeyBillCreated  = "bill_created"
      AlertKeyBillSummary  = "bill_summary"
  )
  ```

#### 4.2.2 `internal/storage/alert_storage.go` (NUEVO)
- **Responsabilidad**: CRUD básico sobre `alerts`.
  ```go
  type AlertStorage struct{ db *sql.DB }
  func (s *AlertStorage) List(ctx) ([]models.Alert, error)
  func (s *AlertStorage) GetByKey(ctx, key) (*models.Alert, error)
  func (s *AlertStorage) SetFlags(ctx, key string, mail, voice *bool) error
  func (s *AlertStorage) Seed(ctx, alerts []models.Alert) error // INSERT OR IGNORE
  ```

#### 4.2.3 `internal/services/alert_service.go` (NUEVO)
- **Responsabilidad**: Lógica de alertas (catalog + flags), wrapping de `AlertStorage`.
  ```go
  type AlertService struct{ storage *storage.AlertStorage }
  func (s *AlertService) Seed(ctx) error                       // llamado desde main.go
  func (s *AlertService) List(ctx) ([]models.Alert, error)
  func (s *AlertService) SetFlags(ctx, key string, mail, voice *bool) error
  func (s *AlertService) IsEnabled(ctx, key, channel string) (bool, error)
  ```
- El seed define el catálogo: `insurance`, `bill_created`, `bill_summary` con títulos, descripciones y speech.

#### 4.2.4 `internal/services/voicemonkey.go` (NUEVO)
- **Responsabilidad**: Enviar anuncios TTS vía Voice Monkey API.
  ```go
  type VoiceMonkeyService struct{ settings *SystemSettingsService }
  func NewVoiceMonkeyService(settings *SystemSettingsService) *VoiceMonkeyService
  func (s *VoiceMonkeyService) Announce(ctx context.Context, speech string) error
  func (s *VoiceMonkeyService) SendTest(ctx context.Context) error
  func (s *VoiceMonkeyService) IsConfigured(ctx context.Context) (bool, error)
  ```
- Internamente: `POST https://api-v3.voicemonkey.io/announce`, JSON `{token, device, speech, voice:"Lucia"}`, timeout 10s, valida `success` y `StatusCode==200`.

#### 4.2.5 `internal/services/system_settings.go` (EXTENDIDO)
- **Nuevos métodos**:
  ```go
  const (
      VoiceMonkeyEnabledKey     = "voicemonkey_enabled"
      VoiceMonkeySendAlertsKey  = "voicemonkey_send_alerts"
      VoiceMonkeyTokenKey       = "voicemonkey_token"
      VoiceMonkeyDeviceKey      = "voicemonkey_device"
  )
  func (s *SystemSettingsService) GetVoiceMonkeyConfig(ctx) (*models.VoiceMonkeyConfig, error)       // interno, con token/device
  func (s *SystemSettingsService) GetVoiceMonkeyConfigPublic(ctx) (*models.VoiceMonkeyConfigPublic, error) // sin credenciales
  func (s *SystemSettingsService) SetVoiceMonkeyConfig(ctx, cfg) error
  func (s *SystemSettingsService) IsVoiceMonkeyConfigured(ctx) (bool, error)
  func (s *SystemSettingsService) IsVoiceMonkeyEnabled(ctx) (bool, error)
  func (s *SystemSettingsService) IsVoiceMonkeySendingAlerts(ctx) (bool, error)
  ```
- `SetVoiceMonkeyConfig` usa `setIfNonEmpty` (patrón SMTP): token/device vacíos no sobrescriben valores existentes.

#### 4.2.6 `internal/models/voicemonkey_config.go` (NUEVO)
- **Responsabilidad**: Structs de config (patrón `SMTPConfig`/`SMTPConfigPublic`).
  ```go
  // VoiceMonkeyConfig — completo, uso interno. NUNCA se serializa a JSON para la API.
  type VoiceMonkeyConfig struct {
      Enabled    bool
      SendAlerts bool
      Token      string
      Device     string
  }
  // VoiceMonkeyConfigPublic — versión segura para API. Sin credenciales.
  type VoiceMonkeyConfigPublic struct {
      Enabled     bool `json:"voicemonkey_enabled"`
      SendAlerts  bool `json:"voicemonkey_send_alerts"`
      Configured  bool `json:"voicemonkey_configured"` // true si token+device set
  }
  ```

#### 4.2.7 Schedulers (MODIFICADO)
- Cada scheduler consulta `AlertService` por su `key` y despacha a los canales habilitados:
  - `mail_enabled` → flujo actual (`EmailService.Send`).
  - `voice_enabled` → verificar master (`voicemonkey_enabled`), `voicemonkey_send_alerts` y config; luego `VoiceMonkeyService.Announce(speech)`.
- La deduplicación diaria se mantiene igual. La generación de facturas (SPEC-030) no se ve afectada por ningún canal.

#### 4.2.8 `internal/api/alerts_handlers.go` (NUEVO)
- `GET /api/alerts` → lista de alertas.
- `PUT /api/alerts/{key}` → actualiza `mail_enabled`/`voice_enabled` (punteros, patrón `settingsRequest`).

#### 4.2.9 `internal/api/system_settings_handlers.go` (EXTENDIDO)
- `settingsRequest` += `VoiceMonkeyEnabled *bool`, `VoiceMonkeySendAlerts *bool`, `VoiceMonkeyToken *string`, `VoiceMonkeyDevice *string`.
- `GetSystemSettings` += `voicemonkey_enabled`, `voicemonkey_send_alerts`, `voicemonkey_configured` (bool). **NUNCA** token/device.
- Nuevo handler `TestVoice` → `POST /api/system-settings/test-voice`.

#### 4.2.10 `internal/api/routes.go` y `cmd/server/main.go` (MODIFICADO)
- Rutas: `GET /api/alerts`, `PUT /api/alerts/{key}`, `POST /api/system-settings/test-voice` (todas con auth).
- `main.go`: crear `AlertStorage`/`AlertService` + `VoiceMonkeyService`, correr `AlertService.Seed(ctx)`, inyectar `VoiceMonkeyService` en los schedulers y handlers.

#### 4.2.11 `frontend/src/pages/SettingsPage.tsx` (MODIFICADO)
- Sección "Alertas": cada fila con título, descripción y **dos toggles** (Mail / Alexa) — estilo iOS.
- Sección "Voice Monkey (Alexa)": toggle maestro (off por defecto) → muestra token, device, toggle "Enviar alertas" y botón "Probar aviso".
- Estado local cargado desde `GET /api/system-settings` + `GET /api/alerts`. Persistencia al cambiar (patrón de toggles de SPEC-032).

#### 4.2.12 `frontend/src/api/index.ts` (EXTENDIDO)
- Tipos de `systemSettings.get/update` con campos VM; nuevo `alerts` (list/update) y `systemSettings.testVoice`.

#### 4.2.13 `public/i18n/{es,en}.json` (EXTENDIDO)
- Claves nuevas bajo `settings.voicemonkey.*` y `settings.alerts.*` (títulos, descripciones, labels de toggles Mail/Alexa, botón probar).

### 4.3 Modelo de datos

**Migración `0013_create_alerts.up.sql`**:
```sql
CREATE TABLE IF NOT EXISTS alerts (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    key           TEXT    NOT NULL UNIQUE,
    title         TEXT    NOT NULL,
    description   TEXT    NOT NULL,
    mail_enabled  INTEGER NOT NULL DEFAULT 0,
    voice_enabled INTEGER NOT NULL DEFAULT 0,
    speech        TEXT    NOT NULL,
    created_at    TEXT    NOT NULL,
    updated_at    TEXT    NOT NULL
);
```

**Migración `0013_create_alerts.down.sql`**:
```sql
DROP TABLE IF EXISTS alerts;
```

**Seed (código Go, `INSERT OR IGNORE`)**:
```
key: insurance     → title "Seguros de autos vencidos"
                    desc "Recibir un email o aviso por Alexa cuando un seguro de auto vence o un auto queda sin cobertura."
                    speech "Paulo, tienes autos sin seguro o con seguros vencidos."
key: bill_created  → title "Nueva factura generada"
                    desc "Informativo cuando el sistema genera una factura automáticamente."
                    speech "Paulo, se generó una nueva factura automática."
key: bill_summary  → title "Resumen diario de facturas"
                    desc "Resumen diario de todas las facturas pendientes."
                    speech "Paulo, tienes {n} facturas pendientes."   -- placeholder dinámico
```

**Claves nuevas en `system_settings`**:
```
- voicemonkey_enabled      TEXT -- "1"/"0" (ausente = apagado)
- voicemonkey_send_alerts  TEXT -- "1"/"0"
- voicemonkey_token        TEXT -- SENSIBLE: API token
- voicemonkey_device       TEXT -- SENSIBLE: device ID del Echo
```

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/alerts` (NUEVO)

**Response 200**:
```json
[
  {
    "key": "insurance",
    "title": "Seguros de autos vencidos",
    "description": "Recibir un email o aviso por Alexa cuando un seguro de auto vence...",
    "mail_enabled": false,
    "voice_enabled": false,
    "speech": "Paulo, tienes autos sin seguro o con seguros vencidos."
  }
]
```

#### Endpoint: `PUT /api/alerts/{key}` (NUEVO)

**Request** (campos opcionales):
```json
{
  "mail_enabled": true,
  "voice_enabled": false
}
```

**Response 200**:
```json
{
  "key": "insurance",
  "mail_enabled": true,
  "voice_enabled": false,
  "message": "Alerta actualizada"
}
```

#### Endpoint: `GET /api/system-settings` (EXTENDIDO)

**Response 200** (campos nuevos marcados con ★):
```json
{
  "billing_generation_hour": 8,
  "smtp_host": "smtp.example.com",
  "smtp_configured": true,
  "alert_emails": "paulo@example.com",
  "voicemonkey_enabled": false,        // ★
  "voicemonkey_send_alerts": false,    // ★
  "voicemonkey_configured": false      // ★ (true si token+device set)
}
```

> **Seguridad**: `voicemonkey_token` y `voicemonkey_device` son credenciales y **NUNCA** se devuelven en el GET (mismo tratamiento que `smtp_user`/`smtp_password`).

#### Endpoint: `PUT /api/system-settings` (EXTENDIDO)

**Request** (campos opcionales):
```json
{
  "voicemonkey_enabled": true,
  "voicemonkey_send_alerts": true,
  "voicemonkey_token": "<api-token>",   // 🔒 solo se envía para guardar
  "voicemonkey_device": "echo-show-7dmsr" // 🔒 solo se envía para guardar
}
```

> **Seguridad**: si `voicemonkey_token` o `voicemonkey_device` no se envían (ausentes/vacíos), se mantiene el valor actual guardado (patrón SMTP).

**Response 200**:
```json
{
  "message": "Configuración actualizada",
  "voicemonkey_configured": true
}
```

#### Endpoint: `POST /api/system-settings/test-voice` (NUEVO)

**Request**: vacío (anuncia un mensaje de prueba).

**Response 200**:
```json
{
  "message": "Aviso de voz enviado"
}
```

**Response 400** (no configurado):
```json
{
  "error": "voicemonkey_not_configured",
  "message": "Configure Voice Monkey (token y device) antes de probar"
}
```

### 4.5 UI — Sección Alertas (diseño iOS)

```
┌─ ALERTAS ─────────────────────────────────────────────┐
│  ┌───────────────────────────────────────────────────┐ │
│  │ Seguros de autos vencidos          [Mail][Alexa]  │ │
│  │ Recibir aviso cuando un seguro de auto vence o    │ │
│  │ un auto queda sin cobertura.                      │ │
│  └───────────────────────────────────────────────────┘ │
│  ┌───────────────────────────────────────────────────┐ │
│  │ Nueva factura generada             [Mail][Alexa]  │ │
│  │ Informativo cuando el sistema genera una factura. │ │
│  └───────────────────────────────────────────────────┘ │
│  ┌───────────────────────────────────────────────────┐ │
│  │ Resumen diario de facturas         [Mail][Alexa]  │ │
│  │ Resumen diario de facturas pendientes.            │ │
│  └───────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────┘

┌─ VOICE MONKEY (ALEXA) ────────────────────────────────┐
│  ┌───────────────────────────────────────────────────┐ │
│  │ Activar Voice Monkey                  [toggle]    │ │  ← OFF por defecto
│  │ Anunciar alertas por voz en un altavoz Alexa.     │ │
│  └───────────────────────────────────────────────────┘ │
│  (si activo)                                          │
│  │ API token   [campo]                                │
│  │ Device      [campo]                                │
│  │ Enviar alertas                          [toggle]   │
│  │ [Probar aviso]  ─ estado: configurado/✓            │
└────────────────────────────────────────────────────────┘
```

### 4.6 Dependencias

- **Internas**:
  - `SystemSettingsService` / `SystemSettingsStorage` (extendido)
  - `AlertStorage` / `AlertService` (nuevos)
  - `VoiceMonkeyService` (nuevo)
  - `AlertScheduler`, `BillingScheduler`, `BillSummaryScheduler` (modificados)
  - `system_settings_handlers.go`, `routes.go`, `main.go` (modificados)
  - `alerts_handlers.go` (nuevo)
  - `SettingsPage.tsx`, `api/index.ts`, `public/i18n/*.json` (frontend)
- **Externas**:
  - **NINGUNA nueva**. Se usa `net/http` y `encoding/json` de la stdlib de Go.
  - API de Voice Monkey (servicio externo, sin SDK).

---

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Dado un usuario que nunca configuró Voice Monkey, cuando abre Settings, entonces la sección "Voice Monkey (Alexa)" está en OFF por defecto y no se envía ningún anuncio.
- [x] CA-002: Dado Voice Monkey activado, cuando el usuario configura token y device, entonces se persisten y `voicemonkey_configured` pasa a `true`.
- [x] CA-003: Dado el toggle "Enviar alertas" en OFF, cuando un scheduler detecta una alerta con `voice_enabled`, entonces NO anuncia por voz.
- [x] CA-004: Dado el toggle "Enviar alertas" en ON, Voice Monkey configurado y una alerta con `voice_enabled`, cuando el scheduler ejecuta, entonces anuncia el `speech` correspondiente por Alexa.
- [x] CA-005: Dado una alerta con `mail_enabled` y `voice_enabled`, cuando el scheduler ejecuta, entonces envía BOTH el email y el anuncio de voz, independientemente.
- [x] CA-006: Dado que el anuncio de voz falla (error de red/API), cuando el scheduler ejecuta, entonces loguea el error (sin token) y el email se envía igualmente (no bloqueante).
- [x] CA-007: Dado `GET /api/system-settings`, entonces la respuesta incluye `voicemonkey_enabled`, `voicemonkey_send_alerts` y `voicemonkey_configured` como booleanos y **NO incluye** `voicemonkey_token` ni `voicemonkey_device`.
- [x] CA-008: Dado `PUT /api/system-settings` sin enviar token/device, entonces se mantienen los valores actuales (no se sobrescriben con vacío).
- [x] CA-009: Dado el botón "Probar aviso", cuando Voice Monkey está configurado, entonces anuncia un mensaje de prueba.
- [x] CA-010: Dado `GET /api/alerts`, entonces devuelve las 3 alertas sembradas con sus títulos, descripciones y `speech`.
- [x] CA-011: Dado `PUT /api/alerts/{key}`, cuando se cambia `mail_enabled`/`voice_enabled`, entonces persiste y sobrevive a un reinicio.
- [x] CA-012: Dado un reinicio del servidor, cuando corre el seed, entonces no sobrescribe los toggles configurados por el usuario (solo inserta filas faltantes).
- [x] CA-013: Dado el botón "Enviar email de prueba" (SPEC-032), cuando SMTP está configurado, entonces sigue funcionando con Voice Monkey activado o no.
- [x] CA-014: Dado Voice Monkey inactivo, sin configurar o con "Enviar alertas" apagado, cuando el usuario ve la sección Alertas, entonces los toggles Alexa están **deshabilitados** y se muestra un aviso guiando a activar Voice Monkey. Al activar las 3 condiciones, los toggles se habilitan.
- [x] CA-015: Dado que `voicemonkey_configured` es `false` (faltan token o device), cuando el usuario ve la sección Voice Monkey, entonces el toggle "Enviar alertas" está **deshabilitado** y no puede activarse. Al guardar token y device, se habilita.
- [x] CA-016: Dado token y device configurados, cuando el usuario ve la sección Voice Monkey, entonces se muestra el estado "Configurado" sin inputs de token/device, con un botón "Reconfigurar". Al pulsarlo se limpian las credenciales y los toggles vuelven a OFF, mostrando el formulario.
- [x] CA-017: Dado que se envía `PUT /api/system-settings` con solo `voicemonkey_enabled: true`, entonces `voicemonkey_send_alerts` conserva su valor anterior. Dado un `PUT` con campos SMTP (sin campos VM), entonces los toggles de Voice Monkey NO cambian.

### 5.2 No funcionales

- [x] CA-NF-001: El binario compilado no aumenta más de 50KB (sin dependencias nuevas).
- [x] CA-NF-002: `go build ./...` compila sin errores.
- [x] CA-NF-003: `go vet ./...` no reporta errores.
- [x] CA-NF-004: `grep -rn "voicemonkey_token\|voicemonkey_device\|smtp_password" internal/ cmd/` no encuentra credenciales hardcoded.
- [x] CA-NF-005: El build de frontend (`npm run build`) compila sin errores.
- [x] CA-NF-006: El envío de voz no bloquea el servidor HTTP (corre en goroutine del scheduler).

### 5.3 Testing

- **Unit tests**:
  - `voicemonkey_test.go`: `Announce` con `httptest.Server` — payload correcto (`token`, `device`, `speech`, `voice`), manejo de `success=false`, timeout.
  - `alert_service_test.go`: seed idempotente (dos corridas no duplican ni resetean flags), `SetFlags`, `IsEnabled`.
  - `system_settings_test.go`: Get/Set config Voice Monkey; `GetVoiceMonkeyConfigPublic` no incluye token/device; campo ausente = apagado.
  - `alert_scheduler_test.go` / `bill_summary_scheduler_test.go`: `voice_enabled` OFF → no llama a `Announce`; ON + config → llama.
- **Integration tests**:
  - `PUT /api/system-settings` con token/device → `GET` los omite y devuelve `voicemonkey_configured: true`.
  - `PUT /api/alerts/{key}` → `GET /api/alerts` refleja el cambio.
  - Scheduler con ambas alertas habilitadas → se llama `EmailService.Send` y `VoiceMonkeyService.Announce`.
- **E2E tests (manuales)**:
  - Activar Voice Monkey en Settings, configurar token/device, probar aviso → escuchar el anuncio en el Echo.
  - Activar Alexa en una alerta → esperar scheduler → escuchar anuncio.
  - Desactivar → no se escucha nada.
  - Recargar página → estado persistido.
- **Carga/Performance**: N/A (volumen de alertas es bajo, < 100).

---

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración `0013_create_alerts` + modelos (`Alert`, `VoiceMonkeyConfig`) | 0.25 día | Ninguna |
| 2 | `AlertStorage` + `AlertService` + seed desde código | 0.5 día | Fase 1 |
| 3 | `SystemSettingsService`: getters/setters Voice Monkey (config + flags, patrón SMTP) | 0.25 día | Fase 1 |
| 4 | `VoiceMonkeyService` con stdlib (Announce, SendTest, IsConfigured) + unit tests | 0.5 día | Fase 3 |
| 5 | Modificar los 3 schedulers: dispatch multicanal (mail + voz) no bloqueante + tests | 0.75 día | Fases 2, 4 |
| 6 | Handlers: `alerts_handlers.go` (GET/PUT) + extender `system_settings_handlers.go` + `test-voice` + rutas + `main.go` | 0.5 día | Fases 2, 3, 4 |
| 7 | Frontend: sección Alertas con toggles Mail+Alexa, sección Voice Monkey, API types, i18n | 1 día | Fase 6 |
| 8 | Build + vet + tests + validación manual local | 0.5 día | Todas |

**Estimación total**: ~4.25 días (depende de SPEC-032, que se implementa primero).

### 6.2 Milestones

1. **MVP**: Backend completo (migración, seed, VoiceMonkeyService, dispatch en schedulers, API) — verificable con curl + un Echo real.
2. **V1.0**: UI completa (toggles por alerta + sección Voice Monkey) + tests + i18n.

---

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Token/device incorrectos configurados por el usuario | Media | Medio | Botón "Probar aviso" para validar antes de depender del scheduler. Logs con errores específicos (sin token) |
| Voice Monkey sin conexión a internet (iHost offline) | Media | Bajo | Se loguea el error y se reintenta en la próxima ejecución. Sin cola (mantener simple) |
| El anuncio se duplica si el scheduler corre dos veces el mismo día | Baja | Medio | Reutilizar la deduplicación diaria existente (`last_*_check`) — el canal de voz comparte el mismo check |
| `voicemonkey_token` filtrado en logs por error de programación | Baja | Alto | Regla explícita: slog nunca recibe token/device. Tests de ausencia de credenciales en logs (patrón SPEC-029) |
| Confusión de la UI entre toggle de alerta y toggle maestro de Voice Monkey | Media | Bajo | Secciones separadas y claras; el master es un toggle independiente con descripción |
| La tabla `alerts` desvía el diseño de SPEC-032 (claves planas) | Media | Bajo | ADR-005 documenta la unificación; se valida con el usuario antes/durante el desarrollo |
| Rate limit o costo de la API de Voice Monkey | Baja | Bajo | Los anuncios son 1-3 por día (máximo). No hay loops |

---

## 8. Notas y Referencias

- Voice Monkey API: `https://www.voicemonkey.io/docs/api/announcement` y `https://www.voicemonkey.io/docs/api/authentication`
- Código de referencia provisto por el usuario: `POST https://api-v3.voicemonkey.io/announce` con `token`, `device`, `speech`, `voice:"Lucia"`.
- SPEC-029: `AlertScheduler`, `EmailService`, seguridad de credenciales (base).
- SPEC-030: email por factura automática en `BillingScheduler`.
- SPEC-031: `BillSummaryScheduler` y resumen diario.
- SPEC-032 (draft): sección "Alertas" con toggles de mail — base de esta spec.
- `internal/services/email_service.go` (patrón de servicio de envío).
- `internal/services/system_settings.go` (patrón de settings + `setIfNonEmpty`).
- `internal/api/system_settings_handlers.go` (GET/PUT con punteros opcionales).

---

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-16 | paulomcnally | Creación inicial de la especificación. Canal de voz (Voice Monkey/Alexa) para alertas, tabla `alerts` sembrada desde código, dispatch multicanal no bloqueante, sección Voice Monkey en Settings (off por defecto), token/device como info sensible. Depende de SPEC-032. |
| 2026-08-16 | paulomcnally | Cambio iterativo solicitado por el usuario: toggles Alexa de la sección Alertas deshabilitados mientras Voice Monkey no esté plenamente activo (enabled + configured + send_alerts), con aviso guiando al usuario (REQ-018, CA-014). |
| 2026-08-16 | paulomcnally | Cambio iterativo solicitado por el usuario: el toggle "Enviar alertas" de Voice Monkey no debe poder activarse sin token/device configurados; queda deshabilitado mientras `voicemonkey_configured` sea `false` (REQ-019, CA-015). |
| 2026-08-16 | paulomcnally | Cambios iterativos solicitados por el usuario: (1) estado "Configurado" con botón "Reconfigurar" que limpia credenciales y resetea toggles a OFF (REQ-020, CA-016); (2) fix de persistencia: los toggles de VM no deben pisarse entre sí en updates parciales (REQ-021, CA-017). |
| 2026-08-16 | paulomcnally | Estado a `in_progress`. Desarrollo iniciado en rama `feature/SPEC-032-033-alertas-multicanal`. |
| 2026-08-16 | paulomcnally | Desarrollo completado y validado en local por el usuario. Criterios de aceptación pasan. Estado a `pending_release`. |