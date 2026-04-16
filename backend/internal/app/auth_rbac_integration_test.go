package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"devops-system/backend/internal/auth"
	"devops-system/backend/internal/config"
	"devops-system/backend/internal/handler"
	"devops-system/backend/internal/models"
	"devops-system/backend/internal/rbac"
	"devops-system/backend/internal/service"
)

type apiResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type mePermissionData struct {
	Permissions []models.Permission `json:"permissions"`
	MenuKeys    []string            `json:"menuKeys"`
	ButtonKeys  []string            `json:"buttonKeys"`
	APIKeys     []string            `json:"apiKeys"`
	AllAccess   bool                `json:"allAccess"`
}

type listPayload[T any] struct {
	List     []T   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

func TestLoginAndMePermissionsIntegration(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)

	token := loginAndGetToken(t, router, "admin", "Admin@123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me/permissions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("me permissions status=%d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal me permissions failed: %v", err)
	}
	if resp.Code != 0 || resp.Message != "ok" {
		t.Fatalf("unexpected response: %+v", resp)
	}

	var data struct {
		AllAccess bool `json:"allAccess"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("unmarshal me permissions data failed: %v", err)
	}
	if !data.AllAccess {
		t.Fatalf("admin should have allAccess=true")
	}
}

func TestTasksAPIABACAuthorizationIntegration(t *testing.T) {
	router, database, enforcer := newRouterForIntegrationTest(t)

	dept := models.Department{Name: "SRE"}
	if err := database.Create(&dept).Error; err != nil {
		t.Fatalf("create department failed: %v", err)
	}
	role := models.Role{Name: "ops"}
	if err := database.Create(&role).Error; err != nil {
		t.Fatalf("create role failed: %v", err)
	}
	passwordHash, err := auth.HashPassword("Ops@123")
	if err != nil {
		t.Fatalf("hash password failed: %v", err)
	}
	user := models.User{
		Username:     "ops-user",
		PasswordHash: passwordHash,
		DisplayName:  "Ops User",
		IsActive:     true,
	}
	if err := database.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	if err := database.Create(&models.UserRole{UserID: user.ID, RoleID: role.ID}).Error; err != nil {
		t.Fatalf("bind user role failed: %v", err)
	}
	if err := database.Create(&models.UserDepartment{UserID: user.ID, DepartmentID: dept.ID}).Error; err != nil {
		t.Fatalf("bind user department failed: %v", err)
	}
	if _, err := enforcer.AddPolicy(role.Name, "/api/v1/tasks", "GET", fmt.Sprintf("%d", dept.ID), "tag-prod", "prod"); err != nil {
		t.Fatalf("add casbin policy failed: %v", err)
	}

	token := loginAndGetToken(t, router, "ops-user", "Ops@123")

	allowReq := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	allowReq.Header.Set("Authorization", "Bearer "+token)
	allowReq.Header.Set("X-Resource-Tag", "tag-prod")
	allowReq.Header.Set("X-Env", "prod")
	allowRec := httptest.NewRecorder()
	router.ServeHTTP(allowRec, allowReq)
	if allowRec.Code != http.StatusOK {
		t.Fatalf("allow tasks status=%d body=%s", allowRec.Code, allowRec.Body.String())
	}
	assertListResponseShape(t, allowRec.Body.Bytes())

	denyReq := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	denyReq.Header.Set("Authorization", "Bearer "+token)
	denyReq.Header.Set("X-Resource-Tag", "tag-prod")
	denyReq.Header.Set("X-Env", "staging")
	denyRec := httptest.NewRecorder()
	router.ServeHTTP(denyRec, denyReq)
	if denyRec.Code != http.StatusForbidden {
		t.Fatalf("deny tasks status=%d body=%s", denyRec.Code, denyRec.Body.String())
	}

	var denyResp apiResponse
	if err := json.Unmarshal(denyRec.Body.Bytes(), &denyResp); err != nil {
		t.Fatalf("unmarshal deny response failed: %v", err)
	}
	if denyResp.Code != 2001 || denyResp.Message != "forbidden" {
		t.Fatalf("unexpected deny payload: %+v", denyResp)
	}
}

func TestMePermissionsRealtimeConsistencyAfterRBACChange(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)

	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	roleID := createRoleViaAPI(t, router, adminToken, "rbac-viewer")
	userID := createUserViaAPI(t, router, adminToken, "rbac-user", "Rbac@123")
	bindUserRolesViaAPI(t, router, adminToken, userID, []uint{roleID})

	menuPermissionID := createPermissionViaAPI(t, router, adminToken, map[string]any{
		"name":     "menu tasks",
		"type":     "menu",
		"key":      "menu.tasks",
		"resource": "menu.tasks",
		"action":   "view",
	})
	buttonPermissionID := createPermissionViaAPI(t, router, adminToken, map[string]any{
		"name":     "button execute task",
		"type":     "button",
		"key":      "button.tasks.execute",
		"resource": "button.tasks.execute",
		"action":   "click",
	})

	bindRolePermissionsViaAPI(t, router, adminToken, roleID, []uint{menuPermissionID})

	userToken := loginAndGetToken(t, router, "rbac-user", "Rbac@123")

	first := getMePermissionsViaAPI(t, router, userToken)
	if first.AllAccess {
		t.Fatalf("rbac-user should not have allAccess")
	}
	if !containsString(first.MenuKeys, "menu.tasks") {
		t.Fatalf("expected initial menu key menu.tasks, got %+v", first.MenuKeys)
	}
	if containsString(first.ButtonKeys, "button.tasks.execute") {
		t.Fatalf("unexpected initial button key in %+v", first.ButtonKeys)
	}

	bindRolePermissionsViaAPI(t, router, adminToken, roleID, []uint{menuPermissionID, buttonPermissionID})

	second := getMePermissionsViaAPI(t, router, userToken)
	if !containsString(second.MenuKeys, "menu.tasks") {
		t.Fatalf("menu key should keep exists after bind update, got %+v", second.MenuKeys)
	}
	if !containsString(second.ButtonKeys, "button.tasks.execute") {
		t.Fatalf("button key should be immediately visible, got %+v", second.ButtonKeys)
	}

	bindRolePermissionsViaAPI(t, router, adminToken, roleID, []uint{buttonPermissionID})

	third := getMePermissionsViaAPI(t, router, userToken)
	if containsString(third.MenuKeys, "menu.tasks") {
		t.Fatalf("menu key should be removed immediately, got %+v", third.MenuKeys)
	}
	if !containsString(third.ButtonKeys, "button.tasks.execute") {
		t.Fatalf("button key should still exist, got %+v", third.ButtonKeys)
	}
}

func TestMePermissionsCompactBundleIntegration(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	roleID := createRoleViaAPI(t, router, adminToken, "compact-viewer")
	userID := createUserViaAPI(t, router, adminToken, "compact-user", "Compact@123")
	bindUserRolesViaAPI(t, router, adminToken, userID, []uint{roleID})

	menuPermissionID := createPermissionViaAPI(t, router, adminToken, map[string]any{
		"name":     "menu compact",
		"type":     "menu",
		"key":      "menu.compact",
		"resource": "menu.compact",
		"action":   "view",
	})
	buttonPermissionID := createPermissionViaAPI(t, router, adminToken, map[string]any{
		"name":     "button compact",
		"type":     "button",
		"key":      "button.compact.execute",
		"resource": "button.compact.execute",
		"action":   "click",
	})
	apiPermissionID := createPermissionViaAPI(t, router, adminToken, map[string]any{
		"name":     "api compact",
		"type":     "api",
		"key":      "api.compact.read",
		"resource": "/api/v1/messages",
		"action":   "GET",
	})
	bindRolePermissionsViaAPI(t, router, adminToken, roleID, []uint{menuPermissionID, buttonPermissionID, apiPermissionID})

	userToken := loginAndGetToken(t, router, "compact-user", "Compact@123")

	compactRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/auth/me/permissions?compact=1", userToken, nil)
	compactResp := assertOKResponse(t, compactRec)
	var compactData mePermissionData
	if err := json.Unmarshal(compactResp.Data, &compactData); err != nil {
		t.Fatalf("unmarshal compact data failed: %v", err)
	}
	if len(compactData.Permissions) != 0 {
		t.Fatalf("compact permissions should be empty, got %d", len(compactData.Permissions))
	}
	if !containsString(compactData.MenuKeys, "menu.compact") {
		t.Fatalf("compact menu keys missing expected key: %+v", compactData.MenuKeys)
	}
	if !containsString(compactData.ButtonKeys, "button.compact.execute") {
		t.Fatalf("compact button keys missing expected key: %+v", compactData.ButtonKeys)
	}
	if !containsString(compactData.APIKeys, "api.compact.read") {
		t.Fatalf("compact api keys missing expected key: %+v", compactData.APIKeys)
	}

	fullData := getMePermissionsViaAPI(t, router, userToken)
	if len(fullData.Permissions) == 0 {
		t.Fatalf("non-compact permissions should include full entries")
	}
}

func TestTasksAPIRuntimeRevocationWithoutReloginIntegration(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	roleID := createRoleViaAPI(t, router, adminToken, "runtime-revoke-role")
	userID := createUserViaAPI(t, router, adminToken, "runtime-revoke-user", "Runtime@123")
	bindUserRolesViaAPI(t, router, adminToken, userID, []uint{roleID})

	permissionID := createPermissionViaAPI(t, router, adminToken, map[string]any{
		"name":             "runtime revoke task",
		"type":             "api",
		"resource":         "/api/v1/tasks",
		"action":           "GET",
		"deptScope":        "*",
		"resourceTagScope": "*",
		"envScope":         "*",
	})
	bindRolePermissionsViaAPI(t, router, adminToken, roleID, []uint{permissionID})

	userToken := loginAndGetToken(t, router, "runtime-revoke-user", "Runtime@123")

	allowRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/tasks", userToken, nil)
	assertOKResponse(t, allowRec)

	bindUserRolesViaAPI(t, router, adminToken, userID, []uint{})

	denyRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/tasks", userToken, nil)
	assertErrorResponse(t, denyRec, http.StatusForbidden, 2001, "forbidden")
}

func TestMePermissionsReflectsUserRoleRevocationWithoutRelogin(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	roleID := createRoleViaAPI(t, router, adminToken, "me-revoke-role")
	userID := createUserViaAPI(t, router, adminToken, "me-revoke-user", "MeRevoke@123")
	bindUserRolesViaAPI(t, router, adminToken, userID, []uint{roleID})

	menuPermissionID := createPermissionViaAPI(t, router, adminToken, map[string]any{
		"name":     "menu me revoke",
		"type":     "menu",
		"key":      "menu.me.revoke",
		"resource": "menu.me.revoke",
		"action":   "view",
	})
	bindRolePermissionsViaAPI(t, router, adminToken, roleID, []uint{menuPermissionID})

	userToken := loginAndGetToken(t, router, "me-revoke-user", "MeRevoke@123")

	first := getMePermissionsViaAPI(t, router, userToken)
	if !containsString(first.MenuKeys, "menu.me.revoke") {
		t.Fatalf("expected menu key exists before revoke, got %+v", first.MenuKeys)
	}

	bindUserRolesViaAPI(t, router, adminToken, userID, []uint{})

	second := getMePermissionsViaAPI(t, router, userToken)
	if containsString(second.MenuKeys, "menu.me.revoke") {
		t.Fatalf("menu key should be removed after role revoke, got %+v", second.MenuKeys)
	}
	if len(second.Permissions) != 0 {
		t.Fatalf("permission entries should be empty after role revoke, got %d", len(second.Permissions))
	}
}

func TestMePermissionsReturnsUnauthorizedForInactiveUser(t *testing.T) {
	router, database, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	userID := createUserViaAPI(t, router, adminToken, "me-inactive-user", "Inactive@123")
	userToken := loginAndGetToken(t, router, "me-inactive-user", "Inactive@123")

	if err := database.Model(&models.User{}).Where("id = ?", userID).Update("is_active", false).Error; err != nil {
		t.Fatalf("deactivate user failed: %v", err)
	}

	rec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/auth/me/permissions", userToken, nil)
	assertErrorResponse(t, rec, http.StatusUnauthorized, 1001, "unauthorized")
}

func TestBindUserRolesValidationIntegration(t *testing.T) {
	router, database, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	userID := createUserViaAPI(t, router, adminToken, "bind-role-user", "BindRole@123")
	roleID := createRoleViaAPI(t, router, adminToken, "bind-role-target")

	invalidZero := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/users/%d/roles", userID), adminToken, map[string]any{
		"roleIds": []uint{0},
	})
	assertErrorResponse(t, invalidZero, http.StatusBadRequest, 3001, "roleIds contains invalid id")

	invalidNotFound := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/users/%d/roles", userID), adminToken, map[string]any{
		"roleIds": []uint{999999},
	})
	assertErrorResponse(t, invalidNotFound, http.StatusBadRequest, 3001, "roleIds contains invalid id")

	oversizeRoleIDs := make([]uint, 201)
	for idx := range oversizeRoleIDs {
		oversizeRoleIDs[idx] = roleID
	}
	oversizeReq := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/users/%d/roles", userID), adminToken, map[string]any{
		"roleIds": oversizeRoleIDs,
	})
	assertErrorResponse(t, oversizeReq, http.StatusBadRequest, 3001, "roleIds exceeds maximum size 200")

	notFoundUser := sendJSONRequest(t, router, http.MethodPost, "/api/v1/users/999999/roles", adminToken, map[string]any{
		"roleIds": []uint{roleID},
	})
	assertErrorResponse(t, notFoundUser, http.StatusNotFound, 4004, "not found")

	okRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/users/%d/roles", userID), adminToken, map[string]any{
		"roleIds": []uint{roleID, roleID},
	})
	_ = assertOKResponse(t, okRec)

	var userRoles []models.UserRole
	if err := database.Where("user_id = ?", userID).Find(&userRoles).Error; err != nil {
		t.Fatalf("query user roles failed: %v", err)
	}
	if len(userRoles) != 1 {
		t.Fatalf("expected deduped user_roles len=1, got %d", len(userRoles))
	}
}

func TestBindUserDepartmentsValidationIntegration(t *testing.T) {
	router, database, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	userID := createUserViaAPI(t, router, adminToken, "bind-dept-user", "BindDept@123")
	departmentID := createDepartmentViaAPI(t, router, adminToken, "bind-dept-target")

	invalidZero := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/users/%d/departments", userID), adminToken, map[string]any{
		"departmentIds": []uint{0},
	})
	assertErrorResponse(t, invalidZero, http.StatusBadRequest, 3001, "departmentIds contains invalid id")

	invalidNotFound := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/users/%d/departments", userID), adminToken, map[string]any{
		"departmentIds": []uint{999999},
	})
	assertErrorResponse(t, invalidNotFound, http.StatusBadRequest, 3001, "departmentIds contains invalid id")

	oversizeDepartmentIDs := make([]uint, 201)
	for idx := range oversizeDepartmentIDs {
		oversizeDepartmentIDs[idx] = departmentID
	}
	oversizeReq := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/users/%d/departments", userID), adminToken, map[string]any{
		"departmentIds": oversizeDepartmentIDs,
	})
	assertErrorResponse(t, oversizeReq, http.StatusBadRequest, 3001, "departmentIds exceeds maximum size 200")

	notFoundUser := sendJSONRequest(t, router, http.MethodPost, "/api/v1/users/999999/departments", adminToken, map[string]any{
		"departmentIds": []uint{departmentID},
	})
	assertErrorResponse(t, notFoundUser, http.StatusNotFound, 4004, "not found")

	okRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/users/%d/departments", userID), adminToken, map[string]any{
		"departmentIds": []uint{departmentID, departmentID},
	})
	_ = assertOKResponse(t, okRec)

	var userDepartments []models.UserDepartment
	if err := database.Where("user_id = ?", userID).Find(&userDepartments).Error; err != nil {
		t.Fatalf("query user departments failed: %v", err)
	}
	if len(userDepartments) != 1 {
		t.Fatalf("expected deduped user_departments len=1, got %d", len(userDepartments))
	}
}

func TestGetUserBindingDetailsIntegration(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	userID := createUserViaAPI(t, router, adminToken, "binding-detail-user", "BindDetail@123")
	roleID := createRoleViaAPI(t, router, adminToken, "binding-detail-role")
	departmentID := createDepartmentViaAPI(t, router, adminToken, "binding-detail-dept")

	bindUserRolesViaAPI(t, router, adminToken, userID, []uint{roleID})
	recBindDepartment := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/users/%d/departments", userID), adminToken, map[string]any{
		"departmentIds": []uint{departmentID},
	})
	_ = assertOKResponse(t, recBindDepartment)

	recRoles := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/users/%d/roles", userID), adminToken, nil)
	respRoles := assertOKResponse(t, recRoles)
	var roleData struct {
		User    models.User   `json:"user"`
		RoleIDs []uint        `json:"roleIds"`
		Roles   []models.Role `json:"roles"`
	}
	if err := json.Unmarshal(respRoles.Data, &roleData); err != nil {
		t.Fatalf("unmarshal user roles data failed: %v", err)
	}
	if roleData.User.ID != userID {
		t.Fatalf("unexpected role user id=%d", roleData.User.ID)
	}
	if !containsUint(roleData.RoleIDs, roleID) {
		t.Fatalf("expected role id=%d in %+v", roleID, roleData.RoleIDs)
	}

	recDepartments := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/users/%d/departments", userID), adminToken, nil)
	respDepartments := assertOKResponse(t, recDepartments)
	var departmentData struct {
		User          models.User         `json:"user"`
		DepartmentIDs []uint              `json:"departmentIds"`
		Departments   []models.Department `json:"departments"`
	}
	if err := json.Unmarshal(respDepartments.Data, &departmentData); err != nil {
		t.Fatalf("unmarshal user departments data failed: %v", err)
	}
	if departmentData.User.ID != userID {
		t.Fatalf("unexpected department user id=%d", departmentData.User.ID)
	}
	if !containsUint(departmentData.DepartmentIDs, departmentID) {
		t.Fatalf("expected department id=%d in %+v", departmentID, departmentData.DepartmentIDs)
	}
}

func TestBindDepartmentUsersValidationIntegration(t *testing.T) {
	router, database, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	departmentID := createDepartmentViaAPI(t, router, adminToken, "dept-bind-users")
	userID1 := createUserViaAPI(t, router, adminToken, "dept-bind-user-1", "DeptBind@123")
	userID2 := createUserViaAPI(t, router, adminToken, "dept-bind-user-2", "DeptBind@123")

	invalidZero := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/departments/%d/users", departmentID), adminToken, map[string]any{
		"userIds": []uint{0},
	})
	assertErrorResponse(t, invalidZero, http.StatusBadRequest, 3001, "userIds contains invalid id")

	invalidNotFound := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/departments/%d/users", departmentID), adminToken, map[string]any{
		"userIds": []uint{999999},
	})
	assertErrorResponse(t, invalidNotFound, http.StatusBadRequest, 3001, "userIds contains invalid id")

	oversizeUserIDs := make([]uint, 201)
	for idx := range oversizeUserIDs {
		oversizeUserIDs[idx] = userID1
	}
	oversizeReq := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/departments/%d/users", departmentID), adminToken, map[string]any{
		"userIds": oversizeUserIDs,
	})
	assertErrorResponse(t, oversizeReq, http.StatusBadRequest, 3001, "userIds exceeds maximum size 200")

	notFoundDepartment := sendJSONRequest(t, router, http.MethodPost, "/api/v1/departments/999999/users", adminToken, map[string]any{
		"userIds": []uint{userID1},
	})
	assertErrorResponse(t, notFoundDepartment, http.StatusNotFound, 4004, "not found")

	okRec := sendJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/departments/%d/users", departmentID), adminToken, map[string]any{
		"userIds": []uint{userID1, userID1, userID2},
	})
	_ = assertOKResponse(t, okRec)

	var userDepartments []models.UserDepartment
	if err := database.Where("department_id = ?", departmentID).Find(&userDepartments).Error; err != nil {
		t.Fatalf("query department members failed: %v", err)
	}
	if len(userDepartments) != 2 {
		t.Fatalf("expected deduped department members len=2, got %d", len(userDepartments))
	}

	getRec := sendJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/departments/%d/users", departmentID), adminToken, nil)
	getResp := assertOKResponse(t, getRec)
	var data struct {
		Department models.Department `json:"department"`
		UserIDs    []uint            `json:"userIds"`
		Users      []models.User     `json:"users"`
	}
	if err := json.Unmarshal(getResp.Data, &data); err != nil {
		t.Fatalf("unmarshal department users data failed: %v", err)
	}
	if data.Department.ID != departmentID {
		t.Fatalf("unexpected department id=%d", data.Department.ID)
	}
	if !containsUint(data.UserIDs, userID1) || !containsUint(data.UserIDs, userID2) {
		t.Fatalf("expected bound user ids in %+v", data.UserIDs)
	}

	getNotFoundRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/departments/999999/users", adminToken, nil)
	assertErrorResponse(t, getNotFoundRec, http.StatusNotFound, 4004, "not found")
}

func TestListDepartmentTreeIntegration(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	parentRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/departments", adminToken, map[string]any{
		"name": "dept-tree-parent",
	})
	parentResp := assertOKResponse(t, parentRec)
	parentID := parseIDFromData(t, parentResp.Data)

	childRec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/departments", adminToken, map[string]any{
		"name":     "dept-tree-child",
		"parentId": parentID,
	})
	childResp := assertOKResponse(t, childRec)
	childID := parseIDFromData(t, childResp.Data)

	treeRec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/departments/tree", adminToken, nil)
	treeResp := assertOKResponse(t, treeRec)
	var tree []struct {
		ID       uint `json:"id"`
		Children []struct {
			ID uint `json:"id"`
		} `json:"children"`
	}
	if err := json.Unmarshal(treeResp.Data, &tree); err != nil {
		t.Fatalf("unmarshal department tree failed: %v", err)
	}

	foundParent := false
	foundChild := false
	for _, root := range tree {
		if root.ID != parentID {
			continue
		}
		foundParent = true
		for _, child := range root.Children {
			if child.ID == childID {
				foundChild = true
				break
			}
		}
	}
	if !foundParent || !foundChild {
		t.Fatalf("expected parent and child in tree, got %+v", tree)
	}
}

func TestBuiltInAdminRoleProtectionIntegration(t *testing.T) {
	router, database, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	var adminRole models.Role
	if err := database.Where("name = ?", "admin").First(&adminRole).Error; err != nil {
		t.Fatalf("query admin role failed: %v", err)
	}

	deleteRec := sendJSONRequest(t, router, http.MethodDelete, fmt.Sprintf("/api/v1/roles/%d", adminRole.ID), adminToken, nil)
	assertErrorResponse(t, deleteRec, http.StatusBadRequest, 4001, "built-in role cannot be deleted")

	updateRec := sendJSONRequest(t, router, http.MethodPut, fmt.Sprintf("/api/v1/roles/%d", adminRole.ID), adminToken, map[string]any{
		"name": "super-admin",
	})
	assertErrorResponse(t, updateRec, http.StatusBadRequest, 4003, "built-in role name cannot be changed")
}

func TestPermissionUpdateRefreshesABACPolicyImmediately(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)

	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")
	roleID := createRoleViaAPI(t, router, adminToken, "abac-updater")
	userID := createUserViaAPI(t, router, adminToken, "abac-user", "Abac@123")
	bindUserRolesViaAPI(t, router, adminToken, userID, []uint{roleID})

	permissionID := createPermissionViaAPI(t, router, adminToken, map[string]any{
		"name":             "tasks read in prod",
		"type":             "api",
		"resource":         "/api/v1/tasks",
		"action":           "GET",
		"deptScope":        "*",
		"resourceTagScope": "*",
		"envScope":         "prod",
	})
	bindRolePermissionsViaAPI(t, router, adminToken, roleID, []uint{permissionID})

	userToken := loginAndGetToken(t, router, "abac-user", "Abac@123")

	allowProdReq := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	allowProdReq.Header.Set("Authorization", "Bearer "+userToken)
	allowProdReq.Header.Set("X-Env", "prod")
	allowProdRec := httptest.NewRecorder()
	router.ServeHTTP(allowProdRec, allowProdReq)
	if allowProdRec.Code != http.StatusOK {
		t.Fatalf("expected prod access ok, status=%d body=%s", allowProdRec.Code, allowProdRec.Body.String())
	}

	denyStaging := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	denyStaging.Header.Set("Authorization", "Bearer "+userToken)
	denyStaging.Header.Set("X-Env", "staging")
	denyStagingRec := httptest.NewRecorder()
	router.ServeHTTP(denyStagingRec, denyStaging)
	assertErrorResponse(t, denyStagingRec, http.StatusForbidden, 2001, "forbidden")

	updatePermissionViaAPI(t, router, adminToken, permissionID, map[string]any{
		"envScope": "*",
	})

	allowStaging := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	allowStaging.Header.Set("Authorization", "Bearer "+userToken)
	allowStaging.Header.Set("X-Env", "staging")
	allowStagingRec := httptest.NewRecorder()
	router.ServeHTTP(allowStagingRec, allowStaging)
	if allowStagingRec.Code != http.StatusOK {
		t.Fatalf("expected staging access ok after update, status=%d body=%s", allowStagingRec.Code, allowStagingRec.Body.String())
	}
	assertListResponseShape(t, allowStagingRec.Body.Bytes())
}

func TestListPermissionsTypeFilterIntegration(t *testing.T) {
	router, _, _ := newRouterForIntegrationTest(t)
	adminToken := loginAndGetToken(t, router, "admin", "Admin@123")

	_ = createPermissionViaAPI(t, router, adminToken, map[string]any{
		"name":     "menu users",
		"type":     "menu",
		"key":      "menu.users",
		"resource": "menu.users",
		"action":   "view",
	})
	_ = createPermissionViaAPI(t, router, adminToken, map[string]any{
		"name":     "button user create",
		"type":     "button",
		"key":      "button.users.create",
		"resource": "button.users.create",
		"action":   "click",
	})

	rec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/permissions?type=menu&page=1&pageSize=20", adminToken, nil)
	resp := assertOKResponse(t, rec)

	var pageData listPayload[models.Permission]
	if err := json.Unmarshal(resp.Data, &pageData); err != nil {
		t.Fatalf("unmarshal list payload failed: %v", err)
	}
	if pageData.Page != 1 || pageData.PageSize != 20 {
		t.Fatalf("unexpected pagination: %+v", pageData)
	}
	if len(pageData.List) == 0 {
		t.Fatalf("expected menu permission list not empty")
	}
	for _, item := range pageData.List {
		if item.Type != "menu" {
			t.Fatalf("expected only menu type, got %+v", item)
		}
	}
}

func newRouterForIntegrationTest(t *testing.T) (*gin.Engine, *gorm.DB, *casbin.Enforcer) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	database, err := gorm.Open(sqlite.Open(memoryDSN(t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := database.AutoMigrate(models.AutoMigrateModels()...); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}
	if err := service.SeedDefaultAdmin(database); err != nil {
		t.Fatalf("seed admin failed: %v", err)
	}

	enforcer, err := rbac.InitEnforcer(database, casbinModelPath(t))
	if err != nil {
		t.Fatalf("init enforcer failed: %v", err)
	}
	jwtManager := auth.NewManager("integration-test-secret", 24)
	h := &handler.Handler{
		DB:       database,
		JWT:      jwtManager,
		Enforcer: enforcer,
	}

	return setupRouter(h, jwtManager, enforcer, database, config.Config{}), database, enforcer
}

func loginAndGetToken(t *testing.T, router *gin.Engine, username string, password string) string {
	t.Helper()
	payload, err := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	if err != nil {
		t.Fatalf("marshal login payload failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal login response failed: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("login response code=%d message=%s", resp.Code, resp.Message)
	}

	var data struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("unmarshal login data failed: %v", err)
	}
	if data.Token == "" {
		t.Fatalf("empty login token")
	}
	return data.Token
}

func createRoleViaAPI(t *testing.T, router *gin.Engine, token string, roleName string) uint {
	t.Helper()
	rec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/roles", token, map[string]any{
		"name":        roleName,
		"description": "integration role",
	})
	resp := assertOKResponse(t, rec)
	return parseIDFromData(t, resp.Data)
}

func createUserViaAPI(t *testing.T, router *gin.Engine, token string, username string, password string) uint {
	t.Helper()
	rec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/users", token, map[string]any{
		"username": username,
		"password": password,
		"isActive": true,
	})
	resp := assertOKResponse(t, rec)
	return parseIDFromData(t, resp.Data)
}

func createPermissionViaAPI(t *testing.T, router *gin.Engine, token string, payload map[string]any) uint {
	t.Helper()
	rec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/permissions", token, payload)
	resp := assertOKResponse(t, rec)
	return parseIDFromData(t, resp.Data)
}

func createDepartmentViaAPI(t *testing.T, router *gin.Engine, token string, name string) uint {
	t.Helper()
	rec := sendJSONRequest(t, router, http.MethodPost, "/api/v1/departments", token, map[string]any{
		"name": name,
	})
	resp := assertOKResponse(t, rec)
	return parseIDFromData(t, resp.Data)
}

func updatePermissionViaAPI(t *testing.T, router *gin.Engine, token string, permissionID uint, payload map[string]any) {
	t.Helper()
	path := fmt.Sprintf("/api/v1/permissions/%d", permissionID)
	rec := sendJSONRequest(t, router, http.MethodPut, path, token, payload)
	_ = assertOKResponse(t, rec)
}

func bindUserRolesViaAPI(t *testing.T, router *gin.Engine, token string, userID uint, roleIDs []uint) {
	t.Helper()
	path := fmt.Sprintf("/api/v1/users/%d/roles", userID)
	rec := sendJSONRequest(t, router, http.MethodPost, path, token, map[string]any{
		"roleIds": roleIDs,
	})
	_ = assertOKResponse(t, rec)
}

func bindRolePermissionsViaAPI(t *testing.T, router *gin.Engine, token string, roleID uint, permissionIDs []uint) {
	t.Helper()
	path := fmt.Sprintf("/api/v1/roles/%d/permissions", roleID)
	rec := sendJSONRequest(t, router, http.MethodPost, path, token, map[string]any{
		"permissionIds": permissionIDs,
	})
	_ = assertOKResponse(t, rec)
}

func getMePermissionsViaAPI(t *testing.T, router *gin.Engine, token string) mePermissionData {
	t.Helper()
	rec := sendJSONRequest(t, router, http.MethodGet, "/api/v1/auth/me/permissions", token, nil)
	resp := assertOKResponse(t, rec)

	var data mePermissionData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("unmarshal me permissions data failed: %v", err)
	}
	return data
}

func sendJSONRequest(t *testing.T, router *gin.Engine, method string, path string, token string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	var body *bytes.Reader
	if payload == nil {
		body = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload failed: %v", err)
		}
		body = bytes.NewReader(raw)
	}

	req := httptest.NewRequest(method, path, body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func assertOKResponse(t *testing.T, rec *httptest.ResponseRecorder) apiResponse {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp apiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.Code != 0 || resp.Message != "ok" {
		t.Fatalf("unexpected api response: %+v", resp)
	}
	return resp
}

func assertErrorResponse(t *testing.T, rec *httptest.ResponseRecorder, expectedHTTP int, expectedCode int, expectedMessage string) {
	t.Helper()
	if rec.Code != expectedHTTP {
		t.Fatalf("unexpected status=%d expected=%d body=%s", rec.Code, expectedHTTP, rec.Body.String())
	}
	var resp apiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error response failed: %v", err)
	}
	if resp.Code != expectedCode || resp.Message != expectedMessage {
		t.Fatalf("unexpected error payload=%+v expectedCode=%d expectedMessage=%s", resp, expectedCode, expectedMessage)
	}
}

func parseIDFromData(t *testing.T, data json.RawMessage) uint {
	t.Helper()
	var entity struct {
		ID uint `json:"id"`
	}
	if err := json.Unmarshal(data, &entity); err != nil {
		t.Fatalf("unmarshal id data failed: %v", err)
	}
	if entity.ID == 0 {
		t.Fatalf("unexpected empty id data: %s", string(data))
	}
	return entity.ID
}

func assertListResponseShape(t *testing.T, body []byte) {
	t.Helper()
	var resp struct {
		Code int `json:"code"`
		Data struct {
			List     []any `json:"list"`
			Total    int64 `json:"total"`
			Page     int   `json:"page"`
			PageSize int   `json:"pageSize"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal list response failed: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("unexpected response code: %d", resp.Code)
	}
	if resp.Data.Page <= 0 || resp.Data.PageSize <= 0 {
		t.Fatalf("invalid pagination data: %+v", resp.Data)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsUint(values []uint, target uint) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func memoryDSN(testName string) string {
	return fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(testName, "/", "_"))
}

func casbinModelPath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve test file path failed")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..", "config", "casbin_model.conf")
}
