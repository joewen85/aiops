package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	"devops-system/backend/internal/ai"
	"devops-system/backend/internal/cloud"
	appErr "devops-system/backend/internal/errors"
	"devops-system/backend/internal/models"
	"devops-system/backend/internal/pagination"
	"devops-system/backend/internal/response"
)

func (h *Handler) ListCloudAccounts(c *gin.Context) {
	page := pagination.Parse(c)
	query := h.DB.Model(&models.CloudAccount{})

	if provider := strings.TrimSpace(c.Query("provider")); provider != "" {
		query = query.Where("provider = ?", provider)
	}
	if region := strings.TrimSpace(c.Query("region")); region != "" {
		query = query.Where("region = ?", region)
	}
	if verifiedRaw := strings.TrimSpace(c.Query("verified")); verifiedRaw != "" {
		verified, ok := parseCloudAccountVerifiedQuery(verifiedRaw)
		if !ok {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "invalid verified"))
			return
		}
		query = query.Where("is_verified = ?", verified)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where("name LIKE ? OR access_key LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	var (
		items []models.CloudAccount
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
func (h *Handler) GetCloudAccount(c *gin.Context)    { getByID[models.CloudAccount](c, h.DB) }
func (h *Handler) CreateCloudAccount(c *gin.Context) { createByModel[models.CloudAccount](c, h.DB) }
func (h *Handler) UpdateCloudAccount(c *gin.Context) { updateByModel[models.CloudAccount](c, h.DB) }
func (h *Handler) DeleteCloudAccount(c *gin.Context) { deleteByModel[models.CloudAccount](c, h.DB) }

func (h *Handler) VerifyCloudAccount(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var account models.CloudAccount
	if err := h.DB.First(&account, id).Error; err != nil {
		response.Internal(c, err)
		return
	}
	provider, exists := h.CloudProviders[account.Provider]
	if !exists {
		response.Error(c, http.StatusBadRequest, appErr.New(4003, "unsupported cloud provider"))
		return
	}
	if err := provider.Verify(cloudCred(account)); err != nil {
		response.Error(c, http.StatusBadRequest, appErr.New(4004, err.Error()))
		return
	}
	if err := h.DB.Model(&account).Update("is_verified", true).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{"id": id, "verified": true})
}

func (h *Handler) SyncCloudAccount(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var account models.CloudAccount
	if err := h.DB.First(&account, id).Error; err != nil {
		response.Internal(c, err)
		return
	}
	provider, exists := h.CloudProviders[account.Provider]
	if !exists {
		response.Error(c, http.StatusBadRequest, appErr.New(4003, "unsupported cloud provider"))
		return
	}
	started := time.Now()
	job := models.CloudSyncJob{
		AccountID: account.ID,
		Provider:  account.Provider,
		Region:    account.Region,
		Status:    "running",
		StartedAt: &started,
		Summary:   datatypes.JSONMap{},
	}
	if err := h.DB.Create(&job).Error; err != nil {
		response.Internal(c, err)
		return
	}

	assets, err := h.collectCloudProviderAssets(provider, cloudCred(account))
	if err != nil {
		finished := time.Now()
		_ = h.DB.Model(&models.CloudSyncJob{}).Where("id = ?", job.ID).Updates(map[string]interface{}{
			"status":      "failed",
			"finished_at": &finished,
			"summary": datatypes.JSONMap{
				"error": err.Error(),
			},
		}).Error
		response.Error(c, http.StatusBadRequest, appErr.New(4005, err.Error()))
		return
	}
	cloudAssets, syncSummary, cloudErr := h.syncCloudAssets(account, assets, "CloudAPI")
	if cloudErr != nil {
		finished := time.Now()
		syncSummary["error"] = cloudErr.Error()
		_ = h.DB.Model(&models.CloudSyncJob{}).Where("id = ?", job.ID).Updates(map[string]interface{}{
			"status":      "failed",
			"finished_at": &finished,
			"summary":     syncSummary,
		}).Error
		response.Internal(c, cloudErr)
		return
	}

	cmdbResources, cmdbErr := h.syncCloudResourcesToCMDB(account, assets)
	if cmdbErr != nil {
		finished := time.Now()
		syncSummary["error"] = cmdbErr.Error()
		syncSummary["providerAssets"] = len(assets)
		syncSummary["cloudAssets"] = len(cloudAssets)
		_ = h.DB.Model(&models.CloudSyncJob{}).Where("id = ?", job.ID).Updates(map[string]interface{}{
			"status":      "failed",
			"finished_at": &finished,
			"summary":     syncSummary,
		}).Error
		response.Internal(c, cmdbErr)
		return
	}
	finished := time.Now()
	syncSummary["providerAssets"] = len(assets)
	syncSummary["cloudAssets"] = len(cloudAssets)
	syncSummary["cmdbAssets"] = len(cmdbResources)
	if err := h.DB.Model(&models.CloudSyncJob{}).Where("id = ?", job.ID).Updates(map[string]interface{}{
		"status":      "success",
		"finished_at": &finished,
		"summary":     syncSummary,
	}).Error; err != nil {
		response.Internal(c, err)
		return
	}
	var savedJob models.CloudSyncJob
	if err := h.DB.First(&savedJob, job.ID).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{
		"id":            id,
		"job":           savedJob,
		"assets":        assets,
		"cloudAssets":   cloudAssets,
		"syncSummary":   syncSummary,
		"cmdbResources": cmdbResources,
		"cmdbAssets":    asCloudAssetSlice(cmdbResources),
	})
}

