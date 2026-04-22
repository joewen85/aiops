package cloud

import (
	"testing"

	aws "github.com/aws/aws-sdk-go-v2/aws"
)

func TestAWSProviderSyncAssetsMockByPrefix(t *testing.T) {
	provider := NewAWSProvider(AWSProviderOptions{
		MockEnabled:   false,
		MockAKPrefix:  "mock_",
		MockSKPrefix:  "mock_",
		DefaultRegion: "us-east-1",
	})

	assets, err := provider.SyncAssets(Credentials{
		AccessKey: "mock_test-ak",
		SecretKey: "real-sk",
		Region:    "ap-southeast-1",
	})
	if err != nil {
		t.Fatalf("expected mock sync success, got err=%v", err)
	}
	if len(assets) == 0 {
		t.Fatalf("expected stub assets not empty")
	}
	if assets[0].Provider != "aws" {
		t.Fatalf("expected provider=aws got=%s", assets[0].Provider)
	}
	if assets[0].Region != "ap-southeast-1" {
		t.Fatalf("expected region=ap-southeast-1 got=%s", assets[0].Region)
	}
}

func TestAWSProviderBuildConfigDefaultRegion(t *testing.T) {
	provider := NewAWSProvider(AWSProviderOptions{
		DefaultRegion: "ap-southeast-1",
	}).(AWSProvider)

	cfg, region, err := provider.buildConfig(Credentials{
		AccessKey: "AKIA_TEST_EXAMPLE",
		SecretKey: "test-secret",
	})
	if err != nil {
		t.Fatalf("build config failed: %v", err)
	}
	if cfg.Region != "ap-southeast-1" {
		t.Fatalf("expected cfg region=ap-southeast-1 got=%s", cfg.Region)
	}
	if region != "ap-southeast-1" {
		t.Fatalf("expected default region=ap-southeast-1 got=%s", region)
	}
}

func TestAWSProviderBuildConfigGlobalRegionFallback(t *testing.T) {
	provider := NewAWSProvider(AWSProviderOptions{
		DefaultRegion: "us-east-1",
	}).(AWSProvider)

	_, region, err := provider.buildConfig(Credentials{
		AccessKey: "AKIA_TEST_EXAMPLE",
		SecretKey: "test-secret",
		Region:    "global",
	})
	if err != nil {
		t.Fatalf("build config failed: %v", err)
	}
	if region != "us-east-1" {
		t.Fatalf("expected fallback region=us-east-1 got=%s", region)
	}
}

func TestAWSProviderResolveSyncRegionsExplicit(t *testing.T) {
	provider := NewAWSProvider(AWSProviderOptions{
		DefaultRegion: "us-east-1",
	}).(AWSProvider)

	regions, err := provider.resolveSyncRegions(aws.Config{}, "ap-southeast-1, us-east-1,ap-southeast-1", "us-east-1")
	if err != nil {
		t.Fatalf("resolveSyncRegions failed: %v", err)
	}
	if len(regions) != 2 {
		t.Fatalf("expected 2 regions got=%d", len(regions))
	}
	if regions[0] != "ap-southeast-1" || regions[1] != "us-east-1" {
		t.Fatalf("unexpected regions=%v", regions)
	}
}

func TestAWSS3BucketRegionFallback(t *testing.T) {
	if region := awsS3BucketRegion("", "ap-southeast-1"); region != "us-east-1" {
		t.Fatalf("expected empty region fallback to us-east-1 got=%s", region)
	}
	if region := awsS3BucketRegion("EU", "us-east-1"); region != "eu-west-1" {
		t.Fatalf("expected EU => eu-west-1 got=%s", region)
	}
}
