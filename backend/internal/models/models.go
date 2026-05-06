package models

import (
	"time"

	"gorm.io/datatypes"
)

type BaseModel struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type User struct {
	BaseModel
	Username     string `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string `gorm:"size:255;not null" json:"-"`
	DisplayName  string `gorm:"size:128" json:"displayName"`
	Email        string `gorm:"size:128" json:"email"`
	IsActive     bool   `gorm:"default:true" json:"isActive"`
}

type Department struct {
	BaseModel
	Name     string `gorm:"size:128;not null" json:"name"`
	ParentID *uint  `json:"parentId"`
}

type Role struct {
	BaseModel
	Name        string `gorm:"uniqueIndex;size:64;not null" json:"name"`
	Description string `gorm:"size:255" json:"description"`
	BuiltIn     bool   `gorm:"default:false" json:"builtIn"`
}

type Permission struct {
	BaseModel
	Name             string `gorm:"size:128;not null" json:"name"`
	Resource         string `gorm:"size:128;not null" json:"resource"`
	Action           string `gorm:"size:64;not null" json:"action"`
	Type             string `gorm:"size:32;default:api;index" json:"type"`
	Key              string `gorm:"size:128;index" json:"key"`
	DeptScope        string `gorm:"size:64;default:*" json:"deptScope"`
	ResourceTagScope string `gorm:"size:64;default:*" json:"resourceTagScope"`
	EnvScope         string `gorm:"size:64;default:*" json:"envScope"`
	Description      string `gorm:"size:255" json:"description"`
}

type RolePermission struct {
	RoleID       uint `gorm:"primaryKey" json:"roleId"`
	PermissionID uint `gorm:"primaryKey" json:"permissionId"`
}

type UserRole struct {
	UserID uint `gorm:"primaryKey" json:"userId"`
	RoleID uint `gorm:"primaryKey" json:"roleId"`
}

type UserDepartment struct {
	UserID       uint `gorm:"primaryKey" json:"userId"`
	DepartmentID uint `gorm:"primaryKey" json:"departmentId"`
}

type ResourceCategory struct {
	BaseModel
	Name        string         `gorm:"size:128;not null" json:"name"`
	Description string         `gorm:"size:255" json:"description"`
	Schema      datatypes.JSON `json:"schema"`
}

type ResourceItem struct {
	BaseModel
	CIID       string                 `gorm:"size:255;uniqueIndex;not null" json:"ciId"`
	Type       string                 `gorm:"size:64;index;not null" json:"type"`
	Name       string                 `gorm:"size:128;not null" json:"name"`
	CategoryID uint                   `json:"categoryId"`
	Cloud      string                 `gorm:"size:32;index" json:"cloud"`
	Region     string                 `gorm:"size:64;index" json:"region"`
	Env        string                 `gorm:"size:32;index;default:prod" json:"env"`
	Owner      string                 `gorm:"size:128;index" json:"owner"`
	Lifecycle  string                 `gorm:"size:32;default:active" json:"lifecycle"`
	Source     string                 `gorm:"size:32;index;default:Manual" json:"source"`
	LastSeenAt time.Time              `gorm:"index" json:"lastSeenAt"`
	Attributes datatypes.JSONMap      `gorm:"type:jsonb" json:"attributes"`
	Extra      map[string]interface{} `gorm:"-" json:"extra,omitempty"`
}

type Tag struct {
	BaseModel
	Name string `gorm:"size:64;uniqueIndex;not null" json:"name"`
}

type ResourceTag struct {
	ResourceID uint `gorm:"primaryKey" json:"resourceId"`
	TagID      uint `gorm:"primaryKey" json:"tagId"`
}

type ResourceRelation struct {
	BaseModel
	FromCIID          string            `gorm:"size:255;index;not null" json:"fromCiId"`
	ToCIID            string            `gorm:"size:255;index;not null" json:"toCiId"`
	RelationType      string            `gorm:"size:64;index;not null" json:"relationType"`
	Direction         string            `gorm:"size:16;default:outbound" json:"direction"`
	Criticality       string            `gorm:"size:8;default:P2" json:"criticality"`
	Confidence        float64           `gorm:"default:1" json:"confidence"`
	Evidence          datatypes.JSONMap `gorm:"type:jsonb" json:"evidence"`
	RelationUpdatedAt time.Time         `gorm:"index" json:"relationUpdatedAt"`
}

type ResourceEvidence struct {
	BaseModel
	CIID       string            `gorm:"size:255;index;not null" json:"ciId"`
	RelationID *uint             `gorm:"index" json:"relationId,omitempty"`
	Source     string            `gorm:"size:32;index;not null" json:"source"`
	RawID      string            `gorm:"size:255;index" json:"rawId"`
	Payload    datatypes.JSONMap `gorm:"type:jsonb" json:"payload"`
	ObservedAt time.Time         `gorm:"index" json:"observedAt"`
}

type ResourceSyncJob struct {
	BaseModel
	Status           string            `gorm:"size:32;index;default:pending" json:"status"`
	RequestedSources datatypes.JSON    `gorm:"type:jsonb" json:"requestedSources"`
	FullScan         bool              `gorm:"default:false" json:"fullScan"`
	StartedAt        *time.Time        `gorm:"index" json:"startedAt,omitempty"`
	FinishedAt       *time.Time        `gorm:"index" json:"finishedAt,omitempty"`
	Summary          datatypes.JSONMap `gorm:"type:jsonb" json:"summary"`
}

type ResourceSyncJobItem struct {
	BaseModel
	JobID        uint              `gorm:"index;not null" json:"jobId"`
	CIID         string            `gorm:"size:255;index;not null" json:"ciId"`
	Source       string            `gorm:"size:32;index;not null" json:"source"`
	Action       string            `gorm:"size:32" json:"action"`
	Status       string            `gorm:"size:32;index" json:"status"`
	Message      string            `gorm:"size:255" json:"message"`
	QualityScore float64           `gorm:"default:1" json:"qualityScore"`
	Data         datatypes.JSONMap `gorm:"type:jsonb" json:"data"`
}

type Task struct {
	BaseModel
	Name          string `gorm:"size:128;not null" json:"name"`
	Description   string `gorm:"size:255" json:"description"`
	PlaybookID    uint   `json:"playbookId"`
	InventoryFrom string `gorm:"size:32;default:cmdb" json:"inventoryFrom"`
	IsHighRisk    bool   `gorm:"default:false" json:"isHighRisk"`
}

type Playbook struct {
	BaseModel
	Name    string `gorm:"size:128;not null" json:"name"`
	Content string `gorm:"type:text;not null" json:"content"`
}

type TaskExecutionLog struct {
	BaseModel
	JobID      string `gorm:"size:64;index;not null" json:"jobId"`
	TaskID     uint   `json:"taskId"`
	Command    string `gorm:"type:text" json:"command"`
	ExitCode   int    `json:"exitCode"`
	Summary    string `gorm:"type:text" json:"summary"`
	Stdout     string `gorm:"type:text" json:"stdout"`
	Stderr     string `gorm:"type:text" json:"stderr"`
	Status     string `gorm:"size:32;index" json:"status"`
	RetryCount int    `json:"retryCount"`
}

type CloudAccount struct {
	BaseModel
	Provider   string `gorm:"size:32;index;not null" json:"provider"`
	Name       string `gorm:"size:128;not null" json:"name"`
	AccessKey  string `gorm:"size:255;not null" json:"accessKey"`
	SecretKey  string `gorm:"size:255;not null" json:"secretKey"`
	Region     string `gorm:"size:64" json:"region"`
	IsVerified bool   `gorm:"default:false" json:"isVerified"`
}

type CloudAsset struct {
	BaseModel
	Provider     string            `gorm:"size:32;index;not null;uniqueIndex:idx_cloud_asset_unique" json:"provider"`
	AccountID    uint              `gorm:"index;not null;default:0;uniqueIndex:idx_cloud_asset_unique" json:"accountId"`
	Region       string            `gorm:"size:64;index;uniqueIndex:idx_cloud_asset_unique" json:"region"`
	Type         string            `gorm:"size:64;index;not null;uniqueIndex:idx_cloud_asset_unique" json:"type"`
	ResourceID   string            `gorm:"size:255;index;not null;uniqueIndex:idx_cloud_asset_unique" json:"resourceId"`
	Name         string            `gorm:"size:128;index;not null" json:"name"`
	Status       string            `gorm:"size:32;index;default:unknown" json:"status"`
	Source       string            `gorm:"size:32;index;default:Manual" json:"source"`
	Tags         datatypes.JSONMap `gorm:"type:jsonb" json:"tags"`
	Metadata     datatypes.JSONMap `gorm:"type:jsonb" json:"metadata"`
	LastSyncedAt *time.Time        `gorm:"index" json:"lastSyncedAt,omitempty"`
	ExpiresAt    *time.Time        `gorm:"index" json:"expiresAt,omitempty"`
}

type CloudSyncJob struct {
	BaseModel
	AccountID  uint              `gorm:"index;not null" json:"accountId"`
	Provider   string            `gorm:"size:32;index;not null" json:"provider"`
	Region     string            `gorm:"size:64;index" json:"region"`
	Status     string            `gorm:"size:32;index;default:running" json:"status"`
	StartedAt  *time.Time        `gorm:"index" json:"startedAt,omitempty"`
	FinishedAt *time.Time        `gorm:"index" json:"finishedAt,omitempty"`
	Summary    datatypes.JSONMap `gorm:"type:jsonb" json:"summary"`
}

type Ticket struct {
	BaseModel
	Title       string `gorm:"size:255;not null" json:"title"`
	Description string `gorm:"type:text" json:"description"`
	Status      string `gorm:"size:32;index;default:pending" json:"status"`
	Priority    string `gorm:"size:32;default:medium" json:"priority"`
}

type Event struct {
	BaseModel
	Source   string `gorm:"size:64;index" json:"source"`
	Level    string `gorm:"size:32;index" json:"level"`
	Type     string `gorm:"size:64;index" json:"type"`
	Summary  string `gorm:"type:text" json:"summary"`
	Metadata string `gorm:"type:text" json:"metadata"`
}

type InAppMessage struct {
	BaseModel
	TraceID string            `gorm:"size:64;index" json:"traceId"`
	Channel string            `gorm:"size:64;index;not null" json:"channel"`
	Target  string            `gorm:"size:128;index;not null" json:"target"`
	Title   string            `gorm:"size:255" json:"title"`
	Content string            `gorm:"type:text;not null" json:"content"`
	Data    datatypes.JSONMap `gorm:"type:jsonb" json:"data"`
	Read    bool              `gorm:"default:false" json:"read"`
}

type MessageReadReceipt struct {
	BaseModel
	MessageID uint      `gorm:"uniqueIndex:idx_message_read_user;index;not null" json:"messageId"`
	UserID    uint      `gorm:"uniqueIndex:idx_message_read_user;index;not null" json:"userId"`
	ReadAt    time.Time `gorm:"index;not null" json:"readAt"`
}

type ToolItem struct {
	BaseModel
	Name        string `gorm:"size:128;not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	Entry       string `gorm:"size:255;not null" json:"entry"`
	Enabled     bool   `gorm:"default:true" json:"enabled"`
}

