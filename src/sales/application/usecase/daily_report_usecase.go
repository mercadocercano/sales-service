package usecase

import (
	"context"
	"fmt"
	"time"

	"sales/src/sales/application/response"
	"sales/src/sales/domain/port"

	"github.com/google/uuid"
)

// DailyReportUseCase caso de uso para reporte diario de ventas
type DailyReportUseCase struct {
	reportRepo port.DailyReportRepository
}

// NewDailyReportUseCase crea una nueva instancia del caso de uso
func NewDailyReportUseCase(reportRepo port.DailyReportRepository) *DailyReportUseCase {
	return &DailyReportUseCase{
		reportRepo: reportRepo,
	}
}

// Execute genera el reporte diario para una fecha específica
func (uc *DailyReportUseCase) Execute(ctx context.Context, tenantID uuid.UUID, date string) (*response.DailyReportResponse, error) {
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}

	from := parsedDate
	to := parsedDate.AddDate(0, 0, 1)

	data, err := uc.reportRepo.GetDailyReport(ctx, tenantID, from, to)
	if err != nil {
		return nil, err
	}

	return &response.DailyReportResponse{
		Date:              date,
		PosSalesCount:     data.PosSalesCount,
		OrdersCount:       data.OrdersCount,
		TotalTransactions: data.PosSalesCount + data.OrdersCount,
		PosGrossTotal:     data.PosGrossTotal,
		PosDiscounts:      data.PosDiscounts,
		PosNetTotal:       data.PosNetTotal,
		FirstTransactionAt: data.FirstTransactionAt,
		LastTransactionAt:  data.LastTransactionAt,
	}, nil
}
