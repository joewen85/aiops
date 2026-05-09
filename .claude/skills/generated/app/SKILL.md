---
name: app
description: "Skill for the App area of aiops. 93 symbols across 17 files."
---

# App

93 symbols | 17 files | Cohesion: 95%

## When to Use

- Working with code in `backend/`
- Understanding how SeedDefaultAdmin, TestPasswordHash, HashPassword work
- Modifying app-related functionality

## Key Files

| File | Symbols |
|------|---------|
| `backend/internal/app/auth_rbac_integration_test.go` | TestLoginAndMePermissionsIntegration, TestTasksAPIABACAuthorizationIntegration, TestMePermissionsRealtimeConsistencyAfterRBACChange, TestMePermissionsCompactBundleIntegration, TestTasksAPIRuntimeRevocationWithoutReloginIntegration (+31) |
| `backend/internal/app/router.go` | setupRouter, registerUserDepartmentRoutes, registerRBACRoutes, registerCMDBRoutes, registerTaskRoutes (+4) |
| `backend/internal/app/cmdb_integration_test.go` | TestCMDBResourceRelationAndImpactIntegration, TestCMDBSyncJobIntegration, TestCMDBSyncJobRejectWhenRunningIntegration, TestCMDBResourceDetailAndVMActionValidationIntegration, TestCMDBVMActionRejectUnsyncedResourceIntegration (+3) |
| `backend/internal/app/docker_integration_test.go` | TestDockerManagementAIOpsProtocolAndDryRunIntegration, TestDockerHostCheckAndResourceQueryIntegration, TestDockerRemoveActionsRequireConfirmAndExecuteIntegration, TestDockerComposeDeployActionRequiresConfirmAndExecutesIntegration, TestDockerEndpointAndDeleteSafetyIntegration (+2) |
| `backend/internal/app/cloud_integration_test.go` | TestCloudAssetCRUDIntegration, TestCloudAccountListFilterIntegration, TestCloudAccountSyncRunningConflictIntegration, TestCloudAccountSecurityAndSyncRobustnessIntegration, TestCloudAccountTencentUpdateFromLegacyInvalidCredentialIntegration (+2) |
| `backend/internal/auth/jwt.go` | HashPassword, ComparePassword, NewManager, GenerateToken, Parse |
| `backend/internal/app/tickets_integration_test.go` | TestTicketManagementIntegration, TestTicketAIOpsContextAndFiltersIntegration, TestTicketVisibilityIsolationIntegration, TestTicketApproveRequiresPendingApproverIntegration, TestTicketSubmitIdempotentIntegration |
| `backend/internal/app/cors_test.go` | TestBuildCORSConfig_DefaultOrigins, TestBuildCORSConfig_Wildcard, contains |
| `backend/internal/auth/jwt_test.go` | TestPasswordHash, TestJWTGenerateAndParse |
| `backend/internal/app/messages_integration_test.go` | TestMessagesRoleVisibilityAndReadReceiptIntegration, TestMessagesAIOpsContextAndModuleFiltersIntegration |

## Entry Points

Start here when exploring this area:

- **`SeedDefaultAdmin`** (Function) — `backend/internal/service/bootstrap.go:11`
- **`TestPasswordHash`** (Function) — `backend/internal/auth/jwt_test.go:22`
- **`HashPassword`** (Function) — `backend/internal/auth/jwt.go:62`
- **`ComparePassword`** (Function) — `backend/internal/auth/jwt.go:70`
- **`TestTicketManagementIntegration`** (Function) — `backend/internal/app/tickets_integration_test.go:12`

## Key Symbols

| Symbol | Type | File | Line |
|--------|------|------|------|
| `SeedDefaultAdmin` | Function | `backend/internal/service/bootstrap.go` | 11 |
| `TestPasswordHash` | Function | `backend/internal/auth/jwt_test.go` | 22 |
| `HashPassword` | Function | `backend/internal/auth/jwt.go` | 62 |
| `ComparePassword` | Function | `backend/internal/auth/jwt.go` | 70 |
| `TestTicketManagementIntegration` | Function | `backend/internal/app/tickets_integration_test.go` | 12 |
| `TestTicketAIOpsContextAndFiltersIntegration` | Function | `backend/internal/app/tickets_integration_test.go` | 174 |
| `TestTicketVisibilityIsolationIntegration` | Function | `backend/internal/app/tickets_integration_test.go` | 273 |
| `TestTicketApproveRequiresPendingApproverIntegration` | Function | `backend/internal/app/tickets_integration_test.go` | 357 |
| `TestTicketSubmitIdempotentIntegration` | Function | `backend/internal/app/tickets_integration_test.go` | 408 |
| `TestMiddlewareManagementIntegration` | Function | `backend/internal/app/middleware_integration_test.go` | 11 |
| `TestMessagesRoleVisibilityAndReadReceiptIntegration` | Function | `backend/internal/app/messages_integration_test.go` | 12 |
| `TestMessagesAIOpsContextAndModuleFiltersIntegration` | Function | `backend/internal/app/messages_integration_test.go` | 98 |
| `TestDockerManagementAIOpsProtocolAndDryRunIntegration` | Function | `backend/internal/app/docker_integration_test.go` | 16 |
| `TestDockerHostCheckAndResourceQueryIntegration` | Function | `backend/internal/app/docker_integration_test.go` | 79 |
| `TestDockerRemoveActionsRequireConfirmAndExecuteIntegration` | Function | `backend/internal/app/docker_integration_test.go` | 150 |
| `TestDockerComposeDeployActionRequiresConfirmAndExecutesIntegration` | Function | `backend/internal/app/docker_integration_test.go` | 218 |
| `TestDockerEndpointAndDeleteSafetyIntegration` | Function | `backend/internal/app/docker_integration_test.go` | 292 |
| `TestDockerActionRejectsDuplicateRunningOperationIntegration` | Function | `backend/internal/app/docker_integration_test.go` | 343 |
| `TestCMDBResourceRelationAndImpactIntegration` | Function | `backend/internal/app/cmdb_integration_test.go` | 19 |
| `TestCMDBSyncJobIntegration` | Function | `backend/internal/app/cmdb_integration_test.go` | 169 |

## Execution Flows

| Flow | Type | Steps |
|------|------|-------|
| `Login → ComparePassword` | cross_community | 3 |

## Connected Areas

| Area | Connections |
|------|-------------|
| Middleware | 3 calls |
| Handler | 2 calls |
| Rbac | 1 calls |
| Cloud | 1 calls |

## How to Explore

1. `gitnexus_context({name: "SeedDefaultAdmin"})` — see callers and callees
2. `gitnexus_query({query: "app"})` — find related execution flows
3. Read key files listed above for implementation details
