---
title: "Analizador de recibos ASSA - Seguro Auto"
id: "SPEC-040"
status: "released"
author: "paulomcnally"
created: "2026-08-16"
updated: "2026-08-16"
github_issue: 40
---

# Analizador de recibos ASSA - Seguro Auto

**ID**: SPEC-040  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-08-16  
**Actualizado**: 2026-08-16

---

## 1. Resumen Ejecutivo

Agregar un nuevo analizador de documentos al sistema para soportar **recibos oficiales de caja** emitidos por **ASSA, Compañía de Seguros S.A.** (Nicaragua) por cobro de prima de seguro de auto.

A diferencia de una factura de servicio, este documento es un **comprobante de pago ya realizado**: no tiene fecha de vencimiento (`DueDate` se deja en `nil`). El analizador extrae: número de recibo, monto cobrado, mes/año de pago y, como datos enriquecidos (`RawData`), número de póliza, código de cliente, número de referencia y tipo de cambio. Usa la librería `ledongthuc/pdf` (ya existente en el proyecto) y regex sobre el texto plano del PDF, siguiendo el patrón exacto de `ClaroAnalyzer` y `DisnorteDissurAnalyzer`.

**iHost**: Sin impacto significativo. Reutiliza dependencias existentes (`ledongthuc/pdf`). El analizador se registra vía `init()` en el paquete `all`. El parsing es en memoria con buffers ligeros.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Crear `internal/analyzers/all/assa_seguro_auto.go` con el `AssaSeguroAutoAnalyzer` (ID `assa_seguro_auto`, Name `ASSA - Seguro Auto`)
2. **REQ-002**: Crear `internal/analyzers/all/assa_seguro_auto_test.go` con los tests del analyzer
3. **REQ-003**: Anclar la detección del documento a `ASSA, Compañía de Seguros` (regex `reAssaAnchor`)
4. **REQ-004**: Extraer número de recibo con formato `No. <letra>[<dígito>] <número>` donde el dígito de la serie es opcional (ej: `No. H3 116495` → `H3116495` y `No. H 254818` → `H254818`), sin confundirse con la autorización DGI
5. **REQ-005**: Extraer monto total en córdobas (C$) anclado a la etiqueta `Recibido:` (ej: `2,468.11`)
6. **REQ-006**: Extraer mes/año de pago desde la fecha del recibo (formato `DD-MMM-YY`, ej: `30-JUN-26` → mes 6, año 2026)
7. **REQ-007**: Dejar `DueDate` en `nil` (es recibo, no factura con vencimiento)
8. **REQ-008**: Poblar `RawData` con `poliza`, `cliente`, `reling` y `tipo_cambio` cuando se detecten
9. **REQ-009**: Verificar que `go test ./internal/analyzers/all/` pasa sin errores
10. **REQ-010**: Verificar que el analizador aparece en el listado de analyzers (el `init()` lo registra automáticamente)

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-011**: Confirmar que el analyzer se detecta correctamente al subir un PDF de ASSA
2. **REQ-013**: Soportar recibos cuya serie de talonario es una sola letra sin dígito (`No. H 254818`), además del formato con dígito (`No. H3 116495`) — corrección de bug real reportado

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-012**: Mantener los helpers `parseAmount` y `parseMonth` reutilizados del paquete `all` (ya existentes en `claro.go`)

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Sin impacto adicional. El parsing es en memoria con buffers ligeros.
- **Seguridad**: Sin cambios. No expone datos sensibles.
- **Almacenamiento**: Sin persistencia adicional. Los datos se almacenan como en los demás analyzers.
- **iHost**: Sin dependencias nuevas. Ya usa `ledongthuc/pdf` que está en go.mod.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- El código fuente del analyzer y sus tests están listos, provistos por el usuario.
- Sigue exactamente el patrón de `DisnorteDissurAnalyzer`: ancla de detección (`reAssaAnchor`), extracción con regex, validación de campos obligatorios y error descriptivo.
- El documento es un **recibo oficial de caja** (comprobante de pago), no una factura. Por lo tanto no aplica `DueDate`.
- Los helpers `parseAmount` y `parseMonth` ya están definidos en `claro.go` dentro del mismo paquete `all`, por lo que son accesibles directamente.
- El texto real del recibo (reimpresión No. H3 116495) se incluye en el test como `assaRealText`. Es un caso realista: las celdas del PDF quedan concatenadas sin separador (ej: `TARRecibido: 2,468.11Total`), por lo que los regex se anclan en etiquetas.
- El regex del número de recibo (`[A-Z]\d` seguido de espacio y 4+ dígitos) evita confundirse con la autorización DGI (`No. ASFC 01/0105/12/2021/1.`) que no sigue ese patrón.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Usar el código fuente provisto | Listo para usar, tests incluidos | Nada significativo | ✅ Seleccionada |
| Reescribir desde cero | Control total | Innecesario, ya funciona | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Integración directa del código fuente proporcionado
- **Contexto**: El usuario proporcionó código funcional con tests que cubren el analyzer completo
- **Decisión**: Crear los archivos directamente en `internal/analyzers/all/`
- **Consecuencias**: Mínimo esfuerzo, análisis inmediato de recibos ASSA

