package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/lychee-ripe/gateway/internal/chain/evm"
	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/repository"
	"github.com/oklog/ulid/v2"
)

const (
	unripeThreshold = 0.15
	maxIDRetry      = 3
)

type AnchorClient interface {
	AnchorBatch(ctx context.Context, req evm.AnchorBatchRequest) (domain.AnchorProofRecord, error)
}

type BatchCreateService struct {
	repo         repository.BatchRepository
	anchorClient AnchorClient
	traceMode    domain.TraceMode
	logger       *slog.Logger

	nowFn        func() time.Time
	batchIDFn    func() string
	traceCodeFn  func() string
	anchorHashFn func(anchorHashPayload) (string, error)
}

type BatchSummaryInput struct {
	Total int `json:"total"`
	Green int `json:"green"`
	Half  int `json:"half"`
	Red   int `json:"red"`
	Young int `json:"young"`
}

type BatchCreateInput struct {
	OrchardID     string            `json:"orchard_id"`
	OrchardName   string            `json:"orchard_name"`
	PlotID        string            `json:"plot_id"`
	PlotName      *string           `json:"plot_name"`
	HarvestedAt   string            `json:"harvested_at"`
	Summary       BatchSummaryInput `json:"summary"`
	Note          *string           `json:"note"`
	ConfirmUnripe bool              `json:"confirm_unripe"`
}

type CreateBatchResult struct {
	Batch      domain.BatchRecord
	HTTPStatus int
}

func NewBatchCreateService(
	repo repository.BatchRepository,
	anchorClient AnchorClient,
	traceMode domain.TraceMode,
	logger *slog.Logger,
) *BatchCreateService {
	if logger == nil {
		logger = slog.Default()
	}
	return &BatchCreateService{
		repo:         repo,
		anchorClient: anchorClient,
		traceMode:    traceMode,
		logger:       logger,
		nowFn:        func() time.Time { return time.Now().UTC() },
		batchIDFn:    defaultBatchID,
		traceCodeFn:  defaultTraceCode,
		anchorHashFn: computeAnchorHash,
	}
}

func (s *BatchCreateService) CreateBatch(ctx context.Context, input BatchCreateInput) (CreateBatchResult, error) {
	normalized, err := normalizeCreateInput(input)
	if err != nil {
		return CreateBatchResult{}, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}

	now := s.nowFn().UTC()
	for attempt := 0; attempt < maxIDRetry; attempt++ {
		batchID := s.batchIDFn()
		traceCode := s.traceCodeFn()

		status := domain.BatchStatusStored
		traceMode := s.traceMode
		var anchorHash *string
		if traceMode == "" {
			traceMode = domain.TraceModeDatabase
		}
		if traceMode == domain.TraceModeBlockchain {
			status = domain.BatchStatusPendingAnchor
			computedHash, err := s.anchorHashFn(anchorHashPayload{
				BatchID:     batchID,
				TraceCode:   traceCode,
				OrchardID:   normalized.OrchardID,
				OrchardName: normalized.OrchardName,
				PlotID:      normalized.PlotID,
				PlotName:    normalized.PlotName,
				HarvestedAt: normalized.HarvestedAt.UTC(),
				Summary:     normalized.Summary,
				Note:        normalized.Note,
			})
			if err != nil {
				return CreateBatchResult{}, fmt.Errorf("%w: compute anchor hash: %v", ErrServiceUnavailable, err)
			}
			anchorHash = &computedHash
		}

		createParams := domain.CreateBatchParams{
			BatchID:       batchID,
			TraceCode:     traceCode,
			TraceMode:     traceMode,
			Status:        status,
			OrchardID:     normalized.OrchardID,
			OrchardName:   normalized.OrchardName,
			PlotID:        normalized.PlotID,
			PlotName:      normalized.PlotName,
			HarvestedAt:   normalized.HarvestedAt.UTC(),
			Summary:       normalized.Summary,
			Note:          normalized.Note,
			AnchorHash:    anchorHash,
			ConfirmUnripe: normalized.ConfirmUnripe,
			RetryCount:    0,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		created, err := s.repo.CreateBatch(ctx, createParams)
		if err != nil {
			if errors.Is(err, repository.ErrConflict) {
				if attempt == maxIDRetry-1 {
					return CreateBatchResult{}, fmt.Errorf("%w: duplicated batch_id or trace_code", ErrConflict)
				}
				continue
			}
			if errors.Is(err, repository.ErrInvalidState) {
				return CreateBatchResult{}, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
			}
			return CreateBatchResult{}, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
		}

		if traceMode == domain.TraceModeDatabase {
			return CreateBatchResult{
				Batch:      created,
				HTTPStatus: http.StatusCreated,
			}, nil
		}

		if s.anchorClient == nil {
			return CreateBatchResult{}, fmt.Errorf("%w: blockchain anchor client unavailable", ErrServiceUnavailable)
		}

		proof, err := s.anchorClient.AnchorBatch(ctx, evm.AnchorBatchRequest{
			BatchID:    created.BatchID,
			AnchorHash: *anchorHash,
			Timestamp:  now,
		})
		if err != nil {
			if errors.Is(err, evm.ErrNodeUnavailable) || errors.Is(err, evm.ErrTxReverted) || errors.Is(err, evm.ErrContractCall) {
				retryCount := 1
				return s.degradeBatch(ctx, created, err.Error(), &retryCount, http.StatusAccepted)
			}
			return CreateBatchResult{}, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
		}

		if err := s.repo.AttachAnchorProof(ctx, created.BatchID, domain.BatchStatusPendingAnchor, created.UpdatedAt, proof, s.nowFn()); err != nil {
			if errors.Is(err, repository.ErrConflict) {
				return CreateBatchResult{}, fmt.Errorf("%w: %v", ErrConflict, err)
			}
			return CreateBatchResult{}, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
		}

		anchored, err := s.repo.GetBatchByID(ctx, created.BatchID, traceMode)
		if err != nil {
			return CreateBatchResult{}, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
		}
		return CreateBatchResult{
			Batch:      anchored,
			HTTPStatus: http.StatusCreated,
		}, nil
	}

	return CreateBatchResult{}, fmt.Errorf("%w: unable to allocate unique ids", ErrConflict)
}

func (s *BatchCreateService) GetBatchByID(ctx context.Context, batchID string) (domain.BatchRecord, error) {
	id := strings.TrimSpace(batchID)
	if id == "" {
		return domain.BatchRecord{}, fmt.Errorf("%w: batch_id is required", ErrInvalidRequest)
	}

	record, err := s.repo.GetBatchByID(ctx, id, s.traceMode)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return domain.BatchRecord{}, fmt.Errorf("%w: batch not found", ErrNotFound)
		}
		return domain.BatchRecord{}, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
	}
	return record, nil
}

