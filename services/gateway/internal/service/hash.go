package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/lychee-ripe/gateway/internal/domain"
)

const unripeRatioDigits = 6

type anchorHashPayload struct {
	BatchID     string
	TraceCode   string
	OrchardID   string
	OrchardName string
	PlotID      string
	PlotName    *string
	HarvestedAt time.Time
	Summary     domain.BatchSummary
	Note        *string
}

type canonicalBatchPayload struct {
	BatchID     string                `json:"batch_id"`
	TraceCode   string                `json:"trace_code"`
	OrchardID   string                `json:"orchard_id"`
	OrchardName string                `json:"orchard_name"`
	PlotID      string                `json:"plot_id"`
	PlotName    *string               `json:"plot_name"`
	HarvestedAt string                `json:"harvested_at"`
	Summary     canonicalBatchSummary `json:"summary"`
	Note        *string               `json:"note"`
}

type canonicalBatchSummary struct {
	Total          int         `json:"total"`
	Green          int         `json:"green"`
	Half           int         `json:"half"`
	Red            int         `json:"red"`
	Young          int         `json:"young"`
	UnripeCount    int         `json:"unripe_count"`
	UnripeRatio    json.Number `json:"unripe_ratio"`
	UnripeHandling string      `json:"unripe_handling"`
}

func computeAnchorHash(payload anchorHashPayload) (string, error) {
	canonical := canonicalBatchPayload{
		BatchID:     payload.BatchID,
		TraceCode:   payload.TraceCode,
		OrchardID:   payload.OrchardID,
		OrchardName: payload.OrchardName,
		PlotID:      payload.PlotID,
		PlotName:    payload.PlotName,
		HarvestedAt: payload.HarvestedAt.UTC().Format(time.RFC3339Nano),
		Summary: canonicalBatchSummary{
			Total:          payload.Summary.Total,
			Green:          payload.Summary.Green,
			Half:           payload.Summary.Half,
			Red:            payload.Summary.Red,
			Young:          payload.Summary.Young,
			UnripeCount:    payload.Summary.UnripeCount,
			UnripeRatio:    json.Number(fmt.Sprintf("%.6f", payload.Summary.UnripeRatio)),
			UnripeHandling: string(payload.Summary.UnripeHandling),
		},
		Note: payload.Note,
	}

	raw, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(raw)
	return "0x" + hex.EncodeToString(sum[:]), nil
}

func roundTo(value float64, digits int) float64 {
	pow := math.Pow10(digits)
	return math.Round(value*pow) / pow
}
