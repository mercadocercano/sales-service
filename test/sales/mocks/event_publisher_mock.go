package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockEventPublisher es un mock de port.EventPublisher
type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) Execute(ctx context.Context, aggregateID, aggregateType, eventType string, payload []byte, publishedBy string) error {
	args := m.Called(ctx, aggregateID, aggregateType, eventType, payload, publishedBy)
	return args.Error(0)
}