func (h *Handler) ListTickets(c *gin.Context)  { listByModel[models.Ticket](c, h.DB) }
func (h *Handler) GetTicket(c *gin.Context)    { getByID[models.Ticket](c, h.DB) }
func (h *Handler) CreateTicket(c *gin.Context) { createByModel[models.Ticket](c, h.DB) }
func (h *Handler) UpdateTicket(c *gin.Context) { updateByModel[models.Ticket](c, h.DB) }
func (h *Handler) DeleteTicket(c *gin.Context) { deleteByModel[models.Ticket](c, h.DB) }

func (h *Handler) TransitionTicket(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if err := h.DB.Model(&models.Ticket{}).Where("id = ?", id).Update("status", req.Status).Error; err != nil {
		response.Internal(c, err)
		return
	}
	getByID[models.Ticket](c, h.DB)
}

func (h *Handler) ApproveTicket(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.DB.Model(&models.Ticket{}).Where("id = ?", id).Update("status", "processing").Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{"id": id, "approved": true})
}

func (h *Handler) ListDockerHosts(c *gin.Context)  { listByModel[models.DockerHost](c, h.DB) }
func (h *Handler) GetDockerHost(c *gin.Context)    { getByID[models.DockerHost](c, h.DB) }
func (h *Handler) CreateDockerHost(c *gin.Context) { createByModel[models.DockerHost](c, h.DB) }
func (h *Handler) UpdateDockerHost(c *gin.Context) { updateByModel[models.DockerHost](c, h.DB) }
func (h *Handler) DeleteDockerHost(c *gin.Context) { deleteByModel[models.DockerHost](c, h.DB) }

func (h *Handler) ListComposeStacks(c *gin.Context) { listByModel[models.DockerComposeStack](c, h.DB) }
func (h *Handler) CreateComposeStack(c *gin.Context) {
	createByModel[models.DockerComposeStack](c, h.DB)
}
func (h *Handler) UpdateComposeStack(c *gin.Context) {
	updateByModel[models.DockerComposeStack](c, h.DB)
}
func (h *Handler) DeleteComposeStack(c *gin.Context) {
	deleteByModel[models.DockerComposeStack](c, h.DB)
}

func (h *Handler) DockerStubAction(c *gin.Context) {
	response.Success(c, gin.H{"result": "docker action accepted", "status": "stub"})
}

func (h *Handler) ListMiddlewareInstances(c *gin.Context) {
	listByModel[models.MiddlewareInstance](c, h.DB)
}
func (h *Handler) GetMiddlewareInstance(c *gin.Context) { getByID[models.MiddlewareInstance](c, h.DB) }
func (h *Handler) CreateMiddlewareInstance(c *gin.Context) {
	createByModel[models.MiddlewareInstance](c, h.DB)
}
func (h *Handler) UpdateMiddlewareInstance(c *gin.Context) {
	updateByModel[models.MiddlewareInstance](c, h.DB)
}
func (h *Handler) DeleteMiddlewareInstance(c *gin.Context) {
	deleteByModel[models.MiddlewareInstance](c, h.DB)
}

func (h *Handler) CheckMiddlewareInstance(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	response.Success(c, gin.H{"id": id, "healthy": true})
}

func parseCloudAccountVerifiedQuery(raw string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes":
		return true, true
	case "0", "false", "no":
		return false, true
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, false
	}
	return value, true
}

func (h *Handler) MiddlewareAction(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Action string `json:"action" binding:"required"`
	}
	if !bindJSON(c, &req) {
		return
	}
	response.Success(c, gin.H{"id": id, "action": req.Action, "status": "accepted"})
}

