---
name: config
description: "Skill for the Config area of aiops. 5 symbols across 1 files."
---

# Config

5 symbols | 1 files | Cohesion: 89%

## When to Use

- Working with code in `backend/`
- Understanding how Load work
- Modifying config-related functionality

## Key Files

| File | Symbols |
|------|---------|
| `backend/internal/config/config.go` | Load, loadEnvFiles, env, envInt, envBool |

## Entry Points

Start here when exploring this area:

- **`Load`** (Function) — `backend/internal/config/config.go:57`

## Key Symbols

| Symbol | Type | File | Line |
|--------|------|------|------|
| `Load` | Function | `backend/internal/config/config.go` | 57 |
| `loadEnvFiles` | Function | `backend/internal/config/config.go` | 101 |
| `env` | Function | `backend/internal/config/config.go` | 108 |
| `envInt` | Function | `backend/internal/config/config.go` | 115 |
| `envBool` | Function | `backend/internal/config/config.go` | 127 |

## How to Explore

1. `gitnexus_context({name: "Load"})` — see callers and callees
2. `gitnexus_query({query: "config"})` — find related execution flows
3. Read key files listed above for implementation details
