// Erros em Go são valores. O Collector tem regras estritas sobre isso, e o
// linter reforça: nada de pkg/errors, nada de comparar mensagem com string.
//
// Rode com: go run ./04-erros
package main

import (
	"errors"
	"fmt"
)

// Erro sentinela: declarado uma vez, no topo do arquivo, comparado com
// errors.Is. Este é o padrão para "condição esperada e nomeada".
var errClienteNaoIniciado = errors.New("cliente não iniciado")

// Erro tipado: quando quem trata precisa dos DADOS do erro, não só da
// identidade dele.
type ErrConfigInvalida struct {
	Campo string
	Valor any
}

func (e *ErrConfigInvalida) Error() string {
	return fmt.Sprintf("campo %s inválido: %v", e.Campo, e.Valor)
}

func enviar(iniciado bool) error {
	if !iniciado {
		// %w EMBRULHA o erro: a identidade original sobrevive e continua
		// visível para errors.Is lá em cima. Com %v ela se perde.
		return fmt.Errorf("enviando lote para o backend: %w", errClienteNaoIniciado)
	}
	return nil
}

func validar(porta int) error {
	if porta < 1024 {
		return fmt.Errorf("validando config: %w", &ErrConfigInvalida{Campo: "port", Valor: porta})
	}
	return nil
}

func main() {
	err := enviar(false)
	fmt.Println("erro:", err)
	// errors.Is percorre a cadeia de embrulhos.
	fmt.Println("é errClienteNaoIniciado?", errors.Is(err, errClienteNaoIniciado))

	err = validar(80)
	var alvo *ErrConfigInvalida
	// errors.As faz o mesmo, mas devolve o valor concreto para você usar.
	if errors.As(err, &alvo) {
		fmt.Printf("campo problemático: %s (valor %v)\n", alvo.Campo, alvo.Valor)
	}

	// errors.Join acumula erros independentes. É o que se usa quando você
	// valida várias coisas e quer reportar todas de uma vez, em vez de parar
	// na primeira. O Collector também usa go.uber.org/multierr para isto.
	juntos := errors.Join(
		validar(80),
		enviar(false),
	)
	fmt.Println("--- acumulado ---")
	fmt.Println(juntos)
	fmt.Println("ainda encontra o sentinela?", errors.Is(juntos, errClienteNaoIniciado))
}
