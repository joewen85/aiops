package app

import (
	"strings"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops-system/backend/internal/auth"
	"devops-system/backend/internal/config"
	"devops-system/backend/internal/handler"
	"devops-system/backend/internal/middleware"
)

func setupRouter(h *handler.Handler, jwtManager auth.Manager, enforcer *casbin.Enforcer, database *gorm.DB, cfg config.Config) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery(), cors.New(buildCORSConfig(cfg)))
	engine.Use(middleware.OptionalAuth(jwtManager), middleware.AuditLogger(database))

	h.RegisterHealthRoutes(engine)
	engine.GET("/ws", h.WebSocket)

	v1 := engine.Group("/api/v1")
	{
		authGroup := v1.Group("/auth")
		authGroup.POST("/login", h.Login)
	}

	authOnly := v1.Group("")
	authOnly.Use(middleware.AuthRequired(jwtManager))
	{
		authOnly.GET("/auth/me/permissions", h.MePermissions)
	}

	secured := v1.Group("")
	roleCacheTTL := time.Duration(cfg.PermissionRuntimeCacheTTLMS) * time.Millisecond
	secured.Use(middleware.AuthRequired(jwtManager), middleware.PermissionRequiredWithRuntimeCache(enforcer, database, cfg.ABACHeaderSignSecret, roleCacheTTL))
	{
		registerUserDepartmentRoutes(secured, h)
		registerRBACRoutes(secured, h)
		registerCMDBRoutes(secured, h)
		registerTaskRoutes(secured, h)
		registerMessageRoutes(secured, h)
		registerPhase2Routes(secured, h)
		registerPhase3Routes(secured, h)
		secured.GET("/audit-logs", h.ListAuditLogs)
	}

	return engine
}

func buildCORSConfig(cfg config.Config) cors.Config {
	c := cors.Config{
		AllowMethods: []string{
			"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
			"X-Resource-Tag",
			"X-Env",
			"X-ABAC-Timestamp",
			"X-ABAC-Signature",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"Content-Type",
		},
		AllowCredentials: cfg.CORSAllowCredentials,
		MaxAge:           12 * time.Hour,
	}

	origins := strings.TrimSpace(cfg.CORSAllowOrigins)
	if origins == "" || origins == "*" {
		c.AllowAllOrigins = true
		return c
	}

	for _, item := range strings.Split(origins, ",") {
		origin := strings.TrimSpace(item)
		if origin == "" {
			continue
		}
		c.AllowOrigins = append(c.AllowOrigins, origin)
	}
	if len(c.AllowOrigins) == 0 {
		c.AllowAllOrigins = true
	}
	return c
}

func registerUserDepartmentRoutes(r *gin.RouterGroup, h *handler.Handler) {
	r.GET("/users", h.ListUsers)
	r.POST("/users", h.CreateUser)
	r.GET("/users/:id", h.GetUser)
	r.PUT("/users/:id", h.UpdateUser)
	r.DELETE("/users/:id", h.DeleteUser)
	r.PATCH("/users/:id/status", h.ToggleUserStatus)
	r.POST("/users/:id/reset-password", h.ResetUserPassword)
	r.GET("/users/:id/roles", h.GetUserRoles)
	r.POST("/users/:id/roles", h.BindUserRoles)
	r.GET("/users/:id/departments", h.GetUserDepartments)
	r.POST("/users/:id/departments", h.BindUserDepartments)

	r.GET("/departments", h.ListDepartments)
	r.POST("/departments", h.CreateDepartment)
	r.GET("/departments/tree", h.ListDepartmentTree)
	r.GET("/departments/:id", h.GetDepartment)
	r.PUT("/departments/:id", h.UpdateDepartment)
	r.DELETE("/departments/:id", h.DeleteDepartment)
	r.GET("/departments/:id/users", h.GetDepartmentUsers)
	r.POST("/departments/:id/users", h.BindDepartmentUsers)
}

func registerRBACRoutes(r *gin.RouterGroup, h *handler.Handler) {
	r.GET("/roles", h.ListRoles)
	r.POST("/roles", h.CreateRole)
	r.GET("/roles/:id", h.GetRole)
	r.PUT("/roles/:id", h.UpdateRole)
	r.DELETE("/roles/:id", h.DeleteRole)
	r.GET("/roles/:id/permissions", h.GetRolePermissions)
	r.POST("/roles/:id/permissions", h.BindRolePermissions)

	r.GET("/permissions", h.ListPermissions)
	r.POST("/permissions", h.CreatePermission)
	r.GET("/permissions/:id", h.GetPermission)
	r.PUT("/permissions/:id", h.UpdatePermission)
	r.DELETE("/permissions/:id", h.DeletePermission)
}

