package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/repository"
)

func TestEnsureBootstrapAdminSkipsWhenAuthDisabled(t *testing.T) {
	t.Parallel()

	repo := &fakeBootstrapAdminRepo{countActiveAdmins: 0}
	if err := EnsureBootstrapAdmin(context.Background(), domain.AuthModeDisabled, "", repo); err != nil {
		t.Fatalf("EnsureBootstrapAdmin returned error: %v", err)
	}
	if repo.createCallCount != 0 {
		t.Fatalf("createCallCount = %d, want 0", repo.createCallCount)
	}
}

func TestEnsureBootstrapAdminFailsWithoutBootstrapEmail(t *testing.T) {
	t.Parallel()

	repo := &fakeBootstrapAdminRepo{countActiveAdmins: 0}
	err := EnsureBootstrapAdmin(context.Background(), domain.AuthModeOIDC, "", repo)
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v, want ErrInvalidRequest", err)
	}
}

func TestEnsureBootstrapAdminCreatesAdminForFreshOIDCStore(t *testing.T) {
	t.Parallel()

	repo := &fakeBootstrapAdminRepo{countActiveAdmins: 0}
	if err := EnsureBootstrapAdmin(context.Background(), domain.AuthModeOIDC, "Admin@Example.com", repo); err != nil {
		t.Fatalf("EnsureBootstrapAdmin returned error: %v", err)
	}
	if repo.createCallCount != 1 {
		t.Fatalf("createCallCount = %d, want 1", repo.createCallCount)
	}
	if repo.created.Email != "admin@example.com" {
		t.Fatalf("email = %q, want admin@example.com", repo.created.Email)
	}
	if repo.created.DisplayName != "admin@example.com" {
		t.Fatalf("display_name = %q, want admin@example.com", repo.created.DisplayName)
	}
	if repo.created.Role != domain.UserRoleAdmin {
		t.Fatalf("role = %q, want admin", repo.created.Role)
	}
	if repo.created.Status != domain.UserStatusActive {
		t.Fatalf("status = %q, want active", repo.created.Status)
	}
	if repo.created.OIDCSubject != nil {
		t.Fatalf("oidc_subject = %v, want nil", repo.created.OIDCSubject)
	}
	if repo.created.ID == "" {
		t.Fatal("expected generated user id")
	}
	if repo.created.CreatedAt.IsZero() || repo.created.UpdatedAt.IsZero() {
		t.Fatal("expected created_at and updated_at to be set")
	}
}

func TestEnsureBootstrapAdminSkipsWhenActiveAdminAlreadyExists(t *testing.T) {
	t.Parallel()

	repo := &fakeBootstrapAdminRepo{countActiveAdmins: 1}
	if err := EnsureBootstrapAdmin(context.Background(), domain.AuthModeOIDC, "admin@example.com", repo); err != nil {
		t.Fatalf("EnsureBootstrapAdmin returned error: %v", err)
	}
	if repo.createCallCount != 0 {
		t.Fatalf("createCallCount = %d, want 0", repo.createCallCount)
	}
}

func TestEnsureBootstrapAdminCreatesAdminWhenNoActiveAdminExists(t *testing.T) {
	t.Parallel()

	repo := &fakeBootstrapAdminRepo{countActiveAdmins: 0}
	if err := EnsureBootstrapAdmin(context.Background(), domain.AuthModeOIDC, "admin@example.com", repo); err != nil {
		t.Fatalf("EnsureBootstrapAdmin returned error: %v", err)
	}
	if repo.createCallCount != 1 {
		t.Fatalf("createCallCount = %d, want 1", repo.createCallCount)
	}
}

