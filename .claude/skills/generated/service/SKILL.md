---
name: service
description: "Skill for the Service area of aiops. 8 symbols across 2 files."
---

# Service

8 symbols | 2 files | Cohesion: 82%

## When to Use

- Working with code in `backend/`
- Understanding how TestSeedRBACDefaults_Idempotent, SeedRBACDefaults work
- Modifying service-related functionality

## Key Files

| File | Symbols |
|------|---------|
| `backend/internal/service/rbac_seed.go` | SeedRBACDefaults, seedPermissionCatalog, bindAllPermissionsToAdmin, normalizeScope, permissionSeeds |
| `backend/internal/service/rbac_seed_test.go` | TestSeedRBACDefaults_Idempotent, assertPermissionExists, memoryDSN |

## Entry Points

Start here when exploring this area:

- **`TestSeedRBACDefaults_Idempotent`** (Function) — `backend/internal/service/rbac_seed_test.go:13`
- **`SeedRBACDefaults`** (Function) — `backend/internal/service/rbac_seed.go:10`

## Key Symbols

| Symbol | Type | File | Line |
|--------|------|------|------|
| `TestSeedRBACDefaults_Idempotent` | Function | `backend/internal/service/rbac_seed_test.go` | 13 |
| `SeedRBACDefaults` | Function | `backend/internal/service/rbac_seed.go` | 10 |
| `assertPermissionExists` | Function | `backend/internal/service/rbac_seed_test.go` | 60 |
| `memoryDSN` | Function | `backend/internal/service/rbac_seed_test.go` | 68 |
| `seedPermissionCatalog` | Function | `backend/internal/service/rbac_seed.go` | 23 |
| `bindAllPermissionsToAdmin` | Function | `backend/internal/service/rbac_seed.go` | 57 |
| `normalizeScope` | Function | `backend/internal/service/rbac_seed.go` | 96 |
| `permissionSeeds` | Function | `backend/internal/service/rbac_seed.go` | 103 |

## Connected Areas

| Area | Connections |
|------|-------------|
| Middleware | 1 calls |
| App | 1 calls |

## How to Explore

1. `gitnexus_context({name: "TestSeedRBACDefaults_Idempotent"})` — see callers and callees
2. `gitnexus_query({query: "service"})` — find related execution flows
3. Read key files listed above for implementation details
