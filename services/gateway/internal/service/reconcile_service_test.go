package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/lychee-ripe/gateway/internal/chain/evm"
	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/repository"
)

func TestTriggerManualReconcileSuccess(t *testing.T) {
	t.Parallel()

	batchRepo := newFakeReconcileBatchRepo()
	rec := samplePendingBatch("batch_r1", 0)
	batchRepo.batches[rec.BatchID] = rec
	reconcileRepo := newFakeReconcileRepo()
	anchor := &fakeReconcileAnchorClient{
		proofs: map[string]domain.AnchorProofRecord{
			rec.BatchID: sampleAnchorProof("0xtx1", *rec.AnchorHash),
		},
	}

	svc := NewReconcileService(batchRepo, reconcileRepo, anchor, domain.TraceModeBlockchain, nil)
	result, err := svc.TriggerManualReconcile(context.Background(), ManualReconcileInput{
		BatchIDs: []string{rec.BatchID},
	})
	if err != nil {
		t.Fatalf("TriggerManualReconcile failed: %v", err)
	}
	if !result.Accepted || result.RequestedCount != 1 || result.ScheduledCount != 1 || result.SkippedCount != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}

	updated := batchRepo.batches[rec.BatchID]
	if updated.Status != domain.BatchStatusAnchored {
		t.Fatalf("status = %q, want anchored", updated.Status)
	}
	if updated.AnchorProof == nil {
		t.Fatal("anchor_proof should not be nil")
	}
	if len(reconcileRepo.jobs) != 1 {
		t.Fatalf("jobs = %d, want 1", len(reconcileRepo.jobs))
	}
	job := reconcileRepo.latestJob()
	if job.TriggerType != domain.ReconcileTriggerManual {
		t.Fatalf("trigger_type = %q, want manual", job.TriggerType)
	}
	if job.Status != domain.ReconcileJobStatusCompleted {
		t.Fatalf("job status = %q, want completed", job.Status)
	}
	items := reconcileRepo.items[job.JobID]
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0].AfterStatus != domain.BatchStatusAnchored {
		t.Fatalf("after_status = %q, want anchored", items[0].AfterStatus)
	}
}

func TestTriggerManualReconcileFailureBelowThreshold(t *testing.T) {
	t.Parallel()

	batchRepo := newFakeReconcileBatchRepo()
	rec := samplePendingBatch("batch_r2", 1)
	batchRepo.batches[rec.BatchID] = rec
	reconcileRepo := newFakeReconcileRepo()
	anchor := &fakeReconcileAnchorClient{
		errs: map[string]error{
			rec.BatchID: fmt.Errorf("%w: timeout", evm.ErrNodeUnavailable),
		},
	}
	svc := NewReconcileService(batchRepo, reconcileRepo, anchor, domain.TraceModeBlockchain, nil)

	result, err := svc.TriggerManualReconcile(context.Background(), ManualReconcileInput{
		BatchIDs: []string{rec.BatchID},
	})
	if err != nil {
		t.Fatalf("TriggerManualReconcile failed: %v", err)
	}
	if result.ScheduledCount != 1 {
		t.Fatalf("scheduled_count = %d, want 1", result.ScheduledCount)
	}

	updated := batchRepo.batches[rec.BatchID]
	if updated.Status != domain.BatchStatusPendingAnchor {
		t.Fatalf("status = %q, want pending_anchor", updated.Status)
	}
	if updated.RetryCount != 2 {
		t.Fatalf("retry_count = %d, want 2", updated.RetryCount)
	}
	if updated.LastError == nil {
		t.Fatal("last_error should not be nil")
	}
}

func TestTriggerManualReconcileFailureAtThreshold(t *testing.T) {
	t.Parallel()

	batchRepo := newFakeReconcileBatchRepo()
	rec := samplePendingBatch("batch_r3", 2)
	batchRepo.batches[rec.BatchID] = rec
	reconcileRepo := newFakeReconcileRepo()
	anchor := &fakeReconcileAnchorClient{
		errs: map[string]error{
			rec.BatchID: fmt.Errorf("%w: status=0", evm.ErrTxReverted),
		},
	}
	svc := NewReconcileService(batchRepo, reconcileRepo, anchor, domain.TraceModeBlockchain, nil)

	_, err := svc.TriggerManualReconcile(context.Background(), ManualReconcileInput{
		BatchIDs: []string{rec.BatchID},
	})
	if err != nil {
		t.Fatalf("TriggerManualReconcile failed: %v", err)
	}

	updated := batchRepo.batches[rec.BatchID]
	if updated.Status != domain.BatchStatusAnchorFailed {
		t.Fatalf("status = %q, want anchor_failed", updated.Status)
	}
	if updated.RetryCount != 3 {
		t.Fatalf("retry_count = %d, want 3", updated.RetryCount)
	}
}

