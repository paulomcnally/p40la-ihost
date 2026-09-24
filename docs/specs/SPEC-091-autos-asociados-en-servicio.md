---
title: "Ver autos y pólizas asociadas en el detalle de un servicio"
id: "SPEC-091"
status: "in_progress"
author: "opencode"
created: "2026-09-24"
updated: "2026-09-24"
github_issue: 94
---

# Ver autos y pólizas asociadas en el detalle de un servicio

**ID**: SPEC-091  
**Estado**: in_progress  
**Autor**: opencode  
**Creado**: 2026-09-24  
**Actualizado**: 2026-09-24

---

## 1. Resumen Ejecutivo

La asociación auto ↔ servicio (póliza de seguro) vive únicamente en `auto_services` y hoy solo es visible desde el lado del auto: `AutoShowPage` lista las pólizas de cada auto agrupadas por institución. Desde el detalle de un servicio (`/services/bills/:id`, `BillsPage`) no hay ninguna vista de qué autos lo tienen asociado como seguro, lo cual deja el dato desconectado del servicio.

Este spec agrega la vista inversa: en `BillsPage` se podrá ver una pestaña "Pólizas" que lista los autos asociados al servicio, con los datos de póliza (`coverage_type`, `policy_number`, `certificate`, `insurer_number`) y navegación al detalle del auto. No se modifica el esquema de base de datos: la relación ya existe en `auto_services` y solo se expone en una nueva dirección. Es una feature de solo lectura, de bajo costo en memoria y sin dependencias nuevas, acorde a las restricciones del iHost.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Nuevo endpoint autenticado `GET /api/services/{id}/autos` que devuelve los autos asociados al servicio vía `auto_services`, con datos del auto y de la póliza.
2. **REQ-002**: En `BillsPage` se agrega una pestaña "Pólizas" que muestra una tarjeta por auto asociado: ícono, marca/modelo, placa y los campos de póliza (`coverage_type` como badge, `policy_number`, `certificate`, `insurer_number`), con link al detalle del auto (`/autos/:id`).
3. **REQ-003**: Claves i18n nuevas en `frontend/public/i18n/{es,en}.json` para el label de la pestaña y los textos de la vista (con fallback al no existir).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-004**: Estado vacío para la pestaña "Pólizas" cuando el servicio no tiene autos asociados (patrón EmptyCard: título, descripción, sin botón de acción ya que la asociación se gestiona desde el auto).
2. **REQ-005**: Tests unitarios para `AutoServiceStorage.ListByService` y el handler del nuevo endpoint.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-006**: Mostrar el badge de vigencia del servicio (Vencido/Activo) en cada tarjeta de auto, reutilizando la lógica de `AutoShowPage` (`end_date < hoy`).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Una sola consulta JOIN `auto_services → autos` por servicio; sin N+1. La respuesta es de pocos KB.
- **Seguridad**: Endpoint protegido por `authMiddleware` (sesión), igual que el resto de `/api/services/*`.
- **Almacenamiento**: Sin cambios de esquema ni migraciones.
- **Disponibilidad**: Solo lectura; no impacta el scheduler ni el webhook.
- **iHost**: Sin dependencias nuevas; consulta liviana sobre índices existentes (`auto_services.service_id` — verificar índice; la PK compuesta es `(auto_id, service_id)`, puede requerir índice en `service_id` para la query inversa).

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- `auto_services` guarda `auto_id`, `service_id`, `coverage_type`, `policy_number`, `certificate`, `insurer_number` (`migrations/0010_add_service_vigencia_and_auto_services.up.sql`).
- `AutoServiceStorage.ListByAuto` (`internal/storage/auto_service.go:22-76`) ya hace el JOIN auto→service con info de institución; se reutiliza su patrón pero invertido (service→auto).
- `AutoShowPage.tsx` (frontend) muestra las pólizas desde el lado del auto y define el patrón visual de las tarjetas de seguro (badges de coverage, Póliza/Certificado/Aseguradora, badge Vencido/Inactivo).
- `BillsPage.tsx` ya tiene el patrón de pestañas `analisis | facturas` con `useSearchParams` (`?tab=`); la nueva pestaña se integra al mismo mecanismo.
- Rutas de autos: `GET /api/autos/{id}/services` (`routes.go:120`), agregando el handler del reverse lookup en `AutoServiceHandlers`.
- Modelo `Auto` (`internal/models/auto.go`): `id, year, model, brand, color, icon, motor, chasis, vin, placa`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Pestaña "Pólizas" en `BillsPage` | Consistente con el patrón de tabs existente (`?tab=`), no sobrecarga el detalle | Un fetch extra al entrar | ✅ Seleccionada |
| Bloque/card fijo encima de las tabs | Visible siempre | Carga extra obligatoria en toda visita, desordena el header del detalle | ❌ Rechazada |
| Sección dentro de la pestaña "Análisis" | Menos clicks | Mezcla conceptos distintos (análisis financiero vs. pólizas) | ❌ Rechazada |
| Agregar `autos` al GET /api/services/{id} | Una sola llamada | Ensucia el contrato del servicio para un dato poco usado; exige tocar el modelo Service y sus consumers | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Endpoint dedicado en vez de inflar `GET /api/services/{id}`**
- **Contexto**: El detalle de servicio se usa en varias páginas; las pólizas son un dato de nicho.
- **Decisión**: Nuevo endpoint `GET /api/services/{id}/autos` manejado por `AutoServiceHandlers`, con storage `AutoServiceStorage.ListByService`.
- **Consecuencias**: Contrato estable, carga bajo demanda solo en la pestaña Pólizas. Se agrega índice en `auto_services.service_id` (migración) para el reverse lookup.

