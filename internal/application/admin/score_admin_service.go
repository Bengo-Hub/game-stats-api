package admin

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/internal/domain/audit"
	"github.com/bengobox/game-stats-api/internal/domain/game"
	"github.com/bengobox/game-stats-api/internal/domain/scoring"
	"github.com/bengobox/game-stats-api/internal/domain/spiritscore"
	"github.com/bengobox/game-stats-api/internal/infrastructure/cache"
	"github.com/google/uuid"
)

// ScoreAdminService handles administrative score modifications
type ScoreAdminService struct {
	gameRepo        game.Repository
	spiritScoreRepo spiritscore.Repository
	scoringRepo     scoring.Repository
	scoreService    *scoring.ScoreService
	auditRepo       audit.Repository
	cache           *cache.RedisClient
}

// NewScoreAdminService creates a new admin service
func NewScoreAdminService(
	gameRepo game.Repository,
	spiritScoreRepo spiritscore.Repository,
	scoringRepo scoring.Repository,
	auditRepo audit.Repository,
	cacheClient *cache.RedisClient,
) *ScoreAdminService {
	return &ScoreAdminService{
		gameRepo:        gameRepo,
		spiritScoreRepo: spiritScoreRepo,
		scoringRepo:     scoringRepo,
		scoreService:    scoring.NewScoreService(scoringRepo),
		auditRepo:       auditRepo,
		cache:           cacheClient,
	}
}

// PlayerScore adjustment
type PlayerScore struct {
	PlayerID uuid.UUID `json:"player_id"`
	Goals    int       `json:"goals"`
}

// UpdateGameScoreRequest contains score update parameters
type UpdateGameScoreRequest struct {
	GameID       uuid.UUID     `json:"game_id" validate:"required"`
	HomeScore    int           `json:"home_score" validate:"min=0"`
	AwayScore    int           `json:"away_score" validate:"min=0"`
	Reason       string        `json:"reason" validate:"required,min=10"`
	PlayerScores []PlayerScore `json:"player_scores,omitempty"`
	AdminUserID  uuid.UUID     `json:"admin_user_id" validate:"required"`
	AdminName    string        `json:"admin_name" validate:"required"`
	IPAddress    string        `json:"ip_address,omitempty"`
	UserAgent    string        `json:"user_agent,omitempty"`
}

// Validate validates the request
func (r *UpdateGameScoreRequest) Validate() error {
	if r.GameID == uuid.Nil {
		return fmt.Errorf("game_id is required")
	}
	if r.HomeScore < 0 {
		return fmt.Errorf("home_score must be non-negative")
	}
	if r.AwayScore < 0 {
		return fmt.Errorf("away_score must be non-negative")
	}
	if len(r.Reason) < 10 {
		return fmt.Errorf("reason must be at least 10 characters")
	}
	if r.AdminUserID == uuid.Nil {
		return fmt.Errorf("admin_user_id is required")
	}
	if r.AdminName == "" {
		return fmt.Errorf("admin_name is required")
	}
	return nil
}

// UpdateGameScoreResponse contains the updated game
type UpdateGameScoreResponse struct {
	GameID     uuid.UUID `json:"game_id"`
	HomeScore  int       `json:"home_score"`
	AwayScore  int       `json:"away_score"`
	UpdatedAt  time.Time `json:"updated_at"`
	AuditLogID uuid.UUID `json:"audit_log_id"`
}

