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
)

func TestGetPublicTracePassWhenOnChainHashMatches(t *testing.T) {
	t.Parallel()

	record := sampleAnchoredTraceBatch(t)
	repo := newFakeBatchRepository()
	repo.batches[record.BatchID] = record
	repo.traceIndex[record.TraceCode] = record.BatchID

	chain := &fakeTraceAnchorClient{
		result: evm.BatchAnchorOnChain{
			BatchID:    record.BatchID,
			AnchorHash: *record.AnchorHash,
			AnchoredAt: record.AnchorProof.AnchoredAt,
		},
	}
	svc := NewTraceService(repo, chain, domain.TraceModeBlockchain)

	got, err := svc.GetPublicTrace(context.Background(), record.TraceCode)
	if err != nil {
		t.Fatalf("GetPublicTrace failed: %v", err)
	}
	if got.VerifyStatus != TraceVerifyStatusPass {
		t.Fatalf("verify_status = %q, want pass", got.VerifyStatus)
	}
	if got.Reason != traceReasonHashMatched {
		t.Fatalf("reason = %q, want %q", got.Reason, traceReasonHashMatched)
	}
}

func TestGetPublicTraceFailWhenOnChainHashMismatches(t *testing.T) {
	t.Parallel()

	record := sampleAnchoredTraceBatch(t)
	repo := newFakeBatchRepository()
	repo.batches[record.BatchID] = record
	repo.traceIndex[record.TraceCode] = record.BatchID

	chain := &fakeTraceAnchorClient{
		result: evm.BatchAnchorOnChain{
			BatchID:    record.BatchID,
			AnchorHash: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			AnchoredAt: record.AnchorProof.AnchoredAt,
		},
	}
	svc := NewTraceService(repo, chain, domain.TraceModeBlockchain)

	got, err := svc.GetPublicTrace(context.Background(), record.TraceCode)
	if err != nil {
		t.Fatalf("GetPublicTrace failed: %v", err)
	}
	if got.VerifyStatus != TraceVerifyStatusFail {
		t.Fatalf("verify_status = %q, want fail", got.VerifyStatus)
	}
	if got.Reason != traceReasonHashMismatched {
		t.Fatalf("reason = %q, want %q", got.Reason, traceReasonHashMismatched)
	}
}

func TestGetPublicTracePendingWhenBatchNotAnchored(t *testing.T) {
	t.Parallel()

	record := sampleAnchoredTraceBatch(t)
	record.Status = domain.BatchStatusPendingAnchor
	record.AnchorProof = nil

	repo := newFakeBatchRepository()
	repo.batches[record.BatchID] = record
	repo.traceIndex[record.TraceCode] = record.BatchID

	svc := NewTraceService(repo, &fakeTraceAnchorClient{}, domain.TraceModeBlockchain)
	got, err := svc.GetPublicTrace(context.Background(), record.TraceCode)
	if err != nil {
		t.Fatalf("GetPublicTrace failed: %v", err)
	}
	if got.VerifyStatus != TraceVerifyStatusPending {
		t.Fatalf("verify_status = %q, want pending", got.VerifyStatus)
	}
	if got.Reason != traceReasonPending {
		t.Fatalf("reason = %q, want %q", got.Reason, traceReasonPending)
	}
}

