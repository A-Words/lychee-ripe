package gorm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

var (
	_ repository.BatchRepository          = (*Repository)(nil)
	_ repository.ReconcileRepository      = (*Repository)(nil)
	_ repository.AuditRepository          = (*Repository)(nil)
	_ repository.DashboardQueryRepository = (*Repository)(nil)
)

func (r *Repository) CreateBatch(ctx context.Context, params domain.CreateBatchParams) (domain.BatchRecord, error) {
	if !isValidTraceMode(params.TraceMode) || !isValidBatchStatus(params.Status) {
		return domain.BatchRecord{}, repository.ErrInvalidState
	}

	record := batchModelFromCreateParams(params)
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return domain.BatchRecord{}, mapGormErr(err)
	}
	return r.GetBatchByID(ctx, params.BatchID, params.TraceMode)
}

func (r *Repository) GetBatchByID(ctx context.Context, batchID string, traceMode domain.TraceMode) (domain.BatchRecord, error) {
	return r.getBatchBy(ctx, "batch_id = ?", batchID, traceMode)
}

func (r *Repository) GetBatchByTraceCode(ctx context.Context, traceCode string, traceMode domain.TraceMode) (domain.BatchRecord, error) {
	return r.getBatchBy(ctx, "trace_code = ?", traceCode, traceMode)
}

func (r *Repository) UpdateBatchStatus(
	ctx context.Context,
	batchID string,
	status domain.BatchStatus,
	lastError *string,
	retryCount *int,
	updatedAt time.Time,
) error {
	if !isValidBatchStatus(status) {
		return repository.ErrInvalidState
	}

	updates := map[string]any{
		"status":     string(status),
		"updated_at": normalizeTime(updatedAt),
	}
	if lastError != nil {
		updates["last_error"] = *lastError
	}
	if retryCount != nil {
		updates["retry_count"] = *retryCount
	}

	res := r.db.WithContext(ctx).Model(&BatchModel{}).Where("batch_id = ?", batchID).Updates(updates)
	if res.Error != nil {
		return mapGormErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *Repository) AttachAnchorProof(
	ctx context.Context,
	batchID string,
	proof domain.AnchorProofRecord,
	updatedAt time.Time,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		proofModel := AnchorProofModel{
			BatchID:         batchID,
			TxHash:          proof.TxHash,
			BlockNumber:     proof.BlockNumber,
			ChainID:         proof.ChainID,
			ContractAddress: proof.ContractAddress,
			AnchorHash:      proof.AnchorHash,
			AnchoredAt:      normalizeTime(proof.AnchoredAt),
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "batch_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"tx_hash":          proofModel.TxHash,
				"block_number":     proofModel.BlockNumber,
				"chain_id":         proofModel.ChainID,
				"contract_address": proofModel.ContractAddress,
				"anchor_hash":      proofModel.AnchorHash,
				"anchored_at":      proofModel.AnchoredAt,
			}),
		}).Create(&proofModel).Error; err != nil {
			return mapGormErr(err)
		}

		res := tx.Model(&BatchModel{}).
			Where("batch_id = ?", batchID).
			Updates(map[string]any{
				"status":      string(domain.BatchStatusAnchored),
				"anchor_hash": proof.AnchorHash,
				"updated_at":  normalizeTime(updatedAt),
			})
		if res.Error != nil {
			return mapGormErr(res.Error)
		}
		if res.RowsAffected == 0 {
			return repository.ErrNotFound
		}
		return nil
	})
}

func (r *Repository) ListPendingBatches(ctx context.Context, limit int) ([]domain.BatchRecord, error) {
	if limit <= 0 {
		limit = 100
	}

	var models []BatchModel
	if err := r.db.WithContext(ctx).
		Preload("AnchorProof").
		Where("trace_mode = ? AND status = ?", string(domain.TraceModeBlockchain), string(domain.BatchStatusPendingAnchor)).
		Order("created_at DESC").
		Limit(limit).
		Find(&models).Error; err != nil {
		return nil, mapGormErr(err)
	}

	out := make([]domain.BatchRecord, 0, len(models))
	for _, m := range models {
		out = append(out, batchModelToDomain(m))
	}
	return out, nil
}

