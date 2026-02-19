# Guía de Pruebas POS-SALE-02

## Resumen

Flujo completo de venta POS con cliente y método de pago.

**Hito**: POS-SALE-02  
**Fecha**: 2026-02-09  

---

## Componentes implementados

### Backend

1. **customer-service** (puerto 8130)
   - `GET /api/v1/customers` - Lista clientes del tenant
   - 6 clientes de ejemplo pre-cargados

2. **payment-method-service** (puerto 8140)
   - `GET /api/v1/payment-methods` - Lista métodos de pago
   - 8 métodos globales (efectivo, débito, crédito, etc.)

3. **order-service** (puerto 8120)
   - `POST /api/v1/pos/sale` - Venta POS extendida
   - `GET /api/v1/pos/sales` - Reporte de pos_sales

### Frontend (backoffice-admin)

1. **Formulario POS** (`/pos`)
   - Dropdown cliente (opcional, default: "Consumidor final")
   - Dropdown método de pago (obligatorio)
   - Input monto total (obligatorio)

2. **Reporte POS** (`/pos/report`)
   - Columnas nuevas: Cliente, Método de pago
   - Fusiona stock sales + pos_sales

### Kong Gateway

Rutas configuradas:
- `/customers` → customer-service:8080
- `/payment-methods` → payment-method-service:8080
- `/orders` → order-service:8080

---

## Pasos para probar

### 1. Levantar servicios

```bash
cd /Users/hornosg/MyProjects/saas-mt

# Opción A: Lite (recomendado - ~2GB)
make lite-start

# Opción B: Completo
make dev-start
```

Verificar que los servicios estén corriendo:

```bash
make lite-status
```

Deberías ver:
- ✅ Customer (8130)
- ✅ Payment Method (8140)
- ✅ Kong Gateway (8001)
- ✅ Order Service (via docker ps)

### 2. Reconstruir order-service (si es necesario)

Si `order-service` tiene una imagen antigua:

```bash
# Detener
docker-compose -f docker-compose.infra-minimal.yml -f docker-compose.services.yml stop order-service

# Reconstruir
docker-compose -f docker-compose.infra-minimal.yml -f docker-compose.services.yml build order-service

# Iniciar
docker-compose -f docker-compose.infra-minimal.yml -f docker-compose.services.yml up -d order-service
```

### 3. Ejecutar script de pruebas automatizado

```bash
./scripts/test-pos-sale-02-complete.sh
```

El script prueba:
1. GET /customers
2. GET /payment-methods
3. POST /pos/sale (con cliente y método de pago)
4. GET /pos/sales (reporte)
5. GET /stock/sales (reporte actual)

### 4. Prueba manual vía curl

#### a) Listar clientes

```bash
TENANT="123e4567-e89b-12d3-a456-426614174003"

curl "http://localhost:8001/customers/api/v1/customers?page=1&page_size=5" \
  -H "X-Tenant-ID: $TENANT"
```

#### b) Listar métodos de pago

```bash
curl "http://localhost:8001/payment-methods/api/v1/payment-methods?active_only=true" \
  -H "X-Tenant-ID: $TENANT"
```

#### c) Crear venta POS

```bash
# Copiar un customer_id y payment_method_id de las respuestas anteriores

CUSTOMER_ID="a0000000-0000-0000-0000-000000000001"
PAYMENT_ID="b0000000-0000-0000-0000-000000000001"

curl -X POST "http://localhost:8001/orders/api/v1/pos/sale" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{
    "variant_sku": "TEST-SKU-001",
    "quantity": 2,
    "customer_id": "'$CUSTOMER_ID'",
    "payment_method_id": "'$PAYMENT_ID'",
    "total_amount": 2500.00,
    "currency": "ARS"
  }'
```

Respuesta esperada:

```json
{
  "pos_sale_id": "uuid",
  "stock_entry_id": "uuid",
  "customer_id": "uuid",
  "payment_method_id": "uuid",
  "total_amount": 2500.00,
  "currency": "ARS",
  "variant_sku": "TEST-SKU-001",
  "quantity": 2
}
```

#### d) Ver reporte pos_sales

```bash
curl "http://localhost:8001/orders/api/v1/pos/sales" \
  -H "X-Tenant-ID: $TENANT"
```

### 5. Prueba desde el frontend

1. Iniciar backoffice:

```bash
cd services/backoffice-admin
npm run dev
```

2. Abrir en navegador: http://localhost:3000/pos

3. Realizar venta:
   - Ingresar SKU (ej: TEST-001)
   - Seleccionar cliente (o dejar "Consumidor final")
   - Seleccionar método de pago
   - Ingresar monto total
   - Click en "Vender"

4. Ver reporte: http://localhost:3000/pos/report
   - Verificar columnas Cliente y Método de pago

---

## Validaciones

### ✅ Casos que DEBEN funcionar

1. Venta con cliente y método de pago
2. Venta sin cliente (consumidor final) pero con método de pago
3. El reporte muestra cliente y método de pago

### ❌ Casos que DEBEN fallar

1. Venta sin método de pago → Error 400
2. Venta con método de pago inválido → Error 400/404
3. Venta sin monto total → Error 400

---

## Troubleshooting

### Error: "name resolution failed" en Kong

Kong no puede resolver `order-service`. Verificar:

```bash
# 1. Verificar que order-service esté corriendo
docker ps | grep order-service

# 2. Verificar que estén en la misma red
docker network inspect saas-mt_saas-network | grep -E "order-service|kong"

# 3. Si no están en la misma red, reiniciar con lite-start
make lite-stop
make lite-start
```

### Error: "404 page not found" en order-service

La imagen de order-service está desactualizada. Rebuild:

```bash
docker-compose -f docker-compose.infra-minimal.yml \
  -f docker-compose.services.yml \
  build --no-cache order-service

docker-compose -f docker-compose.infra-minimal.yml \
  -f docker-compose.services.yml \
  up -d order-service
```

### Error: "connection refused" en puertos 8130/8140

Los servicios no exponen puertos al host. Verifica el compose usado:

```bash
# Ver configuración actual
docker ps --format "table {{.Names}}\t{{.Ports}}"

# Si no ves 0.0.0.0:8130->8080, reiniciar con lite-start
make lite-restart
```

---

## Arquitectura del flujo

```
Browser (backoffice)
  ↓ /api/customers
  ↓ /api/payment-methods
  ↓ /api/pos/sale
Next.js API Routes
  ↓ http://localhost:8001/...
Kong Gateway (8001)
  ├→ /customers → customer-service:8080
  ├→ /payment-methods → payment-method-service:8080
  └→ /orders → order-service:8080
       ├→ POST /pos/sale
       │   ├→ Valida customer_id (si no es null)
       │   ├→ Valida payment_method_id (obligatorio)
       │   ├→ Llama stock-service vía Kong
       │   └→ Crea pos_sale en order_db
       └→ GET /pos/sales
           └→ Lista pos_sales del tenant
```

---

## Siguientes pasos

1. **Enriquecer reporte**: Resolver nombres de cliente/método en backend
2. **Validaciones**: Verificar que customer_id y payment_method_id existan
3. **Multi-item**: Soporte para múltiples productos en una venta
4. **Descuentos**: Campo de descuento en pos_sale
5. **Impresión**: Endpoint para generar ticket PDF

---

**Última actualización**: 2026-02-09  
**Autor**: Claude (implementación POS-SALE-02)
