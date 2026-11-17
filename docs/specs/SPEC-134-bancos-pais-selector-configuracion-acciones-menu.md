---
title: "Bancos: país desde configuración con selector buscable y acciones en menú de 3 puntos"
id: "SPEC-134"
status: "released"
author: "opencode"
created: "2026-11-17"
updated: "2026-11-17"
github_issue: 137
---

# Bancos: país desde configuración con selector buscable y acciones en menú de 3 puntos

**ID**: SPEC-134  
**Estado**: released  
**Autor**: opencode  
**Creado**: 2026-11-17  
**Actualizado**: 2026-11-17

---

## 1. Resumen Ejecutivo

El módulo de bancos (SPEC-133) agregó el campo `country` como un `input` de texto libre. Eso permite valores inconsistentes ("Nicaragua", "Nicaragua ", "NI", "nic"), no ofrece ayuda al usuario y no se puede traducir ni comparar. Esta spec reemplaza ese input por un dropdown buscable con el listado completo de países (ISO 3166-1 alpha-2), usando el componente `Select` del proyecto (obligatorio por SPEC-118) y guardando el código ISO de dos letras.

Además, el país del banco debe sugerirse desde una preferencia global de Configuraciones. Se agrega el setting `country_code` (por defecto `NI` = Nicaragua), editable desde una nueva pantalla de Configuraciones. Al crear un banco, el selector se precarga con ese país; si el setting no existe o está vacío, se usa Nicaragua. Al editar, se muestra el país guardado del banco.

Por último, la página de detalle del banco hoy muestra un ícono de editar separado al lado del menú de 3 puntos. Se elimina ese ícono y la acción **Editar** pasa al menú de 3 puntos (`CardMenu`) junto con **Nueva cuenta**, siguiendo el patrón ya usado en `AutoShowPage` y `InstitutionShowPage`. La regla se documenta de forma permanente en `docs/project-rules.md` y `AGENTS.md` para que no vuelva a ocurrir en otras páginas de detalle.

Consideraciones de iHost: no se agregan dependencias ni llamadas de red. El listado de países es un arreglo estático en el bundle del frontend (≈250 entradas, ~12 KB sin comprimir, ~4 KB gzip), se ordena y filtra en memoria, y no requiere tabla ni migración nueva: el setting se guarda en la tabla `system_settings` existente (clave/valor). El backend solo valida el formato del código (`^[A-Z]{2}$`) y mantiene el límite actual de 80 caracteres de la columna `banks.country`.

