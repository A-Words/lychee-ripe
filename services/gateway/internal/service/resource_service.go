package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/repository"
)

type OrchardRepo interface {
	ListOrchards(ctx context.Context, includeArchived bool) ([]domain.OrchardRecord, error)
	CreateOrchard(ctx context.Context, orchard domain.OrchardRecord) (domain.OrchardRecord, error)
	UpdateOrchard(ctx context.Context, orchard domain.OrchardRecord) (domain.OrchardRecord, error)
	GetOrchard(ctx context.Context, orchardID string) (domain.OrchardRecord, error)
}

type PlotRepo interface {
	ListPlots(ctx context.Context, orchardID string, includeArchived bool) ([]domain.PlotRecord, error)
	CreatePlot(ctx context.Context, plot domain.PlotRecord) (domain.PlotRecord, error)
	UpdatePlot(ctx context.Context, plot domain.PlotRecord) (domain.PlotRecord, error)
	GetPlot(ctx context.Context, plotID string) (domain.PlotRecord, error)
}

type SeedRepo interface {
	CountOrchards(ctx context.Context) (int64, error)
	CreateOrchardIfNotExists(ctx context.Context, orchard domain.OrchardRecord) error
	CreatePlotIfNotExists(ctx context.Context, plot domain.PlotRecord) error
}

type OrchardService struct {
	repo  OrchardRepo
	nowFn func() time.Time
}

type PlotService struct {
	repo  PlotRepo
	nowFn func() time.Time
}

type OrchardInput struct {
	OrchardID   string
	OrchardName string
	Status      domain.ResourceStatus
}

type PlotInput struct {
	PlotID    string
	OrchardID string
	PlotName  string
	Status    domain.ResourceStatus
}

func NewOrchardService(repo OrchardRepo) *OrchardService {
	return &OrchardService{repo: repo, nowFn: func() time.Time { return time.Now().UTC() }}
}

func NewPlotService(repo PlotRepo) *PlotService {
	return &PlotService{repo: repo, nowFn: func() time.Time { return time.Now().UTC() }}
}

func (s *OrchardService) List(ctx context.Context, includeArchived bool) ([]domain.OrchardRecord, error) {
	items, err := s.repo.ListOrchards(ctx, includeArchived)
	if err != nil {
		return nil, ErrServiceUnavailable
	}
	return items, nil
}

func (s *OrchardService) Create(ctx context.Context, input OrchardInput) (domain.OrchardRecord, error) {
	record, err := normalizeOrchardInput(input, s.nowFn(), true)
	if err != nil {
		return domain.OrchardRecord{}, err
	}
	created, err := s.repo.CreateOrchard(ctx, record)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return domain.OrchardRecord{}, ErrConflict
		}
		return domain.OrchardRecord{}, ErrServiceUnavailable
	}
	return created, nil
}

func (s *OrchardService) Update(ctx context.Context, orchardID string, input OrchardInput) (domain.OrchardRecord, error) {
	current, err := s.repo.GetOrchard(ctx, strings.TrimSpace(orchardID))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return domain.OrchardRecord{}, ErrNotFound
		}
		return domain.OrchardRecord{}, ErrServiceUnavailable
	}
	current.OrchardName = strings.TrimSpace(input.OrchardName)
	current.Status = input.Status
	current.UpdatedAt = s.nowFn()
	if strings.TrimSpace(current.OrchardName) == "" {
		return domain.OrchardRecord{}, ErrInvalidRequest
	}
	if current.Status == "" {
		current.Status = domain.ResourceStatusActive
	}
	updated, err := s.repo.UpdateOrchard(ctx, current)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return domain.OrchardRecord{}, ErrNotFound
		}
		return domain.OrchardRecord{}, ErrServiceUnavailable
	}
	return updated, nil
}

func (s *OrchardService) Archive(ctx context.Context, orchardID string) (domain.OrchardRecord, error) {
	current, err := s.repo.GetOrchard(ctx, strings.TrimSpace(orchardID))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return domain.OrchardRecord{}, ErrNotFound
		}
		return domain.OrchardRecord{}, ErrServiceUnavailable
	}
	current.Status = domain.ResourceStatusArchived
	current.UpdatedAt = s.nowFn()
	updated, err := s.repo.UpdateOrchard(ctx, current)
	if err != nil {
		return domain.OrchardRecord{}, ErrServiceUnavailable
	}
	return updated, nil
}

func (s *PlotService) List(ctx context.Context, orchardID string, includeArchived bool) ([]domain.PlotRecord, error) {
	items, err := s.repo.ListPlots(ctx, orchardID, includeArchived)
	if err != nil {
		return nil, ErrServiceUnavailable
	}
	return items, nil
}

func (s *PlotService) Create(ctx context.Context, input PlotInput) (domain.PlotRecord, error) {
	record, err := normalizePlotInput(input, s.nowFn(), true)
	if err != nil {
		return domain.PlotRecord{}, err
	}
	created, err := s.repo.CreatePlot(ctx, record)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return domain.PlotRecord{}, ErrConflict
		}
		return domain.PlotRecord{}, ErrServiceUnavailable
	}
	return created, nil
}

