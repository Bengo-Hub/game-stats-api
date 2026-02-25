package gamemanagement

import (
	"context"
	"errors"
	"fmt"
	"time"

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

	// Verify authorization
	isAuthorized, err := s.permissionService.IsScorekeeperForGame(ctx, userID, gameID)
	if err != nil {
		return nil, err
	}
	if !isAuthorized {
		return nil, ErrUnauthorized
	}

	// Verify player exists
	player, err := s.playerRepo.GetByID(ctx, req.PlayerID)
	if err != nil {
		return nil, err
	}

	// Check for Score Approval requirement
	// If game time has elapsed, scorekeeping might need approval
	gameEndTime := game.ScheduledTime.Add(time.Duration(game.AllocatedTimeMinutes) * time.Minute)
	if game.ActualEndTime != nil {
		gameEndTime = *game.ActualEndTime
	}

	if time.Now().After(gameEndTime) && game.Status != "ended" {
		// Fetch event to check approval configuration
		event, err := s.eventRepo.GetByID(ctx, game.Edges.DivisionPool.Edges.Event.ID)
		if err == nil {
			// If approval is required, but user is NOT an admin/manager, we should handle this
			// For now, we allow admins/managers to bypass, and others might be restricted
			isManager, _ := s.permissionService.CheckPermission(ctx, userID, event.ScoreEditApprovalRole, "event", event.ID)
			if !isManager {
				// Calculate proposed new scores based on the request
				newHomeScore := game.HomeTeamScore
				newAwayScore := game.AwayTeamScore

				// Create ScoreEditRequest
				_, err := s.scoringRepo.CreateScoreEditRequest(ctx, &ent.ScoreEditRequest{
					GameID:            gameID,
					RequestedByID:     userID,
					PreviousHomeScore: game.HomeTeamScore,
					PreviousAwayScore: game.AwayTeamScore,
					NewHomeScore:      newHomeScore,
					NewAwayScore:      newAwayScore,
					PlayerScores: []map[string]interface{}{
						{
							"player_id":   req.PlayerID.String(),
							"player_name": player.Name,
							"goals":       req.Goals,
						},
					},
					Reason: fmt.Sprintf("Post-game score adjustment: Player %s goals set to %d", player.Name, req.Goals),
					Status: "pending",
				})
				if err != nil {
					return nil, err
				}
				return nil, errors.New("score edit requires approval after game time has elapsed")
			}
		}
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
		existingScore.TeamID = req.TeamID

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
			TeamID:  req.TeamID,
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

	// Recalculate game totals using shared domain service
	homeScore, awayScore, err := s.scoreDomainService.RecalculateTotals(ctx, gameID, game.Edges.HomeTeam.ID, game.Edges.AwayTeam.ID)
	if err != nil {
		return nil, err
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
