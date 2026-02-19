package usecase

import (
	"context"
	"sales/src/sales/application/response"
	"sales/src/sales/domain/entity"
	"sales/src/sales/domain/port"

	"github.com/google/uuid"
)

// ListPosSalesUseCase caso de uso para listar ventas POS
// Hito: POS-SALE-02 - Reporte
type ListPosSalesUseCase struct {
	posSaleRepo port.PosSaleRepository
}

// NewListPosSalesUseCase crea una nueva instancia
func NewListPosSalesUseCase(posSaleRepo port.PosSaleRepository) *ListPosSalesUseCase {
	return &ListPosSalesUseCase{posSaleRepo: posSaleRepo}
}

// Execute lista las ventas POS del tenant
func (uc *ListPosSalesUseCase) Execute(ctx context.Context, tenantID uuid.UUID) ([]*response.PosSaleListItem, error) {
	sales, err := uc.posSaleRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return toListItems(sales), nil
}

func toListItems(sales []*entity.PosSale) []*response.PosSaleListItem {
	items := make([]*response.PosSaleListItem, 0, len(sales))
	for _, s := range sales {
		items = append(items, &response.PosSaleListItem{
			ID:              s.ID,
			CustomerID:      s.CustomerID,
			PaymentMethodID: s.PaymentMethodID,
			TotalAmount:     s.TotalAmount,
			DiscountAmount:  s.DiscountAmount,
			FinalAmount:     s.FinalAmount,
			Currency:        s.Currency,
			TotalItems:      s.TotalItems(),
			CreatedAt:       s.CreatedAt,
		})
	}
	return items
}
