package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"devops-system/backend/internal/models"
)

type cmdbResourceCreateData struct {
	Action   string              `json:"action"`
	Resource models.ResourceItem `json:"resource"`
}

func TestCMDBResourceRelationAndImpactIntegration(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	service := createCMDBResourceViaAPI(t, router, adminToken, map[string]any{
		"ciId":      "aliyun:cn-shanghai:service:live-api",
		"type":      "Service",
		"name":      "live-api",
		"cloud":     "aliyun",
		"region":    "cn-shanghai",
		"env":       "prod",
		"owner":     "live-team",
		"source":    "Manual",
		"lifecycle": "active",
	})
	db := createCMDBResourceViaAPI(t, router, adminToken, map[string]any{
		"ciId":      "aliyun:account-a:cn-shanghai:postgres:pg-live",
		"type":      "PostgreSQL",
		"name":      "pg-live",
		"cloud":     "aliyun",
		"region":    "cn-shanghai",
		"env":       "prod",
		"owner":     "dba",
		"source":    "Manual",
		"lifecycle": "active",
	})
	cnCluster := createCMDBResourceViaAPI(t, router, adminToken, map[string]any{
		"ciId":      "aliyun:cn-shanghai:k8scluster:live-cn",
		"type":      "K8sCluster",
		"name":      "live-cn",
		"cloud":     "aliyun",
		"region":    "cn-shanghai",
		"env":       "prod",
		"owner":     "platform",
		"source":    "Manual",
		"lifecycle": "active",
	})
	sgCluster := createCMDBResourceViaAPI(t, router, adminToken, map[string]any{
		"ciId":      "aws:ap-southeast-1:k8scluster:live-sg",
		"type":      "K8sCluster",
		"name":      "live-sg",
		"cloud":     "aws",
		"region":    "ap-southeast-1",
		"env":       "prod",
		"owner":     "platform",
		"source":    "Manual",
		"lifecycle": "active",
	})

	createCMDBRelationViaAPI(t, router, adminToken, map[string]any{
		"fromCiId":     service.CIID,
		"toCiId":       db.CIID,
		"relationType": "connects_to",
		"direction":    "outbound",
		"criticality":  "P1",
		"confidence":   1,
	})
	createCMDBRelationViaAPI(t, router, adminToken, map[string]any{
		"fromCiId":     service.CIID,
		"toCiId":       cnCluster.CIID,
		"relationType": "deployed_on",
		"direction":    "outbound",
		"criticality":  "P0",
		"confidence":   1,
	})
	createCMDBRelationViaAPI(t, router, adminToken, map[string]any{
		"fromCiId":     service.CIID,
		"toCiId":       sgCluster.CIID,
		"relationType": "deployed_on",
		"direction":    "outbound",
		"criticality":  "P0",
		"confidence":   1,
	})

	resourcesRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/cmdb/resources?keyword=live-api&page=1&pageSize=10", adminToken, nil)
	resourcesResp := assertOKResponse(t, resourcesRec)
	var resourcesData listPayload[models.ResourceItem]
	if err := json.Unmarshal(resourcesResp.Data, &resourcesData); err != nil {
		t.Fatalf("unmarshal resources list failed: %v", err)
	}
	if resourcesData.Total < 1 {
		t.Fatalf("expected at least one matched resource")
	}

	downstreamRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/cmdb/resources/%d/downstream", service.ID), adminToken, nil)
	downstreamResp := assertOKResponse(t, downstreamRec)
	var downstreamData struct {
		Nodes     []models.ResourceItem     `json:"nodes"`
		Relations []models.ResourceRelation `json:"relations"`
	}
	if err := json.Unmarshal(downstreamResp.Data, &downstreamData); err != nil {
		t.Fatalf("unmarshal downstream data failed: %v", err)
	}
	if len(downstreamData.Nodes) < 2 {
		t.Fatalf("expected downstream nodes >= 2, got=%d", len(downstreamData.Nodes))
	}
	if len(downstreamData.Relations) < 1 {
		t.Fatalf("expected downstream relations >= 1, got=%d", len(downstreamData.Relations))
	}

	impactRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/cmdb/impact/"+service.CIID, adminToken, nil)
	impactResp := assertOKResponse(t, impactRec)
	var impactData struct {
		ImpactCount int                       `json:"impactCount"`
		Relations   []models.ResourceRelation `json:"relations"`
	}
	if err := json.Unmarshal(impactResp.Data, &impactData); err != nil {
		t.Fatalf("unmarshal impact data failed: %v", err)
	}
	if impactData.ImpactCount < 1 {
		t.Fatalf("expected impact count >= 1, got=%d", impactData.ImpactCount)
	}
	if len(impactData.Relations) < 1 {
		t.Fatalf("expected impact relations >= 1")
	}

	failoverRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/cmdb/regions/cn-shanghai/failover", adminToken, nil)
	failoverResp := assertOKResponse(t, failoverRec)
	var failoverData struct {
		Region   string `json:"region"`
		Services []struct {
			ServiceCIID     string   `json:"serviceCiId"`
			CanFailover     bool     `json:"canFailover"`
			TakeoverRegions []string `json:"takeoverRegions"`
		} `json:"services"`
	}
	if err := json.Unmarshal(failoverResp.Data, &failoverData); err != nil {
		t.Fatalf("unmarshal failover data failed: %v", err)
	}
	if failoverData.Region != "cn-shanghai" {
		t.Fatalf("unexpected failover region=%s", failoverData.Region)
	}
	found := false
	for _, item := range failoverData.Services {
		if item.ServiceCIID != service.CIID {
			continue
		}
		found = true
		if !item.CanFailover {
			t.Fatalf("expected service can failover")
		}
		if len(item.TakeoverRegions) == 0 {
			t.Fatalf("expected takeover regions not empty")
		}
	}
	if !found {
		t.Fatalf("expected service in failover result")
	}
}

