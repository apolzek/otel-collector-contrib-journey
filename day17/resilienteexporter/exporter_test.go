package resilienteexporter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func umSpan() ptrace.Traces {
	td := ptrace.NewTraces()
	td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("span")
	return td
}

// Configuração que espera o resultado, para o teste ser determinístico.
// Em produção wait_for_result fica false: o receiver não deve ficar parado
// esperando o backend.
func cfgSincrona(falhas int) *Config {
	cfg := createDefaultConfig().(*Config)
	cfg.FalhasAteAceitar = falhas

	q := exporterhelper.NewDefaultQueueConfig()
	q.WaitForResult = true
	q.NumConsumers = 1
	cfg.QueueConfig = configoptional.Some(q)

	cfg.BackOffConfig.InitialInterval = time.Millisecond
	cfg.BackOffConfig.MaxInterval = 5 * time.Millisecond
	cfg.BackOffConfig.MaxElapsedTime = 2 * time.Second
	return cfg
}

// O retry é do helper, não do seu código: a função de push só devolve erro.
func TestRetryEntregaDepoisDeFalhar(t *testing.T) {
	cfg := cfgSincrona(3)
	e := newExporter(cfg)

	exp, err := exporterhelper.NewTraces(
		context.Background(),
		exportertest.NewNopSettings(typeStr),
		cfg,
		e.pushTraces,
		exporterhelper.WithQueue(cfg.QueueConfig),
		exporterhelper.WithRetry(cfg.BackOffConfig),
		exporterhelper.WithTimeout(cfg.TimeoutConfig),
	)
	require.NoError(t, err)
	require.NoError(t, exp.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, exp.Shutdown(context.Background())) })

	require.NoError(t, exp.ConsumeTraces(context.Background(), umSpan()))

	assert.Equal(t, int32(4), e.tentativas.Load(), "3 falhas mais a entrega")
	assert.Equal(t, int32(1), e.entregues.Load())
}

// Erro permanente não é retentado: uma tentativa e acabou.
func TestErroPermanenteNaoRetenta(t *testing.T) {
	cfg := cfgSincrona(0)
	e := newExporter(cfg)

	exp, err := exporterhelper.NewTraces(
		context.Background(),
		exportertest.NewNopSettings(typeStr),
		cfg,
		e.pushTraces,
		exporterhelper.WithQueue(cfg.QueueConfig),
		exporterhelper.WithRetry(cfg.BackOffConfig),
	)
	require.NoError(t, err)
	require.NoError(t, exp.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, exp.Shutdown(context.Background())) })

	// Lote vazio dispara o caminho permanente do exporter.
	err = exp.ConsumeTraces(context.Background(), ptrace.NewTraces())
	require.Error(t, err)
	assert.Equal(t, int32(1), e.tentativas.Load(), "permanente não pode ser retentado")
}

// Quando o backoff estoura o prazo, o dado é descartado e o erro sobe.
func TestDesistePassadoOPrazo(t *testing.T) {
	cfg := cfgSincrona(1000)
	cfg.BackOffConfig.MaxElapsedTime = 50 * time.Millisecond
	e := newExporter(cfg)

	exp, err := exporterhelper.NewTraces(
		context.Background(),
		exportertest.NewNopSettings(typeStr),
		cfg,
		e.pushTraces,
		exporterhelper.WithQueue(cfg.QueueConfig),
		exporterhelper.WithRetry(cfg.BackOffConfig),
	)
	require.NoError(t, err)
	require.NoError(t, exp.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, exp.Shutdown(context.Background())) })

	require.Error(t, exp.ConsumeTraces(context.Background(), umSpan()))
	assert.Zero(t, e.entregues.Load())
	assert.Greater(t, e.tentativas.Load(), int32(1))
}
