---
name: hooks
description: "Skill for the Hooks area of aiops. 12 symbols across 4 files."
---

# Hooks

12 symbols | 4 files | Cohesion: 91%

## When to Use

- Working with code in `frontend/`
- Understanding how getToken, useWebSocket, listener work
- Modifying hooks-related functionality

## Key Files

| File | Symbols |
|------|---------|
| `frontend/src/hooks/useWebSocket.ts` | wsDebugLog, emitMessage, closeSharedSocket, reconnectDelay, scheduleSharedSocket (+4) |
| `frontend/src/store/auth.ts` | getToken |
| `frontend/src/pages/DashboardPage.tsx` | DashboardPage |
| `frontend/src/components/ProtectedRoute.tsx` | ProtectedRoute |

## Entry Points

Start here when exploring this area:

- **`getToken`** (Function) — `frontend/src/store/auth.ts:6`
- **`useWebSocket`** (Function) — `frontend/src/hooks/useWebSocket.ts:146`
- **`listener`** (Function) — `frontend/src/hooks/useWebSocket.ts:164`
- **`DashboardPage`** (Function) — `frontend/src/pages/DashboardPage.tsx:2`
- **`ProtectedRoute`** (Function) — `frontend/src/components/ProtectedRoute.tsx:3`

## Key Symbols

| Symbol | Type | File | Line |
|--------|------|------|------|
| `getToken` | Function | `frontend/src/store/auth.ts` | 6 |
| `useWebSocket` | Function | `frontend/src/hooks/useWebSocket.ts` | 146 |
| `listener` | Function | `frontend/src/hooks/useWebSocket.ts` | 164 |
| `DashboardPage` | Function | `frontend/src/pages/DashboardPage.tsx` | 2 |
| `ProtectedRoute` | Function | `frontend/src/components/ProtectedRoute.tsx` | 3 |
| `wsDebugLog` | Function | `frontend/src/hooks/useWebSocket.ts` | 29 |
| `emitMessage` | Function | `frontend/src/hooks/useWebSocket.ts` | 38 |
| `closeSharedSocket` | Function | `frontend/src/hooks/useWebSocket.ts` | 42 |
| `reconnectDelay` | Function | `frontend/src/hooks/useWebSocket.ts` | 57 |
| `scheduleSharedSocket` | Function | `frontend/src/hooks/useWebSocket.ts` | 63 |
| `ensureSharedSocket` | Function | `frontend/src/hooks/useWebSocket.ts` | 76 |
| `openSharedSocket` | Function | `frontend/src/hooks/useWebSocket.ts` | 92 |

## Connected Areas

| Area | Connections |
|------|-------------|
| Handler | 1 calls |
| Pages | 1 calls |

## How to Explore

1. `gitnexus_context({name: "getToken"})` — see callers and callees
2. `gitnexus_query({query: "hooks"})` — find related execution flows
3. Read key files listed above for implementation details
