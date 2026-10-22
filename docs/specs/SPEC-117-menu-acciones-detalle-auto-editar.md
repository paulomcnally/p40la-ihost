---
title: "Menú de acciones (3 puntos) en el detalle del auto con opción Editar"
id: "SPEC-117"
status: "in_progress"
author: "opencode"
created: "2026-10-21"
updated: "2026-10-21"
github_issue: 120
---

# Menú de acciones (3 puntos) en el detalle del auto con opción Editar

**ID**: SPEC-117  
**Estado**: in_progress  
**Autor**: opencode  
**Creado**: 2026-10-21  
**Actualizado**: 2026-10-21

---

## 1. Resumen Ejecutivo

Hoy la página de detalle de un auto (`/autos/:id`) muestra la información del vehículo y el listado de seguros asociados, pero no ofrece ninguna acción sobre el auto. Para editar marca, modelo, año, color, placa, motor, chasis, VIN o ícono, el usuario debe volver a la lista `/autos`, abrir el menú de 3 puntos del card y elegir "Editar". El pedido del usuario es poder editar el auto directamente desde su página de detalle.

Esta spec agrega el menú de acciones de 3 puntos (`CardMenu`) en la card resumen del detalle del auto, con la opción **"Editar auto"**, reutilizando el formulario y la ruta existentes `/autos/edit/:id`. El alcance aprobado por el usuario es **solo Editar**; no se agrega Eliminar ni ninguna otra acción en esta entrega.

El cambio es exclusivamente de frontend: no toca backend, base de datos, APIs ni endpoints. Impacto en iHost: nulo. No se agregan dependencias, no hay cambios de esquema ni de memoria en runtime; solo se reutiliza un componente ya renderizado en la lista de autos (`CardMenu`) y una ruta ya existente. El build de React sigue realizándose fuera del iHost.

Consideraciones de UI:
- No se crean formularios nuevos; el formulario reutilizado (`AutoFormPage`) ya usa los tokens de color del tema. Aun así se verificará el menú y el formulario en darkmode.
- No se crean páginas de detalle nuevas. Tanto `/autos/:id` como `/autos/edit/:id` ya están registradas en `BACK_ROUTES` de `frontend/src/components/DashboardLayout.tsx`, por lo que la flecha atrás del header sigue funcionando sin cambios.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: En la página de detalle del auto (`/autos/:id`), la card resumen del vehículo DEBE mostrar el botón de acciones de 3 puntos (`CardMenu`) en la esquina superior derecha, visible en móvil y desktop.
2. **REQ-002**: Al abrir el menú, DEBE mostrar la opción **"Editar auto"** (en inglés, "Edit car"), utilizando la clave i18n existente `autos.edit`; no se admite texto hardcodeado nuevo.
3. **REQ-003**: Al seleccionar **"Editar auto"**, la app DEBE navegar a `/autos/edit/:id` usando el `id` del auto actual, con el formulario precargado con los datos existentes.
4. **REQ-004**: Al guardar los cambios, se DEBE persistir la edición mediante el flujo existente de `AutoFormPage`. El comportamiento posterior al guardado se mantiene igual que hoy: navegar a `/autos`.
5. **REQ-005**: El menú del detalle DEBE contener únicamente la opción "Editar auto" en esta entrega. No se agrega "Eliminar" ni acciones sobre seguros.
6. **REQ-006**: El menú de 3 puntos de cada seguro (`Editar` / `Eliminar` seguro) DEBE seguir funcionando sin cambios.
7. **REQ-007**: El componente `CardMenu` DEBE conservar su comportamiento actual: se abre al hacer clic, se cierra al seleccionar una opción y al hacer clic fuera.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-008**: El ícono de la nueva opción DEBE ser `edit`, consistente con el ítem "Editar" del menú de la lista de autos.
2. **REQ-009**: El botón de 3 puntos y su dropdown NO DEBEN superponerse con el ícono del auto, el título (`marca modelo`), el año/color/placa ni los datos de motor/chasis/VIN, incluso en viewports de 320 px de ancho.
3. **REQ-010**: El menú DEBE verse correctamente en darkmode (fondo `bg-card`, borde `border-border`, texto con tokens del tema) y en light mode.
4. **REQ-011**: El cambio DEBE compilar con `npm run build` en `frontend/` sin errores ni warnings nuevos, y no debe romper el linter/tests existentes.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-012**: Evaluar (sin implementar en esta entrega) que al guardar una edición iniciada desde el detalle se regrese al detalle (`/autos/:id`) en lugar de la lista, mediante un parámetro `?from=detail`. Requiere cambiar `AutoFormPage` y se documenta como deuda técnica menor.
2. **REQ-013**: Evaluar (sin implementar en esta entrega) la migración de los textos hardcodeados existentes de `AutoShowPage` ("Seguros", "Agregar Seguro", "No hay seguros asociados", "Motor", "Chasis", "VIN") a i18n, fuera del alcance de este pedido.

