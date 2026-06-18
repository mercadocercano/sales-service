package usecase

import (
	"context"
	"fmt"

	"sales/src/sales/application/response"
	"sales/src/sales/domain/port"

	"github.com/google/uuid"
)

// GetPosSaleUseCase recupera el detalle completo de una venta POS (comprobante).
// E18 Tramo B.
type GetPosSaleUseCase struct {
	posSaleRepo       port.PosSaleRepository
	paymentMethodPort port.PaymentMethodPort
	customerPort      port.CustomerPort
}

// Fallbacks de nombre de cliente para el comprobante.
const (
	consumidorFinalName  = "Consumidor Final"
	customerNameFallback = "Cliente" // cliente nombrado pero irresoluble (port nil/error/no encontrado)
)

// NewGetPosSaleUseCase crea una nueva instancia del caso de uso.
// paymentMethodPort puede ser nil (el detalle igual se devuelve, con nombre "Unknown").
// customerPort puede ser nil (el detalle igual se devuelve; nombre del cliente con fallback).
func NewGetPosSaleUseCase(
	posSaleRepo port.PosSaleRepository,
	paymentMethodPort port.PaymentMethodPort,
	customerPort port.CustomerPort,
) *GetPosSaleUseCase {
	return &GetPosSaleUseCase{
		posSaleRepo:       posSaleRepo,
		paymentMethodPort: paymentMethodPort,
		customerPort:      customerPort,
	}
}

// Execute retorna el detalle de la venta POS del tenant.
// Multi-tenancy: una venta de otro tenant (o inexistente) retorna entity.ErrPosSaleNotFound.
func (uc *GetPosSaleUseCase) Execute(ctx context.Context, tenantID, saleID string) (*response.POSSaleDetailResponse, error) {
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant_id format: %w", err)
	}
	saleUUID, err := uuid.Parse(saleID)
	if err != nil {
		return nil, fmt.Errorf("invalid sale id format: %w", err)
	}

	sale, err := uc.posSaleRepo.GetByID(ctx, tenantUUID, saleUUID)
	if err != nil {
		return nil, err // incluye entity.ErrPosSaleNotFound
	}

	items := make([]response.POSSaleDetailItemResponse, 0, len(sale.Items))
	for _, it := range sale.Items {
		items = append(items, response.POSSaleDetailItemResponse{
			SKU:         it.SKU,
			ProductName: it.ProductName,
			Quantity:    it.Quantity,
			UnitPrice:   it.UnitPrice,
			Subtotal:    it.Subtotal,
		})
	}

	paymentMethodName := "Unknown"
	if uc.paymentMethodPort != nil {
		paymentMethodName = uc.paymentMethodPort.GetName(sale.PaymentMethodID)
	}

	customerName := uc.resolveCustomerName(ctx, tenantUUID, sale.CustomerID)

	return &response.POSSaleDetailResponse{
		PosSaleID:         sale.ID,
		SaleNumber:        sale.SaleNumber,
		Items:             items,
		TotalItems:        sale.TotalItems(),
		SubtotalAmount:    sale.TotalAmount,
		DiscountAmount:    sale.DiscountAmount,
		FinalAmount:       sale.FinalAmount,
		AmountPaid:        sale.AmountPaid,
		Change:            sale.Change,
		Currency:          sale.Currency,
		PaymentMethodID:   sale.PaymentMethodID,
		PaymentMethodName: paymentMethodName,
		CustomerID:        sale.CustomerID,
		CustomerName:      customerName,
		CreatedAt:         sale.CreatedAt,
	}, nil
}

// resolveCustomerName resuelve el nombre del cliente para el comprobante,
// siguiendo el mismo patrón nil-safe que el medio de pago:
//   - customerID == uuid.Nil → "Consumidor Final" (no consulta customer-service)
//   - customerPort nil, error técnico, o cliente no encontrado/nombre vacío → "Cliente"
//
// Nunca propaga error: una falla resolviendo el nombre no debe romper el detalle.
func (uc *GetPosSaleUseCase) resolveCustomerName(ctx context.Context, tenantID, customerID uuid.UUID) string {
	if customerID == uuid.Nil {
		return consumidorFinalName
	}
	if uc.customerPort == nil {
		return customerNameFallback
	}
	name, err := uc.customerPort.GetName(ctx, tenantID, customerID)
	if err != nil || name == "" {
		return customerNameFallback
	}
	return name
}
