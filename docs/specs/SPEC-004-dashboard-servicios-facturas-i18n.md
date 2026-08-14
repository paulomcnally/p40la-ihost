---
title: "Dashboard con Sidebar, Header, i18n, Settings estilo iOS, Iconos, Home/Casa y Módulo de Servicios con Facturas"
id: "SPEC-004"
status: "released"
author: "p40la-ihost-team"
created: "2026-08-12"
updated: "2026-08-12"
---

# {{title}}

**ID**: {{id}}  
**Estado**: {{status}}  
**Autor**: {{author}}  
**Creado**: {{created}}  
**Actualizado**: {{updated}}

---

## 1. Resumen Ejecutivo

Esta especificación define la evolución del frontend y backend de `p40la-ihost` hacia un dashboard estructurado con navegación lateral, header y área de contenido dinámica. Se introducen los módulos **Home/Casa**, **Services/Servicios** y **Settings/Configuración**, con soporte de internacionalización (i18n) en español e inglés.

El módulo de servicios permite gestionar servicios recurrentes de pago asociados a una casa (internet, basura, agua, electricidad, etc.), junto con su historial de facturas. Las facturas pueden generarse automáticamente según la frecuencia configurada para cada servicio y se registran con estados **Pendiente** o **Pagada**. Marcar una factura como pagada requiere obligatoriamente un enlace de Google Drive que apunte a un PDF o foto comprobante.

Todas las decisiones técnicas priorizan las restricciones del iHost: bajo consumo de memoria, dependencias mínimas, SQLite sin ORM y frontend vanilla sin frameworks pesados.

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Dashboard con layout de tres zonas: sidebar izquierdo, header superior y área principal de páginas.
2. **REQ-002**: Sidebar con menús que contienen submenús; cada submenú carga una página en el área principal.
3. **REQ-003**: Iconos asociados a cada ítem del menú y submenú del sidebar.
4. **REQ-004**: Header con icono de configuraciones (engranaje) en la parte superior derecha.
5. **REQ-005**: Página de configuraciones con diseño estilo iOS: secciones agrupadas, filas con título, subtítulo y control (toggle, flecha o valor).
6. **REQ-006**: Sistema de i18n global en español e inglés; el idioma se selecciona desde configuraciones y persiste en SQLite.
7. **REQ-007**: Gestión de monedas en configuraciones; monedas iniciales: USD y NIO; debe permitir agregar más monedas.
8. **REQ-008**: CRUD de Home/Casa: nombre y dirección opcional; soft delete.
9. **REQ-009**: No se pueden crear servicios si no existe al menos un Home/Casa.
10. **REQ-010**: CRUD de servicios asociados a un Home: nombre, institución, moneda, frecuencia (mensual/anual), monto sugerido, estado activo (toggle iOS) e icono seleccionado de lista predefinida; soft delete.
11. **REQ-011**: Listado de servicios en cards con filtro simple por Home.
12. **REQ-012**: Generación automática de facturas según la frecuencia configurada del servicio.
13. **REQ-013**: CRUD de facturas asociadas a un servicio: año, mes, monto, número de factura, estado (Pendiente/Pagada) y enlace de Google Drive; soft delete.
14. **REQ-014**: Las facturas creadas automáticamente inician en estado **Pendiente** con el monto sugerido del servicio.
15. **REQ-015**: Registrar o actualizar una factura como **Pagada** requiere obligatoriamente un enlace de Google Drive válido.
16. **REQ-016**: Mejorar el tracker de migraciones con tabla `schema_migrations` para controlar qué migraciones ya fueron aplicadas.
17. **REQ-017**: Los estados vacíos (empty states) deben mostrarse como una card centrada con título, subtítulo y botón de acción clara.
18. **REQ-018**: El botón principal de creación en empty states (ej. "Nueva casa") debe tener un estilo de card atractivo, sin fondo gris plano.
19. **REQ-019**: Los botones de acción deben incluir un icono que represente su función (crear, editar, eliminar, guardar, cancelar, facturas, etc.).
20. **REQ-020**: El botón principal de creación en cada página no debe mostrarse como botón tradicional, sino como un icono de tres puntos (⋮) alineado a la derecha que despliega un dropdown personalizado (no nativo del OS) con la opción de crear un registro. Aplica para Home, Servicios y futuras páginas.
21. **REQ-021**: Cada card debe incluir un icono de tres puntos (⋮) en la esquina superior derecha que despliegue un dropdown personalizado con las opciones "Editar" y "Eliminar", cada una con su icono de referencia.
22. **REQ-022**: La acción de eliminar siempre debe desplegar un modal de confirmación que advierta sobre la acción, con un campo de texto donde el usuario debe escribir "confirmo" para habilitar el botón de eliminar, y una opción de cancelar.
23. **REQ-023**: La navegación entre páginas debe actualizar la URL mediante la API History de HTML5, de forma que al refrescar el navegador en una página específica se cargue esa misma página y no se vuelva al inicio.
24. **REQ-024**: Nunca usar dropdowns nativos del sistema operativo; todos los selectores y menús desplegables deben ser componentes custom implementados con CSS.
25. **REQ-025**: Los formularios de CRUD (crear/editar) deben ser páginas completas, no modales. Los modales solo se usarán cuando el usuario lo solicite explícitamente.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

