---
title: "Detalle de deuda: resumen humano de tiempo restante, último pago y fin estimado"
id: "SPEC-125"
status: "in_progress"
author: "opencode"
created: "2026-11-07"
updated: "2026-11-07"
github_issue: 128
---

# Detalle de deuda: resumen humano de tiempo restante, último pago y fin estimado

**ID**: SPEC-125  
**Estado**: in_progress  
**Autor**: opencode  
**Creado**: 2026-11-07  
**Actualizado**: 2026-11-07

---

## 1. Resumen Ejecutivo

En la pantalla de detalle de una deuda (`/deudas/:id`) el usuario ve hoy el progreso de pago y, más abajo, la lista de cuotas. La fecha de la última cuota solo aparece al final del listado, después de hacer scroll. Para responder la pregunta que realmente le importa —*“¿cuándo termino de pagar esto?”*— tiene que recorrer la lista y hacer el cálculo mental. La información existe, pero no está jerarquizada ni expresada en lenguaje humano.

Esta spec agrega un **resumen de payoff above the fold**, apenas se entra a la deuda: una frase principal con el tiempo restante en meses calendario (“Te faltan 2 años y 9 meses para saldar esta deuda”), la **fecha exacta de la última cuota** como subtítulo (“Última cuota: 15 de agosto de 2029 · si seguís el plan”), el último pago realizado, la próxima cuota (o la cuota vencida) y el progreso por cantidad de cuotas. El resumen se acompaña de una **barra de progreso** y de una **línea de tiempo compacta** (hoy → próxima cuota → última cuota) construida con CSS/SVG, sin librerías de gráficos.

La investigación UX muestra que, para este caso, una barra lineal y un timeline claro comunican mejor que un donut o un chart decorativo. Por eso el donut actual del detalle se **retira**: mezclaba porcentaje por cantidad con segmentos por monto y competía con el titular. La barra por cuotas y la línea de tiempo responden mejor a “cuánto falta” y “cuándo es el último pago”, con menos altura y mejor accesibilidad.

El cambio es **100% frontend**: consume los datos que `GET /api/debts/{id}` y `GET /api/debts/{id}/bills` ya devuelven (`last_due_date`, `pending_count`, `pending_amount`, `last_paid_at` y las cuotas). No requiere endpoints nuevos, migraciones, ni dependencias npm. En el iHost el impacto es nulo: no hay consultas adicionales ni trabajo de servidor; el cálculo es O(n) sobre las cuotas ya cargadas en memoria (decenas o pocos cientos de filas).

**Consideraciones iHost**: sin dependencias nuevas, sin llamadas de red adicionales, sin estado global nuevo, sin intervalos ni animaciones continuas. El bundle aumenta solo por el componente y los helpers (estimado < 5 KB minificados).

**Decisiones de UI obligatorias**:
- Se usan exclusivamente tokens del tema (`bg-card`, `text-text`, `text-text-secondary`, `bg-border`, `bg-primary`, `bg-success/20`, `bg-warning/20`); prohibido `bg-white`, `text-black`, `#30d158`, `#ff9f0a` u otros colores hardcodeados.
- La pantalla no crea una ruta nueva: sigue usando `BACK_ROUTES` de `DashboardLayout.tsx` para el retorno a `/deudas`; no se agregan links “← Título” dentro del contenido.
- No hay formularios ni selectores nuevos; no aplica el guardrail de `<select>` nativo, pero el build debe pasar `npm run check:ui`.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Agregar un bloque de resumen de payoff en la parte superior de la página de detalle de deuda (`frontend/src/pages/DebtBillsPage.tsx`), **antes de los chips y de la lista de cuotas**, visible sin hacer scroll en un viewport móvil de 375×667 px después del header. El bloque reemplaza la card de progreso actual (donut + barra + montos) y absorbe el banner de deuda finalizada; no se muestran dos bloques de progreso ni dos mensajes de cierre. **El donut se elimina del detalle** (ver ADR-003).
2. **REQ-002**: La frase principal debe expresar el **tiempo restante en meses calendario** desde hoy hasta la fecha de la última cuota pendiente. Se calcula con años y meses completos, sin días:
   - 33 meses → “Te faltan 2 años y 9 meses para saldar esta deuda”.
   - 25 meses → “Te faltan 2 años y 1 mes para saldar esta deuda”.
   - 13 meses → “Te falta 1 año y 1 mes para saldar esta deuda”.
   - 12 meses → “Te falta 1 año para saldar esta deuda”.
   - 2 meses → “Te faltan 2 meses para saldar esta deuda”.
   - 1 mes → “Te falta 1 mes para saldar esta deuda”.
   - Menos de un mes (incluido el mes en curso, con fecha futura) → “Te falta menos de un mes para saldar esta deuda”.
   - La última cuota vence hoy → “La última cuota vence hoy”.
   - Fecha de la última cuota ya pasada con cuotas pendientes → “La última cuota venció el 5 de noviembre de 2026; revisá tus cuotas” (fecha exacta, no solo “la fecha ya pasó”).
   La concordancia verbal (“falta”/“faltan”) y los plurales deben ser correctos en español e inglés, incluidas las combinaciones 1 año + 1 mes, 1 año + N meses y N años + 1 mes.
3. **REQ-003**: Debajo de la frase principal se muestra la **fecha exacta de la última cuota** (día, mes y año): “Última cuota: 15 de agosto de 2029 · si seguís el plan”. No se muestra solo “agosto de 2029”: el usuario preguntó específicamente cuándo es el último pago. Si la fecha ya pasó: “Última cuota: 5 de noviembre de 2026 (vencida)”. Si no hay fecha válida: “No pudimos estimar la fecha de la última cuota”. La fecha se calcula con la **última cuota pendiente** (`last_due_date` o el máximo `due_date` de las cuotas pendientes), no con `start_date + installments_total`, porque el plan puede haberse editado y las cuotas existentes no se regeneran (ver §4.3).
4. **REQ-004**: Mostrar el **progreso de pago por cantidad de cuotas** de forma consistente:
   - Texto “Ya pagaste {paid} de {total} cuotas” (o “Ya pagaste la única cuota” si total = 1 y está pagada) y porcentaje.
   - Barra segmentada pagado/pendiente; el tramo pagado usa `bg-primary` (contraste ≥ 3:1 contra el track en light y dark); el track usa `bg-border`.
   - Denominador efectivo: `max(debt.installments_total, bills.length)` para evitar que las cuotas cargadas superen el 100%; si el denominador es 0, no se renderiza la barra.
   - Porcentaje: 100% **solo** cuando no quedan cuotas pendientes; con pendientes, redondear y limitar a 99% para no mostrar un 100% engañoso.
   - Si no hay cuotas, no mostrar 0% ni `NaN`; mostrar el estado “Sin cuotas generadas” o “Faltan generar cuotas” según corresponda (REQ-007).
5. **REQ-005**: Mostrar tres datos accionables en el mismo bloque:
   - **Último pago realizado**: fecha de `last_paid_at` normalizada a `YYYY-MM-DD`, o la fecha máxima de `paid_at` entre las cuotas pagadas. Si no hay pagos: “Sin pagos registrados”. No rotularlo “Último pago” a secas, para no confundirlo con la última cuota del plan.
   - **Próxima cuota** (si `due_date >= hoy`): la cuota pendiente con menor `due_date` (desempate por `installment_number`), con número, fecha y monto (`#8 · 15 nov 2026 · $85.000`). Si la cuota más antigua ya venció, rotularla **“Cuota vencida”** con la fecha y el monto, no “Próxima cuota”.
   - **Última cuota**: fecha exacta de la última cuota pendiente (o “Sin fecha” si no se puede calcular).
   - El **saldo pendiente** (suma de `amount` de cuotas pendientes, no `debt.total`) se muestra en la fila de progreso como “Pendiente: $Y”; no se duplica como cuarta stat. Se formatea con el símbolo de la moneda y, en encabezados, también el código ISO.
6. **REQ-006**: Agregar una **línea de tiempo compacta** que muestre, como mínimo, `Hoy` → `Próxima cuota` (o `Cuota vencida`) → `Última cuota`. Reglas de composición:
   - Orden cronológico real; nunca dibujar un inicio posterior a “Hoy” a la izquierda.
   - Incluir `Inicio` solo si `start_date` es válido y `start_date <= hoy`; si el plan empieza en el futuro, omitir el inicio y mostrar “Empieza el {fecha}” como texto.
   - Si `next_due === start` o `next_due === hoy`, fusionar los hitos para no superponer etiquetas.
   - Si `next_due < hoy`, ubicarlo antes de “Hoy” y rotularlo “Cuota vencida”.
   - Si `next_due > end` o no hay próxima cuota, omitirlo.
   - Si `end < hoy`, marcar el extremo como “Última cuota vencida” y no prometer un countdown activo.
   - Posiciones proporcionales al rango `min(inicio, hoy) → fin`, clampeadas a 0–100%; si faltan fechas, mostrar solo los hitos disponibles sin inventar posiciones.
   - En deudas de más de 36 meses, usar 3 hitos (Hoy, Próxima, Última) sin marcas por año; no renderizar un punto por cuota.
   - La línea es un SVG/CSS liviano, no una librería de charts.
