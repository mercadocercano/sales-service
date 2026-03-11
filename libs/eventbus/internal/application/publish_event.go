package application

import (
	"context"

	"github.com/mercadocercano/eventbus/internal/domain"
	"github.com/mercadocercano/eventbus/internal/shared"
)

// PublishEventUseCase caso de uso para publicar eventos
type PublishEventUseCase struct {
	eventStore domain.EventStore
	logger     *shared.Logger
}

// NewPublishEventUseCase crea una nueva instancia del caso de uso
func NewPublishEventUseCase(eventStore domain.EventStore, logger *shared.Logger) *PublishEventUseCase {
	return &PublishEventUseCase{
		eventStore: eventStore,
		logger:     logger,
	}
}

// Execute publica un evento en el event bus
func (uc *PublishEventUseCase) Execute(
	ctx context.Context,
	aggregateID string,
	aggregateType string,
	eventType string,
	payload []byte,
	publishedBy string,
) error {
	event := domain.NewEvent(aggregateID, aggregateType, eventType, payload, publishedBy)

	if err := uc.eventStore.Save(ctx, event); err != nil {
		uc.logger.Error("Failed to save event", map[string]interface{}{
			"event_id":       event.ID(),
			"aggregate_type": aggregateType,
			"event_type":     eventType,
			"error":          err.Error(),
		})
		return err
	}

	uc.logger.Info("Event published successfully", map[string]interface{}{
		"event_id":       event.ID(),
		"aggregate_id":   aggregateID,
		"aggregate_type": aggregateType,
		"event_type":     eventType,
	})

	return nil
}
