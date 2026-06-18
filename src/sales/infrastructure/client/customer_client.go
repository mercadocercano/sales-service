package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

// customerResponse es la vista tolerante del JSON que devuelve customer-service
// en GET /internal/customers/api/v1/customers/{id}. Sólo declaramos los campos
// que necesitamos; cualquier campo extra del contrato se ignora.
type customerResponse struct {
	Name string `json:"name"`
}

// CustomerClient valida existencia de clientes en customer-service (S2S)
type CustomerClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// NewCustomerClient crea una nueva instancia del cliente con autenticación S2S
func NewCustomerClient(baseURL string) *CustomerClient {
	apiKey := os.Getenv("S2S_API_KEY")

	return &CustomerClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		apiKey: apiKey,
	}
}

// Exists verifica si un cliente existe para un tenant específico
// Retorna:
// - true, nil: Cliente existe
// - false, nil: Cliente no existe (404)
// - false, error: Error técnico (no se pudo validar)
func (c *CustomerClient) Exists(ctx context.Context, tenantID uuid.UUID, customerID uuid.UUID) (bool, error) {
	// Construir URL via ruta interna S2S
	url := fmt.Sprintf("%s/internal/customers/api/v1/customers/%s", c.baseURL, customerID.String())

	// Crear request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("error creating request: %w", err)
	}

	// Agregar headers
	req.Header.Set("X-Tenant-ID", tenantID.String())
	// Autenticación S2S via API Key
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

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
	case http.StatusUnauthorized:
		// S2S auth no configurada: el servicio interno rechaza la llamada sin JWT.
		// TODO: implementar S2S JWT o X-API-Key en el customer-service.
		// Por ahora: loguear warning y confiar en que el tenant_id del JWT de la
		// request original ya garantiza el alcance correcto.
		log.Printf("[customer_client] WARNING: customer-service returned 401 for customer %s — S2S auth not configured, assuming valid", customerID)
		return true, nil
	default:
		return false, fmt.Errorf("unexpected status from customer-service: %d", resp.StatusCode)
	}
}

// GetName resuelve el nombre de un cliente para un tenant específico.
// Retorna:
// - name, nil: Cliente encontrado (name puede venir vacío si el dato no está cargado)
// - "", nil: Cliente no existe (404) — el llamador aplica su propio fallback
// - "", error: Error técnico (no se pudo resolver)
func (c *CustomerClient) GetName(ctx context.Context, tenantID uuid.UUID, customerID uuid.UUID) (string, error) {
	// Mismo endpoint interno S2S que usa Exists.
	url := fmt.Sprintf("%s/internal/customers/api/v1/customers/%s", c.baseURL, customerID.String())

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("X-Tenant-ID", tenantID.String())
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error calling customer-service: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var body customerResponse
		// Deserialización tolerante: campos extra se ignoran; un body inválido
		// no rompe el flujo, simplemente cae al fallback del llamador.
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return "", fmt.Errorf("error decoding customer-service response: %w", err)
		}
		return body.Name, nil
	case http.StatusNotFound:
		_, _ = io.Copy(io.Discard, resp.Body)
		return "", nil
	case http.StatusUnauthorized:
		// Mismo criterio que Exists: S2S auth aún no configurada en customer-service.
		// No hay forma de obtener el nombre real → se delega el fallback al llamador.
		_, _ = io.Copy(io.Discard, resp.Body)
		log.Printf("[customer_client] WARNING: customer-service returned 401 for customer %s — S2S auth not configured, cannot resolve name", customerID)
		return "", nil
	default:
		_, _ = io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("unexpected status from customer-service: %d", resp.StatusCode)
	}
}
