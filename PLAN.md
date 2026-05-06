# 平台可运行骨架 + 模块逐个实现总计划（细化版）

## Summary
- 固定策略：核心闭环优先、后端可运行+前端壳层联通、AutoMigrate 起步。
- 统一规范：统一响应、JWT + Permission 分层、Axios 401 跳转、WebSocket 推送、全 API 一致性回归。
- 实施流程：先收敛平台骨架，再按模块顺序逐个落地，每个模块完成即补测试与文档。

## 全局技术约束
- 统一成功响应：`{ code, message, data }`
- 统一错误响应：`{ code, message }`
- 统一分页结构：`data = { list, total, page, pageSize }`
- 鉴权链路：`AuthMiddleware(JWT)` -> `PermissionMiddleware(RBAC+ABAC)`
- 审计范围：所有关键 `create/update/delete/execute/bind/approve/sync`
- 前端数据层：Axios 拦截器 + 全局错误提示 + 401 自动跳转登录
- 实时通信：`/ws` + Redis Pub/Sub（禁止轮询）

## 实施顺序与阶段目标
1. 平台骨架收敛（可运行）
2. Phase 1 核心闭环：1/2/3/9/12
3. Phase 2 运维扩展：4/8/11/5/6/7/10/13
4. Phase 3 AIOps：14

## 模块细化（功能点 / 数据模型 / 接口清单 / 技术要点与验收）

### 1) RBAC/ABAC 权限管理
- 功能点：权限 CRUD、角色 CRUD、角色-权限绑定、admin 禁删、菜单/按钮权限下发。
- 数据模型：`roles`、`permissions`、`role_permissions`、`casbin_rule`。
- 接口清单：`GET/POST/PUT/DELETE /roles`、`GET/POST/PUT/DELETE /permissions`、`POST /roles/:id/permissions`。
- 技术要点与验收：Casbin + PostgreSQL 持久化；ABAC 条件 `dept_id/resource_tag/env`；验收=接口权限、菜单权限、按钮权限生效且 admin 防删。

### 2) 用户与部门(组)
- 功能点：用户 CRUD、`isActive` 启停、重置密码、用户-角色绑定、用户-部门绑定、部门 CRUD 与成员绑定。
- 数据模型：`users`、`departments`、`user_roles`、`user_departments`。
- 接口清单：`/users`、`/departments`、`POST /users/:id/roles`、`POST /users/:id/departments`、`POST /users/:id/reset-password`。
- 技术要点与验收：BCrypt 存储密码；禁用用户立即失效；部门树查询；验收=登录、启停、绑定关系一致。

### 3) CMDB
- 功能点：
  - 采用“统一资源模型（CI）+关系图谱+自动采集管道+数据质量治理”四件套，核心能力从“记录资源”升级为“实时计算依赖与影响面”。
  - 统一 CI 四层模型：L1 基础设施（`CloudAccount/Region/VPC/Subnet/VM/LB/DNS/ObjectStorage`）、L2 平台（`K8sCluster/Namespace/NodePool/Ingress/Config/SecretRef`）、L3 数据与中间件（`MySQL/PostgreSQL/Redis/RabbitMQ/VOD/LiveStreaming/Kafka`）、L4 业务与交付（`Service/Application/Repo/Pipeline/Release/Team/Owner`）。
  - 关系图谱先固化 10 类关系：`deployed_on`、`runs_in`、`connects_to`、`publishes_to`、`consumes_from`、`fronted_by`、`resolves_via`、`stores_in`、`owned_by`、`provisioned_by`。
  - 采集链路覆盖 IaC、Cloud API、K8s API、APM/Tracing、人工补录，支持多源融合、去重合并、质量校验和发布。
  - 展示层提供业务拓扑、地域容灾、故障影响、变更影响四类视图，节点直接展示类型/地域/环境/owner/健康度/最近变更时间。
  - 资源列表增强展示：云服务器（VM）补充基础配置摘要（如 `CPU/内存/磁盘/IP/OS`），降低排障时跳转详情页成本。
  - 资源时效治理可视化：统一展示“业务过期时间”（如云资源到期）与“数据过期时间”（`last_seen_at + TTL`），支持即将过期/已过期标识。
  - 已同步 CI 资源列表操作增强：新增“查看详情”，可查看资产完整字段（基础信息、来源、attributes 原始数据、最近同步信息）。
  - 已同步 CI 资源动作增强：当资源类型为云服务器（VM）时，在列表操作中提供“重启/停止”能力，并透传到对应云厂商执行。
- 数据模型：
  - `resource_categories`（CI 类型定义，首批冻结核心 15 类，新增走评审）。
  - `resource_items`（CI 实体，`attributes JSONB`；必备字段：`ci_id/type/name/cloud/region/env/owner/lifecycle/source/last_seen_at`）。
  - `resource_relations`（关系边：`from_ci_id/to_ci_id/relation_type/direction/criticality/confidence/evidence/updated_at`）。
  - `resource_evidences`（关系或实体证据明细，记录采集源与原始标识）。
  - `resource_sync_jobs`、`resource_sync_job_items`（采集任务与质量校验结果）。
  - `tags`、`resource_tags`（业务标签、风险等级、成本中心等扩展标签）。
