# 📋 Especificaciones Técnicas

Este directorio contiene todas las especificaciones técnicas del proyecto `p40la-ihost`.

## 📊 Resumen

| Métrica | Valor |
|---------|-------|
| **Total de specs** | 61 |
| **En draft** | 0 🟡 |
| **Pending execution** | 0 🔵 |
| **In progress** | 1 🟣 |
| **Pending release** | 0 🟠 |
| **Released** | 60 🟢 |
| **Canceladas** | 0 ⚫ |
| **Último ID usado** | SPEC-061 |

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
| SPEC-015 | CI/CD con GitHub Actions: build multi-arch por tag | released | 2026-08-15 | paulomcnally |
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
| SPEC-035 | Destinatarios con modal: alta/baja de emails con guardado automático | released | 2026-08-16 | paulomcnally |
| SPEC-036 | Sección de ayuda/info en configuración SMTP y Voice Monkey (Alexa) | released | 2026-08-16 | paulomcnally |
| SPEC-037 | Secciones colapsables de Email/Voice Monkey y gating de toggles de email | released | 2026-08-16 | paulomcnally |
| SPEC-038 | Limpieza automática de worktrees al liberar una spec | released | 2026-08-16 | paulomcnally |
| SPEC-039 | Dropdowns de UI para horas (generación y check de alertas) con formato AM/PM y 24h | released | 2026-08-16 | p40la-ihost-team |
| SPEC-040 | Analizador de recibos ASSA - Seguro Auto | released | 2026-08-16 | paulomcnally |
| SPEC-041 | Subida múltiple de facturas para importación masiva de pagos | released | 2026-08-16 | paulomcnally |
| SPEC-042 | Fix error NULL en file_hash al escanear facturas con registros existentes en iHost | released | 2026-08-17 | paulomcnally |
| SPEC-043 | Acción Pagar en facturas con fecha de pago, comprobante y referencia | released | 2026-08-31 | p40la-ihost-team |
| SPEC-044 | Menú Pensión Alimenticia en el Sidebar con submenús y páginas en blanco | released | 2026-09-02 | p40la-ihost-team |
| SPEC-045 | CRUD de Hijos en módulo Pensión Alimenticia | released | 2026-09-02 | p40la-ihost-team |
| SPEC-046 | CRUD de Notificaciones en módulo Pensión Alimenticia | released | 2026-09-02 | p40la-ihost-team |
| SPEC-047 | CRUD de Salarios en módulo Pensión Alimenticia | released | 2026-09-02 | p40la-ihost-team |
| SPEC-048 | CRUD de Categorías en módulo Pensión Alimenticia | released | 2026-09-02 | p40la-ihost-team |
| SPEC-049 | Backend de Registros Mensuales de Pensión Alimenticia (support_records, salary_payments, month_closings) | released | 2026-09-02 | p40la-ihost-team |
| SPEC-050 | Página de Registros Mensuales de Pensión Alimenticia (frontend, replicar child-support/records de P4OLA) | released | 2026-09-02 | p40la-ihost-team |
| SPEC-051 | Emails y Generación Mensual de Registros de Pensión Alimenticia | released | 2026-09-02 | p40la-ihost-team |
| SPEC-052 | Fix deprecación Node.js 20 en GitHub Actions del build Docker | released | 2026-09-03 | p40la-ihost-team |
| SPEC-053 | Modo oscuro configurable con default activo | released | 2026-09-03 | p40la-ihost-team |
| SPEC-054 | Módulo Deudas: CRUD con generación automática de cuotas y vista Calendario | released | 2026-09-04 | p40la-ihost-team |
| SPEC-055 | Análisis de Deudas por Mes con Gráficos en la página de Deudas | released | 2026-09-04 | p40la-ihost-team |
| SPEC-056 | Build nativo ARM64 con runner ubuntu-24.04-arm en GitHub Actions | released | 2026-09-04 | p40la-ihost-team |
| SPEC-057 | Fix iconos invisibles en iOS Safari (iPhone) | released | 2026-09-04 | paulomcnally |
| SPEC-058 | Formato de moneda configurable con default Nicaragua (1,000.00) | released | 2026-09-04 | paulomcnally |
| SPEC-059 | Scripts para tareas recurrentes del agente (test-user y check-server) para reducir tokens del prompt | released | 2026-09-04 | p40la-ihost-team |
| SPEC-060 | Fix global de texto de inputs en darkmode en todos los formularios + regla para futuras specs | released | 2026-09-04 | paulomcnally |
| SPEC-061 | Reordenar casas con drag & drop en la página de Casas | in_progress | 2026-09-04 | paulomcnally |

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

*Última actualización de este tracker: 2026-09-04 — SPEC-061 creada en draft (drag & drop para reordenar casas).*
