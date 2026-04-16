package handler

import (
	"github.com/gin-gonic/gin"

	"devops-system/backend/internal/models"
	"devops-system/backend/internal/pagination"
	"devops-system/backend/internal/response"
)

func (h *Handler) ListAuditLogs(c *gin.Context) {
	page := pagination.Parse(c)
	actor := c.Query("actor")
	action := c.Query("action")
	resource := c.Query("resource")

	var (
		items []models.AuditLog
		total int64
	)

	query := h.DB.Model(&models.AuditLog{}).Order("id desc")
	if actor != "" {
		query = query.Where("actor = ?", actor)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if resource != "" {
		query = query.Where("resource = ?", resource)
	}

	if err := query.Count(&total).Error; err != nil {
		response.Internal(c, err)
		return
	}
	if err := query.Limit(page.PageSize).Offset(pagination.Offset(page)).Find(&items).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.List(c, items, total, page.Page, page.PageSize)
}
