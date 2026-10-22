---
title: "Instituciones: página de vista y formulario con tabs Información y Analizadores"
id: "SPEC-116"
status: "in_progress"
author: "opencode"
created: "2026-10-21"
updated: "2026-10-21"
github_issue: 119
---

# Instituciones: página de vista y formulario con tabs Información y Analizadores

**ID**: SPEC-116  
**Estado**: in_progress  
**Autor**: opencode  
**Creado**: 2026-10-21  
**Actualizado**: 2026-10-21

---

## 1. Resumen Ejecutivo

Hoy la lista de instituciones (`/institutions`) navega directamente al formulario de edición al hacer clic en un card, y el formulario mezcla en una sola pantalla los datos generales (nombre y categoría) con la gestión de analizadores. Esto obliga a entrar en modo edición solo para consultar una institución y hace que el formulario crezca a medida que se asignan analizadores.

Esta spec incorpora una **página de vista** de la institución (`/institutions/:id`) con un menú de acciones de 3 puntos que incluye **Editar**, siguiendo el patrón ya usado en autos (`/autos/:id`) y automatizaciones (`/automation/:id`). Además, reorganiza el formulario de crear/editar en **dos tabs**: **Información** y **Analizadores**, con el mismo patrón visual y de URL (`?tab=...`) que `AutomationFormPage`.

El resultado esperado: consultar una institución deja de requerir abrir el formulario, editar queda a un clic desde la vista, y el formulario se mantiene compacto y navegable. El cambio es exclusivamente de frontend: no modifica endpoints, esquema SQLite ni lógica de negocio.

**Consideraciones iHost**: sin nuevas dependencias ni procesos. No hay cambios de backend, DB ni consumo de memoria en runtime; solo se agregan una página React y estado de tab en el formulario existente.

**Consideraciones de UI obligatorias**:
- Todo `input`/`select`/`textarea` debe usar los tokens del tema (`bg-card`, `text-text`, `text-text-secondary`). Prohibido `bg-white`, `dark:bg-[#2c2c2e]` o `text-black`.
- El formulario con tabs debe verificarse en darkmode (texto y placeholders legibles).
- La página de detalle debe registrarse en `BACK_ROUTES` de `frontend/src/components/DashboardLayout.tsx` para que la flecha atrás del header reemplace la hamburguesa en móvil. Prohibido agregar links "← Instituciones" dentro del contenido (SPEC-063).

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Crear la página de vista de institución en la ruta `/institutions/:id` (`InstitutionShowPage`), protegida por el layout autenticado existente.
2. **REQ-002**: En `/institutions`, al hacer clic en el card de una institución se debe navegar a `/institutions/:id` (vista), no a `/institutions/edit/:id`.
3. **REQ-003**: El menú de 3 puntos del card de la lista (`CardMenu`) debe conservar la opción **Editar** (navega a `/institutions/edit/:id`) y la opción **Eliminar** (con `DeleteModal`, comportamiento actual).
4. **REQ-004**: La página de vista debe mostrar, como mínimo: nombre de la institución, categoría (nombre e ícono cuando exista), cantidad de analizadores asignados y el listado de analizadores asignados con su nombre legible.
5. **REQ-005**: La página de vista debe tener un menú de acciones de 3 puntos (`CardMenu`) con la opción **Editar** como primera opción, que navegue a `/institutions/edit/:id`.
6. **REQ-006**: Si la institución no existe (ID inválido o eliminado), la página de vista debe mostrar un estado de error amigable con acción para volver a `/institutions`, sin romper la app.
7. **REQ-007**: El formulario de crear/editar (`/institutions/new` y `/institutions/edit/:id`) debe mostrar dos tabs: **Información** y **Analizadores**, con el mismo patrón visual que `AutomationFormPage` (contenedor `bg-bg`, tab activa `bg-card text-primary shadow-ios`).
8. **REQ-008**: La tab **Información** debe contener el campo **Nombre** (obligatorio) y el selector **Categoría** (incluida la opción "Sin categoría").
9. **REQ-009**: La tab **Analizadores** debe contener la lista de analizadores asignados (con acción de remover) y el botón **Agregar analizador** que abre `AnalyzerPickerModal`.
10. **REQ-010**: La tab activa debe persistir en la URL como `?tab=info` o `?tab=analyzers`, con **Información** como valor por defecto (sin parámetro), igual que automatizaciones. Debe sobrevivir a recarga y permitir deep-links.
11. **REQ-011**: El botón **Guardar** debe estar visible en ambas tabs y persistir en una sola operación los datos de la tab Información y los analizadores de la tab Analizadores (comportamiento actual de `handleSubmit`, sin cambios de backend).
12. **REQ-012**: Si se envía el formulario con el nombre vacío desde la tab Analizadores, el formulario debe cambiar a la tab Información, marcar el campo como inválido y mostrar un mensaje de error. No debe crear ni actualizar la institución.
13. **REQ-013**: Al guardar una institución nueva o editada se debe mantener el comportamiento actual: mostrar toast de éxito y navegar a `/institutions`.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-014**: La página de vista debe ofrecer también la opción **Eliminar** en el menú de 3 puntos, reutilizando `DeleteModal` y navegando a `/institutions` tras eliminar. Si el backend rechaza el borrado por tener registros asociados, se debe mostrar un mensaje claro de "registros asociados" (detectando errores de FK) y no navegar.
2. **REQ-015**: La página de vista debe mostrar un contador en formato singular/plural ("1 analizador" / "N analizadores") y un estado vacío claro cuando no haya analizadores.
3. **REQ-016**: Los textos nuevos visibles (tabs, campos, estados vacíos, errores y acciones) deben agregarse a `frontend/public/i18n/es.json` y `frontend/public/i18n/en.json` con paridad de claves. No se deben hardcodear textos nuevos en los componentes.
4. **REQ-017**: La tab debe mostrar un indicador de error (punto rojo) cuando el nombre esté vacío, siguiendo el patrón `hasError` de `AutomationFormPage`.
5. **REQ-018**: La página de vista debe usar `usePageTitle` con el nombre de la institución (y un fallback mientras carga o si no existe).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-019**: Mostrar en la vista la fecha de creación y/o última actualización de la institución (`created_at`, `updated_at`), si se considera útil. No es bloqueante para el release.
2. **REQ-020**: Evaluar (sin implementar en esta entrega) reutilizar la tab de analizadores como componente compartido con otras entidades que gestionan analizadores.

