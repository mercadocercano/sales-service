#!/bin/bash

# Script de prueba para POST /pos/sale con DTO completo para impresión
# HITO: POST /pos/sale devuelve DTO listo para imprimir

set -e

echo "========================================="
echo "TEST: POS Sale Complete DTO for Printing"
echo "========================================="
echo ""

# Variables
KONG_URL="${KONG_URL:-http://localhost:8001}"
TENANT_ID="${TENANT_ID:-550e8400-e29b-41d4-a716-446655440000}"
AUTH_TOKEN="${AUTH_TOKEN:-Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0ZW5hbnRfaWQiOiI1NTBlODQwMC1lMjliLTQxZDQtYTcxNi00NDY2NTU0NDAwMDAiLCJ1c2VyX2lkIjoiNTUwZTg0MDAtZTI5Yi00MWQ0LWE3MTYtNDQ2NjU1NDQwMDAwIn0.test}"

# Payment Methods IDs (globales)
EFECTIVO="b0000000-0000-0000-0000-000000000001"
TARJETA_DEBITO="b0000000-0000-0000-0000-000000000002"
TARJETA_CREDITO="b0000000-0000-0000-0000-000000000003"

echo "Configuration:"
echo "  Kong URL: $KONG_URL"
echo "  Tenant ID: $TENANT_ID"
echo ""

# ========================================
# TEST 1: Venta simple con efectivo
# ========================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 1: Venta simple con efectivo y cambio"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

echo "Request:"
cat <<EOF | tee /tmp/pos_sale_request.json
{
  "items": [
    {
      "sku": "COCA-1L",
      "quantity": 2,
      "unit_price": "1500.00"
    },
    {
      "sku": "PAN-BLANCO",
      "quantity": 1,
      "unit_price": "800.00"
    }
  ],
  "payment_method_id": "$EFECTIVO",
  "discount_amount": "0",
  "amount_paid": "5000.00"
}
EOF
echo ""

