package cache

import (
	"database/sql"
	"log"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// PaymentMethod representa un método de pago en el cache
type PaymentMethod struct {
	ID   uuid.UUID
	Code string
	Name string
}

// PaymentMethodCache cache en memoria de métodos de pago globales
// HITO: POST /pos/sale devuelve DTO listo para imprimir
type PaymentMethodCache struct {
	methods map[uuid.UUID]PaymentMethod
	mu      sync.RWMutex
}

// NewPaymentMethodCache crea un nuevo cache de métodos de pago
func NewPaymentMethodCache() *PaymentMethodCache {
	return &PaymentMethodCache{
		methods: make(map[uuid.UUID]PaymentMethod),
	}
}

// LoadFromDB carga los métodos de pago globales desde la base de datos payment_method_db
func (c *PaymentMethodCache) LoadFromDB(db *sql.DB) error {
	log.Println("🔄 Loading global payment methods into cache...")

	// Query para obtener todos los métodos de pago globales
	query := `
		SELECT id, code, name 
		FROM payment_methods 
		WHERE tenant_id IS NULL AND is_active = true
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("⚠️  Warning: Could not load payment methods: %v", err)
		log.Println("⚠️  Continuing without payment method cache")
		return err
	}
	defer rows.Close()

	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	for rows.Next() {
		var pm PaymentMethod
		err := rows.Scan(&pm.ID, &pm.Code, &pm.Name)
		if err != nil {
			log.Printf("⚠️  Error scanning payment method: %v", err)
			continue
		}
		c.methods[pm.ID] = pm
		count++
	}

	log.Printf("✅ Loaded %d payment methods into cache", count)
	for _, pm := range c.methods {
		log.Printf("   - %s: %s (%s)", pm.ID, pm.Name, pm.Code)
	}

	return nil
}

// Get obtiene el nombre de un método de pago por ID
func (c *PaymentMethodCache) Get(id uuid.UUID) (PaymentMethod, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	pm, ok := c.methods[id]
	return pm, ok
}

// GetByCode busca un método de pago por código (ej: "cash", "efectivo")
func (c *PaymentMethodCache) GetByCode(code string) (PaymentMethod, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, pm := range c.methods {
		if strings.EqualFold(pm.Code, code) {
			return pm, true
		}
	}
	return PaymentMethod{}, false
}

// GetName obtiene solo el nombre de un método de pago por ID
func (c *PaymentMethodCache) GetName(id uuid.UUID) string {
	pm, ok := c.Get(id)
	if !ok {
		return "Unknown"
	}
	return pm.Name
}
