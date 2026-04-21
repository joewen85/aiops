package cloud

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	cdb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cdb/v20170320"
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	cls "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cls/v20201016"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	sdkErr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	sts "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sts/v20180813"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

type TencentProviderOptions struct {
	MockEnabled     bool
	MockAKPrefix    string
	MockSKPrefix    string
	DefaultRegion   string
	RequestTimeoutS int
	PageLimit       int
}

type TencentProvider struct {
	mockEnabled     bool
	mockAKPrefix    string
	mockSKPrefix    string
	defaultRegion   string
	requestTimeoutS int
	pageLimit       int
	stub            Provider
}

func NewTencentProvider(opts TencentProviderOptions) Provider {
	timeout := opts.RequestTimeoutS
	if timeout <= 0 {
		timeout = 10
	}
	pageLimit := opts.PageLimit
	if pageLimit <= 0 {
		pageLimit = 100
	}
	if pageLimit > 100 {
		pageLimit = 100
	}
	return TencentProvider{
		mockEnabled:     opts.MockEnabled,
		mockAKPrefix:    strings.TrimSpace(strings.ToLower(opts.MockAKPrefix)),
		mockSKPrefix:    strings.TrimSpace(strings.ToLower(opts.MockSKPrefix)),
		defaultRegion:   defaultString(strings.TrimSpace(opts.DefaultRegion), "ap-guangzhou"),
		requestTimeoutS: timeout,
		pageLimit:       pageLimit,
		stub:            NewStubProvider("tencent"),
	}
}

func (p TencentProvider) Name() string {
	return "tencent"
}

func (p TencentProvider) Verify(cred Credentials) error {
	if p.shouldMock(cred) {
		return p.stub.Verify(cred)
	}
	credential, region, err := p.buildCredential(cred)
	if err != nil {
		return err
	}
	client, err := sts.NewClient(credential, region, p.clientProfile())
	if err != nil {
		return fmt.Errorf("init tencent sts client failed: %w", err)
	}
	_, err = client.GetCallerIdentity(sts.NewGetCallerIdentityRequest())
	if err != nil {
		return p.wrapSDKError("sts.GetCallerIdentity", err)
	}
	return nil
}

func (p TencentProvider) SyncAssets(cred Credentials) ([]Asset, error) {
	if p.shouldMock(cred) {
		return p.stub.SyncAssets(cred)
	}
	credential, region, err := p.buildCredential(cred)
	if err != nil {
		return nil, err
	}
	regions, err := p.resolveSyncRegions(credential, cred.Region, region)
	if err != nil {
		return nil, err
	}
	if len(regions) == 0 {
		regions = []string{region}
	}
	assets := make([]Asset, 0, 256)
	seen := make(map[string]struct{}, 256)
	cvmErrors := make([]string, 0)
	cvmSuccess := 0

	for _, syncRegion := range regions {
		if cvmAssets, cvmErr := p.collectCVMAssets(credential, syncRegion); cvmErr == nil {
			assets = appendUniqueAssets(assets, cvmAssets, seen)
			cvmSuccess++
		} else {
			cvmErrors = append(cvmErrors, fmt.Sprintf("%s: %v", syncRegion, cvmErr))
		}
		if cdbAssets, cdbErr := p.collectCDBAssets(credential, syncRegion); cdbErr == nil {
			assets = appendUniqueAssets(assets, cdbAssets, seen)
		}
		if vpcAssets, vpcErr := p.collectVPCAssets(credential, syncRegion); vpcErr == nil {
			assets = appendUniqueAssets(assets, vpcAssets, seen)
		}
		if clbAssets, clbErr := p.collectCLBAssets(credential, syncRegion); clbErr == nil {
			assets = appendUniqueAssets(assets, clbAssets, seen)
		}
		if clsAssets, clsErr := p.collectCLSAssets(credential, syncRegion); clsErr == nil {
			assets = appendUniqueAssets(assets, clsAssets, seen)
		}
	}
	if cvmSuccess == 0 && len(assets) == 0 && len(cvmErrors) > 0 {
		return nil, fmt.Errorf("cvm sync failed for all regions: %s", joinLimited(cvmErrors, 3))
	}

	return assets, nil
}

