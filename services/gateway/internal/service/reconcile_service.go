package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/lychee-ripe/gateway/internal/chain/evm"
	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/repository"
	"github.com/oklog/ulid/v2"
)

const (
	defaultReconcileLimit   = 100
	maxReconcileLimit       = 1000
	maxReconcileRetry       = 3
	autoReconcileInterval   = 30 * time.Second
	reconcileAcceptedPrompt = "reconcile accepted"
	reconcileLostRaceReason = "batch claimed by another worker"
)

type ManualReconcileInput struct {
	BatchIDs []string `json:"batch_ids"`
	Limit    *int     `json:"limit"`
}

type ReconcileResult struct {
	Accepted       bool
	RequestedCount int
	ScheduledCount int
	SkippedCount   int
	Message        string
}

type ReconcileService struct {
	batchRepo     repository.BatchRepository
	reconcileRepo repository.ReconcileRepository
	anchorClient  AnchorClient
	traceMode     domain.TraceMode
	logger        *slog.Logger

	nowFn          func() time.Time
	workerInterval time.Duration
	maxRetry       int

	mu sync.Mutex
}

func NewReconcileService(
	batchRepo repository.BatchRepository,
	reconcileRepo repository.ReconcileRepository,
	anchorClient AnchorClient,
	traceMode domain.TraceMode,
	logger *slog.Logger,
) *ReconcileService {
	if logger == nil {
		logger = slog.Default()
	}

	return &ReconcileService{
		batchRepo:      batchRepo,
		reconcileRepo:  reconcileRepo,
		anchorClient:   anchorClient,
		traceMode:      traceMode,
		logger:         logger,
		nowFn:          func() time.Time { return time.Now().UTC() },
		workerInterval: autoReconcileInterval,
		maxRetry:       maxReconcileRetry,
	}
}

func (s *ReconcileService) TriggerManualReconcile(ctx context.Context, input ManualReconcileInput) (ReconcileResult, error) {
	if s.traceMode != domain.TraceModeBlockchain {
		return ReconcileResult{}, fmt.Errorf("%w: reconcile is only available when trace.mode=blockchain", ErrConflict)
	}
	if s.anchorClient == nil {
		return ReconcileResult{}, fmt.Errorf("%w: blockchain anchor client unavailable", ErrServiceUnavailable)
	}

	limit, err := normalizeReconcileLimit(input.Limit)
	if err != nil {
		return ReconcileResult{}, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}

	scheduled, requestedCount, skippedCount, err := s.selectManualTargets(ctx, input.BatchIDs, limit)
	if err != nil {
		return ReconcileResult{}, err
	}
	if len(scheduled) == 0 {
		return ReconcileResult{}, fmt.Errorf("%w: no requested pending batch found", ErrNotFound)
	}

	return s.runReconcile(ctx, domain.ReconcileTriggerManual, scheduled, requestedCount, skippedCount)
}

func (s *ReconcileService) RunAutoReconcileOnce(ctx context.Context) error {
	if s.traceMode != domain.TraceModeBlockchain || s.anchorClient == nil {
		s.logger.Debug("auto reconcile skipped because blockchain mode is disabled")
		return nil
	}

	pending, err := s.batchRepo.ListPendingBatches(ctx, defaultReconcileLimit)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
	}
	if len(pending) == 0 {
		return nil
	}

	_, err = s.runReconcile(ctx, domain.ReconcileTriggerAuto, pending, len(pending), 0)
	return err
}

