package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/repository"
)

type BootstrapAdminRepo interface {
	CountActiveAdmins(ctx context.Context) (int64, error)
	CreateUser(ctx context.Context, user domain.UserRecord) (domain.UserRecord, error)
	ListUsers(ctx context.Context) ([]domain.UserRecord, error)
	UpdateUser(ctx context.Context, user domain.UserRecord) (domain.UserRecord, error)
}

func EnsureBootstrapAdmin(
	ctx context.Context,
	mode domain.AuthMode,
	bootstrapAdminEmail string,
	repo BootstrapAdminRepo,
) error {
	if mode != domain.AuthModeOIDC || repo == nil {
		return nil
	}

	count, err := repo.CountActiveAdmins(ctx)
	if err != nil {
		return fmt.Errorf("%w: count active admins: %v", ErrServiceUnavailable, err)
	}
	if count > 0 {
		return nil
	}

	email := strings.ToLower(strings.TrimSpace(bootstrapAdminEmail))
	if email == "" {
		return fmt.Errorf("%w: auth.bootstrap_admin_email is required when auth.mode=oidc and no active admin exists", ErrInvalidRequest)
	}

	now := time.Now().UTC()
	user, err := normalizeCreateUser(UserCreateInput{
		Email:  email,
		Role:   domain.UserRoleAdmin,
		Status: domain.UserStatusActive,
	}, now)
	if err != nil {
		return err
	}

	if _, err := repo.CreateUser(ctx, user); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			if err := promoteBootstrapAdminByEmail(ctx, repo, email, now); err != nil {
				return err
			}
			return nil
		}
		return fmt.Errorf("%w: create bootstrap admin: %v", ErrServiceUnavailable, err)
	}
	return nil
}

func promoteBootstrapAdminByEmail(
	ctx context.Context,
	repo BootstrapAdminRepo,
	email string,
	now time.Time,
) error {
	users, err := repo.ListUsers(ctx)
	if err != nil {
		return fmt.Errorf("%w: list users for bootstrap recovery: %v", ErrServiceUnavailable, err)
	}

	for _, user := range users {
		if strings.ToLower(strings.TrimSpace(user.Email)) != email {
			continue
		}
		if strings.TrimSpace(user.DisplayName) == "" {
			user.DisplayName = email
		}
		user.Role = domain.UserRoleAdmin
		user.Status = domain.UserStatusActive
		user.UpdatedAt = now
		if _, err := repo.UpdateUser(ctx, user); err != nil {
			switch {
			case errors.Is(err, repository.ErrConflict):
				return fmt.Errorf("%w: bootstrap admin email conflicts with another user record", ErrInvalidRequest)
			case errors.Is(err, repository.ErrInvalidState):
				return fmt.Errorf("%w: bootstrap admin promotion violated user state constraints", ErrInvalidRequest)
			default:
				return fmt.Errorf("%w: promote bootstrap admin: %v", ErrServiceUnavailable, err)
			}
		}
		return nil
	}

	count, countErr := repo.CountActiveAdmins(ctx)
	if countErr != nil {
		return fmt.Errorf("%w: verify bootstrap admin after conflict: %v", ErrServiceUnavailable, countErr)
	}
	if count > 0 {
		return nil
	}
	return fmt.Errorf("%w: bootstrap admin email conflicts with an existing non-active-admin user", ErrInvalidRequest)
}
