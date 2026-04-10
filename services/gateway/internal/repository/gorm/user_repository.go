package gorm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var _ repository.UserRepository = (*Repository)(nil)

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
	resolveOnce := func() error {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var user UserModel
			err := tx.Where("oidc_subject = ?", subject).First(&user).Error
			switch {
			case err == nil:
			case isNotFound(err):
				// First-time binding is intentionally gated on the email claim carried by the
				// presented access token so unknown subjects cannot create or bind local users.
				if email == "" {
					return repository.ErrNotFound
				}
				bound, err := bindPrincipalByEmail(tx, email, subject, displayName, now)
				if err != nil {
					return err
				}
				user = bound
			default:
				return mapGormErr(err)
			}

			if user.Status != string(domain.UserStatusActive) {
				return repository.ErrInvalidState
			}
			principal = userModelToPrincipal(user, mode)
			return nil
		})
	}
	var err error
	switch r.db.Dialector.Name() {
	case "postgres":
		err = resolveOnce()
	default:
		err = retrySQLiteBusy(ctx, resolveOnce)
	}
	if err != nil {
		return domain.Principal{}, err
	}
	return principal, nil
}

func bindPrincipalByEmail(
	tx *gorm.DB,
	email string,
	subject string,
	displayName string,
	now time.Time,
) (UserModel, error) {
	updates := map[string]any{
		"oidc_subject":  subject,
		"updated_at":    normalizeTime(now),
		"last_login_at": timePtr(normalizeTime(now)),
	}
	if displayName != "" {
		updates["display_name"] = displayName
	}

	res := tx.Model(&UserModel{}).
		Where("email = ? AND oidc_subject IS NULL AND status = ?", email, string(domain.UserStatusActive)).
		Updates(updates)
	if res.Error != nil {
		return UserModel{}, mapGormErr(res.Error)
	}
	if res.RowsAffected > 0 {
		var user UserModel
		if err := tx.Where("oidc_subject = ?", subject).First(&user).Error; err != nil {
			return UserModel{}, mapGormErr(err)
		}
		return user, nil
	}

	var existing UserModel
	if err := tx.Where("email = ?", email).First(&existing).Error; err != nil {
		return UserModel{}, mapGormErr(err)
	}
	if existing.Status != string(domain.UserStatusActive) {
		return UserModel{}, repository.ErrInvalidState
	}
	if existing.OIDCSubject != nil && strings.TrimSpace(*existing.OIDCSubject) == subject {
		return existing, nil
	}
	return UserModel{}, repository.ErrNotFound
}

func (r *Repository) GetPrincipalByID(ctx context.Context, userID string) (domain.UserRecord, error) {
	var user UserModel
	if err := r.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(userID)).First(&user).Error; err != nil {
		return domain.UserRecord{}, mapGormErr(err)
	}
	return userModelToDomain(user), nil
}

func (r *Repository) GetUserByOIDCSubject(ctx context.Context, subject string) (domain.UserRecord, error) {
	var user UserModel
	if err := r.db.WithContext(ctx).Where("oidc_subject = ?", strings.TrimSpace(subject)).First(&user).Error; err != nil {
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

func timePtr(t time.Time) *time.Time {
	return &t
}

func derefOrValue(value *string, fallback string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return fallback
	}
	return strings.TrimSpace(*value)
}
