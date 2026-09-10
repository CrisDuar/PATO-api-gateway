# PATO API Gateway

## 1. Descripción

Se implementó un **API Gateway** en Go (Gin) como punto único de entrada para los clientes de PATO. El gateway centraliza:

* Enrutamiento hacia los microservicios internos (actualmente, el **user-service**).
* Autenticación de las rutas protegidas, validando el token de sesión contra **Valkey** antes de reenviar la petición.
* CORS.
* Trazabilidad mediante `X-Request-ID`.
* Health check propio, independiente de los servicios downstream.

El gateway valida un **token de sesión opaco** (UUID) emitido durante el login y almacenado en Valkey. Si el token es válido, el gateway agrega el header `X-User-ID` y reenvía la petición al microservicio correspondiente.

```text
Cliente → Gateway → (Auth contra Valkey) → Microservicio
```

---

# 2. Arquitectura involucrada

```text
internal/
├── config/
│   └── config.go
│
├── database/
│   └── valkey.go
│
├── handlers/
│   └── proxy_handler.go
│
├── middleware/
│   ├── auth.go
│   ├── cors.go
│   └── request_id.go
│
├── proxy/
│   └── proxy.go
│
└── services/
    └── auth_service.go
```

### Responsabilidad de cada componente

| Componente               | Responsabilidad                                              |
| ------------------------- | -------------------------------------------------------------- |
| `config`                  | Cargar variables de entorno (puerto, URLs, Valkey, CORS)       |
| `database/valkey`         | Conectar con Valkey                                             |
| `services/auth_service`   | Validar el token de sesión contra Valkey y obtener el `userID`  |
| `middleware/auth`         | Extraer el header `Authorization`, validar el token, abortar con 401 si falla |
| `middleware/cors`         | Agregar los headers `Access-Control-Allow-*`                    |
| `middleware/request_id`   | Generar o propagar `X-Request-ID`                                |
| `proxy`                   | Construir el reverse proxy hacia el user-service                 |
| `handlers/proxy_handler`  | Delegar la petición HTTP al reverse proxy                        |

---

# 3. Configuración mediante `.env`

```env
APP_PORT=8080
APP_ENV=development
APP_NAME=PATO API Gateway

USER_SERVICE_URL=http://tu_url:8081

APP_VALKEY_ADDR=tu_ip:6379
APP_VALKEY_USER=tu_usuario_valkey
APP_VALKEY_PASSWORD=tu_contraseña_muy_segura
APP_VALKEY_DB=0

APP_CORS_ORIGIN=http://tu_url:3000
```

El archivo `.env` **no debe subirse al repositorio**, por lo que debe estar incluido en `.gitignore`:

```gitignore
.env
```

---

# 4. Autenticación mediante token de sesión

El gateway no verifica una firma criptográfica sobre el token; en su lugar consulta directamente su existencia en Valkey:

1. El cliente hace login (a través del gateway).
2. Se genera un token de sesión (UUID) y se guarda en Valkey junto al `userID`.
3. El cliente envía ese token en cada petición protegida:

```http
Authorization: Bearer <token>
```

4. El gateway busca ese token en Valkey mediante `AuthService.ValidateToken()`. Si existe, obtiene el `userID` asociado; si no, responde `401`.

La expiración del token la controla el TTL de la clave en Valkey.

---

# 5. Middleware de autenticación

El middleware (`internal/middleware/auth.go`) hace lo siguiente:

1. Obtiene el header `Authorization`.
2. Comprueba que use el esquema `Bearer`.
3. Extrae el token.
4. Llama a `authService.ValidateToken(token)`, que consulta Valkey.
5. Si es válido, agrega `userID` al contexto de Gin y `X-User-ID` al request antes de reenviarlo al microservicio.
6. Si no, aborta con `401`.

Flujo:

```text
Request
  │
  ▼
Authorization: Bearer <token> 
  │
  ▼
AuthMiddleware
  │
  ├── ¿Existe el header?
  │       └── No → 401 UNAUTHORIZED
  │
  ├── ¿Formato "Bearer <token>"?
  │       └── No → 401 UNAUTHORIZED
  │
  ├── ¿Token válido en Valkey?
  │       └── No → 401 UNAUTHORIZED
  │
  └── Sí
       │
       ▼
  Set userID + X-User-ID
       │
       ▼
  ProxyHandler → user-service
```