26. **REQ-026**: Edición del monto sugerido y datos de facturas generadas automáticamente cuando llega el documento real de la empresa.
27. **REQ-027**: Visualización del historial de facturas dentro del detalle de cada servicio.
28. **REQ-028**: Indicador visual de servicios con facturas pendientes.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

29. **REQ-029**: Filtro de servicios activos/inactivos.
30. **REQ-030**: Dashboard inicial con resumen de facturas pendientes por Home.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Tiempo de carga inicial del dashboard menor a 1 segundo en iHost.
- **Memoria**: El binario Go + SQLite no debe superar los 100 MB de RAM en uso normal.
- **Dependencias**: Solo librerías aprobadas en `docs/project-rules.md`; sin frameworks SPA ni ORM.
- **Almacenamiento**: SQLite con `journal_mode=WAL`; soft delete para conservar historial.
- **i18n**: Sin librerías externas; diccionario JSON estático servido desde `public/i18n/`.
- **Seguridad**: Validaciones de negocio en backend; nunca confiar solo en frontend.

---

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- El proyecto actual usa Go con `net/http`, SQLite (`modernc.org/sqlite`) y un sistema de migraciones casero.
- El frontend es HTML/CSS/JS vanilla servido desde `public/`.
- No existe aún un ORM ni un framework de UI; el proyecto lo prohíbe explícitamente por recursos limitados del iHost.
- La app actual ya tiene autenticación y tabla `settings` simple (clave/valor).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| ORM (GORM, Ent) | Menos código SQL manual | Más RAM, más dependencias, build más lento, va en contra de project-rules | ❌ Rechazada |
| SQL crudo con `database/sql` | Ligero, control total, cumple reglas | Más verboso | ✅ Seleccionada |
| i18n librería (i18next) | Funciones avanzadas | Dependencia JS pesada, build step | ❌ Rechazada |
| i18n JSON estático + JS vanilla | Cero dependencias, suficiente | Requiere claves consistentes | ✅ Seleccionada |
| Icon font (FontAwesome) | Fácil de usar | Dependencia externa, peso extra | ❌ Rechazada |
| SVG inline propio | Sin dependencias, escalable, liviano | Requiere mantener catálogo | ✅ Seleccionada |
| Sidebar framework (React Router) | Navegación robusta | SPA pesada, build complejo | ❌ Rechazada |
| Vanilla JS con carga dinámica de HTML | Liviano, sin build | Manejo manual de estado | ✅ Seleccionada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Sin ORM, SQL crudo con `database/sql`**
- **Contexto**: El iHost tiene recursos limitados y `docs/project-rules.md` prohíbe ORM.
- **Decisión**: Mantener el patrón `internal/storage/` con queries SQL manuales.
- **Consecuencias**: Mayor control, menor consumo de RAM, menor cantidad de dependencias.

**ADR-002: i18n mediante JSON estático**
- **Contexto**: Se requiere español e inglés sin agregar peso al frontend.
- **Decisión**: Servir archivos `public/i18n/es.json` y `public/i18n/en.json`; un módulo JS carga el diccionario y reemplaza atributos `data-i18n`.
- **Consecuencias**: Cero dependencias, cambio de idioma sin recarga de página.

**ADR-003: Iconos SVG inline**
- **Contexto**: Sidebar y servicios requieren iconos; se deben evitar dependencias externas.
- **Decisión**: Catálogo de iconos como spritesheet SVG embebido en `public/`, referenciados por clave.
- **Consecuencias**: Sin requests extra, sin librerías, totalmente escalable.

