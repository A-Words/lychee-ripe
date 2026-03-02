package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/lychee-ripe/gateway/internal/config"
)

func TestMigrateCreatesCoreTablesAndIsIdempotent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "nested", "gateway.db")

	conn, err := Open(ctx, config.DBConfig{
		Path:          dbPath,
		BusyTimeoutMS: 5000,
		JournalMode:   "WAL",
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer conn.Close()

	if err := Migrate(ctx, conn); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := Migrate(ctx, conn); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	requiredTables := []string{
		"schema_migrations",
		"batches",
		"anchor_proofs",
		"reconcile_jobs",
		"reconcile_job_items",
		"audit_logs",
	}

	for _, table := range requiredTables {
		if !tableExists(ctx, t, conn, table) {
			t.Fatalf("expected table %q to exist", table)
		}
	}

	var migrationRows int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(1) FROM schema_migrations WHERE version = 1").Scan(&migrationRows); err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	if migrationRows != 1 {
		t.Fatalf("migration version=1 rows = %d, want 1", migrationRows)
	}
}

func tableExists(ctx context.Context, t *testing.T, conn *sql.DB, table string) bool {
	t.Helper()

	var count int
	err := conn.QueryRowContext(
		ctx,
		"SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?",
		table,
	).Scan(&count)
	if err != nil {
		t.Fatalf("query sqlite_master for table %s: %v", table, err)
	}
	return count == 1
}
