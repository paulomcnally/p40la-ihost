---
title: "Autos: campos opcionales y color predefinido"
id: "SPEC-118"
status: "in_progress"
author: "opencode"
created: "2026-10-21"
updated: "2026-10-21"
github_issue: 121
---

# Autos: campos opcionales y color predefinido

**ID**: SPEC-118  
**Estado**: in_progress  
**Autor**: opencode  
**Creado**: 2026-10-21  
**Actualizado**: 2026-10-21

---

## 1. Resumen Ejecutivo

El formulario de autos (`/autos/new` y `/autos/edit/:id`) exige hoy siete datos: año, marca, modelo, color, motor, chasis, VIN y placa. En la práctica no todos los vehículos tienen (o el usuario conoce) motor, chasis, VIN y placa; la validación actual bloquea el alta y la edición si cualquiera de esos cuatro campos queda vacío, tanto en el frontend (`required`) como en el backend (`AutoService.Create/Update`). Además, el color es un `input` de texto libre, lo que genera valores inconsistentes ("Blanco", "blanco", "BLANCO", "Blancos") y hace imposible filtrar o mostrar el dato de forma uniforme.

Esta spec hace opcionales los campos **motor**, **chasis**, **VIN** y **placa** en toda la cadena (modelo → storage → service → handler → formulario), y convierte el **color** en un `select` con 16 colores predefinidos. El color se guarda en la DB como **código canónico** (`white`, `black`, `red`, etc.) y la etiqueta visible sale de i18n (`es`/`en`). Una migración de datos normaliza los colores existentes: los nombres conocidos (en español o inglés, con o sin acentos/mayúsculas) se mapean a su código; los valores no reconocidos se dejan vacíos para que el usuario elija uno al editar. Producción tiene hoy solo 3 autos, por lo que el impacto del reseleccionado es mínimo.

Impacto en iHost: nulo en runtime. No se agregan dependencias, no hay cambios de esquema (la migración solo actualiza datos), no se agregan endpoints ni consultas nuevas y la memoria se mantiene constante. El bundle aumenta solo por el listado de 16 colores y las claves i18n.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: En la creación de un auto (`POST /api/autos`) los campos `motor`, `chasis`, `vin` y `placa` DEBEN ser opcionales. Si vienen ausentes o vacíos, se guardan como cadena vacía (`''`) sin devolver error.
2. **REQ-002**: En la edición de un auto (`PUT /api/autos/:id`) los mismos cuatro campos DEBEN ser opcionales y DEBEN poder limpiarse enviando `''` o `null`-equivalente vacío. Lo enviado se persiste tal cual (con `TrimSpace`).
3. **REQ-003**: `year`, `brand`, `model` y `color` siguen siendo obligatorios. El backend DEBE devolver `400 invalid_request` con un mensaje claro si falta alguno.
4. **REQ-004**: `color` DEBE ser un código de la paleta canónica de 16 colores (sección 4.3). El backend DEBE validar el valor contra la whitelist y rechazar con `400` cualquier valor fuera de la paleta. La comparación DEBE ser tolerante a mayúsculas/espacios (`TrimSpace` + `ToLower`), almacenando siempre el código en minúsculas.
5. **REQ-005**: El formulario de autos DEBE reemplazar el `input` de color por el **componente reutilizable `frontend/src/components/Select.tsx`** (dropdown custom), con las 16 opciones traducidas según el idioma activo (es/en) y un placeholder no seleccionable ("Seleccioná un color") cuando el valor esté vacío. **PROHIBIDO** usar `<select>` nativo: muestra el dropdown del sistema operativo y rompe el patrón de UI (SPEC-004 CA-022, SPEC-111 CA-018).
6. **REQ-018**: El `Select` de color DEBE ser **searchable**: al abrirse muestra un buscador con autofocus que filtra por etiqueta, sin distinguir mayúsculas ni acentos (ej. "marron" encuentra "Marrón"). El placeholder del buscador y el estado sin resultados DEBEN salir de i18n (`app.search`, `app.no_results`).
6. **REQ-006**: En el formulario, los campos `motor`, `chasis`, `VIN` y `placa` DEBEN dejar de tener el atributo `required` y DEBEN rotularse como opcionales (sufijo "(opcional)" en es, "(optional)" en en). El submit NO DEBE bloquearse si están vacíos.
7. **REQ-007**: El listado (`AutosPage`) y el detalle (`AutoShowPage`) DEBEN tolerar valores vacíos: no mostrar separadores colgantes (`2020 ·  · `), no renderizar filas vacías de motor/chasis/VIN y mostrar el color traducido, no el código.
8. **REQ-008**: Las alertas que usan la placa (Telegram `telegram_alerts.go` y email de `alert_scheduler.go`) DEBEN degradar sin artefactos cuando `placa` esté vacía: en Telegram se omite el sufijo "— <placa>"; en el email se muestra `—` en la celda.
9. **REQ-009**: DEBE existir la migración de datos `migrations/0040_normalize_auto_colors.up.sql` que normalice los colores existentes:
   - `TrimSpace` + `lower()` sobre el valor actual.
   - Mapeo de nombres conocidos en español e inglés (con y sin acentos, con y sin mayúsculas) al código canónico.
   - Valores no reconocidos → `''` (cadena vacía).
   - Los valores que ya son códigos canónicos no se tocan (migración idempotente).
   - Debe incluir `0040_normalize_auto_colors.down.sql` con el mapeo inverso best-effort (código → etiqueta en español), documentando que los valores limpiados no son recuperables.
