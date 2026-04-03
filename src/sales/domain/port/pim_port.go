package port

import "encoding/json"

// PIMPort define las operaciones de PIM disponibles para los use cases
type PIMPort interface {
	GetSnapshotForSKU(tenantID, sku string) (productSnapshot, variantSnapshot json.RawMessage, err error)
}
