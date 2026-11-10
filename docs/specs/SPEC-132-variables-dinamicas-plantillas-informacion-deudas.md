---
title: "Variables dinámicas en valores por defecto de plantillas de información de deudas"
id: "SPEC-132"
status: "in_progress"
author: "opencode"
created: "2026-11-09"
updated: "2026-11-09"
github_issue: 135
---

# Variables dinámicas en valores por defecto de plantillas de información de deudas

**ID**: SPEC-132  
**Estado**: in_progress  
**Autor**: opencode  
**Creado**: 2026-11-09  
**Actualizado**: 2026-11-09

---

## 1. Resumen Ejecutivo

La SPEC-131 introdujo las **plantillas de información** de deudas: definiciones reutilizables de campos (texto, número, fecha, select, checkbox y calculados) con un `default_value` estático por campo. Hoy ese valor por defecto es un literal: si el usuario quiere que una referencia de pago diga `Lote Demo 13 - Cuota 5`, debe escribir el número a mano en cada deuda y, peor aún, el dato queda congelado y no acompaña a cada cuota.

Esta spec agrega **variables dinámicas** en el `default_value` de los campos de texto y textarea. Una variable se escribe con llaves simples y clave en inglés, por ejemplo `Lote Demo 13 - Cuota {installment_number}`. El texto literal se conserva y la variable se resuelve en el momento de mostrar la información, con el contexto de la deuda y de la cuota que se está pagando. La plantilla **persiste la variable en inglés** (`{installment_number}`), no el valor resuelto, de modo que el mismo campo funciona para todas las cuotas de una deuda sin edición manual.

