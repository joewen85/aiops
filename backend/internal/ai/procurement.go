package ai

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"devops-system/backend/internal/cloud"
)

const ProcurementProtocolVersion = "aiops.procurement.v1alpha1"

type ProcurementProtocolSpec struct {
	ProtocolVersion        string   `json:"protocolVersion"`
	SupportedActions       []string `json:"supportedActions"`
	SupportedProviders     []string `json:"supportedProviders"`
	SupportedResourceTypes []string `json:"supportedResourceTypes"`
	SupportsDryRun         bool     `json:"supportsDryRun"`
	SupportsApprovalFlow   bool     `json:"supportsApprovalFlow"`
}

type ProcurementNLRequest struct {
	RequestID         string                 `json:"requestId"`
	Message           string                 `json:"message"`
	PreferredProvider string                 `json:"preferredProvider"`
	Region            string                 `json:"region"`
	Quantity          int                    `json:"quantity"`
	BudgetLimit       float64                `json:"budgetLimit"`
	Metadata          map[string]interface{} `json:"metadata"`
}

type ProcurementIntent struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	IntentID        string                 `json:"intentId"`
	Action          string                 `json:"action"`
	Provider        string                 `json:"provider"`
	ResourceType    string                 `json:"resourceType"`
	Region          string                 `json:"region"`
	Quantity        int                    `json:"quantity"`
	BudgetLimit     float64                `json:"budgetLimit"`
	RawMessage      string                 `json:"rawMessage"`
	Constraints     map[string]interface{} `json:"constraints"`
}

type ProcurementPlanStep struct {
	Order      int                    `json:"order"`
	Name       string                 `json:"name"`
	Action     string                 `json:"action"`
	Endpoint   string                 `json:"endpoint"`
	Parameters map[string]interface{} `json:"parameters"`
}

type ProcurementPlan struct {
	ProtocolVersion  string                `json:"protocolVersion"`
	PlanID           string                `json:"planId"`
	Intent           ProcurementIntent     `json:"intent"`
	EstimatedCost    float64               `json:"estimatedCost"`
	Currency         string                `json:"currency"`
	RequiresApproval bool                  `json:"requiresApproval"`
	SafetyChecks     []string              `json:"safetyChecks"`
	Steps            []ProcurementPlanStep `json:"steps"`
}

type ProcurementExecutionResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	ExecutionID     string                 `json:"executionId"`
	PlanID          string                 `json:"planId"`
	Status          string                 `json:"status"`
	Summary         map[string]interface{} `json:"summary"`
}

type ProcurementEngine interface {
	ProtocolSpec() ProcurementProtocolSpec
	ParseIntent(req ProcurementNLRequest) (ProcurementIntent, []string, error)
	BuildPlan(intent ProcurementIntent) (ProcurementPlan, error)
	ExecutePlan(plan ProcurementPlan, dryRun bool) (ProcurementExecutionResult, error)
}

type StubProcurementEngine struct{}

func NewStubProcurementEngine() ProcurementEngine {
	return StubProcurementEngine{}
}

func (StubProcurementEngine) ProtocolSpec() ProcurementProtocolSpec {
	return ProcurementProtocolSpec{
		ProtocolVersion:        ProcurementProtocolVersion,
		SupportedActions:       []string{"purchase", "renew", "upgrade"},
		SupportedProviders:     []string{"aws", "aliyun", "tencent", "huawei"},
		SupportedResourceTypes: cloud.BaseResourceTypes,
		SupportsDryRun:         true,
		SupportsApprovalFlow:   true,
	}
}

