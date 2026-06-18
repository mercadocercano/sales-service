# ADR-003: Modelo de Caja / Sesión de Caja (POS)

**Estado:** Aceptado (decisiones de owner resueltas 2026-06-17) — pendiente sign-off final de @security sobre la implementación
**Ceremony level:** L4 (maneja efectivo real)
**Épica:** E18 — Tramo A
**Fecha:** 2026-06-17
**Autores:** @dev-architect (diseño) + @dev-security (threat model), consolidado por meta-router
**Relacionados:** ADR-001 (operación atómica stock), ADR-002 (snapshot histórico order items)

## Contexto

El flujo POS (E07) descuenta stock atómicamente y registra `pos_sales` + `pos_sale_items`, pero no existe el concepto de **caja**: ni apertura/cierre, ni arqueo, ni asociación venta↔caja. Un comercio piloto real (E10) necesita abrir caja al iniciar la jornada, asociar cada venta a la caja abierta, y cerrar con arqueo de efectivo. Como maneja dinero real, es L4: el diseño debe resolverse en este ADR antes de tocar código.

Estado verificado (2026-06-17): Caja no existe (ni tabla ni concepto); `pos_sales` no tiene FK de caja.

## Decisión

### 1. Dónde vive — `sales-service`
La caja vive en **sales-service**, no en ledger-service. `ledger-service` es append-only dirigido por eventos idempotentes (best-effort); el arqueo de efectivo no puede depender de eventos best-effort. Además todos los datos que alimentan el arqueo (`pos_sales`, medios de pago) ya viven en sales.

### 2. Modelo del agregado
- **`cash_register_sessions`** (raíz de agregado): apertura (`opening_amount`), estado, snapshot de cierre inmutable, `version` (optimistic lock, patrón de `sales_orders`).
- **`cash_movements`** (append-only): solo movimientos **manuales** (ingresos/egresos/retiros/correcciones). Las ventas **NO** se duplican acá — se derivan de `pos_sales` (single source of truth).
- Montos `NUMERIC(15,2)`. Estados: `OPEN → CLOSING → CLOSED` / `PENDING_REVIEW`, transiciones unidireccionales validadas en Go.

### 3. Invariante "una caja abierta" — por TERMINAL
Una caja abierta por **terminal** (`point_of_sale_id`) dentro del tenant. Garantía: índice único parcial `UNIQUE (tenant_id, point_of_sale_id) WHERE status='open'` + chequeo en Go dentro de transacción. (No por usuario: varios cajeros pueden rotar en una terminal. No por tenant: un comercio puede tener varias cajas.)

### 4. Vínculo venta↔caja — FK nullable, modo DEGRADADO
`pos_sales.cash_register_session_id` **nullable** (retrocompatible con ventas existentes). **DECISIÓN OWNER (2026-06-17): modo DEGRADADO** — la venta NO se bloquea si no hay caja abierta. Si hay caja abierta para la terminal, la venta se asocia; si no, la venta procede igual con `cash_register_session_id = NULL`. El flag `require_open_cash_register` queda como capacidad futura (default `false`), pero el piloto opera en degradado.

### 5. Arqueo — solo efectivo, esperado derivado server-side
`expected_amount = opening_amount + Σ(ventas en efectivo, por final_amount) + Σ(ingresos manuales) − Σ(egresos/retiros)`. Se usa `final_amount` (no `amount_paid − change`) para evitar doble resta del vuelto. Otros medios de pago no afectan el arqueo de caja. El cliente solo aporta `counted_amount`; `difference = counted − expected` se calcula y persiste server-side. La diferencia se registra sin bloquear el cierre (pero ver REQ-A03 de seguridad: diferencia > umbral → `PENDING_REVIEW`).
> **[DECISIÓN OWNER]** "Efectivo" hoy se identifica por el string `"Efectivo"` hardcodeado en el Flutter. Hace falta un flag `is_cash` en el catálogo de métodos de pago (dependencia con payment-method-service).

### 6. Atomicidad / concurrencia — `SELECT FOR UPDATE`
Apertura, cierre y registro de movimientos dentro de `BeginTx` + `SELECT ... FOR UPDATE` sobre la fila de sesión, revalidando `status` dentro de la transacción. El saga POS (descuento de stock vía HTTP remoto) impide una única transacción global → la asociación venta↔caja se hace re-validando `status='open'` dentro de la tx que inserta `pos_sales`. Cubre doble cierre (C1), doble apertura (C2), venta entrando a caja en cierre (C3 → por decisión del owner se **fuerza reapertura** en vez de rechazar con 409: la venta nunca se pierde), retiros concurrentes (C4). Sin UPSERT defensivo; lógica de negocio en Go, no en triggers.

### 7. Eventos de dominio — notificación, NO fuente de verdad
`sales.cash_register.opened` / `closed` / `movement_registered` como notificación. El arqueo NO se reconstruye desde eventos (best-effort prohibido para flujos financieros).

