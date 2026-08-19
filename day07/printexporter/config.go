package printexporter

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
)

// Config é exatamente o que o usuário escreve no YAML do Collector, embaixo de
// exporters::print. Cada campo precisa de uma tag mapstructure com o nome que
// aparece no YAML: é por ela que o confmap faz o unmarshal.
type Config struct {
	// Path é o arquivo onde as linhas são escritas. O valor "stdout" escreve
	// na saída padrão.
	Path string `mapstructure:"path"`

	// Prefix é escrito no começo de cada linha, útil para separar dois
	// exporters print na mesma saída.
	Prefix string `mapstructure:"prefix"`

	// Campo vazio no fim impede construir o Config com literal posicional.
	// É um idioma que aparece em vários componentes do contrib.
	_ struct{}
}

// Asserção em tempo de compilação: se Config deixar de satisfazer as
// interfaces, o pacote não compila. Não custa nada em runtime.
var (
	_ component.Config  = (*Config)(nil)
	_ confmap.Validator = (*Config)(nil)
)

var errEmptyPath = errors.New("path não pode ser vazio")

// Validate roda depois do unmarshal e antes de o componente ser criado.
// Se devolver erro, o Collector não sobe.
func (c *Config) Validate() error {
	if c.Path == "" {
		return errEmptyPath
	}
	if len(c.Prefix) > 32 {
		return fmt.Errorf("prefix tem %d caracteres, o máximo é 32", len(c.Prefix))
	}
	return nil
}
