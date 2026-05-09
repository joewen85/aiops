package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"devops-system/backend/internal/models"
)

func TestTicketManagementIntegration(t *testing.T) {
	router, database, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	createRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/tickets", adminToken, map[string]any{
		"title":       "生产变更申请",
		"description": "升级核心服务版本",
		"type":        "change",
		"priority":    "P1",
		"severity":    "P1",
		"env":         "prod",
		"tags":        map[string]any{"app": "core"},
		"metadata":    map[string]any{"rollback": "restore previous image"},
	})
	createResp := assertOKResponse(t, createRec)
	var created models.Ticket
	if err := json.Unmarshal(createResp.Data, &created); err != nil {
		t.Fatalf("unmarshal created ticket failed: %v", err)
	}
	if created.ID == 0 || created.TicketNo == "" || created.Status != "draft" {
		t.Fatalf("unexpected created ticket: %+v", created)
	}

	listRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/tickets?page=1&pageSize=10&keyword=生产&type=change&priority=P1", adminToken, nil)
	listResp := assertOKResponse(t, listRec)
	var listData listPayload[models.Ticket]
	if err := json.Unmarshal(listResp.Data, &listData); err != nil {
		t.Fatalf("unmarshal ticket list failed: %v", err)
	}
	if listData.Total != 1 || len(listData.List) != 1 {
		t.Fatalf("expected one ticket, got total=%d len=%d", listData.Total, len(listData.List))
	}

	deleteRunningRec := sendJSONRequest(t, router, http.MethodDelete, fmt.Sprintf("/api/v1/tickets/%d", created.ID), adminToken, map[string]any{
		"confirmationText": "确认删除资源",
	})
	assertOKResponse(t, deleteRunningRec)

	createRec = sendJSONRequest(t, router, http.MethodPost, "/api/v1/tickets", adminToken, map[string]any{
		"title":      "云资源申请",
		"type":       "resource_request",
		"priority":   "P2",
		"env":        "prod",
		"assigneeId": 1,
	})
	createResp = assertOKResponse(t, createRec)
	if err := json.Unmarshal(createResp.Data, &created); err != nil {
		t.Fatalf("unmarshal second ticket failed: %v", err)
	}

	submitRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/submit", created.ID), adminToken, nil)
	assertOKResponse(t, submitRec)

	bypassTransitionRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/transition", created.ID), adminToken, map[string]any{
		"status":  "processing",
		"comment": "尝试绕过审批",
	})
	assertErrorResponse(t, bypassTransitionRec, http.StatusConflict, 4037, "transition to processing requires approval record")

	deleteSubmittedRec := sendJSONRequest(t, router, http.MethodDelete, fmt.Sprintf("/api/v1/tickets/%d", created.ID), adminToken, map[string]any{
		"confirmationText": "确认删除资源",
	})
	assertErrorResponse(t, deleteSubmittedRec, http.StatusConflict, 4026, "ticket cannot be deleted in current status")

	approveRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/approve", created.ID), adminToken, map[string]any{
		"comment": "审批通过",
	})
	assertOKResponse(t, approveRec)

	executeWithoutDryRunRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/operations/execute", created.ID), adminToken, map[string]any{
		"module":           "cloud",
		"action":           "delete_instance",
		"confirmationText": "确认删除资源",
	})
	assertErrorResponse(t, executeWithoutDryRunRec, http.StatusConflict, 4033, "execute requires matching dry-run first")

	dryRunRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/operations/dry-run", created.ID), adminToken, map[string]any{
		"module": "cloud",
		"action": "create_instance",
	})
	dryRunResp := assertOKResponse(t, dryRunRec)
	var dryRunData struct {
		TraceID string `json:"traceId"`
		DryRun  struct {
			ApprovalRequired bool `json:"approvalRequired"`
		} `json:"dryRun"`
	}
	if err := json.Unmarshal(dryRunResp.Data, &dryRunData); err != nil {
		t.Fatalf("unmarshal dry-run failed: %v", err)
	}
	if dryRunData.TraceID == "" || !dryRunData.DryRun.ApprovalRequired {
		t.Fatalf("unexpected dry-run data: %+v", dryRunData)
	}

	mismatchExecuteRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/operations/execute", created.ID), adminToken, map[string]any{
		"module":           "cloud",
		"action":           "delete_instance",
		"confirmationText": "确认删除资源",
	})
	assertErrorResponse(t, mismatchExecuteRec, http.StatusConflict, 4033, "execute requires matching dry-run first")

	invalidModuleDryRunRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/operations/dry-run", created.ID), adminToken, map[string]any{
		"module": "unknown",
		"action": "create_instance",
	})
	assertErrorResponse(t, invalidModuleDryRunRec, http.StatusBadRequest, 3001, "unsupported operation module")

	invalidActionDryRunRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/operations/dry-run", created.ID), adminToken, map[string]any{
		"module": "cloud",
		"action": "delete instance;drop",
	})
	assertErrorResponse(t, invalidActionDryRunRec, http.StatusBadRequest, 3001, "invalid operation action")

	missingConfirmRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/operations/execute", created.ID), adminToken, map[string]any{
		"module": "cloud",
		"action": "delete_instance",
	})
	assertErrorResponse(t, missingConfirmRec, http.StatusBadRequest, 3020, "confirmation text is required")

	deleteDryRunRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/operations/dry-run", created.ID), adminToken, map[string]any{
		"module": "cloud",
		"action": "delete_instance",
	})
	assertOKResponse(t, deleteDryRunRec)

	executeRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/operations/execute", created.ID), adminToken, map[string]any{
		"module":           "cloud",
		"action":           "delete_instance",
		"confirmationText": "确认删除资源",
	})
	assertOKResponse(t, executeRec)

	failedOperation := models.TicketOperation{
		TraceID:      "failed-trace",
		TicketID:     created.ID,
		Module:       "cloud",
		Action:       "delete_instance",
		DryRun:       false,
		Status:       "failed",
		RiskLevel:    "P1",
		Request:      map[string]any{"module": "cloud", "action": "delete_instance", "params": map[string]any{"instanceId": "i-test"}, "dryRun": false},
		Result:       map[string]any{"error": "provider timeout"},
		ErrorMessage: "provider timeout",
	}
	if err := database.Create(&failedOperation).Error; err != nil {
		t.Fatalf("create failed operation failed: %v", err)
	}

	retryRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/operations/%d/retry", created.ID, failedOperation.ID), adminToken, map[string]any{
		"confirmationText": "确认删除资源",
	})
	retryResp := assertOKResponse(t, retryRec)
	var retryData struct {
		Operation models.TicketOperation `json:"operation"`
	}
	if err := json.Unmarshal(retryResp.Data, &retryData); err != nil {
		t.Fatalf("unmarshal retry operation failed: %v", err)
	}
	if retryData.Operation.ID == 0 || retryData.Operation.Status != "success" || retryData.Operation.Request["retryOfOperationId"] == nil {
		t.Fatalf("unexpected retry operation: %+v", retryData.Operation)
	}

	retrySuccessRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/operations/%d/retry", created.ID, retryData.Operation.ID), adminToken, map[string]any{
		"confirmationText": "确认删除资源",
	})
	assertErrorResponse(t, retrySuccessRec, http.StatusConflict, 4041, "only failed execution operation can be retried")

	commentRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/comments", created.ID), adminToken, map[string]any{
		"content": "已完成资源申请 dry-run 和审批",
	})
	assertOKResponse(t, commentRec)

	invalidAttachmentRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/attachments", created.ID), adminToken, map[string]any{
		"fileName":    "report.json",
		"fileSize":    128,
		"contentType": "application/json",
		"storageKey":  "https://example.com/public/report.json",
		"checksum":    strings.Repeat("a", 64),
	})
	assertErrorResponse(t, invalidAttachmentRec, http.StatusBadRequest, 3001, "attachment storageKey cannot be public url")

	missingChecksumAttachmentRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/attachments", created.ID), adminToken, map[string]any{
		"fileName":    "report.json",
		"fileSize":    128,
		"contentType": "application/json",
		"storageKey":  "tickets/reports/report.json",
	})
	assertErrorResponse(t, missingChecksumAttachmentRec, http.StatusBadRequest, 3001, "attachment checksum must be sha256 hex")

	attachmentRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/attachments", created.ID), adminToken, map[string]any{
		"fileName":    "report.json",
		"fileSize":    128,
		"contentType": "application/json; charset=utf-8",
		"storageKey":  "tickets/reports/report.json",
		"checksum":    strings.Repeat("b", 64),
	})
	assertOKResponse(t, attachmentRec)

	linkRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/links", created.ID), adminToken, map[string]any{
		"linkModule": "cloud",
		"linkType":   "asset",
		"linkId":     "i-test",
		"linkName":   "测试资源",
	})
	assertOKResponse(t, linkRec)

	timelineRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/timeline", created.ID), adminToken, nil)
	timelineResp := assertOKResponse(t, timelineRec)
	var summary struct {
		Ticket     models.Ticket            `json:"ticket"`
		Flows      []models.TicketFlow      `json:"flows"`
		Comments   []models.TicketComment   `json:"comments"`
		Links      []models.TicketLink      `json:"links"`
		Operations []models.TicketOperation `json:"operations"`
	}
	if err := json.Unmarshal(timelineResp.Data, &summary); err != nil {
		t.Fatalf("unmarshal timeline failed: %v", err)
	}
	if summary.Ticket.ID != created.ID || len(summary.Flows) == 0 || len(summary.Comments) != 1 || len(summary.Links) != 1 || len(summary.Operations) < 2 {
		t.Fatalf("unexpected timeline summary: %+v", summary)
	}

	protocolRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/tickets/aiops/protocol", adminToken, nil)
	protocolResp := assertOKResponse(t, protocolRec)
	var protocol struct {
		ProtocolVersion string   `json:"protocolVersion"`
		Types           []string `json:"types"`
	}
	if err := json.Unmarshal(protocolResp.Data, &protocol); err != nil {
		t.Fatalf("unmarshal protocol failed: %v", err)
	}
	if protocol.ProtocolVersion == "" || len(protocol.Types) == 0 {
		t.Fatalf("unexpected protocol: %+v", protocol)
	}
}

