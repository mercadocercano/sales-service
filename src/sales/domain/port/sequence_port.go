package port

import "context"

// SequencePort define la operación de obtención de números de secuencia
type SequencePort interface {
	NextNumber(ctx context.Context, tenantID, documentType string) (int, error)
}
