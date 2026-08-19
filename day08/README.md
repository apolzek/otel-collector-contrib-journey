# Day 08: como criar um receiver

Objetivo do dia: escrever um receiver e entender a diferença entre os dois
modos de operação da classe.

O receiver deste dia se chama tick. Ele gera um log record a cada intervalo
configurado. Código em tickreceiver/.

```bash
cd tickreceiver
go test ./...
```

## Os dois tipos de receiver

Esta é a primeira decisão de projeto, e ela muda tudo:

Receiver push fica escutando. Abre uma porta, um socket ou um watcher de
arquivo, e reage quando algo chega. Exemplos: otlp, jaeger, filelog.

Receiver pull faz scrape. Acorda de tempos em tempos, vai buscar dados numa
fonte e devolve. Exemplos: prometheus, hostmetrics, httpcheck.

O receiver do exemplo é um caso simples do primeiro tipo: ele não escuta nada,
mas também não faz polling em fonte externa, apenas produz no seu próprio
ritmo. Serve para isolar o que interessa hoje, que é o ciclo de vida.

Se o seu caso é pull de métricas, não escreva o loop na mão. Use o pacote
scraper do core: você entrega uma função scrape(ctx) que devolve pmetric.Metrics
e ele cuida do intervalo, do timeout, dos erros parciais e da telemetria.

## A diferença de assinatura

```go
func createLogs(
    _ context.Context,
    set receiver.Settings,
    cfg component.Config,
    next consumer.Logs,
) (receiver.Logs, error)
```

Repare no quarto parâmetro. O receiver recebe um consumer, que é o próximo elo
do pipeline. Ele é o único que produz sem consumir, então a cadeia começa nele.

## O ciclo de vida, que é o ponto do dia

Este é o padrão que todo receiver com loop segue:

```go
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

func (r *tickReceiver) Shutdown(context.Context) error {
    if r.cancel != nil {
        r.cancel()
    }
    r.wg.Wait()
    return nil
}
```

Quatro detalhes, e cada um deles é um bug se você esquecer:

Start não bloqueia. Se você rodar o loop direto ali, o Collector nunca termina
de subir.

O contexto do loop deriva de context.Background(), não do ctx recebido. O ctx
do Start é o contexto do boot, e pode ser cancelado assim que o Start retorna,
matando seu loop na hora.

O Shutdown espera a goroutine com wg.Wait. Sem isso o teste de goleak reprova o
componente, e ele reprova mesmo, isso está no CI de todo componente do contrib.

O cancel pode ser nil se o Start nunca rodou. Sempre teste.

## Erro na entrega

Quando ConsumeLogs devolve erro, o receiver decide o que fazer. Não existe
regra única:

* Gerador ou scraper: logue e siga. Perder um ciclo é melhor que morrer.
* Receiver de rede: devolva o erro para o cliente, com o código apropriado, e
  deixe ele decidir se tenta de novo. É o que o otlpreceiver faz.

O que nunca fazer é engolir o erro em silêncio. Se você não devolve e não loga,
o dado some sem rastro.

## Telemetria interna

Um receiver de produção reporta quantos itens aceitou e quantos recusou, com
receiverhelper.ObsReport ou com a telemetria gerada pelo mdatagen a partir do
metadata.yaml. São essas métricas que viram
otelcol_receiver_accepted_log_records e otelcol_receiver_refused_log_records,
que é como as pessoas monitoram o Collector.

## Concorrência em receiver de scrape

Se o seu receiver consulta vários alvos, o padrão do contrib é uma goroutine
por alvo com WaitGroup, mutex protegendo a escrita das métricas e atomic para
contadores. O httpcheckreceiver é um bom caso para ler.

Cuidado com o efeito manada: disparar cem goroutines de scrape no mesmo
instante gera pico de CPU e de rede a cada intervalo. Limite a concorrência.

## Vendo funcionar

Acrescente ao builder-config.yaml do day05:

```yaml
receivers:
  - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver v0.159.0
  - gomod: github.com/apolzek/otel-collector-contrib-journey/day08/tickreceiver v0.0.0

replaces:
  - github.com/apolzek/otel-collector-contrib-journey/day08/tickreceiver => ../../day08/tickreceiver
```

E no config.yaml:

```yaml
receivers:
  tick:
    interval: 2s
    message: "estou vivo"

service:
  pipelines:
    logs:
      receivers: [tick]
      exporters: [print]
```

## Exercícios

1. Faça o receiver parar de gerar depois de N mensagens e verifique com o teste
   de goleak que nenhuma goroutine ficou para trás.
2. Troque a geração por um scrape HTTP de verdade e trate o timeout com context.
3. Migre para o pacote scraper produzindo metrics em vez de logs.

## Checklist do dia

* Sei a diferença entre receiver push e receiver pull.
* Sei o padrão correto de Start com goroutine e Shutdown com wait.
* Sei por que o contexto do loop não pode ser o do Start.
* Sei o que fazer quando o próximo consumer devolve erro.

Próximo: [Day 09, como criar um processor](../day09/README.md)
