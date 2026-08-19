package resilienteexporter

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var errBackendIndisponivel = errors.New("backend indisponível")

// backendFalso simula um destino que recusa as primeiras N tentativas.
type exporterResiliente struct {
	falhasAteAceitar int32
	tentativas       atomic.Int32
	entregues        atomic.Int32
}

func newExporter(cfg *Config) *exporterResiliente {
	return &exporterResiliente{falhasAteAceitar: int32(cfg.FalhasAteAceitar)}
}

func (e *exporterResiliente) pushTraces(_ context.Context, td ptrace.Traces) error {
	n := e.tentativas.Add(1)

	// Erro TEMPORÁRIO: o exporterhelper vai tentar de novo, respeitando o
	// backoff configurado.
	if n <= e.falhasAteAceitar {
		return fmt.Errorf("tentativa %d: %w", n, errBackendIndisponivel)
	}

	// Erro PERMANENTE: nenhum retry adianta. Marcar assim evita o Collector
	// girar para sempre com um dado que nunca vai passar.
	if td.SpanCount() == 0 {
		return consumererror.NewPermanent(errors.New("lote vazio rejeitado pelo backend"))
	}

	e.entregues.Add(int32(td.SpanCount()))
	return nil
}
