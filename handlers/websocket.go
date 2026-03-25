package handlers

import (
	"net/http"

	"github.com/clementhaon/sandbox-api-go/auth"
	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/websocket"

	"github.com/google/uuid"
	ws "github.com/gorilla/websocket"
)

var upgrader = ws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		// TODO: Restrict in production
		return true
	},
}

// HandleWebSocket handles WebSocket connections at /ws
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get token from query parameter
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	// Validate JWT token
	claims, err := auth.ValidateToken(token)
	if err != nil {
		logger.Warn("WebSocket: Invalid token", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("WebSocket: Failed to upgrade connection", err)
		return
	}

	// Create client
	client := &websocket.Client{
		ID:     uuid.New().String(),
		UserID: claims.UserID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	// Register client
	websocket.GlobalManager.Register(client)

	logger.Info("WebSocket: Client connected", map[string]interface{}{
		"client_id": client.ID,
		"user_id":   client.UserID,
	})

	// Start goroutines for reading and writing
	go client.WritePump()
	go client.ReadPump(websocket.GlobalManager)
}
