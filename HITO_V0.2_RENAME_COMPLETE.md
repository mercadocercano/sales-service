# HITO v0.2: Renombramiento Estructural Completo

**Servicio:** order-service → sales-service  
**Fecha:** 2026-02-20  
**Estado:** ✅ COMPLETADO Y VALIDADO  

---

## 🎯 Objetivo

Alinear nombre del servicio con arquitectura ERP y contrato de eventos v1.0.

**NO modificar:**
- Tablas DB
- Rutas HTTP
- Lógica de negocio
- Event types

---

## ✅ Cambios Ejecutados

### 1. Renombramiento de directorios

```bash
services/order-service/ → services/sales-service/
src/order/ → src/sales/
```

### 2. Go module

```go
module order → module sales
```

### 3. Imports actualizados (masivo)

```bash
find . -name "*.go" -exec sed -i '' 's|"order/|"sales/|g' {} +
find . -name "*.go" -exec sed -i '' 's|/src/order/|/src/sales/|g' {} +
```

**Resultado:** Todos los imports coherentes con `sales/src/sales/...`

### 4. main.go actualizado

```go
// Aliases actualizados
salesUseCase "sales/src/sales/application/usecase"
salesCache "sales/src/sales/infrastructure/cache"
salesClient "sales/src/sales/infrastructure/client"
salesController "sales/src/sales/infrastructure/controller"
salesPersistence "sales/src/sales/infrastructure/persistence"

// Función renombrada
func setupSalesModule(...) { ... }
```

### 5. Dockerfile

```dockerfile
# Build
-o sales-service .

# Metadata
LABEL org.opencontainers.image.title="Sales Service"

# Entrypoint
ENTRYPOINT ["./sales-service"]
```

### 6. Docker Compose

```yaml
# docker-compose.yml
sales-service:
  build:
    context: ./services/sales-service
  container_name: sales-service

# docker-compose.services.yml
sales-service:
  context: ./services/sales-service
  container_name: sales-service
```

### 7. Kong Gateway

```yaml
services:
  - name: sales-service
    url: http://sales-service:8080
    
routes:
  - name: sales-route
    service: sales-service
    paths:
      - /orders/  # Se mantienen rutas legacy
```

---

## ✅ Validación E2E

### Test ejecutado:

1. **Crear orden:**
   ```sql
   INSERT INTO orders (...) VALUES ('11111111-1111-1111-1111-111111111111', ...)
   ```

2. **Confirmar orden:**
   ```bash
   POST /api/v1/orders/11111111-1111-1111-1111-111111111111/confirm
   ```

3. **Verificar evento:**
   ```sql
   SELECT * FROM event_bus WHERE aggregate_id = '1111...'
   ```

4. **Verificar ledger:**
   ```sql
   SELECT * FROM ledger_entries WHERE document_id = '1111...'
   ```

### Resultados:

- ✅ Health: OK
- ✅ Confirm: CONFIRMED
- ✅ Evento: `sales.order.confirmed` publicado
- ✅ Ledger: Entry con `debit_base = 250.00`
- ✅ Flujo E2E: Funcional

---

## 📊 Comparación Antes/Después

| Componente | Antes | Después |
|------------|-------|---------|
| **Directorio** | `order-service/` | `sales-service/` |
| **Module** | `module order` | `module sales` |
| **Estructura** | `src/order/` | `src/sales/` |
| **Imports** | `"order/src/order/..."` | `"sales/src/sales/..."` |
| **Binary** | `order-service` | `sales-service` |
| **Container** | `order-service` | `sales-service` |
| **Kong Service** | `order-service` | `sales-service` |
| **Rutas HTTP** | `/api/v1/orders` | `/api/v1/orders` *(sin cambio)* |
| **Tablas DB** | `orders` | `orders` *(sin cambio)* |

---

## 🔒 Decisiones Técnicas

### Por qué renombrar ahora

1. ✅ Contrato de eventos v1.0 ya usa `sales.*`
2. ✅ Arquitectura ERP define dominio **Sales**
3. ✅ Ledger consume eventos de sales
4. ✅ Evita fricción cognitiva futura

### Por qué NO renombrar tablas aún

1. ❌ Requiere migraciones productivas
2. ❌ Puede romper clientes existentes
3. ❌ Fuera de alcance v0.2

**Se hará en HITO v0.3**

---

## ⚠️ Riesgos Mitigados

| Riesgo | Mitigación Aplicada |
|--------|---------------------|
| Imports rotos | sed masivo + go build |
| Contenedor no levanta | Compilación local primero |
| Flujo E2E roto | Validación completa pre-cierre |
| Deuda técnica híbrida | Rename completo, no parcial |

---

## 📝 Notas Importantes

### Rutas HTTP mantenidas

```bash
POST /api/v1/orders              # Se mantiene
POST /api/v1/orders/:id/confirm  # Se mantiene
POST /api/v1/pos/sale            # Se mantiene
```

**Motivo:** Retrocompatibilidad. Cambio en v0.3.

### Tablas DB mantenidas

```sql
orders
order_items
pos_sales
pos_sale_items
```

**Motivo:** Evitar migraciones en v0.2. Cambio en v0.3.

### Event types sin cambio

```
sales.order.confirmed  # Sin cambio
sales.pos.confirmed    # Sin cambio
```

**Motivo:** Ya alineados con arquitectura.

---

## 🚀 Comando de Ejecución

```bash
# Compilar
cd services/sales-service
go build .

# Ejecutar localmente
DB_HOST=localhost \
DB_PORT=5432 \
DB_USER=postgres \
DB_PASSWORD=postgres \
DB_NAME=order_db \
EVENTBUS_DB_HOST=localhost \
EVENTBUS_DB_PORT=5432 \
EVENTBUS_DB_USER=postgres \
EVENTBUS_DB_PASSWORD=postgres \
EVENTBUS_DB_NAME=eventbus \
PORT=8123 \
./sales

# O con Docker
docker-compose up sales-service
```

---

**Preparado por:** System Architecture Team  
**Aprobado:** Technical Lead  
**Estado:** ✅ PRODUCCIÓN-READY (para entorno dev)  
**Próximo hito:** v0.3 (Migraciones DB + Rutas)  
