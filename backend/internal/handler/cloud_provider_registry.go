package handler

import (
	"strings"

	"devops-system/backend/internal/cloud"
	"devops-system/backend/internal/config"
)

func buildCloudProviders(cfg config.Config) map[string]cloud.Provider {
	production := strings.EqualFold(strings.TrimSpace(cfg.AppEnv), "production")
	mockEnabled := cfg.CloudSDKMockEnabled && !production
	mockAKPrefix := cfg.CloudSDKMockAKPrefix
	mockSKPrefix := cfg.CloudSDKMockSKPrefix
	if production {
		mockAKPrefix = ""
		mockSKPrefix = ""
	}
	return map[string]cloud.Provider{
		"aws": cloud.NewAWSProvider(cloud.AWSProviderOptions{
			MockEnabled:     mockEnabled,
			MockAKPrefix:    mockAKPrefix,
			MockSKPrefix:    mockSKPrefix,
			DefaultRegion:   cfg.AWSDefaultRegion,
			RequestTimeoutS: cfg.AWSSDKTimeoutSeconds,
			PageLimit:       cfg.AWSSDKPageLimit,
		}),
		"aliyun": cloud.NewAliyunProvider(cloud.AliyunProviderOptions{
			MockEnabled:     mockEnabled,
			MockAKPrefix:    mockAKPrefix,
			MockSKPrefix:    mockSKPrefix,
			DefaultRegion:   cfg.AliyunDefaultRegion,
			RequestTimeoutS: cfg.AliyunSDKTimeoutSeconds,
			PageLimit:       cfg.AliyunSDKPageLimit,
		}),
		"tencent": cloud.NewTencentProvider(cloud.TencentProviderOptions{
			MockEnabled:     mockEnabled,
			MockAKPrefix:    mockAKPrefix,
			MockSKPrefix:    mockSKPrefix,
			DefaultRegion:   cfg.TencentDefaultRegion,
			RequestTimeoutS: cfg.TencentSDKTimeoutSeconds,
			PageLimit:       cfg.TencentSDKPageLimit,
		}),
		"huawei": cloud.NewHuaweiProvider(cloud.HuaweiProviderOptions{
			MockEnabled:     mockEnabled,
			MockAKPrefix:    mockAKPrefix,
			MockSKPrefix:    mockSKPrefix,
			DefaultRegion:   cfg.HuaweiDefaultRegion,
			RequestTimeoutS: cfg.HuaweiSDKTimeoutSeconds,
			PageLimit:       cfg.HuaweiSDKPageLimit,
		}),
	}
}