func (r *Repository) CreateReconcileJob(ctx context.Context, params domain.CreateReconcileJobParams) (domain.ReconcileJobRecord, error) {
	if !isValidReconcileTriggerType(params.TriggerType) {
		return domain.ReconcileJobRecord{}, repository.ErrInvalidState
	}

	status := params.Status
	if status == "" {
		status = domain.ReconcileJobStatusAccepted
	}
	if !isValidReconcileJobStatus(status) {
		return domain.ReconcileJobRecord{}, repository.ErrInvalidState
	}

	record := ReconcileJobModel{
		JobID:          params.JobID,
		TriggerType:    string(params.TriggerType),
		Status:         string(status),
		RequestedCount: params.RequestedCount,
		ScheduledCount: params.ScheduledCount,
		SkippedCount:   params.SkippedCount,
		ErrorMessage:   params.ErrorMessage,
		CreatedAt:      normalizeTime(params.CreatedAt),
		UpdatedAt:      normalizeTime(params.UpdatedAt),
	}
	if params.UpdatedAt.IsZero() {
		record.UpdatedAt = record.CreatedAt
	}

	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return domain.ReconcileJobRecord{}, mapGormErr(err)
	}

	return domain.ReconcileJobRecord{
		JobID:          record.JobID,
		TriggerType:    params.TriggerType,
		Status:         status,
		RequestedCount: record.RequestedCount,
		ScheduledCount: record.ScheduledCount,
		SkippedCount:   record.SkippedCount,
		ErrorMessage:   record.ErrorMessage,
		CreatedAt:      record.CreatedAt,
		UpdatedAt:      record.UpdatedAt,
	}, nil
}

func (r *Repository) AddReconcileJobItems(ctx context.Context, jobID string, items []domain.ReconcileJobItemRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if !isValidBatchStatus(item.BeforeStatus) || !isValidBatchStatus(item.AfterStatus) {
				return repository.ErrInvalidState
			}
			record := ReconcileJobItemModel{
				JobID:        jobID,
				BatchID:      item.BatchID,
				BeforeStatus: string(item.BeforeStatus),
				AfterStatus:  string(item.AfterStatus),
				AttemptNo:    item.AttemptNo,
				ErrorMessage: item.ErrorMessage,
				CreatedAt:    normalizeTime(item.CreatedAt),
			}
			if err := tx.Create(&record).Error; err != nil {
				return mapGormErr(err)
			}
		}
		return nil
	})
}

