package main

import (
	"context"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
)

// Testar uma função OTTL é testar o statement de ponta a ponta: parse mais
// execução. É o que o contrib faz nos testes de pkg/ottl.
func TestMascarar(t *testing.T) {
	funcs := ottlfuncs.StandardFuncs[*ottlspan.TransformContext]()
	f := NewMascararFactory[*ottlspan.TransformContext]()
	funcs[f.Name()] = f

	parser, err := ottlspan.NewParser(funcs, componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)

	tests := []struct {
		statement string
		esperado  string
	}{
		{`mascarar(span.attributes, "cartao")`, "************1111"},
		{`mascarar(span.attributes, "cartao", 8)`, "********11111111"},
		{`mascarar(span.attributes, "cartao", visivel=0)`, "****************"},
		{`mascarar(span.attributes, "nao.existe")`, "4111111111111111"},
		{`mascarar(span.attributes, "cartao") where span.attributes["cartao"] != nil`, "************1111"},
	}

	for _, tt := range tests {
		t.Run(tt.statement, func(t *testing.T) {
			st, err := parser.ParseStatement(tt.statement)
			require.NoError(t, err)

			tCtx := novoContexto()
			defer tCtx.Close()

			_, _, err = st.Execute(context.Background(), tCtx)
			require.NoError(t, err)

			v, ok := tCtx.GetSpan().Attributes().Get("cartao")
			require.True(t, ok)
			assert.Equal(t, tt.esperado, v.Str())
		})
	}
}

// Erro de sintaxe precisa aparecer no parse, não na execução. É isso que faz o
// Collector recusar subir com um statement inválido no YAML.
func TestSintaxeInvalida(t *testing.T) {
	funcs := ottlfuncs.StandardFuncs[*ottlspan.TransformContext]()
	f := NewMascararFactory[*ottlspan.TransformContext]()
	funcs[f.Name()] = f

	parser, err := ottlspan.NewParser(funcs, componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)

	_, err = parser.ParseStatement(`mascarar(span.attributes)`)
	require.Error(t, err)

	_, err = parser.ParseStatement(`funcao_que_nao_existe(span.attributes, "x")`)
	require.Error(t, err)
}
