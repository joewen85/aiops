package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"devops-system/backend/internal/auth"
	"devops-system/backend/internal/models"
)

func TestPermissionRequiredABACScopes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enforcer := newABACEnforcerForTest(t)

	if _, err := enforcer.AddPolicy("ops", "/api/v1/tasks", "GET", "dept-a", "tag-prod", "prod"); err != nil {
		t.Fatalf("add policy failed: %v", err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(ClaimsContextKey, &auth.Claims{
			Username: "alice",
			Roles:    []string{"ops"},
			DeptID:   "dept-a",
		})
		c.Next()
	})
	router.GET("/api/v1/tasks", PermissionRequired(enforcer, nil, ""), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	allowReq := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	allowReq.Header.Set("X-Resource-Tag", "tag-prod")
	allowReq.Header.Set("X-Env", "prod")
	allowRec := httptest.NewRecorder()
	router.ServeHTTP(allowRec, allowReq)
	if allowRec.Code != http.StatusOK {
		t.Fatalf("expected allow status 200, got %d body=%s", allowRec.Code, allowRec.Body.String())
	}

	denyReq := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	denyReq.Header.Set("X-Resource-Tag", "tag-prod")
	denyReq.Header.Set("X-Env", "staging")
	denyRec := httptest.NewRecorder()
	router.ServeHTTP(denyRec, denyReq)
	if denyRec.Code != http.StatusForbidden {
		t.Fatalf("expected deny status 403, got %d body=%s", denyRec.Code, denyRec.Body.String())
	}
}

func TestPermissionRequiredUsernameFallbackAndUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enforcer := newABACEnforcerForTest(t)

	if _, err := enforcer.AddPolicy("alice", "/api/v1/messages", "GET", "*", "*", "*"); err != nil {
		t.Fatalf("add user policy failed: %v", err)
	}

	router := gin.New()
	router.GET("/api/v1/messages", func(c *gin.Context) {
		claims := &auth.Claims{Username: c.GetHeader("X-User"), Roles: []string{"guest"}}
		c.Set(ClaimsContextKey, claims)
		c.Next()
	}, PermissionRequired(enforcer, nil, ""), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	router.GET("/api/v1/no-claims", PermissionRequired(enforcer, nil, ""), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	allowReq := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	allowReq.Header.Set("X-User", "alice")
	allowRec := httptest.NewRecorder()
	router.ServeHTTP(allowRec, allowReq)
	if allowRec.Code != http.StatusOK {
		t.Fatalf("expected username fallback allow 200, got %d body=%s", allowRec.Code, allowRec.Body.String())
	}

	denyReq := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	denyReq.Header.Set("X-User", "bob")
	denyRec := httptest.NewRecorder()
	router.ServeHTTP(denyRec, denyReq)
	if denyRec.Code != http.StatusForbidden {
		t.Fatalf("expected bob forbidden 403, got %d body=%s", denyRec.Code, denyRec.Body.String())
	}

	unauthReq := httptest.NewRequest(http.MethodGet, "/api/v1/no-claims", nil)
	unauthRec := httptest.NewRecorder()
	router.ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized 401, got %d body=%s", unauthRec.Code, unauthRec.Body.String())
	}
}

