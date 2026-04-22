package cloud

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awscredentials "github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	awsec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	awselbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	awselbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	awsrds "github.com/aws/aws-sdk-go-v2/service/rds"
	awsrdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	awss3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	awssts "github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"
)

type AWSProviderOptions struct {
	MockEnabled     bool
	MockAKPrefix    string
	MockSKPrefix    string
	DefaultRegion   string
	RequestTimeoutS int
	PageLimit       int
}

type AWSProvider struct {
	mockEnabled     bool
	mockAKPrefix    string
	mockSKPrefix    string
	defaultRegion   string
	requestTimeoutS int
	pageLimit       int
	stub            Provider
}

func NewAWSProvider(opts AWSProviderOptions) Provider {
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
	return AWSProvider{
		mockEnabled:     opts.MockEnabled,
		mockAKPrefix:    strings.TrimSpace(strings.ToLower(opts.MockAKPrefix)),
		mockSKPrefix:    strings.TrimSpace(strings.ToLower(opts.MockSKPrefix)),
		defaultRegion:   defaultString(strings.ToLower(strings.TrimSpace(opts.DefaultRegion)), "us-east-1"),
		requestTimeoutS: timeout,
		pageLimit:       pageLimit,
		stub:            NewStubProvider("aws"),
	}
}

func (p AWSProvider) Name() string {
	return "aws"
}

func (p AWSProvider) Verify(cred Credentials) error {
	if p.shouldMock(cred) {
		return p.stub.Verify(cred)
	}
	cfg, region, err := p.buildConfig(cred)
	if err != nil {
		return err
	}
	cfg.Region = region
	client := awssts.NewFromConfig(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.requestTimeoutS)*time.Second)
	defer cancel()
	_, err = client.GetCallerIdentity(ctx, &awssts.GetCallerIdentityInput{})
	if err != nil {
		return p.wrapSDKError("sts.GetCallerIdentity", err)
	}
	return nil
}

func (p AWSProvider) SyncAssets(cred Credentials) ([]Asset, error) {
	if p.shouldMock(cred) {
		return p.stub.SyncAssets(cred)
	}
	cfg, region, err := p.buildConfig(cred)
	if err != nil {
		return nil, err
	}
	regions, err := p.resolveSyncRegions(cfg, cred.Region, region)
	if err != nil {
		return nil, err
	}
	if len(regions) == 0 {
		regions = []string{region}
	}

	assets := make([]Asset, 0, 256)
	seen := make(map[string]struct{}, 256)
	ec2Success := 0
	ec2Errors := make([]string, 0)

	for _, syncRegion := range regions {
		regionalCfg := cfg.Copy()
		regionalCfg.Region = syncRegion
		if ec2Assets, ec2Err := p.collectEC2Assets(regionalCfg, syncRegion); ec2Err == nil {
			assets = appendUniqueAssets(assets, ec2Assets, seen)
			ec2Success++
		} else {
			ec2Errors = append(ec2Errors, fmt.Sprintf("%s: %v", syncRegion, ec2Err))
		}
		if rdsAssets, rdsErr := p.collectRDSAssets(regionalCfg, syncRegion); rdsErr == nil {
			assets = appendUniqueAssets(assets, rdsAssets, seen)
		}
		if vpcAssets, vpcErr := p.collectVPCAssets(regionalCfg, syncRegion); vpcErr == nil {
			assets = appendUniqueAssets(assets, vpcAssets, seen)
		}
		if lbAssets, lbErr := p.collectELBAssets(regionalCfg, syncRegion); lbErr == nil {
			assets = appendUniqueAssets(assets, lbAssets, seen)
		}
	}
	if s3Assets, s3Err := p.collectS3Assets(cfg, region); s3Err == nil {
		assets = appendUniqueAssets(assets, s3Assets, seen)
	}

	if ec2Success == 0 && len(assets) == 0 && len(ec2Errors) > 0 {
		return nil, fmt.Errorf("ec2 sync failed for all regions: %s", joinLimited(ec2Errors, 3))
	}
	return assets, nil
}