### 8. Endpoints
| Endpoint | Descripción |
|----------|-------------|
| `POST /api/v1/sales/cash-sessions` | Abrir caja (monto inicial) |
| `GET /api/v1/sales/cash-sessions/current` | Caja abierta de la terminal |
| `GET /api/v1/sales/cash-sessions/{id}` | Detalle + arqueo |
| `POST /api/v1/sales/cash-sessions/{id}/movements` | Movimiento manual |
| `POST /api/v1/sales/cash-sessions/{id}/close` | Cierre con arqueo |

### 9. Migración retrocompatible
Migraciones aditivas + flag de adopción gradual. Sin backfill. La FK `cash_register_session_id` queda nullable (no se pasa a NOT NULL).

## Gate de seguridad (de @security) — BLOQUEANTE para implementar

El baseline de auth/authz actual de sales-service **NO alcanza para L4 dinero**. NO-GO hasta cumplir:

1. **[BLOQUEANTE · OWNER]** RBAC con roles `cashier`/`supervisor` desde claims firmados del JWT. Hoy **no existe RBAC** — cualquier usuario autenticado del tenant puede todo. **Dependencia: `iam-service` debe emitir roles en el JWT.**
2. **[BLOQUEANTE · OWNER]** Separación de funciones en el cierre: `expected` server-side; diferencia > umbral → `PENDING_REVIEW` aprobado por `supervisor`; el cajero nunca aprueba su propio descuadre.
3. **[BLOQUEANTE · OWNER]** Cerrar el bypass de tenant del middleware (`tenant_validation.go:77-81`): si el JWT no trae `tenant_id` ⇒ 403, nunca `c.Next()`. Es IDOR cross-tenant explotable sobre dinero.
4. **[BLOQUEANTE · OWNER]** `cash_movements` append-only; correcciones por contra-asiento; sin UPDATE/DELETE.
5. **[BLOQUEANTE · OWNER]** Audit log inmutable (`cash_audit_log`) de toda operación de efectivo: `user_id` del JWT, timestamp de servidor, montos, `expected/counted/difference`.
6. **[BLOQUEANTE]** Cierre/apertura con `SELECT FOR UPDATE` + estado revalidado en tx; constraint único de una caja abierta por terminal.
7. **[BLOQUEANTE]** Todas las queries de caja filtran por `tenant_id`; `id` ajeno ⇒ 404.

Antes de producción: Audit mode completo de `owasp-top10` (A01/A04/A08/A09 ≥ COVERED), auditoría `concurrency-transactions` con veredicto GO, rate limiting en Kong sobre endpoints de caja, `Namespace: "mc"` activado, verificar que token/claims nunca aparecen en logs.

## Alternativas consideradas

- **Caja en ledger-service** — descartada: arqueo dependería de eventos best-effort, inaceptable para efectivo.
- **Movimientos de venta duplicados en `cash_movements`** — descartado: rompe single-source-of-truth; las ventas se derivan de `pos_sales`.
- **Invariante por usuario o por tenant** — descartado: por terminal modela mejor la operación real (rotación de cajeros, múltiples cajas).

## Consecuencias

- **Positivas:** arqueo confiable y auditable; retrocompatible; cierra el loop para el piloto (E10).
- **Costo / dependencias:** requiere que iam-service emita roles en el JWT (bloqueante #1) y un flag `is_cash` en payment-method-service (decisión #5). El gate de seguridad agrega trabajo de RBAC + cierre del bypass de tenant que hoy afecta a TODO sales-service, no solo a caja.

## Riesgos

- **Sin RBAC, caja es insegura** → bloqueante #1; depende de iam-service.
- **Bypass de tenant existente** → bloqueante #3; afecta también al nuevo `GET /sales/pos/{id}` (Tramo B) como superficie IDOR (mitigado a nivel query con filtro por tenant + 404).
- **Heterogeneidad ESC/POS** (Tramo C, fuera de este ADR).

## Decisiones del owner — RESUELTAS (2026-06-17)

1. **Exigir caja abierta para vender** → **DEGRADADO**. La venta nunca se bloquea por falta de caja; se asocia si hay caja abierta, si no procede con FK NULL.
2. **Dependencia con iam-service (roles en JWT)** → **APROBADA**. Debe quedar **documentada con diagramas de flujo** (design doc en iam-service). Prerequisito del gate de seguridad #1 (RBAC).
3. **Dependencia con payment-method-service (flag `is_cash`)** → **APROBADA**.
4. **Umbral de diferencia de arqueo** → **CONFIGURABLE** (por tenant; default a definir en implementación). Diferencia > umbral → `PENDING_REVIEW` (supervisor).
5. **Venta que llega durante el cierre (`CLOSING`)** → **FORZAR REAPERTURA**: la venta no se rechaza; reabre la sesión de caja para no perder la venta. (Detalle de implementación a precisar con architect en planning.)
6. **Alcance ampliado (cerrar bypass de tenant + RBAC en todo sales-service)** → **ACEPTADO**.
