package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type Client struct {
	ID      uuid.UUID
	Channel chan Event
	GameID  uuid.UUID
}

type Broker struct {
	mu sync.RWMutex

	// Map of gameID -> list of client channels
	clients map[uuid.UUID]map[uuid.UUID]chan Event

	// Channel for new clients
	newClients chan Client

	// Channel for closed clients
	closedClients chan Client

	// Channel for events
	events chan struct {
		GameID uuid.UUID
		Event  Event
	}

	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

func NewBroker() *Broker {
	ctx, cancel := context.WithCancel(context.Background())

	broker := &Broker{
		clients:       make(map[uuid.UUID]map[uuid.UUID]chan Event),
		newClients:    make(chan Client, 10),
		closedClients: make(chan Client, 10),
		events: make(chan struct {
			GameID uuid.UUID
			Event  Event
		}, 100),
		ctx:    ctx,
		cancel: cancel,
	}

	go broker.listen()

	return broker
}

func (b *Broker) listen() {
	for {
		select {
		case <-b.ctx.Done():
			return

		case client := <-b.newClients:
			b.mu.Lock()
			if b.clients[client.GameID] == nil {
				b.clients[client.GameID] = make(map[uuid.UUID]chan Event)
			}
			b.clients[client.GameID][client.ID] = client.Channel
			b.mu.Unlock()

			// Send connection established event
			client.Channel <- Event{
				Type: "connected",
				Data: map[string]interface{}{
					"game_id":   client.GameID,
					"client_id": client.ID,
					"timestamp": time.Now(),
				},
			}

		case client := <-b.closedClients:
			b.mu.Lock()
			if gameClients, exists := b.clients[client.GameID]; exists {
				delete(gameClients, client.ID)
				if len(gameClients) == 0 {
					delete(b.clients, client.GameID)
				}
			}
			close(client.Channel)
			b.mu.Unlock()

		case event := <-b.events:
			b.mu.RLock()
			gameClients := b.clients[event.GameID]
			b.mu.RUnlock()

			for _, clientChan := range gameClients {
				select {
				case clientChan <- event.Event:
				case <-time.After(1 * time.Second):
					// Client is slow, skip this event
				}
			}
		}
	}
}

func (b *Broker) Subscribe(gameID uuid.UUID) (uuid.UUID, <-chan Event) {
	clientID := uuid.New()
	clientChan := make(chan Event, 10)

	client := Client{
		ID:      clientID,
		Channel: clientChan,
		GameID:  gameID,
	}

	b.newClients <- client

	return clientID, clientChan
}

func (b *Broker) Unsubscribe(gameID, clientID uuid.UUID) {
	b.mu.RLock()
	gameClients, exists := b.clients[gameID]
	b.mu.RUnlock()

	if !exists {
		return
	}

	b.mu.RLock()
	clientChan, exists := gameClients[clientID]
	b.mu.RUnlock()

	if !exists {
		return
	}

	b.closedClients <- Client{
		ID:      clientID,
		Channel: clientChan,
		GameID:  gameID,
	}
}

func (b *Broker) Broadcast(gameID uuid.UUID, eventType string, data interface{}) {
	event := Event{
		Type: eventType,
		Data: data,
	}

	select {
	case b.events <- struct {
		GameID uuid.UUID
		Event  Event
	}{
		GameID: gameID,
		Event:  event,
	}:
	case <-time.After(100 * time.Millisecond):
		// Event queue is full, drop event
	}
}

func (b *Broker) GetClientCount(gameID uuid.UUID) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if gameClients, exists := b.clients[gameID]; exists {
		return len(gameClients)
	}
	return 0
}

func (b *Broker) Shutdown() {
	b.cancel()

	b.mu.Lock()
	defer b.mu.Unlock()

	for _, gameClients := range b.clients {
		for _, clientChan := range gameClients {
			close(clientChan)
		}
	}

	b.clients = make(map[uuid.UUID]map[uuid.UUID]chan Event)
}

// Helper function to format SSE message
func FormatSSE(event Event) string {
	data, _ := json.Marshal(event.Data)
	return fmt.Sprintf("event: %s\ndata: %s\n\n", event.Type, string(data))
}

// Event type constants
const (
	EventGameStarted      = "game_started"
	EventGameFinished     = "game_finished"
	EventGameEnded        = "game_ended"
	EventGoalScored       = "goal_scored"
	EventAssistRecorded   = "assist_recorded"
	EventStoppageRecorded = "stoppage_recorded"
	EventScoreUpdated     = "score_updated"
	EventConnected        = "connected"
	EventHeartbeat        = "heartbeat"
)
