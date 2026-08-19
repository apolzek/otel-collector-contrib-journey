package tagprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

var typeStr = component.MustNewType("tag")

func NewFactory() processor.Factory {
	return processor.NewFactory(
		typeStr,
		createDefaultConfig,
		processor.WithTraces(createTraces, component.StabilityLevelDevelopment),
		processor.WithLogs(createLogs, component.StabilityLevelDevelopment),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		Attributes: map[string]string{},
		Overwrite:  false,
	}
}

// O processor recebe as duas pontas: o cfg e o próximo consumer. Ele fica no
// meio do pipeline, então é ao mesmo tempo consumer (para quem está atrás) e
// produtor (para quem está na frente).
func createTraces(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	next consumer.Traces,
) (processor.Traces, error) {
	p := newTagProcessor(cfg.(*Config))
	return processorhelper.NewTraces(ctx, set, cfg, next, p.processTraces,
		processorhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
}

func createLogs(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	next consumer.Logs,
) (processor.Logs, error) {
	p := newTagProcessor(cfg.(*Config))
	return processorhelper.NewLogs(ctx, set, cfg, next, p.processLogs,
		processorhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
}
