---
title: "Sección de ayuda/info en configuración SMTP y Voice Monkey (Alexa)"
id: "SPEC-036"
status: "pending_release"
author: "paulomcnally"
created: "2026-08-16"
updated: "2026-08-16"
github_issue: 36
---

# Sección de ayuda/info en configuración SMTP y Voice Monkey (Alexa)

**ID**: SPEC-036  
**Estado**: pending_release  
**Autor**: paulomcnally  
**Creado**: 2026-08-16  
**Actualizado**: 2026-08-16

---

## 1. Resumen Ejecutivo

Los formularios de configuración de **SMTP** (`SettingsPage.tsx:363-405`) y **Voice Monkey / Alexa** (`SettingsPage.tsx:467-489`) piden datos técnicos (host, puerto, token, device) que personas sin experiencia no saben de dónde obtener. Esto genera fricción y abandonos en la configuración.

Se propone agregar una **sección de ayuda/información** discreta y no intrusiva dentro de cada formulario que explique, en lenguaje simple y paso a paso, dónde encontrar cada dato (con enlace directo a la pantalla correspondiente del proveedor). El objetivo es que cualquier usuario pueda completar el formulario sin buscar en internet.

El cambio es **frontend-only** (UI + i18n), respetando las restricciones de iHost: cero dependencias nuevas, cero cambios de backend/DB. La ayuda se muestra como un bloque colapsable/desplegable para no saturar el formulario.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Agregar una sección de ayuda dentro del formulario **SMTP** que indique de dónde sacar cada campo (host, puerto, usuario, contraseña, email remitente, nombre remitente), con enlace directo a la configuración SMTP de Mailgun: `https://app.mailgun.com/mg/sending/paulomcnally.com/settings?tab=smtp`.
2. **REQ-002**: Agregar una sección de ayuda dentro del formulario **Voice Monkey (Alexa)** que indique de dónde obtener el **Key/API token** (enlace: `https://app.voicemonkey.io/tokens`) y el **Device** (enlace: `https://app.voicemonkey.io/speakers`).
3. **REQ-003**: La ayuda debe ser **discreta y no intrusiva** (p. ej. un panel colapsable, o un ícono/botón "¿Dónde encuentro estos datos?" que despliega la info). No debe ocupar espacio permanente ni molestar a quien ya sabe configurar.
4. **REQ-004**: Las instrucciones deben ser en lenguaje simple, paso a paso, pensadas para usuarios sin conocimientos técnicos.
5. **REQ-005**: Internacionalizar todas las cadenas nuevas en `frontend/public/i18n/{es,en}.json` (fuente de verdad, AGENTS.md), correr `npm run build` y verificar con `curl`.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-006**: Los enlaces deben abrirse en pestaña nueva (`target="_blank"` / `rel="noopener"`) por seguridad.
2. **REQ-007**: Mostrar la ayuda **solo** cuando el formulario está en modo edición (SMTP no configurado o Voice Monkey no configurado), no en el estado "Configurado".

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-008**: Incluir un ícono de ayuda (p. ej. `?` o `info`) junto al título de cada campo con su explicación, además del bloque resumen.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Sin impacto (UI estática + texto).
- **Seguridad**: Enlaces externos con `target="_blank" rel="noopener noreferrer"`. No se exponen datos sensibles.
- **Almacenamiento**: Sin cambios de DB.
- **Disponibilidad**: Sin cambios en endpoints.
- **iHost**: Cero dependencias nuevas; solo texto/JS en el bundle estático.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- `frontend/src/pages/SettingsPage.tsx:363-405` — Formulario SMTP (host, port, user, password, from_email, from_name).
- `frontend/src/pages/SettingsPage.tsx:467-489` — Formulario Voice Monkey (token, device).
- `frontend/public/i18n/es.json:103-130` — claves `settings.email_alerts.*`.
- `frontend/public/i18n/es.json:139-158` — claves `settings.voicemonkey.*`.
- Enlaces de referencia aportados por el usuario:
  - SMTP/Mailgun: `https://app.mailgun.com/mg/sending/paulomcnally.com/settings?tab=smtp`
  - Voice Monkey Key/token: `https://app.voicemonkey.io/tokens`
  - Voice Monkey Device/speakers: `https://app.voicemonkey.io/speakers`

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Panel colapsable "¿Dónde encuentro estos datos?" | No intrusivo, visible a pedido | Requiere un toggle extra | ✅ Seleccionada |
| Bloque de ayuda siempre visible | No requiere interacción | Ocupa espacio, molesta a usuarios avanzados | ❌ Rechazada |
| Tooltip en cada campo | Detalle fino | Más código, difícil en móvil | ❌ Rechazada (P2 opcional) |
| Enlace externo directo en el label | Simple | No explica el paso a paso | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Ayuda colapsable (acordeón) dentro del formulario
- **Contexto**: El usuario pidió algo "no tan molesto" para personas que no saben de dónde sacar la info.
- **Decisión**: Un bloque desplegable (botón tipo acordeón, similar al existente SMTP open/close en `SettingsPage.tsx:338-348`) con pasos numerados y enlaces. Colapsado por defecto.
- **Consecuencias**: Cero ruido para usuarios avanzados; ayuda disponible con un click para novatos.

