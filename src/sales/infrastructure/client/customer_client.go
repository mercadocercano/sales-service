package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// CustomerClient valida existencia de clientes en customer-service
type CustomerClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewCustomerClient crea una nueva instancia del cliente
func NewCustomerClient(baseURL string) *CustomerClient {
	return &CustomerClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Exists verifica si un cliente existe para un tenant específico
// Retorna:
// - true, nil: Cliente existe
// - false, nil: Cliente no existe (404)
// - false, error: Error técnico (no se pudo validar)
func (c *CustomerClient) Exists(ctx context.Context, tenantID uuid.UUID, customerID uuid.UUID) (bool, error) {
	// Construir URL
	url := fmt.Sprintf("%s/customers/api/v1/customers/%s", c.baseURL, customerID.String())

	// Crear request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("error creating request: %w", err)
	}

	// Agregar headers
	req.Header.Set("X-Tenant-ID", tenantID.String())

	// Ejecutar request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("error calling customer-service: %w", err)
	}
	defer resp.Body.Close()

	// Consumir body para liberar conexión
	_, _ = io.Copy(io.Discard, resp.Body)

	// Analizar response
	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("unexpected status from customer-service: %d", resp.StatusCode)
	}
}
