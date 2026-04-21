package cloud

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	sdkErr "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	alidns "github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
	cas "github.com/aliyun/alibaba-cloud-sdk-go/services/cas"
	cs "github.com/aliyun/alibaba-cloud-sdk-go/services/cs"
	ecs "github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	nas "github.com/aliyun/alibaba-cloud-sdk-go/services/nas"
	rds "github.com/aliyun/alibaba-cloud-sdk-go/services/rds"
	slb "github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
	sts "github.com/aliyun/alibaba-cloud-sdk-go/services/sts"
	logsdk "github.com/aliyun/aliyun-log-go-sdk"
	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type AliyunProviderOptions struct {
	MockEnabled     bool
	MockAKPrefix    string
	MockSKPrefix    string
	DefaultRegion   string
	RequestTimeoutS int
	PageLimit       int
}

type AliyunProvider struct {
	mockEnabled     bool
	mockAKPrefix    string
	mockSKPrefix    string
	defaultRegion   string
	requestTimeoutS int
	pageLimit       int
	stub            Provider
}

func NewAliyunProvider(opts AliyunProviderOptions) Provider {
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
	return AliyunProvider{
		mockEnabled:     opts.MockEnabled,
		mockAKPrefix:    strings.TrimSpace(strings.ToLower(opts.MockAKPrefix)),
		mockSKPrefix:    strings.TrimSpace(strings.ToLower(opts.MockSKPrefix)),
		defaultRegion:   defaultString(strings.TrimSpace(opts.DefaultRegion), "cn-hangzhou"),
		requestTimeoutS: timeout,
		pageLimit:       pageLimit,
		stub:            NewStubProvider("aliyun"),
	}
}

func (p AliyunProvider) Name() string {
	return "aliyun"
}

func (p AliyunProvider) Verify(cred Credentials) error {
	if p.shouldMock(cred) {
		return p.stub.Verify(cred)
	}
	accessKey, secretKey, region, err := p.buildCredential(cred)
	if err != nil {
		return err
	}
	client, err := p.newSTSClient(region, accessKey, secretKey)
	if err != nil {
		return err
	}
	_, err = client.GetCallerIdentity(sts.CreateGetCallerIdentityRequest())
	if err != nil {
		return p.wrapSDKError("sts.GetCallerIdentity", err)
	}
	return nil
}

func (p AliyunProvider) SyncAssets(cred Credentials) ([]Asset, error) {
	if p.shouldMock(cred) {
		return p.stub.SyncAssets(cred)
	}
	accessKey, secretKey, region, err := p.buildCredential(cred)
	if err != nil {
		return nil, err
	}

	regions, err := p.resolveSyncRegions(accessKey, secretKey, cred.Region, region)
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
		if ecsAssets, ecsErr := p.collectECSAssets(accessKey, secretKey, syncRegion); ecsErr == nil {
			assets = appendUniqueAssets(assets, ecsAssets, seen)
			ecsSuccess++
		} else {
			ecsErrors = append(ecsErrors, fmt.Sprintf("%s: %v", syncRegion, ecsErr))
		}
		if rdsAssets, rdsErr := p.collectRDSAssets(accessKey, secretKey, syncRegion); rdsErr == nil {
			assets = appendUniqueAssets(assets, rdsAssets, seen)
		}
		if vpcAssets, vpcErr := p.collectVPCAssets(accessKey, secretKey, syncRegion); vpcErr == nil {
			assets = appendUniqueAssets(assets, vpcAssets, seen)
		}
		if slbAssets, slbErr := p.collectSLBAssets(accessKey, secretKey, syncRegion); slbErr == nil {
			assets = appendUniqueAssets(assets, slbAssets, seen)
		}
		if nasAssets, nasErr := p.collectNASAssets(accessKey, secretKey, syncRegion); nasErr == nil {
			assets = appendUniqueAssets(assets, nasAssets, seen)
		}
		if csAssets, csErr := p.collectContainerAssets(accessKey, secretKey, syncRegion); csErr == nil {
			assets = appendUniqueAssets(assets, csAssets, seen)
		}
		if logAssets, logErr := p.collectLogServiceAssets(accessKey, secretKey, syncRegion); logErr == nil {
			assets = appendUniqueAssets(assets, logAssets, seen)
		}
	}

	if ossAssets, ossErr := p.collectOSSAssets(accessKey, secretKey, region); ossErr == nil {
		assets = appendUniqueAssets(assets, ossAssets, seen)
	}
	if dnsAssets, dnsErr := p.collectDNSAssets(accessKey, secretKey, region); dnsErr == nil {
		assets = appendUniqueAssets(assets, dnsAssets, seen)
	}
	if certAssets, certErr := p.collectSSLAssets(accessKey, secretKey, region); certErr == nil {
		assets = appendUniqueAssets(assets, certAssets, seen)
	}

	if ecsSuccess == 0 && len(assets) == 0 && len(ecsErrors) > 0 {
		return nil, fmt.Errorf("ecs sync failed for all regions: %s", joinLimited(ecsErrors, 3))
	}
	return assets, nil
}

