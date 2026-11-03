---
title: "Importador mensual de gastos escolares y pensión por palabras clave"
id: "SPEC-119"
status: "draft"
author: "opencode"
created: "2026-11-03"
updated: "2026-11-03"
github_issue: 122
---

# Importador mensual de gastos escolares y pensión por palabras clave

**ID**: SPEC-119  
**Estado**: draft  
**Autor**: opencode  
**Creado**: 2026-11-03  
**Actualizado**: 2026-11-03

---

## 1. Resumen Ejecutivo

Cada mes llega un comprobante en PDF con los gastos escolares y la pensión del mes (conceptos como `PENSION`, `COLEGIATURA`, `COMBUSTIBLE` y montos). Hoy esos valores se cargan a mano, uno por uno, en **Pensión → Registros**, repitiendo categorías que ya se usaron el mes anterior. El proceso es lento y propenso a errores de tipeo o de categoría.

Esta spec define un **importador mensual** que recibe el PDF (con capa de texto), un archivo de texto o un CSV, extrae las líneas `concepto + monto`, detecta el período del encabezado y propone un **match contra las categorías de pensión existentes**. El usuario ve un preview editable antes de confirmar: puede ajustar categoría, hijo, monto, período y estado, o **crear una categoría nueva** cuando no haya match. Al confirmar, el sistema crea los registros de pensión de forma idempotente y **aprende la palabra clave** para que el mes siguiente se resuelva casi solo.

El objetivo es que la carga mensual pase de "transcribir a mano" a "revisar y confirmar", sin introducir dependencias pesadas ni servicios externos. El matching es determinista (nombre normalizado + diccionario de palabras clave + historial), por lo que corre en el iHost sin costo de red ni de IA. Un formato CSV propio queda como fallback robusto y exportable cuando el PDF no tenga capa de texto (escaneo/foto), caso en el que el OCR queda explícitamente fuera de alcance por consumo de recursos.

Consideraciones de iHost: el parseo es en memoria sobre archivos de pocos KB, no se agregan binarios externos ni OCR, y las tablas nuevas son de tamaño despreciable (decenas de filas por mes). Los archivos temporales se procesan en memoria y nunca se persisten en disco.

Consideraciones de UI obligatorias:
- Todo `input`/`select`/`textarea` usa los tokens del tema (`bg-card`, `text-text`, `text-text-secondary`), jamás colores hardcodeados.
- Todo selector de categoría, hijo, período o estado usa el componente `Select` (`frontend/src/components/Select.tsx`), con `searchable` cuando corresponda. Prohibido `<select>` nativo.
- La pantalla de importación es una página con lista padre (`/pension/registros`): el retorno va por la flecha atrás del header, registrada en `BACK_ROUTES` de `frontend/src/components/DashboardLayout.tsx`, nunca con un link "← Título" en el contenido.
- La pantalla debe verificarse en darkmode (texto y placeholders legibles) antes de pasar a `pending_release`.

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001 — Entrada de archivo**: permitir importar un PDF con capa de texto, un `.txt` o un `.csv` desde la pantalla de importación. Tamaño máximo 2 MB.
2. **REQ-002 — Extracción y parseo de líneas**: obtener líneas `CONCEPTO ... MONTO` tolerando:
   - separador de miles y decimal estilo es (`24.500,00`) y estilo en (`24,500.00`);
   - decimales con coma o punto (`3845,5`, `3845.50`, `3845`);
   - símbolos de moneda (`C$`, `$`, `NIO`, `USD`) y espacios múltiples;
   - encabezado de período en español (`GASTOS ESCOLARES - OCTUBRE 2026`, `PENSION OCTUBRE 2026`) para inferir `year`/`month`.
