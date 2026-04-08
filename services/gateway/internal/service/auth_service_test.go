package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/repository"
)

func TestUserAdminServiceUpdateUserRejectsDemotingLastActiveAdmin(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 9, 8, 0, 0, 0, time.UTC)
	repo := &fakeUserAdminRepo{
		user: domain.UserRecord{
			ID:          "user-1",
			Email:       "admin@example.com",
			DisplayName: "Admin",
			Role:        domain.UserRoleAdmin,
			Status:      domain.UserStatusActive,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		updateErr: repository.ErrInvalidState,
	}
	svc := NewUserAdminService(repo)
	svc.nowFn = func() time.Time { return now.Add(time.Minute) }

	_, err := svc.UpdateUser(context.Background(), UserUpdateInput{
		ID:          "user-1",
		Email:       "admin@example.com",
		DisplayName: "Admin",
		Role:        domain.UserRoleOperator,
		Status:      domain.UserStatusActive,
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v, want ErrInvalidRequest", err)
	}
	if !repo.updateCalled {
		t.Fatal("expected repository update to be called")
	}
}

func TestUserAdminServiceUpdateUserRejectsDisablingLastActiveAdmin(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 9, 8, 0, 0, 0, time.UTC)
	repo := &fakeUserAdminRepo{
		user: domain.UserRecord{
			ID:          "user-1",
			Email:       "admin@example.com",
			DisplayName: "Admin",
			Role:        domain.UserRoleAdmin,
			Status:      domain.UserStatusActive,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		updateErr: repository.ErrInvalidState,
	}
	svc := NewUserAdminService(repo)
	svc.nowFn = func() time.Time { return now.Add(time.Minute) }

	_, err := svc.UpdateUser(context.Background(), UserUpdateInput{
		ID:          "user-1",
		Email:       "admin@example.com",
		DisplayName: "Admin",
		Role:        domain.UserRoleAdmin,
		Status:      domain.UserStatusDisabled,
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v, want ErrInvalidRequest", err)
	}
	if !repo.updateCalled {
		t.Fatal("expected repository update to be called")
	}
}

func TestUserAdminServiceUpdateUserAllowsChangingOneAdminWhenAnotherExists(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 9, 8, 0, 0, 0, time.UTC)
	repo := &fakeUserAdminRepo{
		user: domain.UserRecord{
			ID:          "user-1",
			Email:       "admin@example.com",
			DisplayName: "Admin",
			Role:        domain.UserRoleAdmin,
			Status:      domain.UserStatusActive,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
	svc := NewUserAdminService(repo)
	svc.nowFn = func() time.Time { return now.Add(time.Minute) }

	updated, err := svc.UpdateUser(context.Background(), UserUpdateInput{
		ID:          "user-1",
		Email:       "admin@example.com",
		DisplayName: "Admin",
		Role:        domain.UserRoleOperator,
		Status:      domain.UserStatusActive,
	})
	if err != nil {
		t.Fatalf("UpdateUser returned error: %v", err)
	}
	if !repo.updateCalled {
		t.Fatal("expected repository update to be called")
	}
	if updated.Role != domain.UserRoleOperator {
		t.Fatalf("updated role = %q, want operator", updated.Role)
	}
}

func TestUserAdminServiceUpdateUserAllowsUpdatingLastAdminProfile(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 9, 8, 0, 0, 0, time.UTC)
	repo := &fakeUserAdminRepo{
		user: domain.UserRecord{
			ID:          "user-1",
			Email:       "admin@example.com",
			DisplayName: "Admin",
			Role:        domain.UserRoleAdmin,
			Status:      domain.UserStatusActive,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
	svc := NewUserAdminService(repo)
	svc.nowFn = func() time.Time { return now.Add(time.Minute) }

	updated, err := svc.UpdateUser(context.Background(), UserUpdateInput{
		ID:          "user-1",
		Email:       "renamed-admin@example.com",
		DisplayName: "Renamed Admin",
		Role:        domain.UserRoleAdmin,
		Status:      domain.UserStatusActive,
	})
	if err != nil {
		t.Fatalf("UpdateUser returned error: %v", err)
	}
	if !repo.updateCalled {
		t.Fatal("expected repository update to be called")
	}
	if updated.Email != "renamed-admin@example.com" {
		t.Fatalf("updated email = %q, want renamed-admin@example.com", updated.Email)
	}
	if updated.DisplayName != "Renamed Admin" {
		t.Fatalf("updated display_name = %q, want Renamed Admin", updated.DisplayName)
	}
}

func TestUserAdminServiceUpdateUserReturnsServiceUnavailableOnRepositoryFailure(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 9, 8, 0, 0, 0, time.UTC)
	repo := &fakeUserAdminRepo{
		user: domain.UserRecord{
			ID:          "user-1",
			Email:       "admin@example.com",
			DisplayName: "Admin",
			Role:        domain.UserRoleAdmin,
			Status:      domain.UserStatusActive,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		updateErr: repository.ErrDBUnavailable,
	}
	svc := NewUserAdminService(repo)
	svc.nowFn = func() time.Time { return now.Add(time.Minute) }

	_, err := svc.UpdateUser(context.Background(), UserUpdateInput{
		ID:          "user-1",
		Email:       "admin@example.com",
		DisplayName: "Admin",
		Role:        domain.UserRoleOperator,
		Status:      domain.UserStatusActive,
	})
	if !errors.Is(err, ErrServiceUnavailable) {
		t.Fatalf("error = %v, want ErrServiceUnavailable", err)
	}
	if !repo.updateCalled {
		t.Fatal("expected repository update to be called")
	}
}

type fakeUserAdminRepo struct {
	user         domain.UserRecord
	getErr       error
	updateErr    error
	updateCalled bool
}

func (f *fakeUserAdminRepo) GetPrincipalByID(_ context.Context, userID string) (domain.UserRecord, error) {
	if f.getErr != nil {
		return domain.UserRecord{}, f.getErr
	}
	if userID != f.user.ID {
		return domain.UserRecord{}, repository.ErrNotFound
	}
	return f.user, nil
}

func (f *fakeUserAdminRepo) ListUsers(_ context.Context) ([]domain.UserRecord, error) {
	return []domain.UserRecord{f.user}, nil
}

func (f *fakeUserAdminRepo) CreateUser(_ context.Context, user domain.UserRecord) (domain.UserRecord, error) {
	return user, nil
}

func (f *fakeUserAdminRepo) UpdateUser(_ context.Context, user domain.UserRecord) (domain.UserRecord, error) {
	f.updateCalled = true
	if f.updateErr != nil {
		return domain.UserRecord{}, f.updateErr
	}
	f.user = user
	return user, nil
}
