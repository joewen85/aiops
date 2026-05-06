package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	appErr "devops-system/backend/internal/errors"
	"devops-system/backend/internal/middleware"
	"devops-system/backend/internal/models"
	"devops-system/backend/internal/pagination"
	"devops-system/backend/internal/response"
	"devops-system/backend/internal/ws"
)

const (
	messageChannelBroadcast  = "broadcast"
	messageChannelUser       = "user"
	messageChannelRole       = "role"
	messageChannelDepartment = "department"
	wsWriteWait              = 10 * time.Second
	wsPongWait               = 60 * time.Second
	wsPingPeriod             = (wsPongWait * 9) / 10
	wsMaxMessageBytes        = 1024
)

type messageResponse struct {
	models.InAppMessage
	ReadAt *time.Time `json:"readAt,omitempty"`
}

func (h *Handler) ListMessages(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok || claims == nil {
		response.Error(c, http.StatusUnauthorized, appErr.ErrUnauthorized)
		return
	}
	page := pagination.Parse(c)
	query := h.visibleMessagesQuery(c)
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where("title LIKE ? OR content LIKE ? OR trace_id LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if rawChannel := strings.TrimSpace(c.Query("channel")); rawChannel != "" {
		channel := normalizeMessageChannel(rawChannel)
		if channel == "" {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "unsupported message channel"))
			return
		}
		query = query.Where("channel = ?", channel)
	}

	readFilter := strings.ToLower(strings.TrimSpace(c.Query("read")))
	if readFilter == "true" || readFilter == "false" {
		readIDsQuery := h.DB.Model(&models.MessageReadReceipt{}).Select("message_id").Where("user_id = ?", claims.UserID)
		if readFilter == "true" {
			query = query.Where("id IN (?)", readIDsQuery)
		} else {
			query = query.Where("id NOT IN (?)", readIDsQuery)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Internal(c, err)
		return
	}
	var messages []models.InAppMessage
	if err := query.Order("id desc").Limit(page.PageSize).Offset(pagination.Offset(page)).Find(&messages).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.List(c, h.withReadState(messages, claims.UserID), total, page.Page, page.PageSize)
}

func (h *Handler) CreateMessage(c *gin.Context) {
	var req struct {
		Channel string                 `json:"channel"`
		Target  string                 `json:"target"`
		Title   string                 `json:"title"`
		Content string                 `json:"content"`
		Data    map[string]interface{} `json:"data"`
	}
	if !bindJSON(c, &req) {
		return
	}
	channel := normalizeMessageChannel(req.Channel)
	if channel == "" {
		channel = messageChannelBroadcast
	}
	target := strings.TrimSpace(req.Target)
	resolvedTarget, err := h.resolveMessageTarget(channel, target)
	if err != nil {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, err.Error()))
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "message content is required"))
		return
	}
	message := models.InAppMessage{
		TraceID: uuid.NewString(),
		Channel: channel,
		Target:  resolvedTarget,
		Title:   strings.TrimSpace(req.Title),
		Content: content,
		Data:    datatypes.JSONMap(req.Data),
	}
	if err := h.DB.Create(&message).Error; err != nil {
		response.Internal(c, err)
		return
	}
	if h.Hub != nil {
		h.Hub.Publish(ws.Message{
			TraceID: message.TraceID,
			Channel: message.Channel,
			Target:  message.Target,
			Title:   message.Title,
			Content: message.Content,
			Data:    message.Data,
		})
	}
	response.Success(c, message)
}

func (h *Handler) MarkMessageRead(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	claims, exists := middleware.GetClaims(c)
	if !exists || claims == nil {
		response.Error(c, http.StatusUnauthorized, appErr.ErrUnauthorized)
		return
	}
	var message models.InAppMessage
	if err := h.visibleMessagesQuery(c).Where("id = ?", id).First(&message).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	now := time.Now()
	receipt := models.MessageReadReceipt{
		MessageID: id,
		UserID:    claims.UserID,
		ReadAt:    now,
	}
	if err := h.DB.Where("message_id = ? AND user_id = ?", id, claims.UserID).FirstOrCreate(&receipt).Error; err != nil {
		response.Internal(c, err)
		return
	}
	if err := h.DB.Model(&models.InAppMessage{}).Where("id = ?", id).Update("read", true).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{"id": id, "traceId": message.TraceID, "read": true, "readAt": receipt.ReadAt})
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
	runtimeRoles, authorized := h.resolveRuntimeRoles(claims)
	if !authorized {
		response.Error(c, http.StatusUnauthorized, appErr.ErrUnauthorized)
		return
	}
	upgrader := websocket.Upgrader{
		CheckOrigin: h.checkWebSocketOrigin,
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := &ws.Client{
		Conn:    conn,
		User:    claims.Username,
		RoleSet: toSet(runtimeRoles),
		DeptSet: toSet([]string{claims.DeptID}),
		Send:    make(chan ws.Message, 32),
	}
	if h.Hub == nil {
		_ = conn.Close()
		return
	}
	h.Hub.Register(client)

	go func() {
		defer h.Hub.Unregister(client)
		conn.SetReadLimit(wsMaxMessageBytes)
		_ = conn.SetReadDeadline(time.Now().Add(wsPongWait))
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(wsPongWait))
		})
		for {
			if _, _, readErr := conn.ReadMessage(); readErr != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(wsPingPeriod)
	defer ticker.Stop()
	for {
		select {
		case msg, ok := <-client.Send:
			if !ok {
				return
			}
			if writeErr := writeWSJSON(conn, msg); writeErr != nil {
				_ = conn.Close()
				return
			}
		case <-ticker.C:
			if pingErr := writeWSPing(conn); pingErr != nil {
				_ = conn.Close()
				return
			}
		}
	}
}

