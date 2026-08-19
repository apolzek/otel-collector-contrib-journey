# Day 06: as interfaces e os helpers do Collector

Objetivo do dia: conhecer o vocabulário que aparece em todo componente. A
partir de amanhã você escreve código; hoje você entende as peças que vai usar.

```bash
cd exemplos
go run ./01-pdata
go run ./02-confmap
go run ./03-pipeline
```

## 1. component.Component

Absolutamente todo componente implementa isto. Receiver, exporter, processor,
connector e extension, todos:

```go
type Component interface {
    // Start é chamado no boot. NÃO deve bloquear: se seu componente tem um
    // loop, suba uma goroutine e retorne. Erro aqui aborta o boot inteiro.
    Start(ctx context.Context, host Host) error

    // Shutdown é chamado no encerramento. Deve ser idempotente e respeitar
    // o ctx, sem bloquear indefinidamente.
    Shutdown(ctx context.Context) error
}
```

As interfaces por classe e sinal, como receiver.Traces, exporter.Logs ou
processor.Metrics, são marker interfaces: Component mais o método de consumo
daquele sinal, quando houver. Elas existem separadas para que o core possa
evoluir cada uma sem quebrar as outras.

Duas regras que o CI cobra:

* Shutdown precisa funcionar mesmo que Start nunca tenha rodado, porque o
  Collector encerra componentes já criados quando o boot falha no meio.
* Nenhuma goroutine pode sobreviver ao Shutdown. O teste de goleak reprova.

## 2. component.Host

É o que o componente pode pedir de volta ao Collector:

```go
type Host interface {
    GetExtensions() map[ID]Component
}
```

Só isso. O padrão de uso é pegar do mapa, testar se aquela extension implementa
a interface de que você precisa, e falhar com mensagem clara se não implementar.
O day10 mostra esse caminho completo.

## 3. Settings

Toda função de criação recebe uma struct Settings da sua classe, por exemplo
exporter.Settings. Dentro dela:

* ID, o component.ID desta instância, no formato tipo/nome.
* TelemetrySettings, com Logger (zap), TracerProvider, MeterProvider e Resource.
* BuildInfo, informações da distribuição.

Regra prática: use set.Logger para tudo. Nunca fmt.Println. O log é estruturado:

```go
logger.Error("falha ao enviar", zap.String("endpoint", e), zap.Error(err))
```

## 4. consumer e Capabilities

O que liga um componente ao próximo:

```go
type Traces interface {
    Capabilities() Capabilities
    ConsumeTraces(ctx context.Context, td ptrace.Traces) error
}
```

Existem consumer.Traces, consumer.Metrics e consumer.Logs. Um pipeline é uma
cadeia desses consumers, montada de trás para a frente pelo service.

Capabilities tem um campo só, e ele é importante:

```go
consumer.Capabilities{MutatesData: true}
```

Isso declara que o componente ESCREVE nos dados. Como pdata é passado por
referência, quando um receiver alimenta dois pipelines e um deles tem um
componente que muta, o Collector clona os dados antes de distribuir. Declarar
errado gera bug silencioso: um pipeline corrompe o outro.

Na dúvida, a regra é simples. Alterou qualquer coisa dentro do pdata recebido,
MutatesData é true.

Exemplo: exemplos/03-pipeline, que monta receiver, processor, exporter e um
fan-out na mão, sem o Collector.

## 5. pdata

O modelo de dados em memória e a fronteira que todos os componentes falam:

| pacote | sinal |
|---|---|
| pdata/ptrace | traces |
| pdata/pmetric | metrics |
| pdata/plog | logs |
| pdata/pprofile | profiles |
| pdata/pcommon | tipos compartilhados: Map, Value, Timestamp, Resource |

Três características que explicam tudo o que estranha nele:

É gerado a partir do protobuf do OTLP. O modelo interno é o mesmo do formato de
fio, o que evita converter a cada salto do pipeline.

A API é toda de acessores, sem campos públicos. É isso que permite trocar a
representação interna sem quebrar centenas de componentes.

É passado por referência. Escrever num atributo altera o que os outros veem.
Para isolar de verdade é preciso chamar CopyTo explicitamente.

A hierarquia se repete nos três sinais: Resource (de quem é o dado, como
service.name) leva a Scope (qual instrumentação gerou) que leva ao dado em si.

Exemplo: exemplos/01-pdata

## 6. confmap

A camada de configuração. Três etapas:

Providers dizem de onde o texto vem: file, env, yaml, http e https. O padrão é
file.

Resolução expande as referências antes do parse, como `${env:BACKEND}`, e faz
merge de vários arquivos passados em sequência. É assim que se separa base e
overlay por ambiente sem template engine.

Unmarshal e Validate despejam o mapa resolvido na struct Config, por cima dos
padrões da factory, seguindo as tags mapstructure. Se a struct implementar
confmap.Validator, o Validate é chamado antes de o componente ser criado.
Config inválida significa Collector que não sobe, com o campo apontado.

Exemplo: exemplos/02-confmap

## 7. Os helpers

Aqui mora a diferença entre escrever um componente em 80 linhas e em 800. Os
helpers do core resolvem o que é difícil e comum.

exporterhelper é o mais completo. Você entrega uma função de push e ele cuida
de fila (em memória ou persistente), retry com backoff exponencial, timeout por
tentativa e a telemetria interna do exporter. Opções: WithStart, WithShutdown,
WithTimeout, WithRetry, WithQueue, WithCapabilities.

processorhelper cuida do ciclo de vida, das capabilities e da telemetria. Você
entrega uma função que recebe os dados e devolve os dados. Devolver
processorhelper.ErrSkipProcessingData descarta o lote sem virar erro no
pipeline.

receiverhelper e scraper são para receiver. O scraper é o esqueleto de coleta
por polling: você escreve só a função scrape(ctx) que devolve métricas, e ele
cuida do intervalo, do timeout e dos erros parciais.

sharedcomponent resolve o caso do mesmo componente aparecer em dois pipelines:
com sync.Once ele garante um único Start e um único Shutdown.

## 8. Ciclo de vida e feature gates

O service monta um grafo, não uma lista. Ele resolve componentes compartilhados,
decide quem precisa clonar, liga os connectors e detecta ciclo.

O Start acontece na ordem inversa do fluxo, exporters primeiro e receivers por
último, para que nenhum receiver aceite dado sem ter para onde mandar. O
Shutdown é na ordem direta, drenando o pipeline.

Feature gates ligam e desligam mudanças de comportamento em tempo de deploy:

```bash
otelcol --config=config.yaml --feature-gates=+minha.feature,-outra.feature
```

O ciclo é copiado do Kubernetes: alpha vem desligado, beta vem ligado, stable é
permanente e deprecated é permanentemente desligado até sumir. Quando o
changelog diz que um comportamento mudou e menciona um gate, é aqui que você
reverte enquanto se adapta.

## Checklist do dia

* Sei o que Start e Shutdown podem e não podem fazer.
* Sei decidir se meu componente precisa de MutatesData true.
* Sei navegar em pdata e entendo por que ele é por referência.
* Sei o que cada helper faz por mim, para não reimplementar fila e retry.

Próximo: [Day 07, como criar um exporter](../day07/README.md)
