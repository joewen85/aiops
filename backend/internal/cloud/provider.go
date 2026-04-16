package cloud

import "fmt"

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
			Metadata: map[string]interface{}{"source": "stub"},
		},
	}, nil
}
