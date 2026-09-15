---
title: "Rediseño UX de la página de Configuración (índice + subpáginas)"
id: "SPEC-075"
status: "pending_execution"
author: "Claude (a pedido de paulomcnally)"
created: "2026-09-14"
updated: "2026-09-14"
github_issue: 78
---

# Rediseño UX de la página de Configuración (índice + subpáginas)

**ID**: SPEC-075
**Estado**: pending_execution
**Autor**: Claude (a pedido de paulomcnally)
**Creado**: 2026-09-14
**Actualizado**: 2026-09-14

---

## 1. Resumen Ejecutivo

`SettingsPage.tsx` concentra hoy **8 secciones heterogéneas en un solo scroll de ~1050 líneas / ~2600px**: General, Facturación automática, Alertas (8 tipos x 2 canales), Alertas por email (SMTP + destinatarios + secretos), Voice Monkey, Webhooks (API key), Monedas y Formato de moneda. Todo se renderiza con el mismo peso visual, mezclando toggles de uso frecuente con configuración de infraestructura que se toca una vez (SMTP, API keys). El resultado, confirmado por captura de pantalla del usuario, es una página larga y difícil de escanear o de usar para encontrar un ajuste puntual.

Esta spec propone convertir `/settings` en una **página índice** (lista de categorías con subtítulo de estado, sin controles inline salvo Idioma y Modo oscuro) que navega a **subpáginas dedicadas** por categoría — exactamente el patrón que el propio repo ya usa para `LanguagePage` y `CurrencyFormPage` (drill-down con flecha atrás vía `BACK_ROUTES`, ver SPEC-063). No es un patrón nuevo: es completar uno que ya existe a medias.

