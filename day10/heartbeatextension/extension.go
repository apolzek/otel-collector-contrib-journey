package heartbeatextension

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.uber.org/zap"
)

// Heartbeater é a capacidade que esta extension oferece aos outros
// componentes. Extension é justamente isso: um objeto compartilhado que
// componentes de pipeline encontram pelo host e usam através de uma interface.
type Heartbeater interface {
	// Batidas devolve quantas batidas já foram escritas.
	Batidas() int64
}

type heartbeatExtension struct {
	cfg    *Config
	logger *zap.Logger

	batidas atomic.Int64
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// extension.Extension é apenas component.Component: Start e Shutdown, nada
// mais. Toda a diferença está no que a extension expõe além disso.
var (
	_ extension.Extension = (*heartbeatExtension)(nil)
	_ Heartbeater         = (*heartbeatExtension)(nil)
)

func newHeartbeatExtension(cfg *Config, logger *zap.Logger) *heartbeatExtension {
	return &heartbeatExtension{cfg: cfg, logger: logger}
}

func (e *heartbeatExtension) Start(_ context.Context, _ component.Host) error {
	// Uma batida imediata evita a janela em que o arquivo ainda não existe.
	if err := e.bater(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel

	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		ticker := time.NewTicker(e.cfg.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := e.bater(); err != nil {
					e.logger.Error("falha ao escrever heartbeat", zap.Error(err))
				}
			}
		}
	}()
	return nil
}

func (e *heartbeatExtension) Shutdown(context.Context) error {
	if e.cancel != nil {
		e.cancel()
	}
	e.wg.Wait()
	return os.Remove(e.cfg.Path)
}

func (e *heartbeatExtension) Batidas() int64 {
	return e.batidas.Load()
}

func (e *heartbeatExtension) bater() error {
	agora := time.Now().UTC().Format(time.RFC3339Nano)
	if err := os.WriteFile(e.cfg.Path, []byte(agora+"\n"), 0o600); err != nil {
		return err
	}
	// atomic.Int64 porque Batidas pode ser chamado de outra goroutine
	// enquanto o loop escreve.
	e.batidas.Add(1)
	return nil
}
