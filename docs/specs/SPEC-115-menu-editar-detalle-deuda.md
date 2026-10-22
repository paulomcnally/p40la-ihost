---
title: "Menú de acciones Editar en el detalle de deuda y estandarización de la etiqueta Editar"
id: "SPEC-115"
status: "released"
author: "opencode"
created: "2026-10-21"
updated: "2026-10-21"
github_issue: 118
---

# Menú de acciones Editar en el detalle de deuda y estandarización de la etiqueta Editar

**ID**: SPEC-115  
**Estado**: released  
**Autor**: opencode  
**Creado**: 2026-10-21  
**Actualizado**: 2026-10-21

---

## 1. Resumen Ejecutivo

En el detalle de una deuda (`/deudas/:id`, por ejemplo `/deudas/1?status=pending`) el header no tiene ningún menú de acciones: para editar la deuda el usuario debe volver a `/deudas` y abrir el menú de 3 puntos del card. Esto rompe el patrón de UI del resto del sistema, donde los detalles (servicios, autos, automation) ofrecen sus acciones desde el header.

Esta spec agrega el menú de 3 puntos (`CreateMenu`) al header del detalle de deuda con la opción **Editar**, que navega al formulario existente `/deudas/edit/:id`. Además, aprovecha para alinear la etiqueta de la opción "Editar servicio" del detalle de servicio (SPEC-114) al texto estándar **Editar**, usando la clave i18n genérica `app.edit`, y deja documentado el estándar en `docs/project-rules.md` y `AGENTS.md`: **las acciones de edición se rotulan "Editar"**, nunca "Editar <entidad>" ni "Editar <página>".

Impacto en iHost: nulo. No se agregan dependencias, no hay cambios de esquema, de backend ni de memoria en runtime. El cambio es exclusivamente de frontend (un menú reutilizado + dos claves i18n) y de documentación. El bundle no cambia de tamaño de forma significativa.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: En la página de detalle de deuda (`/deudas/:id`, componente `DebtBillsPage`), el header DEBE mostrar el menú de acciones de 3 puntos (`CreateMenu`) a la derecha del título.
2. **REQ-002**: El menú DEBE contener como primera (y única, en esta entrega) opción **Editar**, con ícono `edit`.
3. **REQ-003**: Al seleccionar **Editar**, la app DEBE navegar a `/deudas/edit/:id` (formulario `DebtFormPage`), que ya precarga los datos actuales de la deuda.
4. **REQ-004**: La etiqueta DEBE salir de la clave i18n genérica `app.edit` (`Editar` en `es`, `Edit` en `en`). No se permite texto hardcodeado.
5. **REQ-005**: El menú DEBE estar disponible en todos los filtros del detalle (`?status=pending`, `?status=paid`, `?status=all`) y también en deudas cerradas.
6. **REQ-006**: En el menú del header del detalle de servicio (`BillsPage`, SPEC-114), la opción hoy rotulada "Editar servicio" DEBE pasar a rotularse **Editar** usando también `app.edit`.
7. **REQ-007**: El estándar de rotulación DEBE quedar documentado en `docs/project-rules.md` (nueva subsección de UI) y en `AGENTS.md` (regla de UI/acciones), indicando que la acción de edición se llama **Editar**, no "Editar <entidad>" ni "Editar <página>".

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-008**: `CreateMenu` DEBE aceptar un `title`/`aria-label` opcional para no mostrar el tooltip hardcodeado `"Crear"` en menús que solo contienen acciones (caso del detalle de deuda). El valor por defecto DEBE seguir siendo `"Crear"` para no alterar las páginas existentes.
2. **REQ-009**: El detalle de deuda DEBE pasar al menú una etiqueta genérica **Acciones** (nueva clave i18n `app.actions`: `Acciones`/`Actions`).
3. **REQ-010**: El cambio DEBE compilar (`npm run build` en `frontend/`) sin errores nuevos y el i18n servido por el backend DEBE incluir las claves modificadas.
4. **REQ-011**: El comportamiento del menú (abrir/cerrar, cierre al hacer clic fuera y al elegir una opción) DEBE conservarse tal cual lo implementa `CreateMenu`.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-012**: Auditar el resto de los menús de acciones para detectar etiquetas "Editar <entidad>" remanentes y alinearlas al estándar en una entrega posterior. En esta spec solo se corrigen los dos casos reportados (detalle de deuda y detalle de servicio).

### 2.4 Fuera de alcance