func TestCMDBSyncJobIntegration(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	cloudAccountRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/cloud/accounts", adminToken, map[string]any{
		"provider":  "aws",
		"name":      "aws-prod",
		"accessKey": "ak",
		"secretKey": "sk",
		"region":    "ap-southeast-1",
	})
	_ = assertOKResponse(t, cloudAccountRec)

	k8sRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/kubernetes/clusters", adminToken, map[string]any{
		"name":      "k8s-live",
		"apiServer": "https://k8s.example",
		"kubeConfig": `
apiVersion: v1
kind: Config
contexts:
- context:
    cluster: live
    namespace: live-core
  name: live
current-context: live
`,
	})
	_ = assertOKResponse(t, k8sRec)

	syncRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/cmdb/sync/jobs", adminToken, map[string]any{
		"sources":  []string{"CloudAPI", "K8s"},
		"fullScan": false,
	})
	syncResp := assertOKResponse(t, syncRec)
	var job models.ResourceSyncJob
	if err := json.Unmarshal(syncResp.Data, &job); err != nil {
		t.Fatalf("unmarshal sync job failed: %v", err)
	}
	if job.ID == 0 {
		t.Fatalf("expected sync job id > 0")
	}
	if job.Status == "" {
		t.Fatalf("expected sync status")
	}

	jobDetailRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/cmdb/sync/jobs/%d", job.ID), adminToken, nil)
	jobDetailResp := assertOKResponse(t, jobDetailRec)
	var jobDetail struct {
		Job   models.ResourceSyncJob       `json:"job"`
		Items []models.ResourceSyncJobItem `json:"items"`
	}
	if err := json.Unmarshal(jobDetailResp.Data, &jobDetail); err != nil {
		t.Fatalf("unmarshal job detail failed: %v", err)
	}
	if jobDetail.Job.ID != job.ID {
		t.Fatalf("unexpected job detail id=%d", jobDetail.Job.ID)
	}
	if len(jobDetail.Items) == 0 {
		t.Fatalf("expected sync job items not empty")
	}

	retryRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/cmdb/sync/jobs/%d/retry", job.ID), adminToken, nil)
	retryResp := assertOKResponse(t, retryRec)
	var retryJob models.ResourceSyncJob
	if err := json.Unmarshal(retryResp.Data, &retryJob); err != nil {
		t.Fatalf("unmarshal retry job failed: %v", err)
	}
	if retryJob.ID == 0 || retryJob.ID == job.ID {
		t.Fatalf("expected new retry job id, got=%d", retryJob.ID)
	}

	vmRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/cmdb/resources?type=VM&page=1&pageSize=10", adminToken, nil)
	vmResp := assertOKResponse(t, vmRec)
	var vmData listPayload[models.ResourceItem]
	if err := json.Unmarshal(vmResp.Data, &vmData); err != nil {
		t.Fatalf("unmarshal vm list failed: %v", err)
	}
	if vmData.Total < 1 {
		t.Fatalf("expected VM resources from cloud sync")
	}
	var vm models.ResourceItem
	foundVM := false
	for _, item := range vmData.List {
		if item.Type != "VM" {
			continue
		}
		vm = item
		foundVM = true
		break
	}
	if !foundVM {
		t.Fatalf("expected at least one VM item in list")
	}
	if vm.Attributes == nil {
		t.Fatalf("expected VM attributes not nil")
	}
	requiredKeys := []string{"cpu", "memory", "disk", "privateIp", "publicIp", "os", "expiresAt", "accountId", "assetId"}
	for _, key := range requiredKeys {
		value, ok := vm.Attributes[key]
		if !ok {
			t.Fatalf("expected VM attributes contains key=%s", key)
		}
		text := fmt.Sprintf("%v", value)
		if text == "" || text == "<nil>" {
			t.Fatalf("expected VM attributes key=%s has non-empty value", key)
		}
	}
	expiresAtRaw := fmt.Sprintf("%v", vm.Attributes["expiresAt"])
	expiresAt, err := time.Parse(time.RFC3339, expiresAtRaw)
	if err != nil {
		t.Fatalf("expected expiresAt is RFC3339 time, got=%s err=%v", expiresAtRaw, err)
	}
	if !expiresAt.After(time.Now()) {
		t.Fatalf("expected expiresAt is in the future, got=%s", expiresAtRaw)
	}

	clusterListRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/cmdb/resources?type=K8sCluster&page=1&pageSize=10", adminToken, nil)
	clusterListResp := assertOKResponse(t, clusterListRec)
	var clusterData listPayload[models.ResourceItem]
	if err := json.Unmarshal(clusterListResp.Data, &clusterData); err != nil {
		t.Fatalf("unmarshal cluster list failed: %v", err)
	}
	if clusterData.Total < 1 {
		t.Fatalf("expected K8sCluster resources from k8s sync")
	}
}

