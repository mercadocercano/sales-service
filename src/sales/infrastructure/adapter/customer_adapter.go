package adapter

import (
	"context"

	"sales/src/sales/infrastructure/client"

	"github.com/google/uuid"
)

// CustomerAdapter adapta client.CustomerClient a port.CustomerPort
type CustomerAdapter struct {
	client *client.CustomerClient
}

// NewCustomerAdapter crea un nuevo adaptador
func NewCustomerAdapter(c *client.CustomerClient) *CustomerAdapter {
	return &CustomerAdapter{client: c}
}

func (a *CustomerAdapter) Exists(ctx context.Context, tenantID uuid.UUID, customerID uuid.UUID) (bool, error) {
	return a.client.Exists(ctx, tenantID, customerID)
}
