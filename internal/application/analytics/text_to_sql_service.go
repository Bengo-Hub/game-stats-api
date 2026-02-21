package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/google/uuid"
)

// TextToSQLService handles natural language to SQL conversion
type TextToSQLService struct {
	ollamaClient OllamaClientInterface
	dbClient     *ent.Client
	db           *sql.DB // Raw database connection for query execution
}

// NewTextToSQLService creates a new text-to-SQL service
func NewTextToSQLService(ollamaClient OllamaClientInterface, dbClient *ent.Client, db *sql.DB) *TextToSQLService {
	return &TextToSQLService{
		ollamaClient: ollamaClient,
		dbClient:     dbClient,
		db:           db,
	}
}

// NaturalLanguageQueryRequest contains the user's question
type NaturalLanguageQueryRequest struct {
	Question string     `json:"question" validate:"required"`
	EventID  *uuid.UUID `json:"event_id,omitempty"`
	UserID   uuid.UUID  `json:"user_id" validate:"required"`
	Context  string     `json:"context,omitempty"` // e.g., "pool_play", "playoffs"
}

// NaturalLanguageQueryResponse contains the query results
type NaturalLanguageQueryResponse struct {
	Question    string                   `json:"question"`
	SQL         string                   `json:"sql"`
	Explanation string                   `json:"explanation"`
	Results     []map[string]interface{} `json:"results"`
	Confidence  float64                  `json:"confidence"`
	Warning     string                   `json:"warning,omitempty"`
}

// ProcessNaturalLanguageQuery converts natural language to SQL and executes it
func (s *TextToSQLService) ProcessNaturalLanguageQuery(ctx context.Context, req NaturalLanguageQueryRequest) (*NaturalLanguageQueryResponse, error) {
	// Get relevant schema context
	schemaContext := s.getSchemaContext(req.EventID)

	// Build table metadata for the query
	tableContext := s.buildTableContext(req.Context)

	// Generate SQL using Ollama
	sqlReq := GenerateSQLRequest{
		Question:      req.Question,
		SchemaContext: schemaContext,
		TableContext:  tableContext,
	}

	sqlResp, err := s.ollamaClient.GenerateSQL(ctx, sqlReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate SQL: %w", err)
	}

	// Validate SQL for safety (prevent DROP, DELETE, UPDATE, etc.)
	if err := s.validateSQL(sqlResp.SQL); err != nil {
		return &NaturalLanguageQueryResponse{
			Question:    req.Question,
			SQL:         sqlResp.SQL,
			Explanation: sqlResp.Explanation,
			Confidence:  sqlResp.Confidence,
			Warning:     fmt.Sprintf("SQL validation failed: %v", err),
		}, nil
	}

	// Apply row-level security filters
	secureSQL := s.applyRLSFilters(sqlResp.SQL, req.EventID)

	// Execute SQL query
	results, err := s.executeSQL(ctx, secureSQL)
	if err != nil {
		return &NaturalLanguageQueryResponse{
			Question:    req.Question,
			SQL:         secureSQL,
			Explanation: sqlResp.Explanation,
			Confidence:  sqlResp.Confidence,
			Warning:     fmt.Sprintf("Query execution failed: %v", err),
		}, nil
	}

	return &NaturalLanguageQueryResponse{
		Question:    req.Question,
		SQL:         secureSQL,
		Explanation: sqlResp.Explanation,
		Results:     results,
		Confidence:  sqlResp.Confidence,
	}, nil
}

// getSchemaContext returns a text representation of the database schema
func (s *TextToSQLService) getSchemaContext(eventID *uuid.UUID) string {
	schema := `
Database Schema for DigiGameStats:

Core Tables:
- events: Tournaments/competitions (id, name, start_date, end_date, event_type, status)
- teams: Teams participating in events (id, name, division_id, seed)
- players: Individual players (id, first_name, last_name, jersey_number, team_id)
- games: Scheduled matches (id, home_team_id, away_team_id, game_round_id, start_time, status)
- game_rounds: Round structure (id, event_id, round_number, round_type, pool_id)
- divisions_pools: Divisions/pools within events (id, event_id, name, skill_level)

Scoring Tables:
- scoring: Goal records (id, game_id, player_id, scoring_type, timestamp)
- game_events: Game timeline events (id, game_id, event_type, timestamp, metadata)

Spirit of the Game:
- spirit_scores: Team spirit evaluations (id, game_id, evaluating_team_id, rules_knowledge, fouls, fair_mindedness, etc.)
- spirit_of_game_nominations: Spirit award nominations (id, game_id, nominated_player_id, nominating_team_id)
- mvp_nominations: MVP nominations (id, game_id, nominated_player_id)

Rankings & Standings:
- division_ranking_criteria: Ranking configuration (id, division_id, wins_weight, point_diff_weight, spirit_weight)

Common Queries:
- Team standings by division
- Player statistics (goals, assists)
- Spirit score averages
- Game schedules and results
- Playoff brackets
`

	if eventID != nil {
		schema += fmt.Sprintf("\nContext: Querying data for event_id = '%s'\n", eventID.String())
	}

	return schema
}

