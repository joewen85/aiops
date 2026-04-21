package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"devops-system/backend/internal/models"
)

func TestCloudAssetCRUDIntegration(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	createAccountRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/cloud/accounts", adminToken, map[string]any{
		"provider":  "aws",
		"name":      "aws-stage",
		"accessKey": "ak",
		"secretKey": "sk",
		"region":    "ap-southeast-1",
	})
	createAccountResp := assertOKResponse(t, createAccountRec)
	var account models.CloudAccount
	if err := json.Unmarshal(createAccountResp.Data, &account); err != nil {
		t.Fatalf("unmarshal cloud account failed: %v", err)
	}
	if account.ID == 0 {
		t.Fatalf("expected cloud account id > 0")
	}

	syncRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/cloud/accounts/%d/sync", account.ID), adminToken, nil)
	syncResp := assertOKResponse(t, syncRec)
	var syncData map[string]json.RawMessage
	if err := json.Unmarshal(syncResp.Data, &syncData); err != nil {
		t.Fatalf("unmarshal sync response failed: %v", err)
	}
	if _, ok := syncData["job"]; !ok {
		t.Fatalf("expected sync response contains job")
	}
	if _, ok := syncData["assets"]; ok {
		t.Fatalf("expected default sync response does not include provider assets")
	}
	if _, ok := syncData["cloudAssetItems"]; ok {
		t.Fatalf("expected default sync response does not include cloud asset items")
	}
	if _, ok := syncData["cmdbResources"]; ok {
		t.Fatalf("expected default sync response does not include cmdb resources")
	}
	if _, ok := syncData["providerAssetCount"]; !ok {
		t.Fatalf("expected sync response contains providerAssetCount")
	}
	if _, ok := syncData["cloudAssetCount"]; !ok {
		t.Fatalf("expected sync response contains cloudAssetCount")
	}
	if _, ok := syncData["cmdbAssetCount"]; !ok {
		t.Fatalf("expected sync response contains cmdbAssetCount")
	}

	syncVerboseRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/cloud/accounts/%d/sync?verbose=1", account.ID), adminToken, nil)
	syncVerboseResp := assertOKResponse(t, syncVerboseRec)
	var syncVerboseData map[string]json.RawMessage
	if err := json.Unmarshal(syncVerboseResp.Data, &syncVerboseData); err != nil {
		t.Fatalf("unmarshal verbose sync response failed: %v", err)
	}
	if _, ok := syncVerboseData["assets"]; !ok {
		t.Fatalf("expected verbose sync response contains provider assets")
	}
	if _, ok := syncVerboseData["cloudAssetItems"]; !ok {
		t.Fatalf("expected verbose sync response contains cloud asset items")
	}
	if _, ok := syncVerboseData["cmdbResources"]; !ok {
		t.Fatalf("expected verbose sync response contains cmdb resources")
	}

	listAssetsRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/cloud/assets?page=1&pageSize=50", adminToken, nil)
	listAssetsResp := assertOKResponse(t, listAssetsRec)
	var cloudAssetsPage listPayload[models.CloudAsset]
	if err := json.Unmarshal(listAssetsResp.Data, &cloudAssetsPage); err != nil {
		t.Fatalf("unmarshal cloud assets page failed: %v", err)
	}
	if cloudAssetsPage.Total < 1 {
		t.Fatalf("expected synced cloud assets not empty")
	}

	accountAssetsRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/cloud/accounts/%d/assets?page=1&pageSize=50", account.ID), adminToken, nil)
	accountAssetsResp := assertOKResponse(t, accountAssetsRec)
	var accountAssetsPage listPayload[models.CloudAsset]
	if err := json.Unmarshal(accountAssetsResp.Data, &accountAssetsPage); err != nil {
		t.Fatalf("unmarshal cloud account assets page failed: %v", err)
	}
	if accountAssetsPage.Total < 1 {
		t.Fatalf("expected cloud account assets not empty")
	}

	createAssetRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/cloud/assets", adminToken, map[string]any{
		"provider":   "aws",
		"accountId":  account.ID,
		"region":     "ap-southeast-1",
		"type":       "ObjectStorage",
		"resourceId": "manual-oss-001",
		"name":       "manual-oss",
		"status":     "active",
		"source":     "Manual",
		"tags": map[string]any{
			"env": "stage",
		},
		"metadata": map[string]any{
			"class": "standard",
		},
	})
	createAssetResp := assertOKResponse(t, createAssetRec)
	var createAssetData struct {
		Action string            `json:"action"`
		Asset  models.CloudAsset `json:"asset"`
	}
	if err := json.Unmarshal(createAssetResp.Data, &createAssetData); err != nil {
		t.Fatalf("unmarshal create cloud asset failed: %v", err)
	}
	if createAssetData.Asset.ID == 0 {
		t.Fatalf("expected created cloud asset id > 0")
	}

	getAssetRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/cloud/assets/%d", createAssetData.Asset.ID), adminToken, nil)
	getAssetResp := assertOKResponse(t, getAssetRec)
	var cloudAsset models.CloudAsset
	if err := json.Unmarshal(getAssetResp.Data, &cloudAsset); err != nil {
		t.Fatalf("unmarshal cloud asset detail failed: %v", err)
	}
	if cloudAsset.ResourceID != "manual-oss-001" {
		t.Fatalf("unexpected resource id=%s", cloudAsset.ResourceID)
	}

	updateAssetRec := sendJSONRequest(t, router, http.MethodPut, fmt.Sprintf("/api/v1/cloud/assets/%d", createAssetData.Asset.ID), adminToken, map[string]any{
		"status": "inactive",
		"name":   "manual-oss-updated",
	})
	updateAssetResp := assertOKResponse(t, updateAssetRec)
	var updatedAsset models.CloudAsset
	if err := json.Unmarshal(updateAssetResp.Data, &updatedAsset); err != nil {
		t.Fatalf("unmarshal updated cloud asset failed: %v", err)
	}
	if updatedAsset.Status != "inactive" {
		t.Fatalf("expected updated status inactive, got=%s", updatedAsset.Status)
	}

	deleteAssetRec := sendJSONRequest(t, router, http.MethodDelete, fmt.Sprintf("/api/v1/cloud/assets/%d", createAssetData.Asset.ID), adminToken, nil)
	_ = assertOKResponse(t, deleteAssetRec)

	getDeletedRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/cloud/assets/%d", createAssetData.Asset.ID), adminToken, nil)
	if getDeletedRec.Code != http.StatusNotFound {
		t.Fatalf("expected deleted asset returns 404, got=%d body=%s", getDeletedRec.Code, getDeletedRec.Body.String())
	}
}