Es un cambio **puramente de frontend**: sin migraciones, sin cambios de API/contratos, sin impacto en el backend Go ni en SQLite. El riesgo para el iHost (recursos limitados) es mínimo — se reutilizan los mismos componentes (`Toggle`, `Select`, `EmailRecipientsModal`, `Icon`) y las mismas llamadas a `api.systemSettings.get()`, solo repartidas en varios archivos en vez de uno.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: `/settings` se convierte en índice: lista de filas navegables (ícono + título + subtítulo de estado + chevron), agrupadas en secciones. Se mantienen inline únicamente: fila "Idioma" (ya navega a `/settings/language`) y el toggle "Modo oscuro" (un solo control, no amerita subpágina).
2. **REQ-002**: Nueva subpágina `/settings/facturacion` — Hora de generación, Hora de check de alertas, Formato de hora (12h/24h). Misma lógica y llamadas que hoy.
3. **REQ-003**: Nueva subpágina `/settings/alertas` — catálogo de alertas (Seguros vencidos, Nueva factura, Resumen diario, Pensión x5, Cuotas de deudas que vencen hoy) con toggles Mail/Alexa por fila, igual que hoy.
4. **REQ-004**: Nueva subpágina `/settings/alertas/email` — Activar alertas por email, configuración SMTP, destinatarios (reutiliza `EmailRecipientsModal`), botón "Enviar email de prueba".
5. **REQ-005**: Nueva subpágina `/settings/alertas/voz` — Activar Voice Monkey, estado "configurado", botón Reconfigurar, toggle "Enviar alertas", botón "Probar aviso".
6. **REQ-006**: Nueva subpágina `/settings/webhooks` — Habilitar webhooks, URL base, API key global (copiar/regenerar).
7. **REQ-007**: Nueva subpágina `/settings/monedas` — listado de monedas + alta de moneda (la creación/edición sigue en `/settings/currency` y `/settings/currency/:id`, sin cambios).
8. **REQ-008**: Nueva subpágina `/settings/formato-moneda` — presets rápidos, separador de miles/decimal, dígitos decimales, vista previa, guardar.
9. **REQ-009**: Cada fila del índice muestra un subtítulo de estado en vivo (ej. "7:00 AM · 10:00 PM", "3 destinatarios", "Configurado", "Habilitado") para dar contexto sin necesidad de entrar.
10. **REQ-010**: Registrar todas las rutas nuevas en `BACK_ROUTES` (`frontend/src/components/DashboardLayout.tsx`) para que en móvil muestren flecha atrás hacia `/settings` en vez de hamburguesa, según convención ya documentada (SPEC-063).
11. **REQ-011**: Cero cambios de comportamiento funcional: todas las llamadas a `api.*`, validaciones y efectos secundarios existentes se preservan 1:1; esto es una reorganización de UI, no una reescritura de lógica.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-012**: Extraer un componente `SettingsNavRow` (ícono + título + subtítulo + chevron/pill) reutilizable, para no duplicar el JSX de fila que ya se repite ad hoc en el archivo actual.
2. **REQ-013**: Extraer un componente `StatusPill` (`Activado` / `Desactivado` / `Configurado` / `Pendiente`) para los subtítulos de estado del índice.
3. **REQ-014**: Preservar el comportamiento actual de "aviso de configuración pendiente" (hoy implementado como auto-apertura de acordeón vía `emailNeedsConfig` / `vmNeedsConfig`) como un badge de advertencia (⚠) en la fila correspondiente del índice, en vez de auto-navegar o auto-expandir nada.
4. **REQ-015**: Agrupar visualmente el índice en dos bloques: "General y alertas" (uso frecuente) vs. "Avanzado" (Webhooks, Voice Monkey, SMTP — configuración de infraestructura que se toca una vez).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-016**: Buscador simple en la parte superior de `/settings` que filtre las filas del índice por texto (cliente, sin backend).
2. **REQ-017**: Cache compartido de `api.systemSettings.get()` (store simple) para evitar que cada subpágina repita la misma llamada al navegar entre secciones.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: sin nuevas dependencias; el bundle crece marginalmente (más archivos de página, mismo código total repartido). Se espera reducir el DOM renderizado en `/settings` de ~2600px de scroll a <400px.
- **Seguridad**: sin cambios — los campos sensibles (password SMTP, API key) siguen sin persistirse en el cliente más allá de lo que ya hace el código actual.
- **Compatibilidad**: cero cambios de API/contratos de backend; cero migraciones.
- **iHost**: sin impacto adicional en RAM/CPU; el patrón de fetch por página ya existe hoy (cada sección llama `api.systemSettings.get()` en su propio `useEffect`).
- **UI (obligatorio por AGENTS.md)**: todo `input`/`select`/`textarea` nuevo debe usar tokens del tema (`bg-card`, `text-text`, `text-text-secondary`), nunca colores hardcodeados; verificar legibilidad en darkmode. Toda subpágina nueva debe usar flecha atrás en el header vía `BACK_ROUTES`, nunca un link "← Título" en el contenido.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- Se inspeccionó `frontend/src/pages/SettingsPage.tsx` (1053 líneas) completo.
- Se confirmó que `LanguagePage.tsx` y `CurrencyFormPage.tsx` ya implementan el patrón "subpágina con flecha atrás" para dos de las ocho secciones actuales (`settings/language`, `settings/currency(/:id)`), registradas en `BACK_ROUTES` (`frontend/src/components/DashboardLayout.tsx:24`).
- Se revisó `AGENTS.md`, que exige spec previa a cualquier cambio de UI y documenta el patrón de flecha atrás como regla ya aprendida (SPEC-063).
- Se revisó `docs/specs/templates/spec-template.md` para el formato de esta spec.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A. Acordeones colapsables en una sola ruta `/settings` | Cambio pequeño, una sola ruta | Sigue siendo un archivo gigante; no resuelve el problema de fondo (todo con el mismo peso visual); no permite deep-link a una sección concreta | ❌ Rechazada |
| B. Página índice + subpágina por categoría (patrón ya usado en el repo) | Consistente con `LanguagePage`/`CurrencyFormPage` ya existentes; deep-linkeable; cada subpágina es corta y fácil de mantener; encaja con `BACK_ROUTES` ya documentado | Más archivos nuevos; alguna duplicación de `useEffect` de fetch (mitigable con REQ-017) | ✅ Seleccionada |
| C. Modal por ajuste sobre la página raíz | Mantiene una sola ruta | Rompe con el patrón de flecha atrás ya establecido en AGENTS.md/SPEC-063; peor soporte de botón "atrás" nativo en móvil; no deep-linkeable | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Índice de navegación en vez de acordeón
- **Contexto**: el código actual ya tiene lógica de "auto-abrir" sección si falta configurar (`emailNeedsConfig`, `vmNeedsConfig`).
- **Decisión**: esa lógica se traduce en un badge de advertencia en la fila del índice (REQ-014), no en auto-navegación ni auto-expansión.
- **Consecuencias**: comportamiento más predecible; el usuario decide cuándo entrar a resolver la advertencia.