### 2.4 Fuera de alcance

- No se modifica el backend: se reutilizan `GET /api/institutions/{id}`, `GET /api/institutions/{id}/analyzers`, `GET /api/analyzers`, `PUT /api/institutions/{id}/analyzers`, `POST /api/institutions` y `PUT /api/institutions/{id}`.
- No se modifica el esquema SQLite ni se agregan migraciones.
- No se cambia el modelo de datos ni la relación N:M institución–analizador.
- No se rediseña el listado de instituciones ni su patrón de EmptyCard.
- No se optimiza el N+1 actual del listado (una llamada `getAnalyzers` por institución); queda como deuda técnica existente, sin regresión.
- No se agrega retorno automático a la vista después de guardar (se mantiene `/institutions`).

### 2.5 Requerimientos No Funcionales

- **Rendimiento**: la vista hace como máximo 4 requests cortos en paralelo al abrirse (institución, analizadores asignados, catálogo de analizadores y categorías), necesarios para mostrar el nombre/ícono de la categoría. Sin impacto en el listado salvo el cambio de destino del clic.
- **Seguridad**: todas las rutas siguen detrás del layout autenticado y de la sesión existente. No se exponen datos nuevos.
- **Almacenamiento**: sin cambios; cero migraciones.
- **Disponibilidad**: sin cambios en el server Go.
- **iHost**: sin dependencias nuevas; el build de React se sigue haciendo fuera del iHost. Impacto en RAM/CPU despreciable.
- **Accesibilidad**: tabs como botones con `type="button"` y estado activo distinguible; el tab activo debe ser navegable por teclado; los íconos deben tener `aria-label` cuando son la única información.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Archivos y patrones relevantes del repositorio:

