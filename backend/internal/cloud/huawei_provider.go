package cloud

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	obs "github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	huaweiauth "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth"
	huaweihttp "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
	huaweisdkerr "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/sdkerr"
	huaweisdktime "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/sdktime"
	hwecs "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2"
	hwecsmodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2/model"
	hwecsregion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2/region"
	hwelb "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/elb/v3"
	hwelbmodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/elb/v3/model"
	hwelbregion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/elb/v3/region"
	hwrds "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/rds/v3"
	hwrdsmodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/rds/v3/model"
	hwrdsregion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/rds/v3/region"
	hwvpc "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v3"
	hwvpcmodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v3/model"
	hwvpcregion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v3/region"
)

type HuaweiProviderOptions struct {
	MockEnabled     bool
	MockAKPrefix    string
	MockSKPrefix    string
	DefaultRegion   string
	RequestTimeoutS int
	PageLimit       int
}

type HuaweiProvider struct {
	mockEnabled     bool
	mockAKPrefix    string
	mockSKPrefix    string
	defaultRegion   string
	requestTimeoutS int
	pageLimit       int
	stub            Provider
}

func NewHuaweiProvider(opts HuaweiProviderOptions) Provider {
	timeout := opts.RequestTimeoutS
	if timeout <= 0 {
		timeout = 10
	}
	pageLimit := opts.PageLimit
	if pageLimit <= 0 {
		pageLimit = 100
	}
	if pageLimit > 1000 {
		pageLimit = 1000
	}
	return HuaweiProvider{
		mockEnabled:     opts.MockEnabled,
		mockAKPrefix:    strings.TrimSpace(strings.ToLower(opts.MockAKPrefix)),
		mockSKPrefix:    strings.TrimSpace(strings.ToLower(opts.MockSKPrefix)),
		defaultRegion:   defaultString(strings.ToLower(strings.TrimSpace(opts.DefaultRegion)), "cn-north-4"),
		requestTimeoutS: timeout,
		pageLimit:       pageLimit,
		stub:            NewStubProvider("huawei"),
	}
}

func (p HuaweiProvider) Name() string {
	return "huawei"
}

func (p HuaweiProvider) Verify(cred Credentials) error {
	if p.shouldMock(cred) {
		return p.stub.Verify(cred)
	}
	credential, region, err := p.buildCredential(cred)
	if err != nil {
		return err
	}
	_, err = p.newECSClient(credential, region)
	if err != nil {
		return p.wrapSDKError("huawei.Verify", err)
	}
	return nil
}

func (p HuaweiProvider) SyncAssets(cred Credentials) ([]Asset, error) {
	if p.shouldMock(cred) {
		return p.stub.SyncAssets(cred)
	}
	credential, region, err := p.buildCredential(cred)
	if err != nil {
		return nil, err
	}
	regions, err := p.resolveSyncRegions(cred.Region, region)
	if err != nil {
		return nil, err
	}
	if len(regions) == 0 {
		regions = []string{region}
	}

	assets := make([]Asset, 0, 256)
	seen := make(map[string]struct{}, 256)
	ecsSuccess := 0
	ecsErrors := make([]string, 0)

	for _, syncRegion := range regions {
		if ecsAssets, ecsErr := p.collectECSAssets(credential, syncRegion); ecsErr == nil {
			assets = appendUniqueAssets(assets, ecsAssets, seen)
			ecsSuccess++
		} else {
			ecsErrors = append(ecsErrors, fmt.Sprintf("%s: %v", syncRegion, ecsErr))
		}
		if rdsAssets, rdsErr := p.collectRDSAssets(credential, syncRegion); rdsErr == nil {
			assets = appendUniqueAssets(assets, rdsAssets, seen)
		}
		if vpcAssets, vpcErr := p.collectVPCAssets(credential, syncRegion); vpcErr == nil {
			assets = appendUniqueAssets(assets, vpcAssets, seen)
		}
		if lbAssets, lbErr := p.collectELBAssets(credential, syncRegion); lbErr == nil {
			assets = appendUniqueAssets(assets, lbAssets, seen)
		}
	}
	if obsAssets, obsErr := p.collectOBSAssets(strings.TrimSpace(cred.AccessKey), strings.TrimSpace(cred.SecretKey), region); obsErr == nil {
		assets = appendUniqueAssets(assets, obsAssets, seen)
	}

	if ecsSuccess == 0 && len(assets) == 0 && len(ecsErrors) > 0 {
		return nil, fmt.Errorf("ecs sync failed for all regions: %s", joinLimited(ecsErrors, 3))
	}
	return assets, nil
}

