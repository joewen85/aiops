package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	appErr "devops-system/backend/internal/errors"
	"devops-system/backend/internal/middlewaresvc"
	"devops-system/backend/internal/models"
	"devops-system/backend/internal/pagination"
	"devops-system/backend/internal/response"
)

const middlewareProtocolVersion = "aiops.middleware.v1alpha1"
const middlewareConfirmText = "确认删除资源"

type middlewareActionRequest struct {
	InstanceID       uint                   `json:"instanceId"`
	Type             string                 `json:"type"`
	Action           string                 `json:"action"`
	DryRun           *bool                  `json:"dryRun"`
	ConfirmationText string                 `json:"confirmationText"`
	Params           map[string]interface{} `json:"params"`
}

func (h *Handler) ListMiddlewareInstances(c *gin.Context) {
	page := pagination.Parse(c)
	query := h.DB.Model(&models.MiddlewareInstance{})
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where("name LIKE ? OR endpoint LIKE ? OR owner LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if typ := middlewaresvc.NormalizeType(c.Query("type")); typ != "" {
		query = query.Where("type = ?", typ)
	}
	if env := strings.TrimSpace(c.Query("env")); env != "" {
		query = query.Where("env = ?", env)
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	var items []models.MiddlewareInstance
	var total int64
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

func (h *Handler) GetMiddlewareInstance(c *gin.Context) {
	getByID[models.MiddlewareInstance](c, h.DB)
}

func (h *Handler) CreateMiddlewareInstance(c *gin.Context) {
	var req struct {
		Name       string                 `json:"name" binding:"required"`
		Type       string                 `json:"type" binding:"required"`
		Endpoint   string                 `json:"endpoint" binding:"required"`
		HealthPath string                 `json:"healthPath"`
		Env        string                 `json:"env"`
		Owner      string                 `json:"owner"`
		AuthType   string                 `json:"authType"`
		TLSEnable  bool                   `json:"tlsEnable"`
		Username   string                 `json:"username"`
		Password   string                 `json:"password"`
		Token      string                 `json:"token"`
		Labels     map[string]interface{} `json:"labels"`
		Metadata   map[string]interface{} `json:"metadata"`
	}
	if !bindJSON(c, &req) {
		return
	}
	instance, credential, err := h.buildMiddlewareInstance(req.Name, req.Type, req.Endpoint, req.HealthPath, req.Env, req.Owner, req.AuthType, req.TLSEnable, req.Username, req.Password, req.Token, req.Labels, req.Metadata)
	if err != nil {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, err.Error()))
		return
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&instance).Error; err != nil {
			return err
		}
		credential.InstanceID = instance.ID
		if credential.Username != "" || credential.Password != "" || credential.Token != "" {
			if err := tx.Create(&credential).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, instance)
}

