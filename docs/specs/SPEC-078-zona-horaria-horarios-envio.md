---
title: "Zona horaria en horarios de envío de alertas"
id: "SPEC-078"
status: "released"
author: "Agente opencode"
created: "2026-09-16"
updated: "2026-09-16"
github_issue: 81
---

# Zona horaria en horarios de envío de alertas

**ID**: SPEC-078  
**Estado**: released  
**Autor**: Agente opencode  
**Creado**: 2026-09-16  
**Actualizado**: 2026-09-16

---

## 1. Resumen Ejecutivo

El usuario reporta que el horario de envío de alertas configurado en Settings no se respeta: se programa (por ejemplo) a las 7am pero las alertas llegan a otra hora. La investigación confirma la causa raíz: la UI guarda `alert_check_hour` como un entero hora-del-día (0-23) que el usuario interpreta como **hora local**, pero el backend compara ese entero contra `time.Now().Hour()` del **contenedor**, que corre en **UTC** (imagen distroless sin tzdata y sin `TZ` definida). No existe ninguna conversión de zona horaria en backend ni frontend.

La solución requiere introducir una **zona horaria configurable por el usuario** (selector de zonas IANA, que cubre "depende del país") y que todos los schedulers (`alert_scheduler`, `billing_scheduler`, `bill_summary_scheduler`, `debt_due_scheduler`) conviertan la hora local configurada a la hora UTC equivalente del contenedor al momento de disparar. El usuario seleccionó que la conversión aplique a **todos** los horarios (alertas, generación de facturas y resumen).

Consideraciones iHost: se usará el paquete estándar `time/tzdata` embebido en el binario Go (~450KB adicionales, sin dependencias externas ni archivos de sistema), evitando instalar tzdata en la imagen distroless. No hay impacto en SQLite: el nuevo setting es una key/value más en `system_settings`.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Agregar un setting `timezone` (nombre IANA, ej. `America/Argentina/Buenos_Aires`) en `system_settings`, persistible vía `GET/PUT /api/system-settings`.
2. **REQ-002**: Agregar selector de zonas horarias IANA en la página de Settings (frontend), con etiqueta localizada y búsqueda por ciudad/país si es viable. El setting por defecto debe ser la zona del navegador del usuario (o `UTC` si no se puede detectar).
3. **REQ-003**: Los 4 schedulers (`alert_scheduler`, `billing_scheduler`, `bill_summary_scheduler`, `debt_due_scheduler`) deben comparar la hora configurada contra la hora **local del usuario** (convertida desde UTC usando el setting `timezone`), de modo que un horario de 7am local dispare a la hora UTC correspondiente.
4. **REQ-004**: Si el setting `timezone` está vacío o es inválido, los schedulers deben comportarse como hoy (fallback UTC) sin romper, y la UI debe mostrar un aviso para que el usuario lo configure.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-005**: La hora mostrada en la UI (Settings y SettingsBilling) debe indicar la zona horaria aplicada (ej. subtítulo "Hora local (America/Argentina/Buenos_Aires)") para evitar confusión.
2. **REQ-006**: Validar en backend que el valor `timezone` sea un nombre IANA válido (`time.LoadLocation`), descartando con error (no silenciosamente) valores inválidos.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-007**: Mostrar en el selector la hora UTC equivalente al momento de elegir (ej. "07:00 local = 10:00 UTC") como preview.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: `time.LoadLocation` se cachea en memoria (una carga por arranque o lazy-cache); sin impacto medible en el ticker horario.
- **Seguridad**: Sin datos sensibles nuevos. Validación estricta de entrada (nombre IANA únicamente, whitelist por `time.LoadLocation`).
- **Almacenamiento**: 1 key/value adicional en SQLite (orden de bytes, sin impacto).
- **Disponibilidad**: Fallback a UTC si el setting falta o es inválido; nunca crashear el scheduler.
- **iHost**: `time/tzdata` embebido en el binario (~450KB) en lugar de instalar tzdata en la imagen distroless. Cero dependencias externas nuevas.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Investigación del código (ver sección 8, Referencias):

