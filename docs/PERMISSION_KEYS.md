# RBAC/ABAC 权限标识清单（现有 + 将来新增）

本文用于统一维护平台的权限标识（`api` / `menu` / `button`），并约束后续新增功能的命名规则与落库规范。

## 1. 命名规则（必须遵守）

- `menu`：`menu.<module>`
  - 示例：`menu.rbac`、`menu.cmdb`
- `button`：`button.<module>.<action>`
  - 示例：`button.rbac.role.create`
- `api`：`api.<module>.<resource>.<action>`
  - 示例：`api.users.user.list`、`api.tasks.task.execute`

建议动作词固定为：`list/get/create/update/delete/patch/bind/execute/approve/transition/verify/sync/query/search/link/read/check/action/chat/rca`。

## 2. ABAC 字段约定（与权限表一致）

- `deptScope`：部门范围（默认 `*`）
- `resourceTagScope`：资源标签范围（默认 `*`）
- `envScope`：环境范围（默认 `*`，可取 `prod/staging/dev`）

请求头约定：

- `X-Resource-Tag` -> ABAC `resourceTagScope` 判定
- `X-Env` -> ABAC `envScope` 判定
- `deptScope` 由 JWT 内的 `deptId` 参与判定
- 可选签名增强（推荐生产开启）：
  - 当后端配置 `ABAC_HEADER_SIGN_SECRET` 后，若携带 `X-Resource-Tag` 或 `X-Env`，必须同时携带：
    - `X-ABAC-Timestamp`（RFC3339）
    - `X-ABAC-Signature`（`HMAC-SHA256` hex）
  - 签名原文：`METHOD + "\\n" + PATH + "\\n" + ENV + "\\n" + TAG + "\\n" + TIMESTAMP`
- 运行时角色缓存（可选）：
  - `PERMISSION_RUNTIME_CACHE_TTL_MS` > 0 时，权限中间件会对“用户角色列表”做短 TTL 缓存（默认关闭）
  - `isActive` 状态仍每次实时查询，用户禁用可立即生效
  - 角色撤销生效时间受 TTL 影响（最多延迟约 TTL）

## 3. 现有菜单权限标识（已落地）

- `menu.dashboard`：概览
- `menu.rbac`：RBAC/ABAC 权限管理
- `menu.users`：用户与部门
- `menu.cmdb`：CMDB
- `menu.tasks`：任务中心
- `menu.messages`：站内消息
- `menu.cloud`：多云管理
- `menu.tickets`：工单管理
- `menu.docker`：Docker 管理
- `menu.middleware`：中间件管理
- `menu.observability`：可观测性
- `menu.kubernetes`：Kubernetes 管理
- `menu.events`：事件中心
- `menu.tools`：工具市场
- `menu.aiops`：AIOps
- `menu.audit`：审计日志

## 4. 现有按钮权限标识

### 4.1 已落地（当前前端已使用）

- `button.rbac.role.create`：创建角色
- `button.rbac.role.detail`：查看角色详情
- `button.rbac.role.update`：更新角色
- `button.rbac.role.delete`：删除角色
- `button.rbac.permission.create`：创建权限
- `button.rbac.permission.detail`：查看权限详情
- `button.rbac.permission.update`：更新权限
- `button.rbac.permission.delete`：删除权限
- `button.rbac.binding.save`：保存角色-权限绑定

### 4.2 将来新增（建议预留）

- 用户与部门：
  - `button.users.user.create`
  - `button.users.user.update`
  - `button.users.user.delete`
  - `button.users.user.toggle_status`
  - `button.users.user.reset_password`
  - `button.users.user.bind_roles`
  - `button.users.user.bind_departments`
  - `button.users.department.create`
  - `button.users.department.update`
  - `button.users.department.delete`
  - `button.users.department.bind_members`
- CMDB：
  - `button.cmdb.category.create|update|delete`
  - `button.cmdb.resource.create|update|delete|bind_tags`
  - `button.cmdb.tag.create|update|delete`
- 任务中心：
  - `button.tasks.task.create|update|delete|execute`
  - `button.tasks.playbook.create|update|delete`
- 站内消息：
  - `button.messages.message.create|mark_read`
