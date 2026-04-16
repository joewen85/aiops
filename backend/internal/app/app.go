package app

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"devops-system/backend/internal/auth"
	"devops-system/backend/internal/config"
	"devops-system/backend/internal/db"
	"devops-system/backend/internal/executor"
	"devops-system/backend/internal/handler"
	"devops-system/backend/internal/rbac"
	"devops-system/backend/internal/service"
	"devops-system/backend/internal/ws"
)

type App struct {
	Config config.Config
	DB     *gorm.DB
	Redis  *redis.Client
	Rabbit *amqp091.Connection
	Engine *gin.Engine
}

func New() (*App, error) {
	cfg := config.Load()

	database, err := db.InitPostgres(cfg.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("init postgres failed: %w", err)
	}

	redisClient := db.InitRedis(cfg.RedisAddr, cfg.RedisPass, cfg.RedisDB)
	if err := db.PingRedis(redisClient); err != nil {
		log.Printf("redis ping warning: %v", err)
	}

	rabbitConn, err := db.InitRabbitMQ(cfg.RabbitMQURL)
	if err != nil {
		log.Printf("rabbitmq connection warning: %v", err)
	}

	if err := service.SeedRBACDefaults(database); err != nil {
		return nil, fmt.Errorf("seed RBAC defaults failed: %w", err)
	}

	enforcer, err := rbac.InitEnforcer(database, "config/casbin_model.conf")
	if err != nil {
		return nil, fmt.Errorf("init casbin failed: %w", err)
	}

	jwtManager := auth.NewManager(cfg.JWTSecret, cfg.JWTExpire)
	hub := ws.NewHub(redisClient, "devops:ws:messages")
	execRunner := executor.Runner{
		BinPath: cfg.AnsibleBin,
		TmpDir:  cfg.PlaybookTmpDir,
	}

	h := handler.New(database, redisClient, rabbitConn, jwtManager, enforcer, hub, execRunner, cfg)
	engine := setupRouter(h, jwtManager, enforcer, database, cfg)

	return &App{
		Config: cfg,
		DB:     database,
		Redis:  redisClient,
		Rabbit: rabbitConn,
		Engine: engine,
	}, nil
}
