package gamemanagement

import (
	"context"
	"time"

	"github.com/bengobox/game-stats-api/internal/domain/game"
	"github.com/bengobox/game-stats-api/internal/infrastructure/cache"
	"github.com/bengobox/game-stats-api/internal/pkg/logger"
)

// ScoreSyncWorker periodically syncs stale game scores from individual Scoring
// entries back into the denormalised home_team_score / away_team_score columns.
type ScoreSyncWorker struct {
	gameRepo game.Repository
	cache    *cache.RedisClient
	interval time.Duration
	done     chan struct{}
}

// NewScoreSyncWorker creates a new score sync worker.
// interval controls how often the sync runs (e.g. 5*time.Minute).
func NewScoreSyncWorker(gameRepo game.Repository, cacheClient *cache.RedisClient, interval time.Duration) *ScoreSyncWorker {
	return &ScoreSyncWorker{
		gameRepo: gameRepo,
		cache:    cacheClient,
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
		games, err := w.gameRepo.ListByStatus(ctx, status)
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

			updated, err := w.gameRepo.SyncGameScores(ctx, g.ID)
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
				if w.cache != nil {
					cacheKey := cache.CacheKey("game", g.ID.String())
					_ = w.cache.Delete(ctx, cacheKey)
				}
			}
		}
	}

	if synced > 0 {
		logger.Info("ScoreSyncWorker: synced stale game scores",
			logger.Int("count", synced))
	}
}
