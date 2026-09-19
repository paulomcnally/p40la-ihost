---
title: "Edición de token y chat_ids del bot de Telegram sin borrar la configuración"
id: "SPEC-089"
status: "released"
author: "Agente opencode"
created: "2026-09-19"
updated: "2026-09-19"
github_issue: 92
---

# Edición de token y chat_ids del bot de Telegram sin borrar la configuración

**ID**: SPEC-089  
**Estado**: released  
**Autor**: Agente opencode  
**Creado**: 2026-09-19  
**Actualizado**: 2026-09-19

---

## 1. Resumen Ejecutivo

El bot de Telegram está habilitado en producción (`telegram_bot_enabled=1` + token), pero el usuario no puede habilitar las alertas de Telegram en `Configuración → Alertas`: los toggles aparecen deshabilitados. La causa raíz es que la página de alertas requiere `telegram_bot_chat_ids_count > 0` (`frontend/src/pages/SettingsAlertsPage.tsx:45-47`), y la DB de producción **no tiene** `telegram_bot_chat_ids` configurado (allowlist vacía).

El problema de fondo es de UX en `frontend/src/pages/SettingsTelegramBotPage.tsx`: cuando el bot está configurado, los campos de token y chat_ids **no se muestran**. La única vía para cambiarlos es el botón "Reconfigurar", que borra toda la configuración (`DeleteTelegramBot` → `ClearTelegramBot`: token, chat_ids y `enabled=OFF`) y obliga a reingresar todo. El usuario exige poder **cambiar el token y setear los chat_ids sin borrar la información actual**.

Esta spec corrige ambos problemas:
1. **El botón "Reconfigurar" NO borra nada**: se elimina el flujo de borrado total. El formulario de token/chat_ids siempre está disponible cuando el bot está activo, con los chat_ids **precargados** desde el API y el token vacío (si ya existe, no se toca salvo que se ingrese uno nuevo).
2. **Con los chat_ids seteables, las alertas de Telegram se desbloquean**: al existir al menos un chat_id, `chat_ids_count > 0` y los toggles de `SettingsAlertsPage` se habilitan. Las alertas push (`SendAlerts`, `internal/services/telegram_bot.go:202-217`) se envían a todos los chat_ids configurados; con allowlist vacía no se envía nada (`:187-190`), por lo que setear chat_ids es condición necesaria para recibir alertas.

Cambios de UI obligatorios: los inputs del formulario del bot usan tokens del tema (`bg-card`, `text-text`) y se verifican en darkmode (patrón existente del proyecto, ver SPEC-060). No se agregan páginas de detalle nuevas.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: El API de configuración pública del bot (`GET/PUT /api/system-settings`) debe devolver `telegram_bot_chat_ids` (array de strings) con los chat_ids actuales, para precargarlos en el formulario. El token **nunca** se devuelve (se mantiene el flag `telegram_bot_configured`).
2. **REQ-002**: El formulario del bot (`SettingsTelegramBotPage`) debe mostrar SIEMPRE los campos de token y chat_ids cuando el bot está activo, tanto si está configurado como si no.
3. **REQ-003**: El campo de chat_ids se precarga con los valores actuales del API y se persiste con el valor editado al guardar (semántica "set").
4. **REQ-004**: El campo de token se muestra vacío. Al guardar, el token **solo** se envía si el usuario ingresó uno nuevo; si queda vacío, se mantiene el token existente (update parcial — el backend ya lo soporta vía `SetTelegramBotToken` con `setIfNonEmpty`, `internal/services/system_settings.go:769-771`).
5. **REQ-005**: Se elimina el botón "Reconfigurar" y todo el flujo de borrado total (`DeleteTelegramBot` handler, ruta `DELETE /api/system-settings/telegram-bot`, `ClearTelegramBot`, `disconnectTelegramBot` del cliente API). Apagar el toggle maestro `enabled=OFF` deshabilita el bot pero **conserva** token y chat_ids para re-encenderlo.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-006**: Actualizar el hint de "Chat IDs permitidos" para aclarar que las alertas de Telegram se envían a esos chats y que se necesita al menos uno para activarlas (reemplaza el hint actual engañoso "Si queda vacío, el bot responde a cualquier chat", que solo aplica a los comandos interactivos, no a las alertas push).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-007**: Mantener el badge de estado "Bot configurado" como informativo (sin botón de borrado).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Sin impacto medible; solo se agrega un campo a un response JSON.
- **Seguridad**: El token del bot **nunca** se expone en la API (se conserva la política actual). Los chat_ids no son secretos y ya se validan en el backend.
- **Almacenamiento**: Sin cambios de esquema ni migraciones (los datos ya viven en `system_settings`).
- **Disponibilidad**: Sin impacto en health check ni schedulers.
- **iHost**: Sin dependencias nuevas; solo JS frontend y un campo extra en el response Go.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se analizó el flujo completo del bot de Telegram:

