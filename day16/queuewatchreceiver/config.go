package queuewatchreceiver

import (
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/scraper/scraperhelper"

	"github.com/apolzek/otel-collector-contrib-journey/day16/queuewatchreceiver/internal/metadata"
)

// Config combina três blocos, e nenhum deles é escrito à mão por inteiro:
//
//   - ControllerConfig traz collection_interval, initial_delay e timeout,
//     com os mesmos nomes de todo receiver de scrape do projeto.
//   - MetricsBuilderConfig é GERADO pelo mdatagen a partir do metadata.yaml e
//     traz os blocos metrics e resource_attributes, para o usuário ligar e
//     desligar cada métrica.
//   - O resto são os campos específicos deste componente.
//
// A tag squash faz os campos do struct interno aparecerem no nível de cima do
// YAML, em vez de criar um bloco aninhado.
type Config struct {
	scraperhelper.ControllerConfig `mapstructure:",squash"`
	metadata.MetricsBuilderConfig  `mapstructure:",squash"`

	// Queues são as filas observadas.
	Queues []string `mapstructure:"queues"`

	_ struct{}
}

var (
	_ component.Config  = (*Config)(nil)
	_ confmap.Validator = (*Config)(nil)
)

var errSemFilas = errors.New("queues não pode ser vazio")

func (c *Config) Validate() error {
	if len(c.Queues) == 0 {
		return errSemFilas
	}
	return nil
}
