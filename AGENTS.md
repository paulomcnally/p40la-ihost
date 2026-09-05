# AGENTS.md - Reglas del Repositorio

> **Última actualización**: 2026-09-04  
> **Proyecto**: p40la-ihost  
> **Sistema de specs**: v1.2

---

## 🎯 Propósito

Este repositorio utiliza un sistema formal de especificaciones técnicas (specs) para gestionar el desarrollo de funcionalidades. Todo cambio significativo DEBE pasar por el flujo de specs.

El proyecto corre en un **iHost con recursos muy limitados**, por lo que todas las decisiones técnicas deben priorizar simplicidad, bajo consumo de memoria y mínimas dependencias.

---

## ⚠️ Regla de Oro

**Antes de escribir código: la spec primero. Siempre.**  
Si la spec no existe, no se escribe código. Si la spec no contempla el cambio, la spec se actualiza primero. No hay excepciones.

---

## 📋 Reglas Fundamentales

### 0. Reglas críticas (aprendidas de errores)

- **NUNCA ejecutar `rm -f data/app.db` ni ningún comando que elimine la base de datos del usuario.** La DB local es producción. Si se necesitan pruebas limpias, usar `/tmp/test-app.db`.
- **Usar SIEMPRE `killall` para matar procesos, NUNCA `pkill`.** `pkill` falla silenciosamente en este entorno. Comando correcto: `killall p40la-ihost`.
- **Verificar SIEMPRE que el server responde antes de informar al usuario que puede acceder.** Ejecutar `./scripts/check-server.sh` (revisa `GET /health` esperando `{"status":"ok"}`, con reintentos; en fallo muestra el log y sale con error). Solo entonces informar al usuario. Si no responde, revisar el log y reiniciar.
- **Seguir siempre el patrón de UI existente.** Para páginas de listado: cards con menú de acciones (3 puntos) + EmptyCard (título, descripción, botón) cuando no hay registros. Nunca formularios inline en listados. Referencia: `HomesPage.tsx`.
- **Inputs SIEMPRE con tokens del tema, jamás colores hardcodeados.** Todo `input`/`select`/`textarea` DEBE usar `bg-card` (fondo) y `text-text`/`text-text-secondary` (texto/placeholder). Nunca `bg-white`, `dark:bg-[#2c2c2e]` ni `text-black`. El color global de los form controls lo fija `frontend/src/index.css`; no duplicarlo por formulario. **Todo formulario nuevo debe verificarse en darkmode** (texto y placeholders legibles) antes de considerarse completo. Ver SPEC-060.
- **Páginas de detalle: flecha atrás en el header, jamás links "← Título" en el contenido.** Si una página tiene una lista padre (detalle, crear o editar), el header muestra una flecha atrás en móvil en vez de la hamburguesa. Registrar la ruta en el mapa `BACK_ROUTES` de `frontend/src/components/DashboardLayout.tsx`. Nunca crear links de retorno dentro del contenido. Ver SPEC-063.
- **Validar prerequisitos en backend Y frontend.** Si una entidad depende de otra (ej: servicios → instituciones), validar en ambos lados y redirigir al formulario de la dependencia si no existe.
- **Crear usuarios de prueba directamente en la DB cuando se necesite autenticar.** El agente tiene permiso y obligación de crear usuarios vía SQLite para poder probar endpoints protegidos. Ejecutar `./scripts/create-test-user.sh`: crea/actualiza el usuario (default `test@test.com` / `test1234`), hace login y deja las cookies en `/tmp/cookies.txt`. Personalizable con `EMAIL`, `PASSWORD`, `DB_PATH` y `PORT`. Usar luego: `curl -s -b /tmp/cookies.txt http://localhost:8088/api/...`
- **La fuente de verdad del i18n es `frontend/public/i18n/`, NO `public/i18n/`.** El build de Vite (`npm run build` en `frontend/`) usa `emptyOutDir: true` sobre `public/`: borra TODO su contenido y lo regenera desde `frontend/public/` (copia `i18n/`, `index.html`, assets). Por lo tanto:
  - Cualquier cambio de traducción se edita SIEMPRE en `frontend/public/i18n/{es,en}.json`.
  - Nunca editar `public/i18n/*.json` directamente: esos archivos son salida del build y se sobrescriben/perden en el próximo `npm run build`.
  - Después de editar i18n, correr `npm run build` (en `frontend/`) y verificar con `curl -s http://localhost:8088/i18n/es.json` que las claves nuevas estén servidas.
  - Precedente: SPEC-032/033 — las claves `settings.alerts.*` y `settings.voicemonkey.*` se editaron en `public/i18n/` y desaparecieron en el build (fallo de i18n reportado por el usuario).

