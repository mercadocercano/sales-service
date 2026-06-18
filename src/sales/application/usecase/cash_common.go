package usecase

import (
	"context"
	"encoding/json"
	"log"

	"github.com/google/uuid"

	"sales/src/sales/domain/entity"
	"sales/src/sales/domain/port"
)

// resolveCashPaymentMethodID devuelve el id del método de pago "cash" (efectivo) global,
// o uuid.Nil si no está disponible. ADR-003 §5: el efectivo se identifica por code=="cash".
func resolveCashPaymentMethodID(pm port.PaymentMethodPort) uuid.UUID {
	if pm == nil {
		return uuid.Nil
	}
	if info, ok := pm.GetByCode("cash"); ok {
		return info.ID
	}
	return uuid.Nil
}

// publishCashEvent publica un evento de dominio de caja como NOTIFICACIÓN (best-effort,
// ADR-003 §7: el arqueo nunca se reconstruye desde eventos). Un fallo NO revierte la
// operación financiera ya commiteada — solo se loguea.
func publishCashEvent(ctx context.Context, publisher port.EventPublisher, s *entity.CashRegisterSession, eventType string) {
	if publisher == nil {
		return
	}
	payload := map[string]interface{}{
		"tenant_id":        s.TenantID.String(),
		"point_of_sale_id": s.PointOfSaleID,
		"status":           s.Status.String(),
		"opened_by":        s.OpenedBy.String(),
	}
	if s.ExpectedAmount != nil {
		payload["expected_amount"] = s.ExpectedAmount.InexactFloat64()
	}
	if s.CountedAmount != nil {
		payload["counted_amount"] = s.CountedAmount.InexactFloat64()
	}
	if s.Difference != nil {
		payload["difference"] = s.Difference.InexactFloat64()
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("WARNING: failed to marshal cash event %s: %v", eventType, err)
		return
	}
	if err := publisher.Execute(ctx, s.ID.String(), "cash_register_session", eventType, body, "sales-service"); err != nil {
		log.Printf("WARNING: failed to publish %s: %v", eventType, err)
	}
}
