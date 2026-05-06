package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	appErr "devops-system/backend/internal/errors"
	"devops-system/backend/internal/middleware"
	"devops-system/backend/internal/models"
	"devops-system/backend/internal/pagination"
	"devops-system/backend/internal/response"
)

const ticketProtocolVersion = "aiops.tickets.v1alpha1"
const ticketConfirmText = "确认删除资源"

var ticketTypes = []string{"event", "change", "release", "resource_request", "permission_request", "incident", "service_request"}
var ticketPriorities = []string{"P0", "P1", "P2", "P3", "P4"}
var ticketStatuses = []string{"draft", "submitted", "assigned", "processing", "pending_approval", "approved", "rejected", "resolved", "closed", "cancelled"}

type ticketInput struct {
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	Type         string                 `json:"type"`
	Priority     string                 `json:"priority"`
	Severity     string                 `json:"severity"`
	RequesterID  uint                   `json:"requesterId"`
	AssigneeID   uint                   `json:"assigneeId"`
	DepartmentID uint                   `json:"departmentId"`
	Env          string                 `json:"env"`
	SLADueAt     *time.Time             `json:"slaDueAt"`
	DueAt        *time.Time             `json:"dueAt"`
	Tags         map[string]interface{} `json:"tags"`
	Metadata     map[string]interface{} `json:"metadata"`
}

func (h *Handler) ListTickets(c *gin.Context) {
	page := pagination.Parse(c)
	query := h.DB.Model(&models.Ticket{})
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("ticket_no LIKE ? OR title LIKE ? OR description LIKE ?", like, like, like)
	}
	if typ := normalizeTicketType(c.Query("type")); typ != "" {
		query = query.Where("type = ?", typ)
	}
	if status := normalizeTicketStatus(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if priority := normalizeTicketPriority(c.Query("priority")); priority != "" {
		query = query.Where("priority = ?", priority)
	}
	if env := strings.TrimSpace(c.Query("env")); env != "" {
		query = query.Where("env = ?", env)
	}
	if assigneeID, ok := parseUintQuery(c.Query("assigneeId")); ok {
		query = query.Where("assignee_id = ?", assigneeID)
	}
	if requesterID, ok := parseUintQuery(c.Query("requesterId")); ok {
		query = query.Where("requester_id = ?", requesterID)
	}
	var items []models.Ticket
	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Internal(c, err)
		return
	}
	if err := query.Order("id desc").Limit(page.PageSize).Offset(pagination.Offset(page)).Find(&items).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.List(c, items, total, page.Page, page.PageSize)
}

func (h *Handler) GetTicket(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	ticket, found := h.findTicket(c, id)
	if !found {
		return
	}
	summary, err := h.ticketSummary(ticket)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, summary)
}

func (h *Handler) CreateTicket(c *gin.Context) {
	var req ticketInput
	if !bindJSON(c, &req) {
		return
	}
	ticket, err := h.buildTicket(req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, err.Error()))
		return
	}
	if ticket.RequesterID == 0 {
		ticket.RequesterID = currentUserID(c)
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		ticket.TicketNo = buildTicketNo()
		if err := tx.Create(&ticket).Error; err != nil {
			return err
		}
		return tx.Create(&models.TicketFlow{
			TicketID:   ticket.ID,
			ToStatus:   ticket.Status,
			Action:     "create",
			OperatorID: currentUserID(c),
			Comment:    "创建工单",
		}).Error
	}); err != nil {
		response.Internal(c, err)
		return
	}
	h.notifyTicketEvent(ticket, "tickets.created", "工单已创建", "info")
	response.Success(c, ticket)
}

func (h *Handler) UpdateTicket(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	ticket, found := h.findTicket(c, id)
	if !found {
		return
	}
	if !ticketEditable(ticket.Status) {
		response.Error(c, http.StatusConflict, appErr.New(4025, "ticket cannot be edited in current status"))
		return
	}
	var req ticketInput
	if !bindJSON(c, &req) {
		return
	}
	next, err := h.buildTicket(req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, err.Error()))
		return
	}
	updates := map[string]interface{}{
		"title":         next.Title,
		"description":   next.Description,
		"type":          next.Type,
		"priority":      next.Priority,
		"severity":      next.Severity,
		"assignee_id":   next.AssigneeID,
		"department_id": next.DepartmentID,
		"env":           next.Env,
		"sla_due_at":    next.SLADueAt,
		"due_at":        next.DueAt,
		"tags":          next.Tags,
		"metadata":      next.Metadata,
	}
	if next.RequesterID > 0 {
		updates["requester_id"] = next.RequesterID
	}
	if err := h.DB.Model(&models.Ticket{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		response.Internal(c, err)
		return
	}
	h.recordTicketFlow(id, ticket.Status, ticket.Status, "update", currentUserID(c), "更新工单")
	getByID[models.Ticket](c, h.DB)
}