- 接口清单：
  - 基础管理：`/cmdb/categories`、`/cmdb/resources`、`/cmdb/tags`、`POST /cmdb/resources/:id/tags`。
  - 关系管理：`/cmdb/relations`、`GET /cmdb/resources/:id/upstream`、`GET /cmdb/resources/:id/downstream`。
  - 资源详情与动作：`GET /cmdb/resources/:id`（详情查看）、`POST /cmdb/resources/:id/actions/restart`、`POST /cmdb/resources/:id/actions/stop`（仅 VM）。
  - 视图查询：`GET /cmdb/topology/:application`、`GET /cmdb/impact/:ci_id`、`GET /cmdb/regions/:region/failover`、`GET /cmdb/change-impact/:release_id`。
  - 采集任务：`POST /cmdb/sync/jobs`、`GET /cmdb/sync/jobs/:id`、`POST /cmdb/sync/jobs/:id/retry`。
- 技术要点与验收：
  - 采集优先级：`IaC > Cloud API > K8s > APM > Manual`；冲突字段按优先级覆盖，`ci_id` 相同则合并。
  - 唯一标识：云主机=`cloud:account:region:instance_id`；K8s Workload=`cloud:region:cluster:namespace:kind:name`；托管 DB=`cloud:account:region:engine:instance_id`。
  - 采集流程：`Extract -> Normalize -> Identity Resolution -> Relation Build -> Quality Check -> Publish`，最终同时写入 CMDB 表与关系存储。
  - 多云接入：通过多云模块维护的云账号 `access key/secret key` 调用各云 API 拉取纳管资源，映射生成对应 CI。
  - 多集群接入：通过 `kubeconfig` 管理多集群连接，拉取 `Cluster/Namespace/Workload/Service/Ingress` 并映射生成对应 CI 与关系。
  - 风险治理：增量采集 + TTL（24h 未 `last_seen_at` 标黄）；关系 `confidence` 分级（观测推断低于配置声明）；全局命名规范 + 标签字典治理跨云命名。
  - 最小改造路径：不新增表结构，云厂商采集将服务器基础配置与 `expiresAt` 归一化写入 `resource_items.attributes`，前端在 CMDB 列表直接渲染“基础配置/过期时间”列。
  - VM 动作治理：`restart/stop` 仅对可执行云服务器开放；执行前校验资源类型与来源账号可用性，执行过程写入审计与操作结果（成功/失败/错误码）。
  - 分阶段：阶段1（直播核心链路 + 两云两地域 + 关键中间件）→ 阶段2（接入 APM 自动关系 + 故障影响分析）→ 阶段3（接入变更系统 + 发布风控 + 容量/成本视角）。
  - 验收场景（`live-core`）：可展示阿里云 `cn-shanghai` 与 AWS `ap-southeast-1` 双活拓扑；可回答 PostgreSQL 故障影响面、地域故障接管能力、Team 负责的 P0 资源清单。
  - 操作验收补充：CI 列表可一键打开详情抽屉查看完整资产数据；VM 的“重启/停止”在执行后可返回明确结果并可追溯审计。

### 4) 多云管理
- 功能点：
  - 多云账号管理、账号校验、按地域/区域维度的资源同步。
  - 工程化抽象：通过统一 `CloudProvider` + `ResourceCollector` 机制实现“同一套业务流程，多厂商适配器注入”，避免每个运营商重复开发一套逻辑。
  - 首批只接入基础能力资源：云服务器、云数据库 MySQL、私有网络、对象存储、文件存储、容器服务、负载均衡、域名管理、SSL 证书、日志服务。
  - 基础资源管理能力：对纳管基础资源提供统一增删改查（CRUD），支持手工补录、属性编辑、下线删除与详情查询。云资源创建时,需提供各个创建资源的模版, 供选择
  - 预留扩展能力：后续按同一抽象继续接入其它云产品，不改主流程。
