package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/lychee-ripe/gateway/internal/chain/evm"
	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/repository"
)

func TestCreateBatchSuccessAnchored(t *testing.T) {
	t.Parallel()

	repo := newFakeBatchRepository()
	anchor := &fakeAnchorClient{
		fn: func(_ context.Context, req evm.AnchorBatchRequest) (domain.AnchorProofRecord, error) {
			return domain.AnchorProofRecord{
				TxHash:          "0xabc",
				BlockNumber:     100,
				ChainID:         "31337",
				ContractAddress: "0xdef",
				AnchorHash:      req.AnchorHash,
				AnchoredAt:      time.Date(2026, 3, 2, 10, 31, 1, 0, time.UTC),
			}, nil
		},
	}
	svc := NewBatchCreateService(repo, anchor, true, nil)
	svc.batchIDFn = func() string { return "batch_01" }
	svc.traceCodeFn = func() string { return "TRC-ABCD-EFGH" }
	svc.nowFn = func() time.Time { return time.Date(2026, 3, 2, 10, 31, 0, 0, time.UTC) }

	result, err := svc.CreateBatch(context.Background(), sampleCreateInput(10, 1, 3, 6, 0, false))
	if err != nil {
		t.Fatalf("CreateBatch failed: %v", err)
	}
	if result.HTTPStatus != 201 {
		t.Fatalf("http status = %d, want 201", result.HTTPStatus)
	}
	if result.Batch.Status != domain.BatchStatusAnchored {
		t.Fatalf("status = %q, want anchored", result.Batch.Status)
	}
	if result.Batch.AnchorProof == nil {
		t.Fatal("anchor_proof should not be nil")
	}
	if result.Batch.AnchorProof.TxHash != "0xabc" {
		t.Fatalf("tx_hash = %q, want 0xabc", result.Batch.AnchorProof.TxHash)
	}
}

func TestCreateBatchDegradesOnNodeUnavailable(t *testing.T) {
	t.Parallel()

	repo := newFakeBatchRepository()
	anchor := &fakeAnchorClient{
		fn: func(_ context.Context, _ evm.AnchorBatchRequest) (domain.AnchorProofRecord, error) {
			return domain.AnchorProofRecord{}, fmt.Errorf("%w: connection refused", evm.ErrNodeUnavailable)
		},
	}
	svc := NewBatchCreateService(repo, anchor, true, nil)
	svc.batchIDFn = func() string { return "batch_02" }
	svc.traceCodeFn = func() string { return "TRC-IJKL-MNOP" }

	result, err := svc.CreateBatch(context.Background(), sampleCreateInput(10, 1, 3, 6, 0, false))
	if err != nil {
		t.Fatalf("CreateBatch failed: %v", err)
	}
	if result.HTTPStatus != 202 {
		t.Fatalf("http status = %d, want 202", result.HTTPStatus)
	}
	if result.Batch.Status != domain.BatchStatusPendingAnchor {
		t.Fatalf("status = %q, want pending_anchor", result.Batch.Status)
	}
	if result.Batch.LastError == nil || !strings.Contains(*result.Batch.LastError, "node unavailable") {
		t.Fatalf("last_error = %v, want node unavailable message", result.Batch.LastError)
	}
	if result.Batch.RetryCount != 1 {
		t.Fatalf("retry_count = %d, want 1", result.Batch.RetryCount)
	}
}

func TestCreateBatchDegradesOnTxReverted(t *testing.T) {
	t.Parallel()

	repo := newFakeBatchRepository()
	anchor := &fakeAnchorClient{
		fn: func(_ context.Context, _ evm.AnchorBatchRequest) (domain.AnchorProofRecord, error) {
			return domain.AnchorProofRecord{}, fmt.Errorf("%w: status=0", evm.ErrTxReverted)
		},
	}
	svc := NewBatchCreateService(repo, anchor, true, nil)
	svc.batchIDFn = func() string { return "batch_03" }
	svc.traceCodeFn = func() string { return "TRC-QRST-UVWX" }

	result, err := svc.CreateBatch(context.Background(), sampleCreateInput(10, 1, 3, 6, 0, false))
	if err != nil {
		t.Fatalf("CreateBatch failed: %v", err)
	}
	if result.HTTPStatus != 202 {
		t.Fatalf("http status = %d, want 202", result.HTTPStatus)
	}
	if result.Batch.Status != domain.BatchStatusPendingAnchor {
		t.Fatalf("status = %q, want pending_anchor", result.Batch.Status)
	}
}

