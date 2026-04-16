package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"devops-system/backend/internal/auth"
	appErr "devops-system/backend/internal/errors"
	"devops-system/backend/internal/response"
)

const ClaimsContextKey = "claims"

func AuthRequired(jwtManager auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, http.StatusUnauthorized, appErr.ErrUnauthorized)
			c.Abort()
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if token == "" || token == authHeader {
			response.Error(c, http.StatusUnauthorized, appErr.ErrUnauthorized)
			c.Abort()
			return
		}

		claims, err := jwtManager.Parse(token)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, appErr.ErrUnauthorized)
			c.Abort()
			return
		}

		c.Set(ClaimsContextKey, claims)
		c.Next()
	}
}

func OptionalAuth(jwtManager auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if token == "" || token == authHeader {
			c.Next()
			return
		}
		claims, err := jwtManager.Parse(token)
		if err == nil {
			c.Set(ClaimsContextKey, claims)
		}
		c.Next()
	}
}

func GetClaims(c *gin.Context) (*auth.Claims, bool) {
	raw, exists := c.Get(ClaimsContextKey)
	if !exists {
		return nil, false
	}
	claims, ok := raw.(*auth.Claims)
	return claims, ok
}
