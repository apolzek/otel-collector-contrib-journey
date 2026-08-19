package tagprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processortest"
)

func traceCom(attrs map[string]string) ptrace.Traces {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	for k, v := range attrs {
		rs.Resource().Attributes().PutStr(k, v)
	}
	rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("span")
	return td
}

func TestProcessTraces(t *testing.T) {
	tests := []struct {
		name      string
		cfg       Config
		entrada   map[string]string
		esperados map[string]string
	}{
		{
			name:      "adiciona atributo novo",
			cfg:       Config{Attributes: map[string]string{"env": "prod"}},
			entrada:   map[string]string{"service.name": "api"},
			esperados: map[string]string{"service.name": "api", "env": "prod"},
		},
		{
			name:      "não sobrescreve por padrão",
			cfg:       Config{Attributes: map[string]string{"env": "prod"}},
			entrada:   map[string]string{"env": "dev"},
			esperados: map[string]string{"env": "dev"},
		},
		{
			name:      "sobrescreve quando overwrite é true",
			cfg:       Config{Attributes: map[string]string{"env": "prod"}, Overwrite: true},
			entrada:   map[string]string{"env": "dev"},
			esperados: map[string]string{"env": "prod"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTagProcessor(&tt.cfg)
			saida, err := p.processTraces(context.Background(), traceCom(tt.entrada))
			require.NoError(t, err)

			attrs := saida.ResourceSpans().At(0).Resource().Attributes()
			assert.Equal(t, len(tt.esperados), attrs.Len())
			for k, v := range tt.esperados {
				got, ok := attrs.Get(k)
				require.True(t, ok, "atributo %s ausente", k)
				assert.Equal(t, v, got.Str())
			}
		})
	}
}

// Teste de integração leve: monta o processor pela factory, empurra dados e
// confere o que chegou no consumer seguinte.
func TestFactoryEndToEnd(t *testing.T) {
	sink := new(consumertest.TracesSink)

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Attributes = map[string]string{"cluster": "prod-1"}

	p, err := factory.CreateTraces(
		context.Background(),
		processortest.NewNopSettings(typeStr),
		cfg,
		sink,
	)
	require.NoError(t, err)
	require.True(t, p.Capabilities().MutatesData)

	require.NoError(t, p.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, p.ConsumeTraces(context.Background(), traceCom(nil)))
	require.NoError(t, p.Shutdown(context.Background()))

	require.Len(t, sink.AllTraces(), 1)
	v, ok := sink.AllTraces()[0].ResourceSpans().At(0).Resource().Attributes().Get("cluster")
	require.True(t, ok)
	assert.Equal(t, "prod-1", v.Str())
}

func TestConfigValidate(t *testing.T) {
	require.ErrorIs(t, (&Config{}).Validate(), errSemAtributos)
	require.NoError(t, (&Config{Attributes: map[string]string{"a": "b"}}).Validate())
}