- **DB de producción** (`app.db`, copia `app-20260919-bug-telegram.db`): `telegram_bot_enabled=1` y `telegram_bot_token` presentes; **no existe** `telegram_bot_chat_ids`. Todos los `alerts.telegram_enabled=0`.
- **Gate de alertas** (`frontend/src/pages/SettingsAlertsPage.tsx:45-47`): `tbActive = enabled && configured && chat_ids_count > 0`. Con `chat_ids_count=0` los toggles de Telegram están deshabilitados.
- **Form del bot** (`frontend/src/pages/SettingsTelegramBotPage.tsx`): cuando `tbConfigured` es true, los inputs de token/chat_ids no se renderizan; el botón "Reconfigurar" llama a `api.systemSettings.disconnectTelegramBot()` → `DeleteTelegramBot` → `ClearTelegramBot` (borra token, chat_ids y apaga).
- **Backend** (`internal/services/system_settings.go`): `GetTelegramBotConfigPublic` expone `ChatIDsCount` pero no la lista. `SetTelegramBotToken` ya implementa update parcial (`setIfNonEmpty`). `SetTelegramBotChatIDs` persiste la lista completa (semántica "set").
- **Envío de alertas** (`internal/services/telegram_bot.go`): `SendAlerts` envía a todos los chat_ids de la allowlist; con allowlist vacía no envía nada. El hint actual del form ("Si queda vacío, el bot responde a cualquier chat") es correcto para comandos interactivos (`isAuthorized`) pero engañoso para alertas push.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A: Form siempre editable con chat_ids precargados + eliminar borrado | Resuelve el problema de raíz; no hay dead code; update parcial ya soportado por backend | Cambio mayor en frontend + eliminación de endpoint | ✅ Seleccionada |
| B: Solo precargar chat_ids, mantener botón Reconfigurar como reset opcional | Cambio menor | Deja dead code (wipe) y un camino confuso; contradice lo pedido por el usuario | ❌ Rechazada |
| C: Fix solo de datos (INSERT `telegram_bot_chat_ids` en prod) | Desbloquea la UI inmediatamente | No resuelve la UX del formulario; requiere conocer el chat_id y operar sobre la DB de producción | ❌ Rechazada como fix único (se ofrece como acción manual complementaria al usuario) |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Exponer `telegram_bot_chat_ids` (lista) en la config pública**
- **Contexto**: La UI necesita precargar los chat_ids para permitir editarlos sin borrarlos. Hoy solo se expone el conteo (`ChatIDsCount`).
- **Decisión**: Se agrega `ChatIDs []string` con JSON tag `telegram_bot_chat_ids` a `models.TelegramBotConfigPublic` y se incluye en las respuestas de `GET` y `PUT /api/system-settings`.
- **Consecuencias**: Los chat_ids dejan de ser "ocultos" en la API, pero no son credenciales (el token sigue siendo el único secreto y no se expone). El campo permite que el form precargue y edite la allowlist con precisión.

**ADR-002: Eliminar el flujo de borrado total del bot**
- **Contexto**: El usuario exige poder cambiar token/chat_ids sin borrar la información actual; el botón "Reconfigurar" borra todo.
- **Decisión**: Se elimina `DeleteTelegramBot` (handler), la ruta `DELETE /api/system-settings/telegram-bot`, `ClearTelegramBot` (service), `disconnectTelegramBot` (cliente API) y sus claves i18n. El toggle `enabled=OFF` deshabilita el bot conservando credenciales.
- **Consecuencias**: Menos código y una sola semántica clara ("editar" siempre disponible; apagar conserva). Se actualiza el test unitario que verificaba el borrado (`telegram_bot_test.go:615-622`) para verificar que deshabilitar conserva token/chat_ids.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[SettingsTelegramBotPage.tsx] --GET/PUT /api/system-settings--> [SystemSettingsHandlers]
        |                                                              |
        |  telegram_bot_chat_ids (lista, precargada)                    v
        |  telegram_bot_configured, telegram_bot_chat_ids_count    [SystemSettingsService]
        v                                                              |
