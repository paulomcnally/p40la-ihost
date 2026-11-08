---
title: "Modales del proyecto: confirmación por palabra con Enter y overlay bloqueante sin cierre por click fuera"
id: "SPEC-126"
status: "in_progress"
author: "opencode"
created: "2026-11-07"
updated: "2026-11-07"
github_issue: 129
---

# Modales del proyecto: confirmación por palabra con Enter y overlay bloqueante sin cierre por click fuera

**ID**: SPEC-126
**Estado**: in_progress
**Autor**: opencode
**Creado**: 2026-11-07
**Actualizado**: 2026-11-07

---

## 1. Resumen Ejecutivo

SPEC-122 dejó documentadas y aplicadas parcialmente dos reglas de modales que deben regir en **todo el proyecto**:

1. **Confirmación por palabra**: todo modal que habilita una acción destructiva o irreversible escribiendo una palabra (`confirmo`, `reabrir`/`reopen`, etc.) debe enfocar el input automáticamente al abrirse y permitir confirmar con **Enter** cuando la palabra es válida, ejecutando exactamente lo mismo que el botón.
2. **Overlay y cierre explícito**: ningún modal se cierra al hacer click fuera de su contenido; el overlay debe bloquear la interacción con la página de fondo (incluidos los menús de 3 puntos) y el cierre debe ser siempre una acción visible (X o Cancelar/Cerrar/Listo).

La auditoría de `main` (commit `21ef61c`, release de SPEC-122) muestra que la regla de confirmación por palabra se cumple en `DeleteModal` (23 usos en 19 archivos), pero **no** en el modal de reabrir mes de `RegistrosPage`, que tiene `autoFocus` pero no maneja Enter. Por otro lado, **ningún modal de `main` se cierra al hacer click fuera** (solo el drawer móvil del sidebar, que no es un modal), pero **23 de los 29 overlays de modales usan `z-50`** y podrían quedar al mismo nivel que los menús de 3 puntos, y **3 modales no tienen una acción de cierre siempre visible** (`AddInsuranceModal`, `AddPolicyModal` y `WebhookModal`).

Esta spec completa la aplicación de ambas reglas en todos los modales existentes de `main`, centraliza la confirmación por palabra en un hook reutilizable y define una escala de capas (`z-index`) para que los overlays, los dropdowns `Select` y los toasts sigan funcionando correctamente. Es un cambio exclusivamente de frontend: no toca API, base de datos, migraciones ni variables de entorno. En el iHost el impacto es nulo en memoria y almacenamiento porque no se agregan dependencias ni procesos; el bundle solo crece por un hook liviano y ajustes de clases.

Consideraciones de UI obligatorias:
- Todo `input`/`select`/`textarea` DEBE usar los tokens de color del tema (`bg-card`, `text-text`, `text-text-secondary`). Jamás colores hardcodeados (ej: `bg-white`, `dark:bg-[#2c2c2e]`).
- Todo formulario o modal modificado debe verificarse en darkmode: texto y placeholders legibles.
- Ningún modal se cierra al hacer click fuera; el overlay bloquea la interacción con el fondo y el cierre es solo con **Cancelar/Cerrar** o la **X**.
- Los modales de confirmación por palabra deben quedar por encima del modal que los abre.

## 2. Requerimientos

### 2.1 Regla A: confirmación por palabra (P0 - Obligatorios)

1. **REQ-001**: La regla aplica a **todo modal de `main` que tenga un input de texto cuyo valor habilita la acción de confirmación**, sin importar la palabra esperada. Casos actuales:
   - `DeleteModal` con la palabra `confirmo` (23 usos en 19 archivos).
   - Modal de reabrir mes de `RegistrosPage` con la palabra `reabrir` (es) / `reopen` (en).
   - Cualquier modal futuro con el mismo patrón.
2. **REQ-002**: Al abrir el modal, el input de confirmación recibe **foco automáticamente**. La implementación debe usar `ref` + `useEffect` (o un hook compartido que lo haga), no depender solo del atributo HTML `autoFocus`, para que el foco sea confiable en desktop y móvil.
3. **REQ-003**: La palabra se compara normalizada con `trim()` y en minúsculas. El botón de confirmación se habilita únicamente cuando el texto normalizado coincide exactamente con la palabra esperada.
4. **REQ-004**: Al presionar **Enter** en el input, si el botón está habilitado (palabra válida y acción no en curso), se ejecuta **el mismo callback** que el botón de confirmación. Si la palabra es inválida o la acción está en curso, Enter no ejecuta nada.
5. **REQ-005**: El handler de Enter debe llamar a `preventDefault()` para evitar envíos de formulario o navegaciones accidentales. Solo debe invocar la acción cuando corresponda.
6. **REQ-006**: No se debe disparar la acción dos veces por una misma pulsación de Enter ni permitir un segundo envío mientras la operación está en curso. El modal de reabrir mes ya tiene `busy`; el guard debe cubrir también la ventana entre la pulsación y el re-render.
7. **REQ-007**: Centralizar la lógica de foco, normalización y Enter en un hook reutilizable de frontend (`frontend/src/hooks/useConfirmWordInput.ts`), usado por `DeleteModal` y por el modal de reabrir mes. Sin dependencias nuevas.
8. **REQ-008**: Auditar **todos los modales de `main`** y corregir los que no cumplan la regla. El inventario de esta spec debe quedar verificado en el código:
   - `frontend/src/components/DeleteModal.tsx`: ya cumple foco + Enter desde SPEC-122; se refactoriza al hook sin cambiar su API pública ni su comportamiento visual.
   - `frontend/src/pages/RegistrosPage.tsx` (modal de reabrir mes): **no cumple Enter**; se corrige.
   - No existen otros modales con input de confirmación por palabra en `main` al momento de crear esta spec.