- No se agrega la acción **Eliminar** al detalle de deuda (sigue disponible solo desde el card de `/deudas`).
- No se cambia el comportamiento del formulario `DebtFormPage`: al guardar sigue navegando a `/deudas`.
- No se agrega retorno al detalle de la deuda después de guardar (`?from=detail`). Si se desea, se especifica en una spec futura.
- No se modifica el backend, la base de datos ni los endpoints de deudas.
- No se cambia el valor de `services.edit` (`Editar servicio` / `Edit service`), que se mantiene como **título de página** del formulario `ServiceFormPage`; el estándar aplica a etiquetas de acciones, no a títulos descriptivos de página.
- No se modifican otras etiquetas "Editar <entidad>" usadas como título de formulario (`deudas.edit`, `homes.edit`, `autos.edit`, etc.).

### 2.5 Requerimientos No Funcionales

- **Rendimiento**: sin impacto medible; no hay requests adicionales al abrir el menú.
- **Seguridad**: `/deudas/edit/:id` ya está protegida por el layout autenticado y el backend valida permisos como en el flujo actual desde el card.
- **Almacenamiento**: sin cambios.
- **Disponibilidad**: sin cambios.
- **iHost**: sin nuevas dependencias ni procesos; solo se reutiliza un componente React existente.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- El detalle de deuda está implementado en `frontend/src/pages/DebtBillsPage.tsx` (ruta `deudas/:id` en `frontend/src/App.tsx`). Su header solo renderiza el título y el subtítulo; no hay `CreateMenu` ni acciones.
- La edición de deudas ya existe: `DeudasPage.tsx` la expone desde el menú del card con `t('app.edit')` e ícono `edit`, navegando a `/deudas/edit/${debt.id}`. El formulario `DebtFormPage` precarga los datos y al guardar navega a `/deudas`.
- El detalle de servicio (`BillsPage.tsx`) ya usa `CreateMenu`; su opción de edición (SPEC-114) usa `t('services.edit')`, cuyo valor es "Editar servicio" / "Edit service".
- El patrón del resto del sistema es: **menús de acciones → `app.edit` ("Editar")**; títulos de formularios → clave específica ("Editar deuda", "Editar servicio", etc.).
- `frontend/public/i18n/{es,en}.json` es la fuente de verdad del i18n (el build de Vite regenera `public/i18n/`).
- `docs/project-rules.md` §4 concentra las reglas de UI; `AGENTS.md` concentra las reglas operativas para agentes.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A: Agregar `CardMenu` al header del detalle | Ya existe y se usa en cards | Está posicionado de forma absoluta (`absolute top-3 right-3`), pensado para cards; el header necesitaría contenedor relativo y estilos extra | ❌ Rechazada |
| B: Reutilizar `CreateMenu` en el header del detalle | Es el componente del header usado en detalle de servicio, automation y listados; dropdown ya resuelve apertura/cierre | Su tooltip está hardcodeado como "Crear" | ✅ Seleccionada (con prop opcional `title`, REQ-008) |
| C: Botón directo "Editar" en el header | Menos clics | Rompe el patrón de 3 puntos del repo (`docs/project-rules.md` §4.1) | ❌ Rechazada |
| D: Cambiar el valor de `services.edit` a "Editar" | Cambio de una línea | Degrada el título del formulario de edición ("Editar" sin contexto) y rompe la simetría con `deudas.edit`, `homes.edit`, etc. | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Reutilizar `CreateMenu` en el header del detalle de deuda.
- **Contexto**: El detalle de deuda no tiene acciones y el repo exige el patrón de 3 puntos en headers con acciones.
- **Decisión**: Usar `CreateMenu` con una opción `{ label: t('app.edit'), icon: 'edit', onClick: () => navigate('/deudas/edit/:id') }`.
- **Consecuencias**: Consistencia con el resto de los detalles; cero componentes nuevos; el menú queda preparado para futuras acciones (eliminar, pagar cuota, etc.).

**ADR-002**: Separar "etiqueta de acción" de "título de página".
- **Contexto**: `services.edit` se usa tanto en el menú del detalle como en el `<h2>` del formulario. Cambiar su valor afecta ambos usos.
- **Decisión**: Las acciones de menú usan siempre `app.edit` ("Editar"). `services.edit` se conserva para el título de página "Editar servicio".
- **Consecuencias**: El menú cumple el estándar sin degradar el título del formulario; el estándar queda documentado para futuras páginas.

