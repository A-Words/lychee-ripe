package service

import (
	"context"
	"time"

	"github.com/lychee-ripe/gateway/internal/domain"
)

func (s *ReconcileService) StartAutoReconcileWorker(ctx context.Context) {
	if s.traceMode != domain.TraceModeBlockchain || s.anchorClient == nil {
		s.logger.Info("auto reconcile worker disabled because blockchain mode is inactive")
		return
	}

	interval := s.workerInterval
	if interval <= 0 {
		interval = autoReconcileInterval
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("auto reconcile worker stopped")
			return
		case <-ticker.C:
			if err := s.RunAutoReconcileOnce(ctx); err != nil {
				s.logger.Error("auto reconcile iteration failed", "error", err)
			}
		}
	}
}
