package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/service"
)

type BatchGetService interface {
	GetBatchByID(ctx context.Context, batchID string) (domain.BatchRecord, error)
}

func GetBatch(svc BatchGetService, logger *slog.Logger) http.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}

	return func(w http.ResponseWriter, r *http.Request) {
		batchID := r.PathValue("batch_id")

		record, err := svc.GetBatchByID(r.Context(), batchID)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrNotFound):
				writeError(w, r, http.StatusNotFound, "batch_not_found", err.Error(), nil)
			case errors.Is(err, service.ErrServiceUnavailable):
				writeError(w, r, http.StatusServiceUnavailable, "service_unavailable", err.Error(), nil)
			default:
				logger.Error("get batch unexpected error", "batch_id", batchID, "error", err)
				writeError(w, r, http.StatusServiceUnavailable, "service_unavailable", "service unavailable", nil)
			}
			return
		}

		writeJSON(w, http.StatusOK, toBatchResponse(record))
	}
}
