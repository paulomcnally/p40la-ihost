---
title: "Cambio de puerto por defecto a 8088"
id: "SPEC-003"
status: "released"
author: "p40la-ihost-team"
created: "2026-08-12"
updated: "2026-08-12"
---

# Cambio de puerto por defecto a 8088

**ID**: SPEC-003  
**Estado**: in_progress  
**Autor**: p40la-ihost-team  
**Creado**: 2026-08-12  
**Actualizado**: 2026-08-12

---

## 1. Resumen Ejecutivo

El puerto por defecto `8000` definido en SPEC-001 genera conflicto en el iHost porque otro contenedor ya lo utiliza. Este spec actualiza el puerto por defecto de la aplicación a `8088` en todo el código, configuración, scripts y documentación, manteniendo la posibilidad de sobrescribirlo mediante la variable de entorno `PORT`.

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Cambiar el puerto por defecto de `8000` a `8088` en `internal/config/config.go`.
2. **REQ-002**: Actualizar `Dockerfile`: `EXPOSE 8088` y variable de entorno `PORT=8088`.
3. **REQ-003**: Actualizar `docker-compose.yml` para mapear `8088:8088`.
4. **REQ-004**: Actualizar scripts `scripts/run-local.sh` y `scripts/run-docker.sh` para usar `8088`.
5. **REQ-005**: Actualizar `docs/ihost.md` con el nuevo puerto por defecto.
6. **REQ-006**: Actualizar `docs/project-rules.md` para reflejar el puerto por defecto `8088`.

### 2.2 Requerimientos No Funcionales

- **Compatibilidad**: La variable `PORT` debe seguir permitiendo cambiar el puerto en tiempo de ejecución.
- **iHost**: El nuevo puerto debe estar dentro del rango no privilegiado y evitar conflictos comunes.

---

## 3. Investigación y Decisiones Técnicas

### 3.1 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| 8080 | Alineado con project-rules.md original | Puede ser usado por otros servicios comunes | ❌ Rechazada |
| 8088 | Poco común, evita conflictos, fácil de recordar | Necesita actualizar toda la documentación | ✅ Seleccionada |
| 9000 | También poco común | Algunos add-ons de iHost lo usan | ❌ Rechazada |

### 3.2 Decisiones arquitectónicas

**ADR-001**: Puerto por defecto 8088
- **Contexto**: El puerto 8000 ya está ocupado por otro contenedor en el iHost del usuario.
- **Decisión**: Cambiar el puerto por defecto a `8088` en todo el proyecto.
- **Consecuencias**: Se requiere una nueva versión (`v0.1.1`) y actualización de la imagen publicada.

---

## 4. Diseño Técnico

### 4.1 Archivos a modificar

- `internal/config/config.go`: `PORT` default `8088`.
- `Dockerfile`: `ENV PORT=8088`, `EXPOSE 8088`.
- `docker-compose.yml`: `ports: - "8088:8088"`, `environment: PORT=8088`.
- `scripts/run-local.sh`: `PORT="${PORT:-8088}"`.
- `scripts/run-docker.sh`: sin cambios, usa docker-compose.
- `docs/ihost.md`: puerto por defecto `8088`.
- `docs/project-rules.md`: tabla de stack, puerto por defecto `8088`.

### 4.2 APIs / Contratos

Sin cambios en APIs.

### 4.5 Dependencias

Ninguna nueva.

---

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] **CA-001**: Al levantar el contenedor sin definir `PORT`, el servidor escucha en `8088`.
- [ ] **CA-002**: La variable `PORT` puede seguir cambiándose a otro valor.
- [ ] **CA-003**: `docs/ihost.md` indica `8088` como puerto por defecto.

### 5.2 No funcionales

- [ ] **CA-NF-001**: Imagen publicada como `paulomcnally/p40la-ihost:0.1.1`.

### 5.3 Testing

- Levantar contenedor local y verificar `http://localhost:8088/health`.
- Verificar que `PORT=9000` cambia el puerto correctamente.

---

## 6. Plan de Implementación

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Actualizar puerto en código y configuración | 15 min | Ninguna |
| 2 | Actualizar documentación y scripts | 15 min | Fase 1 |
| 3 | Validar build local y contenedor | 15 min | Fase 2 |
| 4 | Publicar imagen `v0.1.1` | 20 min | Fase 3 |
| 5 | Mergear a `main` | 5 min | Fase 4 |

---

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Olvidar alguna referencia al puerto 8000 | Media | Medio | Buscar `"8000"` en todo el repo antes de commitear |
| Imagen v0.1.1 falla en build multi-arch | Baja | Alto | Reutilizar builder y cache de buildx |

---

## 8. Notas y Referencias

- SPEC-001 definió el puerto original 8000.
- project-rules.md indicaba 8080 como default previo.

---

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-12 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-08-12 | p40la-ihost-team | v0.1.1 implementada: puerto por defecto cambiado a 8088, imagen publicada en Docker Hub
