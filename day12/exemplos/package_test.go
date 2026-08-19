package exemplos

import (
	"testing"

	"go.uber.org/goleak"
)

// goleak falha o pacote inteiro se alguma goroutine sobreviver aos testes.
// É a rede de proteção contra o bug mais comum de componente: subir um loop no
// Start e esquecer de encerrá-lo no Shutdown.
//
// No contrib este arquivo é GERADO pelo mdatagen com o nome
// generated_package_test.go, a partir do metadata.yaml. Não se edita à mão.
func TestMain(m *testing.M) {
	// IgnoreTopFunction serve para goroutines de bibliotecas de terceiros que
	// legitimamente não encerram. Use com parcimônia: quase sempre o vazamento
	// é seu mesmo.
	goleak.VerifyTestMain(m)
}