[SettingsAlertsPage.tsx] <-- chat_ids_count > 0 --> habilita toggles    v
                                                                 [system_settings table]
```

### 4.2 Componentes

#### 4.2.1 `internal/models/telegram_bot_config.go`
- **Responsabilidad**: Modelo de configuración pública del bot.
- **Interfaz**: Se agrega `ChatIDs []string` (`json:"telegram_bot_chat_ids,omitempty"`) a `TelegramBotConfigPublic`. El token sigue sin exponerse.

#### 4.2.2 `internal/services/system_settings.go`
- **Responsabilidad**: Servicio de settings.
- **Interfaz**: `GetTelegramBotConfigPublic` setea `ChatIDs: cfg.ChatIDs`. Se elimina `ClearTelegramBot`.
- **Dependencias**: `parseCommaList` (existente).

#### 4.2.3 `internal/api/system_settings_handlers.go`
- **Responsabilidad**: Handlers HTTP.
- **Interfaz**: En `GetSystemSettings` y `UpdateSystemSettings` se agrega `"telegram_bot_chat_ids": telegramBot.ChatIDs` a las respuestas. Se elimina `DeleteTelegramBot`.
- **Dependencias**: `GetTelegramBotConfigPublic`.

#### 4.2.4 `internal/api/routes.go`
- **Responsabilidad**: Registro de rutas.
- **Interfaz**: Se elimina la línea `DELETE /api/system-settings/telegram-bot`.

#### 4.2.5 `frontend/src/pages/SettingsTelegramBotPage.tsx`
- **Responsabilidad**: Página de configuración del bot.
- **Interfaz**: 
  - Carga `telegram_bot_chat_ids` del API y precarga `tbChatIDs`.
  - Los campos token/chat_ids + botón Guardar se renderizan SIEMPRE que `tbEnabled`.
  - Token: input password vacío con placeholder/hint "dejar vacío para mantener el actual".
  - `handleSave`: envía `telegram_bot_token` solo si no vacío; siempre envía `telegram_bot_chat_ids`.
  - Se elimina `handleReconfigure` y la dependencia de `disconnectTelegramBot`.

#### 4.2.6 `frontend/src/api/index.ts`
- **Responsabilidad**: Cliente API.
- **Interfaz**: Se agrega `telegram_bot_chat_ids: string[]` al tipo de `systemSettings.get()`. Se elimina `disconnectTelegramBot`.

#### 4.2.7 `frontend/src/pages/SettingsAlertsPage.tsx`
- **Responsabilidad**: Página de alertas.
- **Interfaz**: Sin cambios de código; el gate `chat_ids_count > 0` queda igual y se desbloquea al setear chat_ids.

### 4.3 Modelo de datos

Sin cambios de esquema. Solo se utiliza la clave existente `telegram_bot_chat_ids` en `system_settings` (formato comma-separated, ya soportada por `SetTelegramBotChatIDs` y `parseCommaList`).

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/system-settings`

**Response 200** (cambios):
```json
{
  "telegram_bot_enabled": true,
  "telegram_bot_configured": true,
  "telegram_bot_chat_ids_count": 1,
  "telegram_bot_chat_ids": ["123456789"],
  "telegram_bot_separator_length": 20,
  "telegram_bot_show_months": 1
}
```

#### Endpoint: `PUT /api/system-settings`

**Request** (cambios; el token solo si se quiere reemplazar):
```json
{
  "telegram_bot_token": "NUEVO_TOKEN",
  "telegram_bot_chat_ids": "123456789,987654321"
}
```

**Response 200**: incluye `telegram_bot_chat_ids` igual que el GET.

**Response Error**:
```json
{
  "error": "internal_error",
  "message": "descripción"
}
```

#### Endpoint eliminado: `DELETE /api/system-settings/telegram-bot`

Se retira (dejaba de existir el flujo de borrado). Cualquier cliente que lo invoque recibe 404.

### 4.5 Dependencias

