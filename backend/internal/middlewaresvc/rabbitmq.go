package middlewaresvc

import (
	"context"
	"strings"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

type RabbitMQDriver struct{}

func (RabbitMQDriver) Type() string { return "rabbitmq" }

func (RabbitMQDriver) Actions() []ActionSpec {
	return []ActionSpec{
		{Name: "overview", Description: "查询 RabbitMQ 连接状态", RiskLevel: "P3"},
		{Name: "queue_inspect", Description: "查询队列摘要", RiskLevel: "P3", Params: map[string]interface{}{"queue": "string|required"}},
		{Name: "queue_purge", Description: "清空队列消息", RiskLevel: "P1", ConfirmationRequired: true, Params: map[string]interface{}{"queue": "string|required"}},
	}
}

func (d RabbitMQDriver) Check(ctx context.Context, instance Instance) (CheckResult, error) {
	if isMock(instance) {
		return mockCheck(instance), nil
	}
	start := time.Now()
	conn, err := d.open(instance)
	if err != nil {
		return CheckResult{}, err
	}
	defer conn.Close()
	return CheckResult{
		Healthy:   true,
		Status:    "healthy",
		Version:   "amqp",
		Role:      "broker",
		LatencyMS: time.Since(start).Milliseconds(),
	}, nil
}

func (d RabbitMQDriver) CollectMetrics(ctx context.Context, instance Instance) ([]Metric, error) {
	if isMock(instance) {
		return mockMetrics(instance), nil
	}
	conn, err := d.open(instance)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	defer ch.Close()
	return []Metric{
		{Type: "connection", Value: 1, Unit: "bool", Data: map[string]interface{}{"connected": true}},
	}, nil
}

func (d RabbitMQDriver) Execute(ctx context.Context, instance Instance, action string, params map[string]interface{}) (ActionResult, error) {
	action = strings.ToLower(strings.TrimSpace(action))
	if isMock(instance) {
		return mockAction(action), nil
	}
	conn, err := d.open(instance)
	if err != nil {
		return ActionResult{}, err
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return ActionResult{}, err
	}
	defer ch.Close()
	switch action {
	case "overview":
		return ActionResult{Status: "success", Message: "rabbitmq connection ok", Data: map[string]interface{}{"connected": true}}, nil
	case "queue_inspect":
		queue, ok := stringParam(params, "queue")
		if !ok {
			return ActionResult{}, ErrInvalidParams
		}
		info, err := ch.QueueInspect(queue)
		return ActionResult{Status: "success", Message: "rabbitmq queue inspected", Data: map[string]interface{}{"queue": queue, "messages": info.Messages, "consumers": info.Consumers}}, err
	case "queue_purge":
		queue, ok := stringParam(params, "queue")
		if !ok {
			return ActionResult{}, ErrInvalidParams
		}
		count, err := ch.QueuePurge(queue, false)
		return ActionResult{Status: "success", Message: "rabbitmq queue purged", Data: map[string]interface{}{"queue": queue, "purged": count}}, err
	default:
		return ActionResult{}, ErrUnsupportedAction
	}
}

func (RabbitMQDriver) open(instance Instance) (*amqp091.Connection, error) {
	return amqp091.Dial(rabbitURL(instance))
}

func stringParam(params map[string]interface{}, key string) (string, bool) {
	value, ok := params[key]
	if !ok {
		return "", false
	}
	text, ok := value.(string)
	text = strings.TrimSpace(text)
	return text, ok && text != ""
}