9. **REQ-009**: Mantener el comportamiento existente de cierre y overlays: click fuera no cierra, overlay bloquea el fondo, cierre solo con **Cancelar/Cerrar** o **X**, y el modal de confirmación queda por encima del modal que lo abre.
10. **REQ-010**: Al cerrar o cancelar el modal, el texto de confirmación debe quedar limpio (o el componente desmontarse) para que al reabrirlo no aparezca una palabra vieja que habilite la acción sin querer.
11. **REQ-011**: Ignorar Enter cuando el evento provenga de una composición IME (`e.nativeEvent.isComposing`) para no confirmar por accidente al escribir con métodos de entrada.

### 2.2 Regla B: overlay, click fuera y cierre explícito (P0 - Obligatorios)

1. **REQ-020**: **Ningún modal se cierra al hacer click fuera de su contenido.** El overlay NO debe tener `onClick` (ni `onMouseDown`) que invoque `onClose`. El cierre debe ser una acción explícita y visible.
2. **REQ-021**: El overlay de todo modal debe ser `fixed inset-0`, cubrir todo el viewport, bloquear la interacción con la página de fondo (sin `pointer-events-none`) y usar **`z-[60]` o superior**.
3. **REQ-022**: Todo modal debe tener **siempre** una acción visible de cierre (X o Cancelar/Cerrar/Listo), incluso:
   - mientras carga datos (`loading`),
   - mientras guarda (`saving`),
   - antes de seleccionar una entidad (por ejemplo, antes de elegir un servicio o un auto).
   Los botones pueden estar deshabilitados durante una operación en curso, pero no deben desaparecer.
4. **REQ-023**: Definir y aplicar una escala de capas consistente:
   - `z-50`: menús de 3 puntos (`CardMenu`), `CreateMenu`, header y sidebar.
   - `z-[60]`: overlay y contenido de un modal base.
   - `z-[70]`: modal abierto desde otro modal o picker (por ejemplo, `DeleteModal` sobre `DebtDocumentModal`, `ServicePickerModal` sobre un formulario modal).
   - `z-[80]`: confirmación por encima de un modal que ya está en `z-[70]` o dropdown explícito dentro de un modal anidado.
   - `z-[90]`: dropdown del componente `Select` (portal), para que nunca quede detrás del overlay.
   - `z-[100]`: toasts globales, para que sigan visibles por encima de cualquier modal.
5. **REQ-024**: Auditar y corregir **los 29 overlays de modales de `main`**. Los 23 que hoy usan `z-50` (o el default dinámico de `IconPickerModal`) deben pasar a `z-[60]` o superior. Los 6 que ya están en `z-[60]`/`z-[70]` se mantienen.
6. **REQ-025**: Ajustar las capas globales para evitar regresiones:
   - `frontend/src/components/Select.tsx`: cambiar el default de `zIndex` de `z-50` a `z-[90]` (los overrides explícitos existentes se conservan).
   - `frontend/src/components/Toast.tsx`: cambiar `z-50` a `z-[100]`.
   - `frontend/src/components/IconPickerModal.tsx`: cambiar el default de `zIndex` de `z-50` a `z-[60]`; los usos anidados que ya pasan `z-[70]`/`z-[80]` se mantienen.
7. **REQ-026**: Corregir los modales sin acción de cierre siempre visible:
   - `frontend/src/components/AddInsuranceModal.tsx`: hoy el botón Cancelar solo aparece después de seleccionar un servicio. Agregar una **X** fija en el header (o un Cancelar siempre visible).
   - `frontend/src/components/AddPolicyModal.tsx`: mismo caso; el Cancelar solo aparece después de seleccionar un auto. Agregar una **X** fija en el header.
   - `frontend/src/components/WebhookModal.tsx`: hoy no hay cierre durante `loading`; agregar una **X** fija en el header, independiente de los estados.
8. **REQ-027**: Verificar que el overlay bloquea los menús de 3 puntos y cualquier otro control de fondo (incluidos `CardMenu`, `CreateMenu` y el sidebar), usando `z-[60]` o superior. No alcanza con que el modal se vea por encima: el fondo no debe recibir clicks.
9. **REQ-028**: El drawer móvil del sidebar (`frontend/src/components/DashboardLayout.tsx`) **no es un modal** y se excluye de esta regla: conserva su cierre por click fuera (`z-40` de overlay, `z-50` del panel). La spec debe dejar explícita esta exclusión para que no se confunda con un modal.
10. **REQ-029**: Actualizar la documentación para que ambas reglas queden generalizadas:
    - `AGENTS.md` (regla crítica de modales).
    - `docs/project-rules.md` §4.7.
    - `docs/specs/templates/spec-template.md`.
    - `.opencode/skills/spec-manager/SKILL.md` (reglas de documentación).
