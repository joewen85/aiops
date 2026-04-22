package config

import (
	"os"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName   string
	AppEnv    string
	AppPort   string
	JWTSecret string
	JWTExpire int

	CORSAllowOrigins            string
	CORSAllowCredentials        bool
	ABACHeaderSignSecret        string
	PermissionRuntimeCacheTTLMS int

	PostgresDSN string
	RedisAddr   string
	RedisPass   string
	RedisDB     int
	RabbitMQURL string

	AnsibleBin     string
	PlaybookTmpDir string

	OpenAIEndpoint    string
	OpenAIAPIKey      string
	AnthropicEndpoint string
	AnthropicAPIKey   string

	CloudSDKMockEnabled       bool
	CloudSDKMockAKPrefix      string
	CloudSDKMockSKPrefix      string
	CloudCredentialEncryptKey string
	AWSDefaultRegion          string
	AWSSDKTimeoutSeconds      int
	AWSSDKPageLimit           int
	AliyunDefaultRegion       string
	AliyunSDKTimeoutSeconds   int
	AliyunSDKPageLimit        int
	TencentDefaultRegion      string
	TencentSDKTimeoutSeconds  int
	TencentSDKPageLimit       int
	HuaweiDefaultRegion       string
	HuaweiSDKTimeoutSeconds   int
	HuaweiSDKPageLimit        int
}

var loadEnvOnce sync.Once

func Load() Config {
	loadEnvFiles()

	return Config{
		AppName:                     env("APP_NAME", "sme-devops"),
		AppEnv:                      env("APP_ENV", "development"),
		AppPort:                     env("APP_PORT", "8080"),
		JWTSecret:                   env("JWT_SECRET", "change-me"),
		JWTExpire:                   envInt("JWT_EXPIRE_HOURS", 24),
		CORSAllowOrigins:            env("CORS_ALLOW_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173,http://localhost:4173,http://127.0.0.1:4173"),
		CORSAllowCredentials:        envBool("CORS_ALLOW_CREDENTIALS", false),
		ABACHeaderSignSecret:        env("ABAC_HEADER_SIGN_SECRET", ""),
		PermissionRuntimeCacheTTLMS: envInt("PERMISSION_RUNTIME_CACHE_TTL_MS", 0),
		PostgresDSN:                 env("POSTGRES_DSN", ""),
		RedisAddr:                   env("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPass:                   env("REDIS_PASSWORD", ""),
		RedisDB:                     envInt("REDIS_DB", 0),
		RabbitMQURL:                 env("RABBITMQ_URL", ""),
		AnsibleBin:                  env("ANSIBLE_BIN", "ansible-playbook"),
		PlaybookTmpDir:              env("PLAYBOOK_TMP_DIR", "/tmp/devops-playbooks"),
		OpenAIEndpoint:              env("OPENAI_ENDPOINT", "https://api.openai.com/v1/chat/completions"),
		OpenAIAPIKey:                env("OPENAI_API_KEY", ""),
		AnthropicEndpoint:           env("ANTHROPIC_ENDPOINT", "https://api.anthropic.com/v1/messages"),
		AnthropicAPIKey:             env("ANTHROPIC_API_KEY", ""),
		CloudSDKMockEnabled:         envBool("CLOUD_SDK_MOCK_ENABLED", false),
		CloudSDKMockAKPrefix:        env("CLOUD_SDK_MOCK_AK_PREFIX", "mock_"),
		CloudSDKMockSKPrefix:        env("CLOUD_SDK_MOCK_SK_PREFIX", "mock_"),
		CloudCredentialEncryptKey:   env("CLOUD_CREDENTIAL_ENCRYPT_KEY", ""),
		AWSDefaultRegion:            env("AWS_DEFAULT_REGION", "us-east-1"),
		AWSSDKTimeoutSeconds:        envInt("AWS_SDK_TIMEOUT_SECONDS", 10),
		AWSSDKPageLimit:             envInt("AWS_SDK_PAGE_LIMIT", 100),
		AliyunDefaultRegion:         env("ALIYUN_DEFAULT_REGION", "cn-hangzhou"),
		AliyunSDKTimeoutSeconds:     envInt("ALIYUN_SDK_TIMEOUT_SECONDS", 10),
		AliyunSDKPageLimit:          envInt("ALIYUN_SDK_PAGE_LIMIT", 100),
		TencentDefaultRegion:        env("TENCENT_DEFAULT_REGION", "ap-guangzhou"),
		TencentSDKTimeoutSeconds:    envInt("TENCENT_SDK_TIMEOUT_SECONDS", 10),
		TencentSDKPageLimit:         envInt("TENCENT_SDK_PAGE_LIMIT", 100),
		HuaweiDefaultRegion:         env("HUAWEI_DEFAULT_REGION", "cn-north-4"),
		HuaweiSDKTimeoutSeconds:     envInt("HUAWEI_SDK_TIMEOUT_SECONDS", 10),
		HuaweiSDKPageLimit:          envInt("HUAWEI_SDK_PAGE_LIMIT", 100),
	}
}

func loadEnvFiles() {
	loadEnvOnce.Do(func() {
		_ = godotenv.Load(".env")
		_ = godotenv.Load("backend/.env")
	})
}

func env(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}