func registerCMDBRoutes(r *gin.RouterGroup, h *handler.Handler) {
	cmdb := r.Group("/cmdb")
	cmdb.GET("/categories", h.ListResourceCategories)
	cmdb.POST("/categories", h.CreateResourceCategory)
	cmdb.GET("/categories/:id", h.GetResourceCategory)
	cmdb.PUT("/categories/:id", h.UpdateResourceCategory)
	cmdb.DELETE("/categories/:id", h.DeleteResourceCategory)

	cmdb.GET("/resources", h.ListResources)
	cmdb.POST("/resources", h.CreateResource)
	cmdb.GET("/resources/:id", h.GetResource)
	cmdb.PUT("/resources/:id", h.UpdateResource)
	cmdb.DELETE("/resources/:id", h.DeleteResource)
	cmdb.POST("/resources/:id/actions/restart", h.RestartCMDBResource)
	cmdb.POST("/resources/:id/actions/stop", h.StopCMDBResource)
	cmdb.POST("/resources/:id/tags", h.BindResourceTags)
	cmdb.GET("/resources/:id/upstream", h.GetResourceUpstream)
	cmdb.GET("/resources/:id/downstream", h.GetResourceDownstream)

	cmdb.GET("/tags", h.ListTags)
	cmdb.POST("/tags", h.CreateTag)
	cmdb.GET("/tags/:id", h.GetTag)
	cmdb.PUT("/tags/:id", h.UpdateTag)
	cmdb.DELETE("/tags/:id", h.DeleteTag)

	cmdb.GET("/relations", h.ListResourceRelations)
	cmdb.POST("/relations", h.CreateResourceRelation)
	cmdb.GET("/topology/:application", h.GetApplicationTopology)
	cmdb.GET("/impact/:ciId", h.GetResourceImpact)
	cmdb.GET("/regions/:region/failover", h.GetRegionFailover)
	cmdb.GET("/change-impact/:releaseId", h.GetChangeImpact)
	cmdb.POST("/sync/jobs", h.CreateCMDBSyncJob)
	cmdb.GET("/sync/jobs/:id", h.GetCMDBSyncJob)
	cmdb.POST("/sync/jobs/:id/retry", h.RetryCMDBSyncJob)
}

func registerTaskRoutes(r *gin.RouterGroup, h *handler.Handler) {
	r.GET("/tasks", h.ListTasks)
	r.POST("/tasks", h.CreateTask)
	r.GET("/tasks/:id", h.GetTask)
	r.PUT("/tasks/:id", h.UpdateTask)
	r.DELETE("/tasks/:id", h.DeleteTask)
	r.POST("/tasks/:id/execute", h.ExecuteTask)

	r.GET("/playbooks", h.ListPlaybooks)
	r.POST("/playbooks", h.CreatePlaybook)
	r.GET("/playbooks/:id", h.GetPlaybook)
	r.PUT("/playbooks/:id", h.UpdatePlaybook)
	r.DELETE("/playbooks/:id", h.DeletePlaybook)

	r.GET("/task-logs", h.ListTaskLogs)
	r.GET("/task-logs/:id", h.GetTaskLog)
}

func registerMessageRoutes(r *gin.RouterGroup, h *handler.Handler) {
	r.GET("/messages", h.ListMessages)
	r.POST("/messages", h.CreateMessage)
	r.POST("/messages/:id/read", h.MarkMessageRead)
}

