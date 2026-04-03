package port

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// TenantInfo contiene la información básica de un tenant
type TenantInfo struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
}

// TenantPort define las operaciones de consulta de tenants
type TenantPort interface {
	GetTenant(ctx context.Context, tenantID uuid.UUID) (*TenantInfo, error)
}