- `frontend/src/pages/InstitutionsPage.tsx`: listado actual. El card navega a `/institutions/edit/:id` y el `CardMenu` ya incluye Editar y Eliminar.
- `frontend/src/pages/InstitutionFormPage.tsx`: formulario único con nombre, categoría y gestión de analizadores (lista + `AnalyzerPickerModal`).
- `frontend/src/pages/AutomationFormPage.tsx`: patrón de tabs con URL (`?tab=info|plugin|frequency`), indicador `hasError` y botón Guardar siempre visible. Es la referencia visual pedida por el usuario.
- `frontend/src/pages/AutomationDetailPage.tsx` y `frontend/src/pages/AutosPage.tsx`/`AutoShowPage.tsx`: patrón de detalle con card clickeable a vista y `CardMenu` con Editar/Eliminar.
- `frontend/src/App.tsx`: rutas de instituciones actuales (`/institutions`, `/institutions/new`, `/institutions/edit/:id`).
- `frontend/src/components/DashboardLayout.tsx`: `BACK_ROUTES` ya incluye `/institutions/(new|edit/\d+)` → `/institutions`; falta la ruta de vista.
- `frontend/src/api/index.ts`: `api.institutions.get`, `getAnalyzers`, `setAnalyzers`, `api.analyzers.list`.
- `frontend/src/types.ts`: `Institution`, `InstitutionCategory`, `InstitutionAnalyzer`, `AnalyzerInfo`.
- `frontend/public/i18n/es.json` y `en.json`: claves `institutions.title`, `institutions.create`, `institutions.edit`; el resto de textos del formulario está hardcodeado en español.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A: Página de vista nueva + tabs con URL en el formulario existente | Sigue el patrón del proyecto (autos/automatizaciones), reutiliza endpoints y componentes, cero backend | Agrega un archivo de página y estado de tab | ✅ Seleccionada |
| B: Modal de vista dentro del listado | No agrega ruta | Rompe el patrón de páginas de detalle, peor UX móvil y sin deep-link | ❌ Rechazada |
| C: Mantener el clic a edición y solo agregar tabs | Menor cambio | No resuelve la necesidad de consultar sin editar | ❌ Rechazada |
| D: Guardar la tab activa solo en estado local (`useState`) | Simple | Pierde la tab al recargar/compartir URL; difiere del patrón de automatizaciones | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: La tab activa vive en la URL (`?tab=info|analyzers`).
- **Contexto**: `AutomationFormPage` ya resolvió el problema con `useSearchParams` y la URL sobrevive a recargas y deep-links.
- **Decisión**: replicar el patrón: `info` es el default sin parámetro; `analyzers` se serializa como `?tab=analyzers`; los cambios usan `replace: true`.
- **Consecuencias**: consistencia de UX, links compartibles y cero estado duplicado.

**ADR-002**: La validación del nombre se hace en JS además del atributo `required`.
- **Contexto**: con tabs y renderizado condicional, el input de Nombre no está montado cuando la tab activa es Analizadores; el `required` HTML no bloquea el submit en ese caso.
- **Decisión**: en `handleSubmit`, si `formData.name.trim()` está vacío, cambiar a la tab Información, enfocar el input y mostrar error; abortar el guardado.
- **Consecuencias**: evita crear instituciones sin nombre y evita guardados parciales (primero se valida, luego se hacen los requests existentes).

**ADR-003**: La vista obtiene los nombres de analizadores desde `GET /api/analyzers`.
- **Contexto**: `GET /api/institutions/{id}/analyzers` devuelve `analyzer_id` (string del registry), no el nombre legible.
- **Decisión**: cargar en paralelo la institución, sus analizadores y el catálogo de analizadores; mapear `analyzer_id` → `name` y usar el ID crudo como fallback si el analizador ya no existe en el registry.
- **Consecuencias**: no requiere cambios de backend y tolera analizadores eliminados/renombrados.

**ADR-004**: Se reutiliza `CardMenu` y `DeleteModal` en la vista.
- **Contexto**: el patrón de acciones de detalle ya existe en automatizaciones y autos.
- **Decisión**: usar `CardMenu` con Editar como primera opción y Eliminar como opción destructiva (P1).
- **Consecuencias**: consistencia visual y cero componentes nuevos de menú.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[/institutions (InstitutionsPage)]
        │ clic en card
        ▼
[/institutions/:id (InstitutionShowPage)] ── CardMenu ──► [Editar] ──► [/institutions/edit/:id]
        │                                              └─► [Eliminar] ─► DELETE ─► [/institutions]
        │
        └── GET /api/institutions/:id
            GET /api/institutions/:id/analyzers
            GET /api/analyzers
                    │
                    ▼
              [SQLite vía API Go existente]