### 2.4 Fuera de alcance

- No se agrega la opción "Eliminar auto" en el detalle (el usuario eligió solo Editar).
- No se modifica el backend, la DB ni los endpoints de autos.
- No se modifica el modelo de datos ni se agregan migraciones.
- No se modifica el menú de la lista `/autos` ni el formulario `AutoFormPage`.
- No se agregan modales nuevos ni páginas nuevas.
- No se cambia el comportamiento de guardado de `AutoFormPage` (sigue navegando a `/autos`).

### 2.5 Requerimientos No Funcionales

- **Rendimiento**: sin impacto medible; no hay requests adicionales al abrir el menú. El dropdown se renderiza con `createPortal` bajo demanda.
- **Seguridad**: la ruta `/autos/edit/:id` ya está protegida por el layout autenticado y el backend aplica las mismas validaciones que en el flujo actual.
- **Almacenamiento**: sin cambios; no se agregan tablas, columnas ni archivos de datos.
- **Disponibilidad**: sin cambios; no se agregan procesos ni health checks.
- **iHost**: sin nuevas dependencias ni consumo adicional de RAM/CPU. El bundle aumenta solo por el ítem de menú (despreciable). El build de React continúa fuera del iHost.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Archivos relevantes del frontend:

- `frontend/src/pages/AutoShowPage.tsx`: página de detalle `/autos/:id`. Renderiza una card resumen del auto (sin menú de acciones) y una card de seguros. Ya importa `CardMenu` para el menú de cada seguro.
- `frontend/src/pages/AutosPage.tsx`: lista de autos. El menú de card ya incluye `{ label: 'Editar', icon: 'edit', onClick: () => navigate('/autos/edit/${auto.id}') }` y `{ label: 'Eliminar', ... }`.
- `frontend/src/pages/AutoFormPage.tsx`: formulario compartido de crear/editar; detecta `isEdit` por el parámetro `id` y al guardar navega a `/autos`.
- `frontend/src/components/CardMenu.tsx`: menú de 3 puntos con `options: { label, icon?, danger?, onClick }[]`. Su raíz es `absolute top-3 right-3 z-50` y el dropdown se renderiza en un portal con `z-index: 9999`. Cierra al hacer clic fuera y al seleccionar una opción.
- `frontend/src/components/DashboardLayout.tsx`: `BACK_ROUTES` ya incluye `/autos/:id` → `/autos` y `/autos/(new|edit/:id)` → `/autos`.
- `frontend/src/App.tsx`: rutas `autos`, `autos/new`, `autos/edit/:id`, `autos/:id`.
- `frontend/public/i18n/es.json` y `en.json`: la clave `autos.edit` ya existe con los textos "Editar auto" / "Edit car".
- `frontend/src/components/Icons.tsx`: el ícono `edit` y el ícono `more` ya existen y se usan en la lista de autos.

