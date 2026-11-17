# 10 — Arquitectura y stack

> **Dominio**: stack, capas, estructura de carpetas y convenciones.
> **Cargar cuando**: se diseña cualquier solución técnica.

---

## ARCH-STACK — Stack obligatorio

| Capa | Tecnología | Motivo |
|---|---|---|
| {{CAPA}} | {{TECNOLOGIA}} | {{MOTIVO}} |

**Prohibido**: agregar dependencias sin justificarlas en la spec.

---

## ARCH-FOLDERS — Estructura de carpetas

```
{{FOLDER_TREE}}
```

Completar con la estructura real. Si una spec agrega carpetas nuevas, debe declararlo.

---

## ARCH-LAYERS — Capas obligatorias

1. **{{CAPA_1}}**: {{RESPONSABILIDAD}}
2. **{{CAPA_2}}**: {{RESPONSABILIDAD}}

**Prohibido**:
- Lógica de negocio en la capa de transporte.
- Acceso a datos fuera de la capa de storage.
- Dependencias cruzadas entre capas no declaradas.

---

## ARCH-CONV — Convenciones de código

- Formateo automático obligatorio (`{{FORMAT_COMMAND}}`).
- Nombres en inglés en el código; documentación/specs en el idioma del equipo.
- Manejo de errores explícito: nunca ignorar un error.
- Configuración por variables de entorno con defaults razonables.
- Logging estructurado.

---

## ARCH-CONFIG — Configuración de agentes

- `AGENTS.md` es la entrada auto-cargada.
- `opencode.json` declara `instructions` con `AGENTS.md` y `docs/rules/*.md`.
- Las reglas viven en `docs/rules/`; no se duplican en la skill.

**Guardrail**: `build` (regla `TEST-LOCAL`) → `{{BUILD_COMMAND}}`.