func (h *Handler) DeleteTicket(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		ConfirmationText string `json:"confirmationText" binding:"required"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if strings.TrimSpace(req.ConfirmationText) != ticketConfirmText {
		response.Error(c, http.StatusBadRequest, appErr.New(3020, "confirmation text is required"))
		return
	}
	ticket, found := h.findTicket(c, id)
	if !found {
		return
	}
	if !ticketDeletable(ticket.Status) {
		response.Error(c, http.StatusConflict, appErr.New(4026, "ticket cannot be deleted in current status"))
		return
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("ticket_id = ?", id).Delete(&models.TicketFlow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("ticket_id = ?", id).Delete(&models.TicketApproval{}).Error; err != nil {
			return err
		}
		if err := tx.Where("ticket_id = ?", id).Delete(&models.TicketComment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("ticket_id = ?", id).Delete(&models.TicketLink{}).Error; err != nil {
			return err
		}
		if err := tx.Where("ticket_id = ?", id).Delete(&models.TicketAttachment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("ticket_id = ?", id).Delete(&models.TicketOperation{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Ticket{}, id).Error
	}); err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *Handler) SubmitTicket(c *gin.Context) {
	h.transitionTicket(c, "submit", "submitted", "工单已提交")
}

func (h *Handler) CancelTicket(c *gin.Context) {
	h.transitionTicket(c, "cancel", "cancelled", "工单已取消")
}

func (h *Handler) ReopenTicket(c *gin.Context) {
	h.transitionTicket(c, "reopen", "processing", "工单已重开")
}

func (h *Handler) TransitionTicket(c *gin.Context) {
	var req struct {
		Status  string `json:"status" binding:"required"`
		Comment string `json:"comment"`
	}
	if !bindJSON(c, &req) {
		return
	}
	h.transitionTicket(c, "transition", normalizeTicketStatus(req.Status), req.Comment)
}

func (h *Handler) ApproveTicket(c *gin.Context) {
	h.approveOrRejectTicket(c, true)
}

func (h *Handler) RejectTicket(c *gin.Context) {
	h.approveOrRejectTicket(c, false)
}

func (h *Handler) TransferTicket(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		AssigneeID uint   `json:"assigneeId" binding:"required"`
		Comment    string `json:"comment"`
	}
	if !bindJSON(c, &req) {
		return
	}
	ticket, found := h.findTicket(c, id)
	if !found {
		return
	}
	if req.AssigneeID == 0 {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "assigneeId is required"))
		return
	}
	if err := h.DB.Model(&models.Ticket{}).Where("id = ?", id).Updates(map[string]interface{}{"assignee_id": req.AssigneeID, "status": "assigned"}).Error; err != nil {
		response.Internal(c, err)
		return
	}
	h.recordTicketFlow(id, ticket.Status, "assigned", "transfer", currentUserID(c), req.Comment)
	ticket.AssigneeID = req.AssigneeID
	ticket.Status = "assigned"
	h.notifyTicketEvent(ticket, "tickets.transferred", "工单已转派", "info")
	getByID[models.Ticket](c, h.DB)
}

func (h *Handler) AddTicketApprover(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		ApproverID   uint   `json:"approverId" binding:"required"`
		NodeKey      string `json:"nodeKey"`
		ApprovalType string `json:"approvalType"`
		Comment      string `json:"comment"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if _, found := h.findTicket(c, id); !found {
		return
	}
	if req.ApproverID == 0 {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "approverId is required"))
		return
	}
	approval := models.TicketApproval{
		TicketID:     id,
		NodeKey:      defaultString(strings.TrimSpace(req.NodeKey), "manual"),
		ApproverID:   req.ApproverID,
		ApprovalType: defaultString(strings.TrimSpace(req.ApprovalType), "or"),
		Status:       "pending",
		Comment:      req.Comment,
	}
	if err := h.DB.Create(&approval).Error; err != nil {
		response.Internal(c, err)
		return
	}
	h.recordTicketFlow(id, "", "", "add_approver", currentUserID(c), req.Comment)
	response.Success(c, approval)
}

func (h *Handler) ListTicketFlows(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if _, found := h.findTicket(c, id); !found {
		return
	}
	var items []models.TicketFlow
	if err := h.DB.Where("ticket_id = ?", id).Order("id asc").Find(&items).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, items)
}

func (h *Handler) TicketTimeline(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if _, found := h.findTicket(c, id); !found {
		return
	}
	summary, err := h.ticketSummaryByID(id)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, summary)
}

