package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"devops-system/backend/internal/models"
)

func TestMiddlewareManagementIntegration(t *testing.T) {
	router, database, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	createRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/middleware/instances", adminToken, map[string]any{
		"name":     "redis-prod",
		"type":     "redis",
		"endpoint": "mock://redis",
		"env":      "prod",
		"owner":    "sre",
		"labels":   map[string]any{"app": "core"},
		"metadata": map[string]any{"deployMode": "standalone"},
	})
	createResp := assertOKResponse(t, createRec)
	instanceID := parseIDFromData(t, createResp.Data)

	listRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/middleware/instances?page=1&pageSize=10&keyword=redis&type=redis", adminToken, nil)
	listResp := assertOKResponse(t, listRec)
	var listData listPayload[models.MiddlewareInstance]
	if err := json.Unmarshal(listResp.Data, &listData); err != nil {
		t.Fatalf("unmarshal middleware list failed: %v", err)
	}
	if listData.Total != 1 || len(listData.List) != 1 {
		t.Fatalf("expected one middleware instance, got total=%d len=%d", listData.Total, len(listData.List))
	}

	checkRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/middleware/instances/%d/check", instanceID), adminToken, nil)
	checkResp := assertOKResponse(t, checkRec)
	var checkData struct {
		Result struct {
			Status string `json:"status"`
		} `json:"result"`
	}
	if err := json.Unmarshal(checkResp.Data, &checkData); err != nil {
		t.Fatalf("unmarshal check response failed: %v", err)
	}
	if checkData.Result.Status != "healthy" {
		t.Fatalf("expected healthy check, got=%s", checkData.Result.Status)
	}

	collectRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/middleware/instances/%d/metrics/collect", instanceID), adminToken, nil)
	assertOKResponse(t, collectRec)
	metricsRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/middleware/instances/%d/metrics?page=1&pageSize=10", instanceID), adminToken, nil)
	metricsResp := assertOKResponse(t, metricsRec)
	var metricsData listPayload[models.MiddlewareMetric]
	if err := json.Unmarshal(metricsResp.Data, &metricsData); err != nil {
		t.Fatalf("unmarshal metrics response failed: %v", err)
	}
	if metricsData.Total == 0 {
		t.Fatalf("expected collected metrics")
	}

	protocolRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/middleware/aiops/protocol", adminToken, nil)
	protocolResp := assertOKResponse(t, protocolRec)
	var protocolData struct {
		ProtocolVersion string `json:"protocolVersion"`
		ActionEndpoint  string `json:"actionEndpoint"`
	}
	if err := json.Unmarshal(protocolResp.Data, &protocolData); err != nil {
		t.Fatalf("unmarshal protocol response failed: %v", err)
	}
	if protocolData.ProtocolVersion == "" || protocolData.ActionEndpoint != "/api/v1/middleware/actions" {
		t.Fatalf("unexpected protocol data: %+v", protocolData)
	}

	dryRunRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/middleware/actions", adminToken, map[string]any{
		"instanceId": instanceID,
		"type":       "redis",
		"action":     "flushdb",
	})
	dryRunResp := assertOKResponse(t, dryRunRec)
	var dryRunData struct {
		DryRun struct {
			RiskLevel string `json:"riskLevel"`
		} `json:"dryRun"`
	}
	if err := json.Unmarshal(dryRunResp.Data, &dryRunData); err != nil {
		t.Fatalf("unmarshal dry-run response failed: %v", err)
	}
	if dryRunData.DryRun.RiskLevel != "P1" {
		t.Fatalf("expected P1 dry-run risk, got=%s", dryRunData.DryRun.RiskLevel)
	}

	unsafeRunRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/middleware/actions", adminToken, map[string]any{
		"instanceId": instanceID,
		"type":       "redis",
		"action":     "flushdb",
		"dryRun":     false,
	})
	assertErrorResponse(t, unsafeRunRec, http.StatusBadRequest, 3020, "confirmation text is required")

	confirmedRunRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/middleware/actions", adminToken, map[string]any{
		"instanceId":       instanceID,
		"type":             "redis",
		"action":           "flushdb",
		"dryRun":           false,
		"confirmationText": "确认删除资源",
	})
	assertOKResponse(t, confirmedRunRec)

	runningOperation := models.MiddlewareOperation{
		TraceID:    "test-running",
		InstanceID: instanceID,
		Type:       "redis",
		Action:     "flushdb",
		Status:     "running",
		DryRun:     false,
		RiskLevel:  "P1",
	}
	if err := database.Create(&runningOperation).Error; err != nil {
		t.Fatalf("create running middleware operation failed: %v", err)
	}
	conflictRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/middleware/actions", adminToken, map[string]any{
		"instanceId":       instanceID,
		"type":             "redis",
		"action":           "flushdb",
		"dryRun":           false,
		"confirmationText": "确认删除资源",
	})
	assertErrorResponse(t, conflictRec, http.StatusConflict, 4023, "middleware action is already running")

	deleteHealthyRec := sendJSONRequest(t, router, http.MethodDelete, fmt.Sprintf("/api/v1/middleware/instances/%d", instanceID), adminToken, map[string]any{
		"confirmationText": "确认删除资源",
	})
	assertErrorResponse(t, deleteHealthyRec, http.StatusConflict, 4020, "healthy middleware instance cannot be deleted")

	unsafeEndpointRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/middleware/instances", adminToken, map[string]any{
		"name":     "redis-unsafe",
		"type":     "redis",
		"endpoint": "redis://169.254.169.254:6379",
		"env":      "prod",
	})
	if unsafeEndpointRec.Code != http.StatusBadRequest {
		t.Fatalf("expected unsafe endpoint rejected, status=%d body=%s", unsafeEndpointRec.Code, unsafeEndpointRec.Body.String())
	}
}
