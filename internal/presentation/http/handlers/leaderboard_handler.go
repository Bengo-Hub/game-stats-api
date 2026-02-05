package handlers

import (
	"net/http"

	"github.com/bengobox/game-stats-api/ent"
	"github.com/bengobox/game-stats-api/ent/scoring"
	"github.com/bengobox/game-stats-api/ent/spiritscore"
)

type LeaderboardHandler struct {
	client *ent.Client
}

func NewLeaderboardHandler(client *ent.Client) *LeaderboardHandler {
	return &LeaderboardHandler{client: client}
}

// PlayerStatResponse represents a player stat in leaderboard
type PlayerStatResponse struct {
	PlayerId    string `json:"playerId"`
	PlayerName  string `json:"playerName"`
	TeamId      string `json:"teamId"`
	TeamName    string `json:"teamName"`
	Goals       int    `json:"goals"`
	Assists     int    `json:"assists"`
	GamesPlayed int    `json:"gamesPlayed"`
}

// SpiritLeaderboardResponse represents a team's spirit score in leaderboard
type SpiritLeaderboardResponse struct {
	TeamId       string                  `json:"teamId"`
	TeamName     string                  `json:"teamName"`
	AverageScore float64                 `json:"averageScore"`
	GamesPlayed  int                     `json:"gamesPlayed"`
	Breakdown    *SpiritBreakdownAverage `json:"breakdown,omitempty"`
}

type SpiritBreakdownAverage struct {
	RulesKnowledge   float64 `json:"rulesKnowledge"`
	FoulsBodyContact float64 `json:"foulsBodyContact"`
	FairMindedness   float64 `json:"fairMindedness"`
	Attitude         float64 `json:"attitude"`
	Communication    float64 `json:"communication"`
}

// GetPlayerLeaderboard godoc
// @Summary Get player leaderboard
// @Description Get top players by goals or assists
// @Tags leaderboards
// @Produce json
// @Param category query string false "Category: goals or assists" default(goals)
// @Param eventId query string false "Filter by event ID" format(uuid)
// @Param divisionPoolId query string false "Filter by division pool ID" format(uuid)
// @Param limit query int false "Limit results" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {array} PlayerStatResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /public/leaderboards/players [get]
func (h *LeaderboardHandler) GetPlayerLeaderboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse parameters
	category := r.URL.Query().Get("category")
	if category == "" {
		category = "goals"
	}

	pagination := ParsePagination(r)

	// Query scoring records to calculate player stats
	query := h.client.Scoring.Query().
		Where(scoring.DeletedAtIsNil()).
		WithPlayer(func(pq *ent.PlayerQuery) {
			pq.WithTeam()
		}).
		WithGame()

	scores, err := query.All(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch leaderboard data")
		return
	}

	// Aggregate by player
	playerStats := make(map[string]*PlayerStatResponse)
	playerGames := make(map[string]map[string]bool) // player -> set of game IDs

	for _, score := range scores {
		if score.Edges.Player == nil {
			continue
		}

		player := score.Edges.Player
		playerID := player.ID.String()

		if _, exists := playerStats[playerID]; !exists {
			teamName := "Unknown Team"
			teamID := ""
			if player.Edges.Team != nil {
				teamName = player.Edges.Team.Name
				teamID = player.Edges.Team.ID.String()
			}

			playerStats[playerID] = &PlayerStatResponse{
				PlayerId:   playerID,
				PlayerName: player.Name,
				TeamId:     teamID,
				TeamName:   teamName,
				Goals:      0,
				Assists:    0,
			}
			playerGames[playerID] = make(map[string]bool)
		}

		// Aggregate stats
		playerStats[playerID].Goals += score.Goals
		playerStats[playerID].Assists += score.Assists

		// Track games played
		if score.Edges.Game != nil {
			playerGames[playerID][score.Edges.Game.ID.String()] = true
		}
	}

	// Update games played count
	for playerID, stat := range playerStats {
		stat.GamesPlayed = len(playerGames[playerID])
	}

	// Convert to slice and sort
	result := make([]PlayerStatResponse, 0, len(playerStats))
	for _, stat := range playerStats {
		result = append(result, *stat)
	}

	// Sort by the category
	if category == "goals" {
		// Sort by goals descending
		for i := 0; i < len(result)-1; i++ {
			for j := i + 1; j < len(result); j++ {
				if result[j].Goals > result[i].Goals {
					result[i], result[j] = result[j], result[i]
				}
			}
		}
	} else {
		// Sort by assists descending
		for i := 0; i < len(result)-1; i++ {
			for j := i + 1; j < len(result); j++ {
				if result[j].Assists > result[i].Assists {
					result[i], result[j] = result[j], result[i]
				}
			}
		}
	}

	// Limit results
	if len(result) > pagination.Limit {
		result = result[:pagination.Limit]
	}

	respondJSON(w, http.StatusOK, result)
}

