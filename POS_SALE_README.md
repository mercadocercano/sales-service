# Endpoint POS Sale - Venta Directa

**Estado:** ✅ IMPLEMENTADO (07/02/2026)  
**Ubicación:** `POST /api/v1/pos/sale`  
**Servicio:** order-service (puerto 8120)

---

## 📋 Descripción

Endpoint para **venta directa POS** sin crear orden.

**Flujo:**
```
Cliente POS → POST /pos/sale → order-service → Kong → stock-service /sale
```

**Características:**
- ✅ **Un solo paso** (no requiere confirm)
- ✅ **Sin reservas** (venta inmediata)
- ✅ **Sin orden** (no crea registro en orders)
- ✅ **Stock directo**: `available ↓`, `total ↓`
- ✅ **Auditoría**: Crea `stock_entry` tipo "sale"

---

## 🔌 Endpoint

### POST /api/v1/pos/sale

**Headers:**
```http
Content-Type: application/json
X-Tenant-ID: <tenant_uuid>
Authorization: Bearer <token>  (opcional)
```

**Request Body:**
```json
{
  "variant_sku": "PROD-001",
  "quantity": 5,
  "reference": "POS-VENTA-123",     // Opcional - Se genera automático si falta
  "notes": "Venta mostrador efectivo"
}
```

**Reference Auto-generado:**
- Formato: `POS-{tenant_id_8chars}-{nanoseconds}`
- Ejemplo: `POS-123e4567-1707332400123456789`
- Garantiza unicidad por tenant y tiempo

**Response 201 Created:**
```json
{
  "entry_id": "uuid-stock-entry",
  "variant_sku": "PROD-001",
  "quantity": 5,
  "available_quantity": 95.0,
  "total_quantity": 95.0,
  "message": "POS sale registered successfully for PROD-001",
  "sale_registered_at": "2026-02-07T15:30:00Z"
}
```

**Response 409 Conflict** (stock insuficiente):
```json
{
  "error": "Insufficient stock for POS sale"
}
```

---

## 🧪 Testing

### Curl Manual

```bash
curl -X POST http://localhost:8120/api/v1/pos/sale \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 123e4567-e89b-12d3-a456-426614174003" \
  -d '{
    "variant_sku": "PROD-001",
    "quantity": 2,
    "reference": "POS-001",
    "notes": "Venta mostrador"
  }'
```

### Script Automatizado

```bash
./test-pos-sale.sh
```

---

## 🔄 Flujo Técnico

```
1. Cliente POS envía request a order-service
   POST /api/v1/pos/sale

2. order-service (POSSaleUseCase)
   - Valida request
   - Genera reference si no viene
   - Llama a stock-service vía Kong

3. stock-service
   POST /api/v1/sale
   - Valida stock disponible
   - Crea stock_entry (tipo "sale")
   - Actualiza stock_availability
     * available ↓
     * total ↓
   - Retorna respuesta

4. order-service retorna al cliente
   HTTP 201 + detalles de venta
```

---

## ⚡ Diferencia vs Order Flow

| Aspecto | POS Sale | Order Flow |
|---------|----------|------------|
| **Pasos** | 1 (sale) | 3 (create → confirm) |
| **Crea orden** | ❌ No | ✅ Sí |
| **Reserva stock** | ❌ No | ✅ Opcional |
| **Estados** | Ninguno | CREATED → CONFIRMED |
| **Latencia** | Baja | Media |
| **Uso** | POS físico | E-commerce/Backoffice |

---

## 🎯 Casos de Uso

### ✅ Cuándo usar /pos/sale

- ✅ Venta en mostrador (POS físico)
- ✅ Venta telefónica directa
- ✅ Pedido simple sin tracking
- ✅ Necesitas latencia mínima

### ❌ Cuándo NO usar /pos/sale

- ❌ E-commerce (usar order-service completo)
- ❌ Necesitas tracking de orden
- ❌ Necesitas estados intermedios
- ❌ Necesitas snapshot de precio/producto

---

## 📊 Integración con Stock Service

**Endpoint utilizado:**
```
POST /stock/api/v1/sale
```

**Ver documentación completa:**
- `services/stock-service/SALE_ENDPOINT_README.md`
- `documentation/components/stock-service.md`

---

## 🔒 Seguridad

- ✅ **X-Tenant-ID obligatorio** (aislamiento multi-tenant)
- ✅ **Authorization opcional** (validado por Kong)
- ✅ **Validación de request** (quantity > 0, sku required)
- ✅ **Kong Gateway** (rate limiting, CORS, auth)

---

## 🚀 Próximos Pasos

**Implementado:**
- ✅ Endpoint `/pos/sale`
- ✅ Use case `POSSaleUseCase`
- ✅ Cliente HTTP a stock-service
- ✅ Tests manuales

**Pendiente:**
- ⏳ Frontend POS (UI)
- ⏳ Tests E2E automatizados
- ⏳ Métricas Prometheus
- ⏳ Logs estructurados

---

**Implementado:** 07/02/2026  
**Hito:** POS-REAL-01 ✅ COMPLETADO
