package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSupersetClient mocks the Superset client
type MockSupersetClient struct {
	mock.Mock
}

func (m *MockSupersetClient) Login(ctx context.Context) (*LoginResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LoginResponse), args.Error(1)
}

func (m *MockSupersetClient) GenerateGuestToken(ctx context.Context, accessToken string, req GuestTokenRequest) (string, error) {
	args := m.Called(ctx, accessToken, req)
	return args.String(0), args.Error(1)
}

func (m *MockSupersetClient) GetDashboards(ctx context.Context, accessToken string) ([]Dashboard, error) {
	args := m.Called(ctx, accessToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Dashboard), args.Error(1)
}

func (m *MockSupersetClient) GetDashboard(ctx context.Context, accessToken string, dashboardUUID uuid.UUID) (*Dashboard, error) {
	args := m.Called(ctx, accessToken, dashboardUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Dashboard), args.Error(1)
}

func (m *MockSupersetClient) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestGenerateEmbedToken_Success(t *testing.T) {
	// Setup
	mockClient := new(MockSupersetClient)
	service := &Service{
		supersetClient: mockClient,
	}

	dashboardUUID := uuid.New()
	userID := uuid.New()
	eventID := uuid.New()
	teamID := uuid.New()

	request := GenerateEmbedTokenRequest{
		DashboardUUID: dashboardUUID,
		UserID:        userID,
		EventID:       &eventID,
		TeamIDs:       []uuid.UUID{teamID},
		Username:      "test@example.com",
		FirstName:     "Test",
		LastName:      "User",
	}

	// Setup mocks
	mockClient.On("Login", mock.Anything).Return(&LoginResponse{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
	}, nil)

	mockClient.On("GenerateGuestToken", mock.Anything, "test-access-token", mock.Anything).
		Return("test-guest-token", nil)

	// Execute
	ctx := context.Background()
	response, err := service.GenerateEmbedToken(ctx, request)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "test-guest-token", response.Token)
	assert.Equal(t, dashboardUUID, response.DashboardUUID)
	assert.True(t, response.ExpiresAt.After(time.Now()))

	mockClient.AssertExpectations(t)
}

func TestGenerateEmbedToken_AuthenticationFailure(t *testing.T) {
	// Setup
	mockClient := new(MockSupersetClient)
	service := &Service{
		supersetClient: mockClient,
	}

	request := GenerateEmbedTokenRequest{
		DashboardUUID: uuid.New(),
		UserID:        uuid.New(),
		Username:      "test@example.com",
		FirstName:     "Test",
		LastName:      "User",
	}

	// Setup mocks - login fails
	mockClient.On("Login", mock.Anything).Return(nil, assert.AnError)

	// Execute
	ctx := context.Background()
	response, err := service.GenerateEmbedToken(ctx, request)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to authenticate")

	mockClient.AssertExpectations(t)
}

func TestBuildRLSRules(t *testing.T) {
	service := &Service{}

	t.Run("With event ID", func(t *testing.T) {
		eventID := uuid.New()
		rules := service.buildRLSRules(&eventID, nil)

		assert.Len(t, rules, 1)
		assert.Contains(t, rules[0].Clause, "event_id")
		assert.Contains(t, rules[0].Clause, eventID.String())
	})

	t.Run("With team IDs", func(t *testing.T) {
		teamIDs := []uuid.UUID{uuid.New(), uuid.New()}
		rules := service.buildRLSRules(nil, teamIDs)

		assert.Len(t, rules, 1)
		assert.Contains(t, rules[0].Clause, "team_id IN")
		assert.Contains(t, rules[0].Clause, teamIDs[0].String())
		assert.Contains(t, rules[0].Clause, teamIDs[1].String())
	})

	t.Run("With both event and team IDs", func(t *testing.T) {
		eventID := uuid.New()
		teamIDs := []uuid.UUID{uuid.New()}
		rules := service.buildRLSRules(&eventID, teamIDs)

		assert.Len(t, rules, 2)
		assert.Contains(t, rules[0].Clause, "event_id")
		assert.Contains(t, rules[1].Clause, "team_id IN")
	})

	t.Run("With no filters", func(t *testing.T) {
		rules := service.buildRLSRules(nil, nil)
		assert.Len(t, rules, 0)
	})
}

func TestListDashboards_Success(t *testing.T) {
	// Setup
	mockClient := new(MockSupersetClient)
	service := &Service{
		supersetClient: mockClient,
	}

	expectedDashboards := []Dashboard{
		{
			ID:            1,
			DashboardUUID: uuid.New(),
			Title:         "Event Overview",
			Slug:          "event-overview",
			Published:     true,
		},
		{
			ID:            2,
			DashboardUUID: uuid.New(),
			Title:         "Player Stats",
			Slug:          "player-stats",
			Published:     true,
		},
	}

	// Setup mocks
	mockClient.On("Login", mock.Anything).Return(&LoginResponse{
		AccessToken: "test-token",
	}, nil)

	mockClient.On("GetDashboards", mock.Anything, "test-token").
		Return(expectedDashboards, nil)

	// Execute
	ctx := context.Background()
	response, err := service.ListDashboards(ctx)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 2, response.Total)
	assert.Len(t, response.Dashboards, 2)
	assert.Equal(t, "Event Overview", response.Dashboards[0].Title)

	mockClient.AssertExpectations(t)
}

func TestGetDashboard_Success(t *testing.T) {
	// Setup
	mockClient := new(MockSupersetClient)
	service := &Service{
		supersetClient: mockClient,
	}

	dashboardUUID := uuid.New()
	expectedDashboard := &Dashboard{
		ID:            1,
		DashboardUUID: dashboardUUID,
		Title:         "Event Overview",
		Slug:          "event-overview",
		Published:     true,
	}

	// Setup mocks
	mockClient.On("Login", mock.Anything).Return(&LoginResponse{
		AccessToken: "test-token",
	}, nil)

	mockClient.On("GetDashboard", mock.Anything, "test-token", dashboardUUID).
		Return(expectedDashboard, nil)

	// Execute
	ctx := context.Background()
	dashboard, err := service.GetDashboard(ctx, dashboardUUID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, dashboard)
	assert.Equal(t, dashboardUUID, dashboard.DashboardUUID)
	assert.Equal(t, "Event Overview", dashboard.Title)

	mockClient.AssertExpectations(t)
}

func TestHealthCheck(t *testing.T) {
	// Setup
	mockClient := new(MockSupersetClient)
	service := &Service{
		supersetClient: mockClient,
	}

	mockClient.On("HealthCheck", mock.Anything).Return(nil)

	// Execute
	ctx := context.Background()
	err := service.HealthCheck(ctx)

	// Assert
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}
