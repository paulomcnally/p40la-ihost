# 📋 Especificaciones Técnicas

Este directorio contiene todas las especificaciones técnicas del proyecto `p40la-ihost`.

## 📊 Resumen

| Métrica | Valor |
|---------|-------|
| **Total de specs** | 34 |
| **En draft** | 0 🟡 |
| **Pending execution** | 0 🔵 |
| **In progress** | 1 🟣 |
| **Pending release** | 0 🟠 |
| **Released** | 33 🟢 |
| **Canceladas** | 0 ⚫ |
| **Último ID usado** | SPEC-034 |

---

## 📑 Índice de Specifications

| ID | Título | Estado | Fecha Creación | Autor |
|----|--------|--------|----------------|-------|
| SPEC-001 | Infraestructura de Despliegue en SONOFF iHost: Docker, Volumen de Datos y Multi-Arquitectura | released | 2026-08-12 | p40la-ihost-team |
| SPEC-002 | Configuración de usuario y login | released | 2026-08-12 | p40la-ihost-team |
| SPEC-003 | Cambio de puerto por defecto a 8088 | released | 2026-08-12 | p40la-ihost-team |
| SPEC-004 | Dashboard con Sidebar, Header, i18n, Settings estilo iOS, Iconos, Home/Casa y Módulo de Servicios con Facturas | released | 2026-08-12 | p40la-ihost-team |
| SPEC-005 | Migración del Frontend a React + Tailwind CSS | released | 2026-08-13 | p40la-ihost-team |
| SPEC-006 | Corrección de validación de iconos y selector modal con búsqueda para servicios | released | 2026-08-15 | p40la-ihost-team |
| SPEC-007 | Formulario de facturas: ocultar mes para servicios anuales | released | 2026-08-15 | p40la-ihost-team |
| SPEC-008 | Sistema de facturación automática: monto fijo/variable y generación programada | released | 2026-08-15 | p40la-ihost-team |
| SPEC-009 | Módulo de Instituciones, Analizadores de Documentos y Extracción Automática de Facturas | released | 2026-08-15 | p40la-ihost-team |
| SPEC-010 | Integración de Specs con GitHub Issues para Gestión Asíncrona y Multi-Agente | released | 2026-08-15 | paulomcnally |
| SPEC-011 | Soporte Responsive para Dispositivos Móviles | released | 2026-08-15 | paulomcnally |
| SPEC-012 | Analizador de facturas Claro: internet residencial e internet móvil | released | 2026-08-15 | paulomcnally |
| SPEC-013 | Mejora de UI para gestión de analizadores en instituciones | released | 2026-08-15 | paulomcnally |
| SPEC-014 | UI de subida de facturas con análisis automático + fix overflow BillsPage | released | 2026-08-15 | paulomcnally |
| SPEC-015 | CI/CD con GitHub Actions: build multi-arch por tag | in_progress | 2026-08-15 | paulomcnally |
| SPEC-016 | Permitir borrar el campo día de facturación con validación por toast | released | 2026-08-15 | paulomcnally |
| SPEC-017 | Responsive BillsPage: tabla en desktop, cards en móvil | released | 2026-08-15 | p40la-ihost-team |
| SPEC-018 | Estado de pago dinámico en cards de servicios | released | 2026-08-15 | paulomcnally |
| SPEC-019 | Analizador de facturas DISNORTE-DISSUR | released | 2026-08-15 | paulomcnally |
| SPEC-020 | Fix posición del label de estado en cards de Bills | released | 2026-08-15 | p40la-ihost-team |
| SPEC-021 | Sobrescribir bill existente cuando el analizador extrae datos | released | 2026-08-15 | p40la-ihost-team |
| SPEC-022 | Loading states en listas de casas y servicios | released | 2026-08-15 | p40la-ihost-team |
| SPEC-023 | Modal del analizador: deshabilitar cierre al hacer click fuera | released | 2026-08-15 | p40la-ihost-team |
| SPEC-024 | Módulo CRUD de Autos | released | 2026-08-16 | p40la-ihost-team |
| SPEC-025 | Página Show de Autos con Seguros + Vigencia y Estado en Servicios | released | 2026-08-16 | p40la-ihost-team |
| SPEC-026 | Categorías de Instituciones con Seed y Filtro de Seguros | released | 2026-08-16 | paulomcnally |
| SPEC-027 | Script de release automático para Docker Hub con bump de versión | released | 2026-08-16 | paulomcnally |
| SPEC-028 | Extensión de campos para autos y pólizas de seguro | released | 2026-08-16 | p40la-ihost-team |
| SPEC-029 | Sistema de Envío de Mails y Alertas Diarias de Seguros Vencidos | released | 2026-08-16 | paulomcnally |
| SPEC-030 | Email informativo al generar factura automática | released | 2026-08-16 | paulomcnally |
| SPEC-031 | Resumen diario de facturas pendientes por email | released | 2026-08-16 | paulomcnally |
| SPEC-032 | Sección Alertas en Configuraciones con toggles por funcionalidad de mail | released | 2026-08-16 | paulomcnally |
| SPEC-033 | Alertas multicanal con Voice Monkey (Alexa): sistema robusto de alertas con formato mail y voz | released | 2026-08-16 | paulomcnally |
| SPEC-034 | Estado Configurado y botón Reconfigurar para SMTP (mirror UI de Voice Monkey) | released | 2026-08-16 | paulomcnally |

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

*Última actualización de este tracker: 2026-08-16 — SPEC-034 released (commit `b25c618`).*