- 数据模型：`cloud_accounts`、`cloud_assets`、`cloud_sync_jobs`（后续扩展）；其中 `cloud_assets` 需支持统一资源主键、region/type/status/metadata 与审计字段。
- 接口清单：`/cloud/accounts` CRUD、`POST /cloud/accounts/:id/verify`、`POST /cloud/accounts/:id/sync`、`GET /cloud/accounts/:id/assets`（按 region/type 过滤）、`/cloud/assets` CRUD、`GET /cloud/assets/:id`。
- 技术要点与验收：
  - 抽象层：统一账号凭据模型、统一资源标准字段（`provider/account/region/type/id/name/status/tags/metadata`）、统一错误码和限流/重试策略。
  - 实现层：分阶段逐个 SDK 接入（先 AWS/Aliyun 基础资源，再 Tencent/Huawei，最后扩展长尾服务）。
  - 多运营商 SDK 补全要求（新增强制）：
    - 厂商范围：`AWS / Aliyun / Tencent / Huawei` 四家必须全部落地真实 SDK（禁止长期使用 stub 作为线上实现）。
    - 统一能力：四家均需具备 `账号校验 + 基础资源同步 + 统一字段标准化 + 同步错误可观测`。
    - 基础资源最小集合：`云服务器 / MySQL / VPC / 对象存储 / 负载均衡`；其余资源按阶段持续补齐。
    - 分阶段验收顺序：`第1阶段 AWS + Aliyun`、`第2阶段 Tencent + Huawei`、`第3阶段 长尾服务扩展（文件存储/容器服务/DNS/SSL/日志服务）`。
    - 当前里程碑（2026-04）：优先完成 `AWS` 真实 SDK 替换并通过联调，再推进其余云厂商收敛。
    - 开发环境联调：保留 mock 前缀能力（`CLOUD_SDK_MOCK_*`），用于无真实凭据场景的流程验证。
    - 验收口径：四家任一账号在校验通过后，`/cloud/accounts/:id/sync` 可返回真实资产（非假数据），并可落库到 `cloud_assets` 与 CMDB 映射链路。
  - 安全：AK/SK 加密存储、最小权限原则、调用审计。
    - 新增强制：`cloud_accounts.access_key/secret_key` 禁止明文落库，统一采用应用层加密（如 AES-GCM）后存储。
    - 密钥管理：生产环境必须配置专用加密种子（`CLOUD_CREDENTIAL_ENCRYPT_KEY`），禁止依赖默认值；支持密钥轮换方案。
    - 兼容迁移：对历史明文数据需支持平滑迁移（读时解密兼容、写时自动转密文），不影响既有账号联调。
    - 回显规范：前端接口仅返回通用错误码/文案，云厂商详细错误仅写服务端日志与审计链路。
  - 验收：任一新运营商仅新增适配器实现即可复用同步主链路；首批基础资源可完成校验、拉取、标准化并写入 CMDB；基础资源 CRUD 能完成新增、查询、修改、删除闭环并保留审计记录。
  - 做好对平台的RBAC/ABAC权限管理.
  - 需按照项目UI/UX实现, 创建资源右边弹窗, 页面为资源列表. 列表字段可自定义
  - 模块需要适配aiops,为后面的aiops自然语言购买资料预留支持协议和接口

### 5) 工单管理
- 功能点：工单创建、流转、审批、状态跟踪、关联资产/任务。
- 数据模型：`tickets`、`ticket_flows`、`ticket_approvals`、`ticket_links`。
- 接口清单：`/tickets` CRUD、`POST /tickets/:id/approve`、`POST /tickets/:id/transition`。
- 技术要点与验收：状态机（pending/processing/resolved/closed）；审批节点可配置；验收=工单全流程闭环并可审计追踪。

### 6) Docker 管理
- 功能点：
  - Docker 主机纳管：支持 Docker API Endpoint、TLS 证书、主机标签、环境、负责人、健康状态、版本信息、资源容量与最后心跳时间维护。
  - 容器管理：容器列表、详情、日志、实时指标、启动、停止、重启、删除、资源限制查看；删除运行中容器需禁用或先停止后按强确认流程处理。
  - 镜像管理：镜像列表、详情、拉取、删除、标签信息、镜像大小、创建时间、RepoDigest 展示；危险清理动作需审计与二次确认。
  - 网络管理：网络列表、详情、创建、删除、容器连接/断开；删除网络前校验是否仍被容器占用。
  - 数据卷管理：数据卷列表、详情、创建、删除、挂载关系展示；删除前校验是否仍被容器引用。
  - Compose 管理：Compose Stack 列表、详情、配置校验、部署、启动、停止、重启、删除、服务列表、日志查看、变更历史。
  - 操作审计：所有写操作记录操作者、目标资源、请求参数摘要、执行结果、耗时、错误码、trace_id，支持按主机/资源/操作类型查询。
- 数据模型：
  - `docker_hosts`：Docker 主机、Endpoint、TLS 状态、证书密文引用、标签、健康状态、版本、容量、负责人、环境。
  - `docker_containers`：容器缓存快照、状态、镜像、端口、网络、挂载、资源配置、主机归属、最近同步时间。
  - `docker_images`：镜像缓存快照、仓库、标签、Digest、大小、创建时间、主机归属、最近同步时间。
  - `docker_networks`：网络缓存快照、驱动、子网、网关、关联容器、主机归属、最近同步时间。
  - `docker_volumes`：数据卷缓存快照、驱动、挂载点、标签、关联容器、主机归属、最近同步时间。
  - `docker_compose_stacks`：Stack 名称、项目路径、Compose 配置、版本、状态、服务数量、部署记录。
  - `docker_operations`、`docker_operation_logs`：操作任务、幂等键、锁标识、执行日志、失败原因、重试次数、审计字段。
