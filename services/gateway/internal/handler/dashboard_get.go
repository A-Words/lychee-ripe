package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/lychee-ripe/gateway/internal/service"
)

type DashboardGetService interface {
	GetOverview(ctx context.Context) (service.DashboardOverviewResult, error)
}

type dashboardOverviewResponse struct {
	Totals               dashboardTotalsResponse             `json:"totals"`
	StatusDistribution   dashboardStatusDistributionResponse `json:"status_distribution"`
	RipenessDistribution dashboardRipenessDistribution       `json:"ripeness_distribution"`
	UnripeMetrics        dashboardUnripeMetricsResponse      `json:"unripe_metrics"`
	RecentAnchors        []dashboardRecentAnchorResponse     `json:"recent_anchors"`
	ReconcileStats       dashboardReconcileStatsResponse     `json:"reconcile_stats"`
}

type dashboardTotalsResponse struct {
	BatchTotal int64 `json:"batch_total"`
}

type dashboardStatusDistributionResponse struct {
	Anchored      int64 `json:"anchored"`
	PendingAnchor int64 `json:"pending_anchor"`
	AnchorFailed  int64 `json:"anchor_failed"`
}

type dashboardRipenessDistribution struct {
	Green int64 `json:"green"`
	Half  int64 `json:"half"`
	Red   int64 `json:"red"`
	Young int64 `json:"young"`
}

type dashboardUnripeMetricsResponse struct {
	UnripeBatchCount int64   `json:"unripe_batch_count"`
	UnripeBatchRatio float64 `json:"unripe_batch_ratio"`
	Threshold        float64 `json:"threshold"`
	UnripeHandling   string  `json:"unripe_handling"`
}

type dashboardRecentAnchorResponse struct {
	BatchID    string     `json:"batch_id"`
	TraceCode  string     `json:"trace_code"`
	Status     string     `json:"status"`
	TxHash     *string    `json:"tx_hash"`
	AnchoredAt *time.Time `json:"anchored_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

type dashboardReconcileStatsResponse struct {
	PendingCount    int64      `json:"pending_count"`
	RetriedTotal    int64      `json:"retried_total"`
	FailedTotal     int64      `json:"failed_total"`
	LastReconcileAt *time.Time `json:"last_reconcile_at"`
}

func GetDashboardOverview(svc DashboardGetService, logger *slog.Logger) http.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}

	return func(w http.ResponseWriter, r *http.Request) {
		result, err := svc.GetOverview(r.Context())
		if err != nil {
			switch {
			case errors.Is(err, service.ErrServiceUnavailable):
				writeError(w, r, http.StatusServiceUnavailable, "service_unavailable", err.Error(), nil)
			default:
				logger.Error("get dashboard overview unexpected error", "error", err)
				writeError(w, r, http.StatusServiceUnavailable, "service_unavailable", "service unavailable", nil)
			}
			return
		}

		writeJSON(w, http.StatusOK, toDashboardOverviewResponse(result))
	}
}

func toDashboardOverviewResponse(result service.DashboardOverviewResult) dashboardOverviewResponse {
	recent := make([]dashboardRecentAnchorResponse, 0, len(result.RecentAnchors))
	for _, item := range result.RecentAnchors {
		recent = append(recent, dashboardRecentAnchorResponse{
			BatchID:    item.BatchID,
			TraceCode:  item.TraceCode,
			Status:     string(item.Status),
			TxHash:     item.TxHash,
			AnchoredAt: item.AnchoredAt,
			CreatedAt:  item.CreatedAt.UTC(),
		})
	}

	return dashboardOverviewResponse{
		Totals: dashboardTotalsResponse{
			BatchTotal: result.Totals.BatchTotal,
		},
		StatusDistribution: dashboardStatusDistributionResponse{
			Anchored:      result.StatusDistribution.Anchored,
			PendingAnchor: result.StatusDistribution.PendingAnchor,
			AnchorFailed:  result.StatusDistribution.AnchorFailed,
		},
		RipenessDistribution: dashboardRipenessDistribution{
			Green: result.RipenessDistribution.Green,
			Half:  result.RipenessDistribution.Half,
			Red:   result.RipenessDistribution.Red,
			Young: result.RipenessDistribution.Young,
		},
		UnripeMetrics: dashboardUnripeMetricsResponse{
			UnripeBatchCount: result.UnripeMetrics.UnripeBatchCount,
			UnripeBatchRatio: result.UnripeMetrics.UnripeBatchRatio,
			Threshold:        result.UnripeMetrics.Threshold,
			UnripeHandling:   result.UnripeMetrics.UnripeHandling,
		},
		RecentAnchors: recent,
		ReconcileStats: dashboardReconcileStatsResponse{
			PendingCount:    result.ReconcileStats.PendingCount,
			RetriedTotal:    result.ReconcileStats.RetriedTotal,
			FailedTotal:     result.ReconcileStats.FailedTotal,
			LastReconcileAt: result.ReconcileStats.LastReconcileAt,
		},
	}
}
