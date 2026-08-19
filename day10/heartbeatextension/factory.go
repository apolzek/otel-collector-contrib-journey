package heartbeatextension

import (
	"context"
	"path/filepath"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

var typeStr = component.MustNewType("heartbeat")

// Repare na diferença para os componentes de pipeline: não há WithTraces,
// WithLogs nem WithMetrics. Uma extension tem uma única função de criação e um
// único nível de estabilidade.
func NewFactory() extension.Factory {
	return extension.NewFactory(
		typeStr,
		createDefaultConfig,
		createExtension,
		component.StabilityLevelDevelopment,
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		Path:     filepath.Join(".", "otelcol-heartbeat"),
		Interval: 5 * time.Second,
	}
}

func createExtension(
	_ context.Context,
	set extension.Settings,
	cfg component.Config,
) (extension.Extension, error) {
	return newHeartbeatExtension(cfg.(*Config), set.Logger), nil
}