func (p AliyunProvider) collectECSAssets(accessKey string, secretKey string, region string) ([]Asset, error) {
	client, err := p.newECSClient(region, accessKey, secretKey)
	if err != nil {
		return nil, err
	}
	assets := make([]Asset, 0, 128)
	pageNumber := 1
	for {
		req := ecs.CreateDescribeInstancesRequest()
		req.PageNumber = requests.NewInteger(pageNumber)
		req.PageSize = requests.NewInteger(p.pageLimit)
		resp, describeErr := client.DescribeInstances(req)
		if describeErr != nil {
			return nil, p.wrapSDKError("ecs.DescribeInstances", describeErr)
		}
		items := resp.Instances.Instance
		if len(items) == 0 {
			break
		}
		for _, instance := range items {
			metadata := map[string]interface{}{
				"instanceType": instance.InstanceType,
				"cpu":          firstPositiveInt(instance.CPU, instance.Cpu),
				"memoryMB":     instance.Memory,
				"memory":       formatMemoryMBToGB(int64(instance.Memory)),
				"os":           firstNonEmpty(instance.OSName, instance.OsType, instance.OSType),
				"status":       instance.Status,
				"zone":         instance.ZoneId,
				"privateIp":    firstAliyunString(instance.InnerIpAddress.IpAddress),
				"publicIp":     firstAliyunString(instance.PublicIpAddress.IpAddress),
				"vpcId":        instance.VpcAttributes.VpcId,
			}
			if instance.EipAddress.IpAddress != "" && metadata["publicIp"] == "" {
				metadata["publicIp"] = instance.EipAddress.IpAddress
			}
			expiresAt := normalizeTimeText(instance.ExpiredTime)
			if expiresAt != "" {
				metadata["expiresAt"] = expiresAt
			}
			assets = append(assets, Asset{
				Provider: "aliyun",
				Type:     "compute",
				ID:       strings.TrimSpace(instance.InstanceId),
				Name:     strings.TrimSpace(firstNonEmpty(instance.InstanceName, instance.Hostname, instance.HostName)),
				Region:   firstNonEmpty(strings.TrimSpace(instance.RegionId), region),
				Metadata: metadataWithTags(metadata, tagsFromAliyunECS(instance.Tags.Tag)),
			})
		}

		if len(items) < p.pageLimit || (resp.TotalCount > 0 && pageNumber*p.pageLimit >= resp.TotalCount) {
			break
		}
		pageNumber++
	}
	return assets, nil
}

func (p AliyunProvider) collectRDSAssets(accessKey string, secretKey string, region string) ([]Asset, error) {
	client, err := p.newRDSClient(region, accessKey, secretKey)
	if err != nil {
		return nil, err
	}
	assets := make([]Asset, 0, 64)
	pageNumber := 1
	for {
		req := rds.CreateDescribeDBInstancesRequest()
		req.PageNumber = requests.NewInteger(pageNumber)
		req.PageSize = requests.NewInteger(p.pageLimit)
		req.Engine = "MySQL"
		resp, describeErr := client.DescribeDBInstances(req)
		if describeErr != nil {
			return nil, p.wrapSDKError("rds.DescribeDBInstances", describeErr)
		}
		items := resp.Items.DBInstance
		if len(items) == 0 {
			break
		}
		for _, instance := range items {
			metadata := map[string]interface{}{
				"status":        firstNonEmpty(instance.DBInstanceStatus, instance.Status),
				"engine":        firstNonEmpty(instance.Engine, "MySQL"),
				"engineVersion": instance.EngineVersion,
				"class":         instance.DBInstanceClass,
				"cpu":           parseAliyunInt64(instance.DBInstanceCPU),
				"memoryMB":      instance.DBInstanceMemory,
				"connection":    instance.ConnectionString,
				"vpcId":         instance.VpcId,
				"vswitchId":     instance.VSwitchId,
			}
			expiresAt := normalizeTimeText(instance.ExpireTime)
			if expiresAt != "" {
				metadata["expiresAt"] = expiresAt
			}
			assets = append(assets, Asset{
				Provider: "aliyun",
				Type:     "mysql",
				ID:       strings.TrimSpace(instance.DBInstanceId),
				Name:     strings.TrimSpace(firstNonEmpty(instance.DBInstanceDescription, instance.DBInstanceName, instance.DBInstanceId)),
				Region:   firstNonEmpty(strings.TrimSpace(instance.RegionId), region),
				Metadata: metadata,
			})
		}

		if len(items) < p.pageLimit || (resp.TotalRecordCount > 0 && pageNumber*p.pageLimit >= resp.TotalRecordCount) {
			break
		}
		pageNumber++
	}
	return assets, nil
}

