---
title: "Skill spec-manager portable: contrato de reglas por dominio, bootstrap obligatorio e init"
id: "SPEC-136"
status: "released"
author: "opencode"
created: "2026-11-17"
updated: "2026-11-17"
github_issue: 139
---

# Skill spec-manager portable: contrato de reglas por dominio, bootstrap obligatorio e init

**ID**: SPEC-136
**Estado**: released
**Autor**: opencode
**Creado**: 2026-11-17
**Actualizado**: 2026-11-17

---

## 1. Resumen Ejecutivo

La skill `spec-manager` administra el ciclo de vida de las specs, pero **no garantiza que las reglas del repositorio se apliquen**. Una auditoría de 79 reglas obligatorias (AGENTS.md + docs/project-rules.md) mostró que la skill cubre el 18%, cubre parcialmente el 20% y omite el 62%. Además no existe ningún paso que lea las reglas antes de redactar una spec ni que advierta cuando esas reglas no existen.

La consecuencia se mide en el repositorio: las reglas que viven en el template (`darkmode`, `BACK_ROUTES`, `Select`, modales) tienen 90–100% de cumplimiento; las que solo viven en AGENTS.md (`app.edit` 43%, prerrequisitos backend+frontend 50%, patrón de listados 71%) fallan sistemáticamente. El sistema no previene: corrige después (SPEC-114→115, SPEC-122→126).

Además, el conocimiento está duplicado en tres archivos que se editan a mano en paralelo (AGENTS.md, docs/project-rules.md y SKILL.md), lo que produce drift: mientras se hacía esta auditoría otra sesión agregó dos reglas nuevas de UI en los tres archivos a la vez.

Esta spec rediseña la skill para que sea **portable a cualquier proyecto**, con un **contrato de estructura de archivos de reglas**, un **paso 0 de bootstrap obligatorio** que aborta si las reglas faltan, **IDs de regla citables** en las specs, un **comando `/spec init`** que genera la estructura en proyectos nuevos, y un **gate de release** que verifica cumplimiento. En este repositorio se migra el contenido a `docs/rules/` por dominio, dejando `AGENTS.md` como entrada y `docs/project-rules.md` como stub de compatibilidad.