// UpdateGameScore updates a game score with audit trail
func (s *ScoreAdminService) UpdateGameScore(
	ctx context.Context,
	req UpdateGameScoreRequest,
) (*UpdateGameScoreResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Get current game with relations
	currentGame, err := s.gameRepo.GetByIDWithRelations(ctx, req.GameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	// Track changes for audit
	changes := make(map[string]audit.ChangeEntry)
	if currentGame.HomeTeamScore != req.HomeScore {
		changes["home_score"] = audit.ChangeEntry{
			OldValue: fmt.Sprintf("%d", currentGame.HomeTeamScore),
			NewValue: fmt.Sprintf("%d", req.HomeScore),
		}
	}
	if currentGame.AwayTeamScore != req.AwayScore {
		changes["away_score"] = audit.ChangeEntry{
			OldValue: fmt.Sprintf("%d", currentGame.AwayTeamScore),
			NewValue: fmt.Sprintf("%d", req.AwayScore),
		}
	}

	if len(changes) == 0 {
		return nil, fmt.Errorf("no changes detected")
	}

	// Create audit log
	auditLogID, err := CreateAuditLog(
		ctx,
		s.auditRepo,
		"game",
		req.GameID,
		audit.ActionUpdate,
		req.AdminUserID,
		req.AdminName,
		changes,
		req.Reason,
		req.IPAddress,
		req.UserAgent,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create audit log: %w", err)
	}

	// Update individual player scores if provided
	if len(req.PlayerScores) > 0 {
		existingScores, err := s.scoringRepo.ListByGame(ctx, req.GameID)
		if err != nil {
			return nil, fmt.Errorf("failed to list existing scores: %w", err)
		}

		scoreMap := make(map[uuid.UUID]*ent.Scoring)
		for _, es := range existingScores {
			if es.Edges.Player != nil {
				scoreMap[es.Edges.Player.ID] = es
			}
		}

		for _, ps := range req.PlayerScores {
			if existing, ok := scoreMap[ps.PlayerID]; ok {
				existing.Goals = ps.Goals
				_, err = s.scoringRepo.Update(ctx, existing)
			} else {
				_, err = s.scoringRepo.Create(ctx, &ent.Scoring{
					Goals: ps.Goals,
					Edges: ent.ScoringEdges{
						Game:   &ent.Game{ID: req.GameID},
						Player: &ent.Player{ID: ps.PlayerID},
					},
				})
			}
			if err != nil {
				return nil, fmt.Errorf("failed to update score for player %s: %w", ps.PlayerID, err)
			}
		}

		// Recalculate totals after updating player scores
		homeTotal, awayTotal, err := s.scoreService.RecalculateTotals(ctx, req.GameID, currentGame.Edges.HomeTeam.ID, currentGame.Edges.AwayTeam.ID)
		if err == nil {
			// If admin provided explicit scores that differ from calculated, we'll use calculated
			req.HomeScore = homeTotal
			req.AwayScore = awayTotal

			// Re-check changes for audit
			if currentGame.HomeTeamScore != req.HomeScore {
				changes["home_score"] = audit.ChangeEntry{
					OldValue: fmt.Sprintf("%d", currentGame.HomeTeamScore),
					NewValue: fmt.Sprintf("%d", req.HomeScore),
				}
			} else {
				delete(changes, "home_score")
			}

			if currentGame.AwayTeamScore != req.AwayScore {
				changes["away_score"] = audit.ChangeEntry{
					OldValue: fmt.Sprintf("%d", currentGame.AwayTeamScore),
					NewValue: fmt.Sprintf("%d", req.AwayScore),
				}
			} else {
				delete(changes, "away_score")
			}
		}
	}

	// Update the game scores
	currentGame.HomeTeamScore = req.HomeScore
	currentGame.AwayTeamScore = req.AwayScore

	updatedGame, err := s.gameRepo.Update(ctx, currentGame)
	if err != nil {
		return nil, fmt.Errorf("failed to update game: %w", err)
	}

	// Invalidate caches
	if err := InvalidateCaches(ctx, s.cache, req.GameID, updatedGame); err != nil {
		// Log but don't fail - cache invalidation is non-critical
		fmt.Printf("Warning: failed to invalidate caches: %v\n", err)
	}

	return &UpdateGameScoreResponse{
		GameID:     updatedGame.ID,
		HomeScore:  updatedGame.HomeTeamScore,
		AwayScore:  updatedGame.AwayTeamScore,
		UpdatedAt:  updatedGame.UpdatedAt,
		AuditLogID: auditLogID,
	}, nil
}

func (s *ScoreAdminService) GetAuditHistory(
	ctx context.Context,
	gameID uuid.UUID,
) ([]*audit.AuditLog, error) {
	logs, err := s.auditRepo.GetByEntity(ctx, "game", gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit history: %w", err)
	}
	return logs, nil
}

// GetScoreEdits retrieves all game score edit logs
func (s *ScoreAdminService) GetScoreEdits(ctx context.Context) ([]*audit.AuditLog, error) {
	logs, err := s.auditRepo.GetByEntity(ctx, "game", uuid.Nil) // uuid.Nil meaning global or all for that entity type
	if err != nil {
		// Fallback: list recent if GetByEntity doesn't support Nil
		return s.auditRepo.GetRecent(ctx, 100)
	}
	return logs, nil
}

// ListPendingScoreEdits retrieves all pending score edit requests
func (s *ScoreAdminService) ListPendingScoreEdits(ctx context.Context) ([]*ent.ScoreEditRequest, error) {
	return s.scoringRepo.ListScoreEditRequests(ctx, "pending")
}

// ReviewScoreEdit approves or rejects a score edit request
func (s *ScoreAdminService) ReviewScoreEdit(
	ctx context.Context,
	requestID uuid.UUID,
	approve bool,
	reviewerID uuid.UUID,
	rejectionReason string,
) error {
	req, err := s.scoringRepo.GetScoreEditRequestByID(ctx, requestID)
	if err != nil {
		return fmt.Errorf("failed to get score edit request: %w", err)
	}

	if req.Status != "pending" {
		return fmt.Errorf("request is already %s", req.Status)
	}

	if approve {
		// Extract player scores if present
		var playerScores []PlayerScore
		for _, ps := range req.PlayerScores {
			pID, _ := uuid.Parse(fmt.Sprint(ps["player_id"]))
			gVal, _ := strconv.Atoi(fmt.Sprint(ps["goals"]))
			playerScores = append(playerScores, PlayerScore{
				PlayerID: pID,
				Goals:    gVal,
			})
		}

		updateReq := UpdateGameScoreRequest{
			GameID:       req.GameID,
			HomeScore:    req.NewHomeScore,
			AwayScore:    req.NewAwayScore,
			Reason:       fmt.Sprintf("Approved score edit request: %s", req.Reason),
			AdminUserID:  reviewerID,
			AdminName:    "System Reviewer", // Should fetch reviewer name
			PlayerScores: playerScores,
		}

		_, err = s.UpdateGameScore(ctx, updateReq)
		if err != nil {
			return fmt.Errorf("failed to apply approved score edit: %w", err)
		}

		req.Status = "approved"
	} else {
		req.Status = "rejected"
		req.RejectionReason = rejectionReason
	}

	req.ReviewedByID = reviewerID
	req.ReviewedAt = time.Now()

	_, err = s.scoringRepo.UpdateScoreEditRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update score edit request status: %w", err)
	}

	return nil
}

