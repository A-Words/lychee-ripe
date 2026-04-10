package gorm

import (
	"context"
	"errors"
	"time"

	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var _ repository.WebSessionRepository = (*Repository)(nil)

func (r *Repository) CreateWebAuthState(ctx context.Context, state domain.WebAuthStateRecord) (domain.WebAuthStateRecord, error) {
	model := webAuthStateModelFromDomain(state)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.WebAuthStateRecord{}, mapGormErr(err)
	}
	return webAuthStateModelToDomain(model), nil
}

func (r *Repository) ConsumeWebAuthState(ctx context.Context, state string, browserBindingHash string, now time.Time) (domain.WebAuthStateRecord, error) {
	var record domain.WebAuthStateRecord
	expired := false
	resolveOnce := func() error {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var model WebAuthStateModel
			query := tx.Where("state = ? AND browser_binding_hash = ?", state, browserBindingHash)
			if tx.Dialector.Name() == "postgres" {
				query = query.Clauses(clause.Locking{Strength: "UPDATE"})
			}
			if err := query.First(&model).Error; err != nil {
				return mapGormErr(err)
			}
			if model.ExpiresAt.Before(normalizeTime(now)) {
				if err := tx.Delete(&model).Error; err != nil {
					return mapGormErr(err)
				}
				expired = true
				return nil
			}
			if err := tx.Delete(&model).Error; err != nil {
				return mapGormErr(err)
			}
			record = webAuthStateModelToDomain(model)
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
		return domain.WebAuthStateRecord{}, err
	}
	if expired {
		return domain.WebAuthStateRecord{}, repository.ErrNotFound
	}
	return record, nil
}

func (r *Repository) CreateWebSession(ctx context.Context, session domain.WebSessionRecord) (domain.WebSessionRecord, error) {
	model := webSessionModelFromDomain(session)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.WebSessionRecord{}, mapGormErr(err)
	}
	return webSessionModelToDomain(model), nil
}

func (r *Repository) GetWebSession(ctx context.Context, sessionIDHash string, now time.Time) (domain.WebSessionRecord, error) {
	var model WebSessionModel
	if err := r.db.WithContext(ctx).Where("session_id_hash = ?", sessionIDHash).First(&model).Error; err != nil {
		return domain.WebSessionRecord{}, mapGormErr(err)
	}
	if model.ExpiresAt.Before(normalizeTime(now)) {
		if err := r.db.WithContext(ctx).Delete(&model).Error; err != nil && !errors.Is(mapGormErr(err), repository.ErrNotFound) {
			return domain.WebSessionRecord{}, mapGormErr(err)
		}
		return domain.WebSessionRecord{}, repository.ErrNotFound
	}
	return webSessionModelToDomain(model), nil
}

func (r *Repository) DeleteWebSession(ctx context.Context, sessionIDHash string) error {
	res := r.db.WithContext(ctx).Where("session_id_hash = ?", sessionIDHash).Delete(&WebSessionModel{})
	if res.Error != nil {
		return mapGormErr(res.Error)
	}
	return nil
}

func (r *Repository) DeleteExpiredSessions(ctx context.Context, now time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Where("expires_at < ?", normalizeTime(now)).Delete(&WebSessionModel{})
	if res.Error != nil {
		return 0, mapGormErr(res.Error)
	}
	return res.RowsAffected, nil
}

func (r *Repository) DeleteExpiredAuthStates(ctx context.Context, now time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Where("expires_at < ?", normalizeTime(now)).Delete(&WebAuthStateModel{})
	if res.Error != nil {
		return 0, mapGormErr(res.Error)
	}
	return res.RowsAffected, nil
}

func webSessionModelFromDomain(session domain.WebSessionRecord) WebSessionModel {
	return WebSessionModel{
		SessionIDHash: session.SessionIDHash,
		UserID:        session.UserID,
		IDToken:       session.IDToken,
		ExpiresAt:     normalizeTime(session.ExpiresAt),
		CreatedAt:     normalizeTime(session.CreatedAt),
		UpdatedAt:     normalizeTime(session.UpdatedAt),
	}
}

func webSessionModelToDomain(session WebSessionModel) domain.WebSessionRecord {
	return domain.WebSessionRecord{
		SessionIDHash: session.SessionIDHash,
		UserID:        session.UserID,
		IDToken:       session.IDToken,
		ExpiresAt:     session.ExpiresAt.UTC(),
		CreatedAt:     session.CreatedAt.UTC(),
		UpdatedAt:     session.UpdatedAt.UTC(),
	}
}

func webAuthStateModelFromDomain(state domain.WebAuthStateRecord) WebAuthStateModel {
	return WebAuthStateModel{
		State:              state.State,
		BrowserBindingHash: state.BrowserBindingHash,
		CodeVerifier:       state.CodeVerifier,
		RedirectPath:       state.RedirectPath,
		ExpiresAt:          normalizeTime(state.ExpiresAt),
		CreatedAt:          normalizeTime(state.CreatedAt),
		UpdatedAt:          normalizeTime(state.UpdatedAt),
	}
}

func webAuthStateModelToDomain(state WebAuthStateModel) domain.WebAuthStateRecord {
	return domain.WebAuthStateRecord{
		State:              state.State,
		BrowserBindingHash: state.BrowserBindingHash,
		CodeVerifier:       state.CodeVerifier,
		RedirectPath:       state.RedirectPath,
		ExpiresAt:          state.ExpiresAt.UTC(),
		CreatedAt:          state.CreatedAt.UTC(),
		UpdatedAt:          state.UpdatedAt.UTC(),
	}
}
