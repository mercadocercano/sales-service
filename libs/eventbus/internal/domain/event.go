package domain

import (
	"time"

	"github.com/google/uuid"
)

// DomainEvent representa un evento de dominio genérico
type DomainEvent interface {
	ID() string
	AggregateID() string
	AggregateType() string
	EventType() string
	Payload() []byte
	OccurredAt() time.Time
	PublishedBy() string
}

// Event es la implementación concreta de DomainEvent
type Event struct {
	id            string
	aggregateID   string
	aggregateType string
	eventType     string
	payload       []byte
	occurredAt    time.Time
	publishedBy   string
}

// NewEvent crea un nuevo evento de dominio
func NewEvent(
	aggregateID string,
	aggregateType string,
	eventType string,
	payload []byte,
	publishedBy string,
) *Event {
	return &Event{
		id:            uuid.New().String(),
		aggregateID:   aggregateID,
		aggregateType: aggregateType,
		eventType:     eventType,
		payload:       payload,
		occurredAt:    time.Now().UTC(),
		publishedBy:   publishedBy,
	}
}

// ReconstructEvent reconstruye un evento desde persistencia
func ReconstructEvent(
	id string,
	aggregateID string,
	aggregateType string,
	eventType string,
	payload []byte,
	occurredAt time.Time,
	publishedBy string,
) *Event {
	return &Event{
		id:            id,
		aggregateID:   aggregateID,
		aggregateType: aggregateType,
		eventType:     eventType,
		payload:       payload,
		occurredAt:    occurredAt,
		publishedBy:   publishedBy,
	}
}

// Getters
func (e *Event) ID() string            { return e.id }
func (e *Event) AggregateID() string   { return e.aggregateID }
func (e *Event) AggregateType() string { return e.aggregateType }
func (e *Event) EventType() string     { return e.eventType }
func (e *Event) Payload() []byte       { return e.payload }
func (e *Event) OccurredAt() time.Time { return e.occurredAt }
func (e *Event) PublishedBy() string   { return e.publishedBy }
