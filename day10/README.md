# Day 10: como criar uma extension

Objetivo do dia: escrever uma extension e entender como componentes de pipeline
encontram e usam capacidades compartilhadas.

A extension deste dia se chama heartbeat. Ela escreve um arquivo periodicamente,
como um liveness probe faria, e expõe um contador para quem quiser consultar.
Código em heartbeatextension/.

```bash
cd heartbeatextension
go test ./...
```

## O que uma extension é

A única classe que não toca em telemetria. Ela não entra em pipeline nenhum,
vive em service.extensions, e serve para duas coisas:

Capacidades operacionais do processo: health check, pprof, zpages, sinal de
liveness.

Capacidades compartilhadas entre componentes: autenticação, armazenamento
persistente, descoberta de alvos, resolução de encoding.

## As diferenças de API

A factory é a mais simples de todas:

```go
func NewFactory() extension.Factory {
    return extension.NewFactory(
        component.MustNewType("heartbeat"),
        createDefaultConfig,
        createExtension,
        component.StabilityLevelDevelopment,
    )
}
```

Sem WithTraces, sem WithLogs, sem WithMetrics. Uma extension não tem sinal, então
tem uma única função de criação e um único nível de estabilidade.

E a interface é literalmente component.Component:

```go
type Extension interface {
    component.Component
}
```

Start e Shutdown, nada mais. Toda a utilidade de uma extension está no que ela
expõe ALÉM disso.

## O padrão que dá sentido à classe

Uma extension útil declara uma interface própria e a implementa:

```go
type Heartbeater interface {
    Batidas() int64
}
```

Do outro lado, um componente de pipeline procura essa capacidade no host:

```go
func (e *meuExporter) start(_ context.Context, host component.Host) error {
    ext, ok := host.GetExtensions()[e.cfg.HeartbeatID]
    if !ok {
        return fmt.Errorf("extension %q não encontrada", e.cfg.HeartbeatID)
    }
    hb, ok := ext.(Heartbeater)
    if !ok {
        return fmt.Errorf("extension %q não implementa Heartbeater", e.cfg.HeartbeatID)
    }
    e.hb = hb
    return nil
}
```

Três coisas para levar daqui:

A busca acontece no Start, não na factory. Na hora da construção as extensions
ainda não existem.

Falhe com mensagem clara quando não achar ou quando o tipo não bater. Esse erro
vai ser lido por alguém que só mexeu no YAML.

O componente depende da interface, não do tipo concreto da extension. É assim
que o mesmo exporter aceita qualquer extension de autenticação.

O teste em extension_test.go monta um host falso e percorre esse caminho
inteiro, que é como você testa isso sem subir um Collector.

## Ordem de inicialização

Extensions sobem antes de todos os componentes de pipeline e descem depois de
todos. É isso que garante que uma extension de autenticação já esteja pronta
quando o primeiro exporter tentar usá-la.

Se a sua extension precisa de trabalho de fundo, ela segue exatamente o mesmo
padrão de goroutine do day08: contexto derivado de Background, WaitGroup, cancel
no Shutdown.

## As interfaces prontas do core

Antes de inventar uma interface própria, veja se já existe uma:

extensionauth para autenticação. Implementando ela, seu componente passa a ser
usável por qualquer exporter ou receiver que fale HTTP ou gRPC com as configs
padrão, sem que eles saibam nada sobre você.

extensioncapabilities para coisas como validação e observação de config.

extensionmiddleware para interceptar requisições HTTP e gRPC.

No contrib existe ainda a interface de storage, usada pela fila persistente do
exporterhelper e por receivers que precisam guardar posição de leitura.

Implementar uma interface existente é sempre melhor que criar uma nova: seu
componente passa a funcionar com o que já existe.

## Vendo funcionar

No builder-config.yaml do day05:

```yaml
extensions:
  - gomod: github.com/apolzek/otel-collector-contrib-journey/day10/heartbeatextension v0.0.0

replaces:
  - github.com/apolzek/otel-collector-contrib-journey/day10/heartbeatextension => ../../day10/heartbeatextension
```

No config.yaml, note que extensions são ligadas em service.extensions e não em
pipelines:

```yaml
extensions:
  heartbeat:
    path: /tmp/otelcol-heartbeat
    interval: 5s

service:
  extensions: [heartbeat]
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [print]
```

## Exercícios

1. Suba um servidor HTTP na extension expondo o contador em /health, usando
   confighttp.ServerConfig.
2. Faça o printexporter do day07 procurar a extension no host e recusar subir
   quando ela não estiver configurada.
3. Leia o extensionauth do core e implemente uma autenticação de token fixo.

## Checklist do dia

* Sei que extension não tem sinal e por que a factory dela é diferente.
* Sei expor uma capacidade por interface e buscá-la pelo host no Start.
* Sei que a busca não pode acontecer na factory.
* Sei procurar uma interface pronta antes de criar a minha.

Próximo: [Day 11, como criar um connector](../day11/README.md)
