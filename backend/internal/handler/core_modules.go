package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	appErr "devops-system/backend/internal/errors"
	"devops-system/backend/internal/models"
	"devops-system/backend/internal/pagination"
	"devops-system/backend/internal/response"
)

const maxBindBatchSize = 200

type departmentTreeNode struct {
	ID       uint                  `json:"id"`
	Name     string                `json:"name"`
	ParentID *uint                 `json:"parentId"`
	Children []*departmentTreeNode `json:"children"`
}

func (h *Handler) ListUsers(c *gin.Context) { listByModel[models.User](c, h.DB) }
func (h *Handler) GetUser(c *gin.Context)   { getByID[models.User](c, h.DB) }

func (h *Handler) CreateUser(c *gin.Context) {
	var req struct {
		Username    string `json:"username" binding:"required"`
		Password    string `json:"password" binding:"required"`
		DisplayName string `json:"displayName"`
		Email       string `json:"email"`
		IsActive    *bool  `json:"isActive"`
	}
	if !bindJSON(c, &req) {
		return
	}
	hash, err := hashPassword(req.Password)
	if err != nil {
		response.Internal(c, err)
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	entity := models.User{
		Username:     req.Username,
		PasswordHash: hash,
		DisplayName:  req.DisplayName,
		Email:        req.Email,
		IsActive:     isActive,
	}
	if err := h.DB.Create(&entity).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, entity)
}

func (h *Handler) UpdateUser(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var updates map[string]interface{}
	if !bindJSON(c, &updates) {
		return
	}
	if raw, exists := updates["password"]; exists {
		if password, okPwd := raw.(string); okPwd && password != "" {
			hash, err := hashPassword(password)
			if err != nil {
				response.Internal(c, err)
				return
			}
			updates["password_hash"] = hash
		}
		delete(updates, "password")
	}
	delete(updates, "id")
	delete(updates, "username")
	if err := h.DB.Model(&models.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		response.Internal(c, err)
		return
	}
	getByID[models.User](c, h.DB)
}

func (h *Handler) DeleteUser(c *gin.Context) { deleteByModel[models.User](c, h.DB) }

func (h *Handler) ToggleUserStatus(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		IsActive bool `json:"isActive"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if err := h.DB.Model(&models.User{}).Where("id = ?", id).Update("is_active", req.IsActive).Error; err != nil {
		response.Internal(c, err)
		return
	}
	getByID[models.User](c, h.DB)
}

func (h *Handler) ResetUserPassword(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if !bindJSON(c, &req) {
		return
	}
	hash, err := hashPassword(req.Password)
	if err != nil {
		response.Internal(c, err)
		return
	}
	if err := h.DB.Model(&models.User{}).Where("id = ?", id).Update("password_hash", hash).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{"id": id, "passwordReset": true})
}

func (h *Handler) BindUserRoles(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if _, err := h.findUserOrNotFound(id); err != nil {
		if isRecordNotFound(err) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	var req struct {
		RoleIDs []uint `json:"roleIds"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if len(req.RoleIDs) > maxBindBatchSize {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "roleIds exceeds maximum size 200"))
		return
	}
	uniqueRoleIDs := make([]uint, 0, len(req.RoleIDs))
	seen := make(map[uint]struct{}, len(req.RoleIDs))
	for _, roleID := range req.RoleIDs {
		if roleID == 0 {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "roleIds contains invalid id"))
			return
		}
		if _, exists := seen[roleID]; exists {
			continue
		}
		seen[roleID] = struct{}{}
		uniqueRoleIDs = append(uniqueRoleIDs, roleID)
	}
	if len(uniqueRoleIDs) > 0 {
		var count int64
		if err := h.DB.Model(&models.Role{}).Where("id IN ?", uniqueRoleIDs).Count(&count).Error; err != nil {
			response.Internal(c, err)
			return
		}
		if count != int64(len(uniqueRoleIDs)) {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "roleIds contains invalid id"))
			return
		}
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", id).Delete(&models.UserRole{}).Error; err != nil {
			return err
		}
		for _, roleID := range uniqueRoleIDs {
			if err := tx.Create(&models.UserRole{UserID: id, RoleID: roleID}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{"id": id, "roleIds": uniqueRoleIDs})
}