type DockerHost struct {
	BaseModel
	Name            string            `gorm:"size:128;not null" json:"name"`
	Endpoint        string            `gorm:"size:255;not null" json:"endpoint"`
	TLSEnable       bool              `gorm:"default:false" json:"tlsEnable"`
	Env             string            `gorm:"size:32;index;default:prod" json:"env"`
	Owner           string            `gorm:"size:128;index" json:"owner"`
	Status          string            `gorm:"size:32;index;default:unknown" json:"status"`
	Version         string            `gorm:"size:64" json:"version"`
	Labels          datatypes.JSONMap `gorm:"type:jsonb" json:"labels"`
	Metadata        datatypes.JSONMap `gorm:"type:jsonb" json:"metadata"`
	LastHeartbeatAt *time.Time        `gorm:"index" json:"lastHeartbeatAt,omitempty"`
}

type DockerComposeStack struct {
	BaseModel
	HostID         uint       `gorm:"index;not null" json:"hostId"`
	Name           string     `gorm:"size:128;not null" json:"name"`
	Status         string     `gorm:"size:32;index;default:draft" json:"status"`
	Services       int        `gorm:"default:0" json:"services"`
	Content        string     `gorm:"type:text;not null" json:"content"`
	LastDeployedAt *time.Time `gorm:"index" json:"lastDeployedAt,omitempty"`
}

