# sales-service

Guía breve para asistentes de código en el repositorio **sales-service** (Mercado Cercano).

## Identidad

Microservicio de **ventas**: órdenes (sales orders), ventas POS, cobranzas, créditos a cuenta y reportes operativos. Publica eventos de dominio al **EventBus** (PostgreSQL) para consumo por ledger y otros servicios.

- **Puerto HTTP en código**: variable `PORT`, default **`8080`** (en Docker suele mapearse al host, p. ej. `8120`, según compose del entorno)
- **Stack**: Go, Gin, PostgreSQL, librería `github.com/mercadocercano/eventbus`
- **Prefijo API**: `/api/v1`

## Comandos esenciales

| Acción | Comando |
|--------|---------|
| Compilar | `go build .` |
| Tests | `go test ./...` |
| Contenedor | `docker-compose up sales-service` (si el compose del workspace define el servicio) |

## Contexto on-demand (cargar según necesidad)

| Archivo | Cuándo cargar |
|---------|---------------|
| `sales-service-management/api-endpoints.md` | Al trabajar con rutas HTTP |
| `sales-service-management/architecture.md` | Dominio, persistencia, eventos, integraciones |
| `sales-service-management/config.md` | Variables de entorno, Kong, despliegue |

## Reglas compartidas (cargar según contexto)

| Regla | Archivo |
|-------|---------|
| Arquitectura hexagonal | `ai-tools/rules/architecture.md` |
| Multi-tenancy | `ai-tools/rules/multi-tenant.md` |

Headers típicos: `Authorization`, `X-Tenant-ID` (`mercadocercano/middleware`). Versión API en `main.go` (p. ej. hito v0.5).