func (StubProcurementEngine) ParseIntent(req ProcurementNLRequest) (ProcurementIntent, []string, error) {
	message := strings.TrimSpace(req.Message)
	if message == "" {
		return ProcurementIntent{}, nil, fmt.Errorf("message cannot be empty")
	}
	provider := normalizeProvider(req.PreferredProvider, message)
	region := normalizeRegion(req.Region, message)
	quantity := normalizeQuantity(req.Quantity, message)
	resourceType := normalizeResourceType(message)
	action := normalizeAction(message)

	intent := ProcurementIntent{
		ProtocolVersion: ProcurementProtocolVersion,
		IntentID:        buildID("intent"),
		Action:          action,
		Provider:        provider,
		ResourceType:    resourceType,
		Region:          region,
		Quantity:        quantity,
		BudgetLimit:     req.BudgetLimit,
		RawMessage:      message,
		Constraints: map[string]interface{}{
			"metadata": req.Metadata,
		},
	}

	clarifications := make([]string, 0, 2)
	if provider == "" {
		clarifications = append(clarifications, "未识别云厂商，建议补充 aws/aliyun/tencent/huawei")
	}
	if region == "" {
		clarifications = append(clarifications, "未识别地域，建议补充如 ap-southeast-1")
	}
	return intent, clarifications, nil
}

func (StubProcurementEngine) BuildPlan(intent ProcurementIntent) (ProcurementPlan, error) {
	if strings.TrimSpace(intent.RawMessage) == "" {
		return ProcurementPlan{}, fmt.Errorf("intent.rawMessage cannot be empty")
	}
	if intent.Quantity <= 0 {
		intent.Quantity = 1
	}
	intent.ResourceType = cloud.NormalizeBaseResourceType(intent.ResourceType)
	estimatedCost := estimateCost(intent.ResourceType, intent.Quantity)
	requiresApproval := estimatedCost >= 5000 || intent.Quantity >= 5
	if intent.BudgetLimit > 0 && estimatedCost > intent.BudgetLimit {
		requiresApproval = true
	}

	plan := ProcurementPlan{
		ProtocolVersion:  ProcurementProtocolVersion,
		PlanID:           buildID("plan"),
		Intent:           intent,
		EstimatedCost:    estimatedCost,
		Currency:         "CNY",
		RequiresApproval: requiresApproval,
		SafetyChecks: []string{
			"检查账号可用与校验状态",
			"检查配额与地域库存",
			"检查预算阈值与审批策略",
		},
		Steps: []ProcurementPlanStep{
			{
				Order:    1,
				Name:     "校验云账号",
				Action:   "verify_account",
				Endpoint: "/api/v1/cloud/accounts/:id/verify",
				Parameters: map[string]interface{}{
					"provider": intent.Provider,
				},
			},
			{
				Order:    2,
				Name:     "预算与审批检查",
				Action:   "check_budget_and_approval",
				Endpoint: "/api/v1/aiops/procurement/plans",
				Parameters: map[string]interface{}{
					"budgetLimit": intent.BudgetLimit,
				},
			},
			{
				Order:    3,
				Name:     "提交资源采购执行",
				Action:   "execute_purchase",
				Endpoint: "/api/v1/aiops/procurement/executions",
				Parameters: map[string]interface{}{
					"provider":     intent.Provider,
					"resourceType": intent.ResourceType,
					"region":       intent.Region,
					"quantity":     intent.Quantity,
				},
			},
		},
	}
	return plan, nil
}

func (StubProcurementEngine) ExecutePlan(plan ProcurementPlan, dryRun bool) (ProcurementExecutionResult, error) {
	if strings.TrimSpace(plan.PlanID) == "" {
		return ProcurementExecutionResult{}, fmt.Errorf("plan.planId cannot be empty")
	}
	status := "accepted"
	if dryRun {
		status = "dry_run"
	}
	return ProcurementExecutionResult{
		ProtocolVersion: ProcurementProtocolVersion,
		ExecutionID:     buildID("exec"),
		PlanID:          plan.PlanID,
		Status:          status,
		Summary: map[string]interface{}{
			"dryRun":           dryRun,
			"provider":         plan.Intent.Provider,
			"resourceType":     plan.Intent.ResourceType,
			"quantity":         plan.Intent.Quantity,
			"estimatedCost":    plan.EstimatedCost,
			"requiresApproval": plan.RequiresApproval,
		},
	}, nil
}

