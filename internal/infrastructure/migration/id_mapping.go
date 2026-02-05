package migration

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/google/uuid"
)

// IDMapping stores mappings between legacy Django integer PKs and new UUIDs
type IDMapping struct {
	mu          sync.RWMutex
	Teams       map[int]uuid.UUID `json:"teams"`
	Players     map[int]uuid.UUID `json:"players"`
	Games       map[int]uuid.UUID `json:"games"`
	GameRounds  map[int]uuid.UUID `json:"game_rounds"`
	Fields      map[int]uuid.UUID `json:"fields"`
	Divisions   map[int]uuid.UUID `json:"divisions"`
	Locations   map[int]uuid.UUID `json:"locations"`
	Countries   map[int]uuid.UUID `json:"countries"`
	Continents  map[int]uuid.UUID `json:"continents"`
	Disciplines map[int]uuid.UUID `json:"disciplines"`
	Events      map[int]uuid.UUID `json:"events"`
	Users       map[int]uuid.UUID `json:"users"`
}

// NewIDMapping creates a new ID mapping instance with initialized maps
func NewIDMapping() *IDMapping {
	return &IDMapping{
		Teams:       make(map[int]uuid.UUID),
		Players:     make(map[int]uuid.UUID),
		Games:       make(map[int]uuid.UUID),
		GameRounds:  make(map[int]uuid.UUID),
		Fields:      make(map[int]uuid.UUID),
		Divisions:   make(map[int]uuid.UUID),
		Locations:   make(map[int]uuid.UUID),
		Countries:   make(map[int]uuid.UUID),
		Continents:  make(map[int]uuid.UUID),
		Disciplines: make(map[int]uuid.UUID),
		Events:      make(map[int]uuid.UUID),
		Users:       make(map[int]uuid.UUID),
	}
}

// SetTeam stores a team ID mapping
func (m *IDMapping) SetTeam(legacyID int, newID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Teams[legacyID] = newID
}

// GetTeam retrieves a team UUID by legacy ID
func (m *IDMapping) GetTeam(legacyID int) (uuid.UUID, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.Teams[legacyID]
	return id, ok
}

// SetPlayer stores a player ID mapping
func (m *IDMapping) SetPlayer(legacyID int, newID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Players[legacyID] = newID
}

// GetPlayer retrieves a player UUID by legacy ID
func (m *IDMapping) GetPlayer(legacyID int) (uuid.UUID, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.Players[legacyID]
	return id, ok
}

// SetGame stores a game ID mapping
func (m *IDMapping) SetGame(legacyID int, newID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Games[legacyID] = newID
}

// GetGame retrieves a game UUID by legacy ID
func (m *IDMapping) GetGame(legacyID int) (uuid.UUID, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.Games[legacyID]
	return id, ok
}

// SetGameRound stores a game round ID mapping
func (m *IDMapping) SetGameRound(legacyID int, newID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.GameRounds[legacyID] = newID
}

// GetGameRound retrieves a game round UUID by legacy ID
func (m *IDMapping) GetGameRound(legacyID int) (uuid.UUID, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.GameRounds[legacyID]
	return id, ok
}

// SetField stores a field ID mapping
func (m *IDMapping) SetField(legacyID int, newID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Fields[legacyID] = newID
}

// GetField retrieves a field UUID by legacy ID
func (m *IDMapping) GetField(legacyID int) (uuid.UUID, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.Fields[legacyID]
	return id, ok
}

// SetDivision stores a division pool ID mapping
func (m *IDMapping) SetDivision(legacyID int, newID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Divisions[legacyID] = newID
}

// GetDivision retrieves a division pool UUID by legacy ID
func (m *IDMapping) GetDivision(legacyID int) (uuid.UUID, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.Divisions[legacyID]
	return id, ok
}

// SetLocation stores a location ID mapping
func (m *IDMapping) SetLocation(legacyID int, newID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Locations[legacyID] = newID
}

// GetLocation retrieves a location UUID by legacy ID
func (m *IDMapping) GetLocation(legacyID int) (uuid.UUID, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.Locations[legacyID]
	return id, ok
}

// SetCountry stores a country ID mapping
func (m *IDMapping) SetCountry(legacyID int, newID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Countries[legacyID] = newID
}

// GetCountry retrieves a country UUID by legacy ID
func (m *IDMapping) GetCountry(legacyID int) (uuid.UUID, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.Countries[legacyID]
	return id, ok
}

// SetContinent stores a continent ID mapping
func (m *IDMapping) SetContinent(legacyID int, newID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Continents[legacyID] = newID
}

// GetContinent retrieves a continent UUID by legacy ID
func (m *IDMapping) GetContinent(legacyID int) (uuid.UUID, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.Continents[legacyID]
	return id, ok
}

// SetDiscipline stores a discipline ID mapping
func (m *IDMapping) SetDiscipline(legacyID int, newID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Disciplines[legacyID] = newID
}

// GetDiscipline retrieves a discipline UUID by legacy ID
func (m *IDMapping) GetDiscipline(legacyID int) (uuid.UUID, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.Disciplines[legacyID]
	return id, ok
}

// SetEvent stores an event ID mapping
func (m *IDMapping) SetEvent(legacyID int, newID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events[legacyID] = newID
}

// GetEvent retrieves an event UUID by legacy ID
func (m *IDMapping) GetEvent(legacyID int) (uuid.UUID, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.Events[legacyID]
	return id, ok
}

// SetUser stores a user ID mapping
func (m *IDMapping) SetUser(legacyID int, newID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Users[legacyID] = newID
}

// GetUser retrieves a user UUID by legacy ID
func (m *IDMapping) GetUser(legacyID int) (uuid.UUID, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.Users[legacyID]
	return id, ok
}

// SaveToFile persists the ID mapping to a JSON file
func (m *IDMapping) SaveToFile(path string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal ID mapping: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write ID mapping file: %w", err)
	}

	return nil
}

// LoadFromFile loads an ID mapping from a JSON file
func LoadIDMapping(path string) (*IDMapping, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewIDMapping(), nil
		}
		return nil, fmt.Errorf("failed to read ID mapping file: %w", err)
	}

	mapping := NewIDMapping()
	if err := json.Unmarshal(data, mapping); err != nil {
		return nil, fmt.Errorf("failed to parse ID mapping: %w", err)
	}

	return mapping, nil
}

// Stats returns statistics about the ID mappings
func (m *IDMapping) Stats() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]int{
		"teams":       len(m.Teams),
		"players":     len(m.Players),
		"games":       len(m.Games),
		"game_rounds": len(m.GameRounds),
		"fields":      len(m.Fields),
		"divisions":   len(m.Divisions),
		"locations":   len(m.Locations),
		"countries":   len(m.Countries),
		"continents":  len(m.Continents),
		"disciplines": len(m.Disciplines),
		"events":      len(m.Events),
		"users":       len(m.Users),
	}
}