- **Backend**: `internal/services/system_settings.go` — `GetAlertCheckHour`/`SetAlertCheckHour` (líneas 266-288) y `GetBillingGenerationHour` (24-43) almacenan entero hora 0-23 como TEXT.
- **Schedulers** (el bug): `internal/services/alert_scheduler.go:87-96`, `billing_scheduler.go:75-84`, `bill_summary_scheduler.go:85-94`, `debt_due_scheduler.go:80-89` — todos comparan `time.Now().Hour()` contra el entero guardado. El contenedor corre en UTC (sin `TZ` en `docker-compose.yml`, distroless sin tzdata).
- **Frontend**: `frontend/src/pages/SettingsBillingPage.tsx:31-104` (Select de horas 0-23) y `SettingsPage.tsx:72-146` (display sin conversión). Ninguna conversión de timezone en `frontend/src`.
- **API**: `internal/api/system_settings_handlers.go:70,115,172,212-217` — `alert_check_hour` como `*int`.
- **DB**: tabla `system_settings` key/value TEXT (migración `0007_add_billing_automation.up.sql`).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Configurar `TZ` en el contenedor | Cambio trivial | La hora del usuario depende de su país; el contenedor es único para todos los usuarios (app single-user en la práctica, pero el setting por usuario es más correcto y flexible). No resuelve "depende del país" si el usuario viaja/cambia. | ❌ Rechazada |
| Guardar offset UTC fijo (ej. `-03:00`) | Simple, sin tzdata | No maneja DST ni offsets de 30/45 min correctamente | ❌ Rechazada |
| Setting `timezone` IANA + conversión en schedulers | Correcto para DST y offsets raros; estándar; el usuario elige su país/zona | Requiere tzdata (~450KB embebido o instalado) | ✅ Seleccionada |
| Instalar tzdata en la imagen distroless | Sin crecimiento del binario | Requiere cambiar base image o multi-stage con apt; distroless está pensada sin shell | ❌ Rechazada (se prefiere `time/tzdata` embebido) |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Usar `import _ "time/tzdata"` en el binario Go
- **Contexto**: Los schedulers necesitan convertir entre UTC (reloj del contenedor) y la zona horaria del usuario. La imagen runtime es `gcr.io/distroless/static-debian12`, sin tzdata ni apt.
- **Decisión**: Embeber tzdata con el paquete estándar `time/tzdata` (~450KB en el binario). `time.LoadLocation` funciona con nombres IANA sin depender del sistema.
- **Consecuencias**: Binario ~450KB más grande (irrelevante en disco de iHost); cero dependencias externas; funciona en cualquier entorno (host, Docker, iHost).

**ADR-002**: La conversión se hace en el scheduler, no en la UI
- **Contexto**: El usuario piensa en hora local y configura "07:00". El contenedor corre en UTC.
- **Decisión**: En cada scheduler, resolver la hora local actual con `now.In(location).Hour()` y comparar contra el entero guardado. La UI guarda la hora local tal cual (sin convertir), y muestra la zona aplicada.
- **Consecuencias**: La DB no cambia de formato (sigue siendo entero 0-23 hora local). El cambio de zona del usuario re-programa los envíos sin migración de datos.

**ADR-003**: Fallback seguro a UTC
- **Contexto**: El setting `timezone` puede faltar en instalaciones existentes.
- **Decisión**: Si el setting falta o `LoadLocation` falla, los schedulers usan `time.UTC` (comportamiento actual) y la UI muestra aviso para configurarlo.
- **Consecuencias**: Migración sin riesgo; instalaciones que no configuren zona mantienen el comportamiento actual.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Settings UI (frontend)] --PUT /api/system-settings {timezone, alert_check_hour}--> [API handlers]
        |                                                                              |
        v                                                                              v
[Selector IANA + preview]                                                       [system_settings storage (SQLite)]
                                                                                           |
                                                                                           v
[Ticker 1h] --> [Scheduler] --> now.In(location).Hour() == storedHour? --> [disparar envío]
                                     ^
                                     |
                     [time.LoadLocation(timezone) — cacheado, time/tzdata embebido]
