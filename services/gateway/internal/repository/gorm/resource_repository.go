package gorm

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	_ repository.OrchardRepository = (*Repository)(nil)
	_ repository.PlotRepository    = (*Repository)(nil)
	_ repository.SeedRepository    = (*Repository)(nil)
)

// ---------------------------------------------------------------------------
// Orchard
// ---------------------------------------------------------------------------

func (r *Repository) ListOrchards(ctx context.Context, includeArchived bool) ([]domain.OrchardRecord, error) {
	query := r.db.WithContext(ctx).Order("created_at ASC")
	if !includeArchived {
		query = query.Where("status = ?", string(domain.ResourceStatusActive))
	}
	var models []OrchardModel
	if err := query.Find(&models).Error; err != nil {
		return nil, mapGormErr(err)
	}
	out := make([]domain.OrchardRecord, 0, len(models))
	for _, item := range models {
		out = append(out, orchardModelToDomain(item))
	}
	return out, nil
}

func (r *Repository) CreateOrchard(ctx context.Context, orchard domain.OrchardRecord) (domain.OrchardRecord, error) {
	model := orchardModelFromDomain(orchard)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.OrchardRecord{}, mapGormErr(err)
	}
	return orchardModelToDomain(model), nil
}

func (r *Repository) UpdateOrchard(ctx context.Context, expectedUpdatedAt time.Time, orchard domain.OrchardRecord) (domain.OrchardRecord, error) {
	model := orchardModelFromDomain(orchard)
	res := r.db.WithContext(ctx).Model(&OrchardModel{}).Where("orchard_id = ?", model.OrchardID).Where("updated_at = ?", normalizeTime(expectedUpdatedAt)).Updates(map[string]any{
		"orchard_name": model.OrchardName,
		"status":       model.Status,
		"updated_at":   model.UpdatedAt,
	})
	if res.Error != nil {
		return domain.OrchardRecord{}, mapGormErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.OrchardRecord{}, classifyCASConflict(r.db.WithContext(ctx), "orchards", "orchard_id", model.OrchardID)
	}
	return r.GetOrchard(ctx, model.OrchardID)
}

func (r *Repository) ArchiveOrchard(ctx context.Context, expectedUpdatedAt time.Time, orchard domain.OrchardRecord) (domain.OrchardRecord, error) {
	model := orchardModelFromDomain(orchard)

	var err error
	switch r.db.Dialector.Name() {
	case "postgres":
		err = r.archiveOrchardPostgres(ctx, expectedUpdatedAt, model)
	default:
		err = retrySQLiteBusy(ctx, func() error { return r.archiveOrchardSQLite(ctx, expectedUpdatedAt, model) })
	}
	if err != nil {
		return domain.OrchardRecord{}, err
	}
	return r.GetOrchard(ctx, model.OrchardID)
}

func (r *Repository) GetOrchard(ctx context.Context, orchardID string) (domain.OrchardRecord, error) {
	var model OrchardModel
	if err := r.db.WithContext(ctx).Where("orchard_id = ?", strings.TrimSpace(orchardID)).First(&model).Error; err != nil {
		return domain.OrchardRecord{}, mapGormErr(err)
	}
	return orchardModelToDomain(model), nil
}

func (r *Repository) CountOrchards(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&OrchardModel{}).Count(&count).Error; err != nil {
		return 0, mapGormErr(err)
	}
	return count, nil
}

func (r *Repository) CreateOrchardIfNotExists(ctx context.Context, orchard domain.OrchardRecord) error {
	model := orchardModelFromDomain(orchard)
	err := r.db.WithContext(ctx).FirstOrCreate(&model, OrchardModel{OrchardID: model.OrchardID}).Error
	return mapGormErr(err)
}

// ---------------------------------------------------------------------------
// Plot
// ---------------------------------------------------------------------------

func (r *Repository) CountActivePlots(ctx context.Context, orchardID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&PlotModel{}).
		Where("orchard_id = ? AND status = ?", strings.TrimSpace(orchardID), string(domain.ResourceStatusActive)).
		Count(&count).Error; err != nil {
		return 0, mapGormErr(err)
	}
	return count, nil
}

