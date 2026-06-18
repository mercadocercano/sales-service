package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"sales/src/sales/domain/entity"
)

// CashRegisterRepository orquesta la persistencia transaccional de la caja (ADR-003).
// Toda mutación que dependa del estado actual usa BeginTx + SELECT ... FOR UPDATE sobre
// la fila de sesión, revalida el estado dentro de la tx, y escribe el asiento de
// auditoría en la MISMA transacción (atomicidad del audit, gate #5).
type CashRegisterRepository interface {
	// OpenSession inserta una sesión nueva + audit(opened). El índice único parcial
	// garantiza una sola caja abierta por terminal: una violación se mapea a
	// entity.ErrCashSessionAlreadyOpen.
	OpenSession(ctx context.Context, session *entity.CashRegisterSession) error

	// GetOpenByTerminal devuelve la caja abierta de una terminal del tenant, o
	// entity.ErrCashSessionNotFound si no hay ninguna.
	GetOpenByTerminal(ctx context.Context, tenantID uuid.UUID, pointOfSaleID string) (*entity.CashRegisterSession, error)

	// GetByID devuelve la sesión filtrada por tenant; una de otro tenant ⇒
	// entity.ErrCashSessionNotFound (no se distingue, no se filtra info).
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.CashRegisterSession, error)

	// ComputeExpected calcula el efectivo esperado server-side para el arqueo (vista de
	// detalle / cierre): opening + Σ(ventas efectivo por final_amount) + Σ(ingresos) −
	// Σ(egresos/retiros). cashPaymentMethodID identifica las ventas en efectivo.
	ComputeExpected(ctx context.Context, session *entity.CashRegisterSession, cashPaymentMethodID uuid.UUID) (decimal.Decimal, error)

	// RegisterMovement bloquea la sesión, revalida que esté abierta, inserta el
	// movimiento (append-only) + audit, y devuelve la sesión. Falla con
	// entity.ErrCashSessionNotOpen si no está abierta.
	RegisterMovement(ctx context.Context, mv *entity.CashMovement) (*entity.CashRegisterSession, error)

	// CloseSession cierra con arqueo en una sola tx: lock, revalida abierta, calcula
	// expected, aplica la regla de dominio (closed vs pending_review por umbral),
	// persiste el snapshot + audit(closed). Devuelve la sesión resultante.
	CloseSession(ctx context.Context, tenantID, sessionID, closedBy, cashPaymentMethodID uuid.UUID, counted decimal.Decimal) (*entity.CashRegisterSession, error)

	// ApproveReview aprueba un cierre en pending_review (lock, valida estado +
	// separación de funciones en la entidad, persiste + audit(review_approved)).
	ApproveReview(ctx context.Context, tenantID, sessionID, approverID uuid.UUID) (*entity.CashRegisterSession, error)
}
