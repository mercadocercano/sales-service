# 🎯 HITO: POST /pos/sale devuelve DTO listo para imprimir

**Fecha:** 2025-02-17  
**Status:** ✅ IMPLEMENTADO

---

## 📋 Resumen Ejecutivo

El endpoint `POST /api/v1/pos/sale` ahora devuelve un DTO completo con todos los datos necesarios para imprimir un ticket térmico de 80mm sin realizar cálculos financieros en el frontend.

### ✅ Criterios de Cierre Cumplidos

1. ✅ `saleNumber` - UUID completo como número de venta
2. ✅ `paymentMethodName` - Nombre legible del método de pago (desde cache)
3. ✅ `amountPaid` - Monto pagado por el cliente (persistido en DB)
4. ✅ `change` - Vuelto calculado automáticamente (persistido en DB)
5. ✅ Totales finales consistentes (subtotal, descuento, final)
6. ✅ Fecha y hora de la transacción
7. ✅ Items completos con detalles

---

## 🔧 Cambios Implementados

### 1. Migración de Base de Datos

**Archivo:** `migrations/007_add_payment_fields_to_pos_sales.sql`

```sql
ALTER TABLE pos_sales 
ADD COLUMN amount_paid DECIMAL(15,2) NOT NULL DEFAULT 0,
ADD COLUMN change DECIMAL(15,2) NOT NULL DEFAULT 0;

-- Constraints
CHECK (amount_paid >= final_amount)
CHECK (change >= 0)
```

### 2. Request DTO Ampliado

**Nuevo campo obligatorio:**

```json
{
  "items": [...],
  "payment_method_id": "uuid",
  "discount_amount": "0",
  "amount_paid": "5000.00"  // ✅ NUEVO
}
```

### 3. Response DTO Completo

```json
{
  "pos_sale_id": "uuid",
  "sale_number": "uuid-completo",           // ✅ NUEVO
  "items": [...],
  "total_items": 2,
  "subtotal_amount": "3800.00",            // ✅ Renombrado (antes: total_amount)
  "discount_amount": "0",
  "final_amount": "3800.00",
  "payment_method_id": "uuid",
  "payment_method_name": "Efectivo",       // ✅ NUEVO
  "amount_paid": "5000.00",                // ✅ NUEVO
  "change": "1200.00",                     // ✅ NUEVO
  "currency": "ARS",
  "created_at": "2025-02-17T14:32:00Z"
}
```

### 4. Cache de Payment Methods

**Implementación:**
- Cache en memoria inicializado al startup
- Conecta a `payment_method_db`
- Carga los 8 métodos globales
- Thread-safe con `sync.RWMutex`
- Fallback graceful si falla la conexión

**Archivo:** `src/order/infrastructure/cache/payment_method_cache.go`

### 5. Validaciones de Negocio

```go
// Entity validation
if amountPaid < finalAmount {
  return ErrInsufficientPayment
}

// Change calculation
change = amountPaid - finalAmount
```

### 6. Persistencia Atómica

```sql
INSERT INTO pos_sales (
  ...,
  amount_paid,
  change,
  ...
)
```

---

## 🚀 Instrucciones de Deployment

### Paso 1: Aplicar Migración

```bash
# Conectar a la base de datos order_db
docker exec -it mc-postgres psql -U postgres -d order_db

# Aplicar migración
\i /path/to/migrations/007_add_payment_fields_to_pos_sales.sql

# Verificar
\d pos_sales
```

**Verificación esperada:**
```
Column       | Type           | Nullable
-------------+----------------+---------
amount_paid  | numeric(15,2)  | NOT NULL
change       | numeric(15,2)  | NOT NULL
```

### Paso 2: Reiniciar Order Service

```bash
# Development
make dev-restart

# O manualmente
docker-compose -f docker-compose.services.yml restart order-service
```

### Paso 3: Verificar Cache de Payment Methods

**Logs esperados:**

```
✅ Conexión a payment_method_db establecida con éxito
🔄 Loading global payment methods into cache...
✅ Loaded 8 payment methods into cache
   - b0...001: Efectivo (cash)
   - b0...002: Tarjeta de Débito (debit_card)
   - b0...003: Tarjeta de Crédito (credit_card)
   ...
```

### Paso 4: Ejecutar Tests

```bash
cd services/order-service
./scripts/test-pos-sale-complete-dto.sh
```

**Output esperado:**

```
✅ TEST 1 PASSED: DTO completo con payment_method_name y change correcto
✅ TEST 2 PASSED: Tarjeta de crédito sin cambio
✅ TEST 3 PASSED: Validación de amount_paid funciona
✅ TEST 4 PASSED: Cálculo correcto con descuento
```

---

## 🧪 Ejemplos de Uso

### Ejemplo 1: Venta Simple con Efectivo

**Request:**

```bash
curl -X POST http://localhost:8001/order/api/v1/pos/sale \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "items": [
      {
        "sku": "COCA-1L",
        "quantity": 2,
        "unit_price": "1500.00"
      }
    ],
    "payment_method_id": "b0000000-0000-0000-0000-000000000001",
    "discount_amount": "0",
    "amount_paid": "5000.00"
  }'
```

**Response:**

