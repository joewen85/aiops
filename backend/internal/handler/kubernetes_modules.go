package handler

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	appErr "devops-system/backend/internal/errors"
	"devops-system/backend/internal/executor"
	kubeclient "devops-system/backend/internal/kube"
	"devops-system/backend/internal/models"
	"devops-system/backend/internal/pagination"
	"devops-system/backend/internal/response"
)

const kubernetesProtocolVersion = "aiops.kubernetes.v1alpha1"
const kubernetesDeleteConfirmText = "确认删除资源"
const kubernetesSubmitConfirmText = "确认提交资源"
const kubernetesMaxReplicas = int64(1<<31 - 1)
const kubernetesMaxManifestBytes = 256 * 1024

var nodeRegisterHostPattern = regexp.MustCompile(`^[a-zA-Z0-9._:\-]+$`)
var nodeRegisterUserPattern = regexp.MustCompile(`^[a-zA-Z0-9._\-]+$`)
var kubeadmTokenFlagPattern = regexp.MustCompile(`(?i)(--token\s+)([^\s]+)`)

type kubernetesClusterRequest struct {
	Name           string                 `json:"name"`
	APIServer      string                 `json:"apiServer"`
	CredentialType string                 `json:"credentialType"`
	KubeConfig     string                 `json:"kubeConfig"`
	Token          string                 `json:"token"`
	Env            string                 `json:"env"`
	Region         string                 `json:"region"`
	Owner          string                 `json:"owner"`
	Labels         map[string]interface{} `json:"labels"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type kubernetesActionRequest struct {
	ClusterID        uint                   `json:"clusterId"`
	Namespace        string                 `json:"namespace"`
	Kind             string                 `json:"kind"`
	Name             string                 `json:"name"`
	Action           string                 `json:"action"`
	DryRun           *bool                  `json:"dryRun"`
	ConfirmationText string                 `json:"confirmationText"`
	Params           map[string]interface{} `json:"params"`
}

type kubernetesManifestRequest struct {
	ClusterID        uint                   `json:"clusterId"`
	Namespace        string                 `json:"namespace"`
	Manifest         map[string]interface{} `json:"manifest"`
	DryRun           *bool                  `json:"dryRun"`
	ConfirmationText string                 `json:"confirmationText"`
}

type kubernetesNodeRegisterRequest struct {
	ClusterID      uint                   `json:"clusterId"`
	Hostname       string                 `json:"hostname"`
	InternalIP     string                 `json:"internalIp"`
	Roles          []string               `json:"roles"`
	CPU            string                 `json:"cpu"`
	Memory         string                 `json:"memory"`
	Pods           string                 `json:"pods"`
	KubeletVersion string                 `json:"kubeletVersion"`
	SSHUser        string                 `json:"sshUser"`
	SSHPassword    string                 `json:"sshPassword"`
	SSHPort        int                    `json:"sshPort"`
	Labels         map[string]interface{} `json:"labels"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type kubernetesNodeRegisterTaskRequest struct {
	ClusterID      uint                   `json:"clusterId"`
	CloudAssetID   uint                   `json:"cloudAssetId"`
	Hostname       string                 `json:"hostname"`
	InternalIP     string                 `json:"internalIp"`
	Roles          []string               `json:"roles"`
	CPU            string                 `json:"cpu"`
	Memory         string                 `json:"memory"`
	Pods           string                 `json:"pods"`
	KubeletVersion string                 `json:"kubeletVersion"`
	JoinCommand    string                 `json:"joinCommand"`
	SSHUser        string                 `json:"sshUser"`
	SSHPassword    string                 `json:"sshPassword"`
	SSHPort        int                    `json:"sshPort"`
	DryRun         *bool                  `json:"dryRun"`
	ExecuteNow     *bool                  `json:"executeNow"`
	Labels         map[string]interface{} `json:"labels"`
	Metadata       map[string]interface{} `json:"metadata"`
}

func (h *Handler) ListKubernetesClusters(c *gin.Context) {
	page := pagination.Parse(c)
	query := h.DB.Model(&models.KubernetesCluster{})
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where("name LIKE ? OR api_server LIKE ? OR owner LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if env := strings.TrimSpace(c.Query("env")); env != "" {
		query = query.Where("env = ?", env)
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	var items []models.KubernetesCluster
	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Internal(c, err)
		return
	}
	if err := query.Order("id desc").Limit(page.PageSize).Offset(pagination.Offset(page)).Find(&items).Error; err != nil {
		response.Internal(c, err)
		return
	}
	result := make([]gin.H, 0, len(items))
	for _, item := range items {
		result = append(result, kubernetesClusterResponse(item))
	}
	response.List(c, result, total, page.Page, page.PageSize)
}

func (h *Handler) GetKubernetesCluster(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	cluster, found := h.findKubernetesCluster(c, id)
	if !found {
		return
	}
	response.Success(c, kubernetesClusterResponse(cluster))
}

func (h *Handler) CreateKubernetesCluster(c *gin.Context) {
	var req kubernetesClusterRequest
	if !bindJSON(c, &req) {
		return
	}
	cluster, credential, err := h.buildKubernetesCluster(req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, err.Error()))
		return
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&cluster).Error; err != nil {
			return err
		}
		credential.ClusterID = cluster.ID
		return tx.Create(&credential).Error
	}); err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, kubernetesClusterResponse(cluster))
}

func (h *Handler) UpdateKubernetesCluster(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var cluster models.KubernetesCluster
	if err := h.DB.First(&cluster, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	var req kubernetesClusterRequest
	if !bindJSON(c, &req) {
		return
	}
	updates := map[string]interface{}{}
	next := cluster
	if strings.TrimSpace(req.Name) != "" {
		next.Name = strings.TrimSpace(req.Name)
		updates["name"] = next.Name
	}
	if strings.TrimSpace(req.APIServer) != "" {
		next.APIServer = strings.TrimSpace(req.APIServer)
		updates["api_server"] = next.APIServer
		updates["status"] = "unknown"
	}
	if strings.TrimSpace(req.CredentialType) != "" {
		next.CredentialType = normalizeKubernetesCredentialType(req.CredentialType)
		updates["credential_type"] = next.CredentialType
	}
	if strings.TrimSpace(req.Env) != "" {
		next.Env = normalizeKubernetesEnv(req.Env)
		updates["env"] = next.Env
	}
	if req.Region != "" {
		updates["region"] = strings.TrimSpace(req.Region)
	}
	if req.Owner != "" {
		updates["owner"] = strings.TrimSpace(req.Owner)
	}
	if req.Labels != nil {
		updates["labels"] = datatypes.JSONMap(req.Labels)
	}
	if req.Metadata != nil {
		updates["metadata"] = datatypes.JSONMap(req.Metadata)
	}
	if err := validateKubernetesCluster(next); err != nil {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, err.Error()))
		return
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&models.KubernetesCluster{}).Where("id = ?", id).Updates(updates).Error; err != nil {
				return err
			}
		}
		if strings.TrimSpace(req.KubeConfig) != "" || strings.TrimSpace(req.Token) != "" {
			return h.upsertKubernetesCredential(tx, id, next.CredentialType, req.KubeConfig, req.Token)
		}
		return nil
	}); err != nil {
		response.Internal(c, err)
		return
	}
	h.GetKubernetesCluster(c)
}

func (h *Handler) DeleteKubernetesCluster(c *gin.Context) {
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
	if strings.TrimSpace(req.ConfirmationText) != kubernetesDeleteConfirmText {
		response.Error(c, http.StatusBadRequest, appErr.New(3020, "confirmation text is required"))
		return
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("cluster_id = ?", id).Delete(&models.KubernetesCredential{}).Error; err != nil {
			return err
		}
		if err := tx.Where("cluster_id = ?", id).Delete(&models.KubernetesResourceSnapshot{}).Error; err != nil {
			return err
		}
		if err := tx.Where("cluster_id = ?", id).Delete(&models.KubernetesEvent{}).Error; err != nil {
			return err
		}
		if err := tx.Where("cluster_id = ?", id).Delete(&models.KubernetesOperation{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.KubernetesCluster{}, id).Error
	}); err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *Handler) CheckKubernetesCluster(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	cluster, found := h.findKubernetesCluster(c, id)
	if !found {
		return
	}
	traceID := uuid.NewString()
	startedAt := time.Now()
	now := time.Now()
	version := kubernetesVersionFromMetadata(cluster.Metadata)
	status := "connected"
	permissionSummary := gin.H{"read": true, "write": true, "mode": "mock-safe"}
	if kubeclient.IsMockAPIServer(cluster.APIServer) {
		if version == "" {
			version = "v1.29.0"
		}
	} else {
		credential, err := h.kubernetesCredential(cluster)
		if err != nil {
			h.handleKubernetesProviderError(c, cluster, traceID, "kubernetes.cluster.check.failed", err)
			return
		}
		result, err := (kubeclient.Client{}).Check(c.Request.Context(), credential)
		if err != nil {
			h.handleKubernetesProviderError(c, cluster, traceID, "kubernetes.cluster.check.failed", err)
			return
		}
		status = result.Status
		version = result.Version
		permissionSummary = gin.H(result.PermissionSummary)
		cluster.CertificateExpiresAt = result.CertificateExpiresAt
	}
	updates := map[string]interface{}{
		"status":          status,
		"version":         version,
		"last_checked_at": &now,
	}
	if cluster.CertificateExpiresAt != nil {
		updates["certificate_expires_at"] = cluster.CertificateExpiresAt
	}
	if err := h.DB.Model(&models.KubernetesCluster{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		response.Internal(c, err)
		return
	}
	cluster.Status = status
	cluster.Version = version
	cluster.LastCheckedAt = &now
	_, _ = h.PublishNotification(NotificationOptions{
		Channel:      "broadcast",
		Title:        "Kubernetes 集群校验通过",
		Content:      fmt.Sprintf("集群 %s 校验通过", cluster.Name),
		Module:       "kubernetes",
		Source:       "kubernetes-check",
		Event:        "kubernetes.cluster.check.success",
		Severity:     "success",
		ResourceType: "kubernetesCluster",
		ResourceID:   strconv.Itoa(int(cluster.ID)),
		TraceID:      traceID,
		Data:         map[string]interface{}{"clusterId": cluster.ID, "status": status},
	})
	response.Success(c, gin.H{
		"id":                   cluster.ID,
		"status":               status,
		"version":              version,
		"checkedAt":            now,
		"traceId":              traceID,
		"latencyMs":            time.Since(startedAt).Milliseconds(),
		"permissionSummary":    permissionSummary,
		"certificateExpiresAt": cluster.CertificateExpiresAt,
	})
}

func (h *Handler) SyncKubernetesCluster(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	cluster, found := h.findKubernetesCluster(c, id)
	if !found {
		return
	}
	now := time.Now()
	snapshots := []models.KubernetesResourceSnapshot{}
	warnings := []string{}
	if kubeclient.IsMockAPIServer(cluster.APIServer) {
		snapshots = defaultKubernetesSnapshots(cluster, now)
	} else {
		credential, err := h.kubernetesCredential(cluster)
		if err != nil {
			h.handleKubernetesProviderError(c, cluster, uuid.NewString(), "kubernetes.cluster.sync.failed", err)
			return
		}
		result, err := (kubeclient.Client{}).SyncSnapshots(c.Request.Context(), credential, cluster, now)
		if err != nil {
			h.handleKubernetesProviderError(c, cluster, uuid.NewString(), "kubernetes.cluster.sync.failed", err)
			return
		}
		snapshots = result.Snapshots
		warnings = result.Warnings
	}
	nextStatus := "connected"
	if len(warnings) > 0 {
		nextStatus = "partial"
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("cluster_id = ?", cluster.ID).Delete(&models.KubernetesResourceSnapshot{}).Error; err != nil {
			return err
		}
		if len(snapshots) > 0 {
			if err := tx.Create(&snapshots).Error; err != nil {
				return err
			}
		}
		return tx.Model(&models.KubernetesCluster{}).Where("id = ?", cluster.ID).Updates(map[string]interface{}{
			"status":         nextStatus,
			"last_synced_at": &now,
		}).Error
	}); err != nil {
		response.Internal(c, err)
		return
	}
	_, _ = h.PublishNotification(NotificationOptions{
		Channel:      "broadcast",
		Title:        "Kubernetes 集群同步完成",
		Content:      fmt.Sprintf("集群 %s 已同步 %d 个资源快照", cluster.Name, len(snapshots)),
		Module:       "kubernetes",
		Source:       "kubernetes-sync",
		Event:        "kubernetes.cluster.sync.success",
		Severity:     "success",
		ResourceType: "kubernetesCluster",
		ResourceID:   strconv.Itoa(int(cluster.ID)),
		Data:         map[string]interface{}{"clusterId": cluster.ID, "count": len(snapshots)},
	})
	response.Success(c, gin.H{"clusterId": cluster.ID, "status": "synced", "count": len(snapshots), "warnings": warnings, "syncedAt": now})
}

