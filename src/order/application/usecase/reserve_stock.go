package usecase

import (
	"fmt"
	"order/src/order/application/request"
	"order/src/order/application/response"
	"order/src/order/infrastructure/client"

	"github.com/google/uuid"
)

// ReserveStockUseCase caso de uso para reservar stock
type ReserveStockUseCase struct {
	stockClient *client.StockClient
}

// NewReserveStockUseCase crea una nueva instancia del caso de uso
func NewReserveStockUseCase(stockClient *client.StockClient) *ReserveStockUseCase {
	return &ReserveStockUseCase{
		stockClient: stockClient,
	}
}

// Execute ejecuta la reserva de stock (multi-item, ALL OR NOTHING)
func (uc *ReserveStockUseCase) Execute(tenantID, authToken string, req *request.ReserveStockRequest) (*response.ReserveStockResponse, error) {
	var itemsResponse []response.ReserveStockItemResponse
	var reservedItems []response.ReserveStockItemResponse

	// Reservar cada item
	for _, item := range req.Items {
		// Generar reference UUID por item
		reference := uuid.New().String()

		stockResp, err := uc.stockClient.ReserveStock(tenantID, authToken, item.SKU, item.Quantity, reference)
		if err != nil {
			// Si falla un item, liberar los ya reservados (rollback)
			for _, reservedItem := range reservedItems {
				_, _ = uc.stockClient.ReleaseStock(tenantID, authToken, reservedItem.SKU, reservedItem.Quantity, reservedItem.Reference)
			}

			// Propagarcomo 409
			if contains(err.Error(), "insufficient stock") || contains(err.Error(), "409") {
				return nil, fmt.Errorf("insufficient_stock for SKU %s: %w", item.SKU, err)
			}
			return nil, fmt.Errorf("error reserving stock for SKU %s: %w", item.SKU, err)
		}

		itemResp := response.ReserveStockItemResponse{
			SKU:       stockResp.SKU,
			Quantity:  stockResp.ReservedQty,
			Reference: stockResp.Reference,
		}

		itemsResponse = append(itemsResponse, itemResp)
		reservedItems = append(reservedItems, itemResp)
	}

	// Todas las reservas exitosas
	return &response.ReserveStockResponse{
		Reserved: true,
		Items:    itemsResponse,
	}, nil
}

// contains verifica si un string contiene otro (helper simple)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
