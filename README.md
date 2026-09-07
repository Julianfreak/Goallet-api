# Goallet API - Motor Transaccional de Billetera Digital

Motor transaccional para billetera digital de alto rendimiento desarrollado en **Go (Golang)**, diseñado bajo los principios de **Arquitectura Hexagonal (Ports and Adapters)** y **Clean Code**.

---

## Arquitectura del Proyecto

El sistema aísla completamente la lógica del negocio de los frameworks web y mecanismos de persistencia:

* **internal/core/domain:** Entidades puras y reglas de negocio inmutables (Cuenta, Transacciones, validaciones de saldo).
* **internal/core/ports:** Interfaces que definen los contratos para repositorios (salida) y casos de uso (entrada).
* **internal/core/services:** Orquestación y casos de uso (Creación de cuentas, Depósitos, Retiros y Transferencias seguras).
* **cmd/api:** Punto de arranque e inicialización de la aplicación.

---

## Características Técnicas

* **Lenguaje:** Go 1.23+
* **Diseño:** Arquitectura Hexagonal y Domain-Driven Design (DDD) básico.
* **Manejo de Errores:** Errores explícitos y tipados de dominio.
* **Contenedorización:** Construcción multietapa (*Multi-stage build*) en Docker con imagen final ultraligera (< 15MB) basada en Alpine Linux.

---

## Cómo ejecutar el proyecto

### Prerrequisitos
* Tener instalado Docker y Docker Compose.

### Ejecución con un solo comando

```bash
docker compose up --build
```

---

## Estado del Proyecto

- [x] Fase 1: Dominio, Puertos y Servicios de Billetera.
- [x] Fase 2: Adaptador de Persistencia en Memoria con protección de concurrencia (`sync.RWMutex`).
- [ ] Fase 3: Adaptador de Entrada HTTP REST con framework **Gin Gonic**.
- [ ] Fase 4: Pruebas Unitarias con Mocks e integración continua (CI).