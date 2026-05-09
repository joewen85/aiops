---
name: middlewaresvc
description: "Skill for the Middlewaresvc area of aiops. 35 symbols across 5 files."
---

# Middlewaresvc

35 symbols | 5 files | Cohesion: 90%

## When to Use

- Working with code in `backend/`
- Understanding how DriverFor, NormalizeType, ValidateEndpoint work
- Modifying middlewaresvc-related functionality

## Key Files

| File | Symbols |
|------|---------|
| `backend/internal/middlewaresvc/driver.go` | redisOptions, postgresDSN, rabbitURL, tlsConfigIfEnabled, isMock (+11) |
| `backend/internal/middlewaresvc/redis.go` | Check, CollectMetrics, Execute, client, parseRedisInfo (+1) |
| `backend/internal/middlewaresvc/rabbitmq.go` | Check, CollectMetrics, Execute, open, stringParam |
| `backend/internal/middlewaresvc/postgres.go` | Check, CollectMetrics, Execute, open |
| `backend/internal/handler/middleware_modules.go` | MiddlewareAIOpsProtocol, buildMiddlewareInstance, validateMiddlewareInstance, middlewareProtocol |

## Entry Points

Start here when exploring this area:

- **`DriverFor`** (Function) — `backend/internal/middlewaresvc/driver.go:71`
- **`NormalizeType`** (Function) — `backend/internal/middlewaresvc/driver.go:84`
- **`ValidateEndpoint`** (Function) — `backend/internal/middlewaresvc/driver.go:97`
- **`Check`** (Method) — `backend/internal/middlewaresvc/redis.go:22`
- **`CollectMetrics`** (Method) — `backend/internal/middlewaresvc/redis.go:48`

## Key Symbols

| Symbol | Type | File | Line |
|--------|------|------|------|
| `DriverFor` | Function | `backend/internal/middlewaresvc/driver.go` | 71 |
| `NormalizeType` | Function | `backend/internal/middlewaresvc/driver.go` | 84 |
| `ValidateEndpoint` | Function | `backend/internal/middlewaresvc/driver.go` | 97 |
| `Check` | Method | `backend/internal/middlewaresvc/redis.go` | 22 |
| `CollectMetrics` | Method | `backend/internal/middlewaresvc/redis.go` | 48 |
| `Execute` | Method | `backend/internal/middlewaresvc/redis.go` | 68 |
| `Check` | Method | `backend/internal/middlewaresvc/rabbitmq.go` | 22 |
| `CollectMetrics` | Method | `backend/internal/middlewaresvc/rabbitmq.go` | 41 |
| `Execute` | Method | `backend/internal/middlewaresvc/rabbitmq.go` | 60 |
| `Check` | Method | `backend/internal/middlewaresvc/postgres.go` | 23 |
| `CollectMetrics` | Method | `backend/internal/middlewaresvc/postgres.go` | 49 |
| `Execute` | Method | `backend/internal/middlewaresvc/postgres.go` | 68 |
| `MiddlewareAIOpsProtocol` | Method | `backend/internal/handler/middleware_modules.go` | 330 |
| `parseRedisInfo` | Function | `backend/internal/middlewaresvc/redis.go` | 101 |
| `truncateString` | Function | `backend/internal/middlewaresvc/redis.go` | 111 |
| `stringParam` | Function | `backend/internal/middlewaresvc/rabbitmq.go` | 101 |
| `redisOptions` | Function | `backend/internal/middlewaresvc/driver.go` | 167 |
| `postgresDSN` | Function | `backend/internal/middlewaresvc/driver.go` | 193 |
| `rabbitURL` | Function | `backend/internal/middlewaresvc/driver.go` | 211 |
| `tlsConfigIfEnabled` | Function | `backend/internal/middlewaresvc/driver.go` | 225 |

## Execution Flows

| Flow | Type | Steps |
|------|------|-------|
| `MiddlewareAIOpsProtocol → NormalizeType` | intra_community | 4 |
| `MiddlewareAIOpsProtocol → RedisDriver` | intra_community | 4 |
| `MiddlewareAIOpsProtocol → PostgresDriver` | intra_community | 4 |
| `MiddlewareAIOpsProtocol → RabbitMQDriver` | intra_community | 4 |

## Connected Areas

| Area | Connections |
|------|-------------|
| Handler | 2 calls |
| Docker | 1 calls |

## How to Explore

1. `gitnexus_context({name: "DriverFor"})` — see callers and callees
2. `gitnexus_query({query: "middlewaresvc"})` — find related execution flows
3. Read key files listed above for implementation details