3. **REQ-003 — Líneas de control**: ignorar como registro las líneas de subtotal/total (`TOTAL A REEMBOLSAR`, `TOTAL`, `SUBTOTAL`), pero usarlas como control: si el total declarado no coincide con la suma de las líneas compartidas, mostrar una advertencia no bloqueante en el preview.
4. **REQ-004 — Preview obligatorio**: nunca escribir en la base sin confirmación. El preview muestra por fila: texto original, concepto normalizado, monto detectado, moneda, período, hijo, categoría sugerida, confianza y motivo del match.
5. **REQ-005 — Matching determinista en capas**:
   1. coincidencia exacta con el nombre normalizado de una categoría de pensión (sin acentos, mayúsculas ni signos);
   2. coincidencia por palabra clave registrada en `pension_category_keywords`;
   3. coincidencia difusa (Levenshtein ≥ 0.85) contra nombres y palabras clave;
   4. historial: último registro de la misma categoría/hijo para sugerir monto y detectar variaciones.
   Cada sugerencia devuelve un score de 0 a 1 y el motivo (`name`, `keyword`, `fuzzy`, `history`).
6. **REQ-006 — Fila sin match**: la fila se muestra como "Sin categoría" con la acción **Crear categoría**, que abre el formulario de categoría de pensión precargado con el concepto detectado. Al crearla, la fila queda asignada a la categoría nueva sin salir del preview.
7. **REQ-007 — Confirmación idempotente**: al confirmar se crean `support_records` para el hijo/período/categoría elegidos. Si ya existe un registro para `(child_id, pension_category_id, year, month)`, se omite y se informa como "ya existente" (no se duplica ni se sobreescribe).
8. **REQ-008 — Aprendizaje de palabras clave**: al confirmar una fila con categoría elegida, se guarda el concepto normalizado como palabra clave de esa categoría (si no existe). Las palabras clave son visibles, editables y borrables desde la UI.
9. **REQ-009 — Formato propio (CSV)**: importar y exportar el formato canónico definido en §4.6, con plantilla descargable y opción "Exportar mes anterior" para reutilizar la estructura del mes previo.
10. **REQ-010 — Autenticación y validación**: todos los endpoints requieren sesión. Se valida período (`year` 2000-2100, `month` 1-12), monto > 0, categoría existente, hijo existente y que el mes no esté cerrado (`month_closings`) para poder confirmar.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-011 — Comparación con el mes anterior**: mostrar el monto del mes previo por categoría y resaltar variaciones mayores al 20 % (ej. `+33 %`).
2. **REQ-012 — Detección de duplicados en el archivo**: si dos líneas producen el mismo `(hijo, categoría, período)`, marcar la segunda y no permitir confirmarla sin resolver.
3. **REQ-013 — Auditoría de importaciones**: registrar lote (archivo, hash, período, cantidad de filas, creadas/omitidas, fecha) para poder consultar qué se importó y repetir o revertir la carga de un lote.
4. **REQ-014 — Edición masiva en preview**: cambiar categoría, hijo o estado (pending/paid) a varias filas seleccionadas a la vez.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-015 — Vínculo con presupuesto**: opción de crear también una `transaction` en la cuenta/categoría de presupuesto mapeada (requiere spec aparte si se automatiza el vínculo).
2. **REQ-016 — Recordatorio mensual**: si no hay importación del mes en curso al día configurado, generar una alerta (reutiliza el sistema de alertas existente).
3. **REQ-017 — Multi-moneda**: conversión con tipo de cambio manual en el preview para líneas en USD.

### 2.4 Fuera de alcance

- **OCR** de PDF escaneado o foto: no es viable en el iHost (peso, CPU y memoria) y agrega una dependencia grande. Estos casos se resuelven con CSV o pegando el texto.
- **Integración con IA/LLM** para interpretar el archivo: el matching debe ser determinista, offline y reproducible.
- Modificar el esquema o el comportamiento de `debts`/`debt_bills`.

### 2.5 Requerimientos No Funcionales