func (r *Repository) ListPlots(ctx context.Context, orchardID string, includeArchived bool) ([]domain.PlotRecord, error) {
	query := r.db.WithContext(ctx).Order("created_at ASC")
	if trimmed := strings.TrimSpace(orchardID); trimmed != "" {
		query = query.Where("orchard_id = ?", trimmed)
	}
	if !includeArchived {
		query = query.Where("status = ?", string(domain.ResourceStatusActive))
	}
	var models []PlotModel
	if err := query.Find(&models).Error; err != nil {
		return nil, mapGormErr(err)
	}
	out := make([]domain.PlotRecord, 0, len(models))
	for _, item := range models {
		out = append(out, plotModelToDomain(item))
	}
	return out, nil
}

func (r *Repository) CreatePlot(ctx context.Context, plot domain.PlotRecord) (domain.PlotRecord, error) {
	model := plotModelFromDomain(plot)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.PlotRecord{}, mapGormErr(err)
	}
	return plotModelToDomain(model), nil
}

func (r *Repository) CreatePlotGuarded(ctx context.Context, plot domain.PlotRecord) (domain.PlotRecord, error) {
	model := plotModelFromDomain(plot)

	var err error
	switch r.db.Dialector.Name() {
	case "postgres":
		err = r.createPlotGuardedPostgres(ctx, model)
	default:
		err = retrySQLiteBusy(ctx, func() error { return r.createPlotGuardedSQLite(ctx, model) })
	}
	if err != nil {
		return domain.PlotRecord{}, err
	}
	return r.GetPlot(ctx, model.PlotID)
}

func (r *Repository) UpdatePlot(ctx context.Context, expectedUpdatedAt time.Time, plot domain.PlotRecord) (domain.PlotRecord, error) {
	model := plotModelFromDomain(plot)
	res := r.db.WithContext(ctx).Model(&PlotModel{}).Where("plot_id = ?", model.PlotID).Where("updated_at = ?", normalizeTime(expectedUpdatedAt)).Updates(map[string]any{
		"orchard_id": model.OrchardID,
		"plot_name":  model.PlotName,
		"status":     model.Status,
		"updated_at": model.UpdatedAt,
	})
	if res.Error != nil {
		return domain.PlotRecord{}, mapGormErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.PlotRecord{}, classifyCASConflict(r.db.WithContext(ctx), "plots", "plot_id", model.PlotID)
	}
	return r.GetPlot(ctx, model.PlotID)
}

func (r *Repository) UpdatePlotGuarded(ctx context.Context, expectedUpdatedAt time.Time, plot domain.PlotRecord) (domain.PlotRecord, error) {
	model := plotModelFromDomain(plot)

	var err error
	switch r.db.Dialector.Name() {
	case "postgres":
		err = r.updatePlotGuardedPostgres(ctx, expectedUpdatedAt, model)
	default:
		err = retrySQLiteBusy(ctx, func() error { return r.updatePlotGuardedSQLite(ctx, expectedUpdatedAt, model) })
	}
	if err != nil {
		return domain.PlotRecord{}, err
	}
	return r.GetPlot(ctx, model.PlotID)
}

func (r *Repository) GetPlot(ctx context.Context, plotID string) (domain.PlotRecord, error) {
	var model PlotModel
	if err := r.db.WithContext(ctx).Where("plot_id = ?", strings.TrimSpace(plotID)).First(&model).Error; err != nil {
		return domain.PlotRecord{}, mapGormErr(err)
	}
	return plotModelToDomain(model), nil
}

func (r *Repository) CreatePlotIfNotExists(ctx context.Context, plot domain.PlotRecord) error {
	model := plotModelFromDomain(plot)
	err := r.db.WithContext(ctx).FirstOrCreate(&model, PlotModel{PlotID: model.PlotID}).Error
	return mapGormErr(err)
}

// ---------------------------------------------------------------------------
// Internal helpers — orchard
// ---------------------------------------------------------------------------

func (r *Repository) archiveOrchardPostgres(ctx context.Context, expectedUpdatedAt time.Time, model OrchardModel) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := lockOrchardByID(tx, model.OrchardID); err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&PlotModel{}).
			Where("orchard_id = ? AND status = ?", model.OrchardID, string(domain.ResourceStatusActive)).
			Count(&count).Error; err != nil {
			return mapGormErr(err)
		}
		if count > 0 {
			return fmt.Errorf("%w: orchard has active plots; archive or move plots first", repository.ErrInvalidState)
		}
		return updateOrchardRecord(tx, expectedUpdatedAt, model)
	})
}

