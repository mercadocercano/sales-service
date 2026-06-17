package port

// SalesEvent es el payload canónico para eventos de dominio de ventas (ADR-001).
// Campos flat, named. Los nombres comunes (tenant_id, user_id) son idénticos al resto
// de la flota para que el LogQL cross-service funcione. Todos opcionales salvo Event.
type SalesEvent struct {
	Event         string // <domain>.<action>_<result>, p.ej. "orders.pos_sale_completed"
	TenantID      string
	UserID        string
	SaleID        string
	SKU           string
	PaymentMethod string
	Items         int
	FinalAmount   string
	Reason        string
}

// SalesEventLogger es el puerto para emitir eventos canónicos de ventas.
// El código de aplicación depende de esta interfaz; el adapter (JSON a stdout,
// Loki push, etc.) la implementa. Nunca al revés.
type SalesEventLogger interface {
	Log(e SalesEvent)
}
