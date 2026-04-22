package handler

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"devops-system/backend/internal/config"
	"devops-system/backend/internal/models"
)

func TestTryAcquireCMDBSyncLockWithoutRedis(t *testing.T) {
	h := newCMDBLockHandlerForTest(t)
	release, acquired, err := h.tryAcquireCMDBSyncLock()
	if err != nil {
		t.Fatalf("try acquire cmdb sync lock failed: %v", err)
	}
	if !acquired {
		t.Fatalf("expected lock acquired without running job")
	}
	release()

	now := time.Now()
	if err := h.DB.Create(&models.ResourceSyncJob{
		Status:    "running",
		StartedAt: &now,
	}).Error; err != nil {
		t.Fatalf("seed running cmdb sync job failed: %v", err)
	}
	_, acquired, err = h.tryAcquireCMDBSyncLock()
	if err != nil {
		t.Fatalf("try acquire cmdb sync lock with running job failed: %v", err)
	}
	if acquired {
		t.Fatalf("expected lock rejected when running job exists")
	}
}

func TestTryAcquireCMDBSyncLockRedisErrorFallback(t *testing.T) {
	h := newCMDBLockHandlerForTest(t)
	h.Redis = redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:0",
		DB:   0,
	})
	defer h.Redis.Close()

	release, acquired, err := h.tryAcquireCMDBSyncLock()
	if err != nil {
		t.Fatalf("try acquire cmdb sync lock failed: %v", err)
	}
	if !acquired {
		t.Fatalf("expected fallback lock acquired when db has no running job")
	}
	release()
}

func TestCMDBSyncRedisLockTTLDefault(t *testing.T) {
	h := &Handler{Config: config.Config{}}
	if got := h.cmdbSyncRedisLockTTL(); got != 30*time.Minute {
		t.Fatalf("expected default lock ttl=30m got=%s", got)
	}
}

func TestCMDBSyncRedisLockTTLCustom(t *testing.T) {
	h := &Handler{Config: config.Config{CMDBSyncRedisLockTTLSeconds: 45}}
	if got := h.cmdbSyncRedisLockTTL(); got != 45*time.Second {
		t.Fatalf("expected custom lock ttl=45s got=%s", got)
	}
}

func newCMDBLockHandlerForTest(t *testing.T) *Handler {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	database, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := database.AutoMigrate(models.AutoMigrateModels()...); err != nil {
		t.Fatalf("auto migrate models failed: %v", err)
	}
	return &Handler{DB: database}
}
