// context.Context aparece como primeiro parâmetro em praticamente toda função
// do Collector. Ele carrega cancelamento, prazo e valores de escopo.
//
// Rode com: go run ./04-context
package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Padrão de loop de fundo: o único jeito correto de parar uma goroutine em Go
// é pedir para ela parar. Não existe kill de goroutine.
func loop(ctx context.Context, nome string, pronto chan<- struct{}) {
	defer close(pronto)
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("%s parou: %v\n", nome, ctx.Err())
			return
		case <-ticker.C:
			fmt.Println(nome, "tick")
		}
	}
}

func chamadaLenta(ctx context.Context) error {
	select {
	case <-time.After(500 * time.Millisecond):
		return nil
	case <-ctx.Done():
		// Devolver ctx.Err() embrulhado preserva a causa para quem chamou.
		return fmt.Errorf("chamada abortada: %w", ctx.Err())
	}
}

func main() {
	// Cancelamento manual: o que um Shutdown de componente faz.
	ctx, cancel := context.WithCancel(context.Background())
	pronto := make(chan struct{})
	go loop(ctx, "receiver", pronto)

	time.Sleep(70 * time.Millisecond)
	cancel()
	<-pronto // esperar a goroutine terminar de fato, senão ela vaza

	// Prazo: o que o exporterhelper aplica em cada tentativa de envio.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel2()

	err := chamadaLenta(ctx2)
	fmt.Println("erro:", err)
	fmt.Println("foi timeout?", errors.Is(err, context.DeadlineExceeded))

	// Regra prática: sempre chame cancel, mesmo quando o contexto já expirou.
	// O defer acima é obrigatório, não opcional: sem ele o timer vaza.
}