func (p AWSProvider) collectEC2Assets(cfg aws.Config, region string) ([]Asset, error) {
	client := awsec2.NewFromConfig(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.requestTimeoutS)*time.Second)
	defer cancel()

	paginator := awsec2.NewDescribeInstancesPaginator(client, &awsec2.DescribeInstancesInput{
		MaxResults: aws.Int32(int32(p.pageLimit)),
	})

	allInstances := make([]awsec2types.Instance, 0, 128)
	instanceTypeSet := make(map[string]struct{}, 64)
	for paginator.HasMorePages() {
		resp, pageErr := paginator.NextPage(ctx)
		if pageErr != nil {
			return nil, p.wrapSDKError("ec2.DescribeInstances", pageErr)
		}
		for _, reservation := range resp.Reservations {
			for _, instance := range reservation.Instances {
				allInstances = append(allInstances, instance)
				instanceType := strings.TrimSpace(string(instance.InstanceType))
				if instanceType != "" {
					instanceTypeSet[instanceType] = struct{}{}
				}
			}
		}
	}
	if len(allInstances) == 0 {
		return []Asset{}, nil
	}

	typeInfoMap, _ := p.describeEC2InstanceTypes(ctx, client, instanceTypeSet)

	assets := make([]Asset, 0, len(allInstances))
	for _, instance := range allInstances {
		instanceID := strings.TrimSpace(aws.ToString(instance.InstanceId))
		if instanceID == "" {
			continue
		}
		instanceType := strings.TrimSpace(string(instance.InstanceType))
		cpu := int64(0)
		memoryMB := int64(0)
		if info, ok := typeInfoMap[instanceType]; ok {
			cpu = info.cpu
			memoryMB = info.memoryMB
		}
		metadata := map[string]interface{}{
			"instanceType": instanceType,
			"cpu":          cpu,
			"memoryMB":     memoryMB,
			"memory":       formatMemoryMBToGB(memoryMB),
			"os":           strings.TrimSpace(aws.ToString(instance.PlatformDetails)),
			"status":       strings.TrimSpace(string(instance.State.Name)),
			"zone":         strings.TrimSpace(aws.ToString(instance.Placement.AvailabilityZone)),
			"privateIp":    strings.TrimSpace(aws.ToString(instance.PrivateIpAddress)),
			"publicIp":     strings.TrimSpace(aws.ToString(instance.PublicIpAddress)),
			"vpcId":        strings.TrimSpace(aws.ToString(instance.VpcId)),
			"subnetId":     strings.TrimSpace(aws.ToString(instance.SubnetId)),
		}
		assets = append(assets, Asset{
			Provider: "aws",
			Type:     "compute",
			ID:       instanceID,
			Name:     strings.TrimSpace(firstNonEmpty(awsEC2InstanceName(instance), instanceID)),
			Region:   region,
			Metadata: metadataWithTags(metadata, tagsFromAWSEC2(instance.Tags)),
		})
	}
	return assets, nil
}

type awsInstanceTypeInfo struct {
	cpu      int64
	memoryMB int64
}