**ADR-002: La pestaña solo lista autos asociados a ESTE servicio**
- **Contexto**: Un auto puede tener varias pólizas (servicios); el reverse lookup es 1 servicio → N autos.
- **Decisión**: La pestaña muestra una tarjeta por `auto_services` row cuyo `service_id` es el del servicio actual, con los datos de póliza de esa fila.
- **Consecuencias**: Semántica clara; cada tarjeta refleja exactamente la asociación `auto_services` de este servicio.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Frontend BillsPage ?tab=polizas] --GET /api/services/{id}/autos--> [AutoServiceHandlers]
                                                                        |
                                                                        v
                                                              [AutoServiceService.ListByService]
                                                                        |
                                                                        v
                                                             [AutoServiceStorage.ListByService]
                                                                        |
                                                                        v
                                                            [SQLite: auto_services JOIN autos]
```

### 4.2 Componentes

#### 4.2.1 `AutoServiceStorage.ListByService`
- **Responsabilidad**: Consultar `auto_services` de un servicio con datos del auto.
- **Interfaz**: `ListByService(ctx, serviceID int64) ([]models.ServiceAutoDetail, error)`
- **Dependencias**: `*sql.DB`.
- **Ubicación**: `internal/storage/auto_service.go`.

#### 4.2.2 `AutoServiceService.ListByService`
- **Responsabilidad**: Orquestar la consulta y devolver el detalle.
- **Interfaz**: `ListByService(ctx, serviceID int64) ([]models.ServiceAutoDetail, error)`
- **Ubicación**: `internal/services/auto_service.go`.

#### 4.2.3 `ServiceAutoDetail` (modelo nuevo)
- **Responsabilidad**: DTO de la API: datos del auto + póliza.
- **Ubicación**: `internal/models/auto_service.go`.

#### 4.2.4 Handler `ListServiceAutos`
- **Responsabilidad**: Parsear `{id}` y responder JSON.
- **Ruta**: `GET /api/services/{id}/autos` (con `authMiddleware`).
- **Ubicación**: `internal/api/auto_service_handlers.go`.

#### 4.2.5 Pestaña "Pólizas" en `BillsPage`
- **Responsabilidad**: Mostrar autos asociados y navegar al detalle del auto.
- **Ubicación**: `frontend/src/pages/BillsPage.tsx` (+ `frontend/src/types/index.ts`, `frontend/src/api/index.ts`).

### 4.3 Modelo de datos

```
Entidad: ServiceAutoDetail (DTO, sin tabla nueva)
- auto_id: int64
- brand, model, color, icon, placa: string (de autos)
- year: int64
- coverage_type: string
- policy_number: string
- certificate: *string
- insurer_number: string
- created_at: string (de auto_services)
- Relaciones: auto_services (N:1) — se consulta desde service_id
```

No hay cambios de esquema. **Migración `NNNN_add_index_auto_services_service_id.up.sql`**: índice en `auto_services.service_id` (la PK actual es `(auto_id, service_id)`; el reverse lookup filtra por `service_id`).

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/services/{id}/autos`

