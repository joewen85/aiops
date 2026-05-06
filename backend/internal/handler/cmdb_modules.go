package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
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

var (
	cmdbSyncRunningMessage = "cmdb sync job is already running"
	cmdbSourcePriority     = map[string]int{
		"IaC":      5,
		"CloudAPI": 4,
		"K8s":      3,
		"APM":      2,
		"Manual":   1,
	}
	cmdbRelationTypes = map[string]struct{}{
		"deployed_on":    {},
		"runs_in":        {},
		"connects_to":    {},
		"publishes_to":   {},
		"consumes_from":  {},
		"fronted_by":     {},
		"resolves_via":   {},
		"stores_in":      {},
		"owned_by":       {},
		"provisioned_by": {},
	}
)

type cmdbResourceCandidate struct {
	Resource models.ResourceItem
	RawID    string
	Payload  datatypes.JSONMap
	Source   string
}

type cmdbRelationCandidate struct {
	Relation models.ResourceRelation
	Source   string
	Message  string
}

func (h *Handler) ListResourceCategories(c *gin.Context) {
	listByModel[models.ResourceCategory](c, h.DB)
}

func (h *Handler) GetResourceCategory(c *gin.Context) { getByID[models.ResourceCategory](c, h.DB) }

func (h *Handler) CreateResourceCategory(c *gin.Context) {
	createByModel[models.ResourceCategory](c, h.DB)
}

func (h *Handler) UpdateResourceCategory(c *gin.Context) {
	updateByModel[models.ResourceCategory](c, h.DB)
}

func (h *Handler) DeleteResourceCategory(c *gin.Context) {
	deleteByModel[models.ResourceCategory](c, h.DB)
}

func (h *Handler) ListResources(c *gin.Context) {
	page := pagination.Parse(c)
	query := h.DB.Model(&models.ResourceItem{})

	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where("name LIKE ? OR ci_id LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if resourceType := strings.TrimSpace(c.Query("type")); resourceType != "" {
		query = query.Where("type = ?", resourceType)
	}
	if cloudName := strings.TrimSpace(c.Query("cloud")); cloudName != "" {
		query = query.Where("cloud = ?", cloudName)
	}
	if region := strings.TrimSpace(c.Query("region")); region != "" {
		query = query.Where("region = ?", region)
	}
	if env := strings.TrimSpace(c.Query("env")); env != "" {
		query = query.Where("env = ?", env)
	}
	if owner := strings.TrimSpace(c.Query("owner")); owner != "" {
		query = query.Where("owner = ?", owner)
	}

	var (
		items []models.ResourceItem
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

func (h *Handler) GetResource(c *gin.Context) {
	getByID[models.ResourceItem](c, h.DB)
}

func (h *Handler) CreateResource(c *gin.Context) {
	var input models.ResourceItem
	if !bindJSON(c, &input) {
		return
	}
	input = normalizeCMDBResource(input, "Manual")
	if input.CIID == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "ciId is required or derivable from resource fields"))
		return
	}

	saved, action, err := h.upsertCMDBResource(input)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{"action": action, "resource": saved})
}

func (h *Handler) UpdateResource(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var existing models.ResourceItem
	if err := h.DB.First(&existing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}

	var updates map[string]interface{}
	if !bindJSON(c, &updates) {
		return
	}
	delete(updates, "id")
	delete(updates, "createdAt")
	delete(updates, "created_at")

	if sourceRaw, ok := updates["source"]; ok {
		source, _ := sourceRaw.(string)
		updates["source"] = normalizeCMDBSource(source)
	}
	if ciRaw, ok := updates["ciId"]; ok {
		ciID, _ := ciRaw.(string)
		ciID = strings.TrimSpace(ciID)
		if ciID == "" {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "ciId cannot be empty"))
			return
		}
		var conflict int64
		if err := h.DB.Model(&models.ResourceItem{}).
			Where("ci_id = ? AND id <> ?", ciID, id).
			Count(&conflict).Error; err != nil {
			response.Internal(c, err)
			return
		}
		if conflict > 0 {
			response.Error(c, http.StatusBadRequest, appErr.New(4008, "ciId already exists"))
			return
		}
		updates["ci_id"] = ciID
		delete(updates, "ciId")
	}
	if _, ok := updates["lastSeenAt"]; !ok {
		updates["last_seen_at"] = time.Now()
	}
	delete(updates, "updatedAt")
	delete(updates, "updated_at")

	if err := h.DB.Model(&models.ResourceItem{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		response.Internal(c, err)
		return
	}
	getByID[models.ResourceItem](c, h.DB)
}

func (h *Handler) DeleteResource(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var item models.ResourceItem
	if err := h.DB.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	if status := readCMDBRuntimeStatus(item.Attributes); isCMDBRunningStatus(status) {
		response.Error(c, http.StatusBadRequest, appErr.New(3013, "resource is running, stop it before deletion"))
		return
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("resource_id = ?", id).Delete(&models.ResourceTag{}).Error; err != nil {
			return err
		}
		if err := tx.Where("from_ci_id = ? OR to_ci_id = ?", item.CIID, item.CIID).Delete(&models.ResourceRelation{}).Error; err != nil {
			return err
		}
		if err := tx.Where("ci_id = ?", item.CIID).Delete(&models.ResourceEvidence{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.ResourceItem{}, id).Error
	}); err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{"id": id, "ciId": item.CIID})
}

func (h *Handler) ListTags(c *gin.Context)  { listByModel[models.Tag](c, h.DB) }
func (h *Handler) GetTag(c *gin.Context)    { getByID[models.Tag](c, h.DB) }
func (h *Handler) CreateTag(c *gin.Context) { createByModel[models.Tag](c, h.DB) }
func (h *Handler) UpdateTag(c *gin.Context) { updateByModel[models.Tag](c, h.DB) }
func (h *Handler) DeleteTag(c *gin.Context) { deleteByModel[models.Tag](c, h.DB) }

