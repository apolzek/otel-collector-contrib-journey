package spancountconnector

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// spanCountConnector é exporter de um lado e receiver do outro: recebe
// ptrace.Traces pelo ConsumeTraces e empurra pmetric.Metrics no consumer do
// pipeline de destino.
type spanCountConnector struct {
	// Embutir StartFunc e ShutdownFunc dá as implementações vazias de
	// component.Component de graça. É um idioma comum quando o componente
	// não tem nada para fazer no boot.
	component.StartFunc
	component.ShutdownFunc

	cfg  *Config
	next consumer.Metrics
}

func newSpanCountConnector(cfg *Config, next consumer.Metrics) *spanCountConnector {
	return &spanCountConnector{cfg: cfg, next: next}
}

// Capabilities responde ao pipeline de traces: este connector só lê os spans.
func (c *spanCountConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (c *spanCountConnector) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	contagem := map[string]int64{}

	for _, rs := range td.ResourceSpans().All() {
		chave := ""
		if c.cfg.Dimension != "" {
			if v, ok := rs.Resource().Attributes().Get(c.cfg.Dimension); ok {
				chave = v.AsString()
			}
		}
		for _, ss := range rs.ScopeSpans().All() {
			contagem[chave] += int64(ss.Spans().Len())
		}
	}

	if len(contagem) == 0 {
		return nil
	}
	return c.next.ConsumeMetrics(ctx, c.montarMetricas(contagem))
}

func (c *spanCountConnector) montarMetricas(contagem map[string]int64) pmetric.Metrics {
	md := pmetric.NewMetrics()
	sm := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("github.com/apolzek/spancountconnector")

	m := sm.Metrics().AppendEmpty()
	m.SetName(c.cfg.MetricName)
	m.SetDescription("Número de spans observados")
	m.SetUnit("{span}")

	soma := m.SetEmptySum()
	soma.SetIsMonotonic(true)
	soma.SetAggregationTemporality(pmetric.AggregationTemporalityDelta)

	agora := pcommon.NewTimestampFromTime(time.Now())
	for chave, total := range contagem {
		dp := soma.DataPoints().AppendEmpty()
		dp.SetTimestamp(agora)
		dp.SetIntValue(total)
		if c.cfg.Dimension != "" && chave != "" {
			dp.Attributes().PutStr(c.cfg.Dimension, chave)
		}
	}
	return md
}