- 多云：
  - `button.cloud.account.create|update|delete|verify|sync`
- 工单：
  - `button.tickets.ticket.create|update|delete|approve|transition`
- Docker：
  - `button.docker.host.create|update|delete`
  - `button.docker.compose_stack.create|update|delete`
  - `button.docker.action.run`
- 中间件：
  - `button.middleware.instance.create|update|delete|check|action`
- 可观测性：
  - `button.observability.source.create|update|delete|query_metrics`
- Kubernetes：
  - `button.kubernetes.cluster.create|update|delete`
  - `button.kubernetes.resource.action`
- 事件中心：
  - `button.events.event.create|update|delete|link`
- 工具市场：
  - `button.tool_market.tool.create|update|delete|execute`
- AIOps：
  - `button.aiops.agent.create|update|delete`
  - `button.aiops.model.create|update|delete`
  - `button.aiops.chat.send`
  - `button.aiops.rca.execute`
  - `button.aiops.procurement.intent_parse|plan_create|execute`

## 5. 现有 API 权限标识（按路由全量映射）

说明：以下接口均在 `/api/v1` 下，除特别说明外默认建议 ABAC 为 `*/*/*`。

### 5.1 用户与部门

- `api.users.user.list` -> `GET /users`
- `api.users.user.create` -> `POST /users`
- `api.users.user.get` -> `GET /users/:id`
- `api.users.user.update` -> `PUT /users/:id`
- `api.users.user.delete` -> `DELETE /users/:id`
- `api.users.user.status_patch` -> `PATCH /users/:id/status`
- `api.users.user.password_reset` -> `POST /users/:id/reset-password`
- `api.users.user.roles_get` -> `GET /users/:id/roles`
- `api.users.user.roles_bind` -> `POST /users/:id/roles`
- `api.users.user.departments_get` -> `GET /users/:id/departments`
- `api.users.user.departments_bind` -> `POST /users/:id/departments`
- `api.users.department.list` -> `GET /departments`
- `api.users.department.create` -> `POST /departments`
- `api.users.department.tree` -> `GET /departments/tree`
- `api.users.department.get` -> `GET /departments/:id`
- `api.users.department.update` -> `PUT /departments/:id`
- `api.users.department.delete` -> `DELETE /departments/:id`
- `api.users.department.members_get` -> `GET /departments/:id/users`
- `api.users.department.members_bind` -> `POST /departments/:id/users`

### 5.2 RBAC

- `api.rbac.role.list` -> `GET /roles`
- `api.rbac.role.create` -> `POST /roles`
- `api.rbac.role.get` -> `GET /roles/:id`
- `api.rbac.role.update` -> `PUT /roles/:id`
- `api.rbac.role.delete` -> `DELETE /roles/:id`
- `api.rbac.role.permissions_list` -> `GET /roles/:id/permissions`
- `api.rbac.role.permissions_bind` -> `POST /roles/:id/permissions`
- `api.rbac.permission.list` -> `GET /permissions`
- `api.rbac.permission.create` -> `POST /permissions`
- `api.rbac.permission.get` -> `GET /permissions/:id`
- `api.rbac.permission.update` -> `PUT /permissions/:id`
- `api.rbac.permission.delete` -> `DELETE /permissions/:id`

### 5.3 CMDB（建议按部门/标签/环境启用 ABAC）

- `api.cmdb.category.list` -> `GET /cmdb/categories`
- `api.cmdb.category.create` -> `POST /cmdb/categories`
- `api.cmdb.category.get` -> `GET /cmdb/categories/:id`
- `api.cmdb.category.update` -> `PUT /cmdb/categories/:id`
- `api.cmdb.category.delete` -> `DELETE /cmdb/categories/:id`
- `api.cmdb.resource.list` -> `GET /cmdb/resources`
- `api.cmdb.resource.create` -> `POST /cmdb/resources`
- `api.cmdb.resource.get` -> `GET /cmdb/resources/:id`
- `api.cmdb.resource.update` -> `PUT /cmdb/resources/:id`
- `api.cmdb.resource.delete` -> `DELETE /cmdb/resources/:id`
- `api.cmdb.resource.tags_bind` -> `POST /cmdb/resources/:id/tags`
- `api.cmdb.tag.list` -> `GET /cmdb/tags`
- `api.cmdb.tag.create` -> `POST /cmdb/tags`
- `api.cmdb.tag.get` -> `GET /cmdb/tags/:id`
- `api.cmdb.tag.update` -> `PUT /cmdb/tags/:id`
- `api.cmdb.tag.delete` -> `DELETE /cmdb/tags/:id`