**ADR-004: Generación automática de facturas**
- **Contexto**: Servicios como internet (mensual) y basura (anual) requieren facturas recurrentes.
- **Decisión**: Al crear/editar un servicio activo, el backend genera la próxima factura pendiente según frecuencia. También se ejecuta un "reconciliation" ligero al consultar servicios.
- **Consecuencias**: El usuario no debe crear facturas manualmente cada periodo, pero puede ajustarlas luego.

**ADR-005: Monto decimal en facturas**
- **Contexto**: El proyecto opera en Nicaragua y maneja monedas como NIO y USD.
- **Decisión**: Almacenar montos como `DECIMAL(12,2)` en SQLite (`REAL` si el driver no soporta).
- **Consecuencias**: Representación exacta de centavos/decimales para uso local.

**ADR-006: Soft delete en entidades principales**
- **Contexto**: Se requiere conservar historial de pagos aunque se elimine un servicio o casa.
- **Decisión**: Agregar `deleted_at` en `homes`, `services` y `bills`; filtrar en queries.
- **Consecuencias**: Integridad histórica preservada; datos "eliminados" no se muestran en UI.

**ADR-007: Mejora del tracker de migraciones**
- **Contexto**: El sistema actual ejecuta todas las migraciones `.up.sql` en cada arranque sin llevar registro.
- **Decisión**: Crear tabla `schema_migrations` y modificar `internal/db/db.go` para aplicar solo migraciones no registradas.
- **Consecuencias**: Permite migraciones no idempotentes y control real del schema.

---

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
┌─────────────────┐     HTTP      ┌──────────────────┐
│  Navegador      │ ◄────────────►│  Go net/http     │
│  (HTML/CSS/JS)  │               │  internal/api/   │
└─────────────────┘               └────────┬─────────┘
                                           │
                              ┌────────────┼────────────┐
                              ▼            ▼            ▼
                       ┌──────────┐ ┌──────────┐ ┌──────────┐
                       │ services │ │  storage │ │ settings │
                       └────┬─────┘ └────┬─────┘ └─────┬────┘
                            │            │             │
                            └────────────┼─────────────┘
                                         ▼
                                  ┌──────────────┐
                                  │  SQLite WAL  │
                                  └──────────────┘
```

### 4.2 Componentes

#### 4.2.1 Frontend (`public/`)
- **Responsabilidad**: Renderizar UI, manejar navegación SPA ligera, i18n, llamadas a API.
- **Interfaz**: Ninguna; es la capa de presentación.
- **Dependencias**: Ninguna externa.
- **Ubicación**: `public/index.html`, `public/css/`, `public/js/`.

#### 4.2.2 API (`internal/api/`)
- **Responsabilidad**: Recibir requests HTTP, validar entrada, llamar servicios, devolver JSON.
- **Interfaz**: Endpoints REST documentados abajo.
- **Dependencias**: `internal/services/`.
- **Ubicación**: `internal/api/handlers.go`, `internal/api/routes.go`.

#### 4.2.3 Services (`internal/services/`)
- **Responsabilidad**: Lógica de negocio (generación de facturas, validaciones, filtros).
- **Interfaz**: Funciones llamadas por handlers.
- **Dependencias**: `internal/storage/`.
- **Ubicación**: `internal/services/`.

#### 4.2.4 Storage (`internal/storage/`)
- **Responsabilidad**: Única capa que ejecuta queries SQL contra SQLite.
- **Interfaz**: Métodos CRUD por entidad.
- **Dependencias**: `database/sql`, `internal/models/`.
- **Ubicación**: `internal/storage/`.

### 4.3 Modelo de datos

**Entidad: `schema_migrations`**
- `version`: TEXT PRIMARY KEY — nombre del archivo de migración
- `applied_at`: DATETIME DEFAULT CURRENT_TIMESTAMP

**Entidad: `settings`**
- `key`: TEXT PRIMARY KEY
- `value`: TEXT NOT NULL
- `updated_at`: DATETIME DEFAULT CURRENT_TIMESTAMP

**Entidad: `currencies`**
- `id`: INTEGER PRIMARY KEY AUTOINCREMENT
- `code`: TEXT UNIQUE NOT NULL — ej. USD, NIO
- `name`: TEXT NOT NULL
- `symbol`: TEXT NOT NULL
- `deleted_at`: DATETIME nullable

**Entidad: `homes`**
- `id`: INTEGER PRIMARY KEY AUTOINCREMENT
- `name`: TEXT NOT NULL
- `address`: TEXT nullable
- `deleted_at`: DATETIME nullable
- `created_at`: DATETIME DEFAULT CURRENT_TIMESTAMP
- `updated_at`: DATETIME DEFAULT CURRENT_TIMESTAMP

**Entidad: `services`**
- `id`: INTEGER PRIMARY KEY AUTOINCREMENT
- `home_id`: INTEGER NOT NULL FOREIGN KEY homes(id)
- `name`: TEXT NOT NULL
- `institution`: TEXT nullable
- `currency_id`: INTEGER NOT NULL FOREIGN KEY currencies(id)
- `frequency`: TEXT NOT NULL — `monthly`, `yearly`
- `suggested_amount`: DECIMAL(12,2) NOT NULL
- `active`: BOOLEAN DEFAULT 1
- `icon_key`: TEXT NOT NULL
- `deleted_at`: DATETIME nullable
- `created_at`: DATETIME DEFAULT CURRENT_TIMESTAMP
- `updated_at`: DATETIME DEFAULT CURRENT_TIMESTAMP

**Entidad: `bills`**
- `id`: INTEGER PRIMARY KEY AUTOINCREMENT
- `service_id`: INTEGER NOT NULL FOREIGN KEY services(id)
- `year`: INTEGER NOT NULL
- `month`: INTEGER NOT NULL
- `amount`: DECIMAL(12,2) NOT NULL
- `invoice_number`: TEXT nullable
- `status`: TEXT NOT NULL — `pending`, `paid`
- `drive_url`: TEXT nullable
- `deleted_at`: DATETIME nullable
- `created_at`: DATETIME DEFAULT CURRENT_TIMESTAMP
- `updated_at`: DATETIME DEFAULT CURRENT_TIMESTAMP

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/settings`
**Response 200**:
```json
{
  "language": "es",
  "theme": "light"
}
```

