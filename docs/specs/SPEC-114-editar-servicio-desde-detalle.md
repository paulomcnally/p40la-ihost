---
title: "Opción Editar servicio en el menú de acciones del detalle de servicio"
id: "SPEC-114"
status: "draft"
author: "opencode"
created: "2026-10-21"
updated: "2026-10-21"
github_issue: 117
---

# Opción Editar servicio en el menú de acciones del detalle de servicio

**ID**: SPEC-114  
**Estado**: draft  
**Autor**: opencode  
**Creado**: 2026-10-21  
**Actualizado**: 2026-10-21

---

## 1. Resumen Ejecutivo

Hoy, para editar los datos de un servicio (nombre, institución, monto, moneda, día de facturación, etc.) el usuario debe volver a la lista `/services` y abrir el menú de 3 puntos del card correspondiente. Desde el detalle del servicio (`/services/bills/:serviceId`, por ejemplo `/services/bills/6?tab=analisis`), el menú de 3 puntos del header solo ofrece acciones sobre facturas (subir, crear, renovar) y webhook, pero no permite editar el servicio que se está viendo.

Esta spec agrega la opción **"Editar servicio"** como **primera opción** del menú de acciones del header del detalle del servicio, reutilizando el formulario existente `/services/edit/:id`. El cambio es exclusivamente de frontend: no toca backend, base de datos, APIs ni i18n (las claves ya existen).

Impacto en iHost: nulo. No se agregan dependencias, no hay cambios de esquema ni de memoria en runtime; solo se agrega un ítem a un array de opciones ya renderizado por un componente existente (`CreateMenu`). El bundle no cambia de tamaño de forma significativa.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: En la página de detalle de servicio (`/services/bills/:serviceId`, en cualquiera de sus tabs: `analisis`, `facturas`, `polizas`), el menú de 3 puntos del header (`CreateMenu`) DEBE mostrar la opción **"Editar servicio"** como **primer ítem** del dropdown.
2. **REQ-002**: Al seleccionar **"Editar servicio"**, la app DEBE navegar a `/services/edit/:serviceId` (mismo formulario que usa la lista de servicios), con el formulario precargado con los datos actuales del servicio.
3. **REQ-003**: El ítem DEBE usar la etiqueta i18n existente `services.edit` (`Editar servicio` en `es`, `Edit service` en `en`), sin texto hardcodeado.
4. **REQ-004**: El orden del resto de las opciones existentes DEBE mantenerse sin cambios: Subir factura, Nueva factura, Renovar, Webhook.
5. **REQ-005**: El comportamiento del menú (abrir/cerrar, cierre al hacer clic en una opción, cierre al hacer clic fuera) DEBE conservarse tal cual lo implementa `CreateMenu`.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-006**: El ícono de la nueva opción DEBE ser `edit`, consistente con el ítem "Editar" del menú de card en `/services`.
2. **REQ-007**: El cambio DEBE compilar (`npm run build` en `frontend/`) y pasar el linter/tests existentes sin errores nuevos.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-008**: Evaluar (sin implementar en esta entrega) cambiar el `title`/`aria-label` hardcodeado `"Crear"` de `CreateMenu` por un texto genérico tipo "Acciones", dado que el menú ahora contiene acciones que no son de creación. Requiere tocar un componente compartido por todas las páginas; se deja documentado como deuda técnica menor.

### 2.4 Fuera de alcance

- No se modifica el comportamiento del formulario de edición: al guardar sigue navegando a `/services` (comportamiento actual de `ServiceFormPage`).
- No se agrega un retorno al detalle del servicio luego de guardar (`?from=detail`). Si se desea, se especifica en una spec futura.
- No se modifica el backend, la DB, ni los endpoints de servicios.
- No se toca el texto hardcodeado `"Subir factura"` (existente, fuera del alcance de este cambio).

### 2.5 Requerimientos No Funcionales

- **Rendimiento**: sin impacto medible; no hay requests adicionales al abrir el menú.
- **Seguridad**: la ruta `/services/edit/:id` ya está protegida por el layout autenticado y el backend valida permisos como en el flujo actual.
- **Almacenamiento**: sin cambios.
- **Disponibilidad**: sin cambios.
- **iHost**: sin nuevas dependencias ni procesos; consumo de RAM/CPU sin variación. El build de React sigue haciéndose fuera del iHost.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Archivos relevantes del frontend:

