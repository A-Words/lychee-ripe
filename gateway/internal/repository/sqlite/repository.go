package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/repository"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

var (
	_ repository.BatchRepository          = (*Repository)(nil)
	_ repository.ReconcileRepository      = (*Repository)(nil)
	_ repository.AuditRepository          = (*Repository)(nil)
	_ repository.DashboardQueryRepository = (*Repository)(nil)
)

func (r *Repository) CreateBatch(ctx context.Context, params domain.CreateBatchParams) (domain.BatchRecord, error) {
	if !isValidBatchStatus(params.Status) {
		return domain.BatchRecord{}, repository.ErrInvalidState
	}

	createdAt := normalizeTime(params.CreatedAt)
	updatedAt := normalizeTime(params.UpdatedAt)
	if params.UpdatedAt.IsZero() {
		updatedAt = createdAt
	}

	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO batches (
			batch_id, trace_code, status, orchard_id, orchard_name, plot_id, plot_name,
			harvested_at, total, green, half, red, young, unripe_count, unripe_ratio, unripe_handling,
			note, anchor_hash, confirm_unripe, retry_count, last_error, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		params.BatchID,
		params.TraceCode,
		string(params.Status),
		params.OrchardID,
		params.OrchardName,
		params.PlotID,
		toNullString(params.PlotName),
		normalizeTime(params.HarvestedAt).Format(time.RFC3339Nano),
		params.Summary.Total,
		params.Summary.Green,
		params.Summary.Half,
		params.Summary.Red,
		params.Summary.Young,
		params.Summary.UnripeCount,
		params.Summary.UnripeRatio,
		string(params.Summary.UnripeHandling),
		toNullString(params.Note),
		toNullString(params.AnchorHash),
		boolToInt(params.ConfirmUnripe),
		params.RetryCount,
		toNullString(params.LastError),
		createdAt.Format(time.RFC3339Nano),
		updatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return domain.BatchRecord{}, mapDBErr(err)
	}

	return r.GetBatchByID(ctx, params.BatchID)
}

func (r *Repository) GetBatchByID(ctx context.Context, batchID string) (domain.BatchRecord, error) {
	return r.getBatchBy(ctx, "b.batch_id = ?", batchID)
}