#### Endpoint: `POST /api/settings`
**Request**:
```json
{
  "language": "en"
}
```
**Response 200**: mismo objeto guardado.

#### Endpoint: `GET /api/currencies`
**Response 200**:
```json
[
  { "id": 1, "code": "NIO", "name": "Córdoba Nicaragüense", "symbol": "C$" },
  { "id": 2, "code": "USD", "name": "Dólar Estadounidense", "symbol": "$" }
]
```

#### Endpoint: `POST /api/currencies`
**Request**:
```json
{
  "code": "EUR",
  "name": "Euro",
  "symbol": "€"
}
```

#### Endpoint: `GET /api/homes`
**Response 200**:
```json
[
  { "id": 1, "name": "Casa Managua", "address": "..." }
]
```

#### Endpoint: `POST /api/homes`
**Request**:
```json
{
  "name": "Casa Granada",
  "address": "Calle Principal 123"
}
```

#### Endpoint: `GET /api/services?home_id=1`
**Response 200**:
```json
[
  {
    "id": 1,
    "home_id": 1,
    "name": "Internet",
    "institution": "Claro",
    "currency_id": 2,
    "frequency": "monthly",
    "suggested_amount": 45.00,
    "active": true,
    "icon_key": "internet"
  }
]
```

#### Endpoint: `POST /api/services`
**Request**:
```json
{
  "home_id": 1,
  "name": "Internet",
  "institution": "Claro",
  "currency_id": 2,
  "frequency": "monthly",
  "suggested_amount": 45.00,
  "active": true,
  "icon_key": "internet"
}
```
**Response Error** (sin homes):
```json
{
  "error": "NO_HOMES",
  "message": "Debe crear al menos un Home/Casa antes de registrar servicios."
}
```

#### Endpoint: `GET /api/services/:id/bills`
**Response 200**:
```json
[
  {
    "id": 1,
    "service_id": 1,
    "year": 2026,
    "month": 8,
    "amount": 45.00,
    "invoice_number": "12345",
    "status": "pending",
    "drive_url": null
  }
]
```

#### Endpoint: `PUT /api/bills/:id`
**Request** (marcar pagada):
```json
{
  "status": "paid",
  "drive_url": "https://drive.google.com/file/d/abc123/view"
}
```
**Response Error** (Drive inválido):
```json
{
  "error": "INVALID_DRIVE_URL",
  "message": "El enlace de Google Drive no es válido."
}
```

### 4.5 Dependencias

- **Internas**: `internal/api`, `internal/services`, `internal/storage`, `internal/models`, `internal/db`.
- **Externas**: `modernc.org/sqlite` (ya en uso), `golang.org/x/crypto` (ya en uso).
- **Nuevas**: Ninguna.

