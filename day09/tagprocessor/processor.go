package tagprocessor

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// tagProcessor guarda só a configuração: um processor sem estado é o caso mais
// comum e o mais fácil de manter.
type tagProcessor struct {
	cfg *Config
}

func newTagProcessor(cfg *Config) *tagProcessor {
	return &tagProcessor{cfg: cfg}
}

// processTraces tem a assinatura processorhelper.ProcessTracesFunc: recebe os
// dados, devolve os dados. Devolver ErrSkipProcessingData descarta o lote sem
// gerar erro no pipeline.
func (p *tagProcessor) processTraces(_ context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	for _, rs := range td.ResourceSpans().All() {
		p.aplicar(rs.Resource().Attributes())
	}
	return td, nil
}

func (p *tagProcessor) processLogs(_ context.Context, ld plog.Logs) (plog.Logs, error) {
	for _, rl := range ld.ResourceLogs().All() {
		p.aplicar(rl.Resource().Attributes())
	}
	return ld, nil
}

// aplicar escreve nos atributos. Como isto MUTA os dados, a factory precisa
// declarar MutatesData: true. Sem isso, dois pipelines que compartilham o
// mesmo receiver enxergariam a alteração um do outro.
func (p *tagProcessor) aplicar(attrs pcommon.Map) {
	for k, v := range p.cfg.Attributes {
		if !p.cfg.Overwrite {
			if _, existe := attrs.Get(k); existe {
				continue
			}
		}
		attrs.PutStr(k, v)
	}
}