### 5.4 任务中心（建议至少按环境启用 ABAC）

- `api.tasks.task.list` -> `GET /tasks`
- `api.tasks.task.create` -> `POST /tasks`
- `api.tasks.task.get` -> `GET /tasks/:id`
- `api.tasks.task.update` -> `PUT /tasks/:id`
- `api.tasks.task.delete` -> `DELETE /tasks/:id`
- `api.tasks.task.execute` -> `POST /tasks/:id/execute`
- `api.tasks.playbook.list` -> `GET /playbooks`
- `api.tasks.playbook.create` -> `POST /playbooks`
- `api.tasks.playbook.get` -> `GET /playbooks/:id`
- `api.tasks.playbook.update` -> `PUT /playbooks/:id`
- `api.tasks.playbook.delete` -> `DELETE /playbooks/:id`
- `api.tasks.log.list` -> `GET /task-logs`
- `api.tasks.log.get` -> `GET /task-logs/:id`

### 5.5 站内消息

- `api.messages.message.list` -> `GET /messages`
- `api.messages.message.create` -> `POST /messages`
- `api.messages.message.read` -> `POST /messages/:id/read`

### 5.6 多云

- `api.cloud.account.list` -> `GET /cloud/accounts`
- `api.cloud.account.create` -> `POST /cloud/accounts`
- `api.cloud.account.get` -> `GET /cloud/accounts/:id`
- `api.cloud.account.update` -> `PUT /cloud/accounts/:id`
- `api.cloud.account.delete` -> `DELETE /cloud/accounts/:id`
- `api.cloud.account.verify` -> `POST /cloud/accounts/:id/verify`
- `api.cloud.account.sync` -> `POST /cloud/accounts/:id/sync`

### 5.7 工单

- `api.tickets.ticket.list` -> `GET /tickets`
- `api.tickets.ticket.create` -> `POST /tickets`
- `api.tickets.ticket.get` -> `GET /tickets/:id`
- `api.tickets.ticket.update` -> `PUT /tickets/:id`
- `api.tickets.ticket.delete` -> `DELETE /tickets/:id`
- `api.tickets.ticket.approve` -> `POST /tickets/:id/approve`
- `api.tickets.ticket.transition` -> `POST /tickets/:id/transition`

### 5.8 Docker

- `api.docker.host.list` -> `GET /docker/hosts`
- `api.docker.host.create` -> `POST /docker/hosts`
- `api.docker.host.get` -> `GET /docker/hosts/:id`
- `api.docker.host.update` -> `PUT /docker/hosts/:id`
- `api.docker.host.delete` -> `DELETE /docker/hosts/:id`
- `api.docker.compose_stack.list` -> `GET /docker/compose/stacks`
- `api.docker.compose_stack.create` -> `POST /docker/compose/stacks`
- `api.docker.compose_stack.update` -> `PUT /docker/compose/stacks/:id`
- `api.docker.compose_stack.delete` -> `DELETE /docker/compose/stacks/:id`
- `api.docker.action.run` -> `POST /docker/actions`

### 5.9 中间件

- `api.middleware.instance.list` -> `GET /middleware/instances`
- `api.middleware.instance.create` -> `POST /middleware/instances`
- `api.middleware.instance.get` -> `GET /middleware/instances/:id`
- `api.middleware.instance.update` -> `PUT /middleware/instances/:id`
- `api.middleware.instance.delete` -> `DELETE /middleware/instances/:id`
- `api.middleware.instance.check` -> `POST /middleware/instances/:id/check`
- `api.middleware.instance.action` -> `POST /middleware/instances/:id/action`

### 5.10 可观测性

- `api.observability.source.list` -> `GET /observability/sources`
- `api.observability.source.create` -> `POST /observability/sources`
- `api.observability.source.get` -> `GET /observability/sources/:id`
- `api.observability.source.update` -> `PUT /observability/sources/:id`
- `api.observability.source.delete` -> `DELETE /observability/sources/:id`
- `api.observability.metrics.query` -> `GET /observability/metrics/query`