func (p AliyunProvider) collectVPCAssets(accessKey string, secretKey string, region string) ([]Asset, error) {
	client, err := p.newECSClient(region, accessKey, secretKey)
	if err != nil {
		return nil, err
	}
	assets := make([]Asset, 0, 64)
	pageNumber := 1
	for {
		req := ecs.CreateDescribeVpcsRequest()
		req.PageNumber = requests.NewInteger(pageNumber)
		req.PageSize = requests.NewInteger(p.pageLimit)
		resp, describeErr := client.DescribeVpcs(req)
		if describeErr != nil {
			return nil, p.wrapSDKError("ecs.DescribeVpcs", describeErr)
		}
		items := resp.Vpcs.Vpc
		if len(items) == 0 {
			break
		}
		for _, item := range items {
			metadata := map[string]interface{}{
				"cidr":      item.CidrBlock,
				"isDefault": item.IsDefault,
				"status":    item.Status,
				"createdAt": normalizeTimeText(item.CreationTime),
				"vrouterId": item.VRouterId,
			}
			assets = append(assets, Asset{
				Provider: "aliyun",
				Type:     "vpc",
				ID:       strings.TrimSpace(item.VpcId),
				Name:     strings.TrimSpace(firstNonEmpty(item.VpcName, item.VpcId)),
				Region:   firstNonEmpty(strings.TrimSpace(item.RegionId), region),
				Metadata: metadata,
			})
		}

		if len(items) < p.pageLimit || (resp.TotalCount > 0 && pageNumber*p.pageLimit >= resp.TotalCount) {
			break
		}
		pageNumber++
	}
	return assets, nil
}

func (p AliyunProvider) collectSLBAssets(accessKey string, secretKey string, region string) ([]Asset, error) {
	client, err := p.newSLBClient(region, accessKey, secretKey)
	if err != nil {
		return nil, err
	}
	assets := make([]Asset, 0, 64)
	pageNumber := 1
	for {
		req := slb.CreateDescribeLoadBalancersRequest()
		req.PageNumber = requests.NewInteger(pageNumber)
		req.PageSize = requests.NewInteger(p.pageLimit)
		resp, describeErr := client.DescribeLoadBalancers(req)
		if describeErr != nil {
			return nil, p.wrapSDKError("slb.DescribeLoadBalancers", describeErr)
		}
		items := resp.LoadBalancers.LoadBalancer
		if len(items) == 0 {
			break
		}
		for _, item := range items {
			metadata := map[string]interface{}{
				"status":      item.LoadBalancerStatus,
				"address":     item.Address,
				"addressType": item.AddressType,
				"networkType": item.NetworkType,
				"vpcId":       item.VpcId,
				"payType":     firstNonEmpty(item.PayType, item.InstanceChargeType),
			}
			assets = append(assets, Asset{
				Provider: "aliyun",
				Type:     "loadbalancer",
				ID:       strings.TrimSpace(item.LoadBalancerId),
				Name:     strings.TrimSpace(firstNonEmpty(item.LoadBalancerName, item.LoadBalancerId)),
				Region:   firstNonEmpty(strings.TrimSpace(item.RegionId), region),
				Metadata: metadataWithTags(metadata, tagsFromAliyunSLB(item.Tags.Tag)),
			})
		}

		if len(items) < p.pageLimit || (resp.TotalCount > 0 && pageNumber*p.pageLimit >= resp.TotalCount) {
			break
		}
		pageNumber++
	}
	return assets, nil
}

