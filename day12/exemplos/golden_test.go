package exemplos

import (
	"context"
	"flag"
	"path/filepath"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/ptracetest"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Flag para regerar os arquivos esperados. O repositório usa exatamente esta
// convenção: go test -update-golden regrava o testdata a partir da saída atual.
// Sempre revise o diff antes de commitar um golden regerado.
var atualizarGolden = flag.Bool("update-golden", false, "regrava os arquivos golden")

// Golden file: em vez de escrever dezenas de asserts, você compara a saída
// inteira com um arquivo YAML versionado. Vale a pena quando a saída é grande,
// como um lote de métricas.
func TestGoldenTraces(t *testing.T) {
	entrada, err := golden.ReadTraces(filepath.Join("testdata", "traces-entrada.yaml"))
	require.NoError(t, err)

	saida, err := NewSanitizer(&Config{
		RemoveKeys:  []string{"user.email"},
		Placeholder: "[REDACTED]",
	}).ProcessTraces(context.Background(), entrada)
	require.NoError(t, err)

	esperadoPath := filepath.Join("testdata", "traces-esperado.yaml")
	if *atualizarGolden {
		require.NoError(t, golden.WriteTraces(t, esperadoPath, saida))
	}

	esperado, err := golden.ReadTraces(esperadoPath)
	require.NoError(t, err)

	// CompareTraces devolve um erro que descreve exatamente qual campo
	// divergiu, bem melhor do que um assert.Equal em cima da struct.
	// As opções Ignore existem para campos que mudam a cada execução:
	// timestamp, id gerado aleatoriamente, ordem dos elementos.
	require.NoError(t, ptracetest.CompareTraces(esperado, saida,
		ptracetest.IgnoreStartTimestamp(),
		ptracetest.IgnoreEndTimestamp(),
	))
}

// Sem golden, o mesmo teste precisaria de um assert por campo. Este é o
// contraste que justifica a técnica.
func TestSemGolden(t *testing.T) {
	td := ptrace.NewTraces()
	span := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.Attributes().PutStr("user.email", "a@b.com")

	saida, err := NewSanitizer(&Config{RemoveKeys: []string{"user.email"}}).ProcessTraces(context.Background(), td)
	require.NoError(t, err)
	require.Equal(t, 0, saida.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Attributes().Len())
}
