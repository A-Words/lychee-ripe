package service

import (
	"context"
	"errors"
	"fmt"
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
		OrchardName:        "Demo Orchard",
		Status:             domain.ResourceStatus("archive"),
		OrchardNamePresent: true,
		StatusPresent:      true,
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v, want ErrInvalidRequest", err)
	}
	if repo.updateCalled {
		t.Fatal("expected invalid status to be rejected before repository update")
	}
}

func TestOrchardServiceUpdatePreservesArchivedStatusWhenOmitted(t *testing.T) {
	t.Parallel()

	repo := &fakeOrchardRepo{
		record: domain.OrchardRecord{
			OrchardID:   "orchard-1",
			OrchardName: "Archived Orchard",
			Status:      domain.ResourceStatusArchived,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		},
	}
	svc := NewOrchardService(repo)

	updated, err := svc.Update(context.Background(), "orchard-1", OrchardInput{
		OrchardName:        "Archived Orchard Renamed",
		OrchardNamePresent: true,
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Status != domain.ResourceStatusArchived {
		t.Fatalf("status = %q, want archived", updated.Status)
	}
	if repo.record.Status != domain.ResourceStatusArchived {
		t.Fatalf("stored status = %q, want archived", repo.record.Status)
	}
}

func TestOrchardServiceUpdatePreservesActiveStatusWhenOmitted(t *testing.T) {
	t.Parallel()

	repo := &fakeOrchardRepo{
		record: domain.OrchardRecord{
			OrchardID:   "orchard-1",
			OrchardName: "Active Orchard",
			Status:      domain.ResourceStatusActive,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		},
	}
	svc := NewOrchardService(repo)

	updated, err := svc.Update(context.Background(), "orchard-1", OrchardInput{
		OrchardName:        "Active Orchard Renamed",
		OrchardNamePresent: true,
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Status != domain.ResourceStatusActive {
		t.Fatalf("status = %q, want active", updated.Status)
	}
	if repo.record.Status != domain.ResourceStatusActive {
		t.Fatalf("stored status = %q, want active", repo.record.Status)
	}
}

func TestOrchardServiceArchiveRejectsWhenActivePlotsExist(t *testing.T) {
	t.Parallel()

	repo := &fakeOrchardRepo{
		record: domain.OrchardRecord{
			OrchardID:   "orchard-1",
			OrchardName: "Demo Orchard",
			Status:      domain.ResourceStatusActive,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		},
		activePlotCount: 2,
	}
	svc := NewOrchardService(repo)

	_, err := svc.Archive(context.Background(), "orchard-1")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("error = %v, want ErrConflict", err)
	}
	if !repo.archiveCalled {
		t.Fatal("expected archive path to call guarded repository method")
	}
}

func TestOrchardServiceUpdateRejectsArchivingWhenActivePlotsExist(t *testing.T) {
	t.Parallel()

	repo := &fakeOrchardRepo{
		record: domain.OrchardRecord{
			OrchardID:   "orchard-1",
			OrchardName: "Demo Orchard",
			Status:      domain.ResourceStatusActive,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		},
		activePlotCount: 1,
	}
	svc := NewOrchardService(repo)

	_, err := svc.Update(context.Background(), "orchard-1", OrchardInput{
		OrchardName:        "Demo Orchard",
		Status:             domain.ResourceStatusArchived,
		OrchardNamePresent: true,
		StatusPresent:      true,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("error = %v, want ErrConflict", err)
	}
	if !repo.archiveCalled {
		t.Fatal("expected archiving update to use guarded repository method")
	}
}

func TestOrchardServiceArchiveSucceedsWhenNoActivePlotsExist(t *testing.T) {
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

	updated, err := svc.Archive(context.Background(), "orchard-1")
	if err != nil {
		t.Fatalf("Archive returned error: %v", err)
	}
	if updated.Status != domain.ResourceStatusArchived {
		t.Fatalf("status = %q, want archived", updated.Status)
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
		Status:        domain.ResourceStatus("archive"),
		StatusPresent: true,
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v, want ErrInvalidRequest", err)
	}
	if repo.updateCalled {
		t.Fatal("expected invalid status to be rejected before repository update")
	}
}

func TestPlotServiceCreateRejectsUnknownOrchard(t *testing.T) {
	t.Parallel()

	repo := &fakePlotRepo{}
	svc := NewPlotService(repo)

	_, err := svc.Create(context.Background(), PlotInput{
		PlotID:    "plot-1",
		OrchardID: "missing-orchard",
		PlotName:  "A1",
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("error = %v, want ErrConflict", err)
	}
	if repo.createCalled {
		t.Fatal("expected guarded create to reject missing orchard")
	}
}

func TestPlotServiceCreateRejectsActivePlotUnderArchivedOrchard(t *testing.T) {
	t.Parallel()

	repo := &fakePlotRepo{
		orchards: map[string]domain.OrchardRecord{
			"orchard-1": {
				OrchardID:   "orchard-1",
				OrchardName: "Archived Orchard",
				Status:      domain.ResourceStatusArchived,
				CreatedAt:   time.Now().UTC(),
				UpdatedAt:   time.Now().UTC(),
			},
		},
	}
	svc := NewPlotService(repo)

	_, err := svc.Create(context.Background(), PlotInput{
		PlotID:    "plot-1",
		OrchardID: "orchard-1",
		PlotName:  "A1",
		Status:    domain.ResourceStatusActive,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("error = %v, want ErrConflict", err)
	}
	if repo.createCalled {
		t.Fatal("expected guarded create to reject archived orchard")
	}
}

func TestPlotServiceUpdateRejectsMovingToUnknownOrchard(t *testing.T) {
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
		orchards: map[string]domain.OrchardRecord{
			"orchard-1": {
				OrchardID:   "orchard-1",
				OrchardName: "Demo Orchard",
				Status:      domain.ResourceStatusActive,
				CreatedAt:   time.Now().UTC(),
				UpdatedAt:   time.Now().UTC(),
			},
		},
	}
	svc := NewPlotService(repo)

	_, err := svc.Update(context.Background(), "plot-1", PlotInput{
		OrchardID:        "missing-orchard",
		OrchardIDPresent: true,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("error = %v, want ErrConflict", err)
	}
	if repo.updateCalled {
		t.Fatal("expected guarded update to reject invalid orchard move")
	}
}

func TestPlotServiceUpdateRejectsActivePlotUnderArchivedOrchard(t *testing.T) {
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
		orchards: map[string]domain.OrchardRecord{
			"orchard-1": {
				OrchardID:   "orchard-1",
				OrchardName: "Active Orchard",
				Status:      domain.ResourceStatusActive,
				CreatedAt:   time.Now().UTC(),
				UpdatedAt:   time.Now().UTC(),
			},
			"orchard-2": {
				OrchardID:   "orchard-2",
				OrchardName: "Archived Orchard",
				Status:      domain.ResourceStatusArchived,
				CreatedAt:   time.Now().UTC(),
				UpdatedAt:   time.Now().UTC(),
			},
		},
	}
	svc := NewPlotService(repo)

	_, err := svc.Update(context.Background(), "plot-1", PlotInput{
		OrchardID:        "orchard-2",
		OrchardIDPresent: true,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("error = %v, want ErrConflict", err)
	}
	if repo.updateCalled {
		t.Fatal("expected guarded update to reject archived orchard move")
	}
}

func TestPlotServiceCreateAcceptsExistingOrchard(t *testing.T) {
	t.Parallel()

	repo := &fakePlotRepo{
		orchards: map[string]domain.OrchardRecord{
			"orchard-1": {
				OrchardID:   "orchard-1",
				OrchardName: "Demo Orchard",
				Status:      domain.ResourceStatusActive,
				CreatedAt:   time.Now().UTC(),
				UpdatedAt:   time.Now().UTC(),
			},
		},
	}
	svc := NewPlotService(repo)

	created, err := svc.Create(context.Background(), PlotInput{
		PlotID:    "plot-1",
		OrchardID: "orchard-1",
		PlotName:  "A1",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.OrchardID != "orchard-1" {
		t.Fatalf("orchard_id = %q, want orchard-1", created.OrchardID)
	}
}

type fakeOrchardRepo struct {
	record          domain.OrchardRecord
	updateCalled    bool
	archiveCalled   bool
	activePlotCount int64
}

func (f *fakeOrchardRepo) ListOrchards(_ context.Context, _ bool) ([]domain.OrchardRecord, error) {
	return []domain.OrchardRecord{f.record}, nil
}

func (f *fakeOrchardRepo) CreateOrchard(_ context.Context, orchard domain.OrchardRecord) (domain.OrchardRecord, error) {
	f.record = orchard
	return orchard, nil
}

func (f *fakeOrchardRepo) UpdateOrchard(_ context.Context, _ time.Time, orchard domain.OrchardRecord) (domain.OrchardRecord, error) {
	f.updateCalled = true
	f.record = orchard
	return orchard, nil
}

func (f *fakeOrchardRepo) ArchiveOrchard(_ context.Context, _ time.Time, orchard domain.OrchardRecord) (domain.OrchardRecord, error) {
	f.archiveCalled = true
	if f.activePlotCount > 0 {
		return domain.OrchardRecord{}, fmt.Errorf("%w: orchard has active plots; archive or move plots first", repository.ErrInvalidState)
	}
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
	createCalled bool
	updateCalled bool
	orchards     map[string]domain.OrchardRecord
}

func (f *fakePlotRepo) ListPlots(_ context.Context, _ string, _ bool) ([]domain.PlotRecord, error) {
	return []domain.PlotRecord{f.record}, nil
}

func (f *fakePlotRepo) CreatePlot(_ context.Context, plot domain.PlotRecord) (domain.PlotRecord, error) {
	f.record = plot
	return plot, nil
}

func (f *fakePlotRepo) CreatePlotGuarded(_ context.Context, plot domain.PlotRecord) (domain.PlotRecord, error) {
	orchard, ok := f.orchards[plot.OrchardID]
	if !ok {
		return domain.PlotRecord{}, fmt.Errorf("%w: orchard_id does not reference an existing orchard", repository.ErrInvalidState)
	}
	if orchard.Status == domain.ResourceStatusArchived && plot.Status == domain.ResourceStatusActive {
		return domain.PlotRecord{}, fmt.Errorf("%w: active plots cannot belong to archived orchards", repository.ErrInvalidState)
	}
	f.createCalled = true
	f.record = plot
	return plot, nil
}

func (f *fakePlotRepo) UpdatePlot(_ context.Context, _ time.Time, plot domain.PlotRecord) (domain.PlotRecord, error) {
	f.updateCalled = true
	f.record = plot
	return plot, nil
}

func (f *fakePlotRepo) UpdatePlotGuarded(_ context.Context, _ time.Time, plot domain.PlotRecord) (domain.PlotRecord, error) {
	orchard, ok := f.orchards[plot.OrchardID]
	if !ok {
		return domain.PlotRecord{}, fmt.Errorf("%w: orchard_id does not reference an existing orchard", repository.ErrInvalidState)
	}
	if orchard.Status == domain.ResourceStatusArchived && plot.Status == domain.ResourceStatusActive {
		return domain.PlotRecord{}, fmt.Errorf("%w: active plots cannot belong to archived orchards", repository.ErrInvalidState)
	}
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