11. **REQ-030**: Mantener el comportamiento de accesibilidad y teclado existente: foco en el input de confirmación, `Escape` donde ya está implementado, labels y botones con `title`/`aria-label`. No se exige focus trap en esta spec.

### 2.3 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-031**: El guard de operación en curso de la confirmación por palabra debe implementarse con una referencia (`useRef`) o equivalente, además del estado `busy`, para cubrir pulsaciones repetidas antes del re-render. Se libera en `finally`.
2. **REQ-032**: El hook de confirmación debe exponer la validez calculada (`isValid`) para que el botón use exactamente la misma condición que el handler de Enter y no haya divergencias.
3. **REQ-033**: El foco automático no debe romper el foco de otros controles dentro del modal ni abrir teclados en cascada; el foco se aplica una sola vez al montar.
4. **REQ-034**: Mantener el texto y el placeholder del input en el idioma correspondiente cuando ya usan i18n (`closing.reopenModalInstruction`, `closing.reopenModalConfirmWord`). No hardcodear palabras nuevas.
5. **REQ-035**: Evitar que el overlay permita scroll de fondo en móvil si el modal ya lo está previniendo; si no existe un lock de scroll, documentarlo como deuda y no introducirlo en esta spec.

### 2.4 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-036**: Evaluar un guardrail estático en `frontend/scripts/check-ui-rules.cjs` que detecte:
   - overlays de modales con `z-50`/`z-40` o sin `z` explícito,
   - overlays con `onClick` de cierre,
   - modales sin una acción de cierre visible.
   Si la detección no es confiable, documentar la regla y dejarla fuera del build.
2. **REQ-037**: Migrar el texto hardcodeado `Escribe "confirmo" para eliminar` de `DeleteModal` a claves i18n, manteniendo la palabra `confirmo` como valor de confirmación. No es requisito para cerrar esta spec.
3. **REQ-038**: Evaluar `role="dialog"` y `aria-modal="true"` en los modales para mejorar accesibilidad; no es requisito para cerrar esta spec.

### 2.5 Requerimientos No Funcionales

- **Rendimiento**: cambio solo de frontend; sin impacto medible en RAM/CPU del iHost. El hook no ejecuta timers ni listeners globales.
- **Seguridad**: no cambia autenticación ni permisos; solo la interacción de confirmación y el bloqueo de fondo en el cliente. La validación real de cada acción sigue en backend.
- **Almacenamiento**: no se agregan archivos de datos, DBs ni migraciones.
- **Dependencias**: cero dependencias nuevas en `frontend/package.json`; se usa React ya presente.
- **Compatibilidad**: desktop, móvil y navegadores soportados (incluido Safari/iOS). El foco programático y los overlays deben verificarse en móvil.
- **Accesibilidad**: el input conserva su `<label>`; el botón mantiene `disabled`; Enter funciona desde el input enfocado; los modales conservan una acción de cierre visible.
- **iHost**: build estático de Vite sin cambios de infraestructura; el bundle crece solo por el hook compartido y ajustes de clases.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **SPEC-122** (`docs/specs/SPEC-122-comprobantes-pdf-pagos-deudas.md`, estado `released`, commit `21ef61c` en `main`) definió:
  - Para `DeleteModal`: input con foco automático, escritura de `confirmo` y Enter para ejecutar la acción; con texto inválido Enter no hace nada.
  - Para todos los modales: sin cierre por click fuera, overlay bloqueante y cierre solo con Cancelar/Cerrar o X (CA-023).
- **`AGENTS.md`** y **`docs/project-rules.md` §4.7** ya documentan ambas reglas, pero la redacción está acotada a `DeleteModal`/`confirmo` y no menciona la escala de capas.
- **`docs/specs/templates/spec-template.md`** y **`.opencode/skills/spec-manager/SKILL.md`** repiten la misma redacción acotada a `DeleteModal`.
- **Auditoría de `main`**:
  - `rg -n "fixed inset-0" frontend/src` → **30 ocurrencias** en 22 archivos: **29 overlays de modales** en 21 archivos + **1 overlay del sidebar móvil** (`DashboardLayout.tsx`, fuera de alcance).
  - `rg -n "<DeleteModal" frontend/src` → **23 usos en 19 archivos**.
  - `rg -n "confirmo|confirmWord|confirmText|reopenModalConfirmWord" frontend/src` → solo `DeleteModal.tsx` y `RegistrosPage.tsx`.
  - **Click fuera**: ningún modal de `main` tiene `onClick` de cierre en su overlay. El único overlay con cierre por click fuera es el drawer del sidebar (`DashboardLayout.tsx:58-59`), que no es un modal.
  - **z-index**: 23 de 29 overlays de modales usan `z-50` o el default dinámico de `IconPickerModal` (`z-50`). Solo 6 ya están en `z-[60]`/`z-[70]`: `DebtDocumentModal`, `DeleteModal`, `ServicePickerModal`, `BudgetPage` (modales de grupo/categoría) y `BudgetTransactionsPage` (modal de cuenta). De esos 6, los formularios anidados y sus pickers requieren un nivel extra para mantener la jerarquía.
  - **Acción de cierre**: 3 modales no tienen una acción siempre visible: `AddInsuranceModal` (Cancelar condicionado a seleccionar servicio), `AddPolicyModal` (Cancelar condicionado a seleccionar auto) y `WebhookModal` (sin cierre durante `loading`).
  - **Dropdowns**: `Select` usa un portal con default `z-50`; si los modales pasan a `z-[60]`, los dropdowns dentro de modales quedarían detrás del overlay. Hay 52 usos de `Select`; solo 2 pasan `zIndex` explícito (en `BudgetPage`).
  - **Toasts**: `Toast` usa `z-50`; si los modales pasan a `z-[60]`, los mensajes quedarían detrás del overlay.
  - **Menús de 3 puntos**: `CardMenu` renderiza su portal con `zIndex: 9999` inline, por encima de cualquier modal. Se normaliza a `z-50` para que el overlay de `z-[60]` lo bloquee.