func (p TencentProvider) resolveSyncRegions(credential *common.Credential, requestedRegion string, fallbackRegion string) ([]string, error) {
	raw := strings.TrimSpace(requestedRegion)
	if raw == "" || strings.EqualFold(raw, "global") || strings.EqualFold(raw, "all") || raw == "*" {
		regions, err := p.discoverTencentRegions(credential, fallbackRegion)
		if err != nil {
			return []string{fallbackRegion}, nil
		}
		if len(regions) == 0 {
			return []string{fallbackRegion}, nil
		}
		return regions, nil
	}

	items := strings.Split(raw, ",")
	result := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		region := strings.TrimSpace(item)
		if region == "" {
			continue
		}
		key := strings.ToLower(region)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, region)
	}
	if len(result) == 0 {
		return []string{fallbackRegion}, nil
	}
	return result, nil
}

func (p TencentProvider) discoverTencentRegions(credential *common.Credential, fallbackRegion string) ([]string, error) {
	client, err := cvm.NewClient(credential, fallbackRegion, p.clientProfile())
	if err != nil {
		return nil, fmt.Errorf("init tencent cvm client failed: %w", err)
	}
	resp, err := client.DescribeRegions(cvm.NewDescribeRegionsRequest())
	if err != nil {
		return nil, p.wrapSDKError("cvm.DescribeRegions", err)
	}
	if resp == nil || resp.Response == nil || len(resp.Response.RegionSet) == 0 {
		return nil, nil
	}

	result := make([]string, 0, len(resp.Response.RegionSet))
	seen := map[string]struct{}{}
	for _, item := range resp.Response.RegionSet {
		if item == nil {
			continue
		}
		region := strings.TrimSpace(safeString(item.Region))
		if region == "" {
			continue
		}
		key := strings.ToLower(region)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, region)
	}
	return result, nil
}

