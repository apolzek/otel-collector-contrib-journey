// Functional options: como o Collector permite que uma função de construção
// aceite dez configurações opcionais sem ter dez parâmetros nem dez
// construtores diferentes.
//
// Rode com: go run ./02-options
package main

import (
	"fmt"
	"time"
)

type servidor struct {
	endereco string
	timeout  time.Duration
	retries  int
	tls      bool
}

// A Option é uma FUNÇÃO que muta o objeto em construção. É só isso.
type Option func(*servidor)

func WithTimeout(d time.Duration) Option {
	return func(s *servidor) { s.timeout = d }
}

func WithRetries(n int) Option {
	return func(s *servidor) { s.retries = n }
}

func WithTLS() Option {
	return func(s *servidor) { s.tls = true }
}

// O construtor define os padrões e depois deixa cada opção sobrescrever o que
// quiser. Chamar sem opção nenhuma tem que produzir algo válido.
func New(endereco string, opts ...Option) *servidor {
	s := &servidor{
		endereco: endereco,
		timeout:  5 * time.Second,
		retries:  3,
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Variação usada no Collector: a Option é uma INTERFACE, não um tipo função.
// Isso permite adicionar métodos e manter o pacote extensível sem quebrar
// quem já usa.
type OptionIface interface {
	aplicar(*servidor)
}

type optionFunc func(*servidor)

func (f optionFunc) aplicar(s *servidor) { f(s) }

func WithEndereco(e string) OptionIface {
	return optionFunc(func(s *servidor) { s.endereco = e })
}

func main() {
	fmt.Printf("%+v\n", New("localhost:4317"))
	fmt.Printf("%+v\n", New("localhost:4317", WithTimeout(30*time.Second), WithTLS()))
	fmt.Printf("%+v\n", New("localhost:4317", WithRetries(0)))

	s := New("a:1")
	WithEndereco("b:2").aplicar(s)
	fmt.Printf("%+v\n", s)
}
