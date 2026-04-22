package cloud

import "testing"

func TestHuaweiProviderSyncAssetsMockByPrefix(t *testing.T) {
	provider := NewHuaweiProvider(HuaweiProviderOptions{
		MockEnabled:   false,
		MockAKPrefix:  "mock_",
		MockSKPrefix:  "mock_",
		DefaultRegion: "cn-north-4",
	})

	assets, err := provider.SyncAssets(Credentials{
		AccessKey: "mock_test-ak",
		SecretKey: "real-sk",
		Region:    "cn-south-1",
	})
	if err != nil {
		t.Fatalf("expected mock sync success, got err=%v", err)
	}
	if len(assets) == 0 {
		t.Fatalf("expected stub assets not empty")
	}
	if assets[0].Provider != "huawei" {
		t.Fatalf("expected provider=huawei got=%s", assets[0].Provider)
	}
	if assets[0].Region != "cn-south-1" {
		t.Fatalf("expected region=cn-south-1 got=%s", assets[0].Region)
	}
}

func TestHuaweiProviderVerifyMockEnabled(t *testing.T) {
	provider := NewHuaweiProvider(HuaweiProviderOptions{
		MockEnabled: true,
	})

	err := provider.Verify(Credentials{
		AccessKey: "ak",
		SecretKey: "sk",
	})
	if err != nil {
		t.Fatalf("expected verify success in mock mode, got err=%v", err)
	}
}

func TestHuaweiProviderBuildCredentialDefaultRegion(t *testing.T) {
	provider := NewHuaweiProvider(HuaweiProviderOptions{
		DefaultRegion: "cn-south-1",
	}).(HuaweiProvider)

	credential, region, err := provider.buildCredential(Credentials{
		AccessKey: "TEST_HUAWEI_AK",
		SecretKey: "test-sk",
	})
	if err != nil {
		t.Fatalf("build credential failed: %v", err)
	}
	if credential == nil {
		t.Fatalf("expected non-nil credential")
	}
	if region != "cn-south-1" {
		t.Fatalf("expected default region=cn-south-1 got=%s", region)
	}
}

func TestHuaweiProviderBuildCredentialGlobalRegionFallback(t *testing.T) {
	provider := NewHuaweiProvider(HuaweiProviderOptions{
		DefaultRegion: "cn-north-4",
	}).(HuaweiProvider)

	_, region, err := provider.buildCredential(Credentials{
		AccessKey: "TEST_HUAWEI_AK",
		SecretKey: "test-sk",
		Region:    "global",
	})
	if err != nil {
		t.Fatalf("build credential failed: %v", err)
	}
	if region != "cn-north-4" {
		t.Fatalf("expected fallback region=cn-north-4 got=%s", region)
	}
}

func TestHuaweiProviderResolveSyncRegionsExplicit(t *testing.T) {
	provider := NewHuaweiProvider(HuaweiProviderOptions{
		DefaultRegion: "cn-north-4",
	}).(HuaweiProvider)

	regions, err := provider.resolveSyncRegions("cn-south-1, cn-north-4,cn-south-1", "cn-north-4")
	if err != nil {
		t.Fatalf("resolveSyncRegions failed: %v", err)
	}
	if len(regions) != 2 {
		t.Fatalf("expected 2 regions got=%d", len(regions))
	}
	if regions[0] != "cn-south-1" || regions[1] != "cn-north-4" {
		t.Fatalf("unexpected regions=%v", regions)
	}
}

func TestHuaweiOBSEndpoint(t *testing.T) {
	if endpoint := huaweiOBSEndpoint("eu-west-101"); endpoint != "https://obs.eu-west-101.myhuaweicloud.eu" {
		t.Fatalf("unexpected eu obs endpoint=%s", endpoint)
	}
	if endpoint := huaweiOBSEndpoint("cn-north-4"); endpoint != "https://obs.cn-north-4.myhuaweicloud.com" {
		t.Fatalf("unexpected cn obs endpoint=%s", endpoint)
	}
}
