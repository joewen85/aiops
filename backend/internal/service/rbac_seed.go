package service

import (
	"errors"

	"gorm.io/gorm"

	"devops-system/backend/internal/models"
)

func SeedRBACDefaults(database *gorm.DB) error {
	if database == nil {
		return nil
	}
	if err := SeedDefaultAdmin(database); err != nil {
		return err
	}
	if err := seedPermissionCatalog(database); err != nil {
		return err
	}
	return bindAllPermissionsToAdmin(database)
}

func seedPermissionCatalog(database *gorm.DB) error {
	for _, item := range permissionSeeds() {
		item.DeptScope = normalizeScope(item.DeptScope)
		item.ResourceTagScope = normalizeScope(item.ResourceTagScope)
		item.EnvScope = normalizeScope(item.EnvScope)

		var current models.Permission
		err := database.Where("type = ? AND key = ?", item.Type, item.Key).First(&current).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if createErr := database.Create(&item).Error; createErr != nil {
				return createErr
			}
			continue
		}
		if err != nil {
			return err
		}

		updates := map[string]interface{}{
			"name":               item.Name,
			"resource":           item.Resource,
			"action":             item.Action,
			"description":        item.Description,
			"dept_scope":         item.DeptScope,
			"resource_tag_scope": item.ResourceTagScope,
			"env_scope":          item.EnvScope,
		}
		if updateErr := database.Model(&models.Permission{}).Where("id = ?", current.ID).Updates(updates).Error; updateErr != nil {
			return updateErr
		}
	}
	return nil
}

func bindAllPermissionsToAdmin(database *gorm.DB) error {
	var adminRole models.Role
	if err := database.Where("name = ?", "admin").First(&adminRole).Error; err != nil {
		return err
	}

	var permissionIDs []uint
	if err := database.Model(&models.Permission{}).Pluck("id", &permissionIDs).Error; err != nil {
		return err
	}
	if len(permissionIDs) == 0 {
		return nil
	}

	var existing []models.RolePermission
	if err := database.Where("role_id = ?", adminRole.ID).Find(&existing).Error; err != nil {
		return err
	}
	existingSet := make(map[uint]struct{}, len(existing))
	for _, item := range existing {
		existingSet[item.PermissionID] = struct{}{}
	}

	missing := make([]models.RolePermission, 0, len(permissionIDs))
	for _, permissionID := range permissionIDs {
		if _, ok := existingSet[permissionID]; ok {
			continue
		}
		missing = append(missing, models.RolePermission{
			RoleID:       adminRole.ID,
			PermissionID: permissionID,
		})
	}
	if len(missing) == 0 {
		return nil
	}
	return database.Create(&missing).Error
}

func normalizeScope(scope string) string {
	if scope == "" {
		return "*"
	}
	return scope
}

