package gamemanagement

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/game"
	"github.com/bengobox/game-stats-api/ent/player"
	"github.com/bengobox/game-stats-api/ent/scoring"
	"github.com/google/uuid"
)

// Scoring System
func (s *Service) RecordScore(ctx context.Context, gameID uuid.UUID, userID uuid.UUID, req RecordScoreRequest) (*GameDTO, error) {
	// Start transaction
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Get game with relations
	gm, err := tx.Game.Query().
		Where(game.ID(gameID)).
		WithHomeTeam().
		WithAwayTeam().
		WithScorekeeper().
		WithDivisionPool(func(dpq *ent.DivisionPoolQuery) {
			dpq.WithEvents()
		}).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrGameNotFound
		}
		return nil, err
	}

	// Verify authorization
	isAuthorized, err := s.permissionService.IsScorekeeperForGame(ctx, userID, gameID)
	if err != nil {
		return nil, err
	}
	if !isAuthorized {
		return nil, ErrUnauthorized
	}

	// Check status
	if gm.Status == "completed" {
		return nil, errors.New("cannot record scores for a completed game")
	}
	if gm.Status == "cancelled" {
		return nil, errors.New("cannot record scores for a cancelled game")
	}

	// Verify player exists
	playerEntity, err := tx.Player.Query().
		Where(player.ID(req.PlayerID)).
		WithTeams().
		Only(ctx)
	if err != nil {
		return nil, err
	}

	// Check for Score Approval requirement
	gameEndTime := gm.ScheduledTime.Add(time.Duration(gm.AllocatedTimeMinutes) * time.Minute)
	if gm.ActualEndTime != nil {
		gameEndTime = *gm.ActualEndTime
	}

	if time.Now().After(gameEndTime) && gm.Status != "ended" && gm.Edges.DivisionPool != nil && len(gm.Edges.DivisionPool.Edges.Events) > 0 {
		eventEntity, err := tx.Event.Get(ctx, gm.Edges.DivisionPool.Edges.Events[0].ID)
		if err == nil {
			isManager, _ := s.permissionService.CheckPermission(ctx, userID, eventEntity.ScoreEditApprovalRole, "event", eventEntity.ID)
			if !isManager {
				_, err := tx.ScoreEditRequest.Create().
					SetGameID(gameID).
					SetRequestedByID(userID).
					SetPreviousHomeScore(gm.HomeTeamScore).
					SetPreviousAwayScore(gm.AwayTeamScore).
					SetNewHomeScore(gm.HomeTeamScore). // Simplified as we don't know the full impact without recalculate
					SetReason(fmt.Sprintf("Post-game score adjustment: Player %s goals set to %d", playerEntity.Name, req.Goals)).
					SetStatus("pending").
					Save(ctx)
				if err != nil {
					return nil, err
				}
				return nil, errors.New("score edit requires approval after game time has elapsed")
			}
		}
	}

	// Check if scoring record exists
	existing, err := tx.Scoring.Query().
		Where(scoring.HasGameWith(game.ID(gameID))).
		Where(scoring.HasPlayerWith(player.ID(req.PlayerID))).
		Where(scoring.DeletedAtIsNil()).
		Only(ctx)

	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}

	if existing != nil {
		err = tx.Scoring.UpdateOneID(existing.ID).
			SetGoals(req.Goals).
			SetAssists(req.Assists).
			SetBlocks(req.Blocks).
			SetTurns(req.Turns).
			SetTeamID(req.TeamID).
			SetUpdatedAt(time.Now()).
			Exec(ctx)
	} else {
		err = tx.Scoring.Create().
			SetGoals(req.Goals).
			SetAssists(req.Assists).
			SetBlocks(req.Blocks).
			SetTurns(req.Turns).
			SetTeamID(req.TeamID).
			SetGameID(gameID).
			SetPlayerID(req.PlayerID).
			Exec(ctx)
	}
	if err != nil {
		return nil, err
	}

	// Recalculate totals directly in transaction
	scoreList, err := tx.Scoring.Query().
		Where(scoring.HasGameWith(game.ID(gameID))).
		Where(scoring.DeletedAtIsNil()).
		All(ctx)
	if err != nil {
		return nil, err
	}

	homeScore := 0
	awayScore := 0
	for _, sl := range scoreList {
		if sl.TeamID == gm.Edges.HomeTeam.ID {
			homeScore += sl.Goals
		} else if sl.TeamID == gm.Edges.AwayTeam.ID {
			awayScore += sl.Goals
		}
	}

	// Update game scores and version
	updatedGm, err := tx.Game.UpdateOneID(gameID).
		SetHomeTeamScore(homeScore).
		SetAwayTeamScore(awayScore).
		SetVersion(gm.Version + 1).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	// Create game event for goal/assist if applicable
	if req.Goals > 0 && req.Minute != nil && req.Second != nil {
		isCallahan := req.Goals > 0 && req.Assists == 0
		description := "Goal scored"
		if isCallahan {
			description = "Callahan goal scored"
		}

		_, err = tx.GameEvent.Create().
			SetEventType("goal_scored").
			SetMinute(*req.Minute).
			SetSecond(*req.Second).
			SetDescription(description).
			SetMetadata(map[string]interface{}{
				"player_id":   req.PlayerID,
				"goals":       req.Goals,
				"is_callahan": isCallahan,
			}).
			SetGameID(gameID).
			Save(ctx)
		if err != nil {
			return nil, err
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.invalidateGameCache(ctx, gameID)
	result, err := s.gameRepo.GetByIDWithRelations(ctx, updatedGm.ID)
	if err != nil {
		return nil, err
	}
	dto := mapGameToDTO(result)
	s.cacheGame(ctx, dto)
	return dto, nil
}

func (s *Service) GetGameScores(ctx context.Context, gameID uuid.UUID) ([]*ScoringDTO, error) {
	scores, err := s.scoringRepo.ListByGame(ctx, gameID)
	if err != nil {
		return nil, err
	}

	result := make([]*ScoringDTO, len(scores))
	for i, score := range scores {
		result[i] = mapScoringToDTO(score)
	}

	return result, nil
}

func mapScoringToDTO(s *ent.Scoring) *ScoringDTO {
	dto := &ScoringDTO{
		ID:        s.ID,
		Goals:     s.Goals,
		Assists:   s.Assists,
		Blocks:    s.Blocks,
		Turns:     s.Turns,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}

	if s.Edges.Player != nil {
		dto.PlayerID = s.Edges.Player.ID
		dto.PlayerName = s.Edges.Player.Name
		dto.PlayerNumber = s.Edges.Player.JerseyNumber

		// Include team information for split view display
		if len(s.Edges.Player.Edges.Teams) > 0 {
			// Try to find the team that matches the game's home or away team if Game edge is loaded
			var foundTeamID *uuid.UUID
			var foundTeamName string
			found := false

			if s.Edges.Game != nil {
				if s.Edges.Game.Edges.HomeTeam != nil {
					for _, t := range s.Edges.Player.Edges.Teams {
						if t.ID == s.Edges.Game.Edges.HomeTeam.ID {
							uid := t.ID
							foundTeamID = &uid
							foundTeamName = t.Name
							found = true
							break
						}
					}
				}
				if !found && s.Edges.Game.Edges.AwayTeam != nil {
					for _, t := range s.Edges.Player.Edges.Teams {
						if t.ID == s.Edges.Game.Edges.AwayTeam.ID {
							uid := t.ID
							foundTeamID = &uid
							foundTeamName = t.Name
							found = true
							break
						}
					}
				}
			}

			// Fallback to the first team if no match or game/teams not loaded adequately
			if !found {
				uid := s.Edges.Player.Edges.Teams[0].ID
				foundTeamID = &uid
				foundTeamName = s.Edges.Player.Edges.Teams[0].Name
			}

			dto.TeamID = foundTeamID
			dto.TeamName = foundTeamName
		}
	}

	return dto
}

func (s *Service) UpdateBulkScores(ctx context.Context, gameID uuid.UUID, userID uuid.UUID, req UpdateGameScoreRequest) (*GameDTO, error) {
	// Start transaction
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Get game with relations for verification and current version
	gm, err := tx.Game.Query().
		Where(game.ID(gameID)).
		WithHomeTeam().
		WithAwayTeam().
		WithScorekeeper().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrGameNotFound
		}
		return nil, err
	}

	// Verify authorization
	authorized := false
	role, _ := ctx.Value("user_role").(string)
	if role == "admin" || role == "event_manager" {
		authorized = true
	} else if gm.Edges.Scorekeeper != nil && gm.Edges.Scorekeeper.ID == userID {
		authorized = true
	}

	if !authorized {
		return nil, ErrUnauthorized
	}

	// Check status
	if gm.Status == "completed" {
		return nil, errors.New("cannot update scores for a completed game")
	}
	if gm.Status == "cancelled" {
		return nil, errors.New("cannot update scores for a cancelled game")
	}

	// Sync player scores
	// List existing scores to decide update vs create vs delete
	existingScores, err := tx.Scoring.Query().
		Where(scoring.HasGameWith(game.ID(gameID))).
		Where(scoring.DeletedAtIsNil()).
		WithPlayer(func(pq *ent.PlayerQuery) {
			pq.WithTeams()
		}).
		All(ctx)
	if err != nil {
		return nil, err
	}

	scoreMap := make(map[uuid.UUID]*ent.Scoring)
	for _, es := range existingScores {
		if es.Edges.Player != nil {
			scoreMap[es.Edges.Player.ID] = es
		}
	}

	processedPlayerIDs := make(map[uuid.UUID]bool)
	for _, ps := range req.PlayerScores {
		// Find player and verify team
		playerEntity, err := tx.Player.Query().
			Where(player.ID(ps.PlayerID)).
			WithTeams().
			Only(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get player %s: %w", ps.PlayerID, err)
		}

		var teamID uuid.UUID
		for _, t := range playerEntity.Edges.Teams {
			if t.ID == gm.Edges.HomeTeam.ID {
				teamID = gm.Edges.HomeTeam.ID
				break
			} else if t.ID == gm.Edges.AwayTeam.ID {
				teamID = gm.Edges.AwayTeam.ID
				break
			}
		}

		if teamID == uuid.Nil {
			return nil, fmt.Errorf("player %s does not belong to either team in this game", ps.PlayerID)
		}

		if existing, ok := scoreMap[ps.PlayerID]; ok {
			err = tx.Scoring.UpdateOneID(existing.ID).
				SetGoals(ps.Goals).
				SetAssists(ps.Assists).
				SetBlocks(ps.Blocks).
				SetTurns(ps.Turns).
				SetTeamID(teamID).
				SetUpdatedAt(time.Now()).
				Exec(ctx)
		} else {
			err = tx.Scoring.Create().
				SetGoals(ps.Goals).
				SetAssists(ps.Assists).
				SetBlocks(ps.Blocks).
				SetTurns(ps.Turns).
				SetTeamID(teamID).
				SetGameID(gameID).
				SetPlayerID(ps.PlayerID).
				Exec(ctx)
		}
		if err != nil {
			return nil, err
		}
		processedPlayerIDs[ps.PlayerID] = true
	}

	// Delete scores for players NOT in the request
	for playerID, existing := range scoreMap {
		if !processedPlayerIDs[playerID] {
			err = tx.Scoring.UpdateOneID(existing.ID).
				SetDeletedAt(time.Now()).
				Exec(ctx)
			if err != nil {
				return nil, err
			}
		}
	}

	// Recalculate totals directly in transaction
	scoreList, err := tx.Scoring.Query().
		Where(scoring.HasGameWith(game.ID(gameID))).
		Where(scoring.DeletedAtIsNil()).
		All(ctx)
	if err != nil {
		return nil, err
	}

	homeScore := 0
	awayScore := 0
	for _, sl := range scoreList {
		if sl.TeamID == gm.Edges.HomeTeam.ID {
			homeScore += sl.Goals
		} else if sl.TeamID == gm.Edges.AwayTeam.ID {
			awayScore += sl.Goals
		}
	}

	// Update game scores and version
	_, err = tx.Game.UpdateOneID(gameID).
		SetHomeTeamScore(homeScore).
		SetAwayTeamScore(awayScore).
		SetVersion(gm.Version + 1).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	// Create event
	_, err = tx.GameEvent.Create().
		SetEventType("bulk_score_update").
		SetMinute(0).
		SetSecond(0).
		SetDescription(fmt.Sprintf("Bulk score update: %s", req.Reason)).
		SetMetadata(map[string]interface{}{
			"reason":     req.Reason,
			"home_score": homeScore,
			"away_score": awayScore,
			"player_cnt": len(req.PlayerScores),
			"updated_by": userID,
		}).
		SetGameID(gameID).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.invalidateGameCache(ctx, gameID)
	// Fetch final result with all relations
	result, err := s.gameRepo.GetByIDWithRelations(ctx, gameID)
	if err != nil {
		return nil, err
	}
	dto := mapGameToDTO(result)
	s.cacheGame(ctx, dto)
	return dto, nil
}