---

# 6. Resumen de endpoints

| Método  | Endpoint                       | Descripción                          | Autenticación |
| ------- | ------------------------------- | -------------------------------------- | -------------- |
| `GET`   | `/health`                       | Comprobar estado del gateway            | No             |
| `POST`  | `/api/users/register`           | Registrar usuario (proxy)               | No             |
| `POST`  | `/api/users/login`              | Iniciar sesión y obtener token (proxy)  | No             |
| `POST`  | `/api/users/verify-email`       | Verificar correo (proxy)                | No             |
| `POST`  | `/api/users/forgot-password`    | Solicitar recuperación de contraseña (proxy) | No        |
| `POST`  | `/api/users/reset-password`     | Restablecer contraseña (proxy)          | No             |
| `GET`   | `/api/users`                    | Listar todos los usuarios               | Sí             |
| `GET`   | `/api/users/me`                 | Obtener perfil del usuario autenticado  | Sí             |
| `POST`  | `/api/users/logout`             | Cerrar sesión                           | Sí             |
| `PATCH` | `/api/users/email`              | Cambiar el correo del usuario           | Sí             |
| `PATCH` | `/api/users/username`           | Cambiar el nombre de usuario            | Sí             |
| `PATCH` | `/api/users/password`           | Cambiar la contraseña del usuario       | Sí             |

---

# 7. Health Check

### Request

```http
GET http://localhost:8080/health
```

### Respuesta esperada

```json
{
    "status": "healthy",
    "app": "PATO API Gateway"
}
```

No depende de Valkey ni del user-service — sirve para confirmar que el proceso del gateway está vivo.

---

# 8. Registro

### Request

```http
POST http://localhost:8080/api/users/register
```

### Body

```json
{
    "username": "natalia",
    "email": "natalia@gmail.com",
    "password": "Password123!",
    "confirm_password": "Password123!"
}
```

### Resultado

El gateway reenvía la petición sin autenticación al microservicio de usuarios, que crea el usuario y dispara el correo de verificación.

---

# 9. Verificación de correo

### Request

```http
POST http://localhost:8080/api/users/verify-email
```

### Body

```json
{
    "email": "natalia@gmail.com",
    "token": "CODIGO_DE_VERIFICACION"
}
```

### Respuesta

```json
{
    "message": "Email verified successfully"
}
```

---

# 10. Recuperación de contraseña

### Request — solicitar token

```http
POST http://localhost:8080/api/users/forgot-password
```

```json
{
    "email": "natalia@gmail.com"
}
```

### Request — restablecer con el token recibido

```http
POST http://localhost:8080/api/users/reset-password
```

```json
{
    "token": "TOKEN_RECIBIDO_POR_CORREO",
    "password": "NuevaPassword456!",
    "confirm_password": "NuevaPassword456!"
}
```

---

# 11. Login

### Request

```http
POST http://localhost:8080/api/users/login
```

### Body

```json
{
    "email": "natalia@gmail.com",
    "password": "Password123!"
}
```

### Resultado esperado

```json
{
    "token": "eb6211f5-e7c6-4ca6-891e-bae599cf9cf0",
    "expires_in": 3600
}
```

El campo `token` es el que se usa como `Authorization: Bearer <token>` en el resto de endpoints protegidos.

### Respuesta si el correo no está verificado

```json
{
    "error": "email not verified",
    "code": "EMAIL_NOT_VERIFIED"
}
```

HTTP `403 Forbidden`. El usuario debe verificar su correo (`/api/users/verify-email`) antes de poder iniciar sesión.

---

# 12. Uso del token en Postman

Para probar un endpoint protegido:

1. Crear una petición.
2. Ir a **Headers**.
3. Agregar `Authorization` con valor `Bearer <token>` (sin símbolos extra como `<` o `>`, solo el valor devuelto por `/login`).
4. Ejecutar la petición.

Postman enviará:

```http
Authorization: Bearer eb6211f5-e7c6-4ca6-891e-bae599cf9cf0
```

Si el token existe en Valkey, el middleware permitirá el acceso.

---

# 13. Prueba de acceso sin token

