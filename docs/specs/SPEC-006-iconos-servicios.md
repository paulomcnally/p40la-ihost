---
title: "Corrección de validación de iconos y selector modal con búsqueda para servicios"
id: "SPEC-006"
status: "released"
author: "p40la-ihost-team"
created: "2026-08-15"
updated: "2026-08-15"
github_issue: 7
---

# Corrección de validación de iconos y selector modal con búsqueda para servicios

**ID**: SPEC-006  
**Estado**: draft  
**Autor**: p40la-ihost-team  
**Creado**: 2026-08-15  
**Actualizado**: 2026-08-15

---

## 1. Resumen Ejecutivo

Al crear un servicio con un icono distinto al predeterminado (por ejemplo `delete`), el backend rechaza la petición con el error `{"error":"invalid_request","message":"el icono seleccionado no es válido"}`. Esto ocurre porque el mapa `AllowedIconKeys` en `internal/services/service.go` solo contiene 9 iconos predefinidos (`internet`, `trash`, `water`, `electricity`, `phone`, `tv`, `gas`, `home`, `other`), y cualquier valor fuera de este conjunto es rechazado.

Este spec aborda dos problemas: (1) corregir la validación para aceptar muchos más iconos, y (2) mejorar la experiencia de usuario con un modal de selección de iconos con búsqueda, permitiendo al usuario escribir y encontrar iconos rápidamente.

Consideraciones de iHost: el catálogo de iconos se define como constante en el frontend y backend, sin carga dinámica ni dependencias externas. El modal es puramente frontend (React), sin impacto en memoria del servidor.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Expandir el catálogo de iconos permitidos en el backend a un conjunto amplio (mínimo 50 iconos) que cubra servicios comunes (basura, internet, agua, luz, gas, teléfono, TV, seguridad, limpieza, mantenimiento, educación, salud, seguros, etc.)
2. **REQ-002**: Corregir la validación para que acepte todos los iconos del catálogo expandido. El icono `delete` (o equivalente `trash`) debe funcionar.
3. **REQ-003**: Crear un componente modal de selección de iconos en el frontend con búsqueda por texto (filtrado en tiempo real mientras el usuario escribe)
4. **REQ-004**: El modal debe mostrar los iconos en una grilla visual con su nombre/etiqueta debajo de cada icono

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-005**: El modal debe ser accesible desde el formulario de creación/edición de servicios al hacer clic en el campo de icono
2. **REQ-006**: El icono seleccionado debe mostrarse como preview en el formulario antes de guardar
3. **REQ-007**: Cerrar el modal al seleccionar un icono o al hacer clic fuera del modal

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-008**: Categorizar iconos por tipo (utilidades, hogar, servicios públicos, etc.) con tabs o secciones en el modal
2. **REQ-009**: Soporte para iconos personalizados futuros (subir imagen)

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: El filtrado de iconos debe ser instantáneo (<50ms) al escribir. Máximo 100 iconos en el catálogo inicial.
- **Almacenamiento**: El catálogo de iconos es una constante en código, cero impacto en SQLite.
- **iHost**: Sin dependencias adicionales de npm. Usar solo React y CSS/Tailwind existentes.
- **Bundle size**: El componente modal no debe agregar más de 5KB al bundle JS.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Backend actual**: `internal/services/service.go:20-30` define `AllowedIconKeys` con solo 9 iconos. La validación en línea 144 rechaza cualquier icono fuera de este mapa.
- **Frontend actual**: El formulario de servicios usa un selector simple (probablemente `<select>` o radio buttons) con los mismos 9 iconos limitados.
- **Iconos disponibles**: El proyecto usa iconos de alguna librería (necesita verificación: Lucide, Heroicons, o SVG inline).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A. Expandir `AllowedIconKeys` con ~80 iconos comunes | Simple, sin dependencias nuevas | Lista hardcodeada | ✅ Seleccionada para backend |
| B. Validación abierta (cualquier string) | Máxima flexibilidad | Riesgo de iconos inválidos en DB | ❌ Rechazada |
| C. Catálogo dinámico desde DB | Flexible, actualizable sin deploy | Complejidad innecesaria para iHost | ❌ Rechazada |
| D. Modal con búsqueda + grilla | UX superior, filtrado rápido | Requiere componente nuevo | ✅ Seleccionada para frontend |
| E. Dropdown con búsqueda | Más simple que modal | Menos espacio visual para iconos | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Catálogo de iconos compartido entre frontend y backend
- **Contexto**: Ambos lados necesitan conocer los iconos válidos.
- **Decisión**: Definir el catálogo completo en el backend (`AllowedIconKeys`) y exportar la misma lista al frontend como constante JSON o constante TypeScript.
- **Consecuencias**: Al agregar nuevos iconos, se modifican ambos archivos. Para iHost esto es aceptable dado el bajo ritmo de cambios.

**ADR-002**: Iconos via Lucide React (si ya se usa) o SVG inline
- **Contexto**: Necesitamos iconos consistentes y ligeros.
- **Decisión**: Usar la misma librería de iconos que ya usa el proyecto. Si no hay librería, usar SVG inline simples.
- **Consecuencias**: Sin dependencias nuevas si ya existe una librería.

## 4. Diseño Técnico

### 4.1 Catálogo de iconos propuesto (~80 iconos)

```
Utilidades: internet, water, electricity, gas, phone, tv, trash, recycling, sewer
Hogar: home, apartment, building, key, lock, door, window, garden, garage, parking
Seguridad: shield, alarm, camera, fire, smoke, co_detector
Limpieza: cleaning, broom, mop, laundry, dishwasher
Mantenimiento: wrench, hammer, screwdriver, paint, plumbing, electrical
Salud: health, hospital, pharmacy, ambulance, medical
Educación: school, university, library, books
Transporte: bus, taxi, car, motorcycle, bicycle
Finanzas: bank, insurance, credit, tax, pension
Comunicación: mail, newspaper, radio, podcast
Otros: other, settings, star, heart, calendar, clock, bell, search, user, group
```

