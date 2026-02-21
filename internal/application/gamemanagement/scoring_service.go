package gamemanagement

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

// Scoring System
func (s *Service) RecordScore(ctx context.Context, gameID uuid.UUID, userID uuid.UUID, req RecordScoreRequest) (*GameDTO, error) {
	game, err := s.gameRepo.GetByIDWithRelations(ctx, gameID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrGameNotFound
		}
		return nil, err
	}

	// Verify scorekeeper
	if game.Edges.Scorekeeper == nil || game.Edges.Scorekeeper.ID != userID {
		return nil, ErrUnauthorized
	}

	// Can only record scores during in-progress or finished games
	if game.Status != "in_progress" && game.Status != "finished" {
		return nil, ErrInvalidGameStatus
	}

	// Verify player exists
	player, err := s.playerRepo.GetByID(ctx, req.PlayerID)
	if err != nil {
		return nil, err
	}

	// Check if scoring record exists for this player in this game
	existingScores, err := s.scoringRepo.ListByGame(ctx, gameID)
	if err != nil {
		return nil, err
	}

	var existingScore *ent.Scoring
	for _, score := range existingScores {
		if score.Edges.Player != nil && score.Edges.Player.ID == req.PlayerID {
			existingScore = score
			break
		}
	}

	if existingScore != nil {
		// Update existing score
		existingScore.Goals = req.Goals
		existingScore.Assists = req.Assists
		existingScore.Blocks = req.Blocks
		existingScore.Turns = req.Turns

		_, err = s.scoringRepo.Update(ctx, existingScore)
		if err != nil {
			return nil, err
		}
	} else {
		// Create new score
		scoreEntity := &ent.Scoring{
			Goals:   req.Goals,
			Assists: req.Assists,
			Blocks:  req.Blocks,
			Turns:   req.Turns,
			Edges: ent.ScoringEdges{
				Game:   game,
				Player: player,
			},
		}

		_, err = s.scoringRepo.Create(ctx, scoreEntity)
		if err != nil {
			return nil, err
		}
	}

	// Recalculate game totals
	updatedScores, err := s.scoringRepo.ListByGame(ctx, gameID)
	if err != nil {
		return nil, err
	}

	homeScore := 0
	awayScore := 0

	for _, score := range updatedScores {
		// Determine team from player's team association
		if score.Edges.Player != nil && score.Edges.Player.Edges.Team != nil {
			playerTeamID := score.Edges.Player.Edges.Team.ID
			if game.Edges.HomeTeam != nil && playerTeamID == game.Edges.HomeTeam.ID {
				homeScore += score.Goals
			} else if game.Edges.AwayTeam != nil && playerTeamID == game.Edges.AwayTeam.ID {
				awayScore += score.Goals
			}
		}
	}

	// Update game scores with optimistic locking
	updatedGame, err := s.gameRepo.UpdateWithVersion(ctx, gameID, game.Version, func(update *ent.GameUpdateOne) *ent.GameUpdateOne {
		return update.
			SetHomeTeamScore(homeScore).
			SetAwayTeamScore(awayScore)
	})
	if err != nil {
		return nil, err
	}

	// Create game event for goal if this was a new goal
	if req.Goals > 0 && req.Minute != nil && req.Second != nil {
		// Detect Callahan: goal scored without an assist (interception in end zone)
		isCallahan := req.Goals > 0 && req.Assists == 0
		description := "Goal scored"
		if isCallahan {
			description = "Callahan goal scored"
		}

		_, err = s.gameEventRepo.Create(ctx, &ent.GameEvent{
			EventType:   "goal_scored",
			Minute:      *req.Minute,
			Second:      *req.Second,
			Description: description,
			Metadata: map[string]interface{}{
				"player_id":   req.PlayerID,
				"goals":       req.Goals,
				"is_callahan": isCallahan,
			},
			Edges: ent.GameEventEdges{
				Game: updatedGame,
			},
		})
		if err != nil {
			return nil, err
		}
	}

	// Create event for assist
	if req.Assists > 0 && req.Minute != nil && req.Second != nil {
		_, err = s.gameEventRepo.Create(ctx, &ent.GameEvent{
			EventType:   "assist_recorded",
			Minute:      *req.Minute,
			Second:      *req.Second,
			Description: "Assist recorded",
			Metadata: map[string]interface{}{
				"player_id": req.PlayerID,
				"assists":   req.Assists,
			},
			Edges: ent.GameEventEdges{
				Game: updatedGame,
			},
		})
		if err != nil {
			return nil, err
		}
	}

	// SSE events are broadcast via the stream handler when clients poll for updates

	result, err := s.gameRepo.GetByIDWithRelations(ctx, updatedGame.ID)
	if err != nil {
		return nil, err
	}

	return mapGameToDTO(result), nil
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
		if s.Edges.Player.Edges.Team != nil {
			teamID := s.Edges.Player.Edges.Team.ID
			dto.TeamID = &teamID
			dto.TeamName = s.Edges.Player.Edges.Team.Name
		}
	}

	return dto
}
