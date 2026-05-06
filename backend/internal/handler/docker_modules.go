package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	dockerclient "devops-system/backend/internal/docker"
	appErr "devops-system/backend/internal/errors"
	"devops-system/backend/internal/models"
	"devops-system/backend/internal/pagination"
	"devops-system/backend/internal/response"
)

const dockerOpsProtocolVersion = "aiops.dockerops.v1alpha1"
const dockerDeleteConfirmText = "确认删除资源"

type dockerActionRequest struct {
	HostID           uint                   `json:"hostId"`
	ResourceType     string                 `json:"resourceType"`
	ResourceID       string                 `json:"resourceId"`
	Action           string                 `json:"action"`
	DryRun           *bool                  `json:"dryRun"`
	ConfirmText      string                 `json:"confirmText"`
	ConfirmationText string                 `json:"confirmationText"`
	Params           map[string]interface{} `json:"params"`
}

type dockerResourceItem struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Status       string                 `json:"status"`
	Image        string                 `json:"image,omitempty"`
	Driver       string                 `json:"driver,omitempty"`
	Size         int64                  `json:"size,omitempty"`
	Raw          map[string]interface{} `json:"raw,omitempty"`
	AIOpsActions []string               `json:"aiopsActions"`
}

func (h *Handler) ListDockerHosts(c *gin.Context) {
	page := pagination.Parse(c)
	query := h.DB.Model(&models.DockerHost{})
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where("name LIKE ? OR endpoint LIKE ? OR owner LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if env := strings.TrimSpace(c.Query("env")); env != "" {
		query = query.Where("env = ?", env)
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if tlsRaw := strings.TrimSpace(c.Query("tls")); tlsRaw != "" {
		tlsEnable, err := strconv.ParseBool(tlsRaw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "invalid tls"))
			return
		}
		query = query.Where("tls_enable = ?", tlsEnable)
	}
	var (
		items []models.DockerHost
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

func (h *Handler) GetDockerHost(c *gin.Context) {
	getByID[models.DockerHost](c, h.DB)
}

func (h *Handler) CreateDockerHost(c *gin.Context) {
	var req struct {
		Name      string                 `json:"name" binding:"required"`
		Endpoint  string                 `json:"endpoint" binding:"required"`
		TLSEnable bool                   `json:"tlsEnable"`
		Env       string                 `json:"env"`
		Owner     string                 `json:"owner"`
		Labels    map[string]interface{} `json:"labels"`
		Metadata  map[string]interface{} `json:"metadata"`
	}
	if !bindJSON(c, &req) {
		return
	}
	host, err := buildDockerHostFromInput(req.Name, req.Endpoint, req.Env, req.Owner, req.TLSEnable, req.Labels, req.Metadata)
	if err != nil {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, err.Error()))
		return
	}
	if err := h.DB.Create(&host).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, host)
}

func (h *Handler) UpdateDockerHost(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var host models.DockerHost
	if err := h.DB.First(&host, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	var req struct {
		Name      *string                `json:"name"`
		Endpoint  *string                `json:"endpoint"`
		TLSEnable *bool                  `json:"tlsEnable"`
		Env       *string                `json:"env"`
		Owner     *string                `json:"owner"`
		Labels    map[string]interface{} `json:"labels"`
		Metadata  map[string]interface{} `json:"metadata"`
	}
	if !bindJSON(c, &req) {
		return
	}
	updates := map[string]interface{}{}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "name cannot be empty"))
			return
		}
		updates["name"] = name
	}
	nextEndpoint := host.Endpoint
	nextTLS := host.TLSEnable
	nextEnv := defaultString(strings.TrimSpace(host.Env), "prod")
	endpointChanged := false
	if req.Endpoint != nil {
		nextEndpoint = strings.TrimSpace(*req.Endpoint)
		endpointChanged = true
		updates["endpoint"] = nextEndpoint
		updates["status"] = "unknown"
	}
	if req.TLSEnable != nil {
		nextTLS = *req.TLSEnable
		updates["tls_enable"] = nextTLS
	}
	if req.Env != nil {
		nextEnv = defaultString(strings.TrimSpace(*req.Env), "prod")
		updates["env"] = nextEnv
	}
	if endpointChanged || req.TLSEnable != nil || req.Env != nil {
		if err := validateDockerEndpoint(nextEndpoint, nextTLS, nextEnv); err != nil {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, err.Error()))
			return
		}
	}
	if req.Owner != nil {
		updates["owner"] = strings.TrimSpace(*req.Owner)
	}
	if req.Labels != nil {
		updates["labels"] = datatypes.JSONMap(req.Labels)
	}
	if req.Metadata != nil {
		updates["metadata"] = datatypes.JSONMap(req.Metadata)
	}
	if len(updates) == 0 {
		response.Success(c, host)
		return
	}
	if err := h.DB.Model(&models.DockerHost{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		response.Internal(c, err)
		return
	}
	getByID[models.DockerHost](c, h.DB)
}

