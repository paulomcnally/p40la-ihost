---
title: "Configuración de usuario y login"
id: "SPEC-002"
status: "pending_release"
author: "p40la-ihost-team"
created: "2026-08-12"
updated: "2026-08-12"
---

# Configuración de usuario y login

**ID**: SPEC-002  
**Estado**: draft  
**Autor**: p40la-ihost-team  
**Creado**: 2026-08-12  
**Actualizado**: 2026-08-12

---

## 1. Resumen Ejecutivo

El sistema actualmente no cuenta con autenticación ni control de acceso. Este spec define el flujo mínimo necesario para proteger el acceso al dashboard: un wizard de configuración inicial que se ejecuta la primera vez que la aplicación carga y que permite crear un único usuario administrador con email y contraseña. A partir de esa primera configuración, toda carga posterior exigirá iniciar sesión para acceder al dashboard.

El alcance se mantiene deliberadamente pequeño. El dashboard resultante no contendrá funcionalidad adicional en esta iteración; solo debe estar protegido y ser accesible tras una autenticación válida. Todas las decisiones técnicas priorizan el entorno objetivo: un iHost con poca RAM, poco almacenamiento y preferencia por dependencias mínimas.

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Al iniciar la aplicación sin usuarios registrados en SQLite, la página raíz (`/`) debe redirigir al wizard de configuración inicial (`/setup`).
2. **REQ-002**: El wizard debe solicitar únicamente: email, contraseña y confirmación de contraseña.
3. **REQ-003**: Validar en backend y frontend que el email tenga formato válido y que la contraseña tenga al menos 8 caracteres, incluyendo al menos una letra y un número.
4. **REQ-004**: Almacenar la contraseña hasheada con bcrypt; nunca debe persistirse en texto plano.
5. **REQ-005**: El sistema solo admite un único usuario. Si ya existe un usuario, cualquier intento de crear otro desde el wizard debe ser rechazado con error `409 Conflict`.
6. **REQ-006**: En cargas posteriores, si no existe sesión válida, la raíz (`/`) debe redirigir a la pantalla de login (`/login`).
7. **REQ-007**: El login debe recibir email y contraseña, verificar el hash con bcrypt y responder de manera genérica ante credenciales inválidas para no revelar existencia del usuario.
8. **REQ-008**: Tras un login exitoso o la finalización del wizard, el backend debe establecer una cookie de sesión firmada con HMAC-SHA256, con banderas `HttpOnly`, `SameSite=Strict` y `Secure` cuando el entorno use HTTPS.
9. **REQ-009**: El dashboard (`/dashboard`) solo debe ser accesible cuando la cookie de sesión sea válida; de lo contrario se redirige a `/login`.
10. **REQ-010**: Debe existir un endpoint para cerrar sesión (`POST /api/logout`) que invalide/limpie la cookie y redirija al login.
11. **REQ-011**: La página raíz (`/`) debe actuar como router de entrada: sin usuarios → `/setup`; sin sesión → `/login`; con sesión → `/dashboard`.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-012**: Si no se configura una clave secreta para firmar cookies mediante variable de entorno, el backend debe generar una aleatoriamente en el primer arranque y persistirla en la tabla `settings` de SQLite.
2. **REQ-013**: Los mensajes de error mostrados al usuario deben ser claros pero no filtrar información sensible (por ejemplo, no distinguir entre email inexistente y contraseña incorrecta).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-014**: Opción "Recordarme" en el login que extienda la duración de la sesión de "hasta cerrar navegador" a 30 días.
2. **REQ-015**: Rate limiting básico por dirección IP en los endpoints `/api/setup` y `/api/login` para mitigar fuerza bruta.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: El endpoint de login debe responder en menos de 200 ms en iHost con el costo por defecto de bcrypt. El costo debe ser configurable para ajustar seguridad vs. CPU.
- **Seguridad**: Contraseñas hasheadas con bcrypt; comparación de hashes en tiempo constante delegada a `bcrypt.CompareHashAndPassword`; sesiones basadas en cookies firmadas, no en almacenamiento server-side.
- **Almacenamiento**: Tabla `users` con índice único en `email`; tabla `settings` para la clave de firma. Sin archivos de sesión ni caché adicional.
- **Disponibilidad**: El endpoint `/health` sigue respondiendo `200 OK` independientemente del estado de autenticación.
- **iHost**: Dependencia externa única recomendada: `golang.org/x/crypto/bcrypt`. Sin frameworks web ni de sesiones; sesión implementada con la stdlib de Go (`crypto/hmac`, `crypto/sha256`, `encoding/base64`).