**Request**: sin body. `{id}` = id del servicio.

**Response 200**:
```json
[
  {
    "auto_id": 1,
    "brand": "Toyota",
    "model": "Hilux",
    "year": 2026,
    "color": "Blanco",
    "icon": "car",
    "placa": "P123ABC",
    "coverage_type": "full_cover",
    "policy_number": "02B 128265",
    "certificate": "1",
    "insurer_number": "1800 9911",
    "created_at": "2026-09-10T19:48:14Z"
  }
]
```

**Response 200 vacío**: `[]`

**Response Error**:
```json
{
  "error": "not_found",
  "message": "servicio no encontrado"
}
```

### 4.5 Dependencias

- **Internas**: `AutoServiceStorage`, `AutoServiceService`, `AutoServiceHandlers` (ruta nueva), `BillsPage` + `api/index.ts` + `types/index.ts` + i18n.
- **Externas**: Ninguna.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: `GET /api/services/{id}/autos` (autenticado) devuelve los autos asociados al servicio con datos de póliza; `[]` si no hay asociaciones.
- [ ] CA-002: En `BillsPage`, la pestaña "Pólizas" lista una tarjeta por auto con ícono, marca/modelo, placa, badge de cobertura, número de póliza, certificado y aseguradora.
- [ ] CA-003: Clic en una tarjeta navega a `/autos/:id` (detalle del auto).
- [ ] CA-004: Sin autos asociados se muestra el estado vacío (EmptyCard).
- [ ] CA-005: El endpoint responde 401 sin sesión.
- [ ] CA-DARK: Las tarjetas usan tokens del tema (`bg-card`, `text-text`, `text-text-secondary`); legibles en darkmode.

### 5.2 No funcionales

- [ ] CA-NF-001: La query es un único JOIN sin N+1.
- [ ] CA-NF-002: Sin dependencias nuevas ni cambios de esquema (solo índice).

### 5.3 Testing

- **Unit tests**: `AutoServiceStorage.ListByService` (con autos asociados y sin asociaciones); handler del endpoint (200, 401, id inválido).
- **Integration tests**: Flujo completo con `./scripts/create-test-user.sh` + `curl` a `/api/services/{id}/autos`.
- **E2E tests**: Manual — abrir servicio con pólizas, cambiar a pestaña Pólizas, navegar al auto.
- **Carga/Performance**: Una query por carga de la pestaña; verificable con log de SQL.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración índice en `auto_services.service_id` | 0.5 h | Ninguna |
| 2 | Backend: modelo `ServiceAutoDetail`, `ListByService` (storage + service), handler + ruta | 2 h | Fase 1 |
| 3 | Tests backend | 1 h | Fase 2 |
| 4 | Frontend: type, api method, pestaña "Pólizas" en `BillsPage`, estado vacío | 2 h | Fase 2 |
| 5 | i18n es/en + build | 0.5 h | Fase 4 |
| 6 | Validación local + pruebas manuales con el usuario | 1 h | Fases 1-5 |

### 6.2 Milestones

1. **MVP**: Endpoint + pestaña "Pólizas" funcional con navegación al auto.
2. **V1.0**: Estado vacío, tests, i18n y badge de vigencia (REQ-006).

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Query inversa lenta sin índice en `service_id` | Media | Bajo | Migración de índice antes de la feature |
| Cambio de contrato en `api.services.get` | Baja | Medio | Endpoint dedicado; no se toca el contrato existente |
| Overlap con SPEC de "Renovar servicio" (discutida con usuario) | Media | Bajo | El reverse lookup es independiente; "Renovar" actualiza fechas, no la asociación |
| Carga extra en `BillsPage` | Baja | Bajo | Fetch solo cuando se activa la pestaña Pólizas |

## 8. Notas y Referencias

- Vista actual desde el lado del auto: `frontend/src/pages/AutoShowPage.tsx`.
- Reverse lookup pattern a imitar: `internal/storage/auto_service.go:22-76` (`ListByAuto`).
- Pestañas existentes en `BillsPage`: `analisis | facturas` (`frontend/src/pages/BillsPage.tsx:21-35`).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-24 | opencode | Creación inicial de la especificación |