// SyncGameScores delegates to repository
func (s *ScoreAdminService) SyncGameScores(ctx context.Context, gameID uuid.UUID) (*UpdateGameScoreResponse, error) {
	g, err := s.gameRepo.SyncGameScores(ctx, gameID)
	if err != nil {
		return nil, err
	}

	return &UpdateGameScoreResponse{
		GameID:    g.ID,
		HomeScore: g.HomeTeamScore,
		AwayScore: g.AwayTeamScore,
		UpdatedAt: g.UpdatedAt,
	}, nil
}

// UpdateSpiritScoreRequest contains spirit score update parameters
type UpdateSpiritScoreRequest struct {
	SpiritScoreID  uuid.UUID `json:"spirit_score_id" validate:"required"`
	RulesKnowledge int       `json:"rules_knowledge" validate:"min=0,max=4"`
	Fouls          int       `json:"fouls" validate:"min=0,max=4"`
	FairMindedness int       `json:"fair_mindedness" validate:"min=0,max=4"`
	Attitude       int       `json:"attitude" validate:"min=0,max=4"`
	Communication  int       `json:"communication" validate:"min=0,max=4"`
	Reason         string    `json:"reason" validate:"required,min=10"`
	AdminUserID    uuid.UUID `json:"admin_user_id" validate:"required"`
	AdminName      string    `json:"admin_name" validate:"required"`
	IPAddress      string    `json:"ip_address,omitempty"`
	UserAgent      string    `json:"user_agent,omitempty"`
}

// UpdateSpiritScoreResponse contains the updated spirit score
type UpdateSpiritScoreResponse struct {
	SpiritScoreID  uuid.UUID `json:"spirit_score_id"`
	RulesKnowledge int       `json:"rules_knowledge"`
	Fouls          int       `json:"fouls"`
	FairMindedness int       `json:"fair_mindedness"`
	Attitude       int       `json:"attitude"`
	Communication  int       `json:"communication"`
	TotalScore     int       `json:"total_score"`
	UpdatedAt      time.Time `json:"updated_at"`
	AuditLogID     uuid.UUID `json:"audit_log_id"`
}

// Validate validates the spirit score update request
func (r *UpdateSpiritScoreRequest) Validate() error {
	if r.SpiritScoreID == uuid.Nil {
		return fmt.Errorf("spirit_score_id is required")
	}
	if r.RulesKnowledge < 0 || r.RulesKnowledge > 4 {
		return fmt.Errorf("rules_knowledge must be between 0 and 4")
	}
	if r.Fouls < 0 || r.Fouls > 4 {
		return fmt.Errorf("fouls must be between 0 and 4")
	}
	if r.FairMindedness < 0 || r.FairMindedness > 4 {
		return fmt.Errorf("fair_mindedness must be between 0 and 4")
	}
	if r.Attitude < 0 || r.Attitude > 4 {
		return fmt.Errorf("attitude must be between 0 and 4")
	}
	if r.Communication < 0 || r.Communication > 4 {
		return fmt.Errorf("communication must be between 0 and 4")
	}
	if len(r.Reason) < 10 {
		return fmt.Errorf("reason must be at least 10 characters")
	}
	if r.AdminUserID == uuid.Nil {
		return fmt.Errorf("admin_user_id is required")
	}
	if r.AdminName == "" {
		return fmt.Errorf("admin_name is required")
	}
	return nil
}

