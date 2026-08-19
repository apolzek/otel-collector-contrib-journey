package tickreceiver

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
)

// Config do receiver. Duração aceita a sintaxe do time.ParseDuration no YAML
// (30s, 1m, 500ms), porque o confmap tem um hook de decode para time.Duration.
type Config struct {
	// Interval é de quanto em quanto tempo um log é gerado.
	Interval time.Duration `mapstructure:"interval"`

	// Message é o corpo do log gerado.
	Message string `mapstructure:"message"`

	_ struct{}
}

var (
	_ component.Config  = (*Config)(nil)
	_ confmap.Validator = (*Config)(nil)
)

var errIntervalMuitoCurto = errors.New("interval precisa ser >= 100ms")

func (c *Config) Validate() error {
	if c.Interval < 100*time.Millisecond {
		return errIntervalMuitoCurto
	}
	return nil
}
