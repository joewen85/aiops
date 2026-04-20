package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"devops-system/backend/internal/cloud"
	appErr "devops-system/backend/internal/errors"
	"devops-system/backend/internal/models"
	"devops-system/backend/internal/pagination"
	"devops-system/backend/internal/response"
)

func (h *Handler) GetCloudAsset(c *gin.Context) { getByID[models.CloudAsset](c, h.DB) }

func (h *Handler) ListCloudAssets(c *gin.Context) {
	page := pagination.Parse(c)
	query := h.DB.Model(&models.CloudAsset{})

	if provider := strings.TrimSpace(c.Query("provider")); provider != "" {
		query = query.Where("provider = ?", provider)
	}
	if accountIDRaw := strings.TrimSpace(c.Query("accountId")); accountIDRaw != "" {
		accountID64, err := strconv.ParseUint(accountIDRaw, 10, 64)
		if err != nil {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "invalid accountId"))
			return
		}
		query = query.Where("account_id = ?", uint(accountID64))
	}
	if region := strings.TrimSpace(c.Query("region")); region != "" {
		query = query.Where("region = ?", region)
	}
	if assetType := strings.TrimSpace(c.Query("type")); assetType != "" {
		query = query.Where("type = ?", cloud.NormalizeBaseResourceType(assetType))
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if source := strings.TrimSpace(c.Query("source")); source != "" {
		query = query.Where("source = ?", normalizeCloudAssetSource(source))
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where("name LIKE ? OR resource_id LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	var (
		items []models.CloudAsset
		total int64
	)
	if err := query.Count(&total).Error; err != nil {
		response.Internal(c, err)
		return
	}
	if err := query.Order("id desc").Limit(page.PageSize).Offset(pagination.Offset(page)).Find(&items).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.List(c, items, total, page.Page, page.PageSize)
}

func (h *Handler) ListCloudAccountAssets(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var account models.CloudAccount
	if err := h.DB.First(&account, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}

	page := pagination.Parse(c)
	query := h.DB.Model(&models.CloudAsset{}).Where("account_id = ?", id)
	if region := strings.TrimSpace(c.Query("region")); region != "" {
		query = query.Where("region = ?", region)
	}
	if assetType := strings.TrimSpace(c.Query("type")); assetType != "" {
		query = query.Where("type = ?", cloud.NormalizeBaseResourceType(assetType))
	}

	var (
		items []models.CloudAsset
		total int64
	)
	if err := query.Count(&total).Error; err != nil {
		response.Internal(c, err)
		return
	}
	if err := query.Order("id desc").Limit(page.PageSize).Offset(pagination.Offset(page)).Find(&items).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.List(c, items, total, page.Page, page.PageSize)
}

func (h *Handler) CreateCloudAsset(c *gin.Context) {
	var req struct {
		Provider   string                 `json:"provider" binding:"required"`
		AccountID  uint                   `json:"accountId"`
		Region     string                 `json:"region"`
		Type       string                 `json:"type" binding:"required"`
		ResourceID string                 `json:"resourceId" binding:"required"`
		Name       string                 `json:"name"`
		Status     string                 `json:"status"`
		Source     string                 `json:"source"`
		Tags       map[string]interface{} `json:"tags"`
		Metadata   map[string]interface{} `json:"metadata"`
		ExpiresAt  string                 `json:"expiresAt"`
	}
	if !bindJSON(c, &req) {
		return
	}
	asset, err := buildManualCloudAsset(req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, err.Error()))
		return
	}
	saved, action, err := h.upsertCloudAsset(asset)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{"action": action, "asset": saved})
}

func (h *Handler) UpdateCloudAsset(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var existing models.CloudAsset
	if err := h.DB.First(&existing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}

	var req map[string]interface{}
	if !bindJSON(c, &req) {
		return
	}

	updates := map[string]interface{}{}
	if providerRaw, ok := req["provider"]; ok {
		provider := strings.TrimSpace(fmt.Sprintf("%v", providerRaw))
		if provider == "" {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "provider cannot be empty"))
			return
		}
		updates["provider"] = provider
	}
	if accountIDRaw, ok := req["accountId"]; ok {
		accountID, err := parseCloudAccountID(accountIDRaw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "invalid accountId"))
			return
		}
		updates["account_id"] = accountID
	}
	if regionRaw, ok := req["region"]; ok {
		updates["region"] = defaultString(strings.TrimSpace(fmt.Sprintf("%v", regionRaw)), "global")
	}
	if typeRaw, ok := req["type"]; ok {
		assetType := cloud.NormalizeBaseResourceType(fmt.Sprintf("%v", typeRaw))
		updates["type"] = assetType
	}
	if resourceIDRaw, ok := req["resourceId"]; ok {
		resourceID := strings.TrimSpace(fmt.Sprintf("%v", resourceIDRaw))
		if resourceID == "" {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "resourceId cannot be empty"))
			return
		}
		updates["resource_id"] = resourceID
	}
	if nameRaw, ok := req["name"]; ok {
		updates["name"] = strings.TrimSpace(fmt.Sprintf("%v", nameRaw))
	}
	if statusRaw, ok := req["status"]; ok {
		updates["status"] = defaultString(strings.TrimSpace(fmt.Sprintf("%v", statusRaw)), "unknown")
	}
	if sourceRaw, ok := req["source"]; ok {
		updates["source"] = normalizeCloudAssetSource(fmt.Sprintf("%v", sourceRaw))
	}
	if tagsRaw, ok := req["tags"]; ok {
		tags, err := castToJSONMap(tagsRaw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "tags must be object"))
			return
		}
		updates["tags"] = tags
	}
	if metadataRaw, ok := req["metadata"]; ok {
		metadata, err := castToJSONMap(metadataRaw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "metadata must be object"))
			return
		}
		updates["metadata"] = metadata
	}
	if expiresAtRaw, ok := req["expiresAt"]; ok {
		expiresAt, err := parseCloudTimeRaw(expiresAtRaw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "expiresAt must be RFC3339"))
			return
		}
		updates["expires_at"] = expiresAt
	}
	if len(updates) == 0 {
		response.Success(c, existing)
		return
	}
	if err := h.DB.Model(&models.CloudAsset{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		response.Internal(c, err)
		return
	}
	getByID[models.CloudAsset](c, h.DB)
}