- **Estado del modal de reabrir mes**: `RegistrosPage.tsx` tiene `autoFocus` en el input (línea 768) y el botón deshabilitado por `busy || !isReopenValid`, pero **no tiene `onKeyDown`**. Por lo tanto, Enter no confirma.
- **`DeleteModal.tsx`**: ya tiene `useEffect` para foco, handler de Enter y comparación `trim().toLowerCase() === 'confirmo'`. Se toma como referencia.
- No se requiere investigación web: el comportamiento es de React/DOM y el patrón ya existe en el repo.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A. Corregir solo `RegistrosPage` y dejar los `z-50` | Cambio mínimo | No cumple el pedido de aplicar la regla a todo el proyecto; los overlays quedan al mismo nivel que los menús | ❌ Rechazada |
| B. Hook compartido + auditoría de los 29 modales + escala de capas documentada | Regla única, cero dependencias, correcciones acotadas y verificables | Requiere tocar varios archivos y validar dropdowns/toasts | ✅ Seleccionada |
| C. Componente genérico `Modal` que reemplace todos los modales | Unifica toda la UI y evita divergencias futuras | Refactor grande y riesgoso: cada modal tiene layouts, tamaños, iconos y flujos distintos; excede el pedido | ❌ Rechazada para esta spec |
| D. Solo documentar la regla sin tocar código | Cero riesgo | Los modales siguen sin cumplirla | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Escala de capas única para todo el proyecto.
- **Contexto**: los overlays en `z-50` compiten con los menús de 3 puntos (`CardMenu`) y `CreateMenu`; los dropdowns `Select` y los toasts también usan `z-50`.
- **Decisión**: aplicar la escala `50 → 60 → 70 → 80 → 90 → 100` definida en REQ-023.
- **Consecuencias**: el overlay bloquea el fondo, los dropdowns dentro de modales se ven y los toasts no quedan ocultos. Cambio mecánico y fácil de auditar.

**ADR-002**: Prohibir el cierre por click fuera en todos los modales.
- **Contexto**: SPEC-122 ya lo documentó, pero no se había auditado el proyecto completo.
- **Decisión**: el overlay no lleva handler de cierre; el cierre es solo con la X o Cancelar/Cerrar/Listo. El drawer del sidebar se excluye por no ser un modal.
- **Consecuencias**: se elimina el riesgo de cerrar un formulario por un click accidental; los usuarios siempre tienen una acción visible.

**ADR-003**: `Select` y `Toast` se elevan por encima de los modales.
- **Contexto**: al subir los modales a `z-[60]`, el portal de `Select` (`z-50`) y los toasts (`z-50`) quedarían detrás.
- **Decisión**: `Select` default `z-[90]`, `Toast` `z-[100]`, `IconPickerModal` default `z-[60]`.
- **Consecuencias**: los dropdowns y mensajes siguen funcionando dentro de modales; los overrides explícitos de `Select` se conservan.

**ADR-004**: Hook compartido para la confirmación por palabra.
- **Contexto**: `DeleteModal` ya implementa foco + Enter; el modal de reabrir mes no. Duplicar la lógica garantiza que vuelvan a divergir.
- **Decisión**: crear `frontend/src/hooks/useConfirmWordInput.ts` con normalización, validez, foco y Enter; usarlo en ambos modales.
- **Consecuencias**: un solo lugar que corregir; `DeleteModal` conserva su API y estilos; el hook es trivial y no impacta el runtime del iHost.

**ADR-005**: No crear un componente genérico de modal en esta spec.
- **Contexto**: hay 29 overlays con estructuras y flujos distintos.
- **Decisión**: corregir capas y cierres in situ; dejar la unificación como deuda futura si aparece una necesidad concreta.
- **Consecuencias**: menor riesgo de regresión; la regla queda documentada y con guardrail opcional.

## 4. Diseño Técnico

### 4.1 Escala de capas (z-index)

| Capa | z-index | Elementos |
|------|---------|-----------|
| Menús y navegación | `z-50` | `CardMenu`, `CreateMenu`, header, sidebar |
| Modal base | `z-[60]` | Overlay y contenido de un modal standalone |
| Modal anidado / picker | `z-[70]` | `DeleteModal` sobre un modal base, `ServicePickerModal`, `IconPickerModal` anidado |
| Confirmación sobre modal anidado | `z-[80]` | Confirmación por palabra o dropdown dentro de un modal ya elevado |
| Dropdown `Select` | `z-[90]` | Portal del componente `Select` |
| Toasts | `z-[100]` | Notificaciones globales |

Regla práctica: si un modal se abre desde otro modal, su overlay debe ser **al menos un nivel superior** al del modal que lo abre.