func TestEnsureBootstrapAdminTreatsCreateConflictAsInitialized(t *testing.T) {
	t.Parallel()

	repo := &fakeBootstrapAdminRepo{
		countActiveAdminsSequence: []int64{0, 1},
		createUserErr:            repository.ErrConflict,
	}
	if err := EnsureBootstrapAdmin(context.Background(), domain.AuthModeOIDC, "admin@example.com", repo); err != nil {
		t.Fatalf("EnsureBootstrapAdmin returned error: %v", err)
	}
	if repo.createCallCount != 1 {
		t.Fatalf("createCallCount = %d, want 1", repo.createCallCount)
	}
}

func TestEnsureBootstrapAdminPromotesExistingSameEmailUserWhenNoActiveAdminExists(t *testing.T) {
	t.Parallel()

	repo := &fakeBootstrapAdminRepo{
		countActiveAdmins: 0,
		createUserErr:     repository.ErrConflict,
		users: []domain.UserRecord{
			{
				ID:          "user-1",
				Email:       "admin@example.com",
				DisplayName: "Bootstrap Candidate",
				Role:        domain.UserRoleOperator,
				Status:      domain.UserStatusDisabled,
				CreatedAt:   time.Now().UTC(),
				UpdatedAt:   time.Now().UTC(),
			},
		},
	}
	if err := EnsureBootstrapAdmin(context.Background(), domain.AuthModeOIDC, "admin@example.com", repo); err != nil {
		t.Fatalf("EnsureBootstrapAdmin returned error: %v", err)
	}
	if repo.updateCallCount != 1 {
		t.Fatalf("updateCallCount = %d, want 1", repo.updateCallCount)
	}
	if repo.updated.Role != domain.UserRoleAdmin {
		t.Fatalf("updated role = %q, want admin", repo.updated.Role)
	}
	if repo.updated.Status != domain.UserStatusActive {
		t.Fatalf("updated status = %q, want active", repo.updated.Status)
	}
}

func TestEnsureBootstrapAdminRejectsConflictWhenStillNoActiveAdmin(t *testing.T) {
	t.Parallel()

	repo := &fakeBootstrapAdminRepo{
		countActiveAdminsSequence: []int64{0, 0},
		createUserErr:            repository.ErrConflict,
	}
	err := EnsureBootstrapAdmin(context.Background(), domain.AuthModeOIDC, "admin@example.com", repo)
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v, want ErrInvalidRequest", err)
	}
}

type fakeBootstrapAdminRepo struct {
	countActiveAdmins         int64
	countActiveAdminsSequence []int64
	countActiveAdminsErr      error
	createUserErr             error
	createCallCount           int
	updateCallCount           int
	created                   domain.UserRecord
	updated                   domain.UserRecord
	users                     []domain.UserRecord
}

func (f *fakeBootstrapAdminRepo) CountActiveAdmins(_ context.Context) (int64, error) {
	if f.countActiveAdminsErr != nil {
		return 0, f.countActiveAdminsErr
	}
	if len(f.countActiveAdminsSequence) > 0 {
		value := f.countActiveAdminsSequence[0]
		f.countActiveAdminsSequence = f.countActiveAdminsSequence[1:]
		return value, nil
	}
	return f.countActiveAdmins, nil
}

func (f *fakeBootstrapAdminRepo) CreateUser(_ context.Context, user domain.UserRecord) (domain.UserRecord, error) {
	f.createCallCount++
	if f.createUserErr != nil {
		return domain.UserRecord{}, f.createUserErr
	}
	f.created = user
	if f.created.CreatedAt.IsZero() {
		now := time.Now().UTC()
		f.created.CreatedAt = now
		f.created.UpdatedAt = now
	}
	return f.created, nil
}

func (f *fakeBootstrapAdminRepo) ListUsers(_ context.Context) ([]domain.UserRecord, error) {
	return append([]domain.UserRecord(nil), f.users...), nil
}

func (f *fakeBootstrapAdminRepo) UpdateUser(_ context.Context, user domain.UserRecord) (domain.UserRecord, error) {
	f.updateCallCount++
	f.updated = user
	for idx := range f.users {
		if f.users[idx].ID == user.ID {
			f.users[idx] = user
			break
		}
	}
	return user, nil
}
