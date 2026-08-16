---
title: "Analizador de facturas DISNORTE-DISSUR"
id: "SPEC-019"
status: "released"
author: "paulomcnally"
created: "2026-08-15"
updated: "2026-08-15"
github_issue: 19
---

# Analizador de facturas DISNORTE-DISSUR

**ID**: SPEC-019  
**Estado**: draft  
**Autor**: paulomcnally  
**Creado**: 2026-08-15  
**Actualizado**: 2026-08-15

---

## 1. Resumen Ejecutivo

Agregar un nuevo analizador de documentos al sistema para soportar facturas de energía eléctrica de **Distribuidora de Electricidad del Sur, S.A. (DISNORTE-DISSUR)**, distribuidora eléctrica de Nicaragua.

El código fuente del analizador ya está disponible en `/home/paulomcnally/Downloads/disnorte_dissur.go` y tests en `disnorte_dissur_test.go`. El analizador extrae: número de factura, monto total, año/mes de facturación y fecha de vencimiento. Usa la librería `ledongthuc/pdf` (ya existente en el proyecto) y regex para parsing del texto plano del PDF.

**iHost**: Sin impacto significativo. Reutiliza dependencias existentes (ledongthuc/pdf). El analizador se registra via `init()` en el paquete `all`.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Copiar `disnorte_dissur.go` a `internal/analyzers/all/disnorte_dissur.go`
2. **REQ-002**: Copiar `disnorte_dissur_test.go` a `internal/analyzers/all/disnorte_dissur_test.go`
3. **REQ-003**: Ajustar imports del test (actualmente importa `"github.com/paulomcnally/p40la-ihost/internal/analyzers/all"` implícitamente vía package `all`)
4. **REQ-004**: Verificar que los tests pasan con `go test ./internal/analyzers/all/`
5. **REQ-005**: Verificar que el analizador aparece en el listado de analyzers (el `init()` lo registra automáticamente)

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-006**: Confirmar que el analyzer se detecta correctamente al subir un PDF de DISNORTE-DISSUR

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-007**: Verificar que los helper functions `parseAmount` y `parseDateDMY` de claro.go se reutilizan (están en el mismo paquete `all`)

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Sin impacto adicional. El parsing es en memoria con buffers ligeros.
- **Seguridad**: Sin cambios. No expone datos sensibles.
- **iHost**: Sin dependencias nuevas. Ya usa `ledongthuc/pdf` que está en go.mod.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- El código fuente está listo para copiar. Sigue exactamente el patrón del `ClaroAnalyzer`.
- Usa `init()` para auto-registrarse en `analyzers.Register()`.
- Las funciones helper `parseAmount` y `parseDateDMY` ya están definidas en `claro.go` dentro del mismo paquete `all`, por lo tanto son accesibles directamente.
- El test incluye el texto real extraído de una factura DISNORTE-DISSUR (F122026071144973).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Copiar tal cual el código fuente | Listo para usar, tests incluidos | Nada significativo | ✅ Seleccionada |
| Reescribir desde cero | Control total | Innecesario, ya funciona | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Integración directa del código fuente proporcionado
- **Contexto**: El usuario proporcionó código funcional con tests que cubren el analyzer completo
- **Decisión**: Copiar los archivos directamente, ajustando solo imports si es necesario
- **Consecuencias**: Mínimo esfuerzo, análisis inmediato de facturas DISNORTE-DISSUR

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[PDF Upload] --> [API /analyze] --> [DisnorteDissurAnalyzer]
                                        |
                                        v
                                [ExtractedBill]
                              {amount, invoice,
                               year, month, due_date}
```

### 4.2 Componentes

#### 4.2.1 DisnorteDissurAnalyzer
- **Responsabilidad**: Extraer datos de facturas DISNORTE-DISSUR
- **Interfaz**: `analyzers.DocumentAnalyzer` (Info + Analyze)
- **Dependencias**: `ledongthuc/pdf`, `analyzers` package
- **Ubicación**: `internal/analyzers/all/disnorte_dissur.go`

### 4.3 Modelo de datos

No hay cambios en el modelo. Usa `analyzers.ExtractedBill` existente:
```
ExtractedBill:
- Amount: float64
- InvoiceNumber: string
- Year: int
- Month: int
- DueDate: *time.Time
```

### 4.4 APIs / Contratos

Sin cambios en APIs. El analyzer se registra automáticamente y se invoca existente `POST /api/institutions/:id/bills/analyze`.

### 4.5 Dependencias

- **Internas**: `analyzers` package (ya existe)
- **Externas**: `ledongthuc/pdf` (ya en go.mod)

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Los archivos `disnorte_dissur.go` y `disnorte_dissur_test.go` están en `internal/analyzers/all/`
- [ ] CA-002: `go test ./internal/analyzers/all/` pasa sin errores
- [ ] CA-003: `analyzers.List()` incluye `"disnorte_dissur"` como ID
- [ ] CA-004: Un PDF de DISNORTE-DISSUR se analiza correctamente (invoice, amount, year, month, due_date)

### 5.2 No funcionales

- [ ] CA-NF-001: Sin dependencias nuevas en go.mod

### 5.3 Testing

- **Unit tests**: Tests ya incluidos en `disnorte_dissur_test.go` (Info, Analyze con mime no soportado, regex anchor, parse completo, campos faltantes, texto vacío)
- **Integration tests**: Verificar que el analyzer aparece en el listado de la API

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Copiar archivos fuente al proyecto | 5 min | Ninguna |
| 2 | Ajustar imports si es necesario | 5 min | Fase 1 |
| 3 | Ejecutar tests y verificar | 5 min | Fase 2 |
| 4 | Verificar integración con API | 10 min | Fase 3 |

### 6.2 Milestones

1. **MVP**: Analyzer registrado y tests pasando
2. **V1.0**: Verificación manual con PDF real

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Import path incorrecto | Baja | Bajo | Verificar path actual del proyecto |
| Conflicto con funciones helper | Baja | Bajo | `parseAmount`/`parseDateDMY` ya están en el paquete `all` |

## 8. Notas y Referencias

- Código fuente: `/home/paulomcnally/Downloads/disnorte_dissur.go`
- Tests: `/home/paulomcnally/Downloads/disnorte_dissur_test.go`
- Analyzer existente similar: `internal/analyzers/all/claro.go`
- Librería PDF: `github.com/ledongthuc/pdf`

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-15 | paulomcnally | Creación inicial de la especificación |