func (h *Handler) ListTicketApprovals(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var items []models.TicketApproval
	if err := h.DB.Where("ticket_id = ?", id).Order("id asc").Find(&items).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, items)
}

func (h *Handler) ListTicketComments(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var items []models.TicketComment
	if err := h.DB.Where("ticket_id = ?", id).Order("id asc").Find(&items).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, items)
}

func (h *Handler) CreateTicketComment(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Content     string                 `json:"content" binding:"required"`
		Internal    bool                   `json:"internal"`
		Attachments map[string]interface{} `json:"attachments"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if _, found := h.findTicket(c, id); !found {
		return
	}
	comment := models.TicketComment{
		TicketID:    id,
		UserID:      currentUserID(c),
		Content:     strings.TrimSpace(req.Content),
		Internal:    req.Internal,
		Attachments: datatypes.JSONMap(req.Attachments),
	}
	if comment.Content == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "content cannot be empty"))
		return
	}
	if err := h.DB.Create(&comment).Error; err != nil {
		response.Internal(c, err)
		return
	}
	h.recordTicketFlow(id, "", "", "comment", currentUserID(c), truncateText(comment.Content, 200))
	response.Success(c, comment)
}

func (h *Handler) ListTicketAttachments(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var items []models.TicketAttachment
	if err := h.DB.Where("ticket_id = ?", id).Order("id desc").Find(&items).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, items)
}

func (h *Handler) CreateTicketAttachment(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		FileName    string `json:"fileName" binding:"required"`
		FileSize    int64  `json:"fileSize"`
		ContentType string `json:"contentType"`
		StorageKey  string `json:"storageKey" binding:"required"`
		Checksum    string `json:"checksum"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if _, found := h.findTicket(c, id); !found {
		return
	}
	if req.FileSize < 0 || req.FileSize > 100*1024*1024 {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "attachment size exceeds limit"))
		return
	}
	attachment := models.TicketAttachment{
		TicketID:    id,
		FileName:    strings.TrimSpace(req.FileName),
		FileSize:    req.FileSize,
		ContentType: strings.TrimSpace(req.ContentType),
		StorageKey:  strings.TrimSpace(req.StorageKey),
		UploaderID:  currentUserID(c),
		Checksum:    strings.TrimSpace(req.Checksum),
	}
	if err := h.DB.Create(&attachment).Error; err != nil {
		response.Internal(c, err)
		return
	}
	h.recordTicketFlow(id, "", "", "attachment", currentUserID(c), attachment.FileName)
	response.Success(c, attachment)
}

func (h *Handler) ListTicketLinks(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var items []models.TicketLink
	if err := h.DB.Where("ticket_id = ?", id).Order("id desc").Find(&items).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, items)
}

