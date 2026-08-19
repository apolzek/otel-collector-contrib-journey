package tickreceiver

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

var typeStr = component.MustNewType("tick")

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithLogs(createLogs, component.StabilityLevelDevelopment),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		Interval: 10 * time.Second,
		Message:  "tick",
	}
}

// A diferença de assinatura para o exporter: o receiver recebe um consumer,
// que é o próximo elo do pipeline. É o Collector que monta essa cadeia.
func createLogs(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	next consumer.Logs,
) (receiver.Logs, error) {
	return newTickReceiver(cfg.(*Config), set.Logger, next), nil
}