Para comprobar que el middleware realmente protege el endpoint:

```text
GET /api/users/me
```

sin enviar el header `Authorization`.

El servidor debe responder:

```json
{
    "error": "Authorization header required",
    "code": "UNAUTHORIZED"
}
```

---

# 14. Prueba de acceso con token

```text
Sin token             → 401 UNAUTHORIZED
Con token válido      → 200 OK
Con token inválido    → 401 UNAUTHORIZED
Con token expirado/usado tras logout → 401 UNAUTHORIZED
```

---

# 15. Obtener perfil

### Request

```http
GET http://localhost:8080/api/users/me
Authorization: Bearer <token>
```

### Respuesta exitosa

```json
{
    "id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "username": "natalia",
    "email": "natalia@gmail.com",
    "created_at": "2026-08-08T..."
}
```

---

# 16. Listado de usuarios

Permite a cualquier usuario autenticado obtener el listado completo de usuarios registrados.

### Request

```http
GET http://localhost:8080/api/users
Authorization: Bearer <token>
```

No recibe body, ya que es una petición `GET`.

### Respuesta exitosa

```json
[
    {
        "id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
        "username": "natalia",
        "email": "natalia@gmail.com",
        "created_at": "2026-08-08T..."
    },
    {
        "id": "yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy",
        "username": "pedro",
        "email": "pedro@gmail.com",
        "created_at": "2026-08-07T..."
    }
]
```

### Posibles errores

| Código HTTP | Code                | Causa                                              |
| ----------- | -------------------- | ----------------------------------------------------- |
| 401         | `UNAUTHORIZED`        | Falta el token, formato inválido, o no existe en Valkey |
| 500         | `USERS_LIST_FAILED`   | Error al consultar la base de datos en el microservicio |

---

# 17. Cambio de correo electrónico

Permite a un usuario autenticado actualizar su correo, confirmando su identidad con la contraseña actual.

### Request

```http
PATCH http://localhost:8080/api/users/email
Authorization: Bearer <token>
```

### Body

```json
{
    "new_email": "nueva-natalia@gmail.com",
    "password": "Password123!"
}
```

### Flujo

```text
PATCH /api/users/email
          │
          ▼
   AuthMiddleware (valida token contra Valkey, agrega X-User-ID)
          │
          ▼
    Proxy hacia el microservicio de usuarios
          │
          ▼
   Actualización del email
          │
          ├── Buscar usuario por ID
          ├── Verificar contraseña actual
          ├── Rechazar si new_email == email actual
          ├── Rechazar si new_email ya está registrado
          └── Actualizar email y marcar email_verified = false
                    │
                    ▼
              Respuesta HTTP
```

### Respuesta exitosa

```json
{
    "id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "username": "natalia",
    "email": "nueva-natalia@gmail.com",
    "created_at": "2026-08-08T..."
}
```

### Posibles errores

| Código HTTP | Code                  | Causa                                                                 |
| ----------- | ---------------------- | ---------------------------------------------------------------------- |
| 401         | `UNAUTHORIZED`          | Falta el token, formato inválido, o no existe en Valkey                |
| 400         | `INVALID_REQUEST`       | JSON mal formado                                                        |
| 400         | `VALIDATION_ERROR`      | `new_email` no es un email válido o falta `password`                    |
| 400         | `EMAIL_UPDATE_FAILED`   | Contraseña incorrecta, email igual al actual, o email ya registrado    |

---

# 17.1 Cambio de nombre de usuario

Permite a un usuario autenticado actualizar su nombre de usuario (`username`).

### Request

```http
PATCH http://localhost:8080/api/users/username
Authorization: Bearer <token>
```

### Body

```json
{
    "new_username": "natalia_nueva"
}
```

### Flujo

```text
PATCH /api/users/username
          │
          ▼
   AuthMiddleware (valida token contra Valkey, agrega X-User-ID)
          │
          ▼
    Proxy hacia el microservicio de usuarios
          │
          ▼
   Actualización del username
          │
          ├── Buscar usuario por ID
          ├── Rechazar si new_username == username actual
          └── Actualizar username
                    │
                    ▼
              Respuesta HTTP
```

### Respuesta exitosa

