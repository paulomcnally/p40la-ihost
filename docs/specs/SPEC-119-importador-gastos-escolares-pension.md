---
title: "Importador mensual de gastos escolares y pensión por palabras clave"
id: "SPEC-119"
status: "in_progress"
author: "opencode"
created: "2026-11-03"
updated: "2026-11-03"
github_issue: 122
---

# Importador mensual de gastos escolares y pensión por palabras clave

**ID**: SPEC-119  
**Estado**: in_progress  
**Autor**: opencode  
**Creado**: 2026-11-03  
**Actualizado**: 2026-11-03

---

## 1. Resumen Ejecutivo

Cada mes llega un comprobante en PDF con los gastos escolares y la pensión del mes (conceptos como `PENSION`, `COLEGIATURA`, `COMBUSTIBLE` y montos). Hoy esos valores se cargan a mano, uno por uno, en **Pensión → Registros**, repitiendo categorías que ya se usaron el mes anterior. El proceso es lento y propenso a errores de tipeo o de categoría.

Esta spec define un **importador mensual** que recibe el PDF (con capa de texto), un archivo de texto o un CSV, extrae las líneas `concepto + monto`, detecta el período del encabezado y propone un **match contra las categorías de pensión existentes**. El usuario ve un preview editable antes de confirmar: puede ajustar categoría, hijo, monto, período y estado, o **crear una categoría nueva** cuando no haya match. Al confirmar, el sistema crea los registros de pensión de forma idempotente y **aprende la palabra clave** para que el mes siguiente se resuelva casi solo.

El objetivo es que la carga mensual pase de "transcribir a mano" a "revisar y confirmar", sin introducir dependencias pesadas ni servicios externos. El matching es determinista (nombre normalizado + diccionario de palabras clave + historial), por lo que corre en el iHost sin costo de red ni de IA. Un formato CSV propio queda como fallback robusto y exportable cuando el PDF no tenga capa de texto (escaneo/foto), caso en el que el OCR queda explícitamente fuera de alcance por consumo de recursos.

**Regla de reparto**: el documento trae los montos totales pagados por la otra persona, pero a nosotros solo nos corresponde una parte. Cada **categoría interna de pensión** define un reparto cerrado: **Compartido 50 %** (se registra la mitad) o **Completo 100 %** (se registra el total). Pensión Alimenticia y Combustible/Transporte van al 100 %; el resto de las categorías escolares, al 50 %. El reparto se decide por la categoría, **nunca por el texto del concepto** (los textos cambian mes a mes). El monto resultante se **redondea a 2 decimales** (medio centavo hacia arriba), porque es lo único transferible; el ajuste de redondeo se guarda en el lote para auditoría y no se suma al registro. La importación es **todo o nada**: si una fila no cumple, no se importa ninguna y se muestra el detalle de cada fila con el motivo del rechazo.

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
3. **REQ-003 — Líneas de control (bloqueante)**: ignorar como registro las líneas de subtotal/total (`TOTAL A REEMBOLSAR`, `TOTAL`, `SUBTOTAL`), pero usarlas como control. El total declarado se compara contra la suma calculada de las líneas compartidas. Se acepta si está entre la **suma exacta** y la **suma redondeada** (la diferencia máxima es medio centavo por cada línea que tuvo ajuste de redondeo). Fuera de ese rango, **se bloquea la importación completa**. Si el documento no trae línea de control, se muestra una advertencia no bloqueante ("sin línea de control").
4. **REQ-004 — Preview obligatorio**: nunca escribir en la base sin confirmación. El preview muestra por fila: texto original, concepto normalizado, monto total detectado, porcentaje de reparto de la categoría, **monto a registrar**, moneda, período, hijo, categoría sugerida, confianza, motivo del match y **estado de validación** (`ok` / `inválida` con el motivo).
5. **REQ-005 — Matching determinista en capas**:
   1. coincidencia exacta con el nombre normalizado de una categoría de pensión (sin acentos, mayúsculas ni signos);
   2. coincidencia por palabra clave registrada en `pension_category_keywords`;
   3. coincidencia difusa (Levenshtein ≥ 0.85) contra nombres y palabras clave;
   4. historial: último registro de la misma categoría/hijo para sugerir monto y detectar variaciones.
   Cada sugerencia devuelve un score de 0 a 1 y el motivo (`name`, `keyword`, `fuzzy`, `history`).
