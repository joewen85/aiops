package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"devops-system/backend/internal/models"
)

func TestDockerManagementAIOpsProtocolAndDryRunIntegration(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	protocolRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/docker/aiops/protocol", adminToken, nil)
	protocolResp := assertOKResponse(t, protocolRec)
	var protocol struct {
		ProtocolVersion string `json:"protocolVersion"`
		ActionEndpoint  string `json:"actionEndpoint"`
	}
	if err := json.Unmarshal(protocolResp.Data, &protocol); err != nil {
		t.Fatalf("unmarshal docker protocol failed: %v", err)
	}
	if protocol.ProtocolVersion != "aiops.dockerops.v1alpha1" || protocol.ActionEndpoint == "" {
		t.Fatalf("unexpected docker protocol: %+v", protocol)
	}

	host := createDockerHostForTest(t, router, adminToken, "http://127.0.0.1:2375")
	actionRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/docker/actions", adminToken, map[string]any{
		"hostId":       host.ID,
		"resourceType": "container",
		"resourceId":   "container-1",
		"action":       "restart",
		"dryRun":       true,
		"params": map[string]any{
			"reason": "integration-test",
		},
	})
	actionResp := assertOKResponse(t, actionRec)
	var actionData struct {
		ProtocolVersion string                 `json:"protocolVersion"`
		TraceID         string                 `json:"traceId"`
		DryRun          map[string]interface{} `json:"dryRun"`
		Operation       models.DockerOperation `json:"operation"`
	}
	if err := json.Unmarshal(actionResp.Data, &actionData); err != nil {
		t.Fatalf("unmarshal dry-run action failed: %v", err)
	}
	if actionData.TraceID == "" || actionData.Operation.Status != "dry_run" || !actionData.Operation.DryRun {
		t.Fatalf("unexpected dry-run response: %+v", actionData)
	}
	if actionData.DryRun["riskLevel"] != "P2" {
		t.Fatalf("expected riskLevel P2, got=%v", actionData.DryRun["riskLevel"])
	}

	defaultDryRunRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/docker/actions", adminToken, map[string]any{
		"hostId":       host.ID,
		"resourceType": "container",
		"resourceId":   "container-1",
		"action":       "restart",
	})
	defaultDryRunResp := assertOKResponse(t, defaultDryRunRec)
	var defaultDryRunData struct {
		Operation models.DockerOperation `json:"operation"`
	}
	if err := json.Unmarshal(defaultDryRunResp.Data, &defaultDryRunData); err != nil {
		t.Fatalf("unmarshal default dry-run response failed: %v", err)
	}
	if defaultDryRunData.Operation.Status != "dry_run" || !defaultDryRunData.Operation.DryRun {
		t.Fatalf("expected omitted dryRun defaults to dry_run, got %+v", defaultDryRunData.Operation)
	}
}

func TestDockerHostCheckAndResourceQueryIntegration(t *testing.T) {
	dockerAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/_ping":
			w.Header().Set("API-Version", "1.45")
			_, _ = w.Write([]byte("OK"))
		case "/version":
			_ = json.NewEncoder(w).Encode(map[string]any{"Version": "25.0.0"})
		case "/containers/json":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"Id": "container-1", "Names": []string{"/web"}, "Image": "nginx:latest", "State": "running", "Status": "Up 2 minutes"},
			})
		case "/containers/container-1/restart":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected docker api request path=%s", r.URL.Path)
		}
	}))
	defer dockerAPI.Close()

	router, _, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")
	host := createDockerHostForTest(t, router, adminToken, dockerAPI.URL)

	checkRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/docker/hosts/%d/check", host.ID), adminToken, nil)
	checkResp := assertOKResponse(t, checkRec)
	var checkData struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(checkResp.Data, &checkData); err != nil {
		t.Fatalf("unmarshal check response failed: %v", err)
	}
	if checkData.Status != "connected" || checkData.Version != "25.0.0" {
		t.Fatalf("unexpected check response: %+v", checkData)
	}

	listRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/docker/hosts/%d/resources?type=containers&page=1&pageSize=10", host.ID), adminToken, nil)
	listResp := assertOKResponse(t, listRec)
	var page listPayload[struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Type   string `json:"type"`
		Status string `json:"status"`
	}]
	if err := json.Unmarshal(listResp.Data, &page); err != nil {
		t.Fatalf("unmarshal docker resources failed: %v", err)
	}
	if page.Total != 1 || page.List[0].Name != "web" || page.List[0].Type != "container" {
		t.Fatalf("unexpected resources: %+v", page)
	}

	actionRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/docker/actions", adminToken, map[string]any{
		"hostId":       host.ID,
		"resourceType": "container",
		"resourceId":   "container-1",
		"action":       "restart",
		"dryRun":       false,
	})
	actionResp := assertOKResponse(t, actionRec)
	var actionData struct {
		Operation models.DockerOperation `json:"operation"`
	}
	if err := json.Unmarshal(actionResp.Data, &actionData); err != nil {
		t.Fatalf("unmarshal action response failed: %v", err)
	}
	if actionData.Operation.Status != "success" || actionData.Operation.TraceID == "" {
		t.Fatalf("unexpected action operation: %+v", actionData.Operation)
	}
}