func (h *Handler) ListObservabilitySources(c *gin.Context) {
	listByModel[models.ObservabilitySource](c, h.DB)
}
func (h *Handler) GetObservabilitySource(c *gin.Context) {
	getByID[models.ObservabilitySource](c, h.DB)
}
func (h *Handler) CreateObservabilitySource(c *gin.Context) {
	createByModel[models.ObservabilitySource](c, h.DB)
}
func (h *Handler) UpdateObservabilitySource(c *gin.Context) {
	updateByModel[models.ObservabilitySource](c, h.DB)
}
func (h *Handler) DeleteObservabilitySource(c *gin.Context) {
	deleteByModel[models.ObservabilitySource](c, h.DB)
}

func (h *Handler) QueryMetrics(c *gin.Context) {
	response.Success(c, gin.H{
		"series": []gin.H{
			{"name": "cpu_usage", "points": []float64{23.1, 25.8, 21.4}},
			{"name": "memory_usage", "points": []float64{61.3, 63.2, 62.9}},
		},
	})
}

func (h *Handler) ListKubernetesClusters(c *gin.Context) {
	listByModel[models.KubernetesCluster](c, h.DB)
}
func (h *Handler) GetKubernetesCluster(c *gin.Context) { getByID[models.KubernetesCluster](c, h.DB) }
func (h *Handler) CreateKubernetesCluster(c *gin.Context) {
	createByModel[models.KubernetesCluster](c, h.DB)
}
func (h *Handler) UpdateKubernetesCluster(c *gin.Context) {
	updateByModel[models.KubernetesCluster](c, h.DB)
}
func (h *Handler) DeleteKubernetesCluster(c *gin.Context) {
	deleteByModel[models.KubernetesCluster](c, h.DB)
}

func (h *Handler) ListKubernetesNodes(c *gin.Context) {
	response.Success(c, gin.H{
		"nodes": []gin.H{{"name": "node-1", "status": "Ready"}, {"name": "node-2", "status": "Ready"}},
	})
}

func (h *Handler) KubernetesResourceAction(c *gin.Context) {
	response.Success(c, gin.H{"result": "accepted", "status": "stub"})
}

func (h *Handler) ListEvents(c *gin.Context)  { listByModel[models.Event](c, h.DB) }
func (h *Handler) GetEvent(c *gin.Context)    { getByID[models.Event](c, h.DB) }
func (h *Handler) CreateEvent(c *gin.Context) { createByModel[models.Event](c, h.DB) }
func (h *Handler) UpdateEvent(c *gin.Context) { updateByModel[models.Event](c, h.DB) }
func (h *Handler) DeleteEvent(c *gin.Context) { deleteByModel[models.Event](c, h.DB) }

func (h *Handler) SearchEvents(c *gin.Context) {
	keyword := c.Query("keyword")
	var items []models.Event
	query := h.DB.Order("id desc")
	if keyword != "" {
		query = query.Where("summary LIKE ?", "%"+keyword+"%")
	}
	if err := query.Find(&items).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{"list": items})
}

func (h *Handler) LinkEvent(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		TicketID uint `json:"ticketId"`
		TaskID   uint `json:"taskId"`
	}
	if !bindJSON(c, &req) {
		return
	}
	response.Success(c, gin.H{"id": id, "ticketId": req.TicketID, "taskId": req.TaskID, "linked": true})
}

func (h *Handler) ListTools(c *gin.Context)  { listByModel[models.ToolItem](c, h.DB) }
func (h *Handler) GetTool(c *gin.Context)    { getByID[models.ToolItem](c, h.DB) }
func (h *Handler) CreateTool(c *gin.Context) { createByModel[models.ToolItem](c, h.DB) }
func (h *Handler) UpdateTool(c *gin.Context) { updateByModel[models.ToolItem](c, h.DB) }
func (h *Handler) DeleteTool(c *gin.Context) { deleteByModel[models.ToolItem](c, h.DB) }

func (h *Handler) ExecuteTool(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	response.Success(c, gin.H{"id": id, "status": "executed", "result": "stub"})
}

func (h *Handler) ListAIAgents(c *gin.Context)  { listByModel[models.AIAgentConfig](c, h.DB) }
func (h *Handler) GetAIAgent(c *gin.Context)    { getByID[models.AIAgentConfig](c, h.DB) }
func (h *Handler) CreateAIAgent(c *gin.Context) { createByModel[models.AIAgentConfig](c, h.DB) }
func (h *Handler) UpdateAIAgent(c *gin.Context) { updateByModel[models.AIAgentConfig](c, h.DB) }
func (h *Handler) DeleteAIAgent(c *gin.Context) { deleteByModel[models.AIAgentConfig](c, h.DB) }

