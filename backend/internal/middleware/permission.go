package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	appAuth "devops-system/backend/internal/auth"
	appErr "devops-system/backend/internal/errors"
	"devops-system/backend/internal/models"
	"devops-system/backend/internal/response"
)

const (
	abacTimestampHeader = "X-ABAC-Timestamp"
	abacSignatureHeader = "X-ABAC-Signature"
)

var (
	abacEnvPattern = regexp.MustCompile(`^[a-zA-Z0-9._:-]{1,32}$`)
	abacTagPattern = regexp.MustCompile(`^[a-zA-Z0-9._:-]{1,64}$`)
)

func PermissionRequired(enforcer *casbin.Enforcer, database *gorm.DB, abacSignSecret string) gin.HandlerFunc {
	return PermissionRequiredWithRuntimeCache(enforcer, database, abacSignSecret, 0)
}

func PermissionRequiredWithRuntimeCache(enforcer *casbin.Enforcer, database *gorm.DB, abacSignSecret string, roleCacheTTL time.Duration) gin.HandlerFunc {
	roleCache := newRoleCache(roleCacheTTL)
	return func(c *gin.Context) {
		claims, ok := GetClaims(c)
		if !ok {
			response.Error(c, http.StatusUnauthorized, appErr.ErrUnauthorized)
			c.Abort()
			return
		}

		runtimeRoles, authorized := resolveRuntimeRoles(c, database, claims, roleCache)
		if !authorized {
			response.Error(c, http.StatusUnauthorized, appErr.ErrUnauthorized)
			c.Abort()
			return
		}

		dept := claims.DeptID
		if dept == "" {
			dept = "*"
		}
		tag, env, valid := parseABACHeaders(c)
		if !valid {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "invalid ABAC headers"))
			c.Abort()
			return
		}
		if !verifyABACSignatureIfNeeded(c, abacSignSecret, env, tag) {
			response.Error(c, http.StatusForbidden, appErr.New(2002, "invalid ABAC signature"))
			c.Abort()
			return
		}

		obj := c.FullPath()
		if obj == "" {
			obj = c.Request.URL.Path
		}
		act := c.Request.Method

		allowed := false
		for _, role := range runtimeRoles {
			okRole, err := enforcer.Enforce(role, obj, act, dept, tag, env)
			if err == nil && okRole {
				allowed = true
				break
			}
		}

		if !allowed {
			okUser, err := enforcer.Enforce(claims.Username, obj, act, dept, tag, env)
			if err == nil && okUser {
				allowed = true
			}
		}

		if !allowed {
			response.Error(c, http.StatusForbidden, appErr.ErrForbidden)
			c.Abort()
			return
		}
		c.Next()
	}
}

type roleCache struct {
	ttl     time.Duration
	mutex   sync.RWMutex
	entries map[uint]roleCacheEntry
}

type roleCacheEntry struct {
	roles     []string
	expiresAt time.Time
}

func newRoleCache(ttl time.Duration) *roleCache {
	if ttl <= 0 {
		return nil
	}
	return &roleCache{
		ttl:     ttl,
		entries: make(map[uint]roleCacheEntry),
	}
}

func (cache *roleCache) get(userID uint) ([]string, bool) {
	if cache == nil || userID == 0 {
		return nil, false
	}
	now := time.Now()
	cache.mutex.RLock()
	entry, exists := cache.entries[userID]
	cache.mutex.RUnlock()
	if !exists {
		return nil, false
	}
	if now.After(entry.expiresAt) {
		cache.mutex.Lock()
		delete(cache.entries, userID)
		cache.mutex.Unlock()
		return nil, false
	}
	return slices.Clone(entry.roles), true
}

func (cache *roleCache) set(userID uint, roles []string) {
	if cache == nil || userID == 0 || len(roles) == 0 {
		return
	}
	cache.mutex.Lock()
	cache.entries[userID] = roleCacheEntry{
		roles:     slices.Clone(roles),
		expiresAt: time.Now().Add(cache.ttl),
	}
	cache.mutex.Unlock()
}

func resolveRuntimeRoles(c *gin.Context, database *gorm.DB, claims *appAuth.Claims, cache *roleCache) ([]string, bool) {
	if database == nil {
		return claims.Roles, true
	}
	var user models.User
	if err := database.Select("id, is_active").Where("id = ?", claims.UserID).First(&user).Error; err != nil {
		return nil, false
	}
	if !user.IsActive {
		return nil, false
	}
	if cachedRoles, ok := cache.get(claims.UserID); ok {
		return cachedRoles, true
	}
	var roles []string
	if err := database.Table("roles").
		Select("roles.name").
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", claims.UserID).
		Scan(&roles).Error; err != nil {
		return nil, false
	}
	cache.set(claims.UserID, roles)
	return roles, true
}

func parseABACHeaders(c *gin.Context) (tag string, env string, valid bool) {
	tag = strings.TrimSpace(c.GetHeader("X-Resource-Tag"))
	env = strings.TrimSpace(c.GetHeader("X-Env"))
	if tag == "" {
		tag = "*"
	}
	if env == "" {
		env = "*"
	}
	if tag != "*" && !abacTagPattern.MatchString(tag) {
		return "", "", false
	}
	if env != "*" && !abacEnvPattern.MatchString(env) {
		return "", "", false
	}
	return tag, env, true
}

func verifyABACSignatureIfNeeded(c *gin.Context, secret string, env string, tag string) bool {
	if strings.TrimSpace(secret) == "" {
		return true
	}
	if env == "*" && tag == "*" {
		return true
	}
	timestamp := strings.TrimSpace(c.GetHeader(abacTimestampHeader))
	signature := strings.TrimSpace(c.GetHeader(abacSignatureHeader))
	if timestamp == "" || signature == "" {
		return false
	}
	ts, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return false
	}
	now := time.Now()
	if ts.After(now.Add(5*time.Minute)) || ts.Before(now.Add(-5*time.Minute)) {
		return false
	}
	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}
	payload := strings.Join([]string{c.Request.Method, path, env, tag, timestamp}, "\n")
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(strings.ToLower(signature)), []byte(strings.ToLower(expected)))
}
