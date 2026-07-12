package realtime

import (
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/rozy/backend/internal/auth"
	"github.com/rozy/backend/internal/platform/httpx"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Handler struct {
	hub    *Hub
	tokens *auth.TokenService
}

func NewHandler(hub *Hub, tokens *auth.TokenService) *Handler {
	return &Handler{hub: hub, tokens: tokens}
}

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		httpx.Error(w, http.StatusUnauthorized, "missing token")
		return
	}

	claims, err := h.tokens.Parse(token)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	h.hub.ServeClient(claims.UserID, conn)
}