func TestTicketAIOpsContextAndFiltersIntegration(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	slaDueAt := time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	createRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/tickets", adminToken, map[string]any{
		"title":    "SLA超时工单",
		"type":     "incident",
		"priority": "P1",
		"env":      "prod",
		"slaDueAt": slaDueAt,
	})
	createResp := assertOKResponse(t, createRec)
	var created models.Ticket
	if err := json.Unmarshal(createResp.Data, &created); err != nil {
		t.Fatalf("unmarshal created ticket failed: %v", err)
	}

	overdueRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/tickets?page=1&pageSize=10&slaOverdue=true", adminToken, nil)
	overdueResp := assertOKResponse(t, overdueRec)
	var overdueList listPayload[models.Ticket]
	if err := json.Unmarshal(overdueResp.Data, &overdueList); err != nil {
		t.Fatalf("unmarshal overdue list failed: %v", err)
	}
	if overdueList.Total == 0 {
		t.Fatalf("expected overdue tickets, got total=%d", overdueList.Total)
	}

	fromDate := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	toDate := time.Now().Add(24 * time.Hour).Format("2006-01-02")
	dateRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/tickets?page=1&pageSize=10&createdFrom="+fromDate+"&createdTo="+toDate, adminToken, nil)
	dateResp := assertOKResponse(t, dateRec)
	var dateList listPayload[models.Ticket]
	if err := json.Unmarshal(dateResp.Data, &dateList); err != nil {
		t.Fatalf("unmarshal date range list failed: %v", err)
	}
	if dateList.Total == 0 {
		t.Fatalf("expected date-range tickets, got total=%d", dateList.Total)
	}

	today := time.Now().Format("2006-01-02")
	todayRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/tickets?page=1&pageSize=10&createdTo="+today, adminToken, nil)
	todayResp := assertOKResponse(t, todayRec)
	var todayList listPayload[models.Ticket]
	if err := json.Unmarshal(todayResp.Data, &todayList); err != nil {
		t.Fatalf("unmarshal createdTo today list failed: %v", err)
	}
	if todayList.Total == 0 {
		t.Fatalf("createdTo date-only should include tickets created during that day")
	}

	addApproverRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/add-approver", created.ID), adminToken, map[string]any{
		"approverId": 1,
		"comment":    "加入审批节点",
	})
	addApproverResp := assertOKResponse(t, addApproverRec)
	var approval models.TicketApproval
	if err := json.Unmarshal(addApproverResp.Data, &approval); err != nil {
		t.Fatalf("unmarshal add-approver failed: %v", err)
	}
	if approval.Status != "pending" {
		t.Fatalf("unexpected approval status: %+v", approval)
	}
	duplicateApproverRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/add-approver", created.ID), adminToken, map[string]any{
		"approverId": 1,
		"comment":    "重复加入审批节点",
	})
	assertOKResponse(t, duplicateApproverRec)

	approvalsRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/approvals", created.ID), adminToken, nil)
	approvalsResp := assertOKResponse(t, approvalsRec)
	var approvals []models.TicketApproval
	if err := json.Unmarshal(approvalsResp.Data, &approvals); err != nil {
		t.Fatalf("unmarshal approvals failed: %v", err)
	}
	pendingForAdmin := 0
	for _, item := range approvals {
		if item.ApproverID == 1 && item.Status == "pending" {
			pendingForAdmin++
		}
	}
	if pendingForAdmin != 1 {
		t.Fatalf("expected duplicate add-approver to be idempotent, pending=%d approvals=%+v", pendingForAdmin, approvals)
	}

	ticketRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d", created.ID), adminToken, nil)
	ticketResp := assertOKResponse(t, ticketRec)
	var ticketSummary struct {
		Ticket models.Ticket `json:"ticket"`
	}
	if err := json.Unmarshal(ticketResp.Data, &ticketSummary); err != nil {
		t.Fatalf("unmarshal ticket summary failed: %v", err)
	}
	if ticketSummary.Ticket.Status != "pending_approval" {
		t.Fatalf("expected pending_approval, got %s", ticketSummary.Ticket.Status)
	}

	cancelRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/cancel", created.ID), adminToken, nil)
	assertOKResponse(t, cancelRec)

	transferCancelledRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/transfer", created.ID), adminToken, map[string]any{
		"assigneeId": 1,
	})
	assertErrorResponse(t, transferCancelledRec, http.StatusConflict, 4030, "closed or cancelled ticket cannot be transferred")

	addApproverCancelledRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/add-approver", created.ID), adminToken, map[string]any{
		"approverId": 1,
	})
	assertErrorResponse(t, addApproverCancelledRec, http.StatusConflict, 4031, "closed or cancelled ticket cannot add approver")

	contextRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/tickets/aiops/context?limit=5", adminToken, nil)
	contextResp := assertOKResponse(t, contextRec)
	var contextData struct {
		ProtocolVersion  string                  `json:"protocolVersion"`
		TraceID          string                  `json:"traceId"`
		OpenTickets      []map[string]any        `json:"openTickets"`
		OverdueTickets   []map[string]any        `json:"overdueTickets"`
		PendingApprovals []models.TicketApproval `json:"pendingApprovals"`
		StatusCounts     map[string]int64        `json:"statusCounts"`
	}
	if err := json.Unmarshal(contextResp.Data, &contextData); err != nil {
		t.Fatalf("unmarshal aiops context failed: %v", err)
	}
	if contextData.ProtocolVersion == "" || contextData.TraceID == "" {
		t.Fatalf("invalid aiops context header: %+v", contextData)
	}
	if len(contextData.StatusCounts) == 0 {
		t.Fatalf("expected status counts in aiops context")
	}
}

