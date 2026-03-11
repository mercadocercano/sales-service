package application

import (
	"context"
	"fmt"
	"time"

	"github.com/mercadocercano/eventbus/internal/domain"
	"github.com/mercadocercano/eventbus/internal/observability/metrics"
	"github.com/mercadocercano/eventbus/internal/shared"
)

const (
	MaxRetries     = 3
	RetryDelayBase = 5 * time.Second
)

// ProcessEventUseCase caso de uso para procesar eventos
type ProcessEventUseCase struct {
	eventStore domain.EventStore
	logger     *shared.Logger
}

// NewProcessEventUseCase crea una nueva instancia del caso de uso
func NewProcessEventUseCase(eventStore domain.EventStore, logger *shared.Logger) *ProcessEventUseCase {
	return &ProcessEventUseCase{
		eventStore: eventStore,
		logger:     logger,
	}
}

// Execute procesa eventos pendientes para un consumidor específico
func (uc *ProcessEventUseCase) Execute(
	ctx context.Context,
	handler domain.EventHandler,
	batchSize int,
) error {
	consumerName := handler.ConsumerName()

	events, err := uc.eventStore.GetUnprocessed(ctx, consumerName, batchSize)
	if err != nil {
		uc.logger.Error("Failed to get unprocessed events", map[string]interface{}{
			"consumer": consumerName,
			"error":    err.Error(),
		})
		return err
	}

	if len(events) == 0 {
		uc.logger.Debug("No events to process", map[string]interface{}{
			"consumer": consumerName,
		})
		return nil
	}

	uc.logger.Info("Processing events", map[string]interface{}{
		"consumer": consumerName,
		"count":    len(events),
	})

	for _, event := range events {
		if err := uc.processEvent(ctx, handler, event); err != nil {
			uc.logger.Error("Failed to process event", map[string]interface{}{
				"event_id":   event.ID(),
				"consumer":   consumerName,
				"event_type": event.EventType(),
				"error":      err.Error(),
			})
		}
	}

	return nil
}

func (uc *ProcessEventUseCase) processEvent(
	ctx context.Context,
	handler domain.EventHandler,
	event domain.DomainEvent,
) error {
	consumerName := handler.ConsumerName()
	eventType := event.EventType()

	// Medir latencia de procesamiento
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		metrics.ProcessingDurationSeconds.WithLabelValues(consumerName, eventType).Observe(duration)
	}()

	if err := handler.Handle(ctx, event); err != nil {
		retryCount, incrementErr := uc.eventStore.IncrementRetryAndGet(ctx, event.ID(), consumerName)
		if incrementErr != nil {
			uc.logger.Error("Failed to increment retry", map[string]interface{}{
				"event_id": event.ID(),
				"consumer": consumerName,
				"error":    incrementErr.Error(),
			})
		}

		// Incrementar contador de retries
		metrics.EventsRetryTotal.WithLabelValues(consumerName, eventType).Inc()

		errorMsg := fmt.Sprintf("Handler error: %v", err)

		if retryCount >= MaxRetries {
			uc.logger.Error("Max retries exceeded, marking as failed", map[string]interface{}{
				"event_id":    event.ID(),
				"consumer":    consumerName,
				"retry_count": retryCount,
				"error":       errorMsg,
			})
			
			// Incrementar contador de fallos permanentes
			metrics.EventsFailedTotal.WithLabelValues(consumerName, eventType).Inc()
			metrics.RetryCountHistogram.Observe(float64(retryCount))
			
			if markErr := uc.eventStore.MarkFailed(ctx, event.ID(), consumerName, errorMsg); markErr != nil {
				uc.logger.Error("Failed to mark event as failed", map[string]interface{}{
					"event_id": event.ID(),
					"consumer": consumerName,
					"error":    markErr.Error(),
				})
			}
		} else {
			uc.logger.Warn("Event handler failed, will retry", map[string]interface{}{
				"event_id":    event.ID(),
				"consumer":    consumerName,
				"retry_count": retryCount,
				"max_retries": MaxRetries,
			})
		}

		return err
	}

	// Éxito: incrementar contador de procesados
	metrics.EventsProcessedTotal.WithLabelValues(consumerName, eventType).Inc()

	if err := uc.eventStore.MarkProcessed(ctx, event.ID(), consumerName); err != nil {
		uc.logger.Error("Failed to mark event as processed", map[string]interface{}{
			"event_id": event.ID(),
			"consumer": consumerName,
			"error":    err.Error(),
		})
		return err
	}

	uc.logger.Info("Event processed successfully", map[string]interface{}{
		"event_id":   event.ID(),
		"consumer":   consumerName,
		"event_type": event.EventType(),
	})

	return nil
}