func (h *Handler) ListKubernetesResources(c *gin.Context) {
	page := pagination.Parse(c)
	query := h.DB.Model(&models.KubernetesResourceSnapshot{})
	if clusterID := parseOptionalUintQuery(c.Query("clusterId")); clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	}
	if namespace := strings.TrimSpace(c.Query("namespace")); namespace != "" {
		query = query.Where("namespace = ?", namespace)
	}
	if kind := normalizeKubernetesKind(c.Query("kind")); kind != "" {
		query = query.Where("kind = ?", kind)
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where("name LIKE ? OR uid LIKE ? OR spec_summary LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	var items []models.KubernetesResourceSnapshot
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

func (h *Handler) GetKubernetesResource(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var item models.KubernetesResourceSnapshot
	if err := h.DB.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) GetKubernetesResourceManifest(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var snapshot models.KubernetesResourceSnapshot
	if err := h.DB.First(&snapshot, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	cluster, found := h.findKubernetesCluster(c, snapshot.ClusterID)
	if !found {
		return
	}
	manifest := manifestFromSnapshot(snapshot)
	if kubeclient.IsMockAPIServer(cluster.APIServer) {
		response.Success(c, gin.H{"clusterId": cluster.ID, "manifest": sanitizeKubernetesManifest(manifest), "source": "snapshot"})
		return
	}
	credential, err := h.kubernetesCredential(cluster)
	if err != nil {
		h.handleKubernetesProviderError(c, cluster, uuid.NewString(), "kubernetes.resource.manifest.failed", err)
		return
	}
	result, err := (kubeclient.Client{}).GetManifest(c.Request.Context(), credential, kubeclient.ManifestRequest{
		Namespace: snapshot.Namespace,
		Manifest:  manifest,
	})
	if err != nil {
		h.handleKubernetesProviderError(c, cluster, uuid.NewString(), "kubernetes.resource.manifest.failed", err)
		return
	}
	response.Success(c, gin.H{"clusterId": cluster.ID, "manifest": result["object"], "source": "cluster", "resourceVersion": result["resourceVersion"]})
}

func (h *Handler) CreateKubernetesResource(c *gin.Context) {
	var req kubernetesManifestRequest
	if !bindJSON(c, &req) {
		return
	}
	h.runKubernetesManifest(c, "create", req, nil)
}

func (h *Handler) UpdateKubernetesResource(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var snapshot models.KubernetesResourceSnapshot
	if err := h.DB.First(&snapshot, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	var req kubernetesManifestRequest
	if !bindJSON(c, &req) {
		return
	}
	req.ClusterID = snapshot.ClusterID
	h.runKubernetesManifest(c, "update", req, &snapshot)
}

func (h *Handler) DeleteKubernetesResource(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var snapshot models.KubernetesResourceSnapshot
	if err := h.DB.First(&snapshot, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	var req kubernetesManifestRequest
	if !bindJSON(c, &req) {
		return
	}
	req.ClusterID = snapshot.ClusterID
	req.Namespace = snapshot.Namespace
	req.Manifest = manifestFromSnapshot(snapshot)
	h.runKubernetesManifest(c, "delete", req, &snapshot)
}

func (h *Handler) ListKubernetesNodes(c *gin.Context) {
	c.Request.URL.RawQuery = withKindQuery(c.Request.URL.RawQuery, "Node")
	h.ListKubernetesResources(c)
}

func (h *Handler) RegisterKubernetesNode(c *gin.Context) {
	var req kubernetesNodeRegisterRequest
	if !bindJSON(c, &req) {
		return
	}
	req.Hostname = strings.TrimSpace(req.Hostname)
	req.InternalIP = strings.TrimSpace(req.InternalIP)
	req.CPU = strings.TrimSpace(req.CPU)
	req.Memory = strings.TrimSpace(req.Memory)
	req.Pods = strings.TrimSpace(req.Pods)
	req.KubeletVersion = strings.TrimSpace(req.KubeletVersion)
	if req.ClusterID == 0 || req.Hostname == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "clusterId and hostname are required"))
		return
	}
	if req.InternalIP != "" && net.ParseIP(req.InternalIP) == nil {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "internalIp is invalid"))
		return
	}
	cluster, found := h.findKubernetesCluster(c, req.ClusterID)
	if !found {
		return
	}
	var exists int64
	if err := h.DB.Model(&models.KubernetesResourceSnapshot{}).
		Where("cluster_id = ? AND kind = ? AND name = ?", req.ClusterID, "Node", req.Hostname).
		Count(&exists).Error; err != nil {
		response.Internal(c, err)
		return
	}
	if exists > 0 {
		response.Error(c, http.StatusConflict, appErr.New(4025, "node already exists in cluster"))
		return
	}
	roles := normalizeNodeRoles(req.Roles)
	manifest := buildNodeRegisterManifest(req, roles)
	traceID := uuid.NewString()
	apiResult := map[string]interface{}{}
	if !kubeclient.IsMockAPIServer(cluster.APIServer) {
		credential, err := h.kubernetesCredential(cluster)
		if err != nil {
			h.handleKubernetesProviderError(c, cluster, traceID, "kubernetes.node.register.failed", err)
			return
		}
		result, err := (kubeclient.Client{}).ExecuteManifest(c.Request.Context(), credential, kubeclient.ManifestRequest{
			Action:   "create",
			DryRun:   false,
			Manifest: manifest,
		})
		if err != nil {
			h.handleKubernetesProviderError(c, cluster, traceID, "kubernetes.node.register.failed", err)
			return
		}
		apiResult = result
		if object, ok := result["object"].(map[string]interface{}); ok && len(object) > 0 {
			manifest = object
		}
	} else {
		apiResult = map[string]interface{}{"mock": true, "message": "node register simulated in mock cluster"}
	}
	now := time.Now()
	snapshot := snapshotFromManifest(cluster, manifest, now)
	snapshot.Namespace = ""
	snapshot.Kind = "Node"
	snapshot.Name = req.Hostname
	snapshot.Status = "Ready"
	snapshot.SpecSummary = nodeRegisterSpecSummary(req, roles)
	if snapshot.Metadata == nil {
		snapshot.Metadata = datatypes.JSONMap{}
	}
	snapshot.Metadata["registerSource"] = "manual-host"
	snapshot.Metadata["roles"] = roles
	if req.InternalIP != "" {
		snapshot.Metadata["internalIp"] = req.InternalIP
	}
	if req.KubeletVersion != "" {
		snapshot.Metadata["kubeletVersion"] = req.KubeletVersion
	}
	if req.Metadata != nil {
		snapshot.Metadata["registerMetadata"] = req.Metadata
	}

	startedAt := now
	finishedAt := time.Now()
	operation := models.KubernetesOperation{
		TraceID:   traceID,
		ClusterID: req.ClusterID,
		Namespace: "",
		Kind:      "Node",
		Name:      req.Hostname,
		Action:    "register",
		Status:    "success",
		DryRun:    false,
		RiskLevel: "P2",
		Request: datatypes.JSONMap{
			"clusterId":      req.ClusterID,
			"hostname":       req.Hostname,
			"internalIp":     req.InternalIP,
			"roles":          roles,
			"cpu":            req.CPU,
			"memory":         req.Memory,
			"pods":           req.Pods,
			"kubeletVersion": req.KubeletVersion,
			"labels":         req.Labels,
			"metadata":       req.Metadata,
		},
		Result: datatypes.JSONMap{
			"message": fmt.Sprintf("node %s registered", req.Hostname),
			"cluster": cluster.Name,
			"result":  apiResult,
		},
		StartedAt:  &startedAt,
		FinishedAt: &finishedAt,
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := upsertKubernetesSnapshot(tx, snapshot); err != nil {
			return err
		}
		return tx.Create(&operation).Error
	}); err != nil {
		response.Internal(c, err)
		return
	}
	_, _ = h.PublishNotification(NotificationOptions{
		TraceID:      traceID,
		Channel:      "broadcast",
		Title:        "Kubernetes 节点注册完成",
		Content:      fmt.Sprintf("主机 %s 已注册到集群 %s", req.Hostname, cluster.Name),
		Module:       "kubernetes",
		Source:       "kubernetes-node-register",
		Event:        "kubernetes.node.register.success",
		Severity:     "success",
		ResourceType: "Node",
		ResourceID:   req.Hostname,
		Data:         map[string]interface{}{"clusterId": req.ClusterID, "hostname": req.Hostname, "internalIp": req.InternalIP},
	})
	response.Success(c, gin.H{"traceId": traceID, "operation": operation, "node": snapshot})
}

