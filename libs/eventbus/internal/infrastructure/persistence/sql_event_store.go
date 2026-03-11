package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mercadocercano/eventbus/internal/domain"
	"github.com/mercadocercano/eventbus/internal/shared"
)

type SQLEventStore struct {
	db     *sql.DB
	logger *shared.Logger
}

func NewSQLEventStore(db *sql.DB, logger *shared.Logger) *SQLEventStore {
	return &SQLEventStore{
		db:     db,
		logger: logger,
	}
}

func (s *SQLEventStore) Save(ctx context.Context, event domain.DomainEvent) error {
	query := `
		INSERT INTO event_bus (
			id, aggregate_type, aggregate_id, event_type, 
			payload_json, occurred_at, published_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	var payloadJSON json.RawMessage
	if err := json.Unmarshal(event.Payload(), &payloadJSON); err != nil {
		return fmt.Errorf("invalid JSON payload: %w", err)
	}

	_, err := s.db.ExecContext(
		ctx,
		query,
		event.ID(),
		event.AggregateType(),
		event.AggregateID(),
		event.EventType(),
		payloadJSON,
		event.OccurredAt(),
		event.PublishedBy(),
	)

	if err != nil {
		return fmt.Errorf("failed to save event: %w", err)
	}

	return nil
}

func (s *SQLEventStore) GetUnprocessed(ctx context.Context, consumerName string, limit int) ([]domain.DomainEvent, error) {
	query := `
		SELECT 
			eb.id, eb.aggregate_type, eb.aggregate_id, eb.event_type,
			eb.payload_json, eb.occurred_at, eb.published_by
		FROM event_bus eb
		WHERE NOT EXISTS (
			SELECT 1 FROM event_consumers ec
			WHERE ec.event_id = eb.id
			AND ec.consumer_name = $1
			AND ec.status IN ('processed', 'failed')
		)
		AND (
			SELECT COALESCE(MAX(retry_count), 0)
			FROM event_consumers
			WHERE event_id = eb.id AND consumer_name = $1
		) < 3
		ORDER BY eb.occurred_at ASC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, consumerName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query unprocessed events: %w", err)
	}
	defer rows.Close()

	var events []domain.DomainEvent
	for rows.Next() {
		var (
			id            string
			aggregateType string
			aggregateID   string
			eventType     string
			payloadJSON   json.RawMessage
			occurredAt    time.Time
			publishedBy   string
		)

		if err := rows.Scan(
			&id, &aggregateType, &aggregateID, &eventType,
			&payloadJSON, &occurredAt, &publishedBy,
		); err != nil {
			return nil, fmt.Errorf("failed to scan event row: %w", err)
		}

		payload, err := json.Marshal(payloadJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}

		event := domain.ReconstructEvent(
			id, aggregateID, aggregateType, eventType,
			payload, occurredAt, publishedBy,
		)

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating event rows: %w", err)
	}

	return events, nil
}

func (s *SQLEventStore) MarkProcessed(ctx context.Context, eventID string, consumerName string) error {
	query := `
		INSERT INTO event_consumers (event_id, consumer_name, processed_at, status, retry_count, last_error)
		VALUES ($1, $2, $3, $4, 0, '')
		ON CONFLICT (event_id, consumer_name) 
		DO UPDATE SET 
			processed_at = EXCLUDED.processed_at,
			status = EXCLUDED.status,
			retry_count = 0,
			last_error = ''
	`

	_, err := s.db.ExecContext(ctx, query, eventID, consumerName, time.Now().UTC(), "processed")
	if err != nil {
		return fmt.Errorf("failed to mark event as processed: %w", err)
	}

	return nil
}

func (s *SQLEventStore) MarkFailed(ctx context.Context, eventID string, consumerName string, errorMsg string) error {
	query := `
		INSERT INTO event_consumers (event_id, consumer_name, processed_at, status, retry_count, last_error)
		VALUES ($1, $2, $3, $4, 1, $5)
		ON CONFLICT (event_id, consumer_name)
		DO UPDATE SET
			processed_at = EXCLUDED.processed_at,
			status = EXCLUDED.status,
			last_error = EXCLUDED.last_error,
			retry_count = event_consumers.retry_count
	`

	_, err := s.db.ExecContext(ctx, query, eventID, consumerName, time.Now().UTC(), "failed", errorMsg)
	if err != nil {
		return fmt.Errorf("failed to mark event as failed: %w", err)
	}

	return nil
}

func (s *SQLEventStore) IncrementRetry(ctx context.Context, eventID string, consumerName string) error {
	_, err := s.IncrementRetryAndGet(ctx, eventID, consumerName)
	return err
}

func (s *SQLEventStore) IncrementRetryAndGet(ctx context.Context, eventID string, consumerName string) (int, error) {
	query := `
		INSERT INTO event_consumers (event_id, consumer_name, processed_at, status, retry_count, last_error)
		VALUES ($1, $2, $3, $4, 1, '')
		ON CONFLICT (event_id, consumer_name)
		DO UPDATE SET
			retry_count = event_consumers.retry_count + 1,
			status = 'retrying',
			processed_at = EXCLUDED.processed_at
		RETURNING retry_count
	`

	var retryCount int
	err := s.db.QueryRowContext(ctx, query, eventID, consumerName, time.Now().UTC(), "retrying").Scan(&retryCount)
	if err != nil {
		return 0, fmt.Errorf("failed to increment retry count: %w", err)
	}

	return retryCount, nil
}
