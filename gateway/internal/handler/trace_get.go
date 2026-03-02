package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/lychee-ripe/gateway/internal/service"
)

type TraceGetService interface {
	GetPublicTrace(ctx context.Context, traceCode string) (service.TraceQueryResult, error)
}

type traceResponse struct {
	Batch        traceBatchResponse        `json:"batch"`
	VerifyResult traceVerifyResultResponse `json:"verify_result"`
}

type traceBatchResponse struct {
	BatchID     string               `json:"batch_id"`
	TraceCode   string               `json:"trace_code"`
	Status      string               `json:"status"`
	OrchardName string               `json:"orchard_name"`
	PlotName    string               `json:"plot_name"`
	HarvestedAt time.Time            `json:"harvested_at"`
	Summary     batchSummaryResponse `json:"summary"`
	CreatedAt   time.Time            `json:"created_at"`
}

type traceVerifyResultResponse struct {
	VerifyStatus string `json:"verify_status"`
	Reason       string `json:"reason"`
}

func GetPublicTrace(svc TraceGetService, logger *slog.Logger) http.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}

	return func(w http.ResponseWriter, r *http.Request) {
		traceCode := r.PathValue("trace_code")

		result, err := svc.GetPublicTrace(r.Context(), traceCode)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrNotFound):
				writeError(w, r, http.StatusNotFound, "trace_not_found", err.Error(), nil)
			case errors.Is(err, service.ErrServiceUnavailable):
				writeError(w, r, http.StatusServiceUnavailable, "service_unavailable", err.Error(), nil)
			default:
				logger.Error("get public trace unexpected error", "trace_code", traceCode, "error", err)
				writeError(w, r, http.StatusServiceUnavailable, "service_unavailable", "service unavailable", nil)
			}
			return
		}

		writeJSON(w, http.StatusOK, toTraceResponse(result))
	}
}

func toTraceResponse(result service.TraceQueryResult) traceResponse {
	plotName := ""
	if result.Batch.PlotName != nil {
		plotName = *result.Batch.PlotName
	}

	return traceResponse{
		Batch: traceBatchResponse{
			BatchID:     result.Batch.BatchID,
			TraceCode:   result.Batch.TraceCode,
			Status:      string(result.Batch.Status),
			OrchardName: result.Batch.OrchardName,
			PlotName:    plotName,
			HarvestedAt: result.Batch.HarvestedAt.UTC(),
			Summary: batchSummaryResponse{
				Total:          result.Batch.Summary.Total,
				Green:          result.Batch.Summary.Green,
				Half:           result.Batch.Summary.Half,
				Red:            result.Batch.Summary.Red,
				Young:          result.Batch.Summary.Young,
				UnripeCount:    result.Batch.Summary.UnripeCount,
				UnripeRatio:    result.Batch.Summary.UnripeRatio,
				UnripeHandling: string(result.Batch.Summary.UnripeHandling),
			},
			CreatedAt: result.Batch.CreatedAt.UTC(),
		},
		VerifyResult: traceVerifyResultResponse{
			VerifyStatus: result.VerifyStatus,
			Reason:       result.Reason,
		},
	}
}
