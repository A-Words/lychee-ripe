package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lychee-ripe/gateway/internal/domain"
)

func TestDashboardServiceGetOverviewEmptyData(t *testing.T) {
	t.Parallel()

	repo := &fakeDashboardRepo{
		recentAnchors: []domain.RecentAnchorRecord{},
	}
	svc := NewDashboardService(repo, repo, domain.TraceModeDatabase)

	result, err := svc.GetOverview(context.Background())
	if err != nil {
		t.Fatalf("GetOverview failed: %v", err)
	}

	if result.Totals.BatchTotal != 0 {
		t.Fatalf("batch_total = %d, want 0", result.Totals.BatchTotal)
	}
	if result.UnripeMetrics.UnripeBatchCount != 0 || result.UnripeMetrics.UnripeBatchRatio != 0 {
		t.Fatalf("unripe_metrics = %+v, want zero values", result.UnripeMetrics)
	}
	if len(result.RecentAnchors) != 0 {
		t.Fatalf("recent_anchors len = %d, want 0", len(result.RecentAnchors))
	}
	if result.ReconcileStats != nil {
		t.Fatalf("reconcile_stats = %v, want nil", result.ReconcileStats)
	}
}

func TestDashboardServiceGetOverviewNormalData(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 2, 11, 0, 0, 0, time.UTC)
	repo := &fakeDashboardRepo{
		countBatches: 12,
		statusDist: domain.StatusDistribution{
			Anchored:      8,
			PendingAnchor: 3,
			AnchorFailed:  1,
		},
		ripenessDist: domain.RipenessDistribution{
			Green: 20,
			Half:  30,
			Red:   40,
			Young: 10,
		},
		unripeCount: 5,
		unripeRatio: 0.4166667,
		recentAnchors: []domain.RecentAnchorRecord{
			{
				BatchID:   "batch_1",
				TraceCode: "TRC-AAAA-BBBB",
				TraceMode: domain.TraceModeBlockchain,
				Status:    domain.BatchStatusAnchored,
				CreatedAt: now,
			},
		},
		reconcileStats: domain.ReconcileStats{
			PendingCount:    3,
			RetriedTotal:    7,
			FailedTotal:     1,
			LastReconcileAt: &now,
		},
	}
	svc := NewDashboardService(repo, repo, domain.TraceModeBlockchain)

	result, err := svc.GetOverview(context.Background())
	if err != nil {
		t.Fatalf("GetOverview failed: %v", err)
	}

	if result.Totals.BatchTotal != 12 {
		t.Fatalf("batch_total = %d, want 12", result.Totals.BatchTotal)
	}
	if result.StatusDistribution.Anchored != 8 || result.StatusDistribution.PendingAnchor != 3 || result.StatusDistribution.AnchorFailed != 1 {
		t.Fatalf("status_distribution = %+v", result.StatusDistribution)
	}
	if result.RipenessDistribution.Green != 20 || result.RipenessDistribution.Half != 30 || result.RipenessDistribution.Red != 40 || result.RipenessDistribution.Young != 10 {
		t.Fatalf("ripeness_distribution = %+v", result.RipenessDistribution)
	}
	if result.UnripeMetrics.Threshold != dashboardUnripeThreshold {
		t.Fatalf("threshold = %v, want %v", result.UnripeMetrics.Threshold, dashboardUnripeThreshold)
	}
	if result.UnripeMetrics.UnripeHandling != string(domain.UnripeHandlingSortedOut) {
		t.Fatalf("unripe_handling = %q, want %q", result.UnripeMetrics.UnripeHandling, domain.UnripeHandlingSortedOut)
	}
	if len(result.RecentAnchors) != 1 || result.RecentAnchors[0].BatchID != "batch_1" {
		t.Fatalf("recent_anchors = %+v", result.RecentAnchors)
	}
	if result.ReconcileStats == nil || result.ReconcileStats.LastReconcileAt == nil || !result.ReconcileStats.LastReconcileAt.Equal(now) {
		t.Fatalf("last_reconcile_at = %v, want %v", result.ReconcileStats.LastReconcileAt, now)
	}
}