Consideraciones de UI obligatorias:
- Todo `input`/`select`/`textarea` DEBE usar los tokens de color del tema (`bg-card`, `text-text`, `text-text-secondary`). Jamás colores hardcodeados.
- El selector DEBE ser el componente `frontend/src/components/Select.tsx` con `searchable`; prohibido `<select>` nativo (el build falla con `frontend/scripts/check-ui-rules.cjs`).
- El formulario debe verificarse en darkmode antes de considerarse completo.
- La página de Configuraciones del país tiene lista padre: el retorno va con flecha atrás en el header vía `BACK_ROUTES` de `frontend/src/components/DashboardLayout.tsx`; no crear links "← Título" dentro del contenido (SPEC-063).

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: El campo **País** del formulario de banco (crear y editar) deja de ser un `input` de texto y pasa a ser un dropdown buscable con todos los países del mundo, usando `Select` (`searchable`) del proyecto.
2. **REQ-002**: Debe existir un país global en Configuraciones (`country_code`), persistido en `system_settings`. Si el setting no existe o está vacío, el valor efectivo es `NI` (Nicaragua). El setting debe ser editable desde la UI de Configuraciones.
3. **REQ-003**: Al crear un banco, el selector de país se precarga con el país del setting global. Si el setting no existe o falla su lectura, se precarga Nicaragua (`NI`).
4. **REQ-004**: Al editar un banco, el selector muestra el país guardado del banco. Si el banco no tiene país, se precarga el país del setting global (o `NI`).
5. **REQ-005**: La página de detalle del banco deja de mostrar el ícono de editar separado. La acción **Editar** (`app.edit`) se mueve al menú de 3 puntos del header, junto con **Nueva cuenta** (`banks.create_account`).
6. **REQ-006**: La regla de UI "las acciones de entidad en páginas de detalle viven en el menú de 3 puntos, nunca en íconos separados" se agrega a `docs/project-rules.md` y a `AGENTS.md` como regla permanente.
7. **REQ-007**: El país guardado se almacena como código ISO 3166-1 alpha-2 (`NI`, `US`, `ES`, …). El backend valida el formato y devuelve error de validación si no cumple.
8. **REQ-008**: El listado y el detalle del banco muestran el nombre del país localizado según el idioma activo (es/en), resolviendo el código guardado. Si el valor almacenado no se puede resolver (dato legacy de texto libre), se muestra tal cual.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-009**: Compatibilidad con datos existentes: los bancos creados con SPEC-133 tienen `country` como texto libre. No se hace migración destructiva. El frontend resuelve nombres legacy (`Nicaragua`, `Estados Unidos`, `United States`, etc.) a su código ISO al editar; al guardar se persiste el código.
2. **REQ-010**: El valor desconocido/no resoluble se conserva visible como una opción temporal en el selector. Si el usuario intenta guardar sin reemplazarlo por un país válido, el formulario muestra una validación y no envía el cambio, de modo que el dato original permanezca en la DB hasta que se corrija.
3. **REQ-011**: El setting global muestra el nombre del país localizado en la fila de Configuraciones y en su pantalla de edición.
4. **REQ-012**: Botón para restaurar el país por defecto (`NI`) en la pantalla de Configuraciones, con feedback por toast.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-013**: Evaluar un guardrail automático en `frontend/scripts/check-ui-rules.cjs` que detecte botones de edición sueltos en páginas de detalle. Si resulta frágil (falsos positivos), dejar solo la regla documental y el checklist.
2. **REQ-014**: Mostrar la bandera/emoji del país junto al nombre en el selector (solo si no agrega peso ni dependencias).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: el listado de países se filtra en memoria; abrir el selector debe ser instantáneo (<50 ms en el iHost). Sin llamadas HTTP adicionales por cada búsqueda.
- **Memoria**: el arreglo estático de países no debe superar ~20 KB de JS sin comprimir; no se agregan librerías de países.
- **Almacenamiento**: sin migraciones nuevas. `system_settings` ya existe y `banks.country` sigue siendo `TEXT NOT NULL DEFAULT ''` (máximo 80 caracteres).
- **Seguridad**: el endpoint de settings requiere sesión como el resto; no se exponen datos sensibles.
- **iHost**: cero dependencias nuevas, cero servicios externos, cero descargas en runtime.
- **Compatibilidad**: la app debe seguir funcionando con bancos existentes sin país y con países en texto libre.

---

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Estado actual del módulo de bancos (SPEC-133)**:
  - `migrations/0047_create_banks.up.sql`: `banks.country TEXT NOT NULL DEFAULT ''`.
  - `internal/models/bank.go`, `internal/services/bank.go`, `internal/storage/bank.go`, `internal/api/bank_handlers.go`: el país es un string libre; el service solo valida longitud (`maxBankCountry = 80`).
  - `frontend/src/pages/BankFormPage.tsx`: `<input type="text">` para país.
  - `frontend/src/pages/BanksPage.tsx` y `BankDetailPage.tsx`: muestran `bank.country` o `banks.no_country`.
  - `BankDetailPage.tsx`: header con botón-ícono de editar + `CreateMenu` para "Nueva cuenta".