### 1. Cero código sin spec

- **No se escribe una sola línea de código sin que exista una spec que lo defina.**
- El agente **JAMÁS** debe hacer cambios en el código que no estén contemplados en una spec existente o en creación.
- Si durante el desarrollo surge un cambio no contemplado (nuevo archivo, nuevo índice, cambio de modelo, etc.), la spec DEBE actualizarse **antes** de escribir el código.

### 2. ¿Qué necesita spec?

**TODO necesita spec, excepto**:
- Hotfixes críticos (servicio caído, seguridad)
- Cambios cosméticos (espacios, comentarios, formato)
- Actualizaciones de dependencias sin cambio de comportamiento

Cualquier cambio que toque lógica de negocio, base de datos, APIs, eventos, configuración de infraestructura o UI requiere spec **obligatoriamente**.

### 3. Flujo estricto

```
1. Identificar necesidad → 2. Crear spec (draft) → 3. Completar spec (pending_execution) → 4. Desarrollar
```

- **Paso 2 y 3**: Se crea la spec, se documenta todo (requerimientos, diseño, criterios de aceptación).
- **Paso 4**: Solo entonces se escribe código. Nunca a la inversa.
- Si se detecta algo no previsto durante el desarrollo → volver al paso 2, actualizar la spec, luego continuar.
- Si durante la evaluación manual local el usuario solicita cambios (incluso pequeños), el agente debe:
  1. Actualizar la spec para reflejar los cambios solicitados.
  2. Crear/actualizar la lista de tareas (`todo`) con cada cambio pendiente.
  3. Implementar los cambios y volver a dejar la app corriendo en local para reevaluación.
  4. Solo cuando el usuario confirme, se considera el spec cerrado.

### 4. Documentación mínima

Cada spec DEBE incluir:
- Requerimientos claros y priorizados (P0/P1/P2)
- Investigación técnica con fuentes cuando aplica
- Decisiones arquitectónicas justificadas
- Criterios de aceptación verificables
- Plan de implementación con estimaciones
- Consideraciones específicas de iHost (recursos limitados, SQLite, etc.)

### 5. IDs únicos y secuenciales

Los IDs de spec (`SPEC-XXX`) son inmutables y secuenciales. Nunca se reutiliza un número.

---

## 🔄 Flujo de Trabajo

### Fase 1: Requerimiento

1. Se identifica una necesidad
2. Se crea una spec usando la skill `spec-manager` o el template
3. Estado inicial: `draft`

### Fase 2: Definición

1. Investigar soluciones técnicas
2. Documentar decisiones y arquitectura
3. Definir criterios de aceptación
4. Validar con stakeholders
5. Mover a estado: `pending_execution`

### Fase 3: Desarrollo

1. Crear rama/feature desde la spec
2. Desarrollar según la especificación
3. Estado: `in_progress`

### Fase 4: Validación

1. Pasar criterios de aceptación
2. Code review
3. QA/Testing (tests automáticos + pruebas manuales locales)
4. El agente debe correr la aplicación en local y dejarla disponible para evaluación del usuario
5. El usuario confirma que los cambios están correctos tras probar manualmente
6. Estado: `pending_release`

### Fase 5: Release