func (h *Handler) GetUserRoles(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	user, err := h.findUserOrNotFound(id)
	if err != nil {
		if isRecordNotFound(err) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}

	var roleIDs []uint
	if err := h.DB.Table("user_roles").Where("user_id = ?", id).Pluck("role_id", &roleIDs).Error; err != nil {
		response.Internal(c, err)
		return
	}

	var roles []models.Role
	if err := h.DB.Order("id asc").Find(&roles).Error; err != nil {
		response.Internal(c, err)
		return
	}

	response.Success(c, gin.H{
		"user":    user,
		"roleIds": roleIDs,
		"roles":   roles,
	})
}

func (h *Handler) BindUserDepartments(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if _, err := h.findUserOrNotFound(id); err != nil {
		if isRecordNotFound(err) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	var req struct {
		DepartmentIDs []uint `json:"departmentIds"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if len(req.DepartmentIDs) > maxBindBatchSize {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "departmentIds exceeds maximum size 200"))
		return
	}
	uniqueDepartmentIDs := make([]uint, 0, len(req.DepartmentIDs))
	seen := make(map[uint]struct{}, len(req.DepartmentIDs))
	for _, departmentID := range req.DepartmentIDs {
		if departmentID == 0 {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "departmentIds contains invalid id"))
			return
		}
		if _, exists := seen[departmentID]; exists {
			continue
		}
		seen[departmentID] = struct{}{}
		uniqueDepartmentIDs = append(uniqueDepartmentIDs, departmentID)
	}
	if len(uniqueDepartmentIDs) > 0 {
		var count int64
		if err := h.DB.Model(&models.Department{}).Where("id IN ?", uniqueDepartmentIDs).Count(&count).Error; err != nil {
			response.Internal(c, err)
			return
		}
		if count != int64(len(uniqueDepartmentIDs)) {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "departmentIds contains invalid id"))
			return
		}
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", id).Delete(&models.UserDepartment{}).Error; err != nil {
			return err
		}
		for _, deptID := range uniqueDepartmentIDs {
			if err := tx.Create(&models.UserDepartment{UserID: id, DepartmentID: deptID}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{"id": id, "departmentIds": uniqueDepartmentIDs})
}

func (h *Handler) GetUserDepartments(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	user, err := h.findUserOrNotFound(id)
	if err != nil {
		if isRecordNotFound(err) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}

	var departmentIDs []uint
	if err := h.DB.Table("user_departments").Where("user_id = ?", id).Pluck("department_id", &departmentIDs).Error; err != nil {
		response.Internal(c, err)
		return
	}

	var departments []models.Department
	if err := h.DB.Order("id asc").Find(&departments).Error; err != nil {
		response.Internal(c, err)
		return
	}

	response.Success(c, gin.H{
		"user":          user,
		"departmentIds": departmentIDs,
		"departments":   departments,
	})
}

func (h *Handler) GetDepartmentUsers(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	department, err := h.findDepartmentOrNotFound(id)
	if err != nil {
		if isRecordNotFound(err) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}

	var userIDs []uint
	if err := h.DB.Table("user_departments").Where("department_id = ?", id).Pluck("user_id", &userIDs).Error; err != nil {
		response.Internal(c, err)
		return
	}

	var users []models.User
	if err := h.DB.Order("id asc").Find(&users).Error; err != nil {
		response.Internal(c, err)
		return
	}

	response.Success(c, gin.H{
		"department": department,
		"userIds":    userIDs,
		"users":      users,
	})
}

func (h *Handler) BindDepartmentUsers(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if _, err := h.findDepartmentOrNotFound(id); err != nil {
		if isRecordNotFound(err) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}

	var req struct {
		UserIDs []uint `json:"userIds"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if len(req.UserIDs) > maxBindBatchSize {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "userIds exceeds maximum size 200"))
		return
	}

	uniqueUserIDs := make([]uint, 0, len(req.UserIDs))
	seen := make(map[uint]struct{}, len(req.UserIDs))
	for _, userID := range req.UserIDs {
		if userID == 0 {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "userIds contains invalid id"))
			return
		}
		if _, exists := seen[userID]; exists {
			continue
		}
		seen[userID] = struct{}{}
		uniqueUserIDs = append(uniqueUserIDs, userID)
	}
	if len(uniqueUserIDs) > 0 {
		var count int64
		if err := h.DB.Model(&models.User{}).Where("id IN ?", uniqueUserIDs).Count(&count).Error; err != nil {
			response.Internal(c, err)
			return
		}
		if count != int64(len(uniqueUserIDs)) {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "userIds contains invalid id"))
			return
		}
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("department_id = ?", id).Delete(&models.UserDepartment{}).Error; err != nil {
			return err
		}
		for _, userID := range uniqueUserIDs {
			if err := tx.Create(&models.UserDepartment{UserID: userID, DepartmentID: id}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		response.Internal(c, err)
		return
	}

	response.Success(c, gin.H{"id": id, "userIds": uniqueUserIDs})
}

