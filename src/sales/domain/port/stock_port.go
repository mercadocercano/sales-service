package port

// StockAvailability representa la disponibilidad de stock de un SKU
type StockAvailability struct {
	VariantSKU        string  `json:"variant_sku"`
	ProductSKU        string  `json:"product_sku"`
	AvailableQuantity float64 `json:"available_quantity"`
	ReservedQuantity  float64 `json:"reserved_quantity"`
	TotalQuantity     float64 `json:"total_quantity"`
	IsOutOfStock      bool    `json:"is_out_of_stock"`
	IsLowStock        bool    `json:"is_low_stock"`
}

// StockReserveResult representa el resultado de una reserva de stock
type StockReserveResult struct {
	SKU          string `json:"sku"`
	ReservedQty  int    `json:"reserved_qty"`
	RemainingQty int    `json:"remaining_qty"`
	Reference    string `json:"reference"`
}

// StockReleaseResult representa el resultado de una liberación de stock
type StockReleaseResult struct {
	SKU          string `json:"sku"`
	ReleasedQty  int    `json:"released_qty"`
	AvailableQty int    `json:"available_qty"`
	ReservedQty  int    `json:"reserved_qty"`
	Reference    string `json:"reference"`
}

// StockConsumeResult representa el resultado de un consumo de stock
type StockConsumeResult struct {
	SKU         string `json:"sku"`
	ConsumedQty int    `json:"consumed_qty"`
	ReservedQty int    `json:"reserved_qty"`
	Reference   string `json:"reference"`
}

// StockRevertResult representa el resultado de una reversión de consumo
type StockRevertResult struct {
	SKU          string `json:"sku"`
	RevertedQty  int    `json:"reverted_qty"`
	AvailableQty int    `json:"available_qty"`
	Reference    string `json:"reference"`
}

// ProcessSaleAtomicResult representa el resultado de una venta atómica
type ProcessSaleAtomicResult struct {
	Success        bool    `json:"success"`
	Message        string  `json:"message"`
	VariantSKU     string  `json:"variant_sku"`
	QuantitySold   float64 `json:"quantity_sold"`
	RemainingStock float64 `json:"remaining_stock"`
	TotalQuantity  float64 `json:"total_quantity"`
	StockEntryID   string  `json:"stock_entry_id"`
}

// StockPort define las operaciones de stock disponibles para los use cases
type StockPort interface {
	ValidateStock(tenantID, sku string, quantity int) (*StockAvailability, bool, error)
	ReserveStock(tenantID, sku string, quantity int, reference string) (*StockReserveResult, error)
	ReleaseStock(tenantID, sku string, quantity int, reference string) (*StockReleaseResult, error)
	ConsumeStock(tenantID, sku string, quantity int, reference string) (*StockConsumeResult, error)
	RevertConsume(tenantID, sku string, quantity int, reference string) (*StockRevertResult, error)
	ProcessSaleAtomic(tenantID, sku string, quantity float64, reference string) (*ProcessSaleAtomicResult, error)
	CompensateSale(tenantID, stockEntryID, reason string) error
}