- **Patrón de acciones en detalle**: `AutoShowPage.tsx` y `InstitutionShowPage.tsx` usan `CardMenu` dentro de una tarjeta `relative`, con `{ label: t('app.edit'), icon: 'edit' }` y, cuando aplica, `{ label: t('app.delete'), danger: true }`.
- **Componente `Select`** (`frontend/src/components/Select.tsx`): dropdown custom con portal, `searchable` (autofocus y filtrado sin mayúsculas ni acentos), `z-[90]`, tokens de tema. Ya usado en `SettingsBillingPage` para zona horaria y en autos para color.
- **Settings globales** (`internal/services/system_settings.go`): tabla clave/valor con getters/setters tipados (`GetTimezone`, `SetTimezone`, `GetCurrencyFormat`…). El endpoint `GET/PATCH /api/system-settings` ya expone múltiples claves y acepta updates parciales.
- **ISO 3166-1 alpha-2**: estándar de códigos de país de dos letras. La lista oficial contiene 249 códigos asignados. Fuente: ISO Online Browsing Platform (<https://www.iso.org/obp/ui/#search/code/>) y la lista publicada en <https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2>. No se requiere descarga en runtime: la lista se versiona en el repo.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Lista estática de países en `frontend/src/constants/countries.ts` | Sin red, sin dependencias, filtrado inmediato, funciona offline en el iHost | Hay que mantener nombres localizados en el repo | ✅ Seleccionada |
| Endpoint backend `GET /api/countries` con lista embebida en Go | Fuente única del lado servidor | Round-trip extra en cada apertura/carga de formulario; duplica datos en backend y frontend | ❌ Rechazada |
| Librería npm (`i18n-iso-countries`, `country-list`) | Nombres y traducciones mantenidos por terceros | Dependencia nueva y bundle más grande en un iHost con recursos limitados; varias incluyen datos de más | ❌ Rechazada |
| Servicio externo (API de países) | Siempre actualizado | Dependencia de red obligatoria, latencia y caída del proveedor; prohibido por las reglas del proyecto | ❌ Rechazada |
| Guardar el nombre localizado del país | Legible directo en la DB | Se rompe al cambiar de idioma; comparaciones frágiles; duplica traducciones en datos | ❌ Rechazada |
| Guardar el código ISO alpha-2 | Estable, traducible, comparable, ocupa 2 caracteres | Requiere lookup para mostrar; datos legacy necesitan fallback | ✅ Seleccionada |
| Migración 0048 para convertir texto libre a códigos | Datos uniformes | Riesgo alto sobre datos reales y sin necesidad: el fallback en lectura/edición resuelve el 99% de los casos | ❌ Rechazada (se resuelve en frontend) |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Lista de países estática en el frontend**
- **Contexto**: el iHost tiene recursos limitados y no debe depender de internet para tareas básicas.
- **Decisión**: versionar un arreglo `COUNTRIES` con `{ code, name_es, name_en }` en `frontend/src/constants/countries.ts`, ordenado por nombre en español, y un helper `countryOptions(lang)`.
- **Consecuencias**: +≈12 KB al bundle (≈4 KB gzip), sin dependencias ni red. Mantener nombres es simple y queda bajo control del repo. El backend no necesita la lista completa porque solo valida formato.

**ADR-002: El setting global se guarda como `country_code` en `system_settings`**
- **Contexto**: ya existe una tabla clave/valor y un endpoint de settings con updates parciales.
- **Decisión**: clave `country_code`, valor ISO alpha-2 en mayúsculas. `GetCountryCode()` devuelve `NI` si no está configurado o si el valor es inválido. `SetCountryCode("")` limpia el setting (vuelve al default).
- **Consecuencias**: sin migración, sin cambios de esquema. El default vive en una constante Go (`DefaultCountryCode = "NI"`) y en el frontend (`DEFAULT_COUNTRY_CODE = 'NI'`), documentado como fallback sincronizado.

**ADR-003: Código ISO como valor persistido con resolución de legacy**
- **Contexto**: hay bancos con país en texto libre creados con SPEC-133.
- **Decisión**: guardar códigos nuevos; al leer, `resolveCountryCode(raw)` acepta código, nombre en español o nombre en inglés (case/acentos-insensible). Si no resuelve, se devuelve el texto original y el formulario lo ofrece como opción legacy visible.
- **Consecuencias**: cero migraciones y cero pérdida de datos. Los códigos nuevos son consistentes y traducibles. Un valor legacy se normaliza a código la próxima vez que se guarde el banco.

**ADR-004: `CardMenu` para las acciones del detalle del banco**
- **Contexto**: el detalle ya tiene un menú de 3 puntos (`CreateMenu`) y un ícono de editar aparte; los otros detalles del proyecto usan `CardMenu`.
- **Decisión**: refactorizar el header del detalle a una tarjeta `relative` (patrón `AutoShowPage`/`InstitutionShowPage`) y usar `CardMenu` con **Editar** y **Nueva cuenta**. Se elimina el botón-ícono de editar.
- **Consecuencias**: una sola entrada de acciones coherente en toda la app; se elimina el `CreateMenu` del detalle del banco. La acción de crear cuenta sigue accesible desde el menú y desde el empty state.

**ADR-005: Regla de UI documentada (no solo corregida en bancos)**
- **Contexto**: el mismo antipatrón (ícono de editar suelto) puede reaparecer en otras páginas.
- **Decisión**: agregar la regla a `docs/project-rules.md` (§4.8) y a `AGENTS.md` (Reglas críticas), y al checklist de calidad. La detección automática queda como P2 si es confiable.
- **Consecuencias**: la regla es vinculante para specs futuras y se revisa en el checklist antes de `pending_release`.

---

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[SettingsPage] --(PATCH country_code)--> [SystemSettingsHandlers] --> [SystemSettingsService] --> [system_settings]
      |                                                                                                   ^
      | GET /api/system-settings (country_code, default NI)                                               |
      v                                                                                                   |
[BankFormPage] --(precarga país)-------------------------------┐                                          |
      |                                                        |                                          |
      | Select(searchable) <-- COUNTRIES (constants/countries)  |                                          |
      v                                                        |                                          |
[POST/PUT /api/banks] --> [BankService: valida ^[A-Z]{2}$] --> [banks.country]                              |
      |                                                                                                   |
      v                                                                                                   |
[BanksPage / BankDetailPage] --(countryName(code, lang))--> nombre localizado / fallback legacy -----------┘
```

### 4.2 Componentes

#### 4.2.1 `frontend/src/constants/countries.ts` (nuevo)
- **Responsabilidad**: listado ISO 3166-1 alpha-2 con nombres en español e inglés, y helpers de resolución.
- **Interfaz**:
  ```ts
  export interface Country { code: string; name_es: string; name_en: string }
  export const DEFAULT_COUNTRY_CODE = 'NI'
  export const COUNTRIES: Country[] // 249, ordenado por name_es
  export function countryName(codeOrLegacy: string, lang: 'es' | 'en'): string
  export function countryOptions(lang: 'es' | 'en'): { value: string; label: string }[]
  export function resolveCountryCode(raw: string): string | null
  ```
- **Dependencias**: ninguna.
- **Ubicación**: `frontend/src/constants/countries.ts`.

#### 4.2.2 `SystemSettingsService` (extender)
- **Responsabilidad**: leer/escribir el país global con default y validación.
- **Interfaz**:
  ```go
  const CountryCodeKey = "country_code"
  const DefaultCountryCode = "NI"
  var countryCodeRegex = regexp.MustCompile(`^[A-Z]{2}$`)
  func (s *SystemSettingsService) GetCountryCode(ctx) (string, error)
  func (s *SystemSettingsService) SetCountryCode(ctx, code string) error // "" limpia; inválido → error
  ```
- **Ubicación**: `internal/services/system_settings.go`.

#### 4.2.3 `SystemSettingsHandlers` (extender)
- **Responsabilidad**: exponer y aceptar `country_code`.
- **Contrato**:
  - `GET /api/system-settings` → agrega `"country_code": "NI"`.
  - `PATCH /api/system-settings` → acepta `country_code?: string`. Valor inválido → `400 invalid_country`.
- **Ubicación**: `internal/api/system_settings_handlers.go`.

#### 4.2.4 `SettingsCountryPage` (nuevo)
- **Responsabilidad**: editar el país global con `Select searchable`; guarda automáticamente al elegir (mismo patrón que la zona horaria en `SettingsBillingPage`) y ofrece "Restaurar Nicaragua".
- **Ruta**: `/settings/pais`. Registrar en `App.tsx` y en `BACK_ROUTES` (`/settings/pais` → `/settings`).
- **Ubicación**: `frontend/src/pages/SettingsCountryPage.tsx`.

#### 4.2.5 `SettingsPage` (extender)
- **Responsabilidad**: nueva fila "País" con el nombre localizado del país global como subtítulo; navega a `/settings/pais`.
- **Ubicación**: `frontend/src/pages/SettingsPage.tsx`.

#### 4.2.6 `BankFormPage` (modificar)
- **Responsabilidad**: reemplazar el input de país por `Select searchable`; precargar default en alta; resolver legacy en edición; enviar el código ISO.
- **Detalle**: en alta, obtener `api.systemSettings.get()` y usar `country_code || DEFAULT_COUNTRY_CODE`. Si la llamada falla, usar `NI` sin bloquear el formulario. En edición, `resolveCountryCode(bank.country) ?? (bank.country || defaultCountry)`; si el valor no resuelve, agregar una opción legacy visible con `value = raw`, `label = raw`.
- **Ubicación**: `frontend/src/pages/BankFormPage.tsx`.

#### 4.2.7 `BankDetailPage` (modificar)
- **Responsabilidad**: eliminar el botón-ícono de editar; header en tarjeta `relative` con `CardMenu` (`Editar`, `Nueva cuenta`); mostrar el país localizado.
- **Ubicación**: `frontend/src/pages/BankDetailPage.tsx`.

#### 4.2.8 `BanksPage` (modificar)
- **Responsabilidad**: mostrar el país localizado en las cards y en el empty/estado sin país.
- **Ubicación**: `frontend/src/pages/BanksPage.tsx`.

#### 4.2.9 Reglas de UI (documentación)
- **Responsabilidad**: incorporar la regla de acciones en menú de 3 puntos.
- **Ubicación**: `docs/project-rules.md` §4.8 y `AGENTS.md` (Reglas críticas + checklist).

### 4.3 Modelo de datos

No hay cambios de esquema.

```
Setting (tabla system_settings, clave/valor)
- key: "country_code"
- value: "NI" | "US" | "ES" | ...   (ISO 3166-1 alpha-2, mayúsculas)
- default efectivo: "NI" si no existe o está vacío

Bank (tabla banks, existente)
- country: TEXT NOT NULL DEFAULT ''
- significado nuevo: código ISO alpha-2; admite texto legacy sin migración
```

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/system-settings`

**Response 200** (fragmento relevante):
```json
{
  "country_code": "NI",
  "timezone": "America/Managua"
}
```

#### Endpoint: `PATCH /api/system-settings`

**Request**:
```json
{
  "country_code": "US"
}
```

**Response 200**: el objeto completo de settings (mismo contrato que GET).

**Response Error**:
```json
{
  "error": "invalid_country",
  "message": "código de país inválido: \"USA\""
}
```

Reglas:
- `""` limpia el setting y el GET vuelve a devolver `NI`.
- Se normaliza a mayúsculas antes de validar; `us` se acepta y se guarda `US`.
- No se valida contra la lista completa en backend (para no duplicar 249 entradas); solo formato ISO alpha-2.

#### Endpoint: `POST /api/banks` y `PUT /api/banks/{id}` (sin cambios de contrato)

`country` sigue siendo string. El nuevo contrato de facto es un código ISO de 2 letras o vacío (para compatibilidad). El service valida:
- longitud ≤ 80 (existente);
- si no está vacío, debe cumplir `^[A-Z]{2}$` (nuevo, P0).

### 4.5 Dependencias

- **Internas**: `SystemSettingsService`, `SystemSettingsHandlers`, `BankService`, `Select`, `CardMenu`, `SettingsPage`, `BankFormPage`, `BankDetailPage`, `BanksPage`, `BACK_ROUTES`.
- **Externas**: ninguna nueva. Sin librerías de países ni APIs externas.

### 4.6 i18n

Nuevas claves en `frontend/public/i18n/es.json` y `en.json` (la fuente de verdad es `frontend/public/i18n/`):

| Clave | ES | EN |
|-------|----|----|
| `settings.nav.country` | País | Country |
| `settings.country.title` | País | Country |
| `settings.country.subtitle` | País usado por defecto al crear bancos | Default country used when creating banks |
| `settings.country.description` | Se sugiere al crear un banco. Podés cambiarlo en cada banco. | Suggested when creating a bank. You can change it per bank. |
| `settings.country.saved` | País guardado | Country saved |
| `settings.country.restore` | Restaurar Nicaragua | Restore Nicaragua |
| `banks.country_placeholder` | Seleccioná un país | Select a country |
| `banks.country_search` | Buscar país... | Search country... |
| `banks.country_legacy` | Valor anterior | Previous value |
| `banks.validation_country` | Elegí un país de la lista. | Choose a country from the list. |

No se elimina `banks.no_country` (fallback defensivo), aunque en la práctica deja de mostrarse.

---

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] **CA-001**: En `/bancos/new`, el campo País es un dropdown buscable (componente `Select`), no un `input` de texto ni un `<select>` nativo.
- [x] **CA-002**: El dropdown lista los 249 países del estándar ISO 3166-1 alpha-2, ordenados alfabéticamente por nombre en el idioma activo.
- [x] **CA-003**: El buscador del dropdown filtra por nombre sin distinguir mayúsculas ni acentos (ej: escribir `espana` encuentra `España`).
- [x] **CA-004**: Con el setting global en `NI`, al abrir `/bancos/new` el selector aparece precargado con Nicaragua.
- [x] **CA-005**: Sin setting global (clave ausente o vacía), al abrir `/bancos/new` el selector aparece precargado con Nicaragua.
- [x] **CA-006**: Al guardar un banco nuevo se persiste el código ISO de dos letras (`NI`, `US`, `ES`, …), verificable en la DB y en el JSON de respuesta del API.
- [x] **CA-007**: Al editar un banco con `country = "NI"` el selector muestra Nicaragua; al guardar sin cambios se conserva `NI`.
- [x] **CA-008**: Al editar un banco legacy con `country = "Nicaragua"` el selector muestra Nicaragua; al guardar se normaliza a `NI` sin error.
- [x] **CA-009**: Al editar un banco con un valor legacy no reconocido, ese valor aparece como opción seleccionable; si se intenta guardar sin elegir un país válido, se muestra un error de validación y no se envía la petición (el valor original no se pierde en la DB).
- [x] **CA-010**: El listado y el detalle del banco muestran el nombre del país localizado según el idioma activo; un valor legacy no resoluble se muestra tal cual.
- [x] **CA-011**: En Configuraciones existe la fila **País** con el país global actual como subtítulo; al tocarla se abre `/settings/pais`.
- [x] **CA-012**: En `/settings/pais` se puede elegir cualquier país con el dropdown buscable; al elegir se guarda automáticamente y se muestra el toast de confirmación; el valor persiste tras recargar.
- [x] **CA-013**: El botón **Restaurar Nicaragua** guarda `NI` y actualiza la UI.
- [x] **CA-014**: En el detalle de un banco (`/bancos/:id`) NO existe un ícono de editar separado; el menú de 3 puntos contiene **Editar** (clave `app.edit`) y **Nueva cuenta**.
- [x] **CA-015**: **Editar** en el menú de 3 puntos navega a `/bancos/edit/:id`.
- [x] **CA-016**: `docs/project-rules.md` contiene la regla §4.8 y `AGENTS.md` la referencia en Reglas críticas/checklist.
- [x] **CA-017**: El backend rechaza con `400` un `country_code` que no sea de dos letras (ej: `USA`, `1`, `N!`), sin persistir el valor.
- [x] **CA-018**: El backend acepta `country_code` en minúsculas y lo normaliza a mayúsculas.
- [x] **CA-019**: El backend rechaza con validación un `country` de banco con formato inválido (distinto de vacío y de `^[A-Z]{2}$`).
- [x] **CA-020**: `npm run check:ui` pasa y `npm run build` compila sin errores.
- [x] **CA-DARK**: Los campos del formulario de banco y de la pantalla de país usan tokens del tema (`bg-card`, `text-text`) y se verificó legibilidad en darkmode (texto y placeholders).
- [x] **CA-BACK**: La pantalla `/settings/pais` tiene flecha atrás en el header registrada en `BACK_ROUTES` y NO tiene links "← Título" en el contenido.

