package gorm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lychee-ripe/gateway/internal/config"
	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/repository"
	gormpostgres "gorm.io/driver/postgres"
	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

func TestCreateBatchConflictOnUniqueKeysSQLite(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo, sqlDB := mustNewSQLiteRepo(t)
	defer sqlDB.Close()

	base := sampleCreateBatchParams("batch-1", "trace-1")
	if _, err := repo.CreateBatch(ctx, base); err != nil {
		t.Fatalf("create batch first time: %v", err)
	}

	dupBatchID := sampleCreateBatchParams("batch-1", "trace-2")
	if _, err := repo.CreateBatch(ctx, dupBatchID); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("duplicate batch_id error = %v, want ErrConflict", err)
	}

	dupTraceCode := sampleCreateBatchParams("batch-2", "trace-1")
	if _, err := repo.CreateBatch(ctx, dupTraceCode); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("duplicate trace_code error = %v, want ErrConflict", err)
	}
}

func TestBatchCRUDAndStatusFlowSQLite(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo, sqlDB := mustNewSQLiteRepo(t)
	defer sqlDB.Close()

	create := sampleCreateBatchParams("batch-2", "trace-2")
	create.Status = domain.BatchStatusPendingAnchor
	batch, err := repo.CreateBatch(ctx, create)
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}
	if batch.BatchID != create.BatchID {
		t.Fatalf("batch_id = %q, want %q", batch.BatchID, create.BatchID)
	}

	fetched, err := repo.GetBatchByID(ctx, create.BatchID, create.TraceMode)
	if err != nil {
		t.Fatalf("get batch by id: %v", err)
	}
	if fetched.TraceCode != create.TraceCode {
		t.Fatalf("trace_code = %q, want %q", fetched.TraceCode, create.TraceCode)
	}

	lastError := "chain unavailable"
	retryCount := 2
	updatedAt := time.Now().UTC().Add(2 * time.Minute)
	if err := repo.UpdateBatchStatus(ctx, create.BatchID, domain.BatchStatusAnchorFailed, &lastError, &retryCount, updatedAt); err != nil {
		t.Fatalf("update batch status: %v", err)
	}

	updated, err := repo.GetBatchByID(ctx, create.BatchID, create.TraceMode)
	if err != nil {
		t.Fatalf("get updated batch: %v", err)
	}
	if updated.Status != domain.BatchStatusAnchorFailed {
		t.Fatalf("status = %q, want %q", updated.Status, domain.BatchStatusAnchorFailed)
	}
	if updated.RetryCount != retryCount {
		t.Fatalf("retry_count = %d, want %d", updated.RetryCount, retryCount)
	}
	if updated.LastError == nil || *updated.LastError != lastError {
		t.Fatalf("last_error = %v, want %q", updated.LastError, lastError)
	}

	proof := domain.AnchorProofRecord{
		TxHash:          "0xabc",
		BlockNumber:     100,
		ChainID:         "31337",
		ContractAddress: "0xdef",
		AnchorHash:      "0xhash",
		AnchoredAt:      time.Now().UTC(),
	}
	if err := repo.AttachAnchorProof(ctx, create.BatchID, proof, time.Now().UTC()); err != nil {
		t.Fatalf("attach anchor proof: %v", err)
	}

	anchored, err := repo.GetBatchByID(ctx, create.BatchID, create.TraceMode)
	if err != nil {
		t.Fatalf("get anchored batch: %v", err)
	}
	if anchored.Status != domain.BatchStatusAnchored {
		t.Fatalf("status = %q, want %q", anchored.Status, domain.BatchStatusAnchored)
	}
	if anchored.AnchorProof == nil || anchored.AnchorProof.TxHash != proof.TxHash {
		t.Fatalf("anchor_proof = %+v, want tx_hash %q", anchored.AnchorProof, proof.TxHash)
	}
}

