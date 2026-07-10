//go:build integration

// Test de aislamiento cross-tenant adversarial (PLAT-E25 T13).
//
// Levanta un Postgres efímero (testcontainers), aplica TODAS las migraciones reales del
// servicio (incluida 024_rls_sales_tables, RLS fail-closed, y 025_create_app_role) y
// conecta bajo un rol sin BYPASSRLS (mismo patrón que `sales_app` en lab-postgres, T3) —
// un superuser SIEMPRE bypasea RLS, así que probar contra él daría falsos verdes (probado
// empíricamente en E24: sql_ledger_repository_test.go).
//
// Cobertura mínima exigida (@dev-security + @dev-architect, revisión 2026-07-08):
//   - sales_orders (columna propia) + sales_order_items (hija, subquery contra el padre)
//   - pos_sales (columna propia) + pos_sale_items (hija, subquery contra el padre)
//   - cash_movements (columna propia, alto volumen de escritura)
//   - Fail-closed sin contexto de tenant (ninguna query se satura por accidente)
//   - Save de aggregate completo bajo un único RLSContext (parent+hijos, WITH CHECK de la
//     hija ve al parent recién insertado en la misma tx — requisito T4/T5)
//
// El contenedor se levanta UNA sola vez para todo el binario de test vía TestMain (no por
// TestXxx) — así los ~3-5s de arranque no se pagan por cada función de test.
//
// Para correrlo:
//
//	go test -tags=integration ./test/sales/infrastructure/persistence/... -run CrossTenant
package persistence_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/hornosg/go-shared/infrastructure/postgres"

	"sales/src/sales/domain/entity"
	"sales/src/sales/infrastructure/persistence"
	"sales/test/sales/domain/mother"
)

// appRoleName/appRolePassword replican en el Postgres efímero el rol `sales_app` creado en
// lab-postgres por T3: sin este rol, la conexión de test cae al usuario `postgres` del
// contenedor (superuser) y un superuser SIEMPRE bypasea RLS — FORCE ROW LEVEL SECURITY no
// lo alcanza.
const (
	appRoleName     = "sales_app_test"
	appRolePassword = "sales_app_test"
)

// appDB es la única conexión que usan los TestXxx de este archivo — bajo el rol sin
// BYPASSRLS, la que de verdad queda sujeta a las policies de 024_rls_sales_tables.
var appDB *sql.DB

// TestMain levanta el Postgres efímero, aplica las migraciones reales y crea el rol de
// aplicación UNA sola vez para todo el binario — se comparte entre todas las TestXxx de este
// paquete en vez de pagar el arranque del contenedor por cada una.
func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("sales_test"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	if err != nil {
		log.Fatalf("error starting postgres container: %v", err)
	}

	superConnStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("error getting connection string: %v", err)
	}

	superDB, err := sql.Open("postgres", superConnStr)
	if err != nil {
		log.Fatalf("error opening database: %v", err)
	}
	if err := superDB.PingContext(ctx); err != nil {
		log.Fatalf("error pinging database: %v", err)
	}

	if err := applyMigrations(superDB); err != nil {
		log.Fatalf("error applying migrations: %v", err)
	}
	if err := createAppRole(superDB); err != nil {
		log.Fatalf("error creating app role: %v", err)
	}

	appConnStr, err := withCredentials(superConnStr, appRoleName, appRolePassword)
	if err != nil {
		log.Fatalf("error building app connection string: %v", err)
	}
	appDB, err = sql.Open("postgres", appConnStr)
	if err != nil {
		log.Fatalf("error opening app-role database: %v", err)
	}
	if err := appDB.PingContext(ctx); err != nil {
		log.Fatalf("error pinging database as %s: %v", appRoleName, err)
	}

	code := m.Run()

	_ = appDB.Close()
	_ = superDB.Close()
	_ = container.Terminate(ctx)

	os.Exit(code)
}

// createAppRole crea (con el superuser) un rol sin DDL ni BYPASSRLS y le otorga los mismos
// grants mínimos que 025_create_app_role.up.sql otorga a `sales_app` en lab-postgres —
// réplica del rol real, no un stand-in simplificado.
func createAppRole(superDB *sql.DB) error {
	stmts := []string{
		`CREATE ROLE ` + appRoleName + ` WITH LOGIN PASSWORD '` + appRolePassword + `' NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS`,
		`GRANT CONNECT ON DATABASE sales_test TO ` + appRoleName,
		`GRANT USAGE ON SCHEMA public TO ` + appRoleName,
		`GRANT SELECT, INSERT, UPDATE, DELETE ON
			sales_orders, sales_order_items, pos_sales, pos_sale_items, sales_payments,
			document_sequences, customer_credits, customer_credit_applications, tenant_metrics,
			cash_register_sessions, cash_movements, cash_audit_log
		 TO ` + appRoleName,
	}
	for _, stmt := range stmts {
		if _, err := superDB.Exec(stmt); err != nil {
			return fmt.Errorf("stmt %q: %w", stmt, err)
		}
	}
	return nil
}

