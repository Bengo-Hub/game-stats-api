package gamemanagement

import (
	"context"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

// GameRound Management
func (s *Service) CreateGameRound(ctx context.Context, req CreateGameRoundRequest) (*GameRoundDTO, error) {
	// Validate event exists
	event, err := s.eventRepo.GetByID(ctx, req.EventID)
	if err != nil {
		return nil, err
	}

	roundEntity := &ent.GameRound{
		Name:        req.Name,
		RoundType:   req.RoundType,
		RoundNumber: req.RoundNumber,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Edges: ent.GameRoundEdges{
			Event: event,
		},
	}

	created, err := s.gameRoundRepo.Create(ctx, roundEntity)
	if err != nil {
		return nil, err
	}

	return mapGameRoundToDTO(created), nil
}

func (s *Service) GetGameRound(ctx context.Context, id uuid.UUID) (*GameRoundDTO, error) {
	round, err := s.gameRoundRepo.GetByIDWithGames(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrGameRoundNotFound
		}
		return nil, err
	}

	dto := mapGameRoundToDTO(round)
	if round.Edges.Games != nil {
		dto.GamesCount = len(round.Edges.Games)
	}

	return dto, nil
}

func (s *Service) ListGameRounds(ctx context.Context, eventID uuid.UUID) ([]*GameRoundDTO, error) {
	rounds, err := s.gameRoundRepo.ListByEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}

	result := make([]*GameRoundDTO, len(rounds))
	for i, round := range rounds {
		result[i] = mapGameRoundToDTO(round)
	}

	return result, nil
}

func (s *Service) UpdateGameRound(ctx context.Context, id uuid.UUID, req UpdateGameRoundRequest) (*GameRoundDTO, error) {
	round, err := s.gameRoundRepo.GetByID(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrGameRoundNotFound
		}
		return nil, err
	}

	if req.Name != nil {
		round.Name = *req.Name
	}
	if req.RoundType != nil {
		round.RoundType = *req.RoundType
	}
	if req.RoundNumber != nil {
		round.RoundNumber = req.RoundNumber
	}
	if req.StartDate != nil {
		round.StartDate = req.StartDate
	}
	if req.EndDate != nil {
		round.EndDate = req.EndDate
	}

	updated, err := s.gameRoundRepo.Update(ctx, round)
	if err != nil {
		return nil, err
	}

	return mapGameRoundToDTO(updated), nil
}

func mapGameRoundToDTO(r *ent.GameRound) *GameRoundDTO {
	dto := &GameRoundDTO{
		ID:          r.ID,
		Name:        r.Name,
		RoundType:   r.RoundType,
		RoundNumber: r.RoundNumber,
		StartDate:   r.StartDate,
		EndDate:     r.EndDate,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}

	if r.Edges.Event != nil {
		dto.EventID = r.Edges.Event.ID
	}

	return dto
}