### 5.2 No funcionales

- [x] **CA-NF-001**: Abrir el selector y filtrar países responde de forma instantánea (percepción <50 ms en el iHost), sin peticiones HTTP por búsqueda.
- [x] **CA-NF-002**: El bundle del frontend crece menos de 20 KB sin comprimir por el listado de países.
- [x] **CA-NF-003**: No se agregan dependencias a `frontend/package.json` ni al `go.mod`.
- [x] **CA-NF-004**: No se agregan migraciones ni se modifica el esquema de `banks`/`system_settings`.
- [x] **CA-NF-005**: Los tests Go (`go test ./...`) y el build del frontend pasan.

### 5.3 Testing

- **Unit tests Go**:
  - `GetCountryCode` devuelve `NI` sin setting, `NI` con valor vacío y el código guardado cuando existe.
  - `SetCountryCode` normaliza minúsculas, acepta `NI`, rechaza `USA`/`1`/`N!` y limpia con `""`.
  - `BankService` rechaza `country` con formato inválido y acepta vacío.
- **Unit tests frontend** (si el proyecto tiene runner, o validación manual si no):
  - `resolveCountryCode('Nicaragua') === 'NI'`, `resolveCountryCode('nicaragua') === 'NI'`, `resolveCountryCode('US') === 'US'`, `resolveCountryCode('Atlantis') === null`.
  - `countryOptions('es')` devuelve 249 opciones ordenadas.
