package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	aliyunErr "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	aliyunECS "github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/gin-gonic/gin"
	tencentCommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tencentErr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	tencentProfile "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tencentCVM "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"gorm.io/gorm"

	"devops-system/backend/internal/cloud"
	appErr "devops-system/backend/internal/errors"
	"devops-system/backend/internal/models"
	"devops-system/backend/internal/response"
)

var (
	errCMDBVMActionUnsupportedSource = errors.New("resource source is not CloudAPI, action is unsupported")
	errCMDBVMActionNotSynced         = errors.New("resource is not linked to synced cloud asset")
	errCMDBVMActionAssetMismatch     = errors.New("resource cloud asset mapping mismatch")
)

func (h *Handler) RestartCMDBResource(c *gin.Context) {
	h.handleCMDBVMAction(c, "restart")
}

func (h *Handler) StopCMDBResource(c *gin.Context) {
	h.handleCMDBVMAction(c, "stop")
}

func (h *Handler) handleCMDBVMAction(c *gin.Context, action string) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var resource models.ResourceItem
	if err := h.DB.First(&resource, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	if !isVMResourceType(resource.Type) {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "resource type is not VM, action is unsupported"))
		return
	}

	accountID, instanceID, region, provider, extractErr := h.extractCMDBVMActionContext(resource)
	if extractErr != nil {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, extractErr.Error()))
		return
	}

	var account models.CloudAccount
	if err := h.DB.First(&account, accountID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusBadRequest, appErr.New(4007, "cloud account referenced by resource not found"))
			return
		}
		response.Internal(c, err)
		return
	}

	resolvedProvider := strings.ToLower(strings.TrimSpace(account.Provider))
	if provider != "" && resolvedProvider != provider {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "resource cloud provider mismatches account provider"))
		return
	}
	if err := h.ensureCMDBVMActionAuthorized(resource, account.ID, instanceID, resolvedProvider); err != nil {
		switch {
		case errors.Is(err, errCMDBVMActionUnsupportedSource),
			errors.Is(err, errCMDBVMActionNotSynced),
			errors.Is(err, errCMDBVMActionAssetMismatch):
			response.Error(c, http.StatusBadRequest, appErr.New(3001, err.Error()))
		default:
			response.Internal(c, err)
		}
		return
	}
	if region == "" || strings.EqualFold(region, "global") {
		region = strings.TrimSpace(account.Region)
	}
	if region == "" || strings.EqualFold(region, "global") {
		switch resolvedProvider {
		case "aliyun":
			region = defaultString(strings.TrimSpace(h.Config.AliyunDefaultRegion), "cn-hangzhou")
		case "tencent", "tencentcloud":
			region = defaultString(strings.TrimSpace(h.Config.TencentDefaultRegion), "ap-guangzhou")
		}
	}
	if region == "" || strings.EqualFold(region, "global") {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "resource region is empty, cannot execute VM action"))
		return
	}

	cred, credErr := h.cloudAccountCredentials(&account)
	if credErr != nil {
		response.Internal(c, credErr)
		return
	}

	if err := h.executeVMAction(resolvedProvider, cred, region, instanceID, action); err != nil {
		response.Error(c, http.StatusBadRequest, appErr.New(4006, h.cloudProviderExternalError("vm action request failed", err)))
		return
	}

	response.Success(c, gin.H{
		"id":         resource.ID,
		"ciId":       resource.CIID,
		"action":     action,
		"status":     "accepted",
		"provider":   resolvedProvider,
		"accountId":  account.ID,
		"instanceId": instanceID,
		"region":     region,
	})
}

func (h *Handler) extractCMDBVMActionContext(resource models.ResourceItem) (uint, string, string, string, error) {
	accountIDRaw := readStringAttr(resource.Attributes, "accountId", "account_id", "account")
	if strings.TrimSpace(accountIDRaw) == "" {
		return 0, "", "", "", fmt.Errorf("resource accountId is empty")
	}
	accountID, err := parseCloudAccountID(accountIDRaw)
	if err != nil || accountID == 0 {
		return 0, "", "", "", fmt.Errorf("invalid resource accountId")
	}
	instanceID := strings.TrimSpace(readStringAttr(resource.Attributes, "instanceId", "instance_id", "assetId", "asset_id"))
	if instanceID == "" {
		return 0, "", "", "", fmt.Errorf("resource instanceId is empty")
	}
	region := strings.TrimSpace(defaultString(resource.Region, readStringAttr(resource.Attributes, "region", "regionId", "region_id")))
	provider := strings.ToLower(strings.TrimSpace(defaultString(resource.Cloud, readStringAttr(resource.Attributes, "provider", "cloud"))))
	return accountID, instanceID, region, provider, nil
}

func (h *Handler) ensureCMDBVMActionAuthorized(resource models.ResourceItem, accountID uint, instanceID string, provider string) error {
	if normalizeCMDBSource(resource.Source) != "CloudAPI" {
		return errCMDBVMActionUnsupportedSource
	}

	var evidenceCount int64
	if err := h.DB.Model(&models.ResourceEvidence{}).
		Where("ci_id = ? AND source = ? AND raw_id = ?", resource.CIID, "CloudAPI", instanceID).
		Count(&evidenceCount).Error; err != nil {
		return err
	}
	if evidenceCount == 0 {
		return errCMDBVMActionNotSynced
	}

	query := h.DB.Model(&models.CloudAsset{}).
		Where("account_id = ? AND resource_id = ? AND type = ?", accountID, instanceID, cloud.ResourceTypeCloudServer)
	if providers := cloudProviderQueryAliases(provider); len(providers) > 0 {
		query = query.Where("lower(provider) IN ?", providers)
	}
	var assetCount int64
	if err := query.Count(&assetCount).Error; err != nil {
		return err
	}
	if assetCount == 0 {
		return errCMDBVMActionAssetMismatch
	}
	return nil
}

