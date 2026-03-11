package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"

	"github.com/mercadocercano/eventbus/internal/application"
	"github.com/mercadocercano/eventbus/internal/domain"
	"github.com/mercadocercano/eventbus/internal/infrastructure/config"
	"github.com/mercadocercano/eventbus/internal/infrastructure/persistence"
	"github.com/mercadocercano/eventbus/internal/infrastructure/worker"
	"github.com/mercadocercano/eventbus/internal/shared"
)

type SaleCreatedPayload struct {
	SaleID      string  `json:"sale_id"`
	CustomerID  string  `json:"customer_id"`
	TotalAmount float64 `json:"total_amount"`
	Items       int     `json:"items"`
}

type LedgerEventHandler struct {
	logger *shared.Logger
}

func NewLedgerEventHandler(logger *shared.Logger) *LedgerEventHandler {
	return &LedgerEventHandler{logger: logger}
}

func (h *LedgerEventHandler) ConsumerName() string {
	return "ledger-service"
}

func (h *LedgerEventHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
	h.logger.Info("Processing event", map[string]interface{}{
		"event_id":   event.ID(),
		"event_type": event.EventType(),
	})

	if event.EventType() == "sale.created" {
		var payload SaleCreatedPayload
		if err := json.Unmarshal(event.Payload(), &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload: %w", err)
		}

		h.logger.Info("Sale created event received", map[string]interface{}{
			"sale_id":      payload.SaleID,
			"customer_id":  payload.CustomerID,
			"total_amount": payload.TotalAmount,
		})

		fmt.Printf("✅ Ledger: Registering sale %s for $%.2f\n", payload.SaleID, payload.TotalAmount)
	}

	return nil
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger := shared.NewLogger(shared.LogLevel(cfg.LogLevel))

	db, err := sql.Open("postgres", cfg.ConnectionString())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	eventStore := persistence.NewSQLEventStore(db, logger)
	processUseCase := application.NewProcessEventUseCase(eventStore, logger)

	eventWorker := worker.NewEventWorker(
		processUseCase,
		logger,
		cfg.Worker.BatchSize,
		cfg.Worker.PollInterval,
	)

	ledgerHandler := NewLedgerEventHandler(logger)
	if err := eventWorker.RegisterHandler(ledgerHandler); err != nil {
		log.Fatalf("Failed to register handler: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := eventWorker.Start(ctx); err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}

	fmt.Println("🚀 Ledger consumer started. Press Ctrl+C to stop.")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n⏹️  Shutting down...")
	cancel()
	eventWorker.Stop()
}