- **Rendimiento**: preview de hasta 100 líneas en < 1 s en el iHost; parseo O(n) sobre el texto.
- **Memoria**: sin picos > 32 MB por request; archivos se procesan en memoria y se descartan.
- **Almacenamiento**: tablas nuevas con decenas de filas/mes; sin archivos persistidos.
- **Seguridad**: endpoints autenticados; no se loguean montos, conceptos ni contenido del archivo; se registra solo nombre de archivo, hash y conteos.
- **Privacidad**: el archivo cargado no se guarda; el PDF de origen nunca se persiste ni se commitea.
- **iHost**: sin dependencias de red ni binarios externos; una única librería Go liviana para PDF (ver ADR-001).
- **Compatibilidad**: multi-arch (`amd64`, `arm/v7`, `arm64`) sin cambios en el Dockerfile.

---

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- Estado actual del módulo de pensión: `support_records` con `UNIQUE(child_id, pension_category_id, year, month)`, categorías en `pension_categories` y cierre mensual en `month_closings` (SPEC-049, SPEC-050, SPEC-051).
- Los registros del mes anterior se crearon manualmente y sirven como historial de matching (misma categoría, mismo hijo, montos similares).
- El PDF de referencia es un PDF de iOS con capa de texto (verificado con extracción de texto); no es un escaneo.
- El módulo de presupuesto (SPEC-093) está disponible pero no se usa para estos gastos; el importador se enfoca en pensión.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **A. Parseo server-side del PDF + preview** | Flujo real del usuario (recibe PDF), una sola pantalla, sin pasos manuales | Dependencia Go para PDF; PDFs escaneados no funcionan | ✅ Seleccionada con fallback CSV/texto |
| B. Solo CSV/formato propio | Cero dependencias, 100 % robusto | Obliga a transcribir/convertir el PDF cada mes | ✅ Soportado como fallback (REQ-009) |
| C. Parseo en el frontend (pdf.js) | Sin tocar el backend | +300 KB de JS, más CPU en móvil, difícil de testear | ❌ Rechazada |
| D. OCR en el iHost | Cubriría escaneos | Dependencia pesada, RAM/CPU altas, lento | ❌ Rechazada (fuera de alcance) |
| E. LLM para interpretar | Tolerante a formatos | Costo, red, no determinista, privacidad | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001 — Librería PDF**
- **Contexto**: el contenedor final es distroless y no tiene `pdftotext`; se necesita extraer texto en Go.
- **Decisión**: usar `github.com/ledongthuc/pdf` (MIT, pure Go, liviana) para leer la capa de texto. Si la extracción devuelve texto vacío o sin líneas válidas, responder `422` con un mensaje que ofrezca CSV/texto pegado.
- **Consecuencias**: +1 dependencia pequeña; PDFs escaneados no se soportan (documentado y con fallback).

**ADR-002 — Matching determinista y sin IA**
- **Contexto**: iHost con recursos limitados, datos financieros privados, necesidad de reproducibilidad.
- **Decisión**: normalización + diccionario en DB + Levenshtein propio + historial. Sin llamadas externas.
- **Consecuencias**: el usuario confirma siempre; el sistema mejora con cada importación vía palabras clave aprendidas.

**ADR-003 — Preview-first, escritura solo al confirmar**
- **Contexto**: el parseo puede fallar o cambiar el formato mes a mes.
- **Decisión**: `preview` no escribe en `support_records`; solo `confirm` crea registros, dentro de una transacción.
- **Consecuencias**: cero cargas erróneas silenciosas; el lote queda auditable.

**ADR-004 — Diccionario de palabras clave en SQLite, no hardcodeado**
- **Contexto**: los nombres de conceptos cambian (COLEGIATURA, MENSUALIDAD, COLEGIO) y las categorías son editables por el usuario.
- **Decisión**: tabla `pension_category_keywords` con seed inicial + aprendizaje al confirmar + CRUD en UI.
- **Consecuencias**: el mapeo se adapta sin deploy; requiere una pantalla simple de administración.

