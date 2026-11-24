---
title: "MCP Fase 2: catálogo de tools para funcionalidades web existentes"
id: "SPEC-154"
status: "released"
author: "opencode"
created: "2026-11-24"
updated: "2026-11-24"
github_issue: 157
---

# MCP Fase 2: catálogo de tools para funcionalidades web existentes

**ID**: SPEC-154<br>
**Estado**: released<br>
**Autor**: opencode<br>
**Creado**: 2026-11-24<br>
**Actualizado**: 2026-11-24

---

## 1. Resumen Ejecutivo

El servidor MCP de p40la-ihost expone hoy **20 tools** (5 de lectura y 15 de escritura) sobre un backend que ya tiene **317 rutas HTTP únicas**: 239 en `internal/api/routes.go` y 78 en módulos con namespace (`realestate.rental`, `realestate.portfolio`, `home.maintenance`, `bank.creditcard`). La cobertura es real pero despareja: Facturas, Autos, Entidades y Banca tienen tools; **Familia/Pensión, Deudas, Alquileres, Presupuesto, Reparaciones del hogar, Automatizaciones, Alertas, Webhooks, Documentos y Configuración tienen cero**.

Esta spec es un **diagnóstico y una hoja de ruta**, no una implementación. Su objetivo es identificar, con evidencia de código, qué tools nuevas se pueden construir **reutilizando endpoints que ya existen**, en qué orden conviene hacerlo y qué no debe exponerse nunca. El resultado es un catálogo de **51 tools candidatas en 3 olas** (38 de lectura y 13 de escritura acotada), una lista separada de **20 operaciones de alto riesgo** que requieren spec propia y confirmación reforzada, y una **lista de no-go** explícita (borrados, binarios, secretos, PII).

**Alcance de implementación**: esta spec implementa la **Ola 1** (19 tools de lectura; `debts_list` queda diferida por `G1`). Las olas 2 y 3 quedan como hoja de ruta para specs posteriores. La ola 1 no agrega dependencias, migraciones ni rutas de backend: reutiliza endpoints existentes y agrega un paquete por dominio en `internal/mcp/tools/`.

---

## 2. Cumplimiento de reglas

Reglas aplicables según `docs/rules/00-index.md`. La spec **no toca frontend, ni rutas de UI, ni esquema de base de datos, ni escribe código**: es un documento de análisis y planificación. Las reglas de UI/UX se declaran `N/A` con justificación.

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | El catálogo solo reutiliza endpoints HTTP existentes con `net/http`; no propone dependencias nuevas | `CA-ARCH-STACK` |
| `ARCH-FOLDERS` | Las tools de cada ola vivirán en `internal/mcp/tools/<dominio>/`; esta spec no crea archivos de código | `CA-ARCH-FOLDERS` |
| `ARCH-LAYERS` | Las tools llaman a la API HTTP; **prohibido** SQL directo desde el MCP. Toda lectura pasa por handler → servicio → storage | `CA-ARCH-LAYERS` |
| `ARCH-MODULES` | Los módulos con namespace (`realestate.rental`, `home.maintenance`, `bank.creditcard`) se consumen por sus rutas; no se importan sus `storage` | `CA-ARCH-MODULES` |
| `ARCH-MCP-TOOLS-MAP` | Toda tool futura exige entrada en `docs/mcp/tools/impact.json`; esta spec no registra tools y el manifiesto queda intacto | `CA-ARCH-MCP-TOOLS-MAP` |
| `ARCH-IHOST` | El cliente MCP no tiene estado ni caché; la ola 1 solo agrega lecturas con salida acotada | `CA-ARCH-IHOST` |
| `ARCH-GO` | Las tools futuras siguen `gofmt`, nombres en inglés y errores explícitos (patrón `toolkit.ToolError`) | `CA-ARCH-GO` |
| `DATA-PROTECTED-SCHEMA` | **N/A**: la spec no modifica ni propone modificar `debts` / `debt_bills`; las tools de deuda solo usan endpoints existentes de insert/update | — |
| `DATA-MIGRATIONS` | **N/A**: no hay migraciones en esta spec ni en las olas 1–3 | — |
| `DATA-NO-DROP` | **N/A**: no se ejecuta nada sobre `data/app.db` | — |
| `DATA-PRODCOPY` | **N/A**: el análisis es estático (código y rutas), no depende del contenido de la DB. Si una ola futura valida comportamiento con datos, aplicará la regla | — |
| `SEC-NO-PII` | El catálogo describe endpoints y **nombres de campos**, nunca payloads. Se marcan como PII `alerts.speech` (nombre personal), `alert_emails`, `telegram_bot_chat_ids`, `notifications.email`, `automation_accounts.identifier` y referencias de comprobantes | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Todos los ejemplos usan `demo@example.com`, `demo-account-001`, `+1 555 0100`, `Proveedor Demo`, `F000000000000001` | `CA-SEC-SYNTHETIC` |
| `SEC-LOGS` | Las tools futuras registran IDs y conteos, no montos, emails ni referencias | `CA-SEC-LOGS` |
| `SEC-COMMIT` | `git diff --cached` se revisa antes del commit; la spec no incluye DBs, PDFs ni exports | `CA-SEC-COMMIT` |
| `TEST-LOCAL` | El gate de cada ola futura exige `go build ./...` y `go test ./...` | `CA-TEST-LOCAL` |
| `TEST-MCP-CONTRACT` | Cada tool nueva obliga a actualizar `docs/mcp/tools/contracts.json` de forma intencional y a correr `TestToolContracts` + `TestMCPBackendContract` | `CA-TEST-MCP-CONTRACT` |
| `DELIV-WORKTREE` | La spec se crea en `../p40la-ihost-spec-154` sobre `feature/SPEC-154`; `./scripts/check-worktree.sh` pasa | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | No hay merge ni push en este ciclo | `CA-DELIV-CONFIRM` |
| `DELIV-MCP-IMPACT` | Sección `## Impacto en tools MCP` completa y `./scripts/check-mcp-impact.sh --files docs/specs/...` → «sin tools afectadas» | `CA-DELIV-MCP-IMPACT` |
| `DELIV-MCP-IMPACT-SUBAGENT` | Análisis delegado a un subagente `general`; veredicto citado en la sección 6 | `CA-DELIV-MCP-IMPACT-SUBAGENT` |
| `DELIV-MCP-DOCS` | Cada ola futura actualiza `docs/mcp/` en el mismo commit; esta spec no toca superficie MCP | `CA-DELIV-MCP-DOCS` |
| `DELIV-MCP-DOCS-SUBAGENT` | Las docs de cada ola se delegan a un subagente en paralelo a la implementación | `CA-DELIV-MCP-DOCS-SUBAGENT` |
| `DELIV-LABELS` | Issue con label `spec/draft`; sin labels genéricos | `CA-DELIV-LABELS` |
| `UI-*` (todas) | **N/A**: la spec no toca frontend ni i18n | — |
| `UX-*` (todas) | **N/A**: la spec no agrega pantallas, rutas, modales ni formularios | — |