func (h *Handler) ListDepartmentTree(c *gin.Context) {
	var departments []models.Department
	if err := h.DB.Order("id asc").Find(&departments).Error; err != nil {
		response.Internal(c, err)
		return
	}

	nodes := make(map[uint]*departmentTreeNode, len(departments))
	for _, department := range departments {
		departmentID := department.ID
		nodes[departmentID] = &departmentTreeNode{
			ID:       departmentID,
			Name:     department.Name,
			ParentID: department.ParentID,
			Children: make([]*departmentTreeNode, 0),
		}
	}

	roots := make([]*departmentTreeNode, 0)
	for _, department := range departments {
		node := nodes[department.ID]
		if department.ParentID != nil {
			parentNode, ok := nodes[*department.ParentID]
			if ok {
				parentNode.Children = append(parentNode.Children, node)
				continue
			}
		}
		roots = append(roots, node)
	}

	response.Success(c, roots)
}

func (h *Handler) ListDepartments(c *gin.Context)  { listByModel[models.Department](c, h.DB) }
func (h *Handler) CreateDepartment(c *gin.Context) { createByModel[models.Department](c, h.DB) }
func (h *Handler) UpdateDepartment(c *gin.Context) { updateByModel[models.Department](c, h.DB) }
func (h *Handler) DeleteDepartment(c *gin.Context) { deleteByModel[models.Department](c, h.DB) }

func (h *Handler) GetDepartment(c *gin.Context) {
	if c.Param("id") == "tree" {
		h.ListDepartmentTree(c)
		return
	}
	getByID[models.Department](c, h.DB)
}

func (h *Handler) ListRoles(c *gin.Context) {
	page := pagination.Parse(c)
	query := h.DB.Model(&models.Role{})

	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	if builtInText := strings.TrimSpace(c.Query("builtIn")); builtInText != "" {
		switch strings.ToLower(builtInText) {
		case "true":
			query = query.Where("built_in = ?", true)
		case "false":
			query = query.Where("built_in = ?", false)
		default:
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "builtIn must be true or false"))
			return
		}
	}

	var (
		items []models.Role
		total int64
	)
	if err := query.Count(&total).Error; err != nil {
		response.Internal(c, err)
		return
	}
	if err := query.Order("id desc").Limit(page.PageSize).Offset(pagination.Offset(page)).Find(&items).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.List(c, items, total, page.Page, page.PageSize)
}

func (h *Handler) GetRole(c *gin.Context) { getByID[models.Role](c, h.DB) }

