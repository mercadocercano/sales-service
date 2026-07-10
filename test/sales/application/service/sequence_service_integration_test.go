//go:build integration

// Integration test de optimistic locking en SequenceService tras migrar a
// WithRLSInTransaction (PLAT-E25 T11).
//
// Requiere una base PostgreSQL real con las migraciones aplicadas (incluye
// 024_rls_sales_tables, que habilita RLS fail-closed sobre document_sequences). NO corre
// con `go test ./...` (build tag `integration`). Para ejecutarlo:
//
//	export SALES_TEST_DB_DSN="postgres://sales_app:sales_app123@localhost:5432/order_db?sslmode=disable"
//	go test -tags=integration ./test/sales/application/service/...
//
// Verifica:
//   - N NextNumber concurrentes del mismo tenant/document_type asignan números
//     correlativos 1..N sin duplicados ni huecos.
//   - Al menos un caller sufre un conflicto de versión y reintenta — evidencia de que
//     ErrConcurrentUpdate sigue fluyendo intacto (sin wrap) fuera de WithRLSInTransaction
//     hasta el retry loop de NextNumber, tal como exige T11.
package service_test

import (
	"bytes"
	"context"
	"database/sql"
	"log"
	"os"
	"sort"
	"sync"
	"testing"

	"sales/src/sales/application/service"

	"github.com/google/uuid"
	"github.com/hornosg/go-shared/infrastructure/postgres"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("SALES_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("SALES_TEST_DB_DSN no seteada — se omite el test de integración de SequenceService")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	return db
}

// seedSequence inserta la fila inicial de document_sequences dentro de una transacción
// con contexto RLS fijado — una fila cruda sin SET LOCAL app.tenant_id no pasa el WITH
// CHECK de la policy tenant_isolation (024_rls_sales_tables), sea cual sea el rol de la
// conexión de test.
func seedSequence(t *testing.T, db *sql.DB, tenantID, documentType string) {
	t.Helper()
	err := postgres.WithRLSInTransaction(context.Background(), db, postgres.RLSContext{TenantID: tenantID}, func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO document_sequences (id, tenant_id, document_type, current_number, version, updated_at)
			 VALUES (gen_random_uuid(), $1, $2, 0, 1, NOW())`,
			tenantID, documentType,
		)
		return err
	})
	require.NoError(t, err)
}

func TestNextNumber_ConcurrentCallers_RetriesOnConflictAndAssignsDistinctNumbers(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	tenantID := uuid.New().String()
	const documentType = "SALES_ORDER_T11_TEST"
	seedSequence(t, db, tenantID, documentType)

	svc := service.NewSequenceService(db)

	// NextNumber loguea "Optimistic locking conflict" solo cuando tryGetNextNumber
	// devuelve ErrConcurrentUpdate — capturar el log es la forma de probar que el
	// sentinel siguió fluyendo intacto fuera de WithRLSInTransaction sin depender de
	// instrumentación ad-hoc en el código de producción.
	var logBuf bytes.Buffer
	prevOutput := log.Writer()
	log.SetOutput(&logBuf)
	defer log.SetOutput(prevOutput)

	const n = 10
	numbers := make([]int, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			number, err := svc.NextNumber(context.Background(), tenantID, documentType)
			numbers[idx] = number
			errs[idx] = err
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		require.NoErrorf(t, err, "caller %d debería obtener número tras reintentar", i)
	}

	sorted := append([]int(nil), numbers...)
	sort.Ints(sorted)
	for i := 0; i < n; i++ {
		require.Equal(t, i+1, sorted[i], "los números deben ser correlativos 1..N sin huecos ni duplicados")
	}

	require.Contains(t, logBuf.String(), "Optimistic locking conflict",
		"al menos un caller debe sufrir un conflicto de versión y reintentar — si esto falla, ErrConcurrentUpdate dejó de fluir fuera de WithRLSInTransaction")
}
