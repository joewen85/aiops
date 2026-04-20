package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	appAuth "devops-system/backend/internal/auth"
	appErr "devops-system/backend/internal/errors"
	"devops-system/backend/internal/middleware"
	"devops-system/backend/internal/models"
	"devops-system/backend/internal/response"
)

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if !bindJSON(c, &req) {
		return
	}

	var user models.User
	if err := h.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		response.Error(c, http.StatusUnauthorized, appErr.ErrUnauthorized)
		return
	}
	if !user.IsActive {
		response.Error(c, http.StatusUnauthorized, appErr.New(1002, "user disabled"))
		return
	}
	if err := comparePassword(user.PasswordHash, req.Password); err != nil {
		response.Error(c, http.StatusUnauthorized, appErr.ErrUnauthorized)
		return
	}

	var roles []string
	if err := h.DB.Table("roles").
		Select("roles.name").
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", user.ID).
		Scan(&roles).Error; err != nil {
		response.Internal(c, err)
		return
	}

	var deptID string
	var dept models.UserDepartment
	query := h.DB.Where("user_id = ?", user.ID).Limit(1).Find(&dept)
	if query.Error != nil {
		response.Internal(c, query.Error)
		return
	}
	if query.RowsAffected > 0 {
		deptID = toStringID(dept.DepartmentID)
	}

	token, err := h.JWT.GenerateToken(user.ID, user.Username, roles, deptID)
	if err != nil {
		response.Internal(c, err)
		return
	}

	response.Success(c, gin.H{
		"token": token,
		"user": gin.H{
			"id":          user.ID,
			"username":    user.Username,
			"displayName": user.DisplayName,
			"email":       user.Email,
			"isActive":    user.IsActive,
			"roles":       roles,
			"deptId":      deptID,
		},
	})
}

func (h *Handler) MePermissions(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, appErr.ErrUnauthorized)
		return
	}
	runtimeRoles, authorized := h.resolveRuntimeRoles(claims)
	if !authorized {
		response.Error(c, http.StatusUnauthorized, appErr.ErrUnauthorized)
		return
	}
	compact := strings.TrimSpace(c.Query("compact")) == "1"

	menuKeys := make([]string, 0)
	buttonKeys := make([]string, 0)
	apiKeys := make([]string, 0)
	menuSet := map[string]struct{}{}
	buttonSet := map[string]struct{}{}
	apiSet := map[string]struct{}{}
	allAccess := false
	for _, role := range runtimeRoles {
		if strings.EqualFold(role, "admin") {
			allAccess = true
			break
		}
	}
	if len(runtimeRoles) == 0 {
		response.Success(c, gin.H{
			"permissions": []models.Permission{},
			"menuKeys":    menuKeys,
			"buttonKeys":  buttonKeys,
			"apiKeys":     apiKeys,
			"allAccess":   allAccess,
		})
		return
	}
	if compact {
		var keyRows []struct {
			Type string `json:"type"`
			Key  string `json:"key"`
		}
		if err := h.DB.Table("permissions").
			Select("DISTINCT permissions.type, permissions.key").
			Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
			Joins("JOIN roles ON roles.id = role_permissions.role_id").
			Where("roles.name IN ?", runtimeRoles).
			Order("permissions.type asc").
			Find(&keyRows).Error; err != nil {
			response.Internal(c, err)
			return
		}
		for _, row := range keyRows {
			key := strings.TrimSpace(row.Key)
			if key == "" {
				continue
			}
			switch row.Type {
			case "menu":
				if _, exists := menuSet[key]; !exists {
					menuSet[key] = struct{}{}
					menuKeys = append(menuKeys, key)
				}
			case "button":
				if _, exists := buttonSet[key]; !exists {
					buttonSet[key] = struct{}{}
					buttonKeys = append(buttonKeys, key)
				}
			default:
				if _, exists := apiSet[key]; !exists {
					apiSet[key] = struct{}{}
					apiKeys = append(apiKeys, key)
				}
			}
		}
		response.Success(c, gin.H{
			"permissions": []models.Permission{},
			"menuKeys":    menuKeys,
			"buttonKeys":  buttonKeys,
			"apiKeys":     apiKeys,
			"allAccess":   allAccess,
		})
		return
	}

	var permissions []models.Permission
	if err := h.DB.Table("permissions").
		Select("DISTINCT permissions.*").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN roles ON roles.id = role_permissions.role_id").
		Where("roles.name IN ?", runtimeRoles).
		Order("permissions.type asc, permissions.id asc").
		Find(&permissions).Error; err != nil {
		response.Internal(c, err)
		return
	}
	for _, permission := range permissions {
		key := strings.TrimSpace(permission.Key)
		if key == "" {
			continue
		}
		switch permission.Type {
		case "menu":
			if _, exists := menuSet[key]; !exists {
				menuSet[key] = struct{}{}
				menuKeys = append(menuKeys, key)
			}
		case "button":
			if _, exists := buttonSet[key]; !exists {
				buttonSet[key] = struct{}{}
				buttonKeys = append(buttonKeys, key)
			}
		default:
			if _, exists := apiSet[key]; !exists {
				apiSet[key] = struct{}{}
				apiKeys = append(apiKeys, key)
			}
		}
	}

	response.Success(c, gin.H{
		"permissions": permissions,
		"menuKeys":    menuKeys,
		"buttonKeys":  buttonKeys,
		"apiKeys":     apiKeys,
		"allAccess":   allAccess,
	})
}

func (h *Handler) resolveRuntimeRoles(claims *appAuth.Claims) ([]string, bool) {
	if claims == nil {
		return nil, false
	}
	if h.DB == nil || claims.UserID == 0 {
		return claims.Roles, true
	}

	var user models.User
	if err := h.DB.Select("id, is_active").Where("id = ?", claims.UserID).First(&user).Error; err != nil {
		return nil, false
	}
	if !user.IsActive {
		return nil, false
	}

	var roles []string
	if err := h.DB.Table("roles").
		Select("roles.name").
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", claims.UserID).
		Scan(&roles).Error; err != nil {
		return nil, false
	}
	return roles, true
}