func (r *Repository) archiveOrchardSQLite(ctx context.Context, expectedUpdatedAt time.Time, model OrchardModel) error {
	res := r.db.WithContext(ctx).
		Model(&OrchardModel{}).
		Where("orchard_id = ?", model.OrchardID).
		Where("updated_at = ?", normalizeTime(expectedUpdatedAt)).
		Where(
			`NOT EXISTS (
				SELECT 1 FROM plots
				WHERE plots.orchard_id = orchards.orchard_id AND plots.status = ?
			)`,
			string(domain.ResourceStatusActive),
		).
		Updates(map[string]any{
			"orchard_name": model.OrchardName,
			"status":       model.Status,
			"updated_at":   model.UpdatedAt,
		})
	if res.Error != nil {
		return mapGormErr(res.Error)
	}
	if res.RowsAffected > 0 {
		return nil
	}
	if err := classifyBlockedOrchardArchive(r.db.WithContext(ctx), model.OrchardID); err != nil {
		if errors.Is(err, repository.ErrInvalidState) || errors.Is(err, repository.ErrNotFound) {
			return err
		}
	}
	return classifyCASConflict(r.db.WithContext(ctx), "orchards", "orchard_id", model.OrchardID)
}

func updateOrchardRecord(tx *gorm.DB, expectedUpdatedAt time.Time, model OrchardModel) error {
	res := tx.Model(&OrchardModel{}).
		Where("orchard_id = ?", model.OrchardID).
		Where("updated_at = ?", normalizeTime(expectedUpdatedAt)).
		Updates(map[string]any{
			"orchard_name": model.OrchardName,
			"status":       model.Status,
			"updated_at":   model.UpdatedAt,
		})
	if res.Error != nil {
		return mapGormErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return classifyCASConflict(tx, "orchards", "orchard_id", model.OrchardID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Internal helpers — plot
// ---------------------------------------------------------------------------

func (r *Repository) createPlotGuardedPostgres(ctx context.Context, model PlotModel) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		target, err := lockOrchardByID(tx, model.OrchardID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return fmt.Errorf("%w: orchard_id does not reference an existing orchard", repository.ErrInvalidState)
			}
			return err
		}
		if target.Status == string(domain.ResourceStatusArchived) && model.Status == string(domain.ResourceStatusActive) {
			return fmt.Errorf("%w: active plots cannot belong to archived orchards", repository.ErrInvalidState)
		}
		if err := tx.Create(&model).Error; err != nil {
			return mapGormErr(err)
		}
		return nil
	})
}

func (r *Repository) createPlotGuardedSQLite(ctx context.Context, model PlotModel) error {
	res := r.db.WithContext(ctx).Exec(
		`INSERT INTO plots (plot_id, orchard_id, plot_name, status, created_at, updated_at)
		 SELECT ?, ?, ?, ?, ?, ?
		 WHERE EXISTS (
		 	SELECT 1 FROM orchards
		 	WHERE orchard_id = ?
		 	AND (? <> ? OR status = ?)
		 )`,
		model.PlotID,
		model.OrchardID,
		model.PlotName,
		model.Status,
		model.CreatedAt,
		model.UpdatedAt,
		model.OrchardID,
		model.Status,
		string(domain.ResourceStatusActive),
		string(domain.ResourceStatusActive),
	)
	if res.Error != nil {
		return mapGormErr(res.Error)
	}
	if res.RowsAffected > 0 {
		return nil
	}
	return classifyBlockedPlotWrite(r.db.WithContext(ctx), "", model.OrchardID, model.Status)
}

func (r *Repository) updatePlotGuardedPostgres(ctx context.Context, expectedUpdatedAt time.Time, model PlotModel) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, err := lockPlotByID(tx, model.PlotID)
		if err != nil {
			return err
		}

		orchards, err := lockOrchards(tx, current.OrchardID, model.OrchardID)
		if err != nil {
			return err
		}
		target, ok := orchards[model.OrchardID]
		if !ok {
			return fmt.Errorf("%w: orchard_id does not reference an existing orchard", repository.ErrInvalidState)
		}
		if target.Status == string(domain.ResourceStatusArchived) && model.Status == string(domain.ResourceStatusActive) {
			return fmt.Errorf("%w: active plots cannot belong to archived orchards", repository.ErrInvalidState)
		}
		return updatePlotRecord(tx, expectedUpdatedAt, model)
	})
}

