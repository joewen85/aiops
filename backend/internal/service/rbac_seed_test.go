package service

import (
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"devops-system/backend/internal/models"
)

func TestSeedRBACDefaults_Idempotent(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(memoryDSN(t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := database.AutoMigrate(models.AutoMigrateModels()...); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	if err := SeedRBACDefaults(database); err != nil {
		t.Fatalf("first seed failed: %v", err)
	}
	if err := SeedRBACDefaults(database); err != nil {
		t.Fatalf("second seed failed: %v", err)
	}

	var adminRole models.Role
	if err := database.Where("name = ?", "admin").First(&adminRole).Error; err != nil {
		t.Fatalf("query admin role failed: %v", err)
	}

	var adminUser models.User
	if err := database.Where("username = ?", "admin").First(&adminUser).Error; err != nil {
		t.Fatalf("query admin user failed: %v", err)
	}

	var count int64
	if err := database.Model(&models.Permission{}).Count(&count).Error; err != nil {
		t.Fatalf("count permissions failed: %v", err)
	}
	if count == 0 {
		t.Fatalf("expected seeded permissions > 0")
	}

	assertPermissionExists(t, database, "menu", "menu.dashboard")
	assertPermissionExists(t, database, "button", "button.rbac.role.create")
	assertPermissionExists(t, database, "api", "api.rbac.role.list")

	var rolePermissionCount int64
	if err := database.Model(&models.RolePermission{}).Where("role_id = ?", adminRole.ID).Count(&rolePermissionCount).Error; err != nil {
		t.Fatalf("count role permissions failed: %v", err)
	}
	if rolePermissionCount != count {
		t.Fatalf("expected admin role permissions=%d got=%d", count, rolePermissionCount)
	}
}

func assertPermissionExists(t *testing.T, database *gorm.DB, permType string, key string) {
	t.Helper()
	var item models.Permission
	if err := database.Where("type = ? AND key = ?", permType, key).First(&item).Error; err != nil {
		t.Fatalf("permission not found type=%s key=%s err=%v", permType, key, err)
	}
}

func memoryDSN(testName string) string {
	return fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(testName, "/", "_"))
}
