---
title: "Selector de casas con modal de búsqueda en la página de Servicios"
id: "SPEC-062"
status: "released"
author: "paulomcnally"
created: "2026-09-04"
updated: "2026-09-04"
github_issue: 63
---

# Selector de casas con modal de búsqueda en la página de Servicios

**ID**: SPEC-062  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-09-04  
**Actualizado**: 2026-09-04

---

## 1. Resumen Ejecutivo

En la página de Servicios (`ServicesPage.tsx`) el filtro por casa se implementa actualmente con el componente `Select` (dropdown con búsqueda inline). El usuario quiere reemplazar ese dropdown por una sección/botón clickeable que, al hacer click, abra un modal dedicado donde se pueda escribir el nombre de una casa (búsqueda) y ver las casas disponibles para seleccionarlas.

El botón muestra el texto **"Casas"** cuando no hay filtro aplicado, o el **nombre de la casa seleccionada** cuando sí hay una casa elegida. Esto mejora la visibilidad del filtro activo y da más espacio/usabilidad a la búsqueda, en línea con los modales de búsqueda ya existentes en la app (ej: `IconPickerModal`, `AnalyzerPickerModal`).

Es un cambio **100% frontend** sin impacto en backend ni SQLite: solo se reemplaza el control de UI en `ServicesPage.tsx` y se agrega un nuevo componente modal reutilizable (`HousePickerModal`). En el iHost solo cambia el bundle estático servido, sin aumento relevante de memoria.

**Consideraciones de UI obligatorias**: el input de búsqueda del modal debe usar los tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y verificarse en darkmode antes de considerar completo el spec.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Reemplazar el dropdown `Select` de filtro de casas en `ServicesPage.tsx` por una sección/botón clickeable que abra un modal de selección de casas. El botón se ubica en el **header** de la página, alineado a la derecha junto al menú de crear (`CreateMenu`), para no dejar espacio vacío a la derecha en desktop (cambio iterativo solicitado por el usuario en evaluación).
2. **REQ-002**: El botón muestra el texto "Casas" (`services.homes`) cuando no hay filtro aplicado, o el nombre de la casa seleccionada cuando hay una casa elegida.
3. **REQ-003**: El modal incluye un campo de búsqueda donde se puede escribir el nombre de una casa; el listado de casas se filtra en tiempo real de forma case-insensitive.
4. **REQ-004**: Al seleccionar una casa en el modal, se aplica el filtro a la lista de servicios, el modal se cierra y el botón muestra el nombre de la casa seleccionada.
5. **REQ-005**: El modal ofrece una opción "Todas las casas" (`services.all_homes`) que limpia el filtro, vuelve a mostrar todos los servicios y deja el botón en el texto "Casas".
6. **REQ-006**: Agregar las claves i18n nuevas en `es` y `en` en `frontend/public/i18n/` (fuente de verdad) y mantener en sync `frontend/src/i18n/` (dict bundled), luego correr `npm run build` y verificar con `curl -s http://localhost:8088/i18n/es.json`.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-007**: El modal cierra con la tecla `Escape` y al hacer click fuera de él (patrón estándar de modales existentes).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-008**: Si en el futuro otra página necesita filtrar por casa (ej: BillsPage), `HousePickerModal` se puede reutilizar; no es obligatorio en este spec.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Sin impacto relevante; filtrado de casas en memoria (listado corto). Cero requests extra al backend.
- **Seguridad**: Sin cambios; el componente usa datos ya cargados en el store (`useAppStore.homes`).
- **Almacenamiento**: Sin cambios en disco ni en SQLite.
- **Disponibilidad**: Sin cambios en health checks ni arranque.
- **iHost**: Solo bundle estático más pequeño; sin dependencias nuevas ni consumo adicional de RAM.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- `ServicesPage.tsx` (líneas 78-89) usa el componente `Select` con `searchable` y la clave i18n `services.all_homes` ("Todas las casas"). El estado `homeFilter` se actualiza y dispara la recarga de servicios vía `useEffect`.
- `Select.tsx` implementa un dropdown con dropdown-menu flotante via `createPortal`; el usuario quiere pasar de este control a un modal dedicado.
- `IconPickerModal.tsx` y `AnalyzerPickerModal.tsx` son los modales de búsqueda existentes y definen el patrón a seguir: overlay `fixed inset-0 bg-black/40`, contenedor `bg-card rounded-ios`, input de búsqueda con `ref` autofocus, filtrado en memoria, cierre por `Escape` y click fuera.
- i18n: la fuente de verdad es `frontend/public/i18n/{es,en}.json` (AGENTS.md); `frontend/src/i18n/` alimenta el dict bundled en `i18nStore.ts` (fallback) y `/i18n/{lang}.json` se sirve desde `public/i18n/` (output del build con `emptyOutDir: true`).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Mantener `Select` y solo cambiar estilos | Cambio mínimo | No cumple el requerimiento (no es modal; búsqueda inline limitada) | ❌ Rechazada |
| Botón + modal nuevo `HousePickerModal` | UX solicitada exacta; reutilizable; patrón ya existente en la app | Un componente y lógica UI nuevos (fácil de mantener) | ✅ Seleccionada |
| Componente nativo `<dialog>` | Semántica HTML | Menos control de estilos/tokens del tema en la base actual | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Reutilizar el patrón de modales existente en vez de `<dialog>` o dropdown.
- **Contexto**: La app ya tiene modales de búsqueda consistentes (`IconPickerModal`), con tokens del tema y cierre por Escape/click-fuera.
- **Decisión**: Crear `HousePickerModal` siguiendo el mismo patrón visual y de comportamiento.
- **Consecuencias**: Consistencia visual garantizada; mínimo código nuevo; sin dependencias externas.