func (r *Repository) updatePlotGuardedSQLite(ctx context.Context, expectedUpdatedAt time.Time, model PlotModel) error {
	res := r.db.WithContext(ctx).
		Model(&PlotModel{}).
		Where("plot_id = ?", model.PlotID).
		Where("updated_at = ?", normalizeTime(expectedUpdatedAt)).
		Where(
			`EXISTS (
				SELECT 1 FROM orchards
				WHERE orchard_id = ?
				AND (? <> ? OR status = ?)
			)`,
			model.OrchardID,
			model.Status,
			string(domain.ResourceStatusActive),
			string(domain.ResourceStatusActive),
		).
		Updates(map[string]any{
			"orchard_id": model.OrchardID,
			"plot_name":  model.PlotName,
			"status":     model.Status,
			"updated_at": model.UpdatedAt,
		})
	if res.Error != nil {
		return mapGormErr(res.Error)
	}
	if res.RowsAffected > 0 {
		return nil
	}
	if err := classifyBlockedPlotWrite(r.db.WithContext(ctx), model.PlotID, model.OrchardID, model.Status); err != nil {
		if errors.Is(err, repository.ErrInvalidState) || errors.Is(err, repository.ErrNotFound) {
			return err
		}
	}
	return classifyCASConflict(r.db.WithContext(ctx), "plots", "plot_id", model.PlotID)
}

func updatePlotRecord(tx *gorm.DB, expectedUpdatedAt time.Time, model PlotModel) error {
	res := tx.Model(&PlotModel{}).
		Where("plot_id = ?", model.PlotID).
		Where("updated_at = ?", normalizeTime(expectedUpdatedAt)).
		Updates(map[string]any{
			"orchard_id": model.OrchardID,
			"plot_name":  model.PlotName,
			"status":     model.Status,
			"updated_at": model.UpdatedAt,
		})
	if res.Error != nil {
		return mapGormErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return classifyCASConflict(tx, "plots", "plot_id", model.PlotID)
	}
	return nil
}

func classifyCASConflict(db *gorm.DB, table string, keyColumn string, keyValue string) error {
	var count int64
	if err := db.Table(table).Where(fmt.Sprintf("%s = ?", keyColumn), strings.TrimSpace(keyValue)).Count(&count).Error; err != nil {
		return mapGormErr(err)
	}
	if count == 0 {
		return repository.ErrNotFound
	}
	return repository.ErrConflict
}

// ---------------------------------------------------------------------------
// Shared lock & classify helpers
// ---------------------------------------------------------------------------

