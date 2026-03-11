package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"

	"github.com/mercadocercano/eventbus/internal/application"
	"github.com/mercadocercano/eventbus/internal/infrastructure/config"
	"github.com/mercadocercano/eventbus/internal/infrastructure/persistence"
	"github.com/mercadocercano/eventbus/internal/infrastructure/worker"
	"github.com/mercadocercano/eventbus/internal/shared"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logger := shared.NewLogger(shared.LogLevel(cfg.LogLevel))
	logger.Info("Starting eventbus worker", map[string]interface{}{
		"service": cfg.ServiceName,
	})

	db, err := sql.Open("postgres", cfg.ConnectionString())
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connection established", nil)

	eventStore := persistence.NewSQLEventStore(db, logger)
	processUseCase := application.NewProcessEventUseCase(eventStore, logger)

	eventWorker := worker.NewEventWorker(
		processUseCase,
		logger,
		cfg.Worker.BatchSize,
		cfg.Worker.PollInterval,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := eventWorker.Start(ctx); err != nil {
		return fmt.Errorf("failed to start worker: %w", err)
	}

	logger.Info("Worker started successfully", eventWorker.GetStats())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logger.Info("Shutdown signal received", nil)

	cancel()
	eventWorker.Stop()

	logger.Info("Worker shutdown complete", nil)
	return nil
}
