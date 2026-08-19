package printexporter

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

// Constrói um ptrace.Traces na mão. Repare que pdata não tem campos públicos:
// tudo é acessor.
func traceComUmSpan() ptrace.Traces {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", "checkout")

	span := rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetName("GET /cart")
	span.SetTraceID(pcommon.TraceID([16]byte{1}))
	span.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, 0)))
	span.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, int64(2*time.Second))))
	return td
}

func TestPushTracesEscreveNoArquivo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "saida.txt")
	e := newPrintExporter(&Config{Path: path, Prefix: "otel "}, zap.NewNop())

	require.NoError(t, e.start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, e.pushTraces(context.Background(), traceComUmSpan()))
	require.NoError(t, e.shutdown(context.Background()))

	conteudo, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(conteudo), "otel span service=checkout name=GET /cart")
	assert.Contains(t, string(conteudo), "duration=2s")
}

func TestStartFalhaComCaminhoInvalido(t *testing.T) {
	e := newPrintExporter(&Config{Path: "/diretorio/que/nao/existe/x.txt"}, zap.NewNop())
	assert.Error(t, e.start(context.Background(), componenttest.NewNopHost()))
}

// Shutdown precisa ser idempotente e seguro mesmo sem start.
func TestShutdownSemStart(t *testing.T) {
	e := newPrintExporter(&Config{Path: "stdout"}, zap.NewNop())
	require.NoError(t, e.shutdown(context.Background()))
	require.NoError(t, e.shutdown(context.Background()))
}
