---
name: ws
description: "Skill for the Ws area of aiops. 6 symbols across 1 files."
---

# Ws

6 symbols | 1 files | Cohesion: 83%

## When to Use

- Working with code in `backend/`
- Understanding how NewHub, Publish work
- Modifying ws-related functionality

## Key Files

| File | Symbols |
|------|---------|
| `backend/internal/ws/hub.go` | NewHub, Publish, broadcast, allow, consumeRedis (+1) |

## Entry Points

Start here when exploring this area:

- **`NewHub`** (Function) — `backend/internal/ws/hub.go:45`
- **`Publish`** (Method) — `backend/internal/ws/hub.go:72`

## Key Symbols

| Symbol | Type | File | Line |
|--------|------|------|------|
| `NewHub` | Function | `backend/internal/ws/hub.go` | 45 |
| `Publish` | Method | `backend/internal/ws/hub.go` | 72 |
| `allow` | Function | `backend/internal/ws/hub.go` | 106 |
| `buildNodeID` | Function | `backend/internal/ws/hub.go` | 147 |
| `broadcast` | Method | `backend/internal/ws/hub.go` | 92 |
| `consumeRedis` | Method | `backend/internal/ws/hub.go` | 121 |

## How to Explore

1. `gitnexus_context({name: "NewHub"})` — see callers and callees
2. `gitnexus_query({query: "ws"})` — find related execution flows
3. Read key files listed above for implementation details
