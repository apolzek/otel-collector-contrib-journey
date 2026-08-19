package exemplos

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

// Carregar a config de um YAML em testdata testa o que o usuário realmente
// escreve, incluindo os nomes das chaves. Testar só a struct em Go deixa passar
// erro de tag mapstructure.
func TestCarregarConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)

	tests := []struct {
		id       string
		esperado Config
		wantErr  error
	}{
		{
			id: "valido",
			esperado: Config{
				RemoveKeys:  []string{"http.request.header.authorization", "user.email"},
				Placeholder: "[REDACTED]",
			},
		},
		{
			id:       "sem_placeholder",
			esperado: Config{RemoveKeys: []string{"user.email"}},
		},
		{
			id:      "invalido",
			wantErr: ErrSemChaves,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			sub, err := cm.Sub(tt.id)
			require.NoError(t, err)

			var cfg Config
			require.NoError(t, sub.Unmarshal(&cfg))

			if tt.wantErr != nil {
				require.ErrorIs(t, cfg.Validate(), tt.wantErr)
				return
			}
			require.NoError(t, cfg.Validate())
			assert.Equal(t, tt.esperado, cfg)
		})
	}
}