func (p AWSProvider) describeEC2InstanceTypes(ctx context.Context, client *awsec2.Client, instanceTypeSet map[string]struct{}) (map[string]awsInstanceTypeInfo, error) {
	if len(instanceTypeSet) == 0 {
		return map[string]awsInstanceTypeInfo{}, nil
	}
	instanceTypes := make([]awsec2types.InstanceType, 0, len(instanceTypeSet))
	for value := range instanceTypeSet {
		instanceTypes = append(instanceTypes, awsec2types.InstanceType(value))
	}

	result := make(map[string]awsInstanceTypeInfo, len(instanceTypes))
	for start := 0; start < len(instanceTypes); start += 100 {
		end := start + 100
		if end > len(instanceTypes) {
			end = len(instanceTypes)
		}
		resp, err := client.DescribeInstanceTypes(ctx, &awsec2.DescribeInstanceTypesInput{
			InstanceTypes: instanceTypes[start:end],
		})
		if err != nil {
			return result, p.wrapSDKError("ec2.DescribeInstanceTypes", err)
		}
		for _, item := range resp.InstanceTypes {
			instanceType := strings.TrimSpace(string(item.InstanceType))
			if instanceType == "" {
				continue
			}
			cpu := int64(aws.ToInt32(item.VCpuInfo.DefaultVCpus))
			memoryMB := int64(aws.ToInt64(item.MemoryInfo.SizeInMiB))
			result[instanceType] = awsInstanceTypeInfo{
				cpu:      cpu,
				memoryMB: memoryMB,
			}
		}
	}
	return result, nil
}

func (p AWSProvider) collectRDSAssets(cfg aws.Config, region string) ([]Asset, error) {
	client := awsrds.NewFromConfig(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.requestTimeoutS)*time.Second)
	defer cancel()

	assets := make([]Asset, 0, 64)
	marker := ""
	rdsPageLimit := int32(p.pageLimit)
	if rdsPageLimit <= 0 {
		rdsPageLimit = 100
	}
	if rdsPageLimit > 100 {
		rdsPageLimit = 100
	}
	for {
		input := &awsrds.DescribeDBInstancesInput{
			MaxRecords: aws.Int32(rdsPageLimit),
		}
		if marker != "" {
			input.Marker = aws.String(marker)
		}
		resp, err := client.DescribeDBInstances(ctx, input)
		if err != nil {
			return nil, p.wrapSDKError("rds.DescribeDBInstances", err)
		}
		if len(resp.DBInstances) == 0 {
			break
		}
		for _, db := range resp.DBInstances {
			engine := strings.TrimSpace(aws.ToString(db.Engine))
			if !strings.Contains(strings.ToLower(engine), "mysql") {
				continue
			}
			dbID := strings.TrimSpace(aws.ToString(db.DBInstanceIdentifier))
			if dbID == "" {
				continue
			}
			metadata := map[string]interface{}{
				"status":        strings.TrimSpace(aws.ToString(db.DBInstanceStatus)),
				"engine":        engine,
				"engineVersion": strings.TrimSpace(aws.ToString(db.EngineVersion)),
				"class":         strings.TrimSpace(aws.ToString(db.DBInstanceClass)),
				"storageGB":     db.AllocatedStorage,
				"connection":    awsRDSConnection(db),
				"privateIp":     strings.TrimSpace(aws.ToString(db.Endpoint.Address)),
				"privatePort":   aws.ToInt32(db.Endpoint.Port),
				"vpcId":         awsRDSVpcID(db),
			}
			assets = append(assets, Asset{
				Provider: "aws",
				Type:     "mysql",
				ID:       dbID,
				Name:     strings.TrimSpace(firstNonEmpty(aws.ToString(db.DBName), dbID)),
				Region:   region,
				Metadata: metadata,
			})
		}
		nextMarker := strings.TrimSpace(aws.ToString(resp.Marker))
		if nextMarker == "" {
			break
		}
		marker = nextMarker
	}
	return assets, nil
}