---

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se evaluaron alternativas para:
- Hashing de contraseñas.
- Gestión de sesiones sin estado server-side.
- Protección del frontend estático sin frameworks SPA.

Se consultaron las siguientes referencias:
- OWASP Password Storage Cheat Sheet: recomienda bcrypt como opción segura y probada para hashing de contraseñas.
- Documentación de `golang.org/x/crypto/bcrypt`: algoritmo adaptativo con coste configurable.
- Documentación de `net/http` de Go y `crypto/hmac` de stdlib: suficientes para implementar cookies firmadas sin librerías adicionales.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| bcrypt (`golang.org/x/crypto/bcrypt`) | Estándar de facto, coste adaptable, protección contra brute-force por diseño | Requiere una dependencia externa; coste elevado consume CPU en iHost | ✅ Seleccionada |
| scrypt (stdlib `golang.org/x/crypto/scrypt`) | Resistente a hardware especializado | Requiere ajuste de parámetros; más complejo de configurar de forma segura | ❌ Rechazada |
| argon2 (`golang.org/x/crypto/argon2`) | Ganadora de Password Hashing Competition | Mayor consumo de memoria; no justificado para un único usuario en iHost | ❌ Rechazada |
| JWT para sesiones | Sin estado server-side | Requiere librería o implementación propia; tokens pueden crecer en tamaño | ❌ Rechazada |
| Cookie firmada con HMAC-SHA256 (stdlib) | Sin dependencias extra; ligera; fácil de invalidar cambiando la clave | No permite revocación individual sin rotar clave global | ✅ Seleccionada |
| Sesiones server-side en SQLite | Revocación individual sencilla | Más queries por request; mayor complejidad; no necesario para un único usuario | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Uso de bcrypt con coste configurable
- **Contexto**: Se necesita hash seguro de contraseñas sin exceder la CPU del iHost.
- **Decisión**: Usar `golang.org/x/crypto/bcrypt` con coste por defecto 10 y variable de entorno `BCRYPT_COST` para ajustarlo.
- **Consecuencias**: Seguridad adecuada; posible latencia en login si el coste se incrementa. Se debe medir en iHost antes de subir de 10.

**ADR-002**: Sesiones mediante cookie firmada con HMAC-SHA256
- **Contexto**: Se requiere autenticación stateless en memoria para minimizar el uso de RAM.
- **Decisión**: Implementar cookie con valor `base64(payload|timestamp|hmac)` usando `crypto/hmac` + `crypto/sha256`. El payload contiene el email del usuario y la expiración. La clave secreta se almacena en `settings` si no viene por entorno.
- **Consecuencias**: Bajo consumo de memoria; logout se implementa limpiando la cookie del cliente; rotación de clave invalida todas las sesiones, lo cual es aceptable para un único usuario.

**ADR-003**: Wizard como puerta de entrada única
- **Contexto**: No hay interfaz de administración de usuarios en este alcance.
- **Decisión**: Si la tabla `users` está vacía, cualquier visita a `/` redirige a `/setup`. Una vez creado el usuario, `/setup` deja de estar disponible.
- **Consecuencias**: Simplifica la lógica; limita el sistema a un único administrador. Futuros specs pueden agregar gestión de usuarios.

---

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
┌─────────────┐      GET /           ┌─────────────────┐
│  Navegador  │ ───────────────────▶ │  Backend (Go)   │
└─────────────┘                      └────────┬────────┘
       │                                      │
       │                                      ▼
       │                            ¿existen usuarios?
       │                                  /    \
       │                                Sí     No
       │                                /        \
       │                    ¿sesión válida?      redirect /setup
       │                         /    \
       │                       Sí      No
       │                       /          \
       │              serve /dashboard   redirect /login
       │
       │ POST /api/setup o /api/login
       │ ─────────────────────────────▶ AuthService ──▶ UserStorage (SQLite)
       │                                       │
       │                                       ▼
       │                              bcrypt hash/verify
       │                                       │
       │ ◀──────────────────────── Set-Cookie (firmada)