func TestCloudAccountListFilterIntegration(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	createAccount := func(provider, name, accessKey, region string) models.CloudAccount {
		rec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/cloud/accounts", adminToken, map[string]any{
			"provider":  provider,
			"name":      name,
			"accessKey": accessKey,
			"secretKey": "sk",
			"region":    region,
		})
		resp := assertOKResponse(t, rec)
		var account models.CloudAccount
		if err := json.Unmarshal(resp.Data, &account); err != nil {
			t.Fatalf("unmarshal cloud account failed: %v", err)
		}
		return account
	}

	accountA := createAccount("aws", "aws-prod", "ak-prod", "ap-southeast-1")
	_ = createAccount("aliyun", "ali-dev", "ak-dev", "cn-hangzhou")
	accountC := createAccount("aws", "aws-qa", "ak-qa", "us-east-1")

	verifyRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/cloud/accounts/%d/verify", accountC.ID), adminToken, nil)
	_ = assertOKResponse(t, verifyRec)

	assertTotal := func(path string, expected int64) {
		rec := sendJSONRequest(t, router, http.MethodGet, path, adminToken, nil)
		resp := assertOKResponse(t, rec)
		var page listPayload[models.CloudAccount]
		if err := json.Unmarshal(resp.Data, &page); err != nil {
			t.Fatalf("unmarshal cloud account page failed: %v", err)
		}
		if page.Total != expected {
			t.Fatalf("path=%s expected total=%d got=%d", path, expected, page.Total)
		}
	}

	assertTotal(tailPath("/api/v1/cloud/accounts", "provider=aws&page=1&pageSize=50"), 2)
	assertTotal(tailPath("/api/v1/cloud/accounts", "region=ap-southeast-1&page=1&pageSize=50"), 1)
	assertTotal(tailPath("/api/v1/cloud/accounts", "keyword=prod&page=1&pageSize=50"), 1)
	assertTotal(tailPath("/api/v1/cloud/accounts", "keyword=ak-dev&page=1&pageSize=50"), 1)
	assertTotal(tailPath("/api/v1/cloud/accounts", "verified=true&page=1&pageSize=50"), 1)
	assertTotal(tailPath("/api/v1/cloud/accounts", "verified=false&page=1&pageSize=50"), 2)
	assertTotal(tailPath("/api/v1/cloud/accounts", "provider=aws&verified=true&page=1&pageSize=50"), 1)

	accountFilterRec := sendJSONRequest(
		t,
		router,
		http.MethodGet,
		tailPath("/api/v1/cloud/accounts", "provider=aws&region=ap-southeast-1&keyword=prod&page=1&pageSize=50"),
		adminToken,
		nil,
	)
	accountFilterResp := assertOKResponse(t, accountFilterRec)
	var accountFilterPage listPayload[models.CloudAccount]
	if err := json.Unmarshal(accountFilterResp.Data, &accountFilterPage); err != nil {
		t.Fatalf("unmarshal filtered account page failed: %v", err)
	}
	if accountFilterPage.Total != 1 {
		t.Fatalf("expected combined filter result total=1 got=%d", accountFilterPage.Total)
	}
	if len(accountFilterPage.List) != 1 || accountFilterPage.List[0].ID != accountA.ID {
		t.Fatalf("expected matched account id=%d got=%+v", accountA.ID, accountFilterPage.List)
	}
}