func TestListPendingBatchesAndPersistenceAfterRestartSQLite(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "gateway.db")

	repo, sqlDB := mustNewRepoWithConfig(t, config.DBConfig{
		Driver:           "sqlite",
		DSN:              dbPath,
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetimeS: 300,
		SQLite:           config.SQLiteDBConfig{JournalMode: "WAL", BusyTimeoutMS: 5000},
	})
	_, err := repo.CreateBatch(ctx, sampleCreateBatchParams("batch-3", "trace-3"))
	if err != nil {
		t.Fatalf("create batch-3: %v", err)
	}
	nonPending := sampleCreateBatchParams("batch-4", "trace-4")
	nonPending.Status = domain.BatchStatusAnchored
	_, err = repo.CreateBatch(ctx, nonPending)
	if err != nil {
		t.Fatalf("create batch-4: %v", err)
	}

	pending, err := repo.ListPendingBatches(ctx, 10)
	if err != nil {
		t.Fatalf("list pending batches: %v", err)
	}
	if len(pending) != 1 || pending[0].BatchID != "batch-3" {
		t.Fatalf("pending batches = %+v, want only batch-3", pending)
	}

	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	reopenedRepo, reopenedSQLDB := mustNewRepoWithConfig(t, config.DBConfig{
		Driver:           "sqlite",
		DSN:              dbPath,
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetimeS: 300,
		SQLite:           config.SQLiteDBConfig{JournalMode: "WAL", BusyTimeoutMS: 5000},
	})
	defer reopenedSQLDB.Close()
	reopened, err := reopenedRepo.GetBatchByID(ctx, "batch-3", domain.TraceModeBlockchain)
	if err != nil {
		t.Fatalf("get batch after reopen: %v", err)
	}
	if reopened.BatchID != "batch-3" {
		t.Fatalf("batch_id after reopen = %q, want batch-3", reopened.BatchID)
	}
}

func TestDashboardAggregationsSQLite(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo, sqlDB := mustNewSQLiteRepo(t)
	defer sqlDB.Close()

	first := sampleCreateBatchParams("batch-5", "trace-5")
	first.Status = domain.BatchStatusPendingAnchor
	first.Summary.Green = 4
	first.Summary.Half = 1
	first.Summary.Red = 1
	first.Summary.Young = 0
	first.Summary.Total = 6
	first.Summary.UnripeCount = 4
	first.Summary.UnripeRatio = 0.66
	if _, err := repo.CreateBatch(ctx, first); err != nil {
		t.Fatalf("create first batch: %v", err)
	}

	second := sampleCreateBatchParams("batch-6", "trace-6")
	second.Status = domain.BatchStatusAnchored
	second.Summary.Green = 1
	second.Summary.Half = 3
	second.Summary.Red = 4
	second.Summary.Young = 0
	second.Summary.Total = 8
	second.Summary.UnripeCount = 1
	second.Summary.UnripeRatio = 0.12
	if _, err := repo.CreateBatch(ctx, second); err != nil {
		t.Fatalf("create second batch: %v", err)
	}

	third := sampleCreateBatchParams("batch-7", "trace-7")
	third.Status = domain.BatchStatusAnchorFailed
	third.Summary.Green = 0
	third.Summary.Half = 2
	third.Summary.Red = 3
	third.Summary.Young = 1
	third.Summary.Total = 6
	third.Summary.UnripeCount = 1
	third.Summary.UnripeRatio = 0.16
	if _, err := repo.CreateBatch(ctx, third); err != nil {
		t.Fatalf("create third batch: %v", err)
	}

	count, err := repo.CountBatches(ctx, domain.TraceModeBlockchain)
	if err != nil {
		t.Fatalf("count batches: %v", err)
	}
	if count != 3 {
		t.Fatalf("batch count = %d, want 3", count)
	}

	status, err := repo.CountByStatus(ctx, domain.TraceModeBlockchain)
	if err != nil {
		t.Fatalf("count by status: %v", err)
	}
	if status.Anchored != 1 || status.PendingAnchor != 1 || status.AnchorFailed != 1 {
		t.Fatalf("status distribution = %+v, want 1/1/1", status)
	}

	ripeness, err := repo.SumRipeness(ctx, domain.TraceModeBlockchain)
	if err != nil {
		t.Fatalf("sum ripeness: %v", err)
	}
	if ripeness.Green != 5 || ripeness.Half != 6 || ripeness.Red != 8 || ripeness.Young != 1 {
		t.Fatalf("ripeness distribution = %+v, want green=5 half=6 red=8 young=1", ripeness)
	}

	unripeCount, unripeRatio, err := repo.CountUnripeBatches(ctx, domain.TraceModeBlockchain, 0.15)
	if err != nil {
		t.Fatalf("count unripe: %v", err)
	}
	if unripeCount != 2 {
		t.Fatalf("unripe count = %d, want 2", unripeCount)
	}
	if unripeRatio <= 0.6 || unripeRatio >= 0.7 {
		t.Fatalf("unripe ratio = %f, want around 0.666", unripeRatio)
	}

	recent, err := repo.ListRecentAnchors(ctx, domain.TraceModeBlockchain, 2)
	if err != nil {
		t.Fatalf("list recent anchors: %v", err)
	}
	if len(recent) != 1 {
		t.Fatalf("recent anchors len = %d, want 1", len(recent))
	}
	for _, item := range recent {
		if item.Status != domain.BatchStatusAnchored {
			t.Fatalf("recent anchor status = %q, want anchored", item.Status)
		}
	}
}