10. **REQ-010**: DEBE existir un test de migración (`internal/db/migrate_040_test.go`) que verifique: mapeo de "Blanco"→`white`, "GRIS"→`gray`, "Marrón"→`brown`, "Rojo "→`red`, código canónico intacto y valor desconocido→`''`.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-011**: Las etiquetas de los 16 colores y las cadenas nuevas del formulario (`autos.color_placeholder`, `autos.optional`, etc.) DEBEN vivir en `frontend/public/i18n/{es,en}.json` bajo el namespace `autos`. Nunca hardcodear etiquetas en el componente.
2. **REQ-012**: La lista canónica de colores del frontend DEBE vivir en un único módulo (`frontend/src/constants/autoColors.ts`) y la del backend en un único punto (`internal/services/auto.go`). Ambas listas DEBEN coincidir exactamente; se documenta en la spec como contrato espejo.
3. **REQ-013**: El `Select` de color DEBE respetar la regla de UI: fondo `bg-card`, texto `text-text`, placeholder `text-text-secondary`, borde `border-border` y altura mínima 44 px. Prohibido `bg-white`, `text-black` o colores hardcodeados (SPEC-060).
4. **REQ-016**: Se DEBE migrar el último `<select>` nativo del frontend (`UploadBillModal`, selector de mes) al componente `Select`, para que la regla no tenga excepciones.
5. **REQ-017**: Se DEBE agregar un guardrail automático (`frontend/scripts/check-ui-rules.cjs`) que falle el build si aparece un `<select>` nativo en `frontend/src`, y documentar la regla en `AGENTS.md` (`docs/project-rules.md` §4.6) y en la skill `spec-manager` para que toda spec nueva que toque formularios exija `Select`.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-014**: Migrar el resto de las etiquetas hardcodeadas de `AutoFormPage` ("Año", "Marca", "Modelo", "Ícono", "Cancelar", "Guardar", etc.) a claves i18n, sin cambiar el comportamiento.
2. **REQ-015**: Mostrar un estado vacío explícito ("Sin especificar") en el detalle para los campos opcionales que estén vacíos, en lugar de ocultarlos, si el usuario prefiere visibilidad total.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: sin consultas ni endpoints nuevos. La migración procesa 3 filas en producción; costo despreciable (< 100 ms).
- **Seguridad**: sin cambios de autenticación. Los endpoints siguen detrás del `AuthMiddleware`. No se agregan datos sensibles.
- **Almacenamiento**: sin cambios de esquema. Solo `UPDATE` de la columna `autos.color`.
- **Disponibilidad**: la migración corre al arrancar la app (`db.OpenDB` → `Migrate`), dentro de una transacción, y queda registrada en `schema_migrations`.
- **iHost**: cero dependencias nuevas, cero procesos nuevos, uso de memoria constante. El `select` nativo no incorpora librerías de UI.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- Estado actual del formulario: `frontend/src/pages/AutoFormPage.tsx` — 8 campos de texto, todos con `required`, color como `input type="text"`.
- Validación backend: `internal/services/auto.go` exige `motor`, `chasis`, `vin` y `placa` no vacíos en `Create` y `Update`.
- Persistencia: `internal/storage/auto.go` y `internal/api/auto_handlers.go` usan `strings` planos; no hay validación de color.
- Esquema: `migrations/0009_create_autos.up.sql` (`color TEXT NOT NULL`) y `migrations/0012_add_auto_and_insurance_fields.up.sql` (los cuatro campos con `DEFAULT ''`). No hace falta cambiar el esquema: las columnas ya aceptan cadena vacía.
- Consumo del color: solo `AutosPage` y `AutoShowPage` lo muestran como texto; `models.AutoServiceDetail` incluye `color` pero el frontend no lo renderiza hoy. Las alertas solo usan `placa`.
- Migraciones: `internal/db/db.go` aplica los `.up.sql` ordenados por nombre y los registra en `schema_migrations`; la última es `0039_merge_automation`.
- Reglas de UI aplicables: SPEC-060 (tokens de tema en inputs), SPEC-063 (flecha atrás en detalle, ya implementada), SPEC-115 (etiqueta "Editar", ya implementada).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A. Colores como códigos canónicos en un `select` + whitelist backend | Integridad de datos, i18n limpio, sin dependencias | Requiere migración de datos y mantener la lista espejo | ✅ Seleccionada |
| B. `input` con `datalist` de sugerencias | No rompe datos existentes, menos código | Sigue permitiendo texto libre; no garantiza integridad | ❌ Rechazada |
| C. Tabla `auto_colors` con FK | Modelo relacional "correcto", editable sin deploy | Cambio de esquema, join extra, más código y RAM en iHost | ❌ Rechazada |
| D. Guardar la etiqueta en español como valor canónico | Cero traducción en DB | Rompe i18n (la UI en inglés mostraría español) y mezcla presentación con dato | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: El color se persiste como código canónico en inglés (`white`, `black`, …), no como etiqueta.
- **Contexto**: la app es bilingüe y el color es un dato de dominio, no una etiqueta de UI.
- **Decisión**: almacenar códigos estables y traducir en el frontend con `autos.colors.<code>`.
- **Consecuencias**: los valores existentes deben migrarse; cualquier consumidor nuevo debe traducir el código. A cambio, filtros y consistencia futura quedan garantizados.

