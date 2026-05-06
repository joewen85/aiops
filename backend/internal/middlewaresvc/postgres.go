package middlewaresvc

import (
	"context"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresDriver struct{}

func (PostgresDriver) Type() string { return "postgresql" }

func (PostgresDriver) Actions() []ActionSpec {
	return []ActionSpec{
		{Name: "version", Description: "查询 PostgreSQL 版本", RiskLevel: "P3"},
		{Name: "activity", Description: "查询活跃连接摘要", RiskLevel: "P3"},
		{Name: "terminate_backend", Description: "终止指定后端连接", RiskLevel: "P1", ConfirmationRequired: true, Params: map[string]interface{}{"pid": "number|required"}},
	}
}

func (d PostgresDriver) Check(ctx context.Context, instance Instance) (CheckResult, error) {
	if isMock(instance) {
		return mockCheck(instance), nil
	}
	start := time.Now()
	db, err := d.open(instance)
	if err != nil {
		return CheckResult{}, err
	}
	sqlDB, err := sqlDB(db)
	if err == nil {
		defer sqlDB.Close()
	}
	var version string
	if err := db.WithContext(ctx).Raw("select version()").Scan(&version).Error; err != nil {
		return CheckResult{}, err
	}
	return CheckResult{
		Healthy:   true,
		Status:    "healthy",
		Version:   version,
		Role:      "database",
		LatencyMS: time.Since(start).Milliseconds(),
	}, nil
}

func (d PostgresDriver) CollectMetrics(ctx context.Context, instance Instance) ([]Metric, error) {
	if isMock(instance) {
		return mockMetrics(instance), nil
	}
	db, err := d.open(instance)
	if err != nil {
		return nil, err
	}
	sqlDB, err := sqlDB(db)
	if err == nil {
		defer sqlDB.Close()
	}
	var connections int64
	_ = db.WithContext(ctx).Raw("select count(*) from pg_stat_activity").Scan(&connections).Error
	return []Metric{
		{Type: "connections", Value: float64(connections), Unit: "count"},
	}, nil
}

func (d PostgresDriver) Execute(ctx context.Context, instance Instance, action string, params map[string]interface{}) (ActionResult, error) {
	action = strings.ToLower(strings.TrimSpace(action))
	if isMock(instance) {
		return mockAction(action), nil
	}
	db, err := d.open(instance)
	if err != nil {
		return ActionResult{}, err
	}
	sqlDB, err := sqlDB(db)
	if err == nil {
		defer sqlDB.Close()
	}
	switch action {
	case "version":
		var version string
		err := db.WithContext(ctx).Raw("select version()").Scan(&version).Error
		return ActionResult{Status: "success", Message: "postgres version collected", Data: map[string]interface{}{"version": version}}, err
	case "activity":
		var count int64
		err := db.WithContext(ctx).Raw("select count(*) from pg_stat_activity").Scan(&count).Error
		return ActionResult{Status: "success", Message: "postgres activity collected", Data: map[string]interface{}{"connections": count}}, err
	case "terminate_backend":
		pid, ok := numberParam(params, "pid")
		if !ok || pid <= 0 {
			return ActionResult{}, ErrInvalidParams
		}
		var terminated bool
		err := db.WithContext(ctx).Raw("select pg_terminate_backend(?)", pid).Scan(&terminated).Error
		return ActionResult{Status: "success", Message: "postgres backend terminated", Data: map[string]interface{}{"pid": pid, "terminated": terminated}}, err
	default:
		return ActionResult{}, ErrUnsupportedAction
	}
}

func (PostgresDriver) open(instance Instance) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(postgresDSN(instance)), &gorm.Config{})
}
