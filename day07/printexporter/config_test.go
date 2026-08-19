package printexporter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Teste table-driven: o formato padrão do repositório. Um slice de casos,
// t.Run para cada um, require quando o teste não pode continuar e assert
// quando pode.
func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name: "config válida",
			cfg:  Config{Path: "stdout"},
		},
		{
			name:    "path vazio",
			cfg:     Config{Path: ""},
			wantErr: "path não pode ser vazio",
		},
		{
			name:    "prefix longo demais",
			cfg:     Config{Path: "stdout", Prefix: strings.Repeat("x", 33)},
			wantErr: "o máximo é 32",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
