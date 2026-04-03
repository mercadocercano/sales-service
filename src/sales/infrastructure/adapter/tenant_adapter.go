package adapter

import (
	"context"

	"sales/src/sales/domain/port"
	"sales/src/sales/infrastructure/client"

	"github.com/google/uuid"
)

// TenantAdapter adapta client.TenantClient a port.TenantPort
type TenantAdapter struct {
	client *client.TenantClient
}

// NewTenantAdapter crea un nuevo adaptador
func NewTenantAdapter(c *client.TenantClient) *TenantAdapter {
	return &TenantAdapter{client: c}
}

func (a *TenantAdapter) GetTenant(ctx context.Context, tenantID uuid.UUID) (*port.TenantInfo, error) {
	tenant, err := a.client.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return &port.TenantInfo{
		ID:        tenant.ID,
		Name:      tenant.Name,
		CreatedAt: tenant.CreatedAt,
	}, nil
}