func (p HuaweiProvider) collectECSAssets(credential *huaweiauth.BasicCredentials, region string) ([]Asset, error) {
	client, err := p.newECSClient(credential, region)
	if err != nil {
		return nil, err
	}
	assets := make([]Asset, 0, 128)
	limit := int32(p.pageLimit)
	offset := int32(1)
	for {
		resp, listErr := client.ListServersDetails(&hwecsmodel.ListServersDetailsRequest{
			Limit:  &limit,
			Offset: &offset,
		})
		if listErr != nil {
			return nil, p.wrapSDKError("ecs.ListServersDetails", listErr)
		}
		if resp == nil || resp.Servers == nil || len(*resp.Servers) == 0 {
			break
		}
		items := *resp.Servers
		for _, instance := range items {
			flavorID := ""
			flavorName := ""
			flavorCPU := ""
			flavorMemory := ""
			if instance.Flavor != nil {
				flavorID = strings.TrimSpace(instance.Flavor.Id)
				flavorName = strings.TrimSpace(instance.Flavor.Name)
				flavorCPU = strings.TrimSpace(instance.Flavor.Vcpus)
				flavorMemory = strings.TrimSpace(instance.Flavor.Ram)
			}
			cpu := huaweiParseInt64(flavorCPU)
			memoryMB := huaweiParseInt64(flavorMemory)
			metadata := map[string]interface{}{
				"instanceType":   strings.TrimSpace(firstNonEmpty(flavorName, flavorID)),
				"instanceTypeId": flavorID,
				"cpu":            cpu,
				"memoryMB":       memoryMB,
				"memory":         formatMemoryMBToGB(memoryMB),
				"os":             strings.TrimSpace(firstNonEmpty(instance.Metadata["os_type"], instance.Metadata["image_name"])),
				"status":         strings.TrimSpace(instance.Status),
				"zone":           strings.TrimSpace(instance.OSEXTAZavailabilityZone),
				"privateIp":      huaweiFirstAddressByType(instance.Addresses, true),
				"publicIp":       huaweiFirstAddressByType(instance.Addresses, false),
				"vpcId":          strings.TrimSpace(firstNonEmpty(instance.Metadata["vpc_id"], instance.Metadata["vpcid"])),
			}
			expiresAt := normalizeTimeText(strings.TrimSpace(instance.AutoTerminateTime))
			if expiresAt != "" {
				metadata["expiresAt"] = expiresAt
			}
			assets = append(assets, Asset{
				Provider: "huawei",
				Type:     "compute",
				ID:       strings.TrimSpace(instance.Id),
				Name:     strings.TrimSpace(firstNonEmpty(instance.Name, instance.OSEXTSRVATTRinstanceName, instance.Id)),
				Region:   region,
				Metadata: metadataWithTags(metadata, tagsFromHuaweiECSText(instance.Tags)),
			})
		}
		if len(items) < p.pageLimit {
			break
		}
		offset++
	}
	return assets, nil
}

