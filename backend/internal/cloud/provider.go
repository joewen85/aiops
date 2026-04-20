package cloud

import (
	"fmt"
	"time"
)

type Asset struct {
	Provider string                 `json:"provider"`
	Type     string                 `json:"type"`
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Region   string                 `json:"region"`
	Metadata map[string]interface{} `json:"metadata"`
}

type Credentials struct {
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
	Region    string `json:"region"`
}

type Provider interface {
	Name() string
	Verify(Credentials) error
	SyncAssets(Credentials) ([]Asset, error)
}

type StubProvider struct {
	provider string
}

func NewStubProvider(provider string) Provider {
	return StubProvider{provider: provider}
}

func (p StubProvider) Name() string {
	return p.provider
}

func (p StubProvider) Verify(cred Credentials) error {
	if cred.AccessKey == "" || cred.SecretKey == "" {
		return fmt.Errorf("access key or secret key is empty")
	}
	return nil
}

func (p StubProvider) SyncAssets(cred Credentials) ([]Asset, error) {
	if err := p.Verify(cred); err != nil {
		return nil, err
	}
	return []Asset{
		{
			Provider: p.provider,
			Type:     "compute",
			ID:       "stub-001",
			Name:     fmt.Sprintf("%s-instance-1", p.provider),
			Region:   cred.Region,
			Metadata: map[string]interface{}{
				"source":       "stub",
				"cpu":          "2",
				"memory":       "4Gi",
				"disk":         "40Gi",
				"privateIp":    "10.0.0.8",
				"publicIp":     "203.0.113.8",
				"os":           "Ubuntu 22.04",
				"instanceType": "ecs.c6.large",
				"status":       "running",
				"tags":         map[string]interface{}{"env": "prod", "team": "platform"},
				"expiresAt":    time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339),
			},
		},
		{
			Provider: p.provider,
			Type:     "mysql",
			ID:       "stub-mysql-001",
			Name:     fmt.Sprintf("%s-mysql-1", p.provider),
			Region:   cred.Region,
			Metadata: map[string]interface{}{"status": "available", "engine": "mysql8"},
		},
		{
			Provider: p.provider,
			Type:     "vpc",
			ID:       "stub-vpc-001",
			Name:     fmt.Sprintf("%s-vpc-main", p.provider),
			Region:   cred.Region,
			Metadata: map[string]interface{}{"cidr": "10.0.0.0/16", "status": "active"},
		},
		{
			Provider: p.provider,
			Type:     "objectstorage",
			ID:       "stub-oss-001",
			Name:     fmt.Sprintf("%s-oss-main", p.provider),
			Region:   cred.Region,
			Metadata: map[string]interface{}{"status": "active"},
		},
		{
			Provider: p.provider,
			Type:     "filestorage",
			ID:       "stub-nas-001",
			Name:     fmt.Sprintf("%s-nas-main", p.provider),
			Region:   cred.Region,
			Metadata: map[string]interface{}{"status": "active"},
		},
		{
			Provider: p.provider,
			Type:     "containerservice",
			ID:       "stub-k8s-001",
			Name:     fmt.Sprintf("%s-k8s-main", p.provider),
			Region:   cred.Region,
			Metadata: map[string]interface{}{"status": "running"},
		},
		{
			Provider: p.provider,
			Type:     "loadbalancer",
			ID:       "stub-lb-001",
			Name:     fmt.Sprintf("%s-lb-main", p.provider),
			Region:   cred.Region,
			Metadata: map[string]interface{}{"status": "active"},
		},
		{
			Provider: p.provider,
			Type:     "dns",
			ID:       "stub-dns-001",
			Name:     "example.com",
			Region:   cred.Region,
			Metadata: map[string]interface{}{"status": "active"},
		},
		{
			Provider: p.provider,
			Type:     "sslcertificate",
			ID:       "stub-cert-001",
			Name:     "example-com-cert",
			Region:   cred.Region,
			Metadata: map[string]interface{}{
				"status":    "issued",
				"expiresAt": time.Now().Add(90 * 24 * time.Hour).Format(time.RFC3339),
			},
		},
		{
			Provider: p.provider,
			Type:     "logservice",
			ID:       "stub-log-001",
			Name:     fmt.Sprintf("%s-log-main", p.provider),
			Region:   cred.Region,
			Metadata: map[string]interface{}{"status": "active"},
		},
	}, nil
}