**ADR-002**: La paleta vive duplicada (Go whitelist + TS constant) en lugar de exponerse por API.
- **Contexto**: iHost prioriza mínima latencia y mínimas dependencias; agregar `GET /api/autos/colors` implica un request extra y estado de carga en el formulario.
- **Decisión**: listas espejo en `internal/services/auto.go` y `frontend/src/constants/autoColors.ts`, con la tabla de la sección 4.3 como fuente de verdad documental.
- **Consecuencias**: un cambio de paleta requiere tocar ambos archivos y la migración si se renombra un código; se mitiga con revisión en el checklist de la spec.

**ADR-003**: La migración normaliza y limpia valores desconocidos (decisión del usuario).
- **Contexto**: producción tiene 3 autos con color de texto libre; no se justifica un mapeo exhaustivo ni conservar valores fuera de la paleta.
- **Decisión**: mapear los nombres reconocidos y dejar `''` el resto, para que el usuario re-elija al editar.
- **Consecuencias**: hasta que se editen esos autos, el color puede verse vacío. Se mitiga con el fallback de display de REQ-007 y con backup previo obligatorio.

**ADR-004**: Los cuatro campos pasan a opcionales sin cambiar el esquema ni el contrato JSON.
- **Contexto**: las columnas ya son `TEXT NOT NULL DEFAULT ''`.
- **Decisión**: solo se relajan las validaciones de servicio y el `required` del formulario; la API mantiene los mismos campos.
- **Consecuencias**: clientes viejos siguen funcionando; no hay migración estructural.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[AutoFormPage.tsx]
   |  color = código canónico; motor/chasis/vin/placa opcionales
   v
[api/auto_handlers.go] --(JSON)--> [services/auto.go: valida color y opcionales]
                                        |
                                        v
                                 [storage/auto.go]
                                        |
                                        v
                                  [SQLite autos]
                                        ^
                                        |
                    [migrations/0040_normalize_auto_colors]
