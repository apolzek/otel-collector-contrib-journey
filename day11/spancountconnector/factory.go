package spancountconnector

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
)

var typeStr = component.MustNewType("spancount")

// Um connector é declarado por PAR de sinais. Este só sabe traces para
// metrics; suportar traces para logs seria outra opção WithTracesToLogs e
// outra função de criação.
func NewFactory() connector.Factory {
	return connector.NewFactory(
		typeStr,
		createDefaultConfig,
		connector.WithTracesToMetrics(createTracesToMetrics, component.StabilityLevelDevelopment),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		MetricName: "spans.count",
		Dimension:  "service.name",
	}
}

func createTracesToMetrics(
	_ context.Context,
	_ connector.Settings,
	cfg component.Config,
	next consumer.Metrics,
) (connector.Traces, error) {
	return newSpanCountConnector(cfg.(*Config), next), nil
}
