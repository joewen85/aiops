package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	appErr "devops-system/backend/internal/errors"
	"devops-system/backend/internal/executor"
	"devops-system/backend/internal/models"
	"devops-system/backend/internal/response"
	"devops-system/backend/internal/ws"
)

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
func (h *Handler) ListResources(c *gin.Context)  { listByModel[models.ResourceItem](c, h.DB) }
func (h *Handler) GetResource(c *gin.Context)    { getByID[models.ResourceItem](c, h.DB) }
func (h *Handler) CreateResource(c *gin.Context) { createByModel[models.ResourceItem](c, h.DB) }
func (h *Handler) UpdateResource(c *gin.Context) { updateByModel[models.ResourceItem](c, h.DB) }
func (h *Handler) DeleteResource(c *gin.Context) { deleteByModel[models.ResourceItem](c, h.DB) }
func (h *Handler) ListTags(c *gin.Context)       { listByModel[models.Tag](c, h.DB) }
func (h *Handler) GetTag(c *gin.Context)         { getByID[models.Tag](c, h.DB) }
func (h *Handler) CreateTag(c *gin.Context)      { createByModel[models.Tag](c, h.DB) }
func (h *Handler) UpdateTag(c *gin.Context)      { updateByModel[models.Tag](c, h.DB) }
func (h *Handler) DeleteTag(c *gin.Context)      { deleteByModel[models.Tag](c, h.DB) }
func (h *Handler) ListTasks(c *gin.Context)      { listByModel[models.Task](c, h.DB) }
func (h *Handler) GetTask(c *gin.Context)        { getByID[models.Task](c, h.DB) }
func (h *Handler) CreateTask(c *gin.Context)     { createByModel[models.Task](c, h.DB) }
func (h *Handler) UpdateTask(c *gin.Context)     { updateByModel[models.Task](c, h.DB) }
func (h *Handler) DeleteTask(c *gin.Context)     { deleteByModel[models.Task](c, h.DB) }
func (h *Handler) ListPlaybooks(c *gin.Context)  { listByModel[models.Playbook](c, h.DB) }
func (h *Handler) GetPlaybook(c *gin.Context)    { getByID[models.Playbook](c, h.DB) }
func (h *Handler) CreatePlaybook(c *gin.Context) { createByModel[models.Playbook](c, h.DB) }
func (h *Handler) UpdatePlaybook(c *gin.Context) { updateByModel[models.Playbook](c, h.DB) }
func (h *Handler) DeletePlaybook(c *gin.Context) { deleteByModel[models.Playbook](c, h.DB) }
func (h *Handler) ListTaskLogs(c *gin.Context)   { listByModel[models.TaskExecutionLog](c, h.DB) }
func (h *Handler) GetTaskLog(c *gin.Context)     { getByID[models.TaskExecutionLog](c, h.DB) }

func (h *Handler) BindResourceTags(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
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
	response.Success(c, gin.H{"id": id, "tagIds": req.TagIDs})
}

func (h *Handler) ExecuteTask(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var task models.Task
	if err := h.DB.First(&task, id).Error; err != nil {
		response.Internal(c, err)
		return
	}

	var playbook models.Playbook
	if err := h.DB.First(&playbook, task.PlaybookID).Error; err != nil {
		response.Internal(c, err)
		return
	}

	var req struct {
		InventoryContent string `json:"inventoryContent"`
		Confirm          bool   `json:"confirm"`
		RetryType        string `json:"retryType"`
	}
	if c.Request.ContentLength > 0 && !bindJSON(c, &req) {
		return
	}

	if task.IsHighRisk && !req.Confirm {
		response.Error(c, http.StatusBadRequest, appErr.New(4002, "high risk task requires confirm=true"))
		return
	}

	result := h.Executor.Run(executor.Request{
		TaskName:         task.Name,
		InventoryContent: req.InventoryContent,
		PlaybookContent:  playbook.Content,
		CheckOnly:        task.IsHighRisk,
		TimeoutSeconds:   300,
	})

	retryCount := 0
	if result.ExitCode != 0 && strings.EqualFold(req.RetryType, "host_unreachable") {
		retryCount = 1
	}

	logEntity := models.TaskExecutionLog{
		JobID:      result.JobID,
		TaskID:     task.ID,
		Command:    result.Command,
		ExitCode:   result.ExitCode,
		Summary:    result.Summary,
		Stdout:     result.Stdout,
		Stderr:     result.Stderr,
		Status:     result.Status,
		RetryCount: retryCount,
	}
	if err := h.DB.Create(&logEntity).Error; err != nil {
		response.Internal(c, err)
		return
	}

	h.Hub.Publish(ws.Message{
		Channel: "broadcast",
		Title:   "Task Execution",
		Content: "task execution finished",
		Data: gin.H{
			"taskId": task.ID,
			"jobId":  result.JobID,
			"status": result.Status,
		},
	})

	response.Success(c, logEntity)
}