func TestPermissionRequiredRuntimeRoleRevocation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enforcer := newABACEnforcerForTest(t)
	database := newPermissionTestDB(t)

	user := models.User{Username: "runtime-user", PasswordHash: "hash", IsActive: true}
	if err := database.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	role := models.Role{Name: "runtime-role"}
	if err := database.Create(&role).Error; err != nil {
		t.Fatalf("create role failed: %v", err)
	}
	if err := database.Create(&models.UserRole{UserID: user.ID, RoleID: role.ID}).Error; err != nil {
		t.Fatalf("bind user role failed: %v", err)
	}
	if _, err := enforcer.AddPolicy(role.Name, "/api/v1/runtime", "GET", "*", "*", "*"); err != nil {
		t.Fatalf("add role policy failed: %v", err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(ClaimsContextKey, &auth.Claims{
			UserID:   user.ID,
			Username: user.Username,
			Roles:    []string{"stale-role"},
		})
		c.Next()
	})
	router.GET("/api/v1/runtime", PermissionRequired(enforcer, database, ""), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	allowReq := httptest.NewRequest(http.MethodGet, "/api/v1/runtime", nil)
	allowRec := httptest.NewRecorder()
	router.ServeHTTP(allowRec, allowReq)
	if allowRec.Code != http.StatusOK {
		t.Fatalf("expected runtime role allow 200, got %d body=%s", allowRec.Code, allowRec.Body.String())
	}

	if err := database.Where("user_id = ?", user.ID).Delete(&models.UserRole{}).Error; err != nil {
		t.Fatalf("revoke role failed: %v", err)
	}

	denyReq := httptest.NewRequest(http.MethodGet, "/api/v1/runtime", nil)
	denyRec := httptest.NewRecorder()
	router.ServeHTTP(denyRec, denyReq)
	if denyRec.Code != http.StatusForbidden {
		t.Fatalf("expected revoked role forbidden 403, got %d body=%s", denyRec.Code, denyRec.Body.String())
	}
}

func TestPermissionRequiredRuntimeRoleRevocationWithCacheTTL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enforcer := newABACEnforcerForTest(t)
	database := newPermissionTestDB(t)

	user := models.User{Username: "runtime-cache-user", PasswordHash: "hash", IsActive: true}
	if err := database.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	role := models.Role{Name: "runtime-cache-role"}
	if err := database.Create(&role).Error; err != nil {
		t.Fatalf("create role failed: %v", err)
	}
	if err := database.Create(&models.UserRole{UserID: user.ID, RoleID: role.ID}).Error; err != nil {
		t.Fatalf("bind user role failed: %v", err)
	}
	if _, err := enforcer.AddPolicy(role.Name, "/api/v1/runtime-cache", "GET", "*", "*", "*"); err != nil {
		t.Fatalf("add role policy failed: %v", err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(ClaimsContextKey, &auth.Claims{
			UserID:   user.ID,
			Username: user.Username,
			Roles:    []string{"stale-role"},
		})
		c.Next()
	})
	router.GET("/api/v1/runtime-cache", PermissionRequiredWithRuntimeCache(enforcer, database, "", 120*time.Millisecond), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	firstReq := httptest.NewRequest(http.MethodGet, "/api/v1/runtime-cache", nil)
	firstRec := httptest.NewRecorder()
	router.ServeHTTP(firstRec, firstReq)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("expected first allow 200, got %d body=%s", firstRec.Code, firstRec.Body.String())
	}

	if err := database.Where("user_id = ?", user.ID).Delete(&models.UserRole{}).Error; err != nil {
		t.Fatalf("revoke role failed: %v", err)
	}

	withinTTLReq := httptest.NewRequest(http.MethodGet, "/api/v1/runtime-cache", nil)
	withinTTLRec := httptest.NewRecorder()
	router.ServeHTTP(withinTTLRec, withinTTLReq)
	if withinTTLRec.Code != http.StatusOK {
		t.Fatalf("expected cached allow 200 within ttl, got %d body=%s", withinTTLRec.Code, withinTTLRec.Body.String())
	}

	time.Sleep(150 * time.Millisecond)

	afterTTLReq := httptest.NewRequest(http.MethodGet, "/api/v1/runtime-cache", nil)
	afterTTLRec := httptest.NewRecorder()
	router.ServeHTTP(afterTTLRec, afterTTLReq)
	if afterTTLRec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden 403 after ttl expiration, got %d body=%s", afterTTLRec.Code, afterTTLRec.Body.String())
	}
}

func TestPermissionRequiredInactiveUserUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enforcer := newABACEnforcerForTest(t)
	database := newPermissionTestDB(t)

	user := models.User{Username: "inactive-user", PasswordHash: "hash", IsActive: true}
	if err := database.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	role := models.Role{Name: "inactive-role"}
	if err := database.Create(&role).Error; err != nil {
		t.Fatalf("create role failed: %v", err)
	}
	if err := database.Create(&models.UserRole{UserID: user.ID, RoleID: role.ID}).Error; err != nil {
		t.Fatalf("bind user role failed: %v", err)
	}
	if _, err := enforcer.AddPolicy(role.Name, "/api/v1/inactive", "GET", "*", "*", "*"); err != nil {
		t.Fatalf("add role policy failed: %v", err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(ClaimsContextKey, &auth.Claims{
			UserID:   user.ID,
			Username: user.Username,
			Roles:    []string{role.Name},
		})
		c.Next()
	})
	router.GET("/api/v1/inactive", PermissionRequired(enforcer, database, ""), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	if err := database.Model(&models.User{}).Where("id = ?", user.ID).Update("is_active", false).Error; err != nil {
		t.Fatalf("deactivate user failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inactive", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected inactive user unauthorized 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPermissionRequiredRejectsInvalidABACHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enforcer := newABACEnforcerForTest(t)
	if _, err := enforcer.AddPolicy("ops", "/api/v1/headers", "GET", "*", "*", "*"); err != nil {
		t.Fatalf("add policy failed: %v", err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(ClaimsContextKey, &auth.Claims{Username: "alice", Roles: []string{"ops"}})
		c.Next()
	})
	router.GET("/api/v1/headers", PermissionRequired(enforcer, nil, ""), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/headers", nil)
	req.Header.Set("X-Env", "prod/blue")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid ABAC header 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPermissionRequiredABACSignatureValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enforcer := newABACEnforcerForTest(t)
	secret := "sign-secret-for-test"
	if _, err := enforcer.AddPolicy("ops", "/api/v1/signed", "GET", "*", "tag-prod", "prod"); err != nil {
		t.Fatalf("add policy failed: %v", err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(ClaimsContextKey, &auth.Claims{Username: "alice", Roles: []string{"ops"}})
		c.Next()
	})
	router.GET("/api/v1/signed", PermissionRequired(enforcer, nil, secret), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	denyReq := httptest.NewRequest(http.MethodGet, "/api/v1/signed", nil)
	denyReq.Header.Set("X-Env", "prod")
	denyReq.Header.Set("X-Resource-Tag", "tag-prod")
	denyRec := httptest.NewRecorder()
	router.ServeHTTP(denyRec, denyReq)
	if denyRec.Code != http.StatusForbidden {
		t.Fatalf("expected unsigned request forbidden 403, got %d body=%s", denyRec.Code, denyRec.Body.String())
	}

	ts := time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)
	signature := signABACRequest(http.MethodGet, "/api/v1/signed", "prod", "tag-prod", ts, secret)

	allowReq := httptest.NewRequest(http.MethodGet, "/api/v1/signed", nil)
	allowReq.Header.Set("X-Env", "prod")
	allowReq.Header.Set("X-Resource-Tag", "tag-prod")
	allowReq.Header.Set("X-ABAC-Timestamp", ts)
	allowReq.Header.Set("X-ABAC-Signature", strings.ToUpper(signature))
	allowRec := httptest.NewRecorder()
	router.ServeHTTP(allowRec, allowReq)
	if allowRec.Code != http.StatusOK {
		t.Fatalf("expected signed request allow 200, got %d body=%s", allowRec.Code, allowRec.Body.String())
	}
}

func newABACEnforcerForTest(t *testing.T) *casbin.Enforcer {
	t.Helper()
	m, err := model.NewModelFromString(`
[request_definition]
r = sub, obj, act, dept, tag, env
[policy_definition]
p = sub, obj, act, dept, tag, env
[role_definition]
g = _, _
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = (g(r.sub, p.sub) || r.sub == p.sub) && keyMatch2(r.obj, p.obj) && regexMatch(r.act, p.act) && (p.dept == "*" || r.dept == p.dept) && (p.tag == "*" || r.tag == p.tag) && (p.env == "*" || r.env == p.env)
`)
	if err != nil {
		t.Fatalf("load model failed: %v", err)
	}
	enforcer, err := casbin.NewEnforcer(m)
	if err != nil {
		t.Fatalf("new enforcer failed: %v", err)
	}
	return enforcer
}

func newPermissionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	database, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := database.AutoMigrate(models.AutoMigrateModels()...); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}
	return database
}

func signABACRequest(method string, path string, env string, tag string, timestamp string, secret string) string {
	payload := strings.Join([]string{method, path, env, tag, timestamp}, "\n")
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
