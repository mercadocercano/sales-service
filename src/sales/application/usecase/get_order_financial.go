package usecase

import (
	"context"
	"sales/src/sales/application/response"
	"sales/src/sales/domain/port"
	"time"
)

// GetOrderFinancialUseCase obtiene el resumen financiero de una orden
type GetOrderFinancialUseCase struct {
	orderRepo port.OrderRepository
}

// NewGetOrderFinancialUseCase crea una nueva instancia
func NewGetOrderFinancialUseCase(orderRepo port.OrderRepository) *GetOrderFinancialUseCase {
	return &GetOrderFinancialUseCase{orderRepo: orderRepo}
}

// Execute devuelve total_amount, total_paid, remaining, status, payment_count, last_payment_at
func (uc *GetOrderFinancialUseCase) Execute(ctx context.Context, tenantID, orderID string) (*response.OrderFinancialResponse, error) {
	s, err := uc.orderRepo.GetOrderFinancialSummary(ctx, orderID, tenantID)
	if err != nil {
		return nil, err
	}
	resp := &response.OrderFinancialResponse{
		OrderID:      s.OrderID,
		TenantID:     s.TenantID,
		TotalAmount:  s.TotalAmount,
		TotalPaid:    s.TotalPaid,
		Remaining:    s.Remaining,
		Status:       s.Status,
		PaymentCount: s.PaymentCount,
	}
	if s.LastPaymentAt != nil {
		t := s.LastPaymentAt.Format(time.RFC3339)
		resp.LastPaymentAt = &t
	}
	return resp, nil
}
