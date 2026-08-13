# 📋 Especificaciones Técnicas

Este directorio contiene todas las especificaciones técnicas del proyecto `p40la-ihost`.

## 📊 Resumen

| Métrica | Valor |
|---------|-------|
| **Total de specs** | 4 |
| **En draft** | 0 🟡 |
| **Pending execution** | 0 🔵 |
| **In progress** | 1 🟣 |
| **Pending release** | 0 🟠 |
| **Released** | 3 🟢 |
| **Canceladas** | 0 ⚫ |
| **Último ID usado** | SPEC-004 |

---

## 📑 Índice de Specifications

| ID | Título | Estado | Fecha Creación | Autor |
|----|--------|--------|----------------|-------|
| SPEC-001 | Infraestructura de Despliegue en SONOFF iHost: Docker, Volumen de Datos y Multi-Arquitectura | released | 2026-08-12 | p40la-ihost-team |
| SPEC-002 | Configuración de usuario y login | released | 2026-08-12 | p40la-ihost-team |
| SPEC-003 | Cambio de puerto por defecto a 8088 | released | 2026-08-12 | p40la-ihost-team |
| SPEC-004 | Dashboard con Sidebar, Header, i18n, Settings estilo iOS, Iconos, Home/Casa y Módulo de Servicios con Facturas | in_progress | 2026-08-12 | p40la-ihost-team |

---

## 🔄 Flujo de Estados

```
     ┌─────────┐     ┌─────────────────┐     ┌───────────┐     ┌───────────────┐     ┌──────────┐
     │  draft  │────▶│ pending_execution │────▶│ in_progress│────▶│pending_release│────▶│ released │
     └─────────┘     └─────────────────┘     └───────────┘     └───────────────┘     └──────────┘
          │                  │                      │                  │                │
          ▼                  ▼                      ▼                  ▼                ▼
     ┌─────────┐       ┌─────────┐            ┌─────────────┐      ┌───────────┐      ┌──────────┐
     │cancelled│       │cancelled│            │pending_execution│  │in_progress│      │in_progress│
     └─────────┘       └─────────┘            └─────────────┘      └───────────┘      └──────────┘
```

### Leyenda de Estados

| Estado | Emoji | Descripción |
|--------|-------|-------------|
| `draft` | 🟡 | En proceso de redacción/investigación |
| `pending_execution` | 🔵 | Lista para desarrollo, no iniciada |
| `in_progress` | 🟣 | Actualmente en desarrollo |
| `pending_release` | 🟠 | Desarrollo completo, lista para staging/release |
| `released` | 🟢 | Subida a iHost o producción |
| `cancelled` | ⚫ | Cancelada o descartada |

---

## 📝 Cómo usar este sistema

### Crear una nueva spec

Usa la skill `spec-manager`:

```
/spec create "Título de la nueva funcionalidad"
```

O pide al asistente: *"Crear spec para [requerimiento]"*

### Cambiar estado de una spec

```
/spec status SPEC-001 pending_execution
```

O pide al asistente: *"Cambiar SPEC-001 a in_progress"*

### Ver detalle de una spec

```
/spec show SPEC-001
```

---

## 📁 Estructura del directorio

```
docs/specs/
├── README.md              # Este archivo - tracking y contador
├── templates/
│   └── spec-template.md   # Template para nuevas specs
└── SPEC-XXX-*.md         # Archivos de specifications
```

---

*Última actualización de este tracker: 2026-08-12 — SPEC-003 en estado `released` (merge a `main`, tag `v0.1.1`, imagen publicada).*