func (h *Handler) ListAIModels(c *gin.Context)  { listByModel[models.AIModelConfig](c, h.DB) }
func (h *Handler) GetAIModel(c *gin.Context)    { getByID[models.AIModelConfig](c, h.DB) }
func (h *Handler) CreateAIModel(c *gin.Context) { createByModel[models.AIModelConfig](c, h.DB) }
func (h *Handler) UpdateAIModel(c *gin.Context) { updateByModel[models.AIModelConfig](c, h.DB) }
func (h *Handler) DeleteAIModel(c *gin.Context) { deleteByModel[models.AIModelConfig](c, h.DB) }

func (h *Handler) AIOpsChat(c *gin.Context) {
	var req struct {
		Provider string                 `json:"provider"`
		Endpoint string                 `json:"endpoint"`
		APIKey   string                 `json:"apiKey"`
		Payload  map[string]interface{} `json:"payload"`
	}
	if !bindJSON(c, &req) {
		return
	}
	provider := req.Provider
	if provider == "" {
		provider = "openai"
	}
	modelProvider, exists := h.ModelProviders[provider]
	if !exists {
		response.Error(c, http.StatusBadRequest, appErr.New(4006, "unsupported model provider"))
		return
	}
	resp, err := modelProvider.Chat(req.Endpoint, req.APIKey, req.Payload)
	if err != nil {
		response.Error(c, http.StatusBadRequest, appErr.New(4007, err.Error()))
		return
	}
	response.Success(c, resp)
}

func (h *Handler) AIOpsRCA(c *gin.Context) {
	var req struct {
		Target string `json:"target" binding:"required"`
	}
	if !bindJSON(c, &req) {
		return
	}
	response.Success(c, gin.H{
		"target": req.Target,
		"analysis": []string{
			"cpu burst detected from observability",
			"related host flagged in cmdb",
			"suggested action: scale deployment and limit noisy workload",
		},
	})
}

func (h *Handler) AIOpsProcurementProtocol(c *gin.Context) {
	if h.Procurement == nil {
		response.Error(c, http.StatusServiceUnavailable, appErr.New(4010, "procurement engine unavailable"))
		return
	}
	response.Success(c, h.Procurement.ProtocolSpec())
}

func (h *Handler) AIOpsParseProcurementIntent(c *gin.Context) {
	if h.Procurement == nil {
		response.Error(c, http.StatusServiceUnavailable, appErr.New(4010, "procurement engine unavailable"))
		return
	}
	var req ai.ProcurementNLRequest
	if !bindJSON(c, &req) {
		return
	}
	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "message is required"))
		return
	}
	intent, clarifications, err := h.Procurement.ParseIntent(req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, appErr.New(4008, err.Error()))
		return
	}
	response.Success(c, gin.H{
		"protocolVersion": ai.ProcurementProtocolVersion,
		"intent":          intent,
		"clarifications":  clarifications,
		"next":            "plan",
	})
}

func (h *Handler) AIOpsBuildProcurementPlan(c *gin.Context) {
	if h.Procurement == nil {
		response.Error(c, http.StatusServiceUnavailable, appErr.New(4010, "procurement engine unavailable"))
		return
	}
	var req struct {
		Intent ai.ProcurementIntent `json:"intent"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if strings.TrimSpace(req.Intent.RawMessage) == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "intent.rawMessage is required"))
		return
	}
	plan, err := h.Procurement.BuildPlan(req.Intent)
	if err != nil {
		response.Error(c, http.StatusBadRequest, appErr.New(4009, err.Error()))
		return
	}
	response.Success(c, gin.H{
		"protocolVersion": ai.ProcurementProtocolVersion,
		"plan":            plan,
		"next":            "execute",
	})
}

func (h *Handler) AIOpsExecuteProcurementPlan(c *gin.Context) {
	if h.Procurement == nil {
		response.Error(c, http.StatusServiceUnavailable, appErr.New(4010, "procurement engine unavailable"))
		return
	}
	var req struct {
		Plan   ai.ProcurementPlan `json:"plan"`
		DryRun bool               `json:"dryRun"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if strings.TrimSpace(req.Plan.PlanID) == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "plan.planId is required"))
		return
	}
	result, err := h.Procurement.ExecutePlan(req.Plan, req.DryRun)
	if err != nil {
		response.Error(c, http.StatusBadRequest, appErr.New(4011, err.Error()))
		return
	}
	response.Success(c, gin.H{
		"protocolVersion": ai.ProcurementProtocolVersion,
		"result":          result,
	})
}

func cloudCred(account models.CloudAccount) cloud.Credentials {
	return cloud.Credentials{
		AccessKey: account.AccessKey,
		SecretKey: account.SecretKey,
		Region:    account.Region,
	}
}