- 接口清单：
  - 主机：`GET/POST /docker/hosts`、`GET/PUT/DELETE /docker/hosts/:id`、`POST /docker/hosts/:id/check`、`POST /docker/hosts/:id/sync`。
  - 容器：`GET /docker/hosts/:id/containers`、`GET /docker/containers/:id`、`GET /docker/containers/:id/logs`、`GET /docker/containers/:id/stats`、`POST /docker/containers/:id/start|stop|restart|remove`。
  - 镜像：`GET /docker/hosts/:id/images`、`POST /docker/hosts/:id/images/pull`、`POST /docker/images/:id/tag`、`POST /docker/images/:id/remove`。
  - 网络：`GET /docker/hosts/:id/networks`、`POST /docker/hosts/:id/networks`、`GET /docker/networks/:id`、`POST /docker/networks/:id/connect|disconnect|remove`。
  - 数据卷：`GET /docker/hosts/:id/volumes`、`POST /docker/hosts/:id/volumes`、`GET /docker/volumes/:id`、`POST /docker/volumes/:id/remove`。
  - Compose：`GET/POST /docker/compose/stacks`、`GET/PUT/DELETE /docker/compose/stacks/:id`、`POST /docker/compose/stacks/:id/validate|deploy|up|down|restart`、`GET /docker/compose/stacks/:id/logs`。
- 安全性关键指标：
  - 权限控制：所有 Docker 写操作必须接入 RBAC/ABAC，按主机、环境、资源类型、操作类型控制权限；高危操作需单独权限点。
  - 连接安全：生产环境优先使用 Docker API + TLS，禁止明文暴露 Docker TCP Socket；证书、Token、私钥必须加密存储，接口返回禁止回显敏感字段。
  - Endpoint 防护：新增主机时校验 Endpoint 协议、地址与端口，禁止任意本地 Socket 路径、内网敏感地址探测与 SSRF 风险入口。
  - 危险操作确认：删除主机、容器、镜像、网络、数据卷、Compose Stack 等删除动作必须二次确认，输入固定文案后才能执行，输入框禁止粘贴。
  - 运行态保护：运行中的容器、仍被引用的网络/数据卷、正在部署的 Compose Stack 默认禁止删除。
  - 命令执行限制：如后续支持 exec，需默认关闭；开启后必须有命令白名单、超时、审计、输出脱敏，禁止无边界交互式 root shell。
  - 日志脱敏：容器日志、Compose 日志、错误信息返回前需做密钥、Token、AK/SK、证书内容脱敏；详细错误仅写服务端日志和审计链路。
- 稳定性关键指标：
  - 超时控制：Docker API 调用必须设置连接超时、读写超时、整体请求超时，避免阻塞接口线程。
  - 幂等与锁：启动、停止、重启、删除、部署等写操作需支持幂等键；同一主机/容器/Stack 的互斥操作需使用分布式锁。
  - 异步任务：耗时操作如镜像拉取、Compose 部署、批量同步需进入任务化执行，前端通过操作记录或任务状态轮询查看结果。
  - 重试策略：仅对网络抖动、超时、临时不可用等可恢复错误做有限重试和指数退避；权限错误、参数错误不重试。
  - 健康熔断：连续失败的 Docker 主机进入异常状态，短时间内降低同步频率并提示用户检查连通性或证书。
  - 补偿能力：优先 Docker API，Docker API 明确不可用且符合安全策略时再使用 Ansible 补偿；补偿路径必须记录来源与差异。
  - 配置回滚：Compose 部署前保留上一版配置摘要和部署记录，失败时支持回滚到上一可用配置。
- 性能关键指标：
  - 分页与过滤：所有列表默认分页 10 条，支持自定义 pageSize；列表查询必须支持主机、状态、名称、镜像、标签、时间范围等服务端过滤。
  - 缓存策略：容器、镜像、网络、数据卷列表支持短 TTL 缓存与手动刷新，避免每次页面加载都全量请求 Docker API。
  - 日志读取：日志接口必须支持 `tail`、时间范围、关键词、最大返回量限制，禁止默认拉取全量日志。
  - 指标采样：容器 stats 使用采样窗口和并发上限，避免同时查看大量容器造成 Docker daemon 压力。
  - 并发控制：按主机限制同步、拉镜像、部署、日志读取并发数；全局设置队列上限与限流。
  - 数据同步：同步任务只保存必要快照字段，详情字段按需加载；大字段如日志、Compose 原文、错误堆栈不进入高频列表查询。
- UI/UX 约束：
  - 列表搜索栏按项目统一紧凑型设计，标题、搜索栏、列表间距保持紧凑；默认分页 10 条并支持自定义分页数量。
  - 创建/编辑 Docker 主机、Compose Stack 使用右侧弹窗，不使用页面中间大弹窗。
  - 列表字段支持自定义显示，字段配置按用户维度保存。
  - 操作按钮超过 3 个时使用 `...` 展示更多操作，弹层根据 `...` 位置向下展开并避免越界。
  - 删除操作遵循项目统一强确认规范：输入“确认删除资源”后才能删除，输入框禁止粘贴；运行中或被引用资源删除按钮禁用。