### 4.2 Componentes y archivos

#### 4.2.1 `useConfirmWordInput` (nuevo)
- **Responsabilidad**: centralizar el estado del texto de confirmación, la normalización, la validez, el foco inicial y el handler de Enter.
- **Interfaz**:
  ```ts
  type UseConfirmWordInputOptions = {
    word: string
    onConfirm: () => void
    disabled?: boolean
  }

  type UseConfirmWordInputResult = {
    value: string
    setValue: React.Dispatch<React.SetStateAction<string>>
    inputRef: React.RefObject<HTMLInputElement | null>
    isValid: boolean
    handleKeyDown: (e: React.KeyboardEvent<HTMLInputElement>) => void
    reset: () => void
  }
  ```
- **Dependencias**: `react` (`useEffect`, `useRef`, `useState`), ya presente.
- **Ubicación**: `frontend/src/hooks/useConfirmWordInput.ts`.

#### 4.2.2 `DeleteModal`
- **Cambio**: reemplazar el estado y el handler locales por `useConfirmWordInput({ word: 'confirmo', onConfirm: handleConfirm, disabled: submitting })`. Agregar un guard interno `submitting` + `useRef` para no disparar dos veces la misma acción (Enter o click) mientras la promesa de `onConfirm` está en curso, incluso antes del re-render. Mantener `title`, `subtitle`, `onConfirm`, `onCancel`, el icono de advertencia, el botón rojo y el `z-[70]`.
- **Ubicación**: `frontend/src/components/DeleteModal.tsx`.

#### 4.2.3 Modal de reabrir mes (`RegistrosPage`)
- **Cambio**: extraer el modal a un componente hijo `ReopenMonthModal`, que se monta solo cuando `showReopenModal` es true. Usar `useConfirmWordInput({ word: confirmWord, onConfirm, disabled: busy })`; reemplazar `autoFocus` por el `inputRef` del hook; el botón queda `disabled={busy || !isValid}`. El componente hijo garantiza que el `useEffect` de foco se ejecute al abrir el modal (no al montar la página).
- **Ubicación**: `frontend/src/pages/RegistrosPage.tsx` (componente `ReopenMonthModal`).

#### 4.2.4 Ajustes de capas y cierres

Rutas relativas a `frontend/src/`.

| Archivo:línea | z actual | Cierre visible | Acción |
|---------------|----------|----------------|--------|
| `components/AddInsuranceModal.tsx:69` | `z-50` | Condicional (solo tras elegir servicio) | Subir a `z-[60]` y agregar X fija |
| `components/AddPolicyModal.tsx:73` | `z-50` | Condicional (solo tras elegir auto) | Subir a `z-[60]` y agregar X fija |
| `components/AnalyzerPickerModal.tsx:47` | `z-50` | Botón Listo siempre | Subir a `z-[60]` |
| `components/BillHistoryModal.tsx:69` | `z-50` | X siempre | Subir a `z-[60]` |
| `components/DebtDocumentModal.tsx:83` | `z-[60]` | X siempre | Sin cambios de capa |
| `components/DebtPayModal.tsx:68` | `z-50` | Cancelar siempre | Subir a `z-[60]` |
| `components/DeleteModal.tsx:29` | `z-[70]` | Cancelar siempre | Sin cambios de capa |
| `components/EmailRecipientsModal.tsx:42` | `z-50` | Cancelar siempre | Subir a `z-[60]` |
| `components/HousePickerModal.tsx:55` | `z-50` | X siempre | Subir a `z-[60]` |
| `components/IconPickerModal.tsx:53` | `${zIndex}` default `z-50` | X siempre | Default `z-[60]`; overrides `z-[70]`/`z-[80]` se mantienen |
| `components/InstitutionCategoriesModal.tsx:101` | `z-50` | X siempre | Subir a `z-[60]` |
| `components/PayBillModal.tsx:55` | `z-50` | Cancelar siempre | Subir a `z-[60]` |
| `components/RenewServiceModal.tsx:90` | `z-50` | Cancelar siempre | Subir a `z-[60]` |
| `components/ServicePickerModal.tsx:54` | `z-[70]` | Botón Listo siempre | Subir a `z-[80]` para quedar sobre formularios anidados |
| `components/UploadBillModal.tsx:213` | `z-50` | X siempre (disabled al guardar) | Subir a `z-[60]` |
| `components/WebhookModal.tsx:62` | `z-50` | Condicional (sin cierre durante loading) | Subir a `z-[60]` y agregar X fija |
| `pages/AutomationDetailPage.tsx:298` | `z-50` | Cerrar siempre | Subir a `z-[60]` |
| `pages/BudgetPage.tsx:571` | `z-50` | X siempre | Subir a `z-[60]` |
| `pages/BudgetPage.tsx:784` | `z-50` | X siempre | Subir a `z-[60]` |
| `pages/BudgetPage.tsx:926` | `z-[60]` | Cancelar siempre | Añadir prop `zIndex` y pasar `z-[70]` desde `BudgetConfigModal` |
| `pages/BudgetPage.tsx:990` | `z-[60]` | Cancelar siempre | Añadir prop `zIndex` y pasar `z-[70]` desde `BudgetConfigModal` |
| `pages/BudgetTransactionsPage.tsx:436` | `z-50` | X siempre | Subir a `z-[60]` |
| `pages/BudgetTransactionsPage.tsx:590` | `z-50` | X siempre | Subir a `z-[60]` |
| `pages/BudgetTransactionsPage.tsx:702` | `z-[60]` | Cancelar siempre | Subir a `z-[70]`: es un formulario anidado dentro de `AccountsModal` |
| `pages/BudgetTransactionsPage.tsx:825` | `z-50` | X siempre | Subir a `z-[60]` |
| `pages/PensionImportPage.tsx:642` | `z-50` | Cancelar siempre | Subir a `z-[60]` |
| `pages/PensionImportPage.tsx:659` | `z-50` | X siempre | Subir a `z-[60]` |
| `pages/PensionImportPage.tsx:708` | `z-50` | X y Cancelar siempre | Subir a `z-[60]` |
| `pages/RegistrosPage.tsx:50` | `z-50` | X siempre | Subir a `z-[60]` |

