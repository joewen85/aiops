package cloud

import "strings"

const (
	ResourceTypeCloudServer     = "CloudServer"
	ResourceTypeMySQL           = "MySQL"
	ResourceTypePrivateNetwork  = "PrivateNetwork"
	ResourceTypeObjectStorage   = "ObjectStorage"
	ResourceTypeFileStorage     = "FileStorage"
	ResourceTypeContainerSvc    = "ContainerService"
	ResourceTypeLoadBalancer    = "LoadBalancer"
	ResourceTypeDNS             = "DNS"
	ResourceTypeSSLCertificate  = "SSLCertificate"
	ResourceTypeLogService      = "LogService"
	ResourceTypeGenericResource = "CloudResource"
)

var baseResourceTypeAliases = map[string]string{
	"compute":           ResourceTypeCloudServer,
	"vm":                ResourceTypeCloudServer,
	"ecs":               ResourceTypeCloudServer,
	"ec2":               ResourceTypeCloudServer,
	"mysql":             ResourceTypeMySQL,
	"rds_mysql":         ResourceTypeMySQL,
	"vpc":               ResourceTypePrivateNetwork,
	"private_network":   ResourceTypePrivateNetwork,
	"objectstorage":     ResourceTypeObjectStorage,
	"object_storage":    ResourceTypeObjectStorage,
	"s3":                ResourceTypeObjectStorage,
	"oss":               ResourceTypeObjectStorage,
	"filestorage":       ResourceTypeFileStorage,
	"file_storage":      ResourceTypeFileStorage,
	"nas":               ResourceTypeFileStorage,
	"containerservice":  ResourceTypeContainerSvc,
	"container_service": ResourceTypeContainerSvc,
	"k8s":               ResourceTypeContainerSvc,
	"loadbalancer":      ResourceTypeLoadBalancer,
	"load_balancer":     ResourceTypeLoadBalancer,
	"slb":               ResourceTypeLoadBalancer,
	"lb":                ResourceTypeLoadBalancer,
	"dns":               ResourceTypeDNS,
	"domain":            ResourceTypeDNS,
	"ssl":               ResourceTypeSSLCertificate,
	"sslcertificate":    ResourceTypeSSLCertificate,
	"ssl_certificate":   ResourceTypeSSLCertificate,
	"logservice":        ResourceTypeLogService,
	"log_service":       ResourceTypeLogService,
	"cls":               ResourceTypeLogService,
}

var BaseResourceTypes = []string{
	ResourceTypeCloudServer,
	ResourceTypeMySQL,
	ResourceTypePrivateNetwork,
	ResourceTypeObjectStorage,
	ResourceTypeFileStorage,
	ResourceTypeContainerSvc,
	ResourceTypeLoadBalancer,
	ResourceTypeDNS,
	ResourceTypeSSLCertificate,
	ResourceTypeLogService,
}

func NormalizeBaseResourceType(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return ResourceTypeGenericResource
	}
	if target, ok := baseResourceTypeAliases[normalized]; ok {
		return target
	}
	return ResourceTypeGenericResource
}

func IsSupportedBaseResourceType(raw string) bool {
	normalized := NormalizeBaseResourceType(raw)
	for _, item := range BaseResourceTypes {
		if normalized == item {
			return true
		}
	}
	return false
}