- `frontend/src/pages/BillsPage.tsx`: página de detalle del servicio. Header con `CreateMenu` (líneas ~129-134) y tabs. Ruta montada en `frontend/src/App.tsx` como `services/bills/:serviceId`.
- `frontend/src/pages/ServicesPage.tsx`: lista de servicios; el menú de card ya incluye `{ label: t('app.edit'), icon: 'edit', onClick: () => navigate('/services/edit/${svc.id}') }`.
- `frontend/src/pages/ServiceFormPage.tsx`: formulario compartido de crear/editar; al guardar navega a `/services`.
- `frontend/src/components/CreateMenu.tsx`: dropdown de 3 puntos del header; recibe `options: { label, icon?, onClick }[]` y renderiza en orden.
- `frontend/src/components/DashboardLayout.tsx`: `BACK_ROUTES` ya incluye `/services/(new|edit/\d+)` → `/services`, por lo que la flecha atrás del header funciona sin cambios.
- `frontend/public/i18n/es.json` y `en.json`: `services.edit` ya existe con los textos "Editar servicio" / "Edit service".

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A: Agregar ítem al `CreateMenu` del header de `BillsPage` apuntando a `/services/edit/:id` | Cambio mínimo (1 archivo), reutiliza formulario, ruta e i18n existentes | Ninguna relevante | ✅ Seleccionada |
| B: Crear una página/modal de edición dentro del detalle | No abandona el contexto del detalle | Duplica formulario y lógica; más código y más superficie de error en un iHost con recursos limitados | ❌ Rechazada |
| C: Agregar un botón "Editar" directo en el header (fuera del menú) | Más visible | Rompe el patrón de UI del proyecto (acciones en el menú de 3 puntos) y el pedido explícito del usuario | ❌ Rechazada |
| D: Agregar el ítem usando el menú de card por factura | — | No aplica: el pedido es editar el servicio, no una factura | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Reutilizar el formulario y la ruta existentes en lugar de duplicar UI.
- **Contexto**: El formulario de edición de servicio ya existe y es el mismo que usa la lista.
- **Decisión**: Navegar a `/services/edit/:id` desde el nuevo ítem del menú.
- **Consecuencias**: Cero duplicación de lógica; el comportamiento posterior al guardar (volver a `/services`) queda igual que hoy. Si en el futuro se quiere volver al detalle, se agrega un parámetro de retorno en una spec aparte.

**ADR-002**: Usar la clave i18n `services.edit` en lugar de `app.edit`.
- **Contexto**: `app.edit` ("Editar") es genérico; en este menú conviven acciones de facturas y del servicio, por lo que un texto ambiguo puede confundir.
- **Decisión**: Etiqueta "Editar servicio" / "Edit service" con la clave ya existente.
- **Consecuencias**: Claridad para el usuario y cero cambios en archivos de traducción.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Usuario] → [/services/bills/:serviceId (BillsPage)]
                 │
                 ├── header: CreateMenu
                 │       ├── Editar servicio  ── navigate() ──► [/services/edit/:id (ServiceFormPage)]
                 │       ├── Subir factura
                 │       ├── Nueva factura
                 │       ├── Renovar
                 │       └── Webhook
                 │
                 └── tabs: analisis | facturas | polizas
