#!/bin/bash
set -e

echo "🚀 EventBus Setup Script"
echo "========================"

# Colores
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Verificar PostgreSQL
echo -e "\n${YELLOW}[1/5]${NC} Verificando PostgreSQL..."
if ! command -v psql &> /dev/null; then
    echo -e "${RED}❌ PostgreSQL no está instalado${NC}"
    exit 1
fi
echo -e "${GREEN}✅ PostgreSQL encontrado${NC}"

# Verificar Go
echo -e "\n${YELLOW}[2/5]${NC} Verificando Go..."
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go no está instalado${NC}"
    exit 1
fi
GO_VERSION=$(go version | awk '{print $3}')
echo -e "${GREEN}✅ Go encontrado: ${GO_VERSION}${NC}"

# Crear .env si no existe
echo -e "\n${YELLOW}[3/5]${NC} Configurando variables de entorno..."
if [ ! -f .env ]; then
    cp .env.example .env
    echo -e "${GREEN}✅ Archivo .env creado desde .env.example${NC}"
    echo -e "${YELLOW}⚠️  Por favor, edita .env con tus credenciales de PostgreSQL${NC}"
else
    echo -e "${GREEN}✅ Archivo .env ya existe${NC}"
fi

# Cargar variables de entorno
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# Crear base de datos
echo -e "\n${YELLOW}[4/5]${NC} Creando base de datos..."
DB_EXISTS=$(psql -U $DB_USER -h $DB_HOST -p $DB_PORT -tAc "SELECT 1 FROM pg_database WHERE datname='$DB_NAME'" 2>/dev/null || echo "")

if [ "$DB_EXISTS" = "1" ]; then
    echo -e "${YELLOW}⚠️  Base de datos '$DB_NAME' ya existe${NC}"
    read -p "¿Deseas recrearla? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        dropdb -U $DB_USER -h $DB_HOST -p $DB_PORT $DB_NAME 2>/dev/null || true
        createdb -U $DB_USER -h $DB_HOST -p $DB_PORT $DB_NAME
        echo -e "${GREEN}✅ Base de datos recreada${NC}"
    fi
else
    createdb -U $DB_USER -h $DB_HOST -p $DB_PORT $DB_NAME
    echo -e "${GREEN}✅ Base de datos creada: $DB_NAME${NC}"
fi

# Ejecutar migraciones
echo -e "\n${YELLOW}[5/5]${NC} Ejecutando migraciones..."
psql -U $DB_USER -h $DB_HOST -p $DB_PORT -d $DB_NAME -f migrations/001_create_event_bus_tables.up.sql > /dev/null 2>&1
echo -e "${GREEN}✅ Migraciones ejecutadas${NC}"

# Verificar tablas
echo -e "\nVerificando tablas creadas..."
TABLES=$(psql -U $DB_USER -h $DB_HOST -p $DB_PORT -d $DB_NAME -tAc "SELECT tablename FROM pg_tables WHERE schemaname='public'" | grep event)
echo -e "${GREEN}Tablas encontradas:${NC}"
echo "$TABLES" | while read table; do
    echo "  - $table"
done

# Descargar dependencias
echo -e "\n${YELLOW}Descargando dependencias Go...${NC}"
go mod download
go mod tidy
echo -e "${GREEN}✅ Dependencias descargadas${NC}"

# Build
echo -e "\n${YELLOW}Compilando binarios...${NC}"
make build
echo -e "${GREEN}✅ Binarios compilados en bin/${NC}"

# Resumen
echo -e "\n${GREEN}════════════════════════════════════════${NC}"
echo -e "${GREEN}✅ Setup completado exitosamente!${NC}"
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo -e "\nPróximos pasos:"
echo -e "  1. Editar .env con tus credenciales (si es necesario)"
echo -e "  2. Ejecutar ejemplo publisher: ${YELLOW}make run-example-publisher${NC}"
echo -e "  3. Ejecutar ejemplo consumer:  ${YELLOW}make run-example-consumer${NC}"
echo -e "  4. Ejecutar worker genérico:   ${YELLOW}make run-worker${NC}"
echo -e "\nDocumentación:"
echo -e "  - README.md para guía de uso"
echo -e "  - ARCHITECTURE.md para detalles técnicos"
