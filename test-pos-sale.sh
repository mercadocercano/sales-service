#!/bin/bash
# Script de prueba rápida para endpoint POS /sale

set -e

echo "🛒 Test POS Sale - Order Service"
echo "================================="
echo ""

# Variables
TENANT_ID="123e4567-e89b-12d3-a456-426614174003"
ORDER_SERVICE_URL="http://localhost:8120"
VARIANT_SKU="TEST-SKU-001"
QUANTITY=5

echo "📋 Configuración:"
echo "  Tenant ID: $TENANT_ID"
echo "  Order Service: $ORDER_SERVICE_URL"
echo "  SKU: $VARIANT_SKU"
echo "  Quantity: $QUANTITY"
echo ""

echo "🔹 Ejecutando venta POS..."
RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X POST "$ORDER_SERVICE_URL/api/v1/pos/sale" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d "{
    \"variant_sku\": \"$VARIANT_SKU\",
    \"quantity\": $QUANTITY,
    \"reference\": \"POS-TEST-$(date +%s)\",
    \"notes\": \"Test venta POS desde script\"
  }")

HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS" | cut -d':' -f2)
BODY=$(echo "$RESPONSE" | sed '/HTTP_STATUS/d')

echo ""
echo "📊 Respuesta:"
echo "  HTTP Status: $HTTP_STATUS"
echo "  Body:"
echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"

echo ""
if [ "$HTTP_STATUS" = "201" ] || [ "$HTTP_STATUS" = "200" ]; then
  echo "✅ VENTA POS EXITOSA"
else
  echo "❌ ERROR EN VENTA POS"
  exit 1
fi
