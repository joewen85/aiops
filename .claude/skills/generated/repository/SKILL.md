---
name: repository
description: "Skill for the Repository area of aiops. 4 symbols across 2 files."
---

# Repository

4 symbols | 2 files | Cohesion: 100%

## When to Use

- Working with code in `backend/`
- Understanding how DB, GetByID, Update work
- Modifying repository-related functionality

## Key Files

| File | Symbols |
|------|---------|
| `backend/internal/repository/gorm_repository.go` | DB, GetByID, Update |
| `backend/internal/middlewaresvc/driver.go` | sqlDB |

## Entry Points

Start here when exploring this area:

- **`DB`** (Method) — `backend/internal/repository/gorm_repository.go:18`
- **`GetByID`** (Method) — `backend/internal/repository/gorm_repository.go:26`
- **`Update`** (Method) — `backend/internal/repository/gorm_repository.go:34`

## Key Symbols

| Symbol | Type | File | Line |
|--------|------|------|------|
| `DB` | Method | `backend/internal/repository/gorm_repository.go` | 18 |
| `GetByID` | Method | `backend/internal/repository/gorm_repository.go` | 26 |
| `Update` | Method | `backend/internal/repository/gorm_repository.go` | 34 |
| `sqlDB` | Function | `backend/internal/middlewaresvc/driver.go` | 271 |

## How to Explore

1. `gitnexus_context({name: "DB"})` — see callers and callees
2. `gitnexus_query({query: "repository"})` — find related execution flows
3. Read key files listed above for implementation details