func (h *Handler) RegisterKubernetesNodeTask(c *gin.Context) {
	var req kubernetesNodeRegisterTaskRequest
	if !bindJSON(c, &req) {
		return
	}
	if req.ClusterID == 0 {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "clusterId is required"))
		return
	}
	cluster, found := h.findKubernetesCluster(c, req.ClusterID)
	if !found {
		return
	}
	nodeReq, err := h.resolveNodeRegisterTaskRequest(req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, err.Error()))
		return
	}
	if nodeReq.InternalIP != "" && net.ParseIP(nodeReq.InternalIP) == nil {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "internalIp is invalid"))
		return
	}
	dryRun := req.DryRun != nil && *req.DryRun
	executeNow := req.ExecuteNow == nil || *req.ExecuteNow
	roles := normalizeNodeRoles(nodeReq.Roles)
	traceID := uuid.NewString()
	joinCommand := strings.TrimSpace(req.JoinCommand)
	if executeNow {
		if joinCommand == "" && !kubeclient.IsMockAPIServer(cluster.APIServer) {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "joinCommand is required for non-mock cluster"))
			return
		}
		if joinCommand == "" {
			joinCommand = "echo mock cluster node register"
		}
		if err := validateNodeRegisterTaskInput(nodeReq, joinCommand); err != nil {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, err.Error()))
			return
		}
	}

	joinCommandB64 := base64.StdEncoding.EncodeToString([]byte(joinCommand))
	inventory, err := buildNodeJoinInventory(nodeReq, joinCommandB64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, err.Error()))
		return
	}
	playbookContent := buildNodeJoinPlaybook()

	playbook := models.Playbook{
		Name:    fmt.Sprintf("k8s-node-register-%s", nodeReq.Hostname),
		Content: playbookContent,
	}
	task := models.Task{
		Name:          fmt.Sprintf("K8s节点注册-%s", nodeReq.Hostname),
		Description:   fmt.Sprintf("将主机 %s 注册到集群 %s", nodeReq.Hostname, cluster.Name),
		InventoryFrom: "manual",
		IsHighRisk:    true,
	}
	result := executor.Result{
		JobID:    traceID,
		Command:  "ansible-playbook -i <inventory> <playbook>",
		ExitCode: 0,
		Summary:  "node register task created",
		Status:   "pending",
	}
	if executeNow && dryRun {
		result.Summary = "node register dry-run"
		result.Status = "dry_run"
	}
	if executeNow && !dryRun {
		result = h.Executor.Run(executor.Request{
			TaskName:         task.Name,
			InventoryContent: inventory,
			PlaybookContent:  playbookContent,
			CheckOnly:        false,
			TimeoutSeconds:   900,
		})
		if result.JobID == "" {
			result.JobID = traceID
		}
	}

	now := time.Now()
	operationStatus := "pending"
	if executeNow && dryRun {
		operationStatus = "dry_run"
	} else if executeNow && result.ExitCode != 0 {
		operationStatus = "failed"
	} else if executeNow {
		operationStatus = "success"
	}
	operation := models.KubernetesOperation{
		TraceID:   traceID,
		ClusterID: req.ClusterID,
		Namespace: "",
		Kind:      "Node",
		Name:      nodeReq.Hostname,
		Action:    "register_task",
		Status:    operationStatus,
		DryRun:    dryRun,
		RiskLevel: "P2",
		Request: datatypes.JSONMap{
			"clusterId":      req.ClusterID,
			"cloudAssetId":   req.CloudAssetID,
			"hostname":       nodeReq.Hostname,
			"internalIp":     nodeReq.InternalIP,
			"roles":          roles,
			"cpu":            nodeReq.CPU,
			"memory":         nodeReq.Memory,
			"pods":           nodeReq.Pods,
			"kubeletVersion": nodeReq.KubeletVersion,
			"executeNow":     executeNow,
			"dryRun":         dryRun,
			"hasJoinCommand": joinCommand != "",
			"sshUser":        nodeReq.SSHUser,
			"sshPort":        nodeReq.SSHPort,
			"hasSSHPassword": nodeReq.SSHPassword != "",
		},
		Result: datatypes.JSONMap{
			"taskStatus": result.Status,
			"summary":    result.Summary,
			"exitCode":   result.ExitCode,
			"jobId":      result.JobID,
			"cloudAsset": req.CloudAssetID,
		},
		StartedAt:  &now,
		FinishedAt: &now,
	}
	if executeNow && result.ExitCode != 0 {
		operation.ErrorMessage = "node register task failed"
	}

	nodeManifest := buildNodeRegisterManifest(nodeReq, roles)
	snapshot := snapshotFromManifest(cluster, nodeManifest, now)
	snapshot.Namespace = ""
	snapshot.Kind = "Node"
	snapshot.Name = nodeReq.Hostname
	snapshot.Status = "Pending"
	if executeNow && dryRun {
		snapshot.Status = "Registering"
	}
	if executeNow && !dryRun && result.ExitCode == 0 {
		snapshot.Status = "Ready"
	}
	snapshot.SpecSummary = nodeRegisterSpecSummary(nodeReq, roles)
	if snapshot.Metadata == nil {
		snapshot.Metadata = datatypes.JSONMap{}
	}
	snapshot.Metadata["registerSource"] = "task-center"
	snapshot.Metadata["roles"] = roles
	snapshot.Metadata["taskJobId"] = result.JobID
	snapshot.Metadata["executeNow"] = executeNow
	if req.CloudAssetID > 0 {
		snapshot.Metadata["cloudAssetId"] = req.CloudAssetID
	}

	logEntity := models.TaskExecutionLog{
		JobID:      result.JobID,
		Command:    result.Command,
		ExitCode:   result.ExitCode,
		Summary:    result.Summary,
		Stdout:     redactNodeRegisterTaskOutput(result.Stdout),
		Stderr:     redactNodeRegisterTaskOutput(result.Stderr),
		Status:     result.Status,
		RetryCount: 0,
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&playbook).Error; err != nil {
			return err
		}
		task.PlaybookID = playbook.ID
		if err := tx.Create(&task).Error; err != nil {
			return err
		}
		logEntity.TaskID = task.ID
		if err := tx.Create(&logEntity).Error; err != nil {
			return err
		}
		if executeNow && !dryRun {
			if err := upsertKubernetesSnapshot(tx, snapshot); err != nil {
				return err
			}
		}
		return tx.Create(&operation).Error
	}); err != nil {
		response.Internal(c, err)
		return
	}

	severity := "info"
	if executeNow && result.ExitCode == 0 {
		severity = "success"
	}
	if executeNow && result.ExitCode != 0 {
		severity = "error"
	}
	_, _ = h.PublishNotification(NotificationOptions{
		TraceID:      traceID,
		Channel:      "broadcast",
		Title:        "Kubernetes 节点注册任务完成",
		Content:      fmt.Sprintf("主机 %s 注册任务状态：%s", nodeReq.Hostname, result.Status),
		Module:       "kubernetes",
		Source:       "kubernetes-node-register-task",
		Event:        "kubernetes.node.register.task.finished",
		Severity:     severity,
		ResourceType: "Node",
		ResourceID:   nodeReq.Hostname,
		Data: map[string]interface{}{
			"clusterId":    req.ClusterID,
			"taskId":       task.ID,
			"jobId":        result.JobID,
			"cloudAssetId": req.CloudAssetID,
		},
	})

	response.Success(c, gin.H{
		"traceId":      traceID,
		"taskId":       task.ID,
		"playbookId":   playbook.ID,
		"taskLog":      logEntity,
		"operation":    operation,
		"node":         snapshot,
		"dryRun":       dryRun,
		"executeNow":   executeNow,
		"cloudAssetId": req.CloudAssetID,
	})
}