// withCredentials reconstruye la connection string del contenedor reemplazando usuario y
// contraseña — evita depender de las APIs internas de testcontainers para host/puerto.
func withCredentials(connStr, user, password string) (string, error) {
	u, err := url.Parse(connStr)
	if err != nil {
		return "", err
	}
	u.User = url.UserPassword(user, password)
	return u.String(), nil
}

// applyMigrations corre en orden TODAS las migraciones .up.sql de esquema de sales-service
// (001..024, incluida 024_rls_sales_tables) — el mismo esquema que corre en
// lab-postgres/order_db tras PLAT-E25 T2. 025_create_app_role queda afuera a propósito: es
// role/grant DDL que hardcodea `GRANT CONNECT ON DATABASE order_db` (el nombre real de la DB
// en lab-postgres) — no existe una DB "order_db" en el contenedor efímero (creado como
// "sales_test"), y este archivo ya replica el rol de aplicación para el container vía
// createAppRole() con sus propios grants.
func applyMigrations(db *sql.DB) error {
	dir := "../../../../migrations"
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	var files []string
	for _, e := range dirEntries {
		name := e.Name()
		if e.IsDir() || filepath.Ext(name) != ".sql" || strings.HasSuffix(name, ".down.sql") {
			continue
		}
		if name == "025_create_app_role.up.sql" {
			continue
		}
		files = append(files, filepath.Join(dir, name))
	}
	sort.Strings(files)
	if len(files) == 0 {
		return fmt.Errorf("no se encontraron migraciones .up.sql en %s", dir)
	}

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("migración %s: %w", f, err)
		}
	}
	return nil
}

// countRaw cuenta filas bajo un RLSContext dado — usado para las aserciones de invisibilidad
// cross-tenant (nunca vía el WHERE tenant_id=... del repository, para no confundir
// aislamiento por RLS con un filtro de la query).
func countRaw(t *testing.T, rc postgres.RLSContext, query string, args ...interface{}) int {
	t.Helper()
	var count int
	err := postgres.WithRLSInTransaction(context.Background(), appDB, rc, func(ctx context.Context, tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, query, args...).Scan(&count)
	})
	require.NoError(t, err)
	return count
}

// TestOrderRepository_CrossTenantIsolation_FailsClosed cubre sales_orders (columna propia)
// y sales_order_items (hija, policy por subquery contra el padre) y, de paso, satisface el
// requisito de "Save de aggregate completo" (@dev-architect): OrderPostgresRepository.Save
// inserta parent+hijos en una única transacción RLS — el WITH CHECK de la hija solo ve el
// parent recién insertado porque comparten contexto/tx (T4).
func TestOrderRepository_CrossTenantIsolation_FailsClosed(t *testing.T) {
	repo := persistence.NewOrderPostgresRepository(appDB)

	tenantA := uuid.New().String()
	tenantB := uuid.New().String()

	// WithUnitPrice activa los snapshots del item (NewOrderItemWithSnapshots) — el
	// constructor plano NewOrderItem no está en uso por ningún caso de uso real (solo por
	// mother.RandomOrderItem() sin opciones) y deja ProductSnapshot/VariantSnapshot como
	// json.RawMessage nil, que lib/pq inserta como cadena vacía en la columna jsonb en vez
	// de NULL ("invalid input syntax for type json") — no es el escenario que ejecuta
	// producción, así que el test usa el mismo constructor que sí usan los casos de uso.
	item := mother.RandomOrderItem().WithUnitPrice(100).MustBuild()
	order := mother.RandomOrder().WithTenantID(tenantA).WithItems([]entity.OrderItem{*item}).MustBuild()
	require.NoError(t, repo.Save(context.Background(), order))

	t.Run("visible para el propio tenant vía el repository", func(t *testing.T) {
		got, err := repo.FindByID(context.Background(), order.OrderID, tenantA)
		require.NoError(t, err)
		require.Equal(t, order.OrderID, got.OrderID)
		require.Len(t, got.Items, len(order.Items))
	})

	t.Run("sales_orders invisible por id crudo desde otro tenant", func(t *testing.T) {
		count := countRaw(t, postgres.RLSContext{TenantID: tenantB},
			`SELECT count(*) FROM sales_orders WHERE id = $1`, order.OrderID)
		require.Equal(t, 0, count, "la orden de tenantA es visible desde tenantB")
	})

	t.Run("sales_order_items invisible desde otro tenant (policy subquery contra el padre)", func(t *testing.T) {
		count := countRaw(t, postgres.RLSContext{TenantID: tenantB},
			`SELECT count(*) FROM sales_order_items WHERE sales_order_id = $1`, order.OrderID)
		require.Equal(t, 0, count, "los items de la orden de tenantA son visibles desde tenantB")
	})

	t.Run("sales_orders inescribible: INSERT con tenant_id=A bajo sesión de tenant B viola WITH CHECK", func(t *testing.T) {
		err := postgres.WithRLSInTransaction(context.Background(), appDB, postgres.RLSContext{TenantID: tenantB},
			func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
					INSERT INTO sales_orders (id, tenant_id, customer_id, status, total_amount, created_at, updated_at, version)
					VALUES ($1, $2, $3, 'CREATED', 100, now(), now(), 1)
				`, uuid.New().String(), tenantA, uuid.New().String())
				return err
			})
		require.Error(t, err, "esperaba que el INSERT con tenant_id=A bajo sesión B fallara por WITH CHECK")
	})

	t.Run("sales_order_items inescribible bajo sesión B: la subquery no ve el parent de tenant A", func(t *testing.T) {
		err := postgres.WithRLSInTransaction(context.Background(), appDB, postgres.RLSContext{TenantID: tenantB},
			func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
					INSERT INTO sales_order_items (id, sales_order_id, sku, quantity, unit_price, created_at)
					VALUES ($1, $2, 'SKU-ADVERSARIAL', 1, 10, now())
				`, uuid.New().String(), order.OrderID)
				return err
			})
		require.Error(t, err, "esperaba que el INSERT de un item contra el parent de tenantA fallara bajo sesión B")
	})
}

