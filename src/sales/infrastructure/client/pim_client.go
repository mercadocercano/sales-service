package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// PIMProductResponse representa la respuesta de PIM para un producto
type PIMProductResponse struct {
	ProductID   string `json:"product_id"`
	ProductSKU  string `json:"product_sku"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CategoryID  string `json:"category_id"`
	BrandID     string `json:"brand_id"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	// Campos adicionales que puedan existir
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// PIMVariantResponse representa la respuesta de PIM para una variante
type PIMVariantResponse struct {
	VariantID    string          `json:"variant_id"`
	ProductID    string          `json:"product_id"`
	VariantSKU   string          `json:"variant_sku"`
	Name         string          `json:"name"`
	Price        float64         `json:"price"`
	CostPrice    float64         `json:"cost_price"`
	ComparePrice float64         `json:"compare_price"`
	Attributes   json.RawMessage `json:"attributes,omitempty"`
	Status       string          `json:"status"`
	CreatedAt    string          `json:"created_at"`
	UpdatedAt    string          `json:"updated_at"`
	// Campos adicionales
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// PIMClient cliente HTTP para comunicarse con PIM service vía Kong
type PIMClient struct {
	httpClient *http.Client
	kongURL    string
	pimPath    string
}

// NewPIMClient crea una nueva instancia del cliente PIM
func NewPIMClient() *PIMClient {
	kongURL := os.Getenv("KONG_INTERNAL_URL")
	if kongURL == "" {
		kongURL = "http://kong:8000" // Default para entorno Docker
	}

	pimPath := os.Getenv("PIM_SERVICE_PATH")
	if pimPath == "" {
		pimPath = "/pim" // Default
	}

	return &PIMClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		kongURL: kongURL,
		pimPath: pimPath,
	}
}

// GetVariantBySKU obtiene una variante por su SKU
func (c *PIMClient) GetVariantBySKU(tenantID, authToken, sku string) (*PIMVariantResponse, error) {
	// Construir URL completa vía Kong
	url := fmt.Sprintf("%s%s/api/v1/variants/by-sku/%s", c.kongURL, c.pimPath, sku)

	// Crear request HTTP
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Headers obligatorios
	req.Header.Set("X-Tenant-ID", tenantID)

	// Pasar Authorization si existe
	if authToken != "" {
		req.Header.Set("Authorization", authToken)
	}

	// Ejecutar request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error calling pim-service: %w", err)
	}
	defer resp.Body.Close()

	// Leer response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	// Verificar status code
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("variant not found: %s", sku)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pim-service returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var variant PIMVariantResponse
	if err := json.Unmarshal(body, &variant); err != nil {
		return nil, fmt.Errorf("error unmarshalling variant response: %w", err)
	}

	return &variant, nil
}

// GetProductByID obtiene un producto por su ID
func (c *PIMClient) GetProductByID(tenantID, authToken, productID string) (*PIMProductResponse, error) {
	// Construir URL completa vía Kong
	url := fmt.Sprintf("%s%s/api/v1/products/%s", c.kongURL, c.pimPath, productID)

	// Crear request HTTP
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Headers obligatorios
	req.Header.Set("X-Tenant-ID", tenantID)

	// Pasar Authorization si existe
	if authToken != "" {
		req.Header.Set("Authorization", authToken)
	}

	// Ejecutar request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error calling pim-service: %w", err)
	}
	defer resp.Body.Close()

	// Leer response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	// Verificar status code
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("product not found: %s", productID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pim-service returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var product PIMProductResponse
	if err := json.Unmarshal(body, &product); err != nil {
		return nil, fmt.Errorf("error unmarshalling product response: %w", err)
	}

	return &product, nil
}

// GetSnapshotForSKU obtiene tanto el producto como la variante y retorna ambos como JSON
func (c *PIMClient) GetSnapshotForSKU(tenantID, authToken, sku string) (productSnapshot, variantSnapshot json.RawMessage, err error) {
	// 1. Obtener variante por SKU
	variant, err := c.GetVariantBySKU(tenantID, authToken, sku)
	if err != nil {
		return nil, nil, fmt.Errorf("error fetching variant: %w", err)
	}

	// 2. Obtener producto asociado
	product, err := c.GetProductByID(tenantID, authToken, variant.ProductID)
	if err != nil {
		return nil, nil, fmt.Errorf("error fetching product: %w", err)
	}

	// 3. Serializar ambos a JSON para almacenar como snapshot
	productJSON, err := json.Marshal(product)
	if err != nil {
		return nil, nil, fmt.Errorf("error marshalling product: %w", err)
	}

	variantJSON, err := json.Marshal(variant)
	if err != nil {
		return nil, nil, fmt.Errorf("error marshalling variant: %w", err)
	}

	return productJSON, variantJSON, nil
}