// GetSpiritLeaderboard godoc
// @Summary Get spirit leaderboard
// @Description Get teams ranked by spirit scores
// @Tags leaderboards
// @Produce json
// @Param eventId query string false "Filter by event ID" format(uuid)
// @Param divisionPoolId query string false "Filter by division pool ID" format(uuid)
// @Param limit query int false "Limit results" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {array} SpiritLeaderboardResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /public/leaderboards/spirit [get]
func (h *LeaderboardHandler) GetSpiritLeaderboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	pagination := ParsePagination(r)

	// Query all spirit scores with team info
	scores, err := h.client.SpiritScore.Query().
		Where(spiritscore.DeletedAtIsNil()).
		WithTeam().
		WithGame().
		All(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch spirit leaderboard")
		return
	}

	// Aggregate by team
	type teamAggregation struct {
		teamName         string
		totalScore       float64
		count            int
		rulesKnowledge   float64
		foulsBodyContact float64
		fairMindedness   float64
		attitude         float64
		communication    float64
		games            map[string]bool
	}

	teamStats := make(map[string]*teamAggregation)

	for _, score := range scores {
		if score.Edges.Team == nil {
			continue
		}

		team := score.Edges.Team
		teamID := team.ID.String()

		if _, exists := teamStats[teamID]; !exists {
			teamStats[teamID] = &teamAggregation{
				teamName: team.Name,
				games:    make(map[string]bool),
			}
		}

		agg := teamStats[teamID]

		// Calculate total score from individual components
		totalScore := score.RulesKnowledge + score.FoulsBodyContact + score.FairMindedness + score.Attitude + score.Communication

		agg.totalScore += float64(totalScore)
		agg.count++
		agg.rulesKnowledge += float64(score.RulesKnowledge)
		agg.foulsBodyContact += float64(score.FoulsBodyContact)
		agg.fairMindedness += float64(score.FairMindedness)
		agg.attitude += float64(score.Attitude)
		agg.communication += float64(score.Communication)

		if score.Edges.Game != nil {
			agg.games[score.Edges.Game.ID.String()] = true
		}
	}

	// Convert to response
	result := make([]SpiritLeaderboardResponse, 0, len(teamStats))
	for teamID, agg := range teamStats {
		if agg.count == 0 {
			continue
		}

		avgScore := agg.totalScore / float64(agg.count)
		resp := SpiritLeaderboardResponse{
			TeamId:       teamID,
			TeamName:     agg.teamName,
			AverageScore: avgScore,
			GamesPlayed:  len(agg.games),
			Breakdown: &SpiritBreakdownAverage{
				RulesKnowledge:   agg.rulesKnowledge / float64(agg.count),
				FoulsBodyContact: agg.foulsBodyContact / float64(agg.count),
				FairMindedness:   agg.fairMindedness / float64(agg.count),
				Attitude:         agg.attitude / float64(agg.count),
				Communication:    agg.communication / float64(agg.count),
			},
		}
		result = append(result, resp)
	}

	// Sort by average score descending
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].AverageScore > result[i].AverageScore {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	// Limit results
	if len(result) > pagination.Limit {
		result = result[:pagination.Limit]
	}

	respondJSON(w, http.StatusOK, result)
}
