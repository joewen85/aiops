package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops-system/backend/internal/models"
)

const maxAuditPayloadBytes = 4096

var sensitiveAuditKeys = map[string]struct{}{
	"password":      {},
	"passwordhash":  {},
	"token":         {},
	"accesstoken":   {},
	"refreshtoken":  {},
	"authorization": {},
	"cookie":        {},
	"accesskey":     {},
	"secret":        {},
	"apikey":        {},
	"secretkey":     {},
	"privatekey":    {},
}

func AuditLogger(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if database == nil {
			c.Next()
			return
		}

		var body string
		if c.Request.Body != nil {
			raw, err := io.ReadAll(c.Request.Body)
			if err == nil {
				body = string(raw)
				c.Request.Body = io.NopCloser(bytes.NewBuffer(raw))
			}
		}

		c.Next()

		if !needAudit(c.Request.Method) || c.Writer.Status() >= 400 {
			return
		}

		claims, _ := GetClaims(c)
		actor := "anonymous"
		if claims != nil {
			actor = claims.Username
		}

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		body = sanitizeAuditPayload(path, body)

		log := models.AuditLog{
			Actor:      actor,
			Action:     strings.ToLower(c.Request.Method),
			Resource:   inferResource(path),
			ResourceID: c.Param("id"),
			Path:       path,
			Method:     c.Request.Method,
			Payload:    body,
		}
		_ = database.Create(&log).Error
	}
}

func needAudit(method string) bool {
	switch method {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

func inferResource(path string) string {
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown"
}

func sanitizeAuditPayload(path string, payload string) string {
	if payload == "" {
		return payload
	}
	if strings.HasSuffix(path, "/auth/login") {
		return `{"masked":true}`
	}
	if len(payload) > maxAuditPayloadBytes {
		payload = payload[:maxAuditPayloadBytes]
	}

	var parsed interface{}
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		return payload
	}
	sanitizeAuditValue(&parsed)
	encoded, err := json.Marshal(parsed)
	if err != nil {
		return payload
	}
	return string(encoded)
}

func sanitizeAuditValue(value *interface{}) {
	switch current := (*value).(type) {
	case map[string]interface{}:
		for key, child := range current {
			if isSensitiveAuditKey(key) {
				current[key] = "***"
				continue
			}
			childCopy := child
			sanitizeAuditValue(&childCopy)
			current[key] = childCopy
		}
	case []interface{}:
		for index := range current {
			childCopy := current[index]
			sanitizeAuditValue(&childCopy)
			current[index] = childCopy
		}
	}
}

func isSensitiveAuditKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(key), "_", ""), "-", ""))
	if normalized == "" {
		return false
	}
	if _, ok := sensitiveAuditKeys[normalized]; ok {
		return true
	}
	return strings.Contains(normalized, "password") || strings.Contains(normalized, "secret") || strings.Contains(normalized, "accesskey")
}
