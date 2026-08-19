// Embedding é como o Go faz reuso. Não é herança: é composição com promoção
// de métodos.
//
// Rode com: go run ./02-embedding
package main

import (
	"context"
	"fmt"
)

type Component interface {
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

// StartFunc e ShutdownFunc são tipos função com um método. O Collector tem
// exatamente estes dois tipos em component/component.go.
type StartFunc func(context.Context) error

func (f StartFunc) Start(ctx context.Context) error {
	if f == nil {
		return nil
	}
	return f(ctx)
}

type ShutdownFunc func(context.Context) error

func (f ShutdownFunc) Shutdown(ctx context.Context) error {
	if f == nil {
		return nil
	}
	return f(ctx)
}

// Embutindo os dois tipos, este componente ganha Start e Shutdown prontos, com
// o comportamento correto para o valor zero: não fazer nada e devolver nil.
// É por isso que muitos componentes do Collector não escrevem Start nem
// Shutdown, mas ainda assim satisfazem component.Component.
type contador struct {
	StartFunc
	ShutdownFunc

	total int
}

var _ Component = (*contador)(nil)

// Embedding de INTERFACE dentro de struct é outro truque: o tipo satisfaz a
// interface imediatamente, e você sobrescreve só os métodos que interessam.
// Usado bastante em testes e em wrappers.
type componenteQueFalhaNoStart struct {
	Component // valor nil, os métodos não sobrescritos entram em pânico se chamados
}

func (componenteQueFalhaNoStart) Start(context.Context) error {
	return fmt.Errorf("falhei de propósito")
}

func main() {
	c := &contador{}
	fmt.Println("start:", c.Start(context.Background()))
	fmt.Println("shutdown:", c.Shutdown(context.Background()))

	// Agora com comportamento de verdade, sem declarar métodos novos.
	c2 := &contador{
		StartFunc: func(context.Context) error {
			fmt.Println("subindo de verdade")
			return nil
		},
	}
	fmt.Println("start:", c2.Start(context.Background()))

	var falho Component = componenteQueFalhaNoStart{}
	fmt.Println("start:", falho.Start(context.Background()))
}
