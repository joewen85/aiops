package cloud

import "testing"

func TestTencentProviderSyncAssetsMockByPrefix(t *testing.T) {
	provider := NewTencentProvider(TencentProviderOptions{
		MockEnabled:   false,
		MockAKPrefix:  "mock_",
		MockSKPrefix:  "mock_",
		DefaultRegion: "ap-guangzhou",
	})

	assets, err := provider.SyncAssets(Credentials{
		AccessKey: "mock_test-ak",
		SecretKey: "real-sk",
		Region:    "ap-shanghai",
	})
	if err != nil {
		t.Fatalf("expected mock sync success, got err=%v", err)
	}
	if len(assets) == 0 {
		t.Fatalf("expected stub assets not empty")
	}
	if assets[0].Provider != "tencent" {
		t.Fatalf("expected provider=tencent got=%s", assets[0].Provider)
	}
	if assets[0].Region != "ap-shanghai" {
		t.Fatalf("expected region=ap-shanghai got=%s", assets[0].Region)
	}
}

func TestTencentProviderVerifyMockEnabled(t *testing.T) {
	provider := NewTencentProvider(TencentProviderOptions{
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

func TestTencentProviderBuildCredentialDefaultRegion(t *testing.T) {
	provider := NewTencentProvider(TencentProviderOptions{
		DefaultRegion: "ap-nanjing",
	}).(TencentProvider)

	credential, region, err := provider.buildCredential(Credentials{
		AccessKey: "AKID_TEST_DEFAULT_REGION",
		SecretKey: "sk",
	})
	if err != nil {
		t.Fatalf("build credential failed: %v", err)
	}
	if credential == nil {
		t.Fatalf("expected non-nil credential")
	}
	if region != "ap-nanjing" {
		t.Fatalf("expected default region=ap-nanjing got=%s", region)
	}
}

func TestTencentProviderBuildCredentialGlobalRegionFallback(t *testing.T) {
	provider := NewTencentProvider(TencentProviderOptions{
		DefaultRegion: "ap-guangzhou",
	}).(TencentProvider)

	_, region, err := provider.buildCredential(Credentials{
		AccessKey: "AKID_TEST_REGION_FALLBACK",
		SecretKey: "sk",
		Region:    "global",
	})
	if err != nil {
		t.Fatalf("build credential failed: %v", err)
	}
	if region != "ap-guangzhou" {
		t.Fatalf("expected fallback region=ap-guangzhou got=%s", region)
	}
}

func TestNormalizeTimeText(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "rfc3339", input: "2026-01-02T03:04:05Z", want: "2026-01-02T03:04:05Z"},
		{name: "datetime", input: "2026-01-02 11:22:33", want: "2026-01-02T11:22:33Z"},
		{name: "date", input: "2026-01-02", want: "2026-01-02T00:00:00Z"},
		{name: "raw", input: "not-a-time", want: "not-a-time"},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			got := normalizeTimeText(item.input)
			if got != item.want {
				t.Fatalf("input=%s expected=%s got=%s", item.input, item.want, got)
			}
		})
	}
}

func TestTencentStatusMappings(t *testing.T) {
	if cdbStatusText(1) != "running" {
		t.Fatalf("expected cdb status running")
	}
	if cdbStatusText(0) != "creating" {
		t.Fatalf("expected cdb status creating")
	}
	if cdbStatusText(999) != "unknown" {
		t.Fatalf("expected cdb status unknown")
	}
	if clbStatusText(1) != "running" {
		t.Fatalf("expected clb status running")
	}
	if clbStatusText(0) != "creating" {
		t.Fatalf("expected clb status creating")
	}
	if clbStatusText(99) != "unknown" {
		t.Fatalf("expected clb status unknown")
	}
	if clsStatusText(true) != "running" {
		t.Fatalf("expected cls status running")
	}
	if clsStatusText(false) != "stopped" {
		t.Fatalf("expected cls status stopped")
	}
}