El editor de plantillas debe **sugerir las variables en español** ("Número de cuota", "Monto de la cuota", "Vencimiento", etc.) mostrando además la clave inglesa que se insertará. La resolución se hace en el frontend con datos que ya recibe (deuda + cuota), sin cambios de esquema SQLite y sin dependencias nuevas: el backend solo valida que las variables existan. Esto mantiene el consumo de RAM/CPU del iHost prácticamente igual y no requiere migración.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Permitir variables en el `default_value` de campos de tipo `text` y `textarea` de las plantillas de información de deudas, con la sintaxis `{clave_en_ingles}`, por ejemplo `Lote Demo 13 - Cuota {installment_number}`. El texto literal se mantiene tal cual y las variables se sustituyen al resolver.
2. **REQ-002**: Persistir en `fields_json.default_value` **la variable en inglés** (el template sin resolver), nunca el valor resuelto. La resolución se hace en tiempo de lectura/render con el contexto de la deuda y la cuota.
3. **REQ-003**: Exponer un catálogo canónico de variables con: clave en inglés (la que se persiste), etiqueta en español (la que se sugiere), ejemplo sintético y descripción breve. El catálogo se sirve desde el backend y tiene un fallback en el frontend.
4. **REQ-004**: En el formulario de plantilla (`DebtInformationTemplateFormPage`), para campos `text`/`textarea`, mostrar un panel de sugerencias de variables con etiqueta en español y la clave inglesa como texto secundario. Al hacer click, insertar `{clave}` en el `default_value` en la posición del cursor.
5. **REQ-005**: Validar las variables en backend y frontend: una variable desconocida o mal formada en `default_value` debe rechazarse con un mensaje claro. Los `default_value` de tipos distintos de `text`/`textarea` no pueden contener variables.
6. **REQ-006**: Resolver las variables al mostrar la información de pago en `DebtInformationView` para todos los campos de texto/textarea (no solo los `computed`), usando la cuota que se está pagando. La acción de copiar debe copiar el valor ya resuelto.
7. **REQ-007**: En el modal de configuración de la forma de pago (`DebtPaymentInformationModal`), mostrar una vista previa resuelta del valor por defecto cuando contenga variables, usando la próxima cuota de la deuda como contexto.
8. **REQ-008**: Mantener compatibilidad total con las plantillas existentes de SPEC-131: los campos `computed` siguen usando `{{token}}` con el catálogo actual en español y las plantillas sin variables se comportan igual que hoy.
9. **REQ-016**: El modal de **Forma de pago** debe abrir en **modo lectura** cuando la deuda ya tiene información guardada: muestra los valores resueltos con la próxima cuota y solo los botones **Editar** y **Cancelar**. Los inputs, la acción **Quitar forma de pago** y el guardado solo aparecen después de pulsar **Editar**; al guardar se vuelve al modo lectura con los valores actualizados.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-009**: Buscador/filtro en el panel de variables (por etiqueta española y por clave inglesa, sin distinguir mayúsculas ni acentos) cuando el catálogo supere las 10 variables.
2. **REQ-010**: Vista previa del `default_value` resuelto dentro del editor de plantillas usando un contexto sintético de demostración (deuda y cuota de ejemplo), para que el administrador vea el resultado sin asignar la plantilla.
3. **REQ-011**: Agregar la variable `today` / `today_formatted` (fecha actual del dispositivo) y variables de fecha de inicio del plan (`debt_start_date`, `debt_start_date_formatted`).
4. **REQ-012**: Permitir variables en `placeholder` y `help_text` de campos de texto/textarea, reutilizando el mismo resolvedor y la misma validación.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-013**: Permitir variables de valor numérico en campos `number` y de fecha en campos `date`, validando que el resultado resuelto sea un número o una fecha ISO válida.
2. **REQ-014**: Mecanismo de escape para llaves literales en `default_value` (por ejemplo `{{`/`}}` o `\{`), documentado en la UI de ayuda.
3. **REQ-015**: Unificar el catálogo de variables de `default_value` y el de tokens de campos `computed` en un único catálogo con alias, sin romper las plantillas existentes.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: la resolución es un reemplazo por regex sobre textos de hasta 500 caracteres por campo (máximo 30 campos por plantilla). Costo despreciable en el iHost; sin llamadas de red adicionales por render.
- **Almacenamiento**: sin migración de base de datos. Las variables viven dentro de `debt_information_templates.fields_json`, que ya existe. Sin tablas ni índices nuevos.
- **Seguridad**: los valores resueltos se muestran como texto plano (React escapa por defecto); no se interpreta HTML. La validación de variables se hace en backend (autoridad) y frontend (UX). Endpoints existentes autenticados.
- **Dependencias**: cero librerías nuevas (ni backend ni frontend). Se usan regex y utilidades existentes.
- **iHost**: sin procesos, workers ni caché adicional. El catálogo es un slice en memoria de tamaño fijo.
- **Privacidad**: ejemplos y tests usan datos sintéticos (`Lote Demo 13`, `demo-account-001`, `Proveedor Demo`). No se persisten datos personales nuevos.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Implementación de referencia (SPEC-131):

- `internal/models/debt_information.go`: `DebtInformationField` con `DefaultValue` y `ValueTemplate`; `DebtInformationToken`.
- `internal/services/debt_information.go`: catálogo `debtInformationTokens` (claves en español: `deuda.*`, `cuota.*`, `pago.*`, `sistema.*`), validación de `value_template` con `{{token}}` y validación de `default_value` por tipo.
- `internal/services/debt_information.go` → `validateDebtInformationValues`: cuando el valor viene vacío, el backend guarda el `DefaultValue` **tal cual** en `values_json` (string sin resolver). Esto confirma que la resolución debe ocurrir al mostrar.
- `frontend/src/utils/debtInformation.ts`: `buildDebtInformationContext` (contexto actual para tokens `{{...}}`), `resolveDebtInformationText` y validaciones compartidas.
- `frontend/src/pages/DebtInformationTemplateFormPage.tsx`: editor de campos, input de `default_value` y panel de tokens para campos `computed`.
- `frontend/src/components/DebtInformationView.tsx`: render de la información de pago; resuelve `computed` pero no valores literales.
- `frontend/src/components/DebtPaymentInformationModal.tsx`: aplica `defaultValueForField` y ya dispone de `debt` y `nextBill`.
- `frontend/src/components/DebtPayModal.tsx`: usa `DebtInformationView` con la cuota que se paga.