[/institutions/new | /institutions/edit/:id (InstitutionFormPage)]
        ├── tab Información  (?tab=info, default)
        │     ├── Nombre
        │     └── Categoría
        └── tab Analizadores (?tab=analyzers)
              ├── asignados + remover
              └── Agregar analizador → AnalyzerPickerModal
```

No hay cambios en backend ni en la base de datos.

### 4.2 Componentes

#### 4.2.1 `InstitutionShowPage` (nuevo)
- **Responsabilidad**: mostrar el detalle de una institución y sus analizadores; ofrecer acciones Editar/Eliminar.
- **Interfaz**: consume `api.institutions.get(id)`, `api.institutions.getAnalyzers(id)`, `api.analyzers.list()`, `api.institutionCategories.list()`, `api.institutions.delete(id)`.
- **Dependencias**: `CardMenu`, `DeleteModal`, `LoadingSpinner`, `Icon`, `usePageTitle`, `useI18nStore`, `useToast`.
- **Ubicación**: `frontend/src/pages/InstitutionShowPage.tsx`.
- **Estados**: `loading` (spinner), `notFound` (card con mensaje y botón a `/institutions`), `loaded` (contenido), `deleteOpen` (modal).

#### 4.2.2 `InstitutionsPage` (modificado)
- **Responsabilidad**: listado de instituciones.
- **Cambio**: el `onClick` del card navega a `/institutions/${inst.id}`. El `CardMenu` mantiene Editar y Eliminar.

#### 4.2.3 `InstitutionFormPage` (modificado)
- **Responsabilidad**: crear/editar institución con tabs.
- **Cambio**: agregar `FormTab = 'info' | 'analyzers'`, `useSearchParams`, barra de tabs, renderizado condicional del contenido y validación de nombre en `handleSubmit`.
- **Ubicación**: `frontend/src/pages/InstitutionFormPage.tsx`.

#### 4.2.4 `App.tsx` (modificado)
- Agregar la ruta `institutions/:id` → `InstitutionShowPage`, después de `institutions/new` e `institutions/edit/:id`.
- La ruta debe ignorar IDs no numéricos (`Number(id)` no finito) mostrando el estado `notFound`.

#### 4.2.5 `DashboardLayout.tsx` (modificado)
- Agregar a `BACK_ROUTES`: `{ pattern: /^\/institutions\/\d+$/, to: '/institutions' }`.
- La entrada existente `/institutions/(new|edit/\d+)` se mantiene.

#### 4.2.6 i18n (modificado)
- Archivos fuente: `frontend/public/i18n/es.json` y `frontend/public/i18n/en.json`.
- Claves propuestas (nombres exactos a definir en implementación, con paridad ES/EN):
  - `institutions.tab_info`, `institutions.tab_analyzers`
  - `institutions.name`, `institutions.category`, `institutions.uncategorized`
  - `institutions.assigned_analyzers`, `institutions.no_analyzers`, `institutions.no_analyzers_hint`
  - `institutions.add_analyzer`, `institutions.remove_analyzer`
  - `institutions.analyzer_count_one`, `institutions.analyzer_count_other`
  - `institutions.name_required`, `institutions.not_found`
  - `institutions.view_details` (si se usa en el menú del listado)
- Verificar con `curl -s http://localhost:8088/i18n/es.json` después del build.

### 4.3 Modelo de datos

Sin cambios. Se reutilizan:

```
Institución (institutions)
- id: integer
- name: text
- category_id: integer nullable → institution_categories
- created_at / updated_at

Institución–Analizador (institution_analyzers)
- institution_id: integer
- analyzer_id: text (registry de analizadores Go)
```

### 4.4 APIs / Contratos

No se agregan ni modifican endpoints. Se consumen los existentes:

#### `GET /api/institutions/{id}`
Response 200: `{ "id": 1, "name": "Proveedor Demo", "category_id": 2, "created_at": "...", "updated_at": "..." }`

#### `GET /api/institutions/{id}/analyzers`
Response 200: `[{ "id": 1, "institution_id": 1, "analyzer_id": "demo-analyzer", "created_at": "..." }]`

