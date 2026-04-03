package adapter

import (
	"sales/src/sales/domain/port"
	"sales/src/sales/infrastructure/cache"

	"github.com/google/uuid"
)

// PaymentMethodAdapter adapta cache.PaymentMethodCache a port.PaymentMethodPort
type PaymentMethodAdapter struct {
	cache *cache.PaymentMethodCache
}

// NewPaymentMethodAdapter crea un nuevo adaptador
func NewPaymentMethodAdapter(c *cache.PaymentMethodCache) *PaymentMethodAdapter {
	return &PaymentMethodAdapter{cache: c}
}

func (a *PaymentMethodAdapter) Get(id uuid.UUID) (port.PaymentMethodInfo, bool) {
	pm, ok := a.cache.Get(id)
	if !ok {
		return port.PaymentMethodInfo{}, false
	}
	return port.PaymentMethodInfo{
		ID:   pm.ID,
		Code: pm.Code,
		Name: pm.Name,
	}, true
}

func (a *PaymentMethodAdapter) GetName(id uuid.UUID) string {
	return a.cache.GetName(id)
}