func (r *Repository) GetBatchByTraceCode(ctx context.Context, traceCode string) (domain.BatchRecord, error) {
	return r.getBatchBy(ctx, "b.trace_code = ?", traceCode)
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

	setLastError := 0
	if lastError != nil {
		setLastError = 1
	}
	setRetryCount := 0
	if retryCount != nil {
		setRetryCount = 1
	}

	res, err := r.db.ExecContext(
		ctx,
		`UPDATE batches
		 SET status = ?,
		     last_error = CASE WHEN ? = 1 THEN ? ELSE last_error END,
		     retry_count = CASE WHEN ? = 1 THEN ? ELSE retry_count END,
		     updated_at = ?
		 WHERE batch_id = ?`,
		string(status),
		setLastError,
		toNullString(lastError),
		setRetryCount,
		retryCountValue(retryCount),
		normalizeTime(updatedAt).Format(time.RFC3339Nano),
		batchID,
	)
	if err != nil {
		return mapDBErr(err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: rows affected: %v", repository.ErrDBUnavailable, err)
	}
	if rows == 0 {
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
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: begin tx: %v", repository.ErrDBUnavailable, err)
	}

	anchoredAt := normalizeTime(proof.AnchoredAt)
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO anchor_proofs (
			batch_id, tx_hash, block_number, chain_id, contract_address, anchor_hash, anchored_at
		 ) VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(batch_id) DO UPDATE SET
		   tx_hash = excluded.tx_hash,
		   block_number = excluded.block_number,
		   chain_id = excluded.chain_id,
		   contract_address = excluded.contract_address,
		   anchor_hash = excluded.anchor_hash,
		   anchored_at = excluded.anchored_at`,
		batchID,
		proof.TxHash,
		proof.BlockNumber,
		proof.ChainID,
		proof.ContractAddress,
		proof.AnchorHash,
		anchoredAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		_ = tx.Rollback()
		return mapDBErr(err)
	}

	res, err := tx.ExecContext(
		ctx,
		`UPDATE batches
		 SET status = ?, anchor_hash = ?, updated_at = ?
		 WHERE batch_id = ?`,
		string(domain.BatchStatusAnchored),
		proof.AnchorHash,
		normalizeTime(updatedAt).Format(time.RFC3339Nano),
		batchID,
	)
	if err != nil {
		_ = tx.Rollback()
		return mapDBErr(err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("%w: rows affected: %v", repository.ErrDBUnavailable, err)
	}
	if rows == 0 {
		_ = tx.Rollback()
		return repository.ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: commit tx: %v", repository.ErrDBUnavailable, err)
	}
	return nil
}

func (r *Repository) ListPendingBatches(ctx context.Context, limit int) ([]domain.BatchRecord, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT
			b.batch_id, b.trace_code, b.status, b.orchard_id, b.orchard_name, b.plot_id, b.plot_name,
			b.harvested_at, b.total, b.green, b.half, b.red, b.young, b.unripe_count, b.unripe_ratio,
			b.unripe_handling, b.note, b.anchor_hash, b.confirm_unripe, b.retry_count, b.last_error,
			b.created_at, b.updated_at,
			ap.tx_hash, ap.block_number, ap.chain_id, ap.contract_address, ap.anchor_hash, ap.anchored_at
		FROM batches b
		LEFT JOIN anchor_proofs ap ON ap.batch_id = b.batch_id
		WHERE b.status = ?
		ORDER BY b.created_at DESC
		LIMIT ?`,
		string(domain.BatchStatusPendingAnchor),
		limit,
	)
	if err != nil {
		return nil, mapDBErr(err)
	}
	defer rows.Close()

	out := make([]domain.BatchRecord, 0)
	for rows.Next() {
		record, err := scanBatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDBErr(err)
	}
	return out, nil
}

func (r *Repository) CreateReconcileJob(
	ctx context.Context,
	params domain.CreateReconcileJobParams,
) (domain.ReconcileJobRecord, error) {
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

	createdAt := normalizeTime(params.CreatedAt)
	updatedAt := normalizeTime(params.UpdatedAt)
	if params.UpdatedAt.IsZero() {
		updatedAt = createdAt
	}

	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO reconcile_jobs (
			job_id, trigger_type, status, requested_count, scheduled_count, skipped_count,
			error_message, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		params.JobID,
		string(params.TriggerType),
		string(status),
		params.RequestedCount,
		params.ScheduledCount,
		params.SkippedCount,
		toNullString(params.ErrorMessage),
		createdAt.Format(time.RFC3339Nano),
		updatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return domain.ReconcileJobRecord{}, mapDBErr(err)
	}

	return domain.ReconcileJobRecord{
		JobID:          params.JobID,
		TriggerType:    params.TriggerType,
		Status:         status,
		RequestedCount: params.RequestedCount,
		ScheduledCount: params.ScheduledCount,
		SkippedCount:   params.SkippedCount,
		ErrorMessage:   params.ErrorMessage,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}, nil
}

