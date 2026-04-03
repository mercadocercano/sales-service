package port

import "context"

// EventPublisher define la operación de publicación de eventos de dominio
type EventPublisher interface {
	Execute(ctx context.Context, aggregateID, aggregateType, eventType string, payload []byte, publishedBy string) error
}
