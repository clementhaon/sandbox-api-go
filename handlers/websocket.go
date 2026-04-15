package handlers

import (
	"net/http"
	"net/url"
	"os"
	"strings"

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
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // No Origin header = same-origin or non-browser client
		}

		allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
		if allowedOrigins == "" {
			// Default: only allow same-origin
			u, err := url.Parse(origin)
			if err != nil {
				return false
			}
			return u.Host == r.Host
		}

		for _, allowed := range strings.Split(allowedOrigins, ",") {
			if strings.TrimSpace(allowed) == origin {
				return true
			}
		}
		return false
	},
}

type WebSocketHandler struct {
	wsManager  *websocket.Manager
	jwtManager *auth.JWTManager
}

func NewWebSocketHandler(wsManager *websocket.Manager, jwtManager *auth.JWTManager) *WebSocketHandler {
	return &WebSocketHandler{wsManager: wsManager, jwtManager: jwtManager}
}

func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	claims, err := h.jwtManager.ValidateToken(token)
	if err != nil {
		logger.Warn("WebSocket: Invalid token", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("WebSocket: Failed to upgrade connection", err)
		return
	}

	client := &websocket.Client{
		ID:     uuid.New().String(),
		UserID: claims.UserID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	h.wsManager.Register(client)

	logger.Info("WebSocket: Client connected", map[string]interface{}{
		"client_id": client.ID,
		"user_id":   client.UserID,
	})

	go client.WritePump()
	go client.ReadPump(h.wsManager)
}
