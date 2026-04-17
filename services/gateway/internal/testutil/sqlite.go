package testutil

import (
	"path/filepath"
	"testing"

	"github.com/lychee-ripe/gateway/internal/config"
)

func TempSQLiteDBConfig(t *testing.T, filename string) config.DBConfig {
	t.Helper()
	return SQLiteDBConfig(filepath.Join(t.TempDir(), filename))
}

func SQLiteDBConfig(dsn string) config.DBConfig {
	return config.DBConfig{
		Driver:           "sqlite",
		DSN:              dsn,
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetimeS: 300,
		SQLite: config.SQLiteDBConfig{
			JournalMode:   "WAL",
			BusyTimeoutMS: 5000,
		},
	}
}
