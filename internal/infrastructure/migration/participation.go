package migration

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/internal/pkg/logger"
)

// migrateEventParticipation links players and teams to events historically
func (m *Migrator) migrateEventParticipation(ctx context.Context, fixturesDir string) error {
	logger.Info("Starting event participation migration...")

	// We'll use the players already migrated to create participation records
	// Since each player in the legacy system belongs to a team, and each team
	// is linked to a division pool, which belongs to an event.

	players, err := m.client.Player.Query().
		WithTeam(func(q *ent.TeamQuery) {
			q.WithDivisionPool(func(dq *ent.DivisionPoolQuery) {
				dq.WithEvent()
			})
		}).
		All(ctx)
	if err != nil {
		return err
	}

	count := 0
	for _, p := range players {
		if p.Edges.Team == nil || p.Edges.Team.Edges.DivisionPool == nil || p.Edges.Team.Edges.DivisionPool.Edges.Event == nil {
			continue
		}

		team := p.Edges.Team
		event := team.Edges.DivisionPool.Edges.Event

		// Create participation (let it fail if unique constraint player/team/event hit)
		_, err = m.client.EventParticipation.Create().
			SetPlayer(p).
			SetTeam(team).
			SetEvent(event).
			SetRole("player").
			SetStatus("active").
			Save(ctx)
		if err != nil {
			// Expected if unique constraint is hit or already exists
			continue
		}
		count++
	}

	logger.Info("Event participation migration complete", logger.Int("records_created", count))
	return nil
}