- **Internas**: `internal/models/telegram_bot_config.go`, `internal/services/system_settings.go`, `internal/api/system_settings_handlers.go`, `internal/api/routes.go`, `internal/services/telegram_bot_test.go`, `frontend/src/pages/SettingsTelegramBotPage.tsx`, `frontend/src/api/index.ts`, `frontend/public/i18n/{es,en}.json`.
- **Externas**: Ninguna.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado un bot configurado (token + chat_ids), al abrir `Configuración → Bot de Telegram` se ven los campos de token (vacío) y chat_ids (precargados) sin necesidad de "Reconfigurar".
- [ ] CA-002: Dado un bot configurado, al guardar con el token vacío y los chat_ids editados, el token existente se mantiene y los chat_ids se actualizan.
- [ ] CA-003: Dado un bot configurado, al guardar con un token nuevo, el token se reemplaza.
- [ ] CA-004: Dado un bot configurado, ya NO existe el botón "Reconfigurar" ni ningún flujo que borre token/chat_ids.
- [ ] CA-005: Dado un bot activo, al apagar el toggle (enabled=OFF) y volverlo a encender, el token y los chat_ids siguen presentes.
- [ ] CA-006: Dado un bot con al menos un chat_id seteado, los toggles de Telegram en `Configuración → Alertas` quedan habilitados y pueden activarse por alerta.
- [ ] CA-007: El API nunca devuelve el token del bot en las respuestas.
- [ ] CA-DARK: Los inputs del formulario del bot usan tokens del tema (`bg-card`, `text-text`) y se verificó legibilidad en darkmode (texto y placeholders).

### 5.2 No funcionales

- [ ] CA-NF-001: `go build ./...` y `go test ./...` pasan en local.
- [ ] CA-NF-002: El frontend compila (`npm run build` en `frontend/`) sin errores de tipos.

### 5.3 Testing

- **Unit tests**: 
  - `telegram_bot_test.go`: actualizar el caso `Clear` → verificar que deshabilitar el bot conserva token/chat_ids; verificar que `GetTelegramBotConfigPublic` devuelve la lista de chat_ids.
  - `alerts_handlers_test.go`: verificar que persiste `telegram_enabled` con chat_ids presentes (existente).
- **Integration tests**: GET/PUT `/api/system-settings` con el flujo de edición de token/chat_ids.
- **E2E tests**: Manual: configurar bot con token+chat_id, activar alerta de Telegram, verificar envío al chat.
- **Carga/Performance**: No aplica (sin impacto).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Backend: modelo + service + handlers + ruta (ChatIDs en public, eliminar DELETE) | 0.5 día | Ninguna |
| 2 | Backend: actualizar tests unitarios | 0.25 día | Fase 1 |
| 3 | Frontend: form siempre editable, precarga chat_ids, eliminar reconfigure | 0.5 día | Fase 1 |
| 4 | i18n es/en + build frontend | 0.25 día | Fase 3 |
| 5 | Build Go, tests, validación en local y en darkmode | 0.25 día | Fases 1-4 |

### 6.2 Milestones

1. **MVP**: Backend + frontend con form editable y chat_ids precargados, sin flujo de borrado.
2. **V1.0**: i18n actualizado, tests verdes, app corriendo en local para evaluación del usuario.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Clientes o scripts existentes dependen del DELETE del bot | Baja | Medio | Endpoint de borrado solo lo usaba la UI; se elimina y la UI ya no lo invoca |
| El hint nuevo de chat_ids confunde a usuarios que solo usan comandos | Baja | Bajo | Texto aclara la diferencia entre comandos (cualquier chat) y alertas (solo chat_ids) |
| Precargar chat_ids expone información sensible | Baja | Bajo | Los chat_ids no son credenciales; el token (único secreto) no se expone |
| Update del token vacío borre el token por error de tipos | Baja | Medio | `SetTelegramBotToken` usa `setIfNonEmpty`; el frontend solo envía el campo si no está vacío |

## 8. Notas y Referencias

- Specs relacionadas: SPEC-079 (bot de Telegram), SPEC-088 (alertas push por Telegram).
- DB analizada: `/home/paulomcnally/p40la-db-backups/app-20260919-bug-telegram.db` (copia de producción).
- Claves de config: `telegram_bot_enabled`, `telegram_bot_token`, `telegram_bot_chat_ids` (formato comma-separated), `telegram_bot_separator_length`, `telegram_bot_show_months`.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-19 | Agente opencode | Creación inicial de la especificación |
| 2026-09-19 | Agente opencode | Implementación (ChatIDs en config pública, form editable con precarga, eliminación del flujo de borrado, i18n, tests) y validación manual del usuario |
| 2026-09-19 | Agente opencode | Release: merge a main (d7f4716), push, issue #92 cerrado con label spec/released |