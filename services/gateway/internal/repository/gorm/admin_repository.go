package gorm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	_ repository.UserRepository    = (*Repository)(nil)
	_ repository.OrchardRepository = (*Repository)(nil)
	_ repository.PlotRepository    = (*Repository)(nil)
	_ repository.SeedRepository    = (*Repository)(nil)
)

func (r *Repository) ResolvePrincipal(
	ctx context.Context,
	identity domain.IdentityClaims,
	mode domain.AuthMode,
	now time.Time,
) (domain.Principal, error) {
	subject := strings.TrimSpace(identity.Subject)
	email := strings.ToLower(strings.TrimSpace(identity.Email))
	displayName := strings.TrimSpace(identity.DisplayName)
	if subject == "" {
		return domain.Principal{}, repository.ErrNotFound
	}

	var principal domain.Principal
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user UserModel
		err := tx.Where("oidc_subject = ?", subject).First(&user).Error
		switch {
		case err == nil:
		case errors.Is(err, gorm.ErrRecordNotFound):
			// First-time binding is intentionally gated on the email claim carried by the
			// presented access token so unknown subjects cannot create or bind local users.
			if email == "" {
				return repository.ErrNotFound
			}
			if err := tx.Where("email = ? AND oidc_subject IS NULL", email).First(&user).Error; err != nil {
				return mapGormErr(err)
			}
			if user.Status != string(domain.UserStatusActive) {
				return repository.ErrInvalidState
			}
			user.OIDCSubject = &subject
			if displayName != "" {
				user.DisplayName = displayName
			}
			user.UpdatedAt = normalizeTime(now)
			user.LastLoginAt = timePtr(normalizeTime(now))
			if err := tx.Save(&user).Error; err != nil {
				return mapGormErr(err)
			}
		default:
			return mapGormErr(err)
		}

		if user.Status != string(domain.UserStatusActive) {
			return repository.ErrInvalidState
		}
		principal = userModelToPrincipal(user, mode)
		return nil
	})
	if err != nil {
		return domain.Principal{}, err
	}
	return principal, nil
}

func (r *Repository) GetPrincipalByID(ctx context.Context, userID string) (domain.UserRecord, error) {
	var user UserModel
	if err := r.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(userID)).First(&user).Error; err != nil {
		return domain.UserRecord{}, mapGormErr(err)
	}
	return userModelToDomain(user), nil
}

func (r *Repository) CountUsers(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&UserModel{}).Count(&count).Error; err != nil {
		return 0, mapGormErr(err)
	}
	return count, nil
}

func (r *Repository) CountActiveAdmins(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&UserModel{}).
		Where("role = ? AND status = ?", string(domain.UserRoleAdmin), string(domain.UserStatusActive)).
		Count(&count).Error; err != nil {
		return 0, mapGormErr(err)
	}
	return count, nil
}

func (r *Repository) ListUsers(ctx context.Context) ([]domain.UserRecord, error) {
	var models []UserModel
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, mapGormErr(err)
	}
	out := make([]domain.UserRecord, 0, len(models))
	for _, item := range models {
		out = append(out, userModelToDomain(item))
	}
	return out, nil
}

func (r *Repository) CreateUser(ctx context.Context, user domain.UserRecord) (domain.UserRecord, error) {
	model := userModelFromDomain(user)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.UserRecord{}, mapGormErr(err)
	}
	return userModelToDomain(model), nil
}

func (r *Repository) UpdateUser(ctx context.Context, user domain.UserRecord) (domain.UserRecord, error) {
	model := userModelFromDomain(user)

	var err error
	switch r.db.Dialector.Name() {
	case "postgres":
		err = r.updateUserPostgres(ctx, model)
	default:
		err = r.updateUserAtomically(ctx, model)
	}
	if err != nil {
		return domain.UserRecord{}, err
	}
	return r.GetPrincipalByID(ctx, model.ID)
}

func (r *Repository) updateUserPostgres(ctx context.Context, model UserModel) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current UserModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", model.ID).
			First(&current).Error; err != nil {
			return mapGormErr(err)
		}

		if removesActiveAdmin(current, model) {
			var activeAdminIDs []string
			if err := tx.Model(&UserModel{}).
				Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("role = ? AND status = ?", string(domain.UserRoleAdmin), string(domain.UserStatusActive)).
				Pluck("id", &activeAdminIDs).Error; err != nil {
				return mapGormErr(err)
			}
			if len(activeAdminIDs) <= 1 {
				return fmt.Errorf("%w: system must retain at least one active admin account", repository.ErrInvalidState)
			}
		}

		return updateUserRecord(tx, model)
	})
}

func (r *Repository) updateUserAtomically(ctx context.Context, model UserModel) error {
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		lastErr = r.updateUserAtomicallyOnce(ctx, model)
		if lastErr == nil {
			return nil
		}
		if !isSQLiteBusy(lastErr) {
			return lastErr
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("%w: %v", repository.ErrDBUnavailable, ctx.Err())
		case <-time.After(time.Duration(attempt+1) * 25 * time.Millisecond):
		}
	}
	return mapGormErr(lastErr)
}

