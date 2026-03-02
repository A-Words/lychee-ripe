package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/lychee-ripe/gateway/internal/config"
)

func Open(ctx context.Context, cfg config.DBConfig) (*sql.DB, error) {
	if err := ensureParentDir(cfg.Path); err != nil {
		return nil, err
	}

	conn, err := sql.Open("sqlite", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)

	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = ON;"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if cfg.BusyTimeoutMS > 0 {
		stmt := fmt.Sprintf("PRAGMA busy_timeout = %d;", cfg.BusyTimeoutMS)
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("set busy timeout: %w", err)
		}
	}

	journalMode := strings.ToUpper(strings.TrimSpace(cfg.JournalMode))
	if journalMode == "" {
		journalMode = "WAL"
	}
	stmt := fmt.Sprintf("PRAGMA journal_mode = %s;", journalMode)
	if _, err := conn.ExecContext(ctx, stmt); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("set journal mode: %w", err)
	}

	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	return conn, nil
}

func ensureParentDir(path string) error {
	if path == "" || path == ":memory:" || strings.HasPrefix(path, "file:") {
		return nil
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create sqlite parent dir %q: %w", dir, err)
	}
	return nil
}
