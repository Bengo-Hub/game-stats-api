package gamemanagement

import (
	"context"
	"errors"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

var (
	ErrDuplicateSpiritScore = errors.New("team has already submitted spirit score for this game")
	ErrInvalidSpiritScore   = errors.New("spirit score values must be between 0 and 4")
	ErrCannotScoreSelf      = errors.New("team cannot score itself")
)

// Spirit Score System
func (s *Service) SubmitSpiritScore(ctx context.Context, gameID uuid.UUID, userID uuid.UUID, req SubmitSpiritScoreRequest) (*SpiritScoreDTO, error) {
	// Get game
	game, err := s.gameRepo.GetByIDWithRelations(ctx, gameID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrGameNotFound
		}
		return nil, err
	}

	// Can only submit spirit scores for in_progress, ended or completed games
	if game.Status != "in_progress" && game.Status != "ended" && game.Status != "completed" {
		return nil, ErrInvalidGameStatus
	}

	// Validate teams are in the game
	if req.ScoredByTeamID != game.Edges.HomeTeam.ID && req.ScoredByTeamID != game.Edges.AwayTeam.ID {
		return nil, ErrTeamNotInGame
	}
	if req.TeamID != game.Edges.HomeTeam.ID && req.TeamID != game.Edges.AwayTeam.ID {
		return nil, ErrTeamNotInGame
	}

	// Team cannot score itself
	if req.ScoredByTeamID == req.TeamID {
		return nil, ErrCannotScoreSelf
	}

	// Validate score ranges (0-4)
	if req.RulesKnowledge < 0 || req.RulesKnowledge > 4 ||
		req.FoulsBodyContact < 0 || req.FoulsBodyContact > 4 ||
		req.FairMindedness < 0 || req.FairMindedness > 4 ||
		req.Attitude < 0 || req.Attitude > 4 ||
		req.Communication < 0 || req.Communication > 4 {
		return nil, ErrInvalidSpiritScore
	}

	// Check if spirit score already exists
	existingScores, err := s.spiritScoreRepo.ListByGame(ctx, gameID)
	if err != nil {
		return nil, err
	}

	for _, score := range existingScores {
		if score.Edges.ScoredByTeam != nil && score.Edges.ScoredByTeam.ID == req.ScoredByTeamID &&
			score.Edges.Team != nil && score.Edges.Team.ID == req.TeamID {
			return nil, ErrDuplicateSpiritScore
		}
	}

	// Get teams
	scoredByTeam, err := s.teamRepo.GetByID(ctx, req.ScoredByTeamID)
	if err != nil {
		return nil, err
	}

	team, err := s.teamRepo.GetByID(ctx, req.TeamID)
	if err != nil {
		return nil, err
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Create spirit score
	spiritScore := &ent.SpiritScore{
		RulesKnowledge:   req.RulesKnowledge,
		FoulsBodyContact: req.FoulsBodyContact,
		FairMindedness:   req.FairMindedness,
		Attitude:         req.Attitude,
		Communication:    req.Communication,
		Comments:         req.Comments,
		Edges: ent.SpiritScoreEdges{
			Game:         game,
			ScoredByTeam: scoredByTeam,
			Team:         team,
			SubmittedBy:  user,
		},
	}

	created, err := s.spiritScoreRepo.Create(ctx, spiritScore)
	if err != nil {
		return nil, err
	}

	// Handle MVP nominations
	if req.MVPMaleNomination != nil {
		if err := s.createMVPNomination(ctx, created, *req.MVPMaleNomination, "mvp_male"); err != nil {
			return nil, err
		}
	}
	if req.MVPFemaleNomination != nil {
		if err := s.createMVPNomination(ctx, created, *req.MVPFemaleNomination, "mvp_female"); err != nil {
			return nil, err
		}
	}

	// Handle Spirit nominations
	if req.SpiritMaleNomination != nil {
		if err := s.createSpiritNomination(ctx, created, *req.SpiritMaleNomination, "spirit_male"); err != nil {
			return nil, err
		}
	}
	if req.SpiritFemaleNomination != nil {
		if err := s.createSpiritNomination(ctx, created, *req.SpiritFemaleNomination, "spirit_female"); err != nil {
			return nil, err
		}
	}

	// Fetch with relations
	result, err := s.spiritScoreRepo.GetByID(ctx, created.ID)
	if err != nil {
		return nil, err
	}

	return mapSpiritScoreToDTO(result), nil
}

func (s *Service) GetGameSpiritScores(ctx context.Context, gameID uuid.UUID) ([]*SpiritScoreDTO, error) {
	scores, err := s.spiritScoreRepo.ListByGame(ctx, gameID)
	if err != nil {
		return nil, err
	}

	result := make([]*SpiritScoreDTO, len(scores))
	for i, score := range scores {
		result[i] = mapSpiritScoreToDTO(score)
	}

	return result, nil
}

func (s *Service) GetEventSpiritScores(ctx context.Context, eventID uuid.UUID, limit, offset int) ([]*SpiritScoreDTO, int, error) {
	scores, total, err := s.spiritScoreRepo.ListByEvent(ctx, eventID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*SpiritScoreDTO, len(scores))
	for i, score := range scores {
		result[i] = mapSpiritScoreToDTO(score)
	}

	return result, total, nil
}

func (s *Service) GetTeamSpiritAverage(ctx context.Context, teamID uuid.UUID) (*TeamSpiritAverageDTO, error) {
	// Get team
	team, err := s.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("team not found")
		}
		return nil, err
	}

	// Get all spirit scores received by this team
	scores, err := s.spiritScoreRepo.ListByTeam(ctx, teamID)
	if err != nil {
		return nil, err
	}

	// Get nomination counts
	mvpCount, _ := s.mvpNominationRepo.CountByTeam(ctx, teamID)
	spiritCount, _ := s.spiritNominationRepo.CountByTeam(ctx, teamID)

	if len(scores) == 0 {
		return &TeamSpiritAverageDTO{
			TeamID:                 teamID,
			TeamName:               team.Name,
			GamesPlayed:            0,
			AverageTotal:           0,
			MVPNominationsCount:    mvpCount,
			SpiritNominationsCount: spiritCount,
		}, nil
	}

	// Calculate averages
	var sumRules, sumFouls, sumFair, sumAttitude, sumComm int
	for _, score := range scores {
		sumRules += score.RulesKnowledge
		sumFouls += score.FoulsBodyContact
		sumFair += score.FairMindedness
		sumAttitude += score.Attitude
		sumComm += score.Communication
	}

	count := float64(len(scores))
	avgRules := float64(sumRules) / count
	avgFouls := float64(sumFouls) / count
	avgFair := float64(sumFair) / count
	avgAttitude := float64(sumAttitude) / count
	avgComm := float64(sumComm) / count
	avgTotal := avgRules + avgFouls + avgFair + avgAttitude + avgComm

	return &TeamSpiritAverageDTO{
		TeamID:                 teamID,
		TeamName:               team.Name,
		GamesPlayed:            len(scores),
		RulesKnowledge:         avgRules,
		FoulsBodyContact:       avgFouls,
		FairMindedness:         avgFair,
		Attitude:               avgAttitude,
		Communication:          avgComm,
		AverageTotal:           avgTotal,
		MVPNominationsCount:    mvpCount,
		SpiritNominationsCount: spiritCount,
	}, nil
}