func normalizeProvider(preferredProvider string, message string) string {
	candidate := strings.ToLower(strings.TrimSpace(preferredProvider))
	switch candidate {
	case "aws", "aliyun", "tencent", "huawei":
		return candidate
	}
	text := strings.ToLower(message)
	switch {
	case strings.Contains(text, "aws"), strings.Contains(text, "ec2"):
		return "aws"
	case strings.Contains(text, "aliyun"), strings.Contains(text, "阿里"):
		return "aliyun"
	case strings.Contains(text, "tencent"), strings.Contains(text, "腾讯"):
		return "tencent"
	case strings.Contains(text, "huawei"), strings.Contains(text, "华为"):
		return "huawei"
	default:
		return ""
	}
}

func normalizeResourceType(message string) string {
	text := strings.ToLower(message)
	switch {
	case strings.Contains(text, "mysql"), strings.Contains(text, "数据库"):
		return cloud.ResourceTypeMySQL
	case strings.Contains(text, "私有网络"), strings.Contains(text, "vpc"):
		return cloud.ResourceTypePrivateNetwork
	case strings.Contains(text, "对象存储"), strings.Contains(text, "s3"), strings.Contains(text, "oss"):
		return cloud.ResourceTypeObjectStorage
	case strings.Contains(text, "文件存储"), strings.Contains(text, "nas"):
		return cloud.ResourceTypeFileStorage
	case strings.Contains(text, "容器"), strings.Contains(text, "k8s"), strings.Contains(text, "kubernetes"):
		return cloud.ResourceTypeContainerSvc
	case strings.Contains(text, "负载均衡"), strings.Contains(text, "slb"), strings.Contains(text, "lb"):
		return cloud.ResourceTypeLoadBalancer
	case strings.Contains(text, "域名"), strings.Contains(text, "dns"):
		return cloud.ResourceTypeDNS
	case strings.Contains(text, "证书"), strings.Contains(text, "ssl"):
		return cloud.ResourceTypeSSLCertificate
	case strings.Contains(text, "日志"):
		return cloud.ResourceTypeLogService
	case strings.Contains(text, "服务器"), strings.Contains(text, "ecs"), strings.Contains(text, "ec2"), strings.Contains(text, "vm"):
		return cloud.ResourceTypeCloudServer
	default:
		return cloud.ResourceTypeCloudServer
	}
}

func normalizeAction(message string) string {
	text := strings.ToLower(message)
	switch {
	case strings.Contains(text, "续费"), strings.Contains(text, "renew"):
		return "renew"
	case strings.Contains(text, "升级"), strings.Contains(text, "upgrade"):
		return "upgrade"
	default:
		return "purchase"
	}
}

func normalizeRegion(region string, message string) string {
	if strings.TrimSpace(region) != "" {
		return strings.TrimSpace(region)
	}
	matches := regexp.MustCompile(`[a-z]{2}-[a-z-]+-\d`).FindString(strings.ToLower(message))
	return matches
}

func normalizeQuantity(quantity int, message string) int {
	if quantity > 0 {
		return quantity
	}
	found := regexp.MustCompile(`\d+`).FindAllString(message, -1)
	if len(found) == 0 {
		return 1
	}
	maxValue := 0
	for _, item := range found {
		parsed, err := strconv.Atoi(item)
		if err != nil {
			continue
		}
		if parsed > maxValue {
			maxValue = parsed
		}
	}
	if maxValue <= 0 {
		return 1
	}
	return maxValue
}

func estimateCost(resourceType string, quantity int) float64 {
	unitCost := map[string]float64{
		cloud.ResourceTypeCloudServer:    800,
		cloud.ResourceTypeMySQL:          1200,
		cloud.ResourceTypePrivateNetwork: 300,
		cloud.ResourceTypeObjectStorage:  200,
		cloud.ResourceTypeFileStorage:    400,
		cloud.ResourceTypeContainerSvc:   1500,
		cloud.ResourceTypeLoadBalancer:   600,
		cloud.ResourceTypeDNS:            80,
		cloud.ResourceTypeSSLCertificate: 120,
		cloud.ResourceTypeLogService:     350,
	}
	base, exists := unitCost[resourceType]
	if !exists {
		base = 500
	}
	if quantity <= 0 {
		quantity = 1
	}
	return base * float64(quantity)
}

func buildID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
