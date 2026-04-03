package mocks

import (
	"encoding/json"

	"github.com/stretchr/testify/mock"
)

// MockPIMPort es un mock de port.PIMPort
type MockPIMPort struct {
	mock.Mock
}

func (m *MockPIMPort) GetSnapshotForSKU(tenantID, sku string) (json.RawMessage, json.RawMessage, error) {
	args := m.Called(tenantID, sku)
	var product, variant json.RawMessage
	if args.Get(0) != nil {
		product = args.Get(0).(json.RawMessage)
	}
	if args.Get(1) != nil {
		variant = args.Get(1).(json.RawMessage)
	}
	return product, variant, args.Error(2)
}
