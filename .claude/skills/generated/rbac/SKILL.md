---
name: rbac
description: "Skill for the Rbac area of aiops. 4 symbols across 2 files."
---

# Rbac

4 symbols | 2 files | Cohesion: 67%

## When to Use

- Working with code in `backend/`
- Understanding how TestInitEnforcerNormalizesLegacyScopeColumns, InitEnforcer work
- Modifying rbac-related functionality

## Key Files

| File | Symbols |
|------|---------|
| `backend/internal/rbac/casbin.go` | InitEnforcer, ensureAdminPolicy, normalizePolicyScopes |
| `backend/internal/rbac/casbin_test.go` | TestInitEnforcerNormalizesLegacyScopeColumns |

## Entry Points

Start here when exploring this area:

- **`TestInitEnforcerNormalizesLegacyScopeColumns`** (Function) — `backend/internal/rbac/casbin_test.go:27`
- **`InitEnforcer`** (Function) — `backend/internal/rbac/casbin.go:8`

## Key Symbols

| Symbol | Type | File | Line |
|--------|------|------|------|
| `TestInitEnforcerNormalizesLegacyScopeColumns` | Function | `backend/internal/rbac/casbin_test.go` | 27 |
| `InitEnforcer` | Function | `backend/internal/rbac/casbin.go` | 8 |
| `ensureAdminPolicy` | Function | `backend/internal/rbac/casbin.go` | 32 |
| `normalizePolicyScopes` | Function | `backend/internal/rbac/casbin.go` | 54 |

## How to Explore

1. `gitnexus_context({name: "TestInitEnforcerNormalizesLegacyScopeColumns"})` — see callers and callees
2. `gitnexus_query({query: "rbac"})` — find related execution flows
3. Read key files listed above for implementation details
