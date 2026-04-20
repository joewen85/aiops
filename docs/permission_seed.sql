-- RBAC/ABAC 权限标识初始化 SQL
-- 说明：该脚本用于手工初始化 permissions 与 admin 角色绑定。
-- 首次启动后端时，应用也会自动执行同等种子逻辑（幂等）。
BEGIN;
INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '概览', 'menu.dashboard', 'view', 'menu', 'menu.dashboard', '*', '*', '*', '菜单权限：概览', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'menu' AND key = 'menu.dashboard');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT 'RBAC/ABAC 权限管理', 'menu.rbac', 'view', 'menu', 'menu.rbac', '*', '*', '*', '菜单权限：RBAC/ABAC 权限管理', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'menu' AND key = 'menu.rbac');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '用户与部门', 'menu.users', 'view', 'menu', 'menu.users', '*', '*', '*', '菜单权限：用户与部门', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'menu' AND key = 'menu.users');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT 'CMDB', 'menu.cmdb', 'view', 'menu', 'menu.cmdb', '*', '*', '*', '菜单权限：CMDB', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'menu' AND key = 'menu.cmdb');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '任务中心', 'menu.tasks', 'view', 'menu', 'menu.tasks', '*', '*', '*', '菜单权限：任务中心', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'menu' AND key = 'menu.tasks');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '站内消息', 'menu.messages', 'view', 'menu', 'menu.messages', '*', '*', '*', '菜单权限：站内消息', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'menu' AND key = 'menu.messages');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '多云管理', 'menu.cloud', 'view', 'menu', 'menu.cloud', '*', '*', '*', '菜单权限：多云管理', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'menu' AND key = 'menu.cloud');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '工单管理', 'menu.tickets', 'view', 'menu', 'menu.tickets', '*', '*', '*', '菜单权限：工单管理', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'menu' AND key = 'menu.tickets');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT 'Docker 管理', 'menu.docker', 'view', 'menu', 'menu.docker', '*', '*', '*', '菜单权限：Docker 管理', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'menu' AND key = 'menu.docker');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '中间件管理', 'menu.middleware', 'view', 'menu', 'menu.middleware', '*', '*', '*', '菜单权限：中间件管理', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'menu' AND key = 'menu.middleware');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '可观测性', 'menu.observability', 'view', 'menu', 'menu.observability', '*', '*', '*', '菜单权限：可观测性', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'menu' AND key = 'menu.observability');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT 'Kubernetes 管理', 'menu.kubernetes', 'view', 'menu', 'menu.kubernetes', '*', '*', '*', '菜单权限：Kubernetes 管理', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'menu' AND key = 'menu.kubernetes');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '事件中心', 'menu.events', 'view', 'menu', 'menu.events', '*', '*', '*', '菜单权限：事件中心', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'menu' AND key = 'menu.events');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '工具市场', 'menu.tools', 'view', 'menu', 'menu.tools', '*', '*', '*', '菜单权限：工具市场', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'menu' AND key = 'menu.tools');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT 'AIOps', 'menu.aiops', 'view', 'menu', 'menu.aiops', '*', '*', '*', '菜单权限：AIOps', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'menu' AND key = 'menu.aiops');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '审计日志', 'menu.audit', 'view', 'menu', 'menu.audit', '*', '*', '*', '菜单权限：审计日志', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'menu' AND key = 'menu.audit');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建角色', 'button.rbac.role.create', 'click', 'button', 'button.rbac.role.create', '*', '*', '*', '按钮权限：创建角色', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.rbac.role.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查看角色详情', 'button.rbac.role.detail', 'click', 'button', 'button.rbac.role.detail', '*', '*', '*', '按钮权限：查看角色详情', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.rbac.role.detail');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新角色', 'button.rbac.role.update', 'click', 'button', 'button.rbac.role.update', '*', '*', '*', '按钮权限：更新角色', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.rbac.role.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除角色', 'button.rbac.role.delete', 'click', 'button', 'button.rbac.role.delete', '*', '*', '*', '按钮权限：删除角色', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.rbac.role.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建权限', 'button.rbac.permission.create', 'click', 'button', 'button.rbac.permission.create', '*', '*', '*', '按钮权限：创建权限', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.rbac.permission.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查看权限详情', 'button.rbac.permission.detail', 'click', 'button', 'button.rbac.permission.detail', '*', '*', '*', '按钮权限：查看权限详情', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.rbac.permission.detail');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新权限', 'button.rbac.permission.update', 'click', 'button', 'button.rbac.permission.update', '*', '*', '*', '按钮权限：更新权限', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.rbac.permission.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除权限', 'button.rbac.permission.delete', 'click', 'button', 'button.rbac.permission.delete', '*', '*', '*', '按钮权限：删除权限', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.rbac.permission.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '保存角色-权限绑定', 'button.rbac.binding.save', 'click', 'button', 'button.rbac.binding.save', '*', '*', '*', '按钮权限：保存角色-权限绑定', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.rbac.binding.save');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建用户', 'button.users.user.create', 'click', 'button', 'button.users.user.create', '*', '*', '*', '按钮权限：创建用户', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.users.user.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新用户', 'button.users.user.update', 'click', 'button', 'button.users.user.update', '*', '*', '*', '按钮权限：更新用户', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.users.user.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除用户', 'button.users.user.delete', 'click', 'button', 'button.users.user.delete', '*', '*', '*', '按钮权限：删除用户', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.users.user.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '启停用户', 'button.users.user.toggle_status', 'click', 'button', 'button.users.user.toggle_status', '*', '*', '*', '按钮权限：启停用户', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.users.user.toggle_status');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '重置用户密码', 'button.users.user.reset_password', 'click', 'button', 'button.users.user.reset_password', '*', '*', '*', '按钮权限：重置用户密码', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.users.user.reset_password');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '绑定用户角色', 'button.users.user.bind_roles', 'click', 'button', 'button.users.user.bind_roles', '*', '*', '*', '按钮权限：绑定用户角色', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.users.user.bind_roles');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '绑定用户部门', 'button.users.user.bind_departments', 'click', 'button', 'button.users.user.bind_departments', '*', '*', '*', '按钮权限：绑定用户部门', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.users.user.bind_departments');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建部门', 'button.users.department.create', 'click', 'button', 'button.users.department.create', '*', '*', '*', '按钮权限：创建部门', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.users.department.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新部门', 'button.users.department.update', 'click', 'button', 'button.users.department.update', '*', '*', '*', '按钮权限：更新部门', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.users.department.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除部门', 'button.users.department.delete', 'click', 'button', 'button.users.department.delete', '*', '*', '*', '按钮权限：删除部门', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.users.department.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '绑定部门成员', 'button.users.department.bind_members', 'click', 'button', 'button.users.department.bind_members', '*', '*', '*', '按钮权限：绑定部门成员', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.users.department.bind_members');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建资源分类', 'button.cmdb.category.create', 'click', 'button', 'button.cmdb.category.create', '*', '*', '*', '按钮权限：创建资源分类', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.cmdb.category.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新资源分类', 'button.cmdb.category.update', 'click', 'button', 'button.cmdb.category.update', '*', '*', '*', '按钮权限：更新资源分类', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.cmdb.category.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除资源分类', 'button.cmdb.category.delete', 'click', 'button', 'button.cmdb.category.delete', '*', '*', '*', '按钮权限：删除资源分类', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.cmdb.category.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建资源', 'button.cmdb.resource.create', 'click', 'button', 'button.cmdb.resource.create', '*', '*', '*', '按钮权限：创建资源', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.cmdb.resource.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新资源', 'button.cmdb.resource.update', 'click', 'button', 'button.cmdb.resource.update', '*', '*', '*', '按钮权限：更新资源', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.cmdb.resource.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除资源', 'button.cmdb.resource.delete', 'click', 'button', 'button.cmdb.resource.delete', '*', '*', '*', '按钮权限：删除资源', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.cmdb.resource.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '绑定资源标签', 'button.cmdb.resource.bind_tags', 'click', 'button', 'button.cmdb.resource.bind_tags', '*', '*', '*', '按钮权限：绑定资源标签', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.cmdb.resource.bind_tags');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建标签', 'button.cmdb.tag.create', 'click', 'button', 'button.cmdb.tag.create', '*', '*', '*', '按钮权限：创建标签', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.cmdb.tag.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新标签', 'button.cmdb.tag.update', 'click', 'button', 'button.cmdb.tag.update', '*', '*', '*', '按钮权限：更新标签', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.cmdb.tag.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除标签', 'button.cmdb.tag.delete', 'click', 'button', 'button.cmdb.tag.delete', '*', '*', '*', '按钮权限：删除标签', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.cmdb.tag.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建任务', 'button.tasks.task.create', 'click', 'button', 'button.tasks.task.create', '*', '*', '*', '按钮权限：创建任务', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.tasks.task.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新任务', 'button.tasks.task.update', 'click', 'button', 'button.tasks.task.update', '*', '*', '*', '按钮权限：更新任务', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.tasks.task.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除任务', 'button.tasks.task.delete', 'click', 'button', 'button.tasks.task.delete', '*', '*', '*', '按钮权限：删除任务', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.tasks.task.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '执行任务', 'button.tasks.task.execute', 'click', 'button', 'button.tasks.task.execute', '*', '*', '*', '按钮权限：执行任务', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.tasks.task.execute');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建Playbook', 'button.tasks.playbook.create', 'click', 'button', 'button.tasks.playbook.create', '*', '*', '*', '按钮权限：创建Playbook', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.tasks.playbook.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新Playbook', 'button.tasks.playbook.update', 'click', 'button', 'button.tasks.playbook.update', '*', '*', '*', '按钮权限：更新Playbook', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.tasks.playbook.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除Playbook', 'button.tasks.playbook.delete', 'click', 'button', 'button.tasks.playbook.delete', '*', '*', '*', '按钮权限：删除Playbook', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.tasks.playbook.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建站内消息', 'button.messages.message.create', 'click', 'button', 'button.messages.message.create', '*', '*', '*', '按钮权限：创建站内消息', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.messages.message.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '标记消息已读', 'button.messages.message.mark_read', 'click', 'button', 'button.messages.message.mark_read', '*', '*', '*', '按钮权限：标记消息已读', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.messages.message.mark_read');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建云账号', 'button.cloud.account.create', 'click', 'button', 'button.cloud.account.create', '*', '*', '*', '按钮权限：创建云账号', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.cloud.account.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新云账号', 'button.cloud.account.update', 'click', 'button', 'button.cloud.account.update', '*', '*', '*', '按钮权限：更新云账号', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.cloud.account.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除云账号', 'button.cloud.account.delete', 'click', 'button', 'button.cloud.account.delete', '*', '*', '*', '按钮权限：删除云账号', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.cloud.account.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '校验云账号', 'button.cloud.account.verify', 'click', 'button', 'button.cloud.account.verify', '*', '*', '*', '按钮权限：校验云账号', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.cloud.account.verify');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '同步云资源', 'button.cloud.account.sync', 'click', 'button', 'button.cloud.account.sync', '*', '*', '*', '按钮权限：同步云资源', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.cloud.account.sync');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建工单', 'button.tickets.ticket.create', 'click', 'button', 'button.tickets.ticket.create', '*', '*', '*', '按钮权限：创建工单', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.tickets.ticket.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新工单', 'button.tickets.ticket.update', 'click', 'button', 'button.tickets.ticket.update', '*', '*', '*', '按钮权限：更新工单', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.tickets.ticket.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除工单', 'button.tickets.ticket.delete', 'click', 'button', 'button.tickets.ticket.delete', '*', '*', '*', '按钮权限：删除工单', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.tickets.ticket.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '审批工单', 'button.tickets.ticket.approve', 'click', 'button', 'button.tickets.ticket.approve', '*', '*', '*', '按钮权限：审批工单', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.tickets.ticket.approve');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '流转工单', 'button.tickets.ticket.transition', 'click', 'button', 'button.tickets.ticket.transition', '*', '*', '*', '按钮权限：流转工单', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.tickets.ticket.transition');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建Docker主机', 'button.docker.host.create', 'click', 'button', 'button.docker.host.create', '*', '*', '*', '按钮权限：创建Docker主机', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.docker.host.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新Docker主机', 'button.docker.host.update', 'click', 'button', 'button.docker.host.update', '*', '*', '*', '按钮权限：更新Docker主机', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.docker.host.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除Docker主机', 'button.docker.host.delete', 'click', 'button', 'button.docker.host.delete', '*', '*', '*', '按钮权限：删除Docker主机', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.docker.host.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建Compose栈', 'button.docker.compose_stack.create', 'click', 'button', 'button.docker.compose_stack.create', '*', '*', '*', '按钮权限：创建Compose栈', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.docker.compose_stack.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新Compose栈', 'button.docker.compose_stack.update', 'click', 'button', 'button.docker.compose_stack.update', '*', '*', '*', '按钮权限：更新Compose栈', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.docker.compose_stack.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除Compose栈', 'button.docker.compose_stack.delete', 'click', 'button', 'button.docker.compose_stack.delete', '*', '*', '*', '按钮权限：删除Compose栈', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.docker.compose_stack.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '执行Docker动作', 'button.docker.action.run', 'click', 'button', 'button.docker.action.run', '*', '*', '*', '按钮权限：执行Docker动作', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.docker.action.run');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建中间件实例', 'button.middleware.instance.create', 'click', 'button', 'button.middleware.instance.create', '*', '*', '*', '按钮权限：创建中间件实例', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.middleware.instance.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新中间件实例', 'button.middleware.instance.update', 'click', 'button', 'button.middleware.instance.update', '*', '*', '*', '按钮权限：更新中间件实例', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.middleware.instance.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除中间件实例', 'button.middleware.instance.delete', 'click', 'button', 'button.middleware.instance.delete', '*', '*', '*', '按钮权限：删除中间件实例', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.middleware.instance.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '检查中间件实例', 'button.middleware.instance.check', 'click', 'button', 'button.middleware.instance.check', '*', '*', '*', '按钮权限：检查中间件实例', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.middleware.instance.check');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '执行中间件动作', 'button.middleware.instance.action', 'click', 'button', 'button.middleware.instance.action', '*', '*', '*', '按钮权限：执行中间件动作', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.middleware.instance.action');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建可观测数据源', 'button.observability.source.create', 'click', 'button', 'button.observability.source.create', '*', '*', '*', '按钮权限：创建可观测数据源', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.observability.source.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新可观测数据源', 'button.observability.source.update', 'click', 'button', 'button.observability.source.update', '*', '*', '*', '按钮权限：更新可观测数据源', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.observability.source.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除可观测数据源', 'button.observability.source.delete', 'click', 'button', 'button.observability.source.delete', '*', '*', '*', '按钮权限：删除可观测数据源', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.observability.source.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询指标', 'button.observability.source.query_metrics', 'click', 'button', 'button.observability.source.query_metrics', '*', '*', '*', '按钮权限：查询指标', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.observability.source.query_metrics');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建K8s集群', 'button.kubernetes.cluster.create', 'click', 'button', 'button.kubernetes.cluster.create', '*', '*', '*', '按钮权限：创建K8s集群', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.kubernetes.cluster.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新K8s集群', 'button.kubernetes.cluster.update', 'click', 'button', 'button.kubernetes.cluster.update', '*', '*', '*', '按钮权限：更新K8s集群', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.kubernetes.cluster.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除K8s集群', 'button.kubernetes.cluster.delete', 'click', 'button', 'button.kubernetes.cluster.delete', '*', '*', '*', '按钮权限：删除K8s集群', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.kubernetes.cluster.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '执行K8s资源动作', 'button.kubernetes.resource.action', 'click', 'button', 'button.kubernetes.resource.action', '*', '*', '*', '按钮权限：执行K8s资源动作', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.kubernetes.resource.action');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建事件', 'button.events.event.create', 'click', 'button', 'button.events.event.create', '*', '*', '*', '按钮权限：创建事件', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.events.event.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新事件', 'button.events.event.update', 'click', 'button', 'button.events.event.update', '*', '*', '*', '按钮权限：更新事件', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.events.event.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除事件', 'button.events.event.delete', 'click', 'button', 'button.events.event.delete', '*', '*', '*', '按钮权限：删除事件', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.events.event.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '关联事件', 'button.events.event.link', 'click', 'button', 'button.events.event.link', '*', '*', '*', '按钮权限：关联事件', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.events.event.link');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建工具', 'button.tool_market.tool.create', 'click', 'button', 'button.tool_market.tool.create', '*', '*', '*', '按钮权限：创建工具', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.tool_market.tool.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新工具', 'button.tool_market.tool.update', 'click', 'button', 'button.tool_market.tool.update', '*', '*', '*', '按钮权限：更新工具', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.tool_market.tool.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除工具', 'button.tool_market.tool.delete', 'click', 'button', 'button.tool_market.tool.delete', '*', '*', '*', '按钮权限：删除工具', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.tool_market.tool.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '执行工具', 'button.tool_market.tool.execute', 'click', 'button', 'button.tool_market.tool.execute', '*', '*', '*', '按钮权限：执行工具', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.tool_market.tool.execute');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建AIOps智能体', 'button.aiops.agent.create', 'click', 'button', 'button.aiops.agent.create', '*', '*', '*', '按钮权限：创建AIOps智能体', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.aiops.agent.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新AIOps智能体', 'button.aiops.agent.update', 'click', 'button', 'button.aiops.agent.update', '*', '*', '*', '按钮权限：更新AIOps智能体', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.aiops.agent.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除AIOps智能体', 'button.aiops.agent.delete', 'click', 'button', 'button.aiops.agent.delete', '*', '*', '*', '按钮权限：删除AIOps智能体', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.aiops.agent.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建AIOps模型', 'button.aiops.model.create', 'click', 'button', 'button.aiops.model.create', '*', '*', '*', '按钮权限：创建AIOps模型', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.aiops.model.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新AIOps模型', 'button.aiops.model.update', 'click', 'button', 'button.aiops.model.update', '*', '*', '*', '按钮权限：更新AIOps模型', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.aiops.model.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除AIOps模型', 'button.aiops.model.delete', 'click', 'button', 'button.aiops.model.delete', '*', '*', '*', '按钮权限：删除AIOps模型', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.aiops.model.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '发送AIOps对话', 'button.aiops.chat.send', 'click', 'button', 'button.aiops.chat.send', '*', '*', '*', '按钮权限：发送AIOps对话', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.aiops.chat.send');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '执行AIOps根因分析', 'button.aiops.rca.execute', 'click', 'button', 'button.aiops.rca.execute', '*', '*', '*', '按钮权限：执行AIOps根因分析', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.aiops.rca.execute');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '解析AIOps采购意图', 'button.aiops.procurement.intent_parse', 'click', 'button', 'button.aiops.procurement.intent_parse', '*', '*', '*', '按钮权限：解析AIOps采购意图', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.aiops.procurement.intent_parse');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '生成AIOps采购计划', 'button.aiops.procurement.plan_create', 'click', 'button', 'button.aiops.procurement.plan_create', '*', '*', '*', '按钮权限：生成AIOps采购计划', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.aiops.procurement.plan_create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '执行AIOps采购计划', 'button.aiops.procurement.execute', 'click', 'button', 'button.aiops.procurement.execute', '*', '*', '*', '按钮权限：执行AIOps采购计划', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'button' AND key = 'button.aiops.procurement.execute');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询用户列表', '/api/v1/users', 'GET', 'api', 'api.users.user.list', '*', '*', '*', 'API权限：查询用户列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.users.user.list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建用户', '/api/v1/users', 'POST', 'api', 'api.users.user.create', '*', '*', '*', 'API权限：创建用户', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.users.user.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询用户详情', '/api/v1/users/:id', 'GET', 'api', 'api.users.user.get', '*', '*', '*', 'API权限：查询用户详情', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.users.user.get');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新用户', '/api/v1/users/:id', 'PUT', 'api', 'api.users.user.update', '*', '*', '*', 'API权限：更新用户', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.users.user.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除用户', '/api/v1/users/:id', 'DELETE', 'api', 'api.users.user.delete', '*', '*', '*', 'API权限：删除用户', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.users.user.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新用户状态', '/api/v1/users/:id/status', 'PATCH', 'api', 'api.users.user.status_patch', '*', '*', '*', 'API权限：更新用户状态', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.users.user.status_patch');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '重置用户密码', '/api/v1/users/:id/reset-password', 'POST', 'api', 'api.users.user.password_reset', '*', '*', '*', 'API权限：重置用户密码', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.users.user.password_reset');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询用户角色绑定', '/api/v1/users/:id/roles', 'GET', 'api', 'api.users.user.roles_get', '*', '*', '*', 'API权限：查询用户角色绑定', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.users.user.roles_get');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '绑定用户角色', '/api/v1/users/:id/roles', 'POST', 'api', 'api.users.user.roles_bind', '*', '*', '*', 'API权限：绑定用户角色', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.users.user.roles_bind');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询用户部门绑定', '/api/v1/users/:id/departments', 'GET', 'api', 'api.users.user.departments_get', '*', '*', '*', 'API权限：查询用户部门绑定', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.users.user.departments_get');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '绑定用户部门', '/api/v1/users/:id/departments', 'POST', 'api', 'api.users.user.departments_bind', '*', '*', '*', 'API权限：绑定用户部门', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.users.user.departments_bind');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询部门列表', '/api/v1/departments', 'GET', 'api', 'api.users.department.list', '*', '*', '*', 'API权限：查询部门列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.users.department.list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建部门', '/api/v1/departments', 'POST', 'api', 'api.users.department.create', '*', '*', '*', 'API权限：创建部门', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.users.department.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询部门树', '/api/v1/departments/tree', 'GET', 'api', 'api.users.department.tree', '*', '*', '*', 'API权限：查询部门树', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.users.department.tree');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询部门详情', '/api/v1/departments/:id', 'GET', 'api', 'api.users.department.get', '*', '*', '*', 'API权限：查询部门详情', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.users.department.get');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新部门', '/api/v1/departments/:id', 'PUT', 'api', 'api.users.department.update', '*', '*', '*', 'API权限：更新部门', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.users.department.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除部门', '/api/v1/departments/:id', 'DELETE', 'api', 'api.users.department.delete', '*', '*', '*', 'API权限：删除部门', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.users.department.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询部门成员绑定', '/api/v1/departments/:id/users', 'GET', 'api', 'api.users.department.members_get', '*', '*', '*', 'API权限：查询部门成员绑定', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.users.department.members_get');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '绑定部门成员', '/api/v1/departments/:id/users', 'POST', 'api', 'api.users.department.members_bind', '*', '*', '*', 'API权限：绑定部门成员', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.users.department.members_bind');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询角色列表', '/api/v1/roles', 'GET', 'api', 'api.rbac.role.list', '*', '*', '*', 'API权限：查询角色列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.rbac.role.list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建角色', '/api/v1/roles', 'POST', 'api', 'api.rbac.role.create', '*', '*', '*', 'API权限：创建角色', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.rbac.role.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询角色详情', '/api/v1/roles/:id', 'GET', 'api', 'api.rbac.role.get', '*', '*', '*', 'API权限：查询角色详情', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.rbac.role.get');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新角色', '/api/v1/roles/:id', 'PUT', 'api', 'api.rbac.role.update', '*', '*', '*', 'API权限：更新角色', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.rbac.role.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除角色', '/api/v1/roles/:id', 'DELETE', 'api', 'api.rbac.role.delete', '*', '*', '*', 'API权限：删除角色', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.rbac.role.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询角色权限', '/api/v1/roles/:id/permissions', 'GET', 'api', 'api.rbac.role.permissions_list', '*', '*', '*', 'API权限：查询角色权限', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.rbac.role.permissions_list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '绑定角色权限', '/api/v1/roles/:id/permissions', 'POST', 'api', 'api.rbac.role.permissions_bind', '*', '*', '*', 'API权限：绑定角色权限', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.rbac.role.permissions_bind');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询权限列表', '/api/v1/permissions', 'GET', 'api', 'api.rbac.permission.list', '*', '*', '*', 'API权限：查询权限列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.rbac.permission.list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建权限', '/api/v1/permissions', 'POST', 'api', 'api.rbac.permission.create', '*', '*', '*', 'API权限：创建权限', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.rbac.permission.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询权限详情', '/api/v1/permissions/:id', 'GET', 'api', 'api.rbac.permission.get', '*', '*', '*', 'API权限：查询权限详情', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.rbac.permission.get');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新权限', '/api/v1/permissions/:id', 'PUT', 'api', 'api.rbac.permission.update', '*', '*', '*', 'API权限：更新权限', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.rbac.permission.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除权限', '/api/v1/permissions/:id', 'DELETE', 'api', 'api.rbac.permission.delete', '*', '*', '*', 'API权限：删除权限', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.rbac.permission.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询资源分类列表', '/api/v1/cmdb/categories', 'GET', 'api', 'api.cmdb.category.list', '*', '*', '*', 'API权限：查询资源分类列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.cmdb.category.list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建资源分类', '/api/v1/cmdb/categories', 'POST', 'api', 'api.cmdb.category.create', '*', '*', '*', 'API权限：创建资源分类', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.cmdb.category.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询资源分类详情', '/api/v1/cmdb/categories/:id', 'GET', 'api', 'api.cmdb.category.get', '*', '*', '*', 'API权限：查询资源分类详情', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.cmdb.category.get');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新资源分类', '/api/v1/cmdb/categories/:id', 'PUT', 'api', 'api.cmdb.category.update', '*', '*', '*', 'API权限：更新资源分类', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.cmdb.category.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除资源分类', '/api/v1/cmdb/categories/:id', 'DELETE', 'api', 'api.cmdb.category.delete', '*', '*', '*', 'API权限：删除资源分类', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.cmdb.category.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询资源列表', '/api/v1/cmdb/resources', 'GET', 'api', 'api.cmdb.resource.list', '*', '*', '*', 'API权限：查询资源列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.cmdb.resource.list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建资源', '/api/v1/cmdb/resources', 'POST', 'api', 'api.cmdb.resource.create', '*', '*', '*', 'API权限：创建资源', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.cmdb.resource.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询资源详情', '/api/v1/cmdb/resources/:id', 'GET', 'api', 'api.cmdb.resource.get', '*', '*', '*', 'API权限：查询资源详情', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.cmdb.resource.get');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新资源', '/api/v1/cmdb/resources/:id', 'PUT', 'api', 'api.cmdb.resource.update', '*', '*', '*', 'API权限：更新资源', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.cmdb.resource.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除资源', '/api/v1/cmdb/resources/:id', 'DELETE', 'api', 'api.cmdb.resource.delete', '*', '*', '*', 'API权限：删除资源', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.cmdb.resource.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '绑定资源标签', '/api/v1/cmdb/resources/:id/tags', 'POST', 'api', 'api.cmdb.resource.tags_bind', '*', '*', '*', 'API权限：绑定资源标签', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.cmdb.resource.tags_bind');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询标签列表', '/api/v1/cmdb/tags', 'GET', 'api', 'api.cmdb.tag.list', '*', '*', '*', 'API权限：查询标签列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.cmdb.tag.list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建标签', '/api/v1/cmdb/tags', 'POST', 'api', 'api.cmdb.tag.create', '*', '*', '*', 'API权限：创建标签', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.cmdb.tag.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询标签详情', '/api/v1/cmdb/tags/:id', 'GET', 'api', 'api.cmdb.tag.get', '*', '*', '*', 'API权限：查询标签详情', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.cmdb.tag.get');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新标签', '/api/v1/cmdb/tags/:id', 'PUT', 'api', 'api.cmdb.tag.update', '*', '*', '*', 'API权限：更新标签', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.cmdb.tag.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除标签', '/api/v1/cmdb/tags/:id', 'DELETE', 'api', 'api.cmdb.tag.delete', '*', '*', '*', 'API权限：删除标签', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.cmdb.tag.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询任务列表', '/api/v1/tasks', 'GET', 'api', 'api.tasks.task.list', '*', '*', '*', 'API权限：查询任务列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tasks.task.list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建任务', '/api/v1/tasks', 'POST', 'api', 'api.tasks.task.create', '*', '*', '*', 'API权限：创建任务', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tasks.task.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询任务详情', '/api/v1/tasks/:id', 'GET', 'api', 'api.tasks.task.get', '*', '*', '*', 'API权限：查询任务详情', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tasks.task.get');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新任务', '/api/v1/tasks/:id', 'PUT', 'api', 'api.tasks.task.update', '*', '*', '*', 'API权限：更新任务', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tasks.task.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除任务', '/api/v1/tasks/:id', 'DELETE', 'api', 'api.tasks.task.delete', '*', '*', '*', 'API权限：删除任务', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tasks.task.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '执行任务', '/api/v1/tasks/:id/execute', 'POST', 'api', 'api.tasks.task.execute', '*', '*', '*', 'API权限：执行任务', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tasks.task.execute');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询Playbook列表', '/api/v1/playbooks', 'GET', 'api', 'api.tasks.playbook.list', '*', '*', '*', 'API权限：查询Playbook列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tasks.playbook.list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建Playbook', '/api/v1/playbooks', 'POST', 'api', 'api.tasks.playbook.create', '*', '*', '*', 'API权限：创建Playbook', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tasks.playbook.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询Playbook详情', '/api/v1/playbooks/:id', 'GET', 'api', 'api.tasks.playbook.get', '*', '*', '*', 'API权限：查询Playbook详情', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tasks.playbook.get');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新Playbook', '/api/v1/playbooks/:id', 'PUT', 'api', 'api.tasks.playbook.update', '*', '*', '*', 'API权限：更新Playbook', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tasks.playbook.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除Playbook', '/api/v1/playbooks/:id', 'DELETE', 'api', 'api.tasks.playbook.delete', '*', '*', '*', 'API权限：删除Playbook', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tasks.playbook.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询任务执行日志列表', '/api/v1/task-logs', 'GET', 'api', 'api.tasks.log.list', '*', '*', '*', 'API权限：查询任务执行日志列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tasks.log.list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询任务执行日志详情', '/api/v1/task-logs/:id', 'GET', 'api', 'api.tasks.log.get', '*', '*', '*', 'API权限：查询任务执行日志详情', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tasks.log.get');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询站内消息列表', '/api/v1/messages', 'GET', 'api', 'api.messages.message.list', '*', '*', '*', 'API权限：查询站内消息列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.messages.message.list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建站内消息', '/api/v1/messages', 'POST', 'api', 'api.messages.message.create', '*', '*', '*', 'API权限：创建站内消息', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.messages.message.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '标记消息已读', '/api/v1/messages/:id/read', 'POST', 'api', 'api.messages.message.read', '*', '*', '*', 'API权限：标记消息已读', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.messages.message.read');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询云账号列表', '/api/v1/cloud/accounts', 'GET', 'api', 'api.cloud.account.list', '*', '*', '*', 'API权限：查询云账号列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.cloud.account.list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建云账号', '/api/v1/cloud/accounts', 'POST', 'api', 'api.cloud.account.create', '*', '*', '*', 'API权限：创建云账号', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.cloud.account.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询云账号详情', '/api/v1/cloud/accounts/:id', 'GET', 'api', 'api.cloud.account.get', '*', '*', '*', 'API权限：查询云账号详情', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.cloud.account.get');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新云账号', '/api/v1/cloud/accounts/:id', 'PUT', 'api', 'api.cloud.account.update', '*', '*', '*', 'API权限：更新云账号', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.cloud.account.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除云账号', '/api/v1/cloud/accounts/:id', 'DELETE', 'api', 'api.cloud.account.delete', '*', '*', '*', 'API权限：删除云账号', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.cloud.account.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '校验云账号', '/api/v1/cloud/accounts/:id/verify', 'POST', 'api', 'api.cloud.account.verify', '*', '*', '*', 'API权限：校验云账号', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.cloud.account.verify');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '同步云资源', '/api/v1/cloud/accounts/:id/sync', 'POST', 'api', 'api.cloud.account.sync', '*', '*', '*', 'API权限：同步云资源', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.cloud.account.sync');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询工单列表', '/api/v1/tickets', 'GET', 'api', 'api.tickets.ticket.list', '*', '*', '*', 'API权限：查询工单列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tickets.ticket.list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建工单', '/api/v1/tickets', 'POST', 'api', 'api.tickets.ticket.create', '*', '*', '*', 'API权限：创建工单', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tickets.ticket.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询工单详情', '/api/v1/tickets/:id', 'GET', 'api', 'api.tickets.ticket.get', '*', '*', '*', 'API权限：查询工单详情', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tickets.ticket.get');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新工单', '/api/v1/tickets/:id', 'PUT', 'api', 'api.tickets.ticket.update', '*', '*', '*', 'API权限：更新工单', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tickets.ticket.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除工单', '/api/v1/tickets/:id', 'DELETE', 'api', 'api.tickets.ticket.delete', '*', '*', '*', 'API权限：删除工单', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tickets.ticket.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '审批工单', '/api/v1/tickets/:id/approve', 'POST', 'api', 'api.tickets.ticket.approve', '*', '*', '*', 'API权限：审批工单', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tickets.ticket.approve');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '流转工单', '/api/v1/tickets/:id/transition', 'POST', 'api', 'api.tickets.ticket.transition', '*', '*', '*', 'API权限：流转工单', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tickets.ticket.transition');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询Docker主机列表', '/api/v1/docker/hosts', 'GET', 'api', 'api.docker.host.list', '*', '*', '*', 'API权限：查询Docker主机列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.docker.host.list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建Docker主机', '/api/v1/docker/hosts', 'POST', 'api', 'api.docker.host.create', '*', '*', '*', 'API权限：创建Docker主机', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.docker.host.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询Docker主机详情', '/api/v1/docker/hosts/:id', 'GET', 'api', 'api.docker.host.get', '*', '*', '*', 'API权限：查询Docker主机详情', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.docker.host.get');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新Docker主机', '/api/v1/docker/hosts/:id', 'PUT', 'api', 'api.docker.host.update', '*', '*', '*', 'API权限：更新Docker主机', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.docker.host.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除Docker主机', '/api/v1/docker/hosts/:id', 'DELETE', 'api', 'api.docker.host.delete', '*', '*', '*', 'API权限：删除Docker主机', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.docker.host.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询Compose栈列表', '/api/v1/docker/compose/stacks', 'GET', 'api', 'api.docker.compose_stack.list', '*', '*', '*', 'API权限：查询Compose栈列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.docker.compose_stack.list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建Compose栈', '/api/v1/docker/compose/stacks', 'POST', 'api', 'api.docker.compose_stack.create', '*', '*', '*', 'API权限：创建Compose栈', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.docker.compose_stack.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新Compose栈', '/api/v1/docker/compose/stacks/:id', 'PUT', 'api', 'api.docker.compose_stack.update', '*', '*', '*', 'API权限：更新Compose栈', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.docker.compose_stack.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除Compose栈', '/api/v1/docker/compose/stacks/:id', 'DELETE', 'api', 'api.docker.compose_stack.delete', '*', '*', '*', 'API权限：删除Compose栈', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.docker.compose_stack.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '执行Docker动作', '/api/v1/docker/actions', 'POST', 'api', 'api.docker.action.run', '*', '*', '*', 'API权限：执行Docker动作', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.docker.action.run');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询中间件实例列表', '/api/v1/middleware/instances', 'GET', 'api', 'api.middleware.instance.list', '*', '*', '*', 'API权限：查询中间件实例列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.middleware.instance.list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建中间件实例', '/api/v1/middleware/instances', 'POST', 'api', 'api.middleware.instance.create', '*', '*', '*', 'API权限：创建中间件实例', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.middleware.instance.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询中间件实例详情', '/api/v1/middleware/instances/:id', 'GET', 'api', 'api.middleware.instance.get', '*', '*', '*', 'API权限：查询中间件实例详情', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.middleware.instance.get');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新中间件实例', '/api/v1/middleware/instances/:id', 'PUT', 'api', 'api.middleware.instance.update', '*', '*', '*', 'API权限：更新中间件实例', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.middleware.instance.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除中间件实例', '/api/v1/middleware/instances/:id', 'DELETE', 'api', 'api.middleware.instance.delete', '*', '*', '*', 'API权限：删除中间件实例', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.middleware.instance.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '检查中间件实例', '/api/v1/middleware/instances/:id/check', 'POST', 'api', 'api.middleware.instance.check', '*', '*', '*', 'API权限：检查中间件实例', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.middleware.instance.check');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '执行中间件动作', '/api/v1/middleware/instances/:id/action', 'POST', 'api', 'api.middleware.instance.action', '*', '*', '*', 'API权限：执行中间件动作', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.middleware.instance.action');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询可观测数据源列表', '/api/v1/observability/sources', 'GET', 'api', 'api.observability.source.list', '*', '*', '*', 'API权限：查询可观测数据源列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.observability.source.list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建可观测数据源', '/api/v1/observability/sources', 'POST', 'api', 'api.observability.source.create', '*', '*', '*', 'API权限：创建可观测数据源', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.observability.source.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询可观测数据源详情', '/api/v1/observability/sources/:id', 'GET', 'api', 'api.observability.source.get', '*', '*', '*', 'API权限：查询可观测数据源详情', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.observability.source.get');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新可观测数据源', '/api/v1/observability/sources/:id', 'PUT', 'api', 'api.observability.source.update', '*', '*', '*', 'API权限：更新可观测数据源', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.observability.source.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除可观测数据源', '/api/v1/observability/sources/:id', 'DELETE', 'api', 'api.observability.source.delete', '*', '*', '*', 'API权限：删除可观测数据源', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.observability.source.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询可观测指标', '/api/v1/observability/metrics/query', 'GET', 'api', 'api.observability.metrics.query', '*', '*', '*', 'API权限：查询可观测指标', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.observability.metrics.query');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询K8s集群列表', '/api/v1/kubernetes/clusters', 'GET', 'api', 'api.kubernetes.cluster.list', '*', '*', '*', 'API权限：查询K8s集群列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.kubernetes.cluster.list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建K8s集群', '/api/v1/kubernetes/clusters', 'POST', 'api', 'api.kubernetes.cluster.create', '*', '*', '*', 'API权限：创建K8s集群', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.kubernetes.cluster.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询K8s集群详情', '/api/v1/kubernetes/clusters/:id', 'GET', 'api', 'api.kubernetes.cluster.get', '*', '*', '*', 'API权限：查询K8s集群详情', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.kubernetes.cluster.get');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新K8s集群', '/api/v1/kubernetes/clusters/:id', 'PUT', 'api', 'api.kubernetes.cluster.update', '*', '*', '*', 'API权限：更新K8s集群', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.kubernetes.cluster.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除K8s集群', '/api/v1/kubernetes/clusters/:id', 'DELETE', 'api', 'api.kubernetes.cluster.delete', '*', '*', '*', 'API权限：删除K8s集群', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.kubernetes.cluster.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询K8s节点列表', '/api/v1/kubernetes/nodes', 'GET', 'api', 'api.kubernetes.node.list', '*', '*', '*', 'API权限：查询K8s节点列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.kubernetes.node.list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '执行K8s资源动作', '/api/v1/kubernetes/resources/action', 'POST', 'api', 'api.kubernetes.resource.action', '*', '*', '*', 'API权限：执行K8s资源动作', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.kubernetes.resource.action');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询事件列表', '/api/v1/events', 'GET', 'api', 'api.events.event.list', '*', '*', '*', 'API权限：查询事件列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.events.event.list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建事件', '/api/v1/events', 'POST', 'api', 'api.events.event.create', '*', '*', '*', 'API权限：创建事件', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.events.event.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询事件详情', '/api/v1/events/:id', 'GET', 'api', 'api.events.event.get', '*', '*', '*', 'API权限：查询事件详情', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.events.event.get');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新事件', '/api/v1/events/:id', 'PUT', 'api', 'api.events.event.update', '*', '*', '*', 'API权限：更新事件', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.events.event.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除事件', '/api/v1/events/:id', 'DELETE', 'api', 'api.events.event.delete', '*', '*', '*', 'API权限：删除事件', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.events.event.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '检索事件', '/api/v1/events/search', 'GET', 'api', 'api.events.event.search', '*', '*', '*', 'API权限：检索事件', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.events.event.search');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '关联事件', '/api/v1/events/:id/link', 'POST', 'api', 'api.events.event.link', '*', '*', '*', 'API权限：关联事件', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.events.event.link');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询工具列表', '/api/v1/tool-market/tools', 'GET', 'api', 'api.tool_market.tool.list', '*', '*', '*', 'API权限：查询工具列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tool_market.tool.list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建工具', '/api/v1/tool-market/tools', 'POST', 'api', 'api.tool_market.tool.create', '*', '*', '*', 'API权限：创建工具', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tool_market.tool.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询工具详情', '/api/v1/tool-market/tools/:id', 'GET', 'api', 'api.tool_market.tool.get', '*', '*', '*', 'API权限：查询工具详情', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tool_market.tool.get');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新工具', '/api/v1/tool-market/tools/:id', 'PUT', 'api', 'api.tool_market.tool.update', '*', '*', '*', 'API权限：更新工具', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tool_market.tool.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除工具', '/api/v1/tool-market/tools/:id', 'DELETE', 'api', 'api.tool_market.tool.delete', '*', '*', '*', 'API权限：删除工具', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tool_market.tool.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '执行工具', '/api/v1/tool-market/tools/:id/execute', 'POST', 'api', 'api.tool_market.tool.execute', '*', '*', '*', 'API权限：执行工具', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.tool_market.tool.execute');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询AIOps智能体列表', '/api/v1/aiops/agents', 'GET', 'api', 'api.aiops.agent.list', '*', '*', '*', 'API权限：查询AIOps智能体列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.aiops.agent.list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建AIOps智能体', '/api/v1/aiops/agents', 'POST', 'api', 'api.aiops.agent.create', '*', '*', '*', 'API权限：创建AIOps智能体', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.aiops.agent.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询AIOps智能体详情', '/api/v1/aiops/agents/:id', 'GET', 'api', 'api.aiops.agent.get', '*', '*', '*', 'API权限：查询AIOps智能体详情', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.aiops.agent.get');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新AIOps智能体', '/api/v1/aiops/agents/:id', 'PUT', 'api', 'api.aiops.agent.update', '*', '*', '*', 'API权限：更新AIOps智能体', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.aiops.agent.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除AIOps智能体', '/api/v1/aiops/agents/:id', 'DELETE', 'api', 'api.aiops.agent.delete', '*', '*', '*', 'API权限：删除AIOps智能体', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.aiops.agent.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询AIOps模型列表', '/api/v1/aiops/models', 'GET', 'api', 'api.aiops.model.list', '*', '*', '*', 'API权限：查询AIOps模型列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.aiops.model.list');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '创建AIOps模型', '/api/v1/aiops/models', 'POST', 'api', 'api.aiops.model.create', '*', '*', '*', 'API权限：创建AIOps模型', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.aiops.model.create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询AIOps模型详情', '/api/v1/aiops/models/:id', 'GET', 'api', 'api.aiops.model.get', '*', '*', '*', 'API权限：查询AIOps模型详情', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.aiops.model.get');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '更新AIOps模型', '/api/v1/aiops/models/:id', 'PUT', 'api', 'api.aiops.model.update', '*', '*', '*', 'API权限：更新AIOps模型', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.aiops.model.update');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '删除AIOps模型', '/api/v1/aiops/models/:id', 'DELETE', 'api', 'api.aiops.model.delete', '*', '*', '*', 'API权限：删除AIOps模型', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.aiops.model.delete');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT 'AIOps对话', '/api/v1/aiops/chat', 'POST', 'api', 'api.aiops.chat', '*', '*', '*', 'API权限：AIOps对话', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.aiops.chat');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT 'AIOps根因分析', '/api/v1/aiops/rca', 'POST', 'api', 'api.aiops.rca', '*', '*', '*', 'API权限：AIOps根因分析', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.aiops.rca');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询AIOps采购协议', '/api/v1/aiops/procurement/protocol', 'GET', 'api', 'api.aiops.procurement.protocol', '*', '*', '*', 'API权限：查询AIOps采购协议', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.aiops.procurement.protocol');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '解析AIOps采购意图', '/api/v1/aiops/procurement/intents', 'POST', 'api', 'api.aiops.procurement.intent_parse', '*', '*', '*', 'API权限：解析AIOps采购意图', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.aiops.procurement.intent_parse');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '生成AIOps采购计划', '/api/v1/aiops/procurement/plans', 'POST', 'api', 'api.aiops.procurement.plan_create', '*', '*', '*', 'API权限：生成AIOps采购计划', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.aiops.procurement.plan_create');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '执行AIOps采购计划', '/api/v1/aiops/procurement/executions', 'POST', 'api', 'api.aiops.procurement.execute', '*', '*', '*', 'API权限：执行AIOps采购计划', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.aiops.procurement.execute');

INSERT INTO permissions (name, resource, action, type, key, dept_scope, resource_tag_scope, env_scope, description, created_at, updated_at)
SELECT '查询审计日志列表', '/api/v1/audit-logs', 'GET', 'api', 'api.audit.log.list', '*', '*', '*', 'API权限：查询审计日志列表', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE type = 'api' AND key = 'api.audit.log.list');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON 1 = 1
WHERE r.name = 'admin'
  AND NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = r.id AND rp.permission_id = p.id
  );

COMMIT;
