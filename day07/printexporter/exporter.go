package printexporter

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

// printExporter é o estado do componente. Repare que ele NÃO implementa
// exporter.Traces na mão: quem faz isso é o exporterhelper, que recebe as
// funções pushTraces e pushLogs abaixo.
type printExporter struct {
	cfg    *Config
	logger *zap.Logger

	// O Collector pode chamar pushTraces de várias goroutines ao mesmo tempo.
	// Toda escrita no arquivo passa por este mutex.
	mu     sync.Mutex
	out    io.Writer
	closer io.Closer
}

func newPrintExporter(cfg *Config, logger *zap.Logger) *printExporter {
	return &printExporter{cfg: cfg, logger: logger}
}

// start abre o destino. Ele roda uma vez, no boot do Collector, e não pode
// bloquear: se precisasse de um loop, a regra seria subir uma goroutine e
// retornar.
func (e *printExporter) start(_ context.Context, _ component.Host) error {
	if e.cfg.Path == "stdout" {
		e.out = os.Stdout
		return nil
	}
	f, err := os.OpenFile(e.cfg.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("abrindo %q: %w", e.cfg.Path, err)
	}
	e.out = f
	e.closer = f
	return nil
}

// shutdown precisa ser idempotente: pode ser chamado mesmo se start falhou.
func (e *printExporter) shutdown(context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closer == nil {
		return nil
	}
	err := e.closer.Close()
	e.closer = nil
	return err
}

// pushTraces é o coração do exporter de traces. A assinatura é exatamente
// consumer.ConsumeTracesFunc. Devolver erro faz o exporterhelper aplicar
// retry e fila, conforme a configuração.
func (e *printExporter) pushTraces(_ context.Context, td ptrace.Traces) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, rs := range td.ResourceSpans().All() {
		service, _ := rs.Resource().Attributes().Get("service.name")
		for _, ss := range rs.ScopeSpans().All() {
			for _, span := range ss.Spans().All() {
				_, err := fmt.Fprintf(e.out, "%sspan service=%s name=%s trace_id=%s duration=%s\n",
					e.cfg.Prefix,
					service.Str(),
					span.Name(),
					span.TraceID(),
					span.EndTimestamp().AsTime().Sub(span.StartTimestamp().AsTime()),
				)
				if err != nil {
					return fmt.Errorf("escrevendo span: %w", err)
				}
			}
		}
	}
	return nil
}

func (e *printExporter) pushLogs(_ context.Context, ld plog.Logs) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, rl := range ld.ResourceLogs().All() {
		for _, sl := range rl.ScopeLogs().All() {
			for _, lr := range sl.LogRecords().All() {
				_, err := fmt.Fprintf(e.out, "%slog severity=%s body=%s\n",
					e.cfg.Prefix, lr.SeverityText(), lr.Body().AsString())
				if err != nil {
					return fmt.Errorf("escrevendo log: %w", err)
				}
			}
		}
	}
	return nil
}