**ADR-002**: Reutilizar componentes existentes sin modificarlos
- **Contexto**: `Toggle`, `Select`, `EmailRecipientsModal`, `Icon` ya cubren toda la interacción necesaria.
- **Decisión**: no se toca ningún componente compartido; solo se mueve el JSX de cada sección de `SettingsPage.tsx` a su propio archivo de página.
- **Consecuencias**: riesgo de regresión bajo, revisión más simple.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
/settings (índice)
 ├─ Idioma (inline, navega a /settings/language)          [ya existe]
 ├─ Modo oscuro (toggle inline)
 ├─ Facturación automática ───────► /settings/facturacion
 ├─ Alertas ──────────────────────► /settings/alertas
 ├─ Alertas por email ────────────► /settings/alertas/email
 ├─ Voice Monkey (Alexa) ─────────► /settings/alertas/voz
 ├─ Webhooks ─────────────────────► /settings/webhooks
 ├─ Monedas ──────────────────────► /settings/monedas ──► /settings/currency(/:id) [ya existe]
 └─ Formato de moneda ────────────► /settings/formato-moneda
```

### 4.2 Componentes

#### 4.2.1 `SettingsPage.tsx` (reescrito, pasa de ~1050 a ~150 líneas)
- **Responsabilidad**: índice de navegación + Idioma + Modo oscuro.
- **Dependencias**: `SettingsNavRow`, `StatusPill`, `api.systemSettings.get()` (solo para calcular subtítulos de estado).
- **Ubicación**: `frontend/src/pages/SettingsPage.tsx`.

#### 4.2.2 Subpáginas nuevas
- `frontend/src/pages/SettingsBillingPage.tsx` (REQ-002)
- `frontend/src/pages/SettingsAlertsPage.tsx` (REQ-003)
- `frontend/src/pages/SettingsEmailAlertsPage.tsx` (REQ-004)
- `frontend/src/pages/SettingsVoiceMonkeyPage.tsx` (REQ-005)
- `frontend/src/pages/SettingsWebhooksPage.tsx` (REQ-006)
- `frontend/src/pages/SettingsCurrenciesPage.tsx` (REQ-007)
- `frontend/src/pages/SettingsCurrencyFormatPage.tsx` (REQ-008)

Cada una recibe el bloque de estado/`useEffect`/handlers correspondiente, recortado tal cual del `SettingsPage.tsx` actual (líneas indicadas a modo de referencia, se deben re-verificar al implementar):
- Facturación: hooks de `billingHour`, `alertCheckHour`, `hourFormat` y sus handlers.
- Alertas: `alerts`, `loadAlerts`, toggles de canal por alerta.
- Alertas por email: `smtp*`, `alertEmails`, `smtpConfigured`, `emailAlertsEnabled`, `showRecipientModal`, `sendingTest`, `savingEmail`.
- Voice Monkey: `vm*`, `savingVoice`, `testingVoice`.
- Webhooks: `webhook*`, `savingWebhookBaseUrl`.
- Monedas: el bloque `currencies.map(...)` + botón "Crear moneda".
- Formato de moneda: `cf*`, `savingFormat`, `applyFormatPreset`, `handleSaveFormat`.

#### 4.2.3 `SettingsNavRow` (nuevo, REQ-012)
- **Responsabilidad**: fila de índice reutilizable (ícono, título, subtítulo, `StatusPill` opcional, chevron).
- **Interfaz**: `{ icon, title, subtitle?, status?, badge?, onClick }`.
- **Ubicación**: `frontend/src/components/SettingsNavRow.tsx`.

#### 4.2.4 `StatusPill` (nuevo, REQ-013)
- **Responsabilidad**: pastilla de estado (`Activado` verde / `Desactivado` gris / `Configurado` verde / `Pendiente` ámbar).
- **Ubicación**: `frontend/src/components/StatusPill.tsx`.

### 4.3 Modelo de datos

Sin cambios. No se agregan ni modifican entidades ni columnas.

### 4.4 APIs / Contratos

Sin cambios. Todas las subpáginas consumen los mismos endpoints ya existentes:
- `GET /api/system-settings` (`api.systemSettings.get`)
- `PUT/PATCH` correspondientes a facturación, email, voice monkey, webhooks, formato de moneda (los mismos que hoy usa `SettingsPage.tsx`)
- `GET /api/webhooks/api-key`, regenerar API key
- `GET/POST /api/currencies`
- `GET/POST /api/alerts`

### 4.5 Dependencias

- **Internas**: `DashboardLayout.tsx` (agregar rutas a `BACK_ROUTES`), `App.tsx` (registrar nuevas `<Route>`), `Icons.tsx` (agregar íconos si faltan para Facturación/Alertas/Email/Voz/Webhooks/Monedas/Formato, siguiendo el mismo estilo de trazo que los existentes).
- **Externas**: ninguna nueva.
- **i18n**: nuevas keys en `frontend/public/i18n/{es,en}.json` bajo `settings.nav.*` (títulos y subtítulos de fila del índice) y `settings.*_page_title` para el header de cada subpágina. **Recordatorio crítico del propio AGENTS.md**: editar solo en `frontend/public/i18n/`, nunca en `public/i18n/` (se sobrescribe en build).

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado que el usuario visita `/settings`, ve una lista de filas navegables con subtítulo de estado, sin controles inline salvo Idioma y Modo oscuro.
- [ ] CA-002: Dado que toca "Facturación automática", navega a `/settings/facturacion` y ve/edita Hora de generación, Hora de check y Formato de hora con el mismo comportamiento de guardado que hoy.
- [ ] CA-003: Igual que CA-002 para Alertas (`/settings/alertas`), Alertas por email (`/settings/alertas/email`), Voice Monkey (`/settings/alertas/voz`), Webhooks (`/settings/webhooks`), Monedas (`/settings/monedas`) y Formato de moneda (`/settings/formato-moneda`) — cada una reproduce 1:1 la funcionalidad actual (togglear, guardar, probar, copiar, regenerar) sin pérdida de features.
- [ ] CA-004: En móvil, cada subpágina nueva muestra flecha atrás (no hamburguesa) que regresa a `/settings`.
- [ ] CA-005: Si `email_alerts_enabled=true` y SMTP no está configurado, la fila "Alertas por email" del índice muestra un badge de advertencia. Comportamiento equivalente para Voice Monkey si aplica.
- [ ] CA-006: Ninguna llamada a la API cambia de contrato; no hay migraciones ni cambios de backend.
- [ ] CA-007: El scroll de `/settings` baja de ~2600px a menos de 400px (solo filas de índice).
- [ ] CA-DARK: Todo input/select nuevo usa tokens del tema (`bg-card`, `text-text`) y se verificó legibilidad en darkmode.
- [ ] CA-BACK: Todas las subpáginas nuevas están registradas en `BACK_ROUTES` y no existe ningún link "← Título" dentro del contenido.

### 5.2 No funcionales

- [ ] CA-NF-001: El bundle de frontend no crece más de lo esperado por la sola división de archivos (sin nuevas dependencias de npm).
- [ ] CA-NF-002: `npm run build` en `frontend/` completa sin errores y `public/i18n/es.json` sirve las nuevas keys tras el build (verificar con `curl http://localhost:8088/i18n/es.json`).

