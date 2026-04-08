package db

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/lychee-ripe/gateway/internal/config"
	repositorygorm "github.com/lychee-ripe/gateway/internal/repository/gorm"
	gormpostgres "gorm.io/driver/postgres"
	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

func OpenGORM(ctx context.Context, cfg config.DBConfig) (*gorm.DB, error) {
	driver := strings.ToLower(strings.TrimSpace(cfg.Driver))
	var dialector gorm.Dialector
	switch driver {
	case "sqlite":
		dsn, err := buildSQLiteDSN(cfg)
		if err != nil {
			return nil, err
		}
		dialector = gormsqlite.New(gormsqlite.Config{
			DriverName: "sqlite",
			DSN:        dsn,
		})
	case "postgres":
		dialector = gormpostgres.Open(buildPostgresDSN(cfg))
	default:
		return nil, fmt.Errorf("unsupported db driver: %q", cfg.Driver)
	}

	gdb, err := gorm.Open(dialector, &gorm.Config{
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, fmt.Errorf("open gorm %s: %w", driver, err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db handle: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	if cfg.ConnMaxLifetimeS > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeS) * time.Second)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping gorm db: %w", err)
	}
	return gdb, nil
}

func AutoMigrate(ctx context.Context, gdb *gorm.DB) error {
	if err := gdb.WithContext(ctx).AutoMigrate(
		&repositorygorm.BatchModel{},
		&repositorygorm.AnchorProofModel{},
		&repositorygorm.ReconcileJobModel{},
		&repositorygorm.ReconcileJobItemModel{},
		&repositorygorm.AuditLogModel{},
		&repositorygorm.UserModel{},
		&repositorygorm.OrchardModel{},
		&repositorygorm.PlotModel{},
	); err != nil {
		return fmt.Errorf("auto migrate models: %w", err)
	}
	return nil
}

func SanitizeDSN(driver, dsn string) string {
	if dsn == "" {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "postgres":
		return sanitizePostgresDSN(dsn)
	case "sqlite":
		return sanitizeSQLiteDSN(dsn)
	default:
		return dsn
	}
}

func buildSQLiteDSN(cfg config.DBConfig) (string, error) {
	raw := strings.TrimSpace(cfg.DSN)
	if raw == "" {
		return "", fmt.Errorf("sqlite dsn is required")
	}

	if raw != ":memory:" && !strings.HasPrefix(raw, "file:") {
		if err := ensureDir(filepath.Dir(raw)); err != nil {
			return "", fmt.Errorf("create sqlite db dir: %w", err)
		}
		raw = "file:" + filepath.ToSlash(raw)
	}

	sep := "?"
	if strings.Contains(raw, "?") {
		sep = "&"
	}
	journalMode := strings.ToUpper(strings.TrimSpace(cfg.SQLite.JournalMode))
	if journalMode == "" {
		journalMode = "WAL"
	}
	if cfg.SQLite.BusyTimeoutMS <= 0 {
		cfg.SQLite.BusyTimeoutMS = 5000
	}

	return fmt.Sprintf(
		"%s%s_foreign_keys=on&_busy_timeout=%d&_journal_mode=%s",
		raw,
		sep,
		cfg.SQLite.BusyTimeoutMS,
		url.QueryEscape(journalMode),
	), nil
}

func buildPostgresDSN(cfg config.DBConfig) string {
	dsn := strings.TrimSpace(cfg.DSN)
	if dsn == "" {
		return dsn
	}

	schema := strings.TrimSpace(cfg.Postgres.Schema)
	if schema == "" {
		schema = "public"
	}
	sslMode := strings.TrimSpace(cfg.Postgres.SSLMode)
	if sslMode == "" {
		sslMode = "disable"
	}

	if strings.Contains(dsn, "://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return dsn
		}
		q := u.Query()
		if q.Get("sslmode") == "" {
			q.Set("sslmode", sslMode)
		}
		if q.Get("search_path") == "" {
			q.Set("search_path", schema)
		}
		u.RawQuery = q.Encode()
		return u.String()
	}

	if !strings.Contains(strings.ToLower(dsn), "sslmode=") {
		dsn += " sslmode=" + sslMode
	}
	if !strings.Contains(strings.ToLower(dsn), "search_path=") {
		dsn += " search_path=" + schema
	}
	return dsn
}

func sanitizePostgresDSN(dsn string) string {
	if strings.Contains(dsn, "://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return "<invalid postgres dsn>"
		}
		if u.User != nil {
			user := u.User.Username()
			if _, ok := u.User.Password(); ok {
				u.User = url.UserPassword(user, "******")
			}
		}
		return u.String()
	}

	re := regexp.MustCompile(`(?i)(password\s*=\s*)([^ ]+)`)
	return re.ReplaceAllString(dsn, `${1}******`)
}

func sanitizeSQLiteDSN(dsn string) string {
	if idx := strings.Index(dsn, "?"); idx >= 0 {
		return dsn[:idx]
	}
	return dsn
}

func ensureDir(path string) error {
	if path == "" || path == "." {
		return nil
	}
	return os.MkdirAll(path, 0o755)
}