func (s *ReconcileService) runReconcile(
	ctx context.Context,
	trigger domain.ReconcileTriggerType,
	scheduled []domain.BatchRecord,
	requestedCount int,
	skippedCount int,
) (ReconcileResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.nowFn().UTC()
	job, err := s.reconcileRepo.CreateReconcileJob(ctx, domain.CreateReconcileJobParams{
		JobID:          defaultReconcileJobID(),
		TriggerType:    trigger,
		Status:         domain.ReconcileJobStatusAccepted,
		RequestedCount: requestedCount,
		ScheduledCount: len(scheduled),
		SkippedCount:   skippedCount,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		return ReconcileResult{}, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
	}

	if err := s.reconcileRepo.UpdateReconcileJobStatus(ctx, job.JobID, domain.ReconcileJobStatusRunning, nil, s.nowFn().UTC()); err != nil {
		return ReconcileResult{}, s.failJob(ctx, job.JobID, err)
	}

	for _, batch := range scheduled {
		item, execErr := s.reconcileOne(ctx, batch)
		if execErr != nil {
			return ReconcileResult{}, s.failJob(ctx, job.JobID, execErr)
		}

		if err := s.reconcileRepo.AddReconcileJobItems(ctx, job.JobID, []domain.ReconcileJobItemRecord{item}); err != nil {
			return ReconcileResult{}, s.failJob(ctx, job.JobID, err)
		}
	}

	if err := s.reconcileRepo.UpdateReconcileJobStatus(ctx, job.JobID, domain.ReconcileJobStatusCompleted, nil, s.nowFn().UTC()); err != nil {
		return ReconcileResult{}, s.failJob(ctx, job.JobID, err)
	}

	return ReconcileResult{
		Accepted:       true,
		RequestedCount: requestedCount,
		ScheduledCount: len(scheduled),
		SkippedCount:   skippedCount,
		Message:        reconcileAcceptedPrompt,
	}, nil
}

func (s *ReconcileService) reconcileOne(ctx context.Context, batch domain.BatchRecord) (domain.ReconcileJobItemRecord, error) {
	if batch.TraceMode != domain.TraceModeBlockchain {
		return domain.ReconcileJobItemRecord{}, fmt.Errorf("%w: batch %s is not a blockchain batch", ErrInvalidRequest, batch.BatchID)
	}
	if batch.Status != domain.BatchStatusPendingAnchor {
		return domain.ReconcileJobItemRecord{}, fmt.Errorf("%w: batch %s is not pending_anchor", ErrInvalidRequest, batch.BatchID)
	}

	attemptNo := batch.RetryCount + 1
	claimAt := s.nowFn().UTC()
	if err := s.batchRepo.ClaimPendingBatch(ctx, batch.BatchID, batch.UpdatedAt, claimAt); err != nil {
		if errors.Is(err, repository.ErrConflict) || errors.Is(err, repository.ErrNotFound) {
			return domain.ReconcileJobItemRecord{
				BatchID:      batch.BatchID,
				BeforeStatus: domain.BatchStatusPendingAnchor,
				AfterStatus:  domain.BatchStatusPendingAnchor,
				AttemptNo:    attemptNo,
				ErrorMessage: stringPtr(reconcileLostRaceReason),
				CreatedAt:    claimAt,
			}, nil
		}
		return domain.ReconcileJobItemRecord{}, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
	}
	batch.Status = domain.BatchStatusAnchoring
	batch.UpdatedAt = claimAt

	anchorHash, err := resolveAnchorHash(batch)
	if err != nil {
		return domain.ReconcileJobItemRecord{}, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
	}

	proof, err := s.anchorClient.AnchorBatch(ctx, evm.AnchorBatchRequest{
		BatchID:    batch.BatchID,
		AnchorHash: anchorHash,
		Timestamp:  s.nowFn().UTC(),
	})
	if err == nil {
		if err := s.batchRepo.AttachAnchorProof(ctx, batch.BatchID, domain.BatchStatusAnchoring, batch.UpdatedAt, proof, s.nowFn().UTC()); err != nil {
			if errors.Is(err, repository.ErrConflict) || errors.Is(err, repository.ErrNotFound) {
				return domain.ReconcileJobItemRecord{
					BatchID:      batch.BatchID,
					BeforeStatus: domain.BatchStatusPendingAnchor,
					AfterStatus:  domain.BatchStatusPendingAnchor,
					AttemptNo:    attemptNo,
					ErrorMessage: stringPtr(reconcileLostRaceReason),
					CreatedAt:    s.nowFn().UTC(),
				}, nil
			}
			return domain.ReconcileJobItemRecord{}, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
		}
		return domain.ReconcileJobItemRecord{
			BatchID:      batch.BatchID,
			BeforeStatus: domain.BatchStatusPendingAnchor,
			AfterStatus:  domain.BatchStatusAnchored,
			AttemptNo:    attemptNo,
			CreatedAt:    s.nowFn().UTC(),
		}, nil
	}

	if !isRecoverableChainError(err) {
		return domain.ReconcileJobItemRecord{}, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
	}

	nextStatus := domain.BatchStatusPendingAnchor
	if attemptNo >= s.maxRetry {
		nextStatus = domain.BatchStatusAnchorFailed
	}
	lastError := err.Error()
	retryCount := attemptNo
	if err := s.batchRepo.UpdateBatchStatus(ctx, batch.BatchID, domain.BatchStatusAnchoring, batch.UpdatedAt, nextStatus, &lastError, &retryCount, s.nowFn().UTC()); err != nil {
		if errors.Is(err, repository.ErrConflict) || errors.Is(err, repository.ErrNotFound) {
			return domain.ReconcileJobItemRecord{
				BatchID:      batch.BatchID,
				BeforeStatus: domain.BatchStatusPendingAnchor,
				AfterStatus:  domain.BatchStatusPendingAnchor,
				AttemptNo:    attemptNo,
				ErrorMessage: stringPtr(reconcileLostRaceReason),
				CreatedAt:    s.nowFn().UTC(),
			}, nil
		}
		return domain.ReconcileJobItemRecord{}, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
	}

	return domain.ReconcileJobItemRecord{
		BatchID:      batch.BatchID,
		BeforeStatus: domain.BatchStatusPendingAnchor,
		AfterStatus:  nextStatus,
		AttemptNo:    attemptNo,
		ErrorMessage: &lastError,
		CreatedAt:    s.nowFn().UTC(),
	}, nil
}

func (s *ReconcileService) selectManualTargets(
	ctx context.Context,
	rawBatchIDs []string,
	limit int,
) ([]domain.BatchRecord, int, int, error) {
	if len(rawBatchIDs) == 0 {
		pending, err := s.batchRepo.ListPendingBatches(ctx, limit)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
		}
		return pending, len(pending), 0, nil
	}

	requestedCount := len(rawBatchIDs)
	skippedCount := 0
	scheduled := make([]domain.BatchRecord, 0, len(rawBatchIDs))
	seen := make(map[string]struct{}, len(rawBatchIDs))

	for _, raw := range rawBatchIDs {
		batchID := strings.TrimSpace(raw)
		if batchID == "" {
			skippedCount++
			continue
		}
		if _, ok := seen[batchID]; ok {
			skippedCount++
			continue
		}
		seen[batchID] = struct{}{}

		record, err := s.batchRepo.GetBatchByID(ctx, batchID, domain.TraceModeBlockchain)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				skippedCount++
				continue
			}
			return nil, 0, 0, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
		}
		if record.TraceMode != domain.TraceModeBlockchain || record.Status != domain.BatchStatusPendingAnchor {
			skippedCount++
			continue
		}
		scheduled = append(scheduled, record)
	}

	return scheduled, requestedCount, skippedCount, nil
}