func TestDockerRemoveActionsRequireConfirmAndExecuteIntegration(t *testing.T) {
	deleted := map[string]bool{}
	dockerAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("unexpected docker api method=%s path=%s", r.Method, r.URL.Path)
		}
		switch r.URL.Path {
		case "/images/sha256:image-1":
			deleted["image"] = true
			_ = json.NewEncoder(w).Encode([]map[string]any{{"Deleted": "sha256:image-1"}})
		case "/networks/network-1":
			deleted["network"] = true
			w.WriteHeader(http.StatusNoContent)
		case "/volumes/volume-1":
			deleted["volume"] = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected docker delete path=%s", r.URL.Path)
		}
	}))
	defer dockerAPI.Close()

	router, _, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")
	host := createDockerHostForTest(t, router, adminToken, dockerAPI.URL)

	missingConfirmRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/docker/actions", adminToken, map[string]any{
		"hostId":       host.ID,
		"resourceType": "image",
		"resourceId":   "sha256:image-1",
		"action":       "remove",
		"dryRun":       false,
	})
	if missingConfirmRec.Code != http.StatusBadRequest {
		t.Fatalf("expected missing confirmation rejected, got status=%d body=%s", missingConfirmRec.Code, missingConfirmRec.Body.String())
	}

	cases := []struct {
		resourceType string
		resourceID   string
		key          string
	}{
		{"image", "sha256:image-1", "image"},
		{"network", "network-1", "network"},
		{"volume", "volume-1", "volume"},
	}
	for _, tc := range cases {
		rec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/docker/actions", adminToken, map[string]any{
			"hostId":           host.ID,
			"resourceType":     tc.resourceType,
			"resourceId":       tc.resourceID,
			"action":           "remove",
			"dryRun":           false,
			"confirmationText": "确认删除资源",
		})
		resp := assertOKResponse(t, rec)
		var payload struct {
			Operation models.DockerOperation `json:"operation"`
		}
		if err := json.Unmarshal(resp.Data, &payload); err != nil {
			t.Fatalf("unmarshal %s remove response failed: %v", tc.resourceType, err)
		}
		if payload.Operation.Status != "success" || !deleted[tc.key] {
			t.Fatalf("expected %s removed successfully, operation=%+v deleted=%+v", tc.resourceType, payload.Operation, deleted)
		}
	}
}

func TestDockerComposeDeployActionRequiresConfirmAndExecutesIntegration(t *testing.T) {
	tempDir := t.TempDir()
	fakeDocker := filepath.Join(tempDir, "docker")
	if err := os.WriteFile(fakeDocker, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$AIOPS_FAKE_DOCKER_ARGS\"\nexit 0\n"), 0755); err != nil {
		t.Fatalf("write fake docker failed: %v", err)
	}
	argsFile := filepath.Join(tempDir, "args.txt")
	oldPath := os.Getenv("PATH")
	oldArgs := os.Getenv("AIOPS_FAKE_DOCKER_ARGS")
	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+oldPath)
	t.Setenv("AIOPS_FAKE_DOCKER_ARGS", argsFile)
	t.Cleanup(func() {
		_ = os.Setenv("PATH", oldPath)
		_ = os.Setenv("AIOPS_FAKE_DOCKER_ARGS", oldArgs)
	})

	router, database, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")
	host := createDockerHostForTest(t, router, adminToken, "http://127.0.0.1:2375")
	stack := models.DockerComposeStack{
		HostID:  host.ID,
		Name:    "web_stack",
		Content: "services:\n  web:\n    image: nginx:latest\n",
		Status:  "draft",
	}
	if err := database.Create(&stack).Error; err != nil {
		t.Fatalf("create compose stack failed: %v", err)
	}

	missingConfirmRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/docker/actions", adminToken, map[string]any{
		"hostId":       host.ID,
		"resourceType": "compose",
		"resourceId":   fmt.Sprintf("%d", stack.ID),
		"action":       "deploy",
		"dryRun":       false,
	})
	if missingConfirmRec.Code != http.StatusBadRequest {
		t.Fatalf("expected compose deploy missing confirmation rejected, got status=%d body=%s", missingConfirmRec.Code, missingConfirmRec.Body.String())
	}

	rec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/docker/actions", adminToken, map[string]any{
		"hostId":           host.ID,
		"resourceType":     "compose",
		"resourceId":       fmt.Sprintf("%d", stack.ID),
		"action":           "deploy",
		"dryRun":           false,
		"confirmationText": "确认删除资源",
	})
	resp := assertOKResponse(t, rec)
	var payload struct {
		Operation models.DockerOperation `json:"operation"`
	}
	if err := json.Unmarshal(resp.Data, &payload); err != nil {
		t.Fatalf("unmarshal compose deploy response failed: %v", err)
	}
	if payload.Operation.Status != "success" {
		t.Fatalf("expected compose deploy success, got %+v", payload.Operation)
	}
	var saved models.DockerComposeStack
	if err := database.First(&saved, stack.ID).Error; err != nil {
		t.Fatalf("query saved compose stack failed: %v", err)
	}
	if saved.Status != "running" || saved.LastDeployedAt == nil {
		t.Fatalf("expected compose stack running after deploy, got %+v", saved)
	}
	rawArgs, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read fake docker args failed: %v", err)
	}
	if string(rawArgs) == "" {
		t.Fatalf("expected fake docker invoked")
	}
}