func TestReconcileAndAuditSQLite(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo, sqlDB := mustNewSQLiteRepo(t)
	defer sqlDB.Close()

	pending := sampleCreateBatchParams("batch-8", "trace-8")
	pending.Status = domain.BatchStatusPendingAnchor
	pending.RetryCount = 3
	if _, err := repo.CreateBatch(ctx, pending); err != nil {
		t.Fatalf("create pending batch: %v", err)
	}

	job, err := repo.CreateReconcileJob(ctx, domain.CreateReconcileJobParams{
		JobID:          "job-1",
		TriggerType:    domain.ReconcileTriggerManual,
		Status:         domain.ReconcileJobStatusAccepted,
		RequestedCount: 1,
		ScheduledCount: 1,
		SkippedCount:   0,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create reconcile job: %v", err)
	}

	if err := repo.AddReconcileJobItems(ctx, job.JobID, []domain.ReconcileJobItemRecord{
		{
			BatchID:      "batch-8",
			BeforeStatus: domain.BatchStatusPendingAnchor,
			AfterStatus:  domain.BatchStatusAnchored,
			AttemptNo:    1,
			CreatedAt:    time.Now().UTC(),
		},
	}); err != nil {
		t.Fatalf("add reconcile items: %v", err)
	}

	if err := repo.UpdateReconcileJobStatus(ctx, job.JobID, domain.ReconcileJobStatusCompleted, nil, time.Now().UTC()); err != nil {
		t.Fatalf("update reconcile job status: %v", err)
	}

	stats, err := repo.ListReconcileStats(ctx)
	if err != nil {
		t.Fatalf("list reconcile stats: %v", err)
	}
	if stats.PendingCount != 1 {
		t.Fatalf("pending_count = %d, want 1", stats.PendingCount)
	}
	if stats.RetriedTotal != 3 {
		t.Fatalf("retried_total = %d, want 3", stats.RetriedTotal)
	}

	message := "batch created"
	payload := `{"k":"v"}`
	if err := repo.AppendAuditLog(ctx, domain.AuditLogRecord{
		EventType:   "batch_created",
		EntityType:  "batch",
		EntityID:    "batch-8",
		Message:     &message,
		PayloadJSON: &payload,
		CreatedAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatalf("append audit log: %v", err)
	}

	var auditCount int64
	if err := repo.db.WithContext(ctx).Model(&AuditLogModel{}).Where("event_type = ?", "batch_created").Count(&auditCount).Error; err != nil {
		t.Fatalf("query audit_logs: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("audit_logs count = %d, want 1", auditCount)
	}
}

func TestPostgresOptionalBasicFlow(t *testing.T) {
	pgDSN := strings.TrimSpace(os.Getenv("LYCHEE_GATEWAY_TEST_PG_DSN"))
	if pgDSN == "" {
		t.Skip("LYCHEE_GATEWAY_TEST_PG_DSN not set, skip postgres integration test")
	}

	ctx := context.Background()
	cfg := config.DBConfig{
		Driver:           "postgres",
		DSN:              pgDSN,
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetimeS: 300,
		Postgres: config.PostgresDBConfig{
			SSLMode: "disable",
			Schema:  "public",
		},
	}

	repo, sqlDB := mustNewRepoWithConfig(t, cfg)
	defer sqlDB.Close()

	_ = repo.db.WithContext(ctx).Exec("DELETE FROM audit_logs")
	_ = repo.db.WithContext(ctx).Exec("DELETE FROM reconcile_job_items")
	_ = repo.db.WithContext(ctx).Exec("DELETE FROM reconcile_jobs")
	_ = repo.db.WithContext(ctx).Exec("DELETE FROM anchor_proofs")
	_ = repo.db.WithContext(ctx).Exec("DELETE FROM batches")

	first := sampleCreateBatchParams("pg-batch-1", "pg-trace-1")
	if _, err := repo.CreateBatch(ctx, first); err != nil {
		t.Fatalf("create postgres batch: %v", err)
	}
	if _, err := repo.CreateBatch(ctx, sampleCreateBatchParams("pg-batch-1", "pg-trace-2")); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("duplicate postgres batch should return ErrConflict, got %v", err)
	}
	loaded, err := repo.GetBatchByID(ctx, "pg-batch-1", domain.TraceModeBlockchain)
	if err != nil {
		t.Fatalf("get postgres batch: %v", err)
	}
	if loaded.TraceCode != "pg-trace-1" {
		t.Fatalf("trace_code = %q, want pg-trace-1", loaded.TraceCode)
	}
}

func TestModeScopedReadsAndDashboardAggregationsSQLite(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo, sqlDB := mustNewSQLiteRepo(t)
	defer sqlDB.Close()

	databaseBatch := sampleCreateBatchParams("batch-db-1", "trace-db-1")
	databaseBatch.TraceMode = domain.TraceModeDatabase
	databaseBatch.Status = domain.BatchStatusStored
	databaseBatch.Summary.Green = 1
	databaseBatch.Summary.Half = 1
	databaseBatch.Summary.Red = 3
	databaseBatch.Summary.Young = 1
	databaseBatch.Summary.Total = 6
	databaseBatch.Summary.UnripeCount = 2
	databaseBatch.Summary.UnripeRatio = 0.33
	if _, err := repo.CreateBatch(ctx, databaseBatch); err != nil {
		t.Fatalf("create database batch: %v", err)
	}

	blockchainBatch := sampleCreateBatchParams("batch-bc-1", "trace-bc-1")
	blockchainBatch.TraceMode = domain.TraceModeBlockchain
	blockchainBatch.Status = domain.BatchStatusAnchored
	blockchainBatch.Summary.Green = 0
	blockchainBatch.Summary.Half = 4
	blockchainBatch.Summary.Red = 5
	blockchainBatch.Summary.Young = 1
	blockchainBatch.Summary.Total = 10
	blockchainBatch.Summary.UnripeCount = 1
	blockchainBatch.Summary.UnripeRatio = 0.1
	if _, err := repo.CreateBatch(ctx, blockchainBatch); err != nil {
		t.Fatalf("create blockchain batch: %v", err)
	}
	if err := repo.AttachAnchorProof(ctx, blockchainBatch.BatchID, domain.AnchorProofRecord{
		TxHash:          "0xtest",
		BlockNumber:     100,
		ChainID:         "31337",
		ContractAddress: "0xdef",
		AnchorHash:      "0xhash",
		AnchoredAt:      time.Now().UTC(),
	}, time.Now().UTC()); err != nil {
		t.Fatalf("attach blockchain proof: %v", err)
	}

	if _, err := repo.GetBatchByID(ctx, databaseBatch.BatchID, domain.TraceModeBlockchain); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("cross-mode get by id error = %v, want ErrNotFound", err)
	}
	if _, err := repo.GetBatchByTraceCode(ctx, blockchainBatch.TraceCode, domain.TraceModeDatabase); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("cross-mode get by trace error = %v, want ErrNotFound", err)
	}

	databaseCount, err := repo.CountBatches(ctx, domain.TraceModeDatabase)
	if err != nil {
		t.Fatalf("count database batches: %v", err)
	}
	if databaseCount != 1 {
		t.Fatalf("database batch count = %d, want 1", databaseCount)
	}

	blockchainCount, err := repo.CountBatches(ctx, domain.TraceModeBlockchain)
	if err != nil {
		t.Fatalf("count blockchain batches: %v", err)
	}
	if blockchainCount != 1 {
		t.Fatalf("blockchain batch count = %d, want 1", blockchainCount)
	}

	databaseStatus, err := repo.CountByStatus(ctx, domain.TraceModeDatabase)
	if err != nil {
		t.Fatalf("count database status: %v", err)
	}
	if databaseStatus.Stored != 1 || databaseStatus.Anchored != 0 {
		t.Fatalf("database status distribution = %+v, want stored only", databaseStatus)
	}

	blockchainStatus, err := repo.CountByStatus(ctx, domain.TraceModeBlockchain)
	if err != nil {
		t.Fatalf("count blockchain status: %v", err)
	}
	if blockchainStatus.Anchored != 1 || blockchainStatus.Stored != 0 {
		t.Fatalf("blockchain status distribution = %+v, want anchored only", blockchainStatus)
	}

	databaseRipeness, err := repo.SumRipeness(ctx, domain.TraceModeDatabase)
	if err != nil {
		t.Fatalf("sum database ripeness: %v", err)
	}
	if databaseRipeness.Green != 1 || databaseRipeness.Red != 3 {
		t.Fatalf("database ripeness = %+v, want green=1 red=3", databaseRipeness)
	}

	blockchainRipeness, err := repo.SumRipeness(ctx, domain.TraceModeBlockchain)
	if err != nil {
		t.Fatalf("sum blockchain ripeness: %v", err)
	}
	if blockchainRipeness.Half != 4 || blockchainRipeness.Red != 5 {
		t.Fatalf("blockchain ripeness = %+v, want half=4 red=5", blockchainRipeness)
	}

	databaseUnripeCount, _, err := repo.CountUnripeBatches(ctx, domain.TraceModeDatabase, 0.15)
	if err != nil {
		t.Fatalf("count database unripe: %v", err)
	}
	if databaseUnripeCount != 1 {
		t.Fatalf("database unripe count = %d, want 1", databaseUnripeCount)
	}

	blockchainUnripeCount, _, err := repo.CountUnripeBatches(ctx, domain.TraceModeBlockchain, 0.15)
	if err != nil {
		t.Fatalf("count blockchain unripe: %v", err)
	}
	if blockchainUnripeCount != 0 {
		t.Fatalf("blockchain unripe count = %d, want 0", blockchainUnripeCount)
	}

	databaseRecent, err := repo.ListRecentAnchors(ctx, domain.TraceModeDatabase, 5)
	if err != nil {
		t.Fatalf("list database recent anchors: %v", err)
	}
	if len(databaseRecent) != 0 {
		t.Fatalf("database recent anchors = %+v, want empty", databaseRecent)
	}

	blockchainRecent, err := repo.ListRecentAnchors(ctx, domain.TraceModeBlockchain, 5)
	if err != nil {
		t.Fatalf("list blockchain recent anchors: %v", err)
	}
	if len(blockchainRecent) != 1 || blockchainRecent[0].BatchID != blockchainBatch.BatchID {
		t.Fatalf("blockchain recent anchors = %+v, want only %s", blockchainRecent, blockchainBatch.BatchID)
	}
}

