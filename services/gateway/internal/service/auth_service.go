package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/repository"
)

type AuthUserRepository interface {
	ResolvePrincipal(ctx context.Context, identity domain.IdentityClaims, mode domain.AuthMode, now time.Time) (domain.Principal, error)
	GetUserByOIDCSubject(ctx context.Context, subject string) (domain.UserRecord, error)
}

type UserAdminRepository interface {
	GetPrincipalByID(ctx context.Context, userID string) (domain.UserRecord, error)
	ListUsers(ctx context.Context) ([]domain.UserRecord, error)
	CreateUser(ctx context.Context, user domain.UserRecord) (domain.UserRecord, error)
	UpdateUser(ctx context.Context, expectedUpdatedAt time.Time, user domain.UserRecord) (domain.UserRecord, error)
}

type AuthService struct {
	repo  AuthUserRepository
	nowFn func() time.Time
}

func NewAuthService(repo AuthUserRepository) *AuthService {
	return &AuthService{
		repo:  repo,
		nowFn: func() time.Time { return time.Now().UTC() },
	}
}

func (s *AuthService) ResolvePrincipal(ctx context.Context, identity domain.IdentityClaims, mode domain.AuthMode) (domain.Principal, error) {
	if s.repo == nil {
		return domain.Principal{}, ErrServiceUnavailable
	}
	principal, err := s.repo.ResolvePrincipal(ctx, identity, mode, s.nowFn())
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return domain.Principal{}, ErrNotFound
		case errors.Is(err, repository.ErrInvalidState):
			return domain.Principal{}, ErrInvalidRequest
		default:
			return domain.Principal{}, ErrServiceUnavailable
		}
	}
	return principal, nil
}

func (s *AuthService) GetUserByOIDCSubject(ctx context.Context, subject string) (domain.UserRecord, error) {
	if s.repo == nil {
		return domain.UserRecord{}, ErrServiceUnavailable
	}
	user, err := s.repo.GetUserByOIDCSubject(ctx, subject)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return domain.UserRecord{}, ErrNotFound
		default:
			return domain.UserRecord{}, ErrServiceUnavailable
		}
	}
	return user, nil
}

type UserAdminService struct {
	repo  UserAdminRepository
	nowFn func() time.Time
}

type UserCreateInput struct {
	Email       string
	DisplayName string
	Role        domain.UserRole
	Status      domain.UserStatus
}

type UserUpdateInput struct {
	ID          string
	Email       string
	DisplayName string
	Role        domain.UserRole
	Status      domain.UserStatus
	EmailPresent       bool
	DisplayNamePresent bool
	RolePresent        bool
	StatusPresent      bool
}

func NewUserAdminService(repo UserAdminRepository) *UserAdminService {
	return &UserAdminService{
		repo:  repo,
		nowFn: func() time.Time { return time.Now().UTC() },
	}
}

func (s *UserAdminService) ListUsers(ctx context.Context) ([]domain.UserRecord, error) {
	users, err := s.repo.ListUsers(ctx)
	if err != nil {
		return nil, ErrServiceUnavailable
	}
	return users, nil
}

func (s *UserAdminService) CreateUser(ctx context.Context, input UserCreateInput) (domain.UserRecord, error) {
	user, err := normalizeCreateUser(input, s.nowFn())
	if err != nil {
		return domain.UserRecord{}, err
	}
	created, err := s.repo.CreateUser(ctx, user)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrConflict):
			return domain.UserRecord{}, ErrConflict
		default:
			return domain.UserRecord{}, ErrServiceUnavailable
		}
	}
	return created, nil
}

func (s *UserAdminService) UpdateUser(ctx context.Context, input UserUpdateInput) (domain.UserRecord, error) {
	current, err := s.repo.GetPrincipalByID(ctx, strings.TrimSpace(input.ID))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return domain.UserRecord{}, ErrNotFound
		}
		return domain.UserRecord{}, ErrServiceUnavailable
	}
	if !input.EmailPresent && !input.DisplayNamePresent && !input.RolePresent && !input.StatusPresent {
		return current, nil
	}
	expectedUpdatedAt := current.UpdatedAt
	if input.EmailPresent {
		current.Email = strings.ToLower(strings.TrimSpace(input.Email))
	}
	if input.DisplayNamePresent {
		current.DisplayName = strings.TrimSpace(input.DisplayName)
	}
	if input.RolePresent {
		current.Role = input.Role
	}
	if input.StatusPresent {
		current.Status = input.Status
	}
	current.UpdatedAt = s.nowFn()
	if err := validateUserRecord(current); err != nil {
		return domain.UserRecord{}, err
	}
	updated, err := s.repo.UpdateUser(ctx, expectedUpdatedAt, current)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrConflict):
			return domain.UserRecord{}, ErrConflict
		case errors.Is(err, repository.ErrNotFound):
			return domain.UserRecord{}, ErrNotFound
		case errors.Is(err, repository.ErrInvalidState):
			return domain.UserRecord{}, ErrConflict
		default:
			return domain.UserRecord{}, ErrServiceUnavailable
		}
	}
	return updated, nil
}

func normalizeCreateUser(input UserCreateInput, now time.Time) (domain.UserRecord, error) {
	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = strings.ToLower(strings.TrimSpace(input.Email))
	}
	user := domain.UserRecord{
		ID:          uuid.NewString(),
		Email:       strings.ToLower(strings.TrimSpace(input.Email)),
		DisplayName: displayName,
		Role:        input.Role,
		Status:      input.Status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := validateUserRecord(user); err != nil {
		return domain.UserRecord{}, err
	}
	return user, nil
}

func validateUserRecord(user domain.UserRecord) error {
	if user.Email == "" {
		return ErrInvalidRequest
	}
	switch user.Role {
	case domain.UserRoleAdmin, domain.UserRoleOperator:
	default:
		return ErrInvalidRequest
	}
	switch user.Status {
	case domain.UserStatusActive, domain.UserStatusDisabled:
	default:
		return ErrInvalidRequest
	}
	if user.DisplayName == "" {
		return ErrInvalidRequest
	}
	return nil
}