func mapSpiritScoreToDTO(s *ent.SpiritScore) *SpiritScoreDTO {
	dto := &SpiritScoreDTO{
		ID:               s.ID,
		RulesKnowledge:   s.RulesKnowledge,
		FoulsBodyContact: s.FoulsBodyContact,
		FairMindedness:   s.FairMindedness,
		Attitude:         s.Attitude,
		Communication:    s.Communication,
		TotalScore:       s.RulesKnowledge + s.FoulsBodyContact + s.FairMindedness + s.Attitude + s.Communication,
		Comments:         s.Comments,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}

	if s.Edges.Game != nil {
		dto.GameID = s.Edges.Game.ID
	}

	if s.Edges.ScoredByTeam != nil {
		dto.ScoredByTeam = &TeamSummaryDTO{
			ID:             s.Edges.ScoredByTeam.ID,
			Name:           s.Edges.ScoredByTeam.Name,
			LogoURL:        s.Edges.ScoredByTeam.LogoURL,
			PrimaryColor:   s.Edges.ScoredByTeam.PrimaryColor,
			SecondaryColor: s.Edges.ScoredByTeam.SecondaryColor,
		}
	}

	if s.Edges.Team != nil {
		dto.Team = &TeamSummaryDTO{
			ID:             s.Edges.Team.ID,
			Name:           s.Edges.Team.Name,
			LogoURL:        s.Edges.Team.LogoURL,
			PrimaryColor:   s.Edges.Team.PrimaryColor,
			SecondaryColor: s.Edges.Team.SecondaryColor,
		}
	}

	if s.Edges.SubmittedBy != nil {
		dto.SubmittedBy = &UserSummaryDTO{
			ID:    s.Edges.SubmittedBy.ID,
			Name:  s.Edges.SubmittedBy.Name,
			Email: s.Edges.SubmittedBy.Email,
		}
	}

	// Map nominations
	for _, nom := range s.Edges.MvpNominations {
		if nom.Edges.Player != nil {
			playerDTO := &PlayerSummaryDTO{
				ID:           nom.Edges.Player.ID,
				Name:         nom.Edges.Player.Name,
				Gender:       nom.Edges.Player.Gender,
				JerseyNumber: nom.Edges.Player.JerseyNumber,
			}
			switch nom.Category {
			case "mvp_male":
				dto.MVPMaleNomination = playerDTO
			case "mvp_female":
				dto.MVPFemaleNomination = playerDTO
			}
		}
	}

	for _, nom := range s.Edges.SpiritNominations {
		if nom.Edges.Player != nil {
			playerDTO := &PlayerSummaryDTO{
				ID:           nom.Edges.Player.ID,
				Name:         nom.Edges.Player.Name,
				Gender:       nom.Edges.Player.Gender,
				JerseyNumber: nom.Edges.Player.JerseyNumber,
			}
			switch nom.Category {
			case "spirit_male":
				dto.SpiritMaleNomination = playerDTO
			case "spirit_female":
				dto.SpiritFemaleNomination = playerDTO
			}
		}
	}

	return dto
}
func (s *Service) createMVPNomination(ctx context.Context, spiritScore *ent.SpiritScore, playerID uuid.UUID, category string) error {
	player, err := s.playerRepo.GetByID(ctx, playerID)
	if err != nil {
		return err
	}

	mvpNomination := &ent.MVP_Nomination{
		Category: category,
		Edges: ent.MVP_NominationEdges{
			SpiritScore: spiritScore,
			Player:      player,
		},
	}

	_, err = s.mvpNominationRepo.Create(ctx, mvpNomination)
	return err
}

func (s *Service) createSpiritNomination(ctx context.Context, spiritScore *ent.SpiritScore, playerID uuid.UUID, category string) error {
	player, err := s.playerRepo.GetByID(ctx, playerID)
	if err != nil {
		return err
	}

	spiritNomination := &ent.SpiritNomination{
		Category: category,
		Edges: ent.SpiritNominationEdges{
			SpiritScore: spiritScore,
			Player:      player,
		},
	}

	_, err = s.spiritNominationRepo.Create(ctx, spiritNomination)
	return err
}