```

### 4.2 Componentes

#### 4.2.1 `internal/services/system_settings.go`
- **Responsabilidad**: Leer/escribir `timezone` con validación IANA.
- **Interfaz**: `GetTimezone(ctx) string`, `SetTimezone(ctx, tz string) error` (valida con `time.LoadLocation`, error si inválido).
- **Dependencias**: `time/tzdata` (import side-effect en `main.go` o en el paquete de settings).
- **Ubicación**: `internal/services/system_settings.go`

#### 4.2.2 Schedulers (4 archivos)
- **Responsabilidad**: Comparar hora local del usuario, no hora del contenedor.
- **Interfaz**: helper compartido `internal/services/hour_utils.go` (o similar): `CurrentUserHour(ctx, settingsSvc) (int, error)` → `time.Now().In(loc).Hour()`.
- **Dependencias**: `system_settings.GetTimezone`, cache de `*time.Location`.
- **Ubicación**: `internal/services/alert_scheduler.go`, `billing_scheduler.go`, `bill_summary_scheduler.go`, `debt_due_scheduler.go`

#### 4.2.3 `internal/api/system_settings_handlers.go`
- **Responsabilidad**: Exponer `timezone` en GET y aceptarlo en PUT.
- **Interfaz**: campo `Timezone *string` en `settingsRequest`; GET devuelve `"timezone": "..."`.
- **Dependencias**: servicio de settings.
- **Ubicación**: `internal/api/system_settings_handlers.go`

#### 4.2.4 Frontend `SettingsPage.tsx` / `SettingsBillingPage.tsx`
- **Responsabilidad**: Selector de zonas IANA + display de hora con zona.
- **Interfaz**: lista estática de zonas IANA (constante en `frontend/src/constants/timezones.ts` o similar) con offset actual calculado por `Intl`; preview de hora UTC.
- **Dependencias**: `Intl.DateTimeFormat` (nativo del navegador, sin librerías).
- **Ubicación**: `frontend/src/pages/SettingsPage.tsx`, `frontend/src/pages/SettingsBillingPage.tsx`

### 4.3 Modelo de datos

```
system_settings (existente, sin migración de esquema)
- key: 'timezone'
- value: 'America/Argentina/Buenos_Aires' | '' | null (ausente = fallback UTC)
```

Formato de almacenamiento: nombre IANA (TEXT). No hay migración de datos: los horarios guardados ya son "hora local" según el usuario; la zona nueva solo re-interpreta el reloj del scheduler.

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/system-settings`

**Response 200** (campos nuevos):
```json
{
  "alert_check_hour": 7,
  "billing_generation_hour": 0,
  "timezone": "America/Argentina/Buenos_Aires"
}
```

#### Endpoint: `PUT /api/system-settings`

**Request**:
```json
{
  "timezone": "America/Argentina/Buenos_Aires",
  "alert_check_hour": 7
}
```

**Response Error 400**:
```json
{
  "error": "invalid_timezone",
  "message": "zona horaria inválida"
}
```

### 4.5 Dependencias

- **Internas**: `system_settings` (nuevo key), 4 schedulers (lógica de hora), handlers API (campo nuevo), frontend Settings (selector + display), i18n (`frontend/public/i18n/{es,en}.json`).
- **Externas**: ninguna nueva. `time/tzdata` es stdlib. Frontend usa `Intl.DateTimeFormat` nativo.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado un usuario en Argentina (UTC-3) que configura `alert_check_hour = 7` y `timezone = America/Argentina/Buenos_Aires`, cuando el reloj del contenedor marca 10:00 UTC, entonces la alerta se dispara (7am local).
- [ ] CA-002: Dado `timezone` vacío/ausente, cuando corre el scheduler, entonces se comporta como hoy (comparación contra hora UTC del contenedor) sin errores ni crashes.
- [ ] CA-003: Dado un `PUT /api/system-settings` con `timezone` inválido (ej. `"Foo/Bar"`), entonces el backend responde 400 y NO persiste el valor.
- [ ] CA-004: Dado el frontend Settings, cuando el usuario selecciona su zona, entonces el selector persiste el valor vía API y la UI muestra la zona aplicada (subtítulo "Hora local (zona)").
- [ ] CA-005: Dado el usuario que cambia de zona horaria, cuando guarda, entonces los horarios programados (alertas, facturas, resumen, deudas) se recalculan sin migración de datos.
- [ ] CA-006: Los 4 schedulers usan el helper compartido de hora local (sin lógica duplicada).
- [ ] CA-DARK: El selector de zona horaria (input/select) usa tokens del tema (`bg-card`, `text-text`/`text-text-secondary`) y se verificó legibilidad en darkmode.
- [ ] CA-BACK: No aplica (Settings es página raíz, no tiene lista padre; no se crean páginas de detalle nuevas).