- AIOps 预留：
  - 为后续自然语言运维预留标准操作协议：资源类型、资源 ID、动作、参数 Schema、风险等级、dry-run、审批要求、回滚策略。
  - 所有写操作先支持 dry-run 校验，返回影响范围、风险提示、所需权限和预计执行步骤，便于后续接入 AIOpsChat。
  - 操作结果需返回机器可读错误码、trace_id、审计 ID，支持 AIOps 自动追踪失败原因。
  - Docker 管理模块需提供独立 AIOps 协议接口 `GET /docker/aiops/protocol`，返回协议版本、支持资源类型、动作清单、参数 Schema、风险等级、是否需要审批、是否支持 dry-run、回滚提示。
  - Docker 运维动作统一通过 `POST /docker/actions` 入口执行，AIOpsChat 后续只需要组装 `{hostId, resourceType, resourceId, action, dryRun, params}` 即可复用现有权限、审计、dry-run、操作记录链路。
  - dry-run 返回必须包含 `steps/impact/riskLevel/approvalRequired/rollback/safetyChecks`，用于自然语言执行前向用户确认影响范围。
  - 镜像/网络/数据卷删除与 Compose `validate/deploy/up/down/restart` 需支持真实执行；删除类和 Compose 高危动作真实执行必须前后端双重校验确认文案“确认删除资源”。
  - 操作记录 `docker_operations` 需保存 `trace_id/request/result/status/error_message`，便于 AIOps 追踪、解释失败原因和生成后续补偿建议。
- 技术要点与验收：
  - 工程实现需抽象 Docker Client、资源同步器、操作执行器、日志读取器、Compose 驱动，避免容器/镜像/网络/Compose 各自重复实现连接、鉴权、审计、锁、重试。
  - 验收=Docker 主机可纳管并通过 TLS 校验；容器/镜像/网络/数据卷/Compose 可分页查询、按条件搜索、查看详情；关键写操作可执行、可审计、可防误删；异常主机不会拖垮整体列表与同步任务。

### 7) 中间件管理
- 功能点：
  - 中间件实例纳管：首批支持 Redis、PostgreSQL、RabbitMQ；后续按插件扩展 MySQL、MongoDB、Kafka、Elasticsearch、Nacos、Consul 等。
  - 实例信息维护：名称、类型、Endpoint、环境、负责人、标签、版本、部署形态、认证方式、TLS 状态、健康状态、最后检查时间。
  - 连接健康检查：按中间件类型执行轻量探活，返回连通性、版本、角色、延迟、错误码、trace_id；失败详情仅写服务端日志。
  - 基础指标采集：连接数、内存/存储使用、QPS/TPS、慢查询/阻塞、队列堆积、复制延迟、主从/集群状态等关键指标。
  - 常用运维动作：Redis flush/dbsize/info、PostgreSQL 连接终止/慢查询查看、RabbitMQ 队列查看/purge/requeue 等；高危动作默认 dry-run。
  - 操作审计：所有写操作记录操作者、实例、动作、参数摘要、风险等级、dry-run、执行结果、耗时、错误码、trace_id。
- 数据模型：
  - `middleware_instances`：实例基础信息、类型、Endpoint、环境、负责人、标签、认证方式、TLS 状态、健康状态、版本、最后检查时间。
  - `middleware_credentials`：凭据密文、证书密文引用、密钥版本、轮换时间；接口禁止回显明文。
  - `middleware_metrics`：指标快照、指标类型、采样时间、实例 ID、关键数值、原始摘要。
  - `middleware_operations`：操作任务、trace_id、动作、参数摘要、dry-run、状态、风险等级、审批状态、结果摘要、失败原因。
  - `middleware_operation_logs`：操作执行日志、步骤、输出脱敏内容、耗时、错误码。
- 接口清单：
  - 实例：`GET/POST /middleware/instances`、`GET/PUT/DELETE /middleware/instances/:id`、`POST /middleware/instances/:id/check`。
  - 指标：`GET /middleware/instances/:id/metrics`、`POST /middleware/instances/:id/metrics/collect`。
  - 动作：`POST /middleware/actions`，统一承载 `{instanceId, type, action, dryRun, confirmationText, params}`。
  - 操作记录：`GET /middleware/operations`、`GET /middleware/operations/:id`。
  - AIOps 协议：`GET /middleware/aiops/protocol`，返回支持类型、动作、参数 Schema、风险等级、dry-run、审批与回滚要求。
- 安全性关键指标：
  - 权限控制：实例查看、凭据维护、健康检查、指标采集、写操作均需接入 RBAC/ABAC；高危动作需独立权限点。
  - 凭据安全：用户名、密码、Token、证书、私钥必须加密存储；接口、日志、审计、前端均禁止回显明文。
  - Endpoint 防护：新增实例需校验协议、地址、端口和类型匹配，禁止 SSRF、metadata 地址、非法本地 Socket、生产环境明文高危连接。
  - TLS 策略：生产环境优先 TLS；允许非 TLS 时必须标记风险并记录审计，后续可接入策略强制。
  - 高危确认：删除实例、清空数据、清空队列、终止连接、批量变更等动作必须二次确认，输入“确认删除资源”后才能执行，输入框禁止粘贴。
  - 动作白名单：所有中间件动作必须来自后端协议白名单，禁止前端透传任意命令、SQL、Lua、Shell。
  - 输出脱敏：慢查询、连接信息、错误堆栈、配置项输出必须脱敏密码、Token、DSN、证书内容。
