package db

import (
	"context"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"devops-system/backend/internal/models"
)

func InitPostgres(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("POSTGRES_DSN is required")
	}
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := database.AutoMigrate(models.AutoMigrateModels()...); err != nil {
		return nil, err
	}
	return database, nil
}

func InitRedis(addr string, password string, redisDB int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       redisDB,
	})
}

func PingRedis(client *redis.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return client.Ping(ctx).Err()
}

func InitRabbitMQ(url string) (*amqp091.Connection, error) {
	if url == "" {
		return nil, nil
	}
	return amqp091.Dial(url)
}
