package port

import (
	"context"

	"github.com/shopspring/decimal"
)

// CreditApplication contiene el resultado de aplicar un crédito
type CreditApplication struct {
	ApplicationID string
	CustomerID    string
}

// CreditRepository define las operaciones de persistencia de créditos
type CreditRepository interface {
	CreateCredit(ctx context.Context, tenantID, customerID string, amount decimal.Decimal) (creditID string, err error)
	ApplyCredit(ctx context.Context, tenantID, creditID, orderID string, amount decimal.Decimal) (*CreditApplication, error)
}