**ADR-002**: Solo en modo edición
- **Contexto**: El estado "Configurado" no necesita ayuda.
- **Decisión**: Renderizar la sección de ayuda únicamente dentro de la rama de formulario no configurado.
- **Consecuencias**: Menos clutter; comportamiento consistente con SMTP/Voice Monkey existentes.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[SettingsPage]
     │
     ├─ SMTP (no configurado) ──▶ [botón "¿Dónde encuentro estos datos?"] ──▶ panel ayuda Mailgun
     │                                       │                                     (pasos + enlace SMTP)
     │                                       ▼
     │                             api/systemSettings  (sin cambios)
     │
     └─ Voice Monkey (no configurado) ──▶ [botón "¿Dónde encuentro estos datos?"] ──▶ panel ayuda VM
                                                   │                                   (paso token + enlace tokens
                                                   │                                    paso device + enlace speakers)
                                                   ▼
                                         api/systemSettings  (sin cambios)
```

### 4.2 Componentes

#### 4.2.1 Bloque de ayuda reutilizable (opcional, recomendado)
- **Responsabilidad**: Renderizar un panel colapsable con título, pasos numerados y enlaces externos.
- **Interfaz**: Props `title`, `children` (contenido/pasos) o similar.
- **Dependencias**: Ninguna nueva.
- **Ubicación**: `frontend/src/components/HelpPanel.tsx` (nuevo, opcional) o inline en `SettingsPage.tsx`.

#### 4.2.2 SettingsPage (secciones SMTP y Voice Monkey)
- **Responsabilidad**: Añadir el bloque de ayuda en cada formulario no configurado.
- **Dependencias**: `HelpPanel` (si se extrae), i18n.
- **Ubicación**: `frontend/src/pages/SettingsPage.tsx`

### 4.3 Modelo de datos

Sin cambios. No se introduce ningún campo nuevo en `system_settings`.

### 4.4 APIs / Contratos

Sin cambios en la API. Solo se modifica el frontend.

### 4.5 Dependencias

- **Internas**: `frontend/src/pages/SettingsPage.tsx`, `frontend/public/i18n/{es,en}.json`.
- **Externas**: Enlaces de Mailgun y Voice Monkey (abiertos en pestaña nueva). Ninguna librería nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: En el formulario SMTP (no configurado) aparece la opción de ayuda que, al expandirla, muestra pasos para obtener host, puerto, usuario y contraseña con enlace a Mailgun.
- [ ] CA-002: En el formulario Voice Monkey (no configurado) aparece la ayuda que indica dónde obtener el Key (enlace a `/tokens`) y el Device (enlace a `/speakers`).
- [ ] CA-003: La ayuda está colapsada por defecto (no intrusiva) y se despliega a pedido.
- [ ] CA-004: La ayuda NO se muestra en el estado "Configurado" (solo en modo edición).
- [ ] CA-005: Los enlaces externos abren en pestaña nueva con `rel="noopener noreferrer"`.
- [ ] CA-006: El texto es comprensible para un usuario no técnico (pasos numerados, lenguaje simple).

### 5.2 No funcionales

- [ ] CA-NF-001: `npm run build` en `frontend/` sin errores y `public/` regenerado; i18n servido por `http://localhost:8088/i18n/es.json`.

### 5.3 Testing

- **Unit tests**: No aplica (solo UI).
- **Integration tests**: No aplican nuevos endpoints.
- **E2E tests**: Expandir ayuda SMTP y Voice Monkey; verificar enlaces; verificar que no aparezca en estado configurado.
- **Carga/Performance**: Sin métricas nuevas.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Crear componente/panel de ayuda + integrar en SMTP y Voice Monkey (modo edición) | 0.5 día | Ninguna |
| 2 | i18n (es/en en `frontend/public/i18n/`), `npm run build`, verificación con `curl` | 0.25 día | Fase 1 |

### 6.2 Milestones

1. **MVP**: Ayuda colapsable en SMTP y Voice Monkey con enlaces y pasos, i18n completo.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Ayuda visible en estado "Configurado" | Baja | Bajo | Condicionar render a la rama de formulario no configurado |
| Perder claves i18n editadas en `public/i18n/` | Baja | Medio | Editar SOLO en `frontend/public/i18n/` (AGENTS.md) + build + `curl` |
| Enlaces rotos por cambio de dominio Mailgun | Baja | Medio | Mantener URLs en i18n/constante centralizada para fácil edición |

## 8. Notas y Referencias

- Pantalla SMTP Mailgun: `https://app.mailgun.com/mg/sending/paulomcnally.com/settings?tab=smtp`
- Voice Monkey tokens: `https://app.voicemonkey.io/tokens`
- Voice Monkey speakers: `https://app.voicemonkey.io/speakers`
- Patrón acordeón existente: `SettingsPage.tsx:338-348`
- Regla i18n: AGENTS.md (fuente de verdad `frontend/public/i18n/`)

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-16 | paulomcnally | Creación inicial de la especificación |