- **Integración**: `GET/PATCH /api/system-settings` con y sin `country_code`; `POST/PUT /api/banks` con `NI`, `nic` y `USA`.
- **Manual (local)**:
  1. Configuraciones → País → elegir España → guardar.
  2. Crear banco → verificar que España aparece precargado, cambiar a Nicaragua, guardar.
  3. Editar banco legacy con "Nicaragua" → verificar selección y normalización.
  4. Verificar darkmode en formulario y pantalla de país.
  5. Detalle de banco → abrir menú de 3 puntos → Editar y Nueva cuenta.
- **Carga/Performance**: no aplica; verificar que el selector abre sin demora en el iHost.

---

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Crear `constants/countries.ts` con los 249 países y helpers (`countryName`, `countryOptions`, `resolveCountryCode`) | 0.5 día | Ninguna |
| 2 | Backend: `GetCountryCode`/`SetCountryCode`, validación de `country` en bancos, GET/PATCH de settings y tests | 0.5 día | Ninguna |
| 3 | Frontend: `BankFormPage` con `Select searchable`, precarga desde settings y resolución legacy | 0.5 día | Fases 1 y 2 |
| 4 | Frontend: `SettingsCountryPage`, fila en `SettingsPage`, ruta, `BACK_ROUTES` e i18n | 0.5 día | Fase 2 |
| 5 | Frontend: `BankDetailPage` con `CardMenu` (Editar + Nueva cuenta); eliminar ícono suelto; `BanksPage` con país localizado | 0.3 día | Fase 1 |
| 6 | Documentación: §4.8 en `docs/project-rules.md`, regla y checklist en `AGENTS.md` | 0.2 día | Fase 5 |
| 7 | Tests, `npm run check:ui`, `npm run build`, `go test ./...` y prueba manual local (incluye darkmode) | 0.5 día | Fases 1-6 |