## 2. Cumplimiento de reglas

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-CONFIG` | `opencode.json` declara `instructions` con `docs/rules/*.md` y comandos vigentes | CA-DELIV-003 |
| `UI-GUARDRAIL` | La skill declara qué cubre y qué no cubre el guardrail de UI | CA-NF-002 |
| `UX-PREREQ` | El Paso 0 valida prerequisitos (estructura de reglas) antes de operar | CA-UX-001 |
| `DATA-PRODCOPY` | Se migra la regla; no se tocan datos en esta implementación | N/A: sin cambios de datos |
| `SEC-SYNTHETIC` | Todos los ejemplos de los templates usan datos `demo-*` | CA-NF-003 |
| `SEC-NO-PII` | No se incluyen datos reales; auditoría describe conteos, no valores | CA-NF-003 |
| `TEST-LOCAL` | `init` se prueba en `/tmp` en tres escenarios | CA-UI-001, CA-UI-002 |
| `DELIV-WORKTREE` | El trabajo se hace en `feature/SPEC-136` | N/A en la spec: verificado en ejecución |
| `DELIV-CONFIRM` | No se mergea ni pushea sin confirmación del usuario | N/A: pendiente de confirmación |

Reglas aplicables al diseño de la skill; el detalle vive en `docs/rules/`.

---

## 3. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: La skill DEBE definir un contrato de estructura de reglas portable, independiente de este repositorio:
   ```
   AGENTS.md                        # entrada + índice (obligatorio)
   docs/rules/00-index.md           # manifiesto de reglas (obligatorio)
   docs/rules/10-architecture.md    # stack, capas, carpetas, DB
   docs/rules/20-ui.md              # componentes, tokens, prohibiciones
   docs/rules/30-ux.md              # navegación, modales, estados, formularios
   docs/rules/40-data.md            # datos, migraciones, copias de producción
   docs/rules/50-security.md        # PII, secretos, higiene de commits
   docs/rules/60-testing.md         # pruebas locales, verificación, criterios
   docs/rules/70-delivery.md        # git, worktrees, release, deploy
   ```
2. **REQ-002**: La skill DEBE incluir un **Paso 0 — Rules Bootstrap** obligatorio, previo a cualquier comando, que:
   - localice `AGENTS.md` y `docs/rules/00-index.md` (o la ruta declarada en el manifiesto);
   - lea el manifiesto y determine qué archivos de reglas existen y cuáles faltan;
   - **DETENGA** el flujo con advertencia explícita si falta la estructura, en lugar de operar sin reglas;
   - declare que, ante conflicto, **las reglas del proyecto ganan sobre la skill**.
3. **REQ-003**: La skill DEBE declarar que **no copia reglas**: solo las referencia. Una regla tiene una única fuente canónica.
4. **REQ-004**: La skill DEBE implementar `/spec init`, que:
   - detecta los archivos de estructura faltantes;
   - los genera desde los templates incluidos en la propia skill;
   - **nunca sobrescribe** archivos existentes (modo no destructivo, reporta «omitido»);
   - no toca código ni configuración del proyecto más allá de la estructura de reglas;
   - al terminar reporta un diff de lo creado y lo omitido.
5. **REQ-005**: La skill DEBE definir **IDs de regla estables** con prefijo por dominio (`ARCH-`, `UI-`, `UX-`, `DATA-`, `SEC-`, `TEST-`, `DELIV-`) y cada archivo de dominio DEBE listar sus reglas con ID, enunciado y guardrail (comando o verificación manual).
6. **REQ-006**: Toda spec generada DEBE incluir una sección **«Cumplimiento de reglas»** con una tabla `Regla | Cómo la cumple la spec | CA`, y solo puede declarar `N/A` con justificación.
7. **REQ-007**: Los criterios de aceptación DEBEN usar IDs canónicos por dominio (`CA-UI-*`, `CA-UX-*`, `CA-ARCH-*`, `CA-DATA-*`, `CA-SEC-*`, `CA-TEST-*`, `CA-DELIV-*`), de modo que el cumplimiento sea auditable por grep.
8. **REQ-008**: La skill DEBE incluir un **gate de pre-release** que: (a) ejecute los guardrails declarados en el manifiesto; (b) verifique que cada regla aplicable esté citada en la spec; (c) bloquee el pasaje a `pending_release`/`released` si falta alguna.
9. **REQ-009**: El ID de la próxima spec DEBE calcularse desde el sistema de archivos (`docs/specs/SPEC-*.md` + issues abiertos), nunca desde el README tracker, que se actualiza solo en release.
10. **REQ-010**: Los comandos documentados en la skill DEBEN coincidir con los comandos reales de OpenCode (`/spec-create`, `/spec-list`, `/spec-status`, `/spec-show`) más los que se agreguen (`/spec-init`, `/spec-check-release`), sin comandos fantasma.
11. **REQ-018**: El repositorio DEBE incluir **hooks de git** versionados en `.githooks/`:
    - `pre-commit`: bloquea archivos prohibidos (`*.db`, `*.db-wal`, `*.db-shm`, `*.sqlite*`, `*.pdf`, `*.dump`, `backups/`), PII/secretos en líneas agregadas, cambios directos en `public/i18n/`, y ejecuta el guardrail de UI cuando hay `.tsx` staged.
    - `commit-msg`: exige el formato `SPEC-XXX: descripción` (o prefijos permitidos: `Merge `, `Revert `, `bump `, `hotfix:`).
    - `scripts/install-hooks.sh`: instalación idempotente vía `git config core.hooksPath .githooks`.
12. **REQ-019**: El repositorio DEBE incluir un **plugin de OpenCode** (`.opencode/plugins/rules-guard/`) que, en `tool.execute.before`, bloquee:
    - comandos que borren o reemplacen `data/app.db` o el directorio `data/`;
    - `pkill` (debe usarse `killall`);
    - `git push --force`/`-f` y `git reset --hard`;
    - escrituras en `public/i18n/**` (debe editarse `frontend/public/i18n/**`);
    - `<select>` nativo y `<input type="checkbox">` en archivos `.tsx`/`.jsx` fuera de la allowlist.
13. **REQ-020**: El plugin DEBE inyectar una vez por sesión un bloque compacto con la ruta de `docs/rules/` y las prohibiciones críticas, para que las reglas estén presentes aun sin invocar la skill.
14. **REQ-021**: `scripts/check-worktree.sh` DEBE advertir (sin bloquear) si `core.hooksPath` no está configurado.
15. **REQ-022**: La skill DEBE incluir templates de hooks en `references/hooks/` y `init` DEBE generarlos en proyectos nuevos.
16. **REQ-023**: Todo hook y plugin DEBE ser **fail-safe**: si su propia lógica falla, no debe romper el commit ni la sesión; solo los patrones positivos bloquean.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-011**: En este repositorio, migrar el contenido de `AGENTS.md` y `docs/project-rules.md` a `docs/rules/` por dominio, dejando:
   - `AGENTS.md`: propósito, regla de oro, reglas de seguridad no negociables (PII/DB/worktrees) y **índice** de `docs/rules/`;
   - `docs/project-rules.md`: stub de compatibilidad de una pantalla que redirige a `docs/rules/`, para no romper las 132 specs que lo citan.
2. **REQ-012**: `opencode.json` DEBE declarar `instructions` con `AGENTS.md` y el glob `docs/rules/*.md`, y usar las claves vigentes (`commands`, `skills` como array), eliminando las formas deprecadas.
3. **REQ-013**: La skill DEBE incluir `references/` con: el manifiesto template, los 7 templates de dominio, el template de spec y el catálogo de reglas base.
4. **REQ-014**: El catálogo de reglas base DEBE ser genérico (no mencionar `Select.tsx`, `BACK_ROUTES` ni rutas de este repo); cada proyecto resuelve el componente o archivo concreto en su manifiesto.
5. **REQ-015**: `docs/specs/templates/spec-template.md` DEBE incorporar la sección «Cumplimiento de reglas» y los IDs canónicos de CA, reemplazando los criterios ad-hoc (`CA-DARK`, `CA-MODAL`, `CA-BACK`) por `CA-UI-*`/`CA-UX-*`.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-016**: La skill DEBE advertir si detecta que una regla está duplicada textualmente en más de un archivo de reglas (riesgo de drift) y sugerir dejar una sola copia.
2. **REQ-017**: `/spec init --migrate` DEBE poder importar un `AGENTS.md`/`project-rules.md` monolítico existente y proponer un mapeo a los archivos por dominio, sin aplicar cambios sin confirmación.

### 2.4 Requerimientos No Funcionales

- **Portabilidad**: la skill no debe contener rutas, componentes ni tecnologías de un proyecto específico, salvo como ejemplo marcado.
- **Compatibilidad**: funciona con OpenCode V2 (skills bajo demanda, `instructions` con globs).
- **Tamaño**: `SKILL.md` ≤ 500 líneas; el detalle vive en `references/` (progressive disclosure).
- **No destructivo**: `init` jamás sobrescribe ni borra.
- **Trazabilidad**: toda regla tiene ID y toda spec cita las reglas que le aplican.

## 4. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- Documentación oficial de OpenCode: skills (`.opencode/skills/<id>/SKILL.md`, carga bajo demanda, `description` obligatoria), instructions (`AGENTS.md` auto-descubierto + array `instructions` con globs), comportamiento de archivos faltantes (se saltean en silencio).
- Auditoría de cobertura: 79 reglas → 14 cubiertas, 16 parciales, 49 ausentes.
- Auditoría empírica de 29 specs recientes con UI: correlación directa entre «regla presente en el template» y cumplimiento.
- Guardrail actual: `frontend/scripts/check-ui-rules.cjs` cubre 1 de 9 reglas de UI, solo `.tsx`, solo en build.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Copiar las reglas dentro de `SKILL.md` | Todo en un archivo, sin dependencias | Drift garantizado, no portable, crece sin límite | ❌ Rechazada |
| Un monolito `docs/rules.md` por proyecto | Simple | No permite cargar solo lo relevante, dificulta IDs y ownership | ❌ Rechazada |
| Índice + archivos por dominio + IDs | Una fuente por regla, citable, portable, auditable | Requiere disciplina de mantenimiento | ✅ Seleccionada |
| Plugin que inyecte reglas | Automático | Código nuevo, dependencia, viola «cero código sin spec» | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Una regla = un ID + una fuente canónica + un guardrail.
- **Contexto**: hoy la misma regla de modales vive en 3 archivos con redacciones distintas.
- **Decisión**: cada regla se define una sola vez con ID estable; los demás documentos la referencian por ID.
- **Consecuencias**: las specs citan `UI-SELECT` en vez de copiar el párrafo; auditable con grep; se elimina el drift.

**ADR-002**: La skill apunta, no copia.
- **Contexto**: el contenido del repo cambia más rápido que la skill.
- **Decisión**: la skill contiene el *contrato* y los *templates*; el contenido vive en el proyecto.
- **Consecuencias**: la skill sirve para N proyectos; el proyecto puede evolucionar sus reglas sin tocar la skill.

**ADR-003**: Falla ruidosa ante reglas ausentes.
- **Contexto**: OpenCode ignora en silencio archivos faltantes.
- **Decisión**: el Paso 0 aborta y ofrece `/spec init`.
- **Consecuencias**: un proyecto nuevo no puede generar specs sin antes declarar sus reglas.

## 5. Diseño Técnico

### 4.1 Diagrama

```
Usuario → /spec-create ─────┐
                            ▼
                   Paso 0: Rules Bootstrap
                            │
              ┌─────────────┴─────────────┐
              │ falta estructura          │ existe
              ▼                           ▼
        ABORTAR + ofrecer          leer 00-index.md
        /spec init                 cargar dominios aplicables
                                          │
                                          ▼
                                  redactar spec
                                  (tabla Cumplimiento de reglas)
                                          │
                                          ▼
                                  pre-release gate
                                  (guardrails + cobertura)
```

### 4.2 Componentes

#### 4.2.1 `.opencode/skills/spec-manager/SKILL.md`
- **Responsabilidad**: contrato, flujo, comandos, gates.
- **Dependencias**: `references/`.

#### 4.2.2 `.opencode/skills/spec-manager/references/`
| Archivo | Contenido |
|---|---|
| `rules-index-template.md` | Template del manifiesto |
| `rules/10-architecture.md` … `rules/70-delivery.md` | Templates de dominio con reglas base genéricas |
| `spec-template.md` | Template portátil de spec con Cumplimiento de reglas |
| `rule-catalog.md` | Catálogo de reglas base + guardrails sugeridos |

#### 4.2.3 `.opencode/skills/spec-manager/scripts/`
- `init-rules.sh`: genera la estructura de reglas de forma determinista y no destructiva.
  - **Interfaz**: `init-rules.sh [directorio-destino]`.
  - **Garantías**: no sobrescribe, no borra, idempotente, no toca `opencode.json` ni crea specs.
  - **Salida**: `creado` / `omitido` por archivo + resumen.
- `check-rules.sh`: verifica la estructura (Paso 0).
  - **Interfaz**: `check-rules.sh [directorio]`.
  - **Salida**: `exit 0` si está completa; `exit 1` con la lista de faltantes y las opciones.

#### 4.2.4 `.githooks/` + `scripts/install-hooks.sh`
- **Responsabilidad**: enforcement determinista e independiente del agente.
- **`pre-commit`**: archivos prohibidos, PII/secretos, `public/i18n`, guardrail de UI.
- **`commit-msg`**: formato `SPEC-XXX: ...`.
- **Instalación**: `git config core.hooksPath .githooks`.

#### 4.2.5 `.opencode/plugins/rules-guard/`
- **Responsabilidad**: prevención en tiempo de ejecución del agente.
- **API**: V2 `Plugin.define` de `@opencode/plugin` (fijado a la versión del runtime).
- **Hooks**: `tool.execute.before` (bloqueo) y `session.hook("context")` (recordatorio de reglas).
- **Fail-safe**: toda comprobación envuelta en `try/catch`; ante error, no bloquea.

#### 4.2.3 `docs/rules/` (por proyecto)
Manifiesto + 7 archivos de dominio. En p40la-ihost se migra el contenido real.

### 4.3 Modelo de datos del manifiesto

`docs/rules/00-index.md` contiene:
- **Perfil del proyecto**: nombre, stack, rutas clave.
- **Tabla de dominios**: archivo | alcance | cuándo cargarlo.
- **Tabla de guardrails**: regla | comando | cuándo se ejecuta.
- **Reglas no negociables** (seguridad) que se cargan siempre.

### 4.4 Contrato de IDs

| Prefijo | Dominio | Ejemplo |
|---|---|---|
| `ARCH-` | arquitectura/stack | `ARCH-LAYERS` |
| `UI-` | UI/componentes | `UI-SELECT` |
| `UX-` | UX/navegación/modales | `UX-MODAL` |
| `DATA-` | datos/migraciones | `DATA-PRODCOPY` |
| `SEC-` | seguridad/PII | `SEC-NO-PII` |
| `TEST-` | pruebas | `TEST-LOCAL` |
| `DELIV-` | git/deploy | `DELIV-WORKTREE` |

## 6. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-ARCH-001: `SKILL.md` no contiene rutas ni componentes de un proyecto específico fuera de ejemplos marcados; funciona en un repo vacío.
- [ ] CA-ARCH-002: Existe el contrato de estructura documentado en la skill y coincide con lo que genera `/spec init`.
- [ ] CA-UI-001: `/spec init` sobre un directorio vacío crea `AGENTS.md`, `docs/rules/00-index.md` y los 7 archivos de dominio.
- [ ] CA-UI-002: `/spec init` sobre un directorio con archivos existentes no sobrescribe ninguno y reporta los omitidos.
- [ ] CA-UX-001: El Paso 0 aborta con advertencia explícita cuando falta `docs/rules/00-index.md` y ofrece `/spec init`.
- [ ] CA-UX-002: El Paso 0 declara que las reglas del proyecto prevalecen sobre la skill.
- [ ] CA-UX-003: Toda spec nueva incluye la tabla «Cumplimiento de reglas» con al menos las reglas aplicables del dominio tocado.
- [ ] CA-SEC-001: `docs/rules/50-security.md` contiene PII, secretos, datos sintéticos, copias de producción en `/tmp`, higiene de commits y protocolo de incidente.
- [ ] CA-SEC-002: `AGENTS.md` conserva en su cuerpo las reglas no negociables de seguridad y un índice a `docs/rules/`.
- [ ] CA-DATA-001: `docs/rules/40-data.md` contiene la regla de validar contra copia de producción y prohíbe escribir en producción.
- [ ] CA-TEST-001: El gate de pre-release ejecuta los guardrails declarados en el manifiesto y falla si alguno falla.
- [ ] CA-DELIV-001: `docs/rules/70-delivery.md` contiene worktrees, merge con confirmación, prohibición de `--force` y formato de commit `SPEC-XXX: ...`.
- [ ] CA-DELIV-002: `docs/project-rules.md` queda como stub que apunta a `docs/rules/` sin contenido duplicado.
- [ ] CA-DELIV-003: `opencode.json` usa `instructions` con `docs/rules/*.md` y claves vigentes.
- [ ] CA-DELIV-004: El cálculo del próximo ID usa `docs/specs/SPEC-*.md` + issues, no el README.
- [ ] CA-DELIV-005: `pre-commit` bloquea un commit con `*.db`, `*.db-wal`, `*.db-shm`, `*.pdf` o `backups/`.
- [ ] CA-DELIV-006: `commit-msg` rechaza un mensaje sin `SPEC-XXX:` ni prefijo permitido.
- [ ] CA-DELIV-007: El plugin bloquea `rm data/app.db`, `pkill`, `git push --force` y `git reset --hard`.
- [ ] CA-UI-003: `pre-commit` y el plugin bloquean `<select>` nativo y checkbox nativo en `.tsx` (fuera de la allowlist).
- [ ] CA-UI-004: El plugin bloquea escribir en `public/i18n/**` y sugiere `frontend/public/i18n/**`.
- [ ] CA-SEC-003: `pre-commit` bloquea un secreto (clave privada, token) en líneas agregadas.
- [ ] CA-ARCH-003: `scripts/install-hooks.sh` es idempotente y deja `core.hooksPath=.githooks` verificado.
- [ ] CA-NF-004: Un hook/plugin que falla internamente deja pasar la operación (fail-safe), sin romper el commit ni la sesión.

### 5.2 No funcionales

- [ ] CA-NF-001: `SKILL.md` ≤ 500 líneas.
- [ ] CA-NF-002: No hay regla duplicada textualmente entre `AGENTS.md`, `docs/rules/*` y `SKILL.md` (solo referencias por ID).
- [ ] CA-NF-003: Nada de PII ni secretos en los archivos generados (datos sintéticos).

### 5.3 Testing

- **Manual**: ejecutar `/spec init` en `/tmp` sobre (a) directorio vacío, (b) directorio con un `AGENTS.md` previo, (c) directorio ya inicializado. Verificar creación, no-sobrescritura e idempotencia.
- **Regresión**: `/spec create` sobre un repo sin reglas debe abortar; con reglas debe producir la tabla de cumplimiento.
- **Verificación**: grep de IDs en las specs de ejemplo.

## 7. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Templates de reglas en `references/` | 2 h | Ninguna |
| 2 | Reescritura de `SKILL.md` (Paso 0, IDs, init, gate) | 3 h | Fase 1 |
| 3 | Migración de `AGENTS.md` + `project-rules.md` → `docs/rules/` | 3 h | Fase 1 |
| 4 | `docs/specs/templates/spec-template.md` + `opencode.json` | 1 h | Fase 3 |
| 5 | Pruebas de `init` en `/tmp` | 1 h | Fase 2 |

### 6.2 Milestones

1. **MVP**: skill portable + Paso 0 + init + estructura en p40la-ihost.
2. **V1.0**: gate de pre-release y cálculo de ID desde filesystem.

## 8. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Conflicto con SPEC-135, que edita `AGENTS.md` y `docs/project-rules.md` en paralelo | Alta | Medio | Se portaron desde ya `UX-DETAIL-READONLY` (§4.8) y `UI-TOGGLE` (§4.9) a `docs/rules/30-ux.md` y `20-ui.md`. Al mergear SPEC-135, resolver conservando la versión de `docs/rules/` y sin reintroducir contenido en `AGENTS.md`/`project-rules.md`. |
| Las 132 specs citan `docs/project-rules.md §X` | Alta | Bajo | Dejar stub con tabla de redirección sección→archivo |
| Duplicación residual | Media | Medio | Una regla = un ID; revisión con grep en el gate |
| `init` destructivo | Baja | Alto | Escritura no destructiva por diseño + prueba en `/tmp` |
| Sobrecarga de contexto | Media | Bajo | Carga por dominio según el tipo de spec |

## 9. Notas y Referencias

- OpenCode skills: `https://opencode.ai/docs/skills/`
- OpenCode instructions: `https://opencode.ai/docs/instructions/`
- Auditoría interna: 79 reglas, 18% cubiertas; 29 specs recientes con UI.
- Precedentes de regresión: SPEC-114→115 (`app.edit`), SPEC-122→126 (modales), SPEC-032/033 (i18n).

## 10. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-17 | opencode | Creación inicial de la especificación |
| 2026-11-17 | opencode | Enforcement con git hooks y plugin `rules-guard` (REQ-018..REQ-023) |
| 2026-11-17 | opencode | Release: merge a `main` de `feature/SPEC-136` (commits `f35d506e`, `6f4f88f6`) |

---

## 11. Verificación de release

| Criterio | Evidencia |
|---|---|
| `CA-ARCH-001` | `check-rules.sh` pasa; `SKILL.md` sin rutas de proyecto fuera de ejemplos |
| `CA-ARCH-002` | Contrato en `SKILL.md` == archivos que genera `init` (12 archivos) |
| `CA-ARCH-003` | `install-hooks.sh` idempotente; `core.hooksPath=.githooks` verificado |
| `CA-UI-001` | `init-rules.sh` en `/tmp` vacío: 12 creados (9 reglas + 3 hooks) |
| `CA-UI-002` | `init-rules.sh` con `AGENTS.md` previo: omitido, contenido intacto |
| `CA-UI-003` | Plugin 14/14 casos: bloquea `<select>` y checkbox en `.tsx` |
| `CA-UI-004` | Plugin bloquea `public/i18n/` y permite `frontend/public/i18n/` |
| `CA-UX-001` | `check-rules.sh` en dir vacío: exit 1 con faltantes y opciones |
| `CA-UX-002` | `docs/rules/00-index.md` §1 declara precedencia del proyecto |
| `CA-UX-003` | Template de spec incluye sección «Cumplimiento de reglas» |
| `CA-SEC-003` | `pre-commit` bloquea clave privada: test 5/5 PASS |
| `CA-DELIV-005` | `pre-commit` bloquea `*.db`: test 5/5 PASS |
| `CA-DELIV-006` | `commit-msg` rechaza mensaje sin `SPEC-XXX:`: test 5/5 PASS |
| `CA-DELIV-007` | Plugin bloquea `rm data/`, `pkill`, `--force`, `reset --hard` |
| `CA-TEST-001` | Gate documentado y guardrails declarados en el manifiesto |
| `CA-NF-001` | `SKILL.md` ≤ 500 líneas |
| `CA-NF-003` | Sin PII real; solo datos `demo-*` |
| `CA-NF-004` | Plugin y hooks envueltos en fail-safe |

Comandos ejecutados: `go build ./...`, `check-rules.sh`, `node test-guard.mjs` (14/14), `bash test-hooks.sh` (5/5), `check-ui-rules.cjs`.
