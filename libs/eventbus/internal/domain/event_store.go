package domain

import "context"

// EventStore define el contrato para persistencia de eventos
type EventStore interface {
	// Save persiste un evento en el store (append-only)
	Save(ctx context.Context, event DomainEvent) error

	// GetUnprocessed retorna eventos no procesados por un consumidor específico
	GetUnprocessed(ctx context.Context, consumerName string, limit int) ([]DomainEvent, error)

	// MarkProcessed marca un evento como procesado por un consumidor
	MarkProcessed(ctx context.Context, eventID string, consumerName string) error

	// MarkFailed marca un evento como fallido y registra el error
	MarkFailed(ctx context.Context, eventID string, consumerName string, errorMsg string) error

	// IncrementRetry incrementa el contador de reintentos
	IncrementRetry(ctx context.Context, eventID string, consumerName string) error

	// IncrementRetryAndGet incrementa el contador y retorna el nuevo valor
	IncrementRetryAndGet(ctx context.Context, eventID string, consumerName string) (int, error)
}
