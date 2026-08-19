// Interfaces em Go são implícitas: nenhum tipo declara "implements". Se ele
// tem os métodos, ele satisfaz a interface.
//
// Rode com: go run ./01-interfaces
package main

import (
	"context"
	"fmt"
)

// Component é uma cópia reduzida da interface que TODO componente do Collector
// implementa. Duas funções, e é só isso.
type Component interface {
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

// Interfaces menores compõem interfaces maiores. No Collector, receiver.Traces
// é exatamente isto: Component mais o método de consumo.
type TracesConsumer interface {
	Component
	ConsumeTraces(ctx context.Context, spans []string) error
}

type impressora struct {
	nome string
}

// Nenhuma linha aqui diz que impressora implementa Component. Ela só tem os
// métodos certos.
func (i *impressora) Start(context.Context) error {
	fmt.Println(i.nome, "subiu")
	return nil
}

func (i *impressora) Shutdown(context.Context) error {
	fmt.Println(i.nome, "desceu")
	return nil
}

func (i *impressora) ConsumeTraces(_ context.Context, spans []string) error {
	for _, s := range spans {
		fmt.Println(i.nome, "recebeu span", s)
	}
	return nil
}

// Asserção em tempo de compilação. Este é o idioma mais repetido do
// repositório do Collector. Se alguém apagar um método, o pacote deixa de
// compilar aqui, com mensagem clara, em vez de quebrar em outro lugar.
var _ TracesConsumer = (*impressora)(nil)

// A função depende da INTERFACE, não do tipo concreto. É isso que permite ao
// Collector orquestrar 300 componentes que ele nunca viu.
func rodar(ctx context.Context, c TracesConsumer) error {
	if err := c.Start(ctx); err != nil {
		return err
	}
	if err := c.ConsumeTraces(ctx, []string{"GET /cart", "SELECT items"}); err != nil {
		return err
	}
	return c.Shutdown(ctx)
}

func main() {
	if err := rodar(context.Background(), &impressora{nome: "print"}); err != nil {
		fmt.Println("erro:", err)
	}

	// Uma interface é um par (tipo, valor). Uma interface nil e uma interface
	// contendo um ponteiro nil são coisas DIFERENTES, e essa confusão é fonte
	// clássica de bug.
	var vazio Component
	fmt.Println("interface vazia é nil?", vazio == nil)

	var ptr *impressora
	vazio = ptr
	fmt.Println("interface com ponteiro nil é nil?", vazio == nil)
}
