# Goallet API - Motor Transaccional de Billetera Digital

Motor transaccional para billetera digital de alto rendimiento desarrollado en **Go (Golang)**, diseñado bajo los principios de **Arquitectura Hexagonal (Ports and Adapters)**, **Clean Code** y **Observabilidad Nativa**.

---

## Arquitectura del Proyecto

El sistema aísla completamente la lógica del negocio de los frameworks web, mecanismos de persistencia y herramientas de telemetría:

- **internal/core/domain:** Entidades puras y reglas de negocio inmutables (Cuenta, Transacciones, validaciones de saldo positivo y restricciones de eliminación).
- **internal/core/ports:** Interfaces que definen los contratos para repositorios de salida y casos de uso de entrada.
- **internal/core/services:** Orquestación y casos de uso (CRUD de cuentas, Depósitos, Retiros y Transferencias seguras).
- **internal/adapters/handlers:** Adaptador de entrada HTTP implementado con **Gin Gonic** y middlewares de registro estructurado.
- **internal/adapters/storage:** Adaptadores de persistencia relacional con **PostgreSQL (`pgxpool`)**, incorporando control de concurrencia transaccional mediante bloqueos exclusivos.
- **cmd/api:** Punto de arranque e inicialización (Composition Root), servidor de métricas secundario y gestión del ciclo de vida de la conexión a base de datos.

---

## Características Técnicas y Observabilidad

- **Lenguaje:** Go 1.23+
- **Framework Web:** Gin Gonic v1.10+
- **Base de Datos:** PostgreSQL 16+ con driver de alto rendimiento `pgx/v5`.
- **Diseño:** Arquitectura Hexagonal y Domain-Driven Design (DDD).
- **Consistencia ACID y Concurrencia Segura:** Uso de transacciones explícitas con **bloqueos a nivel de fila (`SELECT ... FOR UPDATE`)** para prevenir condiciones de carrera (*race conditions*) y dobles gastos bajo alta concurrencia.
- **Precisión Financiera:** Almacenamiento monetario estricto mediante tipos `NUMERIC(15,2)`.
- **Logs Estructurados (JSON):** Implementación de `slog` con salida simultánea a `stdout` y archivo persistente `app.log`.
- **Métricas Prometheus:** Servidor dedicado en goroutine independiente (`:2112/metrics`) para recolección de telemetría de Go y runtime.
- **Dashboards (Grafana):** Tableros integrados conectando métricas de rendimiento (Prometheus) y trazabilidad de logs (Loki).
- **Contenedorización:** Soporte multi-contenedor mediante Docker Compose con paridad total entre entornos de trabajo.

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

| Método | Endpoint | Descripción |
| --- | --- | --- |
| **POST** | `/api/v1/cuentas` | Crea una nueva cuenta bancaria |
| **GET** | `/api/v1/cuentas` | Lista todas las cuentas registradas |
| **GET** | `/api/v1/cuentas/:id` | Consulta los detalles y saldo de una cuenta por ID |
| **PUT** | `/api/v1/cuentas/:id` | Actualiza el nombre del titular de la cuenta |
| **DELETE** | `/api/v1/cuentas/:id` | Elimina una cuenta (solo si su saldo es $0) |
| **POST** | `/api/v1/cuentas/:id/depositar` | Realiza un depósito de fondos |
| **POST** | `/api/v1/cuentas/:id/retirar` | Retira fondos validando saldo disponible |
| **POST** | `/api/v1/cuentas/:id/transferir` | Transfiere saldo de forma atómica entre cuentas |

---

## Estado del Proyecto

- [x] Fase 1: Dominio, Puertos y Servicios de Billetera.
- [x] Fase 2: Adaptador de Persistencia en Memoria inicial (`sync.RWMutex`).
- [x] Fase 3: Adaptador de Entrada HTTP REST con framework **Gin Gonic** y CRUD completo.
- [x] Fase 4: Observabilidad Nativa (Logs estructurados en JSON con `slog` en `app.log` + servidor de métricas Prometheus en `:2112`).
- [x] Fase 5: Integración con el Stack de Monitoreo (Alloy, Prometheus, Loki y Grafana).
- [x] Fase 6: Pruebas Unitarias con Mocks y 100% de Cobertura (`httptest`, `services` y `handlers`).
- [x] Fase 7: Migración a PostgreSQL con `pgxpool`, transacciones ACID y bloqueos de fila (`SELECT ... FOR UPDATE`) para alta concurrencia.
- [x] Fase 8: Pruebas de Integración automatizadas con Testcontainers y PostgreSQL real.
- [x] Fase 9: Pipeline de Integración Continua (CI/CD) con GitHub Actions.