7. **REQ-007**: Definir **estados mutuamente excluyentes** y su precedencia, para que nunca se contradigan:
   1. **Saldada**: no hay cuotas pendientes y (`bills.length > 0` o `status === 'finalizada'`). Mostrar “Deuda saldada” + fecha de cierre + total pagado; no mostrar countdown ni barra de tiempo. Si `status === 'finalizada'` sin pagos, mostrar “Deuda cerrada manualmente” + “Sin pagos registrados”, sin inventar fecha.
   2. **Cuotas faltantes**: `bills.length === 0` pero `pending_count > 0` (o `installments_total > 0`). Mostrar “Faltan generar cuotas” + “No pudimos cargar el plan completo”; no calcular porcentaje ni afirmar una fecha de fin.
   3. **Sin cuotas**: `bills.length === 0` y `pending_count === 0`, con deuda no cerrada. Mostrar “Sin cuotas generadas”; no calcular porcentaje ni fechas.
   4. **En pausa**: `debt.status === 'inactiva'` y hay pendientes. Mostrar “En pausa” + “Quedan N cuotas pendientes”; aclarar que no se generan cuotas nuevas mientras esté inactiva y que el fin es “si reanudás”. Si además hay vencidas, mostrarlas como alerta secundaria.
   5. **Vencida**: hay al menos una cuota pendiente con `due_date < hoy`. Mostrar “Tenés N cuota(s) vencida(s)” con ícono + texto (no toda la tarjeta en rojo); el titular debe decir “Si te ponés al día, terminás en {fecha}” en lugar de prometer el fin como si no hubiera atraso.
   6. **Al día**: deuda activa con pendientes y sin vencidas. Mostrar “Al día” o el estado neutro correspondiente.
   Precedencia: 1 → 2 → 3 → 4 → 5 → 6. Si `debt.status === 'finalizada'` pero hay cuotas pendientes (dato inconsistente), **no** mostrar “Deuda saldada”: usar el estado con pendientes y una advertencia discreta. Si `pending_count === 0` pero hay una cuota pendiente en `bills`, priorizar `bills` y advertir la inconsistencia.
8. **REQ-008**: Usar **tokens del tema** para todos los colores y verificar dark mode. La barra usa `bg-primary` para el tramo pagado y `bg-border` para el track; los badges usan tintes (`bg-success/20`, `bg-warning/20`) con `text-text` + ícono, no texto de color sobre fondo claro. Prohibido hardcodear `#30d158`, `#ff9f0a`, `bg-white`, `text-black`. No se conserva el donut en el detalle.
9. **REQ-009**: Cumplir accesibilidad:
   - La barra de progreso usa `role="progressbar"` con `aria-valuemin`, `aria-valuemax` (denominador efectivo), `aria-valuenow` clampeado y `aria-valuetext` **localizado** (“7 de 24 cuotas pagadas, 29%” / “7 of 24 installments paid, 29%”). El texto visible no vive dentro del elemento con `role="progressbar"`.
   - `aria-labelledby` apunta a una etiqueta dedicada “Progreso de pago”, no al titular de tiempo.
   - No se renderiza la barra si el denominador es 0.
   - El color no es el único indicador de estado: todo badge/estado lleva texto e ícono.
   - La línea de tiempo visual es `aria-hidden="true"`; un párrafo `sr-only` localizado resume las fechas exactas. No duplicar con un `<ol>` visible y un `sr-only` a la vez.
   - Soportar texto al 200% sin truncar datos críticos; no usar `truncate` en montos; usar `tabular-nums`.
   - Respetar `prefers-reduced-motion`: la barra no anima o usa una transición de 200–300 ms desactivable con `motion-reduce:transition-none`.
   - Si se agrega una acción (“Ver cuotas pendientes”), el target debe ser ≥44×44 px.
10. **REQ-010**: Internacionalización completa en `frontend/public/i18n/es.json` y `frontend/public/i18n/en.json` (fuente de verdad). No editar `public/i18n/` ni `frontend/src/i18n/` a mano. El store `t()` no interpola parámetros: usar claves de frase completa o `.replace('{...}', valor)`. Incluir todas las combinaciones singulares/plurales (1 año + 1 mes, 1 año + N meses, N años + 1 mes, 1 cuota, N cuotas) y verificar naturalidad en ambos idiomas.
11. **REQ-011**: **Sin cambios de backend**: no se agregan endpoints, rutas, handlers, servicios, migraciones ni columnas. Se consumen exclusivamente:
    - `GET /api/debts/{id}`: `installments_total`, `pending_count`, `pending_amount`, `paid_count`, `paid_amount`, `next_due_date`, `last_due_date`, `last_paid_at`, `status`, `start_date`, `currency_code`, `currency_id`.
    - `GET /api/debts/{id}/bills`: `installment_number`, `due_date`, `amount`, `status`, `paid_at`.
    El cálculo se hace en frontend con las cuotas ya cargadas; no se agrega un fetch nuevo. Si se necesita el catálogo de monedas, se reutiliza el GET existente de monedas (REQ-013), no un endpoint nuevo.
12. **REQ-012**: Diseño **mobile-first y responsive**: funciona desde 320 px de ancho, sin overflow horizontal; en móvil los tres datos se apilan o se muestran en 2 columnas; la línea de tiempo no genera scroll horizontal de página. La card debe caber en el primer viewport en 375×667 px con texto al 100% (objetivo ≤ 400 px de alto); con texto al 200% puede crecer verticalmente y la lista baja, sin truncar datos.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-013**: Al entrar directo a `/deudas/:id`, cargar el catálogo de monedas si `currencies` está vacío (`loadCurrencies()`), para mostrar el símbolo correcto en lugar del código ISO cuando sea posible. Mantener el fallback al código ISO y nunca inventar una moneda (precedente SPEC-113). Es un GET existente, no un endpoint nuevo.
2. **REQ-014**: Mostrar el último pago también en lenguaje relativo (“hace 2 meses”, “hoy”) cuando la fecha sea válida, usando `Intl.RelativeTimeFormat` o un helper date-only-safe. No reutilizar `formatRelativeTime` tal cual si interpreta `YYYY-MM-DD` como UTC; normalizar a fecha local o extenderlo. La fecha exacta sigue visible.
3. **REQ-015**: Agregar una acción discreta “Ver cuotas pendientes” que aplique `?status=pending` y haga scroll suave a la lista de cuotas. Como `load` depende de `searchParams`, el scroll debe ejecutarse después de que la lista se renderice (ej. `useEffect` + ref o `requestAnimationFrame`), no antes del refetch. Debe respetar `prefers-reduced-motion` y tener target ≥44 px.
4. **REQ-016**: Microcopy de hitos (25/50/75%) de forma sutil, sin gamificación agresiva, sin confeti repetido y sin culpa. Ejemplo: “Vas por la mitad” una sola vez, como texto secundario; no bloquear la lectura.
5. **REQ-017**: Agregar un enlace/expandible “Cómo se calcula” que explique que la fecha se basa en la última cuota pendiente y que se recalcula si se adelanta la **última** cuota o se edita el plan. Texto secundario, sin modal si se puede resolver inline.
6. **REQ-018**: Eliminar el chip duplicado “X de Y cuotas” de la fila de chips (el resumen ya lo muestra); conservar “Total” y “Día de pago”. Si “Total” se mantiene, rotularlo “Total original” para no confundirlo con el saldo pendiente; o quitarlo si agrega ambigüedad.
7. **REQ-019**: Si los montos de las cuotas pendientes varían más de 5% entre sí, mostrar un segundo indicador de progreso por monto (“% del monto pagado”) con etiqueta explícita; el indicador principal sigue siendo por cantidad de cuotas.
8. **REQ-020**: Mostrar “Según las cuotas generadas ({generated} de {total})” y “Faltan generar {count} cuotas” cuando `bills.length < installments_total`, sin afirmar que la fecha de fin es definitiva. Esto cubre el caso de entrada directa por URL sin reconciliación.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-021**: Evaluar en un A/B test futuro si un anillo con porcentaje centrado comunica mejor que la barra lineal; no implementar dos variantes en esta spec. El donut actual no se conserva en el detalle.
2. **REQ-022**: Simulador “si pagás $X extra, terminás en …” con fecha recalculada y supuestos explícitos. Requiere spec propia (cálculo de intereses y amortización); **fuera de alcance**.
3. **REQ-023**: Celebración opcional al saldar la deuda (sin sonido automático, con `prefers-reduced-motion` y posibilidad de desactivar). No es parte del MVP; el estado “Deuda saldada” debe ser claro sin confeti.
4. **REQ-024**: Exportar o compartir el resumen de payoff. Fuera de alcance.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: el cálculo es O(n) sobre las cuotas ya cargadas; sin bucles anidados ni renders adicionales por cuota. Render inicial del bloque < 16 ms en el hardware de referencia. Sin intervalos ni `setInterval`.
- **Memoria**: sin librerías nuevas; sin estado global nuevo; el componente se desmonta con la página. Bundle adicional estimado < 5 KB minificado.
- **Red**: sin endpoints nuevos; se reutilizan los dos requests existentes del detalle y, opcionalmente, el GET existente de monedas si el catálogo está vacío.
- **Base de datos**: cero cambios de esquema y cero escrituras. No se toca `debts`, `debt_bills` ni migraciones.
- **Seguridad**: no se expone información nueva; se mantienen autenticación y autorización existentes. No se registran datos de deuda en logs.
- **iHost**: cero CPU/RAM adicional en el server; el frontend estático ya se sirve desde el mismo binario.
- **Accesibilidad**: WCAG 2.1 AA en contraste (4,5:1 texto normal, 3:1 texto grande y componentes), `role="progressbar"` correcto, texto al 200%, navegación por teclado si hay acción, `prefers-reduced-motion`.
- **Compatibilidad**: navegadores modernos soportados por la app (Chrome/Safari/Firefox actuales); `Intl.DateTimeFormat` y `Intl.RelativeTimeFormat` disponibles. Fallback manual si `Intl` no está disponible.
- **Mantenibilidad**: helpers puros con tests; componente sin lógica de negocio de backend; nombres de archivos y funciones en inglés, textos en i18n.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se hizo una investigación en dos frentes en paralelo:

1. **UX/UI de progreso de deuda y tiempo restante** (agente de investigación UX): se revisaron guías de Nielsen Norman Group, Apple Human Interface Guidelines, W3C/WCAG, MDN/ARIA, documentación pública de apps fintech (Monarch, YNAB, Nubank) y literatura sobre motivación/progreso.
2. **Relevamiento del código existente** (agente de exploración): se mapeó `DebtBillsPage`, `debtMonthlyAggregation.ts`, `DonutChart`, los contratos de `GET /api/debts/{id}` y `/bills`, el modelo SQLite, los charts SVG existentes y las specs previas (SPEC-054, 064, 087, 100–107, 113, 115).
3. **Revisión crítica de la spec** (tercer agente): auditó la primera versión contra el código real y detectó ambigüedades de copy, bugs de borde, problemas de contraste, plurales incompletos y decisiones pendientes (donut, timeline, estados). Esta versión incorpora esos ajustes.