6. **REQ-006 — Fila sin match**: la fila se muestra como "Sin categoría" con la acción **Crear categoría**, que abre el formulario de categoría de pensión precargado con el concepto detectado. Al crearla, la fila queda asignada a la categoría nueva sin salir del preview.
7. **REQ-007 — Confirmación idempotente y todo-o-nada**: al confirmar se crean `support_records` para el hijo/período/categoría elegidos, con el monto calculado según el reparto de la categoría. Si **cualquier** fila del lote es inválida, no se crea ningún registro y se devuelve el detalle por fila con el motivo. Si ya existe un registro para `(child_id, pension_category_id, year, month)` con el mismo monto, se omite y se informa como "ya existente"; si existe con **otro** monto, el lote se bloquea como conflicto.
8. **REQ-008 — Aprendizaje de palabras clave**: al confirmar una fila con categoría elegida, se guarda el concepto normalizado como palabra clave de esa categoría (si no existe). Las palabras clave son visibles, editables y borrables desde la UI.
9. **REQ-009 — Formato propio (CSV)**: importar y exportar el formato canónico definido en §4.6, con plantilla descargable y opción "Exportar mes anterior" para reutilizar la estructura del mes previo.
10. **REQ-010 — Autenticación y validación**: todos los endpoints requieren sesión. Se valida período (`year` 2000-2100, `month` 1-12), monto > 0, categoría existente, hijo existente y que el mes no esté cerrado (`month_closings`) para poder confirmar.
11. **REQ-018 — Reparto por categoría interna**: cada categoría de pensión tiene un reparto cerrado de dos valores: `50` (Compartido) o `100` (Completo). El monto a registrar es `monto_total × reparto / 100`. El reparto se edita en el CRUD de categorías (SPEC-048) y **no depende del texto del concepto** ni de la palabra clave. El preview muestra el porcentaje y el monto resultante.
12. **REQ-019 — Redondeo a 2 decimales**: el importe se calcula con aritmética decimal exacta y se redondea a 2 decimales con **medio centavo hacia arriba** (`700.025 → 700.03`). Solo se soportan 2 decimales, que es lo que se puede transferir. El ajuste (`amount_to_record - importe_exacto`) se guarda en la fila del lote como `rounding_adjustment` (0 o 0.005) y **no** se suma al registro.
13. **REQ-020 — Resumen de validación por fila**: el preview/confirmación devuelve, por cada fila, `ok` o `invalid` con un código y mensaje de motivo (`mixed_concept`, `missing_category`, `invalid_amount`, `currency_mismatch`, `total_mismatch`, `duplicate_in_file`, `existing_conflict`, `closed_month`, `missing_period`). La UI muestra el resumen completo antes de confirmar y bloquea el botón si hay al menos una fila inválida.
14. **REQ-021 — Moneda por categoría/cobro**: cada categoría de pensión define su moneda (default `NIO`). El registro usa la moneda de la categoría y **no se convierte**. Si el archivo declara una moneda distinta a la de la categoría, la fila se marca `currency_mismatch` para revisión. Se soportan todas las monedas configuradas en `currencies` (NIO, USD, etc.).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-011 — Comparación con el mes anterior**: mostrar el monto del mes previo por categoría y resaltar variaciones mayores al 20 % (ej. `+33 %`).
2. **REQ-012 — Detección de duplicados en el archivo**: si dos líneas producen el mismo `(hijo, categoría, período)`, marcar la segunda y no permitir confirmarla sin resolver.
3. **REQ-013 — Auditoría de importaciones**: registrar lote (archivo, hash, período, cantidad de filas, creadas/omitidas, fecha) para poder consultar qué se importó y repetir o revertir la carga de un lote.
4. **REQ-014 — Edición masiva en preview**: cambiar categoría, hijo o estado (pending/paid) a varias filas seleccionadas a la vez.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-015 — Vínculo con presupuesto**: opción de crear también una `transaction` en la cuenta/categoría de presupuesto mapeada (requiere spec aparte si se automatiza el vínculo).
2. **REQ-016 — Recordatorio mensual**: si no hay importación del mes en curso al día configurado, generar una alerta (reutiliza el sistema de alertas existente).
3. **REQ-017 — Conversión de moneda**: conversión con tipo de cambio manual en el preview para líneas en una moneda distinta a la de la categoría (el registro base sigue siendo sin conversión, según REQ-021).

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

**ADR-006 — Reparto 50/100 en la categoría, no en el texto**
- **Contexto**: los conceptos del documento cambian mes a mes, pero la categoría interna es estable.
- **Decisión**: agregar `share_percent` (50 o 100) a `pension_categories`. El cálculo se hace después del matching, usando la categoría elegida. La UI de categorías expone dos opciones cerradas ("Compartido 50 %", "Completo 100 %").
- **Consecuencias**: cambiar el texto de una línea no altera el reparto; agregar una excepción nueva es editar una categoría, no código. Requiere migración y actualizar el formulario de categorías existente.