func TestGetPublicTraceReturnsNotFound(t *testing.T) {
	t.Parallel()

	svc := NewTraceService(newFakeBatchRepository(), &fakeTraceAnchorClient{}, domain.TraceModeBlockchain)
	_, err := svc.GetPublicTrace(context.Background(), "TRC-NOPE-0000")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestGetPublicTraceReturnsServiceUnavailableWhenChainUnavailable(t *testing.T) {
	t.Parallel()

	record := sampleAnchoredTraceBatch(t)
	repo := newFakeBatchRepository()
	repo.batches[record.BatchID] = record
	repo.traceIndex[record.TraceCode] = record.BatchID

	chain := &fakeTraceAnchorClient{
		err: fmt.Errorf("%w: dial tcp timeout", evm.ErrNodeUnavailable),
	}
	svc := NewTraceService(repo, chain, domain.TraceModeBlockchain)
	_, err := svc.GetPublicTrace(context.Background(), record.TraceCode)
	if !errors.Is(err, ErrServiceUnavailable) {
		t.Fatalf("error = %v, want ErrServiceUnavailable", err)
	}
}

func TestGetPublicTraceFailsWhenOnChainAnchorMissing(t *testing.T) {
	t.Parallel()

	record := sampleAnchoredTraceBatch(t)
	repo := newFakeBatchRepository()
	repo.batches[record.BatchID] = record
	repo.traceIndex[record.TraceCode] = record.BatchID

	chain := &fakeTraceAnchorClient{
		err: fmt.Errorf("%w: %s", evm.ErrAnchorNotFound, record.BatchID),
	}
	svc := NewTraceService(repo, chain, domain.TraceModeBlockchain)

	got, err := svc.GetPublicTrace(context.Background(), record.TraceCode)
	if err != nil {
		t.Fatalf("GetPublicTrace failed: %v", err)
	}
	if got.VerifyStatus != TraceVerifyStatusFail {
		t.Fatalf("verify_status = %q, want fail", got.VerifyStatus)
	}
	if got.Reason != traceReasonOnChainNotFound {
		t.Fatalf("reason = %q, want %q", got.Reason, traceReasonOnChainNotFound)
	}
}

type fakeTraceAnchorClient struct {
	result evm.BatchAnchorOnChain
	err    error
}

func (f *fakeTraceAnchorClient) GetBatchAnchor(_ context.Context, _ string) (evm.BatchAnchorOnChain, error) {
	if f.err != nil {
		return evm.BatchAnchorOnChain{}, f.err
	}
	return f.result, nil
}

func sampleAnchoredTraceBatch(t *testing.T) domain.BatchRecord {
	t.Helper()

	plotName := "plot-a1"
	note := "trace note"
	createdAt := time.Date(2026, 3, 2, 10, 31, 0, 0, time.UTC)
	record := domain.BatchRecord{
		BatchID:     "batch_trace_01",
		TraceCode:   "TRC-TRAC-E001",
		TraceMode:   domain.TraceModeBlockchain,
		Status:      domain.BatchStatusAnchored,
		OrchardID:   "orchard-1",
		OrchardName: "orchard",
		PlotID:      "plot-1",
		PlotName:    &plotName,
		HarvestedAt: time.Date(2026, 3, 2, 10, 30, 0, 0, time.UTC),
		Summary: domain.BatchSummary{
			Total:          10,
			Green:          1,
			Half:           3,
			Red:            6,
			Young:          0,
			UnripeCount:    1,
			UnripeRatio:    0.1,
			UnripeHandling: domain.UnripeHandlingSortedOut,
		},
		Note:      &note,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}

	hash, err := computeAnchorHash(anchorHashPayload{
		BatchID:     record.BatchID,
		TraceCode:   record.TraceCode,
		OrchardID:   record.OrchardID,
		OrchardName: record.OrchardName,
		PlotID:      record.PlotID,
		PlotName:    record.PlotName,
		HarvestedAt: record.HarvestedAt,
		Summary:     record.Summary,
		Note:        record.Note,
	})
	if err != nil {
		t.Fatalf("computeAnchorHash failed: %v", err)
	}
	hash = "0x" + strings.ToLower(strings.TrimPrefix(hash, "0x"))
	record.AnchorHash = &hash
	record.AnchorProof = &domain.AnchorProofRecord{
		TxHash:          "0xabc",
		BlockNumber:     101,
		ChainID:         "31337",
		ContractAddress: "0xdef",
		AnchorHash:      hash,
		AnchoredAt:      createdAt,
	}
	return record
}