func TestCloudAccountSecurityAndSyncRobustnessIntegration(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	const (
		rawAccessKey = "AKIA-RAW-123456"
		rawSecretKey = "SK-RAW-987654"
	)
	createRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/cloud/accounts", adminToken, map[string]any{
		"provider":  "aws",
		"name":      "aws-security",
		"accessKey": rawAccessKey,
		"secretKey": rawSecretKey,
		"region":    "ap-southeast-1",
	})
	createResp := assertOKResponse(t, createRec)
	var account models.CloudAccount
	if err := json.Unmarshal(createResp.Data, &account); err != nil {
		t.Fatalf("unmarshal cloud account failed: %v", err)
	}
	if account.ID == 0 {
		t.Fatalf("expected cloud account id > 0")
	}
	if strings.Contains(account.AccessKey, rawAccessKey) || strings.Contains(account.SecretKey, rawSecretKey) {
		t.Fatalf("expected masked credentials in response, got accessKey=%s secretKey=%s", account.AccessKey, account.SecretKey)
	}

	getRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/cloud/accounts/%d", account.ID), adminToken, nil)
	getResp := assertOKResponse(t, getRec)
	var getAccount models.CloudAccount
	if err := json.Unmarshal(getResp.Data, &getAccount); err != nil {
		t.Fatalf("unmarshal cloud account detail failed: %v", err)
	}
	if strings.Contains(getAccount.AccessKey, rawAccessKey) || strings.Contains(getAccount.SecretKey, rawSecretKey) {
		t.Fatalf("expected masked credentials in detail response")
	}

	// patch non-whitelisted field should not elevate verification status.
	patchRec := sendJSONRequest(t, router, http.MethodPut, fmt.Sprintf("/api/v1/cloud/accounts/%d", account.ID), adminToken, map[string]any{
		"isVerified": true,
	})
	patchResp := assertOKResponse(t, patchRec)
	var patched models.CloudAccount
	if err := json.Unmarshal(patchResp.Data, &patched); err != nil {
		t.Fatalf("unmarshal patched cloud account failed: %v", err)
	}
	if patched.IsVerified {
		t.Fatalf("expected isVerified remains false when patching protected field")
	}

	// update without AK/SK should keep existing credentials and still verify/sync.
	updateRec := sendJSONRequest(t, router, http.MethodPut, fmt.Sprintf("/api/v1/cloud/accounts/%d", account.ID), adminToken, map[string]any{
		"name": "aws-security-updated",
	})
	_ = assertOKResponse(t, updateRec)

	verifyRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/cloud/accounts/%d/verify", account.ID), adminToken, nil)
	_ = assertOKResponse(t, verifyRec)

	syncRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/cloud/accounts/%d/sync", account.ID), adminToken, nil)
	_ = assertOKResponse(t, syncRec)

	verify404 := sendJSONRequest(t, router, http.MethodPost, "/api/v1/cloud/accounts/999999/verify", adminToken, nil)
	if verify404.Code != http.StatusNotFound {
		t.Fatalf("expected verify missing account returns 404, got=%d body=%s", verify404.Code, verify404.Body.String())
	}
	sync404 := sendJSONRequest(t, router, http.MethodPost, "/api/v1/cloud/accounts/999999/sync", adminToken, nil)
	if sync404.Code != http.StatusNotFound {
		t.Fatalf("expected sync missing account returns 404, got=%d body=%s", sync404.Code, sync404.Body.String())
	}
}

func TestCloudAccountTencentUpdateFromLegacyInvalidCredentialIntegration(t *testing.T) {
	router, database, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	legacy := models.CloudAccount{
		Provider:   "tencent",
		Name:       "tencent-legacy",
		AccessKey:  "legacy-invalid-ak",
		SecretKey:  "legacy-invalid-sk",
		Region:     "ap-guangzhou",
		IsVerified: false,
	}
	if err := database.Create(&legacy).Error; err != nil {
		t.Fatalf("create legacy cloud account failed: %v", err)
	}

	updateRec := sendJSONRequest(
		t,
		router,
		http.MethodPut,
		fmt.Sprintf("/api/v1/cloud/accounts/%d", legacy.ID),
		adminToken,
		map[string]any{
			"accessKey": "AKIDNEWVALID1234567890",
			"secretKey": "new-valid-secret-key",
		},
	)
	updateResp := assertOKResponse(t, updateRec)
	var updated models.CloudAccount
	if err := json.Unmarshal(updateResp.Data, &updated); err != nil {
		t.Fatalf("unmarshal updated cloud account failed: %v", err)
	}
	if updated.ID != legacy.ID {
		t.Fatalf("expected updated account id=%d got=%d", legacy.ID, updated.ID)
	}
}

func tailPath(path, query string) string {
	if query == "" {
		return path
	}
	return fmt.Sprintf("%s?%s", path, query)
}
