package cloud

type ResourceCollector interface {
	CollectBaseResources(provider Provider, cred Credentials) ([]Asset, error)
}

type DefaultResourceCollector struct{}

func NewDefaultResourceCollector() ResourceCollector {
	return DefaultResourceCollector{}
}

func (DefaultResourceCollector) CollectBaseResources(provider Provider, cred Credentials) ([]Asset, error) {
	assets, err := provider.SyncAssets(cred)
	if err != nil {
		return nil, err
	}
	result := make([]Asset, 0, len(assets))
	for _, asset := range assets {
		if !IsSupportedBaseResourceType(asset.Type) {
			continue
		}
		asset.Type = NormalizeBaseResourceType(asset.Type)
		result = append(result, asset)
	}
	return result, nil
}
