# Goallet API - Motor Transaccional de Billetera Digital

Motor transaccional para billetera digital de alto rendimiento desarrollado en **Go (Golang)**, diseñado bajo los principios de **Arquitectura Hexagonal (Ports and Adapters)**, **Clean Code** y **Observabilidad Nativa**.

---

## Arquitectura del Proyecto

El sistema aísla completamente la lógica del negocio de los frameworks web, mecanismos de persistencia y herramientas de telemetría. ¿Por qué? Para garantizar que el núcleo financiero sea agnóstico a la infraestructura, altamente escalable y testeable mediante la inyección de dependencias (Mocks).

- **internal/core/domain:** Entidades puras y reglas de negocio inmutables (Cuenta, Transacciones, validaciones de saldo positivo y restricciones de eliminación). Decisión: Centralizar aquí las validaciones evita que la lógica de negocio se filtre a los controladores.
- **internal/core/ports:** Interfaces que definen los contratos para repositorios de salida y casos de uso de entrada.
- **internal/core/services:** Orquestación y casos de uso (CRUD de cuentas, Depósitos, Retiros y Transferencias seguras).
- **internal/adapters/handlers:** Adaptador de entrada HTTP implementado con **Gin Gonic** y middlewares de registro estructurado.
- **internal/adapters/storage:** Adaptadores de persistencia relacional con **PostgreSQL (`pgxpool`)**, incorporando control de concurrencia transaccional mediante bloqueos exclusivos.
- **cmd/api:** Punto de arranque e inicialización (Composition Root), servidor de métricas secundario y gestión del ciclo de vida de la conexión a base de datos.

---

## 🚀 Características Técnicas, Seguridad y Observabilidad

- **Lenguaje:** Go 1.23+
- **Framework Web:** Gin Gonic v1.10+
- **Seguridad:** Autenticación y autorización basada en **JSON Web Tokens (JWT)** con firmados HMAC-SHA256 (`github.com/golang-jwt/jwt/v5`).
- **Base de Datos:** PostgreSQL 16+ con driver de alto rendimiento `pgx/v5`.
- **Diseño:** Arquitectura Hexagonal y Domain-Driven Design (DDD).
- **Consistencia ACID y Concurrencia Segura:** Uso de transacciones explícitas con **bloqueos a nivel de fila (`SELECT ... FOR UPDATE`)** para prevenir condiciones de carrera (*race conditions*) y dobles gastos bajo alta concurrencia.
- **Precisión Financiera:** Almacenamiento monetario estricto mediante tipos `NUMERIC(15,2)`.
- **Logs Estructurados (JSON):** Implementación de `slog` con salida simultánea a `stdout` y archivo persistente `app.log`.
- **Métricas Prometheus:** Servidor dedicado en goroutine independiente (`:2112/metrics`) para recolección de telemetría de Go y runtime.
- **Dashboards (Grafana):** Tableros integrados conectando métricas de rendimiento (Prometheus) y trazabilidad de logs (Loki).
- **Contenedorización:** Soporte multi-contenedor mediante Docker Compose con paridad total entre entornos de trabajo.
- **Idempotencia Financiera:** Soporte para el encabezado `X-Idempotency-Key` (UUID) en endpoints transaccionales (depósitos, retiros y transferencias), previniendo cargos duplicados ante reintentos de red o fallos de cliente mediante almacenamiento y respuesta en caché persistente.

## Nuevas Características y Arquitectura

### 1. Ledger y Movimientos Financieros
- **Auditoría de Cuentas:** Cada depósito, retiro o transferencia registra de forma automática su respectivo movimiento en la tabla `movimientos`.
- **Paginación y Filtros:** El endpoint de movimientos (`GET /api/v1/cuentas/{id}/movimientos`) soporta parámetros de consulta para paginación (`limit`, `page`) y rangos de fecha (`fecha_inicio`, `fecha_fin`).

## ⚡ Arquitectura Dirigida por Eventos (Async Worker)

Goallet API implementa un modelo desacoplado basado en eventos para el procesamiento asíncrono de transacciones utilizando **RabbitMQ**.

