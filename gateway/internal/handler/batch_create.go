package handler

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/lychee-ripe/gateway/internal/service"
	"io"
	"log/slog"
	"net/http"
)

type BatchCreateService interface {
	CreateBatch(ctx context.Context, input service.BatchCreateInput) (service.CreateBatchResult, error)
}

type batchCreateRequest struct {
	OrchardID     string                   `json:"orchard_id"`
	OrchardName   string                   `json:"orchard_name"`
	PlotID        string                   `json:"plot_id"`
	PlotName      *string                  `json:"plot_name"`
	HarvestedAt   string                   `json:"harvested_at"`
	Summary       batchSummaryInputRequest `json:"summary"`
	Note          *string                  `json:"note"`
	ConfirmUnripe bool                     `json:"confirm_unripe"`
}

type batchSummaryInputRequest struct {
	Total int `json:"total"`
	Green int `json:"green"`
	Half  int `json:"half"`
	Red   int `json:"red"`
	Young int `json:"young"`
}

func CreateBatch(svc BatchCreateService, logger *slog.Logger) http.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req batchCreateRequest
		if err := decodeJSONBody(r, &req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), nil)
			return
		}

		result, err := svc.CreateBatch(r.Context(), service.BatchCreateInput{
			OrchardID:   req.OrchardID,
			OrchardName: req.OrchardName,
			PlotID:      req.PlotID,
			PlotName:    req.PlotName,
			HarvestedAt: req.HarvestedAt,
			Summary: service.BatchSummaryInput{
				Total: req.Summary.Total,
				Green: req.Summary.Green,
				Half:  req.Summary.Half,
				Red:   req.Summary.Red,
				Young: req.Summary.Young,
			},
			Note:          req.Note,
			ConfirmUnripe: req.ConfirmUnripe,
		})
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInvalidRequest):
				writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), nil)
			case errors.Is(err, service.ErrConflict):
				writeError(w, r, http.StatusConflict, "duplicated_batch", err.Error(), nil)
			case errors.Is(err, service.ErrServiceUnavailable):
				writeError(w, r, http.StatusServiceUnavailable, "service_unavailable", err.Error(), nil)
			default:
				logger.Error("create batch unexpected error", "error", err)
				writeError(w, r, http.StatusServiceUnavailable, "service_unavailable", "service unavailable", nil)
			}
			return
		}

		resp := toBatchResponse(result.Batch)
		writeJSON(w, result.HTTPStatus, resp)
	}
}

func decodeJSONBody(r *http.Request, out any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("request body is required")
		}
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}
