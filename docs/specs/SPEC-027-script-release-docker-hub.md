---
title: "Script de release automático para Docker Hub con bump de versión"
id: "SPEC-027"
status: "released"
author: "paulomcnally"
created: "2026-08-16"
updated: "2026-08-16"
github_issue: 27
---

# {{title}}

**ID**: {{id}}  
**Estado**: {{status}}  
**Autor**: {{author}}  
**Creado**: {{created}}  
**Actualizado**: {{updated}}

---

## 1. Resumen Ejecutivo

Actualmente publicar una nueva imagen Docker requiere editar manualmente `docker-compose.yml`, commitear, crear un tag git `v*` y pushear. Este proceso es repetitivo y propenso a errores (versiones inconsistentes, olvidar el tag, etc.).

Esta spec define un script shell `scripts/release.sh` que automatiza todo el flujo: consulta la última versión publicada en Docker Hub, calcula el siguiente número de patch, actualiza los archivos necesarios, commitea, crea el tag, hace push e imprime la URL del workflow de GitHub Actions que se dispara automáticamente.

El script se ejecuta desde la rama `main` y utiliza el binario `gh` para obtener la URL del workflow. No requiere Node.js ni Docker localmente, solo `bash`, `curl`, `git` y `gh`.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: El script `scripts/release.sh` debe existir y ser ejecutable.
2. **REQ-002**: Debe consultar la API pública de Docker Hub (`https://hub.docker.com/v2/repositories/paulomcnally/p40la-ihost/tags/`) y determinar la última versión semver publicada, ignorando tags como `latest` o sufijos de arquitectura (`-amd64`, `-arm64`, `-armv7`).
3. **REQ-003**: Debe incrementar automáticamente el componente **patch** de la versión (ej: `0.4.4` → `0.4.5`).
4. **REQ-004**: Debe actualizar la variable `VERSION` en `docker-compose.yml`.
5. **REQ-005**: Debe actualizar el campo `version` en `frontend/package.json` para mantener consistencia.
6. **REQ-006**: Debe crear un commit con el mensaje `bump version to X.Y.Z`.
7. **REQ-007**: Debe crear un tag anotado `vX.Y.Z` apuntando al commit de bump.
8. **REQ-008**: Debe hacer `git push` del commit y `git push origin vX.Y.Z`.
9. **REQ-009**: Debe imprimir en log cada paso con prefijo de fecha/hora.
10. **REQ-010**: Al finalizar, debe mostrar la URL del workflow de GitHub Actions disparado por el tag, obtenida mediante el binario `gh`.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-011**: Permitir especificar manualmente la nueva versión como primer argumento (ej: `./scripts/release.sh 0.5.0`), siempre validando que no exista ya en Docker Hub ni como tag git.
2. **REQ-012**: Validar prerequisitos antes de mutar el repo: estar en `main`, working tree limpio, `gh` autenticado, remoto `origin` accesible.
3. **REQ-013**: Abortar con mensaje claro si el tag `vX.Y.Z` ya existe local o remotamente.
4. **REQ-014**: Abortar si la nueva versión ya existe como tag en Docker Hub.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-015**: Actualizar cualquier referencia a la versión en `README.md` como parte del bump.

### 2.4 Requerimientos No Funcionales

- **Seguridad**: No escribir credenciales en logs; usar `gh` y git ya configurados en el entorno del usuario.
- **iHost**: No afecta el runtime del iHost; es una herramienta de desarrollo/release.
- **Dependencias**: Usar solo herramientas estándar (`bash`, `curl`, `git`, `jq` opcionalmente, `gh`).
- **Logs**: Salida clara y verbosa para poder auditarse fácilmente.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Docker Hub expone una API REST pública para listar tags:

```
GET https://hub.docker.com/v2/repositories/paulomcnally/p40la-ihost/tags/?page_size=50
```

La respuesta contiene un array `results` donde cada tag tiene un campo `name`. Filtrando nombres que coincidan con `^[0-9]+\.[0-9]+\.[0-9]+$` se obtienen las versiones semver puras. Ordenando con `sort -V` se obtiene la última.

Para obtener la URL del workflow de GitHub Actions después de hacer push del tag, se puede usar:

```bash
gh run list --workflow="Build & Push Docker Image" --branch vX.Y.Z --json url --jq '.[0].url' --limit 1
```

Esto requiere que `gh` esté autenticado y tenga acceso al repositorio.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Leer versión de `docker-compose.yml` local | Simple, no requiere red | Puede estar desactualizado respecto a Docker Hub | ❌ Rechazada |
| Leer versión desde Docker Hub API | Fuente de verdad del registry | Requiere curl e internet | ✅ Seleccionada |
| Usar `gh release create` | Integrado con GitHub | No dispara directamente el workflow de Docker existente | ❌ Rechazada |
| Usar `git tag` + `git push` | Mantiene el flujo actual | Más pasos manuales | ✅ Seleccionada (automatizada por script) |
| Usar `jq` vs `grep`/`sed` | `jq` es más robusto para JSON | Puede no estar instalado en todos los entornos | ✅ Seleccionada con fallback a `grep`/`sed` si no hay `jq` |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Fuente de la versión actual será Docker Hub
- **Contexto**: `docker-compose.yml` local tiene `VERSION=0.2.1` pero Docker Hub ya tiene `0.4.4`. La fuente confiable es el registry.
- **Decisión**: El script consulta Docker Hub para calcular la siguiente versión.
- **Consecuencias**: Evita versiones duplicadas o saltos hacia atrás; requiere conectividad a internet.

