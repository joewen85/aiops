package middlewaresvc

import (
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	DefaultTimeout = 8 * time.Second
)

type Instance struct {
	ID        uint
	Type      string
	Endpoint  string
	Env       string
	Username  string
	Password  string
	Token     string
	TLSEnable bool
}

type CheckResult struct {
	Healthy   bool                   `json:"healthy"`
	Status    string                 `json:"status"`
	Version   string                 `json:"version,omitempty"`
	Role      string                 `json:"role,omitempty"`
	LatencyMS int64                  `json:"latencyMs"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

type Metric struct {
	Type  string                 `json:"type"`
	Value float64                `json:"value"`
	Unit  string                 `json:"unit,omitempty"`
	Data  map[string]interface{} `json:"data,omitempty"`
}

type ActionResult struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

type Driver interface {
	Type() string
	Actions() []ActionSpec
	Check(ctx context.Context, instance Instance) (CheckResult, error)
	CollectMetrics(ctx context.Context, instance Instance) ([]Metric, error)
	Execute(ctx context.Context, instance Instance, action string, params map[string]interface{}) (ActionResult, error)
}

type ActionSpec struct {
	Name                 string                 `json:"name"`
	Description          string                 `json:"description"`
	RiskLevel            string                 `json:"riskLevel"`
	ConfirmationRequired bool                   `json:"confirmationRequired"`
	Params               map[string]interface{} `json:"params,omitempty"`
}

func DriverFor(kind string) (Driver, bool) {
	switch NormalizeType(kind) {
	case "redis":
		return RedisDriver{}, true
	case "postgresql":
		return PostgresDriver{}, true
	case "rabbitmq":
		return RabbitMQDriver{}, true
	default:
		return nil, false
	}
}

func NormalizeType(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "redis":
		return "redis"
	case "postgres", "postgresql", "pg":
		return "postgresql"
	case "rabbit", "rabbitmq", "amqp":
		return "rabbitmq"
	default:
		return ""
	}
}

func ValidateEndpoint(instance Instance) error {
	endpoint := strings.TrimSpace(instance.Endpoint)
	if endpoint == "" {
		return errors.New("endpoint is required")
	}
	if strings.HasPrefix(endpoint, "mock://") {
		return nil
	}
	kind := NormalizeType(instance.Type)
	parsed, err := url.Parse(endpoint)
	if err == nil && parsed.Scheme != "" {
		if !schemeAllowed(kind, parsed.Scheme) {
			return fmt.Errorf("endpoint scheme %q does not match middleware type", parsed.Scheme)
		}
		if parsed.Hostname() == "" {
			return errors.New("endpoint host is required")
		}
		if err := validateEndpointHost(parsed.Hostname(), instance.Env); err != nil {
			return err
		}
		return nil
	}
	host, _, splitErr := net.SplitHostPort(endpoint)
	if splitErr != nil || host == "" {
		return errors.New("endpoint must be URL or host:port")
	}
	return validateEndpointHost(host, instance.Env)
}

func schemeAllowed(kind string, scheme string) bool {
	scheme = strings.ToLower(strings.TrimSpace(scheme))
	switch kind {
	case "redis":
		return scheme == "redis" || scheme == "rediss"
	case "postgresql":
		return scheme == "postgres" || scheme == "postgresql"
	case "rabbitmq":
		return scheme == "amqp" || scheme == "amqps"
	default:
		return false
	}
}

func validateEndpointHost(host string, env string) error {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return errors.New("endpoint host is required")
	}
	if host == "metadata.google.internal" || strings.HasSuffix(host, ".metadata.google.internal") {
		return errors.New("metadata endpoint is not allowed")
	}
	ip := net.ParseIP(host)
	if ip == nil {
		ips, err := net.LookupIP(host)
		if err == nil && len(ips) > 0 {
			ip = ips[0]
		}
	}
	if ip == nil {
		return nil
	}
	if ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() {
		return errors.New("unsafe endpoint address is not allowed")
	}
	if strings.EqualFold(strings.TrimSpace(env), "prod") && ip.IsLoopback() {
		return errors.New("prod middleware endpoint cannot use loopback address")
	}
	return nil
}

func redisOptions(instance Instance) (*redis.Options, error) {
	endpoint := strings.TrimSpace(instance.Endpoint)
	if strings.HasPrefix(endpoint, "redis://") || strings.HasPrefix(endpoint, "rediss://") {
		options, err := redis.ParseURL(endpoint)
		if err != nil {
			return nil, err
		}
		if instance.Password != "" {
			options.Password = instance.Password
		}
		if instance.Username != "" {
			options.Username = instance.Username
		}
		if instance.TLSEnable || strings.HasPrefix(endpoint, "rediss://") {
			options.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		}
		return options, nil
	}
	return &redis.Options{
		Addr:      endpoint,
		Username:  instance.Username,
		Password:  instance.Password,
		TLSConfig: tlsConfigIfEnabled(instance.TLSEnable),
	}, nil
}

func postgresDSN(instance Instance) string {
	endpoint := strings.TrimSpace(instance.Endpoint)
	if strings.HasPrefix(endpoint, "postgres://") || strings.HasPrefix(endpoint, "postgresql://") {
		return endpoint
	}
	user := url.QueryEscape(defaultString(instance.Username, "postgres"))
	password := url.QueryEscape(instance.Password)
	auth := user
	if password != "" {
		auth += ":" + password
	}
	sslMode := "disable"
	if instance.TLSEnable {
		sslMode = "require"
	}
	return fmt.Sprintf("postgres://%s@%s/postgres?sslmode=%s", auth, endpoint, sslMode)
}

func rabbitURL(instance Instance) string {
	endpoint := strings.TrimSpace(instance.Endpoint)
	if strings.HasPrefix(endpoint, "amqp://") || strings.HasPrefix(endpoint, "amqps://") {
		return endpoint
	}
	scheme := "amqp"
	if instance.TLSEnable {
		scheme = "amqps"
	}
	user := url.QueryEscape(defaultString(instance.Username, "guest"))
	password := url.QueryEscape(defaultString(instance.Password, "guest"))
	return fmt.Sprintf("%s://%s:%s@%s/", scheme, user, password, endpoint)
}

func tlsConfigIfEnabled(enabled bool) *tls.Config {
	if !enabled {
		return nil
	}
	return &tls.Config{MinVersion: tls.VersionTLS12}
}

func isMock(instance Instance) bool {
	return strings.HasPrefix(strings.TrimSpace(instance.Endpoint), "mock://")
}

func mockCheck(instance Instance) CheckResult {
	return CheckResult{
		Healthy:   true,
		Status:    "healthy",
		Version:   "mock-" + NormalizeType(instance.Type),
		Role:      "primary",
		LatencyMS: 1,
		Details:   map[string]interface{}{"mock": true},
	}
}

func mockMetrics(instance Instance) []Metric {
	return []Metric{
		{Type: "connections", Value: 12, Unit: "count", Data: map[string]interface{}{"mock": true}},
		{Type: "memory_usage", Value: 64, Unit: "MiB", Data: map[string]interface{}{"mock": true}},
	}
}

func mockAction(action string) ActionResult {
	return ActionResult{Status: "success", Message: "mock action executed", Data: map[string]interface{}{"action": action, "mock": true}}
}

func defaultString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func toFloat(raw string) float64 {
	value, _ := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	return value
}

func sqlDB(db *gorm.DB) (*sql.DB, error) {
	return db.DB()
}

var ErrUnsupportedAction = errors.New("unsupported middleware action")
var ErrInvalidParams = errors.New("invalid middleware action params")

func numberParam(params map[string]interface{}, key string) (int64, bool) {
	value, ok := params[key]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		return int64(typed), true
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}
