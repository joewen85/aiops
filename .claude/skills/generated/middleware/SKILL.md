---
name: middleware
description: "Skill for the Middleware area of aiops. 26 symbols across 7 files."
---

# Middleware

26 symbols | 7 files | Cohesion: 73%

## When to Use

- Working with code in `backend/`
- Understanding how TestListResponseShape, AutoMigrateModels, TestPermissionRequiredABACScopes work
- Modifying middleware-related functionality

## Key Files

| File | Symbols |
|------|---------|
| `backend/internal/middleware/permission_test.go` | TestPermissionRequiredABACScopes, TestPermissionRequiredUsernameFallbackAndUnauthorized, TestPermissionRequiredRuntimeRoleRevocation, TestPermissionRequiredRuntimeRoleRevocationWithCacheTTL, TestPermissionRequiredInactiveUserUnauthorized (+5) |
| `backend/internal/middleware/audit.go` | AuditLogger, needAudit, inferResource, sanitizeAuditPayload, sanitizeAuditValue (+1) |
| `backend/internal/middleware/permission.go` | PermissionRequired, PermissionRequiredWithRuntimeCache, newRoleCache, parseABACHeaders, verifyABACSignatureIfNeeded |
| `backend/internal/db/db.go` | InitPostgres, InitRabbitMQ |
| `backend/internal/response/response_test.go` | TestListResponseShape |
| `backend/internal/models/models.go` | AutoMigrateModels |
| `backend/internal/app/app.go` | New |

## Entry Points

Start here when exploring this area:

- **`TestListResponseShape`** (Function) — `backend/internal/response/response_test.go:11`
- **`AutoMigrateModels`** (Function) — `backend/internal/models/models.go:510`
- **`TestPermissionRequiredABACScopes`** (Function) — `backend/internal/middleware/permission_test.go:23`
- **`TestPermissionRequiredUsernameFallbackAndUnauthorized`** (Function) — `backend/internal/middleware/permission_test.go:63`
- **`TestPermissionRequiredRuntimeRoleRevocation`** (Function) — `backend/internal/middleware/permission_test.go:107`

## Key Symbols

| Symbol | Type | File | Line |
|--------|------|------|------|
| `TestListResponseShape` | Function | `backend/internal/response/response_test.go` | 11 |
| `AutoMigrateModels` | Function | `backend/internal/models/models.go` | 510 |
| `TestPermissionRequiredABACScopes` | Function | `backend/internal/middleware/permission_test.go` | 23 |
| `TestPermissionRequiredUsernameFallbackAndUnauthorized` | Function | `backend/internal/middleware/permission_test.go` | 63 |
| `TestPermissionRequiredRuntimeRoleRevocation` | Function | `backend/internal/middleware/permission_test.go` | 107 |
| `TestPermissionRequiredRuntimeRoleRevocationWithCacheTTL` | Function | `backend/internal/middleware/permission_test.go` | 159 |
| `TestPermissionRequiredInactiveUserUnauthorized` | Function | `backend/internal/middleware/permission_test.go` | 220 |
| `TestPermissionRequiredRejectsInvalidABACHeaders` | Function | `backend/internal/middleware/permission_test.go` | 265 |
| `TestPermissionRequiredABACSignatureValidation` | Function | `backend/internal/middleware/permission_test.go` | 290 |
| `PermissionRequired` | Function | `backend/internal/middleware/permission.go` | 33 |
| `InitPostgres` | Function | `backend/internal/db/db.go` | 15 |
| `InitRabbitMQ` | Function | `backend/internal/db/db.go` | 43 |
| `New` | Function | `backend/internal/app/app.go` | 29 |
| `AuditLogger` | Function | `backend/internal/middleware/audit.go` | 31 |
| `PermissionRequiredWithRuntimeCache` | Function | `backend/internal/middleware/permission.go` | 37 |
| `newABACEnforcerForTest` | Function | `backend/internal/middleware/permission_test.go` | 331 |
| `newPermissionTestDB` | Function | `backend/internal/middleware/permission_test.go` | 355 |
| `signABACRequest` | Function | `backend/internal/middleware/permission_test.go` | 368 |
| `needAudit` | Function | `backend/internal/middleware/audit.go` | 78 |
| `inferResource` | Function | `backend/internal/middleware/audit.go` | 87 |

## Connected Areas

| Area | Connections |
|------|-------------|
| Handler | 8 calls |
| App | 2 calls |
| Api | 1 calls |
| Config | 1 calls |
| Docker | 1 calls |
| Service | 1 calls |
| Rbac | 1 calls |
| Ws | 1 calls |

## How to Explore

1. `gitnexus_context({name: "TestListResponseShape"})` — see callers and callees
2. `gitnexus_query({query: "middleware"})` — find related execution flows
3. Read key files listed above for implementation details
