#!/bin/bash

# Script para ejecutar la migración 003 - Agregar snapshots a order_items
# Uso: ./scripts/run-migration-003.sh

set -e

echo "🚀 Ejecutando migración 003: Agregar snapshots a order_items"

# Variables de entorno (ajustar según necesidad)
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-postgres}
DB_PASSWORD=${DB_PASSWORD:-postgres}
DB_NAME=${DB_NAME:-order_db}

# Ejecutar migración
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f migrations/003_add_snapshots_to_order_items.sql

echo "✅ Migración 003 ejecutada exitosamente"
echo ""
echo "Verificando estructura de la tabla order_items:"
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "\d order_items"
