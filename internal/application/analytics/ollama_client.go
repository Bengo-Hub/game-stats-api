package analytics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaClientInterface defines contract for Ollama LLM operations
type OllamaClientInterface interface {
	GenerateSQL(ctx context.Context, req GenerateSQLRequest) (*GenerateSQLResponse, error)
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	HealthCheck(ctx context.Context) error
}

// OllamaClient handles communication with Ollama LLM API
type OllamaClient struct {
	BaseURL    string
	Model      string
	httpClient *http.Client
}

// NewOllamaClient creates a new Ollama API client
func NewOllamaClient(baseURL, model string) *OllamaClient {
	return &OllamaClient{
		BaseURL: baseURL,
		Model:   model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // LLM responses can take longer
		},
	}
}

// GenerateSQLRequest contains parameters for SQL generation
type GenerateSQLRequest struct {
	Question      string                 `json:"question"`
	SchemaContext string                 `json:"schema_context"`
	TableContext  []TableMetadata        `json:"table_context"`
	Options       map[string]interface{} `json:"options,omitempty"`
}

// GenerateSQLResponse contains the generated SQL query
type GenerateSQLResponse struct {
	SQL         string  `json:"sql"`
	Explanation string  `json:"explanation"`
	Confidence  float64 `json:"confidence"`
}

// ChatRequest for general LLM chat
type ChatRequest struct {
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

// ChatMessage represents a single message in chat
type ChatMessage struct {
	Role    string `json:"role"` // system, user, assistant
	Content string `json:"content"`
}

// ChatResponse from Ollama
type ChatResponse struct {
	Model     string      `json:"model"`
	Message   ChatMessage `json:"message"`
	Done      bool        `json:"done"`
	TotalTime int64       `json:"total_duration"`
}

// TableMetadata contains schema information for a table
type TableMetadata struct {
	TableName     string         `json:"table_name"`
	Description   string         `json:"description"`
	Columns       []ColumnInfo   `json:"columns"`
	Relationships []Relationship `json:"relationships"`
}

// ColumnInfo describes a database column
type ColumnInfo struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Description  string `json:"description"`
	IsPrimaryKey bool   `json:"is_primary_key"`
	IsForeignKey bool   `json:"is_foreign_key"`
	IsNullable   bool   `json:"is_nullable"`
}

// Relationship describes foreign key relationships
type Relationship struct {
	TargetTable  string `json:"target_table"`
	TargetColumn string `json:"target_column"`
	RelationType string `json:"relation_type"` // one-to-one, one-to-many, many-to-many
}

// GenerateSQL converts natural language to SQL using duckdb-nsql model
func (c *OllamaClient) GenerateSQL(ctx context.Context, req GenerateSQLRequest) (*GenerateSQLResponse, error) {
	// Build prompt for duckdb-nsql model
	prompt := c.buildSQLPrompt(req)

	// Call Ollama generate endpoint
	ollamaReq := map[string]interface{}{
		"model":  c.Model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": 0.2, // Lower temperature for more deterministic SQL
			"top_p":       0.9,
		},
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var ollamaResp struct {
		Response string `json:"response"`
		Done     bool   `json:"done"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Parse the response to extract SQL and explanation
	sql, explanation := c.parseGeneratedSQL(ollamaResp.Response)

	// Calculate confidence based on SQL complexity and model response quality
	confidence := calculateSQLConfidence(sql, explanation)

	return &GenerateSQLResponse{
		SQL:         sql,
		Explanation: explanation,
		Confidence:  confidence,
	}, nil
}

// Chat performs a general chat conversation with the LLM
func (c *OllamaClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	ollamaReq := map[string]interface{}{
		"model":    c.Model,
		"messages": req.Messages,
		"stream":   req.Stream,
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &chatResp, nil
}

// HealthCheck verifies Ollama service is available
func (c *OllamaClient) HealthCheck(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("ollama is not reachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

// buildSQLPrompt constructs the prompt for SQL generation
func (c *OllamaClient) buildSQLPrompt(req GenerateSQLRequest) string {
	prompt := fmt.Sprintf("Given the following database schema:\n\n%s\n\n", req.SchemaContext)

	if len(req.TableContext) > 0 {
		prompt += "Table details:\n"
		for _, table := range req.TableContext {
			prompt += fmt.Sprintf("\nTable: %s\n", table.TableName)
			if table.Description != "" {
				prompt += fmt.Sprintf("Description: %s\n", table.Description)
			}
			prompt += "Columns:\n"
			for _, col := range table.Columns {
				prompt += fmt.Sprintf("  - %s (%s)", col.Name, col.Type)
				if col.IsPrimaryKey {
					prompt += " PRIMARY KEY"
				}
				if col.IsForeignKey {
					prompt += " FOREIGN KEY"
				}
				if col.Description != "" {
					prompt += fmt.Sprintf(" - %s", col.Description)
				}
				prompt += "\n"
			}
		}
		prompt += "\n"
	}

	prompt += fmt.Sprintf("Generate a SQL query to answer: %s\n\n", req.Question)
	prompt += "Provide the SQL query followed by a brief explanation."

	return prompt
}

// parseGeneratedSQL extracts SQL and explanation from LLM response
func (c *OllamaClient) parseGeneratedSQL(response string) (sql string, explanation string) {
	// Simple parsing - look for SQL between ```sql and ```
	// This can be enhanced based on actual model output format

	sqlStart := -1
	sqlEnd := -1

	// Look for SQL code blocks
	if bytes.Contains([]byte(response), []byte("```sql")) {
		sqlStart = bytes.Index([]byte(response), []byte("```sql")) + 6
		remaining := response[sqlStart:]
		sqlEnd = bytes.Index([]byte(remaining), []byte("```"))
		if sqlEnd != -1 {
			sql = remaining[:sqlEnd]
			explanation = remaining[sqlEnd+3:]
		}
	} else {
		// Fallback: use entire response as SQL
		sql = response
	}

	// Clean up
	sql = string(bytes.TrimSpace([]byte(sql)))
	explanation = string(bytes.TrimSpace([]byte(explanation)))

	return sql, explanation
}

// calculateSQLConfidence estimates query quality based on structure
func calculateSQLConfidence(sql string, explanation string) float64 {
	confidence := 0.5 // Base confidence

	// Higher confidence if SQL is not empty
	if len(sql) > 0 {
		confidence += 0.2
	}

	// Higher confidence if explanation provided
	if len(explanation) > 10 {
		confidence += 0.15
	}

	// Higher confidence if SQL contains standard keywords
	if bytes.Contains([]byte(sql), []byte("SELECT")) ||
		bytes.Contains([]byte(sql), []byte("select")) {
		confidence += 0.15
	}

	// Cap at 0.95 to indicate LLM uncertainty
	if confidence > 0.95 {
		confidence = 0.95
	}

	return confidence
}