**Ajustes de anidamiento**: los formularios `GroupFormModal` y `CategoryFormModal` se usan tanto en la página como dentro de `BudgetConfigModal`; por eso reciben una prop `zIndex` (default `z-[60]`, `z-[70]` cuando se abren desde `BudgetConfigModal`). El `IconPickerModal` de ese contexto sube a `z-[80]`, el `ServicePickerModal` sube a `z-[80]` y el `AccountFormModal` de `BudgetTransactionsPage` sube a `z-[70]` porque siempre se abre dentro de `AccountsModal`.

**Exclusión explícita**: `components/DashboardLayout.tsx:58` (`z-40`) es el overlay del drawer móvil del sidebar, no un modal. Conserva su cierre por click fuera y no se modifica.

**Sin offenders de click fuera**: la auditoría no encontró ningún modal con `onClick` de cierre en el overlay. La corrección es preventiva/documental y de capas; no hay que remover handlers de cierre en modales existentes.

#### 4.2.5 Capas globales

- `frontend/src/components/Select.tsx`: `zIndex = 'z-[90]'` por defecto. Los overrides explícitos (`z-[80]` en `BudgetPage`) se mantienen y siguen siendo válidos porque están dentro de modales `z-[60]`.
- `frontend/src/components/Toast.tsx`: `z-[100]`.
- `frontend/src/components/IconPickerModal.tsx`: default `z-[60]`; los usos con `zIndex="z-[70]"`/`z-[80]` se mantienen.
- `frontend/src/components/CardMenu.tsx`: el portal del menú de 3 puntos pasa de `zIndex: 9999` inline a `zIndex: 50`, para que el overlay del modal (`z-[60]`) lo bloquee.
- `frontend/src/components/CreateMenu.tsx`, `Sidebar.tsx` y header: se mantienen en `z-50` para quedar debajo de los modales.

#### 4.2.6 Documentación
- **Cambio**: generalizar la redacción de ambas reglas en `AGENTS.md`, `docs/project-rules.md` §4.7, `docs/specs/templates/spec-template.md` y `.opencode/skills/spec-manager/SKILL.md` para que cubran cualquier palabra de confirmación, cualquier modal, y la escala de capas.
- Incluir explícitamente la exclusión del drawer del sidebar.

### 4.3 Modelo de datos

No aplica. Esta spec no crea ni modifica tablas, columnas, índices ni migraciones.

### 4.4 APIs / Contratos

No aplica. No se agregan ni modifican endpoints. Las acciones siguen llamando a los handlers existentes (`onConfirm` de `DeleteModal`, `handleReopenMonth` de `RegistrosPage`, etc.).

### 4.5 Dependencias