func TestTriggerManualReconcileMixedBatchIDs(t *testing.T) {
	t.Parallel()

	batchRepo := newFakeReconcileBatchRepo()
	pending := samplePendingBatch("batch_r4_pending", 0)
	anchored := samplePendingBatch("batch_r4_anchored", 0)
	anchored.Status = domain.BatchStatusAnchored
	batchRepo.batches[pending.BatchID] = pending
	batchRepo.batches[anchored.BatchID] = anchored

	reconcileRepo := newFakeReconcileRepo()
	anchor := &fakeReconcileAnchorClient{
		proofs: map[string]domain.AnchorProofRecord{
			pending.BatchID: sampleAnchorProof("0xtx4", *pending.AnchorHash),
		},
	}
	svc := NewReconcileService(batchRepo, reconcileRepo, anchor, domain.TraceModeBlockchain, nil)

	result, err := svc.TriggerManualReconcile(context.Background(), ManualReconcileInput{
		BatchIDs: []string{pending.BatchID, "missing", anchored.BatchID},
	})
	if err != nil {
		t.Fatalf("TriggerManualReconcile failed: %v", err)
	}
	if result.RequestedCount != 3 || result.ScheduledCount != 1 || result.SkippedCount != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestTriggerManualReconcileNotFoundWhenNoSchedulableBatch(t *testing.T) {
	t.Parallel()

	svc := NewReconcileService(newFakeReconcileBatchRepo(), newFakeReconcileRepo(), &fakeReconcileAnchorClient{}, domain.TraceModeBlockchain, nil)
	_, err := svc.TriggerManualReconcile(context.Background(), ManualReconcileInput{
		BatchIDs: []string{"missing"},
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestTriggerManualReconcileReturnsConflictInDatabaseMode(t *testing.T) {
	t.Parallel()

	svc := NewReconcileService(newFakeReconcileBatchRepo(), newFakeReconcileRepo(), nil, domain.TraceModeDatabase, nil)
	_, err := svc.TriggerManualReconcile(context.Background(), ManualReconcileInput{})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("error = %v, want ErrConflict", err)
	}
}

func TestRunAutoReconcileOnce(t *testing.T) {
	t.Parallel()

	batchRepo := newFakeReconcileBatchRepo()
	rec := samplePendingBatch("batch_auto_1", 0)
	batchRepo.batches[rec.BatchID] = rec
	reconcileRepo := newFakeReconcileRepo()
	anchor := &fakeReconcileAnchorClient{
		proofs: map[string]domain.AnchorProofRecord{
			rec.BatchID: sampleAnchorProof("0xtx_auto", *rec.AnchorHash),
		},
	}
	svc := NewReconcileService(batchRepo, reconcileRepo, anchor, domain.TraceModeBlockchain, nil)

	if err := svc.RunAutoReconcileOnce(context.Background()); err != nil {
		t.Fatalf("RunAutoReconcileOnce failed: %v", err)
	}
	if len(reconcileRepo.jobs) != 1 {
		t.Fatalf("jobs = %d, want 1", len(reconcileRepo.jobs))
	}
	job := reconcileRepo.latestJob()
	if job.TriggerType != domain.ReconcileTriggerAuto {
		t.Fatalf("trigger_type = %q, want auto", job.TriggerType)
	}

	// No pending batches after first run.
	if err := svc.RunAutoReconcileOnce(context.Background()); err != nil {
		t.Fatalf("second RunAutoReconcileOnce failed: %v", err)
	}
	if len(reconcileRepo.jobs) != 1 {
		t.Fatalf("jobs = %d, want still 1", len(reconcileRepo.jobs))
	}
}

func TestTriggerManualReconcileSkipsBatchWhenClaimLost(t *testing.T) {
	t.Parallel()

	batchRepo := newFakeReconcileBatchRepo()
	rec := samplePendingBatch("batch_claim_lost", 0)
	batchRepo.batches[rec.BatchID] = rec
	batchRepo.claimErr = repository.ErrConflict
	reconcileRepo := newFakeReconcileRepo()
	anchor := &fakeReconcileAnchorClient{}
	svc := NewReconcileService(batchRepo, reconcileRepo, anchor, domain.TraceModeBlockchain, nil)

	result, err := svc.TriggerManualReconcile(context.Background(), ManualReconcileInput{
		BatchIDs: []string{rec.BatchID},
	})
	if err != nil {
		t.Fatalf("TriggerManualReconcile failed: %v", err)
	}
	if !result.Accepted || result.ScheduledCount != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(reconcileRepo.jobs) != 1 {
		t.Fatalf("jobs = %d, want 1", len(reconcileRepo.jobs))
	}
	job := reconcileRepo.latestJob()
	items := reconcileRepo.items[job.JobID]
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0].AfterStatus != domain.BatchStatusPendingAnchor {
		t.Fatalf("after_status = %q, want pending_anchor", items[0].AfterStatus)
	}
	if items[0].ErrorMessage == nil || *items[0].ErrorMessage != reconcileLostRaceReason {
		t.Fatalf("error_message = %v, want lost race reason", items[0].ErrorMessage)
	}
	if updated := batchRepo.batches[rec.BatchID]; updated.Status != domain.BatchStatusPendingAnchor {
		t.Fatalf("status = %q, want pending_anchor", updated.Status)
	}
}

func TestReconcileOneDoesNotOverwriteAnchoredBatchOnStaleFailure(t *testing.T) {
	t.Parallel()

	batchRepo := newFakeReconcileBatchRepo()
	rec := samplePendingBatch("batch_stale_failure", 1)
	batchRepo.batches[rec.BatchID] = rec
	reconcileRepo := newFakeReconcileRepo()
	anchor := &fakeReconcileAnchorClient{
		errs: map[string]error{
			rec.BatchID: fmt.Errorf("%w: timeout", evm.ErrNodeUnavailable),
		},
	}
	svc := NewReconcileService(batchRepo, reconcileRepo, anchor, domain.TraceModeBlockchain, nil)
	claimAt := rec.UpdatedAt.Add(30 * time.Second)
	if err := batchRepo.ClaimPendingBatch(context.Background(), rec.BatchID, rec.UpdatedAt, claimAt); err != nil {
		t.Fatalf("ClaimPendingBatch setup failed: %v", err)
	}
	proof := sampleAnchorProof("0xtx-stale", *rec.AnchorHash)
	if err := batchRepo.AttachAnchorProof(context.Background(), rec.BatchID, domain.BatchStatusAnchoring, claimAt, proof, claimAt.Add(30*time.Second)); err != nil {
		t.Fatalf("AttachAnchorProof setup failed: %v", err)
	}

	item, err := svc.reconcileOne(context.Background(), rec)
	if err != nil {
		t.Fatalf("reconcileOne failed: %v", err)
	}
	if item.AfterStatus != domain.BatchStatusPendingAnchor {
		t.Fatalf("after_status = %q, want pending_anchor", item.AfterStatus)
	}
	if item.ErrorMessage == nil || *item.ErrorMessage != reconcileLostRaceReason {
		t.Fatalf("error_message = %v, want lost race reason", item.ErrorMessage)
	}
	updated := batchRepo.batches[rec.BatchID]
	if updated.Status != domain.BatchStatusAnchored {
		t.Fatalf("status = %q, want anchored", updated.Status)
	}
	if updated.AnchorProof == nil || updated.AnchorProof.TxHash != proof.TxHash {
		t.Fatalf("anchor_proof = %+v, want %q", updated.AnchorProof, proof.TxHash)
	}
}

func TestTriggerManualReconcileRejectsInvalidLimit(t *testing.T) {
	t.Parallel()

	svc := NewReconcileService(newFakeReconcileBatchRepo(), newFakeReconcileRepo(), &fakeReconcileAnchorClient{}, domain.TraceModeBlockchain, nil)
	invalid := 0
	_, err := svc.TriggerManualReconcile(context.Background(), ManualReconcileInput{
		Limit: &invalid,
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v, want ErrInvalidRequest", err)
	}
}

type fakeReconcileBatchRepo struct {
	batches        map[string]domain.BatchRecord
	listPendingErr error
	getByIDErr     error
	claimErr       error
	updateErr      error
	attachErr      error
}

func newFakeReconcileBatchRepo() *fakeReconcileBatchRepo {
	return &fakeReconcileBatchRepo{
		batches: make(map[string]domain.BatchRecord),
	}
}

func (f *fakeReconcileBatchRepo) CreateBatch(_ context.Context, _ domain.CreateBatchParams) (domain.BatchRecord, error) {
	return domain.BatchRecord{}, repository.ErrDBUnavailable
}

func (f *fakeReconcileBatchRepo) GetBatchByID(_ context.Context, batchID string, traceMode domain.TraceMode) (domain.BatchRecord, error) {
	if f.getByIDErr != nil {
		return domain.BatchRecord{}, f.getByIDErr
	}
	record, ok := f.batches[batchID]
	if !ok {
		return domain.BatchRecord{}, repository.ErrNotFound
	}
	if traceMode != "" && record.TraceMode != traceMode {
		return domain.BatchRecord{}, repository.ErrNotFound
	}
	return record, nil
}

func (f *fakeReconcileBatchRepo) GetBatchByTraceCode(_ context.Context, _ string, _ domain.TraceMode) (domain.BatchRecord, error) {
	return domain.BatchRecord{}, repository.ErrNotFound
}

func (f *fakeReconcileBatchRepo) UpdateBatchStatus(
	_ context.Context,
	batchID string,
	expectedStatus domain.BatchStatus,
	expectedUpdatedAt time.Time,
	status domain.BatchStatus,
	lastError *string,
	retryCount *int,
	updatedAt time.Time,
) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	record, ok := f.batches[batchID]
	if !ok {
		return repository.ErrNotFound
	}
	if record.Status != expectedStatus || !record.UpdatedAt.Equal(expectedUpdatedAt) {
		return repository.ErrConflict
	}
	record.Status = status
	record.UpdatedAt = updatedAt
	if lastError != nil {
		msg := *lastError
		record.LastError = &msg
	}
	if retryCount != nil {
		record.RetryCount = *retryCount
	}
	f.batches[batchID] = record
	return nil
}

func (f *fakeReconcileBatchRepo) ClaimPendingBatch(
	_ context.Context,
	batchID string,
	expectedUpdatedAt time.Time,
	claimedAt time.Time,
) error {
	if f.claimErr != nil {
		return f.claimErr
	}
	record, ok := f.batches[batchID]
	if !ok {
		return repository.ErrNotFound
	}
	if record.Status != domain.BatchStatusPendingAnchor || !record.UpdatedAt.Equal(expectedUpdatedAt) {
		return repository.ErrConflict
	}
	record.Status = domain.BatchStatusAnchoring
	record.UpdatedAt = claimedAt
	f.batches[batchID] = record
	return nil
}

func (f *fakeReconcileBatchRepo) AttachAnchorProof(
	_ context.Context,
	batchID string,
	expectedStatus domain.BatchStatus,
	expectedUpdatedAt time.Time,
	proof domain.AnchorProofRecord,
	updatedAt time.Time,
) error {
	if f.attachErr != nil {
		return f.attachErr
	}
	record, ok := f.batches[batchID]
	if !ok {
		return repository.ErrNotFound
	}
	if record.Status != expectedStatus || !record.UpdatedAt.Equal(expectedUpdatedAt) {
		return repository.ErrConflict
	}
	record.Status = domain.BatchStatusAnchored
	record.AnchorProof = &proof
	hash := proof.AnchorHash
	record.AnchorHash = &hash
	record.UpdatedAt = updatedAt
	f.batches[batchID] = record
	return nil
}

func (f *fakeReconcileBatchRepo) ListPendingBatches(_ context.Context, limit int) ([]domain.BatchRecord, error) {
	if f.listPendingErr != nil {
		return nil, f.listPendingErr
	}
	if limit <= 0 {
		limit = defaultReconcileLimit
	}

	out := make([]domain.BatchRecord, 0, len(f.batches))
	for _, record := range f.batches {
		if record.TraceMode == domain.TraceModeBlockchain && record.Status == domain.BatchStatusPendingAnchor {
			out = append(out, record)
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

type fakeReconcileRepo struct {
	jobs            map[string]domain.ReconcileJobRecord
	jobOrder        []string
	items           map[string][]domain.ReconcileJobItemRecord
	createErr       error
	updateErr       error
	addItemsErr     error
	createCallCount int
}

func newFakeReconcileRepo() *fakeReconcileRepo {
	return &fakeReconcileRepo{
		jobs:  make(map[string]domain.ReconcileJobRecord),
		items: make(map[string][]domain.ReconcileJobItemRecord),
	}
}

func (f *fakeReconcileRepo) CreateReconcileJob(_ context.Context, params domain.CreateReconcileJobParams) (domain.ReconcileJobRecord, error) {
	f.createCallCount++
	if f.createErr != nil {
		return domain.ReconcileJobRecord{}, f.createErr
	}
	status := params.Status
	if status == "" {
		status = domain.ReconcileJobStatusAccepted
	}
	record := domain.ReconcileJobRecord{
		JobID:          params.JobID,
		TriggerType:    params.TriggerType,
		Status:         status,
		RequestedCount: params.RequestedCount,
		ScheduledCount: params.ScheduledCount,
		SkippedCount:   params.SkippedCount,
		ErrorMessage:   params.ErrorMessage,
		CreatedAt:      params.CreatedAt,
		UpdatedAt:      params.UpdatedAt,
	}
	f.jobs[record.JobID] = record
	f.jobOrder = append(f.jobOrder, record.JobID)
	return record, nil
}

func (f *fakeReconcileRepo) AddReconcileJobItems(_ context.Context, jobID string, items []domain.ReconcileJobItemRecord) error {
	if f.addItemsErr != nil {
		return f.addItemsErr
	}
	f.items[jobID] = append(f.items[jobID], items...)
	return nil
}

func (f *fakeReconcileRepo) UpdateReconcileJobStatus(
	_ context.Context,
	jobID string,
	status domain.ReconcileJobStatus,
	errMsg *string,
	updatedAt time.Time,
) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	record, ok := f.jobs[jobID]
	if !ok {
		return repository.ErrNotFound
	}
	record.Status = status
	record.UpdatedAt = updatedAt
	record.ErrorMessage = errMsg
	f.jobs[jobID] = record
	return nil
}

func (f *fakeReconcileRepo) ListReconcileStats(_ context.Context) (domain.ReconcileStats, error) {
	return domain.ReconcileStats{}, nil
}

func (f *fakeReconcileRepo) latestJob() domain.ReconcileJobRecord {
	if len(f.jobOrder) == 0 {
		return domain.ReconcileJobRecord{}
	}
	return f.jobs[f.jobOrder[len(f.jobOrder)-1]]
}

type fakeReconcileAnchorClient struct {
	proofs map[string]domain.AnchorProofRecord
	errs   map[string]error
}

func (f *fakeReconcileAnchorClient) AnchorBatch(_ context.Context, req evm.AnchorBatchRequest) (domain.AnchorProofRecord, error) {
	if err, ok := f.errs[req.BatchID]; ok {
		return domain.AnchorProofRecord{}, err
	}
	if proof, ok := f.proofs[req.BatchID]; ok {
		return proof, nil
	}
	return domain.AnchorProofRecord{}, fmt.Errorf("%w: missing proof for %s", evm.ErrContractCall, req.BatchID)
}

func samplePendingBatch(batchID string, retryCount int) domain.BatchRecord {
	anchorHash := "0x1111111111111111111111111111111111111111111111111111111111111111"
	now := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC)
	return domain.BatchRecord{
		BatchID:       batchID,
		TraceCode:     "TRC-RECO-NCIL",
		TraceMode:     domain.TraceModeBlockchain,
		Status:        domain.BatchStatusPendingAnchor,
		OrchardID:     "orchard-1",
		OrchardName:   "orchard",
		PlotID:        "plot-1",
		HarvestedAt:   now,
		Summary:       domain.BatchSummary{Total: 10, Green: 1, Half: 3, Red: 6, Young: 0, UnripeCount: 1, UnripeRatio: 0.1, UnripeHandling: domain.UnripeHandlingSortedOut},
		AnchorHash:    &anchorHash,
		RetryCount:    retryCount,
		ConfirmUnripe: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func sampleAnchorProof(txHash, anchorHash string) domain.AnchorProofRecord {
	return domain.AnchorProofRecord{
		TxHash:          txHash,
		BlockNumber:     123,
		ChainID:         "31337",
		ContractAddress: "0xdef",
		AnchorHash:      anchorHash,
		AnchoredAt:      time.Date(2026, 3, 2, 10, 1, 0, 0, time.UTC),
	}
}