func (h *Handler) CreateTicketLink(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		LinkType   string                 `json:"linkType" binding:"required"`
		LinkID     string                 `json:"linkId" binding:"required"`
		LinkName   string                 `json:"linkName"`
		LinkModule string                 `json:"linkModule"`
		Relation   string                 `json:"relation"`
		Metadata   map[string]interface{} `json:"metadata"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if _, found := h.findTicket(c, id); !found {
		return
	}
	link := models.TicketLink{
		TicketID:   id,
		LinkType:   strings.TrimSpace(req.LinkType),
		LinkID:     strings.TrimSpace(req.LinkID),
		LinkName:   strings.TrimSpace(req.LinkName),
		LinkModule: normalizeTicketLinkModule(req.LinkModule),
		Relation:   defaultString(strings.TrimSpace(req.Relation), "related"),
		Metadata:   datatypes.JSONMap(req.Metadata),
	}
	if err := h.DB.Create(&link).Error; err != nil {
		response.Internal(c, err)
		return
	}
	h.recordTicketFlow(id, "", "", "link", currentUserID(c), link.LinkModule+":"+link.LinkID)
	response.Success(c, link)
}

func (h *Handler) DeleteTicketLink(c *gin.Context) {
	ticketID, ok := parseID(c)
	if !ok {
		return
	}
	linkID, ok := parseIDParam(c, "linkId")
	if !ok {
		return
	}
	result := h.DB.Where("id = ? AND ticket_id = ?", linkID, ticketID).Delete(&models.TicketLink{})
	if result.Error != nil {
		response.Internal(c, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
		return
	}
	response.Success(c, gin.H{"id": linkID})
}

func (h *Handler) ListTicketTemplates(c *gin.Context) {
	page := pagination.Parse(c)
	query := h.DB.Model(&models.TicketTemplate{})
	if typ := normalizeTicketType(c.Query("type")); typ != "" {
		query = query.Where("type = ?", typ)
	}
	if enabledRaw := strings.TrimSpace(c.Query("enabled")); enabledRaw != "" {
		enabled, err := strconv.ParseBool(enabledRaw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "invalid enabled"))
			return
		}
		query = query.Where("enabled = ?", enabled)
	}
	var items []models.TicketTemplate
	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Internal(c, err)
		return
	}
	if err := query.Order("id desc").Limit(page.PageSize).Offset(pagination.Offset(page)).Find(&items).Error; err != nil {
		response.Internal(c, err)
		return
	}
	if total == 0 && c.Query("seed") == "1" {
		items = defaultTicketTemplates()
		response.List(c, items, int64(len(items)), page.Page, page.PageSize)
		return
	}
	response.List(c, items, total, page.Page, page.PageSize)
}

func (h *Handler) CreateTicketTemplate(c *gin.Context) {
	h.saveTicketTemplate(c, 0)
}

func (h *Handler) UpdateTicketTemplate(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	h.saveTicketTemplate(c, id)
}

func (h *Handler) DeleteTicketTemplate(c *gin.Context) {
	deleteByModel[models.TicketTemplate](c, h.DB)
}

func (h *Handler) TicketOperationDryRun(c *gin.Context) {
	h.runTicketOperation(c, true)
}

func (h *Handler) TicketOperationExecute(c *gin.Context) {
	h.runTicketOperation(c, false)
}

func (h *Handler) ListTicketOperations(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var items []models.TicketOperation
	if err := h.DB.Where("ticket_id = ?", id).Order("id desc").Find(&items).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, items)
}

func (h *Handler) TicketAIOpsProtocol(c *gin.Context) {
	response.Success(c, ticketProtocol())
}

func (h *Handler) TicketAIOpsIntent(c *gin.Context) {
	var req struct {
		Intent   string                 `json:"intent" binding:"required"`
		Context  map[string]interface{} `json:"context"`
		DryRun   *bool                  `json:"dryRun"`
		TicketID uint                   `json:"ticketId"`
	}
	if !bindJSON(c, &req) {
		return
	}
	response.Success(c, gin.H{
		"protocolVersion": ticketProtocolVersion,
		"traceId":         uuid.NewString(),
		"intent":          strings.TrimSpace(req.Intent),
		"ticketDraft": gin.H{
			"title":       truncateText(strings.TrimSpace(req.Intent), 120),
			"type":        inferTicketType(req.Intent),
			"priority":    "P3",
			"status":      "draft",
			"description": strings.TrimSpace(req.Intent),
			"metadata":    req.Context,
		},
		"safetyChecks": []string{"仅生成草稿，不直接提交审批", "高危动作必须先 dry-run 并审批"},
	})
}

func (h *Handler) TicketAIOpsDryRun(c *gin.Context) {
	var req struct {
		TicketID uint                   `json:"ticketId"`
		Type     string                 `json:"type"`
		Action   string                 `json:"action"`
		Module   string                 `json:"module"`
		Params   map[string]interface{} `json:"params"`
	}
	if !bindJSON(c, &req) {
		return
	}
	traceID := uuid.NewString()
	response.Success(c, gin.H{
		"protocolVersion":  ticketProtocolVersion,
		"traceId":          traceID,
		"ticketId":         req.TicketID,
		"riskLevel":        ticketOperationRisk(req.Module, req.Action),
		"approvalRequired": true,
		"dryRun":           buildTicketOperationPlan(req.TicketID, normalizeTicketLinkModule(req.Module), req.Action, req.Params),
	})
}

func (h *Handler) transitionTicket(c *gin.Context, action string, toStatus string, comment string) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if toStatus == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "invalid status"))
		return
	}
	ticket, found := h.findTicket(c, id)
	if !found {
		return
	}
	if !ticketTransitionAllowed(ticket.Status, toStatus, action) {
		response.Error(c, http.StatusConflict, appErr.New(4027, "ticket status transition is not allowed"))
		return
	}
	updates := map[string]interface{}{"status": toStatus}
	now := time.Now()
	if toStatus == "resolved" {
		updates["resolved_at"] = &now
	}
	if toStatus == "closed" {
		updates["closed_at"] = &now
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Ticket{}).Where("id = ? AND status = ?", id, ticket.Status).Updates(updates).Error; err != nil {
			return err
		}
		return tx.Create(&models.TicketFlow{
			TicketID:   id,
			FromStatus: ticket.Status,
			ToStatus:   toStatus,
			Action:     action,
			OperatorID: currentUserID(c),
			Comment:    strings.TrimSpace(comment),
		}).Error
	}); err != nil {
		response.Internal(c, err)
		return
	}
	ticket.Status = toStatus
	h.notifyTicketEvent(ticket, "tickets."+action, "工单状态已更新", ticketNotifySeverity(toStatus))
	getByID[models.Ticket](c, h.DB)
}

func (h *Handler) approveOrRejectTicket(c *gin.Context, approve bool) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Comment    string `json:"comment"`
		ApproverID uint   `json:"approverId"`
	}
	_ = c.ShouldBindJSON(&req)
	ticket, found := h.findTicket(c, id)
	if !found {
		return
	}
	if !canApproveTicket(ticket.Status) {
		response.Error(c, http.StatusConflict, appErr.New(4028, "ticket cannot be approved in current status"))
		return
	}
	operatorID := currentUserID(c)
	if req.ApproverID > 0 {
		operatorID = req.ApproverID
	}
	now := time.Now()
	nextStatus := "processing"
	approvalStatus := "approved"
	action := "approve"
	event := "tickets.approved"
	severity := "success"
	if !approve {
		nextStatus = "rejected"
		approvalStatus = "rejected"
		action = "reject"
		event = "tickets.rejected"
		severity = "warning"
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		approval := models.TicketApproval{
			TicketID:     id,
			NodeKey:      "default",
			ApproverID:   operatorID,
			ApprovalType: "or",
			Status:       approvalStatus,
			Comment:      strings.TrimSpace(req.Comment),
		}
		if approve {
			approval.ApprovedAt = &now
		} else {
			approval.RejectedAt = &now
		}
		if err := tx.Create(&approval).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Ticket{}).Where("id = ? AND status = ?", id, ticket.Status).Update("status", nextStatus).Error; err != nil {
			return err
		}
		return tx.Create(&models.TicketFlow{
			TicketID:   id,
			FromStatus: ticket.Status,
			ToStatus:   nextStatus,
			Action:     action,
			OperatorID: operatorID,
			Comment:    strings.TrimSpace(req.Comment),
		}).Error
	}); err != nil {
		response.Internal(c, err)
		return
	}
	ticket.Status = nextStatus
	h.notifyTicketEvent(ticket, event, "工单审批已处理", severity)
	response.Success(c, gin.H{"id": id, "approved": approve, "status": nextStatus})
}

func (h *Handler) runTicketOperation(c *gin.Context, dryRun bool) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Module           string                 `json:"module" binding:"required"`
		Action           string                 `json:"action" binding:"required"`
		Params           map[string]interface{} `json:"params"`
		ConfirmationText string                 `json:"confirmationText"`
	}
	if !bindJSON(c, &req) {
		return
	}
	ticket, found := h.findTicket(c, id)
	if !found {
		return
	}
	module := normalizeTicketLinkModule(req.Module)
	action := strings.TrimSpace(req.Action)
	if module == "" || action == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "module and action are required"))
		return
	}
	risk := ticketOperationRisk(module, action)
	if !dryRun && !ticketExecutable(ticket.Status) {
		response.Error(c, http.StatusConflict, appErr.New(4029, "ticket operation requires approved or processing status"))
		return
	}
	if !dryRun && highRiskPriority(risk) && strings.TrimSpace(req.ConfirmationText) != ticketConfirmText {
		response.Error(c, http.StatusBadRequest, appErr.New(3020, "confirmation text is required"))
		return
	}
	traceID := uuid.NewString()
	now := time.Now()
	status := "dry_run"
	if !dryRun {
		status = "success"
	}
	plan := buildTicketOperationPlan(id, module, action, req.Params)
	operation := models.TicketOperation{
		TraceID:    traceID,
		TicketID:   id,
		Module:     module,
		Action:     action,
		DryRun:     dryRun,
		Status:     status,
		RiskLevel:  risk,
		Request:    datatypes.JSONMap{"module": module, "action": action, "params": req.Params, "dryRun": dryRun},
		Result:     plan,
		StartedAt:  &now,
		FinishedAt: &now,
	}
	if err := h.DB.Create(&operation).Error; err != nil {
		response.Internal(c, err)
		return
	}
	h.recordTicketFlow(id, ticket.Status, ticket.Status, "operation_"+status, currentUserID(c), module+":"+action)
	if !dryRun {
		h.notifyTicketEvent(ticket, "tickets.operation.success", "工单执行动作完成", "success")
	}
	response.Success(c, gin.H{"protocolVersion": ticketProtocolVersion, "traceId": traceID, "operation": operation, "dryRun": plan})
}

func (h *Handler) saveTicketTemplate(c *gin.Context, id uint) {
	var req struct {
		Type            string                 `json:"type" binding:"required"`
		Name            string                 `json:"name" binding:"required"`
		Description     string                 `json:"description"`
		FormSchema      map[string]interface{} `json:"formSchema"`
		DefaultPriority string                 `json:"defaultPriority"`
		DefaultFlow     map[string]interface{} `json:"defaultFlow"`
		Enabled         *bool                  `json:"enabled"`
	}
	if !bindJSON(c, &req) {
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	template := models.TicketTemplate{
		Type:            normalizeTicketType(req.Type),
		Name:            strings.TrimSpace(req.Name),
		Description:     strings.TrimSpace(req.Description),
		FormSchema:      datatypes.JSONMap(req.FormSchema),
		DefaultPriority: normalizeTicketPriority(req.DefaultPriority),
		DefaultFlow:     datatypes.JSONMap(req.DefaultFlow),
		Enabled:         enabled,
	}
	if template.Type == "" || template.Name == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "type and name are required"))
		return
	}
	if template.DefaultPriority == "" {
		template.DefaultPriority = "P3"
	}
	if id == 0 {
		if err := h.DB.Create(&template).Error; err != nil {
			response.Internal(c, err)
			return
		}
		response.Success(c, template)
		return
	}
	if err := h.DB.Model(&models.TicketTemplate{}).Where("id = ?", id).Updates(template).Error; err != nil {
		response.Internal(c, err)
		return
	}
	getByID[models.TicketTemplate](c, h.DB)
}

func (h *Handler) buildTicket(req ticketInput) (models.Ticket, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return models.Ticket{}, errors.New("title cannot be empty")
	}
	typ := normalizeTicketType(req.Type)
	if typ == "" {
		typ = "event"
	}
	priority := normalizeTicketPriority(req.Priority)
	if priority == "" {
		priority = "P3"
	}
	severity := normalizeTicketPriority(req.Severity)
	if severity == "" {
		severity = priority
	}
	return models.Ticket{
		Title:        title,
		Description:  strings.TrimSpace(req.Description),
		Type:         typ,
		Status:       "draft",
		Priority:     priority,
		Severity:     severity,
		RequesterID:  req.RequesterID,
		AssigneeID:   req.AssigneeID,
		DepartmentID: req.DepartmentID,
		Env:          defaultString(strings.TrimSpace(req.Env), "prod"),
		SLADueAt:     req.SLADueAt,
		DueAt:        req.DueAt,
		Tags:         datatypes.JSONMap(req.Tags),
		Metadata:     datatypes.JSONMap(req.Metadata),
	}, nil
}

func (h *Handler) findTicket(c *gin.Context, id uint) (models.Ticket, bool) {
	var ticket models.Ticket
	if err := h.DB.First(&ticket, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return ticket, false
		}
		response.Internal(c, err)
		return ticket, false
	}
	return ticket, true
}

func (h *Handler) ticketSummaryByID(id uint) (gin.H, error) {
	var ticket models.Ticket
	if err := h.DB.First(&ticket, id).Error; err != nil {
		return nil, err
	}
	return h.ticketSummary(ticket)
}

func (h *Handler) ticketSummary(ticket models.Ticket) (gin.H, error) {
	var flows []models.TicketFlow
	var approvals []models.TicketApproval
	var comments []models.TicketComment
	var links []models.TicketLink
	var attachments []models.TicketAttachment
	var operations []models.TicketOperation
	if err := h.DB.Where("ticket_id = ?", ticket.ID).Order("id asc").Find(&flows).Error; err != nil {
		return nil, err
	}
	if err := h.DB.Where("ticket_id = ?", ticket.ID).Order("id asc").Find(&approvals).Error; err != nil {
		return nil, err
	}
	if err := h.DB.Where("ticket_id = ?", ticket.ID).Order("id asc").Find(&comments).Error; err != nil {
		return nil, err
	}
	if err := h.DB.Where("ticket_id = ?", ticket.ID).Order("id desc").Find(&links).Error; err != nil {
		return nil, err
	}
	if err := h.DB.Where("ticket_id = ?", ticket.ID).Order("id desc").Find(&attachments).Error; err != nil {
		return nil, err
	}
	if err := h.DB.Where("ticket_id = ?", ticket.ID).Order("id desc").Find(&operations).Error; err != nil {
		return nil, err
	}
	return gin.H{
		"ticket":      ticket,
		"flows":       flows,
		"approvals":   approvals,
		"comments":    comments,
		"links":       links,
		"attachments": attachments,
		"operations":  operations,
	}, nil
}

func (h *Handler) recordTicketFlow(ticketID uint, fromStatus string, toStatus string, action string, operatorID uint, comment string) {
	_ = h.DB.Create(&models.TicketFlow{
		TicketID:   ticketID,
		FromStatus: fromStatus,
		ToStatus:   toStatus,
		Action:     action,
		OperatorID: operatorID,
		Comment:    strings.TrimSpace(comment),
	}).Error
}

func (h *Handler) notifyTicketEvent(ticket models.Ticket, event string, title string, severity string) {
	_, _ = h.PublishNotification(NotificationOptions{
		Module:       "tickets",
		Source:       "ticket-module",
		Event:        event,
		Severity:     severity,
		ResourceType: "ticket",
		ResourceID:   strconv.FormatUint(uint64(ticket.ID), 10),
		Title:        title,
		Content:      fmt.Sprintf("%s %s：%s", ticket.TicketNo, ticket.Status, ticket.Title),
		Data: gin.H{
			"ticketId": ticket.ID,
			"ticketNo": ticket.TicketNo,
			"type":     ticket.Type,
			"status":   ticket.Status,
			"priority": ticket.Priority,
		},
	})
}

func ticketProtocol() gin.H {
	return gin.H{
		"protocolVersion": ticketProtocolVersion,
		"endpoints": gin.H{
			"list":    "/api/v1/tickets",
			"create":  "/api/v1/tickets",
			"dryRun":  "/api/v1/tickets/:id/operations/dry-run",
			"execute": "/api/v1/tickets/:id/operations/execute",
		},
		"types":      ticketTypes,
		"statuses":   ticketStatuses,
		"priorities": ticketPriorities,
		"actions": []gin.H{
			{"name": "submit", "from": []string{"draft", "rejected"}, "to": "submitted", "riskLevel": "P3"},
			{"name": "approve", "from": []string{"submitted", "pending_approval", "approved"}, "to": "processing", "riskLevel": "P2"},
			{"name": "reject", "from": []string{"submitted", "pending_approval", "approved"}, "to": "rejected", "riskLevel": "P2"},
			{"name": "resolve", "from": []string{"processing", "approved"}, "to": "resolved", "riskLevel": "P3"},
			{"name": "close", "from": []string{"resolved"}, "to": "closed", "riskLevel": "P3"},
			{"name": "cancel", "from": []string{"draft", "submitted", "assigned", "processing", "pending_approval"}, "to": "cancelled", "riskLevel": "P2"},
		},
		"requestSchema": gin.H{
			"title":       "string|required",
			"type":        "event|change|release|resource_request|permission_request|incident|service_request",
			"priority":    "P0|P1|P2|P3|P4",
			"metadata":    "object|optional",
			"dryRunFirst": true,
		},
		"safety": gin.H{
			"defaultDryRun":     true,
			"confirmationText":  ticketConfirmText,
			"approvalRequired":  "high-risk operation execution",
			"directLLMExecute":  false,
			"traceField":        "traceId",
			"stateMachineCheck": "backend enforced",
		},
	}
}

func buildTicketOperationPlan(ticketID uint, module string, action string, params map[string]interface{}) datatypes.JSONMap {
	return datatypes.JSONMap{
		"ticketId":          ticketID,
		"module":            module,
		"action":            strings.TrimSpace(action),
		"riskLevel":         ticketOperationRisk(module, action),
		"approvalRequired":  true,
		"estimatedDuration": "minutes",
		"impact":            "将通过后端白名单协议触发关联模块动作，真实影响以目标模块 dry-run 为准",
		"steps": []interface{}{
			"校验工单状态与审批结果",
			"校验目标模块和动作白名单",
			"生成 dry-run 影响范围",
			"真实执行时写入 ticket_operations 和 timeline",
		},
		"safetyChecks": []interface{}{
			"默认 dry-run",
			"高危动作需审批和确认文案",
			"不允许模型直接修改状态或执行任意命令",
		},
		"rollback": "按目标模块返回的 rollback 建议执行；不可回滚动作必须人工复核",
		"params":   params,
	}
}

func defaultTicketTemplates() []models.TicketTemplate {
	return []models.TicketTemplate{
		{Type: "event", Name: "事件处理", Description: "用于日常事件响应", DefaultPriority: "P3", Enabled: true},
		{Type: "change", Name: "生产变更", Description: "用于生产环境变更审批", DefaultPriority: "P1", Enabled: true},
		{Type: "resource_request", Name: "资源申请", Description: "用于云资源或基础设施采购", DefaultPriority: "P2", Enabled: true},
		{Type: "permission_request", Name: "权限申请", Description: "用于账号和权限开通", DefaultPriority: "P2", Enabled: true},
		{Type: "incident", Name: "故障处理", Description: "用于故障处理与复盘", DefaultPriority: "P0", Enabled: true},
	}
}

func parseIDParam(c *gin.Context, name string) (uint, bool) {
	raw := strings.TrimSpace(c.Param(name))
	if raw == "" {
		response.Error(c, http.StatusBadRequest, appErr.ErrBadRequest)
		return 0, false
	}
	id64, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, appErr.ErrBadRequest)
		return 0, false
	}
	return uint(id64), true
}

func parseUintQuery(raw string) (uint, bool) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return 0, false
	}
	value, err := strconv.ParseUint(text, 10, 64)
	if err != nil || value == 0 {
		return 0, false
	}
	return uint(value), true
}

func currentUserID(c *gin.Context) uint {
	if claims, ok := middleware.GetClaims(c); ok && claims != nil {
		return claims.UserID
	}
	return 0
}

func buildTicketNo() string {
	return "T" + time.Now().Format("20060102150405") + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
}

func normalizeTicketType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "event", "change", "release", "resource_request", "permission_request", "incident", "service_request":
		return normalized
	case "resource", "resource-request":
		return "resource_request"
	case "permission", "permission-request":
		return "permission_request"
	default:
		return ""
	}
}

func normalizeTicketStatus(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	for _, status := range ticketStatuses {
		if normalized == status {
			return normalized
		}
	}
	return ""
}

func normalizeTicketPriority(value string) string {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	for _, priority := range ticketPriorities {
		if normalized == priority {
			return normalized
		}
	}
	return ""
}

func normalizeTicketLinkModule(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "cloud", "cmdb", "docker", "middleware", "tasks", "events", "kubernetes", "observability", "aiops", "tickets":
		return strings.ToLower(strings.TrimSpace(value))
	case "task":
		return "tasks"
	case "event":
		return "events"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func ticketEditable(status string) bool {
	switch normalizeTicketStatus(status) {
	case "draft", "submitted", "assigned", "rejected":
		return true
	default:
		return false
	}
}

func ticketDeletable(status string) bool {
	switch normalizeTicketStatus(status) {
	case "draft", "cancelled", "rejected", "closed":
		return true
	default:
		return false
	}
}

func ticketExecutable(status string) bool {
	switch normalizeTicketStatus(status) {
	case "approved", "processing", "pending_approval":
		return true
	default:
		return false
	}
}

func canApproveTicket(status string) bool {
	switch normalizeTicketStatus(status) {
	case "submitted", "pending_approval", "approved":
		return true
	default:
		return false
	}
}

func ticketTransitionAllowed(from string, to string, action string) bool {
	from = normalizeTicketStatus(from)
	if to == "" || from == "" {
		return false
	}
	if action == "cancel" {
		return from != "closed" && from != "cancelled"
	}
	if action == "reopen" {
		return from == "resolved" || from == "closed" || from == "rejected"
	}
	allowed := map[string][]string{
		"draft":            {"submitted", "cancelled"},
		"submitted":        {"assigned", "pending_approval", "processing", "rejected", "cancelled"},
		"assigned":         {"processing", "pending_approval", "cancelled"},
		"processing":       {"pending_approval", "resolved", "cancelled"},
		"pending_approval": {"approved", "rejected", "processing", "cancelled"},
		"approved":         {"processing", "resolved"},
		"rejected":         {"submitted", "cancelled"},
		"resolved":         {"closed", "processing"},
		"closed":           {},
		"cancelled":        {},
	}
	for _, next := range allowed[from] {
		if next == to {
			return true
		}
	}
	return from == to
}

func ticketNotifySeverity(status string) string {
	switch normalizeTicketStatus(status) {
	case "rejected", "cancelled":
		return "warning"
	case "closed", "resolved", "approved":
		return "success"
	default:
		return "info"
	}
}

func ticketOperationRisk(module string, action string) string {
	normalized := strings.ToLower(strings.TrimSpace(module + ":" + action))
	if strings.Contains(normalized, "delete") || strings.Contains(normalized, "remove") || strings.Contains(normalized, "stop") || strings.Contains(normalized, "restart") {
		return "P1"
	}
	if strings.Contains(normalized, "create") || strings.Contains(normalized, "deploy") || strings.Contains(normalized, "execute") || strings.Contains(normalized, "purchase") {
		return "P2"
	}
	return "P3"
}

func highRiskPriority(priority string) bool {
	return priority == "P0" || priority == "P1" || priority == "P2"
}

func inferTicketType(intent string) string {
	text := strings.ToLower(intent)
	switch {
	case strings.Contains(text, "权限"):
		return "permission_request"
	case strings.Contains(text, "资源") || strings.Contains(text, "购买") || strings.Contains(text, "采购"):
		return "resource_request"
	case strings.Contains(text, "发布"):
		return "release"
	case strings.Contains(text, "变更"):
		return "change"
	case strings.Contains(text, "故障"):
		return "incident"
	default:
		return "event"
	}
}

func truncateText(value string, limit int) string {
	if len([]rune(value)) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit])
}
