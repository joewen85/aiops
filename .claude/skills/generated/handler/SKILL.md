---
name: handler
description: "Skill for the Handler area of aiops. 388 symbols across 33 files."
---

# Handler

388 symbols | 33 files | Cohesion: 79%

## When to Use

- Working with code in `backend/`
- Understanding how Success, Error, Internal work
- Modifying handler-related functionality

## Key Files

| File | Symbols |
|------|---------|
| `backend/internal/handler/cmdb_modules.go` | CreateResource, UpdateResource, DeleteResource, BindResourceTags, CreateResourceRelation (+64) |
| `backend/internal/handler/ticket_modules.go` | GetTicket, CreateTicket, UpdateTicket, DeleteTicket, TransferTicket (+61) |
| `backend/internal/handler/phase2_phase3_modules.go` | ListCloudAccounts, GetCloudAccount, CreateCloudAccount, UpdateCloudAccount, VerifyCloudAccount (+49) |
| `backend/internal/handler/docker_modules.go` | CreateDockerHost, UpdateDockerHost, DeleteDockerHost, CheckDockerHost, CreateComposeStack (+31) |
| `backend/internal/handler/core_modules.go` | CreateUser, UpdateUser, ToggleUserStatus, ResetUserPassword, BindUserRoles (+26) |
| `backend/internal/handler/middleware_modules.go` | CreateMiddlewareInstance, UpdateMiddlewareInstance, DeleteMiddlewareInstance, CheckMiddlewareInstance, CollectMiddlewareMetrics (+15) |
| `backend/internal/handler/cloud_modules.go` | CreateCloudAsset, UpdateCloudAsset, ListCloudAssets, ListCloudAccountAssets, GetCloudAsset (+15) |
| `backend/internal/handler/cmdb_tasks_modules.go` | ExecuteTask, GetTask, GetPlaybook, GetTaskLog, DeleteTask (+8) |
| `backend/internal/handler/messages_ws.go` | CreateMessage, MessageAIOpsProtocol, MessageAIOpsContext, MarkMessageRead, visibleMessagesQuery (+6) |
| `backend/internal/handler/cmdb_resource_actions.go` | handleCMDBVMAction, isVMResourceType, ensureCMDBVMActionAuthorized, cloudProviderQueryAliases, extractCMDBVMActionContext (+4) |

## Entry Points

Start here when exploring this area:

- **`Success`** (Function) — `backend/internal/response/response.go:23`
- **`Error`** (Function) — `backend/internal/response/response.go:40`
- **`Internal`** (Function) — `backend/internal/response/response.go:47`
- **`GetClaims`** (Function) — `backend/internal/middleware/auth.go:63`
- **`TestCloudProviderResolveAppError`** (Function) — `backend/internal/handler/cloud_provider_helpers_test.go:29`

## Key Symbols

| Symbol | Type | File | Line |
|--------|------|------|------|
| `Success` | Function | `backend/internal/response/response.go` | 23 |
| `Error` | Function | `backend/internal/response/response.go` | 40 |
| `Internal` | Function | `backend/internal/response/response.go` | 47 |
| `GetClaims` | Function | `backend/internal/middleware/auth.go` | 63 |
| `TestCloudProviderResolveAppError` | Function | `backend/internal/handler/cloud_provider_helpers_test.go` | 29 |
| `TestValidateCloudCredentialInput` | Function | `backend/internal/handler/cloud_provider_helpers_test.go` | 41 |
| `New` | Function | `backend/internal/errors/errors.go` | 19 |
| `List` | Function | `backend/internal/response/response.go` | 31 |
| `Parse` | Function | `backend/internal/pagination/pagination.go` | 13 |
| `Offset` | Function | `backend/internal/pagination/pagination.go` | 22 |
| `TestTryAcquireCMDBSyncLockWithoutRedis` | Function | `backend/internal/handler/cmdb_sync_lock_test.go` | 15 |
| `TestTryAcquireCMDBSyncLockRedisErrorFallback` | Function | `backend/internal/handler/cmdb_sync_lock_test.go` | 42 |
| `ValidateEndpointSecurity` | Function | `backend/internal/docker/client.go` | 29 |
| `NewClient` | Function | `backend/internal/docker/client.go` | 88 |
| `InitRedis` | Function | `backend/internal/db/db.go` | 29 |
| `TestSyncRolePoliciesFiltersAPIAndKeepsABACScopes` | Function | `backend/internal/handler/rbac_helpers_test.go` | 21 |
| `TestBindRolePermissionsSyncsPolicies` | Function | `backend/internal/handler/rbac_helpers_test.go` | 78 |
| `New` | Function | `backend/internal/handler/handler.go` | 32 |
| `NewDefaultResourceCollector` | Function | `backend/internal/cloud/collector.go` | 8 |
| `NewStubProcurementEngine` | Function | `backend/internal/ai/procurement.go` | 82 |

## Execution Flows

| Flow | Type | Steps |
|------|------|-------|
| `CheckDockerHost → NormalizeEndpoint` | cross_community | 6 |
| `CheckDockerHost → Client` | cross_community | 6 |
| `CreateDockerHost → NormalizeEndpoint` | cross_community | 6 |
| `CreateDockerHost → Client` | cross_community | 6 |
| `ExecuteDockerAction → NormalizeEndpoint` | cross_community | 6 |
| `ExecuteDockerAction → Client` | cross_community | 6 |
| `ToggleUserStatus → APIResponse` | cross_community | 5 |
| `ToggleUserStatus → AppError` | cross_community | 5 |
| `DockerAction → APIResponse` | cross_community | 4 |
| `DockerAction → AppError` | cross_community | 4 |

## Connected Areas

| Area | Connections |
|------|-------------|
| Cloud | 9 calls |
| Middlewaresvc | 5 calls |
| App | 3 calls |
| Middleware | 3 calls |
| Docker | 3 calls |
| Ai | 3 calls |
| Executor | 2 calls |
| Rbac | 1 calls |

## How to Explore

1. `gitnexus_context({name: "Success"})` — see callers and callees
2. `gitnexus_query({query: "handler"})` — find related execution flows
3. Read key files listed above for implementation details
