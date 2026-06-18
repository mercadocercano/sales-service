//go:build integration

// Integration test de la numeración atómica de comprobantes POS (E18 Tramo B).
//
// Requiere una base PostgreSQL real con las migraciones aplicadas. NO corre con
// `go test ./...` (build tag `integration`). Para ejecutarlo:
//
//	export SALES_TEST_DB_DSN="postgres://sales:sales@localhost:5433/sales_e2e?sslmode=disable"
//	go test -tags=integration ./test/sales/infrastructure/persistence/...
//
// Verifica:
//   - T-003: Create asigna sale_number correlativo y lo persiste (1, 2, 3, ...).
//   - T-004: bajo N Creates concurrentes del mismo tenant no hay números duplicados
//     ni saltos (unicidad + atomicidad por SELECT FOR UPDATE dentro de la tx).
package persistence_test

import (
	"context"
	"database/sql"
	"os"
	"sort"
	"sync"
	"testing"

	"sales/src/sales/domain/entity"
	"sales/src/sales/infrastructure/persistence"

	_ "github.com/lib/pq"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("SALES_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("SALES_TEST_DB_DSN no seteada — se omite el test de integración de numeración")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	return db
}

func newTestSale(tenantID uuid.UUID) *entity.PosSale {
	item, _ := entity.NewPosSaleItem(uuid.Nil, "SKU-INT", "Producto Int", 1, decimal.NewFromFloat(100), uuid.New())
	sale, _ := entity.NewPosSale(tenantID, uuid.Nil, uuid.New(), []entity.PosSaleItem{*item}, decimal.Zero, decimal.NewFromFloat(100), "ARS")
	return sale
}

// T-003
func TestCreate_AssignsSequentialSaleNumber(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	repo := persistence.NewPosSalePostgresRepository(db)
	ctx := context.Background()
	tenantID := uuid.New() // tenant fresco → secuencia arranca en 0

	for expected := 1; expected <= 3; expected++ {
		sale := newTestSale(tenantID)
		require.NoError(t, repo.Create(ctx, sale))
		require.NotNil(t, sale.SaleNumber)
		require.Equal(t, expected, *sale.SaleNumber)
	}
}

// T-004
func TestCreate_ConcurrentSales_NoDuplicateNumbers(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	repo := persistence.NewPosSalePostgresRepository(db)
	ctx := context.Background()
	tenantID := uuid.New()

	const n = 25
	numbers := make([]int, n)
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var firstErr error

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sale := newTestSale(tenantID)
			if err := repo.Create(ctx, sale); err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
				return
			}
			numbers[idx] = *sale.SaleNumber
		}(i)
	}
	wg.Wait()
	require.NoError(t, firstErr)

	sort.Ints(numbers)
	for i := 0; i < n; i++ {
		require.Equal(t, i+1, numbers[i], "los números deben ser correlativos 1..N sin huecos ni duplicados")
	}
}
