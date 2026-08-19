// Generics. No Collector aparecem principalmente no OTTL, onde toda a API é
// parametrizada pelo contexto de transformação, e nos helpers de fila.
//
// Rode com: go run ./05-generics
package main

import (
	"context"
	"fmt"
	"strings"
)

// Getter[K] lê um valor de um contexto de tipo K. É a forma reduzida do
// ottl.Getter, a peça central do OTTL.
type Getter[K any] interface {
	Get(ctx context.Context, tCtx K) (any, error)
}

// Setter[K] escreve.
type Setter[K any] interface {
	Set(ctx context.Context, tCtx K, val any) error
}

type GetSetter[K any] interface {
	Getter[K]
	Setter[K]
}

// StandardGetSetter adapta duas funções comuns para a interface. O OTTL tem
// exatamente este tipo, com este nome.
type StandardGetSetter[K any] struct {
	Getter func(ctx context.Context, tCtx K) (any, error)
	Setter func(ctx context.Context, tCtx K, val any) error
}

func (s StandardGetSetter[K]) Get(ctx context.Context, tCtx K) (any, error) {
	return s.Getter(ctx, tCtx)
}

func (s StandardGetSetter[K]) Set(ctx context.Context, tCtx K, val any) error {
	return s.Setter(ctx, tCtx, val)
}

// Uma função que funciona para QUALQUER contexto de transformação, sem
// conhecer o tipo concreto.
func UpperCase[K any](gs GetSetter[K]) func(context.Context, K) error {
	return func(ctx context.Context, tCtx K) error {
		v, err := gs.Get(ctx, tCtx)
		if err != nil {
			return err
		}
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("esperava string, veio %T", v)
		}
		return gs.Set(ctx, tCtx, strings.ToUpper(s))
	}
}

// Dois contextos diferentes, a mesma função UpperCase serve os dois.
type spanCtx struct{ nome string }
type logCtx struct{ corpo string }

// Restrições de tipo. A cláusula ~ aceita tipos DEFINIDOS a partir do tipo
// base, não só o tipo base em si.
type Numero interface {
	~int | ~int64 | ~float64
}

func Somar[T Numero](vs []T) T {
	var total T
	for _, v := range vs {
		total += v
	}
	return total
}

type Bytes int64 // tipo definido a partir de int, aceito por causa do ~

func main() {
	ctx := context.Background()

	span := &spanCtx{nome: "get /cart"}
	gsSpan := StandardGetSetter[*spanCtx]{
		Getter: func(_ context.Context, s *spanCtx) (any, error) { return s.nome, nil },
		Setter: func(_ context.Context, s *spanCtx, v any) error { s.nome = v.(string); return nil },
	}
	if err := UpperCase[*spanCtx](gsSpan)(ctx, span); err != nil {
		panic(err)
	}
	fmt.Println("span:", span.nome)

	log := &logCtx{corpo: "erro de conexão"}
	gsLog := StandardGetSetter[*logCtx]{
		Getter: func(_ context.Context, l *logCtx) (any, error) { return l.corpo, nil },
		Setter: func(_ context.Context, l *logCtx, v any) error { l.corpo = v.(string); return nil },
	}
	// O tipo é inferido pelo argumento: não precisa escrever [*logCtx].
	if err := UpperCase(gsLog)(ctx, log); err != nil {
		panic(err)
	}
	fmt.Println("log:", log.corpo)

	fmt.Println("soma int:", Somar([]int{1, 2, 3}))
	fmt.Println("soma float:", Somar([]float64{1.5, 2.5}))
	fmt.Println("soma tipo definido:", Somar([]Bytes{1024, 2048}))
}
