package spancountconnector

import (
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
)

// Config do connector.
type Config struct {
	// MetricName é o nome da métrica produzida.
	MetricName string `mapstructure:"metric_name"`

	// Dimension é o atributo do Resource copiado para a métrica. Vazio
	// significa nenhuma dimensão.
	Dimension string `mapstructure:"dimension"`

	_ struct{}
}

var (
	_ component.Config  = (*Config)(nil)
	_ confmap.Validator = (*Config)(nil)
)

var errNomeVazio = errors.New("metric_name não pode ser vazio")

func (c *Config) Validate() error {
	if c.MetricName == "" {
		return errNomeVazio
	}
	return nil
}