```json
{
    "id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "username": "natalia_nueva",
    "email": "natalia@gmail.com",
    "created_at": "2026-08-08T..."
}
```

### Posibles errores

| Código HTTP | Code                     | Causa                                                                 |
| ----------- | ------------------------ | ---------------------------------------------------------------------- |
| 401         | `UNAUTHORIZED`           | Falta el token, formato inválido, o no existe en Valkey                |
| 400         | `INVALID_REQUEST`        | JSON mal formado                                                        |
| 400         | `VALIDATION_ERROR`       | `new_username` vacío o fuera del rango de 2 a 50 caracteres             |
| 400         | `USERNAME_UPDATE_FAILED` | `new_username` igual al actual, o usuario no encontrado                |

---

# 18. Cambio de contraseña

### Request

```http
PATCH http://localhost:8080/api/users/password
Authorization: Bearer <token>
```

### Body

```json
{
    "current_password": "Password123!",
    "new_password": "NuevaPassword456!",
    "confirm_new_password": "NuevaPassword456!"
}
```

### Respuesta exitosa

```json
{
    "message": "Password changed successfully"
}
```

### Posibles errores

| Código HTTP | Code                     | Causa                                                                       |
| ----------- | -------------------------- | ------------------------------------------------------------------------------ |
| 401         | `UNAUTHORIZED`             | Falta el token, formato inválido, o no existe en Valkey                        |
| 400         | `INVALID_REQUEST`          | JSON mal formado                                                                 |
| 400         | `VALIDATION_ERROR`         | Contraseñas no coinciden o `new_password` tiene menos de 8 caracteres           |
| 400         | `PASSWORD_CHANGE_FAILED`   | Contraseña actual incorrecta o igual a la nueva                                 |

---

# 19. Logout

### Request

```http
POST http://localhost:8080/api/users/logout
Authorization: Bearer <token>
```

### Respuesta exitosa

```json
{
    "message": "Logged out successfully"
}
```

Tras el logout, el token se elimina de Valkey: cualquier petición posterior con ese mismo token responde `401`.

---

# 20. Flujo completo del sistema

```text
                    ┌──────────────┐
                    │    Cliente   │
                    └──────┬───────┘
                           │
                           │ Register
                           ▼
                    ┌──────────────┐
                    │   Gateway    │  (sin auth)
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────────┐
                    │ Microservicio de │
                    │     usuarios      │
                    └──────┬───────────┘
                           │
                  ┌────────┴────────┐
                  ▼                 ▼
             PostgreSQL       Email Service
                  │                 │
                  └────────┬────────┘
                           │
                      Verificación
                           │
                           ▼
                         Login
                           │
                           ▼
                    Token de sesión (Valkey)
                           │
                           ▼
                    Cliente autenticado
                           │
                           │ Bearer <token>
                           ▼
                    Gateway: AuthMiddleware
                           │
                     ┌─────┴─────┐
                     ▼           ▼
              Válido en Valkey   Inválido
                     │           │
                     ▼           ▼
              Proxy a servicio  401
```

---

# 21. Consideraciones de seguridad

* `APP_VALKEY_PASSWORD` y demás credenciales deben mantenerse fuera del código fuente.
* El archivo `.env` no debe subirse al repositorio.
* El gateway nunca reenvía peticiones a rutas protegidas sin validar el token contra Valkey primero.
* El header `X-User-ID` solo lo agrega el gateway tras validar el token — el microservicio downstream confía en ese header porque solo el gateway puede alcanzarlo directamente en producción.
* El token de sesión debe enviarse mediante HTTPS en ambientes de producción.
* Los mensajes de error de autenticación evitan revelar si el problema es el formato, la existencia del token o su expiración, para no filtrar información sobre sesiones válidas.

---

# 22. Resultado

Con esta implementación, el PATO API Gateway centraliza el enrutamiento y la autenticación de los clientes hacia los microservicios internos:

```text
Cliente
   │
Register / Login (sin auth, vía gateway)
   │
Token de sesión (Valkey)
   │
Autenticación mediante Bearer Token
   │
Gateway valida contra Valkey
   │
Proxy hacia el microservicio con X-User-ID
```

El token de sesión no tiene expiración embebida: su validez depende del TTL configurado en Valkey y se invalida explícitamente al hacer logout.
