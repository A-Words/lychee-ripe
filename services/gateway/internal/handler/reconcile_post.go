package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/lychee-ripe/gateway/internal/service"
)

type ReconcileService interface {
	TriggerManualReconcile(ctx context.Context, input service.ManualReconcileInput) (service.ReconcileResult, error)
}

type reconcileRequest struct {
	BatchIDs []string `json:"batch_ids"`
	Limit    *int     `json:"limit"`
}

type reconcileResponse struct {
	Accepted       bool   `json:"accepted"`
	RequestedCount int    `json:"requested_count"`
	ScheduledCount int    `json:"scheduled_count"`
	SkippedCount   int    `json:"skipped_count"`
	Message        string `json:"message,omitempty"`
}

func ReconcileBatches(svc ReconcileService, logger *slog.Logger) http.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req reconcileRequest
		if err := decodeOptionalJSONBody(r, &req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), nil)
			return
		}

		result, err := svc.TriggerManualReconcile(r.Context(), service.ManualReconcileInput{
			BatchIDs: req.BatchIDs,
			Limit:    req.Limit,
		})
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInvalidRequest):
				writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), nil)
			case errors.Is(err, service.ErrNotFound):
				writeError(w, r, http.StatusNotFound, "pending_batch_not_found", err.Error(), nil)
			case errors.Is(err, service.ErrServiceUnavailable):
				writeError(w, r, http.StatusServiceUnavailable, "service_unavailable", err.Error(), nil)
			default:
				logger.Error("reconcile batches unexpected error", "error", err)
				writeError(w, r, http.StatusServiceUnavailable, "service_unavailable", "service unavailable", nil)
			}
			return
		}

		writeJSON(w, http.StatusAccepted, reconcileResponse{
			Accepted:       result.Accepted,
			RequestedCount: result.RequestedCount,
			ScheduledCount: result.ScheduledCount,
			SkippedCount:   result.SkippedCount,
			Message:        result.Message,
		})
	}
}

func decodeOptionalJSONBody(r *http.Request, out any) error {
	if r.Body == nil {
		return nil
	}

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil
	}

	r.Body = io.NopCloser(bytes.NewReader(raw))
	return decodeJSONBody(r, out)
}