func lockOrchardByID(tx *gorm.DB, orchardID string) (OrchardModel, error) {
	var orchard OrchardModel
	query := tx.Where("orchard_id = ?", strings.TrimSpace(orchardID))
	if tx.Dialector.Name() == "postgres" {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := query.First(&orchard).Error; err != nil {
		return OrchardModel{}, mapGormErr(err)
	}
	return orchard, nil
}

func lockPlotByID(tx *gorm.DB, plotID string) (PlotModel, error) {
	var plot PlotModel
	query := tx.Where("plot_id = ?", strings.TrimSpace(plotID))
	if tx.Dialector.Name() == "postgres" {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := query.First(&plot).Error; err != nil {
		return PlotModel{}, mapGormErr(err)
	}
	return plot, nil
}

func lockOrchards(tx *gorm.DB, orchardIDs ...string) (map[string]OrchardModel, error) {
	unique := make(map[string]struct{}, len(orchardIDs))
	ordered := make([]string, 0, len(orchardIDs))
	for _, orchardID := range orchardIDs {
		trimmed := strings.TrimSpace(orchardID)
		if trimmed == "" {
			continue
		}
		if _, ok := unique[trimmed]; ok {
			continue
		}
		unique[trimmed] = struct{}{}
		ordered = append(ordered, trimmed)
	}
	sort.Strings(ordered)
	if len(ordered) == 0 {
		return map[string]OrchardModel{}, nil
	}

	query := tx.Where("orchard_id IN ?", ordered).Order("orchard_id ASC")
	if tx.Dialector.Name() == "postgres" {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var orchards []OrchardModel
	if err := query.Find(&orchards).Error; err != nil {
		return nil, mapGormErr(err)
	}

	out := make(map[string]OrchardModel, len(orchards))
	for _, orchard := range orchards {
		out[orchard.OrchardID] = orchard
	}
	return out, nil
}

func classifyBlockedOrchardArchive(db *gorm.DB, orchardID string) error {
	var orchard OrchardModel
	if err := db.Where("orchard_id = ?", strings.TrimSpace(orchardID)).First(&orchard).Error; err != nil {
		return mapGormErr(err)
	}
	var count int64
	if err := db.Model(&PlotModel{}).
		Where("orchard_id = ? AND status = ?", strings.TrimSpace(orchardID), string(domain.ResourceStatusActive)).
		Count(&count).Error; err != nil {
		return mapGormErr(err)
	}
	if count > 0 {
		return fmt.Errorf("%w: orchard has active plots; archive or move plots first", repository.ErrInvalidState)
	}
	return fmt.Errorf("%w: orchard archive precondition failed for %s", repository.ErrDBUnavailable, strings.TrimSpace(orchardID))
}

func classifyBlockedPlotWrite(db *gorm.DB, plotID string, orchardID string, plotStatus string) error {
	if trimmedPlotID := strings.TrimSpace(plotID); trimmedPlotID != "" {
		var plotCount int64
		if err := db.Model(&PlotModel{}).Where("plot_id = ?", trimmedPlotID).Count(&plotCount).Error; err != nil {
			return mapGormErr(err)
		}
		if plotCount == 0 {
			return repository.ErrNotFound
		}
	}

	var orchard OrchardModel
	if err := db.Where("orchard_id = ?", strings.TrimSpace(orchardID)).First(&orchard).Error; err != nil {
		if errors.Is(mapGormErr(err), repository.ErrNotFound) {
			return fmt.Errorf("%w: orchard_id does not reference an existing orchard", repository.ErrInvalidState)
		}
		return mapGormErr(err)
	}
	if orchard.Status == string(domain.ResourceStatusArchived) && plotStatus == string(domain.ResourceStatusActive) {
		return fmt.Errorf("%w: active plots cannot belong to archived orchards", repository.ErrInvalidState)
	}
	return fmt.Errorf("%w: plot write precondition failed for orchard %s", repository.ErrDBUnavailable, strings.TrimSpace(orchardID))
}

func orchardModelFromDomain(orchard domain.OrchardRecord) OrchardModel {
	return OrchardModel{
		OrchardID:   strings.TrimSpace(orchard.OrchardID),
		OrchardName: strings.TrimSpace(orchard.OrchardName),
		Status:      string(orchard.Status),
		CreatedAt:   normalizeTime(orchard.CreatedAt),
		UpdatedAt:   normalizeTime(orchard.UpdatedAt),
	}
}

func orchardModelToDomain(orchard OrchardModel) domain.OrchardRecord {
	return domain.OrchardRecord{
		OrchardID:   orchard.OrchardID,
		OrchardName: orchard.OrchardName,
		Status:      domain.ResourceStatus(orchard.Status),
		CreatedAt:   orchard.CreatedAt.UTC(),
		UpdatedAt:   orchard.UpdatedAt.UTC(),
	}
}

func plotModelFromDomain(plot domain.PlotRecord) PlotModel {
	return PlotModel{
		PlotID:    strings.TrimSpace(plot.PlotID),
		OrchardID: strings.TrimSpace(plot.OrchardID),
		PlotName:  strings.TrimSpace(plot.PlotName),
		Status:    string(plot.Status),
		CreatedAt: normalizeTime(plot.CreatedAt),
		UpdatedAt: normalizeTime(plot.UpdatedAt),
	}
}

func plotModelToDomain(plot PlotModel) domain.PlotRecord {
	return domain.PlotRecord{
		PlotID:    plot.PlotID,
		OrchardID: plot.OrchardID,
		PlotName:  plot.PlotName,
		Status:    domain.ResourceStatus(plot.Status),
		CreatedAt: plot.CreatedAt.UTC(),
		UpdatedAt: plot.UpdatedAt.UTC(),
	}
}
