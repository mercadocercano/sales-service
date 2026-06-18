package value_object

// CashMovementType es el tipo de un movimiento MANUAL de efectivo. El monto siempre se
// guarda positivo; el signo en el arqueo lo determina el tipo.
type CashMovementType string

const (
	CashMovementIncome     CashMovementType = "income"     // ingreso manual (+)
	CashMovementExpense    CashMovementType = "expense"    // egreso/gasto (−)
	CashMovementWithdrawal CashMovementType = "withdrawal" // retiro de efectivo (−)
	CashMovementCorrection CashMovementType = "correction" // contra-asiento de corrección (+)
)

func (t CashMovementType) String() string { return string(t) }

func (t CashMovementType) IsValid() bool {
	switch t {
	case CashMovementIncome, CashMovementExpense, CashMovementWithdrawal, CashMovementCorrection:
		return true
	default:
		return false
	}
}

// Sign devuelve +1 si el movimiento suma al efectivo esperado, −1 si resta.
func (t CashMovementType) Sign() int {
	switch t {
	case CashMovementIncome, CashMovementCorrection:
		return 1
	case CashMovementExpense, CashMovementWithdrawal:
		return -1
	default:
		return 0
	}
}
