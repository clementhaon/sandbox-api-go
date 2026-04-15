package websocket

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message represents a WebSocket message
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Client represents a WebSocket client connection
type Client struct {
	ID        string
	UserID    int
	Conn      *websocket.Conn
	Send      chan []byte
	ReadLimit int64 // 0 means use default (4096)
}

// Manager manages WebSocket connections
type Manager struct {
	clients    map[int][]*Client // userID -> connections
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewManager creates a new WebSocket manager and starts its run loop.
func NewManager() *Manager {
	m := &Manager{
		clients:    make(map[int][]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
	go m.Run()
	return m
}

// Run starts the manager's main loop
func (m *Manager) Run() {
	for {
		select {
		case client := <-m.register:
			m.mu.Lock()
			m.clients[client.UserID] = append(m.clients[client.UserID], client)
			m.mu.Unlock()

		case client := <-m.unregister:
			m.mu.Lock()
			if clients, ok := m.clients[client.UserID]; ok {
				for i, c := range clients {
					if c.ID == client.ID {
						m.clients[client.UserID] = append(clients[:i], clients[i+1:]...)
						close(client.Send)
						break
					}
				}
				if len(m.clients[client.UserID]) == 0 {
					delete(m.clients, client.UserID)
				}
			}
			m.mu.Unlock()
		}
	}
}

// Register adds a client to the manager
func (m *Manager) Register(client *Client) {
	m.register <- client
}

// Unregister removes a client from the manager
func (m *Manager) Unregister(client *Client) {
	m.unregister <- client
}

// SendToUser sends a message to all connections of a specific user
func (m *Manager) SendToUser(userID int, message *Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	m.mu.RLock()
	clients := m.clients[userID]
	m.mu.RUnlock()

	for _, client := range clients {
		select {
		case client.Send <- data:
		default:
			slog.Warn("WebSocket: message dropped, client buffer full",
				"client_id", client.ID,
				"user_id", client.UserID,
			)
		}
	}

	return nil
}

// Broadcast sends a message to all connected clients
func (m *Manager) Broadcast(message *Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, clients := range m.clients {
		for _, client := range clients {
			select {
			case client.Send <- data:
			default:
				slog.Warn("WebSocket: broadcast message dropped, client buffer full",
					"client_id", client.ID,
					"user_id", client.UserID,
				)
			}
		}
	}

	return nil
}

// GetConnectedUsers returns the number of connected users
func (m *Manager) GetConnectedUsers() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}

// GetTotalConnections returns the total number of connections
func (m *Manager) GetTotalConnections() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := 0
	for _, clients := range m.clients {
		total += len(clients)
	}
	return total
}

// WritePump pumps messages from the hub to the websocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ReadPump pumps messages from the websocket connection to the hub
func (c *Client) ReadPump(manager *Manager) {
	defer func() {
		manager.Unregister(c)
		c.Conn.Close()
	}()

	readLimit := c.ReadLimit
	if readLimit <= 0 {
		readLimit = 4096
	}
	c.Conn.SetReadLimit(readLimit)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Warn("WebSocket: unexpected close error",
					"client_id", c.ID,
					"user_id", c.UserID,
					"error", err.Error(),
				)
			}
			break
		}

		// Handle incoming messages
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		// Handle ping message
		if msg.Type == "ping" {
			response := Message{
				Type:    "pong",
				Payload: map[string]string{"status": "ok"},
			}
			data, _ := json.Marshal(response)
			c.Send <- data
		}
	}
}