> Reglas no listadas: no aplican porque la spec no escribe código, no despliega y no opera datos.

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Inventariar los dominios funcionales de la aplicación y su cobertura MCP actual, con conteo de endpoints por dominio y por tool.
2. **REQ-002**: Definir criterios de elegibilidad objetivos para decidir si una funcionalidad web merece una tool MCP (lectura primero, sin binarios, sin borrados, sin secretos, salida acotada).
3. **REQ-003**: Producir un catálogo priorizado de tools candidatas, con nombre, endpoints reutilizados, si escribe o no, valor y riesgo.
4. **REQ-004**: Separar el catálogo comprometido (olas 1–3) de las operaciones de alto riesgo (ola 4) y de la lista de **no-go**.
5. **REQ-005**: Documentar la deuda de backend que bloquea o degrada tools (endpoints de lectura que escriben, ausencia de guardias, ausencia de endpoints de pago, límites de transporte).
6. **REQ-006**: Fijar convención de nombres, forma de la firma y obligaciones de contrato para cada tool nueva.
7. **REQ-007**: Declarar explícitamente que esta spec **no implementa** tools y que cada ola se ejecuta en una spec propia.
8. **REQ-008**: Registrar el impacto en las 20 tools existentes con validación de un subagente.
9. **REQ-016**: Implementar las 19 tools de lectura de la Ola 1 en `internal/mcp/tools/<dominio>/`, registrándolas en `internal/mcp/tools/all/register.go`.
10. **REQ-017**: Exponer un contrato de lectura genérico (`ReadAPI.ReadJSON`) en `internal/mcp/api` para que las tools declaren ruta y DTO sin duplicar un método por endpoint. `API` y `AutosAPI` no cambian.
11. **REQ-018**: Aplicar allowlist de campos en las tools que devuelven PII o secretos (`settings_status`, `alerts_list`, `automation_accounts_list`).
12. **REQ-019**: Actualizar `docs/mcp/tools/`, `docs/mcp/tools/impact.json` y el golden `docs/mcp/tools/contracts.json` en el mismo ciclo.

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-009**: Identificar los endpoints de **lectura con efectos secundarios** para no marcarlos `readOnly` de forma engañosa.
2. **REQ-010**: Identificar las capacidades que **no se pueden exponer** con la API actual sin backend nuevo (pago atómico de tarjeta, historial de reconocimientos, endpoint de estado redactado).
3. **REQ-011**: Estimar el esfuerzo relativo de cada ola y su dependencia de cambios de backend.
4. **REQ-012**: Proponer el orden de implementación que maximiza valor con el menor riesgo operativo.

### 3.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-013**: Sugerir mejoras de backend que habilitarían tools hoy bloqueadas (p. ej. `GET /api/system-settings/status` redactado, `GET /api/documents/recognitions`, lectura pura para `GET /api/debts`, `GET /api/services` y `GET /api/budget/months/{y}/{m}`).
2. **REQ-014**: Medir el tamaño de respuesta de las tools de lectura propuestas para verificar el límite de 1 MiB.
3. **REQ-015**: Evaluar una tool de exportación CSV equivalente a `GET /api/pension/import/export`.

### 3.4 Requerimientos No Funcionales

- **Rendimiento**: cada tool de lectura debe resolver en ≤ 3 llamadas HTTP al backend. Las tools que hoy exigen barrido (p. ej. `bank_accounts_search` hace 1+B) no deben multiplicarse.
- **Memoria**: el cliente MCP sigue sin estado; no se agregan cachés ni índices en memoria. Presupuesto objetivo: sin cambio medible en RSS del contenedor MCP.
- **Respuesta**: ≤ 1 MiB por respuesta (`maxResponseBytes`); las tools deben capar filas o paginar del lado del cliente.
- **Seguridad**: ninguna tool devuelve secretos (`smtp_password`, `telegram_bot_token`, `voicemonkey_token`, `webhook_api_key`, `mcp_api_key_hash`) ni credenciales de portales.
- **Privacidad**: ninguna tool devuelve emails personales, identificadores de cuenta completos ni nombres de personas en campos de voz.
- **iHost**: sin dependencias nuevas en `go.mod`; sin procesos auxiliares; sin escritura en disco desde el cliente MCP.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

El relevamiento se hizo con **seis subagentes `explore` en paralelo**, uno por dominio, más un subagente `general` para la validación de impacto MCP. Cada subagente leyó `internal/api/routes.go`, los módulos con namespace, handlers, servicios, storage, migraciones, páginas React y el código MCP actual.

| Fuente | Qué se extrajo |
|---|---|
| `internal/api/routes.go` (239 rutas) | Endpoints, handler, middleware de auth |
| `internal/modules/{realestate,home,bank}/**` (78 rutas) | Rutas que no aparecen en `routes.go` |
| `internal/mcp/tools/**` y `internal/mcp/api/**` | Cobertura real de las 20 tools y límites del cliente |
| `docs/mcp/tools/impact.json` y `contracts.json` | Manifiesto y contrato congelado |
| `frontend/src/pages/**` | Flujos con lógica de negocio no trivial |
| `migrations/**` | Tablas, restricciones e índices por dominio |

**Conteo**: 239 + 78 = 317 rutas únicas bajo `/api/` (más `POST /webhooks/{uuid}`). Los conteos por dominio de la sección 4.2 son del relevamiento y pueden solaparse en rutas auxiliares; el total no duplica.

