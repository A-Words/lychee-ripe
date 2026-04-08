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
	CountUsers(ctx context.Context) (int64, error)
	CreateUser(ctx context.Context, user domain.UserRecord) (domain.UserRecord, error)
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

	count, err := repo.CountUsers(ctx)
	if err != nil {
		return fmt.Errorf("%w: count users: %v", ErrServiceUnavailable, err)
	}
	if count > 0 {
		return nil
	}

	email := strings.ToLower(strings.TrimSpace(bootstrapAdminEmail))
	if email == "" {
		return fmt.Errorf("%w: auth.bootstrap_admin_email is required when auth.mode=oidc and users table is empty", ErrInvalidRequest)
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
			return nil
		}
		return fmt.Errorf("%w: create bootstrap admin: %v", ErrServiceUnavailable, err)
	}
	return nil
}