func TestCMDBSyncJobRejectWhenRunningIntegration(t *testing.T) {
	router, db, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	now := time.Now()
	if err := db.Create(&models.ResourceSyncJob{
		Status:    "running",
		StartedAt: &now,
	}).Error; err != nil {
		t.Fatalf("seed running cmdb sync job failed: %v", err)
	}
	jobRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/cmdb/sync/jobs", adminToken, map[string]any{
		"sources":  []string{"Manual"},
		"fullScan": false,
	})
	assertErrorResponse(t, jobRec, http.StatusConflict, 4013, "cmdb sync job is already running")

	done := time.Now()
	if err := db.Model(&models.ResourceSyncJob{}).Where("status = ?", "running").Updates(map[string]any{
		"status":      "success",
		"finished_at": &done,
	}).Error; err != nil {
		t.Fatalf("close running cmdb sync job failed: %v", err)
	}

	original := models.ResourceSyncJob{Status: "success"}
	if err := db.Create(&original).Error; err != nil {
		t.Fatalf("seed original cmdb sync job failed: %v", err)
	}
	now = time.Now()
	if err := db.Create(&models.ResourceSyncJob{
		Status:    "running",
		StartedAt: &now,
	}).Error; err != nil {
		t.Fatalf("seed second running cmdb sync job failed: %v", err)
	}
	retryRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/cmdb/sync/jobs/%d/retry", original.ID), adminToken, nil)
	assertErrorResponse(t, retryRec, http.StatusConflict, 4013, "cmdb sync job is already running")
}

