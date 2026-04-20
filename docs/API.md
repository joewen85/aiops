# API 设计约定

## 统一响应

- 成功：`{ "code": 0, "message": "ok", "data": ... }`
- 失败：`{ "code": <non-zero>, "message": "..." }`

## 分页响应

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [],
    "total": 0,
    "page": 1,
    "pageSize": 20
  }
}
```

## 鉴权

- Header: `Authorization: Bearer <token>`
- 中间件分层：
  - `AuthMiddleware`：解析 JWT 和基础用户态
  - `PermissionMiddleware`：RBAC + ABAC 授权
- ABAC 上下文头：
  - `X-Env`、`X-Resource-Tag`
  - 当启用 `ABAC_HEADER_SIGN_SECRET` 时，需附带 `X-ABAC-Timestamp` + `X-ABAC-Signature`（HMAC-SHA256）
- 运行时角色查询：
  - 默认每次请求实时查库（角色变更立即生效）
  - 可通过 `PERMISSION_RUNTIME_CACHE_TTL_MS` 开启短 TTL 角色缓存以降低 DB 压力
- 绑定批量限制：
  - `POST /users/:id/roles` 的 `roleIds` 最大 200 条
  - `POST /users/:id/departments` 的 `departmentIds` 最大 200 条
  - `POST /departments/:id/users` 的 `userIds` 最大 200 条

## 错误码分层

- `1000-1999` 鉴权
- `2000-2999` 权限
- `3000-3999` 参数
- `4000-4999` 业务
- `5000-5999` 系统

## 已实现模块路由（v1 runnable scaffold）

- `/api/v1/auth`
- `/api/v1/auth/me/permissions`（支持 `?compact=1`；按当前数据库实时角色计算，不依赖 JWT 内缓存角色）
- `/api/v1/users`
- `/api/v1/departments`
- `/api/v1/departments/tree`
- `/api/v1/departments/:id/users`
- `/api/v1/roles`
- `/api/v1/roles/:id/permissions`
- `/api/v1/permissions`
- `/api/v1/cmdb/*`
- `/api/v1/tasks`
- `/api/v1/task-logs`
- `/api/v1/audit-logs`
- `/api/v1/cloud/accounts`
- `/api/v1/cloud/assets`
- `/api/v1/tickets`
- `/api/v1/docker`
- `/api/v1/middleware`
- `/api/v1/observability`
- `/api/v1/kubernetes`
- `/api/v1/events`
- `/api/v1/tool-market`
- `/api/v1/aiops`
- `/ws`

## CMDB 模块接口细化（v1）

- CI 基础：`GET/POST /api/v1/cmdb/categories`、`GET/POST /api/v1/cmdb/resources`、`GET/POST /api/v1/cmdb/tags`、`POST /api/v1/cmdb/resources/:id/tags`
- 关系图谱：`GET/POST /api/v1/cmdb/relations`、`GET /api/v1/cmdb/resources/:id/upstream`、`GET /api/v1/cmdb/resources/:id/downstream`
- 影响分析：`GET /api/v1/cmdb/topology/:application`、`GET /api/v1/cmdb/impact/:ciId`、`GET /api/v1/cmdb/regions/:region/failover`、`GET /api/v1/cmdb/change-impact/:releaseId`
- 自动采集：`POST /api/v1/cmdb/sync/jobs`、`GET /api/v1/cmdb/sync/jobs/:id`、`POST /api/v1/cmdb/sync/jobs/:id/retry`
- 采集优先级：`IaC > Cloud API > K8s > APM > Manual`
- 唯一标识：云主机=`cloud:account:region:instance_id`，K8s Workload=`cloud:region:cluster:namespace:kind:name`，托管 DB=`cloud:account:region:engine:instance_id`

## 多云管理模块接口细化（v1）

- 云账号：`GET/POST /api/v1/cloud/accounts`、`GET/PUT/DELETE /api/v1/cloud/accounts/:id`
  - 账号列表过滤：`keyword/provider/region/verified`
  - `verified` 支持：`true/false/1/0/yes/no`
- 账号动作：`POST /api/v1/cloud/accounts/:id/verify`、`POST /api/v1/cloud/accounts/:id/sync`
- 账号资源：`GET /api/v1/cloud/accounts/:id/assets`（支持 `region/type` 过滤）
- 云资源：`GET/POST /api/v1/cloud/assets`、`GET/PUT/DELETE /api/v1/cloud/assets/:id`
- 资源类型（首批基础能力）：`CloudServer/MySQL/PrivateNetwork/ObjectStorage/FileStorage/ContainerService/LoadBalancer/DNS/SSLCertificate/LogService`

## AIOps 自然语言采购协议（预留）

- 协议版本：`aiops.procurement.v1alpha1`
- 协议查询：`GET /api/v1/aiops/procurement/protocol`
- 意图解析：`POST /api/v1/aiops/procurement/intents`
  - 输入：自然语言 `message` + 可选 `preferredProvider/region/quantity/budgetLimit`
  - 输出：结构化 `intent`（`action/provider/resourceType/region/quantity`）+ 待澄清项
- 计划生成：`POST /api/v1/aiops/procurement/plans`
  - 输入：`intent`
  - 输出：执行 `plan`（步骤、预算估算、审批要求）
- 执行接口：`POST /api/v1/aiops/procurement/executions`
  - 输入：`plan` + `dryRun`
  - 输出：`result`（`dry_run/accepted`）

## 初始账号

- 用户名：`admin`
- 密码：`Admin@123`
- 角色：`admin`（内置全局策略）

## 当前实现状态

- 已完成：平台骨架、统一响应、JWT + Permission 中间件、审计日志、核心模块 CRUD、WebSocket 推送链路。
- 已完成：前端壳层（主题切换、路由守卫、Axios 拦截器、移动端/折叠屏适配）。
- 已完成：RBAC/ABAC 权限模块（角色/权限 CRUD、角色权限多选绑定、`admin` 禁删、ABAC 条件 `deptScope/resourceTagScope/envScope`）。
- 已完成：前端菜单/按钮按权限 key 动态显隐（`menuKeys` / `buttonKeys`，来源 `/api/v1/auth/me/permissions`）。
- 占位实现：Docker 深度运维、K8s 真实资源操作、多云真实 SDK 调用、AIOps 生产级推理链路（当前为接口骨架 + stub）。

## 权限标识清单

- 详见：`docs/PERMISSION_KEYS.md`
- 包含：现有 API / 菜单 / 按钮权限标识、ABAC 范围约定、后续新增维护流程。
- 初始化脚本：`docs/permission_seed.sql`（同时支持后端启动自动幂等入库）。
