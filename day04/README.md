# Day 04: Go essencial, parte 2

Objetivo do dia: os padrões de Go que estruturam o Collector. Factory,
functional options, concorrência, context e generics. Continua sendo teoria
pura, sem a API do Collector.

```bash
cd exemplos
go run ./01-factory
```

## 1. Factory, o padrão central

Se você entender bem só um padrão deste repositório, que seja este.

O problema: o Collector lê um YAML com a palavra otlp e precisa construir um
componente. Ele não pode conhecer os trezentos componentes existentes em tempo
de compilação, então precisa de um registro que mapeia nome para construtor.

A Factory é esse contrato:

```go
type Factory interface {
    Type() component.Type
    CreateDefaultConfig() component.Config
    // mais as funções de criação por sinal
}
```

Três detalhes que valem ouro:

CreateDefaultConfig existe para separar o padrão do que o usuário escreveu. O
Collector pega a config padrão, despeja o YAML por cima e só então constrói. Um
campo ausente no YAML mantém o padrão, sem código condicional em lugar nenhum.

Você nunca implementa a interface Factory na mão. Sempre chama o construtor
pronto do core:

```go
func NewFactory() processor.Factory {
    return processor.NewFactory(
        component.MustNewType("meuprocessor"),
        createDefaultConfig,
        processor.WithTraces(createTraces, component.StabilityLevelDevelopment),
    )
}
```

NewFactory é a única coisa que um componente precisa exportar. O OCB gera um
components.go que é literalmente um mapa de tipo para NewFactory. Adicionar um
componente ao seu build é acrescentar uma entrada nesse mapa.

Exemplo: exemplos/01-factory

## 2. Functional options

O segundo padrão mais presente. Uma Option é uma função que muta o objeto em
construção:

```go
type Option func(*servidor)

func WithTimeout(d time.Duration) Option {
    return func(s *servidor) { s.timeout = d }
}
```

Resolve o problema de ter dez configurações opcionais sem criar dez parâmetros
nem dez construtores. Você vai ver isso em duas camadas do Collector:

Na factory, com processor.WithTraces, exporter.WithLogs, connector.
WithTracesToMetrics. Cada opção registra o suporte a um sinal e o nível de
estabilidade dele.

Nos helpers, com exporterhelper.WithStart, WithShutdown, WithRetry,
WithCapabilities, WithTimeout.

O core usa a variação em que Option é uma interface e não um tipo função.
Custa uma linha a mais e permite evoluir o pacote sem quebrar quem já usa.

Exemplo: exemplos/02-options

## 3. Concorrência

O Collector é concorrente por natureza: várias goroutines empurram dados no
mesmo componente ao mesmo tempo. As ferramentas:

goroutine e sync.WaitGroup para disparar trabalho paralelo e esperar. Chame
wg.Add antes do go, e wg.Done com defer dentro. Add depois do go é uma corrida
clássica.

sync.Mutex para proteger estruturas compostas. Convenção: o campo protegido vem
declarado logo abaixo do mutex que o protege.

sync/atomic para contadores simples. Cuidado: go.uber.org/atomic é proibido
pelo depguard, use o pacote da stdlib.

sync.Once para garantir que algo aconteça uma vez só. O sharedcomponent do
core usa isso para que um receiver compartilhado por dois pipelines não suba
duas vezes.

Canais para coordenar. Fechar um canal é o sinal de fim; sem isso um range
nunca termina e o WaitGroup trava.

Rode este exemplo com o detector de corrida ligado, que é como o CI roda:

```bash
go run -race ./03-goroutines
```

Exemplo: exemplos/03-goroutines

## 4. context.Context

Aparece como primeiro parâmetro em praticamente toda função do Collector:
Start(ctx, host), Shutdown(ctx), ConsumeTraces(ctx, td), scrape(ctx). O linter
revive exige essa posição.

Ele carrega três coisas: cancelamento, prazo e valores de escopo.

O padrão de loop de fundo é o mais importante para quem escreve componente. Não
existe kill de goroutine em Go, o único jeito de parar uma é pedir para ela
parar:

```go
func (r *receiver) Start(_ context.Context, _ component.Host) error {
    ctx, cancel := context.WithCancel(context.Background())
    r.cancel = cancel
    r.wg.Add(1)
    go func() {
        defer r.wg.Done()
        r.loop(ctx)
    }()
    return nil
}

func (r *receiver) Shutdown(context.Context) error {
    r.cancel()
    r.wg.Wait()
    return nil
}
```

Duas sutilezas aí:

O ctx que chega no Start é o contexto do boot, e pode ser cancelado assim que
o Start retorna. Por isso o loop deriva de context.Background(), não dele.

O wg.Wait no Shutdown não é opcional. Sem ele o teste de goleak acusa
vazamento de goroutine, e o CI do contrib roda esse teste em todo componente.

Exemplo: exemplos/04-context

## 5. Generics

Chegaram no Go 1.18 e no Collector aparecem principalmente no OTTL, em
pkg/ottl, que é a parte mais avançada do repositório em termos de Go.

Toda a API do OTTL é parametrizada pelo contexto de transformação:

```go
type Getter[K any] interface {
    Get(ctx context.Context, tCtx K) (any, error)
}
```

K é o contexto: span, log record, data point. Uma função escrita uma vez serve
para todos, sem duplicação e sem any correndo solto.

Os tipos com prefixo Standard, como StandardGetSetter[K], adaptam funções
comuns para essas interfaces. É o mesmo truque de StartFunc do day03, agora com
tipo parametrizado.

Nas restrições, a cláusula til aceita tipos definidos a partir do tipo base, e
não só o tipo base:

```go
type Numero interface {
    ~int | ~int64 | ~float64
}
```

Se você vai mexer em transformprocessor, filterprocessor ou routingconnector,
precisa estar confortável com isso.

Exemplo: exemplos/05-generics

## Sobre o prefixo x

Antes de sair lendo o código: o prefixo x significa experimental. Existem
xreceiver, xprocessor, xexporter, xconsumer, xconfmap, xprocessorhelper. É onde
vivem APIs ainda instáveis, hoje principalmente o sinal de profiles.

Repare que xprocessor.NewFactory devolve um processor.Factory comum, e quem
quer o que é experimental faz factory.(xprocessor.Factory). É o padrão de
descoberta de capacidade do day03 aplicado à evolução da API.

## Checklist do dia

* Sei desenhar o caminho do YAML até o componente construído.
* Consigo escrever uma função com functional options.
* Sei o padrão correto de subir e encerrar uma goroutine num componente.
* Sei ler uma assinatura com type parameters sem travar.

Próximo: [Day 05, OCB](../day05/README.md)