**ADR-007 — Importación todo-o-nada con validación previa**
- **Contexto**: un lote con una fila mal parseada o un total que no cuadra contamina los registros del mes.
- **Decisión**: validar todas las filas y el total antes de escribir; si hay al menos un error, no se crea ningún registro y se devuelve el resumen por fila. El reparto se redondea a 2 decimales (medio centavo hacia arriba) porque es lo único transferible, y el ajuste se registra en el lote.
- **Consecuencias**: el usuario corrige el archivo y reimporta; nunca quedan cargas parciales. El costo es que un error bloquea todo el mes hasta resolverse.

**ADR-008 — Moneda por categoría, sin conversión**
- **Contexto**: el usuario define la moneda por cobro/categoría y espera que se respete; forzar una moneda única rompería categorías en USD.
- **Decisión**: `pension_categories.currency` (default `NIO`) define la moneda; el registro la hereda. Si el archivo declara otra moneda, se marca `currency_mismatch` en lugar de convertir.
- **Consecuencias**: los montos siempre son comparables dentro de la categoría; la conversión queda como acción manual opcional (REQ-017).

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

#### 4.2.3 `internal/services/pension_import_validation.go`
- **Responsabilidad**: aplicar la regla de reparto y validar cada fila antes de confirmar.
- **Reglas**:
  1. obtener `share_percent` (50 o 100) de la categoría elegida;
  2. `importe_exacto = monto_total × share_percent / 100`, en aritmética decimal exacta;
  3. `amount_to_record = round_half_up(importe_exacto, 2)`; `rounding_adjustment = amount_to_record - importe_exacto` (0 o 0.005);
  4. si el concepto contiene más de un término mapeado a categorías distintas (ej. `COMBUSTIBLE Y COLEGIATURA`) → `mixed_concept`, no se divide y se bloquea;
  5. comparar la suma de las líneas compartidas contra `TOTAL A REEMBOLSAR` cuando exista; se acepta el rango `[suma_exacta, suma_redondeada]` (ajuste máximo de 1 centavo por línea redondeada). Fuera del rango → `total_mismatch` y lote bloqueado;
  6. si la moneda declarada en el archivo no coincide con la de la categoría → `currency_mismatch`;
  7. detectar duplicados dentro del archivo (`hijo, categoría, período`) y conflictos con registros existentes de distinto monto.
- **Salida**: `[]RowValidation {line_no, status, code, message}`.
- **Ubicación**: `internal/services/pension_import_validation.go`.

#### 4.2.4 `internal/services/pension_import_match.go`
- **Responsabilidad**: calcular la mejor categoría para cada línea (no calcula montos).
- **Algoritmo**:
  1. normalizar concepto (mayúsculas, sin acentos, sin signos, espacios colapsados);
  2. nombre de categoría normalizado idéntico → `score 1.00`, motivo `name`;
  3. palabra clave contenida como token/frase → `score 0.95`, motivo `keyword`;
  4. Levenshtein normalizado ≥ 0.85 contra nombres/claves → `score 0.80`, motivo `fuzzy`;
  5. si hay historial de la categoría para el hijo, usar el último monto como referencia (`previous_amount`) y ajustar `score` según diferencia (no bloqueante).
- **Umbrales**: `≥ 0.85` alta, `0.60–0.85` media, `< 0.60` sin match.
- **Restricción**: si el concepto matchea términos de **dos o más** categorías distintas, se devuelve `mixed_concept` en lugar de elegir una.

#### 4.2.5 `internal/storage/pension_import.go`
- **Responsabilidad**: persistencia de keywords, lotes y filas; creación transaccional de `support_records`.
- **Ubicación**: `internal/storage/pension_import.go`.

#### 4.2.6 `internal/api/pension_import_handlers.go`
- **Responsabilidad**: endpoints HTTP, parseo de multipart/JSON, mapeo de errores.
- **Ubicación**: `internal/api/pension_import_handlers.go`.
- **Rutas** (registradas en `internal/api/routes.go`, todas con `authMiddleware`):
  - `POST /api/pension/import/preview`
  - `POST /api/pension/import/confirm`
  - `GET  /api/pension/import/batches`
  - `GET  /api/pension/import/template.csv`
  - `GET  /api/pension/import/export?year=&month=`
  - `GET/POST/DELETE /api/pension/keywords`