// buildTableContext provides detailed metadata for relevant tables
func (s *TextToSQLService) buildTableContext(context string) []TableMetadata {
	tables := []TableMetadata{
		{
			TableName:   "games",
			Description: "Scheduled matches between teams",
			Columns: []ColumnInfo{
				{Name: "id", Type: "uuid", IsPrimaryKey: true},
				{Name: "home_team_id", Type: "uuid", IsForeignKey: true},
				{Name: "away_team_id", Type: "uuid", IsForeignKey: true},
				{Name: "game_round_id", Type: "uuid", IsForeignKey: true},
				{Name: "home_score", Type: "integer"},
				{Name: "away_score", Type: "integer"},
				{Name: "status", Type: "string"},
			},
		},
		{
			TableName:   "teams",
			Description: "Teams participating in events",
			Columns: []ColumnInfo{
				{Name: "id", Type: "uuid", IsPrimaryKey: true},
				{Name: "name", Type: "string"},
				{Name: "division_id", Type: "uuid", IsForeignKey: true},
				{Name: "seed", Type: "integer"},
			},
		},
		{
			TableName:   "players",
			Description: "Individual players on teams",
			Columns: []ColumnInfo{
				{Name: "id", Type: "uuid", IsPrimaryKey: true},
				{Name: "first_name", Type: "string"},
				{Name: "last_name", Type: "string"},
				{Name: "jersey_number", Type: "integer"},
				{Name: "team_id", Type: "uuid", IsForeignKey: true},
			},
		},
		{
			TableName:   "spirit_scores",
			Description: "Spirit of the Game evaluations",
			Columns: []ColumnInfo{
				{Name: "id", Type: "uuid", IsPrimaryKey: true},
				{Name: "game_id", Type: "uuid", IsForeignKey: true},
				{Name: "evaluating_team_id", Type: "uuid", IsForeignKey: true},
				{Name: "rules_knowledge", Type: "integer"},
				{Name: "fouls_and_body_contact", Type: "integer"},
				{Name: "fair_mindedness", Type: "integer"},
				{Name: "positive_attitude", Type: "integer"},
				{Name: "communication", Type: "integer"},
				{Name: "total_score", Type: "integer"},
			},
		},
	}

	return tables
}

// validateSQL ensures the query is safe (read-only)
func (s *TextToSQLService) validateSQL(sql string) error {
	sqlUpper := strings.ToUpper(strings.TrimSpace(sql))

	// Check for dangerous operations
	dangerousKeywords := []string{
		"DROP", "DELETE", "UPDATE", "INSERT", "ALTER",
		"TRUNCATE", "CREATE", "GRANT", "REVOKE",
		"EXEC", "EXECUTE", "CALL",
	}

	for _, keyword := range dangerousKeywords {
		if strings.Contains(sqlUpper, keyword) {
			return fmt.Errorf("SQL contains forbidden keyword: %s", keyword)
		}
	}

	// Ensure it starts with SELECT
	if !strings.HasPrefix(sqlUpper, "SELECT") && !strings.HasPrefix(sqlUpper, "WITH") {
		return fmt.Errorf("only SELECT queries are allowed")
	}

	return nil
}

// applyRLSFilters adds row-level security filters to the SQL
func (s *TextToSQLService) applyRLSFilters(sql string, eventID *uuid.UUID) string {
	if eventID == nil {
		return sql
	}

	// Simple RLS: Add WHERE clause for event_id
	// This is a basic implementation - production would parse SQL AST
	sqlUpper := strings.ToUpper(sql)

	if strings.Contains(sqlUpper, "FROM GAMES") || strings.Contains(sqlUpper, "FROM TEAMS") {
		// Add event filtering via JOIN if not already present
		if !strings.Contains(sqlUpper, "WHERE") {
			sql += fmt.Sprintf(" WHERE event_id = '%s'", eventID.String())
		} else if !strings.Contains(sqlUpper, "EVENT_ID") {
			sql += fmt.Sprintf(" AND event_id = '%s'", eventID.String())
		}
	}

	return sql
}

// executeSQL runs the validated SQL query
func (s *TextToSQLService) executeSQL(ctx context.Context, sql string) ([]map[string]interface{}, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	rows, err := s.db.QueryContext(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	results := make([]map[string]interface{}, 0)

	// Limit results to prevent memory issues
	const maxRows = 1000
	rowCount := 0

	for rows.Next() && rowCount < maxRows {
		// Create slice of interface{} to hold values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert to map
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// Handle []byte for string conversion
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
		rowCount++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return results, nil
}
