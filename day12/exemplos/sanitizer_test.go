package exemplos

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func spanCom(attrs map[string]string) ptrace.Traces {
	td := ptrace.NewTraces()
	span := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetName("POST /login")
	for k, v := range attrs {
		span.Attributes().PutStr(k, v)
	}
	return td
}

// O formato padrão do repositório: slice de casos, subteste por caso.
// Vantagem prática: quando um caso quebra, a saída do go test já diz qual.
func TestProcessTraces(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		entrada  map[string]string
		esperado map[string]string
	}{
		{
			name:     "remove chave",
			cfg:      Config{RemoveKeys: []string{"user.email"}},
			entrada:  map[string]string{"user.email": "a@b.com", "http.method": "POST"},
			esperado: map[string]string{"http.method": "POST"},
		},
		{
			name:     "substitui por placeholder",
			cfg:      Config{RemoveKeys: []string{"user.email"}, Placeholder: "[REDACTED]"},
			entrada:  map[string]string{"user.email": "a@b.com"},
			esperado: map[string]string{"user.email": "[REDACTED]"},
		},
		{
			name:     "chave ausente não faz nada",
			cfg:      Config{RemoveKeys: []string{"nao.existe"}, Placeholder: "[REDACTED]"},
			entrada:  map[string]string{"http.method": "POST"},
			esperado: map[string]string{"http.method": "POST"},
		},
		{
			name:     "sem atributos",
			cfg:      Config{RemoveKeys: []string{"user.email"}},
			entrada:  nil,
			esperado: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saida, err := NewSanitizer(&tt.cfg).ProcessTraces(context.Background(), spanCom(tt.entrada))
			require.NoError(t, err)

			attrs := saida.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Attributes()
			assert.Equal(t, len(tt.esperado), attrs.Len())
			for k, v := range tt.esperado {
				got, ok := attrs.Get(k)
				require.True(t, ok, "faltou o atributo %s", k)
				assert.Equal(t, v, got.Str())
			}
		})
	}
}

// Benchmark: obrigatório em caminho quente, porque o processor roda uma vez por
// span. Rode com go test -bench=. -benchmem
func BenchmarkProcessTraces(b *testing.B) {
	s := NewSanitizer(&Config{RemoveKeys: []string{"user.email"}})

	td := ptrace.NewTraces()
	ss := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty()
	for i := range 100 {
		span := ss.Spans().AppendEmpty()
		span.Attributes().PutStr("user.email", "a@b.com")
		span.Attributes().PutStr("http.route", fmt.Sprintf("/r/%d", i))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		// Uma cópia por iteração, senão a primeira passada já limpa tudo e
		// as seguintes medem trabalho que não existe.
		clone := ptrace.NewTraces()
		td.CopyTo(clone)
		if _, err := s.ProcessTraces(context.Background(), clone); err != nil {
			b.Fatal(err)
		}
	}
}