func cloudProviderQueryAliases(provider string) []string {
	normalized := strings.ToLower(strings.TrimSpace(provider))
	switch normalized {
	case "":
		return nil
	case "tencent", "tencentcloud":
		return []string{"tencent", "tencentcloud"}
	default:
		return []string{normalized}
	}
}

func (h *Handler) executeVMAction(provider string, cred cloud.Credentials, region string, instanceID string, action string) error {
	switch provider {
	case "aliyun":
		return h.executeAliyunVMAction(cred, region, instanceID, action)
	case "tencent", "tencentcloud":
		return h.executeTencentVMAction(cred, region, instanceID, action)
	default:
		return fmt.Errorf("provider=%s does not support VM action yet", provider)
	}
}

func (h *Handler) executeAliyunVMAction(cred cloud.Credentials, region string, instanceID string, action string) error {
	client, err := aliyunECS.NewClientWithOptions(
		region,
		sdk.NewConfig().WithScheme("HTTPS").WithTimeout(time.Duration(maxInt(h.Config.AliyunSDKTimeoutSeconds, 10))*time.Second),
		credentials.NewAccessKeyCredential(strings.TrimSpace(cred.AccessKey), strings.TrimSpace(cred.SecretKey)),
	)
	if err != nil {
		return fmt.Errorf("init aliyun ecs client failed: %w", err)
	}
	switch action {
	case "restart":
		req := aliyunECS.CreateRebootInstanceRequest()
		req.InstanceId = instanceID
		req.ForceStop = requests.NewBoolean(false)
		if _, rebootErr := client.RebootInstance(req); rebootErr != nil {
			return wrapCMDBCloudActionError("aliyun", "ecs.RebootInstance", rebootErr)
		}
	case "stop":
		req := aliyunECS.CreateStopInstanceRequest()
		req.InstanceId = instanceID
		req.ForceStop = requests.NewBoolean(false)
		if _, stopErr := client.StopInstance(req); stopErr != nil {
			return wrapCMDBCloudActionError("aliyun", "ecs.StopInstance", stopErr)
		}
	default:
		return fmt.Errorf("unsupported action=%s", action)
	}
	return nil
}

func (h *Handler) executeTencentVMAction(cred cloud.Credentials, region string, instanceID string, action string) error {
	credential := tencentCommon.NewCredential(strings.TrimSpace(cred.AccessKey), strings.TrimSpace(cred.SecretKey))
	clientProfile := tencentProfile.NewClientProfile()
	clientProfile.HttpProfile.ReqTimeout = maxInt(h.Config.TencentSDKTimeoutSeconds, 10)
	client, err := tencentCVM.NewClient(credential, region, clientProfile)
	if err != nil {
		return fmt.Errorf("init tencent cvm client failed: %w", err)
	}
	switch action {
	case "restart":
		req := tencentCVM.NewRebootInstancesRequest()
		req.InstanceIds = []*string{tencentCommon.StringPtr(instanceID)}
		req.StopType = tencentCommon.StringPtr("SOFT_FIRST")
		if _, rebootErr := client.RebootInstances(req); rebootErr != nil {
			return wrapCMDBCloudActionError("tencent", "cvm.RebootInstances", rebootErr)
		}
	case "stop":
		req := tencentCVM.NewStopInstancesRequest()
		req.InstanceIds = []*string{tencentCommon.StringPtr(instanceID)}
		req.StopType = tencentCommon.StringPtr("SOFT_FIRST")
		if _, stopErr := client.StopInstances(req); stopErr != nil {
			return wrapCMDBCloudActionError("tencent", "cvm.StopInstances", stopErr)
		}
	default:
		return fmt.Errorf("unsupported action=%s", action)
	}
	return nil
}

func wrapCMDBCloudActionError(provider string, scene string, err error) error {
	if provider == "aliyun" {
		var sdkServerErr *aliyunErr.ServerError
		if errors.As(err, &sdkServerErr) {
			code := strings.TrimSpace(sdkServerErr.ErrorCode())
			message := strings.TrimSpace(sdkServerErr.Message())
			if code == "" {
				code = "UnknownServerError"
			}
			if message == "" {
				message = strings.TrimSpace(sdkServerErr.Error())
			}
			return fmt.Errorf("%s failed: code=%s message=%s requestId=%s", scene, code, message, strings.TrimSpace(sdkServerErr.RequestId()))
		}
		var sdkClientErr *aliyunErr.ClientError
		if errors.As(err, &sdkClientErr) {
			return fmt.Errorf("%s failed: code=%s message=%s", scene, strings.TrimSpace(sdkClientErr.ErrorCode()), strings.TrimSpace(sdkClientErr.Message()))
		}
	}
	if provider == "tencent" || provider == "tencentcloud" {
		var sdkErr *tencentErr.TencentCloudSDKError
		if errors.As(err, &sdkErr) {
			return fmt.Errorf("%s failed: code=%s message=%s requestId=%s", scene, sdkErr.GetCode(), sdkErr.GetMessage(), sdkErr.GetRequestId())
		}
	}
	return fmt.Errorf("%s failed: %w", scene, err)
}

func isVMResourceType(resourceType string) bool {
	switch strings.ToLower(strings.TrimSpace(resourceType)) {
	case "vm", "compute", "ecs", "ec2", "cloudserver":
		return true
	default:
		return false
	}
}

func maxInt(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