---

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] **CA-001**: Dado un usuario autenticado, cuando accede al dashboard, entonces ve sidebar, header y área de contenido vacía o con página por defecto.
- [ ] **CA-002**: Dado el sidebar, cuando hace clic en un submenú, entonces se carga la página correspondiente en el área principal sin recargar el navegador.
- [ ] **CA-003**: Dado el header, cuando hace clic en el engranaje, entonces navega a la página de configuraciones.
- [ ] **CA-004**: Dado la página de configuraciones, cuando cambia el idioma a inglés, entonces toda la UI se traduce inmediatamente.
- [ ] **CA-005**: Dado la sección de monedas en configuraciones, cuando agrega una moneda, entonces aparece disponible al crear servicios.
- [ ] **CA-006**: Dado que no existe ningún Home/Casa, cuando intenta crear un servicio, entonces el sistema bloquea la acción y muestra mensaje traducido.
- [ ] **CA-007**: Dado un Home/Casa creado, cuando crea un servicio mensual activo, entonces se genera automáticamente una factura pendiente para el periodo actual.
- [ ] **CA-008**: Dado un servicio anual, cuando llega la fecha correspondiente, entonces se genera la factura anual pendiente.
- [ ] **CA-009**: Dado una factura pendiente, cuando edita el monto, entonces se guarda el nuevo monto sin cambiar el estado.
- [ ] **CA-010**: Dado una factura pendiente, cuando intenta marcarla como pagada sin link de Drive, entonces se rechaza con error traducido.
- [ ] **CA-011**: Dado una factura pendiente, cuando ingresa un link de Drive inválido e intenta pagarla, entonces se rechaza con error de validación.
- [ ] **CA-012**: Dado una factura pendiente, cuando ingresa un link de Drive válido y marca pagada, entonces el estado cambia a `paid`.
- [ ] **CA-013**: Dado el listado de servicios, cuando selecciona un Home del filtro, entonces solo se muestran servicios de ese Home.
- [ ] **CA-014**: Dado un servicio eliminado, cuando se consulta el listado, entonces no aparece (soft delete).
- [ ] **CA-015**: Dado una sección sin registros, cuando se muestra el empty state, entonces aparece como una card centrada con título, subtítulo y botón de acción.
- [ ] **CA-016**: Dado un empty state, cuando se visualiza el botón de creación, entonces tiene estilo de card atractivo (no fondo gris plano).
- [ ] **CA-017**: Dado cualquier botón de acción, cuando se visualiza, entonces incluye un icono representativo de su función.
- [ ] **CA-018**: Dado una página de listado, cuando se visualiza el encabezado, entonces la acción de crear se muestra como un icono de tres puntos con dropdown personalizado.
- [ ] **CA-019**: Dado una card, cuando se hace clic en los tres puntos superiores derechos, entonces se despliega un dropdown con las opciones Editar y Eliminar con sus iconos.
- [ ] **CA-020**: Dado que se selecciona eliminar, cuando se abre el modal de confirmación, entonces el botón de eliminar permanece deshabilitado hasta que el usuario escriba "confirmo".
- [ ] **CA-021**: Dado que se navega a una página, cuando se observa la URL, entonces refleja la página actual y al refrescar se mantiene la misma página.
- [ ] **CA-022**: Dado cualquier selector o menú desplegable, cuando se interactúa con él, entonces es un componente custom de CSS y no el dropdown nativo del OS.
- [ ] **CA-023**: Dado que se crea o edita un registro, cuando se muestra el formulario, entonces se renderiza como una página completa y no como un modal.

### 5.2 No funcionales

- [ ] **CA-NF-001**: El frontend no debe depender de frameworks SPA ni librerías de iconos externas.
- [ ] **CA-NF-002**: El backend no debe agregar ORM ni librerías de query building.
- [ ] **CA-NF-003**: El tiempo de arranque del servidor Go no debe aumentar más de 500 ms tras las mejoras de migraciones.
- [ ] **CA-NF-004**: El consumo de memoria RAM no debe superar los 100 MB en uso normal.

### 5.3 Testing

- **Unit tests**: Generación automática de facturas, validación de URL de Drive, filtros de servicios.
- **Integration tests**: Flujo completo Home → Servicio → Factura → Pago.
- **E2E manual**: Cambio de idioma, navegación sidebar, CRUD completo en UI.
- **Performance**: Medir tiempo de arranque y memoria en el entorno de desarrollo local.

