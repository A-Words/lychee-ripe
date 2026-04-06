package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lychee-ripe/gateway/internal/chain/evm"
	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/repository"
)

const (
	TraceVerifyStatusPass    = "pass"
	TraceVerifyStatusFail    = "fail"
	TraceVerifyStatusPending = "pending"
	TraceVerifyStatusRecorded = "recorded"

	traceReasonPending          = "batch is not anchored yet"
	traceReasonHashMatched      = "anchor_hash matches on-chain record"
	traceReasonHashMismatched   = "anchor_hash does not match on-chain record"
	traceReasonOnChainNotFound  = "on-chain anchor not found"
	traceReasonChainUnavailable = "chain query unavailable"
	traceReasonRecorded         = "batch is recorded in gateway database"
)

type TraceAnchorClient interface {
	GetBatchAnchor(ctx context.Context, batchID string) (evm.BatchAnchorOnChain, error)
}

type TraceQueryResult struct {
	Batch        domain.BatchRecord
	VerifyStatus string
	Reason       string
}

type TraceService struct {
	repo         repository.BatchRepository
	anchorClient TraceAnchorClient
	traceMode    domain.TraceMode
}

func NewTraceService(repo repository.BatchRepository, anchorClient TraceAnchorClient, traceMode domain.TraceMode) *TraceService {
	return &TraceService{
		repo:         repo,
		anchorClient: anchorClient,
		traceMode:    traceMode,
	}
}

func (s *TraceService) GetPublicTrace(ctx context.Context, traceCode string) (TraceQueryResult, error) {
	code := strings.TrimSpace(traceCode)
	if code == "" {
		return TraceQueryResult{}, fmt.Errorf("%w: trace_code is required", ErrInvalidRequest)
	}

	record, err := s.repo.GetBatchByTraceCode(ctx, code)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return TraceQueryResult{}, fmt.Errorf("%w: trace not found", ErrNotFound)
		}
		return TraceQueryResult{}, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
	}

	if s.traceMode == domain.TraceModeDatabase || record.TraceMode == domain.TraceModeDatabase {
		return TraceQueryResult{
			Batch:        record,
			VerifyStatus: TraceVerifyStatusRecorded,
			Reason:       traceReasonRecorded,
		}, nil
	}

	if record.Status != domain.BatchStatusAnchored || record.AnchorProof == nil {
		return TraceQueryResult{
			Batch:        record,
			VerifyStatus: TraceVerifyStatusPending,
			Reason:       traceReasonPending,
		}, nil
	}

	if s.anchorClient == nil {
		return TraceQueryResult{}, fmt.Errorf("%w: %s", ErrServiceUnavailable, traceReasonChainUnavailable)
	}

	onChain, err := s.anchorClient.GetBatchAnchor(ctx, record.BatchID)
	if err != nil {
		switch {
		case errors.Is(err, evm.ErrAnchorNotFound):
			return TraceQueryResult{
				Batch:        record,
				VerifyStatus: TraceVerifyStatusFail,
				Reason:       traceReasonOnChainNotFound,
			}, nil
		case errors.Is(err, evm.ErrNodeUnavailable), errors.Is(err, evm.ErrContractCall):
			return TraceQueryResult{}, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
		default:
			return TraceQueryResult{}, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
		}
	}

	localHash, err := computeAnchorHash(anchorHashPayload{
		BatchID:     record.BatchID,
		TraceCode:   record.TraceCode,
		OrchardID:   record.OrchardID,
		OrchardName: record.OrchardName,
		PlotID:      record.PlotID,
		PlotName:    record.PlotName,
		HarvestedAt: record.HarvestedAt.UTC(),
		Summary:     record.Summary,
		Note:        record.Note,
	})
	if err != nil {
		return TraceQueryResult{}, fmt.Errorf("%w: compute anchor hash: %v", ErrServiceUnavailable, err)
	}

	if strings.EqualFold(localHash, onChain.AnchorHash) {
		return TraceQueryResult{
			Batch:        record,
			VerifyStatus: TraceVerifyStatusPass,
			Reason:       traceReasonHashMatched,
		}, nil
	}

	return TraceQueryResult{
		Batch:        record,
		VerifyStatus: TraceVerifyStatusFail,
		Reason:       traceReasonHashMismatched,
	}, nil
}
