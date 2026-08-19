package tickreceiver

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// tickReceiver é um receiver "push por conta própria": ele não escuta porta
// nenhuma, apenas gera telemetria num intervalo fixo.
type tickReceiver struct {
	cfg    *Config
	logger *zap.Logger
	next   consumer.Logs // para onde o dado vai: o resto do pipeline

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func newTickReceiver(cfg *Config, logger *zap.Logger, next consumer.Logs) *tickReceiver {
	return &tickReceiver{cfg: cfg, logger: logger, next: next}
}

// Start NÃO pode bloquear. O padrão é: criar um contexto cancelável, subir a
// goroutine do loop e retornar imediatamente.
//
// Note que o ctx recebido aqui é o contexto do boot, não o do tempo de vida do
// componente: ele pode ser cancelado assim que o Start retorna. Por isso
// derivamos de context.Background().
func (r *tickReceiver) Start(_ context.Context, _ component.Host) error {
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.loop(ctx)
	}()
	return nil
}

// Shutdown cancela o contexto e espera a goroutine terminar. Sem esse wait o
// teste de goleak acusa vazamento de goroutine.
func (r *tickReceiver) Shutdown(context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}
	r.wg.Wait()
	return nil
}

func (r *tickReceiver) loop(ctx context.Context) {
	ticker := time.NewTicker(r.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Um erro aqui não derruba o Collector: o receiver decide o que
			// fazer. Logar e seguir é o comportamento usual de um gerador.
			if err := r.next.ConsumeLogs(ctx, r.buildLogs()); err != nil {
				r.logger.Error("falha ao entregar logs ao pipeline", zap.Error(err))
			}
		}
	}
}

func (r *tickReceiver) buildLogs() plog.Logs {
	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	rl.Resource().Attributes().PutStr("service.name", "tickreceiver")

	sl := rl.ScopeLogs().AppendEmpty()
	sl.Scope().SetName("github.com/apolzek/tickreceiver")

	lr := sl.LogRecords().AppendEmpty()
	lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	lr.SetTimestamp(lr.ObservedTimestamp())
	lr.SetSeverityNumber(plog.SeverityNumberInfo)
	lr.SetSeverityText("INFO")
	lr.Body().SetStr(r.cfg.Message)
	return ld
}
