package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"

	"github.com/hornosg/go-shared/infrastructure/postgres"

	"sales/src/sales/domain/entity"
	"sales/src/sales/domain/port"
	"sales/src/sales/domain/value_object"
)

// CashRegisterPostgresRepository implementa port.CashRegisterRepository.
// Maneja efectivo real (L4): toda mutación dependiente del estado usa BeginTx +
// SELECT ... FOR UPDATE y escribe la auditoría en la misma transacción.
type CashRegisterPostgresRepository struct {
	db *sql.DB
}

func NewCashRegisterPostgresRepository(db *sql.DB) port.CashRegisterRepository {
	return &CashRegisterPostgresRepository{db: db}
}

const cashSessionColumns = `
	id, tenant_id, point_of_sale_id, status, opened_by, opening_amount, opened_at,
	review_threshold, closed_by, counted_amount, expected_amount, difference, closed_at,
	approved_by, approved_at, version, created_at, updated_at`

type rowScanner interface {
	Scan(dest ...interface{}) error
}

type rowQueryer interface {
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

func scanCashSession(row rowScanner) (*entity.CashRegisterSession, error) {
	var s entity.CashRegisterSession
	var status string
	var closedBy, approvedBy uuid.NullUUID
	var counted, expected, diff decimal.NullDecimal
	var closedAt, approvedAt sql.NullTime

	err := row.Scan(
		&s.ID, &s.TenantID, &s.PointOfSaleID, &status, &s.OpenedBy, &s.OpeningAmount, &s.OpenedAt,
		&s.ReviewThreshold, &closedBy, &counted, &expected, &diff, &closedAt,
		&approvedBy, &approvedAt, &s.Version, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	s.Status = value_object.CashSessionStatus(status)
	if closedBy.Valid {
		s.ClosedBy = &closedBy.UUID
	}
	if approvedBy.Valid {
		s.ApprovedBy = &approvedBy.UUID
	}
	if counted.Valid {
		s.CountedAmount = &counted.Decimal
	}
	if expected.Valid {
		s.ExpectedAmount = &expected.Decimal
	}
	if diff.Valid {
		s.Difference = &diff.Decimal
	}
	if closedAt.Valid {
		s.ClosedAt = &closedAt.Time
	}
	if approvedAt.Valid {
		s.ApprovedAt = &approvedAt.Time
	}
	return &s, nil
}

// OpenSession inserta la sesión + asiento de auditoría 'opened' en una transacción.
func (r *CashRegisterPostgresRepository) OpenSession(ctx context.Context, s *entity.CashRegisterSession) error {
	rc := postgres.RLSContext{TenantID: s.TenantID.String()}
	return postgres.WithRLSInTransaction(ctx, r.db, rc, func(ctx context.Context, tx *sql.Tx) error {
		const q = `
			INSERT INTO cash_register_sessions
				(id, tenant_id, point_of_sale_id, status, opened_by, opening_amount, opened_at,
				 review_threshold, version, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`
		_, err := tx.ExecContext(ctx, q,
			s.ID, s.TenantID, s.PointOfSaleID, s.Status.String(), s.OpenedBy, s.OpeningAmount, s.OpenedAt,
			s.ReviewThreshold, s.Version, s.CreatedAt, s.UpdatedAt,
		)
		if err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == "23505" {
				return entity.ErrCashSessionAlreadyOpen
			}
			return fmt.Errorf("error inserting cash session: %w", err)
		}

		return insertAuditTx(ctx, tx, entity.CashAuditEntry{
			TenantID: s.TenantID, SessionID: s.ID, Action: entity.CashAuditOpened,
			UserID: s.OpenedBy, PointOfSaleID: s.PointOfSaleID, OpeningAmount: &s.OpeningAmount,
		})
	})
}

func (r *CashRegisterPostgresRepository) GetOpenByTerminal(ctx context.Context, tenantID uuid.UUID, pointOfSaleID string) (*entity.CashRegisterSession, error) {
	rc := postgres.RLSContext{TenantID: tenantID.String()}
	var s *entity.CashRegisterSession
	err := postgres.WithRLSInTransaction(ctx, r.db, rc, func(ctx context.Context, tx *sql.Tx) error {
		q := `SELECT ` + cashSessionColumns + ` FROM cash_register_sessions
			WHERE tenant_id = $1 AND point_of_sale_id = $2 AND status = 'open'`
		var err error
		s, err = scanCashSession(tx.QueryRowContext(ctx, q, tenantID, pointOfSaleID))
		if errors.Is(err, sql.ErrNoRows) {
			return entity.ErrCashSessionNotFound
		}
		if err != nil {
			return fmt.Errorf("error querying open session: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *CashRegisterPostgresRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.CashRegisterSession, error) {
	rc := postgres.RLSContext{TenantID: tenantID.String()}
	var s *entity.CashRegisterSession
	err := postgres.WithRLSInTransaction(ctx, r.db, rc, func(ctx context.Context, tx *sql.Tx) error {
		q := `SELECT ` + cashSessionColumns + ` FROM cash_register_sessions WHERE id = $1 AND tenant_id = $2`
		var err error
		s, err = scanCashSession(tx.QueryRowContext(ctx, q, id, tenantID))
		if errors.Is(err, sql.ErrNoRows) {
			return entity.ErrCashSessionNotFound
		}
		if err != nil {
			return fmt.Errorf("error querying session: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *CashRegisterPostgresRepository) ComputeExpected(ctx context.Context, s *entity.CashRegisterSession, cashPaymentMethodID uuid.UUID) (decimal.Decimal, error) {
	rc := postgres.RLSContext{TenantID: s.TenantID.String()}
	var expected decimal.Decimal
	err := postgres.WithRLSInTransaction(ctx, r.db, rc, func(ctx context.Context, tx *sql.Tx) error {
		var err error
		expected, err = computeExpected(ctx, tx, s, cashPaymentMethodID)
		return err
	})
	if err != nil {
		return decimal.Zero, err
	}
	return expected, nil
}

// computeExpected: opening + Σ(ventas efectivo final_amount) + Σ(ingresos) − Σ(egresos/retiros).
// Usa final_amount (no amount_paid − change) para no restar el vuelto dos veces (ADR-003 §5).
func computeExpected(ctx context.Context, q rowQueryer, s *entity.CashRegisterSession, cashPaymentMethodID uuid.UUID) (decimal.Decimal, error) {
	cashSales := decimal.Zero
	if cashPaymentMethodID != uuid.Nil {
		const qSales = `
			SELECT COALESCE(SUM(final_amount), 0) FROM pos_sales
			WHERE cash_register_session_id = $1 AND tenant_id = $2 AND payment_method_id = $3`
		if err := q.QueryRowContext(ctx, qSales, s.ID, s.TenantID, cashPaymentMethodID).Scan(&cashSales); err != nil {
			return decimal.Zero, fmt.Errorf("error summing cash sales: %w", err)
		}
	}

	movementsNet := decimal.Zero
	const qMov = `
		SELECT COALESCE(SUM(
			CASE WHEN type IN ('income','correction') THEN amount
			     WHEN type IN ('expense','withdrawal') THEN -amount
			     ELSE 0 END), 0)
		FROM cash_movements WHERE cash_register_session_id = $1 AND tenant_id = $2`
	if err := q.QueryRowContext(ctx, qMov, s.ID, s.TenantID).Scan(&movementsNet); err != nil {
		return decimal.Zero, fmt.Errorf("error summing movements: %w", err)
	}

	return s.OpeningAmount.Add(cashSales).Add(movementsNet), nil
}

func (r *CashRegisterPostgresRepository) RegisterMovement(ctx context.Context, mv *entity.CashMovement) (*entity.CashRegisterSession, error) {
	rc := postgres.RLSContext{TenantID: mv.TenantID.String()}
	var s *entity.CashRegisterSession
	err := postgres.WithRLSInTransaction(ctx, r.db, rc, func(ctx context.Context, tx *sql.Tx) error {
		var err error
		s, err = lockSessionTx(ctx, tx, mv.TenantID, mv.SessionID)
		if err != nil {
			return err
		}
		if !s.Status.IsOpen() {
			return entity.ErrCashSessionNotOpen
		}

		const qIns = `
			INSERT INTO cash_movements (id, tenant_id, cash_register_session_id, type, amount, reason, created_by, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`
		if _, err := tx.ExecContext(ctx, qIns,
			mv.ID, mv.TenantID, mv.SessionID, mv.Type.String(), mv.Amount, mv.Reason, mv.CreatedBy, mv.CreatedAt,
		); err != nil {
			return fmt.Errorf("error inserting movement: %w", err)
		}

		if err := bumpVersionTx(ctx, tx, s); err != nil {
			return err
		}

		return insertAuditTx(ctx, tx, entity.CashAuditEntry{
			TenantID: mv.TenantID, SessionID: mv.SessionID, Action: entity.CashAuditMovementRegistered,
			UserID: mv.CreatedBy, PointOfSaleID: s.PointOfSaleID,
			MovementType: mv.Type.String(), MovementAmount: &mv.Amount, Detail: mv.Reason,
		})
	})
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *CashRegisterPostgresRepository) CloseSession(ctx context.Context, tenantID, sessionID, closedBy, cashPaymentMethodID uuid.UUID, counted decimal.Decimal) (*entity.CashRegisterSession, error) {
	rc := postgres.RLSContext{TenantID: tenantID.String()}
	var s *entity.CashRegisterSession
	err := postgres.WithRLSInTransaction(ctx, r.db, rc, func(ctx context.Context, tx *sql.Tx) error {
		var err error
		s, err = lockSessionTx(ctx, tx, tenantID, sessionID)
		if err != nil {
			return err
		}

		expected, err := computeExpected(ctx, tx, s, cashPaymentMethodID)
		if err != nil {
			return err
		}

		// Regla de dominio: setea snapshot y decide closed vs pending_review por umbral.
		if err := s.Close(closedBy, counted, expected); err != nil {
			return err
		}

		const qUpd = `
			UPDATE cash_register_sessions
			SET status=$1, closed_by=$2, counted_amount=$3, expected_amount=$4, difference=$5,
			    closed_at=$6, version=version+1, updated_at=NOW()
			WHERE id=$7 AND tenant_id=$8 AND version=$9`
		res, err := tx.ExecContext(ctx, qUpd,
			s.Status.String(), s.ClosedBy, s.CountedAmount, s.ExpectedAmount, s.Difference,
			s.ClosedAt, s.ID, s.TenantID, s.Version,
		)
		if err != nil {
			return fmt.Errorf("error updating session on close: %w", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return entity.ErrCashSessionConflict
		}
		s.Version++

		return insertAuditTx(ctx, tx, entity.CashAuditEntry{
			TenantID: s.TenantID, SessionID: s.ID, Action: entity.CashAuditClosed,
			UserID: closedBy, PointOfSaleID: s.PointOfSaleID,
			ExpectedAmount: s.ExpectedAmount, CountedAmount: s.CountedAmount, Difference: s.Difference,
			Detail: "status=" + s.Status.String(),
		})
	})
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *CashRegisterPostgresRepository) ApproveReview(ctx context.Context, tenantID, sessionID, approverID uuid.UUID) (*entity.CashRegisterSession, error) {
	rc := postgres.RLSContext{TenantID: tenantID.String()}
	var s *entity.CashRegisterSession
	err := postgres.WithRLSInTransaction(ctx, r.db, rc, func(ctx context.Context, tx *sql.Tx) error {
		var err error
		s, err = lockSessionTx(ctx, tx, tenantID, sessionID)
		if err != nil {
			return err
		}

		// Regla de dominio: valida pending_review + separación de funciones (aprobador≠operador).
		if err := s.ApproveReview(approverID); err != nil {
			return err
		}

		const qUpd = `
			UPDATE cash_register_sessions
			SET status=$1, approved_by=$2, approved_at=$3, version=version+1, updated_at=NOW()
			WHERE id=$4 AND tenant_id=$5 AND version=$6`
		res, err := tx.ExecContext(ctx, qUpd,
			s.Status.String(), s.ApprovedBy, s.ApprovedAt, s.ID, s.TenantID, s.Version,
		)
		if err != nil {
			return fmt.Errorf("error updating session on approve: %w", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return entity.ErrCashSessionConflict
		}
		s.Version++

		return insertAuditTx(ctx, tx, entity.CashAuditEntry{
			TenantID: s.TenantID, SessionID: s.ID, Action: entity.CashAuditReviewApproved,
			UserID: approverID, PointOfSaleID: s.PointOfSaleID,
			ExpectedAmount: s.ExpectedAmount, CountedAmount: s.CountedAmount, Difference: s.Difference,
		})
	})
	if err != nil {
		return nil, err
	}
	return s, nil
}

// lockSessionTx bloquea la fila de sesión (FOR UPDATE) y la carga, filtrando por tenant.
func lockSessionTx(ctx context.Context, tx *sql.Tx, tenantID, sessionID uuid.UUID) (*entity.CashRegisterSession, error) {
	q := `SELECT ` + cashSessionColumns + ` FROM cash_register_sessions
		WHERE id = $1 AND tenant_id = $2 FOR UPDATE`
	s, err := scanCashSession(tx.QueryRowContext(ctx, q, sessionID, tenantID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, entity.ErrCashSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error locking session: %w", err)
	}
	return s, nil
}

func bumpVersionTx(ctx context.Context, tx *sql.Tx, s *entity.CashRegisterSession) error {
	const q = `UPDATE cash_register_sessions SET version=version+1, updated_at=NOW() WHERE id=$1 AND version=$2`
	res, err := tx.ExecContext(ctx, q, s.ID, s.Version)
	if err != nil {
		return fmt.Errorf("error bumping version: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return entity.ErrCashSessionConflict
	}
	s.Version++
	return nil
}

func insertAuditTx(ctx context.Context, tx *sql.Tx, e entity.CashAuditEntry) error {
	const q = `
		INSERT INTO cash_audit_log
			(tenant_id, cash_register_session_id, action, user_id, point_of_sale_id,
			 opening_amount, expected_amount, counted_amount, difference, movement_type, movement_amount, detail)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`
	_, err := tx.ExecContext(ctx, q,
		e.TenantID, e.SessionID, e.Action, e.UserID, nullStr(e.PointOfSaleID),
		nullDec(e.OpeningAmount), nullDec(e.ExpectedAmount), nullDec(e.CountedAmount), nullDec(e.Difference),
		nullStr(e.MovementType), nullDec(e.MovementAmount), nullStr(e.Detail),
	)
	if err != nil {
		return fmt.Errorf("error inserting cash audit: %w", err)
	}
	return nil
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullDec(d *decimal.Decimal) interface{} {
	if d == nil {
		return nil
	}
	return *d
}
