package usecase_test

import (
	"context"
	"testing"
	"time"

	"sales/src/sales/application/usecase"
	"sales/src/sales/domain/port"
	mocks "sales/test/sales/mocks"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDailyReportUseCase_Execute_WithSalesData_ReturnsCombinedReport(t *testing.T) {
	// Arrange
	reportRepo := new(mocks.MockDailyReportRepository)
	uc := usecase.NewDailyReportUseCase(reportRepo)

	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	dateStr := "2025-06-15"
	from := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 6, 16, 0, 0, 0, 0, time.UTC)

	firstTx := time.Date(2025, 6, 15, 8, 30, 0, 0, time.UTC)
	lastTx := time.Date(2025, 6, 15, 20, 45, 0, 0, time.UTC)

	reportRepo.On("GetDailyReport", context.Background(), tenantID, from, to).
		Return(&port.DailyReportData{
			PosSalesCount:      5,
			OrdersCount:        3,
			PosGrossTotal:      decimal.NewFromFloat(1500.00),
			PosDiscounts:       decimal.NewFromFloat(100.00),
			PosNetTotal:        decimal.NewFromFloat(1400.00),
			FirstTransactionAt: &firstTx,
			LastTransactionAt:  &lastTx,
		}, nil)

	// Act
	resp, err := uc.Execute(context.Background(), tenantID, dateStr)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "2025-06-15", resp.Date)
	assert.Equal(t, 5, resp.PosSalesCount)
	assert.Equal(t, 3, resp.OrdersCount)
	assert.Equal(t, 8, resp.TotalTransactions)
	assert.True(t, resp.PosGrossTotal.Equal(decimal.NewFromFloat(1500.00)))
	assert.True(t, resp.PosDiscounts.Equal(decimal.NewFromFloat(100.00)))
	assert.True(t, resp.PosNetTotal.Equal(decimal.NewFromFloat(1400.00)))
	assert.NotNil(t, resp.FirstTransactionAt)
	assert.NotNil(t, resp.LastTransactionAt)
	reportRepo.AssertExpectations(t)
}

func TestDailyReportUseCase_Execute_WithInvalidDateFormat_ReturnsError(t *testing.T) {
	// Arrange
	reportRepo := new(mocks.MockDailyReportRepository)
	uc := usecase.NewDailyReportUseCase(reportRepo)

	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	// Act
	resp, err := uc.Execute(context.Background(), tenantID, "15/06/2025")

	// Assert
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid date format")
	reportRepo.AssertNotCalled(t, "GetDailyReport")
}

func TestDailyReportUseCase_Execute_WithNoSales_ReturnsZeroReport(t *testing.T) {
	// Arrange
	reportRepo := new(mocks.MockDailyReportRepository)
	uc := usecase.NewDailyReportUseCase(reportRepo)

	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	dateStr := "2025-06-15"
	from := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 6, 16, 0, 0, 0, 0, time.UTC)

	reportRepo.On("GetDailyReport", context.Background(), tenantID, from, to).
		Return(&port.DailyReportData{
			PosSalesCount:      0,
			OrdersCount:        0,
			PosGrossTotal:      decimal.Zero,
			PosDiscounts:       decimal.Zero,
			PosNetTotal:        decimal.Zero,
			FirstTransactionAt: nil,
			LastTransactionAt:  nil,
		}, nil)

	// Act
	resp, err := uc.Execute(context.Background(), tenantID, dateStr)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "2025-06-15", resp.Date)
	assert.Equal(t, 0, resp.PosSalesCount)
	assert.Equal(t, 0, resp.OrdersCount)
	assert.Equal(t, 0, resp.TotalTransactions)
	assert.True(t, resp.PosGrossTotal.Equal(decimal.Zero))
	assert.Nil(t, resp.FirstTransactionAt)
	assert.Nil(t, resp.LastTransactionAt)
	reportRepo.AssertExpectations(t)
}