#### 4.2.7 Frontend
- **Página**: `frontend/src/pages/PensionImportPage.tsx`, ruta `/pension/import`.
- **Entrada**: acción **Importar gastos** en el menú de 3 puntos de `RegistrosPage.tsx` (no botón directo en el header).
- **Paso 1**: dropzone + selector de archivo, o textarea "Pegar texto", o descarga de plantilla CSV.
- **Paso 2**: preview en cards (móvil) / tabla (desktop) con:
  - concepto original y monto total editable;
  - porcentaje de reparto de la categoría (`Compartido 50 %` / `Completo 100 %`) y **monto a registrar** calculado;
  - `Select` searchable de categoría (incluye "Sin categoría");
  - `Select` de hijo (si hay más de uno) y de estado;
  - badge de confianza y motivo;
  - monto del mes anterior + variación (REQ-011);
  - acción **Crear categoría** por fila cuando no hay match (con el reparto elegido en el formulario de categoría);
  - estado de validación por fila (`ok` / `inválida` + motivo).
- **Paso 3**: resumen de validación por fila (válidas, inválidas con motivo), total calculado vs. `TOTAL A REEMBOLSAR`, y botón **Confirmar importación** deshabilitado si hay cualquier fila inválida. Si se confirma, se crea todo; si falla, no se crea nada.
- **Retorno**: flecha atrás en el header (`BACK_ROUTES` → `/pension/registros`).
- **i18n**: claves nuevas en `frontend/public/i18n/{es,en}.json` bajo `pension.import.*` (fuente de verdad: `frontend/public/i18n/`).

### 4.3 Modelo de datos

Migración `0041_create_pension_import.up.sql` / `.down.sql`.

Incluye un `ALTER TABLE pension_categories ADD COLUMN share_percent INTEGER NOT NULL DEFAULT 50 CHECK (share_percent IN (50,100));` (SQLite no permite agregar CHECK con ALTER de forma directa en todas las versiones; se implementa con `ALTER TABLE ... ADD COLUMN share_percent INTEGER NOT NULL DEFAULT 50` + trigger de validación `CHECK` o validación en storage; ver nota).

> **Nota SQLite**: `ALTER TABLE ... ADD COLUMN` no acepta `CHECK` en versiones antiguas. La migración agrega la columna con default 50 y crea un trigger `BEFORE INSERT/UPDATE` que valida `share_percent IN (50,100)`, manteniendo la integridad sin recrear la tabla.

```
Entidad: pension_categories (modificada)
- share_percent: INTEGER NOT NULL DEFAULT 50   (50 = Compartido, 100 = Completo)
- currency: TEXT NOT NULL DEFAULT 'NIO'        (moneda del cobro; se respeta en el registro)

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
- amount_total: REAL                        (monto del documento)
- amount_to_record: REAL                    (monto calculado y redondeado a 2 decimales)
- rounding_adjustment: REAL NOT NULL DEFAULT 0  (ajuste por redondeo: 0 o 0.005)
- share_percent: INTEGER                    (50 o 100, copiado de la categoría al validar)
- currency: TEXT
- suggested_category_id: INTEGER NULL → pension_categories(id)
- chosen_category_id: INTEGER NULL → pension_categories(id)
- child_id: INTEGER NULL → children(id)
- confidence: REAL
- match_reason: TEXT
- validation_status: TEXT CHECK (validation_status IN ('ok','invalid'))
- validation_code: TEXT
- action: TEXT CHECK (action IN ('create','skip','existing','error'))
- created_at
```

Índices: `pension_import_rows(batch_id)`, `pension_import_batches(period_year, period_month)`.

**Seed inicial de palabras clave** (solo si la categoría existe por nombre, para no fallar en instalaciones personalizadas):

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

**Seed inicial de reparto** (por nombre de categoría):

| Categoría | `share_percent` | Significado |
|---|---:|---|
| Pensión Alimenticia | 100 | Completo |
| Colegio - Transporte (combustible) | 100 | Completo |
| Colegio - Mensualidad | 50 | Compartido |
| Colegio - Uniforme | 50 | Compartido |
| Colegio - Seguro | 50 | Compartido |
| Colegio - Matrícula | 50 | Compartido |
| Colegio - Libros | 50 | Compartido |
| Colegio - Actividad - Hoppers | 50 | Compartido |
| Colegio - Actividad escolar | 50 | Compartido |
| Colegio - Actividad - Encuentro con Cristo | 50 | Compartido |

