package handler

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"devops-system/backend/internal/models"
)

func normalizePermission(permission *models.Permission) {
	permission.Type = strings.TrimSpace(permission.Type)
	if permission.Type == "" {
		permission.Type = "api"
	}

	permission.DeptScope = normalizeScope(permission.DeptScope)
	permission.ResourceTagScope = normalizeScope(permission.ResourceTagScope)
	permission.EnvScope = normalizeScope(permission.EnvScope)

	permission.Resource = strings.TrimSpace(permission.Resource)
	permission.Action = strings.TrimSpace(permission.Action)
	permission.Key = strings.TrimSpace(permission.Key)
}

func normalizeScope(scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return "*"
	}
	return scope
}

func (h *Handler) syncRolePoliciesByID(roleID uint) error {
	var role models.Role
	if err := h.DB.First(&role, roleID).Error; err != nil {
		return err
	}
	return h.syncRolePolicies(role)
}

func (h *Handler) syncRolePolicies(role models.Role) error {
	if role.Name == "admin" || role.BuiltIn {
		return nil
	}

	if _, err := h.Enforcer.RemoveFilteredPolicy(0, role.Name); err != nil {
		return err
	}

	var permissions []models.Permission
	if err := h.DB.Table("permissions").
		Select("permissions.*").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ? AND permissions.type = ?", role.ID, "api").
		Find(&permissions).Error; err != nil {
		return err
	}

	for _, permission := range permissions {
		permission.DeptScope = normalizeScope(permission.DeptScope)
		permission.ResourceTagScope = normalizeScope(permission.ResourceTagScope)
		permission.EnvScope = normalizeScope(permission.EnvScope)
		if _, err := h.Enforcer.AddPolicy(
			role.Name,
			permission.Resource,
			permission.Action,
			permission.DeptScope,
			permission.ResourceTagScope,
			permission.EnvScope,
		); err != nil {
			return err
		}
	}

	return h.Enforcer.SavePolicy()
}

func (h *Handler) syncPoliciesByPermissionID(permissionID uint) error {
	var roleIDs []uint
	if err := h.DB.Table("role_permissions").
		Where("permission_id = ?", permissionID).
		Pluck("role_id", &roleIDs).Error; err != nil {
		return err
	}

	for _, roleID := range roleIDs {
		if err := h.syncRolePoliciesByID(roleID); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) findRoleOrNotFound(roleID uint) (models.Role, error) {
	var role models.Role
	if err := h.DB.First(&role, roleID).Error; err != nil {
		return role, err
	}
	return role, nil
}

func (h *Handler) findUserOrNotFound(userID uint) (models.User, error) {
	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		return user, err
	}
	return user, nil
}

func (h *Handler) findDepartmentOrNotFound(departmentID uint) (models.Department, error) {
	var department models.Department
	if err := h.DB.First(&department, departmentID).Error; err != nil {
		return department, err
	}
	return department, nil
}

func isRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