type DockerOperation struct {
	BaseModel
	TraceID      string            `gorm:"size:64;index;not null" json:"traceId"`
	HostID       uint              `gorm:"index;not null" json:"hostId"`
	ResourceType string            `gorm:"size:64;index;not null" json:"resourceType"`
	ResourceID   string            `gorm:"size:255;index" json:"resourceId"`
	Action       string            `gorm:"size:64;index;not null" json:"action"`
	Status       string            `gorm:"size:32;index;not null" json:"status"`
	DryRun       bool              `gorm:"index;default:true" json:"dryRun"`
	RiskLevel    string            `gorm:"size:16;index;default:P2" json:"riskLevel"`
	Request      datatypes.JSONMap `gorm:"type:jsonb" json:"request"`
	Result       datatypes.JSONMap `gorm:"type:jsonb" json:"result"`
	ErrorMessage string            `gorm:"type:text" json:"errorMessage"`
	StartedAt    *time.Time        `gorm:"index" json:"startedAt,omitempty"`
	FinishedAt   *time.Time        `gorm:"index" json:"finishedAt,omitempty"`
}

type MiddlewareInstance struct {
	BaseModel
	Name       string `gorm:"size:128;not null" json:"name"`
	Type       string `gorm:"size:64;not null" json:"type"`
	Endpoint   string `gorm:"size:255;not null" json:"endpoint"`
	HealthPath string `gorm:"size:255" json:"healthPath"`
}