```

### 4.2 Componentes

#### 4.2.1 `internal/services/auth.go`
- **Responsabilidad**: Lógica de negocio de autenticación: crear primer usuario, validar credenciales, crear/destruir sesiones, firmar/verificar cookies.
- **Interfaz**: `CreateFirstUser(email, password, passwordConfirm) (*User, error)`, `Login(email, password) (*User, error)`, `CreateSession(user) (*http.Cookie, error)`, `ValidateSession(cookie) (*User, error)`, `Logout() *http.Cookie`.
- **Dependencias**: `UserStorage`, `SettingsStorage`, configuración de bcrypt y secret.
- **Ubicación**: `internal/services/auth.go`.

#### 4.2.2 `internal/storage/user.go`
- **Responsabilidad**: Acceso a datos de la tabla `users`.
- **Interfaz**: `Create(email, passwordHash)`, `GetByEmail(email)`, `Count()`, `Exists(email)`.
- **Dependencias**: `*sql.DB`.
- **Ubicación**: `internal/storage/user.go`.

#### 4.2.3 `internal/storage/settings.go`
- **Responsabilidad**: Leer y escribir pares clave/valor en la tabla `settings`, especialmente la clave secreta para firmar cookies.
- **Interfaz**: `Get(key)`, `Set(key, value)`.
- **Dependencias**: `*sql.DB`.
- **Ubicación**: `internal/storage/settings.go`.

#### 4.2.4 `internal/api/auth_handlers.go`
- **Responsabilidad**: Handlers HTTP para `/api/setup-status`, `/api/setup`, `/api/login`, `/api/logout` y `/api/me`.
- **Interfaz**: Funciones `http.HandlerFunc` que parsean JSON, llaman a `AuthService` y escriben respuestas JSON.
- **Dependencias**: `AuthService`.
- **Ubicación**: `internal/api/auth_handlers.go`.

#### 4.2.5 `internal/api/middleware.go`
- **Responsabilidad**: Middleware de autenticación que valida la cookie de sesión y, en su defecto, redirige a `/login` o `/setup` según corresponda.
- **Interfaz**: `AuthMiddleware(next http.Handler) http.Handler`.
- **Dependencias**: `AuthService`.
- **Ubicación**: `internal/api/middleware.go`.

#### 4.2.6 Frontend estático
- **Responsabilidad**: Páginas HTML, CSS y JS vanilla para el wizard, login y dashboard.
- **Archivos**: `public/setup.html`, `public/login.html`, `public/dashboard.html`, `public/js/auth.js`, `public/css/auth.css`.
- **Dependencias**: Ninguna; solo `fetch` nativo.

### 4.3 Modelo de datos

```
Entidad: users
- id: INTEGER PRIMARY KEY AUTOINCREMENT
- email: TEXT UNIQUE NOT NULL
- password_hash: TEXT NOT NULL
- created_at: DATETIME DEFAULT CURRENT_TIMESTAMP
- updated_at: DATETIME DEFAULT CURRENT_TIMESTAMP

Entidad: settings
- key: TEXT PRIMARY KEY
- value: TEXT NOT NULL

Relaciones:
- settings (1) almacena clave global de firma; no relacionada directamente con users.
```

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/setup-status`

**Response 200**:
```json
{
  "setup_completed": false
}
```

#### Endpoint: `POST /api/setup`

**Request**:
```json
{
  "email": "admin@example.com",
  "password": "miPassword123",
  "password_confirm": "miPassword123"
}
```

**Response 201**:
```json
{
  "user_id": 1,
  "email": "admin@example.com"
}
```
Además se envía la cookie de sesión `session`.

**Response 400** (validación fallida):
```json
{
  "error": "invalid_request",
  "message": "La contraseña debe tener al menos 8 caracteres, una letra y un número"
}
```

**Response 409** (ya existe usuario):
```json
{
  "error": "already_setup",
  "message": "El sistema ya fue configurado"
}
```

#### Endpoint: `POST /api/login`

**Request**:
```json
{
  "email": "admin@example.com",
  "password": "miPassword123"
}
```

**Response 200**:
```json
{
  "email": "admin@example.com"
}
```
Con cookie `session`.

**Response 401** (credenciales inválidas):
```json
{
  "error": "invalid_credentials",
  "message": "Email o contraseña incorrectos"
}
```

#### Endpoint: `POST /api/logout`

**Response 200**:
```json
{
  "message": "Sesión cerrada"
}
```
Con cookie `session` expirada.

#### Endpoint: `GET /api/me`

**Response 200**:
```json
{
  "email": "admin@example.com"
}
```

**Response 401**:
```json
{
  "error": "unauthorized",
  "message": "Sesión no válida"
}
```

### 4.5 Dependencias

- **Internas**: Capas `internal/models`, `internal/storage`, `internal/services`, `internal/api`, `internal/config`, `internal/db`.
- **Externas**:
  - `golang.org/x/crypto/bcrypt` para hashing de contraseñas.
  - Posiblemente `golang.org/x/exp/rand` si se requiere generación criptográfica de secret (preferir `crypto/rand` de stdlib).