**ADR-003**: Prop opcional `title` en `CreateMenu`.
- **Contexto**: El componente hardcodea `title="Crear"`, incorrecto para un menú que solo tiene "Editar".
- **Decisión**: Agregar `title?: string` con default `'Crear'`; el detalle de deuda pasa `t('app.actions')`.
- **Consecuencias**: Mejora de accesibilidad sin romper ninguna página existente.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[DebtBillsPage /deudas/:id]
        │  header: título + CreateMenu (Editar)
        │
        └── navigate('/deudas/edit/:id')
                    │
                    ▼
          [DebtFormPage (existente)]
                    │  GET /api/debts/:id
                    ▼
               [SQLite]

[BillsPage /services/bills/:serviceId]
        │  header: CreateMenu
        └── opción "Editar" (app.edit) ──► /services/edit/:id
```

### 4.2 Componentes

#### 4.2.1 `CreateMenu` (`frontend/src/components/CreateMenu.tsx`)
- **Responsabilidad**: Renderizar el botón de 3 puntos y el dropdown de acciones del header.
- **Interfaz**: `CreateMenuOption { label, icon?, onClick }`; nueva prop opcional `title?: string` (default `'Crear'`).
- **Dependencias**: `Icon`.
- **Ubicación**: `frontend/src/components/CreateMenu.tsx`.

#### 4.2.2 `DebtBillsPage` (`frontend/src/pages/DebtBillsPage.tsx`)
- **Responsabilidad**: Mostrar el detalle de deuda y sus cuotas.
- **Cambios**: importar `useNavigate` y `CreateMenu`; agregar el menú al header con la opción Editar.
- **Dependencias**: `CreateMenu`, i18n (`app.edit`, `app.actions`).
- **Ubicación**: `frontend/src/pages/DebtBillsPage.tsx`.

#### 4.2.3 `BillsPage` (`frontend/src/pages/BillsPage.tsx`)
- **Responsabilidad**: Detalle del servicio y sus facturas.
- **Cambios**: la opción de edición del menú pasa de `t('services.edit')` a `t('app.edit')`.
- **Ubicación**: `frontend/src/pages/BillsPage.tsx`.

#### 4.2.4 i18n (`frontend/public/i18n/es.json`, `frontend/public/i18n/en.json`)
- **Cambios**: agregar `app.actions` (`Acciones` / `Actions`). `app.edit` ya existe (`Editar` / `Edit`).

#### 4.2.5 Documentación de estándar
- `docs/project-rules.md`: nueva subsección §4.5 "Etiqueta de acciones de edición".
- `AGENTS.md`: bullet en las reglas de UI/patrones indicando que la acción se rotula **Editar**.

### 4.3 Modelo de datos

Sin cambios. No se agregan tablas, columnas, índices ni migraciones.

### 4.4 APIs / Contratos

Sin cambios. Se reutiliza `GET /api/debts/:id` que ya consume `DebtFormPage`.

### 4.5 Dependencias

- **Internas**: `CreateMenu`, `DebtBillsPage`, `BillsPage`, i18n.
- **Externas**: ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Dado `/deudas/1?status=pending`, cuando carga la página, entonces el header muestra el botón de 3 puntos a la derecha del título.
- [x] CA-002: Dado el menú abierto, cuando el usuario hace clic en **Editar**, entonces navega a `/deudas/edit/1` y el formulario muestra los datos actuales de la deuda.
- [x] CA-003: Dado `/deudas/1?status=paid` o `?status=all` (y también una deuda cerrada), cuando se abre el menú, entonces la opción Editar está disponible.
- [x] CA-004: Dado el detalle de servicio `/services/bills/:id`, cuando se abre el menú, entonces la opción de edición se rotula **Editar** (es) / **Edit** (en), no "Editar servicio".
- [x] CA-005: `docs/project-rules.md` y `AGENTS.md` documentan que la acción de edición se rotula **Editar** (no "Editar <entidad>" ni "Editar <página>").
- [x] CA-006: El tooltip/aria-label del menú del detalle de deuda es **Acciones**, no "Crear"; las demás páginas conservan "Crear".
- [x] CA-007: `frontend/public/i18n/{es,en}.json` contiene `app.actions` y, tras `npm run build`, `curl -s http://localhost:8088/i18n/es.json` sirve la clave.
- [x] CA-DARK: El menú existente usa tokens del tema (`bg-card`, `border-border`, `text-text`); se verifica legibilidad en darkmode.
- [x] CA-BACK: No se crean páginas nuevas; el retorno del formulario de deuda sigue con la flecha atrás del header registrada en `BACK_ROUTES`.

