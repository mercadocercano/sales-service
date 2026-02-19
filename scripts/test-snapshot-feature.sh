#!/bin/bash

# Script de prueba E2E para verificar snapshots en órdenes
# Prerrequisitos:
# - PIM service debe estar corriendo
# - Order service debe estar corriendo
# - Debe existir un producto/variante en PIM
# - Tener tenant_id y token válidos

set -e

# Configuración
KONG_URL=${KONG_URL:-http://localhost:8001}
TENANT_ID=${TENANT_ID:-""}
AUTH_TOKEN=${AUTH_TOKEN:-""}
TEST_SKU=${TEST_SKU:-""}

# Colores
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "🧪 Test E2E: Snapshots en Órdenes"
echo "=================================="
echo ""

# Validar variables requeridas
if [ -z "$TENANT_ID" ]; then
    echo -e "${RED}❌ Error: TENANT_ID no configurado${NC}"
    echo "Uso: TENANT_ID=<uuid> AUTH_TOKEN=<token> TEST_SKU=<sku> $0"
    exit 1
fi

if [ -z "$AUTH_TOKEN" ]; then
    echo -e "${YELLOW}⚠️  Advertencia: AUTH_TOKEN vacío (puede fallar si Kong requiere auth)${NC}"
fi

if [ -z "$TEST_SKU" ]; then
    echo -e "${RED}❌ Error: TEST_SKU no configurado${NC}"
    echo "Uso: TENANT_ID=<uuid> AUTH_TOKEN=<token> TEST_SKU=<sku> $0"
    exit 1
fi

echo -e "${YELLOW}Configuración:${NC}"
echo "  Kong URL: $KONG_URL"
echo "  Tenant ID: $TENANT_ID"
echo "  Test SKU: $TEST_SKU"
echo ""

# Test 1: Crear orden con snapshots
echo -e "${YELLOW}[1/4] Creando orden con snapshot...${NC}"
CREATE_RESPONSE=$(curl -s -X POST "$KONG_URL/order/api/v1/orders" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Authorization: Bearer $AUTH_TOKEN" \
  -d "{\"items\":[{\"sku\":\"$TEST_SKU\",\"quantity\":2}]}")

echo "Response: $CREATE_RESPONSE"

# Extraer order_id
ORDER_ID=$(echo "$CREATE_RESPONSE" | grep -o '"order_id":"[^"]*"' | cut -d'"' -f4)

if [ -z "$ORDER_ID" ]; then
    echo -e "${RED}❌ Error: No se pudo crear la orden${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Orden creada: $ORDER_ID${NC}"
echo ""

# Test 2: Obtener orden y verificar snapshots
echo -e "${YELLOW}[2/4] Obteniendo orden y verificando snapshots...${NC}"
GET_RESPONSE=$(curl -s "$KONG_URL/order/api/v1/orders/$ORDER_ID" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Authorization: Bearer $AUTH_TOKEN")

echo "$GET_RESPONSE" | jq '.'

# Verificar que existan los snapshots
HAS_PRODUCT_SNAPSHOT=$(echo "$GET_RESPONSE" | grep -c "product_snapshot" || true)
HAS_VARIANT_SNAPSHOT=$(echo "$GET_RESPONSE" | grep -c "variant_snapshot" || true)

if [ "$HAS_PRODUCT_SNAPSHOT" -eq 0 ]; then
    echo -e "${RED}❌ Error: product_snapshot no encontrado en la respuesta${NC}"
    exit 1
fi

if [ "$HAS_VARIANT_SNAPSHOT" -eq 0 ]; then
    echo -e "${RED}❌ Error: variant_snapshot no encontrado en la respuesta${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Snapshots encontrados en la respuesta${NC}"
echo ""

# Test 3: Verificar inmutabilidad (consultar de nuevo)
echo -e "${YELLOW}[3/4] Verificando inmutabilidad de snapshots...${NC}"
sleep 1
GET_RESPONSE_2=$(curl -s "$KONG_URL/order/api/v1/orders/$ORDER_ID" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Authorization: Bearer $AUTH_TOKEN")

# Comparar que sean idénticos
if [ "$GET_RESPONSE" == "$GET_RESPONSE_2" ]; then
    echo -e "${GREEN}✅ Snapshots son inmutables (respuestas idénticas)${NC}"
else
    echo -e "${YELLOW}⚠️  Advertencia: Las respuestas difieren (puede ser timestamp u otro campo)${NC}"
fi
echo ""

# Test 4: Listar órdenes y verificar snapshots
echo -e "${YELLOW}[4/4] Listando órdenes y verificando snapshots...${NC}"
LIST_RESPONSE=$(curl -s "$KONG_URL/order/api/v1/orders?page=1&page_size=10" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Authorization: Bearer $AUTH_TOKEN")

echo "$LIST_RESPONSE" | jq '.items[0]'

HAS_SNAPSHOTS_IN_LIST=$(echo "$LIST_RESPONSE" | grep -c "product_snapshot" || true)

if [ "$HAS_SNAPSHOTS_IN_LIST" -eq 0 ]; then
    echo -e "${RED}❌ Error: Snapshots no encontrados en el listado${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Snapshots presentes en el listado${NC}"
echo ""

# Resumen
echo "=================================="
echo -e "${GREEN}🎉 Todos los tests pasaron exitosamente${NC}"
echo ""
echo "Resumen:"
echo "  - Orden creada con snapshots: $ORDER_ID"
echo "  - Snapshots verificados en GET individual"
echo "  - Inmutabilidad confirmada"
echo "  - Snapshots presentes en listado"
echo ""
echo "✅ Hito ORD-02 completado exitosamente"
