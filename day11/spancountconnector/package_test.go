package spancountconnector

import (
	"testing"

	"go.uber.org/goleak"
)

// Falha o pacote se alguma goroutine sobreviver aos testes. No contrib este
// arquivo é gerado pelo mdatagen como generated_package_test.go.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