func TestResolvePrincipalBindsActivePreProvisionedUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo, sqlDB := mustNewSQLiteRepo(t)
	defer sqlDB.Close()

	now := time.Now().UTC()
	user := UserModel{
		ID:          "user-1",
		Email:       "operator@example.com",
		DisplayName: "Provisioned Operator",
		Role:        string(domain.UserRoleOperator),
		Status:      string(domain.UserStatusActive),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repo.db.WithContext(ctx).Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	claims := domain.IdentityClaims{
		Subject:     "oidc-sub-1",
		Email:       "operator@example.com",
		DisplayName: "OIDC Operator",
	}
	principal, err := repo.ResolvePrincipal(ctx, claims, domain.AuthModeOIDC, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("ResolvePrincipal returned error: %v", err)
	}
	if principal.Subject != "oidc-sub-1" {
		t.Fatalf("principal subject = %q, want oidc-sub-1", principal.Subject)
	}

	var stored UserModel
	if err := repo.db.WithContext(ctx).Where("id = ?", user.ID).First(&stored).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if stored.OIDCSubject == nil || *stored.OIDCSubject != "oidc-sub-1" {
		t.Fatalf("oidc_subject = %v, want oidc-sub-1", stored.OIDCSubject)
	}
	if stored.DisplayName != "OIDC Operator" {
		t.Fatalf("display_name = %q, want OIDC Operator", stored.DisplayName)
	}
	if stored.LastLoginAt == nil {
		t.Fatal("expected last_login_at to be set")
	}
}