### Resiliencia y Auditoría de Eventos Fallidos (Fallback Strategy)
Si al procesar o publicar un evento en RabbitMQ ocurre una falla (desconexión o indisponibilidad de la cola), la API implementa un mecanismo de fallback que persiste automáticamente la información en la tabla `eventos_auditoria` de PostgreSQL:
- **Respaldo en BD (`AuditRepository`)**: Evita la pérdida de eventos registrando el `payload` (JSONB), la transacción asociada, el detalle del error y el contador de intentos.
- **Tolerancia a Fallos**: Garantiza consistencia eventual permitiendo auditar o reintentar posteriormente la publicación de mensajes fallidos sin interrumpir la experiencia transaccional del cliente.

### Flujo de Eventos
1. **Publicación (`EventPublisher`)**: Cuando una transferencia se completa con éxito, la API publica un `TransferenciaEvent` con la *routing key* `transferencia.completada` hacia el *topic exchange* de RabbitMQ (`billetera.events`).
2. **Consumo (`EventConsumer`)**: Un worker en segundo plano (Go goroutine) suscrito a la cola `billetera.notificaciones` procesa los eventos de forma asíncrona con confirmación manual (`ack/nack`), garantizando robustez ante fallos.


---

## 🚀 Despliegue y Puesta en Marcha (Docker Compose)

El stack completo se ejecuta mediante contenedores aislados que comparten una red virtual (`monitoring`):

### 1. Configuración de Entorno (`.env`)
Crea un archivo `.env` en la raíz del proyecto basándote en la siguiente configuración (ajustando `API_PORT` en caso de conflictos de puertos en tu máquina anfitriona):

DB_HOST=postgres
DB_PORT=5432
DB_USER=goallet_user
DB_PASSWORD=goallet_password
DB_NAME=goallet_db
DB_SSLMODE=disable
PORT=8080
API_PORT=8090
GF_ADMIN_PASSWORD=tu_password_seguro_local

## 🐳 Despliegue con Docker Compose

El proyecto incluye un entorno completamente contenerizado para la API, PostgreSQL, Prometheus, Grafana y Loki.

### Inicialización de la Base de Datos
Las migraciones SQL se ejecutan automáticamente de forma nativa la primera vez que se crea el contenedor de PostgreSQL gracias al mapeo en `/docker-entrypoint-initdb.d/`.

### ⚠️ Advertencia Importante sobre Volúmenes
Para detener los servicios conservando tus cuentas, transacciones y datos de prueba, utiliza siempre:
\`\`\`bash
docker compose down
\`\`\`
**No utilices** `docker compose down -v` a menos que desees eliminar por completo el volumen persistente de PostgreSQL y reiniciar la base de datos desde cero.

### 2. Levantar el Stack Completo
Ejecuta el siguiente comando para construir la imagen de la API y levantar todos los servicios (PostgreSQL con healthcheck, API, Prometheus, Loki, Grafana y Alloy):

docker compose up --build -d

### 3. Verificar el Estado de los Contenedores
docker compose ps

---

## Documentación Interactiva (OpenAPI / Swagger)

La API dispone de especificación OpenAPI 3.0 accesible desde la interfaz de Swagger UI cuando el servidor está en ejecución:

- **URL de Swagger UI:** `http://localhost:8090/swagger/index.html` *(o el puerto configurado en `API_PORT`)*

### Regenerar la documentación
Si modificas DTOs, comentarios u operaciones en los handlers, regenera los artefactos OpenAPI ejecutando:

swag init -g cmd/api/main.go --parseDependency --parseInternal

---

## Servicios y Puertos Expuestos

| Servicio | Puerto / Ruta | Descripción |
| --- | --- | --- |
| **API REST (Gin)** | `http://localhost:8090/api/v1` | Endpoints de negocio (cuentas, transferencias, depósitos) |
| **Base de Datos (Postgres)** | `localhost:5432` | Motor relacional con persistencia transaccional ACID |
| **Métricas (Prometheus)** | `http://localhost:2112/metrics` | Endpoint de telemetría e inspección de Goroutines / Memoria |
| **Tableros (Grafana)** | `http://localhost:3000` | Visualización centralizada de logs y métricas |
| **Cola de mensajes (RabbitMQ)** | `http://localhost:15672` | Visualización centralizada de mensajes provenientes de transacciones |

