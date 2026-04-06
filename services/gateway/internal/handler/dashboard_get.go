package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/service"
)

type DashboardGetService interface {
	GetOverview(ctx context.Context) (service.DashboardOverviewResult, error)
}

type dashboardOverviewResponse struct {
	TraceMode            string                              `json:"trace_mode"`
	Totals               dashboardTotalsResponse             `json:"totals"`
	StatusDistribution   dashboardStatusDistributionResponse `json:"status_distribution"`
	RipenessDistribution dashboardRipenessDistribution       `json:"ripeness_distribution"`
	UnripeMetrics        dashboardUnripeMetricsResponse      `json:"unripe_metrics"`
	RecentAnchors        []dashboardRecentAnchorResponse     `json:"recent_anchors"`
	ReconcileStats       *dashboardReconcileStatsResponse    `json:"reconcile_stats,omitempty"`
}

type dashboardTotalsResponse struct {
	BatchTotal int64 `json:"batch_total"`
}

type dashboardStatusDistributionResponse struct {
	Stored        *int64 `json:"stored,omitempty"`
	Anchored      *int64 `json:"anchored,omitempty"`
	PendingAnchor *int64 `json:"pending_anchor,omitempty"`
	AnchorFailed  *int64 `json:"anchor_failed,omitempty"`
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
		TraceMode: string(result.TraceMode),
		Totals: dashboardTotalsResponse{
			BatchTotal: result.Totals.BatchTotal,
		},
		StatusDistribution: toDashboardStatusDistributionResponse(result),
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
		ReconcileStats: toDashboardReconcileStatsResponse(result.ReconcileStats),
	}
}

func toDashboardStatusDistributionResponse(result service.DashboardOverviewResult) dashboardStatusDistributionResponse {
	if result.TraceMode == "database" {
		return dashboardStatusDistributionResponse{
			Stored: int64Ptr(result.StatusDistribution.Stored),
		}
	}
	return dashboardStatusDistributionResponse{
		Anchored:      int64Ptr(result.StatusDistribution.Anchored),
		PendingAnchor: int64Ptr(result.StatusDistribution.PendingAnchor),
		AnchorFailed:  int64Ptr(result.StatusDistribution.AnchorFailed),
	}
}

func toDashboardReconcileStatsResponse(stats *domain.ReconcileStats) *dashboardReconcileStatsResponse {
	if stats == nil {
		return nil
	}
	return &dashboardReconcileStatsResponse{
		PendingCount:    stats.PendingCount,
		RetriedTotal:    stats.RetriedTotal,
		FailedTotal:     stats.FailedTotal,
		LastReconcileAt: stats.LastReconcileAt,
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}
