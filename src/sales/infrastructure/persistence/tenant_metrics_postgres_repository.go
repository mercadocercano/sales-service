package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/hornosg/go-shared/infrastructure/postgres"

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
// PLAT-E25 T9: corre dentro de WithRLSInTransaction (SET LOCAL app.tenant_id) — tenant_metrics
// tiene tenant_id como PRIMARY KEY, la policy sigue siendo una comparación de igualdad simple.
func (r *TenantMetricsPostgresRepository) RecordFirstSaleIfNew(ctx context.Context, tenantID uuid.UUID, firstSaleAt time.Time) (bool, error) {
	rc := postgres.RLSContext{TenantID: tenantID.String()}

	var rows int64
	err := postgres.WithRLSInTransaction(ctx, r.db, rc, func(ctx context.Context, tx *sql.Tx) error {
		query := `
			INSERT INTO tenant_metrics (tenant_id, first_sale_at)
			VALUES ($1, $2)
			ON CONFLICT (tenant_id) DO NOTHING
		`
		result, err := tx.ExecContext(ctx, query, tenantID, firstSaleAt)
		if err != nil {
			return fmt.Errorf("error recording first sale: %w", err)
		}
		rows, _ = result.RowsAffected()
		return nil
	})
	if err != nil {
		return false, err
	}
	return rows == 1, nil
}