Casos de uso que motivan variables: referencias de pago o transferencia (`Lote 13 - Cuota 5`), comprobantes, mensajes de WhatsApp/email copiados desde la app y campos que repiten datos de la deuda (institución, identificador, moneda).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Reutilizar `{{token}}` de los campos `computed` también en `default_value` | Un solo dialecto de tokens | Las claves actuales están en español (`cuota.numero`) y chocan con el pedido explícito de claves en inglés; mezcla semántica de "campo calculado" con "valor por defecto" | ❌ Rechazada |
| Sintaxis `{clave_inglesa}` para `default_value` y `{{token}}` para `computed` | Respeta el pedido (`{installment_number}`), no rompe plantillas existentes, distingue visualmente ambos usos | Dos sintaxis conviviendo | ✅ Seleccionada |
| Variables resueltas al guardar la plantilla y persistir el literal | Implementación trivial | Congela el número de cuota; obliga a editar la deuda por cada cuota | ❌ Rechazada |
| Variables resueltas al aplicar la plantilla a una deuda y guardar el literal en `values_json` | El formulario muestra el valor final | El dato queda atado a la cuota del momento de configurar; no sirve para deudas con varias cuotas | ❌ Rechazada |
| Variables resueltas al renderizar (`DebtInformationView`) | El valor cambia por cuota sin reconfigurar; persiste la variable en inglés | El input de edición muestra la variable cruda (mitigado con vista previa) | ✅ Seleccionada |
| Catálogo nuevo en inglés con etiquetas en español (backend fuente de verdad) | Consistencia, validación centralizada, evolución sin tocar frontend | Un endpoint más | ✅ Seleccionada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Dos dialectos con propósitos distintos.
- **Contexto**: SPEC-131 ya publicó `{{token}}` para `computed` y hay plantillas guardadas en producción.
- **Decisión**: `default_value` usa `{clave_inglesa}` (llaves simples). `computed.value_template` conserva `{{token}}` en español. El editor rechaza `{{`/`}}` dentro de `default_value` para evitar ambigüedad.
- **Consecuencias**: no se rompe nada existente; la UI debe explicar la diferencia. Queda como deuda técnica la unificación (REQ-015, P2).

**ADR-002**: El valor por defecto se persiste sin resolver.
- **Contexto**: `values_json` se guarda por deuda y el backend ya copia el `DefaultValue` tal cual cuando el valor está vacío.
- **Decisión**: guardar `{installment_number}` y resolverlo en el render con el contexto de la cuota. Ni el backend ni el guardado necesitan conocer la deuda/cuota.
- **Consecuencias**: la lógica de resolución vive en `frontend/src/utils/debtInformation.ts`; el backend solo valida sintaxis y catálogo. La edición muestra la variable cruda (se agrega vista previa).

**ADR-003**: Catálogo de variables en inglés, etiquetas en español.
- **Contexto**: el pedido exige variables en inglés y sugerencias en español.
- **Decisión**: nueva estructura `DebtInformationVariable` (`Key`, `Label`, `Example`, `Description`) servida en `GET /api/debt-information-templates/variables`. La UI muestra `Label` en español y `{Key}` como texto monoespaciado.
- **Consecuencias**: el contrato es estable y desacopla la presentación del identificador; agregar variables no requiere migración.

**ADR-004**: Validación en ambos lados, sin bloqueo de datos históricos.
- **Contexto**: AGENTS exige validar prerequisitos en backend y frontend.
- **Decisión**: el backend rechaza variables desconocidas/mal formadas al crear o actualizar plantillas; el frontend hace la misma validación antes de enviar. Las plantillas existentes sin variables pasan sin cambios.
- **Consecuencias**: si una variable se elimina del catálogo en el futuro, las plantillas que la usan fallarían al guardar (documentado como riesgo con mitigación de alias).

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Editor de plantilla]
   │  GET /api/debt-information-templates/variables
   ▼
[Backend: catálogo + validación] ── fields_json.default_value = "Lote Demo 13 - Cuota {installment_number}"
   │
   ▼
