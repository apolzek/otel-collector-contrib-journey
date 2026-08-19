package tickreceiver

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestConfigValidate(t *testing.T) {
	require.NoError(t, (&Config{Interval: time.Second}).Validate())
	require.ErrorIs(t, (&Config{Interval: time.Millisecond}).Validate(), errIntervalMuitoCurto)
}

// consumertest.LogsSink é um consumer falso que guarda tudo que recebe.
// É a ferramenta padrão para testar receiver e processor.
func TestReceiverGeraLogs(t *testing.T) {
	sink := new(consumertest.LogsSink)

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Interval = 20 * time.Millisecond
	cfg.Message = "olá"

	rcv, err := factory.CreateLogs(
		context.Background(),
		receivertest.NewNopSettings(typeStr),
		cfg,
		sink,
	)
	require.NoError(t, err)

	require.NoError(t, rcv.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, rcv.Shutdown(context.Background())) })

	// Nunca use sleep fixo para esperar telemetria: o teste fica lento e
	// instável. Eventually tenta de novo até o prazo acabar.
	require.Eventually(t, func() bool {
		return sink.LogRecordCount() >= 2
	}, 2*time.Second, 10*time.Millisecond)

	primeiro := sink.AllLogs()[0]
	body := primeiro.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body()
	assert.Equal(t, "olá", body.Str())
}

// Shutdown sem Start não pode entrar em pânico: o Collector chama Shutdown em
// componentes que nem chegaram a subir quando o boot falha no meio.
func TestShutdownSemStart(t *testing.T) {
	r := newTickReceiver(&Config{Interval: time.Second}, componenttest.NewNopTelemetrySettings().Logger, consumertest.NewNop())
	require.NoError(t, r.Shutdown(context.Background()))
}
