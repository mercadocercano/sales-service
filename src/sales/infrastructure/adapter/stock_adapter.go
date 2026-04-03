package adapter

import (
	"sales/src/sales/domain/port"
	"sales/src/sales/infrastructure/client"
)

// StockAdapter adapta client.StockClient a port.StockPort
type StockAdapter struct {
	client *client.StockClient
}

// NewStockAdapter crea un nuevo adaptador
func NewStockAdapter(c *client.StockClient) *StockAdapter {
	return &StockAdapter{client: c}
}

func (a *StockAdapter) ValidateStock(tenantID, sku string, quantity int) (*port.StockAvailability, bool, error) {
	resp, hasEnough, err := a.client.ValidateStock(tenantID, sku, quantity)
	if err != nil {
		return nil, false, err
	}
	return &port.StockAvailability{
		VariantSKU:        resp.VariantSKU,
		ProductSKU:        resp.ProductSKU,
		AvailableQuantity: resp.AvailableQuantity,
		ReservedQuantity:  resp.ReservedQuantity,
		TotalQuantity:     resp.TotalQuantity,
		IsOutOfStock:      resp.IsOutOfStock,
		IsLowStock:        resp.IsLowStock,
	}, hasEnough, nil
}

func (a *StockAdapter) ReserveStock(tenantID, sku string, quantity int, reference string) (*port.StockReserveResult, error) {
	resp, err := a.client.ReserveStock(tenantID, sku, quantity, reference)
	if err != nil {
		return nil, err
	}
	return &port.StockReserveResult{
		SKU:          resp.SKU,
		ReservedQty:  resp.ReservedQty,
		RemainingQty: resp.RemainingQty,
		Reference:    resp.Reference,
	}, nil
}

func (a *StockAdapter) ReleaseStock(tenantID, sku string, quantity int, reference string) (*port.StockReleaseResult, error) {
	resp, err := a.client.ReleaseStock(tenantID, sku, quantity, reference)
	if err != nil {
		return nil, err
	}
	return &port.StockReleaseResult{
		SKU:          resp.SKU,
		ReleasedQty:  resp.ReleasedQty,
		AvailableQty: resp.AvailableQty,
		ReservedQty:  resp.ReservedQty,
		Reference:    resp.Reference,
	}, nil
}

func (a *StockAdapter) ConsumeStock(tenantID, sku string, quantity int, reference string) (*port.StockConsumeResult, error) {
	resp, err := a.client.ConsumeStock(tenantID, sku, quantity, reference)
	if err != nil {
		return nil, err
	}
	return &port.StockConsumeResult{
		SKU:         resp.SKU,
		ConsumedQty: resp.ConsumedQty,
		ReservedQty: resp.ReservedQty,
		Reference:   resp.Reference,
	}, nil
}

func (a *StockAdapter) RevertConsume(tenantID, sku string, quantity int, reference string) (*port.StockRevertResult, error) {
	resp, err := a.client.RevertConsume(tenantID, sku, quantity, reference)
	if err != nil {
		return nil, err
	}
	return &port.StockRevertResult{
		SKU:          resp.SKU,
		RevertedQty:  resp.RevertedQty,
		AvailableQty: resp.AvailableQty,
		Reference:    resp.Reference,
	}, nil
}

func (a *StockAdapter) ProcessSaleAtomic(tenantID, sku string, quantity float64, reference string) (*port.ProcessSaleAtomicResult, error) {
	resp, err := a.client.ProcessSaleAtomic(tenantID, sku, quantity, reference)
	if err != nil {
		return nil, err
	}
	return &port.ProcessSaleAtomicResult{
		Success:        resp.Success,
		Message:        resp.Message,
		VariantSKU:     resp.VariantSKU,
		QuantitySold:   resp.QuantitySold,
		RemainingStock: resp.RemainingStock,
		TotalQuantity:  resp.TotalQuantity,
		StockEntryID:   resp.StockEntryID,
	}, nil
}

func (a *StockAdapter) CompensateSale(tenantID, stockEntryID, reason string) error {
	return a.client.CompensateSale(tenantID, stockEntryID, reason)
}
