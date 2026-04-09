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

type UserRepository interface {
	CountUsers(ctx context.Context) (int64, error)
	ResolvePrincipal(ctx context.Context, identity domain.IdentityClaims, mode domain.AuthMode, now time.Time) (domain.Principal, error)
	GetPrincipalByID(ctx context.Context, userID string) (domain.UserRecord, error)
	GetUserByOIDCSubject(ctx context.Context, subject string) (domain.UserRecord, error)
	ListUsers(ctx context.Context) ([]domain.UserRecord, error)
	CreateUser(ctx context.Context, user domain.UserRecord) (domain.UserRecord, error)
	UpdateUser(ctx context.Context, user domain.UserRecord) (domain.UserRecord, error)
}

type WebSessionRepository interface {
	CreateWebAuthState(ctx context.Context, state domain.WebAuthStateRecord) (domain.WebAuthStateRecord, error)
	ConsumeWebAuthState(ctx context.Context, state string, now time.Time) (domain.WebAuthStateRecord, error)
	CreateWebSession(ctx context.Context, session domain.WebSessionRecord) (domain.WebSessionRecord, error)
	GetWebSession(ctx context.Context, sessionIDHash string, now time.Time) (domain.WebSessionRecord, error)
	DeleteWebSession(ctx context.Context, sessionIDHash string) error
}

type OrchardRepository interface {
	ListOrchards(ctx context.Context, includeArchived bool) ([]domain.OrchardRecord, error)
	CreateOrchard(ctx context.Context, orchard domain.OrchardRecord) (domain.OrchardRecord, error)
	UpdateOrchard(ctx context.Context, orchard domain.OrchardRecord) (domain.OrchardRecord, error)
	ArchiveOrchard(ctx context.Context, orchard domain.OrchardRecord) (domain.OrchardRecord, error)
	GetOrchard(ctx context.Context, orchardID string) (domain.OrchardRecord, error)
}

type PlotRepository interface {
	ListPlots(ctx context.Context, orchardID string, includeArchived bool) ([]domain.PlotRecord, error)
	CreatePlot(ctx context.Context, plot domain.PlotRecord) (domain.PlotRecord, error)
	CreatePlotGuarded(ctx context.Context, plot domain.PlotRecord) (domain.PlotRecord, error)
	UpdatePlot(ctx context.Context, plot domain.PlotRecord) (domain.PlotRecord, error)
	UpdatePlotGuarded(ctx context.Context, plot domain.PlotRecord) (domain.PlotRecord, error)
	GetPlot(ctx context.Context, plotID string) (domain.PlotRecord, error)
}

type SeedRepository interface {
	CountOrchards(ctx context.Context) (int64, error)
	CreateOrchardIfNotExists(ctx context.Context, orchard domain.OrchardRecord) error
	CreatePlotIfNotExists(ctx context.Context, plot domain.PlotRecord) error
}
