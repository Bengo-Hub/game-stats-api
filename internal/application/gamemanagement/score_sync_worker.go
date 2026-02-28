package gamemanagement

import (
	"context"
	"time"

	"github.com/bengobox/game-stats-api/internal/infrastructure/cache"
	"github.com/bengobox/game-stats-api/internal/pkg/logger"
)

// ScoreSyncWorker periodically syncs stale game scores from individual Scoring
// entries back into the denormalised home_team_score / away_team_score columns.
type ScoreSyncWorker struct {
	service  *Service
	interval time.Duration
	done     chan struct{}
}

// NewScoreSyncWorker creates a new score sync worker.
// interval controls how often the sync runs (e.g. 5*time.Minute).
func NewScoreSyncWorker(service *Service, interval time.Duration) *ScoreSyncWorker {
	return &ScoreSyncWorker{
		service:  service,
		interval: interval,
		done:     make(chan struct{}),
	}
}

// Start runs the worker loop. Call in a goroutine. Cancel the ctx or call Stop to halt.
func (w *ScoreSyncWorker) Start(ctx context.Context) {
	logger.Info("ScoreSyncWorker: starting background score sync",
		logger.String("interval", w.interval.String()))

	// Run immediately on startup
	w.syncStaleScores(ctx)
	w.service.AutoEndExpiredGames(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("ScoreSyncWorker: stopping (context cancelled)")
			return
		case <-w.done:
			logger.Info("ScoreSyncWorker: stopping (done signal)")
			return
		case <-ticker.C:
			w.syncStaleScores(ctx)
			count, err := w.service.AutoEndExpiredGames(ctx)
			if err != nil {
				logger.Error("ScoreSyncWorker: failed to auto-end games", logger.Err(err))
			} else if count > 0 {
				logger.Info("ScoreSyncWorker: auto-ended expired games", logger.Int("count", count))
			}
		}
	}
}

// Stop signals the worker to stop.
func (w *ScoreSyncWorker) Stop() {
	close(w.done)
}

// syncStaleScores finds games with stale scores and recalculates them.
func (w *ScoreSyncWorker) syncStaleScores(ctx context.Context) {
	statuses := []string{"completed", "ended", "in_progress"}
	synced := 0

	for _, status := range statuses {
		games, err := w.service.gameRepo.ListByStatus(ctx, status)
		if err != nil {
			logger.Error("ScoreSyncWorker: failed to list games",
				logger.String("status", status),
				logger.Err(err))
			continue
		}

		for _, g := range games {
			// Only sync games where the denormalised totals are zero
			if g.HomeTeamScore != 0 || g.AwayTeamScore != 0 {
				continue
			}

			updated, err := w.service.gameRepo.SyncGameScores(ctx, g.ID)
			if err != nil {
				logger.Error("ScoreSyncWorker: failed to sync game scores",
					logger.String("gameID", g.ID.String()),
					logger.Err(err))
				continue
			}

			// Only count if we actually found scores
			if updated.HomeTeamScore > 0 || updated.AwayTeamScore > 0 {
				synced++

				// Invalidate cache so next read picks up new scores
				if w.service.cache != nil {
					cacheKey := cache.CacheKey("game", g.ID.String())
					_ = w.service.cache.Delete(ctx, cacheKey)
				}
			}
		}
	}

	if synced > 0 {
		logger.Info("ScoreSyncWorker: synced stale game scores",
			logger.Int("count", synced))
	}
}
