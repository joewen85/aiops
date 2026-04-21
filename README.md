# 中小企业 DevOps 平台（SME DevOps System）

基于 `Go + Gin + Gorm + JWT + Casbin + PostgreSQL + Redis + RabbitMQ`（后端）和 `Vite + React + TypeScript`（前端）的单仓项目。

## 目录结构

- `backend`：后端服务
- `frontend`：前端管理台
- `deploy/compose`：Docker Compose 部署
- `deploy/helm`：Helm 模板
- `docs`：产品/API 文档

## 快速启动（本地）

1. 复制配置文件
   - `cp backend/.env.example backend/.env`
   - `cp frontend/.env.example frontend/.env`
2. 启动后端
   - `cd backend && go run ./cmd/server`
3. 启动前端
   - `cd frontend && npm install && npm run dev`
4. 访问地址
   - 前端：`http://localhost:5173`
   - 后端：`http://localhost:8080/api/v1`
   - 健康检查：`http://localhost:8080/healthz`

默认管理员账号：`admin / Admin@123`

## 必要配置说明（重点）

后端必须配置 `POSTGRES_DSN`，否则启动会报错：

- 报错示例：`init postgres failed: POSTGRES_DSN is required`
- `backend/.env.example` 已给出示例 DSN，可按本机数据库修改。

`RABBITMQ_URL` 可为空（当前核心 RBAC/ABAC 功能不依赖 RabbitMQ 才能启动）。

前端跨域访问后端时，请配置 CORS：

- `CORS_ALLOW_ORIGINS`：允许来源，多个用英文逗号分隔
  - 例如：`http://localhost:5173,http://127.0.0.1:5173`
- `CORS_ALLOW_CREDENTIALS`：是否允许携带凭据（默认 `false`）
- `ABAC_HEADER_SIGN_SECRET`：可选，开启后 `X-Env/X-Resource-Tag` 需携带签名头（见 `docs/PERMISSION_KEYS.md`）
- `PERMISSION_RUNTIME_CACHE_TTL_MS`：可选，运行时角色查询缓存毫秒数（默认 `0` 关闭；建议 1000-3000，在性能与撤权生效时延间权衡）

前端可选调试参数：

- `VITE_WS_DEBUG`：WebSocket 调试日志开关（`true/1/on` 开启，默认 `false`）

## 多云同步参数（腾讯云 SDK）

为支持“真实对接腾讯云 SDK 同步资产”，后端新增以下环境变量（见 `backend/.env.example`）：

- 腾讯云账号字段约定
  - `accessKey` 对应腾讯云 `SecretId`（通常以 `AKID` 开头）
  - `secretKey` 对应腾讯云 `SecretKey`

- `CLOUD_SDK_MOCK_ENABLED`
  - 用途：是否强制走模拟数据。
  - 默认：`false`
  - 建议：开发联调可设为 `true`；生产保持 `false`。
- `CLOUD_SDK_MOCK_AK_PREFIX`
  - 用途：当账号 `AccessKey` 以前缀开头时，走模拟数据。
  - 默认：`mock_`
  - 示例：`mock_test_ak_xxx`
- `CLOUD_SDK_MOCK_SK_PREFIX`
  - 用途：当账号 `SecretKey` 以前缀开头时，走模拟数据。
  - 默认：`mock_`
  - 示例：`mock_test_sk_xxx`
- `ALIYUN_DEFAULT_REGION`
  - 用途：阿里云账号未填写地域时使用的默认地域。
  - 默认：`cn-hangzhou`
  - 示例：`cn-shanghai`
- `ALIYUN_SDK_TIMEOUT_SECONDS`
  - 用途：阿里云 SDK 单次请求超时时间（秒）。
  - 默认：`10`
- `ALIYUN_SDK_PAGE_LIMIT`
  - 用途：阿里云分页接口每次拉取条数（上限 100）。
  - 默认：`100`
- `TENCENT_DEFAULT_REGION`
  - 用途：云账号未填写地域时使用的默认地域。
  - 默认：`ap-guangzhou`
  - 示例：`ap-shanghai`
- `TENCENT_SDK_TIMEOUT_SECONDS`
  - 用途：腾讯云 SDK 单次请求超时时间（秒）。
  - 默认：`10`
  - 建议：内网/跨地域网络波动时可提高到 `20-30`。
- `TENCENT_SDK_PAGE_LIMIT`
  - 用途：腾讯云分页接口每次拉取条数（上限 100）。
  - 默认：`100`
  - 建议：一般保持默认。

推荐配置：

- 开发环境（便于测试）
  - `CLOUD_SDK_MOCK_ENABLED=false`
  - `CLOUD_SDK_MOCK_AK_PREFIX=mock_`
  - `CLOUD_SDK_MOCK_SK_PREFIX=mock_`
  - 使用真实 AK/SK 即走真实 SDK；使用 `mock_` 前缀账号可快速走模拟数据。
