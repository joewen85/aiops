package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	appErr "devops-system/backend/internal/errors"
	"devops-system/backend/internal/executor"
	"devops-system/backend/internal/models"
	"devops-system/backend/internal/response"
)

func (h *Handler) ListTasks(c *gin.Context) {
	listByModel[models.Task](c, h.DB)
}
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

	severity := "success"
	if result.ExitCode != 0 {
		severity = "error"
	}
	_, _ = h.PublishNotification(NotificationOptions{
		TraceID:      result.JobID,
		Module:       "tasks",
		Source:       "task-executor",
		Event:        "task.execution.finished",
		Severity:     severity,
		ResourceType: "task",
		ResourceID:   strconv.FormatUint(uint64(task.ID), 10),
		Title:        "任务执行完成",
		Content:      "任务 " + task.Name + " 执行状态：" + result.Status,
		Data: gin.H{
			"taskId":     task.ID,
			"jobId":      result.JobID,
			"status":     result.Status,
			"exitCode":   result.ExitCode,
			"retryCount": retryCount,
		},
	})

	response.Success(c, logEntity)
}