func (h *Handler) visibleMessagesQuery(c *gin.Context) *gorm.DB {
	claims, ok := middleware.GetClaims(c)
	if !ok || claims == nil {
		return h.DB.Model(&models.InAppMessage{}).Where("1 = 0")
	}
	runtimeRoles, authorized := h.resolveRuntimeRoles(claims)
	if !authorized {
		return h.DB.Model(&models.InAppMessage{}).Where("1 = 0")
	}
	roleTargets := make([]string, 0, len(runtimeRoles))
	for _, role := range runtimeRoles {
		role = strings.TrimSpace(role)
		if role != "" {
			roleTargets = append(roleTargets, role)
		}
	}
	conditions := []string{
		"channel = ?",
		"(channel = ? AND target = ?)",
		"(channel = ? AND target = ?)",
	}
	args := []interface{}{
		messageChannelBroadcast,
		messageChannelUser, claims.Username,
		messageChannelUser, strconv.FormatUint(uint64(claims.UserID), 10),
	}
	for _, roleTarget := range roleTargets {
		conditions = append(conditions, "(channel = ? AND target = ?)")
		args = append(args, messageChannelRole, roleTarget)
	}
	if strings.TrimSpace(claims.DeptID) != "" {
		conditions = append(conditions, "(channel = ? AND target = ?)")
		args = append(args, messageChannelDepartment, claims.DeptID)
	}
	return h.DB.Model(&models.InAppMessage{}).Where(strings.Join(conditions, " OR "), args...)
}

func (h *Handler) withReadState(messages []models.InAppMessage, userID uint) []messageResponse {
	result := make([]messageResponse, 0, len(messages))
	if len(messages) == 0 || userID == 0 {
		for _, message := range messages {
			result = append(result, messageResponse{InAppMessage: message})
		}
		return result
	}
	messageIDs := make([]uint, 0, len(messages))
	for _, message := range messages {
		messageIDs = append(messageIDs, message.ID)
	}
	var receipts []models.MessageReadReceipt
	_ = h.DB.Where("user_id = ? AND message_id IN ?", userID, messageIDs).Find(&receipts).Error
	readAtByMessageID := make(map[uint]time.Time, len(receipts))
	for _, receipt := range receipts {
		readAtByMessageID[receipt.MessageID] = receipt.ReadAt
	}
	for _, message := range messages {
		item := messageResponse{InAppMessage: message}
		if readAt, ok := readAtByMessageID[message.ID]; ok {
			item.Read = true
			item.ReadAt = &readAt
		} else {
			item.Read = false
		}
		result = append(result, item)
	}
	return result
}

func (h *Handler) resolveMessageTarget(channel string, target string) (string, error) {
	switch channel {
	case messageChannelBroadcast:
		return "", nil
	case messageChannelUser:
		if target == "" {
			return "", errors.New("message target is required")
		}
		var user models.User
		query := h.DB.Model(&models.User{}).Where("username = ?", target)
		if id, err := strconv.ParseUint(target, 10, 64); err == nil {
			query = h.DB.Model(&models.User{}).Where("username = ? OR id = ?", target, uint(id))
		}
		if err := query.First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", errors.New("message target user not found")
			}
			return "", err
		}
		return user.Username, nil
	case messageChannelRole:
		if target == "" {
			return "", errors.New("message target is required")
		}
		var role models.Role
		if err := h.DB.Where("name = ?", target).First(&role).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", errors.New("message target role not found")
			}
			return "", err
		}
		return role.Name, nil
	case messageChannelDepartment:
		if target == "" {
			return "", errors.New("message target is required")
		}
		var department models.Department
		query := h.DB.Model(&models.Department{}).Where("name = ?", target)
		if id, err := strconv.ParseUint(target, 10, 64); err == nil {
			query = h.DB.Model(&models.Department{}).Where("id = ? OR name = ?", uint(id), target)
		}
		if err := query.First(&department).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", errors.New("message target department not found")
			}
			return "", err
		}
		return strconv.FormatUint(uint64(department.ID), 10), nil
	default:
		return "", errors.New("unsupported message channel")
	}
}

func normalizeMessageChannel(channel string) string {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case "", "broadcast", "all":
		return messageChannelBroadcast
	case "user":
		return messageChannelUser
	case "role":
		return messageChannelRole
	case "department", "dept":
		return messageChannelDepartment
	default:
		return ""
	}
}

func (h *Handler) checkWebSocketOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	allowed := strings.TrimSpace(h.Config.CORSAllowOrigins)
	if allowed == "" || allowed == "*" {
		return true
	}
	for _, item := range strings.Split(allowed, ",") {
		if strings.TrimSpace(item) == origin {
			return true
		}
	}
	return false
}

func writeWSJSON(conn *websocket.Conn, msg ws.Message) error {
	_ = conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
	return conn.WriteJSON(msg)
}

func writeWSPing(conn *websocket.Conn) error {
	_ = conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
	return conn.WriteMessage(websocket.PingMessage, nil)
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