func (p HuaweiProvider) collectRDSAssets(credential *huaweiauth.BasicCredentials, region string) ([]Asset, error) {
	client, err := p.newRDSClient(credential, region)
	if err != nil {
		return nil, err
	}
	assets := make([]Asset, 0, 64)
	limit := int32(p.pageLimit)
	offset := int32(0)
	mysqlType := hwrdsmodel.GetListInstancesRequestDatastoreTypeEnum().MY_SQL
	for {
		resp, listErr := client.ListInstances(&hwrdsmodel.ListInstancesRequest{
			Limit:         &limit,
			Offset:        &offset,
			DatastoreType: &mysqlType,
		})
		if listErr != nil {
			return nil, p.wrapSDKError("rds.ListInstances", listErr)
		}
		if resp == nil || resp.Instances == nil || len(*resp.Instances) == 0 {
			break
		}
		items := *resp.Instances
		for _, instance := range items {
			cpu := huaweiParseInt64(huaweiSafeString(instance.Cpu))
			memoryGB := huaweiParseFloat64(huaweiSafeString(instance.Mem))
			engine := "MySQL"
			engineVersion := ""
			if instance.Datastore != nil {
				engine = firstNonEmpty(instance.Datastore.Type.Value(), engine)
				engineVersion = strings.TrimSpace(instance.Datastore.Version)
			}
			metadata := map[string]interface{}{
				"status":        strings.TrimSpace(instance.Status),
				"engine":        engine,
				"engineVersion": engineVersion,
				"class":         strings.TrimSpace(instance.FlavorRef),
				"cpu":           cpu,
				"memoryGB":      memoryGB,
				"memory":        formatMemoryFloatGB(memoryGB),
				"privateIp":     huaweiFirstString(instance.PrivateIps),
				"publicIp":      huaweiFirstString(instance.PublicIps),
				"privatePort":   instance.Port,
				"vpcId":         strings.TrimSpace(instance.VpcId),
				"subnetId":      strings.TrimSpace(instance.SubnetId),
			}
			expiresAt := normalizeTimeText(huaweiSafeString(instance.ExpirationTime))
			if expiresAt != "" {
				metadata["expiresAt"] = expiresAt
			}
			assets = append(assets, Asset{
				Provider: "huawei",
				Type:     "mysql",
				ID:       strings.TrimSpace(instance.Id),
				Name:     strings.TrimSpace(firstNonEmpty(instance.Name, instance.Id)),
				Region:   firstNonEmpty(strings.TrimSpace(instance.Region), region),
				Metadata: metadataWithTags(metadata, tagsFromHuaweiRDSTags(instance.Tags)),
			})
		}
		if len(items) < p.pageLimit {
			break
		}
		offset += int32(len(items))
	}
	return assets, nil
}

func (p HuaweiProvider) collectVPCAssets(credential *huaweiauth.BasicCredentials, region string) ([]Asset, error) {
	client, err := p.newVPCClient(credential, region)
	if err != nil {
		return nil, err
	}
	assets := make([]Asset, 0, 64)
	limit := int32(p.pageLimit)
	marker := ""
	for {
		req := &hwvpcmodel.ListVpcsRequest{
			Limit: &limit,
		}
		if marker != "" {
			req.Marker = &marker
		}
		resp, listErr := client.ListVpcs(req)
		if listErr != nil {
			return nil, p.wrapSDKError("vpc.ListVpcs", listErr)
		}
		if resp == nil || resp.Vpcs == nil || len(*resp.Vpcs) == 0 {
			break
		}
		items := *resp.Vpcs
		for _, item := range items {
			metadata := map[string]interface{}{
				"cidr":                strings.TrimSpace(item.Cidr),
				"status":              strings.TrimSpace(item.Status),
				"description":         strings.TrimSpace(item.Description),
				"enterpriseProjectId": strings.TrimSpace(item.EnterpriseProjectId),
				"createdAt":           normalizeHuaweiSDKTime(item.CreatedAt),
				"updatedAt":           normalizeHuaweiSDKTime(item.UpdatedAt),
				"extendCidrs":         item.ExtendCidrs,
			}
			assets = append(assets, Asset{
				Provider: "huawei",
				Type:     "vpc",
				ID:       strings.TrimSpace(item.Id),
				Name:     strings.TrimSpace(firstNonEmpty(item.Name, item.Id)),
				Region:   region,
				Metadata: metadataWithTags(metadata, tagsFromHuaweiVPCTags(item.Tags)),
			})
		}
		next := ""
		if resp.PageInfo != nil && resp.PageInfo.NextMarker != nil {
			next = strings.TrimSpace(*resp.PageInfo.NextMarker)
		}
		if len(items) < p.pageLimit || next == "" {
			break
		}
		marker = next
	}
	return assets, nil
}