- 稳定性关键指标：
  - 超时控制：连接、健康检查、指标采集、动作执行必须设置独立超时和上下文取消，避免接口线程长期阻塞。
  - 幂等与锁：同一实例的高危动作需使用幂等键和分布式锁，避免并发 flush/purge/restart 等冲突。
  - dry-run 优先：所有写操作默认 dry-run；真实执行必须显式 `dryRun=false`，高危动作还需确认文案和权限。
  - 重试策略：仅对网络抖动、临时连接失败做有限重试和指数退避；认证失败、参数错误、权限不足不重试。
  - 熔断降级：连续探活失败实例进入异常状态，降低自动采集频率，避免对故障中间件造成额外压力。
  - 操作任务化：耗时动作和批量采集进入异步任务，前端通过操作记录轮询状态。
  - 回滚提示：支持可回滚动作需记录回滚建议；不可回滚动作必须在 dry-run 中明确提示。
- 性能关键指标：
  - 分页与过滤：所有列表默认分页 10 条，支持自定义 pageSize；服务端支持类型、环境、状态、负责人、关键词过滤。
  - 指标采样：指标采集需限制频率、并发和保留周期；列表只展示摘要，详情按需加载。
  - 连接复用：可复用安全连接池，但需按实例、凭据版本、TLS 配置隔离；连接池设置最大连接数和空闲超时。
  - 并发控制：按实例和全局限制健康检查、指标采集、动作执行并发；避免批量操作压垮中间件。
  - 大结果限制：慢查询、队列消息、连接列表、日志输出必须支持 limit/offset/tail，禁止默认全量拉取。
  - 缓存策略：健康状态和指标摘要支持短 TTL 缓存，手动刷新可绕过缓存。
- UI/UX 约束：
  - 页面遵循项目统一紧凑型搜索栏，标题、搜索栏、列表间距保持紧凑。
  - 创建/编辑中间件实例使用右侧抽屉，不使用页面中部弹窗。
  - 列表字段支持自定义显示，字段配置按用户维度保存。
  - 操作按钮超过 3 个时使用 `...` 展示更多操作，弹层根据 `...` 位置向下展开并避免越界。
  - 删除和高危动作统一使用强确认弹窗，输入“确认删除资源”后才能执行，输入框禁止粘贴。
- AIOps 预留：
  - 提供 `GET /middleware/aiops/protocol`，返回协议版本、支持中间件类型、动作清单、参数 Schema、风险等级、是否需要审批、是否支持 dry-run、回滚提示。
  - 统一动作入口 `POST /middleware/actions`，AIOpsChat 后续只需组装 `{instanceId, type, action, dryRun, params, confirmationText}` 即可复用权限、审计、dry-run、操作记录链路。
  - dry-run 返回必须包含 `steps/impact/riskLevel/approvalRequired/rollback/safetyChecks/estimatedDuration`，用于自然语言执行前确认影响范围。
  - 操作记录 `middleware_operations` 必须保存 `trace_id/request/result/status/error_message`，便于 AIOps 追踪、解释失败原因和生成补偿建议。
  - AIOps 自然语言动作只允许映射到后端协议白名单，禁止模型生成任意 SQL/Lua/Shell 直接执行。
- 技术要点与验收：
  - 工程实现需抽象 Middleware Driver、HealthChecker、MetricCollector、ActionExecutor、CredentialProvider，避免 Redis/PostgreSQL/RabbitMQ 各自重复鉴权、审计、锁、重试、脱敏。
  - 验收=Redis/PostgreSQL/RabbitMQ 至少可完成实例纳管、健康检查、指标摘要查询、dry-run、低风险动作执行和操作审计；高危动作具备权限、确认、审计、回滚提示和测试覆盖。

### 8) 可观测性
- 功能点：指标查询、告警视图、日志链路入口。
- 数据模型：`observability_sources`、`alert_rules`、`alert_records`。
- 接口清单：`/observability/metrics/query`、`/observability/alerts`、`/observability/sources`。
- 技术要点与验收：Prometheus + Alertmanager 对接；验收=可查询核心指标并可查看告警。

### 9) 任务中心
- 功能点：任务 CRUD、Playbook CRUD、执行日志 CRUD、任务执行。
- 数据模型：`tasks`、`playbooks`、`task_execution_logs`。
- 接口清单：`/tasks`、`/playbooks`、`/task-logs`、`POST /tasks/:id/execute`。
- 技术要点与验收：Go 封装 ansible-playbook（超时/并发/日志/JSON/错误码）；`job_id` 全链路；`host_unreachable` 短重试、`task_failed` 默认不重试；高风险强制 `--check` + 人工确认；验收=执行记录可追溯、可审计。