func (p AWSProvider) collectVPCAssets(cfg aws.Config, region string) ([]Asset, error) {
	client := awsec2.NewFromConfig(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.requestTimeoutS)*time.Second)
	defer cancel()

	paginator := awsec2.NewDescribeVpcsPaginator(client, &awsec2.DescribeVpcsInput{
		MaxResults: aws.Int32(int32(p.pageLimit)),
	})
	assets := make([]Asset, 0, 64)
	for paginator.HasMorePages() {
		resp, pageErr := paginator.NextPage(ctx)
		if pageErr != nil {
			return nil, p.wrapSDKError("ec2.DescribeVpcs", pageErr)
		}
		for _, item := range resp.Vpcs {
			vpcID := strings.TrimSpace(aws.ToString(item.VpcId))
			if vpcID == "" {
				continue
			}
			metadata := map[string]interface{}{
				"cidr":      strings.TrimSpace(aws.ToString(item.CidrBlock)),
				"isDefault": item.IsDefault,
				"status":    strings.TrimSpace(string(item.State)),
			}
			assets = append(assets, Asset{
				Provider: "aws",
				Type:     "vpc",
				ID:       vpcID,
				Name:     strings.TrimSpace(firstNonEmpty(awsVPCName(item), vpcID)),
				Region:   region,
				Metadata: metadataWithTags(metadata, tagsFromAWSEC2(item.Tags)),
			})
		}
	}
	return assets, nil
}

func (p AWSProvider) collectELBAssets(cfg aws.Config, region string) ([]Asset, error) {
	client := awselbv2.NewFromConfig(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.requestTimeoutS)*time.Second)
	defer cancel()

	assets := make([]Asset, 0, 64)
	allLoadBalancers := make([]awselbv2types.LoadBalancer, 0, 64)
	marker := ""
	elbPageLimit := int32(p.pageLimit)
	if elbPageLimit <= 0 {
		elbPageLimit = 100
	}
	if elbPageLimit > 400 {
		elbPageLimit = 400
	}
	for {
		input := &awselbv2.DescribeLoadBalancersInput{
			PageSize: aws.Int32(elbPageLimit),
		}
		if marker != "" {
			input.Marker = aws.String(marker)
		}
		resp, err := client.DescribeLoadBalancers(ctx, input)
		if err != nil {
			return nil, p.wrapSDKError("elbv2.DescribeLoadBalancers", err)
		}
		if len(resp.LoadBalancers) == 0 {
			break
		}
		allLoadBalancers = append(allLoadBalancers, resp.LoadBalancers...)
		nextMarker := strings.TrimSpace(aws.ToString(resp.NextMarker))
		if nextMarker == "" {
			break
		}
		marker = nextMarker
	}
	if len(allLoadBalancers) == 0 {
		return assets, nil
	}

	tagsByArn := p.describeELBTags(ctx, client, allLoadBalancers)

	for _, lb := range allLoadBalancers {
		lbArn := strings.TrimSpace(aws.ToString(lb.LoadBalancerArn))
		if lbArn == "" {
			continue
		}
		metadata := map[string]interface{}{
			"status":    strings.TrimSpace(string(lb.State.Code)),
			"type":      strings.TrimSpace(string(lb.Type)),
			"scheme":    strings.TrimSpace(string(lb.Scheme)),
			"vpcId":     strings.TrimSpace(aws.ToString(lb.VpcId)),
			"dnsName":   strings.TrimSpace(aws.ToString(lb.DNSName)),
			"createdAt": awsTimeText(lb.CreatedTime),
		}
		assets = append(assets, Asset{
			Provider: "aws",
			Type:     "loadbalancer",
			ID:       lbArn,
			Name:     strings.TrimSpace(firstNonEmpty(aws.ToString(lb.LoadBalancerName), lbArn)),
			Region:   region,
			Metadata: metadataWithTags(metadata, tagsByArn[lbArn]),
		})
	}
	return assets, nil
}