#### `GET /api/analyzers`
Response 200: `[{ "id": "demo-analyzer", "name": "Analizador Demo" }]`

#### `PUT /api/institutions/{id}/analyzers`
Request: `{ "analyzer_ids": ["demo-analyzer"] }`

#### `POST /api/institutions` / `PUT /api/institutions/{id}`
Request: `{ "name": "Proveedor Demo", "category_id": 2 }`

#### `DELETE /api/institutions/{id}`
Response 204/200. Si falla por dependencias, la UI muestra el mensaje del backend.

### 4.5 Dependencias

- **Internas**: `api.institutions`, `api.analyzers`, `api.institutionCategories`, `CardMenu`, `DeleteModal`, `AnalyzerPickerModal`, `Select`, `LoadingSpinner`, `usePageTitle`, `BACK_ROUTES`.
- **Externas**: ninguna nueva. React, React Router y Tailwind ya están en el proyecto.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado el listado `/institutions`, cuando el usuario hace clic en el card de una institución, entonces navega a `/institutions/:id` y ve la página de vista (no el formulario de edición).
- [ ] CA-002: Dado el listado `/institutions`, cuando el usuario abre el menú de 3 puntos de un card, entonces ve **Editar** y **Eliminar**, y Editar navega a `/institutions/edit/:id` con los datos precargados.
- [ ] CA-003: Dada la vista `/institutions/:id`, entonces se muestran nombre, categoría (o "Sin categoría"), cantidad de analizadores y el listado de analizadores asignados con nombre legible.
- [ ] CA-004: Dada la vista `/institutions/:id`, cuando el usuario abre el menú de 3 puntos, entonces la primera opción es **Editar** y navega al formulario de edición.
- [ ] CA-005: Dado un ID inexistente o no numérico en `/institutions/:id`, entonces se muestra el estado "Institución no encontrada" con botón para volver a `/institutions`, sin errores de consola no controlados.
- [ ] CA-006: Dado el formulario `/institutions/new`, entonces se muestran las tabs **Información** y **Analizadores**, con Información activa por defecto.
- [ ] CA-007: Dado el formulario `/institutions/edit/:id`, entonces se muestran las mismas dos tabs y la tab Información precarga nombre y categoría; la tab Analizadores precarga los analizadores asignados.
- [ ] CA-008: Dada cualquiera de las tabs, cuando el usuario recarga la página, entonces la tab activa se conserva (`?tab=analyzers` para Analizadores; sin parámetro para Información).
- [ ] CA-009: Dada la tab Analizadores, cuando el usuario agrega o remueve analizadores y pulsa Guardar, entonces los cambios se persisten y se reflejan en la vista y en el listado.
- [ ] CA-010: Dada la tab Analizadores con el nombre vacío, cuando el usuario pulsa Guardar, entonces la UI cambia a la tab Información, marca el error de nombre y no ejecuta ningún request de creación/actualización.
- [ ] CA-011: Dada una institución existente, cuando el usuario edita nombre y categoría y guarda, entonces se persisten ambos cambios y la vista muestra los valores actualizados.
- [ ] CA-012: Dado el flujo de eliminar desde la vista, cuando se confirma, entonces la institución se elimina y se navega a `/institutions`; si el backend rechaza el borrado por registros asociados, se muestra un mensaje claro de "registros asociados" y no se navega.
- [ ] CA-DARK: Los inputs/selects del formulario usan tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y se verificó legibilidad de texto y placeholders en darkmode.
- [ ] CA-BACK: La ruta `/institutions/:id` está registrada en `BACK_ROUTES`; en móvil la flecha atrás del header vuelve a `/institutions` y no existen links "← Instituciones" dentro del contenido.

### 5.2 No funcionales

- [ ] CA-NF-001: `npm run build` en `frontend/` finaliza sin errores ni warnings nuevos.
- [ ] CA-NF-002: No se agregan dependencias ni se modifica `go.mod`, `package.json` ni el esquema SQLite.
- [ ] CA-NF-003: La vista no genera más de 4 requests al abrirse (institución, analizadores asignados, catálogo de analizadores y categorías).
- [ ] CA-NF-004: Funciona correctamente en viewport móvil (375 px) y desktop, sin overflow horizontal.

