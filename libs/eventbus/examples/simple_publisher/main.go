package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	_ "github.com/lib/pq"

	"github.com/mercadocercano/eventbus/internal/application"
	"github.com/mercadocercano/eventbus/internal/infrastructure/config"
	"github.com/mercadocercano/eventbus/internal/infrastructure/persistence"
	"github.com/mercadocercano/eventbus/internal/shared"
)

type SaleCreatedPayload struct {
	SaleID      string  `json:"sale_id"`
	CustomerID  string  `json:"customer_id"`
	TotalAmount float64 `json:"total_amount"`
	Items       int     `json:"items"`
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
	publishUseCase := application.NewPublishEventUseCase(eventStore, logger)

	payload := SaleCreatedPayload{
		SaleID:      "sale-123",
		CustomerID:  "customer-456",
		TotalAmount: 1500.50,
		Items:       3,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Failed to marshal payload: %v", err)
	}

	ctx := context.Background()
	if err := publishUseCase.Execute(
		ctx,
		"sale-123",
		"sale",
		"sale.created",
		payloadBytes,
		"sales-service",
	); err != nil {
		log.Fatalf("Failed to publish event: %v", err)
	}

	fmt.Println("Event published successfully!")
}