[debt_information_templates] (SQLite, sin migración)
   │
   ▼
[Modal de forma de pago] ── aplica default_value crudo (o el guardado en values_json)
   │
   ▼
[DebtInformationView] ── resuelve {variables} con deuda + cuota ──► "Lote Demo 13 - Cuota 5"
```

### 4.2 Componentes

#### 4.2.1 Catálogo de variables (backend)
- **Responsabilidad**: definir las variables válidas, sus etiquetas en español y ejemplos; validar `default_value`.
- **Interfaz**: `DebtInformationService.Variables() []models.DebtInformationVariable` y `GET /api/debt-information-templates/variables`.
- **Dependencias**: ninguna nueva.
- **Ubicación**: `internal/models/debt_information.go`, `internal/services/debt_information.go`, `internal/api/debt_information_handlers.go`, `internal/api/routes.go`.

#### 4.2.2 Resolvedor de variables (frontend)
- **Responsabilidad**: reemplazar `{clave}` por el valor del contexto; dejar intactos los desconocidos (defensa) y exponer el catálogo fallback.
- **Interfaz**:
  - `DEBT_INFORMATION_VARIABLE_KEYS: readonly string[]`
  - `DEBT_INFORMATION_VARIABLE_FALLBACK: DebtInformationVariable[]`
  - `buildDebtInformationVariableContext(input): Record<string, string>`
  - `resolveDebtInformationVariables(text, context): string`
  - `extractDebtInformationVariables(text): string[]`
  - `validateDebtInformationDefaultValue(field): DebtInformationValidationIssue[]`
- **Dependencias**: `buildDebtInformationContext` (reutiliza formato de moneda y fechas).
- **Ubicación**: `frontend/src/utils/debtInformation.ts`.

#### 4.2.3 Editor de plantillas
- **Responsabilidad**: sugerir variables en español e insertar la clave inglesa en el `default_value`; validar antes de guardar.
- **Interfaz**: panel inline (no modal) bajo el input de valor por defecto, visible solo para `text`/`textarea`.
- **Dependencias**: `Select`/inputs existentes; tokens del tema.
- **Ubicación**: `frontend/src/pages/DebtInformationTemplateFormPage.tsx`.

#### 4.2.4 Render de información de pago
- **Responsabilidad**: resolver variables en valores de texto/textarea al mostrar y al copiar.
- **Interfaz**: `DebtInformationView` (sin cambios de props).
- **Dependencias**: `nextBill`/`bill` ya disponible.
- **Ubicación**: `frontend/src/components/DebtInformationView.tsx`, `frontend/src/components/DebtPaymentInformationModal.tsx`.

### 4.3 Modelo de datos

No hay cambios de esquema. Se agrega una estructura de dominio (no persistida):

```
Estructura: DebtInformationVariable (nueva, no persistida)
- key: string          // clave inglesa que se persiste, ej: installment_number
- label: string        // sugerencia en español, ej: "Número de cuota"
- example: string      // valor sintético, ej: "5"
- description: string  // ayuda breve, ej: "Número de la cuota que se está pagando"
```

La plantilla existente persiste el template en `fields_json.default_value`, por ejemplo:

```json
{
  "key": "referencia",
  "label": "Referencia de pago",
  "type": "text",
  "default_value": "Lote Demo 13 - Cuota {installment_number}"
}
```

### 4.4 Catálogo propuesto de variables

Todas las claves son estables y en inglés. Las etiquetas se muestran en español.

| Clave (persistida) | Etiqueta sugerida (es) | Ejemplo | Relevancia |
|---|---|---|---|
| `installment_number` | Número de cuota | `5` | Referencias "Cuota N"; caso principal del pedido |
| `installment_amount` | Monto de la cuota (sin formato) | `100.00` | Referencias, cálculos o comprobantes |
| `installment_amount_formatted` | Monto de la cuota con moneda | `$100.00` | Texto listo para copiar |
| `installment_due_date` | Vencimiento de la cuota (ISO) | `2026-12-15` | Integraciones o textos técnicos |
| `installment_due_date_formatted` | Vencimiento de la cuota (localizado) | `15/12/2026` | Mensajes y comprobantes |
| `installment_status` | Estado de la cuota | `Pendiente` | Comprobantes y avisos |
| `installment_progress` | Progreso de cuotas (`actual/total`) | `5/12` | Referencias y resúmenes |
| `installments_total` | Total de cuotas del plan | `12` | Contexto del plan |
| `debt_description` | Descripción de la deuda | `Lote Demo 13` | Repetir el dato sin escribirlo |
| `debt_identifier` | Identificador de la deuda | `demo-account-001` | Referencias bancarias |
| `institution_name` | Institución o acreedor | `Proveedor Demo` | Comprobantes y transferencias |
| `currency_code` | Código de moneda | `USD` | Textos técnicos |
| `currency_symbol` | Símbolo de moneda | `$` | Montos y referencias |
| `debt_total` | Total de la deuda (sin formato) | `1200.00` | Cálculos |
| `debt_total_formatted` | Total de la deuda con moneda | `$1,200.00` | Resúmenes |
| `payment_date` | Fecha de pago seleccionada (ISO) | `2026-12-10` | Comprobantes de pago |
| `payment_date_formatted` | Fecha de pago localizada | `10/12/2026` | Comprobantes de pago |

P1: `today`, `today_formatted`, `debt_start_date`, `debt_start_date_formatted`.

Reglas de resolución:

- Sintaxis: `{clave}` con llaves simples. Se toleran espacios internos (`{ installment_number }`) al resolver.
- Las variables desconocidas se dejan literales en el render (defensa), pero la validación de plantilla impide guardarlas.
- `installment_progress` se compone como `installment_number` + `/` + `debt.installments_total`.
- `payment_date`/`payment_date_formatted` quedan vacíos cuando todavía no hay fecha de pago (igual que `pago.fecha` en los tokens actuales).
- `installment_status` usa la etiqueta i18n de estado de cuota (`bills.status_paid` / `bills.status_pending`).
- No se resuelven variables dentro de `value_template` (`computed`): ese campo sigue usando `{{token}}`.

### 4.5 APIs / Contratos

#### Endpoint: `GET /api/debt-information-templates/variables`

**Request**: sin body. Requiere sesión autenticada (mismo middleware que el resto).

**Response 200**:
```json
[
  {
    "key": "installment_number",
    "label": "Número de cuota",
    "example": "5",
    "description": "Número de la cuota que se está pagando"
  },
  {
    "key": "installment_due_date_formatted",
    "label": "Vencimiento de la cuota (localizado)",
    "example": "15/12/2026",
    "description": "Fecha de vencimiento de la cuota en formato local"
  }
]
```

#### Endpoint existente: `POST/PUT /api/debt-information-templates`

Se mantiene el contrato. `fields[].default_value` ahora admite variables para tipos `text`/`textarea`.

**Response Error** (nuevo código `invalid_variable`):
```json
{
  "error": "invalid_variable",
  "message": "La variable {installment_numer} no es válida en el campo 'Referencia de pago'."
}
```

Reglas de aceptación del backend:
- `text`/`textarea`: hasta 500 caracteres en el `default_value` crudo; cada `{clave}` debe existir en el catálogo; sin `{{`/`}}`; sin llaves sin cerrar.
- `number`, `date`, `select`, `checkbox`: si el `default_value` contiene el patrón `{...}` se rechaza con `invalid_variable`.
- `computed`: sin cambios; sigue validando `{{token}}` contra `debtInformationTokenSet`.

### 4.6 Dependencias

- **Internas**: SPEC-131 (plantillas de información), SPEC-054 (deudas y cuotas), SPEC-122/126 (reglas de modales, solo si se usara un modal; el panel propuesto es inline).
- **Externas**: ninguna.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado el editor de plantillas, cuando se crea un campo `text` con valor por defecto `Lote Demo 13 - Cuota {installment_number}`, entonces la plantilla se guarda con el texto **sin resolver** y la clave inglesa en `fields_json`.
- [ ] CA-002: Dado una deuda con la plantilla de CA-001 y la cuota número 5, cuando se abre el modal de pago de esa cuota, entonces la información muestra `Lote Demo 13 - Cuota 5`; en la cuota 6 muestra `Lote Demo 13 - Cuota 6` sin editar la deuda.
- [ ] CA-003: Dado un `default_value` con una variable inexistente (`{installment_numer}`), cuando se intenta guardar desde la UI o por API, entonces se rechaza con un mensaje que nombra la variable y el campo.
- [ ] CA-004: Dado un campo `number`/`date`/`select`/`checkbox`, cuando su `default_value` contiene `{...}`, entonces la UI y la API lo rechazan.
- [ ] CA-005: Dado el panel de sugerencias, cuando el usuario busca "cuota" o "vencimiento" en español, entonces ve las variables correspondientes con su etiqueta en español y su clave inglesa, y al hacer click se inserta `{clave}` en la posición del cursor.
- [ ] CA-006: Dado un valor resuelto en `DebtInformationView`, cuando el usuario usa "Copiar", entonces el portapapeles recibe el texto resuelto (no la variable).
- [ ] CA-007: Dado el modal de forma de pago con un `default_value` con variables, cuando se selecciona la plantilla, entonces se muestra una vista previa resuelta con la próxima cuota.
- [ ] CA-008: Dadas las plantillas existentes de SPEC-131, cuando se listan, editan y usan sin cambios, entonces siguen funcionando igual (incluidos los campos `computed` con `{{token}}`).
- [ ] CA-009: Dado un `default_value` con llaves dobles (`{{...}}`), cuando se guarda, entonces se rechaza con un mensaje que indica que ese campo usa `{variable}`.
- [ ] CA-010: Dada una deuda con forma de pago guardada, cuando se abre **Forma de pago**, entonces se ve la información resuelta en modo lectura y solo los botones **Editar** y **Cancelar**, sin inputs ni **Quitar forma de pago**.
- [ ] CA-011: Dado el modo lectura de CA-010, cuando se pulsa **Editar**, entonces aparecen los inputs, **Quitar forma de pago**, **Volver** y **Guardar**; al guardar se vuelve al modo lectura con los valores actualizados.
- [ ] CA-DARK: El panel de variables, el input de valor por defecto y la vista previa usan tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y se verifican en darkmode (texto, placeholders y chips legibles).
- [ ] CA-MODAL: No se agregan modales nuevos (el panel de variables es inline). Si durante la implementación se optara por un modal, debe cumplir SPEC-126: sin cierre por click fuera, overlay bloqueante, cierre visible en todos los estados y escala de capas.
- [ ] CA-BACK: No se crean páginas de detalle nuevas; no aplica `BACK_ROUTES`.

### 5.2 No funcionales

- [ ] CA-NF-001: La resolución de variables no agrega dependencias ni llamadas de red por render; el costo es O(campos × largo del texto) y despreciable en el iHost.
- [ ] CA-NF-002: No se agregan migraciones ni tablas; el esquema SQLite de `debt_information_templates` y `debt_information` no cambia.
- [ ] CA-NF-003: `go test ./...` pasa y `npm run build` (incluye `check:ui` y `check:pwa`) pasa sin errores.
- [ ] CA-NF-004: Los ejemplos de la spec, tests y UI usan datos sintéticos; no se incorporan datos privados ni secretos.

### 5.3 Testing

- **Unit tests (Go)**: `validateDebtInformationVariableTemplate` (válida, desconocida, mal formada, llaves dobles); rechazo de variables en tipos no textuales; creación/actualización de plantilla con `default_value` con variables; listado del catálogo `Variables()`.
- **Integration tests (Go)**: `POST/PUT /api/debt-information-templates` con variable válida e inválida; `GET /api/debt-information-templates/variables`.
- **Manual (frontend)**: crear plantilla con `Lote Demo 13 - Cuota {installment_number}`, asignarla a una deuda con varias cuotas, abrir el pago de cuotas distintas y verificar que el número cambia; copiar y pegar; verificar darkmode y responsive.
- **Carga/Performance**: no aplica prueba de carga; verificar que el render del modal sigue siendo instantáneo en el iHost con una plantilla de 30 campos.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Modelo `DebtInformationVariable`, catálogo backend, endpoint `variables` y validación de `default_value` | 0.5 día | Ninguna |
| 2 | Utilidades frontend: contexto inglés, resolvedor, extracción y validación; tests | 0.5 día | Fase 1 |
| 3 | Editor de plantillas: panel de sugerencias con etiquetas en español, inserción en cursor y mensajes de error | 1 día | Fase 2 |
| 4 | Render: `DebtInformationView`, copia resuelta y vista previa en `DebtPaymentInformationModal` | 0.5 día | Fase 2 |
| 5 | i18n (`frontend/public/i18n/es.json` y `en.json`), build, pruebas manuales y validación en darkmode | 0.5 día | Fases 3 y 4 |

### 6.2 Milestones

1. **MVP**: variables en `default_value` de `text`/`textarea`, catálogo mínimo (`installment_number`, `installment_amount`, `installment_due_date`, `debt_description`, `debt_identifier`, `institution_name`), validación y render.
2. **V1.0**: catálogo completo de la sección 4.4, buscador en el panel, vista previa en editor y en forma de pago.
3. **P1/P2**: `today`, fechas de inicio, variables en placeholder/help_text, tipos numéricos/fecha y escape de llaves.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Confusión entre `{variable}` y `{{token}}` | Media | Medio | Ayuda contextual en la UI, validación que rechaza llaves dobles en `default_value` y documentación en la spec |
| Errores de tipeo en variables | Alta | Bajo | Validación bloqueante en backend y frontend con mensaje que nombra la variable |
| Variables renombradas en el futuro rompen plantillas guardadas | Baja | Alto | Mantener claves estables y alias de compatibilidad; documentar en el catálogo |
| El valor resuelto supera el límite de 500 caracteres del campo | Media | Bajo | Validar longitud del texto resuelto en el render/edición y truncar solo visualmente; documentar |
| Datos sensibles dentro de valores por defecto | Baja | Medio | Los defaults son configuración, no datos de usuario; mantener ejemplos sintéticos y evitar PII en tests |
| Regresión en plantillas de SPEC-131 | Media | Alto | Tests de compatibilidad y validación de que `computed` sigue usando `{{token}}` |

## 8. Notas y Referencias

- SPEC-131: `docs/specs/SPEC-131-forma-pago-plantillas-informacion-deudas.md`
- SPEC-054: modelo de deudas y cuotas (`debts`, `debt_bills`)
- SPEC-126: reglas de modales (solo aplicable si se agrega un modal)
- Código base:
  - `internal/models/debt_information.go`
  - `internal/services/debt_information.go`
  - `internal/api/debt_information_handlers.go`
  - `internal/api/routes.go`
  - `frontend/src/utils/debtInformation.ts`
  - `frontend/src/pages/DebtInformationTemplateFormPage.tsx`
  - `frontend/src/components/DebtInformationView.tsx`
  - `frontend/src/components/DebtPaymentInformationModal.tsx`
  - `frontend/src/i18n/es.json`, `frontend/src/i18n/en.json` (fuente de verdad: `frontend/public/i18n/`)

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-09 | opencode | Creación inicial de la especificación |
| 2026-11-09 | opencode | Implementación: catálogo de variables, endpoint `GET /api/debt-information-templates/variables`, validación en backend y frontend, panel de sugerencias en el editor, resolución en `DebtInformationView`, vista previa en el modal de pago y vista previa con datos de ejemplo en el editor de plantillas (REQ-010). Pendiente de validación manual. |
| 2026-11-09 | opencode | Ajuste de UX solicitado por el usuario (REQ-016): el modal de Forma de pago abre en modo lectura cuando ya hay información guardada, con botones Editar y Cancelar; inputs y Quitar forma de pago solo en modo edición; al guardar vuelve al modo lectura. |