### 5.2 No funcionales

- [x] CA-NF-001: `npm run build` en `frontend/` finaliza sin errores nuevos.
- [x] CA-NF-002: Sin cambios de backend, DB ni dependencias; el binario Go no cambia.
- [x] CA-NF-003: El menú abre/cierra con clic fuera y se cierra al elegir una opción, igual que antes.

### 5.3 Testing

- **Unit tests**: no aplica (cambio de UI sin lógica nueva).
- **Integration tests**: no aplica (no hay endpoints nuevos).
- **E2E/manual**: abrir `/deudas/:id` en los tres filtros, editar y guardar; abrir `/services/bills/:id` y verificar la etiqueta del menú; verificar darkmode.
- **Carga/Performance**: sin métricas nuevas; el menú es un dropdown ya existente.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Agregar prop opcional `title` a `CreateMenu` y claves `app.actions` | 10 min | Ninguna |
| 2 | Agregar `CreateMenu` con Editar al header de `DebtBillsPage` | 10 min | Fase 1 |
| 3 | Cambiar la etiqueta de edición de `BillsPage` a `app.edit` | 5 min | Ninguna |
| 4 | Documentar el estándar en `project-rules.md` y `AGENTS.md` | 10 min | Ninguna |
| 5 | Build, verificación de i18n servido y prueba manual en local | 15 min | Fases 1-4 |

### 6.2 Milestones

1. **MVP**: menú Editar funcional en el detalle de deuda y etiqueta "Editar" en el detalle de servicio.
2. **V1.0**: MVP + estándar documentado + verificación en darkmode y filtros.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| El menú del header se superpone con el título en móvil | Baja | Medio | El header ya usa `flex items-center justify-between`; el título tiene `min-w-0` y el menú es de 36px |
| Cambiar `CreateMenu` afecta otras páginas | Baja | Alto | La nueva prop es opcional con default `'Crear'`; no se toca la firma existente |
| Editar traducciones en `public/i18n/` y perderlas en el build | Media | Medio | Editar solo `frontend/public/i18n/` y verificar con `curl` tras `npm run build` |
| Confusión entre `services.edit` (título) y `app.edit` (acción) | Media | Bajo | ADR-002 y estándar documentado; menús usan siempre `app.edit` |
| El usuario espera que "Editar servicio" desaparezca también del título del formulario | Media | Bajo | Fuera de alcance documentado; el estándar aplica a acciones, no a títulos de página |

## 8. Notas y Referencias

- SPEC-063: navegación de retorno con flecha atrás y `BACK_ROUTES`.
- SPEC-114: opción Editar servicio en el menú de acciones del detalle de servicio.
- `docs/project-rules.md` §4: reglas de UI (CreateMenu, tokens de tema, darkmode, retorno).
- `AGENTS.md`: reglas de UI y flujo de specs.
- `frontend/src/components/CreateMenu.tsx`, `frontend/src/pages/DebtBillsPage.tsx`, `frontend/src/pages/BillsPage.tsx`.

**Validación del usuario**: 2026-10-21, el usuario confirmó que las pruebas manuales fueron satisfactorias (menú Editar en el detalle de deuda y etiqueta Editar en el detalle de servicio).

**Release**: mergeado a `main` en el commit `7f198cd` (2026-10-21), sobre la versión 0.5.0. Build de frontend incluido en el merge (sin cambios de backend ni DB).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-10-21 | opencode | Creación inicial de la especificación |
| 2026-10-21 | opencode | Estado a `in_progress`. Implementado: prop `title` en `CreateMenu`, menú Editar en el header de `DebtBillsPage`, etiqueta `app.edit` en el menú de `BillsPage`, clave `app.actions` (es/en) y estándar documentado en `docs/project-rules.md` §4.5 y `AGENTS.md`. Build de frontend OK. |
| 2026-10-21 | opencode | Validación local en puerto 8097: `/health` OK, `/i18n/es.json` sirve `app.actions`, bundle con `app.edit` y `/deudas/edit/`, `GET /api/debts/1` OK. Pendiente prueba manual del usuario (8099 está ocupado por un server del checkout principal). |
| 2026-10-21 | opencode | Confirmación del usuario y release: merge a `main` (`7f198cd`), tracker actualizado, label `spec/released` e issue cerrado. |
