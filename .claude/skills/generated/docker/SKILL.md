---
name: docker
description: "Skill for the Docker area of aiops. 23 symbols across 4 files."
---

# Docker

23 symbols | 4 files | Cohesion: 84%

## When to Use

- Working with code in `backend/`
- Understanding how PingRedis, Ping, ContainerAction work
- Modifying docker-related functionality

## Key Files

| File | Symbols |
|------|---------|
| `backend/internal/docker/client.go` | Ping, ContainerAction, RemoveImage, RemoveNetwork, RemoveVolume (+12) |
| `backend/internal/handler/docker_modules.go` | executeDockerAction, composeStackUpdatesForAction, loadDockerResources |
| `backend/internal/handler/cmdb_sync_lock.go` | tryAcquireCMDBSyncLock, generateLockToken |
| `backend/internal/db/db.go` | PingRedis |

## Entry Points

Start here when exploring this area:

- **`PingRedis`** (Function) — `backend/internal/db/db.go:37`
- **`Ping`** (Method) — `backend/internal/docker/client.go:107`
- **`ContainerAction`** (Method) — `backend/internal/docker/client.go:175`
- **`RemoveImage`** (Method) — `backend/internal/docker/client.go:197`
- **`RemoveNetwork`** (Method) — `backend/internal/docker/client.go:213`

## Key Symbols

| Symbol | Type | File | Line |
|--------|------|------|------|
| `PingRedis` | Function | `backend/internal/db/db.go` | 37 |
| `Ping` | Method | `backend/internal/docker/client.go` | 107 |
| `ContainerAction` | Method | `backend/internal/docker/client.go` | 175 |
| `RemoveImage` | Method | `backend/internal/docker/client.go` | 197 |
| `RemoveNetwork` | Method | `backend/internal/docker/client.go` | 213 |
| `RemoveVolume` | Method | `backend/internal/docker/client.go` | 229 |
| `Version` | Method | `backend/internal/docker/client.go` | 130 |
| `ListContainers` | Method | `backend/internal/docker/client.go` | 138 |
| `ListImages` | Method | `backend/internal/docker/client.go` | 146 |
| `ListNetworks` | Method | `backend/internal/docker/client.go` | 154 |
| `ListVolumes` | Method | `backend/internal/docker/client.go` | 162 |
| `Run` | Method | `backend/internal/docker/client.go` | 249 |
| `composeStackUpdatesForAction` | Function | `backend/internal/handler/docker_modules.go` | 924 |
| `responseError` | Function | `backend/internal/docker/client.go` | 395 |
| `loadDockerResources` | Function | `backend/internal/handler/docker_modules.go` | 699 |
| `generateLockToken` | Function | `backend/internal/handler/cmdb_sync_lock.go` | 66 |
| `composeArgsForAction` | Function | `backend/internal/docker/client.go` | 357 |
| `composeEnv` | Function | `backend/internal/docker/client.go` | 372 |
| `sanitizeComposeProjectName` | Function | `backend/internal/docker/client.go` | 384 |
| `executeDockerAction` | Method | `backend/internal/handler/docker_modules.go` | 583 |

## Execution Flows

| Flow | Type | Steps |
|------|------|-------|
| `ExecuteDockerAction → NormalizeEndpoint` | cross_community | 6 |
| `ExecuteDockerAction → Client` | cross_community | 6 |

## Connected Areas

| Area | Connections |
|------|-------------|
| Handler | 1 calls |

## How to Explore

1. `gitnexus_context({name: "PingRedis"})` — see callers and callees
2. `gitnexus_query({query: "docker"})` — find related execution flows
3. Read key files listed above for implementation details