func (p HuaweiProvider) collectELBAssets(credential *huaweiauth.BasicCredentials, region string) ([]Asset, error) {
	client, err := p.newELBClient(credential, region)
	if err != nil {
		return nil, err
	}
	assets := make([]Asset, 0, 64)
	limit := int32(p.pageLimit)
	marker := ""
	for {
		req := &hwelbmodel.ListLoadBalancersRequest{
			Limit: &limit,
		}
		if marker != "" {
			req.Marker = &marker
		}
		resp, listErr := client.ListLoadBalancers(req)
		if listErr != nil {
			return nil, p.wrapSDKError("elb.ListLoadBalancers", listErr)
		}
		if resp == nil || resp.Loadbalancers == nil || len(*resp.Loadbalancers) == 0 {
			break
		}
		items := *resp.Loadbalancers
		for _, lb := range items {
			metadata := map[string]interface{}{
				"status":             strings.TrimSpace(firstNonEmpty(lb.OperatingStatus, lb.ProvisioningStatus)),
				"operatingStatus":    strings.TrimSpace(lb.OperatingStatus),
				"provisioningStatus": strings.TrimSpace(lb.ProvisioningStatus),
				"vpcId":              strings.TrimSpace(lb.VpcId),
				"vip":                strings.TrimSpace(lb.VipAddress),
				"publicIp":           huaweiELBPublicIP(lb),
				"guaranteed":         lb.Guaranteed,
				"createdAt":          normalizeTimeText(strings.TrimSpace(lb.CreatedAt)),
			}
			assets = append(assets, Asset{
				Provider: "huawei",
				Type:     "loadbalancer",
				ID:       strings.TrimSpace(lb.Id),
				Name:     strings.TrimSpace(firstNonEmpty(lb.Name, lb.Id)),
				Region:   region,
				Metadata: metadataWithTags(metadata, tagsFromHuaweiELBTags(lb.Tags)),
			})
		}
		next := ""
		if resp.PageInfo != nil && resp.PageInfo.NextMarker != nil {
			next = strings.TrimSpace(*resp.PageInfo.NextMarker)
		}
		if len(items) < p.pageLimit || next == "" {
			break
		}
		marker = next
	}
	return assets, nil
}

func (p HuaweiProvider) collectOBSAssets(accessKey string, secretKey string, region string) ([]Asset, error) {
	obsClient, err := obs.New(
		accessKey,
		secretKey,
		huaweiOBSEndpoint(region),
		obs.WithConnectTimeout(p.requestTimeoutS),
		obs.WithSocketTimeout(p.requestTimeoutS),
	)
	if err != nil {
		return nil, fmt.Errorf("init huawei obs client failed: %w", err)
	}
	defer obsClient.Close()

	output, err := obsClient.ListBuckets(&obs.ListBucketsInput{QueryLocation: true})
	if err != nil {
		return nil, p.wrapSDKError("obs.ListBuckets", err)
	}
	if output == nil || len(output.Buckets) == 0 {
		return []Asset{}, nil
	}

	assets := make([]Asset, 0, len(output.Buckets))
	for _, bucket := range output.Buckets {
		bucketName := strings.TrimSpace(bucket.Name)
		if bucketName == "" {
			continue
		}
		bucketRegion := strings.TrimSpace(firstNonEmpty(bucket.Location, region))
		metadata := map[string]interface{}{
			"status":     "active",
			"bucketType": strings.TrimSpace(bucket.BucketType),
			"createdAt":  bucket.CreationDate.UTC().Format(time.RFC3339),
		}
		assets = append(assets, Asset{
			Provider: "huawei",
			Type:     "objectstorage",
			ID:       bucketName,
			Name:     bucketName,
			Region:   bucketRegion,
			Metadata: metadata,
		})
	}
	return assets, nil
}

func (p HuaweiProvider) buildCredential(cred Credentials) (*huaweiauth.BasicCredentials, string, error) {
	accessKey := strings.TrimSpace(cred.AccessKey)
	secretKey := strings.TrimSpace(cred.SecretKey)
	if accessKey == "" || secretKey == "" {
		return nil, "", fmt.Errorf("access key or secret key is empty")
	}
	if strings.Contains(accessKey, "*") || strings.Contains(secretKey, "*") {
		return nil, "", fmt.Errorf("credential looks masked, please input original key in cloud account settings")
	}
	credential, err := huaweiauth.NewBasicCredentialsBuilder().
		WithAk(accessKey).
		WithSk(secretKey).
		SafeBuild()
	if err != nil {
		return nil, "", fmt.Errorf("invalid huawei credentials: %w", err)
	}

	regionInput := strings.ToLower(strings.TrimSpace(cred.Region))
	region := p.defaultRegion
	if regionInput != "" && !huaweiRegionIsGlobal(regionInput) {
		region = regionInput
	}
	return credential, region, nil
}