func TestResolvePrincipalDoesNotBindDisabledPreProvisionedUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo, sqlDB := mustNewSQLiteRepo(t)
	defer sqlDB.Close()

	now := time.Now().UTC()
	user := UserModel{
		ID:          "user-2",
		Email:       "disabled@example.com",
		DisplayName: "Disabled User",
		Role:        string(domain.UserRoleOperator),
		Status:      string(domain.UserStatusDisabled),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repo.db.WithContext(ctx).Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	claims := domain.IdentityClaims{
		Subject:     "oidc-disabled-sub",
		Email:       "disabled@example.com",
		DisplayName: "Attempted Login",
	}
	if _, err := repo.ResolvePrincipal(ctx, claims, domain.AuthModeOIDC, now.Add(time.Minute)); !errors.Is(err, repository.ErrInvalidState) {
		t.Fatalf("ResolvePrincipal error = %v, want ErrInvalidState", err)
	}

	var stored UserModel
	if err := repo.db.WithContext(ctx).Where("id = ?", user.ID).First(&stored).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if stored.OIDCSubject != nil {
		t.Fatalf("oidc_subject = %v, want nil", stored.OIDCSubject)
	}
	if stored.DisplayName != "Disabled User" {
		t.Fatalf("display_name = %q, want Disabled User", stored.DisplayName)
	}
	if stored.LastLoginAt != nil {
		t.Fatalf("last_login_at = %v, want nil", stored.LastLoginAt)
	}
}

