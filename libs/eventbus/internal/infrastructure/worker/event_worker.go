package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mercadocercano/eventbus/internal/application"
	"github.com/mercadocercano/eventbus/internal/domain"
	"github.com/mercadocercano/eventbus/internal/shared"
)

type EventWorker struct {
	processUseCase *application.ProcessEventUseCase
	handlers       map[string]domain.EventHandler
	logger         *shared.Logger
	batchSize      int
	pollInterval   time.Duration
	stopChan       chan struct{}
	wg             sync.WaitGroup
	mu             sync.RWMutex
}

func NewEventWorker(
	processUseCase *application.ProcessEventUseCase,
	logger *shared.Logger,
	batchSize int,
	pollInterval time.Duration,
) *EventWorker {
	return &EventWorker{
		processUseCase: processUseCase,
		handlers:       make(map[string]domain.EventHandler),
		logger:         logger,
		batchSize:      batchSize,
		pollInterval:   pollInterval,
		stopChan:       make(chan struct{}),
	}
}

func (w *EventWorker) RegisterHandler(handler domain.EventHandler) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	consumerName := handler.ConsumerName()
	if _, exists := w.handlers[consumerName]; exists {
		return fmt.Errorf("handler already registered: %s", consumerName)
	}

	w.handlers[consumerName] = handler
	w.logger.Info("Handler registered", map[string]interface{}{
		"consumer": consumerName,
	})

	return nil
}

func (w *EventWorker) Start(ctx context.Context) error {
	w.mu.RLock()
	handlerCount := len(w.handlers)
	w.mu.RUnlock()

	if handlerCount == 0 {
		return fmt.Errorf("no handlers registered")
	}

	w.logger.Info("Starting event worker", map[string]interface{}{
		"handlers":      handlerCount,
		"batch_size":    w.batchSize,
		"poll_interval": w.pollInterval.String(),
	})

	w.mu.RLock()
	for _, handler := range w.handlers {
		w.wg.Add(1)
		go w.processLoop(ctx, handler)
	}
	w.mu.RUnlock()

	return nil
}

func (w *EventWorker) Stop() {
	w.logger.Info("Stopping event worker", nil)
	close(w.stopChan)
	w.wg.Wait()
	w.logger.Info("Event worker stopped", nil)
}

func (w *EventWorker) processLoop(ctx context.Context, handler domain.EventHandler) {
	defer w.wg.Done()

	consumerName := handler.ConsumerName()
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	w.logger.Info("Started processing loop", map[string]interface{}{
		"consumer": consumerName,
	})

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Context cancelled, stopping loop", map[string]interface{}{
				"consumer": consumerName,
			})
			return
		case <-w.stopChan:
			w.logger.Info("Stop signal received, stopping loop", map[string]interface{}{
				"consumer": consumerName,
			})
			return
		case <-ticker.C:
			if err := w.processUseCase.Execute(ctx, handler, w.batchSize); err != nil {
				w.logger.Error("Error processing events", map[string]interface{}{
					"consumer": consumerName,
					"error":    err.Error(),
				})
			}
		}
	}
}

func (w *EventWorker) GetStats() map[string]interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	consumers := make([]string, 0, len(w.handlers))
	for name := range w.handlers {
		consumers = append(consumers, name)
	}

	return map[string]interface{}{
		"handlers":      len(w.handlers),
		"batch_size":    w.batchSize,
		"poll_interval": w.pollInterval.String(),
		"consumers":     consumers,
	}
}