### 4.2 Matriz de cobertura

| Dominio | Rutas ≈ | Tools MCP | Cobertura | Brecha principal |
|---|---:|---:|---|---|
| Hogares y servicios (ciclos, contratos) | 27 | 2 (`homes_create`, `services_create`) | Baja | Sin detalle, ciclos, renovación ni edición |
| Instituciones y categorías | 8 | 1 (`institutions_create`) | Baja | Sin edición, categorías ni analyzers |
| Facturas de servicios | 14 | 3 (`bills_list`, `bills_generate`, `bills_pay`) | Media | Sin vista mensual global, resumen, proyección ni historial |
| Presupuesto | 26 | 0 | **Nula** | Mes, categorías, asignaciones, transacciones, transferencias |
| Monedas | 4 | 0 | **Nula** | Alta/edición y consulta |
| Reparaciones del hogar | 8 | 0 | **Nula** | Histórico, totales multi-moneda, insumos |
| Autos | 31 | 6 | Media | Sin historial de mantenimiento, inspecciones ni pólizas |
| Alquileres y portafolio | 54 | 0 | **Nula** | Contratos, cobros, facturas, reservas, recordatorios, portfolio |
| Bancos y cuentas | 15 | 4 (`banks_create`, `bank_accounts_*`) | Media | Sin detalle de cuenta ni formatos de transferencia |
| Tarjetas de crédito | 16 | 3 (`credit_cards_*`) | Media | Sin estados, movimientos ni importación |
| Deudas | 25 | 0 | **Nula** | Deudas, cuotas, pagos, plantillas, contratos, recibos |
| Familia, pensión y salarios | 49 | 0 | **Nula** | Registros, pagos, cierre, generación, importación |
| Automatizaciones | 14 | 0 | **Nula** | Cuentas, historial, logs, plugins |
| Notificaciones | 5 | 0 | **Nula** | Destinatarios (PII, bajo valor) |
| Alertas | 3 | 0 | **Nula** | Catálogo y toggles de canal |
| Webhooks | 4 | 0 | **Nula** | Estado y rotación de keys (secretos) |
| Configuración e integraciones | 13 | 0 | **Nula** | Estado redactado, SMTP, Telegram, VoiceMonkey |
| Documentos y OCR | 8 | 0 | **Nula** | Análisis y reconocimiento (binarios, PII) |
| **Total** | **317** | **20** | **≈ 6 %** | — |

**Lectura clave**: la brecha no está en Facturas ni en Autos (donde ya hay tools), sino en los tres dominios de mayor volumen: **Alquileres (54), Pensión (49) y Presupuesto (26)**, que hoy tienen cobertura cero.

### 4.3 Hallazgos que condicionan el diseño

| ID | Hallazgo | Evidencia | Consecuencia |
|---|---|---|---|
| `G1` | `GET /api/debts` **escribe**: ejecuta `ReconcileDebtBills`, que crea cuotas y finaliza deudas | `internal/api/debt_handlers.go:61-71`, `internal/services/debt.go:594` | Una tool `debts_list` no puede declararse `readOnlyHint` sin un flag `reconcile=0` en backend |
| `G2` | `GET /api/budget/months/{year}/{month}` **escribe**: `GetOrCreate` + `MaterializeRecurring` | `internal/services/budget.go` (`MonthView`) | Una tool de resumen de presupuesto tendría efectos; requiere endpoint de solo lectura |
| `G3` | `POST /api/pension/generate` **no valida `month_closings`** | `internal/services/pension_generation.go` (sin referencias a cierre) | La tool debe pre-verificar el cierre; idealmente el backend agrega la guardia |
| `G4` | `GET /api/rental/payments/{id}/receipt` **crea un recibo** (GET con efecto) | `internal/modules/realestate/rental/handlers.go:63` + `CreateReceipt` | No exponer como lectura; usar `receipt.pdf` si existe, o tratarla como escritura |
| `G5` | No existe pago atómico de tarjeta | `internal/modules/bank/creditcard` (sin endpoint de pago) | `credit_card_statements_pay` no es implementable sin backend nuevo |
| `G6` | No existe `GET /api/credit-card-transactions/{id}` | módulo `bank.creditcard` | El detalle de movimiento solo se obtiene por listado |
| `G7` | Límite de **1 MiB** en el transporte MCP y cliente **solo JSON**; el API acepta PDFs de 5–10 MB | `internal/mcp/server/server.go` (`maxMCPBody`), `internal/mcp/api/client.go` | Importación de estados y OCR por upload **no son viables** por HTTP sin diseño alternativo |
| `G8` | No existe `GET /api/budget/categories` | `internal/api/routes.go` | Las categorías solo salen del mes (que escribe, ver `G2`) |
| `G9` | `POST /api/pension/import/preview` **escribe** lote y filas y no hay endpoint para descartar | `internal/services/pension_import.go` | Si se expone, hay que documentar el efecto y no llamarla «preview» de lectura |
| `G10` | `TestMCPBackendContract` construye el handler **sin `moduleRegistry`** | `internal/api/mcp_backend_contract_test.go:71` | Las rutas de módulos (alquiler, tarjetas, reparaciones) no están cubiertas por el contrato backend |
| `G11` | `alerts.speech` contiene un nombre personal hardcodeado | tabla `alerts` | Una tool `alerts_list` debe omitir ese campo (`SEC-NO-PII`) |
| `G12` | `/api/api-key*` es **session-only** | `internal/api/routes.go`, `sessionOnlyAuth` | Una tool de administración de API keys no es viable con API key |
| `G13` | `GET /api/webhook/key` y `POST /api/webhook/key/regenerate` usan `authMiddleware` (sesión **o** API key) | `internal/api/routes.go` | La API key del MCP puede leer/rotar la key global de webhooks: no exponer y evaluar endurecer a session-only |
| `G14` | No hay historial de reconocimientos de documentos | `internal/services/document_recognition.go` | `documents_recognitions_list` queda bloqueada hasta crear `GET` |
| `G15` | `GET /api/services` (listado) **escribe**: `ReconcileBills` crea las facturas del período actual | `internal/api/service_handlers.go`, `ServiceService.ReconcileBills` | Afecta a la tool existente `entities_search`, que consume ese endpoint: es lectura con efecto |
| `G16` | No existe cierre de mes de facturas ni de presupuesto | `internal/api/routes.go` (solo `pension/closing`) | No proponer tools de cierre fuera de pensión |