func (s *ReconcileService) failJob(ctx context.Context, jobID string, cause error) error {
	msg := cause.Error()
	if err := s.reconcileRepo.UpdateReconcileJobStatus(ctx, jobID, domain.ReconcileJobStatusFailed, &msg, s.nowFn().UTC()); err != nil {
		return fmt.Errorf("%w: %v (also failed to mark job failed: %v)", ErrServiceUnavailable, cause, err)
	}
	return fmt.Errorf("%w: %v", ErrServiceUnavailable, cause)
}

func resolveAnchorHash(batch domain.BatchRecord) (string, error) {
	if batch.AnchorHash != nil {
		normalized := strings.TrimSpace(*batch.AnchorHash)
		if normalized != "" {
			return normalized, nil
		}
	}

	anchorHash, err := computeAnchorHash(anchorHashPayload{
		BatchID:     batch.BatchID,
		TraceCode:   batch.TraceCode,
		OrchardID:   batch.OrchardID,
		OrchardName: batch.OrchardName,
		PlotID:      batch.PlotID,
		PlotName:    batch.PlotName,
		HarvestedAt: batch.HarvestedAt.UTC(),
		Summary:     batch.Summary,
		Note:        batch.Note,
	})
	if err != nil {
		return "", err
	}
	return anchorHash, nil
}

func isRecoverableChainError(err error) bool {
	return errors.Is(err, evm.ErrNodeUnavailable) ||
		errors.Is(err, evm.ErrTxReverted) ||
		errors.Is(err, evm.ErrContractCall)
}

func normalizeReconcileLimit(limit *int) (int, error) {
	if limit == nil {
		return defaultReconcileLimit, nil
	}
	if *limit < 1 || *limit > maxReconcileLimit {
		return 0, fmt.Errorf("limit must be within 1..%d", maxReconcileLimit)
	}
	return *limit, nil
}

func defaultReconcileJobID() string {
	id := ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rand.Reader)
	return "reconcile_" + strings.ToLower(id.String())
}

func stringPtr(value string) *string {
	return &value
}
