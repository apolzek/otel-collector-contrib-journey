package spancountconnector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func tracesCom(servico string, nSpans int) ptrace.Traces {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", servico)
	ss := rs.ScopeSpans().AppendEmpty()
	for i := 0; i < nSpans; i++ {
		ss.Spans().AppendEmpty().SetName("span")
	}
	return td
}

func TestConsumeTracesGeraMetrica(t *testing.T) {
	sink := new(consumertest.MetricsSink)

	factory := NewFactory()
	conn, err := factory.CreateTracesToMetrics(
		context.Background(),
		connectortest.NewNopSettings(typeStr),
		factory.CreateDefaultConfig(),
		sink,
	)
	require.NoError(t, err)

	require.NoError(t, conn.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, conn.ConsumeTraces(context.Background(), tracesCom("api", 3)))
	require.NoError(t, conn.Shutdown(context.Background()))

	require.Len(t, sink.AllMetrics(), 1)
	m := sink.AllMetrics()[0].ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	assert.Equal(t, "spans.count", m.Name())

	dp := m.Sum().DataPoints().At(0)
	assert.Equal(t, int64(3), dp.IntValue())

	servico, ok := dp.Attributes().Get("service.name")
	require.True(t, ok)
	assert.Equal(t, "api", servico.Str())
}

// Lote vazio não deve gerar métrica nenhuma: emitir ponto zero à toa polui o
// backend e é um erro comum em connector.
func TestLoteVazioNaoEmite(t *testing.T) {
	sink := new(consumertest.MetricsSink)
	conn := newSpanCountConnector(&Config{MetricName: "spans.count"}, sink)

	require.NoError(t, conn.ConsumeTraces(context.Background(), ptrace.NewTraces()))
	assert.Empty(t, sink.AllMetrics())
}

func TestConfigValidate(t *testing.T) {
	require.ErrorIs(t, (&Config{}).Validate(), errNomeVazio)
	require.NoError(t, (&Config{MetricName: "x"}).Validate())
}
