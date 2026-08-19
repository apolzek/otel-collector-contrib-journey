package printexporter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exportertest"
)

func TestCreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig()

	// CheckConfigStruct valida as regras que o repositório exige de qualquer
	// struct de config: todo campo exportado precisa de tag mapstructure, os
	// nomes precisam ser minúsculos, e por aí vai.
	assert.NoError(t, componenttest.CheckConfigStruct(cfg))
	assert.Equal(t, &Config{Path: "stdout"}, cfg)
}

func TestCreateTraces(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	exp, err := factory.CreateTraces(
		context.Background(),
		exportertest.NewNopSettings(typeStr),
		cfg,
	)
	require.NoError(t, err)
	require.NotNil(t, exp)

	// Ciclo de vida completo: um componente precisa sobreviver a
	// Start seguido de Shutdown sem vazar goroutine nem entrar em pânico.
	require.NoError(t, exp.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, exp.Shutdown(context.Background()))
}

func TestCreateLogs(t *testing.T) {
	factory := NewFactory()
	exp, err := factory.CreateLogs(
		context.Background(),
		exportertest.NewNopSettings(typeStr),
		factory.CreateDefaultConfig(),
	)
	require.NoError(t, err)
	require.NoError(t, exp.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, exp.Shutdown(context.Background()))
}