func (p AliyunProvider) collectOSSAssets(accessKey string, secretKey string, region string) ([]Asset, error) {
	endpoint := fmt.Sprintf("https://oss-%s.aliyuncs.com", strings.TrimSpace(region))
	client, err := oss.New(endpoint, accessKey, secretKey)
	if err != nil {
		return nil, fmt.Errorf("init aliyun oss client failed: %w", err)
	}
	assets := make([]Asset, 0, 64)
	marker := ""
	for {
		listResult, listErr := client.ListBuckets(oss.MaxKeys(p.pageLimit), oss.Marker(marker))
		if listErr != nil {
			return nil, fmt.Errorf("oss.ListBuckets failed: %w", listErr)
		}
		for _, bucket := range listResult.Buckets {
			bucketRegion := firstNonEmpty(strings.TrimSpace(bucket.Region), strings.TrimSpace(bucket.Location), region)
			metadata := map[string]interface{}{
				"location":     bucket.Location,
				"storageClass": bucket.StorageClass,
				"createdAt":    bucket.CreationDate.UTC().Format(time.RFC3339),
			}
			assets = append(assets, Asset{
				Provider: "aliyun",
				Type:     "objectstorage",
				ID:       strings.TrimSpace(bucket.Name),
				Name:     strings.TrimSpace(bucket.Name),
				Region:   bucketRegion,
				Metadata: metadata,
			})
		}
		if !listResult.IsTruncated || strings.TrimSpace(listResult.NextMarker) == "" {
			break
		}
		marker = listResult.NextMarker
	}
	return assets, nil
}

func (p AliyunProvider) collectNASAssets(accessKey string, secretKey string, region string) ([]Asset, error) {
	client, err := p.newNASClient(region, accessKey, secretKey)
	if err != nil {
		return nil, err
	}
	assets := make([]Asset, 0, 64)
	pageNumber := 1
	for {
		req := nas.CreateDescribeFileSystemsRequest()
		req.PageNumber = requests.NewInteger(pageNumber)
		req.PageSize = requests.NewInteger(p.pageLimit)
		resp, describeErr := client.DescribeFileSystems(req)
		if describeErr != nil {
			return nil, p.wrapSDKError("nas.DescribeFileSystems", describeErr)
		}
		items := resp.FileSystems.FileSystem
		if len(items) == 0 {
			break
		}
		for _, item := range items {
			metadata := map[string]interface{}{
				"status":       item.Status,
				"storageType":  item.StorageType,
				"protocolType": item.ProtocolType,
				"capacityGiB":  item.Capacity,
				"vpcId":        item.VpcId,
				"zone":         item.ZoneId,
				"createdAt":    normalizeTimeText(item.CreateTime),
			}
			expiresAt := normalizeTimeText(item.ExpiredTime)
			if expiresAt != "" {
				metadata["expiresAt"] = expiresAt
			}
			assets = append(assets, Asset{
				Provider: "aliyun",
				Type:     "filestorage",
				ID:       strings.TrimSpace(item.FileSystemId),
				Name:     strings.TrimSpace(firstNonEmpty(item.Description, item.FileSystemId)),
				Region:   firstNonEmpty(strings.TrimSpace(item.RegionId), region),
				Metadata: metadataWithTags(metadata, tagsFromAliyunNAS(item.Tags.Tag)),
			})
		}

		if len(items) < p.pageLimit || (resp.TotalCount > 0 && pageNumber*p.pageLimit >= resp.TotalCount) {
			break
		}
		pageNumber++
	}
	return assets, nil
}

func (p AliyunProvider) collectContainerAssets(accessKey string, secretKey string, region string) ([]Asset, error) {
	client, err := p.newCSClient(region, accessKey, secretKey)
	if err != nil {
		return nil, err
	}
	assets := make([]Asset, 0, 32)
	pageNumber := 1
	for {
		req := cs.CreateDescribeClustersV1Request()
		req.PageNumber = requests.NewInteger(pageNumber)
		req.PageSize = requests.NewInteger(p.pageLimit)
		resp, describeErr := client.DescribeClustersV1(req)
		if describeErr != nil {
			return nil, p.wrapSDKError("cs.DescribeClustersV1", describeErr)
		}
		clusters := parseAliyunCSClusters(resp.GetHttpContentBytes())
		if len(clusters) == 0 {
			break
		}
		for _, item := range clusters {
			clusterID := strings.TrimSpace(firstNonEmpty(item["cluster_id"], item["clusterId"], item["id"]))
			if clusterID == "" {
				continue
			}
			clusterRegion := strings.TrimSpace(firstNonEmpty(item["region_id"], item["region"], region))
			if clusterRegion == "" {
				clusterRegion = region
			}
			metadata := map[string]interface{}{
				"status":        firstNonEmpty(item["state"], item["status"]),
				"type":          firstNonEmpty(item["cluster_type"], item["clusterType"]),
				"vpcId":         firstNonEmpty(item["vpc_id"], item["vpcId"]),
				"kubernetesVer": firstNonEmpty(item["current_version"], item["cluster_spec"]),
			}
			assets = append(assets, Asset{
				Provider: "aliyun",
				Type:     "containerservice",
				ID:       clusterID,
				Name:     strings.TrimSpace(firstNonEmpty(item["name"], item["cluster_name"], clusterID)),
				Region:   clusterRegion,
				Metadata: metadata,
			})
		}
		if len(clusters) < p.pageLimit {
			break
		}
		pageNumber++
	}
	return assets, nil
}

