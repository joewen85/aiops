package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"devops-system/backend/internal/models"
)

func TestKubernetesManagementIntegration(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	mockProdRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/kubernetes/clusters", adminToken, map[string]any{
		"name":           "prod-mock-k8s",
		"apiServer":      "mock://prod",
		"credentialType": "kubeconfig",
		"kubeConfig":     "apiVersion: v1\nclusters: []",
		"env":            "prod",
	})
	assertErrorResponse(t, mockProdRec, http.StatusBadRequest, 3001, "mock apiServer is only allowed in dev/test/local env")

	createRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/kubernetes/clusters", adminToken, map[string]any{
		"name":           "dev-k8s",
		"apiServer":      "mock://dev",
		"credentialType": "kubeconfig",
		"kubeConfig":     "apiVersion: v1\nclusters: []",
		"env":            "dev",
		"region":         "ap-guangzhou",
		"owner":          "sre",
		"metadata": map[string]any{
			"version": "v1.29.3",
		},
	})
	createResp := assertOKResponse(t, createRec)
	var clusterData struct {
		ID         uint   `json:"id"`
		Name       string `json:"name"`
		KubeConfig string `json:"kubeConfig"`
	}
	if err := json.Unmarshal(createResp.Data, &clusterData); err != nil {
		t.Fatalf("unmarshal cluster response failed: %v", err)
	}
	if clusterData.ID == 0 {
		t.Fatalf("expected cluster id > 0")
	}
	if clusterData.KubeConfig == "apiVersion: v1\nclusters: []" || strings.Contains(clusterData.KubeConfig, "clusters") {
		t.Fatalf("cluster response must not expose plaintext kubeConfig: %q", clusterData.KubeConfig)
	}

	checkRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/kubernetes/clusters/%d/check", clusterData.ID), adminToken, nil)
	checkResp := assertOKResponse(t, checkRec)
	var checkData struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(checkResp.Data, &checkData); err != nil {
		t.Fatalf("unmarshal check response failed: %v", err)
	}
	if checkData.Status != "connected" || checkData.Version != "v1.29.3" {
		t.Fatalf("unexpected check result: %+v", checkData)
	}

	syncRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/kubernetes/clusters/%d/sync", clusterData.ID), adminToken, nil)
	syncResp := assertOKResponse(t, syncRec)
	var syncData struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(syncResp.Data, &syncData); err != nil {
		t.Fatalf("unmarshal sync response failed: %v", err)
	}
	if syncData.Count == 0 {
		t.Fatalf("expected synced resources not empty")
	}

	registerTaskOnlyRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/kubernetes/nodes/register/task", adminToken, map[string]any{
		"clusterId":  clusterData.ID,
		"hostname":   "worker-task-only-01",
		"internalIp": "10.0.0.81",
		"executeNow": false,
	})
	registerTaskOnlyResp := assertOKResponse(t, registerTaskOnlyRec)
	var registerTaskOnlyData struct {
		TaskID     uint `json:"taskId"`
		ExecuteNow bool `json:"executeNow"`
		Operation  struct {
			Status string `json:"status"`
		} `json:"operation"`
		TaskLog struct {
			Status string `json:"status"`
		} `json:"taskLog"`
	}
	if err := json.Unmarshal(registerTaskOnlyResp.Data, &registerTaskOnlyData); err != nil {
		t.Fatalf("unmarshal register task-only response failed: %v", err)
	}
	if registerTaskOnlyData.TaskID == 0 || registerTaskOnlyData.ExecuteNow {
		t.Fatalf("expected created task with executeNow=false, got=%+v", registerTaskOnlyData)
	}
	if registerTaskOnlyData.Operation.Status != "pending" || registerTaskOnlyData.TaskLog.Status != "pending" {
		t.Fatalf("expected pending operation/taskLog, got operation=%s taskLog=%s", registerTaskOnlyData.Operation.Status, registerTaskOnlyData.TaskLog.Status)
	}

	nonMockClusterRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/kubernetes/clusters", adminToken, map[string]any{
		"name":           "dev-non-mock-k8s",
		"apiServer":      "https://10.0.0.1:6443",
		"credentialType": "kubeconfig",
		"kubeConfig":     "apiVersion: v1\nclusters: []",
		"env":            "dev",
	})
	nonMockClusterResp := assertOKResponse(t, nonMockClusterRec)
	var nonMockClusterData struct {
		ID uint `json:"id"`
	}
	if err := json.Unmarshal(nonMockClusterResp.Data, &nonMockClusterData); err != nil {
		t.Fatalf("unmarshal non-mock cluster response failed: %v", err)
	}
	registerMissingJoinRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/kubernetes/nodes/register/task", adminToken, map[string]any{
		"clusterId":  nonMockClusterData.ID,
		"hostname":   "worker-no-join",
		"executeNow": true,
		"dryRun":     false,
	})
	assertErrorResponse(t, registerMissingJoinRec, http.StatusBadRequest, 3001, "joinCommand is required for non-mock cluster")

	resourceRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/kubernetes/resources?clusterId=%d&kind=Deployment&page=1&pageSize=10", clusterData.ID), adminToken, nil)
	resourceResp := assertOKResponse(t, resourceRec)
	var resourcePage listPayload[models.KubernetesResourceSnapshot]
	if err := json.Unmarshal(resourceResp.Data, &resourcePage); err != nil {
		t.Fatalf("unmarshal resources page failed: %v", err)
	}
	if resourcePage.Total == 0 || len(resourcePage.List) == 0 {
		t.Fatalf("expected deployment resources not empty")
	}
	resourceDetailRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/kubernetes/resources/%d", resourcePage.List[0].ID), adminToken, nil)
	_ = assertOKResponse(t, resourceDetailRec)
	resourceManifestRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/kubernetes/resources/%d/manifest", resourcePage.List[0].ID), adminToken, nil)
	_ = assertOKResponse(t, resourceManifestRec)

	customManifest := map[string]any{
		"apiVersion": "platform.aiops.local/v1",
		"kind":       "Widget",
		"metadata": map[string]any{
			"name":      "sample-widget",
			"namespace": "default",
			"labels": map[string]any{
				"app": "sample-widget",
			},
		},
		"spec": map[string]any{
			"replicas": 1,
		},
	}
	createResourceDryRunRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/kubernetes/resources", adminToken, map[string]any{
		"clusterId": clusterData.ID,
		"namespace": "default",
		"manifest":  customManifest,
	})
	_ = assertOKResponse(t, createResourceDryRunRec)

	createResourceRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/kubernetes/resources", adminToken, map[string]any{
		"clusterId":        clusterData.ID,
		"namespace":        "default",
		"manifest":         customManifest,
		"dryRun":           false,
		"confirmationText": "确认提交资源",
	})
	_ = assertOKResponse(t, createResourceRec)

	customResourceRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/kubernetes/resources?clusterId=%d&kind=Widget&page=1&pageSize=10", clusterData.ID), adminToken, nil)
	customResourceResp := assertOKResponse(t, customResourceRec)
	var customResourcePage listPayload[models.KubernetesResourceSnapshot]
	if err := json.Unmarshal(customResourceResp.Data, &customResourcePage); err != nil {
		t.Fatalf("unmarshal custom resources page failed: %v", err)
	}
	if customResourcePage.Total != 1 || customResourcePage.List[0].Name != "sample-widget" {
		t.Fatalf("expected custom resource snapshot, got=%+v", customResourcePage)
	}
	customManifest["spec"] = map[string]any{"replicas": 2}
	updateResourceRec := sendJSONRequest(t, router, http.MethodPut, fmt.Sprintf("/api/v1/kubernetes/resources/%d", customResourcePage.List[0].ID), adminToken, map[string]any{
		"namespace":        "default",
		"manifest":         customManifest,
		"dryRun":           false,
		"confirmationText": "确认提交资源",
	})
	_ = assertOKResponse(t, updateResourceRec)

	deleteResourceRec := sendJSONRequest(t, router, http.MethodDelete, fmt.Sprintf("/api/v1/kubernetes/resources/%d", customResourcePage.List[0].ID), adminToken, map[string]any{
		"dryRun":           false,
		"confirmationText": "确认删除资源",
	})
	_ = assertOKResponse(t, deleteResourceRec)

	clusterScopedManifest := map[string]any{
		"apiVersion": "platform.aiops.local/v1",
		"kind":       "ClusterWidget",
		"metadata": map[string]any{
			"name": "global-widget",
		},
		"spec": map[string]any{"enabled": true},
	}
	clusterScopedRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/kubernetes/resources", adminToken, map[string]any{
		"clusterId":        clusterData.ID,
		"manifest":         clusterScopedManifest,
		"dryRun":           false,
		"confirmationText": "确认提交资源",
	})
	_ = assertOKResponse(t, clusterScopedRec)

	deniedManifestRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/kubernetes/resources", adminToken, map[string]any{
		"clusterId": clusterData.ID,
		"manifest": map[string]any{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "Role",
			"metadata": map[string]any{
				"name":      "danger-role",
				"namespace": "default",
			},
		},
	})
	assertErrorResponse(t, deniedManifestRec, http.StatusBadRequest, 3001, "manifest resource kind is not allowed")

	largeManifestRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/kubernetes/resources", adminToken, map[string]any{
		"clusterId": clusterData.ID,
		"manifest": map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      "too-large",
				"namespace": "default",
			},
			"data": map[string]any{"payload": strings.Repeat("x", 300*1024)},
		},
	})
	assertErrorResponse(t, largeManifestRec, http.StatusBadRequest, 3001, "manifest exceeds 262144 bytes")

	dryRunRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/kubernetes/actions/dry-run", adminToken, map[string]any{
		"clusterId": clusterData.ID,
		"namespace": "default",
		"kind":      "Deployment",
		"name":      "web-api",
		"action":    "restart",
	})
	dryRunResp := assertOKResponse(t, dryRunRec)
	var dryRunData struct {
		TraceID   string `json:"traceId"`
		Operation struct {
			Status string `json:"status"`
			DryRun bool   `json:"dryRun"`
		} `json:"operation"`
	}
	if err := json.Unmarshal(dryRunResp.Data, &dryRunData); err != nil {
		t.Fatalf("unmarshal dry-run action response failed: %v", err)
	}
	if dryRunData.TraceID == "" || dryRunData.Operation.Status != "dry_run" || !dryRunData.Operation.DryRun {
		t.Fatalf("unexpected dry-run action result: %+v", dryRunData)
	}

	invalidScaleRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/kubernetes/actions/dry-run", adminToken, map[string]any{
		"clusterId": clusterData.ID,
		"namespace": "default",
		"kind":      "Deployment",
		"name":      "web-api",
		"action":    "scale",
		"params": map[string]any{
			"replicas": -1,
		},
	})
	assertErrorResponse(t, invalidScaleRec, http.StatusBadRequest, 3001, "params.replicas must be non-negative integer")

	deleteWithoutConfirmRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/kubernetes/actions", adminToken, map[string]any{
		"clusterId": clusterData.ID,
		"namespace": "default",
		"kind":      "Deployment",
		"name":      "web-api",
		"action":    "delete",
		"dryRun":    false,
	})
	assertErrorResponse(t, deleteWithoutConfirmRec, http.StatusBadRequest, 3020, "confirmation text is required")

	deleteWithConfirmRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/kubernetes/actions", adminToken, map[string]any{
		"clusterId":        clusterData.ID,
		"namespace":        "default",
		"kind":             "Deployment",
		"name":             "web-api",
		"action":           "delete",
		"dryRun":           false,
		"confirmationText": "确认删除资源",
	})
	_ = assertOKResponse(t, deleteWithConfirmRec)

	operationRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/kubernetes/operations?clusterId=%d&page=1&pageSize=10", clusterData.ID), adminToken, nil)
	operationResp := assertOKResponse(t, operationRec)
	var operationPage listPayload[models.KubernetesOperation]
	if err := json.Unmarshal(operationResp.Data, &operationPage); err != nil {
		t.Fatalf("unmarshal operation page failed: %v", err)
	}
	if operationPage.Total < 2 {
		t.Fatalf("expected operation records, got total=%d", operationPage.Total)
	}
	operationDetailRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/kubernetes/operations/%d", operationPage.List[0].ID), adminToken, nil)
	_ = assertOKResponse(t, operationDetailRec)
}