### 5.3 Testing

- **Unit tests**: si existen tests de componentes de settings, actualizarlos a las nuevas rutas/archivos; si no existen, no es bloqueante (el proyecto prioriza QA manual local, ver AGENTS.md).
- **Integration tests**: ninguno nuevo requerido (no cambia backend).
- **E2E manual**: recorrer las 7 subpáginas nuevas + índice en desktop y móvil, en modo claro y oscuro, verificando que cada acción (guardar, probar, copiar, regenerar, activar/desactivar) funciona igual que en la página actual antes del cambio.
- **Carga/Performance**: no aplica (no hay cambio de backend/consultas SQLite).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Crear `SettingsNavRow` y `StatusPill`; reescribir `SettingsPage.tsx` como índice (con subtítulos de estado) | 0.5 día | Ninguna |
| 2 | Extraer Facturación, Alertas, Monedas y Formato de moneda a subpáginas (sin secretos, menor riesgo) | 1 día | Fase 1 |
| 3 | Extraer Alertas por email, Voice Monkey y Webhooks (contienen secretos/API keys y lógica de "needs config") | 1 día | Fase 1 |
| 4 | i18n (`settings.nav.*`), íconos nuevos si faltan, registrar `BACK_ROUTES`, QA manual completo en mobile/desktop/darkmode | 0.5 día | Fases 2 y 3 |