```

No hay llamadas nuevas al backend.

### 4.2 Componentes

#### 4.2.1 `BillsPage` (modificado)
- **Responsabilidad**: renderizar el detalle del servicio y sus acciones de header.
- **Cambio**: insertar como primer elemento del array de `options` de `CreateMenu`:
  ```tsx
  { label: t('services.edit'), icon: 'edit', onClick: () => navigate(`/services/edit/${serviceId}`) }
  ```
- **Dependencias**: `t` (i18n), `navigate` (react-router), `serviceId` (params). Todas ya disponibles en el componente.
- **Ubicación**: `frontend/src/pages/BillsPage.tsx`.

#### 4.2.2 `CreateMenu` (sin cambios)
- **Responsabilidad**: dropdown de acciones; renderiza las opciones en el orden recibido.
- **Ubicación**: `frontend/src/components/CreateMenu.tsx`.

#### 4.2.3 `ServiceFormPage` (sin cambios)
- **Responsabilidad**: crear/editar servicios. Ya soporta `:id` en la ruta.
- **Ubicación**: `frontend/src/pages/ServiceFormPage.tsx`.

### 4.3 Modelo de datos

Sin cambios. No se agregan tablas, columnas ni migraciones.

### 4.4 APIs / Contratos

Sin cambios. Se reutilizan los endpoints existentes de servicios (`GET /api/services/:id`, `PUT /api/services/:id`, etc.) consumidos por `ServiceFormPage`.

### 4.5 Dependencias

- **Internas**: `BillsPage`, `CreateMenu`, `ServiceFormPage`, `DashboardLayout` (`BACK_ROUTES`, sin cambios), i18n (`services.edit`, ya existente).
- **Externas**: ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado el detalle `/services/bills/6?tab=analisis`, cuando el usuario abre el menú de 3 puntos del header, entonces la primera opción visible es "Editar servicio".
- [ ] CA-002: Dado el menú abierto en el detalle, cuando el usuario hace clic en "Editar servicio", entonces navega a `/services/edit/6` y el formulario muestra los datos actuales del servicio.
- [ ] CA-003: Dado el formulario abierto desde el detalle, cuando el usuario guarda cambios, entonces se guardan y la app vuelve a `/services` (comportamiento existente), y al reabrir el detalle se ven los datos actualizados.
- [ ] CA-004: Dado el detalle en tabs `facturas` y `polizas`, cuando se abre el menú, entonces la opción "Editar servicio" también aparece primera.
- [ ] CA-005: El orden restante del menú se mantiene: Subir factura, Nueva factura, Renovar, Webhook.
- [ ] CA-006: En idioma inglés la opción se muestra como "Edit service"; no hay textos hardcodeados nuevos.
- [ ] CA-007: El menú mantiene su comportamiento de cierre al seleccionar una opción y al hacer clic fuera.
- [ ] CA-DARK: No aplica (no se crean formularios/modales; el formulario reutilizado ya cumple con los tokens del tema).
- [ ] CA-BACK: No aplica (no se crean páginas nuevas; `/services/edit/:id` ya está registrada en `BACK_ROUTES`).

### 5.2 No funcionales

- [ ] CA-NF-001: `npm run build` en `frontend/` compila sin errores ni warnings nuevos.
- [ ] CA-NF-002: No se agregan dependencias ni cambios de backend/DB (verificable con `git diff --stat`).

### 5.3 Testing

- **Unit tests**: no hay lógica nueva que testear (solo se agrega un ítem de menú).
- **Integration tests**: no aplica.
- **E2E / manual**: abrir `/services/bills/:id` con un servicio sintético, verificar orden del menú, navegación al formulario, edición y guardado; repetir en `es` y `en`; verificar tabs `analisis`, `facturas` y `polizas`.
- **Carga/Performance**: sin impacto esperado; no se requiere medición específica.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Agregar la opción "Editar servicio" como primera del `CreateMenu` en `BillsPage` | 15 min | Ninguna |
| 2 | Verificar i18n (`services.edit`) y `BACK_ROUTES` (sin cambios) | 10 min | Fase 1 |
| 3 | Build del frontend y prueba manual en local (incluye darkmode y ambos idiomas) | 30 min | Fase 1-2 |

### 6.2 Milestones

1. **MVP**: opción visible y navegación funcional al formulario de edición.
2. **V1.0**: MVP + verificación de tabs, idiomas, build y prueba manual aprobada por el usuario.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Ambigüedad entre "Editar servicio" y "Editar factura" (menú de card por factura) | Media | Bajo | Etiqueta explícita "Editar servicio" y ubicación como primera opción del menú del header |
| Regresión en el orden de las opciones existentes | Baja | Medio | Insertar el nuevo ítem al inicio sin reordenar el resto; CA-005 |
| El `title` hardcodeado "Crear" del botón de 3 puntos queda impreciso | Alta | Bajo | Documentado como REQ-008 (P2) para una spec futura; no bloquea esta entrega |
| Diferencia de comportamiento al guardar (vuelve a la lista en vez del detalle) | Media | Bajo | Documentado en "Fuera de alcance"; se puede especificar `?from=detail` en el futuro |

## 8. Notas y Referencias

- Specs relacionadas: SPEC-004 (módulo de servicios), SPEC-005 (React + Tailwind), SPEC-014 (UI de facturas), SPEC-063 (navegación de retorno en páginas de detalle), SPEC-066 (worktrees por sesión).
- Código de referencia: `frontend/src/pages/BillsPage.tsx`, `frontend/src/pages/ServicesPage.tsx`, `frontend/src/components/CreateMenu.tsx`, `frontend/src/components/DashboardLayout.tsx`.
- Captura/URL reportada por el usuario: `/services/bills/6?tab=analisis` (sin datos reales asociados).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-10-21 | opencode | Creación inicial de la especificación a partir del pedido del usuario |
