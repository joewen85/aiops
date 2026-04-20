package app

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAIOpsProcurementProtocolIntegration(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	protocolRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/aiops/procurement/protocol", adminToken, nil)
	protocolResp := assertOKResponse(t, protocolRec)
	var protocolData struct {
		ProtocolVersion string   `json:"protocolVersion"`
		SupportedTypes  []string `json:"supportedResourceTypes"`
	}
	if err := json.Unmarshal(protocolResp.Data, &protocolData); err != nil {
		t.Fatalf("unmarshal protocol response failed: %v", err)
	}
	if protocolData.ProtocolVersion == "" {
		t.Fatalf("expected protocolVersion not empty")
	}
	if len(protocolData.SupportedTypes) == 0 {
		t.Fatalf("expected supportedResourceTypes not empty")
	}

	intentRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/aiops/procurement/intents", adminToken, map[string]any{
		"message": "帮我在 ap-southeast-1 购买 2 台 aws 云服务器",
	})
	intentResp := assertOKResponse(t, intentRec)
	var intentData struct {
		Intent struct {
			IntentID     string `json:"intentId"`
			Provider     string `json:"provider"`
			ResourceType string `json:"resourceType"`
			Region       string `json:"region"`
			Quantity     int    `json:"quantity"`
			RawMessage   string `json:"rawMessage"`
		} `json:"intent"`
	}
	if err := json.Unmarshal(intentResp.Data, &intentData); err != nil {
		t.Fatalf("unmarshal intent response failed: %v", err)
	}
	if intentData.Intent.IntentID == "" {
		t.Fatalf("expected intentId not empty")
	}
	if intentData.Intent.Provider != "aws" {
		t.Fatalf("expected provider aws, got=%s", intentData.Intent.Provider)
	}
	if intentData.Intent.ResourceType != "CloudServer" {
		t.Fatalf("expected resourceType CloudServer, got=%s", intentData.Intent.ResourceType)
	}
	if intentData.Intent.Quantity != 2 {
		t.Fatalf("expected quantity=2, got=%d", intentData.Intent.Quantity)
	}

	planRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/aiops/procurement/plans", adminToken, map[string]any{
		"intent": intentData.Intent,
	})
	planResp := assertOKResponse(t, planRec)
	var planData struct {
		Plan struct {
			PlanID        string  `json:"planId"`
			EstimatedCost float64 `json:"estimatedCost"`
		} `json:"plan"`
	}
	if err := json.Unmarshal(planResp.Data, &planData); err != nil {
		t.Fatalf("unmarshal plan response failed: %v", err)
	}
	if planData.Plan.PlanID == "" {
		t.Fatalf("expected planId not empty")
	}
	if planData.Plan.EstimatedCost <= 0 {
		t.Fatalf("expected estimatedCost > 0, got=%f", planData.Plan.EstimatedCost)
	}

	executeRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/aiops/procurement/executions", adminToken, map[string]any{
		"plan": map[string]any{
			"planId": planData.Plan.PlanID,
			"intent": intentData.Intent,
		},
		"dryRun": true,
	})
	executeResp := assertOKResponse(t, executeRec)
	var executeData struct {
		Result struct {
			Status string `json:"status"`
		} `json:"result"`
	}
	if err := json.Unmarshal(executeResp.Data, &executeData); err != nil {
		t.Fatalf("unmarshal execute response failed: %v", err)
	}
	if executeData.Result.Status != "dry_run" {
		t.Fatalf("expected dry_run status, got=%s", executeData.Result.Status)
	}
}