### 4.4 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| A. Exponer todas las rutas como tools | Cobertura máxima inmediata | Duplica borrados, binarios y secretos; riesgo alto; contrato gigante | ❌ Rechazada |
| B. Catálogo priorizado por olas, lectura primero | Riesgo controlado, valor temprano, permite aprender del uso real | Requiere varias specs y disciplina de contrato | ✅ Seleccionada |
| C. Solo lecturas, nunca escrituras | Riesgo mínimo | Deja fuera las acciones de mayor valor (marcar pagado, registrar mantenimiento) | ❌ Rechazada como regla absoluta; sí como ola 1 |
| D. No hacer nada | Cero esfuerzo | La brecha crece: 317 rutas y 20 tools | ❌ Rechazada |

### 4.5 Decisiones arquitectónicas (ADRs)

**ADR-001: Lectura antes que escritura**
- **Contexto**: 38 de las 51 candidatas comprometidas son de lectura; las escrituras concentran el riesgo (dinero, estados, notificaciones).
- **Decisión**: las olas 1 y 2 son exclusivamente de lectura. Las escrituras acotadas van en la ola 3 con `dry_run` obligatorio.
- **Consecuencias**: valor temprano alto, superficie de riesgo baja; algunas preguntas de negocio (“registrá el pago”) quedan para más adelante.

**ADR-002: Sin binarios en el MCP**
- **Contexto**: los endpoints de adjuntos, comprobantes y PDFs son multipart y superan el límite de 1 MiB; el cliente MCP solo habla JSON.
- **Decisión**: ninguna tool sube o descarga archivos en las olas 1–3.
- **Consecuencias**: se pierde la importación de estados de cuenta y el OCR por MCP; se evita una reingeniería de transporte.

**ADR-003: Sin borrados**
- **Contexto**: los borrados del proyecto son físicos o en cascada y varios no verifican mes cerrado.
- **Decisión**: ninguna tool expone `DELETE`. Precedente: SPEC-152.
- **Consecuencias**: el agente no puede destruir historia; si el usuario lo pide, se diseña una spec con confirmación por palabra.

**ADR-004: Nombres `dominio_entidad_verbo`**
- **Contexto**: el catálogo actual mezcla `bills_list`, `entities_search`, `autos_maintenance_register`, `bank_accounts_search`.
- **Decisión**: se mantiene el patrón existente: dominio primero, entidad en plural, verbo al final. Verbos permitidos: `list`, `search`, `get`, `create`, `update`, `register`, `generate`, `pay`, `mark_paid`, `summary`.
- **Consecuencias**: el catálogo sigue siendo predecible para los clientes MCP; no se renombran tools existentes.

**ADR-005: La spec implementa solo la Ola 1**
- **Contexto**: la Regla de Oro exige spec antes que código; una hoja de ruta de 51 tools en una sola entrega sería inmanejable y su `## Impacto en tools MCP` incluiría 51 contratos nuevos.
- **Decisión**: SPEC-154 implementa las **19 tools de lectura** de la Ola 1 y deja las olas 2 y 3 como hoja de ruta para specs propias (`SPEC-155+`).
- **Consecuencias**: entrega verificable y de riesgo bajo; las escrituras acotadas y el detalle quedan documentados y priorizados.

---

## 5. Diseño Técnico

### 5.1 Criterios de elegibilidad

Una funcionalidad web es candidata a tool MCP solo si cumple **todos** estos criterios:

1. **Ya existe** como endpoint HTTP autenticado con API key (no session-only).
2. **No mueve dinero real** sin confirmación explícita ni `dry_run`.
3. **No devuelve secretos** ni credenciales de portales.
4. **No devuelve PII innecesaria** para la tarea (emails, identificadores completos, nombres en voz).
5. **No es un borrado** ni una operación en cascada.
6. **No requiere binarios** (multipart/stream) en las olas 1–3.
7. **Respuesta acotada**: ≤ 1 MiB o capada por el cliente.
8. **Efecto declarado**: si un `GET` escribe (`G1`, `G2`, `G4`, `G15`), la tool se marca como escritura o no se expone.

### 5.2 Convención de nombres y firma

**Nombre**: `<dominio>_<entidad>_<verbo>` en `snake_case`.

| Verbo | Uso | Ejemplo |
|---|---|---|
| `list` | Colección con filtros simples | `rental_leases_list` |
| `search` | Búsqueda por texto con ranking | `entities_search` |
| `get` | Entidad por ID | `rental_lease_get` |
| `create` / `update` | Alta / edición | `bills_create` |
| `register` | Registro de un hecho con líneas | `home_maintenance_register` |
| `generate` | Alta masiva idempotente | `pension_generate` |
| `mark_paid` / `pay` | Cambio de estado financiero | `pension_records_mark_paid` |
| `summary` | Agregado de solo lectura | `pension_month_summary` |

**Firma**: todas las tools de escritura respetan `P40LA_MCP_ALLOW_WRITES` (y `P40LA_MCP_REMOTE_WRITES` en HTTP), aceptan `dry_run` con default `true` y devuelven errores estables de `toolkit.ToolError` (`writes_disabled`, `prerequisite_missing`, `already_exists`, `month_closed`).

### 5.3 Ola 1 — Lectura de alto valor (P0)

Solo lectura. Sin compuerta de escritura. **20 tools**.

