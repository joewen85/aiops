---
name: api
description: "Skill for the Api area of aiops. 45 symbols across 10 files."
---

# Api

45 symbols | 10 files | Cohesion: 58%

## When to Use

- Working with code in `frontend/`
- Understanding how loadInstances, loadMetrics, loadOperations work
- Modifying api-related functionality

## Key Files

| File | Symbols |
|------|---------|
| `frontend/src/api/tickets.ts` | buildQuery, deleteTicket, createTicketComment, getTicketAIOpsContext, executeTicketOperation (+6) |
| `frontend/src/api/docker.ts` | buildQuery, listDockerHostResources, listComposeStacks, getDockerAIOpsProtocol, runDockerAction (+1) |
| `frontend/src/pages/DockerPage.tsx` | loadResources, loadStacks, loadProtocolAndOperations, executeDockerAction, confirmDockerAction |
| `frontend/src/api/cmdb.ts` | buildQuery, listCMDBResources, listCMDBRelations, getCMDBTopology, getCMDBImpact |
| `frontend/src/api/middleware.ts` | buildQuery, listMiddlewareInstances, listMiddlewareMetrics, listMiddlewareOperations |
| `frontend/src/pages/TicketsPage.tsx` | loadContext, submitComment, confirmDanger, runSimpleAction |
| `frontend/src/pages/MiddlewarePage.tsx` | loadInstances, loadMetrics, loadOperations |
| `frontend/src/api/cloud.ts` | buildQuery, listCloudAssets, listCloudAccountAssets |
| `backend/internal/middleware/permission.go` | get, set, resolveRuntimeRoles |
| `frontend/src/api/aiopsProcurement.ts` | getAIOpsProcurementProtocol |

## Entry Points

Start here when exploring this area:

- **`loadInstances`** (Function) — `frontend/src/pages/MiddlewarePage.tsx:177`
- **`loadMetrics`** (Function) — `frontend/src/pages/MiddlewarePage.tsx:202`
- **`loadOperations`** (Function) — `frontend/src/pages/MiddlewarePage.tsx:211`
- **`loadResources`** (Function) — `frontend/src/pages/DockerPage.tsx:263`
- **`loadStacks`** (Function) — `frontend/src/pages/DockerPage.tsx:282`

## Key Symbols

| Symbol | Type | File | Line |
|--------|------|------|------|
| `loadInstances` | Function | `frontend/src/pages/MiddlewarePage.tsx` | 177 |
| `loadMetrics` | Function | `frontend/src/pages/MiddlewarePage.tsx` | 202 |
| `loadOperations` | Function | `frontend/src/pages/MiddlewarePage.tsx` | 211 |
| `loadResources` | Function | `frontend/src/pages/DockerPage.tsx` | 263 |
| `loadStacks` | Function | `frontend/src/pages/DockerPage.tsx` | 282 |
| `loadProtocolAndOperations` | Function | `frontend/src/pages/DockerPage.tsx` | 298 |
| `executeDockerAction` | Function | `frontend/src/pages/DockerPage.tsx` | 452 |
| `confirmDockerAction` | Function | `frontend/src/pages/DockerPage.tsx` | 476 |
| `listMiddlewareInstances` | Function | `frontend/src/api/middleware.ts` | 45 |
| `listMiddlewareMetrics` | Function | `frontend/src/api/middleware.ts` | 82 |
| `listMiddlewareOperations` | Function | `frontend/src/api/middleware.ts` | 98 |
| `listDockerHostResources` | Function | `frontend/src/api/docker.ts` | 88 |
| `listComposeStacks` | Function | `frontend/src/api/docker.ts` | 99 |
| `getDockerAIOpsProtocol` | Function | `frontend/src/api/docker.ts` | 125 |
| `runDockerAction` | Function | `frontend/src/api/docker.ts` | 130 |
| `listDockerOperations` | Function | `frontend/src/api/docker.ts` | 135 |
| `listCMDBResources` | Function | `frontend/src/api/cmdb.ts` | 41 |
| `listCMDBRelations` | Function | `frontend/src/api/cmdb.ts` | 96 |
| `getCMDBTopology` | Function | `frontend/src/api/cmdb.ts` | 139 |
| `getCMDBImpact` | Function | `frontend/src/api/cmdb.ts` | 145 |

## Execution Flows

| Flow | Type | Steps |
|------|------|-------|
| `CMDBPage → RoleCacheEntry` | cross_community | 6 |
| `DockerPage → RoleCacheEntry` | cross_community | 6 |
| `CloudPage → RoleCacheEntry` | cross_community | 6 |
| `MiddlewarePage → RoleCacheEntry` | cross_community | 6 |
| `HandleCreateResource → RoleCacheEntry` | cross_community | 6 |
| `HandleCreateRelation → RoleCacheEntry` | cross_community | 6 |
| `SubmitStack → RoleCacheEntry` | cross_community | 6 |
| `RBACPage → RoleCacheEntry` | cross_community | 5 |
| `HandleSubmitRole → RoleCacheEntry` | cross_community | 5 |
| `HandleSubmitPermission → RoleCacheEntry` | cross_community | 5 |

## Connected Areas

| Area | Connections |
|------|-------------|
| Pages | 16 calls |

## How to Explore

1. `gitnexus_context({name: "loadInstances"})` — see callers and callees
2. `gitnexus_query({query: "api"})` — find related execution flows
3. Read key files listed above for implementation details
