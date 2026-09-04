---
title: "Fix global de texto de inputs en darkmode en todos los formularios + regla para futuras specs"
id: "SPEC-060"
status: "released"
author: "paulomcnally"
created: "2026-09-04"
updated: "2026-09-04"
github_issue: 61
---

# Fix global de texto de inputs en darkmode en todos los formularios + regla para futuras specs

**ID**: SPEC-060  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-09-04  
**Actualizado**: 2026-09-04

---

## 1. Resumen Ejecutivo

En el formulario de deudas se corrigió la experiencia de usuario del texto de los inputs en darkmode, pero el fix se aplicó de forma puntual (agregando `bg-card` a los inputs de `DebtFormPage.tsx` y `DebtPayModal.tsx`). **Todos los demás formularios del sistema quedaron afectados por el mismo bug**: en modo oscuro el texto de los inputs es ilegible (texto negro sobre fondo oscuro). Esta spec resuelve el problema de raíz de forma global y agrega una regla al repositorio para que **no vuelva a ocurrir** en formularios futuros.

La causa raíz: SPEC-053 movió la paleta a CSS variables y setea `body { color: rgb(var(--color-text)) }`, pero los form controls nativos (`input`, `select`, `textarea`) **no heredan** `color` del `body` (usan el color por defecto del browser, `fieldtext` = negro). En darkmode el fondo de los inputs se vuelve oscuro (`bg-card` → `rgb(28 28 30)`, o `dark:bg-[#2c2c2e]` en SettingsPage) mientras el texto sigue negro → texto invisible. Además, la mayoría de los inputs del sistema no tienen un fondo explícito en darkmode.