func (p AliyunProvider) collectDNSAssets(accessKey string, secretKey string, region string) ([]Asset, error) {
	client, err := p.newDNSClient(region, accessKey, secretKey)
	if err != nil {
		return nil, err
	}
	assets := make([]Asset, 0, 64)
	pageNumber := 1
	for {
		req := alidns.CreateDescribeDomainsRequest()
		req.PageNumber = requests.NewInteger(pageNumber)
		req.PageSize = requests.NewInteger(p.pageLimit)
		resp, describeErr := client.DescribeDomains(req)
		if describeErr != nil {
			return nil, p.wrapSDKError("alidns.DescribeDomains", describeErr)
		}
		items := resp.Domains.Domain
		if len(items) == 0 {
			break
		}
		for _, item := range items {
			metadata := map[string]interface{}{
				"recordCount": item.RecordCount,
				"groupName":   item.GroupName,
				"aliDomain":   item.AliDomain,
				"expired":     item.InstanceExpired,
			}
			expiresAt := normalizeTimeText(item.InstanceEndTime)
			if expiresAt != "" {
				metadata["expiresAt"] = expiresAt
			}
			assets = append(assets, Asset{
				Provider: "aliyun",
				Type:     "dns",
				ID:       strings.TrimSpace(item.DomainId),
				Name:     strings.TrimSpace(firstNonEmpty(item.DomainName, item.PunyCode, item.DomainId)),
				Region:   region,
				Metadata: metadataWithTags(metadata, tagsFromAliyunDNS(item.Tags.Tag)),
			})
		}
		if len(items) < p.pageLimit || (resp.TotalCount > 0 && int64(pageNumber*p.pageLimit) >= resp.TotalCount) {
			break
		}
		pageNumber++
	}
	return assets, nil
}

func (p AliyunProvider) collectLogServiceAssets(accessKey string, secretKey string, region string) ([]Asset, error) {
	client := logsdk.CreateNormalInterface(aliyunLogServiceEndpoint(region), accessKey, secretKey, "")
	if p.requestTimeoutS > 0 {
		timeout := time.Duration(p.requestTimeoutS) * time.Second
		client.SetHTTPClient(&http.Client{Timeout: timeout})
		client.SetRetryTimeout(timeout)
	}
	defer client.Close()

	assets := make([]Asset, 0, 64)
	offset := 0
	for {
		projects, count, total, listErr := client.ListProjectV2(offset, p.pageLimit)
		if listErr != nil {
			return nil, p.wrapSDKError("sls.ListProjectV2", listErr)
		}
		if len(projects) == 0 {
			break
		}
		for _, project := range projects {
			projectName := strings.TrimSpace(project.Name)
			if projectName == "" {
				continue
			}
			projectRegion := firstNonEmpty(strings.TrimSpace(project.Region), region)
			if strings.TrimSpace(projectRegion) == "" {
				projectRegion = region
			}
			metadata := map[string]interface{}{
				"status":             strings.TrimSpace(project.Status),
				"description":        strings.TrimSpace(project.Description),
				"owner":              strings.TrimSpace(project.Owner),
				"location":           strings.TrimSpace(project.Location),
				"dataRedundancyType": strings.TrimSpace(project.DataRedundancyType),
			}
			if createdAt := normalizeAliyunUnixText(project.CreateTime); createdAt != "" {
				metadata["createdAt"] = createdAt
			}
			if modifiedAt := normalizeAliyunUnixText(project.LastModifyTime); modifiedAt != "" {
				metadata["lastModifyAt"] = modifiedAt
			}
			if logstoreCount, countErr := p.countAliyunLogstores(client, projectName); countErr == nil {
				metadata["logstoreCount"] = logstoreCount
			} else {
				metadata["logstoreCountError"] = countErr.Error()
			}
			assets = append(assets, Asset{
				Provider: "aliyun",
				Type:     "logservice",
				ID:       projectName,
				Name:     projectName,
				Region:   projectRegion,
				Metadata: metadata,
			})
		}
		if count <= 0 {
			count = len(projects)
		}
		offset += count
		if count < p.pageLimit || (total > 0 && offset >= total) {
			break
		}
	}
	return assets, nil
}