func registerPhase2Routes(r *gin.RouterGroup, h *handler.Handler) {
	r.GET("/cloud/accounts", h.ListCloudAccounts)
	r.POST("/cloud/accounts", h.CreateCloudAccount)
	r.GET("/cloud/accounts/:id", h.GetCloudAccount)
	r.PUT("/cloud/accounts/:id", h.UpdateCloudAccount)
	r.DELETE("/cloud/accounts/:id", h.DeleteCloudAccount)
	r.POST("/cloud/accounts/:id/verify", h.VerifyCloudAccount)
	r.POST("/cloud/accounts/:id/sync", h.SyncCloudAccount)
	r.GET("/cloud/accounts/:id/assets", h.ListCloudAccountAssets)
	r.GET("/cloud/assets", h.ListCloudAssets)
	r.POST("/cloud/assets", h.CreateCloudAsset)
	r.GET("/cloud/assets/:id", h.GetCloudAsset)
	r.PUT("/cloud/assets/:id", h.UpdateCloudAsset)
	r.DELETE("/cloud/assets/:id", h.DeleteCloudAsset)

	r.GET("/tickets", h.ListTickets)
	r.POST("/tickets", h.CreateTicket)
	r.GET("/tickets/:id", h.GetTicket)
	r.PUT("/tickets/:id", h.UpdateTicket)
	r.DELETE("/tickets/:id", h.DeleteTicket)
	r.POST("/tickets/:id/approve", h.ApproveTicket)
	r.POST("/tickets/:id/transition", h.TransitionTicket)

	r.GET("/docker/hosts", h.ListDockerHosts)
	r.POST("/docker/hosts", h.CreateDockerHost)
	r.GET("/docker/hosts/:id", h.GetDockerHost)
	r.PUT("/docker/hosts/:id", h.UpdateDockerHost)
	r.DELETE("/docker/hosts/:id", h.DeleteDockerHost)
	r.POST("/docker/hosts/:id/check", h.CheckDockerHost)
	r.GET("/docker/hosts/:id/resources", h.ListDockerHostResources)
	r.GET("/docker/compose/stacks", h.ListComposeStacks)
	r.POST("/docker/compose/stacks", h.CreateComposeStack)
	r.PUT("/docker/compose/stacks/:id", h.UpdateComposeStack)
	r.DELETE("/docker/compose/stacks/:id", h.DeleteComposeStack)
	r.GET("/docker/operations", h.ListDockerOperations)
	r.GET("/docker/aiops/protocol", h.DockerAIOpsProtocol)
	r.POST("/docker/actions", h.DockerAction)

	r.GET("/middleware/instances", h.ListMiddlewareInstances)
	r.POST("/middleware/instances", h.CreateMiddlewareInstance)
	r.GET("/middleware/instances/:id", h.GetMiddlewareInstance)
	r.PUT("/middleware/instances/:id", h.UpdateMiddlewareInstance)
	r.DELETE("/middleware/instances/:id", h.DeleteMiddlewareInstance)
	r.POST("/middleware/instances/:id/check", h.CheckMiddlewareInstance)
	r.GET("/middleware/instances/:id/metrics", h.ListMiddlewareMetrics)
	r.POST("/middleware/instances/:id/metrics/collect", h.CollectMiddlewareMetrics)
	r.POST("/middleware/instances/:id/action", h.MiddlewareAction)
	r.GET("/middleware/operations", h.ListMiddlewareOperations)
	r.GET("/middleware/operations/:id", h.GetMiddlewareOperation)
	r.GET("/middleware/aiops/protocol", h.MiddlewareAIOpsProtocol)
	r.POST("/middleware/actions", h.MiddlewareAction)

	r.GET("/observability/sources", h.ListObservabilitySources)
	r.POST("/observability/sources", h.CreateObservabilitySource)
	r.GET("/observability/sources/:id", h.GetObservabilitySource)
	r.PUT("/observability/sources/:id", h.UpdateObservabilitySource)
	r.DELETE("/observability/sources/:id", h.DeleteObservabilitySource)
	r.GET("/observability/metrics/query", h.QueryMetrics)

	r.GET("/kubernetes/clusters", h.ListKubernetesClusters)
	r.POST("/kubernetes/clusters", h.CreateKubernetesCluster)
	r.GET("/kubernetes/clusters/:id", h.GetKubernetesCluster)
	r.PUT("/kubernetes/clusters/:id", h.UpdateKubernetesCluster)
	r.DELETE("/kubernetes/clusters/:id", h.DeleteKubernetesCluster)
	r.GET("/kubernetes/nodes", h.ListKubernetesNodes)
	r.POST("/kubernetes/resources/action", h.KubernetesResourceAction)

	r.GET("/events", h.ListEvents)
	r.POST("/events", h.CreateEvent)
	r.GET("/events/:id", h.GetEvent)
	r.PUT("/events/:id", h.UpdateEvent)
	r.DELETE("/events/:id", h.DeleteEvent)
	r.GET("/events/search", h.SearchEvents)
	r.POST("/events/:id/link", h.LinkEvent)

	r.GET("/tool-market/tools", h.ListTools)
	r.POST("/tool-market/tools", h.CreateTool)
	r.GET("/tool-market/tools/:id", h.GetTool)
	r.PUT("/tool-market/tools/:id", h.UpdateTool)
	r.DELETE("/tool-market/tools/:id", h.DeleteTool)
	r.POST("/tool-market/tools/:id/execute", h.ExecuteTool)
}

func registerPhase3Routes(r *gin.RouterGroup, h *handler.Handler) {
	r.GET("/aiops/agents", h.ListAIAgents)
	r.POST("/aiops/agents", h.CreateAIAgent)
	r.GET("/aiops/agents/:id", h.GetAIAgent)
	r.PUT("/aiops/agents/:id", h.UpdateAIAgent)
	r.DELETE("/aiops/agents/:id", h.DeleteAIAgent)

	r.GET("/aiops/models", h.ListAIModels)
	r.POST("/aiops/models", h.CreateAIModel)
	r.GET("/aiops/models/:id", h.GetAIModel)
	r.PUT("/aiops/models/:id", h.UpdateAIModel)
	r.DELETE("/aiops/models/:id", h.DeleteAIModel)

	r.POST("/aiops/chat", h.AIOpsChat)
	r.POST("/aiops/rca", h.AIOpsRCA)
	r.GET("/aiops/procurement/protocol", h.AIOpsProcurementProtocol)
	r.POST("/aiops/procurement/intents", h.AIOpsParseProcurementIntent)
	r.POST("/aiops/procurement/plans", h.AIOpsBuildProcurementPlan)
	r.POST("/aiops/procurement/executions", h.AIOpsExecuteProcurementPlan)
}