func (h *Handler) DeleteDockerHost(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var stackCount int64
	if err := h.DB.Model(&models.DockerComposeStack{}).Where("host_id = ?", id).Count(&stackCount).Error; err != nil {
		response.Internal(c, err)
		return
	}
	if stackCount > 0 {
		response.Error(c, http.StatusConflict, appErr.New(4013, "docker host has compose stacks"))
		return
	}
	var host models.DockerHost
	if err := h.DB.First(&host, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	if dockerStatusRunning(host.Status) {
		response.Error(c, http.StatusConflict, appErr.New(4017, "connected docker host cannot be deleted"))
		return
	}
	if err := h.DB.Delete(&models.DockerHost{}, id).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *Handler) CheckDockerHost(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	host, found := h.findDockerHost(c, id)
	if !found {
		return
	}
	client, err := dockerClientForHost(host)
	if err != nil {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, err.Error()))
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), dockerclient.DefaultTimeout)
	defer cancel()
	ping, err := client.Ping(ctx)
	status := "connected"
	var version map[string]interface{}
	if err == nil {
		version, err = client.Version(ctx)
	}
	now := time.Now()
	updates := map[string]interface{}{
		"last_heartbeat_at": &now,
	}
	if err != nil {
		log.Printf("docker host check failed host_id=%d endpoint=%s err=%v", host.ID, host.Endpoint, err)
		updates["status"] = "error"
		updates["metadata"] = datatypes.JSONMap{"lastError": err.Error()}
		_ = h.DB.Model(&models.DockerHost{}).Where("id = ?", host.ID).Updates(updates).Error
		response.Error(c, http.StatusBadRequest, appErr.New(4014, "docker host check failed"))
		return
	}
	versionText := stringFromMap(version, "Version")
	updates["status"] = status
	updates["version"] = versionText
	updates["metadata"] = datatypes.JSONMap{
		"ping":    ping,
		"version": version,
	}
	if err := h.DB.Model(&models.DockerHost{}).Where("id = ?", host.ID).Updates(updates).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{"id": host.ID, "status": status, "version": versionText, "ping": ping})
}

func (h *Handler) ListDockerHostResources(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	host, found := h.findDockerHost(c, id)
	if !found {
		return
	}
	resourceType := normalizeDockerResourceType(c.Query("type"))
	if resourceType == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "unsupported docker resource type"))
		return
	}
	client, err := dockerClientForHost(host)
	if err != nil {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, err.Error()))
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), dockerclient.DefaultTimeout)
	defer cancel()
	items, err := loadDockerResources(ctx, client, resourceType)
	if err != nil {
		log.Printf("docker resource query failed host_id=%d type=%s err=%v", host.ID, resourceType, err)
		response.Error(c, http.StatusBadRequest, appErr.New(4015, "docker resource query failed"))
		return
	}
	keyword := strings.ToLower(strings.TrimSpace(c.Query("keyword")))
	normalized := make([]dockerResourceItem, 0, len(items))
	for _, item := range items {
		resource := normalizeDockerResource(resourceType, item)
		if keyword != "" && !strings.Contains(strings.ToLower(resource.ID+" "+resource.Name+" "+resource.Status+" "+resource.Image), keyword) {
			continue
		}
		normalized = append(normalized, resource)
	}
	page := pagination.Parse(c)
	total := int64(len(normalized))
	start := pagination.Offset(page)
	if start > len(normalized) {
		start = len(normalized)
	}
	end := start + page.PageSize
	if end > len(normalized) {
		end = len(normalized)
	}
	response.List(c, normalized[start:end], total, page.Page, page.PageSize)
}