Observación de layout: la card resumen de `AutoShowPage` no tiene `relative`, y su contenido ocupa todo el ancho. Para que `CardMenu` (que es `absolute top-3 right-3`) se posicione respecto de la card y no tape el título, la card debe volverse `relative` y el contenido debe reservar espacio a la derecha (padding derecho) en móvil y desktop.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A: Agregar `CardMenu` a la card resumen de `AutoShowPage` con la opción "Editar auto" apuntando a `/autos/edit/:id` | Cambio mínimo (1 archivo), reutiliza componente, formulario, ruta e i18n existentes; respeta el patrón de UI de 3 puntos | Requiere ajustar `relative`/padding para evitar solapamiento en móvil | ✅ Seleccionada |
| B: Agregar la acción al menú global del header (`CreateMenu` de `DashboardLayout`) | No toca la card | El header es compartido por todas las páginas y las acciones dependen del auto cargado; acopla el layout global a una entidad y complica el estado | ❌ Rechazada |
| C: Botón "Editar" directo en la card (fuera del menú) | Más visible | Rompe el patrón de acciones en menú de 3 puntos y el pedido explícito del usuario | ❌ Rechazada |
| D: Modal de edición dentro del detalle | No abandona el contexto | Duplica el formulario y su lógica; más código, más superficie de error y más peso para un iHost con recursos limitados | ❌ Rechazada |
| E: Reutilizar el menú de la card de seguros | — | No aplica: ese menú edita/elimina un seguro, no el auto | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Reutilizar el formulario y la ruta existentes en lugar de duplicar UI.
- **Contexto**: `AutoFormPage` ya soporta creación y edición mediante `/autos/edit/:id`, y es el mismo formulario que usa la lista.
- **Decisión**: El nuevo ítem navega a `/autos/edit/:id`.
- **Consecuencias**: Cero duplicación de lógica. El guardado mantiene el comportamiento actual (volver a `/autos`); si se desea volver al detalle, se especifica en una spec futura (REQ-012).

**ADR-002**: Usar `CardMenu` en la card resumen, no `CreateMenu` en el header global.
- **Contexto**: `CardMenu` es el patrón del proyecto para acciones sobre una entidad en una card. `CreateMenu` está pensado para acciones de creación del header de una página.
- **Decisión**: Insertar un `CardMenu` dentro de la card resumen del auto.
- **Consecuencias**: Consistencia visual con `/autos` y con las cards de seguros. Se debe marcar la card como `relative` y reservar espacio a la derecha para evitar solapamientos.

**ADR-003**: Etiqueta i18n `autos.edit`.
- **Contexto**: La clave ya existe en ambos idiomas y describe exactamente la acción.
- **Decisión**: Usar `t('autos.edit')` en lugar de un literal.
- **Consecuencias**: Cero cambios en archivos de traducción y texto correcto en ES/EN.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Usuario] → [/autos/:id (AutoShowPage)]
                 │
                 ├── card resumen (relative)
                 │       └── CardMenu (3 puntos)
                 │               └── "Editar auto" ── navigate() ──► [/autos/edit/:id (AutoFormPage)]
                 │
                 └── card de seguros
                         └── CardMenu por seguro (sin cambios: Editar / Eliminar)

[AutoFormPage] ── PUT /api/autos/:id (existente) ──► [SQLite]
      └── al guardar: navigate('/autos')