// TestPosSaleRepository_CrossTenantIsolation_FailsClosed cubre pos_sales (columna propia) y
// pos_sale_items (segunda tabla hija, subquery contra pos_sales) — patrón sin precedente en
// E24/E26, no alcanza con probar sales_order_items: cada tabla hija tiene padre e índice
// propios. PosSalePostgresRepository.Create también inserta parent+hijos en una única
// transacción RLS (T5), mismo requisito que OrderPostgresRepository.Save.
func TestPosSaleRepository_CrossTenantIsolation_FailsClosed(t *testing.T) {
	repo := persistence.NewPosSalePostgresRepository(appDB)

	tenantA := uuid.New()
	tenantB := uuid.New().String()

	sale := mother.RandomPosSale().WithTenantID(tenantA).MustBuild()
	require.NoError(t, repo.Create(context.Background(), sale))

	t.Run("visible para el propio tenant vía consulta cruda", func(t *testing.T) {
		count := countRaw(t, postgres.RLSContext{TenantID: tenantA.String()},
			`SELECT count(*) FROM pos_sales WHERE id = $1`, sale.ID)
		require.Equal(t, 1, count)
	})

	t.Run("pos_sales invisible por id crudo desde otro tenant", func(t *testing.T) {
		count := countRaw(t, postgres.RLSContext{TenantID: tenantB},
			`SELECT count(*) FROM pos_sales WHERE id = $1`, sale.ID)
		require.Equal(t, 0, count, "la venta POS de tenantA es visible desde tenantB")
	})

	t.Run("pos_sale_items invisible desde otro tenant (policy subquery contra el padre)", func(t *testing.T) {
		count := countRaw(t, postgres.RLSContext{TenantID: tenantB},
			`SELECT count(*) FROM pos_sale_items WHERE pos_sale_id = $1`, sale.ID)
		require.Equal(t, 0, count, "los items de la venta de tenantA son visibles desde tenantB")
	})

	t.Run("pos_sales inescribible: INSERT con tenant_id=A bajo sesión de tenant B viola WITH CHECK", func(t *testing.T) {
		err := postgres.WithRLSInTransaction(context.Background(), appDB, postgres.RLSContext{TenantID: tenantB},
			func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
					INSERT INTO pos_sales (id, tenant_id, payment_method_id, total_amount)
					VALUES ($1, $2, $3, 50)
				`, uuid.New().String(), tenantA.String(), uuid.New().String())
				return err
			})
		require.Error(t, err, "esperaba que el INSERT con tenant_id=A bajo sesión B fallara por WITH CHECK")
	})

	t.Run("pos_sale_items inescribible bajo sesión B: la subquery no ve el parent de tenant A", func(t *testing.T) {
		err := postgres.WithRLSInTransaction(context.Background(), appDB, postgres.RLSContext{TenantID: tenantB},
			func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
					INSERT INTO pos_sale_items (id, pos_sale_id, sku, product_name, quantity, unit_price, subtotal, stock_entry_id)
					VALUES ($1, $2, 'SKU-ADVERSARIAL', 'Adversarial', 1, 10, 10, $3)
				`, uuid.New().String(), sale.ID, uuid.New().String())
				return err
			})
		require.Error(t, err, "esperaba que el INSERT de un item contra el parent de tenantA fallara bajo sesión B")
	})
}

