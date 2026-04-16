package rbac

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type casbinRuleRow struct {
	ID    uint   `gorm:"primaryKey;autoIncrement"`
	Ptype string `gorm:"column:ptype;size:100"`
	V0    string `gorm:"column:v0;size:100"`
	V1    string `gorm:"column:v1;size:100"`
	V2    string `gorm:"column:v2;size:100"`
	V3    string `gorm:"column:v3;size:100"`
	V4    string `gorm:"column:v4;size:100"`
	V5    string `gorm:"column:v5;size:100"`
}

func (casbinRuleRow) TableName() string {
	return "casbin_rule"
}

func TestInitEnforcerNormalizesLegacyScopeColumns(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	database, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := database.AutoMigrate(&casbinRuleRow{}); err != nil {
		t.Fatalf("migrate casbin_rule failed: %v", err)
	}

	legacy := casbinRuleRow{
		Ptype: "p",
		V0:    "ops",
		V1:    "/api/v1/tasks",
		V2:    "GET",
		V3:    "",
		V4:    "",
		V5:    "",
	}
	if err := database.Create(&legacy).Error; err != nil {
		t.Fatalf("insert legacy rule failed: %v", err)
	}

	modelPath := filepath.Join("..", "..", "config", "casbin_model.conf")
	enforcer, err := InitEnforcer(database, modelPath)
	if err != nil {
		t.Fatalf("init enforcer failed: %v", err)
	}

	policies, err := enforcer.GetFilteredPolicy(0, "ops")
	if err != nil {
		t.Fatalf("get policy failed: %v", err)
	}
	if len(policies) != 1 {
		t.Fatalf("expected 1 policy, got %d (%v)", len(policies), policies)
	}
	if policies[0][3] != "*" || policies[0][4] != "*" || policies[0][5] != "*" {
		t.Fatalf("legacy scope not normalized: %v", policies[0])
	}
}