func (p AliyunProvider) countAliyunLogstores(client logsdk.ClientInterface, project string) (int, error) {
	offset := 0
	count := 0
	for {
		logstores, listErr := client.ListLogStoreV2(project, offset, p.pageLimit, "None")
		if listErr != nil {
			return 0, p.wrapSDKError("sls.ListLogStoreV2", listErr)
		}
		if len(logstores) == 0 {
			break
		}
		count += len(logstores)
		if len(logstores) < p.pageLimit {
			break
		}
		offset += len(logstores)
	}
	return count, nil
}

func (p AliyunProvider) collectSSLAssets(accessKey string, secretKey string, region string) ([]Asset, error) {
	client, err := p.newCASClient(region, accessKey, secretKey)
	if err != nil {
		return nil, err
	}
	assets := make([]Asset, 0, 64)
	pageNumber := 1
	for {
		req := cas.CreateListUserCertificateOrderRequest()
		req.CurrentPage = requests.NewInteger(pageNumber)
		req.ShowSize = requests.NewInteger(p.pageLimit)
		resp, listErr := client.ListUserCertificateOrder(req)
		if listErr != nil {
			return nil, p.wrapSDKError("cas.ListUserCertificateOrder", listErr)
		}
		items := resp.CertificateOrderList
		if len(items) == 0 {
			break
		}
		for _, item := range items {
			certificateID := strings.TrimSpace(firstNonEmpty(item.InstanceId, strconv.FormatInt(item.CertificateId, 10)))
			if certificateID == "" {
				certificateID = strconv.FormatInt(item.OrderId, 10)
			}
			metadata := map[string]interface{}{
				"status":      item.Status,
				"issuer":      item.Issuer,
				"commonName":  firstNonEmpty(item.CommonName, item.Domain),
				"domain":      item.Domain,
				"resourceId":  item.ResourceGroupId,
				"fingerprint": item.Fingerprint,
			}
			expiresAt := firstNonEmpty(normalizeTimeText(item.EndDate), normalizeTimeFromUnix(item.CertEndTime))
			if expiresAt != "" {
				metadata["expiresAt"] = expiresAt
			}
			assets = append(assets, Asset{
				Provider: "aliyun",
				Type:     "sslcertificate",
				ID:       certificateID,
				Name:     strings.TrimSpace(firstNonEmpty(item.Name, item.CommonName, item.Domain, certificateID)),
				Region:   region,
				Metadata: metadata,
			})
		}
		if len(items) < p.pageLimit || (resp.TotalCount > 0 && int64(pageNumber*p.pageLimit) >= resp.TotalCount) {
			break
		}
		pageNumber++
	}
	return assets, nil
}

