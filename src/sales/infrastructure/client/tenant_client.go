package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

// Tenant representa la respuesta de IAM para GET /tenants/:id
type Tenant struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// TenantClient obtiene datos de tenant desde iam-service vía Kong (S2S)
// H8 TTFS: Solo se usa cuando se detecta la primera venta del tenant
type TenantClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// NewTenantClient crea una nueva instancia del cliente con autenticación S2S
func NewTenantClient(baseURL string) *TenantClient {
	apiKey := os.Getenv("S2S_API_KEY")

	return &TenantClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		apiKey: apiKey,
	}
}

// GetTenant obtiene un tenant por ID desde IAM via ruta interna S2S
// Endpoint: GET /internal/iam/api/v1/tenants/:id
func (c *TenantClient) GetTenant(ctx context.Context, tenantID uuid.UUID) (*Tenant, error) {
	url := fmt.Sprintf("%s/internal/iam/api/v1/tenants/%s", c.baseURL, tenantID.String())

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Headers S2S
	req.Header.Set("X-Tenant-ID", tenantID.String())
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error calling iam-service: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("tenant not found: %s", tenantID)
		}
		return nil, fmt.Errorf("unexpected status from iam-service: %d", resp.StatusCode)
	}

	var tenant Tenant
	if err := json.Unmarshal(body, &tenant); err != nil {
		return nil, fmt.Errorf("error parsing tenant response: %w", err)
	}

	return &tenant, nil
}