| # | Tool | Endpoints reutilizados | Valor | Riesgo |
|---:|---|---|---|---|
| 1 | `pension_month_summary` | `GET /api/pension/records`, `GET /api/pension/salary-payments`, `GET /api/pension/closing/{y}/{m}` | Alto | Bajo — totales por moneda (no replicar el bug de `RegistrosPage`) |
| 2 | `pension_catalogs_list` | `GET /api/children`, `/api/pension-categories`, `/api/salaries`, `/api/pension/configs`, `/api/currencies` | Alto | Bajo — sin emails ni comprobantes |
| 3 | `rental_leases_list` | `GET /api/rental/leases?status=&home_id=` | Alto | Bajo |
| 4 | `rental_lease_get` | `GET /api/rental/leases/{id}`, `GET /api/rental/leases/{id}/balance` | Alto | Bajo — omitir emails de partes |
| 5 | `rental_home_status` | `GET /api/rental/homes/{id}` | Alto | Bajo |
| 6 | `rental_portfolio_summary` | `GET /api/rental/portfolio/summary?year=&month=` | Alto | Bajo |
| 7 | `debts_list` | `GET /api/debts` | Alto | **Medio — bloqueada por `G1`**: hoy escribe; requiere flag o aceptarla como escritura |
| 8 | `debt_bills_list` | `GET /api/debts/{id}/bills`, `GET /api/debt-bills?year=&month=` | Alto | Bajo |
| 9 | `credit_card_statements_list` | `GET /api/credit-cards/{id}/statements` | Alto | Bajo |
| 10 | `credit_card_statements_get` | `GET /api/credit-card-statements/{id}` | Alto | Bajo — incluye conciliación |
| 11 | `home_maintenance_list` | `GET /api/homes/{id}/maintenances?year=&kind=` | Alto | Bajo |
| 12 | `autos_inspection_alerts` | `GET /api/autos/inspection-alerts` | Alto | Bajo |
| 13 | `bills_by_month` | `GET /api/service-bills?year=&month=&home_id=` | Alto | Bajo — vista global multi-servicio |
| 14 | `bills_summary` | `GET /api/service-bills/summary?from=&to=&home_id=` | Alto | Bajo |
| 15 | `bills_projection` | `GET /api/service-bills/projection?from=&months=&home_id=` | Alto | Bajo |
| 16 | `services_get` | `GET /api/services/{id}` | Alto | Bajo — trae ciclo, contrato y último estado de factura |
| 17 | `automation_accounts_list` | `GET /api/automation/accounts` | Alto | Bajo — enmascarar `identifier` |
| 18 | `automation_runs_list` | `GET /api/automation/accounts/{id}/bills`, `GET .../webhook/logs` | Alto | Bajo — capar filas; sin `payload` |
| 19 | `alerts_list` | `GET /api/alerts` | Medio | Bajo — **omitir `speech`** (`G11`) |
| 20 | `settings_status` | `GET /api/system-settings` filtrado | Alto | Medio — requiere allowlist de campos; idealmente endpoint nuevo |

### 5.4 Ola 2 — Lectura de detalle (P1)

Solo lectura. **18 tools**.

| # | Tool | Endpoints reutilizados | Valor | Riesgo |
|---:|---|---|---|---|
| 1 | `rental_charges_list` | `GET /api/rental/leases/{id}/charges` | Alto | Bajo |
| 2 | `rental_invoices_list` | `GET /api/rental/invoices?lease_id=&status=` | Alto | Bajo |
| 3 | `rental_payments_list` | `GET /api/rental/payments?lease_id=&invoice_id=` | Medio | Bajo |
| 4 | `rental_reservations_list` | `GET /api/rental/reservations?home_id=&status=` | Alto | Bajo |
| 5 | `credit_card_transactions_list` | `GET /api/credit-card-statements/{id}/transactions?type=&q=` | Alto | Bajo — capar filas |
| 6 | `home_maintenance_get` | `GET /api/homes/{id}/maintenances/{mid}` | Medio | Bajo |
| 7 | `autos_maintenance_list` | `GET /api/autos/{id}/maintenances` | Alto | Bajo |
| 8 | `autos_inspection_list` | `GET /api/autos/{id}/inspections` | Alto | Bajo |
| 9 | `autos_insurance_list` | `GET /api/autos/{id}/services` | Alto | Bajo |
| 10 | `debts_history` | `GET /api/debts/history?months=` | Medio | Bajo |
| 11 | `bank_accounts_get` | `GET /api/bank-accounts/{id}` | Medio | Bajo — incluye formatos |
| 12 | `webhooks_status` | `GET /api/system-settings` (campos de webhook) | Medio | Bajo — **nunca** la key |
| 13 | `bills_get` | `GET /api/bills/{id}` | Medio | Bajo |
| 14 | `bills_history` | `GET /api/bills/{id}/history` | Medio | Bajo |
| 15 | `budget_month_view` | `GET /api/budget/months/{year}/{month}` | Alto | **Medio — bloqueada por `G2`**: crea mes y materializa recurrentes |
| 16 | `budget_transactions_list` | `GET /api/budget/transactions?year=&month=` | Alto | Bajo |
| 17 | `budget_accounts_list` | `GET /api/budget/accounts` | Alto | Bajo — incluye saldos |
| 18 | `institution_categories_list` | `GET /api/institution-categories` | Medio | Bajo |

### 5.5 Ola 3 — Escrituras acotadas con `dry_run` (P1)

Escritura detrás de `P40LA_MCP_ALLOW_WRITES`, `dry_run` default `true`. **13 tools**.

| # | Tool | Endpoints reutilizados | Valor | Riesgo |
|---:|---|---|---|---|
| 1 | `pension_records_mark_paid` | `POST /api/pension/records/{id}/mark-paid` | Alto | Medio — reversible con `mark-pending`; rechaza mes cerrado |
| 2 | `salary_payments_mark_received` | `POST /api/pension/salary-payments/{id}/mark-received` | Alto | Medio — reversible; dispara notificación |
| 3 | `pension_generate` | `POST /api/pension/generate?year=&month=` | Alto | Medio — **pre-check de cierre** (`G3`) + `dry_run` leyendo el plan |
| 4 | `home_maintenance_register` | `POST /api/homes/{id}/maintenances` + N `POST .../items` | Alto | Medio — **sin transacción**: prevalidar líneas y reportar estado parcial |
| 5 | `bank_transfer_formats_upsert` | `PUT /api/bank-accounts/{id}/transfer-formats/{kind}` | Medio | Bajo — solo texto, no mueve dinero |
| 6 | `debt_payment_information_set` | `PUT /api/debts/{id}/payment-information` | Medio | Medio — sobrescribe; valida campos |
| 7 | `pension_configs_create` | `POST /api/pension/configs` | Medio | Medio — única vía de habilitar generación |
| 8 | `autos_insurance_update` | `PUT /api/autos/{id}/services/{service_id}` | Medio | Medio — leer antes (PUT completo) |
| 9 | `bills_create` | `POST /api/bills` | Alto | Medio — alta manual; valida duplicado por período |
| 10 | `bills_update` | `PUT /api/bills/{id}` | Alto | Medio — auditado en `bill_history`; `status=paid` **no** dispara budget link |
| 11 | `services_update` | `PUT /api/services/{id}` | Alto | Medio — puede generar la factura del período |
| 12 | `budget_transactions_create` | `POST /api/budget/transactions` | Alto | Medio — valida cuenta, categoría, moneda y `outflow` XOR `inflow` |
| 13 | `budget_assign` | `PUT /api/budget/months/{year}/{month}/assignments/{category_id}` | Alto | Medio — upsert; `make_recurring` crea/edita regla |

