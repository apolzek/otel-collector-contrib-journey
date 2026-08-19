// OTTL na prática: parsear statements de texto e executar contra um span.
// É exatamente o que o transformprocessor faz com o que você escreve no YAML.
//
// Rode com: go run ./01-executando
package main

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func main() {
	ctx := context.Background()
	settings := componenttest.NewNopTelemetrySettings()

	// O parser é criado UMA vez, com o conjunto de funções disponíveis.
	// StandardFuncs traz os editores e os converters da biblioteca padrão.
	parser, err := ottlspan.NewParser(
		ottlfuncs.StandardFuncs[*ottlspan.TransformContext](),
		settings,
	)
	if err != nil {
		panic(err)
	}

	statements := []string{
		// Escrita simples num campo do span.
		`set(span.name, "nome-normalizado")`,

		// Escrita num atributo, com valor calculado por um converter.
		`set(span.attributes["nome.maiusculo"], ConvertCase(span.name, "upper"))`,

		// where transforma o statement num condicional. Sem where, roda sempre.
		`set(span.attributes["lento"], true) where span.attributes["duracao.ms"] > 500`,

		// Editor que remove chaves por expressão regular.
		`delete_matching_keys(span.attributes, "^interno\\.")`,

		// Acesso ao Resource a partir do contexto de span.
		`set(span.attributes["servico"], resource.attributes["service.name"])`,
	}

	for _, texto := range statements {
		// O parse acontece na construção do componente, não por span. Erro de
		// sintaxe impede o Collector de subir, que é o comportamento desejado.
		st, err := parser.ParseStatement(texto)
		if err != nil {
			panic(err)
		}

		tCtx := novoContexto()
		if _, _, err := st.Execute(ctx, tCtx); err != nil {
			panic(err)
		}

		fmt.Println("statement:", texto)
		fmt.Println("  nome:", tCtx.GetSpan().Name())
		fmt.Println("  atributos:", tCtx.GetSpan().Attributes().AsRaw())
		tCtx.Close()
	}
}

func novoContexto() *ottlspan.TransformContext {
	rs := ptrace.NewResourceSpans()
	rs.Resource().Attributes().PutStr("service.name", "checkout")

	ss := rs.ScopeSpans().AppendEmpty()
	ss.Scope().SetName("instrumentacao")

	span := ss.Spans().AppendEmpty()
	span.SetName("GET /cart")
	span.Attributes().PutInt("duracao.ms", 1200)
	span.Attributes().PutStr("interno.debug", "x")
	span.Attributes().PutStr("http.method", "GET")

	return ottlspan.NewTransformContextPtr(rs, ss, span)
}