**ADR-005 — Formato CSV canónico como ciudadano de primera**
- **Contexto**: el PDF puede faltar, cambiar o llegar escaneado.
- **Decisión**: definir un CSV estable (§4.6), importable y exportable, con plantilla y exportación del mes anterior.
- **Consecuencias**: integración simple con Excel/Notas; sirve de respaldo y de contrato para futuras fuentes.

---

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[PDF / TXT / CSV] ──upload──► [API /api/pension/import/preview]
                                      │
                                      ▼
                         [PensionImportService]
                         ├── Extractor (PDF/TXT/CSV)
                         ├── Parser de líneas y período
                         ├── Matcher (nombre + keywords + fuzzy + historial)
                         └── Validador de duplicados/totales
                                      │
                                      ▼
                         [Preview JSON (no escribe)]
                                      │
                              usuario confirma
                                      ▼
                    [API /api/pension/import/confirm]
                         ├── INSERT support_records (tx)
                         ├── INSERT keywords aprendidas
                         └── INSERT lote de auditoría
                                      │
                                      ▼
                                  [SQLite]
```

### 4.2 Componentes

#### 4.2.1 `internal/services/pension_import.go`
- **Responsabilidad**: orquestar extracción, parseo, matching y confirmación.
- **Interfaz**: `Preview(ctx, input) (*PreviewResult, error)`, `Confirm(ctx, req) (*ConfirmResult, error)`.
- **Dependencias**: storage de categorías de pensión, hijos, registros de pensión, keywords e import batches.
- **Ubicación**: `internal/services/pension_import.go`.

#### 4.2.2 `internal/services/pension_import_parser.go`
- **Responsabilidad**: convertir texto plano en `[]ParsedLine` (concepto, monto, moneda) y detectar período.
- **Reglas**:
  - ignorar líneas vacías y encabezados sin monto;
  - ignorar `TOTAL`, `SUBTOTAL`, `TOTAL A REEMBOLSAR` (se conservan como control);
  - monto = último token numérico de la línea;
  - concepto = resto de la línea, sin el monto ni símbolos de moneda;
  - normalizar el período desde nombres de mes en español/inglés y año de 4 dígitos.
- **Ubicación**: `internal/services/pension_import_parser.go`.

#### 4.2.3 `internal/services/pension_import_match.go`
- **Responsabilidad**: calcular la mejor categoría para cada línea.
- **Algoritmo**:
  1. normalizar concepto (mayúsculas, sin acentos, sin signos, espacios colapsados);
  2. nombre de categoría normalizado idéntico → `score 1.00`, motivo `name`;
  3. palabra clave contenida como token/frase → `score 0.95`, motivo `keyword`;
  4. Levenshtein normalizado ≥ 0.85 contra nombres/claves → `score 0.80`, motivo `fuzzy`;
  5. si hay historial de la categoría para el hijo, usar el último monto como referencia (`previous_amount`) y ajustar `score` según diferencia (no bloqueante).
- **Umbrales**: `≥ 0.85` alta, `0.60–0.85` media, `< 0.60` sin match.

#### 4.2.4 `internal/storage/pension_import.go`
- **Responsabilidad**: persistencia de keywords, lotes y filas; creación transaccional de `support_records`.
- **Ubicación**: `internal/storage/pension_import.go`.

#### 4.2.5 `internal/api/pension_import_handlers.go`
- **Responsabilidad**: endpoints HTTP, parseo de multipart/JSON, mapeo de errores.
- **Ubicación**: `internal/api/pension_import_handlers.go`.
- **Rutas** (registradas en `internal/api/routes.go`, todas con `authMiddleware`):
  - `POST /api/pension/import/preview`
  - `POST /api/pension/import/confirm`
  - `GET  /api/pension/import/batches`
  - `GET  /api/pension/import/template.csv`
  - `GET  /api/pension/import/export?year=&month=`
  - `GET/POST/DELETE /api/pension/keywords`

#### 4.2.6 Frontend
- **Página**: `frontend/src/pages/PensionImportPage.tsx`, ruta `/pension/import`.
- **Entrada**: acción **Importar gastos** en el menú de 3 puntos de `RegistrosPage.tsx` (no botón directo en el header).
- **Paso 1**: dropzone + selector de archivo, o textarea "Pegar texto", o descarga de plantilla CSV.
- **Paso 2**: preview en cards (móvil) / tabla (desktop) con:
  - concepto original y monto editable;
  - `Select` searchable de categoría (incluye "Sin categoría");
  - `Select` de hijo (si hay más de uno) y de estado;
  - badge de confianza y motivo;
  - monto del mes anterior + variación (REQ-011);
  - acción **Crear categoría** por fila cuando no hay match.
- **Paso 3**: resumen (crear N, omitir M, errores) y botón **Confirmar importación**.
- **Retorno**: flecha atrás en el header (`BACK_ROUTES` → `/pension/registros`).
- **i18n**: claves nuevas en `frontend/public/i18n/{es,en}.json` bajo `pension.import.*` (fuente de verdad: `frontend/public/i18n/`).

### 4.3 Modelo de datos

Migración `0041_create_pension_import.up.sql` / `.down.sql`.

```
Entidad: pension_category_keywords
- id: INTEGER PK
- pension_category_id: INTEGER NOT NULL → pension_categories(id) ON DELETE CASCADE
- keyword: TEXT NOT NULL                 (texto legible, ej. "COLEGIATURA")
- normalized_keyword: TEXT NOT NULL      (normalizado, UNIQUE)
- source: TEXT NOT NULL DEFAULT 'learned' CHECK (source IN ('seed','learned','manual'))
- created_at, updated_at
- UNIQUE(normalized_keyword)