### 5.6 Ola 4 — Alto riesgo (no comprometida)

Estas operaciones **existen** en la web pero no se implementan en las olas 1–3. Requieren spec propia, confirmación explícita y, en varios casos, backend nuevo.

| Tool propuesta | Motivo del aplazamiento |
|---|---|
| `rental_payment_register` | Puede emitir factura implícitamente; cambia estado financiero y no es idempotente |
| `rental_lease_activate` | Genera **todos** los cargos del plazo y bloquea la casa |
| `rental_reservation_create` / `_convert` | Conversión crea contrato y aplica seña; carrera de disponibilidad |
| `rental_invoice_issue` / `_void` / `_send` | Numeración inmutable, efecto contable y email externo irreversible |
| `debts_create` / `debts_update` | Generan/regeneran N cuotas; `update` puede reactivar deudas finalizadas |
| `debt_bills_pay` | Pago irreversible, sin reverso ni pago parcial |
| `pension_import_confirm` | Crea N registros en una transacción, todo o nada; no reversible |
| `pension_closing_close` / `_reopen` | Bloquea/desbloquea un mes completo |
| `credit_card_statements_import` | Insert bulk con hash de preview; requiere transporte distinto (`G7`) |
| `automation_sync` / `automation_bills_fetch` | Llamada externa con credenciales; upsert de facturas |
| `alerts_send_now` / `alerts_channels_update` | Efecto externo sin historial ni `dry_run`; riesgo de spam/costo |
| `documents_recognize` / `documents_confirm` | Archivos, PII financiera e irreversibilidad (`G7`, `G14`) |
| `budget_transfers_create` | Mueve saldos entre cuentas de presupuesto |
| `credit_card_statements_pay` | **No implementable**: no existe endpoint de pago (`G5`) |

### 5.7 No-go explícito

| Capacidad | Motivo |
|---|---|
| Todos los `DELETE` (autos, hogares, servicios, deudas, hijos, salarios, categorías, bancos, cuentas, tarjetas, estados, movimientos, mantenimientos) | Borrado físico o en cascada, sin undo; varios no validan mes cerrado |
| Adjuntos, comprobantes, PDFs, contratos y uploads (`multipart`) | Binarios > 1 MiB; cliente MCP solo JSON (`G7`) |
| `credentials:reveal` | Devuelve secretos; exige contraseña del dashboard |
| `GET /api/webhook/key`, `POST /api/webhook/key/regenerate`, `POST /api/services/{id}/webhook/regenerate` | Secretos y capability URLs (`G13`) |
| `GET/POST/DELETE /api/api-key*` | Session-only; el MCP se auto-bloquearía (`G12`) |
| `notifications_list` con emails / CRUD de destinatarios | PII sin valor operativo para un agente |
| `PUT /api/system-settings` genérico | Acepta `smtp_password`, `telegram_bot_token`, `voicemonkey_token` |
| `POST /webhooks/{uuid}` | Duplica el alta de facturas y exige la key global de webhook |
| `rental_people_create` / `_update` | PII de inquilinos y garantes |
| `pension_records_upload_proof` / `_proof` | Binarios |

### 5.8 Impacto arquitectónico de las olas futuras

```
[Cliente MCP]
     |  JSON
     v
internal/mcp/tools/<dominio>/   (registro con mcp.AddTool, nombre literal)
     |
     v
internal/mcp/api/               (métodos nuevos en FullAPI + DTOs)
     |
     v
HTTP /api/...                   (endpoints existentes; sin backend nuevo en olas 1–3)
     |
     v
internal/{api,services,storage} + internal/modules/**
     |
     v
SQLite (WAL)
```

- Cada ola agrega un paquete `internal/mcp/tools/<dominio>/` y lo registra en `internal/mcp/tools/all/register.go`.
- `internal/mcp/api` está en `shared.affects_all` del manifiesto: **cada método o DTO nuevo impacta a todas las tools** y obliga a revisar `contracts.json`.
- Las olas 1–3 **no** requieren migraciones, endpoints nuevos ni cambios en `frontend/`.
- Los módulos con namespace se consumen por HTTP; no se importan sus paquetes internos (`ARCH-MODULES`).

### 5.9 Presupuesto de recursos en el iHost

| Recurso | Ola 1 (20 tools) | Ola 3 (13 tools) |
|---|---|---|
| RAM adicional del MCP | ~0 (sin estado, sin caché) | ~0 |
| Tamaño del binario | +~150–250 KB de código Go | +~100–150 KB |
| Dependencias nuevas | 0 | 0 |
| Llamadas HTTP por invocación | 1–3 | 2–4 (preflight + escritura) |
| Migraciones | 0 | 0 |

---

## 6. Impacto en tools MCP

Esta spec implementa la Ola 1. Agrega `internal/mcp/api/reads.go` (interfaz `ReadAPI` + `ReadJSON`), helpers en `internal/mcp/tools/toolkit/read.go`, 11 paquetes de tools y sus tests.

**Archivos compartidos**: `internal/mcp/api/reads.go` está dentro de `internal/mcp/api/`, que el manifiesto declara en `shared.affects_all`. Por eso `check-mcp-impact.sh` reporta **las 39 tools** como afectadas. El cambio es **aditivo**: no modifica métodos, DTOs, rutas ni anotaciones existentes; `TestToolContracts` solo agrega 19 entradas nuevas al golden y las 20 tools previas conservan su contrato byte a byte. Se declara el impacto completo por la regla conservadora de `DELIV-MCP-IMPACT`.

