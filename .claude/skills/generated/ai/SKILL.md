---
name: ai
description: "Skill for the Ai area of aiops. 13 symbols across 2 files."
---

# Ai

13 symbols | 2 files | Cohesion: 80%

## When to Use

- Working with code in `backend/`
- Understanding how ParseIntent, BuildPlan, ExecutePlan work
- Modifying ai-related functionality

## Key Files

| File | Symbols |
|------|---------|
| `backend/internal/ai/procurement.go` | ParseIntent, normalizeProvider, normalizeResourceType, normalizeAction, normalizeRegion (+5) |
| `backend/internal/ai/provider.go` | Chat, Chat, postJSON |

## Entry Points

Start here when exploring this area:

- **`ParseIntent`** (Method) — `backend/internal/ai/procurement.go:97`
- **`BuildPlan`** (Method) — `backend/internal/ai/procurement.go:133`
- **`ExecutePlan`** (Method) — `backend/internal/ai/procurement.go:195`
- **`Chat`** (Method) — `backend/internal/ai/provider.go:21`
- **`Chat`** (Method) — `backend/internal/ai/provider.go:31`

## Key Symbols

| Symbol | Type | File | Line |
|--------|------|------|------|
| `ParseIntent` | Method | `backend/internal/ai/procurement.go` | 97 |
| `BuildPlan` | Method | `backend/internal/ai/procurement.go` | 133 |
| `ExecutePlan` | Method | `backend/internal/ai/procurement.go` | 195 |
| `Chat` | Method | `backend/internal/ai/provider.go` | 21 |
| `Chat` | Method | `backend/internal/ai/provider.go` | 31 |
| `normalizeProvider` | Function | `backend/internal/ai/procurement.go` | 219 |
| `normalizeResourceType` | Function | `backend/internal/ai/procurement.go` | 240 |
| `normalizeAction` | Function | `backend/internal/ai/procurement.go` | 268 |
| `normalizeRegion` | Function | `backend/internal/ai/procurement.go` | 280 |
| `normalizeQuantity` | Function | `backend/internal/ai/procurement.go` | 288 |
| `estimateCost` | Function | `backend/internal/ai/procurement.go` | 312 |
| `buildID` | Function | `backend/internal/ai/procurement.go` | 335 |
| `postJSON` | Function | `backend/internal/ai/provider.go` | 35 |

## Connected Areas

| Area | Connections |
|------|-------------|
| Cloud | 1 calls |

## How to Explore

1. `gitnexus_context({name: "ParseIntent"})` — see callers and callees
2. `gitnexus_query({query: "ai"})` — find related execution flows
3. Read key files listed above for implementation details