func TestDockerEndpointAndDeleteSafetyIntegration(t *testing.T) {
	router, database, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	prodHTTPRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/docker/hosts", adminToken, map[string]any{
		"name":     "unsafe-prod",
		"endpoint": "http://127.0.0.1:2375",
		"env":      "prod",
	})
	if prodHTTPRec.Code != http.StatusBadRequest {
		t.Fatalf("expected prod loopback http endpoint rejected, got status=%d body=%s", prodHTTPRec.Code, prodHTTPRec.Body.String())
	}

	metadataRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/docker/hosts", adminToken, map[string]any{
		"name":     "metadata",
		"endpoint": "http://169.254.169.254:2375",
		"env":      "test",
	})
	if metadataRec.Code != http.StatusBadRequest {
		t.Fatalf("expected metadata endpoint rejected, got status=%d body=%s", metadataRec.Code, metadataRec.Body.String())
	}

	host := models.DockerHost{
		Name:     "connected-host",
		Endpoint: "unix:///var/run/docker.sock",
		Env:      "prod",
		Status:   "connected",
	}
	if err := database.Create(&host).Error; err != nil {
		t.Fatalf("create connected docker host failed: %v", err)
	}
	deleteHostRec := sendJSONRequest(t, router, http.MethodDelete, fmt.Sprintf("/api/v1/docker/hosts/%d", host.ID), adminToken, nil)
	if deleteHostRec.Code != http.StatusConflict {
		t.Fatalf("expected connected docker host delete conflict, got status=%d body=%s", deleteHostRec.Code, deleteHostRec.Body.String())
	}

	stack := models.DockerComposeStack{
		HostID:  host.ID,
		Name:    "running-stack",
		Content: "services: {}",
		Status:  "running",
	}
	if err := database.Create(&stack).Error; err != nil {
		t.Fatalf("create running stack failed: %v", err)
	}
	deleteStackRec := sendJSONRequest(t, router, http.MethodDelete, fmt.Sprintf("/api/v1/docker/compose/stacks/%d", stack.ID), adminToken, nil)
	if deleteStackRec.Code != http.StatusConflict {
		t.Fatalf("expected running stack delete conflict, got status=%d body=%s", deleteStackRec.Code, deleteStackRec.Body.String())
	}
}

func TestDockerActionRejectsDuplicateRunningOperationIntegration(t *testing.T) {
	router, database, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")
	host := createDockerHostForTest(t, router, adminToken, "http://127.0.0.1:2375")

	running := models.DockerOperation{
		TraceID:      "trace-running",
		HostID:       host.ID,
		ResourceType: "container",
		ResourceID:   "container-1",
		Action:       "restart",
		Status:       "running",
		DryRun:       false,
		RiskLevel:    "P2",
	}
	if err := database.Create(&running).Error; err != nil {
		t.Fatalf("create running docker operation failed: %v", err)
	}

	rec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/docker/actions", adminToken, map[string]any{
		"hostId":       host.ID,
		"resourceType": "container",
		"resourceId":   "container-1",
		"action":       "restart",
		"dryRun":       false,
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected duplicate running action conflict, got status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func createDockerHostForTest(t *testing.T, router *gin.Engine, token string, endpoint string) models.DockerHost {
	t.Helper()
	rec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/docker/hosts", token, map[string]any{
		"name":     "local-docker",
		"endpoint": endpoint,
		"env":      "test",
		"owner":    "sre",
	})
	resp := assertOKResponse(t, rec)
	var host models.DockerHost
	if err := json.Unmarshal(resp.Data, &host); err != nil {
		t.Fatalf("unmarshal docker host failed: %v", err)
	}
	if host.ID == 0 {
		t.Fatalf("expected docker host id > 0")
	}
	return host
}