func (h *Handler) ListComposeStacks(c *gin.Context) {
	page := pagination.Parse(c)
	query := h.DB.Model(&models.DockerComposeStack{})
	if hostIDRaw := strings.TrimSpace(c.Query("hostId")); hostIDRaw != "" {
		hostID, err := strconv.ParseUint(hostIDRaw, 10, 64)
		if err != nil {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "invalid hostId"))
			return
		}
		query = query.Where("host_id = ?", uint(hostID))
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where("name LIKE ? OR content LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	var (
		items []models.DockerComposeStack
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

func (h *Handler) CreateComposeStack(c *gin.Context) {
	var req struct {
		HostID   uint   `json:"hostId" binding:"required"`
		Name     string `json:"name" binding:"required"`
		Content  string `json:"content" binding:"required"`
		Status   string `json:"status"`
		Services int    `json:"services"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if _, found := h.findDockerHost(c, req.HostID); !found {
		return
	}
	stack := models.DockerComposeStack{
		HostID:   req.HostID,
		Name:     strings.TrimSpace(req.Name),
		Content:  strings.TrimSpace(req.Content),
		Status:   defaultString(strings.TrimSpace(req.Status), "draft"),
		Services: req.Services,
	}
	if stack.Name == "" || stack.Content == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "name and content are required"))
		return
	}
	if err := h.DB.Create(&stack).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, stack)
}

func (h *Handler) UpdateComposeStack(c *gin.Context) {
	updateByModel[models.DockerComposeStack](c, h.DB)
}

func (h *Handler) DeleteComposeStack(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var stack models.DockerComposeStack
	if err := h.DB.First(&stack, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	if dockerStatusRunning(stack.Status) {
		response.Error(c, http.StatusConflict, appErr.New(4018, "running compose stack cannot be deleted"))
		return
	}
	if err := h.DB.Delete(&models.DockerComposeStack{}, id).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *Handler) DockerAIOpsProtocol(c *gin.Context) {
	response.Success(c, dockerAIOpsProtocol())
}

func (h *Handler) DockerAction(c *gin.Context) {
	var req dockerActionRequest
	if !bindJSON(c, &req) {
		return
	}
	req.ResourceType = normalizeDockerResourceType(req.ResourceType)
	req.Action = strings.ToLower(strings.TrimSpace(req.Action))
	req.ResourceID = strings.TrimSpace(req.ResourceID)
	if req.HostID == 0 || req.ResourceType == "" || req.Action == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "hostId, resourceType and action are required"))
		return
	}
	if req.ResourceType != "compose" && req.ResourceID == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "resourceId is required"))
		return
	}
	if req.ResourceType == "compose" {
		if _, err := strconv.ParseUint(req.ResourceID, 10, 64); err != nil {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "compose resourceId must be stack id"))
			return
		}
	}
	if !dockerActionSupported(req.ResourceType, req.Action) {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "unsupported docker action"))
		return
	}
	host, found := h.findDockerHost(c, req.HostID)
	if !found {
		return
	}
	traceID := uuid.NewString()
	started := time.Now()
	dryRun := dockerActionDryRun(req)
	dryRunPlan := buildDockerDryRunPlan(host, req)
	operation := models.DockerOperation{
		TraceID:      traceID,
		HostID:       host.ID,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		Action:       req.Action,
		Status:       "dry_run",
		DryRun:       dryRun,
		RiskLevel:    riskLevelForDockerAction(req.Action),
		Request:      dockerActionRequestJSON(req),
		Result:       dryRunPlan,
		StartedAt:    &started,
	}
	if dryRun {
		finished := time.Now()
		operation.FinishedAt = &finished
		if err := h.DB.Create(&operation).Error; err != nil {
			response.Internal(c, err)
			return
		}
		response.Success(c, gin.H{"protocolVersion": dockerOpsProtocolVersion, "traceId": traceID, "operation": operation, "dryRun": dryRunPlan})
		return
	}
	if dockerActionNeedsConfirm(req) && dockerActionConfirmText(req) != dockerDeleteConfirmText {
		response.Error(c, http.StatusBadRequest, appErr.New(3020, "confirmation text is required"))
		return
	}

	var runningCount int64
	if err := h.DB.Model(&models.DockerOperation{}).
		Where("host_id = ? AND resource_type = ? AND resource_id = ? AND status = ?", host.ID, req.ResourceType, req.ResourceID, "running").
		Count(&runningCount).Error; err != nil {
		response.Internal(c, err)
		return
	}
	if runningCount > 0 {
		response.Error(c, http.StatusConflict, appErr.New(4019, "docker resource action is already running"))
		return
	}

	operation.Status = "running"
	if err := h.DB.Create(&operation).Error; err != nil {
		response.Internal(c, err)
		return
	}
	result, actionErr := h.executeDockerAction(c.Request.Context(), host, req)
	finished := time.Now()
	updates := map[string]interface{}{
		"finished_at": &finished,
		"result":      result,
	}
	if actionErr != nil {
		log.Printf("docker action failed trace_id=%s host_id=%d resource_type=%s resource_id=%s action=%s err=%v", traceID, host.ID, req.ResourceType, req.ResourceID, req.Action, actionErr)
		updates["status"] = "failed"
		updates["error_message"] = actionErr.Error()
		_ = h.DB.Model(&models.DockerOperation{}).Where("id = ?", operation.ID).Updates(updates).Error
		_, _ = h.PublishNotification(NotificationOptions{
			TraceID:      traceID,
			Module:       "docker",
			Source:       "docker-action",
			Event:        "docker.action.failed",
			Severity:     "error",
			ResourceType: req.ResourceType,
			ResourceID:   req.ResourceID,
			Title:        "Docker 动作失败",
			Content:      "Docker " + req.ResourceType + " 执行 " + req.Action + " 失败",
			Data:         gin.H{"hostId": host.ID, "operationId": operation.ID, "action": req.Action},
		})
		response.Error(c, http.StatusBadRequest, appErr.New(4016, "docker action failed"))
		return
	}
	updates["status"] = "success"
	if err := h.DB.Model(&models.DockerOperation{}).Where("id = ?", operation.ID).Updates(updates).Error; err != nil {
		response.Internal(c, err)
		return
	}
	var saved models.DockerOperation
	if err := h.DB.First(&saved, operation.ID).Error; err != nil {
		response.Internal(c, err)
		return
	}
	_, _ = h.PublishNotification(NotificationOptions{
		TraceID:      traceID,
		Module:       "docker",
		Source:       "docker-action",
		Event:        "docker.action.success",
		Severity:     "success",
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		Title:        "Docker 动作完成",
		Content:      "Docker " + req.ResourceType + " 已执行 " + req.Action,
		Data:         gin.H{"hostId": host.ID, "operationId": operation.ID, "action": req.Action},
	})
	response.Success(c, gin.H{"protocolVersion": dockerOpsProtocolVersion, "traceId": traceID, "operation": saved})
}

func (h *Handler) ListDockerOperations(c *gin.Context) {
	page := pagination.Parse(c)
	query := h.DB.Model(&models.DockerOperation{})
	if hostIDRaw := strings.TrimSpace(c.Query("hostId")); hostIDRaw != "" {
		hostID, err := strconv.ParseUint(hostIDRaw, 10, 64)
		if err != nil {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "invalid hostId"))
			return
		}
		query = query.Where("host_id = ?", uint(hostID))
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if action := strings.TrimSpace(c.Query("action")); action != "" {
		query = query.Where("action = ?", action)
	}
	var (
		items []models.DockerOperation
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

func (h *Handler) executeDockerAction(ctx context.Context, host models.DockerHost, req dockerActionRequest) (datatypes.JSONMap, error) {
	client, err := dockerClientForHost(host)
	if err != nil {
		return nil, err
	}
	actionCtx, cancel := context.WithTimeout(ctx, dockerclient.DefaultActionTimeout)
	defer cancel()
	switch req.ResourceType {
	case "container":
		if err := client.ContainerAction(actionCtx, req.ResourceID, req.Action); err != nil {
			return nil, err
		}
	case "image":
		if req.Action != "remove" {
			return nil, fmt.Errorf("unsupported image action %q", req.Action)
		}
		if err := client.RemoveImage(actionCtx, req.ResourceID); err != nil {
			return nil, err
		}
	case "network":
		if req.Action != "remove" {
			return nil, fmt.Errorf("unsupported network action %q", req.Action)
		}
		if err := client.RemoveNetwork(actionCtx, req.ResourceID); err != nil {
			return nil, err
		}
	case "volume":
		if req.Action != "remove" {
			return nil, fmt.Errorf("unsupported volume action %q", req.Action)
		}
		if err := client.RemoveVolume(actionCtx, req.ResourceID); err != nil {
			return nil, err
		}
	case "compose":
		stack, err := h.findComposeStackForAction(req.ResourceID, host.ID)
		if err != nil {
			return nil, err
		}
		output, err := dockerclient.ComposeRunner{Host: dockerclient.HostConfig{Endpoint: host.Endpoint, TLSEnable: host.TLSEnable}}.Run(actionCtx, stack.Name, stack.Content, req.Action)
		if err != nil {
			return nil, err
		}
		updates := composeStackUpdatesForAction(req.Action)
		if len(updates) > 0 {
			if err := h.DB.Model(&models.DockerComposeStack{}).Where("id = ?", stack.ID).Updates(updates).Error; err != nil {
				return nil, err
			}
		}
		return datatypes.JSONMap{"executed": true, "resourceId": req.ResourceID, "action": req.Action, "output": strings.TrimSpace(output)}, nil
	default:
		return nil, fmt.Errorf("unsupported resource type %q", req.ResourceType)
	}
	return datatypes.JSONMap{"executed": true, "resourceId": req.ResourceID, "action": req.Action}, nil
}

func (h *Handler) findComposeStackForAction(resourceID string, hostID uint) (models.DockerComposeStack, error) {
	stackID, err := strconv.ParseUint(strings.TrimSpace(resourceID), 10, 64)
	if err != nil {
		return models.DockerComposeStack{}, errors.New("invalid compose stack id")
	}
	var stack models.DockerComposeStack
	if err := h.DB.Where("id = ? AND host_id = ?", uint(stackID), hostID).First(&stack).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.DockerComposeStack{}, errors.New("compose stack not found")
		}
		return models.DockerComposeStack{}, err
	}
	return stack, nil
}

func (h *Handler) findDockerHost(c *gin.Context, id uint) (models.DockerHost, bool) {
	var host models.DockerHost
	if err := h.DB.First(&host, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return host, false
		}
		response.Internal(c, err)
		return host, false
	}
	return host, true
}

func buildDockerHostFromInput(name string, endpoint string, env string, owner string, tlsEnable bool, labels map[string]interface{}, metadata map[string]interface{}) (models.DockerHost, error) {
	name = strings.TrimSpace(name)
	endpoint = strings.TrimSpace(endpoint)
	if name == "" {
		return models.DockerHost{}, errors.New("name cannot be empty")
	}
	resolvedEnv := defaultString(strings.TrimSpace(env), "prod")
	if err := validateDockerEndpoint(endpoint, tlsEnable, resolvedEnv); err != nil {
		return models.DockerHost{}, err
	}
	return models.DockerHost{
		Name:      name,
		Endpoint:  endpoint,
		TLSEnable: tlsEnable,
		Env:       resolvedEnv,
		Owner:     strings.TrimSpace(owner),
		Status:    "unknown",
		Labels:    datatypes.JSONMap(labels),
		Metadata:  datatypes.JSONMap(metadata),
	}, nil
}

func validateDockerEndpoint(endpoint string, tlsEnable bool, env string) error {
	return dockerclient.ValidateEndpointSecurity(dockerclient.HostConfig{Endpoint: endpoint, TLSEnable: tlsEnable}, env)
}

func dockerClientForHost(host models.DockerHost) (*dockerclient.Client, error) {
	if err := validateDockerEndpoint(host.Endpoint, host.TLSEnable, host.Env); err != nil {
		return nil, err
	}
	return dockerclient.NewClient(dockerclient.HostConfig{Endpoint: host.Endpoint, TLSEnable: host.TLSEnable})
}

func loadDockerResources(ctx context.Context, client *dockerclient.Client, resourceType string) ([]map[string]interface{}, error) {
	switch resourceType {
	case "container":
		return client.ListContainers(ctx)
	case "image":
		return client.ListImages(ctx)
	case "network":
		return client.ListNetworks(ctx)
	case "volume":
		return client.ListVolumes(ctx)
	default:
		return nil, errors.New("unsupported docker resource type")
	}
}

func normalizeDockerResourceType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "container", "containers":
		return "container"
	case "image", "images":
		return "image"
	case "network", "networks":
		return "network"
	case "volume", "volumes":
		return "volume"
	case "compose", "stack", "compose_stack", "compose-stack":
		return "compose"
	default:
		return ""
	}
}

func normalizeDockerResource(resourceType string, raw map[string]interface{}) dockerResourceItem {
	item := dockerResourceItem{Type: resourceType, Raw: raw, AIOpsActions: actionsForDockerResource(resourceType)}
	switch resourceType {
	case "container":
		item.ID = firstString(raw, "Id", "ID")
		item.Name = firstName(raw["Names"], firstString(raw, "Name"))
		item.Status = firstString(raw, "State", "Status")
		item.Image = firstString(raw, "Image")
	case "image":
		item.ID = firstString(raw, "Id", "ID")
		item.Name = firstName(raw["RepoTags"], firstString(raw, "RepoDigests"))
		item.Size = int64FromAny(raw["Size"])
	case "network":
		item.ID = firstString(raw, "Id", "ID")
		item.Name = firstString(raw, "Name")
		item.Driver = firstString(raw, "Driver")
		item.Status = firstString(raw, "Scope")
	case "volume":
		item.ID = firstString(raw, "Name")
		item.Name = firstString(raw, "Name")
		item.Driver = firstString(raw, "Driver")
		item.Status = firstString(raw, "Scope")
	}
	if item.Name == "" {
		item.Name = item.ID
	}
	return item
}

func dockerAIOpsProtocol() gin.H {
	return gin.H{
		"protocolVersion": dockerOpsProtocolVersion,
		"actionEndpoint":  "/api/v1/docker/actions",
		"resources": []gin.H{
			{"type": "container", "actions": []string{"start", "stop", "restart"}, "idField": "containerId"},
			{"type": "image", "actions": []string{"remove"}, "idField": "imageId", "confirmationRequired": true},
			{"type": "network", "actions": []string{"remove"}, "idField": "networkId", "confirmationRequired": true},
			{"type": "volume", "actions": []string{"remove"}, "idField": "volumeName", "confirmationRequired": true},
			{"type": "compose", "actions": []string{"validate", "deploy", "up", "down", "restart"}, "idField": "stackId", "confirmationRequiredActions": []string{"deploy", "up", "down", "restart"}},
		},
		"requestSchema": gin.H{
			"hostId":           "number|required",
			"resourceType":     "container|image|network|volume|compose",
			"resourceId":       "string|required for resource actions; compose uses stack id",
			"action":           "string|required",
			"dryRun":           "boolean|required before natural-language execution",
			"confirmationText": "string|required for destructive/high-risk execution; value=确认删除资源",
			"params":           "object|optional",
		},
		"safety": gin.H{
			"approvalRequiredLevels": []string{"P0", "P1"},
			"defaultDryRun":          true,
			"confirmationText":       dockerDeleteConfirmText,
			"traceField":             "traceId",
			"operationLog":           "docker_operations",
		},
	}
}

func dockerActionSupported(resourceType string, action string) bool {
	for _, item := range actionsForDockerResource(resourceType) {
		if item == action {
			return true
		}
	}
	return false
}

func actionsForDockerResource(resourceType string) []string {
	switch resourceType {
	case "container":
		return []string{"start", "stop", "restart"}
	case "image", "network", "volume":
		return []string{"remove"}
	case "compose":
		return []string{"validate", "deploy", "up", "down", "restart"}
	default:
		return []string{}
	}
}

func buildDockerDryRunPlan(host models.DockerHost, req dockerActionRequest) datatypes.JSONMap {
	risk := riskLevelForDockerAction(req.Action)
	return datatypes.JSONMap{
		"hostId":           host.ID,
		"hostName":         host.Name,
		"resourceType":     req.ResourceType,
		"resourceId":       req.ResourceID,
		"action":           req.Action,
		"riskLevel":        risk,
		"approvalRequired": risk == "P0" || risk == "P1",
		"impact":           dockerActionImpact(req),
		"steps": []interface{}{
			"校验 RBAC/ABAC 权限",
			"校验 Docker 主机连通性与资源状态",
			"执行 Docker API 动作或返回 dry-run 计划",
			"写入 docker_operations 审计记录",
		},
		"safetyChecks": []interface{}{
			"默认先 dry-run",
			"删除类和 Compose 高危动作真实执行必须输入确认文案",
			"运行态资源删除需先停止并二次确认",
		},
		"rollback": dockerActionRollback(req),
		"params":   req.Params,
	}
}

func riskLevelForDockerAction(action string) string {
	switch action {
	case "remove", "down":
		return "P1"
	case "stop", "restart", "deploy":
		return "P2"
	default:
		return "P3"
	}
}

func dockerActionImpact(req dockerActionRequest) string {
	switch req.Action {
	case "start":
		return "启动目标容器，可能增加主机资源占用"
	case "stop":
		return "停止目标容器，可能造成服务中断"
	case "restart":
		return "重启目标容器，存在短暂服务中断"
	case "remove":
		return "删除目标资源，可能导致镜像/网络/数据卷不可恢复"
	case "deploy", "up", "down":
		return "变更 Compose Stack 运行状态，可能影响多个服务"
	default:
		return "执行 Docker 运维动作"
	}
}

func dockerActionRollback(req dockerActionRequest) string {
	switch req.Action {
	case "start":
		return "可执行 stop 回退到停止状态"
	case "stop":
		return "可执行 start 尝试恢复"
	case "restart":
		return "如失败需查看容器日志并按镜像/配置恢复"
	case "deploy", "up", "down":
		return "使用上一版 Compose 配置重新部署"
	default:
		return "删除类动作需依赖备份或重新创建资源"
	}
}

func dockerActionRequestJSON(req dockerActionRequest) datatypes.JSONMap {
	return datatypes.JSONMap{
		"hostId":       req.HostID,
		"resourceType": req.ResourceType,
		"resourceId":   req.ResourceID,
		"action":       req.Action,
		"dryRun":       dockerActionDryRun(req),
		"confirm":      dockerActionConfirmText(req) == dockerDeleteConfirmText,
		"params":       req.Params,
	}
}

func dockerActionDryRun(req dockerActionRequest) bool {
	if req.DryRun == nil {
		return true
	}
	return *req.DryRun
}

func dockerStatusRunning(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "running", "active", "connected", "deploying", "up":
		return true
	default:
		return false
	}
}

func dockerActionNeedsConfirm(req dockerActionRequest) bool {
	if req.Action == "remove" {
		return true
	}
	return req.ResourceType == "compose" && (req.Action == "deploy" || req.Action == "up" || req.Action == "down" || req.Action == "restart")
}

func dockerActionConfirmText(req dockerActionRequest) string {
	if strings.TrimSpace(req.ConfirmationText) != "" {
		return strings.TrimSpace(req.ConfirmationText)
	}
	return strings.TrimSpace(req.ConfirmText)
}

func composeStackUpdatesForAction(action string) map[string]interface{} {
	now := time.Now()
	switch action {
	case "validate":
		return map[string]interface{}{"status": "validated"}
	case "deploy", "up":
		return map[string]interface{}{"status": "running", "last_deployed_at": &now}
	case "down":
		return map[string]interface{}{"status": "stopped"}
	case "restart":
		return map[string]interface{}{"status": "running", "last_deployed_at": &now}
	default:
		return nil
	}
}

func firstString(raw map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := raw[key]; ok {
			switch typed := value.(type) {
			case string:
				if strings.TrimSpace(typed) != "" {
					return strings.TrimSpace(typed)
				}
			case fmt.Stringer:
				return typed.String()
			}
		}
	}
	return ""
}

func firstName(value interface{}, fallback string) string {
	switch typed := value.(type) {
	case []interface{}:
		if len(typed) > 0 {
			if first, ok := typed[0].(string); ok {
				return strings.TrimPrefix(first, "/")
			}
		}
	case []string:
		if len(typed) > 0 {
			return strings.TrimPrefix(typed[0], "/")
		}
	case string:
		return strings.TrimPrefix(typed, "/")
	}
	return strings.TrimPrefix(fallback, "/")
}

func int64FromAny(value interface{}) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	case jsonNumber:
		n, _ := strconv.ParseInt(string(typed), 10, 64)
		return n
	default:
		return 0
	}
}

type jsonNumber string

func stringFromMap(raw map[string]interface{}, key string) string {
	if raw == nil {
		return ""
	}
	if value, ok := raw[key].(string); ok {
		return value
	}
	return ""
}