Impacto iHost: fix 100% frontend, un puñado de reglas CSS y clases en componentes. Cero dependencias nuevas, cero impacto en memoria/CPU/SQLite. El backend no se toca.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Corregir el color de texto de **TODOS** los `input`, `select` y `textarea` en darkmode en todos los formularios y modales del frontend (no solo el formulario de deudas). La corrección debe aplicarse a nivel global en `frontend/src/index.css` para cubrir formularios existentes y futuros.
2. **REQ-002**: Agregar regla CSS global que fuerce `color: rgb(var(--color-text))` sobre `input, select, textarea` (y `::placeholder` con el color secundario del tema), con `color-scheme: light dark` para que los controles nativos (date pickers, dropdowns nativos, autofill) se rendericen correctamente en darkmode.
3. **REQ-003**: Estandarizar el fondo de los inputs: los que no tienen fondo o usan `bg-white dark:bg-[#2c2c2e]` hardcodeado (p. ej. `inputCls` de `SettingsPage.tsx:396`) deben usar el token del tema `bg-card`, igual que el formulario de deudas (`DebtFormPage.tsx`).
4. **REQ-004**: Agregar en `AGENTS.md` (sección "Reglas Fundamentales") una regla permanente que obligue: todo input/select/textarea DEBE usar los tokens de color del tema (`bg-card`, `text-text`), **jamás** colores hardcodeados, y **todo formulario nuevo debe verificarse en darkmode** antes de considerarse completo.
5. **REQ-005**: Agregar la misma regla en `docs/project-rules.md` (sección 4 Reglas de UI) y en `docs/specs/templates/spec-template.md` (criterios de aceptación / consideraciones) para que el requisito de darkmode quede en el checklist de toda spec que cree formularios.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-006**: Verificar y corregir los componentes custom y modales con inputs: `Select.tsx` (trigger button + search input), `PayBillModal.tsx`, `DebtPayModal.tsx`, `UploadBillModal.tsx`, `AddInsuranceModal.tsx`, `EmailRecipientsModal.tsx`, `DeleteModal.tsx`, `IconPickerModal.tsx`, `RegistrosPage.tsx`, `SettingsPage.tsx`.
2. **REQ-007**: Verificar placeholders en darkmode: `::placeholder` debe usar `--color-text-secondary` para contraste legible en ambos modos.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-008**: Centralizar la clase de inputs en una constante/helper compartido (p. ej. `frontend/src/utils/ui.ts` con `inputCls`) para que todos los formularios usen exactamente el mismo set de clases y el patrón sea auditable por grep.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Sin impacto. Reglas CSS estáticas; sin JS adicional en runtime.
- **Seguridad**: Sin cambios de datos ni de APIs.
- **Almacenamiento**: Sin cambios en SQLite. Cero migraciones.
- **Disponibilidad**: El darkmode queda consistente en toda la app; no hay regresión en modo claro.
- **iHost**: Cero dependencias nuevas. Bundle crece ~0.1 KB (reglas CSS). Backend intacto.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **SPEC-053** (commit `e343a5b`) movió la paleta a CSS variables y definió `body { color: rgb(var(--color-text)) }` en `frontend/src/index.css:35-40`. En darkmode `--color-text: 242 242 247` (claro) y `--color-card: 28 28 30` (oscuro).
- **Bug raíz**: los form controls nativos no heredan `color` del `body`. Chrome/Firefox/Safari aplican su color por defecto (`fieldtext`, negro) a `input`/`select`/`textarea` vía UA stylesheet. Por eso en darkmode el texto queda negro sobre fondo oscuro aunque el `body` tenga `color` claro.
- **Formulario de deudas (SPEC-054)**: `DebtFormPage.tsx` (líneas 183, 193, 207, 218, 254, 265, 278, 290, 300) y `DebtPayModal.tsx` (77, 87) usan `... bg-card` en los inputs. Es el patrón correcto de fondo, pero el color de texto no está resuelto ahí — el fix real tiene que ser la regla global de `color`.
- **Inputs afectados (sin `bg-card` y sin color explícito)**: `ServiceFormPage`, `InstitutionFormPage`, `HijoFormPage`, `NotificationFormPage`, `SalaryFormPage`, `CategoryFormPage`, `CurrencyFormPage`, `BillFormPage`, `AutoFormPage`, `LoginPage`, `SetupPage`, `DeleteModal`, `EmailRecipientsModal`, `IconPickerModal`, `UploadBillModal` (algunos inputs), `RegistrosPage` (`inputCls` línea 78).
- **Fondo hardcodeado**: `SettingsPage.tsx:396` `inputCls = '... bg-white dark:bg-[#2c2c2e] ...'`.
- **Componente Select custom** (`Select.tsx`): el trigger es un `<button>` con `bg-card` y el label seleccionado no tiene clase de color → mismo problema de texto oscuro sobre fondo oscuro en darkmode.
- **Solución estándar web**: `color-scheme: light dark` + forzar `color` sobre form controls. Es la práctica recomendada para darkmode con controles nativos (color picker, date, autofill, scrollbars).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A: Regla CSS global `input, select, textarea { color: var(--color-text); background-color: var(--color-card); color-scheme: light dark; }` | Un solo punto de verdad; cubre TODOS los formularios presentes y futuros; elimina la clase de error | Hay que verificar inputs con fondos especiales (búsquedas en modales) | ✅ Seleccionada |
| B: Agregar `text-text` a cada input de cada formulario (~60 lugares) | Explícito por formulario | Repite el patrón que ya falló (fix puntual); cualquier formulario nuevo vuelve a romperse | ❌ Rechazada |
| C: Sólo tocar `SettingsPage` y dejar el resto | Rápido | No resuelve el resto de formularios ni el problema de fondo | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: El fix de color de texto es global (CSS), no por formulario.
- **Contexto**: El error ocurrió porque el fix fue puntual (solo deudas). Forzar el color vía CSS a nivel de `index.css` garantiza que ningún formulario presente o futuro pueda romperse.
- **Decisión**: Regla en `frontend/src/index.css` sobre `input, select, textarea` + `::placeholder` + `color-scheme: light dark`.
- **Consecuencias**: Un solo lugar de verdad. Los formularios que ya tenían `bg-card` quedan correctos automáticamente; los que no, heredan el fondo del token de card.

**ADR-002**: Fondo de inputs estandarizado con token `bg-card`, eliminando `bg-white dark:bg-[#2c2c2e]`.
- **Contexto**: `bg-card` es la variable del tema que se adapta en ambos modos (patrón del formulario de deudas). El hex `#2c2c2e` hardcodeado en `SettingsPage` es frágil y duplica la definición.
- **Decisión**: Reemplazar los fondos hardcodeados por `bg-card` (o dejar que la regla global los cubra).
- **Consecuencias**: Consistencia total de inputs en ambos modos; menos clases duplicadas.

**ADR-003**: Regla permanente en AGENTS.md + project-rules.md + template de specs.
- **Contexto**: Este error es un defecto recurrente de proceso: un fix puntual sin regla que lo generalice. La regla debe vivir en las fuentes de verdad del proyecto.
- **Decisión**: Documentar en AGENTS.md (Reglas Fundamentales) y project-rules.md (Reglas de UI), y en el template de specs como criterio de aceptación para specs con formularios.
- **Consecuencias**: El checklist de calidad obliga a verificar darkmode en formularios; si se omite, la spec no pasa a pending_release.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[frontend/src/index.css]  --regla global-->  input, select, textarea { color, bg-card, color-scheme }
        |                                                      |
        v                                                      v
