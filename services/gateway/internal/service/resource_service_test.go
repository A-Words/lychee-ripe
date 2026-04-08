package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/repository"
)

func TestNormalizeOrchardInputDefaultsActiveStatus(t *testing.T) {
	t.Parallel()

	record, err := normalizeOrchardInput(OrchardInput{
		OrchardID:   "orchard-1",
		OrchardName: "Demo Orchard",
	}, time.Now().UTC(), true)
	if err != nil {
		t.Fatalf("normalizeOrchardInput returned error: %v", err)
	}
	if record.Status != domain.ResourceStatusActive {
		t.Fatalf("status = %q, want active", record.Status)
	}
}

func TestNormalizeOrchardInputRejectsInvalidStatus(t *testing.T) {
	t.Parallel()

	_, err := normalizeOrchardInput(OrchardInput{
		OrchardID:   "orchard-1",
		OrchardName: "Demo Orchard",
		Status:      domain.ResourceStatus("archive"),
	}, time.Now().UTC(), true)
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v, want ErrInvalidRequest", err)
	}
}

func TestNormalizePlotInputDefaultsActiveStatus(t *testing.T) {
	t.Parallel()

	record, err := normalizePlotInput(PlotInput{
		PlotID:    "plot-1",
		OrchardID: "orchard-1",
		PlotName:  "A1",
	}, time.Now().UTC(), true)
	if err != nil {
		t.Fatalf("normalizePlotInput returned error: %v", err)
	}
	if record.Status != domain.ResourceStatusActive {
		t.Fatalf("status = %q, want active", record.Status)
	}
}

func TestNormalizePlotInputRejectsInvalidStatus(t *testing.T) {
	t.Parallel()

	_, err := normalizePlotInput(PlotInput{
		PlotID:    "plot-1",
		OrchardID: "orchard-1",
		PlotName:  "A1",
		Status:    domain.ResourceStatus("archive"),
	}, time.Now().UTC(), true)
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v, want ErrInvalidRequest", err)
	}
}

func TestOrchardServiceUpdateRejectsInvalidStatus(t *testing.T) {
	t.Parallel()

	repo := &fakeOrchardRepo{
		record: domain.OrchardRecord{
			OrchardID:   "orchard-1",
			OrchardName: "Demo Orchard",
			Status:      domain.ResourceStatusActive,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		},
	}
	svc := NewOrchardService(repo)

	_, err := svc.Update(context.Background(), "orchard-1", OrchardInput{
		OrchardName: "Demo Orchard",
		Status:      domain.ResourceStatus("archive"),
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v, want ErrInvalidRequest", err)
	}
	if repo.updateCalled {
		t.Fatal("expected invalid status to be rejected before repository update")
	}
}

func TestPlotServiceUpdateRejectsInvalidStatus(t *testing.T) {
	t.Parallel()

	repo := &fakePlotRepo{
		record: domain.PlotRecord{
			PlotID:    "plot-1",
			OrchardID: "orchard-1",
			PlotName:  "A1",
			Status:    domain.ResourceStatusActive,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
	}
	svc := NewPlotService(repo)

	_, err := svc.Update(context.Background(), "plot-1", PlotInput{
		Status: domain.ResourceStatus("archive"),
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v, want ErrInvalidRequest", err)
	}
	if repo.updateCalled {
		t.Fatal("expected invalid status to be rejected before repository update")
	}
}

type fakeOrchardRepo struct {
	record       domain.OrchardRecord
	updateCalled bool
}

func (f *fakeOrchardRepo) ListOrchards(_ context.Context, _ bool) ([]domain.OrchardRecord, error) {
	return []domain.OrchardRecord{f.record}, nil
}

func (f *fakeOrchardRepo) CreateOrchard(_ context.Context, orchard domain.OrchardRecord) (domain.OrchardRecord, error) {
	f.record = orchard
	return orchard, nil
}

func (f *fakeOrchardRepo) UpdateOrchard(_ context.Context, orchard domain.OrchardRecord) (domain.OrchardRecord, error) {
	f.updateCalled = true
	f.record = orchard
	return orchard, nil
}

func (f *fakeOrchardRepo) GetOrchard(_ context.Context, orchardID string) (domain.OrchardRecord, error) {
	if orchardID != f.record.OrchardID {
		return domain.OrchardRecord{}, repository.ErrNotFound
	}
	return f.record, nil
}

type fakePlotRepo struct {
	record       domain.PlotRecord
	updateCalled bool
}

func (f *fakePlotRepo) ListPlots(_ context.Context, _ string, _ bool) ([]domain.PlotRecord, error) {
	return []domain.PlotRecord{f.record}, nil
}

func (f *fakePlotRepo) CreatePlot(_ context.Context, plot domain.PlotRecord) (domain.PlotRecord, error) {
	f.record = plot
	return plot, nil
}

func (f *fakePlotRepo) UpdatePlot(_ context.Context, plot domain.PlotRecord) (domain.PlotRecord, error) {
	f.updateCalled = true
	f.record = plot
	return plot, nil
}

func (f *fakePlotRepo) GetPlot(_ context.Context, plotID string) (domain.PlotRecord, error) {
	if plotID != f.record.PlotID {
		return domain.PlotRecord{}, repository.ErrNotFound
	}
	return f.record, nil
}