### 5.11 Kubernetes

- `api.kubernetes.cluster.list` -> `GET /kubernetes/clusters`
- `api.kubernetes.cluster.create` -> `POST /kubernetes/clusters`
- `api.kubernetes.cluster.get` -> `GET /kubernetes/clusters/:id`
- `api.kubernetes.cluster.update` -> `PUT /kubernetes/clusters/:id`
- `api.kubernetes.cluster.delete` -> `DELETE /kubernetes/clusters/:id`
- `api.kubernetes.node.list` -> `GET /kubernetes/nodes`
- `api.kubernetes.resource.action` -> `POST /kubernetes/resources/action`

### 5.12 事件中心

- `api.events.event.list` -> `GET /events`
- `api.events.event.create` -> `POST /events`
- `api.events.event.get` -> `GET /events/:id`
- `api.events.event.update` -> `PUT /events/:id`
- `api.events.event.delete` -> `DELETE /events/:id`
- `api.events.event.search` -> `GET /events/search`
- `api.events.event.link` -> `POST /events/:id/link`

### 5.13 工具市场

- `api.tool_market.tool.list` -> `GET /tool-market/tools`
- `api.tool_market.tool.create` -> `POST /tool-market/tools`
- `api.tool_market.tool.get` -> `GET /tool-market/tools/:id`
- `api.tool_market.tool.update` -> `PUT /tool-market/tools/:id`
- `api.tool_market.tool.delete` -> `DELETE /tool-market/tools/:id`
- `api.tool_market.tool.execute` -> `POST /tool-market/tools/:id/execute`

### 5.14 AIOps

- `api.aiops.agent.list` -> `GET /aiops/agents`
- `api.aiops.agent.create` -> `POST /aiops/agents`
- `api.aiops.agent.get` -> `GET /aiops/agents/:id`
- `api.aiops.agent.update` -> `PUT /aiops/agents/:id`
- `api.aiops.agent.delete` -> `DELETE /aiops/agents/:id`
- `api.aiops.model.list` -> `GET /aiops/models`
- `api.aiops.model.create` -> `POST /aiops/models`
- `api.aiops.model.get` -> `GET /aiops/models/:id`
- `api.aiops.model.update` -> `PUT /aiops/models/:id`
- `api.aiops.model.delete` -> `DELETE /aiops/models/:id`
- `api.aiops.chat` -> `POST /aiops/chat`
- `api.aiops.rca` -> `POST /aiops/rca`
- `api.aiops.procurement.protocol` -> `GET /aiops/procurement/protocol`
- `api.aiops.procurement.intent_parse` -> `POST /aiops/procurement/intents`
- `api.aiops.procurement.plan_create` -> `POST /aiops/procurement/plans`
- `api.aiops.procurement.execute` -> `POST /aiops/procurement/executions`

### 5.15 审计

- `api.audit.log.list` -> `GET /audit-logs`

## 6. 非 RBAC API（当前不走 PermissionMiddleware）

- `POST /api/v1/auth/login`：登录接口（公开）
- `GET /api/v1/auth/me/permissions`：仅认证后访问（不走 RBAC）
- `GET /healthz`：健康检查（公开）
- `GET /ws`：WebSocket 接入点（建议后续升级为 token 鉴权）

## 7. 将来新增权限标识维护流程

新增任何功能时，必须同步完成以下步骤：

1. 在本文件登记 `menu/button/api` 标识与中文说明。
2. 在权限管理中新增对应 `permissions` 记录（`type/key/resource/action/deptScope/resourceTagScope/envScope`）。
3. 为角色绑定对应权限，回归验证 `/api/v1/auth/me/permissions` 返回结果。
4. 前端菜单与按钮使用同名 `key` 做动态显隐。
5. 如果是新模块，同时更新 `docs/API.md` 的路由清单。

## 8. 初始化入库

- 手工脚本：`docs/permission_seed.sql`
- 自动初始化：后端启动时会执行幂等种子逻辑（`SeedRBACDefaults`），首次启动即入库。