func (r *Repository) UpdateReconcileJobStatus(
	ctx context.Context,
	jobID string,
	status domain.ReconcileJobStatus,
	errMsg *string,
	updatedAt time.Time,
) error {
	if !isValidReconcileJobStatus(status) {
		return repository.ErrInvalidState
	}

	updates := map[string]any{
		"status":     string(status),
		"updated_at": normalizeTime(updatedAt),
	}
	if errMsg != nil {
		updates["error_message"] = *errMsg
	}

	res := r.db.WithContext(ctx).Model(&ReconcileJobModel{}).Where("job_id = ?", jobID).Updates(updates)
	if res.Error != nil {
		return mapGormErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *Repository) ListReconcileStats(ctx context.Context) (domain.ReconcileStats, error) {
	type row struct {
		PendingCount int64          `gorm:"column:pending_count"`
		RetriedTotal int64          `gorm:"column:retried_total"`
		FailedTotal  int64          `gorm:"column:failed_total"`
		LastRaw      sql.NullString `gorm:"column:last_reconcile_at"`
	}
	var out row
	if err := r.db.WithContext(ctx).Raw(
		`SELECT
			(SELECT COUNT(1) FROM batches WHERE trace_mode = 'blockchain' AND status = 'pending_anchor') AS pending_count,
			COALESCE((SELECT SUM(retry_count) FROM batches WHERE trace_mode = 'blockchain'), 0) AS retried_total,
			(SELECT COUNT(1) FROM batches WHERE trace_mode = 'blockchain' AND status = 'anchor_failed') AS failed_total,
			(SELECT MAX(updated_at) FROM reconcile_jobs) AS last_reconcile_at`,
	).Scan(&out).Error; err != nil {
		return domain.ReconcileStats{}, mapGormErr(err)
	}
	var last *time.Time
	if out.LastRaw.Valid {
		parsed, err := parseDBTime(out.LastRaw.String)
		if err != nil {
			return domain.ReconcileStats{}, err
		}
		last = &parsed
	}
	return domain.ReconcileStats{
		PendingCount:    out.PendingCount,
		RetriedTotal:    out.RetriedTotal,
		FailedTotal:     out.FailedTotal,
		LastReconcileAt: last,
	}, nil
}

func (r *Repository) AppendAuditLog(ctx context.Context, log domain.AuditLogRecord) error {
	record := AuditLogModel{
		EventType:   log.EventType,
		EntityType:  log.EntityType,
		EntityID:    log.EntityID,
		Status:      log.Status,
		Message:     log.Message,
		RequestID:   log.RequestID,
		PayloadJSON: log.PayloadJSON,
		CreatedAt:   normalizeTime(log.CreatedAt),
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return mapGormErr(err)
	}
	return nil
}

func (r *Repository) CountBatches(ctx context.Context, traceMode domain.TraceMode) (int64, error) {
	if !isValidTraceMode(traceMode) {
		return 0, repository.ErrInvalidState
	}

	var count int64
	if err := r.db.WithContext(ctx).Model(&BatchModel{}).Where("trace_mode = ?", string(traceMode)).Count(&count).Error; err != nil {
		return 0, mapGormErr(err)
	}
	return count, nil
}

func (r *Repository) CountByStatus(ctx context.Context, traceMode domain.TraceMode) (domain.StatusDistribution, error) {
	if !isValidTraceMode(traceMode) {
		return domain.StatusDistribution{}, repository.ErrInvalidState
	}

	type row struct {
		Stored        int64 `gorm:"column:stored"`
		Anchored      int64 `gorm:"column:anchored"`
		PendingAnchor int64 `gorm:"column:pending_anchor"`
		AnchorFailed  int64 `gorm:"column:anchor_failed"`
	}
	var out row
	if err := r.db.WithContext(ctx).Raw(
		`SELECT
			COALESCE(SUM(CASE WHEN status = 'stored' THEN 1 ELSE 0 END), 0) AS stored,
			COALESCE(SUM(CASE WHEN status = 'anchored' THEN 1 ELSE 0 END), 0) AS anchored,
			COALESCE(SUM(CASE WHEN status = 'pending_anchor' THEN 1 ELSE 0 END), 0) AS pending_anchor,
			COALESCE(SUM(CASE WHEN status = 'anchor_failed' THEN 1 ELSE 0 END), 0) AS anchor_failed
		FROM batches
		WHERE trace_mode = ?`,
		string(traceMode),
	).Scan(&out).Error; err != nil {
		return domain.StatusDistribution{}, mapGormErr(err)
	}
	return domain.StatusDistribution{
		Stored:        out.Stored,
		Anchored:      out.Anchored,
		PendingAnchor: out.PendingAnchor,
		AnchorFailed:  out.AnchorFailed,
	}, nil
}

func (r *Repository) SumRipeness(ctx context.Context, traceMode domain.TraceMode) (domain.RipenessDistribution, error) {
	if !isValidTraceMode(traceMode) {
		return domain.RipenessDistribution{}, repository.ErrInvalidState
	}

	type row struct {
		Green int64 `gorm:"column:green"`
		Half  int64 `gorm:"column:half"`
		Red   int64 `gorm:"column:red"`
		Young int64 `gorm:"column:young"`
	}
	var out row
	if err := r.db.WithContext(ctx).Raw(
		`SELECT
			COALESCE(SUM(green), 0) AS green,
			COALESCE(SUM(half), 0) AS half,
			COALESCE(SUM(red), 0) AS red,
			COALESCE(SUM(young), 0) AS young
		FROM batches
		WHERE trace_mode = ?`,
		string(traceMode),
	).Scan(&out).Error; err != nil {
		return domain.RipenessDistribution{}, mapGormErr(err)
	}
	return domain.RipenessDistribution{
		Green: out.Green,
		Half:  out.Half,
		Red:   out.Red,
		Young: out.Young,
	}, nil
}

func (r *Repository) CountUnripeBatches(ctx context.Context, traceMode domain.TraceMode, threshold float64) (int64, float64, error) {
	if !isValidTraceMode(traceMode) {
		return 0, 0, repository.ErrInvalidState
	}

	type row struct {
		Total int64 `gorm:"column:total"`
		Cnt   int64 `gorm:"column:cnt"`
	}
	var out row
	if err := r.db.WithContext(ctx).Raw(
		`SELECT
			COUNT(1) AS total,
			COALESCE(SUM(CASE WHEN unripe_ratio > ? THEN 1 ELSE 0 END), 0) AS cnt
		FROM batches
		WHERE trace_mode = ?`,
		threshold,
		string(traceMode),
	).Scan(&out).Error; err != nil {
		return 0, 0, mapGormErr(err)
	}
	if out.Total == 0 {
		return 0, 0, nil
	}
	return out.Cnt, float64(out.Cnt) / float64(out.Total), nil
}

func (r *Repository) ListRecentAnchors(ctx context.Context, traceMode domain.TraceMode, limit int) ([]domain.RecentAnchorRecord, error) {
	if !isValidTraceMode(traceMode) {
		return nil, repository.ErrInvalidState
	}
	if limit <= 0 {
		limit = 20
	}

	type row struct {
		BatchID    string     `gorm:"column:batch_id"`
		TraceCode  string     `gorm:"column:trace_code"`
		TraceMode  string     `gorm:"column:trace_mode"`
		Status     string     `gorm:"column:status"`
		TxHash     *string    `gorm:"column:tx_hash"`
		AnchoredAt *time.Time `gorm:"column:anchored_at"`
		CreatedAt  time.Time  `gorm:"column:created_at"`
	}
	var rows []row
	if err := r.db.WithContext(ctx).Raw(
		`SELECT
			b.batch_id, b.trace_code, b.trace_mode, b.status, ap.tx_hash, ap.anchored_at, b.created_at
		FROM batches b
		LEFT JOIN anchor_proofs ap ON ap.batch_id = b.batch_id
		WHERE b.trace_mode = ? AND b.status = 'anchored'
		ORDER BY b.created_at DESC
		LIMIT ?`,
		string(traceMode),
		limit,
	).Scan(&rows).Error; err != nil {
		return nil, mapGormErr(err)
	}

	out := make([]domain.RecentAnchorRecord, 0, len(rows))
	for _, item := range rows {
		out = append(out, domain.RecentAnchorRecord{
			BatchID:    item.BatchID,
			TraceCode:  item.TraceCode,
			TraceMode:  normalizeTraceMode(item.TraceMode),
			Status:     domain.BatchStatus(item.Status),
			TxHash:     item.TxHash,
			AnchoredAt: item.AnchoredAt,
			CreatedAt:  item.CreatedAt.UTC(),
		})
	}
	return out, nil
}

func (r *Repository) getBatchBy(ctx context.Context, where string, value any, traceMode domain.TraceMode) (domain.BatchRecord, error) {
	if !isValidTraceMode(traceMode) {
		return domain.BatchRecord{}, repository.ErrInvalidState
	}

	var model BatchModel
	if err := r.db.WithContext(ctx).
		Preload("AnchorProof").
		Where(where, value).
		Where("trace_mode = ?", string(traceMode)).
		First(&model).Error; err != nil {
		return domain.BatchRecord{}, mapGormErr(err)
	}
	return batchModelToDomain(model), nil
}

func batchModelFromCreateParams(params domain.CreateBatchParams) BatchModel {
	createdAt := normalizeTime(params.CreatedAt)
	updatedAt := normalizeTime(params.UpdatedAt)
	if params.UpdatedAt.IsZero() {
		updatedAt = createdAt
	}

	return BatchModel{
		BatchID:        params.BatchID,
		TraceCode:      params.TraceCode,
		TraceMode:      string(params.TraceMode),
		Status:         string(params.Status),
		OrchardID:      params.OrchardID,
		OrchardName:    params.OrchardName,
		PlotID:         params.PlotID,
		PlotName:       params.PlotName,
		HarvestedAt:    normalizeTime(params.HarvestedAt),
		Total:          params.Summary.Total,
		Green:          params.Summary.Green,
		Half:           params.Summary.Half,
		Red:            params.Summary.Red,
		Young:          params.Summary.Young,
		UnripeCount:    params.Summary.UnripeCount,
		UnripeRatio:    params.Summary.UnripeRatio,
		UnripeHandling: string(params.Summary.UnripeHandling),
		Note:           params.Note,
		AnchorHash:     params.AnchorHash,
		ConfirmUnripe:  params.ConfirmUnripe,
		RetryCount:     params.RetryCount,
		LastError:      params.LastError,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}
}

func batchModelToDomain(model BatchModel) domain.BatchRecord {
	record := domain.BatchRecord{
		BatchID:       model.BatchID,
		TraceCode:     model.TraceCode,
		TraceMode:     normalizeTraceMode(model.TraceMode),
		Status:        domain.BatchStatus(model.Status),
		OrchardID:     model.OrchardID,
		OrchardName:   model.OrchardName,
		PlotID:        model.PlotID,
		PlotName:      model.PlotName,
		HarvestedAt:   model.HarvestedAt.UTC(),
		Note:          model.Note,
		AnchorHash:    model.AnchorHash,
		ConfirmUnripe: model.ConfirmUnripe,
		RetryCount:    model.RetryCount,
		LastError:     model.LastError,
		CreatedAt:     model.CreatedAt.UTC(),
		UpdatedAt:     model.UpdatedAt.UTC(),
		Summary: domain.BatchSummary{
			Total:          model.Total,
			Green:          model.Green,
			Half:           model.Half,
			Red:            model.Red,
			Young:          model.Young,
			UnripeCount:    model.UnripeCount,
			UnripeRatio:    model.UnripeRatio,
			UnripeHandling: domain.UnripeHandling(model.UnripeHandling),
		},
	}

	if model.AnchorProof != nil {
		record.AnchorProof = &domain.AnchorProofRecord{
			TxHash:          model.AnchorProof.TxHash,
			BlockNumber:     model.AnchorProof.BlockNumber,
			ChainID:         model.AnchorProof.ChainID,
			ContractAddress: model.AnchorProof.ContractAddress,
			AnchorHash:      model.AnchorProof.AnchorHash,
			AnchoredAt:      model.AnchorProof.AnchoredAt.UTC(),
		}
	}
	return record
}

func normalizeTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t.UTC()
}