### 5.2 No funcionales

- [ ] CA-NF-001: El binario sigue compilando para las 3 arquitecturas multi-arch (amd64, arm/v7, arm64) y funciona sin tzdata del sistema (validado con `time.LoadLocation` en la imagen distroless).
- [ ] CA-NF-002: `time.LoadLocation` no se llama por tick (cacheado); sin impacto en el ticker horario.

### 5.3 Testing

- **Unit tests**: 
  - `SetTimezone` acepta IANA válido y rechaza inválido.
  - Helper de hora local: dado un instante fijo UTC y un `timezone`, devuelve la hora local correcta (incluyendo caso DST).
  - Scheduler dispara a la hora local correcta con zona configurada y a la hora UTC con zona vacía.
- **Integration tests**: `PUT /api/system-settings` con timezone válido/inválido; `GET` devuelve el valor persistido.
- **E2E tests**: Seleccionar zona en Settings → recargar → la zona persiste y el display muestra la hora con la zona.
- **Carga/Performance**: Ninguna métrica nueva relevante (operación horaria, 1 lookup cacheado).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Backend: key `timezone` en settings (get/set + validación IANA) + helper de hora local + refactor de los 4 schedulers | 1 día | Ninguna |
| 2 | API: exponer `timezone` en GET/PUT system-settings | 0.5 día | Fase 1 |
| 3 | Frontend: lista de zonas IANA, selector en Settings (SettingsBilling + SettingsPage), display con zona, i18n (es/en) | 1 día | Fase 2 |
| 4 | Build multi-arch + pruebas en local (unit + manual con TZ simulada) | 0.5 día | Fase 3 |

### 6.2 Milestones

1. **MVP**: Fases 1-2 (backend correcto; el setting se puede configurar por API).
2. **V1.0**: Fases 1-4 (selector en UI, i18n, build verificado).

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Zonas con DST cambian la hora UTC de disparo a mitad de año | Media | Medio | Usar IANA (no offset fijo): `now.In(loc)` resuelve DST automáticamente |
| Usuario no configura zona → horarios siguen "rotos" como hoy | Media | Medio | Fallback UTC documentado + aviso en UI cuando falta la zona |
| Lista de zonas IANA muy larga en UI (400+ entradas) | Alta | Bajo | Select con búsqueda/agrupado por offset; o lista reducida de zonas comunes |
| `time/tzdata` agrega ~450KB al binario | Baja | Bajo | Irrelevante en iHost (disco >1GB); sin dependencias runtime |
| Zonas con offsets de 30/45 min (India, Nepal) | Baja | Bajo | IANA las maneja nativamente; cubierto por tests |

## 8. Notas y Referencias

- Código investigado: `internal/services/system_settings.go:24-43,266-288`; `internal/services/alert_scheduler.go:56-96`; `internal/services/billing_scheduler.go:75-84`; `internal/services/bill_summary_scheduler.go:85-94`; `internal/services/debt_due_scheduler.go:80-89`; `internal/api/system_settings_handlers.go:70,115,172,212-217`; `frontend/src/pages/SettingsPage.tsx:72-146`; `frontend/src/pages/SettingsBillingPage.tsx:31-104`; `migrations/0007_add_billing_automation.up.sql`.
- Documentación Go: `time.LoadLocation`, paquete `time/tzdata` (https://pkg.go.dev/time/tzdata).
- Países con múltiples zonas (USA, Brasil, Australia): el selector IANA los cubre por ciudad.
- Precedente de i18n: editar SOLO `frontend/public/i18n/{es,en}.json` (la salida `public/i18n/` se regenera en build).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-16 | Agente opencode | Creación inicial de la especificación |
| 2026-09-16 | Agente opencode | Implementación completa: setting `timezone` (IANA) en backend + API, helper `currentUserNow` y refactor de los 4 schedulers, `time/tzdata` embebido, selector de zonas en UI (SettingsBilling) con default a zona del navegador, display de zona en SettingsPage, i18n es/en. Verificado en local (tests + API + UI). |
| 2026-09-16 | Agente opencode | Release: merge a `main` (`8bb92cc`), push a `origin/main`, issue #81 cerrado con label `spec/released`. |