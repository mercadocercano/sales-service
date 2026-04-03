package mocks

import (
	"context"
	"time"

	"sales/src/sales/domain/port"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockDailyReportRepository es un mock de port.DailyReportRepository
type MockDailyReportRepository struct {
	mock.Mock
}

func (m *MockDailyReportRepository) GetDailyReport(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*port.DailyReportData, error) {
	args := m.Called(ctx, tenantID, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*port.DailyReportData), args.Error(1)
}
