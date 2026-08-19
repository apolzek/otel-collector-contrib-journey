package heartbeatextension

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
)

// Config da extension. Uma extension não tem sinal: ela não aparece em nenhum
// pipeline, e por isso a factory dela recebe um único nível de estabilidade.
type Config struct {
	// Path é o arquivo tocado a cada batida, tipicamente lido por um
	// liveness probe.
	Path string `mapstructure:"path"`

	// Interval é o intervalo entre batidas.
	Interval time.Duration `mapstructure:"interval"`

	_ struct{}
}

var (
	_ component.Config  = (*Config)(nil)
	_ confmap.Validator = (*Config)(nil)
)

var (
	errPathVazio = errors.New("path não pode ser vazio")
	errIntervalo = errors.New("interval precisa ser > 0")
)

func (c *Config) Validate() error {
	if c.Path == "" {
		return errPathVazio
	}
	if c.Interval <= 0 {
		return errIntervalo
	}
	return nil
}
