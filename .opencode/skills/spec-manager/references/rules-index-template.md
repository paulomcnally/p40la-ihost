# 00 — Índice y manifiesto de reglas

> **Proyecto**: {{PROJECT_NAME}}
> **Última actualización**: {{DATE}}

Este archivo es el **punto de entrada** de las reglas del proyecto. La skill `spec-manager` lo lee primero (Paso 0) y desde acá decide qué dominios cargar.

---

## 1. Precedencia

1. Las **reglas de este repositorio** ganan sobre cualquier skill, template o instrucción genérica.
2. Si hay conflicto entre archivos de reglas, gana el archivo del dominio específico sobre este índice.
3. Si hay conflicto entre una spec y una regla, se actualiza la spec **antes** de escribir código.
4. Si una regla no existe y el cambio la necesita: se crea la regla (con ID) y después se implementa.

---

## 2. Perfil del proyecto

| Campo | Valor |
|---|---|
| Nombre | {{PROJECT_NAME}} |
| Tipo | {{PROJECT_TYPE}} |
| Runtime | {{RUNTIME}} |
| Lenguaje(s) | {{LANGUAGES}} |
| Base de datos | {{DATABASE}} |
| Frontend | {{FRONTEND}} |
| Tests | {{TEST_COMMAND}} |
| Deploy | {{DEPLOY}} |
| Specs | `docs/specs/` |

---

## 3. Dominios

| Archivo | Alcance | Cargar cuando… |
|---|---|---|
| [10-architecture.md](10-architecture.md) | Stack, capas, estructura, convenciones | Se diseña una solución técnica |
| [20-ui.md](20-ui.md) | Componentes, tokens, patrones visuales | La spec toca UI |
| [30-ux.md](30-ux.md) | Navegación, modales, detalle, estados, flujos | La spec toca pantallas o flujos |
| [40-data.md](40-data.md) | Datos, migraciones, copias de producción | La spec toca DB o datos reales |
| [50-security.md](50-security.md) | PII, secretos, datos sintéticos | **Siempre** |
| [60-testing.md](60-testing.md) | Pruebas, verificación, flujo con el usuario | Siempre antes de release |
| [70-delivery.md](70-delivery.md) | Git, release, deploy | Antes de commit/merge |

> Borrar las filas de dominios que el proyecto no use, o marcar el archivo como `N/A` con justificación.

---

## 4. Reglas no negociables (cargar siempre)

| ID | Regla | Fuente |
|---|---|---|
| `SEC-NO-PII` | Prohibido commitear PII, datos financieros, secretos o volcados | [50-security.md](50-security.md) |
| `SEC-SYNTHETIC` | Todo ejemplo/test usa datos sintéticos | [50-security.md](50-security.md) |
| `SEC-INCIDENT` | Secreto/PII commiteado = incidente: rotar, purgar, documentar | [50-security.md](50-security.md) |
| `DATA-NO-DROP` | Nunca borrar ni reemplazar la DB del usuario | [40-data.md](40-data.md) |
| `DATA-PRODCOPY` | Validar contra copia de producción, nunca escribir en producción | [40-data.md](40-data.md) |
| `DELIV-CONFIRM` | Merge y push requieren confirmación explícita | [70-delivery.md](70-delivery.md) |

---

## 5. Guardrails ejecutables

| Guardrail | Regla | Comando | Cuándo |
|---|---|---|---|
| `build` | `TEST-LOCAL` | `{{BUILD_COMMAND}}` | Siempre |
| `tests` | `TEST-LOCAL` | `{{TEST_COMMAND}}` | Siempre |
| `ui` | `UI-SELECT` | `{{UI_CHECK_COMMAND}}` | La spec toca UI |
| `diff` | `SEC-COMMIT` | `git diff --cached` (revisión manual) | Antes de commit |

> Si un comando no existe, eliminar la fila. No declarar guardrails que no se ejecutan.

---

## 6. IDs de regla

| Prefijo | Dominio |
|---|---|
| `ARCH-` | Arquitectura y stack |
| `UI-` | Interfaz y componentes |
| `UX-` | Experiencia, navegación y modales |
| `DATA-` | Datos y migraciones |
| `SEC-` | Seguridad y privacidad |
| `TEST-` | Pruebas y verificación |
| `DELIV-` | Entrega, git y deploy |

Formato: `PREFIJO-NOMBRE-CORTO` (`UI-SELECT`, `UX-MODAL`, `DATA-PRODCOPY`).
Criterios de aceptación: `CA-<MISMO-ID>` (`CA-UI-SELECT`).

---

## 7. Mantenimiento

- Una regla se escribe **una sola vez** en su archivo de dominio.
- `AGENTS.md` y la skill **referencian**, no copian.
- Al agregar una regla: asignar ID, agregarla al dominio y, si es verificable, declarar el guardrail acá.
- Al cambiar una regla: mantener el ID si es una aclaración; crear uno nuevo si cambia el significado.
