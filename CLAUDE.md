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

## Memoria persistente (Engram)

Tenés acceso a memoria persistente entre sesiones vía las herramientas MCP de Engram (`mem_save`, `mem_search`, `mem_context`, etc.). Proyecto: **`mercado-cercano`** (memoria unificada del ecosistema, compartida con iam/pim/etc.).

**Cuándo guardar** — sin esperar que te lo pidan:
- Al resolver un bug no trivial: síntoma, causa raíz, fix aplicado.
- Al tomar una decisión de diseño: qué se decidió y por qué.
- Al descubrir un patrón o convención del proyecto que no está documentada.
- Al completar una feature o refactor significativo: qué cambió y dónde.

**Cuándo buscar** — antes de empezar cualquier tarea:
- `mem_context` al inicio de sesión o tras una compaction para recuperar el estado anterior.
- `mem_search` cuando el usuario menciona algo que puede tener historial ("el bug de autenticación", "la migración de la semana pasada").

**Al cerrar sesión**: llamar `mem_session_summary` para dejar un resumen recuperable.
