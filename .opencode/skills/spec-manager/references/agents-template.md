# AGENTS.md — {{PROJECT_NAME}}

> **Última actualización**: {{DATE}}
> **Sistema de specs**: {{SPEC_SYSTEM_VERSION}}

Este archivo es la **entrada** para agentes. Las reglas completas viven en `docs/rules/`, una sola vez cada una, con IDs estables.

---

## ⚠️ Regla de Oro

**Antes de escribir código: la spec primero. Siempre.**
Si la spec no existe, no se escribe código. Si la spec no contempla el cambio, se actualiza primero. Sin excepciones.

**Cerrar una spec es SIEMPRE un release, jamás una cancelación.**
`cancelled` solo existe si el usuario lo ordena explícitamente con las palabras «cancelar» o «descartar». Ante duda, preguntar antes de cambiar el estado.

---

## 🚨 No negociables (aplicar siempre)

| ID | Regla |
|---|---|
| `SEC-NO-PII` | Prohibido commitear PII, datos financieros, secretos o volcados de producción. |
| `SEC-SYNTHETIC` | Todo ejemplo/test usa datos sintéticos. |
| `SEC-INCIDENT` | Secreto/PII commiteado = incidente: rotar, purgar historia, documentar, avisar. |
| `DATA-NO-DROP` | Nunca borrar ni reemplazar la base de datos del usuario. |
| `DATA-PRODCOPY` | Validar contra copia de producción, nunca contra la DB local. Nunca escribir en producción. |
| `DELIV-CONFIRM` | Merge y push requieren confirmación explícita del usuario. Prohibido `--force`. |

Ajustar esta tabla al proyecto: agregar solo reglas realmente no negociables.

Detalle: [`docs/rules/00-index.md`](docs/rules/00-index.md).

---

## 📚 Reglas por dominio

| Archivo | Alcance | Cargar cuando… |
|---|---|---|
| [`docs/rules/00-index.md`](docs/rules/00-index.md) | Manifiesto, precedencia, guardrails, IDs | **Siempre** |
| [`docs/rules/10-architecture.md`](docs/rules/10-architecture.md) | Stack, capas, carpetas, convenciones | Se diseña una solución |
| [`docs/rules/20-ui.md`](docs/rules/20-ui.md) | Componentes, tokens, patrones visuales | Toca frontend/UI |
| [`docs/rules/30-ux.md`](docs/rules/30-ux.md) | Navegación, modales, detalle, estados | Toca pantallas o flujos |
| [`docs/rules/40-data.md`](docs/rules/40-data.md) | Datos, migraciones, copias de producción | Toca DB o datos reales |
| [`docs/rules/50-security.md`](docs/rules/50-security.md) | PII, secretos, datos sintéticos | **Siempre** |
| [`docs/rules/60-testing.md`](docs/rules/60-testing.md) | Pruebas, verificación, flujo con el usuario | Antes de release |
| [`docs/rules/70-delivery.md`](docs/rules/70-delivery.md) | Git, release, deploy | Antes de commit/merge |

---

## 🔄 Sistema de specs (resumen)

```
draft → pending_execution → in_progress → pending_release → released
```

- IDs `SPEC-XXX` inmutables y secuenciales. El próximo ID se calcula desde `docs/specs/SPEC-*.md` + issues, **no** desde el README.
- Toda spec tiene un issue con label `spec/<estado>`.
- Antes de `pending_release`: checklist de `docs/rules/60-testing.md`.
- Antes de declarar terminada una spec: `/spec-check-release`.

Flujo completo: skill `spec-manager`.

---

## 🗂️ Estructura

```
/
├── AGENTS.md
├── docs/
│   ├── rules/          # fuente de verdad de las reglas
│   └── specs/          # specs + tracker
└── .opencode/skills/spec-manager/
```

Completar con la estructura real del proyecto.

---

*Las reglas del proyecto prevalecen sobre cualquier skill o template genérico.*