func parseDBTime(raw string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", raw); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02 15:04:05.999999999", raw); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", raw); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02 15:04:05", raw); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("%w: unsupported time format %q", repository.ErrDBUnavailable, raw)
}

func mapGormErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, repository.ErrNotFound) ||
		errors.Is(err, repository.ErrConflict) ||
		errors.Is(err, repository.ErrInvalidState) ||
		errors.Is(err, repository.ErrDBUnavailable) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return repository.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return fmt.Errorf("%w: %v", repository.ErrConflict, err)
		case "23503":
			return fmt.Errorf("%w: %v", repository.ErrNotFound, err)
		case "23514":
			return fmt.Errorf("%w: %v", repository.ErrInvalidState, err)
		default:
			return fmt.Errorf("%w: %v", repository.ErrDBUnavailable, err)
		}
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "unique constraint failed"),
		strings.Contains(msg, "duplicate key value violates unique constraint"):
		return fmt.Errorf("%w: %v", repository.ErrConflict, err)
	case strings.Contains(msg, "check constraint failed"),
		strings.Contains(msg, "violates check constraint"):
		return fmt.Errorf("%w: %v", repository.ErrInvalidState, err)
	case strings.Contains(msg, "foreign key constraint failed"),
		strings.Contains(msg, "violates foreign key constraint"):
		return fmt.Errorf("%w: %v", repository.ErrNotFound, err)
	default:
		return fmt.Errorf("%w: %v", repository.ErrDBUnavailable, err)
	}
}

func isValidBatchStatus(status domain.BatchStatus) bool {
	switch status {
	case domain.BatchStatusStored, domain.BatchStatusPendingAnchor, domain.BatchStatusAnchored, domain.BatchStatusAnchorFailed:
		return true
	default:
		return false
	}
}

func isValidTraceMode(mode domain.TraceMode) bool {
	switch mode {
	case domain.TraceModeDatabase, domain.TraceModeBlockchain:
		return true
	default:
		return false
	}
}

func normalizeTraceMode(raw string) domain.TraceMode {
	mode := domain.TraceMode(strings.TrimSpace(raw))
	if isValidTraceMode(mode) {
		return mode
	}
	return domain.TraceModeBlockchain
}

func isValidReconcileTriggerType(triggerType domain.ReconcileTriggerType) bool {
	switch triggerType {
	case domain.ReconcileTriggerManual, domain.ReconcileTriggerAuto:
		return true
	default:
		return false
	}
}

func isValidReconcileJobStatus(status domain.ReconcileJobStatus) bool {
	switch status {
	case domain.ReconcileJobStatusAccepted, domain.ReconcileJobStatusRunning, domain.ReconcileJobStatusCompleted, domain.ReconcileJobStatusFailed:
		return true
	default:
		return false
	}
}