func (h *Handler) ListKubernetesOperations(c *gin.Context) {
	page := pagination.Parse(c)
	query := h.DB.Model(&models.KubernetesOperation{})
	if clusterID := parseOptionalUintQuery(c.Query("clusterId")); clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if action := strings.TrimSpace(c.Query("action")); action != "" {
		query = query.Where("action = ?", action)
	}
	var items []models.KubernetesOperation
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

func (h *Handler) GetKubernetesOperation(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var item models.KubernetesOperation
	if err := h.DB.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) KubernetesAIOpsProtocol(c *gin.Context) {
	response.Success(c, gin.H{
		"protocolVersion":          kubernetesProtocolVersion,
		"actionEndpoint":           "/api/v1/kubernetes/actions",
		"dryRunEndpoint":           "/api/v1/kubernetes/actions/dry-run",
		"nodeRegisterEndpoint":     "/api/v1/kubernetes/nodes/register",
		"nodeRegisterTaskEndpoint": "/api/v1/kubernetes/nodes/register/task",
		"crudEndpoints": gin.H{
			"create": "POST /api/v1/kubernetes/resources",
			"get":    "GET /api/v1/kubernetes/resources/:id/manifest",
			"update": "PUT /api/v1/kubernetes/resources/:id",
			"delete": "DELETE /api/v1/kubernetes/resources/:id",
		},
		"resources": []gin.H{
			{"kind": "Deployment", "actions": []string{"restart", "scale", "pause", "resume", "delete"}, "namespaceScoped": true},
			{"kind": "StatefulSet", "actions": []string{"restart", "scale", "delete"}, "namespaceScoped": true},
			{"kind": "DaemonSet", "actions": []string{"restart"}, "namespaceScoped": true},
			{"kind": "Pod", "actions": []string{"delete", "evict"}, "namespaceScoped": true},
			{"kind": "Node", "actions": []string{"cordon", "uncordon"}, "namespaceScoped": false},
			{"kind": "Namespace", "actions": []string{"delete"}, "namespaceScoped": false},
			{"kind": "ConfigMap", "actions": []string{"delete"}, "namespaceScoped": true},
			{"kind": "Secret", "actions": []string{"delete"}, "namespaceScoped": true},
			{"kind": "PVC", "actions": []string{"delete"}, "namespaceScoped": true},
		},
		"requestSchema": gin.H{
			"clusterId":        "number|required",
			"namespace":        "string|required for namespace scoped resources",
			"kind":             "string|required",
			"name":             "string|required",
			"action":           "string|required",
			"dryRun":           "boolean|default true",
			"confirmationText": "string|required for high risk actions",
			"params":           "object|optional",
		},
		"manifestSchema": gin.H{
			"clusterId":        "number|required for create",
			"namespace":        "string|required for namespace scoped resources",
			"manifest":         "object|required Kubernetes manifest, supports custom resources",
			"dryRun":           "boolean|default true",
			"confirmationText": "string|required when dryRun=false",
		},
		"nodeRegisterTaskSchema": gin.H{
			"clusterId":    "number|required",
			"cloudAssetId": "number|optional",
			"hostname":     "string|required when internalIp empty",
			"internalIp":   "string|optional",
			"joinCommand":  "string|required when executeNow=true and cluster is non-mock",
			"sshUser":      "string|optional",
			"sshPort":      "number|default 22",
			"dryRun":       "boolean|default false",
			"executeNow":   "boolean|default true",
		},
		"safety": gin.H{"defaultDryRun": true, "deleteConfirmationText": kubernetesDeleteConfirmText, "manifestConfirmationText": kubernetesSubmitConfirmText, "traceField": "traceId", "deniedGroups": []string{"rbac.authorization.k8s.io", "admissionregistration.k8s.io", "apiextensions.k8s.io"}},
	})
}

func (h *Handler) KubernetesResourceAction(c *gin.Context) {
	h.KubernetesAction(c)
}

func (h *Handler) KubernetesActionDryRun(c *gin.Context) {
	var req kubernetesActionRequest
	if !bindJSON(c, &req) {
		return
	}
	dryRun := true
	req.DryRun = &dryRun
	h.runKubernetesAction(c, req)
}

func (h *Handler) KubernetesAction(c *gin.Context) {
	var req kubernetesActionRequest
	if !bindJSON(c, &req) {
		return
	}
	h.runKubernetesAction(c, req)
}

func (h *Handler) runKubernetesAction(c *gin.Context, req kubernetesActionRequest) {
	req.Kind = normalizeKubernetesKind(req.Kind)
	req.Action = normalizeKubernetesAction(req.Action)
	req.Namespace = strings.TrimSpace(req.Namespace)
	req.Name = strings.TrimSpace(req.Name)
	if req.ClusterID == 0 || req.Kind == "" || req.Action == "" || req.Name == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "clusterId, kind, name and action are required"))
		return
	}
	cluster, found := h.findKubernetesCluster(c, req.ClusterID)
	if !found {
		return
	}
	if kubernetesNamespaceRequired(req.Kind) && req.Namespace == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "namespace is required"))
		return
	}
	if !kubernetesActionSupported(req.Kind, req.Action) {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "unsupported kubernetes action"))
		return
	}
	if err := validateKubernetesActionParams(req); err != nil {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, err.Error()))
		return
	}
	dryRun := kubernetesActionDryRun(req)
	risk := kubernetesActionRisk(req.Kind, req.Action)
	traceID := uuid.NewString()
	now := time.Now()
	operation := models.KubernetesOperation{
		TraceID:   traceID,
		ClusterID: req.ClusterID,
		Namespace: req.Namespace,
		Kind:      req.Kind,
		Name:      req.Name,
		Action:    req.Action,
		Status:    "dry_run",
		DryRun:    dryRun,
		RiskLevel: risk,
		Request:   kubernetesActionRequestJSON(req),
		Result:    kubernetesDryRunPlan(cluster, req, risk),
		StartedAt: &now,
	}
	if !kubeclient.IsMockAPIServer(cluster.APIServer) {
		credential, err := h.kubernetesCredential(cluster)
		if err != nil {
			h.handleKubernetesActionError(c, &operation, err)
			return
		}
		if dryRun {
			apiResult, err := (kubeclient.Client{}).ExecuteAction(c.Request.Context(), credential, kubernetesClientActionRequest(req, true))
			if err != nil {
				h.handleKubernetesActionError(c, &operation, err)
				return
			}
			operation.FinishedAt = &now
			operation.Result = mergeKubernetesActionResult(operation.Result, apiResult)
			if err := h.DB.Create(&operation).Error; err != nil {
				response.Internal(c, err)
				return
			}
			response.Success(c, gin.H{"protocolVersion": kubernetesProtocolVersion, "traceId": traceID, "operation": operation, "dryRun": operation.Result})
			return
		}
		if kubernetesActionNeedsConfirm(req.Kind, req.Action) && strings.TrimSpace(req.ConfirmationText) != kubernetesDeleteConfirmText {
			response.Error(c, http.StatusBadRequest, appErr.New(3020, "confirmation text is required"))
			return
		}
		if kubernetesOperationRunning(h.DB, req) {
			response.Error(c, http.StatusConflict, appErr.New(4025, "kubernetes action is already running"))
			return
		}
		apiResult, err := (kubeclient.Client{}).ExecuteAction(c.Request.Context(), credential, kubernetesClientActionRequest(req, false))
		if err != nil {
			h.handleKubernetesActionError(c, &operation, err)
			return
		}
		finishedAt := time.Now()
		operation.Status = "success"
		operation.FinishedAt = &finishedAt
		operation.Result = mergeKubernetesActionResult(datatypes.JSONMap{
			"message": fmt.Sprintf("%s %s/%s accepted", req.Action, req.Kind, req.Name),
			"cluster": cluster.Name,
			"dryRun":  false,
		}, apiResult)
		if err := h.DB.Create(&operation).Error; err != nil {
			response.Internal(c, err)
			return
		}
		_, _ = h.PublishNotification(NotificationOptions{
			TraceID:      traceID,
			Channel:      "broadcast",
			Title:        "Kubernetes 操作完成",
			Content:      fmt.Sprintf("%s %s/%s 已执行", req.Action, req.Kind, req.Name),
			Module:       "kubernetes",
			Source:       "kubernetes-action",
			Event:        "kubernetes.action.success",
			Severity:     "success",
			ResourceType: req.Kind,
			ResourceID:   req.Name,
			Data:         map[string]interface{}{"clusterId": req.ClusterID, "namespace": req.Namespace, "action": req.Action, "riskLevel": risk},
		})
		response.Success(c, gin.H{"protocolVersion": kubernetesProtocolVersion, "traceId": traceID, "operation": operation})
		return
	}
	if dryRun {
		operation.FinishedAt = &now
		if err := h.DB.Create(&operation).Error; err != nil {
			response.Internal(c, err)
			return
		}
		response.Success(c, gin.H{"protocolVersion": kubernetesProtocolVersion, "traceId": traceID, "operation": operation, "dryRun": operation.Result})
		return
	}
	if kubernetesActionNeedsConfirm(req.Kind, req.Action) && strings.TrimSpace(req.ConfirmationText) != kubernetesDeleteConfirmText {
		response.Error(c, http.StatusBadRequest, appErr.New(3020, "confirmation text is required"))
		return
	}
	if kubernetesOperationRunning(h.DB, req) {
		response.Error(c, http.StatusConflict, appErr.New(4025, "kubernetes action is already running"))
		return
	}
	finishedAt := time.Now()
	operation.Status = "success"
	operation.FinishedAt = &finishedAt
	operation.Result = datatypes.JSONMap{
		"message": fmt.Sprintf("%s %s/%s accepted", req.Action, req.Kind, req.Name),
		"cluster": cluster.Name,
		"dryRun":  false,
	}
	if err := h.DB.Create(&operation).Error; err != nil {
		response.Internal(c, err)
		return
	}
	_, _ = h.PublishNotification(NotificationOptions{
		TraceID:      traceID,
		Channel:      "broadcast",
		Title:        "Kubernetes 操作完成",
		Content:      fmt.Sprintf("%s %s/%s 已执行", req.Action, req.Kind, req.Name),
		Module:       "kubernetes",
		Source:       "kubernetes-action",
		Event:        "kubernetes.action.success",
		Severity:     "success",
		ResourceType: req.Kind,
		ResourceID:   req.Name,
		Data:         map[string]interface{}{"clusterId": req.ClusterID, "namespace": req.Namespace, "action": req.Action, "riskLevel": risk},
	})
	response.Success(c, gin.H{"protocolVersion": kubernetesProtocolVersion, "traceId": traceID, "operation": operation})
}