func (p AWSProvider) describeELBTags(ctx context.Context, client *awselbv2.Client, loadBalancers []awselbv2types.LoadBalancer) map[string]map[string]interface{} {
	result := map[string]map[string]interface{}{}
	arns := make([]string, 0, len(loadBalancers))
	for _, item := range loadBalancers {
		arn := strings.TrimSpace(aws.ToString(item.LoadBalancerArn))
		if arn != "" {
			arns = append(arns, arn)
		}
	}
	for start := 0; start < len(arns); start += 20 {
		end := start + 20
		if end > len(arns) {
			end = len(arns)
		}
		resp, err := client.DescribeTags(ctx, &awselbv2.DescribeTagsInput{
			ResourceArns: arns[start:end],
		})
		if err != nil {
			continue
		}
		for _, desc := range resp.TagDescriptions {
			resourceArn := strings.TrimSpace(aws.ToString(desc.ResourceArn))
			if resourceArn == "" {
				continue
			}
			tags := map[string]interface{}{}
			for _, tag := range desc.Tags {
				key := strings.TrimSpace(aws.ToString(tag.Key))
				if key == "" {
					continue
				}
				tags[key] = strings.TrimSpace(aws.ToString(tag.Value))
			}
			result[resourceArn] = tags
		}
	}
	return result
}

func (p AWSProvider) collectS3Assets(cfg aws.Config, fallbackRegion string) ([]Asset, error) {
	client := awss3.NewFromConfig(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.requestTimeoutS)*time.Second)
	defer cancel()

	resp, err := client.ListBuckets(ctx, &awss3.ListBucketsInput{})
	if err != nil {
		return nil, p.wrapSDKError("s3.ListBuckets", err)
	}
	if len(resp.Buckets) == 0 {
		return []Asset{}, nil
	}

	assets := make([]Asset, 0, len(resp.Buckets))
	for _, bucket := range resp.Buckets {
		bucketName := strings.TrimSpace(aws.ToString(bucket.Name))
		if bucketName == "" {
			continue
		}
		locationResp, locationErr := client.GetBucketLocation(ctx, &awss3.GetBucketLocationInput{
			Bucket: aws.String(bucketName),
		})
		bucketRegion := fallbackRegion
		if locationErr == nil {
			bucketRegion = awsS3BucketRegion(locationResp.LocationConstraint, fallbackRegion)
		}
		metadata := map[string]interface{}{
			"status":    "active",
			"createdAt": awsTimeText(bucket.CreationDate),
		}
		assets = append(assets, Asset{
			Provider: "aws",
			Type:     "objectstorage",
			ID:       bucketName,
			Name:     bucketName,
			Region:   bucketRegion,
			Metadata: metadata,
		})
	}
	return assets, nil
}

func (p AWSProvider) buildConfig(cred Credentials) (aws.Config, string, error) {
	accessKey := strings.TrimSpace(cred.AccessKey)
	secretKey := strings.TrimSpace(cred.SecretKey)
	if accessKey == "" || secretKey == "" {
		return aws.Config{}, "", fmt.Errorf("access key or secret key is empty")
	}
	if strings.Contains(accessKey, "*") || strings.Contains(secretKey, "*") {
		return aws.Config{}, "", fmt.Errorf("credential looks masked, please input original key in cloud account settings")
	}

	regionInput := strings.ToLower(strings.TrimSpace(cred.Region))
	region := p.defaultRegion
	if regionInput != "" && !awsRegionIsGlobal(regionInput) {
		region = regionInput
	}
	cfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(awscredentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		awsconfig.WithHTTPClient(&http.Client{Timeout: time.Duration(p.requestTimeoutS) * time.Second}),
	)
	if err != nil {
		return aws.Config{}, "", fmt.Errorf("load aws sdk config failed: %w", err)
	}
	return cfg, region, nil
}