**Total estimado**: ~3 días.

### 6.2 Milestones

1. **MVP**: Fases 1-2 liberadas — General, Facturación, Alertas, Monedas y Formato de moneda ya navegables por separado.
2. **V1.0**: Fases 3-4 completas — Alertas por email, Voice Monkey y Webhooks migrados, i18n y QA cerrados.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Llamadas duplicadas a `api.systemSettings.get()` al navegar entre subpáginas | Media | Bajo | Impacto mínimo (SQLite local, un solo usuario); opcionalmente implementar REQ-017 (cache) si se nota lentitud |
| Perder el comportamiento de "aviso de configuración pendiente" al quitar el auto-acordeón | Media | Medio | Cubierto explícitamente por CA-005; QA manual dedicado a este caso antes de cerrar la spec |
| Usuarios con la página `/settings` guardada en favoritos/scroll a media página pierden esa posición | Baja | Bajo | Cambio de UX esperado y deseado; no requiere mitigación técnica |
| Falta algún ícono nuevo en `Icons.tsx` y se usa un placeholder poco claro | Baja | Bajo | Revisar `Icons.tsx` en Fase 1 y agregar los que falten con el mismo estilo de trazo |

## 8. Notas y Referencias

- Motivada por captura de pantalla del usuario de `/settings` en producción y su comentario: "hoy dia es un completo desastre de ui".
- Precedente de patrón ya usado en el repo: `LanguagePage.tsx`, `CurrencyFormPage.tsx`.
- Regla de flecha atrás: SPEC-063.
- Regla de inputs con tokens de tema y verificación en darkmode: SPEC-060.
- Regla de fuente de verdad de i18n (`frontend/public/i18n/`): ver sección "Reglas críticas" de `AGENTS.md`.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-14 | Claude (a pedido de paulomcnally) | Creación inicial de la especificación, a partir de una captura de pantalla y solicitud de rediseño de UX de `/settings`. |
| 2026-09-14 | opencode | Copiado del spec desde Downloads, issue de GitHub #78 creado, estado movido a `pending_execution`. |