1. Hacer commit de TODOS los cambios pendientes a `main` con mensaje descriptivo (formato: `SPEC-XXX: descripción corta`)
2. Push a `main`
3. Documentar commit/versión en la spec
4. Estado: `released`
5. **Cerrar el spec SIEMPRE implica commitear y pushear los cambios.** No cerrar un spec con cambios sin commitear.
6. **Actualizar SIEMPRE el label de GitHub al estado final.** Al cambiar el estado de una spec, ejecutar `gh issue edit <number> --remove-label "spec/<anterior>" --add-label "spec/<nuevo>"`. Si el estado es `released` o `cancelled`, además cerrar el issue con `gh issue close <number>`. **No cerrar un issue sin antes actualizar su label al estado correcto.**
7. **Implementación y release son inseparables.** Si el código de una feature está commiteado en `main`, la spec DEBE estar `released`. Toda spec en `pending_release` (o anterior) con código en `main` es un error y debe liberarse o documentar el bloqueo. Antes de declarar terminada una spec, ejecutar `/spec check-release`.
8. **El release incluye TODOS estos pasos en el mismo ciclo:** actualizar `status` del frontmatter y del cuerpo (`**Estado**:`), actualizar el README tracker (tabla + contadores), cambiar el label de GitHub, cerrar el issue y documentar el commit/versión en el historial. No declarar una spec terminada hasta completar todos.

---

##  Restricciones de Despliegue (Agentes)

### Reglas absolutas

- **JAMÁS** ejecutar `git push` a ramas protegidas sin confirmación explícita del usuario.
- **JAMÁS** mergear a `main` directamente desde una rama de feature.
- **JAMÁS** ejecutar `--force` o `--force-with-lease` en `main` o ramas de release.
- **Solo usar labels de spec definidos en la skill.** Los labels válidos son: `spec/draft`, `spec/pending-execution`, `spec/in-progress`, `spec/pending-release`, `spec/released`, `spec/cancelled`. **NUNCA** agregar labels genéricos como `spec` o cualquier otro que no esté en esta lista. Solo se agregan labels cuando el flujo de specs lo requiere explícitamente.

### ️ Regla CRÍTICA: Worktrees por sesión (multi-ventana)

- **Cada ventana/sesión de opencode trabaja en SU PROPIO git worktree.** El proyecto permite hasta 6 ventanas trabajando en specs distintos en paralelo; compartir un mismo checkout git hace que operaciones como `checkout` o `reset` pisen el trabajo sin commitear de otras sesiones (error crítico documentado: SPEC-018/024/025/026 + SPEC-030-033).
- **Para empezar una spec**: `./scripts/new-worktree.sh SPEC-XXX` crea un directorio aislado (`p40la-ihost-spec-XXX`) en su propia rama `feature/SPEC-XXX`. Luego abrir una nueva ventana de opencode en ese directorio.
- **JAMÁS ejecutar `git checkout`, `git switch`, `git reset`, `git stash` ni `git clean`** sobre el worktree principal compartido ni sobre el worktree de otra sesión. Cada sesión solo opera su propio worktree.
- **Verificar antes de operar**: `git worktree list` muestra todos los worktrees y sus ramas. Antes de cualquier operación git, confirmar en qué worktree se está (`git branch --show-current`).
- **JAMÁS ejecutar `git reset --hard`** salvo que el usuario lo pida explícitamente para su propio worktree. Un `reset --hard` descarta TODO cambio sin commitear (propio o ajeno).
- **Para liberar una spec** (pasar a `released`): mergear la rama del worktree a `main` (con confirmación del usuario), luego cerrar el issue y actualizar labels desde cualquier worktree.
- **Al liberar una spec, limpiar SU worktree y rama** (paso obligatorio del release, SPEC-038): `git worktree remove ../p40la-ihost-spec-XXX` y `git branch -d feature/SPEC-XXX` (usar SIEMPRE `-d`, nunca `-D`: protege ramas no mergeadas). Verificar con `git worktree list`. **Esta limpieza SOLO aplica a specs `released`**; los worktrees de specs en desarrollo (`draft`/`in_progress`/`pending_release`) nunca se tocan.