func TestTicketVisibilityIsolationIntegration(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	roleID := createRoleViaAPI(t, router, adminToken, "ticket-visibility-viewer")
	userAID := createUserViaAPI(t, router, adminToken, "ticket-user-a", "TicketUser@123")
	userBID := createUserViaAPI(t, router, adminToken, "ticket-user-b", "TicketUser@123")
	bindUserRolesViaAPI(t, router, adminToken, userAID, []uint{roleID})
	bindUserRolesViaAPI(t, router, adminToken, userBID, []uint{roleID})

	listPermissionID := createPermissionViaAPI(t, router, adminToken, map[string]any{
		"name":     "ticket list",
		"type":     "api",
		"key":      "api.tickets.list.custom",
		"resource": "/api/v1/tickets",
		"action":   "GET",
	})
	detailPermissionID := createPermissionViaAPI(t, router, adminToken, map[string]any{
		"name":     "ticket detail",
		"type":     "api",
		"key":      "api.tickets.detail.custom",
		"resource": "/api/v1/tickets/*",
		"action":   "GET",
	})
	bindRolePermissionsViaAPI(t, router, adminToken, roleID, []uint{listPermissionID, detailPermissionID})

	userAToken := loginAndGetToken(t, router, "ticket-user-a", "TicketUser@123")
	userBToken := loginAndGetToken(t, router, "ticket-user-b", "TicketUser@123")

	createVisibleRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/tickets", adminToken, map[string]any{
		"title":       "用户A可见工单",
		"type":        "incident",
		"priority":    "P2",
		"requesterId": userAID,
		"assigneeId":  userAID,
	})
	createVisibleResp := assertOKResponse(t, createVisibleRec)
	var visibleTicket models.Ticket
	if err := json.Unmarshal(createVisibleResp.Data, &visibleTicket); err != nil {
		t.Fatalf("unmarshal visible ticket failed: %v", err)
	}

	createHiddenRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/tickets", adminToken, map[string]any{
		"title":       "用户B不可见工单",
		"type":        "event",
		"priority":    "P3",
		"requesterId": 1,
		"assigneeId":  1,
	})
	createHiddenResp := assertOKResponse(t, createHiddenRec)
	var hiddenTicket models.Ticket
	if err := json.Unmarshal(createHiddenResp.Data, &hiddenTicket); err != nil {
		t.Fatalf("unmarshal hidden ticket failed: %v", err)
	}

	userAListRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/tickets?page=1&pageSize=10", userAToken, nil)
	userAListResp := assertOKResponse(t, userAListRec)
	var userAList listPayload[models.Ticket]
	if err := json.Unmarshal(userAListResp.Data, &userAList); err != nil {
		t.Fatalf("unmarshal user A list failed: %v", err)
	}
	if userAList.Total != 1 || len(userAList.List) != 1 || userAList.List[0].ID != visibleTicket.ID {
		t.Fatalf("unexpected user A list: %+v", userAList)
	}

	userBListRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/tickets?page=1&pageSize=10", userBToken, nil)
	userBListResp := assertOKResponse(t, userBListRec)
	var userBList listPayload[models.Ticket]
	if err := json.Unmarshal(userBListResp.Data, &userBList); err != nil {
		t.Fatalf("unmarshal user B list failed: %v", err)
	}
	if userBList.Total != 0 {
		t.Fatalf("user B should not see ticket before approval, got total=%d", userBList.Total)
	}

	userBGetRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d", visibleTicket.ID), userBToken, nil)
	assertErrorResponse(t, userBGetRec, http.StatusNotFound, 4004, "not found")

	publicCommentRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/comments", visibleTicket.ID), adminToken, map[string]any{
		"content": "公开处理记录",
	})
	assertOKResponse(t, publicCommentRec)

	internalCommentRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/comments", visibleTicket.ID), adminToken, map[string]any{
		"content":  "内部备注 password=secret-token",
		"internal": true,
	})
	assertOKResponse(t, internalCommentRec)

	userAGetHiddenRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d", hiddenTicket.ID), userAToken, nil)
	assertErrorResponse(t, userAGetHiddenRec, http.StatusNotFound, 4004, "not found")

	addApproverRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/add-approver", visibleTicket.ID), adminToken, map[string]any{
		"approverId": userBID,
		"comment":    "加入B审批",
	})
	assertOKResponse(t, addApproverRec)

	userBGetAfterRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d", visibleTicket.ID), userBToken, nil)
	userBGetAfterResp := assertOKResponse(t, userBGetAfterRec)
	var userBSummary struct {
		Comments []models.TicketComment `json:"comments"`
	}
	if err := json.Unmarshal(userBGetAfterResp.Data, &userBSummary); err != nil {
		t.Fatalf("unmarshal user B summary failed: %v", err)
	}
	if len(userBSummary.Comments) != 1 || userBSummary.Comments[0].Internal {
		t.Fatalf("expected user B sees public comments only, got %+v", userBSummary.Comments)
	}

	userAGetRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d", visibleTicket.ID), userAToken, nil)
	userAGetResp := assertOKResponse(t, userAGetRec)
	var userASummary struct {
		Comments []models.TicketComment `json:"comments"`
	}
	if err := json.Unmarshal(userAGetResp.Data, &userASummary); err != nil {
		t.Fatalf("unmarshal user A summary failed: %v", err)
	}
	if len(userASummary.Comments) != 2 {
		t.Fatalf("expected assignee user A sees internal comments, got %+v", userASummary.Comments)
	}
	if strings.Contains(userASummary.Comments[1].Content, "secret-token") {
		t.Fatalf("expected sensitive comment content redacted, got %+v", userASummary.Comments[1].Content)
	}
}