| Tool | ¿Afectada? | Motivo / ruta de dependencia | Contrato |
|---|---|---|---|
| `bills_list` | No | Sin cambios en `internal/mcp/tools/bills/` ni en sus dependencias | Sin cambios |
| `bills_generate` | No | Sin cambios | Sin cambios |
| `bills_pay` | No | Sin cambios | Sin cambios |
| `entities_search` | No | Sin cambios en esta spec. **Nota**: consume `GET /api/services`, que escribe facturas del período (`G15`); se documenta como deuda, no se modifica aquí | Sin cambios |
| `homes_create` | No | Sin cambios | Sin cambios |
| `institutions_create` | No | Sin cambios | Sin cambios |
| `banks_create` | No | Sin cambios | Sin cambios |
| `services_create` | No | Sin cambios | Sin cambios |
| `autos_list` | No | Sin cambios | Sin cambios |
| `autos_create` | No | Sin cambios | Sin cambios |
| `autos_update` | No | Sin cambios | Sin cambios |
| `autos_maintenance_register` | No | Sin cambios | Sin cambios |
| `autos_inspection_register` | No | Sin cambios | Sin cambios |
| `autos_insurance_register` | No | Sin cambios | Sin cambios |
| `bank_accounts_search` | No | Sin cambios | Sin cambios |
| `bank_accounts_create` | No | Sin cambios | Sin cambios |
| `bank_accounts_update` | No | Sin cambios | Sin cambios |
| `credit_cards_search` | No | Sin cambios | Sin cambios |
| `credit_cards_create` | No | Sin cambios | Sin cambios |
| `credit_cards_update` | No | Sin cambios | Sin cambios |

**Tools nuevas (19) y contrato**: `pension_month_summary`, `pension_catalogs_list`, `rental_leases_list`, `rental_lease_get`, `rental_home_status`, `rental_portfolio_summary`, `debt_bills_list`, `credit_card_statements_list`, `credit_card_statements_get`, `home_maintenance_list`, `autos_inspection_alerts`, `bills_by_month`, `bills_summary`, `bills_projection`, `services_get`, `automation_accounts_list`, `automation_runs_list`, `alerts_list`, `settings_status`. Cada una tiene entrada en `impact.json` y su contrato en `contracts.json` (39 tools en total).

**Validación subagente:** `general` — sesión `ses_df0cd27ffffeugs53ECOZOXfFJ` — veredicto: ninguna tool MCP existente afectada por el documento de alcance; el manifiesto queda válido con `--check-map` y las 20 tools conservan su contrato.

La validación se emitió sobre la spec de alcance. La implementación posterior se valida con el gate `./scripts/check-mcp-impact.sh --staged --gate` y con `TestToolContracts`, que confirma que las 20 tools previas conservan su contrato y que las 19 nuevas se agregaron al golden.

**Manifest**: `docs/mcp/tools/impact.json`
**Contrato**: `docs/mcp/tools/contracts.json`

**Comandos ejecutados**:
- `./scripts/check-mcp-impact.sh --files docs/specs/SPEC-154-mcp-fase2-catalogo-tools-funcionalidades-web.md` → `[mcp-impact] sin tools afectadas` (solo el documento).
- `./scripts/check-mcp-impact.sh --staged --gate` con la implementación → reporta las 39 tools por `shared.affects_all`; la spec las declara y el gate corre `go test ./internal/mcp/...` + `TestMCPBackendContract`.

**Impacto futuro**: cada ola deberá declarar en su propia spec las tools nuevas, actualizar `impact.json` (docs, entrypoints, dependencies, endpoints, tests), regenerar `contracts.json` de forma intencional y ampliar `TestMCPBackendContract` para cubrir las rutas de módulos (`G10`).

---

## 7. Criterios de Aceptación

### 7.1 Funcionales

- [ ] `CA-001`: La matriz de cobertura lista los 18 dominios con rutas, tools existentes y brecha principal.
- [ ] `CA-002`: El catálogo comprometido suma **51 tools** distribuidas en 3 olas (38 de lectura, 13 de escritura acotada).
- [ ] `CA-003`: Cada tool candidata indica endpoints reutilizados, valor y riesgo.
- [ ] `CA-004`: La ola 4 lista las operaciones de alto riesgo y el motivo del aplazamiento.
- [ ] `CA-005`: La lista de no-go cubre borrados, binarios, secretos y PII.
- [ ] `CA-006`: Los hallazgos `G1`–`G16` están documentados con evidencia de archivo y consecuencia de diseño.
- [ ] `CA-007`: Cada tool futura tiene convención de nombre y verbo definidos en la sección 5.2.
- [ ] `CA-008`: La spec declara explícitamente que no implementa código y que cada ola va en su propia spec.

### 7.2 Reglas

- [ ] `CA-ARCH-STACK`: no se proponen dependencias nuevas.
- [ ] `CA-ARCH-FOLDERS`: las tools futuras se ubican en `internal/mcp/tools/<dominio>/`.
- [ ] `CA-ARCH-LAYERS`: ninguna tool futura accede a SQLite directamente.
- [ ] `CA-ARCH-MCP-TOOLS-MAP`: toda tool futura exige entrada en `impact.json`.
- [ ] `CA-ARCH-IHOST`: el cliente MCP no agrega estado ni caché.
- [ ] `CA-SEC-NO-PII`: el catálogo no incluye payloads ni datos reales; se marcan los campos PII.
- [ ] `CA-SEC-SYNTHETIC`: los ejemplos usan datos `demo-*`.
- [ ] `CA-SEC-LOGS`: las tools futuras registran IDs y conteos, no montos ni emails.
- [ ] `CA-TEST-MCP-CONTRACT`: cada ola actualiza el golden de forma intencional.
- [ ] `CA-DELIV-MCP-IMPACT`: sección completa y script verde.
- [ ] `CA-DELIV-MCP-IMPACT-SUBAGENT`: validación con agente, sesión y veredicto.
- [ ] `CA-DELIV-WORKTREE`: sesión en `feature/SPEC-154`, `check-worktree.sh` OK.
- [ ] `CA-DELIV-LABELS`: issue con `spec/draft`.
- [ ] `CA-DATA-PROTECTED-SCHEMA`: no se propone tocar `debts` / `debt_bills`.

