package heartbeatextension

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/extension/extensiontest"
)

func TestConfigValidate(t *testing.T) {
	require.ErrorIs(t, (&Config{Interval: time.Second}).Validate(), errPathVazio)
	require.ErrorIs(t, (&Config{Path: "x"}).Validate(), errIntervalo)
	require.NoError(t, (&Config{Path: "x", Interval: time.Second}).Validate())
}

func TestHeartbeatEscreveArquivo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hb")

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Path = path
	cfg.Interval = 20 * time.Millisecond

	ext, err := factory.Create(context.Background(), extensiontest.NewNopSettings(typeStr), cfg)
	require.NoError(t, err)

	require.NoError(t, ext.Start(context.Background(), componenttest.NewNopHost()))

	// O arquivo já existe logo depois do Start, sem esperar o primeiro tick.
	_, err = os.Stat(path)
	require.NoError(t, err)

	hb := ext.(Heartbeater)
	require.Eventually(t, func() bool { return hb.Batidas() >= 3 }, 2*time.Second, 10*time.Millisecond)

	require.NoError(t, ext.Shutdown(context.Background()))
	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err), "o arquivo devia ser removido no shutdown")
}

// hostComExtensions imita o que o Collector entrega a um componente: um mapa
// de extensions já iniciadas. É assim que um exporter acha a extension de auth
// que ele precisa.
type hostComExtensions struct {
	exts map[component.ID]component.Component
}

func (h hostComExtensions) GetExtensions() map[component.ID]component.Component { return h.exts }

func TestLookupPeloHost(t *testing.T) {
	id := component.NewID(typeStr)
	ext := newHeartbeatExtension(&Config{Path: filepath.Join(t.TempDir(), "hb"), Interval: time.Second}, componenttest.NewNopTelemetrySettings().Logger)

	host := hostComExtensions{exts: map[component.ID]component.Component{id: ext}}

	// Este é o padrão real: pegar do host, testar se implementa a interface
	// que interessa e só então usar.
	achado, ok := host.GetExtensions()[id]
	require.True(t, ok)

	hb, ok := achado.(Heartbeater)
	require.True(t, ok, "a extension precisa implementar Heartbeater")
	assert.Equal(t, int64(0), hb.Batidas())
}