**ADR-002**: Bump automático solo del componente patch
- **Contexto**: La mayoría de los releases son correcciones o cambios pequeños.
- **Decisión**: Por defecto se incrementa `patch`. Para minor/major se usa argumento manual.
- **Consecuencias**: Simplicidad; versiones mayores requieren intención explícita.

**ADR-003**: Uso obligatorio de `gh` para URL del action
- **Contexto**: El usuario pidió explícitamente usar `gh`.
- **Decisión**: El script usa `gh run list` para obtener la URL del workflow.
- **Consecuencias**: Requiere autenticación previa con `gh auth login`.

## 4. Diseño Técnico

### 4.1 Componentes

#### 4.1.1 `scripts/release.sh`
- **Responsabilidad**: Automatizar el flujo completo de release.
- **Interfaz**: `./scripts/release.sh [nueva-version]`.
- **Dependencias**: `bash`, `curl`, `git`, `gh`. Opcionalmente `jq`.
- **Ubicación**: `scripts/release.sh`.

### 4.2 Flujo del script

```
1. Validar prerequisitos (main, working tree limpio, gh autenticado)
2. Consultar Docker Hub API → lista de tags semver
3. Determinar última versión publicada
4. Calcular nueva versión (patch+1 o argumento manual)
5. Validar que nueva versión no exista en Docker Hub ni como git tag
6. Actualizar docker-compose.yml y frontend/package.json
7. git add + commit "bump version to X.Y.Z"
8. git tag -a vX.Y.Z -m "Release vX.Y.Z"
9. git push origin main && git push origin vX.Y.Z
10. Esperar brevemente y obtener URL del workflow con gh run list
11. Imprimir resumen y URL del action
```

### 4.3 Cambios en archivos existentes

- `docker-compose.yml`: actualizar `VERSION=X.Y.Z`.
- `frontend/package.json`: actualizar `"version": "X.Y.Z"`.
- `scripts/release.sh`: archivo nuevo.
- Posiblemente `README.md` si se implementa REQ-015.

### 4.4 Dependencias

- **Internas**: Ninguna modificación al código de la aplicación.
- **Externas**: Docker Hub API, GitHub API vía `gh`.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado que ejecuto `./scripts/release.sh` sin argumentos, cuando termina, entonces crea el tag `vX.Y.Z`, lo pushea y muestra la URL del workflow de GitHub Actions.
- [ ] CA-002: Dado que Docker Hub tiene la versión `0.4.4`, cuando ejecuto el script, entonces calcula `0.4.5` como nueva versión.
- [ ] CA-003: Dado que ejecuto `./scripts/release.sh 0.5.0`, cuando termina, entonces usa `0.5.0` en lugar del cálculo automático.
- [ ] CA-004: Dado que el tag `v0.4.5` ya existe, cuando ejecuto el script, entonces aborta antes de modificar archivos.

### 5.2 No funcionales

- [ ] CA-NF-001: El script termina con código de salida distinto de cero ante cualquier error, sin dejar el repo en estado inconsistente.
- [ ] CA-NF-002: Los logs incluyen timestamp en cada línea y describen claramente el paso en curso.

### 5.3 Testing

- **Manual**: Ejecutar el script en local (sin modificar archivos con `--dry-run` si se implementa) y verificar salida.
- **Real**: Ejecutar el script completo, confirmar que el workflow se dispara y la imagen aparece en Docker Hub.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Crear spec y actualizar tracker | 10 min | Ninguna |
| 2 | Implementar `scripts/release.sh` con validaciones y lógica de Docker Hub | 30 min | Fase 1 |
| 3 | Probar sintaxis y flujo localmente (sin push real) | 15 min | Fase 2 |
| 4 | Ejecutar release real y verificar workflow | 5 min | Fase 3 |
| 5 | Actualizar spec a `released` y cerrar issue | 5 min | Fase 4 |

### 6.2 Milestones

1. **MVP**: Script funcional que lee Docker Hub, actualiza archivos, commitea, taggea y pushea.
2. **V1.0**: Incluye validaciones robustas, logs detallados y URL del action vía `gh`.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Docker Hub API no responde | Baja | Alto | Validar respuesta HTTP y abortar con mensaje claro |
| `gh` no autenticado | Media | Alto | Validar `gh auth status` al inicio |
| Tag ya existe en git o Docker Hub | Baja | Medio | Verificar antes de mutar archivos |
| Working tree sucio | Media | Medio | Abortar si hay cambios sin commitear |
| No estar en rama `main` | Baja | Medio | Validar rama actual |

## 8. Notas y Referencias

- URL del registry: https://hub.docker.com/r/paulomcnally/p40la-ihost/tags
- API de Docker Hub: https://hub.docker.com/v2/repositories/paulomcnally/p40la-ihost/tags/
- Workflow existente: `.github/workflows/docker-publish.yml`
- Documentación actual: `README.md` sección "Releasing a new Docker version via tag"

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-16 | paulomcnally | Creación inicial de la especificación |
| 2026-08-16 | paulomcnally | Pasa a in_progress para implementación |
| 2026-08-16 | paulomcnally | Script implementado y probado localmente; pasa a released |