func permissionSeeds() []models.Permission {
	seeds := make([]models.Permission, 0, 256)

	addMenu := func(key string, name string) {
		seeds = append(seeds, models.Permission{
			Name:             name,
			Type:             "menu",
			Key:              key,
			Resource:         key,
			Action:           "view",
			DeptScope:        "*",
			ResourceTagScope: "*",
			EnvScope:         "*",
			Description:      "菜单权限：" + name,
		})
	}
	addButton := func(key string, name string) {
		seeds = append(seeds, models.Permission{
			Name:             name,
			Type:             "button",
			Key:              key,
			Resource:         key,
			Action:           "click",
			DeptScope:        "*",
			ResourceTagScope: "*",
			EnvScope:         "*",
			Description:      "按钮权限：" + name,
		})
	}
	addAPI := func(key string, method string, path string, name string) {
		seeds = append(seeds, models.Permission{
			Name:             name,
			Type:             "api",
			Key:              key,
			Resource:         path,
			Action:           method,
			DeptScope:        "*",
			ResourceTagScope: "*",
			EnvScope:         "*",
			Description:      "API权限：" + name,
		})
	}

	addMenu("menu.dashboard", "概览")
	addMenu("menu.rbac", "RBAC/ABAC 权限管理")
	addMenu("menu.users", "用户与部门")
	addMenu("menu.cmdb", "CMDB")
	addMenu("menu.tasks", "任务中心")
	addMenu("menu.messages", "站内消息")
	addMenu("menu.cloud", "多云管理")
	addMenu("menu.tickets", "工单管理")
	addMenu("menu.docker", "Docker 管理")
	addMenu("menu.middleware", "中间件管理")
	addMenu("menu.observability", "可观测性")
	addMenu("menu.kubernetes", "Kubernetes 管理")
	addMenu("menu.events", "事件中心")
	addMenu("menu.tools", "工具市场")
	addMenu("menu.aiops", "AIOps")
	addMenu("menu.audit", "审计日志")

	addButton("button.rbac.role.create", "创建角色")
	addButton("button.rbac.role.detail", "查看角色详情")
	addButton("button.rbac.role.update", "更新角色")
	addButton("button.rbac.role.delete", "删除角色")
	addButton("button.rbac.permission.create", "创建权限")
	addButton("button.rbac.permission.detail", "查看权限详情")
	addButton("button.rbac.permission.update", "更新权限")
	addButton("button.rbac.permission.delete", "删除权限")
	addButton("button.rbac.binding.save", "保存角色-权限绑定")
	addButton("button.users.user.create", "创建用户")
	addButton("button.users.user.update", "更新用户")
	addButton("button.users.user.delete", "删除用户")
	addButton("button.users.user.toggle_status", "启停用户")
	addButton("button.users.user.reset_password", "重置用户密码")
	addButton("button.users.user.bind_roles", "绑定用户角色")
	addButton("button.users.user.bind_departments", "绑定用户部门")
	addButton("button.users.department.create", "创建部门")
	addButton("button.users.department.update", "更新部门")
	addButton("button.users.department.delete", "删除部门")
	addButton("button.users.department.bind_members", "绑定部门成员")
	addButton("button.cmdb.category.create", "创建资源分类")
	addButton("button.cmdb.category.update", "更新资源分类")
	addButton("button.cmdb.category.delete", "删除资源分类")
	addButton("button.cmdb.resource.create", "创建资源")
	addButton("button.cmdb.resource.update", "更新资源")
	addButton("button.cmdb.resource.delete", "删除资源")
	addButton("button.cmdb.resource.detail", "查看资源详情")
	addButton("button.cmdb.resource.restart", "重启云服务器")
	addButton("button.cmdb.resource.stop", "停止云服务器")
	addButton("button.cmdb.resource.bind_tags", "绑定资源标签")
	addButton("button.cmdb.resource.graph", "查看资源依赖图谱")
	addButton("button.cmdb.tag.create", "创建标签")
	addButton("button.cmdb.tag.update", "更新标签")
	addButton("button.cmdb.tag.delete", "删除标签")
	addButton("button.cmdb.relation.create", "创建资源关系")
	addButton("button.cmdb.sync.create", "创建CMDB同步任务")
	addButton("button.cmdb.sync.retry", "重试CMDB同步任务")
	addButton("button.tasks.task.create", "创建任务")
	addButton("button.tasks.task.update", "更新任务")
	addButton("button.tasks.task.delete", "删除任务")
	addButton("button.tasks.task.execute", "执行任务")
	addButton("button.tasks.playbook.create", "创建Playbook")
	addButton("button.tasks.playbook.update", "更新Playbook")
	addButton("button.tasks.playbook.delete", "删除Playbook")
	addButton("button.messages.message.create", "创建站内消息")
	addButton("button.messages.message.mark_read", "标记消息已读")
	addButton("button.cloud.account.create", "创建云账号")
	addButton("button.cloud.account.update", "更新云账号")
	addButton("button.cloud.account.delete", "删除云账号")
	addButton("button.cloud.account.verify", "校验云账号")
	addButton("button.cloud.account.sync", "同步云资源")
	addButton("button.cloud.asset.create", "创建云资源")
	addButton("button.cloud.asset.update", "更新云资源")
	addButton("button.cloud.asset.delete", "删除云资源")
	addButton("button.tickets.ticket.create", "创建工单")
	addButton("button.tickets.ticket.update", "更新工单")
	addButton("button.tickets.ticket.delete", "删除工单")
	addButton("button.tickets.ticket.approve", "审批工单")
	addButton("button.tickets.ticket.transition", "流转工单")
	addButton("button.docker.host.create", "创建Docker主机")
	addButton("button.docker.host.update", "更新Docker主机")
	addButton("button.docker.host.delete", "删除Docker主机")
	addButton("button.docker.compose_stack.create", "创建Compose栈")
	addButton("button.docker.compose_stack.update", "更新Compose栈")
	addButton("button.docker.compose_stack.delete", "删除Compose栈")
	addButton("button.docker.action.run", "执行Docker动作")
	addButton("button.middleware.instance.create", "创建中间件实例")
	addButton("button.middleware.instance.update", "更新中间件实例")
	addButton("button.middleware.instance.delete", "删除中间件实例")
	addButton("button.middleware.instance.check", "检查中间件实例")
	addButton("button.middleware.instance.action", "执行中间件动作")
	addButton("button.observability.source.create", "创建可观测数据源")
	addButton("button.observability.source.update", "更新可观测数据源")
	addButton("button.observability.source.delete", "删除可观测数据源")
	addButton("button.observability.source.query_metrics", "查询指标")
	addButton("button.kubernetes.cluster.create", "创建K8s集群")
	addButton("button.kubernetes.cluster.update", "更新K8s集群")
	addButton("button.kubernetes.cluster.delete", "删除K8s集群")
	addButton("button.kubernetes.resource.action", "执行K8s资源动作")
	addButton("button.events.event.create", "创建事件")
	addButton("button.events.event.update", "更新事件")
	addButton("button.events.event.delete", "删除事件")
	addButton("button.events.event.link", "关联事件")
	addButton("button.tool_market.tool.create", "创建工具")
	addButton("button.tool_market.tool.update", "更新工具")
	addButton("button.tool_market.tool.delete", "删除工具")
	addButton("button.tool_market.tool.execute", "执行工具")
	addButton("button.aiops.agent.create", "创建AIOps智能体")
	addButton("button.aiops.agent.update", "更新AIOps智能体")
	addButton("button.aiops.agent.delete", "删除AIOps智能体")
	addButton("button.aiops.model.create", "创建AIOps模型")
	addButton("button.aiops.model.update", "更新AIOps模型")
	addButton("button.aiops.model.delete", "删除AIOps模型")
	addButton("button.aiops.chat.send", "发送AIOps对话")
	addButton("button.aiops.rca.execute", "执行AIOps根因分析")
	addButton("button.aiops.procurement.intent_parse", "解析AIOps采购意图")
	addButton("button.aiops.procurement.plan_create", "生成AIOps采购计划")
	addButton("button.aiops.procurement.execute", "执行AIOps采购计划")

	addAPI("api.users.user.list", "GET", "/api/v1/users", "查询用户列表")
	addAPI("api.users.user.create", "POST", "/api/v1/users", "创建用户")
	addAPI("api.users.user.get", "GET", "/api/v1/users/:id", "查询用户详情")
	addAPI("api.users.user.update", "PUT", "/api/v1/users/:id", "更新用户")
	addAPI("api.users.user.delete", "DELETE", "/api/v1/users/:id", "删除用户")
	addAPI("api.users.user.status_patch", "PATCH", "/api/v1/users/:id/status", "更新用户状态")
	addAPI("api.users.user.password_reset", "POST", "/api/v1/users/:id/reset-password", "重置用户密码")
	addAPI("api.users.user.roles_get", "GET", "/api/v1/users/:id/roles", "查询用户角色绑定")
	addAPI("api.users.user.roles_bind", "POST", "/api/v1/users/:id/roles", "绑定用户角色")
	addAPI("api.users.user.departments_get", "GET", "/api/v1/users/:id/departments", "查询用户部门绑定")
	addAPI("api.users.user.departments_bind", "POST", "/api/v1/users/:id/departments", "绑定用户部门")
	addAPI("api.users.department.list", "GET", "/api/v1/departments", "查询部门列表")
	addAPI("api.users.department.create", "POST", "/api/v1/departments", "创建部门")
	addAPI("api.users.department.tree", "GET", "/api/v1/departments/tree", "查询部门树")
	addAPI("api.users.department.get", "GET", "/api/v1/departments/:id", "查询部门详情")
	addAPI("api.users.department.update", "PUT", "/api/v1/departments/:id", "更新部门")
	addAPI("api.users.department.delete", "DELETE", "/api/v1/departments/:id", "删除部门")
	addAPI("api.users.department.members_get", "GET", "/api/v1/departments/:id/users", "查询部门成员绑定")
	addAPI("api.users.department.members_bind", "POST", "/api/v1/departments/:id/users", "绑定部门成员")

	addAPI("api.rbac.role.list", "GET", "/api/v1/roles", "查询角色列表")
	addAPI("api.rbac.role.create", "POST", "/api/v1/roles", "创建角色")
	addAPI("api.rbac.role.get", "GET", "/api/v1/roles/:id", "查询角色详情")
	addAPI("api.rbac.role.update", "PUT", "/api/v1/roles/:id", "更新角色")
	addAPI("api.rbac.role.delete", "DELETE", "/api/v1/roles/:id", "删除角色")
	addAPI("api.rbac.role.permissions_list", "GET", "/api/v1/roles/:id/permissions", "查询角色权限")
	addAPI("api.rbac.role.permissions_bind", "POST", "/api/v1/roles/:id/permissions", "绑定角色权限")
	addAPI("api.rbac.permission.list", "GET", "/api/v1/permissions", "查询权限列表")
	addAPI("api.rbac.permission.create", "POST", "/api/v1/permissions", "创建权限")
	addAPI("api.rbac.permission.get", "GET", "/api/v1/permissions/:id", "查询权限详情")
	addAPI("api.rbac.permission.update", "PUT", "/api/v1/permissions/:id", "更新权限")
	addAPI("api.rbac.permission.delete", "DELETE", "/api/v1/permissions/:id", "删除权限")

	addAPI("api.cmdb.category.list", "GET", "/api/v1/cmdb/categories", "查询资源分类列表")
	addAPI("api.cmdb.category.create", "POST", "/api/v1/cmdb/categories", "创建资源分类")
	addAPI("api.cmdb.category.get", "GET", "/api/v1/cmdb/categories/:id", "查询资源分类详情")
	addAPI("api.cmdb.category.update", "PUT", "/api/v1/cmdb/categories/:id", "更新资源分类")
	addAPI("api.cmdb.category.delete", "DELETE", "/api/v1/cmdb/categories/:id", "删除资源分类")
	addAPI("api.cmdb.resource.list", "GET", "/api/v1/cmdb/resources", "查询资源列表")
	addAPI("api.cmdb.resource.create", "POST", "/api/v1/cmdb/resources", "创建资源")
	addAPI("api.cmdb.resource.get", "GET", "/api/v1/cmdb/resources/:id", "查询资源详情")
	addAPI("api.cmdb.resource.update", "PUT", "/api/v1/cmdb/resources/:id", "更新资源")
	addAPI("api.cmdb.resource.delete", "DELETE", "/api/v1/cmdb/resources/:id", "删除资源")
	addAPI("api.cmdb.resource.restart", "POST", "/api/v1/cmdb/resources/:id/actions/restart", "重启云服务器资源")
	addAPI("api.cmdb.resource.stop", "POST", "/api/v1/cmdb/resources/:id/actions/stop", "停止云服务器资源")
	addAPI("api.cmdb.resource.tags_bind", "POST", "/api/v1/cmdb/resources/:id/tags", "绑定资源标签")
	addAPI("api.cmdb.resource.upstream", "GET", "/api/v1/cmdb/resources/:id/upstream", "查询资源上游依赖")
	addAPI("api.cmdb.resource.downstream", "GET", "/api/v1/cmdb/resources/:id/downstream", "查询资源下游依赖")
	addAPI("api.cmdb.tag.list", "GET", "/api/v1/cmdb/tags", "查询标签列表")
	addAPI("api.cmdb.tag.create", "POST", "/api/v1/cmdb/tags", "创建标签")
	addAPI("api.cmdb.tag.get", "GET", "/api/v1/cmdb/tags/:id", "查询标签详情")
	addAPI("api.cmdb.tag.update", "PUT", "/api/v1/cmdb/tags/:id", "更新标签")
	addAPI("api.cmdb.tag.delete", "DELETE", "/api/v1/cmdb/tags/:id", "删除标签")
	addAPI("api.cmdb.relation.list", "GET", "/api/v1/cmdb/relations", "查询资源关系列表")
	addAPI("api.cmdb.relation.create", "POST", "/api/v1/cmdb/relations", "创建资源关系")
	addAPI("api.cmdb.topology.get", "GET", "/api/v1/cmdb/topology/:application", "查询业务拓扑视图")
	addAPI("api.cmdb.impact.get", "GET", "/api/v1/cmdb/impact/:ciId", "查询故障影响视图")
	addAPI("api.cmdb.failover.get", "GET", "/api/v1/cmdb/regions/:region/failover", "查询地域容灾视图")
	addAPI("api.cmdb.change_impact.get", "GET", "/api/v1/cmdb/change-impact/:releaseId", "查询变更影响视图")
	addAPI("api.cmdb.sync.create", "POST", "/api/v1/cmdb/sync/jobs", "创建CMDB同步任务")
	addAPI("api.cmdb.sync.get", "GET", "/api/v1/cmdb/sync/jobs/:id", "查询CMDB同步任务详情")
	addAPI("api.cmdb.sync.retry", "POST", "/api/v1/cmdb/sync/jobs/:id/retry", "重试CMDB同步任务")

	addAPI("api.tasks.task.list", "GET", "/api/v1/tasks", "查询任务列表")
	addAPI("api.tasks.task.create", "POST", "/api/v1/tasks", "创建任务")
	addAPI("api.tasks.task.get", "GET", "/api/v1/tasks/:id", "查询任务详情")
	addAPI("api.tasks.task.update", "PUT", "/api/v1/tasks/:id", "更新任务")
	addAPI("api.tasks.task.delete", "DELETE", "/api/v1/tasks/:id", "删除任务")
	addAPI("api.tasks.task.execute", "POST", "/api/v1/tasks/:id/execute", "执行任务")
	addAPI("api.tasks.playbook.list", "GET", "/api/v1/playbooks", "查询Playbook列表")
	addAPI("api.tasks.playbook.create", "POST", "/api/v1/playbooks", "创建Playbook")
	addAPI("api.tasks.playbook.get", "GET", "/api/v1/playbooks/:id", "查询Playbook详情")
	addAPI("api.tasks.playbook.update", "PUT", "/api/v1/playbooks/:id", "更新Playbook")
	addAPI("api.tasks.playbook.delete", "DELETE", "/api/v1/playbooks/:id", "删除Playbook")
	addAPI("api.tasks.log.list", "GET", "/api/v1/task-logs", "查询任务执行日志列表")
	addAPI("api.tasks.log.get", "GET", "/api/v1/task-logs/:id", "查询任务执行日志详情")

	addAPI("api.messages.message.list", "GET", "/api/v1/messages", "查询站内消息列表")
	addAPI("api.messages.message.create", "POST", "/api/v1/messages", "创建站内消息")
	addAPI("api.messages.message.read", "POST", "/api/v1/messages/:id/read", "标记消息已读")

	addAPI("api.cloud.account.list", "GET", "/api/v1/cloud/accounts", "查询云账号列表")
	addAPI("api.cloud.account.create", "POST", "/api/v1/cloud/accounts", "创建云账号")
	addAPI("api.cloud.account.get", "GET", "/api/v1/cloud/accounts/:id", "查询云账号详情")
	addAPI("api.cloud.account.update", "PUT", "/api/v1/cloud/accounts/:id", "更新云账号")
	addAPI("api.cloud.account.delete", "DELETE", "/api/v1/cloud/accounts/:id", "删除云账号")
	addAPI("api.cloud.account.verify", "POST", "/api/v1/cloud/accounts/:id/verify", "校验云账号")
	addAPI("api.cloud.account.sync", "POST", "/api/v1/cloud/accounts/:id/sync", "同步云资源")
	addAPI("api.cloud.account.assets", "GET", "/api/v1/cloud/accounts/:id/assets", "查询云账号资源")
	addAPI("api.cloud.asset.list", "GET", "/api/v1/cloud/assets", "查询云资源列表")
	addAPI("api.cloud.asset.create", "POST", "/api/v1/cloud/assets", "创建云资源")
	addAPI("api.cloud.asset.get", "GET", "/api/v1/cloud/assets/:id", "查询云资源详情")
	addAPI("api.cloud.asset.update", "PUT", "/api/v1/cloud/assets/:id", "更新云资源")
	addAPI("api.cloud.asset.delete", "DELETE", "/api/v1/cloud/assets/:id", "删除云资源")

	addAPI("api.tickets.ticket.list", "GET", "/api/v1/tickets", "查询工单列表")
	addAPI("api.tickets.ticket.create", "POST", "/api/v1/tickets", "创建工单")
	addAPI("api.tickets.ticket.get", "GET", "/api/v1/tickets/:id", "查询工单详情")
	addAPI("api.tickets.ticket.update", "PUT", "/api/v1/tickets/:id", "更新工单")
	addAPI("api.tickets.ticket.delete", "DELETE", "/api/v1/tickets/:id", "删除工单")
	addAPI("api.tickets.ticket.approve", "POST", "/api/v1/tickets/:id/approve", "审批工单")
	addAPI("api.tickets.ticket.transition", "POST", "/api/v1/tickets/:id/transition", "流转工单")

	addAPI("api.docker.host.list", "GET", "/api/v1/docker/hosts", "查询Docker主机列表")
	addAPI("api.docker.host.create", "POST", "/api/v1/docker/hosts", "创建Docker主机")
	addAPI("api.docker.host.get", "GET", "/api/v1/docker/hosts/:id", "查询Docker主机详情")
	addAPI("api.docker.host.update", "PUT", "/api/v1/docker/hosts/:id", "更新Docker主机")
	addAPI("api.docker.host.delete", "DELETE", "/api/v1/docker/hosts/:id", "删除Docker主机")
	addAPI("api.docker.compose_stack.list", "GET", "/api/v1/docker/compose/stacks", "查询Compose栈列表")
	addAPI("api.docker.compose_stack.create", "POST", "/api/v1/docker/compose/stacks", "创建Compose栈")
	addAPI("api.docker.compose_stack.update", "PUT", "/api/v1/docker/compose/stacks/:id", "更新Compose栈")
	addAPI("api.docker.compose_stack.delete", "DELETE", "/api/v1/docker/compose/stacks/:id", "删除Compose栈")
	addAPI("api.docker.action.run", "POST", "/api/v1/docker/actions", "执行Docker动作")

	addAPI("api.middleware.instance.list", "GET", "/api/v1/middleware/instances", "查询中间件实例列表")
	addAPI("api.middleware.instance.create", "POST", "/api/v1/middleware/instances", "创建中间件实例")
	addAPI("api.middleware.instance.get", "GET", "/api/v1/middleware/instances/:id", "查询中间件实例详情")
	addAPI("api.middleware.instance.update", "PUT", "/api/v1/middleware/instances/:id", "更新中间件实例")
	addAPI("api.middleware.instance.delete", "DELETE", "/api/v1/middleware/instances/:id", "删除中间件实例")
	addAPI("api.middleware.instance.check", "POST", "/api/v1/middleware/instances/:id/check", "检查中间件实例")
	addAPI("api.middleware.instance.action", "POST", "/api/v1/middleware/instances/:id/action", "执行中间件动作")

	addAPI("api.observability.source.list", "GET", "/api/v1/observability/sources", "查询可观测数据源列表")
	addAPI("api.observability.source.create", "POST", "/api/v1/observability/sources", "创建可观测数据源")
	addAPI("api.observability.source.get", "GET", "/api/v1/observability/sources/:id", "查询可观测数据源详情")
	addAPI("api.observability.source.update", "PUT", "/api/v1/observability/sources/:id", "更新可观测数据源")
	addAPI("api.observability.source.delete", "DELETE", "/api/v1/observability/sources/:id", "删除可观测数据源")
	addAPI("api.observability.metrics.query", "GET", "/api/v1/observability/metrics/query", "查询可观测指标")

	addAPI("api.kubernetes.cluster.list", "GET", "/api/v1/kubernetes/clusters", "查询K8s集群列表")
	addAPI("api.kubernetes.cluster.create", "POST", "/api/v1/kubernetes/clusters", "创建K8s集群")
	addAPI("api.kubernetes.cluster.get", "GET", "/api/v1/kubernetes/clusters/:id", "查询K8s集群详情")
	addAPI("api.kubernetes.cluster.update", "PUT", "/api/v1/kubernetes/clusters/:id", "更新K8s集群")
	addAPI("api.kubernetes.cluster.delete", "DELETE", "/api/v1/kubernetes/clusters/:id", "删除K8s集群")
	addAPI("api.kubernetes.node.list", "GET", "/api/v1/kubernetes/nodes", "查询K8s节点列表")
	addAPI("api.kubernetes.resource.action", "POST", "/api/v1/kubernetes/resources/action", "执行K8s资源动作")

	addAPI("api.events.event.list", "GET", "/api/v1/events", "查询事件列表")
	addAPI("api.events.event.create", "POST", "/api/v1/events", "创建事件")
	addAPI("api.events.event.get", "GET", "/api/v1/events/:id", "查询事件详情")
	addAPI("api.events.event.update", "PUT", "/api/v1/events/:id", "更新事件")
	addAPI("api.events.event.delete", "DELETE", "/api/v1/events/:id", "删除事件")
	addAPI("api.events.event.search", "GET", "/api/v1/events/search", "检索事件")
	addAPI("api.events.event.link", "POST", "/api/v1/events/:id/link", "关联事件")

	addAPI("api.tool_market.tool.list", "GET", "/api/v1/tool-market/tools", "查询工具列表")
	addAPI("api.tool_market.tool.create", "POST", "/api/v1/tool-market/tools", "创建工具")
	addAPI("api.tool_market.tool.get", "GET", "/api/v1/tool-market/tools/:id", "查询工具详情")
	addAPI("api.tool_market.tool.update", "PUT", "/api/v1/tool-market/tools/:id", "更新工具")
	addAPI("api.tool_market.tool.delete", "DELETE", "/api/v1/tool-market/tools/:id", "删除工具")
	addAPI("api.tool_market.tool.execute", "POST", "/api/v1/tool-market/tools/:id/execute", "执行工具")

	addAPI("api.aiops.agent.list", "GET", "/api/v1/aiops/agents", "查询AIOps智能体列表")
	addAPI("api.aiops.agent.create", "POST", "/api/v1/aiops/agents", "创建AIOps智能体")
	addAPI("api.aiops.agent.get", "GET", "/api/v1/aiops/agents/:id", "查询AIOps智能体详情")
	addAPI("api.aiops.agent.update", "PUT", "/api/v1/aiops/agents/:id", "更新AIOps智能体")
	addAPI("api.aiops.agent.delete", "DELETE", "/api/v1/aiops/agents/:id", "删除AIOps智能体")
	addAPI("api.aiops.model.list", "GET", "/api/v1/aiops/models", "查询AIOps模型列表")
	addAPI("api.aiops.model.create", "POST", "/api/v1/aiops/models", "创建AIOps模型")
	addAPI("api.aiops.model.get", "GET", "/api/v1/aiops/models/:id", "查询AIOps模型详情")
	addAPI("api.aiops.model.update", "PUT", "/api/v1/aiops/models/:id", "更新AIOps模型")
	addAPI("api.aiops.model.delete", "DELETE", "/api/v1/aiops/models/:id", "删除AIOps模型")
	addAPI("api.aiops.chat", "POST", "/api/v1/aiops/chat", "AIOps对话")
	addAPI("api.aiops.rca", "POST", "/api/v1/aiops/rca", "AIOps根因分析")
	addAPI("api.aiops.procurement.protocol", "GET", "/api/v1/aiops/procurement/protocol", "查询AIOps采购协议")
	addAPI("api.aiops.procurement.intent_parse", "POST", "/api/v1/aiops/procurement/intents", "解析AIOps采购意图")
	addAPI("api.aiops.procurement.plan_create", "POST", "/api/v1/aiops/procurement/plans", "生成AIOps采购计划")
	addAPI("api.aiops.procurement.execute", "POST", "/api/v1/aiops/procurement/executions", "执行AIOps采购计划")

	addAPI("api.audit.log.list", "GET", "/api/v1/audit-logs", "查询审计日志列表")

	return seeds
}