func (h *Handler) runKubernetesManifest(c *gin.Context, action string, req kubernetesManifestRequest, existing *models.KubernetesResourceSnapshot) {
	action = normalizeKubernetesAction(action)
	req.Namespace = strings.TrimSpace(req.Namespace)
	if req.ClusterID == 0 || req.Manifest == nil {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "clusterId and manifest are required"))
		return
	}
	if err := validateKubernetesManifestRequest(action, req, existing); err != nil {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, err.Error()))
		return
	}
	cluster, found := h.findKubernetesCluster(c, req.ClusterID)
	if !found {
		return
	}
	dryRun := kubernetesManifestDryRun(req)
	if !dryRun && !kubernetesManifestConfirmationValid(action, req.ConfirmationText) {
		response.Error(c, http.StatusBadRequest, appErr.New(3020, "confirmation text is required"))
		return
	}

	execManifest := deepCopyJSONMap(req.Manifest)
	safeManifest := sanitizeKubernetesManifest(req.Manifest)
	apiVersion, kind, namespace, name := kubernetesManifestIdentity(execManifest, req.Namespace)
	traceID := uuid.NewString()
	now := time.Now()
	risk := kubernetesManifestRisk(kind, action)
	operation := models.KubernetesOperation{
		TraceID:   traceID,
		ClusterID: req.ClusterID,
		Namespace: namespace,
		Kind:      kind,
		Name:      name,
		Action:    action,
		Status:    "dry_run",
		DryRun:    dryRun,
		RiskLevel: risk,
		Request: datatypes.JSONMap{
			"clusterId": req.ClusterID,
			"namespace": namespace,
			"action":    action,
			"dryRun":    dryRun,
			"manifest":  safeManifest,
			"confirmed": kubernetesManifestConfirmationValid(action, req.ConfirmationText),
		},
		Result: datatypes.JSONMap{
			"steps": []string{
				"校验 manifest 基础字段、资源类型与安全黑名单",
				"执行 Kubernetes dynamic client server-side dry-run 或真实写入",
				"写入 kubernetes_operations 审计记录",
				"真实写入成功后更新本地资源快照",
			},
			"apiVersion":        apiVersion,
			"kind":              kind,
			"namespace":         namespace,
			"name":              name,
			"riskLevel":         risk,
			"approvalRequired":  !dryRun,
			"affectedResources": []string{fmt.Sprintf("%s/%s/%s", namespaceOrCluster(namespace), kind, name)},
			"safetyChecks":      []string{"manifest whitelist", "dry-run-first", "confirmation-required", "secret-redaction", "audit trace"},
			"cluster":           cluster.Name,
		},
		StartedAt: &now,
	}

	if !kubeclient.IsMockAPIServer(cluster.APIServer) {
		credential, err := h.kubernetesCredential(cluster)
		if err != nil {
			h.handleKubernetesActionError(c, &operation, err)
			return
		}
		apiResult, err := (kubeclient.Client{}).ExecuteManifest(c.Request.Context(), credential, kubeclient.ManifestRequest{
			Action:    action,
			Namespace: namespace,
			DryRun:    dryRun,
			Manifest:  execManifest,
		})
		if err != nil {
			h.handleKubernetesActionError(c, &operation, err)
			return
		}
		operation.Result = mergeKubernetesActionResult(operation.Result, apiResult)
		if object, ok := apiResult["object"].(map[string]interface{}); ok && len(object) > 0 {
			execManifest = object
			safeManifest = sanitizeKubernetesManifest(object)
		}
	} else {
		operation.Result = mergeKubernetesActionResult(operation.Result, map[string]interface{}{"mock": true, "dryRun": dryRun})
	}

	finishedAt := time.Now()
	operation.FinishedAt = &finishedAt
	if dryRun {
		if err := h.DB.Create(&operation).Error; err != nil {
			response.Internal(c, err)
			return
		}
		response.Success(c, gin.H{"protocolVersion": kubernetesProtocolVersion, "traceId": traceID, "operation": operation, "dryRun": operation.Result})
		return
	}

	operation.Status = "success"
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&operation).Error; err != nil {
			return err
		}
		if action == "delete" {
			if existing != nil {
				return tx.Delete(&models.KubernetesResourceSnapshot{}, existing.ID).Error
			}
			return tx.Where("cluster_id = ? AND namespace = ? AND kind = ? AND name = ?", req.ClusterID, namespace, kind, name).Delete(&models.KubernetesResourceSnapshot{}).Error
		}
		snapshot := snapshotFromManifest(cluster, safeManifest, now)
		return upsertKubernetesSnapshot(tx, snapshot)
	}); err != nil {
		response.Internal(c, err)
		return
	}
	_, _ = h.PublishNotification(NotificationOptions{
		TraceID:      traceID,
		Channel:      "broadcast",
		Title:        "Kubernetes 资源变更完成",
		Content:      fmt.Sprintf("%s %s/%s 已执行", action, kind, name),
		Module:       "kubernetes",
		Source:       "kubernetes-resource-crud",
		Event:        "kubernetes.resource.crud.success",
		Severity:     "success",
		ResourceType: kind,
		ResourceID:   name,
		Data:         map[string]interface{}{"clusterId": req.ClusterID, "namespace": namespace, "action": action, "riskLevel": risk},
	})
	response.Success(c, gin.H{"protocolVersion": kubernetesProtocolVersion, "traceId": traceID, "operation": operation})
}

func (h *Handler) findKubernetesCluster(c *gin.Context, id uint) (models.KubernetesCluster, bool) {
	var cluster models.KubernetesCluster
	if err := h.DB.First(&cluster, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return cluster, false
		}
		response.Internal(c, err)
		return cluster, false
	}
	return cluster, true
}

func (h *Handler) kubernetesCredential(cluster models.KubernetesCluster) (kubeclient.ClusterCredential, error) {
	var stored models.KubernetesCredential
	err := h.DB.Where("cluster_id = ?", cluster.ID).First(&stored).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		kubeConfig, decryptErr := h.decryptCloudCredential(cluster.KubeConfig)
		if decryptErr != nil {
			return kubeclient.ClusterCredential{}, decryptErr
		}
		return kubeclient.ClusterCredential{
			APIServer:      cluster.APIServer,
			CredentialType: cluster.CredentialType,
			KubeConfig:     kubeConfig,
		}, nil
	}
	if err != nil {
		return kubeclient.ClusterCredential{}, err
	}
	kubeConfig, err := h.decryptCloudCredential(stored.KubeConfig)
	if err != nil {
		return kubeclient.ClusterCredential{}, err
	}
	token, err := h.decryptCloudCredential(stored.Token)
	if err != nil {
		return kubeclient.ClusterCredential{}, err
	}
	return kubeclient.ClusterCredential{
		APIServer:      cluster.APIServer,
		CredentialType: stored.Type,
		KubeConfig:     kubeConfig,
		Token:          token,
	}, nil
}

func (h *Handler) handleKubernetesProviderError(c *gin.Context, cluster models.KubernetesCluster, traceID string, event string, err error) {
	_ = c.Error(err)
	now := time.Now()
	_ = h.DB.Model(&models.KubernetesCluster{}).Where("id = ?", cluster.ID).Updates(map[string]interface{}{
		"status":          "error",
		"last_checked_at": &now,
	}).Error
	_, _ = h.PublishNotification(NotificationOptions{
		TraceID:      traceID,
		Channel:      "broadcast",
		Title:        "Kubernetes 集群操作失败",
		Content:      fmt.Sprintf("集群 %s 操作失败，请查看服务端日志 traceId=%s", cluster.Name, traceID),
		Module:       "kubernetes",
		Source:       "kubernetes-client-go",
		Event:        event,
		Severity:     "error",
		ResourceType: "kubernetesCluster",
		ResourceID:   strconv.Itoa(int(cluster.ID)),
		Data:         map[string]interface{}{"clusterId": cluster.ID},
	})
	response.Error(c, http.StatusBadGateway, appErr.New(4004, "kubernetes api call failed"))
}

func (h *Handler) handleKubernetesActionError(c *gin.Context, operation *models.KubernetesOperation, err error) {
	_ = c.Error(err)
	finishedAt := time.Now()
	operation.Status = "failed"
	operation.ErrorMessage = "kubernetes api call failed"
	operation.FinishedAt = &finishedAt
	operation.Result = mergeKubernetesActionResult(operation.Result, map[string]interface{}{"error": "kubernetes api call failed"})
	_ = h.DB.Create(operation).Error
	response.Error(c, http.StatusBadGateway, appErr.New(4004, "kubernetes api call failed"))
}

func kubernetesClientActionRequest(req kubernetesActionRequest, dryRun bool) kubeclient.ActionRequest {
	return kubeclient.ActionRequest{
		Namespace: req.Namespace,
		Kind:      req.Kind,
		Name:      req.Name,
		Action:    req.Action,
		DryRun:    dryRun,
		Params:    req.Params,
	}
}

func mergeKubernetesActionResult(base datatypes.JSONMap, extra map[string]interface{}) datatypes.JSONMap {
	if base == nil {
		base = datatypes.JSONMap{}
	}
	for key, value := range extra {
		base[key] = value
	}
	return base
}

func (h *Handler) buildKubernetesCluster(req kubernetesClusterRequest) (models.KubernetesCluster, models.KubernetesCredential, error) {
	cluster := models.KubernetesCluster{
		Name:           strings.TrimSpace(req.Name),
		APIServer:      strings.TrimSpace(req.APIServer),
		CredentialType: normalizeKubernetesCredentialType(req.CredentialType),
		Env:            normalizeKubernetesEnv(req.Env),
		Region:         strings.TrimSpace(req.Region),
		Owner:          strings.TrimSpace(req.Owner),
		Status:         "unknown",
		Labels:         datatypes.JSONMap(req.Labels),
		Metadata:       datatypes.JSONMap(req.Metadata),
	}
	if err := validateKubernetesCluster(cluster); err != nil {
		return cluster, models.KubernetesCredential{}, err
	}
	credential, err := h.buildKubernetesCredential(0, cluster.CredentialType, req.KubeConfig, req.Token)
	if err != nil {
		return cluster, credential, err
	}
	cluster.KubeConfig = credential.KubeConfig
	return cluster, credential, nil
}

func (h *Handler) buildKubernetesCredential(clusterID uint, typ string, kubeConfig string, token string) (models.KubernetesCredential, error) {
	typ = normalizeKubernetesCredentialType(typ)
	kubeConfig = strings.TrimSpace(kubeConfig)
	token = strings.TrimSpace(token)
	if typ == "kubeconfig" && kubeConfig == "" {
		return models.KubernetesCredential{}, errors.New("kubeConfig is required")
	}
	if typ == "token" && token == "" {
		return models.KubernetesCredential{}, errors.New("token is required")
	}
	credential := models.KubernetesCredential{ClusterID: clusterID, Type: typ, KeyVersion: "v1"}
	if kubeConfig != "" {
		encrypted, err := h.encryptCloudCredential(kubeConfig)
		if err != nil {
			return credential, err
		}
		credential.KubeConfig = encrypted
	}
	if token != "" {
		encrypted, err := h.encryptCloudCredential(token)
		if err != nil {
			return credential, err
		}
		credential.Token = encrypted
	}
	return credential, nil
}

func (h *Handler) upsertKubernetesCredential(tx *gorm.DB, clusterID uint, typ string, kubeConfig string, token string) error {
	credential, err := h.buildKubernetesCredential(clusterID, typ, kubeConfig, token)
	if err != nil {
		return err
	}
	var existing models.KubernetesCredential
	err = tx.Where("cluster_id = ?", clusterID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return tx.Create(&credential).Error
	}
	if err != nil {
		return err
	}
	updates := map[string]interface{}{"type": credential.Type, "rotated_at": time.Now()}
	if credential.KubeConfig != "" {
		updates["kube_config"] = credential.KubeConfig
	}
	if credential.Token != "" {
		updates["token"] = credential.Token
	}
	return tx.Model(&models.KubernetesCredential{}).Where("id = ?", existing.ID).Updates(updates).Error
}