func TestTicketApproveRequiresPendingApproverIntegration(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	roleID := createRoleViaAPI(t, router, adminToken, "ticket-approver-role")
	userAID := createUserViaAPI(t, router, adminToken, "ticket-approver-a", "TicketApprover@123")
	bindUserRolesViaAPI(t, router, adminToken, userAID, []uint{roleID})

	approvePermissionID := createPermissionViaAPI(t, router, adminToken, map[string]any{
		"name":     "ticket approve post",
		"type":     "api",
		"key":      "api.tickets.approve.post",
		"resource": "/api/v1/tickets/*",
		"action":   "POST",
	})
	bindRolePermissionsViaAPI(t, router, adminToken, roleID, []uint{approvePermissionID})

	createRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/tickets", adminToken, map[string]any{
		"title":      "审批人校验工单",
		"type":       "change",
		"priority":   "P1",
		"env":        "prod",
		"assigneeId": 1,
	})
	createResp := assertOKResponse(t, createRec)
	var ticket models.Ticket
	if err := json.Unmarshal(createResp.Data, &ticket); err != nil {
		t.Fatalf("unmarshal ticket failed: %v", err)
	}

	submitRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/submit", ticket.ID), adminToken, nil)
	assertOKResponse(t, submitRec)

	addApproverRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/add-approver", ticket.ID), adminToken, map[string]any{
		"approverId": userAID,
		"comment":    "指定A审批",
	})
	assertOKResponse(t, addApproverRec)

	rejectByAdminRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/reject", ticket.ID), adminToken, map[string]any{
		"comment": "非指定审批人尝试驳回",
	})
	assertErrorResponse(t, rejectByAdminRec, http.StatusForbidden, 4038, "only pending approver can process this ticket")

	userAToken := loginAndGetToken(t, router, "ticket-approver-a", "TicketApprover@123")
	approveByARec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/approve", ticket.ID), userAToken, map[string]any{
		"comment": "指定审批人通过",
	})
	assertOKResponse(t, approveByARec)
}

