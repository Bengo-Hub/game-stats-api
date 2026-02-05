package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/bengobox/game-stats-api/internal/domain/audit"
	"github.com/bengobox/game-stats-api/internal/domain/game"
	"github.com/bengobox/game-stats-api/internal/domain/spiritscore"
	"github.com/bengobox/game-stats-api/internal/infrastructure/cache"
	"github.com/google/uuid"
)

// ScoreAdminService handles administrative score modifications
type ScoreAdminService struct {
	gameRepo        game.Repository
	spiritScoreRepo spiritscore.Repository
	auditRepo       audit.Repository
	cache           *cache.RedisClient
}

// NewScoreAdminService creates a new admin service
func NewScoreAdminService(
	gameRepo game.Repository,
	spiritScoreRepo spiritscore.Repository,
	auditRepo audit.Repository,
	cacheClient *cache.RedisClient,
) *ScoreAdminService {
	return &ScoreAdminService{
		gameRepo:        gameRepo,
		spiritScoreRepo: spiritScoreRepo,
		auditRepo:       auditRepo,
		cache:           cacheClient,
	}
}

// UpdateGameScoreRequest contains score update parameters
type UpdateGameScoreRequest struct {
	GameID      uuid.UUID `json:"game_id" validate:"required"`
	HomeScore   int       `json:"home_score" validate:"min=0"`
	AwayScore   int       `json:"away_score" validate:"min=0"`
	Reason      string    `json:"reason" validate:"required,min=10"`
	AdminUserID uuid.UUID `json:"admin_user_id" validate:"required"`
	AdminName   string    `json:"admin_name" validate:"required"`
	IPAddress   string    `json:"ip_address,omitempty"`
	UserAgent   string    `json:"user_agent,omitempty"`
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

// GetAuditHistory retrieves audit logs for a game
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
	if currentScore.Edges.Team != nil {
		cacheKey := cache.CacheKey("spirit", "team", currentScore.Edges.Team.ID.String())
		_ = s.cache.Delete(ctx, cacheKey)
	}
	if currentScore.Edges.Game != nil {
		cacheKey := cache.CacheKey("spirit", "game", currentScore.Edges.Game.ID.String())
		_ = s.cache.Delete(ctx, cacheKey)
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
