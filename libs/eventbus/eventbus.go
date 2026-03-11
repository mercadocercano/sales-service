package eventbus

import (
	"github.com/mercadocercano/eventbus/internal/application"
	"github.com/mercadocercano/eventbus/internal/domain"
	"github.com/mercadocercano/eventbus/internal/infrastructure/persistence"
	"github.com/mercadocercano/eventbus/internal/infrastructure/worker"
	"github.com/mercadocercano/eventbus/internal/shared"
)

// Re-export public interfaces and types
type (
	DomainEvent         = domain.DomainEvent
	EventHandler        = domain.EventHandler
	Logger              = shared.Logger
	ProcessEventUseCase = application.ProcessEventUseCase
	PublishEventUseCase = application.PublishEventUseCase
	SQLEventStore       = persistence.SQLEventStore
	EventWorker         = worker.EventWorker
)

// Re-export constructors
var (
	NewLogger              = shared.NewLogger
	NewSQLEventStore       = persistence.NewSQLEventStore
	NewProcessEventUseCase = application.NewProcessEventUseCase
	NewPublishEventUseCase = application.NewPublishEventUseCase
	NewEventWorker         = worker.NewEventWorker
)

// Re-export log levels
const (
	LevelDebug = shared.LevelDebug
	LevelInfo  = shared.LevelInfo
	LevelWarn  = shared.LevelWarn
	LevelError = shared.LevelError
)

type LogLevel = shared.LogLevel