### 10) Kubernetes 管理
- 功能点：集群管理、资源管理、节点管理。
- 数据模型：`k8s_clusters`、`k8s_credentials`、`k8s_operations`。
- 接口清单：`/kubernetes/clusters`、`/kubernetes/resources`、`/kubernetes/nodes`。
- 技术要点与验收：`client-go` 多集群接入；验收=可列集群资源并执行基础节点操作。

### 11) 事件中心
- 功能点：事件采集、归一化、检索、关联工单任务。
- 数据模型：`events`、`event_relations`。
- 接口清单：`/events` CRUD、`GET /events/search`、`POST /events/:id/link`。
- 技术要点与验收：Prometheus/Alertmanager 事件拉取；统一严重级别；验收=事件可检索可关联。

### 12) 站内消息
- 功能点：支持广播、用户、角色、部门频道推送；支持消息列表、搜索、频道过滤、已读/未读过滤、按用户标记已读；支持消息 `traceId` 全链路追踪；前端提供站内消息列表、分页、字段自定义、右侧抽屉创建消息、标记已读。
- 数据模型：`in_app_messages` 保存消息主体（`trace_id/channel/target/title/content/data/read`）；`message_read_receipts` 保存用户级已读回执（`message_id/user_id/read_at`）；`message_channels` 为逻辑层概念，频道枚举为 `broadcast/user/role/department`。
- 接口清单：
  - `GET /messages`：分页查询当前用户可见消息，支持 `keyword/channel/read/page/pageSize`。
  - `POST /messages`：创建消息并实时推送，需校验频道与目标合法性；`broadcast` 无目标，`user` 校验用户，`role` 校验角色，`department` 校验部门。
  - `POST /messages/:id/read`：仅允许当前用户对可见消息标记已读，写入用户级读回执。
  - `GET /ws`：WebSocket 实时通道，token 鉴权后按用户、角色、部门过滤推送。
- 后端工程要求：
  - WebSocket 需做 token 鉴权、运行时角色校验、Origin 校验、读写超时、ping/pong 保活、最大消息体限制。
  - Redis Pub/Sub 用于多实例广播；消息需携带节点来源，避免本实例重复消费本实例发布的消息。
  - 消息创建需生成 `traceId`，用于 API 返回、WebSocket 推送、前端列表追踪。
  - 已读/未读必须按用户隔离，禁止使用单个全局 `read` 字段作为最终判定，避免角色/部门/广播消息被一人读取后全员变已读。
  - 历史数据迁移需兼容已有 `in_app_messages`，新增追踪字段不应导致已有数据迁移失败。
- 前端 UI/UX 要求：
  - 页面遵循多云/CMDB 已确定的紧凑搜索栏、分页默认 10 条、自定义分页数量、字段自定义规范。
  - 创建消息必须通过按钮打开右侧抽屉完成，不使用页面中部弹窗。
  - 列表操作遵循项目操作区规范；未读状态需有清晰但克制的视觉提示。
  - WebSocket 断线后需自动重连并避免重复创建连接；实时消息到达后刷新列表并 toast 提示。
- 安全与稳定性验收：
  - 普通用户只能查询和标记自己可见的消息；不可通过 `messageId` 标记不可见消息。
  - 角色/部门/广播消息的已读状态按用户独立计算。
  - Redis 不可用时本实例内推送不受影响；多实例部署时 Redis Pub/Sub 可跨实例推送。
  - WebSocket 断线后可自动重连；长连接空闲不应长期占用僵尸连接。
  - 验收=消息可创建、可按频道送达、可实时到达、可追踪、可按用户已读未读查询，后端集成测试与前端构建通过。

### 13) 工具市场
- 功能点：工具注册、授权调用、执行记录。
- 数据模型：`tool_items`、`tool_permissions`、`tool_exec_logs`。
- 接口清单：`/tool-market/tools`、`POST /tool-market/tools/:id/execute`。
- 技术要点与验收：统一调用网关、限流与审计；验收=工具可配置并按权限执行。

### 14) AIOps
- 功能点：智能助手、RCA 根因分析、优化建议、智能体配置、模型配置。
- 数据模型：`ai_agent_configs`、`ai_model_configs`、`ai_sessions`、`ai_insights`。
- 接口清单：`/aiops/chat`、`/aiops/rca`、`/aiops/agents`、`/aiops/models`。
- 技术要点与验收：数据汇聚层（CMDB+指标+事件+云资源）；`ModelProvider` 兼容 OpenAI/Anthropic；验收=可完成问答、RCA、建议闭环。

## 前端 UI/UX 执行规范（ui-ux-pro-max）
- 风格：专业极简 + 数据密集，浅色主色信任蓝，深色中性背景。
- 字体：`Poppins`（标题）+ `Open Sans`（正文）。
- 交互：所有可点击元素 `cursor-pointer`，过渡 150-300ms，尊重 `prefers-reduced-motion`。
- 响应式：320/768/1024/1440；移动优先；折叠屏双栏与降级布局。
- 可访问性：对比度 >= 4.5:1、表单 label 完整、错误提示 `role=alert`。