func (h *Handler) UpdateMiddlewareInstance(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var instance models.MiddlewareInstance
	if err := h.DB.First(&instance, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	var req struct {
		Name       *string                `json:"name"`
		Type       *string                `json:"type"`
		Endpoint   *string                `json:"endpoint"`
		HealthPath *string                `json:"healthPath"`
		Env        *string                `json:"env"`
		Owner      *string                `json:"owner"`
		AuthType   *string                `json:"authType"`
		TLSEnable  *bool                  `json:"tlsEnable"`
		Username   *string                `json:"username"`
		Password   *string                `json:"password"`
		Token      *string                `json:"token"`
		Labels     map[string]interface{} `json:"labels"`
		Metadata   map[string]interface{} `json:"metadata"`
	}
	if !bindJSON(c, &req) {
		return
	}
	updates := map[string]interface{}{}
	next := instance
	if req.Name != nil {
		next.Name = strings.TrimSpace(*req.Name)
		updates["name"] = next.Name
	}
	if req.Type != nil {
		next.Type = middlewaresvc.NormalizeType(*req.Type)
		updates["type"] = next.Type
	}
	if req.Endpoint != nil {
		next.Endpoint = strings.TrimSpace(*req.Endpoint)
		updates["endpoint"] = next.Endpoint
		updates["status"] = "unknown"
	}
	if req.HealthPath != nil {
		updates["health_path"] = strings.TrimSpace(*req.HealthPath)
	}
	if req.Env != nil {
		next.Env = defaultString(strings.TrimSpace(*req.Env), "prod")
		updates["env"] = next.Env
	}
	if req.Owner != nil {
		updates["owner"] = strings.TrimSpace(*req.Owner)
	}
	if req.AuthType != nil {
		updates["auth_type"] = defaultString(strings.TrimSpace(*req.AuthType), "password")
	}
	if req.TLSEnable != nil {
		next.TLSEnable = *req.TLSEnable
		updates["tls_enable"] = next.TLSEnable
	}
	if req.Labels != nil {
		updates["labels"] = datatypes.JSONMap(req.Labels)
	}
	if req.Metadata != nil {
		updates["metadata"] = datatypes.JSONMap(req.Metadata)
	}
	if err := validateMiddlewareInstance(next); err != nil {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, err.Error()))
		return
	}
	if len(updates) > 0 {
		if err := h.DB.Model(&models.MiddlewareInstance{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			response.Internal(c, err)
			return
		}
	}
	if req.Username != nil || req.Password != nil || req.Token != nil {
		if err := h.upsertMiddlewareCredential(id, req.Username, req.Password, req.Token); err != nil {
			response.Internal(c, err)
			return
		}
	}
	getByID[models.MiddlewareInstance](c, h.DB)
}

