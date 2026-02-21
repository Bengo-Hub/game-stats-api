package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/internal/application/admin"
	"github.com/bengobox/game-stats-api/internal/domain/audit"
	"github.com/bengobox/game-stats-api/internal/domain/game"
	"github.com/bengobox/game-stats-api/internal/infrastructure/cache"
	"github.com/bengobox/game-stats-api/internal/infrastructure/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockGameRepository is a mock for game repository
type MockGameRepository struct {
	mock.Mock
}

func (m *MockGameRepository) GetByIDWithRelations(ctx context.Context, id uuid.UUID) (*ent.Game, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Game), args.Error(1)
}

func (m *MockGameRepository) Update(ctx context.Context, game *ent.Game) (*ent.Game, error) {
	args := m.Called(ctx, game)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.Game), args.Error(1)
}

func (m *MockGameRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.Game, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*ent.Game), args.Error(1)
}

func (m *MockGameRepository) Create(ctx context.Context, game *ent.Game) (*ent.Game, error) {
	return nil, nil
}

func (m *MockGameRepository) ListByDivision(ctx context.Context, divisionID uuid.UUID) ([]*ent.Game, error) {
	return nil, nil
}

func (m *MockGameRepository) ListByRound(ctx context.Context, roundID uuid.UUID) ([]*ent.Game, error) {
	return nil, nil
}

func (m *MockGameRepository) ListWithFilter(ctx context.Context, filter game.SearchFilter) ([]*ent.Game, error) {
	return nil, nil
}

func (m *MockGameRepository) ListByStatus(ctx context.Context, status string) ([]*ent.Game, error) {
	return nil, nil
}

func (m *MockGameRepository) ListByField(ctx context.Context, fieldID uuid.UUID) ([]*ent.Game, error) {
	return nil, nil
}

func (m *MockGameRepository) ListByDateRange(ctx context.Context, start, end time.Time) ([]*ent.Game, error) {
	return nil, nil
}

func (m *MockGameRepository) UpdateWithVersion(ctx context.Context, id uuid.UUID, version int, updateFn func(*ent.GameUpdateOne) *ent.GameUpdateOne) (*ent.Game, error) {
	return nil, nil
}

func (m *MockGameRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockGameRepository) CheckFieldConflict(ctx context.Context, fieldID uuid.UUID, scheduledTime time.Time, duration int) (bool, error) {
	return false, nil
}

func (m *MockGameRepository) List(ctx context.Context, limit, offset int) ([]*ent.Game, error) {
	return nil, nil
}

// MockSpiritScoreRepository is a mock for spirit score repository
type MockSpiritScoreRepository struct {
	mock.Mock
}

func (m *MockSpiritScoreRepository) Create(ctx context.Context, s *ent.SpiritScore) (*ent.SpiritScore, error) {
	return nil, nil
}

func (m *MockSpiritScoreRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.SpiritScore, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.SpiritScore), args.Error(1)
}

func (m *MockSpiritScoreRepository) ListByGame(ctx context.Context, gameID uuid.UUID) ([]*ent.SpiritScore, error) {
	return nil, nil
}

func (m *MockSpiritScoreRepository) ListByTeam(ctx context.Context, teamID uuid.UUID) ([]*ent.SpiritScore, error) {
	return nil, nil
}

func (m *MockSpiritScoreRepository) Update(ctx context.Context, s *ent.SpiritScore) (*ent.SpiritScore, error) {
	args := m.Called(ctx, s)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ent.SpiritScore), args.Error(1)
}