func appendUniqueAssets(dst []Asset, incoming []Asset, seen map[string]struct{}) []Asset {
	for _, item := range incoming {
		key := strings.TrimSpace(item.Provider) + "|" + strings.TrimSpace(item.Type) + "|" + strings.TrimSpace(item.ID) + "|" + strings.TrimSpace(item.Region)
		if key == "|||" {
			dst = append(dst, item)
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		dst = append(dst, item)
	}
	return dst
}

func joinLimited(values []string, limit int) string {
	if len(values) == 0 {
		return ""
	}
	if limit <= 0 || len(values) <= limit {
		return strings.Join(values, "; ")
	}
	return strings.Join(values[:limit], "; ") + fmt.Sprintf(" (and %d more)", len(values)-limit)
}

func (p TencentProvider) collectCVMAssets(credential *common.Credential, region string) ([]Asset, error) {
	client, err := cvm.NewClient(credential, region, p.clientProfile())
	if err != nil {
		return nil, fmt.Errorf("init tencent cvm client failed: %w", err)
	}
	assets := make([]Asset, 0, 128)
	offset := int64(0)
	for {
		req := cvm.NewDescribeInstancesRequest()
		req.Offset = common.Int64Ptr(offset)
		req.Limit = common.Int64Ptr(int64(p.pageLimit))
		resp, describeErr := client.DescribeInstances(req)
		if describeErr != nil {
			return nil, p.wrapSDKError("cvm.DescribeInstances", describeErr)
		}
		if resp == nil || resp.Response == nil || len(resp.Response.InstanceSet) == 0 {
			break
		}
		for _, instance := range resp.Response.InstanceSet {
			if instance == nil {
				continue
			}
			metadata := map[string]interface{}{
				"instanceType": safeString(instance.InstanceType),
				"cpu":          safeInt64(instance.CPU),
				"memory":       formatMemoryGB(safeInt64(instance.Memory)),
				"os":           safeString(instance.OsName),
				"status":       safeString(instance.InstanceState),
				"zone":         safePlacementZone(instance.Placement),
				"privateIp":    firstString(instance.PrivateIpAddresses),
				"publicIp":     firstString(instance.PublicIpAddresses),
				"vpcId":        safeStringFromVPC(instance.VirtualPrivateCloud),
			}
			expiresAt := normalizeTimeText(safeString(instance.ExpiredTime))
			if expiresAt != "" {
				metadata["expiresAt"] = expiresAt
			}
			assets = append(assets, Asset{
				Provider: "tencent",
				Type:     "compute",
				ID:       safeString(instance.InstanceId),
				Name:     safeString(instance.InstanceName),
				Region:   region,
				Metadata: metadataWithTags(metadata, tagsFromCVM(instance.Tags)),
			})
		}

		count := int64(len(resp.Response.InstanceSet))
		offset += count
		total := safeInt64(resp.Response.TotalCount)
		if count < int64(p.pageLimit) || (total > 0 && offset >= total) {
			break
		}
	}
	return assets, nil
}

func (p TencentProvider) collectCDBAssets(credential *common.Credential, region string) ([]Asset, error) {
	client, err := cdb.NewClient(credential, region, p.clientProfile())
	if err != nil {
		return nil, fmt.Errorf("init tencent cdb client failed: %w", err)
	}
	assets := make([]Asset, 0, 64)
	offset := uint64(0)
	for {
		req := cdb.NewDescribeDBInstancesRequest()
		req.Offset = common.Uint64Ptr(offset)
		req.Limit = common.Uint64Ptr(uint64(p.pageLimit))
		resp, describeErr := client.DescribeDBInstances(req)
		if describeErr != nil {
			return nil, p.wrapSDKError("cdb.DescribeDBInstances", describeErr)
		}
		if resp == nil || resp.Response == nil || len(resp.Response.Items) == 0 {
			break
		}
		for _, instance := range resp.Response.Items {
			if instance == nil {
				continue
			}
			metadata := map[string]interface{}{
				"status":        cdbStatusText(safeInt64(instance.Status)),
				"engine":        firstNonEmpty(safeString(instance.EngineType), "mysql"),
				"engineVersion": safeString(instance.EngineVersion),
				"cpu":           safeInt64(instance.Cpu),
				"memoryMB":      safeInt64(instance.Memory),
				"storageGB":     safeInt64(instance.Volume),
				"privateIp":     safeString(instance.Vip),
				"privatePort":   safeInt64(instance.Vport),
				"vpcId":         firstNonEmpty(safeString(instance.UniqVpcId), strconv.FormatInt(safeInt64(instance.VpcId), 10)),
				"subnetId":      firstNonEmpty(safeString(instance.UniqSubnetId), strconv.FormatInt(safeInt64(instance.SubnetId), 10)),
			}
			expiresAt := normalizeTimeText(safeString(instance.DeadlineTime))
			if expiresAt != "" {
				metadata["expiresAt"] = expiresAt
			}
			assets = append(assets, Asset{
				Provider: "tencent",
				Type:     "mysql",
				ID:       safeString(instance.InstanceId),
				Name:     safeString(instance.InstanceName),
				Region:   firstNonEmpty(safeString(instance.Region), region),
				Metadata: metadataWithTags(metadata, tagsFromCDB(instance.TagList)),
			})
		}

		count := uint64(len(resp.Response.Items))
		offset += count
		total := uint64(safeInt64(resp.Response.TotalCount))
		if count < uint64(p.pageLimit) || (total > 0 && offset >= total) {
			break
		}
	}
	return assets, nil
}

func (p TencentProvider) collectVPCAssets(credential *common.Credential, region string) ([]Asset, error) {
	client, err := vpc.NewClient(credential, region, p.clientProfile())
	if err != nil {
		return nil, fmt.Errorf("init tencent vpc client failed: %w", err)
	}
	assets := make([]Asset, 0, 64)
	offset := 0
	for {
		req := vpc.NewDescribeVpcsRequest()
		req.Offset = common.StringPtr(strconv.Itoa(offset))
		req.Limit = common.StringPtr(strconv.Itoa(p.pageLimit))
		resp, describeErr := client.DescribeVpcs(req)
		if describeErr != nil {
			return nil, p.wrapSDKError("vpc.DescribeVpcs", describeErr)
		}
		if resp == nil || resp.Response == nil || len(resp.Response.VpcSet) == 0 {
			break
		}
		for _, item := range resp.Response.VpcSet {
			if item == nil {
				continue
			}
			metadata := map[string]interface{}{
				"cidr":      safeString(item.CidrBlock),
				"isDefault": safeBool(item.IsDefault),
				"createdAt": normalizeTimeText(safeString(item.CreatedTime)),
			}
			assets = append(assets, Asset{
				Provider: "tencent",
				Type:     "vpc",
				ID:       safeString(item.VpcId),
				Name:     safeString(item.VpcName),
				Region:   region,
				Metadata: metadataWithTags(metadata, tagsFromVPC(item.TagSet)),
			})
		}
		count := len(resp.Response.VpcSet)
		offset += count
		total := int(safeUint64(resp.Response.TotalCount))
		if count < p.pageLimit || (total > 0 && offset >= total) {
			break
		}
	}
	return assets, nil
}

func (p TencentProvider) collectCLBAssets(credential *common.Credential, region string) ([]Asset, error) {
	client, err := clb.NewClient(credential, region, p.clientProfile())
	if err != nil {
		return nil, fmt.Errorf("init tencent clb client failed: %w", err)
	}
	assets := make([]Asset, 0, 64)
	offset := int64(0)
	for {
		req := clb.NewDescribeLoadBalancersRequest()
		req.Offset = common.Int64Ptr(offset)
		req.Limit = common.Int64Ptr(int64(p.pageLimit))
		resp, describeErr := client.DescribeLoadBalancers(req)
		if describeErr != nil {
			return nil, p.wrapSDKError("clb.DescribeLoadBalancers", describeErr)
		}
		if resp == nil || resp.Response == nil || len(resp.Response.LoadBalancerSet) == 0 {
			break
		}
		for _, lb := range resp.Response.LoadBalancerSet {
			if lb == nil {
				continue
			}
			metadata := map[string]interface{}{
				"status":      clbStatusText(safeUint64(lb.Status)),
				"type":        safeString(lb.LoadBalancerType),
				"vpcId":       safeString(lb.VpcId),
				"vip":         firstString(lb.LoadBalancerVips),
				"addressIPv6": safeString(lb.AddressIPv6),
				"chargeType":  safeString(lb.ChargeType),
			}
			expiresAt := normalizeTimeText(safeString(lb.ExpireTime))
			if expiresAt != "" {
				metadata["expiresAt"] = expiresAt
			}
			assets = append(assets, Asset{
				Provider: "tencent",
				Type:     "loadbalancer",
				ID:       safeString(lb.LoadBalancerId),
				Name:     safeString(lb.LoadBalancerName),
				Region:   region,
				Metadata: metadataWithTags(metadata, tagsFromCLB(lb.Tags)),
			})
		}
		count := int64(len(resp.Response.LoadBalancerSet))
		offset += count
		total := int64(safeUint64(resp.Response.TotalCount))
		if count < int64(p.pageLimit) || (total > 0 && offset >= total) {
			break
		}
	}
	return assets, nil
}

func (p TencentProvider) collectCLSAssets(credential *common.Credential, region string) ([]Asset, error) {
	client, err := cls.NewClient(credential, region, p.clientProfile())
	if err != nil {
		return nil, fmt.Errorf("init tencent cls client failed: %w", err)
	}
	assets := make([]Asset, 0, 64)
	offset := int64(0)
	for {
		req := cls.NewDescribeTopicsRequest()
		req.Offset = common.Int64Ptr(offset)
		req.Limit = common.Int64Ptr(int64(p.pageLimit))
		resp, describeErr := client.DescribeTopics(req)
		if describeErr != nil {
			return nil, p.wrapSDKError("cls.DescribeTopics", describeErr)
		}
		if resp == nil || resp.Response == nil || len(resp.Response.Topics) == 0 {
			break
		}
		for _, topic := range resp.Response.Topics {
			if topic == nil {
				continue
			}
			metadata := map[string]interface{}{
				"logsetId":    safeString(topic.LogsetId),
				"partition":   safeInt64(topic.PartitionCount),
				"storageType": safeString(topic.StorageType),
				"periodDays":  safeInt64(topic.Period),
				"status":      clsStatusText(safeBool(topic.Status)),
			}
			assets = append(assets, Asset{
				Provider: "tencent",
				Type:     "logservice",
				ID:       safeString(topic.TopicId),
				Name:     safeString(topic.TopicName),
				Region:   region,
				Metadata: metadataWithTags(metadata, tagsFromCLS(topic.Tags)),
			})
		}
		count := int64(len(resp.Response.Topics))
		offset += count
		total := safeInt64(resp.Response.TotalCount)
		if count < int64(p.pageLimit) || (total > 0 && offset >= total) {
			break
		}
	}
	return assets, nil
}

func (p TencentProvider) buildCredential(cred Credentials) (*common.Credential, string, error) {
	accessKey := strings.TrimSpace(cred.AccessKey)
	secretKey := strings.TrimSpace(cred.SecretKey)
	if accessKey == "" || secretKey == "" {
		return nil, "", fmt.Errorf("access key or secret key is empty")
	}
	if strings.Contains(accessKey, "*") || strings.Contains(secretKey, "*") {
		return nil, "", fmt.Errorf("credential looks masked, please input original key in cloud account settings")
	}
	if !strings.HasPrefix(strings.ToUpper(accessKey), "AKID") {
		return nil, "", fmt.Errorf("invalid tencent accessKey format: expected SecretId (normally starts with AKID)")
	}
	regionInput := strings.TrimSpace(cred.Region)
	region := p.defaultRegion
	if regionInput != "" && !strings.EqualFold(regionInput, "global") {
		region = regionInput
	}
	return common.NewCredential(accessKey, secretKey), region, nil
}

func (p TencentProvider) clientProfile() *profile.ClientProfile {
	clientProfile := profile.NewClientProfile()
	httpProfile := profile.NewHttpProfile()
	httpProfile.ReqTimeout = p.requestTimeoutS
	clientProfile.HttpProfile = httpProfile
	return clientProfile
}

func (p TencentProvider) shouldMock(cred Credentials) bool {
	if p.mockEnabled {
		return true
	}
	ak := strings.ToLower(strings.TrimSpace(cred.AccessKey))
	sk := strings.ToLower(strings.TrimSpace(cred.SecretKey))
	if p.mockAKPrefix != "" && strings.HasPrefix(ak, p.mockAKPrefix) {
		return true
	}
	if p.mockSKPrefix != "" && strings.HasPrefix(sk, p.mockSKPrefix) {
		return true
	}
	return false
}

func (p TencentProvider) wrapSDKError(scene string, err error) error {
	var tencentErr *sdkErr.TencentCloudSDKError
	if errors.As(err, &tencentErr) {
		code := tencentErr.GetCode()
		message := tencentErr.GetMessage()
		switch code {
		case "AuthFailure.SecretIdNotFound":
			message = message + "; please check cloud account accessKey is Tencent SecretId (AKID...)"
		case "AuthFailure.SignatureFailure":
			message = message + "; please verify accessKey/secretKey pairing is correct"
		case "AuthFailure.TokenFailure":
			message = message + "; security token may be missing or expired"
		case "InvalidParameterValue":
			if strings.Contains(message, "X-TC-Region") {
				message = message + "; please set account region like ap-guangzhou/ap-shanghai/ap-beijing"
			}
		}
		return fmt.Errorf("%s failed: code=%s message=%s requestId=%s", scene, code, message, tencentErr.GetRequestId())
	}
	return fmt.Errorf("%s failed: %w", scene, err)
}

func metadataWithTags(metadata map[string]interface{}, tags map[string]interface{}) map[string]interface{} {
	if metadata == nil {
		metadata = map[string]interface{}{}
	}
	if len(tags) > 0 {
		metadata["tags"] = tags
	}
	return metadata
}

func tagsFromCVM(tags []*cvm.Tag) map[string]interface{} {
	result := map[string]interface{}{}
	for _, tag := range tags {
		if tag == nil {
			continue
		}
		key := strings.TrimSpace(safeString(tag.Key))
		if key == "" {
			continue
		}
		result[key] = safeString(tag.Value)
	}
	return result
}

func tagsFromVPC(tags []*vpc.Tag) map[string]interface{} {
	result := map[string]interface{}{}
	for _, tag := range tags {
		if tag == nil {
			continue
		}
		key := strings.TrimSpace(safeString(tag.Key))
		if key == "" {
			continue
		}
		result[key] = safeString(tag.Value)
	}
	return result
}

func tagsFromCLB(tags []*clb.TagInfo) map[string]interface{} {
	result := map[string]interface{}{}
	for _, tag := range tags {
		if tag == nil {
			continue
		}
		key := strings.TrimSpace(safeString(tag.TagKey))
		if key == "" {
			continue
		}
		result[key] = safeString(tag.TagValue)
	}
	return result
}

func tagsFromCDB(tags []*cdb.TagInfoItem) map[string]interface{} {
	result := map[string]interface{}{}
	for _, tag := range tags {
		if tag == nil {
			continue
		}
		key := strings.TrimSpace(safeString(tag.TagKey))
		if key == "" {
			continue
		}
		result[key] = safeString(tag.TagValue)
	}
	return result
}

func tagsFromCLS(tags []*cls.Tag) map[string]interface{} {
	result := map[string]interface{}{}
	for _, tag := range tags {
		if tag == nil {
			continue
		}
		key := strings.TrimSpace(safeString(tag.Key))
		if key == "" {
			continue
		}
		result[key] = safeString(tag.Value)
	}
	return result
}

func safeString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func safeInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func safeUint64(value *uint64) uint64 {
	if value == nil {
		return 0
	}
	return *value
}

func safeBool(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func firstString(values []*string) string {
	for _, value := range values {
		text := safeString(value)
		if text != "" {
			return text
		}
	}
	return ""
}

func safePlacementZone(item *cvm.Placement) string {
	if item == nil {
		return ""
	}
	return safeString(item.Zone)
}

func safeStringFromVPC(item *cvm.VirtualPrivateCloud) string {
	if item == nil {
		return ""
	}
	return safeString(item.VpcId)
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func normalizeTimeText(value string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return ""
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, text)
		if err != nil {
			continue
		}
		return parsed.UTC().Format(time.RFC3339)
	}
	return text
}

func cdbStatusText(value int64) string {
	switch value {
	case 0:
		return "creating"
	case 1:
		return "running"
	case 4:
		return "isolating"
	case 5:
		return "isolated"
	default:
		return "unknown"
	}
}

func clbStatusText(value uint64) string {
	switch value {
	case 1:
		return "running"
	case 0:
		return "creating"
	default:
		return "unknown"
	}
}

func clsStatusText(enabled bool) string {
	if enabled {
		return "running"
	}
	return "stopped"
}
