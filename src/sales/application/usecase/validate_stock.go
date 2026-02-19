package usecase

import (
	"fmt"
	"sales/src/sales/application/request"
	"sales/src/sales/application/response"
	"sales/src/sales/infrastructure/client"
)

// ValidateStockUseCase caso de uso para validar stock
type ValidateStockUseCase struct {
	stockClient *client.StockClient
}

// NewValidateStockUseCase crea una nueva instancia del caso de uso
func NewValidateStockUseCase(stockClient *client.StockClient) *ValidateStockUseCase {
	return &ValidateStockUseCase{
		stockClient: stockClient,
	}
}

// Execute ejecuta la validación de stock (multi-item)
func (uc *ValidateStockUseCase) Execute(tenantID, authToken string, req *request.ValidateStockRequest) (*response.ValidateStockResponse, error) {
	var itemsResponse []response.ValidateStockItemResponse
	allValid := true

	// Validar cada item
	for _, item := range req.Items {
		stockResp, hasEnoughStock, err := uc.stockClient.ValidateStock(tenantID, authToken, item.SKU, item.Quantity)
		if err != nil {
			return nil, fmt.Errorf("error validating stock for SKU %s: %w", item.SKU, err)
		}

		itemValid := hasEnoughStock && !stockResp.IsOutOfStock
		if !itemValid {
			allValid = false
		}

		itemsResponse = append(itemsResponse, response.ValidateStockItemResponse{
			SKU:          item.SKU,
			RequestedQty: item.Quantity,
			Available:    itemValid,
			AvailableQty: int(stockResp.AvailableQuantity),
		})
	}

	// Mapear respuesta
	return &response.ValidateStockResponse{
		Valid: allValid,
		Items: itemsResponse,
	}, nil
}
