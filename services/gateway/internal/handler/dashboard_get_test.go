package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/middleware"
	"github.com/lychee-ripe/gateway/internal/service"
)

func TestGetDashboardOverviewReturns200(t *testing.T) {
	t.Parallel()

	last := time.Date(2026, 3, 2, 11, 20, 0, 0, time.UTC)
	tx := "0xabc"
	svc := &fakeDashboardGetService{
		result: service.DashboardOverviewResult{
			TraceMode: domain.TraceModeBlockchain,
			Totals: service.DashboardTotals{BatchTotal: 10},
			StatusDistribution: domain.StatusDistribution{
				Anchored:      7,
				PendingAnchor: 2,
				AnchorFailed:  1,
			},
			RipenessDistribution: domain.RipenessDistribution{
				Green: 10,
				Half:  20,
				Red:   30,
				Young: 5,
			},
			UnripeMetrics: service.DashboardUnripeMetrics{
				UnripeBatchCount: 4,
				UnripeBatchRatio: 0.4,
				Threshold:        0.15,
				UnripeHandling:   "sorted_out",
			},
			RecentAnchors: []domain.RecentAnchorRecord{
				{
					BatchID:    "batch_1",
					TraceCode:  "TRC-1111-AAAA",
					TraceMode:  domain.TraceModeBlockchain,
					Status:     domain.BatchStatusAnchored,
					TxHash:     &tx,
					AnchoredAt: &last,
					CreatedAt:  last,
				},
			},
			ReconcileStats: &domain.ReconcileStats{
				PendingCount:    2,
				RetriedTotal:    3,
				FailedTotal:     1,
				LastReconcileAt: &last,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/dashboard/overview", nil)
	rec := httptest.NewRecorder()
	GetDashboardOverview(svc, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp["totals"] == nil || resp["status_distribution"] == nil || resp["ripeness_distribution"] == nil {
		t.Fatalf("missing top-level fields: %+v", resp)
	}
	if resp["trace_mode"] != "blockchain" {
		t.Fatalf("trace_mode = %v, want blockchain", resp["trace_mode"])
	}
	if resp["recent_anchors"] == nil || resp["reconcile_stats"] == nil {
		t.Fatalf("missing recent_anchors/reconcile_stats: %+v", resp)
	}
}

func TestGetDashboardOverviewRecentAnchorsAndReconcileStatsShape(t *testing.T) {
	t.Parallel()

	svc := &fakeDashboardGetService{
		result: service.DashboardOverviewResult{
			TraceMode:     domain.TraceModeDatabase,
			StatusDistribution: domain.StatusDistribution{
				Stored: 3,
			},
			RecentAnchors: []domain.RecentAnchorRecord{},
			UnripeMetrics: service.DashboardUnripeMetrics{
				Threshold:      0.15,
				UnripeHandling: "sorted_out",
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/dashboard/overview", nil)
	rec := httptest.NewRecorder()
	GetDashboardOverview(svc, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	recent, ok := resp["recent_anchors"].([]any)
	if !ok {
		t.Fatalf("recent_anchors type = %T, want []any", resp["recent_anchors"])
	}
	if len(recent) != 0 {
		t.Fatalf("recent_anchors len = %d, want 0", len(recent))
	}
	status, ok := resp["status_distribution"].(map[string]any)
	if !ok {
		t.Fatalf("status_distribution type = %T, want object", resp["status_distribution"])
	}
	if status["stored"] != float64(3) {
		t.Fatalf("stored = %v, want 3", status["stored"])
	}
	if status["anchored"] != nil || status["pending_anchor"] != nil || status["anchor_failed"] != nil {
		t.Fatalf("unexpected blockchain fields in database mode: %+v", status)
	}
	if _, ok := resp["reconcile_stats"]; ok {
		t.Fatalf("reconcile_stats should be omitted in database mode: %+v", resp)
	}
}

func TestGetDashboardOverviewReturns503(t *testing.T) {
	t.Parallel()

	svc := &fakeDashboardGetService{err: service.ErrServiceUnavailable}
	req := httptest.NewRequest(http.MethodGet, "/v1/dashboard/overview", nil)
	rec := httptest.NewRecorder()
	GetDashboardOverview(svc, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestGetDashboardOverviewErrorContainsRequestID(t *testing.T) {
	t.Parallel()

	svc := &fakeDashboardGetService{err: service.ErrServiceUnavailable}
	req := httptest.NewRequest(http.MethodGet, "/v1/dashboard/overview", nil)
	req.Header.Set("X-Request-ID", "rid-dashboard-503")
	rec := httptest.NewRecorder()

	handler := middleware.RequestID(GetDashboardOverview(svc, nil))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["request_id"] != "rid-dashboard-503" {
		t.Fatalf("request_id = %v, want rid-dashboard-503", resp["request_id"])
	}
}

type fakeDashboardGetService struct {
	result service.DashboardOverviewResult
	err    error
}

func (f *fakeDashboardGetService) GetOverview(_ context.Context) (service.DashboardOverviewResult, error) {
	if f.err != nil {
		return service.DashboardOverviewResult{}, f.err
	}
	return f.result, nil
}
