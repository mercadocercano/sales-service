package port

import (
	"context"

	"github.com/google/uuid"
)

// CustomerPort define las operaciones de validación de clientes
type CustomerPort interface {
	Exists(ctx context.Context, tenantID uuid.UUID, customerID uuid.UUID) (bool, error)
}
