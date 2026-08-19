package queuewatchreceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"github.com/apolzek/otel-collector-contrib-journey/day16/queuewatchreceiver/internal/metadata"
	"github.com/apolzek/otel-collector-contrib-journey/day16/queuewatchreceiver/internal/metadatatest"
)

func TestScrape(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Queues = []string{"pedidos", "notas"}

	s, err := newScraper(cfg, receivertest.NewNopSettings(metadata.Type))
	require.NoError(t, err)

	md, err := s.scrape(context.Background())
	require.NoError(t, err)

	// A métrica messages.processed está com enabled false no metadata.yaml,
	// então ela NÃO sai por padrão, mesmo com o Record sendo chamado.
	require.Equal(t, 1, md.ResourceMetrics().Len())
	sm := md.ResourceMetrics().At(0).ScopeMetrics().At(0)
	require.Equal(t, 1, sm.Metrics().Len())

	m := sm.Metrics().At(0)
	assert.Equal(t, "queuewatch.queue.depth", m.Name())
	assert.Equal(t, 2, m.Gauge().DataPoints().Len())

	// O resource attribute declarado no metadata.yaml está lá.
	host, ok := md.ResourceMetrics().At(0).Resource().Attributes().Get("queuewatch.host")
	require.True(t, ok)
	assert.Equal(t, "localhost", host.Str())
}

// Ligar uma métrica opcional é configuração do usuário, não código.
func TestMetricaOpcionalLigada(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Queues = []string{"pedidos"}
	cfg.Metrics.QueuewatchMessagesProcessed.Enabled = true

	s, err := newScraper(cfg, receivertest.NewNopSettings(metadata.Type))
	require.NoError(t, err)

	md, err := s.scrape(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().Len())
}

// A telemetria INTERNA do componente também é testável, com o pacote
// metadatatest gerado ao lado do metadata.
func TestTelemetriaInterna(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	cfg := createDefaultConfig().(*Config)
	cfg.Queues = []string{"fila-inexistente"}

	s, err := newScraper(cfg, metadatatest.NewSettings(tt))
	require.NoError(t, err)
	s.profundidade = map[string]int64{} // força o caminho de erro

	_, err = s.scrape(context.Background())
	require.NoError(t, err)

	metadatatest.AssertEqualQueuewatchScrapeErrors(t, tt,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())
}

func TestConfigValidate(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	require.NoError(t, cfg.Validate())

	cfg.Queues = nil
	require.ErrorIs(t, cfg.Validate(), errSemFilas)
}