---

## ⚡ Pruebas de Carga y Concurrencia con k6

Para validar el comportamiento del motor transaccional ante alta concurrencia y comprobar la seguridad de los bloqueos de fila ACID, el proyecto incluye un script automatizado en `load-test.js`.

Puedes ejecutar la prueba de estrés mediante un contenedor efímero de `k6` conectado directamente a la red interna de Docker (sin necesidad de instalar herramientas locales):

docker run --rm -i --network goallet-api_monitoring -v $(pwd):/workspace grafana/k6 run /workspace/load-test.js

---

## Pruebas Automatizadas y Cobertura

Para ejecutar la suite completa de pruebas unitarias sobre los casos de uso ejecuta:

go test -v -cover ./internal/core/services/...

---

## Pruebas de Integración (con Testcontainers)

Para validar los repositorios contra una instancia real y efímera de PostgreSQL mediante contenedores Docker ejecuta:

go test -v -tags=integration ./internal/adapters/storage/...

---

## Endpoints de la API (v1)

| Método | Endpoint | Acceso | Descripción |
| --- | --- | --- | --- |
| **POST** | `/api/v1/auth/token` | **Público** | Emite un token de acceso JWT válido por 24 horas |
| **POST** | `/api/v1/cuentas` | **Protegido** | Crea una nueva cuenta bancaria |
| **GET** | `/api/v1/cuentas` | **Protegido** | Lista todas las cuentas registradas |
| **GET** | `/api/v1/cuentas/:id` | **Protegido** | Consulta los detalles y saldo de una cuenta por ID |
| **PUT** | `/api/v1/cuentas/:id` | **Protegido** | Actualiza el nombre del titular de la cuenta |
| **DELETE** | `/api/v1/cuentas/:id` | **Protegido** | Elimina una cuenta (solo si su saldo es $0) |
| **POST** | `/api/v1/cuentas/:id/depositar` | **Protegido** | Realiza un depósito de fondos |
| **POST** | `/api/v1/cuentas/:id/retirar` | **Protegido** | Retira fondos validando saldo disponible |
| **POST** | `/api/v1/cuentas/:id/transferir` | **Protegido** | Transfiere saldo de forma atómica entre cuentas |
| **GET** | `/api/v1/cuentas/:id/movimientos` | **Protegido** | Consulta el extracto histórico y auditoría (Ledger) |

---

## Estado del Proyecto

- [x] Fase 1: Dominio, Puertos y Servicios de Billetera.
- [x] Fase 2: Adaptador de Persistencia en Memoria inicial (`sync.RWMutex`).
- [x] Fase 3: Adaptador de Entrada HTTP REST con framework **Gin Gonic**.
- [x] Fase 4: Observabilidad Nativa (`slog` + Prometheus).
- [x] Fase 5: Stack de Monitoreo (Alloy, Prometheus, Loki y Grafana).
- [x] Fase 6: Pruebas Unitarias con Mocks y Cobertura >80%.
- [x] Fase 7: Migración a PostgreSQL con `pgxpool` y bloqueos de fila (`SELECT ... FOR UPDATE`).
- [x] Fase 8: Pruebas de Integración automatizadas con **Testcontainers**.
- [x] Fase 9: Pipeline de Integración Continua (CI/CD) con **GitHub Actions**.
- [x] Fase 10: Módulo de Auditoría e Historial de Transacciones (**Ledger**).
- [x] Fase 11: Autenticación y Seguridad con **JWT Middleware**.
- [x] Fase 12: Arquitectura Dirigida por Eventos con **RabbitMQ** (Publicador y Worker Asíncrono).
- [x] Fase 13: Estrategia de Fallback y Resiliencia con Tabla de Auditoría (`eventos_auditoria` en PostgreSQL).