### 6.2 Milestones

1. **MVP**: selector de países en el formulario de banco, setting global con default Nicaragua, detalle con Editar en el menú de 3 puntos y regla documentada.
2. **V1.0**: resolución/normalización de datos legacy, país localizado en listado/detalle, pantalla de Configuraciones completa con restaurar default y pruebas.

---

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Nombres de países desactualizados o incompletos en la constante | Baja | Medio | Revisar contra ISO 3166-1 alpha-2 al implementar y dejar un test que valide cantidad (249) y unicidad de códigos |
| Datos legacy no reconocidos que se muestren como placeholder vacío | Media | Medio | Agregar opción temporal con el valor original cuando `resolveCountryCode` devuelve `null` |
| Divergencia del default `NI` entre Go y frontend | Baja | Bajo | Constante documentada en ambos lados + test de settings que verifique el default |
| Duplicación de nombres ES/EN mantenida a mano | Media | Bajo | Mantener ambos idiomas en la misma entrada y ordenar por el idioma activo; revisión en code review |
| El refactor del header del detalle rompe el layout en móvil | Media | Bajo | Usar el patrón ya probado de `AutoShowPage`/`InstitutionShowPage` y verificar en móvil y desktop |
| Regla de UI en docs sin enforcement automático | Media | Bajo | Agregarla al checklist de calidad y evaluar guardrail P2 solo si no genera falsos positivos |
| Setting no disponible al crear banco (fallo de red/endpoint) | Baja | Bajo | Fallback silencioso a `NI` sin bloquear el formulario |

