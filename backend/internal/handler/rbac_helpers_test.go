package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"devops-system/backend/internal/models"
	"devops-system/backend/internal/rbac"
)

func TestSyncRolePoliciesFiltersAPIAndKeepsABACScopes(t *testing.T) {
	h, enforcer := newRBACHandlerForTest(t)

	role := models.Role{Name: "ops-engineer"}
	if err := h.DB.Create(&role).Error; err != nil {
		t.Fatalf("create role failed: %v", err)
	}
	apiPermission := models.Permission{
		Name:             "task-read",
		Type:             "api",
		Resource:         "/api/v1/tasks",
		Action:           "GET",
		DeptScope:        "dept-a",
		ResourceTagScope: "tag-prod",
		EnvScope:         "prod",
		Key:              "api.tasks.read",
	}
	menuPermission := models.Permission{
		Name:     "task-menu",
		Type:     "menu",
		Key:      "menu.tasks",
		Resource: "menu.tasks",
		Action:   "view",
	}
	if err := h.DB.Create(&apiPermission).Error; err != nil {
		t.Fatalf("create api permission failed: %v", err)
	}
	if err := h.DB.Create(&menuPermission).Error; err != nil {
		t.Fatalf("create menu permission failed: %v", err)
	}
	if err := h.DB.Create(&models.RolePermission{RoleID: role.ID, PermissionID: apiPermission.ID}).Error; err != nil {
		t.Fatalf("bind api permission failed: %v", err)
	}
	if err := h.DB.Create(&models.RolePermission{RoleID: role.ID, PermissionID: menuPermission.ID}).Error; err != nil {
		t.Fatalf("bind menu permission failed: %v", err)
	}

	if err := h.syncRolePolicies(role); err != nil {
		t.Fatalf("sync policies failed: %v", err)
	}

	policies, err := enforcer.GetFilteredPolicy(0, role.Name)
	if err != nil {
		t.Fatalf("get filtered policy failed: %v", err)
	}
	if len(policies) != 1 {
		t.Fatalf("expected 1 api policy, got %d (%v)", len(policies), policies)
	}
	got := policies[0]
	expected := []string{role.Name, apiPermission.Resource, apiPermission.Action, "dept-a", "tag-prod", "prod"}
	for idx := range expected {
		if got[idx] != expected[idx] {
			t.Fatalf("unexpected policy at index %d: got %q want %q", idx, got[idx], expected[idx])
		}
	}
}

func TestBindRolePermissionsSyncsPolicies(t *testing.T) {
	h, enforcer := newRBACHandlerForTest(t)

	role := models.Role{Name: "release-ops"}
	if err := h.DB.Create(&role).Error; err != nil {
		t.Fatalf("create role failed: %v", err)
	}
	apiPermission := models.Permission{
		Name:             "message-read",
		Type:             "api",
		Resource:         "/api/v1/messages",
		Action:           "GET",
		ResourceTagScope: "critical",
		EnvScope:         "prod",
	}
	buttonPermission := models.Permission{
		Name:     "task-exec-button",
		Type:     "button",
		Key:      "button.tasks.execute",
		Resource: "button.tasks.execute",
		Action:   "click",
	}
	if err := h.DB.Create(&apiPermission).Error; err != nil {
		t.Fatalf("create api permission failed: %v", err)
	}
	if err := h.DB.Create(&buttonPermission).Error; err != nil {
		t.Fatalf("create button permission failed: %v", err)
	}

	router := gin.New()
	router.POST("/roles/:id/permissions", h.BindRolePermissions)

	rec := performBindRolePermissions(t, router, role.ID, []uint{apiPermission.ID, buttonPermission.ID})
	if rec.Code != http.StatusOK {
		t.Fatalf("bind role permissions status=%d body=%s", rec.Code, rec.Body.String())
	}

	policies, err := enforcer.GetFilteredPolicy(0, role.Name)
	if err != nil {
		t.Fatalf("get filtered policy failed: %v", err)
	}
	if len(policies) != 1 {
		t.Fatalf("expected 1 synced api policy, got %d (%v)", len(policies), policies)
	}
	if policies[0][1] != "/api/v1/messages" || policies[0][2] != "GET" {
		t.Fatalf("unexpected synced policy: %v", policies[0])
	}
	if policies[0][4] != "critical" || policies[0][5] != "prod" {
		t.Fatalf("abac scopes not synced: %v", policies[0])
	}

	rec = performBindRolePermissions(t, router, role.ID, []uint{})
	if rec.Code != http.StatusOK {
		t.Fatalf("clear role permissions status=%d body=%s", rec.Code, rec.Body.String())
	}
	left, err := enforcer.GetFilteredPolicy(0, role.Name)
	if err != nil {
		t.Fatalf("get filtered policy after clear failed: %v", err)
	}
	if len(left) != 0 {
		t.Fatalf("expected no role policies after clear, got %v", left)
	}
}

func newRBACHandlerForTest(t *testing.T) (*Handler, *casbin.Enforcer) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	database, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := database.AutoMigrate(models.AutoMigrateModels()...); err != nil {
		t.Fatalf("auto migrate models failed: %v", err)
	}

	modelPath := filepath.Join("..", "..", "config", "casbin_model.conf")
	enforcer, err := rbac.InitEnforcer(database, modelPath)
	if err != nil {
		t.Fatalf("init enforcer failed: %v", err)
	}
	return &Handler{DB: database, Enforcer: enforcer}, enforcer
}

func performBindRolePermissions(t *testing.T, router *gin.Engine, roleID uint, permissionIDs []uint) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"permissionIds": permissionIDs})
	if err != nil {
		t.Fatalf("marshal request failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/roles/"+toStringID(roleID)+"/permissions", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}
