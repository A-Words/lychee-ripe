package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/middleware"
)

type batchResponse struct {
	BatchID     string               `json:"batch_id"`
	TraceCode   string               `json:"trace_code"`
	TraceMode   string               `json:"trace_mode"`
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

func toBatchResponse(record domain.BatchRecord) batchResponse {
	resp := batchResponse{
		BatchID:     record.BatchID,
		TraceCode:   record.TraceCode,
		TraceMode:   string(record.TraceMode),
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