func TestTicketSubmitIdempotentIntegration(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	createRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/tickets", adminToken, map[string]any{
		"title":    "重复提交幂等测试",
		"type":     "event",
		"priority": "P3",
	})
	createResp := assertOKResponse(t, createRec)
	var ticket models.Ticket
	if err := json.Unmarshal(createResp.Data, &ticket); err != nil {
		t.Fatalf("unmarshal ticket failed: %v", err)
	}

	firstSubmitRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/submit", ticket.ID), adminToken, nil)
	assertOKResponse(t, firstSubmitRec)

	secondSubmitRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/submit", ticket.ID), adminToken, nil)
	assertOKResponse(t, secondSubmitRec)

	flowsRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/flows", ticket.ID), adminToken, nil)
	flowsResp := assertOKResponse(t, flowsRec)
	var flows []models.TicketFlow
	if err := json.Unmarshal(flowsResp.Data, &flows); err != nil {
		t.Fatalf("unmarshal flows failed: %v", err)
	}
	if len(flows) != 2 {
		t.Fatalf("expected idempotent submit keeps 2 flows(create+submit), got %d", len(flows))
	}
}

func TestTicketSLAJobIntegration(t *testing.T) {
	router, database, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	slaDueAt := time.Now().Add(-30 * time.Minute).UTC().Format(time.RFC3339)
	createRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/tickets", adminToken, map[string]any{
		"title":      "SLA任务检测工单",
		"type":       "incident",
		"priority":   "P1",
		"env":        "prod",
		"assigneeId": 1,
		"slaDueAt":   slaDueAt,
	})
	createResp := assertOKResponse(t, createRec)
	var ticket models.Ticket
	if err := json.Unmarshal(createResp.Data, &ticket); err != nil {
		t.Fatalf("unmarshal ticket failed: %v", err)
	}

	jobRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/tickets/sla/jobs", adminToken, map[string]any{
		"limit": 20,
	})
	jobResp := assertOKResponse(t, jobRec)
	var job models.TicketSLAJob
	if err := json.Unmarshal(jobResp.Data, &job); err != nil {
		t.Fatalf("unmarshal sla job failed: %v", err)
	}
	if job.ID == 0 || job.Status == "" {
		t.Fatalf("unexpected sla job: %+v", job)
	}
	if job.ScannedCount <= 0 || job.OverdueCount <= 0 {
		t.Fatalf("expected scanned/overdue > 0, got %+v", job)
	}

	jobDetailRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/tickets/sla/jobs/%d", job.ID), adminToken, nil)
	jobDetailResp := assertOKResponse(t, jobDetailRec)
	var jobDetail models.TicketSLAJob
	if err := json.Unmarshal(jobDetailResp.Data, &jobDetail); err != nil {
		t.Fatalf("unmarshal sla job detail failed: %v", err)
	}
	if jobDetail.ID != job.ID {
		t.Fatalf("unexpected sla job detail id=%d expected=%d", jobDetail.ID, job.ID)
	}

	ticketDetailRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d", ticket.ID), adminToken, nil)
	ticketDetailResp := assertOKResponse(t, ticketDetailRec)
	var summary struct {
		Ticket models.Ticket `json:"ticket"`
	}
	if err := json.Unmarshal(ticketDetailResp.Data, &summary); err != nil {
		t.Fatalf("unmarshal ticket detail failed: %v", err)
	}
	if summary.Ticket.Metadata == nil {
		t.Fatalf("expected metadata updated by sla job")
	}
	if overdue, ok := summary.Ticket.Metadata["slaOverdue"]; !ok || overdue != true {
		t.Fatalf("expected metadata.slaOverdue=true, got %+v", summary.Ticket.Metadata)
	}

	running := models.TicketSLAJob{Status: "running"}
	if err := database.Create(&running).Error; err != nil {
		t.Fatalf("create running sla job failed: %v", err)
	}
	conflictRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/tickets/sla/jobs", adminToken, map[string]any{
		"limit": 20,
	})
	assertErrorResponse(t, conflictRec, http.StatusConflict, 4040, "ticket sla job is already running")
}
