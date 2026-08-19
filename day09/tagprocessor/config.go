package tagprocessor

import (
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
)

// Config do processor. Um mapa de atributos que serão gravados no Resource de
// tudo que passar pelo pipeline.
type Config struct {
	// Attributes são pares chave/valor adicionados ao Resource.
	Attributes map[string]string `mapstructure:"attributes"`

	// Overwrite decide o que fazer quando a chave já existe.
	Overwrite bool `mapstructure:"overwrite"`

	_ struct{}
}

var (
	_ component.Config  = (*Config)(nil)
	_ confmap.Validator = (*Config)(nil)
)

var errSemAtributos = errors.New("attributes não pode ser vazio")

func (c *Config) Validate() error {
	if len(c.Attributes) == 0 {
		return errSemAtributos
	}
	for k := range c.Attributes {
		if k == "" {
			return errors.New("chave de atributo não pode ser vazia")
		}
	}
	return nil
}