type ObservabilitySource struct {
	BaseModel
	Name      string `gorm:"size:128;not null" json:"name"`
	Type      string `gorm:"size:64;not null" json:"type"`
	Endpoint  string `gorm:"size:255;not null" json:"endpoint"`
	AuthToken string `gorm:"size:255" json:"authToken"`
}

type KubernetesCluster struct {
	BaseModel
	Name       string `gorm:"size:128;not null" json:"name"`
	APIServer  string `gorm:"size:255;not null" json:"apiServer"`
	KubeConfig string `gorm:"type:text;not null" json:"kubeConfig"`
}

type AIAgentConfig struct {
	BaseModel
	Name    string `gorm:"size:128;not null" json:"name"`
	Type    string `gorm:"size:32;not null" json:"type"`
	Config  string `gorm:"type:text;not null" json:"config"`
	Enabled bool   `gorm:"default:true" json:"enabled"`
}

type AIModelConfig struct {
	BaseModel
	Name      string `gorm:"size:128;not null" json:"name"`
	Provider  string `gorm:"size:32;not null" json:"provider"`
	Endpoint  string `gorm:"size:255;not null" json:"endpoint"`
	APIKey    string `gorm:"size:255;not null" json:"apiKey"`
	ModelName string `gorm:"size:128;not null" json:"modelName"`
	Enabled   bool   `gorm:"default:true" json:"enabled"`
}

type AuditLog struct {
	BaseModel
	Actor      string `gorm:"size:128;index" json:"actor"`
	Action     string `gorm:"size:64;index" json:"action"`
	Resource   string `gorm:"size:128;index" json:"resource"`
	ResourceID string `gorm:"size:64;index" json:"resourceId"`
	Path       string `gorm:"size:255" json:"path"`
	Method     string `gorm:"size:16" json:"method"`
	Payload    string `gorm:"type:text" json:"payload"`
}

func AutoMigrateModels() []interface{} {
	return []interface{}{
		&User{},
		&Department{},
		&Role{},
		&Permission{},
		&RolePermission{},
		&UserRole{},
		&UserDepartment{},
		&ResourceCategory{},
		&ResourceItem{},
		&Tag{},
		&ResourceTag{},
		&ResourceRelation{},
		&ResourceEvidence{},
		&ResourceSyncJob{},
		&ResourceSyncJobItem{},
		&Task{},
		&Playbook{},
		&TaskExecutionLog{},
		&CloudAccount{},
		&CloudAsset{},
		&CloudSyncJob{},
		&Ticket{},
		&Event{},
		&InAppMessage{},
		&MessageReadReceipt{},
		&ToolItem{},
		&DockerHost{},
		&DockerComposeStack{},
		&DockerOperation{},
		&MiddlewareInstance{},
		&ObservabilitySource{},
		&KubernetesCluster{},
		&AIAgentConfig{},
		&AIModelConfig{},
		&AuditLog{},
	}
}
