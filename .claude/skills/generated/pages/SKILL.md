---
name: pages
description: "Skill for the Pages area of aiops. 371 symbols across 19 files."
---

# Pages

371 symbols | 19 files | Cohesion: 69%

## When to Use

- Working with code in `frontend/`
- Understanding how UsersPage, loadDepartmentPage, loadDepartmentTree work
- Modifying pages-related functionality

## Key Files

| File | Symbols |
|------|---------|
| `frontend/src/pages/UsersPage.tsx` | defaultUserForm, defaultDepartmentForm, UsersPage, loadDepartmentPage, loadDepartmentTree (+44) |
| `frontend/src/pages/TicketsPage.tsx` | defaultForm, TicketsPage, loadProtocol, loadTemplates, openCreateDrawer (+37) |
| `frontend/src/pages/RBACPage.tsx` | extractPermissionModule, normalizeKeyword, containsKeyword, formatModuleLabel, RBACPage (+34) |
| `frontend/src/pages/CloudPage.tsx` | loadAccountPage, loadAssetPage, handleApplyAssetTemplate, handleVerifyAccount, handleSyncAccount (+30) |
| `frontend/src/pages/CMDBPage.tsx` | openResourceDetailDrawer, handleVMAction, defaultResourceFilters, defaultRelationFilters, CMDBPage (+29) |
| `frontend/src/pages/DockerPage.tsx` | defaultHostFilter, defaultResourceFilter, DockerPage, loadHosts, handleDeleteHost (+27) |
| `frontend/src/pages/MiddlewarePage.tsx` | MiddlewarePage, loadProtocol, renderMiddlewareCell, totalPages, formatTime (+23) |
| `frontend/src/api/users.ts` | listDepartments, listDepartmentTree, getDepartment, deleteDepartment, resetUserPassword (+14) |
| `frontend/src/pages/MessagesPage.tsx` | toggleVisibleColumn, defaultFilter, defaultForm, MessagesPage, loadMessages (+10) |
| `frontend/src/api/tickets.ts` | listTicketTemplates, getTicketAIOpsProtocol, getTicket, createTicketLink, deleteTicketLink (+7) |

## Entry Points

Start here when exploring this area:

- **`UsersPage`** (Function) — `frontend/src/pages/UsersPage.tsx:112`
- **`loadDepartmentPage`** (Function) — `frontend/src/pages/UsersPage.tsx:254`
- **`loadDepartmentTree`** (Function) — `frontend/src/pages/UsersPage.tsx:273`
- **`loadDepartmentDetail`** (Function) — `frontend/src/pages/UsersPage.tsx:303`
- **`openUserCreateDrawer`** (Function) — `frontend/src/pages/UsersPage.tsx:363`

## Key Symbols

| Symbol | Type | File | Line |
|--------|------|------|------|
| `UsersPage` | Function | `frontend/src/pages/UsersPage.tsx` | 112 |
| `loadDepartmentPage` | Function | `frontend/src/pages/UsersPage.tsx` | 254 |
| `loadDepartmentTree` | Function | `frontend/src/pages/UsersPage.tsx` | 273 |
| `loadDepartmentDetail` | Function | `frontend/src/pages/UsersPage.tsx` | 303 |
| `openUserCreateDrawer` | Function | `frontend/src/pages/UsersPage.tsx` | 363 |
| `openUserEditDrawer` | Function | `frontend/src/pages/UsersPage.tsx` | 368 |
| `openUserDetailDrawer` | Function | `frontend/src/pages/UsersPage.tsx` | 373 |
| `openUserResetPasswordDrawer` | Function | `frontend/src/pages/UsersPage.tsx` | 377 |
| `openUserBindRolesDrawer` | Function | `frontend/src/pages/UsersPage.tsx` | 382 |
| `openUserBindDepartmentsDrawer` | Function | `frontend/src/pages/UsersPage.tsx` | 386 |
| `openDepartmentCreateDrawer` | Function | `frontend/src/pages/UsersPage.tsx` | 390 |
| `openDepartmentEditDrawer` | Function | `frontend/src/pages/UsersPage.tsx` | 395 |
| `openDepartmentDetailDrawer` | Function | `frontend/src/pages/UsersPage.tsx` | 400 |
| `openDepartmentBindMembersDrawer` | Function | `frontend/src/pages/UsersPage.tsx` | 404 |
| `toggleRoleSelection` | Function | `frontend/src/pages/UsersPage.tsx` | 440 |
| `toggleDepartmentSelection` | Function | `frontend/src/pages/UsersPage.tsx` | 446 |
| `toggleMemberUserSelection` | Function | `frontend/src/pages/UsersPage.tsx` | 452 |
| `handleDeleteDepartment` | Function | `frontend/src/pages/UsersPage.tsx` | 635 |
| `requestDeleteUser` | Function | `frontend/src/pages/UsersPage.tsx` | 653 |
| `requestDeleteDepartment` | Function | `frontend/src/pages/UsersPage.tsx` | 657 |

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
| Api | 61 calls |
| Hooks | 1 calls |

## How to Explore

1. `gitnexus_context({name: "UsersPage"})` — see callers and callees
2. `gitnexus_query({query: "pages"})` — find related execution flows
3. Read key files listed above for implementation details
