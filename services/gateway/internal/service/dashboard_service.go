package service

import (
	"context"
	"fmt"

	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/repository"
)

const (
	dashboardUnripeThreshold = 0.15
	dashboardRecentLimit     = 20
)

type DashboardTotals struct {
	BatchTotal int64
}

type DashboardUnripeMetrics struct {
	UnripeBatchCount int64
	UnripeBatchRatio float64
	Threshold        float64
	UnripeHandling   string
}

type DashboardOverviewResult struct {
	Totals               DashboardTotals
	StatusDistribution   domain.StatusDistribution
	RipenessDistribution domain.RipenessDistribution
	UnripeMetrics        DashboardUnripeMetrics
	RecentAnchors        []domain.RecentAnchorRecord
	ReconcileStats       domain.ReconcileStats
}

type DashboardService struct {
	queryRepo     repository.DashboardQueryRepository
	reconcileRepo repository.ReconcileRepository
}

func NewDashboardService(
	queryRepo repository.DashboardQueryRepository,
	reconcileRepo repository.ReconcileRepository,
) *DashboardService {
	return &DashboardService{
		queryRepo:     queryRepo,
		reconcileRepo: reconcileRepo,
	}
}

func (s *DashboardService) GetOverview(ctx context.Context) (DashboardOverviewResult, error) {
	total, err := s.queryRepo.CountBatches(ctx)
	if err != nil {
		return DashboardOverviewResult{}, fmt.Errorf("%w: count batches: %v", ErrServiceUnavailable, err)
	}

	status, err := s.queryRepo.CountByStatus(ctx)
	if err != nil {
		return DashboardOverviewResult{}, fmt.Errorf("%w: count status distribution: %v", ErrServiceUnavailable, err)
	}

	ripeness, err := s.queryRepo.SumRipeness(ctx)
	if err != nil {
		return DashboardOverviewResult{}, fmt.Errorf("%w: sum ripeness distribution: %v", ErrServiceUnavailable, err)
	}

	unripeCount, unripeRatio, err := s.queryRepo.CountUnripeBatches(ctx, dashboardUnripeThreshold)
	if err != nil {
		return DashboardOverviewResult{}, fmt.Errorf("%w: count unripe batches: %v", ErrServiceUnavailable, err)
	}

	recent, err := s.queryRepo.ListRecentAnchors(ctx, dashboardRecentLimit)
	if err != nil {
		return DashboardOverviewResult{}, fmt.Errorf("%w: list recent anchors: %v", ErrServiceUnavailable, err)
	}
	if recent == nil {
		recent = make([]domain.RecentAnchorRecord, 0)
	}

	reconcileStats, err := s.reconcileRepo.ListReconcileStats(ctx)
	if err != nil {
		return DashboardOverviewResult{}, fmt.Errorf("%w: list reconcile stats: %v", ErrServiceUnavailable, err)
	}

	return DashboardOverviewResult{
		Totals: DashboardTotals{
			BatchTotal: total,
		},
		StatusDistribution:   status,
		RipenessDistribution: ripeness,
		UnripeMetrics: DashboardUnripeMetrics{
			UnripeBatchCount: unripeCount,
			UnripeBatchRatio: unripeRatio,
			Threshold:        dashboardUnripeThreshold,
			UnripeHandling:   string(domain.UnripeHandlingSortedOut),
		},
		RecentAnchors:  recent,
		ReconcileStats: reconcileStats,
	}, nil
}