[AGENTS.md + project-rules.md + spec-template]   [Todos los formularios y modales]
        |   (regla de proceso: verificar darkmode)                  |
        v                                                          v
[Futuras specs con formularios]  ------------------------->  [inputs legibles en darkmode]
```

### 4.2 Componentes

#### 4.2.1 Regla CSS global en `frontend/src/index.css`
- **Responsabilidad**: Fijar color de texto, fondo y color-scheme de form controls.
- **Interfaz**: CSS puro:
  ```css
  input, select, textarea {
    color: rgb(var(--color-text));
    background-color: rgb(var(--color-card));
    color-scheme: light dark;
  }
  input::placeholder, textarea::placeholder {
    color: rgb(var(--color-text-secondary));
  }
  ```
- **Dependencias**: Variables CSS ya definidas (`--color-text`, `--color-card`, `--color-text-secondary`).
- **Ubicación**: `frontend/src/index.css`.

#### 4.2.2 Estandarización de clases de inputs
- **Responsabilidad**: Reemplazar fondos hardcodeados por tokens del tema.
- **Interfaz**: `SettingsPage.tsx:396` → `inputCls = 'w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary bg-card min-h-[44px]'`. En el resto de formularios, la regla global ya cubre el fondo; opcionalmente agregar `bg-card` explícito para consistencia (P2).
- **Ubicación**: `frontend/src/pages/SettingsPage.tsx`, y revisión de los inputs listados en 4.5.

#### 4.2.3 Documentación (reglas)
- **Responsabilidad**: Evitar recurrencia.
- **Interfaz**: Entradas en `AGENTS.md`, `docs/project-rules.md`, `docs/specs/templates/spec-template.md`.
- **Ubicación**: raíz y `docs/`.

### 4.3 Modelo de datos

Sin cambios. No hay migraciones, ni nuevas tablas, ni claves de settings.

### 4.4 APIs / Contratos

Sin cambios. No hay endpoints nuevos ni modificados.

### 4.5 Alcance de corrección en frontend (inputs afectados)

| Archivo | Situación |
|---------|-----------|
| `frontend/src/pages/ServiceFormPage.tsx` | inputs sin fondo (transparente → oscuro en darkmode) |
| `frontend/src/pages/InstitutionFormPage.tsx` | inputs sin fondo |
| `frontend/src/pages/HijoFormPage.tsx` | inputs sin fondo |
| `frontend/src/pages/NotificationFormPage.tsx` | inputs sin fondo |
| `frontend/src/pages/SalaryFormPage.tsx` | inputs sin fondo |
| `frontend/src/pages/CategoryFormPage.tsx` | inputs sin fondo |
| `frontend/src/pages/CurrencyFormPage.tsx` | inputs sin fondo |
| `frontend/src/pages/BillFormPage.tsx` | inputs sin fondo |
| `frontend/src/pages/AutoFormPage.tsx` | inputs sin fondo |
| `frontend/src/pages/LoginPage.tsx` | inputs sin fondo |
| `frontend/src/pages/SetupPage.tsx` | inputs sin fondo |
| `frontend/src/pages/RegistrosPage.tsx` | `inputCls` (línea 78) sin fondo |
| `frontend/src/pages/SettingsPage.tsx` | `inputCls` (línea 396) con `bg-white dark:bg-[#2c2c2e]` |
| `frontend/src/components/DeleteModal.tsx` | input sin fondo |
| `frontend/src/components/EmailRecipientsModal.tsx` | input sin fondo |
| `frontend/src/components/IconPickerModal.tsx` | input de búsqueda sin fondo |
| `frontend/src/components/UploadBillModal.tsx` | algunos inputs sin fondo (líneas 318, 347, 356) |
| `frontend/src/components/PayBillModal.tsx` | inputs con `bg-card` (ok) |
| `frontend/src/components/DebtPayModal.tsx` | inputs con `bg-card` (ok) |
| `frontend/src/components/Select.tsx` | trigger `<button>` + search input (verificar texto) |
| `frontend/src/components/DebtAnalysis.tsx` | input de búsqueda con `bg-card` (línea 325, ok) |

> Nota: con la regla global de la opción A, la columna "sin fondo" queda resuelta automáticamente vía `background-color: rgb(var(--color-card))`; el trabajo por archivo es verificación visual y limpieza de clases hardcodeadas.

### 4.6 Dependencias

- **Internas**: `frontend/src/index.css`, `frontend/src/pages/SettingsPage.tsx`, todos los archivos de 4.5, `AGENTS.md`, `docs/project-rules.md`, `docs/specs/templates/spec-template.md`.
- **Externas**: Ninguna (regla iHost: sin dependencias nuevas).

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: En darkmode, el texto es legible en los inputs de TODOS los formularios: Deudas, Servicios, Instituciones, Hijos, Notificaciones, Salarios, Categorías, Monedas, Facturas, Autos, Login, Setup, Settings, Registros.
- [x] CA-002: En darkmode, el texto es legible en los inputs de TODOS los modales: PayBill, DebtPay, UploadBill, AddInsurance, EmailRecipients, Delete, IconPicker, Select (búsqueda y valor seleccionado).
- [x] CA-003: En darkmode, los placeholders usan `--color-text-secondary` (legibles pero distinguibles del texto escrito).
- [x] CA-004: En darkmode, los controles nativos (input de fecha, calendario del navegador, autofill) se renderizan con esquema oscuro (`color-scheme: light dark`).
- [x] CA-005: En modo claro no hay regresión visual en ningún input (mismos colores que antes).
- [x] CA-006: `SettingsPage` deja de usar `bg-white dark:bg-[#2c2c2e]` y usa `bg-card`.
- [x] CA-007: Existe la regla en `AGENTS.md` que obliga a usar tokens de color del tema en inputs y verificar darkmode en formularios nuevos.
- [x] CA-008: Existe la regla en `docs/project-rules.md` y en el template de specs (criterios de aceptación para specs con formularios).

### 5.2 No funcionales

- [x] CA-NF-001: El bundle del frontend no crece de forma apreciable (< 1 KB) y no se agregan dependencias.
- [x] CA-NF-002: El build de Vite (`npm run build` en `frontend/`) pasa sin errores y el server local sirve el CSS actualizado.

### 5.3 Testing

- **Unit tests**: N/A (cambio CSS). Si se centraliza `inputCls` (REQ-008), no requiere test.
- **Integration tests**: N/A (sin API).
- **E2E tests**: Prueba manual en local con darkmode activo: recorrer cada formulario/modal y verificar legibilidad de texto y placeholder. Repetir en modo claro.
- **Carga/Performance**: N/A.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Regla CSS global en `index.css` (color + background + `color-scheme` + placeholder) | 0.25 día | Ninguna |
| 2 | Limpieza de clases hardcodeadas (`SettingsPage` `inputCls` → `bg-card`) | 0.25 día | Fase 1 |
| 3 | Verificación visual de todos los formularios/modales en darkmode y modo claro; ajustes puntuales | 0.5 día | Fases 1-2 |
| 4 | Reglas en `AGENTS.md`, `docs/project-rules.md` y `docs/specs/templates/spec-template.md` | 0.25 día | Ninguna |
| 5 | Correr server en local (`npm run build` + backend) para validación del usuario | 0.25 día | Fases 1-4 |

**Estimación total**: ~1.5 días.

### 6.2 Milestones

1. **MVP**: Fases 1-3 (fix global de inputs en darkmode).
2. **V1.0**: Fases 4-5 (reglas de proceso + validación con el usuario).

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| La regla global rompe algún input con estilo especial (p. ej. filtros sobre fondos de color) | Media | Medio | Verificación visual de todos los inputs; los inputs con `bg-card`/fondos propios mantienen su clase y la regla solo agrega color de texto |
| `color-scheme: light dark` cambia la apariencia de date pickers/selects nativos | Baja | Bajo | Es el comportamiento deseado en darkmode; se verifica en CA-004 |
| El build de Vite no regenera `public/` y se prueba contra CSS viejo | Baja | Medio | Regla AGENTS.md: correr `npm run build` y verificar con el server local |
| Se vuelve a hacer un fix puntual en una sola página en el futuro | Media | Alto | Regla permanente en AGENTS.md + template de specs; revisión en code review |

## 8. Notas y Referencias

- Regresión reportada: fix puntual de darkmode en el formulario de deudas (SPEC-054) dejó afectados al resto de formularios.
- Precedente SPEC-053: modo oscuro configurable con paleta a CSS variables (`--color-*`).
- Referencia técnica: `color-scheme` (MDN) y herencia de `color` en form controls.
- Reglas AGENTS.md aplicables: "Seguir siempre el patrón de UI existente", "La fuente de verdad del i18n es `frontend/public/i18n/`".

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-04 | paulomcnally | Creación inicial de la especificación |
| 2026-09-04 | paulomcnally | Implementación: regla CSS global en `index.css` (color + background + `color-scheme` + placeholder), `SettingsPage` usa `bg-card`, `Select.tsx` trigger con `text-text`, reglas permanentes en AGENTS.md/project-rules.md/spec-template. Released en main (frontend-only, sin backend ni DB). |