func (p HuaweiProvider) resolveSyncRegions(requestedRegion string, fallbackRegion string) ([]string, error) {
	raw := strings.TrimSpace(requestedRegion)
	if raw == "" || huaweiRegionIsGlobal(raw) {
		return []string{fallbackRegion}, nil
	}
	items := strings.Split(raw, ",")
	result := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		region := strings.ToLower(strings.TrimSpace(item))
		if region == "" {
			continue
		}
		if _, exists := seen[region]; exists {
			continue
		}
		seen[region] = struct{}{}
		result = append(result, region)
	}
	if len(result) == 0 {
		return []string{fallbackRegion}, nil
	}
	return result, nil
}

func (p HuaweiProvider) newECSClient(credential *huaweiauth.BasicCredentials, region string) (*hwecs.EcsClient, error) {
	regionObj, err := hwecsregion.SafeValueOf(region)
	if err != nil {
		return nil, fmt.Errorf("unsupported huawei ecs region %s: %w", region, err)
	}
	hcClient, err := hwecs.EcsClientBuilder().
		WithCredential(credential).
		WithRegion(regionObj).
		WithHttpConfig(p.httpConfig()).
		SafeBuild()
	if err != nil {
		return nil, fmt.Errorf("init huawei ecs client failed: %w", err)
	}
	return hwecs.NewEcsClient(hcClient), nil
}

func (p HuaweiProvider) newRDSClient(credential *huaweiauth.BasicCredentials, region string) (*hwrds.RdsClient, error) {
	regionObj, err := hwrdsregion.SafeValueOf(region)
	if err != nil {
		return nil, fmt.Errorf("unsupported huawei rds region %s: %w", region, err)
	}
	hcClient, err := hwrds.RdsClientBuilder().
		WithCredential(credential).
		WithRegion(regionObj).
		WithHttpConfig(p.httpConfig()).
		SafeBuild()
	if err != nil {
		return nil, fmt.Errorf("init huawei rds client failed: %w", err)
	}
	return hwrds.NewRdsClient(hcClient), nil
}

func (p HuaweiProvider) newVPCClient(credential *huaweiauth.BasicCredentials, region string) (*hwvpc.VpcClient, error) {
	regionObj, err := hwvpcregion.SafeValueOf(region)
	if err != nil {
		return nil, fmt.Errorf("unsupported huawei vpc region %s: %w", region, err)
	}
	hcClient, err := hwvpc.VpcClientBuilder().
		WithCredential(credential).
		WithRegion(regionObj).
		WithHttpConfig(p.httpConfig()).
		SafeBuild()
	if err != nil {
		return nil, fmt.Errorf("init huawei vpc client failed: %w", err)
	}
	return hwvpc.NewVpcClient(hcClient), nil
}

func (p HuaweiProvider) newELBClient(credential *huaweiauth.BasicCredentials, region string) (*hwelb.ElbClient, error) {
	regionObj, err := hwelbregion.SafeValueOf(region)
	if err != nil {
		return nil, fmt.Errorf("unsupported huawei elb region %s: %w", region, err)
	}
	hcClient, err := hwelb.ElbClientBuilder().
		WithCredential(credential).
		WithRegion(regionObj).
		WithHttpConfig(p.httpConfig()).
		SafeBuild()
	if err != nil {
		return nil, fmt.Errorf("init huawei elb client failed: %w", err)
	}
	return hwelb.NewElbClient(hcClient), nil
}

func (p HuaweiProvider) httpConfig() *huaweihttp.HttpConfig {
	return huaweihttp.DefaultHttpConfig().WithTimeout(time.Duration(p.requestTimeoutS) * time.Second)
}

