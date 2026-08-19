package resilienteexporter

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Esta é a diferença entre o exporter mínimo do day07 e um de produção: as
// três configurações de resiliência vêm prontas do core, com os mesmos nomes
// que o usuário já conhece de todos os outros exporters.
type Config struct {
	// timeout: prazo de CADA tentativa.
	TimeoutConfig exporterhelper.TimeoutConfig `mapstructure:"timeout"`

	// retry_on_failure: quantas vezes e com qual backoff.
	BackOffConfig configretry.BackOffConfig `mapstructure:"retry_on_failure"`

	// sending_queue: fila, concorrência de envio e batching.
	QueueConfig configoptional.Optional[exporterhelper.QueueBatchConfig] `mapstructure:"sending_queue"`

	// Específico deste componente: quantas vezes o backend falso falha antes
	// de aceitar. Existe só para o exemplo ser testável.
	FalhasAteAceitar int `mapstructure:"falhas_ate_aceitar"`

	_ struct{}
}

var _ component.Config = (*Config)(nil)