### 5.3 Testing

- **Build**: `cd frontend && npm run build`.
- **Manual / E2E**:
  1. Listado → clic en card → vista.
  2. Vista → menú 3 puntos → Editar → guardar → listado → vista refleja cambios.
  3. Nuevo: completar solo Información y guardar; verificar creación.
  4. Nuevo: ir a Analizadores con nombre vacío y guardar; verificar switch a Información y que no se crea nada.
  5. Editar: agregar/remover analizadores, recargar con `?tab=analyzers` y verificar persistencia.
  6. Deep-link directo a `/institutions/edit/1?tab=analyzers`.
  7. ID inexistente `/institutions/999999` → estado not found.
  8. Darkmode: tabs, inputs, placeholders y estados vacíos.
  9. Móvil: flecha atrás en la vista y en el formulario.
- **Regresión**: listado con y sin instituciones (EmptyCard), eliminar desde el listado, y creación/edición desde el flujo existente.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | i18n ES/EN, ruta `/institutions/:id` y entrada en `BACK_ROUTES` | 0.25 día | Ninguna |
| 2 | `InstitutionShowPage` + cambio de navegación del listado + menú Editar/Eliminar | 0.5 día | Fase 1 |
| 3 | Tabs en `InstitutionFormPage` + validación de nombre + URL `?tab=` | 0.5 día | Fase 1 |
| 4 | Build, pruebas manuales (darkmode, móvil, deep-links) y ajustes | 0.25 día | Fases 2 y 3 |

### 6.2 Milestones

1. **MVP**: vista de institución con menú Editar y formulario con dos tabs funcionales (Información y Analizadores).
2. **V1.0**: eliminación desde la vista, contador/estado vacío de analizadores, i18n completo, validación de nombre y QA en darkmode/móvil.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| `/institutions/new` colisiona con `/institutions/:id` | Baja | Alto | Declarar las rutas estáticas primero y validar `Number(id)` finito; React Router prioriza segmentos estáticos. |
| El `required` HTML no valida el nombre cuando la tab Información no está montada | Alta | Medio | Validación JS en `handleSubmit` + switch a la tab Información (ADR-002). |
| Analizador asignado que ya no existe en el registry | Media | Bajo | Mostrar el `analyzer_id` crudo como fallback. |
| Guardado parcial (institución creada y analizadores fallan) | Media | Medio | Mantener el orden actual y mostrar el error del backend; documentar que no hay transacción entre ambos requests (comportamiento preexistente). |
| N+1 de `getAnalyzers` en el listado | Alta (preexistente) | Bajo | Fuera de alcance; no se agrava con este cambio. Se deja como deuda técnica. |
| Diferencias de texto ES/EN | Media | Bajo | Checklist de paridad de claves y build + verificación de `/i18n/*.json`. |

## 8. Notas y Referencias

- SPEC-009: Módulo de Instituciones, Analizadores de Documentos y Extracción Automática de Facturas.
- SPEC-013: Mejora de UI para gestión de analizadores en instituciones.
- SPEC-063: Flecha atrás en el header para páginas de detalle.
- SPEC-066: Aislamiento de worktrees y `main` de solo lectura.
- SPEC-114: Opción Editar en el menú de acciones del detalle de servicio (precedente del patrón Editar en detalle).
- Patrón de tabs: `frontend/src/pages/AutomationFormPage.tsx`.
- Patrón de detalle con `CardMenu`: `frontend/src/pages/AutomationDetailPage.tsx`, `frontend/src/pages/AutoShowPage.tsx`.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-10-21 | opencode | Creación inicial de la especificación (SPEC-116). |
| 2026-10-21 | opencode | Estado a `in_progress`. Se ajusta el presupuesto de requests de la vista a 4 (se agrega `institutionCategories.list()` para nombre/ícono de categoría). |
| 2026-10-21 | opencode | Implementación frontend completada: `InstitutionShowPage`, navegación desde el listado, tabs `Información`/`Analizadores` con `?tab=`, validación de nombre e i18n ES/EN. Build y tests Go OK. Pendiente validación manual del usuario. |
| 2026-10-21 | opencode | Manejo amigable de errores de FK al eliminar una institución con registros asociados (vista y listado). Sin cambios de backend. |
