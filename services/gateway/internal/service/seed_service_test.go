package service

import (
	"context"
	"errors"
	"testing"

	"github.com/lychee-ripe/gateway/internal/domain"
)

func TestSeedDefaultResourcesSkipsWhenOrchardsExist(t *testing.T) {
	t.Parallel()

	repo := &fakeSeedRepo{count: 1}
	if err := SeedDefaultResources(context.Background(), repo); err != nil {
		t.Fatalf("SeedDefaultResources returned error: %v", err)
	}
	if repo.createdOrchards != 0 || repo.createdPlots != 0 {
		t.Fatalf("unexpected seed writes: orchards=%d plots=%d", repo.createdOrchards, repo.createdPlots)
	}
}

func TestSeedDefaultResourcesWritesDefaultsWhenEmpty(t *testing.T) {
	t.Parallel()

	repo := &fakeSeedRepo{count: 0}
	if err := SeedDefaultResources(context.Background(), repo); err != nil {
		t.Fatalf("SeedDefaultResources returned error: %v", err)
	}
	if repo.createdOrchards != 3 {
		t.Fatalf("created orchards = %d, want 3", repo.createdOrchards)
	}
	if repo.createdPlots != 6 {
		t.Fatalf("created plots = %d, want 6", repo.createdPlots)
	}
}

func TestSeedDefaultResourcesPropagatesCountError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("count failed")
	repo := &fakeSeedRepo{countErr: wantErr}
	err := SeedDefaultResources(context.Background(), repo)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

type fakeSeedRepo struct {
	count           int64
	countErr        error
	createdOrchards int
	createdPlots    int
}

func (f *fakeSeedRepo) CountOrchards(_ context.Context) (int64, error) {
	if f.countErr != nil {
		return 0, f.countErr
	}
	return f.count, nil
}

func (f *fakeSeedRepo) CreateOrchardIfNotExists(_ context.Context, orchard domain.OrchardRecord) error {
	f.createdOrchards += 1
	if orchard.OrchardID == "" {
		return errors.New("missing orchard id")
	}
	return nil
}

func (f *fakeSeedRepo) CreatePlotIfNotExists(_ context.Context, plot domain.PlotRecord) error {
	f.createdPlots += 1
	if plot.PlotID == "" {
		return errors.New("missing plot id")
	}
	return nil
}
