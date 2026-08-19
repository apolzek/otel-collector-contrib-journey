package resilienteexporter

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

var typeStr = component.MustNewType("resiliente")

func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		typeStr,
		createDefaultConfig,
		exporter.WithTraces(createTraces, component.StabilityLevelDevelopment),
	)
}

// Os defaults vêm dos construtores do core. Não invente números: use os
// padrões do projeto e deixe o usuário ajustar.
func createDefaultConfig() component.Config {
	return &Config{
		TimeoutConfig: exporterhelper.NewDefaultTimeoutConfig(),
		BackOffConfig: configretry.NewDefaultBackOffConfig(),
		QueueConfig:   configoptional.Default(exporterhelper.NewDefaultQueueConfig()),
	}
}

func createTraces(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Traces, error) {
	eCfg := cfg.(*Config)
	e := newExporter(eCfg)

	// A ordem em que o helper aplica as camadas, de fora para dentro:
	// fila -> batch -> retry -> timeout -> sua função de push.
	return exporterhelper.NewTraces(ctx, set, cfg, e.pushTraces,
		exporterhelper.WithQueue(eCfg.QueueConfig),
		exporterhelper.WithRetry(eCfg.BackOffConfig),
		exporterhelper.WithTimeout(eCfg.TimeoutConfig),
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
	)
}
