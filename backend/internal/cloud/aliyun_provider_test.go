package cloud

import "testing"

func TestAliyunProviderSyncAssetsMockByPrefix(t *testing.T) {
	provider := NewAliyunProvider(AliyunProviderOptions{
		MockEnabled:   false,
		MockAKPrefix:  "mock_",
		MockSKPrefix:  "mock_",
		DefaultRegion: "cn-hangzhou",
	})

	assets, err := provider.SyncAssets(Credentials{
		AccessKey: "mock_test-ak",
		SecretKey: "real-sk",
		Region:    "cn-shanghai",
	})
	if err != nil {
		t.Fatalf("expected mock sync success, got err=%v", err)
	}
	if len(assets) == 0 {
		t.Fatalf("expected stub assets not empty")
	}
	if assets[0].Provider != "aliyun" {
		t.Fatalf("expected provider=aliyun got=%s", assets[0].Provider)
	}
}

func TestAliyunProviderBuildCredentialDefaultRegion(t *testing.T) {
	provider := NewAliyunProvider(AliyunProviderOptions{
		DefaultRegion: "cn-beijing",
	}).(AliyunProvider)

	accessKey, secretKey, region, err := provider.buildCredential(Credentials{
		AccessKey: "LTAI_TEST_AK",
		SecretKey: "test-sk",
	})
	if err != nil {
		t.Fatalf("build credential failed: %v", err)
	}
	if accessKey != "LTAI_TEST_AK" || secretKey != "test-sk" {
		t.Fatalf("unexpected credential parsed")
	}
	if region != "cn-beijing" {
		t.Fatalf("expected default region=cn-beijing got=%s", region)
	}
}

func TestAliyunProviderBuildCredentialGlobalRegionFallback(t *testing.T) {
	provider := NewAliyunProvider(AliyunProviderOptions{
		DefaultRegion: "cn-hangzhou",
	}).(AliyunProvider)

	_, _, region, err := provider.buildCredential(Credentials{
		AccessKey: "LTAI_TEST_AK",
		SecretKey: "test-sk",
		Region:    "global",
	})
	if err != nil {
		t.Fatalf("build credential failed: %v", err)
	}
	if region != "cn-hangzhou" {
		t.Fatalf("expected fallback region=cn-hangzhou got=%s", region)
	}
}

func TestAliyunProviderResolveSyncRegionsExplicit(t *testing.T) {
	provider := NewAliyunProvider(AliyunProviderOptions{
		DefaultRegion: "cn-hangzhou",
	}).(AliyunProvider)

	regions, err := provider.resolveSyncRegions("ak", "sk", "cn-shanghai, cn-hangzhou ,cn-shanghai", "cn-hangzhou")
	if err != nil {
		t.Fatalf("resolveSyncRegions failed: %v", err)
	}
	if len(regions) != 2 {
		t.Fatalf("expected 2 regions got=%d", len(regions))
	}
	if regions[0] != "cn-shanghai" || regions[1] != "cn-hangzhou" {
		t.Fatalf("unexpected regions=%v", regions)
	}
}

func TestParseAliyunCSClusters(t *testing.T) {
	raw := []byte(`[{"cluster_id":"c-1","name":"prod-1","region_id":"cn-hangzhou"}]`)
	items := parseAliyunCSClusters(raw)
	if len(items) != 1 {
		t.Fatalf("expected one cluster item")
	}
	if items[0]["cluster_id"] != "c-1" {
		t.Fatalf("unexpected cluster_id=%s", items[0]["cluster_id"])
	}
}

func TestNormalizeAliyunUnixText(t *testing.T) {
	got := normalizeAliyunUnixText("1700000000")
	if got != "2023-11-14T22:13:20Z" {
		t.Fatalf("unexpected normalized unix text=%s", got)
	}
}

func TestAliyunLogServiceEndpoint(t *testing.T) {
	endpoint := aliyunLogServiceEndpoint("cn-shanghai")
	if endpoint != "https://cn-shanghai.log.aliyuncs.com" {
		t.Fatalf("unexpected endpoint=%s", endpoint)
	}
	fallback := aliyunLogServiceEndpoint("global")
	if fallback != "https://cn-hangzhou.log.aliyuncs.com" {
		t.Fatalf("unexpected fallback endpoint=%s", fallback)
	}
}