func (m *MockSpiritScoreRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func TestAdminHandler_UpdateGameScore(t *testing.T) {
	// Setup
	gameID := uuid.New()
	divisionID := uuid.New()
	adminUserID := uuid.New()

	mockGameRepo := new(MockGameRepository)
	mockSpiritScoreRepo := new(MockSpiritScoreRepository)
	auditRepo := repository.NewInMemoryAuditRepository()
	cacheClient, _ := cache.NewRedisClient("redis://localhost:6379/0")

	// Create service and handler
	adminService := admin.NewScoreAdminService(mockGameRepo, mockSpiritScoreRepo, auditRepo, cacheClient)
	handler := NewAdminHandler(adminService)

	t.Run("successful game score update", func(t *testing.T) {
		// Create mock game
		existingGame := &ent.Game{
			ID:            gameID,
			HomeTeamScore: 10,
			AwayTeamScore: 8,
			UpdatedAt:     time.Now(),
			Edges: ent.GameEdges{
				DivisionPool: &ent.DivisionPool{ID: divisionID},
			},
		}

		updatedGame := &ent.Game{
			ID:            gameID,
			HomeTeamScore: 12,
			AwayTeamScore: 10,
			UpdatedAt:     time.Now(),
			Edges: ent.GameEdges{
				DivisionPool: &ent.DivisionPool{ID: divisionID},
			},
		}

		mockGameRepo.On("GetByIDWithRelations", mock.Anything, gameID).Return(existingGame, nil)
		mockGameRepo.On("Update", mock.Anything, mock.AnythingOfType("*ent.Game")).Return(updatedGame, nil)

		// Create request
		requestBody := UpdateGameScoreRequestDTO{
			HomeScore: 12,
			AwayScore: 10,
			Reason:    "Score correction after official review",
		}
		bodyBytes, _ := json.Marshal(requestBody)

		req := httptest.NewRequest(http.MethodPut, "/admin/games/"+gameID.String()+"/score", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		// Add user context
		ctx := context.WithValue(req.Context(), "user_id", adminUserID)
		ctx = context.WithValue(ctx, "username", "admin_user")
		ctx = context.WithValue(ctx, "user_role", "admin")
		req = req.WithContext(ctx)

		// Create router with URL param
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", gameID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		// Execute
		handler.UpdateGameScore(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)

		var response admin.UpdateGameScoreResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, gameID, response.GameID)
		assert.Equal(t, 12, response.HomeScore)
		assert.Equal(t, 10, response.AwayScore)
		assert.NotEqual(t, uuid.Nil, response.AuditLogID)

		mockGameRepo.AssertExpectations(t)
	})

	t.Run("invalid game ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/admin/games/invalid-id/score", nil)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "invalid-id")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.UpdateGameScore(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("reason too short", func(t *testing.T) {
		requestBody := UpdateGameScoreRequestDTO{
			HomeScore: 12,
			AwayScore: 10,
			Reason:    "Short", // Less than 10 characters
		}
		bodyBytes, _ := json.Marshal(requestBody)

		req := httptest.NewRequest(http.MethodPut, "/admin/games/"+gameID.String()+"/score", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		ctx := context.WithValue(req.Context(), "user_id", adminUserID)
		ctx = context.WithValue(ctx, "username", "admin_user")
		req = req.WithContext(ctx)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", gameID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.UpdateGameScore(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAdminHandler_GetGameAuditHistory(t *testing.T) {
	gameID := uuid.New()
	adminUserID := uuid.New()

	mockGameRepo := new(MockGameRepository)
	mockSpiritScoreRepo := new(MockSpiritScoreRepository)
	auditRepo := repository.NewInMemoryAuditRepository()
	cacheClient, _ := cache.NewRedisClient("redis://localhost:6379/0")

	// Create some audit logs
	ctx := context.Background()
	log1 := audit.NewAuditLog("game", gameID, audit.ActionUpdate, adminUserID, "admin1")
	log1.AddChange("home_score", "10", "12")
	log1.SetMetadata("127.0.0.1", "Mozilla")
	auditRepo.Create(ctx, log1)

	log2 := audit.NewAuditLog("game", gameID, audit.ActionUpdate, adminUserID, "admin2")
	log2.AddChange("away_score", "8", "10")
	log2.SetMetadata("127.0.0.1", "Chrome")
	auditRepo.Create(ctx, log2)

	adminService := admin.NewScoreAdminService(mockGameRepo, mockSpiritScoreRepo, auditRepo, cacheClient)
	handler := NewAdminHandler(adminService)

	req := httptest.NewRequest(http.MethodGet, "/admin/games/"+gameID.String()+"/audit", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", gameID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.GetGameAuditHistory(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var logs []*audit.AuditLog
	err := json.NewDecoder(w.Body).Decode(&logs)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(logs))
}