### 4.2 Componentes

#### 4.2.1 `IconPickerModal` (frontend)
- **Responsabilidad**: Mostrar grilla de iconos con búsqueda y selección
- **Ubicación**: `frontend/src/components/IconPickerModal.tsx`
- **Props**: `onSelect(iconKey: string)`, `isOpen: boolean`, `onClose()`, `selectedIcon?: string`
- **Estado interno**: `searchQuery: string`, `filteredIcons: IconDefinition[]`

#### 4.2.2 `iconCatalog` (compartido)
- **Responsabilidad**: Definir catálogo completo de iconos con metadata
- **Ubicación**: `frontend/src/data/iconCatalog.ts` y `internal/services/icon_catalog.go`
- **Estructura**:
```typescript
interface IconDefinition {
  key: string;       // "trash", "internet", etc.
  label: string;     // "Basura", "Internet", etc.
  category: string;  // "utilidades", "hogar", etc.
}
```

### 4.3 Modelo de datos

No hay cambios en el modelo de datos. El campo `icon_key` en la tabla `services` ya existe y acepta strings. Solo se expande la validación.

### 4.4 APIs / Contratos

No hay cambios en los endpoints. El endpoint `POST /api/services` sigue aceptando `icon_key` como string, pero ahora validará contra un catálogo expandido.

**Endpoint existente**: `POST /api/services`

**Request** (sin cambios):
```json
{
  "home_id": 1,
  "name": "Recolección de Basura",
  "institution": "Alcaldía",
  "currency_id": 1,
  "frequency": "monthly",
  "suggested_amount": 20,
  "active": true,
  "icon_key": "trash"
}
```

**Nuevo endpoint opcional** (P2): `GET /api/icons`
```json
{
  "icons": [
    {"key": "trash", "label": "Basura", "category": "utilidades"},
    ...
  ]
}
```

### 4.5 Dependencias

- **Internas**: `internal/services/service.go` (expandir `AllowedIconKeys`), formulario de servicios frontend
- **Externas**: Ninguna nueva. Usar librería de iconos existente o SVG inline.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado que el usuario crea un servicio con `icon_key: "trash"`, cuando envía el formulario, entonces el servicio se crea exitosamente sin error de validación
- [ ] CA-002: Dado que el usuario abre el selector de iconos, cuando escribe "basu", entonces se filtra y muestra el icono "trash" (Basura)
- [ ] CA-003: Dado que el usuario hace clic en un icono del modal, entonces el modal se cierra y el icono seleccionado se muestra como preview en el formulario
- [ ] CA-004: Dado que el usuario envía un icono no válido (no en el catálogo), entonces recibe un error de validación descriptivo
- [ ] CA-005: El catálogo de iconos incluye al menos 50 iconos válidos

### 5.2 No funcionales

- [ ] CA-NF-001: El filtrado de iconos responde en menos de 50ms al escribir
- [ ] CA-NF-002: El componente modal agrega menos de 5KB al bundle JS
- [ ] CA-NF-003: No se agregan nuevas dependencias de npm

### 5.3 Testing

- **Unit tests**: Test de validación de iconos en `internal/services/service_test.go` con iconos válidos e inválidos
- **Integration tests**: Test de creación de servicio con diversos iconos del catálogo
- **E2E tests**: Flujo completo de creación de servicio con selección de icono via modal

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Expandir `AllowedIconKeys` en backend con ~80 iconos | 0.5 días | Ninguna |
| 2 | Crear `iconCatalog.ts` con catálogo completo en frontend | 0.5 días | Fase 1 |
| 3 | Crear componente `IconPickerModal` con búsqueda | 1 día | Fase 2 |
| 4 | Integrar modal en formulario de creación/edición de servicios | 0.5 días | Fase 3 |
| 5 | Tests y validación | 0.5 días | Fase 4 |

### 6.2 Milestones

1. **MVP**: Validación corregida + catálogo expandido (Fase 1-2)
2. **V1.0**: Modal con búsqueda integrado en formularios (Fase 3-5)

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Catálogo muy grande afecta rendimiento | Baja | Bajo | Limitar a 100 iconos, filtrado en memoria es O(n) trivial |
| Iconos no coinciden entre frontend y backend | Media | Alto | Generar catálogo desde una única fuente (script o constante compartida) |
| Librería de iconos no tiene todos los iconos | Media | Medio | Usar SVG inline como fallback para iconos faltantes |
| Modal no funciona bien en pantallas pequeñas | Baja | Medio | Diseño responsive con grilla adaptable |
| Errores de validación silenciosos (sin toast) | Alta | Alto | **Lección aprendida**: Siempre implementar toasts de error en formularios. Los errores silenciosos dificultan la depuración y frustran al usuario. |

## 8. Notas y Referencias

- Error original reportado: `curl` a `POST /api/services` con `icon_key: "delete"` retorna `{"error":"invalid_request","message":"el icono seleccionado no es válido"}`
- Código de validación: `internal/services/service.go:144-145`
- Catálogo actual: `internal/services/service.go:20-30` (solo 9 iconos)
- **Lección crítica**: Los errores de API deben mostrarse siempre como toasts visibles. No confiar en que el usuario revise la consola de red.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-15 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-08-15 | p40la-ihost-team | Agregada lección: toasts de error obligatorios en todos los formularios |
