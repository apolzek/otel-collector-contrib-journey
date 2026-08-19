// Escrevendo uma função OTTL própria. É assim que se adiciona vocabulário novo
// ao transformprocessor e ao filterprocessor.
//
// Rode com: go run ./02-funcao-customizada
package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// ---------------------------------------------------------------------------
// Passo 1: a struct de argumentos.
//
// Os NOMES dos campos viram os nomes dos parâmetros nomeados no OTTL, e a
// ORDEM vira a ordem posicional. Os TIPOS dizem ao parser o que aceitar:
// PMapGetSetter para um mapa que será lido e escrito, StringGetter para
// qualquer expressão que resulte em string.
// ---------------------------------------------------------------------------

type MascararArguments[K any] struct {
	Target  ottl.PMapGetSetter[K]
	Key     ottl.StringGetter[K]
	Visivel ottl.Optional[int64]
}

// ---------------------------------------------------------------------------
// Passo 2: a factory. O nome aqui é o nome usado no YAML.
// ---------------------------------------------------------------------------

func NewMascararFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("mascarar", &MascararArguments[K]{}, criarMascarar[K])
}

func criarMascarar[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*MascararArguments[K])
	if !ok {
		return nil, errors.New("mascarar: argumentos precisam ser *MascararArguments[K]")
	}
	return mascarar(args.Target, args.Key, args.Visivel), nil
}

// ---------------------------------------------------------------------------
// Passo 3: a implementação. Ela devolve uma ExprFunc, ou seja, uma closure que
// roda uma vez por span. Tudo que puder ser feito na construção deve ficar
// fora dessa closure.
// ---------------------------------------------------------------------------

func mascarar[K any](target ottl.PMapGetSetter[K], key ottl.StringGetter[K], visivel ottl.Optional[int64]) ottl.ExprFunc[K] {
	// Optional permite parâmetro opcional. O default é resolvido aqui, uma vez
	// só, e não a cada execução.
	n := 4
	if !visivel.IsEmpty() {
		n = int(visivel.Get())
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		attrs, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		nome, err := key.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		v, ok := attrs.Get(nome)
		if !ok {
			// Chave ausente não é erro: o statement simplesmente não faz nada.
			return nil, nil
		}

		s := v.AsString()
		if len(s) <= n {
			attrs.PutStr(nome, strings.Repeat("*", len(s)))
			return nil, target.Set(ctx, tCtx, attrs)
		}
		attrs.PutStr(nome, strings.Repeat("*", len(s)-n)+s[len(s)-n:])
		return nil, target.Set(ctx, tCtx, attrs)
	}
}

func main() {
	ctx := context.Background()

	// Passo 4: registrar a função no mapa entregue ao parser.
	funcs := ottlfuncs.StandardFuncs[*ottlspan.TransformContext]()
	f := NewMascararFactory[*ottlspan.TransformContext]()
	funcs[f.Name()] = f

	parser, err := ottlspan.NewParser(funcs, componenttest.NewNopTelemetrySettings())
	if err != nil {
		panic(err)
	}

	statements := []string{
		`mascarar(span.attributes, "cartao")`,
		`mascarar(span.attributes, "cartao", 8)`,
		`mascarar(span.attributes, "nao.existe")`,
		// Argumento nomeado, com =, permite pular opcionais anteriores.
		`mascarar(span.attributes, "cartao", visivel=0)`,
	}

	for _, texto := range statements {
		st, err := parser.ParseStatement(texto)
		if err != nil {
			panic(err)
		}
		tCtx := novoContexto()
		if _, _, err := st.Execute(ctx, tCtx); err != nil {
			panic(err)
		}
		v, _ := tCtx.GetSpan().Attributes().Get("cartao")
		fmt.Printf("%-55s -> %s\n", texto, v.Str())
		tCtx.Close()
	}
}

func novoContexto() *ottlspan.TransformContext {
	rs := ptrace.NewResourceSpans()
	ss := rs.ScopeSpans().AppendEmpty()
	span := ss.Spans().AppendEmpty()
	span.Attributes().PutStr("cartao", "4111111111111111")
	return ottlspan.NewTransformContextPtr(rs, ss, span)
}
