package adapter

import (
	"encoding/json"

	"sales/src/sales/infrastructure/client"
)

// PIMAdapter adapta client.PIMClient a port.PIMPort
type PIMAdapter struct {
	client *client.PIMClient
}

// NewPIMAdapter crea un nuevo adaptador
func NewPIMAdapter(c *client.PIMClient) *PIMAdapter {
	return &PIMAdapter{client: c}
}

func (a *PIMAdapter) GetSnapshotForSKU(tenantID, sku string) (json.RawMessage, json.RawMessage, error) {
	return a.client.GetSnapshotForSKU(tenantID, sku)
}
