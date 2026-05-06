package middlewaresvc

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisDriver struct{}

func (RedisDriver) Type() string { return "redis" }

func (RedisDriver) Actions() []ActionSpec {
	return []ActionSpec{
		{Name: "info", Description: "查询 Redis INFO 摘要", RiskLevel: "P3"},
		{Name: "dbsize", Description: "查询 key 数量", RiskLevel: "P3"},
		{Name: "flushdb", Description: "清空当前 Redis DB", RiskLevel: "P1", ConfirmationRequired: true},
	}
}

func (d RedisDriver) Check(ctx context.Context, instance Instance) (CheckResult, error) {
	if isMock(instance) {
		return mockCheck(instance), nil
	}
	start := time.Now()
	client, err := d.client(instance)
	if err != nil {
		return CheckResult{}, err
	}
	defer client.Close()
	if err := client.Ping(ctx).Err(); err != nil {
		return CheckResult{}, err
	}
	info, _ := client.Info(ctx, "server", "replication").Result()
	return CheckResult{
		Healthy:   true,
		Status:    "healthy",
		Version:   parseRedisInfo(info, "redis_version"),
		Role:      parseRedisInfo(info, "role"),
		LatencyMS: time.Since(start).Milliseconds(),
		Details: map[string]interface{}{
			"mode": parseRedisInfo(info, "redis_mode"),
		},
	}, nil
}

func (d RedisDriver) CollectMetrics(ctx context.Context, instance Instance) ([]Metric, error) {
	if isMock(instance) {
		return mockMetrics(instance), nil
	}
	client, err := d.client(instance)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	info, err := client.Info(ctx, "clients", "memory", "stats").Result()
	if err != nil {
		return nil, err
	}
	return []Metric{
		{Type: "connected_clients", Value: toFloat(parseRedisInfo(info, "connected_clients")), Unit: "count"},
		{Type: "used_memory", Value: toFloat(parseRedisInfo(info, "used_memory")), Unit: "bytes"},
		{Type: "instantaneous_ops_per_sec", Value: toFloat(parseRedisInfo(info, "instantaneous_ops_per_sec")), Unit: "ops/s"},
	}, nil
}

func (d RedisDriver) Execute(ctx context.Context, instance Instance, action string, params map[string]interface{}) (ActionResult, error) {
	action = strings.ToLower(strings.TrimSpace(action))
	if isMock(instance) {
		return mockAction(action), nil
	}
	client, err := d.client(instance)
	if err != nil {
		return ActionResult{}, err
	}
	defer client.Close()
	switch action {
	case "info":
		info, err := client.Info(ctx).Result()
		return ActionResult{Status: "success", Message: "redis info collected", Data: map[string]interface{}{"info": truncateString(info, 4096)}}, err
	case "dbsize":
		size, err := client.DBSize(ctx).Result()
		return ActionResult{Status: "success", Message: "redis dbsize collected", Data: map[string]interface{}{"dbsize": size}}, err
	case "flushdb":
		err := client.FlushDB(ctx).Err()
		return ActionResult{Status: "success", Message: "redis db flushed"}, err
	default:
		return ActionResult{}, ErrUnsupportedAction
	}
}

func (RedisDriver) client(instance Instance) (*redis.Client, error) {
	options, err := redisOptions(instance)
	if err != nil {
		return nil, err
	}
	return redis.NewClient(options), nil
}

func parseRedisInfo(info string, key string) string {
	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key+":") {
			return strings.TrimSpace(strings.TrimPrefix(line, key+":"))
		}
	}
	return ""
}

func truncateString(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
