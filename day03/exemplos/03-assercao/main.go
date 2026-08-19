// Asserção de tipo e type switch. No Collector isso vira "descoberta de
// capacidade": pergunta-se a um valor se ele também sabe fazer outra coisa.
//
// Rode com: go run ./03-assercao
package main

import "fmt"

type Component interface {
	Start() error
}

// Uma interface opcional. Nem todo componente sabe recarregar config, mas quem
// sabe é descoberto assim.
type Recarregavel interface {
	Recarregar() error
}

type simples struct{}

func (simples) Start() error { return nil }

type completo struct{}

func (completo) Start() error      { return nil }
func (completo) Recarregar() error { fmt.Println("config recarregada"); return nil }

// A forma com dois valores nunca entra em pânico. A forma com um valor só
// entra, e por isso quase nunca aparece em código de produção.
func tentarRecarregar(c Component) {
	r, ok := c.(Recarregavel)
	if !ok {
		fmt.Printf("%T não sabe recarregar, ignorando\n", c)
		return
	}
	_ = r.Recarregar()
}

// Type switch: o jeito de lidar com valores dinâmicos. O pdata e o OTTL vivem
// disso, porque atributos de telemetria são any.
func descrever(v any) string {
	switch t := v.(type) {
	case nil:
		return "nulo"
	case string:
		return "string de " + fmt.Sprint(len(t)) + " caracteres"
	case int, int64:
		return fmt.Sprintf("inteiro %v", t)
	case bool:
		return fmt.Sprintf("booleano %v", t)
	case []any:
		return fmt.Sprintf("lista com %d itens", len(t))
	case map[string]any:
		return fmt.Sprintf("mapa com %d chaves", len(t))
	default:
		return fmt.Sprintf("tipo não tratado: %T", t)
	}
}

func main() {
	tentarRecarregar(simples{})
	tentarRecarregar(completo{})

	for _, v := range []any{nil, "otel", 42, true, []any{1, 2}, map[string]any{"a": 1}, 3.14} {
		fmt.Println(descrever(v))
	}
}
