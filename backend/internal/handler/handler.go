package handler

import (
	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"devops-system/backend/internal/ai"
	"devops-system/backend/internal/auth"
	"devops-system/backend/internal/cloud"
	"devops-system/backend/internal/config"
	"devops-system/backend/internal/executor"
	"devops-system/backend/internal/ws"
)

type Handler struct {
	DB             *gorm.DB
	Redis          *redis.Client
	Rabbit         *amqp091.Connection
	JWT            auth.Manager
	Enforcer       *casbin.Enforcer
	Hub            *ws.Hub
	Executor       executor.Runner
	Config         config.Config
	CloudProviders map[string]cloud.Provider
	CloudCollector cloud.ResourceCollector
	ModelProviders map[string]ai.ModelProvider
	Procurement    ai.ProcurementEngine
}

func New(
	database *gorm.DB,
	redisClient *redis.Client,
	rabbit *amqp091.Connection,
	jwtManager auth.Manager,
	enforcer *casbin.Enforcer,
	hub *ws.Hub,
	execRunner executor.Runner,
	cfg config.Config,
) *Handler {
	return &Handler{
		DB:             database,
		Redis:          redisClient,
		Rabbit:         rabbit,
		JWT:            jwtManager,
		Enforcer:       enforcer,
		Hub:            hub,
		Executor:       execRunner,
		Config:         cfg,
		CloudProviders: buildCloudProviders(cfg),
		CloudCollector: cloud.NewDefaultResourceCollector(),
		ModelProviders: map[string]ai.ModelProvider{
			"openai":    ai.OpenAIProvider{},
			"anthropic": ai.AnthropicProvider{},
		},
		Procurement: ai.NewStubProcurementEngine(),
	}
}

func (h *Handler) RegisterHealthRoutes(r *gin.Engine) {
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{"status": "up"}})
	})
}
