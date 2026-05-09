---
name: store
description: "Skill for the Store area of aiops. 26 symbols across 10 files."
---

# Store

26 symbols | 10 files | Cohesion: 75%

## When to Use

- Working with code in `frontend/`
- Understanding how getPermissionBundle, permissionEventName, hasButtonPermission work
- Modifying store-related functionality

## Key Files

| File | Symbols |
|------|---------|
| `frontend/src/store/permission.ts` | getPermissionBundle, permissionEventName, hasButtonPermission, hasMenuPermission, hasPermissionByType (+3) |
| `frontend/src/store/auth.ts` | getUser, setToken, setUser, clearToken, clearUser (+1) |
| `frontend/src/layouts/AppLayout.tsx` | AppLayout, syncPermissions, handleLogout |
| `frontend/src/hooks/usePermissionBundle.ts` | usePermissionBundle, handler |
| `frontend/src/components/RowActionOverflow.tsx` | RowActionOverflow, handleOverflowActionClick |
| `frontend/src/components/PermissionButton.tsx` | PermissionButton |
| `frontend/src/hooks/useTheme.ts` | useTheme |
| `frontend/src/components/MenuRouteGuard.tsx` | MenuRouteGuard |
| `frontend/src/pages/LoginPage.tsx` | onSubmit |
| `frontend/src/api/auth.ts` | fetchMyPermissions |

## Entry Points

Start here when exploring this area:

- **`getPermissionBundle`** (Function) — `frontend/src/store/permission.ts:6`
- **`permissionEventName`** (Function) — `frontend/src/store/permission.ts:26`
- **`hasButtonPermission`** (Function) — `frontend/src/store/permission.ts:34`
- **`usePermissionBundle`** (Function) — `frontend/src/hooks/usePermissionBundle.ts:5`
- **`handler`** (Function) — `frontend/src/hooks/usePermissionBundle.ts:10`

## Key Symbols

| Symbol | Type | File | Line |
|--------|------|------|------|
| `getPermissionBundle` | Function | `frontend/src/store/permission.ts` | 6 |
| `permissionEventName` | Function | `frontend/src/store/permission.ts` | 26 |
| `hasButtonPermission` | Function | `frontend/src/store/permission.ts` | 34 |
| `usePermissionBundle` | Function | `frontend/src/hooks/usePermissionBundle.ts` | 5 |
| `handler` | Function | `frontend/src/hooks/usePermissionBundle.ts` | 10 |
| `RowActionOverflow` | Function | `frontend/src/components/RowActionOverflow.tsx` | 22 |
| `handleOverflowActionClick` | Function | `frontend/src/components/RowActionOverflow.tsx` | 87 |
| `PermissionButton` | Function | `frontend/src/components/PermissionButton.tsx` | 10 |
| `hasMenuPermission` | Function | `frontend/src/store/permission.ts` | 30 |
| `isAdminUser` | Function | `frontend/src/store/permission.ts` | 48 |
| `getUser` | Function | `frontend/src/store/auth.ts` | 18 |
| `useTheme` | Function | `frontend/src/hooks/useTheme.ts` | 5 |
| `MenuRouteGuard` | Function | `frontend/src/components/MenuRouteGuard.tsx` | 11 |
| `AppLayout` | Function | `frontend/src/layouts/AppLayout.tsx` | 30 |
| `setPermissionBundle` | Function | `frontend/src/store/permission.ts` | 16 |
| `setToken` | Function | `frontend/src/store/auth.ts` | 10 |
| `setUser` | Function | `frontend/src/store/auth.ts` | 28 |
| `onSubmit` | Function | `frontend/src/pages/LoginPage.tsx` | 15 |
| `fetchMyPermissions` | Function | `frontend/src/api/auth.ts` | 4 |
| `syncPermissions` | Function | `frontend/src/layouts/AppLayout.tsx` | 40 |

## Connected Areas

| Area | Connections |
|------|-------------|
| Api | 2 calls |
| Pages | 2 calls |

## How to Explore

1. `gitnexus_context({name: "getPermissionBundle"})` — see callers and callees
2. `gitnexus_query({query: "store"})` — find related execution flows
3. Read key files listed above for implementation details