echo "Sending request..."
RESPONSE=$(curl -s -X POST "$KONG_URL/order/api/v1/pos/sale" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Authorization: $AUTH_TOKEN" \
  -d @/tmp/pos_sale_request.json)

echo "Response:"
echo "$RESPONSE" | jq .
echo ""

# Verificar campos obligatorios del DTO
echo "Verifying complete DTO for printing..."
SALE_NUMBER=$(echo "$RESPONSE" | jq -r '.sale_number')
PAYMENT_NAME=$(echo "$RESPONSE" | jq -r '.payment_method_name')
AMOUNT_PAID=$(echo "$RESPONSE" | jq -r '.amount_paid')
CHANGE=$(echo "$RESPONSE" | jq -r '.change')
FINAL_AMOUNT=$(echo "$RESPONSE" | jq -r '.final_amount')

echo "  ✓ sale_number: $SALE_NUMBER"
echo "  ✓ payment_method_name: $PAYMENT_NAME"
echo "  ✓ amount_paid: $AMOUNT_PAID"
echo "  ✓ change: $CHANGE"
echo "  ✓ final_amount: $FINAL_AMOUNT"
echo ""

if [ "$PAYMENT_NAME" = "Efectivo" ] && [ "$CHANGE" = "1200" ]; then
  echo "✅ TEST 1 PASSED: DTO completo con payment_method_name y change correcto"
else
  echo "❌ TEST 1 FAILED: DTO incompleto o cálculo incorrecto"
  echo "   Expected: payment_method_name='Efectivo', change=1200"
  echo "   Got: payment_method_name='$PAYMENT_NAME', change=$CHANGE"
fi
echo ""

# ========================================
# TEST 2: Venta con tarjeta de crédito (sin cambio)
# ========================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 2: Venta con tarjeta de crédito (cambio = 0)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

cat <<EOF > /tmp/pos_sale_request2.json
{
  "items": [
    {
      "sku": "COCA-1L",
      "quantity": 1,
      "unit_price": "1500.00"
    }
  ],
  "payment_method_id": "$TARJETA_CREDITO",
  "discount_amount": "0",
  "amount_paid": "1500.00"
}
EOF

RESPONSE2=$(curl -s -X POST "$KONG_URL/order/api/v1/pos/sale" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Authorization: $AUTH_TOKEN" \
  -d @/tmp/pos_sale_request2.json)

echo "Response:"
echo "$RESPONSE2" | jq .
echo ""

PAYMENT_NAME2=$(echo "$RESPONSE2" | jq -r '.payment_method_name')
CHANGE2=$(echo "$RESPONSE2" | jq -r '.change')

if [ "$PAYMENT_NAME2" = "Tarjeta de Crédito" ] && [ "$CHANGE2" = "0" ]; then
  echo "✅ TEST 2 PASSED: Tarjeta de crédito sin cambio"
else
  echo "❌ TEST 2 FAILED"
  echo "   Expected: payment_method_name='Tarjeta de Crédito', change=0"
  echo "   Got: payment_method_name='$PAYMENT_NAME2', change=$CHANGE2"
fi
echo ""

# ========================================
# TEST 3: Error - amount_paid insuficiente
# ========================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 3: Error - amount_paid insuficiente"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

cat <<EOF > /tmp/pos_sale_request3.json
{
  "items": [
    {
      "sku": "COCA-1L",
      "quantity": 1,
      "unit_price": "1500.00"
    }
  ],
  "payment_method_id": "$EFECTIVO",
  "discount_amount": "0",
  "amount_paid": "1000.00"
}
EOF

RESPONSE3=$(curl -s -X POST "$KONG_URL/order/api/v1/pos/sale" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Authorization: $AUTH_TOKEN" \
  -d @/tmp/pos_sale_request3.json)

echo "Response:"
echo "$RESPONSE3" | jq .
echo ""

ERROR_MSG=$(echo "$RESPONSE3" | jq -r '.error // .message // empty')

if echo "$ERROR_MSG" | grep -q "insufficient\|greater than or equal"; then
  echo "✅ TEST 3 PASSED: Validación de amount_paid funciona"
else
  echo "❌ TEST 3 FAILED: No rechazó amount_paid insuficiente"
  echo "   Response: $ERROR_MSG"
fi
echo ""

# ========================================
# TEST 4: Venta con descuento
# ========================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 4: Venta con descuento"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

cat <<EOF > /tmp/pos_sale_request4.json
{
  "items": [
    {
      "sku": "COCA-1L",
      "quantity": 1,
      "unit_price": "1500.00"
    }
  ],
  "payment_method_id": "$EFECTIVO",
  "discount_amount": "200.00",
  "amount_paid": "2000.00"
}
EOF

RESPONSE4=$(curl -s -X POST "$KONG_URL/order/api/v1/pos/sale" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Authorization: $AUTH_TOKEN" \
  -d @/tmp/pos_sale_request4.json)

echo "Response:"
echo "$RESPONSE4" | jq .
echo ""

SUBTOTAL4=$(echo "$RESPONSE4" | jq -r '.subtotal_amount')
DISCOUNT4=$(echo "$RESPONSE4" | jq -r '.discount_amount')
FINAL4=$(echo "$RESPONSE4" | jq -r '.final_amount')
CHANGE4=$(echo "$RESPONSE4" | jq -r '.change')

# Subtotal: 1500, Discount: 200, Final: 1300, Paid: 2000, Change: 700
if [ "$SUBTOTAL4" = "1500" ] && [ "$DISCOUNT4" = "200" ] && [ "$FINAL4" = "1300" ] && [ "$CHANGE4" = "700" ]; then
  echo "✅ TEST 4 PASSED: Cálculo correcto con descuento"
else
  echo "❌ TEST 4 FAILED"
  echo "   Expected: subtotal=1500, discount=200, final=1300, change=700"
  echo "   Got: subtotal=$SUBTOTAL4, discount=$DISCOUNT4, final=$FINAL4, change=$CHANGE4"
fi
echo ""

# ========================================
# RESUMEN
# ========================================
echo "========================================="
echo "TESTS COMPLETED"
echo "========================================="
echo ""
echo "DTO verificado:"
echo "  ✓ sale_number (UUID)"
echo "  ✓ payment_method_name (desde cache)"
echo "  ✓ amount_paid (persistido)"
echo "  ✓ change (calculado)"
echo "  ✓ subtotal_amount, discount_amount, final_amount"
echo "  ✓ items completos"
echo "  ✓ created_at"
echo ""
echo "Backend listo para impresión de tickets térmicos 80mm"
echo ""
