---
name: executor
description: "Skill for the Executor area of aiops. 5 symbols across 3 files."
---

# Executor

5 symbols | 3 files | Cohesion: 67%

## When to Use

- Working with code in `backend/`
- Understanding how Run, Error work
- Modifying executor-related functionality

## Key Files

| File | Symbols |
|------|---------|
| `backend/internal/executor/ansible.go` | Run, prepareFiles, errString |
| `backend/internal/errors/errors.go` | Error |
| `backend/cmd/server/main.go` | main |

## Entry Points

Start here when exploring this area:

- **`Run`** (Method) — `backend/internal/executor/ansible.go:37`
- **`Error`** (Method) — `backend/internal/errors/errors.go:7`

## Key Symbols

| Symbol | Type | File | Line |
|--------|------|------|------|
| `Run` | Method | `backend/internal/executor/ansible.go` | 37 |
| `Error` | Method | `backend/internal/errors/errors.go` | 7 |
| `errString` | Function | `backend/internal/executor/ansible.go` | 113 |
| `main` | Function | `backend/cmd/server/main.go` | 8 |
| `prepareFiles` | Method | `backend/internal/executor/ansible.go` | 92 |

## Execution Flows

| Flow | Type | Steps |
|------|------|-------|
| `CreateCloudAccount → Error` | cross_community | 3 |
| `VerifyCloudAccount → Error` | cross_community | 3 |

## Connected Areas

| Area | Connections |
|------|-------------|
| Middleware | 1 calls |

## How to Explore

1. `gitnexus_context({name: "Run"})` — see callers and callees
2. `gitnexus_query({query: "executor"})` — find related execution flows
3. Read key files listed above for implementation details