func (p HuaweiProvider) shouldMock(cred Credentials) bool {
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

func (p HuaweiProvider) wrapSDKError(scene string, err error) error {
	var svcErr *huaweisdkerr.ServiceResponseError
	if errors.As(err, &svcErr) {
		errorCode := strings.TrimSpace(svcErr.ErrorCode)
		if errorCode == "" {
			errorCode = strconv.Itoa(svcErr.StatusCode)
		}
		message := strings.TrimSpace(svcErr.ErrorMessage)
		if message == "" {
			message = "service request failed"
		}
		if strings.Contains(strings.ToLower(message), "project id") {
			message += "; please check region and account permissions, or configure AK/SK for the correct project"
		}
		return fmt.Errorf("%s failed: code=%s message=%s requestId=%s", scene, errorCode, message, strings.TrimSpace(svcErr.RequestId))
	}

	var obsErr obs.ObsError
	if errors.As(err, &obsErr) {
		errorCode := strings.TrimSpace(obsErr.Code)
		if errorCode == "" {
			errorCode = strings.TrimSpace(obsErr.Status)
		}
		return fmt.Errorf("%s failed: code=%s message=%s requestId=%s", scene, errorCode, strings.TrimSpace(obsErr.Message), strings.TrimSpace(obsErr.RequestId))
	}

	return fmt.Errorf("%s failed: %w", scene, err)
}

func huaweiRegionIsGlobal(raw string) bool {
	region := strings.ToLower(strings.TrimSpace(raw))
	return region == "" || region == "global" || region == "all" || region == "*"
}

func huaweiOBSEndpoint(region string) string {
	normalized := strings.ToLower(strings.TrimSpace(region))
	if normalized == "" || huaweiRegionIsGlobal(normalized) {
		normalized = "cn-north-4"
	}
	if normalized == "eu-west-101" {
		return "https://obs.eu-west-101.myhuaweicloud.eu"
	}
	return "https://obs." + normalized + ".myhuaweicloud.com"
}

func huaweiSafeString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func huaweiParseInt64(raw string) int64 {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func huaweiParseFloat64(raw string) float64 {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0
	}
	return value
}

func huaweiFirstString(values []string) string {
	for _, item := range values {
		text := strings.TrimSpace(item)
		if text != "" {
			return text
		}
	}
	return ""
}

func huaweiFirstAddressByType(addresses map[string][]hwecsmodel.ServerAddress, private bool) string {
	if len(addresses) == 0 {
		return ""
	}
	for _, items := range addresses {
		for _, item := range items {
			ip := strings.TrimSpace(item.Addr)
			if ip == "" {
				continue
			}
			ipType := ""
			if item.OSEXTIPStype != nil {
				ipType = strings.ToLower(strings.TrimSpace(item.OSEXTIPStype.Value()))
			}
			if private {
				if ipType == "" || ipType == "fixed" {
					return ip
				}
				continue
			}
			if ipType == "floating" {
				return ip
			}
		}
	}
	return ""
}

func tagsFromHuaweiECSText(tags *[]string) map[string]interface{} {
	result := map[string]interface{}{}
	if tags == nil {
		return result
	}
	for _, item := range *tags {
		token := strings.TrimSpace(item)
		if token == "" {
			continue
		}
		if strings.Contains(token, "=") {
			parts := strings.SplitN(token, "=", 2)
			key := strings.TrimSpace(parts[0])
			if key == "" {
				continue
			}
			value := ""
			if len(parts) > 1 {
				value = strings.TrimSpace(parts[1])
			}
			result[key] = value
			continue
		}
		result[token] = ""
	}
	return result
}

func tagsFromHuaweiRDSTags(tags []hwrdsmodel.TagResponse) map[string]interface{} {
	result := map[string]interface{}{}
	for _, tag := range tags {
		key := strings.TrimSpace(tag.Key)
		if key == "" {
			continue
		}
		result[key] = strings.TrimSpace(tag.Value)
	}
	return result
}

func tagsFromHuaweiVPCTags(tags []hwvpcmodel.Tag) map[string]interface{} {
	result := map[string]interface{}{}
	for _, tag := range tags {
		key := strings.TrimSpace(tag.Key)
		if key == "" {
			continue
		}
		result[key] = strings.TrimSpace(tag.Value)
	}
	return result
}

func tagsFromHuaweiELBTags(tags []hwelbmodel.Tag) map[string]interface{} {
	result := map[string]interface{}{}
	for _, tag := range tags {
		key := huaweiSafeString(tag.Key)
		if key == "" {
			continue
		}
		result[key] = huaweiSafeString(tag.Value)
	}
	return result
}

func huaweiELBPublicIP(loadBalancer hwelbmodel.LoadBalancer) string {
	for _, item := range loadBalancer.Publicips {
		ip := strings.TrimSpace(item.PublicipAddress)
		if ip != "" {
			return ip
		}
	}
	for _, item := range loadBalancer.Eips {
		ip := huaweiSafeString(item.EipAddress)
		if ip != "" {
			return ip
		}
	}
	return ""
}

func normalizeHuaweiSDKTime(value *huaweisdktime.SdkTime) string {
	if value == nil {
		return ""
	}
	return normalizeTimeText(value.String())
}