### ️ Regla CRÍTICA: Arquitectura Multi-Arch (iHost + desarrollo)

- **El SONOFF iHost usa procesador ARM64 (aarch64).**
- **TODAS** las imágenes Docker DEBEN ser **multi-arch con 3 plataformas**: `linux/amd64` + `linux/arm/v7` + `linux/arm64`.
- Las versiones 0.1.x tenían estas 3 arquitecturas. **NUNCA** romper esto en 0.2.x o posteriores.
- Comando obligatorio: `docker buildx build --platform linux/amd64,linux/arm/v7,linux/arm64 --push .`
- Verificar con `docker buildx imagetools inspect <image>` que existan los 3 manifests.
- Esta regla aplica a CUALQUIER imagen Docker del proyecto, sin excepciones.

### ️ Regla CRÍTICA: Acceso a la DB de producción en el iHost (SSH)

La app corre como add-on Docker en un **SONOFF iHost**. La DB SQLite vive en el **volumen Docker `p40la`** (montado en `/app/data` dentro del contenedor). El iHost **NO tiene SSH nativo** (puerto 22 cerrado, no existe "Developer Mode"); el acceso se logra con el add-on Docker **`capiloky/sonoff-ssh`** (port host `2222` → container `22`, volumen `p40la` montado). **Guía completa para agentes: `docs/ssh-ihost-access.md`.**

- **Credenciales SSH**: `ssh sshuser@ihost.local -p 2222` con password `password`; sudo con el mismo password (`echo "password" | sudo -S <cmd>`).
- **Sin `sshpass` en la dev machine**: usar Python `paramiko` (disponible). No usar `base64` en args (falla con `Argument list too long`); para subir archivos usar stdin de `exec_command` (`cat > archivo`). **SFTP NO disponible**.
- **`sqlite3` NO viene instalado** en el contenedor SSH: instalar una vez con `sudo apt-get install -y sqlite3`.
- **Regla de oro de operación**: si una petición del usuario implica manipular la DB del volumen (`data` de producción), el agente DEBE:
  1. Pedir al usuario detener el contenedor `p40la-ihost` ANTES de cualquier escritura (modo WAL: `app.db-wal` activo con el server corriendo → corrupción).
  2. **Respaldar siempre** antes de modificar: `sudo sqlite3 /app/data/app.db ".backup '/app/data/backups/<fecha>-<motivo>/app.db'"` + verificar `PRAGMA integrity_check;` + descargar copia local a `/home/paulomcnally/p40la-db-backups/`.
  3. Probar el SQL en local sobre una copia del backup antes de aplicarlo en producción.
  4. Ejecutar cambios multi-tabla envueltos en `BEGIN/COMMIT` vía `sudo sqlite3 /app/data/app.db ".read /tmp/script.sql"`.
  5. Verificar conteos e integridad, luego pedir al usuario arrancar el contenedor.
- **No modificar el esquema SQLite** de `debts`/`debt_bills` (política del usuario); solo insertar/actualizar datos. Ver SPEC-054 para el modelo de deudas.
- Verificar siempre el server con `curl -s http://ihost.local:8088/health` → `{"status":"ok"}`.
- Para autenticar endpoints protegidos contra la DB del volumen (`/app/data/app.db`), generar el hash con `go run ./scripts/genhash.go <password>` y aplicar el INSERT vía SSH (el script `create-test-user.sh` es solo local por ahora; la variante remota es trabajo futuro, ver SPEC-059 REQ-011).

### Pruebas locales obligatorias

- **Todo cambio significativo debe probarse en local** (tests, build, ejecución básica) antes de intentar commits o push.
- **No se realiza commit** de código que no compile, que falle los tests o que no haya sido validado en el entorno de desarrollo local.
- Antes de solicitar confirmación para merge/push, el agente debe verificar que la aplicación levanta correctamente con la base de datos SQLite y que los criterios de aceptación del spec pasan en local.

### Flujo de pruebas con el usuario

