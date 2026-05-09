---
name: cloud
description: "Skill for the Cloud area of aiops. 165 symbols across 15 files."
---

# Cloud

165 symbols | 15 files | Cohesion: 72%

## When to Use

- Working with code in `backend/`
- Understanding how TestNormalizeTimeText, TestAliyunProviderResolveSyncRegionsExplicit, TestParseAliyunCSClusters work
- Modifying cloud-related functionality

## Key Files

| File | Symbols |
|------|---------|
| `backend/internal/cloud/aliyun_provider.go` | Verify, SyncAssets, collectRDSAssets, collectVPCAssets, collectSLBAssets (+35) |
| `backend/internal/cloud/tencent_provider.go` | normalizeTimeText, Verify, SyncAssets, resolveSyncRegions, discoverTencentRegions (+30) |
| `backend/internal/cloud/huawei_provider.go` | Verify, SyncAssets, collectOBSAssets, shouldMock, wrapSDKError (+25) |
| `backend/internal/cloud/aws_provider.go` | Verify, SyncAssets, collectEC2Assets, describeEC2InstanceTypes, collectRDSAssets (+18) |
| `backend/internal/cloud/aliyun_provider_test.go` | TestAliyunProviderResolveSyncRegionsExplicit, TestParseAliyunCSClusters, TestNormalizeAliyunUnixText, TestAliyunLogServiceEndpoint, TestAliyunProviderBuildCredentialDefaultRegion (+2) |
| `backend/internal/cloud/tencent_provider_test.go` | TestNormalizeTimeText, TestTencentProviderSyncAssetsMockByPrefix, TestTencentProviderBuildCredentialDefaultRegion, TestTencentProviderBuildCredentialGlobalRegionFallback, TestTencentProviderVerifyMockEnabled (+1) |
| `backend/internal/cloud/huawei_provider_test.go` | TestHuaweiOBSEndpoint, TestHuaweiProviderBuildCredentialDefaultRegion, TestHuaweiProviderBuildCredentialGlobalRegionFallback, TestHuaweiProviderResolveSyncRegionsExplicit, TestHuaweiProviderSyncAssetsMockByPrefix (+1) |
| `backend/internal/cloud/aws_provider_test.go` | TestAWSS3BucketRegionFallback, TestAWSProviderSyncAssetsMockByPrefix, TestAWSProviderBuildConfigDefaultRegion, TestAWSProviderBuildConfigGlobalRegionFallback, TestAWSProviderResolveSyncRegionsExplicit |
| `backend/internal/cloud/memory_units.go` | formatMemoryGB, formatMemoryMBToGB, formatMemoryFloatGB |
| `backend/internal/cloud/provider.go` | NewStubProvider, Verify, SyncAssets |

## Entry Points

Start here when exploring this area:

- **`TestNormalizeTimeText`** (Function) — `backend/internal/cloud/tencent_provider_test.go:83`
- **`TestAliyunProviderResolveSyncRegionsExplicit`** (Function) — `backend/internal/cloud/aliyun_provider_test.go:66`
- **`TestParseAliyunCSClusters`** (Function) — `backend/internal/cloud/aliyun_provider_test.go:83`
- **`TestNormalizeAliyunUnixText`** (Function) — `backend/internal/cloud/aliyun_provider_test.go:94`
- **`TestAliyunLogServiceEndpoint`** (Function) — `backend/internal/cloud/aliyun_provider_test.go:101`

## Key Symbols

| Symbol | Type | File | Line |
|--------|------|------|------|
| `TestNormalizeTimeText` | Function | `backend/internal/cloud/tencent_provider_test.go` | 83 |
| `TestAliyunProviderResolveSyncRegionsExplicit` | Function | `backend/internal/cloud/aliyun_provider_test.go` | 66 |
| `TestParseAliyunCSClusters` | Function | `backend/internal/cloud/aliyun_provider_test.go` | 83 |
| `TestNormalizeAliyunUnixText` | Function | `backend/internal/cloud/aliyun_provider_test.go` | 94 |
| `TestAliyunLogServiceEndpoint` | Function | `backend/internal/cloud/aliyun_provider_test.go` | 101 |
| `TestFormatMemoryGB` | Function | `backend/internal/cloud/memory_units_test.go` | 4 |
| `TestAWSS3BucketRegionFallback` | Function | `backend/internal/cloud/aws_provider_test.go` | 90 |
| `TestHuaweiOBSEndpoint` | Function | `backend/internal/cloud/huawei_provider_test.go` | 100 |
| `TestFormatMemoryMBToGB` | Function | `backend/internal/cloud/memory_units_test.go` | 13 |
| `TestHuaweiProviderBuildCredentialDefaultRegion` | Function | `backend/internal/cloud/huawei_provider_test.go` | 45 |
| `TestHuaweiProviderBuildCredentialGlobalRegionFallback` | Function | `backend/internal/cloud/huawei_provider_test.go` | 65 |
| `TestHuaweiProviderResolveSyncRegionsExplicit` | Function | `backend/internal/cloud/huawei_provider_test.go` | 83 |
| `NewHuaweiProvider` | Function | `backend/internal/cloud/huawei_provider.go` | 47 |
| `TestCloudProviderByName` | Function | `backend/internal/handler/cloud_provider_helpers_test.go` | 10 |
| `NewStubProvider` | Function | `backend/internal/cloud/provider.go` | 32 |
| `TestAliyunProviderBuildCredentialDefaultRegion` | Function | `backend/internal/cloud/aliyun_provider_test.go` | 28 |
| `TestAliyunProviderBuildCredentialGlobalRegionFallback` | Function | `backend/internal/cloud/aliyun_provider_test.go` | 48 |
| `NewAliyunProvider` | Function | `backend/internal/cloud/aliyun_provider.go` | 46 |
| `TestTencentProviderSyncAssetsMockByPrefix` | Function | `backend/internal/cloud/tencent_provider_test.go` | 4 |
| `TestTencentProviderBuildCredentialDefaultRegion` | Function | `backend/internal/cloud/tencent_provider_test.go` | 45 |

## Execution Flows

| Flow | Type | Steps |
|------|------|-------|
| `CreateCloudAsset → NormalizeBaseResourceType` | cross_community | 4 |

## Connected Areas

| Area | Connections |
|------|-------------|
| Executor | 2 calls |

## How to Explore

1. `gitnexus_context({name: "TestNormalizeTimeText"})` — see callers and callees
2. `gitnexus_query({query: "cloud"})` — find related execution flows
3. Read key files listed above for implementation details
