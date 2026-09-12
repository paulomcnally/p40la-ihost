---
title: "Fix copiar URL de webhook en modal de servicio y quitar API key"
id: "SPEC-072"
status: "in_progress"
author: "paulomcnally"
created: "2026-09-12"
updated: "2026-09-12"
github_issue: 75
---

# Fix copiar URL de webhook en modal de servicio y quitar API key

**ID**: SPEC-072  
**Estado**: in_progress  
**Autor**: paulomcnally  
**Creado**: 2026-09-12  
**Actualizado**: 2026-09-12

---

## 1. Resumen Ejecutivo

En el modal de webhook dentro de un servicio (abierto desde el menú de 3 puntos de un servicio o de una factura), al hacer click en el ícono de copiar la URL del webhook se muestra un toast de error: *"Ocurrió un error, intenta de nuevo"*. El texto en realidad **sí** se copia en la mayoría de los casos, pero el toast de error aparece igual.

La causa raíz: el modal usa `navigator.clipboard.writeText()`, que **solo funciona en contextos seguros** (HTTPS o `localhost`). El iHost se accede vía `http://ihost.local:8088`, que NO es un contexto seguro, por lo que `navigator.clipboard` no está disponible y el `catch` muestra el error aunque la copia (fallback del navegador) pudo haber funcionado. El mismo problema existe en `SettingsPage.tsx` para copiar la API key global.

Además, el modal muestra el campo **"API key del webhook"**, que duplica información que ya vive en **Configuración → Webhooks**. El usuario solicita eliminar ese campo del modal: la API key es global (aplica a todos los webhooks) y no aporta valor mostrarla por servicio, reduciendo también el ruido visual del modal.

No se espera impacto en memoria, almacenamiento ni SQLite: es un cambio exclusivo de frontend (componente `WebhookModal.tsx`, página de settings y utilidad de clipboard compartida).

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Corregir la copia de la URL del webhook en `WebhookModal.tsx` para que el toast muestre "URL copiada" y no el error, en cualquier contexto (HTTP/iHost incluido).
2. **REQ-002**: Eliminar el bloque "API key del webhook" del modal de webhook del servicio (label, input, botón de copiar y descripciones asociadas). La API key se gestiona únicamente en Configuración.
3. **REQ-003**: Crear una utilidad compartida de copiado (`copyToClipboard`) con fallback a `document.execCommand('copy')` para contextos no seguros, y usarla tanto en `WebhookModal.tsx` como en `SettingsPage.tsx` (que tiene el mismo bug para la API key).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-004**: Limpiar las claves i18n no utilizadas tras eliminar el bloque de API key del modal (`services.webhook_api_key`, `services.webhook_api_key_desc`, `services.webhook_api_key_copy`) en `es.json` y `en.json`, o conservarlas solo si siguen usadas en otro lugar.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-005**: Verificar en darkmode que el modal sigue siendo legible tras los cambios (sin campos adicionales).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Sin impacto. Cambio de UI sin llamadas extra.
- **Seguridad**: La API key global deja de exponerse en el modal por servicio (reducción de superficie de exposición accidental). No se agregan secretos al frontend.
- **Almacenamiento**: Sin impacto.
- **Disponibilidad**: Sin impacto.
- **iHost**: Sin dependencias nuevas; solo JS en frontend.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- `navigator.clipboard` solo está disponible en **contextos seguros** (HTTPS, `localhost` o `127.0.0.1`). En `http://ihost.local:8088` (origen no seguro en una LAN), `navigator.clipboard` es `undefined` o rechaza la promesa → se ejecuta el `catch` y se muestra el toast de error.
- Alternativa estándar para contextos no seguros: crear un `<textarea>` temporal, `document.execCommand('copy')` y eliminarlo. Es la técnica histórica soportada por todos los navegadores.
- Patrón ya usado en el repo: copiado directo con `navigator.clipboard.writeText` en `WebhookModal.tsx:46` y `SettingsPage.tsx:429`. No existe una utilidad compartida.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Usar `navigator.clipboard` directo | API moderna, asíncrona | Falla en HTTP/iHost (contexto no seguro) | ❌ Rechazada (causa del bug) |
| Fallback con `<textarea>` + `execCommand` | Funciona en cualquier contexto | Método deprecated pero 100% soportado; mantiene el texto en el DOM un instante | ✅ Seleccionada como fallback |
| Utilidad compartida `copyToClipboard` con `navigator.clipboard` primero y fallback `execCommand` | Reutilizable, corrige ambos sitios a la vez | Archivo nuevo en `frontend/src/utils/` | ✅ Seleccionada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Utilidad compartida de clipboard en `frontend/src/utils/clipboard.ts`
- **Contexto**: Dos componentes (`WebhookModal`, `SettingsPage`) repiten el mismo patrón defectuoso de copiado.
- **Decisión**: Crear `copyToClipboard(text: string): Promise<void>` que intente `navigator.clipboard.writeText` y, si falla, use el fallback con `textarea` + `document.execCommand('copy')`. Ambos componentes la usan.
- **Consecuencias**: Elimina duplicación, corrige el bug en los dos sitios y centraliza futuras mejoras. Cambio mínimo de comportamiento.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[WebhookModal.tsx] ──copyToClipboard()──▶ [utils/clipboard.ts] ──▶ navigator.clipboard (secure ctx)
                                                                    └─fallback textarea+execCommand (HTTP)
