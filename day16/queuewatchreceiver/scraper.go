package queuewatchreceiver

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"

	"github.com/apolzek/otel-collector-contrib-journey/day16/queuewatchreceiver/internal/metadata"
)

// queueWatchScraper implementa só a coleta. Intervalo, timeout, tratamento de
// erro parcial e telemetria do scrape são do scraperhelper.
type queueWatchScraper struct {
	cfg *Config
	mb  *metadata.MetricsBuilder

	// Telemetria interna do componente, também gerada pelo mdatagen a partir
	// do bloco telemetry do metadata.yaml.
	tb *metadata.TelemetryBuilder

	// Simula um estado externo. Num componente real seria um cliente.
	profundidade map[string]int64
}

func newScraper(cfg *Config, set receiver.Settings) (*queueWatchScraper, error) {
	tb, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}
	s := &queueWatchScraper{
		cfg:          cfg,
		mb:           metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, set),
		tb:           tb,
		profundidade: map[string]int64{},
	}
	for i, q := range cfg.Queues {
		s.profundidade[q] = int64(10 * (i + 1))
	}
	return s, nil
}

// scrape é chamada a cada collection_interval. Repare que ela não sabe nada
// sobre intervalo, ticker ou goroutine.
func (s *queueWatchScraper) scrape(_ context.Context) (pmetric.Metrics, error) {
	agora := pcommon.NewTimestampFromTime(time.Now())

	for _, fila := range s.cfg.Queues {
		v, ok := s.profundidade[fila]
		if !ok {
			// Contador de erro da telemetria interna do componente.
			s.tb.QueuewatchScrapeErrors.Add(context.Background(), 1)
			continue
		}
		// Os métodos Record vêm do mdatagen, um por métrica declarada.
		// Se o usuário desligou a métrica no YAML, a chamada não faz nada.
		s.mb.RecordQueuewatchQueueDepthDataPoint(agora, v, fila)
		s.mb.RecordQueuewatchMessagesProcessedDataPoint(agora, v*3, fila)
	}

	// O ResourceBuilder também é gerado, com um setter por resource attribute.
	rb := s.mb.NewResourceBuilder()
	rb.SetQueuewatchHost("localhost")
	s.mb.EmitForResource(metadata.WithResource(rb.Emit()))

	return s.mb.Emit(), nil
}