func (h *Handler) CreateRole(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if !bindJSON(c, &req) {
		return
	}
	role := models.Role{
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		BuiltIn:     false,
	}
	if role.Name == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "role name is required"))
		return
	}
	if err := h.DB.Create(&role).Error; err != nil {
		response.Internal(c, err)
		return
	}
	if err := h.syncRolePolicies(role); err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, role)
}

func (h *Handler) UpdateRole(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var role models.Role
	if err := h.DB.First(&role, id).Error; err != nil {
		if isRecordNotFound(err) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if !bindJSON(c, &req) {
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name != "" && role.BuiltIn && req.Name != role.Name {
		response.Error(c, http.StatusBadRequest, appErr.New(4003, "built-in role name cannot be changed"))
		return
	}

	oldName := role.Name
	updates := map[string]interface{}{"description": req.Description}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if err := h.DB.Model(&models.Role{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		response.Internal(c, err)
		return
	}

	if req.Name != "" && req.Name != oldName {
		if _, err := h.Enforcer.RemoveFilteredPolicy(0, oldName); err != nil {
			response.Internal(c, err)
			return
		}
		if err := h.Enforcer.SavePolicy(); err != nil {
			response.Internal(c, err)
			return
		}
	}
	if err := h.syncRolePoliciesByID(id); err != nil {
		response.Internal(c, err)
		return
	}
	getByID[models.Role](c, h.DB)
}

func (h *Handler) DeleteRole(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var role models.Role
	if err := h.DB.First(&role, id).Error; err != nil {
		if isRecordNotFound(err) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	if role.BuiltIn || role.Name == "admin" {
		response.Error(c, http.StatusBadRequest, appErr.New(4001, "built-in role cannot be deleted"))
		return
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", id).Delete(&models.RolePermission{}).Error; err != nil {
			return err
		}
		if err := tx.Where("role_id = ?", id).Delete(&models.UserRole{}).Error; err != nil {
			return err
		}
		return tx.Delete(&role).Error
	}); err != nil {
		response.Internal(c, err)
		return
	}
	if _, err := h.Enforcer.RemoveFilteredPolicy(0, role.Name); err != nil {
		response.Internal(c, err)
		return
	}
	if err := h.Enforcer.SavePolicy(); err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *Handler) BindRolePermissions(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if _, err := h.findRoleOrNotFound(id); err != nil {
		if isRecordNotFound(err) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	var req struct {
		PermissionIDs []uint `json:"permissionIds"`
	}
	if !bindJSON(c, &req) {
		return
	}
	uniquePermissionIDs := make([]uint, 0, len(req.PermissionIDs))
	seen := make(map[uint]struct{}, len(req.PermissionIDs))
	for _, permissionID := range req.PermissionIDs {
		if permissionID == 0 {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "permissionIds contains invalid id"))
			return
		}
		if _, exists := seen[permissionID]; exists {
			continue
		}
		seen[permissionID] = struct{}{}
		uniquePermissionIDs = append(uniquePermissionIDs, permissionID)
	}
	if len(uniquePermissionIDs) > 0 {
		var count int64
		if err := h.DB.Model(&models.Permission{}).Where("id IN ?", uniquePermissionIDs).Count(&count).Error; err != nil {
			response.Internal(c, err)
			return
		}
		if count != int64(len(uniquePermissionIDs)) {
			response.Error(c, http.StatusBadRequest, appErr.New(3001, "permissionIds contains invalid id"))
			return
		}
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", id).Delete(&models.RolePermission{}).Error; err != nil {
			return err
		}
		for _, permissionID := range uniquePermissionIDs {
			if err := tx.Create(&models.RolePermission{RoleID: id, PermissionID: permissionID}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		response.Internal(c, err)
		return
	}
	if err := h.syncRolePoliciesByID(id); err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{"id": id, "permissionIds": uniquePermissionIDs})
}

func (h *Handler) GetRolePermissions(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	role, err := h.findRoleOrNotFound(id)
	if err != nil {
		if isRecordNotFound(err) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}

	var permissionIDs []uint
	if err := h.DB.Table("role_permissions").Where("role_id = ?", id).Pluck("permission_id", &permissionIDs).Error; err != nil {
		response.Internal(c, err)
		return
	}

	var permissions []models.Permission
	if err := h.DB.Order("type asc, id asc").Find(&permissions).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{
		"role":          role,
		"permissionIds": permissionIDs,
		"permissions":   permissions,
	})
}

