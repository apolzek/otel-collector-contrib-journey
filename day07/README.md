# Day 07: como criar um exporter

Objetivo do dia: escrever um exporter completo, com config, factory, testes e
rodando dentro de um Collector de verdade.

O exporter deste dia se chama print. Ele escreve uma linha por span e uma por
log record, no stdout ou num arquivo. Código em printexporter/.

```bash
cd printexporter
go test ./...
```

## O que um exporter é

A ponta final do pipeline. Ele consome telemetria e entrega para fora do
Collector. Em termos de interface, é um consumer que não tem próximo.

O que torna um exporter diferente das outras classes é o que acontece quando
ele falha. Backend fora do ar não pode derrubar o Collector nem perder dados
silenciosamente, e é por isso que quase todo exporter é construído em cima do
exporterhelper.

## Anatomia

Quatro arquivos, sempre nesta divisão:

config.go tem a struct Config, as tags mapstructure e o Validate.

factory.go tem NewFactory, createDefaultConfig e uma função de criação por
sinal suportado.

exporter.go tem o estado e as funções de push.

os arquivos _test.go, um por arquivo de código.

## Passo 1: a config

```go
type Config struct {
    Path   string `mapstructure:"path"`
    Prefix string `mapstructure:"prefix"`

    _ struct{}
}

func (c *Config) Validate() error {
    if c.Path == "" {
        return errEmptyPath
    }
    return nil
}
```

O que aprender aqui:

* Toda tag mapstructure em minúsculo e com underscore, seguindo o padrão do
  YAML do Collector.
* Validate roda antes de o componente existir. Prefira falhar no boot a falhar
  no primeiro dado.
* O campo `_ struct{}` no fim impede construção por literal posicional.

## Passo 2: a implementação

```go
func (e *printExporter) pushTraces(_ context.Context, td ptrace.Traces) error {
    e.mu.Lock()
    defer e.mu.Unlock()
    // percorre e escreve
}
```

A assinatura é exatamente consumer.ConsumeTracesFunc. Três pontos:

O Collector pode chamar essa função de várias goroutines ao mesmo tempo. Se o
exporter tem estado mutável, ele precisa de proteção. Aqui é um mutex em volta
da escrita.

Start abre o recurso e não bloqueia. Shutdown fecha e é idempotente, inclusive
quando Start nunca rodou.

O erro devolvido decide o comportamento do exporterhelper.

## Passo 3: erro permanente ou temporário

Esta é a particularidade mais importante da classe. Devolver o erro cru faz o
helper tentar de novo. Nem sempre é o que você quer:

```go
// Vale a pena tentar de novo: rede, timeout, 503.
return fmt.Errorf("enviando lote: %w", err)

// Não adianta tentar: payload inválido, 400, credencial errada.
return consumererror.NewPermanent(fmt.Errorf("payload rejeitado: %w", err))
```

Um erro permanente marcado como temporário faz o Collector girar em retry para
sempre com um dado que nunca vai passar. É um dos bugs mais comuns em exporter
novo.

## Passo 4: a factory

```go
func NewFactory() exporter.Factory {
    return exporter.NewFactory(
        component.MustNewType("print"),
        createDefaultConfig,
        exporter.WithTraces(createTraces, component.StabilityLevelDevelopment),
        exporter.WithLogs(createLogs, component.StabilityLevelDevelopment),
    )
}
```

Cada sinal suportado é uma opção e uma função de criação separada. Componente é
a combinação de classe e sinal: suportar traces e logs significa duas funções,
dois construtores e duas entradas de estabilidade no metadata.yaml.

Dentro da função de criação, o exporterhelper faz o trabalho pesado:

```go
return exporterhelper.NewTraces(ctx, set, cfg, e.pushTraces,
    exporterhelper.WithStart(e.start),
    exporterhelper.WithShutdown(e.shutdown),
    exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
)
```

## O que um exporter de produção adiciona

O exemplo é mínimo de propósito. Um exporter real embute as configurações
comuns do core na sua Config, com squash, e passa as opções correspondentes:

```go
type Config struct {
    QueueConfig  exporterhelper.QueueBatchConfig `mapstructure:"sending_queue"`
    TimeoutConfig exporterhelper.TimeoutConfig   `mapstructure:"timeout"`
    BackOffConfig configretry.BackOffConfig      `mapstructure:"retry_on_failure"`
}
```

Ganhos imediatos, sem escrever nada: fila em memória ou persistente em disco,
retry com backoff exponencial, timeout por tentativa e as métricas internas que
todo mundo já sabe monitorar.

Se o seu exporter fala HTTP ou gRPC, use também confighttp.ClientConfig ou
configgrpc.ClientConfig em vez de campos soltos de endpoint, TLS e headers. São
os mesmos nomes de configuração de todos os outros componentes, e vêm com
autenticação por extension de graça.

## Vendo funcionar

Este exporter já está ligado no builder-config.yaml do day05:

```bash
cd ../day05
builder --config=builder-config.yaml
./_build/meu-otelcol --config=config.yaml
```

Em outro terminal:

```bash
curl -X POST http://127.0.0.1:4318/v1/traces \
  -H 'Content-Type: application/json' -d @../day01/span-exemplo.json
```

Saída:

```
[meu-otelcol] span service=checkout name=POST /checkout trace_id=5b8ef... duration=1.5s
```

## Exercícios

1. Acrescente suporte a metrics. São uma opção na factory, uma função de
   criação e uma função de push.
2. Faça pushTraces devolver erro permanente quando o span não tem nome, e veja
   nos logs a diferença de comportamento.
3. Adicione TimeoutConfig e BackOffConfig na Config e passe as opções
   correspondentes ao helper.

## Checklist do dia

* Escrevi config, factory, implementação e testes de um exporter.
* Sei quando devolver erro permanente e quando devolver temporário.
* Sei o que ganho de graça ao usar o exporterhelper.
* Vi meu componente rodando dentro de um Collector compilado por mim.

Próximo: [Day 08, como criar um receiver](../day08/README.md)