func TestDashboardServiceGetOverviewReturnsServiceUnavailableOnQueryError(t *testing.T) {
	t.Parallel()

	repo := &fakeDashboardRepo{
		countBatchesErr: errors.New("db down"),
	}
	svc := NewDashboardService(repo, repo, domain.TraceModeDatabase)

	_, err := svc.GetOverview(context.Background())
	if !errors.Is(err, ErrServiceUnavailable) {
		t.Fatalf("error = %v, want ErrServiceUnavailable", err)
	}
}

func TestDashboardServiceGetOverviewUsesFixedThresholdAndRecentLimit(t *testing.T) {
	t.Parallel()

	repo := &fakeDashboardRepo{
		recentAnchors: []domain.RecentAnchorRecord{},
	}
	svc := NewDashboardService(repo, repo, domain.TraceModeDatabase)

	if _, err := svc.GetOverview(context.Background()); err != nil {
		t.Fatalf("GetOverview failed: %v", err)
	}

	if repo.lastThreshold != dashboardUnripeThreshold {
		t.Fatalf("threshold = %v, want %v", repo.lastThreshold, dashboardUnripeThreshold)
	}
	if repo.lastRecentLimit != dashboardRecentLimit {
		t.Fatalf("recent limit = %d, want %d", repo.lastRecentLimit, dashboardRecentLimit)
	}
}

type fakeDashboardRepo struct {
	countBatches    int64
	countBatchesErr error

	statusDist domain.StatusDistribution
	statusErr  error

	ripenessDist domain.RipenessDistribution
	ripenessErr  error

	unripeCount int64
	unripeRatio float64
	unripeErr   error

	recentAnchors   []domain.RecentAnchorRecord
	recentErr       error
	lastThreshold   float64
	lastRecentLimit int

	reconcileStats domain.ReconcileStats
	reconcileErr   error
}

func (f *fakeDashboardRepo) CountBatches(_ context.Context) (int64, error) {
	if f.countBatchesErr != nil {
		return 0, f.countBatchesErr
	}
	return f.countBatches, nil
}

func (f *fakeDashboardRepo) CountByStatus(_ context.Context) (domain.StatusDistribution, error) {
	if f.statusErr != nil {
		return domain.StatusDistribution{}, f.statusErr
	}
	return f.statusDist, nil
}

func (f *fakeDashboardRepo) SumRipeness(_ context.Context) (domain.RipenessDistribution, error) {
	if f.ripenessErr != nil {
		return domain.RipenessDistribution{}, f.ripenessErr
	}
	return f.ripenessDist, nil
}

func (f *fakeDashboardRepo) CountUnripeBatches(_ context.Context, threshold float64) (int64, float64, error) {
	f.lastThreshold = threshold
	if f.unripeErr != nil {
		return 0, 0, f.unripeErr
	}
	return f.unripeCount, f.unripeRatio, nil
}

func (f *fakeDashboardRepo) ListRecentAnchors(_ context.Context, limit int) ([]domain.RecentAnchorRecord, error) {
	f.lastRecentLimit = limit
	if f.recentErr != nil {
		return nil, f.recentErr
	}
	return f.recentAnchors, nil
}

func (f *fakeDashboardRepo) CreateReconcileJob(_ context.Context, _ domain.CreateReconcileJobParams) (domain.ReconcileJobRecord, error) {
	return domain.ReconcileJobRecord{}, nil
}

func (f *fakeDashboardRepo) AddReconcileJobItems(_ context.Context, _ string, _ []domain.ReconcileJobItemRecord) error {
	return nil
}

func (f *fakeDashboardRepo) UpdateReconcileJobStatus(_ context.Context, _ string, _ domain.ReconcileJobStatus, _ *string, _ time.Time) error {
	return nil
}

func (f *fakeDashboardRepo) ListReconcileStats(_ context.Context) (domain.ReconcileStats, error) {
	if f.reconcileErr != nil {
		return domain.ReconcileStats{}, f.reconcileErr
	}
	return f.reconcileStats, nil
}