1. **Después de implementar cambios de un spec, preguntar SIEMPRE:** 🤖 *"¿Querés que corra el server en local para que hagas pruebas?"*
2. **NO marcar tareas como completadas** hasta que el usuario confirme que las pruebas manuales fueron satisfactorias.
3. **Preguntar explícitamente:** 🤖 *"¿Las pruebas fueron satisfactorias? ¿Puedo cerrar el spec?"*
4. **Solo después de confirmación del usuario** se marcan los criterios de aceptación como pass y se cambia el estado del spec.
5. **Si el usuario reporta errores**, no cerrar el spec. Corregir, volver a preguntar, repetir.

### Convención de comunicación

- 🤖 **Icono de robot**: Se usa ANTES de cualquier pregunta que sea parte del flujo de interacción (correr server, confirmar implementación, cerrar spec, etc.). Esto indica al usuario que es una pregunta de flujo, no información técnica.

### Flujo permitido

1. Desarrollar en rama `feature/SPEC-XXX-*` (local).
2. Ejecutar pruebas locales (tests, build y validación manual mínima).
3. Correr la aplicación en local para que el usuario evalúe los cambios manualmente.
4. El usuario confirma que todo está correcto; solo entonces se considera el spec listo para cerrar/publicar.
5. Solicitar confirmación al usuario para merge/push.
6. Solo el usuario puede aprobar deploys a producción/iHost.

---

## 🗂️ Estructura de Carpetas

```
/
├── .opencode/
│   └── skills/
│       └── spec-manager/
│           └── SKILL.md         # Skill de gestión de specs
├── docs/
│   ├── specs/
│   │   ├── README.md            # Tracker y contador
│   │   ├── templates/
│   │   │   └── spec-template.md # Template base
│   │   └── SPEC-XXX-*.md        # Especificaciones
│   ├── project-rules.md         # Reglas de stack y arquitectura
│   ├── infrastructure.md        # Infraestructura y deploy
│   └── ssh-ihost-access.md      # Guía de acceso SSH a la DB del iHost
├── AGENTS.md                    # Este archivo
├── opencode.json               # Configuración de opencode
├── src/                        # Código fuente backend
├── public/                     # Frontend estático básico
└── package.json
```

---

## 🤖 Uso de la Skill `spec-manager`

La skill está registrada en `.opencode/skills/spec-manager/SKILL.md`.

### Comandos disponibles

| Comando | Descripción |
|---------|-------------|
| `/spec create "Título"` | Crear nueva spec con investigación automática |
| `/spec list [estado]` | Listar specs, opcionalmente filtradas |
| `/spec status <id> <estado>` | Cambiar estado de una spec |
| `/spec show <id>` | Mostrar resumen de una spec |
| `/spec check-release` | Verificar que no haya specs colgadas (código en main sin estado released) |
| `/spec worktree <ID>` | Crear worktree aislado para desarrollar una spec en su propia rama |

### Activación automática

La skill se activa automáticamente cuando el usuario menciona:
- "crear spec", "nueva especificación", "nuevo requerimiento"
- "estado de spec", "cambiar estado", "pending release"
- "contador de specs", "listar specs", "tracking"

---

## ✅ Checklist de Calidad

Antes de marcar una spec como `pending_release`:

- [ ] Todos los requerimientos P0 están implementados
- [ ] Los criterios de aceptación pasan
- [ ] Hay tests unitarios/integración si aplica
- [ ] Se actualizó la documentación técnica si aplica
- [ ] Se hizo code review
- [ ] No hay deuda técnica crítica sin documentar
- [ ] Se verificó que funciona en entornos con recursos limitados (iHost)

---

## 📞 Escalación

Si una spec se bloquea más de 3 días en cualquier estado, se debe:
1. Documentar el bloqueo en la spec (sección de riesgos)
2. Evaluar si se cancela o se redefine
3. Notificar al equipo

---

*Este documento es la fuente de verdad para el sistema de especificaciones. Cualquier cambio al flujo debe actualizar este archivo y la skill correspondiente.*