---

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Mejorar tracker de migraciones (`schema_migrations`) | 1 día | Ninguna |
| 2 | Migraciones DB: currencies, homes, services, bills | 1 día | Fase 1 |
| 3 | Backend: settings, currencies, homes CRUD | 2 días | Fase 2 |
| 4 | Backend: services CRUD + validación de Home | 2 días | Fase 3 |
| 5 | Backend: bills CRUD + generación automática + validación Drive | 3 días | Fase 4 |
| 6 | Frontend: layout sidebar/header + routing + iconos | 2 días | Ninguna |
| 7 | Frontend: i18n + settings estilo iOS | 2 días | Fase 6 |
| 8 | Frontend: CRUD Home/Casa | 1 día | Fase 7 |
| 9 | Frontend: CRUD servicios con filtro por Home + iconos | 3 días | Fase 8 |
| 10 | Frontend: CRUD facturas + estados + Drive | 2 días | Fase 9 |
| 11 | Tests locales, ajustes y validación en iHost | 2 días | Fase 10 |

### 6.2 Milestones

1. **MVP**: Migraciones mejoradas, layout dashboard, i18n, CRUD Home/Casa.
2. **V1.0**: Servicios con iconos, filtro por Home, facturas automáticas, validación Drive, settings estilo iOS.

---

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| UI compleja en vanilla JS difícil de mantener | Media | Medio | Componentizar funciones de renderizado, mantener estado centralizado simple. |
| Generación automática de facturas con fechas incorrectas | Media | Alto | Cubrir con tests unitarios para mensual y anual; usar zona horaria local. |
| Validación de Google Drive muy permisiva o restrictiva | Baja | Medio | Regex simple pero funcional; documentar formato esperado. |
| Cambio de idioma no aplica a contenido dinámico | Baja | Medio | Refrescar textos al cargar cada página; usar data attributes. |
| Migraciones con `schema_migrations` en instancias existentes | Baja | Alto | La primera migración crea la tabla y registra migraciones previas si aplica. |

---

## 8. Notas y Referencias

- Reglas del proyecto: `docs/project-rules.md`
- Especificaciones anteriores: `SPEC-001`, `SPEC-002`, `SPEC-003`
- Template de specs: `docs/specs/templates/spec-template.md`
- Patrón de almacenamiento actual: `internal/storage/user.go`

### 8.1 Lecciones aprendidas (UI/Frontend)

| Problema | Causa | Solución |
|----------|-------|----------|
| Empty state no centrado en grid | `margin: auto` no funciona dentro de CSS grid | Usar `place-self: center` + `grid-column: 1 / -1` |
| Empty state sin fondo blanco | Clase `.empty-card` no aplicada cuando `inline: true` | Eliminar `inline`; empty state siempre es card completa |
| Formulario sin contenedor card | `.form-page` solo tenía `max-width` y `margin` | Agregar `background`, `border-radius`, `box-shadow`, `padding` |
| Menú de crear no funcionaba | `renderCreateMenu()` generaba HTML pero nunca se llamaba `attachCreateMenu()` | Siempre llamar `attachCreateMenu(options)` después de `setHeaderActions()` |
| Dropdown de card se cortaba | Posicionamiento `absolute` dentro de card con `overflow` | Usar `position: fixed` calculado desde `getBoundingClientRect()` |
| Subtítulo duplicado en empty state | Se pasaba la misma key para título y subtítulo | Usar keys separadas: `titleKey` + `subtitleKey` |
| Modal de eliminar texto redundante | `<br>${I18n.get('app.confirm')}` duplicaba el título | Dejar solo `subtitle` descriptivo del elemento a eliminar |
| AuthMiddleware rompía rutas SPA | Middleware devolvía 401 texto plano en vez de redirigir | Quitar middleware de rutas SPA; `dashboardPageHandler` ya valida auth |
| Caché de navegador ocultaba cambios | Hard refresh necesario tras cambios JS/CSS | Documentar: siempre hard refresh (`Ctrl+Shift+R`) al probar |

---

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-12 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-08-12 | p40la-ihost-team | Mover a `in_progress` para iniciar desarrollo |
| 2026-08-13 | p40la-ihost-team | Solicitud de cambios: empty state como card centrada; menús de 3 puntos; confirmación de eliminación; iconos en botones; navegación por URL; dropdowns custom; formularios como páginas |
| 2026-08-13 | p40la-ihost-team | Correcciones UI: centrado de empty cards, fondo blanco en formularios, posicionamiento de dropdowns, lecciones documentadas |
| 2026-08-13 | p40la-ihost-team | Spec cerrada y marcada como released |