func (s *BatchCreateService) degradeBatch(
	ctx context.Context,
	record domain.BatchRecord,
	lastError string,
	retryCount *int,
	statusCode int,
) (CreateBatchResult, error) {
	updatedAt := s.nowFn().UTC()
	if err := s.repo.UpdateBatchStatus(ctx, record.BatchID, record.Status, record.UpdatedAt, domain.BatchStatusPendingAnchor, &lastError, retryCount, updatedAt); err != nil {
		return CreateBatchResult{}, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
	}

	updated, err := s.repo.GetBatchByID(ctx, record.BatchID, s.traceMode)
	if err == nil {
		return CreateBatchResult{
			Batch:      updated,
			HTTPStatus: statusCode,
		}, nil
	}

	record.LastError = &lastError
	record.Status = domain.BatchStatusPendingAnchor
	record.UpdatedAt = updatedAt
	if retryCount != nil {
		record.RetryCount = *retryCount
	}
	return CreateBatchResult{
		Batch:      record,
		HTTPStatus: statusCode,
	}, nil
}

type normalizedCreateInput struct {
	OrchardID     string
	OrchardName   string
	PlotID        string
	PlotName      *string
	HarvestedAt   time.Time
	Summary       domain.BatchSummary
	Note          *string
	ConfirmUnripe bool
}

func normalizeCreateInput(input BatchCreateInput) (normalizedCreateInput, error) {
	orchardID := strings.TrimSpace(input.OrchardID)
	if orchardID == "" {
		return normalizedCreateInput{}, errors.New("orchard_id is required")
	}
	orchardName := strings.TrimSpace(input.OrchardName)
	if orchardName == "" {
		return normalizedCreateInput{}, errors.New("orchard_name is required")
	}
	plotID := strings.TrimSpace(input.PlotID)
	if plotID == "" {
		return normalizedCreateInput{}, errors.New("plot_id is required")
	}

	harvestedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(input.HarvestedAt))
	if err != nil {
		return normalizedCreateInput{}, errors.New("harvested_at must be RFC3339 date-time")
	}

	if input.Summary.Total <= 0 {
		return normalizedCreateInput{}, errors.New("summary.total must be > 0")
	}
	if input.Summary.Green < 0 || input.Summary.Half < 0 || input.Summary.Red < 0 || input.Summary.Young < 0 {
		return normalizedCreateInput{}, errors.New("summary values must be >= 0")
	}
	sum := input.Summary.Green + input.Summary.Half + input.Summary.Red + input.Summary.Young
	if sum != input.Summary.Total {
		return normalizedCreateInput{}, errors.New("summary counts must equal total")
	}

	unripeCount := input.Summary.Green + input.Summary.Young
	unripeRatioRaw := float64(unripeCount) / float64(input.Summary.Total)
	if unripeRatioRaw > unripeThreshold && !input.ConfirmUnripe {
		return normalizedCreateInput{}, errors.New("confirm_unripe must be true when unripe_ratio > 0.15")
	}

	summary := domain.BatchSummary{
		Total:          input.Summary.Total,
		Green:          input.Summary.Green,
		Half:           input.Summary.Half,
		Red:            input.Summary.Red,
		Young:          input.Summary.Young,
		UnripeCount:    unripeCount,
		UnripeRatio:    roundTo(unripeRatioRaw, unripeRatioDigits),
		UnripeHandling: domain.UnripeHandlingSortedOut,
	}

	return normalizedCreateInput{
		OrchardID:     orchardID,
		OrchardName:   orchardName,
		PlotID:        plotID,
		PlotName:      normalizeOptionalString(input.PlotName),
		HarvestedAt:   harvestedAt.UTC(),
		Summary:       summary,
		Note:          normalizeOptionalString(input.Note),
		ConfirmUnripe: input.ConfirmUnripe,
	}, nil
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func defaultBatchID() string {
	id := ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rand.Reader)
	return "batch_" + strings.ToLower(id.String())
}

const traceAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func defaultTraceCode() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		id := ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rand.Reader)
		raw := strings.ToUpper(id.String())
		return "TRC-" + raw[:4] + "-" + raw[4:8]
	}
	for i := range buf {
		buf[i] = traceAlphabet[int(buf[i])%len(traceAlphabet)]
	}
	return "TRC-" + string(buf[:4]) + "-" + string(buf[4:])
}