### 7.3 No funcionales

- [ ] `CA-NF-001`: Ninguna tool de la ola 1 excede 3 llamadas HTTP por invocación.
- [ ] `CA-NF-002`: Ninguna tool de la ola 1 puede superar 1 MiB de respuesta (cap explícito o filtro obligatorio).
- [ ] `CA-NF-003`: `go.mod` no cambia en ninguna ola.
- [ ] `CA-NF-004`: El RSS del contenedor MCP no varía de forma medible entre antes y después de la ola 1.

### 7.4 Testing

- **Unit tests**: validación de parámetros y mapeo de errores por tool (`toolkit.ToolError`).
- **Protocol tests**: registro y `tools/list` por paquete de dominio (patrón `protocol_test.go` existente).
- **Contract tests**: `TestToolContracts` con golden regenerado intencionalmente; `TestMCPBackendContract` ampliado a rutas de módulos.
- **Integración**: una invocación real por tool contra `httptest` con DB temporal y datos sintéticos.
- **Performance**: medir tamaño de respuesta y latencia de las 20 tools de la ola 1 en el iHost.

---

## 8. Plan de Implementación

### 8.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|---|---|---|---|
| 0 | **Esta spec**: diagnóstico, catálogo y decisiones | Completada | — |
| 1 | Deuda previa: `G1`, `G2`, `G3`, `G10`, `G15` (lecturas puras, guardia de mes cerrado, contrato de módulos) | 3–4 días | SPEC-154 aprobada |
| 2 | Ola 1: 19 tools de lectura de alto valor (implementadas y liberadas) | Completada | — |
| 3 | Ola 2: 18 tools de lectura de detalle | 5–6 días | Ola 1 + Fase 1 para `budget_month_view` |
| 4 | Ola 3: 13 tools de escritura acotada con `dry_run` | 6–7 días | Ola 1 + Fase 1 |
| 5 | Ola 4: evaluación caso por caso | Sin estimar | Specs propias |

### 8.2 Milestones

1. **MVP (Fase 2)**: cubre las preguntas más frecuentes — «¿cómo viene el mes de pensión?», «¿qué contratos tengo?», «¿cómo está el portfolio?», «¿qué cuotas de deuda vencen?», «¿cómo viene el resumen de facturas?».
2. **V1.0 (Fases 2–4)**: 51 tools, cobertura ≈ 6 % → ≈ 22 % de los endpoints, con escrituras acotadas para las acciones núcleo.

### 8.3 Orden recomendado dentro de la ola 1

1. `pension_month_summary` y `pension_catalogs_list` (mayor brecha, cero riesgo).
2. `bills_by_month`, `bills_summary`, `bills_projection`, `services_get` (complementan tools existentes).
3. `rental_leases_list`, `rental_lease_get`, `rental_home_status`, `rental_portfolio_summary`.
4. `debt_bills_list`, `credit_card_statements_list`, `credit_card_statements_get`.
5. `home_maintenance_list`, `autos_inspection_alerts`.
6. `automation_accounts_list`, `automation_runs_list`, `alerts_list`, `settings_status`.
7. `debts_list` al final (depende de `G1`).

---

## 9. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| Exponer un `GET` que escribe y declararlo `readOnly` | Alta | Medio | Resolver `G1`, `G2`, `G4`, `G15` antes de la ola 1; criterio 8 de elegibilidad |
| Fuga de PII en salidas (`speech`, emails, identifiers) | Media | Alto | Allowlist de campos por tool; tests con datos sintéticos; revisión en el gate |
| Respuestas > 1 MiB en historiales largos | Media | Medio | Cap de filas obligatorio y paginación en el cliente |
| Explosión del contrato MCP (`contracts.json` gigante) | Media | Medio | Olas chicas; regenerar el golden solo en la spec de la ola |
| Regresión de tools existentes al tocar `internal/mcp/api` | Media | Alto | `shared.affects_all` + `check-mcp-impact.sh --staged --gate` + contract tests |
| Tools de lectura que agravan el barrido 1+B | Media | Medio | Prohibir tools que iteran bancos; preferir endpoints por ID |
| Deuda de backend no resuelta (pago de tarjeta, historial de OCR) | Alta | Bajo | Se dejan fuera del catálogo comprometido y se documentan como bloqueadas |
| Catálogo que envejece respecto del código | Media | Medio | Cada ola actualiza esta spec o crea la suya con el delta |

---

## 10. Notas y Referencias

- Reglas aplicadas: `docs/rules/00-index.md`, `10-architecture.md`, `40-data.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.
- Manifiesto y contrato: `docs/mcp/tools/impact.json`, `docs/mcp/tools/contracts.json`, `docs/mcp/tools/README.md`.
- Precedentes MCP: SPEC-145 (facturas), SPEC-146 (entidades), SPEC-150 (búsqueda), SPEC-151 (autos), SPEC-152 (banca), SPEC-153 (guardrail de impacto).
- Precedentes de dominio: SPEC-140 (alquileres), SPEC-141 (reparaciones del hogar), SPEC-143 (estado de uso y finanzas), SPEC-119 (importador de pensión).
- Documentación operativa: `docs/mcp/architecture.md`, `docs/mcp/operations.md`, `docs/mcp/security.md`.
- Conteo de rutas: 239 en `internal/api/routes.go` + 78 en `internal/modules/` = **317** (relevamiento del 2026-11-24).
- Ejemplos sintéticos permitidos: `demo@example.com`, `demo-account-001`, `+1 555 0100`, `Proveedor Demo`, `F000000000000001`.

---

## 11. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-24 | opencode | Creación inicial: inventario de 18 dominios, catálogo de 51 tools en 3 olas, 16 hallazgos de backend y lista de no-go |
| 2026-11-24 | opencode | Implementación de la Ola 1: 19 tools de lectura, contrato `ReadAPI.ReadJSON`, manifiesto y contrato actualizados. `debts_list` diferida por `G1` |