- **Internas**: `DeleteModal`, `RegistrosPage`, 23 overlays de modales, `Select`, `Toast`, `IconPickerModal`, nuevo hook compartido, documentación de reglas.
- **Externas**: ninguna nueva. React 19 y TypeScript ya están en el proyecto.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] **CA-001**: El inventario de la spec coincide con `main`: 29 overlays de modales en 21 archivos, 23 usos de `<DeleteModal>` en 19 archivos y un único modal adicional de confirmación por palabra (`RegistrosPage`, reabrir mes).
- [ ] **CA-002**: Al abrir cualquier `DeleteModal`, el input de confirmación recibe el foco automáticamente (`document.activeElement` es el input).
- [ ] **CA-003**: En `DeleteModal`, escribir `confirmo` habilita el botón y presionar Enter ejecuta la misma acción que el botón; con un texto inválido, Enter no ejecuta nada.
- [ ] **CA-004**: Al abrir el modal de reabrir mes, el input recibe el foco automáticamente.
- [ ] **CA-005**: En el modal de reabrir mes, escribir `reabrir` (es) o `reopen` (en) habilita el botón y presionar Enter ejecuta la reapertura; con un texto inválido, Enter no ejecuta nada.
- [ ] **CA-006**: La comparación acepta mayúsculas/minúsculas y espacios al inicio/fin en ambos modales (`Confirmo`, ` confirmo `, `REABRIR`).
- [ ] **CA-007**: Mientras la acción está en curso (`busy`/`submitting`), Enter no dispara una segunda ejecución y el botón permanece deshabilitado, tanto en `DeleteModal` como en el modal de reabrir mes.
- [ ] **CA-008**: Presionar Enter en el input no provoca envío de formulario, navegación ni recarga de la página.
- [ ] **CA-009**: Al reabrir un modal, el input aparece vacío (o el modal se remonta) y no conserva una palabra válida de una apertura anterior.
- [ ] **CA-020**: **Ningún modal de `main` se cierra al hacer click fuera de su contenido.** Se verifica en los 29 overlays, incluidos los que se abren desde otro modal.
- [ ] **CA-021**: El overlay de cada modal bloquea la interacción con la página de fondo: no se puede clickear un menú de 3 puntos, un `CreateMenu`, el sidebar ni un botón de la página mientras el modal está abierto.
- [ ] **CA-022**: Todos los modales tienen una acción de cierre visible **siempre**: X o Cancelar/Cerrar/Listo, incluso en estado `loading`, durante `saving` y antes de seleccionar una entidad. En particular, `AddInsuranceModal`, `AddPolicyModal` y `WebhookModal` dejan de depender de una selección o de un estado para mostrar el cierre.
- [ ] **CA-023**: Los 29 overlays de modales usan `z-[60]` o superior; los anidados usan `z-[70]`/`z-[80]` según la escala. No queda ningún modal en `z-50`.
- [ ] **CA-024**: Los dropdowns de `Select` dentro de modales se ven por encima del overlay (`z-[90]`) y los toasts se ven por encima de los modales (`z-[100]`).
- [ ] **CA-025**: El drawer móvil del sidebar (`DashboardLayout`) conserva su cierre por click fuera y no se ve afectado por la escala de modales.
- [ ] **CA-026**: Los modales conservan las reglas de SPEC-122: el cierre es solo con Cancelar/Cerrar o X, la confirmación queda por encima del modal que la abre y no hay cierre por click fuera.
- [ ] **CA-027**: No se agregaron dependencias a `frontend/package.json` ni cambios de backend, API, DB o migraciones.
- [ ] **CA-028**: `AGENTS.md`, `docs/project-rules.md` §4.7, `docs/specs/templates/spec-template.md` y `.opencode/skills/spec-manager/SKILL.md` describen ambas reglas generales y la escala de capas, con la exclusión del sidebar.
- [ ] **CA-DARK**: Los inputs de ambos modales usan tokens del tema (`bg-card`, `text-text`) y se verificó legibilidad en darkmode (texto y placeholder).
- [ ] **CA-MODAL**: Se verificó manualmente que el click fuera no cierra ni interactúa con el fondo, que el cierre es solo con Cancelar/Cerrar o X, y que Enter confirma cuando el texto es válido.
- [ ] **CA-BACK**: No aplica (no se crean páginas de detalle ni formularios con lista padre).

### 5.2 No funcionales

- [ ] **CA-NF-001**: `npm run build` en `frontend/` pasa sin errores (incluye `check-ui-rules.cjs`, `tsc -b` y Vite).
- [ ] **CA-NF-002**: `npm run check:ui` pasa sin offenders.
- [ ] **CA-NF-003**: El bundle no incorpora dependencias nuevas; el hook compartido no agrega listeners globales ni timers.
- [ ] **CA-NF-004**: Prueba manual en desktop y en viewport móvil (o dispositivo iOS/Safari) del foco, escritura, Enter, overlay y cierre de ambos modales.
- [ ] **CA-NF-005**: No se observan regresiones visuales en los 29 modales (posición, scroll interno, dropdowns y toasts).

### 5.3 Testing

- **Unit tests**: no hay runner de tests en `frontend/` al momento de esta spec. Si se incorpora uno, cubrir `normalizeConfirmWord`/`isValid` y el handler de Enter con texto válido, inválido y `disabled`.
- **Integration tests**: no aplica (no hay suite E2E en el repo).
- **E2E/manual**: abrir un `DeleteModal` desde una página de listado, confirmar con Enter; abrir el modal de reabrir mes, confirmar con Enter; probar texto inválido, mayúsculas/espacios, `busy` y darkmode; abrir los 29 modales y verificar click fuera, overlay, cierre visible y capas.
- **Carga/Performance**: no aplica; cambio de UI sin impacto medible en iHost.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Auditoría de modales de `main` y confirmación del inventario | 0.25 días | Ninguna |
| 2 | Crear `useConfirmWordInput` y refactorizar `DeleteModal` sin cambiar su API | 0.5 días | Fase 1 |
| 3 | Integrar el hook en el modal de reabrir mes (`RegistrosPage`) con guard de `busy` | 0.25 días | Fase 2 |
| 4 | Corregir capas de los 23 overlays en `z-50` y ajustar `Select`/`Toast`/`IconPickerModal` | 0.5 días | Fase 1 |
| 5 | Agregar X fija a `AddInsuranceModal`, `AddPolicyModal` y `WebhookModal` | 0.25 días | Fase 1 |
| 6 | Generalizar la documentación de ambas reglas y la escala de capas | 0.25 días | Fases 2-5 |
| 7 | Build, `check:ui` y QA manual desktop/móvil/darkmode de los 29 modales | 0.5 días | Fases 2-6 |

Estimación total: **~2.5 días**.

### 6.2 Milestones