func (p AWSProvider) resolveSyncRegions(cfg aws.Config, requestedRegion string, fallbackRegion string) ([]string, error) {
	raw := strings.TrimSpace(requestedRegion)
	if raw == "" || awsRegionIsGlobal(raw) {
		regions, err := p.discoverAWSRegions(cfg, fallbackRegion)
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

func (p AWSProvider) discoverAWSRegions(cfg aws.Config, fallbackRegion string) ([]string, error) {
	regionalCfg := cfg.Copy()
	regionalCfg.Region = fallbackRegion
	client := awsec2.NewFromConfig(regionalCfg)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.requestTimeoutS)*time.Second)
	defer cancel()
	resp, err := client.DescribeRegions(ctx, &awsec2.DescribeRegionsInput{})
	if err != nil {
		return nil, p.wrapSDKError("ec2.DescribeRegions", err)
	}
	result := make([]string, 0, len(resp.Regions))
	seen := map[string]struct{}{}
	for _, item := range resp.Regions {
		region := strings.ToLower(strings.TrimSpace(aws.ToString(item.RegionName)))
		if region == "" {
			continue
		}
		if _, exists := seen[region]; exists {
			continue
		}
		seen[region] = struct{}{}
		result = append(result, region)
	}
	return result, nil
}

func (p AWSProvider) shouldMock(cred Credentials) bool {
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

func (p AWSProvider) wrapSDKError(scene string, err error) error {
	requestID := ""
	var responseErr *awshttp.ResponseError
	if errors.As(err, &responseErr) {
		requestID = strings.TrimSpace(responseErr.ServiceRequestID())
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := strings.TrimSpace(apiErr.ErrorCode())
		message := strings.TrimSpace(apiErr.ErrorMessage())
		switch code {
		case "InvalidClientTokenId", "AuthFailure":
			message = message + "; please check accessKey is valid"
		case "SignatureDoesNotMatch":
			message = message + "; please verify accessKey/secretKey pairing is correct"
		case "UnrecognizedClientException":
			message = message + "; aws credentials are not recognized"
		}
		return fmt.Errorf("%s failed: code=%s message=%s requestId=%s", scene, code, message, requestID)
	}
	return fmt.Errorf("%s failed: %w", scene, err)
}

func awsRegionIsGlobal(raw string) bool {
	region := strings.ToLower(strings.TrimSpace(raw))
	return region == "" || region == "global" || region == "all" || region == "*"
}

func tagsFromAWSEC2(tags []awsec2types.Tag) map[string]interface{} {
	result := map[string]interface{}{}
	for _, tag := range tags {
		key := strings.TrimSpace(aws.ToString(tag.Key))
		if key == "" {
			continue
		}
		result[key] = strings.TrimSpace(aws.ToString(tag.Value))
	}
	return result
}

func awsEC2InstanceName(instance awsec2types.Instance) string {
	for _, tag := range instance.Tags {
		key := strings.TrimSpace(aws.ToString(tag.Key))
		if strings.EqualFold(key, "name") {
			return strings.TrimSpace(aws.ToString(tag.Value))
		}
	}
	return ""
}

func awsVPCName(item awsec2types.Vpc) string {
	for _, tag := range item.Tags {
		key := strings.TrimSpace(aws.ToString(tag.Key))
		if strings.EqualFold(key, "name") {
			return strings.TrimSpace(aws.ToString(tag.Value))
		}
	}
	return ""
}

func awsRDSConnection(item awsrdstypes.DBInstance) string {
	endpoint := strings.TrimSpace(aws.ToString(item.Endpoint.Address))
	if endpoint == "" {
		return ""
	}
	port := aws.ToInt32(item.Endpoint.Port)
	if port <= 0 {
		return endpoint
	}
	return endpoint + ":" + strconv.FormatInt(int64(port), 10)
}

func awsRDSVpcID(item awsrdstypes.DBInstance) string {
	if item.DBSubnetGroup == nil {
		return ""
	}
	return strings.TrimSpace(aws.ToString(item.DBSubnetGroup.VpcId))
}

func awsS3BucketRegion(location awss3types.BucketLocationConstraint, fallbackRegion string) string {
	value := strings.TrimSpace(string(location))
	if value == "" {
		return "us-east-1"
	}
	if strings.EqualFold(value, "EU") {
		return "eu-west-1"
	}
	return strings.ToLower(value)
}

func awsTimeText(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