[SettingsPage.tsx] ──copyToClipboard()──┘
```

### 4.2 Componentes

#### 4.2.1 `frontend/src/utils/clipboard.ts` (nuevo)
- **Responsabilidad**: Copiar texto al portapapeles funcionando en contextos seguros y no seguros.
- **Interfaz**: `copyToClipboard(text: string): Promise<void>` — resuelve si se copió, rechaza si falla todo.
- **Dependencias**: Ninguna (vanilla JS).
- **Ubicación**: `frontend/src/utils/clipboard.ts` (coexiste con `currency.ts`, `age.ts`).

#### 4.2.2 `frontend/src/components/WebhookModal.tsx` (modificado)
- **Responsabilidad**: Modal de webhook por servicio. Se elimina el bloque de API key y se usa la utilidad compartida para copiar la URL.
- **Cambios**:
  - Eliminar `webhookApiKey` state, la llamada a `api.webhooks.getApiKey()` y el fetch de la API key en `loadData`.
  - Eliminar el bloque JSX del input/botón de API key (líneas ~132-153).
  - Reemplazar el `copyText` interno por `copyToClipboard` de la utilidad.
  - Mantener el botón de regenerar UUID de la URL y el schema description.

#### 4.2.3 `frontend/src/pages/SettingsPage.tsx` (modificado)
- **Responsabilidad**: Copiar API key global de webhooks.
- **Cambios**: Reemplazar `navigator.clipboard.writeText(webhookApiKey)` por `copyToClipboard(webhookApiKey)`.

### 4.3 Modelo de datos

Sin cambios. No se toca SQLite ni esquema.

### 4.4 APIs / Contratos

Sin cambios en backend. La utilidad de frontend no expone API.

### 4.5 Dependencias

- **Internas**: `WebhookModal.tsx`, `SettingsPage.tsx`, i18n (`es.json`, `en.json`).
- **Externas**: Ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Al abrir el modal de webhook de un servicio y hacer click en "Copiar URL", se muestra el toast de éxito ("URL copiada") en vez de "Ocurrió un error, intenta de nuevo", tanto en local (HTTP) como en iHost.
- [ ] CA-002: La URL efectivamente queda en el portapapeles (verificable pegando en un editor).
- [ ] CA-003: El modal de webhook del servicio YA NO muestra el campo "API key del webhook" ni su botón de copiar ni sus descripciones.
- [ ] CA-004: La API key global sigue siendo visible/copiable únicamente en Configuración → Webhooks y la copia muestra toast de éxito.
- [ ] CA-005: El botón "Regenerar UUID" de la URL sigue funcionando y actualiza la URL mostrada.
- [ ] CA-DARK: El modal se verifica en darkmode y sigue legible (sin regresión por quitar el bloque).
- [ ] CA-BACK: No aplica (no es página de detalle).

### 5.2 No funcionales

- [ ] CA-NF-001: No se agregan dependencias ni llamadas extra a la API; el modal solo pide `systemSettings.get()` para el estado enable y base URL.

### 5.3 Testing

- **Unit tests**: No hay framework de tests de frontend configurado en el repo; validación manual.
- **Integration tests**: Validar copiado en HTTP local (fallback) y en contexto seguro si está disponible.
- **E2E tests**: Abrir modal → copiar URL → verificar toast de éxito; verificar ausencia del bloque API key.
- **Carga/Performance**: Sin impacto.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Crear `frontend/src/utils/clipboard.ts` con `copyToClipboard` (clipboard + fallback execCommand) | 0.5 día | Ninguna |
| 2 | Modificar `WebhookModal.tsx`: usar utilidad, eliminar estado/fetch de API key y bloque JSX | 0.5 día | Fase 1 |
| 3 | Modificar `SettingsPage.tsx`: usar `copyToClipboard` para la API key | 0.25 día | Fase 1 |
| 4 | Limpiar claves i18n no usadas (`services.webhook_api_key*`) en `es.json`/`en.json` | 0.25 día | Fase 2 |
| 5 | Build frontend + pruebas manuales locales + darkmode | 0.5 día | Fase 4 |

### 6.2 Milestones

1. **MVP**: Copiar URL muestra toast de éxito y el bloque API key desaparece del modal.
2. **V1.0**: Settings también usa la utilidad compartida y se limpian claves i18n.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| `document.execCommand('copy')` retorna `false` en algún navegador | Baja | Medio | Si ambos métodos fallan, rechazar y mostrar el error como hoy |
| Borrar claves i18n que otro componente siga usando | Baja | Bajo | Buscar usos de `services.webhook_api_key` antes de eliminarlas |
| Romper el layout del modal al quitar el bloque | Baja | Bajo | Verificar visualmente en móvil y darkmode |

## 8. Notas y Referencias

- MDN: Clipboard API requiere secure context: https://developer.mozilla.org/en-US/docs/Web/API/Clipboard_API
- Espec relacionada: SPEC-069 (webhooks por servicio), SPEC-033/032 (i18n en `frontend/public/i18n/`).
- Regla del repo: la fuente de verdad del i18n es `frontend/public/i18n/`, NO `public/i18n/`.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-12 | paulomcnally | Creación inicial de la especificación |