func (s *PlotService) Update(ctx context.Context, plotID string, input PlotInput) (domain.PlotRecord, error) {
	current, err := s.repo.GetPlot(ctx, strings.TrimSpace(plotID))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return domain.PlotRecord{}, ErrNotFound
		}
		return domain.PlotRecord{}, ErrServiceUnavailable
	}
	if trimmed := strings.TrimSpace(input.OrchardID); trimmed != "" {
		current.OrchardID = trimmed
	}
	if trimmed := strings.TrimSpace(input.PlotName); trimmed != "" {
		current.PlotName = trimmed
	}
	if input.Status != "" {
		current.Status = input.Status
	}
	current.UpdatedAt = s.nowFn()
	updated, err := s.repo.UpdatePlot(ctx, current)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return domain.PlotRecord{}, ErrNotFound
		}
		return domain.PlotRecord{}, ErrServiceUnavailable
	}
	return updated, nil
}

func (s *PlotService) Archive(ctx context.Context, plotID string) (domain.PlotRecord, error) {
	return s.Update(ctx, plotID, PlotInput{Status: domain.ResourceStatusArchived})
}

func SeedDefaultResources(ctx context.Context, repo SeedRepo) error {
	if repo == nil {
		return nil
	}
	count, err := repo.CountOrchards(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	now := time.Now().UTC()
	defaults := []struct {
		orchard domain.OrchardRecord
		plots   []domain.PlotRecord
	}{
		{
			orchard: domain.OrchardRecord{OrchardID: "orchard-demo-01", OrchardName: "荔枝示范园", Status: domain.ResourceStatusActive, CreatedAt: now, UpdatedAt: now},
			plots: []domain.PlotRecord{
				{PlotID: "plot-a01", OrchardID: "orchard-demo-01", PlotName: "A1 区", Status: domain.ResourceStatusActive, CreatedAt: now, UpdatedAt: now},
				{PlotID: "plot-a02", OrchardID: "orchard-demo-01", PlotName: "A2 区", Status: domain.ResourceStatusActive, CreatedAt: now, UpdatedAt: now},
			},
		},
		{
			orchard: domain.OrchardRecord{OrchardID: "orchard-east-02", OrchardName: "东麓果园", Status: domain.ResourceStatusActive, CreatedAt: now, UpdatedAt: now},
			plots: []domain.PlotRecord{
				{PlotID: "plot-e01", OrchardID: "orchard-east-02", PlotName: "东坡 1 号地块", Status: domain.ResourceStatusActive, CreatedAt: now, UpdatedAt: now},
				{PlotID: "plot-e02", OrchardID: "orchard-east-02", PlotName: "东坡 2 号地块", Status: domain.ResourceStatusActive, CreatedAt: now, UpdatedAt: now},
			},
		},
		{
			orchard: domain.OrchardRecord{OrchardID: "orchard-north-03", OrchardName: "北山合作社果园", Status: domain.ResourceStatusActive, CreatedAt: now, UpdatedAt: now},
			plots: []domain.PlotRecord{
				{PlotID: "plot-n01", OrchardID: "orchard-north-03", PlotName: "北山上层区", Status: domain.ResourceStatusActive, CreatedAt: now, UpdatedAt: now},
				{PlotID: "plot-n02", OrchardID: "orchard-north-03", PlotName: "北山下层区", Status: domain.ResourceStatusActive, CreatedAt: now, UpdatedAt: now},
			},
		},
	}
	for _, item := range defaults {
		if err := repo.CreateOrchardIfNotExists(ctx, item.orchard); err != nil {
			return err
		}
		for _, plot := range item.plots {
			if err := repo.CreatePlotIfNotExists(ctx, plot); err != nil {
				return err
			}
		}
	}
	return nil
}

func normalizeOrchardInput(input OrchardInput, now time.Time, setID bool) (domain.OrchardRecord, error) {
	record := domain.OrchardRecord{
		OrchardID:   strings.TrimSpace(input.OrchardID),
		OrchardName: strings.TrimSpace(input.OrchardName),
		Status:      input.Status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if !setID {
		record.OrchardID = ""
	}
	if record.OrchardID == "" || record.OrchardName == "" {
		return domain.OrchardRecord{}, ErrInvalidRequest
	}
	if record.Status == "" {
		record.Status = domain.ResourceStatusActive
	}
	return record, nil
}

func normalizePlotInput(input PlotInput, now time.Time, setID bool) (domain.PlotRecord, error) {
	record := domain.PlotRecord{
		PlotID:    strings.TrimSpace(input.PlotID),
		OrchardID: strings.TrimSpace(input.OrchardID),
		PlotName:  strings.TrimSpace(input.PlotName),
		Status:    input.Status,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if !setID {
		record.PlotID = ""
	}
	if record.PlotID == "" || record.OrchardID == "" || record.PlotName == "" {
		return domain.PlotRecord{}, ErrInvalidRequest
	}
	if record.Status == "" {
		record.Status = domain.ResourceStatusActive
	}
	return record, nil
}
