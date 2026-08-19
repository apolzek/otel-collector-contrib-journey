// pdata é o modelo de dados que todos os componentes falam. Ele é gerado a
// partir do protobuf do OTLP, então não tem campos públicos: tudo é acessor.
//
// Rode com: go run ./01-pdata
package main

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func main() {
	construirTraces()
	construirMetricas()
	construirLogs()
	demonstrarReferencia()
}

// A hierarquia se repete nos três sinais:
// Resource (de quem é) -> Scope (qual instrumentação gerou) -> o dado.
func construirTraces() {
	td := ptrace.NewTraces()

	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", "checkout")
	rs.Resource().Attributes().PutStr("deployment.environment", "prod")

	ss := rs.ScopeSpans().AppendEmpty()
	ss.Scope().SetName("github.com/exemplo/instrumentacao")
	ss.Scope().SetVersion("1.0.0")

	span := ss.Spans().AppendEmpty()
	span.SetName("POST /checkout")
	span.SetKind(ptrace.SpanKindServer)
	span.SetTraceID(pcommon.TraceID([16]byte{0x01, 0x02}))
	span.SetSpanID(pcommon.SpanID([8]byte{0x0a}))
	span.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Unix(1700000000, 0)))
	span.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Unix(1700000001, 500000000)))
	span.Status().SetCode(ptrace.StatusCodeError)
	span.Status().SetMessage("gateway timeout")
	span.Attributes().PutInt("http.status_code", 504)

	fmt.Println("--- traces ---")
	fmt.Println("spans no lote:", td.SpanCount())
	fmt.Println("duração:", span.EndTimestamp().AsTime().Sub(span.StartTimestamp().AsTime()))

	// A partir do Go 1.23 as coleções de pdata têm iteradores All(),
	// bem mais legível que o velho for i := 0; i < X.Len(); i++.
	for _, rs := range td.ResourceSpans().All() {
		for _, ss := range rs.ScopeSpans().All() {
			for _, s := range ss.Spans().All() {
				fmt.Printf("span %s status=%s\n", s.Name(), s.Status().Code())
			}
		}
	}
}

func construirMetricas() {
	md := pmetric.NewMetrics()
	sm := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()

	// Sum: contador. Precisa dizer se é monotônico e qual a temporalidade.
	req := sm.Metrics().AppendEmpty()
	req.SetName("http.server.requests")
	req.SetUnit("{request}")
	soma := req.SetEmptySum()
	soma.SetIsMonotonic(true)
	soma.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	dp := soma.DataPoints().AppendEmpty()
	dp.SetIntValue(42)
	dp.Attributes().PutStr("http.route", "/checkout")

	// Gauge: valor instantâneo, sem temporalidade.
	mem := sm.Metrics().AppendEmpty()
	mem.SetName("process.memory.usage")
	mem.SetUnit("By")
	mem.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(1024 * 1024 * 128)

	fmt.Println("--- metrics ---")
	fmt.Println("data points no lote:", md.DataPointCount())

	// O tipo da métrica é descoberto por switch, porque Sum, Gauge,
	// Histogram e os outros vivem num campo union do protobuf.
	for _, m := range sm.Metrics().All() {
		switch m.Type() {
		case pmetric.MetricTypeSum:
			fmt.Printf("%s é uma Sum com %d pontos\n", m.Name(), m.Sum().DataPoints().Len())
		case pmetric.MetricTypeGauge:
			fmt.Printf("%s é um Gauge com %d pontos\n", m.Name(), m.Gauge().DataPoints().Len())
		default:
			fmt.Printf("%s tem tipo %s\n", m.Name(), m.Type())
		}
	}
}

func construirLogs() {
	ld := plog.NewLogs()
	lr := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	lr.SetSeverityNumber(plog.SeverityNumberError)
	lr.SetSeverityText("ERROR")
	lr.Body().SetStr("conexão recusada")
	lr.Attributes().PutStr("host.name", "node-1")

	fmt.Println("--- logs ---")
	fmt.Println("registros:", ld.LogRecordCount(), "severidade:", lr.SeverityNumber())

	// pcommon.Value é uma união de tipos. Um atributo pode ser mapa ou lista.
	tags := lr.Attributes().PutEmptySlice("tags")
	tags.AppendEmpty().SetStr("urgente")
	tags.AppendEmpty().SetStr("rede")
	fmt.Println("atributos:", lr.Attributes().AsRaw())
}

// A parte que mais confunde quem começa: pdata é passado por REFERÊNCIA.
// Escrever num atributo altera o dado que outros pipelines veem. É daí que
// vem a necessidade de declarar consumer.Capabilities{MutatesData: true}.
func demonstrarReferencia() {
	fmt.Println("--- referência e cópia ---")

	original := ptrace.NewTraces()
	rs := original.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("env", "dev")

	referencia := original // não copia nada: aponta para o mesmo dado
	referencia.ResourceSpans().At(0).Resource().Attributes().PutStr("env", "prod")
	v, _ := original.ResourceSpans().At(0).Resource().Attributes().Get("env")
	fmt.Println("depois de mexer na referência, o original virou:", v.Str())

	// Para isolar de verdade é preciso copiar explicitamente.
	copia := ptrace.NewTraces()
	original.CopyTo(copia)
	copia.ResourceSpans().At(0).Resource().Attributes().PutStr("env", "staging")
	v, _ = original.ResourceSpans().At(0).Resource().Attributes().Get("env")
	fmt.Println("depois de mexer na cópia, o original continua:", v.Str())
}