func (h *Handler) BindResourceTags(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var resourceItem models.ResourceItem
	if err := h.DB.First(&resourceItem, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	var req struct {
		TagIDs []uint `json:"tagIds"`
	}
	if !bindJSON(c, &req) {
		return
	}

	if err := h.DB.Where("resource_id = ?", id).Delete(&models.ResourceTag{}).Error; err != nil {
		response.Internal(c, err)
		return
	}
	for _, tagID := range req.TagIDs {
		if err := h.DB.Create(&models.ResourceTag{ResourceID: id, TagID: tagID}).Error; err != nil {
			response.Internal(c, err)
			return
		}
	}
	response.Success(c, gin.H{"id": id, "ciId": resourceItem.CIID, "tagIds": req.TagIDs})
}

func (h *Handler) ListResourceRelations(c *gin.Context) {
	page := pagination.Parse(c)
	query := h.DB.Model(&models.ResourceRelation{})

	if fromCIID := strings.TrimSpace(c.Query("fromCiId")); fromCIID != "" {
		query = query.Where("from_ci_id = ?", fromCIID)
	}
	if toCIID := strings.TrimSpace(c.Query("toCiId")); toCIID != "" {
		query = query.Where("to_ci_id = ?", toCIID)
	}
	if relationType := strings.TrimSpace(c.Query("relationType")); relationType != "" {
		query = query.Where("relation_type = ?", relationType)
	}

	var (
		items []models.ResourceRelation
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

func (h *Handler) CreateResourceRelation(c *gin.Context) {
	var req struct {
		FromCIID          string                 `json:"fromCiId"`
		FromCIIDSnake     string                 `json:"from_ci_id"`
		ToCIID            string                 `json:"toCiId"`
		ToCIIDSnake       string                 `json:"to_ci_id"`
		RelationType      string                 `json:"relationType"`
		RelationTypeSnake string                 `json:"relation_type"`
		Direction         string                 `json:"direction"`
		Criticality       string                 `json:"criticality"`
		Confidence        *float64               `json:"confidence"`
		Evidence          map[string]interface{} `json:"evidence"`
	}
	if !bindJSON(c, &req) {
		return
	}
	fromCIID := strings.TrimSpace(firstNonEmpty(req.FromCIID, req.FromCIIDSnake))
	toCIID := strings.TrimSpace(firstNonEmpty(req.ToCIID, req.ToCIIDSnake))
	relationType := strings.TrimSpace(firstNonEmpty(req.RelationType, req.RelationTypeSnake))
	if fromCIID == "" || toCIID == "" || relationType == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "fromCiId, toCiId, relationType are required"))
		return
	}
	if _, ok := cmdbRelationTypes[relationType]; !ok {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "unsupported relation type"))
		return
	}
	if req.Confidence != nil && (*req.Confidence < 0 || *req.Confidence > 1) {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "confidence must be between 0 and 1"))
		return
	}
	if !h.cmdbResourceExists(fromCIID) || !h.cmdbResourceExists(toCIID) {
		response.Error(c, http.StatusBadRequest, appErr.New(4009, "fromCiId or toCiId does not exist"))
		return
	}

	confidence := 1.0
	if req.Confidence != nil {
		confidence = *req.Confidence
	}
	relation := models.ResourceRelation{
		FromCIID:          fromCIID,
		ToCIID:            toCIID,
		RelationType:      relationType,
		Direction:         defaultString(strings.TrimSpace(req.Direction), "outbound"),
		Criticality:       defaultString(strings.TrimSpace(req.Criticality), "P2"),
		Confidence:        confidence,
		Evidence:          toJSONMap(req.Evidence),
		RelationUpdatedAt: time.Now(),
	}
	if relation.Evidence == nil {
		relation.Evidence = datatypes.JSONMap{}
	}
	if _, exists := relation.Evidence["source"]; !exists {
		relation.Evidence["source"] = "Manual"
	}

	saved, action, err := h.upsertCMDBRelation(relation)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{"action": action, "relation": saved})
}

func (h *Handler) GetResourceUpstream(c *gin.Context) {
	h.getResourceGraph(c, "upstream")
}

func (h *Handler) GetResourceDownstream(c *gin.Context) {
	h.getResourceGraph(c, "downstream")
}

func (h *Handler) GetApplicationTopology(c *gin.Context) {
	application := strings.TrimSpace(c.Param("application"))
	if application == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "application is required"))
		return
	}
	depth := parseDepth(c, 2)
	var seeds []models.ResourceItem
	if err := h.DB.Where("name = ?", application).Find(&seeds).Error; err != nil {
		response.Internal(c, err)
		return
	}
	if len(seeds) == 0 {
		response.Success(c, gin.H{
			"application": application,
			"nodes":       []models.ResourceItem{},
			"relations":   []models.ResourceRelation{},
		})
		return
	}
	seedCIIDs := make([]string, 0, len(seeds))
	for _, item := range seeds {
		seedCIIDs = append(seedCIIDs, item.CIID)
	}
	nodes, relations, err := h.traverseCMDBGraph(seedCIIDs, "both", depth)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{
		"application": application,
		"seedNodes":   seeds,
		"nodes":       nodes,
		"relations":   relations,
	})
}

func (h *Handler) GetResourceImpact(c *gin.Context) {
	ciID := strings.TrimSpace(c.Param("ciId"))
	if ciID == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "ciId is required"))
		return
	}
	var root models.ResourceItem
	if err := h.DB.Where("ci_id = ?", ciID).First(&root).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	depth := parseDepth(c, 4)
	nodes, relations, err := h.traverseCMDBGraph([]string{root.CIID}, "downstream", depth)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{
		"root":          root,
		"impactedNodes": nodes,
		"relations":     relations,
		"impactCount":   len(nodes) - 1,
	})
}

func (h *Handler) GetRegionFailover(c *gin.Context) {
	region := strings.TrimSpace(c.Param("region"))
	if region == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "region is required"))
		return
	}
	var deployed []models.ResourceRelation
	if err := h.DB.Where("relation_type = ?", "deployed_on").Find(&deployed).Error; err != nil {
		response.Internal(c, err)
		return
	}
	if len(deployed) == 0 {
		response.Success(c, gin.H{"region": region, "services": []gin.H{}})
		return
	}

	ciSet := make(map[string]struct{}, len(deployed)*2)
	for _, edge := range deployed {
		ciSet[edge.FromCIID] = struct{}{}
		ciSet[edge.ToCIID] = struct{}{}
	}
	ciIDs := make([]string, 0, len(ciSet))
	for ciID := range ciSet {
		ciIDs = append(ciIDs, ciID)
	}
	resourceMap, err := h.loadCMDBResourcesMap(ciIDs)
	if err != nil {
		response.Internal(c, err)
		return
	}

	serviceRegions := map[string]map[string]struct{}{}
	for _, edge := range deployed {
		target, ok := resourceMap[edge.ToCIID]
		if !ok {
			continue
		}
		if _, ok := serviceRegions[edge.FromCIID]; !ok {
			serviceRegions[edge.FromCIID] = map[string]struct{}{}
		}
		serviceRegions[edge.FromCIID][target.Region] = struct{}{}
	}

	result := make([]gin.H, 0)
	for serviceCIID, regions := range serviceRegions {
		if _, affected := regions[region]; !affected {
			continue
		}
		service := resourceMap[serviceCIID]
		takeoverRegions := make([]string, 0)
		for other := range regions {
			if other == "" || other == region {
				continue
			}
			takeoverRegions = append(takeoverRegions, other)
		}
		sort.Strings(takeoverRegions)
		result = append(result, gin.H{
			"serviceCiId":     serviceCIID,
			"serviceName":     service.Name,
			"affectedRegion":  region,
			"takeoverRegions": takeoverRegions,
			"canFailover":     len(takeoverRegions) > 0,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return fmt.Sprint(result[i]["serviceName"]) < fmt.Sprint(result[j]["serviceName"])
	})
	response.Success(c, gin.H{"region": region, "services": result})
}

func (h *Handler) GetChangeImpact(c *gin.Context) {
	releaseID := strings.TrimSpace(c.Param("releaseId"))
	if releaseID == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "releaseId is required"))
		return
	}
	var release models.ResourceItem
	err := h.DB.Where("ci_id = ? OR name = ?", releaseID, releaseID).First(&release).Error
	if err != nil {
		if id, parseErr := strconv.ParseUint(releaseID, 10, 64); parseErr == nil {
			err = h.DB.First(&release, uint(id)).Error
		}
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	nodes, relations, err := h.traverseCMDBGraph([]string{release.CIID}, "both", 2)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{
		"release":   release,
		"nodes":     nodes,
		"relations": relations,
	})
}