func (r *Repository) AddReconcileJobItems(
	ctx context.Context,
	jobID string,
	items []domain.ReconcileJobItemRecord,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: begin tx: %v", repository.ErrDBUnavailable, err)
	}

	for _, item := range items {
		if !isValidBatchStatus(item.BeforeStatus) || !isValidBatchStatus(item.AfterStatus) {
			_ = tx.Rollback()
			return repository.ErrInvalidState
		}

		_, err := tx.ExecContext(
			ctx,
			`INSERT INTO reconcile_job_items (
				job_id, batch_id, before_status, after_status, attempt_no, error_message, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			jobID,
			item.BatchID,
			string(item.BeforeStatus),
			string(item.AfterStatus),
			item.AttemptNo,
			toNullString(item.ErrorMessage),
			normalizeTime(item.CreatedAt).Format(time.RFC3339Nano),
		)
		if err != nil {
			_ = tx.Rollback()
			return mapDBErr(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: commit tx: %v", repository.ErrDBUnavailable, err)
	}
	return nil
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

	setErrMsg := 0
	if errMsg != nil {
		setErrMsg = 1
	}

	res, err := r.db.ExecContext(
		ctx,
		`UPDATE reconcile_jobs
		 SET status = ?,
		     error_message = CASE WHEN ? = 1 THEN ? ELSE error_message END,
		     updated_at = ?
		 WHERE job_id = ?`,
		string(status),
		setErrMsg,
		toNullString(errMsg),
		normalizeTime(updatedAt).Format(time.RFC3339Nano),
		jobID,
	)
	if err != nil {
		return mapDBErr(err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: rows affected: %v", repository.ErrDBUnavailable, err)
	}
	if rows == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *Repository) ListReconcileStats(ctx context.Context) (domain.ReconcileStats, error) {
	var (
		pendingCount int64
		retriedTotal int64
		failedTotal  int64
		lastUpdate   sql.NullString
	)

	err := r.db.QueryRowContext(
		ctx,
		`SELECT
			(SELECT COUNT(1) FROM batches WHERE status = 'pending_anchor') AS pending_count,
			COALESCE((SELECT SUM(retry_count) FROM batches), 0) AS retried_total,
			(SELECT COUNT(1) FROM batches WHERE status = 'anchor_failed') AS failed_total,
			(SELECT MAX(updated_at) FROM reconcile_jobs) AS last_reconcile_at`,
	).Scan(&pendingCount, &retriedTotal, &failedTotal, &lastUpdate)
	if err != nil {
		return domain.ReconcileStats{}, mapDBErr(err)
	}

	var lastReconcile *time.Time
	if lastUpdate.Valid {
		parsed, err := parseTime(lastUpdate.String)
		if err != nil {
			return domain.ReconcileStats{}, err
		}
		lastReconcile = &parsed
	}

	return domain.ReconcileStats{
		PendingCount:    pendingCount,
		RetriedTotal:    retriedTotal,
		FailedTotal:     failedTotal,
		LastReconcileAt: lastReconcile,
	}, nil
}

func (r *Repository) AppendAuditLog(ctx context.Context, log domain.AuditLogRecord) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO audit_logs (
			event_type, entity_type, entity_id, status, message, request_id, payload_json, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		log.EventType,
		log.EntityType,
		log.EntityID,
		toNullString(log.Status),
		toNullString(log.Message),
		toNullString(log.RequestID),
		toNullString(log.PayloadJSON),
		normalizeTime(log.CreatedAt).Format(time.RFC3339Nano),
	)
	if err != nil {
		return mapDBErr(err)
	}
	return nil
}

func (r *Repository) CountBatches(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM batches").Scan(&count)
	if err != nil {
		return 0, mapDBErr(err)
	}
	return count, nil
}

func (r *Repository) CountByStatus(ctx context.Context) (domain.StatusDistribution, error) {
	var out domain.StatusDistribution
	err := r.db.QueryRowContext(
		ctx,
		`SELECT
			COALESCE(SUM(CASE WHEN status = 'anchored' THEN 1 ELSE 0 END), 0) AS anchored,
			COALESCE(SUM(CASE WHEN status = 'pending_anchor' THEN 1 ELSE 0 END), 0) AS pending_anchor,
			COALESCE(SUM(CASE WHEN status = 'anchor_failed' THEN 1 ELSE 0 END), 0) AS anchor_failed
		FROM batches`,
	).Scan(&out.Anchored, &out.PendingAnchor, &out.AnchorFailed)
	if err != nil {
		return domain.StatusDistribution{}, mapDBErr(err)
	}
	return out, nil
}

func (r *Repository) SumRipeness(ctx context.Context) (domain.RipenessDistribution, error) {
	var out domain.RipenessDistribution
	err := r.db.QueryRowContext(
		ctx,
		`SELECT
			COALESCE(SUM(green), 0),
			COALESCE(SUM(half), 0),
			COALESCE(SUM(red), 0),
			COALESCE(SUM(young), 0)
		FROM batches`,
	).Scan(&out.Green, &out.Half, &out.Red, &out.Young)
	if err != nil {
		return domain.RipenessDistribution{}, mapDBErr(err)
	}
	return out, nil
}

func (r *Repository) CountUnripeBatches(ctx context.Context, threshold float64) (int64, float64, error) {
	var total, count int64
	err := r.db.QueryRowContext(
		ctx,
		`SELECT
			COUNT(1) AS total,
			COALESCE(SUM(CASE WHEN unripe_ratio > ? THEN 1 ELSE 0 END), 0) AS cnt
		FROM batches`,
		threshold,
	).Scan(&total, &count)
	if err != nil {
		return 0, 0, mapDBErr(err)
	}

	if total == 0 {
		return 0, 0, nil
	}
	return count, float64(count) / float64(total), nil
}

func (r *Repository) ListRecentAnchors(ctx context.Context, limit int) ([]domain.RecentAnchorRecord, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT
			b.batch_id, b.trace_code, b.status, ap.tx_hash, ap.anchored_at, b.created_at
		FROM batches b
		LEFT JOIN anchor_proofs ap ON ap.batch_id = b.batch_id
		ORDER BY b.created_at DESC
		LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, mapDBErr(err)
	}
	defer rows.Close()

	out := make([]domain.RecentAnchorRecord, 0, limit)
	for rows.Next() {
		var (
			record       domain.RecentAnchorRecord
			txHash       sql.NullString
			anchoredAt   sql.NullString
			createdAtStr string
		)
		if err := rows.Scan(
			&record.BatchID,
			&record.TraceCode,
			&record.Status,
			&txHash,
			&anchoredAt,
			&createdAtStr,
		); err != nil {
			return nil, mapDBErr(err)
		}

		createdAt, err := parseTime(createdAtStr)
		if err != nil {
			return nil, err
		}
		record.CreatedAt = createdAt
		record.TxHash = fromNullString(txHash)

		if anchoredAt.Valid {
			t, err := parseTime(anchoredAt.String)
			if err != nil {
				return nil, err
			}
			record.AnchoredAt = &t
		}
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDBErr(err)
	}
	return out, nil
}

func (r *Repository) getBatchBy(ctx context.Context, where string, value any) (domain.BatchRecord, error) {
	query := fmt.Sprintf(
		`SELECT
			b.batch_id, b.trace_code, b.status, b.orchard_id, b.orchard_name, b.plot_id, b.plot_name,
			b.harvested_at, b.total, b.green, b.half, b.red, b.young, b.unripe_count, b.unripe_ratio,
			b.unripe_handling, b.note, b.anchor_hash, b.confirm_unripe, b.retry_count, b.last_error,
			b.created_at, b.updated_at,
			ap.tx_hash, ap.block_number, ap.chain_id, ap.contract_address, ap.anchor_hash, ap.anchored_at
		FROM batches b
		LEFT JOIN anchor_proofs ap ON ap.batch_id = b.batch_id
		WHERE %s
		LIMIT 1`,
		where,
	)
	row := r.db.QueryRowContext(ctx, query, value)
	record, err := scanBatch(row)
	if err != nil {
		return domain.BatchRecord{}, err
	}
	return record, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanBatch(scanner rowScanner) (domain.BatchRecord, error) {
	var (
		record           domain.BatchRecord
		statusStr        string
		plotName         sql.NullString
		harvestedAtStr   string
		note             sql.NullString
		anchorHash       sql.NullString
		confirmUnripeInt int
		lastError        sql.NullString
		createdAtStr     string
		updatedAtStr     string
		txHash           sql.NullString
		blockNumber      sql.NullInt64
		chainID          sql.NullString
		contractAddress  sql.NullString
		proofAnchorHash  sql.NullString
		anchoredAt       sql.NullString
	)

	err := scanner.Scan(
		&record.BatchID,
		&record.TraceCode,
		&statusStr,
		&record.OrchardID,
		&record.OrchardName,
		&record.PlotID,
		&plotName,
		&harvestedAtStr,
		&record.Summary.Total,
		&record.Summary.Green,
		&record.Summary.Half,
		&record.Summary.Red,
		&record.Summary.Young,
		&record.Summary.UnripeCount,
		&record.Summary.UnripeRatio,
		&record.Summary.UnripeHandling,
		&note,
		&anchorHash,
		&confirmUnripeInt,
		&record.RetryCount,
		&lastError,
		&createdAtStr,
		&updatedAtStr,
		&txHash,
		&blockNumber,
		&chainID,
		&contractAddress,
		&proofAnchorHash,
		&anchoredAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.BatchRecord{}, repository.ErrNotFound
		}
		return domain.BatchRecord{}, mapDBErr(err)
	}

	record.Status = domain.BatchStatus(statusStr)
	record.PlotName = fromNullString(plotName)
	record.Note = fromNullString(note)
	record.AnchorHash = fromNullString(anchorHash)
	record.ConfirmUnripe = confirmUnripeInt == 1
	record.LastError = fromNullString(lastError)

	harvestedAt, err := parseTime(harvestedAtStr)
	if err != nil {
		return domain.BatchRecord{}, err
	}
	record.HarvestedAt = harvestedAt

	createdAt, err := parseTime(createdAtStr)
	if err != nil {
		return domain.BatchRecord{}, err
	}
	record.CreatedAt = createdAt

	updatedAt, err := parseTime(updatedAtStr)
	if err != nil {
		return domain.BatchRecord{}, err
	}
	record.UpdatedAt = updatedAt

	if txHash.Valid {
		proof := &domain.AnchorProofRecord{
			TxHash:          txHash.String,
			BlockNumber:     blockNumber.Int64,
			ChainID:         chainID.String,
			ContractAddress: contractAddress.String,
			AnchorHash:      proofAnchorHash.String,
		}
		if anchoredAt.Valid {
			t, err := parseTime(anchoredAt.String)
			if err != nil {
				return domain.BatchRecord{}, err
			}
			proof.AnchoredAt = t
		}
		record.AnchorProof = proof
	}

	return record, nil
}

func mapDBErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, repository.ErrNotFound) ||
		errors.Is(err, repository.ErrConflict) ||
		errors.Is(err, repository.ErrInvalidState) ||
		errors.Is(err, repository.ErrDBUnavailable) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return repository.ErrNotFound
	}

	msg := err.Error()
	switch {
	case strings.Contains(msg, "UNIQUE constraint failed"):
		return fmt.Errorf("%w: %v", repository.ErrConflict, err)
	case strings.Contains(msg, "CHECK constraint failed"):
		return fmt.Errorf("%w: %v", repository.ErrInvalidState, err)
	case strings.Contains(msg, "FOREIGN KEY constraint failed"):
		return fmt.Errorf("%w: %v", repository.ErrNotFound, err)
	default:
		return fmt.Errorf("%w: %v", repository.ErrDBUnavailable, err)
	}
}

func parseTime(input string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, input); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, input); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("%w: parse time %q", repository.ErrDBUnavailable, input)
}

func normalizeTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t.UTC()
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func retryCountValue(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func toNullString(v *string) sql.NullString {
	if v == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *v, Valid: true}
}

func fromNullString(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func isValidBatchStatus(status domain.BatchStatus) bool {
	switch status {
	case domain.BatchStatusPendingAnchor, domain.BatchStatusAnchored, domain.BatchStatusAnchorFailed:
		return true
	default:
		return false
	}
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
