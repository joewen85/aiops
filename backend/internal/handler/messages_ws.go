package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	appErr "devops-system/backend/internal/errors"
	"devops-system/backend/internal/models"
	"devops-system/backend/internal/response"
	"devops-system/backend/internal/ws"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *Handler) ListMessages(c *gin.Context) {
	listByModel[models.InAppMessage](c, h.DB)
}

func (h *Handler) CreateMessage(c *gin.Context) {
	var req models.InAppMessage
	if !bindJSON(c, &req) {
		return
	}
	if req.Channel == "" {
		req.Channel = "broadcast"
	}
	if err := h.DB.Create(&req).Error; err != nil {
		response.Internal(c, err)
		return
	}
	h.Hub.Publish(ws.Message{
		Channel: req.Channel,
		Target:  req.Target,
		Title:   req.Title,
		Content: req.Content,
	})
	response.Success(c, req)
}

func (h *Handler) MarkMessageRead(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.DB.Model(&models.InAppMessage{}).Where("id = ?", id).Update("read", true).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{"id": id, "read": true})
}

func (h *Handler) WebSocket(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		token = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}
	if token == "" {
		response.Error(c, http.StatusUnauthorized, appErr.New(1001, "missing ws token"))
		return
	}
	claims, err := h.JWT.Parse(token)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, appErr.New(1001, "invalid ws token"))
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := &ws.Client{
		Conn:    conn,
		User:    claims.Username,
		RoleSet: toSet(claims.Roles),
		DeptSet: toSet([]string{claims.DeptID}),
		Send:    make(chan ws.Message, 32),
	}
	h.Hub.Register(client)

	go func() {
		defer h.Hub.Unregister(client)
		for {
			if _, _, readErr := conn.ReadMessage(); readErr != nil {
				return
			}
		}
	}()

	for msg := range client.Send {
		if writeErr := conn.WriteJSON(msg); writeErr != nil {
			return
		}
	}
}

func toSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		out[value] = struct{}{}
	}
	return out
}
