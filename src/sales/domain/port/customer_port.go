package port

import (
	"context"

	"github.com/google/uuid"
)

// CustomerPort define las operaciones de validación y lectura de clientes
type CustomerPort interface {
	Exists(ctx context.Context, tenantID uuid.UUID, customerID uuid.UUID) (bool, error)
	// GetName resuelve el nombre de un cliente para un tenant.
	// Retorna ("", nil) cuando el cliente no existe (404) para que el llamador
	// aplique su propio fallback sin tratarlo como error técnico.
	GetName(ctx context.Context, tenantID uuid.UUID, customerID uuid.UUID) (string, error)
}