func kubernetesClusterResponse(cluster models.KubernetesCluster) gin.H {
	return gin.H{
		"id":                   cluster.ID,
		"name":                 cluster.Name,
		"apiServer":            cluster.APIServer,
		"credentialType":       cluster.CredentialType,
		"kubeConfig":           maskCloudCredential(cluster.KubeConfig),
		"env":                  cluster.Env,
		"region":               cluster.Region,
		"owner":                cluster.Owner,
		"status":               cluster.Status,
		"version":              cluster.Version,
		"labels":               cluster.Labels,
		"metadata":             cluster.Metadata,
		"lastCheckedAt":        cluster.LastCheckedAt,
		"lastSyncedAt":         cluster.LastSyncedAt,
		"certificateExpiresAt": cluster.CertificateExpiresAt,
		"createdAt":            cluster.CreatedAt,
		"updatedAt":            cluster.UpdatedAt,
	}
}

func validateKubernetesCluster(cluster models.KubernetesCluster) error {
	if strings.TrimSpace(cluster.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(cluster.APIServer) == "" {
		return errors.New("apiServer is required")
	}
	return validateKubernetesAPIServer(cluster.APIServer, cluster.Env)
}

func validateKubernetesAPIServer(raw string, env string) error {
	text := strings.TrimSpace(raw)
	if strings.HasPrefix(text, "mock://") {
		switch normalizeKubernetesEnv(env) {
		case "dev", "test", "local":
			return nil
		default:
			return errors.New("mock apiServer is only allowed in dev/test/local env")
		}
	}
	parsed, err := url.Parse(text)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("invalid apiServer")
	}
	if parsed.Scheme != "https" {
		return errors.New("apiServer must use https")
	}
	if normalizeKubernetesEnv(env) == "prod" && (strings.HasPrefix(parsed.Hostname(), "127.") || parsed.Hostname() == "localhost") {
		return errors.New("prod apiServer cannot use loopback address")
	}
	return nil
}

func normalizeKubernetesCredentialType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "token", "serviceaccount":
		return "token"
	default:
		return "kubeconfig"
	}
}

func normalizeKubernetesEnv(value string) string {
	text := strings.ToLower(strings.TrimSpace(value))
	if text == "" {
		return "prod"
	}
	return text
}

func normalizeKubernetesKind(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "namespace", "ns":
		return "Namespace"
	case "node":
		return "Node"
	case "pod":
		return "Pod"
	case "deployment", "deploy":
		return "Deployment"
	case "statefulset", "sts":
		return "StatefulSet"
	case "daemonset", "ds":
		return "DaemonSet"
	case "service", "svc":
		return "Service"
	case "ingress", "ing":
		return "Ingress"
	case "configmap", "cm":
		return "ConfigMap"
	case "secret":
		return "Secret"
	case "pvc", "persistentvolumeclaim":
		return "PVC"
	case "pv", "persistentvolume":
		return "PV"
	default:
		return strings.TrimSpace(value)
	}
}

func normalizeKubernetesAction(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func kubernetesActionSupported(kind string, action string) bool {
	allowed := map[string]map[string]struct{}{
		"Deployment":  {"restart": {}, "scale": {}, "pause": {}, "resume": {}, "delete": {}},
		"StatefulSet": {"restart": {}, "scale": {}, "delete": {}},
		"DaemonSet":   {"restart": {}},
		"Pod":         {"delete": {}, "evict": {}},
		"Node":        {"cordon": {}, "uncordon": {}},
		"Namespace":   {"delete": {}},
		"ConfigMap":   {"delete": {}},
		"Secret":      {"delete": {}},
		"PVC":         {"delete": {}},
		"PV":          {"delete": {}},
	}
	actions, ok := allowed[kind]
	if !ok {
		return false
	}
	_, ok = actions[action]
	return ok
}

func validateKubernetesManifestRequest(action string, req kubernetesManifestRequest, existing *models.KubernetesResourceSnapshot) error {
	if action != "create" && action != "update" && action != "delete" {
		return errors.New("unsupported kubernetes manifest action")
	}
	if err := validateKubernetesManifestSize(req.Manifest); err != nil {
		return err
	}
	apiVersion, kind, namespace, name := kubernetesManifestIdentity(req.Manifest, req.Namespace)
	if apiVersion == "" || kind == "" || name == "" {
		return errors.New("manifest apiVersion, kind and metadata.name are required")
	}
	if isKubernetesManifestDenied(apiVersion, kind, action) {
		return errors.New("manifest resource kind is not allowed")
	}
	if kubernetesManifestNamespaceRequired(apiVersion, kind) && namespace == "" {
		return errors.New("namespace is required")
	}
	if existing != nil {
		if existing.ClusterID != req.ClusterID || existing.Kind != kind || existing.Name != name {
			return errors.New("manifest does not match selected resource")
		}
		if strings.TrimSpace(existing.Namespace) != strings.TrimSpace(namespace) {
			return errors.New("manifest namespace does not match selected resource")
		}
	}
	return nil
}

func validateKubernetesManifestSize(manifest map[string]interface{}) error {
	raw, err := json.Marshal(manifest)
	if err != nil {
		return errors.New("manifest must be valid JSON object")
	}
	if len(raw) > kubernetesMaxManifestBytes {
		return fmt.Errorf("manifest exceeds %d bytes", kubernetesMaxManifestBytes)
	}
	return nil
}

func kubernetesManifestDryRun(req kubernetesManifestRequest) bool {
	if req.DryRun == nil {
		return true
	}
	return *req.DryRun
}

func kubernetesManifestIdentity(manifest map[string]interface{}, fallbackNamespace string) (string, string, string, string) {
	apiVersion, _ := manifest["apiVersion"].(string)
	kind, _ := manifest["kind"].(string)
	namespace := strings.TrimSpace(fallbackNamespace)
	name := ""
	if metadata, ok := manifest["metadata"].(map[string]interface{}); ok {
		if value, ok := metadata["namespace"].(string); ok && strings.TrimSpace(value) != "" {
			namespace = strings.TrimSpace(value)
		}
		if value, ok := metadata["name"].(string); ok {
			name = strings.TrimSpace(value)
		}
	}
	return strings.TrimSpace(apiVersion), normalizeKubernetesKind(kind), namespace, name
}

func kubernetesManifestNamespaceRequired(apiVersion string, kind string) bool {
	if isCustomKubernetesAPI(apiVersion) {
		return false
	}
	switch normalizeKubernetesKind(kind) {
	case "Namespace", "Node", "PV":
		return false
	}
	return true
}

func isCustomKubernetesAPI(apiVersion string) bool {
	parts := strings.Split(strings.TrimSpace(apiVersion), "/")
	if len(parts) != 2 {
		return false
	}
	switch parts[0] {
	case "apps", "batch", "networking.k8s.io", "autoscaling", "policy", "storage.k8s.io":
		return false
	default:
		return true
	}
}

func isKubernetesManifestDenied(apiVersion string, kind string, action string) bool {
	kind = normalizeKubernetesKind(kind)
	if action != "delete" {
		switch kind {
		case "Node", "PV":
			return true
		}
	}
	group := ""
	if parts := strings.Split(strings.TrimSpace(apiVersion), "/"); len(parts) == 2 {
		group = parts[0]
	}
	switch group {
	case "rbac.authorization.k8s.io", "admissionregistration.k8s.io", "apiextensions.k8s.io", "authentication.k8s.io", "authorization.k8s.io", "scheduling.k8s.io":
		return true
	}
	switch kind {
	case "Role", "ClusterRole", "RoleBinding", "ClusterRoleBinding", "ServiceAccount", "CustomResourceDefinition", "MutatingWebhookConfiguration", "ValidatingWebhookConfiguration":
		return true
	default:
		return false
	}
}

func kubernetesManifestRisk(kind string, action string) string {
	kind = normalizeKubernetesKind(kind)
	if kind == "Secret" || kind == "Namespace" {
		return "P1"
	}
	if action == "delete" || action == "update" {
		return "P2"
	}
	return "P3"
}

func validateKubernetesActionParams(req kubernetesActionRequest) error {
	if req.Action != "scale" {
		return nil
	}
	if req.Params == nil {
		return errors.New("params.replicas is required")
	}
	value, ok := req.Params["replicas"]
	if !ok {
		return errors.New("params.replicas is required")
	}
	switch typed := value.(type) {
	case int:
		if typed < 0 || int64(typed) > kubernetesMaxReplicas {
			return errors.New("params.replicas must be non-negative integer")
		}
	case int32:
		if typed < 0 {
			return errors.New("params.replicas must be non-negative integer")
		}
	case int64:
		if typed < 0 || typed > kubernetesMaxReplicas {
			return errors.New("params.replicas must be non-negative integer")
		}
	case float64:
		if typed < 0 || typed > float64(kubernetesMaxReplicas) || typed != float64(int32(typed)) {
			return errors.New("params.replicas must be non-negative integer")
		}
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 32)
		if err != nil || parsed < 0 {
			return errors.New("params.replicas must be non-negative integer")
		}
	default:
		return errors.New("params.replicas must be non-negative integer")
	}
	return nil
}

func kubernetesNamespaceRequired(kind string) bool {
	switch kind {
	case "Node", "Namespace", "PV":
		return false
	default:
		return true
	}
}

func kubernetesActionDryRun(req kubernetesActionRequest) bool {
	if req.DryRun == nil {
		return true
	}
	return *req.DryRun
}

func kubernetesActionNeedsConfirm(kind string, action string) bool {
	switch action {
	case "delete":
		return true
	}
	return kind == "Secret"
}

func kubernetesActionRisk(kind string, action string) string {
	if kind == "Namespace" || kind == "Secret" {
		return "P1"
	}
	if action == "delete" {
		return "P2"
	}
	return "P3"
}

func kubernetesManifestConfirmationValid(action string, confirmationText string) bool {
	text := strings.TrimSpace(confirmationText)
	if normalizeKubernetesAction(action) == "delete" {
		return text == kubernetesDeleteConfirmText
	}
	return text == kubernetesSubmitConfirmText || text == kubernetesDeleteConfirmText
}