```

### 4.2 Componentes

#### 4.2.1 `internal/services/auto.go`
- **Responsabilidad**: quitar las validaciones de obligatoriedad de `motor`, `chasis`, `VIN` y `placa`; normalizar y validar `color` contra la whitelist.
- **Interfaz**: `Create(ctx, year, model, brand, color, icon, motor, chasis, vin, placa)` y `Update(...)` sin cambios de firma (los argumentos siguen existiendo, ahora pueden venir vacíos).
- **Dependencias**: `models`, `storage`, `strings`.
- **Ubicación**: `internal/services/auto.go`.

#### 4.2.2 `internal/storage/auto.go`
- **Responsabilidad**: sin cambios funcionales; sigue persistiendo cadenas vacías.
- **Interfaz**: sin cambios.
- **Ubicación**: `internal/storage/auto.go`.

#### 4.2.3 `internal/api/auto_handlers.go`
- **Responsabilidad**: sin cambios de contrato. Propaga los campos opcionales y el error de color inválido como `400`.
- **Ubicación**: `internal/api/auto_handlers.go`.

#### 4.2.4 `frontend/src/constants/autoColors.ts` (nuevo)
- **Responsabilidad**: exportar `AUTO_COLORS: string[]` con los 16 códigos en el orden de la tabla 4.3.
- **Dependencias**: ninguna.
- **Ubicación**: `frontend/src/constants/autoColors.ts`.

#### 4.2.5 `frontend/src/pages/AutoFormPage.tsx`
- **Responsabilidad**: selector de color con el componente custom `Select` searchable, campos opcionales sin `required`, labels i18n, validación de color vacío con toast y envío del código canónico.
- **Interfaz**: usa `api.autos.create/update` sin cambios.
- **Ubicación**: `frontend/src/pages/AutoFormPage.tsx`.

#### 4.2.6 `frontend/src/utils/auto.ts` (nuevo)
- **Responsabilidad**: helpers de presentación:
  - `autoColorLabel(t, code)`: devuelve la etiqueta traducida o `—` si está vacío/desconocido.
  - `autoSubtitle(auto)`: arma `year · colorLabel · placa` omitiendo partes vacías.
- **Ubicación**: `frontend/src/utils/auto.ts`.

#### 4.2.7 Alertas (`internal/services/telegram_alerts.go`, `internal/services/alert_scheduler.go`)
- **Responsabilidad**: omitir la placa vacía en Telegram y mostrar `—` en la columna Placa del email.
- **Ubicación**: ambos archivos.

#### 4.2.8 Migración `0040`
- **Responsabilidad**: normalizar `autos.color` una sola vez.
- **Ubicación**: `migrations/0040_normalize_auto_colors.up.sql` y `.down.sql`.

#### 4.2.9 `frontend/scripts/check-ui-rules.cjs` (guardrail)
- **Responsabilidad**: recorrer `frontend/src` y fallar si encuentra `<select>` nativo. Se ejecuta al inicio de `npm run build` y con `npm run check:ui`.
- **Ubicación**: `frontend/scripts/check-ui-rules.cjs` + `frontend/package.json`.

### 4.3 Modelo de datos

```
Entidad: Auto (tabla autos, sin cambios de esquema)
- id: INTEGER PK
- year: INTEGER NOT NULL
- model: TEXT NOT NULL
- brand: TEXT NOT NULL
- color: TEXT NOT NULL  -> código canónico de la paleta o '' (legado sin normalizar)
- icon: TEXT NOT NULL DEFAULT 'vehicle'
- motor: TEXT NOT NULL DEFAULT ''   -> ahora opcional
- chasis: TEXT NOT NULL DEFAULT ''  -> ahora opcional
- vin: TEXT NOT NULL DEFAULT ''     -> ahora opcional
- placa: TEXT NOT NULL DEFAULT ''   -> ahora opcional
- created_at / updated_at: DATETIME
```

Paleta canónica (fuente de verdad):

| # | Código | es | en |
|---|--------|----|----|
| 1 | `white` | Blanco | White |
| 2 | `black` | Negro | Black |
| 3 | `gray` | Gris | Gray |
| 4 | `silver` | Plata | Silver |
| 5 | `red` | Rojo | Red |
| 6 | `blue` | Azul | Blue |
| 7 | `green` | Verde | Green |
| 8 | `yellow` | Amarillo | Yellow |
| 9 | `orange` | Naranja | Orange |
| 10 | `brown` | Marrón | Brown |
| 11 | `beige` | Beige | Beige |
| 12 | `gold` | Dorado | Gold |
| 13 | `burgundy` | Vino | Burgundy |
| 14 | `light_blue` | Celeste | Light blue |
| 15 | `pink` | Rosado | Pink |
| 16 | `purple` | Morado | Purple |

Migración (borrador del `up.sql`):

```sql
UPDATE autos
SET color = CASE lower(trim(color))
    WHEN 'blanco'  THEN 'white'
    WHEN 'white'   THEN 'white'
    WHEN 'negro'   THEN 'black'
    WHEN 'black'   THEN 'black'
    WHEN 'gris'    THEN 'gray'
    WHEN 'gray'    THEN 'gray'
    WHEN 'grey'    THEN 'gray'
    WHEN 'plata'   THEN 'silver'
    WHEN 'plateado' THEN 'silver'
    WHEN 'silver'  THEN 'silver'
    WHEN 'rojo'    THEN 'red'
    WHEN 'red'     THEN 'red'
    WHEN 'azul'    THEN 'blue'
    WHEN 'blue'    THEN 'blue'
    WHEN 'verde'   THEN 'green'
    WHEN 'green'   THEN 'green'
    WHEN 'amarillo' THEN 'yellow'
    WHEN 'yellow'  THEN 'yellow'
    WHEN 'naranja' THEN 'orange'
    WHEN 'orange'  THEN 'orange'
    WHEN 'marrón'  THEN 'brown'
    WHEN 'marrÓn'  THEN 'brown'
    WHEN 'marron'  THEN 'brown'
    WHEN 'brown'   THEN 'brown'
    WHEN 'beige'   THEN 'beige'
    WHEN 'dorado'  THEN 'gold'
    WHEN 'gold'    THEN 'gold'
    WHEN 'vino'    THEN 'burgundy'
    WHEN 'burgundy' THEN 'burgundy'
    WHEN 'celeste' THEN 'light_blue'
    WHEN 'light blue' THEN 'light_blue'
    WHEN 'light-blue' THEN 'light_blue'
    WHEN 'light_blue' THEN 'light_blue'
    WHEN 'rosado'  THEN 'pink'
    WHEN 'pink'    THEN 'pink'
    WHEN 'morado'  THEN 'purple'
    WHEN 'purple'  THEN 'purple'
    ELSE ''
