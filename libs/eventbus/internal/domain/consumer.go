package domain

import "context"

// EventHandler define el contrato para procesar eventos
type EventHandler interface {
	// Handle procesa un evento de dominio
	Handle(ctx context.Context, event DomainEvent) error
	
	// ConsumerName retorna el nombre único del consumidor
	ConsumerName() string
}

// EventConsumer representa el estado de procesamiento de un evento
type EventConsumer struct {
	EventID      string
	ConsumerName string
	Status       ConsumerStatus
	RetryCount   int
	LastError    string
}

// ConsumerStatus representa los estados posibles de procesamiento
type ConsumerStatus string

const (
	StatusPending   ConsumerStatus = "pending"
	StatusProcessed ConsumerStatus = "processed"
	StatusFailed    ConsumerStatus = "failed"
	StatusRetrying  ConsumerStatus = "retrying"
)

// NewEventConsumer crea un nuevo registro de consumidor
func NewEventConsumer(eventID string, consumerName string) *EventConsumer {
	return &EventConsumer{
		EventID:      eventID,
		ConsumerName: consumerName,
		Status:       StatusPending,
		RetryCount:   0,
		LastError:    "",
	}
}
