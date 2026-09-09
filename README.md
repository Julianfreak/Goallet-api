# Goallet API - Motor Transaccional de Billetera Digital

Motor transaccional para billetera digital de alto rendimiento desarrollado en **Go (Golang)**, diseñado bajo los principios de **Arquitectura Hexagonal (Ports and Adapters)** y **Clean Code**.

---

## Arquitectura del Proyecto

El sistema aísla completamente la lógica del negocio de los frameworks web y mecanismos de persistencia:

* **internal/core/domain:** Entidades puras y reglas de negocio inmutables (Cuenta, Transacciones, validaciones de saldo positivo y restricciones de eliminación).
* **internal/core/ports:** Interfaces que definen los contratos para repositorios (salida) y casos de uso (entrada).
* **internal/core/services:** Orquestación y casos de uso (CRUD de cuentas, Depósitos, Retiros y Transferencias seguras).
* **internal/adapters/handlers:** Adaptador de entrada HTTP implementado con **Gin Gonic**.
* **internal/adapters/storage:** Adaptador de persistencia en memoria con control de concurrencia mediante `sync.RWMutex`.
* **cmd/api:** Punto de arranque e inicialización (Composition Root).

---

## Características Técnicas

* **Lenguaje:** Go 1.23+
* **Framework Web:** Gin Gonic v1.10+
* **Diseño:** Arquitectura Hexagonal y Domain-Driven Design (DDD) básico.
* **Concurrencia Segura:** Uso de `sync.RWMutex` para permitir lecturas masivas concurrentes y escrituras exclusivas sin condiciones de carrera.
* **Contenedorización:** Construcción multietapa (*Multi-stage build*) en Docker con imagen final ultraligera (< 15MB) basada en Alpine Linux.

---

## Cómo ejecutar el proyecto

### Prerrequisitos
* Tener instalado Docker y Docker Compose.

### Ejecución con un solo comando

```bash
docker compose up --build
```

El servicio quedará disponible en: `http://localhost:7077`

---

## Endpoints de la API (v1)

| Método | Endpoint | Descripción |
| :--- | :--- | :--- |
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
- [x] Fase 2: Adaptador de Persistencia en Memoria con protección de concurrencia (`sync.RWMutex`).
- [x] Fase 3: Adaptador de Entrada HTTP REST con framework **Gin Gonic** y CRUD completo.
- [ ] Fase 4: Pruebas Unitarias con Mocks e integración continua (CI).