---

## 8. Notas y Referencias

- SPEC-133: módulo de bancos, cuentas y formatos de transferencia.
- SPEC-118: color predefinido y guardrail de dropdowns (`Select` obligatorio, `check-ui-rules.cjs`).
- SPEC-117: menú Editar en el detalle del auto (precedente del patrón `CardMenu`).
- SPEC-115: estandarización de la etiqueta **Editar** (`app.edit`).
- SPEC-063: navegación de retorno en páginas de detalle (`BACK_ROUTES`).
- SPEC-066: README tracker se actualiza en el release.
- SPEC-078: patrón de setting de zona horaria con validación en el service.
- ISO 3166-1 alpha-2: <https://www.iso.org/obp/ui/#search/code/> y <https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2>.
- Componentes: `frontend/src/components/Select.tsx`, `frontend/src/components/CardMenu.tsx`.
- Reglas: `docs/project-rules.md` §4.1, §4.5, §4.6 y nuevo §4.8.

---

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-17 | opencode | Creación inicial de la especificación (país de banco con dropdown buscable, setting global con default Nicaragua, acciones del detalle en menú de 3 puntos y nueva regla de UI) |
| 2026-11-17 | opencode | Implementación completa, tests Go, `npm run check:ui`, `npm run build` y validación manual satisfactoria confirmada por el usuario. Release v0.6.0: merge a `main` (commit `fc0d99e`, merge `0ac489e`, bump `09820c8`); label `spec/released` e issue #137 cerrado. |
