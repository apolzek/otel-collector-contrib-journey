package queuewatchreceiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scraperhelper"

	"github.com/apolzek/otel-collector-contrib-journey/day16/queuewatchreceiver/internal/metadata"
)

// O tipo e o nível de estabilidade vêm do pacote gerado, não de constantes
// escritas à mão. Mudar o metadata.yaml e regerar muda os dois de uma vez.
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithMetrics(createMetrics, metadata.MetricsStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		ControllerConfig:     scraperhelper.NewDefaultControllerConfig(),
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		Queues:               []string{"pedidos"},
	}
}

func createMetrics(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	next consumer.Metrics,
) (receiver.Metrics, error) {
	rCfg := cfg.(*Config)

	s, err := newScraper(rCfg, set)
	if err != nil {
		return nil, err
	}

	sc, err := scraper.NewMetrics(s.scrape)
	if err != nil {
		return nil, err
	}

	// O controller é quem vira o receiver de fato: ele cria a goroutine, o
	// ticker e o encerramento.
	return scraperhelper.NewMetricsController(
		&rCfg.ControllerConfig, set, next,
		scraperhelper.AddScraper(metadata.Type, sc),
	)
}