```json
{
  "pos_sale_id": "123e4567-e89b-12d3-a456-426614174000",
  "sale_number": "123e4567-e89b-12d3-a456-426614174000",
  "items": [
    {
      "item_id": "uuid",
      "sku": "COCA-1L",
      "product_name": "COCA-1L",
      "quantity": 2,
      "unit_price": "1500",
      "subtotal": "3000",
      "stock_entry_id": "uuid"
    }
  ],
  "total_items": 1,
  "subtotal_amount": "3000",
  "discount_amount": "0",
  "final_amount": "3000",
  "payment_method_id": "b0000000-0000-0000-0000-000000000001",
  "payment_method_name": "Efectivo",
  "amount_paid": "5000",
  "change": "2000",
  "currency": "ARS",
  "created_at": "2025-02-17T14:32:00Z"
}
```

### Ejemplo 2: Error - Pago Insuficiente

**Request:**

```json
{
  "items": [...],
  "payment_method_id": "b0...001",
  "amount_paid": "1000.00"  // Menor que final_amount
}
```

**Response (400):**

```json
{
  "error": "amount_paid must be greater than or equal to final_amount"
}
```

---

## 📊 Tabla de Payment Methods Globales

| UUID | Code | Name |
|------|------|------|
| `b0...001` | `cash` | Efectivo |
| `b0...002` | `debit_card` | Tarjeta de Débito |
| `b0...003` | `credit_card` | Tarjeta de Crédito |
| `b0...004` | `bank_transfer` | Transferencia Bancaria |
| `b0...005` | `mercadopago` | Mercado Pago |
| `b0...006` | `crypto` | Criptomonedas |
| `b0...007` | `on_account` | Cuenta Corriente |
| `b0...008` | `check` | Cheque |

---

## 🛡️ Validaciones Implementadas

### Nivel Request (Gin Binding)

- ✅ `amount_paid` es requerido
- ✅ `items` mínimo 1 item
- ✅ `payment_method_id` es requerido

### Nivel Domain (Entity)

- ✅ `amount_paid >= final_amount` (ErrInsufficientPayment)
- ✅ `change = amount_paid - final_amount >= 0`
- ✅ `discount_amount >= 0`

### Nivel Database (Constraints)

- ✅ `CHECK (amount_paid >= final_amount)`
- ✅ `CHECK (change >= 0)`

---

## 🔒 Decisiones Técnicas Clave

### ¿Por qué cache en memoria en lugar de HTTP call?

**Decisión:** Cache en memoria de payment methods  
**Razón:**
- Sin latencia adicional
- Sin dependencia runtime entre servicios
- Datos determinísticos y estables (8 métodos globales)
- Más simple y robusto

**Alternativa descartada:** HTTP call a payment-method-service
- Agrega latencia a cada venta
- Crea dependencia runtime
- Complejidad innecesaria para datos estáticos

### ¿Por qué UUID como saleNumber?

**Decisión:** Usar `PosSaleID.String()` como `sale_number`  
**Razón:**
- Único y determinístico
- No requiere contador secuencial
- Evita race conditions en multi-tenant
- Más simple para MVP

**Futuro:** Sistema de numeración fiscal (Fase 2)

### ¿Por qué persistir amount_paid y change?

**Decisión:** Persistir en DB, no calcular en frontend  
**Razón:**
- Datos transaccionales financieros
- Necesarios para reimpresión
- Auditoría completa
- Cumple estándares POS

**Rechazado:** Calcular en frontend
- Viola principios transaccionales
- No auditable
- Posible manipulación

---

## 📝 Próximos Pasos

### Fase Actual: ✅ Backend Completo

- [x] Migración DB
- [x] DTO Request ampliado
- [x] DTO Response completo
- [x] Cache payment methods
- [x] Validaciones
- [x] Tests

### Fase Siguiente: Frontend Ticket

- [ ] Componente React ThermalTicket
- [ ] Estilos CSS optimizados 80mm
- [ ] Lógica de impresión automática
- [ ] Fallback manual si falla
- [ ] Integración con flujo POS

---

## 🎯 Contrato Final del Endpoint

### POST /api/v1/pos/sale

**Request:**

```typescript
interface POSSaleRequest {
  items: Array<{
    sku: string;
    quantity: number;
    unit_price: string; // decimal
  }>;
  payment_method_id: string; // UUID
  discount_amount?: string;  // decimal, default: 0
  amount_paid: string;       // decimal, REQUIRED
  currency?: string;         // default: "ARS"
  customer_id?: string;      // UUID, optional
  notes?: string;
}
```

**Response 200:**

```typescript
interface POSSaleResponse {
  pos_sale_id: string;
  sale_number: string;
  items: Array<{
    item_id: string;
    sku: string;
    product_name: string;
    quantity: number;
    unit_price: string;
    subtotal: string;
    stock_entry_id: string;
  }>;
  total_items: number;
  subtotal_amount: string;
  discount_amount: string;
  final_amount: string;
  payment_method_id: string;
  payment_method_name: string;  // "Efectivo", "Tarjeta de Crédito", etc.
  amount_paid: string;
  change: string;
  currency: string;
  customer_id?: string;
  created_at: string; // ISO 8601
}
```

**Errors:**

- `400` - amount_paid insuficiente
- `400` - Validación de request
- `500` - Error de stock o persistencia

---

## ✅ Hito Cerrado

**Criterio verificable:** ✅ Backend entrega DTO completo sin cálculos financieros en frontend

**Siguiente hito:** Frontend imprime ticket térmico 80mm con `window.print()`

---

**Autor:** Sistema  
**Fecha:** 2025-02-17  
**Revisión:** v1.0