**ADR-002**: No tocar backend ni almacenamiento.
- **Contexto**: El filtro de casas ya funciona con `homeFilter` + `api.services.list(homeFilter)`; las casas ya están en `useAppStore`.
- **Decisión**: El cambio es puramente de presentación; se conserva el estado `homeFilter` y su `useEffect` de recarga.
- **Consecuencias**: Cero riesgo sobre DB; el spec es frontend-only.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[ServicesPage.tsx]  --(homeFilter)-->  [api.services.list(homeFilter)] --> [Lista de servicios]
        |
        v
[Button "Casas"/nombre]  --click-->  [HousePickerModal]
                                          |-- search input (filtra useAppStore.homes)
                                          |-- lista de casas
                                          `-- selección -> setHomeFilter(id) | null
```

### 4.2 Componentes

#### 4.2.1 `HousePickerModal` (nuevo)
- **Responsabilidad**: Modal de selección de casa con búsqueda en tiempo real.
- **Interfaz**: Props `isOpen: boolean`, `homes: Home[]`, `selectedHomeId: number | null`, `onSelect: (id: number | null) => void`, `onClose: () => void`. Opcionalmente puede leer `homes` de `useAppStore` internamente.
- **Dependencias**: `useI18nStore`, `Icon` (opcional para el ícono del botón), tokens del tema.
- **Ubicación**: `frontend/src/components/HousePickerModal.tsx`

#### 4.2.2 Modificaciones en `ServicesPage.tsx`
- **Responsabilidad**: Reemplazar el bloque `Select` (líneas 78-89) por el botón + estado de modal, ubicado en el **header** de la página (fila del título), agrupado a la derecha junto a `CreateMenu` en un contenedor `flex items-center gap-2`. El botón es compacto (`inline-flex`, sin `w-full` ni `max-w-xs`), con ícono de casa, texto y chevron; el nombre de la casa se trunca en pantallas pequeñas.
- **Interfaz**: Nuevo estado `pickerOpen: boolean`. Al seleccionar: `setHomeFilter(id)` y cerrar modal; con "Todas las casas": `setHomeFilter(null)`.
- **Dependencias**: `HousePickerModal`.
- **Ubicación**: `frontend/src/pages/ServicesPage.tsx`

### 4.3 Modelo de datos

Sin cambios de modelo. Se usan las entidades existentes:

```
Entidad: Home (existente)
- id: number
- name: string
- Relaciones: Service (1:N) via home_id
```

### 4.4 APIs / Contratos

Sin cambios de API. El botón y el modal operan sobre `useAppStore().homes` y el estado local `homeFilter` (que ya alimenta `api.services.list`).

### 4.5 Dependencias