1. **MVP**: hook compartido + `DeleteModal` refactorizado + modal de reabrir mes con foco y Enter + todos los overlays en `z-[60]` o superior + X fija en los 3 modales sin cierre.
2. **V1.0**: ajustes de `Select`/`Toast`/`IconPickerModal` + documentación generalizada + QA manual completo + build/check:ui en verde.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Regresión en `DeleteModal` al refactorizar (23 usos) | Media | Alto | Mantener la API pública y los estilos; probar al menos un flujo de borrado por cada tipo de entidad y el borrado de documentos desde `DebtDocumentModal` |
| Dropdowns `Select` quedan detrás del overlay al subir los modales a `z-[60]` | Alta si no se ajusta | Alto | Cambiar el default de `Select` a `z-[90]` y verificar los overrides existentes |
| Toasts ocultos detrás de modales | Alta si no se ajusta | Medio | Subir `Toast` a `z-[100]` |
| Modales anidados con capas incorrectas | Media | Alto | Aplicar la escala 60/70/80 y probar `DeleteModal` sobre `DebtDocumentModal`, pickers dentro de formularios y confirmaciones |
| Cierre condicional que sigue desapareciendo en algún estado | Media | Medio | Agregar X fija en `AddInsuranceModal`, `AddPolicyModal` y `WebhookModal`; revisar `loading`/`saving` |
| Enter dispara dos veces por pulsaciones rápidas | Media | Medio | Guard con `busy` + `useRef` en el handler de reabrir; liberar en `finally` |
| El foco programático no se comporta igual en Safari/iOS | Media | Medio | Usar `ref` + `useEffect` como en SPEC-122 y verificar en viewport móvil/dispositivo |
| Divergencia futura si un modal nuevo no usa el hook o no respeta capas | Media | Medio | Documentar la regla; evaluar guardrail estático (P2) |
| Cambio de palabra i18n (`reabrir`/`reopen`) | Baja | Bajo | Normalizar ambos lados con `trim().toLowerCase()`; usar la clave `closing.reopenModalConfirmWord` |
| `preventDefault()` interfiere con otro comportamiento del input | Baja | Bajo | Aplicarlo solo en `keydown` de Enter y verificar que no haya formularios envolventes |
| Aumento de bundle por el hook | Baja | Bajo | Hook de pocas líneas, sin dependencias; verificar build |
| El drawer del sidebar se ve afectado por la nueva escala | Baja | Medio | Excluirlo explícitamente; su overlay queda en `z-40` y el panel en `z-50` |

## 8. Notas y Referencias

- **SPEC-122** (`released` en `main`, commit `21ef61c`): definió la regla original para `DeleteModal` con `confirmo`, foco automático y Enter, y la regla de modales sin cierre por click fuera (CA-023).
- **`AGENTS.md`**, sección "Reglas críticas" (modales).
- **`docs/project-rules.md`**, §4.7 "Modales: cierre explícito y overlay bloqueante".
- **`docs/specs/templates/spec-template.md`**, consideraciones de UI para modales.
- **`.opencode/skills/spec-manager/SKILL.md`**, sección 4 "Reglas de documentación".
- Archivos de código relevantes:
  - `frontend/src/components/DeleteModal.tsx`
  - `frontend/src/pages/RegistrosPage.tsx`
  - `frontend/src/hooks/useConfirmWordInput.ts` (nuevo)
  - `frontend/src/components/Select.tsx`
  - `frontend/src/components/Toast.tsx`
  - `frontend/src/components/IconPickerModal.tsx`
  - Los 29 overlays listados en §4.2.4.
- Comandos de auditoría:
  - `rg -n "confirmo|confirmWord|confirmText|reopenModalConfirmWord" frontend/src`
  - `rg -n "<DeleteModal" frontend/src`
  - `rg -n "fixed inset-0" frontend/src`
- No se usan datos reales ni PII; los ejemplos de palabras (`confirmo`, `reabrir`, `reopen`) son sintéticos.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-07 | opencode | Creación inicial de la especificación (SPEC-126). Auditoría de modales de `main`: `DeleteModal` ya cumple; el modal de reabrir mes de `RegistrosPage` no maneja Enter. Se propone hook compartido y generalización de la regla documentada. |
| 2026-11-07 | opencode | Alcance ampliado a pedido del usuario: la regla de overlay/click fuera se aplica a **todos** los modales de `main`. Auditoría de los 29 overlays: 23 en `z-50`, 3 sin cierre siempre visible y 0 con cierre por click fuera. Se agrega la escala de capas y los ajustes de `Select`/`Toast`/`IconPickerModal`. |
| 2026-11-07 | opencode | Implementación: hook `useConfirmWordInput`, `DeleteModal` con guard de envío, `ReopenMonthModal` con foco/Enter, 23 overlays a `z-[60]`, formularios anidados a `z-[70]` y pickers a `z-[80]`, portal de `CardMenu` de `9999` a `50`, X fija en `AddInsuranceModal`/`AddPolicyModal`/`WebhookModal`, `Select` `z-[90]`, `Toast` `z-[100]`, `IconPickerModal` default `z-[60]` y documentación generalizada. `npm run build` y `npm run check:ui` OK. Pendiente validación manual del usuario. |
| 2026-11-07 | opencode | Validación automatizada con Chrome headless: foco inicial, Enter, guard de doble Enter, click fuera que no cierra, texto inválido sin acción y normalización con espacios/mayúsculas OK en `DeleteModal` y en el modal de reabrir mes. Pendiente validación manual del usuario. |
