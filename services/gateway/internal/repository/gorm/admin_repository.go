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
			if email == "" {
				return repository.ErrNotFound
			}
			if err := tx.Where("email = ? AND oidc_subject IS NULL", email).First(&user).Error; err != nil {
				return mapGormErr(err)
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
		user.LastLoginAt = timePtr(normalizeTime(now))
		if displayName != "" {
			user.DisplayName = displayName
		}
		user.UpdatedAt = normalizeTime(now)
		if err := tx.Model(&UserModel{}).
			Where("id = ?", user.ID).
			Updates(map[string]any{
				"display_name":  user.DisplayName,
				"last_login_at": user.LastLoginAt,
				"updated_at":    user.UpdatedAt,
			}).Error; err != nil {
			return mapGormErr(err)
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
	res := r.db.WithContext(ctx).Model(&UserModel{}).Where("id = ?", model.ID).Updates(map[string]any{
		"email":        model.Email,
		"display_name": model.DisplayName,
		"role":         model.Role,
		"status":       model.Status,
		"updated_at":   model.UpdatedAt,
	})
	if res.Error != nil {
		return domain.UserRecord{}, mapGormErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.UserRecord{}, repository.ErrNotFound
	}
	return r.GetPrincipalByID(ctx, model.ID)
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