Entidad: pension_import_batches
- id: INTEGER PK
- filename: TEXT
- file_hash: TEXT
- period_year: INTEGER
- period_month: INTEGER
- status: TEXT CHECK (status IN ('preview','confirmed','discarded'))
- total_lines, created_count, skipped_count: INTEGER
- created_at, confirmed_at

Entidad: pension_import_rows
- id: INTEGER PK
- batch_id: INTEGER NOT NULL → pension_import_batches(id) ON DELETE CASCADE
- line_no: INTEGER
- raw_text: TEXT
- concept: TEXT
- amount: REAL
- currency: TEXT
- suggested_category_id: INTEGER NULL → pension_categories(id)
- chosen_category_id: INTEGER NULL → pension_categories(id)
- child_id: INTEGER NULL → children(id)
- confidence: REAL
- match_reason: TEXT
- action: TEXT CHECK (action IN ('create','skip','existing','error'))
- created_at
```

Índices: `pension_import_rows(batch_id)`, `pension_import_batches(period_year, period_month)`.

**Seed inicial** (solo si la categoría existe por nombre, para no fallar en instalaciones personalizadas):

| Palabra clave | Categoría destino |
|---|---|
| `PENSION` | Pensión Alimenticia |
| `COLEGIATURA`, `MENSUALIDAD` | Colegio - Mensualidad |
| `COMBUSTIBLE`, `TRANSPORTE` | Colegio - Transporte |
| `UNIFORME` | Colegio - Uniforme |
| `SEGURO` | Colegio - Seguro |
| `MATRICULA` | Colegio - Matrícula |
| `LIBROS` | Colegio - Libros |
| `HOPPERS` | Colegio - Actividad - Hoppers |
| `ENCUENTRO CON CRISTO` | Colegio - Actividad - Encuentro con Cristo |

> **Decisión pendiente de confirmar con el usuario**: `COMBUSTIBLE` se mapea por defecto a **Colegio - Transporte**. Si en la práctica corresponde a combustible del auto, se cambia la palabra clave desde la UI sin tocar código.

### 4.4 APIs / Contratos

#### `POST /api/pension/import/preview`

`multipart/form-data` con `file` **o** `application/json`:

```json
{
  "text": "GASTOS ESCOLARES - OCTUBRE 2026\nCOLEGIATURA 8000\nTOTAL A REEMBOLSAR 4000\nCOMBUSTIBLE 1200\nPENSION OCTUBRE 12000",
  "year": 2026,
  "month": 10
}
```

**Response 200** (datos sintéticos):

```json
{
  "batch_id": 1,
  "period": { "year": 2026, "month": 10 },
  "source": "pdf",
  "warnings": [
    { "code": "total_mismatch", "message": "El total declarado no coincide con la suma de las líneas." }
  ],
  "declared_total": 4000,
  "lines": [
    {
      "line_no": 2,
      "raw_text": "COLEGIATURA 8000",
      "concept": "COLEGIATURA",
      "amount": 8000,
      "currency": "NIO",
      "suggested_category_id": 5,
      "category_name": "Colegio - Mensualidad",
      "confidence": 0.95,
      "match_reason": "keyword",
      "previous_amount": 8000,
      "variation_pct": 0
    },
    {
      "line_no": 4,
      "raw_text": "COMBUSTIBLE 1200",
      "concept": "COMBUSTIBLE",
      "amount": 1200,
      "currency": "NIO",
      "suggested_category_id": 3,
      "category_name": "Colegio - Transporte",
      "confidence": 0.95,
      "match_reason": "keyword",
      "previous_amount": 1500,
      "variation_pct": -20
    },
    {
      "line_no": 5,
      "raw_text": "PENSION OCTUBRE 12000",
      "concept": "PENSION OCTUBRE",
      "amount": 12000,
      "currency": "NIO",
      "suggested_category_id": 1,
      "category_name": "Pensión Alimenticia",
      "confidence": 0.95,
      "match_reason": "keyword",
      "previous_amount": 12000,
      "variation_pct": 0
    }
  ]
}
```

**Response 422** (PDF sin texto):

```json
{ "error": "pdf_without_text", "message": "El PDF no tiene capa de texto. Usá CSV o pegá el texto." }
```

#### `POST /api/pension/import/confirm`

```json
{
  "batch_id": 1,
  "child_id": 1,
  "rows": [
    { "line_no": 2, "action": "create", "category_id": 5, "amount": 4000, "status": "pending" },
    { "line_no": 4, "action": "create", "category_id": 3, "amount": 1200, "status": "pending" },
    { "line_no": 5, "action": "create", "category_id": 1, "amount": 12000, "status": "pending" }
  ]
}
```

**Response 200**:

```json
{
  "batch_id": 1,
  "created": 3,
  "skipped_existing": 0,
  "errors": [],
  "learned_keywords": 3
}
```

**Errores**: `400` validación, `401` sin sesión, `409` mes cerrado o duplicado en el archivo, `422` formato no soportado.

### 4.5 Interfaz de usuario

- **Ruta**: `/pension/import` (header con flecha atrás en móvil).
- **Estados**: cargando, sin archivo, preview, confirmado, error.
- **Acciones**: elegir archivo, pegar texto, descargar plantilla, quitar fila, crear categoría, confirmar.
- **Accesibilidad**: labels asociados, foco visible, contraste en darkmode.
- **Sin `<select>` nativo**: categoría, hijo y estado usan `Select`; categoría con búsqueda (`searchable`).
- **Empty state**: si el preview no detecta líneas, mostrar guía con ejemplo sintético y enlace a la plantilla CSV.

### 4.6 Formato CSV canónico

Encabezado obligatorio, UTF-8, coma, punto decimal, sin separador de miles:

```csv
periodo,concepto,monto,moneda,categoria,hijo,estado,notas
2026-10,PENSION OCTUBRE,12000.00,NIO,Pensión Alimenticia,,pending,
2026-10,COLEGIATURA (50%),4000.00,NIO,Colegio - Mensualidad,,pending,
2026-10,COMBUSTIBLE,1200.00,NIO,Colegio - Transporte,,pending,
```

Reglas:
- `periodo`: `YYYY-MM` (obligatorio).
- `categoria` vacía → se aplica matching (útil para importar solo concepto/monto).
- `hijo` vacío → el único hijo si existe exactamente uno; si hay varios, error de validación.
- `monto`: > 0, punto decimal (se aceptan coma decimal al importar).
- `estado`: `pending` (default) o `paid`.
- `moneda`: `NIO` (default) o código existente en `currencies`.

### 4.7 Dependencias

- **Internas**: `pension_categories`, `children`, `support_records`, `month_closings`, `categories` (solo lectura para P2).
- **Externas**: `github.com/ledongthuc/pdf` (extracción de texto). Sin servicios de red.
- **Migración**: `0041_create_pension_import.up.sql` / `.down.sql`.

---

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] **CA-001**: Dado un PDF con capa de texto que contiene `COLEGIATURA`, `TOTAL A REEMBOLSAR`, `COMBUSTIBLE` y `PENSION`, cuando se sube, entonces el preview muestra 3 líneas (sin la de total) y detecta el período del encabezado.
- [ ] **CA-002**: Dado un monto `24.500,00`, `24500`, `24,500.00` o `C$ 24500.00`, el parser lo interpreta como `24500.00`.
- [ ] **CA-003**: Dado un concepto `PENSION OCTUBRE`, el matcher sugiere `Pensión Alimenticia` con confianza ≥ 0.85 y motivo `keyword`.
- [ ] **CA-004**: Dado un concepto `COLEGIATURA`, el matcher sugiere `Colegio - Mensualidad`; `COMBUSTIBLE` sugiere `Colegio - Transporte` (configurable por palabra clave).
- [ ] **CA-005**: Dado un concepto sin coincidencia, la fila aparece sin categoría y el botón **Crear categoría** abre el formulario precargado con el concepto.
- [ ] **CA-006**: Al crear la categoría desde el preview, la fila queda asignada y la categoría aparece en el selector sin recargar la página.
- [ ] **CA-007**: Al confirmar, se crean exactamente los registros marcados; los existentes para el mismo `(hijo, categoría, período)` se informan como omitidos y no se duplican.
- [ ] **CA-008**: Al confirmar, el concepto normalizado queda guardado como palabra clave; una segunda importación del mismo concepto la resuelve sin intervención.
- [ ] **CA-009**: Si el PDF no tiene capa de texto, la API responde 422 con mensaje accionable y la UI ofrece CSV/pegar texto.
- [ ] **CA-010**: La importación de un CSV canónico válido crea los registros; un CSV con `monto` vacío o `periodo` inválido devuelve error de validación por fila.
- [ ] **CA-011**: La exportación del mes anterior genera un CSV que, reimportado sin cambios, produce un preview válido.
- [ ] **CA-012**: No se puede confirmar sobre un mes cerrado (`month_closings`); la API responde `409` y la UI lo explica.
- [ ] **CA-013**: El lote queda registrado en `pension_import_batches` con conteos de creadas/omitidas y fecha.
- [ ] **CA-014**: La acción **Importar gastos** está en el menú de 3 puntos de Registros, no como botón suelto en el header.
- [ ] **CA-015**: No existe ningún `<select>` nativo en la pantalla nueva (`npm run check:ui` pasa).
- [ ] **CA-DARK**: Los inputs y selectores usan tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y se verificó legibilidad en darkmode (texto y placeholders).
- [ ] **CA-BACK**: La página `/pension/import` tiene flecha atrás en el header registrada en `BACK_ROUTES` y no muestra links "← Título" en el contenido.

### 5.2 No funcionales

- [ ] **CA-NF-001**: Preview de 100 líneas en < 1 s en el iHost.
- [ ] **CA-NF-002**: Consumo adicional de memoria < 32 MB por request.
- [ ] **CA-NF-003**: El archivo subido no se persiste en disco ni en logs.
- [ ] **CA-NF-004**: La imagen sigue compilando para `linux/amd64`, `linux/arm/v7` y `linux/arm64`.

### 5.3 Testing

- **Unit tests**:
  - parser: formatos de monto (`24.500,00`, `24,500.00`, `24500`, `3845,5`), líneas de total, detección de período, líneas vacías;
  - matcher: exacto, keyword, fuzzy, sin match, desempate entre dos categorías;
  - CSV: parseo y generación, encabezado inválido, columnas faltantes.
- **Integration tests** (httptest + SQLite en `/tmp`):
  - preview no escribe en `support_records`;
  - confirm crea N registros y omite duplicados;
  - mes cerrado → 409;
  - aprendizaje de keyword al confirmar;
  - PDF sin texto → 422.
- **Manual**: importar el PDF real del mes, verificar preview, crear una categoría nueva, confirmar y revisar el listado de Registros. Validar contra una copia de la DB de producción, nunca contra `data/app.db`.
- **UI**: build de `frontend/` con `check:ui`, revisión en darkmode y móvil.

---

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración 0041 (keywords, batches, rows) + seeds + storage | 0,5 día | — |
| 2 | Parser PDF/TXT/CSV + detección de período y totales | 1 día | Fase 1 |
| 3 | Matcher (normalización, keyword, fuzzy, historial) + tests | 1 día | Fase 2 |
| 4 | Endpoints `preview`, `confirm`, `batches`, `template.csv`, `export` | 1 día | Fases 2-3 |
| 5 | Frontend: página de importación, preview, crear categoría, i18n | 1,5 días | Fase 4 |
| 6 | Tests de integración, validación con copia de producción, darkmode | 0,5 día | Fase 5 |
| 7 | Documentación y release | 0,5 día | Fase 6 |

### 6.2 Milestones

1. **MVP**: PDF/TXT/CSV → preview → confirmar con categorías existentes, sin crear categorías ni keywords (solo sugerencias).
2. **V1.0**: creación de categoría desde el preview, aprendizaje de palabras clave, auditoría de lotes, exportación del mes anterior.
3. **V1.1 (P2)**: vínculo con presupuesto, recordatorio mensual, multi-moneda.

---

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| El PDF cambia de formato o llega escaneado | Media | Alto | Fallback CSV/texto pegado; 422 accionable; el parser tolera variantes de orden y separadores |
| La librería PDF falla con un generador específico | Media | Medio | Tests con PDFs sintéticos de varios orígenes; fallback a texto/CSV |
| `COMBUSTIBLE` u otro concepto mapea a la categoría equivocada | Media | Medio | Preview obligatorio + keywords editables + confirmación del usuario |
| Duplicados por reimportación | Baja | Alto | `UNIQUE(child, category, year, month)` + acción `existing` explícita |
| Crecimiento de tablas de auditoría | Baja | Bajo | Retención sugerida: borrar batches > 24 meses (job manual o P2) |
| Montos con separadores ambiguos (`1.500` = 1500 o 1,5) | Media | Medio | Regla documentada: si el último grupo tiene 3 dígitos y no hay decimales explícitos, se interpreta como miles; el preview permite corregir |
| Dependencia nueva rompe multi-arch | Baja | Alto | Verificar buildx de las 3 plataformas antes del release |

---

## 8. Notas y Referencias

- SPEC-048: CRUD de categorías de pensión.
- SPEC-049: Backend de registros de pensión (modelo `support_records`).
- SPEC-050: Frontend de registros de pensión.
- SPEC-051: Generación mensual y emails.
- SPEC-093: Sistema de presupuesto (posible vínculo P2).
- SPEC-063: Navegación de retorno en páginas de detalle.
- SPEC-118: Guardrail de dropdowns (`check-ui-rules.cjs`).
- `docs/project-rules.md` §4 (UI), §6 (DB) y §8 (datos privados).
- Librería evaluada: `github.com/ledongthuc/pdf` (MIT, extracción de texto).

---

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-03 | opencode | Creación inicial de la especificación (draft) a partir del análisis del PDF mensual de gastos escolares y de los registros históricos del módulo de pensión. |
