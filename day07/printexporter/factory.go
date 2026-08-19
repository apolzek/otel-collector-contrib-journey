package printexporter

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// typeStr é o nome do componente no YAML: exporters::print.
// MustNewType entra em pânico se o nome tiver caractere inválido, e é chamado
// só uma vez, na inicialização do pacote.
var typeStr = component.MustNewType("print")

// NewFactory é a ÚNICA função que um componente precisa exportar. É ela que o
// OCB escreve no components.go da distribuição.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		typeStr,
		createDefaultConfig,
		exporter.WithTraces(createTraces, component.StabilityLevelDevelopment),
		exporter.WithLogs(createLogs, component.StabilityLevelDevelopment),
	)
}

// createDefaultConfig devolve o Config já preenchido com os padrões. O confmap
// despeja o YAML do usuário POR CIMA deste valor, então tudo que o usuário não
// escrever fica com o default daqui.
func createDefaultConfig() component.Config {
	return &Config{
		Path:   "stdout",
		Prefix: "",
	}
}

func createTraces(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Traces, error) {
	// O cast é seguro: o cfg que chega aqui saiu de createDefaultConfig.
	pCfg := cfg.(*Config)
	e := newPrintExporter(pCfg, set.Logger)

	return exporterhelper.NewTraces(ctx, set, cfg, e.pushTraces,
		exporterhelper.WithStart(e.start),
		exporterhelper.WithShutdown(e.shutdown),
		// Este exporter só lê os dados, nunca escreve. Declarar isso evita
		// que o Collector clone a telemetria à toa.
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
	)
}

func createLogs(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Logs, error) {
	pCfg := cfg.(*Config)
	e := newPrintExporter(pCfg, set.Logger)

	return exporterhelper.NewLogs(ctx, set, cfg, e.pushLogs,
		exporterhelper.WithStart(e.start),
		exporterhelper.WithShutdown(e.shutdown),
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
	)
}
