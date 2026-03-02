package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/lychee-ripe/gateway/internal/config"
	gatewaydb "github.com/lychee-ripe/gateway/internal/db"
	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/repository"
)

func TestCreateBatchConflictOnUniqueKeys(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo, conn := mustNewRepo(t, filepath.Join(t.TempDir(), "gateway.db"))
	defer conn.Close()

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

func TestBatchCRUDAndStatusFlow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo, conn := mustNewRepo(t, filepath.Join(t.TempDir(), "gateway.db"))
	defer conn.Close()

	create := sampleCreateBatchParams("batch-2", "trace-2")
	create.Status = domain.BatchStatusPendingAnchor
	batch, err := repo.CreateBatch(ctx, create)
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}
	if batch.BatchID != create.BatchID {
		t.Fatalf("batch_id = %q, want %q", batch.BatchID, create.BatchID)
	}

	fetched, err := repo.GetBatchByID(ctx, create.BatchID)
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

	updated, err := repo.GetBatchByID(ctx, create.BatchID)
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

	anchored, err := repo.GetBatchByID(ctx, create.BatchID)
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

func TestListPendingBatchesAndPersistenceAfterRestart(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "gateway.db")

	repo, conn := mustNewRepo(t, dbPath)
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

	if err := conn.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	reopenedRepo, reopenedConn := mustNewRepo(t, dbPath)
	defer reopenedConn.Close()
	reopened, err := reopenedRepo.GetBatchByID(ctx, "batch-3")
	if err != nil {
		t.Fatalf("get batch after reopen: %v", err)
	}
	if reopened.BatchID != "batch-3" {
		t.Fatalf("batch_id after reopen = %q, want batch-3", reopened.BatchID)
	}
}

func TestDashboardAggregations(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo, conn := mustNewRepo(t, filepath.Join(t.TempDir(), "gateway.db"))
	defer conn.Close()

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

	count, err := repo.CountBatches(ctx)
	if err != nil {
		t.Fatalf("count batches: %v", err)
	}
	if count != 3 {
		t.Fatalf("batch count = %d, want 3", count)
	}

	status, err := repo.CountByStatus(ctx)
	if err != nil {
		t.Fatalf("count by status: %v", err)
	}
	if status.Anchored != 1 || status.PendingAnchor != 1 || status.AnchorFailed != 1 {
		t.Fatalf("status distribution = %+v, want 1/1/1", status)
	}

	ripeness, err := repo.SumRipeness(ctx)
	if err != nil {
		t.Fatalf("sum ripeness: %v", err)
	}
	if ripeness.Green != 5 || ripeness.Half != 6 || ripeness.Red != 8 || ripeness.Young != 1 {
		t.Fatalf("ripeness distribution = %+v, want green=5 half=6 red=8 young=1", ripeness)
	}

	unripeCount, unripeRatio, err := repo.CountUnripeBatches(ctx, 0.15)
	if err != nil {
		t.Fatalf("count unripe: %v", err)
	}
	if unripeCount != 2 {
		t.Fatalf("unripe count = %d, want 2", unripeCount)
	}
	if unripeRatio <= 0.6 || unripeRatio >= 0.7 {
		t.Fatalf("unripe ratio = %f, want around 0.666", unripeRatio)
	}

	recent, err := repo.ListRecentAnchors(ctx, 2)
	if err != nil {
		t.Fatalf("list recent anchors: %v", err)
	}
	if len(recent) != 2 {
		t.Fatalf("recent anchors len = %d, want 2", len(recent))
	}
}

func TestReconcileAndAudit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo, conn := mustNewRepo(t, filepath.Join(t.TempDir(), "gateway.db"))
	defer conn.Close()

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

	var auditCount int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(1) FROM audit_logs WHERE event_type = ?", "batch_created").Scan(&auditCount); err != nil {
		t.Fatalf("query audit_logs: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("audit_logs count = %d, want 1", auditCount)
	}
}

func mustNewRepo(t *testing.T, path string) (*Repository, *sql.DB) {
	t.Helper()
	ctx := context.Background()
	conn, err := gatewaydb.Open(ctx, config.DBConfig{
		Path:          path,
		BusyTimeoutMS: 5000,
		JournalMode:   "WAL",
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := gatewaydb.Migrate(ctx, conn); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	return New(conn), conn
}

func sampleCreateBatchParams(batchID, traceCode string) domain.CreateBatchParams {
	now := time.Date(2026, 3, 2, 10, 30, 0, 0, time.UTC)
	plotName := "plot-a1"
	note := "note"
	return domain.CreateBatchParams{
		BatchID:     batchID,
		TraceCode:   traceCode,
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
