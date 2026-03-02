package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lychee-ripe/gateway/internal/config"
)

func TestOpenGORMAndAutoMigrateSQLite(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := config.DBConfig{
		Driver:           "sqlite",
		DSN:              filepath.Join(t.TempDir(), "gateway.db"),
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetimeS: 300,
		SQLite: config.SQLiteDBConfig{
			JournalMode:   "WAL",
			BusyTimeoutMS: 5000,
		},
	}

	gdb, err := OpenGORM(ctx, cfg)
	if err != nil {
		t.Fatalf("open sqlite gorm: %v", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	defer sqlDB.Close()

	if err := AutoMigrate(ctx, gdb); err != nil {
		t.Fatalf("first auto migrate: %v", err)
	}
	if err := AutoMigrate(ctx, gdb); err != nil {
		t.Fatalf("second auto migrate: %v", err)
	}

	tables := []string{"batches", "anchor_proofs", "reconcile_jobs", "reconcile_job_items", "audit_logs"}
	for _, table := range tables {
		var count int
		err := gdb.WithContext(ctx).Raw(
			"SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?",
			table,
		).Scan(&count).Error
		if err != nil {
			t.Fatalf("query sqlite_master for table %s: %v", table, err)
		}
		if count != 1 {
			t.Fatalf("table %s not found", table)
		}
	}
}

func TestOpenGORMAndAutoMigratePostgresOptional(t *testing.T) {
	pgDSN := strings.TrimSpace(getEnv("LYCHEE_GATEWAY_TEST_PG_DSN"))
	if pgDSN == "" {
		t.Skip("LYCHEE_GATEWAY_TEST_PG_DSN not set, skip postgres integration test")
	}

	ctx := context.Background()
	schema := fmt.Sprintf("test_%d", time.Now().UTC().UnixNano())
	cfg := config.DBConfig{
		Driver:           "postgres",
		DSN:              pgDSN,
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetimeS: 300,
		Postgres: config.PostgresDBConfig{
			SSLMode: "disable",
			Schema:  schema,
		},
	}

	gdb, err := OpenGORM(ctx, cfg)
	if err != nil {
		t.Fatalf("open postgres gorm: %v", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	defer sqlDB.Close()

	if err := gdb.WithContext(ctx).Exec(`CREATE SCHEMA IF NOT EXISTS "` + schema + `"`).Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}
	defer gdb.Exec(`DROP SCHEMA IF EXISTS "` + schema + `" CASCADE`)

	if err := AutoMigrate(ctx, gdb); err != nil {
		t.Fatalf("auto migrate postgres: %v", err)
	}

	tables := []string{"batches", "anchor_proofs", "reconcile_jobs", "reconcile_job_items", "audit_logs"}
	for _, table := range tables {
		var count int
		err := gdb.WithContext(ctx).Raw(
			`SELECT COUNT(1)
			 FROM information_schema.tables
			 WHERE table_schema = ? AND table_name = ?`,
			schema,
			table,
		).Scan(&count).Error
		if err != nil {
			t.Fatalf("query information_schema for table %s: %v", table, err)
		}
		if count != 1 {
			t.Fatalf("table %s not found in schema %s", table, schema)
		}
	}
}

func getEnv(key string) string {
	return strings.TrimSpace(strings.Trim(os.Getenv(key), "\""))
}
