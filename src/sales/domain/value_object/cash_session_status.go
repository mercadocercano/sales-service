package value_object

// CashSessionStatus es el estado de una sesión de caja. Transiciones unidireccionales
// (ADR-003 §2): open -> closing -> closed | pending_review ; pending_review -> closed.
type CashSessionStatus string

const (
	CashStatusOpen          CashSessionStatus = "open"
	CashStatusClosing       CashSessionStatus = "closing"
	CashStatusClosed        CashSessionStatus = "closed"
	CashStatusPendingReview CashSessionStatus = "pending_review"
)

func (s CashSessionStatus) String() string { return string(s) }

func (s CashSessionStatus) IsValid() bool {
	switch s {
	case CashStatusOpen, CashStatusClosing, CashStatusClosed, CashStatusPendingReview:
		return true
	default:
		return false
	}
}

// IsOpen indica si la caja admite ventas/movimientos.
func (s CashSessionStatus) IsOpen() bool { return s == CashStatusOpen }

// IsTerminal indica si la sesión ya no admite transiciones (cerrada definitivamente).
func (s CashSessionStatus) IsTerminal() bool { return s == CashStatusClosed }

// CanTransitionTo valida la máquina de estados. Es la única fuente de verdad de
// transiciones — el repositorio y los use cases la consultan, no la duplican.
func (s CashSessionStatus) CanTransitionTo(next CashSessionStatus) bool {
	switch s {
	case CashStatusOpen:
		// open puede cerrarse OK, quedar en revisión, o pasar por closing (force-reopen vuelve a open)
		return next == CashStatusClosed || next == CashStatusPendingReview || next == CashStatusClosing
	case CashStatusClosing:
		return next == CashStatusClosed || next == CashStatusPendingReview || next == CashStatusOpen
	case CashStatusPendingReview:
		return next == CashStatusClosed
	case CashStatusClosed:
		return false
	default:
		return false
	}
}
