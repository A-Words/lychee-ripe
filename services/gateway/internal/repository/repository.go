package repository

import (
	"context"
	"errors"
	"time"

	"github.com/lychee-ripe/gateway/internal/domain"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrConflict      = errors.New("conflict")
	ErrInvalidState  = errors.New("invalid state")
	ErrDBUnavailable = errors.New("db unavailable")
)

type BatchRepository interface {
	CreateBatch(ctx context.Context, params domain.CreateBatchParams) (domain.BatchRecord, error)
	GetBatchByID(ctx context.Context, batchID string, traceMode domain.TraceMode) (domain.BatchRecord, error)
	GetBatchByTraceCode(ctx context.Context, traceCode string, traceMode domain.TraceMode) (domain.BatchRecord, error)
	UpdateBatchStatus(ctx context.Context, batchID string, status domain.BatchStatus, lastError *string, retryCount *int, updatedAt time.Time) error
	AttachAnchorProof(ctx context.Context, batchID string, proof domain.AnchorProofRecord, updatedAt time.Time) error
	ListPendingBatches(ctx context.Context, limit int) ([]domain.BatchRecord, error)
}

type ReconcileRepository interface {
	CreateReconcileJob(ctx context.Context, params domain.CreateReconcileJobParams) (domain.ReconcileJobRecord, error)
	AddReconcileJobItems(ctx context.Context, jobID string, items []domain.ReconcileJobItemRecord) error
	UpdateReconcileJobStatus(ctx context.Context, jobID string, status domain.ReconcileJobStatus, errMsg *string, updatedAt time.Time) error
	ListReconcileStats(ctx context.Context) (domain.ReconcileStats, error)
}

type AuditRepository interface {
	AppendAuditLog(ctx context.Context, log domain.AuditLogRecord) error
}

type DashboardQueryRepository interface {
	CountBatches(ctx context.Context, traceMode domain.TraceMode) (int64, error)
	CountByStatus(ctx context.Context, traceMode domain.TraceMode) (domain.StatusDistribution, error)
	SumRipeness(ctx context.Context, traceMode domain.TraceMode) (domain.RipenessDistribution, error)
	CountUnripeBatches(ctx context.Context, traceMode domain.TraceMode, threshold float64) (count int64, ratio float64, err error)
	ListRecentAnchors(ctx context.Context, traceMode domain.TraceMode, limit int) ([]domain.RecentAnchorRecord, error)
}
