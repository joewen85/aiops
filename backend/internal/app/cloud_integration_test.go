package app

import (
	"encoding/json"
	"fmt"
	"net/http"
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

func tailPath(path, query string) string {
	if query == "" {
		return path
	}
	return fmt.Sprintf("%s?%s", path, query)
}