func TestCreateBatchRejectsUnripeWithoutConfirm(t *testing.T) {
	t.Parallel()

	repo := newFakeBatchRepository()
	svc := NewBatchCreateService(repo, nil, false, nil)
	svc.batchIDFn = func() string { return "batch_04" }
	svc.traceCodeFn = func() string { return "TRC-AAAA-BBBB" }

	_, err := svc.CreateBatch(context.Background(), sampleCreateInput(10, 2, 2, 4, 2, false))
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v, want ErrInvalidRequest", err)
	}
}

func TestCreateBatchAllowsThresholdBoundary(t *testing.T) {
	t.Parallel()

	repo := newFakeBatchRepository()
	svc := NewBatchCreateService(repo, nil, false, nil)
	svc.batchIDFn = func() string { return "batch_05" }
	svc.traceCodeFn = func() string { return "TRC-CCCC-DDDD" }

	result, err := svc.CreateBatch(context.Background(), sampleCreateInput(20, 2, 5, 12, 1, false))
	if err != nil {
		t.Fatalf("CreateBatch failed: %v", err)
	}
	if result.HTTPStatus != 202 {
		t.Fatalf("http status = %d, want 202", result.HTTPStatus)
	}
	if result.Batch.Summary.UnripeRatio != 0.15 {
		t.Fatalf("unripe_ratio = %v, want 0.15", result.Batch.Summary.UnripeRatio)
	}
}

func TestCreateBatchRejectsTotalZero(t *testing.T) {
	t.Parallel()

	repo := newFakeBatchRepository()
	svc := NewBatchCreateService(repo, nil, false, nil)

	_, err := svc.CreateBatch(context.Background(), sampleCreateInput(0, 0, 0, 0, 0, false))
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v, want ErrInvalidRequest", err)
	}
}

func TestCreateBatchRejectsSummaryMismatch(t *testing.T) {
	t.Parallel()

	repo := newFakeBatchRepository()
	svc := NewBatchCreateService(repo, nil, false, nil)

	_, err := svc.CreateBatch(context.Background(), sampleCreateInput(10, 1, 1, 1, 1, false))
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v, want ErrInvalidRequest", err)
	}
}

func TestCreateBatchRetriesOnConflict(t *testing.T) {
	t.Parallel()

	repo := newFakeBatchRepository()
	repo.createErrors = []error{repository.ErrConflict, nil}
	svc := NewBatchCreateService(repo, nil, false, nil)

	batchIDs := []string{"batch_conflict", "batch_final"}
	traceCodes := []string{"TRC-FAIL-0001", "TRC-OKAY-0002"}
	svc.batchIDFn = func() string {
		out := batchIDs[0]
		batchIDs = batchIDs[1:]
		return out
	}
	svc.traceCodeFn = func() string {
		out := traceCodes[0]
		traceCodes = traceCodes[1:]
		return out
	}

	result, err := svc.CreateBatch(context.Background(), sampleCreateInput(10, 1, 3, 6, 0, false))
	if err != nil {
		t.Fatalf("CreateBatch failed: %v", err)
	}
	if result.Batch.BatchID != "batch_final" {
		t.Fatalf("batch_id = %q, want batch_final", result.Batch.BatchID)
	}
	if repo.createCallCount != 2 {
		t.Fatalf("create calls = %d, want 2", repo.createCallCount)
	}
}

func TestCreateBatchReturnsConflictAfterMaxRetries(t *testing.T) {
	t.Parallel()

	repo := newFakeBatchRepository()
	repo.createErrors = []error{repository.ErrConflict, repository.ErrConflict, repository.ErrConflict}
	svc := NewBatchCreateService(repo, nil, false, nil)
	svc.batchIDFn = func() string { return "batch_conflict" }
	svc.traceCodeFn = func() string { return "TRC-CONF-LICT" }

	_, err := svc.CreateBatch(context.Background(), sampleCreateInput(10, 1, 3, 6, 0, false))
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("error = %v, want ErrConflict", err)
	}
	if repo.createCallCount != 3 {
		t.Fatalf("create calls = %d, want 3", repo.createCallCount)
	}
}

func sampleCreateInput(total, green, half, red, young int, confirm bool) BatchCreateInput {
	plotName := "plot-a1"
	note := "note"
	return BatchCreateInput{
		OrchardID:   "orchard-1",
		OrchardName: "orchard",
		PlotID:      "plot-1",
		PlotName:    &plotName,
		HarvestedAt: "2026-03-02T10:30:00Z",
		Summary: BatchSummaryInput{
			Total: total,
			Green: green,
			Half:  half,
			Red:   red,
			Young: young,
		},
		Note:          &note,
		ConfirmUnripe: confirm,
	}
}

