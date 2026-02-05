package analytics

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockOllamaClient mocks the Ollama client
type MockOllamaClient struct {
	mock.Mock
}

func (m *MockOllamaClient) GenerateSQL(ctx context.Context, req GenerateSQLRequest) (*GenerateSQLResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GenerateSQLResponse), args.Error(1)
}

func (m *MockOllamaClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ChatResponse), args.Error(1)
}

func (m *MockOllamaClient) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestProcessNaturalLanguageQuery_Success(t *testing.T) {
	// Setup
	mockClient := new(MockOllamaClient)
	service := NewTextToSQLService(mockClient, nil, nil)

	expectedSQL := "SELECT name, COUNT(*) as game_count FROM teams GROUP BY name ORDER BY game_count DESC LIMIT 5"
	mockClient.On("GenerateSQL", mock.Anything, mock.AnythingOfType("GenerateSQLRequest")).
		Return(&GenerateSQLResponse{
			SQL:         expectedSQL,
			Explanation: "This query retrieves the top 5 teams by number of games played",
			Confidence:  0.92,
		}, nil)

	// Execute
	ctx := context.Background()
	req := NaturalLanguageQueryRequest{
		Question: "What are the top 5 teams by number of games played?",
	}

	response, err := service.ProcessNaturalLanguageQuery(ctx, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Contains(t, response.SQL, "SELECT")
	assert.Equal(t, "What are the top 5 teams by number of games played?", response.Question)
	assert.Greater(t, response.Confidence, 0.8)

	mockClient.AssertExpectations(t)
}

func TestValidateSQL_BlocksDangerousQueries(t *testing.T) {
	service := NewTextToSQLService(nil, nil, nil)

	tests := []struct {
		name      string
		sql       string
		shouldErr bool
	}{
		{
			name:      "Valid SELECT query",
			sql:       "SELECT * FROM teams",
			shouldErr: false,
		},
		{
			name:      "Valid SELECT with JOIN",
			sql:       "SELECT t.name, COUNT(g.id) FROM teams t JOIN games g ON t.id = g.home_team_id GROUP BY t.name",
			shouldErr: false,
		},
		{
			name:      "Valid WITH CTE",
			sql:       "WITH team_stats AS (SELECT team_id, COUNT(*) FROM games GROUP BY team_id) SELECT * FROM team_stats",
			shouldErr: false,
		},
		{
			name:      "Blocks DELETE",
			sql:       "DELETE FROM teams WHERE id = '123'",
			shouldErr: true,
		},
		{
			name:      "Blocks UPDATE",
			sql:       "UPDATE teams SET name = 'Hacked' WHERE id = '123'",
			shouldErr: true,
		},
		{
			name:      "Blocks DROP",
			sql:       "DROP TABLE teams",
			shouldErr: true,
		},
		{
			name:      "Blocks INSERT",
			sql:       "INSERT INTO teams (name) VALUES ('Hacked')",
			shouldErr: true,
		},
		{
			name:      "Blocks TRUNCATE",
			sql:       "TRUNCATE TABLE teams",
			shouldErr: true,
		},
		{
			name:      "Blocks EXEC",
			sql:       "EXEC sp_executesql N'SELECT 1'",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateSQL(tt.sql)
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuildSQLPrompt(t *testing.T) {
	client := NewOllamaClient("http://localhost:11434", "duckdb-nsql:7b")

	req := GenerateSQLRequest{
		Question:      "Show me the top 3 scorers",
		SchemaContext: "teams(id, name), players(id, name, team_id)",
		TableContext: []TableMetadata{
			{
				TableName:   "scoring",
				Description: "Goal scoring records",
				Columns: []ColumnInfo{
					{Name: "id", Type: "uuid", IsPrimaryKey: true},
					{Name: "player_id", Type: "uuid", IsForeignKey: true},
					{Name: "game_id", Type: "uuid", IsForeignKey: true},
				},
			},
		},
	}

	prompt := client.buildSQLPrompt(req)

	assert.Contains(t, prompt, "top 3 scorers")
	assert.Contains(t, prompt, "scoring")
	assert.Contains(t, prompt, "player_id")
	assert.Contains(t, prompt, "teams(id, name)")
}

func TestParseGeneratedSQL(t *testing.T) {
	client := NewOllamaClient("http://localhost:11434", "duckdb-nsql:7b")

	tests := []struct {
		name            string
		response        string
		expectedSQL     string
		expectedHasExpl bool
	}{
		{
			name:            "SQL with code blocks",
			response:        "Here's the query:\n\n```sql\nSELECT name, COUNT(*) as goals\nFROM players p\nJOIN scoring s ON p.id = s.player_id\nGROUP BY name\nORDER BY goals DESC\nLIMIT 3\n```\n\nThis query joins players with scoring records and counts goals per player.",
			expectedSQL:     "SELECT name, COUNT(*) as goals",
			expectedHasExpl: true,
		},
		{
			name:            "Plain SQL without markers",
			response:        "SELECT * FROM teams WHERE division_id = '123'",
			expectedSQL:     "SELECT * FROM teams WHERE division_id = '123'",
			expectedHasExpl: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, explanation := client.parseGeneratedSQL(tt.response)
			assert.Contains(t, sql, "SELECT")
			if tt.expectedHasExpl {
				assert.NotEmpty(t, explanation)
			}
		})
	}
}

func TestGetSchemaContext(t *testing.T) {
	service := NewTextToSQLService(nil, nil, nil)

	schema := service.getSchemaContext(nil)

	assert.Contains(t, schema, "events")
	assert.Contains(t, schema, "teams")
	assert.Contains(t, schema, "players")
	assert.Contains(t, schema, "spirit_scores")
	assert.Contains(t, schema, "game_rounds")
}

func TestApplyRLSFilters(t *testing.T) {
	service := NewTextToSQLService(nil, nil, nil)

	tests := []struct {
		name     string
		sql      string
		hasWhere bool
	}{
		{
			name:     "SQL without WHERE",
			sql:      "SELECT * FROM games",
			hasWhere: false,
		},
		{
			name:     "SQL with existing WHERE",
			sql:      "SELECT * FROM games WHERE status = 'completed'",
			hasWhere: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventID, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
			result := service.applyRLSFilters(tt.sql, &eventID)

			assert.Contains(t, result, eventID.String())
			if tt.hasWhere {
				assert.Contains(t, result, "AND")
			} else {
				assert.Contains(t, result, "WHERE")
			}
		})
	}
}