func (h *Handler) CreateCMDBSyncJob(c *gin.Context) {
	var req struct {
		Sources  []string `json:"sources"`
		FullScan bool     `json:"fullScan"`
	}
	if c.Request.ContentLength > 0 && !bindJSON(c, &req) {
		return
	}
	releaseLock, acquired, lockErr := h.tryAcquireCMDBSyncLock()
	if lockErr != nil {
		response.Internal(c, lockErr)
		return
	}
	if !acquired {
		response.Error(c, http.StatusConflict, appErr.New(4013, cmdbSyncRunningMessage))
		return
	}
	defer releaseLock()
	sources := normalizeSourceList(req.Sources)
	now := time.Now()
	requested, _ := json.Marshal(sources)
	job := models.ResourceSyncJob{
		Status:           "running",
		RequestedSources: requested,
		FullScan:         req.FullScan,
		StartedAt:        &now,
		Summary:          datatypes.JSONMap{},
	}
	if err := h.DB.Create(&job).Error; err != nil {
		response.Internal(c, err)
		return
	}

	summary, runErr := h.runCMDBSync(job.ID, sources, req.FullScan)
	finished := time.Now()
	status := "success"
	if runErr != nil {
		status = "failed"
		summary["error"] = h.cloudProviderExternalError("cmdb sync job failed", runErr)
	}
	if err := h.DB.Model(&models.ResourceSyncJob{}).Where("id = ?", job.ID).Updates(map[string]interface{}{
		"status":      status,
		"finished_at": &finished,
		"summary":     summary,
	}).Error; err != nil {
		response.Internal(c, err)
		return
	}
	var saved models.ResourceSyncJob
	if err := h.DB.First(&saved, job.ID).Error; err != nil {
		response.Internal(c, err)
		return
	}
	_, _ = h.PublishNotification(NotificationOptions{
		Module:       "cmdb",
		Source:       "cmdb-sync",
		Event:        "cmdb.sync." + status,
		Severity:     syncStatusSeverity(status),
		ResourceType: "cmdbSyncJob",
		ResourceID:   strconv.FormatUint(uint64(job.ID), 10),
		Title:        "CMDB 同步" + syncStatusTitle(status),
		Content:      "CMDB 同步任务 #" + strconv.FormatUint(uint64(job.ID), 10) + " 状态：" + status,
		Data: gin.H{
			"jobId":   job.ID,
			"status":  status,
			"summary": summary,
		},
	})
	response.Success(c, saved)
}

func (h *Handler) GetCMDBSyncJob(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var job models.ResourceSyncJob
	if err := h.DB.First(&job, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	var items []models.ResourceSyncJobItem
	if err := h.DB.Where("job_id = ?", id).Order("id asc").Limit(500).Find(&items).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{
		"job":   job,
		"items": items,
	})
}

func (h *Handler) RetryCMDBSyncJob(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var original models.ResourceSyncJob
	if err := h.DB.First(&original, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	releaseLock, acquired, lockErr := h.tryAcquireCMDBSyncLock()
	if lockErr != nil {
		response.Internal(c, lockErr)
		return
	}
	if !acquired {
		response.Error(c, http.StatusConflict, appErr.New(4013, cmdbSyncRunningMessage))
		return
	}
	defer releaseLock()
	var sources []string
	if len(original.RequestedSources) > 0 {
		_ = json.Unmarshal(original.RequestedSources, &sources)
	}
	sources = normalizeSourceList(sources)
	now := time.Now()
	requested, _ := json.Marshal(sources)
	retryJob := models.ResourceSyncJob{
		Status:           "running",
		RequestedSources: requested,
		FullScan:         original.FullScan,
		StartedAt:        &now,
		Summary:          datatypes.JSONMap{"retryFromJobId": id},
	}
	if err := h.DB.Create(&retryJob).Error; err != nil {
		response.Internal(c, err)
		return
	}
	summary, runErr := h.runCMDBSync(retryJob.ID, sources, retryJob.FullScan)
	finished := time.Now()
	status := "success"
	if runErr != nil {
		status = "failed"
		summary["error"] = h.cloudProviderExternalError("cmdb sync job failed", runErr)
	}
	if err := h.DB.Model(&models.ResourceSyncJob{}).Where("id = ?", retryJob.ID).Updates(map[string]interface{}{
		"status":      status,
		"finished_at": &finished,
		"summary":     summary,
	}).Error; err != nil {
		response.Internal(c, err)
		return
	}
	var saved models.ResourceSyncJob
	if err := h.DB.First(&saved, retryJob.ID).Error; err != nil {
		response.Internal(c, err)
		return
	}
	_, _ = h.PublishNotification(NotificationOptions{
		Module:       "cmdb",
		Source:       "cmdb-sync",
		Event:        "cmdb.sync.retry." + status,
		Severity:     syncStatusSeverity(status),
		ResourceType: "cmdbSyncJob",
		ResourceID:   strconv.FormatUint(uint64(retryJob.ID), 10),
		Title:        "CMDB 重试同步" + syncStatusTitle(status),
		Content:      "CMDB 重试同步任务 #" + strconv.FormatUint(uint64(retryJob.ID), 10) + " 状态：" + status,
		Data: gin.H{
			"jobId":          retryJob.ID,
			"retryFromJobId": id,
			"status":         status,
			"summary":        summary,
		},
	})
	response.Success(c, saved)
}

func syncStatusSeverity(status string) string {
	if strings.EqualFold(status, "success") {
		return "success"
	}
	return "error"
}

func syncStatusTitle(status string) string {
	if strings.EqualFold(status, "success") {
		return "完成"
	}
	return "失败"
}

func (h *Handler) getResourceGraph(c *gin.Context, direction string) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var root models.ResourceItem
	if err := h.DB.First(&root, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	depth := parseDepth(c, 3)
	nodes, relations, err := h.traverseCMDBGraph([]string{root.CIID}, direction, depth)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{
		"root":      root,
		"nodes":     nodes,
		"relations": relations,
	})
}

