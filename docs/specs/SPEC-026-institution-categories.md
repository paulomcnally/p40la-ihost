---
title: "Categorías de Instituciones con Seed y Filtro de Seguros"
id: "SPEC-026"
status: "released"
author: "paulomcnally"
created: "2026-08-16"
updated: "2026-08-16"
github_issue: 26
---

# Categorías de Instituciones con Seed y Filtro de Seguros

**ID**: SPEC-026
**Estado**: released
**Autor**: paulomcnally
**Creado**: 2026-08-16
**Actualizado**: 2026-08-16

---

## 1. Resumen Ejecutivo

Implementar un sistema de **categorías de instituciones** que permita clasificar a las instituciones (proveedores de servicios) en grupos como Seguros, Telecomunicaciones, Servicios Públicos, etc. Cada categoría tiene un `key` interno único e inmutable en inglés que se usa para filtrado programático.

El caso de uso principal es que al agregar un seguro a un auto, el dropdown de servicios disponibles solo muestre servicios de instituciones categorizadas como `insurance`. Esto evita mostrar servicios de telefonía o electricidad en la sección de seguros de un vehículo.

El acceso al CRUD de categorías se realiza desde la página de instituciones (dropdown en header), sin agregar un ítem nuevo al sidebar.

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Crear tabla `institution_categories` con campos: `id`, `key` (unique, inmutable), `name`, `description`, `icon_key`, `created_at`, `updated_at`
2. **REQ-002**: Seed de 20 categorías iniciales con keys en inglés: `insurance`, `telecommunications`, `pay_tv`, `electricity`, `natural_gas`, `water`, `waste`, `banking`, `loans`, `subscriptions`, `entertainment`, `education`, `health`, `transport`, `housing`, `security`, `internet`, `digital_services`, `maintenance`, `professional`
3. **REQ-003**: Agregar columna `category_id` a la tabla `institutions` (FK nullable)
4. **REQ-004**: CRUD backend completo de categorías: `GET/POST/PUT/DELETE /api/institution-categories/*`
5. **REQ-005**: Crear categoría con validación de `key` (lowercase, sin espacios, solo letras/números/guiones bajos) y unicidad
6. **REQ-006**: Editar categoría sin permitir cambiar el `key`
7. **REQ-007**: Al crear/editar institución, poder seleccionar una categoría del dropdown custom `Select`
8. **REQ-008**: En InstitutionsPage, el CreateMenu debe incluir opción "Categorías de instituciones" que abra modal CRUD
9. **REQ-009**: Las cards de instituciones deben mostrar el nombre de la categoría como badge y usar su icono
10. **REQ-010**: `GET /api/autos/:id/available-services` debe filtrar servicios cuya institución tenga `category.key = 'insurance'`
11. **REQ-011**: El modal `AddInsuranceModal` debe mostrar solo servicios de instituciones con categoría `insurance`

### 2.2 Requerimientos No Funcionales

- **iHost**: Sin dependencias nuevas, solo tabla + seed
- **Rendimiento**: JOIN ligero entre services, institutions e institution_categories
- **Integridad**: `key` UNIQUE garantiza filtrado confiable; servicios que pierden su analyzer no rompen FK

---

## 3. Decisiones Técnicas

### ADR-001: Campo `key` interno inmutable

- **Contexto**: Se necesita filtrar programáticamente la categoría "Seguros" sin depender del nombre visible, que el usuario puede editar o duplicar.
- **Decisión**: Agregar `key TEXT NOT NULL UNIQUE` en inglés. El `name` es el valor visible editable. El `key` se define en el seed y no se edita.
- **Consecuencias**: El filtro `WHERE ic.key = 'insurance'` es robusto incluso si el usuario renombra la categoría.

### ADR-002: Seed de 20 categorías

- **Contexto**: El usuario quiere una lista completa para que agregar nuevas categorías sea poco común.
- **Decisión**: Basado en sistemas de bill payment y apps de finanzas personales, se incluyen 20 categorías que cubren los casos de uso más frecuentes.
- **Consecuencias**: Menor necesidad de CRUD manual por parte del usuario.

### ADR-003: Fix de FK en SetAnalyzers

- **Contexto**: Al editar una institución con analyzers asignados, servicios referenciaban esos analyzers via `institution_analyzer_id`, causando `FOREIGN KEY constraint failed`.
- **Decisión**: Calcular diferencias en `SetAnalyzers`: solo eliminar analyzers removidos, y antes de eliminar setear `institution_analyzer_id = NULL` en servicios afectados.
- **Consecuencias**: Se preservan referencias de analyzers no modificados y se evita el error de constraint.

---

## 4. Diseño Técnico

### 4.1 Modelo de datos

**Migración 0011:**
```sql
CREATE TABLE institution_categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    icon_key TEXT DEFAULT 'other',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE institutions ADD COLUMN category_id INTEGER REFERENCES institution_categories(id);
```

### 4.2 APIs

```
GET    /api/institution-categories
GET    /api/institution-categories/:id
POST   /api/institution-categories
PUT    /api/institution-categories/:id
DELETE /api/institution-categories/:id
```

### 4.3 Componentes Frontend

- `InstitutionCategoriesModal`: CRUD de categorías dentro de modal
- `InstitutionFormPage`: selector de categoría usando componente `Select`
- `InstitutionsPage`: opción de categorías en CreateMenu + badge de categoría en cards
- `AddInsuranceModal`: consume `available-services` ya filtrado por backend

---

## 5. Criterios de Aceptación

- [x] CA-001: Seeded de 20 categorías disponibles vía API
- [x] CA-002: CRUD de categorías funciona desde modal
- [x] CA-003: Selector de categoría usa componente `Select` custom
- [x] CA-004: Crear/editar institución guarda `category_id`
- [x] CA-005: Cards de instituciones muestran categoría e icono
- [x] CA-006: `available-services` solo devuelve servicios de categoría `insurance`
- [x] CA-007: Editar institución con analyzer asignado no produce error FK

---

## 6. Plan de Implementación

| Fase | Descripción | Duración |
|------|-------------|----------|
| 1 | Migración 0011 + seed | 15 min |
| 2 | Backend: modelo, storage, service, handlers, rutas | 45 min |
| 3 | Backend: filtrado insurance en available-services + fix FK | 20 min |
| 4 | Frontend: types, API client, Select en formulario | 20 min |
| 5 | Frontend: modal CRUD + badge en cards + íconos | 30 min |
| 6 | Pruebas locales | 20 min |

---

## 7. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-16 | paulomcnally | Creación de la especificación |
| 2026-08-16 | paulomcnally | Implementación completa y validación |
| 2026-08-16 | paulomcnally | Release: verificado en iHost. Commit `3073e2c` (compartido con SPEC-025), versión v0.4.7. Issue #26 cerrado. |
