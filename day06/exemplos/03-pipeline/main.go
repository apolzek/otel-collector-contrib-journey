// Um pipeline do Collector é só uma cadeia de consumer.Traces ligada na mão
// pelo service. Este programa monta essa cadeia sem o Collector, para deixar
// visível o que acontece por baixo.
//
// Rode com: go run ./03-pipeline
package main

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// ---------------------------------------------------------------------------
// Exporter: a ponta final. Só consome.
// ---------------------------------------------------------------------------

type exporterFalso struct {
	component.StartFunc
	component.ShutdownFunc
	recebidos int
}

func (e *exporterFalso) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (e *exporterFalso) ConsumeTraces(_ context.Context, td ptrace.Traces) error {
	e.recebidos += td.SpanCount()
	for _, rs := range td.ResourceSpans().All() {
		env, _ := rs.Resource().Attributes().Get("env")
		for _, ss := range rs.ScopeSpans().All() {
			for _, s := range ss.Spans().All() {
				fmt.Printf("  exporter: span=%q env=%q\n", s.Name(), env.Str())
			}
		}
	}
	return nil
}

var _ consumer.Traces = (*exporterFalso)(nil)

// ---------------------------------------------------------------------------
// Processor: consome e produz. Fica no meio, então guarda o próximo consumer.
// ---------------------------------------------------------------------------

type processorFalso struct {
	component.StartFunc
	component.ShutdownFunc
	next consumer.Traces
}

// Este processor ESCREVE nos dados, então precisa declarar isso. É essa
// declaração que faz o Collector clonar a telemetria quando o mesmo receiver
// alimenta dois pipelines.
func (p *processorFalso) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (p *processorFalso) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	for _, rs := range td.ResourceSpans().All() {
		rs.Resource().Attributes().PutStr("env", "prod")
	}
	fmt.Println("  processor: marcou env=prod")
	return p.next.ConsumeTraces(ctx, td)
}

var _ consumer.Traces = (*processorFalso)(nil)

// ---------------------------------------------------------------------------
// Receiver: a ponta inicial. Só produz, empurrando no próximo consumer.
// ---------------------------------------------------------------------------

type receiverFalso struct {
	component.ShutdownFunc
	next consumer.Traces
}

func (r *receiverFalso) Start(ctx context.Context, _ component.Host) error {
	td := ptrace.NewTraces()
	ss := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty()
	ss.Spans().AppendEmpty().SetName("GET /cart")
	ss.Spans().AppendEmpty().SetName("SELECT items")

	fmt.Println("  receiver: gerou 2 spans")
	return r.next.ConsumeTraces(ctx, td)
}

func main() {
	ctx := context.Background()

	// O service monta a cadeia DE TRÁS PARA A FRENTE: o exporter existe
	// antes do processor, que existe antes do receiver. Assim nenhum
	// componente recebe dados sem ter para onde mandar.
	exp := &exporterFalso{}
	proc := &processorFalso{next: exp}
	rcv := &receiverFalso{next: proc}

	// Start também é na ordem inversa do fluxo, pelo mesmo motivo.
	fmt.Println("start:")
	componentes := []component.Component{exp, proc, rcv}
	for _, c := range componentes {
		if err := c.Start(ctx, nil); err != nil {
			panic(err)
		}
	}

	// Shutdown é na ordem direta, drenando o pipeline: primeiro para de
	// entrar dado novo, depois esvazia o que está no meio.
	fmt.Println("shutdown:")
	for i := len(componentes) - 1; i >= 0; i-- {
		if err := componentes[i].Shutdown(ctx); err != nil {
			panic(err)
		}
	}

	fmt.Println("spans entregues:", exp.recebidos)

	// Fan-out: um receiver alimentando dois pipelines. O core tem um helper
	// pronto para isso, que é onde o MutatesData é levado em conta.
	fmt.Println("--- fan-out ---")
	a, b := &exporterFalso{}, &exporterFalso{}
	fan := fanout{consumers: []consumer.Traces{a, b}}
	td := ptrace.NewTraces()
	td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("span")
	if err := fan.ConsumeTraces(ctx, td); err != nil {
		panic(err)
	}
	fmt.Println("a recebeu:", a.recebidos, "b recebeu:", b.recebidos)
}

type fanout struct {
	consumers []consumer.Traces
}

func (f fanout) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (f fanout) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	for _, c := range f.consumers {
		// O fanout de verdade do Collector clona aqui quando o consumidor
		// declara MutatesData, para um pipeline não corromper o outro.
		if c.Capabilities().MutatesData {
			clone := ptrace.NewTraces()
			td.CopyTo(clone)
			if err := c.ConsumeTraces(ctx, clone); err != nil {
				return err
			}
			continue
		}
		if err := c.ConsumeTraces(ctx, td); err != nil {
			return err
		}
	}
	return nil
}
