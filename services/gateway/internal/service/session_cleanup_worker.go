package service

import (
	"context"
	"log/slog"
	"time"
)

const defaultSessionCleanupInterval = 30 * time.Minute

// SessionCleanupService periodically deletes expired web sessions and OIDC
// auth states to prevent unbounded table growth. Without this, records
// created by users who never return would accumulate indefinitely because
// the lazy deletion in GetWebSession and ConsumeWebAuthState only fires on
// access.
type SessionCleanupService struct {
	repo     WebAuthRepository
	logger   *slog.Logger
	interval time.Duration
	nowFn    func() time.Time
}

func NewSessionCleanupService(repo WebAuthRepository, logger *slog.Logger, intervalS int) *SessionCleanupService {
	interval := time.Duration(intervalS) * time.Second
	if interval <= 0 {
		interval = defaultSessionCleanupInterval
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &SessionCleanupService{
		repo:     repo,
		logger:   logger,
		interval: interval,
		nowFn:    func() time.Time { return time.Now().UTC() },
	}
}

func (s *SessionCleanupService) StartWorker(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("session cleanup worker stopped")
			return
		case <-ticker.C:
			s.runOnce(ctx)
		}
	}
}

func (s *SessionCleanupService) runOnce(ctx context.Context) {
	now := s.nowFn()
	sessions, err := s.repo.DeleteExpiredSessions(ctx, now)
	if err != nil {
		s.logger.Error("session cleanup: failed to delete expired sessions", "error", err)
	}
	states, err := s.repo.DeleteExpiredAuthStates(ctx, now)
	if err != nil {
		s.logger.Error("session cleanup: failed to delete expired auth states", "error", err)
	}
	if sessions > 0 || states > 0 {
		s.logger.Info("session cleanup completed",
			"expired_sessions", sessions,
			"expired_auth_states", states,
		)
	}
}