### 前端模块页统一样式规范（强制）
- 术语统一：`横排=左右并排`，`竖排=上下排列`，评审与需求描述必须使用该定义避免歧义。
- 当前默认模块排布：子模块卡片采用`上下单列`（非左右双列），所有断点保持一致，禁止同模块在不同页面出现左右/上下混用。
- 卡片一致性：模块头部（标题+描述+主操作按钮）、列表表格、分页区三段式结构保持一致；卡片圆角/边框/间距统一。
- 列表规范：默认展示关键字段 + 操作列（查看详情/创建/修改/删除）；详情与编辑均走右侧贴边抽屉。
- 搜索栏规范（新增强制）：所有列表页统一采用“多云紧凑搜索栏”样式（`cloud-filter-bar/cloud-filter-control/cloud-filter-actions`），禁止使用松散换行布局占用过多垂直空间。
- 间距规范（新增强制）：列表卡片内遵循“标题区 -> 搜索栏 -> 列表”紧凑节奏，搜索栏需贴近标题与表格，避免大面积留白；后续新模块必须复用该节奏。
- 子模块紧凑规范（新增强制）：子模块卡片内“标题、搜索栏、列表、分页”必须采用紧凑垂直节奏，禁止出现标题到搜索栏、搜索栏到列表的大间隙；含搜索栏的列表卡片网格行需显式定义为 `auto auto 1fr auto`（或等价紧凑布局），避免因 `1fr` 误用导致中间留白。
- 创建交互规范（新增强制）：列表页创建/编辑入口只放在标题区按钮，点击后统一使用右侧抽屉表单；禁止在列表页正文内展开大块创建表单。
- 字段自定义规范：所有列表页在“操作”列表头右侧提供 `⚙️` 设置入口；设置弹窗仅用于字段显示/隐藏，不承载业务筛选。
- 操作列溢出规范（新增强制）：CMDB与多云管理列表中，当单行“操作”按钮数量 `>3` 时，仅展示前 3 个按钮 + `...`；点击 `...` 后在触发按钮下方就近弹出操作面板（非页面居中），并自动进行视口边界避让（左右防越界、超高滚动），避免列表横向拥挤与弹层内容溢出。
- 删除安全规范（新增强制）：所有删除操作必须二次确认，统一弹窗要求用户手动输入 `确认删除资源` 后才可提交；确认输入框禁止粘贴。对“运行中”状态的资源（如云资源、CMDB 资源）删除按钮默认禁用，不允许发起删除。
- 默认字段规范：列表首次进入默认仅显示关键字段（含操作列），其余字段由用户按需开启；字段偏好应本地持久化（刷新后保留）。
- 列筛选规范：筛选能力放在具体字段表头（筛选 icon）而非 `⚙️` 弹窗内；筛选弹层需支持搜索 + 单选/多选（按字段语义），避免被表格容器裁剪。
- 抽屉滚动规范：抽屉内容区域独立滚动（`overflow-y` + `overscroll-behavior: contain`），尽量不影响主体页面滚动。
- 分页规范：必须使用后端真分页；默认 `pageSize=10`，支持每页切换 `10/20/50`，支持页码输入跳转，分页控件样式在各模块保持一致。
- 变更约束：新模块页面若需偏离本规范，必须在 PR 描述中说明理由并附对比截图。

## 每轮实现 DoD（完成定义）
- 代码：后端 API + 前端页面/交互联通。
- 测试：单元/集成/响应结构回归通过。
- 文档：`PLAN.md`、`docs/API.md`、`backend/docs/openapi.yaml` 同步。
- 部署：Compose 可启动，部署变更同步 Helm。

## 测试计划
- 单元：鉴权、权限判定、服务层、执行器、适配器、前端 hooks/组件。
- 集成：登录→授权→业务接口、CMDB→任务执行、WebSocket 推送。
- 系统：14 模块主流程串联。
- 回归：全 API 成功/错误/分页结构一致、401 自动跳转、菜单按钮权限渲染。
- 非功能：并发、超时、重连、同步稳定性、高风险预检门禁。

## 假设与默认
- `cashbin` 按 Casbin 落地，适配层保留可替换。
- 架构先模块化单体，后续可按域拆分。
- ABAC 首批条件：`dept_id`、`resource_tag`、`env`。
- 四云首版覆盖账号校验 + 基础资源同步。

## 当前落地进度（本轮）
- ✅ 平台骨架：后端可启动、前端可构建、Compose/Helm/CI 脚手架完成。
- ✅ Phase 1 API 骨架：模块 1/2/3/9/12 已完成统一接口与数据模型。
- ✅ Phase 2 API 骨架：模块 4/5/6/7/8/10/11/13 已完成统一接口骨架。
- ✅ Phase 3 API 骨架：模块 14（AIOps）已完成模型配置、智能体配置、聊天与 RCA 接口骨架。
- ⏭️ 下一步：逐模块把“stub 接口”替换为真实 SDK/平台能力，并补模块集成测试与前端业务页。