- **Internas**: `ServicesPage.tsx` (modificación), `HousePickerModal.tsx` (nuevo), i18n `frontend/public/i18n/{es,en}.json` y `frontend/src/i18n/{es,en}.json` (claves nuevas).
- **Externas**: Ninguna.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Dado el listado de servicios sin filtro, el botón (en el header, junto al menú de crear) muestra "Casas"; al hacer click se abre el modal con el listado de casas y un campo de búsqueda enfocado.
- [x] CA-002: Dado el modal abierto, al escribir texto en la búsqueda se filtran las casas en tiempo real (case-insensitive, por nombre).
- [x] CA-003: Dado el modal abierto, al seleccionar una casa el modal se cierra, la lista de servicios se filtra y el botón muestra el nombre de la casa seleccionada.
- [x] CA-004: Dado un filtro de casa activo, al elegir "Todas las casas" en el modal el filtro se limpia, se muestran todos los servicios y el botón vuelve a mostrar "Casas".
- [x] CA-005: Dado el modal abierto, al presionar `Escape` o clickear fuera se cierra sin cambiar el filtro.
- [x] CA-DARK: El input de búsqueda usa tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y se verificó legibilidad en darkmode (texto y placeholder). No usar `bg-white`/`text-black` ni colores hardcodeados.

### 5.2 No funcionales

- [x] CA-NF-001: No se agregan dependencias nuevas al proyecto y el bundle frontend no aumenta de forma significativa.
- [x] CA-NF-002: Sin cambios en backend, base de datos o endpoints.

### 5.3 Testing

- **Unit tests**: No aplica (no hay lógica de negocio nueva).
- **Integration tests**: Verificar en local que el filtro por casa sigue llamando `api.services.list(homeFilter)` correctamente.
- **E2E tests**: Prueba manual en local: abrir modal, buscar, seleccionar, limpiar filtro; verificar en darkmode y mobile.
- **Carga/Performance**: No aplica (sin requests nuevos; filtrado en memoria).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Crear componente `HousePickerModal.tsx` (overlay, input de búsqueda con autofocus, listado con opción "Todas las casas", cierre por Escape/click-fuera) | 0.5 días | Ninguna |
| 2 | Reemplazar el `Select` en `ServicesPage.tsx` por el botón + modal | 0.5 días | Fase 1 |
| 3 | Agregar claves i18n `services.homes` (y ajustar si hace falta) en es/en, build y verificación | 0.5 días | Fase 2 |

### 6.2 Milestones

1. **MVP**: Botón + modal funcional con búsqueda, selección y limpieza de filtro.
2. **V1.0**: i18n en es/en, darkmode verificado, build de producción servido en local para evaluación del usuario.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Olvidar editar `frontend/public/i18n/` (fuente de verdad) y perder claves en el build | Media | Alto | Editar siempre `frontend/public/i18n/{es,en}.json` + sync con `frontend/src/i18n/`; correr `npm run build` y verificar `/i18n/es.json` (precedente SPEC-032/033) |
| Input de búsqueda ilegible en darkmode | Baja | Medio | Usar tokens del tema (`bg-card`, `text-text`, placeholder `--color-text-secondary`); verificar darkmode antes de cerrar |
| Romper el filtro existente de servicios | Baja | Medio | Conservar `homeFilter` y el `useEffect` de recarga; probar selección y limpieza en local |

## 8. Notas y Referencias

- `frontend/src/pages/ServicesPage.tsx` (bloque `Select` líneas 78-89)
- `frontend/src/components/IconPickerModal.tsx` (patrón de modal de búsqueda a replicar)
- `frontend/src/components/Select.tsx` (componente a reemplazar en esta página)
- AGENTS.md: regla de i18n (`frontend/public/i18n/` es la fuente de verdad) y regla de tokens del tema en inputs/darkmode.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-04 | paulomcnally | Creación inicial de la especificación |
| 2026-09-04 | paulomcnally | Cambio de estado a `pending_execution` |
| 2026-09-04 | paulomcnally | Cambio de estado a `in_progress` e inicio del desarrollo |
| 2026-09-04 | paulomcnally | Cambio iterativo durante evaluación: mover el selector de casas al header (derecha, junto a `CreateMenu`) como botón compacto, para evitar espacio vacío en desktop. Actualiza REQ-001, sección 4.2.2 y CA-001. |
| 2026-09-04 | paulomcnally | Usuario confirma pruebas manuales satisfactorias. Criterios de aceptación en `pass`. Estado a `released`. |