func (r *Repository) updateUserAtomicallyOnce(ctx context.Context, model UserModel) error {
	res := r.db.WithContext(ctx).
		Model(&UserModel{}).
		Where("id = ?", model.ID).
		Where(
			`NOT (role = ? AND status = ?) OR (? = ? AND ? = ?) OR EXISTS (
				SELECT 1 FROM users AS other
				WHERE other.id <> users.id AND other.role = ? AND other.status = ?
			)`,
			string(domain.UserRoleAdmin),
			string(domain.UserStatusActive),
			model.Role,
			string(domain.UserRoleAdmin),
			model.Status,
			string(domain.UserStatusActive),
			string(domain.UserRoleAdmin),
			string(domain.UserStatusActive),
		).
		Updates(userUpdateAssignments(model))
	if res.Error != nil {
		return mapGormErr(res.Error)
	}
	if res.RowsAffected > 0 {
		return nil
	}

	var exists int64
	if err := r.db.WithContext(ctx).
		Model(&UserModel{}).
		Where("id = ?", model.ID).
		Count(&exists).Error; err != nil {
		return mapGormErr(err)
	}
	if exists == 0 {
		return repository.ErrNotFound
	}
	return fmt.Errorf("%w: system must retain at least one active admin account", repository.ErrInvalidState)
}

func updateUserRecord(tx *gorm.DB, model UserModel) error {
	res := tx.Model(&UserModel{}).
		Where("id = ?", model.ID).
		Updates(userUpdateAssignments(model))
	if res.Error != nil {
		return mapGormErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func userUpdateAssignments(model UserModel) map[string]any {
	return map[string]any{
		"email":        model.Email,
		"display_name": model.DisplayName,
		"role":         model.Role,
		"status":       model.Status,
		"updated_at":   model.UpdatedAt,
	}
}

func removesActiveAdmin(current UserModel, next UserModel) bool {
	return current.Role == string(domain.UserRoleAdmin) &&
		current.Status == string(domain.UserStatusActive) &&
		(next.Role != string(domain.UserRoleAdmin) || next.Status != string(domain.UserStatusActive))
}

func isSQLiteBusy(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") || strings.Contains(msg, "sqlite_busy")
}

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

func (r *Repository) UpdateOrchard(ctx context.Context, orchard domain.OrchardRecord) (domain.OrchardRecord, error) {
	model := orchardModelFromDomain(orchard)
	res := r.db.WithContext(ctx).Model(&OrchardModel{}).Where("orchard_id = ?", model.OrchardID).Updates(map[string]any{
		"orchard_name": model.OrchardName,
		"status":       model.Status,
		"updated_at":   model.UpdatedAt,
	})
	if res.Error != nil {
		return domain.OrchardRecord{}, mapGormErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.OrchardRecord{}, repository.ErrNotFound
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

func (r *Repository) UpdatePlot(ctx context.Context, plot domain.PlotRecord) (domain.PlotRecord, error) {
	model := plotModelFromDomain(plot)
	res := r.db.WithContext(ctx).Model(&PlotModel{}).Where("plot_id = ?", model.PlotID).Updates(map[string]any{
		"orchard_id": model.OrchardID,
		"plot_name":  model.PlotName,
		"status":     model.Status,
		"updated_at": model.UpdatedAt,
	})
	if res.Error != nil {
		return domain.PlotRecord{}, mapGormErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.PlotRecord{}, repository.ErrNotFound
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

func (r *Repository) CreatePlotIfNotExists(ctx context.Context, plot domain.PlotRecord) error {
	model := plotModelFromDomain(plot)
	err := r.db.WithContext(ctx).FirstOrCreate(&model, PlotModel{PlotID: model.PlotID}).Error
	return mapGormErr(err)
}

func userModelFromDomain(user domain.UserRecord) UserModel {
	return UserModel{
		ID:          user.ID,
		Email:       strings.ToLower(strings.TrimSpace(user.Email)),
		DisplayName: strings.TrimSpace(user.DisplayName),
		OIDCSubject: user.OIDCSubject,
		Role:        string(user.Role),
		Status:      string(user.Status),
		LastLoginAt: user.LastLoginAt,
		CreatedAt:   normalizeTime(user.CreatedAt),
		UpdatedAt:   normalizeTime(user.UpdatedAt),
	}
}

func userModelToDomain(user UserModel) domain.UserRecord {
	return domain.UserRecord{
		ID:          user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		OIDCSubject: user.OIDCSubject,
		Role:        domain.UserRole(user.Role),
		Status:      domain.UserStatus(user.Status),
		LastLoginAt: user.LastLoginAt,
		CreatedAt:   user.CreatedAt.UTC(),
		UpdatedAt:   user.UpdatedAt.UTC(),
	}
}

func userModelToPrincipal(user UserModel, mode domain.AuthMode) domain.Principal {
	return domain.Principal{
		Subject:     derefOrValue(user.OIDCSubject, user.ID),
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Role:        domain.UserRole(user.Role),
		Status:      domain.UserStatus(user.Status),
		AuthMode:    mode,
	}
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

func timePtr(t time.Time) *time.Time {
	return &t
}

func derefOrValue(value *string, fallback string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return fallback
	}
	return strings.TrimSpace(*value)
}

func validateUserRecord(user domain.UserRecord) error {
	if strings.TrimSpace(user.ID) == "" {
		return fmt.Errorf("%w: user id is required", repository.ErrInvalidState)
	}
	return nil
}
