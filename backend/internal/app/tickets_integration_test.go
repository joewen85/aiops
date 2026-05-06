package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"devops-system/backend/internal/models"
)

func TestTicketManagementIntegration(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)
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
		"title":    "云资源申请",
		"type":     "resource_request",
		"priority": "P2",
		"env":      "prod",
	})
	createResp = assertOKResponse(t, createRec)
	if err := json.Unmarshal(createResp.Data, &created); err != nil {
		t.Fatalf("unmarshal second ticket failed: %v", err)
	}

	submitRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/submit", created.ID), adminToken, nil)
	assertOKResponse(t, submitRec)

	deleteSubmittedRec := sendJSONRequest(t, router, http.MethodDelete, fmt.Sprintf("/api/v1/tickets/%d", created.ID), adminToken, map[string]any{
		"confirmationText": "确认删除资源",
	})
	assertErrorResponse(t, deleteSubmittedRec, http.StatusConflict, 4026, "ticket cannot be deleted in current status")

	approveRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/approve", created.ID), adminToken, map[string]any{
		"comment": "审批通过",
	})
	assertOKResponse(t, approveRec)

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

	missingConfirmRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/operations/execute", created.ID), adminToken, map[string]any{
		"module": "cloud",
		"action": "delete_instance",
	})
	assertErrorResponse(t, missingConfirmRec, http.StatusBadRequest, 3020, "confirmation text is required")

	executeRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/operations/execute", created.ID), adminToken, map[string]any{
		"module":           "cloud",
		"action":           "delete_instance",
		"confirmationText": "确认删除资源",
	})
	assertOKResponse(t, executeRec)

	commentRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/comments", created.ID), adminToken, map[string]any{
		"content": "已完成资源申请 dry-run 和审批",
	})
	assertOKResponse(t, commentRec)

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
