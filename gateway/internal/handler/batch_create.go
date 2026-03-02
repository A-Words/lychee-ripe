package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/middleware"
	"github.com/lychee-ripe/gateway/internal/service"
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

type batchResponse struct {
	BatchID     string               `json:"batch_id"`
	TraceCode   string               `json:"trace_code"`
	Status      string               `json:"status"`
	OrchardID   string               `json:"orchard_id"`
	OrchardName string               `json:"orchard_name"`
	PlotID      string               `json:"plot_id"`
	PlotName    *string              `json:"plot_name"`
	HarvestedAt time.Time            `json:"harvested_at"`
	Summary     batchSummaryResponse `json:"summary"`
	Note        *string              `json:"note"`
	CreatedAt   time.Time            `json:"created_at"`
	AnchorProof *anchorProofResponse `json:"anchor_proof"`
}

type batchSummaryResponse struct {
	Total          int     `json:"total"`
	Green          int     `json:"green"`
	Half           int     `json:"half"`
	Red            int     `json:"red"`
	Young          int     `json:"young"`
	UnripeCount    int     `json:"unripe_count"`
	UnripeRatio    float64 `json:"unripe_ratio"`
	UnripeHandling string  `json:"unripe_handling"`
}

type anchorProofResponse struct {
	TxHash          string    `json:"tx_hash"`
	BlockNumber     int64     `json:"block_number"`
	ChainID         string    `json:"chain_id"`
	ContractAddress string    `json:"contract_address"`
	AnchorHash      string    `json:"anchor_hash"`
	AnchoredAt      time.Time `json:"anchored_at"`
}

type errorResponse struct {
	Error     string         `json:"error"`
	Message   string         `json:"message"`
	RequestID string         `json:"request_id,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
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

func toBatchResponse(record domain.BatchRecord) batchResponse {
	resp := batchResponse{
		BatchID:     record.BatchID,
		TraceCode:   record.TraceCode,
		Status:      string(record.Status),
		OrchardID:   record.OrchardID,
		OrchardName: record.OrchardName,
		PlotID:      record.PlotID,
		PlotName:    record.PlotName,
		HarvestedAt: record.HarvestedAt.UTC(),
		Summary: batchSummaryResponse{
			Total:          record.Summary.Total,
			Green:          record.Summary.Green,
			Half:           record.Summary.Half,
			Red:            record.Summary.Red,
			Young:          record.Summary.Young,
			UnripeCount:    record.Summary.UnripeCount,
			UnripeRatio:    record.Summary.UnripeRatio,
			UnripeHandling: string(record.Summary.UnripeHandling),
		},
		Note:      record.Note,
		CreatedAt: record.CreatedAt.UTC(),
	}

	if record.AnchorProof != nil {
		resp.AnchorProof = &anchorProofResponse{
			TxHash:          record.AnchorProof.TxHash,
			BlockNumber:     record.AnchorProof.BlockNumber,
			ChainID:         record.AnchorProof.ChainID,
			ContractAddress: record.AnchorProof.ContractAddress,
			AnchorHash:      record.AnchorProof.AnchorHash,
			AnchoredAt:      record.AnchorProof.AnchoredAt.UTC(),
		}
	}
	return resp
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

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(
	w http.ResponseWriter,
	r *http.Request,
	statusCode int,
	code string,
	message string,
	details map[string]any,
) {
	reqID := middleware.GetRequestID(r.Context())
	resp := errorResponse{
		Error:   code,
		Message: sanitizeMessage(message),
		Details: details,
	}
	if reqID != "" {
		resp.RequestID = reqID
	}
	writeJSON(w, statusCode, resp)
}

func sanitizeMessage(message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return "request failed"
	}
	return trimmed
}