```

No hay llamadas nuevas al backend.

### 4.2 Componentes

#### 4.2.1 `AutoShowPage` (modificado)
- **Responsabilidad**: renderizar el detalle del auto y sus acciones.
- **Cambio**:
  1. Importar `useNavigate` de `react-router-dom` y `useI18nStore`.
  2. Obtener `navigate` y `t`.
  3. Marcar la card resumen como `relative` y reservar espacio a la derecha del contenido (por ejemplo `pr-10 sm:pr-12`) para que el botón de 3 puntos no tape el título.
  4. Insertar al inicio de la card:
     ```tsx
     <CardMenu
       options={[
         { label: t('autos.edit'), icon: 'edit', onClick: () => navigate(`/autos/edit/${auto.id}`) },
       ]}
     />
     ```
- **Dependencias**: `t` (i18n), `navigate` (react-router), `auto.id` (estado ya cargado). Todas disponibles en el componente.
- **Ubicación**: `frontend/src/pages/AutoShowPage.tsx`.

#### 4.2.2 `CardMenu` (sin cambios)
- **Responsabilidad**: dropdown de acciones de 3 puntos; renderiza las opciones en orden y cierra al seleccionar o hacer clic fuera.
- **Ubicación**: `frontend/src/components/CardMenu.tsx`.

#### 4.2.3 `AutoFormPage` (sin cambios)
- **Responsabilidad**: crear/editar autos. Ya soporta `:id` en la ruta y precarga los datos.
- **Ubicación**: `frontend/src/pages/AutoFormPage.tsx`.

#### 4.2.4 `DashboardLayout` (sin cambios)
- **Responsabilidad**: flecha atrás en páginas de detalle.
- **Estado**: `/autos/:id` y `/autos/edit/:id` ya están en `BACK_ROUTES`; no requiere cambios.
- **Ubicación**: `frontend/src/components/DashboardLayout.tsx`.

### 4.3 Modelo de datos

Sin cambios. No se agregan tablas, columnas ni migraciones. Se reutiliza la entidad `Auto` existente.

### 4.4 APIs / Contratos

Sin cambios. Se reutilizan los endpoints existentes consumidos por `AutoShowPage` y `AutoFormPage`:

- `GET /api/autos/:id` (carga del detalle).
- `GET /api/autos/:id/services` (seguros, sin cambios).
- `PUT /api/autos/:id` (guardado de la edición, ya usado por `AutoFormPage`).

### 4.5 Dependencias

- **Internas**: `AutoShowPage`, `CardMenu`, `AutoFormPage`, `DashboardLayout`, i18n (`autos.edit`, ya existente), `Icons` (`edit`, `more`, ya existentes).
- **Externas**: ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado el detalle `/autos/:id` con un auto existente, cuando carga la página, entonces se ve el botón de 3 puntos en la esquina superior derecha de la card resumen.
- [ ] CA-002: Dado el botón de 3 puntos, cuando el usuario lo pulsa, entonces se abre un dropdown con la opción "Editar auto" (ES) / "Edit car" (EN).
- [ ] CA-003: Dado el menú abierto, cuando el usuario hace clic en "Editar auto", entonces navega a `/autos/edit/:id` y el formulario muestra los datos actuales del auto.
- [ ] CA-004: Dado el formulario abierto desde el detalle, cuando el usuario modifica un campo y guarda, entonces el cambio se persiste y la app vuelve a `/autos` (comportamiento existente); al reabrir el detalle se ven los datos actualizados.
- [ ] CA-005: Dado el detalle, cuando el usuario abre el menú de un seguro, entonces las opciones "Editar" y "Eliminar" del seguro siguen funcionando igual que antes.
- [ ] CA-006: Dado el menú abierto, cuando el usuario hace clic fuera o selecciona una opción, entonces el menú se cierra.
- [ ] CA-007: En viewports de 320 px, 375 px, 768 px y desktop, el botón de 3 puntos y su dropdown no se superponen con el ícono del auto, el título ni los datos del vehículo.
- [ ] CA-008: El menú no contiene opciones distintas de "Editar auto" en esta entrega.
- [ ] CA-DARK: El menú y el formulario reutilizado se verificaron en darkmode: fondo, borde y texto legibles; los inputs usan los tokens del tema (`bg-card`, `text-text`, `text-text-secondary`).
- [ ] CA-BACK: La página de detalle mantiene la flecha atrás del header (`/autos/:id` → `/autos`) y no se agregaron links "← Título" dentro del contenido. `/autos/edit/:id` ya está registrada en `BACK_ROUTES`.

### 5.2 No funcionales

- [ ] CA-NF-001: `npm run build` en `frontend/` compila sin errores ni warnings nuevos.
- [ ] CA-NF-002: No se agregan dependencias ni cambios de backend/DB (verificable con `git diff --stat`).
- [ ] CA-NF-003: No hay requests adicionales al abrir el menú (solo navegación al formulario).

### 5.3 Testing

- **Unit tests**: no hay lógica nueva que testear (solo se agrega un ítem de menú y navegación).
- **Integration tests**: no aplica.
- **E2E / manual**: abrir `/autos/:id` con un auto sintético (por ejemplo "Demo Brand Demo Model", placa `DEMO-001`), verificar el botón, el dropdown, la navegación a `/autos/edit/:id`, la precarga, la edición y el guardado; repetir en `es` y `en`, en light mode y darkmode, y en 320/375/768/desktop.
- **Carga/Performance**: sin impacto esperado; no se requiere medición específica.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Agregar `CardMenu` a la card resumen de `AutoShowPage` con la opción "Editar auto" (`autos.edit`) y navegación a `/autos/edit/:id` | 20 min | Ninguna |
| 2 | Ajustar layout responsive (`relative` + padding derecho) para evitar solapamiento en móvil | 15 min | Fase 1 |
| 3 | Build del frontend y prueba manual en local (ES/EN, light/dark, 320/375/768/desktop) | 30 min | Fase 1-2 |

### 6.2 Milestones

1. **MVP**: botón de 3 puntos visible en el detalle con "Editar auto" y navegación funcional al formulario.
2. **V1.0**: MVP + layout responsive verificado, darkmode, idiomas, build y prueba manual aprobada por el usuario.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| El botón de 3 puntos se superpone con el título o los datos en móvil | Media | Medio | Marcar la card como `relative` y reservar padding derecho; CA-007 con 320/375 px |
| Confusión entre el menú del auto y el menú de cada seguro | Media | Bajo | Ubicar el menú del auto en la card resumen (arriba) y mantener los menús de seguros en cada card; etiqueta "Editar auto" |
| Regresión en el menú de seguros al tocar `AutoShowPage` | Baja | Medio | No modificar los `CardMenu` de seguros; CA-005 |
| El usuario espera volver al detalle después de guardar y vuelve a la lista | Media | Bajo | Documentado en REQ-012 como deuda técnica; comportamiento actual consistente con la lista |
| El dropdown queda detrás de otro elemento | Baja | Bajo | `CardMenu` ya usa portal con `z-index: 9999`; no se modifica |

## 8. Notas y Referencias

- Specs relacionadas: SPEC-024 (CRUD de Autos), SPEC-025 (detalle de autos y seguros), SPEC-038 (limpieza de worktrees), SPEC-063 (flecha atrás en páginas de detalle), SPEC-066 (worktrees por sesión), SPEC-067 (menú de 3 puntos para seguros de auto), SPEC-114 (Editar servicio desde el detalle), SPEC-115 (Editar deuda desde el detalle).
- Código de referencia: `frontend/src/pages/AutoShowPage.tsx`, `frontend/src/pages/AutosPage.tsx`, `frontend/src/pages/AutoFormPage.tsx`, `frontend/src/components/CardMenu.tsx`, `frontend/src/components/DashboardLayout.tsx`.
- URL reportada por el usuario: `/autos/:id` (se usa un identificador genérico; sin datos reales asociados).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-10-21 | opencode | Creación inicial de la especificación a partir del pedido del usuario. Alcance confirmado: solo opción "Editar auto" en el menú de 3 puntos del detalle. |
| 2026-10-21 | opencode | Estado a `in_progress`: inicio de implementación en el worktree `p40la-ihost-spec-117` (rama `feature/SPEC-117`). |
| 2026-10-21 | opencode | Implementado en `AutoShowPage`: `CardMenu` con "Editar auto" (`autos.edit`) en la card resumen, `relative` + `pr-8` para evitar solapamiento. Build de frontend OK. E2E con Playwright: 14/14 checks (ES/EN, dark/light, 320 px, navegación, precarga, guardado y persistencia). Pendiente validación manual del usuario. |
