package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"sales/src/sales/domain/port"

	"github.com/google/uuid"
)

// TenantMetricsPostgresRepository implementa TenantMetricsRepository
// H8: INSERT ON CONFLICT DO NOTHING para TTFS consistente
type TenantMetricsPostgresRepository struct {
	db *sql.DB
}

// NewTenantMetricsPostgresRepository crea una nueva instancia
func NewTenantMetricsPostgresRepository(db *sql.DB) port.TenantMetricsRepository {
	return &TenantMetricsPostgresRepository{db: db}
}

// RecordFirstSaleIfNew inserta solo si el tenant no tiene registro previo
// Retorna true si rows affected == 1 (primera venta real)
func (r *TenantMetricsPostgresRepository) RecordFirstSaleIfNew(ctx context.Context, tenantID uuid.UUID, firstSaleAt time.Time) (bool, error) {
	query := `
		INSERT INTO tenant_metrics (tenant_id, first_sale_at)
		VALUES ($1, $2)
		ON CONFLICT (tenant_id) DO NOTHING
	`
	result, err := r.db.ExecContext(ctx, query, tenantID, firstSaleAt)
	if err != nil {
		return false, fmt.Errorf("error recording first sale: %w", err)
	}
	rows, _ := result.RowsAffected()
	return rows == 1, nil
}