// TestCashMovements_CrossTenantIsolation_FailsClosed cubre cash_movements: columna tenant_id
// propia, y la tabla de mayor volumen de escritura del grupo (arqueo de caja). La FK
// cash_movements.cash_register_session_id → cash_register_sessions SIEMPRE bypasea RLS (así
// documenta Postgres las referential integrity checks) — lo que aísla acá es el WITH CHECK
// de la policy tenant_isolation sobre cash_movements, no la FK.
func TestCashMovements_CrossTenantIsolation_FailsClosed(t *testing.T) {
	tenantA := uuid.New().String()
	tenantB := uuid.New().String()

	var sessionID, movementID string
	err := postgres.WithRLSInTransaction(context.Background(), appDB, postgres.RLSContext{TenantID: tenantA},
		func(ctx context.Context, tx *sql.Tx) error {
			sessionID = uuid.New().String()
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO cash_register_sessions (id, tenant_id, point_of_sale_id, opened_by, opening_amount)
				VALUES ($1, $2, 'POS-1', $3, 0)
			`, sessionID, tenantA, uuid.New().String()); err != nil {
				return err
			}
			movementID = uuid.New().String()
			_, err := tx.ExecContext(ctx, `
				INSERT INTO cash_movements (id, tenant_id, cash_register_session_id, type, amount, reason, created_by)
				VALUES ($1, $2, $3, 'income', 500, 'venta adversarial', $4)
			`, movementID, tenantA, sessionID, uuid.New().String())
			return err
		})
	require.NoError(t, err)

	t.Run("visible para el propio tenant vía consulta cruda", func(t *testing.T) {
		count := countRaw(t, postgres.RLSContext{TenantID: tenantA},
			`SELECT count(*) FROM cash_movements WHERE id = $1`, movementID)
		require.Equal(t, 1, count)
	})

	t.Run("cash_movements invisible por id crudo desde otro tenant", func(t *testing.T) {
		count := countRaw(t, postgres.RLSContext{TenantID: tenantB},
			`SELECT count(*) FROM cash_movements WHERE id = $1`, movementID)
		require.Equal(t, 0, count, "el movimiento de caja de tenantA es visible desde tenantB")
	})

	t.Run("inescribible: INSERT con tenant_id=A bajo sesión de tenant B viola WITH CHECK (incluso con FK válida)", func(t *testing.T) {
		err := postgres.WithRLSInTransaction(context.Background(), appDB, postgres.RLSContext{TenantID: tenantB},
			func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
					INSERT INTO cash_movements (id, tenant_id, cash_register_session_id, type, amount, reason, created_by)
					VALUES ($1, $2, $3, 'income', 999, 'adversarial cross-tenant', $4)
				`, uuid.New().String(), tenantA, sessionID, uuid.New().String())
				return err
			})
		require.Error(t, err, "esperaba que el INSERT con tenant_id=A bajo sesión B fallara por WITH CHECK aunque la FK a la sesión de A sea válida (las FK bypasean RLS)")
	})
}

// TestFailClosed_WithoutTenantContext es el guard que prueba que ningún repository de los 9
// puntos de código quedó sin migrar a WithRLSInTransaction: una consulta corrida SIN fijar
// app.tenant_id (ni SET LOCAL ni transacción RLS) debe devolver cero filas o errorar, nunca
// todas. Corre después de los tests de arriba a propósito: reutiliza el pool `appDB` cuyas
// conexiones físicas YA tuvieron SET LOCAL app.tenant_id fijado (y reseteado al commitear) —
// ese es el escenario real de un pool reutilizado entre requests de distintos tenants, más
// representativo que una conexión nunca tocada.
func TestFailClosed_WithoutTenantContext(t *testing.T) {
	tables := []string{
		"sales_orders", "pos_sales", "cash_movements", "sales_order_items", "pos_sale_items",
	}
	for _, table := range tables {
		t.Run(table, func(t *testing.T) {
			var count int
			err := appDB.QueryRowContext(context.Background(), `SELECT count(*) FROM `+table).Scan(&count)
			if err != nil {
				return // fail-closed vía error (current_setting sin valor fijado): comportamiento esperado
			}
			require.Equal(t, 0, count, "tabla %s devolvió filas sin contexto de tenant fijado — RLS no está aislando", table)
		})
	}
}