func TestCMDBResourceDetailAndVMActionValidationIntegration(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	vm := createCMDBResourceViaAPI(t, router, adminToken, map[string]any{
		"ciId":      "cmdb:vm:manual-001",
		"type":      "VM",
		"name":      "manual-vm-001",
		"cloud":     "aliyun",
		"region":    "cn-hangzhou",
		"env":       "prod",
		"owner":     "platform",
		"source":    "Manual",
		"lifecycle": "active",
		"attributes": map[string]any{
			"cpu":       "2",
			"memory":    "4Gi",
			"privateIp": "10.0.0.10",
		},
	})

	getRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/cmdb/resources/%d", vm.ID), adminToken, nil)
	getResp := assertOKResponse(t, getRec)
	var fetched models.ResourceItem
	if err := json.Unmarshal(getResp.Data, &fetched); err != nil {
		t.Fatalf("unmarshal cmdb resource detail failed: %v", err)
	}
	if fetched.ID != vm.ID {
		t.Fatalf("unexpected resource detail id=%d expected=%d", fetched.ID, vm.ID)
	}
	if fetched.CIID != vm.CIID {
		t.Fatalf("unexpected resource detail ciId=%s expected=%s", fetched.CIID, vm.CIID)
	}

	restartRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/cmdb/resources/%d/actions/restart", vm.ID), adminToken, nil)
	assertErrorResponse(t, restartRec, http.StatusBadRequest, 3001, "resource accountId is empty")

	stopRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/cmdb/resources/%d/actions/stop", vm.ID), adminToken, nil)
	assertErrorResponse(t, stopRec, http.StatusBadRequest, 3001, "resource accountId is empty")
}

func TestCMDBVMActionRejectUnsyncedResourceIntegration(t *testing.T) {
	router, db, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	account := models.CloudAccount{
		Provider:   "aliyun",
		Name:       "aliyun-prod",
		AccessKey:  "ak",
		SecretKey:  "sk",
		Region:     "cn-hangzhou",
		IsVerified: true,
	}
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("seed cloud account failed: %v", err)
	}

	vm := createCMDBResourceViaAPI(t, router, adminToken, map[string]any{
		"ciId":      "aliyun:manual:cn-hangzhou:vm:blocked-001",
		"type":      "VM",
		"name":      "blocked-vm",
		"cloud":     "aliyun",
		"region":    "cn-hangzhou",
		"env":       "prod",
		"owner":     "platform",
		"source":    "CloudAPI",
		"lifecycle": "active",
		"attributes": map[string]any{
			"accountId":  account.ID,
			"instanceId": "i-blocked001",
		},
	})

	restartRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/cmdb/resources/%d/actions/restart", vm.ID), adminToken, nil)
	assertErrorResponse(t, restartRec, http.StatusBadRequest, 3001, "resource is not linked to synced cloud asset")
}

func TestCMDBDeleteResourceRejectRunningStatusIntegration(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	vm := createCMDBResourceViaAPI(t, router, adminToken, map[string]any{
		"ciId":      "aliyun:cn-hangzhou:vm:running-delete-block",
		"type":      "VM",
		"name":      "running-vm",
		"cloud":     "aliyun",
		"region":    "cn-hangzhou",
		"env":       "prod",
		"owner":     "platform",
		"source":    "CloudAPI",
		"lifecycle": "active",
		"attributes": map[string]any{
			"status": "RUNNING",
		},
	})

	deleteRec := sendJSONRequest(t, router, http.MethodDelete, fmt.Sprintf("/api/v1/cmdb/resources/%d", vm.ID), adminToken, nil)
	assertErrorResponse(t, deleteRec, http.StatusBadRequest, 3013, "resource is running, stop it before deletion")
}

func createCMDBResourceViaAPI(t *testing.T, router *gin.Engine, token string, payload map[string]any) models.ResourceItem {
	t.Helper()
	rec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/cmdb/resources", token, payload)
	resp := assertOKResponse(t, rec)
	var data cmdbResourceCreateData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("unmarshal cmdb resource create failed: %v", err)
	}
	if data.Resource.ID == 0 {
		t.Fatalf("expected created resource id > 0")
	}
	return data.Resource
}

func createCMDBRelationViaAPI(t *testing.T, router *gin.Engine, token string, payload map[string]any) models.ResourceRelation {
	t.Helper()
	rec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/cmdb/relations", token, payload)
	resp := assertOKResponse(t, rec)
	var data struct {
		Action   string                  `json:"action"`
		Relation models.ResourceRelation `json:"relation"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("unmarshal cmdb relation create failed: %v", err)
	}
	if data.Relation.ID == 0 {
		t.Fatalf("expected relation id > 0")
	}
	return data.Relation
}