func (h *Handler) DeleteCloudAsset(c *gin.Context) { deleteByModel[models.CloudAsset](c, h.DB) }

func (h *Handler) syncCloudAssets(account models.CloudAccount, assets []cloud.Asset, source string) ([]models.CloudAsset, datatypes.JSONMap, error) {
	savedAssets := make([]models.CloudAsset, 0, len(assets))
	summary := datatypes.JSONMap{
		"created": 0,
		"updated": 0,
		"failed":  0,
	}
	var runErrs []string
	for _, item := range assets {
		normalized := buildCloudAssetFromProviderAsset(account, item, source)
		saved, action, err := h.upsertCloudAsset(normalized)
		if err != nil {
			runErrs = append(runErrs, err.Error())
			summary["failed"] = asInt(summary["failed"]) + 1
			continue
		}
		summary[action] = asInt(summary[action]) + 1
		savedAssets = append(savedAssets, saved)
	}
	if len(runErrs) > 0 {
		return savedAssets, summary, errors.New(strings.Join(runErrs, "; "))
	}
	return savedAssets, summary, nil
}

func (h *Handler) upsertCloudAsset(input models.CloudAsset) (models.CloudAsset, string, error) {
	input = normalizeCloudAsset(input)
	var existing models.CloudAsset
	query := h.DB.Where("provider = ? AND account_id = ? AND region = ? AND type = ? AND resource_id = ?",
		input.Provider, input.AccountID, input.Region, input.Type, input.ResourceID).
		Limit(1).
		Find(&existing)
	if query.Error != nil {
		return models.CloudAsset{}, "", query.Error
	}
	if query.RowsAffected == 0 {
		if err := h.DB.Create(&input).Error; err != nil {
			return models.CloudAsset{}, "", err
		}
		return input, "created", nil
	}
	updates := map[string]interface{}{
		"name":           defaultString(input.Name, existing.Name),
		"status":         defaultString(input.Status, existing.Status),
		"source":         normalizeCloudAssetSource(defaultString(input.Source, existing.Source)),
		"tags":           mergeJSONMap(existing.Tags, input.Tags, true),
		"metadata":       mergeJSONMap(existing.Metadata, input.Metadata, true),
		"last_synced_at": input.LastSyncedAt,
		"expires_at":     input.ExpiresAt,
	}
	if err := h.DB.Model(&models.CloudAsset{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
		return models.CloudAsset{}, "", err
	}
	var saved models.CloudAsset
	if err := h.DB.First(&saved, existing.ID).Error; err != nil {
		return models.CloudAsset{}, "", err
	}
	return saved, "updated", nil
}

func normalizeCloudAsset(input models.CloudAsset) models.CloudAsset {
	input.Provider = strings.TrimSpace(strings.ToLower(input.Provider))
	input.Region = defaultString(strings.TrimSpace(strings.ToLower(input.Region)), "global")
	input.Type = cloud.NormalizeBaseResourceType(input.Type)
	input.ResourceID = strings.TrimSpace(input.ResourceID)
	input.Name = defaultString(strings.TrimSpace(input.Name), input.ResourceID)
	input.Status = defaultString(strings.TrimSpace(strings.ToLower(input.Status)), "unknown")
	input.Source = normalizeCloudAssetSource(defaultString(input.Source, "Manual"))
	if input.LastSyncedAt == nil {
		now := time.Now()
		input.LastSyncedAt = &now
	}
	if input.Tags == nil {
		input.Tags = datatypes.JSONMap{}
	}
	if input.Metadata == nil {
		input.Metadata = datatypes.JSONMap{}
	}
	return input
}

func normalizeCloudAssetSource(source string) string {
	normalized := strings.TrimSpace(strings.ToLower(source))
	switch normalized {
	case "manual":
		return "Manual"
	case "cloudapi", "cloud_api", "cloud":
		return "CloudAPI"
	default:
		if normalized == "" {
			return "Manual"
		}
		return source
	}
}

func buildCloudAssetFromProviderAsset(account models.CloudAccount, asset cloud.Asset, source string) models.CloudAsset {
	now := time.Now()
	provider := defaultString(asset.Provider, account.Provider)
	region := defaultString(asset.Region, account.Region)
	expiresAt, _ := parseCloudTimeString(readMapString(asset.Metadata, "expiresAt", "expireAt", "expireTime", "expiredAt", "expirationTime"))

	return normalizeCloudAsset(models.CloudAsset{
		Provider:     provider,
		AccountID:    account.ID,
		Region:       region,
		Type:         cloud.NormalizeBaseResourceType(asset.Type),
		ResourceID:   asset.ID,
		Name:         defaultString(asset.Name, asset.ID),
		Status:       defaultString(readMapString(asset.Metadata, "status", "state"), "unknown"),
		Source:       source,
		Tags:         readMapJSONMap(asset.Metadata, "tags", "labels"),
		Metadata:     castMapToJSONMap(asset.Metadata),
		LastSyncedAt: &now,
		ExpiresAt:    expiresAt,
	})
}

func buildManualCloudAsset(req struct {
	Provider   string                 `json:"provider" binding:"required"`
	AccountID  uint                   `json:"accountId"`
	Region     string                 `json:"region"`
	Type       string                 `json:"type" binding:"required"`
	ResourceID string                 `json:"resourceId" binding:"required"`
	Name       string                 `json:"name"`
	Status     string                 `json:"status"`
	Source     string                 `json:"source"`
	Tags       map[string]interface{} `json:"tags"`
	Metadata   map[string]interface{} `json:"metadata"`
	ExpiresAt  string                 `json:"expiresAt"`
}) (models.CloudAsset, error) {
	expiresAt, err := parseCloudTimeString(req.ExpiresAt)
	if err != nil {
		return models.CloudAsset{}, err
	}
	return normalizeCloudAsset(models.CloudAsset{
		Provider:   req.Provider,
		AccountID:  req.AccountID,
		Region:     req.Region,
		Type:       req.Type,
		ResourceID: req.ResourceID,
		Name:       req.Name,
		Status:     req.Status,
		Source:     req.Source,
		Tags:       castMapToJSONMap(req.Tags),
		Metadata:   castMapToJSONMap(req.Metadata),
		ExpiresAt:  expiresAt,
	}), nil
}

func parseCloudTimeString(value string) (*time.Time, error) {
	text := strings.TrimSpace(value)
	if text == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseCloudTimeRaw(raw interface{}) (*time.Time, error) {
	text := strings.TrimSpace(fmt.Sprintf("%v", raw))
	if text == "" || text == "<nil>" {
		return nil, nil
	}
	return parseCloudTimeString(text)
}

func parseCloudAccountID(raw interface{}) (uint, error) {
	switch value := raw.(type) {
	case float64:
		if value < 0 {
			return 0, fmt.Errorf("invalid account id")
		}
		return uint(value), nil
	case int:
		if value < 0 {
			return 0, fmt.Errorf("invalid account id")
		}
		return uint(value), nil
	case int64:
		if value < 0 {
			return 0, fmt.Errorf("invalid account id")
		}
		return uint(value), nil
	case uint:
		return value, nil
	case uint64:
		return uint(value), nil
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", raw))
		if text == "" {
			return 0, nil
		}
		parsed, err := strconv.ParseUint(text, 10, 64)
		if err != nil {
			return 0, err
		}
		return uint(parsed), nil
	}
}

func castToJSONMap(raw interface{}) (datatypes.JSONMap, error) {
	switch value := raw.(type) {
	case map[string]interface{}:
		return castMapToJSONMap(value), nil
	case datatypes.JSONMap:
		return value, nil
	case nil:
		return datatypes.JSONMap{}, nil
	default:
		return nil, fmt.Errorf("invalid map")
	}
}

func castMapToJSONMap(raw map[string]interface{}) datatypes.JSONMap {
	result := datatypes.JSONMap{}
	for key, value := range raw {
		result[key] = value
	}
	return result
}

func readMapJSONMap(values map[string]interface{}, keys ...string) datatypes.JSONMap {
	for _, key := range keys {
		raw, ok := values[key]
		if !ok {
			continue
		}
		switch val := raw.(type) {
		case map[string]interface{}:
			return castMapToJSONMap(val)
		}
	}
	return datatypes.JSONMap{}
}

func (h *Handler) collectCloudProviderAssets(provider cloud.Provider, cred cloud.Credentials) ([]cloud.Asset, error) {
	if h.CloudCollector != nil {
		return h.CloudCollector.CollectBaseResources(provider, cred)
	}
	return provider.SyncAssets(cred)
}