Hallazgos principales:

- **La pregunta emocional es “¿cuándo termina?”**, no “¿cuál es el saldo?”. El saldo es un dato contable; el tiempo restante es finito y humano. Por eso el titular debe ser el tiempo, y la fecha/saldo quedan como datos secundarios.
- **“Último pago” es ambiguo**: puede significar el último pago realizado o la última cuota del plan. La UI debe usar “Último pago realizado” y “Última cuota” por separado, con la fecha exacta de esta última.
- **Mostrar progreso acumulado evita la sensación de empezar de cero**. El *endowed progress effect* (Nunes & Drèze, 2006) y la investigación sobre “small wins” (Amabile & Kramer, 2011) sostienen que el progreso visible —aunque sea pequeño— aumenta la persistencia. En la UI: “Ya pagaste 7 de 24 cuotas”, no solo “Te quedan 17”.
- **La barra lineal es más precisa que el donut** para comparar magnitudes: la longitud es un atributo preatentivo que se lee mejor que el ángulo/área (NN/g, *Dashboards: Making Charts and Graphs Easier to Understand*). El donut actual, además, mezclaba porcentaje por cantidad con segmentos por monto; se retira.
- **Los gauges/donuts sin número y los gráficos decorativos no aportan** en una pantalla de deuda. Un timeline de hitos y una barra bien etiquetada comunican más rápido que un chart adicional.
- **La duración debe expresarse en meses calendario**, no en días. “Te quedan 1.234 días” es ruido y aumenta la ansiedad; “2 años y 9 meses” es comprensible y finito. Para < 1 mes, “menos de un mes” es más honesto que “0 meses”. Si la fecha exacta ya pasó, hay que comparar la fecha, no solo el mes.
- **El copy no debe culpar ni generar urgencia falsa**: “Tenés 1 cuota vencida” en lugar de “estás en mora”; “Próxima cuota: 15 nov” en lugar de “última oportunidad”. Evitar “solo”, “todavía”, “deberías”, *confirmshaming* y *streaks* castigadores (FTC, *Bringing Dark Patterns to Light*; UXDA, *Dark Patterns in Banking UX*).
- **Hay que diseñar los estados borde**: deuda saldada, vencida, parcial, adelantada, sin cuotas, cuotas faltantes, fecha incierta, multimoneda. Sin estados explícitos, la UI muestra mensajes contradictorios (“Te faltan 0 meses” o “Cerrada el ”).
- **La fuente de verdad son las cuotas**: los agregados de `debts` se calculan en vivo desde `debt_bills` (no son campos persistidos), pero `GetDebt` no ejecuta `ReconcileDebtBills`. El riesgo real no es un agregado viejo, sino que falten cuotas generadas (`bills.length < installments_total`): en ese caso la UI debe advertirlo y no afirmar una fecha de fin definitiva.
- **Accesibilidad**: `role="progressbar"` con `aria-valuetext` es clave porque los lectores de pantalla suelen anunciar solo el porcentaje; el texto dentro del progressbar se vuelve presentacional (MDN). El color no puede ser el único medio (WCAG 1.4.1) y la barra necesita 3:1 contra el track (WCAG 1.4.11). En light mode, `bg-success` (#34c759) contra `bg-border` (#e5e5ea) da ~1.77:1; por eso la barra usa `bg-primary` o un token más oscuro.
- **iOS HIG**: jerarquía por importancia, *progressive disclosure*, targets de 44×44 pt, Dynamic Type hasta 200%, tipografía del sistema y `tabular-nums` para números. La tarjeta debe caber en el primer viewport sin desplazar la lista, pero puede crecer con texto grande.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **A. Solo una frase de texto** (“Te faltan 2 años y 9 meses”) | Mínimo cambio, cero riesgo, muy claro | No muestra progreso, próximo pago ni fecha; no responde “cuándo es el último pago” de forma visual | ❌ Rechazada como solución completa; se usa como titular |
| **B. Titular + fecha exacta + barra + 3 datos + timeline compacto** | Responde tiempo, progreso, último pago realizado, próxima/vencida y última cuota; accesible; sin dependencias | Requiere componente y helpers nuevos; más contenido en la card | ✅ **Seleccionada** |
| **C. Chart de líneas/barras con librería (recharts/chart.js)** | Visualmente rico; curvas de saldo | Peso de bundle, dependencia, CPU/memoria, no aporta a la pregunta principal; contradice SPEC-055/064/107 | ❌ Rechazada |
| **D. Conservar el donut actual** | Ya existe; estético | No muestra tiempo ni la última cuota; el ángulo es menos preciso; mezcla monto y cantidad; altura extra | ❌ Rechazada; se retira del detalle |
| **E. Pestaña/vista separada de “Payoff”** | No recarga el detalle | No está visible al entrar; el usuario tendría que navegar; no cumple “verlo de un vistazo” | ❌ Rechazada |
| **F. Grilla de 60 puntos, uno por cuota** | Muestra cada cuota | Ilegible en deudas largas, no escala, ruido visual | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Cambio 100% frontend, sin backend nuevo
- **Contexto**: `GET /api/debts/{id}` ya devuelve `pending_count`, `pending_amount`, `last_due_date`, `last_paid_at` y `/bills` devuelve todas las cuotas con fecha, monto y estado.
- **Decisión**: Calcular todo en el cliente y no agregar endpoints, migraciones ni queries.
- **Consecuencias**: Cero impacto en iHost y en la DB; la exactitud depende de que las cuotas estén cargadas. Si faltan cuotas generadas, se advierte (REQ-020) pero no se reconcilia desde el frontend. Si en el futuro se necesita amortización con interés, se hará en una spec aparte.

**ADR-002**: El titular es el tiempo, no el saldo
- **Contexto**: El usuario pidió explícitamente “algo más humano” y la investigación confirma que la pregunta emocional es cuándo termina.
- **Decisión**: Frase principal con años/meses; fecha exacta de la última cuota como subtítulo; saldo y próximo pago como datos secundarios.
- **Consecuencias**: Requiere pluralización y casos borde (<1 mes, vence hoy, fecha pasada, sin fecha). El saldo sigue visible para transparencia, pero no domina la jerarquía.

**ADR-003**: Barra lineal + timeline CSS/SVG, sin librería de gráficos y sin donut
- **Contexto**: El proyecto ya rechazó `recharts`/`chart.js` por el iHost (SPEC-055/064/107) y todos los charts son SVG a mano. El donut actual usa cantidad en el centro y monto en los segmentos, lo que produce dos porcentajes distintos.
- **Decisión**: Barra de progreso con `div`/CSS y timeline con posiciones porcentuales; se retira el donut del detalle. Sin dependencias.
- **Consecuencias**: Bundle mínimo, accesible y consistente con el resto; menos “glamour” visual que una librería, pero más claridad y control. Si se quiere un anillo, se evalúa en un A/B futuro (REQ-021).

**ADR-004**: Progreso por cantidad de cuotas, no por monto
- **Contexto**: El código actual usa cantidad para el porcentaje/barra y monto para el donut; con cuotas irregulares los dos indicadores no coinciden.
- **Decisión**: La métrica principal es cantidad de cuotas (“7 de 24”); el monto se muestra como dato separado (pagado/pendiente). Si los montos varían >5%, se puede mostrar un indicador secundario por monto (P1).
- **Consecuencias**: Coherencia con el lenguaje del producto (“cuotas”) y con el usuario; los montos siguen visibles. Documentar que el porcentaje es de cuotas, no de dinero.

**ADR-005**: Las cuotas son la fuente de verdad para el progreso y la próxima cuota
- **Contexto**: `GetByID` calcula `pending_count`, `pending_amount`, `last_due_date` y `last_paid_at` en vivo desde `debt_bills` (no son campos persistidos). `GetDebt` no ejecuta `ReconcileDebtBills`, pero `/bills` devuelve las mismas filas no eliminadas. El riesgo real es que falten cuotas generadas (`bills.length < installments_total`) al entrar directo por URL.
- **Decisión**: Calcular pagadas/pendientes, próxima cuota, último pago y fin estimado a partir de `bills` cuando estén disponibles; usar `debt.last_due_date` como fallback del fin y `debt.pending_count`/`pending_amount` como fallback. Normalizar fechas por prefijo `YYYY-MM-DD`/RFC3339. Si faltan cuotas generadas, mostrarlo como advertencia (REQ-020) y no afirmar que el fin es definitivo.
- **Consecuencias**: La UI refleja lo que realmente hay en la lista de cuotas; no se agrega una reconciliación oculta ni una escritura. Si `bills` está vacío, se muestra “Faltan generar cuotas” o “Sin cuotas generadas” en lugar de inventar datos.

**ADR-006**: Extraer `DebtPayoffSummary` y absorber el banner de finalizada
- **Contexto**: `DebtBillsPage` ya tiene ~270 líneas y dos specs en curso (SPEC-122 in_progress, SPEC-123 in_progress) tocan la misma página. La card de progreso y el banner de cierre duplican información.
- **Decisión**: Crear un componente nuevo que reciba `debt`, `bills`, `symbol` y `lang`; mover allí el cálculo de progreso, la línea de tiempo y los estados. La página solo importa y renderiza el componente; se elimina el banner duplicado y el chip “X de Y”.
- **Consecuencias**: Menos conflictos de merge, página más simple, una sola fuente de verdad visual. Requiere tests de los helpers y revisión manual de todos los estados. Las referencias de línea de esta spec son orientativas: SPEC-122 ya modificó `DebtBillsPage`, así que hay que rebasar y ubicar los bloques por nombre/función, no por número de línea.

**ADR-007**: Sin dark patterns ni gamificación agresiva
- **Contexto**: Las finanzas personales son sensibles; la culpa y la urgencia falsa aumentan la evitación.
- **Decisión**: Tono neutral, sin “última oportunidad”, sin “solo”, sin confeti automático, sin cuenta regresiva en días, sin streaks castigadores. Los hitos son microcopy sutil y opcional.
- **Consecuencias**: La UI puede sentirse menos “celebrada”, pero es más confiable y menos ansiosa; el estado “Deuda saldada” sigue siendo claro.

**ADR-008**: Fechas date-only y zona horaria configurada
- **Contexto**: Las fechas de cuotas son `YYYY-MM-DD`; `paid_at` puede venir como `YYYY-MM-DD`, timestamp con zona o RFC3339 desde la API (`*time.Time`). `new Date('YYYY-MM-DD')` interpreta UTC y puede mostrar el día anterior. `DueDateBadge` ya usa la zona configurada del sistema para calcular “hoy”.
- **Decisión**: Parsear date-only con regex + `new Date(y, m - 1, d)` local; normalizar `paid_at` a `YYYY-MM-DD` aceptando los formatos conocidos + RFC3339. Para “hoy” y vencidas, usar la misma zona configurada que `DueDateBadge` (o extraer su helper a un util compartido); si no está disponible, la zona del navegador. Nunca usar `toISOString()` para el día local.
- **Consecuencias**: Evita corrimientos de un día y mantiene consistencia con los badges de vencimiento. Requiere tests con zonas y formatos.

### 3.4 Referencias de la investigación

- Nielsen Norman Group — [Progress Indicators Make a Slow System Less Insufferable](https://www.nngroup.com/articles/progress-indicators/)
- Nielsen Norman Group — [Dashboards: Making Charts and Graphs Easier to Understand](https://www.nngroup.com/articles/dashboards-preattentive/)
- Apple — [Human Interface Guidelines: Progress indicators](https://developer.apple.com/design/human-interface-guidelines/progress-indicators)
- Apple — [Human Interface Guidelines: Layout](https://developer.apple.com/design/human-interface-guidelines/layout)
- Apple — [Human Interface Guidelines: Typography](https://developer.apple.com/design/human-interface-guidelines/typography)
- Apple — [Human Interface Guidelines: Accessibility](https://developer.apple.com/design/human-interface-guidelines/accessibility)
- MDN — [ARIA: progressbar role](https://developer.mozilla.org/en-US/docs/Web/Accessibility/ARIA/Reference/Roles/progressbar_role)
- W3C — [Understanding WCAG 1.4.1: Use of Color](https://www.w3.org/WAI/WCAG21/Understanding/use-of-color.html)
- W3C — [Understanding WCAG 1.4.3: Contrast (Minimum)](https://www.w3.org/WAI/WCAG21/Understanding/contrast-minimum.html)
- W3C — [Understanding WCAG 1.4.11: Non-text Contrast](https://www.w3.org/WAI/WCAG21/Understanding/non-text-contrast.html)
- W3C — [Understanding WCAG 2.5.8: Target Size (Minimum)](https://www.w3.org/WAI/WCAG22/Understanding/target-size-minimum.html)
- MDN — [prefers-reduced-motion](https://developer.mozilla.org/en-US/docs/Web/CSS/@media/prefers-reduced-motion)
- Nunes, J. C. & Drèze, X. — [The Endowed Progress Effect](https://academic.oup.com/jcr/article-abstract/32/4/504/1787425) (Journal of Consumer Research, 2006)
- Amabile, T. & Kramer, S. — [The Power of Small Wins](https://www.hbs.edu/faculty/Pages/item.aspx?num=40244) (Harvard Business School, 2011)
- YNAB — [Loan Payoff Simulator](https://www.ynab.com/blog/ynab-loan-planner)
- Monarch — [Using Pay Down Goals](https://help.monarch.com/hc/en-us/articles/44373293932052-Using-Pay-Down-Goals)
- Figma — [Nubank's design system](https://www.figma.com/customers/nubank-design-system-accessible-experiences-with-figma/)
- FTC — [Bringing Dark Patterns to Light](https://www.ftc.gov/reports/bringing-dark-patterns-light)
- UXDA — [Dark Patterns in Banking UX](https://theuxda.com/blog/dark-patterns-in-digital-banking-compromise-financial-brands)
- GOV.UK — [A to Z style guide: Dates](https://www.gov.uk/guidance/style-guide/a-to-z-of-gov-uk-style)

> Nota: las referencias fintech son documentación pública de producto, no estudios controlados. Los umbrales de las métricas son metas de diseño, no garantías. Validar con 5–10 usuarios hispanohablantes y datos sintéticos o anonimizados.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
┌─────────────────────────────┐
│ GET /api/debts/{id}         │  (existente)
│ GET /api/debts/{id}/bills   │  (existente)
└──────────────┬──────────────┘
               │  datos ya cargados en DebtBillsPage
               ▼
┌─────────────────────────────┐
│ DebtBillsPage.tsx           │
│  - header, chips, filtros   │
│  - lista de cuotas          │
│  - <DebtPayoffSummary />    │  ← nuevo
└──────────────┬──────────────┘
               │ props: debt, bills, symbol, lang
               ▼
┌─────────────────────────────┐
│ DebtPayoffSummary.tsx       │
│  - titular de tiempo        │
│  - fecha exacta última cuota│
│  - barra de progreso        │
│  - stats (último pago       │
│    realizado, próxima/vencida│
│    última cuota)            │
│  - timeline CSS/SVG         │
└──────────────┬──────────────┘
               │
               ▼
┌─────────────────────────────┐
│ utils/debtPayoff.ts         │
│  - monthsUntil()            │
│  - remainingTime()          │
│  - nextPendingBill()        │
│  - overdueBills()           │
│  - buildTimeline()          │
│  - payoffState()            │
│  - toDateOnly()             │
└─────────────────────────────┘
```

### 4.2 Componentes

#### 4.2.1 `DebtPayoffSummary` (nuevo)

- **Responsabilidad**: Renderizar el resumen de payoff above the fold y resolver los estados visuales (saldada, cuotas faltantes, sin cuotas, en pausa, vencida, al día).
- **Interfaz**:
  ```ts
  interface DebtPayoffSummaryProps {
    debt: Debt
    bills: DebtBill[]
    symbol: string
    lang: 'es' | 'en'
    now?: Date // inyectable para tests; default new Date()
  }
  ```
- **Dependencias**: helpers de `debtPayoff.ts`, `useI18nStore`, `formatMoney` de `currencyFormatStore`, `Icon`. **No** usa `DonutChart`.
- **Ubicación**: `frontend/src/components/DebtPayoffSummary.tsx`.
- **Comportamiento**: No hace fetch; no escribe estado global; calcula todo con `useMemo` a partir de props. Si `bills` está vacío, muestra el estado correspondiente y no calcula porcentajes. La barra solo se renderiza si el denominador efectivo es > 0.

#### 4.2.2 `debtPayoff.ts` (nuevo)

- **Responsabilidad**: Helpers puros, sin React, testeables con `node --test`.
- **Interfaz propuesta**:
  ```ts
  export type PayoffState = 'paid' | 'missing-bills' | 'no-bills' | 'paused' | 'overdue' | 'active'

  export interface RemainingTime {
    totalMonths: number      // puede ser negativo si la fecha ya pasó
    years: number
    months: number
    kind: 'past' | 'due-today' | 'less-than-month' | 'months'
  }

  export interface TimelinePoint {
    key: 'start' | 'today' | 'next' | 'end'
    date: string             // YYYY-MM-DD
    position: number         // 0..100
    state: 'done' | 'current' | 'future' | 'overdue'
    label?: 'next' | 'overdue'
  }

  export function parseDateOnly(iso: string | null | undefined): Date | null
  export function toDateOnly(iso: string | null | undefined): string | null // acepta date-only, timestamp y RFC3339
  export function monthsUntil(end: Date, now: Date): number
  export function remainingTime(end: Date | null, now: Date): RemainingTime
  export function nextPendingBill(bills: DebtBill[]): DebtBill | null
  export function overdueBills(bills: DebtBill[], todayISO: string): DebtBill[]
  export function paidBills(bills: DebtBill[]): DebtBill[]
  export function lastPaymentDate(debt: Debt, bills: DebtBill[]): string | null
  export function pendingEndDate(debt: Debt, bills: DebtBill[]): string | null
  export function buildTimeline(input: {
    startDate: string | null
    endDate: string | null
    nextDueDate: string | null
    todayISO: string
  }): TimelinePoint[]
  export function payoffState(debt: Debt, bills: DebtBill[], todayISO: string): PayoffState
  export function progressPct(paid: number, total: number, hasPending: boolean): number
  ```
- **Ubicación**: `frontend/src/utils/debtPayoff.ts`.
- **Reglas**:
  - Parsear fechas date-only con regex `^(\d{4})-(\d{2})-(\d{2})$` y `new Date(y, m - 1, d)` (local), **nunca** `new Date('YYYY-MM-DD')` (UTC shift).
  - `monthsUntil` cuenta meses calendario completos: `(endYear - nowYear) * 12 + (endMonth - nowMonth)`, y resta 1 si `endDay < nowDay`. No usa días/30.
  - `remainingTime`: `kind: 'past'` si la **fecha exacta** `end < now` (no solo el mes); `'due-today'` si `end` es el mismo día; `'less-than-month'` si `0 <= totalMonths < 1` con fecha futura; `'months'` en el resto.
  - `nextPendingBill` filtra `status === 'pending'`, ordena por `due_date` ascendente y desempata por `installment_number`.
  - `overdueBills` compara strings `YYYY-MM-DD` contra `todayISO`; no usa `Date` para evitar zonas horarias. Una cuota que vence hoy no está vencida.
  - `pendingEndDate`: máximo `due_date` de cuotas pendientes; si no hay, `debt.last_due_date`; si no, `null`.
  - `lastPaymentDate`: `toDateOnly(debt.last_paid_at)` si es válido; si no, máximo `paid_at` de cuotas pagadas; si no, `null`.
  - `payoffState`: `paid` solo si (`bills.length > 0` y no hay pendientes) o (`status === 'finalizada'` y `pending_count === 0` y no hay bills pendientes); `missing-bills` si `bills.length === 0 && (pending_count > 0 || installments_total > 0) && !paid`; `no-bills` si `bills.length === 0 && pending_count === 0 && installments_total === 0 && !paid`; `paused` si `status === 'inactiva'` y hay pendientes; `overdue` si hay pendientes con `due_date < todayISO`; `active` en el resto. Si `status === 'finalizada'` pero hay pendientes, nunca devolver `paid` (cae en `missing-bills` si no hay bills, o en `overdue`/`active` si hay bills).
  - `buildTimeline`: ordenar cronológicamente; incluir `start` solo si `startDate <= todayISO`; fusionar hitos con la misma fecha; omitir `next` si `nextDueDate > endDate`; si `endDate < todayISO`, marcar el extremo como `overdue`; posiciones proporcionales al rango `min(start/today) → end`, clampeadas a 0–100; si faltan fechas, devolver solo los puntos disponibles.
  - `progressPct`: si `hasPending` es `false`, 100; si `total <= 0`, 0; si no, `Math.min(99, Math.round((paid / total) * 100))`.

#### 4.2.3 `DebtBillsPage` (modificado)

- **Responsabilidad**: Mantener header, chips, filtros, lista y modal de pago; delegar el resumen al componente nuevo.
- **Cambios**:
  - Importar `DebtPayoffSummary` y eliminar `DonutChart` del page (el donut se retira del detalle).
  - Reemplazar el bloque de card de progreso (donut + barra + montos) por `<DebtPayoffSummary debt={debt} bills={bills} symbol={symbol} lang={lang} />`.
  - Eliminar el banner de deuda finalizada (lo absorbe el componente).
  - Eliminar el chip duplicado “X de Y cuotas”; conservar “Total original” (o quitar si resulta ambiguo) y “Día de pago”.
  - Eliminar el `useMemo` de `progress`, `pctColorClass` y la lógica de donut si ya no se usan.
  - Mantener `billMenuOptions`, filtros y `DebtPayModal` sin cambios (salvo que SPEC-122/123 ya los hayan modificado; rebasar).
- **Ubicación**: `frontend/src/pages/DebtBillsPage.tsx`.
- **Nota**: SPEC-122 ya modificó esta página; las referencias de línea de esta spec son orientativas. Ubicar los bloques por nombre/función y rebasar antes de implementar.

#### 4.2.4 i18n (modificado)

- **Responsabilidad**: Textos en español e inglés.
- **Ubicación**: `frontend/public/i18n/es.json` y `frontend/public/i18n/en.json`, bloque `deudas`.
- **Regla**: no editar `public/i18n/*.json` ni `frontend/src/i18n/*.json` a mano; el build los sincroniza.

### 4.3 Modelo de datos

No se modifica el esquema. Se consumen las entidades existentes:

```
Entidad: debts (sin cambios)
- id, institution_id, identifier, description
- total, principal, currency_id, installments_total, installment_amount
- interest_rate, payment_day, start_date, status
- agregados calculados en vivo: pending_count, pending_amount, paid_count,
  paid_amount, next_due_date, last_due_date, last_paid_at

Entidad: debt_bills (sin cambios)
- id, debt_id, installment_number, due_date, amount, status, paid_at
- issue_date (no se usa en esta feature)
```

**Derivaciones y fuentes de verdad**:

| Dato mostrado | Fuente primaria | Fallback | Notas |
|---|---|---|---|
| Cuotas pagadas / pendientes | `bills` (`status`) | `debt.paid_count` / `pending_count` | Si `bills` está vacío, estado “Faltan generar cuotas” o “Sin cuotas generadas” |
| Porcentaje / barra | `paid / effectiveTotal` por cantidad | `debt.paid_count / installments_total` | 100% solo si no hay pendientes; 99% cap; `effectiveTotal = max(installments_total, bills.length)` |
| Último pago realizado | `debt.last_paid_at` normalizado | máximo `paid_at` de cuotas pagadas | Aceptar `YYYY-MM-DD`, timestamp con zona y RFC3339; `toDateOnly()` |
| Próxima cuota | `bills` pendientes ordenadas por `due_date` | `debt.next_due_date` (sin número/monto) | Si `due_date < hoy`, rotular “Cuota vencida” |
| Última cuota | máximo `due_date` pendiente | `debt.last_due_date` | Mostrar día/mes/año exactos; no usar `start_date + installments_total` |
| Saldo pendiente | suma de `amount` pendientes | `debt.pending_amount` | No usar `debt.total`; se muestra en la fila de progreso |
| Total pagado | suma de `amount` de cuotas pagadas | `debt.paid_amount` | Para el estado saldada |
| Fecha de cierre | `debt.last_paid_at` | — | Si no hay pagos y la deuda está cerrada: “Cerrada manualmente”, sin fecha inventada; **no** usar `updated_at` |
| Cuotas vencidas | `bills` pendientes con `due_date < hoy` | — | Comparar strings date-only; la cuota que vence hoy no está vencida |
| Cuotas generadas vs total | `bills.length` vs `installments_total` | — | Si faltan, advertir “Según las cuotas generadas (12 de 24)” y “Faltan generar 12 cuotas” |

**Precedencia de estados** (de mayor a menor):
1. `paid`: no hay pendientes y (`bills.length > 0` o deuda `finalizada`).
2. `missing-bills`: `bills.length === 0` y hay un plan esperado (`pending_count > 0` o `installments_total > 0`) y la deuda no está cerrada.
3. `no-bills`: `bills.length === 0` y no hay plan esperado.
4. `paused`: `status === 'inactiva'` y hay pendientes.
5. `overdue`: hay pendientes con `due_date < hoy`.
6. `active`: activa con pendientes y sin vencidas.

Si `status === 'finalizada'` pero hay pendientes, se usa el estado 3/4/5/6 según corresponda y se muestra una advertencia; nunca “Deuda saldada”. Si `pending_count === 0` pero hay una cuota pendiente en `bills`, se prioriza `bills` y se advierte la inconsistencia.

### 4.4 APIs / Contratos

**No se agregan endpoints ni se modifican contratos.** Se consumen los existentes:

#### `GET /api/debts/{id}`

Campos usados:
```json
{
  "id": 0,
  "description": "Deuda Demo",
  "status": "activa",
  "start_date": "2026-11-15",
  "installments_total": 36,
  "currency_id": 0,
  "currency_code": "XXX",
  "pending_count": 27,
  "pending_amount": 0,
  "paid_count": 9,
  "paid_amount": 0,
  "next_due_date": "2026-12-15",
  "last_due_date": "2029-08-15",
  "last_paid_at": "2026-11-07 00:00:00"
}
```

#### `GET /api/debts/{id}/bills`

Campos usados:
```json
[
  {
    "installment_number": 1,
    "due_date": "2026-12-15",
    "amount": 0,
    "status": "pending",
    "paid_at": null
  }
]
```

**Errores**: los maneja la página como hoy (`LoadingSpinner`/estado vacío). El componente no hace requests; si `debt` es `null`, no se renderiza.

### 4.5 Dependencias

- **Internas**: `DebtBillsPage`, `Icon`, `useI18nStore`, `useCurrencyFormatStore`, `appStore` (`currencies`/`loadCurrencies`), tipos `Debt`/`DebtBill`.
- **Externas**: ninguna nueva. Se usan `Intl.DateTimeFormat`, `Intl.RelativeTimeFormat` y APIs estándar del navegador.
- **Prohibido**: `recharts`, `chart.js`, `d3`, `moment`, `date-fns` u otra librería de fechas/charts.

### 4.6 Diseño de UI

#### 4.6.1 Wireframe (móvil, above the fold)

```
┌────────────────────────────────────────────┐
│ ‹  Deuda Demo                              │  header global (flecha atrás)
│    Proveedor Demo · DEMO-001 · Activa      │
│                                        ⋯   │
├────────────────────────────────────────────┤
│ ESTADO DEL PLAN                  [Al día]  │
│                                            │
│ Te faltan 2 años y 9 meses                 │  ← titular, 22–24 px
│ para saldar esta deuda                     │
│ Última cuota: 15 de agosto de 2029         │
│ si seguís el plan                          │
│                                            │
│ Ya pagaste 7 de 24 cuotas           29%    │
│ ███████████░░░░░░░░░░░░░░░░░░░░░░░         │  ← barra (bg-primary / bg-border)
│ Pagado: $X (7)        Pendiente: $Y (17)   │
│                                            │
│ ┌──────────────┐ ┌──────────────┐          │
│ │ Último pago  │ │ Próxima cuota│          │
│ │ realizado    │ │ #8 · 15 nov  │          │
│ │ 12 oct 2026  │ │ $85.000      │          │
│ └──────────────┘ └──────────────┘          │
│ ┌──────────────────────────────┐           │
│ │ Última cuota                 │           │
│ │ 15 de agosto de 2029         │           │
│ └──────────────────────────────┘           │
│                                            │
│ Hoy             Próxima        Última      │
│ 7 nov 2026      15 nov 2026    15 ago 2029 │
│ ●━━━━━━━━━━━━━━━━●━━━━━━━━━━━━━━━━━━━━━━◆   │
└────────────────────────────────────────────┘
```

#### 4.6.2 Jerarquía visual

1. **Titular**: tiempo restante, 2 líneas máximo, 22–24 px, `font-bold`, `text-text`, `text-balance` si está disponible.
2. **Subtítulo**: “para saldar esta deuda” + fecha exacta de la última cuota, 14–15 px, `text-text-secondary`.
3. **Badge de estado**: “Al día”, “N vencidas”, “En pausa”, “Saldada”, “Sin cuotas”, “Faltan cuotas”; texto + ícono; no solo color.
4. **Progreso**: “Ya pagaste X de Y cuotas” + porcentaje; barra; montos pagado/pendiente.
5. **Stats**: 2 columnas en móvil angosto (<360 px) y 3 columnas desde 360 px; `dl`/`dt`/`dd` para semántica. Orden: último pago realizado, próxima/vencida, última cuota.
6. **Timeline**: 3 hitos principales (Hoy, Próxima/Vencida, Última) + Inicio opcional si ya empezó; etiquetas cortas; sin scroll horizontal de página.
7. **Acción opcional**: “Ver cuotas pendientes” como botón secundario, 44 px.

#### 4.6.3 Timeline

- **Modo proporcional**: si hay `start_date` válido y `<= hoy`, y `end_date` válido, cada hito se posiciona por tiempo transcurrido. La línea tiene dos tramos: `inicio → hoy` (transcurrido, `bg-primary`) y `hoy → fin` (pendiente, `bg-border`), con un marcador en “Hoy” y el extremo final como bandera/diamante.
- **Modos degradados**:
  - Sin `start_date` o `start_date > hoy`: omitir Inicio; mostrar “Hoy → Próxima → Última” y, si aplica, el texto “Empieza el {fecha}”.
  - Sin `end_date`: mostrar “Hoy → Próxima” y el estado “Sin fecha de fin”.
  - `end < start` o fechas inválidas: mostrar las fechas como lista, sin línea proporcional.
  - `today > end`: clamp al 100% y etiqueta “Última cuota vencida”.
  - `next_due < today`: ubicar antes de Hoy y rotular “Cuota vencida”.
  - `next_due === start` o `next_due === today`: fusionar hitos.
  - `next_due > end` o inexistente: omitir el hito.
- **Etiquetas**: `Inicio` (opcional), `Hoy`, `Próxima cuota`/`Cuota vencida`, `Última cuota`.
- **Accesibilidad**: la línea visual es `aria-hidden="true"`; un párrafo `sr-only` localizado resume las fechas exactas (“Inicio 15 nov 2026; hoy 7 nov 2026; última cuota 15 ago 2029”). No usar simultáneamente un `<ol>` visible y un `sr-only` que dupliquen el anuncio.
- **Sin animación** por defecto; si se anima el ancho, `transition-[width] duration-300 motion-reduce:transition-none`.

#### 4.6.4 Estados y copy

| Estado | Condición | Titular | Badge | Acción/CTA |
|---|---|---|---|---|
| **Al día** | `active`, pendientes, sin vencidas | “Te faltan 2 años y 9 meses…” | “Al día” | “Ver cuotas pendientes” (opcional) |
| **Vencida** | pendientes con `due_date < hoy` | “Si te ponés al día, terminás en 15 de agosto de 2029” + “Tenés 2 cuotas vencidas” | “2 vencidas” (ámbar + ícono) | “Ver cuotas pendientes” |
| **En pausa** | `status === 'inactiva'` y pendientes | “Quedan 17 cuotas pendientes” + “Si reanudás, la última cuota sería el 15 de agosto de 2029” | “En pausa” | “Ver cuotas pendientes” |
| **Saldada** | sin pendientes y bills > 0 o finalizada | “Deuda saldada” + fecha + total pagado | “Saldada” (verde + check) | — |
| **Cerrada manualmente** | `status === 'finalizada'`, sin pagos ni pendientes | “Deuda cerrada manualmente” + “Sin pagos registrados” | “Cerrada” | — |
| **Sin cuotas** | `bills.length === 0` y no hay plan esperado | “Sin cuotas generadas” | “Sin cuotas” | — |
| **Faltan cuotas** | `bills.length === 0` y plan esperado, o `bills.length < installments_total` | “Faltan generar cuotas” / “Según las cuotas generadas (12 de 24)” | “Revisar” | — |
| **Fecha vencida** | fin < hoy con pendientes | “La última cuota venció el 5 de noviembre de 2026; revisá tus cuotas” | “Vencida” | “Ver cuotas pendientes” |
| **Dato inconsistente** | `status === 'finalizada'` con pendientes, o `pending_count === 0` con bill pendiente | Estado con pendientes + advertencia | “Revisar” | “Ver cuotas pendientes” |

#### 4.6.5 Claves i18n propuestas

Agregar al bloque `deudas` en `frontend/public/i18n/es.json` y `en.json`:

| Clave | Español | Inglés |
|---|---|---|
| `total_original` | Total original | Original total |
| `payoff_eyebrow` | Estado del plan | Plan status |
| `payoff_remaining_less_than_month` | Te falta menos de un mes para saldar esta deuda | You have less than a month left to pay off this debt |
| `payoff_remaining_due_today` | La última cuota vence hoy | The last installment is due today |
| `payoff_remaining_one_month` | Te falta 1 mes para saldar esta deuda | You have 1 month left to pay off this debt |
| `payoff_remaining_months` | Te faltan {months} meses para saldar esta deuda | You have {months} months left to pay off this debt |
| `payoff_remaining_one_year_one_month` | Te falta 1 año y 1 mes para saldar esta deuda | You have 1 year and 1 month left to pay off this debt |
| `payoff_remaining_one_year_months` | Te falta 1 año y {months} meses para saldar esta deuda | You have 1 year and {months} months left to pay off this debt |
| `payoff_remaining_one_year` | Te falta 1 año para saldar esta deuda | You have 1 year left to pay off this debt |
| `payoff_remaining_years_one_month` | Te faltan {years} años y 1 mes para saldar esta deuda | You have {years} years and 1 month left to pay off this debt |
| `payoff_remaining_years_months` | Te faltan {years} años y {months} meses para saldar esta deuda | You have {years} years and {months} months left to pay off this debt |
| `payoff_remaining_years` | Te faltan {years} años para saldar esta deuda | You have {years} years left to pay off this debt |
| `payoff_remaining_past` | La última cuota venció el {date}; revisá tus cuotas | The last installment was due on {date}; check your installments |
| `payoff_final_installment` | Última cuota | Last installment |
| `payoff_final_installment_hint` | si seguís el plan | if you follow the plan |
| `payoff_final_installment_unknown` | No pudimos estimar la fecha de la última cuota | We couldn't estimate the last installment date |
| `payoff_paid_of` | Ya pagaste {paid} de {total} cuotas | You've paid {paid} of {total} installments |
| `payoff_paid_of_one` | Ya pagaste {paid} de 1 cuota | You've paid {paid} of 1 installment |
| `payoff_paid_single_done` | Ya pagaste la única cuota | You've paid the only installment |
| `payoff_progress_label` | Progreso de pago | Payment progress |
| `payoff_aria_progress` | {paid} de {total} cuotas pagadas, {pct}% | {paid} of {total} installments paid, {pct}% |
| `payoff_last_payment` | Último pago realizado | Last payment made |
| `payoff_no_payments` | Sin pagos registrados | No payments recorded |
| `payoff_next_installment` | Próxima cuota | Next installment |
| `payoff_overdue_installment` | Cuota vencida | Overdue installment |
| `payoff_no_pending` | Sin cuotas pendientes | No pending installments |
| `payoff_remaining_balance` | Saldo pendiente | Remaining balance |
| `payoff_overdue_one` | {count} cuota vencida | {count} overdue installment |
| `payoff_overdue_many` | {count} cuotas vencidas | {count} overdue installments |
| `payoff_on_track` | Al día | On track |
| `payoff_paused` | En pausa | Paused |
| `payoff_paused_note` | No se generan cuotas nuevas mientras esté en pausa. | No new installments are generated while paused. |
| `payoff_catch_up` | Si te ponés al día, terminás en {date} | If you catch up, you'll pay it off by {date} |
| `payoff_settled` | Deuda saldada | Debt paid off |
| `payoff_settled_on` | Deuda saldada el {date} | Paid off on {date} |
| `payoff_settled_total` | Total pagado: {amount} | Total paid: {amount} |
| `payoff_badge_settled` | Saldada | Paid off |
| `payoff_badge_no_bills` | Sin cuotas | No installments |
| `payoff_badge_review` | Revisar | Review |
| `payoff_closed_manually` | Deuda cerrada manualmente | Debt closed manually |
| `payoff_no_installments` | Sin cuotas generadas | No installments generated |
| `payoff_no_installments_detail` | Esta deuda no tiene cuotas registradas para mostrar. | This debt has no installments to show. |
| `payoff_missing_bills` | Faltan generar cuotas | Missing installments |
| `payoff_missing_bills_detail` | No pudimos cargar el plan completo de esta deuda. | We couldn't load the full installment plan. |
| `payoff_generated_of` | Según las cuotas generadas ({generated} de {total}) | Based on generated installments ({generated} of {total}) |
| `payoff_missing_count` | Faltan generar {count} cuotas | {count} installments are missing |
| `payoff_timeline_title` | Línea de tiempo | Timeline |
| `payoff_timeline_start` | Inicio | Start |
| `payoff_timeline_today` | Hoy | Today |
| `payoff_timeline_next` | Próxima cuota | Next installment |
| `payoff_timeline_overdue` | Cuota vencida | Overdue installment |
| `payoff_timeline_end` | Última cuota | Last installment |
| `payoff_timeline_overdue_end` | Última cuota vencida | Overdue last installment |
| `payoff_view_pending` | Ver cuotas pendientes | View pending installments |
| `payoff_how_calculated` | Cómo se calcula | How it's calculated |
| `payoff_how_calculated_detail` | Usamos la fecha de tu última cuota pendiente. Si adelantás la última cuota o cambiás el plan, la fecha se actualiza. | We use the date of your last pending installment. If you pay the last installment ahead or change the plan, the date updates. |
| `payoff_inconsistent_status` | Hay cuotas pendientes con la deuda marcada como finalizada. | There are pending installments but the debt is marked as paid off. |
| `payoff_plan_incomplete` | Hay cuotas generadas de menos para este plan. | This plan has fewer generated installments than expected. |

> Regla: `t()` no interpola; usar `.replace('{months}', String(...))`, etc. Elegir la clave singular/plural antes de interpolar. Para el aria, usar `payoff_aria_progress` y no hardcodear español.

### 4.7 Accesibilidad y dark mode

- **Contraste**: `text-text` sobre `bg-card`; `text-text-secondary` solo para texto secundario. La barra usa `bg-primary` contra `bg-border` (≥3:1 en light y dark); los badges usan tintes (`bg-success/20`, `bg-warning/20`) + `text-text` + ícono. Verificar ratios reales en light/dark; no asumir que `bg-success` sobre `bg-border` cumple (da ~1.77:1 en light).
- **ARIA**:
  ```tsx
  <span id="payoff-progress-label" className="text-sm text-text">
    {t('payoff_progress_label')}
  </span>
  <div
    role="progressbar"
    aria-labelledby="payoff-progress-label"
    aria-valuemin={0}
    aria-valuemax={effectiveTotal}
    aria-valuenow={Math.min(paid, effectiveTotal)}
    aria-valuetext={t('payoff_aria_progress')
      .replace('{paid}', String(paid))
      .replace('{total}', String(effectiveTotal))
      .replace('{pct}', String(pct))}
    className="h-2.5 overflow-hidden rounded-full bg-border"
  >
    <div className="h-full rounded-full bg-primary" style={{ width: `${pct}%` }} />
  </div>
  ```
  El texto “Ya pagaste…” va **fuera** del `role="progressbar"`. No renderizar este bloque si `effectiveTotal === 0`.
- **Timeline**: `aria-hidden="true"` en la línea visual; un `<p className="sr-only">` localizado con “Inicio 15 nov 2026; hoy 7 nov 2026; última cuota 15 ago 2029”. No usar simultáneamente un `<ol>` visible y un `sr-only` que dupliquen el anuncio.
- **Movimiento**: `motion-reduce:transition-none`; sin animación de conteo; sin confeti automático.
- **Texto**: soportar 200% sin truncar; `min-h-11` en botones; no alturas fijas en contenedores de texto; `tabular-nums` en montos/porcentajes; no `truncate` en montos.
- **Orden de lectura**: titular → subtítulo → estado → progreso → stats → timeline → CTA.

### 4.8 Archivos afectados

| Archivo | Acción |
|---|---|
| `frontend/src/components/DebtPayoffSummary.tsx` | Crear |
| `frontend/src/utils/debtPayoff.ts` | Crear |
| `frontend/tests/debtPayoff.test.ts` | Crear |
| `frontend/src/pages/DebtBillsPage.tsx` | Modificar (importar/renderizar, quitar card/banner/chip duplicado) |
| `frontend/public/i18n/es.json` | Modificar (bloque `deudas`) |
| `frontend/public/i18n/en.json` | Modificar (bloque `deudas`) |
| `docs/specs/README.md` | Actualizar tracker en la rama de la spec por pedido explícito del usuario; el release reconcilia el tracker en `main` (SPEC-066) |

**No se modifica**: `internal/`, `migrations/`, `cmd/`, `go.mod`, `frontend/package.json`, `frontend/vite.config.ts`, `public/i18n/` (generado), `frontend/src/i18n/` (generado).

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] **CA-001**: Dada una deuda activa con cuotas pendientes cuya última cuota vence el 2029-08-15, cuando se abre `/deudas/:id` el 2026-11-07, entonces el titular muestra “Te faltan 2 años y 9 meses para saldar esta deuda” y el subtítulo “Última cuota: 15 de agosto de 2029”, visibles sin scroll en 375×667 px.
- [ ] **CA-002**: Dada una deuda con 1 mes restante, entonces muestra “Te falta 1 mes para saldar esta deuda”; con 2 meses, “Te faltan 2 meses…”.
- [ ] **CA-003**: Dadas las combinaciones 13, 14, 25 y 26 meses, entonces muestra “Te falta 1 año y 1 mes…”, “Te falta 1 año y 2 meses…”, “Te faltan 2 años y 1 mes…” y “Te faltan 2 años y 2 meses…”, respectivamente; en inglés, “1 year and 1 month”, “1 year and 2 months”, “2 years and 1 month”, “2 years and 2 months”.
- [ ] **CA-004**: Dada una deuda que termina dentro del mes actual con fecha futura, entonces muestra “Te falta menos de un mes para saldar esta deuda”; si la última cuota vence hoy, “La última cuota vence hoy”.
- [ ] **CA-005**: Dada una deuda cuya última cuota venció ayer, entonces muestra “La última cuota venció el {fecha}; revisá tus cuotas” (no “Te falta menos de un mes”).
- [ ] **CA-006**: Dada una deuda saldada, entonces muestra “Deuda saldada” + fecha de cierre + total pagado; no muestra countdown ni barra de tiempo; el 100% es correcto.
- [ ] **CA-007**: Dada una deuda con cuotas vencidas, entonces muestra “Tenés N cuotas vencidas” con ícono + texto, sin pintar toda la card de rojo; el titular dice “Si te ponés al día, terminás en {fecha}”.
- [ ] **CA-008**: Dada una deuda `inactiva` con pendientes, entonces muestra “En pausa” + “Quedan N cuotas pendientes” y aclara que no se generan cuotas nuevas; si hay vencidas, las muestra como alerta secundaria.
- [ ] **CA-009**: Dada una deuda sin cuotas con plan esperado (`pending_count > 0` o `installments_total > 0`), entonces muestra “Faltan generar cuotas” y no calcula porcentaje ni fecha de fin; si no hay plan esperado, muestra “Sin cuotas generadas”. No muestra `NaN`, `0%` ni fechas inventadas.
- [ ] **CA-010**: Dada una deuda `finalizada` con cuotas pendientes, entonces prioriza el estado con pendientes y muestra la advertencia; no muestra “Deuda saldada”.
- [ ] **CA-011**: Dada una deuda `finalizada` sin pagos ni pendientes, entonces muestra “Deuda cerrada manualmente” + “Sin pagos registrados”, sin fecha inventada ni “Total pagado: $0”.
- [ ] **CA-012**: El bloque muestra el último pago realizado (`YYYY-MM-DD` normalizado) o “Sin pagos registrados”; nunca deja una fecha vacía.
- [ ] **CA-013**: El bloque muestra la próxima cuota con número, fecha y monto; si la más antigua venció, la rotula “Cuota vencida”; si no hay pendientes, “Sin cuotas pendientes”.
- [ ] **CA-014**: El bloque muestra la fecha exacta de la última cuota (día/mes/año) o “No pudimos estimar la fecha de la última cuota”; no muestra solo el mes.
- [ ] **CA-015**: El saldo pendiente es la suma de las cuotas pendientes (no `debt.total`) y se formatea con el símbolo de la moneda; si no hay símbolo, usa el código ISO.
- [ ] **CA-016**: La barra de progreso es por cantidad de cuotas; muestra 100% solo cuando no hay pendientes y limita a 99% cuando queda al menos una; no se renderiza si el denominador es 0.
- [ ] **CA-017**: La línea de tiempo muestra Hoy, Próxima/Vencida y Última cuota en orden cronológico; incluye Inicio solo si `start_date <= hoy`; fusiona hitos con la misma fecha; si el fin ya pasó, lo marca como vencido.
- [ ] **CA-018**: Si `bills.length < installments_total`, muestra “Según las cuotas generadas (12 de 24)” y “Faltan generar 12 cuotas”; no afirma que el fin es definitivo.
- [ ] **CA-019**: Todos los textos nuevos están en `frontend/public/i18n/es.json` y `en.json`; no hay strings hardcodeados; singular/plural correctos en ambos idiomas (incluidos 1/1, 1 año + 1 mes, N años + 1 mes).
- [ ] **CA-020**: La UI usa tokens del tema y se ve correctamente en light y dark mode; no hay colores hardcodeados en el componente nuevo; la barra cumple ≥3:1 contra el track.
- [ ] **CA-021**: En móvil de 320 px no hay overflow horizontal; a 200% de zoom los datos críticos siguen legibles y no se truncan montos; los targets de acción son ≥44 px.
- [ ] **CA-022**: VoiceOver/TalkBack anuncia “X de Y cuotas pagadas, Z%” mediante `aria-valuetext` localizado; la información de la línea de tiempo también está disponible en texto.
- [ ] **CA-023**: Con `prefers-reduced-motion: reduce`, no hay animaciones de barra ni de la línea de tiempo.
- [ ] **CA-024**: No se agregan endpoints ni dependencias npm; la página sigue usando los dos requests existentes (más el GET existente de monedas si hace falta).
- [ ] **CA-025**: `npm run build` en `frontend/` pasa (incluye `check:ui` y `sync-i18n`), `node --test frontend/tests/debtPayoff.test.ts` pasa y `go test ./...` sigue pasando.
- [ ] **CA-026**: Las fechas date-only se parsean sin corrimiento UTC; el 2026-11-07 se muestra como 7 nov 2026 (no 6 nov); los `paid_at` en formato RFC3339 y timestamp también se normalizan.
- [ ] **CA-027**: No se muestra el donut, el banner duplicado de “Cerrada el…” ni el chip duplicado “X de Y cuotas”; hay un solo bloque de progreso.
- [ ] **CA-028**: El CTA “Ver cuotas pendientes” (si se implementa) cambia el filtro y hace scroll después de que la lista se renderiza; no deja la pantalla en un lugar incorrecto.
- [ ] **CA-DARK**: Si la spec toca formularios/modales, los inputs usan tokens del tema (`bg-card`, `text-text`) y se verificó legibilidad en darkmode. **N/A**: no hay formularios nuevos; igual se verifica la card en darkmode.
- [ ] **CA-BACK**: Si la spec crea una página de detalle o formulario con lista padre, el retorno se hace con la flecha atrás del header (registrada en `BACK_ROUTES`) y NO existen links “← Título” dentro del contenido. **N/A**: la ruta `/deudas/:id` ya está registrada en `BACK_ROUTES`; no se agrega una página nueva.

### 5.2 No funcionales

- [ ] **CA-NF-001**: El render del bloque no bloquea la página; el cálculo es O(n) y no hay endpoints nuevos.
- [ ] **CA-NF-002**: El bundle de frontend no incorpora librerías nuevas; el incremento estimado es < 5 KB minificado.
- [ ] **CA-NF-003**: Contraste WCAG AA en light/dark; la barra y los badges no dependen solo del color.
- [ ] **CA-NF-004**: No se escriben datos en la DB; no se toca `data/app.db` durante la implementación.
- [ ] **CA-NF-005**: La card no desborda en 320 px y no empuja la lista fuera del primer viewport en 375×667 px a texto 100% (objetivo ≤ 400 px de alto); a 200% puede crecer sin truncar.

### 5.3 Testing

- **Unit tests** (`frontend/tests/debtPayoff.test.ts`, `node --test`):
  - `monthsUntil`: 33, 32 (día anterior), 25, 26, 13, 14, 12, 1, 0, negativo.
  - `remainingTime`: `past` (ayer, mismo mes), `due-today`, `less-than-month`, `months`, 1 año + 1 mes, 1 año + N meses, N años + 1 mes, N años + N meses.
  - `nextPendingBill`: orden por `due_date`, desempate por `installment_number`, ignora pagadas, pagos adelantados, cuota vencida.
  - `overdueBills`: compara date-only, no marca las de hoy, incluye vencidas de meses anteriores.
  - `lastPaymentDate`: `last_paid_at` con `YYYY-MM-DD`, `YYYY-MM-DD HH:MM:SS`, `YYYY-MM-DD HH:MM:SS +0000 UTC` y RFC3339; fallback a `paid_at`; sin pagos → `null`.
  - `pendingEndDate`: máximo `due_date` pendiente, fallback `last_due_date`, sin datos → `null`.
  - `buildTimeline`: posiciones 0/100, today clamp, sin start, start futuro, start == today, next == start, next < today, next > end, end < start, end pasado, >36 meses.
  - `payoffState`: paid > missing-bills > no-bills > paused > overdue > active; finalizada con pendientes; `pending_count=0` con bill pendiente; `bills=[] + pending_count>0`; activa sin pendientes; cierre manual sin pagos.
  - `progressPct`: 0, 50, 99 cap, 100 solo sin pendientes, total 0, `bills.length > installments_total`.
- **Integration/manual**:
  - Deuda activa al día, con cuotas vencidas (incluida una vencida ayer en el mismo mes), en pausa, en pausa con vencidas, saldada, cerrada manualmente sin pagos, sin cuotas, con cuotas faltantes (`bills < installments_total`), finalizada con pendientes, con pagos adelantados, con una sola cuota, con 24+ cuotas, con fin en el mes actual, con fin hoy, con fin pasado, multimoneda, deuda larga (>10 años, >36 meses).
  - Entrada directa por URL `/deudas/:id` (sin pasar por el listado) para verificar que el catálogo de monedas y los datos se resuelven.
  - Móvil 320/375/390, tablet, desktop; light/dark; 200% zoom; VoiceOver/TalkBack en español e inglés; `prefers-reduced-motion`.
  - Verificar que no haya overflow horizontal ni duplicación del banner/chip/donut.
- **Build/QA**:
  - `cd frontend && npm run build` (incluye `check:ui` y `sync-i18n`).
  - `node --test frontend/tests/debtPayoff.test.ts`.
  - `go test ./...` (sin cambios de backend, pero se corre igual).
  - `curl -s http://localhost:8088/i18n/es.json` para confirmar las claves nuevas servidas.
- **Datos reales**: si la validación depende de datos reales, hacer un backup consistente de la DB de producción a `/tmp` (`.backup`, `PRAGMA integrity_check`) y correr el server con `DATA_DIR` apuntando a la copia. Nunca escribir en producción ni en `data/app.db` local.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Helpers puros `debtPayoff.ts` + tests unitarios | 0.5 día | Ninguna |
| 2 | Componente `DebtPayoffSummary` + integración en `DebtBillsPage` | 1.5 días | Fase 1 |
| 3 | i18n es/en, tokens/dark mode, responsive, accesibilidad y timeline | 1 día | Fase 2 |
| 4 | QA manual (matriz de estados), build, documentación y ajustes | 1 día | Fase 3 |

**Total estimado**: 4 días de desarrollo + QA.

### 6.2 Milestones

1. **MVP**: titular de tiempo restante + fecha exacta de la última cuota + barra + último pago realizado + próxima/vencida + última cuota; estados principales (al día, saldada, sin cuotas, faltan cuotas).
2. **V1.0**: timeline robusta, estados vencida/en pausa/inconsistente, i18n completa, dark mode, accesibilidad, tests y QA mobile.

### 6.3 Coordinación con specs en curso

- **SPEC-122** (`in_progress`, worktree `/home/paulomcnally/github/p40la-ihost-spec-122`): ya modificó `DebtBillsPage`, `DebtPayModal` y la lista de cuotas (comprobantes PDF). Esta spec **no** toca esos menús; extraer `DebtPayoffSummary` reduce el conflicto. Rebasear sobre su rama antes de implementar.
- **SPEC-123** (`in_progress`, worktree `/home/paulomcnally/github/p40la-ihost-spec-123`): agrega “Contrato” al `CreateMenu` del header del detalle. Esta spec no toca el header.
- **SPEC-124** (`feature/SPEC-124` reservado por otra sesión): no tocar ese worktree.
- **Recomendación**: implementar y mergear SPEC-125 después de SPEC-122/123, o rebasar sobre sus ramas antes del release. Verificar `git diff` de `DebtBillsPage.tsx` antes de mergear.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Cuotas faltantes al entrar directo por URL (`GetDebt` no reconcilia) | Media | Alto | Advertir “Faltan generar cuotas” y “Según las cuotas generadas”; no afirmar que el fin es definitivo; no reconciliar desde el frontend |
| Desvío entre `start_date + installments_total` y las cuotas reales por ediciones | Alta | Medio | Usar `last_due_date`/máximo `due_date` pendiente, no `debtEndDate`; mostrar la fecha exacta de la última cuota real |
| `paid_at` con formatos heterogéneos + RFC3339 | Alta | Medio | `toDateOnly()` acepta date-only, timestamp con zona y RFC3339; tests con todos los formatos |
| Corrimiento de fecha por UTC (`new Date('YYYY-MM-DD')`) o zona horaria | Media | Alto | Parsear date-only local; usar la zona configurada de `DueDateBadge` para “hoy”; test explícito |
| Deudas largas con 100+ cuotas | Media | Bajo | No renderizar un punto por cuota; timeline con 3 hitos y sin marcas por año; advertir el horizonte |
| Timeline con fechas superpuestas o invertidas | Media | Medio | Orden cronológico, fusión de hitos iguales, omitir inicio futuro, `next > end`, clamp y etiquetas “Cuota vencida” |
| Conflictos de merge con SPEC-122/123 en `DebtBillsPage` | Alta | Medio | Extraer componente; tocar la página lo mínimo; rebasar/coordinar orden de merge; no usar números de línea |
| Gramática/plurales incorrectos en es/en | Alta | Medio | Claves de frase completa para todas las combinaciones; revisión manual; tests de selección de variante |
| Contraste insuficiente en light mode | Alta | Alto | Barra con `bg-primary`/token más oscuro; badges con tinte + `text-text` + ícono; medir ratios reales |
| Regresión de accesibilidad | Media | Alto | `role="progressbar"` con `aria-valuetext` localizado; `aria-valuemax` efectivo; VoiceOver; 200%; reduced motion |
| El usuario espera un “gráfico” más llamativo | Media | Bajo | Explicar la decisión UX (barra + timeline); el donut se retira por mezclar métricas; A/B futuro si hace falta |
| Scope creep (simulador de pagos extra, intereses) | Media | Medio | Mantener fuera de alcance; spec separada si se pide |

## 8. Notas y Referencias

- **Specs relacionadas**:
  - SPEC-054: módulo de deudas y generación de cuotas.
  - SPEC-064: progreso visual de cuotas en el detalle (donut + barra).
  - SPEC-087: filtros de cuotas (`pending`/`all`/`paid`).
  - SPEC-100/101/102: tab Gráfica, cronograma y responsive.
  - SPEC-104: por pagar vs historial, cuotas adelantadas y agregados (`pending_count`, `last_due_date`, `last_paid_at`).
  - SPEC-105/106/107: historial 12 meses y proyección escalonada.
  - SPEC-113: formato de moneda y símbolo.
  - SPEC-115: menú de acciones del detalle.
  - SPEC-122: comprobantes PDF en pagos de deudas (in_progress).
  - SPEC-123: contratos PDF de deudas/servicios (in_progress).
- **Documentación técnica**:
  - `frontend/src/pages/DebtBillsPage.tsx`
  - `frontend/src/utils/debtMonthlyAggregation.ts`
  - `frontend/src/components/DonutChart.tsx` (se retira del detalle, no se elimina del repo)
  - `frontend/src/components/DueDateBadge.tsx` (zona horaria configurada)
  - `internal/storage/debt.go` (agregados de deuda)
  - `internal/services/debt.go` (generación de cuotas)
- **Reglas del repo**: `AGENTS.md`, `docs/project-rules.md` (i18n en `frontend/public/i18n/`, dark mode, worktrees, datos sintéticos).
- **Datos sintéticos**: los ejemplos de esta spec usan `Deuda Demo`, `Proveedor Demo`, `DEMO-001`, fechas y montos ficticios. No contienen PII ni datos financieros reales.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-07 | opencode | Creación inicial de la especificación (draft): resumen humano de tiempo restante, último pago y fin estimado en el detalle de deuda; investigación UX con multi-agentes; sin cambios de backend. |
| 2026-11-07 | opencode | Revisión crítica incorporada: se separa “último pago realizado” de “última cuota”, se muestra la fecha exacta, se corrige la timeline (orden/futuro/vencida), se agregan estados `missing-bills` y cierre manual, se completan plurales i18n, se cambia la barra a `bg-primary` por contraste, se retira el donut y se amplía la matriz de tests. |
| 2026-11-07 | opencode | Issue de GitHub #128 creado con label `spec/draft`; tracker actualizado en la rama de la spec. |
| 2026-11-07 | opencode | Implementación frontend: helpers `debtPayoff.ts` y `date.ts`, componente `DebtPayoffSummary`, integración en `DebtBillsPage`, i18n es/en, tests unitarios (14/14) y build; validación en Chrome headless con DB sintética (al día, vencida, pausa, saldada, cuotas faltantes, inconsistente y “2 años y 9 meses”). |