func mustNewSQLiteRepo(t *testing.T) (*Repository, *sql.DB) {
	t.Helper()
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
	return mustNewRepoWithConfig(t, cfg)
}

func mustNewRepoWithConfig(t *testing.T, cfg config.DBConfig) (*Repository, *sql.DB) {
	t.Helper()
	ctx := context.Background()
	gdb, err := openGormForTest(cfg)
	if err != nil {
		t.Fatalf("open gorm db: %v", err)
	}
	if err := gdb.WithContext(ctx).AutoMigrate(
		&BatchModel{},
		&AnchorProofModel{},
		&ReconcileJobModel{},
		&ReconcileJobItemModel{},
		&AuditLogModel{},
		&UserModel{},
		&OrchardModel{},
		&PlotModel{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	return New(gdb), sqlDB
}

func openGormForTest(cfg config.DBConfig) (*gorm.DB, error) {
	driver := strings.ToLower(strings.TrimSpace(cfg.Driver))
	switch driver {
	case "sqlite":
		dsn := strings.TrimSpace(cfg.DSN)
		if dsn == "" {
			return nil, errors.New("sqlite dsn required")
		}
		if dsn != ":memory:" && !strings.HasPrefix(dsn, "file:") {
			if err := os.MkdirAll(filepath.Dir(dsn), 0o755); err != nil {
				return nil, err
			}
			dsn = "file:" + filepath.ToSlash(dsn)
		}
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		dsn = fmt.Sprintf("%s%s_foreign_keys=on", dsn, sep)
		return gorm.Open(gormsqlite.New(gormsqlite.Config{
			DriverName: "sqlite",
			DSN:        dsn,
		}), &gorm.Config{
			NowFunc: func() time.Time { return time.Now().UTC() },
		})
	case "postgres":
		return gorm.Open(gormpostgres.Open(strings.TrimSpace(cfg.DSN)), &gorm.Config{
			NowFunc: func() time.Time { return time.Now().UTC() },
		})
	default:
		return nil, fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
}

func sampleCreateBatchParams(batchID, traceCode string) domain.CreateBatchParams {
	now := time.Date(2026, 3, 2, 10, 30, 0, 0, time.UTC)
	plotName := "plot-a1"
	note := "note"
	return domain.CreateBatchParams{
		BatchID:     batchID,
		TraceCode:   traceCode,
		TraceMode:   domain.TraceModeBlockchain,
		Status:      domain.BatchStatusPendingAnchor,
		OrchardID:   "orchard-1",
		OrchardName: "orchard",
		PlotID:      "plot-1",
		PlotName:    &plotName,
		HarvestedAt: now,
		Summary: domain.BatchSummary{
			Total:          10,
			Green:          2,
			Half:           3,
			Red:            4,
			Young:          1,
			UnripeCount:    3,
			UnripeRatio:    0.3,
			UnripeHandling: domain.UnripeHandlingSortedOut,
		},
		Note:          &note,
		ConfirmUnripe: true,
		RetryCount:    0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}
