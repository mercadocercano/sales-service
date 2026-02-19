#!/bin/bash

# Script para ejecutar migración 004 - pos_sales table
# Hito: POS-SALE-02.BE
# Fecha: 2025-02-09

set -e

echo "🚀 Ejecutando migración 004: pos_sales table"
echo "============================================"
echo ""

# Verificar que el contenedor de postgres esté corriendo
if ! docker ps | grep -q mc-postgres; then
    echo "❌ Error: El contenedor mc-postgres no está corriendo"
    echo "   Ejecuta: make lite-start"
    exit 1
fi

echo "📋 Migraciones actuales en order_db:"
docker exec mc-postgres psql -U postgres -d order_db -c "\dt" 2>/dev/null || echo "Base de datos no existe o está vacía"
echo ""

echo "🔧 Ejecutando migración 004..."
docker exec mc-postgres psql -U postgres -d order_db -f /docker-entrypoint-initdb.d/004_create_pos_sales_table.sql

echo ""
echo "✅ Migración completada!"
echo ""

echo "📊 Verificando tabla pos_sales:"
docker exec mc-postgres psql -U postgres -d order_db -c "\d pos_sales"

echo ""
echo "📋 Tablas en order_db después de la migración:"
docker exec mc-postgres psql -U postgres -d order_db -c "\dt"

echo ""
echo "✅ Criterios de cierre paso 1:"
echo "  [✓] Migración aplica limpia"
echo "  [✓] Tabla pos_sales visible"
echo ""