func kubernetesActionRequestJSON(req kubernetesActionRequest) datatypes.JSONMap {
	return datatypes.JSONMap{
		"clusterId": req.ClusterID,
		"namespace": req.Namespace,
		"kind":      req.Kind,
		"name":      req.Name,
		"action":    req.Action,
		"dryRun":    kubernetesActionDryRun(req),
		"confirmed": strings.TrimSpace(req.ConfirmationText) == kubernetesDeleteConfirmText,
		"params":    req.Params,
	}
}

func kubernetesDryRunPlan(cluster models.KubernetesCluster, req kubernetesActionRequest, risk string) datatypes.JSONMap {
	return datatypes.JSONMap{
		"steps": []string{
			"校验集群、命名空间、资源类型与动作白名单",
			"执行 Kubernetes server-side dry-run 或等效校验",
			"写入 kubernetes_operations 审计记录",
			"按风险等级决定是否需要审批或二次确认",
		},
		"impact":            kubernetesActionImpact(req),
		"riskLevel":         risk,
		"approvalRequired":  risk == "P1",
		"affectedResources": []string{fmt.Sprintf("%s/%s/%s", req.Namespace, req.Kind, req.Name)},
		"rollback":          kubernetesActionRollback(req),
		"safetyChecks":      []string{"RBAC/ABAC", "action whitelist", "dry-run-first", "audit trace"},
		"cluster":           cluster.Name,
	}
}

func kubernetesActionImpact(req kubernetesActionRequest) string {
	switch req.Action {
	case "delete":
		return "删除资源会影响关联 Pod、Service 或持久化数据，请先确认影响面"
	case "restart":
		return "重启会触发滚动更新，短时间内可能影响实例可用性"
	case "scale":
		return "扩缩容会改变资源副本数和容量"
	default:
		return "该操作会修改 Kubernetes 资源状态"
	}
}

func kubernetesActionRollback(req kubernetesActionRequest) string {
	switch req.Action {
	case "delete":
		return "请使用备份 YAML 或 GitOps/IaC 版本重新创建资源"
	case "scale":
		return "恢复操作前副本数"
	case "restart":
		return "如滚动异常，回滚到上一 ReplicaSet 或镜像版本"
	default:
		return "按资源类型执行对应回滚或重新应用上一版本配置"
	}
}

func kubernetesOperationRunning(db *gorm.DB, req kubernetesActionRequest) bool {
	var count int64
	if err := db.Model(&models.KubernetesOperation{}).
		Where("cluster_id = ? AND namespace = ? AND kind = ? AND name = ? AND status = ?", req.ClusterID, req.Namespace, req.Kind, req.Name, "running").
		Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func defaultKubernetesSnapshots(cluster models.KubernetesCluster, now time.Time) []models.KubernetesResourceSnapshot {
	labels := datatypes.JSONMap{"env": cluster.Env, "managedBy": "aiops"}
	return []models.KubernetesResourceSnapshot{
		{ClusterID: cluster.ID, Namespace: "", Kind: "Node", Name: "node-1", UID: fmt.Sprintf("k8s:%d:node:node-1", cluster.ID), Status: "Ready", SpecSummary: "cpu=4 memory=16Gi pods=42", Labels: labels, Metadata: datatypes.JSONMap{"role": "worker"}, ResourceVersion: "1", LastSyncedAt: now},
		{ClusterID: cluster.ID, Namespace: "default", Kind: "Namespace", Name: "default", UID: fmt.Sprintf("k8s:%d:ns:default", cluster.ID), Status: "Active", SpecSummary: "default namespace", Labels: labels, Metadata: datatypes.JSONMap{}, ResourceVersion: "1", LastSyncedAt: now},
		{ClusterID: cluster.ID, Namespace: "default", Kind: "Deployment", Name: "web-api", UID: fmt.Sprintf("k8s:%d:deploy:web-api", cluster.ID), Status: "Available", SpecSummary: "replicas=3 image=nginx:latest", Labels: datatypes.JSONMap{"app": "web-api", "env": cluster.Env}, Metadata: datatypes.JSONMap{"replicas": 3, "image": "nginx:latest"}, ResourceVersion: "1", LastSyncedAt: now},
		{ClusterID: cluster.ID, Namespace: "default", Kind: "Pod", Name: "web-api-0", UID: fmt.Sprintf("k8s:%d:pod:web-api-0", cluster.ID), Status: "Running", SpecSummary: "node=node-1 restarts=0", Labels: datatypes.JSONMap{"app": "web-api"}, Metadata: datatypes.JSONMap{"node": "node-1", "restartCount": 0}, ResourceVersion: "1", LastSyncedAt: now},
		{ClusterID: cluster.ID, Namespace: "default", Kind: "Service", Name: "web-api", UID: fmt.Sprintf("k8s:%d:svc:web-api", cluster.ID), Status: "Active", SpecSummary: "type=ClusterIP port=80", Labels: datatypes.JSONMap{"app": "web-api"}, Metadata: datatypes.JSONMap{"type": "ClusterIP", "port": 80}, ResourceVersion: "1", LastSyncedAt: now},
		{ClusterID: cluster.ID, Namespace: "default", Kind: "Ingress", Name: "web-api", UID: fmt.Sprintf("k8s:%d:ing:web-api", cluster.ID), Status: "Ready", SpecSummary: "host=web.example.local tls=false", Labels: datatypes.JSONMap{"app": "web-api"}, Metadata: datatypes.JSONMap{"host": "web.example.local"}, ResourceVersion: "1", LastSyncedAt: now},
		{ClusterID: cluster.ID, Namespace: "default", Kind: "PVC", Name: "web-data", UID: fmt.Sprintf("k8s:%d:pvc:web-data", cluster.ID), Status: "Bound", SpecSummary: "storage=20Gi", Labels: datatypes.JSONMap{"app": "web-api"}, Metadata: datatypes.JSONMap{"storage": "20Gi"}, ResourceVersion: "1", LastSyncedAt: now},
	}
}

func sanitizeKubernetesManifest(manifest map[string]interface{}) map[string]interface{} {
	out := deepCopyJSONMap(manifest)
	if _, kind, _, _ := kubernetesManifestIdentity(out, ""); kind == "Secret" {
		delete(out, "data")
		delete(out, "stringData")
		out["dataRedacted"] = true
	}
	return out
}

func manifestFromSnapshot(snapshot models.KubernetesResourceSnapshot) map[string]interface{} {
	if manifest, ok := snapshot.Metadata["manifest"].(map[string]interface{}); ok && len(manifest) > 0 {
		return deepCopyJSONMap(manifest)
	}
	apiVersion := ""
	if value, ok := snapshot.Metadata["apiVersion"].(string); ok {
		apiVersion = strings.TrimSpace(value)
	}
	if apiVersion == "" {
		apiVersion = defaultKubernetesAPIVersion(snapshot.Kind)
	}
	metadata := map[string]interface{}{"name": snapshot.Name}
	if strings.TrimSpace(snapshot.Namespace) != "" {
		metadata["namespace"] = snapshot.Namespace
	}
	return map[string]interface{}{
		"apiVersion": apiVersion,
		"kind":       snapshot.Kind,
		"metadata":   metadata,
	}
}

func snapshotFromManifest(cluster models.KubernetesCluster, manifest map[string]interface{}, now time.Time) models.KubernetesResourceSnapshot {
	apiVersion, kind, namespace, name := kubernetesManifestIdentity(manifest, "")
	metadata := manifestMetadataMap(manifest)
	uid := stringValue(metadata["uid"])
	if uid == "" {
		uid = fmt.Sprintf("k8s:%d:%s:%s:%s", cluster.ID, kind, namespaceOrCluster(namespace), name)
	}
	resourceVersion := stringValue(metadata["resourceVersion"])
	labels := datatypes.JSONMap{}
	if rawLabels, ok := metadata["labels"].(map[string]interface{}); ok {
		for key, value := range rawLabels {
			labels[key] = value
		}
	}
	status := kubernetesManifestStatus(manifest)
	if status == "" {
		status = "Active"
	}
	safeManifest := sanitizeKubernetesManifest(manifest)
	return models.KubernetesResourceSnapshot{
		ClusterID:       cluster.ID,
		Namespace:       namespace,
		Kind:            kind,
		Name:            name,
		UID:             uid,
		Status:          status,
		SpecSummary:     fmt.Sprintf("apiVersion=%s kind=%s", apiVersion, kind),
		ResourceVersion: resourceVersion,
		Labels:          labels,
		Metadata:        datatypes.JSONMap{"apiVersion": apiVersion, "manifest": safeManifest},
		LastSyncedAt:    now,
	}
}

func upsertKubernetesSnapshot(tx *gorm.DB, snapshot models.KubernetesResourceSnapshot) error {
	var current models.KubernetesResourceSnapshot
	err := tx.Where("cluster_id = ? AND namespace = ? AND kind = ? AND name = ?", snapshot.ClusterID, snapshot.Namespace, snapshot.Kind, snapshot.Name).First(&current).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return tx.Create(&snapshot).Error
	}
	if err != nil {
		return err
	}
	return tx.Model(&models.KubernetesResourceSnapshot{}).Where("id = ?", current.ID).Updates(map[string]interface{}{
		"uid":              snapshot.UID,
		"status":           snapshot.Status,
		"spec_summary":     snapshot.SpecSummary,
		"resource_version": snapshot.ResourceVersion,
		"labels":           snapshot.Labels,
		"metadata":         snapshot.Metadata,
		"last_synced_at":   snapshot.LastSyncedAt,
	}).Error
}

func kubernetesManifestStatus(manifest map[string]interface{}) string {
	status, ok := manifest["status"].(map[string]interface{})
	if !ok {
		return ""
	}
	if phase := stringValue(status["phase"]); phase != "" {
		return phase
	}
	if conditions, ok := status["conditions"].([]interface{}); ok {
		for _, raw := range conditions {
			condition, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			if stringValue(condition["type"]) == "Ready" {
				return stringValue(condition["status"])
			}
		}
	}
	return ""
}

func manifestMetadataMap(manifest map[string]interface{}) map[string]interface{} {
	if metadata, ok := manifest["metadata"].(map[string]interface{}); ok {
		return metadata
	}
	return map[string]interface{}{}
}

