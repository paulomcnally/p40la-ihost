---
title: "UI de subida de facturas con análisis automático + fix overflow en BillsPage"
id: "SPEC-014"
status: "released"
author: "paulomcnally"
created: "2026-08-15"
updated: "2026-08-15"
github_issue: 15
---

# UI de subida de facturas con análisis automático + fix overflow en BillsPage

**ID**: SPEC-014  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-08-15  
**Actualizado**: 2026-08-15

---

## 1. Resumen Ejecutivo

El backend ya expone las APIs para subir y analizar facturas (`POST /api/services/{service_id}/bills/upload` y `POST /api/services/{service_id}/bills/from-extracted`), pero el frontend nunca las conectó. El usuario solo puede cargar facturas manualmente con el formulario `BillFormPage`, sin poder subir el PDF para que el analizador Claro lo procese automáticamente.

Además, en la vista de lista de facturas (`BillsPage`), al desplegar el menú de acciones (CardMenu) de una fila, el dropdown queda recortado por el contenedor con `overflow-x-auto`, provocando scroll horizontal y vertical en vez de desplegarse como overlay.

Esta spec implementa:
1. **UI de subida de PDF** con flujo de análisis → vista previa de datos extraídos → guardar (o editar y guardar).
2. **Fix de overflow** del CardMenu en la tabla de bills para que el menú se despliegue como overlay sin generar scroll.

**Consideraciones de iHost**: Solo frontend + reutilización de APIs existentes. Zero cambios de backend, DB o dependencias.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: En `BillsPage`, agregar en el `CreateMenu` la opción "Subir factura" (icono `upload`).

2. **REQ-002**: Al seleccionar "Subir factura", abrir un modal (`UploadBillModal`) con selector de archivo (accept: `.pdf,.png,.jpg,.jpeg`).

3. **REQ-003**: Al enviar el archivo, llamar a `POST /api/services/{service_id}/bills/upload` con FormData (campo `file`).

4. **REQ-004**: Mostrar estado de carga mientras analiza (spinner/texto "Analizando...").

5. **REQ-005**: Si el análisis es exitoso, mostrar los datos extraídos en un formulario editable (año, mes, monto, número de factura) con el nombre del analizador usado.

6. **REQ-006**: Botón "Guardar factura" que llama a `POST /api/services/{service_id}/bills/from-extracted` con los datos (editados si el usuario cambió algo).

7. **REQ-007**: Si el análisis falla (error del backend), mostrar el mensaje de error descriptivo en el modal.

8. **REQ-008**: Al guardar con éxito, cerrar el modal y refrescar la lista de facturas.

9. **REQ-009**: Fix de overflow: el CardMenu desplegado en la tabla de bills debe mostrarse como overlay sin provocar scroll horizontal/vertical. (Solución: contenedor de tabla sin `overflow-x-auto` en desktop, o usar portal/position fixed para el dropdown, o `overflow: visible` + manejo responsivo.)

### 2.2 Requerimientos No Funcionales

- **Rendimiento**: Zero deps nuevas. FormData nativo.
- **iHost**: Solo componentes React. Sin impacto en backend.
- **UX**: Mensajes de error claros; validar que el archivo sea PDF/PNG/JPG antes de enviar.

## 3. Investigación y Decisiones Técnicas

### 3.1 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Modal `UploadBillModal` con input file + vista editable | Flujo completo, reutiliza API existente | Requiere componente nuevo | ✅ Seleccionada |
| Subir directo a factura (sin vista previa) | Más simple | No permite corregir datos mal extraídos | ❌ Rechazada |
| Redirigir a BillFormPage precargado | Reutiliza form existente | Rompe el flujo natural "subir → ver resultado" | ❌ Rechazada |

### 3.2 Decisiones arquitectónicas