END
WHERE trim(color) <> ''
  AND color NOT IN (
    'white','black','gray','silver','red','blue','green','yellow',
    'orange','brown','beige','gold','burgundy','light_blue','pink','purple'
  );
```

Notas de implementación:
- SQLite `lower()` no convierte caracteres no ASCII: por eso se listan `'marrón'` y `'marrÓn'` por separado. Validar en la migración real contra la copia de producción.
- El `WHERE` excluye solo los códigos canónicos exactos (minúsculas), garantizando idempotencia y permitiendo normalizar variantes como `White` o ` white `.
- El `down.sql` revierte códigos a etiquetas en español (`white`→`Blanco`, …) y documenta que los valores limpiados (`''`) no se recuperan; el rollback real es el backup previo.

### 4.4 APIs / Contratos

#### Endpoint: `POST /api/autos`

**Request** (solo `brand`, `model`, `color` y `year` obligatorios):
```json
{
  "year": 2026,
  "brand": "Toyota",
  "model": "Corolla",
  "color": "white",
  "icon": "vehicle",
  "motor": "",
  "chasis": "",
  "vin": "",
  "placa": ""
}
```

**Response 201**:
```json
{
  "id": 1,
  "year": 2026,
  "brand": "Toyota",
  "model": "Corolla",
  "color": "white",
  "icon": "vehicle",
  "motor": "",
  "chasis": "",
  "vin": "",
  "placa": ""
}
```

**Response Error**:
```json
{
  "error": "invalid_request",
  "message": "el color no es válido"
}
```

#### Endpoint: `PUT /api/autos/:id`

Mismo contrato que `POST`; permite limpiar los cuatro campos opcionales enviando `""`.

Sin cambios de rutas, middleware ni códigos de error existentes.

### 4.5 Dependencias

- **Internas**: `internal/services/auto.go`, `internal/api/auto_handlers.go` (sin cambios de contrato), `internal/storage/auto.go` (sin cambios), `internal/services/telegram_alerts.go`, `internal/services/alert_scheduler.go`, `frontend/src/pages/AutoFormPage.tsx`, `frontend/src/pages/AutosPage.tsx`, `frontend/src/pages/AutoShowPage.tsx`, `frontend/src/types/index.ts` (sin cambios), i18n es/en, `docs/specs/README.md` (solo en el release).
- **Externas**: ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado el formulario de alta, cuando se completa solo año, marca, modelo y color, entonces el auto se crea con `motor`, `chasis`, `vin` y `placa` vacíos y la UI no bloquea el submit.
- [ ] CA-002: Dado un auto existente con datos en los cuatro campos, cuando se edita y se vacían, entonces se persisten como `''` y el detalle no muestra filas vacías.
- [ ] CA-003: Dado el formulario, cuando se abre el color, entonces muestra exactamente las 16 opciones de la tabla 4.3 traducidas al idioma activo, usando el dropdown custom `Select` (no el `<select>` nativo del sistema operativo) con buscador con autofocus.
- [ ] CA-012: Dado el buscador del color, cuando se escribe "marron" o "MARRÓN", entonces filtra las opciones a "Marrón" (búsqueda sin acentos ni mayúsculas).
- [ ] CA-011: `frontend/scripts/check-ui-rules.cjs` pasa sin offenders y el build falla si se agrega un `<select>` nativo.
- [ ] CA-004: Dado un request `POST /api/autos` con `color: "blanco"` o cualquier valor fuera de la paleta, entonces responde `400 invalid_request` con "el color no es válido".
- [ ] CA-005: Dado un request con `color: "WHITE"`, entonces se guarda `white` (normalización a minúsculas y trim).
- [ ] CA-006: Dado un request sin `brand` o sin `model` o sin `year`, entonces responde `400` con el mensaje de campo requerido correspondiente.
- [ ] CA-007: Dado el listado y el detalle, cuando un auto tiene placa vacía, entonces no aparecen separadores `·` colgantes ni celdas vacías; el color se muestra traducido.
- [ ] CA-008: Dada una alerta de Telegram/email de un auto sin placa, entonces el mensaje se genera sin "— " huérfano y con `—` en la celda del email.
- [ ] CA-009: Dada la migración `0040` aplicada sobre una copia de la DB de producción, entonces "Blanco"→`white`, "GRIS"→`gray`, "Marrón"→`brown`, códigos ya canónicos intactos y valores desconocidos→`''`.
- [ ] CA-010: Dada la migración aplicada dos veces, entonces el segundo pase no cambia ningún valor (idempotencia).
- [ ] CA-DARK: El formulario usa tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y se verificó legibilidad de texto y placeholder del `select` en darkmode.
- [ ] CA-BACK: La spec no crea páginas nuevas; los formularios existentes ya usan la flecha atrás del header vía `BACK_ROUTES`. No se agregan links "← Título".

### 5.2 No funcionales

- [ ] CA-NF-001: `go test ./...` pasa, incluido el nuevo test de migración y los tests de servicio de auto.
- [ ] CA-NF-002: `cd frontend && npm run build` compila sin errores de TypeScript y las claves i18n nuevas se sirven desde `frontend/public/i18n/`.
- [ ] CA-NF-003: La migración se aplica en menos de 100 ms sobre la copia de producción y no altera filas fuera de `autos.color`.
- [ ] CA-NF-004: No se agregan dependencias a `go.mod` ni a `frontend/package.json`.

### 5.3 Testing

- **Unit tests**:
  - `internal/services/auto_test.go`: crear/actualizar con los cuatro campos vacíos; color válido, inválido y con mayúsculas; campos obligatorios restantes.
  - `internal/db/migrate_040_test.go`: casos de mapeo, idempotencia y valor desconocido.
- **Integration tests**:
  - `internal/api/auto_handlers_test.go`: `POST`/`PUT` con opcionales vacíos y color inválido → 201/200 y 400.
- **E2E/manual**:
  - Alta y edición de un auto con solo los campos obligatorios en local.
  - Verificación en darkmode del `select` y del formulario.
  - Verificación del listado/detalle con placa y color vacíos.
- **Carga/Performance**: no aplica; validar que la migración corra dentro del arranque normal del server en el iHost.

### 5.4 Resultados de validación (implementación)

- `go test ./...`: todas las suites pasan.
- `npm run build` (frontend): compila sin errores; i18n sincronizado a `src/i18n` y servido en `/i18n/es.json`.
- Migración validada sobre una **copia de producción** (backup `.backup`, `PRAGMA integrity_check` = `ok`):
  - 3 autos existentes: `Blanco`→`white` (2), `Negro`→`black` (1). Ningún valor quedó vacío.
  - Segundo pase de la migración: sin cambios (idempotencia).
  - `down.sql`: revierte a `Blanco`/`Negro`.
- API verificada en local contra la copia: alta mínima (`201`, color normalizado a `white`, opcionales vacíos), color inválido (`400 invalid_request`), edición limpiando los cuatro campos (`200`).
- La copia de producción y los archivos temporales de validación viven solo en `/tmp/opencode` y se eliminan al terminar.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración `0040` + test de migración, validada contra copia de producción | 0.5 día | Backup de producción |
| 2 | Backend: relajar validaciones, whitelist de color, tests de service y handler | 0.5 día | Fase 1 |
| 3 | Frontend: constantes, `select`, campos opcionales, i18n, helpers de display | 1 día | Fase 2 |
| 4 | Alertas: fallback de placa vacía (Telegram/email) + tests | 0.25 día | Fase 2 |
| 5 | QA local: build, tests, darkmode, migración sobre copia de prod y prueba manual | 0.5 día | Fases 1-4 |
| 6 | Guardrail anti `<select>` nativo + migración de `UploadBillModal` + reglas en AGENTS/project-rules/skill | 0.25 día | Fase 3 |

### 6.2 Milestones

1. **MVP**: backend con opcionales y color validado + migración de datos + formulario con `select`, sin retoques de display de alertas.
2. **V1.0**: display tolerante a vacíos en listado/detalle/alertas, i18n completa de las cadenas nuevas y tests.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Valores de color en producción distintos a los previstos | Media | Medio | Inspeccionar la copia de producción antes del release; el `CASE` cubre variantes y lo desconocido se limpia (decisión del usuario) |
| Pérdida irreversible del color limpiado | Baja | Bajo | Backup obligatorio antes de migrar; solo 3 filas; el usuario re-elige color al editar |
| Listas de colores desincronizadas (Go vs TS) | Media | Medio | Tabla 4.3 como fuente de verdad; revisión cruzada en el checklist y test de servicio que enumera los 16 códigos |
| Un consumidor externo envía color como etiqueta ("Blanco") | Baja | Bajo | La API responde 400 con mensaje claro; el único cliente es el frontend de este repo |
| `select` nativo con estilos inconsistentes por plataforma | Media | Bajo | Tokens de tema, `min-h-[44px]` y verificación en darkmode/móvil |
| Alertas con placa vacía generan mensajes feos | Media | Bajo | REQ-008 con fallback explícito y tests en `telegram_alerts_test.go` / `alert_scheduler_test.go` |

## 8. Notas y Referencias

- Formulario actual: `frontend/src/pages/AutoFormPage.tsx`.
- Validación actual: `internal/services/auto.go`.
- Esquema: `migrations/0009_create_autos.up.sql`, `migrations/0012_add_auto_and_insurance_fields.up.sql`.
- Migraciones: `internal/db/db.go` (`OpenDB` → `Migrate`).
- Reglas de UI: SPEC-060 (tokens de tema), SPEC-063 (flecha atrás), SPEC-115 (etiqueta "Editar").
- Dropdowns: `frontend/src/components/Select.tsx`; prohibido `<select>` nativo (SPEC-004 CA-022, SPEC-111 CA-018, SPEC-118). Guardrail: `frontend/scripts/check-ui-rules.cjs`.
- Acceso a la DB de producción: `docs/ssh-ihost-access.md`.
- Política de datos: `docs/project-rules.md` §6.1 y §8 (validar contra copia de producción; nunca datos reales en el repo).
- Nota de proceso: el README tracker (`docs/specs/README.md`) se actualiza recién en el release (SPEC-066), no al crear la spec.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-10-21 | opencode | Creación inicial de la especificación |
| 2026-10-21 | opencode | Inicio de implementación (estado `in_progress`) |
| 2026-10-21 | opencode | Implementación completa: migración 0040, validación de color en backend, formulario con `select` i18n, helpers de display, fallback de alertas y tests. Validada contra copia de producción. |
| 2026-10-21 | opencode | Corrección: el selector de color usa el componente custom `Select` (no `<select>` nativo del OS), se migra el último select nativo y se agrega guardrail de build + reglas en AGENTS/project-rules/skill. |
| 2026-10-21 | opencode | El `Select` de color incorpora buscador con autofocus, filtrado sin mayúsculas ni acentos y textos i18n (`app.search`, `app.no_results`). |