---

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] **CA-001**: Dado que la base de datos no tiene usuarios, cuando se carga `/`, entonces se redirige a `/setup`.
- [ ] **CA-002**: Dado el wizard de configuración, cuando se envían email y contraseña válidos, entonces se crea el usuario, se establece la sesión y se redirige al dashboard.
- [ ] **CA-003**: Dado que ya existe un usuario, cuando se intenta acceder a `/setup` o POST a `/api/setup`, entonces se devuelve error 409.
- [ ] **CA-004**: Dado que el usuario no ha iniciado sesión, cuando carga `/`, entonces se redirige a `/login`.
- [ ] **CA-005**: Dado el formulario de login, cuando se ingresan credenciales válidas, entonces se establece la sesión y se redirige al dashboard.
- [ ] **CA-006**: Dado el formulario de login, cuando se ingresan credenciales inválidas, entonces se muestra un mensaje genérico sin revelar si el email existe.
- [ ] **CA-007**: Dado una sesión activa, cuando se accede a `/dashboard`, entonces se muestra la página del dashboard.
- [ ] **CA-008**: Dado una sesión activa, cuando se hace clic en cerrar sesión, entonces la cookie se elimina y se redirige a `/login`.
- [ ] **CA-009**: Dado que no hay sesión, cuando se intenta acceder a `/dashboard` directamente, entonces se redirige a `/login`.

### 5.2 No funcionales

- [ ] **CA-NF-001**: La contraseña nunca se almacena ni transmite en texto plano en el servidor; solo se mantiene el hash bcrypt.
- [ ] **CA-NF-002**: La cookie de sesión incluye las banderas `HttpOnly`, `SameSite=Strict` y `Secure` bajo HTTPS.
- [ ] **CA-NF-003**: El login responde en menos de 200 ms en el entorno de desarrollo local y se mide en iHost antes del release.
- [ ] **CA-NF-004**: No se agregan frameworks web; la sesión se implementa con stdlib de Go.

### 5.3 Testing

- **Unit tests**: Funciones de validación de email/contraseña; `hashPassword` y `verifyPassword`; firma y verificación de cookie.
- **Integration tests**: Flujo completo wizard → login → acceso dashboard → logout usando base de datos SQLite en memoria.
- **E2E manual**: Ejecutar el servidor localmente, limpiar la base de datos, verificar redirecciones y mensajes de error en navegador.
- **Carga/Performance**: Medir tiempo de respuesta de `/api/login` en iHost con al menos 10 intentos consecutivos.

---

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Crear migraciones SQL para `users` y `settings` | 0.5 días | Ninguna |
| 2 | Implementar modelos y storage (`users`, `settings`) | 1 día | Fase 1 |
| 3 | Implementar `AuthService` con bcrypt y cookies firmadas | 1 día | Fase 2 |
| 4 | Implementar handlers y middleware de autenticación | 1 día | Fase 3 |
| 5 | Crear frontend vanilla: `setup.html`, `login.html`, `dashboard.html` | 1 día | Fase 4 |
| 6 | Escribir tests unitarios e integración | 1 día | Fases 2-4 |
| 7 | Validación local (build, tests, ejecución manual) | 0.5 días | Fases 1-6 |

### 6.2 Milestones

1. **MVP**: Wizard funcional, login funcional, dashboard vacío protegido, tests pasan en local.
2. **V1.0**: Incluye rate limiting básico, opción "Recordarme" y ajuste de costo bcrypt tras medición en iHost.

---

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| bcrypt con coste 10 genera latencia alta en iHost | Media | Medio | Hacer el coste configurable (`BCRYPT_COST`) y medir en iHost; bajar a 9 si es necesario |
| Generación de clave secreta débil | Baja | Alto | Usar `crypto/rand` de stdlib para generar 32 bytes; almacenar en SQLite |
| Rotación de clave secreta invalida sesiones activas | Baja | Medio | Aceptable para un único usuario; documentar que el logout se resuelve limpiando cookie |
| Fuga de información por mensajes de error detallados | Baja | Medio | Usar mensajes genéricos en login y validar siempre todos los campos |
| Usuario pierde acceso si olvida contraseña | Media | Alto | Fuera de alcance de este spec; se documenta como limitación conocida a resolver en futuro |

---

## 8. Notas y Referencias

- OWASP Password Storage Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
- `golang.org/x/crypto/bcrypt`: https://pkg.go.dev/golang.org/x/crypto/bcrypt
- Go `net/http` cookies: https://pkg.go.dev/net/http#Cookie
- Go `crypto/hmac`: https://pkg.go.dev/crypto/hmac

---

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-12 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-08-12 | p40la-ihost-team | v0.1.0 implementada: wizard, autenticación bcrypt, sesiones HMAC y dashboard protegido