type fakeAnchorClient struct {
	fn func(ctx context.Context, req evm.AnchorBatchRequest) (domain.AnchorProofRecord, error)
}

func (f *fakeAnchorClient) AnchorBatch(ctx context.Context, req evm.AnchorBatchRequest) (domain.AnchorProofRecord, error) {
	return f.fn(ctx, req)
}

type fakeBatchRepository struct {
	batches         map[string]domain.BatchRecord
	traceIndex      map[string]string
	createErrors    []error
	createCallCount int
}

func newFakeBatchRepository() *fakeBatchRepository {
	return &fakeBatchRepository{
		batches:    make(map[string]domain.BatchRecord),
		traceIndex: make(map[string]string),
	}
}

func (f *fakeBatchRepository) CreateBatch(_ context.Context, params domain.CreateBatchParams) (domain.BatchRecord, error) {
	f.createCallCount++
	if len(f.createErrors) > 0 {
		err := f.createErrors[0]
		f.createErrors = f.createErrors[1:]
		if err != nil {
			return domain.BatchRecord{}, err
		}
	}

	if _, ok := f.batches[params.BatchID]; ok {
		return domain.BatchRecord{}, repository.ErrConflict
	}
	if _, ok := f.traceIndex[params.TraceCode]; ok {
		return domain.BatchRecord{}, repository.ErrConflict
	}

	record := domain.BatchRecord{
		BatchID:       params.BatchID,
		TraceCode:     params.TraceCode,
		Status:        params.Status,
		OrchardID:     params.OrchardID,
		OrchardName:   params.OrchardName,
		PlotID:        params.PlotID,
		PlotName:      params.PlotName,
		HarvestedAt:   params.HarvestedAt,
		Summary:       params.Summary,
		Note:          params.Note,
		AnchorHash:    params.AnchorHash,
		ConfirmUnripe: params.ConfirmUnripe,
		RetryCount:    params.RetryCount,
		LastError:     params.LastError,
		CreatedAt:     params.CreatedAt,
		UpdatedAt:     params.UpdatedAt,
	}
	f.batches[params.BatchID] = record
	f.traceIndex[params.TraceCode] = params.BatchID
	return record, nil
}

func (f *fakeBatchRepository) GetBatchByID(_ context.Context, batchID string) (domain.BatchRecord, error) {
	record, ok := f.batches[batchID]
	if !ok {
		return domain.BatchRecord{}, repository.ErrNotFound
	}
	return record, nil
}

func (f *fakeBatchRepository) GetBatchByTraceCode(_ context.Context, traceCode string) (domain.BatchRecord, error) {
	batchID, ok := f.traceIndex[traceCode]
	if !ok {
		return domain.BatchRecord{}, repository.ErrNotFound
	}
	return f.batches[batchID], nil
}

func (f *fakeBatchRepository) UpdateBatchStatus(
	_ context.Context,
	batchID string,
	status domain.BatchStatus,
	lastError *string,
	retryCount *int,
	updatedAt time.Time,
) error {
	record, ok := f.batches[batchID]
	if !ok {
		return repository.ErrNotFound
	}
	record.Status = status
	if lastError != nil {
		msg := *lastError
		record.LastError = &msg
	}
	if retryCount != nil {
		record.RetryCount = *retryCount
	}
	record.UpdatedAt = updatedAt
	f.batches[batchID] = record
	return nil
}

func (f *fakeBatchRepository) AttachAnchorProof(
	_ context.Context,
	batchID string,
	proof domain.AnchorProofRecord,
	updatedAt time.Time,
) error {
	record, ok := f.batches[batchID]
	if !ok {
		return repository.ErrNotFound
	}
	record.Status = domain.BatchStatusAnchored
	record.AnchorProof = &proof
	anchorHash := proof.AnchorHash
	record.AnchorHash = &anchorHash
	record.UpdatedAt = updatedAt
	f.batches[batchID] = record
	return nil
}

func (f *fakeBatchRepository) ListPendingBatches(_ context.Context, _ int) ([]domain.BatchRecord, error) {
	out := make([]domain.BatchRecord, 0)
	for _, record := range f.batches {
		if record.Status == domain.BatchStatusPendingAnchor {
			out = append(out, record)
		}
	}
	return out, nil
}