- 生产环境
  - `CLOUD_SDK_MOCK_ENABLED=false`
  - `CLOUD_SDK_MOCK_AK_PREFIX` / `CLOUD_SDK_MOCK_SK_PREFIX` 可改为内部专用前缀或留空（避免误触发模拟）。

## 安全基线与可信能力（RBAC/ABAC）

当前版本已内置以下安全加固能力（默认启用或可配置启用）：

- **权限实时生效**：`PermissionMiddleware` 和 `/auth/me/permissions` 均按数据库实时用户角色判定，撤权后无需重新登录即可收敛权限。
- **用户禁用立即生效**：请求阶段实时校验 `is_active`，被禁用用户会被立即拒绝访问。
- **ABAC 头部安全校验**：`X-Env`、`X-Resource-Tag` 启用格式校验，非法值直接拒绝（400）。
- **ABAC 可选签名防篡改**：配置 `ABAC_HEADER_SIGN_SECRET` 后，ABAC 头需携带 `X-ABAC-Timestamp` + `X-ABAC-Signature`（HMAC-SHA256，默认 ±5 分钟时窗）。
- **敏感审计脱敏**：审计日志自动对 `password/token/secret` 等敏感字段脱敏，避免敏感信息落盘。
- **内置角色保护**：`admin` / 内置角色禁止删除，内置角色名称不可被篡改。
- **绑定接口防脏数据**：角色/部门绑定接口增加存在性校验、重复 ID 去重、非法 ID 拦截，并增加批量上限保护（每次最多 200 条）。
- **权限获取精简模式**：`/auth/me/permissions?compact=1` 返回精简权限包，降低前端鉴权拉取开销。

## 安全参数速查（后端）

以下参数用于控制 RBAC/ABAC 安全与性能行为：

- `JWT_SECRET`
  - 用途：JWT 签名密钥。
  - 建议：生产环境使用高强度随机字符串并定期轮换。
- `ABAC_HEADER_SIGN_SECRET`
  - 用途：开启 ABAC 头签名校验（防伪造/防篡改）。
  - 生效行为：当请求携带 `X-Env` 或 `X-Resource-Tag` 时，必须同时携带：
    - `X-ABAC-Timestamp`（RFC3339）
    - `X-ABAC-Signature`（`HMAC-SHA256` hex）
  - 签名原文：`METHOD + "\n" + PATH + "\n" + ENV + "\n" + TAG + "\n" + TIMESTAMP`
- `PERMISSION_RUNTIME_CACHE_TTL_MS`
  - 用途：运行时“用户角色列表”短 TTL 缓存（毫秒）。
  - 默认：`0`（关闭，角色变更实时生效）。
  - 建议：`1000-3000`（在高并发下可减轻 DB 压力，角色撤销生效会有最多 TTL 的延迟）。
- `CORS_ALLOW_ORIGINS`
  - 用途：限制允许跨域来源，避免任意站点调用后台接口。
  - 建议：生产环境使用明确域名白名单，不要使用 `*`。
- `CORS_ALLOW_CREDENTIALS`
  - 用途：是否允许浏览器携带凭据进行跨域请求。
  - 建议：仅在明确需要 Cookie/凭据跨域时开启。

## Casbin 策略模型与数据库迁移说明（已内置）

为避免启动时报错：

- 报错示例：`init casbin failed: invalid policy rule size`

当前已统一为 6 段策略模型（与 `gorm-adapter` 存储结构一致）：

- `p = sub, obj, act, dept, tag, env`

并在服务启动时自动执行策略规范化迁移（`casbin_rule` 表）：

- 对 `ptype='p'` 且 `v3/v4/v5` 为空的历史数据自动补 `*`；
- 自动保证管理员基础策略存在。

如需手工修复历史数据，可执行：

```sql
UPDATE casbin_rule SET v3='*' WHERE ptype='p' AND (v3 IS NULL OR v3='');
UPDATE casbin_rule SET v4='*' WHERE ptype='p' AND (v4 IS NULL OR v4='');
UPDATE casbin_rule SET v5='*' WHERE ptype='p' AND (v5 IS NULL OR v5='');
```

## 权限标识初始化（新增）

项目首次启动后端时会自动执行 RBAC/ABAC 权限种子初始化（幂等）：

- 自动写入 `api/menu/button` 权限标识到 `permissions` 表；
- 自动将所有权限绑定到内置 `admin` 角色（`role_permissions`）；
- 重复启动不会重复插入。

同时提供手工脚本（与代码种子同源生成）：

- `docs/permission_seed.sql`

手工执行示例（PostgreSQL）：

```bash
psql "$POSTGRES_DSN" -f docs/permission_seed.sql
```

## 一键部署

```bash
docker compose -f deploy/compose/docker-compose.yml up --build
```

## Helm 部署

- Chart 路径：`deploy/helm/devops-system`
- 示例命令：
  - `helm upgrade --install devops-system deploy/helm/devops-system`