> **Decisión pendiente de confirmar con el usuario (mapeo de combustible)**: `COMBUSTIBLE` se mapea por defecto a **Colegio - Transporte** con reparto 100 %. Si se prefiere separar conceptos, se crea una categoría interna **`Combustible`** al 100 % y `Colegio - Transporte` pasa al 50 % para transporte escolar real. El cambio se hace en datos, sin tocar código.

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
  "validation": {
    "status": "ok",
    "declared_reimbursable_total": 4000,
    "computed_reimbursable_total": 4000,
    "computed_grand_total": 17200,
    "errors": []
  },
  "lines": [
    {
      "line_no": 2,
      "raw_text": "COLEGIATURA 8000",
      "concept": "COLEGIATURA",
      "amount_total": 8000,
      "share_percent": 50,
      "amount_to_record": 4000,
      "currency": "NIO",
      "suggested_category_id": 5,
      "category_name": "Colegio - Mensualidad",
      "confidence": 0.95,
      "match_reason": "keyword",
      "previous_amount": 8000,
      "variation_pct": 0,
      "validation_status": "ok",
      "validation_code": null
    },
    {
      "line_no": 4,
      "raw_text": "COMBUSTIBLE 1200",
      "concept": "COMBUSTIBLE",
      "amount_total": 1200,
      "share_percent": 100,
      "amount_to_record": 1200,
      "currency": "NIO",
      "suggested_category_id": 3,
      "category_name": "Colegio - Transporte",
      "confidence": 0.95,
      "match_reason": "keyword",
      "previous_amount": 1500,
      "variation_pct": -20,
      "validation_status": "ok",
      "validation_code": null
    },
    {
      "line_no": 5,
      "raw_text": "PENSION OCTUBRE 12000",
      "concept": "PENSION OCTUBRE",
      "amount_total": 12000,
      "share_percent": 100,
      "amount_to_record": 12000,
      "currency": "NIO",
      "suggested_category_id": 1,
      "category_name": "Pensión Alimenticia",
      "confidence": 0.95,
      "match_reason": "keyword",
      "previous_amount": 12000,
      "variation_pct": 0,
      "validation_status": "ok",
      "validation_code": null
    }
  ]
}
```

> `declared_reimbursable_total` y `computed_reimbursable_total` cubren **solo las líneas compartidas (50 %)**, que son las que la otra persona reembolsa. `computed_grand_total` es la suma de todos los `amount_to_record` (incluye pensión y combustible al 100 %). El control bloqueante se hace contra el primero.

**Response 200 con lote inválido** (no se importa nada):

```json
{
  "batch_id": 2,
  "validation": {
    "status": "invalid",
    "declared_reimbursable_total": 500.01,
    "computed_reimbursable_total": 500.01,
    "computed_grand_total": 530.01,
    "errors": [
      { "line_no": 3, "code": "mixed_concept", "message": "La línea mezcla conceptos de dos categorías distintas." },
      { "line_no": 5, "code": "currency_mismatch", "message": "La moneda del archivo no coincide con la de la categoría." }
    ]
  },
  "lines": [
    {
      "line_no": 3,
      "raw_text": "COLEGIATURA Y COMBUSTIBLE 5000",
      "validation_status": "invalid",
      "validation_code": "mixed_concept",
      "message": "La línea mezcla conceptos de dos categorías distintas."
    },
    {
      "line_no": 4,
      "raw_text": "UNIFORME 1000.01",
      "amount_total": 1000.01,
      "share_percent": 50,
      "amount_to_record": 500.01,
      "rounding_adjustment": 0.005,
      "validation_status": "ok",
      "validation_code": null
    },
    {
      "line_no": 5,
      "raw_text": "SEGURO 60.00 USD",
      "amount_total": 60,
      "currency": "USD",
      "share_percent": 50,
      "amount_to_record": 30,
      "validation_status": "invalid",
      "validation_code": "currency_mismatch",
      "message": "La moneda del archivo (USD) no coincide con la de la categoría (NIO)."
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
    { "line_no": 2, "action": "create", "category_id": 5, "amount_total": 8000, "status": "pending" },
    { "line_no": 4, "action": "create", "category_id": 3, "amount_total": 1200, "status": "pending" },
    { "line_no": 5, "action": "create", "category_id": 1, "amount_total": 12000, "status": "pending" }
  ]
}
```

> El cliente envía el **monto total del documento**; el backend vuelve a calcular `amount_to_record` con el `share_percent` de la categoría y valida exactitud. Nunca se confía en un monto ya dividido enviado por el cliente.

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

**Response 409** (lote inválido, no se crea nada):

```json
{
  "error": "batch_invalid",
  "message": "El lote tiene filas inválidas; no se importó ningún registro.",
  "validation": { "status": "invalid", "errors": [ { "line_no": 3, "code": "mixed_concept" } ] }
}
```

**Errores**: `400` validación de request, `401` sin sesión, `409` lote inválido / mes cerrado / conflicto con registro existente, `422` formato no soportado.

### 4.5 Interfaz de usuario

- **Ruta**: `/pension/import` (header con flecha atrás en móvil).
- **Estados**: cargando, sin archivo, preview, confirmado, error.
- **Acciones**: elegir archivo, pegar texto, descargar plantilla, quitar fila, crear categoría, confirmar.
- **Accesibilidad**: labels asociados, foco visible, contraste en darkmode.
- **Sin `<select>` nativo**: categoría, hijo y estado usan `Select`; categoría con búsqueda (`searchable`).
- **Empty state**: si el preview no detecta líneas, mostrar guía con ejemplo sintético y enlace a la plantilla CSV.

### 4.6 Formato CSV canónico

Encabezado obligatorio, UTF-8, coma, punto decimal, sin separador de miles. **La columna `monto` es el monto total del documento (antes del reparto)**; el sistema aplica el porcentaje de la categoría, igual que con el PDF.

```csv
periodo,concepto,monto,moneda,categoria,hijo,estado,notas
2026-10,PENSION OCTUBRE,12000.00,NIO,Pensión Alimenticia,,pending,
2026-10,COLEGIATURA,8000.00,NIO,Colegio - Mensualidad,,pending,
2026-10,COMBUSTIBLE,1200.00,NIO,Colegio - Transporte,,pending,
```

Importe registrado con este ejemplo: 12.000 (pensión, 100 %) + 4.000 (colegiatura, 50 %) + 1.200 (combustible, 100 %) = 17.200.

Reglas:
- `periodo`: `YYYY-MM` (obligatorio).
- `categoria` vacía → se aplica matching (útil para importar solo concepto/monto).
- `hijo` vacío → el único hijo si existe exactamente uno; si hay varios, error de validación.
- `monto`: > 0, **total del documento**, punto decimal (se aceptan coma decimal al importar).
- `estado`: `pending` (default) o `paid`.
- `moneda`: `NIO` (default) o código existente en `currencies`; si no coincide con la moneda de la categoría → `currency_mismatch` (no se convierte).
- **Exportación**: el CSV exportado reconstruye el total a partir del registro guardado (`total = registrado × 100 / share_percent`), de modo que reimportarlo produce los mismos montos registrados (el redondeo vuelve a dar el mismo resultado). Si un registro no permite la reconstrucción, la exportación lo marca como error en lugar de inventar un monto.

### 4.7 Dependencias

- **Internas**: `pension_categories` (nuevos `share_percent` y `currency`, formulario de categorías), `currencies` (lectura), `pension_category_keywords` (nueva), `children`, `support_records`, `month_closings`, `categories` (solo lectura para P2).
- **Externas**: `github.com/ledongthuc/pdf` (extracción de texto). Sin servicios de red.
- **Migración**: `0041_create_pension_import.up.sql` / `.down.sql`.

---

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] **CA-001**: Dado un PDF con capa de texto que contiene `COLEGIATURA`, `TOTAL A REEMBOLSAR`, `COMBUSTIBLE` y `PENSION`, cuando se sube, entonces el preview muestra 3 líneas (sin la de total) y detecta el período del encabezado.
- [ ] **CA-002**: Dado un monto `24.500,00`, `24500`, `24,500.00` o `C$ 24500.00`, el parser lo interpreta como `24500.00`.
- [ ] **CA-003**: Dado un concepto `PENSION OCTUBRE`, el matcher sugiere `Pensión Alimenticia` con confianza ≥ 0.85 y motivo `keyword`.
- [ ] **CA-004**: Dado un concepto `COLEGIATURA`, el matcher sugiere `Colegio - Mensualidad`; `COMBUSTIBLE` sugiere `Colegio - Transporte` (configurable por palabra clave).
- [ ] **CA-004b**: El reparto se toma de la **categoría**: `Pensión Alimenticia` y `Colegio - Transporte` registran el 100 %; `Colegio - Mensualidad` registra el 50 %; cambiar el texto del concepto no cambia el reparto.
- [ ] **CA-004c**: Si el 50 % de un monto da medio centavo (ej. `1000.01 → 500.005`), el registro se redondea a `500.01` (medio centavo hacia arriba), `rounding_adjustment = 0.005`, la fila queda `ok` y el ajuste no se suma al registro.
- [ ] **CA-005**: Dado un concepto sin coincidencia, la fila aparece sin categoría y el botón **Crear categoría** abre el formulario precargado con el concepto.
- [ ] **CA-006**: Al crear la categoría desde el preview, la fila queda asignada y la categoría aparece en el selector sin recargar la página.
- [ ] **CA-007**: Al confirmar, se crean exactamente los registros válidos; si existe un registro con el mismo monto se informa "ya existente" y se omite; si existe con **otro** monto, el lote se bloquea como conflicto.
- [ ] **CA-008**: Al confirmar, el concepto normalizado queda guardado como palabra clave; una segunda importación del mismo concepto la resuelve sin intervención.
- [ ] **CA-009**: Si el PDF no tiene capa de texto, la API responde 422 con mensaje accionable y la UI ofrece CSV/pegar texto.
- [ ] **CA-010**: La importación de un CSV canónico válido crea los registros; un CSV con `monto` vacío o `periodo` inválido devuelve error de validación por fila.
- [ ] **CA-011**: La exportación del mes anterior genera un CSV con los **totales reconstruidos** que, reimportado sin cambios, produce un preview válido con los mismos montos registrados.
- [ ] **CA-012**: No se puede confirmar sobre un mes cerrado (`month_closings`); la API responde `409` y la UI lo explica.
- [ ] **CA-013**: El lote queda registrado en `pension_import_batches` con conteos de creadas/omitidas y fecha.
- [ ] **CA-014**: La acción **Importar gastos** está en el menú de 3 puntos de Registros, no como botón suelto en el header.
- [ ] **CA-015**: No existe ningún `<select>` nativo en la pantalla nueva (`npm run check:ui` pasa).
- [ ] **CA-016**: Si `TOTAL A REEMBOLSAR` queda fuera del rango `[suma exacta, suma redondeada]` de las líneas compartidas, el lote queda `invalid`, se muestra el desvío y **no se crea ningún registro**. Un desvío de hasta medio centavo por línea redondeada se acepta.
- [ ] **CA-017**: Si una línea contiene dos conceptos mapeados a categorías distintas (ej. `COMBUSTIBLE Y COLEGIATURA`), queda `invalid` con `mixed_concept` y el lote completo se bloquea.
- [ ] **CA-018**: La confirmación es **todo o nada**: con al menos una fila inválida, `created = 0` y la respuesta incluye el resumen por fila con código y mensaje.
- [ ] **CA-019**: El preview muestra por fila `amount_total`, `share_percent`, `amount_to_record`, `rounding_adjustment`, `currency` y `validation_status` con el motivo cuando corresponde.
- [ ] **CA-020**: Una línea en una moneda distinta a la de la categoría queda `invalid` con `currency_mismatch` y no se convierte automáticamente.
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
  - matcher: exacto, keyword, fuzzy, sin match, desempate entre dos categorías, `mixed_concept`;
  - reparto: 50 % exacto (`8000 → 4000`), 100 % (`12000 → 12000`), medio centavo (`1000.01 → 500.01` con ajuste 0.005);
  - validación: total declarado vs. calculado, duplicados en archivo, conflicto con registro existente de otro monto;
  - CSV: parseo y generación, encabezado inválido, columnas faltantes, reconstrucción del total al exportar.
- **Integration tests** (httptest + SQLite en `/tmp`):
  - preview no escribe en `support_records`;
  - confirm crea N registros y omite duplicados con el mismo monto;
  - lote con una fila inválida → `created = 0` y detalle por fila;
  - redondeo: `1000.01 → 500.01` con `rounding_adjustment = 0.005`; `8000 → 4000` sin ajuste;
  - `currency_mismatch` bloquea la fila;
  - `total_mismatch` bloquea el lote;
  - mes cerrado → 409;
  - aprendizaje de keyword al confirmar;
  - PDF sin texto → 422.
- **Manual**: importar el PDF real del mes, verificar preview (montos totales vs. registrados), crear una categoría nueva, confirmar y revisar el listado de Registros. Validar contra una copia de la DB de producción, nunca contra `data/app.db`.
- **UI**: build de `frontend/` con `check:ui`, revisión en darkmode y móvil.

---

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración 0041 (`share_percent` + keywords, batches, rows) + seeds + storage | 1 día | — |
| 2 | Parser PDF/TXT/CSV + detección de período y totales | 1 día | Fase 1 |
| 3 | Matcher (normalización, keyword, fuzzy, historial) + validador de reparto y totales + tests | 1,5 días | Fase 2 |
| 4 | Endpoints `preview`, `confirm`, `batches`, `template.csv`, `export` | 1 día | Fases 2-3 |
| 5 | Frontend: página de importación, preview con validación por fila, crear categoría, i18n | 1,5 días | Fase 4 |
| 6 | Formulario de categorías: reparto (50/100) y moneda | 0,5 día | Fase 1 |
| 7 | Tests de integración, validación con copia de producción, darkmode | 0,5 día | Fases 5-6 |
| 8 | Documentación y release | 0,5 día | Fase 7 |

### 6.2 Milestones

1. **MVP**: PDF/TXT/CSV → preview con reparto y validación → confirmar con categorías existentes, sin crear categorías ni keywords.
2. **V1.0**: creación de categoría desde el preview, aprendizaje de palabras clave, auditoría de lotes, exportación del mes anterior, edición del reparto en categorías.
3. **V1.1 (P2)**: vínculo con presupuesto, recordatorio mensual, conversión manual de moneda.

---

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| El PDF cambia de formato o llega escaneado | Media | Alto | Fallback CSV/texto pegado; 422 accionable; el parser tolera variantes de orden y separadores |
| La librería PDF falla con un generador específico | Media | Medio | Tests con PDFs sintéticos de varios orígenes; fallback a texto/CSV |
| `COMBUSTIBLE` u otro concepto mapea a la categoría equivocada | Media | Medio | Preview obligatorio + keywords editables + reparto por categoría + confirmación del usuario |
| El redondeo de medio centavo genera un ajuste de 0,005 por línea | Media | Bajo | Se redondea half-up a 2 decimales y el ajuste queda registrado en el lote; el registro coincide con la transferencia |
| El total declarado no coincide y bloquea el lote | Media | Medio | El preview muestra el desvío exacto; se corrige el archivo antes de importar |
| Duplicados por reimportación | Baja | Alto | `UNIQUE(child, category, year, month)` + acción `existing` explícita y conflicto si el monto difiere |
| Crecimiento de tablas de auditoría | Baja | Bajo | Retención sugerida: borrar batches > 24 meses (job manual o P2) |
| Montos con separadores ambiguos (`1.500` = 1500 o 1,5) | Media | Medio | Regla documentada: si el último grupo tiene 3 dígitos y no hay decimales explícitos, se interpreta como miles; el preview permite corregir |
| Dependencia nueva rompe multi-arch | Baja | Alto | Verificar buildx de las 3 plataformas antes del release |

---

## 8. Notas y Referencias

### 8.1 Definiciones adoptadas

1. **Estado inicial**: `pending` por defecto; se puede cambiar a `paid` por fila en el preview o después desde Registros.
2. **Moneda**: cada categoría define su moneda (default `NIO`); se respeta y no se convierte automáticamente (REQ-021).
3. **Combustible**: se mantiene `Colegio - Transporte` al 100 %; no se crea una categoría `Combustible` separada.
4. **Centavos**: se soportan 2 decimales; el medio centavo se redondea half-up (`700.025 → 700.03`) y el ajuste queda en el lote.

### 8.2 Referencias

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
| 2026-11-03 | opencode | Se incorpora la regla de reparto por categoría interna (50 % compartido / 100 % completo), validación bloqueante del total y del lote completo (todo o nada), resumen de validación por fila y ajustes en el CSV (monto total + reconstrucción al exportar). |
| 2026-11-03 | opencode | Estado pasa a `in_progress` para implementación. |
| 2026-11-03 | opencode | Implementación completa (commit `9fd11d9`): migración 0041, parser PDF/TXT/CSV, matcher, validador, endpoints, pantalla de importación, CRUD de palabras clave y reparto/moneda en categorías. |
| 2026-11-03 | opencode | Se define moneda por categoría sin conversión, se confirma `Colegio - Transporte` al 100 % para combustible y se reemplaza la división exacta por redondeo half-up a 2 decimales con ajuste auditado en el lote. Estado inicial `pending` por defecto. |

---

## 10. Implementación

**Commit de implementación**: `9fd11d9` (rama `feature/SPEC-119`).

### 10.1 Archivos principales

- `migrations/0041_create_pension_import.up.sql` / `.down.sql`: `share_percent`, `currency`, triggers de validación, tablas de palabras clave/lotes/filas y seeds.
- `internal/models/pension_import.go`, `internal/models/pension_category.go`.
- `internal/storage/pension_import.go`, `internal/storage/pension_category.go`.
- `internal/services/pension_import.go`, `pension_import_parser.go`, `pension_import_match.go`.
- `internal/api/pension_import_handlers.go`, rutas y wiring en `cmd/server/main.go`.
- `frontend/src/pages/PensionImportPage.tsx`, ruta `/pension/import`, acción en Registros, formulario de categorías con reparto/moneda, i18n `pensionImport.*`.

### 10.2 Funcionalidad entregada

- Parseo de PDF con capa de texto, TXT y CSV (incluye PDFs con concepto y monto en líneas separadas).
- Detección de período, línea de control bloqueante y validación todo-o-nada.
- Matching por nombre, palabras clave (seed + aprendizaje + CRUD) y fuzzy.
- Reparto por categoría (50/100) con redondeo half-up a 2 decimales y ajuste auditado.
- Moneda por categoría sin conversión; `currency_mismatch` por fila.
- Preview editable, creación de categoría desde la fila, auditoría de lotes y exportación del mes anterior.

### 10.3 Pendientes conocidos (P1/P2)

- REQ-014 (edición masiva en el preview) no implementado.
- REQ-015 a REQ-017 (vínculo con presupuesto, recordatorio mensual, conversión manual) quedan para una próxima spec.
- El total declarado se compara con tolerancia de medio centavo por línea redondeada; si el documento usa otro criterio de redondeo, el lote se bloquea y se corrige el archivo.