// UpdateSpiritScore updates a spirit score with audit trail
func (s *ScoreAdminService) UpdateSpiritScore(
	ctx context.Context,
	req UpdateSpiritScoreRequest,
) (*UpdateSpiritScoreResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Get current spirit score
	currentScore, err := s.spiritScoreRepo.GetByID(ctx, req.SpiritScoreID)
	if err != nil {
		return nil, fmt.Errorf("failed to get spirit score: %w", err)
	}

	// Track changes for audit
	changes := make(map[string]audit.ChangeEntry)
	if currentScore.RulesKnowledge != req.RulesKnowledge {
		changes["rules_knowledge"] = audit.ChangeEntry{
			OldValue: currentScore.RulesKnowledge,
			NewValue: req.RulesKnowledge,
		}
	}
	if currentScore.FoulsBodyContact != req.Fouls {
		changes["fouls_body_contact"] = audit.ChangeEntry{
			OldValue: currentScore.FoulsBodyContact,
			NewValue: req.Fouls,
		}
	}
	if currentScore.FairMindedness != req.FairMindedness {
		changes["fair_mindedness"] = audit.ChangeEntry{
			OldValue: currentScore.FairMindedness,
			NewValue: req.FairMindedness,
		}
	}
	if currentScore.Attitude != req.Attitude {
		changes["attitude"] = audit.ChangeEntry{
			OldValue: currentScore.Attitude,
			NewValue: req.Attitude,
		}
	}
	if currentScore.Communication != req.Communication {
		changes["communication"] = audit.ChangeEntry{
			OldValue: currentScore.Communication,
			NewValue: req.Communication,
		}
	}

	if len(changes) == 0 {
		return nil, fmt.Errorf("no changes detected")
	}

	// Create audit log
	auditLogID, err := CreateAuditLog(
		ctx,
		s.auditRepo,
		"spirit_score",
		req.SpiritScoreID,
		audit.ActionUpdate,
		req.AdminUserID,
		req.AdminName,
		changes,
		req.Reason,
		req.IPAddress,
		req.UserAgent,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create audit log: %w", err)
	}

	// Update the spirit score
	currentScore.RulesKnowledge = req.RulesKnowledge
	currentScore.FoulsBodyContact = req.Fouls
	currentScore.FairMindedness = req.FairMindedness
	currentScore.Attitude = req.Attitude
	currentScore.Communication = req.Communication

	updatedScore, err := s.spiritScoreRepo.Update(ctx, currentScore)
	if err != nil {
		return nil, fmt.Errorf("failed to update spirit score: %w", err)
	}

	// Calculate total score
	totalScore := updatedScore.RulesKnowledge + updatedScore.FoulsBodyContact +
		updatedScore.FairMindedness + updatedScore.Attitude + updatedScore.Communication

	// Invalidate spirit-related caches
	if s.cache != nil {
		if currentScore.Edges.Team != nil {
			cacheKey := cache.CacheKey("spirit", "team", currentScore.Edges.Team.ID.String())
			_ = s.cache.Delete(ctx, cacheKey)
		}
		if currentScore.Edges.Game != nil {
			cacheKey := cache.CacheKey("spirit", "game", currentScore.Edges.Game.ID.String())
			_ = s.cache.Delete(ctx, cacheKey)
		}
	}

	return &UpdateSpiritScoreResponse{
		SpiritScoreID:  updatedScore.ID,
		RulesKnowledge: updatedScore.RulesKnowledge,
		Fouls:          updatedScore.FoulsBodyContact,
		FairMindedness: updatedScore.FairMindedness,
		Attitude:       updatedScore.Attitude,
		Communication:  updatedScore.Communication,
		TotalScore:     totalScore,
		UpdatedAt:      updatedScore.UpdatedAt,
		AuditLogID:     auditLogID,
	}, nil
}

// GetSpiritScoreAuditHistory retrieves audit logs for a spirit score
func (s *ScoreAdminService) GetSpiritScoreAuditHistory(
	ctx context.Context,
	spiritScoreID uuid.UUID,
) ([]*audit.AuditLog, error) {
	logs, err := s.auditRepo.GetByEntity(ctx, "spirit_score", spiritScoreID)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit history: %w", err)
	}
	return logs, nil
}