func (p AliyunProvider) resolveSyncRegions(accessKey string, secretKey string, requestedRegion string, fallbackRegion string) ([]string, error) {
	raw := strings.TrimSpace(requestedRegion)
	if raw == "" || strings.EqualFold(raw, "global") || strings.EqualFold(raw, "all") || raw == "*" {
		regions, err := p.discoverAliyunRegions(accessKey, secretKey, fallbackRegion)
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

func (p AliyunProvider) discoverAliyunRegions(accessKey string, secretKey string, fallbackRegion string) ([]string, error) {
	client, err := p.newECSClient(fallbackRegion, accessKey, secretKey)
	if err != nil {
		return nil, err
	}
	resp, err := client.DescribeRegions(ecs.CreateDescribeRegionsRequest())
	if err != nil {
		return nil, p.wrapSDKError("ecs.DescribeRegions", err)
	}
	result := make([]string, 0, len(resp.Regions.Region))
	seen := map[string]struct{}{}
	for _, item := range resp.Regions.Region {
		region := strings.TrimSpace(item.RegionId)
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

func (p AliyunProvider) sdkConfig() *sdk.Config {
	timeout := p.requestTimeoutS
	if timeout <= 0 {
		timeout = 10
	}
	return sdk.NewConfig().
		WithScheme("HTTPS").
		WithTimeout(time.Duration(timeout) * time.Second)
}

func (p AliyunProvider) newSTSClient(region string, accessKey string, secretKey string) (*sts.Client, error) {
	client, err := sts.NewClientWithOptions(region, p.sdkConfig(), credentials.NewAccessKeyCredential(accessKey, secretKey))
	if err != nil {
		return nil, fmt.Errorf("init aliyun sts client failed: %w", err)
	}
	return client, nil
}

func (p AliyunProvider) newECSClient(region string, accessKey string, secretKey string) (*ecs.Client, error) {
	client, err := ecs.NewClientWithOptions(region, p.sdkConfig(), credentials.NewAccessKeyCredential(accessKey, secretKey))
	if err != nil {
		return nil, fmt.Errorf("init aliyun ecs client failed: %w", err)
	}
	return client, nil
}

func (p AliyunProvider) newRDSClient(region string, accessKey string, secretKey string) (*rds.Client, error) {
	client, err := rds.NewClientWithOptions(region, p.sdkConfig(), credentials.NewAccessKeyCredential(accessKey, secretKey))
	if err != nil {
		return nil, fmt.Errorf("init aliyun rds client failed: %w", err)
	}
	return client, nil
}

func (p AliyunProvider) newSLBClient(region string, accessKey string, secretKey string) (*slb.Client, error) {
	client, err := slb.NewClientWithOptions(region, p.sdkConfig(), credentials.NewAccessKeyCredential(accessKey, secretKey))
	if err != nil {
		return nil, fmt.Errorf("init aliyun slb client failed: %w", err)
	}
	return client, nil
}

func (p AliyunProvider) newNASClient(region string, accessKey string, secretKey string) (*nas.Client, error) {
	client, err := nas.NewClientWithOptions(region, p.sdkConfig(), credentials.NewAccessKeyCredential(accessKey, secretKey))
	if err != nil {
		return nil, fmt.Errorf("init aliyun nas client failed: %w", err)
	}
	return client, nil
}

func (p AliyunProvider) newCSClient(region string, accessKey string, secretKey string) (*cs.Client, error) {
	client, err := cs.NewClientWithOptions(region, p.sdkConfig(), credentials.NewAccessKeyCredential(accessKey, secretKey))
	if err != nil {
		return nil, fmt.Errorf("init aliyun cs client failed: %w", err)
	}
	return client, nil
}

func (p AliyunProvider) newDNSClient(region string, accessKey string, secretKey string) (*alidns.Client, error) {
	dnsRegion := firstNonEmpty(strings.TrimSpace(region), "cn-hangzhou")
	client, err := alidns.NewClientWithOptions(dnsRegion, p.sdkConfig(), credentials.NewAccessKeyCredential(accessKey, secretKey))
	if err != nil {
		return nil, fmt.Errorf("init aliyun alidns client failed: %w", err)
	}
	return client, nil
}

func (p AliyunProvider) newCASClient(region string, accessKey string, secretKey string) (*cas.Client, error) {
	casRegion := firstNonEmpty(strings.TrimSpace(region), "cn-hangzhou")
	client, err := cas.NewClientWithOptions(casRegion, p.sdkConfig(), credentials.NewAccessKeyCredential(accessKey, secretKey))
	if err != nil {
		return nil, fmt.Errorf("init aliyun cas client failed: %w", err)
	}
	return client, nil
}

func (p AliyunProvider) buildCredential(cred Credentials) (string, string, string, error) {
	accessKey := strings.TrimSpace(cred.AccessKey)
	secretKey := strings.TrimSpace(cred.SecretKey)
	if accessKey == "" || secretKey == "" {
		return "", "", "", fmt.Errorf("access key or secret key is empty")
	}
	if strings.Contains(accessKey, "*") || strings.Contains(secretKey, "*") {
		return "", "", "", fmt.Errorf("credential looks masked, please input original key in cloud account settings")
	}
	regionInput := strings.TrimSpace(cred.Region)
	region := p.defaultRegion
	if regionInput != "" && !strings.EqualFold(regionInput, "global") {
		region = regionInput
	}
	return accessKey, secretKey, region, nil
}

func (p AliyunProvider) shouldMock(cred Credentials) bool {
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

func (p AliyunProvider) wrapSDKError(scene string, err error) error {
	var logErr *logsdk.Error
	if errors.As(err, &logErr) {
		code := strings.TrimSpace(logErr.Code)
		message := strings.TrimSpace(logErr.Message)
		if code == "" {
			code = "ClientError"
		}
		if message == "" {
			message = strings.TrimSpace(logErr.Error())
		}
		switch code {
		case "SignatureNotMatch":
			message = message + "; please verify accessKey/secretKey pairing is correct"
		case "Unauthorized", "Project.Forbidden", "Project.NotExist":
			message = message + "; please verify aliyun log service permissions in this region"
		}
		return fmt.Errorf("%s failed: code=%s message=%s requestId=%s", scene, code, message, strings.TrimSpace(logErr.RequestID))
	}

	var sdkServerErr *sdkErr.ServerError
	if errors.As(err, &sdkServerErr) {
		code := strings.TrimSpace(sdkServerErr.ErrorCode())
		message := strings.TrimSpace(sdkServerErr.Message())
		if code == "" {
			code = "UnknownServerError"
		}
		if message == "" {
			message = strings.TrimSpace(sdkServerErr.Error())
		}
		switch code {
		case "InvalidRegionId.NotFound", "InvalidRegionId":
			message = message + "; please set valid region like cn-hangzhou/cn-shanghai"
		case "InvalidAccessKeyId.NotFound":
			message = message + "; please check aliyun accessKey is correct"
		case "SignatureDoesNotMatch":
			message = message + "; please verify accessKey/secretKey pairing is correct"
		}
		return fmt.Errorf("%s failed: code=%s message=%s requestId=%s", scene, code, message, strings.TrimSpace(sdkServerErr.RequestId()))
	}

	var sdkClientErr *sdkErr.ClientError
	if errors.As(err, &sdkClientErr) {
		return fmt.Errorf("%s failed: code=%s message=%s", scene, strings.TrimSpace(sdkClientErr.ErrorCode()), strings.TrimSpace(sdkClientErr.Message()))
	}
	return fmt.Errorf("%s failed: %w", scene, err)
}

func parseAliyunCSClusters(raw []byte) []map[string]string {
	payload := strings.TrimSpace(string(raw))
	if payload == "" {
		return nil
	}
	var listRaw []map[string]interface{}
	if err := json.Unmarshal(raw, &listRaw); err == nil {
		return normalizeAliyunCSClusterItems(listRaw)
	}
	var wrapped map[string]interface{}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil
	}
	for _, key := range []string{"clusters", "data", "items"} {
		value, ok := wrapped[key]
		if !ok {
			continue
		}
		items, ok := value.([]interface{})
		if !ok {
			continue
		}
		tmp := make([]map[string]interface{}, 0, len(items))
		for _, item := range items {
			if m, ok := item.(map[string]interface{}); ok {
				tmp = append(tmp, m)
			}
		}
		return normalizeAliyunCSClusterItems(tmp)
	}
	return nil
}

func normalizeAliyunCSClusterItems(items []map[string]interface{}) []map[string]string {
	result := make([]map[string]string, 0, len(items))
	for _, item := range items {
		normalized := map[string]string{}
		for key, value := range item {
			normalized[key] = fmt.Sprintf("%v", value)
		}
		result = append(result, normalized)
	}
	return result
}

func tagsFromAliyunECS(tags []ecs.Tag) map[string]interface{} {
	result := map[string]interface{}{}
	for _, tag := range tags {
		key := strings.TrimSpace(firstNonEmpty(tag.Key, tag.TagKey))
		if key == "" {
			continue
		}
		result[key] = strings.TrimSpace(firstNonEmpty(tag.Value, tag.TagValue))
	}
	return result
}

func tagsFromAliyunSLB(tags []slb.Tag) map[string]interface{} {
	result := map[string]interface{}{}
	for _, tag := range tags {
		key := strings.TrimSpace(tag.TagKey)
		if key == "" {
			continue
		}
		result[key] = strings.TrimSpace(tag.TagValue)
	}
	return result
}

func tagsFromAliyunNAS(tags []nas.Tag) map[string]interface{} {
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

func tagsFromAliyunDNS(tags []alidns.Tag) map[string]interface{} {
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

func parseAliyunInt64(raw string) int64 {
	text := strings.TrimSpace(raw)
	if text == "" {
		return 0
	}
	value, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func normalizeTimeFromUnix(raw int64) string {
	if raw <= 0 {
		return ""
	}
	value := raw
	if raw > 1_000_000_000_000 {
		value = raw / 1000
	}
	return time.Unix(value, 0).UTC().Format(time.RFC3339)
}

func normalizeAliyunUnixText(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return ""
	}
	value, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return normalizeTimeText(text)
	}
	return normalizeTimeFromUnix(value)
}

func aliyunLogServiceEndpoint(region string) string {
	regionText := strings.TrimSpace(region)
	if regionText == "" || strings.EqualFold(regionText, "global") {
		regionText = "cn-hangzhou"
	}
	return fmt.Sprintf("https://%s.log.aliyuncs.com", regionText)
}

func firstAliyunString(values []string) string {
	for _, value := range values {
		text := strings.TrimSpace(value)
		if text != "" {
			return text
		}
	}
	return ""
}

func firstPositiveInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
