// Concorrência: goroutine, WaitGroup, Mutex e atomic. Um receiver que faz
// scrape de vários alvos usa exatamente este arranjo.
//
// Rode com: go run -race ./03-goroutines
package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type coletor struct {
	// Mutex protege estruturas compostas. A regra: o campo protegido vem
	// logo abaixo do mutex que o protege.
	mu        sync.Mutex
	resultado map[string]int

	// atomic serve para contadores simples, sem precisar de lock.
	erros atomic.Int64
}

func (c *coletor) coletar(alvo string) {
	time.Sleep(10 * time.Millisecond) // simula uma chamada de rede

	if alvo == "quebrado" {
		c.erros.Add(1)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.resultado[alvo] = len(alvo)
}

func main() {
	c := &coletor{resultado: map[string]int{}}
	alvos := []string{"api", "banco", "cache", "quebrado", "fila"}

	// WaitGroup conta goroutines em voo. Add ANTES de disparar, Done com
	// defer dentro. Add depois do go é uma corrida clássica.
	var wg sync.WaitGroup
	for _, alvo := range alvos {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.coletar(alvo)
		}()
	}
	wg.Wait()

	fmt.Println("resultado:", c.resultado)
	fmt.Println("erros:", c.erros.Load())

	// Canais: o outro jeito de coordenar. Um worker pool com fan-in.
	trabalhos := make(chan int, len(alvos))
	saidas := make(chan string, len(alvos))

	var wg2 sync.WaitGroup
	for w := 1; w <= 3; w++ {
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			for t := range trabalhos {
				saidas <- fmt.Sprintf("worker %d processou %d", w, t)
			}
		}()
	}

	for i := 1; i <= 5; i++ {
		trabalhos <- i
	}
	// Fechar o canal é o sinal de "não vem mais nada". Sem isso o range dos
	// workers nunca termina e o wg2.Wait trava.
	close(trabalhos)

	wg2.Wait()
	close(saidas)

	total := 0
	for range saidas {
		total++
	}
	fmt.Println("mensagens processadas:", total)
}