func deepCopyJSONMap(input map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for key, value := range input {
		if nested, ok := value.(map[string]interface{}); ok {
			out[key] = deepCopyJSONMap(nested)
			continue
		}
		if list, ok := value.([]interface{}); ok {
			next := make([]interface{}, 0, len(list))
			for _, item := range list {
				if nested, ok := item.(map[string]interface{}); ok {
					next = append(next, deepCopyJSONMap(nested))
				} else {
					next = append(next, item)
				}
			}
			out[key] = next
			continue
		}
		out[key] = value
	}
	return out
}

func stringValue(value interface{}) string {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func namespaceOrCluster(namespace string) string {
	if strings.TrimSpace(namespace) == "" {
		return "cluster"
	}
	return strings.TrimSpace(namespace)
}

func defaultKubernetesAPIVersion(kind string) string {
	switch normalizeKubernetesKind(kind) {
	case "Deployment", "StatefulSet", "DaemonSet":
		return "apps/v1"
	case "Ingress":
		return "networking.k8s.io/v1"
	default:
		return "v1"
	}
}

func kubernetesVersionFromMetadata(metadata datatypes.JSONMap) string {
	if metadata == nil {
		return ""
	}
	if value, ok := metadata["version"].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func (h *Handler) resolveNodeRegisterTaskRequest(req kubernetesNodeRegisterTaskRequest) (kubernetesNodeRegisterRequest, error) {
	target := kubernetesNodeRegisterRequest{
		ClusterID:      req.ClusterID,
		Hostname:       strings.TrimSpace(req.Hostname),
		InternalIP:     strings.TrimSpace(req.InternalIP),
		Roles:          req.Roles,
		CPU:            strings.TrimSpace(req.CPU),
		Memory:         strings.TrimSpace(req.Memory),
		Pods:           strings.TrimSpace(req.Pods),
		KubeletVersion: strings.TrimSpace(req.KubeletVersion),
		SSHUser:        strings.TrimSpace(req.SSHUser),
		SSHPassword:    strings.TrimSpace(req.SSHPassword),
		SSHPort:        req.SSHPort,
		Labels:         req.Labels,
		Metadata:       req.Metadata,
	}
	if target.SSHPort <= 0 {
		target.SSHPort = 22
	}
	if target.SSHUser == "" {
		target.SSHUser = "root"
	}
	if req.CloudAssetID == 0 {
		if target.Hostname == "" && target.InternalIP == "" {
			return target, errors.New("hostname or internalIp is required")
		}
		return target, nil
	}
	var asset models.CloudAsset
	if err := h.DB.First(&asset, req.CloudAssetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return target, errors.New("cloud asset not found")
		}
		return target, err
	}
	if !strings.EqualFold(asset.Type, "CloudServer") {
		return target, errors.New("cloud asset type must be CloudServer")
	}
	if target.Hostname == "" {
		target.Hostname = defaultString(strings.TrimSpace(asset.Name), strings.TrimSpace(asset.ResourceID))
	}
	if target.InternalIP == "" {
		target.InternalIP = readCloudAssetString(asset.Metadata, "internalIp", "privateIp", "privateIP", "ip")
	}
	if target.SSHUser == "root" {
		if value := readCloudAssetString(asset.Metadata, "sshUser", "username", "user"); value != "" {
			target.SSHUser = value
		}
	}
	if target.SSHPassword == "" {
		target.SSHPassword = readCloudAssetString(asset.Metadata, "sshPassword", "password")
	}
	if target.SSHPort == 22 {
		target.SSHPort = readCloudAssetInt(asset.Metadata, 22, "sshPort", "port")
	}
	if target.CPU == "" {
		target.CPU = readCloudAssetString(asset.Metadata, "cpu", "cpuCore")
	}
	if target.Memory == "" {
		target.Memory = readCloudAssetString(asset.Metadata, "memory", "memoryGiB")
	}
	if target.Pods == "" {
		target.Pods = readCloudAssetString(asset.Metadata, "pods", "podLimit")
	}
	if target.Metadata == nil {
		target.Metadata = map[string]interface{}{}
	}
	target.Metadata["cloudAssetId"] = asset.ID
	target.Metadata["cloudProvider"] = asset.Provider
	target.Metadata["cloudRegion"] = asset.Region
	target.Metadata["cloudResourceId"] = asset.ResourceID
	if target.Hostname == "" && target.InternalIP == "" {
		return target, errors.New("hostname or internalIp is required")
	}
	return target, nil
}

func buildNodeJoinInventory(req kubernetesNodeRegisterRequest, joinCommandB64 string) (string, error) {
	target := defaultString(strings.TrimSpace(req.InternalIP), strings.TrimSpace(req.Hostname))
	user := defaultString(strings.TrimSpace(req.SSHUser), "root")
	port := req.SSHPort
	if port <= 0 {
		port = 22
	}
	if !nodeRegisterHostPattern.MatchString(target) {
		return "", errors.New("host contains invalid characters")
	}
	if !nodeRegisterUserPattern.MatchString(user) {
		return "", errors.New("sshUser contains invalid characters")
	}
	line := fmt.Sprintf("target ansible_host=%s ansible_user=%s ansible_port=%d ansible_ssh_common_args=%s join_command_b64=%s",
		quoteInventoryValue(target),
		quoteInventoryValue(user),
		port,
		quoteInventoryValue("-o StrictHostKeyChecking=no"),
		quoteInventoryValue(joinCommandB64),
	)
	if req.SSHPassword != "" {
		line += fmt.Sprintf(" ansible_password=%s ansible_become_password=%s",
			quoteInventoryValue(req.SSHPassword),
			quoteInventoryValue(req.SSHPassword),
		)
	}
	return "[targets]\n" + line + "\n", nil
}

func buildNodeJoinPlaybook() string {
	return `---
- hosts: targets
  gather_facts: false
  become: true
  tasks:
    - name: 执行 kubeadm join
      ansible.builtin.shell: "{{ join_command_b64 | b64decode }}"
      args:
        executable: /bin/bash
	`
}

func validateNodeRegisterTaskInput(req kubernetesNodeRegisterRequest, joinCommand string) error {
	host := defaultString(strings.TrimSpace(req.InternalIP), strings.TrimSpace(req.Hostname))
	if host == "" {
		return errors.New("hostname or internalIp is required")
	}
	if req.SSHPort <= 0 || req.SSHPort > 65535 {
		return errors.New("sshPort must be between 1 and 65535")
	}
	if req.SSHUser != "" && !nodeRegisterUserPattern.MatchString(strings.TrimSpace(req.SSHUser)) {
		return errors.New("sshUser contains invalid characters")
	}
	trimmedCommand := strings.TrimSpace(joinCommand)
	if strings.Contains(trimmedCommand, "\n") || strings.Contains(trimmedCommand, "\r") {
		return errors.New("joinCommand must be single line")
	}
	if !strings.HasPrefix(trimmedCommand, "kubeadm join ") && !strings.HasPrefix(trimmedCommand, "echo mock cluster node register") {
		return errors.New("joinCommand must start with `kubeadm join`")
	}
	return nil
}

func quoteInventoryValue(raw string) string {
	clean := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(raw, "\r", " "), "\n", " "), "\t", " "))
	return strconv.Quote(clean)
}

func redactNodeRegisterTaskOutput(raw string) string {
	if raw == "" {
		return ""
	}
	redacted := kubeadmTokenFlagPattern.ReplaceAllString(raw, "${1}***REDACTED***")
	return redacted
}

func readCloudAssetString(metadata datatypes.JSONMap, keys ...string) string {
	for _, key := range keys {
		if value, ok := metadata[key]; ok {
			text := strings.TrimSpace(fmt.Sprintf("%v", value))
			if text != "" && text != "<nil>" {
				return text
			}
		}
	}
	return ""
}

func readCloudAssetInt(metadata datatypes.JSONMap, fallback int, keys ...string) int {
	for _, key := range keys {
		if value, ok := metadata[key]; ok {
			switch typed := value.(type) {
			case int:
				if typed > 0 {
					return typed
				}
			case int64:
				if typed > 0 {
					return int(typed)
				}
			case float64:
				if typed > 0 {
					return int(typed)
				}
			case string:
				n, err := strconv.Atoi(strings.TrimSpace(typed))
				if err == nil && n > 0 {
					return n
				}
			}
		}
	}
	return fallback
}

func normalizeNodeRoles(raw []string) []string {
	seen := map[string]struct{}{}
	roles := make([]string, 0, len(raw))
	for _, item := range raw {
		role := strings.ToLower(strings.TrimSpace(item))
		if role == "" {
			continue
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		roles = append(roles, role)
	}
	if len(roles) == 0 {
		return []string{"worker"}
	}
	return roles
}

func buildNodeRegisterManifest(req kubernetesNodeRegisterRequest, roles []string) map[string]interface{} {
	labels := map[string]interface{}{
		"managedBy": "aiops",
	}
	for key, value := range req.Labels {
		labels[strings.TrimSpace(key)] = value
	}
	for _, role := range roles {
		labels[fmt.Sprintf("node-role.kubernetes.io/%s", role)] = ""
	}
	annotations := map[string]interface{}{
		"aiops.devops/registerSource": "manual-host",
	}
	if req.InternalIP != "" {
		annotations["aiops.devops/internalIP"] = req.InternalIP
	}
	if req.KubeletVersion != "" {
		annotations["aiops.devops/kubeletVersion"] = req.KubeletVersion
	}
	return map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Node",
		"metadata": map[string]interface{}{
			"name":        req.Hostname,
			"labels":      labels,
			"annotations": annotations,
		},
	}
}

func nodeRegisterSpecSummary(req kubernetesNodeRegisterRequest, roles []string) string {
	parts := make([]string, 0, 6)
	if req.CPU != "" {
		parts = append(parts, "cpu="+req.CPU)
	}
	if req.Memory != "" {
		parts = append(parts, "memory="+req.Memory)
	}
	if req.Pods != "" {
		parts = append(parts, "pods="+req.Pods)
	}
	if req.InternalIP != "" {
		parts = append(parts, "ip="+req.InternalIP)
	}
	if len(roles) > 0 {
		parts = append(parts, "roles="+strings.Join(roles, ","))
	}
	if len(parts) == 0 {
		return "registered by aiops"
	}
	return strings.Join(parts, " ")
}

func parseOptionalUintQuery(raw string) uint {
	value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0
	}
	return uint(value)
}

func withKindQuery(rawQuery string, kind string) string {
	values, _ := url.ParseQuery(rawQuery)
	values.Set("kind", kind)
	return values.Encode()
}