func (h *Handler) traverseCMDBGraph(seedCIIDs []string, direction string, depth int) ([]models.ResourceItem, []models.ResourceRelation, error) {
	if depth < 1 {
		depth = 1
	}
	visited := make(map[string]struct{}, len(seedCIIDs))
	frontier := make([]string, 0, len(seedCIIDs))
	for _, ciID := range seedCIIDs {
		if ciID == "" {
			continue
		}
		if _, ok := visited[ciID]; ok {
			continue
		}
		visited[ciID] = struct{}{}
		frontier = append(frontier, ciID)
	}
	edgeMap := map[uint]models.ResourceRelation{}

	for i := 0; i < depth && len(frontier) > 0; i++ {
		var rels []models.ResourceRelation
		query := h.DB.Model(&models.ResourceRelation{})
		switch direction {
		case "upstream":
			query = query.Where("to_ci_id IN ?", frontier)
		case "downstream":
			query = query.Where("from_ci_id IN ?", frontier)
		default:
			query = query.Where("from_ci_id IN ? OR to_ci_id IN ?", frontier, frontier)
		}
		if err := query.Find(&rels).Error; err != nil {
			return nil, nil, err
		}
		next := make([]string, 0, len(rels))
		for _, rel := range rels {
			edgeMap[rel.ID] = rel
			candidates := make([]string, 0, 2)
			switch direction {
			case "upstream":
				candidates = append(candidates, rel.FromCIID)
			case "downstream":
				candidates = append(candidates, rel.ToCIID)
			default:
				candidates = append(candidates, rel.FromCIID, rel.ToCIID)
			}
			for _, ciID := range candidates {
				if ciID == "" {
					continue
				}
				if _, ok := visited[ciID]; ok {
					continue
				}
				visited[ciID] = struct{}{}
				next = append(next, ciID)
			}
		}
		frontier = next
	}

	ciIDs := make([]string, 0, len(visited))
	for ciID := range visited {
		ciIDs = append(ciIDs, ciID)
	}
	nodes := make([]models.ResourceItem, 0, len(ciIDs))
	if len(ciIDs) > 0 {
		if err := h.DB.Where("ci_id IN ?", ciIDs).Find(&nodes).Error; err != nil {
			return nil, nil, err
		}
	}
	relations := make([]models.ResourceRelation, 0, len(edgeMap))
	for _, edge := range edgeMap {
		relations = append(relations, edge)
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	sort.Slice(relations, func(i, j int) bool { return relations[i].ID < relations[j].ID })
	return nodes, relations, nil
}

func (h *Handler) runCMDBSync(jobID uint, sources []string, fullScan bool) (datatypes.JSONMap, error) {
	summary := datatypes.JSONMap{
		"jobId":       jobID,
		"sources":     sources,
		"fullScan":    fullScan,
		"created":     0,
		"updated":     0,
		"refreshed":   0,
		"failed":      0,
		"relations":   0,
		"startTime":   time.Now().Format(time.RFC3339),
		"warnings":    []string{},
		"qualityLow":  0,
		"processedCi": 0,
	}
	var runErrs []string

	for _, source := range sources {
		switch source {
		case "IaC":
			candidates := h.collectIaCCandidates()
			h.applyResourceCandidates(jobID, candidates, summary, &runErrs)
		case "CloudAPI":
			candidates, warnings := h.collectCloudCandidates()
			appendWarnings(summary, warnings)
			h.applyResourceCandidates(jobID, candidates, summary, &runErrs)
		case "K8s":
			candidates := h.collectK8sCandidates()
			h.applyResourceCandidates(jobID, candidates, summary, &runErrs)
		case "APM":
			relationCandidates := h.collectAPMRelationCandidates()
			for _, candidate := range relationCandidates {
				relation := candidate.Relation
				relation.Direction = defaultString(relation.Direction, "outbound")
				relation.Criticality = defaultString(relation.Criticality, "P2")
				relation.RelationUpdatedAt = time.Now()
				if relation.Evidence == nil {
					relation.Evidence = datatypes.JSONMap{}
				}
				relation.Evidence["source"] = candidate.Source
				_, action, err := h.upsertCMDBRelation(relation)
				if err != nil {
					runErrs = append(runErrs, err.Error())
					summary["failed"] = asInt(summary["failed"]) + 1
					_ = h.DB.Create(&models.ResourceSyncJobItem{
						JobID:   jobID,
						CIID:    relation.FromCIID + "->" + relation.ToCIID,
						Source:  source,
						Action:  "relation_upsert",
						Status:  "failed",
						Message: h.cloudProviderExternalWarning("cmdb relation upsert failed", err),
					}).Error
					continue
				}
				summary["relations"] = asInt(summary["relations"]) + 1
				_ = h.DB.Create(&models.ResourceSyncJobItem{
					JobID:   jobID,
					CIID:    relation.FromCIID + "->" + relation.ToCIID,
					Source:  source,
					Action:  action,
					Status:  "success",
					Message: defaultString(candidate.Message, "relation built"),
					Data:    datatypes.JSONMap{"relationType": relation.RelationType},
				}).Error
			}
		case "Manual":
		default:
			appendWarnings(summary, []string{"unsupported source ignored: " + source})
		}
	}

	summary["endTime"] = time.Now().Format(time.RFC3339)
	if len(runErrs) > 0 {
		return summary, errors.New(strings.Join(runErrs, "; "))
	}
	return summary, nil
}

func (h *Handler) applyResourceCandidates(jobID uint, candidates []cmdbResourceCandidate, summary datatypes.JSONMap, runErrs *[]string) {
	for _, candidate := range candidates {
		resource := normalizeCMDBResource(candidate.Resource, candidate.Source)
		saved, action, err := h.upsertCMDBResource(resource)
		if err != nil {
			*runErrs = append(*runErrs, err.Error())
			summary["failed"] = asInt(summary["failed"]) + 1
			_ = h.DB.Create(&models.ResourceSyncJobItem{
				JobID:   jobID,
				CIID:    resource.CIID,
				Source:  normalizeCMDBSource(candidate.Source),
				Action:  "upsert",
				Status:  "failed",
				Message: h.cloudProviderExternalWarning("cmdb resource upsert failed", err),
			}).Error
			continue
		}
		source := normalizeCMDBSource(candidate.Source)
		quality := resourceQualityScore(saved)
		if quality < 1 {
			summary["qualityLow"] = asInt(summary["qualityLow"]) + 1
		}
		summary[action] = asInt(summary[action]) + 1
		summary["processedCi"] = asInt(summary["processedCi"]) + 1
		_ = h.DB.Create(&models.ResourceEvidence{
			CIID:       saved.CIID,
			Source:     source,
			RawID:      candidate.RawID,
			Payload:    candidate.Payload,
			ObservedAt: time.Now(),
		}).Error
		_ = h.DB.Create(&models.ResourceSyncJobItem{
			JobID:        jobID,
			CIID:         saved.CIID,
			Source:       source,
			Action:       action,
			Status:       "success",
			Message:      "resource upserted",
			QualityScore: quality,
			Data:         datatypes.JSONMap{"resourceId": saved.ID, "type": saved.Type},
		}).Error
	}
}

func (h *Handler) collectIaCCandidates() []cmdbResourceCandidate {
	now := time.Now()
	return []cmdbResourceCandidate{
		{
			Resource: models.ResourceItem{
				Type:       "Pipeline",
				Name:       "iac-bootstrap",
				Cloud:      "multi-cloud",
				Region:     "global",
				Env:        "prod",
				Owner:      "platform",
				Lifecycle:  "active",
				Source:     "IaC",
				LastSeenAt: now,
				Attributes: datatypes.JSONMap{"tool": "terraform", "state": "managed"},
			},
			RawID:   "iac:pipeline:iac-bootstrap",
			Source:  "IaC",
			Payload: datatypes.JSONMap{"collector": "iac"},
		},
	}
}

func (h *Handler) collectCloudCandidates() ([]cmdbResourceCandidate, []string) {
	var accounts []models.CloudAccount
	if err := h.DB.Find(&accounts).Error; err != nil {
		return nil, []string{h.cloudProviderExternalWarning("load cloud accounts failed", err)}
	}
	now := time.Now()
	candidates := make([]cmdbResourceCandidate, 0)
	warnings := make([]string, 0)
	for _, account := range accounts {
		provider, providerErr := h.cloudProviderByAccount(account)
		if providerErr != nil {
			warnings = append(warnings, h.cloudProviderExternalWarning(fmt.Sprintf("cloud account %d provider error", account.ID), providerErr))
			continue
		}
		cred, credErr := h.cloudAccountCredentials(&account)
		if credErr != nil {
			warnings = append(warnings, h.cloudProviderExternalWarning(fmt.Sprintf("cloud account %d credential error", account.ID), credErr))
			continue
		}
		assets, err := h.collectCloudProviderAssets(provider, cred)
		if err != nil {
			warnings = append(warnings, h.cloudProviderExternalWarning(fmt.Sprintf("cloud account %d sync failed", account.ID), err))
			continue
		}
		for _, asset := range assets {
			resourceType := mapCloudAssetToCIType(asset.Type)
			attributes := buildCloudResourceAttributes(account, asset)
			resource := models.ResourceItem{
				Type:       resourceType,
				Name:       defaultString(asset.Name, asset.ID),
				Cloud:      defaultString(asset.Provider, account.Provider),
				Region:     defaultString(asset.Region, account.Region),
				Env:        "prod",
				Owner:      "cloud-" + account.Provider,
				Lifecycle:  "active",
				Source:     "CloudAPI",
				LastSeenAt: now,
				Attributes: attributes,
			}
			resource.CIID = buildCMDBCIID(resource)
			if resource.CIID == "" {
				resource.CIID = fmt.Sprintf("%s:%d:%s:%s", normalizeToken(account.Provider), account.ID, normalizeToken(asset.Type), normalizeToken(asset.ID))
			}
			candidates = append(candidates, cmdbResourceCandidate{
				Resource: resource,
				RawID:    asset.ID,
				Source:   "CloudAPI",
				Payload:  datatypes.JSONMap{"asset": asset},
			})
		}
	}
	return candidates, warnings
}

func (h *Handler) collectK8sCandidates() []cmdbResourceCandidate {
	var clusters []models.KubernetesCluster
	if err := h.DB.Find(&clusters).Error; err != nil {
		return nil
	}
	now := time.Now()
	candidates := make([]cmdbResourceCandidate, 0, len(clusters)*2)
	for _, cluster := range clusters {
		clusterResource := models.ResourceItem{
			Type:       "K8sCluster",
			Name:       cluster.Name,
			Cloud:      "k8s",
			Region:     "global",
			Env:        "prod",
			Owner:      "platform",
			Lifecycle:  "active",
			Source:     "K8s",
			LastSeenAt: now,
			Attributes: datatypes.JSONMap{
				"apiServer":  cluster.APIServer,
				"kubeconfig": maskKubeConfig(cluster.KubeConfig),
			},
		}
		clusterResource.CIID = buildCMDBCIID(clusterResource)
		candidates = append(candidates, cmdbResourceCandidate{
			Resource: clusterResource,
			RawID:    cluster.Name,
			Source:   "K8s",
			Payload:  datatypes.JSONMap{"clusterId": cluster.ID},
		})

		namespaces := extractNamespacesFromKubeConfig(cluster.KubeConfig)
		if len(namespaces) == 0 {
			namespaces = []string{"default"}
		}
		for _, namespace := range namespaces {
			nsResource := models.ResourceItem{
				Type:       "Namespace",
				Name:       namespace,
				Cloud:      "k8s",
				Region:     "global",
				Env:        "prod",
				Owner:      "platform",
				Lifecycle:  "active",
				Source:     "K8s",
				LastSeenAt: now,
				Attributes: datatypes.JSONMap{
					"cluster": cluster.Name,
					"kind":    "Namespace",
					"name":    namespace,
				},
			}
			nsResource.CIID = buildCMDBCIID(nsResource)
			candidates = append(candidates, cmdbResourceCandidate{
				Resource: nsResource,
				RawID:    cluster.Name + ":" + namespace,
				Source:   "K8s",
				Payload:  datatypes.JSONMap{"clusterId": cluster.ID, "namespace": namespace},
			})
		}
	}
	return candidates
}

func (h *Handler) collectAPMRelationCandidates() []cmdbRelationCandidate {
	var services []models.ResourceItem
	if err := h.DB.Where("type = ?", "Service").Find(&services).Error; err != nil {
		return nil
	}
	candidates := make([]cmdbRelationCandidate, 0)
	for _, service := range services {
		targets := readStringSliceFromAttributes(service.Attributes, "connectsToCIIDs", "connects_to_ci_ids", "dependsOn")
		for _, target := range targets {
			if strings.TrimSpace(target) == "" {
				continue
			}
			candidates = append(candidates, cmdbRelationCandidate{
				Relation: models.ResourceRelation{
					FromCIID:     service.CIID,
					ToCIID:       target,
					RelationType: "connects_to",
					Direction:    "outbound",
					Criticality:  "P1",
					Confidence:   0.6,
					Evidence:     datatypes.JSONMap{"service": service.Name},
				},
				Source:  "APM",
				Message: "inferred from service attributes",
			})
		}
	}
	return candidates
}

func (h *Handler) syncCloudResourcesToCMDB(account models.CloudAccount, assets []cloud.Asset) ([]models.ResourceItem, error) {
	if len(assets) == 0 {
		return []models.ResourceItem{}, nil
	}

	now := time.Now()
	latestByCIID := make(map[string]models.ResourceItem, len(assets))
	orderedCIIDs := make([]string, 0, len(assets))

	for _, asset := range assets {
		resource := models.ResourceItem{
			Type:       mapCloudAssetToCIType(asset.Type),
			Name:       defaultString(asset.Name, asset.ID),
			Cloud:      defaultString(asset.Provider, account.Provider),
			Region:     defaultString(asset.Region, account.Region),
			Env:        "prod",
			Owner:      "cloud-" + account.Provider,
			Lifecycle:  "active",
			Source:     "CloudAPI",
			LastSeenAt: now,
			Attributes: buildCloudResourceAttributes(account, asset),
		}
		resource.CIID = buildCMDBCIID(resource)
		if resource.CIID == "" {
			resource.CIID = fmt.Sprintf("%s:%d:%s:%s", normalizeToken(account.Provider), account.ID, normalizeToken(asset.Type), normalizeToken(asset.ID))
		}
		if _, exists := latestByCIID[resource.CIID]; !exists {
			orderedCIIDs = append(orderedCIIDs, resource.CIID)
		}
		latestByCIID[resource.CIID] = normalizeCMDBResource(resource, resource.Source)
	}

	existingMap, err := h.loadCMDBResourcesMap(orderedCIIDs)
	if err != nil {
		return nil, err
	}

	createItems := make([]models.ResourceItem, 0)
	type cmdbResourceUpdateOp struct {
		ID      uint
		Updates map[string]interface{}
	}
	updateOps := make([]cmdbResourceUpdateOp, 0)

	for _, ciID := range orderedCIIDs {
		input := latestByCIID[ciID]
		existing, exists := existingMap[ciID]
		if !exists {
			createItems = append(createItems, input)
			continue
		}

		newPriority := sourcePriority(input.Source)
		oldPriority := sourcePriority(existing.Source)
		updates := map[string]interface{}{
			"last_seen_at": input.LastSeenAt,
		}
		if newPriority >= oldPriority {
			updates["type"] = defaultString(input.Type, existing.Type)
			updates["name"] = defaultString(input.Name, existing.Name)
			updates["category_id"] = input.CategoryID
			updates["cloud"] = defaultString(input.Cloud, existing.Cloud)
			updates["region"] = defaultString(input.Region, existing.Region)
			updates["env"] = defaultString(input.Env, existing.Env)
			updates["owner"] = defaultString(input.Owner, existing.Owner)
			updates["lifecycle"] = defaultString(input.Lifecycle, existing.Lifecycle)
			updates["source"] = normalizeCMDBSource(input.Source)
			updates["attributes"] = mergeJSONMap(existing.Attributes, input.Attributes, true)
		} else {
			updates["attributes"] = mergeJSONMap(existing.Attributes, input.Attributes, false)
		}
		updateOps = append(updateOps, cmdbResourceUpdateOp{
			ID:      existing.ID,
			Updates: updates,
		})
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		const batchSize = 200
		if len(createItems) > 0 {
			for start := 0; start < len(createItems); start += batchSize {
				end := start + batchSize
				if end > len(createItems) {
					end = len(createItems)
				}
				chunk := createItems[start:end]
				if err := tx.Create(&chunk).Error; err != nil {
					return err
				}
			}
		}
		for _, op := range updateOps {
			if err := tx.Model(&models.ResourceItem{}).Where("id = ?", op.ID).Updates(op.Updates).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	savedMap, err := h.loadCMDBResourcesMap(orderedCIIDs)
	if err != nil {
		return nil, err
	}
	savedResources := make([]models.ResourceItem, 0, len(orderedCIIDs))
	for _, ciID := range orderedCIIDs {
		saved, ok := savedMap[ciID]
		if !ok {
			continue
		}
		savedResources = append(savedResources, saved)
	}
	return savedResources, nil
}

func (h *Handler) upsertCMDBResource(input models.ResourceItem) (models.ResourceItem, string, error) {
	input = normalizeCMDBResource(input, input.Source)
	if input.CIID == "" {
		return models.ResourceItem{}, "", fmt.Errorf("ciId is empty")
	}
	var existing models.ResourceItem
	query := h.DB.Where("ci_id = ?", input.CIID).Limit(1).Find(&existing)
	if query.Error != nil {
		return models.ResourceItem{}, "", query.Error
	}
	if query.RowsAffected == 0 {
		if err := h.DB.Create(&input).Error; err != nil {
			return models.ResourceItem{}, "", err
		}
		return input, "created", nil
	}

	newPriority := sourcePriority(input.Source)
	oldPriority := sourcePriority(existing.Source)
	updates := map[string]interface{}{
		"last_seen_at": input.LastSeenAt,
	}
	action := "refreshed"
	if newPriority >= oldPriority {
		updates["type"] = defaultString(input.Type, existing.Type)
		updates["name"] = defaultString(input.Name, existing.Name)
		updates["category_id"] = input.CategoryID
		updates["cloud"] = defaultString(input.Cloud, existing.Cloud)
		updates["region"] = defaultString(input.Region, existing.Region)
		updates["env"] = defaultString(input.Env, existing.Env)
		updates["owner"] = defaultString(input.Owner, existing.Owner)
		updates["lifecycle"] = defaultString(input.Lifecycle, existing.Lifecycle)
		updates["source"] = normalizeCMDBSource(input.Source)
		updates["attributes"] = mergeJSONMap(existing.Attributes, input.Attributes, true)
		action = "updated"
	} else {
		updates["attributes"] = mergeJSONMap(existing.Attributes, input.Attributes, false)
	}
	if err := h.DB.Model(&models.ResourceItem{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
		return models.ResourceItem{}, "", err
	}
	var saved models.ResourceItem
	if err := h.DB.First(&saved, existing.ID).Error; err != nil {
		return models.ResourceItem{}, "", err
	}
	return saved, action, nil
}

func (h *Handler) upsertCMDBRelation(input models.ResourceRelation) (models.ResourceRelation, string, error) {
	var existing models.ResourceRelation
	query := h.DB.Where("from_ci_id = ? AND to_ci_id = ? AND relation_type = ?", input.FromCIID, input.ToCIID, input.RelationType).
		Limit(1).
		Find(&existing)
	if query.Error != nil {
		return models.ResourceRelation{}, "", query.Error
	}
	if query.RowsAffected == 0 {
		if err := h.DB.Create(&input).Error; err != nil {
			return models.ResourceRelation{}, "", err
		}
		return input, "created", nil
	}

	newPriority := sourcePriority(readRelationSource(input))
	oldPriority := sourcePriority(readRelationSource(existing))
	updates := map[string]interface{}{
		"relation_updated_at": input.RelationUpdatedAt,
	}
	action := "refreshed"
	if newPriority >= oldPriority {
		updates["direction"] = input.Direction
		updates["criticality"] = input.Criticality
		updates["confidence"] = input.Confidence
		updates["evidence"] = mergeJSONMap(existing.Evidence, input.Evidence, true)
		action = "updated"
	} else {
		updates["evidence"] = mergeJSONMap(existing.Evidence, input.Evidence, false)
	}
	if err := h.DB.Model(&models.ResourceRelation{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
		return models.ResourceRelation{}, "", err
	}
	var saved models.ResourceRelation
	if err := h.DB.First(&saved, existing.ID).Error; err != nil {
		return models.ResourceRelation{}, "", err
	}
	return saved, action, nil
}

func (h *Handler) cmdbResourceExists(ciID string) bool {
	var count int64
	if err := h.DB.Model(&models.ResourceItem{}).Where("ci_id = ?", ciID).Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func (h *Handler) loadCMDBResourcesMap(ciIDs []string) (map[string]models.ResourceItem, error) {
	result := make(map[string]models.ResourceItem, len(ciIDs))
	if len(ciIDs) == 0 {
		return result, nil
	}
	var items []models.ResourceItem
	if err := h.DB.Where("ci_id IN ?", ciIDs).Find(&items).Error; err != nil {
		return nil, err
	}
	for _, item := range items {
		result[item.CIID] = item
	}
	return result, nil
}

func normalizeCMDBResource(input models.ResourceItem, fallbackSource string) models.ResourceItem {
	input.Source = normalizeCMDBSource(defaultString(input.Source, fallbackSource))
	input.Env = defaultString(input.Env, "prod")
	input.Lifecycle = defaultString(input.Lifecycle, "active")
	if input.LastSeenAt.IsZero() {
		input.LastSeenAt = time.Now()
	}
	if input.CIID == "" {
		input.CIID = buildCMDBCIID(input)
	}
	return input
}

func normalizeCMDBSource(source string) string {
	normalized := strings.TrimSpace(source)
	switch strings.ToLower(normalized) {
	case "iac":
		return "IaC"
	case "cloudapi", "cloud_api", "cloud":
		return "CloudAPI"
	case "k8s", "kubernetes":
		return "K8s"
	case "apm", "tracing":
		return "APM"
	case "manual":
		return "Manual"
	default:
		if normalized == "" {
			return "Manual"
		}
		return normalized
	}
}

func normalizeSourceList(sources []string) []string {
	if len(sources) == 0 {
		return []string{"IaC", "CloudAPI", "K8s", "APM", "Manual"}
	}
	uniq := make(map[string]struct{}, len(sources))
	list := make([]string, 0, len(sources))
	for _, source := range sources {
		normalized := normalizeCMDBSource(source)
		if _, exists := uniq[normalized]; exists {
			continue
		}
		uniq[normalized] = struct{}{}
		list = append(list, normalized)
	}
	return list
}

func (h *Handler) cmdbSyncRunning() (bool, error) {
	var runningCount int64
	if err := h.DB.Model(&models.ResourceSyncJob{}).Where("lower(status) = ?", "running").Count(&runningCount).Error; err != nil {
		return false, err
	}
	return runningCount > 0, nil
}

func buildCMDBCIID(item models.ResourceItem) string {
	itemType := strings.TrimSpace(strings.ToLower(item.Type))
	cloudName := normalizeToken(item.Cloud)
	region := normalizeToken(item.Region)
	attrs := item.Attributes

	if itemType == "vm" {
		account := normalizeToken(readStringAttr(attrs, "accountId", "account_id", "account"))
		instanceID := normalizeToken(readStringAttr(attrs, "instanceId", "instance_id", "assetId", "asset_id"))
		if cloudName != "" && account != "" && region != "" && instanceID != "" {
			return strings.Join([]string{cloudName, account, region, instanceID}, ":")
		}
	}

	if isK8sWorkload(itemType) {
		cluster := normalizeToken(readStringAttr(attrs, "cluster", "clusterName", "cluster_name"))
		namespace := normalizeToken(readStringAttr(attrs, "namespace"))
		kind := normalizeToken(readStringAttr(attrs, "kind"))
		name := normalizeToken(readStringAttr(attrs, "name"))
		if cluster != "" && namespace != "" && kind != "" && name != "" {
			return strings.Join([]string{
				defaultString(cloudName, "k8s"),
				defaultString(region, "global"),
				cluster,
				namespace,
				kind,
				name,
			}, ":")
		}
	}

	if isManagedDB(itemType) {
		account := normalizeToken(readStringAttr(attrs, "accountId", "account_id", "account"))
		engine := normalizeToken(readStringAttr(attrs, "engine"))
		instanceID := normalizeToken(readStringAttr(attrs, "instanceId", "instance_id", "assetId", "asset_id"))
		if cloudName != "" && account != "" && region != "" && engine != "" && instanceID != "" {
			return strings.Join([]string{cloudName, account, region, engine, instanceID}, ":")
		}
	}

	resourceName := normalizeToken(item.Name)
	resourceType := normalizeToken(item.Type)
	if resourceType == "" || resourceName == "" {
		return ""
	}
	if cloudName != "" && region != "" {
		return strings.Join([]string{cloudName, region, resourceType, resourceName}, ":")
	}
	return strings.Join([]string{"cmdb", resourceType, resourceName}, ":")
}

func readStringAttr(attrs datatypes.JSONMap, keys ...string) string {
	for _, key := range keys {
		if attrs == nil {
			continue
		}
		raw, ok := attrs[key]
		if !ok {
			continue
		}
		switch val := raw.(type) {
		case string:
			if strings.TrimSpace(val) != "" {
				return val
			}
		case fmt.Stringer:
			text := val.String()
			if strings.TrimSpace(text) != "" {
				return text
			}
		default:
			text := fmt.Sprintf("%v", val)
			if strings.TrimSpace(text) != "" && text != "<nil>" {
				return text
			}
		}
	}
	return ""
}

func readStringSliceFromAttributes(attrs datatypes.JSONMap, keys ...string) []string {
	for _, key := range keys {
		raw, ok := attrs[key]
		if !ok {
			continue
		}
		switch val := raw.(type) {
		case []string:
			return val
		case []interface{}:
			list := make([]string, 0, len(val))
			for _, item := range val {
				text := strings.TrimSpace(fmt.Sprintf("%v", item))
				if text != "" && text != "<nil>" {
					list = append(list, text)
				}
			}
			return list
		}
	}
	return nil
}

func mergeJSONMap(existing datatypes.JSONMap, incoming datatypes.JSONMap, override bool) datatypes.JSONMap {
	merged := datatypes.JSONMap{}
	for key, value := range existing {
		merged[key] = value
	}
	for key, value := range incoming {
		if !override {
			if _, exists := merged[key]; exists {
				continue
			}
		}
		merged[key] = value
	}
	return merged
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func normalizeToken(value string) string {
	replacer := strings.NewReplacer(" ", "-", "/", "-", ":", "-", "\t", "-", "\n", "-", "\r", "-")
	return strings.ToLower(strings.Trim(replacer.Replace(value), "-"))
}

func parseDepth(c *gin.Context, fallback int) int {
	depthRaw := strings.TrimSpace(c.Query("depth"))
	if depthRaw == "" {
		return fallback
	}
	depth, err := strconv.Atoi(depthRaw)
	if err != nil || depth < 1 {
		return fallback
	}
	if depth > 8 {
		return 8
	}
	return depth
}

func sourcePriority(source string) int {
	priority, ok := cmdbSourcePriority[normalizeCMDBSource(source)]
	if !ok {
		return 0
	}
	return priority
}

func readRelationSource(relation models.ResourceRelation) string {
	if relation.Evidence == nil {
		return "Manual"
	}
	source, ok := relation.Evidence["source"]
	if !ok {
		return "Manual"
	}
	return normalizeCMDBSource(fmt.Sprintf("%v", source))
}

func asInt(value interface{}) int {
	switch val := value.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	default:
		return 0
	}
}

func appendWarnings(summary datatypes.JSONMap, warnings []string) {
	if len(warnings) == 0 {
		return
	}
	current, ok := summary["warnings"].([]string)
	if !ok {
		current = []string{}
	}
	current = append(current, warnings...)
	summary["warnings"] = current
}

func readCMDBRuntimeStatus(attrs datatypes.JSONMap) string {
	status := strings.ToLower(strings.TrimSpace(readStringAttr(attrs, "status", "state", "runtimeStatus", "runtime_status")))
	if status != "" {
		return status
	}
	if attrs == nil {
		return ""
	}
	rawMetadata, ok := attrs["metadata"]
	if !ok || rawMetadata == nil {
		return ""
	}
	switch metadata := rawMetadata.(type) {
	case map[string]interface{}:
		return strings.ToLower(strings.TrimSpace(readMapString(metadata, "status", "state")))
	case datatypes.JSONMap:
		return strings.ToLower(strings.TrimSpace(readStringAttr(metadata, "status", "state")))
	default:
		return ""
	}
}

func isCMDBRunningStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "running", "active", "starting", "pending", "booting", "provisioning", "initializing", "inservice":
		return true
	default:
		return false
	}
}

func buildCloudResourceAttributes(account models.CloudAccount, asset cloud.Asset) datatypes.JSONMap {
	attributes := datatypes.JSONMap{
		"accountId":   account.ID,
		"accountName": account.Name,
		"assetType":   asset.Type,
		"assetId":     asset.ID,
		"metadata":    asset.Metadata,
	}
	normalized := normalizeCloudResourceMetadata(asset.Metadata)
	for key, value := range normalized {
		attributes[key] = value
	}
	return attributes
}

func normalizeCloudResourceMetadata(metadata map[string]interface{}) datatypes.JSONMap {
	result := datatypes.JSONMap{}
	if len(metadata) == 0 {
		return result
	}

	if cpu := readMapString(metadata, "cpu", "vcpu", "vCpu", "cpuCore", "cpuCores"); cpu != "" {
		result["cpu"] = cpu
	}
	if memory := normalizeMemoryMetadata(metadata); memory != "" {
		result["memory"] = memory
	}
	if disk := readMapString(metadata, "disk", "diskGb", "diskGB", "diskSize"); disk != "" {
		result["disk"] = disk
	}
	if privateIP := readMapString(metadata, "privateIp", "private_ip", "innerIp", "privateIpAddress"); privateIP != "" {
		result["privateIp"] = privateIP
	}
	if publicIP := readMapString(metadata, "publicIp", "public_ip", "eip", "publicIpAddress"); publicIP != "" {
		result["publicIp"] = publicIP
	}
	if operatingSystem := readMapString(metadata, "os", "osName", "image", "imageName"); operatingSystem != "" {
		result["os"] = operatingSystem
	}
	if instanceType := readMapString(metadata, "instanceType", "instance_type", "flavor", "spec"); instanceType != "" {
		result["instanceType"] = instanceType
	}
	if expiresAt := readMapString(metadata, "expiresAt", "expireAt", "expireTime", "expiredAt", "expirationTime"); expiresAt != "" {
		result["expiresAt"] = expiresAt
	}
	return result
}

func readMapString(values map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		raw, ok := values[key]
		if !ok {
			continue
		}
		text := strings.TrimSpace(fmt.Sprintf("%v", raw))
		if text == "" || text == "<nil>" {
			continue
		}
		return text
	}
	return ""
}

func normalizeMemoryMetadata(metadata map[string]interface{}) string {
	if memoryRaw, ok := readMapValue(metadata, "memory", "memoryGb", "memoryGB", "mem"); ok {
		if normalized := normalizeMemoryToGB(memoryRaw, false); normalized != "" {
			return normalized
		}
	}
	if memoryMBRaw, ok := readMapValue(metadata, "memoryMB", "memoryMb", "memMB"); ok {
		if normalized := normalizeMemoryToGB(memoryMBRaw, true); normalized != "" {
			return normalized
		}
	}
	return ""
}

func readMapValue(values map[string]interface{}, keys ...string) (interface{}, bool) {
	for _, key := range keys {
		raw, ok := values[key]
		if !ok || raw == nil {
			continue
		}
		if text, ok := raw.(string); ok {
			if strings.TrimSpace(text) == "" {
				continue
			}
		}
		return raw, true
	}
	return nil, false
}

func normalizeMemoryToGB(raw interface{}, fromMB bool) string {
	if raw == nil {
		return ""
	}
	switch value := raw.(type) {
	case string:
		text := strings.TrimSpace(value)
		if text == "" || text == "<nil>" {
			return ""
		}
		if fromMB {
			normalized := strings.ToLower(strings.ReplaceAll(text, " ", ""))
			for _, suffix := range []string{"mib", "mb", "m"} {
				if strings.HasSuffix(normalized, suffix) {
					normalized = strings.TrimSuffix(normalized, suffix)
					break
				}
			}
			if parsed, err := strconv.ParseFloat(normalized, 64); err == nil {
				return formatMemoryGBText(parsed / 1024)
			}
		}
		if containsAlphabet(text) {
			return text
		}
		if parsed, err := strconv.ParseFloat(text, 64); err == nil {
			if fromMB {
				return formatMemoryGBText(parsed / 1024)
			}
			return formatMemoryGBText(parsed)
		}
		return text
	case int:
		if fromMB {
			return formatMemoryGBText(float64(value) / 1024)
		}
		return formatMemoryGBText(float64(value))
	case int8:
		if fromMB {
			return formatMemoryGBText(float64(value) / 1024)
		}
		return formatMemoryGBText(float64(value))
	case int16:
		if fromMB {
			return formatMemoryGBText(float64(value) / 1024)
		}
		return formatMemoryGBText(float64(value))
	case int32:
		if fromMB {
			return formatMemoryGBText(float64(value) / 1024)
		}
		return formatMemoryGBText(float64(value))
	case int64:
		if fromMB {
			return formatMemoryGBText(float64(value) / 1024)
		}
		return formatMemoryGBText(float64(value))
	case float32:
		if fromMB {
			return formatMemoryGBText(float64(value) / 1024)
		}
		return formatMemoryGBText(float64(value))
	case float64:
		if fromMB {
			return formatMemoryGBText(value / 1024)
		}
		return formatMemoryGBText(value)
	case json.Number:
		if parsed, err := value.Float64(); err == nil {
			if fromMB {
				return formatMemoryGBText(parsed / 1024)
			}
			return formatMemoryGBText(parsed)
		}
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", raw))
		if text == "" || text == "<nil>" {
			return ""
		}
		return normalizeMemoryToGB(text, fromMB)
	}
	return ""
}

func formatMemoryGBText(value float64) string {
	if value <= 0 {
		return ""
	}
	text := strconv.FormatFloat(value, 'f', 2, 64)
	text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
	return text + "G"
}

func containsAlphabet(value string) bool {
	for _, ch := range value {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
			return true
		}
	}
	return false
}

func mapCloudAssetToCIType(assetType string) string {
	switch strings.ToLower(strings.TrimSpace(assetType)) {
	case "compute", "ecs", "ec2", "vm", "cloudserver":
		return "VM"
	case "rds", "postgres", "postgresql":
		return "PostgreSQL"
	case "mysql":
		return "MySQL"
	case "redis":
		return "Redis"
	case "mq", "rabbitmq":
		return "RabbitMQ"
	case "lb", "slb", "loadbalancer":
		return "LB"
	case "dns":
		return "DNS"
	case "objectstorage", "oss", "s3":
		return "ObjectStorage"
	case "filestorage":
		return "FileStorage"
	case "containerservice":
		return "ContainerService"
	case "privatenetwork", "vpc":
		return "VPC"
	case "sslcertificate":
		return "SSLCertificate"
	case "logservice":
		return "LogService"
	default:
		return "CloudResource"
	}
}

func resourceQualityScore(item models.ResourceItem) float64 {
	score := 0.0
	if item.CIID != "" {
		score += 0.2
	}
	if item.Type != "" {
		score += 0.2
	}
	if item.Name != "" {
		score += 0.2
	}
	if item.Source != "" {
		score += 0.2
	}
	if !item.LastSeenAt.IsZero() {
		score += 0.2
	}
	return score
}

func isManagedDB(itemType string) bool {
	switch itemType {
	case "mysql", "postgresql", "postgres", "redis":
		return true
	default:
		return false
	}
}

func isK8sWorkload(itemType string) bool {
	switch itemType {
	case "namespace", "deployment", "statefulset", "daemonset", "service", "pod", "k8scluster":
		return true
	default:
		return false
	}
}

func maskKubeConfig(kubeConfig string) string {
	if strings.TrimSpace(kubeConfig) == "" {
		return ""
	}
	lines := strings.Split(kubeConfig, "\n")
	masked := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), "token:") || strings.Contains(strings.ToLower(trimmed), "client-key-data:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				masked = append(masked, parts[0]+": ******")
				continue
			}
		}
		masked = append(masked, line)
	}
	return strings.Join(masked, "\n")
}

func extractNamespacesFromKubeConfig(kubeConfig string) []string {
	lines := strings.Split(kubeConfig, "\n")
	namespaces := map[string]struct{}{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(strings.ToLower(trimmed), "namespace:") {
			continue
		}
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) < 2 {
			continue
		}
		ns := strings.TrimSpace(parts[1])
		if ns == "" {
			continue
		}
		namespaces[ns] = struct{}{}
	}
	list := make([]string, 0, len(namespaces))
	for ns := range namespaces {
		list = append(list, ns)
	}
	sort.Strings(list)
	return list
}

func toJSONMap(input map[string]interface{}) datatypes.JSONMap {
	if input == nil {
		return datatypes.JSONMap{}
	}
	result := datatypes.JSONMap{}
	for key, value := range input {
		result[key] = value
	}
	return result
}

func asCloudAssetSlice(items []models.ResourceItem) []cloud.Asset {
	assets := make([]cloud.Asset, 0, len(items))
	for _, item := range items {
		assets = append(assets, cloud.Asset{
			Provider: item.Cloud,
			Type:     item.Type,
			ID:       item.CIID,
			Name:     item.Name,
			Region:   item.Region,
			Metadata: item.Attributes,
		})
	}
	return assets
}
