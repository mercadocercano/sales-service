package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockSequencePort es un mock de port.SequencePort
type MockSequencePort struct {
	mock.Mock
}

func (m *MockSequencePort) NextNumber(ctx context.Context, tenantID, documentType string) (int, error) {
	args := m.Called(ctx, tenantID, documentType)
	return args.Int(0), args.Error(1)
}