**ADR-014-001**: Componente `UploadBillModal` reutilizable.
- **Contexto**: Se necesita subir y analizar un PDF desde la vista de facturas.
- **Decisión**: Crear `frontend/src/components/UploadBillModal.tsx` con 3 estados: selección → analizando → resultado editable.
- **Consecuencias**: Flujo claro y reusable para futuros analizadores.

**ADR-014-002**: Fix de overflow del CardMenu en tablas.
- **Contexto**: El contenedor de la tabla tiene `overflow-x-auto` que recorta el dropdown del CardMenu.
- **Decisión**: Quitar `overflow-x-auto` del contenedor de tabla y envolver en un wrapper responsivo; en pantallas pequeñas el menú de acciones se posiciona dentro del card. Alternativa robusta: usar dropdown con `position: fixed` calculado desde el rect del botón.
- **Consecuencias**: El menú se despliega como overlay sin generar scroll. En pantallas muy pequeñas puede necesitar scroll de la tabla, aceptable.

## 4. Diseño Técnico

### 4.1 Componentes a modificar/crear

| Archivo | Cambio |
|---------|--------|
| `frontend/src/components/UploadBillModal.tsx` | Crear modal de subida con 3 estados |
| `frontend/src/pages/BillsPage.tsx` | Agregar opción "Subir factura" en CreateMenu + fix overflow tabla |
| `frontend/src/components/Icons.tsx` | Agregar icono `upload` |
| `frontend/src/api/index.ts` | Agregar `uploadAndAnalyze` y `createBillFromExtracted` |

### 4.2 APIs existentes (sin cambios)

- `POST /api/services/{service_id}/bills/upload` → FormData `file`, devuelve `{ extracted: {...}, analyzer_used: "..." }`
- `POST /api/services/{service_id}/bills/from-extracted` → JSON `{ amount, invoice_number, year, month }`

### 4.3 Flujo de interacción

```
[BillsPage → CreateMenu]
  └── "Subir factura" → [UploadBillModal]
        ├── Estado 1: selección de archivo
        │     ├── input file (accept pdf/png/jpg/jpeg)
        │     └── [Analizar]
        ├── Estado 2: analizando (spinner)
        └── Estado 3: resultado editable
              ├── analizador usado (badge)
              ├── año, mes, monto, número de factura (editables)
              ├── [Cancelar]
              └── [Guardar factura] → POST from-extracted → refresh lista
```

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: BillsPage tiene opción "Subir factura" en CreateMenu.
- [x] CA-002: El modal permite seleccionar archivo PDF/PNG/JPG.
- [x] CA-003: Al analizar, muestra estado de carga.
- [x] CA-004: Con PDF válido, muestra datos extraídos editables y nombre del analizador.
- [x] CA-005: Al guardar, crea la factura y refresca la lista.
- [x] CA-006: Con PDF inválido, muestra mensaje de error descriptivo.
- [x] CA-007: El CardMenu de las filas de bills se despliega como overlay sin scroll horizontal/vertical.

### 5.2 No funcionales

- [x] CA-NF-001: Build de Vite sin errores.
- [x] CA-NF-002: Sin dependencias npm nuevas.

## 6. Plan de Implementación

| Fase | Descripción | Estimación |
|------|-------------|------------|
| 1 | Agregar icono upload + métodos API | 10 min |
| 2 | Crear UploadBillModal | 40 min |
| 3 | Integrar en BillsPage + fix overflow | 20 min |
| 4 | Build + tests manuales | 15 min |

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Análisis lento con PDFs grandes | Baja | Medio | Mostrar spinner mientras analiza |
| Datos mal extraídos | Media | Medio | Vista editable para corregir antes de guardar |
| CardMenu recortado en pantallas pequeñas | Media | Bajo | Dropdown overlay + testeo responsivo |

## 8. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-15 | paulomcnally | Creación inicial de la especificación |
| 2026-08-15 | paulomcnally | Implementación: icono upload, API methods, UploadBillModal, integración BillsPage, fix CardMenu overlay. Build Vite OK |