func (h *Handler) DeleteMiddlewareInstance(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		ConfirmationText string `json:"confirmationText" binding:"required"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if strings.TrimSpace(req.ConfirmationText) != middlewareConfirmText {
		response.Error(c, http.StatusBadRequest, appErr.New(3020, "confirmation text is required"))
		return
	}
	var instance models.MiddlewareInstance
	if err := h.DB.First(&instance, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	if strings.EqualFold(instance.Status, "healthy") {
		response.Error(c, http.StatusConflict, appErr.New(4020, "healthy middleware instance cannot be deleted"))
		return
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("instance_id = ?", id).Delete(&models.MiddlewareCredential{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.MiddlewareInstance{}, id).Error
	}); err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *Handler) CheckMiddlewareInstance(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	instance, driver, svcInstance, found := h.middlewareDriverForInstance(c, id)
	if !found {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), middlewaresvc.DefaultTimeout)
	defer cancel()
	result, err := driver.Check(ctx, svcInstance)
	now := time.Now()
	updates := map[string]interface{}{"last_checked_at": &now}
	if err != nil {
		log.Printf("middleware check failed instance_id=%d type=%s err=%v", id, instance.Type, err)
		updates["status"] = "error"
		updates["metadata"] = datatypes.JSONMap{"lastError": "middleware check failed"}
		_ = h.DB.Model(&models.MiddlewareInstance{}).Where("id = ?", id).Updates(updates).Error
		response.Error(c, http.StatusBadRequest, appErr.New(4021, "middleware check failed"))
		return
	}
	updates["status"] = result.Status
	updates["version"] = result.Version
	if err := h.DB.Model(&models.MiddlewareInstance{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{"id": id, "traceId": uuid.NewString(), "result": result})
}

func (h *Handler) CollectMiddlewareMetrics(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	_, driver, svcInstance, found := h.middlewareDriverForInstance(c, id)
	if !found {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), middlewaresvc.DefaultTimeout)
	defer cancel()
	metrics, err := driver.CollectMetrics(ctx, svcInstance)
	if err != nil {
		log.Printf("middleware metrics collect failed instance_id=%d type=%s err=%v", id, svcInstance.Type, err)
		response.Error(c, http.StatusBadRequest, appErr.New(4022, "middleware metrics collect failed"))
		return
	}
	now := time.Now()
	items := make([]models.MiddlewareMetric, 0, len(metrics))
	for _, metric := range metrics {
		items = append(items, models.MiddlewareMetric{
			InstanceID:  id,
			MetricType:  metric.Type,
			Value:       metric.Value,
			Unit:        metric.Unit,
			Data:        datatypes.JSONMap(metric.Data),
			CollectedAt: now,
		})
	}
	if len(items) > 0 {
		if err := h.DB.Create(&items).Error; err != nil {
			response.Internal(c, err)
			return
		}
	}
	response.Success(c, gin.H{"instanceId": id, "metrics": items})
}

func (h *Handler) ListMiddlewareMetrics(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	page := pagination.Parse(c)
	query := h.DB.Model(&models.MiddlewareMetric{}).Where("instance_id = ?", id)
	if metricType := strings.TrimSpace(c.Query("metricType")); metricType != "" {
		query = query.Where("metric_type = ?", metricType)
	}
	var items []models.MiddlewareMetric
	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Internal(c, err)
		return
	}
	if err := query.Order("collected_at desc").Limit(page.PageSize).Offset(pagination.Offset(page)).Find(&items).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.List(c, items, total, page.Page, page.PageSize)
}

func (h *Handler) MiddlewareAIOpsProtocol(c *gin.Context) {
	response.Success(c, middlewareProtocol())
}

func (h *Handler) MiddlewareAction(c *gin.Context) {
	var req middlewareActionRequest
	if !bindJSON(c, &req) {
		return
	}
	if req.InstanceID == 0 {
		if idRaw := strings.TrimSpace(c.Param("id")); idRaw != "" {
			id, err := strconv.ParseUint(idRaw, 10, 64)
			if err != nil {
				response.Error(c, http.StatusBadRequest, appErr.New(3001, "invalid instanceId"))
				return
			}
			req.InstanceID = uint(id)
		}
	}
	req.Type = middlewaresvc.NormalizeType(req.Type)
	req.Action = strings.ToLower(strings.TrimSpace(req.Action))
	if req.InstanceID == 0 || req.Action == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "instanceId and action are required"))
		return
	}
	instance, driver, svcInstance, found := h.middlewareDriverForInstance(c, req.InstanceID)
	if !found {
		return
	}
	if req.Type != "" && req.Type != instance.Type {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "middleware type mismatch"))
		return
	}
	if !middlewareActionSupported(driver, req.Action) {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "unsupported middleware action"))
		return
	}
	dryRun := middlewareActionDryRun(req)
	risk := middlewareActionRisk(driver, req.Action)
	traceID := uuid.NewString()
	started := time.Now()
	plan := buildMiddlewareDryRunPlan(instance, req.Action, risk, req.Params)
	operation := models.MiddlewareOperation{
		TraceID:    traceID,
		InstanceID: instance.ID,
		Type:       instance.Type,
		Action:     req.Action,
		Status:     "dry_run",
		DryRun:     dryRun,
		RiskLevel:  risk,
		Request:    middlewareActionRequestJSON(req),
		Result:     plan,
		StartedAt:  &started,
	}
	if dryRun {
		finished := time.Now()
		operation.FinishedAt = &finished
		if err := h.DB.Create(&operation).Error; err != nil {
			response.Internal(c, err)
			return
		}
		response.Success(c, gin.H{"protocolVersion": middlewareProtocolVersion, "traceId": traceID, "operation": operation, "dryRun": plan})
		return
	}
	if middlewareActionNeedsConfirm(driver, req.Action) && strings.TrimSpace(req.ConfirmationText) != middlewareConfirmText {
		response.Error(c, http.StatusBadRequest, appErr.New(3020, "confirmation text is required"))
		return
	}
	var runningCount int64
	if err := h.DB.Model(&models.MiddlewareOperation{}).Where("instance_id = ? AND action = ? AND status = ?", instance.ID, req.Action, "running").Count(&runningCount).Error; err != nil {
		response.Internal(c, err)
		return
	}
	if runningCount > 0 {
		response.Error(c, http.StatusConflict, appErr.New(4023, "middleware action is already running"))
		return
	}
	operation.Status = "running"
	if err := h.DB.Create(&operation).Error; err != nil {
		response.Internal(c, err)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), middlewaresvc.DefaultTimeout)
	defer cancel()
	actionResult, err := driver.Execute(ctx, svcInstance, req.Action, req.Params)
	finished := time.Now()
	updates := map[string]interface{}{"finished_at": &finished}
	if err != nil {
		log.Printf("middleware action failed trace_id=%s instance_id=%d type=%s action=%s err=%v", traceID, instance.ID, instance.Type, req.Action, err)
		updates["status"] = "failed"
		updates["error_message"] = "middleware action failed"
		updates["result"] = datatypes.JSONMap{"message": "middleware action failed"}
		_ = h.DB.Model(&models.MiddlewareOperation{}).Where("id = ?", operation.ID).Updates(updates).Error
		_, _ = h.PublishNotification(NotificationOptions{
			TraceID:      traceID,
			Module:       "middleware",
			Source:       "middleware-action",
			Event:        "middleware.action.failed",
			Severity:     "error",
			ResourceType: instance.Type,
			ResourceID:   strconv.FormatUint(uint64(instance.ID), 10),
			Title:        "中间件动作失败",
			Content:      instance.Name + " 执行 " + req.Action + " 失败",
			Data:         gin.H{"instanceId": instance.ID, "operationId": operation.ID, "action": req.Action},
		})
		response.Error(c, http.StatusBadRequest, appErr.New(4024, "middleware action failed"))
		return
	}
	updates["status"] = "success"
	updates["result"] = datatypes.JSONMap{"status": actionResult.Status, "message": actionResult.Message, "data": actionResult.Data}
	if err := h.DB.Model(&models.MiddlewareOperation{}).Where("id = ?", operation.ID).Updates(updates).Error; err != nil {
		response.Internal(c, err)
		return
	}
	var saved models.MiddlewareOperation
	if err := h.DB.First(&saved, operation.ID).Error; err != nil {
		response.Internal(c, err)
		return
	}
	_, _ = h.PublishNotification(NotificationOptions{
		TraceID:      traceID,
		Module:       "middleware",
		Source:       "middleware-action",
		Event:        "middleware.action.success",
		Severity:     "success",
		ResourceType: instance.Type,
		ResourceID:   strconv.FormatUint(uint64(instance.ID), 10),
		Title:        "中间件动作完成",
		Content:      instance.Name + " 已执行 " + req.Action,
		Data:         gin.H{"instanceId": instance.ID, "operationId": operation.ID, "action": req.Action},
	})
	response.Success(c, gin.H{"protocolVersion": middlewareProtocolVersion, "traceId": traceID, "operation": saved})
}

func (h *Handler) ListMiddlewareOperations(c *gin.Context) {
	page := pagination.Parse(c)
	query := h.DB.Model(&models.MiddlewareOperation{})
	if instanceIDRaw := strings.TrimSpace(c.Query("instanceId")); instanceIDRaw != "" {
		instanceID, err := strconv.ParseUint(instanceIDRaw, 10, 64)
		if err != nil {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "invalid instanceId"))
			return
		}
		query = query.Where("instance_id = ?", uint(instanceID))
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	var items []models.MiddlewareOperation
	var total int64
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

func (h *Handler) GetMiddlewareOperation(c *gin.Context) {
	getByID[models.MiddlewareOperation](c, h.DB)
}

func (h *Handler) buildMiddlewareInstance(name, typ, endpoint, healthPath, env, owner, authType string, tlsEnable bool, username, password, token string, labels, metadata map[string]interface{}) (models.MiddlewareInstance, models.MiddlewareCredential, error) {
	instance := models.MiddlewareInstance{
		Name:       strings.TrimSpace(name),
		Type:       middlewaresvc.NormalizeType(typ),
		Endpoint:   strings.TrimSpace(endpoint),
		HealthPath: strings.TrimSpace(healthPath),
		Env:        defaultString(strings.TrimSpace(env), "prod"),
		Owner:      strings.TrimSpace(owner),
		AuthType:   defaultString(strings.TrimSpace(authType), "password"),
		TLSEnable:  tlsEnable,
		Status:     "unknown",
		Labels:     datatypes.JSONMap(labels),
		Metadata:   datatypes.JSONMap(metadata),
	}
	if err := validateMiddlewareInstance(instance); err != nil {
		return instance, models.MiddlewareCredential{}, err
	}
	credential, err := h.buildMiddlewareCredential(username, password, token)
	return instance, credential, err
}

func validateMiddlewareInstance(instance models.MiddlewareInstance) error {
	if strings.TrimSpace(instance.Name) == "" {
		return errors.New("name cannot be empty")
	}
	if instance.Type == "" {
		return errors.New("unsupported middleware type")
	}
	return middlewaresvc.ValidateEndpoint(middlewaresvc.Instance{Type: instance.Type, Endpoint: instance.Endpoint, Env: instance.Env, TLSEnable: instance.TLSEnable})
}

func (h *Handler) buildMiddlewareCredential(username, password, token string) (models.MiddlewareCredential, error) {
	encryptedPassword, err := h.encryptCloudCredential(password)
	if err != nil {
		return models.MiddlewareCredential{}, err
	}
	encryptedToken, err := h.encryptCloudCredential(token)
	if err != nil {
		return models.MiddlewareCredential{}, err
	}
	return models.MiddlewareCredential{
		Username:   strings.TrimSpace(username),
		Password:   encryptedPassword,
		Token:      encryptedToken,
		KeyVersion: "v1",
	}, nil
}

func (h *Handler) upsertMiddlewareCredential(instanceID uint, username, password, token *string) error {
	updates := map[string]interface{}{}
	if username != nil {
		updates["username"] = strings.TrimSpace(*username)
	}
	if password != nil && strings.TrimSpace(*password) != "" {
		encrypted, err := h.encryptCloudCredential(*password)
		if err != nil {
			return err
		}
		updates["password"] = encrypted
	}
	if token != nil && strings.TrimSpace(*token) != "" {
		encrypted, err := h.encryptCloudCredential(*token)
		if err != nil {
			return err
		}
		updates["token"] = encrypted
	}
	if len(updates) == 0 {
		return nil
	}
	now := time.Now()
	updates["rotated_at"] = &now
	var existing models.MiddlewareCredential
	if err := h.DB.Where("instance_id = ?", instanceID).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cred := models.MiddlewareCredential{InstanceID: instanceID, KeyVersion: "v1", RotatedAt: &now}
			if value, ok := updates["username"].(string); ok {
				cred.Username = value
			}
			if value, ok := updates["password"].(string); ok {
				cred.Password = value
			}
			if value, ok := updates["token"].(string); ok {
				cred.Token = value
			}
			return h.DB.Create(&cred).Error
		}
		return err
	}
	return h.DB.Model(&models.MiddlewareCredential{}).Where("instance_id = ?", instanceID).Updates(updates).Error
}

func (h *Handler) middlewareDriverForInstance(c *gin.Context, id uint) (models.MiddlewareInstance, middlewaresvc.Driver, middlewaresvc.Instance, bool) {
	var instance models.MiddlewareInstance
	if err := h.DB.First(&instance, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return instance, nil, middlewaresvc.Instance{}, false
		}
		response.Internal(c, err)
		return instance, nil, middlewaresvc.Instance{}, false
	}
	driver, ok := middlewaresvc.DriverFor(instance.Type)
	if !ok {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "unsupported middleware type"))
		return instance, nil, middlewaresvc.Instance{}, false
	}
	var credential models.MiddlewareCredential
	_ = h.DB.Where("instance_id = ?", instance.ID).First(&credential).Error
	password, err := h.decryptCloudCredential(credential.Password)
	if err != nil {
		response.Internal(c, err)
		return instance, nil, middlewaresvc.Instance{}, false
	}
	token, err := h.decryptCloudCredential(credential.Token)
	if err != nil {
		response.Internal(c, err)
		return instance, nil, middlewaresvc.Instance{}, false
	}
	return instance, driver, middlewaresvc.Instance{
		ID:        instance.ID,
		Type:      instance.Type,
		Endpoint:  instance.Endpoint,
		Env:       instance.Env,
		Username:  credential.Username,
		Password:  password,
		Token:     token,
		TLSEnable: instance.TLSEnable,
	}, true
}

func middlewareProtocol() gin.H {
	resources := []gin.H{}
	for _, typ := range []string{"redis", "postgresql", "rabbitmq"} {
		driver, _ := middlewaresvc.DriverFor(typ)
		resources = append(resources, gin.H{"type": typ, "actions": driver.Actions()})
	}
	return gin.H{
		"protocolVersion": middlewareProtocolVersion,
		"actionEndpoint":  "/api/v1/middleware/actions",
		"supportedTypes":  []string{"redis", "postgresql", "rabbitmq"},
		"resources":       resources,
		"requestSchema": gin.H{
			"instanceId":       "number|required",
			"type":             "redis|postgresql|rabbitmq",
			"action":           "string|required",
			"dryRun":           "boolean|default true",
			"confirmationText": "string|required for high-risk execution; value=确认删除资源",
			"params":           "object|optional",
		},
		"safety": gin.H{"defaultDryRun": true, "confirmationText": middlewareConfirmText, "traceField": "traceId"},
	}
}

func middlewareActionSupported(driver middlewaresvc.Driver, action string) bool {
	for _, spec := range driver.Actions() {
		if spec.Name == action {
			return true
		}
	}
	return false
}

func middlewareActionRisk(driver middlewaresvc.Driver, action string) string {
	for _, spec := range driver.Actions() {
		if spec.Name == action {
			return spec.RiskLevel
		}
	}
	return "P2"
}

func middlewareActionNeedsConfirm(driver middlewaresvc.Driver, action string) bool {
	for _, spec := range driver.Actions() {
		if spec.Name == action {
			return spec.ConfirmationRequired
		}
	}
	return false
}

func middlewareActionDryRun(req middlewareActionRequest) bool {
	if req.DryRun == nil {
		return true
	}
	return *req.DryRun
}

func middlewareActionRequestJSON(req middlewareActionRequest) datatypes.JSONMap {
	return datatypes.JSONMap{
		"instanceId": req.InstanceID,
		"type":       req.Type,
		"action":     req.Action,
		"dryRun":     middlewareActionDryRun(req),
		"confirmed":  strings.TrimSpace(req.ConfirmationText) == middlewareConfirmText,
		"params":     req.Params,
	}
}

func buildMiddlewareDryRunPlan(instance models.MiddlewareInstance, action string, risk string, params map[string]interface{}) datatypes.JSONMap {
	return datatypes.JSONMap{
		"instanceId":        instance.ID,
		"instanceName":      instance.Name,
		"type":              instance.Type,
		"action":            action,
		"riskLevel":         risk,
		"approvalRequired":  risk == "P0" || risk == "P1",
		"impact":            middlewareActionImpact(action),
		"estimatedDuration": "seconds",
		"steps": []interface{}{
			"校验 RBAC/ABAC 权限",
			"校验中间件实例与动作白名单",
			"执行 dry-run 或真实驱动动作",
			"写入 middleware_operations 审计记录",
		},
		"safetyChecks": []interface{}{
			"默认 dry-run",
			"高危动作真实执行需确认文案",
			"同一实例同一动作禁止并发执行",
		},
		"rollback": middlewareActionRollback(action),
		"params":   params,
	}
}

func middlewareActionImpact(action string) string {
	switch action {
	case "flushdb":
		return "清空 Redis 当前 DB，数据可能不可恢复"
	case "terminate_backend":
		return "终止 PostgreSQL 后端连接，可能中断业务会话"
	case "queue_purge":
		return "清空 RabbitMQ 队列消息，消息可能不可恢复"
	default:
		return "执行低风险查询或检查动作"
	}
}

func middlewareActionRollback(action string) string {
	switch action {
	case "flushdb", "queue_purge":
		return "不可直接回滚，需依赖备份或上游重放"
	case "terminate_backend":
		return "业务连接通常可自动重连，需观察连接池恢复情况"
	default:
		return "无需回滚"
	}
}