**ADR-002**: `DueDate` nulo porque es recibo, no factura
- **Contexto**: El recibo de ASSA es comprobante de pago ya realizado, sin vencimiento pendiente
- **Decisión**: No rellenar `DueDate` (queda `nil`)
- **Consecuencias**: El bill generado no tendrá fecha de vencimiento; el mes/año se toman de la fecha del recibo

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[PDF Upload] --> [API /analyze] --> [AssaSeguroAutoAnalyzer]
                                        |
                                        v
                                [ExtractedBill]
                              {amount, invoice,
                               year, month, due_date: nil,
                               raw_data: {poliza, cliente, reling, tipo_cambio}}
```

### 4.2 Componentes

#### 4.2.1 AssaSeguroAutoAnalyzer
- **Responsabilidad**: Extraer datos de recibos oficiales de caja de ASSA Compañía de Seguros
- **Interfaz**: `analyzers.DocumentAnalyzer` (Info + Analyze)
- **Dependencias**: `ledongthuc/pdf`, `analyzers` package
- **Ubicación**: `internal/analyzers/all/assa_seguro_auto.go`

### 4.3 Modelo de datos

No hay cambios en el modelo. Usa `analyzers.ExtractedBill` existente:
```
ExtractedBill:
- Amount: float64
- InvoiceNumber: string
- Year: int
- Month: int
- DueDate: *time.Time  (nil para este analyzer)
- RawData: map[string]interface{} {
    poliza, cliente, reling: string
    tipo_cambio: float64
  }
```

### 4.4 APIs / Contratos

Sin cambios en APIs. El analyzer se registra automáticamente y se invoca con el endpoint existente `POST /api/institutions/:id/bills/analyze`.

### 4.5 Dependencias

- **Internas**: `analyzers` package (ya existe)
- **Externas**: `ledongthuc/pdf` (ya en go.mod)

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: `internal/analyzers/all/assa_seguro_auto.go` y `assa_seguro_auto_test.go` existen
- [ ] CA-002: `go test ./internal/analyzers/all/` pasa sin errores
- [ ] CA-003: `analyzers.List()` incluye `"assa_seguro_auto"` como ID y `"ASSA - Seguro Auto"` como nombre
- [ ] CA-004: Con el texto real del recibo (`assaRealText`), se extrae invoice `H3116495`, amount `2468.11`, year `2026`, month `6`
- [ ] CA-005: `DueDate` es `nil` en el resultado
- [ ] CA-006: `RawData` incluye `poliza=02B128265`, `cliente=1079011`, `reling=5233844`, `tipo_cambio=36.6243`
- [ ] CA-007: PDFs de otras empresas no se detectan como ASSA (el ancla no matchea)
- [ ] CA-008: El regex de número de recibo no matchea la autorización DGI (`No. ASFC ...`)
- [ ] CA-009: Con el texto real de serie sin dígito (`assaRealTextSerieSinDigito`), se extrae invoice `H254818`, amount `202.16`, year `2025`, month `10` (regresión)

### 5.2 No funcionales

- [ ] CA-NF-001: Sin dependencias nuevas en go.mod

### 5.3 Testing

- **Unit tests**: Tests provistos en `assa_seguro_auto_test.go` (Info, mime no soportado, ancla, parse completo con texto real, serie sin dígito, campos faltantes, texto vacío, no-confusión con DGI)
- **Integration tests**: Verificar que el analyzer aparece en el listado de la API

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Crear `assa_seguro_auto.go` con el analyzer | 5 min | Ninguna |
| 2 | Crear `assa_seguro_auto_test.go` con los tests | 5 min | Ninguna |
| 3 | Ejecutar `go test ./internal/analyzers/all/` y verificar | 5 min | Fases 1-2 |
| 4 | Verificar integración con API (listado de analyzers) | 10 min | Fase 3 |

### 6.2 Milestones

1. **MVP**: Analyzer registrado y tests pasando
2. **V1.0**: Verificación manual con PDF real de ASSA

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Variaciones de formato en otros recibos ASSA | Media | Medio | Tests con texto real; el ancla y los regex son tolerantes a celdas concatenadas |
| Confusión del número de recibo con la autorización DGI | Baja | Bajo | El regex exige patrón `letra+dígito espacio 4+ dígitos`, test dedicado |
| Texto extraído sin separadores entre celdas | Media | Medio | Regex anclados a etiquetas (`Recibido:`, `Fecha:`, `Cliente:`) |

## 8. Notas y Referencias

- Código fuente y tests provistos por el usuario en la conversación
- Analyzer existente similar: `internal/analyzers/all/claro.go` y `internal/analyzers/all/disnorte_dissur.go`
- Librería PDF: `github.com/ledongthuc/pdf`
- Documento analizado: Recibo Oficial de Caja de ASSA (reimpresión No. H3 116495), prima de auto póliza `02B128265`

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-16 | paulomcnally | Creación inicial de la especificación |
| 2026-08-16 | paulomcnally | Cambio de estado a in_progress (inicio de desarrollo) |
| 2026-08-16 | paulomcnally | Fix bug real: soporte de serie de talonario sin dígito (`No. H 254818`) con dígito opcional en `reAssaInvoice` + test de regresión `TestParseAssaSeguroAuto_SerieSinDigito` |
| 2026-08-16 | paulomcnally | Cambio de estado a pending_release (pruebas manuales del usuario satisfactorias) |
| 2026-08-16 | paulomcnally | Release: merge de feature/SPEC-040 a main (commit fa04da7) y push. Estado a released |