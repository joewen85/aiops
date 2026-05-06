package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"

	"devops-system/backend/internal/models"
	"devops-system/backend/internal/ws"
)

const messageAIOpsProtocolVersion = "aiops.messages.v1alpha1"

type NotificationOptions struct {
	TraceID      string
	Channel      string
	Target       string
	Title        string
	Content      string
	Module       string
	Source       string
	Event        string
	Severity     string
	ResourceType string
	ResourceID   string
	Data         map[string]interface{}
}

func (h *Handler) PublishNotification(options NotificationOptions) (models.InAppMessage, error) {
	channel := normalizeMessageChannel(options.Channel)
	if channel == "" {
		channel = messageChannelBroadcast
	}
	traceID := strings.TrimSpace(options.TraceID)
	if traceID == "" {
		traceID = uuid.NewString()
	}
	message := models.InAppMessage{
		TraceID:      traceID,
		Channel:      channel,
		Target:       strings.TrimSpace(options.Target),
		Title:        defaultString(strings.TrimSpace(options.Title), "平台通知"),
		Content:      strings.TrimSpace(options.Content),
		Module:       normalizeMessageModule(options.Module),
		Source:       defaultString(strings.TrimSpace(options.Source), "system"),
		Event:        strings.TrimSpace(options.Event),
		Severity:     normalizeMessageSeverity(options.Severity),
		ResourceType: strings.TrimSpace(options.ResourceType),
		ResourceID:   strings.TrimSpace(options.ResourceID),
		Data:         datatypes.JSONMap(enrichNotificationData(options)),
	}
	if message.Content == "" {
		message.Content = message.Title
	}
	if err := h.DB.Create(&message).Error; err != nil {
		return message, err
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
	return message, nil
}

func enrichNotificationData(options NotificationOptions) map[string]interface{} {
	data := map[string]interface{}{}
	for key, value := range options.Data {
		data[key] = value
	}
	if strings.TrimSpace(options.Module) != "" {
		data["module"] = normalizeMessageModule(options.Module)
	}
	if strings.TrimSpace(options.Source) != "" {
		data["source"] = strings.TrimSpace(options.Source)
	}
	if strings.TrimSpace(options.Event) != "" {
		data["event"] = strings.TrimSpace(options.Event)
	}
	if strings.TrimSpace(options.Severity) != "" {
		data["severity"] = normalizeMessageSeverity(options.Severity)
	}
	if strings.TrimSpace(options.ResourceType) != "" {
		data["resourceType"] = strings.TrimSpace(options.ResourceType)
	}
	if strings.TrimSpace(options.ResourceID) != "" {
		data["resourceId"] = strings.TrimSpace(options.ResourceID)
	}
	return data
}

func normalizeMessageModule(module string) string {
	switch strings.ToLower(strings.TrimSpace(module)) {
	case "task", "tasks":
		return "tasks"
	case "cloud", "multicloud":
		return "cloud"
	case "cmdb":
		return "cmdb"
	case "docker":
		return "docker"
	case "middleware":
		return "middleware"
	case "ticket", "tickets":
		return "tickets"
	case "event", "events":
		return "events"
	case "kubernetes", "k8s":
		return "kubernetes"
	case "observability":
		return "observability"
	case "aiops":
		return "aiops"
	case "system", "":
		return "system"
	default:
		return strings.ToLower(strings.TrimSpace(module))
	}
}

func normalizeMessageSeverity(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "debug":
		return "debug"
	case "info", "":
		return "info"
	case "success", "ok":
		return "success"
	case "warning", "warn":
		return "warning"
	case "error", "failed", "failure":
		return "error"
	case "critical", "fatal":
		return "critical"
	default:
		return "info"
	}
}

func messageAIOpsProtocol() gin.H {
	return gin.H{
		"protocolVersion": messageAIOpsProtocolVersion,
		"readEndpoint":    "/api/v1/messages/aiops/context",
		"supportedModules": []string{
			"tasks", "cloud", "cmdb", "docker", "middleware", "tickets", "events", "kubernetes", "observability", "aiops", "system",
		},
		"supportedSeverity": []string{"debug", "info", "success", "warning", "error", "critical"},
		"querySchema": gin.H{
			"module":     "string|optional",
			"severity":   "string|optional",
			"unreadOnly": "boolean|default false",
			"limit":      "number|default 20|max 50",
		},
		"messageFields": []string{"traceId", "module", "source", "event", "severity", "resourceType", "resourceId", "title", "content", "data", "readAt"},
		"safety": gin.H{
			"visibleScope": "current user visible messages only",
			"writeAccess":  "read endpoint does not mutate read receipt",
		},
	}
}
