package usecase

import (
	"fmt"
	"sales/src/sales/application/request"
	"sales/src/sales/application/response"
	"sales/src/sales/domain/port"
)

// ReleaseStockUseCase caso de uso para liberar stock reservado
type ReleaseStockUseCase struct {
	stockPort port.StockPort
}

// NewReleaseStockUseCase crea una nueva instancia del caso de uso
func NewReleaseStockUseCase(stockPort port.StockPort) *ReleaseStockUseCase {
	return &ReleaseStockUseCase{
		stockPort: stockPort,
	}
}

// Execute ejecuta la liberación de stock
func (uc *ReleaseStockUseCase) Execute(tenantID string, req *request.ReleaseStockRequest) (*response.ReleaseStockResponse, error) {
	// Llamar a stock-service vía Kong (S2S)
	stockResp, err := uc.stockPort.ReleaseStock(tenantID, req.SKU, req.Quantity, req.Reference)
	if err != nil {
		// Si es error de stock reservado insuficiente, propagarlo como 409
		if contains(err.Error(), "insufficient reserved stock") || contains(err.Error(), "409") {
			return nil, fmt.Errorf("insufficient_reserved_stock: %w", err)
		}
		return nil, fmt.Errorf("error releasing stock: %w", err)
	}

	// Mapear respuesta
	return &response.ReleaseStockResponse{
		Released:  true,
		SKU:       stockResp.SKU,
		Quantity:  stockResp.ReleasedQty,
		Reference: stockResp.Reference,
	}, nil
}