func (h *Handler) ListPermissions(c *gin.Context) {
	page := pagination.Parse(c)
	query := h.DB.Model(&models.Permission{})

	if typeFilter := strings.TrimSpace(c.Query("type")); typeFilter != "" {
		query = query.Where("type = ?", typeFilter)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where(
			"name LIKE ? OR key LIKE ? OR resource LIKE ? OR action LIKE ? OR description LIKE ?",
			"%"+keyword+"%",
			"%"+keyword+"%",
			"%"+keyword+"%",
			"%"+keyword+"%",
			"%"+keyword+"%",
		)
	}

	var items []models.Permission
	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Internal(c, err)
		return
	}
	if err := query.Order("id desc").Limit(page.PageSize).Offset(pagination.Offset(page)).Find(&items).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.List(c, items, total, page.Page, page.PageSize)
}

func (h *Handler) GetPermission(c *gin.Context) { getByID[models.Permission](c, h.DB) }

func (h *Handler) CreatePermission(c *gin.Context) {
	var permission models.Permission
	if !bindJSON(c, &permission) {
		return
	}
	normalizePermission(&permission)
	if permission.Name == "" || permission.Resource == "" || permission.Action == "" {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, "name/resource/action are required"))
		return
	}
	if err := h.DB.Create(&permission).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, permission)
}

func (h *Handler) UpdatePermission(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var current models.Permission
	if err := h.DB.First(&current, id).Error; err != nil {
		if isRecordNotFound(err) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}

	var req models.Permission
	if !bindJSON(c, &req) {
		return
	}
	if req.Name == "" {
		req.Name = current.Name
	}
	if req.Resource == "" {
		req.Resource = current.Resource
	}
	if req.Action == "" {
		req.Action = current.Action
	}
	if req.Type == "" {
		req.Type = current.Type
	}
	if req.Key == "" {
		req.Key = current.Key
	}
	if req.DeptScope == "" {
		req.DeptScope = current.DeptScope
	}
	if req.ResourceTagScope == "" {
		req.ResourceTagScope = current.ResourceTagScope
	}
	if req.EnvScope == "" {
		req.EnvScope = current.EnvScope
	}
	if req.Description == "" {
		req.Description = current.Description
	}
	normalizePermission(&req)

	if err := h.DB.Model(&models.Permission{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name":               req.Name,
		"resource":           req.Resource,
		"action":             req.Action,
		"type":               req.Type,
		"key":                req.Key,
		"dept_scope":         req.DeptScope,
		"resource_tag_scope": req.ResourceTagScope,
		"env_scope":          req.EnvScope,
		"description":        req.Description,
	}).Error; err != nil {
		response.Internal(c, err)
		return
	}
	if err := h.syncPoliciesByPermissionID(id); err != nil {
		response.Internal(c, err)
		return
	}
	getByID[models.Permission](c, h.DB)
}

func (h *Handler) DeletePermission(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var roleIDs []uint
	if err := h.DB.Table("role_permissions").Where("permission_id = ?", id).Pluck("role_id", &roleIDs).Error; err != nil {
		response.Internal(c, err)
		return
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("permission_id = ?", id).Delete(&models.RolePermission{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Permission{}, id).Error
	}); err != nil {
		response.Internal(c, err)
		return
	}
	for _, roleID := range roleIDs {
		if err := h.syncRolePoliciesByID(roleID); err != nil {
			response.Internal(c, err)
			return
		}
	}
	response.Success(c, gin.H{"id": id})
}
